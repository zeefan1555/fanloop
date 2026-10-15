package flow

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/zeefan1555/fanloop/internal/idl/flowidl"
	"github.com/zeefan1555/fanloop/internal/store"
)

func TestInitPinsSnapshotAfterCurrentMoves(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "real")
	if err := os.Mkdir(real, 0755); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(base, "linked")
	if err := os.Symlink(real, linked); err != nil {
		t.Fatal(err)
	}
	dataRoot := filepath.Join(linked, "data")
	source := filepath.Join(dataRoot, "source")
	if err := os.CopyFS(filepath.Join(source, "fanloop-workflow", "linear-runtime"), os.DirFS(filepath.Join("..", "workflow", "testdata", "linear-runtime"))); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(source)
	if err != nil {
		t.Fatal(err)
	}
	if resolved == source {
		t.Fatal("fixture did not resolve a symlink")
	}
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_CONFIG_SOURCE", source)
	root := t.TempDir()
	runtime := DefaultRuntime()
	runtime.ProvisionTrace = func(context.Context, string, string) string { return "" }
	runtime.ProjectCard = nil
	runtime.AutoSyncTrace = func(context.Context, string) {}
	_, err = runtime.Init(context.Background(), root, &flowidl.FlowInitRequest{
		Workflow:    "linear-runtime",
		Requirement: &flowidl.Requirement{Title: "Pinned requirement"},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	pin, err := os.ReadFile(filepath.Join(root, ".fanloop", "configuration-snapshot"))
	if err != nil {
		t.Fatal(err)
	}
	if string(pin) != resolved+"\n" {
		t.Fatalf("pin = %q", pin)
	}
	t.Setenv("FANLOOP_CONFIG_SOURCE", t.TempDir())
	local, failure := store.New(root)
	if failure != nil {
		t.Fatal(failure)
	}
	_, loaded, failure := local.LoadBound()
	if failure != nil || loaded.Ref.ID != "linear-runtime" {
		t.Fatalf("loaded = %#v failure = %v", loaded.Ref, failure)
	}
}
