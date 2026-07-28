package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

type complexPromotionRollupPaths struct {
	Mission       string
	Workgraph     string
	RunLinksRoot  string
	ClosureRoot   string
	NodeGates     []string
	FinalNodeGate string
}

func buildComplexRepoMutationPromotionRollup(paths complexPromotionRollupPaths) (ComplexRepoMutationPromotionRollup, error) {
	rollup := ComplexRepoMutationPromotionRollup{
		SchemaVersion:                    complexPromotionRollupSchema,
		Status:                           "blocked",
		MutationClass:                    "complex_repo_mutation",
		HighestProvenLiveClass:           "multi_repo_low_risk",
		NextDeniedClass:                  "complex_repo_mutation",
		FullyUnsupervisedComplexMutation: "denied",
		RSI:                              "denied",
		Blockers:                         []string{},
		Checks: map[string]bool{
			"all_nodes_completed":            false,
			"run_links_complete":             false,
			"node_gates_safe":                false,
			"no_concurrent_mutation":         false,
			"pr_ci_merge_evidence":           false,
			"rollback_evidence":              false,
			"sentinel_evidence":              false,
			"promoter_evidence":              false,
			"command_readback":               false,
			"atlas_final_workgraph_complete": false,
			"bounded_authority":              false,
			"forbidden_surfaces_clear":       false,
		},
		AuthorityBoundaries: map[string]bool{
			"schedules_work":                      false,
			"executes_work":                       false,
			"approves_work":                       false,
			"mutates_repositories":                false,
			"release_or_publish_allowed":          false,
			"provider_calls_allowed":              false,
			"credential_or_secret_access_allowed": false,
			"fully_unsupervised_claimed":          false,
			"rsi_claimed":                         false,
		},
		PublicWordingReview: "complex_repo_mutation may be marked live-proven only for this governed 12-node rehearsal; fully unsupervised complex mutation and RSI remain denied.",
		EvaluatedAtUTC:      nowUTC(),
	}
	missionSource, mission, err := readComplexNodeGateObject("mission_continuation_evidence", paths.Mission)
	if err != nil {
		return rollup, err
	}
	workgraphSource, workgraph, err := readComplexNodeGateObject("atlas_final_workgraph", paths.Workgraph)
	if err != nil {
		return rollup, err
	}
	rollup.SourceEvidence = append(rollup.SourceEvidence, missionSource, workgraphSource)
	rollup.Mission = classGateString(mission, "mission")
	nodes := classGateObjectSlice(workgraph["nodes"])
	rollup.TotalNodes = len(nodes)
	rollup.CompletedNodes = int(classGateNumber(mission, "completed_nodes"))
	blockers := []string{}
	if classGateString(mission, "schema") != "ao.atlas.private-mission-continuation-evidence.v0.1" {
		blockers = append(blockers, "mission continuation evidence schema mismatch")
	}
	if classGateString(mission, "status") != "all_nodes_completed_with_foundry_evidence" {
		blockers = append(blockers, "mission continuation evidence must report all nodes completed")
	}
	if rollup.CompletedNodes != rollup.TotalNodes || int(classGateNumber(mission, "total_atlas_nodes")) != rollup.TotalNodes {
		blockers = append(blockers, "mission completed node count must match final workgraph")
	}
	completedIDs := stringSet(classGateStringSlice(mission, "completed_node_ids"))
	workgraphComplete := len(nodes) > 0
	nodeIDs := []string{}
	for _, node := range nodes {
		nodeID := classGateString(node, "id")
		nodeIDs = append(nodeIDs, nodeID)
		if nodeID == "" {
			workgraphComplete = false
			blockers = append(blockers, "workgraph node missing id")
			continue
		}
		if classGateString(node, "status") != "completed" {
			workgraphComplete = false
			blockers = append(blockers, "workgraph node "+nodeID+" must be completed")
		}
		if !completedIDs[nodeID] {
			workgraphComplete = false
			blockers = append(blockers, "mission completed_node_ids missing "+nodeID)
		}
	}
	rollup.Checks["all_nodes_completed"] = rollup.CompletedNodes == rollup.TotalNodes && rollup.TotalNodes > 0 && workgraphComplete
	rollup.Checks["atlas_final_workgraph_complete"] = workgraphComplete
	if len(classGateObjectSlice(mission["blocked_nodes"])) == 0 &&
		classGateString(mission, "active_node") == "" &&
		int(classGateNumber(mission, "executable_node_count")) == 0 {
		rollup.Checks["no_concurrent_mutation"] = true
	}
	gateOverrides, overrideEvidence, err := loadComplexPromotionNodeGateOverrides(paths.NodeGates, paths.FinalNodeGate)
	if err != nil {
		return rollup, err
	}
	rollup.SourceEvidence = append(rollup.SourceEvidence, overrideEvidence...)
	runLinksOK := true
	nodeGatesOK := true
	prCIOK := true
	rollbackOK := true
	sentinelOK := true
	promoterOK := true
	commandOK := true
	boundedOK := true
	for _, nodeID := range nodeIDs {
		runLinkPath := filepath.Join(paths.RunLinksRoot, nodeID, "run-link.json")
		runLinkSource, runLink, err := readComplexNodeGateObject("run_link:"+nodeID, runLinkPath)
		if err != nil {
			runLinksOK = false
			blockers = append(blockers, "run-link "+nodeID+" is missing")
			continue
		}
		rollup.SourceEvidence = append(rollup.SourceEvidence, runLinkSource)
		evidence, _ := runLink["evidence"].(map[string]any)
		nodeSummary := ComplexRepoMutationRollupNode{
			NodeID:           nodeID,
			TaskID:           classGateString(runLink, "task_id"),
			Status:           classGateString(runLink, "status"),
			ChangedFile:      classGateString(evidence, "changed_file"),
			PullRequest:      classGateString(evidence, "pr"),
			MergeCommit:      classGateString(evidence, "merge_commit"),
			CI:               classGateString(evidence, "ci"),
			NodeGatePath:     classGateString(evidence, "node_gate"),
			RunLinkPath:      runLinkPath,
			RunLinkSHA256:    runLinkSource.SHA256,
			RollbackEvidence: classGateString(evidence, "rollback"),
			SentinelEvidence: classGateString(evidence, "sentinel"),
			PromoterEvidence: classGateString(evidence, "promoter"),
			CommandReadback:  classGateString(evidence, "command_readback"),
		}
		if nodeSummary.Status != "completed" {
			runLinksOK = false
			blockers = append(blockers, "run-link "+nodeID+" must be completed")
		}
		if nodeSummary.ChangedFile == "" {
			runLinksOK = false
			blockers = append(blockers, "run-link "+nodeID+" requires changed_file evidence")
		}
		if nodeSummary.PullRequest == "" || nodeSummary.MergeCommit == "" || !statusPassed(nodeSummary.CI) {
			prCIOK = false
			blockers = append(blockers, "run-link "+nodeID+" requires passed CI evidence")
		}
		if nodeSummary.NodeGatePath == "" {
			if override, ok := gateOverrides[nodeID]; ok {
				nodeSummary.NodeGatePath = override
			}
		}
		gate, gateSource, err := loadComplexPromotionNodeGate(nodeSummary.NodeGatePath)
		if err != nil {
			nodeGatesOK = false
			blockers = append(blockers, "node gate "+nodeID+" is missing")
		} else {
			rollup.SourceEvidence = append(rollup.SourceEvidence, gateSource)
			nodeSummary.NodeGateSHA256 = gateSource.SHA256
			nodeSummary.SafeToExecuteBeforeRun = gate.Status == "ready" && gate.NodeID == nodeID && gate.SafeToRequest && gate.SafeToExecute && gate.LiveExecutionAuthority && len(gate.Blockers) == 0
			if !nodeSummary.SafeToExecuteBeforeRun {
				nodeGatesOK = false
				blockers = append(blockers, "node gate "+nodeID+" must be ready and safe_to_execute=true")
			}
			if gate.SchedulesWork || gate.ExecutesWork || gate.ApprovesWork || gate.MutatesRepositories {
				boundedOK = false
				blockers = append(blockers, "node gate "+nodeID+" expands forbidden authority")
			}
			required := stringSet(gate.RequiredGates)
			for _, want := range []string{"rollback_record_complete", "sentinel_hold_default", "promoter_no_promotion", "command_readback_required", "forge_ao2_packet_required"} {
				if !required[want] {
					boundedOK = false
					blockers = append(blockers, "node gate "+nodeID+" missing required gate "+want)
				}
			}
		}
		completedAction := map[string]string{
			"task_id":      nodeSummary.TaskID,
			"changed_file": nodeSummary.ChangedFile,
			"pull_request": nodeSummary.PullRequest,
			"merge_commit": nodeSummary.MergeCommit,
			"ci":           nodeSummary.CI,
		}
		if strings.TrimSpace(paths.ClosureRoot) != "" {
			for _, spec := range complexClosureRoleSpecs() {
				current := ""
				switch spec.Field {
				case "rollback":
					current = nodeSummary.RollbackEvidence
				case "sentinel":
					current = nodeSummary.SentinelEvidence
				case "promoter":
					current = nodeSummary.PromoterEvidence
				case "command_readback":
					current = nodeSummary.CommandReadback
				}
				if current != "" {
					continue
				}
				closurePath, closureSource, blocker := loadComplexPromotionClosureEvidence(paths, spec, nodeID, runLinkSource.SHA256, nodeSummary.NodeGateSHA256, missionSource.SHA256, workgraphSource.SHA256, completedAction)
				if blocker != "" {
					blockers = append(blockers, blocker)
					switch spec.Field {
					case "rollback":
						rollbackOK = false
					case "sentinel":
						sentinelOK = false
					case "promoter":
						promoterOK = false
					case "command_readback":
						commandOK = false
					}
					continue
				}
				rollup.SourceEvidence = append(rollup.SourceEvidence, closureSource)
				switch spec.Field {
				case "rollback":
					nodeSummary.RollbackEvidence = closurePath
				case "sentinel":
					nodeSummary.SentinelEvidence = closurePath
				case "promoter":
					nodeSummary.PromoterEvidence = closurePath
				case "command_readback":
					nodeSummary.CommandReadback = closurePath
				}
			}
		}
		if nodeSummary.RollbackEvidence == "" {
			rollbackOK = false
			blockers = append(blockers, "run-link "+nodeID+" requires rollback evidence")
		}
		if nodeSummary.SentinelEvidence == "" {
			sentinelOK = false
			blockers = append(blockers, "run-link "+nodeID+" requires Sentinel evidence")
		}
		if nodeSummary.PromoterEvidence == "" {
			promoterOK = false
			blockers = append(blockers, "run-link "+nodeID+" requires Promoter evidence")
		}
		if nodeSummary.CommandReadback == "" {
			commandOK = false
			blockers = append(blockers, "run-link "+nodeID+" requires Command readback")
		}
		rollup.Nodes = append(rollup.Nodes, nodeSummary)
	}
	rollup.Checks["run_links_complete"] = runLinksOK && len(rollup.Nodes) == rollup.TotalNodes
	rollup.Checks["node_gates_safe"] = nodeGatesOK
	rollup.Checks["pr_ci_merge_evidence"] = prCIOK
	rollup.Checks["rollback_evidence"] = rollbackOK
	rollup.Checks["sentinel_evidence"] = sentinelOK
	rollup.Checks["promoter_evidence"] = promoterOK
	rollup.Checks["command_readback"] = commandOK
	rollup.Checks["bounded_authority"] = boundedOK
	rollup.Checks["forbidden_surfaces_clear"] = boundedOK
	rollup.PromoterVerdictReady = promoterOK
	rollup.CommandReadbackReady = commandOK
	rollup.Blockers = uniqueStrings(blockers)
	if len(rollup.Blockers) > 0 {
		rollup.FirstFailingCheck = rollup.Blockers[0]
		return rollup, nil
	}
	rollup.Status = "ready"
	rollup.SafeToPromote = true
	rollup.ComplexRepoMutationLiveProven = true
	rollup.HighestProvenLiveClass = "complex_repo_mutation"
	rollup.NextDeniedClass = "fully_unsupervised_complex_mutation"
	return rollup, nil
}

