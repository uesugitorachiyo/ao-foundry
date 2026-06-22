# One-Shot Factory Run

This runbook describes how AO Forge should bootstrap AO Foundry v0.1 safely.

## Inputs

- SDD prompt describing AO Foundry v0.1.
- Factory brief that targets the AO Foundry workspace.
- Covenant binary built locally for Forge gating.
- Existing AO Foundry seed workspace.

## Preflight

Run the local checks before planning:

```sh
git status --short --branch
go test ./...
../ao2/target/debug/ao2 sdd --help
uv run --project ../agy-swarms agy-swarms --help
```

Preflight should confirm a clean AO Forge worktree, passing Forge tests, an AO2
SDD command surface, and an available agy-swarms CLI.

## SDD Plan

Generate or recover an SDD plan for the Foundry target, then validate and
dry-run dispatch it:

```sh
../ao2/target/debug/ao2 sdd validate --plan <kit>/ao-foundry.sdd-plan.json
../ao2/target/debug/ao2 sdd dispatch --plan <kit>/ao-foundry.sdd-plan.json --runner ao2 --out <kit>/ao-foundry.sdd-run.yaml --dry-run
```

If the provider protocol is unavailable, record the blocker in the one-shot kit
and create a deterministic fallback plan from the prompt. The fallback plan must
still pass AO2 validation and dry-run dispatch.

## Forge Gate

Build the local Covenant binary into the kit, then run Forge plan and gate:

```sh
go run ./cmd/forge plan --brief <kit>/ao-foundry.factory-brief.json --out <kit>/ao-foundry.factory-plan.json
go run ./cmd/forge gate --plan <kit>/ao-foundry.factory-plan.json --covenant <kit>/covenant --out <kit>/ao-foundry.gate.json
```

The gate must fail closed unless Covenant allows the local plan.

## Live Run

Run Forge live and non-interactive:

```sh
go run ./cmd/forge run --plan <kit>/ao-foundry.factory-plan.json --gate-result <kit>/ao-foundry.gate.json --out <kit>/ao-foundry.packet.json --live --non-interactive --no-dashboard
```

A zero exit and packet are not enough. Inspect the target repository. If the
target remains a seed scaffold, complete the SDD directly inside AO Foundry and
preserve the Forge packet as evidence.

## Verification

Inside AO Foundry:

```sh
go test ./...
```

From the Forge workspace, scan AO Foundry for local paths, credential-like
strings, and non-public coordination language. Then check AO Forge status to
confirm only allowed kit files changed.

## Closeout

Write the one-shot result in the kit with command status, Forge automation
outcome, files produced, safety scan result, and next production-readiness
tasks. Do not push, tag, publish, or upload artifacts as part of the one-shot.
