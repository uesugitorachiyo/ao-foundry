# Blueprint Authorization Pulse Compatibility Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` or `superpowers:executing-plans` to execute this plan task by task. Keep the change on an isolated Foundry branch; do not implement against `main`.

**Goal:** Make Foundry Pulse consume the producer-owned Blueprint build authorization contract without rewriting its bytes, while preserving Atlas digest binding and all existing denied authorities.

**Architecture:** Add a small canonical Blueprint authorization reader in `internal/cli` rather than extending the generic Pulse source loader. The reader recognizes only the producer’s `schema` field, validates the authorization decision and required digest fields, preserves the exact file bytes for SHA-256 calculation, and returns a normal `PulseIntakeSource` for the existing Pulse/Atlas flow. Keep generic `loadPulseIntakeSource` for the unrelated Blueprint clarification request. Extend the existing Atlas binding validator only with fields necessary to compare the canonical project and pack digest.

**Tech Stack:** Go standard library, existing Foundry CLI harness, existing Atlas import fixtures, Blueprint producer JSON artifacts.

## Verified Inputs And Non-Goals

- Canonical producer artifact: `ao-mission/.ao-mission/handoffs/mission-4d91b0a9e4ab273e/blueprint-pack/build-authorization.json`.
- Canonical producer schema: `ao.blueprint.build-authorization.v0.1` in the JSON field `schema`.
- Exact input SHA-256: `b7e18a1967b31b2806184444ab9aeab5e984e050f66261431ec57ece4cc833ee`.
- Atlas Blueprint import: `ao-atlas/.atlas-local/mission-4d91b0a9e4ab273e/atlas-import/blueprint-import.json`.
- Atlas import binds the authorization digest and declares every authority flag false.
- The red reproduction from the shared workspace root is `pulse intake-preflight` with the built Foundry binary and those artifacts. It exits `1` with `unexpected source artifact schema ""`, because `loadPulseIntakeSource` reads only `schema_version` and `contract_version`.
- Do not change Blueprint output bytes or schema ownership. Do not accept `schema_version` as an alias for a producer-owned Blueprint authorization. Do not weaken `validateEvidencePath`, permit `..`, add a generic adapter, call a provider, execute AO2, or clear Foundry readiness.

## Exact Files

| Path | Change |
| --- | --- |
| `internal/cli/blueprint_authorization.go` | Add focused canonical producer contract parsing and validation. |
| `internal/cli/blueprint_authorization_test.go` | Add focused positive and fail-closed contract tests. |
| `internal/cli/cli.go` | Replace only the `--blueprint-authorization` generic-loader call with the canonical reader; extend Atlas binding validation with project and pack checks. |
| `internal/cli/cli_test.go` | Keep CLI integration coverage here only where `Run` and Pulse flags are exercised. |
| `examples/pulse-intake/blueprint-authorization.ready.json` | Replace the Foundry-owned legacy shape with a canonical producer-shaped ready fixture. |
| `examples/pulse-intake/blueprint-authorization.blocked.json` | Make the blocked fixture canonical so the decision path is tested through the same reader. |
| `examples/atlas/blueprint-import.low-risk-code.json` | Recompute and replace the ready fixture’s `build_authorization.digest`; keep its project and pack fields equal to the canonical fixture. |
| `examples/pulse-overnight-start-gate/ready.intake-preflight.json` | Recompute the recorded Blueprint authorization source hash if the fixture is consumed as a closed evidence chain. |
| `examples/contract-fixtures/valid/foundry-pulse-intake-preflight-v0.1.json` | Recompute the fixture source hash if it embeds the ready authorization digest. |
| `internal/cli/testdata/blueprint-authorization-*.json` | Add minimal negative inputs: missing/wrong schema, alias-only schema version, unapproved, incomplete, and digest/project mismatch fixtures as appropriate. |
| `scripts/verify-blueprint-atlas-pulse-contract.sh` | Add an opt-in shared-workspace integration verifier that builds Foundry and invokes the exact real artifacts with repository-relative paths. |

## Global Acceptance Conditions

