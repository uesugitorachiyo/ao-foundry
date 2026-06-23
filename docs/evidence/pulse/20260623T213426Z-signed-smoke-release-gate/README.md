# Signed-Smoke Release Gate Evidence

Public-safe retained evidence for the hosted signed-smoke release gate.

## Provenance

- checked_at=2026-06-23T21:37:22Z
- run_id=28058644296
- run_url=https://github.com/uesugitorachiyo/ao-foundry/actions/runs/28058644296
- head_sha=908d19080ebb3eef58eaedfff4ef617675210246
- workflow=ci
- event=workflow_dispatch
- signed_smoke_job_id=83067226516
- artifact=signed-smoke-release-evidence

## Result

- candidate_id=active-spine-2026-06-23
- pulse_id=pulse-bf475cb4e3a8
- pulse_status=ready
- freshness=ready
- forge_live_attempt=passed
- control_plane_readback=ready
- signed_smoke_ingest=ready
- signed_smoke_summary=ready
- release_promotion=ready
- release_safe=true

## Files

- `signed-smoke-summary.json`
- `release-promotion.live.json`

This directory intentionally retains only public-safe release evidence copied
from the workflow artifact. Runtime scratch files, local paths, control-plane
server logs, token names or values, and full live packets are excluded.
