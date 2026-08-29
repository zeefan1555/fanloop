package traceconfig

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"
)

type RegistryProfile string

const (
	RegistryProduction RegistryProfile = "production"
	RegistryTest       RegistryProfile = "test"
)

type RegistryFields struct {
	TraceKey           string            `yaml:"trace_key"`
	Title              string            `yaml:"title"`
	Owner              string            `yaml:"owner"`
	Location           string            `yaml:"location"`
	Status             string            `yaml:"status"`
	LoopCount          string            `yaml:"loop_count"`
	SourceURL          string            `yaml:"source_url"`
	SourceOutput       string            `yaml:"source_output,omitempty"`
	SourceDocumentOnly bool              `yaml:"source_document_only,omitempty"`
	TraceURL           string            `yaml:"trace_url"`
	UpdatedAt          string            `yaml:"updated_at"`
	Origin             string            `yaml:"origin"`
	CLILogURL          string            `yaml:"cli_log_url,omitempty"`
	Outputs            map[string]string `yaml:"outputs,omitempty"`
}

type Registry struct {
	Profile               RegistryProfile `yaml:"-"`
	URL                   string          `yaml:"url"`
	BaseToken             string          `yaml:"base_token"`
	TableID               string          `yaml:"table_id"`
	ViewID                string          `yaml:"view_id"`
	RequireCLILogDocument bool            `yaml:"require_cli_log_document"`
	Fields                RegistryFields  `yaml:"fields"`
}

type profileDocument struct {
	Default   Registry            `yaml:"default"`
	Workflows map[string]Registry `yaml:"workflows,omitempty"`
}

type registryDocument struct {
	SchemaVersion int                                 `yaml:"schema_version"`
	Profiles      map[RegistryProfile]profileDocument `yaml:"profiles"`
}

var configIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

//go:embed registry.yaml
var registryYAML []byte

var registries = mustDecodeRegistries(registryYAML)

func Resolve(profile RegistryProfile, workflowID string) (Registry, bool) {
	if profile == "" {
		profile = RegistryProduction
	}
	endpoints, ok := registries[profile]
	if !ok {
		return Registry{}, false
	}
	if registry, ok := endpoints[workflowID]; ok {
		return registry, true
	}
	registry, ok := endpoints[""]
	return registry, ok
}

func Valid(profile RegistryProfile) bool {
	_, ok := registries[profile]
	return ok
}

func mustDecodeRegistries(content []byte) map[RegistryProfile]map[string]Registry {
	result, err := decodeRegistries(content)
	if err != nil {
		panic("invalid embedded Trace Registry configuration: " + err.Error())
	}
	return result
}

func decodeRegistries(content []byte) (map[RegistryProfile]map[string]Registry, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	var document registryDocument
	if err := decoder.Decode(&document); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("trailing YAML document")
		}
		return nil, err
	}
	if document.SchemaVersion != 1 {
		return nil, fmt.Errorf("unsupported schema_version %d", document.SchemaVersion)
	}
	if len(document.Profiles) != 2 {
		return nil, fmt.Errorf("production and test profiles are required")
	}
	result := make(map[RegistryProfile]map[string]Registry, len(document.Profiles))
	for _, profile := range []RegistryProfile{RegistryProduction, RegistryTest} {
		configured, ok := document.Profiles[profile]
		if !ok {
			return nil, fmt.Errorf("profile %q is required", profile)
		}
		endpoints := make(map[string]Registry, len(configured.Workflows)+1)
		configured.Default.Profile = profile
		if err := validateRegistry(configured.Default); err != nil {
			return nil, fmt.Errorf("profile %q default: %w", profile, err)
		}
		endpoints[""] = configured.Default
		for workflowID, registry := range configured.Workflows {
			if !configIDPattern.MatchString(workflowID) {
				return nil, fmt.Errorf("profile %q has invalid Workflow ID %q", profile, workflowID)
			}
			registry.Profile = profile
			if err := validateRegistry(registry); err != nil {
				return nil, fmt.Errorf("profile %q Workflow %q: %w", profile, workflowID, err)
			}
			endpoints[workflowID] = registry
		}
		result[profile] = endpoints
	}
	return result, nil
}

func validateRegistry(registry Registry) error {
	for name, value := range map[string]string{
		"url": registry.URL, "base_token": registry.BaseToken, "table_id": registry.TableID, "view_id": registry.ViewID,
		"fields.trace_key": registry.Fields.TraceKey, "fields.title": registry.Fields.Title, "fields.owner": registry.Fields.Owner,
		"fields.location": registry.Fields.Location, "fields.status": registry.Fields.Status, "fields.loop_count": registry.Fields.LoopCount,
		"fields.source_url": registry.Fields.SourceURL, "fields.trace_url": registry.Fields.TraceURL,
		"fields.updated_at": registry.Fields.UpdatedAt, "fields.origin": registry.Fields.Origin,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	if registry.RequireCLILogDocument != (strings.TrimSpace(registry.Fields.CLILogURL) != "") {
		return fmt.Errorf("fields.cli_log_url must be present exactly when require_cli_log_document is true")
	}
	if registry.Fields.SourceOutput != "" && !configIDPattern.MatchString(registry.Fields.SourceOutput) {
		return fmt.Errorf("fields.source_output is invalid")
	}
	fieldNames := map[string]bool{}
	for _, value := range []string{
		registry.Fields.TraceKey, registry.Fields.Title, registry.Fields.Owner, registry.Fields.Location,
		registry.Fields.Status, registry.Fields.LoopCount, registry.Fields.SourceURL, registry.Fields.TraceURL,
		registry.Fields.UpdatedAt, registry.Fields.Origin, registry.Fields.CLILogURL,
	} {
		if value == "" {
			continue
		}
		if fieldNames[value] {
			return fmt.Errorf("Registry field %q is mapped more than once", value)
		}
		fieldNames[value] = true
	}
	for outputKey, fieldName := range registry.Fields.Outputs {
		if !configIDPattern.MatchString(outputKey) || strings.TrimSpace(fieldName) == "" {
			return fmt.Errorf("invalid Registry Output field mapping %q", outputKey)
		}
		if fieldNames[fieldName] {
			return fmt.Errorf("Registry field %q is mapped more than once", fieldName)
		}
		fieldNames[fieldName] = true
	}
	return nil
}
