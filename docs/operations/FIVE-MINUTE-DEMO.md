# AO Foundry Five-Minute Demo

## Positioning

AO Foundry is the engineering operations factory above AO Forge. It coordinates
registries, tasks, readiness, runs, evals, traces, and scheduler gates. It does
not replace AO Forge; governed implementation runs are delegated to AO Forge.

## Flow

1. Show the local AO stack registry with `foundry status`.
2. Validate the bootstrap task and GoalRun readiness.
3. Emit an AO Forge brief with `foundry next --out`.
4. Inspect the AO Forge packet fixture as governed execution evidence.
5. Ingest the packet into a Foundry run record.
6. Score the run with `foundry eval run`.
7. Show the next safe action with `foundry demo status`.

## Guardrails

- No credentials are required.
- No network access is required.
- No release, tag, push, upload, or sibling-repository mutation is performed.
- Internal coordination material is not part of the public demo.
