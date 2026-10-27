package flow

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cardruntime "github.com/zeefan1555/fanloop/internal/card"
	"github.com/zeefan1555/fanloop/internal/idl/flowidl"
)

func TestInitCreatesCardProjectionWhenTraceProvisioningWarns(t *testing.T) {
	dataRoot := t.TempDir()
	workflowsRoot := filepath.Join(dataRoot, "source", "fanloop-workflow")
	if err := os.CopyFS(filepath.Join(workflowsRoot, "linear-runtime"), os.DirFS(filepath.Join("..", "workflow", "testdata", "linear-runtime"))); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_CONFIG_SOURCE", filepath.Join(dataRoot, "source"))
	root := t.TempDir()
	warnings := []string{}
	runtime := Runtime{
		Clock:             func() time.Time { return time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC) },
		EventID:           func() string { return "event-init" },
		ControllerRelease: testControllerRelease,
		ReleaseVersion:    "test",
		ProjectCard:       cardruntime.WriteProjection,
		ProvisionTrace:    func(context.Context, string, string) string { return "Trace provisioning warning" },
		Warn:              func(message string) { warnings = append(warnings, message) },
	}
	if _, err := runtime.Init(context.Background(), root, &flowidl.FlowInitRequest{
		Workflow: "linear-runtime", Requirement: &flowidl.Requirement{Title: "warning still projects"},
	}, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cardruntime.ProjectionPath(root)); err != nil {
		t.Fatalf("Trace warning prevented Card projection: %v", err)
	}
	if len(warnings) != 2 || !strings.Contains(warnings[0], "author/live") || !strings.Contains(warnings[1], "Trace provisioning warning") {
		t.Fatalf("warnings = %#v", warnings)
	}
}
