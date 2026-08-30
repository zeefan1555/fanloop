package runtime_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentApprovalsAdvanceTechnicalSolutionWorkflow(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "Technical solution approvals"), "flow.init")

	for _, report := range []struct {
		step       string
		conditions []string
		next       string
		evidence   string
	}{
		{step: "frame_technical_problem", conditions: []string{conditionResult("technical_problem_defined", "path", `".technical-solution/problem.md"`)}, next: "confirm_technical_problem"},
		{step: "confirm_technical_problem", conditions: []string{conditionResult("agent_approved", "enum_value", `"approved"`)}, next: "derive_technical_solution", evidence: `{"source":"ai","content":"reviewed .technical-solution/problem.md: no blockers","ref":"agent-problem-approval"}`},
		{step: "derive_technical_solution", conditions: []string{conditionResult("technical_solution_derived", "path", `".technical-solution/proposal.md"`)}, next: "confirm_solution_direction"},
		{step: "confirm_solution_direction", conditions: []string{conditionResult("agent_approved", "enum_value", `"approved"`)}, next: "write_technical_solution", evidence: `{"source":"ai","content":"reviewed problem.md and proposal.md: no blockers","ref":"agent-direction-approval"}`},
		{step: "write_technical_solution", conditions: []string{
			conditionResult("technical_solution_written", "path", `"technical-solution.md"`),
			conditionResult("architecture_diagram_written", "path", `".technical-solution/architecture.mmd"`),
		}, next: "review_technical_solution"},
		{step: "review_technical_solution", conditions: []string{
			conditionResult("technical_solution_review_passed", "enum_value", `"passed"`),
			conditionResult("technical_solution_review_written", "path", `".technical-solution/review.md"`),
		}, next: "confirm_technical_solution"},
	} {
		args := []string{"flow", "report", "result", "--root", root, "--step-id", report.step, "--next-step-id", report.next, "--summary", "accepted"}
		for _, condition := range report.conditions {
			args = append(args, "--condition-result", condition)
		}
		if report.evidence != "" {
			args = append(args, "--evidence", report.evidence)
		}
		result := run(binary, args...)
		assertSuccess(t, result, "flow.report.result")
		assertFlowEffect(t, result.stdout, "advanced", report.next)
	}

	completed := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "confirm_technical_solution",
		"--condition-result", conditionResult("agent_approved", "enum_value", `"approved"`),
		"--evidence", `{"source":"ai","content":"reviewed problem.md, proposal.md, technical-solution.md, architecture.mmd, and review.md: no blockers","ref":"agent-solution-approval"}`,
		"--terminal", "--summary", "technical solution approved")
	assertSuccess(t, completed, "flow.report.result")
	if !strings.Contains(completed.stdout, `"effect": "completed"`) || !strings.Contains(completed.stdout, `"status": "completed"`) {
		t.Fatalf("completion response = %s", completed.stdout)
	}
	events := string(readFile(t, filepath.Join(root, ".fanloop", "trace", "events.jsonl")))
	for _, want := range []string{"agent-problem-approval", "agent-direction-approval", "agent-solution-approval"} {
		if !strings.Contains(events, want) {
			t.Fatalf("Flow Events do not contain %q:\n%s", want, events)
		}
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
