# AO Foundry

AO Foundry is the engineering operations factory above AO Forge. It does not
replace AO Forge. Foundry coordinates many repositories, goals, branches, CI
signals, release trains, evidence queues, and overnight advancement loops, then
delegates each individual governed implementation run to AO Forge.

AO Forge remains the trusted factory brain for one governed run. AO Foundry owns
the higher-level operating view: what work is queued, which repository is ready,
which evidence is waiting, which branch or release train is blocked, and what
the next safe delegated factory action should be.

## v0.1 Scope

This first slice provides:

- Public contracts for a Foundry registry, task, and run record.
- A design document for boundaries, architecture, state, operator journeys, and
  rollout.
- A one-shot operations runbook for using AO Forge to bootstrap Foundry safely.
- Example registry and task records for the local AO stack.
- A minimal Go CLI:
  - `foundry status --registry <path>`
  - `foundry registry validate --registry <path>`
  - `foundry task validate --task <path>`
  - `foundry next --registry <path> --task <path>`
  - `foundry readiness audit --registry <path> --task <path> [--out <path>]`
  - `foundry readiness snapshot --ledger <path> [--out <markdown>]`
  - `foundry readiness evidence-check --ledger <path> --github-runs-report <path>`
  - `foundry readiness ledger-refresh-proposal --ledger <path> --github-runs-report <path> --out <markdown>`
  - `foundry release handoff --candidate <path> --signed-smoke-summary <path> --promotion-out <path> --notes-out <markdown> --manifest-out <manifest.json>`
  - `foundry release candidate validate --ledger <path>`
  - `foundry release candidate notes --ledger <path> --promotion <path> --out <markdown>`
  - `foundry release promotion validate --candidate <path> --signed-smoke-summary <path> --out <path>`
  - `foundry goal validate --goal-run <path>`
  - `foundry goal readiness --goal-run <path> --registry <path> --task <path> [--out <path>]`
  - `foundry pulse run --out <dir>`
  - `foundry repo board --registry <path>`
  - `ao status`, `ao next`, `ao run`, `ao audit`, `ao demo` through `cmd/ao`

## Boundary Rule

Foundry can decide that a repository or task is ready for the next step, but the
governed execution step is delegated to AO Forge. Forge then applies Covenant
policy, invokes the execution adapter, preserves evidence, and returns a packet
that Foundry can record in a later run model.

Foundry v0.1 is local-first and public-safe. It does not push, tag, publish,
upload evidence, or mutate sibling repositories by default.

## Status

This public export is intentionally local-first:

- no provider API-key authentication paths;
- no bundled private runtime evidence or private coordination material;
- no remote publishing, release upload, tag, or push automation in normal
  verification;
- no credential storage or sibling-repository mutation authority.

## Quick Start

