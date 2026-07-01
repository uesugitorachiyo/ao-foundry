# Recursive Improvement Safe Intermediate Claim Evidence

This directory records public-safe tracked evidence for `public_safe_intermediate_causal_review_claim_evidence`.

Strongest approved wording:

> AO has public-safe intermediate causal-review evidence that bounded improvement evidence can guide and constrain later claim review across independent roles; stronger recursive-improvement wording and broad_RSI remain denied.

The evidence carries forward `public_safe_reviewed_causal_chain_boundary_generalization_evidence` and tests whether a safer intermediate claim can be approved without widening into stronger recursive-improvement wording or broad_RSI.

Results:

- Completed nodes: 560 / 560.
- Stronger recursive-improvement wording remains denied.
- broad_RSI remains denied.
- Unrestricted self-modification remains denied.
- Hidden instruction mutation remains denied.
- Policy-changing autonomy remains denied.
- Public-safety scan passed.
- Retraction path is available for the approved narrow wording.

Reproducibility:

1. Inspect `evidence-index.json`.
2. Inspect `candidate-wording-results.json` and `review-role-results.json`.
3. Inspect `final-rollup.json` for the final class decision.
4. Sample node evidence under `nodes/` and verify every node records a completed status and gate result.
