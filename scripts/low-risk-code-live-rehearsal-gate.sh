#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
usage: scripts/low-risk-code-live-rehearsal-gate.sh --chain <summary.json> --out <gate.json> [--atlas-blueprint-import <blueprint-import.json>] [--atlas-status <atlas-status.json>] [--live-policy-evidence <policy.json>] [--bounded-packet-proof <proof.json>] [--sentinel-verdict <verdict.json>] [--promoter-verdict <verdict.json>] [--command-readback <readback.json>] [--json]

Decides whether a first low_risk_code live rehearsal may start. This gate emits
evidence only. It never creates branches, creates worktrees, mutates
repositories, opens PRs, merges, publishes, uploads, releases, approves work,
or calls providers.
USAGE
}

chain=""
atlas_blueprint_import=""
atlas_status=""
live_policy_evidence=""
bounded_packet_proof=""
sentinel_verdict=""
promoter_verdict=""
command_readback=""
out=""
json=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --chain) chain="${2:-}"; shift 2 ;;
    --atlas-blueprint-import) atlas_blueprint_import="${2:-}"; shift 2 ;;
    --atlas-status) atlas_status="${2:-}"; shift 2 ;;
    --live-policy-evidence) live_policy_evidence="${2:-}"; shift 2 ;;
    --bounded-packet-proof) bounded_packet_proof="${2:-}"; shift 2 ;;
    --sentinel-verdict) sentinel_verdict="${2:-}"; shift 2 ;;
    --promoter-verdict) promoter_verdict="${2:-}"; shift 2 ;;
    --command-readback) command_readback="${2:-}"; shift 2 ;;
    --out) out="${2:-}"; shift 2 ;;
    --json) json=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "low-risk-code-live-rehearsal-gate: unknown argument $1" >&2; usage >&2; exit 2 ;;
  esac
done

if [[ -z "$chain" || -z "$out" ]]; then
  usage >&2
  exit 2
fi
if [[ "$out" == "$chain" || "$out" == "$atlas_blueprint_import" || "$out" == "$atlas_status" || "$out" == "$live_policy_evidence" || "$out" == "$bounded_packet_proof" || "$out" == "$sentinel_verdict" || "$out" == "$promoter_verdict" || "$out" == "$command_readback" ]]; then
  echo "low-risk-code-live-rehearsal-gate: --out must not overwrite input artifacts" >&2
  exit 2
fi
path_bundle="$chain:$atlas_blueprint_import:$atlas_status:$live_policy_evidence:$bounded_packet_proof:$sentinel_verdict:$promoter_verdict:$command_readback:$out"
private_mac_root="/""Users/"
private_linux_root="/""home/"
private_tmp_root="/""tmp/"
for unsafe_marker in "/.." "../" "~" "$private_mac_root" "$private_linux_root" "$private_tmp_root"; do
  if [[ "$path_bundle" == *"$unsafe_marker"* ]]; then
    echo "low-risk-code-live-rehearsal-gate: paths must be public-safe relative paths" >&2
    exit 2
  fi
done
if [[ ! -f "$chain" ]]; then
  echo "low-risk-code-live-rehearsal-gate: chain not found: $chain" >&2
  exit 2
fi
if [[ -n "$atlas_blueprint_import" && ! -f "$atlas_blueprint_import" ]]; then
  echo "low-risk-code-live-rehearsal-gate: Atlas Blueprint import not found: $atlas_blueprint_import" >&2
  exit 2
fi
if [[ -n "$atlas_status" && ! -f "$atlas_status" ]]; then
  echo "low-risk-code-live-rehearsal-gate: Atlas status not found: $atlas_status" >&2
  exit 2
fi
if [[ -n "$live_policy_evidence" && ! -f "$live_policy_evidence" ]]; then
  echo "low-risk-code-live-rehearsal-gate: live policy evidence not found: $live_policy_evidence" >&2
  exit 2
