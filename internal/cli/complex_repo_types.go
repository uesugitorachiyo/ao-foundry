package cli

type MutationClassGate struct {
	SchemaVersion         string                          `json:"schema_version"`
	Status                string                          `json:"status"`
	MutationClass         string                          `json:"mutation_class"`
	SafeToRequest         bool                            `json:"safe_to_request"`
	SafeToExecute         bool                            `json:"safe_to_execute"`
	FirstFailingCheck     string                          `json:"first_failing_check"`
	RequiredEvidence      []string                        `json:"required_evidence"`
	SourceEvidence        []MutationClassGateEvidence     `json:"source_evidence"`
	ClassBoundaryChecks   *MutationClassBoundaryChecks    `json:"class_boundary_checks,omitempty"`
	DeniedClasses         []string                        `json:"denied_classes"`
	AuthorityBoundary     string                          `json:"authority_boundary"`
	NextActions           []string                        `json:"next_actions"`
	DenialAudit           *LowRiskCodeDenialAudit         `json:"denial_audit,omitempty"`
	LiveRehearsalDecision *MultiRepoLiveRehearsalDecision `json:"live_rehearsal_decision,omitempty"`
	LowRiskLiveSuccess    *LowRiskCodeLiveSuccessReadback `json:"low_risk_code_live_success,omitempty"`
	ComplexNodeGate       *ComplexRepoMutationNodeGate    `json:"complex_node_gate,omitempty"`
	RepoExecutionPlan     []MutationClassRepoState        `json:"repo_execution_plan,omitempty"`
	RepoSafety            *MutationClassRepoSafety        `json:"repo_safety,omitempty"`
	SchedulesWork         bool                            `json:"schedules_work"`
	ExecutesWork          bool                            `json:"executes_work"`
	ApprovesWork          bool                            `json:"approves_work"`
	MutatesRepositories   bool                            `json:"mutates_repositories"`
}

type LowRiskCodeDenialAudit struct {
	SchemaVersion                   string   `json:"schema_version"`
	Status                          string   `json:"status"`
	MutationClass                   string   `json:"mutation_class"`
	CurrentProvenLiveClass          string   `json:"current_proven_live_class"`
	NextDeniedClass                 string   `json:"next_denied_class"`
	SafeToRequest                   bool     `json:"safe_to_request"`
	SafeToExecute                   bool     `json:"safe_to_execute"`
	MissingPolicyEvidence           []string `json:"missing_policy_evidence"`
	MissingRollbackEvidence         []string `json:"missing_rollback_evidence"`
	MissingSentinelPromoterEvidence []string `json:"missing_sentinel_promoter_evidence"`
	SentinelState                   string   `json:"sentinel_state"`
	PromoterState                   string   `json:"promoter_state"`
	CIRequirements                  []string `json:"ci_requirements"`
	ExactNextAction                 string   `json:"exact_next_action"`
	DenialReason                    string   `json:"denial_reason"`
}

type MultiRepoLiveRehearsalDecision struct {
	SchemaVersion                string   `json:"schema_version"`
	Status                       string   `json:"status"`
	MutationClass                string   `json:"mutation_class"`
	CurrentClass                 string   `json:"current_class"`
	NextClass                    string   `json:"next_class"`
	CurrentProvenLiveClass       string   `json:"current_proven_live_class"`
	LowerClassLiveEvidenceStatus string   `json:"lower_class_live_evidence_status"`
	SafeToRequest                bool     `json:"safe_to_request"`
	SafeToExecute                bool     `json:"safe_to_execute"`
	LiveExecutionAuthority       bool     `json:"live_execution_authority"`
	MissingEvidence              []string `json:"missing_evidence"`
	DenialReason                 string   `json:"denial_reason"`
	ExactNextAction              string   `json:"exact_next_action"`
	RepoExecutionPolicy          string   `json:"repo_execution_policy"`
	SchedulesWork                bool     `json:"schedules_work"`
	ExecutesWork                 bool     `json:"executes_work"`
	MutatesRepositories          bool     `json:"mutates_repositories"`
}

