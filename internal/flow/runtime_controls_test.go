package flow

import (
	"context"
	"testing"
	"time"

	"github.com/zeefan1555/fanloop/internal/idl/commonidl"
	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/idl/flowidl"
)

func TestRuntimeStartAndJumpControls(t *testing.T) {
	root := t.TempDir()
	nextID := 0
	runtime := Runtime{
		Clock:          func() time.Time { return time.Date(2026, 9, 17, 0, 0, nextID, 0, time.UTC) },
		EventID:        func() string { nextID++; return "event-" + string(rune('0'+nextID)) },
		ReleaseVersion: "test",
	}
	initialized, err := runtime.Init(context.Background(), root, &flowidl.FlowInitRequest{Workflow: "technical-solution-design", Requirement: &flowidl.Requirement{Title: "controls"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if initialized.State.Current.Execution.Status != flowidl.StepStatus_awaiting_confirmation || len(initialized.State.Current.AvailableRoutes) != 17 {
		t.Fatalf("initialized state = %#v", initialized.State)
	}
	if _, err := runtime.Progress(context.Background(), root, &flowidl.FlowProgressRequest{StepId: "frame_requirement_background", Status: flowidl.ProgressStatus_in_progress, Summary: "work", Evidence: []*flowidl.Evidence{}}, false); PublicError(err).Code != erroridl.ErrorCode_REPORT_NOT_ALLOWED {
		t.Fatalf("progress while awaiting error = %v", err)
	}
	confirmed := "confirmed"
	started, err := runtime.Result(context.Background(), root, &flowidl.FlowResultRequest{
		StepId: "frame_requirement_background", Summary: "confirmed", Evidence: humanEvidence(),
		ConditionResults: []*flowidl.ConditionResult{{ConditionId: "step_scope_confirmed", Output: &flowidl.OutputValue{Type: flowidl.OutputType_enum_value, Value: &commonidl.JsonValue{StringValue: &confirmed}}}},
		Route:            &flowidl.RouteSelection{StartCurrentStep: boolPointer(true)},
	}, false)
	if err != nil || started.Effect != flowidl.ResultEffect_started || started.State.Current.Execution.Status != flowidl.StepStatus_in_progress {
		t.Fatalf("started = %#v, error = %v", started, err)
	}
	target := "research_solution_options"
	jumped, err := runtime.Result(context.Background(), root, &flowidl.FlowResultRequest{
		StepId: "frame_requirement_background", Summary: "jump", Evidence: humanEvidence(),
		ConditionResults: []*flowidl.ConditionResult{{ConditionId: "human_step_jump_requested", Output: &flowidl.OutputValue{Type: flowidl.OutputType_string, Value: &commonidl.JsonValue{StringValue: &target}}}},
		Route:            &flowidl.RouteSelection{JumpStepId: &target},
	}, false)
	if err != nil || jumped.Effect != flowidl.ResultEffect_jumped || jumped.State.Current.Context.StepId != target || jumped.State.Current.Execution.Status != flowidl.StepStatus_awaiting_confirmation || len(jumped.State.SkippedStepIds) != 4 {
		t.Fatalf("jumped = %#v, error = %v", jumped, err)
	}
}

func humanEvidence() []*flowidl.Evidence {
	return []*flowidl.Evidence{{Source: flowidl.EvidenceSource_human, Content: "approved"}}
}