- The canonical authorization is parsed through `schema`, not `schema_version` or `contract_version`.
- SHA-256 is computed from the bytes read from the input file. No marshal, rewrite, normalization, or adapter artifact appears in the validation path.
- A Pulse ready result with `--requires-atlas` requires matching authorization SHA, project ID, and Blueprint pack digest across producer and Atlas artifacts.
- `status=ready`, `score=100`, and `approved_by_user=true` are all required; none independently grants readiness.
- All Atlas compilation-only authority fields remain false.
- Tests run from the Foundry repository remain clean-clone-safe. The real cross-workspace check is an explicit script, run from the shared workspace root without parent-traversing artifact arguments.

---

### Task 1: Establish The Canonical Contract With A Focused Red Test

**Files:**
- Create: `internal/cli/blueprint_authorization_test.go`
- Modify: `examples/pulse-intake/blueprint-authorization.ready.json`
- Modify: `examples/atlas/blueprint-import.low-risk-code.json`
- Modify when its chain is digest-checked: `examples/pulse-overnight-start-gate/ready.intake-preflight.json`, `examples/contract-fixtures/valid/foundry-pulse-intake-preflight-v0.1.json`
- Create: `internal/cli/testdata/blueprint-authorization-schema-version-only.json`

- [ ] **Step 1: Write the failing producer-contract test.**

Create a helper test that calls a new, not-yet-implemented reader with the canonical ready fixture. Assert the result has `SchemaVersion == "ao.blueprint.build-authorization.v0.1"`, `Status == "ready"`, and `SHA256 == fileSHA256HexForTest(t, fixturePath)`. Exercise Pulse through `Run` with the canonical fixture plus the existing Atlas fixtures so the first red test proves command behavior, not merely struct parsing.

```go
func TestPulseIntakePreflightAcceptsCanonicalBlueprintAuthorization(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"pulse", "intake-preflight",
		"--blueprint-authorization", "examples/pulse-intake/blueprint-authorization.ready.json",
		"--atlas-blueprint-import", "examples/atlas/blueprint-import.low-risk-code.json",
		"--atlas-import", "examples/atlas/foundry-import.json",
		"--atlas-status", "examples/contract-fixtures/valid/foundry-atlas-status-v0.1.json",
		"--requires-atlas",
	}, &stdout, &stderr)
	if code != 0 { t.Fatalf("Run returned %d; stdout=%s stderr=%s", code, stdout.String(), stderr.String()) }
}
```

Use a fixture whose top-level canonical fields are `schema`, `project_id`, `status`, `score`, `approved_by_user`, `blocking_assumptions`, `blueprint_pack_digest`, `requirements_digest`, `traceability_digest`, `sdd_plan_digest`, and `next_allowed_action`. Set its `project_id` and `blueprint_pack_digest` to the values in `examples/atlas/blueprint-import.low-risk-code.json`, then compute the fixture’s raw-byte SHA-256 and update both `build_authorization.digest` and `digests.build_authorization` in that Atlas fixture. Search the repository for the old authorization digest and update only digest references that represent this exact fixture source; do not change the real Mission artifact digest.

- [ ] **Step 2: Run the red test before implementation.**

```sh
go test ./internal/cli -run TestPulseIntakePreflightAcceptsCanonicalBlueprintAuthorization -count=1 -v
```

Expected: FAIL with `unexpected source artifact schema ""`. Do not change the expected failure by adding `schema_version` to the canonical fixture.

---

### Task 2: Implement A Focused Canonical Authorization Reader

**Files:**
- Create: `internal/cli/blueprint_authorization.go`
- Modify: `internal/cli/cli.go`

- [ ] **Step 1: Add typed contract and source result.**

Add a private typed representation that makes producer ownership explicit:

