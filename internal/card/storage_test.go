package card

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/traceconfig"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

func TestProjectionWritesStorageThriftSchema(t *testing.T) {
	loaded := loadInstalledLinearRuntime(t)
	step, _ := loaded.Workflow.FirstStepID()
	now := time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC)
	current := state.State{
		SchemaVersion: state.CurrentStateSchemaVersion,
		Requirement:   state.Requirement{Title: "Storage Thrift"},
		Release:       state.Release{Version: "dev", Workflow: state.WorkflowRefFrom(loaded.Ref)},
		CurrentStepID: &step, CurrentStepStatus: state.StepReady,
		Outputs: map[string]state.RegisteredOutput{}, Integrations: state.Integrations{},
		LastEventID: "e1", CreatedAt: now, UpdatedAt: now,
	}
	root := t.TempDir()
	if err := WriteProjection(root, current); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(ProjectionPath(root))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(content, &document); err != nil {
		t.Fatal(err)
	}
	if document["schema_version"] != float64(7) {
		t.Fatalf("schema_version = %v, want 7", document["schema_version"])
	}
	if _, _, err := LoadProjection(root); err != nil {
		t.Fatal(err)
	}
	document["cli_log_document_url"] = "https://example.invalid/redacted"
	content, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ProjectionPath(root), content, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadProjection(root); err == nil {
		t.Fatal("Card projection accepted a CLI log document without a Trace document")
	}
}

func TestTestRegistryProjectionKeepsSingleDocumentBinding(t *testing.T) {
	loaded := loadInstalledLinearRuntime(t)
	step, _ := loaded.Workflow.FirstStepID()
	now := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	current := state.State{
		SchemaVersion: state.CurrentStateSchemaVersion,
		Requirement:   state.Requirement{Title: "Test registry"},
		Release:       state.Release{Version: "dev", Workflow: state.WorkflowRefFrom(loaded.Ref)},
		CurrentStepID: &step, CurrentStepStatus: state.StepReady,
		Outputs: map[string]state.RegisteredOutput{},
		Integrations: state.Integrations{Trace: &state.TraceBinding{
			DocumentURL: "https://example.invalid/redacted", Registry: traceconfig.RegistryTest,
		}},
		LastEventID: "e1", CreatedAt: now, UpdatedAt: now,
	}
	root := t.TempDir()
	if err := WriteProjection(root, current); err != nil {
		t.Fatal(err)
	}
	projection, _, err := LoadProjection(root)
	if err != nil {
		t.Fatal(err)
	}
	if projection.CLILogDocumentURL != "" || projection.TraceDocumentURL != current.Integrations.Trace.DocumentURL {
		t.Fatalf("projection = %#v", projection)
	}
}

func loadInstalledLinearRuntime(t *testing.T) workflow.Loaded {
	t.Helper()
	dataRoot := t.TempDir()
	bundleRoot := filepath.Join(dataRoot, "source", "skills", "linear-runtime")
	if err := os.CopyFS(bundleRoot, os.DirFS(filepath.Join("..", "workflow", "testdata", "linear-runtime"))); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_CONFIG_SOURCE", filepath.Join(dataRoot, "source"))
	loaded, err := workflow.Load("linear-runtime")
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func TestBindingWritesStorageThriftSchema(t *testing.T) {
	t.Setenv("BOTMUX_CHAT_ID", "oc_storage")
	t.Setenv("BOTMUX_SESSION_ID", "session-storage")
	root := t.TempDir()
	if _, found, err := LoadOrCaptureBinding(root); err != nil || !found {
		t.Fatalf("found = %v, error = %v", found, err)
	}
	content, err := os.ReadFile(BindingPath(root))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(content, &document); err != nil {
		t.Fatal(err)
	}
	if document["schema_version"] != float64(2) {
		t.Fatalf("schema_version = %v, want 2", document["schema_version"])
	}
	if binding, found, err := LoadBinding(root); err != nil || !found || binding.ChatID != "oc_storage" {
		t.Fatalf("binding = %#v, found = %v, error = %v", binding, found, err)
	}
}

func TestBindingRejectsWhitespaceIdentifiers(t *testing.T) {
	root := t.TempDir()
	path := BindingPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"schema_version":2,"chat_id":" ","session_id":"session"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadBinding(root); err == nil {
		t.Fatal("expected whitespace Card binding to be rejected")
	}
}

func TestCardDomainModelsDoNotDefinePersistenceJSON(t *testing.T) {
	for _, value := range []any{Projection{}, BotmuxBinding{}} {
		typeOf := reflect.TypeOf(value)
		for index := 0; index < typeOf.NumField(); index++ {
			if tag := typeOf.Field(index).Tag.Get("json"); tag != "" {
				t.Fatalf("%s.%s retains persistence JSON tag %q", typeOf.Name(), typeOf.Field(index).Name, tag)
			}
		}
	}
}
