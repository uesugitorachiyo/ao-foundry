# Foundry Repo Board Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a read-only Foundry repo board that turns registry and health data into portfolio-level recommendations.

**Architecture:** Reuse the existing registry model and `RepoHealthReport` reader. Add a `repo board` subcommand that classifies repos by strategic tier, marks dirty or otherwise blocked repos as hygiene blockers, and emits text or JSON without mutating sibling repositories.

**Tech Stack:** Go CLI, existing JSON registry files, local git status readers.

## Global Constraints

- The board must be local-first and read-only.
- The board must not push, tag, publish, upload, or mutate sibling repositories.
- Dirty sibling repositories must be reported as hygiene blockers, not modified.
- JSON output must be schema-versioned.
- Text output must give concrete next actions.

---

### Task 1: Repo Board Model And Command

**Files:**
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/cli_test.go`
- Modify: `README.md`

**Interfaces:**
- Consumes: `buildRepoHealthReport(registryPath, repoID string) (RepoHealthReport, error)`
- Produces: `RepoBoard`, `RepoBoardEntry`, `buildRepoBoard(registryPath string) (RepoBoard, error)`, and `foundry repo board --registry <path> [--json]`

- [ ] **Step 1: Write failing tests**

Add tests that assert `repo board` emits a JSON board with `active-spine`, `candidate-demote`, and `blocked-hygiene` entries and a text board with next actions.

- [ ] **Step 2: Verify red**

Run: `go test ./internal/cli -run 'TestRepoBoard' -v`

Expected: FAIL because `repo board` does not exist.

- [ ] **Step 3: Implement minimal command**

Add the board data model, builder, command dispatch, text output, and help text.

- [ ] **Step 4: Verify green**

Run: `go test ./internal/cli -run 'TestRepoBoard' -v`

Expected: PASS.

- [ ] **Step 5: Update README**

Document `foundry repo board --registry examples/registry/local-ao-stack.foundry-registry.json`.

- [ ] **Step 6: Full verification**

Run:

```sh
go test ./...
jq empty docs/contracts/*.json examples/**/*.json
go run ./cmd/foundry repo board --registry examples/registry/local-ao-stack.foundry-registry.json
gitleaks detect --no-git --source . --redact --verbose
```

Expected: all commands succeed.
