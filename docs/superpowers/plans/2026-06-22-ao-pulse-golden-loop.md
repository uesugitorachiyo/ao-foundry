# AO Pulse Golden Loop Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build one clean-clone-safe AO Foundry command that produces a complete public evidence bundle for the local production factory loop.

**Architecture:** Reuse the existing CLI command functions and builder helpers. Add a small `PulseEvent` summary type and `foundry pulse run` orchestration that calls the same validation/building helpers used by readiness, loop, run, eval, trace, demo, release, and competitive commands.

**Tech Stack:** Go standard library, existing AO Foundry JSON contracts and fixtures, existing CLI test harness.

## Global Constraints

- AO Foundry must operate above AO Forge and must not replace it.
- Public files must not contain private paths, credentials, tokens, or internal handoff content.
- The pulse loop must be local-only and clean-clone safe.
- The command must not push, tag, publish, upload, or mutate sibling repositories.
- Production-readiness requires score 100/100 and explicit evidence.

---

### Task 1: Pulse Command Contract

**Files:**
- Modify: `internal/cli/cli_test.go`
- Modify: `internal/cli/cli.go`

**Interfaces:**
- Produces: `foundry pulse run --out <dir>` defaulting public fixture inputs.
- Produces: `PulseEvent` JSON at `<out>/pulse-event.json`.

- [ ] **Step 1: Write the failing test**

Add a test that runs `pulse run --out <tempdir>` with default fixtures, asserts success, loads `pulse-event.json`, and verifies status `ready`, score `100`, non-empty artifacts, and public-safe artifact paths.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli -run TestPulseRunWritesGoldenLoopBundle -v`
Expected: FAIL with unknown command `pulse`.

- [ ] **Step 3: Implement minimal command routing**

Add `PulseEvent`, `PulseArtifact`, `PulseCheck`, route `pulse` in `Run`, add help text, and implement `runPulse` / `runPulseRun`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/cli -run TestPulseRunWritesGoldenLoopBundle -v`
Expected: PASS.

### Task 2: Evidence Bundle Coverage

**Files:**
- Modify: `internal/cli/cli_test.go`
- Modify: `internal/cli/cli.go`

**Interfaces:**
- Consumes: `runPulseRun`.
- Produces artifact files named `production-readiness-audit.json`, `goal-readiness-audit.json`, `forge-brief.json`, `foundry-run.json`, `eval-result.json`, `pulse.trace.jsonl`, `trace-inspect.json`, `demo-status.json`, `release-manifest.json`, `competitive-readiness-audit.json`.

- [ ] **Step 1: Write the failing test**

Extend the pulse test to require every artifact file to exist, be valid JSON except trace JSONL, and appear in the event artifacts list with a 64-character SHA-256 digest.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli -run TestPulseRunWritesGoldenLoopBundle -v`
Expected: FAIL on missing artifacts.

- [ ] **Step 3: Implement artifact generation**

Use existing builders and writers: `buildReadinessAudit`, `buildGoalReadinessAudit`, `buildForgeBrief`, `buildFoundryRun`, `buildEvalResult`, `readTraceSpans`, `buildDemoStatus`, `buildReleaseManifest`, and `buildCompetitiveAudit`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/cli -run TestPulseRunWritesGoldenLoopBundle -v`
Expected: PASS.

### Task 3: Failure Path And Documentation

**Files:**
- Modify: `internal/cli/cli_test.go`
- Modify: `docs/operations/AO2-PULSE-EVENT-LOOP.md`
- Modify: `README.md`

**Interfaces:**
- Consumes: `foundry pulse run`.
- Produces: operator docs showing the exact command and generated artifacts.

- [ ] **Step 1: Write the failing test**

Add a test that points `--registry` at the blocked registry fixture and asserts non-zero exit plus a failed `pulse-event.json`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli -run TestPulseRunWritesFailedEventForBlockedReadiness -v`
Expected: FAIL until failure event writing exists.

- [ ] **Step 3: Implement failed event writing and docs**

Write a failed event when a step returns a readiness or validation error. Update the operations doc and README command list.

- [ ] **Step 4: Run all verification**

Run: `go test ./...`, then the pulse command, trace inspect, competitive audit, and safety scan from the SDD.
Expected: all pass; safety scan returns no matches.
