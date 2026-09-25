package state

import (
	"encoding/json"
	"maps"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/zeefan1555/fanloop/internal/workflow"
)

func TestHistoryReplaysProgressFlowResultAndLoopInvalidation(t *testing.T) {
	loaded, err := loadLinearRuntime()
	if err != nil {
		t.Fatal(err)
	}
	stepStart := loaded.Workflow.StepStart
	loaded.Workflow.StepStart, loaded.Workflow.Jump = nil, nil
	loaded.Workflow.CommonSkills, loaded.Workflow.CommonConditions = nil, nil
	first, _ := loaded.Workflow.FirstStepID()
	second := "second"
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
			ConditionID: "first_completed", Output: OutputValue{Type: workflow.OutputPath, Value: json.RawMessage(`"first.md"`)},
		}},
		Summary: "background defined", Effect: ResultAdvanced,
		Transition:    Transition{Direction: TransitionFlow, FromStepID: first, ToStepID: second},
		OutputChanges: OutputChanges{Accepted: []string{"first_artifact"}},
	}
	current.CurrentStepID = &second
	current.CurrentStepStatus = StepReady
	current.CurrentStepSummary = flowResult.Summary
	current.CurrentEvidence = nil
	current.Outputs = map[string]RegisteredOutput{
		"first_artifact": {Type: workflow.OutputPath, Value: json.RawMessage(`"first.md"`), ProducerStepID: first},
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
			ConditionID: "second_rework", Output: OutputValue{Type: workflow.OutputEnum, Value: json.RawMessage(`"rework"`)},
		}},
		Summary: "background changed", Effect: ResultLooped,
		Transition: Transition{Direction: TransitionLoop, FromStepID: second, ToStepID: first},
		OutputChanges: OutputChanges{
			Accepted:    []string{"second_revision"},
			Invalidated: []string{"first_artifact"},
		},
	}
	current.CurrentStepID = &first
	current.CurrentStepStatus = StepFixing
	current.CurrentStepSummary = loopResult.Summary
	current.Outputs = map[string]RegisteredOutput{
		"second_revision": {Type: workflow.OutputEnum, Value: json.RawMessage(`"rework"`), ProducerStepID: second},
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
	current.CurrentStepStatus = StepReady
	if err := ValidateHistory(events, current, loaded); err == nil {
		t.Fatal("history replay accepted a looped tail that was not fixing")
	}
	current.CurrentStepStatus = StepFixing
	loaded.Workflow.Flows[first] = nil
	loaded.Workflow.Loops[second] = nil
	loaded.Workflow.StepStart = stepStart
	if err := ValidateHistory(events, current, loaded); err != nil {
		t.Fatalf("historical events were re-evaluated against live Routes or controls: %v", err)
	}
}

