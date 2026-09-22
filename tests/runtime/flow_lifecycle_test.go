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
		ensureTechnicalStepStarted(t, binary, root, step)
		args := []string{"flow", "report", "result", "--root", root, "--step-id", step, "--next-step-id", next, "--summary", "ready"}
		for _, condition := range conditions {
			args = append(args, "--condition-result", condition)
		}
		got := run(binary, args...)
		assertSuccess(t, got, "flow.report.result")
		assertFlowEffect(t, got.stdout, "advanced", next)
		return got
	}

	ensureTechnicalStepStarted(t, binary, root, "frame_requirement_background")
	progress := run(binary, "flow", "report", "progress", "--root", root,
		"--step-id", "frame_requirement_background", "--status", "in_progress", "--summary", "framing")
	assertSuccess(t, progress, "flow.report.progress")
	assertFlowEffect(t, progress.stdout, "status_updated", "frame_requirement_background")

	advance("frame_requirement_background", "define_goals_and_problems",
		conditionResult("background_defined", "path", `".technical-solution/sections/01-business-background.md"`))
	advance("define_goals_and_problems", "define_business_constraints",
		conditionResult("goals_and_problems_defined", "path", `".technical-solution/sections/02-goals-and-problems.md"`))
	advance("define_business_constraints", "confirm_technical_problem",
		conditionResult("business_constraints_defined", "path", `".technical-solution/sections/03-business-constraints.md"`))

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

	ensureTechnicalStepStarted(t, binary, root, "confirm_technical_problem")
	rejected := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "confirm_technical_problem",
		"--condition-result", conditionResult("problem_document_published", "url", `"https://example.com/problem-definition"`),
		"--condition-result", conditionResult("panorama_card_published", "path", `".fanloop/card/problem-feedback.json"`),
		"--condition-result", conditionResult("goals_and_problems_changed", "enum_value", `"goals_and_problems"`),
		"--back-step-id", "define_goals_and_problems", "--summary", "goals and problems need revision")
	assertSuccess(t, rejected, "flow.report.result")
	assertFlowEffect(t, rejected.stdout, "looped", "define_goals_and_problems")
	assertOutputAbsent(t, rejected.stdout, "goals_and_problems_section_path")
	assertOutputAbsent(t, rejected.stdout, "business_constraints_section_path")
	assertOutputPresent(t, rejected.stdout, "problem_document_url")
	assertOutputPresent(t, rejected.stdout, "panorama_snapshot_path")
	assertOutputPresent(t, rejected.stdout, "technical_feedback_scope")
	if !strings.Contains(rejected.stdout, `"background_section_path"`) {
		t.Fatalf("problem loop removed approved background Output: %s", rejected.stdout)
	}

	advance("define_goals_and_problems", "define_business_constraints",
		conditionResult("goals_and_problems_defined", "path", `".technical-solution/sections/02-goals-and-problems.md"`))
	advance("define_business_constraints", "confirm_technical_problem",
		conditionResult("business_constraints_defined", "path", `".technical-solution/sections/03-business-constraints.md"`))
	advance("confirm_technical_problem", "research_solution_options",
		conditionResult("problem_document_published", "url", `"https://example.com/problem-definition"`),
		conditionResult("panorama_card_published", "path", `".fanloop/card/problem-approved.json"`),
		conditionResult("technical_problem_approved", "enum_value", `"approved"`))
	advance("research_solution_options", "design_overall_solution",
		conditionResult("solution_research_completed", "path", `".technical-solution/sections/04-research.md"`))
	advance("design_overall_solution", "design_key_solutions",
		conditionResult("overall_solution_designed", "path", `".technical-solution/sections/05-overall-solution.md"`),
		conditionResult("architecture_diagram_written", "path", `".technical-solution/architecture.mmd"`))
	advance("design_key_solutions", "record_technical_decisions",
		conditionResult("key_modules_designed", "path", `".technical-solution/sections/06-key-modules.md"`))
	advance("record_technical_decisions", "confirm_solution_direction",
		conditionResult("technical_decisions_recorded", "path", `".technical-solution/sections/07-decisions.md"`))
	advance("confirm_solution_direction", "plan_solution_delivery",
		conditionResult("solution_document_published", "url", `"https://example.com/solution-design"`),
		conditionResult("panorama_card_published", "path", `".fanloop/card/solution-approved.json"`),
		conditionResult("solution_direction_approved", "enum_value", `"approved"`))
	advance("plan_solution_delivery", "evaluate_solution_benefits",
		conditionResult("delivery_plan_defined", "path", `".technical-solution/sections/08-delivery-and-risk.md"`))
	advance("evaluate_solution_benefits", "write_retrospective_and_roadmap",
		conditionResult("results_and_benefits_defined", "path", `".technical-solution/sections/09-results.md"`))
	advance("write_retrospective_and_roadmap", "write_summary",
		conditionResult("retrospective_and_roadmap_defined", "path", `".technical-solution/sections/10-retrospective-and-roadmap.md"`))
	advance("write_summary", "write_technical_solution",
		conditionResult("summary_defined", "path", `".technical-solution/sections/00-summary.md"`))
	advance("write_technical_solution", "review_technical_solution",
		conditionResult("technical_solution_written", "path", `"technical-solution.md"`))
	ensureTechnicalStepStarted(t, binary, root, "review_technical_solution")
	reviewed := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "review_technical_solution",
		"--condition-result", conditionResult("technical_solution_review_written", "path", `".technical-solution/review.md"`),
		"--condition-result", conditionResult("presentation_changed", "enum_value", `"presentation"`),
		"--back-step-id", "write_technical_solution", "--summary", "presentation needs revision")
	assertSuccess(t, reviewed, "flow.report.result")
	assertFlowEffect(t, reviewed.stdout, "looped", "write_technical_solution")
	assertOutputAbsent(t, reviewed.stdout, "technical_solution_path")
	assertOutputPresent(t, reviewed.stdout, "technical_solution_review_path")
	assertOutputPresent(t, reviewed.stdout, "technical_feedback_scope")
	if !strings.Contains(reviewed.stdout, `"architecture_diagram_path"`) || !strings.Contains(reviewed.stdout, `"delivery_section_path"`) {
		t.Fatalf("presentation loop removed approved design Outputs: %s", reviewed.stdout)
	}

	events := string(readFile(t, filepath.Join(root, ".fanloop", "trace", "events.jsonl")))
	for _, fact := range []string{`"kind":"flow_progressed"`, `"effect":"advanced"`, `"effect":"looped"`, `"condition_id":"goals_and_problems_changed"`, `"condition_id":"presentation_changed"`} {
		if !strings.Contains(events, fact) {
			t.Fatalf("Event audit missing %s:\n%s", fact, events)
		}
	}
}

