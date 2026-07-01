# AO Foundry

AO Foundry is the engineering operations factory above AO Forge. It does not
replace AO Forge. Foundry coordinates many repositories, goals, branches, CI
signals, release trains, evidence queues, and overnight advancement loops, then
delegates each individual governed implementation run to AO Forge.

## AO Stack Architecture

This repository is part of the AO agent orchestration stack. Start with the
central architecture guide at
[uesugitorachiyo/ao-architecture](https://github.com/uesugitorachiyo/ao-architecture);
the AO Foundry-specific architecture page is
[ao-foundry](https://github.com/uesugitorachiyo/ao-architecture/tree/main/ao-foundry).

AO Forge remains the trusted factory brain for one governed run. AO Foundry owns
the higher-level operating view: what work is queued, which repository is ready,
which evidence is waiting, which branch or release train is blocked, and what
the next safe delegated factory action should be.

Canonical upstream intake is AO Blueprint -> AO Atlas -> AO Foundry. Blueprint
owns requirements interview and build authorization. Atlas compiles authorized
oversized objectives into stack instances, workgraphs, context packs, Foundry
handoff material, and run-link readback. Foundry validates those artifacts and
decides whether a ready item should be delegated, but it does not treat raw
operator ideas or underspecified Atlas material as implementation-ready work.

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
  - `foundry pulse run [--start-gate <pulse-overnight-start-gate.json>] --out <dir> [--rsi-baseline <eval.json>] [--rsi-min-improvement <percent>]`
  - `foundry pulse intake-preflight --blueprint-authorization <path> [--requires-atlas --atlas-blueprint-import <path> --atlas-import <path> --atlas-status <path>] [--out <path>]`
  - `foundry pulse lifecycle inspect --state <pulse-pr-lifecycle.json> [--json]`
  - `foundry pulse overnight-start-gate --intake-preflight <path> --lifecycle <path> --out <path> [--start-implementation] [--json]`
  - `foundry pulse event-loop-policy --class-gate <path> --promotion-state <path> --ci <path> --repo-state <path> --evidence-freshness <path> --sentinel <path> --promoter <path> --rollback <path> --branch-cleanup <path> --scope <path> --out <path> [--json]`
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
  - `foundry rsi improvement-gate --baseline <eval.json> --candidate <eval.json> --min-improvement <percent> --out <gate.json>`
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
go run ./cmd/foundry pulse run --start-gate examples/pulse-overnight-start-gate/ready.json --out tmp/pulse --rsi-baseline examples/evals/rsi-baseline.eval-result.json --rsi-min-improvement 5
go run ./cmd/foundry pulse intake-preflight --blueprint-authorization examples/pulse-intake/blueprint-authorization.ready.json --requires-atlas --atlas-blueprint-import examples/atlas/blueprint-import.low-risk-code.json --atlas-import examples/atlas/foundry-import.json --atlas-status examples/contract-fixtures/valid/foundry-atlas-status-v0.1.json --out tmp/pulse-intake-preflight.json
go run ./cmd/foundry pulse lifecycle inspect --state examples/pulse-lifecycle/ready-to-start-next-slice.json --json
go run ./cmd/foundry pulse overnight-start-gate --intake-preflight examples/pulse-overnight-start-gate/ready.intake-preflight.json --lifecycle examples/pulse-lifecycle/ready-to-start-next-slice.json --out tmp/pulse-overnight-start-gate.json
go run ./cmd/foundry class-gate evaluate --atlas examples/class-gate/atlas-classification.docs-multi.json --covenant examples/class-gate/covenant-ticket.docs-multi.json --sentinel examples/class-gate/sentinel.no-hold.docs-multi.json --promoter examples/class-gate/promoter.ready.docs-multi.json --rollback examples/class-gate/rollback.passed.docs-multi.json --command examples/class-gate/command-readback.docs-multi.json --ci examples/class-gate/ci.passed.docs-multi.json --out tmp/class-gate.json
go run ./cmd/foundry rsi improvement-gate --baseline examples/evals/rsi-baseline.eval-result.json --candidate examples/evals/bootstrap.eval-result.json --min-improvement 5 --out tmp/rsi-improvement-gate.json
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
attempt, control-plane readback, run, eval, RSI candidate, RSI improvement gate,
RSI next improvement task, trace, demo, release dry-run, competitive audit, and
a final `pulse-event.json` summary. It is a scheduler and evidence loop only; live
implementation remains delegated to AO Forge.

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
For `multi_repo_low_risk`, Foundry also requires a serialized per-repo dry-run
plan with ordered PR dependencies, fresh clean-main repo state, rollback ready
for each repo, CI passing for each repo, an armed kill switch, and
`max_active_repos=1`. It reports the repo plan for readback and fails closed on
unsafe concurrent execution, missing per-repo rollback or CI, stale repo state,
disarmed kill switch, or a dependency that does not point to an earlier repo in
the serialized order. Before the first serialized live rehearsal completed, its
`live_rehearsal_decision` readback kept live multi-repo execution blocked until
`low_risk_code` live evidence, rollback, Sentinel, Promoter, Command, and
clean-main CI evidence were complete. The 2026-06-30 Atlas-first rehearsal then
completed repo one through AO Command PR #56 and repo two through AO Foundry PR
#122, both with exact test-only scope, passing CI, branch cleanup, rollback
readiness, Sentinel/Promoter/Command evidence, and public-safety review. That
evidence made `multi_repo_low_risk` the highest proven live class. The
2026-06-30 complex_repo_mutation mission then completed all 12 governed nodes
with digest-bound run-link, node-gate, rollback, Sentinel, Promoter, Command,
CI, merge, and forbidden-surface closure evidence. The 2026-07-01
fully_unsupervised_complex_mutation first non-planning mission then completed
all 26 serialized nodes with final rollup, Promoter verdict, Command readback,
branch cleanup, no concurrent mutation, no forbidden surfaces, and RSI denial
preserved. The highest proven live class is now
`fully_unsupervised_complex_mutation`.

AO Foundry PR #175, commit
`b12ac9b62ab8d20b4092d2a5d13081607567e816`, records the final closure for
`bounded_rsi_evidence_rehearsal`: all 32 evidence nodes completed, every stop
gate cleared, Foundry final rollup accepted, Promoter verdict
`promote_bounded_rsi_evidence_rehearsal` accepted, and Command readback
`promote_bounded_rsi_evidence_rehearsal_keep_broad_rsi_denied` accepted. This is
bounded evidence-rehearsal proof only. Broad RSI, hidden instruction mutation,
unrestricted self-modification, and policy/auth/secret/provider/deploy/release/
config/dependency expansion remain denied. The next denied class remains `RSI`.

`foundry pulse event-loop-policy` consumes the class-gate result plus
promotion-state, CI, repo-cleanliness, evidence-freshness, Sentinel, Promoter,
rollback, branch-cleanup, and scope-boundary evidence. It emits
`ao.foundry.pulse-event-loop-policy.v0.1` with `safe_to_continue=true` only
when every gate is ready for the current proven class and the class gate has
`safe_to_execute=true`. The policy stops on failing CI, dirty or unsynced
repos, stale evidence, Sentinel holds, Promoter denial, rollback failure,
branch cleanup failure, broadened scope, or any attempted jump to a class that
is not proven by promotion evidence. It does not schedule, execute, approve,
open PRs, merge PRs, call providers, or mutate repositories.

`scripts/blueprint-atlas-pulse-e2e-dry-run.sh` proves the fixture-only
Blueprint -> Atlas -> Foundry -> AO Command control path. The ready path starts
the runner after digest-bound Blueprint authorization, Atlas Blueprint import,
Atlas Foundry import, Foundry readback, and Pulse gates pass. The blocked
Blueprint path writes a blocked runner decision and AO Command readback, but
does not produce `pulse-event.json` or start implementation.

`scripts/complex-refactor-workgraph-rehearsal.sh` is the reference oversized
task demo. It validates an Atlas workgraph with completed, ready, blocked, and
stitch nodes; validates Foundry import/readback; runs the Pulse gate e2e proof;
emits blocked-node repair and needs-context repack artifacts; writes AO Command
complex-refactor status readback; and reports the next ready factory task
without starting blocked work. It also asks Atlas to emit a Foundry import for
only the `workgraph next` node and fails if the selected import contains more
than that one dependency-safe task.

`scripts/overnight-rehearsal-runner.sh` wraps that rehearsal as a dry-run
overnight control-chain check. It validates Pulse gate/lifecycle state, Atlas
import/readback, repair/repack artifacts, and AO Command readback, then emits
`ao.foundry.overnight-rehearsal-runner.v0.1` without scheduling or executing
work.

`scripts/fresh-overnight-rehearsal-artifact.sh` runs the same dry-run chain into
a fresh timestamped output directory and emits
`ao.foundry.overnight-rehearsal-artifact.v0.1`. The artifact links the runner
summary, complex-refactor summary, and AO Command readback with SHA-256 digests
so operators can preserve the exact rehearsal evidence without treating it as
live mutation authority. The stable operator sequence is documented in
`docs/operations/OVERNIGHT-REFRACTOR-REHEARSAL-RUNBOOK.md`.

`scripts/atlas-stress-readiness.sh` consumes AO Atlas's large workgraph stress
fixture from Foundry. It validates the stress workgraph, generates Atlas
Foundry-import material, validates that import through Foundry, and emits
`ao.foundry.atlas-stress-readiness.v0.1` with ready, blocked, completed, and
imported task counts.

Foundry's dry-run live-mutation request fixture lives at
`examples/contract-fixtures/valid/foundry-live-mutation-request-v0.1.json`. It
requests Covenant `covenant.live-mutation-authority.v1` evaluation for a tiny
docs-only mutation class while preserving `mode=dry_run_only`,
`live_mutation_allowed=false`, `provider_calls_allowed=false`, and
`release_or_publish_allowed=false`. It is request material, not execution
authority.

`scripts/live-mutation-worktree-isolation-proof.sh` consumes a public-safe
worktree candidate fixture and emits
`ao.foundry.worktree-isolation-proof.v0.1`. The proof is ready only when the
candidate uses a clean, isolated, non-reused `.foundry-local/worktrees/...`
worktree on a fresh `codex/*` branch from synced `main`. Dirty worktrees,
untracked changes, reused branches/worktrees, unsafe paths, or expanded
authority block the proof. The script is dry-run only: it does not create a
worktree, switch branches, mutate repositories, schedule work, approve work,
call providers, publish, or release.

`scripts/live-mutation-rollback-rehearsal.sh` consumes a public-safe rollback
candidate fixture and emits
`ao.foundry.live-mutation-rollback-rehearsal.v0.1`. The rehearsal is ready only
when the proposed patch and rollback patch are digest-bound, the rollback plan
uses an ignored `.foundry-local/quarantine/...` path, the operator kill switch
is armed, and verification commands stay local. Missing rollback material,
unsafe paths, disabled kill switch state, or expanded authority block the
rehearsal. The script does not apply either patch and does not grant live
mutation authority.

`scripts/governed-live-mutation-dry-run-chain.sh` combines the current
fixture-only control chain into
`ao.foundry.governed-live-mutation-dry-run-chain.v0.1`: Blueprint/Atlas complex
task evidence, Foundry start gate, Covenant authority dry-run, Forge dry-run
plan, AO2 dry-run packet, worktree isolation, rollback rehearsal, Sentinel hold
verdict, Promoter boundary, and AO Command readback. A ready result means the
first tiny live-mutation class is safe to request through a separate governed
operator approval path. It does not perform live mutation and does not claim
ungated authority.

With `--mutation-class low_risk_code`, the same script emits a low-risk-code
dry-run chain for one tiny source-plus-test mutation. The chain includes Atlas
classification, Foundry class gate, Covenant ticket, Forge dry-run plan, AO2
bounded patch packet, rollback proof, Sentinel hold verdict, Promoter boundary,
and AO Command readback. It reports `safe_to_request=true` when those dry-run
inputs are present, but keeps `safe_to_execute=false` until later live
promotion evidence exists.

`scripts/low-risk-code-live-rehearsal-gate.sh` consumes that low-risk dry-run
chain and decides whether the first bounded low-risk code live rehearsal may
start. Without explicit live policy evidence bound to the dry-run chain digest,
it emits `safe_to_request=true`, `safe_to_execute=false`, and
`exact_next_step=collect_low_risk_code_live_policy_evidence`. The gate itself
does not create branches, open PRs, mutate repositories, schedule work, approve
work, call providers, publish, or release.

`scripts/live-mutation-readiness-rollup.sh` consumes that chain summary and
emits `ao.foundry.live-mutation-readiness-rollup.v0.1`. The rollup answers the
operator question precisely: the first tiny docs-only live-mutation class can be
requested when the chain is ready, but it is still not safe to execute until a
separate operator approval request exists. It keeps `safe_to_execute=false`,
`live_mutation_allowed=false`, and `mutates_repositories=false`.

Public readiness wording for this phase is intentionally narrow:

- dry-run governed live mutation readiness is ready;
- first docs-only live request readiness is ready;
- execution readiness is conditional on explicit exact-scope approval and every
  downstream gate passing;
- `fully_unsupervised_complex_mutation` is proven only for the governed 26-node
  first non-planning rehearsal boundary;
- `bounded_rsi_evidence_rehearsal` is live-proven only as a bounded evidence
  rehearsal state.

The first approval request contract is
`ao.foundry.live-mutation-approval-request.v0.1`, with a public-safe example at
`examples/contract-fixtures/valid/foundry-live-mutation-approval-request-v0.1.json`.
It pins the target repo, `codex/live-docs-` branch prefix, docs-only allowlist,
forbidden paths, maximum changed file count, rollback plan, kill-switch,
Covenant authority, Sentinel hold, and AO Command readback references. This is
only a request artifact: it keeps `safe_to_execute=false` until a future
Covenant approval ticket exactly matches the scope.

`scripts/live-docs-approval-gate.sh` reads that approval request and a Covenant
`covenant.live-docs-approval-ticket.v1` ticket, then emits
`ao.foundry.live-docs-approval-gate.v0.1`. The gate sets
`safe_to_execute=true` only when the ticket is approved, unexpired, unconsumed,
has an approver identity, and exact-matches the request's repo, branch policy,
docs allowlist, forbidden paths, and changed-file limit. It emits evidence only;
it does not create a branch, mutate files, approve work, publish, upload, or call
providers.

`scripts/live-docs-worktree-prepare.sh` reads the ready approval gate and a
public-safe preparation candidate, then emits
`ao.foundry.live-docs-worktree-prepare.v0.1`. The gate is ready only when the
candidate uses a fresh ignored `.foundry-local/worktrees/...` path, a
`codex/live-docs-*` branch from `main`, a clean non-reused worktree, an armed
kill switch, and a changed-file plan bounded to docs-only markdown paths. It is
validation-only: it does not create a worktree, create a branch, mutate
repositories, execute work, approve work, publish, upload, release, or call
providers.

`scripts/live-docs-rollback-execution-rehearsal.sh` consumes a docs-only
rollback execution candidate and emits
`ao.foundry.live-docs-rollback-execution-rehearsal.v0.1`. Unlike the earlier
metadata-only rollback proof, this rehearsal initializes a temporary fixture
Git workspace, applies the proposed docs patch, applies the rollback patch, and
verifies the target docs file is removed again. It never applies patches to the
live repository and keeps `live_mutation_allowed=false` and
`mutates_repositories=false`.

`scripts/approved-live-docs-dry-run-chain.sh` links the first approved
docs-only path end to end as
`ao.foundry.approved-live-docs-dry-run-chain.v0.1`: approval request, Covenant
approval ticket, Foundry approval gate, Forge execution guard, AO2 docs-only
patch packet, worktree preparation, rollback execution rehearsal, Sentinel
verdict, Promoter boundary, and AO Command readback. This is still dry-run
evidence only. A ready result means the chain is prepared for the next live docs
PR rehearsal gate; it does not create a branch, open a PR, mutate repositories,
call providers, publish, upload, release, or claim broad autonomous mutation.

`scripts/live-docs-pr-rehearsal-gate.sh` is the final decision gate before a
first real docs-only branch/PR rehearsal may start. Without an explicit
approval artifact it emits `safe_to_execute=false` and
`exact_next_step=request_operator_approval`. With an approved, unconsumed
Covenant ticket whose digest matches the approved dry-run chain, it can emit
`safe_to_execute=true` for the first docs-only PR rehearsal decision only. The
gate itself still does not create branches, create worktrees, open PRs, merge,
mutate repositories, call providers, publish, upload, tag, or release.

`scripts/first-live-docs-readiness-rollup.sh` summarizes the approved docs-only
chain and final PR rehearsal gate as
`ao.foundry.first-live-docs-readiness-rollup.v0.1`. It reports
`first_live_class=docs_only`, `approved_scope=docs_only`, `safe_to_request`,
`safe_to_execute`, and `exact_next_step` while preserving
`mutates_repositories=false`, `opens_pr=false`, and
`fully_unsupervised_complex_mutation_claimed=false`. A ready rollup means only
that the first docs-only PR rehearsal decision is ready; it does not perform
the live branch/PR rehearsal.

The approval artifact is not blanket consent. It must name the exact repo,
branch prefix, docs-only path allowlist, forbidden paths, maximum changed files,
rollback plan, kill switch, and unconsumed Covenant ticket digest for the first
docs-only class. Even when the rollup reports `safe_to_execute=true`, Foundry is
only saying the first docs-only PR rehearsal may be started by explicit operator
instruction under those gates. It is not permission for broad live mutation,
provider calls, releases, tags, uploads, direct-main changes, or fully
unsupervised complex repository mutation.

The pulse loop writes `ao.foundry.rsi-candidate.v0.1` evidence after generating
the local candidate eval result and before running the gate. The RSI improvement
gate then compares the baseline eval result to that generated candidate eval
result and requires a measurable improvement, such as 5 percentage points. It
writes `ao.foundry.rsi-improvement-gate.v0.1` evidence with source hashes,
`autonomous_claim=measured_local_improvement`, and
`mutates_repositories=false`; it blocks when the threshold is not met. When the
gate passes, the loop writes `ao.foundry.rsi-next-improvement-task.v0.1`
evidence that binds the candidate and gate artifact paths to the current GoalRun
next task without mutating repositories.

Foundry can also retain an `ao.foundry.rsi-control-surface-packet.v0.1`
portfolio readback packet that links Blueprint build authorization, Forge
retained GoalRun evidence, AO2 RSI evidence, and control-plane observer
readback. The packet preserves the same boundary as the pulse loop:
`bounded_governed_rsi` may be supported by evidence, while
`full_autonomous_self_mutating_rsi` remains denied and cannot be published by
Foundry.

This is bounded governed RSI evidence only. AO Foundry proves a local candidate
improved by the configured threshold, such as 5 percentage points, and preserves
`mutates_repositories=false`. Downstream AO Command RSI health may report
`claim_level=bounded_governed_rsi decision=allowed` for that read-only evidence
chain, while `claim_level=full_autonomous_self_mutating_rsi decision=denied`
remains the correct boundary until mutation authority, rollback, live
self-change evidence, and AO Covenant approval exist.

The later `bounded_rsi_evidence_rehearsal` closure is also bounded-only. It may
be reported as live-proven for the governed 32-node evidence rehearsal, but it
does not prove broad RSI, does not authorize unrestricted self-modification, and
does not allow hidden or policy-changing self-modification.

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

## Verified Active Stack Snapshot

<!-- foundry:active-stack-readiness:start -->
Last local sweep: 2026-06-23.

| Repository | Current status | Verification evidence |
| --- | --- | --- |
| AO Foundry | Ready | `go test ./...`, `go vet ./...`, `go build ./cmd/foundry ./cmd/ao`, `go run ./cmd/foundry registry validate --registry examples/registry/local-ao-stack.foundry-registry.json`, `go run ./cmd/foundry task validate --task examples/tasks/ao-foundry-bootstrap.foundry-task.json`, `go run ./cmd/foundry repo board --registry examples/registry/local-ao-stack.foundry-registry.json`, scripts/active-stack-readiness-loop.sh --out tmp/active-stack-readiness-loop.json, scripts/active-stack-github-runs-report.sh --out tmp/active-stack-github-runs-report.json, `go run ./cmd/foundry release handoff --candidate examples/readiness/active-spine-release-candidate.ledger.json --signed-smoke-summary examples/contract-fixtures/valid/foundry-signed-smoke-summary-v0.1.json --promotion-out tmp/release-promotion.handoff.json --notes-out tmp/release-candidate.handoff.md --manifest-out tmp/release-manifest.handoff.json`, `go run ./cmd/foundry readiness evidence-check --ledger examples/readiness/active-stack-readiness.ledger.json --github-runs-report tmp/active-stack-github-runs-report.json`, scripts/verify-branch-protection.sh, .github/workflows/production-readiness-ops.yml, signed-smoke release promotion release_safe=true |
| AO Atlas | Ready | `go test ./...`, `go vet ./...`, `go build ./cmd/atlas`, scripts/production-readiness.sh, scripts/atlas-foundry-roundtrip-smoke.sh, `go run ./cmd/foundry atlas status --registry examples/registry/atlas-demo.foundry-registry.json --import examples/atlas/foundry-import.json --run-link examples/atlas/run-link.completed.json`, ao.foundry.atlas-status.v0.1, schedules_work=false, executes_work=false, approves_work=false, Production Readiness Ops run 28346305510, PR #15 merged, main CI run `28346305503` |
| AO Forge | Ready | license policy, license policy required in branch protection, GoalRun fixtures, `go test ./...`, `go vet ./...`, `go build`, production-readiness schemas, actionlint, Release Preview run 28066645263, Production Readiness Ops run 28321477720, PR #135 merged, main CI run `28246591616` |
| AO Command | Ready | AO2-first boundary audit, release dry-run chain, ao-command rsi health --arena-gate ../ao-arena/tmp/arena-promotion-gate.json --crucible-gate ../ao-crucible/tmp/crucible-hardening-gate.json --sentinel-verdict ../ao-sentinel/tmp/sentinel-verdict.json --promoter-gate ../ao-promoter/tmp/promotion-gate.json --json, rsi_mode=governed_fixture_local, mutates_repositories=false, production readiness 100, 36/36 gates, license policy required in branch protection, Production Readiness Ops run 28321548229, PR #36 merged, ao-command atlas status --status ../ao-foundry/examples/contract-fixtures/valid/foundry-atlas-status-v0.1.json, main CI run `28345912142` |
| AO2 | Ready | `npm run release:readiness:static`, `npm run verify`, native AO2 runtime evidence tests, Production Readiness Ops run 28321735689, PR #243 merged, main CI run `28339961675` |
| AO2 Control Plane | Ready | license policy, `cargo fmt --all --check`, Python guard tests, `cargo test --workspace`, `cargo clippy --workspace --all-targets -- -D warnings`, `cargo deny check bans licenses sources`, `cargo audit --deny warnings`, `cargo build --release -p ao2-cp-server`, active stack handoff readback gate, Production Readiness Ops run 28321488512, PR #90 merged, main CI run `28280708823` |
| AO Covenant | Ready | `covenant policy spine --json`, covenant.policy-spine-result.v1, Release Readiness run 28067529569, branch protection verifier, Production Readiness Ops run 28321567179, PR #59 merged, main CI run `28186617447` |

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
- [RSI candidate schema](docs/contracts/foundry-rsi-candidate-v0.1.schema.json)
- [RSI control-surface packet schema](docs/contracts/foundry-rsi-control-surface-packet-v0.1.schema.json)
- [RSI improvement gate schema](docs/contracts/foundry-rsi-improvement-gate-v0.1.schema.json)
- [RSI next improvement task schema](docs/contracts/foundry-rsi-next-improvement-task-v0.1.schema.json)
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
