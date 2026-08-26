package state

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/zeefan1555/fanloop/internal/workflow"
)

func TestHistoryReplaysProgressFlowResultAndLoopInvalidation(t *testing.T) {
	loaded, err := workflow.Load("promotion-design")
	if err != nil {
		t.Fatal(err)
	}
	first, _ := loaded.Workflow.FirstStepID()
	now := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	current := State{
		SchemaVersion: CurrentStateSchemaVersion,
		Requirement:   Requirement{Title: "State v8"},
		Release:       Release{Version: "dev", Workflow: WorkflowRefFrom(loaded.Ref)},
		CurrentStepID: &first, CurrentStepStatus: StepReady,
		CurrentStepSummary: "workflow initialized",
		Outputs:            map[string]RegisteredOutput{},
		Integrations:       Integrations{},
		LastEventID:        "e1",
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	events := []Event{{
		SchemaVersion: CurrentEventSchemaVersion, ID: "e1", OccurredAt: now,
		Kind: EventFlowInitialized, Command: "flow.init", Workflow: current.Release.Workflow,
		Payload: Payload(FlowInitializedPayload{StepID: first, StepStatus: StepReady}),
	}}

	progress := FlowProgressPayload{
		FromStepID: first, FromStepStatus: StepReady, ToStepStatus: StepInProgress,
		Summary: "checking repositories", Evidence: []Evidence{{Source: EvidenceSystem, Content: "git status"}},
	}
	current.CurrentStepStatus = StepInProgress
	current.CurrentStepSummary = progress.Summary
	current.CurrentEvidence = progress.Evidence
	current.LastEventID = "e2"
	current.UpdatedAt = now.Add(time.Minute)
	events = append(events, Event{
		SchemaVersion: CurrentEventSchemaVersion, ID: "e2", OccurredAt: current.UpdatedAt,
		Kind: EventFlowProgressed, Command: "flow.report.progress", Workflow: current.Release.Workflow,
		CausedByEventID: "e1", Payload: Payload(progress),
	})

	second := "confirm_requirements"
	flowResult := FlowResultPayload{
		ConditionResults: []ConditionResult{{
			ConditionID: "requirements_ready", Output: OutputValue{Type: workflow.OutputPath, Value: json.RawMessage(`"require_points.md"`)},
		}},
		Summary: "requirements ready", Effect: ResultAdvanced,
		Transition:    Transition{Direction: TransitionFlow, FromStepID: first, ToStepID: second},
		OutputChanges: OutputChanges{Accepted: []string{"require_points_path"}},
	}
	current.CurrentStepID = &second
	current.CurrentStepStatus = StepReady
	current.CurrentStepSummary = flowResult.Summary
	current.CurrentEvidence = nil
	current.Outputs = map[string]RegisteredOutput{
		"require_points_path": {Type: workflow.OutputPath, Value: json.RawMessage(`"require_points.md"`), ProducerStepID: first},
	}
	current.LastEventID = "e3"
	current.UpdatedAt = now.Add(2 * time.Minute)
	events = append(events, Event{
		SchemaVersion: CurrentEventSchemaVersion, ID: "e3", OccurredAt: current.UpdatedAt,
		Kind: EventFlowResult, Command: "flow.report.result", Workflow: current.Release.Workflow,
		CausedByEventID: "e2", Payload: Payload(flowResult),
	})

	loopResult := FlowResultPayload{
		ConditionResults: []ConditionResult{{
			ConditionID: "requirements_rejected",
			Output:      OutputValue{Type: workflow.OutputEnum, Value: json.RawMessage(`"rejected"`)},
		}},
		Summary: "requirements rejected", Effect: ResultLooped,
		Transition: Transition{Direction: TransitionLoop, FromStepID: second, ToStepID: first},
		OutputChanges: OutputChanges{
			Accepted:    []string{"requirements_decision"},
			Invalidated: []string{"require_points_path", "requirements_decision"},
		},
	}
	current.CurrentStepID = &first
	current.CurrentStepSummary = loopResult.Summary
	current.Outputs = map[string]RegisteredOutput{}
	current.LastEventID = "e4"
	current.UpdatedAt = now.Add(3 * time.Minute)
	events = append(events, Event{
		SchemaVersion: CurrentEventSchemaVersion, ID: "e4", OccurredAt: current.UpdatedAt,
		Kind: EventFlowResult, Command: "flow.report.result", Workflow: current.Release.Workflow,
		CausedByEventID: "e3", Payload: Payload(loopResult),
	})

	if err := current.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := current.ValidateAgainst(loaded.Workflow); err != nil {
		t.Fatal(err)
	}
	if err := ValidateHistory(events, current, loaded.Workflow); err != nil {
		t.Fatal(err)
	}
}

func TestHistoryRejectsIncompleteLoopInvalidation(t *testing.T) {
	loaded, err := workflow.Load("promotion-design")
	if err != nil {
		t.Fatal(err)
	}
	first, _ := loaded.Workflow.FirstStepID()
	now := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	current := State{
		SchemaVersion: CurrentStateSchemaVersion,
		Requirement:   Requirement{Title: "State v8"}, Release: Release{Version: "dev", Workflow: WorkflowRefFrom(loaded.Ref)},
		CurrentStepID: &first, CurrentStepStatus: StepReady, CurrentStepSummary: "requirements rejected",
		Outputs: map[string]RegisteredOutput{}, Integrations: Integrations{}, LastEventID: "e3", CreatedAt: now, UpdatedAt: now.Add(2 * time.Minute),
	}
	second := "confirm_requirements"
	events := []Event{
		{SchemaVersion: CurrentEventSchemaVersion, ID: "e1", OccurredAt: now, Kind: EventFlowInitialized, Command: "flow.init", Workflow: current.Release.Workflow, Payload: Payload(FlowInitializedPayload{StepID: first, StepStatus: StepReady})},
		{SchemaVersion: CurrentEventSchemaVersion, ID: "e2", OccurredAt: now.Add(time.Minute), Kind: EventFlowResult, Command: "flow.report.result", Workflow: current.Release.Workflow, CausedByEventID: "e1", Payload: Payload(FlowResultPayload{
			ConditionResults: []ConditionResult{{ConditionID: "requirements_ready", Output: OutputValue{Type: workflow.OutputPath, Value: json.RawMessage(`"require_points.md"`)}}},
			Summary:          "requirements ready", Effect: ResultAdvanced, Transition: Transition{Direction: TransitionFlow, FromStepID: first, ToStepID: second}, OutputChanges: OutputChanges{Accepted: []string{"require_points_path"}},
		})},
		{SchemaVersion: CurrentEventSchemaVersion, ID: "e3", OccurredAt: now.Add(2 * time.Minute), Kind: EventFlowResult, Command: "flow.report.result", Workflow: current.Release.Workflow, CausedByEventID: "e2", Payload: Payload(FlowResultPayload{
			ConditionResults: []ConditionResult{{ConditionID: "requirements_rejected", Output: OutputValue{Type: workflow.OutputEnum, Value: json.RawMessage(`"rejected"`)}}},
			Summary:          "requirements rejected", Effect: ResultLooped, Transition: Transition{Direction: TransitionLoop, FromStepID: second, ToStepID: first}, OutputChanges: OutputChanges{Accepted: []string{"requirements_decision"}, Invalidated: []string{"requirements_decision"}},
		})},
	}
	if err := ValidateHistory(events, current, loaded.Workflow); err == nil {
		t.Fatal("expected incomplete invalidation to be rejected")
	}
}

func TestDecodeRejectsOldGuardState(t *testing.T) {
	content := []byte(`{"schema_version":8,"requirement":{"title":"x"},"release":{"version":"dev","workflow":{"id":"fanloop","version":"7.0.0","digest":"sha256:x"}},"current_step_id":"confirm_repository_scope","current_step_status":"ready","current_guard_result":{"status":"failed"},"outputs":{},"integrations":{},"last_event_id":"e1","created_at":"2026-08-15T00:00:00Z","updated_at":"2026-08-15T00:00:00Z"}`)
	if _, err := Decode(content, map[string]RegisteredOutput{}); err == nil {
		t.Fatal("expected retired current_guard_result to be rejected")
	}
}

func TestTraceSyncedEventRejectsUnknownTargetFacts(t *testing.T) {
	valid := TraceSyncedPayload{Outcome: TraceSyncPartial, Targets: []TraceTargetResult{
		{Name: "trace_document", Status: "succeeded"},
		{Name: "registry", Status: "failed", Error: &TraceTargetError{Code: "REGISTRY_UPDATE_FAILED", Message: "unavailable", Retryable: true}},
	}}
	for name, mutate := range map[string]func(*TraceSyncedPayload){
		"target": func(value *TraceSyncedPayload) { value.Targets[0].Name = "unknown" },
		"status": func(value *TraceSyncedPayload) { value.Targets[0].Status = "unknown" },
		"code":   func(value *TraceSyncedPayload) { value.Targets[1].Error.Code = "UNKNOWN" },
	} {
		t.Run(name, func(t *testing.T) {
			payload := valid
			payload.Targets = append([]TraceTargetResult(nil), valid.Targets...)
			failure := *valid.Targets[1].Error
			payload.Targets[1].Error = &failure
			mutate(&payload)
			event := Event{Kind: EventTraceSynced, Command: "trace.sync", CausedByEventID: "e1", Payload: Payload(payload)}
			if err := ValidateEventPayload(event); err == nil {
				t.Fatal("expected invalid Trace target facts to be rejected")
			}
		})
	}
}
