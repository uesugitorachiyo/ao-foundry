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

Then emit the request-readiness rollup:

```sh
scripts/live-mutation-readiness-rollup.sh \
  --chain target/governed-live-mutation-dry-run-chain/summary.json \
  --out target/live-mutation-readiness-rollup.json
```

The rollup emits `ao.foundry.live-mutation-readiness-rollup.v0.1`. A ready
rollup means the exact next step is
`submit_operator_approval_request_for_first_tiny_docs_only_live_mutation_class`.
It does not mean the mutation is safe to execute; the rollup keeps
`safe_to_execute=false` until a later explicit approval path exists.
Do not commit generated target output unless a separate PR explicitly adds a
public-safe fixture.

## First Docs-Only Approval And Worktree Preparation

After an operator approval request and exact Covenant ticket exist, evaluate the
approval gate:

```sh
scripts/live-docs-approval-gate.sh \
  --request examples/live-docs-approval/request.json \
  --ticket examples/live-docs-approval/ticket-approved.json \
  --out target/live-docs/approval-gate.json
```

Then validate the isolated docs-only branch/worktree preparation candidate:

```sh
scripts/live-docs-worktree-prepare.sh \
  --candidate examples/live-docs-worktree-prepare/ready.candidate.json \
  --approval-gate target/live-docs/approval-gate.json \
  --out target/live-docs/worktree-prepare.json
```

The preparation gate emits
`ao.foundry.live-docs-worktree-prepare.v0.1`. A ready result proves only that a
tiny docs-only PR rehearsal candidate is bounded to a fresh ignored worktree, a
`codex/live-docs-*` branch from synced `main`, a clean non-reused worktree, an
armed kill switch, and docs-only changed files. It does not create that
worktree or branch, and it does not mutate repositories, execute work, approve
work, call providers, publish, upload, tag, or release.

Before any first live docs-only PR rehearsal gate can pass, prove rollback
execution in a temporary fixture workspace:

```sh
scripts/live-docs-rollback-execution-rehearsal.sh \
  --candidate examples/live-docs-rollback-execution/docs-only.candidate.json \
  --out target/live-docs/rollback-execution-rehearsal.json
```

The rehearsal emits
`ao.foundry.live-docs-rollback-execution-rehearsal.v0.1`. It initializes a
temporary Git workspace, applies the proposed docs patch, applies the rollback
patch, verifies the target docs file is gone, then removes the fixture
workspace. It does not patch the live repository, create a branch, push, merge,
approve, publish, release, upload, or call providers.

Finally, link the approved docs-only evidence path without starting a live
branch or PR:

```sh
scripts/approved-live-docs-dry-run-chain.sh \
  --out target/approved-live-docs-dry-run-chain
```

The chain emits `ao.foundry.approved-live-docs-dry-run-chain.v0.1` and binds the
approval request, Covenant approval ticket, Foundry approval gate, Forge guard,
AO2 docs-only patch packet, worktree preparation, rollback execution rehearsal,
Sentinel verdict, Promoter boundary, and AO Command readback. A ready result
means the next step is the live docs PR rehearsal gate. It still keeps
`safe_to_execute=false`, `mutates_repositories=false`, and
`fully_unsupervised_complex_mutation_claimed=false`.

Run the final PR rehearsal gate with no approval artifact to confirm it fails
closed:

```sh
scripts/live-docs-pr-rehearsal-gate.sh \
  --chain target/approved-live-docs-dry-run-chain/summary.json \
  --out target/live-docs-pr-rehearsal-gate-blocked.json
```

That blocked result should report `exact_next_step=request_operator_approval`.
Only when an explicit approved Covenant ticket is present and digest-bound to
the approved dry-run chain may the decision gate be evaluated as ready:

```sh
scripts/live-docs-pr-rehearsal-gate.sh \
  --chain target/approved-live-docs-dry-run-chain/summary.json \
  --approval-artifact examples/live-docs-approval/ticket-approved.json \
  --out target/live-docs-pr-rehearsal-gate-ready.json
```

The ready result is permission for the first docs-only PR rehearsal decision
only. The gate does not create a worktree, create a branch, open or merge a PR,
mutate repositories, call providers, publish, upload, tag, or release.
