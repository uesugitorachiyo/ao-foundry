#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
usage: scripts/live-docs-worktree-prepare.sh --candidate <candidate.json> --approval-gate <gate.json> --out <prepare.json> [--json]

Validates the isolated branch/worktree candidate for the first tiny docs-only
live PR rehearsal. This gate emits evidence only; it does not create branches,
check out worktrees, mutate repositories, call providers, upload, publish, tag,
release, or approve work.
USAGE
}

candidate=""
approval_gate=""
out=""
json=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --candidate) candidate="${2:-}"; shift 2 ;;
    --approval-gate) approval_gate="${2:-}"; shift 2 ;;
    --out) out="${2:-}"; shift 2 ;;
    --json) json=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "live-docs-worktree-prepare: unknown argument $1" >&2; usage >&2; exit 2 ;;
  esac
done

if [[ -z "$candidate" || -z "$approval_gate" || -z "$out" ]]; then
  usage >&2
  exit 2
fi

private_mac_root="/""Users/"
private_linux_root="/""home/"
private_tmp_root="/""tmp/"
path_bundle="$candidate:$approval_gate:$out"
for unsafe_arg_marker in ".." "~" "$private_mac_root" "$private_linux_root" "$private_tmp_root"; do
  if [[ "$path_bundle" == *"$unsafe_arg_marker"* ]]; then
    echo "live-docs-worktree-prepare: paths must be public-safe relative paths" >&2
    exit 2
  fi
done

if [[ "$out" == "$candidate" || "$out" == "$approval_gate" ]]; then
  echo "live-docs-worktree-prepare: --out must not overwrite input artifacts" >&2
  exit 2
fi

if [[ ! -f "$candidate" ]]; then
  echo "live-docs-worktree-prepare: candidate not found: $candidate" >&2
  exit 2
fi
if [[ ! -f "$approval_gate" ]]; then
  echo "live-docs-worktree-prepare: approval gate not found: $approval_gate" >&2
  exit 2
fi
if ! command -v jq >/dev/null 2>&1; then
  echo "live-docs-worktree-prepare: jq is required" >&2
  exit 2
fi

sha256_file() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    echo "live-docs-worktree-prepare: shasum or sha256sum is required" >&2
    exit 2
  fi
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
checks_file="$tmpdir/checks.jsonl"
: > "$checks_file"
first_failing_check=""

json_string() {
  jq -Rsa .
}

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

gate_value() {
  jq -r "$1 // \"\"" "$approval_gate"
}

candidate_schema="$(candidate_value '.schema_version')"
gate_schema="$(gate_value '.schema_version')"
gate_status="$(gate_value '.status')"
gate_safe_to_execute="$(jq -r 'if .safe_to_execute == true then "true" else "false" end' "$approval_gate")"
candidate_id="$(candidate_value '.candidate_id')"
repo_id="$(candidate_value '.target_repo.id')"
repo_remote="$(candidate_value '.target_repo.remote')"
target_branch="$(candidate_value '.target_repo.target_branch')"
change_class="$(candidate_value '.change_class')"
worktree_path="$(candidate_value '.worktree.path')"
branch="$(candidate_value '.worktree.branch')"
base_branch="$(candidate_value '.worktree.base_branch')"
clean="$(jq -r 'if .worktree.clean == true then "true" else "false" end' "$candidate")"
reused="$(jq -r 'if .worktree.reused == true then "true" else "false" end' "$candidate")"
active_elsewhere="$(jq -r 'if .worktree.active_elsewhere == true then "true" else "false" end' "$candidate")"
has_untracked_changes="$(jq -r 'if .worktree.has_untracked_changes == true then "true" else "false" end' "$candidate")"
remote_branch_exists="$(jq -r 'if .worktree.remote_branch_exists == true then "true" else "false" end' "$candidate")"
max_changed_files="$(jq -r '.change_plan.max_changed_files // 0' "$candidate")"
kill_switch_state="$(candidate_value '.kill_switch_ref.state')"

if [[ "$candidate_schema" == "ao.foundry.live-docs-worktree-preparation-candidate.v0.1" ]]; then
  add_check "candidate_schema" "passed" "candidate uses the expected first live docs worktree preparation schema"
else
  add_check "candidate_schema" "blocked" "candidate schema is missing or unsupported"
fi

if [[ "$gate_schema" == "ao.foundry.live-docs-approval-gate.v0.1" && "$gate_status" == "ready" && "$gate_safe_to_execute" == "true" ]]; then
  add_check "approval_gate" "passed" "approval gate is ready and exact-scope safe_to_execute is true"
