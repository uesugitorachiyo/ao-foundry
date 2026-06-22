# Release Checklist

1. Run `go test ./...`.
2. Run `jq empty docs/contracts/*.json examples/**/*.json`.
3. Run the public-safety scan documented in the CI workflow.
4. Run `go run ./cmd/foundry release dry-run --out tmp/release-manifest.json`.
5. Inspect `tmp/release-manifest.json` for expected public files and checksums.
6. Confirm no release step requires credentials, remote services, sibling
   repositories, tags, pushes, uploads, or publishing.

This checklist prepares a release candidate only. It does not publish artifacts.
