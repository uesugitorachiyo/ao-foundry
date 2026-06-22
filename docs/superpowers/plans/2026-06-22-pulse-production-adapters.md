# Pulse Production Adapters Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend AO Foundry pulse with explicit live Forge packet evidence, control-plane readback evidence, and a compact `ao` operator command surface.

**Architecture:** Keep the existing `foundry pulse run` golden loop as the local default. Add optional artifact readers for live packet and control-plane receipt paths, fallback blocker/readback summaries when absent, and route `foundry ao ...` to existing Foundry commands.

**Tech Stack:** Go standard library, existing AO Foundry CLI, existing JSON fixture and artifact pattern.

## Global Constraints

- AO Foundry must operate above AO Forge and must not replace it.
- Default pulse must remain local-only and clean-clone safe.
- Live AO Forge evidence is bundled only when an operator provides a packet path.
- Control-plane readback is bundled only when an operator provides a receipt path.
- No pushes, tags, releases, uploads, network calls, or credential handling.
- Public files must not include private paths, credentials, tokens, or internal handoff content.

---

### Task 1: Live Forge Adapter Evidence

**Files:**
- Modify: `internal/cli/cli_test.go`
- Modify: `internal/cli/cli.go`

**Interfaces:**
- Produces: `foundry pulse run --forge-live-packet <path>`.
- Produces: `forge_live_attempt` artifact in `pulse-event.json`.

- [ ] **Step 1: Write failing tests**

Add one test asserting the default pulse includes `forge_live_attempt` with `status=blocked`, and one test asserting `--forge-live-packet examples/packets/ao-foundry-bootstrap.factory-packet.json` includes `status=passed`.

- [ ] **Step 2: Run red tests**

Run: `go test ./internal/cli -run 'TestPulseRun.*ForgeLive' -v`
Expected: FAIL because the artifact is absent.

- [ ] **Step 3: Implement minimal adapter**

Add a `ForgeLiveAttempt` type and optional `--forge-live-packet` flag. Validate provided packets with `loadForgePacket`; otherwise write a blocked local-only summary.

- [ ] **Step 4: Run green tests**

Run: `go test ./internal/cli -run 'TestPulseRun.*ForgeLive' -v`
Expected: PASS.

### Task 2: Control-Plane Readback Evidence

**Files:**
- Modify: `internal/cli/cli_test.go`
- Modify: `internal/cli/cli.go`

**Interfaces:**
- Produces: `foundry pulse run --control-plane-receipt <path>`.
- Produces: `control_plane_readback` artifact in `pulse-event.json`.

- [ ] **Step 1: Write failing tests**

Add one test asserting default pulse includes an unavailable readback artifact, and one test with a temp receipt JSON asserting readback status `ready`.

- [ ] **Step 2: Run red tests**

Run: `go test ./internal/cli -run 'TestPulseRun.*ControlPlane' -v`
Expected: FAIL because the artifact is absent.

- [ ] **Step 3: Implement minimal adapter**

Add `ControlPlaneReadback` with schema, status, source, receipt schema, and explanation. Validate that provided receipt JSON has a non-empty `schema_version`.

- [ ] **Step 4: Run green tests**

Run: `go test ./internal/cli -run 'TestPulseRun.*ControlPlane' -v`
Expected: PASS.

### Task 3: AO Operator Surface

**Files:**
- Modify: `internal/cli/cli_test.go`
- Modify: `internal/cli/cli.go`
- Modify: `README.md`
- Modify: `docs/operations/AO2-PULSE-EVENT-LOOP.md`

**Interfaces:**
- Produces: `foundry ao status`, `foundry ao next`, `foundry ao run --out <dir>`, `foundry ao audit --out <path>`, `foundry ao demo`.

- [ ] **Step 1: Write failing tests**

Add tests for `ao status`, `ao run`, and `ao audit`.

- [ ] **Step 2: Run red tests**

Run: `go test ./internal/cli -run 'TestAOSurface' -v`
Expected: FAIL with unknown command.

- [ ] **Step 3: Implement routing**

Route `ao` subcommands to existing Foundry functions with default public fixture paths.

- [ ] **Step 4: Run final verification**

Run `go test ./...`, pulse run, `ao run`, competitive audit, JSON parse, schema validation, and public-safety scan.
