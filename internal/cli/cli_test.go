package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRegistryValidateAcceptsExample(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"registry", "validate", "--registry", registryFixture()}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "registry valid") {
		t.Fatalf("expected validation output, got %q", stdout.String())
	}
}

func TestRegistryValidateRejectsMalformedFixture(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"registry", "validate", "--registry", filepath.Join("testdata", "invalid-registry.json")}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for invalid registry; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "registry") {
		t.Fatalf("expected registry error, got %q", stderr.String())
	}
}

func TestRegistryValidateRejectsSchemaEnumAndNestedStringViolations(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"registry", "validate", "--registry", filepath.Join("testdata", "schema-invalid-registry.json")}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for schema-invalid registry; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "invalid role") {
		t.Fatalf("expected role validation error, got %q", stderr.String())
	}
}

func TestTaskValidateAcceptsExample(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"task", "validate", "--task", taskFixture()}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "task valid") {
		t.Fatalf("expected validation output, got %q", stdout.String())
	}
}

func TestTaskValidateRejectsMalformedFixture(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"task", "validate", "--task", filepath.Join("testdata", "invalid-task.json")}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for invalid task; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "task") {
		t.Fatalf("expected task error, got %q", stderr.String())
	}
}

func TestTaskValidateRejectsSchemaEnumAndNestedStringViolations(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"task", "validate", "--task", filepath.Join("testdata", "schema-invalid-task.json")}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for schema-invalid task; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "invalid priority") {
		t.Fatalf("expected priority validation error, got %q", stderr.String())
	}
}

func TestStatusSummarizesRegistry(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"status", "--registry", registryFixture()}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"AO Foundry", "6 repos", "ao-foundry", "ready: 6"} {
		if !strings.Contains(out, want) {
			t.Fatalf("status output missing %q: %s", want, out)
		}
	}
}

func TestNextReportsDelegatedForgeAction(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"next", "--registry", registryFixture(), "--task", taskFixture()}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"ao-foundry-bootstrap", "delegate", "AO Forge", "go test ./..."} {
		if !strings.Contains(out, want) {
			t.Fatalf("next output missing %q: %s", want, out)
		}
	}
}

func TestNextWritesForgeBrief(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "ao-foundry.forge-brief.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"next", "--registry", registryFixture(), "--task", taskFixture(), "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "forge_brief="+outPath) {
		t.Fatalf("expected forge brief path in output, got %q", stdout.String())
	}
	var brief map[string]any
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read forge brief: %v", err)
	}
	if err := json.Unmarshal(data, &brief); err != nil {
		t.Fatalf("forge brief is not JSON: %v", err)
	}
	if brief["schema_version"] != "ao.forge.factory-brief.v0.1" {
		t.Fatalf("unexpected brief schema: %v", brief["schema_version"])
	}
	objective := brief["objective"].(map[string]any)
	if objective["workspace"] != "../ao-foundry" {
		t.Fatalf("expected workspace ../ao-foundry, got %v", objective["workspace"])
	}
	constraints := brief["constraints"].(map[string]any)
	if constraints["allow_network"] != false || constraints["allow_release_mutation"] != false {
		t.Fatalf("brief must not allow network or release mutation: %#v", constraints)
	}
	workcells := brief["expected_workcells"].([]any)
	if len(workcells) == 0 {
		t.Fatalf("expected workcells in brief")
	}
}

func TestNextJSONReportsReadyAction(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "ao-foundry.forge-brief.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"next", "--registry", registryFixture(), "--task", taskFixture(), "--out", outPath, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	var action map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &action); err != nil {
		t.Fatalf("next JSON output is not JSON: %v; output=%s", err, stdout.String())
	}
	if action["schema_version"] != "ao.foundry.next-action.v0.1" || action["status"] != "ready" {
		t.Fatalf("unexpected action output: %#v", action)
	}
	if action["delegate_to"] != "ao-forge" || action["forge_brief"] != outPath {
		t.Fatalf("unexpected action delegation: %#v", action)
	}
}

func TestNextRefusesBlockedReadiness(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "blocked.forge-brief.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"next", "--registry", filepath.Join("testdata", "blocked-registry.json"), "--task", filepath.Join("testdata", "blocked-task.json"), "--out", outPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for blocked readiness; stdout=%s", stdout.String())
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("blocked next should not write brief, stat err=%v", err)
	}
	if !strings.Contains(stderr.String(), "production readiness below 100") {
		t.Fatalf("expected readiness failure, got %q", stderr.String())
	}
}

func TestNextRefusesUnsafeTaskSafety(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "unsafe.forge-brief.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"next", "--registry", registryFixture(), "--task", filepath.Join("testdata", "unsafe-task.json"), "--out", outPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for unsafe task; stdout=%s", stdout.String())
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("unsafe next should not write brief, stat err=%v", err)
	}
}

func TestNextRefusesMissingForgeDelegation(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "missing-delegation.forge-brief.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"next", "--registry", registryFixture(), "--task", filepath.Join("testdata", "missing-forge-delegation-task.json"), "--out", outPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for missing Forge delegation; stdout=%s", stdout.String())
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("missing delegation should not write brief, stat err=%v", err)
	}
}

func TestReadinessAuditScoresExampleAt100(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "readiness.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"readiness", "audit", "--registry", registryFixture(), "--task", taskFixture(), "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "production readiness: 100/100") {
		t.Fatalf("expected 100 readiness output, got %q", stdout.String())
	}
	var audit map[string]any
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read audit output: %v", err)
	}
	if err := json.Unmarshal(data, &audit); err != nil {
		t.Fatalf("audit output is not JSON: %v", err)
	}
	if audit["schema_version"] != "ao.foundry.production-readiness-audit.v0.1" {
		t.Fatalf("unexpected schema: %v", audit["schema_version"])
	}
	if audit["status"] != "ready" || audit["score"] != float64(100) {
		t.Fatalf("expected ready score 100, got status=%v score=%v", audit["status"], audit["score"])
	}
	if _, ok := audit["next_actions"].([]any); !ok {
		t.Fatalf("expected next_actions to be an array, got %T", audit["next_actions"])
	}
}

func TestReadinessAuditCreatesOutputParentDir(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "nested", "readiness.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"readiness", "audit", "--registry", registryFixture(), "--task", taskFixture(), "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected readiness output at nested path: %v", err)
	}
}

func TestReadinessAuditRejectsBlockedSignal(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"readiness", "audit", "--registry", filepath.Join("testdata", "blocked-registry.json"), "--task", filepath.Join("testdata", "blocked-task.json")}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for blocked readiness; stdout=%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "production readiness: 80/100") {
		t.Fatalf("expected 80 readiness output, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "production readiness below 100") {
		t.Fatalf("expected readiness failure, got %q", stderr.String())
	}
}

func TestGoalValidateAcceptsExample(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"goal", "validate", "--goal-run", goalFixture()}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "goal valid") {
		t.Fatalf("expected goal validation output, got %q", stdout.String())
	}
}

func TestGoalValidateRejectsUnsafeEvidencePath(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"goal", "validate", "--goal-run", filepath.Join("testdata", "unsafe-evidence.goal-run.json")}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for unsafe evidence path; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "unsafe evidence path") {
		t.Fatalf("expected unsafe evidence path error, got %q", stderr.String())
	}
}

func TestGoalValidateRejectsUnsafeNextActionGuard(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"goal", "validate", "--goal-run", filepath.Join("testdata", "unsafe-guard.goal-run.json")}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for unsafe next action guard; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "next_action_guard") {
		t.Fatalf("expected next_action_guard error, got %q", stderr.String())
	}
}

func TestGoalReadinessScoresExampleAt100(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "goal-readiness.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"goal", "readiness", "--goal-run", goalFixture(), "--registry", registryFixture(), "--task", taskFixture(), "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "goal readiness: 100/100") {
		t.Fatalf("expected 100 goal readiness output, got %q", stdout.String())
	}
	var audit map[string]any
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read audit output: %v", err)
	}
	if err := json.Unmarshal(data, &audit); err != nil {
		t.Fatalf("goal readiness audit output is not JSON: %v", err)
	}
	if audit["schema_version"] != "ao.foundry.goal-readiness-audit.v0.1" {
		t.Fatalf("unexpected schema: %v", audit["schema_version"])
	}
	if audit["status"] != "ready" || audit["score"] != float64(100) {
		t.Fatalf("expected ready score 100, got status=%v score=%v", audit["status"], audit["score"])
	}
}

func TestGoalReadinessRejectsDigestMismatch(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"goal", "readiness", "--goal-run", filepath.Join("testdata", "digest-mismatch.goal-run.json"), "--registry", registryFixture(), "--task", taskFixture()}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for digest mismatch; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "goal readiness below 100") {
		t.Fatalf("expected readiness failure, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "goal readiness: 80/100") {
		t.Fatalf("expected 80 goal readiness output, got %q", stdout.String())
	}
}

func TestGoalReadinessRejectsTerminalPhase(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"goal", "readiness", "--goal-run", filepath.Join("testdata", "terminal.goal-run.json"), "--registry", registryFixture(), "--task", taskFixture()}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for terminal goal phase; stdout=%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "goal readiness: 80/100") {
		t.Fatalf("expected 80 goal readiness output, got %q", stdout.String())
	}
}

func TestRunIngestWritesDeterministicFoundryRun(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "ao-foundry-bootstrap.foundry-run.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"run", "ingest", "--registry", registryFixture(), "--task", taskFixture(), "--packet", validPacketFixture(), "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "run_record="+outPath) {
		t.Fatalf("expected output path, got %q", stdout.String())
	}
	var run map[string]any
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read run: %v", err)
	}
	if err := json.Unmarshal(data, &run); err != nil {
		t.Fatalf("run is not JSON: %v", err)
	}
	if run["schema_version"] != "ao.foundry.run.v0.1" || run["task_id"] != "ao-foundry-bootstrap" {
		t.Fatalf("unexpected run identity: %#v", run)
	}
	if run["registry_id"] != "local-ao-stack" || run["delegated_to"] != "ao-forge" || run["status"] != "passed" {
		t.Fatalf("unexpected run status/delegation: %#v", run)
	}
	packet := run["forge_packet"].(map[string]any)
	if packet["path"] != validPacketFixture() || packet["status"] != "passed" {
		t.Fatalf("unexpected packet ref: %#v", packet)
	}
	if len(packet["sha256"].(string)) != 64 {
		t.Fatalf("packet digest must be a sha256, got %#v", packet["sha256"])
	}
	if len(run["evidence"].([]any)) != 1 || len(run["decisions"].([]any)) != 1 {
		t.Fatalf("expected copied evidence and decisions: %#v", run)
	}
}

func TestRunValidateAcceptsExample(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"run", "validate", "--run", filepath.Join("..", "..", "examples", "runs", "ao-foundry-bootstrap.foundry-run.json")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "run valid: ao-foundry-bootstrap") {
		t.Fatalf("expected run validation output, got %q", stdout.String())
	}
}

func TestRunInspectPrintsSummary(t *testing.T) {
	runPath := filepath.Join(t.TempDir(), "ao-foundry-bootstrap.foundry-run.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"run", "ingest", "--registry", registryFixture(), "--task", taskFixture(), "--packet", validPacketFixture(), "--out", runPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("ingest returned %d; stderr=%s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"run", "inspect", "--run", runPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("inspect returned %d; stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"status=passed", "task_id=ao-foundry-bootstrap", "delegated_to=ao-forge", "packet_sha256=", "evidence_count=1"} {
		if !strings.Contains(out, want) {
			t.Fatalf("inspect output missing %q: %s", want, out)
		}
	}
}

func TestRunInspectJSON(t *testing.T) {
	runPath := filepath.Join(t.TempDir(), "ao-foundry-bootstrap.foundry-run.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"run", "ingest", "--registry", registryFixture(), "--task", taskFixture(), "--packet", validPacketFixture(), "--out", runPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("ingest returned %d; stderr=%s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"run", "inspect", "--run", runPath, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("inspect returned %d; stderr=%s", code, stderr.String())
	}
	var run map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &run); err != nil {
		t.Fatalf("inspect JSON output is not JSON: %v; output=%s", err, stdout.String())
	}
	if run["schema_version"] != "ao.foundry.run.v0.1" {
		t.Fatalf("unexpected inspect JSON: %#v", run)
	}
}

func TestRunIngestRejectsMissingPacket(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"run", "ingest", "--registry", registryFixture(), "--task", taskFixture(), "--packet", filepath.Join("testdata", "missing-packet.json"), "--out", filepath.Join(t.TempDir(), "run.json")}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for missing packet; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "packet") {
		t.Fatalf("expected packet error, got %q", stderr.String())
	}
}

func TestRunIngestRejectsUnsafeEvidencePath(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"run", "ingest", "--registry", registryFixture(), "--task", taskFixture(), "--packet", filepath.Join("testdata", "unsafe-evidence-forge-packet.json"), "--out", filepath.Join(t.TempDir(), "run.json")}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for unsafe evidence path; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "unsafe evidence path") {
		t.Fatalf("expected unsafe evidence path error, got %q", stderr.String())
	}
}

func TestRunValidateRejectsTamperedPacketDigest(t *testing.T) {
	runPath := filepath.Join(t.TempDir(), "tampered.foundry-run.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"run", "ingest", "--registry", registryFixture(), "--task", taskFixture(), "--packet", validPacketFixture(), "--out", runPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("ingest returned %d; stderr=%s", code, stderr.String())
	}
	data, err := os.ReadFile(runPath)
	if err != nil {
		t.Fatalf("read run: %v", err)
	}
	var run FoundryRun
	if err := json.Unmarshal(data, &run); err != nil {
		t.Fatalf("unmarshal run: %v", err)
	}
	run.ForgePacket.SHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
	data, err = json.MarshalIndent(run, "", "  ")
	if err != nil {
		t.Fatalf("marshal tampered run: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(runPath, data, 0o644); err != nil {
		t.Fatalf("write tampered run: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"run", "validate", "--run", runPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for tampered packet digest; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "sha256 mismatch") {
		t.Fatalf("expected digest mismatch error, got %q", stderr.String())
	}
}

func TestRepoHealthJSONReportsLocalOnlyHealth(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"repo", "health", "--registry", registryFixture(), "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	var health map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &health); err != nil {
		t.Fatalf("health output is not JSON: %v; output=%s", err, stdout.String())
	}
	if health["schema_version"] != "ao.foundry.repo-health.v0.1" || health["registry_id"] != "local-ao-stack" {
		t.Fatalf("unexpected health identity: %#v", health)
	}
	repos := health["repos"].([]any)
	if len(repos) != 6 {
		t.Fatalf("expected 6 repo health entries, got %d", len(repos))
	}
}

func TestRepoHealthFiltersRepo(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"repo", "health", "--registry", registryFixture(), "--repo", "ao-foundry", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	var health RepoHealthReport
	if err := json.Unmarshal(stdout.Bytes(), &health); err != nil {
		t.Fatalf("health output is not JSON: %v; output=%s", err, stdout.String())
	}
	if len(health.Repos) != 1 || health.Repos[0].RepoID != "ao-foundry" {
		t.Fatalf("expected only ao-foundry health, got %#v", health.Repos)
	}
}

func TestRepoHealthRejectsMissingRepo(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"repo", "health", "--registry", registryFixture(), "--repo", "missing"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for missing repo; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "repo id") {
		t.Fatalf("expected missing repo error, got %q", stderr.String())
	}
}

