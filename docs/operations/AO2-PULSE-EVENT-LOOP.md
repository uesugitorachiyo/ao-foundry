# AO2 Pulse Event Loop For AO Foundry

AO Foundry should use an AO2 Pulse-style loop as a scheduler, not as an
authority boundary. The scheduler may trigger a Foundry readiness audit and
select the next queued task, but Foundry must still delegate governed execution
to AO Forge.

The next pulse refactor must follow the current stack intake order:
AO Blueprint handles requirements sufficiency and build authorization, AO Atlas
compiles oversized authorized objectives into workgraphs and bounded context
packs, AO Foundry schedules the next safe ready item, AO Forge runs one
governed factory job, AO Covenant gates side effects, AO2 executes, and
readback surfaces observe. See
[`../sdd/AO-FOUNDRY-PULSE-BLUEPRINT-ATLAS-REFACTOR.md`](../sdd/AO-FOUNDRY-PULSE-BLUEPRINT-ATLAS-REFACTOR.md).

## v0.1 Loop

```text
load registry and task
-> foundry readiness audit
-> foundry goal readiness
-> if score < 100, stop and report blockers
-> if score = 100, foundry next
-> create a Forge factory brief
-> delegate execution to AO Forge
-> retain the Forge packet and policy gate summary
-> record Forge packet in a Foundry run record
-> score the run with local evals
-> derive RSI candidate, improvement gate, and next improvement task evidence
-> inspect a local trace
-> write a pulse-event summary
```

Run the local clean-clone-safe pulse:

```sh
go run ./cmd/foundry pulse intake-preflight \
  --blueprint-authorization examples/pulse-intake/blueprint-authorization.ready.json \
  --requires-atlas \
  --atlas-import examples/atlas/foundry-import.json \
  --atlas-status examples/contract-fixtures/valid/foundry-atlas-status-v0.1.json \
  --out tmp/pulse-intake-preflight.json
go run ./cmd/foundry pulse lifecycle inspect \
  --state examples/pulse-lifecycle/ready-to-start-next-slice.json \
  --json
go run ./cmd/foundry pulse overnight-start-gate \
  --intake-preflight examples/pulse-overnight-start-gate/ready.intake-preflight.json \
  --lifecycle examples/pulse-lifecycle/ready-to-start-next-slice.json \
  --out tmp/pulse-overnight-start-gate.json
go run ./cmd/foundry pulse run \
  --start-gate examples/pulse-overnight-start-gate/ready.json \
  --out tmp/pulse
go run ./cmd/foundry pulse freshness --pulse tmp/pulse/pulse-event.json
go run ./cmd/foundry trace inspect --trace tmp/pulse/pulse.trace.jsonl
go run ./cmd/ao status
go run ./cmd/ao run --out tmp/ao-pulse
```

The command first writes `tmp/pulse/pulse-runner-start-decision.json` from the
Pulse overnight start gate. A ready decision is required before the runner
generates implementation evidence. Blocked or failed decisions exit non-zero and
stop before `pulse-event.json` is produced.

After a ready decision, the command writes `tmp/pulse/pulse-event.json` plus the
generated readiness, GoalRun, Forge brief, Forge packet, policy gate, live Forge
attempt, control-plane readback, run, eval, RSI candidate, RSI improvement gate,
RSI next improvement task, trace, demo, release dry-run, and competitive audit
artifacts. The loop exits non-zero and still writes a blocked event when a
readiness gate fails after the start gate. The command also prints an operator
status line such as
`freshness=ready forge_live_packet=not_provided control_plane_readback=not_provided`.
The same values are recorded in `pulse-event.json` under `freshness_summary`.

To prove the full dry-run control path with AO Command readback, run:

```sh
scripts/blueprint-atlas-pulse-e2e-dry-run.sh \
  --out docs/evidence/pulse/blueprint-atlas-pulse-e2e-local
```

The script consumes public fixtures for Blueprint ready authorization,
Blueprint blocked clarification, Atlas import/run-link evidence, Foundry Pulse
preflight/lifecycle/start-gate evidence, runner start decisions, and AO Command
`pulse status` readback. It proves that the ready path may start the runner and
that the blocked Blueprint path cannot produce `pulse-event.json`. It is
fixture-only and does not schedule, execute, approve, publish, call providers,
or mutate repositories.

To rehearse a larger refactor as bounded Atlas factory tasks, run:

