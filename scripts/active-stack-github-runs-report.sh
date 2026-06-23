#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="tmp/active-stack-github-runs-report.json"
BRANCH="main"

usage() {
  cat <<'EOF'
usage: scripts/active-stack-github-runs-report.sh [--branch <branch>] [--out <path>]

Writes a read-only active-stack GitHub Actions evidence report with the latest
ci.yml and production-readiness-ops.yml run for each active repository.
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

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

MANIFEST="$TMPDIR/manifest.tsv"
: > "$MANIFEST"

REPOS=(
  "uesugitorachiyo/ao-foundry"
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
python3 - "$MANIFEST" "$OUT_PATH" "$BRANCH" <<'PY'
import datetime
import json
import pathlib
import sys

manifest_path = pathlib.Path(sys.argv[1])
out_path = pathlib.Path(sys.argv[2])
branch = sys.argv[3]

repos = {}
status = "ready"
next_actions = []

for line in manifest_path.read_text().splitlines():
    if not line.strip():
        continue
    repo, kind, workflow, path = line.split("\t", 3)
    runs = json.loads(pathlib.Path(path).read_text())
    entry = repos.setdefault(repo, {"repository": repo})
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
        status = "blocked"
        next_actions.append(f"{repo}: {kind} is {run['status']}/{run['conclusion']}")

ordered_repos = [
    "uesugitorachiyo/ao-foundry",
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
    "generated_at": datetime.datetime.now(datetime.timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z"),
    "repositories": [repos[repo] for repo in ordered_repos],
    "next_actions": next_actions or ["Refresh the readiness ledger with the latest successful ci.yml and production-readiness-ops.yml run IDs."],
}
out_path.write_text(json.dumps(payload, indent=2) + "\n")
print(f"active_stack_github_runs_report={out_path}")
print(f"status={status}")

if status != "ready":
    sys.exit(1)
PY