func TestRepoHealthBlocksDirtyWorkspace(t *testing.T) {
	workspace := initTempGitRepo(t)
	if err := os.WriteFile(filepath.Join(workspace, "dirty.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}
	registryPath := writeHealthRegistry(t, workspace, "main", nil, nil)
	var stdout, stderr bytes.Buffer
	code := Run([]string{"repo", "health", "--registry", registryPath, "--json"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for dirty workspace; stdout=%s", stdout.String())
	}
	var health RepoHealthReport
	if err := json.Unmarshal(stdout.Bytes(), &health); err != nil {
		t.Fatalf("health output is not JSON: %v; output=%s", err, stdout.String())
	}
	if health.Status != "blocked" || health.Repos[0].Status != "blocked" {
		t.Fatalf("expected blocked dirty health, got %#v", health)
	}
}

func TestRepoHealthBlocksMissingBranchAndEvidence(t *testing.T) {
	workspace := initTempGitRepo(t)
	registryPath := writeHealthRegistry(t, workspace, "missing-branch", []string{"missing.txt"}, nil)
	var stdout, stderr bytes.Buffer
	code := Run([]string{"repo", "health", "--registry", registryPath, "--json"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for missing branch/evidence; stdout=%s", stdout.String())
	}
	var health RepoHealthReport
	if err := json.Unmarshal(stdout.Bytes(), &health); err != nil {
		t.Fatalf("health output is not JSON: %v; output=%s", err, stdout.String())
	}
	checks := health.Repos[0].Checks
	var branchBlocked, evidenceBlocked bool
	for _, check := range checks {
		if check.Name == "branch_present" && check.Status == "blocked" {
			branchBlocked = true
		}
		if check.Name == "readiness_file_exists" && check.Status == "blocked" {
			evidenceBlocked = true
		}
	}
	if !branchBlocked || !evidenceBlocked {
		t.Fatalf("expected blocked branch and evidence checks, got %#v", checks)
	}
}

func TestRepoHealthReportsUnavailableGitHubReaderUnknown(t *testing.T) {
	workspace := initTempGitRepo(t)
	registryPath := writeHealthRegistry(t, workspace, "main", nil, &HealthReaderConfig{AllowNetworkRead: false, GitHubActions: true})
	var stdout, stderr bytes.Buffer
	code := Run([]string{"repo", "health", "--registry", registryPath, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0 for optional unknown GitHub reader; stderr=%s", code, stderr.String())
	}
	var health RepoHealthReport
	if err := json.Unmarshal(stdout.Bytes(), &health); err != nil {
		t.Fatalf("health output is not JSON: %v; output=%s", err, stdout.String())
	}
	var foundUnknown bool
	for _, check := range health.Repos[0].Checks {
		if check.Name == "github_actions_status" && check.Status == "unknown" {
			foundUnknown = true
		}
	}
	if !foundUnknown {
		t.Fatalf("expected unknown GitHub Actions check, got %#v", health.Repos[0].Checks)
	}
}

func TestRepoBoardJSONClassifiesPortfolio(t *testing.T) {
	clean := initTempGitRepo(t)
	dirty := initTempGitRepo(t)
	if err := os.WriteFile(filepath.Join(dirty, "dirty.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}
	registry := Registry{
		SchemaVersion: "ao.foundry.registry.v0.1",
		FoundryID:     "board-fixture",
		Name:          "Board Fixture",
		Repos: []Repo{
			boardFixtureRepo("ao2", "AO2", "execution-engine", clean),
			boardFixtureRepo("ao-forge", "AO Forge", "factory-brain", dirty),
			boardFixtureRepo("ao-conductor", "AO Conductor", "workflow-conductor", clean),
		},
	}
	registryPath := writeRegistryFixture(t, registry)
	var stdout, stderr bytes.Buffer
	code := Run([]string{"repo", "board", "--registry", registryPath, "--json"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for dirty board; stdout=%s", stdout.String())
	}
	var board RepoBoard
	if err := json.Unmarshal(stdout.Bytes(), &board); err != nil {
		t.Fatalf("board output is not JSON: %v; output=%s", err, stdout.String())
	}
	if board.SchemaVersion != repoBoardSchema || board.RegistryID != "board-fixture" || board.Status != "blocked" {
		t.Fatalf("unexpected board identity/status: %#v", board)
	}
	entries := map[string]RepoBoardEntry{}
	for _, entry := range board.Repos {
		entries[entry.RepoID] = entry
	}
	if entries["ao2"].Tier != "active-spine" || entries["ao2"].Recommendation != "advance" {
		t.Fatalf("expected ao2 active spine advance, got %#v", entries["ao2"])
	}
	if entries["ao-forge"].Tier != "blocked-hygiene" || entries["ao-forge"].Recommendation != "clean-worktree" {
		t.Fatalf("expected dirty ao-forge hygiene blocker, got %#v", entries["ao-forge"])
	}
	if entries["ao-conductor"].Tier != "candidate-demote" || entries["ao-conductor"].Recommendation != "freeze-or-archive" {
		t.Fatalf("expected ao-conductor demotion candidate, got %#v", entries["ao-conductor"])
	}
}

func TestRepoBoardTextReportsNextActions(t *testing.T) {
	clean := initTempGitRepo(t)
	registry := Registry{
		SchemaVersion: "ao.foundry.registry.v0.1",
		FoundryID:     "board-text-fixture",
		Name:          "Board Text Fixture",
		Repos: []Repo{
			boardFixtureRepo("ao-foundry", "AO Foundry", "operations-factory", clean),
			boardFixtureRepo("ao-command", "AO Command", "operator-command", clean),
			boardFixtureRepo("ao2-control-plane", "AO2 Control Plane", "evidence-observer", clean),
		},
	}
	registryPath := writeRegistryFixture(t, registry)
	var stdout, stderr bytes.Buffer
	code := Run([]string{"repo", "board", "--registry", registryPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"repo board: 3 repos status=ready", "ao-foundry", "active-spine", "ao-command", "read-only operator/readback surface for ao-forge, ao2, ao2-control-plane, and ao-covenant", "do not route archived or subscription-backed scope through it", "ao2-control-plane", "active-spine", "next_action="} {
		if !strings.Contains(out, want) {
			t.Fatalf("board text missing %q: %s", want, out)
		}
	}
}

func TestActiveStackReadinessLoopScriptDocumentsLocalAuditChain(t *testing.T) {
	script, err := os.ReadFile(repoPath("scripts/active-stack-readiness-loop.sh"))
	if err != nil {
		t.Fatalf("read active stack readiness loop script: %v", err)
	}
	readme, err := os.ReadFile(repoPath("README.md"))
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	scriptText := string(script)
	readmeText := string(readme)
	for _, want := range []string{
		"ao.foundry.active-stack-readiness-loop.v0.1",
		"schema_version",
		"registry validate",
		"readiness snapshot",
		"readiness rollup",
		"active-stack-production-readiness-rollup",
		"repo board",
		"release handoff",
		"loop preflight",
		"first_failing_check",
	} {
		if !strings.Contains(scriptText, want) {
			t.Fatalf("active stack readiness loop script missing %q", want)
		}
	}
	for _, forbidden := range []string{"git push", "gh release", "gh repo edit", "gh pr merge"} {
		if strings.Contains(scriptText, forbidden) {
			t.Fatalf("active stack readiness loop script contains publishing command %q", forbidden)
		}
	}
	if !strings.Contains(readmeText, "scripts/active-stack-readiness-loop.sh") {
		t.Fatalf("README does not document active stack readiness loop")
	}
}

func TestActiveStackGitHubRunsReportScriptDocumentsRemoteEvidenceChain(t *testing.T) {
	script, err := os.ReadFile(repoPath("scripts/active-stack-github-runs-report.sh"))
	if err != nil {
		t.Fatalf("read active stack GitHub runs report script: %v", err)
	}
	readme, err := os.ReadFile(repoPath("README.md"))
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	scriptText := string(script)
	readmeText := string(readme)
	for _, want := range []string{
		"ao.foundry.active-stack-github-runs-report.v0.1",
		"ci.yml",
		"production-readiness-ops.yml",
		"gh run list",
		"uesugitorachiyo/ao-foundry",
		"uesugitorachiyo/ao-forge",
		"uesugitorachiyo/ao-command",
		"uesugitorachiyo/ao2",
		"uesugitorachiyo/ao2-control-plane",
		"uesugitorachiyo/ao-covenant",
		"latest_ci",
		"latest_ops",
		"readiness evidence-check",
		"CURRENT_REPO",
		"current_repo_skipped",
	} {
		if !strings.Contains(scriptText, want) {
			t.Fatalf("active stack GitHub runs report script missing %q", want)
		}
	}
	for _, forbidden := range []string{"gh pr merge", "gh workflow run", "git push", "gh release", "ao-operator", "ao-runtime", "ao-control-plane", "ao-conductor", "agy-swarms", "codex-cron"} {
		if strings.Contains(scriptText, forbidden) {
			t.Fatalf("active stack GitHub runs report script contains forbidden scope or mutation %q", forbidden)
		}
	}
	if !strings.Contains(readmeText, "scripts/active-stack-github-runs-report.sh") {
		t.Fatalf("README does not document active stack GitHub runs report")
	}
}

func TestBranchProtectionVerifierDocumentsRequiredChecks(t *testing.T) {
	script, err := os.ReadFile(repoPath("scripts/verify-branch-protection.sh"))
	if err != nil {
		t.Fatalf("read branch protection verifier: %v", err)
	}
	doc, err := os.ReadFile(repoPath("docs/operations/BRANCH-PROTECTION.md"))
	if err != nil {
		t.Fatalf("read branch protection docs: %v", err)
	}
	readme, err := os.ReadFile(repoPath("README.md"))
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	scriptText := string(script)
	docText := string(doc)
	readmeText := string(readme)
	for _, want := range []string{
		"gh api",
		"branches/$BRANCH/protection",
		"branches/$BRANCH\"",
		"mode=limited",
		"test (ubuntu-latest)",
		"test (macos-latest)",
		"test (windows-latest)",
		"enforce_admins",
		"required_linear_history",
		"branch_protection=passed",
	} {
		if !strings.Contains(scriptText, want) {
			t.Fatalf("branch protection verifier missing %q", want)
		}
		if !strings.Contains(docText, want) && strings.HasPrefix(want, "test ") {
			t.Fatalf("branch protection docs missing required check %q", want)
		}
	}
	for _, forbidden := range []string{"git push", "gh pr merge", "-X PUT", "-X PATCH", "gh repo edit"} {
		if strings.Contains(scriptText, forbidden) {
			t.Fatalf("branch protection verifier contains mutating command %q", forbidden)
		}
	}
	if !strings.Contains(readmeText, "scripts/verify-branch-protection.sh") {
		t.Fatalf("README does not document branch protection verifier")
	}
}

func TestActiveStackReadinessLedgerMatchesRegistry(t *testing.T) {
	schema, err := readArbitraryJSON("docs/contracts/foundry-active-stack-readiness-v0.1.schema.json")
	if err != nil {
		t.Fatalf("read active stack readiness schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("active stack readiness schema is not an object: %#v", schema)
	}
	ledger, err := readArbitraryJSON("examples/readiness/active-stack-readiness.ledger.json")
	if err != nil {
		t.Fatalf("read active stack readiness ledger: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, ledger, "$"); err != nil {
		t.Fatalf("active stack readiness ledger failed schema: %v", err)
	}
	registry, err := loadRegistry(registryFixture())
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	ledgerObject, ok := ledger.(map[string]any)
	if !ok {
		t.Fatalf("ledger is not an object: %#v", ledger)
	}
	if ledgerObject["registry_id"] != registry.FoundryID || ledgerObject["status"] != "ready" {
		t.Fatalf("unexpected ledger identity/status: %#v", ledgerObject)
	}
	active := map[string]bool{}
	for _, repo := range registry.Repos {
		active[repo.ID] = true
	}
	entries, ok := ledgerObject["repositories"].([]any)
	if !ok {
		t.Fatalf("ledger repositories are not an array: %#v", ledgerObject["repositories"])
	}
	if len(entries) != len(active) {
		t.Fatalf("ledger has %d repositories, registry has %d", len(entries), len(active))
	}
	for _, entry := range entries {
		repo, ok := entry.(map[string]any)
		if !ok {
			t.Fatalf("ledger repository is not an object: %#v", entry)
		}
		id, ok := repo["id"].(string)
		if !ok || !active[id] {
			t.Fatalf("ledger contains non-registry repo: %#v", repo)
		}
		if repo["status"] != "ready" {
			t.Fatalf("ledger repo %s is not ready: %#v", id, repo)
		}
		if id == "ao-foundry" {
			ci, ok := repo["ci"].(map[string]any)
			if !ok {
				t.Fatalf("ao-foundry ledger entry has no ci object: %#v", repo)
			}
			if _, hasRunID := ci["run_id"]; hasRunID {
				t.Fatalf("ao-foundry ledger entry must not self-reference a mutable main CI run: %#v", ci)
			}
			evidence := fmt.Sprintf("%v", repo["verification_evidence"])
			for _, forbidden := range []string{"main CI run ", "Production Readiness Ops run ", "PR #"} {
				if strings.Contains(evidence, forbidden) {
					t.Fatalf("ao-foundry ledger entry must not self-reference mutable evidence %q: %#v", forbidden, repo["verification_evidence"])
				}
			}
		}
		delete(active, id)
	}
	if len(active) != 0 {
		t.Fatalf("ledger missing registry repos: %#v", active)
	}
	for _, excluded := range []string{"ao-operator", "ao-runtime", "ao-control-plane", "ao-conductor", "agy-swarms", "codex-cron"} {
		if strings.Contains(fmt.Sprintf("%v", ledgerObject), excluded) {
			t.Fatalf("ledger contains excluded repo %q: %#v", excluded, ledgerObject)
		}
	}
}

func TestActiveStackReadinessLedgerIncludesReleaseHandoffChain(t *testing.T) {
	ledger, err := readArbitraryJSON("examples/readiness/active-stack-readiness.ledger.json")
	if err != nil {
		t.Fatalf("read active stack readiness ledger: %v", err)
	}
	ledgerObject, ok := ledger.(map[string]any)
	if !ok {
		t.Fatalf("ledger is not an object: %#v", ledger)
	}
	handoff, ok := ledgerObject["release_handoff"].(map[string]any)
	if !ok {
		t.Fatalf("ledger release_handoff missing or not object: %#v", ledgerObject["release_handoff"])
	}
	if handoff["status"] != "ready" {
		t.Fatalf("release_handoff status = %#v, want ready", handoff["status"])
	}
	gates, ok := handoff["gates"].([]any)
	if !ok {
		t.Fatalf("release_handoff gates missing or not array: %#v", handoff["gates"])
	}
	required := map[string][]string{
		"foundry-release-candidate": {
			"go run ./cmd/foundry release candidate validate --ledger examples/readiness/active-spine-release-candidate.ledger.json",
			"go run ./cmd/foundry release candidate active-stack-parity --ledger examples/readiness/active-spine-release-candidate.ledger.json --readiness-ledger examples/readiness/active-stack-readiness.ledger.json",
		},
		"forge-release-candidate-handoff": {
			"forge release-candidate validate --candidate examples/release-preview/release-candidate.v0.1.example.json",
		},
		"covenant-policy-spine": {
			"covenant policy spine --json",
			"covenant.policy-spine-result.v1",
		},
	}
	seen := map[string]bool{}
	for _, rawGate := range gates {
		gate, ok := rawGate.(map[string]any)
		if !ok {
			t.Fatalf("release_handoff gate is not object: %#v", rawGate)
		}
		name, _ := gate["name"].(string)
		evidence, ok := gate["evidence"].([]any)
		if !ok {
			t.Fatalf("release_handoff gate %q evidence missing or not array: %#v", name, gate["evidence"])
		}
		for requiredName, requiredEvidence := range required {
			if name != requiredName {
				continue
			}
			seen[requiredName] = true
			evidenceText := fmt.Sprintf("%v", evidence)
			for _, want := range requiredEvidence {
				if !strings.Contains(evidenceText, want) {
					t.Fatalf("release_handoff gate %q evidence missing %q: %#v", name, want, evidence)
				}
			}
		}
	}
	for requiredName := range required {
		if !seen[requiredName] {
			t.Fatalf("release_handoff missing gate %q: %#v", requiredName, gates)
		}
	}
}

func TestReadinessSnapshotRendersReadmeBlockFromLedger(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"readiness", "snapshot", "--ledger", "examples/readiness/active-stack-readiness.ledger.json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	readme, err := os.ReadFile(repoPath("README.md"))
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	want := readmeBlock(t, string(readme), "<!-- foundry:active-stack-readiness:start -->", "<!-- foundry:active-stack-readiness:end -->")
	if stdout.String() != want {
		t.Fatalf("snapshot output does not match README block\nwant:\n%s\ngot:\n%s", want, stdout.String())
	}
	for _, required := range []string{
		"Release handoff gates:",
		"foundry-release-candidate",
		"forge-release-candidate-handoff",
		"covenant-policy-spine",
	} {
		if !strings.Contains(stdout.String(), required) {
			t.Fatalf("snapshot missing release handoff detail %q:\n%s", required, stdout.String())
		}
	}
}

func TestReadinessEvidenceCheckRejectsStaleSiblingRunEvidence(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "active-stack-github-runs-report.json")
	report := `{
  "schema_version": "ao.foundry.active-stack-github-runs-report.v0.1",
  "status": "ready",
  "branch": "main",
  "generated_at": "2026-06-23T12:00:00Z",
  "repositories": [
    {
      "repository": "uesugitorachiyo/ao-forge",
      "latest_ci": {
        "workflow": "ci.yml",
        "status": "completed",
        "conclusion": "success",
        "run_id": "99999999999",
        "url": "https://github.com/uesugitorachiyo/ao-forge/actions/runs/99999999999"
      },
      "latest_ops": {
        "workflow": "production-readiness-ops.yml",
        "status": "completed",
        "conclusion": "success",
        "run_id": "28056174653",
        "url": "https://github.com/uesugitorachiyo/ao-forge/actions/runs/28056174653"
      }
    }
  ],
  "next_actions": []
}
`
	if err := os.WriteFile(reportPath, []byte(report), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"readiness", "evidence-check",
		"--ledger", "examples/readiness/active-stack-readiness.ledger.json",
		"--github-runs-report", reportPath,
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for stale run evidence; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "ao-forge latest_ci run 99999999999 is not recorded") {
		t.Fatalf("expected stale ao-forge CI evidence error, got %q", stderr.String())
	}
}

func TestReadinessLedgerRefreshProposalRendersRunUpdates(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "ledger-refresh-proposal.md")
	reportPath := filepath.Join(t.TempDir(), "active-stack-github-runs-report.json")
	report := `{
  "schema_version": "ao.foundry.active-stack-github-runs-report.v0.1",
  "status": "ready",
  "branch": "main",
  "current_repo": "ao-foundry",
  "current_repo_skipped": false,
  "generated_at": "2026-06-23T12:00:00Z",
  "repositories": [
    {
      "repository": "uesugitorachiyo/ao-foundry",
      "latest_ci": {
        "workflow": "ci.yml",
        "status": "completed",
        "conclusion": "success",
        "run_id": "99999999991",
        "url": "https://github.com/uesugitorachiyo/ao-foundry/actions/runs/99999999991"
      },
      "latest_ops": {
        "workflow": "production-readiness-ops.yml",
        "status": "completed",
        "conclusion": "success",
        "run_id": "99999999992",
        "url": "https://github.com/uesugitorachiyo/ao-foundry/actions/runs/99999999992"
      }
    },
    {
      "repository": "uesugitorachiyo/ao-forge",
      "latest_ci": {
        "workflow": "ci.yml",
        "status": "completed",
        "conclusion": "success",
        "run_id": "28040935640"
      },
      "latest_ops": {
        "workflow": "production-readiness-ops.yml",
        "status": "completed",
        "conclusion": "success",
        "run_id": "28056174653"
      }
    }
  ],
  "next_actions": []
}
`
	if err := os.WriteFile(reportPath, []byte(report), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"readiness", "ledger-refresh-proposal",
		"--ledger", "examples/readiness/active-stack-readiness.ledger.json",
		"--github-runs-report", reportPath,
		"--out", outPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "ledger_refresh_proposal="+outPath) {
		t.Fatalf("expected proposal output path, got %q", stdout.String())
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read proposal: %v", err)
	}
	proposal := string(data)
	for _, want := range []string{
		"# Active Stack Ledger Refresh Proposal",
		"Generated from: " + reportPath,
		"| ao-foundry | ci.yml | 99999999991 | ignored_current_self_evidence |",
		"| ao-foundry | production-readiness-ops.yml | 99999999992 | ignored_current_self_evidence |",
		"| ao-forge | ci.yml | 28040935640 | already_recorded |",
		"Update examples/readiness/active-stack-readiness.ledger.json",
		"go run ./cmd/foundry readiness snapshot --ledger examples/readiness/active-stack-readiness.ledger.json",
	} {
		if !strings.Contains(proposal, want) {
			t.Fatalf("proposal missing %q:\n%s", want, proposal)
		}
	}
}

func TestReadinessLedgerRefreshProposalApplyUpdatesLedgerAndReadme(t *testing.T) {
	dir := t.TempDir()
	ledgerPath := filepath.Join(dir, "active-stack-readiness.ledger.json")
	readmePath := filepath.Join(dir, "README.md")
	reportPath := filepath.Join(dir, "active-stack-github-runs-report.json")
	copyFileForTest(t, repoPath("examples/readiness/active-stack-readiness.ledger.json"), ledgerPath)
	copyFileForTest(t, repoPath("README.md"), readmePath)
	report := `{
  "schema_version": "ao.foundry.active-stack-github-runs-report.v0.1",
  "status": "ready",
  "branch": "main",
  "current_repo": "ao-foundry",
  "current_repo_skipped": false,
  "generated_at": "2026-06-23T12:00:00Z",
  "repositories": [
    {
      "repository": "uesugitorachiyo/ao-foundry",
      "latest_ci": {
        "workflow": "ci.yml",
        "status": "completed",
        "conclusion": "success",
        "run_id": "99999999991",
        "display_title": "Apply ledger refresh automation (#99)"
      },
      "latest_ops": {
        "workflow": "production-readiness-ops.yml",
        "status": "completed",
        "conclusion": "success",
        "run_id": "99999999992"
      }
    }
  ],
  "next_actions": []
}
`
	if err := os.WriteFile(reportPath, []byte(report), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"readiness", "ledger-refresh-proposal",
		"--ledger", ledgerPath,
		"--github-runs-report", reportPath,
		"--readme", readmePath,
		"--apply",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "ledger_refresh_apply=ready") {
		t.Fatalf("expected apply confirmation, got %q", stdout.String())
	}
	ledger, err := os.ReadFile(ledgerPath)
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	ledgerText := string(ledger)
	for _, forbidden := range []string{"main CI run 99999999991", "Production Readiness Ops run 99999999992", "PR #99 merged"} {
		if strings.Contains(ledgerText, forbidden) {
			t.Fatalf("apply must not add current-repo mutable evidence %q:\n%s", forbidden, ledgerText)
		}
	}
	if strings.Contains(ledgerText, `"run_id": "99999999991"`) {
		t.Fatalf("apply must not add self-referential ao-foundry ci.run_id:\n%s", ledgerText)
	}
	readme, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	for _, forbidden := range []string{"main CI run 99999999991", "Production Readiness Ops run 99999999992", "PR #99 merged"} {
		if strings.Contains(string(readme), forbidden) {
			t.Fatalf("README snapshot must not include current-repo mutable evidence %q:\n%s", forbidden, string(readme))
		}
	}
}

func TestReadinessLedgerRefreshProposalIgnoresCurrentRepoEvidenceRefreshLoop(t *testing.T) {
	testReadinessLedgerRefreshProposalIgnoresCurrentRepoEvidenceRefreshLoop(t, "Refresh Foundry readiness evidence (#99)")
}

func TestReadinessLedgerRefreshProposalIgnoresCurrentRepoFoundryEvidenceRefreshLoop(t *testing.T) {
	testReadinessLedgerRefreshProposalIgnoresCurrentRepoEvidenceRefreshLoop(t, "Refresh Foundry evidence after loop guard (#99)")
}

func testReadinessLedgerRefreshProposalIgnoresCurrentRepoEvidenceRefreshLoop(t *testing.T, displayTitle string) {
	t.Helper()
	outPath := filepath.Join(t.TempDir(), "ledger-refresh-proposal.md")
	reportPath := filepath.Join(t.TempDir(), "active-stack-github-runs-report.json")
	report := `{
  "schema_version": "ao.foundry.active-stack-github-runs-report.v0.1",
  "status": "ready",
  "branch": "main",
  "current_repo": "ao-foundry",
  "current_repo_skipped": false,
  "generated_at": "2026-06-23T12:00:00Z",
  "repositories": [
    {
      "repository": "uesugitorachiyo/ao-foundry",
      "latest_ci": {
        "workflow": "ci.yml",
        "status": "completed",
        "conclusion": "success",
        "run_id": "99999999991",
        "display_title": "` + displayTitle + `"
      },
      "latest_ops": {
        "workflow": "production-readiness-ops.yml",
        "status": "completed",
        "conclusion": "success",
        "run_id": "99999999992"
      }
    },
    {
      "repository": "uesugitorachiyo/ao-forge",
      "latest_ci": {
        "workflow": "ci.yml",
        "status": "completed",
        "conclusion": "success",
        "run_id": "28040935640"
      },
      "latest_ops": {
        "workflow": "production-readiness-ops.yml",
        "status": "completed",
        "conclusion": "success",
        "run_id": "28056174653"
      }
    }
  ],
  "next_actions": []
}
`
	if err := os.WriteFile(reportPath, []byte(report), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"readiness", "ledger-refresh-proposal",
		"--ledger", "examples/readiness/active-stack-readiness.ledger.json",
		"--github-runs-report", reportPath,
		"--out", outPath,
		"--fail-on-non-current-update",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read proposal: %v", err)
	}
	proposal := string(data)
	for _, want := range []string{
		"| ao-foundry | ci.yml | 99999999991 | ignored_current_refresh_loop |",
		"| ao-foundry | production-readiness-ops.yml | 99999999992 | ignored_current_refresh_loop |",
		"| ao-forge | ci.yml | 28040935640 | already_recorded |",
	} {
		if !strings.Contains(proposal, want) {
			t.Fatalf("proposal missing %q:\n%s", want, proposal)
		}
	}
	if strings.Contains(proposal, "| ao-foundry | ci.yml | 99999999991 | update |") {
		t.Fatalf("current repo evidence refresh loop must not stay actionable:\n%s", proposal)
	}
}

func TestReadinessLedgerRefreshProposalFailsOnNonCurrentUpdates(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "active-stack-github-runs-report.json")
	report := `{
  "schema_version": "ao.foundry.active-stack-github-runs-report.v0.1",
  "status": "ready",
  "branch": "main",
  "current_repo": "ao-foundry",
  "current_repo_skipped": false,
  "generated_at": "2026-06-23T12:00:00Z",
  "repositories": [
    {
      "repository": "uesugitorachiyo/ao-forge",
      "latest_ci": {
        "workflow": "ci.yml",
        "status": "completed",
        "conclusion": "success",
        "run_id": "99999999993"
      },
      "latest_ops": {
        "workflow": "production-readiness-ops.yml",
        "status": "completed",
        "conclusion": "success",
        "run_id": "28056174653"
      }
    }
  ],
  "next_actions": []
}
`
	if err := os.WriteFile(reportPath, []byte(report), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"readiness", "ledger-refresh-proposal",
		"--ledger", "examples/readiness/active-stack-readiness.ledger.json",
		"--github-runs-report", reportPath,
		"--fail-on-non-current-update",
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for stale sibling proposal; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "ao-forge ci.yml has update row") {
		t.Fatalf("expected non-current update failure, got %q", stderr.String())
	}
}

func TestReadinessLedgerRefreshProposalAllowsCurrentRepoSelfWindow(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "active-stack-github-runs-report.json")
	report := `{
  "schema_version": "ao.foundry.active-stack-github-runs-report.v0.1",
  "status": "ready",
  "branch": "main",
  "current_repo": "ao-foundry",
  "current_repo_skipped": true,
  "generated_at": "2026-06-23T12:00:00Z",
  "repositories": [
    {
      "repository": "uesugitorachiyo/ao-foundry",
      "latest_ci": {
        "workflow": "ci.yml",
        "status": "completed",
        "conclusion": "success",
        "run_id": "99999999994",
        "display_title": "Self window (#100)"
      },
      "latest_ops": {
        "workflow": "production-readiness-ops.yml",
        "status": "in_progress",
        "conclusion": "",
        "run_id": "99999999995"
      }
    },
    {
      "repository": "uesugitorachiyo/ao-forge",
      "latest_ci": {
        "workflow": "ci.yml",
        "status": "completed",
        "conclusion": "success",
        "run_id": "28040935640"
      },
      "latest_ops": {
        "workflow": "production-readiness-ops.yml",
        "status": "completed",
        "conclusion": "success",
        "run_id": "28056174653"
      }
    }
  ],
  "next_actions": []
}
`
	if err := os.WriteFile(reportPath, []byte(report), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"readiness", "ledger-refresh-proposal",
		"--ledger", "examples/readiness/active-stack-readiness.ledger.json",
		"--github-runs-report", reportPath,
		"--fail-on-non-current-update",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("current-repo self window should not fail; code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
}

func TestReadinessRollupReportsReadyAndManualPromotionGate(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "active-stack-github-runs-report.json")
	outPath := filepath.Join(dir, "active-stack-production-readiness-rollup.json")
	markdownPath := filepath.Join(dir, "active-stack-production-readiness-rollup.md")
	writeActiveStackGithubRunsReportForTest(t, reportPath, nil)

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"readiness", "rollup",
		"--ledger", "examples/readiness/active-stack-readiness.ledger.json",
		"--github-runs-report", reportPath,
		"--out", outPath,
		"--markdown-out", markdownPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "readiness_rollup="+outPath) || !strings.Contains(stdout.String(), "status=ready") {
		t.Fatalf("expected rollup output path and ready status, got %q", stdout.String())
	}

	var rollup map[string]any
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read rollup: %v", err)
	}
	if err := json.Unmarshal(data, &rollup); err != nil {
		t.Fatalf("unmarshal rollup: %v", err)
	}
	if rollup["schema_version"] != "ao.foundry.active-stack-production-readiness-rollup.v0.1" || rollup["status"] != "ready" {
		t.Fatalf("unexpected rollup identity/status: %#v", rollup)
	}
	if !rollupRowsContain(t, rollup["release_handoff"], "name", "signed-smoke-release-gate", "classification", "promotion_manual_gate") {
		t.Fatalf("signed-smoke gate was not classified as a manual promotion gate: %#v", rollup["release_handoff"])
	}
	if !rollupRowsContain(t, rollup["drift"], "repository", "ao-foundry", "action", "ignored_current_refresh_loop") {
		t.Fatalf("current repo refresh row was not suppressed in drift: %#v", rollup["drift"])
	}
	if !rollupRowsContain(t, rollup["drift"], "repository", "ao-forge", "action", "already_recorded") {
		t.Fatalf("sibling recorded evidence missing from drift: %#v", rollup["drift"])
	}

	markdown, err := os.ReadFile(markdownPath)
	if err != nil {
		t.Fatalf("read markdown rollup: %v", err)
	}
	for _, want := range []string{
		"# Active Stack Production Readiness Rollup",
		"Status: ready",
		"| signed-smoke-release-gate | manual_required | promotion_manual_gate |",
		"| ao-foundry | ci.yml | 99999999991 | ignored_current_refresh_loop |",
	} {
		if !strings.Contains(string(markdown), want) {
			t.Fatalf("markdown rollup missing %q:\n%s", want, string(markdown))
		}
	}
}

func TestReadinessRollupBlocksNonCurrentRunUpdate(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "active-stack-github-runs-report.json")
	outPath := filepath.Join(dir, "active-stack-production-readiness-rollup.json")
	writeActiveStackGithubRunsReportForTest(t, reportPath, map[string]string{"ao-forge": "99999999993"})

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"readiness", "rollup",
		"--ledger", "examples/readiness/active-stack-readiness.ledger.json",
		"--github-runs-report", reportPath,
		"--out", outPath,
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for non-current update; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "ao-forge ci.yml has update row") {
		t.Fatalf("expected non-current update failure, got %q", stderr.String())
	}

	var rollup map[string]any
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read blocked rollup: %v", err)
	}
	if err := json.Unmarshal(data, &rollup); err != nil {
		t.Fatalf("unmarshal blocked rollup: %v", err)
	}
	if rollup["status"] != "blocked" {
		t.Fatalf("expected blocked rollup, got %#v", rollup)
	}
	if rollup["blocked_repositories"] != float64(1) {
		t.Fatalf("expected one blocked repository, got %#v", rollup["blocked_repositories"])
	}
	if !rollupRowsContain(t, rollup["repositories"], "id", "ao-forge", "status", "blocked") {
		t.Fatalf("ao-forge repository row was not blocked: %#v", rollup["repositories"])
	}
}

func TestReadinessRollupAllowsCurrentRepoSelfWindow(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "active-stack-github-runs-report.json")
	outPath := filepath.Join(dir, "active-stack-production-readiness-rollup.json")
	writeActiveStackGithubRunsReportForTest(t, reportPath, map[string]string{"ao-foundry": "28027834300"})
	rewriteActiveStackGithubRunsReportForTest(t, reportPath, func(report *ActiveStackGithubRunsReport) {
		report.CurrentRepoSkipped = true
		for i := range report.Repositories {
			if report.Repositories[i].Repository != "uesugitorachiyo/ao-foundry" {
				continue
			}
			report.Repositories[i].LatestOps.Status = "in_progress"
			report.Repositories[i].LatestOps.Conclusion = ""
			report.Repositories[i].LatestOps.RunID = "99999999995"
		}
	})

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"readiness", "rollup",
		"--ledger", "examples/readiness/active-stack-readiness.ledger.json",
		"--github-runs-report", reportPath,
		"--out", outPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("current repo self window should not fail; code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	var rollup map[string]any
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read self-window rollup: %v", err)
	}
	if err := json.Unmarshal(data, &rollup); err != nil {
		t.Fatalf("unmarshal self-window rollup: %v", err)
	}
	if !rollupRowsContain(t, rollup["drift"], "repository", "ao-foundry", "action", "ignored_current_self_window") {
		t.Fatalf("current repo self window was not visible as ignored drift: %#v", rollup["drift"])
	}
}

func TestLoopPreflightPassesForExample(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"loop", "preflight", "--goal-run", goalFixture(), "--registry", registryFixture(), "--task", taskFixture()}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "loop preflight: ready") {
		t.Fatalf("expected ready preflight, got %q", stdout.String())
	}
}

