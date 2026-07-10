package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLoadCanonicalBlueprintAuthorizationAcceptsProducerShape(t *testing.T) {
	path := "examples/pulse-intake/blueprint-authorization.ready.json"
	loaded, err := loadCanonicalBlueprintAuthorization(path)
	if err != nil {
		t.Fatalf("load canonical Blueprint authorization: %v", err)
	}
	if loaded.PulseSource.SchemaVersion != blueprintBuildAuthorizationSchema || loaded.PulseSource.Status != "ready" {
		t.Fatalf("unexpected source: %#v", loaded.PulseSource)
	}
	if loaded.ProjectID != "low-risk-code-rehearsal-public-placeholder" || loaded.PackDigest != "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("unexpected canonical authorization binding: %#v", loaded)
	}
	if want := fileSHA256HexForTest(t, repoPath(path)); loaded.PulseSource.SHA256 != want {
		t.Fatalf("source SHA256=%q, want %q", loaded.PulseSource.SHA256, want)
	}
}

func TestLoadCanonicalBlueprintAuthorizationRejectsInvalidProducerContracts(t *testing.T) {
	base := readObjectFixture(t, "examples/pulse-intake/blueprint-authorization.ready.json")
	cases := []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "missing schema",
			mutate: func(document map[string]any) {
				delete(document, "schema")
			},
			want: "schema is required",
		},
		{
			name: "wrong schema",
			mutate: func(document map[string]any) {
				document["schema"] = "ao.blueprint.other.v0.1"
			},
			want: "schema must be",
		},
		{
			name: "schema version only",
			mutate: func(document map[string]any) {
				delete(document, "schema")
				document["schema_version"] = blueprintBuildAuthorizationSchema
			},
			want: "schema is required",
		},
		{
			name: "blocked status",
			mutate: func(document map[string]any) {
				document["status"] = "blocked"
			},
			want: "status must be ready",
		},
		{
			name: "unapproved",
			mutate: func(document map[string]any) {
				document["approved_by_user"] = false
			},
			want: "approved_by_user=true",
		},
		{
			name: "score below threshold",
			mutate: func(document map[string]any) {
				document["score"] = 99
			},
			want: "score must be 100",
		},
		{
			name: "score above threshold",
			mutate: func(document map[string]any) {
				document["score"] = 101
			},
			want: "score must be 100",
		},
		{
			name: "blocking assumptions",
			mutate: func(document map[string]any) {
				document["blocking_assumptions"] = []any{"still blocked"}
			},
			want: "blocking_assumptions must be empty",
		},
		{
			name: "missing project id",
			mutate: func(document map[string]any) {
				delete(document, "project_id")
			},
			want: "project_id is required",
		},
		{
			name: "missing pack digest",
			mutate: func(document map[string]any) {
				delete(document, "blueprint_pack_digest")
			},
			want: "blueprint_pack_digest must start with sha256:",
		},
		{
			name: "missing requirements digest",
			mutate: func(document map[string]any) {
				delete(document, "requirements_digest")
			},
			want: "requirements_digest must start with sha256:",
		},
		{
			name: "missing traceability digest",
			mutate: func(document map[string]any) {
				delete(document, "traceability_digest")
			},
			want: "traceability_digest must start with sha256:",
		},
		{
			name: "missing sdd digest",
			mutate: func(document map[string]any) {
				delete(document, "sdd_plan_digest")
			},
			want: "sdd_plan_digest must start with sha256:",
		},
		{
			name: "wrong next action",
			mutate: func(document map[string]any) {
				document["next_allowed_action"] = "ao-foundry"
			},
			want: "next_allowed_action must be ao-atlas",
		},
		{
			name: "malformed digest",
			mutate: func(document map[string]any) {
				document["requirements_digest"] = "sha256:not-a-digest"
			},
			want: "sha256 must be 64 lowercase hex characters",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			document := cloneJSONMapForTest(t, base)
			test.mutate(document)
			data, err := json.Marshal(document)
			if err != nil {
				t.Fatalf("marshal fixture: %v", err)
			}
			if _, err := parseCanonicalBlueprintAuthorization(data, "internal/cli/testdata/generated-blueprint-authorization.json"); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("load error=%v, want %q", err, test.want)
			}
		})
	}
}

func TestValidateAtlasBlueprintImportRejectsAuthorityClaims(t *testing.T) {
	base := loadAtlasBlueprintImportForTest(t, "examples/atlas/blueprint-import.low-risk-code.json")
	for _, field := range []string{
		"SchedulesWork",
		"ExecutesWork",
		"ApprovesWork",
		"MutatesRepositories",
		"CallsProviders",
		"ReleaseOrPublishAllowed",
	} {
		t.Run(field, func(t *testing.T) {
			artifact := base
			switch field {
			case "SchedulesWork":
				artifact.SchedulesWork = true
			case "ExecutesWork":
				artifact.ExecutesWork = true
			case "ApprovesWork":
				artifact.ApprovesWork = true
			case "MutatesRepositories":
				artifact.MutatesRepositories = true
			case "CallsProviders":
				artifact.CallsProviders = true
			case "ReleaseOrPublishAllowed":
				artifact.ReleaseOrPublishAllowed = true
			}
			if err := validateAtlasBlueprintImport(artifact); err == nil {
				t.Fatalf("validateAtlasBlueprintImport accepted %s=true", field)
			}
		})
	}
}

