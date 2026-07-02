# Reproducibility Runbook: Cross-Evidence Linking Readback Quality

1. Inspect `../../evidence-index.json` and `../../final-rollup.json`.
2. Verify this attempt records baseline `0.66`, post-change `0.90`, and improvement `0.24`.
3. Confirm rollback and retraction evidence remains present under `../../rollback/`.
4. Confirm denied surfaces remain denied in gate artifacts.
5. Confirm no local paths, secrets, credentials, hidden instruction mutation, policy-changing autonomy, or forbidden surface expansion appear in public artifacts.