func TestLoopLeaseAcquireRefusesActiveLease(t *testing.T) {
	leasePath := filepath.Join(t.TempDir(), "loop-lease.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"loop", "lease", "acquire", "--goal-run", goalFixture(), "--lease", leasePath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("first acquire returned %d; stderr=%s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"loop", "lease", "acquire", "--goal-run", goalFixture(), "--lease", leasePath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("second acquire succeeded; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "active lease") {
		t.Fatalf("expected active lease error, got %q", stderr.String())
	}
}

func TestLoopLeaseAcquireReportsStaleLease(t *testing.T) {
	leasePath := filepath.Join(t.TempDir(), "loop-lease.json")
	lease := LoopLease{
		SchemaVersion: "ao.foundry.loop-lease.v0.1",
		GoalID:        "ao-foundry-production-readiness",
		LeaseID:       "stale",
		AcquiredAtUTC: "2026-01-01T00:00:00Z",
		ExpiresAtUTC:  "2026-01-01T01:00:00Z",
		Status:        "active",
	}
	data, err := json.MarshalIndent(lease, "", "  ")
	if err != nil {
		t.Fatalf("marshal lease: %v", err)
	}
	if err := os.WriteFile(leasePath, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write lease: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"loop", "lease", "acquire", "--goal-run", goalFixture(), "--lease", leasePath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("acquire succeeded for stale lease; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "stale lease") {
		t.Fatalf("expected stale lease error, got %q", stderr.String())
	}
}

func TestLoopNextWritesForgeBrief(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "loop.forge-brief.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"loop", "next", "--goal-run", goalFixture(), "--registry", registryFixture(), "--task", taskFixture(), "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "forge_brief="+outPath) {
		t.Fatalf("expected brief path, got %q", stdout.String())
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected brief output: %v", err)
	}
}

func TestLoopPreflightBlocksTerminalGoal(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"loop", "preflight", "--goal-run", filepath.Join("testdata", "terminal.goal-run.json"), "--registry", registryFixture(), "--task", taskFixture()}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("preflight succeeded for terminal goal; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "goal readiness below 100") {
		t.Fatalf("expected goal readiness error, got %q", stderr.String())
	}
}

func TestLoopPreflightBlocksBudgetExhaustion(t *testing.T) {
	goalPath := writeBudgetGoal(t, 100, 100)
	var stdout, stderr bytes.Buffer
	code := Run([]string{"loop", "preflight", "--goal-run", goalPath, "--registry", registryFixture(), "--task", taskFixture()}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("preflight succeeded for exhausted budget; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "spend budget exhausted") {
		t.Fatalf("expected budget error, got %q", stderr.String())
	}
}

func TestApprovalRequestWritesTaskDigestAndSideEffects(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "approval-request.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"approval", "request", "--task", networkTaskFixture(), "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	var request ApprovalRequest
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read request: %v", err)
	}
	if err := json.Unmarshal(data, &request); err != nil {
		t.Fatalf("request is not JSON: %v", err)
	}
	if request.TaskSHA256 == "" || !sameStringSet(request.RequestedSideEffects, []string{"network access", "non-local execution"}) {
		t.Fatalf("unexpected request: %#v", request)
	}
}

func TestApprovalValidateAcceptsExample(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"approval", "validate", "--decision", filepath.Join("..", "..", "examples", "approvals", "network-read.approval-decision.json"), "--task", networkTaskFixture()}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "approval valid") {
		t.Fatalf("expected approval valid, got %q", stdout.String())
	}
}

func TestApprovalValidateRejectsExpiredDecision(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"approval", "validate", "--decision", filepath.Join("testdata", "expired-approval-decision.json"), "--task", networkTaskFixture()}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for expired decision; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "expired") {
		t.Fatalf("expected expired error, got %q", stderr.String())
	}
}

func TestApprovalValidateRejectsDigestMismatch(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"approval", "validate", "--decision", filepath.Join("testdata", "digest-mismatch-approval-decision.json"), "--task", networkTaskFixture()}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for digest mismatch; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "task digest mismatch") {
		t.Fatalf("expected digest mismatch error, got %q", stderr.String())
	}
}

func TestApprovalValidateRejectsBroadenedDecision(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"approval", "validate", "--decision", filepath.Join("testdata", "broadened-approval-decision.json"), "--task", networkTaskFixture()}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for broadened decision; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "broaden") {
		t.Fatalf("expected broaden error, got %q", stderr.String())
	}
}

func TestNextBlocksNonLocalTaskWithoutApproval(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "network.forge-brief.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"next", "--registry", registryFixture(), "--task", networkTaskFixture(), "--out", outPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success without approval; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "approval") {
		t.Fatalf("expected approval error, got %q", stderr.String())
	}
}

