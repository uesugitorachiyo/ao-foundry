package cli

import (
	"strings"
)

func buildFullyUnsupervisedFirstNonPlanningGate(paths fullyUnsupervisedFirstNonPlanningGatePaths) (ComplexRepoMutationNodeGate, error) {
	gate := ComplexRepoMutationNodeGate{
		SchemaVersion:                    complexNodeGateSchema,
		Status:                           "blocked",
		MutationClass:                    "complex_repo_mutation",
		HighestProvenLiveClass:           "complex_repo_mutation",
		NextDeniedClass:                  "fully_unsupervised_complex_mutation",
		ExactNextAction:                  "repair_first_non_planning_gate_evidence_before_execution",
		AuthorityBoundary:                "first_bounded_non_planning_executable_support_docs_node",
		SourceEvidence:                   []MutationClassGateEvidence{},
		Blockers:                         []string{},
		FullyUnsupervisedComplexMutation: "denied",
		RSI:                              "denied",
		SchedulesWork:                    false,
		ExecutesWork:                     false,
		ApprovesWork:                     false,
		MutatesRepositories:              false,
		LiveExecutionAuthority:           false,
	}
	blueprintSource, blueprintImport, err := readComplexNodeGateObject("atlas_blueprint_import", paths.BlueprintImport)
	if err != nil {
		return gate, err
	}
	workgraphSource, workgraph, err := readComplexNodeGateObject("atlas_workgraph", paths.Workgraph)
	if err != nil {
		return gate, err
	}
	foundryImportSource, foundryImport, err := readComplexNodeGateObject("foundry_import", paths.FoundryImport)
	if err != nil {
		return gate, err
	}
	handoffSource, handoff, err := readComplexNodeGateObject("foundry_continuation_handoff", paths.ContinuationHandoff)
	if err != nil {
		return gate, err
	}
	summarySource, summary, err := readComplexNodeGateObject("atlas_first_summary", paths.AtlasSummary)
	if err != nil {
		return gate, err
	}
	manifestSource, manifest, err := readComplexNodeGateObject("sdd_slice_completion_manifest", paths.SliceManifest)
	if err != nil {
		return gate, err
	}
	finalSource, finalSynthesis, err := readComplexNodeGateObject("atlas_final_synthesis", paths.FinalSynthesis)
	if err != nil {
		return gate, err
	}
	candidateSource, candidate, err := readComplexNodeGateObject("candidate_record", paths.Candidate)
	if err != nil {
		return gate, err
	}
	rollbackSource, rollback, err := readComplexNodeGateObject("rollback_record", paths.Rollback)
	if err != nil {
		return gate, err
	}
	gate.SourceEvidence = append(gate.SourceEvidence, blueprintSource, workgraphSource, foundryImportSource, handoffSource, summarySource, manifestSource, finalSource, candidateSource, rollbackSource)
	var stopGateClearance map[string]any
	if strings.TrimSpace(paths.StopGateClearance) != "" {
		stopGateSource, clearance, err := readComplexNodeGateObject("stop_gate_clearance", paths.StopGateClearance)
		if err != nil {
			return gate, err
		}
		gate.SourceEvidence = append(gate.SourceEvidence, stopGateSource)
		stopGateClearance = clearance
	}

	nodes := classGateObjectSlice(workgraph["nodes"])
	totalNodes := len(nodes)
	readyNodes := countWorkgraphNodesWithStatus(nodes, "ready")
	blockedNodes := countWorkgraphNodesWithStatus(nodes, "blocked")
	tasks := classGateObjectSlice(foundryImport["tasks"])
	gate.FoundryImportID = classGateString(foundryImport, "id")
	gate.FoundryImportStatus = classGateString(foundryImport, "status")
	gate.FoundryImportTaskCount = len(tasks)
	gate.FoundryImportSchedulesWork = classGateBool(foundryImport, "schedules_work")
	gate.FoundryImportExecutesWork = classGateBool(foundryImport, "executes_work")
	gate.FoundryImportApprovesWork = classGateBool(foundryImport, "approves_work")
	gate.WorkgraphID = classGateString(workgraph, "id")
	gate.CandidateStatus = classGateString(candidate, "status")
	gate.RollbackStatus = classGateFirstNonEmpty(classGateString(rollback, "status"), "validated")
	gate.CandidateExecutableReady = classGateBool(candidate, "selected_first_safe_executable_node")
	gate.CandidateSafeToExecute = classGateStringSliceContains(classGateStringSlice(candidate, "required_evidence"), "safe_to_execute_in_foundry_only_under_gates:true")
	gate.RollbackSafeToExecute = len(classGateStringSlice(rollback, "rollback_scope")) > 0 && len(classGateStringSlice(rollback, "auto_stop_triggers")) > 0

	var task map[string]any
	if len(tasks) == 1 {
		task = tasks[0]
	}
	taskNodeID := classGateString(task, "node_id")
	taskID := classGateFirstNonEmpty(classGateString(task, "task_id"), classGateNestedString(task, "task", "id"))
	gate.TargetFactoryRepo = classGateFirstNonEmpty(classGateString(task, "target_factory_repo"), classGateNestedString(task, "task", "target_factory_repo"))
	candidateNodeID := classGateString(candidate, "node_id")
	rollbackNodeID := classGateString(rollback, "node_id")
	gate.NodeID = classGateFirstNonEmpty(candidateNodeID, taskNodeID, rollbackNodeID)
	gate.TaskID = classGateFirstNonEmpty(classGateString(candidate, "task_id"), taskID, classGateString(rollback, "task_id"))
	gate.RequiredGates = classGateFirstNonEmptyStringSlice(classGateStringSlice(task, "required_gates"), classGateStringSlice(candidate, "required_gates"), classGateNestedStringSlice(task, "task", "required_gates"))
	importWriteScope := classGateFirstNonEmptyStringSlice(classGateStringSlice(task, "write_scope"), classGateNestedStringSlice(task, "task", "write_scope"))
	importRollbackScope := classGateFirstNonEmptyStringSlice(classGateStringSlice(task, "rollback_scope"), classGateNestedStringSlice(task, "task", "rollback_scope"))
	importRequiredEvidence := classGateFirstNonEmptyStringSlice(classGateStringSlice(task, "required_evidence"), classGateNestedStringSlice(task, "task", "required_evidence"))
	scope := ""
	if len(importWriteScope) == 1 {
		scope = importWriteScope[0]
	}
	node, nodeFound := findWorkgraphNode(workgraph, gate.NodeID)
	nodeStatus := classGateString(node, "status")
	taskMutationClass := classGateFirstNonEmpty(classGateString(task, "mutation_class"), classGateNestedString(task, "task", "mutation_class"))
	serializedAfterStopGate := stopGateClearance != nil

	blockers := []string{}
	if classGateString(blueprintImport, "contract_version") != atlasBlueprintImportSchema ||
		classGateString(blueprintImport, "status") != "ready" ||
		classGateString(blueprintImport, "mutation_class") != "complex_repo_mutation" ||
		classGateBool(blueprintImport, "safe_to_execute") ||
		classGateBool(blueprintImport, "live_execution_proven") ||
		classGateBool(blueprintImport, "schedules_work") ||
		classGateBool(blueprintImport, "executes_work") ||
		classGateBool(blueprintImport, "approves_work") ||
		classGateBool(blueprintImport, "mutates_repositories") {
		blockers = append(blockers, "Atlas Blueprint import must be ready, complex_repo_mutation, and non-executable")
	}
	if classGateString(foundryImport, "contract_version") != atlasImportSchema ||
		gate.FoundryImportStatus != "ready_for_foundry_fixture_import" ||
		len(tasks) != 1 ||
		gate.FoundryImportSchedulesWork ||
		gate.FoundryImportExecutesWork ||
		gate.FoundryImportApprovesWork {
		blockers = append(blockers, "Foundry import must contain exactly one non-scheduling first non-planning node")
	}
	if classGateString(handoff, "contract_version") != "ao.atlas.foundry-continuation-handoff.v0.1" ||
		classGateString(handoff, "first_safe_node") != gate.NodeID ||
		int(classGateNumber(handoff, "total_node_count")) != totalNodes ||
		int(classGateNumber(handoff, "ready_node_count")) != readyNodes ||
		int(classGateNumber(handoff, "blocked_node_count")) != blockedNodes ||
		classGateBool(handoff, "schedules_work") ||
		classGateBool(handoff, "executes_work") ||
		classGateBool(handoff, "approves_work") {
		blockers = append(blockers, "Foundry continuation handoff must match workgraph counts and remain non-scheduling")
	}
	if classGateString(summary, "schema") != "ao.atlas.private-first-non-planning-summary.v0.1" ||
		classGateBool(summary, "atlas_executes_work") ||
		classGateBool(summary, "atlas_approves_work") ||
		classGateBool(summary, "fully_unsupervised_complex_mutation_live_proven") ||
		classGateString(summary, "rsi") != "denied" ||
		classGateString(summary, "highest_proven_live_class") != "complex_repo_mutation" ||
		classGateString(summary, "next_denied_class") != "fully_unsupervised_complex_mutation" ||
		(!serializedAfterStopGate && (classGateString(summary, "first_safe_executable_node") != gate.NodeID ||
			int(classGateNumber(summary, "planned_node_count")) != totalNodes ||
			int(classGateNumber(summary, "ready_node_count")) != readyNodes ||
			int(classGateNumber(summary, "blocked_node_count")) != blockedNodes)) {
		blockers = append(blockers, "Atlas first non-planning summary must match workgraph counts and preserve denial boundaries")
	}
	if classGateString(manifest, "schema") != "ao.atlas.private-first-non-planning-slice-manifest.v0.1" ||
		int(classGateNumber(manifest, "total_slices")) == 0 ||
		int(classGateNumber(manifest, "total_slices")) != int(classGateNumber(manifest, "completed_slices")) ||
		classGateBool(manifest, "fully_unsupervised_complex_mutation_live_proven") ||
		classGateString(manifest, "rsi") != "denied" {
		blockers = append(blockers, "Atlas first non-planning slice manifest must be complete and keep RSI denied")
	}
	if classGateString(finalSynthesis, "schema") != "ao.atlas.private-first-non-planning-final-evidence-synthesis.v0.1" ||
		(!serializedAfterStopGate && classGateString(finalSynthesis, "first_safe_executable_node") != gate.NodeID) ||
		classGateBool(finalSynthesis, "atlas_executes_work") ||
		classGateBool(finalSynthesis, "atlas_approves_work") ||
		classGateBool(finalSynthesis, "live_execution_performed_by_atlas") ||
		!classGateBool(finalSynthesis, "first_bounded_non_planning_rehearsal_requested") ||
		classGateBool(finalSynthesis, "fully_unsupervised_complex_mutation_live_proven") ||
		classGateString(finalSynthesis, "rsi") != "denied" ||
		classGateString(finalSynthesis, "highest_proven_live_class") != "complex_repo_mutation" ||
		classGateString(finalSynthesis, "next_denied_class") != "fully_unsupervised_complex_mutation" {
		blockers = append(blockers, "Atlas final synthesis must request first non-planning rehearsal while preserving denial boundaries")
	}
	if gate.NodeID == "" || gate.TaskID == "" {
		blockers = append(blockers, "first non-planning gate requires node_id and task_id")
	}
	if !nodeFound || nodeStatus != "ready" {
		blockers = append(blockers, "workgraph selected first non-planning node must be ready")
	}
	if taskNodeID != "" && candidateNodeID != "" && taskNodeID != candidateNodeID {
		blockers = append(blockers, "Foundry import node_id must match first non-planning candidate")
	}
	if rollbackNodeID != "" && gate.NodeID != "" && rollbackNodeID != gate.NodeID {
		blockers = append(blockers, "rollback record node_id must match selected first non-planning node")
	}
	if taskMutationClass != "complex_repo_mutation" {
		blockers = append(blockers, "selected first non-planning node must remain class complex_repo_mutation")
	}
	if serializedAfterStopGate {
		if classGateString(task, "authority_boundary") != "blocked_until_predecessor_terminal_evidence_and_stop_gate_clear" ||
			classGateString(stopGateClearance, "schema_version") != firstNonPlanningStopGateClearanceSchema ||
			classGateString(stopGateClearance, "status") != "ready" ||
			!classGateBool(stopGateClearance, "safe_to_continue") ||
			classGateString(stopGateClearance, "before_node") != gate.NodeID ||
			classGateString(stopGateClearance, "workgraph_id") != gate.WorkgraphID ||
			classGateString(stopGateClearance, "fully_unsupervised_complex_mutation") != "denied" ||
			classGateString(stopGateClearance, "rsi") != "denied" {
			blockers = append(blockers, "serialized first non-planning node requires ready predecessor stop-gate clearance")
		}
	} else if classGateString(task, "authority_boundary") != "first_bounded_non_planning_executable_support_docs_node" {
		blockers = append(blockers, "Foundry import authority boundary must be first bounded support/docs node")
	}
	if classGateString(candidate, "schema") != "ao.atlas.private-first-non-planning-candidate.v0.1" ||
		(!serializedAfterStopGate && (gate.CandidateStatus != "ready" ||
			!gate.CandidateExecutableReady ||
			classGateString(candidate, "candidate_class") != "first executable support/docs ticket node")) ||
		(serializedAfterStopGate && (gate.CandidateStatus != "blocked" ||
			gate.CandidateExecutableReady ||
			classGateString(candidate, "candidate_class") == "" ||
			classGateString(candidate, "safe_first_node_reason") != "serialized behind predecessor stop gate")) {
		blockers = append(blockers, "first non-planning candidate must match the active serialized node state")
	}
	if classGateBool(candidate, "fully_unsupervised_complex_mutation_live_proven") {
		blockers = append(blockers, "first non-planning candidate must not claim fully_unsupervised_complex_mutation live-proven")
	}
	if classGateString(candidate, "rsi") != "denied" {
		blockers = append(blockers, "first non-planning candidate must keep RSI denied")
	}
	if scope == "" || !classGateStringSliceContains(classGateStringSlice(candidate, "allowed_surfaces"), scope) {
		blockers = append(blockers, "first non-planning candidate allowed surface must match imported write scope")
	}
	if !classGateStringSliceEqual(importWriteScope, importRollbackScope) || !classGateStringSliceEqual(importWriteScope, classGateStringSlice(rollback, "rollback_scope")) {
		blockers = append(blockers, "first non-planning rollback scope must match imported write scope")
	}
	requiredEvidence := []string{
		"highest_proven_live_class:complex_repo_mutation",
		"next_denied_class:fully_unsupervised_complex_mutation",
		"rsi:denied",
		"readiness_only:false",
		"readback_only:false",
		"non_planning_rehearsal:true",
		"node_id:" + gate.NodeID,
		"safe_to_execute_in_foundry_only_under_gates:true",
	}
	for _, want := range requiredEvidence {
		if !classGateStringSliceContains(classGateStringSlice(candidate, "required_evidence"), want) ||
			!classGateStringSliceContains(importRequiredEvidence, want) {
			blockers = append(blockers, "first non-planning required evidence missing "+want)
			break
		}
	}
	for _, want := range []string{"no provider calls", "no credentials", "no dependency updates", "no auth policy config widening", "no secret env exposure", "no direct main mutation", "no concurrent mutation", "no RSI claim", "no release deploy publish upload tag"} {
		if !classGateStringSliceContains(classGateStringSlice(candidate, "denied_surfaces"), want) {
			blockers = append(blockers, "first non-planning candidate denied surfaces missing "+want)
			break
		}
	}
	for _, want := range []string{"CI failure", "Sentinel hold", "kill switch", "rollback failure", "unsafe scope drift", "RSI boundary crossing"} {
		if !classGateStringSliceContains(classGateStringSlice(rollback, "auto_stop_triggers"), want) {
			blockers = append(blockers, "first non-planning rollback auto-stop triggers missing "+want)
			break
		}
	}
	if classGateBool(rollback, "fully_unsupervised_complex_mutation_live_proven") {
		blockers = append(blockers, "first non-planning rollback must not claim fully_unsupervised_complex_mutation live-proven")
	}
	if classGateString(rollback, "rsi") != "denied" {
		blockers = append(blockers, "first non-planning rollback must keep RSI denied")
	}

	gate.SafeToRequest = gate.FoundryImportStatus != "" && ((gate.CandidateStatus == "ready" && !serializedAfterStopGate) || (gate.CandidateStatus == "blocked" && serializedAfterStopGate)) && nodeStatus == "ready" && len(tasks) == 1 && !gate.FoundryImportSchedulesWork && !gate.FoundryImportExecutesWork && !gate.FoundryImportApprovesWork
	if len(blockers) > 0 {
		gate.Blockers = blockers
		gate.FirstFailingCheck = blockers[0]
		return gate, nil
	}
	gate.Status = "ready"
	gate.SafeToRequest = true
	gate.SafeToExecute = true
	gate.LiveExecutionAuthority = true
	if serializedAfterStopGate {
		gate.ExactNextAction = "execute_exact_serialized_first_non_planning_node"
		gate.AuthorityBoundary = "blocked_until_predecessor_terminal_evidence_and_stop_gate_clear"
	} else {
		gate.ExactNextAction = "execute_exact_first_non_planning_support_docs_node"
	}
	gate.Blockers = []string{}
	return gate, nil
}
