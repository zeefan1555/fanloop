package traceconfig

import (
	"strings"
	"testing"
)

func TestResolveUsesConfiguredOverrideAndProfileDefault(t *testing.T) {
	configured, err := decodeRegistries([]byte(syntheticRegistryYAML))
	if err != nil {
		t.Fatal(err)
	}
	previous := registries
	registries = configured
	t.Cleanup(func() { registries = previous })

	ordinary, ok := Resolve(RegistryProduction, "unconfigured-workflow")
	if !ok || ordinary.Profile != RegistryProduction || ordinary.RequireCLILogDocument {
		t.Fatalf("production default = %#v, ok=%t", ordinary, ok)
	}
	override, ok := Resolve(RegistryProduction, "custom-flow")
	if !ok || !override.RequireCLILogDocument || override.Fields.CLILogURL != "CLI log" {
		t.Fatalf("Workflow override = %#v, ok=%t", override, ok)
	}
	if got := override.Fields.Outputs["artifact_links"]; got != "Artifact" {
		t.Fatalf("configured Output mapping = %q, want Artifact", got)
	}
	testRegistry, ok := Resolve(RegistryTest, "custom-flow")
	if !ok || testRegistry.Profile != RegistryTest || testRegistry.RequireCLILogDocument {
		t.Fatalf("test fallback = %#v, ok=%t", testRegistry, ok)
	}
}

func TestRegistryYAMLIsStrictAndRejectsAmbiguousFields(t *testing.T) {
	unknown := strings.Replace(syntheticRegistryYAML, "schema_version: 1", "schema_version: 1\nunknown: true", 1)
	if _, err := decodeRegistries([]byte(unknown)); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("unknown field error = %v", err)
	}
	collision := strings.Replace(syntheticRegistryYAML, "owner: Owner", "owner: Title", 1)
	if _, err := decodeRegistries([]byte(collision)); err == nil || !strings.Contains(err.Error(), "mapped more than once") {
		t.Fatalf("duplicate field error = %v", err)
	}
}

func TestProviderNoneDisablesWorkflowWithoutLarkTargets(t *testing.T) {
	disabled := strings.Replace(syntheticRegistryYAML, `    workflows:
`, `    workflows:
      quiet-flow:
        provider: none
`, 1)
	configured, err := decodeRegistries([]byte(disabled))
	if err != nil {
		t.Fatal(err)
	}
	previous := registries
	registries = configured
	t.Cleanup(func() { registries = previous })

	quiet, ok := Resolve(RegistryProduction, "quiet-flow")
	if !ok || !quiet.Disabled() || quiet.RequireCLILogDocument {
		t.Fatalf("provider none = %#v, ok=%t", quiet, ok)
	}
	if ordinary, ok := Resolve(RegistryProduction, "unconfigured-workflow"); !ok || ordinary.Disabled() {
		t.Fatalf("default provider = %#v, ok=%t", ordinary, ok)
	}
}

func TestProviderIsRequiredAndNoneRejectsLarkTargets(t *testing.T) {
	missing := strings.Replace(syntheticRegistryYAML, "      provider: lark\n", "", 1)
	if _, err := decodeRegistries([]byte(missing)); err == nil || !strings.Contains(err.Error(), "provider must be") {
		t.Fatalf("missing provider error = %v", err)
	}
	unknown := strings.Replace(syntheticRegistryYAML, "      provider: lark\n", "      provider: notion\n", 1)
	if _, err := decodeRegistries([]byte(unknown)); err == nil || !strings.Contains(err.Error(), "provider must be") {
		t.Fatalf("unknown provider error = %v", err)
	}
	mixed := strings.Replace(syntheticRegistryYAML, "      provider: lark\n", "      provider: none\n", 1)
	if _, err := decodeRegistries([]byte(mixed)); err == nil || !strings.Contains(err.Error(), "must not configure Lark targets") {
		t.Fatalf("provider none with Lark targets error = %v", err)
	}
}

const syntheticRegistryYAML = `schema_version: 1
profiles:
  production:
    default:
      provider: lark
      url: https://example.com/default
      base_token: default-base
      table_id: default-table
      view_id: default-view
      require_cli_log_document: false
      fields: &default_fields
        trace_key: Trace key
        title: Title
        owner: Owner
        location: Location
        status: Status
        loop_count: Loop count
        source_url: Source
        trace_url: Trace
        updated_at: Updated at
        origin: Origin
    workflows:
      custom-flow:
        provider: lark
        url: https://example.com/custom
        base_token: custom-base
        table_id: custom-table
        view_id: custom-view
        require_cli_log_document: true
        fields:
          trace_key: Trace key
          title: Title
          owner: Owner
          location: Location
          status: Status
          loop_count: Loop count
          source_url: Source
          trace_url: Trace
          updated_at: Updated at
          origin: Origin
          cli_log_url: CLI log
          outputs:
            artifact_links: Artifact
  test:
    default:
      provider: lark
      url: https://example.com/test
      base_token: test-base
      table_id: test-table
      view_id: test-view
      require_cli_log_document: false
      fields: *default_fields
`

func TestEmbeddedRegistryDisablesLarkProjection(t *testing.T) {
	for _, workflowID := range []string{"", "fanloop-maintainer", "code-change", "unconfigured-workflow"} {
		registry, ok := Resolve(RegistryProduction, workflowID)
		if !ok || !registry.Disabled() || registry.BaseToken != "" || registry.RequireCLILogDocument {
			t.Fatalf("embedded production %q = %#v, ok=%t; want provider none", workflowID, registry, ok)
		}
	}
	if registry, ok := Resolve(RegistryTest, "code-change"); !ok || registry.Disabled() {
		t.Fatalf("embedded test profile = %#v, ok=%t; want explicit Lark test Registry", registry, ok)
	}
}