type LowRiskCodeLiveSuccessReadback struct {
	SchemaVersion     string   `json:"schema_version"`
	Status            string   `json:"status"`
	MutationClass     string   `json:"mutation_class"`
	ProvenLiveClass   string   `json:"proven_live_class"`
	Repo              string   `json:"repo"`
	PullRequest       string   `json:"pull_request"`
	PullRequestNumber int      `json:"pull_request_number"`
	BaseBranch        string   `json:"base_branch"`
	WorkBranch        string   `json:"work_branch"`
	MergeCommit       string   `json:"merge_commit"`
	MergeState        string   `json:"merge_state"`
	ChangedFiles      []string `json:"changed_files"`
	FileAllowlist     []string `json:"file_allowlist"`
}

type MutationClassGateEvidence struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	SchemaVersion string `json:"schema_version"`
	Status        string `json:"status"`
	SHA256        string `json:"sha256"`
}

type ComplexRepoMutationNodeGate struct {
	SchemaVersion                    string                      `json:"schema_version"`
	Status                           string                      `json:"status"`
	MutationClass                    string                      `json:"mutation_class"`
	HighestProvenLiveClass           string                      `json:"highest_proven_live_class"`
	NextDeniedClass                  string                      `json:"next_denied_class"`
	WorkgraphID                      string                      `json:"workgraph_id"`
	NodeID                           string                      `json:"node_id"`
	TaskID                           string                      `json:"task_id"`
	TargetFactoryRepo                string                      `json:"target_factory_repo,omitempty"`
	SafeToRequest                    bool                        `json:"safe_to_request"`
	SafeToExecute                    bool                        `json:"safe_to_execute"`
	LiveExecutionAuthority           bool                        `json:"live_execution_authority"`
	FirstFailingCheck                string                      `json:"first_failing_check"`
	Blockers                         []string                    `json:"blockers"`
	ExactNextAction                  string                      `json:"exact_next_action"`
	FoundryImportID                  string                      `json:"foundry_import_id"`
	FoundryImportStatus              string                      `json:"foundry_import_status"`
	FoundryImportTaskCount           int                         `json:"foundry_import_task_count"`
	FoundryImportSchedulesWork       bool                        `json:"foundry_import_schedules_work"`
	FoundryImportExecutesWork        bool                        `json:"foundry_import_executes_work"`
	FoundryImportApprovesWork        bool                        `json:"foundry_import_approves_work"`
	CandidateStatus                  string                      `json:"candidate_status"`
	CandidateExecutableReady         bool                        `json:"candidate_executable_ready"`
	CandidateSafeToExecute           bool                        `json:"candidate_safe_to_execute"`
	RollbackStatus                   string                      `json:"rollback_status"`
	RollbackSafeToExecute            bool                        `json:"rollback_safe_to_execute"`
	AuthorityBoundary                string                      `json:"authority_boundary"`
	RequiredGates                    []string                    `json:"required_gates"`
	SourceEvidence                   []MutationClassGateEvidence `json:"source_evidence"`
	SchedulesWork                    bool                        `json:"schedules_work"`
	ExecutesWork                     bool                        `json:"executes_work"`
	ApprovesWork                     bool                        `json:"approves_work"`
	MutatesRepositories              bool                        `json:"mutates_repositories"`
	FullyUnsupervisedComplexMutation string                      `json:"fully_unsupervised_complex_mutation"`
	RSI                              string                      `json:"rsi"`
}

type ComplexRepoMutationPromotionRollup struct {
	SchemaVersion                    string                          `json:"schema_version"`
	Status                           string                          `json:"status"`
	MutationClass                    string                          `json:"mutation_class"`
	SafeToPromote                    bool                            `json:"safe_to_promote"`
	ComplexRepoMutationLiveProven    bool                            `json:"complex_repo_mutation_live_proven"`
	HighestProvenLiveClass           string                          `json:"highest_proven_live_class"`
	NextDeniedClass                  string                          `json:"next_denied_class"`
	FullyUnsupervisedComplexMutation string                          `json:"fully_unsupervised_complex_mutation"`
	RSI                              string                          `json:"rsi"`
	Mission                          string                          `json:"mission"`
	CompletedNodes                   int                             `json:"completed_nodes"`
	TotalNodes                       int                             `json:"total_nodes"`
	FirstFailingCheck                string                          `json:"first_failing_check"`
	Blockers                         []string                        `json:"blockers"`
	Checks                           map[string]bool                 `json:"checks"`
	Nodes                            []ComplexRepoMutationRollupNode `json:"nodes"`
	SourceEvidence                   []MutationClassGateEvidence     `json:"source_evidence"`
	AuthorityBoundaries              map[string]bool                 `json:"authority_boundaries"`
	PromoterVerdictReady             bool                            `json:"promoter_verdict_ready"`
	CommandReadbackReady             bool                            `json:"command_readback_ready"`
	PublicWordingReview              string                          `json:"public_wording_review"`
	EvaluatedAtUTC                   string                          `json:"evaluated_at_utc"`
}

