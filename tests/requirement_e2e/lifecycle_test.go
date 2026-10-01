package requiremente2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type lifecycleStatus struct {
	Data struct {
		State struct {
			Status  string `json:"status"`
			Current *struct {
				Context struct {
					StepID string `json:"step_id"`
				} `json:"context"`
			} `json:"current"`
			Outputs map[string]struct {
				Value any `json:"value"`
			} `json:"outputs"`
		} `json:"state"`
	} `json:"data"`
}

func TestRequirementWorkflowE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("run through .github/scripts/run-requirement-validation")
	}
	binary := os.Getenv("FANLOOP_E2E_BINARY")
	if binary == "" {
		t.Fatal("FANLOOP_E2E_BINARY is required")
	}
	runRoot := os.Getenv("FANLOOP_E2E_RUN_ROOT")
	if runRoot == "" {
		runRoot = t.TempDir()
	}
	dataHome := filepath.Join(runRoot, "data")
	requirementRoot := filepath.Join(runRoot, "requirement")
	if err := os.MkdirAll(requirementRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate repository root")
	}
	repo := filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
	bundleRoot := filepath.Join(dataHome, "source", "skills", "linear-runtime")
	if err := os.MkdirAll(bundleRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(bundleRoot, os.DirFS(filepath.Join(repo, "internal", "workflow", "testdata", "linear-runtime"))); err != nil {
		t.Fatal(err)
	}

	run := func(args ...string) string {
		t.Helper()
		command := exec.Command(binary, args...)
		command.Env = lifecycleEnvironment(dataHome)
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("fanloop %s: %v\n%s", strings.Join(args, " "), err, output)
		}
		return string(output)
	}
	report := func(args ...string) {
		t.Helper()
		run(append(args, "--dry-run")...)
		run(args...)
	}
	status := func() lifecycleStatus {
		t.Helper()
		var result lifecycleStatus
		if err := json.Unmarshal([]byte(run("flow", "status", "--root", requirementRoot)), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}

	run("flow", "init", "--root", requirementRoot, "--workflow", "linear-runtime", "--title", "Synthetic lifecycle")
	run("flow", "report", "progress", "--root", requirementRoot, "--step-id", "first", "--status", "in_progress", "--summary", "first running")
	report("flow", "report", "result", "--root", requirementRoot, "--step-id", "first", "--condition-result", `{"condition_id":"first_completed","output":{"type":"path","value":"first.md"}}`, "--summary", "first complete", "--next-step-id", "second")
	report("flow", "report", "result", "--root", requirementRoot, "--step-id", "second", "--condition-result", `{"condition_id":"second_rework","output":{"type":"enum_value","value":"rework"}}`, "--summary", "second requests rework", "--back-step-id", "first")
	afterBackflow := status()
	if afterBackflow.Data.State.Current == nil || afterBackflow.Data.State.Current.Context.StepID != "first" {
		t.Fatalf("backflow did not return to first: %#v", afterBackflow.Data.State.Current)
	}
	if _, exists := afterBackflow.Data.State.Outputs["first_artifact"]; exists {
		t.Fatal("backflow did not invalidate the target Step output")
	}

	report("flow", "report", "result", "--root", requirementRoot, "--step-id", "first", "--condition-result", `{"condition_id":"first_completed","output":{"type":"path","value":"first-revised.md"}}`, "--summary", "first revised", "--next-step-id", "second")
	report("flow", "report", "result", "--root", requirementRoot, "--step-id", "second", "--condition-result", `{"condition_id":"second_completed","output":{"type":"path","value":"second.md"}}`, "--condition-result", `{"condition_id":"second_evidence","output":{"type":"path","value":"evidence.json"}}`, "--summary", "second complete", "--next-step-id", "third")
	report("flow", "report", "result", "--root", requirementRoot, "--step-id", "third", "--condition-result", `{"condition_id":"third_completed","output":{"type":"path","value":"third.md"}}`, "--summary", "third complete", "--terminal")
	completed := status()
	if completed.Data.State.Status != "completed" || completed.Data.State.Current != nil {
		t.Fatalf("lifecycle did not complete: %#v", completed.Data.State)
	}
	for key := range map[string]bool{"first_artifact": true, "second_artifact": true, "second_evidence_path": true, "third_artifact": true} {
		if _, ok := completed.Data.State.Outputs[key]; !ok {
			t.Errorf("completed lifecycle missing output %s", key)
		}
	}
	if !strings.Contains(run("card", "render", "--root", requirementRoot, "--view", "panorama", "--format", "markdown", "--dry-run"), "100%") {
		t.Fatal("completed panorama does not report 100%")
	}
	run("doctor", "--root", requirementRoot)

	reportPath := filepath.Join(runRoot, "E2E_REPORT.md")
	content := fmt.Sprintf("# Synthetic Requirement Lifecycle\n\n- Workflow: `linear-runtime`\n- Status: `completed`\n- Requirement Root: `%s`\n", requirementRoot)
	if err := os.WriteFile(reportPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func lifecycleEnvironment(dataHome string) []string {
	environment := make([]string, 0, len(os.Environ())+5)
	for _, item := range os.Environ() {
		if strings.HasPrefix(item, "FANLOOP_CONFIG_SOURCE=") || strings.HasPrefix(item, "FANLOOP_DATA_HOME=") || strings.HasPrefix(item, "FANLOOP_CODEX_SKILLS_ROOT=") || strings.HasPrefix(item, "FANLOOP_AGENT_SKILLS_ROOT=") || strings.HasPrefix(item, "FANLOOP_TRAE_SKILLS_ROOT=") || strings.HasPrefix(item, "FANLOOP_CLAUDE_SKILLS_ROOT=") || strings.HasPrefix(item, "BOTMUX_CHAT_ID=") || strings.HasPrefix(item, "BOTMUX_SESSION_ID=") {
			continue
		}
		environment = append(environment, item)
	}
	environment = append(environment,
		"FANLOOP_DATA_HOME="+dataHome,
		"FANLOOP_CONFIG_SOURCE="+filepath.Join(dataHome, "source"),
		"FANLOOP_CODEX_SKILLS_ROOT="+filepath.Join(dataHome, "skill-roots", "codex"),
		"FANLOOP_AGENT_SKILLS_ROOT="+filepath.Join(dataHome, "skill-roots", "agent"),
		"FANLOOP_TRAE_SKILLS_ROOT="+filepath.Join(dataHome, "skill-roots", "trae"),
		"FANLOOP_CLAUDE_SKILLS_ROOT="+filepath.Join(dataHome, "skill-roots", "claude"),
	)
	return environment
}
