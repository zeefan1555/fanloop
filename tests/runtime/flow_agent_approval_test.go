package runtime_test

import (
	"strings"
	"testing"
)

func TestAgentSubmittedTechnicalSolutionApprovalsAdvanceWorkflow(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "Technical solution approvals"), "flow.init")

	for _, report := range []struct {
		step       string
		conditions []string
		next       string
	}{
		{step: "frame_technical_problem", conditions: []string{conditionResult("technical_problem_defined", "path", `".technical-solution/problem.md"`)}, next: "confirm_technical_problem"},
		{step: "confirm_technical_problem", conditions: []string{conditionResult("technical_problem_approved", "enum_value", `"approved"`)}, next: "derive_technical_solution"},
		{step: "derive_technical_solution", conditions: []string{conditionResult("technical_solution_derived", "path", `".technical-solution/proposal.md"`)}, next: "confirm_solution_direction"},
		{step: "confirm_solution_direction", conditions: []string{conditionResult("solution_direction_approved", "enum_value", `"approved"`)}, next: "write_technical_solution"},
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
		result := run(binary, args...)
		assertSuccess(t, result, "flow.report.result")
		assertFlowEffect(t, result.stdout, "advanced", report.next)
	}

	completed := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "confirm_technical_solution",
		"--condition-result", conditionResult("technical_solution_approved", "enum_value", `"approved"`),
		"--terminal", "--summary", "technical solution approved")
	assertSuccess(t, completed, "flow.report.result")
	if !strings.Contains(completed.stdout, `"effect": "completed"`) || !strings.Contains(completed.stdout, `"status": "completed"`) {
		t.Fatalf("completion response = %s", completed.stdout)
	}
}
