# AO Foundry Pulse Blueprint/Atlas Refactor Design

## Objective

Refactor the Foundry pulse/event-loop direction so the autonomous loop follows
the current AO stack source of truth:

```text
AO Blueprint -> AO Atlas -> AO Foundry -> AO Forge -> AO Covenant -> AO2 -> readback
```

The goal is not to give Foundry more authority. The goal is to make the loop
consume the right upstream evidence before it schedules work, run one bounded
slice at a time, and stop cleanly when readiness or PR lifecycle conditions are
not satisfied.

## Canonical Loop

1. AO Blueprint handles underspecified work.
   - If an objective lacks requirements, constraints, tests, safety limits, or
     build authorization, the loop must emit or reference a Blueprint request
     and stop.
   - A transcript, operator note, or Foundry queue item is not enough by itself
     to treat work as implementation-ready.
2. AO Atlas compiles oversized authorized work.
   - Atlas owns stack-instance manifests, workgraphs, factory tasks, bounded
     context packs, Foundry handoff/import material, and run-link readback.
   - Atlas does not schedule, execute, approve, publish, call providers, or
     mutate sibling repositories.
3. AO Foundry schedules the next safe ready item.
   - Foundry validates registry, readiness, Atlas handoff/readback, active-stack
     evidence, branch hygiene, PR/check state, and stop conditions.
   - Foundry owns the one-slice PR lifecycle: branch, implement, verify, push,
     open PR, wait, fix failures, merge, sync, delete branch, and only then
     continue.
4. AO Forge executes one governed factory run.
   - Foundry delegates implementation to Forge; it does not become the factory
     brain for a single run.
5. AO Covenant gates policy and side effects.
   - Provider, release, claim-publish, repository mutation, and authority
     changes remain policy-gated.
6. AO2 executes bounded local work and records evidence.
   - AO2 owns local-first execution, exact-digest approvals, Pulse evidence, and
     evaluator closure.
7. Readback surfaces observe.
   - ao2-control-plane and AO Command read evidence. They do not approve,
     schedule, execute, publish, or mutate repositories.

## Existing Pulse Behavior To Refactor

The current `foundry pulse run` is useful as a local evidence bundle, but the
next production loop should lift the following behavior into an explicit
Blueprint/Atlas-aware scheduler contract:

- intake readiness: require Blueprint authorization or a blocked Blueprint
  request before scheduling implementation;
- oversized objective routing: require Atlas workgraph/context-pack handoff
  before Foundry treats large work as ready;
- active-stack registry: include AO Atlas as an active repo and keep deprecated
  operator/runtime/conductor/swarm repos out of scope;
- PR lifecycle: allow only one active feature branch and one unmerged PR per
  slice;
- checks: wait on GitHub checks and repair failures on the same branch instead
  of starting new work;
- cleanup: sync main, prune remotes, delete merged branches, and verify clean
  status before the next slice;
- readback: record Atlas run-link, Foundry status, Forge packet, AO2 evidence,
  and control-plane/Command observer evidence as durable artifacts;
- stop behavior: distinguish blocking next actions from maintenance
  suggestions so a ready stack does not invent new work.

## Required Stop Conditions

The loop must stop when any of these is true:

- Blueprint authorization is missing, denied, expired, or underspecified.
- Atlas handoff/readback is missing for oversized work.
- The active-stack readiness loop fails.
- A target repository is dirty, not on synced `main`, or has an unrelated local
  change.
- A PR, check, or job is still open/running for the current slice.
- Verification fails after reasonable repair attempts.
- Production-readiness tasks have no blocking next action.
- The operator requests stop.
- The next action would require provider calls, release/tag/upload, direct main
  mutation, or claim publication without explicit authority.

## Non-Goals

- No multi-branch pileups.
- No stacked unmerged PRs.
- No direct main bypass.
- No provider calls or provider API-key paths.
- No release, tag, upload, or live publication side effects.
- No sibling-repository mutation outside the current approved slice.
- No use of deprecated `ao-operator`, `ao-runtime`, `ao-control-plane`,
  `ao-conductor`, or subscription-backed swarm repos.
- No conversion of bounded governed RSI evidence into full autonomous RSI
  publication authority.

## Refactor Slices

### Slice A: Design And Readback Contract

Document the Blueprint/Atlas-aware loop and define the required evidence names,
stop conditions, and non-goals. This document is that first slice.

### Slice B: Pulse Intake Preflight

Add a Foundry pulse preflight that accepts Blueprint authorization and Atlas
handoff/readback inputs, then fails closed when missing or blocked.

The executable slice is:

```sh
go run ./cmd/foundry pulse intake-preflight \
  --blueprint-authorization examples/pulse-intake/blueprint-authorization.ready.json \
  --requires-atlas \
  --atlas-blueprint-import examples/atlas/blueprint-import.low-risk-code.json \
  --atlas-import examples/atlas/foundry-import.json \
  --atlas-status examples/contract-fixtures/valid/foundry-atlas-status-v0.1.json \
  --out tmp/pulse-intake-preflight.json
```

It emits `ao.foundry.pulse-intake-preflight.v0.1`. A blocked Blueprint request
is a valid blocked intake result, not a ready scheduling signal. Missing
Blueprint authorization, blocked authorization treated as ready, missing Atlas
Blueprint import, missing Atlas handoff/readback, Atlas authority claims, and
unsafe source artifact paths fail closed. For oversized and live-mutation work,
Foundry accepts Blueprint material only after Atlas has compiled it into a ready
Blueprint import, Foundry import, and Foundry status/readback chain. This
preflight does not schedule, execute, approve, upload, publish, call providers,
or mutate sibling repositories.

