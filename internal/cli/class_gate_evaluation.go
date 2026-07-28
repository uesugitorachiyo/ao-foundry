package cli

import (
	"fmt"
	"strings"
)

type classGateEvidencePaths struct {
	Atlas                  string
	Covenant               string
	Sentinel               string
	Promoter               string
	Rollback               string
	Command                string
	CI                     string
	TestOnlySuccess        string
	MultiRepoPlan          string
	LowRiskCodeLiveSuccess string
	ComplexNodeGate        string
}

func evaluateMutationClassGate(paths classGateEvidencePaths) (MutationClassGate, error) {
	requiredEvidence := []string{"atlas_classification", "covenant_class_ticket", "sentinel_no_hold", "promoter_ready", "rollback_proof", "command_readback", "ci_passed"}
	gate := MutationClassGate{
		SchemaVersion:       classGateSchema,
		Status:              "blocked",
		RequiredEvidence:    requiredEvidence,
		DeniedClasses:       deniedMutationClasses(""),
		AuthorityBoundary:   "single_class_only",
		NextActions:         []string{},
		SchedulesWork:       false,
		ExecutesWork:        false,
		ApprovesWork:        false,
		MutatesRepositories: false,
	}
	checks := []classGateCheck{
		{Name: "atlas_classification", Path: paths.Atlas, SchemaVersion: "ao.atlas.mutation-classification.v0.1", StatusField: "status", ReadyStatuses: []string{"ready"}},
		{Name: "covenant_class_ticket", Path: paths.Covenant, SchemaVersion: "covenant.mutation-class-authority-ticket.v1", StatusField: "approval_state", ReadyStatuses: []string{"approved"}},
		{Name: "sentinel_no_hold", Path: paths.Sentinel, SchemaVersion: "ao.sentinel.mutation-class-hold.v0.1", StatusField: "status", ReadyStatuses: []string{"no_hold"}},
		{Name: "promoter_ready", Path: paths.Promoter, SchemaVersion: "ao.promoter.mutation-class-promotion.v0.1", StatusField: "status", ReadyStatuses: []string{"ready"}},
		{Name: "rollback_proof", Path: paths.Rollback, SchemaVersion: "ao.foundry.mutation-class-rollback.v0.1", StatusField: "status", ReadyStatuses: []string{"passed"}},
		{Name: "command_readback", Path: paths.Command, SchemaVersion: "ao.command.atlas-authority-ladder.v0.1", StatusField: "readback_status", ReadyStatuses: []string{"ready"}},
		{Name: "ci_passed", Path: paths.CI, SchemaVersion: "ao.foundry.ci-readiness.v0.1", StatusField: "status", ReadyStatuses: []string{"passed"}},
	}
	var className string
	var blockers []string
	documents := map[string]map[string]any{}
	for _, check := range checks {
		evidence, document, blocker, err := evaluateClassGateCheck(check, className)
		if err != nil {
			return gate, err
		}
		gate.SourceEvidence = append(gate.SourceEvidence, evidence)
		documents[check.Name] = document
		documentClass := classGateString(document, "mutation_class")
		if documentClass == "" && check.Name == "command_readback" {
			documentClass = classGateString(document, "next_class")
		}
		if className == "" && documentClass != "" {
			className = documentClass
			gate.MutationClass = className
			gate.DeniedClasses = deniedMutationClasses(className)
		}
		if blocker != "" {
			blockers = append(blockers, blocker)
		}
	}
	if className == "" {
		blockers = append([]string{"atlas_classification missing mutation_class"}, blockers...)
	}
	if className == "low_risk_code" {
		requiredEvidence = append(requiredEvidence, "test_only_success")
		gate.RequiredEvidence = requiredEvidence
		testOnlySuccessReady := false
		if strings.TrimSpace(paths.TestOnlySuccess) == "" {
			blockers = append(blockers, "test_only_success evidence is required for low_risk_code")
		} else {
			evidence, blocker, err := evaluateTestOnlySuccessEvidence(paths.TestOnlySuccess)
			if err != nil {
				return gate, err
			}
			gate.SourceEvidence = append(gate.SourceEvidence, evidence)
			if blocker != "" {
				blockers = append(blockers, blocker)
			} else {
				testOnlySuccessReady = true
			}
		}
		boundaryChecks, boundaryBlockers := evaluateLowRiskCodeBoundaryChecks(documents, testOnlySuccessReady)
		gate.ClassBoundaryChecks = boundaryChecks
		blockers = append(blockers, boundaryBlockers...)
		gate.DenialAudit = lowRiskCodeDenialAudit(len(blockers) == 0)
		if len(blockers) == 0 {
			gate.Status = "ready"
			gate.SafeToRequest = true
			gate.SafeToExecute = false
			gate.ClassBoundaryChecks.SafeToRequest = true
			gate.NextActions = []string{"Request a low_risk_code dry-run design only; live code execution remains denied until a later promotion slice."}
			return gate, nil
		}
	}
	if className == "multi_repo_low_risk" {
		requiredEvidence = append(requiredEvidence, "low_risk_code_live_success", "multi_repo_sequencing_plan", "per_repo_rollback", "ci_per_repo", "operator_kill_switch", "fresh_repo_state")
		gate.RequiredEvidence = requiredEvidence
		lowRiskLiveReady := false
		if strings.TrimSpace(paths.LowRiskCodeLiveSuccess) == "" {
			blockers = append(blockers, "low_risk_code_live_success evidence is required for multi_repo_low_risk")
		} else {
			evidence, success, blocker, err := evaluateLowRiskCodeLiveSuccessEvidence(paths.LowRiskCodeLiveSuccess)
			if err != nil {
				return gate, err
			}
			gate.SourceEvidence = append(gate.SourceEvidence, evidence)
			gate.LowRiskLiveSuccess = success
			if blocker != "" {
				blockers = append(blockers, blocker)
			} else {
				lowRiskLiveReady = true
			}
		}
		if strings.TrimSpace(paths.MultiRepoPlan) == "" {
			blockers = append(blockers, "multi_repo_sequencing_plan evidence is required for multi_repo_low_risk")
		} else {
			evidence, repoPlan, repoSafety, blocker, err := evaluateMultiRepoPlanEvidence(paths.MultiRepoPlan)
			if err != nil {
				return gate, err
			}
			gate.SourceEvidence = append(gate.SourceEvidence, evidence)
			gate.RepoExecutionPlan = repoPlan
			gate.RepoSafety = repoSafety
			if blocker != "" {
				blockers = append(blockers, blocker)
			} else {
				blockers = append(blockers, evaluateMultiRepoAuthorityEvidence(repoPlan, documents["rollback_proof"], documents["ci_passed"])...)
			}
		}
		gate.LiveRehearsalDecision = multiRepoLiveRehearsalDecision(documents["command_readback"], len(blockers) == 0, lowRiskLiveReady)
		if len(blockers) == 0 {
			gate.Status = "ready"
			gate.SafeToRequest = true
			gate.SafeToExecute = lowRiskLiveReady
			if lowRiskLiveReady {
				gate.NextActions = []string{"Repo-one multi_repo_low_risk live rehearsal is ready for the exact approved candidate; do not execute without the operator prompt that explicitly authorizes that live step."}
			} else {
				gate.NextActions = []string{"Request multi_repo_low_risk dry-run sequencing only; live multi-repo execution remains denied until per-repo live evidence, rollback, CI, Sentinel, Promoter, and Command readback pass."}
			}
			return gate, nil
		}
	}
	if className == "complex_repo_mutation" {
		requiredEvidence = append(requiredEvidence, "complex_node_gate")
		gate.RequiredEvidence = requiredEvidence
		if strings.TrimSpace(paths.ComplexNodeGate) == "" {
			blockers = append(blockers, "complex_repo_mutation requires complex_node_gate evidence")
		} else {
			evidence, nodeGate, blocker, err := evaluateComplexNodeGateEvidence(paths.ComplexNodeGate)
			if err != nil {
				return gate, err
			}
			gate.SourceEvidence = append(gate.SourceEvidence, evidence)
			gate.ComplexNodeGate = nodeGate
			if blocker != "" {
				blockers = append(blockers, blocker)
			}
		}
		if len(blockers) == 0 {
			gate.Status = "ready"
			gate.SafeToRequest = true
			gate.SafeToExecute = true
			gate.NextActions = []string{"Execute only the exact complex_repo_mutation node named by complex_node_gate; keep one executable node active and require PR, CI, merge, rollback, Sentinel, Promoter, and Command readback before selecting the next node."}
			return gate, nil
		}
	}
	if len(blockers) == 0 {
		gate.Status = "ready"
		gate.SafeToRequest = true
		gate.SafeToExecute = true
		gate.NextActions = []string{"Request exactly one " + className + " mutation through the next governed gate; do not broaden class scope."}
		return gate, nil
	}
	gate.FirstFailingCheck = blockers[0]
	gate.NextActions = blockers
	return gate, nil
}

