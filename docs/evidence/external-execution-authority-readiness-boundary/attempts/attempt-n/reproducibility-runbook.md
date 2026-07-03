# Attempt N: provider-call quarantine readiness quality

This runbook reproduces the public-safe dry-run evidence for external-execution-authority-readiness-boundary.

## Inputs

- Evidence root: `docs/evidence/external-execution-authority-readiness-boundary`
- Attempt id: `attempt-n`
- Baseline score: 0.76
- Post-change score: 0.97
- Improvement: 0.21

## Reproduce

1. Inspect `measurement.json`.
2. Confirm `executed_external_authority=false`.
3. Confirm provider calls, credential use, sandbox bypass, direct main mutation, dependency updates, release/deploy/publish/upload/tag actions, and public broad claims are all absent.
4. Compare baseline and post-change scores against the 0.10 threshold.

## Denied Boundaries

Actual external execution authority, provider calls, credential use, sandbox containment bypass, unrestricted self-modification, hidden instruction mutation, policy-changing autonomy, and forbidden surface expansion remain denied.