type ComplexRepoMutationRollupNode struct {
	NodeID                 string `json:"node_id"`
	TaskID                 string `json:"task_id"`
	Status                 string `json:"status"`
	ChangedFile            string `json:"changed_file"`
	PullRequest            string `json:"pull_request"`
	MergeCommit            string `json:"merge_commit"`
	CI                     string `json:"ci"`
	NodeGatePath           string `json:"node_gate_path"`
	NodeGateSHA256         string `json:"node_gate_sha256"`
	RunLinkPath            string `json:"run_link_path"`
	RunLinkSHA256          string `json:"run_link_sha256"`
	SafeToExecuteBeforeRun bool   `json:"safe_to_execute_before_run"`
	RollbackEvidence       string `json:"rollback_evidence"`
	SentinelEvidence       string `json:"sentinel_evidence"`
	PromoterEvidence       string `json:"promoter_evidence"`
	CommandReadback        string `json:"command_readback"`
}

type MutationClassBoundaryChecks struct {
	MutationClass                       string `json:"mutation_class"`
	AtlasClassificationOnly             bool   `json:"atlas_classification_only"`
	AtlasRequiredGatesComplete          bool   `json:"atlas_required_gates_complete"`
	CovenantExactScope                  bool   `json:"covenant_exact_scope"`
	CovenantClassBound                  bool   `json:"covenant_class_bound"`
	CovenantDigestBound                 bool   `json:"covenant_digest_bound"`
	CovenantSingleUse                   bool   `json:"covenant_single_use"`
	CovenantUnconsumed                  bool   `json:"covenant_unconsumed"`
	CovenantLiveMutationDenied          bool   `json:"covenant_live_mutation_denied"`
	SentinelNoHold                      bool   `json:"sentinel_no_hold"`
	PromoterBoundary                    string `json:"promoter_boundary"`
	RollbackPatchPresent                bool   `json:"rollback_patch_present"`
	RollbackVerificationCommandsPresent bool   `json:"rollback_verification_commands_present"`
	CommandReadOnly                     bool   `json:"command_read_only"`
	CommandCurrentClass                 string `json:"command_current_class"`
	CommandNextClass                    string `json:"command_next_class"`
	CommandMutatesRepositories          bool   `json:"command_mutates_repositories"`
	CIPassed                            bool   `json:"ci_passed"`
	CIRequiredChecksPresent             bool   `json:"ci_required_checks_present"`
	TestOnlyLiveEvidence                bool   `json:"test_only_live_evidence"`
	SafeToRequest                       bool   `json:"safe_to_request"`
	SafeToExecute                       bool   `json:"safe_to_execute"`
}

type MutationClassRepoState struct {
	Repo                   string   `json:"repo"`
	Order                  int      `json:"order"`
	PlannedPR              string   `json:"planned_pr"`
	Status                 string   `json:"status"`
	ExecutionStatus        string   `json:"execution_status"`
	WriteScope             []string `json:"write_scope"`
	RollbackScope          []string `json:"rollback_scope"`
	RollbackRequired       bool     `json:"rollback_required"`
	RollbackStatus         string   `json:"rollback_status"`
	RepoStateStatus        string   `json:"repo_state_status"`
	RepoStateObservedAtUTC string   `json:"repo_state_observed_at_utc"`
	RepoStateExpiresAtUTC  string   `json:"repo_state_expires_at_utc"`
	DependsOn              []string `json:"depends_on"`
	MergeAfter             []string `json:"merge_after"`
}

type MutationClassRepoSafety struct {
	Policy                             string `json:"policy"`
	MaxActiveRepos                     int    `json:"max_active_repos"`
	ConcurrentExecutionAllowed         bool   `json:"concurrent_execution_allowed"`
	UnsafeConcurrentExecutionPrevented bool   `json:"unsafe_concurrent_execution_prevented"`
	RequiredSerializedDependencyOrder  bool   `json:"required_serialized_dependency_order"`
	KillSwitchState                    string `json:"kill_switch_state"`
	LiveMultiRepoExecutionAuthority    bool   `json:"live_multi_repo_execution_authority"`
}