else
  add_check "approval_gate" "blocked" "approval gate must be ready before a docs-only PR rehearsal can be prepared"
fi

unsafe_regex="$(printf '(^/|^~|(^|/)\\.\\.(/|$)|%s|%s|%s|^tmp/)' "$private_mac_root" "$private_linux_root" "$private_tmp_root")"
unsafe_strings="$(jq -r --arg unsafe_re "$unsafe_regex" '.. | strings | select(test($unsafe_re))' "$candidate" "$approval_gate" | head -20)"
if [[ -z "$unsafe_strings" ]]; then
  add_check "public_safe_paths" "passed" "candidate and approval gate contain no absolute, home, temp, or parent-relative paths"
else
  add_check "public_safe_paths" "blocked" "candidate or approval gate contains unsafe local paths"
fi

if [[ "$repo_id" =~ ^[a-z0-9][a-z0-9_-]*$ && "$repo_remote" =~ ^uesugitorachiyo/[a-z0-9][a-z0-9_-]*$ && "$target_branch" == "main" ]]; then
  add_check "target_repo" "passed" "target repo is public-safe and targets main through PR lifecycle"
else
  add_check "target_repo" "blocked" "target repo metadata is missing or unsafe"
fi

if [[ "$change_class" == "docs_only" ]]; then
  add_check "change_class" "passed" "candidate is scoped to the first tiny docs-only live class"
else
  add_check "change_class" "blocked" "candidate change class must be docs_only"
fi

if [[ "$worktree_path" =~ ^\.foundry-local/worktrees/[A-Za-z0-9._/-]+$ && "$branch" =~ ^codex/live-docs-[A-Za-z0-9._/-]+$ && "$base_branch" == "main" ]]; then
  add_check "branch_isolation" "passed" "candidate uses an isolated ignored worktree on a live-docs codex branch from main"
else
  add_check "branch_isolation" "blocked" "candidate must use .foundry-local/worktrees and a codex/live-docs-* branch from main"
fi

if [[ "$clean" == "true" && "$has_untracked_changes" == "false" ]]; then
  add_check "clean_worktree" "passed" "candidate worktree is clean and has no untracked changes"
else
  add_check "clean_worktree" "blocked" "candidate worktree is dirty or has untracked changes"
fi

if [[ "$reused" == "false" && "$active_elsewhere" == "false" && "$remote_branch_exists" == "false" ]]; then
  add_check "reuse_block" "passed" "candidate worktree and branch are fresh and not active elsewhere"
else
  add_check "reuse_block" "blocked" "candidate worktree or branch is reused, active elsewhere, or already remote"
fi

