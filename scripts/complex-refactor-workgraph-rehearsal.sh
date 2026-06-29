#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
usage: scripts/complex-refactor-workgraph-rehearsal.sh --out <public-safe-relative-dir> [--ao-atlas-root ../ao-atlas] [--ao-command-root ../ao-command]

Runs a fixture-only rehearsal for an oversized AO stack refactor represented as
an AO Atlas workgraph. The script validates decomposition, dependency state,
Foundry import/readback, Pulse start-gate evidence, and AO Command readback.
It does not schedule, execute, approve, publish, upload, call providers, or
mutate sibling repositories.
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

if [[ "$AO_ATLAS_ROOT" != "../ao-atlas" ]]; then
  echo "--ao-atlas-root currently expects the public-safe sibling path ../ao-atlas" >&2
  exit 2
fi
if [[ "$AO_COMMAND_ROOT" != "../ao-command" ]]; then
  echo "--ao-command-root currently expects the public-safe sibling path ../ao-command" >&2
  exit 2
fi
if [[ ! -f "$AO_ATLAS_ROOT/go.mod" ]]; then
  echo "AO Atlas checkout not found at $AO_ATLAS_ROOT" >&2
  exit 2
fi
if [[ ! -f "$AO_COMMAND_ROOT/go.mod" ]]; then
  echo "AO Command checkout not found at $AO_COMMAND_ROOT" >&2
  exit 2
fi

WORKGRAPH="examples/complex-refactor-workgraph/workgraph.json"
INTAKE="examples/complex-refactor-workgraph/intake.json"
STACK_INSTANCE="examples/complex-refactor-workgraph/stack-instance.json"
FOUNDRY_IMPORT="examples/complex-refactor-workgraph/foundry-import.json"
RUN_LINK="examples/complex-refactor-workgraph/run-link.pulse-runner-split.completed.json"
BLOCKED_RUN_LINK="examples/complex-refactor-workgraph/run-link.command-readback.blocked.json"
NEEDS_CONTEXT_RUN_LINK="examples/complex-refactor-workgraph/run-link.command-readback.needs-context.json"
CONTEXT_DIR="examples/complex-refactor-workgraph/context-packs"
FOUNDRY_FROM_ATLAS="../ao-foundry"
FOUNDRY_FROM_COMMAND="../ao-foundry"

mkdir -p "$OUT"

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

atlas_run() {
  (cd "$AO_ATLAS_ROOT" && go run ./cmd/atlas "$@")
}

