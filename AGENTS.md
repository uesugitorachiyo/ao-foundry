# AO Foundry Agent Instructions

## Status And Role

AO Foundry is the active portfolio coordinator for bounded engineering work. It owns registry and task validation, readiness aggregation, next-safe-work selection, run records, and bounded handoffs to Forge.

Foundry consumes Blueprint authorization, Atlas imports, repository and CI state, policy/readback evidence, and completed-run evidence. It does not replace those producers, implement their contracts, execute AO2 side effects, approve policy, mutate a target merely because it is ready, or publish a release.

## Sources Of Truth

- [docs/design/AO-FOUNDRY-V0.1.md](docs/design/AO-FOUNDRY-V0.1.md) defines the portfolio model and component boundary.
- [docs/sdd/AO-FOUNDRY-PRODUCTION-READINESS-SDD.md](docs/sdd/AO-FOUNDRY-PRODUCTION-READINESS-SDD.md) and [docs/sdd/AO-FOUNDRY-GOAL-RUN-READINESS-SDD.md](docs/sdd/AO-FOUNDRY-GOAL-RUN-READINESS-SDD.md) define readiness and GoalRun behavior.
- `docs/contracts/`, `internal/cli/`, and their tests own schemas and implemented command behavior. [REFERENCE.md](REFERENCE.md) is the current command reference.
- [docs/operations/ONE-SHOT-FACTORY-RUN.md](docs/operations/ONE-SHOT-FACTORY-RUN.md), [docs/operations/AO2-PULSE-EVENT-LOOP.md](docs/operations/AO2-PULSE-EVENT-LOOP.md), and [`.github/workflows/ci.yml`](.github/workflows/ci.yml) define bounded operation and CI gates.

## Ownership And Boundaries

- Keep every next-work decision bound to exact repository state, task scope, approvals, policy, source heads, and evidence digests. Missing, stale, mismatched, or over-authority inputs must block.
- Coordinate one bounded run at a time. A readiness audit, dry run, observer readback, generated prompt, or historical completion does not grant target mutation, live execution, release, or publication authority.
- Preserve `docs/evidence/`, release records, committed readiness ledgers, and contract fixtures as historical or source-owned material. Do not rewrite them to make a current claim pass.
- Keep generated snapshots, reports, tools, and run output in ignored `tmp/`, `bin/`, or `dist/`. Never hand-edit generated evidence to satisfy a gate.
- Do not record credentials, bearer values, private logs, account identifiers, machine-local paths, or non-public operational notes. Provider, credential, permission, and target-repository authority remain explicit and bounded.
- Release, deployment, publication, live mutation, credentialed operation, and direct-main changes require separate explicit authority and all executable gates.

## Working Method

- Change the smallest coordination surface and preserve producer ownership, provenance, freshness, rollback, single-active-run, and fail-closed rules.
- Add negative coverage for stale readbacks, digest drift, missing approvals, invalid paths, conflicting ownership, or claimed authority beyond a producer's contract.
- The completed CLI decomposition-readiness assessment is immutable prerequisite evidence, not pilot authority. This guidance-only change advances the Foundry source head; a later pilot handoff must revalidate the assessment's recorded 17-declaration and test boundary as byte-equivalent at the new head and obtain separate user approval before implementation.
- Update this file in the same pull request when durable commands, architecture, ownership, or authority boundaries change.

## Verification

- Foundry coordination or command changes: `go test ./internal/cli -count=1`.
- Format relevant Go source with `gofmt -d cmd internal`; run `go test ./... -count=1`, `go vet ./...`, and `go build -o bin/foundry ./cmd/foundry` plus `go build -o bin/ao ./cmd/ao`.
- Run `scripts/verify-branch-protection.sh` when repository protection or required-check documentation changes. Live-mutation, Pulse, signed-smoke, release, and publication commands require separate authority and are not instruction checks.
- For instruction changes run `python3 ../ao-architecture/scripts/verify_agent_instruction_layout.py --workspace-root .. --repository ao-foundry`. Always run `git diff --check`.

## Evidence And Completion

- Record source and target heads, commands and exits, approval and policy bindings, and relevant artifact digests. Report skipped, unavailable, stale, or failed checks explicitly.
- Completion requires focused and broad gates, green pull-request CI, clean synchronized `main`, and task-branch cleanup. A green readiness result does not widen the authorized task.