func TestNextReferencesApprovalInForgeBrief(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "network.forge-brief.json")
	decisionPath := filepath.Join("..", "..", "examples", "approvals", "network-read.approval-decision.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"next", "--registry", registryFixture(), "--task", networkTaskFixture(), "--approval-decision", decisionPath, "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read brief: %v", err)
	}
	expectedDecisionPath := filepath.ToSlash(filepath.Clean(decisionPath))
	if !bytes.Contains(data, []byte("approval decision: "+expectedDecisionPath)) {
		t.Fatalf("brief does not reference approval decision: %s", string(data))
	}
}

func TestEvalRunScoresBootstrapReady(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "bootstrap.eval-result.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"eval", "run", "--run", filepath.Join("..", "..", "examples", "runs", "ao-foundry-bootstrap.foundry-run.json"), "--scorecard", filepath.Join("..", "..", "examples", "evals", "bootstrap.scorecard.json"), "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	var result EvalResult
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("result is not JSON: %v", err)
	}
	if result.Status != "ready" || result.Score != 100 {
		t.Fatalf("expected ready 100 score, got %#v", result)
	}
}

func TestEvalRunFailsBelowThreshold(t *testing.T) {
	scorecard := EvalScorecard{
		SchemaVersion: "ao.foundry.eval-scorecard.v0.1",
		ScorecardID:   "bad-dimension",
		Threshold:     100,
		Dimensions:    []EvalDimensionDef{{Name: "unknown_dimension", MaxScore: 100}},
	}
	data, err := json.MarshalIndent(scorecard, "", "  ")
	if err != nil {
		t.Fatalf("marshal scorecard: %v", err)
	}
	scorecardPath := filepath.Join(t.TempDir(), "scorecard.json")
	if err := os.WriteFile(scorecardPath, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write scorecard: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"eval", "run", "--run", filepath.Join("..", "..", "examples", "runs", "ao-foundry-bootstrap.foundry-run.json"), "--scorecard", scorecardPath, "--out", filepath.Join(t.TempDir(), "result.json")}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success below threshold; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "below threshold") {
		t.Fatalf("expected threshold error, got %q", stderr.String())
	}
}

func TestReadinessTraceInspectSummarizesTerminalSpan(t *testing.T) {
	tracePath := filepath.Join(t.TempDir(), "readiness.trace.jsonl")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"readiness", "audit", "--registry", registryFixture(), "--task", taskFixture(), "--trace", tracePath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("readiness returned %d; stderr=%s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"trace", "inspect", "--trace", tracePath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("trace inspect returned %d; stderr=%s", code, stderr.String())
	}
	for _, want := range []string{"spans=1", "failed_spans=0", "evidence_refs=2"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("trace output missing %q: %s", want, stdout.String())
		}
	}
}

func TestNextTrace(t *testing.T) {
	tracePath := filepath.Join(t.TempDir(), "next.trace.jsonl")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"next", "--registry", registryFixture(), "--task", taskFixture(), "--trace", tracePath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("next returned %d; stderr=%s", code, stderr.String())
	}
	assertTraceInspectPasses(t, tracePath)
}

func TestRunIngestTrace(t *testing.T) {
	tracePath := filepath.Join(t.TempDir(), "run-ingest.trace.jsonl")
	outPath := filepath.Join(t.TempDir(), "run.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"run", "ingest", "--registry", registryFixture(), "--task", taskFixture(), "--packet", validPacketFixture(), "--out", outPath, "--trace", tracePath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run ingest returned %d; stderr=%s", code, stderr.String())
	}
	assertTraceInspectPasses(t, tracePath)
}

func TestLoopNextTrace(t *testing.T) {
	tracePath := filepath.Join(t.TempDir(), "loop-next.trace.jsonl")
	outPath := filepath.Join(t.TempDir(), "loop.forge-brief.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"loop", "next", "--goal-run", goalFixture(), "--registry", registryFixture(), "--task", taskFixture(), "--out", outPath, "--trace", tracePath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("loop next returned %d; stderr=%s", code, stderr.String())
	}
	assertTraceInspectPasses(t, tracePath)
}

func TestApprovalValidateTrace(t *testing.T) {
	tracePath := filepath.Join(t.TempDir(), "approval.trace.jsonl")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"approval", "validate", "--decision", filepath.Join("..", "..", "examples", "approvals", "network-read.approval-decision.json"), "--task", networkTaskFixture(), "--trace", tracePath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("approval validate returned %d; stderr=%s", code, stderr.String())
	}
	assertTraceInspectPasses(t, tracePath)
}

func TestEvalRunTrace(t *testing.T) {
	tracePath := filepath.Join(t.TempDir(), "eval.trace.jsonl")
	outPath := filepath.Join(t.TempDir(), "eval.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"eval", "run", "--run", filepath.Join("..", "..", "examples", "runs", "ao-foundry-bootstrap.foundry-run.json"), "--scorecard", filepath.Join("..", "..", "examples", "evals", "bootstrap.scorecard.json"), "--out", outPath, "--trace", tracePath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("eval run returned %d; stderr=%s", code, stderr.String())
	}
	assertTraceInspectPasses(t, tracePath)
}

func TestTraceInspectRejectsMalformedTrace(t *testing.T) {
	tracePath := filepath.Join(t.TempDir(), "bad.trace.jsonl")
	if err := os.WriteFile(tracePath, []byte("{bad json}\n"), 0o644); err != nil {
		t.Fatalf("write trace: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"trace", "inspect", "--trace", tracePath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("trace inspect succeeded for malformed trace; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "malformed") {
		t.Fatalf("expected malformed error, got %q", stderr.String())
	}
}

func assertTraceInspectPasses(t *testing.T, tracePath string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run([]string{"trace", "inspect", "--trace", tracePath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("trace inspect returned %d; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "spans=1") {
		t.Fatalf("expected one span, got %q", stdout.String())
	}
}

func TestTraceInspectRejectsMissingTerminalSpan(t *testing.T) {
	tracePath := filepath.Join(t.TempDir(), "unterminated.trace.jsonl")
	span := TraceSpan{
		SchemaVersion: "ao.foundry.trace.v0.1",
		TraceID:       "trace-test",
		SpanID:        "span-test",
		Component:     "foundry",
		Operation:     "test",
		Status:        "running",
		StartedAt:     "2026-06-22T00:00:00Z",
		Attributes:    map[string]string{},
		EvidenceRefs:  []string{},
	}
	data, err := json.Marshal(span)
	if err != nil {
		t.Fatalf("marshal span: %v", err)
	}
	if err := os.WriteFile(tracePath, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write trace: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"trace", "inspect", "--trace", tracePath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("trace inspect succeeded without terminal span; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "terminal") {
		t.Fatalf("expected terminal error, got %q", stderr.String())
	}
}

func TestDemoStatusJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"demo", "status", "--registry", registryFixture(), "--task", taskFixture(), "--run", filepath.Join("..", "..", "examples", "runs", "ao-foundry-bootstrap.foundry-run.json"), "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	var status DemoStatus
	if err := json.Unmarshal(stdout.Bytes(), &status); err != nil {
		t.Fatalf("demo status is not JSON: %v; output=%s", err, stdout.String())
	}
	if status.Status != "ready" || status.RegistryID != "local-ao-stack" || len(status.Story) != 7 {
		t.Fatalf("unexpected demo status: %#v", status)
	}
}

func TestDemoScriptWritesPublicSafeMarkdown(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "demo.md")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"demo", "script", "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read script: %v", err)
	}
	for _, want := range []string{"AO Foundry", "above AO Forge", "does not replace AO Forge"} {
		if !bytes.Contains(data, []byte(want)) {
			t.Fatalf("demo script missing %q: %s", want, string(data))
		}
	}
	for _, forbidden := range []string{"/" + "Users/", "excluded" + "/", "private " + "handoff", "ANTI" + "GRAVITY"} {
		if bytes.Contains(data, []byte(forbidden)) {
			t.Fatalf("demo script contains forbidden text %q", forbidden)
		}
	}
}

func TestCompetitiveAuditScoresCurrentPublicReadinessAt100(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "competitive-readiness-audit.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"competitive", "audit", "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "competitive readiness: 100/100 status=ready") {
		t.Fatalf("expected 100 competitive readiness output, got %q", stdout.String())
	}
	var audit CompetitiveReadinessAudit
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read audit: %v", err)
	}
	if err := json.Unmarshal(data, &audit); err != nil {
		t.Fatalf("audit is not JSON: %v", err)
	}
	if audit.SchemaVersion != "ao.foundry.competitive-readiness-audit.v0.1" || audit.Status != "ready" || audit.Score != 100 {
		t.Fatalf("unexpected audit: %#v", audit)
	}
	if len(audit.Categories) != 9 {
		t.Fatalf("expected 9 competitive categories, got %d", len(audit.Categories))
	}
}

func TestCompetitiveAuditJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"competitive", "audit", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	var audit CompetitiveReadinessAudit
	if err := json.Unmarshal(stdout.Bytes(), &audit); err != nil {
		t.Fatalf("audit JSON output is not JSON: %v; output=%s", err, stdout.String())
	}
	if audit.Score != 100 || audit.Status != "ready" {
		t.Fatalf("unexpected JSON audit: %#v", audit)
	}
}

func TestCompetitiveAuditIgnoresScratchRuntimeBinaries(t *testing.T) {
	scratchDir := repoPath("tmp/live-tools")
	if err := os.MkdirAll(scratchDir, 0o755); err != nil {
		t.Fatalf("mkdir scratch dir: %v", err)
	}
	scratchFile := filepath.Join(scratchDir, "covenant")
	marker := "/" + "Users/example/private " + "handoff"
	if err := os.WriteFile(scratchFile, []byte(marker), 0o644); err != nil {
		t.Fatalf("write scratch file: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(repoPath("tmp/live-tools"))
	}()

	var stdout, stderr bytes.Buffer
	code := Run([]string{"competitive", "audit", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	var audit CompetitiveReadinessAudit
	if err := json.Unmarshal(stdout.Bytes(), &audit); err != nil {
		t.Fatalf("audit JSON output is not JSON: %v; output=%s", err, stdout.String())
	}
	if audit.Score != 100 || audit.Status != "ready" {
		t.Fatalf("scratch runtime file should not block public readiness: %#v", audit)
	}
}

func TestCIWorkflowRunsPulseSmoke(t *testing.T) {
	data, err := os.ReadFile(repoPath(".github/workflows/ci.yml"))
	if err != nil {
		t.Fatalf("read CI workflow: %v", err)
	}
	workflow := string(data)
	for _, want := range []string{
		"go run ./cmd/foundry contract fixtures validate",
		"go run ./cmd/foundry release handoff --candidate examples/readiness/active-spine-release-candidate.ledger.json",
		"go test ./internal/cli -run 'TestPulseRunBlocksStaleForgeLivePacket|TestPulseRunBlocksStaleControlPlaneReadback|TestPulseRunBlocksControlPlaneReadbackDigestMismatch' -v",
		"go test ./internal/cli -run TestSignedSmokeFreshnessCIFixtureValidates -v",
		"go test ./internal/cli -run TestStaleControlPlaneReadbackCIFixtureValidates -v",
		"go test ./internal/cli -run TestStaleFreshnessDerivedDecisionCIFixtureValidates -v",
		"go test ./internal/cli -run TestStaleControlPlaneDerivedDecisionCIFixtureValidates -v",
		"go run ./cmd/foundry pulse run --out tmp/pulse-smoke",
		"go run ./cmd/foundry pulse summarize-signed-smoke --pulse tmp/pulse-smoke/pulse-event.json --out tmp/pulse-smoke/signed-smoke-summary.json",
		"go run ./cmd/foundry trace inspect --trace tmp/pulse-smoke/pulse.trace.jsonl",
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("CI workflow missing pulse smoke command %q", want)
		}
	}
}

func TestProductionReadinessOpsWorkflowRunsBranchProtectionVerifier(t *testing.T) {
	data, err := os.ReadFile(repoPath(".github/workflows/production-readiness-ops.yml"))
	if err != nil {
		t.Fatalf("read production readiness ops workflow: %v", err)
	}
	workflow := string(data)
	for _, want := range []string{
		"name: production-readiness-ops",
		"workflow_dispatch:",
		"schedule:",
		"cron:",
		"GH_TOKEN: ${{ github.token }}",
		"scripts/verify-branch-protection.sh",
		"scripts/active-stack-github-runs-report.sh --out tmp/active-stack-github-runs-report.json",
		"go run ./cmd/foundry readiness evidence-check --ledger examples/readiness/active-stack-readiness.ledger.json --github-runs-report tmp/active-stack-github-runs-report.json",
		"go run ./cmd/foundry readiness ledger-refresh-proposal --ledger examples/readiness/active-stack-readiness.ledger.json --github-runs-report tmp/active-stack-github-runs-report.json --fail-on-non-current-update",
		"go run ./cmd/foundry readiness rollup --ledger examples/readiness/active-stack-readiness.ledger.json --github-runs-report tmp/active-stack-github-runs-report.json --out tmp/active-stack-production-readiness-rollup.json --markdown-out tmp/active-stack-production-readiness-rollup.md",
		"actions/upload-artifact",
		"active-stack-github-runs-report",
		"active-stack-production-readiness-rollup",
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("production readiness ops workflow missing %q", want)
		}
	}
	for _, forbidden := range []string{"gh pr merge", "git push", "-X PUT", "-X PATCH", "gh repo edit"} {
		if strings.Contains(workflow, forbidden) {
			t.Fatalf("production readiness ops workflow contains mutating command %q", forbidden)
		}
	}
}

func TestCleanCloneSmokeRunsContractFixtureValidation(t *testing.T) {
	data, err := os.ReadFile(repoPath("docs/operations/CLEAN-CLONE-SMOKE.md"))
	if err != nil {
		t.Fatalf("read clean-clone smoke docs: %v", err)
	}
	if !strings.Contains(string(data), "go run ./cmd/foundry contract fixtures validate") {
		t.Fatalf("clean-clone smoke docs missing contract fixture validation")
	}
}

func TestSignedSmokeEvidenceRetentionPolicyExists(t *testing.T) {
	data, err := os.ReadFile(repoPath("docs/operations/SIGNED-SMOKE-EVIDENCE-RETENTION.md"))
	if err != nil {
		t.Fatalf("read signed-smoke retention policy: %v", err)
	}
	policy := string(data)
	for _, want := range []string{
		"docs/evidence/pulse/local-live-smoke",
		"not included in the release manifest",
		"tmp/",
		"public-safe summaries",
	} {
		if !strings.Contains(policy, want) {
			t.Fatalf("signed-smoke retention policy missing %q", want)
		}
	}
}

func TestSignedSmokeReleaseGatePolicyExists(t *testing.T) {
	data, err := os.ReadFile(repoPath("docs/operations/SIGNED-SMOKE-RELEASE-GATE.md"))
	if err != nil {
		t.Fatalf("read signed-smoke release gate policy: %v", err)
	}
	policy := string(data)
	for _, want := range []string{
		"Manual release gate",
		"not required for pull_request or push CI",
		"required before release promotion",
		"workflow_dispatch",
		"signed_smoke=true",
		"AO2_CP_API_TOKEN",
		"freshness_summary.status=ready",
		"release_safe=true",
		"older than 24h",
		"docs/operations/SIGNED-SMOKE-EVIDENCE-RETENTION.md",
	} {
		if !strings.Contains(policy, want) {
			t.Fatalf("signed-smoke release gate policy missing %q", want)
		}
	}
	readme, err := os.ReadFile(repoPath("README.md"))
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	if !strings.Contains(string(readme), "Signed-smoke release gate") || !strings.Contains(string(readme), "docs/operations/SIGNED-SMOKE-RELEASE-GATE.md") {
		t.Fatalf("README missing signed-smoke release gate link")
	}
	ledger, err := os.ReadFile(repoPath("examples/readiness/active-stack-readiness.ledger.json"))
	if err != nil {
		t.Fatalf("read active stack readiness ledger: %v", err)
	}
	if !strings.Contains(string(ledger), "Signed-smoke release gate policy documented") {
		t.Fatalf("active stack readiness ledger missing signed-smoke policy next action")
	}
}

func TestReleaseChecklistCoversActiveStackHandoff(t *testing.T) {
	data, err := os.ReadFile(repoPath("docs/operations/RELEASE-CHECKLIST.md"))
	if err != nil {
		t.Fatalf("read release checklist: %v", err)
	}
	checklist := string(data)
	for _, want := range []string{
		"go run ./cmd/foundry release candidate validate --ledger examples/readiness/active-spine-release-candidate.ledger.json",
		"go run ./cmd/foundry release candidate active-stack-parity --ledger examples/readiness/active-spine-release-candidate.ledger.json --readiness-ledger examples/readiness/active-stack-readiness.ledger.json",
		"go run ./cmd/foundry release handoff --candidate examples/readiness/active-spine-release-candidate.ledger.json",
		"forge release-candidate validate --candidate examples/release-preview/release-candidate.v0.1.example.json",
		"covenant policy spine --json",
		"covenant.policy-spine-result.v1",
		"go run ./cmd/foundry readiness snapshot --ledger examples/readiness/active-stack-readiness.ledger.json",
		"go run ./cmd/foundry readiness ledger-refresh-proposal --ledger examples/readiness/active-stack-readiness.ledger.json",
		"--apply --readme README.md",
		"--fail-on-non-current-update",
		"go run ./cmd/foundry release candidate notes --ledger examples/readiness/active-spine-release-candidate.ledger.json",
		"diff -u",
		"workflow_dispatch signed_smoke=true",
		"release_safe=true",
	} {
		if !strings.Contains(checklist, want) {
			t.Fatalf("release checklist missing active-stack handoff item %q", want)
		}
	}
	for _, excluded := range []string{"ao-operator", "ao-runtime", "ao-control-plane", "ao-conductor", "agy-swarms", "codex-cron"} {
		if strings.Contains(checklist, excluded) {
			t.Fatalf("release checklist contains excluded scope %q", excluded)
		}
	}
}

func TestFreshSignedSmokeRunSummaryIsPublicSafe(t *testing.T) {
	data, err := os.ReadFile(repoPath("docs/evidence/pulse/local-live-smoke/FRESH-SIGNED-SMOKE-SUMMARY.md"))
	if err != nil {
		t.Fatalf("read fresh signed-smoke summary: %v", err)
	}
	summary := string(data)
	for _, want := range []string{
		"freshness=ready",
		"forge_live_packet=ready",
		"control_plane_readback=ready",
		"signed_smoke_summary=ready",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("fresh signed-smoke summary missing %q", want)
		}
	}
	for _, unsafe := range []string{"/" + "Users/", "ghp" + "_", "github" + "_pat_", "api" + "_key", "access" + "_token", strings.Repeat("x", 32)} {
		if strings.Contains(summary, unsafe) {
			t.Fatalf("fresh signed-smoke summary contains unsafe content %q", unsafe)
		}
	}
}

