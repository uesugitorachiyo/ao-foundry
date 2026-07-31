package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAOMissionSmokeValidatesFixtureReadbacks(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "ao-mission-smoke.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"ao-mission", "smoke",
		"--route", filepath.Join("examples", "ao-mission-smoke", "mission-route-readback.json"),
		"--snapshot", filepath.Join("examples", "ao-mission-smoke", "governance-snapshot-readback.json"),
		"--out", outPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ao_mission_smoke="+outPath) {
		t.Fatalf("expected smoke output path, got %q", stdout.String())
	}
	var smoke map[string]any
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &smoke); err != nil {
		t.Fatal(err)
	}
	if smoke["status"] != "ready" || smoke["executes_work"] != false || smoke["approves_work"] != false {
		t.Fatalf("bad smoke readback: %#v", smoke)
	}
	if smoke["route"] != "ao-atlas" || smoke["current_owner"] != "ao-mission" {
		t.Fatalf("bad route/current owner: %#v", smoke)
	}
}

func TestAOMissionFinalRollupSmokeValidatesClosure(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "ao-mission-final-rollup-smoke.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"ao-mission", "final-rollup-smoke",
		"--mission-final-rollup", filepath.Join("examples", "ao-mission-smoke", "mission-final-rollup.json"),
		"--foundry-final-rollup", filepath.Join("examples", "ao-mission-smoke", "foundry-final-rollup.json"),
		"--gateway-readiness-rollup", filepath.Join("examples", "ao-mission-smoke", "gateway-readiness-rollup.json"),
		"--out", outPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ao_mission_final_rollup_smoke="+outPath) {
		t.Fatalf("expected final-rollup smoke output path, got %q", stdout.String())
	}
	var smoke map[string]any
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &smoke); err != nil {
		t.Fatal(err)
	}
	if smoke["status"] != "ready" || smoke["completed_nodes"].(float64) != smoke["total_nodes"].(float64) {
		t.Fatalf("bad final-rollup smoke readback: %#v", smoke)
	}
	if smoke["executes_work"] != false || smoke["approves_work"] != false || smoke["mutates_repositories"] != false {
		t.Fatalf("final-rollup smoke widened authority: %#v", smoke)
	}
	if smoke["gateway_readiness_rollup_bound"] != true || smoke["correlation_id"] != "corr-gateway-001" {
		t.Fatalf("final-rollup smoke did not bind gateway readiness correlation: %#v", smoke)
	}
}

