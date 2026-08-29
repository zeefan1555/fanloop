package state

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/zeefan1555/commonloop/internal/idl/storageidl"
	"github.com/zeefan1555/commonloop/internal/traceconfig"
	"github.com/zeefan1555/commonloop/internal/workflow"
)

func TestEveryEventPayloadRoundTripsThroughStorageThriftUnion(t *testing.T) {
	now := time.Date(2026, 8, 18, 1, 2, 3, 4, time.UTC)
	workflowRef := WorkflowRef{ID: "commonloop", Digest: "sha256:test"}
	result := FlowResultPayload{
		ConditionResults: []ConditionResult{{
			ConditionID: "repository_workspace_prepared",
			Output:      OutputValue{Type: workflow.OutputPath, Value: json.RawMessage(`"scope.md"`)},
		}},
		Summary: "accepted", Effect: ResultAdvanced,
		Transition:    Transition{Direction: TransitionFlow, FromStepID: "bootstrap_techdesign", ToStepID: "clarify_requirements"},
		OutputChanges: OutputChanges{Accepted: []string{"repository_scope_path"}},
	}
	tests := []Event{
		{SchemaVersion: CurrentEventSchemaVersion, ID: "e1", OccurredAt: now, Kind: EventFlowInitialized, Command: "flow.init", Workflow: workflowRef, Payload: Payload(FlowInitializedPayload{StepID: "bootstrap_techdesign", StepStatus: StepReady})},
		{SchemaVersion: CurrentEventSchemaVersion, ID: "e2", OccurredAt: now, Kind: EventFlowProgressed, Command: "flow.report.progress", Workflow: workflowRef, CausedByEventID: "e1", Payload: Payload(FlowProgressPayload{FromStepID: "bootstrap_techdesign", FromStepStatus: StepReady, ToStepStatus: StepInProgress, Summary: "working", Evidence: []Evidence{{Source: EvidenceSystem, Content: "git status"}}})},
		{SchemaVersion: CurrentEventSchemaVersion, ID: "e3", OccurredAt: now, Kind: EventFlowResult, Command: "flow.report.result", Workflow: workflowRef, CausedByEventID: "e2", Payload: Payload(result)},
		{SchemaVersion: CurrentEventSchemaVersion, ID: "e4", OccurredAt: now, Kind: EventTraceDocumentBound, Command: "trace.bind", Workflow: workflowRef, CausedByEventID: "e3", Payload: Payload(TraceDocumentBoundPayload{DocumentURL: "https://bytedance.larkoffice.com/docx/Trace", Registry: traceconfig.RegistryProduction})},
		{SchemaVersion: CurrentEventSchemaVersion, ID: "e5", OccurredAt: now, Kind: EventTraceSyncStarted, Command: "trace.sync", Workflow: workflowRef, CausedByEventID: "e4", Payload: Payload(TraceSyncStartedPayload{Targets: []string{"trace_document", "registry"}})},
		{SchemaVersion: CurrentEventSchemaVersion, ID: "e6", OccurredAt: now, Kind: EventTraceSynced, Command: "trace.sync", Workflow: workflowRef, CausedByEventID: "e5", Payload: Payload(TraceSyncedPayload{Outcome: TraceSyncPartial, Targets: []TraceTargetResult{
			{Name: "trace_document", Status: "succeeded"},
			{Name: "registry", Status: "failed", Error: &TraceTargetError{Code: "REGISTRY_UPDATE_FAILED", Message: "unavailable", Retryable: true}},
		}})},
	}
	for _, event := range tests {
		t.Run(string(event.Kind), func(t *testing.T) {
			content, err := EncodeEvent(event)
			if err != nil {
				t.Fatal(err)
			}
			var stored storageidl.Event
			if err := json.Unmarshal(content, &stored); err != nil {
				t.Fatal(err)
			}
			if stored.Payload == nil || stored.Payload.CountSetFieldsEventPayload() != 1 {
				t.Fatalf("payload = %#v", stored.Payload)
			}
			decoded, err := DecodeEvent(content)
			if err != nil {
				t.Fatal(err)
			}
			if decoded.SchemaVersion != event.SchemaVersion || decoded.ID != event.ID || decoded.OccurredAt != event.OccurredAt ||
				decoded.Kind != event.Kind || decoded.Command != event.Command || decoded.Workflow != event.Workflow ||
				decoded.CausedByEventID != event.CausedByEventID || !reflect.DeepEqual(decoded.Payload, event.Payload) {
				t.Fatalf("decoded = %#v, want %#v", decoded, event)
			}
		})
	}
}

