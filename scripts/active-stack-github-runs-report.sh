#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="tmp/active-stack-github-runs-report.json"
BRANCH="main"
LEDGER=""
ENFORCE_LEDGER=false
CURRENT_REPO="ao-foundry"

usage() {
  cat <<'EOF'
usage: scripts/active-stack-github-runs-report.sh [--branch <branch>] [--out <path>] [--ledger <path>] [--enforce-ledger] [--current-repo <repo-id>]

Writes a read-only active-stack GitHub Actions evidence report with the latest
ci.yml and production-readiness-ops.yml run for each active repository.

When --enforce-ledger is set, also runs:
  go run ./cmd/foundry readiness evidence-check --ledger <path> --github-runs-report <out>

The current repository is self-referential in ops because its latest
production-readiness-ops.yml run may be the workflow currently collecting this
report. CURRENT_REPO defaults to ao-foundry and is not allowed to block report
status while its own run is in progress; readiness evidence-check still skips
that repo by default and enforces sibling evidence.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --branch)
      BRANCH="${2:?missing --branch value}"
      shift 2
      ;;
    --out)
      OUT="${2:?missing --out value}"
      shift 2
      ;;
    --ledger)
      LEDGER="${2:?missing --ledger value}"
      shift 2
      ;;
    --enforce-ledger)
      ENFORCE_LEDGER=true
      shift
      ;;
    --current-repo)
      CURRENT_REPO="${2:?missing --current-repo value}"
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

if [[ "$OUT" = /* ]]; then
  OUT_PATH="$OUT"
else
  OUT_PATH="$ROOT/$OUT"
fi

if [[ -n "$LEDGER" && "$LEDGER" = /* ]]; then
  LEDGER_PATH="$LEDGER"
elif [[ -n "$LEDGER" ]]; then
  LEDGER_PATH="$ROOT/$LEDGER"
else
  LEDGER_PATH=""
fi

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

MANIFEST="$TMPDIR/manifest.tsv"
: > "$MANIFEST"

REPOS=(
  "uesugitorachiyo/ao-foundry"
  "uesugitorachiyo/ao-atlas"
  "uesugitorachiyo/ao-forge"
  "uesugitorachiyo/ao-command"
  "uesugitorachiyo/ao2"
  "uesugitorachiyo/ao2-control-plane"
  "uesugitorachiyo/ao-covenant"
)

fetch_run() {
  local repo="$1"
  local kind="$2"
  local workflow="$3"
  local safe_repo="${repo//\//__}"
  local path="$TMPDIR/${safe_repo}_${kind}.json"
  gh run list \
    --repo "$repo" \
    --workflow "$workflow" \
    --branch "$BRANCH" \
    --limit 1 \
    --json databaseId,status,conclusion,createdAt,headSha,displayTitle,url > "$path"
  printf '%s\t%s\t%s\t%s\n' "$repo" "$kind" "$workflow" "$path" >> "$MANIFEST"
}

for repo in "${REPOS[@]}"; do
  fetch_run "$repo" "latest_ci" "ci.yml"
  fetch_run "$repo" "latest_ops" "production-readiness-ops.yml"
done

mkdir -p "$(dirname "$OUT_PATH")"
python3 - "$MANIFEST" "$OUT_PATH" "$BRANCH" "$CURRENT_REPO" <<'PY'
import datetime
import json
import pathlib
import sys

manifest_path = pathlib.Path(sys.argv[1])
out_path = pathlib.Path(sys.argv[2])
branch = sys.argv[3]
current_repo = sys.argv[4]

repos = {}
status = "ready"
next_actions = []
current_repo_skipped = False

for line in manifest_path.read_text().splitlines():
    if not line.strip():
        continue
    repo, kind, workflow, path = line.split("\t", 3)
    runs = json.loads(pathlib.Path(path).read_text())
    entry = repos.setdefault(repo, {"repository": repo})
    repo_id = repo.rsplit("/", 1)[-1]
    if not runs:
        run = {
            "workflow": workflow,
            "status": "missing",
            "conclusion": "missing",
            "run_id": None,
            "url": None,
        }
    else:
        latest = runs[0]
        run = {
            "workflow": workflow,
            "status": latest.get("status") or "",
            "conclusion": latest.get("conclusion") or "",
            "run_id": str(latest.get("databaseId") or ""),
            "created_at": latest.get("createdAt") or "",
            "head_sha": latest.get("headSha") or "",
            "display_title": latest.get("displayTitle") or "",
            "url": latest.get("url") or "",
        }
    entry[kind] = run
    if run["status"] != "completed" or run["conclusion"] != "success":
        if repo_id == current_repo:
            current_repo_skipped = True
            continue
        status = "blocked"
        next_actions.append(f"{repo}: {kind} is {run['status']}/{run['conclusion']}")

ordered_repos = [
    "uesugitorachiyo/ao-foundry",
    "uesugitorachiyo/ao-atlas",
    "uesugitorachiyo/ao-forge",
    "uesugitorachiyo/ao-command",
    "uesugitorachiyo/ao2",
    "uesugitorachiyo/ao2-control-plane",
    "uesugitorachiyo/ao-covenant",
]

payload = {
    "schema_version": "ao.foundry.active-stack-github-runs-report.v0.1",
    "status": status,
    "branch": branch,
    "current_repo": current_repo,
    "current_repo_skipped": current_repo_skipped,
    "generated_at": datetime.datetime.now(datetime.timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z"),
    "repositories": [repos[repo] for repo in ordered_repos],
    "next_actions": next_actions or ["Refresh the readiness ledger with the latest successful ci.yml and production-readiness-ops.yml run IDs."],
}
out_path.write_text(json.dumps(payload, indent=2) + "\n")
print(f"active_stack_github_runs_report={out_path}")
print(f"status={status}")
if current_repo_skipped:
    print(f"current_repo_skipped={current_repo}")

if status != "ready":
    sys.exit(1)
PY

if [[ "$ENFORCE_LEDGER" == true ]]; then
  if [[ -z "$LEDGER_PATH" ]]; then
    echo "missing --ledger for --enforce-ledger" >&2
    exit 2
  fi
  cd "$ROOT"
  go run ./cmd/foundry readiness evidence-check --ledger "$LEDGER_PATH" --github-runs-report "$OUT_PATH"
fi
