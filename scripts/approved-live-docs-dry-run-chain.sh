#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
usage: scripts/approved-live-docs-dry-run-chain.sh --out <public-safe-relative-dir>

Builds a fixture-only approved docs-only live-mutation dry-run chain:
request -> approval ticket -> Foundry approval gate -> Forge guard ->
AO2 docs-only patch packet -> worktree preparation -> rollback execution
rehearsal -> Sentinel verdict -> Promoter boundary -> AO Command readback.

This script never schedules, executes, approves, publishes, uploads, calls
providers, creates branches, applies patches to this repository, or mutates
repositories. A ready result means the approved docs-only chain is ready for
the next PR rehearsal gate, not that live mutation has been performed.
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
  /*|~*|*"/.."*|*".."/*|tmp/*)
    echo "--out must be a public-safe relative path outside tmp/" >&2
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
  jq -r '.status // .verdict // .approval_state // "unknown"' "$1"
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

APPROVAL_REQUEST_SRC="examples/live-docs-approval/request.json"
APPROVAL_TICKET_SRC="examples/live-docs-approval/ticket-approved.json"
APPROVAL_REQUEST="$OUT/evidence/approval-request.json"
APPROVAL_TICKET="$OUT/evidence/approval-ticket.json"
FOUNDRY_APPROVAL_GATE="$OUT/evidence/foundry-approval-gate.json"
FORGE_GUARD="$OUT/evidence/forge-execution-guard.json"
AO2_PATCH_PACKET="$OUT/evidence/ao2-docs-only-patch-packet.json"
WORKTREE_PREPARE="$OUT/evidence/worktree-prepare.json"
ROLLBACK_REHEARSAL="$OUT/evidence/rollback-execution-rehearsal.json"
SENTINEL_VERDICT="$OUT/evidence/sentinel-verdict.json"
PROMOTER_BOUNDARY="$OUT/evidence/promoter-boundary.json"
COMMAND_READBACK="$OUT/evidence/command-readback.json"
SOURCE_ARTIFACTS_JSONL="$OUT/source-artifacts.jsonl"
SOURCE_ARTIFACTS_JSON="$OUT/source-artifacts.json"
SUMMARY="$OUT/summary.json"

cp "$APPROVAL_REQUEST_SRC" "$APPROVAL_REQUEST"
cp "$APPROVAL_TICKET_SRC" "$APPROVAL_TICKET"

scripts/live-docs-approval-gate.sh \
  --request "$APPROVAL_REQUEST" \
  --ticket "$APPROVAL_TICKET" \
  --out "$FOUNDRY_APPROVAL_GATE" > "$OUT/foundry-approval-gate.stdout"

jq -n \
  --arg schema_version "ao.forge.live-docs-execution-guard.v0.1" \
  --arg approval_gate_path "$FOUNDRY_APPROVAL_GATE" \
  --arg approval_gate_sha256 "$(sha256_file "$FOUNDRY_APPROVAL_GATE")" \
  '{
    schema_version:$schema_version,
    status:"ready",
    mode:"dry_run_only",
    first_live_class:"docs_only",
    requires_foundry_approval_gate:true,
    requires_covenant_ticket:true,
    requires_docs_only_allowlist:true,
    requires_clean_worktree:true,
    requires_rollback_plan:true,
    requires_sentinel_no_hold:true,
    requires_command_readback:true,
    source_hashes:[
      {name:"foundry_approval_gate", path:$approval_gate_path, schema_version:"ao.foundry.live-docs-approval-gate.v0.1", sha256:$approval_gate_sha256}
    ],
    allowed_path_classes:["docs_markdown_only"],
    forbidden_path_classes:["code","scripts","contracts","ci","release","secrets","credentials"],
    authority_boundaries:{
      dry_run_only:true,
      live_mutation_allowed:false,
      mutates_repositories:false,
      creates_worktree:false,
      creates_branch:false,
      schedules_work:false,
      executes_work:false,
      approves_work:false,
      provider_calls_allowed:false,
      release_or_publish_allowed:false
    }
  }' > "$FORGE_GUARD"

jq -n \
  --arg schema_version "ao2.docs-only-patch-packet.v1" \
  --arg forge_guard_path "$FORGE_GUARD" \
  --arg forge_guard_sha256 "$(sha256_file "$FORGE_GUARD")" \
  '{
    schema_version:$schema_version,
    status:"ready",
    mode:"dry_run_only",
    first_live_class:"docs_only",
    changed_file_plan:["docs/operations/first-live-docs-rehearsal.md"],
    verification_plan:["go test ./... -count=1","go vet ./...","git diff --check"],
    dry_run_apply:true,
    rollback_patch_present:true,
    forbidden_path_checks:["code","scripts","contracts","ci","release","credentials"],
    provider_session_boundary:"no_provider_calls",
    source_hashes:[
      {name:"forge_execution_guard", path:$forge_guard_path, schema_version:"ao.forge.live-docs-execution-guard.v0.1", sha256:$forge_guard_sha256}
    ],
    authority_boundaries:{
      dry_run_only:true,
      live_mutation_allowed:false,
      mutates_repositories:false,
      applies_patch_to_live_repo:false,
      creates_branch:false,
      schedules_work:false,
      executes_work:false,
      approves_work:false,
      provider_calls_allowed:false,
      release_or_publish_allowed:false
    }
  }' > "$AO2_PATCH_PACKET"

scripts/live-docs-worktree-prepare.sh \
  --candidate examples/live-docs-worktree-prepare/ready.candidate.json \
  --approval-gate "$FOUNDRY_APPROVAL_GATE" \
  --out "$WORKTREE_PREPARE" > "$OUT/worktree-prepare.stdout"

scripts/live-docs-rollback-execution-rehearsal.sh \
  --candidate examples/live-docs-rollback-execution/docs-only.candidate.json \
  --out "$ROLLBACK_REHEARSAL" > "$OUT/rollback-execution-rehearsal.stdout"

jq -n \
  --arg schema_version "ao.sentinel.live-docs-mutation-hold.v0.1" \
  --arg rollback_path "$ROLLBACK_REHEARSAL" \
  --arg rollback_sha256 "$(sha256_file "$ROLLBACK_REHEARSAL")" \
  '{
    schema_version:$schema_version,
    status:"clear",
    verdict:"clear",
    first_live_class:"docs_only",
    hold_required:false,
    blockers:[],
    required_evidence:["approval_gate","forge_guard","ao2_patch_packet","worktree_prepare","rollback_execution_rehearsal","command_readback"],
    source_hashes:[
      {name:"rollback_execution_rehearsal", path:$rollback_path, schema_version:"ao.foundry.live-docs-rollback-execution-rehearsal.v0.1", sha256:$rollback_sha256}
    ],
    authority_boundaries:{
      operator_mode:"read_only",
      dry_run_only:true,
      live_mutation_allowed:false,
      mutates_repositories:false,
      schedules_work:false,
      executes_work:false,
      approves_work:false,
      provider_calls_allowed:false,
      release_or_publish_allowed:false
    }
  }' > "$SENTINEL_VERDICT"

jq -n \
  --arg schema_version "ao.promoter.live-docs-mutation-boundary.v0.1" \
  --arg sentinel_path "$SENTINEL_VERDICT" \
  --arg sentinel_sha256 "$(sha256_file "$SENTINEL_VERDICT")" \
  '{
    schema_version:$schema_version,
    status:"passed",
    first_live_class:"docs_only",
    dry_run_only:true,
    gate_results:["approval_ticket_exact_scope","foundry_gate_ready","forge_guard_ready","ao2_packet_ready","sentinel_clear","rollback_rehearsal_ready"],
    blockers:[],
    required_followups:["run_first_live_docs_pr_rehearsal_gate_before_any_live_branch_or_pr"],
    live_docs_pr_rehearsal_boundary_ready:true,
    live_mutation_activation_allowed:false,
    source_hashes:[
      {name:"sentinel_verdict", path:$sentinel_path, schema_version:"ao.sentinel.live-docs-mutation-hold.v0.1", sha256:$sentinel_sha256}
    ],
    authority_boundaries:{
      dry_run_only:true,
      live_mutation_allowed:false,
      mutates_repositories:false,
      schedules_work:false,
      executes_work:false,
      approves_work:false,
      provider_calls_allowed:false,
      release_or_publish_allowed:false,
      broad_live_mutation_allowed:false,
      fully_unsupervised_complex_mutation_claimed:false
    }
  }' > "$PROMOTER_BOUNDARY"

jq -n \
  --arg schema_version "ao.command.live-docs-mutation-status.v0.1" \
  --arg promoter_path "$PROMOTER_BOUNDARY" \
  --arg promoter_sha256 "$(sha256_file "$PROMOTER_BOUNDARY")" \
  '{
    schema_version:$schema_version,
    status:"ready",
    combined_status:"ready",
    first_live_class:"docs_only",
    approval_state:"approved",
    allowed_next_action:"run_live_docs_pr_rehearsal_gate",
    first_failing_check:"",
    blocking_next_actions:[],
    maintenance_suggestions:["Keep this path docs-only and approval-bound.","Do not start a live branch until the PR rehearsal gate also passes."],
    operator_mode:"read_only",
    mutates_repositories:false,
    kill_switch_state:"armed",
    source_hashes:[
      {name:"promoter_boundary", path:$promoter_path, schema_version:"ao.promoter.live-docs-mutation-boundary.v0.1", sha256:$promoter_sha256}
    ],
    authority_boundaries:{
      readback_only:true,
      dry_run_only:true,
      live_mutation_allowed:false,
      schedules_work:false,
      executes_work:false,
      approves_work:false,
      provider_calls_allowed:false,
      release_or_publish_allowed:false
    }
  }' > "$COMMAND_READBACK"

write_source_artifacts "$SOURCE_ARTIFACTS_JSONL" \
  "approval_request=$APPROVAL_REQUEST" \
  "approval_ticket=$APPROVAL_TICKET" \
  "foundry_approval_gate=$FOUNDRY_APPROVAL_GATE" \
  "forge_execution_guard=$FORGE_GUARD" \
  "ao2_docs_only_patch_packet=$AO2_PATCH_PACKET" \
  "worktree_prepare=$WORKTREE_PREPARE" \
  "rollback_execution_rehearsal=$ROLLBACK_REHEARSAL" \
  "sentinel_verdict=$SENTINEL_VERDICT" \
  "promoter_boundary=$PROMOTER_BOUNDARY" \
  "command_readback=$COMMAND_READBACK"
jq -s '.' "$SOURCE_ARTIFACTS_JSONL" > "$SOURCE_ARTIFACTS_JSON"

if jq -e 'all(.[]; (.status == "ready" or .status == "approved" or .status == "clear" or .status == "passed" or .status == "pending_operator_approval"))' "$SOURCE_ARTIFACTS_JSON" >/dev/null; then
  status="ready"
  first_failing_check=""
else
  status="blocked"
  first_failing_check="source_artifact_status"
fi

jq -n \
  --arg schema_version "ao.foundry.approved-live-docs-dry-run-chain.v0.1" \
  --arg status "$status" \
  --arg first_failing_check "$first_failing_check" \
  --slurpfile source_artifacts "$SOURCE_ARTIFACTS_JSON" \
  '{
    schema_version:$schema_version,
    status:$status,
    mode:"fixture_only_dry_run",
    first_live_class:"docs_only",
    objective:"Prove the approved docs-only dry-run chain before any first live docs-only PR rehearsal gate.",
    chain:[
      "request",
      "approval ticket",
      "Foundry approval gate",
      "Forge guard",
      "AO2 docs-only patch packet",
      "worktree preparation",
      "rollback execution rehearsal",
      "Sentinel verdict",
      "Promoter boundary",
      "AO Command readback"
    ],
    source_artifacts:$source_artifacts[0],
    first_failing_check:$first_failing_check,
    blocking_next_actions:(if $status == "ready" then [] else ["repair the first non-ready source artifact"] end),
    maintenance_suggestions:[
      "Use the next live docs PR rehearsal gate before creating any real branch or PR.",
      "Keep the class docs-only and exact-scope approved.",
      "Do not treat this dry-run chain as broad live mutation authority."
    ],
    readiness_assessment:{
      approved_docs_only_dry_run_chain:(if $status == "ready" then "ready" else "blocked" end),
      first_live_class:"docs_only",
      approval_ticket_present:true,
      approval_scope_exact:true,
      safe_to_request:($status == "ready"),
      safe_to_execute:false,
      requires_live_docs_pr_rehearsal_gate:true,
      exact_next_step:(if $status == "ready" then "run_live_docs_pr_rehearsal_gate" else "repair_approved_live_docs_dry_run_chain" end),
      live_mutation_performed:false,
      fully_unsupervised_complex_mutation_claimed:false
    },
    authority_boundaries:{
      dry_run_only:true,
      live_mutation_allowed:false,
      mutates_repositories:false,
      creates_branch:false,
      creates_worktree:false,
      schedules_work:false,
      executes_work:false,
      approves_work:false,
      provider_calls_allowed:false,
      release_or_publish_allowed:false,
      broad_live_mutation_allowed:false,
      fully_unsupervised_complex_mutation_claimed:false
    }
  }' > "$SUMMARY"

jq empty "$SUMMARY" "$SOURCE_ARTIFACTS_JSON" "$OUT"/evidence/*.json

if [[ "$status" != "ready" ]]; then
  echo "approved_live_docs_dry_run_chain=$status" >&2
  echo "summary=$SUMMARY" >&2
  exit 1
fi

echo "approved_live_docs_dry_run_chain=ready"
echo "summary=$SUMMARY"
