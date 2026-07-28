package cli

import (
	"fmt"
	"path/filepath"
	"strings"
)

func buildFullyUnsupervisedFirstNonPlanningFinalClosure(paths fullyUnsupervisedFirstNonPlanningFinalClosurePaths) (map[string]any, map[string]any, map[string]any, map[string]any, error) {
	sources := []MutationClassGateEvidence{}
	workgraphSource, workgraph, err := readComplexNodeGateObject("final_workgraph", paths.Workgraph)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	sources = append(sources, workgraphSource)
	if strings.TrimSpace(paths.FinalSynthesis) != "" {
		finalSource, _, err := readComplexNodeGateObject("pre_execution_final_synthesis", paths.FinalSynthesis)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		sources = append(sources, finalSource)
	}

	nodes := classGateObjectSlice(workgraph["nodes"])
	totalNodes := len(nodes)
	completedNodes := countWorkgraphNodesWithStatus(nodes, "completed")
	readyNodes := countWorkgraphNodesWithStatus(nodes, "ready")
	blockedNodes := countWorkgraphNodesWithStatus(nodes, "blocked")
	blockers := []string{}
	nodeIDs := []string{}
	allNodeEvidence := totalNodes > 0
	allCI := true
	allPRMerged := true
	noForbiddenSurfaces := true
	rsiDenied := true
	everyStopGateCleared := totalNodes > 1

	if totalNodes != 26 {
		blockers = append(blockers, fmt.Sprintf("workgraph must contain 26 nodes, got %d", totalNodes))
	}
	if completedNodes != totalNodes || readyNodes != 0 || blockedNodes != 0 {
		blockers = append(blockers, "workgraph must have all nodes completed with zero ready or blocked nodes")
	}

	for _, node := range nodes {
		nodeID := classGateString(node, "id")
		if strings.TrimSpace(nodeID) == "" {
			allNodeEvidence = false
			blockers = append(blockers, "workgraph node missing id")
			continue
		}
		nodeIDs = append(nodeIDs, nodeID)
		if classGateString(node, "status") != "completed" {
			allNodeEvidence = false
			blockers = append(blockers, fmt.Sprintf("node %s must be completed before promotion", nodeID))
		}
		runLinkPath := filepath.Join(paths.RunLinksRoot, nodeID, "run-link.json")
		runLinkSource, runLink, found, err := readFullyUnsupervisedClosureObject("run-link "+nodeID, runLinkPath)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		if !found {
			allNodeEvidence = false
			blockers = append(blockers, fmt.Sprintf("run-link missing for %s", nodeID))
			continue
		}
		sources = append(sources, runLinkSource)
		if classGateString(runLink, "contract_version") != atlasRunLinkSchema || classGateString(runLink, "status") != "completed" {
			allNodeEvidence = false
			blockers = append(blockers, fmt.Sprintf("run-link %s must be completed terminal evidence", nodeID))
		}
		evidence := classGateObject(runLink["evidence"])
		if !statusPassed(classGateString(evidence, "ci")) {
			allCI = false
			blockers = append(blockers, fmt.Sprintf("run-link %s requires passed CI evidence", nodeID))
		}
		if classGateString(evidence, "pr") == "" || classGateString(evidence, "merge_commit") == "" {
			allPRMerged = false
			blockers = append(blockers, fmt.Sprintf("run-link %s requires PR and merge evidence", nodeID))
		}
		if classGateString(evidence, "changed_file") == "" {
			allNodeEvidence = false
			blockers = append(blockers, fmt.Sprintf("run-link %s requires changed file evidence", nodeID))
		}
		gatePath := classGateString(evidence, "node_gate")
		if strings.TrimSpace(gatePath) == "" {
			allNodeEvidence = false
			blockers = append(blockers, fmt.Sprintf("run-link %s requires node gate evidence", nodeID))
			continue
		}
		nodeGateSource, nodeGate, found, err := readFullyUnsupervisedClosureObject("node-gate "+nodeID, gatePath)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		if !found {
			allNodeEvidence = false
			blockers = append(blockers, fmt.Sprintf("node gate missing for %s", nodeID))
			continue
		}
		sources = append(sources, nodeGateSource)
		if classGateString(nodeGate, "schema_version") != complexNodeGateSchema ||
			classGateString(nodeGate, "status") != "ready" ||
			!classGateBool(nodeGate, "safe_to_request") ||
			!classGateBool(nodeGate, "safe_to_execute") ||
			!classGateBool(nodeGate, "live_execution_authority") {
			allNodeEvidence = false
			blockers = append(blockers, fmt.Sprintf("node gate %s must be ready and executable", nodeID))
		}
		if classGateBool(nodeGate, "schedules_work") ||
			classGateBool(nodeGate, "executes_work") ||
			classGateBool(nodeGate, "approves_work") ||
			classGateBool(nodeGate, "mutates_repositories") {
			noForbiddenSurfaces = false
			blockers = append(blockers, fmt.Sprintf("node gate %s expands forbidden authority", nodeID))
		}
		if classGateString(nodeGate, "rsi") != "denied" {
			rsiDenied = false
			blockers = append(blockers, fmt.Sprintf("node gate %s must keep RSI denied", nodeID))
		}
	}

	for i := 1; i < len(nodeIDs); i++ {
		afterNode := nodeIDs[i-1]
		beforeNode := nodeIDs[i]
		stopGateID := fmt.Sprintf("stop-gate-%02d-%s-to-%s", i, afterNode, beforeNode)
		stopGateDir := filepath.Join(paths.StopGatesRoot, stopGateID)
		clearancePath := filepath.Join(stopGateDir, "stop-gate-clearance.json")
		clearanceSource, clearance, found, err := readFullyUnsupervisedClosureObject("stop-gate clearance "+stopGateID, clearancePath)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		if !found {
			everyStopGateCleared = false
			blockers = append(blockers, fmt.Sprintf("stop-gate clearance missing for %s", stopGateID))
			continue
		}
		sources = append(sources, clearanceSource)
		if classGateString(clearance, "schema_version") != firstNonPlanningStopGateClearanceSchema ||
			classGateString(clearance, "status") != "ready" ||
			!classGateBool(clearance, "safe_to_continue") ||
			classGateString(clearance, "after_node") != afterNode ||
			classGateString(clearance, "before_node") != beforeNode {
			everyStopGateCleared = false
			blockers = append(blockers, fmt.Sprintf("stop-gate clearance must be ready for %s", stopGateID))
		}
		if classGateString(clearance, "rsi") != "denied" {
			rsiDenied = false
			blockers = append(blockers, fmt.Sprintf("stop-gate clearance %s must keep RSI denied", stopGateID))
		}
		sentinelSource, sentinel, found, err := readFullyUnsupervisedClosureObject("Sentinel verdict "+stopGateID, filepath.Join(stopGateDir, "sentinel-verdict.json"))
		if err != nil {
			return nil, nil, nil, nil, err
		}
		if !found {
			everyStopGateCleared = false
			blockers = append(blockers, fmt.Sprintf("Sentinel verdict missing for %s", stopGateID))
		} else {
			sources = append(sources, sentinelSource)
			if classGateString(sentinel, "status") != "clear" || classGateString(sentinel, "rsi") != "denied" ||
				classGateString(sentinel, "fully_unsupervised_complex_mutation") != "denied" {
				everyStopGateCleared = false
				blockers = append(blockers, fmt.Sprintf("Sentinel verdict must be clear for %s", stopGateID))
			}
		}
		promoterSource, promoter, found, err := readFullyUnsupervisedClosureObject("Promoter verdict "+stopGateID, filepath.Join(stopGateDir, "promoter-verdict.json"))
		if err != nil {
			return nil, nil, nil, nil, err
		}
		if !found {
			everyStopGateCleared = false
			blockers = append(blockers, fmt.Sprintf("Promoter verdict missing for %s", stopGateID))
		} else {
			sources = append(sources, promoterSource)
			if classGateString(promoter, "verdict") != "no_promotion" || classGateString(promoter, "rsi") != "denied" ||
				classGateString(promoter, "fully_unsupervised_complex_mutation") != "denied" {
				everyStopGateCleared = false
				blockers = append(blockers, fmt.Sprintf("Promoter verdict must be no_promotion for %s", stopGateID))
			}
		}
		commandSource, commandReadback, found, err := readFullyUnsupervisedClosureObject("Command readback "+stopGateID, filepath.Join(stopGateDir, "command-readback.json"))
		if err != nil {
			return nil, nil, nil, nil, err
		}
		if !found {
			everyStopGateCleared = false
			blockers = append(blockers, fmt.Sprintf("Command readback missing for %s", stopGateID))
		} else {
			sources = append(sources, commandSource)
			if classGateString(commandReadback, "status") != "accepted" ||
				!classGateBool(commandReadback, "safe_to_continue") ||
				classGateString(commandReadback, "rsi") != "denied" ||
				classGateString(commandReadback, "fully_unsupervised_complex_mutation") != "denied" {
				everyStopGateCleared = false
				blockers = append(blockers, fmt.Sprintf("Command readback must agree for %s", stopGateID))
			}
		}
	}

	blockers = uniqueStrings(blockers)
	promoted := len(blockers) == 0
	highestProvenLiveClass := "complex_repo_mutation"
	nextDeniedClass := "fully_unsupervised_complex_mutation"
	if promoted {
		highestProvenLiveClass = "fully_unsupervised_complex_mutation"
		nextDeniedClass = "RSI"
	}
	status := "blocked"
	verdict := "no_promotion"
	commandDecision := "deny_promotion"
	exactNextAction := "repair_final_closure_evidence_before_promotion"
	if promoted {
		status = "promoted"
		verdict = "promote"
		commandDecision = "promote_fully_unsupervised_complex_mutation"
		exactNextAction = "record_canonical_promotion_and_keep_rsi_denied"
	}

	missionCompletion := map[string]any{
		"schema":                  firstNonPlanningMissionCompletionSchema,
		"status":                  classGateFirstNonEmpty(mapStatus(promoted, "completed", "blocked"), status),
		"mission":                 "fully_unsupervised_complex_mutation_first_non_planning_rehearsal",
		"total_nodes":             float64(totalNodes),
		"completed_nodes":         float64(completedNodes),
		"blocked_nodes":           float64(blockedNodes),
		"ready_nodes":             float64(readyNodes),
		"every_node_has_evidence": allNodeEvidence && promoted,
		"every_stop_gate_cleared": everyStopGateCleared && promoted,
		"all_node_prs_merged":     allPRMerged && promoted,
		"all_ci_passed":           allCI && promoted,
		"branch_cleanup_complete": promoted,
		"no_concurrent_mutation":  promoted,
		"no_forbidden_surfaces":   noForbiddenSurfaces && promoted,
		"fully_unsupervised_complex_mutation_live_proven": promoted,
		"highest_proven_live_class":                       highestProvenLiveClass,
		"next_denied_class":                               nextDeniedClass,
		"rsi":                                             "denied",
		"rsi_denied":                                      rsiDenied && promoted,
		"node_ids":                                        nodeIDs,
		"blockers":                                        blockers,
		"source_evidence":                                 sources,
		"generated_at":                                    nowUTC(),
	}
	rollup := map[string]any{
		"schema":          firstNonPlanningFinalRollupSchema,
		"status":          status,
		"safe_to_promote": promoted,
		"fully_unsupervised_complex_mutation_live_proven": promoted,
		"highest_proven_live_class":                       highestProvenLiveClass,
		"next_denied_class":                               nextDeniedClass,
		"rsi":                                             "denied",
		"total_nodes":                                     float64(totalNodes),
		"completed_nodes":                                 float64(completedNodes),
		"blocked_nodes":                                   float64(blockedNodes),
		"ready_nodes":                                     float64(readyNodes),
		"every_node_has_evidence":                         allNodeEvidence && promoted,
		"every_stop_gate_cleared":                         everyStopGateCleared && promoted,
		"all_node_prs_merged":                             allPRMerged && promoted,
		"all_ci_passed":                                   allCI && promoted,
		"branch_cleanup_complete":                         promoted,
		"no_concurrent_mutation":                          promoted,
		"no_forbidden_surfaces":                           noForbiddenSurfaces && promoted,
		"rsi_denied":                                      rsiDenied && promoted,
		"final_synthesis_reevaluated":                     strings.TrimSpace(paths.FinalSynthesis) != "",
		"first_failing_check":                             "",
		"blockers":                                        blockers,
		"exact_next_action":                               exactNextAction,
		"source_evidence":                                 sources,
		"generated_at":                                    nowUTC(),
	}
	if len(blockers) > 0 {
		rollup["first_failing_check"] = blockers[0]
	}
	promoter := map[string]any{
		"schema":  firstNonPlanningPromoterFinalSchema,
		"status":  "accepted",
		"verdict": verdict,
		"fully_unsupervised_complex_mutation_live_proven": promoted,
		"highest_proven_live_class":                       highestProvenLiveClass,
		"next_denied_class":                               nextDeniedClass,
		"rsi":                                             "denied",
		"safe_to_promote":                                 promoted,
		"blockers":                                        blockers,
		"generated_at":                                    nowUTC(),
	}
	commandReadback := map[string]any{
		"schema":   firstNonPlanningCommandFinalSchema,
		"status":   "accepted",
		"decision": commandDecision,
		"fully_unsupervised_complex_mutation_live_proven": promoted,
		"highest_proven_live_class":                       highestProvenLiveClass,
		"next_denied_class":                               nextDeniedClass,
		"rsi":                                             "denied",
		"safe_to_promote":                                 promoted,
		"blockers":                                        blockers,
		"exact_next_action":                               exactNextAction,
		"generated_at":                                    nowUTC(),
	}
	return rollup, missionCompletion, promoter, commandReadback, nil
}

func mapStatus(condition bool, trueValue, falseValue string) string {
	if condition {
		return trueValue
	}
	return falseValue
}
