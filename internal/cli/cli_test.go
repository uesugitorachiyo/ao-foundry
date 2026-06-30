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

func TestRegistryValidateAcceptsAtlasDemoFixture(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"registry", "validate", "--registry", atlasRegistryFixture()}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "registry valid") {
		t.Fatalf("expected validation output, got %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"status", "--registry", atlasRegistryFixture()}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("status returned %d, want 0; stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"atlas-demo-stack", "2 repos", "ao-atlas", "ready: 2"} {
		if !strings.Contains(out, want) {
			t.Fatalf("status output missing %q: %s", want, out)
		}
	}

	registry, err := loadRegistry(atlasRegistryFixture())
	if err != nil {
		t.Fatal(err)
	}
	var hasImportSource bool
	for _, repo := range registry.Repos {
		if repo.ID != "ao-atlas" {
			continue
		}
		for _, source := range repo.EvidenceSources {
			if source.Kind == "atlas-foundry-import" && source.Location == "examples/valid/foundry-import.json" {
				hasImportSource = true
			}
		}
	}
	if !hasImportSource {
		t.Fatal("atlas demo registry must identify the Atlas foundry-import packet as Foundry's first consumer artifact")
	}
}

func TestAtlasImportValidateAcceptsFixtureOnlyPacket(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"atlas", "import", "validate", "--import", atlasImportFixture()}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"atlas import valid", "source_artifacts=2", "tasks=1", "schedules_work=false", "executes_work=false", "approves_work=false"} {
		if !strings.Contains(out, want) {
			t.Fatalf("atlas import output missing %q: %s", want, out)
		}
	}
}

func TestAtlasImportValidateAcceptsMutationAuthorityMetadata(t *testing.T) {
	artifact, err := loadAtlasFoundryImport(filepath.Join("testdata", "atlas-foundry-import-mutation-metadata.json"))
	if err != nil {
		t.Fatalf("expected mutation metadata import to validate: %v", err)
	}
	if len(artifact.Tasks) != 1 {
		t.Fatalf("expected one task, got %#v", artifact.Tasks)
	}
	task := artifact.Tasks[0]
	if task.MutationClass != "docs_only_single_file" {
		t.Fatalf("expected mutation class metadata, got %#v", task)
	}
	if !stringSliceContains(task.RequiredGates, "atlas_classification") {
		t.Fatalf("expected required gate metadata, got %#v", task.RequiredGates)
	}
	if len(task.RollbackScope) == 0 || task.AuthorityBoundary == "" {
		t.Fatalf("expected rollback scope and authority boundary, got %#v", task)
	}
}

func TestAtlasImportValidateRejectsMissingMutationAuthorityMetadata(t *testing.T) {
	_, err := loadAtlasFoundryImport(filepath.Join("testdata", "atlas-foundry-import-missing-metadata.json"))
	if err == nil || !strings.Contains(err.Error(), "mutation_class") {
		t.Fatalf("expected missing mutation metadata rejection, got %v", err)
	}
}

func TestAtlasImportValidateRejectsExecutionAuthority(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"atlas", "import", "validate", "--import", filepath.Join("testdata", "atlas-foundry-import-executes-work.json")}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for executable Atlas import; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "executes_work must be false") {
		t.Fatalf("expected execution authority rejection, got %q", stderr.String())
	}
}

func TestAtlasReadbackWritesObserverReport(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "atlas-readback.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"atlas", "readback",
		"--import", atlasImportFixture(),
		"--run-link", atlasRunLinkFixture(),
		"--out", outPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "atlas_readback="+outPath) {
		t.Fatalf("expected readback output path, got %q", stdout.String())
	}
	var report map[string]any
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("report is not JSON: %v", err)
	}
	for key, want := range map[string]any{
		"schema_version":  "ao.foundry.atlas-readback.v0.1",
		"status":          "ready",
		"mode":            "fixture_only_readback",
		"task_id":         "atlas-readiness-task",
		"workgraph_id":    "atlas-readiness-workgraph",
		"target_instance": "demo-stack",
		"schedules_work":  false,
		"executes_work":   false,
		"approves_work":   false,
	} {
		if report[key] != want {
			t.Fatalf("report[%s] = %#v, want %#v; report=%#v", key, report[key], want, report)
		}
	}
	if report["task_digest"] != "sha256:1d49e1952c8db016899a8f3f054bff4ab92eca3b45c291b5f3498e734e857396" {
		t.Fatalf("report must preserve Atlas task digest: %#v", report)
	}
	evidence, ok := report["evidence"].(map[string]any)
	if !ok || evidence["foundry"] != "evidence/foundry/atlas-readiness.json" {
		t.Fatalf("report must preserve public-safe run-link evidence: %#v", report["evidence"])
	}
}

func TestAtlasReadbackRejectsIncompleteRunLink(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"atlas", "readback",
		"--import", atlasImportFixture(),
		"--run-link", filepath.Join("testdata", "atlas-run-link-blocked.json"),
		"--out", filepath.Join(t.TempDir(), "atlas-readback.json"),
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for blocked Atlas run-link; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "run-link status must be completed") {
		t.Fatalf("expected completed status rejection, got %q", stderr.String())
	}
}

func TestAtlasReadbackRejectsMissingMatchingTask(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"atlas", "readback",
		"--import", atlasImportFixture(),
		"--run-link", filepath.Join("testdata", "atlas-run-link-missing-task.json"),
		"--out", filepath.Join(t.TempDir(), "atlas-readback.json"),
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for missing Atlas task match; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "no matching Atlas import task") {
		t.Fatalf("expected missing task rejection, got %q", stderr.String())
	}
}

func TestAtlasStatusSummarizesRegistryImportAndReadback(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"atlas", "status",
		"--registry", atlasRegistryFixture(),
		"--import", atlasImportFixture(),
		"--run-link", atlasRunLinkFixture(),
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"atlas status: ready",
		"registry=atlas-demo-stack",
		"import=atlas-readiness-workgraph-foundry-import",
		"readback=ao.foundry.atlas-readback.v0.1",
		"task_id=atlas-readiness-task",
		"schedules_work=false",
		"executes_work=false",
		"approves_work=false",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("atlas status output missing %q: %s", want, out)
		}
	}
}

func TestAtlasStatusWritesObserverReport(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "atlas-status.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"atlas", "status",
		"--registry", atlasRegistryFixture(),
		"--import", atlasImportFixture(),
		"--run-link", atlasRunLinkFixture(),
		"--out", outPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "atlas_status="+outPath) {
		t.Fatalf("expected status output path, got %q", stdout.String())
	}
	var report map[string]any
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("report is not JSON: %v", err)
	}
	for key, want := range map[string]any{
		"schema_version":  "ao.foundry.atlas-status.v0.1",
		"status":          "ready",
		"mode":            "fixture_only_readback",
		"registry_id":     "atlas-demo-stack",
		"import_id":       "atlas-readiness-workgraph-foundry-import",
		"readback_status": "ready",
		"task_id":         "atlas-readiness-task",
		"schedules_work":  false,
		"executes_work":   false,
		"approves_work":   false,
	} {
		if report[key] != want {
			t.Fatalf("report[%s] = %#v, want %#v; report=%#v", key, report[key], want, report)
		}
	}
}

func TestAtlasStatusRejectsIncompleteRunLink(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"atlas", "status",
		"--registry", atlasRegistryFixture(),
		"--import", atlasImportFixture(),
		"--run-link", filepath.Join("testdata", "atlas-run-link-blocked.json"),
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for blocked Atlas run-link; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "run-link status must be completed") {
		t.Fatalf("expected completed status rejection, got %q", stderr.String())
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
	for _, want := range []string{"AO Foundry", "7 repos", "ao-foundry", "ao-atlas", "ready: 7"} {
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
	if len(repos) != 7 {
		t.Fatalf("expected 7 repo health entries, got %d", len(repos))
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
		"atlas_readback_consumer",
		"loop preflight",
		"first_failing_check",
		"blocking_next_actions",
		"maintenance_suggestions",
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
	if strings.Contains(scriptText, "--out tmp/governed-live-mutation-dry-run-chain") ||
		strings.Contains(scriptText, "--chain tmp/governed-live-mutation-dry-run-chain/summary.json") {
		t.Fatalf("active stack readiness loop should use per-run governed live-mutation evidence paths")
	}
	if strings.Contains(scriptText, `RUN_TMP_REL="tmp/`) ||
		!strings.Contains(scriptText, `EXCLUDED_DIR="excluded"`) ||
		!strings.Contains(scriptText, `RUN_TMP_REL="$EXCLUDED_DIR/active-stack-readiness-loop-`) {
		t.Fatalf("active stack readiness loop should keep internal scratch out of repo-root tmp")
	}
	if !strings.Contains(readmeText, "scripts/active-stack-readiness-loop.sh") {
		t.Fatalf("README does not document active stack readiness loop")
	}
}

func TestComplexRefactorRehearsalScriptIncludesRepairRepackAndCommandReadback(t *testing.T) {
	script, err := os.ReadFile(repoPath("scripts/complex-refactor-workgraph-rehearsal.sh"))
	if err != nil {
		t.Fatalf("read complex refactor rehearsal script: %v", err)
	}
	scriptText := string(script)
	for _, want := range []string{
		"atlas-repair-plan.json",
		"atlas-context-repack.json",
		"ao-command-complex-refactor-status.json",
		"repair_plan",
		"context_repack",
		"command_readback",
	} {
		if !strings.Contains(scriptText, want) {
			t.Fatalf("complex refactor rehearsal script missing %q", want)
		}
	}
	for _, path := range []string{
		"examples/complex-refactor-workgraph/run-link.command-readback.blocked.json",
		"examples/complex-refactor-workgraph/run-link.command-readback.needs-context.json",
	} {
		if _, err := os.Stat(repoPath(path)); err != nil {
			t.Fatalf("missing complex refactor fixture %s: %v", path, err)
		}
	}
}

func TestOvernightRehearsalRunnerScriptIsDryRunAndValidatesControlChain(t *testing.T) {
	script, err := os.ReadFile(repoPath("scripts/overnight-rehearsal-runner.sh"))
	if err != nil {
		t.Fatalf("read overnight rehearsal runner script: %v", err)
	}
	scriptText := string(script)
	for _, want := range []string{
		"ao.foundry.overnight-rehearsal-runner.v0.1",
		"complex-refactor-workgraph-rehearsal.sh",
		"pulse_gate_status",
		"lifecycle_status",
		"atlas_import_status",
		"repair_plan_status",
		"context_repack_status",
		"command_status",
		"mutates_repositories:false",
		"executes_work:false",
		"schedules_work:false",
	} {
		if !strings.Contains(scriptText, want) {
			t.Fatalf("overnight rehearsal runner script missing %q", want)
		}
	}
	for _, forbidden := range []string{"gh pr merge", "git push", "gh release", "provider", "upload-artifact"} {
		if strings.Contains(scriptText, forbidden) {
			t.Fatalf("overnight rehearsal runner script contains forbidden live action %q", forbidden)
		}
	}
}

func TestFreshOvernightRehearsalArtifactScriptPreservesCommandReadback(t *testing.T) {
	script, err := os.ReadFile(repoPath("scripts/fresh-overnight-rehearsal-artifact.sh"))
	if err != nil {
		t.Fatalf("read fresh overnight rehearsal artifact script: %v", err)
	}
	scriptText := string(script)
	for _, want := range []string{
		"ao.foundry.overnight-rehearsal-artifact.v0.1",
		"overnight-rehearsal-runner.sh",
		"fresh_output_root",
		"runner_summary",
		"command_readback",
		"source_digests",
		"mutates_repositories:false",
		"executes_work:false",
		"schedules_work:false",
		"approves_work:false",
	} {
		if !strings.Contains(scriptText, want) {
			t.Fatalf("fresh overnight rehearsal artifact script missing %q", want)
		}
	}
	for _, forbidden := range []string{"gh pr merge", "git push", "gh release", "OPENAI" + "_API_KEY", "ANTHROPIC" + "_API_KEY", "upload-artifact"} {
		if strings.Contains(scriptText, forbidden) {
			t.Fatalf("fresh overnight rehearsal artifact script contains forbidden live action %q", forbidden)
		}
	}
}

func TestGovernedLiveMutationDryRunChainScript(t *testing.T) {
	script := repoPath("scripts/governed-live-mutation-dry-run-chain.sh")
	scriptData, err := os.ReadFile(script)
	if err != nil {
		t.Fatalf("read governed live mutation chain script: %v", err)
	}
	scriptText := string(scriptData)
	for _, want := range []string{
		"ao.foundry.governed-live-mutation-dry-run-chain.v0.1",
		"Blueprint/Atlas complex task",
		"Covenant authority dry-run",
		"Forge dry-run plan",
		"AO2 dry-run packet",
		"worktree isolation",
		"rollback rehearsal",
		"Sentinel hold verdict",
		"Promoter activation boundary",
		"AO Command readback",
		"live_mutation_performed:false",
		"ungated_live_mutation_claim:false",
	} {
		if !strings.Contains(scriptText, want) {
			t.Fatalf("governed live mutation chain script missing %q", want)
		}
	}
	for _, forbidden := range []string{"git apply", "git checkout", "git switch", "git worktree add", "gh pr merge", "git " + "push", "gh " + "release", "curl "} {
		if strings.Contains(scriptText, forbidden) {
			t.Fatalf("governed live mutation chain script contains forbidden live action %q", forbidden)
		}
	}
	outDir := filepath.ToSlash(filepath.Join("tmp", strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())))
	t.Cleanup(func() { _ = os.RemoveAll(repoPath(outDir)) })
	cmd := exec.Command("bash", script, "--out", outDir)
	cmd.Dir = repoPath(".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("governed live mutation chain failed: %v\n%s", err, string(out))
	}
	summary := readObjectFixture(t, filepath.Join(outDir, "summary.json"))
	if summary["status"] != "ready" {
		t.Fatalf("expected ready governed live mutation chain: %#v", summary)
	}
	assessment := summary["readiness_assessment"].(map[string]any)
	if assessment["live_mutation_performed"] != false ||
		assessment["ungated_live_mutation_claim"] != false ||
		assessment["first_tiny_live_mutation_class_safe_to_request"] != true {
		t.Fatalf("unexpected governed live mutation assessment: %#v", assessment)
	}
	boundaries := summary["authority_boundaries"].(map[string]any)
	if boundaries["live_mutation_allowed"] != false ||
		boundaries["mutates_repositories"] != false ||
		boundaries["schedules_work"] != false ||
		boundaries["executes_work"] != false ||
		boundaries["approves_work"] != false {
		t.Fatalf("chain must remain dry-run and non-mutating: %#v", boundaries)
	}
	artifacts := summary["source_artifacts"].([]any)
	if len(artifacts) != 10 {
		t.Fatalf("expected ten source artifacts, got %#v", artifacts)
	}
}

func TestGovernedLiveMutationDryRunChainScriptLowRiskCode(t *testing.T) {
	script := repoPath("scripts/governed-live-mutation-dry-run-chain.sh")
	scriptData, err := os.ReadFile(script)
	if err != nil {
		t.Fatalf("read governed live mutation chain script: %v", err)
	}
	scriptText := string(scriptData)
	for _, want := range []string{
		"--mutation-class docs_only_multi_file|low_risk_code",
		"Atlas classification",
		"Foundry class gate",
		"Covenant authority ticket",
		"Forge dry-run plan",
		"AO2 bounded patch packet",
		"Sentinel hold verdict",
		"AO Command readback",
		"low_risk_code dry-run/readback is complete",
		"safe_to_execute=$safe_to_execute",
	} {
		if !strings.Contains(scriptText, want) {
			t.Fatalf("low-risk governed chain script missing %q", want)
		}
	}
	for _, forbidden := range []string{"git apply", "git checkout", "git switch", "git worktree add", "gh pr merge", "git " + "push", "gh " + "release", "curl "} {
		if strings.Contains(scriptText, forbidden) {
			t.Fatalf("low-risk governed chain script contains forbidden live action %q", forbidden)
		}
	}
	outDir := filepath.ToSlash(filepath.Join("tmp", strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())))
	t.Cleanup(func() { _ = os.RemoveAll(repoPath(outDir)) })
	cmd := exec.Command("bash", script, "--mutation-class", "low_risk_code", "--out", outDir)
	cmd.Dir = repoPath(".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("low-risk governed chain failed: %v\n%s", err, string(out))
	}
	summary := readObjectFixture(t, filepath.Join(outDir, "summary.json"))
	if summary["status"] != "ready" ||
		summary["mutation_class"] != "low_risk_code" ||
		summary["current_proven_live_class"] != "test_only" ||
		summary["safe_to_request"] != true ||
		summary["safe_to_execute"] != false ||
		summary["exact_next_action"] != "build_low_risk_code_live_execution_gate" {
		t.Fatalf("unexpected low-risk governed chain summary: %#v", summary)
	}
	assessment := summary["readiness_assessment"].(map[string]any)
	for _, field := range []string{"includes_atlas", "includes_covenant", "includes_forge", "includes_ao2", "includes_rollback", "includes_sentinel", "includes_promoter", "includes_command"} {
		if assessment[field] != true {
			t.Fatalf("low-risk chain assessment missing %s: %#v", field, assessment)
		}
	}
	if assessment["live_mutation_performed"] != false || assessment["ungated_live_mutation_claim"] != false {
		t.Fatalf("low-risk chain must remain dry-run: %#v", assessment)
	}
	artifacts := summary["source_artifacts"].([]any)
	if len(artifacts) != 10 {
		t.Fatalf("expected ten low-risk source artifacts, got %#v", artifacts)
	}
}

