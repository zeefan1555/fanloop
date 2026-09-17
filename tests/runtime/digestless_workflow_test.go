package runtime_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const oldWorkflowDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestCurrentSchemaRequirementContinuesAcrossWorkflowDigestChange(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "fanloop-maintainer", "--title", "Digestless continuation"), "flow.init")
	rewriteRequirementProvenance(t, root)

	status := run(binary, "flow", "status", "--root", root)
	assertSuccess(t, status, "flow.status")
	for _, want := range []string{`"step_id": "define_verification_contract"`, `"id": "fanloop-dev-bootstrap"`} {
		if !strings.Contains(status.stdout, want) {
			t.Fatalf("current Workflow projection is missing %s:\n%s", want, status.stdout)
		}
	}
	if !strings.Contains(status.stdout, oldWorkflowDigest) {
		t.Fatalf("status replaced persisted Workflow provenance:\n%s", status.stdout)
	}

	progress := run(binary, "flow", "report", "progress", "--root", root, "--step-id", "define_verification_contract", "--status", "in_progress", "--summary", "continue with current bundle")
	assertSuccess(t, progress, "flow.report.progress")
	assertSuccess(t, run(binary, "card", "render", "--root", root, "--view", "current", "--format", "markdown"), "card.render")
	assertSuccess(t, run(binary, "trace", "status", "--root", root), "trace.status")
	assertSuccess(t, run(binary, "trace", "sync", "--root", root), "trace.sync")

	doctor := run(binary, "doctor", "--root", root)
	if doctor.exitCode != 0 || strings.Contains(doctor.stdout, "workflow digest mismatch") || !strings.Contains(doctor.stdout, `"id": "workflow_binding"`) {
		t.Fatalf("doctor rejected digest-only provenance drift: exit=%d\nstdout=%s\nstderr=%s", doctor.exitCode, doctor.stdout, doctor.stderr)
	}
	assertPersistedProvenance(t, root)
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
			messages: []string{"removed_step", "define_verification_contract", "build_until_verified"},
		},
		{
			name: "invalid Output type",
			breakState: func(t *testing.T, root string) {
				mutateJSONObject(t, filepath.Join(root, ".fanloop", "output", "state.json"), func(value map[string]any) {
					value["outputs"].(map[string]any)["verification_contract_path"] = map[string]any{
						"type": "string", "value": "requirements.md", "producer_step_id": "define_verification_contract",
					}
				})
			},
			exitCode: 5,
			code:     "STATE_CORRUPT",
			messages: []string{`Output \"verification_contract_path\" type does not match its definition`},
		},
		{
			name: "invalid Output producer",
			breakState: func(t *testing.T, root string) {
				mutateJSONObject(t, filepath.Join(root, ".fanloop", "output", "state.json"), func(value map[string]any) {
					value["outputs"].(map[string]any)["verification_contract_path"] = map[string]any{
						"type": "path", "value": "requirements.md", "producer_step_id": "removed_step",
					}
				})
			},
			exitCode: 5,
			code:     "STATE_CORRUPT",
			messages: []string{`Output \"verification_contract_path\" has an unknown producer Step`},
		},
		{
			name: "invalid Output position",
			breakState: func(t *testing.T, root string) {
				mutateJSONObject(t, filepath.Join(root, ".fanloop", "output", "state.json"), func(value map[string]any) {
					value["outputs"].(map[string]any)["verification_contract_path"] = map[string]any{
						"type": "path", "value": "requirements.md", "producer_step_id": "define_verification_contract",
					}
				})
			},
			exitCode: 5,
			code:     "STATE_CORRUPT",
			messages: []string{`Output \"verification_contract_path\" is not valid at the current Step`},
		},
		{
			name:    "invalid historical Condition",
			prepare: prepareDigestDifferentRequirementAfterBootstrap,
			breakState: func(t *testing.T, root string) {
				mutateFlowResult(t, root, func(result map[string]any) {
					conditions := result["condition_results"].([]any)
					conditions[0].(map[string]any)["condition_id"] = "self_validation_passed"
				})
			},
			exitCode: 5,
			code:     "STATE_CORRUPT",
			messages: []string{`Condition \"self_validation_passed\" is not available at Step \"define_verification_contract\"`},
		},
		{
			name:    "historical transition tail mismatch",
			prepare: prepareDigestDifferentRequirementAfterBootstrap,
			breakState: func(t *testing.T, root string) {
				mutateFlowResult(t, root, func(result map[string]any) {
					result["transition"].(map[string]any)["to_step_id"] = "certify_candidate"
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
			} else {
				rewriteRequirementProvenance(t, root)
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

func TestDigestDifferentRequirementPreservesCorruptionInvariants(t *testing.T) {
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
					value["workflow"].(map[string]any)["digest"] = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
				})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			binary, root := buildCLI(t), t.TempDir()
			assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "fanloop-maintainer", "--title", test.name), "flow.init")
			rewriteRequirementProvenance(t, root)
			test.corrupt(t, root)
			paths, before := requirementFacts(t, root)

			assertError(t, run(binary, "flow", "status", "--root", root), 5, "STATE_CORRUPT")
			assertFactsUnchanged(t, paths, before)
		})
	}
}

