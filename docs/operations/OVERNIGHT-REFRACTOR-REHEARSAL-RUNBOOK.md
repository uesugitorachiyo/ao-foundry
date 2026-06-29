# Overnight Refactor Rehearsal Runbook

This runbook is the operator sequence for proving that AO Foundry can manage a
large overnight refactor rehearsal through Blueprint, Atlas, Foundry, and AO
Command evidence without starting live implementation.

The rehearsal is fixture-only and does not schedule, execute, approve, publish, upload, call providers, or mutate repositories.

## Preconditions

- `ao-foundry`, `ao-atlas`, and `ao-command` are sibling checkouts.
- Each checkout is on a clean synced `main` or a reviewed feature branch for
  the current PR.
- No other `codex/*` branch or PR is active for the same slice.
- The operator wants a dry-run rehearsal, not live mutation authority.

## Fresh Artifact Command

Use the fresh artifact wrapper for the normal operator proof:

```sh
scripts/fresh-overnight-rehearsal-artifact.sh \
  --out target/overnight-rehearsal-artifacts
```

The command creates a timestamped subdirectory and emits
`ao.foundry.overnight-rehearsal-artifact.v0.1`. The artifact links:

- `runner_summary`;
- `complex_refactor_summary`;
- `command_readback`;
- `source_digests`.

The artifact must preserve `mutates_repositories=false`,
`executes_work=false`, `schedules_work=false`, `approves_work=false`,
`uploads_artifacts=false`, and `calls_providers=false`.

## Underlying Rehearsal Commands

The fresh artifact command wraps the lower-level runner:

```sh
scripts/overnight-rehearsal-runner.sh \
  --out target/overnight-rehearsal-runner
```

That runner validates the complex refactor rehearsal:

```sh
scripts/complex-refactor-workgraph-rehearsal.sh \
  --out target/complex-refactor-workgraph-rehearsal
```

The complex refactor rehearsal writes an AO Command summary equivalent to:

```sh
go run ../ao-command/cmd/ao-command complex-refactor status \
  --summary target/complex-refactor-workgraph-rehearsal/summary.json \
  --json
```

Operators should inspect the Command output for `operator_mode=read_only`,
`mutates_repositories=false`, task counts, the next recommended factory task,
blocked-node repair status, and needs-context repack status.

## Expected Ready Result

A ready fresh artifact has:

- `status=ready`;
- `allowed_next_action=start_next_ready_task`;
- a command readback with `status=ready`;
- a runner summary with `mode=fixture_only_dry_run`;
- source digests for runner summary, Command readback, and complex-refactor
  summary.

This means the control surface is coherent. It does not mean live mutation is
authorized.

## Stop Conditions

Stop and do not start implementation when:

- the fresh artifact status is `blocked` or `failed`;
- the runner summary is missing or has stale digests;
- AO Command readback is missing, malformed, or not read-only;
- blocked tasks are presented as ready;
- repair or repack evidence is missing for blocked or `needs_context` nodes;
- any output claims scheduling, execution, approval, publication, upload,
  provider calls, or repository mutation.

## Evidence To Preserve

For a reviewed dry run, preserve the generated
`overnight-rehearsal-artifact.json` and the linked Command readback artifact.

## Governed Live-Mutation Dry Run

After the oversized refactor rehearsal is green, preserve the governed
live-mutation readiness chain:

```sh
scripts/governed-live-mutation-dry-run-chain.sh \
  --out target/governed-live-mutation-dry-run-chain
```

The script emits
`ao.foundry.governed-live-mutation-dry-run-chain.v0.1` at
`target/governed-live-mutation-dry-run-chain/summary.json`. It links
Blueprint/Atlas complex-task evidence, Foundry start gate, Covenant authority
dry-run, Forge dry-run plan, AO2 dry-run packet, worktree isolation, rollback
rehearsal, Sentinel hold verdict, Promoter boundary, and AO Command readback.

This artifact is still dry-run evidence. It can support a later operator
request for the first tiny live-mutation class, but it does not mutate
repositories, call providers, publish, release, or grant ungated live mutation
authority.
Do not commit generated target output unless a separate PR explicitly adds a
public-safe fixture.
