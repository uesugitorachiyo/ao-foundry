# Recursive Improvement Reviewer-Approved Wording Evidence

This directory records public-safe tracked evidence for `public_safe_reviewer_approved_bounded_recursive_improvement_wording_evidence`.

Approved wording:

> AO has public-safe reviewer-approved bounded recursive-improvement wording evidence showing guided evidence application can improve later evidence attempts under independent review gates; broad_RSI remains denied.

This evidence carries forward `public_safe_guided_evidence_application_four_attempts` and tests whether reviewer-approved bounded recursive-improvement wording can be supported without implying broad_RSI, unrestricted self-modification, hidden instruction mutation, or policy-changing autonomy.

## Measurements

| Attempt | Task type | Baseline | Post-change | Improvement |
| --- | --- | ---: | ---: | ---: |
| attempt-q | public-reader comprehension quality | 0.66 | 0.94 | 0.28 |
| attempt-r | adversarial overbreadth remediation quality | 0.64 | 0.92 | 0.28 |
| attempt-s | Covenant packet specificity quality | 0.65 | 0.91 | 0.26 |
| attempt-t | Sentinel public-risk boundary clarity | 0.63 | 0.90 | 0.27 |

## Gate Results

- Public-reader: approves exact bounded wording only.
- Adversarial wording: passes exact bounded wording and denies broad_RSI control wording.
- Covenant: approves exact bounded wording and denies broad_RSI.
- Architecture: approves exact bounded wording and denies broad_RSI.
- Sentinel: clears exact bounded wording and holds broad_RSI public-risk wording.
- Promoter: promotes `public_safe_reviewer_approved_bounded_recursive_improvement_wording_evidence` only.
- Command: confirms `public_safe_reviewer_approved_bounded_recursive_improvement_wording_evidence` and keeps `broad_RSI` denied.

This evidence does not approve unrestricted self-modification, hidden instruction mutation, policy-changing autonomy, or any policy/auth/secret/provider/deploy/release/config/dependency expansion.