func TestLowRiskCodeLiveRehearsalGateBlocksWithoutPolicyEvidence(t *testing.T) {
	chainScript := repoPath("scripts/governed-live-mutation-dry-run-chain.sh")
	gateScript := repoPath("scripts/low-risk-code-live-rehearsal-gate.sh")
	gateScriptData, err := os.ReadFile(gateScript)
	if err != nil {
		t.Fatalf("read low-risk live rehearsal gate script: %v", err)
	}
	gateScriptText := string(gateScriptData)
	for _, want := range []string{
		"ao.foundry.low-risk-code-live-rehearsal-gate.v0.1",
		"collect_low_risk_code_live_policy_evidence",
		"live policy evidence is required",
		"safe_to_execute=$safe_to_execute",
		"max_source_files:1",
		"max_test_files:1",
	} {
		if !strings.Contains(gateScriptText, want) {
			t.Fatalf("low-risk live rehearsal gate script missing %q", want)
		}
	}
	for _, forbidden := range []string{"git apply", "git checkout", "git switch", "git worktree add", "gh pr merge", "git " + "push", "gh " + "release", "curl "} {
		if strings.Contains(gateScriptText, forbidden) {
			t.Fatalf("low-risk live rehearsal gate contains forbidden live action %q", forbidden)
		}
	}

	outDir := filepath.ToSlash(filepath.Join("tmp", strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())))
	t.Cleanup(func() { _ = os.RemoveAll(repoPath(outDir)) })
	chainCmd := exec.Command("bash", chainScript, "--mutation-class", "low_risk_code", "--out", filepath.Join(outDir, "chain"))
	chainCmd.Dir = repoPath(".")
	if out, err := chainCmd.CombinedOutput(); err != nil {
		t.Fatalf("low-risk governed chain failed: %v\n%s", err, string(out))
	}
	gatePath := filepath.Join(outDir, "gate.json")
	gateCmd := exec.Command("bash", gateScript, "--chain", filepath.Join(outDir, "chain", "summary.json"), "--out", gatePath)
	gateCmd.Dir = repoPath(".")
	if out, err := gateCmd.CombinedOutput(); err != nil {
		t.Fatalf("low-risk live rehearsal gate failed: %v\n%s", err, string(out))
	}
	gate := readObjectFixture(t, gatePath)
	if gate["status"] != "blocked" ||
		gate["mutation_class"] != "low_risk_code" ||
		gate["safe_to_request"] != true ||
		gate["safe_to_execute"] != false ||
		gate["first_failing_check"] != "live_policy_evidence" ||
		gate["exact_next_step"] != "collect_low_risk_code_live_policy_evidence" {
		t.Fatalf("unexpected low-risk live rehearsal gate: %#v", gate)
	}
	boundaries := gate["authority_boundaries"].(map[string]any)
	for _, field := range []string{"mutates_repositories", "creates_branch", "creates_worktree", "opens_pr", "merges_pr", "schedules_work", "executes_work", "approves_work", "provider_calls_allowed", "release_or_publish_allowed", "multi_repo_mutation_allowed", "complex_repo_mutation_allowed", "fully_unsupervised_complex_mutation_claimed"} {
		if boundaries[field] != false {
			t.Fatalf("low-risk live rehearsal gate boundary %s must be false: %#v", field, boundaries)
		}
	}
}

func TestLiveMutationReadinessRollupScript(t *testing.T) {
	chainScript := repoPath("scripts/governed-live-mutation-dry-run-chain.sh")
	rollupScript := repoPath("scripts/live-mutation-readiness-rollup.sh")
	rollupScriptData, err := os.ReadFile(rollupScript)
	if err != nil {
		t.Fatalf("read live mutation readiness rollup script: %v", err)
	}
	rollupScriptText := string(rollupScriptData)
	for _, want := range []string{
		"ao.foundry.live-mutation-readiness-rollup.v0.1",
		"safe_to_request",
		"safe_to_execute:false",
		"requires_operator_approval:true",
		"live_mutation_allowed:false",
		"mutates_repositories:false",
		"submit_operator_approval_request_for_first_tiny_docs_only_live_mutation_class",
	} {
		if !strings.Contains(rollupScriptText, want) {
			t.Fatalf("live mutation readiness rollup script missing %q", want)
		}
	}
	for _, forbidden := range []string{"git apply", "git checkout", "git switch", "git worktree add", "gh pr merge", "git " + "push", "gh " + "release", "curl "} {
		if strings.Contains(rollupScriptText, forbidden) {
			t.Fatalf("live mutation readiness rollup script contains forbidden live action %q", forbidden)
		}
	}
	outDir := filepath.ToSlash(filepath.Join("tmp", strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())))
	t.Cleanup(func() { _ = os.RemoveAll(repoPath(outDir)) })
	chainCmd := exec.Command("bash", chainScript, "--out", filepath.Join(outDir, "chain"))
	chainCmd.Dir = repoPath(".")
	if out, err := chainCmd.CombinedOutput(); err != nil {
		t.Fatalf("governed chain failed: %v\n%s", err, string(out))
	}
	rollupPath := filepath.Join(outDir, "rollup.json")
	rollupCmd := exec.Command("bash", rollupScript, "--chain", filepath.Join(outDir, "chain", "summary.json"), "--out", rollupPath)
	rollupCmd.Dir = repoPath(".")
	if out, err := rollupCmd.CombinedOutput(); err != nil {
		t.Fatalf("readiness rollup failed: %v\n%s", err, string(out))
	}
	rollup := readObjectFixture(t, rollupPath)
	if rollup["status"] != "ready" || fmt.Sprint(rollup["score"]) != "100" {
		t.Fatalf("unexpected live mutation readiness rollup: %#v", rollup)
	}
	tinyClass := rollup["first_tiny_live_mutation_class"].(map[string]any)
	if tinyClass["safe_to_request"] != true ||
		tinyClass["safe_to_execute"] != false ||
		tinyClass["requires_operator_approval"] != true {
		t.Fatalf("unexpected tiny live mutation class assessment: %#v", tinyClass)
	}
	boundaries := rollup["authority_boundaries"].(map[string]any)
	if boundaries["live_mutation_allowed"] != false ||
		boundaries["mutates_repositories"] != false ||
		boundaries["executes_work"] != false {
		t.Fatalf("readiness rollup must remain non-mutating: %#v", boundaries)
	}
}

func TestOvernightRehearsalRunbookDocumentsDryRunOperatorSequence(t *testing.T) {
	runbook, err := os.ReadFile(repoPath("docs/operations/OVERNIGHT-REFRACTOR-REHEARSAL-RUNBOOK.md"))
	if err != nil {
		t.Fatalf("read overnight rehearsal runbook: %v", err)
	}
	text := string(runbook)
	for _, want := range []string{
		"scripts/fresh-overnight-rehearsal-artifact.sh",
		"scripts/overnight-rehearsal-runner.sh",
		"scripts/complex-refactor-workgraph-rehearsal.sh",
		"ao-command complex-refactor status",
		"ao.foundry.overnight-rehearsal-artifact.v0.1",
		"operator_mode=read_only",
		"mutates_repositories=false",
		"does not schedule, execute, approve, publish, upload, call providers, or mutate repositories",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("overnight rehearsal runbook missing %q", want)
		}
	}
}

func TestAtlasStressReadinessScriptConsumesLargeWorkgraph(t *testing.T) {
	script, err := os.ReadFile(repoPath("scripts/atlas-stress-readiness.sh"))
	if err != nil {
		t.Fatalf("read Atlas stress readiness script: %v", err)
	}
	scriptText := string(script)
	for _, want := range []string{
		"ao.foundry.atlas-stress-readiness.v0.1",
		"workgraph-large-stress.json",
		"foundry import",
		"foundry atlas import validate",
		"ready_tasks",
		"blocked_tasks",
		"imported_tasks",
		"schedules_work:false",
		"executes_work:false",
		"approves_work:false",
	} {
		if !strings.Contains(scriptText, want) {
			t.Fatalf("Atlas stress readiness script missing %q", want)
		}
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
		"test (macos-26)",
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

func TestCIWorkflowPinsMacOSRunner(t *testing.T) {
	data, err := os.ReadFile(repoPath(".github/workflows/ci.yml"))
	if err != nil {
		t.Fatalf("read CI workflow: %v", err)
	}
	workflow := string(data)
	for _, want := range []string{
		"os: [ubuntu-latest, macos-26, windows-latest]",
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("CI workflow missing pinned macOS runner detail %q", want)
		}
	}
	if strings.Contains(workflow, "macos-latest") {
		t.Fatalf("CI workflow still uses moving macOS runner label")
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
		"ao-command-rsi-health": {
			"ao-command rsi health",
			"rsi_mode=governed_fixture_local",
			"mutates_repositories=false",
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
		"ao-command-rsi-health",
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
        "run_id": "28321477720",
        "url": "https://github.com/uesugitorachiyo/ao-forge/actions/runs/28321477720"
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
        "run_id": "28246591616"
      },
      "latest_ops": {
        "workflow": "production-readiness-ops.yml",
        "status": "completed",
        "conclusion": "success",
        "run_id": "28321477720"
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
		"| ao-forge | ci.yml | 28246591616 | already_recorded |",
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
        "run_id": "28246591616"
      },
      "latest_ops": {
        "workflow": "production-readiness-ops.yml",
        "status": "completed",
        "conclusion": "success",
        "run_id": "28321477720"
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
		"| ao-forge | ci.yml | 28246591616 | already_recorded |",
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
        "run_id": "28321477720"
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
        "run_id": "28246591616"
      },
      "latest_ops": {
        "workflow": "production-readiness-ops.yml",
        "status": "completed",
        "conclusion": "success",
        "run_id": "28321477720"
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

func TestRSIImprovementGatePassesFivePercentImprovement(t *testing.T) {
	dir := t.TempDir()
	baseline := writeEvalResultFixture(t, dir, "baseline.eval-result.json", 90, 100, "ready")
	candidate := writeEvalResultFixture(t, dir, "candidate.eval-result.json", 96, 100, "ready")
	outPath := filepath.Join(dir, "rsi-improvement-gate.json")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"rsi", "improvement-gate", "--baseline", baseline, "--candidate", candidate, "--min-improvement", "5", "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "rsi improvement: passed delta=6.00 required=5.00") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}

	var gate RSIImprovementGate
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read gate: %v", err)
	}
	if err := json.Unmarshal(data, &gate); err != nil {
		t.Fatalf("gate is not JSON: %v", err)
	}
	if gate.SchemaVersion != "ao.foundry.rsi-improvement-gate.v0.1" ||
		gate.Status != "passed" ||
		gate.RequiredImprovementPercent != 5 ||
		gate.ActualImprovementPercent != 6 ||
		gate.AutonomousClaim != "measured_local_improvement" ||
		gate.MutatesRepositories {
		t.Fatalf("unexpected gate: %+v", gate)
	}
	if len(gate.Evidence) != 2 || gate.Evidence[0].SHA256 == "" || gate.Evidence[1].SHA256 == "" {
		t.Fatalf("gate missing evidence hashes: %+v", gate.Evidence)
	}
}

func TestRSIImprovementGateBlocksBelowFivePercentImprovement(t *testing.T) {
	dir := t.TempDir()
	baseline := writeEvalResultFixture(t, dir, "baseline.eval-result.json", 90, 100, "ready")
	candidate := writeEvalResultFixture(t, dir, "candidate.eval-result.json", 94, 100, "ready")
	outPath := filepath.Join(dir, "rsi-improvement-gate.json")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"rsi", "improvement-gate", "--baseline", baseline, "--candidate", candidate, "--min-improvement", "5", "--out", outPath}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run returned %d, want 1; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "improvement below required threshold") {
		t.Fatalf("expected threshold blocker, got %q", stderr.String())
	}

	var gate RSIImprovementGate
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read gate: %v", err)
	}
	if err := json.Unmarshal(data, &gate); err != nil {
		t.Fatalf("gate is not JSON: %v", err)
	}
	if gate.Status != "blocked" || gate.ActualImprovementPercent != 4 || len(gate.NextActions) == 0 {
		t.Fatalf("unexpected blocked gate: %+v", gate)
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
		"jq empty docs/contracts/*.json examples/**/*.json docs/evidence/pulse/**/*.json",
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
		"Go 1.26",
		"`../ao2`",
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
		"docs/evidence/pulse/20260623T213426Z-signed-smoke-release-gate",
		"--signed-smoke-summary docs/evidence/pulse/20260623T213426Z-signed-smoke-release-gate/signed-smoke-summary.json",
		"--promotion-out tmp/release-promotion.final.json",
		"--notes-out docs/operations/ACTIVE-SPINE-2026-06-23-RELEASE-CANDIDATE.md",
		"--manifest-out tmp/release-manifest.final.json",
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

func TestDurableSignedSmokeReleaseEvidenceIsPublicSafe(t *testing.T) {
	dir := repoPath("docs/evidence/pulse/20260623T213426Z-signed-smoke-release-gate")
	readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatalf("read durable signed-smoke evidence README: %v", err)
	}
	readmeText := string(readme)
	for _, want := range []string{
		"run_id=28065612579",
		"head_sha=f03e80c269b94f8d7e34baf50021928ad2bad098",
		"signed_smoke_job_id=83089128722",
		"artifact=signed-smoke-release-evidence",
		"pulse_id=pulse-bf475cb4e3a8",
		"release_safe=true",
	} {
		if !strings.Contains(readmeText, want) {
			t.Fatalf("durable signed-smoke evidence README missing %q", want)
		}
	}
	summary := readObjectFixture(t, filepath.Join(dir, "signed-smoke-summary.json"))
	if summary["schema_version"] != "ao.foundry.signed-smoke-summary.v0.1" ||
		summary["status"] != "ready" ||
		summary["pulse_id"] != "pulse-bf475cb4e3a8" ||
		summary["release_safe"] != true {
		t.Fatalf("unexpected durable signed-smoke summary: %#v", summary)
	}
	promotion := readObjectFixture(t, filepath.Join(dir, "release-promotion.live.json"))
	if promotion["schema_version"] != "ao.foundry.release-promotion.v0.1" ||
		promotion["status"] != "ready" ||
		promotion["release_safe"] != true ||
		promotion["signed_smoke_pulse_id"] != "pulse-bf475cb4e3a8" {
		t.Fatalf("unexpected durable release promotion evidence: %#v", promotion)
	}
	for _, path := range []string{
		filepath.Join(dir, "README.md"),
		filepath.Join(dir, "signed-smoke-summary.json"),
		filepath.Join(dir, "release-promotion.live.json"),
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read durable signed-smoke evidence %s: %v", path, err)
		}
		text := string(data)
		for _, unsafe := range []string{"/" + "Users/", "ghp" + "_", "github" + "_pat_", "AO2_CP_API_TOKEN", "local-signed-smoke-token", "api" + "_key", "access" + "_token", "BEGIN " + "RSA", "BEGIN " + "OPENSSH"} {
			if strings.Contains(text, unsafe) {
				t.Fatalf("durable signed-smoke evidence %s contains unsafe content %q", path, unsafe)
			}
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
		"git clone --depth 1 https://github.com/uesugitorachiyo/ao2.git ../ao2",
		"git clone --depth 1 https://github.com/uesugitorachiyo/ao2-control-plane.git ../ao2-control-plane",
		"cargo build -p ao2-cli",
		"cargo build -p ao2-cp-server",
		"go run ./cmd/foundry pulse signed-smoke-script --out tmp/signed-smoke.sh",
		"bash tmp/signed-smoke.sh",
		"Upload signed-smoke release evidence",
		"actions/upload-artifact@v7.0.1",
		"signed-smoke-release-evidence",
		"test -x ../ao2/target/debug/ao2",
		"tmp/pulse-live/signed-smoke-summary.json",
		"tmp/release-promotion.live.json",
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("CI workflow missing manual signed-smoke detail %q", want)
		}
	}
	signedSmokeStart := strings.Index(workflow, "\n  signed-smoke:")
	if signedSmokeStart < 0 {
		t.Fatalf("CI workflow missing signed-smoke job")
	}
	if !strings.Contains(workflow[:signedSmokeStart], "go-version-file: go.mod") {
		t.Fatalf("CI workflow regular test job should use Foundry go.mod")
	}
	if !strings.Contains(workflow[signedSmokeStart:], "go-version: \"1.26\"") {
		t.Fatalf("CI workflow signed-smoke job should use Go 1.26 for sibling AO tools")
	}
}

