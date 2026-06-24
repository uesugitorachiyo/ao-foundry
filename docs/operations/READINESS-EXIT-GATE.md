# Readiness Exit Gate

The active-stack readiness loop stops when Foundry reports production readiness
at 100/100 and the active-stack loop has no blocking next actions.

This is the exit condition for autonomous readiness work:

- `foundry goal readiness` reports `status=ready` and `score=100/100`.
- `foundry competitive audit` reports `status=ready` and `score=100/100`.
- `scripts/active-stack-readiness-loop.sh` reports `status=passed`.
- `blocking_next_actions` is empty.

When those conditions hold, automation must stop. Live execution, release
promotion, signed-smoke dispatch, or new implementation work requires explicit
operator intent.

`maintenance_suggestions` are not blockers. They are reminders for future
manual or scheduled review, such as keeping the active registry scoped to
`ao-foundry`, `ao-forge`, `ao-command`, `ao2`, `ao2-control-plane`, and
`ao-covenant`, or refreshing readiness evidence after a merged readiness PR.

ao2-control-plane remains a read-only observer in this exit gate. It may run
scheduled or manual verification workflows, but those workflows must not approve
AO2 runs, mutate AO artifacts, publish releases, or continue a Pulse/event loop
automatically.