```go
const blueprintBuildAuthorizationSchema = "ao.blueprint.build-authorization.v0.1"

type canonicalBlueprintAuthorization struct {
	Schema              string   `json:"schema"`
	ProjectID           string   `json:"project_id"`
	Status              string   `json:"status"`
	Score               int      `json:"score"`
	ApprovedByUser      bool     `json:"approved_by_user"`
	BlockingAssumptions []string `json:"blocking_assumptions"`
	BlueprintPackDigest string   `json:"blueprint_pack_digest"`
	RequirementsDigest  string   `json:"requirements_digest"`
	TraceabilityDigest  string   `json:"traceability_digest"`
	SDDPlanDigest       string   `json:"sdd_plan_digest"`
	NextAllowedAction   string   `json:"next_allowed_action"`
}

type canonicalBlueprintAuthorizationSource struct {
	PulseSource PulseIntakeSource
	ProjectID   string
	PackDigest  string
}
```

- [ ] **Step 2: Add a fail-closed loader.**

Implement `loadCanonicalBlueprintAuthorization(path string) (canonicalBlueprintAuthorizationSource, error)` in the new file. It must:

1. Call `validateEvidencePath(path)` before reading.
2. Read the raw bytes once with `os.ReadFile`, resolving a repository-relative path only through the existing safe-root helper pattern; do not use a path whose caller has `..`.
3. Decode a raw `map[string]any` to run `validatePublicSafeJSONStrings` before typed decoding.
4. Require `schema == blueprintBuildAuthorizationSchema`. A `schema_version`-only document fails as missing canonical schema even if the value happens to match.
5. Require non-empty public-safe project ID, `status == "ready"`, score exactly `100`, `approved_by_user == true`, an empty `blocking_assumptions`, and `next_allowed_action == "ao-atlas"`.
6. Require each producer digest field to begin with `sha256:` and validate the suffix with existing `validateSHA256`.
7. Hash the same raw byte slice with `sha256.Sum256` and populate `PulseIntakeSource{Name: "blueprint_authorization", Path: ..., SchemaVersion: authorization.Schema, Status: authorization.Status, SHA256: ...}`.

Return explicit errors such as `canonical Blueprint authorization schema is required`, `canonical Blueprint authorization must be approved_by_user=true`, and `canonical Blueprint authorization blueprint_pack_digest must start with sha256:`. Never report a raw absolute path or source bytes in the error.

- [ ] **Step 3: Make the minimal Pulse call-site substitution.**

In `buildPulseIntakePreflight`, replace only this authorization branch:

```go
source, status, err := loadPulseIntakeSource("blueprint_authorization", blueprintAuthorizationPath, "ao.blueprint.build-authorization.v0.1")
```

with the canonical reader result. Keep `loadPulseIntakeSource` unchanged for `--blueprint-request`. Set `blueprintSource` from `authorization.PulseSource`, `result.BlueprintStatus` from the canonical source, and retain the existing result/check/error structure.

- [ ] **Step 4: Run the focused green tests.**

```sh
go test ./internal/cli -run 'Test(PulseIntakePreflightAcceptsCanonicalBlueprintAuthorization|LoadCanonicalBlueprintAuthorization)' -count=1 -v
```

Expected: PASS. The original legacy `schema_version` ready fixture must no longer be used as the producer fixture.

---

### Task 3: Bind Producer Semantics To Atlas, Not Only Bytes

