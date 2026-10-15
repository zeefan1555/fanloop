package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/configuration"
)

func TestCommandPathCheckWarnsForDistributionInstall(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FANLOOP_DATA_HOME", root)
	if err := os.MkdirAll(filepath.Join(root, "releases", "1.2.3", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("releases", "1.2.3"), filepath.Join(root, "current")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())
	check := (Runtime{}).commandPathCheck()
	if check == nil || check.Status != statusWarn || check.Hint == nil || !strings.Contains(*check.Hint, `export PATH="`) {
		t.Fatalf("missing path check = %+v", check)
	}
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "fanloop"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	check = (Runtime{}).commandPathCheck()
	if check == nil || check.Status != statusPass {
		t.Fatalf("reachable path check = %+v", check)
	}
	if err := os.Remove(filepath.Join(root, "current")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "current"), 0o755); err != nil {
		t.Fatal(err)
	}
	if (Runtime{}).commandPathCheck() != nil {
		t.Fatal("development current should not add a PATH check")
	}
}

func TestWorkflowCheckNamesFornaxReleaseSet(t *testing.T) {
	dataRoot := t.TempDir()
	source := t.TempDir()
	if err := os.CopyFS(filepath.Join(source, "fanloop-workflow", "linear-runtime"), os.DirFS(filepath.Join("..", "workflow", "testdata", "linear-runtime"))); err != nil {
		t.Fatal(err)
	}
	releases := []configuration.ReleaseRef{{Key: "fanloop.workflow.linear_runtime", Version: "1.0.0", WorkflowID: "linear-runtime"}}
	snapshot, err := configuration.PublishRelease(dataRoot, source, releases)
	if err != nil {
		t.Fatal(err)
	}
	check := Runtime{WorkflowRoot: filepath.Join(snapshot, "fanloop-workflow")}.workflowCheck()
	if check.Status != statusPass || !strings.Contains(check.Summary, "release_set_digest=8ad2b03e5cbf97096c3fa421c6ea9cd5a273e87cef72c330fff555a22119de99") || strings.Contains(check.Summary, "commit_sha=") {
		t.Fatalf("summary = %q", check.Summary)
	}
}