```sh
go test ./...
go run ./cmd/foundry status --registry examples/registry/local-ao-stack.foundry-registry.json
go run ./cmd/foundry registry validate --registry examples/registry/local-ao-stack.foundry-registry.json
go run ./cmd/foundry task validate --task examples/tasks/ao-foundry-bootstrap.foundry-task.json
go run ./cmd/foundry next --registry examples/registry/local-ao-stack.foundry-registry.json --task examples/tasks/ao-foundry-bootstrap.foundry-task.json
go run ./cmd/foundry readiness audit --registry examples/registry/local-ao-stack.foundry-registry.json --task examples/tasks/ao-foundry-bootstrap.foundry-task.json --out examples/readiness/ao-foundry-bootstrap.production-readiness-audit.json
go run ./cmd/foundry readiness snapshot --ledger examples/readiness/active-stack-readiness.ledger.json
go run ./cmd/foundry release candidate validate --ledger examples/readiness/active-spine-release-candidate.ledger.json
go run ./cmd/foundry release promotion validate --candidate examples/readiness/active-spine-release-candidate.ledger.json --signed-smoke-summary examples/contract-fixtures/valid/foundry-signed-smoke-summary-v0.1.json --out tmp/release-promotion.fixture.json
go run ./cmd/foundry release handoff --candidate examples/readiness/active-spine-release-candidate.ledger.json --signed-smoke-summary examples/contract-fixtures/valid/foundry-signed-smoke-summary-v0.1.json --promotion-out tmp/release-promotion.handoff.json --notes-out tmp/release-candidate.handoff.md --manifest-out tmp/release-manifest.handoff.json
go run ./cmd/foundry goal validate --goal-run examples/goals/ao-foundry-production-readiness.goal-run.json
go run ./cmd/foundry goal readiness --goal-run examples/goals/ao-foundry-production-readiness.goal-run.json --registry examples/registry/local-ao-stack.foundry-registry.json --task examples/tasks/ao-foundry-bootstrap.foundry-task.json --out examples/readiness/ao-foundry-production-readiness.goal-readiness-audit.json
go run ./cmd/foundry pulse run --out tmp/pulse
scripts/active-stack-readiness-loop.sh --out tmp/active-stack-readiness-loop.json
scripts/active-stack-github-runs-report.sh --out tmp/active-stack-github-runs-report.json
go run ./cmd/foundry readiness ledger-refresh-proposal --ledger examples/readiness/active-stack-readiness.ledger.json --github-runs-report tmp/active-stack-github-runs-report.json --out tmp/active-stack-ledger-refresh-proposal.md
go run ./cmd/foundry readiness ledger-refresh-proposal --ledger examples/readiness/active-stack-readiness.ledger.json --github-runs-report tmp/active-stack-github-runs-report.json --apply --readme README.md
scripts/verify-branch-protection.sh
go run ./cmd/ao status
go run ./cmd/ao run --out tmp/ao-pulse
```

The pulse command writes a local evidence bundle with readiness, GoalRun,
Forge-brief, Forge-packet, policy-gate, optional live Forge attempt,
control-plane readback, run, eval, trace, demo, release dry-run, competitive
audit, and a final `pulse-event.json` summary. It is a scheduler and evidence
loop only; live implementation remains delegated to AO Forge.

## Portfolio Board

When the sibling AO repositories are checked out next to AO Foundry, use the
read-only repo board to classify the portfolio and surface hygiene blockers:

```sh
go run ./cmd/foundry repo board --registry examples/registry/local-ao-stack.foundry-registry.json
```

The active sibling portfolio is AO Forge, AO2, ao2-control-plane, AO Covenant,
and AO Command. The board reports active-spine, supporting, and blocked-hygiene
entries for that live set. It exits non-zero when a registered sibling checkout
is dirty or otherwise blocked so cleanup happens before new strategy work.
Archived subscription-backed swarm, conductor, and deprecated operator/runtime
repositories are intentionally excluded from the active registry.

Use `scripts/active-stack-readiness-loop.sh` for the local active-stack gate. It
runs registry validation, README readiness snapshot parity, repo board, release
candidate validation, and loop preflight, then writes
`ao.foundry.active-stack-readiness-loop.v0.1` JSON with `first_failing_check`
and `next_actions`.

Use `scripts/active-stack-github-runs-report.sh` after each readiness PR merge to
collect the latest successful `ci.yml` and `production-readiness-ops.yml` run IDs
for the six active repositories. The script is read-only, uses `gh run list`,
and writes `ao.foundry.active-stack-github-runs-report.v0.1` JSON for ledger
refreshes. Add `--ledger examples/readiness/active-stack-readiness.ledger.json
--enforce-ledger` to fail when sibling repository run evidence is newer than the
ledger records; Foundry's own latest run is skipped by default to avoid a
self-referential main-branch gate. Use `foundry readiness ledger-refresh-proposal`
against the report to generate a markdown patch plan for ledger and README
snapshot refreshes. The production-readiness ops workflow uploads the latest
report as the `active-stack-github-runs-report` artifact. Use
`--apply --readme README.md` to apply report run IDs to the ledger and regenerate
the README snapshot. Ops also runs `--fail-on-non-current-update` so sibling
repository evidence drift blocks the workflow while current-repo self refreshes
remain actionable. If the only pending updates are Foundry's own CI and ops
runs from a readiness-evidence refresh PR, the proposal marks them
`ignored_current_refresh_loop` so the automation does not keep opening
ledger-only refresh PRs for its own bookkeeping.

