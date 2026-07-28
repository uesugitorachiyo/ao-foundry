package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

func evaluateLowRiskCodeBoundaryChecks(documents map[string]map[string]any, testOnlySuccessReady bool) (*MutationClassBoundaryChecks, []string) {
	atlas := documents["atlas_classification"]
	covenant := documents["covenant_class_ticket"]
	sentinel := documents["sentinel_no_hold"]
	promoter := documents["promoter_ready"]
	rollback := documents["rollback_proof"]
	command := documents["command_readback"]
	ci := documents["ci_passed"]
	checks := &MutationClassBoundaryChecks{
		MutationClass:                       "low_risk_code",
		AtlasClassificationOnly:             classGateString(atlas, "authority_boundary") == "atlas_classification_only" && !classGateBool(atlas, "safe_to_execute"),
		AtlasRequiredGatesComplete:          classGateContainsAll(atlas, "required_gates", []string{"atlas_classification", "test_only_success", "covenant_class_ticket", "sentinel_no_hold", "promoter_ready", "rollback_proof", "command_readback", "ci_passed"}),
		CovenantExactScope:                  classGateNestedBool(covenant, "authority_boundaries", "exact_scope"),
		CovenantClassBound:                  classGateNestedBool(covenant, "authority_boundaries", "class_bound"),
		CovenantDigestBound:                 classGateNestedBool(covenant, "authority_boundaries", "digest_bound"),
		CovenantSingleUse:                   classGateNestedBool(covenant, "authority_boundaries", "single_use"),
		CovenantUnconsumed:                  !classGateBool(covenant, "consumed"),
		CovenantLiveMutationDenied:          !classGateNestedBool(covenant, "authority_boundaries", "live_mutation_grant"),
		SentinelNoHold:                      classGateString(sentinel, "status") == "no_hold" && !classGateBool(sentinel, "hold"),
		PromoterBoundary:                    classGateString(promoter, "promotion_boundary"),
		RollbackPatchPresent:                classGateString(rollback, "rollback_patch") != "",
		RollbackVerificationCommandsPresent: len(classGateStringSlice(rollback, "verification_commands")) > 0,
		CommandReadOnly:                     classGateString(command, "operator_mode") == "read_only",
		CommandCurrentClass:                 classGateString(command, "current_class"),
		CommandNextClass:                    classGateString(command, "next_class"),
		CommandMutatesRepositories:          classGateBool(command, "mutates_repositories"),
		CIPassed:                            classGateString(ci, "status") == "passed",
		CIRequiredChecksPresent:             len(classGateStringSlice(ci, "required_checks")) > 0,
		TestOnlyLiveEvidence:                testOnlySuccessReady,
		SafeToRequest:                       false,
		SafeToExecute:                       false,
	}
	blockers := []string{}
	if !checks.AtlasClassificationOnly {
		blockers = append(blockers, "atlas_classification must be classification-only for low_risk_code")
	}
	if !checks.AtlasRequiredGatesComplete {
		blockers = append(blockers, "atlas_classification missing required low_risk_code gates")
	}
	if !checks.CovenantExactScope || !checks.CovenantClassBound || !checks.CovenantDigestBound || !checks.CovenantSingleUse || !checks.CovenantUnconsumed {
		blockers = append(blockers, "covenant_class_ticket must remain exact-scope, class-bound, digest-bound, unconsumed, and single-use")
	}
	if !checks.CovenantLiveMutationDenied {
		blockers = append(blockers, "covenant_class_ticket must not grant live mutation execution")
	}
	if !checks.SentinelNoHold {
		blockers = append(blockers, "sentinel_no_hold must be an explicit no-hold verdict")
	}
	if checks.PromoterBoundary != "low_risk_code_only" {
		blockers = append(blockers, "promoter_ready must be bounded to low_risk_code_only")
	}
	if !checks.RollbackPatchPresent || !checks.RollbackVerificationCommandsPresent {
		blockers = append(blockers, "rollback_proof requires rollback_patch and verification_commands")
	}
	if !checks.CommandReadOnly || checks.CommandCurrentClass != "test_only" || checks.CommandNextClass != "low_risk_code" || checks.CommandMutatesRepositories {
		blockers = append(blockers, "command_readback must remain read-only from test_only to low_risk_code")
	}
	if !checks.CIPassed || !checks.CIRequiredChecksPresent {
		blockers = append(blockers, "ci_passed must pass and list required checks")
	}
	return checks, blockers
}

