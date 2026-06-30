package cli

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	registrySchema                   = "ao.foundry.registry.v0.1"
	taskSchema                       = "ao.foundry.task.v0.1"
	readinessSchema                  = "ao.foundry.production-readiness-audit.v0.1"
	goalRunSchema                    = "ao.foundry.goal-run.v0.1"
	goalReadinessSchema              = "ao.foundry.goal-readiness-audit.v0.1"
	runSchema                        = "ao.foundry.run.v0.1"
	repoHealthSchema                 = "ao.foundry.repo-health.v0.1"
	repoBoardSchema                  = "ao.foundry.repo-board.v0.1"
	loopLeaseSchema                  = "ao.foundry.loop-lease.v0.1"
	forgePacketSchema                = "ao.forge.factory-packet.v0.1"
	pulseEventSchema                 = "ao.foundry.pulse-event.v0.1"
	atlasImportSchema                = "ao.atlas.foundry-import.v0.1"
	atlasBlueprintImportSchema       = "ao.atlas.blueprint-import.v0.1"
	atlasTaskSchema                  = "ao.atlas.factory-task.v0.1"
	atlasRunLinkSchema               = "ao.atlas.run-link.v0.1"
	atlasReadbackSchema              = "ao.foundry.atlas-readback.v0.1"
	atlasStatusSchema                = "ao.foundry.atlas-status.v0.1"
	pulseIntakeSchema                = "ao.foundry.pulse-intake-preflight.v0.1"
	pulseLifecycleSchema             = "ao.foundry.pulse-pr-lifecycle.v0.1"
	pulseStartGateSchema             = "ao.foundry.pulse-overnight-start-gate.v0.1"
	pulseLoopPolicySchema            = "ao.foundry.pulse-event-loop-policy.v0.1"
	pulseRunnerSchema                = "ao.foundry.pulse-runner-start-decision.v0.1"
	classGateSchema                  = "ao.foundry.mutation-class-gate.v0.1"
	complexNodeGateSchema            = "ao.foundry.complex-repo-mutation-node-gate.v0.1"
	complexPromotionRollupSchema     = "ao.foundry.complex-repo-mutation-promotion-rollup.v0.1"
	complexClosureManifestSchema     = "ao.foundry.complex-repo-mutation-node-closure-manifest.v0.1"
	complexRollbackClosureSchema     = "ao.foundry.complex-repo-mutation-node-rollback-closure.v0.1"
	complexSentinelClosureSchema     = "ao.sentinel.complex-repo-mutation-node-closure-verdict.v0.1"
	complexPromoterClosureSchema     = "ao.promoter.complex-repo-mutation-node-closure-verdict.v0.1"
	complexCommandClosureSchema      = "ao.command.complex-repo-mutation-node-closure-readback.v0.1"
	fullyUnsupervisedReadinessSchema = "ao.foundry.fully-unsupervised-complex-readiness-rollup.v0.1"
	fullyUnsupervisedCommandSchema   = "ao.command.fully-unsupervised-complex-readiness-readback.v0.1"
)

var classGateSHA256Pattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

const liveEvidenceFreshnessWindow = 24 * time.Hour

type Registry struct {
	SchemaVersion string `json:"schema_version"`
	FoundryID     string `json:"foundry_id"`
	Name          string `json:"name"`
	Repos         []Repo `json:"repos"`
}

type Repo struct {
	ID                string             `json:"id"`
	Name              string             `json:"name"`
	Role              string             `json:"role"`
	DelegatesTo       string             `json:"delegates_to"`
	Workspace         string             `json:"workspace"`
	Branches          []string           `json:"branches"`
	EvidenceSources   []EvidenceSource   `json:"evidence_sources"`
	AllowedAutomation []string           `json:"allowed_automation"`
	ReadinessSignals  []ReadinessSignal  `json:"readiness_signals"`
	Health            HealthReaderConfig `json:"health,omitempty"`
}

type EvidenceSource struct {
	Kind     string `json:"kind"`
	Location string `json:"location"`
	Owner    string `json:"owner"`
}

type ReadinessSignal struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Source string `json:"source"`
}

type Task struct {
	SchemaVersion      string       `json:"schema_version"`
	TaskID             string       `json:"task_id"`
	Title              string       `json:"title"`
	Objective          string       `json:"objective"`
	Priority           string       `json:"priority"`
	State              string       `json:"state"`
	TargetRepos        []string     `json:"target_repos"`
	RequiredDelegation []Delegation `json:"required_delegations"`
	Acceptance         []string     `json:"acceptance"`
	Verification       []string     `json:"verification"`
	Safety             TaskSafety   `json:"safety"`
}

type Delegation struct {
	DelegateTo string `json:"delegate_to"`
	Reason     string `json:"reason"`
}

type TaskSafety struct {
	LocalOnly            bool     `json:"local_only"`
	AllowedWriteRoots    []string `json:"allowed_write_roots"`
	ForbiddenActions     []string `json:"forbidden_actions"`
	AllowNetwork         bool     `json:"allow_network,omitempty"`
	AllowReleaseMutation bool     `json:"allow_release_mutation,omitempty"`
}

type AtlasFoundryImport struct {
	ContractVersion string                   `json:"contract_version"`
	ID              string                   `json:"id"`
	WorkgraphID     string                   `json:"workgraph_id"`
	TargetInstance  string                   `json:"target_instance"`
	Status          string                   `json:"status"`
	SourceArtifacts []AtlasSourceArtifact    `json:"source_artifacts"`
	Tasks           []AtlasImportTaskFixture `json:"tasks"`
	SchedulesWork   bool                     `json:"schedules_work"`
	ExecutesWork    bool                     `json:"executes_work"`
	ApprovesWork    bool                     `json:"approves_work"`
}

type AtlasBlueprintImport struct {
	ContractVersion         string              `json:"contract_version"`
	ID                      string              `json:"id"`
	ProjectID               string              `json:"project_id"`
	Status                  string              `json:"status"`
	Reason                  string              `json:"reason"`
	BlueprintPack           AtlasSourceArtifact `json:"blueprint_pack"`
	BuildAuthorization      AtlasSourceArtifact `json:"build_authorization"`
	TargetInstance          string              `json:"target_instance"`
	WorkgraphID             string              `json:"workgraph_id"`
	MutationClass           string              `json:"mutation_class"`
	DownstreamFoundryImport AtlasSourceArtifact `json:"downstream_foundry_import"`
	Digests                 map[string]string   `json:"digests"`
	SafetyLimits            []string            `json:"safety_limits"`
	ReadyForFoundry         bool                `json:"ready_for_foundry"`
	SafeToExecute           bool                `json:"safe_to_execute"`
	LiveExecutionProven     bool                `json:"live_execution_proven"`
	SchedulesWork           bool                `json:"schedules_work"`
	ExecutesWork            bool                `json:"executes_work"`
	ApprovesWork            bool                `json:"approves_work"`
	MutatesRepositories     bool                `json:"mutates_repositories"`
	CallsProviders          bool                `json:"calls_providers"`
	ReleaseOrPublishAllowed bool                `json:"release_or_publish_allowed"`
}

type AtlasSourceArtifact struct {
	Ref    string `json:"ref"`
	Digest string `json:"digest"`
}

type AtlasImportTaskFixture struct {
	NodeID            string           `json:"node_id"`
	TaskID            string           `json:"task_id"`
	Path              string           `json:"path"`
	MutationClass     string           `json:"mutation_class"`
	WriteScope        []string         `json:"write_scope"`
	RollbackScope     []string         `json:"rollback_scope"`
	RequiredGates     []string         `json:"required_gates"`
	RequiredEvidence  []string         `json:"required_evidence"`
	AuthorityBoundary string           `json:"authority_boundary"`
	Task              AtlasFactoryTask `json:"task"`
	TaskDigest        string           `json:"task_digest"`
}

type AtlasFactoryTask struct {
	ContractVersion   string   `json:"contract_version"`
	ID                string   `json:"id"`
	Objective         string   `json:"objective"`
	TargetFactoryRepo string   `json:"target_factory_repo"`
	FactoryFolder     string   `json:"factory_folder"`
	MutationClass     string   `json:"mutation_class"`
	Acceptance        []string `json:"acceptance_criteria"`
	NonGoals          []string `json:"non_goals"`
	WriteScope        []string `json:"write_scope"`
	RequiredGates     []string `json:"required_gates"`
	RollbackScope     []string `json:"rollback_scope"`
	Verification      []string `json:"verification_commands"`
	RequiredEvidence  []string `json:"required_evidence"`
	SafetyLimits      []string `json:"safety_limits"`
	AuthorityBoundary string   `json:"authority_boundary"`
	DependencyRefs    []string `json:"dependency_refs"`
	ContextPackRefs   []string `json:"context_pack_refs"`
}

type AtlasRunLink struct {
	ContractVersion string            `json:"contract_version"`
	TaskID          string            `json:"task_id"`
	Status          string            `json:"status"`
	Evidence        map[string]string `json:"evidence"`
	Digest          string            `json:"digest"`
}

type AtlasReadback struct {
	SchemaVersion  string            `json:"schema_version"`
	Status         string            `json:"status"`
	Mode           string            `json:"mode"`
	AtlasImportID  string            `json:"atlas_import_id"`
	WorkgraphID    string            `json:"workgraph_id"`
	TargetInstance string            `json:"target_instance"`
	TaskID         string            `json:"task_id"`
	TaskDigest     string            `json:"task_digest"`
	RunLinkDigest  string            `json:"run_link_digest"`
	Evidence       map[string]string `json:"evidence"`
	SchedulesWork  bool              `json:"schedules_work"`
	ExecutesWork   bool              `json:"executes_work"`
	ApprovesWork   bool              `json:"approves_work"`
	NextActions    []string          `json:"next_actions"`
}

type AtlasStatus struct {
	SchemaVersion  string            `json:"schema_version"`
	Status         string            `json:"status"`
	Mode           string            `json:"mode"`
	RegistryID     string            `json:"registry_id"`
	ImportID       string            `json:"import_id"`
	WorkgraphID    string            `json:"workgraph_id"`
	TargetInstance string            `json:"target_instance"`
	ReadbackStatus string            `json:"readback_status"`
	TaskID         string            `json:"task_id"`
	TaskDigest     string            `json:"task_digest"`
	RunLinkDigest  string            `json:"run_link_digest"`
	SchedulesWork  bool              `json:"schedules_work"`
	ExecutesWork   bool              `json:"executes_work"`
	ApprovesWork   bool              `json:"approves_work"`
	Evidence       map[string]string `json:"evidence"`
	NextActions    []string          `json:"next_actions"`
}

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

type ReadinessAudit struct {
	SchemaVersion string           `json:"schema_version"`
	Status        string           `json:"status"`
	Score         int              `json:"score"`
	MaxScore      int              `json:"max_score"`
	RegistryID    string           `json:"registry_id"`
	TaskID        string           `json:"task_id"`
	Checks        []ReadinessCheck `json:"checks"`
	NextActions   []string         `json:"next_actions"`
}

type ReadinessCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Score  int    `json:"score"`
	Reason string `json:"reason"`
}

type GoalRun struct {
	SchemaVersion      string            `json:"schema_version"`
	GoalID             string            `json:"goal_id"`
	Objective          string            `json:"objective"`
	AcceptanceCriteria []string          `json:"acceptance_criteria"`
	AllowedScope       []string          `json:"allowed_scope"`
	StopConditions     []string          `json:"stop_conditions"`
	CurrentPhase       string            `json:"current_phase"`
	NextTask           string            `json:"next_task"`
	ContinuationPrompt string            `json:"continuation_prompt"`
	LoopOwner          string            `json:"loop_owner"`
	NextActionGuard    string            `json:"next_action_guard"`
	LastIteration      GoalLastIteration `json:"last_iteration"`
	LoopPolicy         LoopPolicy        `json:"loop_policy,omitempty"`
}

type GoalLastIteration struct {
	Evidence []EvidenceRef `json:"evidence"`
}

type LoopPolicy struct {
	MaxIterations     int `json:"max_iterations,omitempty"`
	Iterations        int `json:"iterations,omitempty"`
	MaxElapsedMinutes int `json:"max_elapsed_minutes,omitempty"`
	ElapsedMinutes    int `json:"elapsed_minutes,omitempty"`
	MaxSpendCents     int `json:"max_spend_cents,omitempty"`
	SpendCents        int `json:"spend_cents,omitempty"`
}

type LoopLease struct {
	SchemaVersion string `json:"schema_version"`
	GoalID        string `json:"goal_id"`
	LeaseID       string `json:"lease_id"`
	AcquiredAtUTC string `json:"acquired_at_utc"`
	ExpiresAtUTC  string `json:"expires_at_utc"`
	Status        string `json:"status"`
}

type ApprovalRequest struct {
	SchemaVersion        string   `json:"schema_version"`
	TaskID               string   `json:"task_id"`
	TaskSHA256           string   `json:"task_sha256"`
	RequestedSideEffects []string `json:"requested_side_effects"`
	Reason               string   `json:"reason"`
}

type ApprovalDecision struct {
	SchemaVersion        string   `json:"schema_version"`
	TaskID               string   `json:"task_id"`
	TaskSHA256           string   `json:"task_sha256"`
	RequestedSideEffects []string `json:"requested_side_effects"`
	ApprovedSideEffects  []string `json:"approved_side_effects"`
	ExpiresAtUTC         string   `json:"expires_at_utc"`
	Operator             string   `json:"operator"`
	Reason               string   `json:"reason"`
	Decision             string   `json:"decision"`
}

type EvalScorecard struct {
	SchemaVersion string             `json:"schema_version"`
	ScorecardID   string             `json:"scorecard_id"`
	Threshold     int                `json:"threshold"`
	Dimensions    []EvalDimensionDef `json:"dimensions"`
}

type EvalDimensionDef struct {
	Name     string `json:"name"`
	MaxScore int    `json:"max_score"`
}

type EvalResult struct {
	SchemaVersion string          `json:"schema_version"`
	ScorecardID   string          `json:"scorecard_id"`
	RunID         string          `json:"run_id"`
	Status        string          `json:"status"`
	Score         int             `json:"score"`
	MaxScore      int             `json:"max_score"`
	Threshold     int             `json:"threshold"`
	Dimensions    []EvalDimension `json:"dimensions"`
	NextActions   []string        `json:"next_actions"`
}

type RSIImprovementGate struct {
	SchemaVersion              string                `json:"schema_version"`
	Status                     string                `json:"status"`
	BaselineScorePercent       float64               `json:"baseline_score_percent"`
	CandidateScorePercent      float64               `json:"candidate_score_percent"`
	RequiredImprovementPercent float64               `json:"required_improvement_percent"`
	ActualImprovementPercent   float64               `json:"actual_improvement_percent"`
	AutonomousClaim            string                `json:"autonomous_claim"`
	MutatesRepositories        bool                  `json:"mutates_repositories"`
	Evidence                   []RSIImprovementProof `json:"evidence"`
	NextActions                []string              `json:"next_actions"`
}

type RSIImprovementProof struct {
	Label         string `json:"label"`
	Path          string `json:"path"`
	SchemaVersion string `json:"schema_version"`
	Status        string `json:"status"`
	Score         int    `json:"score"`
	MaxScore      int    `json:"max_score"`
	SHA256        string `json:"sha256"`
}

type RSICandidate struct {
	SchemaVersion         string                 `json:"schema_version"`
	Status                string                 `json:"status"`
	GeneratedBy           string                 `json:"generated_by"`
	ImprovementHypothesis string                 `json:"improvement_hypothesis"`
	BaselineEvalResult    RSICandidateEvalResult `json:"baseline_eval_result"`
	CandidateEvalResult   RSICandidateEvalResult `json:"candidate_eval_result"`
	MutatesRepositories   bool                   `json:"mutates_repositories"`
	NextActions           []string               `json:"next_actions"`
}

type RSICandidateEvalResult struct {
	Path          string `json:"path"`
	SchemaVersion string `json:"schema_version"`
	Status        string `json:"status"`
	Score         int    `json:"score"`
	MaxScore      int    `json:"max_score"`
	SHA256        string `json:"sha256"`
}

type RSINextImprovementTask struct {
	SchemaVersion              string   `json:"schema_version"`
	Status                     string   `json:"status"`
	GeneratedBy                string   `json:"generated_by"`
	GoalID                     string   `json:"goal_id"`
	RecommendedTaskID          string   `json:"recommended_task_id"`
	RecommendedAction          string   `json:"recommended_action"`
	ImprovementRationale       string   `json:"improvement_rationale"`
	CandidateEvidencePath      string   `json:"candidate_evidence_path"`
	GateEvidencePath           string   `json:"gate_evidence_path"`
	RequiredImprovementPercent float64  `json:"required_improvement_percent"`
	ActualImprovementPercent   float64  `json:"actual_improvement_percent"`
	AutonomousClaim            string   `json:"autonomous_claim"`
	MutatesRepositories        bool     `json:"mutates_repositories"`
	NextActions                []string `json:"next_actions"`
}

type TraceSpan struct {
	SchemaVersion string            `json:"schema_version"`
	TraceID       string            `json:"trace_id"`
	SpanID        string            `json:"span_id"`
	ParentSpanID  string            `json:"parent_span_id,omitempty"`
	Component     string            `json:"component"`
	Operation     string            `json:"operation"`
	Status        string            `json:"status"`
	StartedAt     string            `json:"started_at"`
	EndedAt       string            `json:"ended_at"`
	DurationMS    int               `json:"duration_ms"`
	Attributes    map[string]string `json:"attributes"`
	EvidenceRefs  []string          `json:"evidence_refs"`
	Problem       string            `json:"problem,omitempty"`
}

type AO2SDDPlan struct {
	SchemaVersion string `json:"schema_version"`
	PlanID        string `json:"plan_id"`
	Prompt        struct {
		Text string `json:"text"`
	} `json:"prompt"`
	Target struct {
		RepoPath string `json:"repo_path"`
	} `json:"target"`
	Plan struct {
		Title string `json:"title"`
		Goal  string `json:"goal"`
		Steps []struct {
			Acceptance []string `json:"acceptance"`
		} `json:"steps"`
	} `json:"plan"`
}

type CapabilityMatrix struct {
	SchemaVersion string              `json:"schema_version"`
	Capabilities  []CapabilityMapping `json:"capabilities"`
}

type DemoStatus struct {
	SchemaVersion string   `json:"schema_version"`
	RegistryID    string   `json:"registry_id"`
	TaskID        string   `json:"task_id"`
	RunID         string   `json:"run_id"`
	Status        string   `json:"status"`
	Story         []string `json:"story"`
	NextAction    string   `json:"next_action"`
}

type ReleaseManifest struct {
	SchemaVersion string             `json:"schema_version"`
	Status        string             `json:"status"`
	Files         []ReleaseFileEntry `json:"files"`
	Checks        []string           `json:"checks"`
}

type ReleaseFileEntry struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type ReleaseCandidateLedger struct {
	SchemaVersion string                 `json:"schema_version"`
	CandidateID   string                 `json:"candidate_id"`
	Status        string                 `json:"status"`
	ActiveSpine   []ReleaseCandidateRepo `json:"active_spine"`
	Gates         []ReleaseCandidateGate `json:"gates"`
	NextActions   []string               `json:"next_actions"`
}

type ReleaseCandidateRepo struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Role     string   `json:"role"`
	Status   string   `json:"status"`
	Evidence []string `json:"evidence"`
}

type ReleaseCandidateGate struct {
	Name                    string   `json:"name"`
	Status                  string   `json:"status"`
	RequiredBeforePromotion bool     `json:"required_before_promotion"`
	Evidence                []string `json:"evidence"`
}

type ReleasePromotionLedger struct {
	SchemaVersion            string                     `json:"schema_version"`
	CandidateID              string                     `json:"candidate_id"`
	Status                   string                     `json:"status"`
	ReleaseSafe              bool                       `json:"release_safe"`
	SignedSmokePulseID       string                     `json:"signed_smoke_pulse_id"`
	SignedSmokeSummaryStatus string                     `json:"signed_smoke_summary_status"`
	PulseStatus              string                     `json:"pulse_status"`
	Evidence                 []ReleasePromotionEvidence `json:"evidence"`
	NextActions              []string                   `json:"next_actions"`
}

type ReleasePromotionEvidence struct {
	Name          string `json:"name"`
	Status        string `json:"status"`
	SchemaVersion string `json:"schema_version"`
}

type CompetitiveReadinessAudit struct {
	SchemaVersion string                     `json:"schema_version"`
	Status        string                     `json:"status"`
	Score         int                        `json:"score"`
	MaxScore      int                        `json:"max_score"`
	Categories    []CompetitiveAuditCategory `json:"categories"`
	NextActions   []string                   `json:"next_actions"`
}

type CompetitiveAuditCategory struct {
	ID          string                  `json:"id"`
	Status      string                  `json:"status"`
	Score       int                     `json:"score"`
	MaxScore    int                     `json:"max_score"`
	Checks      []CompetitiveAuditCheck `json:"checks"`
	NextActions []string                `json:"next_actions"`
}

type CompetitiveAuditCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type PulseEvent struct {
	SchemaVersion string                `json:"schema_version"`
	PulseID       string                `json:"pulse_id"`
	Status        string                `json:"status"`
	Score         int                   `json:"score"`
	MaxScore      int                   `json:"max_score"`
	RegistryID    string                `json:"registry_id"`
	TaskID        string                `json:"task_id"`
	GoalID        string                `json:"goal_id"`
	Artifacts     []PulseArtifact       `json:"artifacts"`
	Checks        []PulseCheck          `json:"checks"`
	Freshness     PulseFreshnessSummary `json:"freshness_summary"`
	NextAction    string                `json:"next_action"`
}

type PulseFreshnessSummary struct {
	SchemaVersion        string `json:"schema_version"`
	Status               string `json:"status"`
	ForgeLivePacket      string `json:"forge_live_packet"`
	ControlPlaneReadback string `json:"control_plane_readback"`
	Explanation          string `json:"explanation"`
}

type PulseArtifact struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	SHA256        string `json:"sha256"`
	SchemaVersion string `json:"schema_version"`
	Status        string `json:"status"`
}

type PulseCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type TraceInspectSummary struct {
	SchemaVersion string `json:"schema_version"`
	Status        string `json:"status"`
	Spans         int    `json:"spans"`
	FailedSpans   int    `json:"failed_spans"`
	EvidenceRefs  int    `json:"evidence_refs"`
}

type PolicyGateSummary struct {
	SchemaVersion string        `json:"schema_version"`
	Status        string        `json:"status"`
	Decisions     []RunDecision `json:"decisions"`
	Explanation   string        `json:"explanation"`
}

type ForgeLiveAttempt struct {
	SchemaVersion       string `json:"schema_version"`
	Status              string `json:"status"`
	Source              string `json:"source"`
	PacketSchemaVersion string `json:"packet_schema_version,omitempty"`
	PacketStatus        string `json:"packet_status,omitempty"`
	Explanation         string `json:"explanation"`
}

type ControlPlaneReadback struct {
	SchemaVersion        string `json:"schema_version"`
	Status               string `json:"status"`
	Source               string `json:"source"`
	ReceiptSchemaVersion string `json:"receipt_schema_version,omitempty"`
	Explanation          string `json:"explanation"`
}

type SignedSmokeResult struct {
	SchemaVersion        string `json:"schema_version"`
	Status               string `json:"status"`
	PulseEvent           string `json:"pulse_event"`
	ForgeLivePacket      string `json:"forge_live_packet"`
	ControlPlaneReadback string `json:"control_plane_readback"`
}

type SignedSmokeIngest struct {
	SchemaVersion        string `json:"schema_version"`
	Status               string `json:"status"`
	Result               string `json:"result"`
	ResultSHA256         string `json:"result_sha256"`
	PulseEvent           string `json:"pulse_event"`
	ForgeLivePacket      string `json:"forge_live_packet"`
	ControlPlaneReadback string `json:"control_plane_readback"`
	Explanation          string `json:"explanation"`
}

type SignedSmokeSummary struct {
	SchemaVersion string                       `json:"schema_version"`
	Status        string                       `json:"status"`
	PulseID       string                       `json:"pulse_id"`
	PulseStatus   string                       `json:"pulse_status"`
	ReleaseSafe   bool                         `json:"release_safe"`
	Evidence      []SignedSmokeSummaryEvidence `json:"evidence"`
	Explanation   string                       `json:"explanation"`
}

type SignedSmokeSummaryEvidence struct {
	Name          string `json:"name"`
	Status        string `json:"status"`
	SchemaVersion string `json:"schema_version"`
}

type SignedSmokePreflight struct {
	SchemaVersion string             `json:"schema_version"`
	Status        string             `json:"status"`
	Workspace     string             `json:"workspace"`
	Checks        []SignedSmokeCheck `json:"checks"`
	NextActions   []string           `json:"next_actions"`
}

type SignedSmokeCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type AO2LoopDecision struct {
	SchemaVersion string              `json:"schema_version"`
	EventLoop     AO2LoopDecisionBody `json:"event_loop"`
}

type AO2LoopDecisionBody struct {
	Action     string                `json:"action"`
	Reason     string                `json:"reason"`
	NextTaskID string                `json:"next_task_id"`
	Freshness  PulseFreshnessSummary `json:"freshness"`
}

type PulseIntakePreflight struct {
	SchemaVersion          string              `json:"schema_version"`
	Status                 string              `json:"status"`
	BlueprintStatus        string              `json:"blueprint_status"`
	AtlasStatus            string              `json:"atlas_status"`
	AtlasBlueprintStatus   string              `json:"atlas_blueprint_status,omitempty"`
	FirstFailingCheck      string              `json:"first_failing_check"`
	Checks                 []PulseIntakeCheck  `json:"checks"`
	BlockingNextActions    []string            `json:"blocking_next_actions"`
	MaintenanceSuggestions []string            `json:"maintenance_suggestions"`
	SourceArtifacts        []PulseIntakeSource `json:"source_artifacts"`
}

type PulseIntakeCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type PulseIntakeSource struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	SchemaVersion string `json:"schema_version"`
	Status        string `json:"status"`
	SHA256        string `json:"sha256"`
}

type PulsePRLifecycle struct {
	SchemaVersion     string `json:"schema_version"`
	CurrentSlice      string `json:"current_slice"`
	TargetRepo        string `json:"target_repo"`
	Branch            string `json:"branch"`
	PRNumber          int    `json:"pr_number"`
	PRURL             string `json:"pr_url"`
	PRState           string `json:"pr_state"`
	CheckState        string `json:"check_state"`
	MergeState        string `json:"merge_state"`
	CleanupState      string `json:"cleanup_state"`
	AllowedNextAction string `json:"allowed_next_action"`
	BlockerReason     string `json:"blocker_reason"`
}

type PulseOvernightStartGate struct {
	SchemaVersion          string                 `json:"schema_version"`
	Status                 string                 `json:"status"`
	AllowedNextAction      string                 `json:"allowed_next_action"`
	FirstFailingCheck      string                 `json:"first_failing_check"`
	BlockingNextActions    []string               `json:"blocking_next_actions"`
	MaintenanceSuggestions []string               `json:"maintenance_suggestions"`
	SourceHashes           []PulseStartGateSource `json:"source_hashes"`
}

type PulseEventLoopPolicy struct {
	SchemaVersion          string                       `json:"schema_version"`
	Status                 string                       `json:"status"`
	MutationClass          string                       `json:"mutation_class"`
	ProvenLiveClass        string                       `json:"proven_live_class"`
	ApprovedMutationClass  string                       `json:"approved_mutation_class"`
	AllowedNextAction      string                       `json:"allowed_next_action"`
	SafeToContinue         bool                         `json:"safe_to_continue"`
	SafeToRequest          bool                         `json:"safe_to_request"`
	SafeToExecute          bool                         `json:"safe_to_execute"`
	OperatorPromptRequired bool                         `json:"operator_prompt_required"`
	FirstFailingCheck      string                       `json:"first_failing_check"`
	RequiredChecks         []string                     `json:"required_checks"`
	SourceEvidence         []PulseEventLoopPolicySource `json:"source_evidence"`
	BlockingNextActions    []string                     `json:"blocking_next_actions"`
	AuthorityBoundary      string                       `json:"authority_boundary"`
	SchedulesWork          bool                         `json:"schedules_work"`
	ExecutesWork           bool                         `json:"executes_work"`
	ApprovesWork           bool                         `json:"approves_work"`
	MutatesRepositories    bool                         `json:"mutates_repositories"`
	CallsProviders         bool                         `json:"calls_providers"`
	OpensPR                bool                         `json:"opens_pr"`
	MergesPR               bool                         `json:"merges_pr"`
}

type PulseEventLoopPolicySource struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	SchemaVersion string `json:"schema_version"`
	Status        string `json:"status"`
	SHA256        string `json:"sha256"`
}

type PulseStartGateSource struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	SchemaVersion string `json:"schema_version"`
	SHA256        string `json:"sha256"`
}

type PulseRunnerStartDecision struct {
	SchemaVersion       string                 `json:"schema_version"`
	Status              string                 `json:"status"`
	StartGatePath       string                 `json:"start_gate_path"`
	AllowedNextAction   string                 `json:"allowed_next_action"`
	FirstFailingCheck   string                 `json:"first_failing_check"`
	BlockingNextActions []string               `json:"blocking_next_actions"`
	SourceDigests       []PulseStartGateSource `json:"source_digests"`
}

type ContractFixtureValidationResult struct {
	ValidFixtures   int
	InvalidFixtures int
}

type CapabilityMapping struct {
	Capability string `json:"capability"`
	Status     string `json:"status"`
	Foundry    string `json:"foundry"`
	Evidence   string `json:"evidence"`
}

type EvalDimension struct {
	Name     string `json:"name"`
	Score    int    `json:"score"`
	MaxScore int    `json:"max_score"`
	Status   string `json:"status"`
	Reason   string `json:"reason"`
	Evidence string `json:"evidence,omitempty"`
}

type EvidenceRef struct {
	Label  string `json:"label"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type GoalReadinessAudit struct {
	SchemaVersion string           `json:"schema_version"`
	Status        string           `json:"status"`
	Score         int              `json:"score"`
	MaxScore      int              `json:"max_score"`
	GoalID        string           `json:"goal_id"`
	Checks        []ReadinessCheck `json:"checks"`
	NextActions   []string         `json:"next_actions"`
}

type ForgeBrief struct {
	SchemaVersion     string           `json:"schema_version"`
	Objective         ForgeObjective   `json:"objective"`
	Constraints       ForgeConstraints `json:"constraints"`
	ExpectedWorkcells []ForgeWorkcell  `json:"expected_workcells"`
	ExpectedEvidence  []string         `json:"expected_evidence"`
}

type ForgeObjective struct {
	Text        string `json:"text"`
	Workspace   string `json:"workspace"`
	ReleaseMode bool   `json:"release_mode"`
}

type ForgeConstraints struct {
	LocalFirst                  bool `json:"local_first"`
	AllowNetwork                bool `json:"allow_network"`
	AllowReleaseMutation        bool `json:"allow_release_mutation"`
	RequireControlPlaneReadback bool `json:"require_control_plane_readback"`
}

type ForgeWorkcell struct {
	WorkcellID string   `json:"workcell_id"`
	Kind       string   `json:"kind"`
	Workspace  string   `json:"workspace,omitempty"`
	Executor   string   `json:"executor,omitempty"`
	Task       string   `json:"task,omitempty"`
	MaxRepairs int      `json:"max_repairs,omitempty"`
	DependsOn  []string `json:"depends_on"`
}

type NextAction struct {
	SchemaVersion string   `json:"schema_version"`
	Status        string   `json:"status"`
	TaskID        string   `json:"task_id"`
	DelegateTo    string   `json:"delegate_to"`
	ForgeBrief    string   `json:"forge_brief"`
	NextActions   []string `json:"next_actions"`
}

type HealthReaderConfig struct {
	RequireCleanWorktree bool     `json:"require_clean_worktree"`
	VerificationCommands []string `json:"verification_commands"`
	ReadinessFiles       []string `json:"readiness_files"`
	RequireTags          []string `json:"require_tags"`
	AllowNetworkRead     bool     `json:"allow_network_read"`
	GitHubActions        bool     `json:"github_actions"`
}

type RepoHealthReport struct {
	SchemaVersion string       `json:"schema_version"`
	RegistryID    string       `json:"registry_id"`
	Status        string       `json:"status"`
	Repos         []RepoHealth `json:"repos"`
	NextActions   []string     `json:"next_actions"`
}

type RepoHealth struct {
	RepoID        string            `json:"repo_id"`
	Workspace     string            `json:"workspace"`
	Status        string            `json:"status"`
	CurrentBranch string            `json:"current_branch,omitempty"`
	Checks        []RepoHealthCheck `json:"checks"`
	NextActions   []string          `json:"next_actions"`
}

type RepoHealthCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type RepoBoard struct {
	SchemaVersion string           `json:"schema_version"`
	RegistryID    string           `json:"registry_id"`
	Status        string           `json:"status"`
	Repos         []RepoBoardEntry `json:"repos"`
	NextActions   []string         `json:"next_actions"`
}

type RepoBoardEntry struct {
	RepoID         string   `json:"repo_id"`
	Name           string   `json:"name"`
	Role           string   `json:"role"`
	Tier           string   `json:"tier"`
	Workspace      string   `json:"workspace"`
	HealthStatus   string   `json:"health_status"`
	CurrentBranch  string   `json:"current_branch,omitempty"`
	Recommendation string   `json:"recommendation"`
	NextActions    []string `json:"next_actions"`
}

type ActiveStackReadinessLedger struct {
	SchemaVersion         string                           `json:"schema_version"`
	RegistryID            string                           `json:"registry_id"`
	GeneratedFromRegistry string                           `json:"generated_from_registry"`
	LastSweepDate         string                           `json:"last_sweep_date"`
	Status                string                           `json:"status"`
	Repositories          []ActiveStackReadinessRepository `json:"repositories"`
	ReleaseHandoff        ReleaseHandoff                   `json:"release_handoff"`
	NextActions           []string                         `json:"next_actions"`
}

type ReleaseHandoff struct {
	Status string               `json:"status"`
	Gates  []ReleaseHandoffGate `json:"gates"`
}

type ReleaseHandoffGate struct {
	Name                    string   `json:"name"`
	Status                  string   `json:"status"`
	RequiredBeforePromotion bool     `json:"required_before_promotion"`
	Evidence                []string `json:"evidence"`
}

type ActiveStackReadinessRepository struct {
	ID                   string       `json:"id"`
	Name                 string       `json:"name"`
	Role                 string       `json:"role"`
	Status               string       `json:"status"`
	CI                   *ReadinessCI `json:"ci,omitempty"`
	VerificationEvidence []string     `json:"verification_evidence"`
	Notes                []string     `json:"notes,omitempty"`
}

type ReadinessCI struct {
	Status string `json:"status"`
	RunID  string `json:"run_id,omitempty"`
	URL    string `json:"url,omitempty"`
}

type ActiveStackGithubRunsReport struct {
	SchemaVersion      string                            `json:"schema_version"`
	Status             string                            `json:"status"`
	Branch             string                            `json:"branch"`
	CurrentRepo        string                            `json:"current_repo,omitempty"`
	CurrentRepoSkipped bool                              `json:"current_repo_skipped,omitempty"`
	GeneratedAt        string                            `json:"generated_at"`
	Repositories       []ActiveStackGithubRunsRepository `json:"repositories"`
	NextActions        []string                          `json:"next_actions"`
}

type ActiveStackGithubRunsRepository struct {
	Repository string               `json:"repository"`
	LatestCI   ActiveStackGithubRun `json:"latest_ci"`
	LatestOps  ActiveStackGithubRun `json:"latest_ops"`
}

type ActiveStackGithubRun struct {
	Workflow    string `json:"workflow"`
	Status      string `json:"status"`
	Conclusion  string `json:"conclusion"`
	RunID       string `json:"run_id"`
	CreatedAt   string `json:"created_at,omitempty"`
	HeadSHA     string `json:"head_sha,omitempty"`
	DisplayName string `json:"display_title,omitempty"`
	URL         string `json:"url,omitempty"`
}

type ActiveStackProductionReadinessRollup struct {
	SchemaVersion        string                        `json:"schema_version"`
	Status               string                        `json:"status"`
	Ledger               string                        `json:"ledger"`
	GithubRunsReport     string                        `json:"github_runs_report"`
	ActiveRepositories   int                           `json:"active_repositories"`
	ReadyRepositories    int                           `json:"ready_repositories"`
	BlockedRepositories  int                           `json:"blocked_repositories"`
	CurrentRepo          string                        `json:"current_repo"`
	CurrentRepoSkipped   bool                          `json:"current_repo_skipped"`
	Repositories         []ActiveStackRollupRepository `json:"repositories"`
	ReleaseHandoff       []ActiveStackRollupGate       `json:"release_handoff"`
	Drift                []ActiveStackRollupDriftRow   `json:"drift"`
	ManualPromotionGates []string                      `json:"manual_promotion_gates"`
	Problems             []string                      `json:"problems,omitempty"`
	NextActions          []string                      `json:"next_actions"`
}

type ActiveStackRollupRepository struct {
	ID              string `json:"id"`
	Status          string `json:"status"`
	LatestCIRunID   string `json:"latest_ci_run_id,omitempty"`
	LatestCIStatus  string `json:"latest_ci_status,omitempty"`
	LatestOpsRunID  string `json:"latest_ops_run_id,omitempty"`
	LatestOpsStatus string `json:"latest_ops_status,omitempty"`
}

type ActiveStackRollupGate struct {
	Name                    string `json:"name"`
	Status                  string `json:"status"`
	RequiredBeforePromotion bool   `json:"required_before_promotion"`
	Classification          string `json:"classification"`
}

type ActiveStackRollupDriftRow struct {
	Repository string `json:"repository"`
	Workflow   string `json:"workflow"`
	RunID      string `json:"run_id,omitempty"`
	Action     string `json:"action"`
}

type FoundryRun struct {
	SchemaVersion string           `json:"schema_version"`
	RunID         string           `json:"run_id"`
	TaskID        string           `json:"task_id"`
	RegistryID    string           `json:"registry_id"`
	Status        string           `json:"status"`
	DelegatedTo   string           `json:"delegated_to"`
	ForgePacket   RunPacketRef     `json:"forge_packet"`
	Evidence      []RunEvidenceRef `json:"evidence"`
	Decisions     []RunDecision    `json:"decisions"`
	NextActions   []RunNextAction  `json:"next_actions"`
}

type RunPacketRef struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Status string `json:"status"`
}

type RunEvidenceRef struct {
	Label         string `json:"label"`
	Path          string `json:"path"`
	SHA256        string `json:"sha256"`
	SchemaVersion string `json:"schema_version"`
	Status        string `json:"status"`
}

type RunDecision struct {
	DecisionID  string `json:"decision_id"`
	Target      string `json:"target"`
	Decision    string `json:"decision"`
	Explanation string `json:"explanation"`
	Source      string `json:"source,omitempty"`
}

type RunNextAction struct {
	ActionID    string `json:"action_id"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

type ForgePacket struct {
	SchemaVersion string `json:"schema_version"`
	Status        string `json:"status"`
	Objective     struct {
		Text        string `json:"text"`
		Workspace   string `json:"workspace"`
		ReleaseMode bool   `json:"release_mode"`
	} `json:"objective"`
	FactoryPlan struct {
		PlanID        string `json:"plan_id"`
		WorkcellCount int    `json:"workcell_count"`
	} `json:"factory_plan"`
	PolicyDecisions []RunDecision `json:"policy_decisions"`
	Workcells       []struct {
		WorkcellID       string   `json:"workcell_id"`
		Kind             string   `json:"kind"`
		Workspace        string   `json:"workspace,omitempty"`
		Executor         string   `json:"executor,omitempty"`
		Peers            int      `json:"peers,omitempty"`
		MaxRepairs       int      `json:"max_repairs,omitempty"`
		Task             string   `json:"task,omitempty"`
		Status           string   `json:"status"`
		DependsOn        []string `json:"depends_on"`
		AO2Run           string   `json:"ao2_run,omitempty"`
		Summary          string   `json:"summary,omitempty"`
		RepairsAttempted int      `json:"repairs_attempted,omitempty"`
	} `json:"workcells"`
	Evidence      []RunEvidenceRef `json:"evidence"`
	TrustBoundary struct {
		LocalFirst               bool `json:"local_first"`
		MutatesReleases          bool `json:"mutates_releases"`
		StoresCredentials        bool `json:"stores_credentials"`
		ControlPlaneApprovesWork bool `json:"control_plane_approves_work"`
	} `json:"trust_boundary"`
	NextActions []RunNextAction `json:"next_actions"`
}

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "help" {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "status":
		return runStatus(args[1:], stdout, stderr)
	case "registry":
		return runRegistry(args[1:], stdout, stderr)
	case "task":
		return runTask(args[1:], stdout, stderr)
	case "next":
		return runNext(args[1:], stdout, stderr)
	case "readiness":
		return runReadiness(args[1:], stdout, stderr)
	case "goal":
		return runGoal(args[1:], stdout, stderr)
	case "run":
		return runRun(args[1:], stdout, stderr)
	case "atlas":
		return runAtlas(args[1:], stdout, stderr)
	case "class-gate":
		return runClassGate(args[1:], stdout, stderr)
	case "complex-repo":
		return runComplexRepo(args[1:], stdout, stderr)
	case "fully-unsupervised":
		return runFullyUnsupervised(args[1:], stdout, stderr)
	case "repo":
		return runRepo(args[1:], stdout, stderr)
	case "loop":
		return runLoop(args[1:], stdout, stderr)
	case "approval":
		return runApproval(args[1:], stdout, stderr)
	case "eval":
		return runEval(args[1:], stdout, stderr)
	case "rsi":
		return runRSI(args[1:], stdout, stderr)
	case "trace":
		return runTrace(args[1:], stdout, stderr)
	case "import":
		return runImport(args[1:], stdout, stderr)
	case "export":
		return runExport(args[1:], stdout, stderr)
	case "compare":
		return runCompare(args[1:], stdout, stderr)
	case "demo":
		return runDemo(args[1:], stdout, stderr)
	case "release":
		return runRelease(args[1:], stdout, stderr)
	case "competitive":
		return runCompetitive(args[1:], stdout, stderr)
	case "pulse":
		return runPulse(args[1:], stdout, stderr)
	case "contract":
		return runContract(args[1:], stdout, stderr)
	case "ao":
		return runAO(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "foundry: unknown command %q\n", args[0])
		printHelp(stderr)
		return 2
	}
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "AO Foundry operations CLI")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  foundry status --registry <path>")
	fmt.Fprintln(w, "  foundry registry validate --registry <path>")
	fmt.Fprintln(w, "  foundry task validate --task <path>")
	fmt.Fprintln(w, "  foundry next --registry <path> --task <path>")
	fmt.Fprintln(w, "  foundry readiness audit --registry <path> --task <path> [--out <path>]")
	fmt.Fprintln(w, "  foundry readiness snapshot --ledger <path> [--out <markdown>]")
	fmt.Fprintln(w, "  foundry readiness evidence-check --ledger <path> --github-runs-report <path> [--check-current-repo]")
	fmt.Fprintln(w, "  foundry readiness ledger-refresh-proposal --ledger <path> --github-runs-report <path> [--out <markdown>] [--apply --readme <path>] [--fail-on-non-current-update]")
	fmt.Fprintln(w, "  foundry readiness rollup --ledger <path> --github-runs-report <path> [--out <json>] [--markdown-out <markdown>]")
	fmt.Fprintln(w, "  foundry goal validate --goal-run <path>")
	fmt.Fprintln(w, "  foundry goal readiness --goal-run <path> --registry <path> --task <path> [--out <path>]")
	fmt.Fprintln(w, "  foundry run validate --run <path>")
	fmt.Fprintln(w, "  foundry run ingest --registry <path> --task <path> --packet <forge-packet.json> --out <foundry-run.json>")
	fmt.Fprintln(w, "  foundry run inspect --run <path> [--json]")
	fmt.Fprintln(w, "  foundry atlas import validate --import <atlas-foundry-import.json>")
	fmt.Fprintln(w, "  foundry atlas readback --import <atlas-foundry-import.json> --run-link <atlas-run-link.json> [--out <foundry-atlas-readback.json>]")
	fmt.Fprintln(w, "  foundry atlas status --registry <registry.json> --import <atlas-foundry-import.json> --run-link <atlas-run-link.json> [--out <foundry-atlas-status.json>] [--json]")
	fmt.Fprintln(w, "  foundry class-gate evaluate --atlas <path> --covenant <path> --sentinel <path> --promoter <path> --rollback <path> --command <path> --ci <path> --out <path>")
	fmt.Fprintln(w, "  foundry complex-repo node-gate evaluate --workgraph <path> --foundry-import <path> --candidate <path> --rollback <path> --out <path>")
	fmt.Fprintln(w, "  foundry fully-unsupervised readiness evaluate --blueprint-import <path> --workgraph <path> --foundry-import <path> --atlas-summary <path> --slice-manifest <path> --final-synthesis <path> --first-node-gate <path> --task-root <dir> --context-root <dir> --candidate-root <dir> --rollback-root <dir> --node-evidence-root <dir> --repair-root <dir> --context-repack-root <dir> --out <path>")
	fmt.Fprintln(w, "  foundry repo health --registry <path> [--repo <repo-id>] [--json]")
	fmt.Fprintln(w, "  foundry repo board --registry <path> [--json]")
	fmt.Fprintln(w, "  foundry loop preflight --goal-run <path> --registry <path> --task <path>")
	fmt.Fprintln(w, "  foundry loop lease acquire --goal-run <path> --lease <path>")
	fmt.Fprintln(w, "  foundry loop lease release --lease <path>")
	fmt.Fprintln(w, "  foundry loop next --goal-run <path> --registry <path> --task <path> --out <forge-brief.json>")
	fmt.Fprintln(w, "  foundry approval request --task <path> --out <approval-request.json>")
	fmt.Fprintln(w, "  foundry approval validate --decision <approval-decision.json> --task <path>")
	fmt.Fprintln(w, "  foundry eval run --run <foundry-run.json> --scorecard <scorecard.json> --out <eval-result.json>")
	fmt.Fprintln(w, "  foundry rsi improvement-gate --baseline <eval.json> --candidate <eval.json> --min-improvement <percent> --out <gate.json>")
	fmt.Fprintln(w, "  foundry trace inspect --trace <path> [--json]")
	fmt.Fprintln(w, "  foundry import ao2-sdd --plan <ao2.sdd-plan.json> --out <foundry-task.json>")
	fmt.Fprintln(w, "  foundry export forge-brief --task <foundry-task.json> --registry <path> --out <forge-brief.json>")
	fmt.Fprintln(w, "  foundry compare capabilities --out <capability-matrix.json>")
	fmt.Fprintln(w, "  foundry demo status --registry <path> --task <path> --run <path> [--json]")
	fmt.Fprintln(w, "  foundry demo script --out <markdown>")
	fmt.Fprintln(w, "  foundry release manifest --out <manifest.json>")
	fmt.Fprintln(w, "  foundry release dry-run --out <manifest.json>")
	fmt.Fprintln(w, "  foundry release handoff --candidate <path> --signed-smoke-summary <path> --promotion-out <path> --notes-out <markdown> --manifest-out <manifest.json>")
	fmt.Fprintln(w, "  foundry release candidate validate --ledger <path>")
	fmt.Fprintln(w, "  foundry release candidate active-stack-parity --ledger <path> --readiness-ledger <path>")
	fmt.Fprintln(w, "  foundry release candidate notes --ledger <path> --promotion <path> --out <markdown>")
	fmt.Fprintln(w, "  foundry release promotion validate --candidate <path> --signed-smoke-summary <path> --out <path>")
	fmt.Fprintln(w, "  foundry competitive audit --out <audit.json> [--json]")
	fmt.Fprintln(w, "  foundry contract fixtures validate")
	fmt.Fprintln(w, "  foundry pulse run [--start-gate <path>] [--registry <path>] [--task <path>] [--goal-run <path>] [--packet <path>] [--scorecard <path>] [--rsi-baseline <path>] [--rsi-min-improvement <percent>] [--signed-smoke-result <path>] --out <dir>")
	fmt.Fprintln(w, "  foundry pulse intake-preflight [--blueprint-authorization <path> | --blueprint-request <path>] [--requires-atlas --atlas-blueprint-import <path> --atlas-import <path> --atlas-status <path>] [--out <path>] [--json]")
	fmt.Fprintln(w, "  foundry pulse lifecycle inspect --state <pulse-pr-lifecycle.json> [--json]")
	fmt.Fprintln(w, "  foundry pulse overnight-start-gate --intake-preflight <path> --lifecycle <path> --out <path> [--start-implementation] [--json]")
	fmt.Fprintln(w, "  foundry pulse event-loop-policy --class-gate <path> --promotion-state <path> --ci <path> --repo-state <path> --evidence-freshness <path> --sentinel <path> --promoter <path> --rollback <path> --branch-cleanup <path> --scope <path> --out <path> [--json]")
	fmt.Fprintln(w, "  foundry pulse signed-smoke-script --out <script.sh>")
	fmt.Fprintln(w, "  foundry pulse signed-smoke-preflight --workspace <path> --out <preflight.json>")
	fmt.Fprintln(w, "  foundry pulse signed-smoke-cleanup")
	fmt.Fprintln(w, "  foundry pulse ingest-signed-smoke --result <signed-smoke-result.json> --out <ingest.json>")
	fmt.Fprintln(w, "  foundry pulse summarize-signed-smoke --pulse <pulse-event.json> --out <summary.json>")
	fmt.Fprintln(w, "  foundry pulse decision --action stop|continue --reason <text> --next-task-id <id> --out <decision.json>")
	fmt.Fprintln(w, "  foundry pulse derive-next --pulse <pulse-event.json> [--audit <audit.json>] --out <decision.json>")
	fmt.Fprintln(w, "  foundry pulse freshness --pulse <pulse-event.json>")
	fmt.Fprintln(w, "  foundry ao status|next|run|audit|demo")
}

func runStatus(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("status", stderr)
	registryPath := fs.String("registry", "", "registry path")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	registry, err := loadRegistry(*registryPath)
	if err != nil {
		fmt.Fprintf(stderr, "registry: %v\n", err)
		return 2
	}

	ready, blocked := readinessCounts(registry)
	fmt.Fprintf(stdout, "AO Foundry registry %q: %d repos, ready: %d, blocked: %d\n", registry.FoundryID, len(registry.Repos), ready, blocked)
	for _, repo := range registry.Repos {
		fmt.Fprintf(stdout, "- %s: role=%s delegates_to=%s\n", repo.ID, repo.Role, repo.DelegatesTo)
	}
	return 0
}

func runRegistry(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "validate" {
		fmt.Fprintln(stderr, "registry: expected subcommand validate")
		return 2
	}
	fs := newFlagSet("registry validate", stderr)
	registryPath := fs.String("registry", "", "registry path")
	if !parseFlags(fs, args[1:], stderr) {
		return 2
	}
	registry, err := loadRegistry(*registryPath)
	if err != nil {
		fmt.Fprintf(stderr, "registry: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "registry valid: %d repos\n", len(registry.Repos))
	return 0
}

func runTask(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "validate" {
		fmt.Fprintln(stderr, "task: expected subcommand validate")
		return 2
	}
	fs := newFlagSet("task validate", stderr)
	taskPath := fs.String("task", "", "task path")
	if !parseFlags(fs, args[1:], stderr) {
		return 2
	}
	task, err := loadTask(*taskPath)
	if err != nil {
		fmt.Fprintf(stderr, "task: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "task valid: %s\n", task.TaskID)
	return 0
}

func runNext(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("next", stderr)
	registryPath := fs.String("registry", "", "registry path")
	taskPath := fs.String("task", "", "task path")
	outPath := fs.String("out", "", "Forge brief output path")
	jsonOut := fs.Bool("json", false, "emit JSON next-action output")
	approvalPath := fs.String("approval-decision", "", "approval decision path for non-local side effects")
	tracePath := fs.String("trace", "", "trace output path")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	traceStatus := "failed"
	traceProblem := ""
	defer func() {
		writeTraceSpan(*tracePath, "foundry", "next", traceStatus, map[string]string{"registry": *registryPath, "task": *taskPath}, []string{*outPath}, traceProblem)
	}()
	registry, err := loadRegistry(*registryPath)
	if err != nil {
		traceProblem = err.Error()
		fmt.Fprintf(stderr, "registry: %v\n", err)
		return 2
	}
	task, err := loadTask(*taskPath)
	if err != nil {
		traceProblem = err.Error()
		fmt.Fprintf(stderr, "task: %v\n", err)
		return 2
	}
	if err := taskTargetsRegistered(task, registry); err != nil {
		traceProblem = err.Error()
		fmt.Fprintf(stderr, "task: %v\n", err)
		return 2
	}
	if err := targetReposReady(task, registry); err != nil {
		traceProblem = err.Error()
		fmt.Fprintf(stderr, "readiness: production readiness below 100: %v\n", err)
		return 1
	}
	if err := forgeDelegationReady(task); err != nil {
		traceProblem = err.Error()
		fmt.Fprintf(stderr, "readiness: production readiness below 100: %v\n", err)
		return 1
	}
	if err := approvalReady(*taskPath, task, *approvalPath); err != nil {
		traceProblem = err.Error()
		fmt.Fprintf(stderr, "approval: %v\n", err)
		return 1
	}

	if *outPath != "" {
		brief, err := buildForgeBrief(registry, task)
		if err != nil {
			traceProblem = err.Error()
			fmt.Fprintf(stderr, "next: %v\n", err)
			return 2
		}
		if *approvalPath != "" {
			brief.ExpectedEvidence = append(brief.ExpectedEvidence, "approval decision: "+portableEvidencePath(*approvalPath))
		}
		if err := writeJSONFile(*outPath, brief); err != nil {
			traceProblem = err.Error()
			fmt.Fprintf(stderr, "next: write Forge brief: %v\n", err)
			return 2
		}
	}

	if *jsonOut {
		action := NextAction{
			SchemaVersion: "ao.foundry.next-action.v0.1",
			Status:        "ready",
			TaskID:        task.TaskID,
			DelegateTo:    "ao-forge",
			ForgeBrief:    *outPath,
			NextActions:   []string{},
		}
		if err := writeJSON(stdout, action); err != nil {
			traceProblem = err.Error()
			fmt.Fprintf(stderr, "next: marshal action: %v\n", err)
			return 2
		}
		traceStatus = "passed"
		return 0
	}
	fmt.Fprintf(stdout, "next task: %s\n", task.TaskID)
	fmt.Fprintln(stdout, "action: delegate to ao-forge for governed implementation through AO Forge")
	if *outPath != "" {
		fmt.Fprintf(stdout, "forge_brief=%s\n", *outPath)
	}
	if len(task.Verification) > 0 {
		fmt.Fprintf(stdout, "verification: %s\n", strings.Join(task.Verification, "; "))
	}
	traceStatus = "passed"
	return 0
}

func runReadiness(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "readiness: expected subcommand audit, snapshot, evidence-check, ledger-refresh-proposal, or rollup")
		return 2
	}
	switch args[0] {
	case "audit":
		return runReadinessAudit(args[1:], stdout, stderr)
	case "snapshot":
		return runReadinessSnapshot(args[1:], stdout, stderr)
	case "evidence-check":
		return runReadinessEvidenceCheck(args[1:], stdout, stderr)
	case "ledger-refresh-proposal":
		return runReadinessLedgerRefreshProposal(args[1:], stdout, stderr)
	case "rollup":
		return runReadinessRollup(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "readiness: unknown subcommand %q\n", args[0])
		return 2
	}
}

func runReadinessAudit(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("readiness audit", stderr)
	registryPath := fs.String("registry", "", "registry path")
	taskPath := fs.String("task", "", "task path")
	outPath := fs.String("out", "", "audit output path")
	tracePath := fs.String("trace", "", "trace output path")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	audit, err := buildReadinessAudit(*registryPath, *taskPath)
	if err != nil {
		writeTraceSpan(*tracePath, "foundry", "readiness.audit", "failed", map[string]string{"registry": *registryPath, "task": *taskPath}, nil, err.Error())
		fmt.Fprintf(stderr, "readiness: %v\n", err)
		return 2
	}
	if *outPath != "" {
		if err := writeJSONFile(*outPath, audit); err != nil {
			fmt.Fprintf(stderr, "readiness: write audit: %v\n", err)
			return 2
		}
	}
	fmt.Fprintf(stdout, "production readiness: %d/%d status=%s\n", audit.Score, audit.MaxScore, audit.Status)
	if audit.Score != audit.MaxScore {
		writeTraceSpan(*tracePath, "foundry", "readiness.audit", "failed", map[string]string{"registry": *registryPath, "task": *taskPath}, nil, "production readiness below 100")
		fmt.Fprintln(stderr, "readiness: production readiness below 100")
		return 1
	}
	writeTraceSpan(*tracePath, "foundry", "readiness.audit", "passed", map[string]string{"registry": *registryPath, "task": *taskPath}, []string{*registryPath, *taskPath}, "")
	return 0
}

func runReadinessSnapshot(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("readiness snapshot", stderr)
	ledgerPath := fs.String("ledger", "", "active stack readiness ledger path")
	outPath := fs.String("out", "", "markdown output path")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	ledger, err := loadActiveStackReadinessLedger(*ledgerPath)
	if err != nil {
		fmt.Fprintf(stderr, "readiness: %v\n", err)
		return 2
	}
	snapshot := renderActiveStackReadinessSnapshot(*ledgerPath, ledger)
	if *outPath != "" {
		if err := writeTextFile(*outPath, snapshot); err != nil {
			fmt.Fprintf(stderr, "readiness: write snapshot: %v\n", err)
			return 2
		}
	}
	if _, err := io.WriteString(stdout, snapshot); err != nil {
		fmt.Fprintf(stderr, "readiness: write output: %v\n", err)
		return 2
	}
	return 0
}

func runReadinessEvidenceCheck(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("readiness evidence-check", stderr)
	ledgerPath := fs.String("ledger", "", "active stack readiness ledger path")
	reportPath := fs.String("github-runs-report", "", "active stack GitHub runs report path")
	currentRepo := fs.String("current-repo", "ao-foundry", "current repository id")
	checkCurrentRepo := fs.Bool("check-current-repo", false, "require current repository evidence to match the latest report run IDs")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	ledger, err := loadActiveStackReadinessLedger(*ledgerPath)
	if err != nil {
		fmt.Fprintf(stderr, "readiness evidence-check: %v\n", err)
		return 2
	}
	report, err := loadActiveStackGithubRunsReport(*reportPath)
	if err != nil {
		fmt.Fprintf(stderr, "readiness evidence-check: %v\n", err)
		return 2
	}
	result := checkActiveStackGithubRunEvidence(ledger, report, *currentRepo, *checkCurrentRepo)
	if len(result.Problems) > 0 {
		for _, problem := range result.Problems {
			fmt.Fprintf(stderr, "readiness evidence-check: %s\n", problem)
		}
		return 1
	}
	fmt.Fprintln(stdout, "readiness_evidence=ready")
	fmt.Fprintf(stdout, "repositories_checked=%d\n", result.Checked)
	if result.SkippedCurrentRepo {
		fmt.Fprintf(stdout, "current_repo_skipped=%s\n", *currentRepo)
	}
	return 0
}

func runReadinessLedgerRefreshProposal(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("readiness ledger-refresh-proposal", stderr)
	ledgerPath := fs.String("ledger", "", "active stack readiness ledger path")
	reportPath := fs.String("github-runs-report", "", "active stack GitHub runs report path")
	outPath := fs.String("out", "", "ledger refresh proposal markdown output path")
	readmePath := fs.String("readme", "", "README path to refresh when --apply is set")
	apply := fs.Bool("apply", false, "apply report run IDs to the ledger and README snapshot")
	failOnNonCurrentUpdate := fs.Bool("fail-on-non-current-update", false, "fail when proposal has update rows outside the current repository")
	currentRepo := fs.String("current-repo", "ao-foundry", "current repository id")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	if strings.TrimSpace(*outPath) == "" && !*apply && !*failOnNonCurrentUpdate {
		fmt.Fprintln(stderr, "readiness ledger-refresh-proposal: missing --out")
		return 2
	}
	if *apply && strings.TrimSpace(*readmePath) == "" {
		fmt.Fprintln(stderr, "readiness ledger-refresh-proposal: missing --readme for --apply")
		return 2
	}
	ledger, err := loadActiveStackReadinessLedger(*ledgerPath)
	if err != nil {
		fmt.Fprintf(stderr, "readiness ledger-refresh-proposal: %v\n", err)
		return 2
	}
	report, err := loadActiveStackGithubRunsReport(*reportPath)
	if err != nil {
		fmt.Fprintf(stderr, "readiness ledger-refresh-proposal: %v\n", err)
		return 2
	}
	rows := activeStackLedgerRefreshRows(ledger, report)
	rows = suppressCurrentRepoRefreshLoopRows(rows, *currentRepo)
	rows = suppressCurrentRepoMutableEvidenceRows(rows, *currentRepo)
	if *failOnNonCurrentUpdate {
		if problems := nonCurrentUpdateProblems(rows, *currentRepo); len(problems) > 0 {
			for _, problem := range problems {
				fmt.Fprintf(stderr, "readiness ledger-refresh-proposal: %s\n", problem)
			}
			return 1
		}
	}
	if *apply {
		updated, changes := applyActiveStackLedgerRefresh(ledger, report, rows)
		if err := writeJSONFile(*ledgerPath, updated); err != nil {
			fmt.Fprintf(stderr, "readiness ledger-refresh-proposal: write ledger: %v\n", err)
			return 2
		}
		if err := refreshReadmeActiveStackSnapshot(*readmePath, *ledgerPath, updated); err != nil {
			fmt.Fprintf(stderr, "readiness ledger-refresh-proposal: refresh README: %v\n", err)
			return 2
		}
		fmt.Fprintln(stdout, "ledger_refresh_apply=ready")
		fmt.Fprintf(stdout, "changes=%d\n", len(changes))
	}
	if strings.TrimSpace(*outPath) != "" {
		proposal := renderActiveStackLedgerRefreshProposal(*ledgerPath, *reportPath, rows)
		if err := writeTextFile(*outPath, proposal); err != nil {
			fmt.Fprintf(stderr, "readiness ledger-refresh-proposal: write proposal: %v\n", err)
			return 2
		}
		fmt.Fprintf(stdout, "ledger_refresh_proposal=%s\n", *outPath)
	}
	return 0
}

func runReadinessRollup(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("readiness rollup", stderr)
	ledgerPath := fs.String("ledger", "", "active stack readiness ledger path")
	reportPath := fs.String("github-runs-report", "", "active stack GitHub runs report path")
	outPath := fs.String("out", "", "production readiness rollup JSON output path")
	markdownOutPath := fs.String("markdown-out", "", "production readiness rollup markdown output path")
	currentRepo := fs.String("current-repo", "ao-foundry", "current repository id")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	rollup, err := buildActiveStackProductionReadinessRollup(*ledgerPath, *reportPath, *currentRepo)
	if err != nil {
		fmt.Fprintf(stderr, "readiness rollup: %v\n", err)
		return 2
	}
	if strings.TrimSpace(*outPath) != "" {
		if err := writeJSONFile(*outPath, rollup); err != nil {
			fmt.Fprintf(stderr, "readiness rollup: write rollup: %v\n", err)
			return 2
		}
		fmt.Fprintf(stdout, "readiness_rollup=%s\n", *outPath)
	} else {
		data, err := json.MarshalIndent(rollup, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "readiness rollup: marshal rollup: %v\n", err)
			return 2
		}
		if _, err := stdout.Write(append(data, '\n')); err != nil {
			fmt.Fprintf(stderr, "readiness rollup: write output: %v\n", err)
			return 2
		}
	}
	if strings.TrimSpace(*markdownOutPath) != "" {
		if err := writeTextFile(*markdownOutPath, renderActiveStackProductionReadinessRollupMarkdown(rollup)); err != nil {
			fmt.Fprintf(stderr, "readiness rollup: write markdown rollup: %v\n", err)
			return 2
		}
		fmt.Fprintf(stdout, "readiness_rollup_markdown=%s\n", *markdownOutPath)
	}
	fmt.Fprintf(stdout, "status=%s\n", rollup.Status)
	if rollup.Status != "ready" {
		for _, problem := range rollup.Problems {
			fmt.Fprintf(stderr, "readiness rollup: %s\n", problem)
		}
		return 1
	}
	return 0
}

func runGoal(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "goal: expected subcommand validate or readiness")
		return 2
	}
	switch args[0] {
	case "validate":
		return runGoalValidate(args[1:], stdout, stderr)
	case "readiness":
		return runGoalReadiness(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "goal: unknown subcommand %q\n", args[0])
		return 2
	}
}

func runGoalValidate(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("goal validate", stderr)
	goalPath := fs.String("goal-run", "", "goal-run path")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	goal, err := loadGoalRun(*goalPath)
	if err != nil {
		fmt.Fprintf(stderr, "goal: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "goal valid: %s\n", goal.GoalID)
	return 0
}

func runGoalReadiness(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("goal readiness", stderr)
	goalPath := fs.String("goal-run", "", "goal-run path")
	registryPath := fs.String("registry", "", "registry path")
	taskPath := fs.String("task", "", "task path")
	outPath := fs.String("out", "", "audit output path")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	audit, err := buildGoalReadinessAudit(*goalPath, *registryPath, *taskPath)
	if err != nil {
		fmt.Fprintf(stderr, "goal: %v\n", err)
		return 2
	}
	if *outPath != "" {
		data, err := json.MarshalIndent(audit, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "goal: marshal readiness audit: %v\n", err)
			return 2
		}
		data = append(data, '\n')
		if err := os.WriteFile(*outPath, data, 0o644); err != nil {
			fmt.Fprintf(stderr, "goal: write readiness audit: %v\n", err)
			return 2
		}
	}
	fmt.Fprintf(stdout, "goal readiness: %d/%d status=%s\n", audit.Score, audit.MaxScore, audit.Status)
	if audit.Score != audit.MaxScore {
		fmt.Fprintln(stderr, "goal: goal readiness below 100")
		return 1
	}
	return 0
}

func runRun(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "run: expected subcommand validate, ingest, or inspect")
		return 2
	}
	switch args[0] {
	case "validate":
		return runRunValidate(args[1:], stdout, stderr)
	case "ingest":
		return runRunIngest(args[1:], stdout, stderr)
	case "inspect":
		return runRunInspect(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "run: unknown subcommand %q\n", args[0])
		return 2
	}
}

func runRunValidate(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("run validate", stderr)
	runPath := fs.String("run", "", "run path")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	run, err := loadFoundryRun(*runPath)
	if err != nil {
		fmt.Fprintf(stderr, "run: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "run valid: %s status=%s\n", run.TaskID, run.Status)
	return 0
}

func runRunIngest(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("run ingest", stderr)
	registryPath := fs.String("registry", "", "registry path")
	taskPath := fs.String("task", "", "task path")
	packetPath := fs.String("packet", "", "Forge packet path")
	outPath := fs.String("out", "", "Foundry run output path")
	tracePath := fs.String("trace", "", "trace output path")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	traceStatus := "failed"
	traceProblem := ""
	defer func() {
		writeTraceSpan(*tracePath, "foundry", "run.ingest", traceStatus, map[string]string{"registry": *registryPath, "task": *taskPath, "packet": *packetPath}, []string{*outPath, *packetPath}, traceProblem)
	}()
	if *outPath == "" {
		traceProblem = "missing --out"
		fmt.Fprintln(stderr, "run: missing --out")
		return 2
	}
	run, err := buildFoundryRun(*registryPath, *taskPath, *packetPath)
	if err != nil {
		traceProblem = err.Error()
		fmt.Fprintf(stderr, "run: %v\n", err)
		return 2
	}
	if err := writeJSONFile(*outPath, run); err != nil {
		traceProblem = err.Error()
		fmt.Fprintf(stderr, "run: write run record: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "run_record=%s\n", *outPath)
	fmt.Fprintf(stdout, "status=%s\n", run.Status)
	traceStatus = "passed"
	return 0
}

func runRunInspect(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("run inspect", stderr)
	runPath := fs.String("run", "", "run path")
	jsonOut := fs.Bool("json", false, "emit JSON run record")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	run, err := loadFoundryRun(*runPath)
	if err != nil {
		fmt.Fprintf(stderr, "run: %v\n", err)
		return 2
	}
	if *jsonOut {
		if err := writeJSON(stdout, run); err != nil {
			fmt.Fprintf(stderr, "run: marshal run record: %v\n", err)
			return 2
		}
		return 0
	}
	fmt.Fprintf(stdout, "run_id=%s\n", run.RunID)
	fmt.Fprintf(stdout, "status=%s\n", run.Status)
	fmt.Fprintf(stdout, "task_id=%s\n", run.TaskID)
	fmt.Fprintf(stdout, "delegated_to=%s\n", run.DelegatedTo)
	fmt.Fprintf(stdout, "packet_sha256=%s\n", run.ForgePacket.SHA256)
	fmt.Fprintf(stdout, "evidence_count=%d\n", len(run.Evidence))
	return 0
}

func runAtlas(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "atlas: expected subcommand import validate, readback, or status")
		return 2
	}
	if len(args) >= 2 && args[0] == "import" && args[1] == "validate" {
		return runAtlasImportValidate(args[2:], stdout, stderr)
	}
	if args[0] == "readback" {
		return runAtlasReadback(args[1:], stdout, stderr)
	}
	if args[0] == "status" {
		return runAtlasStatus(args[1:], stdout, stderr)
	}
	fmt.Fprintln(stderr, "atlas: expected subcommand import validate, readback, or status")
	return 2
}

func runAtlasImportValidate(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("atlas import validate", stderr)
	importPath := fs.String("import", "", "Atlas foundry-import packet path")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	artifact, err := loadAtlasFoundryImport(*importPath)
	if err != nil {
		fmt.Fprintf(stderr, "atlas import validate: %v\n", err)
		return 2
	}
	fmt.Fprintln(stdout, "atlas import valid")
	fmt.Fprintf(stdout, "id=%s\n", artifact.ID)
	fmt.Fprintf(stdout, "workgraph=%s\n", artifact.WorkgraphID)
	fmt.Fprintf(stdout, "target_instance=%s\n", artifact.TargetInstance)
	fmt.Fprintf(stdout, "source_artifacts=%d\n", len(artifact.SourceArtifacts))
	fmt.Fprintf(stdout, "tasks=%d\n", len(artifact.Tasks))
	fmt.Fprintf(stdout, "schedules_work=%t\n", artifact.SchedulesWork)
	fmt.Fprintf(stdout, "executes_work=%t\n", artifact.ExecutesWork)
	fmt.Fprintf(stdout, "approves_work=%t\n", artifact.ApprovesWork)
	return 0
}

func runAtlasReadback(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("atlas readback", stderr)
	importPath := fs.String("import", "", "Atlas foundry-import packet path")
	runLinkPath := fs.String("run-link", "", "Atlas run-link path")
	outPath := fs.String("out", "", "output path")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	report, err := buildAtlasReadback(*importPath, *runLinkPath)
	if err != nil {
		fmt.Fprintf(stderr, "atlas readback: %v\n", err)
		return 2
	}
	if strings.TrimSpace(*outPath) == "" {
		if err := writeJSON(stdout, report); err != nil {
			fmt.Fprintf(stderr, "atlas readback: marshal report: %v\n", err)
			return 2
		}
		return 0
	}
	for _, inputPath := range []string{*importPath, *runLinkPath} {
		if sameCleanPath(*outPath, inputPath) {
			fmt.Fprintln(stderr, "atlas readback: --out must not overwrite input artifacts")
			return 2
		}
	}
	if err := writeJSONFile(*outPath, report); err != nil {
		fmt.Fprintf(stderr, "atlas readback: write report: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "atlas_readback=%s\n", *outPath)
	fmt.Fprintf(stdout, "status=%s\n", report.Status)
	fmt.Fprintf(stdout, "mode=%s\n", report.Mode)
	fmt.Fprintf(stdout, "task_id=%s\n", report.TaskID)
	fmt.Fprintf(stdout, "schedules_work=%t\n", report.SchedulesWork)
	fmt.Fprintf(stdout, "executes_work=%t\n", report.ExecutesWork)
	fmt.Fprintf(stdout, "approves_work=%t\n", report.ApprovesWork)
	return 0
}

func runAtlasStatus(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("atlas status", stderr)
	registryPath := fs.String("registry", "", "Foundry registry path")
	importPath := fs.String("import", "", "Atlas foundry-import packet path")
	runLinkPath := fs.String("run-link", "", "Atlas run-link path")
	outPath := fs.String("out", "", "output path")
	jsonOut := fs.Bool("json", false, "emit JSON status report")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	report, err := buildAtlasStatus(*registryPath, *importPath, *runLinkPath)
	if err != nil {
		fmt.Fprintf(stderr, "atlas status: %v\n", err)
		return 2
	}
	if strings.TrimSpace(*outPath) != "" {
		for _, inputPath := range []string{*registryPath, *importPath, *runLinkPath} {
			if sameCleanPath(*outPath, inputPath) {
				fmt.Fprintln(stderr, "atlas status: --out must not overwrite input artifacts")
				return 2
			}
		}
		if err := writeJSONFile(*outPath, report); err != nil {
			fmt.Fprintf(stderr, "atlas status: write report: %v\n", err)
			return 2
		}
		fmt.Fprintf(stdout, "atlas_status=%s\n", *outPath)
		return 0
	}
	if *jsonOut {
		if err := writeJSON(stdout, report); err != nil {
			fmt.Fprintf(stderr, "atlas status: marshal report: %v\n", err)
			return 2
		}
		return 0
	}
	fmt.Fprintf(stdout, "atlas status: %s\n", report.Status)
	fmt.Fprintf(stdout, "mode=%s\n", report.Mode)
	fmt.Fprintf(stdout, "registry=%s\n", report.RegistryID)
	fmt.Fprintf(stdout, "import=%s\n", report.ImportID)
	fmt.Fprintf(stdout, "readback=%s\n", atlasReadbackSchema)
	fmt.Fprintf(stdout, "task_id=%s\n", report.TaskID)
	fmt.Fprintf(stdout, "schedules_work=%t\n", report.SchedulesWork)
	fmt.Fprintf(stdout, "executes_work=%t\n", report.ExecutesWork)
	fmt.Fprintf(stdout, "approves_work=%t\n", report.ApprovesWork)
	return 0
}

func runClassGate(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: foundry class-gate evaluate --atlas <path> --covenant <path> --sentinel <path> --promoter <path> --rollback <path> --command <path> --ci <path> --out <path>")
		return 2
	}
	switch args[0] {
	case "evaluate":
		return runClassGateEvaluate(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "foundry class-gate: unknown command %q\n", args[0])
		return 2
	}
}

func runComplexRepo(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		fmt.Fprintln(stderr, "usage: foundry complex-repo <node-gate evaluate|node execute|closure backfill|promotion-rollup evaluate> ...")
		return 2
	}
	switch {
	case args[0] == "node-gate" && args[1] == "evaluate":
		return runComplexRepoNodeGateEvaluate(args[2:], stdout, stderr)
	case args[0] == "node" && args[1] == "execute":
		return runComplexRepoNodeExecute(args[2:], stdout, stderr)
	case args[0] == "closure" && args[1] == "backfill":
		return runComplexRepoClosureBackfill(args[2:], stdout, stderr)
	case args[0] == "promotion-rollup" && args[1] == "evaluate":
		return runComplexRepoPromotionRollupEvaluate(args[2:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "foundry complex-repo: unknown command %q\n", strings.Join(args, " "))
		return 2
	}
}

func runFullyUnsupervised(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		fmt.Fprintln(stderr, "usage: foundry fully-unsupervised <readiness evaluate> ...")
		return 2
	}
	switch {
	case args[0] == "readiness" && args[1] == "evaluate":
		return runFullyUnsupervisedReadinessEvaluate(args[2:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "foundry fully-unsupervised: unknown command %q\n", strings.Join(args, " "))
		return 2
	}
}

type repeatedStringFlag []string

func (f *repeatedStringFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *repeatedStringFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("empty value")
	}
	*f = append(*f, value)
	return nil
}

type fullyUnsupervisedReadinessPaths struct {
	BlueprintImport  string
	Workgraph        string
	FoundryImport    string
	AtlasSummary     string
	SliceManifest    string
	FinalSynthesis   string
	FirstNodeGate    string
	TaskRoot         string
	ContextRoot      string
	CandidateRoot    string
	RollbackRoot     string
	NodeEvidenceRoot string
	RepairRoot       string
	RepackRoot       string
}

func runFullyUnsupervisedReadinessEvaluate(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("fully-unsupervised readiness evaluate", stderr)
	blueprintImportPath := fs.String("blueprint-import", "", "Atlas Blueprint import")
	workgraphPath := fs.String("workgraph", "", "Atlas fully unsupervised readiness workgraph")
	foundryImportPath := fs.String("foundry-import", "", "Atlas Foundry import")
	atlasSummaryPath := fs.String("atlas-summary", "", "Atlas first-phase summary")
	sliceManifestPath := fs.String("slice-manifest", "", "Atlas SDD slice completion manifest")
	finalSynthesisPath := fs.String("final-synthesis", "", "Atlas final evidence synthesis")
	firstNodeGatePath := fs.String("first-node-gate", "", "Foundry first-node gate output")
	taskRoot := fs.String("task-root", "", "Atlas task record directory")
	contextRoot := fs.String("context-root", "", "Atlas context pack directory")
	candidateRoot := fs.String("candidate-root", "", "Atlas candidate record directory")
	rollbackRoot := fs.String("rollback-root", "", "Atlas rollback record directory")
	nodeEvidenceRoot := fs.String("node-evidence-root", "", "Atlas node evidence directory")
	repairRoot := fs.String("repair-root", "", "Atlas repair plan directory")
	repackRoot := fs.String("context-repack-root", "", "Atlas context repack plan directory")
	outPath := fs.String("out", "", "readiness rollup output path")
	jsonOut := fs.Bool("json", false, "also write JSON to stdout")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	required := map[string]string{
		"--blueprint-import":    *blueprintImportPath,
		"--workgraph":           *workgraphPath,
		"--foundry-import":      *foundryImportPath,
		"--atlas-summary":       *atlasSummaryPath,
		"--slice-manifest":      *sliceManifestPath,
		"--final-synthesis":     *finalSynthesisPath,
		"--first-node-gate":     *firstNodeGatePath,
		"--task-root":           *taskRoot,
		"--context-root":        *contextRoot,
		"--candidate-root":      *candidateRoot,
		"--rollback-root":       *rollbackRoot,
		"--node-evidence-root":  *nodeEvidenceRoot,
		"--repair-root":         *repairRoot,
		"--context-repack-root": *repackRoot,
		"--out":                 *outPath,
	}
	missing := []string{}
	for flagName, value := range required {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, flagName)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		fmt.Fprintf(stderr, "%s are required\n", strings.Join(missing, ", "))
		return 2
	}
	rollup, err := buildFullyUnsupervisedReadinessRollup(fullyUnsupervisedReadinessPaths{
		BlueprintImport:  *blueprintImportPath,
		Workgraph:        *workgraphPath,
		FoundryImport:    *foundryImportPath,
		AtlasSummary:     *atlasSummaryPath,
		SliceManifest:    *sliceManifestPath,
		FinalSynthesis:   *finalSynthesisPath,
		FirstNodeGate:    *firstNodeGatePath,
		TaskRoot:         *taskRoot,
		ContextRoot:      *contextRoot,
		CandidateRoot:    *candidateRoot,
		RollbackRoot:     *rollbackRoot,
		NodeEvidenceRoot: *nodeEvidenceRoot,
		RepairRoot:       *repairRoot,
		RepackRoot:       *repackRoot,
	})
	if err != nil {
		fmt.Fprintf(stderr, "fully unsupervised readiness: %v\n", err)
		return 1
	}
	if err := writeJSONFile(*outPath, rollup); err != nil {
		fmt.Fprintf(stderr, "write fully unsupervised readiness rollup: %v\n", err)
		return 1
	}
	if *jsonOut {
		if err := writeJSON(stdout, rollup); err != nil {
			fmt.Fprintf(stderr, "write fully unsupervised readiness json: %v\n", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "fully_unsupervised_readiness_rollup=%s\n", *outPath)
	fmt.Fprintf(stdout, "status=%s\n", classGateString(rollup, "status"))
	fmt.Fprintf(stdout, "safe_to_execute=%t\n", classGateBool(rollup, "safe_to_execute"))
	if first := classGateString(rollup, "first_failing_check"); first != "" {
		fmt.Fprintf(stdout, "first_failing_check=%s\n", first)
	}
	return 0
}

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

func countWorkgraphNodesWithStatus(nodes []map[string]any, status string) int {
	count := 0
	for _, node := range nodes {
		if classGateString(node, "status") == status {
			count++
		}
	}
	return count
}

func runComplexRepoClosureBackfill(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("complex-repo closure backfill", stderr)
	missionPath := fs.String("mission", "", "Atlas mission continuation evidence")
	workgraphPath := fs.String("workgraph", "", "Atlas final complex_repo_mutation workgraph")
	runLinksRoot := fs.String("run-links-root", "", "root containing per-node run-link.json evidence")
	closureRoot := fs.String("closure-root", "", "output root for digest-bound per-node closure evidence")
	finalNodeGatePath := fs.String("final-node-gate", "", "final synthesis node gate evidence")
	var nodeGatePaths repeatedStringFlag
	fs.Var(&nodeGatePaths, "node-gate", "additional per-node gate evidence; may be repeated")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	if strings.TrimSpace(*missionPath) == "" ||
		strings.TrimSpace(*workgraphPath) == "" ||
		strings.TrimSpace(*runLinksRoot) == "" ||
		strings.TrimSpace(*closureRoot) == "" ||
		strings.TrimSpace(*finalNodeGatePath) == "" {
		fmt.Fprintln(stderr, "--mission, --workgraph, --run-links-root, --closure-root, and --final-node-gate are required")
		return 2
	}
	manifest, err := buildComplexRepoClosureEvidence(complexPromotionRollupPaths{
		Mission:       *missionPath,
		Workgraph:     *workgraphPath,
		RunLinksRoot:  *runLinksRoot,
		ClosureRoot:   *closureRoot,
		NodeGates:     nodeGatePaths,
		FinalNodeGate: *finalNodeGatePath,
	})
	if err != nil {
		fmt.Fprintf(stderr, "complex closure backfill: %v\n", err)
		return 1
	}
	manifestPath := filepath.Join(*closureRoot, "closure-manifest.json")
	if err := writeJSONFile(manifestPath, manifest); err != nil {
		fmt.Fprintf(stderr, "write complex closure manifest: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "complex_closure_root=%s\n", *closureRoot)
	fmt.Fprintf(stdout, "node_count=%v\n", manifest["node_count"])
	fmt.Fprintf(stdout, "evidence_item_count=%v\n", manifest["evidence_item_count"])
	return 0
}

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

func runComplexRepoPromotionRollupEvaluate(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("complex-repo promotion-rollup evaluate", stderr)
	missionPath := fs.String("mission", "", "Atlas mission continuation evidence")
	workgraphPath := fs.String("workgraph", "", "Atlas final complex_repo_mutation workgraph")
	runLinksRoot := fs.String("run-links-root", "", "root containing per-node run-link.json evidence")
	closureRoot := fs.String("closure-root", "", "root containing digest-bound per-node closure evidence")
	finalNodeGatePath := fs.String("final-node-gate", "", "final synthesis node gate evidence")
	outPath := fs.String("out", "", "promotion rollup output path")
	jsonOut := fs.Bool("json", false, "also write JSON to stdout")
	var nodeGatePaths repeatedStringFlag
	fs.Var(&nodeGatePaths, "node-gate", "additional per-node gate evidence; may be repeated")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	if strings.TrimSpace(*missionPath) == "" ||
		strings.TrimSpace(*workgraphPath) == "" ||
		strings.TrimSpace(*runLinksRoot) == "" ||
		strings.TrimSpace(*finalNodeGatePath) == "" ||
		strings.TrimSpace(*outPath) == "" {
		fmt.Fprintln(stderr, "--mission, --workgraph, --run-links-root, --final-node-gate, and --out are required")
		return 2
	}
	rollup, err := buildComplexRepoMutationPromotionRollup(complexPromotionRollupPaths{
		Mission:       *missionPath,
		Workgraph:     *workgraphPath,
		RunLinksRoot:  *runLinksRoot,
		ClosureRoot:   *closureRoot,
		NodeGates:     nodeGatePaths,
		FinalNodeGate: *finalNodeGatePath,
	})
	if err != nil {
		fmt.Fprintf(stderr, "complex promotion rollup: %v\n", err)
		return 1
	}
	if err := writeJSONFile(*outPath, rollup); err != nil {
		fmt.Fprintf(stderr, "write complex promotion rollup: %v\n", err)
		return 1
	}
	if *jsonOut {
		if err := writeJSON(stdout, rollup); err != nil {
			fmt.Fprintf(stderr, "write complex promotion rollup json: %v\n", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "complex_promotion_rollup=%s\n", *outPath)
	fmt.Fprintf(stdout, "status=%s\n", rollup.Status)
	fmt.Fprintf(stdout, "safe_to_promote=%t\n", rollup.SafeToPromote)
	if rollup.FirstFailingCheck != "" {
		fmt.Fprintf(stdout, "first_failing_check=%s\n", rollup.FirstFailingCheck)
	}
	return 0
}

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

func runComplexRepoNodeGateEvaluate(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("complex-repo node-gate evaluate", stderr)
	workgraphPath := fs.String("workgraph", "", "Atlas complex_repo_mutation workgraph")
	foundryImportPath := fs.String("foundry-import", "", "Atlas Foundry import for the selected complex node")
	candidatePath := fs.String("candidate", "", "Atlas candidate record for the selected complex node")
	rollbackPath := fs.String("rollback", "", "Atlas rollback record for the selected complex node")
	outPath := fs.String("out", "", "complex node gate output path")
	jsonOut := fs.Bool("json", false, "also write JSON to stdout")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	if strings.TrimSpace(*workgraphPath) == "" ||
		strings.TrimSpace(*foundryImportPath) == "" ||
		strings.TrimSpace(*candidatePath) == "" ||
		strings.TrimSpace(*rollbackPath) == "" ||
		strings.TrimSpace(*outPath) == "" {
		fmt.Fprintln(stderr, "--workgraph, --foundry-import, --candidate, --rollback, and --out are required")
		return 2
	}
	gate, err := buildComplexRepoMutationNodeGate(complexNodeGatePaths{
		Workgraph:     *workgraphPath,
		FoundryImport: *foundryImportPath,
		Candidate:     *candidatePath,
		Rollback:      *rollbackPath,
	})
	if err != nil {
		fmt.Fprintf(stderr, "complex node gate: %v\n", err)
		return 1
	}
	if err := writeJSONFile(*outPath, gate); err != nil {
		fmt.Fprintf(stderr, "write complex node gate: %v\n", err)
		return 1
	}
	if *jsonOut {
		if err := writeJSON(stdout, gate); err != nil {
			fmt.Fprintf(stderr, "write complex node gate json: %v\n", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "complex_node_gate=%s\n", *outPath)
	fmt.Fprintf(stdout, "status=%s\n", gate.Status)
	fmt.Fprintf(stdout, "node_id=%s\n", gate.NodeID)
	fmt.Fprintf(stdout, "safe_to_request=%t\n", gate.SafeToRequest)
	fmt.Fprintf(stdout, "safe_to_execute=%t\n", gate.SafeToExecute)
	if gate.FirstFailingCheck != "" {
		fmt.Fprintf(stdout, "first_failing_check=%s\n", gate.FirstFailingCheck)
	}
	return 0
}

func runComplexRepoNodeExecute(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("complex-repo node execute", stderr)
	nodeGatePath := fs.String("node-gate", "", "ready complex_repo_mutation node gate")
	nodeRecordOut := fs.String("node-record-out", "", "node record output path")
	runLinkOut := fs.String("run-link-out", "", "Atlas run-link output path")
	nodeClass := fs.String("node-class", "", "selected node class")
	scope := fs.String("scope", "", "exact node write scope")
	summary := fs.String("summary", "", "node execution summary")
	pr := fs.String("pr", "", "merged execution pull request")
	mergeCommit := fs.String("merge-commit", "", "merged execution commit")
	ci := fs.String("ci", "passed", "CI evidence readback")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	if strings.TrimSpace(*nodeGatePath) == "" ||
		strings.TrimSpace(*nodeRecordOut) == "" ||
		strings.TrimSpace(*runLinkOut) == "" ||
		strings.TrimSpace(*nodeClass) == "" ||
		strings.TrimSpace(*scope) == "" ||
		strings.TrimSpace(*summary) == "" {
		fmt.Fprintln(stderr, "--node-gate, --node-record-out, --run-link-out, --node-class, --scope, and --summary are required")
		return 2
	}
	if err := validateAtlasPublicString(*scope); err != nil {
		fmt.Fprintf(stderr, "complex node execute scope: %v\n", err)
		return 1
	}
	gate, err := loadComplexRepoMutationNodeGate(*nodeGatePath)
	if err != nil {
		fmt.Fprintf(stderr, "complex node execute: %v\n", err)
		return 1
	}
	if gate.Status != "ready" || !gate.SafeToRequest || !gate.SafeToExecute || !gate.LiveExecutionAuthority {
		fmt.Fprintln(stderr, "complex node execute requires ready node gate with safe_to_request=true and safe_to_execute=true")
		return 1
	}
	record := buildComplexNodeRecord(gate, *nodeClass, *scope, *summary)
	if err := writeJSONFile(*nodeRecordOut, record); err != nil {
		fmt.Fprintf(stderr, "write complex node record: %v\n", err)
		return 1
	}
	link := buildComplexNodeRunLink(gate, *scope, *nodeGatePath, *pr, *mergeCommit, *ci)
	if err := writeJSONFile(*runLinkOut, link); err != nil {
		fmt.Fprintf(stderr, "write complex node run-link: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "status=completed\nnode_id=%s\ntask_id=%s\nnode_record=%s\nrun_link=%s\n", gate.NodeID, gate.TaskID, *nodeRecordOut, *runLinkOut)
	return 0
}

type complexNodeGatePaths struct {
	Workgraph     string
	FoundryImport string
	Candidate     string
	Rollback      string
}

func runClassGateEvaluate(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("class-gate evaluate", stderr)
	atlasPath := fs.String("atlas", "", "Atlas mutation-class classification evidence")
	covenantPath := fs.String("covenant", "", "Covenant mutation-class authority ticket")
	sentinelPath := fs.String("sentinel", "", "Sentinel mutation-class hold verdict")
	promoterPath := fs.String("promoter", "", "Promoter mutation-class readiness evidence")
	rollbackPath := fs.String("rollback", "", "rollback rehearsal evidence")
	commandPath := fs.String("command", "", "AO Command authority-ladder readback")
	ciPath := fs.String("ci", "", "CI status evidence")
	testOnlySuccessPath := fs.String("test-only-success", "", "completed test_only live rehearsal evidence for low_risk_code dry-run readiness")
	multiRepoPlanPath := fs.String("multi-repo-plan", "", "multi-repo low-risk sequencing and rollback plan evidence")
	lowRiskCodeLiveSuccessPath := fs.String("low-risk-code-live-success", "", "completed low_risk_code live rehearsal success readback for multi_repo_low_risk readiness")
	complexNodeGatePath := fs.String("complex-node-gate", "", "complex_repo_mutation node gate readback evidence")
	outPath := fs.String("out", "", "class gate output path")
	jsonOut := fs.Bool("json", false, "also write JSON to stdout")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	if strings.TrimSpace(*atlasPath) == "" ||
		strings.TrimSpace(*covenantPath) == "" ||
		strings.TrimSpace(*sentinelPath) == "" ||
		strings.TrimSpace(*promoterPath) == "" ||
		strings.TrimSpace(*rollbackPath) == "" ||
		strings.TrimSpace(*commandPath) == "" ||
		strings.TrimSpace(*ciPath) == "" ||
		strings.TrimSpace(*outPath) == "" {
		fmt.Fprintln(stderr, "--atlas, --covenant, --sentinel, --promoter, --rollback, --command, --ci, and --out are required")
		return 2
	}
	gate, err := evaluateMutationClassGate(classGateEvidencePaths{
		Atlas:                  *atlasPath,
		Covenant:               *covenantPath,
		Sentinel:               *sentinelPath,
		Promoter:               *promoterPath,
		Rollback:               *rollbackPath,
		Command:                *commandPath,
		CI:                     *ciPath,
		TestOnlySuccess:        *testOnlySuccessPath,
		MultiRepoPlan:          *multiRepoPlanPath,
		LowRiskCodeLiveSuccess: *lowRiskCodeLiveSuccessPath,
		ComplexNodeGate:        *complexNodeGatePath,
	})
	if err != nil {
		fmt.Fprintf(stderr, "class gate: %v\n", err)
		return 1
	}
	if err := writeJSONFile(*outPath, gate); err != nil {
		fmt.Fprintf(stderr, "write class gate: %v\n", err)
		return 1
	}
	if *jsonOut {
		if err := writeJSON(stdout, gate); err != nil {
			fmt.Fprintf(stderr, "write class gate json: %v\n", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "class_gate=%s\n", *outPath)
	fmt.Fprintf(stdout, "status=%s\n", gate.Status)
	fmt.Fprintf(stdout, "mutation_class=%s\n", gate.MutationClass)
	fmt.Fprintf(stdout, "safe_to_request=%t\n", gate.SafeToRequest)
	fmt.Fprintf(stdout, "safe_to_execute=%t\n", gate.SafeToExecute)
	return 0
}

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
	return map[string]any{
		"schema":         "ao.atlas.complex-repo-mutation-node-record.v0.1",
		"node_id":        gate.NodeID,
		"task_id":        gate.TaskID,
		"node_class":     strings.TrimSpace(nodeClass),
		"status":         "completed",
		"mutation_class": gate.MutationClass,
		"scope":          cleanScope,
		"summary":        strings.TrimSpace(summary),
		"accepted_evidence": []string{
			"live_rehearsal:multi_repo_low_risk",
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
			"complex_repo_mutation_live_proven":   "false",
			"fully_unsupervised_complex_mutation": "denied",
			"rsi":                                 "denied",
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

func classGateString(document map[string]any, key string) string {
	value, _ := document[key].(string)
	return value
}

func classGateBool(document map[string]any, key string) bool {
	value, _ := document[key].(bool)
	return value
}

func classGateInt(document map[string]any, key string) int {
	switch value := document[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	case json.Number:
		parsed, err := value.Int64()
		if err != nil {
			return 0
		}
		return int(parsed)
	default:
		return 0
	}
}

func classGateNumber(document map[string]any, key string) float64 {
	switch value := document[key].(type) {
	case float64:
		return value
	case int:
		return float64(value)
	case json.Number:
		parsed, err := value.Float64()
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}

func stringSet(values []string) map[string]bool {
	set := map[string]bool{}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			set[value] = true
		}
	}
	return set
}

func statusPassed(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "passed", "pass", "success", "successful", "ready", "clear":
		return true
	default:
		return false
	}
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	unique := []string{}
	for _, value := range values {
		if strings.TrimSpace(value) == "" || seen[value] {
			continue
		}
		seen[value] = true
		unique = append(unique, value)
	}
	return unique
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func classGateNestedString(document map[string]any, outer, inner string) string {
	nested, _ := document[outer].(map[string]any)
	return classGateString(nested, inner)
}

func classGateNestedBool(document map[string]any, outer, inner string) bool {
	nested, _ := document[outer].(map[string]any)
	return classGateBool(nested, inner)
}

func classGateStringSlice(document map[string]any, key string) []string {
	rawValues, _ := document[key].([]any)
	values := []string{}
	for _, raw := range rawValues {
		value, ok := raw.(string)
		if ok {
			values = append(values, value)
		}
	}
	return values
}

func classGateMapSliceByRepo(value any) map[string]map[string]any {
	rawValues, _ := value.([]any)
	byRepo := map[string]map[string]any{}
	for _, raw := range rawValues {
		document, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		repo := classGateString(document, "repo")
		if repo != "" {
			byRepo[repo] = document
		}
	}
	return byRepo
}

func classGateTimestampExpired(value string) bool {
	if strings.TrimSpace(value) == "" {
		return true
	}
	expiresAt, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return true
	}
	return !expiresAt.After(time.Now().UTC())
}

func classGateContainsAll(document map[string]any, key string, wants []string) bool {
	values := classGateStringSlice(document, key)
	for _, want := range wants {
		if !classGateStringSliceContains(values, want) {
			return false
		}
	}
	return true
}

func classGateStringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func deniedMutationClasses(allowed string) []string {
	classes := []string{
		"docs_only_single_file",
		"docs_only_multi_file",
		"docs_config_only",
		"test_only",
		"low_risk_code",
		"multi_repo_low_risk",
		"complex_repo_mutation",
	}
	denied := []string{}
	for _, className := range classes {
		if className != allowed {
			denied = append(denied, className)
		}
	}
	return denied
}

func runRepo(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "repo: expected subcommand health or board")
		return 2
	}
	switch args[0] {
	case "health":
		return runRepoHealth(args[1:], stdout, stderr)
	case "board":
		return runRepoBoard(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "repo: unknown subcommand %q\n", args[0])
		return 2
	}
}

func runRepoHealth(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("repo health", stderr)
	registryPath := fs.String("registry", "", "registry path")
	repoID := fs.String("repo", "", "repo id")
	jsonOut := fs.Bool("json", false, "emit JSON health report")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	report, err := buildRepoHealthReport(*registryPath, *repoID)
	if err != nil {
		fmt.Fprintf(stderr, "repo: %v\n", err)
		return 2
	}
	if *jsonOut {
		if err := writeJSON(stdout, report); err != nil {
			fmt.Fprintf(stderr, "repo: marshal health report: %v\n", err)
			return 2
		}
	} else {
		fmt.Fprintf(stdout, "repo_health=%s status=%s repos=%d\n", report.RegistryID, report.Status, len(report.Repos))
		for _, repo := range report.Repos {
			fmt.Fprintf(stdout, "- %s status=%s branch=%s checks=%d\n", repo.RepoID, repo.Status, repo.CurrentBranch, len(repo.Checks))
		}
	}
	if report.Status == "blocked" {
		fmt.Fprintln(stderr, "repo: health blocked")
		return 1
	}
	return 0
}

func runRepoBoard(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("repo board", stderr)
	registryPath := fs.String("registry", "", "registry path")
	jsonOut := fs.Bool("json", false, "emit JSON board")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	board, err := buildRepoBoard(*registryPath)
	if err != nil {
		fmt.Fprintf(stderr, "repo: %v\n", err)
		return 2
	}
	if *jsonOut {
		if err := writeJSON(stdout, board); err != nil {
			fmt.Fprintf(stderr, "repo: marshal board: %v\n", err)
			return 2
		}
	} else {
		fmt.Fprintf(stdout, "repo board: %d repos status=%s\n", len(board.Repos), board.Status)
		for _, repo := range board.Repos {
			fmt.Fprintf(stdout, "- %s tier=%s health=%s recommendation=%s\n", repo.RepoID, repo.Tier, repo.HealthStatus, repo.Recommendation)
			for _, action := range repo.NextActions {
				fmt.Fprintf(stdout, "  next_action=%s\n", action)
			}
		}
		for _, action := range board.NextActions {
			fmt.Fprintf(stdout, "next_action=%s\n", action)
		}
	}
	if board.Status == "blocked" {
		fmt.Fprintln(stderr, "repo: board blocked")
		return 1
	}
	return 0
}

func runLoop(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "loop: expected subcommand preflight, lease, or next")
		return 2
	}
	switch args[0] {
	case "preflight":
		return runLoopPreflight(args[1:], stdout, stderr)
	case "lease":
		return runLoopLease(args[1:], stdout, stderr)
	case "next":
		return runLoopNext(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "loop: unknown subcommand %q\n", args[0])
		return 2
	}
}

func runLoopPreflight(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("loop preflight", stderr)
	goalPath := fs.String("goal-run", "", "goal-run path")
	registryPath := fs.String("registry", "", "registry path")
	taskPath := fs.String("task", "", "task path")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	if err := loopPreflight(*goalPath, *registryPath, *taskPath); err != nil {
		fmt.Fprintf(stderr, "loop: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "loop preflight: ready")
	return 0
}

func runLoopNext(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("loop next", stderr)
	goalPath := fs.String("goal-run", "", "goal-run path")
	registryPath := fs.String("registry", "", "registry path")
	taskPath := fs.String("task", "", "task path")
	outPath := fs.String("out", "", "Forge brief output path")
	approvalPath := fs.String("approval-decision", "", "approval decision path for non-local side effects")
	tracePath := fs.String("trace", "", "trace output path")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	traceStatus := "failed"
	traceProblem := ""
	defer func() {
		writeTraceSpan(*tracePath, "scheduler", "loop.next", traceStatus, map[string]string{"goal_run": *goalPath, "registry": *registryPath, "task": *taskPath}, []string{*outPath}, traceProblem)
	}()
	if *outPath == "" {
		traceProblem = "missing --out"
		fmt.Fprintln(stderr, "loop: missing --out")
		return 2
	}
	if err := loopPreflight(*goalPath, *registryPath, *taskPath); err != nil {
		traceProblem = err.Error()
		fmt.Fprintf(stderr, "loop: %v\n", err)
		return 1
	}
	registry, err := loadRegistry(*registryPath)
	if err != nil {
		traceProblem = err.Error()
		fmt.Fprintf(stderr, "loop: %v\n", err)
		return 2
	}
	task, err := loadTask(*taskPath)
	if err != nil {
		traceProblem = err.Error()
		fmt.Fprintf(stderr, "loop: %v\n", err)
		return 2
	}
	if err := approvalReady(*taskPath, task, *approvalPath); err != nil {
		traceProblem = err.Error()
		fmt.Fprintf(stderr, "loop approval: %v\n", err)
		return 1
	}
	brief, err := buildForgeBrief(registry, task)
	if err != nil {
		traceProblem = err.Error()
		fmt.Fprintf(stderr, "loop: %v\n", err)
		return 2
	}
	if *approvalPath != "" {
		brief.ExpectedEvidence = append(brief.ExpectedEvidence, "approval decision: "+portableEvidencePath(*approvalPath))
	}
	if err := writeJSONFile(*outPath, brief); err != nil {
		traceProblem = err.Error()
		fmt.Fprintf(stderr, "loop: write Forge brief: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "forge_brief=%s\n", *outPath)
	traceStatus = "passed"
	return 0
}

func runLoopLease(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "loop lease: expected subcommand acquire or release")
		return 2
	}
	switch args[0] {
	case "acquire":
		fs := newFlagSet("loop lease acquire", stderr)
		goalPath := fs.String("goal-run", "", "goal-run path")
		leasePath := fs.String("lease", "", "lease path")
		if !parseFlags(fs, args[1:], stderr) {
			return 2
		}
		lease, err := acquireLoopLease(*goalPath, *leasePath)
		if err != nil {
			fmt.Fprintf(stderr, "loop lease: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "lease_acquired=%s\n", lease.LeaseID)
		return 0
	case "release":
		fs := newFlagSet("loop lease release", stderr)
		leasePath := fs.String("lease", "", "lease path")
		if !parseFlags(fs, args[1:], stderr) {
			return 2
		}
		if *leasePath == "" {
			fmt.Fprintln(stderr, "loop lease: missing --lease")
			return 2
		}
		if err := os.Remove(*leasePath); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(stderr, "loop lease: release: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "lease_released=%s\n", *leasePath)
		return 0
	default:
		fmt.Fprintf(stderr, "loop lease: unknown subcommand %q\n", args[0])
		return 2
	}
}

func runApproval(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "approval: expected subcommand request or validate")
		return 2
	}
	switch args[0] {
	case "request":
		fs := newFlagSet("approval request", stderr)
		taskPath := fs.String("task", "", "task path")
		outPath := fs.String("out", "", "approval request output path")
		if !parseFlags(fs, args[1:], stderr) {
			return 2
		}
		request, err := buildApprovalRequest(*taskPath)
		if err != nil {
			fmt.Fprintf(stderr, "approval: %v\n", err)
			return 2
		}
		if *outPath == "" {
			fmt.Fprintln(stderr, "approval: missing --out")
			return 2
		}
		if err := writeJSONFile(*outPath, request); err != nil {
			fmt.Fprintf(stderr, "approval: write request: %v\n", err)
			return 2
		}
		fmt.Fprintf(stdout, "approval_request=%s\n", *outPath)
		return 0
	case "validate":
		fs := newFlagSet("approval validate", stderr)
		decisionPath := fs.String("decision", "", "approval decision path")
		taskPath := fs.String("task", "", "task path")
		tracePath := fs.String("trace", "", "trace output path")
		if !parseFlags(fs, args[1:], stderr) {
			return 2
		}
		traceStatus := "failed"
		traceProblem := ""
		defer func() {
			writeTraceSpan(*tracePath, "approval", "approval.validate", traceStatus, map[string]string{"decision": *decisionPath, "task": *taskPath}, []string{*decisionPath, *taskPath}, traceProblem)
		}()
		if err := validateApprovalDecision(*decisionPath, *taskPath); err != nil {
			traceProblem = err.Error()
			fmt.Fprintf(stderr, "approval: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "approval valid")
		traceStatus = "passed"
		return 0
	default:
		fmt.Fprintf(stderr, "approval: unknown subcommand %q\n", args[0])
		return 2
	}
}

func runEval(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "run" {
		fmt.Fprintln(stderr, "eval: expected subcommand run")
		return 2
	}
	fs := newFlagSet("eval run", stderr)
	runPath := fs.String("run", "", "Foundry run path")
	scorecardPath := fs.String("scorecard", "", "scorecard path")
	outPath := fs.String("out", "", "eval result output path")
	tracePath := fs.String("trace", "", "trace output path")
	if !parseFlags(fs, args[1:], stderr) {
		return 2
	}
	traceStatus := "failed"
	traceProblem := ""
	defer func() {
		writeTraceSpan(*tracePath, "eval", "eval.run", traceStatus, map[string]string{"run": *runPath, "scorecard": *scorecardPath}, []string{*outPath, *runPath, *scorecardPath}, traceProblem)
	}()
	result, err := buildEvalResult(*runPath, *scorecardPath)
	if err != nil {
		traceProblem = err.Error()
		fmt.Fprintf(stderr, "eval: %v\n", err)
		return 2
	}
	if *outPath != "" {
		if err := writeJSONFile(*outPath, result); err != nil {
			traceProblem = err.Error()
			fmt.Fprintf(stderr, "eval: write result: %v\n", err)
			return 2
		}
	}
	fmt.Fprintf(stdout, "eval score: %d/%d status=%s\n", result.Score, result.MaxScore, result.Status)
	if result.Score < result.Threshold {
		traceProblem = "score below threshold"
		fmt.Fprintln(stderr, "eval: score below threshold")
		return 1
	}
	traceStatus = "passed"
	return 0
}

func runRSI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "improvement-gate" {
		fmt.Fprintln(stderr, "rsi: expected subcommand improvement-gate")
		return 2
	}
	fs := newFlagSet("rsi improvement-gate", stderr)
	baselinePath := fs.String("baseline", "", "baseline eval result path")
	candidatePath := fs.String("candidate", "", "candidate eval result path")
	minImprovement := fs.Float64("min-improvement", 5, "minimum improvement percentage points")
	outPath := fs.String("out", "", "RSI improvement gate output path")
	if !parseFlags(fs, args[1:], stderr) {
		return 2
	}
	gate, err := buildRSIImprovementGate(*baselinePath, *candidatePath, *minImprovement)
	if err != nil {
		fmt.Fprintf(stderr, "rsi improvement-gate: %v\n", err)
		return 2
	}
	if *outPath != "" {
		if err := writeJSONFile(*outPath, gate); err != nil {
			fmt.Fprintf(stderr, "rsi improvement-gate: write gate: %v\n", err)
			return 2
		}
	}
	fmt.Fprintf(stdout, "rsi improvement: %s delta=%.2f required=%.2f\n", gate.Status, gate.ActualImprovementPercent, gate.RequiredImprovementPercent)
	if gate.Status != "passed" {
		fmt.Fprintln(stderr, "rsi improvement-gate: improvement below required threshold")
		return 1
	}
	return 0
}

func runTrace(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "inspect" {
		fmt.Fprintln(stderr, "trace: expected subcommand inspect")
		return 2
	}
	fs := newFlagSet("trace inspect", stderr)
	tracePath := fs.String("trace", "", "trace path")
	jsonOut := fs.Bool("json", false, "emit JSON trace spans")
	if !parseFlags(fs, args[1:], stderr) {
		return 2
	}
	spans, err := readTraceSpans(*tracePath)
	if err != nil {
		fmt.Fprintf(stderr, "trace: %v\n", err)
		return 2
	}
	if *jsonOut {
		if err := writeJSON(stdout, spans); err != nil {
			fmt.Fprintf(stderr, "trace: marshal spans: %v\n", err)
			return 2
		}
		return 0
	}
	failed := 0
	evidenceRefs := 0
	duration := 0
	for _, span := range spans {
		if span.Status == "failed" {
			failed++
		}
		evidenceRefs += len(span.EvidenceRefs)
		duration += span.DurationMS
	}
	fmt.Fprintf(stdout, "trace spans=%d duration_ms=%d failed_spans=%d evidence_refs=%d\n", len(spans), duration, failed, evidenceRefs)
	return 0
}

func runImport(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "ao2-sdd" {
		fmt.Fprintln(stderr, "import: expected subcommand ao2-sdd")
		return 2
	}
	fs := newFlagSet("import ao2-sdd", stderr)
	planPath := fs.String("plan", "", "AO2 SDD plan path")
	outPath := fs.String("out", "", "Foundry task output path")
	if !parseFlags(fs, args[1:], stderr) {
		return 2
	}
	task, err := importAO2SDDPlan(*planPath)
	if err != nil {
		fmt.Fprintf(stderr, "import: %v\n", err)
		return 2
	}
	if *outPath == "" {
		fmt.Fprintln(stderr, "import: missing --out")
		return 2
	}
	if err := writeJSONFile(*outPath, task); err != nil {
		fmt.Fprintf(stderr, "import: write task: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "foundry_task=%s\n", *outPath)
	return 0
}

func runExport(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "forge-brief" {
		fmt.Fprintln(stderr, "export: expected subcommand forge-brief")
		return 2
	}
	fs := newFlagSet("export forge-brief", stderr)
	taskPath := fs.String("task", "", "Foundry task path")
	registryPath := fs.String("registry", "", "registry path")
	outPath := fs.String("out", "", "Forge brief output path")
	if !parseFlags(fs, args[1:], stderr) {
		return 2
	}
	registry, err := loadRegistry(*registryPath)
	if err != nil {
		fmt.Fprintf(stderr, "export: %v\n", err)
		return 2
	}
	task, err := loadTask(*taskPath)
	if err != nil {
		fmt.Fprintf(stderr, "export: %v\n", err)
		return 2
	}
	brief, err := buildForgeBrief(registry, task)
	if err != nil {
		fmt.Fprintf(stderr, "export: %v\n", err)
		return 2
	}
	if *outPath == "" {
		fmt.Fprintln(stderr, "export: missing --out")
		return 2
	}
	if err := writeJSONFile(*outPath, brief); err != nil {
		fmt.Fprintf(stderr, "export: write brief: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "forge_brief=%s\n", *outPath)
	return 0
}

func runCompare(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "capabilities" {
		fmt.Fprintln(stderr, "compare: expected subcommand capabilities")
		return 2
	}
	fs := newFlagSet("compare capabilities", stderr)
	outPath := fs.String("out", "", "capability matrix output path")
	if !parseFlags(fs, args[1:], stderr) {
		return 2
	}
	matrix := CapabilityMatrix{
		SchemaVersion: "ao.foundry.capability-matrix.v0.1",
		Capabilities: []CapabilityMapping{
			{Capability: "durable workflow state", Status: "supported", Foundry: "GoalRun, run records, loop leases", Evidence: "docs/contracts/foundry-goal-run-v0.1.schema.json"},
			{Capability: "traces and evals", Status: "supported", Foundry: "local trace spans and eval scorecards", Evidence: "docs/contracts/foundry-trace-v0.1.schema.json"},
			{Capability: "crews and flows", Status: "partial", Foundry: "delegates governed execution to AO Forge workcells", Evidence: "docs/contracts/foundry-task-v0.1.schema.json"},
			{Capability: "hosted dashboard", Status: "out-of-scope", Foundry: "local CLI and public-safe evidence only", Evidence: "README.md"},
		},
	}
	if *outPath == "" {
		fmt.Fprintln(stderr, "compare: missing --out")
		return 2
	}
	if err := writeJSONFile(*outPath, matrix); err != nil {
		fmt.Fprintf(stderr, "compare: write matrix: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "capability_matrix=%s\n", *outPath)
	return 0
}

func runDemo(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "demo: expected subcommand status or script")
		return 2
	}
	switch args[0] {
	case "status":
		fs := newFlagSet("demo status", stderr)
		registryPath := fs.String("registry", "", "registry path")
		taskPath := fs.String("task", "", "task path")
		runPath := fs.String("run", "", "run path")
		jsonOut := fs.Bool("json", false, "emit JSON demo status")
		if !parseFlags(fs, args[1:], stderr) {
			return 2
		}
		status, err := buildDemoStatus(*registryPath, *taskPath, *runPath)
		if err != nil {
			fmt.Fprintf(stderr, "demo: %v\n", err)
			return 2
		}
		if *jsonOut {
			if err := writeJSON(stdout, status); err != nil {
				fmt.Fprintf(stderr, "demo: marshal status: %v\n", err)
				return 2
			}
			return 0
		}
		fmt.Fprintf(stdout, "demo status=%s registry=%s task=%s run=%s\n", status.Status, status.RegistryID, status.TaskID, status.RunID)
		for _, step := range status.Story {
			fmt.Fprintf(stdout, "- %s\n", step)
		}
		fmt.Fprintf(stdout, "next_action=%s\n", status.NextAction)
		return 0
	case "script":
		fs := newFlagSet("demo script", stderr)
		outPath := fs.String("out", "", "markdown output path")
		if !parseFlags(fs, args[1:], stderr) {
			return 2
		}
		if *outPath == "" {
			fmt.Fprintln(stderr, "demo: missing --out")
			return 2
		}
		if err := writeDemoScript(*outPath); err != nil {
			fmt.Fprintf(stderr, "demo: write script: %v\n", err)
			return 2
		}
		fmt.Fprintf(stdout, "demo_script=%s\n", *outPath)
		return 0
	default:
		fmt.Fprintf(stderr, "demo: unknown subcommand %q\n", args[0])
		return 2
	}
}

func runRelease(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "release: expected subcommand manifest, dry-run, handoff, candidate, promotion, or validate-manifest")
		return 2
	}
	switch args[0] {
	case "manifest", "dry-run":
		fs := newFlagSet("release "+args[0], stderr)
		outPath := fs.String("out", "", "release manifest output path")
		if !parseFlags(fs, args[1:], stderr) {
			return 2
		}
		if *outPath == "" {
			fmt.Fprintln(stderr, "release: missing --out")
			return 2
		}
		manifest, err := buildReleaseManifest(args[0] == "dry-run")
		if err != nil {
			fmt.Fprintf(stderr, "release: %v\n", err)
			return 2
		}
		if err := writeJSONFile(*outPath, manifest); err != nil {
			fmt.Fprintf(stderr, "release: write manifest: %v\n", err)
			return 2
		}
		fmt.Fprintf(stdout, "release_manifest=%s\n", *outPath)
		fmt.Fprintf(stdout, "files=%d status=%s\n", len(manifest.Files), manifest.Status)
		return 0
	case "validate-manifest":
		fs := newFlagSet("release validate-manifest", stderr)
		manifestPath := fs.String("manifest", "", "release manifest path")
		if !parseFlags(fs, args[1:], stderr) {
			return 2
		}
		if err := validateReleaseManifestFile(*manifestPath); err != nil {
			fmt.Fprintf(stderr, "release: %v\n", err)
			return 2
		}
		fmt.Fprintf(stdout, "release manifest valid: %s\n", *manifestPath)
		return 0
	case "handoff":
		return runReleaseHandoff(args[1:], stdout, stderr)
	case "candidate":
		return runReleaseCandidate(args[1:], stdout, stderr)
	case "promotion":
		return runReleasePromotion(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "release: unknown subcommand %q\n", args[0])
		return 2
	}
}

func runReleaseHandoff(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("release handoff", stderr)
	candidatePath := fs.String("candidate", "", "release candidate ledger path")
	summaryPath := fs.String("signed-smoke-summary", "", "signed-smoke summary path")
	promotionOut := fs.String("promotion-out", "", "release promotion ledger output path")
	notesOut := fs.String("notes-out", "", "release candidate notes markdown output path")
	manifestOut := fs.String("manifest-out", "", "release manifest output path")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	for flagName, value := range map[string]string{
		"--promotion-out": *promotionOut,
		"--notes-out":     *notesOut,
		"--manifest-out":  *manifestOut,
	} {
		if strings.TrimSpace(value) == "" {
			fmt.Fprintf(stderr, "release handoff: missing %s\n", flagName)
			return 2
		}
	}
	candidate, err := loadReleaseCandidateLedger(*candidatePath)
	if err != nil {
		fmt.Fprintf(stderr, "release handoff: %v\n", err)
		return 2
	}
	promotion, err := buildReleasePromotionLedger(*candidatePath, *summaryPath)
	if err != nil {
		fmt.Fprintf(stderr, "release handoff: %v\n", err)
		return 2
	}
	if promotion.CandidateID != candidate.CandidateID {
		fmt.Fprintf(stderr, "release handoff: promotion candidate %q does not match %q\n", promotion.CandidateID, candidate.CandidateID)
		return 2
	}
	if err := writeJSONFile(*promotionOut, promotion); err != nil {
		fmt.Fprintf(stderr, "release handoff: write promotion ledger: %v\n", err)
		return 2
	}
	notes := renderReleaseCandidateNotes(candidate, promotion)
	if err := writeTextFile(*notesOut, notes); err != nil {
		fmt.Fprintf(stderr, "release handoff: write notes: %v\n", err)
		return 2
	}
	manifest, err := buildReleaseManifest(true)
	if err != nil {
		fmt.Fprintf(stderr, "release handoff: %v\n", err)
		return 2
	}
	if err := writeJSONFile(*manifestOut, manifest); err != nil {
		fmt.Fprintf(stderr, "release handoff: write manifest: %v\n", err)
		return 2
	}
	if err := validateReleaseManifestFile(*manifestOut); err != nil {
		fmt.Fprintf(stderr, "release handoff: %v\n", err)
		return 2
	}
	fmt.Fprintln(stdout, "release_handoff=ready")
	fmt.Fprintf(stdout, "candidate=%s\n", candidate.CandidateID)
	fmt.Fprintf(stdout, "release_safe=%t\n", promotion.ReleaseSafe)
	fmt.Fprintf(stdout, "release_promotion=%s\n", *promotionOut)
	fmt.Fprintf(stdout, "release_candidate_notes=%s\n", *notesOut)
	fmt.Fprintf(stdout, "release_manifest=%s\n", *manifestOut)
	return 0
}

func runReleaseCandidate(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "release candidate: expected subcommand validate, active-stack-parity, or notes")
		return 2
	}
	switch args[0] {
	case "validate":
		fs := newFlagSet("release candidate validate", stderr)
		ledgerPath := fs.String("ledger", "", "release candidate ledger path")
		if !parseFlags(fs, args[1:], stderr) {
			return 2
		}
		ledger, err := loadReleaseCandidateLedger(*ledgerPath)
		if err != nil {
			fmt.Fprintf(stderr, "release candidate: %v\n", err)
			return 2
		}
		fmt.Fprintf(stdout, "release_candidate=%s\n", ledger.CandidateID)
		fmt.Fprintf(stdout, "status=%s\n", ledger.Status)
		fmt.Fprintf(stdout, "repos=%d\n", len(ledger.ActiveSpine))
		if gate, ok := releaseCandidateGateByName(ledger, "signed_smoke_release_gate"); ok {
			fmt.Fprintf(stdout, "signed_smoke=%s\n", gate.Status)
		}
		return 0
	case "active-stack-parity":
		fs := newFlagSet("release candidate active-stack-parity", stderr)
		ledgerPath := fs.String("ledger", "", "release candidate ledger path")
		readinessLedgerPath := fs.String("readiness-ledger", "", "active-stack readiness ledger path")
		if !parseFlags(fs, args[1:], stderr) {
			return 2
		}
		candidate, err := loadReleaseCandidateLedger(*ledgerPath)
		if err != nil {
			fmt.Fprintf(stderr, "release candidate active-stack parity: %v\n", err)
			return 2
		}
		readiness, err := loadActiveStackReadinessLedger(*readinessLedgerPath)
		if err != nil {
			fmt.Fprintf(stderr, "release candidate active-stack parity: %v\n", err)
			return 2
		}
		issues, reposChecked := checkReleaseCandidateActiveStackParity(candidate, readiness)
		if len(issues) > 0 {
			for _, issue := range issues {
				fmt.Fprintf(stderr, "release candidate active-stack parity: %s\n", issue)
			}
			return 1
		}
		fmt.Fprintln(stdout, "release_candidate_active_stack_parity=ready")
		fmt.Fprintf(stdout, "candidate=%s\n", candidate.CandidateID)
		fmt.Fprintf(stdout, "repos_checked=%d\n", reposChecked)
		return 0
	case "notes":
		fs := newFlagSet("release candidate notes", stderr)
		ledgerPath := fs.String("ledger", "", "release candidate ledger path")
		promotionPath := fs.String("promotion", "", "release promotion ledger path")
		outPath := fs.String("out", "", "release candidate notes markdown output path")
		if !parseFlags(fs, args[1:], stderr) {
			return 2
		}
		if strings.TrimSpace(*outPath) == "" {
			fmt.Fprintln(stderr, "release candidate notes: missing --out")
			return 2
		}
		candidate, err := loadReleaseCandidateLedger(*ledgerPath)
		if err != nil {
			fmt.Fprintf(stderr, "release candidate notes: %v\n", err)
			return 2
		}
		promotion, err := loadReleasePromotionLedger(*promotionPath)
		if err != nil {
			fmt.Fprintf(stderr, "release candidate notes: %v\n", err)
			return 2
		}
		if promotion.CandidateID != candidate.CandidateID {
			fmt.Fprintf(stderr, "release candidate notes: promotion candidate %q does not match %q\n", promotion.CandidateID, candidate.CandidateID)
			return 2
		}
		notes := renderReleaseCandidateNotes(candidate, promotion)
		if err := os.MkdirAll(parentDir(*outPath), 0o755); err != nil {
			fmt.Fprintf(stderr, "release candidate notes: mkdir output parent: %v\n", err)
			return 2
		}
		if err := os.WriteFile(*outPath, []byte(notes), 0o644); err != nil {
			fmt.Fprintf(stderr, "release candidate notes: write notes: %v\n", err)
			return 2
		}
		fmt.Fprintf(stdout, "release_candidate_notes=%s\n", *outPath)
		fmt.Fprintf(stdout, "candidate=%s\n", candidate.CandidateID)
		fmt.Fprintf(stdout, "release_safe=%t\n", promotion.ReleaseSafe)
		return 0
	default:
		fmt.Fprintf(stderr, "release candidate: unknown subcommand %q\n", args[0])
		return 2
	}
}

func runReleasePromotion(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "release promotion: expected subcommand validate")
		return 2
	}
	switch args[0] {
	case "validate":
		fs := newFlagSet("release promotion validate", stderr)
		candidatePath := fs.String("candidate", "", "release candidate ledger path")
		summaryPath := fs.String("signed-smoke-summary", "", "signed-smoke summary path")
		outPath := fs.String("out", "", "release promotion ledger output path")
		if !parseFlags(fs, args[1:], stderr) {
			return 2
		}
		if strings.TrimSpace(*outPath) == "" {
			fmt.Fprintln(stderr, "release promotion: missing --out")
			return 2
		}
		ledger, err := buildReleasePromotionLedger(*candidatePath, *summaryPath)
		if err != nil {
			fmt.Fprintf(stderr, "release promotion: %v\n", err)
			return 2
		}
		if err := writeJSONFile(*outPath, ledger); err != nil {
			fmt.Fprintf(stderr, "release promotion: write ledger: %v\n", err)
			return 2
		}
		fmt.Fprintf(stdout, "release_promotion=%s\n", *outPath)
		fmt.Fprintf(stdout, "candidate=%s\n", ledger.CandidateID)
		fmt.Fprintf(stdout, "status=%s\n", ledger.Status)
		fmt.Fprintf(stdout, "release_safe=%t\n", ledger.ReleaseSafe)
		fmt.Fprintf(stdout, "signed_smoke=%s\n", ledger.SignedSmokePulseID)
		return 0
	default:
		fmt.Fprintf(stderr, "release promotion: unknown subcommand %q\n", args[0])
		return 2
	}
}

func runContract(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "contract: expected subcommand fixtures")
		return 2
	}
	switch args[0] {
	case "fixtures":
		return runContractFixtures(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "contract: unknown subcommand %q\n", args[0])
		return 2
	}
}

func runContractFixtures(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "contract fixtures: expected subcommand validate")
		return 2
	}
	switch args[0] {
	case "validate":
		result, err := validateContractFixtures()
		if err != nil {
			fmt.Fprintf(stderr, "contract fixtures: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "contract_fixtures=valid")
		fmt.Fprintf(stdout, "valid_fixtures=%d\n", result.ValidFixtures)
		fmt.Fprintf(stdout, "invalid_fixtures=%d\n", result.InvalidFixtures)
		return 0
	default:
		fmt.Fprintf(stderr, "contract fixtures: unknown subcommand %q\n", args[0])
		return 2
	}
}

func runCompetitive(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "audit" {
		fmt.Fprintln(stderr, "competitive: expected subcommand audit")
		return 2
	}
	fs := newFlagSet("competitive audit", stderr)
	outPath := fs.String("out", "", "competitive audit output path")
	jsonOut := fs.Bool("json", false, "emit JSON audit to stdout")
	if !parseFlags(fs, args[1:], stderr) {
		return 2
	}
	audit := buildCompetitiveAudit()
	if *outPath != "" {
		if err := writeJSONFile(*outPath, audit); err != nil {
			fmt.Fprintf(stderr, "competitive: write audit: %v\n", err)
			return 2
		}
	}
	if *jsonOut {
		if err := writeJSON(stdout, audit); err != nil {
			fmt.Fprintf(stderr, "competitive: marshal audit: %v\n", err)
			return 2
		}
	} else {
		fmt.Fprintf(stdout, "competitive readiness: %d/%d status=%s\n", audit.Score, audit.MaxScore, audit.Status)
		if *outPath != "" {
			fmt.Fprintf(stdout, "competitive_audit=%s\n", *outPath)
		}
		for _, action := range audit.NextActions {
			fmt.Fprintf(stdout, "next_action=%s\n", action)
		}
	}
	if audit.Status != "ready" {
		return 1
	}
	return 0
}

func runPulse(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "pulse: expected subcommand run, intake-preflight, lifecycle, overnight-start-gate, event-loop-policy, or signed-smoke-script")
		return 2
	}
	switch args[0] {
	case "run":
		return runPulseRun(args[1:], stdout, stderr)
	case "intake-preflight":
		return runPulseIntakePreflight(args[1:], stdout, stderr)
	case "lifecycle":
		return runPulseLifecycle(args[1:], stdout, stderr)
	case "overnight-start-gate":
		return runPulseOvernightStartGate(args[1:], stdout, stderr)
	case "event-loop-policy":
		return runPulseEventLoopPolicy(args[1:], stdout, stderr)
	case "signed-smoke-script":
		return runPulseSignedSmokeScript(args[1:], stdout, stderr)
	case "signed-smoke-preflight":
		return runPulseSignedSmokePreflight(args[1:], stdout, stderr)
	case "signed-smoke-cleanup":
		return runPulseSignedSmokeCleanup(args[1:], stdout, stderr)
	case "ingest-signed-smoke":
		return runPulseIngestSignedSmoke(args[1:], stdout, stderr)
	case "summarize-signed-smoke":
		return runPulseSummarizeSignedSmoke(args[1:], stdout, stderr)
	case "decision":
		return runPulseDecision(args[1:], stdout, stderr)
	case "derive-next":
		return runPulseDeriveNext(args[1:], stdout, stderr)
	case "freshness":
		return runPulseFreshness(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "pulse: unknown subcommand %q\n", args[0])
		return 2
	}
}

func runPulseIntakePreflight(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("pulse intake-preflight", stderr)
	blueprintAuthorizationPath := fs.String("blueprint-authorization", "", "Blueprint build authorization artifact")
	blueprintRequestPath := fs.String("blueprint-request", "", "Blueprint blocked clarification request artifact")
	atlasBlueprintImportPath := fs.String("atlas-blueprint-import", "", "Atlas Blueprint import artifact")
	atlasImportPath := fs.String("atlas-import", "", "Atlas foundry-import artifact")
	atlasStatusPath := fs.String("atlas-status", "", "Foundry Atlas status/readback artifact")
	requiresAtlas := fs.Bool("requires-atlas", false, "require Atlas handoff/readback evidence")
	outPath := fs.String("out", "", "preflight result output path")
	jsonOut := fs.Bool("json", false, "emit JSON result to stdout")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	result, err := buildPulseIntakePreflight(*blueprintAuthorizationPath, *blueprintRequestPath, *atlasBlueprintImportPath, *atlasImportPath, *atlasStatusPath, *requiresAtlas)
	if strings.TrimSpace(*outPath) != "" {
		for _, inputPath := range []string{*blueprintAuthorizationPath, *blueprintRequestPath, *atlasBlueprintImportPath, *atlasImportPath, *atlasStatusPath} {
			if sameCleanPath(*outPath, inputPath) {
				fmt.Fprintln(stderr, "pulse intake-preflight: --out must not overwrite input artifacts")
				return 2
			}
		}
		if writeErr := writeJSONFile(*outPath, result); writeErr != nil {
			fmt.Fprintf(stderr, "pulse intake-preflight: write result: %v\n", writeErr)
			return 2
		}
	}
	if *jsonOut {
		if writeErr := writeJSON(stdout, result); writeErr != nil {
			fmt.Fprintf(stderr, "pulse intake-preflight: marshal result: %v\n", writeErr)
			return 2
		}
	} else {
		fmt.Fprintf(stdout, "pulse_intake_preflight=%s\n", result.Status)
		fmt.Fprintf(stdout, "blueprint_status=%s\n", result.BlueprintStatus)
		fmt.Fprintf(stdout, "atlas_blueprint_status=%s\n", result.AtlasBlueprintStatus)
		fmt.Fprintf(stdout, "atlas_status=%s\n", result.AtlasStatus)
		if strings.TrimSpace(*outPath) != "" {
			fmt.Fprintf(stdout, "preflight_result=%s\n", *outPath)
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "pulse intake-preflight: %v\n", err)
		return 1
	}
	if result.Status != "ready" {
		return 1
	}
	return 0
}

func buildPulseIntakePreflight(blueprintAuthorizationPath, blueprintRequestPath, atlasBlueprintImportPath, atlasImportPath, atlasStatusPath string, requiresAtlas bool) (PulseIntakePreflight, error) {
	result := PulseIntakePreflight{
		SchemaVersion:          pulseIntakeSchema,
		Status:                 "ready",
		BlueprintStatus:        "missing",
		AtlasStatus:            "not_required",
		AtlasBlueprintStatus:   "not_required",
		Checks:                 []PulseIntakeCheck{},
		BlockingNextActions:    []string{},
		MaintenanceSuggestions: []string{"keep pulse intake preflight fixture/local; do not schedule, execute, approve, upload, or mutate sibling repositories"},
		SourceArtifacts:        []PulseIntakeSource{},
	}
	var blueprintSource PulseIntakeSource
	if strings.TrimSpace(blueprintAuthorizationPath) != "" && strings.TrimSpace(blueprintRequestPath) != "" {
		return failPulseIntake(result, "blueprint_build_authorization", "failed", "provide either Blueprint authorization or Blueprint request, not both", "Use exactly one Blueprint intake artifact.")
	}
	switch {
	case strings.TrimSpace(blueprintAuthorizationPath) != "":
		source, status, err := loadPulseIntakeSource("blueprint_authorization", blueprintAuthorizationPath, "ao.blueprint.build-authorization.v0.1")
		if err != nil {
			return failPulseIntake(result, "blueprint_build_authorization", "failed", err.Error(), "Regenerate the Blueprint build authorization artifact.")
		}
		result.SourceArtifacts = append(result.SourceArtifacts, source)
		blueprintSource = source
		result.BlueprintStatus = status
		if status != "ready" {
			return failPulseIntake(result, "blueprint_build_authorization", "failed", "Blueprint authorization is blocked; Pulse must not proceed as ready", "Return to AO Blueprint for requirements clarification.")
		}
		result.Checks = append(result.Checks, PulseIntakeCheck{Name: "blueprint_build_authorization", Status: "pass", Reason: "Blueprint build authorization is ready."})
	case strings.TrimSpace(blueprintRequestPath) != "":
		source, status, err := loadPulseIntakeSource("blueprint_request", blueprintRequestPath, "ao.blueprint.request.v0.1")
		if err != nil {
			return failPulseIntake(result, "blueprint_build_authorization", "failed", err.Error(), "Regenerate the Blueprint clarification request.")
		}
		result.SourceArtifacts = append(result.SourceArtifacts, source)
		result.BlueprintStatus = "blocked"
		if status != "blocked" && status != "needs_clarification" {
			return failPulseIntake(result, "blueprint_build_authorization", "failed", "Blueprint request must be blocked or needs_clarification", "Regenerate the Blueprint request with a blocked clarification status.")
		}
		return failPulseIntake(result, "blueprint_build_authorization", "blocked", "Blueprint request exists; work is not build-ready.", "Answer the Blueprint clarification request before scheduling implementation.")
	default:
		return failPulseIntake(result, "blueprint_build_authorization", "failed", "Blueprint authorization is required for build-ready Pulse intake", "Run AO Blueprint and provide a ready build authorization or blocked clarification request.")
	}

	if requiresAtlas {
		result.AtlasStatus = "missing"
		result.AtlasBlueprintStatus = "missing"
		if strings.TrimSpace(atlasBlueprintImportPath) == "" {
			return failPulseIntake(result, "atlas_blueprint_import", "failed", "Atlas Blueprint import is required before Foundry gates for oversized or live-mutation work", "Run AO Atlas blueprint import and provide the ready import record.")
		}
		if strings.TrimSpace(atlasImportPath) == "" || strings.TrimSpace(atlasStatusPath) == "" {
			return failPulseIntake(result, "atlas_handoff_readback", "failed", "Atlas handoff/readback is required for oversized Pulse intake", "Provide Atlas Blueprint import, Foundry import, and Foundry Atlas status/readback artifacts.")
		}
		importArtifact, err := loadAtlasFoundryImport(atlasImportPath)
		if err != nil {
			if isAtlasAuthorityError(err) {
				return failPulseIntake(result, "atlas_authority_boundary", "failed", "Atlas artifact claims forbidden authority", "Regenerate Atlas artifacts with schedules_work=false, executes_work=false, and approves_work=false.")
			}
			return failPulseIntake(result, "atlas_handoff_readback", "failed", err.Error(), "Regenerate the Atlas Foundry handoff/import artifact.")
		}
		importSource, err := pulseIntakeSourceFromFile("atlas_import", atlasImportPath, importArtifact.ContractVersion, importArtifact.Status)
		if err != nil {
			return failPulseIntake(result, "atlas_handoff_readback", "failed", err.Error(), "Use a public-safe Atlas import path.")
		}
		result.SourceArtifacts = append(result.SourceArtifacts, importSource)

		blueprintImport, err := loadAtlasBlueprintImport(atlasBlueprintImportPath)
		if err != nil {
			if isAtlasAuthorityError(err) {
				return failPulseIntake(result, "atlas_authority_boundary", "failed", "Atlas Blueprint import claims forbidden authority", "Regenerate Atlas Blueprint import with schedules_work=false, executes_work=false, and approves_work=false.")
			}
			return failPulseIntake(result, "atlas_blueprint_import", "failed", err.Error(), "Regenerate the Atlas Blueprint import artifact.")
		}
		if err := validateAtlasBlueprintImportForFoundry(blueprintImport, importArtifact, blueprintSource, importSource); err != nil {
			if isAtlasAuthorityError(err) {
				return failPulseIntake(result, "atlas_authority_boundary", "failed", "Atlas Blueprint import claims forbidden authority", "Regenerate Atlas Blueprint import with schedules_work=false, executes_work=false, and approves_work=false.")
			}
			return failPulseIntake(result, "atlas_blueprint_import", "failed", err.Error(), "Regenerate the Atlas Blueprint import artifact.")
		}
		blueprintImportSource, err := pulseIntakeSourceFromFile("atlas_blueprint_import", atlasBlueprintImportPath, blueprintImport.ContractVersion, blueprintImport.Status)
		if err != nil {
			return failPulseIntake(result, "atlas_blueprint_import", "failed", err.Error(), "Use a public-safe Atlas Blueprint import path.")
		}
		result.SourceArtifacts = append(result.SourceArtifacts, blueprintImportSource)

		var atlasStatus AtlasStatus
		if err := readJSONFile(atlasStatusPath, &atlasStatus); err != nil {
			return failPulseIntake(result, "atlas_handoff_readback", "failed", fmt.Sprintf("read Atlas status: %v", err), "Regenerate the Foundry Atlas status/readback artifact.")
		}
		if err := validatePulseAtlasStatus(atlasStatus, importArtifact); err != nil {
			if isAtlasAuthorityError(err) {
				return failPulseIntake(result, "atlas_authority_boundary", "failed", "Atlas artifact claims forbidden authority", "Regenerate Atlas status with schedules_work=false, executes_work=false, and approves_work=false.")
			}
			return failPulseIntake(result, "atlas_handoff_readback", "failed", err.Error(), "Regenerate the Foundry Atlas status/readback artifact.")
		}
		statusSource, err := pulseIntakeSourceFromFile("atlas_status", atlasStatusPath, atlasStatus.SchemaVersion, atlasStatus.Status)
		if err != nil {
			return failPulseIntake(result, "atlas_handoff_readback", "failed", err.Error(), "Use a public-safe Atlas status path.")
		}
		result.SourceArtifacts = append(result.SourceArtifacts, statusSource)
		result.AtlasBlueprintStatus = "ready"
		result.AtlasStatus = "ready"
		result.Checks = append(result.Checks,
			PulseIntakeCheck{Name: "atlas_blueprint_import", Status: "pass", Reason: "Atlas Blueprint import is ready and digest-bound."},
			PulseIntakeCheck{Name: "atlas_handoff_readback", Status: "pass", Reason: "Atlas handoff/import and Foundry status readback are ready."},
			PulseIntakeCheck{Name: "atlas_authority_boundary", Status: "pass", Reason: "Atlas artifacts preserve compile-only authority."},
		)
	} else {
		result.Checks = append(result.Checks, PulseIntakeCheck{Name: "atlas_handoff_readback", Status: "pass", Reason: "Atlas handoff/readback is not required for this bounded intake."})
	}
	return result, nil
}

func loadPulseIntakeSource(name, path, expectedSchema string) (PulseIntakeSource, string, error) {
	if err := validateEvidencePath(path); err != nil {
		return PulseIntakeSource{}, "", fmt.Errorf("unsafe source artifact path: %w", err)
	}
	document, err := readArbitraryJSON(path)
	if err != nil {
		return PulseIntakeSource{}, "", err
	}
	object, ok := document.(map[string]any)
	if !ok {
		return PulseIntakeSource{}, "", errors.New("source artifact must be a JSON object")
	}
	if err := validatePublicSafeJSONStrings(object); err != nil {
		return PulseIntakeSource{}, "", err
	}
	schemaVersion, _ := object["schema_version"].(string)
	if schemaVersion == "" {
		schemaVersion, _ = object["contract_version"].(string)
	}
	if schemaVersion != expectedSchema {
		return PulseIntakeSource{}, "", fmt.Errorf("unexpected source artifact schema %q", schemaVersion)
	}
	status, _ := object["status"].(string)
	if strings.TrimSpace(status) == "" {
		return PulseIntakeSource{}, "", errors.New("source artifact status is required")
	}
	source, err := pulseIntakeSourceFromFile(name, path, schemaVersion, status)
	if err != nil {
		return PulseIntakeSource{}, "", err
	}
	return source, status, nil
}

func pulseIntakeSourceFromFile(name, path, schemaVersion, status string) (PulseIntakeSource, error) {
	if err := validateEvidencePath(path); err != nil {
		return PulseIntakeSource{}, fmt.Errorf("unsafe source artifact path: %w", err)
	}
	sum, err := fileSHA256(path)
	if err != nil {
		return PulseIntakeSource{}, err
	}
	return PulseIntakeSource{
		Name:          name,
		Path:          filepath.ToSlash(filepath.Clean(path)),
		SchemaVersion: schemaVersion,
		Status:        status,
		SHA256:        sum,
	}, nil
}

func validatePulseAtlasStatus(status AtlasStatus, artifact AtlasFoundryImport) error {
	if status.SchemaVersion != atlasStatusSchema {
		return fmt.Errorf("Atlas status schema_version must be %s", atlasStatusSchema)
	}
	if status.Status != "ready" || status.ReadbackStatus != "ready" {
		return errors.New("Atlas status and readback_status must be ready")
	}
	if status.ImportID != artifact.ID || status.WorkgraphID != artifact.WorkgraphID || status.TargetInstance != artifact.TargetInstance {
		return errors.New("Atlas status must match Atlas import identity")
	}
	if status.SchedulesWork {
		return errors.New("schedules_work must be false")
	}
	if status.ExecutesWork {
		return errors.New("executes_work must be false")
	}
	if status.ApprovesWork {
		return errors.New("approves_work must be false")
	}
	for label, path := range status.Evidence {
		if strings.TrimSpace(label) == "" {
			return errors.New("Atlas status evidence labels must not be empty")
		}
		if err := validateEvidencePath(path); err != nil {
			return fmt.Errorf("Atlas status evidence %s: %w", label, err)
		}
	}
	return validatePublicSafeJSONStrings(status)
}

func failPulseIntake(result PulseIntakePreflight, checkName, status, reason, nextAction string) (PulseIntakePreflight, error) {
	result.Status = status
	if status == "blocked" {
		result.BlueprintStatus = "blocked"
	} else if result.BlueprintStatus == "" || result.BlueprintStatus == "missing" {
		result.BlueprintStatus = "missing"
	}
	result.FirstFailingCheck = checkName
	result.Checks = append(result.Checks, PulseIntakeCheck{Name: checkName, Status: "fail", Reason: reason})
	result.BlockingNextActions = append(result.BlockingNextActions, nextAction)
	return result, errors.New(reason)
}

func isAtlasAuthorityError(err error) bool {
	message := err.Error()
	return strings.Contains(message, "schedules_work") ||
		strings.Contains(message, "executes_work") ||
		strings.Contains(message, "approves_work") ||
		strings.Contains(message, "mutates_repositories") ||
		strings.Contains(message, "calls_providers") ||
		strings.Contains(message, "release_or_publish_allowed") ||
		strings.Contains(message, "live execution safe or proven")
}

func validatePublicSafeJSONStrings(value any) error {
	switch typed := value.(type) {
	case map[string]any:
		for _, nested := range typed {
			if err := validatePublicSafeJSONStrings(nested); err != nil {
				return err
			}
		}
	case []any:
		for _, nested := range typed {
			if err := validatePublicSafeJSONStrings(nested); err != nil {
				return err
			}
		}
	case string:
		lower := strings.ToLower(strings.ReplaceAll(typed, "\\", "/"))
		for _, marker := range []string{
			"/" + "users/",
			"/" + "home/",
			"/" + "tmp/",
			"/" + "var/folders/",
			"downloads/",
			"file://",
			"api" + "_key",
			"access" + "_token",
			"authorization: bearer",
			"begin " + "rsa",
			"begin " + "openssh",
		} {
			if strings.Contains(lower, marker) {
				return fmt.Errorf("unsafe public artifact value %q", typed)
			}
		}
	}
	return nil
}

func runPulseLifecycle(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "inspect" {
		fmt.Fprintln(stderr, "pulse lifecycle: expected subcommand inspect")
		return 2
	}
	return runPulseLifecycleInspect(args[1:], stdout, stderr)
}

func runPulseLifecycleInspect(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("pulse lifecycle inspect", stderr)
	statePath := fs.String("state", "", "Pulse PR lifecycle state path")
	jsonOut := fs.Bool("json", false, "emit JSON lifecycle state")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	var state PulsePRLifecycle
	if strings.TrimSpace(*statePath) == "" {
		fmt.Fprintln(stderr, "pulse lifecycle: missing --state")
		return 2
	}
	if err := validateEvidencePath(*statePath); err != nil {
		fmt.Fprintf(stderr, "pulse lifecycle: unsafe state path: %v\n", err)
		return 2
	}
	if err := readJSONFile(*statePath, &state); err != nil {
		fmt.Fprintf(stderr, "pulse lifecycle: read state: %v\n", err)
		return 2
	}
	if err := validatePulsePRLifecycle(state); err != nil {
		if *jsonOut {
			_ = writeJSON(stdout, state)
		} else {
			fmt.Fprintf(stdout, "pulse_pr_lifecycle=blocked\n")
			fmt.Fprintf(stdout, "allowed_next_action=%s\n", state.AllowedNextAction)
		}
		fmt.Fprintf(stderr, "pulse lifecycle: %v\n", err)
		return 1
	}
	if *jsonOut {
		if err := writeJSON(stdout, state); err != nil {
			fmt.Fprintf(stderr, "pulse lifecycle: marshal state: %v\n", err)
			return 2
		}
		return 0
	}
	fmt.Fprintln(stdout, "pulse_pr_lifecycle=ready")
	fmt.Fprintf(stdout, "allowed_next_action=%s\n", state.AllowedNextAction)
	return 0
}

func validatePulsePRLifecycle(state PulsePRLifecycle) error {
	if state.SchemaVersion != pulseLifecycleSchema {
		return fmt.Errorf("schema_version must be %s", pulseLifecycleSchema)
	}
	if strings.TrimSpace(state.CurrentSlice) == "" || strings.TrimSpace(state.TargetRepo) == "" || strings.TrimSpace(state.Branch) == "" {
		return errors.New("current_slice, target_repo, and branch are required")
	}
	if err := validateAtlasPublicString(state.CurrentSlice); err != nil {
		return fmt.Errorf("current_slice: %w", err)
	}
	if err := validateAtlasPublicString(state.TargetRepo); err != nil {
		return fmt.Errorf("target_repo: %w", err)
	}
	if err := validateAtlasPublicString(state.Branch); err != nil {
		return fmt.Errorf("branch: %w", err)
	}
	switch state.PRState {
	case "none", "open", "merged", "closed":
	default:
		return fmt.Errorf("invalid pr_state %q", state.PRState)
	}
	switch state.CheckState {
	case "none", "pending", "passing", "failing":
	default:
		return fmt.Errorf("invalid check_state %q", state.CheckState)
	}
	switch state.MergeState {
	case "not_started", "unmerged", "merged":
	default:
		return fmt.Errorf("invalid merge_state %q", state.MergeState)
	}
	switch state.CleanupState {
	case "clean", "local_branch_exists", "remote_branch_exists", "dirty_worktree", "main_not_synced", "multiple_active_branches":
	default:
		return fmt.Errorf("invalid cleanup_state %q", state.CleanupState)
	}
	switch state.AllowedNextAction {
	case "start_next_slice", "wait_for_checks", "fix_failed_checks", "cleanup_branch", "sync_main", "stop_blocked":
	default:
		return fmt.Errorf("invalid allowed_next_action %q", state.AllowedNextAction)
	}
	if state.PRURL != "" && errContainsUnsafeLifecycleURL(state.PRURL) {
		return fmt.Errorf("unsafe pr_url %q", state.PRURL)
	}
	if state.CheckState == "pending" {
		return errors.New("current slice PR checks are pending")
	}
	if state.CheckState == "failing" {
		return errors.New("current slice PR checks are failing")
	}
	if state.PRState == "merged" && state.CleanupState != "clean" {
		return errors.New("merged PR cleanup is incomplete")
	}
	switch state.CleanupState {
	case "main_not_synced":
		return errors.New("pulse lifecycle main is not synced with origin/main")
	case "dirty_worktree":
		return errors.New("pulse lifecycle dirty worktree blocks starting a new slice")
	case "multiple_active_branches":
		return errors.New("pulse lifecycle has multiple active codex branches")
	case "local_branch_exists", "remote_branch_exists":
		return errors.New("merged PR cleanup is incomplete")
	}
	if state.Branch != "main" && (state.PRState == "open" || state.MergeState != "merged") {
		return errors.New("current branch is not main and has an open or unmerged PR")
	}
	if state.AllowedNextAction != "start_next_slice" {
		if strings.TrimSpace(state.BlockerReason) == "" {
			return errors.New("blocked lifecycle state requires blocker_reason")
		}
		return errors.New(state.BlockerReason)
	}
	if state.Branch != "main" || state.PRState != "none" || state.CleanupState != "clean" {
		return errors.New("start_next_slice requires clean synced main with no active PR")
	}
	if strings.TrimSpace(state.BlockerReason) != "" {
		return errors.New("ready lifecycle state must not include blocker_reason")
	}
	return nil
}

func errContainsUnsafeLifecycleURL(value string) bool {
	lower := strings.ToLower(strings.ReplaceAll(value, "\\", "/"))
	return strings.Contains(lower, "/"+"users/") ||
		strings.Contains(lower, "/"+"home/") ||
		strings.Contains(lower, "/"+"tmp/") ||
		strings.Contains(lower, "api"+"_key") ||
		strings.Contains(lower, "access"+"_"+"to"+"ken")
}

func runPulseOvernightStartGate(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("pulse overnight-start-gate", stderr)
	preflightPath := fs.String("intake-preflight", "", "Pulse intake preflight result")
	lifecyclePath := fs.String("lifecycle", "", "Pulse PR lifecycle state")
	outPath := fs.String("out", "", "overnight start gate result output path")
	startImplementation := fs.Bool("start-implementation", false, "fail if a blocked preflight would be used to start implementation")
	jsonOut := fs.Bool("json", false, "emit JSON result to stdout")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	if strings.TrimSpace(*preflightPath) == "" || strings.TrimSpace(*lifecyclePath) == "" || strings.TrimSpace(*outPath) == "" {
		fmt.Fprintln(stderr, "pulse overnight-start-gate: --intake-preflight, --lifecycle, and --out are required")
		return 2
	}
	if sameCleanPath(*outPath, *preflightPath) || sameCleanPath(*outPath, *lifecyclePath) {
		fmt.Fprintln(stderr, "pulse overnight-start-gate: --out must not overwrite input artifacts")
		return 2
	}
	result, err := buildPulseOvernightStartGate(*preflightPath, *lifecyclePath, *startImplementation)
	if writeErr := writeJSONFile(*outPath, result); writeErr != nil {
		fmt.Fprintf(stderr, "pulse overnight-start-gate: write result: %v\n", writeErr)
		return 2
	}
	if *jsonOut {
		if writeErr := writeJSON(stdout, result); writeErr != nil {
			fmt.Fprintf(stderr, "pulse overnight-start-gate: marshal result: %v\n", writeErr)
			return 2
		}
	} else {
		fmt.Fprintf(stdout, "pulse_overnight_start_gate=%s\n", result.Status)
		fmt.Fprintf(stdout, "allowed_next_action=%s\n", result.AllowedNextAction)
		fmt.Fprintf(stdout, "start_gate_result=%s\n", *outPath)
	}
	if err != nil {
		fmt.Fprintf(stderr, "pulse overnight-start-gate: %v\n", err)
		return 1
	}
	if result.Status == "failed" {
		return 1
	}
	return 0
}

func buildPulseOvernightStartGate(preflightPath, lifecyclePath string, startImplementation bool) (PulseOvernightStartGate, error) {
	result := PulseOvernightStartGate{
		SchemaVersion:       pulseStartGateSchema,
		Status:              "ready",
		AllowedNextAction:   "start_next_slice",
		BlockingNextActions: []string{},
		MaintenanceSuggestions: []string{
			"run this gate before autonomous overnight/event-loop advancement",
			"the gate only decides readiness; it must not schedule, execute, approve, upload, publish, or mutate repositories",
		},
		SourceHashes: []PulseStartGateSource{},
	}
	preflight, err := loadPulseStartGatePreflight(preflightPath)
	if err != nil {
		return failPulseStartGate(result, "intake_preflight", "failed", err.Error(), "Regenerate the Pulse intake preflight artifact.")
	}
	preflightSource, err := pulseStartGateSourceFromFile("intake_preflight", preflightPath, preflight.SchemaVersion)
	if err != nil {
		return failPulseStartGate(result, "intake_preflight", "failed", err.Error(), "Use a public-safe Pulse intake preflight path.")
	}
	result.SourceHashes = append(result.SourceHashes, preflightSource)

	lifecycle, err := loadPulseStartGateLifecycle(lifecyclePath)
	if err != nil {
		return failPulseStartGate(result, "pulse_pr_lifecycle", "failed", err.Error(), "Regenerate the Pulse PR lifecycle state artifact.")
	}
	lifecycleSource, err := pulseStartGateSourceFromFile("pulse_pr_lifecycle", lifecyclePath, lifecycle.SchemaVersion)
	if err != nil {
		return failPulseStartGate(result, "pulse_pr_lifecycle", "failed", err.Error(), "Use a public-safe Pulse PR lifecycle path.")
	}
	result.SourceHashes = append(result.SourceHashes, lifecycleSource)

	if err := validatePulsePRLifecycle(lifecycle); err != nil {
		return failPulseStartGate(result, "pulse_pr_lifecycle", "failed", err.Error(), "Finish PR checks, merge, sync main, and clean branches before starting another slice.")
	}
	switch preflight.Status {
	case "ready":
		if err := validatePulseStartGateSourceArtifacts(preflight.SourceArtifacts); err != nil {
			return failPulseStartGate(result, "evidence_digest", "failed", err.Error(), "Regenerate digest-bound Blueprint and Atlas evidence before starting the loop.")
		}
		if preflight.BlueprintStatus != "ready" {
			return failPulseStartGate(result, "intake_preflight", "failed", "ready preflight requires blueprint_status=ready", "Regenerate the Pulse intake preflight artifact.")
		}
		return result, nil
	case "blocked":
		if err := validatePulseStartGateSourceArtifacts(preflight.SourceArtifacts); err != nil {
			return failPulseStartGate(result, "evidence_digest", "failed", err.Error(), "Regenerate digest-bound Blueprint and Atlas evidence before starting the loop.")
		}
		if startImplementation {
			return failPulseStartGate(result, "blueprint_blocked_start_attempt", "failed", "Blueprint clarification is blocked but caller attempted to start implementation", "Answer the Blueprint clarification request before starting implementation.")
		}
		result.Status = "blocked"
		result.AllowedNextAction = "request_blueprint_clarification"
		result.FirstFailingCheck = "intake_preflight"
		result.BlockingNextActions = append(result.BlockingNextActions, preflight.BlockingNextActions...)
		if len(result.BlockingNextActions) == 0 {
			result.BlockingNextActions = append(result.BlockingNextActions, "Answer the Blueprint clarification request before starting implementation.")
		}
		return result, nil
	case "failed":
		return failPulseStartGate(result, "intake_preflight", "failed", "Pulse intake preflight failed", "Fix the Pulse intake preflight failure before starting the loop.")
	default:
		return failPulseStartGate(result, "intake_preflight", "failed", fmt.Sprintf("invalid preflight status %q", preflight.Status), "Regenerate the Pulse intake preflight artifact.")
	}
}

func loadPulseStartGatePreflight(path string) (PulseIntakePreflight, error) {
	if err := validateEvidencePath(path); err != nil {
		return PulseIntakePreflight{}, fmt.Errorf("unsafe intake preflight path: %w", err)
	}
	var preflight PulseIntakePreflight
	if err := readJSONFile(path, &preflight); err != nil {
		return PulseIntakePreflight{}, fmt.Errorf("read intake preflight: %w", err)
	}
	if preflight.SchemaVersion != pulseIntakeSchema {
		return PulseIntakePreflight{}, fmt.Errorf("intake preflight schema_version must be %s", pulseIntakeSchema)
	}
	if err := validatePublicSafeJSONStrings(preflight); err != nil {
		return PulseIntakePreflight{}, err
	}
	if strings.TrimSpace(preflight.Status) == "" {
		return PulseIntakePreflight{}, errors.New("intake preflight status is required")
	}
	return preflight, nil
}

func loadPulseStartGateLifecycle(path string) (PulsePRLifecycle, error) {
	if err := validateEvidencePath(path); err != nil {
		return PulsePRLifecycle{}, fmt.Errorf("unsafe lifecycle path: %w", err)
	}
	var lifecycle PulsePRLifecycle
	if err := readJSONFile(path, &lifecycle); err != nil {
		return PulsePRLifecycle{}, fmt.Errorf("read lifecycle: %w", err)
	}
	if err := validatePublicSafeJSONStrings(lifecycle); err != nil {
		return PulsePRLifecycle{}, err
	}
	return lifecycle, nil
}

func pulseStartGateSourceFromFile(name, path, schemaVersion string) (PulseStartGateSource, error) {
	if err := validateEvidencePath(path); err != nil {
		return PulseStartGateSource{}, fmt.Errorf("unsafe source path: %w", err)
	}
	sum, err := fileSHA256(path)
	if err != nil {
		return PulseStartGateSource{}, err
	}
	return PulseStartGateSource{
		Name:          name,
		Path:          filepath.ToSlash(filepath.Clean(path)),
		SchemaVersion: schemaVersion,
		SHA256:        sum,
	}, nil
}

func validatePulseStartGateSourceArtifacts(sources []PulseIntakeSource) error {
	if len(sources) == 0 {
		return errors.New("Pulse intake preflight must include digest-bound source_artifacts")
	}
	for _, source := range sources {
		if strings.TrimSpace(source.Name) == "" || strings.TrimSpace(source.Path) == "" || strings.TrimSpace(source.SchemaVersion) == "" || strings.TrimSpace(source.Status) == "" {
			return errors.New("Pulse intake source artifacts require name, path, schema_version, and status")
		}
		if err := validateEvidencePath(source.Path); err != nil {
			return fmt.Errorf("source artifact %s has unsafe path: %w", source.Name, err)
		}
		if !isHexSHA256(source.SHA256) {
			return fmt.Errorf("source artifact %s has missing or invalid sha256 digest", source.Name)
		}
		actual, err := fileSHA256(source.Path)
		if err != nil {
			return fmt.Errorf("source artifact %s cannot be hashed: %w", source.Name, err)
		}
		if actual != source.SHA256 {
			return fmt.Errorf("source artifact %s digest is stale", source.Name)
		}
	}
	return nil
}

func isHexSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}

func failPulseStartGate(result PulseOvernightStartGate, checkName, status, reason, nextAction string) (PulseOvernightStartGate, error) {
	result.Status = status
	result.AllowedNextAction = "stop_blocked"
	result.FirstFailingCheck = checkName
	result.BlockingNextActions = append(result.BlockingNextActions, nextAction)
	return result, errors.New(reason)
}

func runPulseEventLoopPolicy(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("pulse event-loop-policy", stderr)
	classGatePath := fs.String("class-gate", "", "Foundry mutation class gate output")
	promotionStatePath := fs.String("promotion-state", "", "proven mutation-class promotion state evidence")
	ciPath := fs.String("ci", "", "CI readiness evidence")
	repoStatePath := fs.String("repo-state", "", "repo cleanliness evidence")
	evidenceFreshnessPath := fs.String("evidence-freshness", "", "evidence freshness evidence")
	sentinelPath := fs.String("sentinel", "", "Sentinel mutation-class hold verdict")
	promoterPath := fs.String("promoter", "", "Promoter mutation-class readiness evidence")
	rollbackPath := fs.String("rollback", "", "rollback integrity evidence")
	branchCleanupPath := fs.String("branch-cleanup", "", "branch cleanup evidence")
	scopePath := fs.String("scope", "", "scope boundary evidence")
	outPath := fs.String("out", "", "event-loop policy output path")
	jsonOut := fs.Bool("json", false, "emit JSON result to stdout")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	inputs := []string{*classGatePath, *promotionStatePath, *ciPath, *repoStatePath, *evidenceFreshnessPath, *sentinelPath, *promoterPath, *rollbackPath, *branchCleanupPath, *scopePath}
	for _, input := range inputs {
		if strings.TrimSpace(input) == "" {
			fmt.Fprintln(stderr, "pulse event-loop-policy: --class-gate, --promotion-state, --ci, --repo-state, --evidence-freshness, --sentinel, --promoter, --rollback, --branch-cleanup, --scope, and --out are required")
			return 2
		}
	}
	if strings.TrimSpace(*outPath) == "" {
		fmt.Fprintln(stderr, "pulse event-loop-policy: --class-gate, --promotion-state, --ci, --repo-state, --evidence-freshness, --sentinel, --promoter, --rollback, --branch-cleanup, --scope, and --out are required")
		return 2
	}
	for _, input := range inputs {
		if sameCleanPath(*outPath, input) {
			fmt.Fprintln(stderr, "pulse event-loop-policy: --out must not overwrite input artifacts")
			return 2
		}
	}
	result, err := buildPulseEventLoopPolicy(pulseEventLoopPolicyPaths{
		ClassGate:         *classGatePath,
		PromotionState:    *promotionStatePath,
		CI:                *ciPath,
		RepoState:         *repoStatePath,
		EvidenceFreshness: *evidenceFreshnessPath,
		Sentinel:          *sentinelPath,
		Promoter:          *promoterPath,
		Rollback:          *rollbackPath,
		BranchCleanup:     *branchCleanupPath,
		Scope:             *scopePath,
	})
	if writeErr := writeJSONFile(*outPath, result); writeErr != nil {
		fmt.Fprintf(stderr, "pulse event-loop-policy: write result: %v\n", writeErr)
		return 2
	}
	if *jsonOut {
		if writeErr := writeJSON(stdout, result); writeErr != nil {
			fmt.Fprintf(stderr, "pulse event-loop-policy: marshal result: %v\n", writeErr)
			return 2
		}
	} else {
		fmt.Fprintf(stdout, "pulse_event_loop_policy=%s\n", result.Status)
		fmt.Fprintf(stdout, "allowed_next_action=%s\n", result.AllowedNextAction)
		fmt.Fprintf(stdout, "safe_to_continue=%t\n", result.SafeToContinue)
		fmt.Fprintf(stdout, "policy_result=%s\n", *outPath)
	}
	if err != nil {
		fmt.Fprintf(stderr, "pulse event-loop-policy: %v\n", err)
		return 1
	}
	if result.Status != "ready" {
		return 1
	}
	return 0
}

type pulseEventLoopPolicyPaths struct {
	ClassGate         string
	PromotionState    string
	CI                string
	RepoState         string
	EvidenceFreshness string
	Sentinel          string
	Promoter          string
	Rollback          string
	BranchCleanup     string
	Scope             string
}

type pulseEventLoopPolicyCheck struct {
	Name          string
	Path          string
	SchemaVersion string
	StatusField   string
	ReadyStatuses []string
}

func buildPulseEventLoopPolicy(paths pulseEventLoopPolicyPaths) (PulseEventLoopPolicy, error) {
	requiredChecks := []string{
		"class_gate",
		"promotion_state",
		"ci_status",
		"repo_cleanliness",
		"evidence_freshness",
		"sentinel_hold",
		"promoter_readiness",
		"rollback_integrity",
		"branch_cleanup",
		"scope_boundary",
	}
	result := PulseEventLoopPolicy{
		SchemaVersion:          pulseLoopPolicySchema,
		Status:                 "ready",
		AllowedNextAction:      "continue_next_slice",
		SafeToContinue:         true,
		OperatorPromptRequired: false,
		RequiredChecks:         requiredChecks,
		SourceEvidence:         []PulseEventLoopPolicySource{},
		BlockingNextActions:    []string{},
		AuthorityBoundary:      "policy_approved_class_only",
		SchedulesWork:          false,
		ExecutesWork:           false,
		ApprovesWork:           false,
		MutatesRepositories:    false,
		CallsProviders:         false,
		OpensPR:                false,
		MergesPR:               false,
	}

	classGate, classGateSource, blocker, err := evaluatePulseEventLoopClassGate(paths.ClassGate)
	if err != nil {
		return failPulseEventLoopPolicy(result, "class_gate", err.Error(), "Regenerate the mutation class gate evidence.")
	}
	result.SourceEvidence = append(result.SourceEvidence, classGateSource)
	result.MutationClass = classGate.MutationClass
	result.SafeToRequest = classGate.SafeToRequest
	result.SafeToExecute = classGate.SafeToExecute
	if blocker != "" {
		return failPulseEventLoopPolicy(result, "class_gate", blocker, "Stop the event loop and rerun the Foundry mutation class gate.")
	}
	if !classGate.SafeToExecute {
		return failPulseEventLoopPolicy(result, "class_gate", "class_gate safe_to_execute is false", "Stop the event loop until the current class has exact safe_to_execute evidence.")
	}

	promotionStateSource, provenLiveClass, approvedMutationClass, blocker, err := evaluatePulseEventLoopPromotionState(paths.PromotionState, result.MutationClass)
	if err != nil {
		return failPulseEventLoopPolicy(result, "promotion_state", err.Error(), "Regenerate the promotion_state evidence.")
	}
	result.SourceEvidence = append(result.SourceEvidence, promotionStateSource)
	result.ProvenLiveClass = provenLiveClass
	result.ApprovedMutationClass = approvedMutationClass
	if blocker != "" {
		return failPulseEventLoopPolicy(result, "promotion_state", blocker, "Stop the event loop until promotion evidence approves exactly the current proven class.")
	}

	checks := []pulseEventLoopPolicyCheck{
		{Name: "ci_status", Path: paths.CI, SchemaVersion: "ao.foundry.ci-readiness.v0.1", StatusField: "status", ReadyStatuses: []string{"passed"}},
		{Name: "repo_cleanliness", Path: paths.RepoState, SchemaVersion: "ao.foundry.repo-cleanliness.v0.1", StatusField: "status", ReadyStatuses: []string{"clean"}},
		{Name: "evidence_freshness", Path: paths.EvidenceFreshness, SchemaVersion: "ao.foundry.evidence-freshness.v0.1", StatusField: "status", ReadyStatuses: []string{"fresh"}},
		{Name: "sentinel_hold", Path: paths.Sentinel, SchemaVersion: "ao.sentinel.mutation-class-hold.v0.1", StatusField: "status", ReadyStatuses: []string{"no_hold"}},
		{Name: "promoter_readiness", Path: paths.Promoter, SchemaVersion: "ao.promoter.mutation-class-promotion.v0.1", StatusField: "status", ReadyStatuses: []string{"ready"}},
		{Name: "rollback_integrity", Path: paths.Rollback, SchemaVersion: "ao.foundry.mutation-class-rollback.v0.1", StatusField: "status", ReadyStatuses: []string{"passed"}},
		{Name: "branch_cleanup", Path: paths.BranchCleanup, SchemaVersion: "ao.foundry.branch-cleanup.v0.1", StatusField: "status", ReadyStatuses: []string{"passed"}},
		{Name: "scope_boundary", Path: paths.Scope, SchemaVersion: "ao.foundry.scope-boundary.v0.1", StatusField: "status", ReadyStatuses: []string{"passed"}},
	}
	for _, check := range checks {
		source, blocker, err := evaluatePulseEventLoopCheck(check, result.MutationClass)
		if err != nil {
			return failPulseEventLoopPolicy(result, check.Name, err.Error(), "Regenerate the "+check.Name+" evidence.")
		}
		result.SourceEvidence = append(result.SourceEvidence, source)
		if blocker != "" {
			return failPulseEventLoopPolicy(result, check.Name, blocker, pulseEventLoopBlockerAction(check.Name))
		}
	}
	return result, nil
}

func evaluatePulseEventLoopClassGate(path string) (MutationClassGate, PulseEventLoopPolicySource, string, error) {
	if err := validateEvidencePath(path); err != nil {
		return MutationClassGate{}, PulseEventLoopPolicySource{}, "", fmt.Errorf("unsafe class gate path: %w", err)
	}
	var gate MutationClassGate
	if err := readJSONFile(path, &gate); err != nil {
		return gate, PulseEventLoopPolicySource{}, "", fmt.Errorf("read class gate: %w", err)
	}
	if err := validatePublicSafeJSONStrings(gate); err != nil {
		return gate, PulseEventLoopPolicySource{}, "", err
	}
	source, err := pulseEventLoopPolicySourceFromFile("class_gate", path, gate.SchemaVersion, gate.Status)
	if err != nil {
		return gate, source, "", err
	}
	switch {
	case gate.SchemaVersion != classGateSchema:
		return gate, source, "class_gate schema_version must be " + classGateSchema, nil
	case !isKnownMutationClass(gate.MutationClass):
		return gate, source, "class_gate missing known mutation_class", nil
	case gate.Status != "ready":
		return gate, source, "class_gate status is " + gate.Status, nil
	case !gate.SafeToRequest:
		return gate, source, "class_gate safe_to_request is false", nil
	case gate.SchedulesWork || gate.ExecutesWork || gate.ApprovesWork || gate.MutatesRepositories:
		return gate, source, "class_gate must not schedule, execute, approve, or mutate repositories", nil
	default:
		return gate, source, "", nil
	}
}

func evaluatePulseEventLoopPromotionState(path, mutationClass string) (PulseEventLoopPolicySource, string, string, string, error) {
	if err := validateEvidencePath(path); err != nil {
		return PulseEventLoopPolicySource{}, "", "", "", fmt.Errorf("unsafe promotion_state path: %w", err)
	}
	document, err := readArbitraryJSON(path)
	if err != nil {
		return PulseEventLoopPolicySource{}, "", "", "", fmt.Errorf("read promotion_state evidence: %w", err)
	}
	object, ok := document.(map[string]any)
	if !ok {
		return PulseEventLoopPolicySource{}, "", "", "", fmt.Errorf("promotion_state evidence must be a JSON object")
	}
	if err := validatePublicSafeJSONStrings(object); err != nil {
		return PulseEventLoopPolicySource{}, "", "", "", err
	}
	status := classGateString(object, "status")
	source, err := pulseEventLoopPolicySourceFromFile("promotion_state", path, classGateString(object, "schema_version"), status)
	if err != nil {
		return source, "", "", "", err
	}
	provenLiveClass := classGateString(object, "proven_live_class")
	approvedMutationClass := classGateString(object, "approved_mutation_class")
	switch {
	case source.SchemaVersion != "ao.foundry.mutation-class-promotion-state.v0.1":
		return source, provenLiveClass, approvedMutationClass, "promotion_state schema_version must be ao.foundry.mutation-class-promotion-state.v0.1", nil
	case status != "ready":
		return source, provenLiveClass, approvedMutationClass, "promotion_state status is " + status, nil
	case classGateString(object, "mutation_class") != mutationClass:
		return source, provenLiveClass, approvedMutationClass, fmt.Sprintf("promotion_state mutation_class %s does not match %s", classGateString(object, "mutation_class"), mutationClass), nil
	case provenLiveClass == "":
		return source, provenLiveClass, approvedMutationClass, "promotion_state missing proven_live_class", nil
	case approvedMutationClass == "":
		return source, provenLiveClass, approvedMutationClass, "promotion_state missing approved_mutation_class", nil
	case provenLiveClass != mutationClass:
		return source, provenLiveClass, approvedMutationClass, fmt.Sprintf("promotion_state proven_live_class %s does not match %s", provenLiveClass, mutationClass), nil
	case approvedMutationClass != mutationClass:
		return source, provenLiveClass, approvedMutationClass, fmt.Sprintf("promotion_state approved_mutation_class %s does not match %s", approvedMutationClass, mutationClass), nil
	case classGateBool(object, "class_jump_requested"):
		return source, provenLiveClass, approvedMutationClass, "promotion_state class_jump_requested is true", nil
	case !classGateBool(object, "live_evidence_merged"):
		return source, provenLiveClass, approvedMutationClass, "promotion_state live_evidence_merged is false", nil
	case !classGateBool(object, "promotion_evidence_present"):
		return source, provenLiveClass, approvedMutationClass, "promotion_state promotion_evidence_present is false", nil
	case classGateBool(object, "operator_prompt_required"):
		return source, provenLiveClass, approvedMutationClass, "promotion_state operator_prompt_required is true", nil
	default:
		return source, provenLiveClass, approvedMutationClass, "", nil
	}
}

func evaluatePulseEventLoopCheck(check pulseEventLoopPolicyCheck, mutationClass string) (PulseEventLoopPolicySource, string, error) {
	if err := validateEvidencePath(check.Path); err != nil {
		return PulseEventLoopPolicySource{}, "", fmt.Errorf("unsafe %s path: %w", check.Name, err)
	}
	document, err := readArbitraryJSON(check.Path)
	if err != nil {
		return PulseEventLoopPolicySource{}, "", fmt.Errorf("read %s evidence: %w", check.Name, err)
	}
	object, ok := document.(map[string]any)
	if !ok {
		return PulseEventLoopPolicySource{}, "", fmt.Errorf("%s evidence must be a JSON object", check.Name)
	}
	if err := validatePublicSafeJSONStrings(object); err != nil {
		return PulseEventLoopPolicySource{}, "", err
	}
	status := classGateString(object, check.StatusField)
	source, err := pulseEventLoopPolicySourceFromFile(check.Name, check.Path, classGateString(object, "schema_version"), status)
	if err != nil {
		return source, "", err
	}
	if source.SchemaVersion != check.SchemaVersion {
		return source, fmt.Sprintf("%s schema_version must be %s", check.Name, check.SchemaVersion), nil
	}
	if !classGateStringSliceContains(check.ReadyStatuses, status) {
		return source, fmt.Sprintf("%s status is %s", check.Name, status), nil
	}
	documentClass := classGateString(object, "mutation_class")
	if documentClass == "" {
		return source, check.Name + " missing mutation_class", nil
	}
	if mutationClass != "" && documentClass != mutationClass {
		return source, fmt.Sprintf("%s mutation_class %s does not match %s", check.Name, documentClass, mutationClass), nil
	}
	switch check.Name {
	case "repo_cleanliness":
		switch {
		case classGateBool(object, "dirty_repo"):
			return source, "repo_cleanliness dirty_repo is true", nil
		case !classGateBool(object, "main_synced"):
			return source, "repo_cleanliness main_synced is false", nil
		case classGateBool(object, "stale_codex_branches"):
			return source, "repo_cleanliness stale_codex_branches is true", nil
		}
	case "evidence_freshness":
		if classGateBool(object, "stale_evidence") {
			return source, "evidence_freshness stale_evidence is true", nil
		}
	case "rollback_integrity":
		switch {
		case classGateBool(object, "rollback_failure"):
			return source, "rollback_integrity rollback_failure is true", nil
		case !classGateBool(object, "rollback_verified"):
			return source, "rollback_integrity rollback_verified is false", nil
		}
	case "branch_cleanup":
		switch {
		case classGateBool(object, "stale_codex_branches"):
			return source, "branch_cleanup stale_codex_branches is true", nil
		case !classGateBool(object, "local_branch_deleted"):
			return source, "branch_cleanup local_branch_deleted is false", nil
		case !classGateBool(object, "remote_branch_deleted"):
			return source, "branch_cleanup remote_branch_deleted is false", nil
		}
	case "scope_boundary":
		if classGateBool(object, "broadened_scope") {
			return source, "scope_boundary broadened_scope is true", nil
		}
	}
	return source, "", nil
}

func pulseEventLoopPolicySourceFromFile(name, path, schemaVersion, status string) (PulseEventLoopPolicySource, error) {
	sum, err := fileSHA256(path)
	if err != nil {
		return PulseEventLoopPolicySource{}, err
	}
	return PulseEventLoopPolicySource{
		Name:          name,
		Path:          filepath.ToSlash(filepath.Clean(path)),
		SchemaVersion: schemaVersion,
		Status:        status,
		SHA256:        sum,
	}, nil
}

func failPulseEventLoopPolicy(result PulseEventLoopPolicy, checkName, reason, nextAction string) (PulseEventLoopPolicy, error) {
	result.Status = "blocked"
	result.AllowedNextAction = "stop_event_loop"
	result.SafeToContinue = false
	result.SafeToRequest = false
	result.SafeToExecute = false
	result.FirstFailingCheck = checkName
	result.BlockingNextActions = append(result.BlockingNextActions, nextAction)
	return result, errors.New(reason)
}

func pulseEventLoopBlockerAction(checkName string) string {
	switch checkName {
	case "ci_status":
		return "Stop the event loop until CI passes."
	case "repo_cleanliness":
		return "Stop the event loop until every touched repo is clean and synced with origin/main."
	case "evidence_freshness":
		return "Stop the event loop until stale evidence is regenerated."
	case "sentinel_hold":
		return "Stop the event loop until Sentinel reports no active hold."
	case "promoter_readiness":
		return "Stop the event loop until Promoter reports class promotion readiness."
	case "rollback_integrity":
		return "Stop the event loop until rollback evidence passes."
	case "branch_cleanup":
		return "Stop the event loop until local and remote codex branches are cleaned up."
	case "scope_boundary":
		return "Stop the event loop because the requested scope broadened."
	default:
		return "Stop the event loop and regenerate required evidence."
	}
}

func isKnownMutationClass(className string) bool {
	return classGateStringSliceContains([]string{
		"docs_only_single_file",
		"docs_only_multi_file",
		"docs_config_only",
		"test_only",
		"low_risk_code",
		"multi_repo_low_risk",
		"complex_repo_mutation",
	}, className)
}

func buildPulseRunnerStartDecision(startGatePath string) (PulseRunnerStartDecision, error) {
	decision := PulseRunnerStartDecision{
		SchemaVersion:       pulseRunnerSchema,
		Status:              "failed",
		StartGatePath:       filepath.ToSlash(filepath.Clean(startGatePath)),
		AllowedNextAction:   "stop_blocked",
		FirstFailingCheck:   "pulse_overnight_start_gate",
		BlockingNextActions: []string{"Provide a ready Pulse overnight start gate before running the event loop."},
		SourceDigests:       []PulseStartGateSource{},
	}
	if strings.TrimSpace(startGatePath) == "" {
		return decision, errors.New("Pulse runner start gate is required")
	}
	if err := validateEvidencePath(startGatePath); err != nil {
		return decision, fmt.Errorf("unsafe Pulse runner start gate path: %w", err)
	}
	var startGate PulseOvernightStartGate
	if err := readJSONFile(startGatePath, &startGate); err != nil {
		return decision, fmt.Errorf("read Pulse runner start gate: %w", err)
	}
	if startGate.SchemaVersion != pulseStartGateSchema {
		return decision, fmt.Errorf("Pulse runner start gate schema_version must be %s", pulseStartGateSchema)
	}
	if err := validatePublicSafeJSONStrings(startGate); err != nil {
		return decision, err
	}
	source, err := pulseStartGateSourceFromFile("pulse_overnight_start_gate", startGatePath, startGate.SchemaVersion)
	if err != nil {
		return decision, err
	}
	decision.SourceDigests = append(decision.SourceDigests, source)
	if len(startGate.SourceHashes) == 0 {
		return decision, errors.New("Pulse runner start gate lacks digest-bound source hashes")
	}
	for _, sourceHash := range startGate.SourceHashes {
		if strings.TrimSpace(sourceHash.Name) == "" || strings.TrimSpace(sourceHash.Path) == "" || strings.TrimSpace(sourceHash.SchemaVersion) == "" {
			return decision, errors.New("Pulse runner start gate source hashes require name, path, and schema_version")
		}
		if !isHexSHA256(sourceHash.SHA256) {
			return decision, fmt.Errorf("Pulse runner start gate source %s has missing or invalid sha256 digest", sourceHash.Name)
		}
		if err := validateEvidencePath(sourceHash.Path); err != nil {
			return decision, fmt.Errorf("Pulse runner start gate source %s has unsafe path: %w", sourceHash.Name, err)
		}
		decision.SourceDigests = append(decision.SourceDigests, sourceHash)
	}
	decision.Status = startGate.Status
	decision.AllowedNextAction = startGate.AllowedNextAction
	decision.FirstFailingCheck = startGate.FirstFailingCheck
	decision.BlockingNextActions = append([]string{}, startGate.BlockingNextActions...)
	if decision.Status != "ready" {
		if decision.FirstFailingCheck == "" {
			decision.FirstFailingCheck = "pulse_overnight_start_gate"
		}
		if len(decision.BlockingNextActions) == 0 {
			decision.BlockingNextActions = append(decision.BlockingNextActions, "Regenerate the Pulse overnight start gate with ready status before running the event loop.")
		}
		return decision, fmt.Errorf("Pulse runner start gate is %s", decision.Status)
	}
	if decision.AllowedNextAction != "start_next_slice" {
		decision.Status = "blocked"
		decision.FirstFailingCheck = "pulse_overnight_start_gate"
		decision.BlockingNextActions = append(decision.BlockingNextActions, "Pulse runner start gate must allow start_next_slice.")
		return decision, errors.New("Pulse runner start gate does not allow start_next_slice")
	}
	return decision, nil
}

func runPulseSignedSmokeScript(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("pulse signed-smoke-script", stderr)
	outPath := fs.String("out", "", "signed control-plane smoke shell script output path")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	if *outPath == "" {
		fmt.Fprintln(stderr, "pulse: missing --out")
		return 2
	}
	if err := writeSignedSmokeScript(*outPath); err != nil {
		fmt.Fprintf(stderr, "pulse: write signed smoke script: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "signed_smoke_script=%s\n", *outPath)
	return 0
}

func runPulseSignedSmokePreflight(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("pulse signed-smoke-preflight", stderr)
	workspace := fs.String("workspace", "..", "prepared AO workspace root")
	outPath := fs.String("out", "", "signed smoke preflight output path")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	if *outPath == "" {
		fmt.Fprintln(stderr, "pulse: missing --out")
		return 2
	}
	preflight := buildSignedSmokePreflight(*workspace)
	if err := writeJSONFile(*outPath, preflight); err != nil {
		fmt.Fprintf(stderr, "pulse: write signed smoke preflight: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "signed_smoke_preflight=%s\n", *outPath)
	fmt.Fprintf(stdout, "status=%s\n", preflight.Status)
	if preflight.Status != "ready" {
		fmt.Fprintln(stderr, "pulse: signed smoke preflight blocked")
		return 1
	}
	return 0
}

func runPulseSignedSmokeCleanup(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("pulse signed-smoke-cleanup", stderr)
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	removed, err := cleanupSignedSmokeScratch()
	if err != nil {
		fmt.Fprintf(stderr, "pulse: signed smoke cleanup: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "removed=%d\n", removed)
	return 0
}

func runPulseIngestSignedSmoke(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("pulse ingest-signed-smoke", stderr)
	resultPath := fs.String("result", "", "signed-smoke-result.json path")
	outPath := fs.String("out", "", "signed smoke ingest output path")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	if *resultPath == "" || *outPath == "" {
		fmt.Fprintln(stderr, "pulse: missing --result or --out")
		return 2
	}
	ingest, err := buildSignedSmokeIngest(*resultPath)
	if err != nil {
		fmt.Fprintf(stderr, "pulse: ingest signed smoke: %v\n", err)
		return 2
	}
	if err := writeJSONFile(*outPath, ingest); err != nil {
		fmt.Fprintf(stderr, "pulse: write signed smoke ingest: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "signed_smoke_ingest=%s\n", *outPath)
	fmt.Fprintf(stdout, "status=%s\n", ingest.Status)
	return 0
}

func runPulseSummarizeSignedSmoke(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("pulse summarize-signed-smoke", stderr)
	pulsePath := fs.String("pulse", "", "pulse-event.json path")
	outPath := fs.String("out", "", "public-safe signed smoke summary output path")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	if *pulsePath == "" || *outPath == "" {
		fmt.Fprintln(stderr, "pulse: missing --pulse or --out")
		return 2
	}
	summary, err := buildSignedSmokeSummary(*pulsePath)
	if err != nil {
		fmt.Fprintf(stderr, "pulse: summarize signed smoke: %v\n", err)
		return 2
	}
	if err := writeJSONFile(*outPath, summary); err != nil {
		fmt.Fprintf(stderr, "pulse: write signed smoke summary: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "signed_smoke_summary=%s\n", *outPath)
	fmt.Fprintf(stdout, "status=%s\n", summary.Status)
	return 0
}

func runPulseDecision(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("pulse decision", stderr)
	action := fs.String("action", "stop", "AO2 event-loop action")
	reason := fs.String("reason", "", "AO2 event-loop decision reason")
	nextTaskID := fs.String("next-task-id", "", "next recommended task id")
	outPath := fs.String("out", "", "AO2 event-loop decision output path")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	decision, err := buildAO2LoopDecision(*action, *reason, *nextTaskID)
	if err != nil {
		fmt.Fprintf(stderr, "pulse: decision: %v\n", err)
		return 2
	}
	if *outPath == "" {
		fmt.Fprintln(stderr, "pulse: missing --out")
		return 2
	}
	if err := writeJSONFile(*outPath, decision); err != nil {
		fmt.Fprintf(stderr, "pulse: write decision: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "ao2_loop_decision=%s\n", *outPath)
	fmt.Fprintf(stdout, "next_task_id=%s\n", decision.EventLoop.NextTaskID)
	return 0
}

func runPulseDeriveNext(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("pulse derive-next", stderr)
	pulsePath := fs.String("pulse", "", "pulse-event.json path")
	auditPath := fs.String("audit", "", "optional competitive audit JSON path")
	outPath := fs.String("out", "", "AO2 event-loop decision output path")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	if *pulsePath == "" || *outPath == "" {
		fmt.Fprintln(stderr, "pulse: missing --pulse or --out")
		return 2
	}
	decision, err := buildDerivedAO2LoopDecision(*pulsePath, *auditPath)
	if err != nil {
		fmt.Fprintf(stderr, "pulse: derive next: %v\n", err)
		return 2
	}
	if err := writeJSONFile(*outPath, decision); err != nil {
		fmt.Fprintf(stderr, "pulse: write derived decision: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "ao2_loop_decision=%s\n", *outPath)
	fmt.Fprintf(stdout, "next_task_id=%s\n", decision.EventLoop.NextTaskID)
	return 0
}

func runPulseFreshness(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("pulse freshness", stderr)
	pulsePath := fs.String("pulse", "", "pulse-event.json path")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	if *pulsePath == "" {
		fmt.Fprintln(stderr, "pulse: missing --pulse")
		return 2
	}
	var event PulseEvent
	if err := readJSONFile(*pulsePath, &event); err != nil {
		fmt.Fprintf(stderr, "pulse: freshness: %v\n", err)
		return 2
	}
	if event.SchemaVersion != pulseEventSchema {
		fmt.Fprintf(stderr, "pulse: freshness: unexpected pulse schema %q\n", event.SchemaVersion)
		return 2
	}
	freshness := event.Freshness
	if strings.TrimSpace(freshness.SchemaVersion) == "" {
		freshness = newPulseFreshnessSummary()
	}
	fmt.Fprintf(stdout, "freshness=%s forge_live_packet=%s control_plane_readback=%s\n", freshness.Status, freshness.ForgeLivePacket, freshness.ControlPlaneReadback)
	if freshness.Status != "ready" {
		return 1
	}
	return 0
}

func runPulseRun(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("pulse run", stderr)
	registryPath := fs.String("registry", "examples/registry/local-ao-stack.foundry-registry.json", "registry path")
	taskPath := fs.String("task", "examples/tasks/ao-foundry-bootstrap.foundry-task.json", "task path")
	goalPath := fs.String("goal-run", "examples/goals/ao-foundry-production-readiness.goal-run.json", "goal-run path")
	packetPath := fs.String("packet", "examples/packets/ao-foundry-bootstrap.factory-packet.json", "Forge packet path")
	forgeLivePacketPath := fs.String("forge-live-packet", "", "AO Forge live packet path")
	controlPlaneReceiptPath := fs.String("control-plane-receipt", "", "control-plane readback receipt path")
	signedSmokeResultPath := fs.String("signed-smoke-result", "", "signed smoke result path")
	startGatePath := fs.String("start-gate", "examples/pulse-overnight-start-gate/ready.json", "Pulse overnight start gate result path")
	scorecardPath := fs.String("scorecard", "examples/evals/bootstrap.scorecard.json", "eval scorecard path")
	rsiBaselinePath := fs.String("rsi-baseline", "examples/evals/rsi-baseline.eval-result.json", "RSI baseline eval result path")
	rsiMinImprovement := fs.Float64("rsi-min-improvement", 5, "minimum RSI improvement percentage points")
	outDir := fs.String("out", "tmp/pulse", "pulse bundle output directory")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	decision, decisionErr := buildPulseRunnerStartDecision(*startGatePath)
	decisionPath := filepath.Join(*outDir, "pulse-runner-start-decision.json")
	if writeErr := writeJSONFile(decisionPath, decision); writeErr != nil {
		fmt.Fprintf(stderr, "pulse: write runner start decision: %v\n", writeErr)
		return 2
	}
	fmt.Fprintf(stdout, "pulse_runner_start_decision=%s\n", decisionPath)
	fmt.Fprintf(stdout, "runner_start_status=%s\n", decision.Status)
	if decisionErr != nil {
		fmt.Fprintf(stderr, "pulse: %v\n", decisionErr)
		return 1
	}
	event, err := buildPulseBundle(*registryPath, *taskPath, *goalPath, *packetPath, *scorecardPath, *rsiBaselinePath, *rsiMinImprovement, *outDir, *forgeLivePacketPath, *controlPlaneReceiptPath, *signedSmokeResultPath)
	eventPath := filepath.Join(*outDir, "pulse-event.json")
	if writeErr := writeJSONFile(eventPath, event); writeErr != nil {
		fmt.Fprintf(stderr, "pulse: write event: %v\n", writeErr)
		return 2
	}
	fmt.Fprintf(stdout, "pulse_event=%s\n", eventPath)
	fmt.Fprintf(stdout, "status=%s\n", event.Status)
	fmt.Fprintf(stdout, "score=%d/%d\n", event.Score, event.MaxScore)
	fmt.Fprintf(stdout, "freshness=%s forge_live_packet=%s control_plane_readback=%s\n", event.Freshness.Status, event.Freshness.ForgeLivePacket, event.Freshness.ControlPlaneReadback)
	fmt.Fprintf(stdout, "next_action=%s\n", event.NextAction)
	if err != nil {
		fmt.Fprintf(stderr, "pulse: %v\n", err)
		return 1
	}
	return 0
}

func runAO(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "ao: expected subcommand status, next, run, audit, or demo")
		return 2
	}
	switch args[0] {
	case "status":
		return runStatus(defaultArgs(args[1:], "--registry", "examples/registry/local-ao-stack.foundry-registry.json"), stdout, stderr)
	case "next":
		return runNext(defaultArgs(defaultArgs(args[1:], "--registry", "examples/registry/local-ao-stack.foundry-registry.json"), "--task", "examples/tasks/ao-foundry-bootstrap.foundry-task.json"), stdout, stderr)
	case "run":
		return runPulseRun(args[1:], stdout, stderr)
	case "audit":
		return runCompetitive(append([]string{"audit"}, args[1:]...), stdout, stderr)
	case "demo":
		demoArgs := defaultArgs(args[1:], "--registry", "examples/registry/local-ao-stack.foundry-registry.json")
		demoArgs = defaultArgs(demoArgs, "--task", "examples/tasks/ao-foundry-bootstrap.foundry-task.json")
		demoArgs = defaultArgs(demoArgs, "--run", "examples/runs/ao-foundry-bootstrap.foundry-run.json")
		return runDemo(append([]string{"status"}, demoArgs...), stdout, stderr)
	default:
		fmt.Fprintf(stderr, "ao: unknown subcommand %q\n", args[0])
		return 2
	}
}

func defaultArgs(args []string, flagName, value string) []string {
	for i, arg := range args {
		if arg == flagName {
			return args
		}
		if strings.HasPrefix(arg, flagName+"=") {
			return args
		}
		if i == len(args)-1 && strings.HasPrefix(arg, flagName) {
			return args
		}
	}
	return append(args, flagName, value)
}

func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}

func parseFlags(fs *flag.FlagSet, args []string, stderr io.Writer) bool {
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", fs.Name(), err)
		return false
	}
	return true
}

func loadRegistry(path string) (Registry, error) {
	var registry Registry
	if path == "" {
		return registry, errors.New("missing --registry")
	}
	if err := readJSONFile(path, &registry); err != nil {
		return registry, err
	}
	return registry, validateRegistry(registry)
}

func loadTask(path string) (Task, error) {
	var task Task
	if path == "" {
		return task, errors.New("missing --task")
	}
	if err := readJSONFile(path, &task); err != nil {
		return task, err
	}
	return task, validateTask(task)
}

func loadGoalRun(path string) (GoalRun, error) {
	var goal GoalRun
	if path == "" {
		return goal, errors.New("missing --goal-run")
	}
	if err := readJSONFile(path, &goal); err != nil {
		return goal, err
	}
	return goal, validateGoalRun(goal)
}

func loadFoundryRun(path string) (FoundryRun, error) {
	var run FoundryRun
	if path == "" {
		return run, errors.New("missing --run")
	}
	if err := readJSONFile(path, &run); err != nil {
		return run, err
	}
	return run, validateFoundryRun(run)
}

func loadAtlasFoundryImport(path string) (AtlasFoundryImport, error) {
	var artifact AtlasFoundryImport
	if path == "" {
		return artifact, errors.New("missing --import")
	}
	if err := readJSONFile(path, &artifact); err != nil {
		return artifact, err
	}
	return artifact, validateAtlasFoundryImport(artifact)
}

func loadAtlasBlueprintImport(path string) (AtlasBlueprintImport, error) {
	var artifact AtlasBlueprintImport
	if path == "" {
		return artifact, errors.New("missing --atlas-blueprint-import")
	}
	if err := readJSONFile(path, &artifact); err != nil {
		return artifact, err
	}
	return artifact, validateAtlasBlueprintImport(artifact)
}

func loadAtlasRunLink(path string) (AtlasRunLink, error) {
	var link AtlasRunLink
	if path == "" {
		return link, errors.New("missing --run-link")
	}
	if err := readJSONFile(path, &link); err != nil {
		return link, err
	}
	return link, validateAtlasRunLink(link)
}

func loadActiveStackReadinessLedger(path string) (ActiveStackReadinessLedger, error) {
	var ledger ActiveStackReadinessLedger
	if strings.TrimSpace(path) == "" {
		return ledger, errors.New("missing --ledger")
	}
	if err := readJSONFile(path, &ledger); err != nil {
		return ledger, err
	}
	if ledger.SchemaVersion != "ao.foundry.active-stack-readiness.v0.1" {
		return ledger, errors.New("invalid active stack readiness schema_version")
	}
	if ledger.RegistryID == "" || ledger.GeneratedFromRegistry == "" || ledger.LastSweepDate == "" || ledger.Status == "" {
		return ledger, errors.New("active stack readiness ledger requires registry_id, generated_from_registry, last_sweep_date, and status")
	}
	if len(ledger.Repositories) == 0 {
		return ledger, errors.New("active stack readiness ledger requires repositories")
	}
	if ledger.ReleaseHandoff.Status == "" || len(ledger.ReleaseHandoff.Gates) == 0 {
		return ledger, errors.New("active stack readiness ledger requires release_handoff gates")
	}
	return ledger, nil
}

func loadActiveStackGithubRunsReport(path string) (ActiveStackGithubRunsReport, error) {
	var report ActiveStackGithubRunsReport
	if strings.TrimSpace(path) == "" {
		return report, errors.New("missing --github-runs-report")
	}
	if err := readJSONFile(path, &report); err != nil {
		return report, err
	}
	if report.SchemaVersion != "ao.foundry.active-stack-github-runs-report.v0.1" {
		return report, errors.New("invalid active stack GitHub runs report schema_version")
	}
	if report.Status != "ready" {
		return report, fmt.Errorf("GitHub runs report status must be ready, got %q", report.Status)
	}
	if len(report.Repositories) == 0 {
		return report, errors.New("GitHub runs report requires repositories")
	}
	return report, nil
}

func loadReleaseCandidateLedger(path string) (ReleaseCandidateLedger, error) {
	var ledger ReleaseCandidateLedger
	if strings.TrimSpace(path) == "" {
		return ledger, errors.New("missing --ledger")
	}
	if err := readJSONFile(path, &ledger); err != nil {
		return ledger, err
	}
	return ledger, validateReleaseCandidateLedger(ledger)
}

type ActiveStackGithubEvidenceCheck struct {
	Checked            int
	SkippedCurrentRepo bool
	Problems           []string
}

func checkActiveStackGithubRunEvidence(ledger ActiveStackReadinessLedger, report ActiveStackGithubRunsReport, currentRepo string, checkCurrentRepo bool) ActiveStackGithubEvidenceCheck {
	result := ActiveStackGithubEvidenceCheck{}
	ledgerRepos := map[string]ActiveStackReadinessRepository{}
	for _, repo := range ledger.Repositories {
		ledgerRepos[repo.ID] = repo
	}
	for _, repoReport := range report.Repositories {
		repoID := githubRepositoryID(repoReport.Repository)
		if repoID == "" {
			result.Problems = append(result.Problems, fmt.Sprintf("report repository %q has no repository id", repoReport.Repository))
			continue
		}
		ledgerRepo, ok := ledgerRepos[repoID]
		if !ok {
			result.Problems = append(result.Problems, fmt.Sprintf("%s is not recorded in active stack readiness ledger", repoID))
			continue
		}
		if repoID == currentRepo && !checkCurrentRepo {
			result.SkippedCurrentRepo = true
			continue
		}
		result.Checked++
		result.Problems = append(result.Problems, checkGithubRunEvidence(ledgerRepo, "latest_ci", repoReport.LatestCI)...)
		result.Problems = append(result.Problems, checkGithubRunEvidence(ledgerRepo, "latest_ops", repoReport.LatestOps)...)
	}
	return result
}

func checkGithubRunEvidence(repo ActiveStackReadinessRepository, kind string, run ActiveStackGithubRun) []string {
	var problems []string
	runID := strings.TrimSpace(run.RunID)
	if run.Status != "completed" || run.Conclusion != "success" {
		problems = append(problems, fmt.Sprintf("%s %s is %s/%s", repo.ID, kind, run.Status, run.Conclusion))
	}
	if runID == "" {
		problems = append(problems, fmt.Sprintf("%s %s has no run_id", repo.ID, kind))
		return problems
	}
	if !activeStackRepoEvidenceContainsRun(repo, runID) {
		problems = append(problems, fmt.Sprintf("%s %s run %s is not recorded in readiness ledger evidence", repo.ID, kind, runID))
	}
	return problems
}

func activeStackRepoEvidenceContainsRun(repo ActiveStackReadinessRepository, runID string) bool {
	if repo.CI != nil && repo.CI.RunID == runID {
		return true
	}
	for _, evidence := range repo.VerificationEvidence {
		if strings.Contains(evidence, runID) {
			return true
		}
	}
	return false
}

func githubRepositoryID(repository string) string {
	parts := strings.Split(strings.TrimSpace(repository), "/")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[len(parts)-1])
}

type ActiveStackLedgerRefreshRow struct {
	Repository string
	Workflow   string
	RunID      string
	Action     string
	Run        ActiveStackGithubRun
}

func activeStackLedgerRefreshRows(ledger ActiveStackReadinessLedger, report ActiveStackGithubRunsReport) []ActiveStackLedgerRefreshRow {
	var rows []ActiveStackLedgerRefreshRow
	ledgerRepos := map[string]ActiveStackReadinessRepository{}
	for _, repo := range ledger.Repositories {
		ledgerRepos[repo.ID] = repo
	}
	for _, repoReport := range report.Repositories {
		repoID := githubRepositoryID(repoReport.Repository)
		ledgerRepo, ok := ledgerRepos[repoID]
		if !ok {
			rows = append(rows,
				ledgerRefreshRow(repoID, "ci.yml", repoReport.LatestCI, "missing_repository"),
				ledgerRefreshRow(repoID, "production-readiness-ops.yml", repoReport.LatestOps, "missing_repository"),
			)
			continue
		}
		rows = append(rows,
			classifyLedgerRefreshRow(ledgerRepo, "ci.yml", repoReport.LatestCI),
			classifyLedgerRefreshRow(ledgerRepo, "production-readiness-ops.yml", repoReport.LatestOps),
		)
	}
	return rows
}

func classifyLedgerRefreshRow(repo ActiveStackReadinessRepository, workflow string, run ActiveStackGithubRun) ActiveStackLedgerRefreshRow {
	action := "already_recorded"
	if run.Status != "completed" || run.Conclusion != "success" {
		action = "blocked"
	} else if strings.TrimSpace(run.RunID) == "" || !activeStackRepoEvidenceContainsRun(repo, strings.TrimSpace(run.RunID)) {
		action = "update"
	}
	return ledgerRefreshRow(repo.ID, workflow, run, action)
}

func ledgerRefreshRow(repoID, workflow string, run ActiveStackGithubRun, action string) ActiveStackLedgerRefreshRow {
	return ActiveStackLedgerRefreshRow{
		Repository: repoID,
		Workflow:   workflow,
		RunID:      strings.TrimSpace(run.RunID),
		Action:     action,
		Run:        run,
	}
}

func nonCurrentUpdateProblems(rows []ActiveStackLedgerRefreshRow, currentRepo string) []string {
	var problems []string
	for _, row := range rows {
		if row.Repository == currentRepo {
			continue
		}
		if row.Action == "update" && row.Repository != currentRepo {
			problems = append(problems, fmt.Sprintf("%s %s has update row for run %s", row.Repository, row.Workflow, row.RunID))
		}
		if row.Action == "blocked" || row.Action == "missing_repository" {
			problems = append(problems, fmt.Sprintf("%s %s has %s row", row.Repository, row.Workflow, row.Action))
		}
	}
	return problems
}

func suppressCurrentRepoRefreshLoopRows(rows []ActiveStackLedgerRefreshRow, currentRepo string) []ActiveStackLedgerRefreshRow {
	if !onlyCurrentRepoRowsNeedRefresh(rows, currentRepo) || !currentRepoCIIsReadinessEvidenceRefresh(rows, currentRepo) {
		return rows
	}
	filtered := make([]ActiveStackLedgerRefreshRow, len(rows))
	copy(filtered, rows)
	for i := range filtered {
		if filtered[i].Repository == currentRepo && filtered[i].Action == "update" {
			filtered[i].Action = "ignored_current_refresh_loop"
		}
	}
	return filtered
}

func suppressCurrentRepoMutableEvidenceRows(rows []ActiveStackLedgerRefreshRow, currentRepo string) []ActiveStackLedgerRefreshRow {
	filtered := make([]ActiveStackLedgerRefreshRow, len(rows))
	copy(filtered, rows)
	for i := range filtered {
		if filtered[i].Repository == currentRepo && filtered[i].Action == "update" {
			filtered[i].Action = "ignored_current_self_evidence"
		}
	}
	return filtered
}

func suppressCurrentRepoSelfWindowRows(rows []ActiveStackLedgerRefreshRow, currentRepo string, currentRepoSkipped bool) []ActiveStackLedgerRefreshRow {
	if !currentRepoSkipped {
		return rows
	}
	filtered := make([]ActiveStackLedgerRefreshRow, len(rows))
	copy(filtered, rows)
	for i := range filtered {
		if filtered[i].Repository == currentRepo && filtered[i].Action == "blocked" {
			filtered[i].Action = "ignored_current_self_window"
		}
	}
	return filtered
}

func onlyCurrentRepoRowsNeedRefresh(rows []ActiveStackLedgerRefreshRow, currentRepo string) bool {
	currentNeedsRefresh := false
	for _, row := range rows {
		switch row.Action {
		case "update":
			if row.Repository != currentRepo {
				return false
			}
			currentNeedsRefresh = true
		case "blocked", "missing_repository":
			if row.Repository != currentRepo {
				return false
			}
		}
	}
	return currentNeedsRefresh
}

func currentRepoCIIsReadinessEvidenceRefresh(rows []ActiveStackLedgerRefreshRow, currentRepo string) bool {
	for _, row := range rows {
		if row.Repository != currentRepo || row.Workflow != "ci.yml" {
			continue
		}
		title := strings.ToLower(strings.TrimSpace(row.Run.DisplayName))
		return strings.Contains(title, "refresh") &&
			(strings.Contains(title, "readiness evidence") || strings.Contains(title, "foundry evidence"))
	}
	return false
}

func buildActiveStackProductionReadinessRollup(ledgerPath, reportPath, currentRepo string) (ActiveStackProductionReadinessRollup, error) {
	ledger, err := loadActiveStackReadinessLedger(ledgerPath)
	if err != nil {
		return ActiveStackProductionReadinessRollup{}, err
	}
	report, err := loadActiveStackGithubRunsReport(reportPath)
	if err != nil {
		return ActiveStackProductionReadinessRollup{}, err
	}
	rollup := ActiveStackProductionReadinessRollup{
		SchemaVersion:      "ao.foundry.active-stack-production-readiness-rollup.v0.1",
		Status:             "ready",
		Ledger:             ledgerPath,
		GithubRunsReport:   reportPath,
		ActiveRepositories: len(ledger.Repositories),
		CurrentRepo:        currentRepo,
		CurrentRepoSkipped: report.CurrentRepoSkipped,
	}
	problems := map[string]bool{}
	addProblem := func(problem string) {
		problem = strings.TrimSpace(problem)
		if problem != "" && !problems[problem] {
			problems[problem] = true
			rollup.Problems = append(rollup.Problems, problem)
		}
	}

	reportRepos := map[string]ActiveStackGithubRunsRepository{}
	for _, repoReport := range report.Repositories {
		repoID := githubRepositoryID(repoReport.Repository)
		if repoID != "" {
			reportRepos[repoID] = repoReport
		}
	}
	for _, repo := range ledger.Repositories {
		row := ActiveStackRollupRepository{ID: repo.ID, Status: repo.Status}
		if repoReport, ok := reportRepos[repo.ID]; ok {
			row.LatestCIRunID = strings.TrimSpace(repoReport.LatestCI.RunID)
			row.LatestCIStatus = githubRunRollupStatus(repoReport.LatestCI)
			row.LatestOpsRunID = strings.TrimSpace(repoReport.LatestOps.RunID)
			row.LatestOpsStatus = githubRunRollupStatus(repoReport.LatestOps)
		} else {
			row.Status = "blocked"
			addProblem(fmt.Sprintf("%s is missing from GitHub runs report", repo.ID))
		}
		if repo.Status != "ready" {
			row.Status = "blocked"
			addProblem(fmt.Sprintf("%s readiness ledger status is %s", repo.ID, repo.Status))
		}
		rollup.Repositories = append(rollup.Repositories, row)
	}

	if ledger.Status != "ready" {
		addProblem(fmt.Sprintf("active stack readiness ledger status is %s", ledger.Status))
	}
	evidence := checkActiveStackGithubRunEvidence(ledger, report, currentRepo, false)
	rollup.CurrentRepoSkipped = rollup.CurrentRepoSkipped || evidence.SkippedCurrentRepo
	for _, problem := range evidence.Problems {
		addProblem(problem)
	}
	rows := suppressCurrentRepoRefreshLoopRows(activeStackLedgerRefreshRows(ledger, report), currentRepo)
	rows = suppressCurrentRepoMutableEvidenceRows(rows, currentRepo)
	rows = suppressCurrentRepoSelfWindowRows(rows, currentRepo, report.CurrentRepoSkipped)
	for _, problem := range nonCurrentUpdateProblems(rows, currentRepo) {
		addProblem(problem)
	}
	for _, row := range rows {
		if row.Repository == currentRepo && (row.Action == "blocked" || row.Action == "missing_repository") {
			addProblem(fmt.Sprintf("%s %s has %s row", row.Repository, row.Workflow, row.Action))
		}
		if (row.Action == "update" && row.Repository != currentRepo) || row.Action == "blocked" || row.Action == "missing_repository" {
			markRollupRepositoryBlocked(&rollup, row.Repository)
		}
		rollup.Drift = append(rollup.Drift, ActiveStackRollupDriftRow{
			Repository: row.Repository,
			Workflow:   row.Workflow,
			RunID:      row.RunID,
			Action:     row.Action,
		})
	}

	if ledger.ReleaseHandoff.Status != "ready" {
		addProblem(fmt.Sprintf("release handoff status is %s", ledger.ReleaseHandoff.Status))
	}
	for _, gate := range ledger.ReleaseHandoff.Gates {
		classification := classifyReleaseHandoffGate(gate)
		rollup.ReleaseHandoff = append(rollup.ReleaseHandoff, ActiveStackRollupGate{
			Name:                    gate.Name,
			Status:                  gate.Status,
			RequiredBeforePromotion: gate.RequiredBeforePromotion,
			Classification:          classification,
		})
		switch classification {
		case "ready":
		case "promotion_manual_gate":
			rollup.ManualPromotionGates = append(rollup.ManualPromotionGates, gate.Name)
		default:
			addProblem(fmt.Sprintf("release handoff gate %s is %s", gate.Name, gate.Status))
		}
	}

	recountRollupRepositories(&rollup)
	if len(rollup.Problems) > 0 {
		rollup.Status = "blocked"
		rollup.NextActions = append(rollup.NextActions, rollup.Problems...)
	} else {
		rollup.NextActions = []string{
			"Keep sibling active-stack readiness evidence current after readiness PR merges.",
			"Run the signed-smoke release gate manually before promotion.",
		}
	}
	return rollup, nil
}

func markRollupRepositoryBlocked(rollup *ActiveStackProductionReadinessRollup, repoID string) {
	for i := range rollup.Repositories {
		if rollup.Repositories[i].ID == repoID {
			rollup.Repositories[i].Status = "blocked"
			return
		}
	}
}

func recountRollupRepositories(rollup *ActiveStackProductionReadinessRollup) {
	rollup.ReadyRepositories = 0
	rollup.BlockedRepositories = 0
	for _, repo := range rollup.Repositories {
		if repo.Status == "ready" {
			rollup.ReadyRepositories++
		} else {
			rollup.BlockedRepositories++
		}
	}
}

func githubRunRollupStatus(run ActiveStackGithubRun) string {
	status := strings.TrimSpace(run.Status)
	conclusion := strings.TrimSpace(run.Conclusion)
	if status == "" && conclusion == "" {
		return ""
	}
	if conclusion == "" {
		return status
	}
	return status + "/" + conclusion
}

func classifyReleaseHandoffGate(gate ReleaseHandoffGate) string {
	switch gate.Status {
	case "ready":
		return "ready"
	case "manual_required":
		if gate.RequiredBeforePromotion {
			return "promotion_manual_gate"
		}
		return "manual_gate"
	default:
		return "blocked"
	}
}

func renderActiveStackProductionReadinessRollupMarkdown(rollup ActiveStackProductionReadinessRollup) string {
	var b strings.Builder
	fmt.Fprintln(&b, "# Active Stack Production Readiness Rollup")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "Status: %s\n\n", rollup.Status)
	fmt.Fprintf(&b, "Ledger: %s\n\n", rollup.Ledger)
	fmt.Fprintf(&b, "GitHub runs report: %s\n\n", rollup.GithubRunsReport)
	fmt.Fprintf(&b, "Repositories: %d ready / %d active\n\n", rollup.ReadyRepositories, rollup.ActiveRepositories)
	fmt.Fprintln(&b, "| Repository | Status | Latest CI | Latest Ops |")
	fmt.Fprintln(&b, "| --- | --- | --- | --- |")
	for _, repo := range rollup.Repositories {
		fmt.Fprintf(&b, "| %s | %s | %s %s | %s %s |\n",
			escapeMarkdownCell(repo.ID),
			escapeMarkdownCell(repo.Status),
			escapeMarkdownCell(repo.LatestCIRunID),
			escapeMarkdownCell(repo.LatestCIStatus),
			escapeMarkdownCell(repo.LatestOpsRunID),
			escapeMarkdownCell(repo.LatestOpsStatus),
		)
	}
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "| Gate | Status | Classification |")
	fmt.Fprintln(&b, "| --- | --- | --- |")
	for _, gate := range rollup.ReleaseHandoff {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", escapeMarkdownCell(gate.Name), escapeMarkdownCell(gate.Status), escapeMarkdownCell(gate.Classification))
	}
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "| Repository | Workflow | Latest run | Action |")
	fmt.Fprintln(&b, "| --- | --- | --- | --- |")
	for _, row := range rollup.Drift {
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", escapeMarkdownCell(row.Repository), escapeMarkdownCell(row.Workflow), escapeMarkdownCell(row.RunID), escapeMarkdownCell(row.Action))
	}
	if len(rollup.NextActions) > 0 {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "## Next Actions")
		fmt.Fprintln(&b)
		for _, action := range rollup.NextActions {
			fmt.Fprintf(&b, "- %s\n", action)
		}
	}
	return b.String()
}

func renderActiveStackLedgerRefreshProposal(ledgerPath, reportPath string, rows []ActiveStackLedgerRefreshRow) string {
	var b strings.Builder
	fmt.Fprintln(&b, "# Active Stack Ledger Refresh Proposal")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "Generated from: %s\n\n", reportPath)
	fmt.Fprintf(&b, "Ledger target: %s\n\n", ledgerPath)
	fmt.Fprintln(&b, "| Repository | Workflow | Latest run | Action |")
	fmt.Fprintln(&b, "| --- | --- | --- | --- |")
	for _, row := range rows {
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", row.Repository, row.Workflow, row.RunID, row.Action)
	}
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "## Apply")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "1. Update %s with any `update` rows.\n", ledgerPath)
	fmt.Fprintf(&b, "2. Regenerate README snapshot with `go run ./cmd/foundry readiness snapshot --ledger %s`.\n", ledgerPath)
	fmt.Fprintf(&b, "3. Run `go run ./cmd/foundry readiness evidence-check --ledger %s --github-runs-report %s`.\n", ledgerPath, reportPath)
	return b.String()
}

func applyActiveStackLedgerRefresh(ledger ActiveStackReadinessLedger, report ActiveStackGithubRunsReport, rows []ActiveStackLedgerRefreshRow) (ActiveStackReadinessLedger, []string) {
	var changes []string
	actions := map[string]string{}
	for _, row := range rows {
		actions[row.Repository+" "+row.Workflow] = row.Action
	}
	for _, repoReport := range report.Repositories {
		repoID := githubRepositoryID(repoReport.Repository)
		for i := range ledger.Repositories {
			if ledger.Repositories[i].ID != repoID {
				continue
			}
			if actions[repoID+" ci.yml"] == "update" && repoReport.LatestCI.Status == "completed" && repoReport.LatestCI.Conclusion == "success" {
				if applyCIRefresh(&ledger.Repositories[i], repoReport.LatestCI) {
					changes = append(changes, repoID+" ci.yml")
				}
				if pr := pullRequestNumber(repoReport.LatestCI.DisplayName); pr != "" {
					if replaceOrAppendEvidence(&ledger.Repositories[i], regexp.MustCompile(`^PR #\d+ merged$`), "PR #"+pr+" merged") {
						changes = append(changes, repoID+" pr")
					}
				}
			}
			if actions[repoID+" production-readiness-ops.yml"] == "update" && repoReport.LatestOps.Status == "completed" && repoReport.LatestOps.Conclusion == "success" {
				if replaceOrAppendEvidence(&ledger.Repositories[i], regexp.MustCompile(`^Production Readiness Ops run \d+$`), "Production Readiness Ops run "+strings.TrimSpace(repoReport.LatestOps.RunID)) {
					changes = append(changes, repoID+" production-readiness-ops.yml")
				}
			}
		}
	}
	return ledger, changes
}

func applyCIRefresh(repo *ActiveStackReadinessRepository, run ActiveStackGithubRun) bool {
	runID := strings.TrimSpace(run.RunID)
	if runID == "" {
		return false
	}
	if repo.ID != "ao-foundry" && repo.CI != nil && repo.CI.RunID != "" {
		if repo.CI.RunID == runID {
			return false
		}
		repo.CI.RunID = runID
		return true
	}
	return replaceOrAppendEvidence(repo, regexp.MustCompile(`^main CI run \d+$`), "main CI run "+runID)
}

func replaceOrAppendEvidence(repo *ActiveStackReadinessRepository, pattern *regexp.Regexp, value string) bool {
	for i, evidence := range repo.VerificationEvidence {
		if pattern.MatchString(evidence) {
			if evidence == value {
				return false
			}
			repo.VerificationEvidence[i] = value
			return true
		}
	}
	repo.VerificationEvidence = append(repo.VerificationEvidence, value)
	return true
}

func pullRequestNumber(title string) string {
	match := regexp.MustCompile(`\(#(\d+)\)`).FindStringSubmatch(title)
	if len(match) != 2 {
		return ""
	}
	return match[1]
}

func refreshReadmeActiveStackSnapshot(readmePath, ledgerPath string, ledger ActiveStackReadinessLedger) error {
	data, err := os.ReadFile(readmePath)
	if err != nil {
		return err
	}
	snapshot := renderActiveStackReadinessSnapshot(ledgerPath, ledger)
	updated, err := replaceMarkedBlock(string(data), "<!-- foundry:active-stack-readiness:start -->", "<!-- foundry:active-stack-readiness:end -->", strings.TrimSuffix(snapshot, "\n"))
	if err != nil {
		return err
	}
	return writeTextFile(readmePath, updated)
}

func replaceMarkedBlock(text, startMarker, endMarker, replacement string) (string, error) {
	start := strings.Index(text, startMarker)
	if start < 0 {
		return "", fmt.Errorf("missing start marker %q", startMarker)
	}
	end := strings.Index(text, endMarker)
	if end < 0 {
		return "", fmt.Errorf("missing end marker %q", endMarker)
	}
	if end <= start {
		return "", errors.New("marker order is invalid")
	}
	end += len(endMarker)
	return text[:start] + replacement + text[end:], nil
}

func validateReleaseCandidateLedger(ledger ReleaseCandidateLedger) error {
	if ledger.SchemaVersion != "ao.foundry.release-candidate.v0.1" {
		return errors.New("invalid release candidate schema_version")
	}
	if strings.TrimSpace(ledger.CandidateID) == "" {
		return errors.New("release candidate requires candidate_id")
	}
	if ledger.Status != "ready" {
		return errors.New("release candidate status must be ready")
	}
	expectedRepos := map[string]bool{
		"ao2":               false,
		"ao2-control-plane": false,
		"ao-foundry":        false,
	}
	if len(ledger.ActiveSpine) != len(expectedRepos) {
		return fmt.Errorf("release candidate active spine must include exactly %d repositories", len(expectedRepos))
	}
	for _, repo := range ledger.ActiveSpine {
		if _, ok := expectedRepos[repo.ID]; !ok {
			return fmt.Errorf("release candidate includes non-spine repository %q", repo.ID)
		}
		if expectedRepos[repo.ID] {
			return fmt.Errorf("release candidate duplicates repository %q", repo.ID)
		}
		expectedRepos[repo.ID] = true
		if strings.TrimSpace(repo.Name) == "" || strings.TrimSpace(repo.Role) == "" {
			return fmt.Errorf("release candidate repository %q requires name and role", repo.ID)
		}
		if repo.Status != "ready" {
			return fmt.Errorf("release candidate repository %q must be ready", repo.ID)
		}
		if len(repo.Evidence) == 0 {
			return fmt.Errorf("release candidate repository %q requires evidence", repo.ID)
		}
		for _, evidence := range repo.Evidence {
			if strings.TrimSpace(evidence) == "" {
				return fmt.Errorf("release candidate repository %q has blank evidence", repo.ID)
			}
		}
	}
	for repoID, seen := range expectedRepos {
		if !seen {
			return fmt.Errorf("release candidate missing repository %q", repoID)
		}
	}
	if len(ledger.Gates) == 0 {
		return errors.New("release candidate requires gates")
	}
	for _, gate := range ledger.Gates {
		if strings.TrimSpace(gate.Name) == "" {
			return errors.New("release candidate gate requires name")
		}
		switch gate.Status {
		case "ready", "manual_required", "blocked":
		default:
			return fmt.Errorf("release candidate gate %q has invalid status %q", gate.Name, gate.Status)
		}
		if len(gate.Evidence) == 0 {
			return fmt.Errorf("release candidate gate %q requires evidence", gate.Name)
		}
		for _, evidence := range gate.Evidence {
			if strings.TrimSpace(evidence) == "" {
				return fmt.Errorf("release candidate gate %q has blank evidence", gate.Name)
			}
		}
	}
	signedSmokeGate, ok := releaseCandidateGateByName(ledger, "signed_smoke_release_gate")
	if !ok {
		return errors.New("release candidate requires signed_smoke_release_gate")
	}
	if signedSmokeGate.Status != "manual_required" {
		return errors.New("signed_smoke_release_gate must be manual_required until promotion evidence is attached")
	}
	if !signedSmokeGate.RequiredBeforePromotion {
		return errors.New("signed_smoke_release_gate must be required before promotion")
	}
	return nil
}

func checkReleaseCandidateActiveStackParity(candidate ReleaseCandidateLedger, readiness ActiveStackReadinessLedger) ([]string, int) {
	readinessRepos := map[string]ActiveStackReadinessRepository{}
	for _, repo := range readiness.Repositories {
		readinessRepos[repo.ID] = repo
	}
	var issues []string
	reposChecked := 0
	for _, candidateRepo := range candidate.ActiveSpine {
		readinessRepo, ok := readinessRepos[candidateRepo.ID]
		if !ok {
			issues = append(issues, fmt.Sprintf("%s missing from active-stack readiness ledger", candidateRepo.ID))
			continue
		}
		reposChecked++
		required := releaseCandidateRequiredActiveStackEvidence(readinessRepo)
		requiredKinds := map[string]bool{}
		for _, requiredEvidence := range required {
			if kind := releaseCandidateEvidenceKind(requiredEvidence); kind != "" {
				requiredKinds[kind] = true
			}
			if !releaseCandidateEvidenceContains(candidateRepo.Evidence, requiredEvidence) {
				issues = append(issues, fmt.Sprintf("%s missing active-stack evidence %q", candidateRepo.ID, requiredEvidence))
			}
			for _, staleEvidence := range releaseCandidateStaleEvidenceFor(candidateRepo.Evidence, requiredEvidence) {
				issues = append(issues, fmt.Sprintf("%s has stale evidence %q", candidateRepo.ID, staleEvidence))
			}
		}
		for _, evidence := range candidateRepo.Evidence {
			kind := releaseCandidateEvidenceKind(evidence)
			if kind != "" && !requiredKinds[kind] {
				issues = append(issues, fmt.Sprintf("%s has unrequired evidence %q", candidateRepo.ID, evidence))
			}
		}
	}
	sort.Strings(issues)
	return issues, reposChecked
}

func releaseCandidateRequiredActiveStackEvidence(repo ActiveStackReadinessRepository) []string {
	var required []string
	seen := map[string]bool{}
	add := func(evidence string) {
		evidence = strings.TrimSpace(evidence)
		if evidence == "" || seen[evidence] {
			return
		}
		seen[evidence] = true
		required = append(required, evidence)
	}
	if repo.CI != nil && strings.TrimSpace(repo.CI.RunID) != "" {
		add("main CI run " + strings.TrimSpace(repo.CI.RunID))
	}
	for _, evidence := range repo.VerificationEvidence {
		if releaseCandidateEvidenceKind(evidence) != "" {
			add(evidence)
		}
	}
	return required
}

func releaseCandidateEvidenceContains(evidence []string, want string) bool {
	for _, item := range evidence {
		if item == want {
			return true
		}
	}
	return false
}

func releaseCandidateStaleEvidenceFor(candidateEvidence []string, requiredEvidence string) []string {
	requiredKind := releaseCandidateEvidenceKind(requiredEvidence)
	if requiredKind == "" {
		return nil
	}
	var stale []string
	for _, evidence := range candidateEvidence {
		if releaseCandidateEvidenceKind(evidence) == requiredKind && evidence != requiredEvidence {
			stale = append(stale, evidence)
		}
	}
	return stale
}

func releaseCandidateEvidenceKind(evidence string) string {
	evidence = strings.TrimSpace(evidence)
	switch {
	case regexp.MustCompile(`^main CI run [0-9]+$`).MatchString(evidence):
		return "main-ci"
	case regexp.MustCompile(`^Production Readiness Ops run [0-9]+$`).MatchString(evidence):
		return "production-readiness-ops"
	case regexp.MustCompile(`^PR #[0-9]+ merged$`).MatchString(evidence):
		return "merged-pr"
	default:
		return ""
	}
}

func releaseCandidateGateByName(ledger ReleaseCandidateLedger, name string) (ReleaseCandidateGate, bool) {
	for _, gate := range ledger.Gates {
		if gate.Name == name {
			return gate, true
		}
	}
	return ReleaseCandidateGate{}, false
}

func buildReleasePromotionLedger(candidatePath, summaryPath string) (ReleasePromotionLedger, error) {
	candidate, err := loadReleaseCandidateLedger(candidatePath)
	if err != nil {
		return ReleasePromotionLedger{}, err
	}
	summary, err := loadSignedSmokeSummary(summaryPath)
	if err != nil {
		return ReleasePromotionLedger{}, err
	}
	evidence := make([]ReleasePromotionEvidence, 0, len(summary.Evidence))
	for _, item := range summary.Evidence {
		evidence = append(evidence, ReleasePromotionEvidence{
			Name:          item.Name,
			Status:        item.Status,
			SchemaVersion: item.SchemaVersion,
		})
	}
	return ReleasePromotionLedger{
		SchemaVersion:            "ao.foundry.release-promotion.v0.1",
		CandidateID:              candidate.CandidateID,
		Status:                   "ready",
		ReleaseSafe:              true,
		SignedSmokePulseID:       summary.PulseID,
		SignedSmokeSummaryStatus: summary.Status,
		PulseStatus:              summary.PulseStatus,
		Evidence:                 evidence,
		NextActions: []string{
			"Attach release-promotion ledger to release notes",
			"Promote only the bound active-spine candidate",
		},
	}, nil
}

func loadSignedSmokeSummary(path string) (SignedSmokeSummary, error) {
	var summary SignedSmokeSummary
	if strings.TrimSpace(path) == "" {
		return summary, errors.New("missing --signed-smoke-summary")
	}
	if err := readJSONFile(path, &summary); err != nil {
		return summary, err
	}
	return summary, validateSignedSmokeSummary(summary)
}

func loadReleasePromotionLedger(path string) (ReleasePromotionLedger, error) {
	var ledger ReleasePromotionLedger
	if strings.TrimSpace(path) == "" {
		return ledger, errors.New("missing --promotion")
	}
	if err := readJSONFile(path, &ledger); err != nil {
		return ledger, err
	}
	if ledger.SchemaVersion != "ao.foundry.release-promotion.v0.1" {
		return ledger, errors.New("invalid release promotion schema_version")
	}
	if strings.TrimSpace(ledger.CandidateID) == "" {
		return ledger, errors.New("release promotion requires candidate_id")
	}
	if ledger.Status != "ready" {
		return ledger, errors.New("release promotion status must be ready")
	}
	if !ledger.ReleaseSafe {
		return ledger, errors.New("release promotion must be release_safe")
	}
	if strings.TrimSpace(ledger.SignedSmokePulseID) == "" || ledger.SignedSmokeSummaryStatus != "ready" || ledger.PulseStatus != "ready" {
		return ledger, errors.New("release promotion requires ready signed-smoke and pulse evidence")
	}
	if len(ledger.Evidence) == 0 {
		return ledger, errors.New("release promotion requires evidence")
	}
	return ledger, nil
}

func validateSignedSmokeSummary(summary SignedSmokeSummary) error {
	if summary.SchemaVersion != "ao.foundry.signed-smoke-summary.v0.1" {
		return errors.New("invalid signed-smoke summary schema_version")
	}
	if strings.TrimSpace(summary.PulseID) == "" {
		return errors.New("signed-smoke summary requires pulse_id")
	}
	if summary.Status != "ready" {
		return errors.New("signed-smoke summary status must be ready")
	}
	if summary.PulseStatus != "ready" {
		return errors.New("signed-smoke summary pulse_status must be ready")
	}
	if !summary.ReleaseSafe {
		return errors.New("signed-smoke summary must be release_safe")
	}
	requiredEvidence := map[string]string{
		"forge_live_attempt":     "passed",
		"control_plane_readback": "ready",
		"signed_smoke_ingest":    "ready",
	}
	seenEvidence := map[string]bool{}
	for _, evidence := range summary.Evidence {
		wantStatus, required := requiredEvidence[evidence.Name]
		if !required {
			continue
		}
		if evidence.Status != wantStatus {
			return fmt.Errorf("signed-smoke summary evidence %q status = %q, want %q", evidence.Name, evidence.Status, wantStatus)
		}
		if strings.TrimSpace(evidence.SchemaVersion) == "" || evidence.SchemaVersion == "missing" {
			return fmt.Errorf("signed-smoke summary evidence %q requires schema_version", evidence.Name)
		}
		seenEvidence[evidence.Name] = true
	}
	for name := range requiredEvidence {
		if !seenEvidence[name] {
			return fmt.Errorf("signed-smoke summary missing evidence %q", name)
		}
	}
	return nil
}

func loadEvalScorecard(path string) (EvalScorecard, error) {
	var scorecard EvalScorecard
	if path == "" {
		return scorecard, errors.New("missing --scorecard")
	}
	if err := readJSONFile(path, &scorecard); err != nil {
		return scorecard, err
	}
	if scorecard.SchemaVersion != "ao.foundry.eval-scorecard.v0.1" {
		return scorecard, errors.New("invalid scorecard schema_version")
	}
	if scorecard.ScorecardID == "" || scorecard.Threshold <= 0 {
		return scorecard, errors.New("scorecard_id and positive threshold are required")
	}
	if len(scorecard.Dimensions) == 0 {
		return scorecard, errors.New("scorecard dimensions are required")
	}
	for _, dim := range scorecard.Dimensions {
		if dim.Name == "" || dim.MaxScore <= 0 {
			return scorecard, errors.New("scorecard dimensions require name and positive max_score")
		}
	}
	return scorecard, nil
}

func loadForgePacket(path string) (ForgePacket, []byte, error) {
	var packet ForgePacket
	if path == "" {
		return packet, nil, errors.New("missing --packet")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		data, err = readRepoRelativeFile(path)
		if err != nil {
			return packet, nil, fmt.Errorf("packet: %w", err)
		}
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&packet); err != nil {
		return packet, nil, fmt.Errorf("packet: invalid JSON: %w", err)
	}
	if err := validateForgePacket(packet); err != nil {
		return packet, nil, err
	}
	return packet, data, nil
}

func readJSONFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		data, err = readRepoRelativeFile(path)
		if err != nil {
			return err
		}
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

func readRepoRelativeFile(path string) ([]byte, error) {
	if filepath.IsAbs(path) {
		return nil, os.ErrNotExist
	}
	root, err := repoRoot()
	if err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(root, filepath.Clean(filepath.FromSlash(path))))
}

func writeJSONFile(path string, value any) error {
	if err := os.MkdirAll(parentDir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return writeJSON(file, value)
}

func writeTextFile(path, value string) error {
	if err := os.MkdirAll(parentDir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(value), 0o644)
}

func writeJSON(w io.Writer, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = w.Write(data)
	return err
}

func writeTraceSpan(path, component, operation, status string, attributes map[string]string, evidenceRefs []string, problem string) {
	if path == "" {
		return
	}
	filteredEvidence := []string{}
	for _, ref := range evidenceRefs {
		if strings.TrimSpace(ref) != "" {
			filteredEvidence = append(filteredEvidence, ref)
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	traceID := "trace-" + shortSHA256(component+":"+operation+":"+strings.Join(filteredEvidence, ","))
	span := TraceSpan{
		SchemaVersion: "ao.foundry.trace.v0.1",
		TraceID:       traceID,
		SpanID:        "span-" + shortSHA256(traceID+":"+status),
		Component:     component,
		Operation:     operation,
		Status:        status,
		StartedAt:     now,
		EndedAt:       now,
		DurationMS:    0,
		Attributes:    attributes,
		EvidenceRefs:  filteredEvidence,
		Problem:       problem,
	}
	if err := os.MkdirAll(parentDir(path), 0o755); err != nil {
		return
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	data, err := json.Marshal(span)
	if err != nil {
		return
	}
	_, _ = file.Write(append(data, '\n'))
}

func readTraceSpans(path string) ([]TraceSpan, error) {
	if path == "" {
		return nil, errors.New("missing --trace")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && strings.TrimSpace(lines[0]) == "") {
		return nil, errors.New("trace is empty")
	}
	spans := make([]TraceSpan, 0, len(lines))
	terminal := false
	for _, line := range lines {
		var span TraceSpan
		if err := json.Unmarshal([]byte(line), &span); err != nil {
			return nil, fmt.Errorf("malformed trace: %w", err)
		}
		if span.SchemaVersion != "ao.foundry.trace.v0.1" || span.TraceID == "" || span.SpanID == "" || span.Component == "" || span.Operation == "" {
			return nil, errors.New("malformed trace span")
		}
		if span.Status == "passed" || span.Status == "failed" {
			if span.EndedAt == "" {
				return nil, errors.New("terminal span missing ended_at")
			}
			terminal = true
		}
		spans = append(spans, span)
	}
	if !terminal {
		return nil, errors.New("trace missing terminal span")
	}
	return spans, nil
}

func parentDir(path string) string {
	idx := strings.LastIndexAny(path, `/\`)
	if idx < 0 {
		return "."
	}
	return path[:idx]
}

func validateRegistry(registry Registry) error {
	if registry.SchemaVersion != registrySchema {
		return fmt.Errorf("schema_version must be %s", registrySchema)
	}
	if registry.FoundryID == "" {
		return errors.New("foundry_id is required")
	}
	if len(registry.Repos) == 0 {
		return errors.New("repos must contain at least one repository")
	}
	seen := map[string]bool{}
	for _, repo := range registry.Repos {
		if repo.ID == "" || repo.Name == "" || repo.Role == "" {
			return errors.New("each repo requires id, name, and role")
		}
		if !allowedRole(repo.Role) {
			return fmt.Errorf("repo %q has invalid role %q", repo.ID, repo.Role)
		}
		if seen[repo.ID] {
			return fmt.Errorf("duplicate repo id %q", repo.ID)
		}
		seen[repo.ID] = true
		if repo.DelegatesTo == "" {
			return fmt.Errorf("repo %q requires delegates_to", repo.ID)
		}
		if repo.Workspace == "" {
			return fmt.Errorf("repo %q requires workspace", repo.ID)
		}
		if len(repo.Branches) == 0 {
			return fmt.Errorf("repo %q requires branches", repo.ID)
		}
		for _, branch := range repo.Branches {
			if branch == "" {
				return fmt.Errorf("repo %q has empty branch", repo.ID)
			}
		}
		if len(repo.EvidenceSources) == 0 {
			return fmt.Errorf("repo %q requires evidence_sources", repo.ID)
		}
		for _, source := range repo.EvidenceSources {
			if source.Kind == "" || source.Location == "" || source.Owner == "" {
				return fmt.Errorf("repo %q has incomplete evidence source", repo.ID)
			}
		}
		if len(repo.AllowedAutomation) == 0 {
			return fmt.Errorf("repo %q requires allowed_automation", repo.ID)
		}
		for _, automation := range repo.AllowedAutomation {
			if automation == "" {
				return fmt.Errorf("repo %q has empty allowed automation", repo.ID)
			}
		}
		if len(repo.ReadinessSignals) == 0 {
			return fmt.Errorf("repo %q requires readiness_signals", repo.ID)
		}
		for _, signal := range repo.ReadinessSignals {
			if signal.Name == "" || signal.Source == "" {
				return fmt.Errorf("repo %q has incomplete readiness signal", repo.ID)
			}
			switch signal.Status {
			case "ready", "blocked", "unknown":
			default:
				return fmt.Errorf("repo %q has invalid readiness status %q", repo.ID, signal.Status)
			}
		}
		for _, command := range repo.Health.VerificationCommands {
			if strings.TrimSpace(command) == "" {
				return fmt.Errorf("repo %q has empty health verification command", repo.ID)
			}
		}
		for _, file := range repo.Health.ReadinessFiles {
			if strings.TrimSpace(file) == "" {
				return fmt.Errorf("repo %q has empty health readiness file", repo.ID)
			}
			if err := validateEvidencePath(file); err != nil {
				return fmt.Errorf("repo %q has unsafe health readiness file: %w", repo.ID, err)
			}
		}
		for _, tag := range repo.Health.RequireTags {
			if strings.TrimSpace(tag) == "" {
				return fmt.Errorf("repo %q has empty required tag", repo.ID)
			}
		}
	}
	return nil
}

func validateTask(task Task) error {
	if task.SchemaVersion != taskSchema {
		return fmt.Errorf("schema_version must be %s", taskSchema)
	}
	if task.TaskID == "" || task.Title == "" || task.Objective == "" {
		return errors.New("task_id, title, and objective are required")
	}
	if len(task.TargetRepos) == 0 {
		return errors.New("target_repos must contain at least one repository")
	}
	if len(task.RequiredDelegation) == 0 {
		return errors.New("required_delegations must contain at least one delegation")
	}
	if len(task.Acceptance) == 0 {
		return errors.New("acceptance must contain at least one criterion")
	}
	if len(task.Verification) == 0 {
		return errors.New("verification must contain at least one command")
	}
	if !allowedPriority(task.Priority) {
		return fmt.Errorf("invalid priority %q", task.Priority)
	}
	if !allowedTaskState(task.State) {
		return fmt.Errorf("invalid task state %q", task.State)
	}
	for _, delegation := range task.RequiredDelegation {
		if delegation.DelegateTo == "" || delegation.Reason == "" {
			return errors.New("required_delegations entries require delegate_to and reason")
		}
	}
	for _, forbidden := range task.Safety.ForbiddenActions {
		if forbidden == "" {
			return errors.New("safety.forbidden_actions entries must not be empty")
		}
	}
	if len(task.Safety.AllowedWriteRoots) == 0 {
		return errors.New("safety.allowed_write_roots must not be empty")
	}
	return nil
}

func validateAtlasFoundryImport(artifact AtlasFoundryImport) error {
	if artifact.ContractVersion != atlasImportSchema {
		return fmt.Errorf("contract_version must be %s", atlasImportSchema)
	}
	if artifact.ID == "" || artifact.WorkgraphID == "" || artifact.TargetInstance == "" {
		return errors.New("id, workgraph_id, and target_instance are required")
	}
	if artifact.Status != "ready_for_foundry_fixture_import" {
		return errors.New("status must be ready_for_foundry_fixture_import")
	}
	if artifact.SchedulesWork {
		return errors.New("schedules_work must be false")
	}
	if artifact.ExecutesWork {
		return errors.New("executes_work must be false")
	}
	if artifact.ApprovesWork {
		return errors.New("approves_work must be false")
	}
	if len(artifact.SourceArtifacts) == 0 {
		return errors.New("source_artifacts must not be empty")
	}
	for i, source := range artifact.SourceArtifacts {
		if strings.TrimSpace(source.Ref) == "" {
			return fmt.Errorf("source_artifacts[%d].ref must not be empty", i)
		}
		if err := validateEvidencePath(source.Ref); err != nil {
			return fmt.Errorf("source_artifacts[%d].ref: %w", i, err)
		}
		if !strings.HasPrefix(source.Digest, "sha256:") {
			return fmt.Errorf("source_artifacts[%d].digest must start with sha256:", i)
		}
		if err := validateSHA256(strings.TrimPrefix(source.Digest, "sha256:"), fmt.Sprintf("source_artifacts[%d].digest", i)); err != nil {
			return err
		}
	}
	if len(artifact.Tasks) == 0 {
		return errors.New("tasks must not be empty")
	}
	seenPaths := map[string]bool{}
	for i, fixture := range artifact.Tasks {
		if fixture.NodeID == "" || fixture.TaskID == "" || fixture.Path == "" {
			return fmt.Errorf("tasks[%d] requires node_id, task_id, and path", i)
		}
		if fixture.MutationClass == "" {
			return fmt.Errorf("tasks[%d].mutation_class must not be empty", i)
		}
		if len(fixture.WriteScope) == 0 || len(fixture.RollbackScope) == 0 || len(fixture.RequiredGates) == 0 || len(fixture.RequiredEvidence) == 0 || fixture.AuthorityBoundary == "" {
			return fmt.Errorf("tasks[%d] requires write_scope, rollback_scope, required_gates, required_evidence, and authority_boundary", i)
		}
		if !validAtlasMutationClass(fixture.MutationClass) {
			return fmt.Errorf("tasks[%d].mutation_class is not supported", i)
		}
		if seenPaths[fixture.Path] {
			return fmt.Errorf("tasks[%d].path must be unique", i)
		}
		seenPaths[fixture.Path] = true
		if err := validateEvidencePath(fixture.Path); err != nil {
			return fmt.Errorf("tasks[%d].path: %w", i, err)
		}
		for _, values := range [][]string{fixture.WriteScope, fixture.RollbackScope, fixture.RequiredGates, fixture.RequiredEvidence} {
			for _, value := range values {
				if strings.TrimSpace(value) == "" {
					return fmt.Errorf("tasks[%d] authority metadata lists must not contain empty values", i)
				}
				if err := validateAtlasPublicString(value); err != nil {
					return fmt.Errorf("tasks[%d] authority metadata: %w", i, err)
				}
			}
		}
		if err := validateAtlasPublicString(fixture.AuthorityBoundary); err != nil {
			return fmt.Errorf("tasks[%d].authority_boundary: %w", i, err)
		}
		if fixture.TaskID != fixture.Task.ID {
			return fmt.Errorf("tasks[%d].task_id must match task.id", i)
		}
		if err := validateAtlasFactoryTask(fixture.Task); err != nil {
			return fmt.Errorf("tasks[%d].task: %w", i, err)
		}
		if fixture.MutationClass != fixture.Task.MutationClass {
			return fmt.Errorf("tasks[%d].mutation_class must match task.mutation_class", i)
		}
		if !equalStringSlices(fixture.WriteScope, fixture.Task.WriteScope) {
			return fmt.Errorf("tasks[%d].write_scope must match task.write_scope", i)
		}
		if !equalStringSlices(fixture.RollbackScope, fixture.Task.RollbackScope) {
			return fmt.Errorf("tasks[%d].rollback_scope must match task.rollback_scope", i)
		}
		if !equalStringSlices(fixture.RequiredGates, fixture.Task.RequiredGates) {
			return fmt.Errorf("tasks[%d].required_gates must match task.required_gates", i)
		}
		if !equalStringSlices(fixture.RequiredEvidence, fixture.Task.RequiredEvidence) {
			return fmt.Errorf("tasks[%d].required_evidence must match task.required_evidence", i)
		}
		if fixture.AuthorityBoundary != fixture.Task.AuthorityBoundary {
			return fmt.Errorf("tasks[%d].authority_boundary must match task.authority_boundary", i)
		}
		if fixture.TaskDigest != digestAtlasFactoryTask(fixture.Task) {
			return fmt.Errorf("tasks[%d].task_digest does not match embedded task", i)
		}
	}
	return nil
}

func validateAtlasBlueprintImport(artifact AtlasBlueprintImport) error {
	if artifact.ContractVersion != atlasBlueprintImportSchema {
		return fmt.Errorf("contract_version must be %s", atlasBlueprintImportSchema)
	}
	if artifact.ID == "" || artifact.ProjectID == "" || artifact.Status == "" {
		return errors.New("id, project_id, and status are required")
	}
	if artifact.Status != "ready" {
		return errors.New("Atlas Blueprint import status must be ready")
	}
	if !artifact.ReadyForFoundry {
		return errors.New("Atlas Blueprint import ready_for_foundry must be true")
	}
	if artifact.SafeToExecute || artifact.LiveExecutionProven {
		return errors.New("Atlas Blueprint import must not mark live execution safe or proven")
	}
	if artifact.SchedulesWork {
		return errors.New("schedules_work must be false")
	}
	if artifact.ExecutesWork {
		return errors.New("executes_work must be false")
	}
	if artifact.ApprovesWork {
		return errors.New("approves_work must be false")
	}
	if artifact.MutatesRepositories {
		return errors.New("mutates_repositories must be false")
	}
	if artifact.CallsProviders {
		return errors.New("calls_providers must be false")
	}
	if artifact.ReleaseOrPublishAllowed {
		return errors.New("release_or_publish_allowed must be false")
	}
	for name, source := range map[string]AtlasSourceArtifact{
		"blueprint_pack":            artifact.BlueprintPack,
		"build_authorization":       artifact.BuildAuthorization,
		"downstream_foundry_import": artifact.DownstreamFoundryImport,
	} {
		if strings.TrimSpace(source.Ref) == "" {
			return fmt.Errorf("%s.ref must not be empty", name)
		}
		if err := validateEvidencePath(source.Ref); err != nil {
			return fmt.Errorf("%s.ref: %w", name, err)
		}
		if !strings.HasPrefix(source.Digest, "sha256:") {
			return fmt.Errorf("%s.digest must start with sha256:", name)
		}
		if err := validateSHA256(strings.TrimPrefix(source.Digest, "sha256:"), name+".digest"); err != nil {
			return err
		}
	}
	requiredDigests := []string{
		"blueprint_pack",
		"build_authorization",
		"implementation_spec",
		"quality_profile",
		"candidate_rules",
		"mutation_class_model",
		"candidate_selection",
		"workgraph",
		"downstream_foundry_import",
	}
	for _, key := range requiredDigests {
		digest := artifact.Digests[key]
		if strings.TrimSpace(digest) == "" {
			return fmt.Errorf("digests.%s must not be empty", key)
		}
		if !strings.HasPrefix(digest, "sha256:") {
			return fmt.Errorf("digests.%s must start with sha256:", key)
		}
		if err := validateSHA256(strings.TrimPrefix(digest, "sha256:"), "digests."+key); err != nil {
			return err
		}
	}
	if artifact.Digests["downstream_foundry_import"] != artifact.DownstreamFoundryImport.Digest {
		return errors.New("Atlas Blueprint import downstream Foundry import digest must match digests.downstream_foundry_import")
	}
	if artifact.MutationClass == "" || !validAtlasMutationClass(artifact.MutationClass) {
		return errors.New("Atlas Blueprint import mutation_class must be supported")
	}
	if len(artifact.SafetyLimits) == 0 {
		return errors.New("Atlas Blueprint import safety_limits must not be empty")
	}
	for _, value := range artifact.SafetyLimits {
		if err := validateAtlasPublicString(value); err != nil {
			return fmt.Errorf("safety_limits: %w", err)
		}
	}
	return validatePublicSafeJSONStrings(artifact)
}

func validateAtlasBlueprintImportForFoundry(blueprintImport AtlasBlueprintImport, foundryImport AtlasFoundryImport, blueprintSource, importSource PulseIntakeSource) error {
	if blueprintImport.WorkgraphID != foundryImport.WorkgraphID || blueprintImport.TargetInstance != foundryImport.TargetInstance {
		return errors.New("Atlas Blueprint import must match downstream Foundry import identity")
	}
	if blueprintImport.BuildAuthorization.Digest != "sha256:"+blueprintSource.SHA256 {
		return errors.New("Atlas Blueprint import build authorization digest must match provided Blueprint authorization")
	}
	if blueprintImport.DownstreamFoundryImport.Digest != blueprintImport.Digests["downstream_foundry_import"] {
		return errors.New("Atlas Blueprint import downstream digest binding is inconsistent")
	}
	if blueprintImport.DownstreamFoundryImport.Digest != "sha256:"+importSource.SHA256 {
		return errors.New("Atlas Blueprint import downstream digest must match provided Foundry import")
	}
	return nil
}

func validateAtlasFactoryTask(task AtlasFactoryTask) error {
	if task.ContractVersion != atlasTaskSchema {
		return fmt.Errorf("contract_version must be %s", atlasTaskSchema)
	}
	if task.ID == "" || task.Objective == "" || task.TargetFactoryRepo == "" || task.FactoryFolder == "" {
		return errors.New("id, objective, target_factory_repo, and factory_folder are required")
	}
	if task.MutationClass == "" || task.AuthorityBoundary == "" {
		return errors.New("mutation_class and authority_boundary must not be empty")
	}
	if !validAtlasMutationClass(task.MutationClass) {
		return errors.New("mutation_class is not supported")
	}
	if len(task.Acceptance) == 0 || len(task.NonGoals) == 0 || len(task.WriteScope) == 0 || len(task.RequiredGates) == 0 || len(task.RollbackScope) == 0 || len(task.Verification) == 0 || len(task.RequiredEvidence) == 0 || len(task.SafetyLimits) == 0 {
		return errors.New("acceptance, non_goals, write_scope, required_gates, rollback_scope, verification_commands, required_evidence, and safety_limits must not be empty")
	}
	if err := validateAtlasPublicString(task.TargetFactoryRepo); err != nil {
		return fmt.Errorf("target_factory_repo: %w", err)
	}
	if err := validateEvidencePath(task.FactoryFolder); err != nil {
		return fmt.Errorf("factory_folder: %w", err)
	}
	if err := validateAtlasPublicString(task.AuthorityBoundary); err != nil {
		return fmt.Errorf("authority_boundary: %w", err)
	}
	for _, values := range [][]string{task.Acceptance, task.NonGoals, task.WriteScope, task.RequiredGates, task.RollbackScope, task.Verification, task.RequiredEvidence, task.SafetyLimits, task.DependencyRefs, task.ContextPackRefs} {
		for _, value := range values {
			if strings.TrimSpace(value) == "" {
				return errors.New("task lists must not contain empty values")
			}
			if err := validateAtlasPublicString(value); err != nil {
				return err
			}
		}
	}
	return nil
}

func validAtlasMutationClass(class string) bool {
	switch class {
	case "docs_only_single_file",
		"docs_only_multi_file",
		"docs_config_only",
		"test_only",
		"low_risk_code",
		"multi_repo_low_risk",
		"complex_repo_mutation":
		return true
	default:
		return false
	}
}

func validateGoalRun(goal GoalRun) error {
	if goal.SchemaVersion != goalRunSchema {
		return fmt.Errorf("schema_version must be %s", goalRunSchema)
	}
	if goal.GoalID == "" || goal.Objective == "" || goal.NextTask == "" {
		return errors.New("goal_id, objective, and next_task are required")
	}
	if len(goal.AcceptanceCriteria) == 0 {
		return errors.New("acceptance_criteria must not be empty")
	}
	if len(goal.AllowedScope) == 0 {
		return errors.New("allowed_scope must not be empty")
	}
	for _, scope := range goal.AllowedScope {
		if scope == "" || strings.HasPrefix(scope, "/") || strings.Contains(scope, ".."+string(os.PathSeparator)+"..") {
			return fmt.Errorf("unsafe allowed scope %q", scope)
		}
	}
	if len(goal.StopConditions) == 0 {
		return errors.New("stop_conditions must not be empty")
	}
	if !allowedGoalPhase(goal.CurrentPhase) {
		return fmt.Errorf("invalid goal phase %q", goal.CurrentPhase)
	}
	if goal.ContinuationPrompt == "" || goal.LoopOwner == "" || goal.NextActionGuard == "" {
		return errors.New("continuation_prompt, loop_owner, and next_action_guard are required")
	}
	if err := validateNextActionGuard(goal.NextActionGuard); err != nil {
		return err
	}
	if len(goal.LastIteration.Evidence) == 0 {
		return errors.New("last_iteration.evidence must not be empty")
	}
	for _, evidence := range goal.LastIteration.Evidence {
		if evidence.Label == "" || evidence.Path == "" || evidence.SHA256 == "" {
			return errors.New("evidence entries require label, path, and sha256")
		}
		if err := validateEvidencePath(evidence.Path); err != nil {
			return err
		}
		if len(evidence.SHA256) != 64 {
			return fmt.Errorf("evidence %q sha256 must be 64 lowercase hex characters", evidence.Label)
		}
		for _, c := range evidence.SHA256 {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				return fmt.Errorf("evidence %q sha256 must be 64 lowercase hex characters", evidence.Label)
			}
		}
	}
	return nil
}

func validateFoundryRun(run FoundryRun) error {
	if run.SchemaVersion != runSchema {
		return fmt.Errorf("schema_version must be %s", runSchema)
	}
	if run.RunID == "" || run.TaskID == "" || run.RegistryID == "" {
		return errors.New("run_id, task_id, and registry_id are required")
	}
	if !allowedRunStatus(run.Status) {
		return fmt.Errorf("invalid run status %q", run.Status)
	}
	if run.DelegatedTo != "ao-forge" {
		return errors.New("delegated_to must be ao-forge")
	}
	if run.ForgePacket.Path == "" || run.ForgePacket.SHA256 == "" || run.ForgePacket.Status == "" {
		return errors.New("forge_packet requires path, sha256, and status")
	}
	if err := validateEvidencePath(run.ForgePacket.Path); err != nil {
		return err
	}
	if err := validateSHA256(run.ForgePacket.SHA256, "forge_packet"); err != nil {
		return err
	}
	actual, err := fileSHA256(run.ForgePacket.Path)
	if err != nil {
		return fmt.Errorf("forge_packet: %w", err)
	}
	if actual != run.ForgePacket.SHA256 {
		return fmt.Errorf("forge_packet sha256 mismatch: expected %s got %s", run.ForgePacket.SHA256, actual)
	}
	if len(run.Evidence) == 0 {
		return errors.New("evidence must not be empty")
	}
	for _, evidence := range run.Evidence {
		if err := validateRunEvidence(evidence); err != nil {
			return err
		}
	}
	if len(run.Decisions) == 0 {
		return errors.New("decisions must include Covenant policy decision evidence")
	}
	for _, decision := range run.Decisions {
		if decision.DecisionID == "" || decision.Target == "" || decision.Decision == "" || decision.Explanation == "" {
			return errors.New("decisions require decision_id, target, decision, and explanation")
		}
	}
	return nil
}

func validateForgePacket(packet ForgePacket) error {
	if packet.SchemaVersion != forgePacketSchema {
		return fmt.Errorf("packet schema_version must be %s", forgePacketSchema)
	}
	if !allowedRunStatus(packet.Status) && packet.Status != "denied" {
		return fmt.Errorf("packet has invalid status %q", packet.Status)
	}
	if packet.Objective.Text == "" || packet.Objective.Workspace == "" {
		return errors.New("packet objective.text and objective.workspace are required")
	}
	if packet.FactoryPlan.PlanID == "" {
		return errors.New("packet factory_plan.plan_id is required")
	}
	if len(packet.PolicyDecisions) == 0 {
		return errors.New("packet policy_decisions must include Covenant decision evidence")
	}
	if len(packet.Evidence) == 0 {
		return errors.New("packet evidence must not be empty")
	}
	for _, evidence := range packet.Evidence {
		if err := validateRunEvidence(evidence); err != nil {
			return err
		}
	}
	return nil
}

func validateRunEvidence(evidence RunEvidenceRef) error {
	if evidence.Label == "" || evidence.Path == "" || evidence.SHA256 == "" || evidence.SchemaVersion == "" || evidence.Status == "" {
		return errors.New("evidence entries require label, path, sha256, schema_version, and status")
	}
	if err := validateEvidencePath(evidence.Path); err != nil {
		return err
	}
	if err := validateSHA256(evidence.SHA256, "evidence "+evidence.Label); err != nil {
		return err
	}
	actual, err := fileSHA256(evidence.Path)
	if err != nil {
		return fmt.Errorf("evidence %q: %w", evidence.Label, err)
	}
	if actual != evidence.SHA256 {
		return fmt.Errorf("evidence %q sha256 mismatch: expected %s got %s", evidence.Label, evidence.SHA256, actual)
	}
	return nil
}

func allowedRole(role string) bool {
	switch role {
	case "operations-factory", "factory-brain", "operator-command", "execution-engine", "evidence-observer", "policy-kernel", "agent-orchestrator", "workflow-conductor":
		return true
	default:
		return false
	}
}

func allowedPriority(priority string) bool {
	switch priority {
	case "low", "normal", "high":
		return true
	default:
		return false
	}
}

func allowedTaskState(state string) bool {
	switch state {
	case "queued", "planned", "delegated", "verifying", "passed", "blocked", "failed":
		return true
	default:
		return false
	}
}

func allowedGoalPhase(phase string) bool {
	switch phase {
	case "planning", "implementation", "verification", "blocked", "backoff", "complete", "stopped":
		return true
	default:
		return false
	}
}

func allowedRunStatus(status string) bool {
	switch status {
	case "queued", "planned", "delegated", "verifying", "passed", "blocked", "failed":
		return true
	default:
		return false
	}
}

func terminalGoalPhase(phase string) bool {
	return phase == "complete" || phase == "stopped"
}

func validateNextActionGuard(guard string) error {
	normalized := strings.ToLower(guard)
	if !strings.Contains(normalized, "ao forge") {
		return errors.New("next_action_guard must require AO Forge delegation")
	}
	for _, forbidden := range []string{"run provider directly", "execute provider directly", "push", "tag", "publish", "upload", "credential", "sibling repositor"} {
		if strings.Contains(normalized, forbidden) {
			return fmt.Errorf("next_action_guard contains forbidden action %q", forbidden)
		}
	}
	return nil
}

func taskTargetsRegistered(task Task, registry Registry) error {
	registered := map[string]bool{}
	for _, repo := range registry.Repos {
		registered[repo.ID] = true
	}
	var missing []string
	for _, id := range task.TargetRepos {
		if !registered[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("target repos are not registered: %s", strings.Join(missing, ", "))
	}
	return nil
}

func readinessCounts(registry Registry) (ready int, blocked int) {
	for _, repo := range registry.Repos {
		status := "unknown"
		if len(repo.ReadinessSignals) > 0 {
			status = repo.ReadinessSignals[0].Status
		}
		switch status {
		case "ready":
			ready++
		case "blocked":
			blocked++
		}
	}
	return ready, blocked
}

func firstDelegation(task Task) string {
	for _, delegation := range task.RequiredDelegation {
		if delegation.DelegateTo != "" {
			if delegation.DelegateTo == "ao-forge" {
				return "AO Forge"
			}
			return delegation.DelegateTo
		}
	}
	return "AO Forge"
}

func buildForgeBrief(registry Registry, task Task) (ForgeBrief, error) {
	target, err := primaryTargetRepo(registry, task)
	if err != nil {
		return ForgeBrief{}, err
	}
	return ForgeBrief{
		SchemaVersion: "ao.forge.factory-brief.v0.1",
		Objective: ForgeObjective{
			Text:        task.Objective,
			Workspace:   target.Workspace,
			ReleaseMode: false,
		},
		Constraints: ForgeConstraints{
			LocalFirst:                  true,
			AllowNetwork:                false,
			AllowReleaseMutation:        false,
			RequireControlPlaneReadback: false,
		},
		ExpectedWorkcells: []ForgeWorkcell{
			{
				WorkcellID: "foundry-" + task.TaskID + "-execute",
				Kind:       "execute",
				Workspace:  target.Workspace,
				Executor:   "ao2",
				Task:       task.Objective,
				MaxRepairs: 1,
				DependsOn:  []string{},
			},
			{
				WorkcellID: "foundry-" + task.TaskID + "-verify",
				Kind:       "verify",
				Workspace:  target.Workspace,
				Executor:   "ao2",
				Task:       strings.Join(task.Verification, "; "),
				MaxRepairs: 0,
				DependsOn:  []string{"foundry-" + task.TaskID + "-execute"},
			},
		},
		ExpectedEvidence: []string{
			"AO Forge factory packet",
			"AO Foundry verification output",
			"public-safety scan",
		},
	}, nil
}

func primaryTargetRepo(registry Registry, task Task) (Repo, error) {
	if len(task.TargetRepos) == 0 {
		return Repo{}, errors.New("task has no target repositories")
	}
	for _, repo := range registry.Repos {
		if repo.ID == task.TargetRepos[0] {
			if repo.Workspace == "" {
				return Repo{}, fmt.Errorf("repo %q has no workspace", repo.ID)
			}
			return repo, nil
		}
	}
	return Repo{}, fmt.Errorf("target repo %q is not registered", task.TargetRepos[0])
}

func buildReadinessAudit(registryPath, taskPath string) (ReadinessAudit, error) {
	registry, registryErr := loadRegistry(registryPath)
	task, taskErr := loadTask(taskPath)

	audit := ReadinessAudit{
		SchemaVersion: readinessSchema,
		Status:        "blocked",
		MaxScore:      100,
		RegistryID:    registry.FoundryID,
		TaskID:        task.TaskID,
	}

	audit.Checks = append(audit.Checks, readinessCheck("registry_valid", registryErr == nil, errReason(registryErr, "registry contract is valid")))
	audit.Checks = append(audit.Checks, readinessCheck("task_valid", taskErr == nil, errReason(taskErr, "task contract is valid")))
	if registryErr != nil || taskErr != nil {
		audit.finalize()
		return audit, nil
	}

	targetsErr := taskTargetsRegistered(task, registry)
	audit.Checks = append(audit.Checks, readinessCheck("target_repos_registered", targetsErr == nil, errReason(targetsErr, "all target repositories are registered")))

	readyErr := targetReposReady(task, registry)
	audit.Checks = append(audit.Checks, readinessCheck("target_repos_ready", readyErr == nil, errReason(readyErr, "all target repository readiness signals are ready")))

	delegationErr := forgeDelegationReady(task)
	audit.Checks = append(audit.Checks, readinessCheck("forge_delegation_and_local_safety", delegationErr == nil, errReason(delegationErr, "task delegates governed work to AO Forge and remains local-only")))

	audit.finalize()
	return audit, nil
}

func renderActiveStackReadinessSnapshot(ledgerPath string, ledger ActiveStackReadinessLedger) string {
	var b strings.Builder
	b.WriteString("<!-- foundry:active-stack-readiness:start -->\n")
	fmt.Fprintf(&b, "Last local sweep: %s.\n\n", ledger.LastSweepDate)
	b.WriteString("| Repository | Current status | Verification evidence |\n")
	b.WriteString("| --- | --- | --- |\n")
	for _, repo := range ledger.Repositories {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", escapeMarkdownCell(repo.Name), titleStatus(repo.Status), escapeMarkdownCell(formatReadinessEvidence(repo)))
	}
	if len(ledger.ReleaseHandoff.Gates) > 0 {
		b.WriteString("\n")
		b.WriteString("Release handoff gates:\n\n")
		b.WriteString("| Gate | Current status | Required before promotion | Evidence |\n")
		b.WriteString("| --- | --- | --- | --- |\n")
		for _, gate := range ledger.ReleaseHandoff.Gates {
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", escapeMarkdownCell(gate.Name), titleStatus(gate.Status), boolStatus(gate.RequiredBeforePromotion), escapeMarkdownCell(formatEvidenceItems(gate.Evidence)))
		}
	}
	b.WriteString("\n")
	b.WriteString("The machine-readable source for this snapshot is\n")
	fmt.Fprintf(&b, "[`%s`](%s).\n", ledgerPath, ledgerPath)
	b.WriteString("The AO2 active-spine release candidate ledger is\n")
	b.WriteString("[`examples/readiness/active-spine-release-candidate.ledger.json`](examples/readiness/active-spine-release-candidate.ledger.json).\n")
	b.WriteString("<!-- foundry:active-stack-readiness:end -->\n")
	return b.String()
}

func formatReadinessEvidence(repo ActiveStackReadinessRepository) string {
	evidence := make([]string, 0, len(repo.VerificationEvidence)+1)
	for _, item := range repo.VerificationEvidence {
		evidence = append(evidence, formatEvidenceItem(item))
	}
	if repo.CI != nil && repo.CI.RunID != "" {
		evidence = append(evidence, "main CI run `"+repo.CI.RunID+"`")
	}
	return strings.Join(evidence, ", ")
}

func formatEvidenceItems(items []string) string {
	evidence := make([]string, 0, len(items))
	for _, item := range items {
		evidence = append(evidence, formatEvidenceItem(item))
	}
	return strings.Join(evidence, ", ")
}

func boolStatus(value bool) string {
	if value {
		return "Yes"
	}
	return "No"
}

func formatEvidenceItem(item string) string {
	switch {
	case strings.HasPrefix(item, "go "),
		strings.HasPrefix(item, "npm "),
		strings.HasPrefix(item, "cargo "),
		strings.HasPrefix(item, "python "),
		strings.HasPrefix(item, "python3 "),
		strings.HasPrefix(item, "forge "),
		strings.HasPrefix(item, "covenant "),
		strings.HasPrefix(item, "docs/"),
		strings.HasPrefix(item, "examples/"):
		return "`" + item + "`"
	default:
		return item
	}
}

func titleStatus(status string) string {
	if status == "" {
		return ""
	}
	status = strings.ReplaceAll(status, "_", " ")
	words := strings.Fields(status)
	for i, word := range words {
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}

func escapeMarkdownCell(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}

func readinessCheck(name string, pass bool, reason string) ReadinessCheck {
	if pass {
		return ReadinessCheck{Name: name, Status: "pass", Score: 20, Reason: reason}
	}
	return ReadinessCheck{Name: name, Status: "fail", Score: 0, Reason: reason}
}

func buildGoalReadinessAudit(goalPath, registryPath, taskPath string) (GoalReadinessAudit, error) {
	goal, goalErr := loadGoalRun(goalPath)
	audit := GoalReadinessAudit{
		SchemaVersion: goalReadinessSchema,
		Status:        "blocked",
		MaxScore:      100,
		GoalID:        goal.GoalID,
	}

	audit.Checks = append(audit.Checks, readinessCheck("goal_run_valid", goalErr == nil, errReason(goalErr, "GoalRun contract is valid")))
	if goalErr != nil {
		audit.finalize()
		return audit, nil
	}

	phaseErr := nonTerminalGoalPhase(goal)
	audit.Checks = append(audit.Checks, readinessCheck("goal_phase_active", phaseErr == nil, errReason(phaseErr, "GoalRun phase is active")))

	pathErr := verifyEvidencePaths(goal)
	audit.Checks = append(audit.Checks, readinessCheck("evidence_paths_durable", pathErr == nil, errReason(pathErr, "GoalRun evidence paths are durable and public-safe")))

	hashErr := verifyEvidenceHashes(goal)
	audit.Checks = append(audit.Checks, readinessCheck("evidence_hashes_match", hashErr == nil, errReason(hashErr, "GoalRun evidence digests match referenced files")))

	readinessAudit, readinessErr := buildReadinessAudit(registryPath, taskPath)
	if readinessErr == nil && readinessAudit.Score != readinessAudit.MaxScore {
		readinessErr = fmt.Errorf("production readiness is %d/%d", readinessAudit.Score, readinessAudit.MaxScore)
	}
	audit.Checks = append(audit.Checks, readinessCheck("production_readiness_ready", readinessErr == nil, errReason(readinessErr, "registry/task production readiness is 100")))

	audit.finalize()
	return audit, nil
}

func (audit *GoalReadinessAudit) finalize() {
	score := 0
	next := []string{}
	for _, check := range audit.Checks {
		score += check.Score
		if check.Status != "pass" {
			next = append(next, check.Reason)
		}
	}
	audit.Score = score
	if score == audit.MaxScore {
		audit.Status = "ready"
	} else {
		audit.Status = "blocked"
	}
	audit.NextActions = next
}

func nonTerminalGoalPhase(goal GoalRun) error {
	if terminalGoalPhase(goal.CurrentPhase) {
		return fmt.Errorf("goal phase %q is terminal", goal.CurrentPhase)
	}
	return nil
}

func verifyEvidencePaths(goal GoalRun) error {
	for _, evidence := range goal.LastIteration.Evidence {
		if err := validateEvidencePath(evidence.Path); err != nil {
			return err
		}
	}
	return nil
}

func validateEvidencePath(path string) error {
	cleaned := strings.ReplaceAll(path, "\\", "/")
	if cleaned == "" ||
		strings.HasPrefix(cleaned, "/") ||
		strings.HasPrefix(cleaned, "~/") ||
		strings.HasPrefix(cleaned, "../") ||
		strings.Contains(cleaned, "/../") ||
		cleaned == "tmp" ||
		strings.HasPrefix(cleaned, "tmp/") ||
		isWindowsAbsolutePath(cleaned) {
		return fmt.Errorf("unsafe evidence path %q", path)
	}
	return nil
}

func validateAtlasRunLink(link AtlasRunLink) error {
	if link.ContractVersion != atlasRunLinkSchema {
		return fmt.Errorf("contract_version must be %s", atlasRunLinkSchema)
	}
	if strings.TrimSpace(link.TaskID) == "" {
		return errors.New("run-link task_id is required")
	}
	if err := validateAtlasPublicString(link.TaskID); err != nil {
		return fmt.Errorf("task_id: %w", err)
	}
	if link.Status != "completed" {
		return errors.New("run-link status must be completed")
	}
	if len(link.Evidence) == 0 {
		return errors.New("run-link evidence must not be empty")
	}
	for label, path := range link.Evidence {
		if strings.TrimSpace(label) == "" {
			return errors.New("run-link evidence labels must not be empty")
		}
		if err := validateAtlasPublicString(label); err != nil {
			return fmt.Errorf("evidence label: %w", err)
		}
		if err := validateEvidencePath(path); err != nil {
			return fmt.Errorf("evidence %s: %w", label, err)
		}
	}
	if !strings.HasPrefix(link.Digest, "sha256:") {
		return errors.New("run-link digest must start with sha256:")
	}
	if err := validateSHA256(strings.TrimPrefix(link.Digest, "sha256:"), "run-link digest"); err != nil {
		return err
	}
	return nil
}

func buildAtlasReadback(importPath, runLinkPath string) (AtlasReadback, error) {
	artifact, err := loadAtlasFoundryImport(importPath)
	if err != nil {
		return AtlasReadback{}, err
	}
	link, err := loadAtlasRunLink(runLinkPath)
	if err != nil {
		return AtlasReadback{}, err
	}
	var matched *AtlasImportTaskFixture
	for i := range artifact.Tasks {
		if artifact.Tasks[i].TaskID == link.TaskID {
			matched = &artifact.Tasks[i]
			break
		}
	}
	if matched == nil {
		return AtlasReadback{}, fmt.Errorf("no matching Atlas import task for run-link task_id %q", link.TaskID)
	}
	return AtlasReadback{
		SchemaVersion:  atlasReadbackSchema,
		Status:         "ready",
		Mode:           "fixture_only_readback",
		AtlasImportID:  artifact.ID,
		WorkgraphID:    artifact.WorkgraphID,
		TargetInstance: artifact.TargetInstance,
		TaskID:         link.TaskID,
		TaskDigest:     matched.TaskDigest,
		RunLinkDigest:  link.Digest,
		Evidence:       link.Evidence,
		SchedulesWork:  false,
		ExecutesWork:   false,
		ApprovesWork:   false,
		NextActions:    []string{"keep Atlas scheduling and execution outside Foundry readback"},
	}, nil
}

func buildAtlasStatus(registryPath, importPath, runLinkPath string) (AtlasStatus, error) {
	registry, err := loadRegistry(registryPath)
	if err != nil {
		return AtlasStatus{}, err
	}
	readback, err := buildAtlasReadback(importPath, runLinkPath)
	if err != nil {
		return AtlasStatus{}, err
	}
	return AtlasStatus{
		SchemaVersion:  atlasStatusSchema,
		Status:         "ready",
		Mode:           "fixture_only_readback",
		RegistryID:     registry.FoundryID,
		ImportID:       readback.AtlasImportID,
		WorkgraphID:    readback.WorkgraphID,
		TargetInstance: readback.TargetInstance,
		ReadbackStatus: readback.Status,
		TaskID:         readback.TaskID,
		TaskDigest:     readback.TaskDigest,
		RunLinkDigest:  readback.RunLinkDigest,
		SchedulesWork:  false,
		ExecutesWork:   false,
		ApprovesWork:   false,
		Evidence:       readback.Evidence,
		NextActions:    []string{"keep Atlas status as observer-only readback"},
	}, nil
}

func validateAtlasPublicString(value string) error {
	normalized := strings.ReplaceAll(value, "\\", "/")
	lower := strings.ToLower(normalized)
	if normalized == "" ||
		strings.HasPrefix(normalized, "/") ||
		strings.HasPrefix(normalized, "~/") ||
		strings.HasPrefix(normalized, "../") ||
		strings.Contains(normalized, "/../") ||
		isWindowsAbsolutePath(normalized) {
		return fmt.Errorf("unsafe Atlas value %q", value)
	}
	for _, marker := range []string{
		"/" + "users/",
		"/" + "home/",
		"/" + "tmp/",
		"downloads" + "/",
		"file:" + "//",
		"api" + "_key",
		"api" + "-key",
		"access" + "_token",
		"access" + "-token",
	} {
		if strings.Contains(lower, marker) {
			return fmt.Errorf("unsafe Atlas value %q", value)
		}
	}
	return nil
}

func digestAtlasFactoryTask(task AtlasFactoryTask) string {
	data, err := json.Marshal(task)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return "sha256:" + fmt.Sprintf("%x", sum[:])
}

func equalStringSlices(left, right []string) bool {
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

func sameCleanPath(left, right string) bool {
	if strings.TrimSpace(left) == "" || strings.TrimSpace(right) == "" {
		return false
	}
	return filepath.Clean(filepath.FromSlash(left)) == filepath.Clean(filepath.FromSlash(right))
}

func isWindowsAbsolutePath(path string) bool {
	return len(path) >= 3 && ((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) && path[1] == ':' && path[2] == '/'
}

func validateSHA256(value, label string) error {
	if len(value) != 64 {
		return fmt.Errorf("%s sha256 must be 64 lowercase hex characters", label)
	}
	for _, c := range value {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return fmt.Errorf("%s sha256 must be 64 lowercase hex characters", label)
		}
	}
	return nil
}

func verifyEvidenceHashes(goal GoalRun) error {
	root, err := repoRoot()
	if err != nil {
		return err
	}
	for _, evidence := range goal.LastIteration.Evidence {
		data, err := os.ReadFile(root + string(os.PathSeparator) + evidence.Path)
		if err != nil {
			return fmt.Errorf("read evidence %q: %w", evidence.Path, err)
		}
		sum := sha256.Sum256(data)
		actual := fmt.Sprintf("%x", sum[:])
		if actual != evidence.SHA256 {
			return fmt.Errorf("evidence %q sha256 mismatch", evidence.Label)
		}
	}
	return nil
}

func buildFoundryRun(registryPath, taskPath, packetPath string) (FoundryRun, error) {
	registry, err := loadRegistry(registryPath)
	if err != nil {
		return FoundryRun{}, fmt.Errorf("registry: %w", err)
	}
	task, err := loadTask(taskPath)
	if err != nil {
		return FoundryRun{}, fmt.Errorf("task: %w", err)
	}
	if err := taskTargetsRegistered(task, registry); err != nil {
		return FoundryRun{}, fmt.Errorf("task: %w", err)
	}
	packet, packetData, err := loadForgePacket(packetPath)
	if err != nil {
		return FoundryRun{}, err
	}
	if err := packetMapsToTask(packet, registry, task); err != nil {
		return FoundryRun{}, err
	}
	packetSum := sha256.Sum256(packetData)
	run := FoundryRun{
		SchemaVersion: runSchema,
		RunID:         "foundry-run-" + task.TaskID + "-" + shortSHA256(fmt.Sprintf("%s:%s:%s", registry.FoundryID, task.TaskID, packet.FactoryPlan.PlanID)),
		TaskID:        task.TaskID,
		RegistryID:    registry.FoundryID,
		Status:        packet.Status,
		DelegatedTo:   "ao-forge",
		ForgePacket: RunPacketRef{
			Path:   packetPath,
			SHA256: fmt.Sprintf("%x", packetSum[:]),
			Status: packet.Status,
		},
		Evidence:    append([]RunEvidenceRef(nil), packet.Evidence...),
		Decisions:   append([]RunDecision(nil), packet.PolicyDecisions...),
		NextActions: append([]RunNextAction(nil), packet.NextActions...),
	}
	if err := validateFoundryRun(run); err != nil {
		return FoundryRun{}, err
	}
	return run, nil
}

func buildApprovalRequest(taskPath string) (ApprovalRequest, error) {
	task, err := loadTask(taskPath)
	if err != nil {
		return ApprovalRequest{}, err
	}
	sum, err := fileSHA256(taskPath)
	if err != nil {
		return ApprovalRequest{}, err
	}
	effects := requestedSideEffects(task)
	return ApprovalRequest{
		SchemaVersion:        "ao.foundry.approval-request.v0.1",
		TaskID:               task.TaskID,
		TaskSHA256:           sum,
		RequestedSideEffects: effects,
		Reason:               "Human approval is required before AO Foundry may delegate non-local side effects.",
	}, nil
}

func approvalReady(taskPath string, task Task, decisionPath string) error {
	if len(requestedSideEffects(task)) == 0 {
		return nil
	}
	if decisionPath == "" {
		return errors.New("non-local side effects require --approval-decision")
	}
	return validateApprovalDecision(decisionPath, taskPath)
}

func validateApprovalDecision(decisionPath, taskPath string) error {
	if decisionPath == "" {
		return errors.New("missing --decision")
	}
	task, err := loadTask(taskPath)
	if err != nil {
		return err
	}
	var decision ApprovalDecision
	if err := readJSONFile(decisionPath, &decision); err != nil {
		return err
	}
	if decision.SchemaVersion != "ao.foundry.approval-decision.v0.1" {
		return errors.New("invalid approval decision schema_version")
	}
	if decision.Decision != "approved" {
		return errors.New("approval decision is not approved")
	}
	if strings.TrimSpace(decision.Operator) == "" || strings.EqualFold(decision.Operator, "ao-foundry") {
		return errors.New("approval requires an external operator identity")
	}
	if strings.TrimSpace(decision.Reason) == "" {
		return errors.New("approval reason is required")
	}
	expires, err := time.Parse(time.RFC3339, decision.ExpiresAtUTC)
	if err != nil {
		return fmt.Errorf("invalid approval expiration: %w", err)
	}
	if time.Now().UTC().After(expires) {
		return errors.New("approval expired")
	}
	taskSum, err := fileSHA256(taskPath)
	if err != nil {
		return err
	}
	if decision.TaskID != task.TaskID || decision.TaskSHA256 != taskSum {
		return errors.New("task digest mismatch")
	}
	requested := requestedSideEffects(task)
	if !sameStringSet(decision.RequestedSideEffects, requested) {
		return errors.New("approval requested side effects do not match task")
	}
	if !subsetStringSet(decision.ApprovedSideEffects, requested) {
		return errors.New("approval cannot broaden allowed side effects beyond request")
	}
	if !sameStringSet(decision.ApprovedSideEffects, requested) {
		return errors.New("approval does not cover all requested side effects")
	}
	return nil
}

func requestedSideEffects(task Task) []string {
	set := map[string]bool{}
	if !task.Safety.LocalOnly {
		set["non-local execution"] = true
	}
	if task.Safety.AllowNetwork {
		set["network access"] = true
	}
	if task.Safety.AllowReleaseMutation {
		set["release mutation"] = true
	}
	for _, root := range task.Safety.AllowedWriteRoots {
		if root != "../ao-foundry" {
			set["cross-repo write: "+root] = true
		}
	}
	effects := make([]string, 0, len(set))
	for effect := range set {
		effects = append(effects, effect)
	}
	sort.Strings(effects)
	return effects
}

func sameStringSet(a, b []string) bool {
	return subsetStringSet(a, b) && subsetStringSet(b, a)
}

func subsetStringSet(subset, superset []string) bool {
	allowed := map[string]bool{}
	for _, item := range superset {
		allowed[item] = true
	}
	for _, item := range subset {
		if !allowed[item] {
			return false
		}
	}
	return true
}

func buildRepoHealthReport(registryPath, repoID string) (RepoHealthReport, error) {
	registry, err := loadRegistry(registryPath)
	if err != nil {
		return RepoHealthReport{}, err
	}
	repos := registry.Repos
	if repoID != "" {
		var found []Repo
		for _, repo := range registry.Repos {
			if repo.ID == repoID {
				found = append(found, repo)
				break
			}
		}
		if len(found) == 0 {
			return RepoHealthReport{}, fmt.Errorf("repo id %q is not registered", repoID)
		}
		repos = found
	}
	report := RepoHealthReport{
		SchemaVersion: repoHealthSchema,
		RegistryID:    registry.FoundryID,
		Status:        "ready",
		Repos:         make([]RepoHealth, 0, len(repos)),
		NextActions:   []string{},
	}
	for _, repo := range repos {
		health := readRepoHealth(repo)
		report.Repos = append(report.Repos, health)
		if health.Status == "blocked" {
			report.Status = "blocked"
			report.NextActions = append(report.NextActions, health.NextActions...)
		}
	}
	return report, nil
}

func buildRepoBoard(registryPath string) (RepoBoard, error) {
	registry, err := loadRegistry(registryPath)
	if err != nil {
		return RepoBoard{}, err
	}
	healthReport, err := buildRepoHealthReport(registryPath, "")
	if err != nil {
		return RepoBoard{}, err
	}
	healthByRepo := map[string]RepoHealth{}
	for _, health := range healthReport.Repos {
		healthByRepo[health.RepoID] = health
	}
	board := RepoBoard{
		SchemaVersion: repoBoardSchema,
		RegistryID:    registry.FoundryID,
		Status:        "ready",
		Repos:         make([]RepoBoardEntry, 0, len(registry.Repos)),
		NextActions:   []string{},
	}
	for _, repo := range registry.Repos {
		health := healthByRepo[repo.ID]
		if !healthConfigured(repo.Health) {
			health = readRepoHealth(repoWithBoardHealth(repo))
		}
		entry := classifyRepoBoardEntry(repo, health)
		board.Repos = append(board.Repos, entry)
		board.NextActions = append(board.NextActions, entry.NextActions...)
		if entry.Tier == "blocked-hygiene" {
			board.Status = "blocked"
		}
	}
	if len(board.NextActions) == 0 {
		board.NextActions = append(board.NextActions, "advance active-spine repos; freeze or archive demotion candidates before expanding scope")
	}
	return board, nil
}

func classifyRepoBoardEntry(repo Repo, health RepoHealth) RepoBoardEntry {
	tier := repoBoardTier(repo)
	recommendation := repoBoardRecommendation(tier)
	nextActions := repoBoardNextActions(repo, tier)
	if health.Status == "blocked" {
		tier = "blocked-hygiene"
		recommendation = "clean-worktree"
		nextActions = append([]string{}, health.NextActions...)
		if len(nextActions) == 0 {
			nextActions = append(nextActions, "clear local hygiene blockers before strategy work")
		}
	}
	return RepoBoardEntry{
		RepoID:         repo.ID,
		Name:           repo.Name,
		Role:           repo.Role,
		Tier:           tier,
		Workspace:      repo.Workspace,
		HealthStatus:   health.Status,
		CurrentBranch:  health.CurrentBranch,
		Recommendation: recommendation,
		NextActions:    nextActions,
	}
}

func repoBoardTier(repo Repo) string {
	switch repo.ID {
	case "ao2", "ao-forge", "ao-covenant", "ao2-control-plane", "codex-cron", "ao-foundry":
		return "active-spine"
	case "ao-command", "financial-services-profile", "secure-agent-profile":
		return "supporting"
	case "ao-operator", "ao-runtime", "ao-control-plane", "ao-conductor", "agy-swarms", "ai-teams", "ao-covenant-stub-20260617", "memory-ext":
		return "candidate-demote"
	default:
		switch repo.Role {
		case "execution-engine", "factory-brain", "policy-kernel", "evidence-observer", "operations-factory", "scheduler":
			return "active-spine"
		case "agent-orchestrator", "workflow-conductor":
			return "candidate-demote"
		default:
			return "supporting"
		}
	}
}

func repoBoardRecommendation(tier string) string {
	switch tier {
	case "active-spine":
		return "advance"
	case "candidate-demote":
		return "freeze-or-archive"
	case "blocked-hygiene":
		return "clean-worktree"
	default:
		return "hold-supporting"
	}
}

func repoBoardNextActions(repo Repo, tier string) []string {
	switch tier {
	case "active-spine":
		return []string{fmt.Sprintf("%s: keep in the active Foundry spine and maintain release/security evidence", repo.ID)}
	case "candidate-demote":
		if repo.ID == "agy-swarms" {
			return []string{"agy-swarms: archived for active AO spine work; use docs/archive-handoff.md as reference and do not add new product scope"}
		}
		if repo.ID == "ao-conductor" {
			return []string{"ao-conductor: archived for active AO spine work; use docs/archive-handoff.md as reference and route new orchestration through AO Forge and AO2"}
		}
		if repo.ID == "ao-operator" {
			return []string{"ao-operator: deprecated for active AO work; use Foundry deprecation record as reference and route execution/control-plane work to ao2 and ao2-control-plane"}
		}
		if repo.ID == "ao-runtime" {
			return []string{"ao-runtime: deprecated with ao-operator; route execution work to ao2"}
		}
		if repo.ID == "ao-control-plane" {
			return []string{"ao-control-plane: deprecated with ao-operator; route typed state and evidence work to ao2-control-plane"}
		}
		return []string{fmt.Sprintf("%s: freeze, archive, or extract unique ideas before further AO spine work", repo.ID)}
	default:
		if repo.ID == "ao-command" {
			return []string{"ao-command: keep as the read-only operator/readback surface for ao-forge, ao2, ao2-control-plane, and ao-covenant; do not route archived or subscription-backed scope through it"}
		}
		return []string{fmt.Sprintf("%s: keep supporting, but do not expand until the active spine is clean", repo.ID)}
	}
}

func repoWithBoardHealth(repo Repo) Repo {
	repo.Health = HealthReaderConfig{
		RequireCleanWorktree: true,
		VerificationCommands: []string{"git status"},
		AllowNetworkRead:     false,
		GitHubActions:        false,
	}
	return repo
}

func buildEvalResult(runPath, scorecardPath string) (EvalResult, error) {
	run, err := loadFoundryRun(runPath)
	if err != nil {
		return EvalResult{}, err
	}
	scorecard, err := loadEvalScorecard(scorecardPath)
	if err != nil {
		return EvalResult{}, err
	}
	result := EvalResult{
		SchemaVersion: "ao.foundry.eval-result.v0.1",
		ScorecardID:   scorecard.ScorecardID,
		RunID:         run.RunID,
		Status:        "ready",
		Threshold:     scorecard.Threshold,
		Dimensions:    []EvalDimension{},
		NextActions:   []string{},
	}
	for _, def := range scorecard.Dimensions {
		dim := scoreDimension(def, run)
		result.Dimensions = append(result.Dimensions, dim)
		result.Score += dim.Score
		result.MaxScore += dim.MaxScore
		if dim.Status != "pass" {
			result.NextActions = append(result.NextActions, dim.Reason)
		}
	}
	if result.Score < result.Threshold || len(result.NextActions) > 0 {
		result.Status = "blocked"
	}
	return result, nil
}

func buildRSIImprovementGate(baselinePath, candidatePath string, requiredImprovement float64) (RSIImprovementGate, error) {
	if baselinePath == "" || candidatePath == "" {
		return RSIImprovementGate{}, errors.New("baseline and candidate are required")
	}
	if requiredImprovement <= 0 {
		return RSIImprovementGate{}, errors.New("min-improvement must be greater than zero")
	}
	baseline, err := loadEvalResultForImprovement("baseline", baselinePath)
	if err != nil {
		return RSIImprovementGate{}, err
	}
	candidate, err := loadEvalResultForImprovement("candidate", candidatePath)
	if err != nil {
		return RSIImprovementGate{}, err
	}
	baselinePercent := scorePercent(baseline)
	candidatePercent := scorePercent(candidate)
	actualImprovement := roundPercent(candidatePercent - baselinePercent)
	status := "passed"
	nextActions := []string{}
	if actualImprovement < requiredImprovement {
		status = "blocked"
		nextActions = append(nextActions, "produce a candidate eval result that improves by at least the required percentage points")
	}
	baselineHash, err := fileSHA256(baselinePath)
	if err != nil {
		return RSIImprovementGate{}, fmt.Errorf("hash baseline evidence: %w", err)
	}
	candidateHash, err := fileSHA256(candidatePath)
	if err != nil {
		return RSIImprovementGate{}, fmt.Errorf("hash candidate evidence: %w", err)
	}
	return RSIImprovementGate{
		SchemaVersion:              "ao.foundry.rsi-improvement-gate.v0.1",
		Status:                     status,
		BaselineScorePercent:       baselinePercent,
		CandidateScorePercent:      candidatePercent,
		RequiredImprovementPercent: requiredImprovement,
		ActualImprovementPercent:   actualImprovement,
		AutonomousClaim:            "measured_local_improvement",
		MutatesRepositories:        false,
		Evidence: []RSIImprovementProof{
			rsiImprovementProof("baseline", baselinePath, baseline, baselineHash),
			rsiImprovementProof("candidate", candidatePath, candidate, candidateHash),
		},
		NextActions: nextActions,
	}, nil
}

func loadEvalResultForImprovement(label, path string) (EvalResult, error) {
	var result EvalResult
	if err := readJSONFile(path, &result); err != nil {
		return EvalResult{}, fmt.Errorf("read %s eval result: %w", label, err)
	}
	if result.SchemaVersion != "ao.foundry.eval-result.v0.1" {
		return EvalResult{}, fmt.Errorf("%s eval result schema_version must be ao.foundry.eval-result.v0.1", label)
	}
	if result.Status != "ready" {
		return EvalResult{}, fmt.Errorf("%s eval result status must be ready", label)
	}
	if result.MaxScore <= 0 {
		return EvalResult{}, fmt.Errorf("%s eval result max_score must be greater than zero", label)
	}
	if result.Score < 0 || result.Score > result.MaxScore {
		return EvalResult{}, fmt.Errorf("%s eval result score must be between 0 and max_score", label)
	}
	return result, nil
}

func scorePercent(result EvalResult) float64 {
	return roundPercent(float64(result.Score) / float64(result.MaxScore) * 100)
}

func roundPercent(value float64) float64 {
	return math.Round(value*100) / 100
}

func rsiImprovementProof(label, path string, result EvalResult, hash string) RSIImprovementProof {
	return RSIImprovementProof{
		Label:         label,
		Path:          filepath.ToSlash(path),
		SchemaVersion: result.SchemaVersion,
		Status:        result.Status,
		Score:         result.Score,
		MaxScore:      result.MaxScore,
		SHA256:        hash,
	}
}

func importAO2SDDPlan(planPath string) (Task, error) {
	var plan AO2SDDPlan
	if planPath == "" {
		return Task{}, errors.New("missing --plan")
	}
	data, err := os.ReadFile(planPath)
	if err != nil {
		return Task{}, err
	}
	if err := json.Unmarshal(data, &plan); err != nil {
		return Task{}, err
	}
	if plan.SchemaVersion == "" || plan.PlanID == "" {
		return Task{}, errors.New("AO2 SDD plan requires schema_version and plan_id")
	}
	targetRepo := filepath.Base(filepath.Clean(filepath.FromSlash(plan.Target.RepoPath)))
	if targetRepo == "." || targetRepo == string(filepath.Separator) || targetRepo == "" {
		targetRepo = "ao-foundry"
	}
	acceptance := []string{}
	for _, step := range plan.Plan.Steps {
		acceptance = append(acceptance, step.Acceptance...)
	}
	if len(acceptance) == 0 {
		acceptance = []string{"Imported AO2 SDD plan has explicit acceptance criteria"}
	}
	title := plan.Plan.Title
	if title == "" {
		title = "Imported AO2 SDD plan"
	}
	objective := plan.Plan.Goal
	if objective == "" {
		objective = plan.Prompt.Text
	}
	task := Task{
		SchemaVersion: "ao.foundry.task.v0.1",
		TaskID:        "imported-" + strings.ToLower(strings.ReplaceAll(plan.PlanID, "_", "-")),
		Title:         title,
		Objective:     objective,
		Priority:      "normal",
		State:         "queued",
		TargetRepos:   []string{targetRepo},
		RequiredDelegation: []Delegation{{
			DelegateTo: "ao-forge",
			Reason:     "Imported SDD work must be delegated through AO Forge for governed execution.",
		}},
		Acceptance:   acceptance,
		Verification: []string{"go test ./..."},
		Safety: TaskSafety{
			LocalOnly:         true,
			AllowedWriteRoots: []string{"../ao-foundry"},
			ForbiddenActions:  []string{"push", "tag", "publish-release", "upload-artifacts", "credential-access"},
		},
	}
	if err := validateTask(task); err != nil {
		return Task{}, err
	}
	return task, nil
}

func buildDemoStatus(registryPath, taskPath, runPath string) (DemoStatus, error) {
	registry, err := loadRegistry(registryPath)
	if err != nil {
		return DemoStatus{}, err
	}
	task, err := loadTask(taskPath)
	if err != nil {
		return DemoStatus{}, err
	}
	run, err := loadFoundryRun(runPath)
	if err != nil {
		return DemoStatus{}, err
	}
	if run.TaskID != task.TaskID {
		return DemoStatus{}, errors.New("run task_id does not match task")
	}
	return DemoStatus{
		SchemaVersion: "ao.foundry.demo-status.v0.1",
		RegistryID:    registry.FoundryID,
		TaskID:        task.TaskID,
		RunID:         run.RunID,
		Status:        "ready",
		Story: []string{
			"Foundry knows the AO stack registry.",
			"Foundry validates the task and readiness gates.",
			"Foundry emits an AO Forge brief instead of executing directly.",
			"AO Forge governs execution and returns packet evidence.",
			"Foundry ingests the packet into a run record.",
			"Foundry scores the run with local evals.",
			"Foundry reports the next safe delegated action.",
		},
		NextAction: "delegate governed implementation to AO Forge",
	}, nil
}

func buildPulseBundle(registryPath, taskPath, goalPath, packetPath, scorecardPath, rsiBaselinePath string, rsiMinImprovement float64, outDir, forgeLivePacketPath, controlPlaneReceiptPath, signedSmokeResultPath string) (PulseEvent, error) {
	event := PulseEvent{
		SchemaVersion: pulseEventSchema,
		PulseID:       "pulse-" + shortSHA256(strings.Join([]string{registryPath, taskPath, goalPath, packetPath, scorecardPath, rsiBaselinePath, fmt.Sprintf("%.2f", rsiMinImprovement)}, ":")),
		Status:        "blocked",
		MaxScore:      100,
		Artifacts:     []PulseArtifact{},
		Checks:        []PulseCheck{},
		Freshness:     newPulseFreshnessSummary(),
		NextAction:    "resolve pulse blockers and rerun",
	}
	fail := func(name string, err error) (PulseEvent, error) {
		event.Checks = append(event.Checks, PulseCheck{Name: name, Status: "fail", Reason: err.Error()})
		return event, err
	}
	pass := func(name, reason string) {
		event.Checks = append(event.Checks, PulseCheck{Name: name, Status: "pass", Reason: reason})
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fail("output_directory", err)
	}

	registry, err := loadRegistry(registryPath)
	if err != nil {
		return fail("registry_valid", err)
	}
	event.RegistryID = registry.FoundryID
	pass("registry_valid", "registry contract is valid")

	task, err := loadTask(taskPath)
	if err != nil {
		return fail("task_valid", err)
	}
	event.TaskID = task.TaskID
	pass("task_valid", "task contract is valid")

	goal, err := loadGoalRun(goalPath)
	if err != nil {
		return fail("goal_valid", err)
	}
	event.GoalID = goal.GoalID
	pass("goal_valid", "GoalRun contract is valid")

	readinessAudit, err := buildReadinessAudit(registryPath, taskPath)
	if err != nil {
		return fail("production_readiness_audit", err)
	}
	readinessPath := filepath.Join(outDir, "production-readiness-audit.json")
	if err := writeJSONFile(readinessPath, readinessAudit); err != nil {
		return fail("production_readiness_audit", err)
	}
	if err := event.addArtifact("production_readiness_audit", readinessPath, readinessAudit.SchemaVersion, readinessAudit.Status); err != nil {
		return fail("production_readiness_artifact", err)
	}
	if readinessAudit.Score != readinessAudit.MaxScore {
		return fail("production_readiness_ready", fmt.Errorf("production readiness is %d/%d", readinessAudit.Score, readinessAudit.MaxScore))
	}
	pass("production_readiness_ready", "production readiness is 100/100")

	goalAudit, err := buildGoalReadinessAudit(goalPath, registryPath, taskPath)
	if err != nil {
		return fail("goal_readiness_audit", err)
	}
	goalReadinessPath := filepath.Join(outDir, "goal-readiness-audit.json")
	if err := writeJSONFile(goalReadinessPath, goalAudit); err != nil {
		return fail("goal_readiness_audit", err)
	}
	if err := event.addArtifact("goal_readiness_audit", goalReadinessPath, goalAudit.SchemaVersion, goalAudit.Status); err != nil {
		return fail("goal_readiness_artifact", err)
	}
	if goalAudit.Score != goalAudit.MaxScore {
		return fail("goal_readiness_ready", fmt.Errorf("goal readiness is %d/%d", goalAudit.Score, goalAudit.MaxScore))
	}
	pass("goal_readiness_ready", "goal readiness is 100/100")

	brief, err := buildForgeBrief(registry, task)
	if err != nil {
		return fail("forge_brief", err)
	}
	briefPath := filepath.Join(outDir, "forge-brief.json")
	if err := writeJSONFile(briefPath, brief); err != nil {
		return fail("forge_brief", err)
	}
	if err := event.addArtifact("forge_brief", briefPath, brief.SchemaVersion, "ready"); err != nil {
		return fail("forge_brief_artifact", err)
	}
	pass("forge_brief", "Forge brief emitted for delegated governed execution")

	packet, _, err := loadForgePacket(packetPath)
	if err != nil {
		return fail("forge_packet", err)
	}
	packetCopyPath := filepath.Join(outDir, "forge-packet.json")
	if err := writeJSONFile(packetCopyPath, packet); err != nil {
		return fail("forge_packet", err)
	}
	if err := event.addArtifact("forge_packet", packetCopyPath, forgePacketSchema, packet.Status); err != nil {
		return fail("forge_packet_artifact", err)
	}
	if packet.Status != "passed" {
		return fail("forge_packet_passed", fmt.Errorf("Forge packet status is %s", packet.Status))
	}
	pass("forge_packet_passed", "Forge packet is available as delegated execution evidence")

	gate := buildPolicyGateSummary(packet)
	gatePath := filepath.Join(outDir, "policy-gate.json")
	if err := writeJSONFile(gatePath, gate); err != nil {
		return fail("policy_gate", err)
	}
	if err := event.addArtifact("policy_gate", gatePath, gate.SchemaVersion, gate.Status); err != nil {
		return fail("policy_gate_artifact", err)
	}
	if gate.Status != "ready" {
		return fail("policy_gate_ready", errors.New(gate.Explanation))
	}
	pass("policy_gate_ready", "policy gate has no denying decisions")

	liveAttempt, err := buildForgeLiveAttempt(forgeLivePacketPath)
	if err != nil {
		return fail("forge_live_attempt", err)
	}
	liveAttemptPath := filepath.Join(outDir, "forge-live-attempt.json")
	if err := writeJSONFile(liveAttemptPath, liveAttempt); err != nil {
		return fail("forge_live_attempt", err)
	}
	if err := event.addArtifact("forge_live_attempt", liveAttemptPath, liveAttempt.SchemaVersion, liveAttempt.Status); err != nil {
		return fail("forge_live_attempt_artifact", err)
	}
	event.Freshness.setForgeLiveAttempt(liveAttempt)
	if liveAttempt.Status == "passed" {
		pass("forge_live_attempt", "operator-provided AO Forge live packet is bundled")
	} else if liveAttempt.Source != "not-provided" {
		return fail("forge_live_attempt", errors.New(liveAttempt.Explanation))
	} else {
		pass("forge_live_attempt", "AO Forge live execution is not attempted by the local public pulse")
	}

	readbackReceiptPath := controlPlaneReceiptPath
	if strings.TrimSpace(readbackReceiptPath) == "" {
		readbackReceiptPath = discoverControlPlaneReceiptPath(forgeLivePacketPath)
	}
	readback, err := buildControlPlaneReadback(readbackReceiptPath, forgeLivePacketPath)
	if err != nil {
		return fail("control_plane_readback", err)
	}
	readbackPath := filepath.Join(outDir, "control-plane-readback.json")
	if err := writeJSONFile(readbackPath, readback); err != nil {
		return fail("control_plane_readback", err)
	}
	if err := event.addArtifact("control_plane_readback", readbackPath, readback.SchemaVersion, readback.Status); err != nil {
		return fail("control_plane_readback_artifact", err)
	}
	event.Freshness.setControlPlaneReadback(readback)
	if readback.Status == "ready" {
		pass("control_plane_readback", "operator-provided control-plane readback is bundled")
	} else if readback.Status == "blocked" || readback.Status == "stale" {
		return fail("control_plane_readback", errors.New(readback.Explanation))
	} else {
		pass("control_plane_readback", "control-plane readback is unavailable in the local public pulse")
	}

	if strings.TrimSpace(signedSmokeResultPath) != "" {
		ingest, err := buildSignedSmokeIngest(signedSmokeResultPath)
		if err != nil {
			return fail("signed_smoke_ingest", err)
		}
		ingestPath := filepath.Join(outDir, "signed-smoke-ingest.json")
		if err := writeJSONFile(ingestPath, ingest); err != nil {
			return fail("signed_smoke_ingest", err)
		}
		if err := event.addArtifact("signed_smoke_ingest", ingestPath, ingest.SchemaVersion, ingest.Status); err != nil {
			return fail("signed_smoke_ingest_artifact", err)
		}
		pass("signed_smoke_ingest", "signed smoke result is validated and bundled")
	}

	run, err := buildFoundryRun(registryPath, taskPath, packetPath)
	if err != nil {
		return fail("foundry_run", err)
	}
	runPath := filepath.Join(outDir, "foundry-run.json")
	if err := writeJSONFile(runPath, run); err != nil {
		return fail("foundry_run", err)
	}
	if err := event.addArtifact("foundry_run", runPath, run.SchemaVersion, run.Status); err != nil {
		return fail("foundry_run_artifact", err)
	}
	if run.Status != "passed" {
		return fail("foundry_run_passed", fmt.Errorf("Foundry run status is %s", run.Status))
	}
	pass("foundry_run_passed", "Foundry run ingested AO Forge packet evidence")

	evalResult, err := buildEvalResult(runPath, scorecardPath)
	if err != nil {
		return fail("eval_result", err)
	}
	evalPath := filepath.Join(outDir, "eval-result.json")
	if err := writeJSONFile(evalPath, evalResult); err != nil {
		return fail("eval_result", err)
	}
	if err := event.addArtifact("eval_result", evalPath, evalResult.SchemaVersion, evalResult.Status); err != nil {
		return fail("eval_result_artifact", err)
	}
	if evalResult.Score < evalResult.Threshold {
		return fail("eval_threshold", fmt.Errorf("eval score is %d below threshold %d", evalResult.Score, evalResult.Threshold))
	}
	pass("eval_threshold", "eval score meets threshold")

	rsiCandidate, err := buildRSICandidate(rsiBaselinePath, evalPath)
	if err != nil {
		return fail("rsi_candidate", err)
	}
	rsiCandidatePath := filepath.Join(outDir, "rsi-candidate.json")
	if err := writeJSONFile(rsiCandidatePath, rsiCandidate); err != nil {
		return fail("rsi_candidate", err)
	}
	if err := event.addArtifact("rsi_candidate", rsiCandidatePath, rsiCandidate.SchemaVersion, rsiCandidate.Status); err != nil {
		return fail("rsi_candidate_artifact", err)
	}
	pass("rsi_candidate", "RSI candidate eval result was generated by the local pulse")

	rsiGate, err := buildRSIImprovementGate(rsiBaselinePath, evalPath, rsiMinImprovement)
	if err != nil {
		return fail("rsi_improvement_gate", err)
	}
	rsiGatePath := filepath.Join(outDir, "rsi-improvement-gate.json")
	if err := writeJSONFile(rsiGatePath, rsiGate); err != nil {
		return fail("rsi_improvement_gate", err)
	}
	if err := event.addArtifact("rsi_improvement_gate", rsiGatePath, rsiGate.SchemaVersion, rsiGate.Status); err != nil {
		return fail("rsi_improvement_gate_artifact", err)
	}
	if rsiGate.Status != "passed" {
		return fail("rsi_improvement_gate", fmt.Errorf("RSI improvement %.2f is below required %.2f", rsiGate.ActualImprovementPercent, rsiGate.RequiredImprovementPercent))
	}
	pass("rsi_improvement_gate", "RSI improvement gate meets threshold")

	rsiNextTask, err := buildRSINextImprovementTask(goal, rsiCandidatePath, rsiGatePath, rsiCandidate, rsiGate)
	if err != nil {
		return fail("rsi_next_improvement_task", err)
	}
	rsiNextTaskPath := filepath.Join(outDir, "rsi-next-improvement-task.json")
	if err := writeJSONFile(rsiNextTaskPath, rsiNextTask); err != nil {
		return fail("rsi_next_improvement_task", err)
	}
	if err := event.addArtifact("rsi_next_improvement_task", rsiNextTaskPath, rsiNextTask.SchemaVersion, rsiNextTask.Status); err != nil {
		return fail("rsi_next_improvement_task_artifact", err)
	}
	pass("rsi_next_improvement_task", "RSI next improvement task was derived from candidate and gate evidence")

	demoStatus, err := buildDemoStatus(registryPath, taskPath, runPath)
	if err != nil {
		return fail("demo_status", err)
	}
	demoPath := filepath.Join(outDir, "demo-status.json")
	if err := writeJSONFile(demoPath, demoStatus); err != nil {
		return fail("demo_status", err)
	}
	if err := event.addArtifact("demo_status", demoPath, demoStatus.SchemaVersion, demoStatus.Status); err != nil {
		return fail("demo_status_artifact", err)
	}
	pass("demo_status", "operator demo status is ready")

	releaseManifest, err := buildReleaseManifest(true)
	if err != nil {
		return fail("release_manifest", err)
	}
	releasePath := filepath.Join(outDir, "release-manifest.json")
	if err := writeJSONFile(releasePath, releaseManifest); err != nil {
		return fail("release_manifest", err)
	}
	if err := event.addArtifact("release_manifest", releasePath, releaseManifest.SchemaVersion, releaseManifest.Status); err != nil {
		return fail("release_manifest_artifact", err)
	}
	pass("release_manifest", "release dry-run manifest is ready")

	competitiveAudit := buildCompetitiveAudit()
	competitivePath := filepath.Join(outDir, "competitive-readiness-audit.json")
	if err := writeJSONFile(competitivePath, competitiveAudit); err != nil {
		return fail("competitive_audit", err)
	}
	if err := event.addArtifact("competitive_readiness_audit", competitivePath, competitiveAudit.SchemaVersion, competitiveAudit.Status); err != nil {
		return fail("competitive_audit_artifact", err)
	}
	if competitiveAudit.Score != competitiveAudit.MaxScore {
		return fail("competitive_readiness_ready", fmt.Errorf("competitive readiness is %d/%d", competitiveAudit.Score, competitiveAudit.MaxScore))
	}
	pass("competitive_readiness_ready", "competitive readiness is 100/100")

	tracePath := filepath.Join(outDir, "pulse.trace.jsonl")
	artifactPaths := make([]string, 0, len(event.Artifacts))
	for _, artifact := range event.Artifacts {
		artifactPaths = append(artifactPaths, artifact.Path)
	}
	writeTraceSpan(tracePath, "pulse", "pulse.run", "passed", map[string]string{"registry": registryPath, "task": taskPath, "goal_run": goalPath}, artifactPaths, "")
	if err := event.addArtifact("pulse_trace", tracePath, "ao.foundry.trace.v0.1", "passed"); err != nil {
		return fail("pulse_trace_artifact", err)
	}
	spans, err := readTraceSpans(tracePath)
	if err != nil {
		return fail("trace_inspect", err)
	}
	traceSummary := summarizeTraceSpans(spans)
	traceInspectPath := filepath.Join(outDir, "trace-inspect.json")
	if err := writeJSONFile(traceInspectPath, traceSummary); err != nil {
		return fail("trace_inspect", err)
	}
	if err := event.addArtifact("trace_inspect", traceInspectPath, traceSummary.SchemaVersion, traceSummary.Status); err != nil {
		return fail("trace_inspect_artifact", err)
	}
	pass("trace_inspect", "pulse trace has a terminal passed span")

	event.Status = "ready"
	event.Score = event.MaxScore
	event.NextAction = "stop autonomous readiness loop; live execution requires operator intent"
	return event, nil
}

func buildRSICandidate(baselinePath, candidateEvalPath string) (RSICandidate, error) {
	if strings.TrimSpace(baselinePath) == "" || strings.TrimSpace(candidateEvalPath) == "" {
		return RSICandidate{}, errors.New("baseline and candidate eval result are required")
	}
	baseline, err := loadEvalResultForImprovement("baseline", baselinePath)
	if err != nil {
		return RSICandidate{}, err
	}
	candidate, err := loadEvalResultForImprovement("candidate", candidateEvalPath)
	if err != nil {
		return RSICandidate{}, err
	}
	baselineHash, err := fileSHA256(baselinePath)
	if err != nil {
		return RSICandidate{}, fmt.Errorf("hash baseline eval result: %w", err)
	}
	candidateHash, err := fileSHA256(candidateEvalPath)
	if err != nil {
		return RSICandidate{}, fmt.Errorf("hash candidate eval result: %w", err)
	}
	return RSICandidate{
		SchemaVersion:         "ao.foundry.rsi-candidate.v0.1",
		Status:                "ready",
		GeneratedBy:           "foundry pulse run",
		ImprovementHypothesis: "Local pulse generated the candidate eval result from the current Foundry run before measuring the RSI improvement gate.",
		BaselineEvalResult:    rsiCandidateEvalResult(baselinePath, baseline, baselineHash),
		CandidateEvalResult:   rsiCandidateEvalResult(candidateEvalPath, candidate, candidateHash),
		MutatesRepositories:   false,
		NextActions:           []string{},
	}, nil
}

func rsiCandidateEvalResult(path string, result EvalResult, hash string) RSICandidateEvalResult {
	return RSICandidateEvalResult{
		Path:          filepath.ToSlash(path),
		SchemaVersion: result.SchemaVersion,
		Status:        result.Status,
		Score:         result.Score,
		MaxScore:      result.MaxScore,
		SHA256:        hash,
	}
}

func buildRSINextImprovementTask(goal GoalRun, candidatePath, gatePath string, candidate RSICandidate, gate RSIImprovementGate) (RSINextImprovementTask, error) {
	if strings.TrimSpace(goal.GoalID) == "" || strings.TrimSpace(goal.NextTask) == "" {
		return RSINextImprovementTask{}, errors.New("goal_id and next_task are required")
	}
	if strings.TrimSpace(candidatePath) == "" || strings.TrimSpace(gatePath) == "" {
		return RSINextImprovementTask{}, errors.New("candidate and gate evidence paths are required")
	}
	if candidate.SchemaVersion != "ao.foundry.rsi-candidate.v0.1" || candidate.Status != "ready" {
		return RSINextImprovementTask{}, errors.New("RSI candidate must be ready")
	}
	if gate.SchemaVersion != "ao.foundry.rsi-improvement-gate.v0.1" || gate.Status != "passed" {
		return RSINextImprovementTask{}, errors.New("RSI improvement gate must be passed")
	}
	if candidate.MutatesRepositories || gate.MutatesRepositories {
		return RSINextImprovementTask{}, errors.New("RSI next task cannot be derived from mutating evidence")
	}
	if gate.RequiredImprovementPercent <= 0 || gate.ActualImprovementPercent < gate.RequiredImprovementPercent {
		return RSINextImprovementTask{}, errors.New("RSI improvement gate does not meet the required improvement")
	}
	recommendedTaskID := "rsi-next-" + shortSHA256(strings.Join([]string{goal.GoalID, goal.NextTask, candidatePath, gatePath}, ":"))
	return RSINextImprovementTask{
		SchemaVersion:              "ao.foundry.rsi-next-improvement-task.v0.1",
		Status:                     "ready",
		GeneratedBy:                "foundry pulse run",
		GoalID:                     goal.GoalID,
		RecommendedTaskID:          recommendedTaskID,
		RecommendedAction:          goal.NextTask,
		ImprovementRationale:       "The local pulse produced an RSI candidate and a passing improvement gate, so the next bounded task can be retained as governed evidence before delegation.",
		CandidateEvidencePath:      filepath.ToSlash(candidatePath),
		GateEvidencePath:           filepath.ToSlash(gatePath),
		RequiredImprovementPercent: gate.RequiredImprovementPercent,
		ActualImprovementPercent:   gate.ActualImprovementPercent,
		AutonomousClaim:            "derived_local_next_improvement",
		MutatesRepositories:        false,
		NextActions: []string{
			"retain rsi_next_improvement_task with RSI candidate and gate evidence",
			"delegate any repository-changing implementation through governed AO Forge execution",
		},
	}, nil
}

func newPulseFreshnessSummary() PulseFreshnessSummary {
	return PulseFreshnessSummary{
		SchemaVersion:        "ao.foundry.pulse-freshness-summary.v0.1",
		Status:               "ready",
		ForgeLivePacket:      "not_provided",
		ControlPlaneReadback: "not_provided",
		Explanation:          "no operator-provided production freshness evidence was bundled; local public pulse remains runnable without live credentials",
	}
}

func (summary *PulseFreshnessSummary) setForgeLiveAttempt(attempt ForgeLiveAttempt) {
	summary.ForgeLivePacket = freshnessStateFromForgeLiveAttempt(attempt)
	summary.refresh()
}

func (summary *PulseFreshnessSummary) setControlPlaneReadback(readback ControlPlaneReadback) {
	summary.ControlPlaneReadback = freshnessStateFromControlPlaneReadback(readback)
	summary.refresh()
}

func (summary *PulseFreshnessSummary) refresh() {
	if summary.ForgeLivePacket == "stale" || summary.ControlPlaneReadback == "stale" {
		summary.Status = "blocked"
		summary.Explanation = "operator-provided production freshness evidence is stale; rerun signed smoke before treating the pulse as production-ready"
		return
	}
	if summary.ForgeLivePacket == "blocked" || summary.ForgeLivePacket == "failed" || summary.ControlPlaneReadback == "blocked" {
		summary.Status = "blocked"
		summary.Explanation = "operator-provided production freshness evidence failed validation; resolve the evidence blocker and rerun pulse"
		return
	}
	if summary.ForgeLivePacket == "ready" && summary.ControlPlaneReadback == "ready" {
		summary.Status = "ready"
		summary.Explanation = "operator-provided AO Forge live packet and control-plane readback are fresh"
		return
	}
	if summary.ForgeLivePacket == "ready" || summary.ControlPlaneReadback == "ready" {
		summary.Status = "ready"
		summary.Explanation = "some operator-provided production freshness evidence is fresh; missing live evidence is recorded as not_provided"
		return
	}
	summary.Status = "ready"
	summary.Explanation = "no operator-provided production freshness evidence was bundled; local public pulse remains runnable without live credentials"
}

func freshnessStateFromForgeLiveAttempt(attempt ForgeLiveAttempt) string {
	if attempt.Source == "not-provided" {
		return "not_provided"
	}
	switch attempt.Status {
	case "passed":
		return "ready"
	case "stale":
		return "stale"
	case "failed":
		return "failed"
	default:
		return "blocked"
	}
}

func freshnessStateFromControlPlaneReadback(readback ControlPlaneReadback) string {
	if readback.Source == "not-provided" {
		return "not_provided"
	}
	switch readback.Status {
	case "ready":
		return "ready"
	case "stale":
		return "stale"
	default:
		return "blocked"
	}
}

func buildForgeLiveAttempt(packetPath string) (ForgeLiveAttempt, error) {
	attempt := ForgeLiveAttempt{
		SchemaVersion: "ao.foundry.forge-live-attempt.v0.1",
		Status:        "blocked",
		Source:        "not-provided",
		Explanation:   "AO Forge live execution was not attempted by the local public pulse; provide --forge-live-packet with an operator-produced packet to bundle live evidence.",
	}
	if strings.TrimSpace(packetPath) == "" {
		return attempt, nil
	}
	packet, err := loadForgeLivePacket(packetPath)
	if err != nil {
		return ForgeLiveAttempt{}, err
	}
	attempt.Status = packet.Status
	attempt.Source = publicArtifactSource(packetPath)
	attempt.PacketSchemaVersion = packet.SchemaVersion
	attempt.PacketStatus = packet.Status
	if info, err := os.Stat(packetPath); err == nil {
		if age := time.Since(info.ModTime()); age > liveEvidenceFreshnessWindow {
			attempt.Status = "stale"
			attempt.Explanation = "operator-provided AO Forge live packet is older than 24h; rerun signed smoke before using it as production-readiness evidence"
			return attempt, nil
		}
	} else {
		return ForgeLiveAttempt{}, err
	}
	attempt.Explanation = "operator-provided AO Forge live packet was validated and bundled"
	return attempt, nil
}

func buildControlPlaneReadback(receiptPath, packetPath string) (ControlPlaneReadback, error) {
	readback := ControlPlaneReadback{
		SchemaVersion: "ao.foundry.control-plane-readback.v0.1",
		Status:        "unavailable",
		Source:        "not-provided",
		Explanation:   "control-plane readback was not provided; produce a receipt through AO Forge or ao2-control-plane and rerun with --control-plane-receipt.",
	}
	if strings.TrimSpace(receiptPath) == "" {
		return readback, nil
	}
	var receipt map[string]any
	if err := readJSONFile(receiptPath, &receipt); err != nil {
		return ControlPlaneReadback{}, err
	}
	schema, ok := receipt["schema_version"].(string)
	if !ok || strings.TrimSpace(schema) == "" {
		return ControlPlaneReadback{}, errors.New("control-plane receipt requires schema_version")
	}
	resolvedReceiptPath, err := resolveReadablePath(receiptPath)
	if err != nil {
		return ControlPlaneReadback{}, err
	}
	if info, err := os.Stat(resolvedReceiptPath); err == nil {
		if age := time.Since(info.ModTime()); age > liveEvidenceFreshnessWindow {
			readback.Status = "stale"
			readback.Source = publicArtifactSource(receiptPath)
			readback.ReceiptSchemaVersion = schema
			readback.Explanation = "operator-provided control-plane readback receipt is older than 24h; rerun signed smoke before using it as production-readiness evidence"
			return readback, nil
		}
	} else {
		return ControlPlaneReadback{}, err
	}
	if expected, ok := expectedControlPlaneReceiptDigest(packetPath, receiptPath); ok {
		actual, err := fileSHA256(resolvedReceiptPath)
		if err != nil {
			return ControlPlaneReadback{}, err
		}
		if actual != expected {
			readback.Status = "blocked"
			readback.Source = publicArtifactSource(receiptPath)
			readback.ReceiptSchemaVersion = schema
			readback.Explanation = fmt.Sprintf("control-plane readback receipt digest mismatch: packet expected %s got %s", expected, actual)
			return readback, nil
		}
	}
	readback.Status = "ready"
	readback.Source = publicArtifactSource(receiptPath)
	readback.ReceiptSchemaVersion = schema
	readback.Explanation = "operator-provided control-plane readback receipt was validated and bundled"
	return readback, nil
}

func discoverControlPlaneReceiptPath(packetPath string) string {
	if strings.TrimSpace(packetPath) == "" {
		return ""
	}
	packet, err := loadForgeLivePacket(packetPath)
	if err != nil {
		return ""
	}
	for _, evidence := range packet.Evidence {
		if evidence.SchemaVersion == "ao2.cp-ingest-receipt.v1" || strings.EqualFold(evidence.Label, "control plane readback receipt") {
			return evidence.Path
		}
	}
	return ""
}

func loadForgeLivePacket(path string) (ForgePacket, error) {
	var packet ForgePacket
	if strings.TrimSpace(path) == "" {
		return packet, errors.New("missing live packet path")
	}
	if err := readJSONFile(path, &packet); err != nil {
		return packet, err
	}
	if packet.SchemaVersion != forgePacketSchema {
		return packet, fmt.Errorf("packet schema_version must be %s", forgePacketSchema)
	}
	if !allowedRunStatus(packet.Status) && packet.Status != "denied" {
		return packet, fmt.Errorf("packet has invalid status %q", packet.Status)
	}
	return packet, nil
}

func expectedControlPlaneReceiptDigest(packetPath, receiptPath string) (string, bool) {
	if strings.TrimSpace(packetPath) == "" || strings.TrimSpace(receiptPath) == "" {
		return "", false
	}
	packet, err := loadForgeLivePacket(packetPath)
	if err != nil {
		return "", false
	}
	receiptClean := filepath.ToSlash(filepath.Clean(receiptPath))
	for _, evidence := range packet.Evidence {
		if evidence.SchemaVersion != "ao2.cp-ingest-receipt.v1" && !strings.EqualFold(evidence.Label, "control plane readback receipt") {
			continue
		}
		evidenceClean := filepath.ToSlash(filepath.Clean(evidence.Path))
		if evidenceClean == receiptClean || filepath.Base(evidenceClean) == filepath.Base(receiptClean) {
			return evidence.SHA256, true
		}
	}
	return "", false
}

func resolveReadablePath(path string) (string, error) {
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if filepath.IsAbs(path) {
		return "", err
	}
	root, err := repoRoot()
	if err != nil {
		return "", err
	}
	resolved := filepath.Join(root, filepath.Clean(filepath.FromSlash(path)))
	if _, err := os.Stat(resolved); err != nil {
		return "", err
	}
	return resolved, nil
}

func publicArtifactSource(path string) string {
	if filepath.IsAbs(path) {
		return "operator-provided:" + filepath.Base(path)
	}
	return filepath.ToSlash(path)
}

func buildPolicyGateSummary(packet ForgePacket) PolicyGateSummary {
	gate := PolicyGateSummary{
		SchemaVersion: "ao.foundry.policy-gate-summary.v0.1",
		Status:        "ready",
		Decisions:     append([]RunDecision(nil), packet.PolicyDecisions...),
		Explanation:   "all policy decisions allow delegated execution",
	}
	if len(gate.Decisions) == 0 {
		gate.Status = "blocked"
		gate.Explanation = "policy decisions are missing"
		return gate
	}
	for _, decision := range gate.Decisions {
		if decision.Decision == "deny" || decision.Decision == "blocked" {
			gate.Status = "blocked"
			gate.Explanation = "policy decision blocks delegated execution"
			return gate
		}
	}
	return gate
}

func (event *PulseEvent) addArtifact(name, path, schemaVersion, status string) error {
	sum, err := fileSHA256(path)
	if err != nil {
		return err
	}
	event.Artifacts = append(event.Artifacts, PulseArtifact{
		Name:          name,
		Path:          filepath.ToSlash(path),
		SHA256:        sum,
		SchemaVersion: schemaVersion,
		Status:        status,
	})
	return nil
}

func summarizeTraceSpans(spans []TraceSpan) TraceInspectSummary {
	summary := TraceInspectSummary{
		SchemaVersion: "ao.foundry.trace-inspect.v0.1",
		Status:        "ready",
		Spans:         len(spans),
	}
	for _, span := range spans {
		if span.Status == "failed" {
			summary.FailedSpans++
		}
		summary.EvidenceRefs += len(span.EvidenceRefs)
	}
	if summary.FailedSpans > 0 || summary.Spans == 0 {
		summary.Status = "blocked"
	}
	return summary
}

func writeDemoScript(path string) error {
	script := `# AO Foundry Five-Minute Demo

## Positioning

AO Foundry is the engineering operations factory above AO Forge. It coordinates registries, tasks, readiness, runs, evals, traces, and scheduler gates. It does not replace AO Forge; individual governed implementation runs are delegated to AO Forge.

## Flow

1. Show the local AO stack registry with ` + "`foundry status`" + `.
2. Validate the bootstrap task and GoalRun readiness.
3. Emit an AO Forge brief with ` + "`foundry next --out`" + `.
4. Inspect the AO Forge packet fixture as governed execution evidence.
5. Ingest the packet into a Foundry run record.
6. Score the run with ` + "`foundry eval run`" + `.
7. Show the next safe action from ` + "`foundry demo status`" + `.

## Guardrails

- No credentials are required.
- No network access is required.
- No release, tag, push, upload, or sibling-repository mutation is performed.
- Internal coordination material is not part of the public demo.
`
	if err := os.MkdirAll(parentDir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(script), 0o644)
}

func writeSignedSmokeScript(path string) error {
	script := `#!/usr/bin/env bash
set -euo pipefail

: "${AO2_CP_API_TOKEN:?set AO2_CP_API_TOKEN}"
if [ "${#AO2_CP_API_TOKEN}" -lt 32 ]; then
  printf 'AO2_CP_API_TOKEN must be at least 32 characters\n' >&2
  exit 2
fi

mkdir -p tmp/live-tools tmp/control-plane docs/evidence/pulse/local-live-smoke

AO2_CP_API_TOKEN="$AO2_CP_API_TOKEN" ../ao2-control-plane/target/debug/ao2-cp-server --bind 127.0.0.1:18746 \
  --data-dir tmp/control-plane &
AO2_CP_PID="$!"
trap 'kill "$AO2_CP_PID" 2>/dev/null || true' EXIT

sleep 1

(cd ../ao-forge && go build -o ../ao-foundry/tmp/live-tools/forge ./cmd/forge)
(cd ../ao-covenant && go build -o ../ao-foundry/tmp/live-tools/covenant ./cmd/covenant)

go run ./cmd/foundry pulse run --out tmp/pulse

tmp/live-tools/forge plan \
  --brief tmp/pulse/forge-brief.json \
  --out docs/evidence/pulse/local-live-smoke/factory-plan.json

tmp/live-tools/forge gate \
  --plan docs/evidence/pulse/local-live-smoke/factory-plan.json \
  --covenant tmp/live-tools/covenant \
  --out docs/evidence/pulse/local-live-smoke/gate-result.json

AO2_CP_API_TOKEN="$AO2_CP_API_TOKEN" tmp/live-tools/forge run \
  --plan docs/evidence/pulse/local-live-smoke/factory-plan.json \
  --gate-result docs/evidence/pulse/local-live-smoke/gate-result.json \
  --out docs/evidence/pulse/local-live-smoke/factory-packet.json \
  --control-plane http://127.0.0.1:18746 \
  --live --non-interactive --no-dashboard

go run ./cmd/foundry pulse run \
  --out tmp/pulse-live \
  --forge-live-packet docs/evidence/pulse/local-live-smoke/factory-packet.json

go run ./cmd/foundry trace inspect --trace tmp/pulse-live/pulse.trace.jsonl

cat > tmp/pulse-live/signed-smoke-result.json <<'JSON'
{
  "schema_version": "ao.foundry.signed-smoke-result.v0.1",
  "status": "ready",
  "pulse_event": "tmp/pulse-live/pulse-event.json",
  "forge_live_packet": "docs/evidence/pulse/local-live-smoke/factory-packet.json",
  "control_plane_readback": "ready"
}
JSON

go run ./cmd/foundry pulse run \
  --out tmp/pulse-live \
  --forge-live-packet docs/evidence/pulse/local-live-smoke/factory-packet.json \
  --signed-smoke-result tmp/pulse-live/signed-smoke-result.json

go run ./cmd/foundry pulse summarize-signed-smoke --pulse tmp/pulse-live/pulse-event.json --out tmp/pulse-live/signed-smoke-summary.json

go run ./cmd/foundry release promotion validate --candidate examples/readiness/active-spine-release-candidate.ledger.json --signed-smoke-summary tmp/pulse-live/signed-smoke-summary.json --out tmp/release-promotion.live.json

printf 'signed_smoke_result=tmp/pulse-live/signed-smoke-result.json\n'
printf 'signed_smoke_summary=tmp/pulse-live/signed-smoke-summary.json\n'
printf 'release_promotion=tmp/release-promotion.live.json\n'
`
	if err := os.MkdirAll(parentDir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		return err
	}
	return nil
}

func buildSignedSmokePreflight(workspace string) SignedSmokePreflight {
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		workspace = ".."
	}
	checks := []SignedSmokeCheck{
		signedSmokeDirCheck("ao_forge_repo", filepath.Join(workspace, "ao-forge")),
		signedSmokeDirCheck("ao_covenant_repo", filepath.Join(workspace, "ao-covenant")),
		signedSmokeExecutableCheck("ao2_control_plane_server", filepath.Join(workspace, "ao2-control-plane", "target", "debug", executableName("ao2-cp-server"))),
	}
	preflight := SignedSmokePreflight{
		SchemaVersion: "ao.foundry.signed-smoke-preflight.v0.1",
		Status:        "ready",
		Workspace:     filepath.ToSlash(filepath.Clean(workspace)),
		Checks:        checks,
		NextActions:   []string{},
	}
	for _, check := range checks {
		if check.Status != "ready" {
			preflight.Status = "blocked"
			preflight.NextActions = append(preflight.NextActions, check.Reason)
		}
	}
	return preflight
}

func cleanupSignedSmokeScratch() (int, error) {
	paths := []string{
		"tmp/live-tools",
		"tmp/control-plane",
		"tmp/signed-smoke.sh",
		"tmp/signed-smoke-preflight.json",
		"tmp/pulse-live",
		"tmp/pulse-live-bundled",
	}
	removed := 0
	for _, path := range paths {
		clean := filepath.Clean(filepath.FromSlash(path))
		if filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return removed, fmt.Errorf("unsafe cleanup path %q", path)
		}
		abs := repoPath(clean)
		if _, err := os.Stat(abs); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return removed, err
		}
		if err := os.RemoveAll(abs); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

func signedSmokeDirCheck(name, path string) SignedSmokeCheck {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return SignedSmokeCheck{
			Name:   name,
			Status: "blocked",
			Path:   filepath.ToSlash(path),
			Reason: fmt.Sprintf("%s directory is required", filepath.ToSlash(path)),
		}
	}
	return SignedSmokeCheck{Name: name, Status: "ready", Path: filepath.ToSlash(path), Reason: "directory is available"}
}

func signedSmokeExecutableCheck(name, path string) SignedSmokeCheck {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return SignedSmokeCheck{
			Name:   name,
			Status: "blocked",
			Path:   filepath.ToSlash(path),
			Reason: fmt.Sprintf("%s executable is required", filepath.ToSlash(path)),
		}
	}
	if os.PathSeparator == '\\' && strings.EqualFold(filepath.Ext(path), ".exe") {
		return SignedSmokeCheck{Name: name, Status: "ready", Path: filepath.ToSlash(path), Reason: "executable is available"}
	}
	if info.Mode()&0o111 == 0 {
		return SignedSmokeCheck{
			Name:   name,
			Status: "blocked",
			Path:   filepath.ToSlash(path),
			Reason: fmt.Sprintf("%s must be executable", filepath.ToSlash(path)),
		}
	}
	return SignedSmokeCheck{Name: name, Status: "ready", Path: filepath.ToSlash(path), Reason: "executable is available"}
}

func executableName(name string) string {
	if os.PathSeparator == '\\' {
		return name + ".exe"
	}
	return name
}

func buildSignedSmokeIngest(resultPath string) (SignedSmokeIngest, error) {
	var result SignedSmokeResult
	if err := readJSONFile(resultPath, &result); err != nil {
		return SignedSmokeIngest{}, err
	}
	if result.SchemaVersion != "ao.foundry.signed-smoke-result.v0.1" {
		return SignedSmokeIngest{}, errors.New("invalid signed smoke result schema_version")
	}
	if result.Status != "ready" {
		return SignedSmokeIngest{}, fmt.Errorf("signed smoke result status must be ready, got %q", result.Status)
	}
	if result.ControlPlaneReadback != "ready" {
		return SignedSmokeIngest{}, fmt.Errorf("signed smoke control_plane_readback must be ready, got %q", result.ControlPlaneReadback)
	}
	for label, path := range map[string]string{
		"pulse_event":       result.PulseEvent,
		"forge_live_packet": result.ForgeLivePacket,
	} {
		if err := validateSignedSmokeResultPath(path); err != nil {
			return SignedSmokeIngest{}, fmt.Errorf("%s: %w", label, err)
		}
	}
	sum, err := fileSHA256(resultPath)
	if err != nil {
		return SignedSmokeIngest{}, err
	}
	return SignedSmokeIngest{
		SchemaVersion:        "ao.foundry.signed-smoke-ingest.v0.1",
		Status:               "ready",
		Result:               publicResultPath(resultPath),
		ResultSHA256:         sum,
		PulseEvent:           result.PulseEvent,
		ForgeLivePacket:      result.ForgeLivePacket,
		ControlPlaneReadback: result.ControlPlaneReadback,
		Explanation:          "signed AO Forge and control-plane smoke result was validated for Foundry ingestion",
	}, nil
}

func buildSignedSmokeSummary(pulsePath string) (SignedSmokeSummary, error) {
	var event PulseEvent
	if err := readJSONFile(pulsePath, &event); err != nil {
		return SignedSmokeSummary{}, err
	}
	if event.SchemaVersion != pulseEventSchema {
		return SignedSmokeSummary{}, fmt.Errorf("unexpected pulse schema %q", event.SchemaVersion)
	}
	required := []struct {
		name       string
		readyState string
	}{
		{name: "forge_live_attempt", readyState: "passed"},
		{name: "control_plane_readback", readyState: "ready"},
		{name: "signed_smoke_ingest", readyState: "ready"},
	}
	artifacts := map[string]PulseArtifact{}
	for _, artifact := range event.Artifacts {
		artifacts[artifact.Name] = artifact
	}
	summary := SignedSmokeSummary{
		SchemaVersion: "ao.foundry.signed-smoke-summary.v0.1",
		Status:        "ready",
		PulseID:       event.PulseID,
		PulseStatus:   event.Status,
		ReleaseSafe:   true,
		Evidence:      []SignedSmokeSummaryEvidence{},
		Explanation:   "Public-safe signed-smoke summary omits source paths, digests, tokens, server logs, and runtime scratch details.",
	}
	for _, requirement := range required {
		artifact, ok := artifacts[requirement.name]
		if !ok {
			summary.Status = "blocked"
			summary.Evidence = append(summary.Evidence, SignedSmokeSummaryEvidence{
				Name:          requirement.name,
				Status:        "missing",
				SchemaVersion: "missing",
			})
			continue
		}
		if artifact.Status != requirement.readyState {
			summary.Status = "blocked"
		}
		summary.Evidence = append(summary.Evidence, SignedSmokeSummaryEvidence{
			Name:          artifact.Name,
			Status:        artifact.Status,
			SchemaVersion: artifact.SchemaVersion,
		})
	}
	if event.Status != "ready" {
		summary.Status = "blocked"
	}
	return summary, nil
}

func renderReleaseCandidateNotes(candidate ReleaseCandidateLedger, promotion ReleasePromotionLedger) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Active Spine Release Candidate: %s\n\n", candidate.CandidateID)
	fmt.Fprintf(&b, "Status: %s\n\n", candidate.Status)
	fmt.Fprintf(&b, "Release safe: %t\n", promotion.ReleaseSafe)
	fmt.Fprintf(&b, "Signed smoke pulse: %s\n", promotion.SignedSmokePulseID)
	fmt.Fprintf(&b, "Signed smoke summary: %s\n", promotion.SignedSmokeSummaryStatus)
	fmt.Fprintf(&b, "Pulse status: %s\n\n", promotion.PulseStatus)

	b.WriteString("## Active Spine\n\n")
	b.WriteString("| Repository | Role | Status | Evidence |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, repo := range candidate.ActiveSpine {
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", escapeMarkdownCell(repo.Name), escapeMarkdownCell(repo.Role), escapeMarkdownCell(repo.Status), escapeMarkdownCell(formatEvidenceItems(repo.Evidence)))
	}

	b.WriteString("\n## Gates\n\n")
	b.WriteString("| Gate | Status | Required before promotion | Evidence |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, gate := range candidate.Gates {
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", escapeMarkdownCell(gate.Name), escapeMarkdownCell(gate.Status), boolStatus(gate.RequiredBeforePromotion), escapeMarkdownCell(formatEvidenceItems(gate.Evidence)))
	}

	b.WriteString("\n## Promotion Evidence\n\n")
	b.WriteString("| Evidence | Status | Schema |\n")
	b.WriteString("| --- | --- | --- |\n")
	for _, item := range promotion.Evidence {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", escapeMarkdownCell(item.Name), escapeMarkdownCell(item.Status), escapeMarkdownCell(item.SchemaVersion))
	}

	b.WriteString("\n## Tag plan\n\n")
	fmt.Fprintf(&b, "- Candidate tag: `%s`\n", candidate.CandidateID)
	b.WriteString("- Promote only after the signed-smoke summary is fresh for the promotion window.\n")
	for _, action := range promotion.NextActions {
		fmt.Fprintf(&b, "- %s\n", action)
	}
	return b.String()
}

func validateSignedSmokeResultPath(path string) error {
	cleaned := strings.ReplaceAll(path, "\\", "/")
	if cleaned == "" ||
		strings.HasPrefix(cleaned, "/") ||
		strings.HasPrefix(cleaned, "~/") ||
		strings.HasPrefix(cleaned, "../") ||
		strings.Contains(cleaned, "/../") ||
		isWindowsAbsolutePath(cleaned) {
		return fmt.Errorf("unsafe signed smoke result path %q", path)
	}
	return nil
}

func publicResultPath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Base(path)
	}
	return filepath.ToSlash(filepath.Clean(path))
}

func portableEvidencePath(path string) string {
	return filepath.ToSlash(filepath.Clean(path))
}

func buildAO2LoopDecision(action, reason, nextTaskID string) (AO2LoopDecision, error) {
	action = strings.TrimSpace(action)
	reason = strings.TrimSpace(reason)
	nextTaskID = strings.TrimSpace(nextTaskID)
	if action != "stop" && action != "continue" {
		return AO2LoopDecision{}, fmt.Errorf("action must be stop or continue, got %q", action)
	}
	if reason == "" {
		return AO2LoopDecision{}, errors.New("reason is required")
	}
	if nextTaskID == "" {
		return AO2LoopDecision{}, errors.New("next-task-id is required")
	}
	return buildAO2LoopDecisionWithFreshness(action, reason, nextTaskID, newPulseFreshnessSummary()), nil
}

func buildAO2LoopDecisionWithFreshness(action, reason, nextTaskID string, freshness PulseFreshnessSummary) AO2LoopDecision {
	if strings.TrimSpace(freshness.SchemaVersion) == "" {
		freshness = newPulseFreshnessSummary()
	}
	return AO2LoopDecision{
		SchemaVersion: "ao2.pulse-event-loop-decision.v1",
		EventLoop: AO2LoopDecisionBody{
			Action:     action,
			Reason:     reason,
			NextTaskID: nextTaskID,
			Freshness:  freshness,
		},
	}
}

func buildDerivedAO2LoopDecision(pulsePath, auditPath string) (AO2LoopDecision, error) {
	var event PulseEvent
	if err := readJSONFile(pulsePath, &event); err != nil {
		return AO2LoopDecision{}, fmt.Errorf("read pulse event: %w", err)
	}
	if event.SchemaVersion != pulseEventSchema {
		return AO2LoopDecision{}, fmt.Errorf("unexpected pulse schema %q", event.SchemaVersion)
	}
	var audit *CompetitiveReadinessAudit
	if strings.TrimSpace(auditPath) != "" {
		var loaded CompetitiveReadinessAudit
		if err := readJSONFile(auditPath, &loaded); err != nil {
			return AO2LoopDecision{}, fmt.Errorf("read audit: %w", err)
		}
		audit = &loaded
	}
	nextTaskID := deriveNextTaskID(event, audit)
	reason := fmt.Sprintf("Foundry derived next task %q from pulse status %q.", nextTaskID, event.Status)
	return buildAO2LoopDecisionWithFreshness("stop", reason, nextTaskID, event.Freshness), nil
}

func deriveNextTaskID(event PulseEvent, audit *CompetitiveReadinessAudit) string {
	if strings.TrimSpace(event.Freshness.Status) == "blocked" {
		switch {
		case event.Freshness.ForgeLivePacket == "stale":
			return "refresh-forge-live-packet"
		case event.Freshness.ControlPlaneReadback == "stale":
			return "refresh-control-plane-readback"
		case event.Freshness.ForgeLivePacket == "blocked" || event.Freshness.ForgeLivePacket == "failed" || event.Freshness.ControlPlaneReadback == "blocked":
			return "resolve-production-evidence-freshness"
		}
	}
	if strings.TrimSpace(event.Status) != "ready" {
		for _, check := range event.Checks {
			if strings.TrimSpace(check.Status) != "pass" {
				if id := slugTaskID(check.Name); id != "" {
					return "resolve-" + id
				}
			}
		}
		return "resolve-pulse-blockers"
	}
	if audit != nil {
		for _, action := range audit.NextActions {
			if id := slugTaskID(action); id != "" {
				return id
			}
		}
		for _, category := range audit.Categories {
			for _, action := range category.NextActions {
				if id := slugTaskID(action); id != "" {
					return id
				}
			}
		}
		if strings.TrimSpace(audit.Status) != "ready" || audit.Score < audit.MaxScore {
			return "resolve-competitive-readiness"
		}
	}
	if event.Score >= event.MaxScore {
		return "readiness-exit-gate-satisfied"
	}
	if id := slugTaskID(event.NextAction); id != "" {
		return id
	}
	return "resolve-pulse-readiness-gap"
}

func slugTaskID(input string) string {
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(input)) {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		default:
			if builder.Len() > 0 && !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	slug := strings.Trim(builder.String(), "-")
	if len(slug) > 80 {
		slug = strings.TrimRight(slug[:80], "-")
	}
	return slug
}

func buildReleaseManifest(dryRun bool) (ReleaseManifest, error) {
	root, err := repoRoot()
	if err != nil {
		return ReleaseManifest{}, err
	}
	includeDirs := map[string]bool{"cmd": true, "docs": true, "examples": true, "internal": true}
	includeFiles := map[string]bool{
		"README.md":       true,
		"go.mod":          true,
		"LICENSE":         true,
		"NOTICE":          true,
		"SECURITY.md":     true,
		"CONTRIBUTING.md": true,
	}
	manifest := ReleaseManifest{
		SchemaVersion: "ao.foundry.release-manifest.v0.1",
		Status:        "ready",
		Files:         []ReleaseFileEntry{},
		Checks:        []string{},
	}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if entry.IsDir() {
			if rel == "tmp" || rel == ".git" || rel == "docs/evidence" || strings.HasPrefix(rel, ".github/workflows") {
				if rel == ".github/workflows" {
					return nil
				}
				return filepath.SkipDir
			}
			top := strings.Split(rel, "/")[0]
			if includeDirs[top] || rel == ".github" {
				return nil
			}
			return filepath.SkipDir
		}
		top := strings.Split(rel, "/")[0]
		if !includeDirs[top] && !includeFiles[rel] && !strings.HasPrefix(rel, ".github/workflows/") {
			return nil
		}
		sum, err := fileSHA256(rel)
		if err != nil {
			return err
		}
		manifest.Files = append(manifest.Files, ReleaseFileEntry{Path: rel, SHA256: sum})
		return nil
	})
	if err != nil {
		return ReleaseManifest{}, err
	}
	sort.Slice(manifest.Files, func(i, j int) bool { return manifest.Files[i].Path < manifest.Files[j].Path })
	if dryRun {
		if _, err := loadRegistry("examples/registry/local-ao-stack.foundry-registry.json"); err != nil {
			return ReleaseManifest{}, err
		}
		if _, err := loadTask("examples/tasks/ao-foundry-bootstrap.foundry-task.json"); err != nil {
			return ReleaseManifest{}, err
		}
		if _, err := loadFoundryRun("examples/runs/ao-foundry-bootstrap.foundry-run.json"); err != nil {
			return ReleaseManifest{}, err
		}
		if _, err := validateContractFixtures(); err != nil {
			return ReleaseManifest{}, err
		}
		manifest.Checks = append(manifest.Checks, "registry fixture valid", "task fixture valid", "run fixture valid", "contract fixtures valid")
	}
	return manifest, nil
}

func validateReleaseManifestFile(path string) error {
	if path == "" {
		return errors.New("missing --manifest")
	}
	var manifest ReleaseManifest
	if err := readJSONFile(path, &manifest); err != nil {
		return err
	}
	if manifest.SchemaVersion != "ao.foundry.release-manifest.v0.1" {
		return errors.New("invalid release manifest schema_version")
	}
	if manifest.Status != "ready" {
		return errors.New("release manifest status must be ready")
	}
	if len(manifest.Files) == 0 {
		return errors.New("release manifest must include files")
	}
	for _, file := range manifest.Files {
		if file.Path == "" {
			return errors.New("release manifest file path is required")
		}
		if err := validateEvidencePath(file.Path); err != nil {
			return err
		}
		if err := validateSHA256(file.SHA256, "release file "+file.Path); err != nil {
			return err
		}
	}
	return nil
}

func buildCompetitiveAudit() CompetitiveReadinessAudit {
	categories := []CompetitiveAuditCategory{
		competitiveCategory("clean_clone_public_readiness", 15, []CompetitiveAuditCheck{
			checkFileExists("clean_clone_commands_documented", "docs/operations/CLEAN-CLONE-SMOKE.md"),
			checkFileExists("clean_clone_registry_fixture", "examples/registry/clean-clone.foundry-registry.json"),
			checkFileExists("clean_clone_task_fixture", "examples/tasks/clean-clone-smoke.foundry-task.json"),
			checkNoPublicSiblingDependency("public_release_checklist_has_no_sibling_dependency"),
		}),
		competitiveCategory("contract_depth", 15, contractDepthChecks()),
		competitiveCategory("real_delegated_forge_loop", 15, []CompetitiveAuditCheck{
			checkFileExists("forge_brief_fixture", "examples/briefs/ao-foundry-bootstrap.forge-brief.json"),
			checkFileExists("forge_packet_fixture", "examples/packets/ao-foundry-bootstrap.factory-packet.json"),
			checkFileExists("foundry_run_record", "examples/runs/ao-foundry-bootstrap.foundry-run.json"),
			checkFileExists("bootstrap_eval_result", "examples/evals/bootstrap.eval-result.json"),
			checkFileExists("demo_status_fixture", "examples/demo/ao-foundry-demo-status.json"),
			checkFileExists("pulse_golden_loop_sdd", "docs/sdd/AO-FOUNDRY-PULSE-GOLDEN-LOOP-SDD.md"),
			checkFileExists("pulse_production_adapters_sdd", "docs/sdd/AO-FOUNDRY-PULSE-PRODUCTION-ADAPTERS-SDD.md"),
			checkFileContains("pulse_golden_loop_test", "internal/cli/cli_test.go", "TestPulseRunWritesGoldenLoopBundle"),
			checkFileContains("pulse_blocked_event_test", "internal/cli/cli_test.go", "TestPulseRunWritesFailedEventForBlockedReadiness"),
			checkFileContains("forge_live_adapter_test", "internal/cli/cli_test.go", "TestPulseRunRecordsProvidedForgeLivePacket"),
			checkFileContains("control_plane_readback_test", "internal/cli/cli_test.go", "TestPulseRunRecordsProvidedControlPlaneReadback"),
			checkFileContains("ao_surface_test", "internal/cli/cli_test.go", "TestAOSurfaceStatusRunAndAudit"),
		}),
		competitiveCategory("scheduler_safety", 10, []CompetitiveAuditCheck{
			checkFileContains("lease_overlap_test", "internal/cli/cli_test.go", "TestLoopLeaseAcquireRefusesActiveLease"),
			checkFileContains("stale_lease_test", "internal/cli/cli_test.go", "TestLoopLeaseAcquireReportsStaleLease"),
			checkFileContains("terminal_goal_blocks_test", "internal/cli/cli_test.go", "TestLoopPreflightBlocksTerminalGoal"),
			checkFileContains("budget_blocks_test", "internal/cli/cli_test.go", "TestLoopPreflightBlocksBudgetExhaustion"),
			checkFileContains("loop_emits_forge_brief", "internal/cli/cli_test.go", "TestLoopNextWritesForgeBrief"),
		}),
		competitiveCategory("hitl_and_policy_safety", 10, []CompetitiveAuditCheck{
			checkFileContains("approval_required_for_network", "internal/cli/cli_test.go", "TestNextBlocksNonLocalTaskWithoutApproval"),
			checkFileContains("expired_approval_fails", "internal/cli/cli_test.go", "TestApprovalValidateRejectsExpiredDecision"),
			checkFileContains("digest_mismatch_approval_fails", "internal/cli/cli_test.go", "TestApprovalValidateRejectsDigestMismatch"),
			checkFileContains("broadened_approval_fails", "internal/cli/cli_test.go", "TestApprovalValidateRejectsBroadenedDecision"),
			checkFileContains("approval_evidence_in_brief", "internal/cli/cli_test.go", "TestNextReferencesApprovalInForgeBrief"),
		}),
		competitiveCategory("observability_and_eval_coverage", 10, []CompetitiveAuditCheck{
			checkFileContains("trace_next", "internal/cli/cli_test.go", "TestNextTrace"),
			checkFileContains("trace_run_ingest", "internal/cli/cli_test.go", "TestRunIngestTrace"),
			checkFileContains("trace_loop", "internal/cli/cli_test.go", "TestLoopNextTrace"),
			checkFileContains("trace_approval", "internal/cli/cli_test.go", "TestApprovalValidateTrace"),
			checkFileContains("trace_eval", "internal/cli/cli_test.go", "TestEvalRunTrace"),
			checkFileContains("eval_threshold_failure", "internal/cli/cli_test.go", "TestEvalRunFailsBelowThreshold"),
		}),
		competitiveCategory("release_hardening", 10, []CompetitiveAuditCheck{
			checkFileExists("license", "LICENSE"),
			checkFileExists("notice", "NOTICE"),
			checkFileExists("security_policy", "SECURITY.md"),
			checkFileExists("contributing", "CONTRIBUTING.md"),
			checkFileExists("release_manifest_schema", "docs/contracts/foundry-release-manifest-v0.1.schema.json"),
			checkFileContains("ci_validates_release_manifest", ".github/workflows/ci.yml", "release validate-manifest"),
		}),
		competitiveCategory("competitive_public_demo", 10, []CompetitiveAuditCheck{
			checkFileExists("five_minute_demo", "docs/operations/FIVE-MINUTE-DEMO.md"),
			checkFileExists("demo_status_fixture", "examples/demo/ao-foundry-demo-status.json"),
			checkFileExists("capability_matrix", "examples/capabilities/foundry-capability-matrix.json"),
			checkFileContains("demo_no_unsupported_claims", "examples/capabilities/foundry-capability-matrix.json", "out-of-scope"),
		}),
		competitiveCategory("public_safety", 5, []CompetitiveAuditCheck{
			checkPublicSafety("public_files_scan_clean"),
			checkFileContains("security_contact_finalized", "SECURITY.md", "GitHub Security Advisories"),
		}),
	}
	audit := CompetitiveReadinessAudit{
		SchemaVersion: "ao.foundry.competitive-readiness-audit.v0.1",
		Status:        "ready",
		MaxScore:      100,
		Categories:    categories,
		NextActions:   []string{},
	}
	for _, category := range categories {
		audit.Score += category.Score
		if category.Status != "ready" {
			audit.Status = "blocked"
			audit.NextActions = append(audit.NextActions, category.NextActions...)
		}
	}
	if audit.Score != audit.MaxScore || len(audit.NextActions) > 0 {
		audit.Status = "blocked"
	}
	return audit
}

func competitiveCategory(id string, maxScore int, checks []CompetitiveAuditCheck) CompetitiveAuditCategory {
	category := CompetitiveAuditCategory{
		ID:          id,
		Status:      "ready",
		MaxScore:    maxScore,
		Checks:      checks,
		NextActions: []string{},
	}
	for _, check := range checks {
		if check.Status != "pass" {
			category.Status = "blocked"
			category.NextActions = append(category.NextActions, check.Reason)
		}
	}
	if category.Status == "ready" {
		category.Score = maxScore
	}
	return category
}

func contractDepthChecks() []CompetitiveAuditCheck {
	checks := []CompetitiveAuditCheck{
		checkFileExists("contract_fixture_index", "docs/contracts/CONTRACT-FIXTURES.md"),
	}
	for _, schema := range publicSchemaNames() {
		valid := "examples/contract-fixtures/valid/" + schema + ".json"
		invalid := "examples/contract-fixtures/invalid/" + schema + ".json"
		checks = append(checks, checkFileExists("valid_fixture_"+schema, valid))
		checks = append(checks, checkFileExists("invalid_fixture_"+schema, invalid))
	}
	return checks
}

func validateContractFixtures() (ContractFixtureValidationResult, error) {
	result := ContractFixtureValidationResult{}
	for _, schemaName := range publicSchemaNames() {
		schemaPath := "docs/contracts/" + schemaName + ".schema.json"
		validPath := "examples/contract-fixtures/valid/" + schemaName + ".json"
		invalidPath := "examples/contract-fixtures/invalid/" + schemaName + ".json"
		schema, err := readArbitraryJSON(schemaPath)
		if err != nil {
			return result, fmt.Errorf("read schema %s: %w", schemaPath, err)
		}
		root, ok := schema.(map[string]any)
		if !ok {
			return result, fmt.Errorf("schema %s is not an object", schemaPath)
		}
		validDocument, err := readArbitraryJSON(validPath)
		if err != nil {
			return result, fmt.Errorf("read valid fixture %s: %w", validPath, err)
		}
		if err := validateJSONSchemaValue(root, root, validDocument, "$"); err != nil {
			return result, fmt.Errorf("valid fixture %s failed %s: %w", validPath, schemaPath, err)
		}
		result.ValidFixtures++

		invalidDocument, err := readArbitraryJSON(invalidPath)
		if err != nil {
			return result, fmt.Errorf("read invalid fixture %s: %w", invalidPath, err)
		}
		if err := validateJSONSchemaValue(root, root, invalidDocument, "$"); err == nil {
			return result, fmt.Errorf("invalid fixture %s unexpectedly passed %s", invalidPath, schemaPath)
		}
		result.InvalidFixtures++
	}
	return result, nil
}

func readArbitraryJSON(path string) (any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		data, err = readRepoRelativeFile(path)
		if err != nil {
			return nil, err
		}
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return value, nil
}

func validateJSONSchemaValue(root, schema map[string]any, value any, path string) error {
	if ref, ok := schema["$ref"].(string); ok {
		resolved, err := resolveLocalSchemaRef(root, ref)
		if err != nil {
			return err
		}
		return validateJSONSchemaValue(root, resolved, value, path)
	}
	if clauses, ok := schema["allOf"].([]any); ok {
		for i, clause := range clauses {
			clauseSchema, ok := clause.(map[string]any)
			if !ok {
				return fmt.Errorf("%s allOf[%d] is not an object", path, i)
			}
			if err := validateConditionalSchema(root, clauseSchema, value, path); err != nil {
				return err
			}
		}
	}
	if expected, ok := schema["const"]; ok && !jsonValuesEqual(expected, value) {
		return fmt.Errorf("%s must equal %v", path, expected)
	}
	if enum, ok := schema["enum"].([]any); ok && !jsonValueInEnum(value, enum) {
		return fmt.Errorf("%s must match one of %v", path, enum)
	}
	if typeName, ok := schema["type"].(string); ok {
		if err := validateJSONSchemaType(typeName, value, path); err != nil {
			return err
		}
	}
	if properties, ok := schema["properties"].(map[string]any); ok {
		object, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("%s must be an object", path)
		}
		if required, ok := schema["required"].([]any); ok {
			for _, name := range required {
				property, ok := name.(string)
				if !ok {
					return fmt.Errorf("%s required property name is not a string", path)
				}
				if _, exists := object[property]; !exists {
					return fmt.Errorf("%s missing required property %q", path, property)
				}
			}
		}
		if err := validateAdditionalProperties(root, schema, properties, object, path); err != nil {
			return err
		}
		for name, propertySchemaValue := range properties {
			propertyValue, exists := object[name]
			if !exists {
				continue
			}
			propertySchema, ok := propertySchemaValue.(map[string]any)
			if !ok {
				return fmt.Errorf("%s.%s schema is not an object", path, name)
			}
			if err := validateJSONSchemaValue(root, propertySchema, propertyValue, path+"."+name); err != nil {
				return err
			}
		}
	}
	if err := validateStringConstraints(schema, value, path); err != nil {
		return err
	}
	if err := validateNumericConstraints(schema, value, path); err != nil {
		return err
	}
	if err := validateArrayConstraints(root, schema, value, path); err != nil {
		return err
	}
	return nil
}

func validateConditionalSchema(root, schema map[string]any, value any, path string) error {
	if ifValue, ok := schema["if"].(map[string]any); ok {
		if err := validateJSONSchemaValue(root, ifValue, value, path); err == nil {
			if thenValue, ok := schema["then"].(map[string]any); ok {
				return validateJSONSchemaValue(root, thenValue, value, path)
			}
		}
		return nil
	}
	return validateJSONSchemaValue(root, schema, value, path)
}

func resolveLocalSchemaRef(root map[string]any, ref string) (map[string]any, error) {
	const prefix = "#/$defs/"
	if !strings.HasPrefix(ref, prefix) {
		return nil, fmt.Errorf("unsupported schema ref %q", ref)
	}
	defs, ok := root["$defs"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("schema ref %q has no $defs", ref)
	}
	target, ok := defs[strings.TrimPrefix(ref, prefix)].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("schema ref %q not found", ref)
	}
	return target, nil
}

func validateJSONSchemaType(typeName string, value any, path string) error {
	switch typeName {
	case "object":
		if _, ok := value.(map[string]any); !ok {
			return fmt.Errorf("%s must be an object", path)
		}
	case "array":
		if _, ok := value.([]any); !ok {
			return fmt.Errorf("%s must be an array", path)
		}
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("%s must be a string", path)
		}
	case "integer":
		number, ok := value.(json.Number)
		if !ok {
			return fmt.Errorf("%s must be an integer", path)
		}
		if _, err := number.Int64(); err != nil {
			return fmt.Errorf("%s must be an integer", path)
		}
	case "number":
		if _, ok := value.(json.Number); !ok {
			return fmt.Errorf("%s must be a number", path)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%s must be a boolean", path)
		}
	default:
		return fmt.Errorf("%s uses unsupported schema type %q", path, typeName)
	}
	return nil
}

func validateAdditionalProperties(root, schema map[string]any, properties map[string]any, object map[string]any, path string) error {
	additional, exists := schema["additionalProperties"]
	if !exists {
		return nil
	}
	for name, value := range object {
		if _, known := properties[name]; known {
			continue
		}
		switch typed := additional.(type) {
		case bool:
			if !typed {
				return fmt.Errorf("%s has unexpected property %q", path, name)
			}
		case map[string]any:
			if err := validateJSONSchemaValue(root, typed, value, path+"."+name); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s has unsupported additionalProperties rule", path)
		}
	}
	return nil
}

func validateStringConstraints(schema map[string]any, value any, path string) error {
	text, ok := value.(string)
	if !ok {
		return nil
	}
	if minLength, ok := schema["minLength"].(json.Number); ok {
		min, err := minLength.Int64()
		if err != nil {
			return fmt.Errorf("%s minLength is not an integer", path)
		}
		if int64(len(text)) < min {
			return fmt.Errorf("%s must have length at least %d", path, min)
		}
	}
	if pattern, ok := schema["pattern"].(string); ok {
		matched, err := regexp.MatchString(pattern, text)
		if err != nil {
			return fmt.Errorf("%s has invalid pattern %q: %w", path, pattern, err)
		}
		if !matched {
			return fmt.Errorf("%s must match pattern %q", path, pattern)
		}
	}
	return nil
}

func validateNumericConstraints(schema map[string]any, value any, path string) error {
	number, ok := value.(json.Number)
	if !ok {
		return nil
	}
	actual, err := number.Float64()
	if err != nil {
		return fmt.Errorf("%s must be numeric", path)
	}
	if minimum, ok := schema["minimum"].(json.Number); ok {
		min, err := minimum.Float64()
		if err != nil {
			return fmt.Errorf("%s minimum is not numeric", path)
		}
		if actual < min {
			return fmt.Errorf("%s must be at least %v", path, minimum)
		}
	}
	if maximum, ok := schema["maximum"].(json.Number); ok {
		max, err := maximum.Float64()
		if err != nil {
			return fmt.Errorf("%s maximum is not numeric", path)
		}
		if actual > max {
			return fmt.Errorf("%s must be at most %v", path, maximum)
		}
	}
	return nil
}

func validateArrayConstraints(root, schema map[string]any, value any, path string) error {
	array, ok := value.([]any)
	if !ok {
		return nil
	}
	if minItems, ok := schema["minItems"].(json.Number); ok {
		min, err := minItems.Int64()
		if err != nil {
			return fmt.Errorf("%s minItems is not an integer", path)
		}
		if int64(len(array)) < min {
			return fmt.Errorf("%s must have at least %d items", path, min)
		}
	}
	if maxItems, ok := schema["maxItems"].(json.Number); ok {
		max, err := maxItems.Int64()
		if err != nil {
			return fmt.Errorf("%s maxItems is not an integer", path)
		}
		if int64(len(array)) > max {
			return fmt.Errorf("%s must have at most %d items", path, max)
		}
	}
	if itemSchema, ok := schema["items"].(map[string]any); ok {
		for i, item := range array {
			if err := validateJSONSchemaValue(root, itemSchema, item, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

func jsonValueInEnum(value any, enum []any) bool {
	for _, candidate := range enum {
		if jsonValuesEqual(candidate, value) {
			return true
		}
	}
	return false
}

func jsonValuesEqual(left, right any) bool {
	leftNumber, leftIsNumber := left.(json.Number)
	rightNumber, rightIsNumber := right.(json.Number)
	if leftIsNumber && rightIsNumber {
		return leftNumber.String() == rightNumber.String()
	}
	return reflect.DeepEqual(left, right)
}

func publicSchemaNames() []string {
	return []string{
		"foundry-ao2-loop-decision-v0.1",
		"foundry-active-stack-production-readiness-rollup-v0.1",
		"foundry-active-stack-readiness-v0.1",
		"foundry-atlas-readback-v0.1",
		"foundry-atlas-status-v0.1",
		"foundry-approval-decision-v0.1",
		"foundry-approval-request-v0.1",
		"foundry-approved-live-docs-dry-run-chain-v0.1",
		"foundry-capability-matrix-v0.1",
		"foundry-competitive-readiness-audit-v0.1",
		"foundry-control-plane-readback-v0.1",
		"foundry-eval-result-v0.1",
		"foundry-eval-scorecard-v0.1",
		"foundry-forge-live-attempt-v0.1",
		"foundry-first-live-docs-readiness-rollup-v0.1",
		"foundry-goal-readiness-audit-v0.1",
		"foundry-goal-run-v0.1",
		"foundry-governed-live-mutation-dry-run-chain-v0.1",
		"foundry-live-docs-approval-gate-v0.1",
		"foundry-live-docs-pr-rehearsal-gate-v0.1",
		"foundry-live-docs-rollback-execution-rehearsal-v0.1",
		"foundry-live-docs-worktree-prepare-v0.1",
		"foundry-live-mutation-approval-request-v0.1",
		"foundry-live-mutation-readiness-rollup-v0.1",
		"foundry-loop-event-log-v0.1",
		"foundry-loop-lease-v0.1",
		"foundry-live-mutation-rollback-rehearsal-v0.1",
		"foundry-mutation-class-gate-v0.1",
		"foundry-production-readiness-audit-v0.1",
		"foundry-pulse-event-loop-policy-v0.1",
		"foundry-pulse-intake-preflight-v0.1",
		"foundry-pulse-overnight-start-gate-v0.1",
		"foundry-pulse-pr-lifecycle-v0.1",
		"foundry-pulse-runner-start-decision-v0.1",
		"foundry-pulse-event-v0.1",
		"foundry-registry-v0.1",
		"foundry-release-candidate-v0.1",
		"foundry-release-manifest-v0.1",
		"foundry-release-promotion-v0.1",
		"foundry-repo-health-v0.1",
		"foundry-rsi-candidate-v0.1",
		"foundry-rsi-control-surface-packet-v0.1",
		"foundry-rsi-improvement-gate-v0.1",
		"foundry-rsi-next-improvement-task-v0.1",
		"foundry-run-v0.1",
		"foundry-signed-smoke-ingest-v0.1",
		"foundry-signed-smoke-preflight-v0.1",
		"foundry-signed-smoke-result-v0.1",
		"foundry-signed-smoke-summary-v0.1",
		"foundry-task-v0.1",
		"foundry-trace-v0.1",
		"foundry-worktree-isolation-proof-v0.1",
	}
}

func checkFileExists(name, path string) CompetitiveAuditCheck {
	if _, err := os.Stat(repoPath(path)); err != nil {
		return CompetitiveAuditCheck{Name: name, Status: "fail", Reason: "missing " + path}
	}
	return CompetitiveAuditCheck{Name: name, Status: "pass", Reason: path + " exists"}
}

func checkFileContains(name, path, needle string) CompetitiveAuditCheck {
	data, err := os.ReadFile(repoPath(path))
	if err != nil {
		return CompetitiveAuditCheck{Name: name, Status: "fail", Reason: "missing " + path}
	}
	if !strings.Contains(string(data), needle) {
		return CompetitiveAuditCheck{Name: name, Status: "fail", Reason: path + " does not contain required evidence " + needle}
	}
	return CompetitiveAuditCheck{Name: name, Status: "pass", Reason: path + " contains required evidence"}
}

func checkNoPublicSiblingDependency(name string) CompetitiveAuditCheck {
	data, err := os.ReadFile(repoPath("docs/operations/RELEASE-CHECKLIST.md"))
	if err != nil {
		return CompetitiveAuditCheck{Name: name, Status: "fail", Reason: "missing release checklist"}
	}
	for _, marker := range []string{"../ao-forge", "../ao2", "../ao2-control-plane", "../ao-covenant", "../ao-command"} {
		if strings.Contains(string(data), marker) {
			return CompetitiveAuditCheck{Name: name, Status: "fail", Reason: "release checklist depends on sibling repo " + marker}
		}
	}
	return CompetitiveAuditCheck{Name: name, Status: "pass", Reason: "release checklist avoids sibling repositories"}
}

func repoPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	root, err := repoRoot()
	if err != nil {
		return path
	}
	return filepath.Join(root, filepath.FromSlash(path))
}

func checkPublicSafety(name string) CompetitiveAuditCheck {
	root, err := repoRoot()
	if err != nil {
		return CompetitiveAuditCheck{Name: name, Status: "fail", Reason: err.Error()}
	}
	for _, dir := range []string{"README.md", "docs", "examples", "internal", "cmd"} {
		if err := scanPublicSafetyPath(filepath.Join(root, dir)); err != nil {
			return CompetitiveAuditCheck{Name: name, Status: "fail", Reason: err.Error()}
		}
	}
	return CompetitiveAuditCheck{Name: name, Status: "pass", Reason: "public safety scan has no matches"}
}

func scanPublicSafetyPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	if !info.IsDir() {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return scanPublicSafetyBytes(path, data)
	}
	return filepath.WalkDir(path, func(file string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		return scanPublicSafetyBytes(file, data)
	})
}

func scanPublicSafetyBytes(path string, data []byte) error {
	text := string(data)
	for _, marker := range publicSafetyMarkers() {
		if strings.Contains(text, marker) {
			return fmt.Errorf("%s contains public-safety marker", path)
		}
	}
	return nil
}

func publicSafetyMarkers() []string {
	return []string{
		"excluded" + "/",
		"/" + "Users/",
		"ghp" + "_",
		"github" + "_pat_",
		"BEGIN " + "RSA",
		"BEGIN " + "OPENSSH",
		"ANTI" + "GRAVITY",
		"private " + "handoff",
		"api" + "_key",
		"api" + "-key",
		"access" + "_token",
		"access" + "-token",
	}
}

func scoreDimension(def EvalDimensionDef, run FoundryRun) EvalDimension {
	pass := func(reason, evidence string) EvalDimension {
		return EvalDimension{Name: def.Name, Score: def.MaxScore, MaxScore: def.MaxScore, Status: "pass", Reason: reason, Evidence: evidence}
	}
	fail := func(reason string) EvalDimension {
		return EvalDimension{Name: def.Name, Score: 0, MaxScore: def.MaxScore, Status: "fail", Reason: reason}
	}
	switch def.Name {
	case "correctness":
		if run.Status == "passed" {
			return pass("run completed with passed status", run.ForgePacket.Path)
		}
		return fail("run status is not passed")
	case "verification_strength":
		if len(run.Evidence) > 0 {
			return pass("run includes verification evidence", run.Evidence[0].Path)
		}
		return fail("run has no verification evidence")
	case "evidence_completeness":
		if run.ForgePacket.Path != "" && run.ForgePacket.SHA256 != "" && len(run.Evidence) > 0 {
			return pass("Forge packet and evidence digests are present", run.ForgePacket.Path)
		}
		return fail("missing Forge packet evidence")
	case "policy_safety":
		for _, decision := range run.Decisions {
			if decision.Decision == "deny" {
				return fail("policy decision denied the run")
			}
		}
		return pass("policy decisions did not deny the run", run.ForgePacket.Path)
	case "replayability":
		if run.ForgePacket.SHA256 != "" {
			return pass("packet digest supports replay", run.ForgePacket.Path)
		}
		return fail("missing packet digest")
	case "recovery_behavior":
		if len(run.NextActions) > 0 {
			return pass("run records next actions for recovery", run.ForgePacket.Path)
		}
		return fail("run lacks recovery next actions")
	case "cost_budget", "latency_budget":
		return pass("no budget breach recorded in run", run.ForgePacket.Path)
	case "operator_clarity":
		if len(run.Decisions) > 0 && run.Decisions[0].Explanation != "" {
			return pass("policy decision includes operator-readable explanation", run.ForgePacket.Path)
		}
		return fail("missing operator-readable explanation")
	case "public_safety_cleanliness":
		for _, evidence := range run.Evidence {
			if err := validateEvidencePath(evidence.Path); err != nil {
				return fail("unsafe evidence path")
			}
		}
		return pass("evidence paths are public-safe", run.ForgePacket.Path)
	default:
		return fail("unknown score dimension " + def.Name)
	}
}

func loopPreflight(goalPath, registryPath, taskPath string) error {
	goal, err := loadGoalRun(goalPath)
	if err != nil {
		return fmt.Errorf("goal: %w", err)
	}
	goalAudit, err := buildGoalReadinessAudit(goalPath, registryPath, taskPath)
	if err != nil {
		return err
	}
	if goalAudit.Score != goalAudit.MaxScore {
		return fmt.Errorf("goal readiness below 100")
	}
	readinessAudit, err := buildReadinessAudit(registryPath, taskPath)
	if err != nil {
		return err
	}
	if readinessAudit.Score != readinessAudit.MaxScore {
		return fmt.Errorf("production readiness below 100")
	}
	if err := validateLoopPolicy(goal.LoopPolicy); err != nil {
		return err
	}
	return nil
}

func validateLoopPolicy(policy LoopPolicy) error {
	if policy.MaxIterations > 0 && policy.Iterations >= policy.MaxIterations {
		return fmt.Errorf("max iterations exhausted")
	}
	if policy.MaxElapsedMinutes > 0 && policy.ElapsedMinutes >= policy.MaxElapsedMinutes {
		return fmt.Errorf("max elapsed time exhausted")
	}
	if policy.MaxSpendCents > 0 && policy.SpendCents >= policy.MaxSpendCents {
		return fmt.Errorf("spend budget exhausted")
	}
	return nil
}

func acquireLoopLease(goalPath, leasePath string) (LoopLease, error) {
	if leasePath == "" {
		return LoopLease{}, errors.New("missing --lease")
	}
	goal, err := loadGoalRun(goalPath)
	if err != nil {
		return LoopLease{}, fmt.Errorf("goal: %w", err)
	}
	if existing, err := loadLoopLease(leasePath); err == nil && existing.Status == "active" {
		if isLeaseStale(existing) {
			return LoopLease{}, fmt.Errorf("stale lease exists at %s; release or recover it explicitly", leasePath)
		}
		return LoopLease{}, fmt.Errorf("active lease exists at %s", leasePath)
	} else if err != nil && !os.IsNotExist(err) {
		return LoopLease{}, err
	}
	now := time.Now().UTC()
	lease := LoopLease{
		SchemaVersion: loopLeaseSchema,
		GoalID:        goal.GoalID,
		LeaseID:       "loop-lease-" + shortSHA256(goal.GoalID+":"+now.Format(time.RFC3339Nano)),
		AcquiredAtUTC: now.Format(time.RFC3339),
		ExpiresAtUTC:  now.Add(8 * time.Hour).Format(time.RFC3339),
		Status:        "active",
	}
	if err := os.MkdirAll(parentDir(leasePath), 0o755); err != nil {
		return LoopLease{}, err
	}
	file, err := os.OpenFile(leasePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return LoopLease{}, fmt.Errorf("active lease exists at %s", leasePath)
		}
		return LoopLease{}, err
	}
	defer file.Close()
	if err := writeJSON(file, lease); err != nil {
		return LoopLease{}, err
	}
	return lease, nil
}

func loadLoopLease(path string) (LoopLease, error) {
	var lease LoopLease
	if err := readJSONFile(path, &lease); err != nil {
		return lease, err
	}
	if lease.SchemaVersion != loopLeaseSchema {
		return lease, fmt.Errorf("lease schema_version must be %s", loopLeaseSchema)
	}
	return lease, nil
}

func isLeaseStale(lease LoopLease) bool {
	expires, err := time.Parse(time.RFC3339, lease.ExpiresAtUTC)
	if err != nil {
		return false
	}
	return time.Now().UTC().After(expires)
}

func readRepoHealth(repo Repo) RepoHealth {
	health := RepoHealth{
		RepoID:      repo.ID,
		Workspace:   repo.Workspace,
		Status:      "ready",
		Checks:      []RepoHealthCheck{},
		NextActions: []string{},
	}
	if !healthConfigured(repo.Health) {
		health.addCheck("health_config", "unknown", "repo has no configured health readers")
		health.finalize()
		return health
	}
	workspace := resolveWorkspacePath(repo.Workspace)
	if _, err := os.Stat(workspace); err != nil {
		health.addCheck("workspace_exists", "blocked", err.Error())
		health.finalize()
		return health
	}
	if _, err := os.Stat(filepath.Join(workspace, ".git")); err != nil {
		health.addCheck("git_repository", "blocked", "workspace is not a git repository")
		health.finalize()
		return health
	}
	health.addCheck("workspace_exists", "pass", "workspace exists")
	health.addCheck("git_repository", "pass", "git metadata exists")

	if branch, err := gitOutput(workspace, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		health.CurrentBranch = branch
		health.addCheck("current_branch", "pass", branch)
	} else {
		health.addCheck("current_branch", "blocked", err.Error())
	}

	for _, branch := range repo.Branches {
		if out, err := gitOutput(workspace, "branch", "--list", branch); err == nil && strings.TrimSpace(out) != "" {
			health.addCheck("branch_present", "pass", branch)
		} else if err != nil {
			health.addCheck("branch_present", "blocked", err.Error())
		} else {
			health.addCheck("branch_present", "blocked", "missing branch "+branch)
		}
	}

	if repo.Health.RequireCleanWorktree {
		if out, err := gitOutput(workspace, "status", "--short"); err == nil && out == "" {
			health.addCheck("clean_worktree", "pass", "worktree is clean")
		} else if err != nil {
			health.addCheck("clean_worktree", "blocked", err.Error())
		} else {
			health.addCheck("clean_worktree", "blocked", "worktree has local changes")
		}
	} else {
		health.addCheck("clean_worktree", "unknown", "clean worktree is not required by registry")
	}

	for _, tag := range repo.Health.RequireTags {
		if out, err := gitOutput(workspace, "tag", "--list", tag); err == nil && strings.TrimSpace(out) != "" {
			health.addCheck("tag_present", "pass", tag)
		} else if err != nil {
			health.addCheck("tag_present", "blocked", err.Error())
		} else {
			health.addCheck("tag_present", "blocked", "missing tag "+tag)
		}
	}

	for _, command := range repo.Health.VerificationCommands {
		executable := strings.Fields(command)
		if len(executable) == 0 {
			health.addCheck("verification_command_exists", "blocked", "empty verification command")
			continue
		}
		if _, err := exec.LookPath(executable[0]); err == nil {
			health.addCheck("verification_command_exists", "pass", command)
		} else {
			health.addCheck("verification_command_exists", "blocked", "missing executable "+executable[0])
		}
	}

	for _, file := range repo.Health.ReadinessFiles {
		if _, err := os.Stat(filepath.Join(workspace, filepath.FromSlash(file))); err == nil {
			health.addCheck("readiness_file_exists", "pass", file)
		} else {
			health.addCheck("readiness_file_exists", "blocked", "missing readiness file "+file)
		}
	}

	if repo.Health.GitHubActions {
		if !repo.Health.AllowNetworkRead {
			health.addCheck("github_actions_status", "unknown", "network read is not permitted")
		} else if _, err := exec.LookPath("gh"); err != nil {
			health.addCheck("github_actions_status", "unknown", "gh command is unavailable")
		} else {
			health.addCheck("github_actions_status", "unknown", "GitHub Actions reader is configured but no workflow query was run")
		}
	}

	health.finalize()
	return health
}

func healthConfigured(config HealthReaderConfig) bool {
	return config.RequireCleanWorktree ||
		len(config.VerificationCommands) > 0 ||
		len(config.ReadinessFiles) > 0 ||
		len(config.RequireTags) > 0 ||
		config.AllowNetworkRead ||
		config.GitHubActions
}

func resolveWorkspacePath(workspace string) string {
	if filepath.IsAbs(workspace) {
		return workspace
	}
	root, err := repoRoot()
	if err != nil {
		return workspace
	}
	return filepath.Clean(filepath.Join(root, filepath.FromSlash(workspace)))
}

func (health *RepoHealth) addCheck(name, status, detail string) {
	health.Checks = append(health.Checks, RepoHealthCheck{Name: name, Status: status, Detail: detail})
}

func (health *RepoHealth) finalize() {
	health.Status = "ready"
	for _, check := range health.Checks {
		if check.Status == "blocked" {
			health.Status = "blocked"
			health.NextActions = append(health.NextActions, fmt.Sprintf("%s: %s", check.Name, check.Detail))
		}
	}
}

func gitOutput(workspace string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", workspace}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func packetMapsToTask(packet ForgePacket, registry Registry, task Task) error {
	target, err := primaryTargetRepo(registry, task)
	if err != nil {
		return err
	}
	if packet.Objective.Text != task.Objective {
		return errors.New("packet objective does not map to task objective")
	}
	if packet.Objective.Workspace != target.Workspace {
		return fmt.Errorf("packet workspace %q does not map to target workspace %q", packet.Objective.Workspace, target.Workspace)
	}
	return nil
}

func shortSHA256(value string) string {
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", sum[:])[:12]
}

func fileSHA256(path string) (string, error) {
	if data, err := os.ReadFile(path); err == nil {
		sum := sha256.Sum256(data)
		return fmt.Sprintf("%x", sum[:]), nil
	}
	root, err := repoRoot()
	if err != nil {
		return "", err
	}
	cleaned := filepath.Clean(filepath.FromSlash(strings.ReplaceAll(path, "\\", "/")))
	data, err := os.ReadFile(filepath.Join(root, cleaned))
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:]), nil
}

func repoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(wd + string(os.PathSeparator) + "go.mod"); err == nil {
			return wd, nil
		}
		parent := wd[:strings.LastIndex(wd, string(os.PathSeparator))]
		if parent == "" || parent == wd {
			return "", errors.New("could not locate repository root")
		}
		wd = parent
	}
}

func errReason(err error, ok string) string {
	if err == nil {
		return ok
	}
	return err.Error()
}

func (audit *ReadinessAudit) finalize() {
	score := 0
	next := []string{}
	for _, check := range audit.Checks {
		score += check.Score
		if check.Status != "pass" {
			next = append(next, check.Reason)
		}
	}
	audit.Score = score
	if score == audit.MaxScore {
		audit.Status = "ready"
	} else {
		audit.Status = "blocked"
	}
	audit.NextActions = next
}

func targetReposReady(task Task, registry Registry) error {
	repos := map[string]Repo{}
	for _, repo := range registry.Repos {
		repos[repo.ID] = repo
	}
	var blocked []string
	for _, id := range task.TargetRepos {
		repo, ok := repos[id]
		if !ok {
			continue
		}
		for _, signal := range repo.ReadinessSignals {
			if signal.Status != "ready" {
				blocked = append(blocked, fmt.Sprintf("%s:%s=%s", id, signal.Name, signal.Status))
			}
		}
	}
	if len(blocked) > 0 {
		sort.Strings(blocked)
		return fmt.Errorf("target readiness not ready: %s", strings.Join(blocked, ", "))
	}
	return nil
}

func forgeDelegationReady(task Task) error {
	for _, delegation := range task.RequiredDelegation {
		if delegation.DelegateTo == "ao-forge" {
			return nil
		}
	}
	return errors.New("task does not delegate governed work to ao-forge")
}
