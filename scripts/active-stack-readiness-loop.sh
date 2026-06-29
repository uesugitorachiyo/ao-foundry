#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REGISTRY="examples/registry/local-ao-stack.foundry-registry.json"
LEDGER="examples/readiness/active-stack-readiness.ledger.json"
GITHUB_RUNS_REPORT="tmp/active-stack-github-runs-report.json"
ROLLUP_OUT="tmp/active-stack-production-readiness-rollup.json"
ROLLUP_MARKDOWN_OUT="tmp/active-stack-production-readiness-rollup.md"
RELEASE_CANDIDATE_LEDGER="examples/readiness/active-spine-release-candidate.ledger.json"
GOAL_RUN="examples/goals/ao-foundry-production-readiness.goal-run.json"
TASK="examples/tasks/ao-foundry-bootstrap.foundry-task.json"
OUT="tmp/active-stack-readiness-loop.json"

usage() {
  cat <<'EOF'
usage: scripts/active-stack-readiness-loop.sh [--registry <path>] [--ledger <path>] [--github-runs-report <path>] [--out <path>]

Runs the local-only active stack readiness loop:
  registry validate
  readiness snapshot README parity
  production readiness rollup when a GitHub runs report exists
  repo board
  release handoff
  pulse overnight start gate
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
    --github-runs-report)
      GITHUB_RUNS_REPORT="${2:?missing --github-runs-report value}"
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

check_readiness_rollup() {
  local report_path output
  if [[ "$GITHUB_RUNS_REPORT" = /* ]]; then
    report_path="$GITHUB_RUNS_REPORT"
  else
    report_path="$ROOT/$GITHUB_RUNS_REPORT"
  fi
  if [[ ! -f "$report_path" ]]; then
    add_check "readiness_rollup" "passed" "production readiness rollup skipped because GitHub runs report is absent" ""
    return
  fi
  if output="$(
    cd "$ROOT" &&
    go run ./cmd/foundry readiness rollup \
      --ledger "$LEDGER" \
      --github-runs-report "$GITHUB_RUNS_REPORT" \
      --out "$ROLLUP_OUT" \
      --markdown-out "$ROLLUP_MARKDOWN_OUT"
  )"; then
    add_check "readiness_rollup" "passed" "active-stack production readiness rollup is ready" "$output"
  else
    add_check "readiness_rollup" "failed" "active-stack production readiness rollup is ready" "$output"
  fi
}

run_check "registry_validate" \
  "active registry validates" \
  go run ./cmd/foundry registry validate --registry "$REGISTRY"
check_readiness_snapshot_parity
check_readiness_rollup
run_check "repo_board" \
  "active sibling portfolio is ready" \
  go run ./cmd/foundry repo board --registry "$REGISTRY" --json
run_check "release_handoff" \
  "active-spine release handoff validates candidate, promotion, notes, and manifest" \
  go run ./cmd/foundry release handoff \
    --candidate "$RELEASE_CANDIDATE_LEDGER" \
    --signed-smoke-summary examples/contract-fixtures/valid/foundry-signed-smoke-summary-v0.1.json \
    --promotion-out "$TMPDIR/release-promotion.fixture.json" \
    --notes-out "$TMPDIR/release-candidate.md" \
    --manifest-out "$TMPDIR/release-manifest.json"
run_check "atlas_readback_consumer" \
  "Atlas import and run-link readback remain fixture-only and observer-safe" \
  go run ./cmd/foundry atlas readback \
    --import examples/atlas/foundry-import.json \
    --run-link examples/atlas/run-link.completed.json \
    --out "$TMPDIR/atlas-readback.json"
run_check "atlas_status_surface" \
  "Atlas operator status summarizes registry, import, and readback without authority expansion" \
  go run ./cmd/foundry atlas status \
    --registry examples/registry/atlas-demo.foundry-registry.json \
    --import examples/atlas/foundry-import.json \
    --run-link examples/atlas/run-link.completed.json \
    --out "$TMPDIR/atlas-status.json"
run_check "pulse_overnight_start_gate" \
  "Pulse overnight/event-loop advancement is gated by Blueprint/Atlas preflight and one-slice lifecycle state" \
  go run ./cmd/foundry pulse overnight-start-gate \
    --intake-preflight examples/pulse-overnight-start-gate/ready.intake-preflight.json \
    --lifecycle examples/pulse-lifecycle/ready-to-start-next-slice.json \
    --out "$TMPDIR/pulse-overnight-start-gate.json"
run_check "worktree_isolation_proof" \
  "live-mutation candidates require clean isolated non-reused worktrees before authority can advance" \
  scripts/live-mutation-worktree-isolation-proof.sh \
    --candidate examples/live-mutation-worktree-isolation/clean-isolated.candidate.json \
    --out "$TMPDIR/worktree-isolation-proof.json"
run_check "live_mutation_rollback_rehearsal" \
  "live-mutation candidates require digest-bound rollback and quarantine rehearsal before authority can advance" \
  scripts/live-mutation-rollback-rehearsal.sh \
    --candidate examples/live-mutation-rollback/docs-only-rollback.candidate.json \
    --out "$TMPDIR/live-mutation-rollback-rehearsal.json"
run_check "governed_live_mutation_dry_run_chain" \
  "governed live-mutation readiness chain remains dry-run and evidence-bound" \
  scripts/governed-live-mutation-dry-run-chain.sh \
    --out tmp/governed-live-mutation-dry-run-chain
run_check "live_mutation_readiness_rollup" \
  "live-mutation readiness rollup summarizes request readiness without execution authority" \
  scripts/live-mutation-readiness-rollup.sh \
    --chain tmp/governed-live-mutation-dry-run-chain/summary.json \
    --out tmp/live-mutation-readiness-rollup.json
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
blocking_next_actions = []
maintenance_suggestions = []
if first_failing_check:
    blocking_next_actions.append(f"Fix first failing check: {first_failing_check}")
else:
    maintenance_suggestions.extend([
        "Keep the active stack registry limited to ao-foundry, ao-atlas, ao-forge, ao-command, ao2, ao2-control-plane, and ao-covenant.",
        "Refresh the readiness ledger after each merged release-readiness PR.",
    ])
payload = {
    "schema_version": "ao.foundry.active-stack-readiness-loop.v0.1",
    "status": status,
    "first_failing_check": first_failing_check,
    "checks": checks,
    "next_actions": blocking_next_actions,
    "blocking_next_actions": blocking_next_actions,
    "maintenance_suggestions": maintenance_suggestions,
}
out_path.write_text(json.dumps(payload, indent=2) + "\n")
PY

if [[ "$status" == "passed" ]]; then
  echo "active stack readiness loop: passed"
else
  echo "active stack readiness loop: failed first_failing_check=$first_failing_check" >&2
  exit 1
fi