func TestHistoryPreservesOutputsAcceptedByLoopResult(t *testing.T) {
	loaded, err := loadLinearRuntime()
	if err != nil {
		t.Fatal(err)
	}
	loaded.Workflow.StepStart, loaded.Workflow.Jump = nil, nil
	loaded.Workflow.CommonSkills, loaded.Workflow.CommonConditions = nil, nil
	first, _ := loaded.Workflow.FirstStepID()
	second := "second"
	now := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	current := State{
		SchemaVersion: CurrentStateSchemaVersion,
		Requirement:   Requirement{Title: "State v8"}, Release: Release{Version: "dev", Workflow: WorkflowRefFrom(loaded.Ref)},
		CurrentStepID: &first, CurrentStepStatus: StepFixing, CurrentStepSummary: "second rework",
		Outputs: map[string]RegisteredOutput{
			"second_revision": {Type: workflow.OutputEnum, Value: json.RawMessage(`"rework"`), ProducerStepID: second},
		}, Integrations: Integrations{}, LastEventID: "e3", CreatedAt: now, UpdatedAt: now.Add(2 * time.Minute),
	}
	events := []Event{
		{SchemaVersion: CurrentEventSchemaVersion, ID: "e1", OccurredAt: now, Kind: EventFlowInitialized, Command: "flow.init", Workflow: current.Release.Workflow, Payload: Payload(FlowInitializedPayload{StepID: first, StepStatus: StepReady})},
		{SchemaVersion: CurrentEventSchemaVersion, ID: "e2", OccurredAt: now.Add(time.Minute), Kind: EventFlowResult, Command: "flow.report.result", Workflow: current.Release.Workflow, CausedByEventID: "e1", Payload: Payload(FlowResultPayload{
			ConditionResults: []ConditionResult{{ConditionID: "first_completed", Output: OutputValue{Type: workflow.OutputPath, Value: json.RawMessage(`"first.md"`)}}},
			Summary:          "first complete", Effect: ResultAdvanced, Transition: Transition{Direction: TransitionFlow, FromStepID: first, ToStepID: second}, OutputChanges: OutputChanges{Accepted: []string{"first_artifact"}},
		})},
		{SchemaVersion: CurrentEventSchemaVersion, ID: "e3", OccurredAt: now.Add(2 * time.Minute), Kind: EventFlowResult, Command: "flow.report.result", Workflow: current.Release.Workflow, CausedByEventID: "e2", Payload: Payload(FlowResultPayload{
			ConditionResults: []ConditionResult{{
				ConditionID: "second_rework", Output: OutputValue{Type: workflow.OutputEnum, Value: json.RawMessage(`"rework"`)},
			}},
			Summary: "second rework", Effect: ResultLooped, Transition: Transition{Direction: TransitionLoop, FromStepID: second, ToStepID: first}, OutputChanges: OutputChanges{
				Accepted: []string{"second_revision"}, Invalidated: []string{"first_artifact"},
			},
		})},
	}
	if err := ValidateHistory(events, current, loaded); err != nil {
		t.Fatalf("loop result lost its newly accepted Output: %v", err)
	}
}

