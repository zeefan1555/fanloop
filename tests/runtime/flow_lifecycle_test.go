package runtime_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestTechnicalSolutionProgressAndLoopsInvalidateOutputs(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "Technical solution lifecycle"), "flow.init")

	progress := run(binary, "flow", "report", "progress", "--root", root,
		"--step-id", "frame_technical_problem", "--status", "in_progress", "--summary", "framing")
	assertSuccess(t, progress, "flow.report.progress")
	assertFlowEffect(t, progress.stdout, "status_updated", "frame_technical_problem")

	ready := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "frame_technical_problem",
		"--condition-result", conditionResult("technical_problem_defined", "path", `".technical-solution/problem.md"`),
		"--next-step-id", "confirm_technical_problem", "--summary", "technical problem defined")
	assertSuccess(t, ready, "flow.report.result")
	assertFlowEffect(t, ready.stdout, "advanced", "confirm_technical_problem")

	flowState := readFile(t, filepath.Join(root, ".fanloop", "flow", "state.json"))
	if bytes.Contains(flowState, []byte(`"outputs"`)) {
		t.Fatalf("Flow State still embeds Outputs:\n%s", flowState)
	}
	registry := readFile(t, filepath.Join(root, ".fanloop", "output", "state.json"))
	for _, want := range []string{`"problem_definition_path"`, `"producer_step_id": "frame_technical_problem"`} {
		if !bytes.Contains(registry, []byte(want)) {
			t.Fatalf("Output Registry does not contain %s:\n%s", want, registry)
		}
	}

	rejected := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "confirm_technical_problem",
		"--condition-result", conditionResult("technical_problem_rejected", "enum_value", `"rejected"`),
		"--back-step-id", "frame_technical_problem", "--summary", "technical problem needs revision")
	assertSuccess(t, rejected, "flow.report.result")
	assertFlowEffect(t, rejected.stdout, "looped", "frame_technical_problem")
	assertOutputAbsent(t, rejected.stdout, "problem_definition_path")

	ready = run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "frame_technical_problem",
		"--condition-result", conditionResult("technical_problem_defined", "path", `".technical-solution/problem.md"`),
		"--next-step-id", "confirm_technical_problem", "--summary", "technical problem defined again")
	assertSuccess(t, ready, "flow.report.result")
	approved := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "confirm_technical_problem",
		"--condition-result", conditionResult("technical_problem_approved", "enum_value", `"approved"`),
		"--next-step-id", "derive_technical_solution", "--summary", "technical problem approved")
	assertSuccess(t, approved, "flow.report.result")
	derived := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "derive_technical_solution",
		"--condition-result", conditionResult("technical_solution_derived", "path", `".technical-solution/proposal.md"`),
		"--next-step-id", "confirm_solution_direction", "--summary", "solution derived")
	assertSuccess(t, derived, "flow.report.result")
	directionApproved := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "confirm_solution_direction",
		"--condition-result", conditionResult("solution_direction_approved", "enum_value", `"approved"`),
		"--next-step-id", "write_technical_solution", "--summary", "solution direction approved")
	assertSuccess(t, directionApproved, "flow.report.result")
	written := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "write_technical_solution",
		"--condition-result", conditionResult("technical_solution_written", "path", `"technical-solution.md"`),
		"--condition-result", conditionResult("architecture_diagram_written", "path", `".technical-solution/architecture.mmd"`),
		"--next-step-id", "review_technical_solution", "--summary", "technical solution written")
	assertSuccess(t, written, "flow.report.result")
	reviewed := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "review_technical_solution",
		"--condition-result", conditionResult("technical_solution_review_failed", "enum_value", `"failed"`),
		"--condition-result", conditionResult("technical_solution_review_written", "path", `".technical-solution/review.md"`),
		"--back-step-id", "write_technical_solution", "--summary", "technical solution needs revision")
	assertSuccess(t, reviewed, "flow.report.result")
	assertFlowEffect(t, reviewed.stdout, "looped", "write_technical_solution")
	assertOutputAbsent(t, reviewed.stdout, "technical_solution_path")
	assertOutputAbsent(t, reviewed.stdout, "architecture_diagram_path")
	if !strings.Contains(reviewed.stdout, `"problem_definition_path"`) {
		t.Fatalf("design loop removed approved Requirement Output: %s", reviewed.stdout)
	}

	events := string(readFile(t, filepath.Join(root, ".fanloop", "trace", "events.jsonl")))
	for _, fact := range []string{`"kind":"flow_progressed"`, `"effect":"advanced"`, `"effect":"looped"`, `"condition_id":"technical_solution_review_failed"`} {
		if !strings.Contains(events, fact) {
			t.Fatalf("Event audit missing %s:\n%s", fact, events)
		}
	}
}

func TestFlowResultAcceptsExplicitTechnicalSolutionRoute(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "Explicit route"), "flow.init")
	reported := run(binary, "flow", "report", "result", "--root", root, "--input", `{
  "step_id": "frame_technical_problem",
  "condition_results": [{"condition_id":"technical_problem_defined","output":{"type":"path","value":".technical-solution/problem.md"}}],
  "route": {"next_step_id":"confirm_technical_problem"},
  "evidence": [],
  "summary": "requirements ready"
}`)
	assertSuccess(t, reported, "flow.report.result")
	assertFlowEffect(t, reported.stdout, "advanced", "confirm_technical_problem")
}

func TestFlowReportRejectsRetiredCommandShapes(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "No aliases"), "flow.init")
	for _, retired := range [][]string{
		{"flow", "report", "output", "--root", root},
		{"flow", "report", "loop", "--root", root},
		{"flow", "report", "--root", root, "--type", "output"},
	} {
		if got := run(binary, retired...); got.exitCode == 0 {
			t.Fatalf("retired flow report shape was accepted: %v\n%s", retired, got.stdout)
		}
	}
}

func assertFlowEffect(t *testing.T, content, effect, stepID string) {
	t.Helper()
	var envelope struct {
		Data struct {
			Effect string `json:"effect"`
			State  struct {
				Current *struct {
					Context struct {
						StepID string `json:"step_id"`
					} `json:"context"`
				} `json:"current"`
			} `json:"state"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(content), &envelope); err != nil || envelope.Data.Effect != effect {
		t.Fatalf("effect = %q, want %q, error = %v\n%s", envelope.Data.Effect, effect, err, content)
	}
	if stepID != "" && (envelope.Data.State.Current == nil || envelope.Data.State.Current.Context.StepID != stepID) {
		t.Fatalf("current Step = %#v, want %s\n%s", envelope.Data.State.Current, stepID, content)
	}
}

func assertOutputAbsent(t *testing.T, content, key string) {
	t.Helper()
	var envelope struct {
		Data struct {
			State struct {
				Outputs map[string]json.RawMessage `json:"outputs"`
			} `json:"state"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(content), &envelope); err != nil {
		t.Fatal(err)
	}
	if _, exists := envelope.Data.State.Outputs[key]; exists {
		t.Fatalf("Loop response retained invalidated Output %q:\n%s", key, content)
	}
}
