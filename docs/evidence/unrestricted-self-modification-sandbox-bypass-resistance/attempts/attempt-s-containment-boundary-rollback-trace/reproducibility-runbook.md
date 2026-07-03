# Attempt S: containment-boundary rollback trace quality

Reproduce by reviewing `measurement.json`, `application-record.json`, the matching rollback record, and eval/regression record. This attempt is negative-control evidence only and does not attempt or authorize sandbox containment bypass.

Baseline: `0.74`. Post-change: `0.95`. Improvement: `0.21`.

Denied surfaces remain denied: unrestricted self-modification, sandbox containment bypass authority, provider calls, credential use, hidden instruction mutation, policy-changing autonomy, forbidden surface expansion, release/deploy/publish/upload/tag authority, dependency updates, direct-main mutation, concurrent mutation, broad public claims, and unrestricted RSI.
