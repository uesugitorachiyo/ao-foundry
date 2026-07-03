#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="${1:-tmp/ao-mission-atlas-foundry-e2e-smoke}"
OUT_ABS="$ROOT/$OUT"
mkdir -p "$OUT_ABS"

sha256_file() {
  shasum -a 256 "$ROOT/$1" | awk '{print "sha256:" $1}'
}

ROUTE="examples/ao-mission-smoke/mission-route-readback.json"
SNAPSHOT="examples/ao-mission-smoke/governance-snapshot-readback.json"
MISSION_ROLLUP="examples/ao-mission-smoke/mission-final-rollup.json"
FOUNDRY_ROLLUP="examples/ao-mission-smoke/foundry-final-rollup.json"
ATLAS_METADATA="examples/ao-mission-smoke/atlas-workgraph-metadata.json"
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
  --out "$SMOKE"

sed 's/sha256:[0-9a-f][0-9a-f]*/sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb/' "$MANIFEST" > "$BAD_MANIFEST"

if go run ./cmd/foundry ao-mission e2e-smoke \
  --route "$ROUTE" \
  --snapshot "$SNAPSHOT" \
  --mission-final-rollup "$MISSION_ROLLUP" \
  --foundry-final-rollup "$FOUNDRY_ROLLUP" \
  --atlas-metadata "$ATLAS_METADATA" \
  --artifact-manifest "$BAD_MANIFEST" \
  --out "$OUT_ABS/should-not-exist.json" >/tmp/ao-mission-e2e-negative.out 2>/tmp/ao-mission-e2e-negative.err; then
  cat /tmp/ao-mission-e2e-negative.out
  cat /tmp/ao-mission-e2e-negative.err >&2
  echo "expected digest-mismatch manifest to fail" >&2
  exit 1
fi

if ! rg -q "digest mismatch" /tmp/ao-mission-e2e-negative.err; then
  cat /tmp/ao-mission-e2e-negative.err >&2
  echo "negative smoke did not report digest mismatch" >&2
  exit 1
fi

echo "ao_mission_atlas_foundry_e2e_smoke=$SMOKE"
echo "digest_negative_smoke=passed"
