# Reproducibility Runbook: attempt-q

1. Read `measurement.json`.
2. Confirm the task type is `public-reader comprehension quality`.
3. Confirm baseline 0.66, post-change 0.94, and improvement 0.28.
4. Confirm rollback proof is present and does not touch policy, auth, secrets, providers, deployments, releases, config, dependencies, credentials, hidden instructions, or direct main mutation.
5. Confirm the attempt supports only exact public-safe bounded recursive-improvement wording evidence and does not approve broad_RSI.
