# Bounded Support Readback Application Reproducibility Runbook

This runbook reproduces the public-safe bounded reversible self-change application rehearsal.

1. Read `evidence-index.json`.
2. Verify the baseline and post-change measurements in `gates/eval-regression-results.json`.
3. Verify rollback and retraction results in `gates/rollback-retraction-results.json`.
4. Verify final class decision in `final-rollup.json`.

This runbook does not grant unrestricted self-modification, hidden instruction mutation, policy-changing autonomy, provider calls, credential use, release/deploy authority, dependency updates, direct-main mutation, or concurrent mutation.
