package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type cliResult struct {
	stdout   string
	stderr   string
	exitCode int
	err      error
}

func TestPublicFlowTraceCardDomainsRunWithoutPython(t *testing.T) {
	repository := repositoryRoot(t)
	binary := filepath.Join(t.TempDir(), "fanloop")
	build := exec.Command("go", "build", "-buildvcs=false", "-o", binary, ".")
	build.Dir = repository
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fanloop: %v\n%s", err, output)
	}

	root := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		command := exec.Command(binary, args...)
		command.Env = append(withoutBotmuxBinding(os.Environ()), "FANLOOP_PYTHON="+filepath.Join(root, "missing-python"))
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		if err := command.Run(); err != nil {
			t.Fatalf("fanloop %s: %v\nstdout: %s\nstderr: %s", strings.Join(args, " "), err, stdout.String(), stderr.String())
		}
		return stdout.String()
	}

	if output := run("flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "E2E"); !strings.Contains(output, `"command": "flow.init"`) {
		t.Fatalf("flow output = %s", output)
	}
	if output := run("flow", "report", "progress", "--step-id", "frame_technical_problem", "--status", "in_progress", "--summary", "working", "--root", root); !strings.Contains(output, `"effect": "status_updated"`) {
		t.Fatalf("flow progress result = %s", output)
	}
	if output := run("trace", "render", "--root", root); !strings.Contains(output, `"event_count": 2`) {
		t.Fatalf("trace output = %s", output)
	}
	if output := run("card", "render", "--root", root, "--view", "current", "--format", "markdown"); !strings.Contains(output, `"format": "markdown"`) {
		t.Fatalf("card output = %s", output)
	}
	for _, relative := range []string{".fanloop/flow/state.json", ".fanloop/trace/events.jsonl", ".fanloop/trace/events.md"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(relative))); err != nil {
			t.Fatalf("final commands did not write Driver layout %s: %v", relative, err)
		}
	}
	for _, deprecated := range []string{".fanloop/state.json", ".fanloop/events.jsonl", ".fanloop/events.md", ".prd-flow"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(deprecated))); !os.IsNotExist(err) {
			t.Fatalf("final commands wrote deprecated requirement path %s: %v", deprecated, err)
		}
	}
}

func TestE2ECommandsScrubBotmuxBindingByDefault(t *testing.T) {
	t.Setenv("BOTMUX_CHAT_ID", "oc_real_chat_must_not_escape")
	t.Setenv("BOTMUX_SESSION_ID", "real_session_must_not_escape")
	for _, item := range withoutBotmuxBinding(os.Environ()) {
		if strings.HasPrefix(item, "BOTMUX_CHAT_ID=") || strings.HasPrefix(item, "BOTMUX_SESSION_ID=") {
			t.Fatalf("E2E command inherited Botmux binding: %q", item)
		}
	}
}

func withoutBotmuxBinding(environment []string) []string {
	filtered := make([]string, 0, len(environment))
	for _, item := range environment {
		key, _, _ := strings.Cut(item, "=")
		if key != "BOTMUX_CHAT_ID" && key != "BOTMUX_SESSION_ID" {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func runCurrent(dataRoot, codexRoot, agentsRoot, registryURL string, args ...string) cliResult {
	command := exec.Command(filepath.Join(dataRoot, "current", "bin", "fanloop"), args...)
	command.Env = append(withoutBotmuxBinding(os.Environ()),
		"FANLOOP_DATA_HOME="+dataRoot,
		"FANLOOP_CODEX_SKILLS_ROOT="+codexRoot,
		"FANLOOP_AGENT_SKILLS_ROOT="+agentsRoot,
		"FANLOOP_TRAE_SKILLS_ROOT="+filepath.Join(agentsRoot, ".trae-skills"),
		"FANLOOP_CLAUDE_SKILLS_ROOT="+filepath.Join(agentsRoot, ".claude-skills"),
		"FANLOOP_NPM_REGISTRY="+registryURL,
	)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	exitCode := 0
	err := command.Run()
	if exit, ok := err.(*exec.ExitError); ok {
		exitCode = exit.ExitCode()
	} else if err != nil {
		exitCode = -1
	}
	return cliResult{stdout: stdout.String(), stderr: stderr.String(), exitCode: exitCode, err: err}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate repository")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
