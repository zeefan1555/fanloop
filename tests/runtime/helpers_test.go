package runtime_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func conditionResult(id, outputType, valueJSON string) string {
	return fmt.Sprintf(`{"condition_id":%q,"output":{"type":%q,"value":%s}}`, id, outputType, valueJSON)
}

type result struct {
	exitCode int
	stdout   string
	stderr   string
}

func buildCLI(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "fanloop")
	command := exec.Command("go", "build", "-buildvcs=false", "-o", binary, ".")
	command.Dir = repositoryRoot(t)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build fanloop: %v\n%s", err, output)
	}
	seedLiveWorkflows(t, filepath.Join(filepath.Dir(binary), ".fanloop-user"))
	return binary
}

func seedLiveWorkflows(t *testing.T, dataRoot string) {
	t.Helper()
	target := filepath.Join(dataRoot, "current", "workflows")
	for _, id := range []string{"linear-runtime", "runtime-controls"} {
		source := filepath.Join(repositoryRoot(t), "internal", "workflow", "testdata", id)
		if err := os.CopyFS(filepath.Join(target, id), os.DirFS(source)); err != nil {
			t.Fatalf("seed synthetic Workflow %s: %v", id, err)
		}
	}
}

func run(binary string, args ...string) result {
	return collect(commandFor(binary, args...))
}

func commandFor(binary string, args ...string) *exec.Cmd {
	command := exec.Command(binary, args...)
	command.Env = withoutEnvironment(os.Environ(), "BOTMUX_CHAT_ID", "BOTMUX_SESSION_ID", "FANLOOP_DATA_HOME", "PATH")
	command.Env = append(command.Env,
		"FANLOOP_DATA_HOME="+filepath.Join(filepath.Dir(binary), ".fanloop-user"),
		"PATH="+filepath.Dir(binary),
	)
	return command
}

func withoutEnvironment(environment []string, keys ...string) []string {
	blocked := make(map[string]bool, len(keys))
	for _, key := range keys {
		blocked[key] = true
	}
	filtered := make([]string, 0, len(environment))
	for _, item := range environment {
		key, _, _ := strings.Cut(item, "=")
		if !blocked[key] {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func TestCommandForScrubsBotmuxBinding(t *testing.T) {
	t.Setenv("BOTMUX_CHAT_ID", "oc_real_chat_must_not_escape")
	t.Setenv("BOTMUX_SESSION_ID", "real_session_must_not_escape")
	t.Setenv("PATH", "/tmp/real-user-bin"+string(os.PathListSeparator)+os.Getenv("PATH"))
	command := commandFor("loop", "version")
	if command.Env == nil {
		t.Fatal("command inherits the process environment implicitly")
	}
	wantDataRoot := "FANLOOP_DATA_HOME=" + filepath.Join(filepath.Dir("loop"), ".fanloop-user")
	wantPath := "PATH=" + filepath.Dir("loop")
	foundDataRoot := false
	foundPath := false
	for _, item := range command.Env {
		if strings.HasPrefix(item, "BOTMUX_CHAT_ID=") || strings.HasPrefix(item, "BOTMUX_SESSION_ID=") {
			t.Fatalf("command inherited Botmux binding: %q", item)
		}
		foundDataRoot = foundDataRoot || item == wantDataRoot
		foundPath = foundPath || item == wantPath
		if strings.HasPrefix(item, "PATH=") && item != wantPath {
			t.Fatalf("command inherited an uncontrolled executable path: %q", item)
		}
	}
	if !foundDataRoot {
		t.Fatal("command inherited the real Loop data root")
	}
	if !foundPath {
		t.Fatal("command did not install the isolated executable path")
	}
}

func collect(command *exec.Cmd) result {
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	exitCode := 0
	if err := command.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			exitCode = exit.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return result{exitCode: exitCode, stdout: stdout.String(), stderr: stderr.String()}
}

func assertSuccess(t *testing.T, got result, command string) {
	t.Helper()
	if got.exitCode != 0 || got.stderr != "" {
		t.Fatalf("exit = %d, stderr = %s, stdout = %s", got.exitCode, got.stderr, got.stdout)
	}
	var envelope struct {
		OK   bool `json:"ok"`
		Meta struct {
			Command string `json:"command"`
		} `json:"meta"`
	}
	if err := json.Unmarshal([]byte(got.stdout), &envelope); err != nil || !envelope.OK || envelope.Meta.Command != command {
		t.Fatalf("invalid success envelope: %v\n%s", err, got.stdout)
	}
}

func assertError(t *testing.T, got result, exitCode int, code string) {
	t.Helper()
	if got.exitCode != exitCode || got.stdout != "" {
		t.Fatalf("exit = %d, stdout = %s, stderr = %s", got.exitCode, got.stdout, got.stderr)
	}
	var envelope struct {
		OK    bool `json:"ok"`
		Error struct {
			Code string `json:"code"`
			Hint string `json:"hint"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(got.stderr), &envelope); err != nil || envelope.OK || envelope.Error.Code != code || envelope.Error.Hint == "" {
		t.Fatalf("invalid error envelope: %v\n%s", err, got.stderr)
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func bindTestTrace(t *testing.T, binary, root, documentURL string) {
	t.Helper()
	assertSuccess(t, run(binary, "trace", "bind", "--root", root, "--document-url", documentURL), "trace.bind")
}

func ensureCurrentStepStarted(t *testing.T, binary, root, stepID string) {
	t.Helper()
	status := run(binary, "flow", "status", "--root", root)
	assertSuccess(t, status, "flow.status")
	var envelope struct {
		Data struct {
			State struct {
				Current struct {
					Context struct {
						StepID string `json:"step_id"`
					} `json:"context"`
					Execution struct {
						Status string `json:"status"`
					} `json:"execution"`
				} `json:"current"`
			} `json:"state"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(status.stdout), &envelope); err != nil {
		t.Fatal(err)
	}
	current := envelope.Data.State.Current
	if current.Context.StepID != stepID {
		t.Fatalf("current Step = %q, want %q", current.Context.StepID, stepID)
	}
	if current.Execution.Status != "awaiting_confirmation" {
		return
	}
	started := run(binary, "flow", "report", "result", "--root", root, "--step-id", stepID,
		"--condition-result", conditionResult("start_confirmed", "enum_value", `"confirmed"`),
		"--evidence", `{"source":"human","content":"confirmed"}`,
		"--summary", "step confirmed", "--start-current-step")
	assertSuccess(t, started, "flow.report.result")
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate repository")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
