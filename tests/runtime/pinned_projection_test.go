package runtime_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/configuration"
)

func TestReportAndRenderStayOnPinnedSnapshotAfterCurrentMoves(t *testing.T) {
	binary := buildCLI(t)
	dataHome := t.TempDir()
	sourceA := filepath.Join(dataHome, "source-a")
	sourceB := filepath.Join(dataHome, "source-b")
	copyLinearRuntime(t, sourceA)
	copyLinearRuntime(t, sourceB)
	replaceFile(t, filepath.Join(sourceA, "fanloop-workflow", "linear-runtime", "condition.yaml"), "key: first_artifact", "key: user_draft_approval")
	replaceFile(t, filepath.Join(sourceB, "fanloop-workflow", "linear-runtime", "condition.yaml"), "key: first_artifact", "key: draft_approval")
	replaceFile(t, filepath.Join(sourceB, "fanloop-workflow", "linear-runtime", "workflow.yaml"), "name: Second", "name: Second Renamed")
	snapshotA := publishSnapshot(t, dataHome, sourceA, "1.0.0")
	snapshotB := publishSnapshot(t, dataHome, sourceB, "2.0.0")

	root := t.TempDir()
	assertSuccess(t, collect(commandWithConfig(binary, dataHome, snapshotA, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Pinned outputs")), "flow.init")

	if err := configuration.Activate(dataHome, snapshotB); err != nil {
		t.Fatal(err)
	}

	reported := collect(commandWithConfig(binary, dataHome, "", "flow", "report", "result", "--root", root,
		"--step-id", "first", "--condition-result", conditionResult("first_completed", "path", `"draft.md"`),
		"--next-step-id", "second", "--summary", "first complete"))
	assertSuccess(t, reported, "flow.report.result")
	if strings.Contains(reported.stderr, "could not update") || strings.Contains(reported.stdout, "Second Renamed") || strings.Contains(reported.stdout, `"draft_approval"`) {
		t.Fatalf("report followed config/current:\nstdout: %s\nstderr: %s", reported.stdout, reported.stderr)
	}
	projection := readFile(t, filepath.Join(root, ".fanloop", "card", "projection.json"))
	if !strings.Contains(string(projection), `"user_draft_approval"`) || strings.Contains(string(projection), `"draft_approval"`) {
		t.Fatalf("projection did not keep the pinned output:\n%s", projection)
	}

	card := collect(commandWithConfig(binary, dataHome, "", "card", "render", "--root", root, "--view", "panorama", "--format", "markdown"))
	assertSuccess(t, card, "card.render")
	content := string(decodeCard(t, card.stdout).Data.Content)
	if !strings.Contains(content, "Second") || strings.Contains(content, "Second Renamed") {
		t.Fatalf("card render used config/current:\n%s", content)
	}

	flowStatus := collect(commandWithConfig(binary, dataHome, "", "flow", "status", "--root", root))
	assertSuccess(t, flowStatus, "flow.status")
	if !strings.Contains(flowStatus.stdout, `"step_name": "Second"`) || strings.Contains(flowStatus.stdout, "Second Renamed") {
		t.Fatalf("flow status used config/current:\n%s", flowStatus.stdout)
	}
	skills := collect(commandWithConfig(binary, dataHome, "", "flow", "skills", "--root", root))
	assertSuccess(t, skills, "flow.skills")
	if !strings.Contains(skills.stdout, snapshotA) || strings.Contains(skills.stdout, snapshotB) {
		t.Fatalf("flow skills used config/current:\n%s", skills.stdout)
	}
	traced := collect(commandWithConfig(binary, dataHome, "", "trace", "render", "--root", root, "--dry-run"))
	assertSuccess(t, traced, "trace.render")

	diagnosed := collect(commandWithConfig(binary, dataHome, "", "doctor", "--root", root))
	assertSuccess(t, diagnosed, "doctor")
	if workflows := doctorCheck(t, diagnosed.stdout, "workflows"); workflows.Status != "passed" || !strings.Contains(workflows.Summary, snapshotB) {
		t.Fatalf("installation doctor did not follow config/current: %#v", workflows)
	}
	for _, id := range []string{"workflow_binding", "state_invariants", "outputs", "events", "card_projection"} {
		check := doctorCheck(t, diagnosed.stdout, id)
		if check.Status != "passed" {
			t.Fatalf("doctor %s = %#v\n%s", id, check, diagnosed.stdout)
		}
	}
}

func publishSnapshot(t *testing.T, dataHome, source, version string) string {
	t.Helper()
	published, err := configuration.PublishRelease(dataHome, source, []configuration.ReleaseRef{{
		Key: "fanloop.workflow.linear_runtime", Version: version, WorkflowID: "linear-runtime",
	}})
	if err != nil {
		t.Fatal(err)
	}
	return published
}

func copyLinearRuntime(t *testing.T, dest string) {
	t.Helper()
	source := filepath.Join(repositoryRoot(t), "internal", "workflow", "testdata", "linear-runtime")
	if err := os.CopyFS(filepath.Join(dest, "fanloop-workflow", "linear-runtime"), os.DirFS(source)); err != nil {
		t.Fatal(err)
	}
}

func replaceFile(t *testing.T, path, old, updated string) {
	t.Helper()
	content := string(readFile(t, path))
	if !strings.Contains(content, old) {
		t.Fatalf("%s does not contain %q", path, old)
	}
	if err := os.WriteFile(path, []byte(strings.ReplaceAll(content, old, updated)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func commandWithConfig(binary, dataHome, configSource string, args ...string) *exec.Cmd {
	command := exec.Command(binary, args...)
	command.Env = withoutEnvironment(os.Environ(), "BOTMUX_CHAT_ID", "BOTMUX_SESSION_ID", "FANLOOP_DATA_HOME", "FANLOOP_CONFIG_SOURCE", "PATH")
	command.Env = append(command.Env, "FANLOOP_DATA_HOME="+dataHome, "PATH="+filepath.Dir(binary))
	if configSource != "" {
		command.Env = append(command.Env, "FANLOOP_CONFIG_SOURCE="+configSource)
	}
	return command
}
