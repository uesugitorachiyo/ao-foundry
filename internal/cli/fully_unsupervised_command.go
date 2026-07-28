package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

func runFullyUnsupervised(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		fmt.Fprintln(stderr, "usage: foundry fully-unsupervised <readiness evaluate|authority-gates evaluate|first-non-planning gate evaluate|first-non-planning stop-gate evaluate|first-non-planning final-closure evaluate> ...")
		return 2
	}
	switch {
	case args[0] == "readiness" && args[1] == "evaluate":
		return runFullyUnsupervisedReadinessEvaluate(args[2:], stdout, stderr)
	case args[0] == "authority-gates" && args[1] == "evaluate":
		return runFullyUnsupervisedAuthorityGatesEvaluate(args[2:], stdout, stderr)
	case len(args) >= 3 && args[0] == "first-non-planning" && args[1] == "gate" && args[2] == "evaluate":
		return runFullyUnsupervisedFirstNonPlanningGateEvaluate(args[3:], stdout, stderr)
	case len(args) >= 3 && args[0] == "first-non-planning" && args[1] == "stop-gate" && args[2] == "evaluate":
		return runFullyUnsupervisedFirstNonPlanningStopGateEvaluate(args[3:], stdout, stderr)
	case len(args) >= 3 && args[0] == "first-non-planning" && args[1] == "final-closure" && args[2] == "evaluate":
		return runFullyUnsupervisedFirstNonPlanningFinalClosureEvaluate(args[3:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "foundry fully-unsupervised: unknown command %q\n", strings.Join(args, " "))
		return 2
	}
}

type fullyUnsupervisedReadinessPaths struct {
	BlueprintImport  string
	Workgraph        string
	FoundryImport    string
	AtlasSummary     string
	SliceManifest    string
	FinalSynthesis   string
	FirstNodeGate    string
	TaskRoot         string
	ContextRoot      string
	CandidateRoot    string
	RollbackRoot     string
	NodeEvidenceRoot string
	RepairRoot       string
	RepackRoot       string
}

type fullyUnsupervisedAuthorityGatePaths struct {
	fullyUnsupervisedReadinessPaths
	ContinuationHandoff string
}

type fullyUnsupervisedFirstNonPlanningGatePaths struct {
	BlueprintImport     string
	Workgraph           string
	FoundryImport       string
	ContinuationHandoff string
	AtlasSummary        string
	SliceManifest       string
	FinalSynthesis      string
	Candidate           string
	Rollback            string
	StopGateClearance   string
}

type fullyUnsupervisedFirstNonPlanningStopGatePaths struct {
	Workgraph       string
	StopGateGraph   string
	NodeGate        string
	RunLink         string
	Rollback        string
	Sentinel        string
	Promoter        string
	CommandReadback string
	WorkgraphOut    string
}

type fullyUnsupervisedFirstNonPlanningFinalClosurePaths struct {
	Workgraph         string
	RunLinksRoot      string
	StopGatesRoot     string
	FinalSynthesis    string
	RollupOut         string
	MissionCompletion string
	Promoter          string
	CommandReadback   string
}