func lowRiskCodeDenialAudit(safeToRequest bool) *LowRiskCodeDenialAudit {
	return &LowRiskCodeDenialAudit{
		SchemaVersion:          "ao.foundry.low-risk-code-denial-audit.v0.1",
		Status:                 "blocked",
		MutationClass:          "low_risk_code",
		CurrentProvenLiveClass: "test_only",
		NextDeniedClass:        "low_risk_code",
		SafeToRequest:          safeToRequest,
		SafeToExecute:          false,
		MissingPolicyEvidence: []string{
			"policy:low_risk_code_live_promotion",
			"command_readback:low_risk_code_live",
		},
		MissingRollbackEvidence: []string{
			"rollback_proof:low_risk_code_live",
		},
		MissingSentinelPromoterEvidence: []string{
			"sentinel_clear:low_risk_code_live",
			"promoter_promotion:low_risk_code_live",
		},
		SentinelState:   "missing_live_no_hold",
		PromoterState:   "missing_live_promotion",
		CIRequirements:  []string{"ci_passed:low_risk_code_live"},
		ExactNextAction: "build_low_risk_code_promotion_prerequisites",
		DenialReason:    "low_risk_code live execution remains denied until policy promotion, rollback proof, Sentinel clear verdict, Promoter promotion, Command readback, and PR CI evidence all exist for the exact class scope.",
	}
}

func evaluateTestOnlySuccessEvidence(path string) (MutationClassGateEvidence, string, error) {
	document, err := readArbitraryJSON(path)
	if err != nil {
		return MutationClassGateEvidence{}, "", fmt.Errorf("read test_only_success evidence: %w", err)
	}
	object, ok := document.(map[string]any)
	if !ok {
		return MutationClassGateEvidence{}, "", fmt.Errorf("test_only_success evidence must be a JSON object")
	}
	sum, err := fileSHA256(path)
	if err != nil {
		return MutationClassGateEvidence{}, "", fmt.Errorf("hash test_only_success evidence: %w", err)
	}
	evidence := MutationClassGateEvidence{
		Name:          "test_only_success",
		Path:          path,
		SchemaVersion: classGateString(object, "schema_version"),
		Status:        classGateString(object, "status"),
		SHA256:        sum,
	}
	switch {
	case evidence.SchemaVersion != "ao.foundry.mutation-class-live-success.v0.1":
		return evidence, "test_only_success schema_version must be ao.foundry.mutation-class-live-success.v0.1", nil
	case evidence.Status != "passed":
		return evidence, fmt.Sprintf("test_only_success status is %s", evidence.Status), nil
	case classGateString(object, "proven_live_class") != "test_only" && classGateString(object, "mutation_class") != "test_only":
		return evidence, "test_only_success must prove the test_only live class", nil
	case classGateNestedString(object, "rollback_proof", "status") != "passed":
		return evidence, "test_only_success rollback_proof must pass", nil
	case classGateNestedString(object, "ci_status", "status") != "passed":
		return evidence, "test_only_success ci_status must pass", nil
	default:
		return evidence, "", nil
	}
}

