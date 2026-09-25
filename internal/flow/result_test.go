package flow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/zeefan1555/fanloop/internal/idl/commonidl"
	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/idl/flowidl"
	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

func TestEvaluateResultRoutesTechnicalReviewAndInvalidatesByProducer(t *testing.T) {
	loaded, err := loadRuntimeControls()
	if err != nil {
		t.Fatal(err)
	}
	reviewState := state.State{Outputs: map[string]state.RegisteredOutput{
		"first_artifact": {
			Type: workflow.OutputPath, Value: json.RawMessage(`"first.md"`), ProducerStepID: "first",
		},
	}}

	t.Run("rework loops to the previous step", func(t *testing.T) {
		request := resultRequest("second", backRoute("first"),
			condition("second_rework", flowidl.OutputType_enum_value, "rework"),
		)
		evaluation, failure := evaluateResult(loaded.Workflow, reviewState, request)
		if failure != nil {
			t.Fatal(failure)
		}
		want := []string{"first_artifact"}
		if evaluation.effect != flowidl.ResultEffect_looped || evaluation.transition.GetToStepId() != "first" || !sameStrings(evaluation.invalidated, want) {
			t.Fatalf("evaluation = %#v", evaluation)
		}
	})

	t.Run("exclusive conditions conflict", func(t *testing.T) {
		request := resultRequest("second", nextRoute("third"),
			condition("second_completed", flowidl.OutputType_path, "second.md"),
			condition("second_evidence", flowidl.OutputType_path, "evidence.md"),
			condition("second_rework", flowidl.OutputType_enum_value, "rework"),
		)
		_, failure := evaluateResult(loaded.Workflow, reviewState, request)
		if failure == nil || failure.Code != erroridl.ErrorCode_CONDITION_CONFLICT {
			t.Fatalf("failure = %#v", failure)
		}
	})

	t.Run("route is required", func(t *testing.T) {
		request := resultRequest("second", nil,
			condition("second_rework", flowidl.OutputType_enum_value, "rework"),
		)
		failure := validateResultRequest(request)
		if failure == nil || failure.Code != erroridl.ErrorCode_INVALID_ARGUMENT {
			t.Fatalf("failure = %#v", failure)
		}
	})

	t.Run("incomplete condition set matches no route", func(t *testing.T) {
		request := resultRequest("second", nextRoute("third"),
			condition("second_completed", flowidl.OutputType_path, "second.md"),
		)
		_, failure := evaluateResult(loaded.Workflow, reviewState, request)
		if failure == nil || failure.Code != erroridl.ErrorCode_ROUTE_NOT_MATCHED {
			t.Fatalf("failure = %#v", failure)
		}
	})

	t.Run("Flow conditions do not match selected Loop Route", func(t *testing.T) {
		request := resultRequest("second", backRoute("first"),
			condition("second_completed", flowidl.OutputType_path, "second.md"),
			condition("second_evidence", flowidl.OutputType_path, "evidence.md"),
		)
		_, failure := evaluateResult(loaded.Workflow, reviewState, request)
		if failure == nil || failure.Code != erroridl.ErrorCode_ROUTE_NOT_MATCHED {
			t.Fatalf("failure = %#v", failure)
		}
	})

	t.Run("rework cannot advance", func(t *testing.T) {
		request := resultRequest("second", nextRoute("third"),
			condition("second_rework", flowidl.OutputType_enum_value, "rework"),
		)
		_, failure := evaluateResult(loaded.Workflow, reviewState, request)
		if failure == nil || failure.Code != erroridl.ErrorCode_ROUTE_NOT_MATCHED {
			t.Fatalf("failure = %#v", failure)
		}
	})

	t.Run("unknown target is not allowed", func(t *testing.T) {
		request := resultRequest("second", nextRoute("missing_step"),
			condition("second_completed", flowidl.OutputType_path, "second.md"),
			condition("second_evidence", flowidl.OutputType_path, "evidence.md"),
		)
		_, failure := evaluateResult(loaded.Workflow, reviewState, request)
		if failure == nil || failure.Code != erroridl.ErrorCode_ROUTE_NOT_ALLOWED {
			t.Fatalf("failure = %#v", failure)
		}
	})

	t.Run("multiple Flow Routes are rejected", func(t *testing.T) {
		ambiguous := loaded.Workflow
		ambiguous.Flows = cloneFlowRoutes(loaded.Workflow.Flows)
		ambiguous.Flows["second"] = append(ambiguous.Flows["second"], ambiguous.Flows["second"][0])
		request := resultRequest("second", nextRoute("third"),
			condition("second_completed", flowidl.OutputType_path, "second.md"),
			condition("second_evidence", flowidl.OutputType_path, "evidence.md"),
		)
		_, failure := evaluateResult(ambiguous, reviewState, request)
		if failure == nil || failure.Code != erroridl.ErrorCode_ROUTE_AMBIGUOUS {
			t.Fatalf("failure = %#v", failure)
		}
	})

	t.Run("multiple Loop Routes for one target are rejected", func(t *testing.T) {
		ambiguous := loaded.Workflow
		ambiguous.Loops = cloneLoopRoutes(loaded.Workflow.Loops)
		routes := ambiguous.Loops["second"]
		ambiguous.Loops["second"] = append(routes, routes[len(routes)-1])
		request := resultRequest("second", backRoute("first"),
			condition("second_rework", flowidl.OutputType_enum_value, "rework"),
		)
		_, failure := evaluateResult(ambiguous, reviewState, request)
		if failure == nil || failure.Code != erroridl.ErrorCode_ROUTE_AMBIGUOUS {
			t.Fatalf("failure = %#v", failure)
		}
	})
}