func TestWorkflowsUseCurrentUploadArtifactAction(t *testing.T) {
	for _, path := range []string{
		".github/workflows/ci.yml",
		".github/workflows/production-readiness-ops.yml",
	} {
		data, err := os.ReadFile(repoPath(path))
		if err != nil {
			t.Fatalf("read workflow %s: %v", path, err)
		}
		workflow := string(data)
		if !strings.Contains(workflow, "actions/upload-artifact@v7.0.1") {
			t.Fatalf("workflow %s does not use current upload-artifact action", path)
		}
		for _, stale := range []string{
			"actions/upload-artifact@v4",
			"actions/upload-artifact@v5",
			"actions/upload-artifact@v6",
		} {
			if strings.Contains(workflow, stale) {
				t.Fatalf("workflow %s contains stale artifact action %q", path, stale)
			}
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

func TestPulseDocsDeclareRSIClaimBoundary(t *testing.T) {
	checks := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "README",
			path: "README.md",
			want: []string{
				"claim_level=bounded_governed_rsi decision=allowed",
				"claim_level=full_autonomous_self_mutating_rsi decision=denied",
				"5 percentage points",
				"mutates_repositories=false",
			},
		},
		{
			name: "pulse SDD",
			path: "docs/sdd/AO-FOUNDRY-PULSE-GOLDEN-LOOP-SDD.md",
			want: []string{
				"bounded_governed_rsi",
				"full_autonomous_self_mutating_rsi",
				"not a claim of full autonomous self-mutating RSI",
				"mutation authority, rollback, and live self-change evidence",
			},
		},
		{
			name: "AO2 pulse event loop docs",
			path: "docs/operations/AO2-PULSE-EVENT-LOOP.md",
			want: []string{
				"bounded_governed_rsi",
				"full_autonomous_self_mutating_rsi",
				"read-only evidence loop",
				"AO Command RSI health",
			},
		},
	}
	for _, check := range checks {
		data, err := os.ReadFile(repoPath(check.path))
		if err != nil {
			t.Fatalf("read %s: %v", check.path, err)
		}
		doc := string(data)
		for _, want := range check.want {
			if !strings.Contains(doc, want) {
				t.Fatalf("%s missing RSI claim-boundary detail %q", check.name, want)
			}
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
		NextAction:    "Stop autonomous readiness loop; live execution requires operator intent.",
	}
	mustWriteJSONForTest(t, pulsePath, pulse)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"pulse", "derive-next", "--pulse", pulsePath, "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "next_task_id=readiness-exit-gate-satisfied") {
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
		decision.EventLoop.NextTaskID != "readiness-exit-gate-satisfied" {
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
		"docs/contracts/foundry-pulse-intake-preflight-v0.1.schema.json",
		"docs/contracts/foundry-pulse-pr-lifecycle-v0.1.schema.json",
		"examples/contract-fixtures/valid/foundry-ao2-loop-decision-v0.1.json",
		"examples/contract-fixtures/invalid/foundry-ao2-loop-decision-v0.1.json",
		"examples/contract-fixtures/valid/foundry-pulse-intake-preflight-v0.1.json",
		"examples/contract-fixtures/invalid/foundry-pulse-intake-preflight-v0.1.json",
		"examples/contract-fixtures/valid/foundry-pulse-pr-lifecycle-v0.1.json",
		"examples/contract-fixtures/invalid/foundry-pulse-pr-lifecycle-v0.1.json",
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
	stale := strings.ReplaceAll(string(data), "main CI run 28280708823", "main CI run 28016224096")
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
		"release candidate active-stack parity: ao2-control-plane missing active-stack evidence \"main CI run 28280708823\"",
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
	if event["next_action"] != "stop autonomous readiness loop; live execution requires operator intent" {
		t.Fatalf("expected stop-oriented next_action, got %#v", event["next_action"])
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
		"rsi_candidate":               false,
		"eval_result":                 false,
		"rsi_improvement_gate":        false,
		"rsi_next_improvement_task":   false,
		"demo_status":                 false,
		"release_manifest":            false,
		"competitive_readiness_audit": false,
		"pulse_trace":                 false,
		"trace_inspect":               false,
	}
	artifactPathsByName := map[string]string{}
	for _, raw := range artifacts {
		artifact := raw.(map[string]any)
		name := artifact["name"].(string)
		artifactPathsByName[name] = artifact["path"].(string)
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
		if name == "rsi_candidate" {
			var candidate RSICandidate
			data, err := os.ReadFile(filepath.FromSlash(path))
			if err != nil {
				t.Fatalf("read RSI candidate: %v", err)
			}
			if err := json.Unmarshal(data, &candidate); err != nil {
				t.Fatalf("RSI candidate is not JSON: %v", err)
			}
			if candidate.SchemaVersion != "ao.foundry.rsi-candidate.v0.1" ||
				candidate.Status != "ready" ||
				candidate.GeneratedBy != "foundry pulse run" ||
				candidate.MutatesRepositories ||
				candidate.CandidateEvalResult.Path != filepath.ToSlash(filepath.Join(outDir, "eval-result.json")) ||
				candidate.BaselineEvalResult.Path == "" ||
				candidate.CandidateEvalResult.SHA256 == "" ||
				len(candidate.ImprovementHypothesis) == 0 {
				t.Fatalf("unexpected RSI candidate: %+v", candidate)
			}
		}
		if name == "rsi_improvement_gate" {
			var gate RSIImprovementGate
			data, err := os.ReadFile(filepath.FromSlash(path))
			if err != nil {
				t.Fatalf("read RSI improvement gate: %v", err)
			}
			if err := json.Unmarshal(data, &gate); err != nil {
				t.Fatalf("RSI improvement gate is not JSON: %v", err)
			}
			if gate.Status != "passed" ||
				gate.RequiredImprovementPercent != 5 ||
				gate.ActualImprovementPercent < 5 ||
				gate.AutonomousClaim != "measured_local_improvement" ||
				gate.MutatesRepositories {
				t.Fatalf("unexpected RSI improvement gate: %+v", gate)
			}
			if len(gate.Evidence) != 2 ||
				gate.Evidence[1].Label != "candidate" ||
				gate.Evidence[1].Path != filepath.ToSlash(filepath.Join(outDir, "eval-result.json")) {
				t.Fatalf("RSI gate is not bound to generated eval result: %+v", gate.Evidence)
			}
		}
		if name == "rsi_next_improvement_task" {
			var nextTask struct {
				SchemaVersion              string   `json:"schema_version"`
				Status                     string   `json:"status"`
				GeneratedBy                string   `json:"generated_by"`
				GoalID                     string   `json:"goal_id"`
				RecommendedTaskID          string   `json:"recommended_task_id"`
				RecommendedAction          string   `json:"recommended_action"`
				CandidateEvidencePath      string   `json:"candidate_evidence_path"`
				GateEvidencePath           string   `json:"gate_evidence_path"`
				RequiredImprovementPercent float64  `json:"required_improvement_percent"`
				ActualImprovementPercent   float64  `json:"actual_improvement_percent"`
				MutatesRepositories        bool     `json:"mutates_repositories"`
				NextActions                []string `json:"next_actions"`
			}
			data, err := os.ReadFile(filepath.FromSlash(path))
			if err != nil {
				t.Fatalf("read RSI next improvement task: %v", err)
			}
			if err := json.Unmarshal(data, &nextTask); err != nil {
				t.Fatalf("RSI next improvement task is not JSON: %v", err)
			}
			if nextTask.SchemaVersion != "ao.foundry.rsi-next-improvement-task.v0.1" ||
				nextTask.Status != "ready" ||
				nextTask.GeneratedBy != "foundry pulse run" ||
				nextTask.GoalID == "" ||
				nextTask.RecommendedTaskID == "" ||
				nextTask.RecommendedAction == "" ||
				nextTask.CandidateEvidencePath != artifactPathsByName["rsi_candidate"] ||
				nextTask.GateEvidencePath != artifactPathsByName["rsi_improvement_gate"] ||
				nextTask.RequiredImprovementPercent != 5 ||
				nextTask.ActualImprovementPercent < 5 ||
				nextTask.MutatesRepositories ||
				len(nextTask.NextActions) == 0 {
				t.Fatalf("unexpected RSI next improvement task: %+v", nextTask)
			}
		}
	}
	for name, found := range requiredArtifacts {
		if !found {
			t.Fatalf("pulse event missing artifact %q: %#v", name, artifacts)
		}
	}
	if artifactPathsByName["rsi_candidate"] == "" || artifactPathsByName["eval_result"] == "" || artifactPathsByName["rsi_improvement_gate"] == "" || artifactPathsByName["rsi_next_improvement_task"] == "" {
		t.Fatalf("pulse event missing RSI artifact chain: %#v", artifactPathsByName)
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

func TestPulseIntakePreflightReadyRequiresBlueprintAndAtlasEvidence(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "pulse-intake-preflight.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"pulse", "intake-preflight",
		"--blueprint-authorization", "examples/pulse-intake/blueprint-authorization.ready.json",
		"--atlas-import", "examples/atlas/foundry-import.json",
		"--atlas-status", "examples/contract-fixtures/valid/foundry-atlas-status-v0.1.json",
		"--requires-atlas",
		"--out", outPath,
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	result := readObjectFixture(t, outPath)
	if result["schema_version"] != "ao.foundry.pulse-intake-preflight.v0.1" ||
		result["status"] != "ready" ||
		result["blueprint_status"] != "ready" ||
		result["atlas_status"] != "ready" ||
		result["first_failing_check"] != "" {
		t.Fatalf("unexpected ready preflight result: %#v", result)
	}
	if !strings.Contains(stdout.String(), `"status": "ready"`) {
		t.Fatalf("expected JSON stdout for ready preflight, got %s", stdout.String())
	}
}

func TestPulseIntakePreflightBlocksForBlueprintClarification(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "pulse-intake-preflight.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"pulse", "intake-preflight",
		"--blueprint-request", "examples/pulse-intake/blueprint-request.blocked.json",
		"--out", outPath,
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run returned %d, want blocked exit 1; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	result := readObjectFixture(t, outPath)
	if result["status"] != "blocked" || result["blueprint_status"] != "blocked" {
		t.Fatalf("unexpected blocked preflight result: %#v", result)
	}
	if result["first_failing_check"] != "blueprint_build_authorization" {
		t.Fatalf("blocked preflight should identify Blueprint authorization check, got %#v", result)
	}
}

func TestPulseIntakePreflightFailsClosedForMissingBlueprintAuthorization(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"pulse", "intake-preflight",
		"--atlas-import", "examples/atlas/foundry-import.json",
		"--atlas-status", "examples/contract-fixtures/valid/foundry-atlas-status-v0.1.json",
		"--requires-atlas",
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for missing Blueprint authorization; stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "Blueprint authorization is required") {
		t.Fatalf("expected missing Blueprint error, got %q", stderr.String())
	}
}

func TestPulseIntakePreflightFailsClosedForBlockedBlueprintMarkedReady(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"pulse", "intake-preflight",
		"--blueprint-authorization", "examples/pulse-intake/blueprint-authorization.blocked.json",
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for blocked Blueprint authorization; stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "Blueprint authorization is blocked") {
		t.Fatalf("expected blocked Blueprint authorization error, got %q", stderr.String())
	}
}

func TestPulseIntakePreflightFailsClosedForMissingAtlasEvidence(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"pulse", "intake-preflight",
		"--blueprint-authorization", "examples/pulse-intake/blueprint-authorization.ready.json",
		"--requires-atlas",
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for missing Atlas evidence; stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "Atlas handoff/readback is required") {
		t.Fatalf("expected missing Atlas error, got %q", stderr.String())
	}
}

func TestPulseIntakePreflightFailsClosedForAtlasAuthorityClaim(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"pulse", "intake-preflight",
		"--blueprint-authorization", "examples/pulse-intake/blueprint-authorization.ready.json",
		"--atlas-import", "internal/cli/testdata/atlas-foundry-import-executes-work.json",
		"--atlas-status", "examples/contract-fixtures/valid/foundry-atlas-status-v0.1.json",
		"--requires-atlas",
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for Atlas authority claim; stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "Atlas artifact claims forbidden authority") {
		t.Fatalf("expected Atlas authority error, got %q", stderr.String())
	}
}

func TestPulseIntakePreflightContractFixtureValidates(t *testing.T) {
	schema, err := readArbitraryJSON("docs/contracts/foundry-pulse-intake-preflight-v0.1.schema.json")
	if err != nil {
		t.Fatalf("read pulse intake preflight schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("pulse intake preflight schema is not an object: %#v", schema)
	}
	validFixture, err := readArbitraryJSON("examples/contract-fixtures/valid/foundry-pulse-intake-preflight-v0.1.json")
	if err != nil {
		t.Fatalf("read valid pulse intake preflight fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, validFixture, "$"); err != nil {
		t.Fatalf("valid pulse intake preflight fixture failed schema: %v", err)
	}
	invalidFixture, err := readArbitraryJSON("examples/contract-fixtures/invalid/foundry-pulse-intake-preflight-v0.1.json")
	if err != nil {
		t.Fatalf("read invalid pulse intake preflight fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, invalidFixture, "$"); err == nil {
		t.Fatalf("invalid pulse intake preflight fixture unexpectedly passed schema")
	}
}

func TestPulseLifecycleInspectAllowsReadyState(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"pulse", "lifecycle", "inspect",
		"--state", "examples/pulse-lifecycle/ready-to-start-next-slice.json",
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("lifecycle JSON output invalid: %v\n%s", err, stdout.String())
	}
	if result["allowed_next_action"] != "start_next_slice" || result["blocker_reason"] != "" {
		t.Fatalf("unexpected ready lifecycle result: %#v", result)
	}
}

func TestPulseLifecycleInspectBlocksOpenPRPendingChecks(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"pulse", "lifecycle", "inspect",
		"--state", "examples/pulse-lifecycle/open-pr-pending-checks.json",
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for pending PR checks; stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "current slice PR checks are pending") {
		t.Fatalf("expected pending check blocker, got %q", stderr.String())
	}
}

func TestPulseLifecycleInspectBlocksFailedChecks(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"pulse", "lifecycle", "inspect",
		"--state", "examples/pulse-lifecycle/checks-failed.json",
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for failed PR checks; stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "current slice PR checks are failing") {
		t.Fatalf("expected failed check blocker, got %q", stderr.String())
	}
}

func TestPulseLifecycleInspectBlocksCleanupIncomplete(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"pulse", "lifecycle", "inspect",
		"--state", "examples/pulse-lifecycle/merged-cleanup-incomplete.json",
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for incomplete cleanup; stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "merged PR cleanup is incomplete") {
		t.Fatalf("expected cleanup blocker, got %q", stderr.String())
	}
}

func TestPulseLifecycleInspectBlocksUnsyncedMainAndDirtyWorktree(t *testing.T) {
	for _, fixture := range []string{
		"examples/pulse-lifecycle/local-main-not-synced.json",
		"examples/pulse-lifecycle/dirty-worktree-blocked.json",
		"examples/pulse-lifecycle/multiple-active-branches.json",
	} {
		var stdout, stderr bytes.Buffer
		code := Run([]string{"pulse", "lifecycle", "inspect", "--state", fixture}, &stdout, &stderr)
		if code == 0 {
			t.Fatalf("Run returned success for blocked lifecycle fixture %s; stdout=%s stderr=%s", fixture, stdout.String(), stderr.String())
		}
		if !strings.Contains(stderr.String(), "pulse lifecycle") {
			t.Fatalf("expected lifecycle blocker for %s, got %q", fixture, stderr.String())
		}
	}
}