fi
for evidence_pair in \
  "bounded packet proof:$bounded_packet_proof" \
  "Sentinel verdict:$sentinel_verdict" \
  "Promoter verdict:$promoter_verdict" \
  "Command readback:$command_readback"; do
  evidence_label="${evidence_pair%%:*}"
  evidence_path="${evidence_pair#*:}"
  if [[ -n "$evidence_path" && ! -f "$evidence_path" ]]; then
    echo "low-risk-code-live-rehearsal-gate: $evidence_label not found: $evidence_path" >&2
    exit 2
  fi
done
if ! command -v jq >/dev/null 2>&1; then
  echo "low-risk-code-live-rehearsal-gate: jq is required" >&2
  exit 2
fi

sha256_file() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}' | sed 's/^\\//'
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}' | sed 's/^\\//'
  else
    echo "low-risk-code-live-rehearsal-gate: shasum or sha256sum is required" >&2
    exit 2
  fi
}

json_string() {
  jq -Rsa .
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
checks_file="$tmpdir/checks.jsonl"
: > "$checks_file"
first_failing_check=""

add_check() {
  local name="$1"
  local status="$2"
  local summary="$3"
  printf '{"name":%s,"status":%s,"summary":%s}\n' \
    "$(printf '%s' "$name" | json_string)" \
    "$(printf '%s' "$status" | json_string)" \
    "$(printf '%s' "$summary" | json_string)" >> "$checks_file"
  if [[ "$status" != "passed" && -z "$first_failing_check" ]]; then
    first_failing_check="$name"
  fi
}

validate_downstream_evidence() {
  local check_name="$1"
  local evidence_path="$2"
  local expected_schema="$3"
  local expected_status="$4"
  local missing_summary="$5"
  local passed_summary="$6"
  local blocked_summary="$7"
  if [[ -z "$evidence_path" ]]; then
    add_check "$check_name" "blocked" "$missing_summary"
    return
  fi
  if jq -e \
    --arg expected_schema "$expected_schema" \
    --arg expected_status "$expected_status" \
    --arg chain_sha "$chain_sha" \
    '
      .schema_version == $expected_schema and
      .status == $expected_status and
      .mutation_class == "low_risk_code" and
      .dry_run_chain_sha256 == $chain_sha and
      .safe_to_execute == true and
      (.live_execution_grants // false) == false and
      .candidate.repo == "ao-atlas" and
      .candidate.base_branch == "main" and
      .candidate.work_branch == "codex/low-risk-code-rehearsal-one" and
      .candidate.file_allowlist == ["internal/atlas/validate.go"] and
      .candidate.command_allowlist == ["git diff --check","go test ./..."] and
      .candidate.rollback_plan.strategy == "git_restore_exact_file" and
      .candidate.rollback_plan.files == ["internal/atlas/validate.go"]
    ' "$evidence_path" >/dev/null; then
    add_check "$check_name" "passed" "$passed_summary"
  else
    add_check "$check_name" "blocked" "$blocked_summary"
  fi
}

chain_sha="$(sha256_file "$chain")"
chain_schema="$(jq -r '.schema_version // ""' "$chain")"
chain_status="$(jq -r '.status // ""' "$chain")"
chain_class="$(jq -r '.mutation_class // ""' "$chain")"
chain_safe_to_request="$(jq -r 'if .safe_to_request == true then "true" else "false" end' "$chain")"
chain_safe_to_execute="$(jq -r 'if .safe_to_execute == true then "true" else "false" end' "$chain")"

if [[ "$chain_schema" == "ao.foundry.governed-live-mutation-dry-run-chain.v0.1" && "$chain_status" == "ready" && "$chain_class" == "low_risk_code" && "$chain_safe_to_request" == "true" ]]; then
  add_check "low_risk_code_dry_run_chain" "passed" "low_risk_code dry-run chain is ready and requestable"
else
  add_check "low_risk_code_dry_run_chain" "blocked" "low_risk_code dry-run chain must be ready before live rehearsal can be considered"
fi

atlas_blueprint_schema=""
atlas_blueprint_status=""
atlas_blueprint_class=""
atlas_blueprint_workgraph=""
atlas_blueprint_ready_for_foundry=""
atlas_blueprint_safe_to_execute=""
atlas_blueprint_live_proven=""
atlas_blueprint_downstream_digest=""
atlas_blueprint_pack_digest=""
atlas_blueprint_authorization_digest=""
atlas_blueprint_candidate_selection_digest=""
atlas_blueprint_mutation_class_model_digest=""
atlas_blueprint_sha=""
atlas_status_schema=""
atlas_status_status=""
atlas_status_readback=""
atlas_status_workgraph=""
atlas_status_sha=""
if [[ -z "$atlas_blueprint_import" || -z "$atlas_status" ]]; then
  add_check "atlas_blueprint_import" "blocked" "Atlas Blueprint import and Foundry Atlas status readback are required before low_risk_code live gates can run"
else
  atlas_blueprint_schema="$(jq -r '.contract_version // ""' "$atlas_blueprint_import")"
  atlas_blueprint_status="$(jq -r '.status // ""' "$atlas_blueprint_import")"
  atlas_blueprint_class="$(jq -r '.mutation_class // ""' "$atlas_blueprint_import")"
  atlas_blueprint_workgraph="$(jq -r '.workgraph_id // ""' "$atlas_blueprint_import")"
  atlas_blueprint_ready_for_foundry="$(jq -r 'if .ready_for_foundry == true then "true" else "false" end' "$atlas_blueprint_import")"
  atlas_blueprint_safe_to_execute="$(jq -r 'if .safe_to_execute == true then "true" else "false" end' "$atlas_blueprint_import")"
  atlas_blueprint_live_proven="$(jq -r 'if .live_execution_proven == true then "true" else "false" end' "$atlas_blueprint_import")"
  atlas_blueprint_downstream_digest="$(jq -r '.digests.downstream_foundry_import // ""' "$atlas_blueprint_import")"
  atlas_blueprint_pack_digest="$(jq -r '.digests.blueprint_pack // ""' "$atlas_blueprint_import")"
  atlas_blueprint_authorization_digest="$(jq -r '.digests.build_authorization // ""' "$atlas_blueprint_import")"
  atlas_blueprint_candidate_selection_digest="$(jq -r '.digests.candidate_selection // ""' "$atlas_blueprint_import")"
  atlas_blueprint_mutation_class_model_digest="$(jq -r '.digests.mutation_class_model // ""' "$atlas_blueprint_import")"
  atlas_blueprint_sha="$(sha256_file "$atlas_blueprint_import")"
  atlas_status_schema="$(jq -r '.schema_version // ""' "$atlas_status")"
  atlas_status_status="$(jq -r '.status // ""' "$atlas_status")"
  atlas_status_readback="$(jq -r '.readback_status // ""' "$atlas_status")"
  atlas_status_workgraph="$(jq -r '.workgraph_id // ""' "$atlas_status")"
  atlas_status_sha="$(sha256_file "$atlas_status")"
  if jq -e '
    .contract_version == "ao.atlas.blueprint-import.v0.1" and
    .status == "ready" and
    .mutation_class == "low_risk_code" and
    .ready_for_foundry == true and
    .safe_to_execute == false and
    .live_execution_proven == false and
    .schedules_work == false and
    .executes_work == false and
    .approves_work == false and
    .mutates_repositories == false and
    .calls_providers == false and
    .release_or_publish_allowed == false and
    (.digests.blueprint_pack // "") != "" and
    (.digests.build_authorization // "") != "" and
    (.digests.candidate_selection // "") != "" and
    (.digests.mutation_class_model // "") != "" and
    (.digests.downstream_foundry_import // "") != ""
  ' "$atlas_blueprint_import" >/dev/null &&
    jq -e --arg workgraph "$atlas_blueprint_workgraph" '
      .schema_version == "ao.foundry.atlas-status.v0.1" and
      .status == "ready" and
      .readback_status == "ready" and
      .workgraph_id == $workgraph and
      .schedules_work == false and
      .executes_work == false and
      .approves_work == false
    ' "$atlas_status" >/dev/null; then
    add_check "atlas_blueprint_import" "passed" "Atlas Blueprint import and Foundry Atlas readback are ready for the exact low_risk_code candidate"
  else
    add_check "atlas_blueprint_import" "blocked" "Atlas Blueprint import must be ready, low_risk_code scoped, Foundry-readback bound, and non-authoritative"
  fi
fi

for evidence_name in atlas_classification foundry_class_gate covenant_class_ticket forge_dry_run_plan ao2_dry_run_packet rollback_proof sentinel_hold promoter_ready command_readback test_only_success; do
  if jq -e --arg name "$evidence_name" '.source_artifacts[]? | select(.name == $name)' "$chain" >/dev/null; then
    add_check "evidence_$evidence_name" "passed" "$evidence_name is present in the dry-run chain"
  else
    add_check "evidence_$evidence_name" "blocked" "$evidence_name is missing from the dry-run chain"
  fi
done

if jq -e '
  .authority_boundaries.dry_run_only == true and
  .authority_boundaries.live_mutation_allowed == false and
  .authority_boundaries.mutates_repositories == false and
  .authority_boundaries.schedules_work == false and
  .authority_boundaries.executes_work == false and
  .authority_boundaries.approves_work == false and
  .authority_boundaries.provider_calls_allowed == false and
  .authority_boundaries.release_or_publish_allowed == false
' "$chain" >/dev/null; then
  add_check "dry_run_authority_boundaries" "passed" "input chain preserves dry-run non-authority boundaries"
else
  add_check "dry_run_authority_boundaries" "blocked" "input chain attempts live mutation, execution, approval, provider, or release authority"
fi

policy_schema=""
policy_status=""
policy_class=""
policy_chain_sha=""
policy_scope=""
policy_expires_at=""
policy_sha=""
policy_atlas_blueprint_sha=""
policy_blueprint_pack_digest=""
policy_blueprint_authorization_digest=""
policy_candidate_selection_digest=""
policy_mutation_class_model_digest=""
if [[ -z "$live_policy_evidence" ]]; then
  add_check "live_policy_evidence" "blocked" "explicit low_risk_code live policy evidence is required before a live code PR rehearsal can execute"
else
  policy_schema="$(jq -r '.schema_version // ""' "$live_policy_evidence")"
  policy_status="$(jq -r '.status // ""' "$live_policy_evidence")"
  policy_class="$(jq -r '.mutation_class // ""' "$live_policy_evidence")"
  policy_chain_sha="$(jq -r '.dry_run_chain_sha256 // ""' "$live_policy_evidence")"
  policy_scope="$(jq -r '.scope // ""' "$live_policy_evidence")"
  policy_expires_at="$(jq -r '.expires_at_utc // ""' "$live_policy_evidence")"
  policy_sha="$(sha256_file "$live_policy_evidence")"
  policy_atlas_blueprint_sha="$(jq -r '.atlas_blueprint_import_sha256 // ""' "$live_policy_evidence")"
  policy_blueprint_pack_digest="$(jq -r '.blueprint_pack_digest // ""' "$live_policy_evidence")"
  policy_blueprint_authorization_digest="$(jq -r '.blueprint_authorization_digest // ""' "$live_policy_evidence")"
  policy_candidate_selection_digest="$(jq -r '.candidate_selection_digest // ""' "$live_policy_evidence")"
  policy_mutation_class_model_digest="$(jq -r '.mutation_class_model_digest // ""' "$live_policy_evidence")"
  if jq -e \
    --arg chain_sha "$chain_sha" \
    --arg atlas_blueprint_sha "$atlas_blueprint_sha" \
    --arg blueprint_pack_digest "$atlas_blueprint_pack_digest" \
    --arg blueprint_authorization_digest "$atlas_blueprint_authorization_digest" \
    --arg candidate_selection_digest "$atlas_blueprint_candidate_selection_digest" \
    --arg mutation_class_model_digest "$atlas_blueprint_mutation_class_model_digest" \
    '
      .schema_version == "ao.foundry.low-risk-code-live-execution-policy.v0.1" and
      .status == "approved" and
      .mutation_class == "low_risk_code" and
      .scope == "single_source_plus_test" and
      .dry_run_chain_sha256 == $chain_sha and
      .atlas_blueprint_import_sha256 == $atlas_blueprint_sha and
      .blueprint_pack_digest == $blueprint_pack_digest and
      .blueprint_authorization_digest == $blueprint_authorization_digest and
      .candidate_selection_digest == $candidate_selection_digest and
      .mutation_class_model_digest == $mutation_class_model_digest and
      .safe_to_execute == false and
      (.live_execution_grants // false) == false and
      (.fully_unsupervised_complex_repo_live // false) == false and
      .candidate.repo == "ao-atlas" and
      .candidate.base_branch == "main" and
      .candidate.work_branch == "codex/low-risk-code-rehearsal-one" and
      .candidate.file_allowlist == ["internal/atlas/validate.go"] and
      .candidate.command_allowlist == ["git diff --check","go test ./..."] and
      .candidate.rollback_plan.strategy == "git_restore_exact_file" and
      .candidate.rollback_plan.files == ["internal/atlas/validate.go"]
    ' "$live_policy_evidence" >/dev/null; then
    add_check "live_policy_evidence" "passed" "live policy evidence is approved, Atlas-first digest-bound, class-bound, and exact-candidate scoped"
  else
    add_check "live_policy_evidence" "blocked" "live policy evidence must include Atlas Blueprint import digest, Blueprint pack digest, Blueprint authorization digest, candidate selection digest, mutation class model digest, and exact candidate scope"
  fi
fi

if [[ -n "$live_policy_evidence" && -n "$policy_expires_at" ]]; then
  if expires_epoch="$(date -u -j -f "%Y-%m-%dT%H:%M:%SZ" "$policy_expires_at" "+%s" 2>/dev/null)"; then
    :
  elif expires_epoch="$(date -u -d "$policy_expires_at" "+%s" 2>/dev/null)"; then
    :
  else
    expires_epoch="0"
  fi
  now_epoch="$(date -u "+%s")"
  if [[ "$expires_epoch" != "0" && "$expires_epoch" -gt "$now_epoch" ]]; then
    add_check "live_policy_expiry" "passed" "live policy evidence has not expired"
  else
    add_check "live_policy_expiry" "blocked" "live policy evidence is expired or has an invalid expiry"
  fi
fi

bounded_schema=""
bounded_status=""
bounded_sha=""
sentinel_schema=""
sentinel_status=""
sentinel_sha=""
promoter_schema=""
promoter_status=""
promoter_sha=""
command_schema=""
command_status=""
command_sha=""
if [[ -n "$bounded_packet_proof" ]]; then
  bounded_schema="$(jq -r '.schema_version // ""' "$bounded_packet_proof")"
  bounded_status="$(jq -r '.status // ""' "$bounded_packet_proof")"
  bounded_sha="$(sha256_file "$bounded_packet_proof")"
fi
if [[ -n "$sentinel_verdict" ]]; then
  sentinel_schema="$(jq -r '.schema_version // ""' "$sentinel_verdict")"
  sentinel_status="$(jq -r '.status // ""' "$sentinel_verdict")"
  sentinel_sha="$(sha256_file "$sentinel_verdict")"
fi
if [[ -n "$promoter_verdict" ]]; then
  promoter_schema="$(jq -r '.schema_version // ""' "$promoter_verdict")"
  promoter_status="$(jq -r '.status // ""' "$promoter_verdict")"
  promoter_sha="$(sha256_file "$promoter_verdict")"
fi
if [[ -n "$command_readback" ]]; then
  command_schema="$(jq -r '.schema_version // ""' "$command_readback")"
  command_status="$(jq -r '.status // ""' "$command_readback")"
  command_sha="$(sha256_file "$command_readback")"
fi

validate_downstream_evidence \
  "bounded_packet_enforcement" \
  "$bounded_packet_proof" \
  "ao.forge_ao2.low-risk-code-bounded-packet-enforcement.v0.1" \
  "ready" \
  "Forge/AO2 bounded packet proof is required before low_risk_code live execution can execute" \
  "Forge/AO2 bounded packet proof matches the exact held candidate" \
  "Forge/AO2 bounded packet proof must be exact-scope, chain-bound, and safe_to_execute=true"

validate_downstream_evidence \
  "sentinel_low_risk_live_verdict" \
  "$sentinel_verdict" \
  "ao.sentinel.low-risk-code-live-hold-verdict.v0.1" \
  "clear" \
  "Sentinel low_risk_code hold/clear verdict is required before low_risk_code live execution can execute" \
  "Sentinel low_risk_code hold/clear verdict clears the exact held candidate" \
  "Sentinel low_risk_code hold/clear verdict must clear the exact held candidate with safe_to_execute=true"

validate_downstream_evidence \
  "promoter_low_risk_live_verdict" \
  "$promoter_verdict" \
  "ao.promoter.low-risk-code-live-verdict.v0.1" \
  "no_promotion_before_execution" \
  "Promoter low_risk_code no-promotion verdict is required before low_risk_code live execution can execute" \
  "Promoter low_risk_code verdict is limited to no promotion before execution" \
  "Promoter low_risk_code verdict must be exact-scope, chain-bound, and deny broader promotion"

validate_downstream_evidence \
  "command_live_readback" \
  "$command_readback" \
  "ao.command.low-risk-code-live-readback.v0.1" \
  "ready" \
  "AO Command low_risk_code readback is required before low_risk_code live execution can execute" \
  "AO Command low_risk_code readback matches Atlas import status and policy evidence status for the exact held command set" \
  "AO Command low_risk_code readback must match the exact held candidate and command allowlist"

status="ready"
safe_to_execute="true"
exact_next_step="request_low_risk_code_live_rehearsal"
allowed_next_action="request_low_risk_code_live_rehearsal"
if [[ -n "$first_failing_check" ]]; then
  status="blocked"
  safe_to_execute="false"
  case "$first_failing_check" in
    atlas_blueprint_import)
      exact_next_step="collect_atlas_blueprint_import_readback"
      allowed_next_action="collect_atlas_blueprint_import_readback"
      ;;
    live_policy_evidence|live_policy_expiry)
      exact_next_step="collect_low_risk_code_live_policy_evidence"
      allowed_next_action="collect_low_risk_code_live_policy_evidence"
      ;;
    bounded_packet_enforcement)
      exact_next_step="collect_forge_ao2_bounded_packet_enforcement_proof"
      allowed_next_action="collect_forge_ao2_bounded_packet_enforcement_proof"
      ;;
    sentinel_low_risk_live_verdict)
      exact_next_step="request_sentinel_low_risk_live_hold_verdict"
      allowed_next_action="request_sentinel_low_risk_live_hold_verdict"
      ;;
    promoter_low_risk_live_verdict)
      exact_next_step="request_promoter_low_risk_live_verdict"
      allowed_next_action="request_promoter_low_risk_live_verdict"
      ;;
    command_live_readback)
      exact_next_step="collect_command_low_risk_live_readback"
      allowed_next_action="collect_command_low_risk_live_readback"
      ;;
    *)
      exact_next_step="repair_low_risk_code_live_rehearsal_gate"
      allowed_next_action="repair_low_risk_code_live_rehearsal_gate"
      ;;
  esac
fi

mkdir -p "$(dirname "$out")"
checks_json="$tmpdir/checks.json"
jq -s '.' "$checks_file" > "$checks_json"

jq -n \
  --arg schema_version "ao.foundry.low-risk-code-live-rehearsal-gate.v0.1" \
  --arg status "$status" \
  --arg mutation_class "low_risk_code" \
  --argjson safe_to_execute "$safe_to_execute" \
  --arg exact_next_step "$exact_next_step" \
  --arg allowed_next_action "$allowed_next_action" \
  --arg first_failing_check "$first_failing_check" \
  --arg chain_path "$chain" \
  --arg chain_sha256 "$chain_sha" \
  --arg atlas_blueprint_path "$atlas_blueprint_import" \
  --arg atlas_blueprint_schema "$atlas_blueprint_schema" \
  --arg atlas_blueprint_status "$atlas_blueprint_status" \
  --arg atlas_blueprint_sha256 "$atlas_blueprint_sha" \
  --arg atlas_blueprint_workgraph "$atlas_blueprint_workgraph" \
  --arg atlas_blueprint_pack_digest "$atlas_blueprint_pack_digest" \
  --arg atlas_blueprint_authorization_digest "$atlas_blueprint_authorization_digest" \
  --arg atlas_blueprint_candidate_selection_digest "$atlas_blueprint_candidate_selection_digest" \
  --arg atlas_blueprint_mutation_class_model_digest "$atlas_blueprint_mutation_class_model_digest" \
  --arg atlas_status_path "$atlas_status" \
  --arg atlas_status_schema "$atlas_status_schema" \
  --arg atlas_status_status "$atlas_status_status" \
  --arg atlas_status_readback "$atlas_status_readback" \
  --arg atlas_status_workgraph "$atlas_status_workgraph" \
  --arg atlas_status_sha256 "$atlas_status_sha" \
  --arg policy_path "$live_policy_evidence" \
  --arg policy_schema "$policy_schema" \
  --arg policy_status "$policy_status" \
  --arg policy_sha256 "$policy_sha" \
  --arg policy_atlas_blueprint_sha "$policy_atlas_blueprint_sha" \
  --arg policy_blueprint_pack_digest "$policy_blueprint_pack_digest" \
  --arg policy_blueprint_authorization_digest "$policy_blueprint_authorization_digest" \
  --arg policy_candidate_selection_digest "$policy_candidate_selection_digest" \
  --arg policy_mutation_class_model_digest "$policy_mutation_class_model_digest" \
  --arg bounded_path "$bounded_packet_proof" \
  --arg bounded_schema "$bounded_schema" \
  --arg bounded_status "$bounded_status" \
  --arg bounded_sha256 "$bounded_sha" \
  --arg sentinel_path "$sentinel_verdict" \
  --arg sentinel_schema "$sentinel_schema" \
  --arg sentinel_status "$sentinel_status" \
  --arg sentinel_sha256 "$sentinel_sha" \
  --arg promoter_path "$promoter_verdict" \
  --arg promoter_schema "$promoter_schema" \
  --arg promoter_status "$promoter_status" \
  --arg promoter_sha256 "$promoter_sha" \
  --arg command_path "$command_readback" \
  --arg command_schema "$command_schema" \
  --arg command_status "$command_status" \
  --arg command_sha256 "$command_sha" \
  --slurpfile checks "$checks_json" \
  '{
    schema_version:$schema_version,
    status:$status,
    mutation_class:$mutation_class,
    safe_to_request:true,
    safe_to_execute:$safe_to_execute,
    exact_next_step:$exact_next_step,
    allowed_next_action:$allowed_next_action,
    first_failing_check:$first_failing_check,
    checks:$checks[0],
    source_hashes:([
      {name:"low_risk_code_dry_run_chain", path:$chain_path, schema_version:"ao.foundry.governed-live-mutation-dry-run-chain.v0.1", sha256:$chain_sha256}
    ] + (if $atlas_blueprint_path == "" then [] else [
      {name:"atlas_blueprint_import", path:$atlas_blueprint_path, schema_version:$atlas_blueprint_schema, status:$atlas_blueprint_status, workgraph_id:$atlas_blueprint_workgraph, sha256:$atlas_blueprint_sha256, blueprint_pack_digest:$atlas_blueprint_pack_digest, blueprint_authorization_digest:$atlas_blueprint_authorization_digest, candidate_selection_digest:$atlas_blueprint_candidate_selection_digest, mutation_class_model_digest:$atlas_blueprint_mutation_class_model_digest}
    ] end) + (if $atlas_status_path == "" then [] else [
      {name:"atlas_status", path:$atlas_status_path, schema_version:$atlas_status_schema, status:$atlas_status_status, readback_status:$atlas_status_readback, workgraph_id:$atlas_status_workgraph, sha256:$atlas_status_sha256}
    ] end) + (if $policy_path == "" then [] else [
      {name:"live_policy_evidence", path:$policy_path, schema_version:$policy_schema, status:$policy_status, sha256:$policy_sha256, atlas_blueprint_import_sha256:$policy_atlas_blueprint_sha, blueprint_pack_digest:$policy_blueprint_pack_digest, blueprint_authorization_digest:$policy_blueprint_authorization_digest, candidate_selection_digest:$policy_candidate_selection_digest, mutation_class_model_digest:$policy_mutation_class_model_digest}
    ] end) + (if $bounded_path == "" then [] else [
      {name:"bounded_packet_enforcement", path:$bounded_path, schema_version:$bounded_schema, status:$bounded_status, sha256:$bounded_sha256}
    ] end) + (if $sentinel_path == "" then [] else [
      {name:"sentinel_low_risk_live_verdict", path:$sentinel_path, schema_version:$sentinel_schema, status:$sentinel_status, sha256:$sentinel_sha256}
    ] end) + (if $promoter_path == "" then [] else [
      {name:"promoter_low_risk_live_verdict", path:$promoter_path, schema_version:$promoter_schema, status:$promoter_status, sha256:$promoter_sha256}
    ] end) + (if $command_path == "" then [] else [
      {name:"command_live_readback", path:$command_path, schema_version:$command_schema, status:$command_status, sha256:$command_sha256}
    ] end)),
    atlas_blueprint_import_status:{
      present:($atlas_blueprint_path != ""),
      status:$atlas_blueprint_status,
      schema_version:$atlas_blueprint_schema,
      sha256:$atlas_blueprint_sha256,
      workgraph_id:$atlas_blueprint_workgraph
    },
    policy_evidence_status:{
      present:($policy_path != ""),
      status:$policy_status,
      schema_version:$policy_schema,
      sha256:$policy_sha256,
      atlas_blueprint_import_sha256:$policy_atlas_blueprint_sha,
      blueprint_pack_digest:$policy_blueprint_pack_digest,
      blueprint_authorization_digest:$policy_blueprint_authorization_digest,
      candidate_selection_digest:$policy_candidate_selection_digest,
      mutation_class_model_digest:$policy_mutation_class_model_digest
    },
    rewired_downstream_evidence:{
      bounded_packet_enforcement:($bounded_path != ""),
      sentinel_low_risk_live_verdict:($sentinel_path != ""),
      promoter_low_risk_live_verdict:($promoter_path != ""),
      command_live_readback:($command_path != "")
    },
    blocking_next_actions:(if $status == "ready" then [] else [$exact_next_step] end),
    denial_reason:(if $status == "ready" then "" else "low_risk_code live execution remains denied until exact live policy evidence is approved and digest-bound to the dry-run chain." end),
    maintenance_suggestions:[
      "Keep low_risk_code live rehearsal bounded to one source file plus one test file.",
      "Do not treat this gate as permission for multi-repo, complex, or unsupervised mutation."
    ],
    authority_boundaries:{
      emits_decision_only:true,
      mutation_class:"low_risk_code",
      max_source_files:1,
      max_test_files:1,
      broad_live_mutation_allowed:false,
      multi_repo_mutation_allowed:false,
      complex_repo_mutation_allowed:false,
      fully_unsupervised_complex_mutation_claimed:false,
      mutates_repositories:false,
      creates_branch:false,
      creates_worktree:false,
      opens_pr:false,
      merges_pr:false,
      schedules_work:false,
      executes_work:false,
      approves_work:false,
      provider_calls_allowed:false,
      release_or_publish_allowed:false
    }
  }' > "$out"

if [[ "$json" == "1" ]]; then
  cat "$out"
else
  echo "low_risk_code_live_rehearsal_gate=$status"
  echo "safe_to_execute=$safe_to_execute"
  echo "exact_next_step=$exact_next_step"
  echo "gate=$out"
fi
