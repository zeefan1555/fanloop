package workflow

import (
	"errors"
	"io/fs"
	"reflect"
	"strings"
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
	if got := []string{loaded.Workflow.Stages[0].Name, loaded.Workflow.Stages[1].Name, loaded.Workflow.Stages[2].Name}; !reflect.DeepEqual(got, []string{"业务问题", "技术判断", "结果与规划"}) {
		t.Fatalf("Stage names = %v", got)
	}
	wantSteps := []Step{
		{ID: "frame_requirement_background", Name: "业务背景", Executor: StepExecutorAgent},
		{ID: "define_goals_and_problems", Name: "目标与问题定义", Executor: StepExecutorAgent},
		{ID: "define_business_constraints", Name: "业务特点与技术约束", Executor: StepExecutorAgent},
		{ID: "confirm_technical_problem", Name: "问题与约束审核", Executor: StepExecutorHuman},
		{ID: "research_solution_options", Name: "业界/业内方案调研", Executor: StepExecutorAgent},
		{ID: "design_overall_solution", Name: "总体方案设计", Executor: StepExecutorAgent},
		{ID: "design_key_solutions", Name: "关键模块设计", Executor: StepExecutorAgent},
		{ID: "record_technical_decisions", Name: "核心技术决策与取舍", Executor: StepExecutorAgent},
		{ID: "confirm_solution_direction", Name: "方案与决策审核", Executor: StepExecutorHuman},
		{ID: "plan_solution_delivery", Name: "落地路径与风险控制", Executor: StepExecutorAgent},
		{ID: "evaluate_solution_benefits", Name: "结果收益", Executor: StepExecutorAgent},
		{ID: "write_retrospective_and_roadmap", Name: "复盘与后续规划", Executor: StepExecutorAgent},
		{ID: "write_summary", Name: "摘要", Executor: StepExecutorAgent},
		{ID: "write_technical_solution", Name: "文档组装", Executor: StepExecutorAgent},
		{ID: "review_technical_solution", Name: "文档审校", Executor: StepExecutorAgent},
		{ID: "confirm_technical_solution", Name: "文档终审", Executor: StepExecutorHuman},
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
	if got := loaded.Workflow.CommonSkills; len(got) != 2 || got[0].ID != "grill-with-docs" || got[0].Optional == nil || *got[0].Optional || got[1].ID != "human-step-jump" || got[1].Optional == nil || !*got[1].Optional {
		t.Fatalf("Common Skills = %#v", got)
	}
	if loaded.Workflow.StepStart == nil || loaded.Workflow.StepStart.PromptRef.PromptID != "step_start_control" || !reflect.DeepEqual(loaded.Workflow.StepStart.When.AnyOf, [][]string{{"step_scope_confirmed"}}) {
		t.Fatalf("Step start control = %#v", loaded.Workflow.StepStart)
	}
	if loaded.Workflow.Jump == nil || loaded.Workflow.Jump.PromptRef.PromptID != "human_step_jump_control" || !reflect.DeepEqual(loaded.Workflow.Jump.When.AnyOf, [][]string{{"human_step_jump_requested"}}) {
		t.Fatalf("Step jump control = %#v", loaded.Workflow.Jump)
	}
	for _, conditionID := range []string{"step_scope_confirmed", "human_step_jump_requested"} {
		condition, ok := loaded.Workflow.CommonConditions[conditionID]
		if !ok || condition.ExclusiveGroup != "common_control_action" {
			t.Fatalf("Common Condition %s = %#v", conditionID, condition)
		}
	}
	wantSkills := map[string]string{
		"frame_requirement_background_flow":    "technical-background-framing",
		"define_goals_and_problems_flow":       "technical-goals-and-problems",
		"define_business_constraints_flow":     "technical-business-constraints",
		"confirm_technical_problem_flow":       "technical-problem-approval",
		"research_solution_options_flow":       "technical-solution-research",
		"design_overall_solution_flow":         "technical-overall-solution",
		"design_key_solutions_flow":            "technical-key-solutions",
		"record_technical_decisions_flow":      "technical-decision-recording",
		"confirm_solution_direction_flow":      "technical-direction-approval",
		"plan_solution_delivery_flow":          "technical-solution-delivery",
		"evaluate_solution_benefits_flow":      "technical-solution-benefits",
		"write_retrospective_and_roadmap_flow": "technical-retrospective-planning",
		"write_summary_flow":                   "technical-summary-writing",
		"write_technical_solution_flow":        "technical-solution-writing",
		"review_technical_solution_flow":       "technical-solution-review",
		"confirm_technical_solution_flow":      "technical-solution-approval",
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
	assertWorkflowRoute(t, loaded, "frame_requirement_background", []string{"background_defined"}, "define_goals_and_problems", false)
	assertWorkflowRoute(t, loaded, "define_goals_and_problems", []string{"goals_and_problems_defined"}, "define_business_constraints", false)
	assertWorkflowRoute(t, loaded, "define_business_constraints", []string{"business_constraints_defined"}, "confirm_technical_problem", false)
	assertWorkflowRoute(t, loaded, "confirm_technical_problem", []string{"problem_document_published", "panorama_card_published", "technical_problem_approved"}, "research_solution_options", false)
	assertWorkflowRoute(t, loaded, "research_solution_options", []string{"solution_research_completed"}, "design_overall_solution", false)
	assertWorkflowRoute(t, loaded, "design_overall_solution", []string{"overall_solution_designed", "architecture_diagram_written"}, "design_key_solutions", false)
	assertWorkflowRoute(t, loaded, "design_key_solutions", []string{"key_modules_designed"}, "record_technical_decisions", false)
	assertWorkflowRoute(t, loaded, "record_technical_decisions", []string{"technical_decisions_recorded"}, "confirm_solution_direction", false)
	assertWorkflowRoute(t, loaded, "confirm_solution_direction", []string{"solution_document_published", "panorama_card_published", "solution_direction_approved"}, "plan_solution_delivery", false)
	assertWorkflowRoute(t, loaded, "plan_solution_delivery", []string{"delivery_plan_defined"}, "evaluate_solution_benefits", false)
	assertWorkflowRoute(t, loaded, "evaluate_solution_benefits", []string{"results_and_benefits_defined"}, "write_retrospective_and_roadmap", false)
	assertWorkflowRoute(t, loaded, "write_retrospective_and_roadmap", []string{"retrospective_and_roadmap_defined"}, "write_summary", false)
	assertWorkflowRoute(t, loaded, "write_summary", []string{"summary_defined"}, "write_technical_solution", false)
	assertWorkflowRoute(t, loaded, "write_technical_solution", []string{"technical_solution_written"}, "review_technical_solution", false)
	assertWorkflowRoute(t, loaded, "review_technical_solution", []string{"technical_solution_review_passed", "technical_solution_review_written"}, "confirm_technical_solution", false)
	assertWorkflowRoute(t, loaded, "confirm_technical_solution", []string{"technical_solution_document_published", "panorama_card_published", "technical_solution_approved"}, "", true)

	feedback := []struct {
		condition string
		backStep  string
	}{
		{"background_changed", "frame_requirement_background"},
		{"goals_and_problems_changed", "define_goals_and_problems"},
		{"business_constraints_changed", "define_business_constraints"},
		{"research_changed", "research_solution_options"},
		{"overall_solution_changed", "design_overall_solution"},
		{"key_modules_changed", "design_key_solutions"},
		{"technical_decisions_changed", "record_technical_decisions"},
		{"delivery_changed", "plan_solution_delivery"},
		{"results_changed", "evaluate_solution_benefits"},
		{"retrospective_changed", "write_retrospective_and_roadmap"},
		{"summary_changed", "write_summary"},
		{"presentation_changed", "write_technical_solution"},
	}
	for _, conditionID := range []string{
		"background_defined", "goals_and_problems_defined", "business_constraints_defined", "technical_problem_approved",
		"solution_research_completed", "overall_solution_designed", "key_modules_designed", "technical_decisions_recorded",
		"solution_direction_approved", "delivery_plan_defined", "results_and_benefits_defined",
		"retrospective_and_roadmap_defined", "summary_defined", "technical_solution_written",
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
	for _, item := range feedback[:7] {
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
	if got := len(loaded.Workflow.Conditions); got != 34 {
		t.Fatalf("Condition count = %d, want 34", got)
	}
}

func TestProductionMaterialFlashcardsWorkflow(t *testing.T) {
	loaded, err := Load("material-flashcards")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Ref.Digest != "sha256:14eaf31c1d7b06872dc67fc1a0d7498dbe201c335d8d333c2fea3f1c9aefd1c7" {
		t.Fatalf("material-flashcards digest = %s", loaded.Ref.Digest)
	}
	wants := []struct {
		step, name, stage, job string
		executor               StepExecutor
	}{
		{"frame_review_goal", "确认复习目标", "framing", "framing", StepExecutorAgent},
		{"understand_source", "理解原始材料", "framing", "framing", StepExecutorAgent},
		{"select_knowledge", "筛选高价值知识", "curation", "curation", StepExecutorAgent},
		{"plan_card_set", "选择类型并原子拆卡", "curation", "curation", StepExecutorAgent},
		{"draft_cards", "生成或修订卡片草稿", "drafting", "drafting", StepExecutorAgent},
		{"review_card_quality", "独立质量审核", "drafting", "drafting", StepExecutorAgent},
		{"confirm_card_preview", "预览并确认卡片", "approval", "approval", StepExecutorHuman},
		{"persist_approved_cards", "写入已批准卡片", "delivery", "delivery", StepExecutorAgent},
		{"validate_persisted_cards", "落盘后校验", "delivery", "delivery", StepExecutorAgent},
	}
	wantIDs := make([]string, 0, len(wants))
	jobs := 0
	for _, stage := range loaded.Workflow.Stages {
		jobs += len(stage.Jobs)
	}
	for _, want := range wants {
		wantIDs = append(wantIDs, want.step)
		context, _, ok := loaded.Workflow.FindStep(want.step)
		if !ok || context.Stage.ID != want.stage || context.Job.ID != want.job || context.Step.Name != want.name || context.Step.Executor != want.executor {
			t.Fatalf("material-flashcards Step %s = %#v, want name=%s stage=%s job=%s executor=%s", want.step, context, want.name, want.stage, want.job, want.executor)
		}
	}
	if len(loaded.Workflow.Stages) != 5 || jobs != 5 || !reflect.DeepEqual(loaded.Workflow.OrderedStepIDs(), wantIDs) || len(loaded.Workflow.Conditions) != 23 || len(loaded.Workflow.Prompts) != 24 {
		t.Fatalf("material-flashcards shape = stages:%d jobs:%d steps:%v conditions:%d prompts:%d", len(loaded.Workflow.Stages), jobs, loaded.Workflow.OrderedStepIDs(), len(loaded.Workflow.Conditions), len(loaded.Workflow.Prompts))
	}

	wantSkills := map[string][]struct {
		id       string
		optional bool
	}{
		"frame_review_goal_flow":        {{"flashcard-goal-framing", false}},
		"understand_source_flow":        {{"flashcard-source-understanding", false}},
		"select_knowledge_flow":         {{"flashcard-knowledge-selection", false}},
		"plan_card_set_flow":            {{"flashcard-card-planning", false}, {"flashcard", true}},
		"draft_cards_flow":              {{"flashcard", false}},
		"review_card_quality_flow":      {{"flashcard-quality-review", false}, {"flashcard", false}},
		"confirm_card_preview_flow":     {{"flashcard-preview-approval", false}},
		"persist_approved_cards_flow":   {{"flashcard", false}},
		"validate_persisted_cards_flow": {{"flashcard", false}},
	}
	for promptID, want := range wantSkills {
		got := loaded.Workflow.Prompts[promptID].Skills
		if len(got) != len(want) {
			t.Fatalf("Prompt %s Skills = %#v", promptID, got)
		}
		for index := range want {
			if got[index].ID != want[index].id || got[index].Optional == nil || *got[index].Optional != want[index].optional {
				t.Fatalf("Prompt %s Skill %d = %#v, want %#v", promptID, index, got[index], want[index])
			}
		}
	}
	assertConditionSkill(t, loaded, "panorama_card_published", "material-flashcards-panorama")

	assertWorkflowRoute(t, loaded, "frame_review_goal", []string{"review_goal_framed"}, "understand_source", false)
	assertWorkflowRoute(t, loaded, "understand_source", []string{"source_understood"}, "select_knowledge", false)
	assertWorkflowRoute(t, loaded, "select_knowledge", []string{"knowledge_selected"}, "plan_card_set", false)
	assertWorkflowRoute(t, loaded, "select_knowledge", []string{"no_valuable_knowledge"}, "", true)
	assertWorkflowRoute(t, loaded, "plan_card_set", []string{"card_plan_ready"}, "draft_cards", false)
	assertWorkflowRoute(t, loaded, "draft_cards", []string{"card_draft_ready"}, "review_card_quality", false)
	assertWorkflowRoute(t, loaded, "review_card_quality", []string{"quality_review_written", "card_quality_passed"}, "confirm_card_preview", false)
	assertWorkflowRoute(t, loaded, "confirm_card_preview", []string{"card_preview_published", "panorama_card_published", "card_preview_approved"}, "persist_approved_cards", false)
	assertWorkflowRoute(t, loaded, "persist_approved_cards", []string{"cards_persisted"}, "validate_persisted_cards", false)
	assertWorkflowRoute(t, loaded, "validate_persisted_cards", []string{"post_write_validation_written", "persisted_cards_validated"}, "", true)

	assertWorkflowLoop(t, loaded, "review_card_quality", []string{"quality_review_written", "review_goal_changed"}, "frame_review_goal")
	assertWorkflowLoop(t, loaded, "review_card_quality", []string{"quality_review_written", "source_understanding_changed"}, "understand_source")
	assertWorkflowLoop(t, loaded, "review_card_quality", []string{"quality_review_written", "knowledge_selection_changed"}, "select_knowledge")
	assertWorkflowLoop(t, loaded, "review_card_quality", []string{"quality_review_written", "card_plan_changed"}, "plan_card_set")
	assertWorkflowLoop(t, loaded, "review_card_quality", []string{"quality_review_written", "card_draft_changed"}, "draft_cards")
	assertWorkflowLoop(t, loaded, "confirm_card_preview", []string{"preview_draft_changed"}, "draft_cards")
	assertWorkflowLoop(t, loaded, "confirm_card_preview", []string{"card_preview_published", "panorama_card_published", "source_understanding_changed"}, "understand_source")
	assertWorkflowLoop(t, loaded, "persist_approved_cards", []string{"persistence_retry_required"}, "persist_approved_cards")
	assertWorkflowLoop(t, loaded, "persist_approved_cards", []string{"approved_draft_changed"}, "draft_cards")
	assertWorkflowLoop(t, loaded, "validate_persisted_cards", []string{"post_write_validation_retry_required"}, "validate_persisted_cards")
	if _, ok := loaded.Workflow.Condition("persisted_cards_invalid"); ok {
		t.Fatal("real post-write mismatch must remain blocked instead of routing")
	}
}

func TestProductionMaintainerUsesMainAgentAcceptanceTopology(t *testing.T) {
	loaded, err := Load("fanloop-maintainer")
	if err != nil {
		t.Fatal(err)
	}
	wants := []struct {
		step, name, stage, job string
		executor               StepExecutor
	}{
		{"bootstrap_techdesign", "仓库范围确定", "techdesign", "techdesign", StepExecutorAgent},
		{"clarify_requirements", "需求澄清", "techdesign", "techdesign", StepExecutorAgent},
		{"design_technical_solution", "方案设计", "techdesign", "techdesign", StepExecutorAgent},
		{"confirm_technical_solution", "方案自主评审", "techdesign", "techdesign", StepExecutorAgent},
		{"implement_code", "代码实现与过程 CR", "implement", "implement", StepExecutorAgent},
		{"review_code", "整体 Code Review", "implement", "implement", StepExecutorAgent},
		{"execute_agent_acceptance", "Agent 端到端测试", "test", "test", StepExecutorAgent},
		{"confirm_main_agent_acceptance", "主 Agent 验收决策", "test", "test", StepExecutorAgent},
		{"merge_and_update_local", "PR 合码与本地更新", "test", "test", StepExecutorAgent},
	}
	wantSteps := make([]string, 0, len(wants))
	for _, want := range wants {
		wantSteps = append(wantSteps, want.step)
		context, _, ok := loaded.Workflow.FindStep(want.step)
		if !ok || context.Stage.ID != want.stage || context.Job.ID != want.job || context.Step.Name != want.name || context.Step.Executor != want.executor {
			t.Fatalf("maintainer Step %s = %#v, want name=%s stage=%s job=%s executor=%s", want.step, context, want.name, want.stage, want.job, want.executor)
		}
	}
	if got := loaded.Workflow.OrderedStepIDs(); !reflect.DeepEqual(got, wantSteps) {
		t.Fatalf("maintainer Steps = %v, want %v", got, wantSteps)
	}
	for _, skillID := range []string{
		"fanloop-dev-bootstrap", "fanloop-dev-grill-with-docs", "fanloop-dev-grilling",
		"fanloop-dev-domain-modeling", "fanloop-dev-decision-receipt", "fanloop-dev-human-step-jump",
		"fanloop-dev-to-spec", "fanloop-dev-to-tickets",
		"fanloop-dev-implement", "fanloop-dev-tdd", "fanloop-dev-code-review",
		"fanloop-dev-agent-acceptance", "fanloop-dev-merge-and-update-local", "resolving-merge-conflicts",
		"fanloop-dev-panorama",
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
	assertConditionSkill(t, loaded, "panorama_presented", "fanloop-dev-panorama")
	jump := []string{"human_step_jump_requested", "human_step_jump_context_written", "human_step_jump_recorded", "panorama_presented"}
	assertWorkflowRouteAnyOf(t, loaded, "clarify_requirements", [][]string{
		{"requirements_grilled", "requirements_document_published", "requirements_approved", "requirements_approval_recorded", "requirements_evidence_written", "panorama_presented"},
		jump,
	}, "design_technical_solution", false)
	assertWorkflowRoute(t, loaded, "clarify_requirements", []string{"requirements_grilled", "requirements_document_published", "requirements_rejected", "requirements_approval_recorded", "requirements_evidence_written", "panorama_presented"}, "", true)
	assertWorkflowRouteAnyOf(t, loaded, "design_technical_solution", [][]string{{"spec_written", "tickets_written", "technical_solution_document_published", "panorama_presented"}, jump}, "confirm_technical_solution", false)
	assertWorkflowRouteAnyOf(t, loaded, "confirm_technical_solution", [][]string{{"technical_solution_review_passed", "panorama_presented"}, jump}, "implement_code", false)
	assertWorkflowRouteAnyOf(t, loaded, "implement_code", [][]string{{"implementation_completed", "implementation_report_written", "panorama_presented"}, jump}, "review_code", false)
	assertWorkflowRouteAnyOf(t, loaded, "execute_agent_acceptance", [][]string{{"agent_acceptance_passed", "acceptance_report_written", "acceptance_document_published", "panorama_presented"}, jump}, "confirm_main_agent_acceptance", false)
	assertWorkflowRouteAnyOf(t, loaded, "confirm_main_agent_acceptance", [][]string{{"main_agent_acceptance_passed", "main_agent_acceptance_recorded", "panorama_presented"}, jump}, "merge_and_update_local", false)
	assertWorkflowRouteAnyOf(t, loaded, "merge_and_update_local", [][]string{
		{"handoff_main_unchanged", "merge_request_published", "remote_checks_passed", "review_comment_synced", "code_merged", "source_repository_updated", "local_cli_updated", "delivery_record_written", "panorama_presented"},
		{"handoff_main_integrated", "handoff_integration_review_passed", "handoff_integration_tests_passed", "handoff_integration_main_agent_verified", "merge_request_published", "remote_checks_passed", "review_comment_synced", "code_merged", "source_repository_updated", "local_cli_updated", "delivery_record_written", "panorama_presented"},
	}, "", true)
	assertWorkflowLoop(t, loaded, "confirm_technical_solution", []string{"technical_solution_review_failed", "panorama_presented"}, "design_technical_solution")
	assertWorkflowLoop(t, loaded, "review_code", []string{"code_review_blocked", "review_report_written", "code_review_document_published", "panorama_presented"}, "implement_code")
	assertWorkflowLoop(t, loaded, "execute_agent_acceptance", []string{"agent_acceptance_failed", "requirements_changed", "acceptance_report_written", "acceptance_document_published", "panorama_presented"}, "clarify_requirements")
	assertWorkflowLoop(t, loaded, "confirm_main_agent_acceptance", []string{"requirements_changed", "main_agent_acceptance_recorded", "panorama_presented"}, "clarify_requirements")
	assertWorkflowLoop(t, loaded, "confirm_main_agent_acceptance", []string{"implementation_changes_requested", "main_agent_acceptance_recorded", "panorama_presented"}, "implement_code")
	assertWorkflowLoop(t, loaded, "merge_and_update_local", []string{"merge_request_published", "remote_checks_failed", "panorama_presented"}, "implement_code")

	for _, removed := range []string{
		"confirm_requirements", "merge_code", "update_local_cli", "handoff_merge_request", "confirm_human_acceptance", "maintain_verification_skill",
		"maintain_feature_map", "execute_test_cases", "coordinate_eval", "execute_eval_candidates",
		"judge_eval", "publish_candidate", "verify_ci_gates", "confirm_human_acceptance",
	} {
		if _, _, ok := loaded.Workflow.FindStep(removed); ok {
			t.Fatalf("maintainer Workflow still contains removed Step %s", removed)
		}
	}
	for _, removed := range []string{
		"agent_approved", "implementation_required", "implementation_not_required", "review_passed",
		"review_failed", "candidate_head_frozen", "merge_request_handed_off", "handoff_record_written",
		"human_acceptance_passed", "human_acceptance_skipped", "human_acceptance_result_recorded",
		"human_review_written", "handoff_integration_human_verified",
	} {
		if _, ok := loaded.Workflow.Condition(removed); ok {
			t.Fatalf("maintainer Workflow still contains removed Condition %s", removed)
		}
	}
	flowRoutes, loopRoutes := 0, 0
	for _, routes := range loaded.Workflow.Flows {
		flowRoutes += len(routes)
	}
	for _, routes := range loaded.Workflow.Loops {
		loopRoutes += len(routes)
	}
	for conditionID, want := range map[string]struct{ key, description string }{
		"requirements_document_published":       {"requirement_document_url", "需求确认报告"},
		"technical_solution_document_published": {"technical_design_document_url", "技术方案文档"},
		"code_review_document_published":        {"code_review_document_url", "Code Review 报告"},
		"acceptance_document_published":         {"acceptance_document_url", "Agent 验收报告"},
	} {
		condition, _ := loaded.Workflow.Condition(conditionID)
		if condition.Output.Key != want.key || condition.Output.Description != want.description {
			t.Fatalf("Condition %s Output = %#v", conditionID, condition.Output)
		}
	}
	if got := len(loaded.Workflow.Conditions); got != 51 {
		t.Fatalf("maintainer Conditions = %d, want 51", got)
	}
	if flowRoutes != 38 || loopRoutes != 45 || len(loaded.Workflow.Prompts) != 57 {
		t.Fatalf("maintainer route/prompt counts = flow:%d loop:%d prompts:%d, want 38/45/57", flowRoutes, loopRoutes, len(loaded.Workflow.Prompts))
	}
	for promptID, prompt := range loaded.Workflow.Prompts {
		if strings.Contains(prompt.Prompt, "./tests/run-e2e") {
			t.Fatalf("maintainer Prompt %s still requires retired ./tests/run-e2e", promptID)
		}
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
