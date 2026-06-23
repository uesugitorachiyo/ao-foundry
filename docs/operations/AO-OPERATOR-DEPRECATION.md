# AO Operator Deprecation

AO Operator, AO Runtime, and AO Control Plane are deprecated for active AO
product work.

The replacement path is:

- `ao2` for execution, provider-free command paths, SDD-oriented command execution, and active runtime behavior.
- `ao2-control-plane` for typed state, evidence readback, retention, and control-plane workflows.

## Foundry Policy

Foundry no longer tracks `ao-operator` in the active local stack registry.
Foundry must not register `ao-operator`, `ao-runtime`, or `ao-control-plane` as
active or supporting stack repos.

New work should route to `ao2` or `ao2-control-plane`. Do not add product,
marketing, adapter, runtime, or control-plane scope to the deprecated repos.

## Migration Notes

Useful historical material may be extracted from the deprecated repos only when
it directly simplifies the AO2 or AO2 Control Plane implementation. Carry ideas
forward, not the old product boundary.