func TestApplyFlowResultCoversEveryEffect(t *testing.T) {
	loaded, err := loadLinearRuntime()
	if err != nil {
		t.Fatal(err)
	}
	evidence := []Evidence{{Source: EvidenceSystem, Content: "checked"}}
	artifact := func(conditionID, value string) []ConditionResult {
		return []ConditionResult{{ConditionID: conditionID, Output: OutputValue{Type: workflow.OutputPath, Value: json.RawMessage(`"` + value + `"`)}}}
	}
	base := func(step string) State {
		return State{
			CurrentStepID: &step, CurrentStepStatus: StepInProgress,
			Outputs: map[string]RegisteredOutput{
				"first_artifact": {Type: workflow.OutputPath, Value: json.RawMessage(`"first.md"`), ProducerStepID: "first"},
				"third_artifact": {Type: workflow.OutputPath, Value: json.RawMessage(`"third.md"`), ProducerStepID: "third"},
			},
		}
	}
	tests := []struct {
		name        string
		current     State
		payload     FlowResultPayload
		wantStep    string
		wantStatus  StepStatus
		wantOutputs []string
		wantSkipped []string
	}{
		{
			name: "advanced", current: base("first"), wantStep: "second", wantStatus: StepReady,
			payload: FlowResultPayload{ConditionResults: artifact("first_completed", "next.md"), Summary: "advanced", Evidence: evidence, Effect: ResultAdvanced,
				Transition: Transition{Direction: TransitionFlow, FromStepID: "first", ToStepID: "second"}, OutputChanges: OutputChanges{Accepted: []string{"first_artifact"}}},
			wantOutputs: []string{"first_artifact", "third_artifact"},
		},
		{
			name: "looped", current: base("second"), wantStep: "first", wantStatus: StepFixing,
			payload: FlowResultPayload{ConditionResults: []ConditionResult{{ConditionID: "second_rework", Output: OutputValue{Type: workflow.OutputEnum, Value: json.RawMessage(`"rework"`)}}}, Summary: "looped", Evidence: evidence, Effect: ResultLooped,
				Transition: Transition{Direction: TransitionLoop, FromStepID: "second", ToStepID: "first"}, OutputChanges: OutputChanges{Accepted: []string{"second_revision"}, Invalidated: []string{"first_artifact", "third_artifact"}}},
			wantOutputs: []string{"second_revision"},
		},
		{
			name: "completed", current: base("third"),
			payload: FlowResultPayload{ConditionResults: artifact("third_completed", "done.md"), Summary: "completed", Evidence: evidence, Effect: ResultCompleted,
				Transition: Transition{Direction: TransitionFlow, FromStepID: "third"}, OutputChanges: OutputChanges{Accepted: []string{"third_artifact"}}},
			wantOutputs: []string{"first_artifact", "third_artifact"},
		},
		{
			name: "started", current: base("first"), wantStep: "first", wantStatus: StepInProgress,
			payload: FlowResultPayload{ConditionResults: []ConditionResult{{ConditionID: "start_confirmed", Output: OutputValue{Type: workflow.OutputEnum, Value: json.RawMessage(`"confirmed"`)}}}, Summary: "started", Evidence: evidence, Effect: ResultStarted,
				Transition: Transition{Direction: TransitionStart, FromStepID: "first", ToStepID: "first"}},
			wantOutputs: []string{"first_artifact", "third_artifact"},
		},
		{
			name: "jumped", current: base("first"), wantStep: "third", wantStatus: StepReady,
			payload: FlowResultPayload{ConditionResults: []ConditionResult{{ConditionID: "jump_requested", Output: OutputValue{Type: workflow.OutputString, Value: json.RawMessage(`"third"`)}}}, Summary: "jumped", Evidence: evidence, Effect: ResultJumped,
				Transition: Transition{Direction: TransitionJump, FromStepID: "first", ToStepID: "third"}, OutputChanges: OutputChanges{Invalidated: []string{"third_artifact"}}},
			wantOutputs: []string{"first_artifact"}, wantSkipped: []string{"first", "second"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			next, err := ApplyFlowResult(loaded.Workflow, test.current, test.payload, ResultTargetStatus(loaded.Workflow, test.payload.Effect))
			if err != nil {
				t.Fatal(err)
			}
			if got := StateRef(next); got != (StepState{StepID: test.wantStep, Status: test.wantStatus}) {
				t.Fatalf("current = %#v, want step=%q status=%q", got, test.wantStep, test.wantStatus)
			}
			if next.CurrentStepSummary != map[bool]string{true: "", false: test.payload.Summary}[test.payload.Effect == ResultCompleted] ||
				!slices.Equal(next.CurrentEvidence, map[bool][]Evidence{true: nil, false: evidence}[test.payload.Effect == ResultCompleted]) ||
				!slices.Equal(sortedKeys(next.Outputs), test.wantOutputs) || !slices.Equal(next.SkippedStepIDs, test.wantSkipped) {
				t.Fatalf("next = %#v", next)
			}
		})
	}

	current := base("first")
	current.SkippedStepIDs = []string{"first"}
	beforeOutputs := maps.Clone(current.Outputs)
	beforeSkipped := append([]string(nil), current.SkippedStepIDs...)
	_, err = ApplyFlowResult(loaded.Workflow, current, FlowResultPayload{
		ConditionResults: []ConditionResult{{ConditionID: "missing", Output: OutputValue{Type: workflow.OutputPath, Value: json.RawMessage(`"missing.md"`)}}},
		Effect:           ResultAdvanced, Transition: Transition{Direction: TransitionFlow, FromStepID: "first", ToStepID: "second"},
	}, StepReady)
	if err == nil {
		t.Fatal("invalid Result was applied")
	}
	if !maps.EqualFunc(current.Outputs, beforeOutputs, equalRegisteredOutput) || !slices.Equal(current.SkippedStepIDs, beforeSkipped) {
		t.Fatalf("failed Result mutated input State: %#v", current)
	}
}