### Slice C: One-Slice PR Lifecycle State

Add a local state contract that records current branch, PR number, check status,
merge status, cleanup status, and the next allowed transition. It must reject
starting another slice while a branch, PR, or check is active.

The executable slice is:

```sh
go run ./cmd/foundry pulse lifecycle inspect \
  --state examples/pulse-lifecycle/ready-to-start-next-slice.json \
  --json
```

It emits `ao.foundry.pulse-pr-lifecycle.v0.1`. The gate allows
`start_next_slice` only when the target repo is clean on synced `main`, no
current PR/check is active or failing, merged branch cleanup is complete, and
there is no blocker reason. Open PRs, pending checks, failed checks, branch
cleanup gaps, unsynced main, dirty worktrees, and multiple active `codex/*`
branches fail closed. This command inspects local fixture/state only; it does
not create branches, push, merge, delete branches, schedule work, execute work,
approve work, publish, call providers, or mutate sibling repositories.

### Slice G: Overnight Start Gate

Compose the Blueprint/Atlas intake preflight with one-slice PR lifecycle state
before any autonomous overnight/event-loop advancement begins.

The executable slice is:

```sh
go run ./cmd/foundry pulse overnight-start-gate \
  --intake-preflight examples/pulse-overnight-start-gate/ready.intake-preflight.json \
  --lifecycle examples/pulse-lifecycle/ready-to-start-next-slice.json \
  --out tmp/pulse-overnight-start-gate.json
```

It emits `ao.foundry.pulse-overnight-start-gate.v0.1`. The gate allows
`start_next_slice` only when the intake preflight is ready, Blueprint/Atlas
source evidence is digest-bound and current, lifecycle state allows starting,
main is synced, the worktree is clean, no current-slice PR/check is active, and
merged branch cleanup is complete. A Blueprint clarification preflight returns
`status=blocked` and `allowed_next_action=request_blueprint_clarification`
unless `--start-implementation` is supplied, in which case it fails closed.
Failed preflight, pending/failing checks, incomplete cleanup, dirty worktrees,
unsynced main, and stale evidence digests fail closed. This command writes gate
evidence only; it does not start implementation, schedule work, execute work,
approve work, publish, call providers, or mutate repositories.

### Slice J: Runner Start-Gate Enforcement

Wire the overnight start gate into the runner itself. `foundry pulse run`
accepts `--start-gate <pulse-overnight-start-gate.json>` and writes
`pulse-runner-start-decision.json` with
`schema_version=ao.foundry.pulse-runner-start-decision.v0.1`.

The runner may generate a pulse bundle only when the start gate status is
`ready`, `allowed_next_action=start_next_slice`, and the gate carries
digest-bound source evidence. Blocked Blueprint clarification, failed preflight,
stale lifecycle state, missing digests, stale hashes, dirty worktrees, or
unsynced local main stop before implementation evidence is produced.

### Slice K: Blueprint To Command Dry Run

Add a fixture-only end-to-end script:

```sh
scripts/blueprint-atlas-pulse-e2e-dry-run.sh \
  --out docs/evidence/pulse/blueprint-atlas-pulse-e2e-local
```

The script proves the intended control chain without live execution:
Blueprint authorization/request -> Atlas import/readback -> Foundry intake
preflight -> PR lifecycle -> overnight start gate -> runner start decision ->
AO Command `pulse status` readback. The ready path produces a ready runner
decision and pulse event. The blocked Blueprint path produces blocked preflight,
start-gate, runner-decision, and Command readback evidence, but no pulse event.

### Slice L: Complex Refactor Workgraph Rehearsal

Add a realistic fixture-only oversized-task rehearsal:

```sh
scripts/complex-refactor-workgraph-rehearsal.sh \
  --out docs/evidence/pulse/complex-refactor-workgraph-rehearsal-local
```

The rehearsal uses `examples/complex-refactor-workgraph/` to model a complex AO
stack refactor as Atlas factory tasks with completed, ready, blocked, and stitch
nodes. It validates Atlas workgraph/context-pack/mission status, Foundry
import/readback, Pulse start-gate evidence, blocked-node repair plans,
needs-context context repacks, and AO Command readback. The summary identifies
the next ready factory task and explains that blocked tasks must wait for
completed run-link evidence.

### Slice D: Atlas Workgraph Scheduler Input

Teach Foundry to read Atlas ready nodes as scheduler input while preserving
Atlas compile-only authority. Foundry may select a ready node; Atlas must not
schedule it.

### Slice E: Closure Packet

Emit one closure packet that links Blueprint authorization, Atlas handoff,
Foundry scheduling, Forge run evidence, Covenant gate, AO2 evidence, observer
readback, verification commands, PR merge, branch cleanup, and stop/continue
decision.

## Verification Direction

The refactor should keep existing public-safe verification and add targeted
checks as each slice becomes executable:

```sh
go test ./...
go vet ./...
go build ./cmd/foundry ./cmd/ao
go run ./cmd/foundry registry validate --registry examples/registry/local-ao-stack.foundry-registry.json
go run ./cmd/foundry contract fixtures validate
scripts/active-stack-readiness-loop.sh --out tmp/active-stack-readiness-loop.json
```

Future executable slices should add fixtures for missing Blueprint
authorization, blocked Atlas handoff, existing open PR, failed check repair,
successful merge cleanup, and stop-at-readiness behavior.

## Authority Boundary

This design keeps Foundry as the scheduler/coordinator only. Blueprint decides
whether work is specified enough. Atlas compiles bounded work material. Forge
runs one governed job. Covenant gates side effects. AO2 executes. Readback
surfaces observe. Foundry must not collapse those roles into a single
self-authorizing loop.
