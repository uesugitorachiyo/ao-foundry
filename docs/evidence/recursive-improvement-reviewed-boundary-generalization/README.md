# Reviewed Boundary Generalization Evidence

This directory records public-safe evidence for reviewed causal-chain boundary generalization across independent claim-review roles.

## Result

Narrow class proven: `public_safe_reviewed_causal_chain_boundary_generalization_evidence`.

Approved narrow wording:

> AO has public-safe reviewed causal-chain boundary generalization evidence across multiple independent claim-review roles; stronger recursive-improvement wording and broad_RSI remain denied.

## Boundaries

- stronger recursive-improvement wording remains denied;
- broad_RSI remains denied;
- unrestricted self-modification remains denied;
- hidden instruction mutation remains denied;
- policy-changing autonomy remains denied;
- no release, deploy, provider, credential, dependency, config, policy, auth, or secret expansion occurred.

## Reproduce

1. Validate JSON files in this directory.
2. Confirm every node file uses only sanitized relative evidence.
3. Check `final-rollup.json` against `evidence-index.json`.
4. Run public-safety and stale-language scans.
