package cli

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type complexNodeGatePaths struct {
	Workgraph     string
	FoundryImport string
	Candidate     string
	Rollback      string
}

func buildComplexRepoMutationNodeGate(paths complexNodeGatePaths) (ComplexRepoMutationNodeGate, error) {
	gate := ComplexRepoMutationNodeGate{
		SchemaVersion:                    complexNodeGateSchema,
		Status:                           "blocked",
		MutationClass:                    "complex_repo_mutation",
		HighestProvenLiveClass:           "multi_repo_low_risk",
		NextDeniedClass:                  "complex_repo_mutation",
		ExactNextAction:                  "complete_complex_node_gate_prerequisites_before_execution",
		AuthorityBoundary:                "complex_node_exact_scope_only",
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
	workgraphSource, workgraph, err := readComplexNodeGateObject("atlas_workgraph", paths.Workgraph)
	if err != nil {
		return gate, err
	}
	importSource, foundryImport, err := readComplexNodeGateObject("foundry_import", paths.FoundryImport)
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
	gate.SourceEvidence = append(gate.SourceEvidence, workgraphSource, importSource, candidateSource, rollbackSource)
	gate.WorkgraphID = classGateString(workgraph, "id")
	gate.FoundryImportID = classGateString(foundryImport, "id")
	gate.FoundryImportStatus = classGateString(foundryImport, "status")
	gate.FoundryImportSchedulesWork = classGateBool(foundryImport, "schedules_work")
	gate.FoundryImportExecutesWork = classGateBool(foundryImport, "executes_work")
	gate.FoundryImportApprovesWork = classGateBool(foundryImport, "approves_work")
	gate.CandidateStatus = classGateString(candidate, "status")
	gate.CandidateExecutableReady = classGateBool(candidate, "executable_ready")
	gate.CandidateSafeToExecute = classGateBool(candidate, "safe_to_execute")
	gate.RollbackStatus = classGateString(rollback, "status")
	gate.RollbackSafeToExecute = classGateBool(rollback, "safe_to_execute")
	gate.RequiredGates = classGateStringSlice(candidate, "required_gates")

	tasks := classGateObjectSlice(foundryImport["tasks"])
	gate.FoundryImportTaskCount = len(tasks)
	var task map[string]any
	if len(tasks) == 1 {
		task = tasks[0]
	}
	taskNodeID := classGateString(task, "node_id")
	taskID := classGateFirstNonEmpty(classGateString(task, "task_id"), classGateNestedString(task, "factory_task", "id"))
	gate.TargetFactoryRepo = classGateFirstNonEmpty(classGateString(task, "target_factory_repo"), classGateNestedString(task, "factory_task", "target_factory_repo"))
	candidateNodeID := classGateString(candidate, "node_id")
	rollbackNodeID := classGateString(rollback, "node_id")
	gate.NodeID = classGateFirstNonEmpty(candidateNodeID, taskNodeID, rollbackNodeID)
	gate.TaskID = classGateFirstNonEmpty(classGateString(candidate, "task_id"), taskID, classGateString(rollback, "task_id"))
	node, nodeFound := findWorkgraphNode(workgraph, gate.NodeID)
	nodeStatus := classGateString(node, "status")
	taskMutationClass := classGateFirstNonEmpty(classGateString(task, "mutation_class"), classGateNestedString(task, "factory_task", "mutation_class"))
	importRequiredEvidence := classGateStringSlice(task, "required_evidence")
	if len(importRequiredEvidence) == 0 {
		importRequiredEvidence = classGateNestedStringSlice(task, "factory_task", "required_evidence")
	}
	gate.SafeToRequest = gate.FoundryImportStatus != "" &&
		gate.CandidateStatus == "ready" &&
		nodeStatus == "ready" &&
		len(tasks) == 1 &&
		!gate.FoundryImportSchedulesWork &&
		!gate.FoundryImportExecutesWork &&
		!gate.FoundryImportApprovesWork

	var blockers []string
	switch gate.FoundryImportStatus {
	case "ready", "ready_for_foundry_fixture_import":
	default:
		blockers = append(blockers, "foundry import status must be ready")
	}
	if gate.FoundryImportSchedulesWork || gate.FoundryImportExecutesWork || gate.FoundryImportApprovesWork {
		blockers = append(blockers, "foundry import must not schedule, execute, or approve work")
	}
	if len(tasks) != 1 {
		blockers = append(blockers, "foundry import must contain exactly one selected node")
	}
	if gate.NodeID == "" || gate.TaskID == "" {
		blockers = append(blockers, "complex node gate requires node_id and task_id")
	}
	if taskNodeID != "" && candidateNodeID != "" && taskNodeID != candidateNodeID {
		blockers = append(blockers, "foundry import node_id must match candidate record")
	}
	if rollbackNodeID != "" && gate.NodeID != "" && rollbackNodeID != gate.NodeID {
		blockers = append(blockers, "rollback record node_id must match selected node")
	}
	if !nodeFound {
		blockers = append(blockers, "workgraph must contain the selected node")
	} else if nodeStatus != "ready" {
		blockers = append(blockers, "workgraph selected node status must be ready")
	}
	if taskMutationClass != "complex_repo_mutation" {
		blockers = append(blockers, "selected node evidence must be class complex_repo_mutation")
	}
	if gate.CandidateStatus != "ready" {
		blockers = append(blockers, "complex candidate record status must be ready")
	}
	if !gate.CandidateExecutableReady {
		blockers = append(blockers, "complex candidate record executable_ready must be true")
	}
	if !gate.CandidateSafeToExecute {
		blockers = append(blockers, "complex candidate record safe_to_execute is false")
	}
	if classGateStringSliceContains(gate.RequiredGates, "safe_to_execute:false") {
		blockers = append(blockers, "complex candidate record requires safe_to_execute:false")
	}
	if gate.RollbackStatus != "" && gate.RollbackStatus != "ready" {
		blockers = append(blockers, "complex rollback record status must be ready")
	}
	if !gate.RollbackSafeToExecute {
		blockers = append(blockers, "complex rollback record safe_to_execute is false")
	}
	if classGateStringSliceContains(importRequiredEvidence, "safe_to_execute:false") {
		blockers = append(blockers, "complex Foundry import requires safe_to_execute:false")
	} else if !classGateStringSliceContains(importRequiredEvidence, "safe_to_execute:true") {
		blockers = append(blockers, "complex Foundry import must bind safe_to_execute:true before execution")
	}
	if blockers == nil {
		blockers = []string{}
	}
	gate.Blockers = blockers
	if len(blockers) > 0 {
		gate.FirstFailingCheck = blockers[0]
		return gate, nil
	}
	gate.Status = "ready"
	gate.SafeToRequest = true
	gate.SafeToExecute = true
	gate.LiveExecutionAuthority = true
	gate.ExactNextAction = "execute_exact_complex_node_candidate"
	return gate, nil
}

func readComplexNodeGateObject(name, path string) (MutationClassGateEvidence, map[string]any, error) {
	document, err := readArbitraryJSON(path)
	if err != nil {
		return MutationClassGateEvidence{}, nil, fmt.Errorf("read %s: %w", name, err)
	}
	object, ok := document.(map[string]any)
	if !ok {
		return MutationClassGateEvidence{}, nil, fmt.Errorf("%s must be a JSON object", name)
	}
	sum, err := fileSHA256(path)
	if err != nil {
		return MutationClassGateEvidence{}, nil, fmt.Errorf("hash %s: %w", name, err)
	}
	status := classGateFirstNonEmpty(classGateString(object, "status"), "validated")
	source := MutationClassGateEvidence{
		Name:          name,
		Path:          path,
		SchemaVersion: classGateFirstNonEmpty(classGateString(object, "schema_version"), classGateString(object, "contract_version"), classGateString(object, "schema")),
		Status:        status,
		SHA256:        sum,
	}
	return source, object, nil
}

func loadComplexRepoMutationNodeGate(path string) (ComplexRepoMutationNodeGate, error) {
	document, err := readArbitraryJSON(path)
	if err != nil {
		return ComplexRepoMutationNodeGate{}, err
	}
	data, err := json.Marshal(document)
	if err != nil {
		return ComplexRepoMutationNodeGate{}, err
	}
	var gate ComplexRepoMutationNodeGate
	if err := json.Unmarshal(data, &gate); err != nil {
		return ComplexRepoMutationNodeGate{}, err
	}
	if gate.SchemaVersion != complexNodeGateSchema {
		return ComplexRepoMutationNodeGate{}, fmt.Errorf("node gate schema_version must be %s", complexNodeGateSchema)
	}
	return gate, nil
}

func buildComplexNodeRecord(gate ComplexRepoMutationNodeGate, nodeClass, scope, summary string) map[string]any {
	cleanScope := filepath.ToSlash(strings.TrimSpace(scope))
	highestProven := classGateFirstNonEmpty(gate.HighestProvenLiveClass, "unknown")
	fullyUnsupervised := classGateFirstNonEmpty(gate.FullyUnsupervisedComplexMutation, "denied")
	rsi := classGateFirstNonEmpty(gate.RSI, "denied")
	return map[string]any{
		"schema":              "ao.atlas.complex-repo-mutation-node-record.v0.1",
		"node_id":             gate.NodeID,
		"task_id":             gate.TaskID,
		"node_class":          strings.TrimSpace(nodeClass),
		"target_factory_repo": strings.TrimSpace(gate.TargetFactoryRepo),
		"status":              "completed",
		"mutation_class":      gate.MutationClass,
		"scope":               cleanScope,
		"summary":             strings.TrimSpace(summary),
		"accepted_evidence": []string{
			"mutation_class:" + gate.MutationClass,
			"highest_proven_live_class:" + highestProven,
			"safe_to_execute:true",
			"node_id:" + gate.NodeID,
		},
		"authority_boundaries": map[string]bool{
			"schedules_work":                      false,
			"executes_providers":                  false,
			"approves_work":                       false,
			"release_or_publish_allowed":          false,
			"credential_or_secret_access_allowed": false,
			"direct_main_mutation_allowed":        false,
			"public_claim_broadening_allowed":     false,
		},
		"class_state": map[string]string{
			"complex_repo_mutation_live_proven":   fmt.Sprint(highestProven == "complex_repo_mutation"),
			"fully_unsupervised_complex_mutation": fullyUnsupervised,
			"rsi":                                 rsi,
		},
		"rollback": map[string]string{
			"scope":  cleanScope,
			"method": "governed revert pull request if rollback is required after merge",
		},
	}
}

func buildComplexNodeRunLink(gate ComplexRepoMutationNodeGate, scope, nodeGatePath, pr, mergeCommit, ci string) AtlasRunLink {
	evidence := map[string]string{
		"changed_file": filepath.ToSlash(filepath.Join(strings.TrimSpace(scope), "node-record.json")),
		"ci":           strings.TrimSpace(ci),
		"node_gate":    filepath.ToSlash(nodeGatePath),
	}
	if strings.TrimSpace(pr) != "" {
		evidence["pr"] = strings.TrimSpace(pr)
	}
	if strings.TrimSpace(mergeCommit) != "" {
		evidence["merge_commit"] = strings.TrimSpace(mergeCommit)
	}
	link := AtlasRunLink{
		ContractVersion: atlasRunLinkSchema,
		TaskID:          gate.TaskID,
		Status:          "completed",
		Evidence:        evidence,
	}
	link.Digest = digestAtlasRunLink(link)
	return link
}

func digestAtlasRunLink(link AtlasRunLink) string {
	link.Digest = ""
	data, err := json.Marshal(link)
	if err != nil {
		return "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	}
	sum := sha256.Sum256(data)
	return "sha256:" + fmt.Sprintf("%x", sum[:])
}

func evaluateComplexNodeGateEvidence(path string) (MutationClassGateEvidence, *ComplexRepoMutationNodeGate, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return MutationClassGateEvidence{}, nil, "", fmt.Errorf("read complex_node_gate: %w", err)
	}
	var nodeGate ComplexRepoMutationNodeGate
	if err := json.Unmarshal(data, &nodeGate); err != nil {
		return MutationClassGateEvidence{}, nil, "", fmt.Errorf("parse complex_node_gate: %w", err)
	}
	sum, err := fileSHA256(path)
	if err != nil {
		return MutationClassGateEvidence{}, nil, "", fmt.Errorf("hash complex_node_gate: %w", err)
	}
	evidence := MutationClassGateEvidence{
		Name:          "complex_node_gate",
		Path:          path,
		SchemaVersion: nodeGate.SchemaVersion,
		Status:        nodeGate.Status,
		SHA256:        sum,
	}
	switch {
	case nodeGate.SchemaVersion != complexNodeGateSchema:
		return evidence, &nodeGate, "complex_node_gate schema_version must be " + complexNodeGateSchema, nil
	case nodeGate.MutationClass != "complex_repo_mutation":
		return evidence, &nodeGate, "complex_node_gate mutation_class must be complex_repo_mutation", nil
	case nodeGate.Status != "ready":
		if nodeGate.FirstFailingCheck != "" {
			return evidence, &nodeGate, nodeGate.FirstFailingCheck, nil
		}
		return evidence, &nodeGate, "complex_node_gate status must be ready", nil
	case !nodeGate.SafeToRequest || !nodeGate.SafeToExecute || !nodeGate.LiveExecutionAuthority:
		return evidence, &nodeGate, "complex_node_gate must grant exact safe_to_execute authority", nil
	case nodeGate.NodeID == "" || nodeGate.TaskID == "":
		return evidence, &nodeGate, "complex_node_gate requires node_id and task_id", nil
	case nodeGate.SchedulesWork || nodeGate.ExecutesWork || nodeGate.ApprovesWork || nodeGate.MutatesRepositories:
		return evidence, &nodeGate, "complex_node_gate evidence must not schedule, execute, approve, or mutate repositories", nil
	default:
		return evidence, &nodeGate, "", nil
	}
}

func findWorkgraphNode(workgraph map[string]any, nodeID string) (map[string]any, bool) {
	for _, node := range classGateObjectSlice(workgraph["nodes"]) {
		if classGateString(node, "id") == nodeID {
			return node, true
		}
	}
	return nil, false
}
