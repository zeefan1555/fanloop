package runtime_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFlowWithoutPanoramaConditionRunsWithoutAcknowledgment(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	initialized := run(binary, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "No panorama gate")
	assertSuccess(t, initialized, "flow.init")
	if strings.Contains(initialized.stdout, "pending_actions") || strings.Contains(initialized.stdout, "panorama_delivery_required") {
		t.Fatalf("flow.init injected a Runtime action:\n%s", initialized.stdout)
	}

	progress := run(binary, "flow", "report", "progress", "--root", root,
		"--step-id", "first", "--status", "in_progress", "--summary", "working")
	assertSuccess(t, progress, "flow.report.progress")

	advanced := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "first", "--condition-result", conditionResult("first_completed", "path", `"first.md"`),
		"--next-step-id", "second", "--summary", "first complete")
	assertSuccess(t, advanced, "flow.report.result")
	if strings.Contains(advanced.stdout, "pending_actions") || strings.Contains(advanced.stdout, "panorama_delivery_required") {
		t.Fatalf("flow.report.result injected a Runtime action:\n%s", advanced.stdout)
	}
}

func TestWorkflowOwnedPanoramaConditionCanRenderAndRoute(t *testing.T) {
	binary := buildCLI(t)
	source := filepath.Join(repositoryRoot(t), "examples", "workflows", "example")
	target := filepath.Join(filepath.Dir(binary), ".fanloop-user", "source", "fanloop-workflow", "example")
	if err := os.CopyFS(target, os.DirFS(source)); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	initialized := run(binary, "flow", "init", "--root", root, "--workflow", "example", "--title", "Workflow panorama")
	assertSuccess(t, initialized, "flow.init")
	assertWorkflowPanoramaBinding(t, initialized.stdout)

	rendered := run(binary, "card", "render", "--root", root, "--view", "panorama", "--format", "markdown")
	assertSuccess(t, rendered, "card.render")
	snapshot := decodeCard(t, rendered.stdout).Data.SnapshotPath
	if snapshot == "" {
		t.Fatal("panorama render did not return snapshot_path")
	}

	advanced := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "analyze_materials",
		"--condition-result", conditionResult("materials_analyzed", "path", `"summary.md"`),
		"--condition-result", conditionResult("panorama_presented", "path", jsonString(snapshot)),
		"--next-step-id", "draft_result", "--summary", "materials analyzed and panorama delivered")
	assertSuccess(t, advanced, "flow.report.result")
	if !strings.Contains(advanced.stdout, `"step_id": "draft_result"`) {
		t.Fatalf("configured panorama route did not advance:\n%s", advanced.stdout)
	}
}

func assertWorkflowPanoramaBinding(t *testing.T, content string) {
	t.Helper()
	var envelope struct {
		Data struct {
			State struct {
				Current struct {
					Conditions []struct {
						ID     string `json:"id"`
						Prompt struct {
							Skills []struct {
								ID   string `json:"id"`
								Path string `json:"path"`
							} `json:"skills"`
						} `json:"prompt"`
					} `json:"conditions"`
				} `json:"current"`
			} `json:"state"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(content), &envelope); err != nil {
		t.Fatal(err)
	}
	for _, condition := range envelope.Data.State.Current.Conditions {
		if condition.ID != "panorama_presented" {
			continue
		}
		for _, skill := range condition.Prompt.Skills {
			if skill.ID == "example-panorama" && filepath.IsAbs(skill.Path) {
				return
			}
		}
		t.Fatalf("panorama condition has no bound delivery Skill: %#v", condition)
	}
	t.Fatal("flow state does not expose workflow-owned panorama condition")
}

func jsonString(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