```sh
scripts/complex-refactor-workgraph-rehearsal.sh \
  --out docs/evidence/pulse/complex-refactor-workgraph-rehearsal-local
scripts/overnight-rehearsal-runner.sh \
  --out docs/evidence/pulse/overnight-rehearsal-runner-local
```

The rehearsal validates `examples/complex-refactor-workgraph/workgraph.json`,
its context packs, the Foundry import/readback fixture, and the Pulse gate e2e
proof. Its summary reports total, ready, blocked, completed, and failed task
counts, blocked-node repair-plan output, needs-context repack output, AO
Command read-only status output, the next recommended factory task, and why the
loop may start the next ready task while blocked tasks remain denied.

The overnight rehearsal runner wraps the complex-refactor rehearsal and emits
`ao.foundry.overnight-rehearsal-runner.v0.1`. It is dry-run only: it validates
the start gate, lifecycle state, Atlas import/readback, repair/repack artifacts,
and AO Command status before reporting `allowed_next_action`.

The intake preflight writes `tmp/pulse-intake-preflight.json` with
`schema_version=ao.foundry.pulse-intake-preflight.v0.1`. It returns success only
when Blueprint authorization is ready and required Atlas import/status readback
artifacts preserve `schedules_work=false`, `executes_work=false`, and
`approves_work=false`. A Blueprint clarification request returns a blocked
preflight instead of pretending work is ready.

The lifecycle inspect command reads
`ao.foundry.pulse-pr-lifecycle.v0.1` and reports whether Pulse may start another
slice. It fails closed when the current slice has an open PR, pending or failed
checks, incomplete merged-branch cleanup, unsynced main, dirty worktree state,
or multiple active `codex/*` branches. It is a local inspection gate only; it
does not branch, push, merge, delete branches, schedule work, or execute work.

The overnight start gate reads the preflight and lifecycle artifacts, writes
`tmp/pulse-overnight-start-gate.json`, and is the required precondition before
autonomous overnight advancement. It emits
`ao.foundry.pulse-overnight-start-gate.v0.1`, requires digest-bound
Blueprint/Atlas evidence, fails closed on failed preflight, stale digests,
pending or failing checks, incomplete cleanup, unsynced main, and dirty
worktrees, and cleanly blocks for Blueprint clarification when implementation
is not being started. It does not start a loop or mutate repositories.

`foundry pulse run` enforces that start gate through
`--start-gate <pulse-overnight-start-gate.json>`. The default public fixture is
ready for local smoke runs, but production overnight operation should pass the
freshly generated gate result for the current intake and lifecycle state.

The RSI sequence is a read-only evidence loop. AO Foundry produces the
candidate, improvement gate, and next-task artifacts that support the
`bounded_governed_rsi` claim, then AO Command RSI health and AO Covenant retain
the public claim boundary. `full_autonomous_self_mutating_rsi` remains denied
until the stack has mutation authority, rollback, and live self-change evidence.

Derive the next AO2 loop decision from the pulse output:

```sh
go run ./cmd/foundry pulse derive-next \
  --pulse tmp/pulse/pulse-event.json \
  --audit tmp/pulse/competitive-audit.json \
  --out tmp/pulse/ao2-loop-decision.json
```

The derived decision includes `event_loop.freshness` from the pulse event.
Blocked freshness takes precedence over generic failed checks: stale Forge live
packets derive `refresh-forge-live-packet`, stale control-plane readbacks derive
`refresh-control-plane-readback`, and other production-evidence freshness
failures derive `resolve-production-evidence-freshness`. If freshness is not
blocked, the decision uses the first failed pulse check for blocked events, then
competitive audit next actions for ready events, and finally the pulse
`next_action` text.

To bundle externally produced production evidence:

```sh
go run ./cmd/foundry pulse run --out tmp/pulse \
  --forge-live-packet path/to/factory-packet.json \
  --control-plane-receipt path/to/control-plane-receipt.json
```

## Signed Control-Plane Smoke

Use this local operator smoke when the AO sibling repositories are checked out
next to AO Foundry and the AO Forge/Covenant binaries can be built locally. The
token value is an operator-provided local secret with at least 32 characters; do
not commit it or copy it into evidence.

```sh
AO2_CP_API_TOKEN=<local-token> ../ao2-control-plane/target/debug/ao2-cp-server --bind 127.0.0.1:18746 \
  --data-dir tmp/control-plane
```

