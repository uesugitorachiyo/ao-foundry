package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

type pulseAtlasSchedulerInput struct {
	SchemaVersion         string                        `json:"schema_version"`
	Status                string                        `json:"status"`
	AllowedNextAction     string                        `json:"allowed_next_action"`
	FirstFailingCheck     string                        `json:"first_failing_check"`
	BlockingNextActions   []string                      `json:"blocking_next_actions"`
	SelectedNodeID        string                        `json:"selected_node_id"`
	SelectedTaskID        string                        `json:"selected_task_id"`
	AtlasCompileOnly      bool                          `json:"atlas_compile_only"`
	SafeNodeFoundryImport pulseSafeNodeFoundryImport    `json:"safe_node_foundry_import"`
	TaskCounts            pulseAtlasSchedulerTaskCounts `json:"task_counts"`
	SourceDigests         []pulseArtifactSource         `json:"source_digests"`
	SchedulesWork         bool                          `json:"schedules_work"`
	ExecutesWork          bool                          `json:"executes_work"`
	ApprovesWork          bool                          `json:"approves_work"`
	MutatesRepositories   bool                          `json:"mutates_repositories"`
	CallsProviders        bool                          `json:"calls_providers"`
	UploadsArtifacts      bool                          `json:"uploads_artifacts"`
	OpensPR               bool                          `json:"opens_pr"`
	MergesPR              bool                          `json:"merges_pr"`
}

type pulseSafeNodeFoundryImport struct {
	Status                      string `json:"status"`
	SelectedNodeID              string `json:"selected_node_id"`
	SelectedTaskID              string `json:"selected_task_id"`
	ImportedTaskCount           int    `json:"imported_task_count"`
	RejectsUnselectedReadyNodes bool   `json:"rejects_unselected_ready_nodes"`
}