func TestLoadCanonicalBlueprintAuthorizationRejectsUnsafePaths(t *testing.T) {
	for _, path := range []string{"../blueprint-authorization.json", "/tmp/blueprint-authorization.json"} {
		t.Run(path, func(t *testing.T) {
			if _, err := loadCanonicalBlueprintAuthorization(path); err == nil || !strings.Contains(err.Error(), "unsafe source artifact path") {
				t.Fatalf("load error=%v, want unsafe path rejection", err)
			}
		})
	}
}

func TestValidateAtlasBlueprintImportForFoundryBindsCanonicalAuthorization(t *testing.T) {
	blueprintImport := loadAtlasBlueprintImportForTest(t, "examples/atlas/blueprint-import.low-risk-code.json")
	foundryImport := loadAtlasFoundryImportForTest(t, "examples/atlas/foundry-import.json")
	authorization, err := loadCanonicalBlueprintAuthorization("examples/pulse-intake/blueprint-authorization.ready.json")
	if err != nil {
		t.Fatalf("load authorization: %v", err)
	}
	importSource, err := pulseIntakeSourceFromFile("atlas_import", "examples/atlas/foundry-import.json", foundryImport.ContractVersion, foundryImport.Status)
	if err != nil {
		t.Fatalf("load import source: %v", err)
	}
	if err := validateAtlasBlueprintImportForFoundry(blueprintImport, foundryImport, authorization, importSource); err != nil {
		t.Fatalf("validate matching artifacts: %v", err)
	}
	cases := []struct {
		name   string
		mutate func(*AtlasBlueprintImport, *canonicalBlueprintAuthorizationSource)
		want   string
	}{
		{
			name: "authorization digest mismatch",
			mutate: func(artifact *AtlasBlueprintImport, _ *canonicalBlueprintAuthorizationSource) {
				artifact.BuildAuthorization.Digest = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
			},
			want: "build authorization digest",
		},
		{
			name: "project mismatch",
			mutate: func(_ *AtlasBlueprintImport, source *canonicalBlueprintAuthorizationSource) {
				source.ProjectID = "other-project"
			},
			want: "project_id must match",
		},
		{
			name: "pack mismatch",
			mutate: func(_ *AtlasBlueprintImport, source *canonicalBlueprintAuthorizationSource) {
				source.PackDigest = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
			},
			want: "Blueprint pack digest must match",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			artifact := blueprintImport
			source := authorization
			test.mutate(&artifact, &source)
			if err := validateAtlasBlueprintImportForFoundry(artifact, foundryImport, source, importSource); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("validation error=%v, want %q", err, test.want)
			}
		})
	}
}

func TestLoadAtlasBlueprintImportAcceptsCandidateSelection(t *testing.T) {
	artifact := loadAtlasBlueprintImportForTest(t, "examples/atlas/blueprint-import.low-risk-code.json")
	if artifact.CandidateSelection == nil {
		t.Fatal("expected candidate_selection to be decoded")
	}
	if artifact.CandidateSelection.ContractVersion != "ao.atlas.blueprint-candidate-selection.v0.1" || artifact.CandidateSelection.WorkgraphID != artifact.WorkgraphID {
		t.Fatalf("unexpected candidate_selection: %#v", artifact.CandidateSelection)
	}
	if artifact.DownstreamFoundryContinuationHandoff.Digest != artifact.Digests["downstream_foundry_continuation_handoff"] {
		t.Fatalf("unexpected continuation handoff binding: %#v", artifact.DownstreamFoundryContinuationHandoff)
	}
}

func cloneJSONMapForTest(t *testing.T, value map[string]any) map[string]any {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	var clone map[string]any
	if err := json.Unmarshal(data, &clone); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	return clone
}

func loadAtlasBlueprintImportForTest(t *testing.T, path string) AtlasBlueprintImport {
	t.Helper()
	artifact, err := loadAtlasBlueprintImport(path)
	if err != nil {
		t.Fatalf("load Atlas Blueprint import: %v", err)
	}
	return artifact
}

func loadAtlasFoundryImportForTest(t *testing.T, path string) AtlasFoundryImport {
	t.Helper()
	artifact, err := loadAtlasFoundryImport(path)
	if err != nil {
		t.Fatalf("load Atlas Foundry import: %v", err)
	}
	return artifact
}
