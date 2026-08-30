package workflow

import (
	"errors"
	"io/fs"
	"reflect"
	"testing"
)

func TestProductionBuildContainsOnlyFanloopWorkflows(t *testing.T) {
	for _, retired := range []string{"fanloop", "douyin-game", "fanloop-dev", "promotion-design"} {
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
		{ID: "frame_requirement_background", Name: "需求背景", Executor: StepExecutorAgent},
		{ID: "analyze_core_problem", Name: "核心问题", Executor: StepExecutorAgent},
		{ID: "define_design_objectives", Name: "设计目标", Executor: StepExecutorAgent},
		{ID: "confirm_technical_problem", Name: "问题审核", Executor: StepExecutorHuman},
		{ID: "research_solution_options", Name: "方案调研", Executor: StepExecutorAgent},
		{ID: "design_overall_solution", Name: "总体方案", Executor: StepExecutorAgent},
		{ID: "design_key_solutions", Name: "难点解法", Executor: StepExecutorAgent},
		{ID: "confirm_solution_direction", Name: "方案审核", Executor: StepExecutorHuman},
		{ID: "evaluate_solution_benefits", Name: "方案收益", Executor: StepExecutorAgent},
		{ID: "plan_solution_delivery", Name: "落地规划", Executor: StepExecutorAgent},
		{ID: "write_technical_solution", Name: "方案成文", Executor: StepExecutorAgent},
		{ID: "review_technical_solution", Name: "方案审校", Executor: StepExecutorAgent},
		{ID: "confirm_technical_solution", Name: "方案终审", Executor: StepExecutorHuman},
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
		"frame_requirement_background_flow": "technical-background-framing",
		"analyze_core_problem_flow":         "technical-problem-analysis",
		"define_design_objectives_flow":     "technical-objective-setting",
		"confirm_technical_problem_flow":    "technical-problem-approval",
		"research_solution_options_flow":    "technical-solution-research",
		"design_overall_solution_flow":      "technical-overall-solution",
		"design_key_solutions_flow":         "technical-key-solutions",
		"confirm_solution_direction_flow":   "technical-direction-approval",
		"evaluate_solution_benefits_flow":   "technical-solution-benefits",
		"plan_solution_delivery_flow":       "technical-solution-delivery",
		"write_technical_solution_flow":     "technical-solution-writing",
		"review_technical_solution_flow":    "technical-solution-review",
		"confirm_technical_solution_flow":   "technical-solution-approval",
	}
	for promptID, skillID := range wantSkills {
		skilled := loaded.Workflow.Prompts[promptID].Skills
		if len(skilled) != 1 || skilled[0].ID != skillID || skilled[0].Optional == nil || *skilled[0].Optional {
			t.Fatalf("Prompt %s Skills = %#v, want required %s", promptID, skilled, skillID)
		}
	}
	assertConditionSkill(t, loaded, "panorama_card_published", "technical-solution-panorama")
	if _, ok := loaded.Workflow.Condition("agent_approved"); ok {
		t.Fatal("technical-solution-design must require human approval")
	}
	assertWorkflowRoute(t, loaded, "frame_requirement_background", []string{"background_defined"}, "analyze_core_problem", false)
	assertWorkflowRoute(t, loaded, "analyze_core_problem", []string{"core_problem_defined"}, "define_design_objectives", false)
	assertWorkflowRoute(t, loaded, "define_design_objectives", []string{"design_objectives_defined"}, "confirm_technical_problem", false)
	assertWorkflowRoute(t, loaded, "confirm_technical_problem", []string{"problem_document_published", "panorama_card_published", "technical_problem_approved"}, "research_solution_options", false)
	assertWorkflowRoute(t, loaded, "research_solution_options", []string{"solution_research_completed"}, "design_overall_solution", false)
	assertWorkflowRoute(t, loaded, "design_overall_solution", []string{"overall_solution_designed", "architecture_diagram_written"}, "design_key_solutions", false)
	assertWorkflowRoute(t, loaded, "design_key_solutions", []string{"key_solutions_designed"}, "confirm_solution_direction", false)
	assertWorkflowRoute(t, loaded, "confirm_solution_direction", []string{"solution_document_published", "panorama_card_published", "solution_direction_approved"}, "evaluate_solution_benefits", false)
	assertWorkflowRoute(t, loaded, "evaluate_solution_benefits", []string{"solution_benefits_defined"}, "plan_solution_delivery", false)
	assertWorkflowRoute(t, loaded, "plan_solution_delivery", []string{"delivery_plan_defined"}, "write_technical_solution", false)
	assertWorkflowRoute(t, loaded, "write_technical_solution", []string{"technical_solution_written"}, "review_technical_solution", false)
	assertWorkflowRoute(t, loaded, "review_technical_solution", []string{"technical_solution_review_passed", "technical_solution_review_written"}, "confirm_technical_solution", false)
	assertWorkflowRoute(t, loaded, "confirm_technical_solution", []string{"technical_solution_document_published", "panorama_card_published", "technical_solution_approved"}, "", true)

	feedback := []struct {
		condition string
		backStep  string
	}{
		{"background_changed", "frame_requirement_background"},
		{"problem_changed", "analyze_core_problem"},
		{"objectives_changed", "define_design_objectives"},
		{"research_changed", "research_solution_options"},
		{"overall_solution_changed", "design_overall_solution"},
		{"key_solutions_changed", "design_key_solutions"},
		{"benefits_changed", "evaluate_solution_benefits"},
		{"delivery_changed", "plan_solution_delivery"},
		{"presentation_changed", "write_technical_solution"},
	}
	for _, conditionID := range []string{
		"background_defined", "core_problem_defined", "design_objectives_defined", "technical_problem_approved",
		"solution_research_completed", "overall_solution_designed", "key_solutions_designed", "solution_direction_approved",
		"solution_benefits_defined", "delivery_plan_defined", "technical_solution_written",
		"technical_solution_review_passed", "technical_solution_approved",
	} {
		condition, _ := loaded.Workflow.Condition(conditionID)
		if condition.ExclusiveGroup != "technical_decision_outcome" {
			t.Fatalf("Condition %s exclusive_group = %q", conditionID, condition.ExclusiveGroup)
		}
	}
	for _, item := range feedback[:3] {
		assertWorkflowLoop(t, loaded, "confirm_technical_problem", []string{"problem_document_published", "panorama_card_published", item.condition}, item.backStep)
	}
	for _, item := range feedback[:6] {
		assertWorkflowLoop(t, loaded, "confirm_solution_direction", []string{"solution_document_published", "panorama_card_published", item.condition}, item.backStep)
	}
	for _, item := range feedback {
		condition, _ := loaded.Workflow.Condition(item.condition)
		if condition.ExclusiveGroup != "technical_decision_outcome" {
			t.Fatalf("Condition %s exclusive_group = %q", item.condition, condition.ExclusiveGroup)
		}
		assertWorkflowLoop(t, loaded, "review_technical_solution", []string{"technical_solution_review_written", item.condition}, item.backStep)
		assertWorkflowLoop(t, loaded, "confirm_technical_solution", []string{"technical_solution_document_published", "panorama_card_published", item.condition}, item.backStep)
	}
	if got := len(loaded.Workflow.Conditions); got != 28 {
		t.Fatalf("Condition count = %d, want 28", got)
	}
}

