package configuration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestGitDriftInSyncLocalBehindModifiedAndUnavailable(t *testing.T) {
	source := remoteFixture(t)
	data := t.TempDir()
	ctx := context.Background()
	synced, err := Sync(ctx, data)
	if err != nil {
		t.Fatal(err)
	}
	report, err := Drift(ctx, data)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != DriftInSync || report.ConfigurationCommit != synced.CommitSHA || report.RemoteCommit != synced.CommitSHA || len(report.Workflows) != 2 {
		t.Fatalf("in_sync = %+v", report)
	}
	if report.Workflows[0].WorkflowID != "linear-runtime" || report.Workflows[1].WorkflowID != "runtime-controls" || report.Workflows[0].Status != DriftInSync || report.Workflows[0].Detail != "" {
		t.Fatalf("workflows = %+v", report.Workflows)
	}

	extra := filepath.Join(source, Category, "linear-runtime", "notes.md")
	if err = os.WriteFile(extra, []byte("later\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	command(t, source, "add", ".")
	command(t, source, "commit", "-m", "advance")
	report, err = Drift(ctx, data)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != DriftLocalBehind || report.ConfigurationCommit != synced.CommitSHA || report.RemoteCommit == synced.CommitSHA || len(report.RemoteCommit) != 40 {
		t.Fatalf("local_behind = %+v", report)
	}
	if report.Workflows[0].Status != DriftLocalBehind || report.Workflows[0].Detail != "" {
		t.Fatalf("behind workflows = %+v", report.Workflows)
	}

	current, err := os.Readlink(filepath.Join(data, "config", "current"))
	if err != nil {
		t.Fatal(err)
	}
	edited := filepath.Join(current, "fanloop-workflow", "linear-runtime", "workflow.yaml")
	if err = os.Chmod(edited, 0o644); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(edited)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(edited, append(content, []byte("\n# edited\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err = Drift(ctx, data)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != DriftLocalModified || report.ConfigurationCommit != synced.CommitSHA || report.RemoteCommit == "" {
		t.Fatalf("local_modified = %+v", report)
	}
	for _, item := range report.Workflows {
		if item.Status != DriftLocalModified || item.Detail != "package content differs" {
			t.Fatalf("modified workflow = %+v", item)
		}
	}

	denyProductionRemotes(t)
	if err = os.WriteFile(edited, content, 0o644); err != nil {
		t.Fatal(err)
	}
	report, err = Drift(ctx, data)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != DriftUnavailable || report.ConfigurationCommit != synced.CommitSHA || report.RemoteCommit != "" {
		t.Fatalf("unavailable = %+v", report)
	}
	if report.Workflows[0].Status != DriftUnavailable || report.Workflows[0].Detail != "configuration fetch failed" {
		t.Fatalf("unavailable workflows = %+v", report.Workflows)
	}
}

func TestGitDriftFornaxOriginSnapshot(t *testing.T) {
	remoteFixture(t)
	data := t.TempDir()
	stage := t.TempDir()
	destination := filepath.Join(stage, "fanloop-workflow", "linear-runtime")
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(destination, os.DirFS(filepath.Join("..", "workflow", "testdata", "linear-runtime"))); err != nil {
		t.Fatal(err)
	}
	snapshot, err := PublishRelease(data, stage, []ReleaseRef{{
		Key: "fanloop.workflow.linear_runtime", Version: "1.0.0", WorkflowID: "linear-runtime",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err = Activate(data, snapshot); err != nil {
		t.Fatal(err)
	}
	report, err := Drift(context.Background(), data)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != DriftLocalBehind || report.ConfigurationCommit != "" || len(report.RemoteCommit) != 40 || len(report.Workflows) != 1 {
		t.Fatalf("fornax origin = %+v", report)
	}
	if report.Workflows[0].WorkflowID != "linear-runtime" || report.Workflows[0].Status != DriftLocalBehind || report.Workflows[0].Detail != "snapshot came from Fornax; run fanloop update config" {
		t.Fatalf("workflow = %+v", report.Workflows[0])
	}
}

func TestGitDriftRefusesAuthorSource(t *testing.T) {
	t.Setenv("FANLOOP_CONFIG_SOURCE", t.TempDir())
	_, err := Drift(context.Background(), t.TempDir())
	if !errors.Is(err, ErrDriftInput) {
		t.Fatalf("error = %v", err)
	}
}
