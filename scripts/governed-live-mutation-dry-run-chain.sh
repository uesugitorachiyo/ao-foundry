#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
usage: scripts/governed-live-mutation-dry-run-chain.sh --out <public-safe-relative-dir>

Builds a fixture-only governed live-mutation dry-run chain:
Blueprint/Atlas complex task -> Foundry gate -> Covenant authority dry-run ->
Forge dry-run plan -> AO2 dry-run packet -> worktree isolation -> rollback
rehearsal -> Sentinel hold verdict -> Promoter boundary -> AO Command readback.

This script never schedules, executes, approves, publishes, uploads, calls
providers, creates branches, applies patches, or mutates repositories.
USAGE
}

OUT=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --out)
      OUT="${2:-}"
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
  /*|~*|*"/.."*|*".."/*)
    echo "--out must be a public-safe relative path" >&2
    exit 2
    ;;
esac

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required" >&2
  exit 2
fi

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

artifact_status() {
  jq -r '.status // .verdict // "unknown"' "$1"
}

artifact_schema() {
  jq -r '.schema_version // "unknown"' "$1"
}

write_source_artifacts() {
  local out="$1"
  shift
  : > "$out"
  for item in "$@"; do
    local name path schema status sha
    name="${item%%=*}"
    path="${item#*=}"
    schema="$(artifact_schema "$path")"
    status="$(artifact_status "$path")"
    sha="$(sha256_file "$path")"
    jq -n \
      --arg name "$name" \
      --arg path "$path" \
      --arg schema_version "$schema" \
      --arg status "$status" \
      --arg sha256 "$sha" \
      '{name:$name,path:$path,schema_version:$schema_version,status:$status,sha256:$sha256}' >> "$out"
  done
}

mkdir -p "$OUT/evidence"

BLUEPRINT_ATLAS="$OUT/evidence/blueprint-atlas-complex-task.json"
FOUNDRY_GATE="$OUT/evidence/foundry-start-gate.json"
COVENANT_AUTHORITY="$OUT/evidence/covenant-authority.json"
FORGE_PLAN="$OUT/evidence/forge-dry-run-plan.json"
AO2_PACKET="$OUT/evidence/ao2-dry-run-packet.json"
WORKTREE_ISOLATION="$OUT/evidence/worktree-isolation.json"
ROLLBACK_REHEARSAL="$OUT/evidence/rollback-rehearsal.json"
SENTINEL_HOLD="$OUT/evidence/sentinel-hold.json"
COMMAND_READBACK="$OUT/evidence/command-readback.json"
PROMOTER_BOUNDARY="$OUT/evidence/promoter-boundary.json"
SOURCE_ARTIFACTS_JSONL="$OUT/source-artifacts.jsonl"
SOURCE_ARTIFACTS_JSON="$OUT/source-artifacts.json"
SUMMARY="$OUT/summary.json"

jq -n \
  --arg schema_version "ao.foundry.blueprint-atlas-complex-task.v0.1" \
  '{
    schema_version:$schema_version,
    status:"ready",
    mode:"fixture_only_dry_run",
    blueprint_role:"requirements_interview_and_build_authorization",
    atlas_role:"oversized_objective_workgraph_context_pack_compiler",
    complex_task_ref:"examples/complex-refactor-workgraph/workgraph.json",
    mutates_repositories:false,
    schedules_work:false,
    executes_work:false,
    approves_work:false,
    provider_calls_allowed:false,
    release_or_publish_allowed:false
  }' > "$BLUEPRINT_ATLAS"

jq -n \
  --arg schema_version "ao.foundry.pulse-overnight-start-gate.v0.1" \
  '{
    schema_version:$schema_version,
    status:"ready",
    allowed_next_action:"start_next_slice",
    first_failing_check:"",
    blocking_next_actions:[],
    maintenance_suggestions:["keep start gate dry-run until live-mutation authority is separately approved"],
    mutates_repositories:false,
    schedules_work:false,
    executes_work:false,
    approves_work:false
  }' > "$FOUNDRY_GATE"

jq -n \
  --arg schema_version "covenant.live-mutation-authority.v1" \
  '{
    schema_version:$schema_version,
    status:"approved",
    mode:"dry_run_only",
    scope:"docs_only_fixture",
    repo:"ao-foundry",
    allowed_path_class:"docs",
    rollback_plan_required:true,
    operator_kill_switch_required:true,
    mutates_live_state:false,
    mutates_repositories:false,
    schedules_work:false,
    executes_work:false,
    approves_work:false,
    provider_calls_allowed:false,
    release_or_publish_allowed:false
  }' > "$COVENANT_AUTHORITY"

jq -n \
  --arg schema_version "ao.forge.live-mutation-dry-run-plan.v0.1" \
  '{
    schema_version:$schema_version,
    status:"ready",
    mode:"dry_run_only",
    job_class:"docs_only_fixture",
    requires_covenant_authority:true,
    requires_isolated_branch:true,
    requires_pr_lifecycle:true,
    mutates_live_state:false,
    mutates_repositories:false,
    schedules_work:false,
    executes_work:false,
    approves_work:false,
    provider_calls_allowed:false,
    release_or_publish_allowed:false
  }' > "$FORGE_PLAN"

jq -n \
  --arg schema_version "ao2.live-mutation-dry-run-packet.v1" \
  '{
    schema_version:$schema_version,
    status:"ready",
    mode:"dry_run_only",
    changed_file_plan:["docs/operations/OVERNIGHT-REFRACTOR-REHEARSAL-RUNBOOK.md"],
    verification_plan:["go test ./... -count=1","go vet ./...","git diff --check"],
    rollback_artifact_present:true,
    provider_session_boundary:"no_provider_calls",
    mutates_live_state:false,
    mutates_repositories:false,
    schedules_work:false,
    executes_work:false,
    approves_work:false,
    provider_calls_allowed:false,
    release_or_publish_allowed:false
  }' > "$AO2_PACKET"

scripts/live-mutation-worktree-isolation-proof.sh \
  --candidate examples/live-mutation-worktree-isolation/clean-isolated.candidate.json \
  --out "$WORKTREE_ISOLATION" > "$OUT/worktree-isolation.stdout"

scripts/live-mutation-rollback-rehearsal.sh \
  --candidate examples/live-mutation-rollback/docs-only-rollback.candidate.json \
  --out "$ROLLBACK_REHEARSAL" > "$OUT/rollback-rehearsal.stdout"

jq -n \
  --arg schema_version "ao.sentinel.live-mutation-hold.v0.1" \
  '{
    schema_version:$schema_version,
    status:"clear",
    hold_required:false,
    promoter_hold_required:false,
    rollback_recommended:false,
    blockers:[],
    recommended_actions:[],
    operator_mode:"read_only",
    mutates_live_state:false,
    mutates_repositories:false,
    schedules_work:false,
    executes_work:false,
    approves_work:false,
    provider_calls_allowed:false,
    release_or_publish_allowed:false
  }' > "$SENTINEL_HOLD"

jq -n \
  --arg schema_version "ao.command.live-mutation-status.v0.1" \
  '{
    schema_version:$schema_version,
    status:"ready",
    allowed_next_action:"request_first_tiny_live_mutation_class",
    kill_switch_state:"armed",
    operator_mode:"read_only",
    mutates_live_state:false,
    mutates_repositories:false,
    schedules_work:false,
    executes_work:false,
    approves_work:false,
    calls_providers:false,
    provider_calls_allowed:false,
    release_or_publish_allowed:false
  }' > "$COMMAND_READBACK"

jq -n \
  --arg schema_version "ao.promoter.live-mutation-boundary.v0.1" \
  '{
    schema_version:$schema_version,
    status:"passed",
    gate_results:[],
    blockers:[],
    required_followups:[],
    live_mutation_activation_allowed:true,
    dry_run_only:true,
    mutates_live_state:false,
    mutates_repositories:false,
    schedules_work:false,
    executes_work:false,
    approves_work:false,
    provider_calls_allowed:false,
    release_or_publish_allowed:false,
    operator_approval_still_required:true,
    first_tiny_live_class_still_gated:true
  }' > "$PROMOTER_BOUNDARY"

write_source_artifacts "$SOURCE_ARTIFACTS_JSONL" \
  "blueprint_atlas_complex_task=$BLUEPRINT_ATLAS" \
  "foundry_start_gate=$FOUNDRY_GATE" \
  "covenant_authority=$COVENANT_AUTHORITY" \
  "forge_dry_run_plan=$FORGE_PLAN" \
  "ao2_dry_run_packet=$AO2_PACKET" \
  "worktree_isolation=$WORKTREE_ISOLATION" \
  "rollback_rehearsal=$ROLLBACK_REHEARSAL" \
  "sentinel_hold=$SENTINEL_HOLD" \
  "promoter_boundary=$PROMOTER_BOUNDARY" \
  "command_readback=$COMMAND_READBACK"
jq -s '.' "$SOURCE_ARTIFACTS_JSONL" > "$SOURCE_ARTIFACTS_JSON"

if jq -e 'all(.[]; (.status == "ready" or .status == "approved" or .status == "clear" or .status == "passed"))' "$SOURCE_ARTIFACTS_JSON" >/dev/null; then
  status="ready"
  first_failing_check=""
else
  status="blocked"
  first_failing_check="source_artifact_status"
fi

jq -n \
  --arg schema_version "ao.foundry.governed-live-mutation-dry-run-chain.v0.1" \
  --arg status "$status" \
  --arg first_failing_check "$first_failing_check" \
  --slurpfile source_artifacts "$SOURCE_ARTIFACTS_JSON" \
  '{
    schema_version:$schema_version,
    status:$status,
    mode:"fixture_only_dry_run",
    objective:"Prove governed live-mutation readiness evidence without performing live mutation.",
    chain:[
      "Blueprint/Atlas complex task",
      "Foundry start gate",
      "Covenant authority dry-run",
      "Forge dry-run plan",
      "AO2 dry-run packet",
      "worktree isolation",
      "rollback rehearsal",
      "Sentinel hold verdict",
      "Promoter activation boundary",
      "AO Command readback"
    ],
    source_artifacts:$source_artifacts[0],
    first_failing_check:$first_failing_check,
    blocking_next_actions:(if $status == "ready" then [] else ["repair the first non-ready source artifact"] end),
    maintenance_suggestions:[
      "Keep this chain dry-run until a separate operator request authorizes the first tiny live mutation class.",
      "Do not treat this chain as ungated live mutation authority."
    ],
    readiness_assessment:{
      oversized_task_management:"proven_by_fixture_chain",
      governed_live_mutation:"dry_run_ready_for_request",
      first_tiny_live_mutation_class_safe_to_request:($status == "ready"),
      live_mutation_performed:false,
      ungated_live_mutation_claim:false
    },
    authority_boundaries:{
      dry_run_only:true,
      live_mutation_allowed:false,
      mutates_repositories:false,
      schedules_work:false,
      executes_work:false,
      approves_work:false,
      provider_calls_allowed:false,
      release_or_publish_allowed:false
    }
  }' > "$SUMMARY"

jq empty "$SUMMARY" "$SOURCE_ARTIFACTS_JSON" "$OUT"/evidence/*.json

if [[ "$status" != "ready" ]]; then
  echo "governed_live_mutation_dry_run_chain=$status" >&2
  echo "summary=$SUMMARY" >&2
  exit 1
fi

echo "governed_live_mutation_dry_run_chain=ready"
echo "summary=$SUMMARY"