func evaluateLowRiskCodeLiveSuccessEvidence(path string) (MutationClassGateEvidence, *LowRiskCodeLiveSuccessReadback, string, error) {
	document, err := readArbitraryJSON(path)
	if err != nil {
		return MutationClassGateEvidence{}, nil, "", fmt.Errorf("read low_risk_code_live_success evidence: %w", err)
	}
	object, ok := document.(map[string]any)
	if !ok {
		return MutationClassGateEvidence{}, nil, "", errors.New("low_risk_code_live_success evidence must be a JSON object")
	}
	sum, err := fileSHA256(path)
	if err != nil {
		return MutationClassGateEvidence{}, nil, "", fmt.Errorf("hash low_risk_code_live_success evidence: %w", err)
	}
	status := classGateString(object, "status")
	evidence := MutationClassGateEvidence{
		Name:          "low_risk_code_live_success",
		Path:          path,
		SchemaVersion: classGateString(object, "schema_version"),
		Status:        status,
		SHA256:        sum,
	}
	success := &LowRiskCodeLiveSuccessReadback{
		SchemaVersion:     evidence.SchemaVersion,
		Status:            status,
		MutationClass:     classGateString(object, "mutation_class"),
		ProvenLiveClass:   classGateString(object, "proven_live_class"),
		Repo:              classGateString(object, "repo"),
		PullRequest:       classGateString(object, "pull_request"),
		PullRequestNumber: classGateInt(object, "pull_request_number"),
		BaseBranch:        classGateString(object, "base_branch"),
		WorkBranch:        classGateString(object, "work_branch"),
		MergeCommit:       classGateString(object, "merge_commit"),
		MergeState:        classGateString(object, "merge_state"),
		ChangedFiles:      classGateStringSlice(object, "changed_files"),
		FileAllowlist:     classGateStringSlice(object, "file_allowlist"),
	}
	switch {
	case success.SchemaVersion != "ao.foundry.low-risk-code-live-success-readback.v0.1":
		return evidence, success, "low_risk_code_live_success schema_version must be ao.foundry.low-risk-code-live-success-readback.v0.1", nil
	case status != "accepted" && status != "passed":
		return evidence, success, "low_risk_code_live_success status must be accepted", nil
	case success.MutationClass != "low_risk_code" || success.ProvenLiveClass != "low_risk_code":
		return evidence, success, "low_risk_code_live_success mutation_class must be low_risk_code", nil
	case success.Repo != "ao-atlas" || success.PullRequestNumber != 37 || !strings.Contains(success.PullRequest, "/ao-atlas/pull/37"):
		return evidence, success, "low_risk_code_live_success must reference AO Atlas PR #37", nil
	case success.BaseBranch != "main" || success.WorkBranch != "codex/low-risk-code-rehearsal-one" || success.MergeState != "merged" || success.MergeCommit != "a6aee5621dd367a7169f099a87050f1cbd0f88da":
		return evidence, success, "low_risk_code_live_success branch, PR, and merge evidence must match AO Atlas PR #37", nil
	case !equalStringSlices(success.ChangedFiles, []string{"internal/atlas/validate.go"}) || !equalStringSlices(success.FileAllowlist, []string{"internal/atlas/validate.go"}):
		return evidence, success, "low_risk_code_live_success scope must match AO Atlas PR #37", nil
	case classGateNestedString(object, "ci_evidence", "status") != "passed" || len(classGateNestedStringSlice(object, "ci_evidence", "checks")) == 0:
		return evidence, success, "low_risk_code_live_success requires clean main CI evidence", nil
	case !lowRiskLiveRollbackAccepted(object):
		return evidence, success, "low_risk_code_live_success requires rollback proof", nil
	case !lowRiskLiveSentinelAccepted(object):
		return evidence, success, "low_risk_code_live_success requires Sentinel no-hold evidence", nil
	case !lowRiskLivePromoterAccepted(object):
		return evidence, success, "low_risk_code_live_success requires Promoter class-boundary evidence", nil
	case !lowRiskLiveCommandAccepted(object):
		return evidence, success, "low_risk_code_live_success requires Command readback", nil
	case !lowRiskLivePublicSafetyAccepted(object):
		return evidence, success, "low_risk_code_live_success requires public-safety scope validation", nil
	}
	if blocker := validateLowRiskLiveSourceArtifactDigests(path, object); blocker != "" {
		return evidence, success, blocker, nil
	}
	return evidence, success, "", nil
}