func TestCIWorkflowHasManualSignedSmoke(t *testing.T) {
	data, err := os.ReadFile(repoPath(".github/workflows/ci.yml"))
	if err != nil {
		t.Fatalf("read CI workflow: %v", err)
	}
	workflow := string(data)
	for _, want := range []string{
		"workflow_dispatch:",
		"signed_smoke:",
		"if: ${{ github.event_name == 'workflow_dispatch' && inputs.signed_smoke == 'true' }}",
		"AO2_CP_API_TOKEN: ${{ secrets.AO2_CP_API_TOKEN }}",
		"Prepare sibling AO workspace",
		"git clone --depth 1 https://github.com/uesugitorachiyo/ao-forge.git ../ao-forge",
		"git clone --depth 1 https://github.com/uesugitorachiyo/ao-covenant.git ../ao-covenant",
		"git clone --depth 1 https://github.com/uesugitorachiyo/ao2-control-plane.git ../ao2-control-plane",
		"cargo build -p ao2-cp-server",
		"go run ./cmd/foundry pulse signed-smoke-script --out tmp/signed-smoke.sh",
		"bash tmp/signed-smoke.sh",
		"Upload signed-smoke release evidence",
		"actions/upload-artifact@v4",
		"signed-smoke-release-evidence",
		"tmp/pulse-live/signed-smoke-summary.json",
		"tmp/release-promotion.live.json",
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("CI workflow missing manual signed-smoke detail %q", want)
		}
	}
}

func TestPulseEventLoopDocsIncludeSignedControlPlaneSmoke(t *testing.T) {
	data, err := os.ReadFile(repoPath("docs/operations/AO2-PULSE-EVENT-LOOP.md"))
	if err != nil {
		t.Fatalf("read pulse event loop docs: %v", err)
	}
	doc := string(data)
	for _, want := range []string{
		"ao2-cp-server --bind 127.0.0.1:",
		"--control-plane http://127.0.0.1:",
		"--forge-live-packet docs/evidence/pulse/",
		"control_plane_readback",
		"--signed-smoke-result tmp/pulse-live/signed-smoke-result.json",
		"signed_smoke_ingest",
		"go run ./cmd/foundry pulse derive-next",
		"--out tmp/pulse/ao2-loop-decision.json",
		"go run ./cmd/foundry pulse freshness --pulse tmp/pulse/pulse-event.json",
		"at least 32 characters",
		"older than 24h",
		"Control-plane readback receipts older than 24h",
		"go run ./cmd/foundry pulse summarize-signed-smoke",
		"go run ./cmd/foundry pulse signed-smoke-cleanup",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("pulse event loop docs missing signed smoke detail %q", want)
		}
	}
}

func TestPulseEventLoopDocsReferenceSignedSmokeFreshnessFixture(t *testing.T) {
	data, err := os.ReadFile(repoPath("docs/operations/AO2-PULSE-EVENT-LOOP.md"))
	if err != nil {
		t.Fatalf("read pulse event loop docs: %v", err)
	}
	doc := string(data)
	for _, want := range []string{
		"examples/ci/signed-smoke-freshness.pulse-event.json",
		"TestSignedSmokeFreshnessCIFixtureValidates",
		"freshness_summary.status=ready",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("pulse event loop docs missing signed-smoke freshness fixture detail %q", want)
		}
	}
}

func TestPulseWritesSignedSmokeScript(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "signed-pulse-smoke.sh")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"pulse", "signed-smoke-script", "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "signed_smoke_script="+outPath) {
		t.Fatalf("expected script output path, got %q", stdout.String())
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read script: %v", err)
	}
	script := string(data)
	for _, want := range []string{
		"set -euo pipefail",
		": \"${AO2_CP_API_TOKEN:?set AO2_CP_API_TOKEN}\"",
		"\"${#AO2_CP_API_TOKEN}\" -lt 32",
		"AO2_CP_API_TOKEN must be at least 32 characters",
		"ao2-cp-server --bind 127.0.0.1:18746",
		"--control-plane http://127.0.0.1:18746",
		"--forge-live-packet docs/evidence/pulse/local-live-smoke/factory-packet.json",
		"tmp/pulse-live/signed-smoke-result.json",
		"ao.foundry.signed-smoke-result.v0.1",
		"--signed-smoke-result tmp/pulse-live/signed-smoke-result.json",
		"go run ./cmd/foundry pulse summarize-signed-smoke --pulse tmp/pulse-live/pulse-event.json --out tmp/pulse-live/signed-smoke-summary.json",
		"go run ./cmd/foundry release promotion validate",
		"--signed-smoke-summary tmp/pulse-live/signed-smoke-summary.json",
		"signed_smoke_result=tmp/pulse-live/signed-smoke-result.json",
		"signed_smoke_summary=tmp/pulse-live/signed-smoke-summary.json",
		"release_promotion=tmp/release-promotion.live.json",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("signed smoke script missing %q", want)
		}
	}
	if strings.Contains(script, "/"+"Users/") || strings.Contains(script, "ghp"+"_") || strings.Contains(script, "github"+"_pat_") {
		t.Fatalf("signed smoke script contains unsafe local/private content: %s", script)
	}
}

func TestPulseSignedSmokeCleanupRemovesScratchAndKeepsEvidence(t *testing.T) {
	scratchFiles := []string{
		"tmp/live-tools/forge",
		"tmp/control-plane/state.json",
		"tmp/signed-smoke.sh",
		"tmp/signed-smoke-preflight.json",
		"tmp/pulse-live/pulse-event.json",
		"tmp/pulse-live-bundled/pulse-event.json",
	}
	for _, path := range scratchFiles {
		abs := repoPath(path)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir scratch parent: %v", err)
		}
		if err := os.WriteFile(abs, []byte("scratch\n"), 0o644); err != nil {
			t.Fatalf("write scratch %s: %v", path, err)
		}
	}
	evidencePath := repoPath("docs/evidence/pulse/local-live-smoke/cleanup-keep.json")
	if err := os.MkdirAll(filepath.Dir(evidencePath), 0o755); err != nil {
		t.Fatalf("mkdir evidence parent: %v", err)
	}
	if err := os.WriteFile(evidencePath, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write evidence marker: %v", err)
	}
	defer func() {
		_ = os.Remove(evidencePath)
	}()

	var stdout, stderr bytes.Buffer
	code := Run([]string{"pulse", "signed-smoke-cleanup"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "removed=") {
		t.Fatalf("cleanup stdout missing removed count: %q", stdout.String())
	}
	for _, path := range scratchFiles {
		if _, err := os.Stat(repoPath(path)); err == nil {
			t.Fatalf("cleanup left scratch file %s", path)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat scratch file %s: %v", path, err)
		}
	}
	if _, err := os.Stat(evidencePath); err != nil {
		t.Fatalf("cleanup should keep public evidence marker: %v", err)
	}
}

func TestPulseSignedSmokePreflightReportsPreparedWorkspace(t *testing.T) {
	workspace := t.TempDir()
	for _, dir := range []string{"ao-forge", "ao-covenant", filepath.Join("ao2-control-plane", "target", "debug")} {
		if err := os.MkdirAll(filepath.Join(workspace, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	serverPath := filepath.Join(workspace, "ao2-control-plane", "target", "debug", executableName("ao2-cp-server"))
	if err := os.WriteFile(serverPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write server: %v", err)
	}
	outPath := filepath.Join(t.TempDir(), "signed-smoke-preflight.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"pulse", "signed-smoke-preflight", "--workspace", workspace, "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "signed_smoke_preflight="+outPath) {
		t.Fatalf("expected preflight output path, got %q", stdout.String())
	}
	var preflight map[string]any
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read preflight: %v", err)
	}
	if err := json.Unmarshal(data, &preflight); err != nil {
		t.Fatalf("preflight is not JSON: %v", err)
	}
	if preflight["schema_version"] != "ao.foundry.signed-smoke-preflight.v0.1" || preflight["status"] != "ready" {
		t.Fatalf("unexpected preflight: %#v", preflight)
	}
	checks := preflight["checks"].([]any)
	if len(checks) != 3 {
		t.Fatalf("expected 3 preflight checks, got %#v", checks)
	}
}

func TestPulseSignedSmokePreflightBlocksMissingWorkspacePrerequisite(t *testing.T) {
	workspace := t.TempDir()
	outPath := filepath.Join(t.TempDir(), "signed-smoke-preflight.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"pulse", "signed-smoke-preflight", "--workspace", workspace, "--out", outPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for missing prerequisites; stdout=%s", stdout.String())
	}
	var preflight map[string]any
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read blocked preflight: %v", err)
	}
	if err := json.Unmarshal(data, &preflight); err != nil {
		t.Fatalf("preflight is not JSON: %v", err)
	}
	if preflight["status"] != "blocked" {
		t.Fatalf("expected blocked preflight, got %#v", preflight)
	}
	if !strings.Contains(stderr.String(), "signed smoke preflight blocked") {
		t.Fatalf("expected blocked stderr, got %q", stderr.String())
	}
}

func TestPulseIngestsSignedSmokeResult(t *testing.T) {
	dir := t.TempDir()
	resultPath := filepath.Join(dir, "signed-smoke-result.json")
	result := `{
  "schema_version": "ao.foundry.signed-smoke-result.v0.1",
  "status": "ready",
  "pulse_event": "tmp/pulse-live/pulse-event.json",
  "forge_live_packet": "docs/evidence/pulse/local-live-smoke/factory-packet.json",
  "control_plane_readback": "ready"
}
`
	if err := os.WriteFile(resultPath, []byte(result), 0o644); err != nil {
		t.Fatalf("write result: %v", err)
	}
	outPath := filepath.Join(dir, "signed-smoke-ingest.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"pulse", "ingest-signed-smoke", "--result", resultPath, "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "signed_smoke_ingest="+outPath) {
		t.Fatalf("expected ingest output path, got %q", stdout.String())
	}
	var ingest map[string]any
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read ingest: %v", err)
	}
	if err := json.Unmarshal(data, &ingest); err != nil {
		t.Fatalf("ingest is not JSON: %v", err)
	}
	if ingest["schema_version"] != "ao.foundry.signed-smoke-ingest.v0.1" || ingest["status"] != "ready" {
		t.Fatalf("unexpected ingest: %#v", ingest)
	}
	if len(ingest["result_sha256"].(string)) != 64 {
		t.Fatalf("missing result digest: %#v", ingest)
	}
}

func TestPulseWritesPublicSafeSignedSmokeSummary(t *testing.T) {
	dir := t.TempDir()
	pulsePath := filepath.Join(dir, "pulse-event.json")
	outPath := filepath.Join(dir, "signed-smoke-summary.json")
	pulse := PulseEvent{
		SchemaVersion: pulseEventSchema,
		PulseID:       "pulse-signed-smoke",
		Status:        "ready",
		Score:         100,
		MaxScore:      100,
		Artifacts: []PulseArtifact{
			{Name: "forge_live_attempt", Path: "/" + "Users/local/private/factory-packet.json", SchemaVersion: "ao.foundry.forge-live-attempt.v0.1", Status: "passed"},
			{Name: "control_plane_readback", Path: "tmp/pulse-live/control-plane-readback.json", SchemaVersion: "ao.foundry.control-plane-readback.v0.1", Status: "ready"},
			{Name: "signed_smoke_ingest", Path: "tmp/pulse-live/signed-smoke-ingest.json", SchemaVersion: "ao.foundry.signed-smoke-ingest.v0.1", Status: "ready"},
		},
		NextAction: "continue",
	}
	mustWriteJSONForTest(t, pulsePath, pulse)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"pulse", "summarize-signed-smoke", "--pulse", pulsePath, "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "signed_smoke_summary="+outPath) {
		t.Fatalf("expected summary output path, got %q", stdout.String())
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read summary: %v", err)
	}
	text := string(data)
	if strings.Contains(text, "/"+"Users/") || strings.Contains(text, "tmp/") || strings.Contains(text, "private") {
		t.Fatalf("summary leaked local source detail: %s", text)
	}
	var summary map[string]any
	if err := json.Unmarshal(data, &summary); err != nil {
		t.Fatalf("summary is not JSON: %v", err)
	}
	if summary["schema_version"] != "ao.foundry.signed-smoke-summary.v0.1" || summary["status"] != "ready" || summary["release_safe"] != true {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	if len(summary["evidence"].([]any)) != 3 {
		t.Fatalf("expected three public evidence entries, got %#v", summary["evidence"])
	}
}

func TestPulseRunBundlesSignedSmokeResult(t *testing.T) {
	dir := t.TempDir()
	resultPath := filepath.Join(dir, "signed-smoke-result.json")
	result := `{
  "schema_version": "ao.foundry.signed-smoke-result.v0.1",
  "status": "ready",
  "pulse_event": "tmp/pulse-live/pulse-event.json",
  "forge_live_packet": "docs/evidence/pulse/local-live-smoke/factory-packet.json",
  "control_plane_readback": "ready"
}
`
	if err := os.WriteFile(resultPath, []byte(result), 0o644); err != nil {
		t.Fatalf("write result: %v", err)
	}
	event := runPulseForEvent(t, []string{"pulse", "run", "--out", t.TempDir(), "--signed-smoke-result", resultPath})
	artifact := pulseArtifact(t, event, "signed_smoke_ingest")
	if artifact["status"] != "ready" {
		t.Fatalf("expected ready signed smoke ingest artifact, got %#v", artifact)
	}
	data := readPulseArtifactJSON(t, artifact)
	if data["schema_version"] != "ao.foundry.signed-smoke-ingest.v0.1" || data["status"] != "ready" {
		t.Fatalf("unexpected signed smoke ingest: %#v", data)
	}
}

func TestPulseRejectsMalformedSignedSmokeResult(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "blocked_status",
			body: `{
  "schema_version": "ao.foundry.signed-smoke-result.v0.1",
  "status": "blocked",
  "pulse_event": "tmp/pulse-live/pulse-event.json",
  "forge_live_packet": "docs/evidence/pulse/local-live-smoke/factory-packet.json",
  "control_plane_readback": "ready"
}
`,
			want: "status must be ready",
		},
		{
			name: "unsafe_path",
			body: `{
  "schema_version": "ao.foundry.signed-smoke-result.v0.1",
  "status": "ready",
  "pulse_event": "/tmp/pulse-live/pulse-event.json",
  "forge_live_packet": "docs/evidence/pulse/local-live-smoke/factory-packet.json",
  "control_plane_readback": "ready"
}
`,
			want: "unsafe signed smoke result path",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			resultPath := filepath.Join(dir, "signed-smoke-result.json")
			if err := os.WriteFile(resultPath, []byte(tc.body), 0o644); err != nil {
				t.Fatalf("write result: %v", err)
			}
			var stdout, stderr bytes.Buffer
			code := Run([]string{"pulse", "ingest-signed-smoke", "--result", resultPath, "--out", filepath.Join(dir, "ingest.json")}, &stdout, &stderr)
			if code == 0 {
				t.Fatalf("Run returned success for malformed result; stdout=%s", stdout.String())
			}
			if !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("expected stderr to contain %q, got %q", tc.want, stderr.String())
			}
		})
	}
}

func TestSignedSmokeContractFixturesExist(t *testing.T) {
	contracts := []struct {
		name    string
		schema  string
		version string
		target  any
	}{
		{
			name:    "foundry-signed-smoke-preflight-v0.1",
			schema:  "foundry-signed-smoke-preflight-v0.1.schema.json",
			version: "ao.foundry.signed-smoke-preflight.v0.1",
			target:  &SignedSmokePreflight{},
		},
		{
			name:    "foundry-signed-smoke-result-v0.1",
			schema:  "foundry-signed-smoke-result-v0.1.schema.json",
			version: "ao.foundry.signed-smoke-result.v0.1",
			target:  &SignedSmokeResult{},
		},
		{
			name:    "foundry-signed-smoke-ingest-v0.1",
			schema:  "foundry-signed-smoke-ingest-v0.1.schema.json",
			version: "ao.foundry.signed-smoke-ingest.v0.1",
			target:  &SignedSmokeIngest{},
		},
		{
			name:    "foundry-signed-smoke-summary-v0.1",
			schema:  "foundry-signed-smoke-summary-v0.1.schema.json",
			version: "ao.foundry.signed-smoke-summary.v0.1",
			target:  &SignedSmokeSummary{},
		},
	}
	for _, contract := range contracts {
		t.Run(contract.name, func(t *testing.T) {
			schemaData, err := os.ReadFile(repoPath(filepath.Join("docs", "contracts", contract.schema)))
			if err != nil {
				t.Fatalf("read schema: %v", err)
			}
			if !json.Valid(schemaData) || !strings.Contains(string(schemaData), contract.version) {
				t.Fatalf("schema missing version %q", contract.version)
			}
			validData, err := os.ReadFile(repoPath(filepath.Join("examples", "contract-fixtures", "valid", contract.name+".json")))
			if err != nil {
				t.Fatalf("read valid fixture: %v", err)
			}
			if err := json.Unmarshal(validData, contract.target); err != nil {
				t.Fatalf("valid fixture is not typed JSON: %v", err)
			}
			invalidData, err := os.ReadFile(repoPath(filepath.Join("examples", "contract-fixtures", "invalid", contract.name+".json")))
			if err != nil {
				t.Fatalf("read invalid fixture: %v", err)
			}
			if !json.Valid(invalidData) || !strings.Contains(string(invalidData), `"status": "unsupported"`) {
				t.Fatalf("invalid fixture should be parseable JSON with unsupported status")
			}
		})
	}
}

func TestContractFixturesValidateAgainstSchemas(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"contract", "fixtures", "validate"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	output := stdout.String()
	fixtureCount := fmt.Sprintf("%d", len(publicSchemaNames()))
	for _, want := range []string{
		"contract_fixtures=valid",
		"valid_fixtures=" + fixtureCount,
		"invalid_fixtures=" + fixtureCount,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got %q", want, output)
		}
	}
}

