package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/zeefan1555/fanloop/internal/workflow"
)

func TestDashboardInitialReadUsesConfigurationLock(t *testing.T) {
	dataRoot := t.TempDir()
	workflowRoot := filepath.Join(dataRoot, "skills")
	if err := os.CopyFS(filepath.Join(workflowRoot, "runtime-controls"), os.DirFS(filepath.Join("..", "internal", "workflow", "testdata", "runtime-controls"))); err != nil {
		t.Fatal(err)
	}
	unlockExclusive, err := workflow.LockSource(workflowRoot, true)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := newDashboardHandler(workflowRoot, false)
		done <- err
	}()
	select {
	case err := <-done:
		unlockExclusive()
		t.Fatalf("Dashboard initial read bypassed the configuration lock: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	unlockExclusive()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Dashboard initial read did not resume after the configuration lock was released")
	}
}
