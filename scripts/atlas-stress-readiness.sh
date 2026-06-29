#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
usage: scripts/atlas-stress-readiness.sh --out <public-safe-relative-dir> [--ao-atlas-root ../ao-atlas]

Validates AO Atlas large-workgraph stress material from Foundry. The script is
fixture-only: it validates, imports, and summarizes; it does not schedule,
execute, approve, publish, upload, or mutate repositories.
USAGE
}

OUT=""
AO_ATLAS_ROOT="../ao-atlas"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --out)
      OUT="${2:-}"
      shift 2
      ;;
    --ao-atlas-root)
      AO_ATLAS_ROOT="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$OUT" ]]; then
  echo "--out is required" >&2
  usage >&2
  exit 2
fi

case "$OUT" in
  /*|~*|tmp|tmp/*|*"/.."*|*".."/*)
    echo "--out must be a public-safe relative path inside this repo and outside tmp/" >&2
    exit 2
    ;;
esac

if [[ "$AO_ATLAS_ROOT" != "../ao-atlas" ]]; then
  echo "--ao-atlas-root currently expects the public-safe sibling path ../ao-atlas" >&2
  exit 2
fi
if [[ ! -f "$AO_ATLAS_ROOT/go.mod" ]]; then
  echo "AO Atlas checkout not found at $AO_ATLAS_ROOT" >&2
  exit 2
fi

mkdir -p "$OUT"

WORKGRAPH="examples/valid/workgraph-large-stress.json"
INSTANCE="examples/valid/stack-instance.json"
FOUNDRY_FROM_ATLAS="../ao-foundry"

atlas_run() {
  (cd "$AO_ATLAS_ROOT" && go run ./cmd/atlas "$@")
}

jq empty "$AO_ATLAS_ROOT/$WORKGRAPH" "$AO_ATLAS_ROOT/$INSTANCE"

atlas_run workgraph validate --workgraph "$WORKGRAPH" > "$FOUNDRY_FROM_ATLAS/$OUT/atlas-workgraph-validate.txt"
atlas_run workgraph status --workgraph "$WORKGRAPH" --json > "$FOUNDRY_FROM_ATLAS/$OUT/atlas-workgraph-status.json"
atlas_run foundry import \
  --workgraph "$WORKGRAPH" \
  --instance "$INSTANCE" \
  --out "$FOUNDRY_FROM_ATLAS/$OUT/foundry-import" > "$FOUNDRY_FROM_ATLAS/$OUT/atlas-foundry-import.stdout"

go run ./cmd/foundry atlas import validate \
  --import "$OUT/foundry-import/foundry-import.json" > "$OUT/foundry-atlas-import-validate.txt"

ready_tasks="$(jq -r '.ready' "$OUT/atlas-workgraph-status.json")"
blocked_tasks="$(jq -r '.blocked' "$OUT/atlas-workgraph-status.json")"
completed_tasks="$(jq -r '.completed' "$OUT/atlas-workgraph-status.json")"
imported_tasks="$(jq '.tasks | length' "$OUT/foundry-import/foundry-import.json")"

jq -n \
  --arg schema_version "ao.foundry.atlas-stress-readiness.v0.1" \
  --arg status "ready" \
  --arg workgraph "$AO_ATLAS_ROOT/$WORKGRAPH" \
  --arg foundry_import "$OUT/foundry-import/foundry-import.json" \
  --argjson ready_tasks "$ready_tasks" \
  --argjson blocked_tasks "$blocked_tasks" \
  --argjson completed_tasks "$completed_tasks" \
  --argjson imported_tasks "$imported_tasks" \
  '{
    schema_version:$schema_version,
    status:$status,
    workgraph:$workgraph,
    foundry_import:$foundry_import,
    ready_tasks:$ready_tasks,
    blocked_tasks:$blocked_tasks,
    completed_tasks:$completed_tasks,
    imported_tasks:$imported_tasks,
    schedules_work:false,
    executes_work:false,
    approves_work:false,
    mutates_repositories:false
  }' > "$OUT/atlas-stress-readiness.json"

jq empty "$OUT/atlas-stress-readiness.json" "$OUT/foundry-import/foundry-import.json"

echo "atlas_stress_readiness=ready"
echo "summary=$OUT/atlas-stress-readiness.json"
echo "ready_tasks=$ready_tasks"
echo "blocked_tasks=$blocked_tasks"
echo "imported_tasks=$imported_tasks"
