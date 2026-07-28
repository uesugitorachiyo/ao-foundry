package cli

import (
	"strings"
)

func buildFullyUnsupervisedFirstNonPlanningStopGateClearance(paths fullyUnsupervisedFirstNonPlanningStopGatePaths) (map[string]any, map[string]any, error) {
	sources := []MutationClassGateEvidence{}
	workgraphSource, workgraph, err := readComplexNodeGateObject("atlas_workgraph_after_complete", paths.Workgraph)
	if err != nil {
		return nil, nil, err
	}
	stopGateSource, stopGateGraph, err := readComplexNodeGateObject("atlas_stop_gate_graph", paths.StopGateGraph)
	if err != nil {
		return nil, nil, err
	}
	nodeGateSource, nodeGate, err := readComplexNodeGateObject("foundry_node_gate", paths.NodeGate)
	if err != nil {
		return nil, nil, err
	}
	runLinkSource, runLink, err := readComplexNodeGateObject("atlas_run_link", paths.RunLink)
	if err != nil {
		return nil, nil, err
	}
	rollbackSource, rollback, err := readComplexNodeGateObject("rollback_record", paths.Rollback)
	if err != nil {
		return nil, nil, err
	}
	sources = append(sources, workgraphSource, stopGateSource, nodeGateSource, runLinkSource, rollbackSource)
	var sentinel map[string]any
	if strings.TrimSpace(paths.Sentinel) != "" {
		sentinelSource, sentinelDocument, err := readComplexNodeGateObject("sentinel_stop_gate_verdict", paths.Sentinel)
		if err != nil {
			return nil, nil, err
		}
		sources = append(sources, sentinelSource)
		sentinel = sentinelDocument
	}
	var promoter map[string]any
	if strings.TrimSpace(paths.Promoter) != "" {
		promoterSource, promoterDocument, err := readComplexNodeGateObject("promoter_stop_gate_verdict", paths.Promoter)
		if err != nil {
			return nil, nil, err
		}
		sources = append(sources, promoterSource)
		promoter = promoterDocument
	}
	var commandReadback map[string]any
	if strings.TrimSpace(paths.CommandReadback) != "" {
		commandSource, commandDocument, err := readComplexNodeGateObject("command_stop_gate_readback", paths.CommandReadback)
		if err != nil {
			return nil, nil, err
		}
		sources = append(sources, commandSource)
		commandReadback = commandDocument
	}

	afterNode := classGateString(nodeGate, "node_id")
	taskID := classGateString(nodeGate, "task_id")
	workgraphID := classGateString(workgraph, "id")
	var gate map[string]any
	for _, candidate := range classGateObjectSlice(stopGateGraph["stop_gates"]) {
		if classGateString(candidate, "after_node") == afterNode {
			gate = candidate
			break
		}
	}
	stopGateID := classGateString(gate, "id")
	beforeNode := classGateString(gate, "before_node")
	blockers := []string{}
	if classGateString(stopGateGraph, "schema") != "ao.atlas.private-first-non-planning-stop-gate-graph.v0.1" ||
		classGateString(stopGateGraph, "workgraph_id") != workgraphID {
		blockers = append(blockers, "stop-gate graph must bind to the completed workgraph")
	}
	if stopGateID == "" || beforeNode == "" {
		blockers = append(blockers, "stop-gate graph must contain a gate for predecessor node")
	}
	if classGateString(nodeGate, "status") != "ready" || !classGateBool(nodeGate, "safe_to_execute") || !classGateBool(nodeGate, "live_execution_authority") {
		blockers = append(blockers, "predecessor node gate must have been safe_to_execute before execution")
	}
	if classGateString(nodeGate, "highest_proven_live_class") != "complex_repo_mutation" ||
		classGateString(nodeGate, "next_denied_class") != "fully_unsupervised_complex_mutation" ||
		classGateString(nodeGate, "fully_unsupervised_complex_mutation") != "denied" ||
		classGateString(nodeGate, "rsi") != "denied" {
		blockers = append(blockers, "predecessor node gate must preserve class denial boundaries")
	}
	if classGateString(runLink, "contract_version") != atlasRunLinkSchema || classGateString(runLink, "status") != "completed" {
		blockers = append(blockers, "run-link must be completed terminal evidence")
	}
	if taskID == "" || classGateString(runLink, "task_id") != taskID {
		blockers = append(blockers, "run-link task_id must match predecessor node gate")
	}
	runLinkEvidence := classGateObject(runLink["evidence"])
	if !statusPassed(classGateString(runLinkEvidence, "ci")) {
		blockers = append(blockers, "run-link CI evidence must be passed")
	}
	if classGateString(runLinkEvidence, "pr") == "" || classGateString(runLinkEvidence, "merge_commit") == "" || classGateString(runLinkEvidence, "changed_file") == "" {
		blockers = append(blockers, "run-link must include PR, merge commit, and changed file evidence")
	}
	if classGateString(rollback, "node_id") != afterNode || len(classGateStringSlice(rollback, "rollback_scope")) == 0 ||
		len(classGateStringSlice(rollback, "rollback_plan")) == 0 {
		blockers = append(blockers, "rollback record must bind to predecessor node")
	}
	if classGateBool(rollback, "fully_unsupervised_complex_mutation_live_proven") || classGateString(rollback, "rsi") != "denied" {
		blockers = append(blockers, "rollback record must preserve denied class boundaries")
	}
	if sentinel == nil {
		blockers = append(blockers, "Sentinel clear evidence is required")
	} else if classGateString(sentinel, "node_id") != afterNode || classGateString(sentinel, "stop_gate_id") != stopGateID || classGateString(sentinel, "status") != "clear" {
		blockers = append(blockers, "Sentinel evidence must be clear and bind to the stop gate")
	} else if classGateString(sentinel, "fully_unsupervised_complex_mutation") != "denied" || classGateString(sentinel, "rsi") != "denied" {
		blockers = append(blockers, "Sentinel evidence must preserve denied class boundaries")
	}
	if promoter == nil {
		blockers = append(blockers, "Promoter no-promotion evidence is required")
	} else if classGateString(promoter, "node_id") != afterNode || classGateString(promoter, "stop_gate_id") != stopGateID ||
		classGateString(promoter, "verdict") != "no_promotion" {
		blockers = append(blockers, "Promoter evidence must be no_promotion")
	} else if classGateString(promoter, "fully_unsupervised_complex_mutation") != "denied" || classGateString(promoter, "rsi") != "denied" {
		blockers = append(blockers, "Promoter evidence must preserve denied class boundaries")
	}
	if commandReadback == nil {
		blockers = append(blockers, "Command readback evidence is required")
	} else if classGateString(commandReadback, "node_id") != afterNode || classGateString(commandReadback, "stop_gate_id") != stopGateID ||
		classGateString(commandReadback, "status") != "accepted" || !classGateBool(commandReadback, "safe_to_continue") {
		blockers = append(blockers, "Command readback must accept the stop-gate continuation")
	} else if classGateString(commandReadback, "fully_unsupervised_complex_mutation") != "denied" || classGateString(commandReadback, "rsi") != "denied" {
		blockers = append(blockers, "Command readback must preserve denied class boundaries")
	} else if !classGateBool(commandReadback, "no_reprompt_proof") || classGateString(commandReadback, "public_claim_guard") != "passed" {
		blockers = append(blockers, "Command readback must include no-reprompt proof and public claim guard")
	}

	nodes := classGateObjectSlice(workgraph["nodes"])
	afterCompleted := false
	beforeBlocked := false
	for _, node := range nodes {
		switch classGateString(node, "id") {
		case afterNode:
			afterCompleted = classGateString(node, "status") == "completed"
		case beforeNode:
			beforeBlocked = classGateString(node, "status") == "blocked"
		}
	}
	if !afterCompleted || !beforeBlocked {
		blockers = append(blockers, "workgraph must have predecessor completed and next node blocked before clearance")
	}

	clearance := map[string]any{
		"schema_version":                      firstNonPlanningStopGateClearanceSchema,
		"status":                              "blocked",
		"workgraph_id":                        workgraphID,
		"stop_gate_id":                        stopGateID,
		"after_node":                          afterNode,
		"before_node":                         beforeNode,
		"safe_to_continue":                    false,
		"first_failing_check":                 "",
		"blockers":                            blockers,
		"exact_next_action":                   "repair_first_non_planning_stop_gate_evidence_before_next_node",
		"highest_proven_live_class":           "complex_repo_mutation",
		"next_denied_class":                   "fully_unsupervised_complex_mutation",
		"fully_unsupervised_complex_mutation": "denied",
		"rsi":                                 "denied",
		"rollback_disposition":                "available",
		"sentinel_verdict":                    classGateString(sentinel, "status"),
		"promoter_verdict":                    classGateString(promoter, "verdict"),
		"command_readback":                    classGateString(commandReadback, "status"),
		"source_evidence":                     sources,
	}
	workgraphOut := map[string]any{}
	if len(blockers) > 0 {
		clearance["first_failing_check"] = blockers[0]
		return clearance, workgraphOut, nil
	}
	clearance["status"] = "ready"
	clearance["safe_to_continue"] = true
	clearance["blockers"] = []string{}
	clearance["exact_next_action"] = "import_next_first_non_planning_node_after_stop_gate_clearance"
	workgraphOut, err = cloneJSONMap(workgraph)
	if err != nil {
		return nil, nil, err
	}
	updatedNodes := classGateObjectSlice(workgraphOut["nodes"])
	rawNodes := make([]any, 0, len(updatedNodes))
	for _, node := range updatedNodes {
		if classGateString(node, "id") == beforeNode {
			node["status"] = "ready"
			node["blockers"] = []any{}
		}
		rawNodes = append(rawNodes, node)
	}
	workgraphOut["nodes"] = rawNodes
	return clearance, workgraphOut, nil
}