## Verified Active Stack Snapshot

<!-- foundry:active-stack-readiness:start -->
Last local sweep: 2026-06-23.

| Repository | Current status | Verification evidence |
| --- | --- | --- |
| AO Foundry | Ready | `go test ./...`, `go vet ./...`, `go build ./cmd/foundry ./cmd/ao`, `go run ./cmd/foundry registry validate --registry examples/registry/local-ao-stack.foundry-registry.json`, `go run ./cmd/foundry task validate --task examples/tasks/ao-foundry-bootstrap.foundry-task.json`, `go run ./cmd/foundry repo board --registry examples/registry/local-ao-stack.foundry-registry.json`, scripts/active-stack-readiness-loop.sh --out tmp/active-stack-readiness-loop.json, scripts/active-stack-github-runs-report.sh --out tmp/active-stack-github-runs-report.json, `go run ./cmd/foundry release handoff --candidate examples/readiness/active-spine-release-candidate.ledger.json --signed-smoke-summary examples/contract-fixtures/valid/foundry-signed-smoke-summary-v0.1.json --promotion-out tmp/release-promotion.handoff.json --notes-out tmp/release-candidate.handoff.md --manifest-out tmp/release-manifest.handoff.json`, `go run ./cmd/foundry readiness evidence-check --ledger examples/readiness/active-stack-readiness.ledger.json --github-runs-report tmp/active-stack-github-runs-report.json`, scripts/verify-branch-protection.sh, .github/workflows/production-readiness-ops.yml, main CI run 28026152149, Production Readiness Ops run 28026243635, PR #16 merged, signed-smoke release promotion release_safe=true |
| AO Forge | Ready | license policy, license policy required in branch protection, GoalRun fixtures, `go test ./...`, `go vet ./...`, `go build`, production-readiness schemas, actionlint, Release Preview run 28011603944, Production Readiness Ops run 28017685064, PR #129 merged, main CI run `28017583706` |
| AO Command | Ready | AO2-first boundary audit, release dry-run chain, production readiness 100, 30/30 gates, license policy required in branch protection, Production Readiness Ops run 28018093029, PR #14 merged, main CI run `28018015778` |
| AO2 | Ready | `npm run release:readiness:static`, `npm run verify`, native AO2 runtime evidence tests, Production Readiness Ops run 28019892957, PR #192 merged, main CI run `28019192996` |
| AO2 Control Plane | Ready | license policy, `cargo fmt --all --check`, Python guard tests, `cargo test --workspace`, `cargo clippy --workspace --all-targets -- -D warnings`, `cargo deny check bans licenses sources`, `cargo audit --deny warnings`, `cargo build --release -p ao2-cp-server`, active stack handoff readback gate, Production Readiness Ops run 28016250935, PR #63 merged, main CI run `28016224096` |
| AO Covenant | Ready | `covenant policy spine --json`, covenant.policy-spine-result.v1, Release Readiness run 28006538855, branch protection verifier, Production Readiness Ops run 28020887257, PR #49 merged, main CI run `28020877698` |

Release handoff gates:

| Gate | Current status | Required before promotion | Evidence |
| --- | --- | --- | --- |
| foundry-release-candidate | Ready | Yes | `go run ./cmd/foundry release candidate validate --ledger examples/readiness/active-spine-release-candidate.ledger.json`, `go run ./cmd/foundry release promotion validate --candidate examples/readiness/active-spine-release-candidate.ledger.json --signed-smoke-summary examples/contract-fixtures/valid/foundry-signed-smoke-summary-v0.1.json --out tmp/release-promotion.fixture.json`, `go run ./cmd/foundry release handoff --candidate examples/readiness/active-spine-release-candidate.ledger.json --signed-smoke-summary examples/contract-fixtures/valid/foundry-signed-smoke-summary-v0.1.json --promotion-out tmp/release-promotion.handoff.json --notes-out tmp/release-candidate.handoff.md --manifest-out tmp/release-manifest.handoff.json`, `go run ./cmd/foundry release promotion validate --candidate examples/readiness/active-spine-release-candidate.ledger.json --signed-smoke-summary tmp/pulse-live/signed-smoke-summary.json --out tmp/release-promotion.live.json` |
| forge-release-candidate-handoff | Ready | Yes | `forge release-candidate validate --candidate examples/release-preview/release-candidate.v0.1.example.json`, ao-forge main CI run 28017583706, ao-forge Release Preview run 28011603944, ao-forge Production Readiness Ops run 28017685064 |
| covenant-policy-spine | Ready | Yes | `covenant policy spine --json`, covenant.policy-spine-result.v1, ao-covenant main CI run 28020877698, ao-covenant Release Readiness run 28006538855, ao-covenant Production Readiness Ops run 28020887257 |
| signed-smoke-release-gate | Manual Required | Yes | `docs/operations/SIGNED-SMOKE-RELEASE-GATE.md`, workflow_dispatch signed_smoke=true, freshness_summary.status=ready, signed_smoke_summary=ready, release_safe=true, tmp/release-promotion.live.json |

