# AO Foundry Production Readiness SDD

## Objective

Advance AO Foundry toward production-readiness 100% without drifting from its
role as the engineering operations factory above AO Forge.

## Scope

This SDD slice adds a measurable production-readiness audit. Foundry reads a
registry and a task, evaluates whether the next delegated action is safe, emits
a public-safe audit record, and refuses to advance when readiness is below 100.

This slice does not execute providers, mutate sibling repositories, push, tag,
publish, upload artifacts, or replace AO Forge. AO Forge still owns governed
single-run execution.

## Contract

The audit schema is `docs/contracts/foundry-production-readiness-audit-v0.1.schema.json`.
The audit uses five 20-point gates:

1. Registry contract is valid.
2. Task contract is valid.
3. All task target repositories are registered.
4. All target repository readiness signals are `ready`.
5. The task delegates governed work to AO Forge and remains local-only.

`status=ready` requires `score=100`, `max_score=100`, and no `next_actions`.
Any failed check produces `status=blocked`, a score below 100, and a non-zero
CLI exit.

## Operator Flow

```sh
go run ./cmd/foundry readiness audit \
  --registry examples/registry/local-ao-stack.foundry-registry.json \
  --task examples/tasks/ao-foundry-bootstrap.foundry-task.json \
  --out examples/readiness/ao-foundry-bootstrap.production-readiness-audit.json
```

Then run:

```sh
go test ./...
```

## Self Evaluation And Forecast

Subagent review identified the same near-term slice as the safest next step:
implement `foundry readiness audit --registry <path> --task <path> [--out <path>]`
with a 100-point gate, non-zero exits below 100, and no execution side effects.

The Pulse mechanics review found that AO Foundry should later mimic AO Forge's
GoalRun model: durable loop records, readiness audits, evidence hash checks,
negative fixtures, and retained evidence paths. That is the next direction, but
it is intentionally not bundled into this slice. The current slice creates the
scoring boundary that future GoalRun-style loops can call without taking over
Forge execution.

## Drift Controls

- Foundry scores readiness and recommends or blocks.
- Forge executes governed factory runs.
- Covenant remains the policy gate.
- AO2 remains the execution engine.
- Control-plane upload is evidence observation, not approval.
