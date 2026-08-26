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

func TestEvaluateResultRoutesPromotionReviewAndInvalidatesByProducer(t *testing.T) {
	loaded, err := workflow.Load("promotion-design")
	if err != nil {
		t.Fatal(err)
	}
	reviewState := state.State{Outputs: map[string]state.RegisteredOutput{
		"require_points_path": {
			Type: workflow.OutputPath, Value: json.RawMessage(`"require_points.md"`), ProducerStepID: "clarify_requirements",
		},
		"promotion_design_path": {
			Type: workflow.OutputPath, Value: json.RawMessage(`"方案.md"`), ProducerStepID: "write_promotion_design",
		},
	}}

	t.Run("review failure loops to design", func(t *testing.T) {
		request := resultRequest("review_promotion_design", backRoute("write_promotion_design"),
			condition("design_review_failed", flowidl.OutputType_enum_value, "failed"),
			condition("design_review_written", flowidl.OutputType_path, ".promotion/review.md"),
		)
		evaluation, failure := evaluateResult(loaded.Workflow, reviewState, request)
		if failure != nil {
			t.Fatal(failure)
		}
		want := []string{"design_review_path", "design_review_result", "promotion_design_path"}
		if evaluation.effect != flowidl.ResultEffect_looped || evaluation.transition.GetToStepId() != "write_promotion_design" || !sameStrings(evaluation.invalidated, want) {
			t.Fatalf("evaluation = %#v", evaluation)
		}
	})

	t.Run("exclusive conditions conflict", func(t *testing.T) {
		request := resultRequest("review_promotion_design", nextRoute("confirm_promotion_design"),
			condition("design_review_passed", flowidl.OutputType_enum_value, "passed"),
			condition("design_review_failed", flowidl.OutputType_enum_value, "failed"),
			condition("design_review_written", flowidl.OutputType_path, ".promotion/review.md"),
		)
		_, failure := evaluateResult(loaded.Workflow, reviewState, request)
		if failure == nil || failure.Code != erroridl.ErrorCode_CONDITION_CONFLICT {
			t.Fatalf("failure = %#v", failure)
		}
	})

	t.Run("route is required", func(t *testing.T) {
		request := resultRequest("review_promotion_design", nil,
			condition("design_review_failed", flowidl.OutputType_enum_value, "failed"),
		)
		failure := validateResultRequest(request)
		if failure == nil || failure.Code != erroridl.ErrorCode_INVALID_ARGUMENT {
			t.Fatalf("failure = %#v", failure)
		}
	})

	t.Run("review verdict without report matches no route", func(t *testing.T) {
		request := resultRequest("review_promotion_design", nextRoute("confirm_promotion_design"),
			condition("design_review_passed", flowidl.OutputType_enum_value, "passed"),
		)
		_, failure := evaluateResult(loaded.Workflow, reviewState, request)
		if failure == nil || failure.Code != erroridl.ErrorCode_ROUTE_NOT_MATCHED {
			t.Fatalf("failure = %#v", failure)
		}
	})

	t.Run("Flow conditions do not match selected Loop Route", func(t *testing.T) {
		request := resultRequest("review_promotion_design", backRoute("write_promotion_design"),
			condition("design_review_passed", flowidl.OutputType_enum_value, "passed"),
			condition("design_review_written", flowidl.OutputType_path, ".promotion/review.md"),
		)
		_, failure := evaluateResult(loaded.Workflow, reviewState, request)
		if failure == nil || failure.Code != erroridl.ErrorCode_ROUTE_NOT_MATCHED {
			t.Fatalf("failure = %#v", failure)
		}
	})

	t.Run("review failure cannot advance", func(t *testing.T) {
		request := resultRequest("review_promotion_design", nextRoute("confirm_promotion_design"),
			condition("design_review_failed", flowidl.OutputType_enum_value, "failed"),
			condition("design_review_written", flowidl.OutputType_path, ".promotion/review.md"),
		)
		_, failure := evaluateResult(loaded.Workflow, reviewState, request)
		if failure == nil || failure.Code != erroridl.ErrorCode_ROUTE_NOT_MATCHED {
			t.Fatalf("failure = %#v", failure)
		}
	})

	t.Run("unknown target is not allowed", func(t *testing.T) {
		request := resultRequest("review_promotion_design", nextRoute("missing_step"),
			condition("design_review_passed", flowidl.OutputType_enum_value, "passed"),
			condition("design_review_written", flowidl.OutputType_path, ".promotion/review.md"),
		)
		_, failure := evaluateResult(loaded.Workflow, reviewState, request)
		if failure == nil || failure.Code != erroridl.ErrorCode_ROUTE_NOT_ALLOWED {
			t.Fatalf("failure = %#v", failure)
		}
	})

	t.Run("multiple Flow Routes are rejected", func(t *testing.T) {
		ambiguous := loaded.Workflow
		ambiguous.Flows = cloneFlowRoutes(loaded.Workflow.Flows)
		ambiguous.Flows["review_promotion_design"] = append(ambiguous.Flows["review_promotion_design"], ambiguous.Flows["review_promotion_design"][0])
		request := resultRequest("review_promotion_design", nextRoute("confirm_promotion_design"),
			condition("design_review_passed", flowidl.OutputType_enum_value, "passed"),
			condition("design_review_written", flowidl.OutputType_path, ".promotion/review.md"),
		)
		_, failure := evaluateResult(ambiguous, reviewState, request)
		if failure == nil || failure.Code != erroridl.ErrorCode_ROUTE_AMBIGUOUS {
			t.Fatalf("failure = %#v", failure)
		}
	})

	t.Run("multiple Loop Routes for one target are rejected", func(t *testing.T) {
		ambiguous := loaded.Workflow
		ambiguous.Loops = cloneLoopRoutes(loaded.Workflow.Loops)
		ambiguous.Loops["review_promotion_design"] = append(ambiguous.Loops["review_promotion_design"], ambiguous.Loops["review_promotion_design"][1])
		request := resultRequest("review_promotion_design", backRoute("write_promotion_design"),
			condition("design_review_failed", flowidl.OutputType_enum_value, "failed"),
			condition("design_review_written", flowidl.OutputType_path, ".promotion/review.md"),
		)
		_, failure := evaluateResult(ambiguous, reviewState, request)
		if failure == nil || failure.Code != erroridl.ErrorCode_ROUTE_AMBIGUOUS {
			t.Fatalf("failure = %#v", failure)
		}
	})
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
