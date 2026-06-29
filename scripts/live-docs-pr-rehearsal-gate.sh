#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
usage: scripts/live-docs-pr-rehearsal-gate.sh --chain <summary.json> --out <gate.json> [--approval-artifact <ticket.json>] [--json]

Decides whether the first tightly scoped docs-only live branch/PR rehearsal may
start. This gate emits evidence only. It never creates branches, creates
worktrees, mutates repositories, opens PRs, merges, publishes, uploads, releases,
approves work, or calls providers.
USAGE
}

chain=""
approval_artifact=""
out=""
json=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --chain) chain="${2:-}"; shift 2 ;;
    --approval-artifact) approval_artifact="${2:-}"; shift 2 ;;
    --out) out="${2:-}"; shift 2 ;;
    --json) json=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "live-docs-pr-rehearsal-gate: unknown argument $1" >&2; usage >&2; exit 2 ;;
  esac
done

if [[ -z "$chain" || -z "$out" ]]; then
  usage >&2
  exit 2
fi
if [[ "$out" == "$chain" || "$out" == "$approval_artifact" ]]; then
  echo "live-docs-pr-rehearsal-gate: --out must not overwrite input artifacts" >&2
  exit 2
fi
path_bundle="$chain:$approval_artifact:$out"
private_mac_root="/""Users/"
private_linux_root="/""home/"
private_tmp_root="/""tmp/"
for unsafe_marker in "/.." "../" "~" "$private_mac_root" "$private_linux_root" "$private_tmp_root"; do
  if [[ "$path_bundle" == *"$unsafe_marker"* ]]; then
    echo "live-docs-pr-rehearsal-gate: paths must be public-safe relative paths" >&2
    exit 2
  fi
done
if [[ ! -f "$chain" ]]; then
  echo "live-docs-pr-rehearsal-gate: chain not found: $chain" >&2
  exit 2
fi
if [[ -n "$approval_artifact" && ! -f "$approval_artifact" ]]; then
  echo "live-docs-pr-rehearsal-gate: approval artifact not found: $approval_artifact" >&2
  exit 2
fi
if ! command -v jq >/dev/null 2>&1; then
  echo "live-docs-pr-rehearsal-gate: jq is required" >&2
  exit 2
fi

sha256_file() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    echo "live-docs-pr-rehearsal-gate: shasum or sha256sum is required" >&2
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

chain_schema="$(jq -r '.schema_version // ""' "$chain")"
chain_status="$(jq -r '.status // ""' "$chain")"
chain_safe_to_execute="$(jq -r 'if .readiness_assessment.safe_to_execute == false then "false" else "true" end' "$chain")"
chain_next_step="$(jq -r '.readiness_assessment.exact_next_step // ""' "$chain")"
chain_ticket_sha="$(jq -r '.source_artifacts[]? | select(.name == "approval_ticket") | .sha256' "$chain" | head -1)"

if [[ "$chain_schema" == "ao.foundry.approved-live-docs-dry-run-chain.v0.1" && "$chain_status" == "ready" && "$chain_safe_to_execute" == "false" && "$chain_next_step" == "run_live_docs_pr_rehearsal_gate" ]]; then
  add_check "approved_dry_run_chain" "passed" "approved docs-only dry-run chain is ready and still non-executing"
else
  add_check "approved_dry_run_chain" "blocked" "approved docs-only dry-run chain must be ready before any PR rehearsal may start"
fi

approval_schema=""
approval_state=""
approval_consumed="true"
approval_approver=""
approval_expires_at=""
approval_ticket_sha=""
if [[ -z "$approval_artifact" ]]; then
  add_check "approval_artifact" "blocked" "explicit approval artifact is required before the first docs-only PR rehearsal can execute"
else
  approval_schema="$(jq -r '.schema_version // ""' "$approval_artifact")"
  approval_state="$(jq -r '.approval_state // ""' "$approval_artifact")"
  approval_consumed="$(jq -r 'if .consumed == false then "false" else "true" end' "$approval_artifact")"
  approval_approver="$(jq -r '.approver_identity // ""' "$approval_artifact")"
  approval_expires_at="$(jq -r '.expires_at // ""' "$approval_artifact")"
  approval_ticket_sha="$(sha256_file "$approval_artifact")"
  if [[ "$approval_schema" == "covenant.live-docs-approval-ticket.v1" && "$approval_state" == "approved" && "$approval_consumed" == "false" && -n "$approval_approver" ]]; then
    add_check "approval_artifact" "passed" "explicit approval artifact is approved, unconsumed, and has an approver identity"
  else
    add_check "approval_artifact" "blocked" "approval artifact must be an approved, unconsumed Covenant live docs approval ticket"
  fi
fi

