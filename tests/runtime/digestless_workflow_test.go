package runtime_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCurrentSchemaRequirementReloadsLiveWorkflowAndSkill(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "runtime-controls", "--title", "Live reload"), "flow.init")
	liveWorkflow := filepath.Join(filepath.Dir(binary), ".fanloop-user", "source", "skills", "fanloop-workflow", "runtime-controls")
	promptPath := filepath.Join(liveWorkflow, "prompt.yaml")
	prompt := readFile(t, promptPath)
	marker := "live prompt reload marker"
	updated := bytes.Replace(prompt, []byte("Start the current Step after human confirmation."), []byte(marker), 1)
	if bytes.Equal(updated, prompt) {
		t.Fatal("prompt fixture replacement did not match")
	}
	if err := os.WriteFile(promptPath, updated, 0o600); err != nil {
		t.Fatal(err)
	}
	skillPath := filepath.Join(liveWorkflow, "skills", "common-control-guide", "SKILL.md")
	skillMarker := "live skill reload marker"
	if err := os.WriteFile(skillPath, append(readFile(t, skillPath), []byte("\n"+skillMarker+"\n")...), 0o600); err != nil {
		t.Fatal(err)
	}

	status := run(binary, "flow", "status", "--root", root)
	assertSuccess(t, status, "flow.status")
	for _, want := range []string{`"step_id": "first"`, `"id": "common-control-guide"`, marker, skillPath} {
		if !strings.Contains(status.stdout, want) {
			t.Fatalf("live Workflow projection is missing %s:\n%s", want, status.stdout)
		}
	}
	ensureCurrentStepStarted(t, binary, root, "first")
	assertSuccess(t, run(binary, "flow", "report", "progress", "--root", root, "--step-id", "first", "--status", "in_progress", "--summary", "continue"), "flow.report.progress")
	assertSuccess(t, run(binary, "card", "render", "--root", root, "--view", "current", "--format", "markdown"), "card.render")
	assertSuccess(t, run(binary, "trace", "status", "--root", root), "trace.status")
	assertSuccess(t, run(binary, "trace", "sync", "--root", root), "trace.sync")
	if doctor := run(binary, "doctor", "--root", root); doctor.exitCode != 0 || !strings.Contains(doctor.stdout, `"id": "workflow_binding"`) {
		t.Fatalf("doctor rejected compatible live reload: exit=%d\nstdout=%s\nstderr=%s", doctor.exitCode, doctor.stdout, doctor.stderr)
	}
}

func TestCurrentSchemaRequirementRejectsUnsafeWorkflowChangeWithoutWriting(t *testing.T) {
	tests := []struct {
		name     string
		prepare  bool
		corrupt  func(*testing.T, string)
		exitCode int
		code     string
		message  string
	}{
		{
			name: "missing current Step", exitCode: 1, code: "WORKFLOW_MISMATCH", message: "removed_step",
			corrupt: func(t *testing.T, root string) {
				mutateJSONObject(t, filepath.Join(root, ".fanloop", "flow", "state.json"), func(value map[string]any) { value["current_step_id"] = "removed_step" })
			},
		},
		{
			name: "invalid Output type", prepare: true, exitCode: 5, code: "STATE_CORRUPT", message: "first_artifact",
			corrupt: func(t *testing.T, root string) {
				mutateJSONObject(t, filepath.Join(root, ".fanloop", "output", "state.json"), func(value map[string]any) {
					value["outputs"].(map[string]any)["first_artifact"].(map[string]any)["type"] = "string"
				})
			},
		},
		{
			name: "invalid historical Condition Output", prepare: true, exitCode: 5, code: "STATE_CORRUPT", message: "accepted Output changes do not match ConditionResults",
			corrupt: func(t *testing.T, root string) {
				mutateFlowResult(t, root, func(result map[string]any) {
					result["condition_results"].([]any)[0].(map[string]any)["condition_id"] = "second_completed"
				})
			},
		},
		{
			name: "historical transition tail mismatch", prepare: true, exitCode: 5, code: "STATE_CORRUPT", message: "event history tail does not match current State",
			corrupt: func(t *testing.T, root string) {
				mutateFlowResult(t, root, func(result map[string]any) { result["transition"].(map[string]any)["to_step_id"] = "third" })
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			binary, root := buildCLI(t), t.TempDir()
			assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", test.name), "flow.init")
			if test.prepare {
				prepareRequirementAfterFirstStep(t, binary, root)
			}
			test.corrupt(t, root)
			paths, before := requirementFacts(t, root)
			got := run(binary, "flow", "status", "--root", root)
			assertError(t, got, test.exitCode, test.code)
			if !strings.Contains(got.stderr, test.message) {
				t.Fatalf("error does not contain %q:\n%s", test.message, got.stderr)
			}
			assertFactsUnchanged(t, paths, before)
		})
	}
}

