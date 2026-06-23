#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REGISTRY="examples/registry/local-ao-stack.foundry-registry.json"
LEDGER="examples/readiness/active-stack-readiness.ledger.json"
RELEASE_CANDIDATE_LEDGER="examples/readiness/active-spine-release-candidate.ledger.json"
GOAL_RUN="examples/goals/ao-foundry-production-readiness.goal-run.json"
TASK="examples/tasks/ao-foundry-bootstrap.foundry-task.json"
OUT="tmp/active-stack-readiness-loop.json"

usage() {
  cat <<'EOF'
usage: scripts/active-stack-readiness-loop.sh [--registry <path>] [--ledger <path>] [--out <path>]

Runs the local-only active stack readiness loop:
  registry validate
  readiness snapshot README parity
  repo board
  release candidate validate
  loop preflight
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --registry)
      REGISTRY="${2:?missing --registry value}"
      shift 2
      ;;
    --ledger)
      LEDGER="${2:?missing --ledger value}"
      shift 2
      ;;
    --out)
      OUT="${2:?missing --out value}"
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

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

CHECKS_FILE="$TMPDIR/checks.jsonl"
: > "$CHECKS_FILE"
first_failing_check=""

json_escape() {
  python3 -c 'import json,sys; print(json.dumps(sys.stdin.read()))'
}

add_check() {
  local name="$1"
  local status="$2"
  local summary="$3"
  local output="$4"
  local escaped_summary escaped_output
  escaped_summary="$(printf '%s' "$summary" | json_escape)"
  escaped_output="$(printf '%s' "$output" | json_escape)"
  printf '{"name":%s,"status":%s,"summary":%s,"output":%s}\n' \
    "$(printf '%s' "$name" | json_escape)" \
    "$(printf '%s' "$status" | json_escape)" \
    "$escaped_summary" \
    "$escaped_output" >> "$CHECKS_FILE"
  if [[ "$status" != "passed" && -z "$first_failing_check" ]]; then
    first_failing_check="$name"
  fi
}

run_check() {
  local name="$1"
  local summary="$2"
  shift 2
  local output
  if output="$(cd "$ROOT" && "$@" 2>&1)"; then
    add_check "$name" "passed" "$summary" "$output"
  else
    add_check "$name" "failed" "$summary" "$output"
  fi
}

check_readiness_snapshot_parity() {
  local output snapshot readme_snapshot
  snapshot="$TMPDIR/readiness-snapshot.md"
  readme_snapshot="$TMPDIR/readme-readiness-snapshot.md"
  if output="$(
    cd "$ROOT" &&
    go run ./cmd/foundry readiness snapshot --ledger "$LEDGER" > "$snapshot" &&
    sed -n '/<!-- foundry:active-stack-readiness:start -->/,/<!-- foundry:active-stack-readiness:end -->/p' README.md > "$readme_snapshot" &&
    diff -u "$readme_snapshot" "$snapshot"
  )"; then
    add_check "readiness_snapshot_parity" "passed" "README active-stack snapshot matches the readiness ledger" "$output"
  else
    add_check "readiness_snapshot_parity" "failed" "README active-stack snapshot matches the readiness ledger" "$output"
  fi
}

run_check "registry_validate" \
  "active registry validates" \
  go run ./cmd/foundry registry validate --registry "$REGISTRY"
check_readiness_snapshot_parity
run_check "repo_board" \
  "active sibling portfolio is ready" \
  go run ./cmd/foundry repo board --registry "$REGISTRY" --json
run_check "release_candidate_validate" \
  "active-spine release candidate ledger validates" \
  go run ./cmd/foundry release candidate validate --ledger "$RELEASE_CANDIDATE_LEDGER"
run_check "loop_preflight" \
  "goal, registry, task, and production readiness preflight passes" \
  go run ./cmd/foundry loop preflight --goal-run "$GOAL_RUN" --registry "$REGISTRY" --task "$TASK"

status="passed"
if [[ -n "$first_failing_check" ]]; then
  status="failed"
fi

if [[ "$OUT" = /* ]]; then
  OUT_PATH="$OUT"
else
  OUT_PATH="$ROOT/$OUT"
fi

mkdir -p "$(dirname "$OUT_PATH")"
python3 - "$CHECKS_FILE" "$OUT_PATH" "$status" "$first_failing_check" <<'PY'
import json
import pathlib
import sys

checks_path = pathlib.Path(sys.argv[1])
out_path = pathlib.Path(sys.argv[2])
status = sys.argv[3]
first_failing_check = sys.argv[4] or None
checks = [json.loads(line) for line in checks_path.read_text().splitlines() if line.strip()]
next_actions = []
if first_failing_check:
    next_actions.append(f"Fix first failing check: {first_failing_check}")
else:
    next_actions.extend([
        "Keep the active stack registry limited to ao-foundry, ao-forge, ao-command, ao2, ao2-control-plane, and ao-covenant.",
        "Refresh the readiness ledger after each merged release-readiness PR.",
    ])
payload = {
    "schema_version": "ao.foundry.active-stack-readiness-loop.v0.1",
    "status": status,
    "first_failing_check": first_failing_check,
    "checks": checks,
    "next_actions": next_actions,
}
out_path.write_text(json.dumps(payload, indent=2) + "\n")
PY

if [[ "$status" == "passed" ]]; then
  echo "active stack readiness loop: passed"
else
  echo "active stack readiness loop: failed first_failing_check=$first_failing_check" >&2
  exit 1
fi