if [[ -n "$approval_artifact" ]]; then
  if [[ -n "$chain_ticket_sha" && "$approval_ticket_sha" == "$chain_ticket_sha" ]]; then
    add_check "approval_digest_binding" "passed" "approval artifact digest matches the approval ticket bound into the dry-run chain"
  else
    add_check "approval_digest_binding" "blocked" "approval artifact digest must match the approval ticket bound into the dry-run chain"
  fi
fi

if [[ -n "$approval_artifact" && -n "$approval_expires_at" ]]; then
  if expires_epoch="$(date -u -j -f "%Y-%m-%dT%H:%M:%SZ" "$approval_expires_at" "+%s" 2>/dev/null)"; then
    :
  elif expires_epoch="$(date -u -d "$approval_expires_at" "+%s" 2>/dev/null)"; then
    :
  else
    expires_epoch="0"
  fi
  now_epoch="$(date -u "+%s")"
  if [[ "$expires_epoch" != "0" && "$expires_epoch" -gt "$now_epoch" ]]; then
    add_check "approval_expiry" "passed" "approval artifact has not expired"
  else
    add_check "approval_expiry" "blocked" "approval artifact is expired or has an invalid expiry"
  fi
fi

if jq -e '
  .authority_boundaries.dry_run_only == true and
  .authority_boundaries.live_mutation_allowed == false and
  .authority_boundaries.mutates_repositories == false and
  .authority_boundaries.creates_branch == false and
  .authority_boundaries.creates_worktree == false and
  .authority_boundaries.schedules_work == false and
  .authority_boundaries.executes_work == false and
  .authority_boundaries.approves_work == false and
  .authority_boundaries.provider_calls_allowed == false and
  .authority_boundaries.release_or_publish_allowed == false and
  .authority_boundaries.broad_live_mutation_allowed == false and
  .authority_boundaries.fully_unsupervised_complex_mutation_claimed == false
' "$chain" >/dev/null; then
  add_check "authority_boundaries" "passed" "input chain preserves dry-run non-authority boundaries"
else
  add_check "authority_boundaries" "blocked" "input chain attempts mutation, branch creation, execution, approval, provider, release, or broad authority"
fi

status="ready"
safe_to_execute="true"
exact_next_step="start_first_docs_only_live_pr_rehearsal"
allowed_next_action="start_first_docs_only_live_pr_rehearsal"
if [[ -n "$first_failing_check" ]]; then
  status="blocked"
  safe_to_execute="false"
  if [[ "$first_failing_check" == "approval_artifact" ]]; then
    exact_next_step="request_operator_approval"
    allowed_next_action="request_operator_approval"
  else
    exact_next_step="repair_live_docs_pr_rehearsal_gate"
    allowed_next_action="repair_live_docs_pr_rehearsal_gate"
  fi
fi

mkdir -p "$(dirname "$out")"
checks_json="$tmpdir/checks.json"
jq -s '.' "$checks_file" > "$checks_json"

jq -n \
  --arg schema_version "ao.foundry.live-docs-pr-rehearsal-gate.v0.1" \
  --arg status "$status" \
  --argjson safe_to_execute "$safe_to_execute" \
  --arg first_live_class "docs_only" \
  --arg exact_next_step "$exact_next_step" \
  --arg allowed_next_action "$allowed_next_action" \
  --arg first_failing_check "$first_failing_check" \
  --arg chain_path "$chain" \
  --arg chain_sha256 "$(sha256_file "$chain")" \
  --arg approval_path "$approval_artifact" \
  --arg approval_schema "$approval_schema" \
  --arg approval_state "$approval_state" \
  --arg approval_sha256 "$approval_ticket_sha" \
  --slurpfile checks "$checks_json" \
  '{
    schema_version:$schema_version,
    status:$status,
    first_live_class:$first_live_class,
    safe_to_request:true,
    safe_to_execute:$safe_to_execute,
    exact_next_step:$exact_next_step,
    allowed_next_action:$allowed_next_action,
    first_failing_check:$first_failing_check,
    checks:$checks[0],
    source_hashes:([
      {name:"approved_live_docs_dry_run_chain", path:$chain_path, schema_version:"ao.foundry.approved-live-docs-dry-run-chain.v0.1", sha256:$chain_sha256}
    ] + (if $approval_path == "" then [] else [
      {name:"approval_artifact", path:$approval_path, schema_version:$approval_schema, status:$approval_state, sha256:$approval_sha256}
    ] end)),
    blocking_next_actions:(if $status == "ready" then [] else [$exact_next_step] end),
    maintenance_suggestions:[
      "Keep the first live branch/PR rehearsal docs-only and exact-scope approved.",
      "Do not treat this gate as permission for broad or unsupervised live mutation."
    ],
    authority_boundaries:{
      emits_decision_only:true,
      first_live_class:"docs_only",
      broad_live_mutation_allowed:false,
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
  echo "live_docs_pr_rehearsal_gate=$status"
  echo "safe_to_execute=$safe_to_execute"
  echo "exact_next_step=$exact_next_step"
  echo "gate=$out"
fi
