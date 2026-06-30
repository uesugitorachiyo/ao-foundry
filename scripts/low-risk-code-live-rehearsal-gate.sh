#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
usage: scripts/low-risk-code-live-rehearsal-gate.sh --chain <summary.json> --out <gate.json> [--live-policy-evidence <policy.json>] [--json]

Decides whether a first low_risk_code live rehearsal may start. This gate emits
evidence only. It never creates branches, creates worktrees, mutates
repositories, opens PRs, merges, publishes, uploads, releases, approves work,
or calls providers.
USAGE
}

chain=""
live_policy_evidence=""
out=""
json=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --chain) chain="${2:-}"; shift 2 ;;
    --live-policy-evidence) live_policy_evidence="${2:-}"; shift 2 ;;
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
if [[ "$out" == "$chain" || "$out" == "$live_policy_evidence" ]]; then
  echo "low-risk-code-live-rehearsal-gate: --out must not overwrite input artifacts" >&2
  exit 2
fi
path_bundle="$chain:$live_policy_evidence:$out"
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
if [[ -n "$live_policy_evidence" && ! -f "$live_policy_evidence" ]]; then
  echo "low-risk-code-live-rehearsal-gate: live policy evidence not found: $live_policy_evidence" >&2
  exit 2
fi
if ! command -v jq >/dev/null 2>&1; then
  echo "low-risk-code-live-rehearsal-gate: jq is required" >&2
  exit 2
fi

sha256_file() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
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
  if [[ "$policy_schema" == "ao.foundry.low-risk-code-live-execution-policy.v0.1" && "$policy_status" == "approved" && "$policy_class" == "low_risk_code" && "$policy_scope" == "single_source_plus_test" && "$policy_chain_sha" == "$chain_sha" ]]; then
    add_check "live_policy_evidence" "passed" "live policy evidence is approved, class-bound, exact-scope, and dry-run-chain digest-bound"
  else
    add_check "live_policy_evidence" "blocked" "live policy evidence must be approved, class-bound, exact-scope, and digest-bound to this dry-run chain"
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

status="ready"
safe_to_execute="true"
exact_next_step="request_low_risk_code_live_rehearsal"
allowed_next_action="request_low_risk_code_live_rehearsal"
if [[ -n "$first_failing_check" ]]; then
  status="blocked"
  safe_to_execute="false"
  if [[ "$first_failing_check" == "live_policy_evidence" ]]; then
    exact_next_step="collect_low_risk_code_live_policy_evidence"
    allowed_next_action="collect_low_risk_code_live_policy_evidence"
  else
    exact_next_step="repair_low_risk_code_live_rehearsal_gate"
    allowed_next_action="repair_low_risk_code_live_rehearsal_gate"
  fi
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
  --arg policy_path "$live_policy_evidence" \
  --arg policy_schema "$policy_schema" \
  --arg policy_status "$policy_status" \
  --arg policy_sha256 "$policy_sha" \
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
    ] + (if $policy_path == "" then [] else [
      {name:"live_policy_evidence", path:$policy_path, schema_version:$policy_schema, status:$policy_status, sha256:$policy_sha256}
    ] end)),
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
