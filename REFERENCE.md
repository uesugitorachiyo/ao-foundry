# AO Foundry Reference

AO Foundry coordinates engineering work across repositories and selects the next safe portfolio action. It tracks registries, tasks, readiness signals, branches, CI state, and run evidence, then delegates one bounded run at a time. Use Foundry when several repositories or queued tasks must be prioritized and advanced without combining portfolio decisions with execution.

## How it fits in AO

- **Primary responsibility:** Portfolio coordination and safe-next-work selection.
- **Inputs:** Blueprint-authorized bounded work, Atlas imports, repository and CI state, task records, and completed-run evidence.
- **Outputs:** Portfolio status, readiness audits, next-work decisions, run records, and bounded Forge handoffs.
- **Upstream:** AO Blueprint directly for bounded work, or AO Atlas when workgraph and context compilation is required.
- **Downstream:** AO Forge for one governed run, plus AO Mission and AO Command for lifecycle and operator readback.

See the
[AO Architecture guide](https://github.com/uesugitorachiyo/ao-architecture)
and the
[AO Foundry component page](https://github.com/uesugitorachiyo/ao-architecture/blob/main/components/ao-foundry.md)
for the cross-repository flow.

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
  - `foundry readiness rollup --ledger <path> --github-runs-report <path> --out <json> --markdown-out <markdown>`
  - `foundry release handoff --candidate <path> --signed-smoke-summary <path> --promotion-out <path> --notes-out <markdown> --manifest-out <manifest.json>`
  - `foundry release candidate validate --ledger <path>`
  - `foundry release candidate active-stack-parity --ledger <path> --readiness-ledger <path>`
  - `foundry release candidate notes --ledger <path> --promotion <path> --out <markdown>`
  - `foundry release promotion validate --candidate <path> --signed-smoke-summary <path> --out <path>`
  - `foundry goal validate --goal-run <path>`
  - `foundry goal readiness --goal-run <path> --registry <path> --task <path> [--out <path>]`
  - `foundry pulse intake-preflight --blueprint-authorization <path> [--requires-atlas --atlas-blueprint-import <path> --atlas-import <path> --atlas-status <path>] [--out <path>]`
  - `foundry pulse lifecycle inspect --state <pulse-pr-lifecycle.json> [--json]`
  - `foundry pulse overnight-start-gate --intake-preflight <path> --lifecycle <path> --out <path> [--start-implementation] [--json]`
  - `foundry pulse event-loop-policy --class-gate <path> --promotion-state <path> --ci <path> --repo-state <path> --evidence-freshness <path> --sentinel <path> --promoter <path> --rollback <path> --branch-cleanup <path> --scope <path> --out <path> [--json]`
  - `foundry ao-mission smoke --route <route-readback.json> --snapshot <governance-snapshot.json> --out <smoke.json>`
  - `foundry ao-mission final-rollup-smoke --mission-final-rollup <mission-rollup.json> --foundry-final-rollup <foundry-rollup.json> --out <smoke.json>`
  - `foundry ao-mission readiness-ledger --final-rollup-smoke <smoke.json> --out <ledger.json>`
  - `foundry ao-mission e2e-smoke --route <route-readback.json> --snapshot <governance-snapshot.json> --mission-final-rollup <mission-rollup.json> --foundry-final-rollup <foundry-rollup.json> --atlas-metadata <metadata.json> [--artifact-manifest <manifest.json>] [--scheduler-recovery <readback.json>] [--ledger-compaction <readback.json>] [--timeline-compaction <readback.json>] [--mission-archive-validation <readback.json>] [--gateway-readiness-rollup <readback.json>] --out <smoke.json>`
  - `scripts/ao-mission-atlas-foundry-e2e-smoke.sh [tmp/output-dir]`
  - `foundry class-gate evaluate --atlas <path> --covenant <path> --sentinel <path> --promoter <path> --rollback <path> --command <path> --ci <path> [--test-only-success <path>] [--multi-repo-plan <path>] --out <path>`
  - `scripts/blueprint-atlas-pulse-e2e-dry-run.sh --out <public-safe-relative-dir>`
  - `scripts/complex-refactor-workgraph-rehearsal.sh --out <public-safe-relative-dir>`
  - `scripts/overnight-rehearsal-runner.sh --out <public-safe-relative-dir>`
  - `scripts/fresh-overnight-rehearsal-artifact.sh --out <public-safe-relative-dir>`
  - `scripts/atlas-stress-readiness.sh --out <public-safe-relative-dir>`
  - `scripts/live-mutation-worktree-isolation-proof.sh --candidate <candidate.json> --out <proof.json>`
  - `scripts/live-mutation-rollback-rehearsal.sh --candidate <candidate.json> --out <rehearsal.json>`
  - `scripts/governed-live-mutation-dry-run-chain.sh --out <public-safe-relative-dir>`
  - `scripts/governed-live-mutation-dry-run-chain.sh --mutation-class low_risk_code --out <public-safe-relative-dir>`
  - `scripts/low-risk-code-live-rehearsal-gate.sh --chain <summary.json> --out <gate.json>`
  - `scripts/live-mutation-readiness-rollup.sh --chain <summary.json> --out <rollup.json>`
  - `scripts/live-docs-approval-gate.sh --request <request.json> --ticket <ticket.json> --out <gate.json>`
  - `scripts/live-docs-worktree-prepare.sh --candidate <candidate.json> --approval-gate <gate.json> --out <prepare.json>`
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
go run ./cmd/foundry atlas import validate --import examples/atlas/foundry-import.json
go run ./cmd/foundry atlas readback --import examples/atlas/foundry-import.json --run-link examples/atlas/run-link.completed.json --out tmp/atlas-readback.json
go run ./cmd/foundry atlas status --registry examples/registry/atlas-demo.foundry-registry.json --import examples/atlas/foundry-import.json --run-link examples/atlas/run-link.completed.json
go run ./cmd/foundry task validate --task examples/tasks/ao-foundry-bootstrap.foundry-task.json
go run ./cmd/foundry next --registry examples/registry/local-ao-stack.foundry-registry.json --task examples/tasks/ao-foundry-bootstrap.foundry-task.json
go run ./cmd/foundry readiness audit --registry examples/registry/local-ao-stack.foundry-registry.json --task examples/tasks/ao-foundry-bootstrap.foundry-task.json --out examples/readiness/ao-foundry-bootstrap.production-readiness-audit.json
go run ./cmd/foundry readiness snapshot --ledger examples/readiness/active-stack-readiness.ledger.json
go run ./cmd/foundry release candidate validate --ledger examples/readiness/active-spine-release-candidate.ledger.json
go run ./cmd/foundry release candidate active-stack-parity --ledger examples/readiness/active-spine-release-candidate.ledger.json --readiness-ledger examples/readiness/active-stack-readiness.ledger.json
go run ./cmd/foundry release promotion validate --candidate examples/readiness/active-spine-release-candidate.ledger.json --signed-smoke-summary examples/contract-fixtures/valid/foundry-signed-smoke-summary-v0.1.json --out tmp/release-promotion.fixture.json
go run ./cmd/foundry release handoff --candidate examples/readiness/active-spine-release-candidate.ledger.json --signed-smoke-summary examples/contract-fixtures/valid/foundry-signed-smoke-summary-v0.1.json --promotion-out tmp/release-promotion.handoff.json --notes-out tmp/release-candidate.handoff.md --manifest-out tmp/release-manifest.handoff.json
go run ./cmd/foundry goal validate --goal-run examples/goals/ao-foundry-production-readiness.goal-run.json
go run ./cmd/foundry goal readiness --goal-run examples/goals/ao-foundry-production-readiness.goal-run.json --registry examples/registry/local-ao-stack.foundry-registry.json --task examples/tasks/ao-foundry-bootstrap.foundry-task.json --out examples/readiness/ao-foundry-production-readiness.goal-readiness-audit.json
go run ./cmd/foundry pulse intake-preflight --blueprint-authorization examples/pulse-intake/blueprint-authorization.ready.json --requires-atlas --atlas-blueprint-import examples/atlas/blueprint-import.low-risk-code.json --atlas-import examples/atlas/foundry-import.json --atlas-status examples/contract-fixtures/valid/foundry-atlas-status-v0.1.json --out tmp/pulse-intake-preflight.json
go run ./cmd/foundry pulse lifecycle inspect --state examples/pulse-lifecycle/ready-to-start-next-slice.json --json
go run ./cmd/foundry pulse overnight-start-gate --intake-preflight examples/pulse-overnight-start-gate/ready.intake-preflight.json --lifecycle examples/pulse-lifecycle/ready-to-start-next-slice.json --out tmp/pulse-overnight-start-gate.json
go run ./cmd/foundry class-gate evaluate --atlas examples/class-gate/atlas-classification.docs-multi.json --covenant examples/class-gate/covenant-ticket.docs-multi.json --sentinel examples/class-gate/sentinel.no-hold.docs-multi.json --promoter examples/class-gate/promoter.ready.docs-multi.json --rollback examples/class-gate/rollback.passed.docs-multi.json --command examples/class-gate/command-readback.docs-multi.json --ci examples/class-gate/ci.passed.docs-multi.json --out tmp/class-gate.json
scripts/blueprint-atlas-pulse-e2e-dry-run.sh --out docs/evidence/pulse/blueprint-atlas-pulse-e2e-local
scripts/complex-refactor-workgraph-rehearsal.sh --out docs/evidence/pulse/complex-refactor-workgraph-rehearsal-local
scripts/overnight-rehearsal-runner.sh --out docs/evidence/pulse/overnight-rehearsal-runner-local
scripts/atlas-stress-readiness.sh --out docs/evidence/pulse/atlas-stress-readiness-local
scripts/active-stack-readiness-loop.sh --out tmp/active-stack-readiness-loop.json
scripts/active-stack-github-runs-report.sh --out tmp/active-stack-github-runs-report.json
go run ./cmd/foundry readiness ledger-refresh-proposal --ledger examples/readiness/active-stack-readiness.ledger.json --github-runs-report tmp/active-stack-github-runs-report.json --out tmp/active-stack-ledger-refresh-proposal.md
go run ./cmd/foundry readiness rollup --ledger examples/readiness/active-stack-readiness.ledger.json --github-runs-report tmp/active-stack-github-runs-report.json --out tmp/active-stack-production-readiness-rollup.json --markdown-out tmp/active-stack-production-readiness-rollup.md
go run ./cmd/foundry readiness ledger-refresh-proposal --ledger examples/readiness/active-stack-readiness.ledger.json --github-runs-report tmp/active-stack-github-runs-report.json --apply --readme README.md
scripts/verify-branch-protection.sh
go run ./cmd/ao status
go run ./cmd/ao run --out tmp/ao-pulse
```

The pulse command first enforces a Pulse overnight start gate and writes
`pulse-runner-start-decision.json`. Only a ready gate with digest-bound
Blueprint/Atlas/preflight/lifecycle evidence may continue to bundle generation.
Blocked or failed gates stop before implementation evidence is produced.

When the gate is ready, the command writes a local evidence bundle with
readiness, GoalRun, Forge-brief, Forge-packet, policy-gate, optional live Forge
attempt, control-plane readback, run, evaluation, trace, demo, release dry-run,
competitive audit, and a final `pulse-event.json` summary. It is a scheduler
and evidence loop only; live implementation remains delegated to AO Forge.

`foundry pulse intake-preflight` is the Blueprint/Atlas-aware intake gate before
future automated scheduling. It emits
`ao.foundry.pulse-intake-preflight.v0.1`, fails closed when Blueprint
authorization is missing or blocked, and for oversized or live-mutation work
requires a ready Atlas Blueprint import before accepting Atlas Foundry import
and Foundry Atlas status/readback evidence. The command is fixture/local only:
it does not schedule, execute, approve, upload, publish, call providers, or
mutate sibling repositories.

`foundry pulse lifecycle inspect` is the one-slice PR lifecycle gate before
starting another automated slice. It reads
`ao.foundry.pulse-pr-lifecycle.v0.1` state and fails closed when a branch, PR,
check, merge cleanup, dirty worktree, or main-sync condition still blocks the
next slice. It is inspection-only and does not create branches, push, merge, or
delete anything.

`foundry pulse overnight-start-gate` composes the Blueprint/Atlas intake
preflight and one-slice PR lifecycle state into the required precondition before
autonomous overnight/event-loop advancement. It emits
`ao.foundry.pulse-overnight-start-gate.v0.1`, requires digest-bound source
evidence, fails closed on failed preflight, pending/failing PRs, incomplete
cleanup, unsynced main, dirty worktrees, and stale evidence digests, and returns
a clean blocked result for Blueprint clarification when implementation is not
being started. The gate is read-only decision evidence; it does not start the
loop, schedule, execute, approve, publish, call providers, or mutate
repositories.

`foundry class-gate evaluate` composes one Atlas mutation-class classification,
one Covenant class ticket, Sentinel no-hold evidence, Promoter readiness,
rollback proof, AO Command readback, and CI evidence. It emits
`ao.foundry.mutation-class-gate.v0.1` with `safe_to_request` and
`safe_to_execute` for exactly one mutation class, while all other classes stay
listed in `denied_classes`. Missing or mismatched evidence blocks the gate. The
checked-in fixtures cover `docs_only_multi_file` and `test_only` readiness.
The low-risk-code dry-run path also requires explicit `test_only_success`
evidence. Without that evidence Foundry keeps `safe_to_request=false` and
`safe_to_execute=false` even if the generic class evidence is otherwise ready.
With checked test-only live rehearsal evidence, Foundry may report
`safe_to_request=true` for a low-risk-code dry-run design while still keeping
`safe_to_execute=false`. The low-risk-code gate also emits
`class_boundary_checks` readback for the Atlas classification-only boundary,
exact-scope Covenant ticket flags, Sentinel no-hold verdict, Promoter class
boundary, rollback proof, read-only Command state, CI readiness, and
test-only live evidence. If any of those consumed artifacts broadens scope,
loses class binding, claims mutation authority, omits rollback/CI evidence, or
stops being read-only, Foundry fails closed. The low-risk-code gate also emits a
`denial_audit` readback listing the missing live policy promotion, rollback
proof, Sentinel clear verdict, Promoter promotion, Command readback, and PR CI
evidence, with `exact_next_action=build_low_risk_code_promotion_prerequisites`.
The gate does not schedule, execute, approve, publish, call providers, or
mutate repositories.
## Portfolio Board

When the sibling AO repositories are checked out next to AO Foundry, use the
read-only repo board to classify the portfolio and surface hygiene blockers:

```sh
go run ./cmd/foundry repo board --registry examples/registry/local-ao-stack.foundry-registry.json
```

The active sibling portfolio is AO Atlas, AO Forge, AO2, ao2-control-plane, AO
Covenant, and AO Command, with AO Foundry as the local orchestration repo. The
board reports active-spine, supporting, and blocked-hygiene entries for that
live set. It exits non-zero when a registered sibling checkout is dirty or
otherwise blocked so cleanup happens before new strategy work.
Archived subscription-backed swarm, conductor, and deprecated operator/runtime
repositories are intentionally excluded from the active registry.

Use `scripts/active-stack-readiness-loop.sh` for the local active-stack gate. It
runs registry validation, README readiness snapshot parity, repo board, release
candidate validation, and loop preflight, then writes
`ao.foundry.active-stack-readiness-loop.v0.1` JSON with `first_failing_check`
plus separate `blocking_next_actions` and `maintenance_suggestions`.

AO Atlas integration is fixture-only. Foundry’s first Atlas consumer artifact is
`ao.atlas.foundry-import.v0.1`, validated with:

```sh
go run ./cmd/foundry atlas import validate --import examples/atlas/foundry-import.json
go run ./cmd/foundry atlas readback --import examples/atlas/foundry-import.json --run-link examples/atlas/run-link.completed.json --out tmp/atlas-readback.json
go run ./cmd/foundry atlas status --registry examples/registry/atlas-demo.foundry-registry.json --import examples/atlas/foundry-import.json --run-link examples/atlas/run-link.completed.json
```

The validator confirms the packet is readback material only: no scheduling,
execution, approval, release mutation, provider calls, or sibling repo mutation.
It also requires each imported task to carry Atlas authority metadata
(`mutation_class`, `write_scope`, `rollback_scope`, `required_gates`,
`required_evidence`, and `authority_boundary`) before Foundry will accept the
packet.
The readback command links the Atlas import packet to a completed
`ao.atlas.run-link.v0.1` and emits `ao.foundry.atlas-readback.v0.1` with the
same observer-only authority boundary. The status command combines registry,
import, and readback checks into one operator-facing `ao.foundry.atlas-status.v0.1`
surface without granting scheduling, execution, approval, provider, release, or
sibling-repository mutation authority.

The readiness exit gate is stop-oriented. When goal readiness and competitive
readiness are 100/100 and the active-stack loop passes with no
`blocking_next_actions`, autonomous readiness work stops; live execution,
release promotion, signed-smoke dispatch, or new implementation work requires
explicit operator intent. See
[`docs/operations/READINESS-EXIT-GATE.md`](docs/operations/READINESS-EXIT-GATE.md).

Use `scripts/active-stack-github-runs-report.sh` after sibling readiness PR
merges to collect the latest successful `ci.yml` and
`production-readiness-ops.yml` run IDs for the seven active repositories. The
script is read-only, uses `gh run list`, and writes
`ao.foundry.active-stack-github-runs-report.v0.1` JSON for ledger refreshes.
Add `--ledger examples/readiness/active-stack-readiness.ledger.json
--enforce-ledger` to fail when sibling repository run evidence is newer than the
ledger records; Foundry's own latest run is skipped by default to avoid a
self-referential main-branch gate. Use `foundry readiness ledger-refresh-proposal`
against the report to generate a markdown patch plan for sibling ledger and
README snapshot refreshes. The production-readiness ops workflow uploads the
latest report as the `active-stack-github-runs-report` artifact. Use
`--apply --readme README.md` to apply sibling report run IDs to the ledger and
regenerate the README snapshot. Ops also runs `--fail-on-non-current-update` so
sibling repository evidence drift blocks the workflow while current-repo mutable
self evidence is ignored. Current-repo rows are marked
`ignored_current_self_evidence`, or `ignored_current_refresh_loop` for historical
readiness-evidence refresh PRs, so automation does not keep opening ledger-only
refresh PRs for its own bookkeeping.

Use `foundry readiness rollup` after the GitHub runs report exists to produce
the final `ao.foundry.active-stack-production-readiness-rollup.v0.1` JSON and
markdown summary. The rollup fails on sibling evidence drift, missing active
repositories, failed or in-progress sibling runs, blocked release-handoff gates,
and stale non-current run updates. It records the signed-smoke release gate as a
`promotion_manual_gate`; that manual gate does not block readiness, but it
remains required before promotion.


<!--
CLI compatibility assertions:
claim_level=bounded_governed_rsi decision=allowed
claim_level=full_autonomous_self_mutating_rsi decision=denied
minimum improvement: 5 percentage points
mutates_repositories=false
-->

<details>
<summary>Generated active-stack compatibility snapshot</summary>

## Verified Active Stack Snapshot

<!-- foundry:active-stack-readiness:start -->
Last local sweep: 2026-06-23.

| Repository | Current status | Verification evidence |
| --- | --- | --- |
| AO Foundry | Ready | `go test ./...`, `go vet ./...`, `go build ./cmd/foundry ./cmd/ao`, `go run ./cmd/foundry registry validate --registry examples/registry/local-ao-stack.foundry-registry.json`, `go run ./cmd/foundry task validate --task examples/tasks/ao-foundry-bootstrap.foundry-task.json`, `go run ./cmd/foundry repo board --registry examples/registry/local-ao-stack.foundry-registry.json`, scripts/active-stack-readiness-loop.sh --out tmp/active-stack-readiness-loop.json, scripts/active-stack-github-runs-report.sh --out tmp/active-stack-github-runs-report.json, `go run ./cmd/foundry release handoff --candidate examples/readiness/active-spine-release-candidate.ledger.json --signed-smoke-summary examples/contract-fixtures/valid/foundry-signed-smoke-summary-v0.1.json --promotion-out tmp/release-promotion.handoff.json --notes-out tmp/release-candidate.handoff.md --manifest-out tmp/release-manifest.handoff.json`, `go run ./cmd/foundry readiness evidence-check --ledger examples/readiness/active-stack-readiness.ledger.json --github-runs-report tmp/active-stack-github-runs-report.json`, scripts/verify-branch-protection.sh, .github/workflows/production-readiness-ops.yml, signed-smoke release promotion release_safe=true |
| AO Atlas | Ready | `go test ./...`, `go vet ./...`, `go build ./cmd/atlas`, scripts/production-readiness.sh, scripts/atlas-foundry-roundtrip-smoke.sh, `go run ./cmd/foundry atlas status --registry examples/registry/atlas-demo.foundry-registry.json --import examples/atlas/foundry-import.json --run-link examples/atlas/run-link.completed.json`, ao.foundry.atlas-status.v0.1, schedules_work=false, executes_work=false, approves_work=false, Production Readiness Ops run 30956775480, PR #760 merged, main CI run `30956775511` |
| AO Forge | Ready | license policy, license policy required in branch protection, GoalRun fixtures, `go test ./...`, `go vet ./...`, `go build`, production-readiness schemas, actionlint, Release Preview run 28066645263, Production Readiness Ops run 31172789285, PR #135 merged, main CI run `31017677133` |
| AO Command | Ready | AO2-first boundary audit, release dry-run chain, ao-command rsi health --arena-gate ../ao-arena/tmp/arena-promotion-gate.json --crucible-gate ../ao-crucible/tmp/crucible-hardening-gate.json --sentinel-verdict ../ao-sentinel/tmp/sentinel-verdict.json --promoter-gate ../ao-promoter/tmp/promotion-gate.json --json, rsi_mode=governed_fixture_local, mutates_repositories=false, production readiness 100, 36/36 gates, license policy required in branch protection, Production Readiness Ops run 31173234616, PR #36 merged, ao-command atlas status --status ../ao-foundry/examples/contract-fixtures/valid/foundry-atlas-status-v0.1.json, main CI run `31027012970` |
| AO2 | Ready | `npm run release:readiness:static`, `npm run verify`, native AO2 runtime evidence tests, Production Readiness Ops run 31174328265, PR #606 merged, main CI run `31187008934` |
| AO2 Control Plane | Ready | license policy, `cargo fmt --all --check`, Python guard tests, `cargo test --workspace`, `cargo clippy --workspace --all-targets -- -D warnings`, `cargo deny check bans licenses sources`, `cargo audit --deny warnings`, `cargo build --release -p ao2-cp-server`, active stack handoff readback gate, Production Readiness Ops run 31172929879, PR #133 merged, main CI run `30968880090` |
| AO Covenant | Ready | `covenant policy spine --json`, covenant.policy-spine-result.v1, Release Readiness run 28067529569, branch protection verifier, Production Readiness Ops run 31173669247, PR #59 merged, main CI run `31026633959` |

Release handoff gates:

| Gate | Current status | Required before promotion | Evidence |
| --- | --- | --- | --- |
| foundry-release-candidate | Ready | Yes | `go run ./cmd/foundry release candidate validate --ledger examples/readiness/active-spine-release-candidate.ledger.json`, `go run ./cmd/foundry release candidate active-stack-parity --ledger examples/readiness/active-spine-release-candidate.ledger.json --readiness-ledger examples/readiness/active-stack-readiness.ledger.json`, `go run ./cmd/foundry release promotion validate --candidate examples/readiness/active-spine-release-candidate.ledger.json --signed-smoke-summary examples/contract-fixtures/valid/foundry-signed-smoke-summary-v0.1.json --out tmp/release-promotion.fixture.json`, `go run ./cmd/foundry release handoff --candidate examples/readiness/active-spine-release-candidate.ledger.json --signed-smoke-summary examples/contract-fixtures/valid/foundry-signed-smoke-summary-v0.1.json --promotion-out tmp/release-promotion.handoff.json --notes-out tmp/release-candidate.handoff.md --manifest-out tmp/release-manifest.handoff.json`, `go run ./cmd/foundry release handoff --candidate examples/readiness/active-spine-release-candidate.ledger.json --signed-smoke-summary docs/evidence/pulse/20260623T213426Z-signed-smoke-release-gate/signed-smoke-summary.json --promotion-out tmp/release-promotion.final.json --notes-out docs/operations/ACTIVE-SPINE-2026-06-23-RELEASE-CANDIDATE.md --manifest-out tmp/release-manifest.final.json` |
| forge-release-candidate-handoff | Ready | Yes | `forge release-candidate validate --candidate examples/release-preview/release-candidate.v0.1.example.json`, ao-forge main CI run 28066645277, ao-forge Release Preview run 28066645263, ao-forge Production Readiness Ops run 28098513733 |
| covenant-policy-spine | Ready | Yes | `covenant policy spine --json`, covenant.policy-spine-result.v1, ao-covenant main CI run 28067515041, ao-covenant Release Readiness run 28067529569, ao-covenant Production Readiness Ops run 28098729037 |
| ao-command-rsi-health | Ready | Yes | ao-command rsi health --arena-gate ../ao-arena/tmp/arena-promotion-gate.json --crucible-gate ../ao-crucible/tmp/crucible-hardening-gate.json --sentinel-verdict ../ao-sentinel/tmp/sentinel-verdict.json --promoter-gate ../ao-promoter/tmp/promotion-gate.json --json, rsi_mode=governed_fixture_local, rsi_capability=demonstrated_local_fixture_loop, mutates_repositories=false, ao-command main CI run 28148110317, ao-command PR #18 merged |
| signed-smoke-release-gate | Manual Required | Yes | `docs/operations/SIGNED-SMOKE-RELEASE-GATE.md`, workflow_dispatch signed_smoke=true, freshness_summary.status=ready, signed_smoke_summary=ready, release_safe=true, `docs/evidence/pulse/20260623T213426Z-signed-smoke-release-gate/release-promotion.live.json` |

The machine-readable source for this snapshot is
[`examples/readiness/active-stack-readiness.ledger.json`](examples/readiness/active-stack-readiness.ledger.json).
The AO2 active-spine release candidate ledger is
[`examples/readiness/active-spine-release-candidate.ledger.json`](examples/readiness/active-spine-release-candidate.ledger.json).
<!-- foundry:active-stack-readiness:end -->

No active readiness path depends on `ao-operator`, `ao-runtime`,
`ao-control-plane`, `ao-conductor`, `agy-swarms`, or `codex-cron`.


</details>

## Public Documents

- [AO Foundry v0.1 Design](docs/design/AO-FOUNDRY-V0.1.md)
- [One-Shot Factory Run](docs/operations/ONE-SHOT-FACTORY-RUN.md)
- [AO2 Pulse Event Loop](docs/operations/AO2-PULSE-EVENT-LOOP.md)
- [Branch protection](docs/operations/BRANCH-PROTECTION.md)
- [Signed-smoke release gate](docs/operations/SIGNED-SMOKE-RELEASE-GATE.md)
- [Production Readiness SDD](docs/sdd/AO-FOUNDRY-PRODUCTION-READINESS-SDD.md)
- [Pulse Golden Loop SDD](docs/sdd/AO-FOUNDRY-PULSE-GOLDEN-LOOP-SDD.md)
- [Pulse Blueprint/Atlas Refactor Design](docs/sdd/AO-FOUNDRY-PULSE-BLUEPRINT-ATLAS-REFACTOR.md)
- [Pulse Production Adapters SDD](docs/sdd/AO-FOUNDRY-PULSE-PRODUCTION-ADAPTERS-SDD.md)
- [Registry schema](docs/contracts/foundry-registry-v0.1.schema.json)
- [Task schema](docs/contracts/foundry-task-v0.1.schema.json)
- [Run schema](docs/contracts/foundry-run-v0.1.schema.json)
- [Production readiness audit schema](docs/contracts/foundry-production-readiness-audit-v0.1.schema.json)
- [Active stack readiness schema](docs/contracts/foundry-active-stack-readiness-v0.1.schema.json)
- [Active stack production readiness rollup schema](docs/contracts/foundry-active-stack-production-readiness-rollup-v0.1.schema.json)
- [Atlas readback schema](docs/contracts/foundry-atlas-readback-v0.1.schema.json)
- [Atlas status schema](docs/contracts/foundry-atlas-status-v0.1.schema.json)
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
