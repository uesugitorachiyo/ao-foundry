package cli

import (
	"path/filepath"
)

func buildFullyUnsupervisedReadinessRollup(paths fullyUnsupervisedReadinessPaths) (map[string]any, error) {
	sources := []MutationClassGateEvidence{}
	blueprintSource, blueprintImport, err := readComplexNodeGateObject("atlas_blueprint_import", paths.BlueprintImport)
	if err != nil {
		return nil, err
	}
	workgraphSource, workgraph, err := readComplexNodeGateObject("atlas_workgraph", paths.Workgraph)
	if err != nil {
		return nil, err
	}
	foundryImportSource, foundryImport, err := readComplexNodeGateObject("foundry_import", paths.FoundryImport)
	if err != nil {
		return nil, err
	}
	summarySource, summary, err := readComplexNodeGateObject("atlas_first_summary", paths.AtlasSummary)
	if err != nil {
		return nil, err
	}
	manifestSource, manifest, err := readComplexNodeGateObject("sdd_slice_completion_manifest", paths.SliceManifest)
	if err != nil {
		return nil, err
	}
	finalSource, finalSynthesis, err := readComplexNodeGateObject("atlas_final_synthesis", paths.FinalSynthesis)
	if err != nil {
		return nil, err
	}
	gateSource, firstGateObject, err := readComplexNodeGateObject("first_node_gate", paths.FirstNodeGate)
	if err != nil {
		return nil, err
	}
	sources = append(sources, blueprintSource, workgraphSource, foundryImportSource, summarySource, manifestSource, finalSource, gateSource)

	nodes := classGateObjectSlice(workgraph["nodes"])
	totalNodes := len(nodes)
	readyNodes := 0
	blockedNodes := 0
	nodeSummaries := []map[string]any{}
	blockers := []string{}
	checks := map[string]bool{
		"blueprint_import_ready":         false,
		"foundry_import_first_node_only": false,
		"slice_manifest_complete":        false,
		"atlas_summary_matches":          false,
		"final_synthesis_denies_class":   false,
		"all_node_artifacts_present":     false,
		"all_nodes_non_executable":       false,
		"first_node_gate_blocks_live":    false,
		"forbidden_surfaces_clear":       false,
		"command_readback_ready":         false,
	}
	if classGateString(blueprintImport, "contract_version") == atlasBlueprintImportSchema &&
		classGateString(blueprintImport, "status") == "ready" &&
		classGateString(blueprintImport, "mutation_class") == "complex_repo_mutation" &&
		!classGateBool(blueprintImport, "safe_to_execute") &&
		!classGateBool(blueprintImport, "live_execution_proven") {
		checks["blueprint_import_ready"] = true
	} else {
		blockers = append(blockers, "Atlas Blueprint import must be ready, complex_repo_mutation, and non-executable")
	}
	foundryTasks := classGateObjectSlice(foundryImport["tasks"])
	firstImportedNode := ""
	if len(foundryTasks) == 1 {
		firstImportedNode = classGateString(foundryTasks[0], "node_id")
	}
	if classGateString(foundryImport, "contract_version") == atlasImportSchema &&
		classGateString(foundryImport, "status") == "ready_for_foundry_fixture_import" &&
		len(foundryTasks) == 1 &&
		!classGateBool(foundryImport, "schedules_work") &&
		!classGateBool(foundryImport, "executes_work") &&
		!classGateBool(foundryImport, "approves_work") {
		checks["foundry_import_first_node_only"] = true
	} else {
		blockers = append(blockers, "Foundry import must contain exactly one non-executing first node")
	}
	totalSlices := int(classGateNumber(manifest, "total_slices"))
	completedSlices := int(classGateNumber(manifest, "completed_slices"))
	if classGateString(manifest, "schema") == "ao.atlas.private-sdd-slice-completion-manifest.v0.1" &&
		totalSlices > 0 &&
		completedSlices == totalSlices &&
		classGateString(manifest, "rsi") == "denied" {
		checks["slice_manifest_complete"] = true
	} else {
		blockers = append(blockers, "SDD slice completion manifest must be complete and keep RSI denied")
	}
	if int(classGateNumber(summary, "planned_node_count")) == totalNodes &&
		int(classGateNumber(summary, "ready_node_count")) == countWorkgraphNodesWithStatus(nodes, "ready") &&
		int(classGateNumber(summary, "blocked_node_count")) == countWorkgraphNodesWithStatus(nodes, "blocked") &&
		classGateString(summary, "highest_proven_live_class") == "complex_repo_mutation" &&
		classGateString(summary, "next_denied_class") == "fully_unsupervised_complex_mutation" &&
		!classGateBool(summary, "safe_to_execute") &&
		!classGateBool(summary, "fully_unsupervised_complex_mutation_live_proven") &&
		classGateString(summary, "rsi") == "denied" {
		checks["atlas_summary_matches"] = true
	} else {
		blockers = append(blockers, "Atlas summary must match workgraph counts and deny fully unsupervised execution")
	}
	if classGateString(finalSynthesis, "schema") == "ao.atlas.private-fully-unsupervised-readiness-final-synthesis.v0.1" &&
		classGateString(finalSynthesis, "status") == "atlas_first_readiness_phase_complete" &&
		classGateString(finalSynthesis, "highest_proven_live_class") == "complex_repo_mutation" &&
		classGateString(finalSynthesis, "next_denied_class") == "fully_unsupervised_complex_mutation" &&
		classGateString(finalSynthesis, "rsi") == "denied" {
		checks["final_synthesis_denies_class"] = true
	} else {
		blockers = append(blockers, "Atlas final synthesis must preserve fully_unsupervised_complex_mutation and RSI denial")
	}
	if classGateString(firstGateObject, "schema_version") == complexNodeGateSchema &&
		classGateString(firstGateObject, "status") == "blocked" &&
		classGateBool(firstGateObject, "safe_to_request") &&
		!classGateBool(firstGateObject, "safe_to_execute") &&
		!classGateBool(firstGateObject, "live_execution_authority") &&
		classGateString(firstGateObject, "fully_unsupervised_complex_mutation") == "denied" &&
		classGateString(firstGateObject, "rsi") == "denied" {
		checks["first_node_gate_blocks_live"] = true
	} else {
		blockers = append(blockers, "first node gate must block live execution while allowing request readback")
	}
	if firstImportedNode != "" && classGateString(firstGateObject, "node_id") != "" && firstImportedNode != classGateString(firstGateObject, "node_id") {
		blockers = append(blockers, "Foundry import selected node must match first node gate")
	}

	allArtifactsPresent := totalNodes > 0
	allNonExecutable := totalNodes > 0
	for _, node := range nodes {
		nodeID := classGateString(node, "id")
		nodeStatus := classGateString(node, "status")
		task := map[string]any{}
		if raw, ok := node["factory_task"].(map[string]any); ok {
			task = raw
		}
		taskID := classGateFirstNonEmpty(classGateString(task, "id"), nodeID+"-task")
		if nodeStatus == "ready" {
			readyNodes++
		}
		if nodeStatus == "blocked" {
			blockedNodes++
		}
		summaryNode, present, nonExecutable, nodeSources, nodeBlockers := evaluateFullyUnsupervisedNodeReadiness(paths, nodeID, taskID, nodeStatus, task)
		sources = append(sources, nodeSources...)
		nodeSummaries = append(nodeSummaries, summaryNode)
		if !present {
			allArtifactsPresent = false
		}
		if !nonExecutable {
			allNonExecutable = false
		}
		blockers = append(blockers, nodeBlockers...)
	}
	checks["all_node_artifacts_present"] = allArtifactsPresent
	checks["all_nodes_non_executable"] = allNonExecutable
	checks["forbidden_surfaces_clear"] = allNonExecutable &&
		!classGateBool(blueprintImport, "schedules_work") &&
		!classGateBool(blueprintImport, "executes_work") &&
		!classGateBool(blueprintImport, "approves_work") &&
		!classGateBool(blueprintImport, "mutates_repositories") &&
		!classGateBool(blueprintImport, "calls_providers") &&
		!classGateBool(blueprintImport, "release_or_publish_allowed")
	checks["command_readback_ready"] = len(nodeSummaries) == totalNodes && totalNodes > 0 && allArtifactsPresent

	uniqueBlockers := uniqueStrings(blockers)
	status := "denied"
	if len(uniqueBlockers) > 0 {
		status = "blocked"
	}
	commandReadback := map[string]any{
		"schema_version":                      fullyUnsupervisedCommandSchema,
		"status":                              "ready",
		"operator_mode":                       "read_only",
		"highest_proven_live_class":           "complex_repo_mutation",
		"next_denied_class":                   "fully_unsupervised_complex_mutation",
		"fully_unsupervised_complex_mutation": "denied",
		"rsi":                                 "denied",
		"safe_to_request":                     true,
		"safe_to_execute":                     false,
		"live_execution_authority":            false,
		"nodes_consumed":                      len(nodeSummaries),
		"total_nodes":                         totalNodes,
		"first_node_gate":                     filepath.ToSlash(paths.FirstNodeGate),
	}
	rollup := map[string]any{
		"schema_version":                      fullyUnsupervisedReadinessSchema,
		"status":                              status,
		"mutation_class":                      "complex_repo_mutation",
		"target_class":                        "fully_unsupervised_complex_mutation",
		"highest_proven_live_class":           "complex_repo_mutation",
		"next_denied_class":                   "fully_unsupervised_complex_mutation",
		"fully_unsupervised_complex_mutation": "denied",
		"rsi":                                 "denied",
		"safe_to_request":                     true,
		"safe_to_execute":                     false,
		"live_execution_authority":            false,
		"safe_to_promote":                     false,
		"total_nodes":                         totalNodes,
		"ready_nodes":                         readyNodes,
		"blocked_nodes":                       blockedNodes,
		"nodes_consumed":                      len(nodeSummaries),
		"node_evidence":                       nodeSummaries,
		"checks":                              checks,
		"blockers":                            uniqueBlockers,
		"first_failing_check":                 "",
		"exact_next_action":                   "fully_unsupervised_complex_mutation_remains_denied_until_non_planning_gates_authorize_execution",
		"foundry_continuation_handoff_result": "validated_first_node_import_and_consumed_all_atlas_readiness_records",
		"command_readback":                    commandReadback,
		"source_evidence":                     sources,
		"evaluated_at_utc":                    nowUTC(),
	}
	if len(uniqueBlockers) > 0 {
		rollup["first_failing_check"] = uniqueBlockers[0]
		rollup["exact_next_action"] = "repair_fully_unsupervised_readiness_evidence_before_requesting_any_live_authority"
		commandReadback["status"] = "blocked"
	}
	return rollup, nil
}