func TestEvaluateCommonStepControls(t *testing.T) {
	loaded, err := loadRuntimeControls()
	if err != nil {
		t.Fatal(err)
	}
	first := "first"
	human := []*flowidl.Evidence{{Source: flowidl.EvidenceSource_human, Content: "confirmed"}}

	start := resultRequest(first, &flowidl.RouteSelection{StartCurrentStep: boolPointer(true)}, condition("start_confirmed", flowidl.OutputType_enum_value, "confirmed"))
	start.Evidence = human
	evaluation, failure := evaluateResult(loaded.Workflow, state.State{CurrentStepStatus: state.StepAwaitingConfirmation, Outputs: map[string]state.RegisteredOutput{}}, start)
	if failure != nil || evaluation.effect != flowidl.ResultEffect_started || len(evaluation.accepted) != 0 || evaluation.transition.GetToStepId() != first {
		t.Fatalf("start evaluation = %#v, failure = %#v", evaluation, failure)
	}

	target := "third"
	jump := resultRequest(first, &flowidl.RouteSelection{JumpStepId: &target}, condition("jump_requested", flowidl.OutputType_string, target))
	jump.Evidence = human
	evaluation, failure = evaluateResult(loaded.Workflow, state.State{CurrentStepStatus: state.StepInProgress, Outputs: map[string]state.RegisteredOutput{}}, jump)
	wantSkipped := []string{"first", "second"}
	if failure != nil || evaluation.effect != flowidl.ResultEffect_jumped {
		t.Fatalf("jump evaluation = %#v, failure = %#v", evaluation, failure)
	}
	jump.Evidence = nil
	if _, failure = evaluateResult(loaded.Workflow, state.State{CurrentStepStatus: state.StepInProgress, Outputs: map[string]state.RegisteredOutput{}}, jump); failure == nil || failure.Code != erroridl.ErrorCode_REPORT_NOT_ALLOWED {
		t.Fatalf("jump without human Evidence failure = %#v", failure)
	}
	jump.Evidence = human
	mismatch := "second"
	jump.Route.JumpStepId = &mismatch
	if _, failure = evaluateResult(loaded.Workflow, state.State{CurrentStepStatus: state.StepInProgress, Outputs: map[string]state.RegisteredOutput{}}, jump); failure == nil || failure.Code != erroridl.ErrorCode_OUTPUT_INVALID {
		t.Fatalf("jump target mismatch failure = %#v", failure)
	}
	jump.Route.JumpStepId = &target

	backTarget := "first"
	back := resultRequest("third", &flowidl.RouteSelection{JumpStepId: &backTarget}, condition("jump_requested", flowidl.OutputType_string, backTarget))
	back.Evidence = human
	evaluation, failure = evaluateResult(loaded.Workflow, state.State{CurrentStepStatus: state.StepInProgress, SkippedStepIDs: wantSkipped, Outputs: map[string]state.RegisteredOutput{
		"first_artifact":  {Type: workflow.OutputPath, Value: json.RawMessage(`"first.md"`), ProducerStepID: first},
		"second_artifact": {Type: workflow.OutputPath, Value: json.RawMessage(`"second.md"`), ProducerStepID: "second"},
	}}, back)
	if failure != nil || !sameStrings(evaluation.invalidated, []string{"first_artifact", "second_artifact"}) {
		t.Fatalf("backward jump evaluation = %#v, failure = %#v", evaluation, failure)
	}
	currentTarget := "third"
	currentJump := resultRequest(currentTarget, &flowidl.RouteSelection{JumpStepId: &currentTarget}, condition("jump_requested", flowidl.OutputType_string, currentTarget))
	currentJump.Evidence = human
	evaluation, failure = evaluateResult(loaded.Workflow, state.State{CurrentStepStatus: state.StepBlocked, SkippedStepIDs: wantSkipped, Outputs: map[string]state.RegisteredOutput{}}, currentJump)
	if failure != nil {
		t.Fatalf("current jump evaluation = %#v, failure = %#v", evaluation, failure)
	}

	mixed := resultRequest(first, &flowidl.RouteSelection{JumpStepId: &target}, condition("first_completed", flowidl.OutputType_path, "first.md"))
	mixed.Evidence = human
	if _, failure = evaluateResult(loaded.Workflow, state.State{Outputs: map[string]state.RegisteredOutput{}}, mixed); failure == nil || failure.Code != erroridl.ErrorCode_CONDITION_NOT_ALLOWED {
		t.Fatalf("mixed common/business failure = %#v", failure)
	}
}

