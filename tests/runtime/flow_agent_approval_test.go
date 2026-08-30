package runtime_test

import (
	"os"
	"path/filepath"
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
		{step: "frame_requirement_background", conditions: []string{conditionResult("background_defined", "path", `".technical-solution/sections/01-background.md"`)}, next: "analyze_core_problem"},
		{step: "analyze_core_problem", conditions: []string{conditionResult("core_problem_defined", "path", `".technical-solution/sections/02-problem.md"`)}, next: "define_design_objectives"},
		{step: "define_design_objectives", conditions: []string{conditionResult("design_objectives_defined", "path", `".technical-solution/sections/03-objectives.md"`)}, next: "confirm_technical_problem"},
	} {
		args := []string{"flow", "report", "result", "--root", root, "--step-id", report.step, "--next-step-id", report.next, "--summary", "accepted"}
		for _, condition := range report.conditions {
			args = append(args, "--condition-result", condition)
		}
		result := run(binary, args...)
		assertSuccess(t, result, "flow.report.result")
		assertFlowEffect(t, result.stdout, "advanced", report.next)
	}

	rejected := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "confirm_technical_problem",
		"--condition-result", conditionResult("agent_approved", "enum_value", `"approved"`),
		"--next-step-id", "research_solution_options", "--summary", "agent tried to approve")
	if rejected.exitCode == 0 || !strings.Contains(rejected.stderr, "agent_approved") {
		t.Fatalf("retired Agent approval was accepted:\nstdout: %s\nstderr: %s", rejected.stdout, rejected.stderr)
	}
}

func TestAgentApprovalAdvancesOrCompletesMaintainerWorkflow(t *testing.T) {
	binary := buildCLI(t)
	for _, test := range []struct {
		name      string
		condition string
		effect    string
		next      string
		terminal  bool
	}{
		{name: "implementation required", condition: "implementation_required", effect: "advanced", next: "design_technical_solution"},
		{name: "implementation not required", condition: "implementation_not_required", effect: "completed", terminal: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			workspace := filepath.Join(root, "issue-workspace")
			if err := os.Mkdir(workspace, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(workspace, "requirements.md"), []byte("approved requirements\n"), 0o600); err != nil {
				t.Fatal(err)
			}

			assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "fanloop-maintainer", "--title", "Maintainer agent approval"), "flow.init")
			assertSuccess(t, run(binary, "flow", "report", "result", "--root", root,
				"--step-id", "bootstrap_techdesign",
				"--condition-result", conditionResult("repository_workspace_prepared", "path", `"issue-workspace"`),
				"--next-step-id", "clarify_requirements", "--summary", "workspace prepared"), "flow.report.result")
			assertSuccess(t, run(binary, "flow", "report", "result", "--root", root,
				"--step-id", "clarify_requirements",
				"--condition-result", conditionResult("requirements_grilled", "path", `"issue-workspace/requirements.md"`),
				"--condition-result", conditionResult("requirements_document_published", "url", `"https://example.com/requirements"`),
				"--next-step-id", "confirm_requirements", "--summary", "requirements ready"), "flow.report.result")

			args := []string{"flow", "report", "result", "--root", root,
				"--step-id", "confirm_requirements",
				"--condition-result", conditionResult("agent_approved", "enum_value", `"approved"`),
				"--condition-result", conditionResult(test.condition, "boolean", `true`),
				"--evidence", `{"source":"ai","content":"reviewed issue-workspace/requirements.md: no blockers","ref":"maintainer-agent-approval"}`,
				"--summary", "agent approved requirements"}
			if test.terminal {
				args = append(args, "--terminal")
			} else {
				args = append(args, "--next-step-id", test.next)
			}
			result := run(binary, args...)
			assertSuccess(t, result, "flow.report.result")
			assertFlowEffect(t, result.stdout, test.effect, test.next)
			events := string(readFile(t, filepath.Join(root, ".fanloop", "trace", "events.jsonl")))
			for _, want := range []string{"reviewed issue-workspace/requirements.md: no blockers", "maintainer-agent-approval"} {
				if !strings.Contains(events, want) {
					t.Fatalf("Flow Events do not contain %q:\n%s", want, events)
				}
			}
		})
	}
}

