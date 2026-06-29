#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
usage: scripts/live-docs-rollback-execution-rehearsal.sh --candidate <candidate.json> --out <rehearsal.json> [--json]

Executes a docs-only proposed patch and rollback patch inside a temporary
fixture workspace, then emits ao.foundry.live-docs-rollback-execution-rehearsal.v0.1.
It never applies patches to this repository, creates branches, pushes, uploads,
publishes, releases, calls providers, approves work, or mutates sibling repos.
USAGE
}

candidate=""
out=""
json=0
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --candidate) candidate="${2:-}"; shift 2 ;;
    --out) out="${2:-}"; shift 2 ;;
    --json) json=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "live-docs-rollback-execution-rehearsal: unknown argument $1" >&2; usage >&2; exit 2 ;;
  esac
done

if [[ -z "$candidate" || -z "$out" ]]; then
  usage >&2
  exit 2
fi
if [[ "$out" == "$candidate" ]]; then
  echo "live-docs-rollback-execution-rehearsal: --out must not overwrite candidate" >&2
  exit 2
fi
for path_arg in "$candidate" "$out"; do
  case "$path_arg" in
    /*|~*|*"/.."*|*".."/*)
      echo "live-docs-rollback-execution-rehearsal: paths must be public-safe relative paths" >&2
      exit 2
      ;;
  esac
done
if [[ ! -f "$candidate" ]]; then
  echo "live-docs-rollback-execution-rehearsal: candidate not found: $candidate" >&2
  exit 2
fi
if ! command -v jq >/dev/null 2>&1; then
  echo "live-docs-rollback-execution-rehearsal: jq is required" >&2
  exit 2
fi
if ! command -v git >/dev/null 2>&1; then
  echo "live-docs-rollback-execution-rehearsal: git is required" >&2
  exit 2
fi

sha256_file() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    echo "live-docs-rollback-execution-rehearsal: shasum or sha256sum is required" >&2
    exit 2
  fi
}

sha256_file_or_empty() {
  if [[ -n "$1" && -f "$1" ]]; then
    sha256_file "$1"
  else
    printf ''
  fi
}

sha256_text() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 | awk '{print $1}'
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum | awk '{print $1}'
  else
    echo "live-docs-rollback-execution-rehearsal: shasum or sha256sum is required" >&2
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

candidate_value() {
  jq -r "$1 // \"\"" "$candidate"
}

schema_version="$(candidate_value '.schema_version')"
candidate_id="$(candidate_value '.candidate_id')"
repo_id="$(candidate_value '.target_repo.id')"
repo_remote="$(candidate_value '.target_repo.remote')"
target_branch="$(candidate_value '.target_repo.target_branch')"
change_class="$(candidate_value '.change_class')"
target_file="$(candidate_value '.target_file')"
proposed_patch_path="$(candidate_value '.proposed_patch_path')"
rollback_patch_path="$(candidate_value '.rollback_patch_path')"
kill_switch_state="$(candidate_value '.kill_switch_state')"

if [[ "$schema_version" == "ao.foundry.live-docs-rollback-execution-candidate.v0.1" ]]; then
  add_check "candidate_schema" "passed" "candidate uses the expected live docs rollback execution schema"
else
  add_check "candidate_schema" "blocked" "candidate schema is missing or unsupported"
fi
unsafe_regex='(^/|^~|(^|/)\.\.(/|$))'
unsafe_strings="$(jq -r --arg unsafe_re "$unsafe_regex" '.. | strings | select(test($unsafe_re))' "$candidate" | head -20)"
if [[ -z "$unsafe_strings" ]]; then
  add_check "public_safe_paths" "passed" "candidate contains no absolute, home, temp, or parent-relative paths"
else
  add_check "public_safe_paths" "blocked" "candidate contains unsafe paths"
fi
if [[ "$repo_id" =~ ^[a-z0-9][a-z0-9_-]*$ && "$repo_remote" =~ ^uesugitorachiyo/[a-z0-9][a-z0-9_-]*$ && "$target_branch" == "main" ]]; then
  add_check "target_repo" "passed" "target repo is public-safe and targets main through PR lifecycle"
else
  add_check "target_repo" "blocked" "target repo metadata is missing or unsafe"
fi
if [[ "$change_class" == "docs_only" && "$target_file" =~ ^docs/[A-Za-z0-9._/-]+\.md$ ]]; then
  add_check "docs_only_target" "passed" "rollback execution target is a docs-only markdown path"
else
  add_check "docs_only_target" "blocked" "rollback execution target must be docs-only markdown"
fi
if [[ "$proposed_patch_path" =~ ^examples/live-docs-rollback-execution/[A-Za-z0-9._/-]+\.patch$ && -f "$proposed_patch_path" ]]; then
  add_check "proposed_patch_present" "passed" "proposed docs patch fixture exists"
else
  add_check "proposed_patch_present" "blocked" "proposed docs patch fixture is missing or unsafe"
fi
if [[ "$rollback_patch_path" =~ ^examples/live-docs-rollback-execution/[A-Za-z0-9._/-]+\.patch$ && -f "$rollback_patch_path" ]]; then
  add_check "rollback_patch_present" "passed" "rollback docs patch fixture exists"
else
  add_check "rollback_patch_present" "blocked" "rollback docs patch fixture is missing or unsafe"
fi
if [[ "$kill_switch_state" == "armed" ]]; then
  add_check "kill_switch" "passed" "operator kill switch is armed before rollback execution rehearsal"
else
  add_check "kill_switch" "blocked" "operator kill switch must be armed"
fi
if jq -e '
  .authority_boundaries.fixture_workspace_only == true and
  .authority_boundaries.live_mutation_allowed == false and
  .authority_boundaries.mutates_repositories == false and
  .authority_boundaries.schedules_work == false and
  .authority_boundaries.executes_work == false and
  .authority_boundaries.approves_work == false and
  .authority_boundaries.provider_calls_allowed == false and
  .authority_boundaries.release_or_publish_allowed == false
' "$candidate" >/dev/null; then
  add_check "authority_boundaries" "passed" "candidate preserves fixture-workspace-only non-authority boundaries"
else
  add_check "authority_boundaries" "blocked" "candidate attempts mutation, execution, approval, provider, release, or publish authority"
fi

apply_sha=""
rollback_sha=""
if [[ -z "$first_failing_check" ]]; then
  workspace="$tmpdir/workspace"
  mkdir -p "$workspace"
  git -C "$workspace" init -q
  if git -C "$workspace" apply --whitespace=nowarn --check "$repo_root/$proposed_patch_path" &&
    git -C "$workspace" apply --whitespace=nowarn "$repo_root/$proposed_patch_path" &&
    [[ -f "$workspace/$target_file" ]]; then
    apply_sha="$(sha256_file "$workspace/$target_file")"
    add_check "proposed_patch_apply" "passed" "proposed docs patch applies inside the fixture workspace"
  else
    add_check "proposed_patch_apply" "blocked" "proposed docs patch failed inside the fixture workspace"
  fi
fi
if [[ -z "$first_failing_check" ]]; then
  if git -C "$workspace" apply --whitespace=nowarn --check "$repo_root/$rollback_patch_path" &&
    git -C "$workspace" apply --whitespace=nowarn "$repo_root/$rollback_patch_path" &&
    [[ ! -e "$workspace/$target_file" ]]; then
    rollback_sha="$(printf 'removed:%s' "$target_file" | sha256_text)"
    add_check "rollback_patch_apply" "passed" "rollback patch restores the fixture workspace by removing the proposed docs file"
  else
    add_check "rollback_patch_apply" "blocked" "rollback patch failed to restore the fixture workspace"
  fi
fi

status="ready"
rollback_verified="true"
if [[ -n "$first_failing_check" ]]; then
  status="blocked"
  rollback_verified="false"
fi

mkdir -p "$(dirname "$out")"
checks_json="$tmpdir/checks.json"
jq -s '.' "$checks_file" > "$checks_json"

jq -n \
  --arg schema_version "ao.foundry.live-docs-rollback-execution-rehearsal.v0.1" \
  --arg status "$status" \
  --argjson rollback_verified "$rollback_verified" \
  --arg candidate_path "$candidate" \
  --arg candidate_sha256 "$(sha256_file "$candidate")" \
  --arg candidate_id "$candidate_id" \
  --arg repo_id "$repo_id" \
  --arg repo_remote "$repo_remote" \
  --arg target_branch "$target_branch" \
  --arg target_file "$target_file" \
  --arg proposed_patch_path "$proposed_patch_path" \
  --arg proposed_patch_sha "$(sha256_file_or_empty "$proposed_patch_path")" \
  --arg rollback_patch_path "$rollback_patch_path" \
  --arg rollback_patch_sha "$(sha256_file_or_empty "$rollback_patch_path")" \
  --arg apply_sha "$apply_sha" \
  --arg rollback_sha "$rollback_sha" \
  --arg first_failing_check "$first_failing_check" \
  --slurpfile checks "$checks_json" \
  '{
    schema_version:$schema_version,
    status:$status,
    mode:"fixture_workspace_only",
    rollback_verified:$rollback_verified,
    candidate_id:$candidate_id,
    target_repo:{id:$repo_id, remote:$repo_remote, target_branch:$target_branch},
    target_file:$target_file,
    source_hashes:([
      {name:"candidate", path:$candidate_path, schema_version:"ao.foundry.live-docs-rollback-execution-candidate.v0.1", sha256:$candidate_sha256}
    ] +
    (if $proposed_patch_sha == "" then [] else [
      {name:"proposed_patch", path:$proposed_patch_path, schema_version:"patch", sha256:$proposed_patch_sha}
    ] end) +
    (if $rollback_patch_sha == "" then [] else [
      {name:"rollback_patch", path:$rollback_patch_path, schema_version:"patch", sha256:$rollback_patch_sha}
    ] end)),
    execution_summary:{
      proposed_patch_applied:($status == "ready"),
      rollback_patch_applied:($status == "ready"),
      target_after_apply_sha256:$apply_sha,
      target_after_rollback_sha256:$rollback_sha,
      fixture_workspace_removed:true
    },
    checks:$checks[0],
    first_failing_check:$first_failing_check,
    blocking_next_actions:(if $status == "ready" then [] else [
      "Repair the proposed and rollback docs-only patch fixtures.",
      "Keep the rollback target within the docs-only allowlist.",
      "Rerun this rehearsal before any first live docs-only PR rehearsal gate can pass."
    ] end),
    maintenance_suggestions:[
      "This rehearsal executes patches only inside a temporary fixture workspace.",
      "Do not treat this as permission to apply patches to the live repository."
    ],
    authority_boundaries:{
      fixture_workspace_only:true,
      live_mutation_allowed:false,
      mutates_repositories:false,
      schedules_work:false,
      executes_work:false,
      approves_work:false,
      provider_calls_allowed:false,
      release_or_publish_allowed:false
    }
  }' > "$out"

if [[ "$json" -eq 1 ]]; then
  cat "$out"
else
  echo "live_docs_rollback_execution_rehearsal=$status"
  echo "rollback_verified=$rollback_verified"
  echo "rehearsal=$out"
fi

if [[ "$status" != "ready" ]]; then
  exit 1
fi
