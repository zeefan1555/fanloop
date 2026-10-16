package distribution

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func linkRelease(t *testing.T, root, version string) {
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

func TestAutoUpdateSkipsDownloadWhenAlreadyLatest(t *testing.T) {
	root := t.TempDir()
	linkRelease(t, root, "1.2.3")
	assets := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/latest" {
			assets++
			http.NotFound(w, request)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v1.2.3"})
	}))
	defer server.Close()
	options := Options{DataRoot: root, Client: server.Client(), LatestURL: server.URL + "/latest"}
	result, err := AutoUpdate(context.Background(), options)
	if err != nil || !result.Unchanged || result.Version != "1.2.3" || assets != 0 {
		t.Fatalf("result=%+v err=%v assets=%d", result, err, assets)
	}
	if err := os.Remove(filepath.Join(root, "current")); err != nil {
		t.Fatal(err)
	}
	linkRelease(t, root, "1.2.4")
	result, err = AutoUpdate(context.Background(), options)
	if err != nil || !result.Unchanged || result.Version != "1.2.4" || assets != 0 {
		t.Fatalf("newer local result=%+v err=%v assets=%d", result, err, assets)
	}
	if err := RefreshCLI(context.Background(), options, func(string, []string, []string) error {
		t.Fatal("already-latest refresh re-executed")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRefreshCLIFallsBackWithoutSwitching(t *testing.T) {
	root := t.TempDir()
	linkRelease(t, root, "1.2.3")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusInternalServerError)
	}))
	defer server.Close()
	err := RefreshCLI(context.Background(), Options{DataRoot: root, Client: server.Client(), LatestURL: server.URL + "/latest"}, func(string, []string, []string) error {
		t.Fatal("failed refresh re-executed")
		return nil
	})
	if !errors.Is(err, ErrAutomaticUpdate) {
		t.Fatalf("err = %v", err)
	}
	target, readErr := os.Readlink(filepath.Join(root, "current"))
	if readErr != nil || filepath.Clean(target) != filepath.Join("releases", "1.2.3") {
		t.Fatalf("current -> %s (%v)", target, readErr)
	}
}

func TestAutoUpdateStopsAtShortDeadline(t *testing.T) {
	root := t.TempDir()
	linkRelease(t, root, "1.2.3")
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		close(started)
		<-request.Context().Done()
		http.Error(w, "slow", http.StatusGatewayTimeout)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	begun := time.Now()
	_, err := AutoUpdate(ctx, Options{DataRoot: root, Client: server.Client(), LatestURL: server.URL + "/latest"})
	if err == nil || time.Since(begun) > 2*time.Second {
		t.Fatalf("err=%v elapsed=%s", err, time.Since(begun))
	}
}

func TestRefreshCLIReexecsNewerRelease(t *testing.T) {
	version := "1.2.3"
	archive := validArchive(t, version, "#!/bin/sh\necho newer\n", "# workflow\n", "# author\n")
	root := t.TempDir()
	linkRelease(t, root, "1.2.2")
	server := serveRelease(t, version, archive, nil)
	defer server.Close()
	var argv0 string
	var env []string
	err := RefreshCLI(context.Background(), Options{
		DataRoot: root, Client: server.Client(), LatestURL: server.URL + "/latest",
		Platform: Platform{OS: "linux", Arch: "amd64"}, SkillRoots: testSkillRoots(t, root),
	}, func(binary string, argv []string, environment []string) error {
		argv0 = binary
		env = environment
		if len(argv) == 0 || argv[0] != binary {
			t.Fatalf("argv = %#v", argv)
		}
		return nil
	})
	if !errors.Is(err, ErrProcessReplaced) {
		t.Fatalf("err = %v", err)
	}
	if argv0 != filepath.Join(root, "current", "bin", "fanloop") {
		t.Fatalf("binary = %s", argv0)
	}
	if !strings.Contains(strings.Join(env, "\n"), "FANLOOP_AUTO_UPDATE_REEXEC=1") {
		t.Fatal("re-exec environment missing one-shot guard")
	}
	target, readErr := os.Readlink(filepath.Join(root, "current"))
	if readErr != nil || filepath.Clean(target) != filepath.Join("releases", version) {
		t.Fatalf("current -> %s (%v)", target, readErr)
	}
	body, readErr := os.ReadFile(argv0)
	if readErr != nil || !strings.Contains(string(body), "echo newer") {
		t.Fatalf("new binary = %q %v", body, readErr)
	}
}

func TestRefreshCLIOptOutAndReexecGuardSkipNetwork(t *testing.T) {
	root := t.TempDir()
	linkRelease(t, root, "1.2.3")
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits++ }))
	defer server.Close()
	options := Options{DataRoot: root, Client: server.Client(), LatestURL: server.URL + "/latest"}
	t.Setenv(EnvNoAutoUpdate, "1")
	if err := RefreshCLI(context.Background(), options, nil); err != nil || hits != 0 {
		t.Fatalf("opt-out err=%v hits=%d", err, hits)
	}
	t.Setenv(EnvNoAutoUpdate, "")
	if err := os.Unsetenv(EnvNoAutoUpdate); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FANLOOP_AUTO_UPDATE_REEXEC", "1")
	if err := RefreshCLI(context.Background(), options, nil); err != nil || hits != 0 {
		t.Fatalf("guard err=%v hits=%d", err, hits)
	}
}

func TestRefreshCLIIgnoresDevelopmentCurrent(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "current"), 0o755); err != nil {
		t.Fatal(err)
	}
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits++ }))
	defer server.Close()
	err := RefreshCLI(context.Background(), Options{DataRoot: root, Client: server.Client(), LatestURL: server.URL + "/latest"}, nil)
	if err != nil || hits != 0 {
		t.Fatalf("err=%v hits=%d", err, hits)
	}
}