func TestMaintainerTrustCurveAdvancesToAutomaticMerge(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "fanloop-maintainer", "--title", "Trust curve auto merge"), "flow.init")

	advance := func(step, next string, conditions ...string) {
		t.Helper()
		args := []string{"flow", "report", "result", "--root", root, "--step-id", step, "--next-step-id", next, "--summary", step + " complete"}
		for _, condition := range conditions {
			args = append(args, "--condition-result", condition)
		}
		assertSuccess(t, run(binary, args...), "flow.report.result")
	}
	advance("bootstrap_techdesign", "clarify_requirements",
		conditionResult("repository_workspace_prepared", "path", "\"issue-workspace\""))
	advance("clarify_requirements", "confirm_requirements",
		conditionResult("requirements_grilled", "path", "\"requirements.md\""),
		conditionResult("requirements_document_published", "url", "\"https://example.com/requirements\""))
	advance("confirm_requirements", "design_technical_solution",
		conditionResult("agent_approved", "enum_value", "\"approved\""),
		conditionResult("implementation_required", "boolean", "true"))
	advance("design_technical_solution", "implement_code",
		conditionResult("spec_written", "path", "\"spec.md\""),
		conditionResult("tickets_written", "path", "\"ticket-01.md\""),
		conditionResult("technical_solution_document_published", "url", "\"https://example.com/design\""))
	advance("implement_code", "maintain_verification_skill",
		conditionResult("implementation_completed", "enum_value", "\"completed\""))
	advance("maintain_verification_skill", "maintain_feature_map",
		conditionResult("verification_skill_ready", "path", "\".agents/skills/verify-fanloop/SKILL.md\""))
	advance("maintain_feature_map", "execute_test_cases",
		conditionResult("feature_map_current", "enum_value", "\"clean\""))
	advance("execute_test_cases", "review_code",
		conditionResult("validation_profile_selected", "enum_value", "\"e2e\""),
		conditionResult("e2e_entrypoint_passed", "enum_value", "\"passed\""),
		conditionResult("local_test_report_written", "path", "\"local-test-report.md\""))
	candidateHead := "0123456789abcdef0123456789abcdef01234567"
	advance("review_code", "coordinate_eval",
		conditionResult("review_passed", "enum_value", "\"passed\""),
		conditionResult("review_report_written", "path", "\"review-report.md\""),
		conditionResult("candidate_head_frozen", "string", "\""+candidateHead+"\""))
	advance("coordinate_eval", "execute_eval_candidates",
		conditionResult("eval_playbook_frozen", "path", "\"eval-playbook.md\""))
	advance("execute_eval_candidates", "judge_eval",
		conditionResult("eval_candidates_completed", "path", "\"eval-candidates-report.md\""))
	advance("judge_eval", "publish_candidate",
		conditionResult("agent_eval_passed", "enum_value", "\"passed\""),
		conditionResult("acceptance_report_written", "path", "\"acceptance-report.md\""))
	advance("publish_candidate", "verify_ci_gates",
		conditionResult("pull_request_published", "url", "\"https://github.com/zeefan1555/fanloop/pull/7\""))
	advance("verify_ci_gates", "execute_agent_acceptance",
		conditionResult("repository_guardrails_verified", "string", "\"1234\""),
		conditionResult("ci_gates_passed", "string", "\""+candidateHead+"\""))
	advance("execute_agent_acceptance", "merge_code",
		conditionResult("agent_acceptance_passed", "enum_value", "\"passed\""),
		conditionResult("acceptance_report_written", "path", "\"acceptance-report.md\""))

	completed := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "merge_code",
		"--condition-result", conditionResult("code_merged", "url_list", `["https://github.com/zeefan1555/fanloop/pull/7"]`),
		"--condition-result", conditionResult("acceptance_report_written", "path", "\"acceptance-report.md\""),
		"--terminal", "--summary", "reviewed head merged")
	assertSuccess(t, completed, "flow.report.result")
}