type pulseAtlasSchedulerTaskCounts struct {
	Total     int `json:"total"`
	Ready     int `json:"ready"`
	Blocked   int `json:"blocked"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
}

func runPulseAtlasSchedulerInput(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("pulse atlas-scheduler-input", stderr)
	workgraphPath := fs.String("workgraph", "", "Atlas workgraph artifact")
	foundryImportPath := fs.String("foundry-import", "", "Atlas Foundry import artifact")
	outPath := fs.String("out", "", "scheduler input output path")
	jsonOut := fs.Bool("json", false, "emit JSON result to stdout")
	if !parseFlags(fs, args, stderr) {
		return 2
	}
	if strings.TrimSpace(*workgraphPath) == "" || strings.TrimSpace(*foundryImportPath) == "" || strings.TrimSpace(*outPath) == "" {
		fmt.Fprintln(stderr, "pulse atlas-scheduler-input: --workgraph, --foundry-import, and --out are required")
		return 2
	}
	if sameCleanPath(*outPath, *workgraphPath) || sameCleanPath(*outPath, *foundryImportPath) {
		fmt.Fprintln(stderr, "pulse atlas-scheduler-input: --out must not overwrite input artifacts")
		return 2
	}
	result, err := buildPulseAtlasSchedulerInput(*workgraphPath, *foundryImportPath)
	if writeErr := writeJSONFile(*outPath, result); writeErr != nil {
		fmt.Fprintf(stderr, "pulse atlas-scheduler-input: write result: %v\n", writeErr)
		return 2
	}
	if *jsonOut {
		if writeErr := writeJSON(stdout, result); writeErr != nil {
			fmt.Fprintf(stderr, "pulse atlas-scheduler-input: marshal result: %v\n", writeErr)
			return 2
		}
	} else {
		fmt.Fprintf(stdout, "pulse_atlas_scheduler_input=%s\n", result.Status)
		fmt.Fprintf(stdout, "allowed_next_action=%s\n", result.AllowedNextAction)
		fmt.Fprintf(stdout, "selected_node_id=%s\n", result.SelectedNodeID)
		fmt.Fprintf(stdout, "scheduler_input=%s\n", *outPath)
	}
	if err != nil {
		fmt.Fprintf(stderr, "pulse atlas-scheduler-input: %v\n", err)
		return 1
	}
	if result.Status != "ready" {
		return 1
	}
	return 0
}

func buildPulseAtlasSchedulerInput(workgraphPath, foundryImportPath string) (pulseAtlasSchedulerInput, error) {
	result := basePulseAtlasSchedulerInput()
	workgraph, workgraphSource, err := loadPulseJSONSource("atlas_workgraph", workgraphPath)
	if err != nil {
		return failPulseAtlasSchedulerInput(result, "atlas_workgraph", err.Error())
	}
	result.SourceDigests = append(result.SourceDigests, workgraphSource)
	foundryImport, importSource, err := loadPulseJSONSource("foundry_import", foundryImportPath)
	if err != nil {
		return failPulseAtlasSchedulerInput(result, "foundry_import", err.Error())
	}
	result.SourceDigests = append(result.SourceDigests, importSource)
	if classGateString(foundryImport, "contract_version") != atlasImportSchema {
		return failPulseAtlasSchedulerInput(result, "foundry_import", "foundry import contract_version must be "+atlasImportSchema)
	}
	if status := classGateString(foundryImport, "status"); status != "ready_for_foundry_fixture_import" && status != "ready" {
		return failPulseAtlasSchedulerInput(result, "foundry_import", "foundry import status must be ready_for_foundry_fixture_import or ready")
	}
	if classGateString(workgraph, "id") != "" && classGateString(foundryImport, "workgraph_id") != "" &&
		classGateString(workgraph, "id") != classGateString(foundryImport, "workgraph_id") {
		return failPulseAtlasSchedulerInput(result, "workgraph_identity", "foundry import workgraph_id must match workgraph id")
	}

	var selectedNode map[string]any
	for _, node := range classGateObjectSlice(workgraph["nodes"]) {
		switch classGateString(node, "status") {
		case "ready":
			result.TaskCounts.Ready++
			if selectedNode == nil {
				selectedNode = node
			}
		case "blocked":
			result.TaskCounts.Blocked++
		case "completed":
			result.TaskCounts.Completed++
		case "failed":
			result.TaskCounts.Failed++
		}
		result.TaskCounts.Total++
	}
	if selectedNode == nil {
		return failPulseAtlasSchedulerInput(result, "atlas_workgraph_ready_node", "Atlas workgraph has no ready node")
	}
	selectedNodeID := classGateString(selectedNode, "id")
	selectedTaskID := classGateNestedString(selectedNode, "factory_task", "id")
	if selectedNodeID == "" || selectedTaskID == "" {
		return failPulseAtlasSchedulerInput(result, "atlas_workgraph_ready_node", "Atlas ready node requires id and factory_task.id")
	}

	var matchingTasks []map[string]any
	for _, task := range classGateObjectSlice(foundryImport["tasks"]) {
		if classGateString(task, "node_id") == selectedNodeID {
			matchingTasks = append(matchingTasks, task)
		}
	}
	result.SelectedNodeID = selectedNodeID
	result.SelectedTaskID = selectedTaskID
	result.SafeNodeFoundryImport.SelectedNodeID = selectedNodeID
	result.SafeNodeFoundryImport.SelectedTaskID = selectedTaskID
	result.SafeNodeFoundryImport.ImportedTaskCount = len(matchingTasks)
	if len(matchingTasks) != 1 {
		return failPulseAtlasSchedulerInput(result, "foundry_import_selected_node", "Foundry import must contain exactly one task for the Atlas-selected ready node")
	}
	if classGateString(matchingTasks[0], "task_id") != selectedTaskID {
		return failPulseAtlasSchedulerInput(result, "foundry_import_selected_node", "Foundry import selected task_id must match Atlas factory_task.id")
	}

	result.Status = "ready"
	result.AllowedNextAction = "start_next_slice"
	result.FirstFailingCheck = ""
	result.BlockingNextActions = []string{}
	result.SafeNodeFoundryImport.Status = "ready"
	result.SafeNodeFoundryImport.ImportedTaskCount = 1
	return result, nil
}

func basePulseAtlasSchedulerInput() pulseAtlasSchedulerInput {
	return pulseAtlasSchedulerInput{
		SchemaVersion:       pulseAtlasSchedulerInputSchema,
		Status:              "blocked",
		AllowedNextAction:   "stop_blocked",
		BlockingNextActions: []string{},
		AtlasCompileOnly:    true,
		SafeNodeFoundryImport: pulseSafeNodeFoundryImport{
			Status:                      "blocked",
			RejectsUnselectedReadyNodes: true,
		},
		SourceDigests:       []pulseArtifactSource{},
		SchedulesWork:       false,
		ExecutesWork:        false,
		ApprovesWork:        false,
		MutatesRepositories: false,
		CallsProviders:      false,
		UploadsArtifacts:    false,
		OpensPR:             false,
		MergesPR:            false,
	}
}

func failPulseAtlasSchedulerInput(result pulseAtlasSchedulerInput, checkName, reason string) (pulseAtlasSchedulerInput, error) {
	result.Status = "blocked"
	result.AllowedNextAction = "stop_blocked"
	result.FirstFailingCheck = checkName
	result.BlockingNextActions = []string{"Regenerate Atlas workgraph and Foundry import so exactly one Atlas-selected ready node is present."}
	return result, errors.New(reason)
}
