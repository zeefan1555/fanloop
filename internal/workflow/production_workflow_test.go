package workflow

import (
	"errors"
	"io/fs"
	"reflect"
	"testing"
)

func TestProductionBuildContainsOnlyCommonloopWorkflows(t *testing.T) {
	for _, retired := range []string{"commonloop", "douyin-game", "commonloop-dev", "promotion-design"} {
		if _, err := Load(retired); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("retired Workflow %q is loadable: %v", retired, err)
		}
	}
}

func TestProductionTechnicalSolutionDesignWorkflow(t *testing.T) {
	loaded, err := Load("technical-solution-design")
	if err != nil {
		t.Fatal(err)
	}
	wantSteps := []Step{
		{ID: "frame_technical_problem", Name: "技术问题定义", Executor: StepExecutorAgent},
		{ID: "confirm_technical_problem", Name: "问题人工确认", Executor: StepExecutorHuman},
		{ID: "derive_technical_solution", Name: "技术方案推导", Executor: StepExecutorAgent},
		{ID: "confirm_solution_direction", Name: "方案方向人工确认", Executor: StepExecutorHuman},
		{ID: "write_technical_solution", Name: "技术方案写作", Executor: StepExecutorAgent},
		{ID: "review_technical_solution", Name: "技术方案审校", Executor: StepExecutorAgent},
		{ID: "confirm_technical_solution", Name: "技术方案人工确认", Executor: StepExecutorHuman},
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
	wantSkills := map[string]string{
		"frame_technical_problem_flow":    "technical-problem-framing",
		"confirm_technical_problem_flow":  "technical-problem-approval",
		"derive_technical_solution_flow":  "technical-solution-derivation",
		"confirm_solution_direction_flow": "technical-direction-approval",
		"write_technical_solution_flow":   "technical-solution-writing",
		"review_technical_solution_flow":  "technical-solution-review",
		"confirm_technical_solution_flow": "technical-solution-approval",
	}
	for promptID, skillID := range wantSkills {
		skilled := loaded.Workflow.Prompts[promptID].Skills
		if len(skilled) != 1 || skilled[0].ID != skillID || skilled[0].Optional == nil || *skilled[0].Optional {
			t.Fatalf("Prompt %s Skills = %#v, want required %s", promptID, skilled, skillID)
		}
	}
	assertConditionSkill(t, loaded, "panorama_card_published", "technical-solution-panorama")
	assertWorkflowRoute(t, loaded, "frame_technical_problem", []string{"technical_problem_defined"}, "confirm_technical_problem", false)
	assertWorkflowRoute(t, loaded, "confirm_technical_problem", []string{"panorama_card_published", "technical_problem_approved"}, "derive_technical_solution", false)
	assertWorkflowRoute(t, loaded, "derive_technical_solution", []string{"technical_solution_derived"}, "confirm_solution_direction", false)
	assertWorkflowRoute(t, loaded, "confirm_solution_direction", []string{"panorama_card_published", "solution_direction_approved"}, "write_technical_solution", false)
	assertWorkflowRoute(t, loaded, "write_technical_solution", []string{"technical_solution_written", "architecture_diagram_written"}, "review_technical_solution", false)
	assertWorkflowRoute(t, loaded, "review_technical_solution", []string{"technical_solution_review_passed", "technical_solution_review_written"}, "confirm_technical_solution", false)
	assertWorkflowRoute(t, loaded, "confirm_technical_solution", []string{"panorama_card_published", "technical_solution_approved"}, "", true)
	assertWorkflowLoop(t, loaded, "frame_technical_problem", []string{"technical_problem_rework_requested"}, "frame_technical_problem")
	assertWorkflowLoop(t, loaded, "confirm_technical_problem", []string{"panorama_card_published", "technical_problem_rejected"}, "frame_technical_problem")
	assertWorkflowLoop(t, loaded, "derive_technical_solution", []string{"technical_problem_changed"}, "frame_technical_problem")
	assertWorkflowLoop(t, loaded, "confirm_solution_direction", []string{"panorama_card_published", "solution_direction_rejected"}, "derive_technical_solution")
	assertWorkflowLoop(t, loaded, "write_technical_solution", []string{"solution_direction_changed"}, "derive_technical_solution")
	assertWorkflowLoop(t, loaded, "review_technical_solution", []string{"technical_solution_review_failed", "technical_solution_review_written"}, "write_technical_solution")
	assertWorkflowLoop(t, loaded, "confirm_technical_solution", []string{"panorama_card_published", "technical_solution_rejected"}, "write_technical_solution")
	if got := len(loaded.Workflow.Conditions); got != 17 {
		t.Fatalf("Condition count = %d, want 17", got)
	}
}

func TestProductionMaintainerWorkflowUsesRenamedSelfIterationSkills(t *testing.T) {
	loaded, err := Load("commonloop-maintainer")
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
		"commonloop-dev-bootstrap", "commonloop-dev-grill-with-docs", "commonloop-dev-grilling",
		"commonloop-dev-domain-modeling", "commonloop-dev-to-spec", "commonloop-dev-to-tickets",
		"commonloop-dev-implement", "commonloop-dev-tdd", "commonloop-dev-verify",
		"commonloop-dev-code-review", "commonloop-dev-mr-handoff", "commonloop-dev-panorama",
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
	assertConditionSkill(t, loaded, "panorama_card_published", "commonloop-dev-panorama")
	assertWorkflowRoute(t, loaded, "implement_code", []string{"implementation_completed"}, "execute_test_cases", false)
	assertWorkflowRoute(t, loaded, "confirm_requirements", []string{"panorama_card_published", "requirements_approved", "requirements_approval_recorded", "requirements_evidence_written", "implementation_required"}, "design_technical_solution", false)
	assertWorkflowRoute(t, loaded, "review_code", []string{"review_passed", "review_report_written"}, "handoff_merge_request", false)
	assertWorkflowRoute(t, loaded, "handoff_merge_request", []string{"merge_request_created", "merge_request_handed_off", "handoff_record_written"}, "", true)
	assertWorkflowLoop(t, loaded, "execute_test_cases", []string{"local_validation_failed"}, "implement_code")
	assertWorkflowLoop(t, loaded, "review_code", []string{"review_failed"}, "implement_code")
}

func assertConditionSkill(t *testing.T, loaded Loaded, conditionID, skillID string) {
	t.Helper()
	condition, ok := loaded.Workflow.Condition(conditionID)
	if !ok {
		t.Fatalf("Condition %s is missing", conditionID)
	}
	prompt, ok := loaded.Workflow.Prompt(condition.PromptRef)
	if !ok || len(prompt.Skills) != 1 || prompt.Skills[0].ID != skillID ||
		prompt.Skills[0].Optional == nil || *prompt.Skills[0].Optional {
		t.Fatalf("Condition %s Skills = %#v, want required %s", conditionID, prompt.Skills, skillID)
	}
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
