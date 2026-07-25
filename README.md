# AO Foundry

AO Foundry coordinates engineering work across repositories and selects the
next safe portfolio action. It tracks tasks, readiness signals, branches, CI
state, and run evidence, then delegates one bounded run at a time.

## Role In AO

- **Inputs:** Blueprint-authorized work, Atlas imports, repository and CI
  state, task records, and completed-run evidence.
- **Outputs:** Portfolio status, readiness audits, next-work decisions, run
  records, and bounded Forge handoffs.
- **Upstream:** AO Blueprint for bounded work and AO Atlas for workgraph-driven
  work.
- **Downstream:** AO Forge, AO Mission, and AO Command.

See the [AO Architecture guide](https://github.com/uesugitorachiyo/ao-architecture)
and the
[AO Foundry component page](https://github.com/uesugitorachiyo/ao-architecture/blob/main/components/ao-foundry.md)
for the cross-repository flow.

## Quick Start

```sh
go test ./...
go build -o bin/foundry ./cmd/foundry
bin/foundry status --registry examples/registry/local-ao-stack.foundry-registry.json
bin/foundry registry validate \
  --registry examples/registry/local-ao-stack.foundry-registry.json
bin/foundry next \
  --registry examples/registry/local-ao-stack.foundry-registry.json \
  --task examples/tasks/ao-foundry-bootstrap.foundry-task.json
```

The [full command and readiness reference](REFERENCE.md) documents release
handoffs, Pulse gates, AO Mission smokes, Atlas imports, live-mutation
rehearsals, and generated active-stack snapshots.

## Coordination Boundary

Foundry selects and gates work at portfolio level. It does not replace
Blueprint authorization, Atlas workgraph compilation, Covenant policy, Forge
run state, or AO2 execution. A ready portfolio item still requires the
component-specific gates for its declared side effects.

## Documentation

- [AO Foundry v0.1 Design](docs/design/AO-FOUNDRY-V0.1.md)
- [One-Shot Factory Run](docs/operations/ONE-SHOT-FACTORY-RUN.md)
- [AO2 Pulse Event Loop](docs/operations/AO2-PULSE-EVENT-LOOP.md)
- [Branch Protection](docs/operations/BRANCH-PROTECTION.md)
- [Signed-Smoke Release Gate](docs/operations/SIGNED-SMOKE-RELEASE-GATE.md)
- [Production Readiness SDD](docs/sdd/AO-FOUNDRY-PRODUCTION-READINESS-SDD.md)
- [Pulse Golden Loop SDD](docs/sdd/AO-FOUNDRY-PULSE-GOLDEN-LOOP-SDD.md)
- [Full Reference](REFERENCE.md)

## Verification

```sh
go test ./...
go vet ./...
go build ./cmd/foundry ./cmd/ao
scripts/verify-branch-protection.sh
```

## Security

Treat public fixtures and evidence as publishable artifacts. Do not include
credentials, bearer tokens, machine-local paths, private logs, or non-public
operational notes. See [SECURITY.md](SECURITY.md).

## License

AO Foundry is licensed under Apache 2.0. See [LICENSE](LICENSE) and
[NOTICE](NOTICE).