**Files:**
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/blueprint_authorization_test.go`
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Change the binding interface.**

Change `validateAtlasBlueprintImportForFoundry` to receive `canonicalBlueprintAuthorizationSource` instead of a bare `PulseIntakeSource`. Continue comparing:

```go
blueprintImport.BuildAuthorization.Digest == "sha256:"+authorization.PulseSource.SHA256
```

Then require the producer project and pack digest to match Atlas:

```go
if blueprintImport.ProjectID != authorization.ProjectID {
	return errors.New("Atlas Blueprint import project_id must match canonical Blueprint authorization")
}
if blueprintImport.BlueprintPack.Digest != authorization.PackDigest ||
	blueprintImport.Digests["blueprint_pack"] != authorization.PackDigest {
	return errors.New("Atlas Blueprint import Blueprint pack digest must match canonical Blueprint authorization")
}
```

Keep all existing Foundry-import identity and downstream import digest checks. Do not infer project ID from a path or let a matching SHA override a project/pack mismatch.

- [ ] **Step 2: Add negative bindings.**

Add focused tests that mutate copied temp fixtures, then run `pulse intake-preflight --requires-atlas`, asserting nonzero exit and the exact failing check `atlas_blueprint_import` for:

- authorization byte SHA mismatch against `build_authorization.digest`;
- mismatched producer project ID;
- mismatched producer pack digest;
- an Atlas Blueprint import with any of `schedules_work`, `executes_work`, `approves_work`, `mutates_repositories`, `calls_providers`, or `release_or_publish_allowed` true.

The test should write fixture copies under `t.TempDir()` and pass the temp paths only to the test harness. This validates data tampering without changing committed canonical evidence.

- [ ] **Step 3: Run binding tests.**

```sh
go test ./internal/cli -run 'TestPulseIntakePreflight.*(Digest|Project|Pack|Authority)' -count=1 -v
```

Expected: each negative test passes by observing the intentional rejection; no test accepts an authority claim.

---

### Task 4: Exhaust The Producer Contract’s Fail-Closed Cases

**Files:**
- Modify: `internal/cli/blueprint_authorization_test.go`
- Create or modify: `internal/cli/testdata/blueprint-authorization-*.json`
- Modify: `examples/pulse-intake/blueprint-authorization.blocked.json`

- [ ] **Step 1: Add table-driven parser cases.**

For each case, copy the canonical ready fixture into a temp file, mutate one field, call `loadCanonicalBlueprintAuthorization`, and require an error:

| Case | Required assertion |
| --- | --- |
| missing `schema` | Rejects missing canonical schema. |
| wrong `schema` | Rejects unexpected canonical schema. |
| `schema_version` only | Rejects it even when its value equals the producer schema. |
| `status=blocked` | Rejects as not build-ready. |
| `approved_by_user=false` | Rejects despite status ready and score 100. |
| score below or above 100 | Rejects; score alone cannot authorize. |
| missing project ID, pack digest, requirements digest, traceability digest, SDD digest, or next action | Rejects each required producer field. |
| non-empty blocking assumptions | Rejects. |
| malformed digest | Rejects lowercase SHA-256 validation. |
| `../` or absolute input path | Fails through `validateEvidencePath` before file access. |

Include a `TestPulseIntakePreflightFailsClosedForBlockedCanonicalBlueprintAuthorization` CLI test using the canonical blocked fixture. It should keep the current exit-1 behavior and error class while proving the canonical reader parsed the input.

- [ ] **Step 2: Run focused contract tests.**

```sh
go test ./internal/cli -run 'Test(LoadCanonicalBlueprintAuthorization|PulseIntakePreflightFailsClosedForBlockedCanonicalBlueprintAuthorization)' -count=1 -v
```

Expected: PASS. The negative corpus has no “accept alias” compatibility exception.

---

### Task 5: Add A Shared-Workspace Integration Verifier Without Weakening Paths

**Files:**
- Create: `scripts/verify-blueprint-atlas-pulse-contract.sh`
- Modify: `README.md` only if the repository’s existing verification-command section requires listing this script; otherwise omit README changes.

- [ ] **Step 1: Implement an opt-in verifier.**

The script must derive the shared workspace as the parent of the Foundry repository root and fail clearly if `ao-mission` or `ao-atlas` is absent. Use `mktemp -d`, `trap` cleanup, `go build -o "$tmpdir/foundry" ./cmd/foundry`, and run from the shared workspace root. Invoke the binary with repository-relative arguments exactly as follows:

```sh
"$tmpdir/foundry" pulse intake-preflight \
  --blueprint-authorization ao-mission/.ao-mission/handoffs/mission-4d91b0a9e4ab273e/blueprint-pack/build-authorization.json \
  --atlas-blueprint-import ao-atlas/.atlas-local/mission-4d91b0a9e4ab273e/atlas-import/blueprint-import.json \
  --atlas-import ao-atlas/.atlas-local/mission-4d91b0a9e4ab273e/atlas-import/foundry-import/foundry-import.json \
  --atlas-status ao-atlas/.atlas-local/mission-4d91b0a9e4ab273e/atlas-import/atlas-status.json \
  --requires-atlas --json --out "$tmpdir/pulse-intake-preflight.json"