func TestEvaluateResultRequiresPassedVerifyManifest(t *testing.T) {
	loaded, err := loadRuntimeControls()
	if err != nil {
		t.Fatal(err)
	}
	current := state.State{Outputs: map[string]state.RegisteredOutput{}}
	definition := loaded.Workflow
	verified := definition.Conditions["third_completed"]
	verified.Output.Type = workflow.OutputEnum
	verified.Output.Values = []string{"passed"}
	verified.Output.Source = workflow.OutputSourceVerifyEvidenceOutcome
	definition.Conditions["third_completed"] = verified
	request := resultRequest("third", terminalRoute(), condition("third_completed", flowidl.OutputType_enum_value, "passed"))
	assertVerifySourceMismatch(t, definition, current, request)

	failedPath := writeVerifyManifest(t, "failed")
	request.Evidence = fileEvidence(failedPath)
	assertVerifySourceMismatch(t, definition, current, request)

	passedPath := writeVerifyManifest(t, "passed")
	request.Evidence = fileEvidence(passedPath)
	evaluation, failure := evaluateResult(definition, current, request)
	if failure != nil || evaluation.effect != flowidl.ResultEffect_completed || evaluation.transition == nil || evaluation.transition.Direction != flowidl.TransitionDirection_flow || evaluation.transition.ToStepId != nil {
		t.Fatalf("evaluation = %#v, failure = %#v", evaluation, failure)
	}
}

func assertVerifySourceMismatch(t *testing.T, definition workflow.Workflow, current state.State, request *flowidl.FlowResultRequest) {
	t.Helper()
	_, failure := evaluateResult(definition, current, request)
	if failure == nil || failure.Code != erroridl.ErrorCode_OUTPUT_INVALID || failure.Message != "Condition Output value does not match its authoritative source" || failure.Details["condition_id"] != "third_completed" || failure.Details["source"] != workflow.OutputSourceVerifyEvidenceOutcome {
		t.Fatalf("failure = %#v", failure)
	}
}

func writeVerifyManifest(t *testing.T, outcome string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "manifest.json")
	content, err := json.Marshal(map[string]any{"schema_version": 1, "outcome": outcome, "run_id": "test"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(content, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func fileEvidence(path string) []*flowidl.Evidence {
	return []*flowidl.Evidence{{Source: flowidl.EvidenceSource_file, Content: "fanloop verify manifest", Ref: &path}}
}

func cloneFlowRoutes(source map[string][]workflow.FlowRoute) map[string][]workflow.FlowRoute {
	result := make(map[string][]workflow.FlowRoute, len(source))
	for key, routes := range source {
		result[key] = append([]workflow.FlowRoute(nil), routes...)
	}
	return result
}

func cloneLoopRoutes(source map[string][]workflow.LoopRoute) map[string][]workflow.LoopRoute {
	result := make(map[string][]workflow.LoopRoute, len(source))
	for key, routes := range source {
		result[key] = append([]workflow.LoopRoute(nil), routes...)
	}
	return result
}

func resultRequest(stepID string, route *flowidl.RouteSelection, results ...*flowidl.ConditionResult) *flowidl.FlowResultRequest {
	return &flowidl.FlowResultRequest{StepId: stepID, ConditionResults: results, Evidence: []*flowidl.Evidence{}, Summary: "reported", Route: route}
}

func nextRoute(stepID string) *flowidl.RouteSelection {
	return &flowidl.RouteSelection{NextStepId: &stepID}
}

func terminalRoute() *flowidl.RouteSelection {
	return &flowidl.RouteSelection{Terminal: boolPointer(true)}
}

func backRoute(stepID string) *flowidl.RouteSelection {
	return &flowidl.RouteSelection{BackStepId: &stepID}
}

func condition(id string, outputType flowidl.OutputType, value string) *flowidl.ConditionResult {
	return &flowidl.ConditionResult{ConditionId: id, Output: &flowidl.OutputValue{
		Type: outputType, Value: &commonidl.JsonValue{StringValue: &value},
	}}
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func boolPointer(value bool) *bool { return &value }

func loadRuntimeControls() (workflow.Loaded, error) {
	return workflow.LoadFrom(filepath.Join("..", "workflow", "testdata"), "runtime-controls")
}
