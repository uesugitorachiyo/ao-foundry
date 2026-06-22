# Signed-Smoke Evidence Retention

Signed control-plane smoke runs create local operator evidence under
`docs/evidence/pulse/local-live-smoke` and runtime scratch under `tmp/`.

These files are useful for local audit, debugging, and pulse-loop scoring, but
they are not included in the release manifest. Keep them out of public release
packages unless an operator deliberately curates a public-safe summary.

Retention rules:

- Keep `tmp/` as disposable runtime scratch.
- Keep `docs/evidence/pulse/local-live-smoke` as local evidence only.
- Publish public-safe summaries in reviewed docs when the result matters for a
  release note or readiness report.
- Do not copy local tokens, private paths, server logs, or full control-plane
  scratch directories into public-safe summaries.
- Regenerate signed-smoke evidence when live packets or control-plane readback
  receipts are older than 24h.
