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
	return binary
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
	command := commandFor("fanloop", "version")
	if command.Env == nil {
		t.Fatal("command inherits the process environment implicitly")
	}
	wantDataRoot := "FANLOOP_DATA_HOME=" + filepath.Join(filepath.Dir("fanloop"), ".fanloop-user")
	wantPath := "PATH=" + filepath.Dir("fanloop")
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
		t.Fatal("command inherited the real Fanloop data root")
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

func writeRepositoryScope(t *testing.T, root string) {
	t.Helper()
	directory := filepath.Join(root, ".techdesign")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "repo_scope.md"), []byte("repository scope\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "requirement_source.json"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func bindTestTrace(t *testing.T, binary, root, documentURL string) {
	t.Helper()
	assertSuccess(t, run(binary, "trace", "bind", "--root", root, "--document-url", documentURL), "trace.bind")
}

func reportBootstrap(binary, root, documentURL string) result {
	return run(binary, "flow", "report", "result",
		"--root", root,
		"--step-id", "bootstrap_techdesign",
		"--condition-result", conditionResult("requirement_source_resolved", "path", `".techdesign/requirement_source.json"`),
		"--condition-result", conditionResult("repository_workspace_prepared", "path", `".techdesign/repo_scope.md"`),
		"--condition-result", conditionResult("trace_document_bound", "url", `"`+documentURL+`"`),
		"--next-step-id", "clarify_requirements",
		"--summary", "TechDesign bootstrap complete",
	)
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate repository")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
