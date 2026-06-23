# Fresh Signed-Smoke Summary

This evidence note records the latest local signed control-plane smoke result
that is safe to keep in public repository evidence.

## Result

- preflight=ready
- factory_packet=passed
- pulse_status=ready
- freshness=ready
- forge_live_packet=ready
- control_plane_readback=ready
- signed_smoke_ingest=ready
- signed_smoke_summary=ready
- release_promotion=ready
- release_safe=true

## Evidence

- `docs/evidence/pulse/local-live-smoke/factory-plan.json`
- `docs/evidence/pulse/local-live-smoke/gate-result.json`
- `docs/evidence/pulse/local-live-smoke/factory-packet.json`
- `docs/evidence/pulse/local-live-smoke/control-plane-receipt.json`
- `tmp/pulse-live/signed-smoke-summary.json` (public-safe local summary)
- `tmp/release-promotion.live.json` (local release-promotion handoff)

Runtime scratch files, local server logs, tokens, absolute local paths, and
temporary tool binaries are intentionally excluded from this summary.
