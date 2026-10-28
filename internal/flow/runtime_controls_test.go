package flow

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/zeefan1555/fanloop/internal/idl/commonidl"
	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/idl/flowidl"
)

func TestRuntimeStartAndJumpControls(t *testing.T) {
	dataRoot := t.TempDir()
	workflowsRoot := filepath.Join(dataRoot, "source", "fanloop-workflow")
	if err := os.CopyFS(filepath.Join(workflowsRoot, "runtime-controls"), os.DirFS(filepath.Join("..", "workflow", "testdata", "runtime-controls"))); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_CONFIG_SOURCE", filepath.Join(dataRoot, "source"))
	root := t.TempDir()
	nextID := 0
	runtime := Runtime{
		Clock:             func() time.Time { return time.Date(2026, 9, 17, 0, 0, nextID, 0, time.UTC) },
		EventID:           func() string { nextID++; return "event-" + string(rune('0'+nextID)) },
		ControllerRelease: testControllerRelease,
		ReleaseVersion:    "test",
	}
	initialized, err := runtime.Init(context.Background(), root, &flowidl.FlowInitRequest{Workflow: "runtime-controls", Requirement: &flowidl.Requirement{Title: "controls"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if initialized.State.Current.Execution.Status != flowidl.StepStatus_awaiting_confirmation {
		t.Fatalf("initialized state = %#v", initialized.State)
	}
	if _, err := runtime.Progress(context.Background(), root, &flowidl.FlowProgressRequest{StepId: "first", Status: flowidl.ProgressStatus_in_progress, Summary: "work", Evidence: []*flowidl.Evidence{}}, false); PublicError(err).Code != erroridl.ErrorCode_REPORT_NOT_ALLOWED {
		t.Fatalf("progress while awaiting error = %v", err)
	}
	confirmed := "confirmed"
	started, err := runtime.Result(context.Background(), root, &flowidl.FlowResultRequest{
		StepId: "first", Summary: "confirmed", Evidence: humanEvidence(),
		ConditionResults: []*flowidl.ConditionResult{{ConditionId: "start_confirmed", Output: &flowidl.OutputValue{Type: flowidl.OutputType_enum_value, Value: &commonidl.JsonValue{StringValue: &confirmed}}}},
		Route:            &flowidl.RouteSelection{StartCurrentStep: boolPointer(true)},
	}, false)
	if err != nil || started.Effect != flowidl.ResultEffect_started || started.State.Current.Execution.Status != flowidl.StepStatus_in_progress {
		t.Fatalf("started = %#v, error = %v", started, err)
	}
	startedStatus, err := runtime.Status(context.Background(), root, &flowidl.FlowStatusRequest{})
	if err != nil || startedStatus.State.Current.Execution.Status != flowidl.StepStatus_in_progress {
		t.Fatalf("started replay = %#v, error = %v", startedStatus, err)
	}
	target := "third"
	jumped, err := runtime.Result(context.Background(), root, &flowidl.FlowResultRequest{
		StepId: "first", Summary: "jump", Evidence: humanEvidence(),
		ConditionResults: []*flowidl.ConditionResult{{ConditionId: "jump_requested", Output: &flowidl.OutputValue{Type: flowidl.OutputType_string, Value: &commonidl.JsonValue{StringValue: &target}}}},
		Route:            &flowidl.RouteSelection{JumpStepId: &target},
	}, false)
	if err != nil || jumped.Effect != flowidl.ResultEffect_jumped || jumped.State.Current.Context.StepId != target || jumped.State.Current.Execution.Status != flowidl.StepStatus_awaiting_confirmation || len(jumped.State.SkippedStepIds) != 2 {
		t.Fatalf("jumped = %#v, error = %v", jumped, err)
	}
	jumpedStatus, err := runtime.Status(context.Background(), root, &flowidl.FlowStatusRequest{})
	if err != nil || jumpedStatus.State.Current.Context.StepId != target || jumpedStatus.State.Current.Execution.Status != flowidl.StepStatus_awaiting_confirmation || len(jumpedStatus.State.SkippedStepIds) != 2 {
		t.Fatalf("jumped replay = %#v, error = %v", jumpedStatus, err)
	}
}

func humanEvidence() []*flowidl.Evidence {
	return []*flowidl.Evidence{{Source: flowidl.EvidenceSource_human, Content: "approved"}}
}