func TestProductionMaintainerWorkflowUsesRenamedSelfIterationSkills(t *testing.T) {
	loaded, err := Load("fanloop-maintainer")
	if err != nil {
		t.Fatal(err)
	}
	wantSteps := []string{
		"bootstrap_techdesign", "clarify_requirements", "confirm_requirements", "design_technical_solution",
		"implement_code", "execute_test_cases", "review_code", "execute_agent_acceptance",
		"confirm_human_acceptance", "handoff_merge_request",
	}
	if got := loaded.Workflow.OrderedStepIDs(); !reflect.DeepEqual(got, wantSteps) {
		t.Fatalf("maintainer Steps = %v, want %v", got, wantSteps)
	}
	for _, want := range []struct {
		step     string
		stage    string
		executor StepExecutor
	}{
		{step: "review_code", stage: "implementation", executor: StepExecutorAgent},
		{step: "execute_agent_acceptance", stage: "delivery", executor: StepExecutorAgent},
		{step: "confirm_human_acceptance", stage: "delivery", executor: StepExecutorHuman},
	} {
		context, _, ok := loaded.Workflow.FindStep(want.step)
		if !ok || context.Stage.ID != want.stage || context.Step.Executor != want.executor {
			t.Fatalf("maintainer Step %s = %#v, want stage=%s executor=%v", want.step, context, want.stage, want.executor)
		}
	}
	for _, skillID := range []string{
		"fanloop-dev-bootstrap", "fanloop-dev-grill-with-docs", "fanloop-dev-grilling",
		"fanloop-dev-domain-modeling", "fanloop-dev-to-spec", "fanloop-dev-to-tickets",
		"fanloop-dev-implement", "fanloop-dev-tdd", "fanloop-dev-verify",
		"fanloop-dev-code-review", "fanloop-dev-maintain-verification", "fanloop-dev-agent-acceptance",
		"fanloop-dev-mr-handoff", "fanloop-dev-panorama",
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
	assertConditionSkill(t, loaded, "panorama_card_published", "fanloop-dev-panorama")
	assertAgentApprovalCondition(t, loaded)
	assertWorkflowRoute(t, loaded, "implement_code", []string{"implementation_completed"}, "execute_test_cases", false)
	assertWorkflowRouteAnyOf(t, loaded, "confirm_requirements", [][]string{
		{"panorama_card_published", "requirements_approved", "requirements_approval_recorded", "requirements_evidence_written", "implementation_required"},
		{"agent_approved", "implementation_required"},
	}, "design_technical_solution", false)
	assertWorkflowRouteAnyOf(t, loaded, "confirm_requirements", [][]string{
		{"panorama_card_published", "requirements_approved", "requirements_approval_recorded", "requirements_evidence_written", "implementation_not_required"},
		{"agent_approved", "implementation_not_required"},
	}, "", true)
	assertWorkflowRoute(t, loaded, "review_code", []string{"review_passed", "review_report_written"}, "execute_agent_acceptance", false)
	assertWorkflowRoute(t, loaded, "execute_agent_acceptance", []string{"agent_acceptance_passed", "acceptance_report_written"}, "confirm_human_acceptance", false)
	assertWorkflowRoute(t, loaded, "confirm_human_acceptance", []string{"panorama_card_published", "human_acceptance_passed", "human_acceptance_result_recorded", "acceptance_report_written"}, "handoff_merge_request", false)
	assertWorkflowRoute(t, loaded, "confirm_human_acceptance", []string{"panorama_card_published", "human_acceptance_skipped", "human_acceptance_result_recorded", "acceptance_report_written"}, "handoff_merge_request", false)
	assertWorkflowRoute(t, loaded, "handoff_merge_request", []string{"merge_request_created", "merge_request_handed_off", "handoff_record_written", "acceptance_report_written"}, "", true)
	assertWorkflowLoop(t, loaded, "execute_test_cases", []string{"local_validation_failed"}, "implement_code")
	assertWorkflowLoop(t, loaded, "review_code", []string{"requirements_changed", "review_report_written"}, "clarify_requirements")
	assertWorkflowLoop(t, loaded, "review_code", []string{"technical_solution_changes_requested", "review_report_written"}, "design_technical_solution")
	assertWorkflowLoop(t, loaded, "review_code", []string{"review_failed", "review_report_written"}, "implement_code")
	assertWorkflowLoop(t, loaded, "execute_agent_acceptance", []string{"agent_acceptance_failed", "requirements_changed", "acceptance_report_written"}, "clarify_requirements")
	assertWorkflowLoop(t, loaded, "execute_agent_acceptance", []string{"agent_acceptance_failed", "technical_solution_changes_requested", "acceptance_report_written"}, "design_technical_solution")
	assertWorkflowLoop(t, loaded, "execute_agent_acceptance", []string{"agent_acceptance_failed", "implementation_changes_requested", "acceptance_report_written"}, "implement_code")
	assertWorkflowLoop(t, loaded, "confirm_human_acceptance", []string{"panorama_card_published", "requirements_changed", "human_acceptance_result_recorded", "acceptance_report_written"}, "clarify_requirements")
	assertWorkflowLoop(t, loaded, "confirm_human_acceptance", []string{"panorama_card_published", "implementation_changes_requested", "human_acceptance_result_recorded", "acceptance_report_written"}, "implement_code")
	assertWorkflowLoop(t, loaded, "confirm_human_acceptance", []string{"implementation_changes_requested", "acceptance_report_written"}, "implement_code")
	assertWorkflowLoop(t, loaded, "handoff_merge_request", []string{"implementation_changes_requested", "acceptance_report_written"}, "implement_code")
	assertWorkflowLoop(t, loaded, "handoff_merge_request", []string{"merge_request_handoff_failed", "acceptance_report_written"}, "handoff_merge_request")

	for _, route := range loaded.Workflow.Flows["confirm_human_acceptance"] {
		for _, group := range route.When.AnyOf {
			for _, conditionID := range group {
				if conditionID == "agent_approved" {
					t.Fatal("confirm_human_acceptance must not allow Agent approval")
				}
			}
		}
	}
	flowRoutes, loopRoutes := 0, 0
	for _, routes := range loaded.Workflow.Flows {
		flowRoutes += len(routes)
	}
	for _, routes := range loaded.Workflow.Loops {
		loopRoutes += len(routes)
	}
	if got := len(loaded.Workflow.Conditions); got != 37 {
		t.Fatalf("maintainer Conditions = %d, want 37", got)
	}
	if flowRoutes != 12 || loopRoutes != 20 || len(loaded.Workflow.Prompts) != 43 {
		t.Fatalf("maintainer route/prompt counts = flow:%d loop:%d prompts:%d, want 12/20/43", flowRoutes, loopRoutes, len(loaded.Workflow.Prompts))
	}
}

func assertAgentApprovalCondition(t *testing.T, loaded Loaded) {
	t.Helper()
	condition, ok := loaded.Workflow.Condition("agent_approved")
	if !ok || condition.Output.Key != "agent_approval_decision" || condition.Output.Type != OutputEnum ||
		!reflect.DeepEqual(condition.Output.Values, []string{"approved"}) {
		t.Fatalf("agent_approved = %#v", condition)
	}
}

func assertConditionSkill(t *testing.T, loaded Loaded, conditionID, skillID string) {
	t.Helper()
	condition, ok := loaded.Workflow.Condition(conditionID)
	if !ok {
		t.Fatalf("Condition %s is missing", conditionID)
	}
	if condition.Output.Key != "panorama_snapshot_path" || condition.Output.Type != OutputPath {
		t.Fatalf("Condition %s Output = %#v, want panorama_snapshot_path:path", conditionID, condition.Output)
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

func assertWorkflowRouteAnyOf(t *testing.T, loaded Loaded, stepID string, groups [][]string, nextStepID string, terminal bool) {
	t.Helper()
	for _, route := range loaded.Workflow.Flows[stepID] {
		if reflect.DeepEqual(route.When.AnyOf, groups) && route.NextStepID == nextStepID && route.Terminal == terminal {
			return
		}
	}
	t.Fatalf("%s has no expected Flow for %v: %#v", stepID, groups, loaded.Workflow.Flows[stepID])
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
