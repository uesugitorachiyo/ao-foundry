package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

type pulseRefactorClosurePacket struct {
	SchemaVersion                string                `json:"schema_version"`
	Status                       string                `json:"status"`
	AllowedNextAction            string                `json:"allowed_next_action"`
	FirstFailingCheck            string                `json:"first_failing_check"`
	BlockingNextActions          []string              `json:"blocking_next_actions"`
	BlueprintAuthorizationStatus string                `json:"blueprint_authorization_status"`
	AtlasSchedulerInputStatus    string                `json:"atlas_scheduler_input_status"`
	AtlasSchedulerStatus         string                `json:"atlas_scheduler_status"`
	IntakePreflightStatus        string                `json:"intake_preflight_status"`
	StartGateStatus              string                `json:"start_gate_status"`
	RunnerDecisionStatus         string                `json:"runner_decision_status"`
	EventLoopPolicyStatus        string                `json:"event_loop_policy_status"`
	CommandReadbackStatus        string                `json:"command_readback_status"`
	SelectedNodeID               string                `json:"selected_node_id"`
	SelectedTaskID               string                `json:"selected_task_id"`
	ClosesRefactorSlices         []string              `json:"closes_refactor_slices"`
	SourceDigests                []pulseArtifactSource `json:"source_digests"`
	SchedulesWork                bool                  `json:"schedules_work"`
	ExecutesWork                 bool                  `json:"executes_work"`
	ApprovesWork                 bool                  `json:"approves_work"`
	MutatesRepositories          bool                  `json:"mutates_repositories"`
	CallsProviders               bool                  `json:"calls_providers"`
	UploadsArtifacts             bool                  `json:"uploads_artifacts"`
	OpensPR                      bool                  `json:"opens_pr"`
	MergesPR                     bool                  `json:"merges_pr"`
}

func runPulseClosurePacket(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("pulse closure-packet", stderr)
	blueprintAuthorizationPath := fs.String("blueprint-authorization", "", "Blueprint authorization artifact")
	atlasSchedulerPath := fs.String("atlas-scheduler-input", "", "Pulse Atlas scheduler input artifact")
	intakePreflightPath := fs.String("intake-preflight", "", "Pulse intake preflight artifact")
	startGatePath := fs.String("start-gate", "", "Pulse overnight start gate artifact")
	runnerDecisionPath := fs.String("runner-decision", "", "Pulse runner start decision artifact")
	eventLoopPolicyPath := fs.String("event-loop-policy", "", "Pulse event-loop policy artifact")
	commandReadbackPath := fs.String("command-readback", "", "optional AO Command readback artifact")
	outPath := fs.String("out", "", "closure packet output path")
	jsonOut := fs.Bool("json", false, "emit JSON result to stdout")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	required := []string{*blueprintAuthorizationPath, *atlasSchedulerPath, *intakePreflightPath, *startGatePath, *runnerDecisionPath, *eventLoopPolicyPath, *outPath}
	for _, value := range required {
		if strings.TrimSpace(value) == "" {
			fmt.Fprintln(stderr, "pulse closure-packet: --blueprint-authorization, --atlas-scheduler-input, --intake-preflight, --start-gate, --runner-decision, --event-loop-policy, and --out are required")
			return 2
		}
	}
	for _, input := range []string{*blueprintAuthorizationPath, *atlasSchedulerPath, *intakePreflightPath, *startGatePath, *runnerDecisionPath, *eventLoopPolicyPath, *commandReadbackPath} {
		if strings.TrimSpace(input) != "" && sameCleanPath(*outPath, input) {
			fmt.Fprintln(stderr, "pulse closure-packet: --out must not overwrite input artifacts")
			return 2
		}
	}
	result, err := buildPulseClosurePacket(map[string]string{
		"blueprint_authorization": *blueprintAuthorizationPath,
		"atlas_scheduler_input":   *atlasSchedulerPath,
		"intake_preflight":        *intakePreflightPath,
		"start_gate":              *startGatePath,
		"runner_decision":         *runnerDecisionPath,
		"event_loop_policy":       *eventLoopPolicyPath,
		"command_readback":        *commandReadbackPath,
	})
	if writeErr := writeJSONFile(*outPath, result); writeErr != nil {
		fmt.Fprintf(stderr, "pulse closure-packet: write result: %v\n", writeErr)
		return 2
	}
	if *jsonOut {
		if writeErr := writeJSON(stdout, result); writeErr != nil {
			fmt.Fprintf(stderr, "pulse closure-packet: marshal result: %v\n", writeErr)
			return 2
		}
	} else {
		fmt.Fprintf(stdout, "pulse_refactor_closure_packet=%s\n", result.Status)
		fmt.Fprintf(stdout, "allowed_next_action=%s\n", result.AllowedNextAction)
		fmt.Fprintf(stdout, "selected_node_id=%s\n", result.SelectedNodeID)
		fmt.Fprintf(stdout, "closure_packet=%s\n", *outPath)
	}
	if err != nil {
		fmt.Fprintf(stderr, "pulse closure-packet: %v\n", err)
		return 1
	}
	if result.Status != "ready" {
		return 1
	}
	return 0
}