func loadComplexPromotionClosureEvidence(paths complexPromotionRollupPaths, spec complexClosureRoleSpec, nodeID, runLinkSHA, nodeGateSHA, missionSHA, workgraphSHA string, completedAction map[string]string) (string, MutationClassGateEvidence, string) {
	rel := filepath.Join(nodeID, spec.Filename)
	relDisplay := filepath.ToSlash(rel)
	path := filepath.Join(paths.ClosureRoot, rel)
	source, object, err := readComplexNodeGateObject("closure:"+nodeID+":"+spec.Field, path)
	if err != nil {
		return "", MutationClassGateEvidence{}, "closure evidence " + relDisplay + " is missing"
	}
	if classGateString(object, "schema_version") != spec.SchemaVersion {
		return "", source, "closure evidence " + relDisplay + " schema mismatch"
	}
	if classGateString(object, "status") != spec.Status {
		return "", source, "closure evidence " + relDisplay + " status mismatch"
	}
	if classGateString(object, "mutation_class") != "complex_repo_mutation" {
		return "", source, "closure evidence " + relDisplay + " mutation_class mismatch"
	}
	if classGateString(object, "node_id") != nodeID {
		return "", source, "closure evidence " + relDisplay + " node_id mismatch"
	}
	if classGateString(object, "run_link_sha256") != runLinkSHA {
		return "", source, "closure evidence " + relDisplay + " run-link digest mismatch"
	}
	if classGateString(object, "node_gate_sha256") != nodeGateSHA {
		return "", source, "closure evidence " + relDisplay + " node-gate digest mismatch"
	}
	if classGateString(object, "mission_sha256") != missionSHA {
		return "", source, "closure evidence " + relDisplay + " mission digest mismatch"
	}
	if classGateString(object, "workgraph_sha256") != workgraphSHA {
		return "", source, "closure evidence " + relDisplay + " workgraph digest mismatch"
	}
	if classGateString(object, "forbidden_surface_result") != "clear" {
		return "", source, "closure evidence " + relDisplay + " forbidden surface result must be clear"
	}
	if classGateString(object, "rollback_disposition") != "ready" {
		return "", source, "closure evidence " + relDisplay + " rollback disposition must be ready"
	}
	if classGateBool(object, "schedules_work") || classGateBool(object, "executes_work") || classGateBool(object, "approves_work") || classGateBool(object, "mutates_repositories") {
		return "", source, "closure evidence " + relDisplay + " expands forbidden authority"
	}
	if classGateString(object, "fully_unsupervised_complex_mutation") != "denied" || classGateString(object, "rsi") != "denied" {
		return "", source, "closure evidence " + relDisplay + " must keep higher classes denied"
	}
	action, _ := object["completed_action"].(map[string]any)
	for key, want := range completedAction {
		if classGateString(action, key) != want {
			return "", source, "closure evidence " + relDisplay + " completed action mismatch"
		}
	}
	return path, source, ""
}

