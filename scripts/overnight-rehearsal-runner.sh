#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
usage: scripts/overnight-rehearsal-runner.sh --out <public-safe-relative-dir> [--ao-atlas-root ../ao-atlas] [--ao-command-root ../ao-command]

Runs the fixture-only overnight rehearsal control chain. It validates the Pulse
gate, lifecycle state, Atlas import/readback, blocked-node repair, needs-context
repack, and AO Command readback. It does not schedule, execute, approve,
publish, upload, call live services, or mutate repositories.
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

mkdir -p "$OUT"

REHEARSAL_OUT="$OUT/complex-refactor"
scripts/complex-refactor-workgraph-rehearsal.sh \
  --out "$REHEARSAL_OUT" \
  --ao-atlas-root "$AO_ATLAS_ROOT" \
  --ao-command-root "$AO_COMMAND_ROOT" > "$OUT/complex-refactor.stdout"

SUMMARY="$REHEARSAL_OUT/summary.json"
COMMAND_READBACK="$REHEARSAL_OUT/ao-command-complex-refactor-status.json"
PULSE_SUMMARY="$REHEARSAL_OUT/pulse-gate/summary.json"
ATLAS_READBACK="$REHEARSAL_OUT/foundry-atlas-readback.json"
CLOSURE_PACKET="$REHEARSAL_OUT/pulse-refactor-closure-packet.json"

jq empty "$SUMMARY" "$COMMAND_READBACK" "$PULSE_SUMMARY" "$ATLAS_READBACK" "$CLOSURE_PACKET"

pulse_gate_status="$(jq -r '.status' "$PULSE_SUMMARY")"
lifecycle_status="$(jq -r '.ready_path.allowed_next_action // "unknown"' "$PULSE_SUMMARY")"
atlas_import_status="$(jq -r '.status' "$ATLAS_READBACK")"
repair_plan_status="$(jq -r '.repair_plan.status' "$SUMMARY")"
context_repack_status="$(jq -r '.context_repack.status' "$SUMMARY")"
command_status="$(jq -r '.status' "$COMMAND_READBACK")"
closure_status="$(jq -r '.status' "$CLOSURE_PACKET")"
allowed_next_action="$(jq -r '.next_action' "$COMMAND_READBACK")"

if [[ "$pulse_gate_status" != "ready" || "$lifecycle_status" != "start_next_slice" || "$atlas_import_status" != "ready" || "$repair_plan_status" != "repair_required" || "$context_repack_status" != "ready" || "$command_status" != "ready" || "$closure_status" != "ready" ]]; then
  status="blocked"
else
  status="ready"
fi

jq -n \
  --arg schema_version "ao.foundry.overnight-rehearsal-runner.v0.1" \
  --arg status "$status" \
  --arg pulse_gate_status "$pulse_gate_status" \
  --arg lifecycle_status "$lifecycle_status" \
  --arg atlas_import_status "$atlas_import_status" \
  --arg repair_plan_status "$repair_plan_status" \
  --arg context_repack_status "$context_repack_status" \
  --arg command_status "$command_status" \
  --arg closure_status "$closure_status" \
  --arg allowed_next_action "$allowed_next_action" \
  --arg summary "$SUMMARY" \
  --arg command_readback "$COMMAND_READBACK" \
  --arg closure_packet "$CLOSURE_PACKET" \
  '{
    schema_version:$schema_version,
    status:$status,
    mode:"fixture_only_dry_run",
    pulse_gate_status:$pulse_gate_status,
    lifecycle_status:$lifecycle_status,
    atlas_import_status:$atlas_import_status,
    repair_plan_status:$repair_plan_status,
    context_repack_status:$context_repack_status,
    command_status:$command_status,
    closure_status:$closure_status,
    allowed_next_action:$allowed_next_action,
    complex_refactor_summary:$summary,
    command_readback:$command_readback,
    closure_packet:$closure_packet,
    mutates_repositories:false,
    executes_work:false,
    schedules_work:false,
    approves_work:false,
    uploads_artifacts:false
  }' > "$OUT/overnight-rehearsal-runner.json"

if [[ "$status" != "ready" ]]; then
  echo "overnight_rehearsal_runner=$status" >&2
  echo "summary=$OUT/overnight-rehearsal-runner.json" >&2
  exit 1
fi

echo "overnight_rehearsal_runner=ready"
echo "summary=$OUT/overnight-rehearsal-runner.json"
echo "allowed_next_action=$allowed_next_action"
