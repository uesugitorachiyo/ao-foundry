# Reproducibility Runbook: Security/public-safety scan quality

1. Inspect `docs/evidence/recursive-improvement-public-evidence-expansion/attempts/attempt-f-security-public-safety-scan-quality/baseline-measurement.json` and record the baseline score.
2. Inspect `docs/evidence/recursive-improvement-public-evidence-expansion/attempts/attempt-f-security-public-safety-scan-quality/bounded-application.json` and confirm the change stayed inside docs/evidence/readback support scope.
3. Inspect `docs/evidence/recursive-improvement-public-evidence-expansion/attempts/attempt-f-security-public-safety-scan-quality/post-change-measurement.json` and recompute improvement as post-change minus baseline.
4. Inspect `docs/evidence/recursive-improvement-public-evidence-expansion/attempts/attempt-f-security-public-safety-scan-quality/eval-regression-proof.json` and confirm all regression checks passed.
5. Inspect `docs/evidence/recursive-improvement-public-evidence-expansion/attempts/attempt-f-security-public-safety-scan-quality/rollback-retraction-proof.json` and confirm the rollback path is reversible.
6. Confirm no artifact claims `broad_RSI` or stronger recursive-improvement wording approval.

Expected result: improvement >= 0.15; actual improvement = 0.26.
