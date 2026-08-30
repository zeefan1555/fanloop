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
	advance := func(step, next string, conditions ...string) result {
		t.Helper()
		args := []string{"flow", "report", "result", "--root", root, "--step-id", step, "--next-step-id", next, "--summary", "ready"}
		for _, condition := range conditions {
			args = append(args, "--condition-result", condition)
		}
		got := run(binary, args...)
		assertSuccess(t, got, "flow.report.result")
		assertFlowEffect(t, got.stdout, "advanced", next)
		return got
	}

	progress := run(binary, "flow", "report", "progress", "--root", root,
		"--step-id", "frame_requirement_background", "--status", "in_progress", "--summary", "framing")
	assertSuccess(t, progress, "flow.report.progress")
	assertFlowEffect(t, progress.stdout, "status_updated", "frame_requirement_background")

	advance("frame_requirement_background", "analyze_core_problem",
		conditionResult("background_defined", "path", `".technical-solution/sections/01-background.md"`))
	advance("analyze_core_problem", "define_design_objectives",
		conditionResult("core_problem_defined", "path", `".technical-solution/sections/02-problem.md"`))
	advance("define_design_objectives", "confirm_technical_problem",
		conditionResult("design_objectives_defined", "path", `".technical-solution/sections/03-objectives.md"`))

	flowState := readFile(t, filepath.Join(root, ".fanloop", "flow", "state.json"))
	if bytes.Contains(flowState, []byte(`"outputs"`)) {
		t.Fatalf("Flow State still embeds Outputs:\n%s", flowState)
	}
	registry := readFile(t, filepath.Join(root, ".fanloop", "output", "state.json"))
	for _, want := range []string{`"background_section_path"`, `"producer_step_id": "frame_requirement_background"`} {
		if !bytes.Contains(registry, []byte(want)) {
			t.Fatalf("Output Registry does not contain %s:\n%s", want, registry)
		}
	}

	rejected := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "confirm_technical_problem",
		"--condition-result", conditionResult("problem_document_published", "url", `"https://example.com/problem-definition"`),
		"--condition-result", conditionResult("panorama_card_published", "path", `".fanloop/card/problem-feedback.json"`),
		"--condition-result", conditionResult("problem_changed", "enum_value", `"problem"`),
		"--back-step-id", "analyze_core_problem", "--summary", "core problem needs revision")
	assertSuccess(t, rejected, "flow.report.result")
	assertFlowEffect(t, rejected.stdout, "looped", "analyze_core_problem")
	assertOutputAbsent(t, rejected.stdout, "problem_section_path")
	assertOutputAbsent(t, rejected.stdout, "objectives_section_path")
	assertOutputAbsent(t, rejected.stdout, "problem_document_url")
	assertOutputAbsent(t, rejected.stdout, "panorama_snapshot_path")
	if !strings.Contains(rejected.stdout, `"background_section_path"`) {
		t.Fatalf("problem loop removed approved background Output: %s", rejected.stdout)
	}

	advance("analyze_core_problem", "define_design_objectives",
		conditionResult("core_problem_defined", "path", `".technical-solution/sections/02-problem.md"`))
	advance("define_design_objectives", "confirm_technical_problem",
		conditionResult("design_objectives_defined", "path", `".technical-solution/sections/03-objectives.md"`))
	advance("confirm_technical_problem", "research_solution_options",
		conditionResult("problem_document_published", "url", `"https://example.com/problem-definition"`),
		conditionResult("panorama_card_published", "path", `".fanloop/card/problem-approved.json"`),
		conditionResult("technical_problem_approved", "enum_value", `"approved"`))
	advance("research_solution_options", "design_overall_solution",
		conditionResult("solution_research_completed", "path", `".technical-solution/sections/04-research.md"`))
	advance("design_overall_solution", "design_key_solutions",
		conditionResult("overall_solution_designed", "path", `".technical-solution/sections/05-overall-solution.md"`),
		conditionResult("architecture_diagram_written", "path", `".technical-solution/architecture.mmd"`))
	advance("design_key_solutions", "confirm_solution_direction",
		conditionResult("key_solutions_designed", "path", `".technical-solution/sections/06-key-solutions.md"`))
	advance("confirm_solution_direction", "evaluate_solution_benefits",
		conditionResult("solution_document_published", "url", `"https://example.com/solution-design"`),
		conditionResult("panorama_card_published", "path", `".fanloop/card/solution-approved.json"`),
		conditionResult("solution_direction_approved", "enum_value", `"approved"`))
	advance("evaluate_solution_benefits", "plan_solution_delivery",
		conditionResult("solution_benefits_defined", "path", `".technical-solution/sections/07-benefits.md"`))
	advance("plan_solution_delivery", "write_technical_solution",
		conditionResult("delivery_plan_defined", "path", `".technical-solution/sections/08-delivery.md"`))
	advance("write_technical_solution", "review_technical_solution",
		conditionResult("technical_solution_written", "path", `"technical-solution.md"`))
	reviewed := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "review_technical_solution",
		"--condition-result", conditionResult("technical_solution_review_written", "path", `".technical-solution/review.md"`),
		"--condition-result", conditionResult("presentation_changed", "enum_value", `"presentation"`),
		"--back-step-id", "write_technical_solution", "--summary", "presentation needs revision")
	assertSuccess(t, reviewed, "flow.report.result")
	assertFlowEffect(t, reviewed.stdout, "looped", "write_technical_solution")
	assertOutputAbsent(t, reviewed.stdout, "technical_solution_path")
	assertOutputAbsent(t, reviewed.stdout, "technical_solution_review_path")
	if !strings.Contains(reviewed.stdout, `"architecture_diagram_path"`) || !strings.Contains(reviewed.stdout, `"delivery_section_path"`) {
		t.Fatalf("presentation loop removed approved design Outputs: %s", reviewed.stdout)
	}

	events := string(readFile(t, filepath.Join(root, ".fanloop", "trace", "events.jsonl")))
	for _, fact := range []string{`"kind":"flow_progressed"`, `"effect":"advanced"`, `"effect":"looped"`, `"condition_id":"problem_changed"`, `"condition_id":"presentation_changed"`} {
		if !strings.Contains(events, fact) {
			t.Fatalf("Event audit missing %s:\n%s", fact, events)
		}
	}
}

func TestFlowResultAcceptsExplicitTechnicalSolutionRoute(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "Explicit route"), "flow.init")
	reported := run(binary, "flow", "report", "result", "--root", root, "--input", `{
  "step_id": "frame_requirement_background",
  "condition_results": [{"condition_id":"background_defined","output":{"type":"path","value":".technical-solution/sections/01-background.md"}}],
  "route": {"next_step_id":"analyze_core_problem"},
  "evidence": [],
  "summary": "requirements ready"
}`)
	assertSuccess(t, reported, "flow.report.result")
	assertFlowEffect(t, reported.stdout, "advanced", "analyze_core_problem")
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