func lowRiskLiveRollbackAccepted(object map[string]any) bool {
	rollback, _ := object["rollback_proof"].(map[string]any)
	status := classGateString(rollback, "status")
	return (status == "ready" || status == "passed") && equalStringSlices(classGateStringSlice(rollback, "scope"), []string{"internal/atlas/validate.go"})
}

func lowRiskLiveSentinelAccepted(object map[string]any) bool {
	sentinel, _ := object["sentinel_verdict"].(map[string]any)
	status := classGateString(sentinel, "status")
	return (status == "no_hold" || status == "clear") && !classGateBool(sentinel, "hold_required")
}

func lowRiskLivePromoterAccepted(object map[string]any) bool {
	promoter, _ := object["promoter_verdict"].(map[string]any)
	status := classGateString(promoter, "status")
	boundary := classGateString(promoter, "promotion_boundary")
	return (status == "ready" || status == "passed" || status == "accepted") && boundary != ""
}

func lowRiskLiveCommandAccepted(object map[string]any) bool {
	command, _ := object["command_readback"].(map[string]any)
	return classGateString(command, "status") == "ready" && classGateString(command, "operator_mode") == "read_only"
}

func lowRiskLivePublicSafetyAccepted(object map[string]any) bool {
	publicSafety, _ := object["public_safety_scope"].(map[string]any)
	return classGateString(publicSafety, "status") == "passed" &&
		!classGateBool(publicSafety, "forbidden_surfaces_changed") &&
		!classGateBool(publicSafety, "dependencies_added")
}

func validateLowRiskLiveSourceArtifactDigests(evidencePath string, object map[string]any) string {
	rawArtifacts, _ := object["source_artifacts"].([]any)
	if len(rawArtifacts) == 0 {
		return "low_risk_code_live_success requires digest-bound source artifacts"
	}
	for _, raw := range rawArtifacts {
		artifact, ok := raw.(map[string]any)
		if !ok {
			return "low_risk_code_live_success source artifacts must be objects"
		}
		artifactPath := classGateString(artifact, "path")
		expectedSHA := classGateString(artifact, "sha256")
		if artifactPath == "" || !classGateSHA256Pattern.MatchString(expectedSHA) {
			return "low_risk_code_live_success source artifact path and sha256 are required"
		}
		resolvedPath := artifactPath
		if !filepath.IsAbs(resolvedPath) {
			resolvedPath = filepath.Join(filepath.Dir(evidencePath), filepath.FromSlash(artifactPath))
		}
		actualSHA, err := fileSHA256(resolvedPath)
		if err != nil {
			return "low_risk_code_live_success source artifact is missing"
		}
		if actualSHA != expectedSHA {
			return "low_risk_code_live_success source artifact digest mismatch"
		}
	}
	return ""
}