The machine-readable source for this snapshot is
[`examples/readiness/active-stack-readiness.ledger.json`](examples/readiness/active-stack-readiness.ledger.json).
The AO2 active-spine release candidate ledger is
[`examples/readiness/active-spine-release-candidate.ledger.json`](examples/readiness/active-spine-release-candidate.ledger.json).
<!-- foundry:active-stack-readiness:end -->

No active readiness path depends on `ao-operator`, `ao-runtime`,
`ao-control-plane`, `ao-conductor`, `agy-swarms`, or `codex-cron`.

## Public Documents

- [AO Foundry v0.1 Design](docs/design/AO-FOUNDRY-V0.1.md)
- [One-Shot Factory Run](docs/operations/ONE-SHOT-FACTORY-RUN.md)
- [AO2 Pulse Event Loop](docs/operations/AO2-PULSE-EVENT-LOOP.md)
- [Branch protection](docs/operations/BRANCH-PROTECTION.md)
- [Signed-smoke release gate](docs/operations/SIGNED-SMOKE-RELEASE-GATE.md)
- [Production Readiness SDD](docs/sdd/AO-FOUNDRY-PRODUCTION-READINESS-SDD.md)
- [Pulse Golden Loop SDD](docs/sdd/AO-FOUNDRY-PULSE-GOLDEN-LOOP-SDD.md)
- [Pulse Production Adapters SDD](docs/sdd/AO-FOUNDRY-PULSE-PRODUCTION-ADAPTERS-SDD.md)
- [Registry schema](docs/contracts/foundry-registry-v0.1.schema.json)
- [Task schema](docs/contracts/foundry-task-v0.1.schema.json)
- [Run schema](docs/contracts/foundry-run-v0.1.schema.json)
- [Production readiness audit schema](docs/contracts/foundry-production-readiness-audit-v0.1.schema.json)
- [Active stack readiness schema](docs/contracts/foundry-active-stack-readiness-v0.1.schema.json)
- [Release candidate schema](docs/contracts/foundry-release-candidate-v0.1.schema.json)
- [Release promotion schema](docs/contracts/foundry-release-promotion-v0.1.schema.json)
- [GoalRun schema](docs/contracts/foundry-goal-run-v0.1.schema.json)
- [Goal readiness audit schema](docs/contracts/foundry-goal-readiness-audit-v0.1.schema.json)
- [Pulse event schema](docs/contracts/foundry-pulse-event-v0.1.schema.json)
- [Forge live attempt schema](docs/contracts/foundry-forge-live-attempt-v0.1.schema.json)
- [Control-plane readback schema](docs/contracts/foundry-control-plane-readback-v0.1.schema.json)

## Security

AO Foundry treats public fixtures and evidence as publishable artifacts. Public
files should not include credentials, local absolute paths, non-public
operational notes, private server logs, or raw control-plane bearer tokens.

Report security issues through GitHub Security Advisories when available. See
[SECURITY.md](SECURITY.md) for the supported reporting path and local safety
model.

## License

AO Foundry is licensed under the Apache License, Version 2.0. See
[LICENSE](LICENSE) and [NOTICE](NOTICE).
