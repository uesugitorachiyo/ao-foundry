#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
usage: scripts/first-live-docs-readiness-rollup.sh --chain <summary.json> --pr-gate <gate.json> --out <rollup.json>

Summarizes the first tiny docs-only live-mutation approval path:
request/ticket/gate/guard/patch/worktree/rollback/Sentinel/Promoter/Command
evidence plus the final PR rehearsal gate decision. This is evidence readback
only; it never creates branches, opens PRs, mutates repositories, calls
providers, publishes, uploads, tags, or releases.
USAGE
}

chain=""
pr_gate=""
out=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --chain) chain="${2:-}"; shift 2 ;;
    --pr-gate) pr_gate="${2:-}"; shift 2 ;;
    --out) out="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "first-live-docs-readiness-rollup: unknown argument $1" >&2; usage >&2; exit 2 ;;
  esac
done

if [[ -z "$chain" || -z "$pr_gate" || -z "$out" ]]; then
  usage >&2
  exit 2
fi
if [[ "$out" == "$chain" || "$out" == "$pr_gate" ]]; then
  echo "first-live-docs-readiness-rollup: --out must not overwrite input artifacts" >&2
  exit 2
fi
path_bundle="$chain:$pr_gate:$out"
private_mac_root="/""Users/"
private_linux_root="/""home/"
private_tmp_root="/""tmp/"
for unsafe_marker in "/.." "../" "~" "$private_mac_root" "$private_linux_root" "$private_tmp_root"; do
  if [[ "$path_bundle" == *"$unsafe_marker"* ]]; then
    echo "first-live-docs-readiness-rollup: paths must be public-safe relative paths" >&2
    exit 2
  fi
done
if [[ ! -f "$chain" ]]; then
  echo "first-live-docs-readiness-rollup: chain not found: $chain" >&2
  exit 2
fi
if [[ ! -f "$pr_gate" ]]; then
  echo "first-live-docs-readiness-rollup: PR gate not found: $pr_gate" >&2
  exit 2
fi
if ! command -v jq >/dev/null 2>&1; then
  echo "first-live-docs-readiness-rollup: jq is required" >&2
  exit 2
fi

sha256_file() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    echo "first-live-docs-readiness-rollup: shasum or sha256sum is required" >&2
    exit 2
  fi
}

chain_schema="$(jq -r '.schema_version // ""' "$chain")"
chain_status="$(jq -r '.status // ""' "$chain")"
chain_class="$(jq -r '.first_live_class // ""' "$chain")"
chain_safe_to_request="$(jq -r 'if .readiness_assessment.safe_to_request == true then "true" else "false" end' "$chain")"
chain_safe_to_execute="$(jq -r 'if .readiness_assessment.safe_to_execute == false then "false" else "true" end' "$chain")"
gate_schema="$(jq -r '.schema_version // ""' "$pr_gate")"
gate_status="$(jq -r '.status // ""' "$pr_gate")"
gate_safe_to_execute="$(jq -r 'if .safe_to_execute == true then "true" else "false" end' "$pr_gate")"
gate_next_step="$(jq -r '.exact_next_step // ""' "$pr_gate")"
gate_first_failing_check="$(jq -r '.first_failing_check // ""' "$pr_gate")"

status="ready"
first_failing_check=""
if [[ "$chain_schema" != "ao.foundry.approved-live-docs-dry-run-chain.v0.1" || "$chain_status" != "ready" || "$chain_class" != "docs_only" || "$chain_safe_to_request" != "true" || "$chain_safe_to_execute" != "false" ]]; then
  status="blocked"
  first_failing_check="approved_live_docs_dry_run_chain"
fi
if [[ "$gate_schema" != "ao.foundry.live-docs-pr-rehearsal-gate.v0.1" ]]; then
  status="blocked"
  if [[ -z "$first_failing_check" ]]; then first_failing_check="live_docs_pr_rehearsal_gate"; fi
