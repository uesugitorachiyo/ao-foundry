#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
usage: scripts/live-mutation-rollback-rehearsal.sh --candidate <candidate.json> --out <rehearsal.json> [--json]

Validates a dry-run rollback rehearsal candidate and emits
ao.foundry.live-mutation-rollback-rehearsal.v0.1 evidence. This script never
applies patches, creates branches, mutates repositories, calls providers,
uploads, publishes, releases, or approves live mutation.
USAGE
}

CANDIDATE=""
OUT=""
JSON=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --candidate)
      CANDIDATE="${2:-}"
      shift 2
      ;;
    --out)
      OUT="${2:-}"
      shift 2
      ;;
    --json)
      JSON=1
      shift
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

if [[ -z "$CANDIDATE" || -z "$OUT" ]]; then
  echo "--candidate and --out are required" >&2
  usage >&2
  exit 2
fi

case "$CANDIDATE" in
  /*|~*|tmp/*|*"/.."*|*".."/*)
    echo "--candidate must be a public-safe relative path" >&2
    exit 2
    ;;
esac

case "$OUT" in
  ~*|*"/.."*|*".."/*)
    echo "--out must not use home or parent-relative paths" >&2
    exit 2
    ;;
esac

if [[ "$OUT" == "$CANDIDATE" ]]; then
  echo "--out must not overwrite the candidate artifact" >&2
  exit 2
fi

if [[ ! -f "$CANDIDATE" ]]; then
  echo "candidate not found: $CANDIDATE" >&2
  exit 2
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required" >&2
  exit 2
fi

sha256_file() {
  if [[ -f "$1" ]]; then
    if command -v shasum >/dev/null 2>&1; then
      shasum -a 256 "$1" | awk '{print $1}'
    elif command -v sha256sum >/dev/null 2>&1; then
      sha256sum "$1" | awk '{print $1}'
    else
      echo "shasum or sha256sum is required" >&2
      exit 2
    fi
  else
    printf '%064d' 0
  fi
}

json_escape() {
  python3 -c 'import json,sys; print(json.dumps(sys.stdin.read()))'
}

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT
CHECKS_FILE="$TMPDIR/checks.jsonl"
: > "$CHECKS_FILE"
first_failing_check=""

add_check() {
  local name="$1"
  local status="$2"
  local summary="$3"
  printf '{"name":%s,"status":%s,"summary":%s}\n' \
    "$(printf '%s' "$name" | json_escape)" \
    "$(printf '%s' "$status" | json_escape)" \
    "$(printf '%s' "$summary" | json_escape)" >> "$CHECKS_FILE"
  if [[ "$status" != "passed" && -z "$first_failing_check" ]]; then
    first_failing_check="$name"
  fi
}

candidate_value() {
  jq -r "$1 // \"\"" "$CANDIDATE"
}

schema_version="$(candidate_value '.schema_version')"
repo_id="$(candidate_value '.target_repo.id')"
repo_remote="$(candidate_value '.target_repo.remote')"
target_branch="$(candidate_value '.target_repo.target_branch')"
change_class="$(candidate_value '.change_class')"
proposed_patch_path="$(candidate_value '.proposed_patch_path')"
rollback_patch_path="$(candidate_value '.rollback_patch_path')"
strategy="$(candidate_value '.rollback_plan.strategy')"
quarantine_path="$(candidate_value '.rollback_plan.quarantine_path')"
kill_switch_state="$(candidate_value '.rollback_plan.kill_switch_state')"

if [[ "$schema_version" == "ao.foundry.live-mutation-rollback-candidate.v0.1" ]]; then
  add_check "candidate_schema" "passed" "candidate uses the expected dry-run rollback schema"
else
  add_check "candidate_schema" "blocked" "candidate schema is missing or unsupported"
fi

unsafe_strings="$(jq -r '.. | strings | select(test("(^/|^~|(^|/)\\.\\.(/|$)|/Users/|/home/|/tmp/|^tmp/)"))' "$CANDIDATE" | head -20)"
if [[ -z "$unsafe_strings" ]]; then
  add_check "public_safe_paths" "passed" "candidate contains no absolute, home, temp, or parent-relative paths"
else
  add_check "public_safe_paths" "blocked" "candidate contains unsafe local paths"
fi

if [[ "$repo_id" =~ ^[a-z0-9][a-z0-9_-]*$ && "$repo_remote" =~ ^uesugitorachiyo/[a-z0-9][a-z0-9_-]*$ && "$target_branch" == "main" ]]; then
  add_check "target_repo" "passed" "target repo is public-safe and targets main through PR lifecycle"
else
  add_check "target_repo" "blocked" "target repo metadata is missing or unsafe"
fi

if [[ "$change_class" == "docs_only" || "$change_class" == "fixture_only" || "$change_class" == "test_only" ]]; then
  add_check "change_class" "passed" "candidate uses a tiny allowed live-mutation class"
else
  add_check "change_class" "blocked" "candidate change class is not allowed for rollback rehearsal"
fi

if [[ "$proposed_patch_path" =~ ^examples/live-mutation-rollback/[A-Za-z0-9._/-]+\.patch$ && -f "$proposed_patch_path" ]]; then
  add_check "proposed_patch_present" "passed" "proposed patch fixture exists and can be digest-bound"
else
  add_check "proposed_patch_present" "blocked" "proposed patch fixture is missing or unsafe"
fi

if [[ "$rollback_patch_path" =~ ^examples/live-mutation-rollback/[A-Za-z0-9._/-]+\.patch$ && -f "$rollback_patch_path" ]]; then
  add_check "rollback_patch_present" "passed" "rollback patch fixture exists and can be digest-bound"
else
  add_check "rollback_patch_present" "blocked" "rollback patch fixture is missing or unsafe"
fi

if [[ "$strategy" == "reverse_patch_then_quarantine" || "$strategy" == "restore_from_clean_base_then_quarantine" ]] &&
  [[ "$quarantine_path" =~ ^\.foundry-local/quarantine/[A-Za-z0-9._/-]+$ ]]; then
  add_check "quarantine_plan" "passed" "rollback plan includes a local ignored quarantine path"
else
  add_check "quarantine_plan" "blocked" "rollback plan must include an ignored .foundry-local quarantine path"
fi

if [[ "$kill_switch_state" == "armed" ]]; then
  add_check "kill_switch" "passed" "operator kill switch is armed before live mutation authority can advance"
else
  add_check "kill_switch" "blocked" "operator kill switch must be armed"
fi

if jq -e '.rollback_plan.verification_commands | type == "array" and length > 0 and all(.[]; test("^(go test|go vet|go build|scripts/|jq empty|git diff --check)"))' "$CANDIDATE" >/dev/null; then
  add_check "verification_plan" "passed" "rollback rehearsal includes local verification commands only"
else
  add_check "verification_plan" "blocked" "rollback rehearsal verification plan is missing or unsafe"
fi

if jq -e '
  .authority_boundaries.dry_run_only == true and
  .authority_boundaries.live_mutation_allowed == false and
  .authority_boundaries.mutates_repositories == false and
  .authority_boundaries.schedules_work == false and
  .authority_boundaries.executes_work == false and
  .authority_boundaries.approves_work == false and
  .authority_boundaries.provider_calls_allowed == false and
  .authority_boundaries.release_or_publish_allowed == false
' "$CANDIDATE" >/dev/null; then
  add_check "authority_boundaries" "passed" "candidate preserves dry-run-only non-authority boundaries"
else
  add_check "authority_boundaries" "blocked" "candidate attempts scheduling, execution, approval, mutation, provider, release, or publish authority"
fi

status="ready"
if [[ -n "$first_failing_check" ]]; then
  status="blocked"
fi

mkdir -p "$(dirname "$OUT")"
checks_json="$TMPDIR/checks.json"
jq -s '.' "$CHECKS_FILE" > "$checks_json"

jq -n \
  --arg schema_version "ao.foundry.live-mutation-rollback-rehearsal.v0.1" \
  --arg status "$status" \
  --arg candidate_path "$CANDIDATE" \
  --arg candidate_sha256 "$(sha256_file "$CANDIDATE")" \
  --arg repo_id "$repo_id" \
  --arg repo_remote "$repo_remote" \
  --arg target_branch "$target_branch" \
  --arg change_class "$change_class" \
  --arg strategy "$strategy" \
  --arg quarantine_path "$quarantine_path" \
  --arg kill_switch_state "$kill_switch_state" \
  --arg proposed_patch_path "$proposed_patch_path" \
  --arg proposed_patch_sha "$(sha256_file "$proposed_patch_path")" \
  --arg rollback_patch_path "$rollback_patch_path" \
  --arg rollback_patch_sha "$(sha256_file "$rollback_patch_path")" \
  --arg first_failing_check "$first_failing_check" \
  --slurpfile checks "$checks_json" \
  --slurpfile candidate "$CANDIDATE" \
  '{
    schema_version:$schema_version,
    status:$status,
    mode:"dry_run_only",
    candidate_path:$candidate_path,
    candidate_sha256:$candidate_sha256,
    target_repo:{id:$repo_id, remote:$repo_remote, target_branch:$target_branch},
    change_class:$change_class,
    rollback_plan:{
      strategy:$strategy,
      quarantine_path:$quarantine_path,
      kill_switch_state:$kill_switch_state,
      verification_commands:($candidate[0].rollback_plan.verification_commands // [])
    },
    patch_artifacts:[
      {name:"proposed_patch", path:$proposed_patch_path, sha256:$proposed_patch_sha},
      {name:"rollback_patch", path:$rollback_patch_path, sha256:$rollback_patch_sha}
    ],
    checks:$checks[0],
    first_failing_check:$first_failing_check,
    blocking_next_actions:(if $status == "ready" then [] else [
      "Provide a digest-bound rollback patch before requesting live-mutation authority.",
      "Confirm the operator kill switch is armed.",
      "Define an ignored .foundry-local quarantine path and local verification commands."
    ] end),
    maintenance_suggestions:[
      "Keep rollback rehearsal dry-run until Sentinel, Promoter, and Command readback evidence also pass.",
      "Do not apply proposed or rollback patches from this proof."
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

if [[ "$JSON" -eq 1 ]]; then
  cat "$OUT"
else
  echo "live_mutation_rollback_rehearsal=$status"
  echo "rehearsal=$OUT"
fi

if [[ "$status" != "ready" ]]; then
  exit 1
fi