func evaluateFullyUnsupervisedNodeReadiness(paths fullyUnsupervisedReadinessPaths, nodeID, taskID, nodeStatus string, workgraphTask map[string]any) (map[string]any, bool, bool, []MutationClassGateEvidence, []string) {
	sources := []MutationClassGateEvidence{}
	blockers := []string{}
	present := true
	nonExecutable := true
	node := map[string]any{
		"node_id":          nodeID,
		"task_id":          taskID,
		"workgraph_status": nodeStatus,
		"candidate":        "missing",
		"rollback":         "missing",
		"node_evidence":    "missing",
		"repair_plan":      "missing",
		"context_repack":   "missing",
		"task":             "missing",
		"context_pack":     "missing",
		"safe_to_execute":  false,
	}
	if nodeID == "" {
		return node, false, false, sources, []string{"workgraph node missing id"}
	}
	checkFile := func(role, root, filename, schema string) (map[string]any, bool) {
		path := filepath.Join(root, filename)
		source, object, err := readComplexNodeGateObject(role+":"+nodeID, path)
		if err != nil {
			return nil, false
		}
		sources = append(sources, source)
		if schema != "" && classGateString(object, "schema") != schema && classGateString(object, "contract_version") != schema {
			blockers = append(blockers, role+" "+nodeID+" schema mismatch")
			return object, false
		}
		if objectNodeID := classGateString(object, "node_id"); objectNodeID != "" && objectNodeID != nodeID {
			blockers = append(blockers, role+" "+nodeID+" node_id mismatch")
			return object, false
		}
		if objectTaskID := classGateString(object, "task_id"); objectTaskID != "" && taskID != "" && objectTaskID != taskID {
			blockers = append(blockers, role+" "+nodeID+" task_id mismatch")
			return object, false
		}
		return object, true
	}
	candidate, candidateOK := checkFile("candidate", paths.CandidateRoot, nodeID+"-candidate.json", "ao.atlas.private-candidate-record.v0.1")
	rollback, rollbackOK := checkFile("rollback", paths.RollbackRoot, nodeID+"-rollback.json", "ao.atlas.private-rollback-record.v0.1")
	evidence, evidenceOK := checkFile("node_evidence", paths.NodeEvidenceRoot, nodeID+"-completion-evidence.json", "ao.atlas.private-node-evidence.v0.1")
	repair, repairOK := checkFile("repair_plan", paths.RepairRoot, nodeID+"-repair-plan.json", "ao.atlas.private-repair-plan.v0.1")
	repack, repackOK := checkFile("context_repack", paths.RepackRoot, nodeID+"-context-repack-plan.json", "ao.atlas.private-context-repack-plan.v0.1")
	task, taskOK := checkFile("task", paths.TaskRoot, nodeID+"-task.json", atlasTaskSchema)
	context, contextOK := checkFile("context_pack", paths.ContextRoot, nodeID+"-context.json", "")

	for role, ok := range map[string]bool{
		"candidate": candidateOK, "rollback": rollbackOK, "node_evidence": evidenceOK,
		"repair_plan": repairOK, "context_repack": repackOK, "task": taskOK, "context_pack": contextOK,
	} {
		if ok {
			node[role] = "present"
		} else {
			present = false
			blockers = append(blockers, role+" "+nodeID+" is missing or invalid")
		}
	}
	candidateStatus := classGateString(candidate, "status")
	expectedCandidateStatus := "blocked"
	if nodeStatus == "ready" {
		expectedCandidateStatus = "ready"
	}
	if candidateOK && candidateStatus != expectedCandidateStatus {
		blockers = append(blockers, "candidate "+nodeID+" status must match workgraph readiness")
	}
	if candidateOK && classGateBool(candidate, "safe_to_execute") {
		nonExecutable = false
		blockers = append(blockers, "candidate "+nodeID+" must keep safe_to_execute=false")
	}
	if candidateOK && !classGateStringSliceContains(classGateStringSlice(candidate, "denied_boundaries"), "RSI") {
		blockers = append(blockers, "candidate "+nodeID+" must deny RSI")
	}
	if rollbackOK && classGateBool(rollback, "safe_to_execute") {
		nonExecutable = false
		blockers = append(blockers, "rollback "+nodeID+" must keep safe_to_execute=false")
	}
	if evidenceOK && classGateBool(evidence, "safe_to_execute") {
		nonExecutable = false
		blockers = append(blockers, "node evidence "+nodeID+" must keep safe_to_execute=false")
	}
	if repairOK && classGateBool(repair, "safe_to_execute") {
		nonExecutable = false
		blockers = append(blockers, "repair plan "+nodeID+" must keep safe_to_execute=false")
	}
	if repackOK && classGateBool(repack, "safe_to_execute") {
		nonExecutable = false
		blockers = append(blockers, "context repack "+nodeID+" must keep safe_to_execute=false")
	}
	taskMutationClass := classGateFirstNonEmpty(classGateString(task, "mutation_class"), classGateString(workgraphTask, "mutation_class"))
	if taskOK && taskMutationClass != "complex_repo_mutation" {
		blockers = append(blockers, "task "+nodeID+" must remain complex_repo_mutation planning evidence")
	}
	if contextOK {
		node["context_pack_status"] = classGateFirstNonEmpty(classGateString(context, "status"), "present")
	}
	node["candidate_status"] = candidateStatus
	node["node_class"] = classGateString(candidate, "node_class")
	node["safe_to_execute"] = false
	node["non_executable"] = nonExecutable
	return node, present, nonExecutable, sources, blockers
}