fi
if [[ "$gate_status" == "blocked" && -z "$first_failing_check" ]]; then
  status="blocked"
  first_failing_check="$(jq -r '.first_failing_check // "live_docs_pr_rehearsal_gate"' "$pr_gate")"
fi
if [[ "$gate_status" == "ready" && "$gate_safe_to_execute" != "true" && -z "$first_failing_check" ]]; then
  status="blocked"
  first_failing_check="safe_to_execute"
fi

safe_to_request="false"
if [[ "$chain_safe_to_request" == "true" && "$chain_status" == "ready" ]]; then
  safe_to_request="true"
fi
safe_to_execute="false"
if [[ "$status" == "ready" && "$gate_safe_to_execute" == "true" ]]; then
  safe_to_execute="true"
fi
exact_next_step="$gate_next_step"
if [[ -z "$exact_next_step" ]]; then
  exact_next_step="request_operator_approval"
fi
if [[ "$safe_to_execute" != "true" && "$exact_next_step" != "request_operator_approval" ]]; then
  exact_next_step="repair_first_live_docs_readiness"
fi

mkdir -p "$(dirname "$out")"
jq -n \
  --arg schema_version "ao.foundry.first-live-docs-readiness-rollup.v0.1" \
  --arg status "$status" \
  --arg first_live_class "docs_only" \
  --argjson safe_to_request "$safe_to_request" \
  --argjson safe_to_execute "$safe_to_execute" \
  --arg approved_scope "docs_only" \
  --arg exact_next_step "$exact_next_step" \
  --arg first_failing_check "$first_failing_check" \
  --arg chain_path "$chain" \
  --arg chain_sha256 "$(sha256_file "$chain")" \
  --arg pr_gate_path "$pr_gate" \
  --arg pr_gate_sha256 "$(sha256_file "$pr_gate")" \
  --slurpfile chain_doc "$chain" \
  --slurpfile gate_doc "$pr_gate" \
  '{
    schema_version:$schema_version,
    status:$status,
    first_live_class:$first_live_class,
    approved_scope:$approved_scope,
    safe_to_request:$safe_to_request,
    safe_to_execute:$safe_to_execute,
    explicit_operator_approval_required:($safe_to_execute != true),
    exact_next_step:$exact_next_step,
    first_failing_check:$first_failing_check,
    evidence:{
      approved_live_docs_dry_run_chain:{path:$chain_path,schema_version:"ao.foundry.approved-live-docs-dry-run-chain.v0.1",sha256:$chain_sha256,status:$chain_doc[0].status},
      live_docs_pr_rehearsal_gate:{path:$pr_gate_path,schema_version:"ao.foundry.live-docs-pr-rehearsal-gate.v0.1",sha256:$pr_gate_sha256,status:$gate_doc[0].status},
      source_artifacts:($chain_doc[0].source_artifacts + $gate_doc[0].source_hashes)
    },
    blocking_next_actions:(if $status == "ready" then [] else [$exact_next_step] end),
    maintenance_suggestions:[
      "Keep the first live class docs-only and exact-scope approved.",
      "Do not treat PR rehearsal readiness as broad live mutation authority.",
      "Fully unsupervised complex repository mutation remains out of scope."
    ],
    authority_boundaries:{
      emits_rollup_only:true,
      live_mutation_performed:false,
      mutates_repositories:false,
      creates_branch:false,
      creates_worktree:false,
      opens_pr:false,
      merges_pr:false,
      schedules_work:false,
      executes_work:false,
      approves_work:false,
      provider_calls_allowed:false,
      release_or_publish_allowed:false,
      broad_live_mutation_allowed:false,
      fully_unsupervised_complex_mutation_claimed:false
    }
  }' > "$out"

jq empty "$out"
echo "first_live_docs_readiness_rollup=$status"
echo "safe_to_request=$safe_to_request"
echo "safe_to_execute=$safe_to_execute"
echo "exact_next_step=$exact_next_step"
echo "rollup=$out"
