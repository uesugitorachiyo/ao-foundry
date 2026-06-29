#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
usage: scripts/blueprint-atlas-pulse-e2e-dry-run.sh --out <public-safe-relative-dir> [--ao-command-root <relative-dir>]

Runs a fixture-only Blueprint -> Atlas -> Foundry -> AO Command Pulse dry run.
The script does not schedule, execute, approve, publish, upload, call providers,
or mutate sibling repositories.
USAGE
}

OUT=""
AO_COMMAND_ROOT="../ao-command"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --out)
      OUT="${2:-}"
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

case "$AO_COMMAND_ROOT" in
  /*|~*|*"/.."*|*".."/*)
    if [[ "$AO_COMMAND_ROOT" != "../ao-command" ]]; then
      echo "--ao-command-root must be a public-safe relative sibling path" >&2
      exit 2
    fi
    ;;
esac

if [[ ! -f "$AO_COMMAND_ROOT/go.mod" ]]; then
  echo "AO Command checkout not found at $AO_COMMAND_ROOT" >&2
  exit 2
fi

mkdir -p "$OUT/ready" "$OUT/blocked"

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

json_status() {
  jq -r '.status' "$1"
}

require_status() {
  local path="$1"
  local want="$2"
  local got
  got="$(json_status "$path")"
  if [[ "$got" != "$want" ]]; then
    echo "$path status=$got want=$want" >&2
    exit 1
  fi
}

write_source_digest() {
  local name="$1"
  local path="$2"
  jq -n \
    --arg name "$name" \
    --arg path "$path" \
    --arg sha256 "$(sha256_file "$path")" \
    '{name:$name,path:$path,sha256:$sha256}'
}

READY_ATLAS_READBACK="$OUT/ready/atlas-readback.json"
READY_ATLAS_STATUS="$OUT/ready/atlas-status.json"
READY_PREFLIGHT="$OUT/ready/pulse-intake-preflight.json"
READY_START_GATE="$OUT/ready/pulse-overnight-start-gate.json"
READY_PULSE_DIR="$OUT/ready/pulse-run"
READY_COMMAND_STATUS="$OUT/ready/ao-command-pulse-status.json"

go run ./cmd/foundry atlas readback \
  --import examples/atlas/foundry-import.json \
  --run-link examples/atlas/run-link.completed.json \
  --out "$READY_ATLAS_READBACK" >/dev/null

go run ./cmd/foundry atlas status \
  --registry examples/registry/atlas-demo.foundry-registry.json \
  --import examples/atlas/foundry-import.json \
  --run-link examples/atlas/run-link.completed.json \
  --out "$READY_ATLAS_STATUS" >/dev/null

go run ./cmd/foundry pulse intake-preflight \
  --blueprint-authorization examples/pulse-intake/blueprint-authorization.ready.json \
  --requires-atlas \
  --atlas-import examples/atlas/foundry-import.json \
  --atlas-status "$READY_ATLAS_STATUS" \
  --out "$READY_PREFLIGHT" >/dev/null

go run ./cmd/foundry pulse overnight-start-gate \
  --intake-preflight "$READY_PREFLIGHT" \
  --lifecycle examples/pulse-lifecycle/ready-to-start-next-slice.json \
  --out "$READY_START_GATE" >/dev/null

go run ./cmd/foundry pulse run \
  --start-gate "$READY_START_GATE" \
  --out "$READY_PULSE_DIR" >/dev/null

(cd "$AO_COMMAND_ROOT" && \
  go run ./cmd/ao-command pulse status \
    --preflight "../ao-foundry/$READY_PREFLIGHT" \
    --lifecycle ../ao-foundry/examples/pulse-lifecycle/ready-to-start-next-slice.json \
    --start-gate "../ao-foundry/$READY_START_GATE" \
    --json) > "$READY_COMMAND_STATUS"

BLOCKED_PREFLIGHT="$OUT/blocked/pulse-intake-preflight.json"
BLOCKED_START_GATE="$OUT/blocked/pulse-overnight-start-gate.json"
BLOCKED_PULSE_DIR="$OUT/blocked/pulse-run"
BLOCKED_COMMAND_STATUS="$OUT/blocked/ao-command-pulse-status.json"

if go run ./cmd/foundry pulse intake-preflight \
  --blueprint-request examples/pulse-intake/blueprint-request.blocked.json \
  --out "$BLOCKED_PREFLIGHT" >/dev/null 2>"$OUT/blocked/pulse-intake-preflight.stderr"; then
  echo "blocked Blueprint path unexpectedly returned ready preflight" >&2
  exit 1
fi

if [[ ! -f "$BLOCKED_PREFLIGHT" ]]; then
  echo "blocked Blueprint path did not write preflight artifact" >&2
  exit 1
fi

go run ./cmd/foundry pulse overnight-start-gate \
  --intake-preflight "$BLOCKED_PREFLIGHT" \
  --lifecycle examples/pulse-lifecycle/ready-to-start-next-slice.json \
  --out "$BLOCKED_START_GATE" >/dev/null

if go run ./cmd/foundry pulse run \
  --start-gate "$BLOCKED_START_GATE" \
  --out "$BLOCKED_PULSE_DIR" >/dev/null 2>"$OUT/blocked/pulse-run.stderr"; then
  echo "blocked Blueprint path unexpectedly started implementation" >&2
  exit 1
fi

if [[ -f "$BLOCKED_PULSE_DIR/pulse-event.json" ]]; then
  echo "blocked Blueprint path unexpectedly wrote pulse-event.json" >&2
  exit 1
fi

(cd "$AO_COMMAND_ROOT" && \
  go run ./cmd/ao-command pulse status \
    --preflight "../ao-foundry/$BLOCKED_PREFLIGHT" \
    --lifecycle ../ao-foundry/examples/pulse-lifecycle/ready-to-start-next-slice.json \
    --start-gate "../ao-foundry/$BLOCKED_START_GATE" \
    --json) > "$BLOCKED_COMMAND_STATUS"

require_status "$READY_ATLAS_READBACK" "ready"
require_status "$READY_ATLAS_STATUS" "ready"
require_status "$READY_PREFLIGHT" "ready"
require_status "$READY_START_GATE" "ready"
require_status "$READY_PULSE_DIR/pulse-runner-start-decision.json" "ready"
require_status "$READY_PULSE_DIR/pulse-event.json" "ready"
require_status "$READY_COMMAND_STATUS" "ready"

require_status "$BLOCKED_PREFLIGHT" "blocked"
require_status "$BLOCKED_START_GATE" "blocked"
require_status "$BLOCKED_PULSE_DIR/pulse-runner-start-decision.json" "blocked"
require_status "$BLOCKED_COMMAND_STATUS" "blocked"

jq -n \
  --arg schema_version "ao.foundry.blueprint-atlas-pulse-e2e-dry-run.v0.1" \
  --arg status "ready" \
  --arg ready_next_action "$(jq -r '.allowed_next_action' "$READY_START_GATE")" \
  --arg blocked_next_action "$(jq -r '.allowed_next_action' "$BLOCKED_START_GATE")" \
  --arg ready_command_action "$(jq -r '.allowed_next_action' "$READY_COMMAND_STATUS")" \
  --arg blocked_command_action "$(jq -r '.allowed_next_action' "$BLOCKED_COMMAND_STATUS")" \
  --argjson source_digests "[
    $(write_source_digest blueprint_ready examples/pulse-intake/blueprint-authorization.ready.json),
    $(write_source_digest blueprint_blocked examples/pulse-intake/blueprint-request.blocked.json),
    $(write_source_digest atlas_import examples/atlas/foundry-import.json),
    $(write_source_digest atlas_run_link examples/atlas/run-link.completed.json),
    $(write_source_digest ready_start_gate "$READY_START_GATE"),
    $(write_source_digest ready_runner_decision "$READY_PULSE_DIR/pulse-runner-start-decision.json"),
    $(write_source_digest ready_command_status "$READY_COMMAND_STATUS"),
    $(write_source_digest blocked_start_gate "$BLOCKED_START_GATE"),
    $(write_source_digest blocked_runner_decision "$BLOCKED_PULSE_DIR/pulse-runner-start-decision.json"),
    $(write_source_digest blocked_command_status "$BLOCKED_COMMAND_STATUS")
  ]" \
  '{
    schema_version:$schema_version,
    status:$status,
    mode:"fixture_only_dry_run",
    mutates_repositories:false,
    schedules_work:false,
    executes_work:false,
    approves_work:false,
    calls_providers:false,
    ready_path:{
      starts_runner:true,
      allowed_next_action:$ready_next_action,
      command_allowed_next_action:$ready_command_action
    },
    blocked_blueprint_path:{
      starts_runner:false,
      allowed_next_action:$blocked_next_action,
      command_allowed_next_action:$blocked_command_action
    },
    source_digests:$source_digests,
    artifacts:{
      ready:{
        atlas_readback:"'$READY_ATLAS_READBACK'",
        atlas_status:"'$READY_ATLAS_STATUS'",
        pulse_intake_preflight:"'$READY_PREFLIGHT'",
        pulse_overnight_start_gate:"'$READY_START_GATE'",
        pulse_runner_start_decision:"'$READY_PULSE_DIR'/pulse-runner-start-decision.json",
        pulse_event:"'$READY_PULSE_DIR'/pulse-event.json",
        ao_command_pulse_status:"'$READY_COMMAND_STATUS'"
      },
      blocked:{
        pulse_intake_preflight:"'$BLOCKED_PREFLIGHT'",
        pulse_overnight_start_gate:"'$BLOCKED_START_GATE'",
        pulse_runner_start_decision:"'$BLOCKED_PULSE_DIR'/pulse-runner-start-decision.json",
        ao_command_pulse_status:"'$BLOCKED_COMMAND_STATUS'"
      }
    }
  }' > "$OUT/summary.json"

jq empty "$OUT"/ready/*.json "$OUT"/ready/pulse-run/*.json "$OUT"/blocked/*.json "$OUT"/blocked/pulse-run/*.json "$OUT/summary.json"

echo "blueprint_atlas_pulse_e2e=$OUT/summary.json"
echo "status=ready"
echo "ready_path=start_allowed"
echo "blocked_blueprint_path=start_denied"
