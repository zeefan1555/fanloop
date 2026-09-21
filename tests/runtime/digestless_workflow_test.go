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
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "fanloop-maintainer", "--title", "Live reload"), "flow.init")
	liveWorkflow := filepath.Join(filepath.Dir(binary), ".fanloop-user", "current", "workflows", "fanloop-maintainer")
	promptPath := filepath.Join(liveWorkflow, "prompt.yaml")
	prompt := readFile(t, promptPath)
	marker := "live prompt reload marker"
	updated := bytes.Replace(prompt, []byte("准备并回读独立 Issue Workspace"), []byte(marker), 1)
	if bytes.Equal(updated, prompt) {
		t.Fatal("prompt fixture replacement did not match")
	}
	if err := os.WriteFile(promptPath, updated, 0o600); err != nil {
		t.Fatal(err)
	}
	skillPath := filepath.Join(liveWorkflow, "skills", "fanloop-dev-bootstrap", "SKILL.md")
	skillMarker := "live skill reload marker"
	if err := os.WriteFile(skillPath, append(readFile(t, skillPath), []byte("\n"+skillMarker+"\n")...), 0o600); err != nil {
		t.Fatal(err)
	}

	status := run(binary, "flow", "status", "--root", root)
	assertSuccess(t, status, "flow.status")
	for _, want := range []string{`"step_id": "bootstrap_techdesign"`, `"id": "fanloop-dev-bootstrap"`, marker, skillPath} {
		if !strings.Contains(status.stdout, want) {
			t.Fatalf("live Workflow projection is missing %s:\n%s", want, status.stdout)
		}
	}
	if !strings.Contains(string(readFile(t, skillPath)), skillMarker) {
		t.Fatal("live Skill edit was not preserved")
	}

	progress := run(binary, "flow", "report", "progress", "--root", root, "--step-id", "bootstrap_techdesign", "--status", "in_progress", "--summary", "continue with live bundle")
	assertSuccess(t, progress, "flow.report.progress")
	assertSuccess(t, run(binary, "card", "render", "--root", root, "--view", "current", "--format", "markdown"), "card.render")
	assertSuccess(t, run(binary, "trace", "status", "--root", root), "trace.status")
	assertSuccess(t, run(binary, "trace", "sync", "--root", root), "trace.sync")

	doctor := run(binary, "doctor", "--root", root)
	if doctor.exitCode != 0 || !strings.Contains(doctor.stdout, `"id": "workflow_binding"`) {
		t.Fatalf("doctor rejected compatible live reload: exit=%d\nstdout=%s\nstderr=%s", doctor.exitCode, doctor.stdout, doctor.stderr)
	}
}

