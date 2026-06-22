# AO Foundry Pulse Golden Loop SDD

## Objective

Create a clean-clone-safe AO Pulse event loop that proves AO Foundry can run the
competitive production factory path end to end from public fixtures.

The loop must make AO Foundry more competitive by turning its separate readiness,
delegation, packet, policy-gate, run, eval, trace, release, and competitive
audit commands into one audited operator action. It must not replace AO Forge or
execute providers directly.

## Scope

This slice adds a local `foundry pulse run` command that writes an evidence
bundle under an operator-selected output directory. The command uses existing
Foundry contracts and fixtures to produce:

- production-readiness audit,
- goal-readiness audit,
- Forge brief,
- Forge packet copy,
- policy gate summary,
- Foundry run record from an existing Forge packet,
- eval result,
- trace spans and trace inspection summary,
- demo status,
- release dry-run manifest,
- competitive readiness audit,
- final pulse event summary.

The command is deterministic except for trace timestamps. It is public-safe,
local-only, and clean-clone safe.

## Non-Goals

- No live provider execution.
- No sibling repository mutation.
- No pushes, tags, releases, uploads, or credential handling.
- No replacement of AO Forge, AO2, AO Covenant, AO Command, or control-plane
  ownership boundaries.

## CLI

```sh
foundry pulse run \
  --registry examples/registry/local-ao-stack.foundry-registry.json \
  --task examples/tasks/ao-foundry-bootstrap.foundry-task.json \
  --goal-run examples/goals/ao-foundry-production-readiness.goal-run.json \
  --packet examples/packets/ao-foundry-bootstrap.factory-packet.json \
  --scorecard examples/evals/bootstrap.scorecard.json \
  --out tmp/pulse
```

Successful output ends with:

```text
pulse_event=<out>/pulse-event.json
status=ready
score=100/100
next_action=continue with governed AO Forge live execution when an executor is available
```

## Pulse Event Contract

The summary document uses `ao.foundry.pulse-event.v0.1` and includes:

- `pulse_id`,
- `status`,
- `score`,
- `max_score`,
- `registry_id`,
- `task_id`,
- `goal_id`,
- `artifacts`,
- `checks`,
- `next_action`.

`status=ready` requires every generated artifact check to pass and the competitive
audit to score 100/100. Any failed step must stop the loop, preserve artifacts
already written, write a failed pulse event when possible, and exit non-zero.

## Drift Controls

- Foundry schedules, audits, bundles, and recommends.
- Forge remains the delegated governed execution boundary.
- Covenant decisions are represented through packet/run evidence.
- AO2-style traces remain evidence, not authority.
- Release dry-run is a rehearsal artifact only and cannot publish.

## Verification

Required verification:

```sh
go test ./...
go run ./cmd/foundry pulse run --out tmp/pulse
go run ./cmd/foundry trace inspect --trace tmp/pulse/pulse.trace.jsonl
go run ./cmd/foundry competitive audit --out tmp/competitive-readiness-audit.json
run the repository public-safety scan over README.md, docs, examples, internal, and cmd
```

The safety scan may return no matches with exit code 1; that is the expected
safe result.
