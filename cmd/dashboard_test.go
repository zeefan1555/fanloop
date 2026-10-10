package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zeefan1555/fanloop/internal/configuration"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

func TestDashboardInitialReadUsesConfigurationLock(t *testing.T) {
	dataRoot := t.TempDir()
	workflowRoot := filepath.Join(dataRoot, "fanloop-workflow")
	if err := os.CopyFS(filepath.Join(workflowRoot, "runtime-controls"), os.DirFS(filepath.Join("..", "internal", "workflow", "testdata", "runtime-controls"))); err != nil {
		t.Fatal(err)
	}
	unlockExclusive, err := workflow.LockSource(workflowRoot, true)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := newDashboardHandler(workflowRoot, true)
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

func dashboardSnapshot(t *testing.T) string {
	t.Helper()
	source := t.TempDir()
	if err := os.CopyFS(filepath.Join(source, "fanloop-workflow", "runtime-controls"), os.DirFS(filepath.Join("..", "internal", "workflow", "testdata", "runtime-controls"))); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-b", "main"}, {"add", "."}, {"-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-m", "fixture"}} {
		c := exec.Command("git", args...)
		c.Dir = source
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("Git fixture: %v %s", err, out)
		}
	}
	snapshot, err := configuration.Import(context.Background(), t.TempDir(), source)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(snapshot, "fanloop-workflow")
}

func TestDashboardSnapshotsRemainReadOnlyWithExplicitSource(t *testing.T) {
	root := dashboardSnapshot(t)
	handler, err := newDashboardHandler(root, true)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/api/files", nil))
	if !strings.Contains(response.Body.String(), `"editable":false`) {
		t.Fatal("snapshot became editable")
	}
}

func TestDashboardSnapshotIdentityAcrossRequestLifetime(t *testing.T) {
	for _, running := range []bool{false, true} {
		for _, damage := range []string{"bytes", "missing", "wrong_commit"} {
			for _, explicit := range []bool{false, true} {
				t.Run(fmt.Sprintf("running=%t/damage=%s/explicit=%t", running, damage, explicit), func(t *testing.T) {
					root := dashboardSnapshot(t)
					var handler http.Handler
					if running {
						var err error
						handler, err = newDashboardHandler(root, explicit)
						if err != nil {
							t.Fatal(err)
						}
					}
					if damage == "missing" {
						if err := os.Remove(filepath.Join(filepath.Dir(root), "configuration.json")); err != nil {
							t.Fatal(err)
						}
					} else if damage == "wrong_commit" {
						source := filepath.Dir(root)
						identity, err := configuration.Read(source)
						if err != nil {
							t.Fatal(err)
						}
						identity.CommitSHA = strings.Repeat("1", 40)
						content, err := json.Marshal(identity)
						if err != nil {
							t.Fatal(err)
						}
						path := filepath.Join(source, "configuration.json")
						if err := os.Chmod(path, 0644); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(path, content, 0444); err != nil {
							t.Fatal(err)
						}
					} else {
						path := filepath.Join(root, "runtime-controls", "SKILL.md")
						if err := os.Chmod(path, 0644); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(path, []byte("# Tampered but structurally valid\n"), 0644); err != nil {
							t.Fatal(err)
						}
					}
					if !running {
						if _, err := newDashboardHandler(root, explicit); err == nil {
							t.Fatal("Dashboard accepted broken snapshot at startup")
						}
						return
					}
					for _, path := range []string{"/api/files", "/api/file?path=runtime-controls/SKILL.md", "/api/topology?workflow=runtime-controls"} {
						response := httptest.NewRecorder()
						handler.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
						if response.Code == http.StatusOK || strings.Contains(response.Body.String(), "Tampered but structurally valid") {
							t.Fatalf("%s served broken snapshot: %d %s", path, response.Code, response.Body.String())
						}
					}
				})
			}
		}
	}
}
