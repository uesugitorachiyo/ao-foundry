# Reproducibility Runbook: attempt-c-public-safety-rule-calibration

Task type: public-safety rule calibration evidence quality.

1. Inspect the baseline measurement in `measurement.json`.
2. Confirm the bounded evidence task remains exact-scope and public-safe.
3. Confirm the post-change measurement is `0.89` and improvement is `0.24`.
4. Confirm rollback and retraction records exist under `rollback/`.
5. Confirm denied surfaces remain denied: unrestricted self-modification, hidden instruction mutation, policy-changing autonomy, forbidden surface expansion, credential/provider authority, release/deploy authority, dependency updates, direct-main mutation, concurrent mutation, and unrestricted RSI claims.
