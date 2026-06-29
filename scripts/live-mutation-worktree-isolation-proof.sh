#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
usage: scripts/live-mutation-worktree-isolation-proof.sh --candidate <candidate.json> --out <proof.json> [--json]

Validates a dry-run live-mutation worktree candidate and emits
ao.foundry.worktree-isolation-proof.v0.1 evidence. This script never creates
branches, checks out worktrees, mutates repositories, calls providers, uploads,
publishes, or approves live mutation.
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
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    echo "shasum or sha256sum is required" >&2
    exit 2
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
worktree_path="$(candidate_value '.worktree.path')"
branch="$(candidate_value '.worktree.branch')"
base_branch="$(candidate_value '.worktree.base_branch')"
clean="$(jq -r '.worktree.clean' "$CANDIDATE")"
reused="$(jq -r '.worktree.reused' "$CANDIDATE")"
active_elsewhere="$(jq -r '.worktree.active_elsewhere' "$CANDIDATE")"
has_untracked_changes="$(jq -r '.worktree.has_untracked_changes' "$CANDIDATE")"
remote_branch_exists="$(jq -r '.worktree.remote_branch_exists' "$CANDIDATE")"

if [[ "$schema_version" == "ao.foundry.live-mutation-worktree-candidate.v0.1" ]]; then
  add_check "candidate_schema" "passed" "candidate uses the expected dry-run worktree schema"
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
  add_check "change_class" "blocked" "candidate change class is not allowed for live-mutation rehearsal"
fi

if [[ "$worktree_path" =~ ^\.foundry-local/worktrees/[A-Za-z0-9._/-]+$ && "$branch" =~ ^codex/[A-Za-z0-9._/-]+$ && "$base_branch" == "main" ]]; then
  add_check "branch_isolation" "passed" "candidate uses an isolated codex branch worktree rooted in ignored local state"
else
  add_check "branch_isolation" "blocked" "candidate must use an isolated .foundry-local worktree on a codex/* branch from main"
fi

if [[ "$clean" == "true" && "$has_untracked_changes" == "false" ]]; then
  add_check "clean_worktree" "passed" "candidate worktree is clean"
else
  add_check "clean_worktree" "blocked" "candidate worktree is dirty or has untracked changes"
fi

if [[ "$reused" == "false" && "$active_elsewhere" == "false" && "$remote_branch_exists" == "false" ]]; then
  add_check "reuse_block" "passed" "candidate worktree and branch are not reused or active elsewhere"
else
  add_check "reuse_block" "blocked" "candidate worktree or branch is reused, active elsewhere, or already remote"
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
  --arg schema_version "ao.foundry.worktree-isolation-proof.v0.1" \
  --arg status "$status" \
  --arg candidate_path "$CANDIDATE" \
  --arg candidate_sha256 "$(sha256_file "$CANDIDATE")" \
  --arg repo_id "$repo_id" \
  --arg repo_remote "$repo_remote" \
  --arg target_branch "$target_branch" \
  --arg change_class "$change_class" \
  --arg worktree_path "$worktree_path" \
  --arg branch "$branch" \
  --arg base_branch "$base_branch" \
  --argjson clean "$(jq '.worktree.clean // false' "$CANDIDATE")" \
  --argjson reused "$(jq '.worktree.reused // false' "$CANDIDATE")" \
  --argjson active_elsewhere "$(jq '.worktree.active_elsewhere // false' "$CANDIDATE")" \
  --argjson has_untracked_changes "$(jq '.worktree.has_untracked_changes // false' "$CANDIDATE")" \
  --argjson remote_branch_exists "$(jq '.worktree.remote_branch_exists // false' "$CANDIDATE")" \
  --arg first_failing_check "$first_failing_check" \
  --slurpfile checks "$checks_json" \
  '{
    schema_version:$schema_version,
    status:$status,
    mode:"dry_run_only",
    candidate_path:$candidate_path,
    candidate_sha256:$candidate_sha256,
    target_repo:{id:$repo_id, remote:$repo_remote, target_branch:$target_branch},
    change_class:$change_class,
    worktree:{
      path:$worktree_path,
      branch:$branch,
      base_branch:$base_branch,
      clean:$clean,
      reused:$reused,
      active_elsewhere:$active_elsewhere,
      has_untracked_changes:$has_untracked_changes,
      remote_branch_exists:$remote_branch_exists
    },
    checks:$checks[0],
    first_failing_check:$first_failing_check,
    blocking_next_actions:(if $status == "ready" then [] else [
      "Create a fresh isolated worktree under .foundry-local/worktrees.",
      "Use a new codex/* branch from synced main.",
      "Clean dirty and untracked state before requesting live-mutation authority."
    ] end),
    maintenance_suggestions:[
      "Keep worktree isolation evidence dry-run until the full Covenant, Forge, AO2, Sentinel, Promoter, and Command chain is green.",
      "Do not reuse a worktree or branch for a separate live-mutation candidate."
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
  echo "worktree_isolation_proof=$status"
  echo "proof=$OUT"
fi

if [[ "$status" != "ready" ]]; then
  exit 1
fi