func buildPulseClosurePacket(paths map[string]string) (pulseRefactorClosurePacket, error) {
	result := basePulseClosurePacket()
	required := []pulseClosureExpectedSource{
		{key: "blueprint_authorization", schema: "ao.blueprint.build-authorization.v0.1", statusField: "status", statusValues: []string{"ready"}, checkName: "blueprint_authorization"},
		{key: "atlas_scheduler_input", schema: pulseAtlasSchedulerInputSchema, statusField: "status", statusValues: []string{"ready"}, checkName: "atlas_scheduler_input"},
		{key: "intake_preflight", schema: pulseIntakeSchema, statusField: "status", statusValues: []string{"ready"}, checkName: "intake_preflight"},
		{key: "start_gate", schema: pulseStartGateSchema, statusField: "status", statusValues: []string{"ready"}, checkName: "start_gate"},
		{key: "runner_decision", schema: pulseRunnerSchema, statusField: "status", statusValues: []string{"ready"}, checkName: "runner_decision"},
		{key: "event_loop_policy", schema: pulseLoopPolicySchema, statusField: "status", statusValues: []string{"ready"}, checkName: "event_loop_policy"},
	}
	objects := map[string]map[string]any{}
	for _, requiredSource := range required {
		object, source, err := loadPulseJSONSource(requiredSource.key, paths[requiredSource.key])
		if err != nil {
			return failPulseClosurePacket(result, requiredSource.checkName, err.Error())
		}
		result.SourceDigests = append(result.SourceDigests, source)
		objects[requiredSource.key] = object
		schema := classGateFirstNonEmpty(classGateString(object, "schema_version"), classGateString(object, "contract_version"), classGateString(object, "schema"))
		if schema != requiredSource.schema {
			return failPulseClosurePacket(result, requiredSource.checkName, fmt.Sprintf("%s schema must be %s", requiredSource.key, requiredSource.schema))
		}
		status := classGateString(object, requiredSource.statusField)
		if !classGateStringSliceContains(requiredSource.statusValues, status) {
			return failPulseClosurePacket(result, requiredSource.checkName, fmt.Sprintf("%s status is %s", requiredSource.key, status))
		}
		result.setSourceStatus(requiredSource.key, status)
	}
	if strings.TrimSpace(paths["command_readback"]) != "" {
		object, source, err := loadPulseJSONSource("command_readback", paths["command_readback"])
		if err != nil {
			return failPulseClosurePacket(result, "command_readback", err.Error())
		}
		status := classGateString(object, "status")
		if !classGateStringSliceContains([]string{"ready", "accepted"}, status) {
			return failPulseClosurePacket(result, "command_readback", "command_readback status is "+status)
		}
		result.SourceDigests = append(result.SourceDigests, source)
		result.CommandReadbackStatus = status
	}

	scheduler := objects["atlas_scheduler_input"]
	policy := objects["event_loop_policy"]
	if !classGateBool(scheduler, "atlas_compile_only") {
		return failPulseClosurePacket(result, "atlas_scheduler_input", "atlas scheduler input must preserve atlas_compile_only=true")
	}
	for _, field := range []string{"schedules_work", "executes_work", "approves_work", "mutates_repositories", "calls_providers", "uploads_artifacts", "opens_pr", "merges_pr"} {
		if classGateBool(scheduler, field) {
			return failPulseClosurePacket(result, "atlas_authority_boundary", "atlas scheduler input must not grant "+field)
		}
	}
	if !classGateBool(policy, "safe_to_continue") {
		return failPulseClosurePacket(result, "event_loop_policy", "event_loop_policy safe_to_continue is false")
	}
	result.SelectedNodeID = classGateString(scheduler, "selected_node_id")
	result.SelectedTaskID = classGateString(scheduler, "selected_task_id")
	if result.SelectedNodeID == "" || result.SelectedTaskID == "" {
		return failPulseClosurePacket(result, "atlas_scheduler_input", "atlas scheduler input requires selected_node_id and selected_task_id")
	}
	result.Status = "ready"
	result.AllowedNextAction = "continue_next_slice"
	result.FirstFailingCheck = ""
	result.BlockingNextActions = []string{}
	return result, nil
}