func TestPulseWritesAO2LoopDecisionPacket(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "decision.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"pulse", "decision",
		"--action", "stop",
		"--reason", "Foundry generated the next AO2 loop decision.",
		"--next-task-id", "ingest-signed-smoke-result",
		"--out", outPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "ao2_loop_decision="+outPath) {
		t.Fatalf("expected decision output path, got %q", stdout.String())
	}
	var decision map[string]any
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read decision: %v", err)
	}
	if err := json.Unmarshal(data, &decision); err != nil {
		t.Fatalf("decision is not JSON: %v", err)
	}
	if decision["schema_version"] != "ao2.pulse-event-loop-decision.v1" {
		t.Fatalf("unexpected decision schema: %#v", decision)
	}
	eventLoop := decision["event_loop"].(map[string]any)
	if eventLoop["action"] != "stop" ||
		eventLoop["reason"] != "Foundry generated the next AO2 loop decision." ||
		eventLoop["next_task_id"] != "ingest-signed-smoke-result" {
		t.Fatalf("unexpected event_loop decision: %#v", eventLoop)
	}
}

func TestPulseDerivesAO2LoopDecisionFromReadyPulse(t *testing.T) {
	dir := t.TempDir()
	pulsePath := filepath.Join(dir, "pulse-event.json")
	outPath := filepath.Join(dir, "decision.json")
	pulse := PulseEvent{
		SchemaVersion: "ao.foundry.pulse-event.v0.1",
		PulseID:       "pulse-ready",
		Status:        "ready",
		Score:         100,
		MaxScore:      100,
		NextAction:    "Continue with governed AO Forge live execution.",
	}
	mustWriteJSONForTest(t, pulsePath, pulse)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"pulse", "derive-next", "--pulse", pulsePath, "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "next_task_id=continue-with-governed-ao-forge-live-execution") {
		t.Fatalf("expected derived task id in stdout, got %q", stdout.String())
	}
	var decision AO2LoopDecision
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read decision: %v", err)
	}
	if err := json.Unmarshal(data, &decision); err != nil {
		t.Fatalf("decision is not JSON: %v", err)
	}
	if decision.SchemaVersion != "ao2.pulse-event-loop-decision.v1" ||
		decision.EventLoop.Action != "stop" ||
		decision.EventLoop.NextTaskID != "continue-with-governed-ao-forge-live-execution" {
		t.Fatalf("unexpected derived decision: %#v", decision)
	}
}

func TestPulseFreshnessReportsPulseFreshnessSummary(t *testing.T) {
	dir := t.TempDir()
	pulsePath := filepath.Join(dir, "pulse-event.json")
	pulse := PulseEvent{
		SchemaVersion: "ao.foundry.pulse-event.v0.1",
		PulseID:       "pulse-freshness-status",
		Status:        "blocked",
		Score:         0,
		MaxScore:      100,
		Freshness: PulseFreshnessSummary{
			SchemaVersion:        "ao.foundry.pulse-freshness-summary.v0.1",
			Status:               "blocked",
			ForgeLivePacket:      "ready",
			ControlPlaneReadback: "stale",
			Explanation:          "operator-provided production freshness evidence is stale",
		},
		NextAction: "resolve pulse blockers and rerun",
	}
	mustWriteJSONForTest(t, pulsePath, pulse)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"pulse", "freshness", "--pulse", pulsePath}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run returned %d, want blocked exit 1; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		"freshness=blocked",
		"forge_live_packet=ready",
		"control_plane_readback=stale",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("freshness output missing %q: %s", want, stdout.String())
		}
	}
}

func TestPulseDerivesAO2LoopDecisionFromBlockedPulseCheck(t *testing.T) {
	dir := t.TempDir()
	pulsePath := filepath.Join(dir, "pulse-event.json")
	outPath := filepath.Join(dir, "decision.json")
	pulse := PulseEvent{
		SchemaVersion: "ao.foundry.pulse-event.v0.1",
		PulseID:       "pulse-blocked",
		Status:        "blocked",
		Score:         80,
		MaxScore:      100,
		Checks: []PulseCheck{
			{Name: "release_manifest", Status: "pass"},
			{Name: "signed_smoke_preflight", Status: "fail", Reason: "missing signed smoke input"},
		},
		NextAction: "Resolve failed pulse checks.",
	}
	mustWriteJSONForTest(t, pulsePath, pulse)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"pulse", "derive-next", "--pulse", pulsePath, "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var decision AO2LoopDecision
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read decision: %v", err)
	}
	if err := json.Unmarshal(data, &decision); err != nil {
		t.Fatalf("decision is not JSON: %v", err)
	}
	if decision.EventLoop.NextTaskID != "resolve-signed-smoke-preflight" {
		t.Fatalf("unexpected next task id: %#v", decision.EventLoop)
	}
}

func TestPulseDerivesAO2LoopDecisionFromStaleFreshness(t *testing.T) {
	dir := t.TempDir()
	pulsePath := filepath.Join(dir, "pulse-event.json")
	outPath := filepath.Join(dir, "decision.json")
	pulse := PulseEvent{
		SchemaVersion: "ao.foundry.pulse-event.v0.1",
		PulseID:       "pulse-stale-freshness",
		Status:        "blocked",
		Score:         80,
		MaxScore:      100,
		Freshness: PulseFreshnessSummary{
			SchemaVersion:        "ao.foundry.pulse-freshness-summary.v0.1",
			Status:               "blocked",
			ForgeLivePacket:      "stale",
			ControlPlaneReadback: "not_provided",
			Explanation:          "operator-provided production freshness evidence is stale",
		},
		Checks: []PulseCheck{
			{Name: "forge_live_attempt", Status: "fail", Reason: "older than 24h"},
		},
		NextAction: "Resolve failed pulse checks.",
	}
	mustWriteJSONForTest(t, pulsePath, pulse)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"pulse", "derive-next", "--pulse", pulsePath, "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var decision AO2LoopDecision
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read decision: %v", err)
	}
	if err := json.Unmarshal(data, &decision); err != nil {
		t.Fatalf("decision is not JSON: %v", err)
	}
	if decision.EventLoop.NextTaskID != "refresh-forge-live-packet" {
		t.Fatalf("unexpected next task id: %#v", decision.EventLoop)
	}
	if decision.EventLoop.Freshness.Status != "blocked" ||
		decision.EventLoop.Freshness.ForgeLivePacket != "stale" ||
		decision.EventLoop.Freshness.ControlPlaneReadback != "not_provided" {
		t.Fatalf("decision did not carry pulse freshness: %#v", decision.EventLoop.Freshness)
	}
}

func TestAO2PulseRunLoopConsumesFoundryDecisionPacket(t *testing.T) {
	ao2Path := repoPath("../ao2/target/debug/ao2")
	if _, err := os.Stat(ao2Path); err != nil {
		t.Skipf("AO2 binary unavailable: %v", err)
	}
	dir := t.TempDir()
	decisionPath := filepath.Join(dir, "decision.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"pulse", "decision",
		"--action", "stop",
		"--reason", "Foundry integration test generated the next AO2 loop decision.",
		"--next-task-id", "verify-foundry-generated-loop-decision",
		"--out", decisionPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	outDir := filepath.Join(dir, "ao2-run-loop")
	cmd := exec.Command(
		ao2Path,
		"pulse", "run-loop",
		"--command", "go test ./cmd/foundry",
		"--decision-file", decisionPath,
		"--max-chain-runs", "1",
		"--max-runtime-seconds", "60",
		"--out-dir", outDir,
		"--json",
	)
	cmd.Dir = repoPath(".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ao2 pulse run-loop failed: %v\n%s", err, string(out))
	}
	var run map[string]any
	if err := json.Unmarshal(out, &run); err != nil {
		t.Fatalf("AO2 run-loop output is not JSON: %v\n%s", err, string(out))
	}
	if run["status"] != "stopped" || run["next_task_id"] != "verify-foundry-generated-loop-decision" {
		t.Fatalf("unexpected AO2 run-loop output: %#v", run)
	}
}

func TestAO2LoopDecisionContractFixtureExists(t *testing.T) {
	schemaPath := repoPath("docs/contracts/foundry-ao2-loop-decision-v0.1.schema.json")
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	if !json.Valid(schemaData) {
		t.Fatalf("schema is not JSON")
	}
	schemaText := string(schemaData)
	for _, want := range []string{
		"ao2.pulse-event-loop-decision.v1",
		"next_task_id",
		"stop",
		"continue",
	} {
		if !strings.Contains(schemaText, want) {
			t.Fatalf("schema missing %q", want)
		}
	}
	fixturePath := repoPath("examples/contract-fixtures/valid/foundry-ao2-loop-decision-v0.1.json")
	fixtureData, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture AO2LoopDecision
	if err := json.Unmarshal(fixtureData, &fixture); err != nil {
		t.Fatalf("fixture is not decision JSON: %v", err)
	}
	if fixture.SchemaVersion != "ao2.pulse-event-loop-decision.v1" || fixture.EventLoop.NextTaskID == "" {
		t.Fatalf("unexpected fixture: %#v", fixture)
	}
	invalidFixturePath := repoPath("examples/contract-fixtures/invalid/foundry-ao2-loop-decision-v0.1.json")
	invalidFixtureData, err := os.ReadFile(invalidFixturePath)
	if err != nil {
		t.Fatalf("read invalid fixture: %v", err)
	}
	var invalidFixture AO2LoopDecision
	if err := json.Unmarshal(invalidFixtureData, &invalidFixture); err != nil {
		t.Fatalf("invalid fixture should remain parseable JSON: %v", err)
	}
	if invalidFixture.EventLoop.Action == "stop" || invalidFixture.EventLoop.Action == "continue" {
		t.Fatalf("invalid fixture should violate action enum: %#v", invalidFixture)
	}
}

func TestReleaseDryRunExcludesRuntimeScratchAndEvidence(t *testing.T) {
	scratchPaths := []string{
		filepath.Join(".ao2", "runs", "local-only.json"),
		filepath.Join("tmp", "release-scratch.json"),
		filepath.Join("docs", "evidence", "pulse", "local-scratch", "transient.json"),
	}
	for _, path := range scratchPaths {
		abs := repoPath(path)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir scratch parent: %v", err)
		}
		if err := os.WriteFile(abs, []byte(`{"scratch":true}`+"\n"), 0o644); err != nil {
			t.Fatalf("write scratch %s: %v", path, err)
		}
	}
	defer func() {
		_ = os.RemoveAll(repoPath(".ao2"))
		_ = os.RemoveAll(repoPath("tmp"))
		_ = os.RemoveAll(repoPath(filepath.Join("docs", "evidence", "pulse", "local-scratch")))
	}()

	outPath := filepath.Join(t.TempDir(), "release-manifest.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"release", "dry-run", "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var manifest ReleaseManifest
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("manifest is not JSON: %v", err)
	}
	manifestFiles := map[string]bool{}
	for _, file := range manifest.Files {
		manifestFiles[file.Path] = true
		if strings.HasPrefix(file.Path, ".ao2/") || strings.HasPrefix(file.Path, "tmp/") || strings.HasPrefix(file.Path, "docs/evidence/") {
			t.Fatalf("release manifest included runtime scratch path: %s", file.Path)
		}
	}
	if manifestFiles["docs/evidence/pulse/local-live-smoke/FRESH-SIGNED-SMOKE-SUMMARY.md"] {
		t.Fatalf("release manifest included public evidence summary that should remain outside release payload")
	}
	for _, want := range []string{
		"docs/contracts/foundry-ao2-loop-decision-v0.1.schema.json",
		"docs/contracts/foundry-signed-smoke-preflight-v0.1.schema.json",
		"docs/contracts/foundry-signed-smoke-result-v0.1.schema.json",
		"docs/contracts/foundry-signed-smoke-ingest-v0.1.schema.json",
		"docs/contracts/foundry-signed-smoke-summary-v0.1.schema.json",
		"examples/contract-fixtures/valid/foundry-ao2-loop-decision-v0.1.json",
		"examples/contract-fixtures/invalid/foundry-ao2-loop-decision-v0.1.json",
		"examples/contract-fixtures/valid/foundry-signed-smoke-ingest-v0.1.json",
		"examples/contract-fixtures/invalid/foundry-signed-smoke-ingest-v0.1.json",
		"examples/contract-fixtures/valid/foundry-signed-smoke-summary-v0.1.json",
		"examples/contract-fixtures/invalid/foundry-signed-smoke-summary-v0.1.json",
	} {
		if !manifestFiles[want] {
			t.Fatalf("release manifest missing loop contract artifact: %s", want)
		}
	}
	if !stringSliceContains(manifest.Checks, "contract fixtures valid") {
		t.Fatalf("release manifest checks missing contract fixture validation: %#v", manifest.Checks)
	}
}

func TestReleaseCandidateLedgerValidatesActiveSpine(t *testing.T) {
	ledgerPath := "examples/readiness/active-spine-release-candidate.ledger.json"
	var stdout, stderr bytes.Buffer
	code := Run([]string{"release", "candidate", "validate", "--ledger", ledgerPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		"release_candidate=active-spine-2026-06-23",
		"status=ready",
		"repos=3",
		"signed_smoke=manual_required",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("expected stdout to contain %q, got %q", want, stdout.String())
		}
	}

	schema, err := readArbitraryJSON(repoPath("docs/contracts/foundry-release-candidate-v0.1.schema.json"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("schema root is not object")
	}
	ledger, err := readArbitraryJSON(repoPath(ledgerPath))
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, ledger, "$"); err != nil {
		t.Fatalf("release candidate ledger should satisfy schema: %v", err)
	}
	ledgerObject, ok := ledger.(map[string]any)
	if !ok {
		t.Fatalf("ledger root is not object")
	}
	repos, ok := ledgerObject["active_spine"].([]any)
	if !ok {
		t.Fatalf("active_spine is not an array")
	}
	gotRepos := map[string]bool{}
	for _, rawRepo := range repos {
		repo, ok := rawRepo.(map[string]any)
		if !ok {
			t.Fatalf("active_spine item is not object: %#v", rawRepo)
		}
		id, _ := repo["id"].(string)
		gotRepos[id] = true
		if repo["status"] != "ready" {
			t.Fatalf("repo %s status = %v, want ready", id, repo["status"])
		}
	}
	for _, wantRepo := range []string{"ao2", "ao2-control-plane", "ao-foundry"} {
		if !gotRepos[wantRepo] {
			t.Fatalf("active spine missing repo %s: %#v", wantRepo, gotRepos)
		}
	}
	if len(gotRepos) != 3 {
		t.Fatalf("active spine repo count = %d, want 3: %#v", len(gotRepos), gotRepos)
	}
	renderedLedger := fmt.Sprintf("%v", ledgerObject)
	for _, excluded := range []string{"ao-operator", "ao-runtime", "ao-control-plane", "ao-conductor", "agy-swarms", "codex-cron"} {
		if strings.Contains(renderedLedger, excluded) {
			t.Fatalf("release candidate ledger should exclude out-of-scope repo %s", excluded)
		}
	}
}

func TestReleaseCandidateNotesRenderPromotionHandoff(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "active-spine-notes.md")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"release", "candidate", "notes",
		"--ledger", "examples/readiness/active-spine-release-candidate.ledger.json",
		"--promotion", "examples/contract-fixtures/valid/foundry-release-promotion-v0.1.json",
		"--out", outPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "release_candidate_notes="+outPath) {
		t.Fatalf("expected notes output path, got %q", stdout.String())
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read notes: %v", err)
	}
	notes := string(data)
	for _, want := range []string{
		"# Active Spine Release Candidate: active-spine-2026-06-23",
		"Status: ready",
		"Release safe: true",
		"Signed smoke pulse: pulse-signed-smoke",
		"| AO2 | execution-engine | ready |",
		"| AO2 Control Plane | evidence-observer | ready |",
		"| AO Foundry | operations-factory | ready |",
		"Tag plan",
		"active-spine-2026-06-23",
		"Promote only the bound active-spine candidate",
	} {
		if !strings.Contains(notes, want) {
			t.Fatalf("release candidate notes missing %q:\n%s", want, notes)
		}
	}
	for _, excluded := range []string{"ao-operator", "ao-runtime", "ao-control-plane", "ao-conductor", "agy-swarms", "codex-cron"} {
		if strings.Contains(notes, excluded) {
			t.Fatalf("release candidate notes contain excluded scope %q:\n%s", excluded, notes)
		}
	}
}

func TestReleaseCandidateActiveStackParityPassesForCurrentLedger(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"release", "candidate", "active-stack-parity",
		"--ledger", "examples/readiness/active-spine-release-candidate.ledger.json",
		"--readiness-ledger", "examples/readiness/active-stack-readiness.ledger.json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		"release_candidate_active_stack_parity=ready",
		"candidate=active-spine-2026-06-23",
		"repos_checked=3",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("expected stdout to contain %q, got %q", want, stdout.String())
		}
	}
}

func TestReleaseCandidateActiveStackParityBlocksStaleEvidence(t *testing.T) {
	tmp := t.TempDir()
	candidatePath := filepath.Join(tmp, "candidate.json")
	data, err := os.ReadFile(repoPath("examples/readiness/active-spine-release-candidate.ledger.json"))
	if err != nil {
		t.Fatalf("read candidate: %v", err)
	}
	stale := strings.ReplaceAll(string(data), "main CI run 28050638200", "main CI run 28016224096")
	if err := os.WriteFile(candidatePath, []byte(stale), 0o644); err != nil {
		t.Fatalf("write stale candidate: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"release", "candidate", "active-stack-parity",
		"--ledger", candidatePath,
		"--readiness-ledger", "examples/readiness/active-stack-readiness.ledger.json",
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned 0, want failure; stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	for _, want := range []string{
		"release candidate active-stack parity: ao2-control-plane missing active-stack evidence \"main CI run 28050638200\"",
		"release candidate active-stack parity: ao2-control-plane has stale evidence \"main CI run 28016224096\"",
	} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("expected stderr to contain %q, got %q", want, stderr.String())
		}
	}
}

