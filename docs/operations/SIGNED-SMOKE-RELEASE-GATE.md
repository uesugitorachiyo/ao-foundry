# Signed-Smoke Release Gate

Manual release gate.

Signed smoke is a manual release gate. It is not required for pull_request or push CI.
It needs a prepared AO workspace, sibling repository checkouts, local binaries,
and an operator-provided `AO2_CP_API_TOKEN`.

Signed smoke is required before release promotion. A release candidate can pass
normal CI without signed smoke, but it must not be called release-safe until a
fresh signed-smoke run has been completed with `workflow_dispatch` and
`signed_smoke=true`.

## Required Evidence

Before release promotion, run the `signed-smoke` workflow with:

- `workflow_dispatch`
- `signed_smoke=true`
- a prepared runner with `../ao-forge`, `../ao-covenant`, and
  `../ao2-control-plane/target/debug/ao2-cp-server`
- `AO2_CP_API_TOKEN` configured as a GitHub Actions secret or local operator
  environment variable

The resulting pulse evidence must show:

- `freshness_summary.status=ready`
- `forge_live_packet=ready`
- `control_plane_readback=ready`
- `signed_smoke_ingest=ready`
- public-safe signed-smoke summary has `release_safe=true`

Evidence older than 24h is stale for release promotion. Rerun signed smoke when
the Forge live packet or control-plane readback receipt is older than 24h.

## Retention

Keep runtime scratch under `tmp/` and local live evidence under
`docs/evidence/pulse/local-live-smoke`. Publish only reviewed public-safe
summaries. Follow `docs/operations/SIGNED-SMOKE-EVIDENCE-RETENTION.md` for
what may be retained or copied into release notes.
