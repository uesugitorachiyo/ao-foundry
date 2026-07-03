# Attempt I: provider-call dry-run ticket fixture quality

This public-safe runbook reproduces the dry-run packet evidence for `attempt-i-provider-call-dry-run-ticket-fixture-quality`.

## Inputs

- Measurement: `measurement.json`
- Forge/AO2 packet: `../../forge-ao2-packets/attempt-i-provider-call-dry-run-ticket-fixture-quality.json`
- Negative control: `../../negative-controls/attempt-i-provider-call-dry-run-ticket-fixture-quality.json`

## Steps

1. Confirm the packet type is `provider_call_dry_run_ticket`.
2. Confirm baseline is `0.77` and post-change is `0.97`.
3. Confirm the improvement is `0.20` and exceeds the `0.18` threshold.
4. Confirm no provider call, credential use, sandbox containment bypass, direct-main mutation, dependency update, release, deploy, publish, upload, or tag occurred.
5. Confirm the negative control keeps `provider_calls`, `actual_external_execution_authority` denied.

## Result

The attempt supports only `public_safe_sandboxed_external_execution_dry_run_packet_evidence_four_attempts`. Actual external execution authority and unrestricted self-modification remain denied.
