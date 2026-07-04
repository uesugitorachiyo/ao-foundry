#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="${1:-tmp/ao-mission-atlas-foundry-e2e-smoke}"
OUT_ABS="$ROOT/$OUT"
mkdir -p "$OUT_ABS"

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$ROOT/$1" | awk '{print "sha256:" $1}'
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$ROOT/$1" | awk '{print "sha256:" $1}'
    return
  fi
  if command -v openssl >/dev/null 2>&1; then
    openssl dgst -sha256 "$ROOT/$1" | awk '{print "sha256:" $NF}'
    return
  fi
  echo "no SHA-256 tool available" >&2
  exit 1
}

ROUTE="examples/ao-mission-smoke/mission-route-readback.json"
SNAPSHOT="examples/ao-mission-smoke/governance-snapshot-readback.json"
MISSION_ROLLUP="examples/ao-mission-smoke/mission-final-rollup.json"
FOUNDRY_ROLLUP="examples/ao-mission-smoke/foundry-final-rollup.json"
ATLAS_METADATA="examples/ao-mission-smoke/atlas-workgraph-metadata.json"
SCHEDULER_RECOVERY="examples/ao-mission-smoke/scheduler-recovery-readback.json"
LEDGER_COMPACTION="examples/ao-mission-smoke/ledger-compaction-readback.json"
GATEWAY_READINESS_ROLLUP="examples/ao-mission-smoke/gateway-readiness-rollup.json"
MANIFEST="$OUT_ABS/artifact-manifest.json"
BAD_MANIFEST="$OUT_ABS/artifact-manifest.digest-mismatch.json"
SMOKE="$OUT_ABS/ao-mission-e2e-smoke.json"

cat > "$MANIFEST" <<JSON
{
  "schema": "ao.mission.artifact-manifest.v0.1",
  "mission_id": "mission-demo",
  "status": "ready",
  "operator_mode": "read_only",
  "artifact_refs": [
    {
      "schema": "ao.mission.artifact-ref.v0.1",
      "ref": "$ROUTE",
      "digest": "$(sha256_file "$ROUTE")",
      "kind": "route_readback"
    },
    {
      "schema": "ao.mission.artifact-ref.v0.1",
      "ref": "$SNAPSHOT",
      "digest": "$(sha256_file "$SNAPSHOT")",
      "kind": "governance_snapshot"
    },
	    {
	      "schema": "ao.mission.artifact-ref.v0.1",
	      "ref": "$ATLAS_METADATA",
	      "digest": "$(sha256_file "$ATLAS_METADATA")",
	      "kind": "atlas_metadata"
	    },
	    {
	      "schema": "ao.mission.artifact-ref.v0.1",
	      "ref": "$SCHEDULER_RECOVERY",
	      "digest": "$(sha256_file "$SCHEDULER_RECOVERY")",
	      "kind": "scheduler_recovery"
	    },
	    {
	      "schema": "ao.mission.artifact-ref.v0.1",
	      "ref": "$LEDGER_COMPACTION",
	      "digest": "$(sha256_file "$LEDGER_COMPACTION")",
	      "kind": "ledger_compaction"
	    },
	    {
	      "schema": "ao.mission.artifact-ref.v0.1",
	      "ref": "$GATEWAY_READINESS_ROLLUP",
	      "digest": "$(sha256_file "$GATEWAY_READINESS_ROLLUP")",
	      "kind": "gateway_readiness_rollup"
	    }
	  ],
  "safe_to_execute": false,
  "executes_work": false,
  "approves_work": false,
  "mutates_repositories": false,
  "exact_next_action": "AO Foundry validates Mission and Atlas readbacks before implementation handoff"
}
JSON

go run ./cmd/foundry ao-mission e2e-smoke \
  --route "$ROUTE" \
  --snapshot "$SNAPSHOT" \
  --mission-final-rollup "$MISSION_ROLLUP" \
  --foundry-final-rollup "$FOUNDRY_ROLLUP" \
  --atlas-metadata "$ATLAS_METADATA" \
  --artifact-manifest "$MANIFEST" \
  --scheduler-recovery "$SCHEDULER_RECOVERY" \
  --ledger-compaction "$LEDGER_COMPACTION" \
  --gateway-readiness-rollup "$GATEWAY_READINESS_ROLLUP" \
  --out "$SMOKE"

sed 's/sha256:[0-9a-f][0-9a-f]*/sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb/' "$MANIFEST" > "$BAD_MANIFEST"

if go run ./cmd/foundry ao-mission e2e-smoke \
  --route "$ROUTE" \
  --snapshot "$SNAPSHOT" \
  --mission-final-rollup "$MISSION_ROLLUP" \
  --foundry-final-rollup "$FOUNDRY_ROLLUP" \
  --atlas-metadata "$ATLAS_METADATA" \
  --artifact-manifest "$BAD_MANIFEST" \
  --scheduler-recovery "$SCHEDULER_RECOVERY" \
  --ledger-compaction "$LEDGER_COMPACTION" \
  --out "$OUT_ABS/should-not-exist.json" >/tmp/ao-mission-e2e-negative.out 2>/tmp/ao-mission-e2e-negative.err; then
  cat /tmp/ao-mission-e2e-negative.out
  cat /tmp/ao-mission-e2e-negative.err >&2
  echo "expected digest-mismatch manifest to fail" >&2
  exit 1
fi

if ! grep -q "digest mismatch" /tmp/ao-mission-e2e-negative.err; then
  cat /tmp/ao-mission-e2e-negative.err >&2
  echo "negative smoke did not report digest mismatch" >&2
  exit 1
fi

echo "ao_mission_atlas_foundry_e2e_smoke=$SMOKE"
echo "digest_negative_smoke=passed"
