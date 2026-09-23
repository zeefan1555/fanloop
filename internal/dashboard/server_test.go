package dashboard

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestHandlerListsReadsAndRejectsWrites(t *testing.T) {
	root := copyWorkflowTree(t)
	handler, err := NewHandler(root)
	if err != nil {
		t.Fatal(err)
	}
	path := "fanloop-maintainer/prompt.yaml"
	before := readFileResponse(t, handler, path)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, saveRequest(t, path, "changed", before.SHA256))
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("write status=%d", response.Code)
	}
	if after := readFileResponse(t, handler, path); after != before {
		t.Fatal("rejected write changed file")
	}
	for _, path := range []string{"../outside", "/tmp/outside"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/file?path="+path, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("unsafe read status=%d", response.Code)
		}
	}
	target := filepath.Join(root, "fanloop-maintainer/skills/fanloop-dev-tdd/SKILL.md")
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/etc/hosts", target); err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/file?path=fanloop-maintainer/skills/fanloop-dev-tdd/SKILL.md", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("symlink read status=%d", response.Code)
	}
}

func TestEmbeddedDashboardIsReadOnly(t *testing.T) {
	html, err := assets.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	script, err := assets.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "readonly") || strings.Contains(string(html), `id="save"`) || strings.Contains(string(script), "method: 'POST'") {
		t.Fatal("Dashboard exposes editing")
	}
}

func TestHandlerRejectsSymlinkedCurrentAncestor(t *testing.T) {
	realRoot := copyWorkflowTree(t)
	dataRoot := t.TempDir()
	legacy := filepath.Join(dataRoot, "releases", "legacy")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Dir(realRoot), legacy); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(legacy, filepath.Join(dataRoot, "current")); err != nil {
		t.Fatal(err)
	}

	if _, err := NewHandler(filepath.Join(dataRoot, "current", filepath.Base(realRoot))); err == nil || !strings.Contains(err.Error(), "real directory") {
		t.Fatalf("NewHandler accepted symlinked current ancestor: %v", err)
	}
}

func readFileResponse(t *testing.T, handler http.Handler, path string) fileResponse {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/file?path="+path, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("read status=%d body=%s", response.Code, response.Body.String())
	}
	var value fileResponse
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func saveRequest(t *testing.T, path, content, observed string) *http.Request {
	t.Helper()
	body, err := json.Marshal(map[string]string{"path": path, "content": content, "observed_sha256": observed})
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewRequest(http.MethodPost, "/api/file", bytes.NewReader(body))
}

func copyWorkflowTree(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate repository")
	}
	source := filepath.Join(filepath.Dir(file), "..", "..", "tests", "testdata", "workflows")
	target := filepath.Join(t.TempDir(), "workflows")
	if err := os.CopyFS(target, os.DirFS(source)); err != nil {
		t.Fatal(err)
	}
	return target
}
