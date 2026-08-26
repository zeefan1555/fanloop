package runtime_test

import (
	"strings"
	"testing"
)

func TestAgentSubmittedPromotionApprovalsAdvanceWorkflow(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "promotion-design", "--title", "Promotion approvals"), "flow.init")

	for _, report := range []struct {
		step       string
		conditions []string
		next       string
	}{
		{step: "clarify_requirements", conditions: []string{conditionResult("requirements_ready", "path", `"require_points.md"`)}, next: "confirm_requirements"},
		{step: "confirm_requirements", conditions: []string{conditionResult("requirements_approved", "enum_value", `"approved"`)}, next: "write_promotion_design"},
		{step: "write_promotion_design", conditions: []string{conditionResult("promotion_design_written", "path", `"方案.md"`)}, next: "review_promotion_design"},
		{step: "review_promotion_design", conditions: []string{
			conditionResult("design_review_passed", "enum_value", `"passed"`),
			conditionResult("design_review_written", "path", `".promotion/review.md"`),
		}, next: "confirm_promotion_design"},
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
		"--step-id", "confirm_promotion_design",
		"--condition-result", conditionResult("promotion_design_approved", "enum_value", `"approved"`),
		"--terminal", "--summary", "promotion design approved")
	assertSuccess(t, completed, "flow.report.result")
	if !strings.Contains(completed.stdout, `"effect": "completed"`) || !strings.Contains(completed.stdout, `"status": "completed"`) {
		t.Fatalf("completion response = %s", completed.stdout)
	}
}
