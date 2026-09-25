package runtime_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyntheticWorkflowProgressAndLoopInvalidateOutputs(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Synthetic lifecycle"), "flow.init")

	progress := run(binary, "flow", "report", "progress", "--root", root, "--step-id", "first", "--status", "in_progress", "--summary", "working")
	assertSuccess(t, progress, "flow.report.progress")
	assertFlowEffect(t, progress.stdout, "status_updated", "first")

	advanced := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "first", "--condition-result", conditionResult("first_completed", "path", `"first.md"`),
		"--next-step-id", "second", "--summary", "first complete")
	assertSuccess(t, advanced, "flow.report.result")
	assertFlowEffect(t, advanced.stdout, "advanced", "second")
	assertCurrentExecutionStatus(t, advanced.stdout, "ready")
	advancedCard := run(binary, "card", "render", "--root", root, "--view", "panorama", "--format", "lark-json", "--dry-run")
	assertSuccess(t, advancedCard, "card.render")
	if content := decodeCard(t, advancedCard.stdout).Data.Content; !bytes.Contains(content, []byte("🚧 当前进行中")) {
		t.Fatalf("advanced Panorama does not retain the ready action:\n%s", content)
	}

	flowState := readFile(t, filepath.Join(root, ".fanloop", "flow", "state.json"))
	if bytes.Contains(flowState, []byte(`"outputs"`)) {
		t.Fatalf("Flow State still embeds Outputs:\n%s", flowState)
	}
	registry := readFile(t, filepath.Join(root, ".fanloop", "output", "state.json"))
	for _, want := range []string{`"first_artifact"`, `"producer_step_id": "first"`} {
		if !bytes.Contains(registry, []byte(want)) {
			t.Fatalf("Output Registry does not contain %s:\n%s", want, registry)
		}
	}

	looped := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "second", "--condition-result", conditionResult("second_rework", "enum_value", `"rework"`),
		"--back-step-id", "first", "--summary", "revise first")
	assertSuccess(t, looped, "flow.report.result")
	assertFlowEffect(t, looped.stdout, "looped", "first")
	assertCurrentExecutionStatus(t, looped.stdout, "fixing")
	assertOutputAbsent(t, looped.stdout, "first_artifact")
	assertOutputPresent(t, looped.stdout, "second_revision")

	markdownResult := run(binary, "card", "render", "--root", root, "--view", "panorama", "--format", "markdown", "--dry-run")
	assertSuccess(t, markdownResult, "card.render")
	var markdown string
	if err := json.Unmarshal(decodeCard(t, markdownResult.stdout).Data.Content, &markdown); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(markdown, "🔄 正在修复") || strings.Contains(markdown, "🚧 当前进行中") {
		t.Fatalf("looped Markdown Panorama does not show the fixing action:\n%s", markdown)
	}

	larkResult := run(binary, "card", "render", "--root", root, "--view", "panorama", "--format", "lark-json", "--dry-run")
	assertSuccess(t, larkResult, "card.render")
	if content := decodeCard(t, larkResult.stdout).Data.Content; !bytes.Contains(content, []byte("🔄 正在修复")) || bytes.Contains(content, []byte("🚧 当前进行中")) {
		t.Fatalf("looped Lark Panorama does not show the fixing action:\n%s", content)
	}

	events := string(readFile(t, filepath.Join(root, ".fanloop", "trace", "events.jsonl")))
	for _, fact := range []string{`"kind":"flow_progressed"`, `"effect":"advanced"`, `"effect":"looped"`, `"condition_id":"second_rework"`} {
		if !strings.Contains(events, fact) {
			t.Fatalf("Event audit missing %s:\n%s", fact, events)
		}
	}

	assertSuccess(t, run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "first", "--condition-result", conditionResult("first_completed", "path", `"first-again.md"`),
		"--next-step-id", "second", "--summary", "first complete again"), "flow.report.result")
	assertSuccess(t, run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "second",
		"--condition-result", conditionResult("second_completed", "path", `"second.md"`),
		"--condition-result", conditionResult("second_evidence", "path", `"evidence.md"`),
		"--next-step-id", "third", "--summary", "second complete"), "flow.report.result")
	completed := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "third", "--condition-result", conditionResult("third_completed", "path", `"third.md"`),
		"--terminal", "--summary", "workflow complete")
	assertSuccess(t, completed, "flow.report.result")
	assertFlowEffect(t, completed.stdout, "completed", "")
	status := run(binary, "flow", "status", "--root", root)
	assertSuccess(t, status, "flow.status")
	if strings.Contains(status.stdout, `"current"`) || !strings.Contains(status.stdout, `"status": "completed"`) {
		t.Fatalf("completed replay tail does not match State:\n%s", status.stdout)
	}
}

func TestFlowResultAcceptsExplicitSyntheticRoute(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Explicit route"), "flow.init")
	reported := run(binary, "flow", "report", "result", "--root", root, "--input", `{
  "step_id": "first",
  "condition_results": [{"condition_id":"first_completed","output":{"type":"path","value":"first.md"}}],
  "route": {"next_step_id":"second"},
  "evidence": [],
  "summary": "first complete"
}`)
	assertSuccess(t, reported, "flow.report.result")
	assertFlowEffect(t, reported.stdout, "advanced", "second")
}

func TestFlowReportRejectsRetiredCommandShapes(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "No aliases"), "flow.init")
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

func assertCurrentExecutionStatus(t *testing.T, content, status string) {
	t.Helper()
	var envelope struct {
		Data struct {
			State struct {
				Current *struct {
					Execution struct {
						Status string `json:"status"`
					} `json:"execution"`
				} `json:"current"`
			} `json:"state"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(content), &envelope); err != nil || envelope.Data.State.Current == nil || envelope.Data.State.Current.Execution.Status != status {
		t.Fatalf("current execution status = %#v, want %q, error = %v\n%s", envelope.Data.State.Current, status, err, content)
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
