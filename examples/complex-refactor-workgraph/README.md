# Complex Refactor Workgraph Rehearsal

This fixture models an oversized AO stack refactor as Atlas factory tasks. It is
public-safe and rehearsal-only: it does not schedule, execute, approve, publish,
call providers, duplicate AO stack folders, or mutate sibling repositories.

The workgraph intentionally includes completed, ready, and blocked nodes:

- completed: architecture and boundary audit;
- ready: Pulse runner module split and Atlas/Foundry fixture path;
- blocked: AO Command readback follow-up and final stitch/integration task.

Run the rehearsal from the AO Foundry repo root:

```sh
scripts/complex-refactor-workgraph-rehearsal.sh \
  --out docs/evidence/pulse/complex-refactor-workgraph-rehearsal-local
```

The output summary reports task counts, next recommended factory task, and why
the overnight loop may or may not start.
