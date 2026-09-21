package dashboard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestHandlerListsReadsAndProtectsLiveFiles(t *testing.T) {
	root := copyWorkflowTree(t)
	handler, err := NewHandler(root)
	if err != nil {
		t.Fatal(err)
	}

	list := httptest.NewRecorder()
	handler.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/files", nil))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), `"path":"fanloop-maintainer/prompt.yaml"`) || !strings.Contains(list.Body.String(), `"path":"fanloop-maintainer/skills/fanloop-dev-tdd/SKILL.md"`) {
		t.Fatalf("list status=%d body=%s", list.Code, list.Body.String())
	}

	path := "fanloop-maintainer/prompt.yaml"
	read := httptest.NewRecorder()
	handler.ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/api/file?path="+path, nil))
	if read.Code != http.StatusOK {
		t.Fatalf("read status=%d body=%s", read.Code, read.Body.String())
	}
	var current fileResponse
	if err := json.Unmarshal(read.Body.Bytes(), &current); err != nil || current.SHA256 == "" {
		t.Fatalf("read response=%#v err=%v", current, err)
	}

	outside := filepath.Join(filepath.Dir(root), "outside")
	if err := os.WriteFile(outside, []byte("preserve\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []string{"../outside", "/tmp/outside"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, saveRequest(t, invalid, "overwrite\n", current.SHA256))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("save %q status=%d body=%s", invalid, response.Code, response.Body.String())
		}
	}

	external := strings.Replace(current.Content, "最新 main 基线", "external main 基线", 1)
	if external == current.Content {
		t.Fatal("prompt fixture replacement did not match")
	}
	target := filepath.Join(root, filepath.FromSlash(path))
	if err := os.WriteFile(target, []byte(external), 0o600); err != nil {
		t.Fatal(err)
	}
	stale := httptest.NewRecorder()
	handler.ServeHTTP(stale, saveRequest(t, path, current.Content, current.SHA256))
	if stale.Code != http.StatusConflict || !strings.Contains(stale.Body.String(), `"code":"stale_file"`) {
		t.Fatalf("stale status=%d body=%s", stale.Code, stale.Body.String())
	}
	if content, err := os.ReadFile(target); err != nil || string(content) != external {
		t.Fatalf("stale save changed file: %v %q", err, content)
	}

	refreshed := readFileResponse(t, handler, path)
	invalidYAML := httptest.NewRecorder()
	handler.ServeHTTP(invalidYAML, saveRequest(t, path, "invalid: true\n", refreshed.SHA256))
	if invalidYAML.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid YAML status=%d body=%s", invalidYAML.Code, invalidYAML.Body.String())
	}
	valid := strings.Replace(refreshed.Content, "external main 基线", "dashboard main 基线", 1)
	validYAML := httptest.NewRecorder()
	handler.ServeHTTP(validYAML, saveRequest(t, path, valid, refreshed.SHA256))
	if validYAML.Code != http.StatusOK {
		t.Fatalf("valid YAML status=%d body=%s", validYAML.Code, validYAML.Body.String())
	}
	if content, err := os.ReadFile(target); err != nil || string(content) != valid {
		t.Fatalf("valid YAML bytes=%q err=%v", content, err)
	}

	skillPath := "fanloop-maintainer/skills/fanloop-dev-tdd/SKILL.md"
	skill := readFileResponse(t, handler, skillPath)
	emptySkill := httptest.NewRecorder()
	handler.ServeHTTP(emptySkill, saveRequest(t, skillPath, " \n", skill.SHA256))
	if emptySkill.Code != http.StatusUnprocessableEntity {
		t.Fatalf("empty Skill status=%d body=%s", emptySkill.Code, emptySkill.Body.String())
	}

	skillTarget := filepath.Join(root, filepath.FromSlash(skillPath))
	if err := os.Remove(skillTarget); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, skillTarget); err != nil {
		t.Fatal(err)
	}
	symlink := httptest.NewRecorder()
	handler.ServeHTTP(symlink, saveRequest(t, skillPath, "overwrite\n", skill.SHA256))
	if symlink.Code != http.StatusBadRequest {
		t.Fatalf("symlink save status=%d body=%s", symlink.Code, symlink.Body.String())
	}
	if content, err := os.ReadFile(outside); err != nil || string(content) != "preserve\n" {
		t.Fatalf("symlink save changed outside file: %v %q", err, content)
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

func TestEmbeddedEditorNeverOverwritesDirtyTextDuringPolling(t *testing.T) {
	content, err := assets.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	script := string(content)
	for _, required := range []string{
		"const generation = ++loadGeneration",
		"if (generation !== loadGeneration) return",
		"if (!force && selected === path && dirty)",
		"if (next.sha256 !== observed) stale = true",
		"const path = selected",
		"if (selected === path)",
		"if (!confirmDiscard()) return",
		"editor.addEventListener('input', () => { ++loadGeneration; updateState(); })",
		"updateState();\n    return;",
	} {
		if !strings.Contains(script, required) {
			t.Fatalf("editor polling contract missing %q", required)
		}
	}
}

func TestHandlerSerializesConcurrentSavesWithTheSameObservedDigest(t *testing.T) {
	root := copyWorkflowTree(t)
	value, err := NewHandler(root)
	if err != nil {
		t.Fatal(err)
	}
	handler := value.(*handler)
	path := "fanloop-maintainer/prompt.yaml"
	current := readFileResponse(t, handler, path)

	const writers = 16
	start := make(chan struct{})
	responses := make(chan int, writers)
	candidates := make(map[string]bool, writers)
	for index := 0; index < writers; index++ {
		content := strings.Replace(current.Content, "最新 main 基线", fmt.Sprintf("并发保存 %d", index), 1)
		if content == current.Content {
			t.Fatal("prompt fixture replacement did not match")
		}
		candidates[content] = true
		go func(content string) {
			<-start
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, saveRequest(t, path, content, current.SHA256))
			responses <- response.Code
		}(content)
	}
	close(start)

	succeeded, conflicted := 0, 0
	for index := 0; index < writers; index++ {
		switch code := <-responses; code {
		case http.StatusOK:
			succeeded++
		case http.StatusConflict:
			conflicted++
		default:
			t.Fatalf("concurrent save status = %d", code)
		}
	}
	if succeeded != 1 || conflicted != writers-1 {
		t.Fatalf("concurrent saves succeeded=%d conflicted=%d", succeeded, conflicted)
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil || !candidates[string(content)] {
		t.Fatalf("final content is not one complete candidate: %v\n%s", err, content)
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
	body, err := json.Marshal(saveRequestBody{Path: path, Content: content, ObservedSHA256: observed})
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
	source := filepath.Join(filepath.Dir(file), "..", "..", "workflows")
	target := filepath.Join(t.TempDir(), "workflows")
	if err := os.CopyFS(target, os.DirFS(source)); err != nil {
		t.Fatal(err)
	}
	return target
}
