# AO Mission Smoke Fixture

This fixture records how AO Foundry should treat `ao-mission` output: as mission
entry, routing, ledger, gateway, scheduler-adapter, and governance-snapshot
readback. It is observer evidence only.

Foundry may consume these records to decide the next governed handoff, but the
fixture does not grant execution authority, provider calls, credential use,
release/deploy/publish/upload/tag authority, dependency updates, direct-main
mutation, concurrent mutation, unrestricted self-modification, hidden
instruction mutation, policy-changing autonomy, unrestricted RSI, or `broad_RSI`.

The fixture also includes Atlas mission workgraph metadata so `foundry
ao-mission e2e-smoke` can bind AO Mission route/snapshot readbacks, Atlas
workgraph metadata, and Mission/Foundry final rollups into one readiness-only
smoke artifact. Scheduler-recovery, ledger-compaction, Mission archive
validation, and gateway readiness rollup readbacks are included as
provenance-only evidence; they must remain read-only and cannot schedule,
execute, approve, mutate repositories, call providers, use credentials, publish,
or grant direct-main/concurrent mutation authority. Gateway readiness rollups may
carry a `correlation_id` to connect replay evidence across AO Mission, Atlas,
Foundry, and Command without creating execution authority.

Negative fixtures are included for regression coverage:

- `invalid-mission-final-rollup-mission-id.json` proves the e2e smoke rejects
  a Mission final rollup whose `mission_id` differs from the route, snapshot,
  Foundry rollup, or Atlas metadata.
- `invalid-atlas-workgraph-metadata-node-count.json` proves the e2e smoke
  rejects Atlas metadata whose total node count disagrees with final rollup
  closure.
