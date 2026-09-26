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
	command.Env = append(command.Env, "FANLOOP_CONFIG_SOURCE="+filepath.Join(filepath.Dir(binary), ".fanloop-user", "current"))
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

	t.Run("malformed bundle", func(t *testing.T) {
		workflowPath := filepath.Join(filepath.Dir(binary), ".fanloop-user", "current", "workflows", "linear-runtime", "workflow.yaml")
		if err := os.WriteFile(workflowPath, []byte(":"), 0o600); err != nil {
			t.Fatal(err)
		}
		result := run(binary, "flow", "init", "--root", t.TempDir(), "--workflow", "linear-runtime", "--title", "Malformed Workflow")
		assertError(t, result, 5, "WORKFLOW_INVALID")
	})
}
