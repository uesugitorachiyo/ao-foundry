package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

func runClassGate(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: foundry class-gate evaluate --atlas <path> --covenant <path> --sentinel <path> --promoter <path> --rollback <path> --command <path> --ci <path> --out <path>")
		return 2
	}
	switch args[0] {
	case "evaluate":
		return runClassGateEvaluate(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "foundry class-gate: unknown command %q\n", args[0])
		return 2
	}
}

func runComplexRepo(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		fmt.Fprintln(stderr, "usage: foundry complex-repo <node-gate evaluate|node execute|closure backfill|promotion-rollup evaluate> ...")
		return 2
	}
	switch {
	case args[0] == "node-gate" && args[1] == "evaluate":
		return runComplexRepoNodeGateEvaluate(args[2:], stdout, stderr)
	case args[0] == "node" && args[1] == "execute":
		return runComplexRepoNodeExecute(args[2:], stdout, stderr)
	case args[0] == "closure" && args[1] == "backfill":
		return runComplexRepoClosureBackfill(args[2:], stdout, stderr)
	case args[0] == "promotion-rollup" && args[1] == "evaluate":
		return runComplexRepoPromotionRollupEvaluate(args[2:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "foundry complex-repo: unknown command %q\n", strings.Join(args, " "))
		return 2
	}
}