func runFullyUnsupervisedReadinessEvaluate(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("fully-unsupervised readiness evaluate", stderr)
	blueprintImportPath := fs.String("blueprint-import", "", "Atlas Blueprint import")
	workgraphPath := fs.String("workgraph", "", "Atlas fully unsupervised readiness workgraph")
	foundryImportPath := fs.String("foundry-import", "", "Atlas Foundry import")
	atlasSummaryPath := fs.String("atlas-summary", "", "Atlas first-phase summary")
	sliceManifestPath := fs.String("slice-manifest", "", "Atlas SDD slice completion manifest")
	finalSynthesisPath := fs.String("final-synthesis", "", "Atlas final evidence synthesis")
	firstNodeGatePath := fs.String("first-node-gate", "", "Foundry first-node gate output")
	taskRoot := fs.String("task-root", "", "Atlas task record directory")
	contextRoot := fs.String("context-root", "", "Atlas context pack directory")
	candidateRoot := fs.String("candidate-root", "", "Atlas candidate record directory")
	rollbackRoot := fs.String("rollback-root", "", "Atlas rollback record directory")
	nodeEvidenceRoot := fs.String("node-evidence-root", "", "Atlas node evidence directory")
	repairRoot := fs.String("repair-root", "", "Atlas repair plan directory")
	repackRoot := fs.String("context-repack-root", "", "Atlas context repack plan directory")
	outPath := fs.String("out", "", "readiness rollup output path")
	jsonOut := fs.Bool("json", false, "also write JSON to stdout")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	required := map[string]string{
		"--blueprint-import":    *blueprintImportPath,
		"--workgraph":           *workgraphPath,
		"--foundry-import":      *foundryImportPath,
		"--atlas-summary":       *atlasSummaryPath,
		"--slice-manifest":      *sliceManifestPath,
		"--final-synthesis":     *finalSynthesisPath,
		"--first-node-gate":     *firstNodeGatePath,
		"--task-root":           *taskRoot,
		"--context-root":        *contextRoot,
		"--candidate-root":      *candidateRoot,
		"--rollback-root":       *rollbackRoot,
		"--node-evidence-root":  *nodeEvidenceRoot,
		"--repair-root":         *repairRoot,
		"--context-repack-root": *repackRoot,
		"--out":                 *outPath,
	}
	missing := []string{}
	for flagName, value := range required {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, flagName)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		fmt.Fprintf(stderr, "%s are required\n", strings.Join(missing, ", "))
		return 2
	}
	rollup, err := buildFullyUnsupervisedReadinessRollup(fullyUnsupervisedReadinessPaths{
		BlueprintImport:  *blueprintImportPath,
		Workgraph:        *workgraphPath,
		FoundryImport:    *foundryImportPath,
		AtlasSummary:     *atlasSummaryPath,
		SliceManifest:    *sliceManifestPath,
		FinalSynthesis:   *finalSynthesisPath,
		FirstNodeGate:    *firstNodeGatePath,
		TaskRoot:         *taskRoot,
		ContextRoot:      *contextRoot,
		CandidateRoot:    *candidateRoot,
		RollbackRoot:     *rollbackRoot,
		NodeEvidenceRoot: *nodeEvidenceRoot,
		RepairRoot:       *repairRoot,
		RepackRoot:       *repackRoot,
	})
	if err != nil {
		fmt.Fprintf(stderr, "fully unsupervised readiness: %v\n", err)
		return 1
	}
	if err := writeJSONFile(*outPath, rollup); err != nil {
		fmt.Fprintf(stderr, "write fully unsupervised readiness rollup: %v\n", err)
		return 1
	}
	if *jsonOut {
		if err := writeJSON(stdout, rollup); err != nil {
			fmt.Fprintf(stderr, "write fully unsupervised readiness json: %v\n", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "fully_unsupervised_readiness_rollup=%s\n", *outPath)
	fmt.Fprintf(stdout, "status=%s\n", classGateString(rollup, "status"))
	fmt.Fprintf(stdout, "safe_to_execute=%t\n", classGateBool(rollup, "safe_to_execute"))
	if first := classGateString(rollup, "first_failing_check"); first != "" {
		fmt.Fprintf(stdout, "first_failing_check=%s\n", first)
	}
	return 0
}

func runFullyUnsupervisedAuthorityGatesEvaluate(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("fully-unsupervised authority-gates evaluate", stderr)
	blueprintImportPath := fs.String("blueprint-import", "", "Atlas Blueprint import")
	workgraphPath := fs.String("workgraph", "", "Atlas fully unsupervised authority-gates workgraph")
	foundryImportPath := fs.String("foundry-import", "", "Atlas Foundry import")
	continuationHandoffPath := fs.String("continuation-handoff", "", "Atlas Foundry continuation handoff")
	atlasSummaryPath := fs.String("atlas-summary", "", "Atlas first-phase summary")
	sliceManifestPath := fs.String("slice-manifest", "", "Atlas SDD slice completion manifest")
	finalSynthesisPath := fs.String("final-synthesis", "", "Atlas final evidence synthesis")
	firstNodeGatePath := fs.String("first-node-gate", "", "Foundry first-node gate output")
	taskRoot := fs.String("task-root", "", "Atlas task record directory")
	contextRoot := fs.String("context-root", "", "Atlas context pack directory")
	candidateRoot := fs.String("candidate-root", "", "Atlas candidate record directory")
	rollbackRoot := fs.String("rollback-root", "", "Atlas rollback record directory")
	nodeEvidenceRoot := fs.String("node-evidence-root", "", "Atlas node evidence directory")
	repairRoot := fs.String("repair-root", "", "Atlas repair plan directory")
	repackRoot := fs.String("context-repack-root", "", "Atlas context repack plan directory")
	outPath := fs.String("out", "", "authority-gates rollup output path")
	jsonOut := fs.Bool("json", false, "also write JSON to stdout")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	required := map[string]string{
		"--blueprint-import":     *blueprintImportPath,
		"--workgraph":            *workgraphPath,
		"--foundry-import":       *foundryImportPath,
		"--continuation-handoff": *continuationHandoffPath,
		"--atlas-summary":        *atlasSummaryPath,
		"--slice-manifest":       *sliceManifestPath,
		"--final-synthesis":      *finalSynthesisPath,
		"--first-node-gate":      *firstNodeGatePath,
		"--task-root":            *taskRoot,
		"--context-root":         *contextRoot,
		"--candidate-root":       *candidateRoot,
		"--rollback-root":        *rollbackRoot,
		"--node-evidence-root":   *nodeEvidenceRoot,
		"--repair-root":          *repairRoot,
		"--context-repack-root":  *repackRoot,
		"--out":                  *outPath,
	}
	missing := []string{}
	for flagName, value := range required {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, flagName)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		fmt.Fprintf(stderr, "%s are required\n", strings.Join(missing, ", "))
		return 2
	}
	rollup, err := buildFullyUnsupervisedAuthorityGatesRollup(fullyUnsupervisedAuthorityGatePaths{
		fullyUnsupervisedReadinessPaths: fullyUnsupervisedReadinessPaths{
			BlueprintImport:  *blueprintImportPath,
			Workgraph:        *workgraphPath,
			FoundryImport:    *foundryImportPath,
			AtlasSummary:     *atlasSummaryPath,
			SliceManifest:    *sliceManifestPath,
			FinalSynthesis:   *finalSynthesisPath,
			FirstNodeGate:    *firstNodeGatePath,
			TaskRoot:         *taskRoot,
			ContextRoot:      *contextRoot,
			CandidateRoot:    *candidateRoot,
			RollbackRoot:     *rollbackRoot,
			NodeEvidenceRoot: *nodeEvidenceRoot,
			RepairRoot:       *repairRoot,
			RepackRoot:       *repackRoot,
		},
		ContinuationHandoff: *continuationHandoffPath,
	})
	if err != nil {
		fmt.Fprintf(stderr, "fully unsupervised authority gates: %v\n", err)
		return 1
	}
	if err := writeJSONFile(*outPath, rollup); err != nil {
		fmt.Fprintf(stderr, "write fully unsupervised authority-gates rollup: %v\n", err)
		return 1
	}
	if *jsonOut {
		if err := writeJSON(stdout, rollup); err != nil {
			fmt.Fprintf(stderr, "write fully unsupervised authority-gates json: %v\n", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "fully_unsupervised_authority_gates_rollup=%s\n", *outPath)
	fmt.Fprintf(stdout, "status=%s\n", classGateString(rollup, "status"))
	fmt.Fprintf(stdout, "safe_to_execute=%t\n", classGateBool(rollup, "safe_to_execute"))
	if first := classGateString(rollup, "first_failing_check"); first != "" {
		fmt.Fprintf(stdout, "first_failing_check=%s\n", first)
	}
	return 0
}

func runFullyUnsupervisedFirstNonPlanningGateEvaluate(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("fully-unsupervised first-non-planning gate evaluate", stderr)
	blueprintImportPath := fs.String("blueprint-import", "", "Atlas Blueprint import")
	workgraphPath := fs.String("workgraph", "", "Atlas fully unsupervised first non-planning workgraph")
	foundryImportPath := fs.String("foundry-import", "", "Atlas Foundry import")
	continuationHandoffPath := fs.String("continuation-handoff", "", "Atlas Foundry continuation handoff")
	atlasSummaryPath := fs.String("atlas-summary", "", "Atlas first-phase summary")
	sliceManifestPath := fs.String("slice-manifest", "", "Atlas SDD slice completion manifest")
	finalSynthesisPath := fs.String("final-synthesis", "", "Atlas final evidence synthesis")
	candidatePath := fs.String("candidate", "", "Atlas first non-planning candidate record")
	rollbackPath := fs.String("rollback", "", "Atlas first non-planning rollback record")
	stopGateClearancePath := fs.String("stop-gate-clearance", "", "optional prior stop-gate clearance for serialized nodes")
	outPath := fs.String("out", "", "first non-planning node gate output path")
	jsonOut := fs.Bool("json", false, "also write JSON to stdout")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	required := map[string]string{
		"--blueprint-import":     *blueprintImportPath,
		"--workgraph":            *workgraphPath,
		"--foundry-import":       *foundryImportPath,
		"--continuation-handoff": *continuationHandoffPath,
		"--atlas-summary":        *atlasSummaryPath,
		"--slice-manifest":       *sliceManifestPath,
		"--final-synthesis":      *finalSynthesisPath,
		"--candidate":            *candidatePath,
		"--rollback":             *rollbackPath,
		"--out":                  *outPath,
	}
	missing := []string{}
	for flagName, value := range required {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, flagName)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		fmt.Fprintf(stderr, "%s are required\n", strings.Join(missing, ", "))
		return 2
	}
	gate, err := buildFullyUnsupervisedFirstNonPlanningGate(fullyUnsupervisedFirstNonPlanningGatePaths{
		BlueprintImport:     *blueprintImportPath,
		Workgraph:           *workgraphPath,
		FoundryImport:       *foundryImportPath,
		ContinuationHandoff: *continuationHandoffPath,
		AtlasSummary:        *atlasSummaryPath,
		SliceManifest:       *sliceManifestPath,
		FinalSynthesis:      *finalSynthesisPath,
		Candidate:           *candidatePath,
		Rollback:            *rollbackPath,
		StopGateClearance:   *stopGateClearancePath,
	})
	if err != nil {
		fmt.Fprintf(stderr, "fully unsupervised first non-planning gate: %v\n", err)
		return 1
	}
	if err := writeJSONFile(*outPath, gate); err != nil {
		fmt.Fprintf(stderr, "write fully unsupervised first non-planning gate: %v\n", err)
		return 1
	}
	if *jsonOut {
		if err := writeJSON(stdout, gate); err != nil {
			fmt.Fprintf(stderr, "write fully unsupervised first non-planning gate json: %v\n", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "first_non_planning_node_gate=%s\n", *outPath)
	fmt.Fprintf(stdout, "status=%s\n", gate.Status)
	fmt.Fprintf(stdout, "node_id=%s\n", gate.NodeID)
	fmt.Fprintf(stdout, "safe_to_request=%t\n", gate.SafeToRequest)
	fmt.Fprintf(stdout, "safe_to_execute=%t\n", gate.SafeToExecute)
	if gate.FirstFailingCheck != "" {
		fmt.Fprintf(stdout, "first_failing_check=%s\n", gate.FirstFailingCheck)
	}
	return 0
}

func runFullyUnsupervisedFirstNonPlanningStopGateEvaluate(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("fully-unsupervised first-non-planning stop-gate evaluate", stderr)
	workgraphPath := fs.String("workgraph", "", "Atlas workgraph after predecessor completion")
	stopGateGraphPath := fs.String("stop-gate-graph", "", "Atlas first non-planning stop-gate graph")
	nodeGatePath := fs.String("node-gate", "", "predecessor Foundry node gate")
	runLinkPath := fs.String("run-link", "", "predecessor Atlas run-link")
	rollbackPath := fs.String("rollback", "", "predecessor rollback record")
	sentinelPath := fs.String("sentinel", "", "Sentinel stop-gate verdict")
	promoterPath := fs.String("promoter", "", "Promoter stop-gate verdict")
	commandReadbackPath := fs.String("command-readback", "", "Command stop-gate readback")
	outPath := fs.String("out", "", "stop-gate clearance output path")
	workgraphOutPath := fs.String("workgraph-out", "", "optional derived workgraph output with next node ready")
	jsonOut := fs.Bool("json", false, "also write JSON to stdout")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	required := map[string]string{
		"--workgraph":       *workgraphPath,
		"--stop-gate-graph": *stopGateGraphPath,
		"--node-gate":       *nodeGatePath,
		"--run-link":        *runLinkPath,
		"--rollback":        *rollbackPath,
		"--out":             *outPath,
	}
	missing := []string{}
	for flagName, value := range required {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, flagName)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		fmt.Fprintf(stderr, "%s are required\n", strings.Join(missing, ", "))
		return 2
	}
	clearance, workgraphOut, err := buildFullyUnsupervisedFirstNonPlanningStopGateClearance(fullyUnsupervisedFirstNonPlanningStopGatePaths{
		Workgraph:       *workgraphPath,
		StopGateGraph:   *stopGateGraphPath,
		NodeGate:        *nodeGatePath,
		RunLink:         *runLinkPath,
		Rollback:        *rollbackPath,
		Sentinel:        *sentinelPath,
		Promoter:        *promoterPath,
		CommandReadback: *commandReadbackPath,
		WorkgraphOut:    *workgraphOutPath,
	})
	if err != nil {
		fmt.Fprintf(stderr, "fully unsupervised first non-planning stop-gate: %v\n", err)
		return 1
	}
	if err := writeJSONFile(*outPath, clearance); err != nil {
		fmt.Fprintf(stderr, "write fully unsupervised first non-planning stop-gate clearance: %v\n", err)
		return 1
	}
	if classGateBool(clearance, "safe_to_continue") && strings.TrimSpace(*workgraphOutPath) != "" {
		if err := writeJSONFile(*workgraphOutPath, workgraphOut); err != nil {
			fmt.Fprintf(stderr, "write fully unsupervised first non-planning workgraph: %v\n", err)
			return 1
		}
	}
	if *jsonOut {
		if err := writeJSON(stdout, clearance); err != nil {
			fmt.Fprintf(stderr, "write fully unsupervised first non-planning stop-gate json: %v\n", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "first_non_planning_stop_gate_clearance=%s\n", *outPath)
	fmt.Fprintf(stdout, "status=%s\n", classGateString(clearance, "status"))
	fmt.Fprintf(stdout, "safe_to_continue=%t\n", classGateBool(clearance, "safe_to_continue"))
	if first := classGateString(clearance, "first_failing_check"); first != "" {
		fmt.Fprintf(stdout, "first_failing_check=%s\n", first)
	}
	return 0
}

func runFullyUnsupervisedFirstNonPlanningFinalClosureEvaluate(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("fully-unsupervised first-non-planning final-closure evaluate", stderr)
	workgraphPath := fs.String("workgraph", "", "final Atlas workgraph after all nodes complete")
	runLinksRoot := fs.String("run-links-root", "", "Atlas foundry-execution root containing per-node run-links")
	stopGatesRoot := fs.String("stop-gates-root", "", "Foundry evidence root containing stop-gate clearance directories")
	finalSynthesisPath := fs.String("final-synthesis", "", "optional pre-execution Atlas final synthesis to re-evaluate")
	missionCompletionOut := fs.String("mission-completion-out", "", "optional Atlas mission completion evidence output path")
	promoterOut := fs.String("promoter-out", "", "optional Promoter final verdict output path")
	commandReadbackOut := fs.String("command-readback-out", "", "optional Command final readback output path")
	outPath := fs.String("out", "", "Foundry final promotion rollup output path")
	jsonOut := fs.Bool("json", false, "also write JSON to stdout")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	required := map[string]string{
		"--workgraph":       *workgraphPath,
		"--run-links-root":  *runLinksRoot,
		"--stop-gates-root": *stopGatesRoot,
		"--out":             *outPath,
	}
	missing := []string{}
	for flagName, value := range required {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, flagName)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		fmt.Fprintf(stderr, "%s are required\n", strings.Join(missing, ", "))
		return 2
	}
	paths := fullyUnsupervisedFirstNonPlanningFinalClosurePaths{
		Workgraph:         *workgraphPath,
		RunLinksRoot:      *runLinksRoot,
		StopGatesRoot:     *stopGatesRoot,
		FinalSynthesis:    *finalSynthesisPath,
		RollupOut:         *outPath,
		MissionCompletion: *missionCompletionOut,
		Promoter:          *promoterOut,
		CommandReadback:   *commandReadbackOut,
	}
	rollup, missionCompletion, promoter, commandReadback, err := buildFullyUnsupervisedFirstNonPlanningFinalClosure(paths)
	if err != nil {
		fmt.Fprintf(stderr, "fully unsupervised first non-planning final closure: %v\n", err)
		return 1
	}
	if err := writeJSONFile(*outPath, rollup); err != nil {
		fmt.Fprintf(stderr, "write fully unsupervised first non-planning final rollup: %v\n", err)
		return 1
	}
	if strings.TrimSpace(*missionCompletionOut) != "" {
		if err := writeJSONFile(*missionCompletionOut, missionCompletion); err != nil {
			fmt.Fprintf(stderr, "write fully unsupervised first non-planning mission completion: %v\n", err)
			return 1
		}
	}
	if strings.TrimSpace(*promoterOut) != "" {
		if err := writeJSONFile(*promoterOut, promoter); err != nil {
			fmt.Fprintf(stderr, "write fully unsupervised first non-planning promoter verdict: %v\n", err)
			return 1
		}
	}
	if strings.TrimSpace(*commandReadbackOut) != "" {
		if err := writeJSONFile(*commandReadbackOut, commandReadback); err != nil {
			fmt.Fprintf(stderr, "write fully unsupervised first non-planning command readback: %v\n", err)
			return 1
		}
	}
	if *jsonOut {
		if err := writeJSON(stdout, rollup); err != nil {
			fmt.Fprintf(stderr, "write fully unsupervised first non-planning final rollup json: %v\n", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "fully_unsupervised_final_rollup=%s\n", *outPath)
	fmt.Fprintf(stdout, "status=%s\n", classGateString(rollup, "status"))
	fmt.Fprintf(stdout, "safe_to_promote=%t\n", classGateBool(rollup, "safe_to_promote"))
	fmt.Fprintf(stdout, "fully_unsupervised_complex_mutation_live_proven=%t\n", classGateBool(rollup, "fully_unsupervised_complex_mutation_live_proven"))
	fmt.Fprintf(stdout, "next_denied_class=%s\n", classGateString(rollup, "next_denied_class"))
	if first := classGateString(rollup, "first_failing_check"); first != "" {
		fmt.Fprintf(stdout, "first_failing_check=%s\n", first)
	}
	return 0
}