In another shell, build temporary local tools, create the Foundry pulse brief,
run AO Forge against the local control-plane observer, and rerun the Foundry
pulse from the live packet:

```sh
go run ./cmd/foundry pulse signed-smoke-preflight --workspace .. \
  --out tmp/signed-smoke-preflight.json
go run ./cmd/foundry pulse run --out tmp/pulse
mkdir -p tmp/live-tools
(cd ../ao-forge && go build -o ../ao-foundry/tmp/live-tools/forge ./cmd/forge)
(cd ../ao-covenant && go build -o ../ao-foundry/tmp/live-tools/covenant ./cmd/covenant)
tmp/live-tools/forge plan \
  --brief tmp/pulse/forge-brief.json \
  --out docs/evidence/pulse/local-live-smoke/factory-plan.json
tmp/live-tools/forge gate \
  --plan docs/evidence/pulse/local-live-smoke/factory-plan.json \
  --covenant tmp/live-tools/covenant \
  --out docs/evidence/pulse/local-live-smoke/gate-result.json
AO2_CP_API_TOKEN=<local-token> tmp/live-tools/forge run \
  --plan docs/evidence/pulse/local-live-smoke/factory-plan.json \
  --gate-result docs/evidence/pulse/local-live-smoke/gate-result.json \
  --out docs/evidence/pulse/local-live-smoke/factory-packet.json \
  --control-plane http://127.0.0.1:18746 \
  --live --non-interactive --no-dashboard
go run ./cmd/foundry pulse run \
  --out tmp/pulse-live \
  --forge-live-packet docs/evidence/pulse/local-live-smoke/factory-packet.json
go run ./cmd/foundry pulse ingest-signed-smoke \
  --result tmp/pulse-live/signed-smoke-result.json \
  --out tmp/pulse-live/signed-smoke-ingest.json
go run ./cmd/foundry pulse run \
  --out tmp/pulse-live-bundled \
  --forge-live-packet docs/evidence/pulse/local-live-smoke/factory-packet.json \
  --signed-smoke-result tmp/pulse-live/signed-smoke-result.json
go run ./cmd/foundry pulse summarize-signed-smoke \
  --pulse tmp/pulse-live-bundled/pulse-event.json \
  --out tmp/pulse-live-bundled/signed-smoke-summary.json
go run ./cmd/foundry pulse signed-smoke-cleanup
```

The final `tmp/pulse-live/pulse-event.json` should report
`forge_live_attempt=passed`, `control_plane_readback=ready`, and
`freshness_summary.status=ready`. The bundled
`tmp/pulse-live-bundled/pulse-event.json` should also include a
`signed_smoke_ingest` artifact. The summary omits source paths, digests, tokens,
server logs, and runtime scratch details. Stop the local control-plane server
after the smoke completes. `foundry pulse signed-smoke-cleanup` removes signed
smoke scratch under `tmp/` while preserving local audit evidence under
`docs/evidence/pulse/local-live-smoke`.

CI keeps a public-safe fixture for the successful signed-smoke freshness shape
at `examples/ci/signed-smoke-freshness.pulse-event.json`. Validate it locally
without credentials:

```sh
go test ./internal/cli -run TestSignedSmokeFreshnessCIFixtureValidates -v
```

Operator-provided live packets older than 24h are treated as stale production
evidence. Rerun the signed smoke before using an older packet in readiness
scoring.

Control-plane readback receipts older than 24h are also treated as stale.
Foundry compares discovered receipt digests against the Forge packet evidence
before scoring the readback as ready.

## Stop Conditions

- Registry or task contract validation fails.
- A target repository is not registered.
- Any target readiness signal is not `ready`.
- The task does not delegate governed work to AO Forge.
- The task is not local-only.
- Verification commands are missing.
- GoalRun evidence uses unsafe paths.
- GoalRun evidence digests are missing or stale.
- GoalRun phase is terminal.
- GoalRun next-action guard allows direct provider execution or sibling repo mutation.

## Next Loop Hardening

The next production-readiness slice should make pulse intake Blueprint/Atlas
aware and enforce the one-slice PR lifecycle before adding broader live
execution behavior. The loop should refuse direct provider execution, preserve
blocked events, keep delegated implementation inside AO Forge, and allow only
one active branch/PR/check cycle at a time.