json_status() {
  jq -r '.status // .completion_status' "$1"
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

jq empty "$WORKGRAPH" "$INTAKE" "$STACK_INSTANCE" "$FOUNDRY_IMPORT" "$RUN_LINK" "$BLOCKED_RUN_LINK" "$NEEDS_CONTEXT_RUN_LINK" "$CONTEXT_DIR"/*.json

atlas_run workgraph validate --workgraph "$FOUNDRY_FROM_ATLAS/$WORKGRAPH" > "$OUT/atlas-workgraph-validate.txt"
atlas_run workgraph next --workgraph "$FOUNDRY_FROM_ATLAS/$WORKGRAPH" --json > "$OUT/atlas-next-ready.json"
atlas_run workgraph status --workgraph "$FOUNDRY_FROM_ATLAS/$WORKGRAPH" > "$OUT/atlas-workgraph-status.txt"

for pack in "$CONTEXT_DIR"/*.json; do
  atlas_run context-pack validate --pack "$FOUNDRY_FROM_ATLAS/$pack" > "$OUT/context-pack-$(basename "$pack" .json).txt"
done

atlas_run mission status \
  --intake "$FOUNDRY_FROM_ATLAS/$INTAKE" \
  --workgraph "$FOUNDRY_FROM_ATLAS/$WORKGRAPH" \
  --run-link "$FOUNDRY_FROM_ATLAS/$RUN_LINK" \
  --json > "$OUT/atlas-mission-status.json"

atlas_run workgraph repair-plan \
  --workgraph "$FOUNDRY_FROM_ATLAS/$WORKGRAPH" \
  --run-link "$FOUNDRY_FROM_ATLAS/$BLOCKED_RUN_LINK" \
  --out "$FOUNDRY_FROM_ATLAS/$OUT/atlas-repair-plan.json" > "$OUT/atlas-repair-plan.stdout"

jq '.nodes[] | select(.id == "command-readback-followup").factory_task' "$WORKGRAPH" > "$OUT/command-readback-task.json"
atlas_run context-pack repack \
  --task "$FOUNDRY_FROM_ATLAS/$OUT/command-readback-task.json" \
  --run-link "$FOUNDRY_FROM_ATLAS/$NEEDS_CONTEXT_RUN_LINK" \
  --source-ref "$CONTEXT_DIR/command-readback.context-pack.json" \
  --source-digest "sha256:$(sha256_file "$CONTEXT_DIR/command-readback.context-pack.json")" \
  --budget 4096 \
  --out "$FOUNDRY_FROM_ATLAS/$OUT/atlas-context-repack.json" > "$OUT/atlas-context-repack.stdout"

go run ./cmd/foundry atlas import validate --import "$FOUNDRY_IMPORT" > "$OUT/foundry-import-validate.txt"
go run ./cmd/foundry atlas readback \
  --import "$FOUNDRY_IMPORT" \
  --run-link "$RUN_LINK" \
  --out "$OUT/foundry-atlas-readback.json" >/dev/null

scripts/blueprint-atlas-pulse-e2e-dry-run.sh \
  --out "$OUT/pulse-gate" \
  --ao-command-root "$AO_COMMAND_ROOT" > "$OUT/pulse-gate.stdout"

require_status "$OUT/foundry-atlas-readback.json" "ready"
require_status "$OUT/pulse-gate/summary.json" "ready"
require_status "$OUT/atlas-repair-plan.json" "repair_required"

TOTAL_TASKS="$(jq '.nodes | length' "$WORKGRAPH")"
READY_TASKS="$(jq '[.nodes[] | select(.status == "ready")] | length' "$WORKGRAPH")"
BLOCKED_TASKS="$(jq '[.nodes[] | select(.status == "blocked")] | length' "$WORKGRAPH")"
COMPLETED_TASKS="$(jq '[.nodes[] | select(.status == "completed")] | length' "$WORKGRAPH")"
FAILED_TASKS="$(jq '[.nodes[] | select(.status == "failed")] | length' "$WORKGRAPH")"
NEXT_TASK_ID="$(jq -r '.factory_task.id' "$OUT/atlas-next-ready.json")"
NEXT_NODE_ID="$(jq -r '.id' "$OUT/atlas-next-ready.json")"
NEXT_FACTORY_REPO="$(jq -r '.factory_task.target_factory_repo' "$OUT/atlas-next-ready.json")"
READY_GATE_ACTION="$(jq -r '.ready_path.allowed_next_action' "$OUT/pulse-gate/summary.json")"
BLOCKED_GATE_ACTION="$(jq -r '.blocked_blueprint_path.allowed_next_action' "$OUT/pulse-gate/summary.json")"
REPAIR_TASK_ID="$(jq -r '.repair_tasks[0].id' "$OUT/atlas-repair-plan.json")"
REPACK_REASON="$(jq -r '.missing_context_reason' "$OUT/atlas-context-repack.json")"

if [[ "$READY_TASKS" -gt 0 && "$READY_GATE_ACTION" == "start_next_slice" ]]; then
  LOOP_MAY_START_NEXT_READY_TASK=true
else
  LOOP_MAY_START_NEXT_READY_TASK=false
fi

jq -n \
  --arg schema_version "ao.foundry.complex-refactor-workgraph-rehearsal.v0.1" \
  --arg status "ready" \
  --argjson total_tasks "$TOTAL_TASKS" \
  --argjson ready_tasks "$READY_TASKS" \
  --argjson blocked_tasks "$BLOCKED_TASKS" \
  --argjson completed_tasks "$COMPLETED_TASKS" \
  --argjson failed_tasks "$FAILED_TASKS" \
  --arg next_node_id "$NEXT_NODE_ID" \
  --arg next_task_id "$NEXT_TASK_ID" \
  --arg next_factory_repo "$NEXT_FACTORY_REPO" \
  --arg ready_gate_action "$READY_GATE_ACTION" \
  --arg blocked_gate_action "$BLOCKED_GATE_ACTION" \
  --argjson loop_may_start_next_ready_task "$LOOP_MAY_START_NEXT_READY_TASK" \
  --arg workgraph_sha "$(sha256_file "$WORKGRAPH")" \
  --arg import_sha "$(sha256_file "$FOUNDRY_IMPORT")" \
  --arg run_link_sha "$(sha256_file "$RUN_LINK")" \
  --arg blocked_run_link_sha "$(sha256_file "$BLOCKED_RUN_LINK")" \
  --arg needs_context_run_link_sha "$(sha256_file "$NEEDS_CONTEXT_RUN_LINK")" \
  --arg repair_plan_sha "$(sha256_file "$OUT/atlas-repair-plan.json")" \
  --arg context_repack_sha "$(sha256_file "$OUT/atlas-context-repack.json")" \
  --arg pulse_summary_sha "$(sha256_file "$OUT/pulse-gate/summary.json")" \
  --arg repair_task_id "$REPAIR_TASK_ID" \
  --arg repack_reason "$REPACK_REASON" \
  '{
    schema_version:$schema_version,
    status:$status,
    mode:"fixture_only_rehearsal",
    mutates_repositories:false,
    schedules_work:false,
    executes_work:false,
    approves_work:false,
    calls_providers:false,
    no_duplicated_stack_folders:true,
    task_counts:{
      total:$total_tasks,
      ready:$ready_tasks,
      blocked:$blocked_tasks,
      completed:$completed_tasks,
      failed:$failed_tasks
    },
    next_recommended_factory_task:{
      node_id:$next_node_id,
      task_id:$next_task_id,
      target_factory_repo:$next_factory_repo
    },
    loop_decision:{
      may_start_next_ready_task:$loop_may_start_next_ready_task,
      must_not_start_blocked_tasks:true,
      ready_gate_action:$ready_gate_action,
      blocked_blueprint_action:$blocked_gate_action,
      why:"A dependency-ready factory task exists and the ready Pulse gate allows start_next_slice; blocked tasks remain blocked until upstream run-link evidence is completed."
    },
    repair_plan:{
      status:"repair_required",
      path:"'$OUT'/atlas-repair-plan.json",
      repair_task_id:$repair_task_id,
      schedules_work:false,
      executes_work:false,
      approves_work:false
    },
    context_repack:{
      status:"ready",
      path:"'$OUT'/atlas-context-repack.json",
      missing_context_reason:$repack_reason,
      schedules_work:false,
      executes_work:false,
      approves_work:false
    },
    source_digests:[
      {name:"workgraph",path:"'$WORKGRAPH'",sha256:$workgraph_sha},
      {name:"foundry_import",path:"'$FOUNDRY_IMPORT'",sha256:$import_sha},
      {name:"run_link",path:"'$RUN_LINK'",sha256:$run_link_sha},
      {name:"blocked_run_link",path:"'$BLOCKED_RUN_LINK'",sha256:$blocked_run_link_sha},
      {name:"needs_context_run_link",path:"'$NEEDS_CONTEXT_RUN_LINK'",sha256:$needs_context_run_link_sha},
      {name:"repair_plan",path:"'$OUT'/atlas-repair-plan.json",sha256:$repair_plan_sha},
      {name:"context_repack",path:"'$OUT'/atlas-context-repack.json",sha256:$context_repack_sha},
      {name:"pulse_gate_summary",path:"'$OUT'/pulse-gate/summary.json",sha256:$pulse_summary_sha}
    ],
    artifacts:{
      atlas_next_ready:"'$OUT'/atlas-next-ready.json",
      atlas_mission_status:"'$OUT'/atlas-mission-status.json",
      repair_plan:"'$OUT'/atlas-repair-plan.json",
      context_repack:"'$OUT'/atlas-context-repack.json",
      foundry_atlas_readback:"'$OUT'/foundry-atlas-readback.json",
      pulse_gate_summary:"'$OUT'/pulse-gate/summary.json",
      command_readback:"'$OUT'/ao-command-complex-refactor-status.json"
    }
  }' > "$OUT/summary.json"

(cd "$AO_COMMAND_ROOT" && go run ./cmd/ao-command complex-refactor status \
  --summary "$FOUNDRY_FROM_COMMAND/$OUT/summary.json" \
  --json > "$FOUNDRY_FROM_COMMAND/$OUT/ao-command-complex-refactor-status.json")

jq empty "$OUT/atlas-next-ready.json" "$OUT/atlas-mission-status.json" "$OUT/atlas-repair-plan.json" "$OUT/atlas-context-repack.json" "$OUT/foundry-atlas-readback.json" "$OUT/pulse-gate/summary.json" "$OUT/summary.json" "$OUT/ao-command-complex-refactor-status.json"

echo "complex_refactor_workgraph_rehearsal=$OUT/summary.json"
echo "status=ready"
echo "total_tasks=$TOTAL_TASKS"
echo "ready_tasks=$READY_TASKS"
echo "blocked_tasks=$BLOCKED_TASKS"
echo "completed_tasks=$COMPLETED_TASKS"
echo "failed_tasks=$FAILED_TASKS"
echo "next_recommended_factory_task=$NEXT_TASK_ID"
echo "repair_task=$REPAIR_TASK_ID"
echo "context_repack_reason=$REPACK_REASON"
echo "command_readback=$OUT/ao-command-complex-refactor-status.json"
echo "loop_may_start_next_ready_task=$LOOP_MAY_START_NEXT_READY_TASK"