func loadComplexPromotionNodeGateOverrides(nodeGatePaths []string, finalNodeGatePath string) (map[string]string, []MutationClassGateEvidence, error) {
	overrides := map[string]string{}
	evidence := []MutationClassGateEvidence{}
	allPaths := append([]string{}, nodeGatePaths...)
	allPaths = append(allPaths, finalNodeGatePath)
	for _, path := range uniqueStrings(allPaths) {
		gate, source, err := loadComplexPromotionNodeGate(path)
		if err != nil {
			return nil, nil, err
		}
		overrides[gate.NodeID] = path
		evidence = append(evidence, source)
	}
	return overrides, evidence, nil
}

func loadComplexPromotionNodeGate(path string) (ComplexRepoMutationNodeGate, MutationClassGateEvidence, error) {
	if strings.TrimSpace(path) == "" {
		return ComplexRepoMutationNodeGate{}, MutationClassGateEvidence{}, errors.New("empty node gate path")
	}
	source, object, err := readComplexNodeGateObject("complex_node_gate", path)
	if err != nil {
		return ComplexRepoMutationNodeGate{}, MutationClassGateEvidence{}, err
	}
	data, err := json.Marshal(object)
	if err != nil {
		return ComplexRepoMutationNodeGate{}, MutationClassGateEvidence{}, err
	}
	var gate ComplexRepoMutationNodeGate
	if err := json.Unmarshal(data, &gate); err != nil {
		return ComplexRepoMutationNodeGate{}, MutationClassGateEvidence{}, err
	}
	if gate.SchemaVersion != complexNodeGateSchema {
		return ComplexRepoMutationNodeGate{}, MutationClassGateEvidence{}, fmt.Errorf("node gate schema_version must be %s", complexNodeGateSchema)
	}
	return gate, source, nil
}
