package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func validateExternalArtifactRoot(path string) (string, error) {
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("artifact root must be a directory")
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	if filepath.Dir(resolved) == resolved {
		return "", errors.New("artifact root must not be a filesystem root")
	}
	root, err := repoRoot()
	if err != nil {
		return "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	rootInfo, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	for ancestor := resolved; ; ancestor = filepath.Dir(ancestor) {
		ancestorInfo, err := os.Stat(ancestor)
		if err != nil {
			return "", err
		}
		if os.SameFile(rootInfo, ancestorInfo) {
			return "", errors.New("artifact root must be outside repository")
		}
		if filepath.Dir(ancestor) == ancestor {
			break
		}
	}
	return resolved, nil
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func writeSignedSmokeScript(path, artifactRoot string) error {
	script := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail

: "${AO2_CP_API_TOKEN:?set AO2_CP_API_TOKEN}"
if [ "${#AO2_CP_API_TOKEN}" -lt 32 ]; then
  printf 'AO2_CP_API_TOKEN must be at least 32 characters\n' >&2
  exit 2
fi

ARTIFACT_ROOT=%s
mkdir -p tmp/live-tools tmp/control-plane "$ARTIFACT_ROOT"

AO2_CP_API_TOKEN="$AO2_CP_API_TOKEN" ../ao2-control-plane/target/debug/ao2-cp-server --bind 127.0.0.1:18746 \
  --data-dir tmp/control-plane &
AO2_CP_PID="$!"
trap 'kill "$AO2_CP_PID" 2>/dev/null || true' EXIT

sleep 1

(cd ../ao-forge && go build -o ../ao-foundry/tmp/live-tools/forge ./cmd/forge)
(cd ../ao-covenant && go build -o ../ao-foundry/tmp/live-tools/covenant ./cmd/covenant)

go run ./cmd/foundry pulse run --out tmp/pulse

tmp/live-tools/forge plan \
  --brief tmp/pulse/forge-brief.json \
  --out "$ARTIFACT_ROOT/factory-plan.json"

tmp/live-tools/forge gate \
  --plan "$ARTIFACT_ROOT/factory-plan.json" \
  --covenant tmp/live-tools/covenant \
  --out "$ARTIFACT_ROOT/gate-result.json"

AO2_CP_API_TOKEN="$AO2_CP_API_TOKEN" tmp/live-tools/forge run \
  --plan "$ARTIFACT_ROOT/factory-plan.json" \
  --gate-result "$ARTIFACT_ROOT/gate-result.json" \
  --out "$ARTIFACT_ROOT/factory-packet.json" \
  --control-plane http://127.0.0.1:18746 \
  --live --non-interactive --no-dashboard

go run ./cmd/foundry pulse run \
  --out tmp/pulse-live \
  --forge-live-packet "$ARTIFACT_ROOT/factory-packet.json"

go run ./cmd/foundry trace inspect --trace tmp/pulse-live/pulse.trace.jsonl

cat > tmp/pulse-live/signed-smoke-result.json <<'JSON'
{
  "schema_version": "ao.foundry.signed-smoke-result.v0.1",
  "status": "ready",
  "pulse_event": "tmp/pulse-live/pulse-event.json",
  "forge_live_packet": "signed-smoke/factory-packet.json",
  "control_plane_readback": "ready"
}
JSON

go run ./cmd/foundry pulse run \
  --out tmp/pulse-live \
  --forge-live-packet "$ARTIFACT_ROOT/factory-packet.json" \
  --signed-smoke-result tmp/pulse-live/signed-smoke-result.json

go run ./cmd/foundry pulse summarize-signed-smoke --pulse tmp/pulse-live/pulse-event.json --out tmp/pulse-live/signed-smoke-summary.json

go run ./cmd/foundry release promotion validate --candidate examples/readiness/active-spine-release-candidate.ledger.json --signed-smoke-summary tmp/pulse-live/signed-smoke-summary.json --out tmp/release-promotion.live.json

printf 'signed_smoke_result=tmp/pulse-live/signed-smoke-result.json\n'
printf 'signed_smoke_summary=tmp/pulse-live/signed-smoke-summary.json\n'
printf 'release_promotion=tmp/release-promotion.live.json\n'
`, shellSingleQuote(filepath.ToSlash(artifactRoot)))
	if err := os.MkdirAll(parentDir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		return err
	}
	return nil
}
