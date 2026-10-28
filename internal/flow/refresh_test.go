package flow

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/distribution"
	"github.com/zeefan1555/fanloop/internal/idl/flowidl"
)

func TestInitSkipsAutoUpdateWhenReleaseIsCurrent(t *testing.T) {
	dataRoot, source := authorSource(t)
	linkFanloopRelease(t, dataRoot, "1.2.3")
	assets := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/latest" {
			assets++
			http.NotFound(w, request)
			return
		}
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.3"}`))
	}))
	defer server.Close()
	restoreRefresh(t, server)
	execProcess = func(string, []string, []string) error {
		t.Fatal("already-latest init re-executed")
		return errors.New("re-exec")
	}
	warnings := []string{}
	runtime := quietRuntime()
	runtime.Warn = func(value string) { warnings = append(warnings, value) }
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_CONFIG_SOURCE", source)
	if _, err := runtime.Init(context.Background(), t.TempDir(), initRequest(), false); err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "author/live") || assets != 0 {
		t.Fatalf("warnings=%v assets=%d", warnings, assets)
	}
}

func TestInitFallsBackWhenAutoUpdateFails(t *testing.T) {
	dataRoot, source := authorSource(t)
	linkFanloopRelease(t, dataRoot, "1.2.3")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusInternalServerError)
	}))
	defer server.Close()
	restoreRefresh(t, server)
	execProcess = func(string, []string, []string) error {
		t.Fatal("failed update re-executed")
		return errors.New("re-exec")
	}
	warnings := []string{}
	runtime := quietRuntime()
	runtime.Warn = func(value string) { warnings = append(warnings, value) }
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_CONFIG_SOURCE", source)
	if _, err := runtime.Init(context.Background(), t.TempDir(), initRequest(), false); err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 2 || warnings[0] != "automatic update failed; checking for an installed fallback" || !strings.Contains(warnings[1], "author/live") {
		t.Fatalf("warnings = %#v", warnings)
	}
	target, err := os.Readlink(filepath.Join(dataRoot, "current"))
	if err != nil || filepath.Clean(target) != filepath.Join("releases", "1.2.3") {
		t.Fatalf("current -> %s (%v)", target, err)
	}
}

func TestInitDoesNotAutoUpdateOnDryRunOptOutOrExistingRequirement(t *testing.T) {
	dataRoot, source := authorSource(t)
	linkFanloopRelease(t, dataRoot, "0.1.0")
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits++ }))
	defer server.Close()
	restoreRefresh(t, server)
	runtime := quietRuntime()
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_CONFIG_SOURCE", source)
	root := t.TempDir()
	if _, err := runtime.Init(context.Background(), root, initRequest(), true); err != nil {
		t.Fatal(err)
	}
	t.Setenv(distribution.EnvNoAutoUpdate, "1")
	if _, err := runtime.Init(context.Background(), root, initRequest(), false); err != nil {
		t.Fatal(err)
	}
	if err := os.Unsetenv(distribution.EnvNoAutoUpdate); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Init(context.Background(), root, initRequest(), false); err == nil || !strings.Contains(err.Error(), "already initialized") {
		t.Fatalf("second init err = %v", err)
	}
	if hits != 0 {
		t.Fatalf("update requests = %d", hits)
	}
}

func TestInitStopsBeforeStateWhenReleaseReexecs(t *testing.T) {
	previous := refreshRelease
	refreshRelease = func(context.Context, distribution.Options, func(string, []string, []string) error) error {
		return distribution.ErrProcessReplaced
	}
	t.Cleanup(func() { refreshRelease = previous })
	exited := -1
	previousExit := exitProcess
	exitProcess = func(code int) { exited = code }
	t.Cleanup(func() { exitProcess = previousExit })
	t.Setenv("FANLOOP_DATA_HOME", t.TempDir())
	root := t.TempDir()
	_, err := quietRuntime().Init(context.Background(), root, initRequest(), false)
	if exited != 0 || err == nil {
		t.Fatalf("exited=%d err=%v", exited, err)
	}
	if _, statErr := os.Stat(filepath.Join(root, ".fanloop")); !os.IsNotExist(statErr) {
		t.Fatal("re-exec wrote requirement state")
	}
}

func restoreRefresh(t *testing.T, server *httptest.Server) {
	t.Helper()
	previousOptions := refreshOptions
	previousExec := execProcess
	refreshOptions = func(dataRoot string) distribution.Options {
		return distribution.Options{DataRoot: dataRoot, Client: server.Client(), LatestURL: server.URL + "/latest"}
	}
	t.Cleanup(func() {
		refreshOptions = previousOptions
		execProcess = previousExec
	})
}

func quietRuntime() Runtime {
	runtime := DefaultRuntime()
	runtime.ControllerRelease = testControllerRelease
	runtime.ProvisionTrace = func(context.Context, string, string) string { return "" }
	runtime.ProjectCard = nil
	runtime.AutoSyncTrace = func(context.Context, string) {}
	return runtime
}

func initRequest() *flowidl.FlowInitRequest {
	return &flowidl.FlowInitRequest{Workflow: "linear-runtime", Requirement: &flowidl.Requirement{Title: "Refresh"}}
}

func authorSource(t *testing.T) (string, string) {
	t.Helper()
	dataRoot := t.TempDir()
	source := filepath.Join(dataRoot, "source")
	if err := os.CopyFS(filepath.Join(source, "fanloop-workflow", "linear-runtime"), os.DirFS(filepath.Join("..", "workflow", "testdata", "linear-runtime"))); err != nil {
		t.Fatal(err)
	}
	return dataRoot, source
}

func linkFanloopRelease(t *testing.T, root, version string) {
	t.Helper()
	dir := filepath.Join(root, "releases", version, "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fanloop"), []byte(version), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("releases", version), filepath.Join(root, "current")); err != nil {
		t.Fatal(err)
	}
}
