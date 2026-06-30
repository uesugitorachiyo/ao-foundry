#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
usage: scripts/governed-live-mutation-dry-run-chain.sh --out <public-safe-relative-dir> [--mutation-class docs_only_multi_file|low_risk_code]

Builds a fixture-only governed live-mutation dry-run chain:
Blueprint/Atlas complex task -> Foundry gate -> Covenant authority dry-run ->
Forge dry-run plan -> AO2 dry-run packet -> worktree isolation -> rollback
rehearsal -> Sentinel hold verdict -> Promoter boundary -> AO Command readback.

This script never schedules, executes, approves, publishes, uploads, calls
providers, creates branches, applies patches, or mutates repositories.
USAGE
}

OUT=""
MUTATION_CLASS="docs_only_multi_file"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --out)
      OUT="${2:-}"
      shift 2
      ;;
    --mutation-class)
      MUTATION_CLASS="${2:-}"
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

case "$MUTATION_CLASS" in
  docs_only_multi_file|low_risk_code)
    ;;
  *)
    echo "--mutation-class must be docs_only_multi_file or low_risk_code" >&2
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
  jq -r '.status // .verdict // .approval_state // .state // "unknown"' "$1"
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

write_low_risk_code_chain() {
  mkdir -p "$OUT/evidence"

  local atlas_classification="examples/class-gate/atlas-classification.low-risk-code.json"
  local foundry_class_gate="examples/class-gate/gate.dry-run.low-risk-code.json"
  local covenant_ticket="examples/class-gate/covenant-ticket.low-risk-code.json"
  local rollback_proof="examples/class-gate/rollback.passed.low-risk-code.json"
  local promoter_ready="examples/class-gate/promoter.ready.low-risk-code.json"
  local test_only_success="examples/class-gate/test-only-success.low-risk-code.json"
  local forge_plan="$OUT/evidence/forge-low-risk-dry-run-plan.json"
  local ao2_packet="$OUT/evidence/ao2-low-risk-dry-run-packet.json"
  local sentinel_hold="$OUT/evidence/sentinel-low-risk-hold.json"
  local command_readback="$OUT/evidence/command-low-risk-readback.json"
  local source_artifacts_jsonl="$OUT/source-artifacts.jsonl"
  local source_artifacts_json="$OUT/source-artifacts.json"
  local summary="$OUT/summary.json"

  jq -n \
    --arg schema_version "ao.forge.live-mutation-dry-run-plan.v0.1" \
    '{
      schema_version:$schema_version,
      status:"ready",
      mode:"dry_run_only",
      job_class:"low_risk_code",
      mutation_class:"low_risk_code",
      current_mutation_class:"test_only",
      next_mutation_class:"low_risk_code",
      candidate_change:{
        source_file:"internal/atlas/validate.go",
        test_file:"internal/atlas/validate_test.go",
        summary:"tiny validation helper adjustment with matching test coverage"
      },
      patch_limits:{
        max_source_files:1,
        max_test_files:1,
        max_changed_files:2,
        requires_rollback_patch:true,
        requires_verification_commands:true,
        denied_path_classes:["scripts","ci_workflows","release","secrets","config_expansion","provider_paths","broad_refactors"]
      },
      verification_commands:["go test ./...","git diff --check"],
      rollback_patch_required:true,
      safe_to_request:true,
      safe_to_execute:false,
      mutates_live_state:false,
      mutates_repositories:false,
      schedules_work:false,
      executes_work:false,
      approves_work:false,
      provider_calls_allowed:false,
      release_or_publish_allowed:false
    }' > "$forge_plan"

  jq -n \
    --arg schema_version "ao2.live-mutation-dry-run-packet.v1" \
    '{
      schema_version:$schema_version,
      status:"ready",
      mode:"dry_run_only",
      change_class:"low_risk_code",
      mutation_class:"low_risk_code",
      current_mutation_class:"test_only",
      next_mutation_class:"low_risk_code",
      allowed_paths:["internal/atlas/validate.go","internal/atlas/validate_test.go"],
      forbidden_paths:[".github/","cmd/","config/","docs/","examples/","release/","scripts/","secrets/","internal/provider/","go.mod","go.sum"],
      rollback_patch:"examples/class-gate/rollback.low-risk-code.patch",
      verification_commands:["go test ./...","git diff --check"],
      expected_diff_limits:{max_source_files:1,max_test_files:1,max_changed_files:2,max_lines_changed:160},
      evidence_digests:["atlas_classification","covenant_class_ticket","foundry_class_gate","rollback_proof"],
      safe_to_request:true,
      safe_to_execute:false,
      mutates_live_state:false,
      mutates_repositories:false,
      schedules_work:false,
      executes_work:false,
      approves_work:false,
      provider_calls_allowed:false,
      release_or_publish_allowed:false
    }' > "$ao2_packet"

  jq -n \
    --arg schema_version "ao.sentinel.live-mutation-hold.v0.1" \
    '{
      schema_version:$schema_version,
      status:"clear",
      mutation_class:"low_risk_code",
      class_hold_verdict:{
        status:"clear",
        mutation_class:"low_risk_code",
        max_files:2,
        max_lines_changed:160,
        max_source_files:1,
        max_test_files:1,
        files_changed:2,
        source_files_changed:1,
        test_files_changed:1,
        lines_changed:22,
        forbidden_path_classes:[],
        test_coverage_status:"passed",
        rollback_status:"ready",
        diff_size_status:"passed",
        file_class_status:"passed",
        evidence_freshness_status:"fresh",
        ci_status:"passed",
        blockers:[]
      },
      hold_required:false,
      promoter_hold_required:false,
      rollback_recommended:false,
      first_failing_check:"",
      blockers:[],
      recommended_actions:[],
      source_artifacts:[],
      operator_mode:"read_only",
      mutates_live_state:false,
      mutates_repositories:false,
      schedules_work:false,
      executes_work:false,
      approves_work:false,
      provider_calls_allowed:false,
      release_or_publish_allowed:false
    }' > "$sentinel_hold"

  jq -n \
    --arg schema_version "ao.command.live-mutation-status.v0.1" \
    '{
      schema_version:$schema_version,
      status:"ready",
      allowed_next_action:"request_low_risk_code_dry_run",
      current_mutation_class:"test_only",
      next_mutation_class:"low_risk_code",
      safe_to_request:true,
      safe_to_execute:false,
      kill_switch_state:"armed",
      operator_mode:"read_only",
      mutates_live_state:false,
      mutates_repositories:false,
      schedules_work:false,
      executes_work:false,
      approves_work:false,
      calls_providers:false,
      provider_calls_allowed:false,
      release_or_publish_allowed:false,
      low_risk_code_denial_audit:{
        status:"blocked",
        next_denied_class:"low_risk_code",
        safe_to_request:true,
        safe_to_execute:false,
        exact_next_action:"build_low_risk_code_live_execution_gate",
        denial_reason:"dry-run chain is complete, but live code mutation remains denied until exact live policy, rollback, Sentinel, Promoter, Command, and CI evidence all exist."
      }
    }' > "$command_readback"

  write_source_artifacts "$source_artifacts_jsonl" \
    "atlas_classification=$atlas_classification" \
    "foundry_class_gate=$foundry_class_gate" \
    "covenant_class_ticket=$covenant_ticket" \
    "forge_dry_run_plan=$forge_plan" \
    "ao2_dry_run_packet=$ao2_packet" \
    "rollback_proof=$rollback_proof" \
    "sentinel_hold=$sentinel_hold" \
    "promoter_ready=$promoter_ready" \
    "command_readback=$command_readback" \
    "test_only_success=$test_only_success"
  jq -s '.' "$source_artifacts_jsonl" > "$source_artifacts_json"

  local source_status_ok="false"
  if jq -e 'all(.[]; (.status == "ready" or .status == "approved" or .status == "clear" or .status == "passed"))' "$source_artifacts_json" >/dev/null; then
    source_status_ok="true"
  fi
  local safe_to_request safe_to_execute status first_failing_check
  safe_to_request="$(jq -r '.safe_to_request == true' "$foundry_class_gate")"
  safe_to_execute="$(jq -r '.safe_to_execute == true' "$foundry_class_gate")"
  if [[ "$source_status_ok" == "true" && "$safe_to_request" == "true" ]]; then
    status="ready"
    first_failing_check=""
  else
    status="blocked"
    first_failing_check="low_risk_code_source_artifact_status"
  fi

  jq -n \
    --arg schema_version "ao.foundry.governed-live-mutation-dry-run-chain.v0.1" \
    --arg status "$status" \
    --arg first_failing_check "$first_failing_check" \
    --argjson safe_to_request "$safe_to_request" \
    --argjson safe_to_execute "$safe_to_execute" \
    --slurpfile source_artifacts "$source_artifacts_json" \
    '{
      schema_version:$schema_version,
      status:$status,
      mode:"fixture_only_dry_run",
      mutation_class:"low_risk_code",
      current_proven_live_class:"test_only",
      next_denied_class:"low_risk_code",
      objective:"Prove the low_risk_code dry-run chain for one tiny source-plus-test mutation without executing it.",
      chain:[
        "Atlas classification",
        "Foundry class gate",
        "Covenant authority ticket",
        "Forge dry-run plan",
        "AO2 bounded patch packet",
        "rollback proof",
        "Sentinel hold verdict",
        "Promoter promotion boundary",
        "AO Command readback"
      ],
      candidate_change:{
        source_file:"internal/atlas/validate.go",
        test_file:"internal/atlas/validate_test.go",
        max_source_files:1,
        max_test_files:1
      },
      source_artifacts:$source_artifacts[0],
      first_failing_check:$first_failing_check,
      safe_to_request:$safe_to_request,
      safe_to_execute:$safe_to_execute,
      execution_denial_reason:(if $safe_to_execute then "" else "low_risk_code dry-run/readback is complete, but live code execution remains denied until live promotion evidence exists." end),
      exact_next_action:(if $safe_to_execute then "request_low_risk_code_live_rehearsal" else "build_low_risk_code_live_execution_gate" end),
      blocking_next_actions:(if $status == "ready" then [] else ["repair the first non-ready low_risk_code source artifact"] end),
      maintenance_suggestions:[
        "Keep this low_risk_code chain dry-run until a later slice promotes live code mutation.",
        "Do not execute the patch while safe_to_execute is false."
      ],
      readiness_assessment:{
        low_risk_code_dry_run_chain:($status == "ready"),
        includes_atlas:true,
        includes_covenant:true,
        includes_forge:true,
        includes_ao2:true,
        includes_rollback:true,
        includes_sentinel:true,
        includes_promoter:true,
        includes_command:true,
        safe_to_request:$safe_to_request,
        safe_to_execute:$safe_to_execute,
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
    }' > "$summary"

  jq empty "$summary" "$source_artifacts_json" "$OUT"/evidence/*.json

  if [[ "$status" != "ready" ]]; then
    echo "low_risk_code_dry_run_chain=$status" >&2
    echo "summary=$summary" >&2
    exit 1
  fi

  echo "low_risk_code_dry_run_chain=ready"
  echo "safe_to_request=$safe_to_request"
  echo "safe_to_execute=$safe_to_execute"
  echo "summary=$summary"
}

mkdir -p "$OUT/evidence"

if [[ "$MUTATION_CLASS" == "low_risk_code" ]]; then
  write_low_risk_code_chain
  exit 0
fi

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
