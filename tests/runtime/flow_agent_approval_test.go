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

func TestMaintainerFourStepLifecycleMergesAndUpdatesLocal(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "fanloop-maintainer", "--title", "Four-step delivery"), "flow.init")

	advance := func(step, next string, conditions ...string) {
		t.Helper()
		args := []string{"flow", "report", "result", "--root", root, "--step-id", step, "--next-step-id", next, "--summary", step + " complete"}
		for _, condition := range conditions {
			args = append(args, "--condition-result", condition)
		}
		assertSuccess(t, run(binary, args...), "flow.report.result")
	}
	advance("define_verification_contract", "build_until_verified",
		conditionResult("verification_contract_written", "path", "\"requirements.md\""),
		conditionResult("requirements_approved", "enum_value", "\"approved\""),
		conditionResult("requirements_decision_recorded", "string", "\"decision-requirements\""),
		conditionResult("panorama_presented", "path", "\".fanloop/card/define.md\""))
	base := "1111111111111111111111111111111111111111"
	head := "2222222222222222222222222222222222222222"
	identity := `{"candidate_head":"` + head + `","git_tree":"tree-1","binary_sha256":"sha256:binary","verification_contract_sha256":"sha256:contract","feature_impact_sha256":"sha256:impact"}`
	advance("build_until_verified", "certify_candidate",
		conditionResult("self_validation_passed", "enum_value", "\"passed\""),
		conditionResult("verification_report_written", "path", "\"verification-report.md\""),
		conditionResult("candidate_identity_recorded", "object", identity),
		conditionResult("panorama_presented", "path", "\".fanloop/card/build.md\""))
	certification := `{"certification_base":"` + base + `","certification_head":"` + head + `","git_tree":"tree-1","binary_sha256":"sha256:binary","verification_assets_sha256":"sha256:assets"}`
	advance("certify_candidate", "merge_and_update_local",
		conditionResult("independent_review_passed", "enum_value", "\"passed\""),
		conditionResult("blackbox_verification_passed", "enum_value", "\"passed\""),
		conditionResult("main_agent_acceptance_passed", "enum_value", "\"passed\""),
		conditionResult("certification_report_written", "path", "\"certification-report.md\""),
		conditionResult("certification_identity_frozen", "object", certification),
		conditionResult("panorama_presented", "path", "\".fanloop/card/certify.md\""))

	mergeCommit := "3333333333333333333333333333333333333333"
	completed := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "merge_and_update_local",
		"--condition-result", conditionResult("delivery_main_unchanged", "enum_value", "\"unchanged\""),
		"--condition-result", conditionResult("merge_request_published", "url_list", `["https://github.com/zeefan1555/fanloop/pull/7"]`),
		"--condition-result", conditionResult("remote_checks_passed", "enum_value", "\"passed\""),
		"--condition-result", conditionResult("review_comment_synced", "string", "\"comment-7\""),
		"--condition-result", conditionResult("code_merged", "string", "\""+mergeCommit+"\""),
		"--condition-result", conditionResult("merge_tree_verified", "enum_value", "\"passed\""),
		"--condition-result", conditionResult("source_repository_updated", "string", "\""+mergeCommit+"\""),
		"--condition-result", conditionResult("local_cli_updated", "string", "\""+mergeCommit+"\""),
		"--condition-result", conditionResult("post_merge_smoke_passed", "enum_value", "\"passed\""),
		"--condition-result", conditionResult("delivery_record_written", "path", "\"delivery-record.md\""),
		"--condition-result", conditionResult("panorama_presented", "path", "\".fanloop/card/handoff.md\""),
		"--terminal", "--summary", "PR merged and local CLI updated")
	assertSuccess(t, completed, "flow.report.result")
	assertFlowEffect(t, completed.stdout, "completed", "")
}
