package traceconfig

import (
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/workflow"
)

func TestEmbeddedRegistriesUseDefaultAndWorkflowPolicies(t *testing.T) {
	ordinary, ok := Resolve(RegistryProduction, "unconfigured-workflow")
	if !ok || ordinary.Profile != RegistryProduction || ordinary.RequireCLILogDocument {
		t.Fatalf("production default = %#v, ok=%t", ordinary, ok)
	}
	override, ok := Resolve(RegistryProduction, "fanloop-maintainer")
	if !ok || !override.RequireCLILogDocument || override.Fields.CLILogURL == "" {
		t.Fatalf("Workflow override = %#v, ok=%t", override, ok)
	}
	if got := override.Fields.Outputs["technical_design_document_url"]; got != "技术方案" {
		t.Fatalf("configured Output mapping = %q, want 技术方案", got)
	}
	testRegistry, ok := Resolve(RegistryTest, "fanloop-maintainer")
	if !ok || testRegistry.Profile != RegistryTest || testRegistry.RequireCLILogDocument {
		t.Fatalf("test fallback = %#v, ok=%t", testRegistry, ok)
	}
}

func TestEmbeddedWorkflowOverridesReferencePublishedFacts(t *testing.T) {
	for profile, endpoints := range registries {
		for workflowID, registry := range endpoints {
			if workflowID == "" {
				continue
			}
			loaded, err := workflow.Load(workflowID)
			if err != nil {
				t.Fatalf("profile %s references unknown Workflow %q: %v", profile, workflowID, err)
			}
			outputs := map[string]bool{}
			for _, condition := range loaded.Workflow.Conditions {
				outputs[condition.Output.Key] = true
			}
			for outputKey := range registry.Fields.Outputs {
				if !outputs[outputKey] {
					t.Fatalf("profile %s Workflow %s maps unknown Output %q", profile, workflowID, outputKey)
				}
			}
		}
	}
}

func TestRegistryYAMLIsStrictAndRejectsAmbiguousFields(t *testing.T) {
	unknown := strings.Replace(string(registryYAML), "schema_version: 1", "schema_version: 1\nunknown: true", 1)
	if _, err := decodeRegistries([]byte(unknown)); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("unknown field error = %v", err)
	}
	collision := strings.Replace(string(registryYAML), "owner: 负责人", "owner: 需求", 1)
	if _, err := decodeRegistries([]byte(collision)); err == nil || !strings.Contains(err.Error(), "mapped more than once") {
		t.Fatalf("duplicate field error = %v", err)
	}
}
