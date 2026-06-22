# GoalRun Readiness Loop Evidence

## Objective

Advance AO Foundry from task-level production readiness to durable GoalRun-style
AO2 Pulse loop readiness.

## SDD Used

- `docs/sdd/AO-FOUNDRY-GOAL-RUN-READINESS-SDD.md`

## Subagent Self Evaluation

The SDD review agreed this is the correct next slice but required tighter
acceptance around negative fixtures, GoalRun compatibility with the existing
run-record contract, `next_action_guard` semantics, terminal phases, and
mandatory evidence hashes.

Those concerns were incorporated before closeout:

- GoalRun is documented as loop-control state, not a replacement for
  `foundry-run-v0.1`.
- Unsafe evidence path, stale digest, unsafe guard, terminal phase, and blocked
  target readiness fixtures fail closed.
- Evidence hashes are mandatory and verified against repository-relative files.
- `next_action_guard` must require AO Forge delegation and reject unsafe direct
  mutation language.

## Commands

```sh
go test ./...
go run ./cmd/foundry goal readiness --goal-run examples/goals/ao-foundry-production-readiness.goal-run.json --registry examples/registry/local-ao-stack.foundry-registry.json --task examples/tasks/ao-foundry-bootstrap.foundry-task.json --out examples/readiness/ao-foundry-production-readiness.goal-readiness-audit.json
go run ../ao-forge/cmd/forge contract validate --schema docs/contracts/foundry-goal-run-v0.1.schema.json --document examples/goals/ao-foundry-production-readiness.goal-run.json
go run ../ao-forge/cmd/forge contract validate --schema docs/contracts/foundry-goal-readiness-audit-v0.1.schema.json --document examples/readiness/ao-foundry-production-readiness.goal-readiness-audit.json
```

## Result

The GoalRun readiness audit for `ao-foundry-production-readiness` reports
`status=ready` and `score=100`.

## Next Loop Direction

The next readiness loop should add a guarded GoalRun update command that refuses
in-place writes, records changed fields, and advances phases only after a
passing GoalRun readiness audit.
