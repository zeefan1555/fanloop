package flow

import (
	"encoding/json"
	"testing"

	"github.com/zeefan1555/fanloop/internal/idl/commonidl"
	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/idl/flowidl"
	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

func TestEvaluateResultRoutesTechnicalReviewAndInvalidatesByProducer(t *testing.T) {
	loaded, err := workflow.Load("technical-solution-design")
	if err != nil {
		t.Fatal(err)
	}
	reviewState := state.State{Outputs: map[string]state.RegisteredOutput{
		"background_section_path": {
			Type: workflow.OutputPath, Value: json.RawMessage(`".technical-solution/sections/01-business-background.md"`), ProducerStepID: "frame_requirement_background",
		},
		"technical_solution_path": {
			Type: workflow.OutputPath, Value: json.RawMessage(`"technical-solution.md"`), ProducerStepID: "write_technical_solution",
		},
	}}

	t.Run("presentation finding loops to writing", func(t *testing.T) {
		request := resultRequest("review_technical_solution", backRoute("write_technical_solution"),
			condition("technical_solution_review_written", flowidl.OutputType_path, ".technical-solution/review.md"),
			condition("presentation_changed", flowidl.OutputType_enum_value, "presentation"),
		)
		evaluation, failure := evaluateResult(loaded.Workflow, reviewState, request)
		if failure != nil {
			t.Fatal(failure)
		}
		want := []string{"technical_feedback_scope", "technical_solution_path", "technical_solution_review_path"}
		if evaluation.effect != flowidl.ResultEffect_looped || evaluation.transition.GetToStepId() != "write_technical_solution" || !sameStrings(evaluation.invalidated, want) {
			t.Fatalf("evaluation = %#v", evaluation)
		}
	})

	t.Run("exclusive conditions conflict", func(t *testing.T) {
		request := resultRequest("review_technical_solution", nextRoute("confirm_technical_solution"),
			condition("technical_solution_review_passed", flowidl.OutputType_enum_value, "passed"),
			condition("presentation_changed", flowidl.OutputType_enum_value, "presentation"),
			condition("technical_solution_review_written", flowidl.OutputType_path, ".technical-solution/review.md"),
		)
		_, failure := evaluateResult(loaded.Workflow, reviewState, request)
		if failure == nil || failure.Code != erroridl.ErrorCode_CONDITION_CONFLICT {
			t.Fatalf("failure = %#v", failure)
		}
	})

	t.Run("route is required", func(t *testing.T) {
		request := resultRequest("review_technical_solution", nil,
			condition("presentation_changed", flowidl.OutputType_enum_value, "presentation"),
		)
		failure := validateResultRequest(request)
		if failure == nil || failure.Code != erroridl.ErrorCode_INVALID_ARGUMENT {
			t.Fatalf("failure = %#v", failure)
		}
	})

	t.Run("review verdict without report matches no route", func(t *testing.T) {
		request := resultRequest("review_technical_solution", nextRoute("confirm_technical_solution"),
			condition("technical_solution_review_passed", flowidl.OutputType_enum_value, "passed"),
		)
		_, failure := evaluateResult(loaded.Workflow, reviewState, request)
		if failure == nil || failure.Code != erroridl.ErrorCode_ROUTE_NOT_MATCHED {
			t.Fatalf("failure = %#v", failure)
		}
	})

	t.Run("Flow conditions do not match selected Loop Route", func(t *testing.T) {
		request := resultRequest("review_technical_solution", backRoute("write_technical_solution"),
			condition("technical_solution_review_passed", flowidl.OutputType_enum_value, "passed"),
			condition("technical_solution_review_written", flowidl.OutputType_path, ".technical-solution/review.md"),
		)
		_, failure := evaluateResult(loaded.Workflow, reviewState, request)
		if failure == nil || failure.Code != erroridl.ErrorCode_ROUTE_NOT_MATCHED {
			t.Fatalf("failure = %#v", failure)
		}
	})

	t.Run("presentation finding cannot advance", func(t *testing.T) {
		request := resultRequest("review_technical_solution", nextRoute("confirm_technical_solution"),
			condition("technical_solution_review_written", flowidl.OutputType_path, ".technical-solution/review.md"),
			condition("presentation_changed", flowidl.OutputType_enum_value, "presentation"),
		)
		_, failure := evaluateResult(loaded.Workflow, reviewState, request)
		if failure == nil || failure.Code != erroridl.ErrorCode_ROUTE_NOT_MATCHED {
			t.Fatalf("failure = %#v", failure)
		}
	})

	t.Run("unknown target is not allowed", func(t *testing.T) {
		request := resultRequest("review_technical_solution", nextRoute("missing_step"),
			condition("technical_solution_review_passed", flowidl.OutputType_enum_value, "passed"),
			condition("technical_solution_review_written", flowidl.OutputType_path, ".technical-solution/review.md"),
		)
		_, failure := evaluateResult(loaded.Workflow, reviewState, request)
		if failure == nil || failure.Code != erroridl.ErrorCode_ROUTE_NOT_ALLOWED {
			t.Fatalf("failure = %#v", failure)
		}
	})

	t.Run("multiple Flow Routes are rejected", func(t *testing.T) {
		ambiguous := loaded.Workflow
		ambiguous.Flows = cloneFlowRoutes(loaded.Workflow.Flows)
		ambiguous.Flows["review_technical_solution"] = append(ambiguous.Flows["review_technical_solution"], ambiguous.Flows["review_technical_solution"][0])
		request := resultRequest("review_technical_solution", nextRoute("confirm_technical_solution"),
			condition("technical_solution_review_passed", flowidl.OutputType_enum_value, "passed"),
			condition("technical_solution_review_written", flowidl.OutputType_path, ".technical-solution/review.md"),
		)
		_, failure := evaluateResult(ambiguous, reviewState, request)
		if failure == nil || failure.Code != erroridl.ErrorCode_ROUTE_AMBIGUOUS {
			t.Fatalf("failure = %#v", failure)
		}
	})

	t.Run("multiple Loop Routes for one target are rejected", func(t *testing.T) {
		ambiguous := loaded.Workflow
		ambiguous.Loops = cloneLoopRoutes(loaded.Workflow.Loops)
		routes := ambiguous.Loops["review_technical_solution"]
		ambiguous.Loops["review_technical_solution"] = append(routes, routes[len(routes)-1])
		request := resultRequest("review_technical_solution", backRoute("write_technical_solution"),
			condition("technical_solution_review_written", flowidl.OutputType_path, ".technical-solution/review.md"),
			condition("presentation_changed", flowidl.OutputType_enum_value, "presentation"),
		)
		_, failure := evaluateResult(ambiguous, reviewState, request)
		if failure == nil || failure.Code != erroridl.ErrorCode_ROUTE_AMBIGUOUS {
			t.Fatalf("failure = %#v", failure)
		}
	})
}

func TestEvaluateCommonStepControls(t *testing.T) {
	loaded, err := workflow.Load("technical-solution-design")
	if err != nil {
		t.Fatal(err)
	}
	first := "frame_requirement_background"
	human := []*flowidl.Evidence{{Source: flowidl.EvidenceSource_human, Content: "confirmed"}}

	start := resultRequest(first, &flowidl.RouteSelection{StartCurrentStep: boolPointer(true)}, condition("step_scope_confirmed", flowidl.OutputType_enum_value, "confirmed"))
	start.Evidence = human
	evaluation, failure := evaluateResult(loaded.Workflow, state.State{CurrentStepStatus: state.StepAwaitingConfirmation, Outputs: map[string]state.RegisteredOutput{}}, start)
	if failure != nil || evaluation.effect != flowidl.ResultEffect_started || len(evaluation.accepted) != 0 || evaluation.transition.GetToStepId() != first {
		t.Fatalf("start evaluation = %#v, failure = %#v", evaluation, failure)
	}

	target := "research_solution_options"
	jump := resultRequest(first, &flowidl.RouteSelection{JumpStepId: &target}, condition("human_step_jump_requested", flowidl.OutputType_string, target))
	jump.Evidence = human
	evaluation, failure = evaluateResult(loaded.Workflow, state.State{CurrentStepStatus: state.StepInProgress, Outputs: map[string]state.RegisteredOutput{}}, jump)
	wantSkipped := []string{"frame_requirement_background", "define_goals_and_problems", "define_business_constraints", "confirm_technical_problem"}
	if failure != nil || evaluation.effect != flowidl.ResultEffect_jumped || !sameStrings(evaluation.skippedStepIDs, wantSkipped) || !evaluation.updateSkipped {
		t.Fatalf("jump evaluation = %#v, failure = %#v", evaluation, failure)
	}
	jump.Evidence = nil
	if _, failure = evaluateResult(loaded.Workflow, state.State{CurrentStepStatus: state.StepInProgress, Outputs: map[string]state.RegisteredOutput{}}, jump); failure == nil || failure.Code != erroridl.ErrorCode_REPORT_NOT_ALLOWED {
		t.Fatalf("jump without human Evidence failure = %#v", failure)
	}
	jump.Evidence = human
	mismatch := "design_overall_solution"
	jump.Route.JumpStepId = &mismatch
	if _, failure = evaluateResult(loaded.Workflow, state.State{CurrentStepStatus: state.StepInProgress, Outputs: map[string]state.RegisteredOutput{}}, jump); failure == nil || failure.Code != erroridl.ErrorCode_OUTPUT_INVALID {
		t.Fatalf("jump target mismatch failure = %#v", failure)
	}
	jump.Route.JumpStepId = &target

	backTarget := "define_goals_and_problems"
	back := resultRequest("design_overall_solution", &flowidl.RouteSelection{JumpStepId: &backTarget}, condition("human_step_jump_requested", flowidl.OutputType_string, backTarget))
	back.Evidence = human
	evaluation, failure = evaluateResult(loaded.Workflow, state.State{CurrentStepStatus: state.StepInProgress, SkippedStepIDs: wantSkipped, Outputs: map[string]state.RegisteredOutput{
		"background_section_path": {Type: workflow.OutputPath, Value: json.RawMessage(`"background.md"`), ProducerStepID: first},
		"research_section_path":   {Type: workflow.OutputPath, Value: json.RawMessage(`"research.md"`), ProducerStepID: "research_solution_options"},
	}}, back)
	if failure != nil || !sameStrings(evaluation.invalidated, []string{"research_section_path"}) || len(evaluation.skippedStepIDs) != 1 || evaluation.skippedStepIDs[0] != first {
		t.Fatalf("backward jump evaluation = %#v, failure = %#v", evaluation, failure)
	}
	currentTarget := "design_overall_solution"
	currentJump := resultRequest(currentTarget, &flowidl.RouteSelection{JumpStepId: &currentTarget}, condition("human_step_jump_requested", flowidl.OutputType_string, currentTarget))
	currentJump.Evidence = human
	evaluation, failure = evaluateResult(loaded.Workflow, state.State{CurrentStepStatus: state.StepBlocked, SkippedStepIDs: wantSkipped, Outputs: map[string]state.RegisteredOutput{}}, currentJump)
	if failure != nil || !sameStrings(evaluation.skippedStepIDs, wantSkipped) {
		t.Fatalf("current jump evaluation = %#v, failure = %#v", evaluation, failure)
	}

	mixed := resultRequest(first, &flowidl.RouteSelection{JumpStepId: &target}, condition("background_defined", flowidl.OutputType_path, "background.md"))
	mixed.Evidence = human
	if _, failure = evaluateResult(loaded.Workflow, state.State{Outputs: map[string]state.RegisteredOutput{}}, mixed); failure == nil || failure.Code != erroridl.ErrorCode_CONDITION_NOT_ALLOWED {
		t.Fatalf("mixed common/business failure = %#v", failure)
	}
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