func runComplexRepoClosureBackfill(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("complex-repo closure backfill", stderr)
	missionPath := fs.String("mission", "", "Atlas mission continuation evidence")
	workgraphPath := fs.String("workgraph", "", "Atlas final complex_repo_mutation workgraph")
	runLinksRoot := fs.String("run-links-root", "", "root containing per-node run-link.json evidence")
	closureRoot := fs.String("closure-root", "", "output root for digest-bound per-node closure evidence")
	finalNodeGatePath := fs.String("final-node-gate", "", "final synthesis node gate evidence")
	var nodeGatePaths repeatedStringFlag
	fs.Var(&nodeGatePaths, "node-gate", "additional per-node gate evidence; may be repeated")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	if strings.TrimSpace(*missionPath) == "" ||
		strings.TrimSpace(*workgraphPath) == "" ||
		strings.TrimSpace(*runLinksRoot) == "" ||
		strings.TrimSpace(*closureRoot) == "" ||
		strings.TrimSpace(*finalNodeGatePath) == "" {
		fmt.Fprintln(stderr, "--mission, --workgraph, --run-links-root, --closure-root, and --final-node-gate are required")
		return 2
	}
	manifest, err := buildComplexRepoClosureEvidence(complexPromotionRollupPaths{
		Mission:       *missionPath,
		Workgraph:     *workgraphPath,
		RunLinksRoot:  *runLinksRoot,
		ClosureRoot:   *closureRoot,
		NodeGates:     nodeGatePaths,
		FinalNodeGate: *finalNodeGatePath,
	})
	if err != nil {
		fmt.Fprintf(stderr, "complex closure backfill: %v\n", err)
		return 1
	}
	manifestPath := filepath.Join(*closureRoot, "closure-manifest.json")
	if err := writeJSONFile(manifestPath, manifest); err != nil {
		fmt.Fprintf(stderr, "write complex closure manifest: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "complex_closure_root=%s\n", *closureRoot)
	fmt.Fprintf(stdout, "node_count=%v\n", manifest["node_count"])
	fmt.Fprintf(stdout, "evidence_item_count=%v\n", manifest["evidence_item_count"])
	return 0
}

func runComplexRepoPromotionRollupEvaluate(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("complex-repo promotion-rollup evaluate", stderr)
	missionPath := fs.String("mission", "", "Atlas mission continuation evidence")
	workgraphPath := fs.String("workgraph", "", "Atlas final complex_repo_mutation workgraph")
	runLinksRoot := fs.String("run-links-root", "", "root containing per-node run-link.json evidence")
	closureRoot := fs.String("closure-root", "", "root containing digest-bound per-node closure evidence")
	finalNodeGatePath := fs.String("final-node-gate", "", "final synthesis node gate evidence")
	outPath := fs.String("out", "", "promotion rollup output path")
	jsonOut := fs.Bool("json", false, "also write JSON to stdout")
	var nodeGatePaths repeatedStringFlag
	fs.Var(&nodeGatePaths, "node-gate", "additional per-node gate evidence; may be repeated")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	if strings.TrimSpace(*missionPath) == "" ||
		strings.TrimSpace(*workgraphPath) == "" ||
		strings.TrimSpace(*runLinksRoot) == "" ||
		strings.TrimSpace(*finalNodeGatePath) == "" ||
		strings.TrimSpace(*outPath) == "" {
		fmt.Fprintln(stderr, "--mission, --workgraph, --run-links-root, --final-node-gate, and --out are required")
		return 2
	}
	rollup, err := buildComplexRepoMutationPromotionRollup(complexPromotionRollupPaths{
		Mission:       *missionPath,
		Workgraph:     *workgraphPath,
		RunLinksRoot:  *runLinksRoot,
		ClosureRoot:   *closureRoot,
		NodeGates:     nodeGatePaths,
		FinalNodeGate: *finalNodeGatePath,
	})
	if err != nil {
		fmt.Fprintf(stderr, "complex promotion rollup: %v\n", err)
		return 1
	}
	if err := writeJSONFile(*outPath, rollup); err != nil {
		fmt.Fprintf(stderr, "write complex promotion rollup: %v\n", err)
		return 1
	}
	if *jsonOut {
		if err := writeJSON(stdout, rollup); err != nil {
			fmt.Fprintf(stderr, "write complex promotion rollup json: %v\n", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "complex_promotion_rollup=%s\n", *outPath)
	fmt.Fprintf(stdout, "status=%s\n", rollup.Status)
	fmt.Fprintf(stdout, "safe_to_promote=%t\n", rollup.SafeToPromote)
	if rollup.FirstFailingCheck != "" {
		fmt.Fprintf(stdout, "first_failing_check=%s\n", rollup.FirstFailingCheck)
	}
	return 0
}

func runComplexRepoNodeGateEvaluate(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("complex-repo node-gate evaluate", stderr)
	workgraphPath := fs.String("workgraph", "", "Atlas complex_repo_mutation workgraph")
	foundryImportPath := fs.String("foundry-import", "", "Atlas Foundry import for the selected complex node")
	candidatePath := fs.String("candidate", "", "Atlas candidate record for the selected complex node")
	rollbackPath := fs.String("rollback", "", "Atlas rollback record for the selected complex node")
	outPath := fs.String("out", "", "complex node gate output path")
	jsonOut := fs.Bool("json", false, "also write JSON to stdout")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	if strings.TrimSpace(*workgraphPath) == "" ||
		strings.TrimSpace(*foundryImportPath) == "" ||
		strings.TrimSpace(*candidatePath) == "" ||
		strings.TrimSpace(*rollbackPath) == "" ||
		strings.TrimSpace(*outPath) == "" {
		fmt.Fprintln(stderr, "--workgraph, --foundry-import, --candidate, --rollback, and --out are required")
		return 2
	}
	gate, err := buildComplexRepoMutationNodeGate(complexNodeGatePaths{
		Workgraph:     *workgraphPath,
		FoundryImport: *foundryImportPath,
		Candidate:     *candidatePath,
		Rollback:      *rollbackPath,
	})
	if err != nil {
		fmt.Fprintf(stderr, "complex node gate: %v\n", err)
		return 1
	}
	if err := writeJSONFile(*outPath, gate); err != nil {
		fmt.Fprintf(stderr, "write complex node gate: %v\n", err)
		return 1
	}
	if *jsonOut {
		if err := writeJSON(stdout, gate); err != nil {
			fmt.Fprintf(stderr, "write complex node gate json: %v\n", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "complex_node_gate=%s\n", *outPath)
	fmt.Fprintf(stdout, "status=%s\n", gate.Status)
	fmt.Fprintf(stdout, "node_id=%s\n", gate.NodeID)
	fmt.Fprintf(stdout, "safe_to_request=%t\n", gate.SafeToRequest)
	fmt.Fprintf(stdout, "safe_to_execute=%t\n", gate.SafeToExecute)
	if gate.FirstFailingCheck != "" {
		fmt.Fprintf(stdout, "first_failing_check=%s\n", gate.FirstFailingCheck)
	}
	return 0
}

func runComplexRepoNodeExecute(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("complex-repo node execute", stderr)
	nodeGatePath := fs.String("node-gate", "", "ready complex_repo_mutation node gate")
	nodeRecordOut := fs.String("node-record-out", "", "node record output path")
	runLinkOut := fs.String("run-link-out", "", "Atlas run-link output path")
	nodeClass := fs.String("node-class", "", "selected node class")
	scope := fs.String("scope", "", "exact node write scope")
	summary := fs.String("summary", "", "node execution summary")
	pr := fs.String("pr", "", "merged execution pull request")
	mergeCommit := fs.String("merge-commit", "", "merged execution commit")
	ci := fs.String("ci", "passed", "CI evidence readback")
	repoID := fs.String("repo-id", "", "current repository id, required when node gate declares target_factory_repo")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	if strings.TrimSpace(*nodeGatePath) == "" ||
		strings.TrimSpace(*nodeRecordOut) == "" ||
		strings.TrimSpace(*runLinkOut) == "" ||
		strings.TrimSpace(*nodeClass) == "" ||
		strings.TrimSpace(*scope) == "" ||
		strings.TrimSpace(*summary) == "" {
		fmt.Fprintln(stderr, "--node-gate, --node-record-out, --run-link-out, --node-class, --scope, and --summary are required")
		return 2
	}
	if err := validateAtlasPublicString(*scope); err != nil {
		fmt.Fprintf(stderr, "complex node execute scope: %v\n", err)
		return 1
	}
	gate, err := loadComplexRepoMutationNodeGate(*nodeGatePath)
	if err != nil {
		fmt.Fprintf(stderr, "complex node execute: %v\n", err)
		return 1
	}
	if gate.Status != "ready" || !gate.SafeToRequest || !gate.SafeToExecute || !gate.LiveExecutionAuthority {
		fmt.Fprintln(stderr, "complex node execute requires ready node gate with safe_to_request=true and safe_to_execute=true")
		return 1
	}
	if strings.TrimSpace(gate.TargetFactoryRepo) != "" {
		if strings.TrimSpace(*repoID) == "" {
			fmt.Fprintln(stderr, "complex node execute requires --repo-id matching target_factory_repo")
			return 1
		}
		if strings.TrimSpace(*repoID) != strings.TrimSpace(gate.TargetFactoryRepo) {
			fmt.Fprintf(stderr, "complex node execute repo mismatch: target_factory_repo=%s repo_id=%s\n", gate.TargetFactoryRepo, *repoID)
			return 1
		}
	}
	record := buildComplexNodeRecord(gate, *nodeClass, *scope, *summary)
	if err := writeJSONFile(*nodeRecordOut, record); err != nil {
		fmt.Fprintf(stderr, "write complex node record: %v\n", err)
		return 1
	}
	link := buildComplexNodeRunLink(gate, *scope, *nodeGatePath, *pr, *mergeCommit, *ci)
	if err := writeJSONFile(*runLinkOut, link); err != nil {
		fmt.Fprintf(stderr, "write complex node run-link: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "status=completed\nnode_id=%s\ntask_id=%s\nnode_record=%s\nrun_link=%s\n", gate.NodeID, gate.TaskID, *nodeRecordOut, *runLinkOut)
	return 0
}

func runClassGateEvaluate(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("class-gate evaluate", stderr)
	atlasPath := fs.String("atlas", "", "Atlas mutation-class classification evidence")
	covenantPath := fs.String("covenant", "", "Covenant mutation-class authority ticket")
	sentinelPath := fs.String("sentinel", "", "Sentinel mutation-class hold verdict")
	promoterPath := fs.String("promoter", "", "Promoter mutation-class readiness evidence")
	rollbackPath := fs.String("rollback", "", "rollback rehearsal evidence")
	commandPath := fs.String("command", "", "AO Command authority-ladder readback")
	ciPath := fs.String("ci", "", "CI status evidence")
	testOnlySuccessPath := fs.String("test-only-success", "", "completed test_only live rehearsal evidence for low_risk_code dry-run readiness")
	multiRepoPlanPath := fs.String("multi-repo-plan", "", "multi-repo low-risk sequencing and rollback plan evidence")
	lowRiskCodeLiveSuccessPath := fs.String("low-risk-code-live-success", "", "completed low_risk_code live rehearsal success readback for multi_repo_low_risk readiness")
	complexNodeGatePath := fs.String("complex-node-gate", "", "complex_repo_mutation node gate readback evidence")
	outPath := fs.String("out", "", "class gate output path")
	jsonOut := fs.Bool("json", false, "also write JSON to stdout")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	if strings.TrimSpace(*atlasPath) == "" ||
		strings.TrimSpace(*covenantPath) == "" ||
		strings.TrimSpace(*sentinelPath) == "" ||
		strings.TrimSpace(*promoterPath) == "" ||
		strings.TrimSpace(*rollbackPath) == "" ||
		strings.TrimSpace(*commandPath) == "" ||
		strings.TrimSpace(*ciPath) == "" ||
		strings.TrimSpace(*outPath) == "" {
		fmt.Fprintln(stderr, "--atlas, --covenant, --sentinel, --promoter, --rollback, --command, --ci, and --out are required")
		return 2
	}
	gate, err := evaluateMutationClassGate(classGateEvidencePaths{
		Atlas:                  *atlasPath,
		Covenant:               *covenantPath,
		Sentinel:               *sentinelPath,
		Promoter:               *promoterPath,
		Rollback:               *rollbackPath,
		Command:                *commandPath,
		CI:                     *ciPath,
		TestOnlySuccess:        *testOnlySuccessPath,
		MultiRepoPlan:          *multiRepoPlanPath,
		LowRiskCodeLiveSuccess: *lowRiskCodeLiveSuccessPath,
		ComplexNodeGate:        *complexNodeGatePath,
	})
	if err != nil {
		fmt.Fprintf(stderr, "class gate: %v\n", err)
		return 1
	}
	if err := writeJSONFile(*outPath, gate); err != nil {
		fmt.Fprintf(stderr, "write class gate: %v\n", err)
		return 1
	}
	if *jsonOut {
		if err := writeJSON(stdout, gate); err != nil {
			fmt.Fprintf(stderr, "write class gate json: %v\n", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "class_gate=%s\n", *outPath)
	fmt.Fprintf(stdout, "status=%s\n", gate.Status)
	fmt.Fprintf(stdout, "mutation_class=%s\n", gate.MutationClass)
	fmt.Fprintf(stdout, "safe_to_request=%t\n", gate.SafeToRequest)
	fmt.Fprintf(stdout, "safe_to_execute=%t\n", gate.SafeToExecute)
	return 0
}
