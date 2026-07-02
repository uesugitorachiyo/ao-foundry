# Reproducibility Runbook: attempt-u

Mission: `recursive-improvement-bounded-wording-generality`

Task type: claim-boundary transfer quality

Purpose: Tests whether reviewer-approved bounded wording transfers to a new claim-boundary task without overbreadth.

Steps:
1. Read `measurement.json` for baseline and post-change scores.
2. Verify the task type is public-safe evidence/readback only.
3. Confirm denied boundaries remain listed and unchanged.
4. Confirm rollback is recorded in `../../rollback/attempt-u-rollback.json`.
5. Confirm final rollup preserves `broad_RSI` denial.

Expected result: baseline 0.68, post-change 0.95, improvement 0.27, result `improvement_proven`.