func TestPulseLifecycleContractFixtureValidates(t *testing.T) {
	schema, err := readArbitraryJSON("docs/contracts/foundry-pulse-pr-lifecycle-v0.1.schema.json")
	if err != nil {
		t.Fatalf("read pulse lifecycle schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("pulse lifecycle schema is not an object: %#v", schema)
	}
	validFixture, err := readArbitraryJSON("examples/contract-fixtures/valid/foundry-pulse-pr-lifecycle-v0.1.json")
	if err != nil {
		t.Fatalf("read valid pulse lifecycle fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, validFixture, "$"); err != nil {
		t.Fatalf("valid pulse lifecycle fixture failed schema: %v", err)
	}
	invalidFixture, err := readArbitraryJSON("examples/contract-fixtures/invalid/foundry-pulse-pr-lifecycle-v0.1.json")
	if err != nil {
		t.Fatalf("read invalid pulse lifecycle fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, invalidFixture, "$"); err == nil {
		t.Fatalf("invalid pulse lifecycle fixture unexpectedly passed schema")
	}
}

func TestPulseOvernightStartGateAllowsReadyState(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "pulse-overnight-start-gate.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"pulse", "overnight-start-gate",
		"--intake-preflight", "examples/pulse-overnight-start-gate/ready.intake-preflight.json",
		"--lifecycle", "examples/pulse-lifecycle/ready-to-start-next-slice.json",
		"--out", outPath,
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	result := readObjectFixture(t, outPath)
	if result["schema_version"] != "ao.foundry.pulse-overnight-start-gate.v0.1" ||
		result["status"] != "ready" ||
		result["allowed_next_action"] != "start_next_slice" ||
		result["first_failing_check"] != "" {
		t.Fatalf("unexpected ready start gate result: %#v", result)
	}
	if !strings.Contains(stdout.String(), `"status": "ready"`) {
		t.Fatalf("expected JSON stdout for ready gate, got %s", stdout.String())
	}
}

func TestPulseOvernightStartGateBlocksForBlueprintClarification(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "pulse-overnight-start-gate.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"pulse", "overnight-start-gate",
		"--intake-preflight", "examples/pulse-overnight-start-gate/blocked.intake-preflight.json",
		"--lifecycle", "examples/pulse-lifecycle/ready-to-start-next-slice.json",
		"--out", outPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want clean blocked exit 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	result := readObjectFixture(t, outPath)
	if result["status"] != "blocked" || result["allowed_next_action"] != "request_blueprint_clarification" {
		t.Fatalf("unexpected blocked start gate result: %#v", result)
	}
}

func TestPulseOvernightStartGateFailsBlockedPreflightWhenStartingImplementation(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "pulse-overnight-start-gate.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"pulse", "overnight-start-gate",
		"--intake-preflight", "examples/pulse-overnight-start-gate/blocked.intake-preflight.json",
		"--lifecycle", "examples/pulse-lifecycle/ready-to-start-next-slice.json",
		"--out", outPath,
		"--start-implementation",
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for blocked start attempt; stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	result := readObjectFixture(t, outPath)
	if result["first_failing_check"] != "blueprint_blocked_start_attempt" {
		t.Fatalf("expected blocked start attempt failure, got %#v", result)
	}
}

func TestPulseOvernightStartGateFailsClosedForFailedPreflight(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "pulse-overnight-start-gate.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"pulse", "overnight-start-gate",
		"--intake-preflight", "examples/pulse-overnight-start-gate/failed.intake-preflight.json",
		"--lifecycle", "examples/pulse-lifecycle/ready-to-start-next-slice.json",
		"--out", outPath,
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for failed preflight; stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	result := readObjectFixture(t, outPath)
	if result["status"] != "failed" || result["first_failing_check"] != "intake_preflight" {
		t.Fatalf("unexpected failed preflight gate result: %#v", result)
	}
}

func TestPulseOvernightStartGateFailsClosedForLifecycleBlockers(t *testing.T) {
	for _, tc := range []struct {
		name    string
		fixture string
		want    string
	}{
		{name: "pending_pr", fixture: "examples/pulse-lifecycle/open-pr-pending-checks.json", want: "current slice PR checks are pending"},
		{name: "cleanup_incomplete", fixture: "examples/pulse-lifecycle/merged-cleanup-incomplete.json", want: "merged PR cleanup is incomplete"},
		{name: "dirty_worktree", fixture: "examples/pulse-lifecycle/dirty-worktree-blocked.json", want: "dirty worktree"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			outPath := filepath.Join(t.TempDir(), "pulse-overnight-start-gate.json")
			var stdout, stderr bytes.Buffer
			code := Run([]string{
				"pulse", "overnight-start-gate",
				"--intake-preflight", "examples/pulse-overnight-start-gate/ready.intake-preflight.json",
				"--lifecycle", tc.fixture,
				"--out", outPath,
			}, &stdout, &stderr)
			if code == 0 {
				t.Fatalf("Run returned success for lifecycle blocker %s; stdout=%s stderr=%s", tc.fixture, stdout.String(), stderr.String())
			}
			if !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("expected lifecycle blocker %q, got %q", tc.want, stderr.String())
			}
			result := readObjectFixture(t, outPath)
			if result["first_failing_check"] != "pulse_pr_lifecycle" {
				t.Fatalf("expected lifecycle failure, got %#v", result)
			}
		})
	}
}

func TestPulseOvernightStartGateFailsClosedForStaleDigest(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "pulse-overnight-start-gate.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"pulse", "overnight-start-gate",
		"--intake-preflight", "examples/pulse-overnight-start-gate/stale-digest.intake-preflight.json",
		"--lifecycle", "examples/pulse-lifecycle/ready-to-start-next-slice.json",
		"--out", outPath,
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for stale digest; stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "digest is stale") {
		t.Fatalf("expected stale digest failure, got %q", stderr.String())
	}
	result := readObjectFixture(t, outPath)
	if result["first_failing_check"] != "evidence_digest" {
		t.Fatalf("expected evidence digest failure, got %#v", result)
	}
}

