# Release Checklist

1. Run `go test ./...`.
2. Run `jq empty docs/contracts/*.json examples/**/*.json`.
3. Run the public-safety scan documented in the CI workflow.
4. Run `go run ./cmd/foundry release dry-run --out tmp/release-manifest.json`.
5. Inspect `tmp/release-manifest.json` for expected public files and checksums.
6. Validate the active-spine release candidate:
   `go run ./cmd/foundry release candidate validate --ledger examples/readiness/active-spine-release-candidate.ledger.json`.
7. Verify the release candidate matches the active-stack readiness evidence:
   `go run ./cmd/foundry release candidate active-stack-parity --ledger examples/readiness/active-spine-release-candidate.ledger.json --readiness-ledger examples/readiness/active-stack-readiness.ledger.json`.
8. Validate the release-promotion fixture:
   `go run ./cmd/foundry release promotion validate --candidate examples/readiness/active-spine-release-candidate.ledger.json --signed-smoke-summary examples/contract-fixtures/valid/foundry-signed-smoke-summary-v0.1.json --out tmp/release-promotion.fixture.json`.
9. Generate the active-spine release-candidate notes:
   `go run ./cmd/foundry release candidate notes --ledger examples/readiness/active-spine-release-candidate.ledger.json --promotion examples/contract-fixtures/valid/foundry-release-promotion-v0.1.json --out docs/operations/ACTIVE-SPINE-2026-06-23-RELEASE-CANDIDATE.md`.
10. Run the consolidated Foundry release handoff:
   `go run ./cmd/foundry release handoff --candidate examples/readiness/active-spine-release-candidate.ledger.json --signed-smoke-summary examples/contract-fixtures/valid/foundry-signed-smoke-summary-v0.1.json --promotion-out tmp/release-promotion.handoff.json --notes-out tmp/release-candidate.handoff.md --manifest-out tmp/release-manifest.handoff.json`.
11. Collect active-stack GitHub run evidence and enforce sibling ledger freshness:
    `scripts/active-stack-github-runs-report.sh --out tmp/active-stack-github-runs-report.json`,
    then `go run ./cmd/foundry readiness evidence-check --ledger examples/readiness/active-stack-readiness.ledger.json --github-runs-report tmp/active-stack-github-runs-report.json`.
12. Generate the ledger refresh proposal from the active-stack GitHub run evidence:
    `go run ./cmd/foundry readiness ledger-refresh-proposal --ledger examples/readiness/active-stack-readiness.ledger.json --github-runs-report tmp/active-stack-github-runs-report.json --out tmp/active-stack-ledger-refresh-proposal.md`.
13. Fail on sibling ledger drift and apply current-repo refreshes:
    `go run ./cmd/foundry readiness ledger-refresh-proposal --ledger examples/readiness/active-stack-readiness.ledger.json --github-runs-report tmp/active-stack-github-runs-report.json --fail-on-non-current-update`,
    then `go run ./cmd/foundry readiness ledger-refresh-proposal --ledger examples/readiness/active-stack-readiness.ledger.json --github-runs-report tmp/active-stack-github-runs-report.json --apply --readme README.md`.
14. Confirm AO Forge can validate its release-candidate handoff fixture:
   `forge release-candidate validate --candidate examples/release-preview/release-candidate.v0.1.example.json`.
15. Confirm AO Covenant can emit the AO2-first policy-spine artifact:
   `covenant policy spine --json`; validate the captured output against
   `covenant.policy-spine-result.v1`.
16. Regenerate and compare the active-stack README snapshot:
    `go run ./cmd/foundry readiness snapshot --ledger examples/readiness/active-stack-readiness.ledger.json > tmp/readiness-snapshot.md`,
    then `diff -u tmp/readiness-snapshot.md <(sed -n '/<!-- foundry:active-stack-readiness:start -->/,/<!-- foundry:active-stack-readiness:end -->/p' README.md)`.
17. Confirm the readiness exit gate before starting new work:
    `scripts/active-stack-readiness-loop.sh --out tmp/active-stack-readiness-loop.json`.
    If the loop passes with no `blocking_next_actions`, stop autonomous
    readiness work. Live execution, release promotion, signed-smoke dispatch,
    or new implementation work requires explicit operator intent. See
    `docs/operations/READINESS-EXIT-GATE.md`.
18. Before promotion, run the signed-smoke workflow with
    `workflow_dispatch signed_smoke=true`, require `release_safe=true`, and
    download the `signed-smoke-release-evidence` artifact for the public-safe
    summary and release-promotion JSON. Retain the reviewed public-safe copy
    under `docs/evidence/pulse/20260623T213426Z-signed-smoke-release-gate`.
19. Run the final release handoff against the retained public-safe signed-smoke
    summary:
    `go run ./cmd/foundry release handoff --candidate examples/readiness/active-spine-release-candidate.ledger.json --signed-smoke-summary docs/evidence/pulse/20260623T213426Z-signed-smoke-release-gate/signed-smoke-summary.json --promotion-out tmp/release-promotion.final.json --notes-out docs/operations/ACTIVE-SPINE-2026-06-23-RELEASE-CANDIDATE.md --manifest-out tmp/release-manifest.final.json`.
20. Confirm no release step requires credentials, remote services, sibling
   repositories, tags, pushes, uploads, or publishing.

This checklist prepares a release candidate only. It does not publish artifacts.
