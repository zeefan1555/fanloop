package runtime_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestPromotionDesignProgressAndLoopsInvalidateOutputs(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "promotion-design", "--title", "Promotion lifecycle"), "flow.init")

	progress := run(binary, "flow", "report", "progress", "--root", root,
		"--step-id", "clarify_requirements", "--status", "in_progress", "--summary", "clarifying")
	assertSuccess(t, progress, "flow.report.progress")
	assertFlowEffect(t, progress.stdout, "status_updated", "clarify_requirements")

	ready := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "clarify_requirements",
		"--condition-result", conditionResult("requirements_ready", "path", `"require_points.md"`),
		"--next-step-id", "confirm_requirements", "--summary", "requirements ready")
	assertSuccess(t, ready, "flow.report.result")
	assertFlowEffect(t, ready.stdout, "advanced", "confirm_requirements")

	flowState := readFile(t, filepath.Join(root, ".fanloop", "flow", "state.json"))
	if bytes.Contains(flowState, []byte(`"outputs"`)) {
		t.Fatalf("Flow State still embeds Outputs:\n%s", flowState)
	}
	registry := readFile(t, filepath.Join(root, ".fanloop", "output", "state.json"))
	for _, want := range []string{`"require_points_path"`, `"producer_step_id": "clarify_requirements"`} {
		if !bytes.Contains(registry, []byte(want)) {
			t.Fatalf("Output Registry does not contain %s:\n%s", want, registry)
		}
	}

	rejected := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "confirm_requirements",
		"--condition-result", conditionResult("requirements_rejected", "enum_value", `"rejected"`),
		"--back-step-id", "clarify_requirements", "--summary", "requirements need revision")
	assertSuccess(t, rejected, "flow.report.result")
	assertFlowEffect(t, rejected.stdout, "looped", "clarify_requirements")
	assertOutputAbsent(t, rejected.stdout, "require_points_path")

	ready = run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "clarify_requirements",
		"--condition-result", conditionResult("requirements_ready", "path", `"require_points.md"`),
		"--next-step-id", "confirm_requirements", "--summary", "requirements ready again")
	assertSuccess(t, ready, "flow.report.result")
	approved := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "confirm_requirements",
		"--condition-result", conditionResult("requirements_approved", "enum_value", `"approved"`),
		"--next-step-id", "write_promotion_design", "--summary", "requirements approved")
	assertSuccess(t, approved, "flow.report.result")
	written := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "write_promotion_design",
		"--condition-result", conditionResult("promotion_design_written", "path", `"方案.md"`),
		"--next-step-id", "review_promotion_design", "--summary", "design written")
	assertSuccess(t, written, "flow.report.result")
	reviewed := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "review_promotion_design",
		"--condition-result", conditionResult("design_review_failed", "enum_value", `"failed"`),
		"--condition-result", conditionResult("design_review_written", "path", `".promotion/review.md"`),
		"--back-step-id", "write_promotion_design", "--summary", "design needs revision")
	assertSuccess(t, reviewed, "flow.report.result")
	assertFlowEffect(t, reviewed.stdout, "looped", "write_promotion_design")
	assertOutputAbsent(t, reviewed.stdout, "promotion_design_path")
	if !strings.Contains(reviewed.stdout, `"require_points_path"`) {
		t.Fatalf("design loop removed approved Requirement Output: %s", reviewed.stdout)
	}

	events := string(readFile(t, filepath.Join(root, ".fanloop", "trace", "events.jsonl")))
	for _, fact := range []string{`"kind":"flow_progressed"`, `"effect":"advanced"`, `"effect":"looped"`, `"condition_id":"design_review_failed"`} {
		if !strings.Contains(events, fact) {
			t.Fatalf("Event audit missing %s:\n%s", fact, events)
		}
	}
}

func TestFlowResultAcceptsExplicitPromotionRoute(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "promotion-design", "--title", "Explicit route"), "flow.init")
	reported := run(binary, "flow", "report", "result", "--root", root, "--input", `{
  "step_id": "clarify_requirements",
  "condition_results": [{"condition_id":"requirements_ready","output":{"type":"path","value":"require_points.md"}}],
  "route": {"next_step_id":"confirm_requirements"},
  "evidence": [],
  "summary": "requirements ready"
}`)
	assertSuccess(t, reported, "flow.report.result")
	assertFlowEffect(t, reported.stdout, "advanced", "confirm_requirements")
}

func TestFlowReportRejectsRetiredCommandShapes(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "promotion-design", "--title", "No aliases"), "flow.init")
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