func TestCurrentSchemaRequirementRejectsUnsafeWorkflowChangeWithoutWriting(t *testing.T) {
	tests := []struct {
		name       string
		prepare    func(*testing.T, string, string)
		breakState func(*testing.T, string)
		exitCode   int
		code       string
		messages   []string
	}{
		{
			name: "missing current Step",
			breakState: func(t *testing.T, root string) {
				mutateJSONObject(t, filepath.Join(root, ".fanloop", "flow", "state.json"), func(value map[string]any) {
					value["current_step_id"] = "removed_step"
				})
			},
			exitCode: 1,
			code:     "WORKFLOW_MISMATCH",
			messages: []string{"removed_step", "bootstrap_techdesign", "clarify_requirements"},
		},
		{
			name: "invalid Output type",
			breakState: func(t *testing.T, root string) {
				mutateJSONObject(t, filepath.Join(root, ".fanloop", "output", "state.json"), func(value map[string]any) {
					value["outputs"].(map[string]any)["issue_workspace_path"] = map[string]any{
						"type": "string", "value": "issue-workspace", "producer_step_id": "bootstrap_techdesign",
					}
				})
			},
			exitCode: 5,
			code:     "STATE_CORRUPT",
			messages: []string{`Output \"issue_workspace_path\" type does not match its definition`},
		},
		{
			name: "invalid Output producer",
			breakState: func(t *testing.T, root string) {
				mutateJSONObject(t, filepath.Join(root, ".fanloop", "output", "state.json"), func(value map[string]any) {
					value["outputs"].(map[string]any)["issue_workspace_path"] = map[string]any{
						"type": "path", "value": "issue-workspace", "producer_step_id": "removed_step",
					}
				})
			},
			exitCode: 5,
			code:     "STATE_CORRUPT",
			messages: []string{`Output \"issue_workspace_path\" has an unknown producer Step`},
		},
		{
			name: "invalid Output position",
			breakState: func(t *testing.T, root string) {
				mutateJSONObject(t, filepath.Join(root, ".fanloop", "output", "state.json"), func(value map[string]any) {
					value["outputs"].(map[string]any)["issue_workspace_path"] = map[string]any{
						"type": "path", "value": "issue-workspace", "producer_step_id": "bootstrap_techdesign",
					}
				})
			},
			exitCode: 5,
			code:     "STATE_CORRUPT",
			messages: []string{`Output \"issue_workspace_path\" is not valid at the current Step`},
		},
		{
			name:    "invalid historical Condition Output",
			prepare: prepareRequirementAfterBootstrap,
			breakState: func(t *testing.T, root string) {
				mutateFlowResult(t, root, func(result map[string]any) {
					conditions := result["condition_results"].([]any)
					conditions[0].(map[string]any)["condition_id"] = "implementation_completed"
				})
			},
			exitCode: 5,
			code:     "STATE_CORRUPT",
			messages: []string{`Condition \"implementation_completed\" has invalid Output`},
		},
		{
			name:    "historical transition tail mismatch",
			prepare: prepareRequirementAfterBootstrap,
			breakState: func(t *testing.T, root string) {
				mutateFlowResult(t, root, func(result map[string]any) {
					result["transition"].(map[string]any)["to_step_id"] = "design_technical_solution"
				})
			},
			exitCode: 5,
			code:     "STATE_CORRUPT",
			messages: []string{"event history tail does not match current State"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			binary, root := buildCLI(t), t.TempDir()
			assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "fanloop-maintainer", "--title", test.name), "flow.init")
			if test.prepare != nil {
				test.prepare(t, binary, root)
			}
			test.breakState(t, root)
			paths, before := requirementFacts(t, root)

			got := run(binary, "flow", "status", "--root", root)
			assertError(t, got, test.exitCode, test.code)
			for _, message := range test.messages {
				if !strings.Contains(got.stderr, message) {
					t.Fatalf("error does not contain %q:\n%s", message, got.stderr)
				}
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
		{
			name: "malformed event JSON",
			corrupt: func(t *testing.T, root string) {
				path := filepath.Join(root, ".fanloop", "trace", "events.jsonl")
				if err := os.WriteFile(path, []byte("not-json\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "payload union mismatch",
			corrupt: func(t *testing.T, root string) {
				mutateEvents(t, root, func(event map[string]any) {
					event["payload"] = map[string]any{"trace_synced": map[string]any{"outcome": "skipped", "targets": []any{}}}
				})
			},
		},
		{
			name: "causal chain mismatch",
			corrupt: func(t *testing.T, root string) {
				mutateEvents(t, root, func(event map[string]any) { event["caused_by_event_id"] = "unexpected" })
			},
		},
		{
			name: "replay tail mismatch",
			corrupt: func(t *testing.T, root string) {
				mutateJSONObject(t, filepath.Join(root, ".fanloop", "flow", "state.json"), func(value map[string]any) {
					value["current_step_summary"] = "forged"
				})
			},
		},
		{
			name: "Output registry binding mismatch",
			corrupt: func(t *testing.T, root string) {
				mutateJSONObject(t, filepath.Join(root, ".fanloop", "output", "state.json"), func(value map[string]any) {
					value["workflow"].(map[string]any)["id"] = "material-flashcards"
				})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			binary, root := buildCLI(t), t.TempDir()
			assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "fanloop-maintainer", "--title", test.name), "flow.init")
			test.corrupt(t, root)
			paths, before := requirementFacts(t, root)

			assertError(t, run(binary, "flow", "status", "--root", root), 5, "STATE_CORRUPT")
			assertFactsUnchanged(t, paths, before)
		})
	}
}

func prepareRequirementAfterBootstrap(t *testing.T, binary, root string) {
	t.Helper()
	assertSuccess(t, run(binary, "flow", "report", "result", "--root", root, "--step-id", "bootstrap_techdesign",
		"--condition-result", conditionResult("repository_workspace_prepared", "path", `"issue-workspace"`),
		"--condition-result", conditionResult("panorama_presented", "path", `".fanloop/card/bootstrap.json"`),
		"--next-step-id", "clarify_requirements", "--summary", "bootstrap complete"), "flow.report.result")
}

func mutateFlowResult(t *testing.T, root string, mutate func(map[string]any)) {
	t.Helper()
	found := false
	mutateEvents(t, root, func(event map[string]any) {
		if event["kind"] != "flow_result" {
			return
		}
		mutate(event["payload"].(map[string]any)["flow_result"].(map[string]any))
		found = true
	})
	if !found {
		t.Fatal("flow_result event not found")
	}
}

func requirementFacts(t *testing.T, root string) ([]string, [][]byte) {
	t.Helper()
	paths := []string{
		filepath.Join(root, ".fanloop", "flow", "state.json"),
		filepath.Join(root, ".fanloop", "output", "state.json"),
		filepath.Join(root, ".fanloop", "trace", "events.jsonl"),
	}
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
	content = append(content, '\n')
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
}
