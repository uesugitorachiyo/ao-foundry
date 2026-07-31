package cli

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func runAOMission(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "foundry ao-mission requires smoke or final-rollup-smoke")
		return 2
	}
	if args[0] == "--help" || args[0] == "help" {
		printHelp(stdout)
		return 0
	}
	switch args[0] {
	case "e2e-smoke":
		return runAOMissionE2ESmoke(args[1:], stdout, stderr)
	case "final-rollup-smoke":
		return runAOMissionFinalRollupSmoke(args[1:], stdout, stderr)
	case "readiness-ledger":
		return runAOMissionReadinessLedger(args[1:], stdout, stderr)
	case "rollup-summary":
		return runAOMissionRollupSummary(args[1:], stdout, stderr)
	case "smoke":
		return runAOMissionSmoke(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "foundry ao-mission: unknown command %q\n", strings.Join(args, " "))
		return 2
	}
}

func runAOMissionSmoke(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ao-mission smoke", flag.ContinueOnError)
	fs.SetOutput(stderr)
	routePath := fs.String("route", "", "ao-mission route readback fixture")
	snapshotPath := fs.String("snapshot", "", "ao-mission governance snapshot fixture")
	outPath := fs.String("out", "", "smoke readback output path")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *routePath == "" || *snapshotPath == "" || *outPath == "" {
		fmt.Fprintln(stderr, "ao-mission smoke requires --route, --snapshot, and --out")
		return 2
	}
	var route map[string]any
	if err := readJSONFile(*routePath, &route); err != nil {
		fmt.Fprintf(stderr, "ao-mission smoke: read route: %v\n", err)
		return 1
	}
	var snapshot map[string]any
	if err := readJSONFile(*snapshotPath, &snapshot); err != nil {
		fmt.Fprintf(stderr, "ao-mission smoke: read snapshot: %v\n", err)
		return 1
	}
	for _, field := range []string{"safe_to_execute", "executes_work", "approves_work", "mutates_repositories"} {
		if route[field] != false {
			fmt.Fprintf(stderr, "ao-mission smoke: route %s must be false\n", field)
			return 1
		}
		if snapshot[field] != false {
			fmt.Fprintf(stderr, "ao-mission smoke: snapshot %s must be false\n", field)
			return 1
		}
	}
	if route["status"] != "ready" || snapshot["status"] != "ready" {
		fmt.Fprintln(stderr, "ao-mission smoke: route and snapshot status must be ready")
		return 1
	}
	readback := map[string]any{
		"schema":                      "ao.foundry.ao-mission-smoke-readback.v0.1",
		"status":                      "ready",
		"mission_id":                  route["mission_id"],
		"route":                       route["route"],
		"current_owner":               snapshot["current_owner"],
		"current_phase":               snapshot["current_phase"],
		"safe_to_execute":             false,
		"executes_work":               false,
		"approves_work":               false,
		"mutates_repositories":        false,
		"provider_calls":              false,
		"credential_use":              false,
		"exact_next_action":           "ao-mission fixtures validated; use Atlas/Foundry gates for implementation work",
		"route_readback":              *routePath,
		"governance_snapshot":         *snapshotPath,
		"generated_at_utc":            time.Now().UTC().Format(time.RFC3339),
		"mutation_authority":          false,
		"gateway_authority":           "intent_readback_only",
		"scheduler_authority":         "wakeup_adapter_only",
		"public_safe_readback":        true,
		"direct_main_mutation":        false,
		"concurrent_mutation":         false,
		"release_or_publish":          false,
		"dependency_updates":          false,
		"policy_auth_expansion":       false,
		"hidden_instruction_mutation": false,
	}
	if err := writeJSONFile(*outPath, readback); err != nil {
		fmt.Fprintf(stderr, "ao-mission smoke: write output: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "ao_mission_smoke=%s\n", *outPath)
	return 0
}

func runAOMissionFinalRollupSmoke(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ao-mission final-rollup-smoke", flag.ContinueOnError)
	fs.SetOutput(stderr)
	missionRollupPath := fs.String("mission-final-rollup", "", "ao-mission final rollup fixture")
	foundryRollupPath := fs.String("foundry-final-rollup", "", "ao-foundry final rollup fixture")
	gatewayReadinessRollupPath := fs.String("gateway-readiness-rollup", "", "optional ao-mission gateway readiness rollup fixture")
	outPath := fs.String("out", "", "smoke readback output path")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *missionRollupPath == "" || *foundryRollupPath == "" || *outPath == "" {
		fmt.Fprintln(stderr, "ao-mission final-rollup-smoke requires --mission-final-rollup, --foundry-final-rollup, and --out")
		return 2
	}
	var missionRollup map[string]any
	if err := readJSONFile(*missionRollupPath, &missionRollup); err != nil {
		fmt.Fprintf(stderr, "ao-mission final-rollup-smoke: read mission rollup: %v\n", err)
		return 1
	}
	var foundryRollup map[string]any
	if err := readJSONFile(*foundryRollupPath, &foundryRollup); err != nil {
		fmt.Fprintf(stderr, "ao-mission final-rollup-smoke: read foundry rollup: %v\n", err)
		return 1
	}
	if missionRollup["mission_id"] != foundryRollup["mission_id"] {
		fmt.Fprintln(stderr, "ao-mission final-rollup-smoke: mission_id mismatch")
		return 1
	}
	var gatewayReadinessRollup map[string]any
	if *gatewayReadinessRollupPath != "" {
		if err := readJSONFile(*gatewayReadinessRollupPath, &gatewayReadinessRollup); err != nil {
			fmt.Fprintf(stderr, "ao-mission final-rollup-smoke: read gateway readiness rollup: %v\n", err)
			return 1
		}
		if gatewayReadinessRollup["schema"] != "ao.mission.gateway-readiness-rollup.v0.1" || gatewayReadinessRollup["status"] != "ready" {
			fmt.Fprintln(stderr, "ao-mission final-rollup-smoke: gateway readiness rollup must be ready")
			return 1
		}
		if gatewayReadinessRollup["mission_id"] != nil && gatewayReadinessRollup["mission_id"] != missionRollup["mission_id"] {
			fmt.Fprintln(stderr, "ao-mission final-rollup-smoke: gateway readiness rollup mission_id mismatch")
			return 1
		}
		if aoMissionAuthorityClaimed(gatewayReadinessRollup) {
			fmt.Fprintln(stderr, "ao-mission final-rollup-smoke: gateway readiness rollup must not claim authority")
			return 1
		}
	}
	for _, field := range []string{"safe_to_execute", "executes_work", "approves_work", "mutates_repositories"} {
		if missionRollup[field] != false {
			fmt.Fprintf(stderr, "ao-mission final-rollup-smoke: mission rollup %s must be false\n", field)
			return 1
		}
		if foundryRollup[field] != false {
			fmt.Fprintf(stderr, "ao-mission final-rollup-smoke: foundry rollup %s must be false\n", field)
			return 1
		}
	}
	missionCompleted, missionTotal := intFromJSONNumber(missionRollup["completed_nodes"]), intFromJSONNumber(missionRollup["total_nodes"])
	foundryCompleted, foundryTotal := intFromJSONNumber(foundryRollup["completed_nodes"]), intFromJSONNumber(foundryRollup["total_nodes"])
	foundryTerminalStatus := normalizeAOMissionTerminalStatus(stringFromJSONValue(foundryRollup["status"]))
	switch foundryTerminalStatus {
	case "completed", "promoted":
		if missionCompleted == 0 || missionTotal == 0 || missionCompleted != missionTotal || foundryCompleted != foundryTotal || missionCompleted != foundryCompleted || missionTotal != foundryTotal {
			fmt.Fprintln(stderr, "ao-mission final-rollup-smoke: completed and total node counts must match and be complete")
			return 1
		}
	case "denied", "blocked":
		if missionCompleted == 0 || missionTotal == 0 || foundryCompleted != missionCompleted || foundryTotal != missionTotal {
			fmt.Fprintln(stderr, "ao-mission final-rollup-smoke: denied or blocked terminal rollups must carry matching node counts")
			return 1
		}
	default:
		fmt.Fprintln(stderr, "ao-mission final-rollup-smoke: completed and total node counts must match and be complete")
		return 1
	}
	readbackStatus := "ready"
	exactNextAction := "AO Mission and AO Foundry final rollups agree; no execution authority is granted"
	claimsCompletionOnly := true
	if foundryTerminalStatus == "denied" || foundryTerminalStatus == "blocked" {
		readbackStatus = "blocked"
		exactNextAction = "AO Mission and AO Foundry terminal rollups agree on " + foundryTerminalStatus + "; route repair or support work through AO Atlas before retry"
		claimsCompletionOnly = false
	}
	terminalReadinessSummary := buildAOMissionTerminalReadinessSummary(foundryTerminalStatus, readbackStatus, missionCompleted, missionTotal, exactNextAction)
	readback := map[string]any{
		"schema":                         "ao.foundry.ao-mission-final-rollup-smoke.v0.1",
		"status":                         readbackStatus,
		"mission_id":                     missionRollup["mission_id"],
		"completed_nodes":                missionCompleted,
		"total_nodes":                    missionTotal,
		"foundry_terminal_status":        foundryTerminalStatus,
		"terminal_readiness_summary":     terminalReadinessSummary,
		"safe_to_execute":                false,
		"executes_work":                  false,
		"approves_work":                  false,
		"mutates_repositories":           false,
		"mission_final_rollup":           *missionRollupPath,
		"foundry_final_rollup":           *foundryRollupPath,
		"gateway_readiness_rollup":       *gatewayReadinessRollupPath,
		"gateway_readiness_rollup_bound": *gatewayReadinessRollupPath != "",
		"exact_next_action":              exactNextAction,
		"generated_at_utc":               time.Now().UTC().Format(time.RFC3339),
		"public_safe_readback":           true,
		"mutation_authority":             false,
		"scheduler_authority":            "none",
		"gateway_authority":              "none",
		"direct_main_mutation":           false,
		"concurrent_mutation":            false,
		"release_or_publish":             false,
		"dependency_updates":             false,
		"policy_auth_expansion":          false,
		"provider_calls":                 false,
		"credential_use":                 false,
		"claims_completion_only":         claimsCompletionOnly,
	}
	if value, ok := gatewayReadinessRollup["correlation_id"].(string); ok && strings.TrimSpace(value) != "" {
		readback["correlation_id"] = strings.TrimSpace(value)
	}
	if err := writeJSONFile(*outPath, readback); err != nil {
		fmt.Fprintf(stderr, "ao-mission final-rollup-smoke: write output: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "ao_mission_final_rollup_smoke=%s\n", *outPath)
	return 0
}

func runAOMissionReadinessLedger(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ao-mission readiness-ledger", flag.ContinueOnError)
	fs.SetOutput(stderr)
	smokePath := fs.String("final-rollup-smoke", "", "ao-mission final-rollup smoke readback")
	outPath := fs.String("out", "", "readiness ledger output path")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *smokePath == "" || *outPath == "" {
		fmt.Fprintln(stderr, "ao-mission readiness-ledger requires --final-rollup-smoke and --out")
		return 2
	}
	var smoke map[string]any
	if err := readJSONFile(*smokePath, &smoke); err != nil {
		fmt.Fprintf(stderr, "ao-mission readiness-ledger: read final-rollup smoke: %v\n", err)
		return 1
	}
	smokeStatus := stringFromJSONValue(smoke["status"])
	foundryTerminalStatus := normalizeAOMissionTerminalStatus(stringFromJSONValue(smoke["foundry_terminal_status"]))
	if smoke["schema"] != "ao.foundry.ao-mission-final-rollup-smoke.v0.1" || !aoMissionTerminalSmokeStatusAllowed(smokeStatus, foundryTerminalStatus) {
		fmt.Fprintln(stderr, "ao-mission readiness-ledger: final-rollup smoke must be ready or blocked with denied terminal status")
		return 1
	}
	if aoMissionAuthorityClaimed(smoke) {
		fmt.Fprintln(stderr, "ao-mission readiness-ledger: final-rollup smoke must not claim authority")
		return 1
	}
	completedNodes := intFromJSONNumber(smoke["completed_nodes"])
	totalNodes := intFromJSONNumber(smoke["total_nodes"])
	exactNextAction := "AO Mission final-rollup smoke is ready; keep execution behind Atlas and Foundry gates"
	if !aoMissionTerminalCanClose(foundryTerminalStatus) {
		exactNextAction = stringFromJSONValue(smoke["exact_next_action"])
	}
	terminalReadinessSummary := buildAOMissionTerminalReadinessSummary(foundryTerminalStatus, smokeStatus, completedNodes, totalNodes, exactNextAction)
	ledger := map[string]any{
		"schema":                           "ao.foundry.ao-mission-readiness-ledger.v0.1",
		"status":                           smokeStatus,
		"mission_id":                       smoke["mission_id"],
		"completed_nodes":                  completedNodes,
		"total_nodes":                      totalNodes,
		"foundry_terminal_status":          foundryTerminalStatus,
		"terminal_readiness_summary":       terminalReadinessSummary,
		"terminal_readiness_summary_bound": true,
		"final_rollup_smoke":               *smokePath,
		"safe_to_execute":                  false,
		"executes_work":                    false,
		"approves_work":                    false,
		"mutates_repositories":             false,
		"exact_next_action":                exactNextAction,
		"generated_at_utc":                 time.Now().UTC().Format(time.RFC3339),
		"public_safe_readback":             true,
		"claims_readiness_only":            aoMissionTerminalCanClose(foundryTerminalStatus),
		"claims_terminal_readback_only":    true,
		"scheduler_authority":              "none",
		"gateway_authority":                "none",
		"direct_main_mutation":             false,
		"concurrent_mutation":              false,
		"release_or_publish":               false,
		"dependency_updates":               false,
		"policy_auth_expansion":            false,
		"provider_calls":                   false,
		"credential_use":                   false,
	}
	if err := writeJSONFile(*outPath, ledger); err != nil {
		fmt.Fprintf(stderr, "ao-mission readiness-ledger: write output: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "ao_mission_readiness_ledger=%s\n", *outPath)
	return 0
}

func runAOMissionRollupSummary(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ao-mission rollup-summary", flag.ContinueOnError)
	fs.SetOutput(stderr)
	smokePath := fs.String("final-rollup-smoke", "", "ao-mission final-rollup smoke readback")
	ledgerPath := fs.String("readiness-ledger", "", "ao-mission readiness ledger")
	outPath := fs.String("out", "", "rollup summary output path")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *smokePath == "" || *ledgerPath == "" || *outPath == "" {
		fmt.Fprintln(stderr, "ao-mission rollup-summary requires --final-rollup-smoke, --readiness-ledger, and --out")
		return 2
	}
	var smoke map[string]any
	if err := readJSONFile(*smokePath, &smoke); err != nil {
		fmt.Fprintf(stderr, "ao-mission rollup-summary: read final-rollup smoke: %v\n", err)
		return 1
	}
	smokeStatus := stringFromJSONValue(smoke["status"])
	foundryTerminalStatus := normalizeAOMissionTerminalStatus(stringFromJSONValue(smoke["foundry_terminal_status"]))
	if smoke["schema"] != "ao.foundry.ao-mission-final-rollup-smoke.v0.1" || !aoMissionTerminalSmokeStatusAllowed(smokeStatus, foundryTerminalStatus) {
		fmt.Fprintln(stderr, "ao-mission rollup-summary: final-rollup smoke must be ready or blocked with denied terminal status")
		return 1
	}
	var ledger map[string]any
	if err := readJSONFile(*ledgerPath, &ledger); err != nil {
		fmt.Fprintf(stderr, "ao-mission rollup-summary: read readiness ledger: %v\n", err)
		return 1
	}
	ledgerStatus := stringFromJSONValue(ledger["status"])
	if ledger["schema"] != "ao.foundry.ao-mission-readiness-ledger.v0.1" || !aoMissionTerminalSmokeStatusAllowed(ledgerStatus, foundryTerminalStatus) {
		fmt.Fprintln(stderr, "ao-mission rollup-summary: readiness ledger must be ready or blocked with denied terminal status")
		return 1
	}
	if ledgerStatus != smokeStatus {
		fmt.Fprintln(stderr, "ao-mission rollup-summary: readiness ledger status must match final-rollup smoke status")
		return 1
	}
	if smoke["mission_id"] != ledger["mission_id"] {
		fmt.Fprintln(stderr, "ao-mission rollup-summary: mission_id mismatch")
		return 1
	}
	if aoMissionAuthorityClaimed(smoke) || aoMissionAuthorityClaimed(ledger) {
		fmt.Fprintln(stderr, "ao-mission rollup-summary: inputs must not claim authority")
		return 1
	}
	smokeCompleted, smokeTotal := intFromJSONNumber(smoke["completed_nodes"]), intFromJSONNumber(smoke["total_nodes"])
	ledgerCompleted, ledgerTotal := intFromJSONNumber(ledger["completed_nodes"]), intFromJSONNumber(ledger["total_nodes"])
	if smokeCompleted == 0 || smokeTotal == 0 || ledgerCompleted != smokeCompleted || ledgerTotal != smokeTotal {
		fmt.Fprintln(stderr, "ao-mission rollup-summary: completed and total node counts must match")
		return 1
	}
	if aoMissionTerminalCanClose(foundryTerminalStatus) && smokeCompleted != smokeTotal {
		fmt.Fprintln(stderr, "ao-mission rollup-summary: promoted or completed terminal rollups must be complete")
		return 1
	}
	exactNextAction := "AO Mission rollup is bound to active-stack readiness; keep execution behind Foundry gates"
	if !aoMissionTerminalCanClose(foundryTerminalStatus) {
		exactNextAction = stringFromJSONValue(smoke["exact_next_action"])
	}
	terminalReadinessSummary := buildAOMissionTerminalReadinessSummary(foundryTerminalStatus, smokeStatus, smokeCompleted, smokeTotal, exactNextAction)
	summary := map[string]any{
		"schema":                           "ao.foundry.ao-mission-rollup-summary.v0.1",
		"status":                           smokeStatus,
		"mission_id":                       smoke["mission_id"],
		"portfolio_binding":                "active_stack_readiness",
		"readiness_bound":                  true,
		"final_rollup_smoke_bound":         true,
		"readiness_ledger_bound":           true,
		"completed_nodes":                  smokeCompleted,
		"total_nodes":                      smokeTotal,
		"foundry_terminal_status":          foundryTerminalStatus,
		"terminal_readiness_summary":       terminalReadinessSummary,
		"terminal_readiness_summary_bound": true,
		"final_rollup_smoke":               *smokePath,
		"readiness_ledger":                 *ledgerPath,
		"safe_to_execute":                  false,
		"executes_work":                    false,
		"approves_work":                    false,
		"mutates_repositories":             false,
		"provider_calls":                   false,
		"credential_use":                   false,
		"direct_main_mutation":             false,
		"concurrent_mutation":              false,
		"release_or_publish":               false,
		"dependency_updates":               false,
		"policy_auth_expansion":            false,
		"hidden_instruction_mutation":      false,
		"claims_readiness_only":            aoMissionTerminalCanClose(foundryTerminalStatus),
		"claims_terminal_readback_only":    true,
		"public_safe_readback":             true,
		"scheduler_authority":              "none",
		"gateway_authority":                "none",
		"generated_at_utc":                 time.Now().UTC().Format(time.RFC3339),
		"exact_next_action":                exactNextAction,
	}
	if err := writeJSONFile(*outPath, summary); err != nil {
		fmt.Fprintf(stderr, "ao-mission rollup-summary: write output: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "ao_mission_rollup_summary=%s\n", *outPath)
	return 0
}

func runAOMissionE2ESmoke(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ao-mission e2e-smoke", flag.ContinueOnError)
	fs.SetOutput(stderr)
	routePath := fs.String("route", "", "ao-mission route readback fixture")
	snapshotPath := fs.String("snapshot", "", "ao-mission governance snapshot fixture")
	missionRollupPath := fs.String("mission-final-rollup", "", "ao-mission final rollup fixture")
	foundryRollupPath := fs.String("foundry-final-rollup", "", "ao-foundry final rollup fixture")
	atlasMetadataPath := fs.String("atlas-metadata", "", "ao-atlas mission workgraph metadata fixture")
	artifactManifestPath := fs.String("artifact-manifest", "", "optional ao-mission artifact manifest fixture")
	schedulerRecoveryPath := fs.String("scheduler-recovery", "", "optional ao-mission scheduler recovery readback fixture")
	ledgerCompactionPath := fs.String("ledger-compaction", "", "optional ao-mission ledger compaction readback fixture")
	timelineCompactionPath := fs.String("timeline-compaction", "", "optional ao-mission timeline compaction readback fixture")
	missionArchiveValidationPath := fs.String("mission-archive-validation", "", "optional ao-mission archive validation fixture")
	gatewayReadinessRollupPath := fs.String("gateway-readiness-rollup", "", "optional ao-mission gateway readiness rollup fixture")
	outPath := fs.String("out", "", "e2e smoke output path")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *routePath == "" || *snapshotPath == "" || *missionRollupPath == "" || *foundryRollupPath == "" || *atlasMetadataPath == "" || *outPath == "" {
		fmt.Fprintln(stderr, "ao-mission e2e-smoke requires --route, --snapshot, --mission-final-rollup, --foundry-final-rollup, --atlas-metadata, and --out")
		return 2
	}
	artifacts := map[string]map[string]any{}
	for name, path := range map[string]string{
		"route":                *routePath,
		"snapshot":             *snapshotPath,
		"mission_final_rollup": *missionRollupPath,
		"foundry_final_rollup": *foundryRollupPath,
		"atlas_workgraph_meta": *atlasMetadataPath,
	} {
		var doc map[string]any
		if err := readJSONFile(path, &doc); err != nil {
			fmt.Fprintf(stderr, "ao-mission e2e-smoke: read %s: %v\n", name, err)
			return 1
		}
		if aoMissionAuthorityClaimed(doc) {
			fmt.Fprintf(stderr, "ao-mission e2e-smoke: %s must not claim authority\n", name)
			return 1
		}
		artifacts[name] = doc
	}
	missionID := artifacts["route"]["mission_id"]
	for name, doc := range artifacts {
		if doc["mission_id"] != missionID {
			fmt.Fprintf(stderr, "ao-mission e2e-smoke: %s mission_id mismatch\n", name)
			return 1
		}
	}
	if artifacts["route"]["status"] != "ready" || artifacts["snapshot"]["status"] != "ready" {
		fmt.Fprintln(stderr, "ao-mission e2e-smoke: route and snapshot must be ready")
		return 1
	}
	if *artifactManifestPath != "" {
		var manifest map[string]any
		if err := readJSONFile(*artifactManifestPath, &manifest); err != nil {
			fmt.Fprintf(stderr, "ao-mission e2e-smoke: read artifact manifest: %v\n", err)
			return 1
		}
		if aoMissionAuthorityClaimed(manifest) {
			fmt.Fprintln(stderr, "ao-mission e2e-smoke: artifact manifest must not claim authority")
			return 1
		}
		if manifest["mission_id"] != missionID {
			fmt.Fprintln(stderr, "ao-mission e2e-smoke: artifact manifest mission_id mismatch")
			return 1
		}
		if err := validateAOMissionArtifactManifestDigests(*artifactManifestPath, manifest); err != nil {
			fmt.Fprintf(stderr, "ao-mission e2e-smoke: %v\n", err)
			return 1
		}
		artifacts["artifact_manifest"] = manifest
	}
	for name, input := range map[string]struct {
		path   string
		schema string
	}{
		"scheduler_recovery":         {path: *schedulerRecoveryPath, schema: "ao.mission.scheduler-recovery-readback.v0.1"},
		"ledger_compaction":          {path: *ledgerCompactionPath, schema: "ao.mission.ledger-compaction-readback.v0.1"},
		"timeline_compaction":        {path: *timelineCompactionPath, schema: "ao.mission.timeline-compaction-readback.v0.1"},
		"mission_archive_validation": {path: *missionArchiveValidationPath, schema: "ao.mission.archive-validation.v0.1"},
		"gateway_readiness_rollup":   {path: *gatewayReadinessRollupPath, schema: "ao.mission.gateway-readiness-rollup.v0.1"},
	} {
		if input.path == "" {
			continue
		}
		var doc map[string]any
		if err := readJSONFile(input.path, &doc); err != nil {
			fmt.Fprintf(stderr, "ao-mission e2e-smoke: read %s: %v\n", name, err)
			return 1
		}
		if doc["schema"] != input.schema {
			fmt.Fprintf(stderr, "ao-mission e2e-smoke: %s schema mismatch\n", name)
			return 1
		}
		if aoMissionAuthorityClaimed(doc) {
			fmt.Fprintf(stderr, "ao-mission e2e-smoke: %s must not claim authority\n", name)
			return 1
		}
		if doc["mission_id"] != missionID {
			fmt.Fprintf(stderr, "ao-mission e2e-smoke: %s mission_id mismatch\n", name)
			return 1
		}
		artifacts[name] = doc
	}
	missionCompleted, missionTotal := intFromJSONNumber(artifacts["mission_final_rollup"]["completed_nodes"]), intFromJSONNumber(artifacts["mission_final_rollup"]["total_nodes"])
	foundryCompleted, foundryTotal := intFromJSONNumber(artifacts["foundry_final_rollup"]["completed_nodes"]), intFromJSONNumber(artifacts["foundry_final_rollup"]["total_nodes"])
	if missionCompleted == 0 || missionTotal == 0 || missionCompleted != missionTotal || foundryCompleted != foundryTotal || missionCompleted != foundryCompleted || missionTotal != foundryTotal {
		fmt.Fprintln(stderr, "ao-mission e2e-smoke: final rollup node counts must be complete and equal")
		return 1
	}
	atlasCounts, _ := artifacts["atlas_workgraph_meta"]["node_counts"].(map[string]any)
	if atlasTotal := intFromJSONNumber(atlasCounts["total"]); atlasTotal != missionTotal {
		fmt.Fprintln(stderr, "ao-mission e2e-smoke: Atlas metadata node total must match final rollup total")
		return 1
	}
	primaryProvenance, _ := artifacts["atlas_workgraph_meta"]["primary_mission_provenance"].(string)
	provenanceDiagnostics, _ := artifacts["atlas_workgraph_meta"]["provenance_diagnostics"].(string)
	if strings.TrimSpace(primaryProvenance) == "" || strings.TrimSpace(provenanceDiagnostics) == "" {
		fmt.Fprintln(stderr, "ao-mission e2e-smoke: Atlas metadata requires Mission provenance diagnostics")
		return 1
	}
	atlasSourceArtifacts, _ := artifacts["atlas_workgraph_meta"]["source_artifacts"].(map[string]any)
	missionProvenance, _ := artifacts["atlas_workgraph_meta"]["mission_provenance"].(map[string]any)
	smoke := map[string]any{
		"schema":                           "ao.foundry.ao-mission-e2e-smoke.v0.1",
		"status":                           "ready",
		"mission_id":                       missionID,
		"route":                            artifacts["route"]["route"],
		"current_owner":                    artifacts["snapshot"]["current_owner"],
		"atlas_workgraph_id":               artifacts["atlas_workgraph_meta"]["workgraph_id"],
		"target_instance":                  artifacts["atlas_workgraph_meta"]["target_instance"],
		"completed_nodes":                  missionCompleted,
		"total_nodes":                      missionTotal,
		"route_readback":                   *routePath,
		"governance_snapshot":              *snapshotPath,
		"mission_final_rollup":             *missionRollupPath,
		"foundry_final_rollup":             *foundryRollupPath,
		"atlas_metadata":                   *atlasMetadataPath,
		"atlas_source_artifact_count":      len(atlasSourceArtifacts),
		"mission_provenance":               missionProvenance,
		"primary_mission_provenance":       artifacts["atlas_workgraph_meta"]["primary_mission_provenance"],
		"provenance_diagnostics":           artifacts["atlas_workgraph_meta"]["provenance_diagnostics"],
		"artifact_manifest":                *artifactManifestPath,
		"scheduler_recovery":               *schedulerRecoveryPath,
		"ledger_compaction":                *ledgerCompactionPath,
		"timeline_compaction":              *timelineCompactionPath,
		"mission_archive_validation":       *missionArchiveValidationPath,
		"gateway_readiness_rollup":         *gatewayReadinessRollupPath,
		"scheduler_recovery_bound":         *schedulerRecoveryPath != "",
		"ledger_compaction_bound":          *ledgerCompactionPath != "",
		"timeline_compaction_bound":        *timelineCompactionPath != "",
		"mission_archive_validation_bound": *missionArchiveValidationPath != "",
		"gateway_readiness_rollup_bound":   *gatewayReadinessRollupPath != "",
		"safe_to_execute":                  false,
		"executes_work":                    false,
		"approves_work":                    false,
		"mutates_repositories":             false,
		"exact_next_action":                "AO Mission, Atlas, and Foundry smoke artifacts agree; no execution authority is granted",
		"generated_at_utc":                 time.Now().UTC().Format(time.RFC3339),
		"public_safe_readback":             true,
		"gateway_authority":                "intent_readback_only",
		"scheduler_authority":              "wakeup_adapter_only",
		"ledger_compaction_authority":      "readback_only",
		"direct_main_mutation":             false,
		"concurrent_mutation":              false,
		"release_or_publish":               false,
		"dependency_updates":               false,
		"policy_auth_expansion":            false,
		"provider_calls":                   false,
		"credential_use":                   false,
	}
	if err := writeJSONFile(*outPath, smoke); err != nil {
		fmt.Fprintf(stderr, "ao-mission e2e-smoke: write output: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "ao_mission_e2e_smoke=%s\n", *outPath)
	return 0
}

func aoMissionAuthorityClaimed(doc map[string]any) bool {
	for _, field := range []string{
		"safe_to_execute",
		"executes_work",
		"approves_work",
		"mutates_repositories",
		"provider_calls",
		"credential_use",
		"release_or_publish",
		"direct_main_mutation",
		"concurrent_mutation",
		"dependency_updates",
		"policy_auth_expansion",
		"hidden_instruction_mutation",
		"schedules_work",
	} {
		if doc[field] == true {
			return true
		}
	}
	return false
}

func stringFromJSONValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func normalizeAOMissionTerminalStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "complete", "completed", "done":
		return "completed"
	case "promote", "promoted", "promotion_ready":
		return "promoted"
	case "deny", "denied":
		return "denied"
	case "block", "blocked":
		return "blocked"
	default:
		return strings.ToLower(strings.TrimSpace(status))
	}
}

func aoMissionTerminalCanClose(status string) bool {
	switch normalizeAOMissionTerminalStatus(status) {
	case "completed", "promoted":
		return true
	default:
		return false
	}
}

func aoMissionTerminalRequiresRepair(status string) bool {
	switch normalizeAOMissionTerminalStatus(status) {
	case "denied", "blocked":
		return true
	default:
		return false
	}
}

func aoMissionTerminalSmokeStatusAllowed(readbackStatus, terminalStatus string) bool {
	switch normalizeAOMissionTerminalStatus(terminalStatus) {
	case "completed", "promoted":
		return readbackStatus == "ready"
	case "denied", "blocked":
		return readbackStatus == "blocked"
	default:
		return false
	}
}

func buildAOMissionTerminalReadinessSummary(terminalStatus, readbackStatus string, completedNodes, totalNodes int, exactNextAction string) map[string]any {
	normalized := normalizeAOMissionTerminalStatus(terminalStatus)
	return map[string]any{
		"schema":                  "ao.foundry.ao-mission-terminal-readiness-summary.v0.1",
		"status":                  readbackStatus,
		"foundry_terminal_status": normalized,
		"can_close_mission":       aoMissionTerminalCanClose(normalized),
		"requires_repair":         aoMissionTerminalRequiresRepair(normalized),
		"completed_nodes":         completedNodes,
		"total_nodes":             totalNodes,
		"exact_next_action":       exactNextAction,
		"source":                  "foundry_final_rollup",
		"safe_to_execute":         false,
		"executes_work":           false,
		"approves_work":           false,
		"mutates_repositories":    false,
		"provider_calls":          false,
		"credential_use":          false,
	}
}

func validateAOMissionArtifactManifestDigests(manifestPath string, manifest map[string]any) error {
	refs, ok := manifest["artifact_refs"].([]any)
	if !ok {
		return nil
	}
	for i, raw := range refs {
		ref, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("artifact manifest artifact_refs[%d] must be an object", i)
		}
		path, _ := ref["ref"].(string)
		if path == "" {
			path, _ = ref["path"].(string)
		}
		want, _ := ref["digest"].(string)
		if want == "" {
			want, _ = ref["sha256"].(string)
		}
		if strings.TrimSpace(path) == "" || strings.TrimSpace(want) == "" {
			return fmt.Errorf("artifact manifest artifact_refs[%d] requires ref/path and digest/sha256", i)
		}
		if !strings.HasPrefix(want, "sha256:") {
			return fmt.Errorf("artifact manifest artifact_refs[%d] digest must start with sha256:", i)
		}
		actualPath, err := resolveAOMissionArtifactRef(manifestPath, path)
		if err != nil {
			return fmt.Errorf("artifact manifest ref %q: %w", path, err)
		}
		got, err := digestPath(actualPath)
		if err != nil {
			return fmt.Errorf("artifact manifest ref %q: %w", path, err)
		}
		if got != want {
			return fmt.Errorf("artifact manifest ref %q digest mismatch", path)
		}
	}
	return nil
}

func resolveAOMissionArtifactRef(manifestPath, ref string) (string, error) {
	if filepath.IsAbs(ref) {
		if _, err := os.Stat(ref); err != nil {
			return "", err
		}
		return ref, nil
	}
	if _, err := os.Stat(ref); err == nil {
		return ref, nil
	}
	candidate := filepath.Join(filepath.Dir(manifestPath), ref)
	if _, err := os.Stat(candidate); err != nil {
		return "", err
	}
	return candidate, nil
}

func digestPath(path string) (string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return "sha256:" + fmt.Sprintf("%x", sum[:]), nil
}

func intFromJSONNumber(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}