func TestLiveWorkflowRequirementPreservesCorruptionInvariants(t *testing.T) {
	tests := []struct {
		name    string
		corrupt func(*testing.T, string)
	}{
		{"malformed event JSON", func(t *testing.T, root string) {
			_ = os.WriteFile(filepath.Join(root, ".fanloop", "trace", "events.jsonl"), []byte("not-json\n"), 0o600)
		}},
		{"payload union mismatch", func(t *testing.T, root string) {
			mutateEvents(t, root, func(event map[string]any) {
				event["payload"] = map[string]any{"trace_synced": map[string]any{"outcome": "skipped", "targets": []any{}}}
			})
		}},
		{"causal chain mismatch", func(t *testing.T, root string) {
			mutateEvents(t, root, func(event map[string]any) { event["caused_by_event_id"] = "unexpected" })
		}},
		{"replay tail mismatch", func(t *testing.T, root string) {
			mutateJSONObject(t, filepath.Join(root, ".fanloop", "flow", "state.json"), func(value map[string]any) { value["current_step_summary"] = "forged" })
		}},
		{"Output registry binding mismatch", func(t *testing.T, root string) {
			mutateJSONObject(t, filepath.Join(root, ".fanloop", "output", "state.json"), func(value map[string]any) { value["workflow"].(map[string]any)["id"] = "runtime-controls" })
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			binary, root := buildCLI(t), t.TempDir()
			assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", test.name), "flow.init")
			test.corrupt(t, root)
			paths, before := requirementFacts(t, root)
			assertError(t, run(binary, "flow", "status", "--root", root), 5, "STATE_CORRUPT")
			assertFactsUnchanged(t, paths, before)
		})
	}
}

func TestCurrentSchemaRequirementDoesNotRevalidateHistoricalRoutes(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Historical route drift"), "flow.init")
	prepareRequirementAfterFirstStep(t, binary, root)
	liveFlow := filepath.Join(filepath.Dir(binary), ".fanloop-user", "source", "skills", "fanloop-workflow", "linear-runtime", "flow.yaml")
	content := readFile(t, liveFlow)
	changed := bytes.Replace(content, []byte("[[first_completed]]"), []byte("[[first_completed, second_evidence]]"), 1)
	if bytes.Equal(changed, content) {
		t.Fatal("flow fixture replacement did not match")
	}
	if err := os.WriteFile(liveFlow, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	assertSuccess(t, run(binary, "flow", "status", "--root", root), "flow.status")
	assertSuccess(t, run(binary, "card", "render", "--root", root, "--view", "current", "--format", "markdown", "--dry-run"), "card.render")

	paths, before := requirementFacts(t, root)
	currentResult := run(binary, "flow", "report", "result", "--root", root, "--step-id", "second",
		"--condition-result", conditionResult("second_completed", "path", `"second.md"`),
		"--next-step-id", "third", "--summary", "missing current route gate")
	if currentResult.exitCode == 0 || !strings.Contains(currentResult.stderr, "ROUTE_NOT_MATCHED") {
		t.Fatalf("new Result bypassed current Route validation: exit=%d\nstdout=%s\nstderr=%s", currentResult.exitCode, currentResult.stdout, currentResult.stderr)
	}
	assertFactsUnchanged(t, paths, before)
}

func prepareRequirementAfterFirstStep(t *testing.T, binary, root string) {
	t.Helper()
	assertSuccess(t, run(binary, "flow", "report", "result", "--root", root, "--step-id", "first",
		"--condition-result", conditionResult("first_completed", "path", `"first.md"`),
		"--next-step-id", "second", "--summary", "first complete"), "flow.report.result")
}

func mutateFlowResult(t *testing.T, root string, mutate func(map[string]any)) {
	t.Helper()
	found := false
	mutateEvents(t, root, func(event map[string]any) {
		if event["kind"] == "flow_result" {
			mutate(event["payload"].(map[string]any)["flow_result"].(map[string]any))
			found = true
		}
	})
	if !found {
		t.Fatal("flow_result event not found")
	}
}

func requirementFacts(t *testing.T, root string) ([]string, [][]byte) {
	t.Helper()
	paths := []string{filepath.Join(root, ".fanloop", "flow", "state.json"), filepath.Join(root, ".fanloop", "output", "state.json"), filepath.Join(root, ".fanloop", "trace", "events.jsonl")}
	before := make([][]byte, len(paths))
	for index, path := range paths {
		before[index] = readFile(t, path)
	}
	return paths, before
}

func assertFactsUnchanged(t *testing.T, paths []string, before [][]byte) {
	t.Helper()
	for index, path := range paths {
		if after := readFile(t, path); !bytes.Equal(after, before[index]) {
			t.Fatalf("rejected command changed %s", path)
		}
	}
}

func mutateEvents(t *testing.T, root string, mutate func(map[string]any)) {
	t.Helper()
	path := filepath.Join(root, ".fanloop", "trace", "events.jsonl")
	lines := bytes.Split(bytes.TrimSpace(readFile(t, path)), []byte{'\n'})
	var output bytes.Buffer
	for _, line := range lines {
		var event map[string]any
		if err := json.Unmarshal(line, &event); err != nil {
			t.Fatal(err)
		}
		mutate(event)
		encoded, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		output.Write(encoded)
		output.WriteByte('\n')
	}
	if err := os.WriteFile(path, output.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

func mutateJSONObject(t *testing.T, path string, mutate func(map[string]any)) {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(readFile(t, path), &value); err != nil {
		t.Fatal(err)
	}
	mutate(value)
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(content, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}