func evaluateMultiRepoPlanEvidence(path string) (MutationClassGateEvidence, []MutationClassRepoState, *MutationClassRepoSafety, string, error) {
	document, err := readArbitraryJSON(path)
	if err != nil {
		return MutationClassGateEvidence{}, nil, nil, "", fmt.Errorf("read multi_repo_sequencing_plan evidence: %w", err)
	}
	object, ok := document.(map[string]any)
	if !ok {
		return MutationClassGateEvidence{}, nil, nil, "", errors.New("multi_repo_sequencing_plan evidence must be a JSON object")
	}
	sum, err := fileSHA256(path)
	if err != nil {
		return MutationClassGateEvidence{}, nil, nil, "", fmt.Errorf("hash multi_repo_sequencing_plan evidence: %w", err)
	}
	evidence := MutationClassGateEvidence{
		Name:          "multi_repo_sequencing_plan",
		Path:          path,
		SchemaVersion: classGateString(object, "schema_version"),
		Status:        classGateString(object, "status"),
		SHA256:        sum,
	}
	switch {
	case evidence.SchemaVersion != "ao.foundry.multi-repo-low-risk-plan.v0.1":
		return evidence, nil, nil, "multi_repo_sequencing_plan schema_version is " + evidence.SchemaVersion, nil
	case evidence.Status != "ready":
		return evidence, nil, nil, "multi_repo_sequencing_plan status is " + evidence.Status, nil
	case classGateString(object, "mutation_class") != "multi_repo_low_risk":
		return evidence, nil, nil, "multi_repo_sequencing_plan mutation_class must be multi_repo_low_risk", nil
	case classGateBool(object, "schedules_work") || classGateBool(object, "executes_work") || classGateBool(object, "mutates_repositories"):
		return evidence, nil, nil, "multi_repo_sequencing_plan must not schedule, execute, or mutate repositories", nil
	}
	killSwitchState := classGateString(object, "kill_switch_state")
	if killSwitchState != "armed" {
		return evidence, nil, &MutationClassRepoSafety{
			KillSwitchState:                 killSwitchState,
			LiveMultiRepoExecutionAuthority: false,
		}, "multi_repo_sequencing_plan kill switch must be armed", nil
	}
	policyObject, _ := object["concurrency_policy"].(map[string]any)
	maxActiveRepos := classGateInt(policyObject, "max_active_repos")
	concurrentExecutionAllowed := classGateBool(policyObject, "concurrent_execution_allowed")
	requiredSerializedDependencyOrder := classGateBool(policyObject, "required_serialized_dependency_order")
	if concurrentExecutionAllowed || maxActiveRepos != 1 || !requiredSerializedDependencyOrder {
		return evidence, nil, &MutationClassRepoSafety{
			Policy:                             classGateString(policyObject, "policy"),
			MaxActiveRepos:                     maxActiveRepos,
			ConcurrentExecutionAllowed:         concurrentExecutionAllowed,
			UnsafeConcurrentExecutionPrevented: false,
			RequiredSerializedDependencyOrder:  requiredSerializedDependencyOrder,
			KillSwitchState:                    killSwitchState,
			LiveMultiRepoExecutionAuthority:    false,
		}, "multi_repo_sequencing_plan has unsafe concurrent execution", nil
	}
	rawStates, _ := object["repo_states"].([]any)
	if len(rawStates) < 2 {
		return evidence, nil, nil, "multi_repo_sequencing_plan requires at least two repo states", nil
	}
	seenRepos := map[string]bool{}
	readyToExecute := 0
	states := []MutationClassRepoState{}
	for _, rawState := range rawStates {
		stateObject, ok := rawState.(map[string]any)
		if !ok {
			return evidence, states, nil, "multi_repo_sequencing_plan repo_states must be objects", nil
		}
		state := MutationClassRepoState{
			Repo:                   classGateString(stateObject, "repo"),
			Order:                  classGateInt(stateObject, "order"),
			PlannedPR:              classGateString(stateObject, "planned_pr"),
			Status:                 classGateString(stateObject, "status"),
			ExecutionStatus:        classGateString(stateObject, "execution_status"),
			WriteScope:             classGateStringSlice(stateObject, "write_scope"),
			RollbackScope:          classGateStringSlice(stateObject, "rollback_scope"),
			RollbackRequired:       classGateBool(stateObject, "rollback_required"),
			RollbackStatus:         classGateString(stateObject, "rollback_status"),
			RepoStateStatus:        classGateString(stateObject, "repo_state_status"),
			RepoStateObservedAtUTC: classGateString(stateObject, "repo_state_observed_at_utc"),
			RepoStateExpiresAtUTC:  classGateString(stateObject, "repo_state_expires_at_utc"),
			DependsOn:              classGateStringSlice(stateObject, "depends_on"),
			MergeAfter:             classGateStringSlice(stateObject, "merge_after"),
		}
		expectedOrder := len(states) + 1
		switch {
		case state.Repo == "":
			return evidence, states, nil, "multi_repo_sequencing_plan repo_state missing repo", nil
		case seenRepos[state.Repo]:
			return evidence, states, nil, "multi_repo_sequencing_plan duplicate repo " + state.Repo, nil
		case state.Order != expectedOrder:
			return evidence, states, nil, fmt.Sprintf("multi_repo_sequencing_plan repo %s order must be %d", state.Repo, expectedOrder), nil
		case state.PlannedPR == "":
			return evidence, states, nil, "multi_repo_sequencing_plan repo " + state.Repo + " requires planned_pr", nil
		case state.Status != "ready":
			return evidence, states, nil, "multi_repo_sequencing_plan repo " + state.Repo + " status is " + state.Status, nil
		case state.ExecutionStatus == "executing" || state.ExecutionStatus == "active":
			return evidence, states, nil, "multi_repo_sequencing_plan has unsafe concurrent execution", nil
		case state.ExecutionStatus == "ready_to_execute":
			readyToExecute++
		case state.ExecutionStatus != "sequenced_dry_run_only":
			return evidence, states, nil, "multi_repo_sequencing_plan repo " + state.Repo + " execution_status is " + state.ExecutionStatus, nil
		case len(state.WriteScope) == 0 || len(state.RollbackScope) == 0:
			return evidence, states, nil, "multi_repo_sequencing_plan repo " + state.Repo + " requires write_scope and rollback_scope", nil
		case !state.RollbackRequired || state.RollbackStatus != "ready":
			return evidence, states, nil, "multi_repo_sequencing_plan repo " + state.Repo + " requires ready rollback", nil
		case state.RepoStateStatus != "clean_synced" || classGateTimestampExpired(state.RepoStateExpiresAtUTC) || state.RepoStateObservedAtUTC == "":
			return evidence, states, nil, "multi_repo_sequencing_plan repo " + state.Repo + " state evidence is stale", nil
		}
		if !equalStringSlices(state.DependsOn, state.MergeAfter) {
			return evidence, states, nil, "multi_repo_sequencing_plan repo " + state.Repo + " merge_after must match depends_on", nil
		}
		for _, dependency := range state.DependsOn {
			if !seenRepos[dependency] {
				return evidence, states, nil, "multi_repo_sequencing_plan repo " + state.Repo + " dependency " + dependency + " must appear earlier in dependency order", nil
			}
		}
		seenRepos[state.Repo] = true
		states = append(states, state)
	}
	if readyToExecute > 1 {
		return evidence, states, nil, "multi_repo_sequencing_plan has unsafe concurrent execution", nil
	}
	return evidence, states, &MutationClassRepoSafety{
		Policy:                             classGateString(policyObject, "policy"),
		MaxActiveRepos:                     maxActiveRepos,
		ConcurrentExecutionAllowed:         false,
		UnsafeConcurrentExecutionPrevented: true,
		RequiredSerializedDependencyOrder:  requiredSerializedDependencyOrder,
		KillSwitchState:                    killSwitchState,
		LiveMultiRepoExecutionAuthority:    false,
	}, "", nil
}

func evaluateMultiRepoAuthorityEvidence(repoPlan []MutationClassRepoState, rollback map[string]any, ci map[string]any) []string {
	blockers := []string{}
	rollbackByRepo := classGateMapSliceByRepo(rollback["per_repo_rollback"])
	ciByRepo := classGateMapSliceByRepo(ci["per_repo_ci"])
	for _, repoState := range repoPlan {
		rollbackState := rollbackByRepo[repoState.Repo]
		if rollbackState == nil || classGateString(rollbackState, "status") != "ready" || len(classGateStringSlice(rollbackState, "rollback_scope")) == 0 {
			blockers = append(blockers, "per_repo_rollback missing ready rollback for "+repoState.Repo)
		}
		ciState := ciByRepo[repoState.Repo]
		ciStatus := classGateString(ciState, "status")
		if ciState == nil || !classGateBool(ciState, "required") || (ciStatus != "passed" && ciStatus != "success") {
			blockers = append(blockers, "per_repo_ci missing passing CI for "+repoState.Repo)
		}
	}
	return blockers
}
