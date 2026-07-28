package cli

import (
	"errors"
	"fmt"
	"path/filepath"
)

type complexClosureRoleSpec struct {
	Field         string
	Filename      string
	SchemaVersion string
	Status        string
	Extra         map[string]any
}

func complexClosureRoleSpecs() []complexClosureRoleSpec {
	return []complexClosureRoleSpec{
		{
			Field:         "rollback",
			Filename:      "rollback.json",
			SchemaVersion: complexRollbackClosureSchema,
			Status:        "ready",
			Extra: map[string]any{
				"rollback_disposition": "ready",
			},
		},
		{
			Field:         "sentinel",
			Filename:      "sentinel.json",
			SchemaVersion: complexSentinelClosureSchema,
			Status:        "clear",
			Extra: map[string]any{
				"hold_required":          false,
				"promoter_hold_required": false,
			},
		},
		{
			Field:         "promoter",
			Filename:      "promoter.json",
			SchemaVersion: complexPromoterClosureSchema,
			Status:        "no_promotion",
			Extra: map[string]any{
				"promotion_allowed": false,
				"promotion_scope":   "per_node_closure_only",
			},
		},
		{
			Field:         "command_readback",
			Filename:      "command-readback.json",
			SchemaVersion: complexCommandClosureSchema,
			Status:        "ready",
			Extra: map[string]any{
				"read_only":       true,
				"safe_to_execute": false,
			},
		},
	}
}

func buildComplexRepoClosureEvidence(paths complexPromotionRollupPaths) (map[string]any, error) {
	missionSource, mission, err := readComplexNodeGateObject("mission_continuation_evidence", paths.Mission)
	if err != nil {
		return nil, err
	}
	workgraphSource, workgraph, err := readComplexNodeGateObject("atlas_final_workgraph", paths.Workgraph)
	if err != nil {
		return nil, err
	}
	nodes := classGateObjectSlice(workgraph["nodes"])
	if len(nodes) == 0 {
		return nil, errors.New("workgraph must contain completed nodes")
	}
	if classGateString(mission, "status") != "all_nodes_completed_with_foundry_evidence" {
		return nil, errors.New("mission continuation evidence must report all nodes completed")
	}
	gateOverrides, _, err := loadComplexPromotionNodeGateOverrides(paths.NodeGates, paths.FinalNodeGate)
	if err != nil {
		return nil, err
	}
	generated := []map[string]any{}
	for _, node := range nodes {
		nodeID := classGateString(node, "id")
		if nodeID == "" {
			return nil, errors.New("workgraph node missing id")
		}
		if classGateString(node, "status") != "completed" {
			return nil, fmt.Errorf("workgraph node %s must be completed", nodeID)
		}
		runLinkPath := filepath.Join(paths.RunLinksRoot, nodeID, "run-link.json")
		runLinkSource, runLink, err := readComplexNodeGateObject("run_link:"+nodeID, runLinkPath)
		if err != nil {
			return nil, fmt.Errorf("run-link %s is missing: %w", nodeID, err)
		}
		evidence, _ := runLink["evidence"].(map[string]any)
		nodeGatePath := classGateString(evidence, "node_gate")
		if nodeGatePath == "" {
			nodeGatePath = gateOverrides[nodeID]
		}
		gate, gateSource, err := loadComplexPromotionNodeGate(nodeGatePath)
		if err != nil {
			return nil, fmt.Errorf("node gate %s is missing: %w", nodeID, err)
		}
		if gate.Status != "ready" || gate.NodeID != nodeID || !gate.SafeToExecute || !gate.SafeToRequest || len(gate.Blockers) != 0 {
			return nil, fmt.Errorf("node gate %s must be ready and safe_to_execute=true", nodeID)
		}
		if gate.SchedulesWork || gate.ExecutesWork || gate.ApprovesWork || gate.MutatesRepositories {
			return nil, fmt.Errorf("node gate %s expands forbidden authority", nodeID)
		}
		completedAction := map[string]any{
			"task_id":        classGateString(runLink, "task_id"),
			"changed_file":   classGateString(evidence, "changed_file"),
			"pull_request":   classGateString(evidence, "pr"),
			"merge_commit":   classGateString(evidence, "merge_commit"),
			"ci":             classGateString(evidence, "ci"),
			"run_status":     classGateString(runLink, "status"),
			"node_gate":      nodeGatePath,
			"mutation_class": "complex_repo_mutation",
		}
		if completedAction["changed_file"] == "" || completedAction["pull_request"] == "" || completedAction["merge_commit"] == "" || !statusPassed(classGateString(evidence, "ci")) {
			return nil, fmt.Errorf("run-link %s requires changed_file, PR, merge commit, and passed CI evidence", nodeID)
		}
		for _, spec := range complexClosureRoleSpecs() {
			doc := map[string]any{
				"schema_version":                      spec.SchemaVersion,
				"status":                              spec.Status,
				"mutation_class":                      "complex_repo_mutation",
				"evidence_role":                       spec.Field,
				"node_id":                             nodeID,
				"task_id":                             classGateString(runLink, "task_id"),
				"run_link_path":                       filepath.ToSlash(runLinkPath),
				"run_link_sha256":                     runLinkSource.SHA256,
				"node_gate_path":                      filepath.ToSlash(nodeGatePath),
				"node_gate_sha256":                    gateSource.SHA256,
				"workgraph_path":                      filepath.ToSlash(paths.Workgraph),
				"workgraph_sha256":                    workgraphSource.SHA256,
				"mission_path":                        filepath.ToSlash(paths.Mission),
				"mission_sha256":                      missionSource.SHA256,
				"completed_action":                    completedAction,
				"forbidden_surface_result":            "clear",
				"rollback_disposition":                "ready",
				"safe_to_execute_before_run":          true,
				"schedules_work":                      false,
				"executes_work":                       false,
				"approves_work":                       false,
				"mutates_repositories":                false,
				"fully_unsupervised_complex_mutation": "denied",
				"rsi":                                 "denied",
				"generated_at_utc":                    nowUTC(),
			}
			for key, value := range spec.Extra {
				doc[key] = value
			}
			path := filepath.Join(paths.ClosureRoot, nodeID, spec.Filename)
			if err := writeJSONFile(path, doc); err != nil {
				return nil, err
			}
			sha, err := fileSHA256(path)
			if err != nil {
				return nil, err
			}
			generated = append(generated, map[string]any{
				"node_id":        nodeID,
				"evidence_role":  spec.Field,
				"path":           filepath.ToSlash(path),
				"schema_version": spec.SchemaVersion,
				"status":         spec.Status,
				"sha256":         sha,
			})
		}
	}
	return map[string]any{
		"schema_version":                      complexClosureManifestSchema,
		"status":                              "ready",
		"mutation_class":                      "complex_repo_mutation",
		"mission":                             classGateString(mission, "mission"),
		"node_count":                          len(nodes),
		"evidence_item_count":                 len(generated),
		"evidence":                            generated,
		"fully_unsupervised_complex_mutation": "denied",
		"rsi":                                 "denied",
		"generated_at_utc":                    nowUTC(),
	}, nil
}
