package card

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/store"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

func TestWriteProjectionUsesPinnedSnapshotAfterCurrentMoves(t *testing.T) {
	base := t.TempDir()
	snapshotA := filepath.Join(base, "snapshot-a")
	snapshotB := filepath.Join(base, "snapshot-b")
	copyLinearRuntime(t, snapshotA)
	copyLinearRuntime(t, snapshotB)
	replaceWorkflowText(t, filepath.Join(snapshotA, "fanloop-workflow", "linear-runtime", "condition.yaml"), "key: first_artifact", "key: user_draft_approval")
	replaceWorkflowText(t, filepath.Join(snapshotB, "fanloop-workflow", "linear-runtime", "condition.yaml"), "key: first_artifact", "key: draft_approval")
	t.Setenv("FANLOOP_DATA_HOME", base)
	t.Setenv("FANLOOP_CONFIG_SOURCE", snapshotA)

	loaded, err := workflow.Load("linear-runtime")
	if err != nil {
		t.Fatal(err)
	}
	step, _ := loaded.Workflow.FirstStepID()
	now := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	current := state.State{
		SchemaVersion: state.CurrentStateSchemaVersion,
		Requirement:   state.Requirement{Title: "Pinned projection"},
		Release:       state.Release{Version: "dev", Workflow: state.WorkflowRefFrom(loaded.Ref)},
		CurrentStepID: &step, CurrentStepStatus: state.StepReady,
		Outputs: map[string]state.RegisteredOutput{
			"user_draft_approval": {Type: workflow.OutputPath, Value: json.RawMessage(`"draft.md"`), ProducerStepID: step},
		},
		LastEventID: "e1", CreatedAt: now, UpdatedAt: now,
	}
	root := t.TempDir()
	local, failure := store.New(root)
	if failure != nil {
		t.Fatal(failure)
	}
	if err := local.PinConfiguration(snapshotA); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FANLOOP_CONFIG_SOURCE", snapshotB)
	if err := WriteProjection(root, current); err != nil {
		t.Fatal(err)
	}
	projection, bound, err := LoadProjection(root)
	if err != nil {
		t.Fatal(err)
	}
	condition, ok := bound.Workflow.Condition("first_completed")
	if _, exists := projection.Outputs["user_draft_approval"]; !exists || !ok || condition.Output.Key != "user_draft_approval" {
		t.Fatalf("projection bound %q outputs %#v", condition.Output.Key, projection.Outputs)
	}

	unpinned := t.TempDir()
	if err := WriteProjection(unpinned, current); err == nil || !strings.Contains(err.Error(), `Output "user_draft_approval" is not declared`) {
		t.Fatalf("unpinned projection error = %v", err)
	}
}

func copyLinearRuntime(t *testing.T, dest string) {
	t.Helper()
	source := filepath.Join("..", "workflow", "testdata", "linear-runtime")
	if err := os.CopyFS(filepath.Join(dest, "fanloop-workflow", "linear-runtime"), os.DirFS(source)); err != nil {
		t.Fatal(err)
	}
}

func replaceWorkflowText(t *testing.T, path, old, updated string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), old) {
		t.Fatalf("%s does not contain %q", path, old)
	}
	if err := os.WriteFile(path, []byte(strings.ReplaceAll(string(content), old, updated)), 0o644); err != nil {
		t.Fatal(err)
	}
}
