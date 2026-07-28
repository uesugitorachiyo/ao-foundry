package cli

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type architectureBaseline struct {
	SchemaVersion string            `json:"schema_version"`
	SourceTree    string            `json:"source_tree"`
	PriorSources  map[string]string `json:"prior_sources"`
	Measurements  map[string]struct {
		Lines        int      `json:"lines"`
		Declarations []string `json:"owned_declarations"`
	} `json:"measurements"`
}

func TestArchitectureOwnershipRatchet(t *testing.T) {
	const (
		sourceMovementTree            = "6d6d9aa029d6523748855be9c964c2733c9b9ef1"
		fullyUnsupervisedMergedCommit = "d84012307e9853b409cc01242ec2ff05f803baaa"
	)

	root := repoPath(".")
	baselinePath := filepath.Join(root, ".github", "architecture-baseline.json")
	body, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatalf("read architecture baseline: %v", err)
	}
	var baseline architectureBaseline
	if err := json.Unmarshal(body, &baseline); err != nil {
		t.Fatalf("decode architecture baseline: %v", err)
	}
	if baseline.SchemaVersion != "ao-foundry.go-architecture-baseline.v2" {
		t.Fatalf("unexpected architecture baseline schema: %q", baseline.SchemaVersion)
	}
	if baseline.SourceTree != sourceMovementTree {
		t.Fatalf("architecture baseline source tree drifted: %q != %q", baseline.SourceTree, sourceMovementTree)
	}
	if baseline.PriorSources["fully_unsupervised_merged_commit"] != fullyUnsupervisedMergedCommit {
		t.Fatalf("fully-unsupervised provenance drifted: %q", baseline.PriorSources["fully_unsupervised_merged_commit"])
	}
	if len(baseline.Measurements) == 0 {
		t.Fatal("architecture baseline has no measurements")
	}

	expectedOwnership := make(map[string][]string, len(baseline.Measurements))
	for relative, expected := range baseline.Measurements {
		if len(expected.Declarations) == 0 {
			t.Errorf("%s has no expected owned declarations", relative)
		}
		path := filepath.Join(root, filepath.FromSlash(relative))
		source, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read measured source %s: %v", relative, err)
			continue
		}
		lines := strings.Count(string(source), "\n")
		if lines > expected.Lines {
			t.Errorf("%s grew above architecture ratchet: %d > %d lines", relative, lines, expected.Lines)
		}
		expectedOwnership[relative] = expected.Declarations
	}

	actualOwnership := map[string][]string{}
	cliDir := filepath.Join(root, "internal", "cli")
	entries, err := os.ReadDir(cliDir)
	if err != nil {
		t.Fatalf("read CLI package: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(cliDir, entry.Name())
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Errorf("parse CLI source %s: %v", entry.Name(), err)
			continue
		}
		actual := measuredArchitectureDeclarations(parsed)
		if len(actual) != 0 {
			relative := filepath.ToSlash(filepath.Join("internal", "cli", entry.Name()))
			actualOwnership[relative] = actual
		}
	}
	if !equalOwnership(actualOwnership, expectedOwnership) {
		t.Errorf("architecture ownership drifted:\nactual:   %v\nexpected: %v", actualOwnership, expectedOwnership)
	}
}

func measuredArchitectureDeclarations(file *ast.File) []string {
	var result []string
	for _, declaration := range file.Decls {
		switch value := declaration.(type) {
		case *ast.FuncDecl:
			if measuredArchitectureName(value.Name.Name) {
				result = append(result, "func:"+value.Name.Name)
			}
		case *ast.GenDecl:
			for _, spec := range value.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if ok && measuredArchitectureName(typeSpec.Name.Name) {
					result = append(result, "type:"+typeSpec.Name.Name)
				}
			}
		}
	}
	sort.Strings(result)
	return result
}

func measuredArchitectureName(name string) bool {
	if strings.Contains(strings.ToLower(name), "fullyunsupervised") || name == "mapStatus" {
		return true
	}
	prefixes := []string{
		"MutationClass",
		"LowRiskCode",
		"MultiRepoLiveRehearsal",
		"ComplexRepo",
		"runClassGate",
		"runComplexRepo",
		"complexClosure",
		"buildComplexRepo",
		"complexPromotion",
		"loadComplexPromotion",
		"complexNodeGate",
		"readComplex",
		"loadComplexRepo",
		"buildComplexNode",
		"evaluateComplexNode",
		"classGate",
		"evaluateClassGate",
		"evaluateMutationClassGate",
		"evaluateTestOnly",
		"multiRepoLiveRehearsalDecision",
		"evaluateLowRisk",
		"lowRisk",
		"validateLowRisk",
		"evaluateMultiRepo",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return name == "digestAtlasRunLink" || name == "findWorkgraphNode"
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func equalOwnership(left, right map[string][]string) bool {
	if len(left) != len(right) {
		return false
	}
	for path, declarations := range left {
		if !equalStrings(declarations, right[path]) {
			return false
		}
	}
	return true
}