func TestHistoryDoesNotReinterpretIntermediateResultStatus(t *testing.T) {
	loaded, err := loadLinearRuntime()
	if err != nil {
		t.Fatal(err)
	}
	loaded.Workflow.StepStart, loaded.Workflow.Jump = nil, nil
	loaded.Workflow.CommonSkills, loaded.Workflow.CommonConditions = nil, nil
	now := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	workflowRef := WorkflowRefFrom(loaded.Ref)
	current := State{
		SchemaVersion: CurrentStateSchemaVersion, Requirement: Requirement{Title: "historical loop status"},
		Release: Release{Version: "dev", Workflow: workflowRef}, CurrentStepID: stringPointer("first"), CurrentStepStatus: StepInProgress,
		CurrentStepSummary: "continued under historical status", Outputs: map[string]RegisteredOutput{"second_revision": {Type: workflow.OutputEnum, Value: json.RawMessage(`"rework"`), ProducerStepID: "second"}},
		SkippedStepIDs: []string{}, Integrations: Integrations{}, LastEventID: "e4", CreatedAt: now, UpdatedAt: now.Add(3 * time.Minute),
	}
	events := []Event{
		{SchemaVersion: CurrentEventSchemaVersion, ID: "e1", OccurredAt: now, Kind: EventFlowInitialized, Command: "flow.init", Workflow: workflowRef, Payload: Payload(FlowInitializedPayload{StepID: "first", StepStatus: StepReady})},
		{SchemaVersion: CurrentEventSchemaVersion, ID: "e2", OccurredAt: now.Add(time.Minute), Kind: EventFlowResult, Command: "flow.report.result", Workflow: workflowRef, CausedByEventID: "e1", Payload: Payload(FlowResultPayload{
			ConditionResults: []ConditionResult{{ConditionID: "first_completed", Output: OutputValue{Type: workflow.OutputPath, Value: json.RawMessage(`"first.md"`)}}},
			Summary:          "advanced", Effect: ResultAdvanced, Transition: Transition{Direction: TransitionFlow, FromStepID: "first", ToStepID: "second"}, OutputChanges: OutputChanges{Accepted: []string{"first_artifact"}},
		})},
		{SchemaVersion: CurrentEventSchemaVersion, ID: "e3", OccurredAt: now.Add(2 * time.Minute), Kind: EventFlowResult, Command: "flow.report.result", Workflow: workflowRef, CausedByEventID: "e2", Payload: Payload(FlowResultPayload{
			ConditionResults: []ConditionResult{{ConditionID: "second_rework", Output: OutputValue{Type: workflow.OutputEnum, Value: json.RawMessage(`"rework"`)}}},
			Summary:          "looped before fixing status existed", Effect: ResultLooped, Transition: Transition{Direction: TransitionLoop, FromStepID: "second", ToStepID: "first"}, OutputChanges: OutputChanges{Accepted: []string{"second_revision"}, Invalidated: []string{"first_artifact"}},
		})},
		{SchemaVersion: CurrentEventSchemaVersion, ID: "e4", OccurredAt: now.Add(3 * time.Minute), Kind: EventFlowProgressed, Command: "flow.report.progress", Workflow: workflowRef, CausedByEventID: "e3", Payload: Payload(FlowProgressPayload{
			FromStepID: "first", FromStepStatus: StepReady, ToStepStatus: StepInProgress, Summary: "continued under historical status",
		})},
	}
	if err := ValidateHistory(events, current, loaded); err != nil {
		t.Fatalf("intermediate Result status was reinterpreted: %v", err)
	}
}

