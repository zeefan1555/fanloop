package runtime_test

import (
	"strings"
	"testing"
)

func TestTechnicalSolutionWorkflowRejectsAgentApproval(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "Human approval required"), "flow.init")

	for _, report := range []struct {
		step       string
		conditions []string
		next       string
	}{
		{step: "frame_requirement_background", conditions: []string{conditionResult("background_defined", "path", `".technical-solution/sections/01-business-background.md"`)}, next: "define_goals_and_problems"},
		{step: "define_goals_and_problems", conditions: []string{conditionResult("goals_and_problems_defined", "path", `".technical-solution/sections/02-goals-and-problems.md"`)}, next: "define_business_constraints"},
		{step: "define_business_constraints", conditions: []string{conditionResult("business_constraints_defined", "path", `".technical-solution/sections/03-business-constraints.md"`)}, next: "confirm_technical_problem"},
	} {
		ensureTechnicalStepStarted(t, binary, root, report.step)
		args := []string{"flow", "report", "result", "--root", root, "--step-id", report.step, "--next-step-id", report.next, "--summary", "accepted"}
		for _, condition := range report.conditions {
			args = append(args, "--condition-result", condition)
		}
		result := run(binary, args...)
		assertSuccess(t, result, "flow.report.result")
		assertFlowEffect(t, result.stdout, "advanced", report.next)
	}

	ensureTechnicalStepStarted(t, binary, root, "confirm_technical_problem")
	rejected := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "confirm_technical_problem",
		"--condition-result", conditionResult("agent_approved", "enum_value", `"approved"`),
		"--next-step-id", "research_solution_options", "--summary", "agent tried to approve")
	if rejected.exitCode == 0 || !strings.Contains(rejected.stderr, "agent_approved") {
		t.Fatalf("retired Agent approval was accepted:\nstdout: %s\nstderr: %s", rejected.stdout, rejected.stderr)
	}
}

func TestMaintainerLifecycleEndsAfterMainAgentAcceptanceAndPRHandoff(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "fanloop-maintainer", "--title", "Three-stage delivery"), "flow.init")

	advance := func(step, next string, conditions ...string) {
		t.Helper()
		args := []string{"flow", "report", "result", "--root", root, "--step-id", step, "--next-step-id", next, "--summary", step + " complete"}
		for _, condition := range conditions {
			args = append(args, "--condition-result", condition)
		}
		assertSuccess(t, run(binary, args...), "flow.report.result")
	}
	advance("bootstrap_techdesign", "clarify_requirements",
		conditionResult("repository_workspace_prepared", "path", "\"issue-workspace\""),
		conditionResult("panorama_presented", "path", "\".fanloop/card/bootstrap.md\""))
	advance("clarify_requirements", "design_technical_solution",
		conditionResult("requirements_grilled", "path", "\"requirements.md\""),
		conditionResult("requirements_document_published", "url", "\"https://example.com/requirements\""),
		conditionResult("requirements_approved", "enum_value", "\"approved\""),
		conditionResult("requirements_approval_recorded", "string", "\"decision-requirements\""),
		conditionResult("requirements_evidence_written", "path", "\"requirements.md\""),
		conditionResult("panorama_presented", "path", "\".fanloop/card/requirements.md\""))
	advance("design_technical_solution", "confirm_technical_solution",
		conditionResult("spec_written", "path", "\"spec.md\""),
		conditionResult("tickets_written", "path", "\"issues\""),
		conditionResult("technical_solution_document_published", "url", "\"https://example.com/design\""),
		conditionResult("panorama_presented", "path", "\".fanloop/card/design.md\""))
	advance("confirm_technical_solution", "implement_code",
		conditionResult("technical_solution_review_passed", "enum_value", "\"passed\""),
		conditionResult("panorama_presented", "path", "\".fanloop/card/design-review.md\""))
	reviewBase := "1111111111111111111111111111111111111111"
	reviewedHead := "2222222222222222222222222222222222222222"
	advance("implement_code", "review_code",
		conditionResult("implementation_completed", "string", "\""+reviewedHead+"\""),
		conditionResult("implementation_report_written", "path", "\"implementation-report.md\""),
		conditionResult("panorama_presented", "path", "\".fanloop/card/implementation.md\""))
	advance("review_code", "execute_agent_acceptance",
		conditionResult("code_review_approved", "enum_value", "\"Approve\""),
		conditionResult("local_validation_passed", "enum_value", "\"passed\""),
		conditionResult("review_report_written", "path", "\"review-report.md\""),
		conditionResult("review_base_frozen", "string", "\""+reviewBase+"\""),
		conditionResult("reviewed_head_frozen", "string", "\""+reviewedHead+"\""),
		conditionResult("code_review_document_published", "url", "\"https://example.com/review\""),
		conditionResult("panorama_presented", "path", "\".fanloop/card/review.md\""))
	advance("execute_agent_acceptance", "confirm_main_agent_acceptance",
		conditionResult("agent_acceptance_passed", "enum_value", "\"passed\""),
		conditionResult("acceptance_report_written", "path", "\"acceptance-report.md\""),
		conditionResult("acceptance_document_published", "url", "\"https://example.com/acceptance\""),
		conditionResult("panorama_presented", "path", "\".fanloop/card/acceptance.md\""))
	advance("confirm_main_agent_acceptance", "handoff_merge_request",
		conditionResult("main_agent_acceptance_passed", "enum_value", "\"passed\""),
		conditionResult("main_agent_acceptance_recorded", "path", "\"main-agent-review.md\""),
		conditionResult("panorama_presented", "path", "\".fanloop/card/main-agent-acceptance.md\""))

	completed := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "handoff_merge_request",
		"--condition-result", conditionResult("handoff_main_unchanged", "enum_value", "\"unchanged\""),
		"--condition-result", conditionResult("merge_request_published", "url_list", `["https://github.com/zeefan1555/fanloop/pull/7"]`),
		"--condition-result", conditionResult("remote_checks_passed", "enum_value", "\"passed\""),
		"--condition-result", conditionResult("review_comment_synced", "string", "\"comment-7\""),
		"--condition-result", conditionResult("merge_request_handed_off", "enum_value", "\"passed\""),
		"--condition-result", conditionResult("handoff_record_written", "path", "\"handoff-record.md\""),
		"--condition-result", conditionResult("panorama_presented", "path", "\".fanloop/card/handoff.md\""),
		"--terminal", "--summary", "PR handed off")
	assertSuccess(t, completed, "flow.report.result")
	assertFlowEffect(t, completed.stdout, "completed", "")
}
