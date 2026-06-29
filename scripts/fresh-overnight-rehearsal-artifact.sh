#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
usage: scripts/fresh-overnight-rehearsal-artifact.sh --out <public-safe-relative-dir> [--ao-atlas-root ../ao-atlas] [--ao-command-root ../ao-command]

Runs a fresh fixture-only overnight rehearsal and preserves the AO Command
readback artifact in a digest-bound summary. It does not schedule, execute,
approve, publish, upload, call live services, or mutate repositories.
USAGE
}

OUT=""
AO_ATLAS_ROOT="../ao-atlas"
AO_COMMAND_ROOT="../ao-command"

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
    --ao-command-root)
      AO_COMMAND_ROOT="${2:-}"
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

sha256_file() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    echo "shasum or sha256sum is required" >&2
    exit 2
  fi
}

utc_stamp() {
  date -u +"%Y%m%dT%H%M%SZ"
}

mkdir -p "$OUT"

STAMP="$(utc_stamp)"
FRESH_OUTPUT_ROOT="$OUT/$STAMP-overnight-rehearsal"
if [[ -e "$FRESH_OUTPUT_ROOT" ]]; then
  echo "fresh output path already exists: $FRESH_OUTPUT_ROOT" >&2
  exit 2
fi
mkdir -p "$FRESH_OUTPUT_ROOT"

scripts/overnight-rehearsal-runner.sh \
  --out "$FRESH_OUTPUT_ROOT/runner" \
  --ao-atlas-root "$AO_ATLAS_ROOT" \
  --ao-command-root "$AO_COMMAND_ROOT" > "$FRESH_OUTPUT_ROOT/runner.stdout"

RUNNER_SUMMARY="$FRESH_OUTPUT_ROOT/runner/overnight-rehearsal-runner.json"
COMMAND_READBACK="$(jq -r '.command_readback' "$RUNNER_SUMMARY")"
COMPLEX_SUMMARY="$(jq -r '.complex_refactor_summary' "$RUNNER_SUMMARY")"

jq empty "$RUNNER_SUMMARY" "$COMMAND_READBACK" "$COMPLEX_SUMMARY"

runner_status="$(jq -r '.status' "$RUNNER_SUMMARY")"
command_status="$(jq -r '.command_status' "$RUNNER_SUMMARY")"
allowed_next_action="$(jq -r '.allowed_next_action' "$RUNNER_SUMMARY")"

if [[ "$runner_status" != "ready" || "$command_status" != "ready" ]]; then
  artifact_status="blocked"
else
  artifact_status="ready"
fi

jq -n \
  --arg schema_version "ao.foundry.overnight-rehearsal-artifact.v0.1" \
  --arg status "$artifact_status" \
  --arg fresh_output_root "$FRESH_OUTPUT_ROOT" \
  --arg runner_summary "$RUNNER_SUMMARY" \
  --arg command_readback "$COMMAND_READBACK" \
  --arg complex_refactor_summary "$COMPLEX_SUMMARY" \
  --arg allowed_next_action "$allowed_next_action" \
  --arg runner_sha "$(sha256_file "$RUNNER_SUMMARY")" \
  --arg command_sha "$(sha256_file "$COMMAND_READBACK")" \
  --arg complex_sha "$(sha256_file "$COMPLEX_SUMMARY")" \
  '{
    schema_version:$schema_version,
    status:$status,
    mode:"fixture_only_fresh_rehearsal",
    fresh_output_root:$fresh_output_root,
    runner_summary:$runner_summary,
    command_readback:$command_readback,
    complex_refactor_summary:$complex_refactor_summary,
    allowed_next_action:$allowed_next_action,
    source_digests:[
      {name:"runner_summary", path:$runner_summary, schema_version:"ao.foundry.overnight-rehearsal-runner.v0.1", sha256:$runner_sha},
      {name:"command_readback", path:$command_readback, schema_version:"ao.command.complex-refactor-status.v0.1", sha256:$command_sha},
      {name:"complex_refactor_summary", path:$complex_refactor_summary, schema_version:"ao.foundry.complex-refactor-workgraph-rehearsal.v0.1", sha256:$complex_sha}
    ],
    mutates_repositories:false,
    executes_work:false,
    schedules_work:false,
    approves_work:false,
    uploads_artifacts:false,
    calls_providers:false
  }' > "$FRESH_OUTPUT_ROOT/overnight-rehearsal-artifact.json"

if [[ "$artifact_status" != "ready" ]]; then
  echo "overnight_rehearsal_artifact=$artifact_status" >&2
  echo "artifact=$FRESH_OUTPUT_ROOT/overnight-rehearsal-artifact.json" >&2
  exit 1
fi

echo "overnight_rehearsal_artifact=ready"
echo "artifact=$FRESH_OUTPUT_ROOT/overnight-rehearsal-artifact.json"
echo "command_readback=$COMMAND_READBACK"