func TestHistoryPreservesStartedStatusBetweenEvents(t *testing.T) {
	loaded, err := loadRuntimeControls()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	workflowRef := WorkflowRefFrom(loaded.Ref)
	evidence := []Evidence{{Source: EvidenceHuman, Content: "approved"}}
	current := State{
		SchemaVersion: CurrentStateSchemaVersion, Requirement: Requirement{Title: "started status"},
		Release: Release{Version: "dev", Workflow: workflowRef}, CurrentStepID: stringPointer("first"), CurrentStepStatus: StepInProgress,
		CurrentStepSummary: "continued", Outputs: map[string]RegisteredOutput{}, SkippedStepIDs: []string{}, Integrations: Integrations{},
		LastEventID: "e3", CreatedAt: now, UpdatedAt: now.Add(2 * time.Minute),
	}
	events := []Event{
		{SchemaVersion: CurrentEventSchemaVersion, ID: "e1", OccurredAt: now, Kind: EventFlowInitialized, Command: "flow.init", Workflow: workflowRef, Payload: Payload(FlowInitializedPayload{StepID: "first", StepStatus: StepAwaitingConfirmation})},
		{SchemaVersion: CurrentEventSchemaVersion, ID: "e2", OccurredAt: now.Add(time.Minute), Kind: EventFlowResult, Command: "flow.report.result", Workflow: workflowRef, CausedByEventID: "e1", Payload: Payload(FlowResultPayload{
			ConditionResults: []ConditionResult{{ConditionID: "start_confirmed", Output: OutputValue{Type: workflow.OutputEnum, Value: json.RawMessage(`"confirmed"`)}}},
			Summary:          "started", Evidence: evidence, Effect: ResultStarted, Transition: Transition{Direction: TransitionStart, FromStepID: "first", ToStepID: "first"}, OutputChanges: OutputChanges{},
		})},
		{SchemaVersion: CurrentEventSchemaVersion, ID: "e3", OccurredAt: now.Add(2 * time.Minute), Kind: EventFlowProgressed, Command: "flow.report.progress", Workflow: workflowRef, CausedByEventID: "e2", Payload: Payload(FlowProgressPayload{
			FromStepID: "first", FromStepStatus: StepReady, ToStepStatus: StepInProgress, Summary: "continued",
		})},
	}
	if err := ValidateHistory(events, current, loaded); err == nil {
		t.Fatal("history replay accepted Progress from a status that did not follow started")
	}
}

func TestHistoricalControlResultsAreNotReevaluatedAgainstLiveControls(t *testing.T) {
	loaded, err := loadRuntimeControls()
	if err != nil {
		t.Fatal(err)
	}
	steps := loaded.Workflow.OrderedStepIDs()
	evidence := []Evidence{{Source: EvidenceHuman, Content: "approved"}}
	started := FlowResultPayload{
		ConditionResults: []ConditionResult{{
			ConditionID: "start_confirmed",
			Output:      OutputValue{Type: workflow.OutputEnum, Value: json.RawMessage(`"confirmed"`)},
		}},
		Summary: "started", Effect: ResultStarted,
		Transition: Transition{Direction: TransitionStart, FromStepID: steps[0], ToStepID: steps[0]},
		Evidence:   evidence,
	}
	jumped := FlowResultPayload{
		ConditionResults: []ConditionResult{{
			ConditionID: "jump_requested",
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

func TestResultSelectsConditionsByExplicitFlowTarget(t *testing.T) {
	loaded, err := loadLinearRuntime()
	if err != nil {
		t.Fatal(err)
	}
	payload := FlowResultPayload{
		ConditionResults: []ConditionResult{{
			ConditionID: "first_completed", Output: OutputValue{Type: workflow.OutputPath, Value: json.RawMessage(`"first.md"`)},
		}},
		Summary: "first complete",
		Effect:  ResultAdvanced,
		Transition: Transition{
			Direction: TransitionFlow, FromStepID: "first", ToStepID: "second",
		},
		OutputChanges: OutputChanges{Accepted: []string{"first_artifact"}},
	}
	if err := validateResultAgainst(loaded.Workflow, payload); err != nil {
		t.Fatalf("explicit Flow target rejected: %v", err)
	}
}

func loadLinearRuntime() (workflow.Loaded, error) {
	return workflow.LoadFrom(filepath.Join("..", "workflow", "testdata"), "linear-runtime")
}

func loadRuntimeControls() (workflow.Loaded, error) {
	return workflow.LoadFrom(filepath.Join("..", "workflow", "testdata"), "runtime-controls")
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
