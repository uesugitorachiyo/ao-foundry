# AO Foundry Pulse Production Adapters SDD

## Objective

Advance AO Foundry from a local golden loop to a production-ready operator loop
that can:

- attempt a governed AO Forge live packet when explicitly requested,
- record control-plane readback evidence when a receipt is available,
- expose a short `ao` operator command surface over Foundry status, next, run,
  audit, and demo flows.

## Scope

This slice extends `foundry pulse run` without making external services required
for the public path. The default pulse remains clean-clone safe and fixture
backed. Optional flags add production adapters:

```sh
foundry pulse run --out tmp/pulse \
  --forge-live-packet <path> \
  --control-plane-receipt <path>
```

The live packet path is a packet produced by an explicit AO Forge live run. AO
Foundry validates and bundles it as `forge-live-packet.json`. If the flag is not
provided, the pulse writes `forge-live-blocker.json` explaining that live
execution is not attempted in the local public path.

The control-plane receipt path is a readback receipt produced by AO Forge or the
control-plane. AO Foundry validates and bundles it as
`control-plane-readback.json`. If the flag is not provided, the pulse writes
`control-plane-readback.json` with `status=unavailable` and an operator next
action.

The `ao` surface is a thin alias command:

```sh
ao status
ao next
ao run --out tmp/pulse
ao audit --out tmp/competitive-readiness-audit.json
ao demo
```

Inside this repository it is implemented as `foundry ao ...` so it can be tested
without installing a global binary. A later packaging slice may ship a dedicated
`cmd/ao` wrapper.

## Boundaries

- No implicit live AO Forge invocation in the default pulse.
- No network access unless the operator has already produced a receipt artifact.
- No pushes, tags, releases, uploads, or credential handling.
- AO Forge remains the governed execution owner.
- ao2-control-plane remains the evidence observer; Foundry records readback
  evidence, not approval.

## Contracts

This slice adds lightweight bundle-only summaries:

- `ao.foundry.forge-live-attempt.v0.1`
- `ao.foundry.control-plane-readback.v0.1`

Both summaries are pulse artifacts and appear in `pulse-event.json`. They do not
replace AO Forge packet or control-plane receipt contracts.

## Verification

```sh
go test ./...
go run ./cmd/foundry pulse run --out tmp/pulse
go run ./cmd/foundry ao run --out tmp/ao-pulse
go run ./cmd/foundry ao status
go run ./cmd/foundry competitive audit --out tmp/competitive-readiness-audit.json
```

The expected production-readiness result is still 100/100 with no public-safety
scan matches.
