package state

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/zeefan1555/fanloop/internal/workflow"
)

func TestHistoryReplaysProgressFlowResultAndLoopInvalidation(t *testing.T) {
	loaded, err := workflow.Load("technical-solution-design")
	if err != nil {
		t.Fatal(err)
	}
	stepStart := loaded.Workflow.StepStart
	loaded.Workflow.StepStart, loaded.Workflow.Jump = nil, nil
	loaded.Workflow.CommonSkills, loaded.Workflow.CommonConditions = nil, nil
	first, _ := loaded.Workflow.FirstStepID()
	second := "define_goals_and_problems"
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

	flowResult := FlowResultPayload{
		ConditionResults: []ConditionResult{{
			ConditionID: "background_defined", Output: OutputValue{Type: workflow.OutputPath, Value: json.RawMessage(`".technical-solution/sections/01-business-background.md"`)},
		}},
		Summary: "background defined", Effect: ResultAdvanced,
		Transition:    Transition{Direction: TransitionFlow, FromStepID: first, ToStepID: second},
		OutputChanges: OutputChanges{Accepted: []string{"background_section_path"}},
	}
	current.CurrentStepID = &second
	current.CurrentStepStatus = StepReady
	current.CurrentStepSummary = flowResult.Summary
	current.CurrentEvidence = nil
	current.Outputs = map[string]RegisteredOutput{
		"background_section_path": {Type: workflow.OutputPath, Value: json.RawMessage(`".technical-solution/sections/01-business-background.md"`), ProducerStepID: first},
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
			ConditionID: "background_changed", Output: OutputValue{Type: workflow.OutputEnum, Value: json.RawMessage(`"background"`)},
		}},
		Summary: "background changed", Effect: ResultLooped,
		Transition: Transition{Direction: TransitionLoop, FromStepID: second, ToStepID: first},
		OutputChanges: OutputChanges{
			Accepted:    []string{"technical_feedback_scope"},
			Invalidated: []string{"background_section_path"},
		},
	}
	current.CurrentStepID = &first
	current.CurrentStepSummary = loopResult.Summary
	current.Outputs = map[string]RegisteredOutput{
		"technical_feedback_scope": {Type: workflow.OutputEnum, Value: json.RawMessage(`"background"`), ProducerStepID: second},
	}
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
	if err := ValidateHistory(events, current, loaded); err != nil {
		t.Fatal(err)
	}
	loaded.Workflow.Flows[first] = nil
	loaded.Workflow.Loops[second] = nil
	loaded.Workflow.StepStart = stepStart
	if err := ValidateHistory(events, current, loaded); err != nil {
		t.Fatalf("historical events were re-evaluated against live Routes or controls: %v", err)
	}
}

func TestHistoryPreservesOutputsAcceptedByLoopResult(t *testing.T) {
	loaded, err := workflow.Load("technical-solution-design")
	if err != nil {
		t.Fatal(err)
	}
	loaded.Workflow.StepStart, loaded.Workflow.Jump = nil, nil
	loaded.Workflow.CommonSkills, loaded.Workflow.CommonConditions = nil, nil
	first, _ := loaded.Workflow.FirstStepID()
	second := "define_goals_and_problems"
	now := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	current := State{
		SchemaVersion: CurrentStateSchemaVersion,
		Requirement:   Requirement{Title: "State v8"}, Release: Release{Version: "dev", Workflow: WorkflowRefFrom(loaded.Ref)},
		CurrentStepID: &first, CurrentStepStatus: StepReady, CurrentStepSummary: "background changed",
		Outputs: map[string]RegisteredOutput{
			"technical_feedback_scope": {Type: workflow.OutputEnum, Value: json.RawMessage(`"background"`), ProducerStepID: second},
		}, Integrations: Integrations{}, LastEventID: "e3", CreatedAt: now, UpdatedAt: now.Add(2 * time.Minute),
	}
	events := []Event{
		{SchemaVersion: CurrentEventSchemaVersion, ID: "e1", OccurredAt: now, Kind: EventFlowInitialized, Command: "flow.init", Workflow: current.Release.Workflow, Payload: Payload(FlowInitializedPayload{StepID: first, StepStatus: StepReady})},
		{SchemaVersion: CurrentEventSchemaVersion, ID: "e2", OccurredAt: now.Add(time.Minute), Kind: EventFlowResult, Command: "flow.report.result", Workflow: current.Release.Workflow, CausedByEventID: "e1", Payload: Payload(FlowResultPayload{
			ConditionResults: []ConditionResult{{ConditionID: "background_defined", Output: OutputValue{Type: workflow.OutputPath, Value: json.RawMessage(`".technical-solution/sections/01-business-background.md"`)}}},
			Summary:          "background defined", Effect: ResultAdvanced, Transition: Transition{Direction: TransitionFlow, FromStepID: first, ToStepID: second}, OutputChanges: OutputChanges{Accepted: []string{"background_section_path"}},
		})},
		{SchemaVersion: CurrentEventSchemaVersion, ID: "e3", OccurredAt: now.Add(2 * time.Minute), Kind: EventFlowResult, Command: "flow.report.result", Workflow: current.Release.Workflow, CausedByEventID: "e2", Payload: Payload(FlowResultPayload{
			ConditionResults: []ConditionResult{{
				ConditionID: "background_changed", Output: OutputValue{Type: workflow.OutputEnum, Value: json.RawMessage(`"background"`)},
			}},
			Summary: "background changed", Effect: ResultLooped, Transition: Transition{Direction: TransitionLoop, FromStepID: second, ToStepID: first}, OutputChanges: OutputChanges{
				Accepted: []string{"technical_feedback_scope"}, Invalidated: []string{"background_section_path"},
			},
		})},
	}
	if err := ValidateHistory(events, current, loaded); err != nil {
		t.Fatalf("loop result lost its newly accepted Output: %v", err)
	}
}