func TestDigestDifferentRequirementDoesNotRevalidateHistoricalRoutes(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "fanloop-maintainer", "--title", "Historical route drift"), "flow.init")
	prepareDigestDifferentRequirementAfterBootstrap(t, binary, root)

	mutateFlowResult(t, root, func(result map[string]any) {
		conditions := result["condition_results"].([]any)
		result["condition_results"] = conditions[:1]
		result["output_changes"].(map[string]any)["accepted"] = []any{"verification_contract_path"}
	})
	for _, path := range []string{
		filepath.Join(root, ".fanloop", "output", "state.json"),
		filepath.Join(root, ".fanloop", "card", "projection.json"),
	} {
		mutateJSONObject(t, path, func(value map[string]any) {
			for _, key := range []string{"requirements_decision", "requirements_decision_receipt_id", "panorama_snapshot_path"} {
				delete(value["outputs"].(map[string]any), key)
			}
		})
	}

	assertSuccess(t, run(binary, "flow", "status", "--root", root), "flow.status")
	assertSuccess(t, run(binary, "card", "render", "--root", root, "--view", "current", "--format", "markdown", "--dry-run"), "card.render")

	paths, before := requirementFacts(t, root)
	currentResult := run(binary, "flow", "report", "result", "--root", root, "--step-id", "build_until_verified",
		"--condition-result", conditionResult("self_validation_passed", "enum_value", `"passed"`),
		"--next-step-id", "certify_candidate", "--summary", "missing current route gates")
	if currentResult.exitCode == 0 || !strings.Contains(currentResult.stderr, "ROUTE_NOT_MATCHED") {
		t.Fatalf("new Result bypassed current Route validation: exit=%d\nstdout=%s\nstderr=%s", currentResult.exitCode, currentResult.stdout, currentResult.stderr)
	}
	assertFactsUnchanged(t, paths, before)
}

func prepareDigestDifferentRequirementAfterBootstrap(t *testing.T, binary, root string) {
	t.Helper()
	assertSuccess(t, run(binary, "flow", "report", "result", "--root", root, "--step-id", "define_verification_contract",
		"--condition-result", conditionResult("verification_contract_written", "path", `"requirements.md"`),
		"--condition-result", conditionResult("requirements_approved", "enum_value", `"approved"`),
		"--condition-result", conditionResult("requirements_decision_recorded", "string", `"decision-requirements"`),
		"--condition-result", conditionResult("panorama_presented", "path", `".fanloop/card/define.json"`),
		"--next-step-id", "build_until_verified", "--summary", "verification contract complete"), "flow.report.result")
	rewriteRequirementProvenance(t, root)
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

func rewriteRequirementProvenance(t *testing.T, root string) {
	t.Helper()
	mutateJSONObject(t, filepath.Join(root, ".fanloop", "flow", "state.json"), func(value map[string]any) {
		value["release"].(map[string]any)["version"] = "0.5.0"
		value["release"].(map[string]any)["workflow"].(map[string]any)["digest"] = oldWorkflowDigest
	})
	mutateJSONObject(t, filepath.Join(root, ".fanloop", "output", "state.json"), func(value map[string]any) {
		value["workflow"].(map[string]any)["digest"] = oldWorkflowDigest
	})
	mutateEvents(t, root, func(event map[string]any) {
		event["workflow"].(map[string]any)["digest"] = oldWorkflowDigest
	})
	mutateJSONObject(t, filepath.Join(root, ".fanloop", "card", "projection.json"), func(value map[string]any) {
		value["release"].(map[string]any)["version"] = "0.5.0"
		value["release"].(map[string]any)["workflow"].(map[string]any)["digest"] = oldWorkflowDigest
	})
}

func assertPersistedProvenance(t *testing.T, root string) {
	t.Helper()
	for _, path := range []string{
		filepath.Join(root, ".fanloop", "flow", "state.json"),
		filepath.Join(root, ".fanloop", "output", "state.json"),
		filepath.Join(root, ".fanloop", "trace", "events.jsonl"),
		filepath.Join(root, ".fanloop", "card", "projection.json"),
	} {
		content := readFile(t, path)
		if !bytes.Contains(content, []byte(oldWorkflowDigest)) {
			t.Fatalf("persisted provenance changed in %s:\n%s", path, content)
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
