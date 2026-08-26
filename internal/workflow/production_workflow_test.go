package workflow

import (
	"errors"
	"io/fs"
	"reflect"
	"testing"
)

func TestProductionBuildContainsOnlyFanloopWorkflows(t *testing.T) {
	for _, retired := range []string{"fanloop", "douyin-game", "fanloop-dev"} {
		if _, err := Load(retired); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("retired Workflow %q is loadable: %v", retired, err)
		}
	}
}

func TestProductionPromotionDesignWorkflow(t *testing.T) {
	loaded, err := Load("promotion-design")
	if err != nil {
		t.Fatal(err)
	}
	wantSteps := []Step{
		{ID: "clarify_requirements", Name: "需求澄清", Executor: StepExecutorAgent},
		{ID: "confirm_requirements", Name: "需求人工确认", Executor: StepExecutorHuman},
		{ID: "write_promotion_design", Name: "晋升方案写作", Executor: StepExecutorAgent},
		{ID: "review_promotion_design", Name: "陌生评委审校", Executor: StepExecutorAgent},
		{ID: "confirm_promotion_design", Name: "方案人工确认", Executor: StepExecutorHuman},
	}
	wantIDs := make([]string, 0, len(wantSteps))
	for _, want := range wantSteps {
		wantIDs = append(wantIDs, want.ID)
		context, _, ok := loaded.Workflow.FindStep(want.ID)
		if !ok || context.Step != want {
			t.Fatalf("Step %q = %#v, want %#v", want.ID, context.Step, want)
		}
	}
	if got := loaded.Workflow.OrderedStepIDs(); !reflect.DeepEqual(got, wantIDs) {
		t.Fatalf("Steps = %v, want %v", got, wantIDs)
	}
	for _, promptID := range []string{"clarify_requirements_flow", "write_promotion_design_flow", "review_promotion_design_flow"} {
		skilled := loaded.Workflow.Prompts[promptID].Skills
		if len(skilled) != 1 || skilled[0].ID != "promotion-design" || skilled[0].Optional == nil || *skilled[0].Optional {
			t.Fatalf("Prompt %s Skills = %#v", promptID, skilled)
		}
	}
	assertWorkflowRoute(t, loaded, "clarify_requirements", []string{"requirements_ready"}, "confirm_requirements", false)
	assertWorkflowRoute(t, loaded, "confirm_requirements", []string{"requirements_approved"}, "write_promotion_design", false)
	assertWorkflowRoute(t, loaded, "write_promotion_design", []string{"promotion_design_written"}, "review_promotion_design", false)
	assertWorkflowRoute(t, loaded, "review_promotion_design", []string{"design_review_passed", "design_review_written"}, "confirm_promotion_design", false)
	assertWorkflowRoute(t, loaded, "confirm_promotion_design", []string{"promotion_design_approved"}, "", true)
	assertWorkflowLoop(t, loaded, "confirm_requirements", []string{"requirements_rejected"}, "clarify_requirements")
	assertWorkflowLoop(t, loaded, "review_promotion_design", []string{"design_review_failed", "design_review_written"}, "write_promotion_design")
	assertWorkflowLoop(t, loaded, "confirm_promotion_design", []string{"promotion_design_rejected"}, "write_promotion_design")
	if got := len(loaded.Workflow.Conditions); got != 11 {
		t.Fatalf("Condition count = %d, want 11", got)
	}
}

func TestProductionMaintainerWorkflowUsesRenamedSelfIterationSkills(t *testing.T) {
	loaded, err := Load("fanloop-maintainer")
	if err != nil {
		t.Fatal(err)
	}
	wantSteps := []string{
		"bootstrap_techdesign", "clarify_requirements", "confirm_requirements", "design_technical_solution",
		"implement_code", "execute_test_cases", "review_code", "handoff_merge_request",
	}
	if got := loaded.Workflow.OrderedStepIDs(); !reflect.DeepEqual(got, wantSteps) {
		t.Fatalf("maintainer Steps = %v, want %v", got, wantSteps)
	}
	for _, skillID := range []string{
		"fanloop-dev-bootstrap", "fanloop-dev-grill-with-docs", "fanloop-dev-grilling",
		"fanloop-dev-domain-modeling", "fanloop-dev-to-spec", "fanloop-dev-to-tickets",
		"fanloop-dev-implement", "fanloop-dev-tdd", "fanloop-dev-verify",
		"fanloop-dev-code-review", "fanloop-dev-mr-handoff",
	} {
		found := false
		for _, prompt := range loaded.Workflow.Prompts {
			for _, binding := range prompt.Skills {
				found = found || binding.ID == skillID
			}
		}
		if !found {
			t.Fatalf("maintainer Workflow does not bind %s", skillID)
		}
	}
	assertWorkflowRoute(t, loaded, "implement_code", []string{"implementation_completed"}, "execute_test_cases", false)
	assertWorkflowRoute(t, loaded, "review_code", []string{"review_passed", "review_report_written"}, "handoff_merge_request", false)
	assertWorkflowRoute(t, loaded, "handoff_merge_request", []string{"merge_request_created", "merge_request_handed_off", "handoff_record_written"}, "", true)
	assertWorkflowLoop(t, loaded, "execute_test_cases", []string{"local_validation_failed"}, "implement_code")
	assertWorkflowLoop(t, loaded, "review_code", []string{"review_failed"}, "implement_code")
}

func assertWorkflowRoute(t *testing.T, loaded Loaded, stepID string, conditions []string, nextStepID string, terminal bool) {
	t.Helper()
	for _, route := range loaded.Workflow.Flows[stepID] {
		if reflect.DeepEqual(route.When.AnyOf, [][]string{conditions}) && route.NextStepID == nextStepID && route.Terminal == terminal {
			return
		}
	}
	t.Fatalf("%s has no expected Flow for %v: %#v", stepID, conditions, loaded.Workflow.Flows[stepID])
}

func assertWorkflowLoop(t *testing.T, loaded Loaded, stepID string, conditions []string, backStepID string) {
	t.Helper()
	for _, route := range loaded.Workflow.Loops[stepID] {
		if route.BackStepID != backStepID {
			continue
		}
		for _, group := range route.When.AnyOf {
			if reflect.DeepEqual(group, conditions) {
				return
			}
		}
	}
	t.Fatalf("%s has no expected Loop for %v -> %s: %#v", stepID, conditions, backStepID, loaded.Workflow.Loops[stepID])
}
