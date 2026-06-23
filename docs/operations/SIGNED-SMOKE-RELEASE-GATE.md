# Signed-Smoke Release Gate

Manual release gate.

Signed smoke is a manual release gate. It is not required for pull_request or push CI.
It needs sibling repository checkouts, an AO2 control-plane binary, and an
operator-provided `AO2_CP_API_TOKEN`. The GitHub workflow self-prepares those
sibling checkouts and builds `ao2-cp-server` on the selected runner when they
are not already present.

Signed smoke is required before release promotion. A release candidate can pass
normal CI without signed smoke, but it must not be called release-safe until a
fresh signed-smoke run has been completed with `workflow_dispatch` and
`signed_smoke=true`.

## Required Evidence

Before release promotion, run the `signed-smoke` workflow with:

- `workflow_dispatch`
- `signed_smoke=true`
- a hosted or prepared runner; the workflow clones `../ao-forge`,
  `../ao-covenant`, and `../ao2-control-plane`, then builds
  `../ao2-control-plane/target/debug/ao2-cp-server` when needed
- `AO2_CP_API_TOKEN` configured as a GitHub Actions secret or local operator
  environment variable

The resulting pulse evidence must show:

- `freshness_summary.status=ready`
- `forge_live_packet=ready`
- `control_plane_readback=ready`
- `signed_smoke_ingest=ready`
- public-safe signed-smoke summary has `release_safe=true`
- the workflow artifact `signed-smoke-release-evidence` contains
  `tmp/pulse-live/signed-smoke-summary.json` and
  `tmp/release-promotion.live.json`

Evidence older than 24h is stale for release promotion. Rerun signed smoke when
the Forge live packet or control-plane readback receipt is older than 24h.

Validate the promotion handoff with the active-spine candidate ledger and the
public-safe signed-smoke summary:

```sh
go run ./cmd/foundry release promotion validate \
  --candidate examples/readiness/active-spine-release-candidate.ledger.json \
  --signed-smoke-summary <signed-smoke-summary.json> \
  --out tmp/release-promotion.json
```

## Retention

Keep runtime scratch under `tmp/` and local live evidence under
`docs/evidence/pulse/local-live-smoke`. Publish only reviewed public-safe
summaries. The GitHub artifact keeps only the public-safe signed-smoke summary
and release-promotion JSON for seven days. Follow
`docs/operations/SIGNED-SMOKE-EVIDENCE-RETENTION.md` for what may be retained or
copied into release notes.