func TestRemnoteFlashcardsRecoveryPreservesEarlierFacts(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "remnote-flashcards", "--title", "Synthetic RemNote lifecycle"), "flow.init")
	advance := func(step, next, condition, value string) {
		t.Helper()
		got := run(binary, "flow", "report", "result", "--root", root, "--step-id", step, "--condition-result", conditionResult(condition, "object", value), "--next-step-id", next, "--summary", "synthetic receipt; no RemNote write")
		assertSuccess(t, got, "flow.report.result")
		assertFlowEffect(t, got.stdout, "advanced", next)
	}
	advance("resolve_home_tag", "write_flashcards", "home_tag_ready", `{"tag_id":"synthetic-tag"}`)
	advance("write_flashcards", "update_wiki_links", "flashcards_written", `{"created_rem_ids":[],"reused_rem_ids":["synthetic-card"],"skipped_count":1}`)
	advance("update_wiki_links", "verify_result", "wiki_links_updated", `{"reused_page_ids":["synthetic-wiki"],"linked_card_ids":["synthetic-card"]}`)
	loop := run(binary, "flow", "report", "result", "--root", root, "--step-id", "verify_result", "--condition-result", conditionResult("wiki_links_need_fix", "enum_value", `"wiki_links"`), "--back-step-id", "update_wiki_links", "--summary", "synthetic broken reference")
	assertSuccess(t, loop, "flow.report.result")
	assertFlowEffect(t, loop.stdout, "looped", "update_wiki_links")
	assertOutputPresent(t, loop.stdout, "home_tag")
	assertOutputPresent(t, loop.stdout, "flashcard_receipt")
	assertOutputAbsent(t, loop.stdout, "wiki_link_receipt")
	blocked := run(binary, "flow", "report", "progress", "--root", root, "--step-id", "update_wiki_links", "--status", "blocked", "--summary", "reference unavailable")
	assertSuccess(t, blocked, "flow.report.progress")
	assertFlowEffect(t, blocked.stdout, "status_updated", "update_wiki_links")
	assertOutputPresent(t, blocked.stdout, "flashcard_receipt")
	advance("update_wiki_links", "verify_result", "wiki_links_updated", `{"reused_page_ids":["synthetic-wiki"],"linked_card_ids":["synthetic-card"]}`)
	done := run(binary, "flow", "report", "result", "--root", root, "--step-id", "verify_result", "--condition-result", conditionResult("result_verified", "enum_value", `"passed"`), "--terminal", "--summary", "synthetic verification")
	assertSuccess(t, done, "flow.report.result")
	assertFlowEffect(t, done.stdout, "completed", "")
}

func TestFlowResultAcceptsExplicitTechnicalSolutionRoute(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "Explicit route"), "flow.init")
	ensureTechnicalStepStarted(t, binary, root, "frame_requirement_background")
	reported := run(binary, "flow", "report", "result", "--root", root, "--input", `{
  "step_id": "frame_requirement_background",
  "condition_results": [{"condition_id":"background_defined","output":{"type":"path","value":".technical-solution/sections/01-business-background.md"}}],
  "route": {"next_step_id":"define_goals_and_problems"},
  "evidence": [],
  "summary": "requirements ready"
}`)
	assertSuccess(t, reported, "flow.report.result")
	assertFlowEffect(t, reported.stdout, "advanced", "define_goals_and_problems")
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

func assertOutputPresent(t *testing.T, content, key string) {
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
	if _, exists := envelope.Data.State.Outputs[key]; !exists {
		t.Fatalf("Loop response dropped newly accepted Output %q:\n%s", key, content)
	}
}