func TestReleaseCandidateActiveStackParityBlocksUnrequiredMutableEvidence(t *testing.T) {
	tmp := t.TempDir()
	candidatePath := filepath.Join(tmp, "candidate.json")
	data, err := os.ReadFile(repoPath("examples/readiness/active-spine-release-candidate.ledger.json"))
	if err != nil {
		t.Fatalf("read candidate: %v", err)
	}
	mutated := strings.Replace(
		string(data),
		`"go run ./cmd/foundry release validate-manifest --manifest tmp/release-manifest.json"`,
		`"go run ./cmd/foundry release validate-manifest --manifest tmp/release-manifest.json",
        "main CI run 99999999991"`,
		1,
	)
	if err := os.WriteFile(candidatePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("write mutated candidate: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"release", "candidate", "active-stack-parity",
		"--ledger", candidatePath,
		"--readiness-ledger", "examples/readiness/active-stack-readiness.ledger.json",
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned 0, want failure; stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	want := `release candidate active-stack parity: ao-foundry has unrequired evidence "main CI run 99999999991"`
	if !strings.Contains(stderr.String(), want) {
		t.Fatalf("expected stderr to contain %q, got %q", want, stderr.String())
	}
}

func TestReleaseHandoffRunsCandidatePromotionNotesAndManifest(t *testing.T) {
	outDir := t.TempDir()
	promotionPath := filepath.Join(outDir, "release-promotion.json")
	notesPath := filepath.Join(outDir, "release-candidate.md")
	manifestPath := filepath.Join(outDir, "release-manifest.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"release", "handoff",
		"--candidate", "examples/readiness/active-spine-release-candidate.ledger.json",
		"--signed-smoke-summary", "examples/contract-fixtures/valid/foundry-signed-smoke-summary-v0.1.json",
		"--promotion-out", promotionPath,
		"--notes-out", notesPath,
		"--manifest-out", manifestPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		"release_handoff=ready",
		"candidate=active-spine-2026-06-23",
		"release_safe=true",
		"release_promotion=" + promotionPath,
		"release_candidate_notes=" + notesPath,
		"release_manifest=" + manifestPath,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("release handoff stdout missing %q: %s", want, stdout.String())
		}
	}
	for _, path := range []string{promotionPath, notesPath, manifestPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected release handoff artifact %s: %v", path, err)
		}
	}
	notes, err := os.ReadFile(notesPath)
	if err != nil {
		t.Fatalf("read notes: %v", err)
	}
	if !strings.Contains(string(notes), "Release safe: true") || !strings.Contains(string(notes), "Signed smoke pulse: pulse-signed-smoke") {
		t.Fatalf("release handoff notes missing promotion evidence:\n%s", string(notes))
	}
}

