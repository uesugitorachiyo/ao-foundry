# AO Foundry v0.1

## Product Thesis

AO Foundry is the engineering operations factory above AO Forge. It watches the
whole engineering system: repositories, goals, branches, CI signals, release
trains, evidence queues, and scheduled advancement loops. It chooses the next
safe operation and delegates governed execution to AO Forge.

The value of Foundry is not another executor. The value is a durable operations
model that can answer: what is ready, what is blocked, what evidence is missing,
which train can advance, and which Forge run should happen next.

## Boundaries

AO Foundry owns cross-repo operating state, prioritization, queueing, readiness
summaries, and the next-action recommendation. AO Forge owns one governed
factory run, including plan generation, Covenant gating, execution adapter
routing, evidence capture, and the operator packet. AO Covenant remains the
policy authority. AO2 remains the governed execution engine. The control plane
remains an evidence observer. AO Command remains the operator command surface.

Foundry must not approve its own side effects, execute providers directly, or
turn evidence upload into approval. A Foundry action that mutates code should
become a delegated AO Forge run.

## Architecture

The v0.1 architecture has four layers:

1. Registry: a public-safe inventory of repositories, roles, evidence sources,
   allowed automation, branches, and readiness signals.
2. Task queue: queued engineering operations tasks with target repos,
   acceptance criteria, safety boundaries, and required delegations.
3. Run record: a future durable record of a Foundry loop, including decisions,
   delegated Forge packets, evidence references, and terminal state.
4. CLI: a minimal local operator tool for validation, status, and the next
   delegated action.

GoalRun records are pulse-loop control state. They hold the current objective,
allowed scope, stop conditions, next task, continuation prompt, action guard,
and evidence hashes used to decide whether another loop may run. They do not
replace the Foundry run record; a run record remains the historical record of a
delegated operation and its Forge packet references.

## Data Flow

```text
registry + task queue
-> foundry status / foundry next
-> delegated AO Forge factory brief
-> Covenant gate inside Forge
-> AO2 or adapter execution through Forge
-> Forge packet and evidence
-> Foundry run record in a later slice
```

Foundry reads public contracts and examples in this repository. It can produce a
recommendation, but the execution packet comes from Forge.

The production-readiness audit is the first measurable v0.1 gate. It scores a
registry and task against contract validity, target registration, target
readiness, Forge delegation, and local-only safety. `foundry next` must not
recommend delegated execution unless those gates reach 100.

Goal readiness adds a second gate for the AO2 Pulse loop: the GoalRun must be
valid, non-terminal, guarded to delegate through AO Forge, and backed by durable
evidence paths with matching SHA-256 digests.

## State Model

The registry records repository readiness as `ready`, `blocked`, or `unknown`.
The task record uses `queued`, `planned`, `delegated`, `verifying`, `passed`,
`blocked`, or `failed`. The run record uses the same progression at the loop
level and adds links to decisions, delegated runs, evidence, and closeout notes.

State changes must be evidence-backed. A readiness signal should point to a
source such as a local test command, a Forge packet, a CI check name, or a
release rehearsal result. For v0.1 these are strings; later versions can bind
them to signed evidence records.

## Operator Journeys

An operator starts with `foundry status` to see the registered stack and
readiness counts. They run `foundry registry validate` before relying on a
registry fixture. They run `foundry task validate` before queueing or delegating
a task. They run `foundry next` to see the next safe action, including which
task should be delegated to AO Forge and which verification command closes the
loop.

For overnight advancement, a scheduler can run the same read-only commands,
select a queued task whose target repos are ready, and create a Forge brief for
the delegated run. The scheduler should stop when readiness is blocked, the
task safety boundary is not local-only, or verification is missing.

## Non-Goals

Foundry v0.1 does not replace AO Forge, AO2, AO Covenant, AO Command, the
control plane, or repository CI. It does not push branches, create release
tags, publish artifacts, upload evidence, manage credentials, or run providers
directly. It does not claim production scheduling authority; it only defines the
first public-safe operating model and CLI scaffold.

## Rollout

The rollout path is:

1. Land public contracts, examples, design, and CLI validation.
2. Add generated Forge brief output from `foundry next`.
3. Add durable Foundry run records with delegated Forge packet references.
4. Add CI and release-train readers.
5. Add overnight loop controls with stop conditions and retained evidence.

Each step should preserve the boundary that Forge handles governed execution.

## Verification

The v0.1 verification command is:

```sh
go test ./...
```

Public-safety verification scans generated files for local absolute paths,
credential-like strings, and non-public coordination notes. Contract validation
is currently implemented by the Go CLI for registry and task examples; run
record validation is a documented contract for the next implementation slice.
Production readiness is verified by:

```sh
foundry readiness audit --registry <path> --task <path> --out <path>
```

The command emits `ao.foundry.production-readiness-audit.v0.1`.

## Dry-Run Pitfalls

A dry-run can prove shape without proving execution. A passed Forge packet may
show that workcells were accepted while the target files remain unchanged. The
operator must inspect target content and run target verification before treating
the one-shot as complete. Missing optional control-plane upload should not block
local completion, but missing implementation or failing tests should.
