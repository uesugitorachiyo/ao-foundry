# Recursive Improvement Evidence Selection Guidance

This directory records public-safe tracked evidence for `public_safe_causal_review_evidence_selection_guidance`.

Approved wording:

> AO has public-safe causal-review evidence that prior bounded evidence can guide later evidence-selection and blocker prioritization under independent review gates; stronger recursive-improvement wording and broad_RSI remain denied.

This evidence carries forward `public_safe_intermediate_causal_review_claim_evidence` and tests whether prior bounded evidence can guide later evidence-selection and blocker prioritization under independent gates.

## Measurements

| Attempt | Task type | Baseline | Post-change | Improvement |
| --- | --- | ---: | ---: | ---: |
| attempt-i | public-safe evidence-selection quality | 0.69 | 0.92 | 0.23 |
| attempt-j | claim blocker prioritization quality | 0.66 | 0.91 | 0.25 |
| attempt-k | independent review-role consistency | 0.64 | 0.89 | 0.25 |
| attempt-l | cross-role evidence consistency | 0.63 | 0.90 | 0.27 |

## Gate Results

- Public-reader: approves narrow evidence-selection guidance only.
- Adversarial wording: approves narrow guidance wording and denies capability-adjacent wording.
- Covenant: approves narrow publication wording and denies stronger recursive-improvement wording and broad_RSI.
- Architecture: approves narrow wording and denies stronger recursive-improvement wording and broad_RSI.
- Sentinel: clears narrow wording and holds stronger public-risk wording.
- Promoter: promotes `public_safe_causal_review_evidence_selection_guidance` only.
- Command: confirms `public_safe_causal_review_evidence_selection_guidance` and keeps `broad_RSI` denied.

This evidence does not approve unrestricted self-modification, hidden instruction mutation, policy-changing autonomy, or any policy/auth/secret/provider/deploy/release/config/dependency expansion.
