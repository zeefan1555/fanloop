package runtime_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFlowInitRejectsUnknownWorkflowWithStablePublicError(t *testing.T) {
	binary := buildCLI(t)
	root := t.TempDir()

	command := commandFor(binary, "flow", "init", "--root", root, "--workflow", "definitely-unknown", "--title", "Unknown Workflow")
	command.Env = append(command.Env, "FANLOOP_CONFIG_SOURCE="+filepath.Join(filepath.Dir(binary), ".fanloop-user", "source"))
	result := collect(command)

	assertError(t, result, 1, "WORKFLOW_NOT_FOUND")
	if strings.Contains(result.stderr, "lstat") {
		t.Fatalf("public error leaked filesystem details: %s", result.stderr)
	}
}

func TestFlowInitMapsLiveWorkflowSelectionErrors(t *testing.T) {
	binary := buildCLI(t)

	t.Run("invalid selector", func(t *testing.T) {
		result := run(binary, "flow", "init", "--root", t.TempDir(), "--workflow", "../bad", "--title", "Invalid Workflow")
		assertError(t, result, 2, "INVALID_ARGUMENT")
	})
	t.Run("invalid configuration source", func(t *testing.T) {
		command := commandFor(binary, "flow", "init", "--root", t.TempDir(), "--workflow", "linear-runtime", "--title", "Invalid Source")
		command.Env = append(command.Env, "FANLOOP_CONFIG_SOURCE=relative")
		result := collect(command)
		assertError(t, result, 5, "WORKFLOW_INVALID")
	})
	t.Run("missing configuration source", func(t *testing.T) {
		command := commandFor(binary, "flow", "init", "--root", t.TempDir(), "--workflow", "linear-runtime", "--title", "Missing Source")
		command.Env = append(command.Env, "FANLOOP_CONFIG_SOURCE="+filepath.Join(t.TempDir(), "missing"))
		result := collect(command)
		assertError(t, result, 5, "WORKFLOW_INVALID")
	})
	for _, test := range []struct {
		name string
		link func(*testing.T, string)
	}{
		{
			name: "symlinked Workflow directory",
			link: func(t *testing.T, configRoot string) {
				t.Helper()
				path := filepath.Join(configRoot, "linear-runtime")
				target := filepath.Join(t.TempDir(), "linear-runtime")
				if err := os.Rename(path, target); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "symlinked Workflow file",
			link: func(t *testing.T, configRoot string) {
				t.Helper()
				path := filepath.Join(configRoot, "linear-runtime", "workflow.yaml")
				target := filepath.Join(t.TempDir(), "workflow.yaml")
				if err := os.Rename(path, target); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "unreadable Workflow directory",
			link: func(t *testing.T, configRoot string) {
				t.Helper()
				path := filepath.Join(configRoot, "linear-runtime")
				if err := os.Chmod(path, 0); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(path, 0o700) })
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dataRoot := filepath.Join(filepath.Dir(binary), ".fanloop-user")
			if err := os.RemoveAll(dataRoot); err != nil {
				t.Fatal(err)
			}
			seedLiveWorkflows(t, dataRoot)
			configSource := filepath.Join(dataRoot, "source")
			test.link(t, filepath.Join(configSource, "fanloop-workflow"))
			command := commandFor(binary, "flow", "init", "--root", t.TempDir(), "--workflow", "linear-runtime", "--title", "Unsafe Source")
			command.Env = append(command.Env, "FANLOOP_CONFIG_SOURCE="+configSource)
			result := collect(command)
			assertError(t, result, 5, "WORKFLOW_INVALID")
		})
	}

	configRoot := filepath.Join(filepath.Dir(binary), ".fanloop-user", "source", "fanloop-workflow")
	for _, test := range []struct {
		name    string
		file    string
		mutate  func([]byte) []byte
		message string
	}{
		{name: "syntax", file: "workflow.yaml", mutate: func([]byte) []byte { return []byte(":") }, message: "workflow.yaml"},
		{name: "structure", file: "workflow.yaml", mutate: func(content []byte) []byte { return append(content, []byte("unknown: true\n")...) }, message: "field unknown not found"},
		{name: "Step", file: "workflow.yaml", mutate: replace("executor: agent", "executor: robot"), message: "not a valid Executor string"},
		{name: "Condition reference", file: "flow.yaml", mutate: replace("first_completed", "missing_condition"), message: "unknown or duplicate Condition"},
		{name: "Prompt reference", file: "flow.yaml", mutate: replace("prompt_id: advance", "prompt_id: missing_prompt"), message: "unknown Prompt"},
		{name: "Route target", file: "flow.yaml", mutate: replace("next_step_id: second", "next_step_id: missing_step"), message: "invalid next_step_id"},
		{name: "Skill reference", file: "prompt.yaml", mutate: replace("skills: []", "skills:\n      - {id: missing-skill, prompt: Missing skill, optional: false}"), message: "unknown Skill"},
	} {
		t.Run(test.name, func(t *testing.T) {
			dataRoot := filepath.Join(filepath.Dir(binary), ".fanloop-user")
			if err := os.RemoveAll(dataRoot); err != nil {
				t.Fatal(err)
			}
			seedLiveWorkflows(t, dataRoot)
			path := filepath.Join(configRoot, "linear-runtime", test.file)
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, test.mutate(content), 0o600); err != nil {
				t.Fatal(err)
			}

			root := t.TempDir()
			command := commandFor(binary, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Invalid Workflow")
			command.Env = append(command.Env, "FANLOOP_CONFIG_SOURCE="+filepath.Dir(configRoot))
			result := collect(command)
			assertError(t, result, 5, "WORKFLOW_INVALID")
			if !strings.Contains(result.stderr, test.message) {
				t.Fatalf("stderr = %s, want %q", result.stderr, test.message)
			}
			if _, err := os.Stat(filepath.Join(root, ".fanloop", "flow", "state.json")); !os.IsNotExist(err) {
				t.Fatalf("failed load created state: %v", err)
			}
		})
	}
}

func replace(old, new string) func([]byte) []byte {
	return func(content []byte) []byte {
		return []byte(strings.Replace(string(content), old, new, 1))
	}
}
