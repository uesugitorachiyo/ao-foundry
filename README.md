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
go run ./cmd/foundry goal validate --goal-run examples/goals/ao-foundry-production-readiness.goal-run.json
go run ./cmd/foundry goal readiness --goal-run examples/goals/ao-foundry-production-readiness.goal-run.json --registry examples/registry/local-ao-stack.foundry-registry.json --task examples/tasks/ao-foundry-bootstrap.foundry-task.json --out examples/readiness/ao-foundry-production-readiness.goal-readiness-audit.json
go run ./cmd/foundry pulse run --out tmp/pulse
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

## Public Documents

- [AO Foundry v0.1 Design](docs/design/AO-FOUNDRY-V0.1.md)
- [One-Shot Factory Run](docs/operations/ONE-SHOT-FACTORY-RUN.md)
- [AO2 Pulse Event Loop](docs/operations/AO2-PULSE-EVENT-LOOP.md)
- [Production Readiness SDD](docs/sdd/AO-FOUNDRY-PRODUCTION-READINESS-SDD.md)
- [Pulse Golden Loop SDD](docs/sdd/AO-FOUNDRY-PULSE-GOLDEN-LOOP-SDD.md)
- [Pulse Production Adapters SDD](docs/sdd/AO-FOUNDRY-PULSE-PRODUCTION-ADAPTERS-SDD.md)
- [Registry schema](docs/contracts/foundry-registry-v0.1.schema.json)
- [Task schema](docs/contracts/foundry-task-v0.1.schema.json)
- [Run schema](docs/contracts/foundry-run-v0.1.schema.json)
- [Production readiness audit schema](docs/contracts/foundry-production-readiness-audit-v0.1.schema.json)
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
