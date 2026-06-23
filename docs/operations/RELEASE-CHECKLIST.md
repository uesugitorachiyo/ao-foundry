# Release Checklist

1. Run `go test ./...`.
2. Run `jq empty docs/contracts/*.json examples/**/*.json`.
3. Run the public-safety scan documented in the CI workflow.
4. Run `go run ./cmd/foundry release dry-run --out tmp/release-manifest.json`.
5. Inspect `tmp/release-manifest.json` for expected public files and checksums.
6. Validate the active-spine release candidate:
   `go run ./cmd/foundry release candidate validate --ledger examples/readiness/active-spine-release-candidate.ledger.json`.
7. Validate the release-promotion fixture:
   `go run ./cmd/foundry release promotion validate --candidate examples/readiness/active-spine-release-candidate.ledger.json --signed-smoke-summary examples/contract-fixtures/valid/foundry-signed-smoke-summary-v0.1.json --out tmp/release-promotion.fixture.json`.
8. Generate the active-spine release-candidate notes:
   `go run ./cmd/foundry release candidate notes --ledger examples/readiness/active-spine-release-candidate.ledger.json --promotion examples/contract-fixtures/valid/foundry-release-promotion-v0.1.json --out docs/operations/ACTIVE-SPINE-2026-06-23-RELEASE-CANDIDATE.md`.
9. Run the consolidated Foundry release handoff:
   `go run ./cmd/foundry release handoff --candidate examples/readiness/active-spine-release-candidate.ledger.json --signed-smoke-summary examples/contract-fixtures/valid/foundry-signed-smoke-summary-v0.1.json --promotion-out tmp/release-promotion.handoff.json --notes-out tmp/release-candidate.handoff.md --manifest-out tmp/release-manifest.handoff.json`.
10. Collect active-stack GitHub run evidence and enforce sibling ledger freshness:
    `scripts/active-stack-github-runs-report.sh --out tmp/active-stack-github-runs-report.json`,
    then `go run ./cmd/foundry readiness evidence-check --ledger examples/readiness/active-stack-readiness.ledger.json --github-runs-report tmp/active-stack-github-runs-report.json`.
11. Generate the ledger refresh proposal from the active-stack GitHub run evidence:
    `go run ./cmd/foundry readiness ledger-refresh-proposal --ledger examples/readiness/active-stack-readiness.ledger.json --github-runs-report tmp/active-stack-github-runs-report.json --out tmp/active-stack-ledger-refresh-proposal.md`.
12. Fail on sibling ledger drift and apply current-repo refreshes:
    `go run ./cmd/foundry readiness ledger-refresh-proposal --ledger examples/readiness/active-stack-readiness.ledger.json --github-runs-report tmp/active-stack-github-runs-report.json --fail-on-non-current-update`,
    then `go run ./cmd/foundry readiness ledger-refresh-proposal --ledger examples/readiness/active-stack-readiness.ledger.json --github-runs-report tmp/active-stack-github-runs-report.json --apply --readme README.md`.
13. Confirm AO Forge can validate its release-candidate handoff fixture:
   `forge release-candidate validate --candidate examples/release-preview/release-candidate.v0.1.example.json`.
14. Confirm AO Covenant can emit the AO2-first policy-spine artifact:
   `covenant policy spine --json`; validate the captured output against
   `covenant.policy-spine-result.v1`.
15. Regenerate and compare the active-stack README snapshot:
    `go run ./cmd/foundry readiness snapshot --ledger examples/readiness/active-stack-readiness.ledger.json > tmp/readiness-snapshot.md`,
    then `diff -u tmp/readiness-snapshot.md <(sed -n '/<!-- foundry:active-stack-readiness:start -->/,/<!-- foundry:active-stack-readiness:end -->/p' README.md)`.
16. Before promotion, run the signed-smoke workflow with
    `workflow_dispatch signed_smoke=true` and require `release_safe=true`.
17. Confirm no release step requires credentials, remote services, sibling
   repositories, tags, pushes, uploads, or publishing.

This checklist prepares a release candidate only. It does not publish artifacts.
