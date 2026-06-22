# AO Foundry GoalRun Readiness SDD

## Objective

Add Foundry-owned GoalRun-style durable loop records so AO Foundry can keep an
AO2 Pulse event loop focused on production readiness without replacing AO Forge.

## Scope

This slice introduces a public-safe Foundry GoalRun contract, example GoalRun,
goal validation, and goal readiness auditing. The readiness audit verifies the
GoalRun record, retained evidence path policy, evidence hashes, next-action
guard, terminal phase policy, and the existing registry/task
production-readiness audit.

This slice does not execute providers, create branches, push, tag, publish,
upload evidence, or mutate sibling repositories. A ready goal can recommend the
next delegated AO Forge action, but AO Forge still owns governed execution.

## Contract

The GoalRun record is loop-control state. It does not replace
`foundry-run-v0.1`; the run record remains the durable record of a delegated
Foundry operation and its Forge packet references. GoalRun records drive pulse
continuation and readiness gating before a run is created or advanced.

The GoalRun record must include:

- `schema_version`
- `goal_id`
- `objective`
- `acceptance_criteria`
- `allowed_scope`
- `stop_conditions`
- `current_phase`
- `next_task`
- `continuation_prompt`
- `loop_owner`
- `next_action_guard`
- `last_iteration.evidence`

Evidence references must be repository-relative durable paths. They must not
start with `tmp/`, contain parent traversal, or use absolute machine paths.
Readiness-critical evidence requires a `sha256`; missing files and digest
mismatches fail closed.

`next_action_guard` must require AO Forge delegation and must not allow direct
provider execution, pushes, tags, release publication, uploads, credentials, or
sibling repository mutation.

Terminal phases are `complete` and `stopped`. They are valid GoalRun phases but
fail readiness because the pulse loop must not continue terminal work.

## CLI

```sh
foundry goal validate --goal-run <path>
foundry goal readiness --goal-run <path> --registry <path> --task <path> [--out <path>]
```

Readiness emits `ao.foundry.goal-readiness-audit.v0.1` JSON and exits non-zero
unless every gate passes.

## Gates

The goal readiness audit uses five 20-point gates:

1. GoalRun contract is valid.
2. GoalRun phase is not terminal.
3. Evidence paths are durable and public-safe.
4. Evidence digests match referenced files.
5. Registry/task production readiness is 100.

## Negative Fixtures

Closeout must include fail-closed tests for:

- unsafe evidence paths,
- stale evidence digests,
- unsafe next-action guards,
- terminal phases,
- blocked target readiness.

## Forecast

This slice prepares Foundry for a real AO2 Pulse loop. The next direction after
this is a guarded update command that refuses in-place writes, records changed
fields, and advances phases only when the readiness audit is passing. That
future command should still delegate implementation work to AO Forge.

## Drift Controls

- Foundry owns cross-repo goal state and readiness checks.
- Foundry does not execute implementation providers directly.
- Foundry stores evidence references and hashes, not credentials.
- Forge remains the governed execution boundary.
