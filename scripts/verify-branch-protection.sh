#!/usr/bin/env bash
set -euo pipefail

REPO="uesugitorachiyo/ao-foundry"
BRANCH="main"
REQUIRED_CHECKS=(
  "test (ubuntu-latest)"
  "test (macos-latest)"
  "test (windows-latest)"
)

usage() {
  cat <<'EOF'
usage: scripts/verify-branch-protection.sh [--repo <owner/name>] [--branch <branch>]

Verifies that the live GitHub branch protection for ao-foundry requires the
current CI matrix and forbids force pushes/deletions. This script is read-only.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --repo)
      REPO="${2:?missing --repo value}"
      shift 2
      ;;
    --branch)
      BRANCH="${2:?missing --branch value}"
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

require_tool() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required tool: $1" >&2
    exit 2
  fi
}

require_tool gh
require_tool jq

# Emits mode=limited when the token can read branch metadata but cannot read
# the full branch protection endpoint.
mode="full"
if protection="$(gh api "repos/$REPO/branches/$BRANCH/protection" 2>/tmp/ao-foundry-branch-protection.err)"; then
  :
else
  mode="limited"
  protection="$(gh api "repos/$REPO/branches/$BRANCH")"
fi

check_jq() {
  local name="$1"
  local filter="$2"
  if ! printf '%s' "$protection" | jq -e "$filter" >/dev/null; then
    echo "branch_protection=failed check=$name" >&2
    printf '%s\n' "$protection" | jq . >&2
    exit 1
  fi
}

if [[ "$mode" == "full" ]]; then
  check_jq "required_status_checks_strict" '.required_status_checks.strict == true'
  check_jq "enforce_admins" '.enforce_admins.enabled == true'
  check_jq "required_linear_history" '.required_linear_history.enabled == true'
  check_jq "allow_force_pushes_disabled" '.allow_force_pushes.enabled == false'
  check_jq "allow_deletions_disabled" '.allow_deletions.enabled == false'
  actual_checks="$(printf '%s' "$protection" | jq -r '.required_status_checks.contexts[]?' | sort)"
else
  check_jq "branch_protected" '.protected == true'
  check_jq "required_status_checks_enforced" '.protection.required_status_checks.enforcement_level == "everyone"'
  actual_checks="$(printf '%s' "$protection" | jq -r '.protection.required_status_checks.contexts[]?' | sort)"
fi

expected_checks="$(printf '%s\n' "${REQUIRED_CHECKS[@]}" | sort)"
if [[ "$actual_checks" != "$expected_checks" ]]; then
  echo "branch_protection=failed check=required_status_checks" >&2
  echo "expected:" >&2
  printf '%s\n' "$expected_checks" >&2
  echo "actual:" >&2
  printf '%s\n' "$actual_checks" >&2
  exit 1
fi

echo "branch_protection=passed"
echo "mode=$mode"
echo "repo=$REPO"
echo "branch=$BRANCH"
printf 'required_check=%s\n' "${REQUIRED_CHECKS[@]}"
