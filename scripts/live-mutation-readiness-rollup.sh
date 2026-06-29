#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
usage: scripts/live-mutation-readiness-rollup.sh --chain <summary.json> --out <rollup.json>

Summarizes governed live-mutation dry-run readiness. This rollup does not
execute work, approve live mutation, create branches, apply patches, call
providers, upload, publish, release, or mutate repositories.
USAGE
}

CHAIN=""
OUT=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --chain)
      CHAIN="${2:-}"
      shift 2
      ;;
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

if [[ -z "$CHAIN" || -z "$OUT" ]]; then
  echo "--chain and --out are required" >&2
  usage >&2
  exit 2
fi

case "$CHAIN" in
  /*|~*|*"/.."*|*".."/*)
    echo "--chain must be a public-safe relative path" >&2
    exit 2
    ;;
esac

case "$OUT" in
  /*|~*|*"/.."*|*".."/*)
    echo "--out must be a public-safe relative path" >&2
    exit 2
    ;;
esac

if [[ "$OUT" == "$CHAIN" ]]; then
  echo "--out must not overwrite the chain summary" >&2
  exit 2
fi

if [[ ! -f "$CHAIN" ]]; then
  echo "chain summary not found: $CHAIN" >&2
  exit 2
fi

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

schema_version="$(jq -r '.schema_version // ""' "$CHAIN")"
chain_status="$(jq -r '.status // ""' "$CHAIN")"
chain_assessment="$(jq -r '.readiness_assessment.governed_live_mutation // "blocked"' "$CHAIN")"
safe_to_request="$(jq -r '.readiness_assessment.first_tiny_live_mutation_class_safe_to_request // false' "$CHAIN")"
live_mutation_performed="$(jq -r 'if .readiness_assessment.live_mutation_performed == false then "false" else "true" end' "$CHAIN")"
ungated_claim="$(jq -r 'if .readiness_assessment.ungated_live_mutation_claim == false then "false" else "true" end' "$CHAIN")"
first_failing_check="$(jq -r '.first_failing_check // ""' "$CHAIN")"
source_count="$(jq -r '.source_artifacts | length' "$CHAIN")"

status="ready"
score=100
exact_next_step="submit_operator_approval_request_for_first_tiny_docs_only_live_mutation_class"
if [[ "$schema_version" != "ao.foundry.governed-live-mutation-dry-run-chain.v0.1" ||
  "$chain_status" != "ready" ||
  "$chain_assessment" != "dry_run_ready_for_request" ||
  "$safe_to_request" != "true" ||
  "$live_mutation_performed" != "false" ||
  "$ungated_claim" != "false" ||
  "$source_count" -lt 10 ]]; then
  status="blocked"
  score=80
  exact_next_step="repair_governed_live_mutation_dry_run_chain_before_request"
fi

mkdir -p "$(dirname "$OUT")"
jq -n \
  --arg schema_version "ao.foundry.live-mutation-readiness-rollup.v0.1" \
  --arg status "$status" \
  --argjson score "$score" \
  --arg chain_path "$CHAIN" \
  --arg chain_sha256 "$(sha256_file "$CHAIN")" \
  --arg chain_status "$chain_status" \
  --arg chain_assessment "$chain_assessment" \
  --arg first_failing_check "$first_failing_check" \
  --arg exact_next_step "$exact_next_step" \
  --argjson source_count "$source_count" \
  '{
    schema_version:$schema_version,
    status:$status,
    score:$score,
    source_chain:{
      path:$chain_path,
      schema_version:"ao.foundry.governed-live-mutation-dry-run-chain.v0.1",
      status:$chain_status,
      readiness_assessment:$chain_assessment,
      sha256:$chain_sha256,
      source_artifact_count:$source_count
    },
    dry_run_authority_path:[
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
    first_failing_check:$first_failing_check,
    residual_blockers:(if $status == "ready" then [] else ["governed live-mutation dry-run chain is not ready"] end),
    exact_next_step:$exact_next_step,
    first_tiny_live_mutation_class:{
      class:"docs_only",
      safe_to_request:($status == "ready"),
      safe_to_execute:false,
      requires_operator_approval:true,
      requires_new_branch_pr_lifecycle:true,
      requires_kill_switch_armed:true
    },
    remaining_risks:[
      "No live repository mutation has been executed by this proof.",
      "The first tiny live-mutation class still needs a separate operator-approved request.",
      "Any execution attempt must preserve Covenant, Sentinel, Promoter, rollback, worktree, PR lifecycle, and Command readback evidence."
    ],
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
  }' > "$OUT"

if [[ "$status" == "ready" ]]; then
  echo "live_mutation_readiness_rollup=ready"
  echo "score=$score"
  echo "exact_next_step=$exact_next_step"
else
  echo "live_mutation_readiness_rollup=blocked first_failing_check=$first_failing_check" >&2
  exit 1
fi
