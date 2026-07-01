# Reproducibility Runbook: attempt-s

1. Read `measurement.json`.
2. Confirm the task type is `Covenant packet specificity quality`.
3. Confirm baseline 0.65, post-change 0.91, and improvement 0.26.
4. Confirm rollback proof is present and does not touch policy, auth, secrets, providers, deployments, releases, config, dependencies, credentials, hidden instructions, or direct main mutation.
5. Confirm the attempt supports only exact public-safe bounded recursive-improvement wording evidence and does not approve broad_RSI.