func TestPulseOvernightStartGateContractFixtureValidates(t *testing.T) {
	schema, err := readArbitraryJSON("docs/contracts/foundry-pulse-overnight-start-gate-v0.1.schema.json")
	if err != nil {
		t.Fatalf("read pulse overnight start gate schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("pulse overnight start gate schema is not an object: %#v", schema)
	}
	validFixture, err := readArbitraryJSON("examples/contract-fixtures/valid/foundry-pulse-overnight-start-gate-v0.1.json")
	if err != nil {
		t.Fatalf("read valid pulse overnight start gate fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, validFixture, "$"); err != nil {
		t.Fatalf("valid pulse overnight start gate fixture failed schema: %v", err)
	}
	invalidFixture, err := readArbitraryJSON("examples/contract-fixtures/invalid/foundry-pulse-overnight-start-gate-v0.1.json")
	if err != nil {
		t.Fatalf("read invalid pulse overnight start gate fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, invalidFixture, "$"); err == nil {
		t.Fatalf("invalid pulse overnight start gate fixture unexpectedly passed schema")
	}
}

func TestPulseEventLoopPolicyAllowsNextSliceWhenApprovedAndGatesPass(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "pulse-event-loop-policy.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"pulse", "event-loop-policy",
		"--class-gate", "examples/class-gate/gate.dry-run.low-risk-code.json",
		"--ci", "examples/pulse-event-loop-policy/ci.passed.json",
		"--repo-state", "examples/pulse-event-loop-policy/repo.clean.json",
		"--evidence-freshness", "examples/pulse-event-loop-policy/evidence.fresh.json",
		"--sentinel", "examples/pulse-event-loop-policy/sentinel.no-hold.json",
		"--promoter", "examples/pulse-event-loop-policy/promoter.ready.json",
		"--branch-cleanup", "examples/pulse-event-loop-policy/branch-cleanup.passed.json",
		"--scope", "examples/pulse-event-loop-policy/scope.passed.json",
		"--out", outPath,
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	result := readObjectFixture(t, outPath)
	if result["schema_version"] != "ao.foundry.pulse-event-loop-policy.v0.1" ||
		result["status"] != "ready" ||
		result["mutation_class"] != "low_risk_code" ||
		result["allowed_next_action"] != "continue_next_slice" ||
		result["first_failing_check"] != "" {
		t.Fatalf("unexpected event-loop policy result: %#v", result)
	}
	if result["safe_to_continue"] != true || result["safe_to_request"] != true || result["safe_to_execute"] != false {
		t.Fatalf("policy did not preserve request/execute boundary: %#v", result)
	}
	for _, field := range []string{"operator_prompt_required", "schedules_work", "executes_work", "approves_work", "mutates_repositories", "calls_providers", "opens_pr", "merges_pr"} {
		if result[field] != false {
			t.Fatalf("event-loop policy must not grant side-effect authority, %s=%#v in %#v", field, result[field], result)
		}
	}
	if !strings.Contains(stdout.String(), `"safe_to_continue": true`) {
		t.Fatalf("expected JSON stdout for ready policy, got %s", stdout.String())
	}
}

func TestPulseEventLoopPolicyStopsOnRequiredBlockers(t *testing.T) {
	base := map[string]string{
		"class-gate":         "examples/class-gate/gate.dry-run.low-risk-code.json",
		"ci":                 "examples/pulse-event-loop-policy/ci.passed.json",
		"repo-state":         "examples/pulse-event-loop-policy/repo.clean.json",
		"evidence-freshness": "examples/pulse-event-loop-policy/evidence.fresh.json",
		"sentinel":           "examples/pulse-event-loop-policy/sentinel.no-hold.json",
		"promoter":           "examples/pulse-event-loop-policy/promoter.ready.json",
		"branch-cleanup":     "examples/pulse-event-loop-policy/branch-cleanup.passed.json",
		"scope":              "examples/pulse-event-loop-policy/scope.passed.json",
	}
	for _, tc := range []struct {
		name      string
		flag      string
		fixture   string
		wantCheck string
	}{
		{name: "class_gate", flag: "class-gate", fixture: "examples/class-gate/gate.blocked.low-risk-code.json", wantCheck: "class_gate"},
		{name: "failing_ci", flag: "ci", fixture: "examples/pulse-event-loop-policy/ci.failed.json", wantCheck: "ci_status"},
		{name: "dirty_repo", flag: "repo-state", fixture: "examples/pulse-event-loop-policy/repo.dirty.json", wantCheck: "repo_cleanliness"},
		{name: "stale_evidence", flag: "evidence-freshness", fixture: "examples/pulse-event-loop-policy/evidence.stale.json", wantCheck: "evidence_freshness"},
		{name: "sentinel_hold", flag: "sentinel", fixture: "examples/pulse-event-loop-policy/sentinel.hold.json", wantCheck: "sentinel_hold"},
		{name: "promoter_denial", flag: "promoter", fixture: "examples/pulse-event-loop-policy/promoter.denied.json", wantCheck: "promoter_readiness"},
		{name: "branch_cleanup_failure", flag: "branch-cleanup", fixture: "examples/pulse-event-loop-policy/branch-cleanup.failed.json", wantCheck: "branch_cleanup"},
		{name: "broadened_scope", flag: "scope", fixture: "examples/pulse-event-loop-policy/scope.broadened.json", wantCheck: "scope_boundary"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			paths := make(map[string]string, len(base))
			for key, value := range base {
				paths[key] = value
			}
			paths[tc.flag] = tc.fixture
			outPath := filepath.Join(t.TempDir(), "pulse-event-loop-policy.json")
			args := []string{"pulse", "event-loop-policy"}
			for _, flag := range []string{"class-gate", "ci", "repo-state", "evidence-freshness", "sentinel", "promoter", "branch-cleanup", "scope"} {
				args = append(args, "--"+flag, paths[flag])
			}
			args = append(args, "--out", outPath)
			var stdout, stderr bytes.Buffer
			code := Run(args, &stdout, &stderr)
			if code == 0 {
				t.Fatalf("Run returned success for blocker %s; stdout=%s stderr=%s", tc.name, stdout.String(), stderr.String())
			}
			result := readObjectFixture(t, outPath)
			if result["status"] != "blocked" ||
				result["allowed_next_action"] != "stop_event_loop" ||
				result["safe_to_continue"] != false ||
				result["first_failing_check"] != tc.wantCheck {
				t.Fatalf("unexpected blocked event-loop policy result: %#v", result)
			}
		})
	}
}

func TestPulseEventLoopPolicyContractFixtureValidates(t *testing.T) {
	schema, err := readArbitraryJSON("docs/contracts/foundry-pulse-event-loop-policy-v0.1.schema.json")
	if err != nil {
		t.Fatalf("read pulse event-loop policy schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("pulse event-loop policy schema is not an object: %#v", schema)
	}
	validFixture, err := readArbitraryJSON("examples/contract-fixtures/valid/foundry-pulse-event-loop-policy-v0.1.json")
	if err != nil {
		t.Fatalf("read valid pulse event-loop policy fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, validFixture, "$"); err != nil {
		t.Fatalf("valid pulse event-loop policy fixture failed schema: %v", err)
	}
	invalidFixture, err := readArbitraryJSON("examples/contract-fixtures/invalid/foundry-pulse-event-loop-policy-v0.1.json")
	if err != nil {
		t.Fatalf("read invalid pulse event-loop policy fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, invalidFixture, "$"); err == nil {
		t.Fatalf("invalid pulse event-loop policy fixture unexpectedly passed schema")
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

func TestPulseRunRequiresReadyStartGateBeforeBundle(t *testing.T) {
	outDir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"pulse", "run",
		"--start-gate", "examples/pulse-overnight-start-gate/blocked-blueprint-clarification.json",
		"--out", outDir,
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run returned success for blocked start gate; stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	decisionPath := filepath.Join(outDir, "pulse-runner-start-decision.json")
	decision := readObjectFixture(t, decisionPath)
	if decision["schema_version"] != "ao.foundry.pulse-runner-start-decision.v0.1" ||
		decision["status"] != "blocked" ||
		decision["allowed_next_action"] != "request_blueprint_clarification" ||
		decision["first_failing_check"] != "intake_preflight" {
		t.Fatalf("unexpected blocked runner decision: %#v", decision)
	}
	if _, err := os.Stat(filepath.Join(outDir, "pulse-event.json")); err == nil {
		t.Fatalf("blocked start gate should refuse to build pulse-event.json")
	}
	if !strings.Contains(stderr.String(), "Pulse runner start gate is blocked") {
		t.Fatalf("stderr missing blocked start gate reason: %s", stderr.String())
	}
}

func TestCIWorkflowRunsPulseStartGateRegression(t *testing.T) {
	workflow, err := os.ReadFile(repoPath(".github/workflows/ci.yml"))
	if err != nil {
		t.Fatalf("read CI workflow: %v", err)
	}
	text := string(workflow)
	for _, want := range []string{
		"Pulse start gate regression",
		"TestPulseRunRequiresReadyStartGateBeforeBundle",
		"TestPulseRunAcceptsReadyStartGateAndWritesDecision",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("CI workflow missing %q", want)
		}
	}
}

func TestPulseRunAcceptsReadyStartGateAndWritesDecision(t *testing.T) {
	outDir := t.TempDir()
	event := runPulseForEvent(t, []string{
		"pulse", "run",
		"--start-gate", "examples/pulse-overnight-start-gate/ready.json",
		"--out", outDir,
	})
	if event["status"] != "ready" {
		t.Fatalf("ready start gate should allow pulse bundle, got %#v", event)
	}
	decision := readObjectFixture(t, filepath.Join(outDir, "pulse-runner-start-decision.json"))
	if decision["schema_version"] != "ao.foundry.pulse-runner-start-decision.v0.1" ||
		decision["status"] != "ready" ||
		decision["start_gate_path"] != "examples/pulse-overnight-start-gate/ready.json" {
		t.Fatalf("unexpected ready runner decision: %#v", decision)
	}
	sourceDigests, ok := decision["source_digests"].([]any)
	if !ok || len(sourceDigests) < 3 {
		t.Fatalf("runner decision should include the start gate and source evidence digests: %#v", decision)
	}
}

func TestPulseRunnerStartDecisionContractFixtureValidates(t *testing.T) {
	schema, err := readArbitraryJSON("docs/contracts/foundry-pulse-runner-start-decision-v0.1.schema.json")
	if err != nil {
		t.Fatalf("read pulse runner start decision schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("pulse runner start decision schema is not an object: %#v", schema)
	}
	validFixture, err := readArbitraryJSON("examples/contract-fixtures/valid/foundry-pulse-runner-start-decision-v0.1.json")
	if err != nil {
		t.Fatalf("read valid pulse runner start decision fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, validFixture, "$"); err != nil {
		t.Fatalf("valid pulse runner start decision fixture failed schema: %v", err)
	}
	invalidFixture, err := readArbitraryJSON("examples/contract-fixtures/invalid/foundry-pulse-runner-start-decision-v0.1.json")
	if err != nil {
		t.Fatalf("read invalid pulse runner start decision fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, invalidFixture, "$"); err == nil {
		t.Fatalf("invalid pulse runner start decision fixture unexpectedly passed schema")
	}
}

func TestLiveMutationRequestContractFixtureValidates(t *testing.T) {
	schema, err := readArbitraryJSON("docs/contracts/foundry-live-mutation-request-v0.1.schema.json")
	if err != nil {
		t.Fatalf("read live mutation request schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("live mutation request schema is not an object: %#v", schema)
	}
	validFixture, err := readArbitraryJSON("examples/contract-fixtures/valid/foundry-live-mutation-request-v0.1.json")
	if err != nil {
		t.Fatalf("read valid live mutation request fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, validFixture, "$"); err != nil {
		t.Fatalf("valid live mutation request fixture failed schema: %v", err)
	}
	request := validFixture.(map[string]any)
	if request["mode"] != "dry_run_only" || request["requested_authority_schema"] != "covenant.live-mutation-authority.v1" {
		t.Fatalf("unexpected live mutation request authority boundary: %#v", request)
	}
	boundaries := request["authority_boundaries"].(map[string]any)
	if boundaries["live_mutation_allowed"] != false || boundaries["provider_calls_allowed"] != false {
		t.Fatalf("live mutation request must not allow live mutation or providers: %#v", boundaries)
	}
	invalidFixture, err := readArbitraryJSON("examples/contract-fixtures/invalid/foundry-live-mutation-request-v0.1.json")
	if err != nil {
		t.Fatalf("read invalid live mutation request fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, invalidFixture, "$"); err == nil {
		t.Fatalf("invalid live mutation request fixture unexpectedly passed schema")
	}
}

func TestLiveDocsApprovalRequestContractFixtureValidates(t *testing.T) {
	schema, err := readArbitraryJSON("docs/contracts/foundry-live-mutation-approval-request-v0.1.schema.json")
	if err != nil {
		t.Fatalf("read live docs approval request schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("live docs approval request schema is not an object: %#v", schema)
	}
	validFixture, err := readArbitraryJSON("examples/contract-fixtures/valid/foundry-live-mutation-approval-request-v0.1.json")
	if err != nil {
		t.Fatalf("read valid live docs approval request fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, validFixture, "$"); err != nil {
		t.Fatalf("valid live docs approval request fixture failed schema: %v", err)
	}
	request := validFixture.(map[string]any)
	if request["status"] != "pending_operator_approval" || request["first_live_class"] != "docs_only" {
		t.Fatalf("unexpected live docs approval request identity: %#v", request)
	}
	if request["safe_to_request"] != true || request["safe_to_execute"] != false {
		t.Fatalf("approval request must be requestable but not executable: %#v", request)
	}
	boundaries := request["authority_boundaries"].(map[string]any)
	if boundaries["mutates_repositories"] != false ||
		boundaries["provider_calls_allowed"] != false ||
		boundaries["release_or_publish_allowed"] != false ||
		boundaries["fully_unsupervised_complex_mutation_claimed"] != false {
		t.Fatalf("approval request must preserve non-execution boundaries: %#v", boundaries)
	}
	invalidFixture, err := readArbitraryJSON("examples/contract-fixtures/invalid/foundry-live-mutation-approval-request-v0.1.json")
	if err != nil {
		t.Fatalf("read invalid live docs approval request fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, invalidFixture, "$"); err == nil {
		t.Fatalf("invalid live docs approval request fixture unexpectedly passed schema")
	}
}

func TestLiveDocsApprovalGateScript(t *testing.T) {
	script := repoPath("scripts/live-docs-approval-gate.sh")
	outDir := t.TempDir()
	readyOut := filepath.Join(outDir, "ready.json")
	readyCmd := exec.Command("bash", script,
		"--request", repoPath("examples/live-docs-approval/request.json"),
		"--ticket", repoPath("examples/live-docs-approval/ticket-approved.json"),
		"--out", readyOut,
	)
	readyCmd.Dir = repoPath(".")
	if out, err := readyCmd.CombinedOutput(); err != nil {
		t.Fatalf("approval gate ready failed: %v\n%s", err, string(out))
	}
	ready := readObjectFixture(t, readyOut)
	if ready["status"] != "ready" || ready["safe_to_execute"] != true || ready["approval_state"] != "approved" {
		t.Fatalf("unexpected ready approval gate: %#v", ready)
	}

	blockedOut := filepath.Join(outDir, "blocked.json")
	blockedCmd := exec.Command("bash", script,
		"--request", repoPath("examples/live-docs-approval/request.json"),
		"--ticket", repoPath("examples/live-docs-approval/ticket-pending.json"),
		"--out", blockedOut,
	)
	blockedCmd.Dir = repoPath(".")
	if out, err := blockedCmd.CombinedOutput(); err != nil {
		t.Fatalf("approval gate blocked should write blocked result, got error: %v\n%s", err, string(out))
	}
	blocked := readObjectFixture(t, blockedOut)
	if blocked["status"] != "blocked" || blocked["safe_to_execute"] != false || blocked["first_failing_check"] != "approval_state" {
		t.Fatalf("unexpected blocked approval gate: %#v", blocked)
	}
}

func TestLiveDocsApprovalGateContractFixtureValidates(t *testing.T) {
	schema, err := readArbitraryJSON("docs/contracts/foundry-live-docs-approval-gate-v0.1.schema.json")
	if err != nil {
		t.Fatalf("read live docs approval gate schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("live docs approval gate schema is not an object: %#v", schema)
	}
	validFixture, err := readArbitraryJSON("examples/contract-fixtures/valid/foundry-live-docs-approval-gate-v0.1.json")
	if err != nil {
		t.Fatalf("read valid live docs approval gate fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, validFixture, "$"); err != nil {
		t.Fatalf("valid live docs approval gate fixture failed schema: %v", err)
	}
	invalidFixture, err := readArbitraryJSON("examples/contract-fixtures/invalid/foundry-live-docs-approval-gate-v0.1.json")
	if err != nil {
		t.Fatalf("read invalid live docs approval gate fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, invalidFixture, "$"); err == nil {
		t.Fatalf("invalid live docs approval gate fixture unexpectedly passed schema")
	}
}

func TestMutationClassGateEvaluateReady(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "class-gate.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"class-gate", "evaluate",
		"--atlas", "examples/class-gate/atlas-classification.docs-multi.json",
		"--covenant", "examples/class-gate/covenant-ticket.docs-multi.json",
		"--sentinel", "examples/class-gate/sentinel.no-hold.docs-multi.json",
		"--promoter", "examples/class-gate/promoter.ready.docs-multi.json",
		"--rollback", "examples/class-gate/rollback.passed.docs-multi.json",
		"--command", "examples/class-gate/command-readback.docs-multi.json",
		"--ci", "examples/class-gate/ci.passed.docs-multi.json",
		"--out", outPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("class gate returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	gate := readObjectFixture(t, outPath)
	if gate["schema_version"] != "ao.foundry.mutation-class-gate.v0.1" ||
		gate["status"] != "ready" ||
		gate["mutation_class"] != "docs_only_multi_file" ||
		gate["safe_to_request"] != true ||
		gate["safe_to_execute"] != true {
		t.Fatalf("unexpected ready class gate: %#v", gate)
	}
	if gate["authority_boundary"] != "single_class_only" {
		t.Fatalf("class gate must remain single-class: %#v", gate)
	}
	denied, ok := gate["denied_classes"].([]any)
	if !ok || len(denied) == 0 {
		t.Fatalf("class gate must deny other classes: %#v", gate)
	}
}

func TestMutationClassGateEvaluateTestOnlyReady(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "class-gate.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"class-gate", "evaluate",
		"--atlas", "examples/class-gate/atlas-classification.test-only.json",
		"--covenant", "examples/class-gate/covenant-ticket.test-only.json",
		"--sentinel", "examples/class-gate/sentinel.no-hold.test-only.json",
		"--promoter", "examples/class-gate/promoter.ready.test-only.json",
		"--rollback", "examples/class-gate/rollback.passed.test-only.json",
		"--command", "examples/class-gate/command-readback.test-only.json",
		"--ci", "examples/class-gate/ci.passed.test-only.json",
		"--out", outPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test_only class gate returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	gate := readObjectFixture(t, outPath)
	if gate["status"] != "ready" ||
		gate["mutation_class"] != "test_only" ||
		gate["safe_to_request"] != true ||
		gate["safe_to_execute"] != true {
		t.Fatalf("unexpected test_only class gate: %#v", gate)
	}
}

func TestMutationClassGateBlocksLowRiskCodeWithoutTestOnlySuccess(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "class-gate.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"class-gate", "evaluate",
		"--atlas", "examples/class-gate/atlas-classification.low-risk-code.json",
		"--covenant", "examples/class-gate/covenant-ticket.low-risk-code.json",
		"--sentinel", "examples/class-gate/sentinel.no-hold.low-risk-code.json",
		"--promoter", "examples/class-gate/promoter.ready.low-risk-code.json",
		"--rollback", "examples/class-gate/rollback.passed.low-risk-code.json",
		"--command", "examples/class-gate/command-readback.low-risk-code.json",
		"--ci", "examples/class-gate/ci.passed.low-risk-code.json",
		"--out", outPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("class gate returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	gate := readObjectFixture(t, outPath)
	if gate["status"] != "blocked" ||
		gate["mutation_class"] != "low_risk_code" ||
		gate["safe_to_request"] != false ||
		gate["safe_to_execute"] != false ||
		gate["first_failing_check"] != "test_only_success evidence is required for low_risk_code" {
		t.Fatalf("low_risk_code must remain blocked without test_only_success: %#v", gate)
	}
	required, ok := gate["required_evidence"].([]any)
	if !ok {
		t.Fatalf("low_risk_code gate missing required evidence list: %#v", gate["required_evidence"])
	}
	hasTestOnlySuccess := false
	for _, item := range required {
		if item == "test_only_success" {
			hasTestOnlySuccess = true
		}
	}
	if !hasTestOnlySuccess {
		t.Fatalf("low_risk_code gate must require test_only_success: %#v", gate["required_evidence"])
	}
}

func TestMutationClassGateLowRiskCodeDryRunReadyWithTestOnlySuccess(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "class-gate.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"class-gate", "evaluate",
		"--atlas", "examples/class-gate/atlas-classification.low-risk-code.json",
		"--covenant", "examples/class-gate/covenant-ticket.low-risk-code.json",
		"--sentinel", "examples/class-gate/sentinel.no-hold.low-risk-code.json",
		"--promoter", "examples/class-gate/promoter.ready.low-risk-code.json",
		"--rollback", "examples/class-gate/rollback.passed.low-risk-code.json",
		"--command", "examples/class-gate/command-readback.low-risk-code.json",
		"--ci", "examples/class-gate/ci.passed.low-risk-code.json",
		"--test-only-success", "examples/class-gate/test-only-success.low-risk-code.json",
		"--out", outPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("class gate returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	gate := readObjectFixture(t, outPath)
	if gate["status"] != "ready" ||
		gate["mutation_class"] != "low_risk_code" ||
		gate["safe_to_request"] != true ||
		gate["safe_to_execute"] != false ||
		gate["first_failing_check"] != "" {
		t.Fatalf("low_risk_code dry-run gate should be requestable but not executable: %#v", gate)
	}
	required, ok := gate["required_evidence"].([]any)
	if !ok {
		t.Fatalf("low_risk_code gate must retain test_only_success evidence: %#v", gate["required_evidence"])
	}
	hasTestOnlySuccess := false
	for _, item := range required {
		if item == "test_only_success" {
			hasTestOnlySuccess = true
		}
	}
	if !hasTestOnlySuccess {
		t.Fatalf("low_risk_code gate must retain test_only_success evidence: %#v", gate["required_evidence"])
	}
	if !strings.Contains(fmt.Sprint(gate["next_actions"]), "dry-run") {
		t.Fatalf("low_risk_code next action must stay dry-run only: %#v", gate["next_actions"])
	}
	boundary, ok := gate["class_boundary_checks"].(map[string]any)
	if !ok {
		t.Fatalf("low_risk_code gate must include class boundary readback: %#v", gate)
	}
	for _, field := range []string{
		"atlas_classification_only",
		"atlas_required_gates_complete",
		"covenant_exact_scope",
		"covenant_class_bound",
		"covenant_digest_bound",
		"covenant_single_use",
		"covenant_unconsumed",
		"covenant_live_mutation_denied",
		"sentinel_no_hold",
		"rollback_patch_present",
		"rollback_verification_commands_present",
		"command_read_only",
		"ci_passed",
		"ci_required_checks_present",
		"test_only_live_evidence",
		"safe_to_request",
	} {
		if boundary[field] != true {
			t.Fatalf("low_risk_code boundary %s must be true: %#v", field, boundary)
		}
	}
	if boundary["safe_to_execute"] != false ||
		boundary["promoter_boundary"] != "low_risk_code_only" ||
		boundary["command_current_class"] != "test_only" ||
		boundary["command_next_class"] != "low_risk_code" ||
		boundary["command_mutates_repositories"] != false {
		t.Fatalf("low_risk_code boundary readback drifted: %#v", boundary)
	}
	audit, ok := gate["denial_audit"].(map[string]any)
	if !ok {
		t.Fatalf("low_risk_code gate must include denial_audit readback: %#v", gate)
	}
	if audit["next_denied_class"] != "low_risk_code" ||
		audit["safe_to_execute"] != false ||
		audit["exact_next_action"] != "build_low_risk_code_promotion_prerequisites" {
		t.Fatalf("low_risk_code denial audit drifted: %#v", audit)
	}
	for _, want := range []string{
		"policy:low_risk_code_live_promotion",
		"rollback_proof:low_risk_code_live",
		"sentinel_clear:low_risk_code_live",
		"promoter_promotion:low_risk_code_live",
		"ci_passed:low_risk_code_live",
	} {
		if !objectStringSliceContains(audit, "missing_policy_evidence", want) &&
			!objectStringSliceContains(audit, "missing_rollback_evidence", want) &&
			!objectStringSliceContains(audit, "missing_sentinel_promoter_evidence", want) &&
			!objectStringSliceContains(audit, "ci_requirements", want) {
			t.Fatalf("low_risk_code denial audit missing %s: %#v", want, audit)
		}
	}
	checked := readObjectFixture(t, "examples/class-gate/gate.dry-run.low-risk-code.json")
	if checked["status"] != "ready" ||
		checked["mutation_class"] != "low_risk_code" ||
		checked["safe_to_request"] != true ||
		checked["safe_to_execute"] != false {
		t.Fatalf("checked low_risk_code dry-run fixture drifted: %#v", checked)
	}
	if _, ok := checked["denial_audit"].(map[string]any); !ok {
		t.Fatalf("checked low_risk_code dry-run fixture must include denial audit: %#v", checked)
	}
	if _, ok := checked["class_boundary_checks"].(map[string]any); !ok {
		t.Fatalf("checked low_risk_code dry-run fixture must include class boundary readback: %#v", checked)
	}
}

func TestMutationClassGateLowRiskCodeFailsClosedOnBoundaryEvidence(t *testing.T) {
	baseArgs := func(covenantPath, sentinelPath, commandPath, outPath string) []string {
		return []string{
			"class-gate", "evaluate",
			"--atlas", "examples/class-gate/atlas-classification.low-risk-code.json",
			"--covenant", covenantPath,
			"--sentinel", sentinelPath,
			"--promoter", "examples/class-gate/promoter.ready.low-risk-code.json",
			"--rollback", "examples/class-gate/rollback.passed.low-risk-code.json",
			"--command", commandPath,
			"--ci", "examples/class-gate/ci.passed.low-risk-code.json",
			"--test-only-success", "examples/class-gate/test-only-success.low-risk-code.json",
			"--out", outPath,
		}
	}
	cases := []struct {
		name          string
		covenantPath  string
		sentinelPath  string
		commandPath   string
		mutate        func(t *testing.T) (covenantPath, sentinelPath, commandPath string)
		wantFirstFail string
	}{
		{
			name: "covenant_not_exact_scope",
			mutate: func(t *testing.T) (string, string, string) {
				t.Helper()
				covenant := readObjectFixture(t, "examples/class-gate/covenant-ticket.low-risk-code.json")
				boundaries, ok := covenant["authority_boundaries"].(map[string]any)
				if !ok {
					t.Fatalf("fixture missing authority_boundaries: %#v", covenant)
				}
				boundaries["exact_scope"] = false
				path := filepath.Join(t.TempDir(), "covenant-ticket.json")
				mustWriteJSONForTest(t, path, covenant)
				return path, "examples/class-gate/sentinel.no-hold.low-risk-code.json", "examples/class-gate/command-readback.low-risk-code.json"
			},
			wantFirstFail: "covenant_class_ticket must remain exact-scope, class-bound, digest-bound, unconsumed, and single-use",
		},
		{
			name: "sentinel_hold_flag",
			mutate: func(t *testing.T) (string, string, string) {
				t.Helper()
				sentinel := readObjectFixture(t, "examples/class-gate/sentinel.no-hold.low-risk-code.json")
				sentinel["hold"] = true
				path := filepath.Join(t.TempDir(), "sentinel.json")
				mustWriteJSONForTest(t, path, sentinel)
				return "examples/class-gate/covenant-ticket.low-risk-code.json", path, "examples/class-gate/command-readback.low-risk-code.json"
			},
			wantFirstFail: "sentinel_no_hold must be an explicit no-hold verdict",
		},
		{
			name: "command_not_read_only",
			mutate: func(t *testing.T) (string, string, string) {
				t.Helper()
				command := readObjectFixture(t, "examples/class-gate/command-readback.low-risk-code.json")
				command["operator_mode"] = "write_enabled"
				path := filepath.Join(t.TempDir(), "command.json")
				mustWriteJSONForTest(t, path, command)
				return "examples/class-gate/covenant-ticket.low-risk-code.json", "examples/class-gate/sentinel.no-hold.low-risk-code.json", path
			},
			wantFirstFail: "command_readback must remain read-only from test_only to low_risk_code",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			covenantPath, sentinelPath, commandPath := tc.mutate(t)
			outPath := filepath.Join(t.TempDir(), "class-gate.json")
			var stdout, stderr bytes.Buffer
			code := Run(baseArgs(covenantPath, sentinelPath, commandPath, outPath), &stdout, &stderr)
			if code != 0 {
				t.Fatalf("class gate returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
			}
			gate := readObjectFixture(t, outPath)
			if gate["status"] != "blocked" ||
				gate["mutation_class"] != "low_risk_code" ||
				gate["safe_to_request"] != false ||
				gate["safe_to_execute"] != false ||
				gate["first_failing_check"] != tc.wantFirstFail {
				t.Fatalf("low_risk_code boundary must fail closed on %s: %#v", tc.name, gate)
			}
		})
	}
}

func TestMutationClassGateMultiRepoLowRiskDryRunRequiresSafeSequencedPlan(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "class-gate.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"class-gate", "evaluate",
		"--atlas", "examples/class-gate/atlas-classification.multi-repo-low-risk.json",
		"--covenant", "examples/class-gate/covenant-ticket.multi-repo-low-risk.json",
		"--sentinel", "examples/class-gate/sentinel.no-hold.multi-repo-low-risk.json",
		"--promoter", "examples/class-gate/promoter.ready.multi-repo-low-risk.json",
		"--rollback", "examples/class-gate/rollback.passed.multi-repo-low-risk.json",
		"--command", "examples/class-gate/command-readback.multi-repo-low-risk.json",
		"--ci", "examples/class-gate/ci.passed.multi-repo-low-risk.json",
		"--multi-repo-plan", "examples/class-gate/multi-repo-plan.safe.json",
		"--out", outPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("class gate returned %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	gate := readObjectFixture(t, outPath)
	if gate["status"] != "ready" ||
		gate["mutation_class"] != "multi_repo_low_risk" ||
		gate["safe_to_request"] != true ||
		gate["safe_to_execute"] != false ||
		gate["first_failing_check"] != "" {
		t.Fatalf("multi_repo_low_risk dry-run gate should be requestable but not executable: %#v", gate)
	}
	if !strings.Contains(fmt.Sprint(gate["required_evidence"]), "multi_repo_sequencing_plan") {
		t.Fatalf("multi_repo_low_risk gate must require sequencing plan evidence: %#v", gate["required_evidence"])
	}
	if !strings.Contains(fmt.Sprint(gate["repo_execution_plan"]), "ao-atlas") ||
		!strings.Contains(fmt.Sprint(gate["repo_execution_plan"]), "ao-foundry") ||
		!strings.Contains(fmt.Sprint(gate["repo_execution_plan"]), "ao-command") ||
		!strings.Contains(fmt.Sprint(gate["repo_execution_plan"]), "dry-run-pr:ao-atlas") ||
		!strings.Contains(fmt.Sprint(gate["repo_execution_plan"]), "merge_after") {
		t.Fatalf("multi_repo_low_risk gate must retain repo execution plan: %#v", gate["repo_execution_plan"])
	}
	if !strings.Contains(fmt.Sprint(gate["repo_safety"]), "prevent_concurrent_unsafe_execution") {
		t.Fatalf("multi_repo_low_risk gate must record unsafe concurrency prevention: %#v", gate["repo_safety"])
	}

	blockedOut := filepath.Join(t.TempDir(), "class-gate-blocked.json")
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{
		"class-gate", "evaluate",
		"--atlas", "examples/class-gate/atlas-classification.multi-repo-low-risk.json",
		"--covenant", "examples/class-gate/covenant-ticket.multi-repo-low-risk.json",
		"--sentinel", "examples/class-gate/sentinel.no-hold.multi-repo-low-risk.json",
		"--promoter", "examples/class-gate/promoter.ready.multi-repo-low-risk.json",
		"--rollback", "examples/class-gate/rollback.passed.multi-repo-low-risk.json",
		"--command", "examples/class-gate/command-readback.multi-repo-low-risk.json",
		"--ci", "examples/class-gate/ci.passed.multi-repo-low-risk.json",
		"--multi-repo-plan", "examples/class-gate/multi-repo-plan.unsafe-concurrent.json",
		"--out", blockedOut,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("blocked class gate should still emit a decision, got %d; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	blocked := readObjectFixture(t, blockedOut)
	if blocked["status"] != "blocked" ||
		blocked["safe_to_request"] != false ||
		blocked["safe_to_execute"] != false ||
		!strings.Contains(fmt.Sprint(blocked["first_failing_check"]), "unsafe concurrent") {
		t.Fatalf("unsafe concurrent multi-repo plan must fail closed: %#v", blocked)
	}

	unorderedOut := filepath.Join(t.TempDir(), "class-gate-unordered.json")
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{
		"class-gate", "evaluate",
		"--atlas", "examples/class-gate/atlas-classification.multi-repo-low-risk.json",
		"--covenant", "examples/class-gate/covenant-ticket.multi-repo-low-risk.json",
		"--sentinel", "examples/class-gate/sentinel.no-hold.multi-repo-low-risk.json",
		"--promoter", "examples/class-gate/promoter.ready.multi-repo-low-risk.json",
		"--rollback", "examples/class-gate/rollback.passed.multi-repo-low-risk.json",
		"--command", "examples/class-gate/command-readback.multi-repo-low-risk.json",
		"--ci", "examples/class-gate/ci.passed.multi-repo-low-risk.json",
		"--multi-repo-plan", "examples/class-gate/multi-repo-plan.unordered-dependency.json",
		"--out", unorderedOut,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("unordered class gate should still emit a decision, got %d; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	unordered := readObjectFixture(t, unorderedOut)
	if unordered["status"] != "blocked" ||
		unordered["safe_to_request"] != false ||
		unordered["safe_to_execute"] != false ||
		!strings.Contains(fmt.Sprint(unordered["first_failing_check"]), "must appear earlier in dependency order") {
		t.Fatalf("unordered multi-repo plan must fail closed: %#v", unordered)
	}
}

func TestMutationClassGateFailsClosedWithoutSentinel(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "class-gate.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"class-gate", "evaluate",
		"--atlas", "examples/class-gate/atlas-classification.docs-multi.json",
		"--covenant", "examples/class-gate/covenant-ticket.docs-multi.json",
		"--promoter", "examples/class-gate/promoter.ready.docs-multi.json",
		"--rollback", "examples/class-gate/rollback.passed.docs-multi.json",
		"--command", "examples/class-gate/command-readback.docs-multi.json",
		"--ci", "examples/class-gate/ci.passed.docs-multi.json",
		"--out", outPath,
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("class gate missing sentinel returned %d, want 2; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "--sentinel") {
		t.Fatalf("stderr missing sentinel requirement: %s", stderr.String())
	}
}

func TestMutationClassGateContractFixtureValidates(t *testing.T) {
	schema, err := readArbitraryJSON("docs/contracts/foundry-mutation-class-gate-v0.1.schema.json")
	if err != nil {
		t.Fatalf("read mutation class gate schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("mutation class gate schema is not an object: %#v", schema)
	}
	validFixture, err := readArbitraryJSON("examples/contract-fixtures/valid/foundry-mutation-class-gate-v0.1.json")
	if err != nil {
		t.Fatalf("read valid mutation class gate fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, validFixture, "$"); err != nil {
		t.Fatalf("valid mutation class gate fixture failed schema: %v", err)
	}
	invalidFixture, err := readArbitraryJSON("examples/contract-fixtures/invalid/foundry-mutation-class-gate-v0.1.json")
	if err != nil {
		t.Fatalf("read invalid mutation class gate fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, invalidFixture, "$"); err == nil {
		t.Fatalf("invalid mutation class gate fixture unexpectedly passed schema")
	}
}

func TestWorktreeIsolationProofScriptBlocksDirtyAndReusedCandidates(t *testing.T) {
	script := repoPath("scripts/live-mutation-worktree-isolation-proof.sh")
	scriptData, err := os.ReadFile(script)
	if err != nil {
		t.Fatalf("read worktree isolation proof script: %v", err)
	}
	for _, want := range []string{
		"ao.foundry.worktree-isolation-proof.v0.1",
		"clean_worktree",
		"reuse_block",
		"dry_run_only",
		"live_mutation_allowed:false",
	} {
		if !strings.Contains(string(scriptData), want) {
			t.Fatalf("worktree isolation proof script missing %q", want)
		}
	}
	for _, forbidden := range []string{"git checkout", "git switch", "git worktree add", "gh pr merge", "gh release", "curl "} {
		if strings.Contains(string(scriptData), forbidden) {
			t.Fatalf("worktree isolation proof script contains forbidden live action %q", forbidden)
		}
	}

	runProof := func(t *testing.T, candidate string, wantReady bool, wantFailingCheck string) map[string]any {
		t.Helper()
		outPath := filepath.ToSlash(filepath.Join("tmp", strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())+".json"))
		t.Cleanup(func() { _ = os.Remove(repoPath(outPath)) })
		cmd := exec.Command(
			"bash",
			script,
			"--candidate", candidate,
			"--out", outPath,
			"--json",
		)
		cmd.Dir = repoPath(".")
		out, err := cmd.CombinedOutput()
		if wantReady && err != nil {
			t.Fatalf("worktree isolation proof failed: %v\n%s", err, string(out))
		}
		if !wantReady && err == nil {
			t.Fatalf("worktree isolation proof unexpectedly passed for %s:\n%s", candidate, string(out))
		}
		result := readObjectFixture(t, outPath)
		if wantReady && result["status"] != "ready" {
			t.Fatalf("expected ready proof, got %#v", result)
		}
		if !wantReady {
			if result["status"] != "blocked" || result["first_failing_check"] != wantFailingCheck {
				t.Fatalf("expected blocked proof at %s, got %#v", wantFailingCheck, result)
			}
		}
		boundaries := result["authority_boundaries"].(map[string]any)
		if boundaries["mutates_repositories"] != false ||
			boundaries["schedules_work"] != false ||
			boundaries["executes_work"] != false ||
			boundaries["approves_work"] != false {
			t.Fatalf("proof must stay read-only/dry-run: %#v", boundaries)
		}
		return result
	}

	t.Run("ready", func(t *testing.T) {
		result := runProof(t, "examples/live-mutation-worktree-isolation/clean-isolated.candidate.json", true, "")
		if result["candidate_sha256"] == "" {
			t.Fatalf("ready proof missing candidate digest: %#v", result)
		}
	})
	t.Run("dirty", func(t *testing.T) {
		runProof(t, "examples/live-mutation-worktree-isolation/dirty-worktree.candidate.json", false, "clean_worktree")
	})
	t.Run("reused", func(t *testing.T) {
		runProof(t, "examples/live-mutation-worktree-isolation/reused-worktree.candidate.json", false, "reuse_block")
	})
}

func TestWorktreeIsolationProofContractFixtureValidates(t *testing.T) {
	schema, err := readArbitraryJSON("docs/contracts/foundry-worktree-isolation-proof-v0.1.schema.json")
	if err != nil {
		t.Fatalf("read worktree isolation proof schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("worktree isolation proof schema is not an object: %#v", schema)
	}
	validFixture, err := readArbitraryJSON("examples/contract-fixtures/valid/foundry-worktree-isolation-proof-v0.1.json")
	if err != nil {
		t.Fatalf("read valid worktree isolation proof fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, validFixture, "$"); err != nil {
		t.Fatalf("valid worktree isolation proof fixture failed schema: %v", err)
	}
	invalidFixture, err := readArbitraryJSON("examples/contract-fixtures/invalid/foundry-worktree-isolation-proof-v0.1.json")
	if err != nil {
		t.Fatalf("read invalid worktree isolation proof fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, invalidFixture, "$"); err == nil {
		t.Fatalf("invalid worktree isolation proof fixture unexpectedly passed schema")
	}
}

func TestLiveDocsWorktreePrepareScriptBlocksUnsafeCandidates(t *testing.T) {
	script := repoPath("scripts/live-docs-worktree-prepare.sh")
	scriptData, err := os.ReadFile(script)
	if err != nil {
		t.Fatalf("read live docs worktree prepare script: %v", err)
	}
	for _, want := range []string{
		"ao.foundry.live-docs-worktree-prepare.v0.1",
		"approval_gate",
		"branch_isolation",
		"clean_worktree",
		"reuse_block",
		"docs_only_path_plan",
		"validation_only:true",
	} {
		if !strings.Contains(string(scriptData), want) {
			t.Fatalf("live docs worktree prepare script missing %q", want)
		}
	}
	for _, forbidden := range []string{"git checkout", "git switch", "git worktree add", "gh pr merge", "gh release", "curl "} {
		if strings.Contains(string(scriptData), forbidden) {
			t.Fatalf("live docs worktree prepare script contains forbidden live action %q", forbidden)
		}
	}

	runPrepare := func(t *testing.T, candidate string, gate string, wantReady bool, wantFailingCheck string) map[string]any {
		t.Helper()
		outPath := filepath.ToSlash(filepath.Join("tmp", strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())+".json"))
		t.Cleanup(func() { _ = os.Remove(repoPath(outPath)) })
		cmd := exec.Command(
			"bash",
			script,
			"--candidate", candidate,
			"--approval-gate", gate,
			"--out", outPath,
			"--json",
		)
		cmd.Dir = repoPath(".")
		out, err := cmd.CombinedOutput()
		if wantReady && err != nil {
			t.Fatalf("live docs worktree prepare failed: %v\n%s", err, string(out))
		}
		if !wantReady && err == nil {
			t.Fatalf("live docs worktree prepare unexpectedly passed for %s:\n%s", candidate, string(out))
		}
		result := readObjectFixture(t, outPath)
		if wantReady && result["status"] != "ready" {
			t.Fatalf("expected ready prepare result, got %#v", result)
		}
		if !wantReady {
			if result["status"] != "blocked" || result["first_failing_check"] != wantFailingCheck {
				t.Fatalf("expected blocked prepare result at %s, got %#v", wantFailingCheck, result)
			}
		}
		boundaries := result["authority_boundaries"].(map[string]any)
		if boundaries["mutates_repositories"] != false ||
			boundaries["creates_worktree"] != false ||
			boundaries["creates_branch"] != false ||
			boundaries["executes_work"] != false ||
			boundaries["approves_work"] != false {
			t.Fatalf("prepare gate must stay validation-only: %#v", boundaries)
		}
		return result
	}

	readyGate := "examples/contract-fixtures/valid/foundry-live-docs-approval-gate-v0.1.json"
	blockedGate := "examples/contract-fixtures/invalid/foundry-live-docs-approval-gate-v0.1.json"

	t.Run("ready", func(t *testing.T) {
		result := runPrepare(t, "examples/live-docs-worktree-prepare/ready.candidate.json", readyGate, true, "")
		if result["can_start_docs_only_pr_rehearsal"] != true {
			t.Fatalf("ready result must allow docs-only PR rehearsal start: %#v", result)
		}
	})
	t.Run("approval gate blocked", func(t *testing.T) {
		runPrepare(t, "examples/live-docs-worktree-prepare/ready.candidate.json", blockedGate, false, "approval_gate")
	})
	t.Run("dirty", func(t *testing.T) {
		runPrepare(t, "examples/live-docs-worktree-prepare/dirty-worktree.candidate.json", readyGate, false, "clean_worktree")
	})
	t.Run("reused", func(t *testing.T) {
		runPrepare(t, "examples/live-docs-worktree-prepare/reused-branch.candidate.json", readyGate, false, "reuse_block")
	})
	t.Run("wrong branch", func(t *testing.T) {
		runPrepare(t, "examples/live-docs-worktree-prepare/wrong-branch.candidate.json", readyGate, false, "branch_isolation")
	})
	t.Run("unsafe path", func(t *testing.T) {
		runPrepare(t, "examples/live-docs-worktree-prepare/unsafe-path.candidate.json", readyGate, false, "docs_only_path_plan")
	})
}

func TestLiveDocsWorktreePrepareContractFixtureValidates(t *testing.T) {
	schema, err := readArbitraryJSON("docs/contracts/foundry-live-docs-worktree-prepare-v0.1.schema.json")
	if err != nil {
		t.Fatalf("read live docs worktree prepare schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("live docs worktree prepare schema is not an object: %#v", schema)
	}
	validFixture, err := readArbitraryJSON("examples/contract-fixtures/valid/foundry-live-docs-worktree-prepare-v0.1.json")
	if err != nil {
		t.Fatalf("read valid live docs worktree prepare fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, validFixture, "$"); err != nil {
		t.Fatalf("valid live docs worktree prepare fixture failed schema: %v", err)
	}
	invalidFixture, err := readArbitraryJSON("examples/contract-fixtures/invalid/foundry-live-docs-worktree-prepare-v0.1.json")
	if err != nil {
		t.Fatalf("read invalid live docs worktree prepare fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, invalidFixture, "$"); err == nil {
		t.Fatalf("invalid live docs worktree prepare fixture unexpectedly passed schema")
	}
}

func TestLiveDocsRollbackExecutionRehearsalScriptExecutesAndRollsBackInFixtureWorkspace(t *testing.T) {
	script := repoPath("scripts/live-docs-rollback-execution-rehearsal.sh")
	scriptData, err := os.ReadFile(script)
	if err != nil {
		t.Fatalf("read live docs rollback execution rehearsal script: %v", err)
	}
	for _, want := range []string{
		"ao.foundry.live-docs-rollback-execution-rehearsal.v0.1",
		"fixture_workspace_only",
		"git -C \"$workspace\" apply",
		"proposed_patch_apply",
		"rollback_patch_apply",
		"mutates_repositories:false",
	} {
		if !strings.Contains(string(scriptData), want) {
			t.Fatalf("live docs rollback execution script missing %q", want)
		}
	}
	for _, forbidden := range []string{"git checkout", "git switch", "git worktree add", "gh pr merge", "gh release", "curl "} {
		if strings.Contains(string(scriptData), forbidden) {
			t.Fatalf("live docs rollback execution script contains forbidden live action %q", forbidden)
		}
	}

	runRehearsal := func(t *testing.T, candidate string, wantReady bool, wantFailingCheck string) map[string]any {
		t.Helper()
		outPath := filepath.ToSlash(filepath.Join("tmp", strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())+".json"))
		t.Cleanup(func() { _ = os.Remove(repoPath(outPath)) })
		cmd := exec.Command(
			"bash",
			script,
			"--candidate", candidate,
			"--out", outPath,
			"--json",
		)
		cmd.Dir = repoPath(".")
		out, err := cmd.CombinedOutput()
		if wantReady && err != nil {
			t.Fatalf("live docs rollback execution failed: %v\n%s", err, string(out))
		}
		if !wantReady && err == nil {
			t.Fatalf("live docs rollback execution unexpectedly passed for %s:\n%s", candidate, string(out))
		}
		result := readObjectFixture(t, outPath)
		if wantReady && (result["status"] != "ready" || result["rollback_verified"] != true) {
			t.Fatalf("expected ready verified rehearsal, got %#v", result)
		}
		if !wantReady {
			if result["status"] != "blocked" || result["first_failing_check"] != wantFailingCheck {
				t.Fatalf("expected blocked rehearsal at %s, got %#v", wantFailingCheck, result)
			}
		}
		boundaries := result["authority_boundaries"].(map[string]any)
		if boundaries["fixture_workspace_only"] != true ||
			boundaries["live_mutation_allowed"] != false ||
			boundaries["mutates_repositories"] != false ||
			boundaries["schedules_work"] != false ||
			boundaries["executes_work"] != false ||
			boundaries["approves_work"] != false {
			t.Fatalf("rehearsal must stay fixture-workspace-only: %#v", boundaries)
		}
		return result
	}

	t.Run("ready", func(t *testing.T) {
		result := runRehearsal(t, "examples/live-docs-rollback-execution/docs-only.candidate.json", true, "")
		summary := result["execution_summary"].(map[string]any)
		if summary["proposed_patch_applied"] != true || summary["rollback_patch_applied"] != true {
			t.Fatalf("ready rehearsal must apply and roll back inside fixture workspace: %#v", result)
		}
	})
	t.Run("unsafe_path", func(t *testing.T) {
		runRehearsal(t, "examples/live-docs-rollback-execution/unsafe-path.candidate.json", false, "docs_only_target")
	})
	t.Run("missing_rollback", func(t *testing.T) {
		runRehearsal(t, "examples/live-docs-rollback-execution/missing-rollback.candidate.json", false, "rollback_patch_present")
	})
}

func TestLiveDocsRollbackExecutionRehearsalContractFixtureValidates(t *testing.T) {
	schema, err := readArbitraryJSON("docs/contracts/foundry-live-docs-rollback-execution-rehearsal-v0.1.schema.json")
	if err != nil {
		t.Fatalf("read live docs rollback execution schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("live docs rollback execution schema is not an object: %#v", schema)
	}
	validFixture, err := readArbitraryJSON("examples/contract-fixtures/valid/foundry-live-docs-rollback-execution-rehearsal-v0.1.json")
	if err != nil {
		t.Fatalf("read valid live docs rollback execution fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, validFixture, "$"); err != nil {
		t.Fatalf("valid live docs rollback execution fixture failed schema: %v", err)
	}
	invalidFixture, err := readArbitraryJSON("examples/contract-fixtures/invalid/foundry-live-docs-rollback-execution-rehearsal-v0.1.json")
	if err != nil {
		t.Fatalf("read invalid live docs rollback execution fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, invalidFixture, "$"); err == nil {
		t.Fatalf("invalid live docs rollback execution fixture unexpectedly passed schema")
	}
}

func TestApprovedLiveDocsDryRunChainScript(t *testing.T) {
	script := repoPath("scripts/approved-live-docs-dry-run-chain.sh")
	scriptData, err := os.ReadFile(script)
	if err != nil {
		t.Fatalf("read approved live docs dry-run chain script: %v", err)
	}
	scriptText := string(scriptData)
	for _, want := range []string{
		"ao.foundry.approved-live-docs-dry-run-chain.v0.1",
		"live-docs-approval-gate.sh",
		"live-docs-worktree-prepare.sh",
		"live-docs-rollback-execution-rehearsal.sh",
		"ao.forge.live-docs-execution-guard.v0.1",
		"ao2.docs-only-patch-packet.v1",
		"ao.sentinel.live-docs-mutation-hold.v0.1",
		"ao.promoter.live-docs-mutation-boundary.v0.1",
		"ao.command.live-docs-mutation-status.v0.1",
		"safe_to_execute:false",
		"fully_unsupervised_complex_mutation_claimed:false",
	} {
		if !strings.Contains(scriptText, want) {
			t.Fatalf("approved live docs dry-run chain script missing %q", want)
		}
	}
	for _, forbidden := range []string{"git checkout", "git switch", "git worktree add", "gh pr merge", "git " + "push", "gh " + "release", "npm publish", "twine upload", "docker push", "kubectl apply", "terraform apply", "curl "} {
		if strings.Contains(scriptText, forbidden) {
			t.Fatalf("approved live docs dry-run chain script contains forbidden live action %q", forbidden)
		}
	}

	outDir := filepath.ToSlash(filepath.Join("target", strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())))
	t.Cleanup(func() { _ = os.RemoveAll(repoPath(outDir)) })
	cmd := exec.Command("bash", script, "--out", outDir)
	cmd.Dir = repoPath(".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("approved live docs dry-run chain failed: %v\n%s", err, string(out))
	}
	summary := readObjectFixture(t, filepath.Join(outDir, "summary.json"))
	if summary["status"] != "ready" || summary["first_live_class"] != "docs_only" {
		t.Fatalf("unexpected approved live docs chain identity: %#v", summary)
	}
	assessment := summary["readiness_assessment"].(map[string]any)
	if assessment["approved_docs_only_dry_run_chain"] != "ready" ||
		assessment["safe_to_request"] != true ||
		assessment["safe_to_execute"] != false ||
		assessment["requires_live_docs_pr_rehearsal_gate"] != true ||
		assessment["live_mutation_performed"] != false ||
		assessment["fully_unsupervised_complex_mutation_claimed"] != false {
		t.Fatalf("unexpected approved live docs readiness assessment: %#v", assessment)
	}
	boundaries := summary["authority_boundaries"].(map[string]any)
	if boundaries["live_mutation_allowed"] != false ||
		boundaries["mutates_repositories"] != false ||
		boundaries["creates_branch"] != false ||
		boundaries["creates_worktree"] != false ||
		boundaries["executes_work"] != false ||
		boundaries["approves_work"] != false {
		t.Fatalf("chain must remain dry-run and non-mutating: %#v", boundaries)
	}
	artifacts := summary["source_artifacts"].([]any)
	if len(artifacts) != 10 {
		t.Fatalf("expected ten approved live docs source artifacts, got %#v", artifacts)
	}
}

func TestApprovedLiveDocsDryRunChainContractFixtureValidates(t *testing.T) {
	schema, err := readArbitraryJSON("docs/contracts/foundry-approved-live-docs-dry-run-chain-v0.1.schema.json")
	if err != nil {
		t.Fatalf("read approved live docs dry-run chain schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("approved live docs dry-run chain schema is not an object: %#v", schema)
	}
	validFixture, err := readArbitraryJSON("examples/contract-fixtures/valid/foundry-approved-live-docs-dry-run-chain-v0.1.json")
	if err != nil {
		t.Fatalf("read valid approved live docs dry-run chain fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, validFixture, "$"); err != nil {
		t.Fatalf("valid approved live docs dry-run chain fixture failed schema: %v", err)
	}
	assessment := validFixture.(map[string]any)["readiness_assessment"].(map[string]any)
	if assessment["safe_to_execute"] != false || assessment["fully_unsupervised_complex_mutation_claimed"] != false {
		t.Fatalf("valid approved live docs chain must stay dry-run: %#v", assessment)
	}
	invalidFixture, err := readArbitraryJSON("examples/contract-fixtures/invalid/foundry-approved-live-docs-dry-run-chain-v0.1.json")
	if err != nil {
		t.Fatalf("read invalid approved live docs dry-run chain fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, invalidFixture, "$"); err == nil {
		t.Fatalf("invalid approved live docs dry-run chain fixture unexpectedly passed schema")
	}
}

func TestLiveDocsPRRehearsalGateScriptRequiresExplicitApprovalArtifact(t *testing.T) {
	chainScript := repoPath("scripts/approved-live-docs-dry-run-chain.sh")
	gateScript := repoPath("scripts/live-docs-pr-rehearsal-gate.sh")
	gateData, err := os.ReadFile(gateScript)
	if err != nil {
		t.Fatalf("read live docs PR rehearsal gate script: %v", err)
	}
	gateText := string(gateData)
	for _, want := range []string{
		"ao.foundry.live-docs-pr-rehearsal-gate.v0.1",
		"approval_artifact",
		"approval_digest_binding",
		"request_operator_approval",
		"start_first_docs_only_live_pr_rehearsal",
		"mutates_repositories:false",
		"creates_branch:false",
		"opens_pr:false",
		"fully_unsupervised_complex_mutation_claimed:false",
	} {
		if !strings.Contains(gateText, want) {
			t.Fatalf("live docs PR rehearsal gate script missing %q", want)
		}
	}
	for _, forbidden := range []string{"git checkout", "git switch", "git worktree add", "gh pr create", "gh pr merge", "git " + "push", "gh " + "release", "npm publish", "curl "} {
		if strings.Contains(gateText, forbidden) {
			t.Fatalf("live docs PR rehearsal gate script contains forbidden live action %q", forbidden)
		}
	}

	outDir := filepath.ToSlash(filepath.Join("target", strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())))
	chainDir := outDir + "/chain"
	chainSummary := chainDir + "/summary.json"
	blockedPath := outDir + "/blocked-gate.json"
	readyPath := outDir + "/ready-gate.json"
	t.Cleanup(func() { _ = os.RemoveAll(repoPath(outDir)) })
	chainCmd := exec.Command("bash", chainScript, "--out", chainDir)
	chainCmd.Dir = repoPath(".")
	if out, err := chainCmd.CombinedOutput(); err != nil {
		t.Fatalf("approved live docs chain failed: %v\n%s", err, string(out))
	}

	blockedCmd := exec.Command("bash", gateScript,
		"--chain", chainSummary,
		"--out", blockedPath,
		"--json",
	)
	blockedCmd.Dir = repoPath(".")
	if out, err := blockedCmd.CombinedOutput(); err != nil {
		t.Fatalf("blocked live docs PR rehearsal gate should emit blocked result: %v\n%s", err, string(out))
	}
	blocked := readObjectFixture(t, blockedPath)
	if blocked["status"] != "blocked" ||
		blocked["safe_to_execute"] != false ||
		blocked["exact_next_step"] != "request_operator_approval" ||
		blocked["first_failing_check"] != "approval_artifact" {
		t.Fatalf("missing approval artifact should block execution: %#v", blocked)
	}

	readyCmd := exec.Command("bash", gateScript,
		"--chain", chainSummary,
		"--approval-artifact", "examples/live-docs-approval/ticket-approved.json",
		"--out", readyPath,
		"--json",
	)
	readyCmd.Dir = repoPath(".")
	if out, err := readyCmd.CombinedOutput(); err != nil {
		t.Fatalf("ready live docs PR rehearsal gate failed: %v\n%s", err, string(out))
	}
	ready := readObjectFixture(t, readyPath)
	if ready["status"] != "ready" ||
		ready["safe_to_execute"] != true ||
		ready["exact_next_step"] != "start_first_docs_only_live_pr_rehearsal" {
		t.Fatalf("explicit approval artifact should allow docs-only PR rehearsal gate: %#v", ready)
	}
	boundaries := ready["authority_boundaries"].(map[string]any)
	if boundaries["mutates_repositories"] != false ||
		boundaries["creates_branch"] != false ||
		boundaries["creates_worktree"] != false ||
		boundaries["opens_pr"] != false ||
		boundaries["executes_work"] != false ||
		boundaries["approves_work"] != false {
		t.Fatalf("PR rehearsal gate must only emit a decision: %#v", boundaries)
	}
}

func TestLiveDocsPRRehearsalGateContractFixtureValidates(t *testing.T) {
	schema, err := readArbitraryJSON("docs/contracts/foundry-live-docs-pr-rehearsal-gate-v0.1.schema.json")
	if err != nil {
		t.Fatalf("read live docs PR rehearsal gate schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("live docs PR rehearsal gate schema is not an object: %#v", schema)
	}
	validFixture, err := readArbitraryJSON("examples/contract-fixtures/valid/foundry-live-docs-pr-rehearsal-gate-v0.1.json")
	if err != nil {
		t.Fatalf("read valid live docs PR rehearsal gate fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, validFixture, "$"); err != nil {
		t.Fatalf("valid live docs PR rehearsal gate fixture failed schema: %v", err)
	}
	if validFixture.(map[string]any)["safe_to_execute"] != true {
		t.Fatalf("valid PR rehearsal gate fixture should allow only the first docs-only rehearsal decision")
	}
	invalidFixture, err := readArbitraryJSON("examples/contract-fixtures/invalid/foundry-live-docs-pr-rehearsal-gate-v0.1.json")
	if err != nil {
		t.Fatalf("read invalid live docs PR rehearsal gate fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, invalidFixture, "$"); err == nil {
		t.Fatalf("invalid live docs PR rehearsal gate fixture unexpectedly passed schema")
	}
}

func TestFirstLiveDocsReadinessRollupScript(t *testing.T) {
	chainScript := repoPath("scripts/approved-live-docs-dry-run-chain.sh")
	gateScript := repoPath("scripts/live-docs-pr-rehearsal-gate.sh")
	rollupScript := repoPath("scripts/first-live-docs-readiness-rollup.sh")
	rollupData, err := os.ReadFile(rollupScript)
	if err != nil {
		t.Fatalf("read first live docs readiness rollup script: %v", err)
	}
	rollupText := string(rollupData)
	for _, want := range []string{
		"ao.foundry.first-live-docs-readiness-rollup.v0.1",
		"safe_to_request",
		"safe_to_execute",
		"approved_scope",
		"docs_only",
		"fully_unsupervised_complex_mutation_claimed:false",
		"mutates_repositories:false",
	} {
		if !strings.Contains(rollupText, want) {
			t.Fatalf("first live docs readiness rollup script missing %q", want)
		}
	}
	for _, forbidden := range []string{"git checkout", "git switch", "git worktree add", "gh pr create", "gh pr merge", "git " + "push", "gh " + "release", "npm publish", "curl "} {
		if strings.Contains(rollupText, forbidden) {
			t.Fatalf("first live docs readiness rollup script contains forbidden live action %q", forbidden)
		}
	}

	outDir := filepath.ToSlash(filepath.Join("target", strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())))
	chainDir := outDir + "/chain"
	blockedGate := outDir + "/blocked-gate.json"
	readyGate := outDir + "/ready-gate.json"
	blockedRollup := outDir + "/blocked-rollup.json"
	readyRollup := outDir + "/ready-rollup.json"
	t.Cleanup(func() { _ = os.RemoveAll(repoPath(outDir)) })
	chainCmd := exec.Command("bash", chainScript, "--out", chainDir)
	chainCmd.Dir = repoPath(".")
	if out, err := chainCmd.CombinedOutput(); err != nil {
		t.Fatalf("approved live docs chain failed: %v\n%s", err, string(out))
	}
	blockedGateCmd := exec.Command("bash", gateScript, "--chain", chainDir+"/summary.json", "--out", blockedGate)
	blockedGateCmd.Dir = repoPath(".")
	if out, err := blockedGateCmd.CombinedOutput(); err != nil {
		t.Fatalf("blocked PR gate failed: %v\n%s", err, string(out))
	}
	readyGateCmd := exec.Command("bash", gateScript, "--chain", chainDir+"/summary.json", "--approval-artifact", "examples/live-docs-approval/ticket-approved.json", "--out", readyGate)
	readyGateCmd.Dir = repoPath(".")
	if out, err := readyGateCmd.CombinedOutput(); err != nil {
		t.Fatalf("ready PR gate failed: %v\n%s", err, string(out))
	}

	runRollup := func(path string, gate string) map[string]any {
		t.Helper()
		cmd := exec.Command("bash", rollupScript, "--chain", chainDir+"/summary.json", "--pr-gate", gate, "--out", path)
		cmd.Dir = repoPath(".")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("first live docs readiness rollup failed: %v\n%s", err, string(out))
		}
		return readObjectFixture(t, path)
	}
	blocked := runRollup(blockedRollup, blockedGate)
	if blocked["status"] != "blocked" ||
		blocked["safe_to_request"] != true ||
		blocked["safe_to_execute"] != false ||
		blocked["exact_next_step"] != "request_operator_approval" {
		t.Fatalf("unexpected blocked first live docs rollup: %#v", blocked)
	}
	ready := runRollup(readyRollup, readyGate)
	if ready["status"] != "ready" ||
		ready["safe_to_request"] != true ||
		ready["safe_to_execute"] != true ||
		ready["approved_scope"] != "docs_only" ||
		ready["exact_next_step"] != "start_first_docs_only_live_pr_rehearsal" {
		t.Fatalf("unexpected ready first live docs rollup: %#v", ready)
	}
	boundaries := ready["authority_boundaries"].(map[string]any)
	if boundaries["live_mutation_performed"] != false ||
		boundaries["mutates_repositories"] != false ||
		boundaries["creates_branch"] != false ||
		boundaries["opens_pr"] != false ||
		boundaries["fully_unsupervised_complex_mutation_claimed"] != false {
		t.Fatalf("rollup must stay non-mutating and scoped: %#v", boundaries)
	}
}

func TestFirstLiveDocsReadinessRollupContractFixtureValidates(t *testing.T) {
	schema, err := readArbitraryJSON("docs/contracts/foundry-first-live-docs-readiness-rollup-v0.1.schema.json")
	if err != nil {
		t.Fatalf("read first live docs readiness rollup schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("first live docs readiness rollup schema is not an object: %#v", schema)
	}
	validFixture, err := readArbitraryJSON("examples/contract-fixtures/valid/foundry-first-live-docs-readiness-rollup-v0.1.json")
	if err != nil {
		t.Fatalf("read valid first live docs readiness rollup fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, validFixture, "$"); err != nil {
		t.Fatalf("valid first live docs readiness rollup fixture failed schema: %v", err)
	}
	if validFixture.(map[string]any)["safe_to_execute"] != true {
		t.Fatalf("valid first live docs rollup should be ready for only the docs-only PR rehearsal decision")
	}
	invalidFixture, err := readArbitraryJSON("examples/contract-fixtures/invalid/foundry-first-live-docs-readiness-rollup-v0.1.json")
	if err != nil {
		t.Fatalf("read invalid first live docs readiness rollup fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, invalidFixture, "$"); err == nil {
		t.Fatalf("invalid first live docs readiness rollup fixture unexpectedly passed schema")
	}
}

func TestLiveMutationRollbackRehearsalScriptBlocksMissingRollbackAndUnsafeAuthority(t *testing.T) {
	script := repoPath("scripts/live-mutation-rollback-rehearsal.sh")
	scriptData, err := os.ReadFile(script)
	if err != nil {
		t.Fatalf("read rollback rehearsal script: %v", err)
	}
	for _, want := range []string{
		"ao.foundry.live-mutation-rollback-rehearsal.v0.1",
		"rollback_patch_present",
		"quarantine_plan",
		"kill_switch",
		"dry_run_only",
		"live_mutation_allowed:false",
	} {
		if !strings.Contains(string(scriptData), want) {
			t.Fatalf("rollback rehearsal script missing %q", want)
		}
	}
	for _, forbidden := range []string{"git apply", "git checkout", "git switch", "git worktree add", "gh pr merge", "gh release", "curl "} {
		if strings.Contains(string(scriptData), forbidden) {
			t.Fatalf("rollback rehearsal script contains forbidden live action %q", forbidden)
		}
	}

	runRehearsal := func(t *testing.T, candidate string, wantReady bool, wantFailingCheck string) map[string]any {
		t.Helper()
		outPath := filepath.ToSlash(filepath.Join("tmp", strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())+".json"))
		t.Cleanup(func() { _ = os.Remove(repoPath(outPath)) })
		cmd := exec.Command(
			"bash",
			script,
			"--candidate", candidate,
			"--out", outPath,
			"--json",
		)
		cmd.Dir = repoPath(".")
		out, err := cmd.CombinedOutput()
		if wantReady && err != nil {
			t.Fatalf("rollback rehearsal failed: %v\n%s", err, string(out))
		}
		if !wantReady && err == nil {
			t.Fatalf("rollback rehearsal unexpectedly passed for %s:\n%s", candidate, string(out))
		}
		result := readObjectFixture(t, outPath)
		if wantReady && result["status"] != "ready" {
			t.Fatalf("expected ready rehearsal, got %#v", result)
		}
		if !wantReady {
			if result["status"] != "blocked" || result["first_failing_check"] != wantFailingCheck {
				t.Fatalf("expected blocked rehearsal at %s, got %#v", wantFailingCheck, result)
			}
		}
		boundaries := result["authority_boundaries"].(map[string]any)
		if boundaries["mutates_repositories"] != false ||
			boundaries["schedules_work"] != false ||
			boundaries["executes_work"] != false ||
			boundaries["approves_work"] != false {
			t.Fatalf("rehearsal must stay read-only/dry-run: %#v", boundaries)
		}
		return result
	}

	t.Run("ready", func(t *testing.T) {
		result := runRehearsal(t, "examples/live-mutation-rollback/docs-only-rollback.candidate.json", true, "")
		artifacts := result["patch_artifacts"].([]any)
		if len(artifacts) != 2 {
			t.Fatalf("ready rehearsal should bind proposed and rollback patch artifacts: %#v", result)
		}
	})
	t.Run("missing_rollback", func(t *testing.T) {
		runRehearsal(t, "examples/live-mutation-rollback/missing-rollback.candidate.json", false, "rollback_patch_present")
	})
	t.Run("unsafe_authority", func(t *testing.T) {
		runRehearsal(t, "examples/live-mutation-rollback/unsafe-authority.candidate.json", false, "authority_boundaries")
	})
}

func TestLiveMutationRollbackRehearsalContractFixtureValidates(t *testing.T) {
	schema, err := readArbitraryJSON("docs/contracts/foundry-live-mutation-rollback-rehearsal-v0.1.schema.json")
	if err != nil {
		t.Fatalf("read rollback rehearsal schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("rollback rehearsal schema is not an object: %#v", schema)
	}
	validFixture, err := readArbitraryJSON("examples/contract-fixtures/valid/foundry-live-mutation-rollback-rehearsal-v0.1.json")
	if err != nil {
		t.Fatalf("read valid rollback rehearsal fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, validFixture, "$"); err != nil {
		t.Fatalf("valid rollback rehearsal fixture failed schema: %v", err)
	}
	invalidFixture, err := readArbitraryJSON("examples/contract-fixtures/invalid/foundry-live-mutation-rollback-rehearsal-v0.1.json")
	if err != nil {
		t.Fatalf("read invalid rollback rehearsal fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, invalidFixture, "$"); err == nil {
		t.Fatalf("invalid rollback rehearsal fixture unexpectedly passed schema")
	}
}

func TestGovernedLiveMutationDryRunChainContractFixtureValidates(t *testing.T) {
	schema, err := readArbitraryJSON("docs/contracts/foundry-governed-live-mutation-dry-run-chain-v0.1.schema.json")
	if err != nil {
		t.Fatalf("read governed live mutation chain schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("governed live mutation chain schema is not an object: %#v", schema)
	}
	validFixture, err := readArbitraryJSON("examples/contract-fixtures/valid/foundry-governed-live-mutation-dry-run-chain-v0.1.json")
	if err != nil {
		t.Fatalf("read valid governed live mutation chain fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, validFixture, "$"); err != nil {
		t.Fatalf("valid governed live mutation chain fixture failed schema: %v", err)
	}
	assessment := validFixture.(map[string]any)["readiness_assessment"].(map[string]any)
	if assessment["live_mutation_performed"] != false || assessment["ungated_live_mutation_claim"] != false {
		t.Fatalf("valid governed chain must not claim live mutation: %#v", assessment)
	}
	invalidFixture, err := readArbitraryJSON("examples/contract-fixtures/invalid/foundry-governed-live-mutation-dry-run-chain-v0.1.json")
	if err != nil {
		t.Fatalf("read invalid governed live mutation chain fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, invalidFixture, "$"); err == nil {
		t.Fatalf("invalid governed live mutation chain fixture unexpectedly passed schema")
	}
}

func TestLiveMutationReadinessRollupContractFixtureValidates(t *testing.T) {
	schema, err := readArbitraryJSON("docs/contracts/foundry-live-mutation-readiness-rollup-v0.1.schema.json")
	if err != nil {
		t.Fatalf("read live mutation readiness rollup schema: %v", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("live mutation readiness rollup schema is not an object: %#v", schema)
	}
	validFixture, err := readArbitraryJSON("examples/contract-fixtures/valid/foundry-live-mutation-readiness-rollup-v0.1.json")
	if err != nil {
		t.Fatalf("read valid live mutation readiness rollup fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, validFixture, "$"); err != nil {
		t.Fatalf("valid live mutation readiness rollup fixture failed schema: %v", err)
	}
	tinyClass := validFixture.(map[string]any)["first_tiny_live_mutation_class"].(map[string]any)
	if tinyClass["safe_to_request"] != true || tinyClass["safe_to_execute"] != false {
		t.Fatalf("valid readiness rollup must allow request but not execution: %#v", tinyClass)
	}
	invalidFixture, err := readArbitraryJSON("examples/contract-fixtures/invalid/foundry-live-mutation-readiness-rollup-v0.1.json")
	if err != nil {
		t.Fatalf("read invalid live mutation readiness rollup fixture: %v", err)
	}
	if err := validateJSONSchemaValue(root, root, invalidFixture, "$"); err == nil {
		t.Fatalf("invalid live mutation readiness rollup fixture unexpectedly passed schema")
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
	packetPath := filepath.Join(t.TempDir(), "factory-packet.json")
	packetData, err := os.ReadFile(repoPath("examples/packets/ao-foundry-bootstrap.factory-packet.json"))
	if err != nil {
		t.Fatalf("read packet fixture: %v", err)
	}
	if err := os.WriteFile(packetPath, packetData, 0o644); err != nil {
		t.Fatalf("write packet: %v", err)
	}
	fresh := time.Now()
	if err := os.Chtimes(packetPath, fresh, fresh); err != nil {
		t.Fatalf("make packet fresh: %v", err)
	}

	event := runPulseForEvent(t, []string{"pulse", "run", "--out", t.TempDir(), "--forge-live-packet", packetPath})
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

func readObjectFixture(t *testing.T, path string) map[string]any {
	t.Helper()
	value, err := readArbitraryJSON(path)
	if err != nil {
		t.Fatalf("read JSON object %s: %v", path, err)
	}
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("JSON fixture %s is not an object: %#v", path, value)
	}
	return object
}

func objectStringSliceContains(object map[string]any, key, want string) bool {
	values, ok := object[key].([]any)
	if !ok {
		return false
	}
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
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
		"ao-atlas":          "28346305503",
		"ao-forge":          "28246591616",
		"ao-command":        "28345912142",
		"ao2":               "28339961675",
		"ao2-control-plane": "28280708823",
		"ao-covenant":       "28186617447",
	}
	opsRuns := map[string]string{
		"ao-foundry":        "28027968419",
		"ao-atlas":          "28346305510",
		"ao-forge":          "28321477720",
		"ao-command":        "28321548229",
		"ao2":               "28321735689",
		"ao2-control-plane": "28321488512",
		"ao-covenant":       "28321567179",
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
	for _, repo := range []string{"ao-foundry", "ao-atlas", "ao-forge", "ao-command", "ao2", "ao2-control-plane", "ao-covenant"} {
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

func atlasRegistryFixture() string {
	return filepath.Join("..", "..", "examples", "registry", "atlas-demo.foundry-registry.json")
}

func atlasImportFixture() string {
	return filepath.Join("..", "..", "examples", "atlas", "foundry-import.json")
}

func atlasRunLinkFixture() string {
	return filepath.Join("..", "..", "examples", "atlas", "run-link.completed.json")
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

func writeEvalResultFixture(t *testing.T, dir, name string, score, maxScore int, status string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	result := EvalResult{
		SchemaVersion: "ao.foundry.eval-result.v0.1",
		ScorecardID:   "rsi-self-improvement",
		RunID:         strings.TrimSuffix(name, ".eval-result.json"),
		Status:        status,
		Score:         score,
		MaxScore:      maxScore,
		Threshold:     score,
		Dimensions:    []EvalDimension{},
		NextActions:   []string{},
	}
	mustWriteJSONForTest(t, path, result)
	return path
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
