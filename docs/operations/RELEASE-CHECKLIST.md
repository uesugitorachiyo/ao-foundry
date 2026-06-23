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
8. Confirm AO Forge can validate its release-candidate handoff fixture:
   `forge release-candidate validate --candidate examples/release-preview/release-candidate.v0.1.example.json`.
9. Confirm AO Covenant can emit the AO2-first policy-spine artifact:
   `covenant policy spine --json`; validate the captured output against
   `covenant.policy-spine-result.v1`.
10. Regenerate and compare the active-stack README snapshot:
    `go run ./cmd/foundry readiness snapshot --ledger examples/readiness/active-stack-readiness.ledger.json > tmp/readiness-snapshot.md`,
    then `diff -u tmp/readiness-snapshot.md <(sed -n '/<!-- foundry:active-stack-readiness:start -->/,/<!-- foundry:active-stack-readiness:end -->/p' README.md)`.
11. Before promotion, run the signed-smoke workflow with
    `workflow_dispatch signed_smoke=true` and require `release_safe=true`.
12. Confirm no release step requires credentials, remote services, sibling
   repositories, tags, pushes, uploads, or publishing.

This checklist prepares a release candidate only. It does not publish artifacts.
