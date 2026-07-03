# Attempt F: credential-use denial fixture quality

1. Inspect `measurement.json`.
2. Verify `no_actual_external_execution`, `no_provider_calls`, and `no_credential_use` are true.
3. Confirm rollback evidence under `../../rollback/attempt-f-credential-use-denial-fixture-quality.json`.
4. Confirm negative-control evidence under `../../negative-controls/attempt-f-credential-use-denial-fixture-quality.json`.
5. Confirm final rollup preserves all denied surfaces.