func multiRepoLiveRehearsalDecision(command map[string]any, safeToRequest bool, lowRiskLiveReady bool) *MultiRepoLiveRehearsalDecision {
	currentClass := classGateFirstNonEmpty(classGateString(command, "current_class"), "low_risk_code")
	nextClass := classGateFirstNonEmpty(classGateString(command, "next_class"), "multi_repo_low_risk")
	provenClass := classGateFirstNonEmpty(classGateString(command, "highest_proven_live_class"), "test_only")
	lowerEvidenceStatus := classGateFirstNonEmpty(classGateString(command, "low_risk_code_live_evidence_status"), "missing")
	denialReason := classGateFirstNonEmpty(classGateString(command, "next_denied_reason"), "denied until low_risk_code live rehearsal evidence is recorded")
	missingEvidence := []string{
		"low_risk_code_live_success",
		"rollback_proof:low_risk_code_live",
		"sentinel_no_hold:low_risk_code_live",
		"promoter_promotion:low_risk_code_live",
		"command_readback:low_risk_code_live",
		"clean_main_ci:low_risk_code_live",
	}
	status := "denied"
	exactNextAction := "complete_low_risk_code_live_rehearsal_before_multi_repo_live"
	if lowRiskLiveReady {
		status = "accepted"
		provenClass = "low_risk_code"
		lowerEvidenceStatus = "accepted"
		denialReason = "low_risk_code live rehearsal evidence accepted"
		missingEvidence = []string{}
		exactNextAction = "request_repo_one_multi_repo_low_risk_live_rehearsal"
	}
	return &MultiRepoLiveRehearsalDecision{
		SchemaVersion:                "ao.foundry.multi-repo-live-rehearsal-decision.v0.1",
		Status:                       status,
		MutationClass:                "multi_repo_low_risk",
		CurrentClass:                 currentClass,
		NextClass:                    nextClass,
		CurrentProvenLiveClass:       provenClass,
		LowerClassLiveEvidenceStatus: lowerEvidenceStatus,
		SafeToRequest:                safeToRequest,
		SafeToExecute:                safeToRequest && lowRiskLiveReady,
		LiveExecutionAuthority:       safeToRequest && lowRiskLiveReady,
		MissingEvidence:              missingEvidence,
		DenialReason:                 denialReason,
		ExactNextAction:              exactNextAction,
		RepoExecutionPolicy:          "sequenced_dry_run_only",
		SchedulesWork:                false,
		ExecutesWork:                 false,
		MutatesRepositories:          false,
	}
}

func classGateFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func classGateFirstNonEmptyStringSlice(values ...[]string) []string {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}
	return nil
}

func classGateStringSliceEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func classGateNestedStringSlice(document map[string]any, outer, inner string) []string {
	nested := classGateObject(document[outer])
	return classGateStringSlice(nested, inner)
}

func classGateObject(value any) map[string]any {
	object, _ := value.(map[string]any)
	return object
}

func classGateObjectSlice(value any) []map[string]any {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	objects := make([]map[string]any, 0, len(values))
	for _, value := range values {
		if object, ok := value.(map[string]any); ok {
			objects = append(objects, object)
		}
	}
	return objects
}

type classGateCheck struct {
	Name          string
	Path          string
	SchemaVersion string
	StatusField   string
	ReadyStatuses []string
}

func evaluateClassGateCheck(check classGateCheck, expectedClass string) (MutationClassGateEvidence, map[string]any, string, error) {
	document, err := readArbitraryJSON(check.Path)
	if err != nil {
		return MutationClassGateEvidence{}, nil, "", fmt.Errorf("read %s evidence: %w", check.Name, err)
	}
	object, ok := document.(map[string]any)
	if !ok {
		return MutationClassGateEvidence{}, nil, "", fmt.Errorf("%s evidence must be a JSON object", check.Name)
	}
	sum, err := fileSHA256(check.Path)
	if err != nil {
		return MutationClassGateEvidence{}, nil, "", fmt.Errorf("hash %s evidence: %w", check.Name, err)
	}
	status := classGateString(object, check.StatusField)
	evidence := MutationClassGateEvidence{
		Name:          check.Name,
		Path:          check.Path,
		SchemaVersion: classGateString(object, "schema_version"),
		Status:        status,
		SHA256:        sum,
	}
	if evidence.SchemaVersion != check.SchemaVersion {
		return evidence, object, fmt.Sprintf("%s schema_version must be %s", check.Name, check.SchemaVersion), nil
	}
	if !classGateStringSliceContains(check.ReadyStatuses, status) {
		return evidence, object, fmt.Sprintf("%s status is %s", check.Name, status), nil
	}
	className := classGateString(object, "mutation_class")
	if className == "" && check.Name == "command_readback" {
		className = classGateString(object, "next_class")
	}
	if className == "" {
		return evidence, object, fmt.Sprintf("%s missing mutation_class", check.Name), nil
	}
	if expectedClass != "" && className != expectedClass {
		return evidence, object, fmt.Sprintf("%s mutation_class %s does not match %s", check.Name, className, expectedClass), nil
	}
	if check.Name == "covenant_class_ticket" && classGateBool(object, "consumed") {
		return evidence, object, "covenant_class_ticket is already consumed", nil
	}
	return evidence, object, "", nil
}