```

Before and after Pulse, compute the authorization file SHA with `shasum -a 256`; require `b7e18a1967b31b2806184444ab9aeab5e984e050f66261431ec57ece4cc833ee`. Use `jq -e` to require result status `ready`, all three input statuses `ready`, and all authority flags declared false in the Atlas import. The script must not alter the input artifacts.

- [ ] **Step 2: Add a negative integration assertion.**

Run the same binary from inside Foundry with a `../ao-mission/...` input in a `set +e` block. Require nonzero exit and `unsafe source artifact path`. This proves the workspace-root design solves the invocation context without widening the path policy.

- [ ] **Step 3: Run the verifier.**

```sh
scripts/verify-blueprint-atlas-pulse-contract.sh
```

Expected after implementation: PASS with `status=ready`; SHA unchanged before/after; all Atlas authority flags false. Before implementation, it is the retained red reproduction and must fail only at canonical schema recognition.

---

### Task 6: Regression, Cross-Repository Readback, And Review

**Files:**
- Verify: all files above
- Verify only: `ao-blueprint` producer artifact and `ao-atlas` existing import artifacts

- [ ] **Step 1: Run Foundry verification.**

```sh
go test ./internal/cli -count=1
go test ./... -count=1
go vet ./...
go build ./cmd/foundry
git diff --check
```

Expected: all commands pass. No generated binary or integration output is staged.

- [ ] **Step 2: Verify producer and Atlas artifacts without mutation.**

From the shared workspace root:

```sh
jq -e '.schema == "ao.blueprint.build-authorization.v0.1" and .status == "ready" and .score == 100 and .approved_by_user == true' ao-mission/.ao-mission/handoffs/mission-4d91b0a9e4ab273e/blueprint-pack/build-authorization.json
shasum -a 256 ao-mission/.ao-mission/handoffs/mission-4d91b0a9e4ab273e/blueprint-pack/build-authorization.json
(cd ao-foundry && go run ./cmd/foundry atlas import validate --import ../ao-atlas/.atlas-local/mission-4d91b0a9e4ab273e/atlas-import/foundry-import/foundry-import.json)
jq -e '.project_id == "ao-stack-six-month-productization-v01" and .build_authorization.digest == "sha256:b7e18a1967b31b2806184444ab9aeab5e984e050f66261431ec57ece4cc833ee" and ([.schedules_work,.executes_work,.approves_work,.mutates_repositories,.calls_providers,.release_or_publish_allowed] | all(. == false))' ao-atlas/.atlas-local/mission-4d91b0a9e4ab273e/atlas-import/blueprint-import.json
```

Expected: all commands pass after implementation; the authorization SHA remains exact and the import remains compile-only.

- [ ] **Step 3: Perform self-review before requesting implementation authorization.**

Review the diff specifically for: raw-byte preservation, schema-alias rejection, absolute/parent path rejection, required-field coverage, project/pack/digest binding, no authority expansion, fixture migration completeness, and no references to local absolute paths in committed files.

## Atlas Repack Status

The requested `foundry-blueprint-authorization-contract-bridge` dependency cannot be inserted through current supported Atlas/Mission contracts. `atlas blueprint import` creates one generated ready node from a candidate-rules input, and `workgraph repair-plan` requires a real matching run-link. Creating that run-link or editing the recorded workgraph would fabricate evidence. This plan therefore treats the Foundry compatibility repair as the first bounded implementation item while preserving the authoritative current workgraph. A subsequent Atlas capability change, separately planned and reviewed, is required to insert a durable dependency node and repack its digest.

## Implementation Authorization Required

Authorize implementation only with this exact scope: “Implement `docs/superpowers/plans/2026-07-09-blueprint-authorization-pulse-compatibility.md` in an isolated `ao-foundry` worktree. The scope is limited to canonical Blueprint authorization consumption, Atlas project/pack/digest binding, focused fixtures/tests, and the workspace-root verification script. Do not modify Blueprint producer bytes, Atlas workgraph evidence, AO2, policy, credentials, release/deploy behavior, or default branches. Open a PR only after local verification passes.”