if jq -e '
  (.change_plan.changed_files // []) as $files |
  (.change_plan.docs_only_path_allowlist // []) as $allowlist |
  (.change_plan.forbidden_paths // []) as $forbidden |
  (.change_plan.max_changed_files // 0) as $max |
  ($files | type == "array" and length > 0) and
  ($max | type == "number") and
  ($max >= ($files | length)) and
  ($max <= 2) and
  all($files[]; test("^docs/[A-Za-z0-9._/-]+\\.md$")) and
  all($allowlist[]; test("^docs/")) and
  all($forbidden[]; test("^(\\.github/|cmd/|internal/|scripts/|docs/contracts/|examples/|go\\.mod|go\\.sum|LICENSE|SECURITY\\.md|README\\.md)$"))
' "$candidate" >/dev/null; then
  add_check "docs_only_path_plan" "passed" "changed-file plan is bounded to docs/*.md and excludes code, scripts, contracts, and repo metadata"
else
  add_check "docs_only_path_plan" "blocked" "changed-file plan must stay within allowed docs-only markdown paths and max file count"
fi

if [[ "$kill_switch_state" == "armed" ]]; then
  add_check "kill_switch" "passed" "operator kill switch evidence is armed before preparation may pass"
else
  add_check "kill_switch" "blocked" "operator kill switch must be armed"
fi

if jq -e '
  .authority_boundaries.validation_only == true and
  .authority_boundaries.live_mutation_allowed == false and
  .authority_boundaries.mutates_repositories == false and
  .authority_boundaries.creates_worktree == false and
  .authority_boundaries.creates_branch == false and
  .authority_boundaries.schedules_work == false and
  .authority_boundaries.executes_work == false and
  .authority_boundaries.approves_work == false and
  .authority_boundaries.provider_calls_allowed == false and
  .authority_boundaries.release_or_publish_allowed == false
' "$candidate" >/dev/null; then
  add_check "authority_boundaries" "passed" "candidate preserves validation-only non-authority boundaries"
else
  add_check "authority_boundaries" "blocked" "candidate attempts mutation, worktree creation, branch creation, scheduling, execution, approval, provider, release, or publish authority"
fi

status="ready"
can_start="true"
allowed_next_action="start_first_docs_only_live_pr_rehearsal"
if [[ -n "$first_failing_check" ]]; then
  status="blocked"
  can_start="false"
  allowed_next_action="repair_live_docs_worktree_preparation"
fi

mkdir -p "$(dirname "$out")"
checks_json="$tmpdir/checks.json"
jq -s '.' "$checks_file" > "$checks_json"

jq -n \
  --arg schema_version "ao.foundry.live-docs-worktree-prepare.v0.1" \
  --arg status "$status" \
  --argjson can_start_docs_only_pr_rehearsal "$can_start" \
  --arg allowed_next_action "$allowed_next_action" \
  --arg first_failing_check "$first_failing_check" \
  --arg candidate_path "$candidate" \
  --arg candidate_sha256 "$(sha256_file "$candidate")" \
  --arg approval_gate_path "$approval_gate" \
  --arg approval_gate_sha256 "$(sha256_file "$approval_gate")" \
  --arg candidate_id "$candidate_id" \
  --arg repo_id "$repo_id" \
  --arg repo_remote "$repo_remote" \
  --arg target_branch "$target_branch" \
  --arg change_class "$change_class" \
  --arg worktree_path "$worktree_path" \
  --arg branch "$branch" \
  --arg base_branch "$base_branch" \
  --argjson clean "$(jq '.worktree.clean // false' "$candidate")" \
  --argjson reused "$(jq '.worktree.reused // false' "$candidate")" \
  --argjson active_elsewhere "$(jq '.worktree.active_elsewhere // false' "$candidate")" \
  --argjson has_untracked_changes "$(jq '.worktree.has_untracked_changes // false' "$candidate")" \
  --argjson remote_branch_exists "$(jq '.worktree.remote_branch_exists // false' "$candidate")" \
  --argjson max_changed_files "$max_changed_files" \
  --slurpfile checks "$checks_json" \
  --slurpfile candidate_doc "$candidate" \
  '{
    schema_version:$schema_version,
    status:$status,
    can_start_docs_only_pr_rehearsal:$can_start_docs_only_pr_rehearsal,
    allowed_next_action:$allowed_next_action,
    first_failing_check:$first_failing_check,
    candidate_id:$candidate_id,
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
    change_plan:{
      changed_files:($candidate_doc[0].change_plan.changed_files // []),
      docs_only_path_allowlist:($candidate_doc[0].change_plan.docs_only_path_allowlist // []),
      forbidden_paths:($candidate_doc[0].change_plan.forbidden_paths // []),
      max_changed_files:$max_changed_files
    },
    source_hashes:[
      {name:"worktree_candidate", path:$candidate_path, schema_version:($candidate_doc[0].schema_version // ""), sha256:$candidate_sha256},
      {name:"approval_gate", path:$approval_gate_path, schema_version:"ao.foundry.live-docs-approval-gate.v0.1", sha256:$approval_gate_sha256}
    ],
    checks:$checks[0],
    blocking_next_actions:(if $status == "ready" then [] else [
      "Regenerate a fresh docs-only worktree candidate from synced main.",
      "Use a codex/live-docs-* branch under .foundry-local/worktrees.",
      "Keep changed files within the approved docs-only allowlist.",
      "Confirm approval gate and kill switch evidence before rehearsal."
    ] end),
    maintenance_suggestions:[
      "This gate validates preparation evidence only; it does not create a branch or worktree.",
      "Run the PR rehearsal only after the exact approval, Forge guard, AO2 patch packet, Sentinel, Promoter, rollback, and Command readback chain is green."
    ],
    authority_boundaries:{
      validation_only:true,
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
  }' > "$out"

if [[ "$json" -eq 1 ]]; then
  cat "$out"
else
  echo "live_docs_worktree_prepare=$status"
  echo "can_start_docs_only_pr_rehearsal=$can_start"
  echo "prepare=$out"
fi

if [[ "$status" != "ready" ]]; then
  exit 1
fi
