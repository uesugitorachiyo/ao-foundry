#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
usage: scripts/live-docs-approval-gate.sh --request <request.json> --ticket <ticket.json> --out <gate.json>

Converts a first live docs-only approval request into safe_to_execute=true only
when the Covenant ticket is approved, unexpired, unconsumed, and exact-scope.
This script emits evidence only; it does not mutate repositories.
USAGE
}

request=""
ticket=""
out=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --request) request="${2:-}"; shift 2 ;;
    --ticket) ticket="${2:-}"; shift 2 ;;
    --out) out="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "live-docs-approval-gate: unknown argument $1" >&2; usage; exit 2 ;;
  esac
done

if [[ -z "$request" || -z "$ticket" || -z "$out" ]]; then
  usage
  exit 2
fi
if [[ ! -f "$request" ]]; then
  echo "live-docs-approval-gate: request not found: $request" >&2
  exit 1
fi
if [[ ! -f "$ticket" ]]; then
  echo "live-docs-approval-gate: ticket not found: $ticket" >&2
  exit 1
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

json_string() {
  jq -Rsa .
}

sha256_file() {
  shasum -a 256 "$1" | awk '{print $1}'
}

display_path() {
  local path="$1"
  local abs
  abs="$(cd "$(dirname "$path")" && pwd)/$(basename "$path")"
  if [[ "$abs" == "$repo_root/"* ]]; then
    printf '%s' "${abs#"$repo_root/"}"
  else
    printf '%s' "$path"
  fi
}

scope_json() {
  jq -S '{repo, branch_policy, docs_only_path_allowlist, forbidden_paths, max_changed_files}' "$1"
}

ticket_scope_json() {
  jq -S '.approved_scope' "$1"
}

request_schema="$(jq -r '.schema_version // ""' "$request")"
ticket_schema="$(jq -r '.schema_version // ""' "$ticket")"
request_id="$(jq -r '.request_id // ""' "$request")"
ticket_request_id="$(jq -r '.request_id // ""' "$ticket")"
approval_state="$(jq -r '.approval_state // ""' "$ticket")"
expires_at="$(jq -r '.expires_at // ""' "$ticket")"
consumed="$(jq -r 'if .consumed == false then "false" else "true" end' "$ticket")"
approver="$(jq -r '.approver_identity // ""' "$ticket")"
required_ticket_schema="$(jq -r '.foundry_required_ticket_schema // ""' "$ticket")"
request_safe_to_request="$(jq -r 'if .safe_to_request == true then "true" else "false" end' "$request")"
request_safe_to_execute="$(jq -r 'if .safe_to_execute == false then "false" else "true" end' "$request")"

status="ready"
safe_to_execute="true"
first_failing_check=""
allowed_next_action="start_first_docs_only_live_pr_rehearsal"

block() {
  if [[ "$status" == "ready" ]]; then
    status="blocked"
    safe_to_execute="false"
    first_failing_check="$1"
    allowed_next_action="$2"
  fi
}

if [[ "$request_schema" != "ao.foundry.live-mutation-approval-request.v0.1" ]]; then
  block "request_schema" "refresh_foundry_approval_request"
fi
if [[ "$ticket_schema" != "covenant.live-docs-approval-ticket.v1" ]]; then
  block "ticket_schema" "request_covenant_live_docs_approval_ticket"
fi
if [[ "$required_ticket_schema" != "ao.covenant.live-docs-approval-ticket.v0.1" ]]; then
  block "ticket_schema_compatibility" "request_covenant_live_docs_approval_ticket"
fi
if [[ "$request_id" == "" || "$ticket_request_id" != "$request_id" ]]; then
  block "request_id" "request_exact_scope_approval_ticket"
fi
if [[ "$request_safe_to_request" != "true" || "$request_safe_to_execute" != "false" ]]; then
  block "request_boundary" "refresh_foundry_approval_request"
fi
if [[ "$approval_state" != "approved" ]]; then
  block "approval_state" "request_operator_approval"
fi
if [[ "$approver" == "" ]]; then
  block "approver_identity" "request_operator_approval"
fi
if [[ "$consumed" != "false" ]]; then
  block "ticket_consumed" "request_new_operator_approval"
fi
if ! expires_epoch="$(date -u -j -f "%Y-%m-%dT%H:%M:%SZ" "$expires_at" "+%s" 2>/dev/null)"; then
  if ! expires_epoch="$(date -u -d "$expires_at" "+%s" 2>/dev/null)"; then
    block "ticket_expiry" "request_new_operator_approval"
    expires_epoch="0"
  fi
fi
now_epoch="$(date -u "+%s")"
if [[ "$expires_epoch" != "0" && "$expires_epoch" -le "$now_epoch" ]]; then
  block "ticket_expired" "request_new_operator_approval"
fi
if [[ "$(scope_json "$request")" != "$(ticket_scope_json "$ticket")" ]]; then
  block "scope_mismatch" "request_exact_scope_approval_ticket"
fi

mkdir -p "$(dirname "$out")"
{
  printf '{\n'
  printf '  "schema_version": "ao.foundry.live-docs-approval-gate.v0.1",\n'
  printf '  "status": %s,\n' "$(printf '%s' "$status" | json_string)"
  printf '  "safe_to_request": true,\n'
  printf '  "safe_to_execute": %s,\n' "$safe_to_execute"
  printf '  "approval_state": %s,\n' "$(printf '%s' "$approval_state" | json_string)"
  printf '  "request_id": %s,\n' "$(printf '%s' "$request_id" | json_string)"
  printf '  "ticket_id": %s,\n' "$(jq '.ticket_id // ""' "$ticket")"
  printf '  "first_failing_check": %s,\n' "$(printf '%s' "$first_failing_check" | json_string)"
  printf '  "allowed_next_action": %s,\n' "$(printf '%s' "$allowed_next_action" | json_string)"
  printf '  "source_hashes": [\n'
  printf '    {"name":"approval_request","path":%s,"schema_version":%s,"sha256":"%s"},\n' "$(display_path "$request" | json_string)" "$(printf '%s' "$request_schema" | json_string)" "$(sha256_file "$request")"
  printf '    {"name":"approval_ticket","path":%s,"schema_version":%s,"sha256":"%s"}\n' "$(display_path "$ticket" | json_string)" "$(printf '%s' "$ticket_schema" | json_string)" "$(sha256_file "$ticket")"
  printf '  ],\n'
  printf '  "authority_boundaries": {\n'
  printf '    "mutates_repositories": false,\n'
  printf '    "approves_work": false,\n'
  printf '    "executes_work": false,\n'
  printf '    "calls_providers": false,\n'
  printf '    "release_or_publish_allowed": false\n'
  printf '  }\n'
  printf '}\n'
} > "$out"

echo "live_docs_approval_gate=$status"
echo "safe_to_execute=$safe_to_execute"
echo "gate=$out"
