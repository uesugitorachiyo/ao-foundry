# Beta Rollback Drill Runbook

This runbook defines the fixture-only rollback drill used during the Month 6 beta launch preparation wave. It plans the operator sequence and expected evidence without executing a provider call, mutating a repository, opening or merging a PR, releasing, deploying, publishing, uploading, or tagging.

## Inputs

- `examples/contract-fixtures/valid/foundry-beta-rollback-drill-runbook-v0.1.json`
- `examples/contract-fixtures/valid/foundry-live-mutation-rollback-rehearsal-v0.1.json`
- `examples/contract-fixtures/valid/foundry-live-mutation-readiness-rollup-v0.1.json`
- `scripts/live-mutation-rollback-rehearsal.sh`

## Operator Sequence

1. Record clean checkout, current branch, current commit, and absence of release or deploy activity.
2. Select the fixture candidate for the dry-run rollback rehearsal.
3. Run the existing rollback rehearsal script in dry-run mode and store the JSON under `target/month6-beta/rollback-drill/`.
4. Compare the generated Foundry rollback evidence with Mission or Command readback.
5. Record closure with pass/fail gates, no-promotion status, and RSI denial.

## Pass Gates

- clean checkout recorded
- rollback patch is digest-bound
- kill switch state is armed
- observer readback agrees with fixture-only status
- no live mutation, release, deploy, publish, upload, tag, or provider call occurred

## Closure

The drill is complete only when the operator readback says `rollback_drill_planned_not_executed`, Promoter remains `no_promotion_requested`, and unrestricted RSI remains denied.
