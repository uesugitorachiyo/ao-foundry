package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

type pulseArtifactSource struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	SchemaVersion string `json:"schema_version"`
	Status        string `json:"status"`
	SHA256        string `json:"sha256"`
}

func loadPulseJSONSource(name, path string) (map[string]any, pulseArtifactSource, error) {
	if strings.TrimSpace(path) == "" {
		return nil, pulseArtifactSource{}, errors.New("missing source path")
	}
	document, err := readArbitraryJSON(path)
	if err != nil {
		return nil, pulseArtifactSource{}, fmt.Errorf("read %s: %w", name, err)
	}
	object, ok := document.(map[string]any)
	if !ok {
		return nil, pulseArtifactSource{}, fmt.Errorf("%s must be a JSON object", name)
	}
	if err := validatePublicSafeJSONStrings(object); err != nil {
		return nil, pulseArtifactSource{}, err
	}
	schema := classGateFirstNonEmpty(classGateString(object, "schema_version"), classGateString(object, "contract_version"), classGateString(object, "schema"))
	status := classGateFirstNonEmpty(classGateString(object, "status"), classGateString(object, "completion_status"), classGateString(object, "verdict"))
	source, err := pulseArtifactSourceFromFile(name, path, schema, status)
	if err != nil {
		return nil, pulseArtifactSource{}, err
	}
	return object, source, nil
}

func pulseArtifactSourceFromFile(name, path, schemaVersion, status string) (pulseArtifactSource, error) {
	sum, err := fileSHA256(path)
	if err != nil {
		return pulseArtifactSource{}, err
	}
	return pulseArtifactSource{
		Name:          name,
		Path:          filepath.ToSlash(filepath.Clean(publicArtifactSource(path))),
		SchemaVersion: schemaVersion,
		Status:        status,
		SHA256:        sum,
	}, nil
}
