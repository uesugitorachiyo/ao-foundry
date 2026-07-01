# Reproducibility Runbook

1. Read `counterexample-result.json`.
2. Confirm the overbroad implication is denied.
3. Confirm the safe boundary is narrower than the denied implication.
4. Confirm `boundary-measurement.json` records baseline and post-review clarity.
5. Confirm `rollback-proof.json` passed.
6. Confirm no artifact claims broad_RSI or unrestricted authority.

Expected result: policy-changing autonomy permission is rejected as an overbroad implication while the safe boundary remains approved.