func TestHistoricalControlResultsAreNotReevaluatedAgainstLiveControls(t *testing.T) {
	loaded, err := workflow.Load("technical-solution-design")
	if err != nil {
		t.Fatal(err)
	}
	steps := loaded.Workflow.OrderedStepIDs()
	evidence := []Evidence{{Source: EvidenceHuman, Content: "approved"}}
	started := FlowResultPayload{
		ConditionResults: []ConditionResult{{
			ConditionID: "step_scope_confirmed",
			Output:      OutputValue{Type: workflow.OutputEnum, Value: json.RawMessage(`"confirmed"`)},
		}},
		Summary: "started", Effect: ResultStarted,
		Transition: Transition{Direction: TransitionStart, FromStepID: steps[0], ToStepID: steps[0]},
		Evidence:   evidence,
	}
	jumped := FlowResultPayload{
		ConditionResults: []ConditionResult{{
			ConditionID: "human_step_jump_requested",
			Output:      OutputValue{Type: workflow.OutputString, Value: json.RawMessage(`"` + steps[1] + `"`)},
		}},
		Summary: "jumped", Effect: ResultJumped,
		Transition: Transition{Direction: TransitionJump, FromStepID: steps[0], ToStepID: steps[1]},
		Evidence:   evidence,
	}
	if err := validateResultAgainst(loaded.Workflow, started); err != nil {
		t.Fatalf("valid start Result rejected before live edit: %v", err)
	}
	if err := validateResultAgainst(loaded.Workflow, jumped); err != nil {
		t.Fatalf("valid jump Result rejected before live edit: %v", err)
	}

	loaded.Workflow.StepStart = nil
	loaded.Workflow.Jump = nil
	loaded.Workflow.CommonConditions = nil
	if err := validateHistoricalResultAgainst(loaded.Workflow, started); err != nil {
		t.Fatalf("historical start Result was re-evaluated against live controls: %v", err)
	}
	if err := validateHistoricalResultAgainst(loaded.Workflow, jumped); err != nil {
		t.Fatalf("historical jump Result was re-evaluated against live controls: %v", err)
	}
}

func TestResultSelectsSharedConditionsByExplicitFlowTarget(t *testing.T) {
	loaded, err := workflow.Load("fanloop-maintainer")
	if err != nil {
		t.Fatal(err)
	}
	payload := FlowResultPayload{
		ConditionResults: []ConditionResult{
			{ConditionID: "verification_contract_written", Output: OutputValue{Type: workflow.OutputPath, Value: json.RawMessage(`"requirements.md"`)}},
			{ConditionID: "requirements_approved", Output: OutputValue{Type: workflow.OutputEnum, Value: json.RawMessage(`"approved"`)}},
			{ConditionID: "requirements_decision_recorded", Output: OutputValue{Type: workflow.OutputString, Value: json.RawMessage(`"approval-receipt"`)}},
			{ConditionID: "panorama_presented", Output: OutputValue{Type: workflow.OutputPath, Value: json.RawMessage(`".fanloop/card/define.json"`)}},
		},
		Summary: "verification contract approved",
		Effect:  ResultAdvanced,
		Transition: Transition{
			Direction: TransitionFlow, FromStepID: "define_verification_contract", ToStepID: "build_until_verified",
		},
		OutputChanges: OutputChanges{Accepted: []string{
			"verification_contract_path", "requirements_decision", "requirements_decision_receipt_id", "panorama_snapshot_path",
		}},
	}
	if err := validateResultAgainst(loaded.Workflow, payload); err != nil {
		t.Fatalf("explicit Flow target rejected: %v", err)
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