func TestMaintainerTraceBindingRoundTripsCLILogDocument(t *testing.T) {
	now := time.Date(2026, 8, 25, 1, 2, 3, 4, time.UTC)
	step := "bootstrap_techdesign"
	value := State{
		SchemaVersion: CurrentStateSchemaVersion,
		Requirement:   Requirement{Title: "Maintainer"},
		Release: Release{Version: "dev", Workflow: WorkflowRef{
			ID: "commonloop-maintainer", Digest: "sha256:test",
		}},
		CurrentStepID: &step, CurrentStepStatus: StepReady,
		Outputs: map[string]RegisteredOutput{},
		Integrations: Integrations{Trace: &TraceBinding{
			DocumentURL: "https://bytedance.larkoffice.com/docx/Trace", Registry: traceconfig.RegistryProduction,
			CLILogDocumentURL: "https://bytedance.larkoffice.com/docx/CLILog",
		}},
		LastEventID: "e1", CreatedAt: now, UpdatedAt: now,
	}
	content, err := Encode(value)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(content, map[string]RegisteredOutput{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded.Integrations.Trace, value.Integrations.Trace) {
		t.Fatalf("Trace binding = %#v, want %#v", decoded.Integrations.Trace, value.Integrations.Trace)
	}

	event := Event{
		SchemaVersion: CurrentEventSchemaVersion, ID: "e2", OccurredAt: now, Kind: EventTraceDocumentBound,
		Command: "trace.bind", Workflow: value.Release.Workflow, CausedByEventID: "e1",
		Payload: Payload(TraceDocumentBoundPayload{
			DocumentURL: value.Integrations.Trace.DocumentURL, Registry: value.Integrations.Trace.Registry,
			CLILogDocumentURL: value.Integrations.Trace.CLILogDocumentURL,
		}),
	}
	encodedEvent, err := EncodeEvent(event)
	if err != nil {
		t.Fatal(err)
	}
	decodedEvent, err := DecodeEvent(encodedEvent)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decodedEvent.Payload, event.Payload) {
		t.Fatalf("Trace Event payload = %#v, want %#v", decodedEvent.Payload, event.Payload)
	}
}

func TestStorageEventRejectsKindPayloadMismatch(t *testing.T) {
	event := Event{
		SchemaVersion: CurrentEventSchemaVersion, ID: "e1", OccurredAt: time.Now().UTC(),
		Kind: EventFlowInitialized, Command: "flow.init", Workflow: WorkflowRef{ID: "commonloop", Digest: "sha256:test"},
		Payload: Payload(FlowInitializedPayload{StepID: "bootstrap_techdesign", StepStatus: StepReady}),
	}
	content, err := EncodeEvent(event)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(content, &document); err != nil {
		t.Fatal(err)
	}
	document["kind"] = "flow_progressed"
	tampered, _ := json.Marshal(document)
	if _, err := DecodeEvent(tampered); err == nil {
		t.Fatal("expected kind/payload mismatch to be rejected")
	}
}

func TestRuntimeDomainModelsDoNotDefinePersistenceJSON(t *testing.T) {
	values := []any{
		State{}, Requirement{}, WorkflowRef{}, Release{}, Integrations{}, TraceBinding{}, Evidence{},
		OutputValue{}, RegisteredOutput{}, ConditionResult{}, Transition{}, OutputChanges{}, Event{},
		FlowInitializedPayload{}, FlowProgressPayload{}, FlowResultPayload{}, TraceDocumentBoundPayload{},
		TraceSyncStartedPayload{}, TraceSyncedPayload{}, TraceTargetResult{}, TraceTargetError{}, OutputRegistry{},
	}
	for _, value := range values {
		typeOf := reflect.TypeOf(value)
		for index := 0; index < typeOf.NumField(); index++ {
			if tag := typeOf.Field(index).Tag.Get("json"); tag != "" {
				t.Fatalf("%s.%s retains persistence JSON tag %q", typeOf.Name(), typeOf.Field(index).Name, tag)
			}
		}
	}
}