func TestAOMissionFinalRollupSmokeAcceptsPromotedTerminalStatus(t *testing.T) {
	dir := t.TempDir()
	missionPath := filepath.Join(dir, "mission-final-rollup.json")
	foundryPath := filepath.Join(dir, "foundry-final-rollup.json")
	outPath := filepath.Join(dir, "smoke.json")
	if err := os.WriteFile(missionPath, []byte(`{"schema":"ao.mission.final-rollup.v0.1","mission_id":"mission-promoted","status":"done","completed_nodes":2,"total_nodes":2,"safe_to_execute":false,"executes_work":false,"approves_work":false,"mutates_repositories":false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(foundryPath, []byte(`{"schema":"ao.foundry.final-rollup.v0.1","mission_id":"mission-promoted","status":"promoted","completed_nodes":2,"total_nodes":2,"safe_to_execute":false,"executes_work":false,"approves_work":false,"mutates_repositories":false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"ao-mission", "final-rollup-smoke", "--mission-final-rollup", missionPath, "--foundry-final-rollup", foundryPath, "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("promoted final-rollup smoke failed: %s", stderr.String())
	}
	var smoke map[string]any
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &smoke); err != nil {
		t.Fatal(err)
	}
	if smoke["status"] != "ready" || smoke["foundry_terminal_status"] != "promoted" || smoke["executes_work"] != false {
		t.Fatalf("bad promoted terminal smoke: %#v", smoke)
	}
	summary, ok := smoke["terminal_readiness_summary"].(map[string]any)
	if !ok ||
		summary["foundry_terminal_status"] != "promoted" ||
		summary["status"] != "ready" ||
		summary["can_close_mission"] != true ||
		summary["requires_repair"] != false {
		t.Fatalf("promoted terminal smoke missing readiness summary: %#v", smoke)
	}
}

func TestAOMissionFinalRollupSmokeBindsDeniedTerminalStatus(t *testing.T) {
	dir := t.TempDir()
	missionPath := filepath.Join(dir, "mission-final-rollup.json")
	foundryPath := filepath.Join(dir, "foundry-final-rollup.json")
	outPath := filepath.Join(dir, "smoke.json")
	if err := os.WriteFile(missionPath, []byte(`{"schema":"ao.mission.final-rollup.v0.1","mission_id":"mission-denied","status":"blocked","completed_nodes":1,"total_nodes":2,"safe_to_execute":false,"executes_work":false,"approves_work":false,"mutates_repositories":false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(foundryPath, []byte(`{"schema":"ao.foundry.final-rollup.v0.1","mission_id":"mission-denied","status":"denied","completed_nodes":1,"total_nodes":2,"safe_to_execute":false,"executes_work":false,"approves_work":false,"mutates_repositories":false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"ao-mission", "final-rollup-smoke", "--mission-final-rollup", missionPath, "--foundry-final-rollup", foundryPath, "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("denied final-rollup smoke failed: %s", stderr.String())
	}
	var smoke map[string]any
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &smoke); err != nil {
		t.Fatal(err)
	}
	if smoke["status"] != "blocked" || smoke["foundry_terminal_status"] != "denied" || smoke["safe_to_execute"] != false {
		t.Fatalf("bad denied terminal smoke: %#v", smoke)
	}
	if !strings.Contains(fmt.Sprint(smoke["exact_next_action"]), "repair") {
		t.Fatalf("denied terminal smoke missing repair next action: %#v", smoke)
	}
	summary, ok := smoke["terminal_readiness_summary"].(map[string]any)
	if !ok ||
		summary["foundry_terminal_status"] != "denied" ||
		summary["status"] != "blocked" ||
		summary["can_close_mission"] != false ||
		summary["requires_repair"] != true {
		t.Fatalf("denied terminal smoke missing readiness summary: %#v", smoke)
	}
}

func TestAOMissionReadinessLedgerAndSummaryBindDeniedTerminalStatus(t *testing.T) {
	dir := t.TempDir()
	missionPath := filepath.Join(dir, "mission-final-rollup.json")
	foundryPath := filepath.Join(dir, "foundry-final-rollup.json")
	smokePath := filepath.Join(dir, "smoke.json")
	ledgerPath := filepath.Join(dir, "ledger.json")
	summaryPath := filepath.Join(dir, "summary.json")
	if err := os.WriteFile(missionPath, []byte(`{"schema":"ao.mission.final-rollup.v0.1","mission_id":"mission-denied","status":"blocked","completed_nodes":1,"total_nodes":2,"safe_to_execute":false,"executes_work":false,"approves_work":false,"mutates_repositories":false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(foundryPath, []byte(`{"schema":"ao.foundry.final-rollup.v0.1","mission_id":"mission-denied","status":"denied","completed_nodes":1,"total_nodes":2,"safe_to_execute":false,"executes_work":false,"approves_work":false,"mutates_repositories":false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"ao-mission", "final-rollup-smoke", "--mission-final-rollup", missionPath, "--foundry-final-rollup", foundryPath, "--out", smokePath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("denied final-rollup smoke failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"ao-mission", "readiness-ledger", "--final-rollup-smoke", smokePath, "--out", ledgerPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("denied readiness ledger failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"ao-mission", "rollup-summary", "--final-rollup-smoke", smokePath, "--readiness-ledger", ledgerPath, "--out", summaryPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("denied rollup summary failed: %s", stderr.String())
	}
	var ledger map[string]any
	data, err := os.ReadFile(ledgerPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &ledger); err != nil {
		t.Fatal(err)
	}
	if ledger["status"] != "blocked" || ledger["foundry_terminal_status"] != "denied" ||
		ledger["terminal_readiness_summary_bound"] != true ||
		ledger["safe_to_execute"] != false ||
		!strings.Contains(fmt.Sprint(ledger["exact_next_action"]), "repair") {
		t.Fatalf("denied readiness ledger missing terminal binding: %#v", ledger)
	}
	var rollupSummary map[string]any
	data, err = os.ReadFile(summaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &rollupSummary); err != nil {
		t.Fatal(err)
	}
	if rollupSummary["status"] != "blocked" ||
		rollupSummary["foundry_terminal_status"] != "denied" ||
		rollupSummary["completed_nodes"].(float64) != 1 ||
		rollupSummary["total_nodes"].(float64) != 2 ||
		rollupSummary["terminal_readiness_summary_bound"] != true ||
		rollupSummary["safe_to_execute"] != false ||
		!strings.Contains(fmt.Sprint(rollupSummary["exact_next_action"]), "repair") {
		t.Fatalf("denied rollup summary missing terminal binding: %#v", rollupSummary)
	}
}

func TestAOMissionReadinessLedgerConsumesFinalRollupSmoke(t *testing.T) {
	smokePath := filepath.Join(t.TempDir(), "ao-mission-final-rollup-smoke.json")
	ledgerPath := filepath.Join(t.TempDir(), "ao-mission-readiness-ledger.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"ao-mission", "final-rollup-smoke",
		"--mission-final-rollup", filepath.Join("examples", "ao-mission-smoke", "mission-final-rollup.json"),
		"--foundry-final-rollup", filepath.Join("examples", "ao-mission-smoke", "foundry-final-rollup.json"),
		"--out", smokePath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("final-rollup smoke failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{
		"ao-mission", "readiness-ledger",
		"--final-rollup-smoke", smokePath,
		"--out", ledgerPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("readiness ledger failed: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "ao_mission_readiness_ledger="+ledgerPath) {
		t.Fatalf("expected readiness ledger output path, got %q", stdout.String())
	}
	var ledger map[string]any
	data, err := os.ReadFile(ledgerPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &ledger); err != nil {
		t.Fatal(err)
	}
	if ledger["schema"] != "ao.foundry.ao-mission-readiness-ledger.v0.1" || ledger["status"] != "ready" {
		t.Fatalf("bad readiness ledger: %#v", ledger)
	}
	if ledger["executes_work"] != false || ledger["approves_work"] != false || ledger["mutates_repositories"] != false {
		t.Fatalf("readiness ledger widened authority: %#v", ledger)
	}
}

func TestAOMissionRollupSummaryBindsPortfolioAndReadiness(t *testing.T) {
	dir := t.TempDir()
	smokePath := filepath.Join(dir, "ao-mission-final-rollup-smoke.json")
	ledgerPath := filepath.Join(dir, "ao-mission-readiness-ledger.json")
	summaryPath := filepath.Join(dir, "ao-mission-rollup-summary.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"ao-mission", "final-rollup-smoke",
		"--mission-final-rollup", filepath.Join("examples", "ao-mission-smoke", "mission-final-rollup.json"),
		"--foundry-final-rollup", filepath.Join("examples", "ao-mission-smoke", "foundry-final-rollup.json"),
		"--out", smokePath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("final-rollup smoke failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{
		"ao-mission", "readiness-ledger",
		"--final-rollup-smoke", smokePath,
		"--out", ledgerPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("readiness ledger failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{
		"ao-mission", "rollup-summary",
		"--final-rollup-smoke", smokePath,
		"--readiness-ledger", ledgerPath,
		"--out", summaryPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("rollup summary failed: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "ao_mission_rollup_summary="+summaryPath) {
		t.Fatalf("expected rollup summary output path, got %q", stdout.String())
	}
	var summary map[string]any
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &summary); err != nil {
		t.Fatal(err)
	}
	if summary["schema"] != "ao.foundry.ao-mission-rollup-summary.v0.1" || summary["status"] != "ready" {
		t.Fatalf("bad rollup summary: %#v", summary)
	}
	if summary["portfolio_binding"] != "active_stack_readiness" || summary["readiness_bound"] != true {
		t.Fatalf("summary did not bind portfolio readiness: %#v", summary)
	}
	if summary["final_rollup_smoke_bound"] != true || summary["readiness_ledger_bound"] != true {
		t.Fatalf("summary did not bind both inputs: %#v", summary)
	}
	if summary["completed_nodes"].(float64) != summary["total_nodes"].(float64) {
		t.Fatalf("summary did not preserve complete node counts: %#v", summary)
	}
	if summary["safe_to_execute"] != false || summary["executes_work"] != false || summary["approves_work"] != false || summary["mutates_repositories"] != false {
		t.Fatalf("rollup summary widened authority: %#v", summary)
	}
}

func TestAOMissionE2ESmokeIsLockedIntoCI(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(body)
	for _, want := range []string{
		"name: AO Mission e2e smoke",
		"shell: bash",
		"scripts/ao-mission-atlas-foundry-e2e-smoke.sh",
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("CI workflow missing %q", want)
		}
	}
	script, err := os.ReadFile(filepath.Join("..", "..", "scripts", "ao-mission-atlas-foundry-e2e-smoke.sh"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"artifact-manifest.json",
		"scheduler-recovery-readback.json",
		"ledger-compaction-readback.json",
		"digest_negative_smoke=passed",
		"expected digest-mismatch manifest to fail",
	} {
		if !strings.Contains(string(script), want) {
			t.Fatalf("AO Mission e2e smoke script missing %q", want)
		}
	}
}

func TestAOMissionHelpListsRecoveryAndCompactionInputs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"ao-mission", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("ao-mission help failed: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	help := stdout.String()
	for _, want := range []string{
		"[--scheduler-recovery <readback.json>]",
		"[--ledger-compaction <readback.json>]",
		"[--timeline-compaction <readback.json>]",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("ao-mission help missing %q:\n%s", want, help)
		}
	}
}

func TestAOMissionE2ESmokeBindsMissionAtlasAndFoundryArtifacts(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "ao-mission-e2e-smoke.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"ao-mission", "e2e-smoke",
		"--route", repoPath(filepath.Join("examples", "ao-mission-smoke", "mission-route-readback.json")),
		"--snapshot", repoPath(filepath.Join("examples", "ao-mission-smoke", "governance-snapshot-readback.json")),
		"--mission-final-rollup", repoPath(filepath.Join("examples", "ao-mission-smoke", "mission-final-rollup.json")),
		"--foundry-final-rollup", repoPath(filepath.Join("examples", "ao-mission-smoke", "foundry-final-rollup.json")),
		"--atlas-metadata", repoPath(filepath.Join("examples", "ao-mission-smoke", "atlas-workgraph-metadata.json")),
		"--scheduler-recovery", repoPath(filepath.Join("examples", "ao-mission-smoke", "scheduler-recovery-readback.json")),
		"--ledger-compaction", repoPath(filepath.Join("examples", "ao-mission-smoke", "ledger-compaction-readback.json")),
		"--timeline-compaction", repoPath(filepath.Join("examples", "ao-mission-smoke", "timeline-compaction-readback.json")),
		"--mission-archive-validation", repoPath(filepath.Join("examples", "ao-mission-smoke", "mission-archive-validation.json")),
		"--gateway-readiness-rollup", repoPath(filepath.Join("examples", "ao-mission-smoke", "gateway-readiness-rollup.json")),
		"--out", outPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("e2e smoke failed: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "ao_mission_e2e_smoke="+outPath) {
		t.Fatalf("expected e2e smoke output path, got %q", stdout.String())
	}
	var smoke map[string]any
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &smoke); err != nil {
		t.Fatal(err)
	}
	if smoke["schema"] != "ao.foundry.ao-mission-e2e-smoke.v0.1" || smoke["status"] != "ready" {
		t.Fatalf("bad e2e smoke: %#v", smoke)
	}
	if smoke["atlas_workgraph_id"] != "atlas-readiness-workgraph" || smoke["mission_id"] != "mission-demo" {
		t.Fatalf("bad mission/atlas binding: %#v", smoke)
	}
	if smoke["executes_work"] != false || smoke["approves_work"] != false || smoke["mutates_repositories"] != false {
		t.Fatalf("e2e smoke widened authority: %#v", smoke)
	}
	if smoke["scheduler_recovery_bound"] != true || smoke["ledger_compaction_bound"] != true || smoke["timeline_compaction_bound"] != true {
		t.Fatalf("e2e smoke did not bind recovery/compaction readbacks: %#v", smoke)
	}
	if smoke["mission_archive_validation_bound"] != true {
		t.Fatalf("e2e smoke did not bind archive validation readback: %#v", smoke)
	}
	if smoke["gateway_readiness_rollup_bound"] != true {
		t.Fatalf("e2e smoke did not bind gateway readiness rollup: %#v", smoke)
	}
	if smoke["atlas_source_artifact_count"] != float64(2) {
		t.Fatalf("e2e smoke missing Atlas source artifact count: %#v", smoke)
	}
	provenance, ok := smoke["mission_provenance"].(map[string]any)
	if !ok || provenance["scheduler_recovery"] != float64(1) || provenance["ledger_compaction"] != float64(1) || provenance["timeline_compaction"] != float64(1) {
		t.Fatalf("e2e smoke missing Mission provenance summary: %#v", smoke)
	}
	if smoke["primary_mission_provenance"] != "artifact_manifest" || !strings.Contains(fmt.Sprint(smoke["provenance_diagnostics"]), "route_history=1") {
		t.Fatalf("e2e smoke missing Mission provenance diagnostics: %#v", smoke)
	}
}

func TestAOMissionE2ESmokeRejectsUnsafeGatewayReadinessRollup(t *testing.T) {
	dir := t.TempDir()
	rollupPath := filepath.Join(dir, "gateway-readiness-rollup.json")
	if err := os.WriteFile(rollupPath, []byte(`{
  "schema": "ao.mission.gateway-readiness-rollup.v0.1",
  "mission_id": "mission-demo",
  "status": "ready",
  "safe_to_execute": true
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(dir, "ao-mission-e2e-smoke.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"ao-mission", "e2e-smoke",
		"--route", filepath.Join("examples", "ao-mission-smoke", "mission-route-readback.json"),
		"--snapshot", filepath.Join("examples", "ao-mission-smoke", "governance-snapshot-readback.json"),
		"--mission-final-rollup", filepath.Join("examples", "ao-mission-smoke", "mission-final-rollup.json"),
		"--foundry-final-rollup", filepath.Join("examples", "ao-mission-smoke", "foundry-final-rollup.json"),
		"--atlas-metadata", filepath.Join("examples", "ao-mission-smoke", "atlas-workgraph-metadata.json"),
		"--gateway-readiness-rollup", rollupPath,
		"--out", outPath,
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected unsafe gateway readiness rollup to fail")
	}
	if !strings.Contains(stderr.String(), "gateway_readiness_rollup must not claim authority") {
		t.Fatalf("expected gateway rollup authority error, got stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestAOMissionE2ESmokeRejectsUnsafeMissionArchiveValidationFixture(t *testing.T) {
	dir := t.TempDir()
	archiveValidationPath := filepath.Join(dir, "mission-archive-validation.json")
	if err := os.WriteFile(archiveValidationPath, []byte(`{
  "schema": "ao.mission.archive-validation.v0.1",
  "mission_id": "mission-demo",
  "status": "ready",
  "safe_to_execute": true
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(dir, "ao-mission-e2e-smoke.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"ao-mission", "e2e-smoke",
		"--route", filepath.Join("examples", "ao-mission-smoke", "mission-route-readback.json"),
		"--snapshot", filepath.Join("examples", "ao-mission-smoke", "governance-snapshot-readback.json"),
		"--mission-final-rollup", filepath.Join("examples", "ao-mission-smoke", "mission-final-rollup.json"),
		"--foundry-final-rollup", filepath.Join("examples", "ao-mission-smoke", "foundry-final-rollup.json"),
		"--atlas-metadata", filepath.Join("examples", "ao-mission-smoke", "atlas-workgraph-metadata.json"),
		"--mission-archive-validation", archiveValidationPath,
		"--out", outPath,
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected unsafe mission archive validation fixture to fail")
	}
	if !strings.Contains(stderr.String(), "mission_archive_validation must not claim authority") {
		t.Fatalf("expected archive validation authority error, got stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestAOMissionE2ESmokeRejectsUnsafeSchedulerRecoveryFixture(t *testing.T) {
	dir := t.TempDir()
	schedulerRecoveryPath := filepath.Join(dir, "scheduler-recovery-readback.json")
	if err := os.WriteFile(schedulerRecoveryPath, []byte(`{
  "schema": "ao.mission.scheduler-recovery-readback.v0.1",
  "mission_id": "mission-demo",
  "safe_to_execute": true
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(dir, "ao-mission-e2e-smoke.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"ao-mission", "e2e-smoke",
		"--route", filepath.Join("examples", "ao-mission-smoke", "mission-route-readback.json"),
		"--snapshot", filepath.Join("examples", "ao-mission-smoke", "governance-snapshot-readback.json"),
		"--mission-final-rollup", filepath.Join("examples", "ao-mission-smoke", "mission-final-rollup.json"),
		"--foundry-final-rollup", filepath.Join("examples", "ao-mission-smoke", "foundry-final-rollup.json"),
		"--atlas-metadata", filepath.Join("examples", "ao-mission-smoke", "atlas-workgraph-metadata.json"),
		"--scheduler-recovery", schedulerRecoveryPath,
		"--out", outPath,
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected unsafe scheduler recovery fixture to fail")
	}
	if !strings.Contains(stderr.String(), "scheduler_recovery must not claim authority") {
		t.Fatalf("expected scheduler recovery authority error, got stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestAOMissionE2ESmokeRejectsMissionIDMismatchFixture(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "ao-mission-e2e-smoke.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"ao-mission", "e2e-smoke",
		"--route", filepath.Join("examples", "ao-mission-smoke", "mission-route-readback.json"),
		"--snapshot", filepath.Join("examples", "ao-mission-smoke", "governance-snapshot-readback.json"),
		"--mission-final-rollup", filepath.Join("examples", "ao-mission-smoke", "invalid-mission-final-rollup-mission-id.json"),
		"--foundry-final-rollup", filepath.Join("examples", "ao-mission-smoke", "foundry-final-rollup.json"),
		"--atlas-metadata", filepath.Join("examples", "ao-mission-smoke", "atlas-workgraph-metadata.json"),
		"--out", outPath,
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected mission_id mismatch fixture to fail")
	}
	if !strings.Contains(stderr.String(), "mission_final_rollup mission_id mismatch") {
		t.Fatalf("expected mission_id mismatch error, got stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestAOMissionE2ESmokeRejectsAtlasNodeCountMismatchFixture(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "ao-mission-e2e-smoke.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"ao-mission", "e2e-smoke",
		"--route", filepath.Join("examples", "ao-mission-smoke", "mission-route-readback.json"),
		"--snapshot", filepath.Join("examples", "ao-mission-smoke", "governance-snapshot-readback.json"),
		"--mission-final-rollup", filepath.Join("examples", "ao-mission-smoke", "mission-final-rollup.json"),
		"--foundry-final-rollup", filepath.Join("examples", "ao-mission-smoke", "foundry-final-rollup.json"),
		"--atlas-metadata", filepath.Join("examples", "ao-mission-smoke", "invalid-atlas-workgraph-metadata-node-count.json"),
		"--out", outPath,
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected Atlas node count mismatch fixture to fail")
	}
	if !strings.Contains(stderr.String(), "Atlas metadata node total must match final rollup total") {
		t.Fatalf("expected Atlas node-count mismatch error, got stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestAOMissionE2ESmokeRejectsAtlasMetadataWithoutProvenanceDiagnostics(t *testing.T) {
	dir := t.TempDir()
	var metadata map[string]any
	body, err := os.ReadFile(filepath.Join("..", "..", "examples", "ao-mission-smoke", "atlas-workgraph-metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, &metadata); err != nil {
		t.Fatal(err)
	}
	delete(metadata, "primary_mission_provenance")
	delete(metadata, "provenance_diagnostics")
	metadataPath := filepath.Join(dir, "atlas-workgraph-metadata-missing-provenance.json")
	writeJSONFixtureForTest(t, metadataPath, metadata)
	outPath := filepath.Join(dir, "ao-mission-e2e-smoke.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"ao-mission", "e2e-smoke",
		"--route", filepath.Join("examples", "ao-mission-smoke", "mission-route-readback.json"),
		"--snapshot", filepath.Join("examples", "ao-mission-smoke", "governance-snapshot-readback.json"),
		"--mission-final-rollup", filepath.Join("examples", "ao-mission-smoke", "mission-final-rollup.json"),
		"--foundry-final-rollup", filepath.Join("examples", "ao-mission-smoke", "foundry-final-rollup.json"),
		"--atlas-metadata", metadataPath,
		"--out", outPath,
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected missing provenance diagnostics to fail")
	}
	if !strings.Contains(stderr.String(), "Atlas metadata requires Mission provenance diagnostics") {
		t.Fatalf("expected provenance diagnostics error, got stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestAOMissionE2ESmokeValidatesArtifactManifestDigests(t *testing.T) {
	manifestPath := writeAOMissionArtifactManifestFixture(t, digestOverride{})
	outPath := filepath.Join(t.TempDir(), "ao-mission-e2e-smoke.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"ao-mission", "e2e-smoke",
		"--route", filepath.Join("examples", "ao-mission-smoke", "mission-route-readback.json"),
		"--snapshot", filepath.Join("examples", "ao-mission-smoke", "governance-snapshot-readback.json"),
		"--mission-final-rollup", filepath.Join("examples", "ao-mission-smoke", "mission-final-rollup.json"),
		"--foundry-final-rollup", filepath.Join("examples", "ao-mission-smoke", "foundry-final-rollup.json"),
		"--atlas-metadata", filepath.Join("examples", "ao-mission-smoke", "atlas-workgraph-metadata.json"),
		"--artifact-manifest", manifestPath,
		"--out", outPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("e2e smoke with artifact manifest failed: %s", stderr.String())
	}
	var smoke map[string]any
	body, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, &smoke); err != nil {
		t.Fatal(err)
	}
	if smoke["artifact_manifest"] != manifestPath || smoke["executes_work"] != false {
		t.Fatalf("bad e2e smoke artifact manifest binding: %#v", smoke)
	}
}

func TestAOMissionE2ESmokeRejectsArtifactManifestDigestMismatch(t *testing.T) {
	manifestPath := writeAOMissionArtifactManifestFixture(t, digestOverride{
		Path:   repoPath(filepath.Join("examples", "ao-mission-smoke", "mission-route-readback.json")),
		Digest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	})
	outPath := filepath.Join(t.TempDir(), "ao-mission-e2e-smoke.json")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"ao-mission", "e2e-smoke",
		"--route", repoPath(filepath.Join("examples", "ao-mission-smoke", "mission-route-readback.json")),
		"--snapshot", repoPath(filepath.Join("examples", "ao-mission-smoke", "governance-snapshot-readback.json")),
		"--mission-final-rollup", repoPath(filepath.Join("examples", "ao-mission-smoke", "mission-final-rollup.json")),
		"--foundry-final-rollup", repoPath(filepath.Join("examples", "ao-mission-smoke", "foundry-final-rollup.json")),
		"--atlas-metadata", repoPath(filepath.Join("examples", "ao-mission-smoke", "atlas-workgraph-metadata.json")),
		"--artifact-manifest", manifestPath,
		"--out", outPath,
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected artifact manifest digest mismatch to fail")
	}
	if !strings.Contains(stderr.String(), "digest mismatch") {
		t.Fatalf("expected digest mismatch error, got stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

type digestOverride struct {
	Path   string
	Digest string
}

func writeAOMissionArtifactManifestFixture(t *testing.T, override digestOverride) string {
	t.Helper()
	artifactPaths := []string{
		repoPath(filepath.Join("examples", "ao-mission-smoke", "mission-route-readback.json")),
		repoPath(filepath.Join("examples", "ao-mission-smoke", "governance-snapshot-readback.json")),
		repoPath(filepath.Join("examples", "ao-mission-smoke", "atlas-workgraph-metadata.json")),
	}
	refs := make([]map[string]any, 0, len(artifactPaths))
	for _, path := range artifactPaths {
		digest, err := digestPath(path)
		if err != nil {
			t.Fatal(err)
		}
		if override.Path == path {
			digest = override.Digest
		}
		refs = append(refs, map[string]any{
			"schema": "ao.mission.artifact-ref.v0.1",
			"ref":    path,
			"digest": digest,
			"kind":   "readback",
		})
	}
	manifest := map[string]any{
		"schema":                "ao.mission.artifact-manifest.v0.1",
		"mission_id":            "mission-demo",
		"status":                "ready",
		"operator_mode":         "read_only",
		"artifact_refs":         refs,
		"safe_to_execute":       false,
		"executes_work":         false,
		"approves_work":         false,
		"mutates_repositories":  false,
		"exact_next_action":     "AO Foundry validates manifest digests before implementation handoff",
		"manifest_digest":       "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"generated_at_utc":      "2026-07-03T00:00:00Z",
		"mutation_authority":    false,
		"gateway_authority":     "intent_readback_only",
		"scheduler_authority":   "wakeup_adapter_only",
		"release_or_publish":    false,
		"provider_calls":        false,
		"credential_use":        false,
		"direct_main_mutation":  false,
		"concurrent_mutation":   false,
		"dependency_updates":    false,
		"policy_auth_expansion": false,
	}
	path := filepath.Join(t.TempDir(), "artifact-manifest.json")
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(body, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
func TestCIWorkflowRunsAOMissionE2ESmokeFixture(t *testing.T) {
	workflow, err := os.ReadFile(repoPath(".github/workflows/ci.yml"))
	if err != nil {
		t.Fatalf("read CI workflow: %v", err)
	}
	text := string(workflow)
	for _, want := range []string{
		"AO Mission e2e smoke",
		"shell: bash",
		"scripts/ao-mission-atlas-foundry-e2e-smoke.sh tmp/ao-mission-atlas-foundry-e2e-ci",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("CI workflow missing AO Mission e2e smoke detail %q", want)
		}
	}
}
