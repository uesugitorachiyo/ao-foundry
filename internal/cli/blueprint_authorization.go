package cli

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const blueprintBuildAuthorizationSchema = "ao.blueprint.build-authorization.v0.1"

type canonicalBlueprintAuthorization struct {
	Schema              string   `json:"schema"`
	ProjectID           string   `json:"project_id"`
	Status              string   `json:"status"`
	Score               int      `json:"score"`
	ApprovedByUser      bool     `json:"approved_by_user"`
	BlockingAssumptions []string `json:"blocking_assumptions"`
	BlueprintPackDigest string   `json:"blueprint_pack_digest"`
	RequirementsDigest  string   `json:"requirements_digest"`
	TraceabilityDigest  string   `json:"traceability_digest"`
	SDDPlanDigest       string   `json:"sdd_plan_digest"`
	NextAllowedAction   string   `json:"next_allowed_action"`
}

type canonicalBlueprintAuthorizationSource struct {
	PulseSource PulseIntakeSource
	ProjectID   string
	PackDigest  string
}

func loadCanonicalBlueprintAuthorization(path string) (canonicalBlueprintAuthorizationSource, error) {
	if err := validateEvidencePath(path); err != nil {
		return canonicalBlueprintAuthorizationSource{}, fmt.Errorf("unsafe source artifact path: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		data, err = readRepoRelativeFile(path)
		if err != nil {
			return canonicalBlueprintAuthorizationSource{}, fmt.Errorf("read canonical Blueprint authorization: %w", err)
		}
	}
	return parseCanonicalBlueprintAuthorization(data, path)
}

func parseCanonicalBlueprintAuthorization(data []byte, path string) (canonicalBlueprintAuthorizationSource, error) {
	if err := validateEvidencePath(path); err != nil {
		return canonicalBlueprintAuthorizationSource{}, fmt.Errorf("unsafe source artifact path: %w", err)
	}
	var document any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&document); err != nil {
		return canonicalBlueprintAuthorizationSource{}, fmt.Errorf("invalid canonical Blueprint authorization JSON: %w", err)
	}
	object, ok := document.(map[string]any)
	if !ok {
		return canonicalBlueprintAuthorizationSource{}, errors.New("canonical Blueprint authorization must be a JSON object")
	}
	if err := validatePublicSafeJSONStrings(object); err != nil {
		return canonicalBlueprintAuthorizationSource{}, err
	}
	var authorization canonicalBlueprintAuthorization
	if err := json.Unmarshal(data, &authorization); err != nil {
		return canonicalBlueprintAuthorizationSource{}, fmt.Errorf("decode canonical Blueprint authorization: %w", err)
	}
	if authorization.Schema == "" {
		return canonicalBlueprintAuthorizationSource{}, errors.New("canonical Blueprint authorization schema is required")
	}
	if authorization.Schema != blueprintBuildAuthorizationSchema {
		return canonicalBlueprintAuthorizationSource{}, fmt.Errorf("canonical Blueprint authorization schema must be %s", blueprintBuildAuthorizationSchema)
	}
	if strings.TrimSpace(authorization.ProjectID) == "" {
		return canonicalBlueprintAuthorizationSource{}, errors.New("canonical Blueprint authorization project_id is required")
	}
	if err := validateAtlasPublicString(authorization.ProjectID); err != nil {
		return canonicalBlueprintAuthorizationSource{}, fmt.Errorf("canonical Blueprint authorization project_id: %w", err)
	}
	if authorization.Status != "ready" {
		return canonicalBlueprintAuthorizationSource{}, errors.New("canonical Blueprint authorization status must be ready")
	}
	if authorization.Score != 100 {
		return canonicalBlueprintAuthorizationSource{}, errors.New("canonical Blueprint authorization score must be 100")
	}
	if !authorization.ApprovedByUser {
		return canonicalBlueprintAuthorizationSource{}, errors.New("canonical Blueprint authorization must be approved_by_user=true")
	}
	if len(authorization.BlockingAssumptions) != 0 {
		return canonicalBlueprintAuthorizationSource{}, errors.New("canonical Blueprint authorization blocking_assumptions must be empty")
	}
	if authorization.NextAllowedAction != "ao-atlas" {
		return canonicalBlueprintAuthorizationSource{}, errors.New("canonical Blueprint authorization next_allowed_action must be ao-atlas")
	}
	for _, digest := range []struct {
		name  string
		value string
	}{
		{name: "blueprint_pack_digest", value: authorization.BlueprintPackDigest},
		{name: "requirements_digest", value: authorization.RequirementsDigest},
		{name: "traceability_digest", value: authorization.TraceabilityDigest},
		{name: "sdd_plan_digest", value: authorization.SDDPlanDigest},
	} {
		if !strings.HasPrefix(digest.value, "sha256:") {
			return canonicalBlueprintAuthorizationSource{}, fmt.Errorf("canonical Blueprint authorization %s must start with sha256:", digest.name)
		}
		if err := validateSHA256(strings.TrimPrefix(digest.value, "sha256:"), "canonical Blueprint authorization "+digest.name); err != nil {
			return canonicalBlueprintAuthorizationSource{}, err
		}
	}
	sum := sha256.Sum256(data)
	return canonicalBlueprintAuthorizationSource{
		PulseSource: PulseIntakeSource{
			Name:          "blueprint_authorization",
			Path:          filepath.ToSlash(filepath.Clean(path)),
			SchemaVersion: authorization.Schema,
			Status:        authorization.Status,
			SHA256:        fmt.Sprintf("%x", sum[:]),
		},
		ProjectID:  authorization.ProjectID,
		PackDigest: authorization.BlueprintPackDigest,
	}, nil
}
