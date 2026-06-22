# Threat Model

## Assets

- Public schemas and examples.
- Local run, trace, eval, approval, and release evidence.
- Operator trust in the separation between AO Foundry and AO Forge.

## Threats

- Accidental publication of local paths, credentials, or internal coordination
  material.
- Treating observation evidence as approval.
- Running network, release, upload, tag, push, or cross-repository write actions
  without explicit approval.
- Allowing stale loop leases or stale approval decisions to advance work.

## Controls

- Public-safety scan in CI and release dry-run.
- HITL approval records bind task digests and requested side effects.
- Loop leases refuse overlap and require explicit release.
- AO Foundry emits Forge briefs and does not directly perform governed
  implementation work.
