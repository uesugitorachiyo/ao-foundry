# Active Spine Release Candidate: active-spine-2026-06-23

Status: ready

Release safe: true
Signed smoke pulse: pulse-signed-smoke
Signed smoke summary: ready
Pulse status: ready

## Active Spine

| Repository | Role | Status | Evidence |
| --- | --- | --- | --- |
| AO2 | execution-engine | ready | `npm run release:readiness:static`, `npm run verify`, main CI run 28053626014, Production Readiness Ops run 28054451606, PR #195 merged |
| AO2 Control Plane | evidence-observer | ready | license policy, `cargo fmt --all --check`, Python guard tests, `cargo test --workspace`, `cargo clippy --workspace --all-targets -- -D warnings`, `cargo deny check bans licenses sources`, `cargo audit --deny warnings`, `cargo build --release -p ao2-cp-server`, main CI run 28050638200, Production Readiness Ops run 28051422824, PR #67 merged |
| AO Foundry | operations-factory | ready | `go test ./...`, `go vet ./...`, `go build ./cmd/foundry ./cmd/ao`, `go run ./cmd/foundry contract fixtures validate`, `go run ./cmd/foundry release dry-run --out tmp/release-manifest.json`, `go run ./cmd/foundry release validate-manifest --manifest tmp/release-manifest.json`, main CI run 28054699582, Production Readiness Ops run 28054793631, PR #38 merged |

## Gates

| Gate | Status | Required before promotion | Evidence |
| --- | --- | --- | --- |
| signed_smoke_release_gate | manual_required | Yes | `docs/operations/SIGNED-SMOKE-RELEASE-GATE.md`, workflow_dispatch signed_smoke=true, freshness_summary.status=ready, signed_smoke_summary=ready, release_safe=true, `go run ./cmd/foundry release promotion validate --candidate examples/readiness/active-spine-release-candidate.ledger.json --signed-smoke-summary tmp/pulse-live/signed-smoke-summary.json --out tmp/release-promotion.live.json` |
| release_manifest_dry_run | ready | No | `go run ./cmd/foundry release dry-run --out tmp/release-manifest.json`, `go run ./cmd/foundry release validate-manifest --manifest tmp/release-manifest.json` |
| readiness_snapshot_parity | ready | No | `go run ./cmd/foundry readiness snapshot --ledger examples/readiness/active-stack-readiness.ledger.json` |

## Promotion Evidence

| Evidence | Status | Schema |
| --- | --- | --- |
| forge_live_attempt | passed | ao.foundry.forge-live-attempt.v0.1 |
| control_plane_readback | ready | ao.foundry.control-plane-readback.v0.1 |
| signed_smoke_ingest | ready | ao.foundry.signed-smoke-ingest.v0.1 |

## Tag plan

- Candidate tag: `active-spine-2026-06-23`
- Promote only after the signed-smoke summary is fresh for the promotion window.
- Attach release-promotion ledger to release notes
- Promote only the bound active-spine candidate