func TestReleasePromotionLedgerRequiresSignedSmokeSummary(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "release-promotion.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"release", "promotion", "validate",
		"--candidate", "examples/readiness/active-spine-release-candidate.ledger.json",
		"--signed-smoke-summary", "examples/contract-fixtures/valid/foundry-signed-smoke-summary-v0.1.json",
		"--out", outPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		"release_promotion=" + outPath,
		"candidate=active-spine-2026-06-23",
		"status=ready",
		"release_safe=true",
		"signed_smoke=pulse-signed-smoke",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("expected stdout to contain %q, got %q", want, stdout.String())
		}
	}

	schema, err := readArbitraryJSON(repoPath("docs/contracts/foundry-release-promotion-v0.1.schema.json"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("schema root is not object")
	}
	ledger, err := readArbitraryJSON(outPath)
	if err != nil {
		t.Fatalf("read promotion ledger: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, ledger, "$"); err != nil {
		t.Fatalf("release promotion ledger should satisfy schema: %v", err)
	}
	ledgerObject, ok := ledger.(map[string]any)
	if !ok {
		t.Fatalf("ledger root is not object")
	}
	if ledgerObject["release_safe"] != true || ledgerObject["status"] != "ready" {
		t.Fatalf("promotion ledger should be release-safe and ready: %#v", ledgerObject)
	}
	if ledgerObject["candidate_id"] != "active-spine-2026-06-23" || ledgerObject["signed_smoke_pulse_id"] != "pulse-signed-smoke" {
		t.Fatalf("promotion ledger did not bind candidate and signed-smoke summary: %#v", ledgerObject)
	}
	renderedLedger := fmt.Sprintf("%v", ledgerObject)
	for _, excluded := range []string{"ao-operator", "ao-runtime", "ao-control-plane", "ao-conductor", "agy-swarms", "codex-cron"} {
		if strings.Contains(renderedLedger, excluded) {
			t.Fatalf("release promotion ledger should exclude out-of-scope repo %s", excluded)
		}
	}
}

func TestPulseRunWritesGoldenLoopBundle(t *testing.T) {
	outDir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Run([]string{"pulse", "run", "--out", outDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "pulse_event="+filepath.Join(outDir, "pulse-event.json")) {
		t.Fatalf("expected pulse event output path, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "freshness=ready forge_live_packet=not_provided control_plane_readback=not_provided") {
		t.Fatalf("expected operator freshness status in stdout, got %q", stdout.String())
	}
	eventPath := filepath.Join(outDir, "pulse-event.json")
	data, err := os.ReadFile(eventPath)
	if err != nil {
		t.Fatalf("read pulse event: %v", err)
	}
	var event map[string]any
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("pulse event is not JSON: %v", err)
	}
	if event["schema_version"] != "ao.foundry.pulse-event.v0.1" || event["status"] != "ready" {
		t.Fatalf("unexpected pulse identity: %#v", event)
	}
	if event["score"] != float64(100) || event["max_score"] != float64(100) {
		t.Fatalf("expected 100/100 pulse score, got %#v", event)
	}
	assertPulseFreshnessSummary(t, event, "ready", "not_provided", "not_provided")
	artifacts, ok := event["artifacts"].([]any)
	if !ok || len(artifacts) == 0 {
		t.Fatalf("expected pulse artifacts, got %#v", event["artifacts"])
	}
	requiredArtifacts := map[string]bool{
		"production_readiness_audit":  false,
		"goal_readiness_audit":        false,
		"forge_brief":                 false,
		"forge_packet":                false,
		"policy_gate":                 false,
		"forge_live_attempt":          false,
		"control_plane_readback":      false,
		"foundry_run":                 false,
		"eval_result":                 false,
		"demo_status":                 false,
		"release_manifest":            false,
		"competitive_readiness_audit": false,
		"pulse_trace":                 false,
		"trace_inspect":               false,
	}
	for _, raw := range artifacts {
		artifact := raw.(map[string]any)
		name := artifact["name"].(string)
		if _, ok := requiredArtifacts[name]; ok {
			requiredArtifacts[name] = true
		}
		path := artifact["path"].(string)
		if runtime.GOOS != "windows" && strings.Contains(path, "/"+"Users/") {
			t.Fatalf("pulse artifact path is not public-safe: %q", path)
		}
		if strings.Contains(path, "private "+"handoff") {
			t.Fatalf("pulse artifact path is not public-safe: %q", path)
		}
		if len(artifact["sha256"].(string)) != 64 {
			t.Fatalf("pulse artifact missing digest: %#v", artifact)
		}
		if name != "pulse_trace" {
			if data, err := os.ReadFile(filepath.FromSlash(path)); err != nil {
				t.Fatalf("read artifact %s: %v", name, err)
			} else if !json.Valid(data) {
				t.Fatalf("artifact %s is not valid JSON: %s", name, string(data))
			}
		}
	}
	for name, found := range requiredArtifacts {
		if !found {
			t.Fatalf("pulse event missing artifact %q: %#v", name, artifacts)
		}
	}
}

func TestSignedSmokeFreshnessCIFixtureValidates(t *testing.T) {
	schema, err := readArbitraryJSON("docs/contracts/foundry-pulse-event-v0.1.schema.json")
	if err != nil {
		t.Fatalf("read pulse schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("pulse schema is not an object: %#v", schema)
	}
	fixture, err := readArbitraryJSON("examples/ci/signed-smoke-freshness.pulse-event.json")
	if err != nil {
		t.Fatalf("read signed-smoke freshness fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, fixture, "$"); err != nil {
		t.Fatalf("signed-smoke freshness fixture failed pulse schema: %v", err)
	}
	event, ok := fixture.(map[string]any)
	if !ok {
		t.Fatalf("fixture is not an object: %#v", fixture)
	}
	freshness, ok := event["freshness_summary"].(map[string]any)
	if !ok {
		t.Fatalf("fixture missing freshness summary: %#v", event)
	}
	if freshness["status"] != "ready" || freshness["forge_live_packet"] != "ready" || freshness["control_plane_readback"] != "ready" {
		t.Fatalf("fixture should model fresh signed-smoke evidence, got %#v", freshness)
	}
	if pulseArtifactFromRaw(t, event, "signed_smoke_ingest")["status"] != "ready" {
		t.Fatalf("fixture should include ready signed smoke ingest: %#v", event["artifacts"])
	}
}

func TestStaleControlPlaneReadbackCIFixtureValidates(t *testing.T) {
	schema, err := readArbitraryJSON("docs/contracts/foundry-pulse-event-v0.1.schema.json")
	if err != nil {
		t.Fatalf("read pulse schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("pulse schema is not an object: %#v", schema)
	}
	fixture, err := readArbitraryJSON("examples/ci/stale-control-plane-readback.pulse-event.json")
	if err != nil {
		t.Fatalf("read stale control-plane readback fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, fixture, "$"); err != nil {
		t.Fatalf("stale control-plane readback fixture failed pulse schema: %v", err)
	}
	event, ok := fixture.(map[string]any)
	if !ok {
		t.Fatalf("fixture is not an object: %#v", fixture)
	}
	freshness, ok := event["freshness_summary"].(map[string]any)
	if !ok {
		t.Fatalf("fixture missing freshness summary: %#v", event)
	}
	if freshness["status"] != "blocked" || freshness["forge_live_packet"] != "ready" || freshness["control_plane_readback"] != "stale" {
		t.Fatalf("fixture should model stale control-plane readback freshness, got %#v", freshness)
	}
	if pulseArtifactFromRaw(t, event, "control_plane_readback")["status"] != "stale" {
		t.Fatalf("fixture should include stale control-plane readback artifact: %#v", event["artifacts"])
	}
}

func TestStaleFreshnessDerivedDecisionCIFixtureValidates(t *testing.T) {
	schema, err := readArbitraryJSON("docs/contracts/foundry-ao2-loop-decision-v0.1.schema.json")
	if err != nil {
		t.Fatalf("read AO2 loop decision schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("AO2 loop decision schema is not an object: %#v", schema)
	}
	fixture, err := readArbitraryJSON("examples/ci/stale-freshness.ao2-loop-decision.json")
	if err != nil {
		t.Fatalf("read stale freshness decision fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, fixture, "$"); err != nil {
		t.Fatalf("stale freshness decision fixture failed schema: %v", err)
	}
	decision, ok := fixture.(map[string]any)
	if !ok {
		t.Fatalf("fixture is not an object: %#v", fixture)
	}
	eventLoop := decision["event_loop"].(map[string]any)
	if eventLoop["next_task_id"] != "refresh-forge-live-packet" {
		t.Fatalf("fixture should route stale Forge packet to refresh task: %#v", eventLoop)
	}
	freshness := eventLoop["freshness"].(map[string]any)
	if freshness["status"] != "blocked" || freshness["forge_live_packet"] != "stale" {
		t.Fatalf("fixture should model stale Forge live packet freshness: %#v", freshness)
	}
}

func TestStaleControlPlaneDerivedDecisionCIFixtureValidates(t *testing.T) {
	schema, err := readArbitraryJSON("docs/contracts/foundry-ao2-loop-decision-v0.1.schema.json")
	if err != nil {
		t.Fatalf("read AO2 loop decision schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("AO2 loop decision schema is not an object: %#v", schema)
	}
	fixture, err := readArbitraryJSON("examples/ci/stale-control-plane-readback.ao2-loop-decision.json")
	if err != nil {
		t.Fatalf("read stale control-plane decision fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, fixture, "$"); err != nil {
		t.Fatalf("stale control-plane decision fixture failed schema: %v", err)
	}
	decision, ok := fixture.(map[string]any)
	if !ok {
		t.Fatalf("fixture is not an object: %#v", fixture)
	}
	eventLoop := decision["event_loop"].(map[string]any)
	if eventLoop["next_task_id"] != "refresh-control-plane-readback" {
		t.Fatalf("fixture should route stale readback to refresh task: %#v", eventLoop)
	}
	freshness := eventLoop["freshness"].(map[string]any)
	if freshness["status"] != "blocked" || freshness["forge_live_packet"] != "ready" || freshness["control_plane_readback"] != "stale" {
		t.Fatalf("fixture should model stale control-plane readback freshness: %#v", freshness)
	}
}

func TestPulseRunWritesFailedEventForBlockedReadiness(t *testing.T) {
	outDir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Run([]string{"pulse", "run", "--registry", filepath.Join("testdata", "blocked-registry.json"), "--task", filepath.Join("testdata", "blocked-task.json"), "--out", outDir}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for blocked readiness; stdout=%s", stdout.String())
	}
	eventPath := filepath.Join(outDir, "pulse-event.json")
	data, err := os.ReadFile(eventPath)
	if err != nil {
		t.Fatalf("read failed pulse event: %v", err)
	}
	var event map[string]any
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("failed pulse event is not JSON: %v", err)
	}
	if event["schema_version"] != "ao.foundry.pulse-event.v0.1" || event["status"] != "blocked" {
		t.Fatalf("unexpected failed pulse event: %#v", event)
	}
	if event["score"] != float64(0) || event["max_score"] != float64(100) {
		t.Fatalf("failed pulse event should retain 0/100 score: %#v", event)
	}
	checks := event["checks"].([]any)
	last := checks[len(checks)-1].(map[string]any)
	if last["status"] != "fail" || !strings.Contains(last["reason"].(string), "production readiness") {
		t.Fatalf("expected production readiness failure check, got %#v", checks)
	}
}

func TestPulseRunRecordsBlockedForgeLiveAttemptByDefault(t *testing.T) {
	event := runPulseForEvent(t, []string{"pulse", "run", "--out", t.TempDir()})
	artifact := pulseArtifact(t, event, "forge_live_attempt")
	if artifact["status"] != "blocked" {
		t.Fatalf("expected blocked forge live attempt, got %#v", artifact)
	}
	data := readPulseArtifactJSON(t, artifact)
	if data["schema_version"] != "ao.foundry.forge-live-attempt.v0.1" || data["status"] != "blocked" {
		t.Fatalf("unexpected forge live attempt: %#v", data)
	}
}

func TestPulseRunRecordsProvidedForgeLivePacket(t *testing.T) {
	event := runPulseForEvent(t, []string{"pulse", "run", "--out", t.TempDir(), "--forge-live-packet", filepath.Join("..", "..", "examples", "packets", "ao-foundry-bootstrap.factory-packet.json")})
	artifact := pulseArtifact(t, event, "forge_live_attempt")
	if artifact["status"] != "passed" {
		t.Fatalf("expected passed forge live attempt, got %#v", artifact)
	}
	data := readPulseArtifactJSON(t, artifact)
	if data["status"] != "passed" || data["packet_schema_version"] != "ao.forge.factory-packet.v0.1" {
		t.Fatalf("unexpected provided forge live attempt: %#v", data)
	}
}

func TestPulseRunBlocksStaleForgeLivePacket(t *testing.T) {
	packetPath := filepath.Join(t.TempDir(), "factory-packet.json")
	packetData, err := os.ReadFile(repoPath("examples/packets/ao-foundry-bootstrap.factory-packet.json"))
	if err != nil {
		t.Fatalf("read packet fixture: %v", err)
	}
	if err := os.WriteFile(packetPath, packetData, 0o644); err != nil {
		t.Fatalf("write packet: %v", err)
	}
	stale := time.Now().Add(-49 * time.Hour)
	if err := os.Chtimes(packetPath, stale, stale); err != nil {
		t.Fatalf("make packet stale: %v", err)
	}

	event := runBlockedPulseForEvent(t, []string{"pulse", "run", "--out", t.TempDir(), "--forge-live-packet", packetPath})
	artifact := pulseArtifact(t, event, "forge_live_attempt")
	if artifact["status"] != "stale" {
		t.Fatalf("expected stale forge live attempt, got %#v", artifact)
	}
	data := readPulseArtifactJSON(t, artifact)
	if data["status"] != "stale" || !strings.Contains(data["explanation"].(string), "older than 24h") {
		t.Fatalf("unexpected stale forge live attempt: %#v", data)
	}
	assertPulseFreshnessSummary(t, event, "blocked", "stale", "not_provided")
}

func TestPulseRunRecordsUnavailableControlPlaneReadbackByDefault(t *testing.T) {
	event := runPulseForEvent(t, []string{"pulse", "run", "--out", t.TempDir()})
	artifact := pulseArtifact(t, event, "control_plane_readback")
	if artifact["status"] != "unavailable" {
		t.Fatalf("expected unavailable control-plane readback, got %#v", artifact)
	}
	data := readPulseArtifactJSON(t, artifact)
	if data["schema_version"] != "ao.foundry.control-plane-readback.v0.1" || data["status"] != "unavailable" {
		t.Fatalf("unexpected control-plane readback: %#v", data)
	}
}

func TestPulseRunRecordsProvidedControlPlaneReadback(t *testing.T) {
	receiptPath := filepath.Join(t.TempDir(), "control-plane-receipt.json")
	receiptSHA := strings.Repeat("a", 64)
	receipt := []byte(`{"schema_version":"ao2.cp-ingest-receipt.v1","status":"stored","sha256":"` + receiptSHA + `"}` + "\n")
	if err := os.WriteFile(receiptPath, receipt, 0o644); err != nil {
		t.Fatalf("write receipt: %v", err)
	}
	event := runPulseForEvent(t, []string{"pulse", "run", "--out", t.TempDir(), "--control-plane-receipt", receiptPath})
	artifact := pulseArtifact(t, event, "control_plane_readback")
	if artifact["status"] != "ready" {
		t.Fatalf("expected ready control-plane readback, got %#v", artifact)
	}
	data := readPulseArtifactJSON(t, artifact)
	if data["status"] != "ready" || data["receipt_schema_version"] != "ao2.cp-ingest-receipt.v1" {
		t.Fatalf("unexpected provided control-plane readback: %#v", data)
	}
	if strings.HasPrefix(data["source"].(string), "/") {
		t.Fatalf("control-plane readback source must not preserve absolute local path: %#v", data)
	}
}

func TestPulseRunBlocksStaleControlPlaneReadback(t *testing.T) {
	receiptPath := filepath.Join(t.TempDir(), "control-plane-receipt.json")
	receiptSHA := strings.Repeat("a", 64)
	receipt := []byte(`{"schema_version":"ao2.cp-ingest-receipt.v1","status":"stored","sha256":"` + receiptSHA + `"}` + "\n")
	if err := os.WriteFile(receiptPath, receipt, 0o644); err != nil {
		t.Fatalf("write receipt: %v", err)
	}
	stale := time.Now().Add(-49 * time.Hour)
	if err := os.Chtimes(receiptPath, stale, stale); err != nil {
		t.Fatalf("make receipt stale: %v", err)
	}

	event := runBlockedPulseForEvent(t, []string{"pulse", "run", "--out", t.TempDir(), "--control-plane-receipt", receiptPath})
	artifact := pulseArtifact(t, event, "control_plane_readback")
	if artifact["status"] != "stale" {
		t.Fatalf("expected stale control-plane readback, got %#v", artifact)
	}
	data := readPulseArtifactJSON(t, artifact)
	if data["status"] != "stale" || !strings.Contains(data["explanation"].(string), "older than 24h") {
		t.Fatalf("unexpected stale control-plane readback: %#v", data)
	}
	assertPulseFreshnessSummary(t, event, "blocked", "not_provided", "stale")
}

func TestPulseRunBlocksControlPlaneReadbackDigestMismatch(t *testing.T) {
	dir := t.TempDir()
	receiptPath := filepath.Join(dir, "digest-mismatch-control-plane-receipt.json")
	receiptPacketPath := filepath.ToSlash(receiptPath)
	receipt := []byte(`{"schema_version":"ao2.cp-ingest-receipt.v1","status":"stored","sha256":"` + strings.Repeat("a", 64) + `"}` + "\n")
	if err := os.WriteFile(receiptPath, receipt, 0o644); err != nil {
		t.Fatalf("write receipt: %v", err)
	}
	packetPath := filepath.Join(dir, "factory-packet.json")
	packet := `{
		"schema_version": "ao.forge.factory-packet.v0.1",
		"status": "passed",
		"objective": {"text": "test", "workspace": ".", "release_mode": false},
		"factory_plan": {"plan_id": "forge-plan-abcdef123456", "workcell_count": 1},
		"policy_decisions": [
			{"decision_id": "allow-local-plan", "target": "factory-plan", "decision": "allow", "explanation": "allowed"}
		],
		"workcells": [
			{"workcell_id": "verify", "kind": "verify", "status": "passed", "depends_on": []}
		],
		"evidence": [
			{
				"label": "control plane readback receipt",
				"schema_version": "ao2.cp-ingest-receipt.v1",
				"status": "passed",
				"path": "` + receiptPacketPath + `",
				"sha256": "` + strings.Repeat("b", 64) + `"
			}
		],
		"trust_boundary": {
			"local_first": true,
			"mutates_releases": false,
			"stores_credentials": false,
			"control_plane_approves_work": false
		},
		"next_actions": []
	}`
	if err := os.WriteFile(packetPath, []byte(packet), 0o644); err != nil {
		t.Fatalf("write packet: %v", err)
	}

	event := runBlockedPulseForEvent(t, []string{"pulse", "run", "--out", t.TempDir(), "--forge-live-packet", packetPath})
	artifact := pulseArtifact(t, event, "control_plane_readback")
	if artifact["status"] != "blocked" {
		t.Fatalf("expected blocked control-plane readback, got %#v", artifact)
	}
	data := readPulseArtifactJSON(t, artifact)
	if !strings.Contains(data["explanation"].(string), "digest mismatch") {
		t.Fatalf("unexpected control-plane readback mismatch: %#v", data)
	}
}

func TestPulseRunDiscoversControlPlaneReadbackFromForgeLivePacket(t *testing.T) {
	dir := t.TempDir()
	receiptRelPath := filepath.ToSlash(filepath.Join("internal", "cli", "testdata", "generated-control-plane-receipt.json"))
	receiptPath := repoPath(receiptRelPath)
	receipt := []byte(`{"schema_version":"ao2.cp-ingest-receipt.v1","status":"stored","sha256":"` + strings.Repeat("a", 64) + `"}` + "\n")
	if err := os.WriteFile(receiptPath, receipt, 0o644); err != nil {
		t.Fatalf("write receipt: %v", err)
	}
	defer func() {
		_ = os.Remove(receiptPath)
	}()
	receiptSum := sha256.Sum256(receipt)
	receiptSHA := fmt.Sprintf("%x", receiptSum[:])
	packetPath := filepath.Join(dir, "factory-packet.json")
	packet := `{
		"schema_version": "ao.forge.factory-packet.v0.1",
		"status": "passed",
		"objective": {"text": "test", "workspace": ".", "release_mode": false},
		"factory_plan": {"plan_id": "forge-plan-abcdef123456", "workcell_count": 1},
		"policy_decisions": [
			{"decision_id": "allow-local-plan", "target": "factory-plan", "decision": "allow", "explanation": "allowed"}
		],
		"workcells": [
			{"workcell_id": "verify", "kind": "verify", "status": "passed", "depends_on": []}
		],
		"evidence": [
			{
				"label": "control plane readback receipt",
				"schema_version": "ao2.cp-ingest-receipt.v1",
				"status": "passed",
				"path": "` + receiptRelPath + `",
				"sha256": "` + receiptSHA + `"
			}
		],
		"trust_boundary": {
			"local_first": true,
			"mutates_releases": false,
			"stores_credentials": false,
			"control_plane_approves_work": false
		},
		"next_actions": []
	}`
	if err := os.WriteFile(packetPath, []byte(packet), 0o644); err != nil {
		t.Fatalf("write packet: %v", err)
	}

	event := runPulseForEvent(t, []string{"pulse", "run", "--out", t.TempDir(), "--forge-live-packet", packetPath})
	artifact := pulseArtifact(t, event, "control_plane_readback")
	if artifact["status"] != "ready" {
		t.Fatalf("expected ready control-plane readback discovered from packet, got %#v", artifact)
	}
	data := readPulseArtifactJSON(t, artifact)
	if data["status"] != "ready" || data["receipt_schema_version"] != "ao2.cp-ingest-receipt.v1" {
		t.Fatalf("unexpected discovered control-plane readback: %#v", data)
	}
}

func TestAOSurfaceStatusRunAndAudit(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"ao", "status"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("ao status returned %d; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "AO Foundry") {
		t.Fatalf("ao status missing Foundry output: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	pulseOut := t.TempDir()
	code = Run([]string{"ao", "run", "--out", pulseOut}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("ao run returned %d; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(pulseOut, "pulse-event.json")); err != nil {
		t.Fatalf("ao run did not write pulse event: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	auditOut := filepath.Join(t.TempDir(), "competitive-readiness-audit.json")
	code = Run([]string{"ao", "audit", "--out", auditOut}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("ao audit returned %d; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(auditOut); err != nil {
		t.Fatalf("ao audit did not write audit: %v", err)
	}
}

func runPulseForEvent(t *testing.T, args []string) map[string]any {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	outDir := ""
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--out" {
			outDir = args[i+1]
			break
		}
	}
	if outDir == "" {
		t.Fatalf("test helper requires --out")
	}
	data, err := os.ReadFile(filepath.Join(outDir, "pulse-event.json"))
	if err != nil {
		t.Fatalf("read pulse event: %v", err)
	}
	var event map[string]any
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("pulse event is not JSON: %v", err)
	}
	return event
}

func runBlockedPulseForEvent(t *testing.T, args []string) map[string]any {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for blocked pulse; stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	outDir := ""
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--out" {
			outDir = args[i+1]
			break
		}
	}
	if outDir == "" {
		t.Fatalf("test helper requires --out")
	}
	data, err := os.ReadFile(filepath.Join(outDir, "pulse-event.json"))
	if err != nil {
		t.Fatalf("read blocked pulse event: %v; stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	var event map[string]any
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("blocked pulse event is not JSON: %v", err)
	}
	if event["status"] != "blocked" {
		t.Fatalf("expected blocked pulse event, got %#v", event)
	}
	return event
}

func pulseArtifact(t *testing.T, event map[string]any, name string) map[string]any {
	t.Helper()
	for _, raw := range event["artifacts"].([]any) {
		artifact := raw.(map[string]any)
		if artifact["name"] == name {
			return artifact
		}
	}
	t.Fatalf("missing pulse artifact %q in %#v", name, event["artifacts"])
	return nil
}

func pulseArtifactFromRaw(t *testing.T, event map[string]any, name string) map[string]any {
	t.Helper()
	rawArtifacts, ok := event["artifacts"].([]any)
	if !ok {
		t.Fatalf("event artifacts are not an array: %#v", event["artifacts"])
	}
	for _, raw := range rawArtifacts {
		artifact, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("event artifact is not an object: %#v", raw)
		}
		if artifact["name"] == name {
			return artifact
		}
	}
	t.Fatalf("missing pulse artifact %q in %#v", name, event["artifacts"])
	return nil
}

func readPulseArtifactJSON(t *testing.T, artifact map[string]any) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.FromSlash(artifact["path"].(string)))
	if err != nil {
		t.Fatalf("read pulse artifact: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("artifact is not JSON: %v", err)
	}
	return decoded
}

func readmeBlock(t *testing.T, text, startMarker, endMarker string) string {
	t.Helper()
	start := strings.Index(text, startMarker)
	if start < 0 {
		t.Fatalf("README missing start marker %q", startMarker)
	}
	end := strings.Index(text, endMarker)
	if end < 0 {
		t.Fatalf("README missing end marker %q", endMarker)
	}
	if end <= start {
		t.Fatalf("README marker order is invalid")
	}
	return text[start:end+len(endMarker)] + "\n"
}

func copyFileForTest(t *testing.T, from, to string) {
	t.Helper()
	data, err := os.ReadFile(from)
	if err != nil {
		t.Fatalf("read %s: %v", from, err)
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(to), err)
	}
	if err := os.WriteFile(to, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", to, err)
	}
}

func writeActiveStackGithubRunsReportForTest(t *testing.T, path string, ciOverrides map[string]string) {
	t.Helper()
	ciRuns := map[string]string{
		"ao-foundry":        "99999999991",
		"ao-forge":          "28040935640",
		"ao-command":        "28049179216",
		"ao2":               "28053626014",
		"ao2-control-plane": "28050638200",
		"ao-covenant":       "28048024016",
	}
	opsRuns := map[string]string{
		"ao-foundry":        "28027968419",
		"ao-forge":          "28056174653",
		"ao-command":        "28049279592",
		"ao2":               "28054451606",
		"ao2-control-plane": "28051422824",
		"ao-covenant":       "28055014809",
	}
	for repo, runID := range ciOverrides {
		ciRuns[repo] = runID
	}
	report := ActiveStackGithubRunsReport{
		SchemaVersion:      "ao.foundry.active-stack-github-runs-report.v0.1",
		Status:             "ready",
		Branch:             "main",
		CurrentRepo:        "ao-foundry",
		CurrentRepoSkipped: false,
		GeneratedAt:        "2026-06-23T12:00:00Z",
	}
	for _, repo := range []string{"ao-foundry", "ao-forge", "ao-command", "ao2", "ao2-control-plane", "ao-covenant"} {
		displayTitle := ""
		if repo == "ao-foundry" {
			displayTitle = "Refresh Foundry readiness evidence (#99)"
		}
		report.Repositories = append(report.Repositories, ActiveStackGithubRunsRepository{
			Repository: "uesugitorachiyo/" + repo,
			LatestCI: ActiveStackGithubRun{
				Workflow:    "ci.yml",
				Status:      "completed",
				Conclusion:  "success",
				RunID:       ciRuns[repo],
				DisplayName: displayTitle,
				URL:         "https://github.com/uesugitorachiyo/" + repo + "/actions/runs/" + ciRuns[repo],
			},
			LatestOps: ActiveStackGithubRun{
				Workflow:   "production-readiness-ops.yml",
				Status:     "completed",
				Conclusion: "success",
				RunID:      opsRuns[repo],
				URL:        "https://github.com/uesugitorachiyo/" + repo + "/actions/runs/" + opsRuns[repo],
			},
		})
	}
	mustWriteJSONForTest(t, path, report)
}

func rewriteActiveStackGithubRunsReportForTest(t *testing.T, path string, mutate func(*ActiveStackGithubRunsReport)) {
	t.Helper()
	var report ActiveStackGithubRunsReport
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}
	mutate(&report)
	mustWriteJSONForTest(t, path, report)
}

func rollupRowsContain(t *testing.T, raw any, matchKey, matchValue, assertKey, assertValue string) bool {
	t.Helper()
	rows, ok := raw.([]any)
	if !ok {
		t.Fatalf("rollup field is not an array: %#v", raw)
	}
	for _, rawRow := range rows {
		row, ok := rawRow.(map[string]any)
		if !ok {
			t.Fatalf("rollup row is not an object: %#v", rawRow)
		}
		if row[matchKey] == matchValue && row[assertKey] == assertValue {
			return true
		}
	}
	return false
}

func assertPulseFreshnessSummary(t *testing.T, event map[string]any, wantStatus, wantForgeLivePacket, wantControlPlaneReadback string) {
	t.Helper()
	summary, ok := event["freshness_summary"].(map[string]any)
	if !ok {
		t.Fatalf("pulse event missing freshness_summary: %#v", event)
	}
	if summary["schema_version"] != "ao.foundry.pulse-freshness-summary.v0.1" {
		t.Fatalf("unexpected freshness summary schema: %#v", summary)
	}
	if summary["status"] != wantStatus || summary["forge_live_packet"] != wantForgeLivePacket || summary["control_plane_readback"] != wantControlPlaneReadback {
		t.Fatalf("unexpected freshness summary: got %#v want status=%q forge_live_packet=%q control_plane_readback=%q", summary, wantStatus, wantForgeLivePacket, wantControlPlaneReadback)
	}
	if strings.TrimSpace(summary["explanation"].(string)) == "" {
		t.Fatalf("freshness summary missing explanation: %#v", summary)
	}
}

func registryFixture() string {
	return filepath.Join("..", "..", "examples", "registry", "local-ao-stack.foundry-registry.json")
}

func taskFixture() string {
	return filepath.Join("..", "..", "examples", "tasks", "ao-foundry-bootstrap.foundry-task.json")
}

func goalFixture() string {
	return filepath.Join("..", "..", "examples", "goals", "ao-foundry-production-readiness.goal-run.json")
}

func validPacketFixture() string {
	return filepath.Join("testdata", "valid-forge-packet.json")
}

func networkTaskFixture() string {
	return filepath.Join("..", "..", "examples", "tasks", "network-read-task.foundry-task.json")
}

func initTempGitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# fixture\n"), 0o644); err != nil {
		t.Fatalf("write fixture readme: %v", err)
	}
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "-c", "user.name=AO Foundry", "-c", "user.email=foundry@example.invalid", "commit", "-m", "initial")
	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
}

func writeHealthRegistry(t *testing.T, workspace, branch string, readinessFiles []string, healthOverride *HealthReaderConfig) string {
	t.Helper()
	health := HealthReaderConfig{
		RequireCleanWorktree: true,
		VerificationCommands: []string{"git status"},
		ReadinessFiles:       readinessFiles,
		RequireTags:          []string{},
		AllowNetworkRead:     false,
		GitHubActions:        false,
	}
	if healthOverride != nil {
		health.AllowNetworkRead = healthOverride.AllowNetworkRead
		health.GitHubActions = healthOverride.GitHubActions
	}
	registry := Registry{
		SchemaVersion: "ao.foundry.registry.v0.1",
		FoundryID:     "health-fixture",
		Name:          "Health Fixture",
		Repos: []Repo{
			{
				ID:                "fixture",
				Name:              "Fixture",
				Role:              "operations-factory",
				DelegatesTo:       "ao-forge",
				Workspace:         workspace,
				Branches:          []string{branch},
				EvidenceSources:   []EvidenceSource{{Kind: "readiness", Location: "README.md", Owner: "fixture"}},
				AllowedAutomation: []string{"read-health"},
				ReadinessSignals:  []ReadinessSignal{{Name: "fixture", Status: "ready", Source: "git status"}},
				Health:            health,
			},
		},
	}
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		t.Fatalf("marshal registry: %v", err)
	}
	path := filepath.Join(t.TempDir(), "registry.json")
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write registry: %v", err)
	}
	return path
}

func boardFixtureRepo(id, name, role, workspace string) Repo {
	return Repo{
		ID:                id,
		Name:              name,
		Role:              role,
		DelegatesTo:       "ao-forge",
		Workspace:         workspace,
		Branches:          []string{"main"},
		EvidenceSources:   []EvidenceSource{{Kind: "readiness", Location: "README.md", Owner: id}},
		AllowedAutomation: []string{"read-health"},
		ReadinessSignals:  []ReadinessSignal{{Name: "fixture", Status: "ready", Source: "git status"}},
		Health: HealthReaderConfig{
			RequireCleanWorktree: true,
			VerificationCommands: []string{"git status"},
			AllowNetworkRead:     false,
			GitHubActions:        false,
		},
	}
}

func writeRegistryFixture(t *testing.T, registry Registry) string {
	t.Helper()
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		t.Fatalf("marshal registry: %v", err)
	}
	path := filepath.Join(t.TempDir(), "registry.json")
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write registry: %v", err)
	}
	return path
}

func writeBudgetGoal(t *testing.T, maxSpend, spend int) string {
	t.Helper()
	var goal GoalRun
	data, err := os.ReadFile(goalFixture())
	if err != nil {
		t.Fatalf("read goal fixture: %v", err)
	}
	if err := json.Unmarshal(data, &goal); err != nil {
		t.Fatalf("unmarshal goal fixture: %v", err)
	}
	goal.LoopPolicy.MaxSpendCents = maxSpend
	goal.LoopPolicy.SpendCents = spend
	out, err := json.MarshalIndent(goal, "", "  ")
	if err != nil {
		t.Fatalf("marshal budget goal: %v", err)
	}
	path := filepath.Join(t.TempDir(), "budget.goal-run.json")
	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
		t.Fatalf("write budget goal: %v", err)
	}
	return path
}

func mustWriteJSONForTest(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal JSON: %v", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write JSON: %v", err)
	}
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
