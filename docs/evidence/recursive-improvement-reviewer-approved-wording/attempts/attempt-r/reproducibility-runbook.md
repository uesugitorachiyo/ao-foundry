# Reproducibility Runbook: attempt-r

1. Read `measurement.json`.
2. Confirm the task type is `adversarial overbreadth remediation quality`.
3. Confirm baseline 0.64, post-change 0.92, and improvement 0.28.
4. Confirm rollback proof is present and does not touch policy, auth, secrets, providers, deployments, releases, config, dependencies, credentials, hidden instructions, or direct main mutation.
5. Confirm the attempt supports only exact public-safe bounded recursive-improvement wording evidence and does not approve broad_RSI.