type pulseClosureExpectedSource struct {
	key          string
	schema       string
	statusField  string
	statusValues []string
	checkName    string
}

func (packet *pulseRefactorClosurePacket) setSourceStatus(key, status string) {
	switch key {
	case "blueprint_authorization":
		packet.BlueprintAuthorizationStatus = status
	case "atlas_scheduler_input":
		packet.AtlasSchedulerInputStatus = status
		packet.AtlasSchedulerStatus = status
	case "intake_preflight":
		packet.IntakePreflightStatus = status
	case "start_gate":
		packet.StartGateStatus = status
	case "runner_decision":
		packet.RunnerDecisionStatus = status
	case "event_loop_policy":
		packet.EventLoopPolicyStatus = status
	}
}

func basePulseClosurePacket() pulseRefactorClosurePacket {
	return pulseRefactorClosurePacket{
		SchemaVersion:                pulseRefactorClosurePacketSchema,
		Status:                       "blocked",
		AllowedNextAction:            "stop_event_loop",
		BlockingNextActions:          []string{},
		BlueprintAuthorizationStatus: "missing",
		AtlasSchedulerInputStatus:    "missing",
		AtlasSchedulerStatus:         "missing",
		IntakePreflightStatus:        "missing",
		StartGateStatus:              "missing",
		RunnerDecisionStatus:         "missing",
		EventLoopPolicyStatus:        "missing",
		CommandReadbackStatus:        "not_provided",
		ClosesRefactorSlices:         []string{"D", "E"},
		SourceDigests:                []pulseArtifactSource{},
		SchedulesWork:                false,
		ExecutesWork:                 false,
		ApprovesWork:                 false,
		MutatesRepositories:          false,
		CallsProviders:               false,
		UploadsArtifacts:             false,
		OpensPR:                      false,
		MergesPR:                     false,
	}
}

func failPulseClosurePacket(result pulseRefactorClosurePacket, checkName, reason string) (pulseRefactorClosurePacket, error) {
	result.Status = "blocked"
	result.AllowedNextAction = "stop_event_loop"
	result.FirstFailingCheck = checkName
	result.BlockingNextActions = []string{"Stop the Pulse event loop until closure evidence is regenerated and ready."}
	return result, errors.New(reason)
}
