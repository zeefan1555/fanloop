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
	root := copyRuntimeControls(t)
	handler, err := NewHandler(root, true)
	if err != nil {
		t.Fatal(err)
	}

	list := httptest.NewRecorder()
	handler.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/files", nil))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), `"path":"runtime-controls/prompt.yaml"`) || !strings.Contains(list.Body.String(), `"path":"runtime-controls/skills/common-control-guide/SKILL.md"`) {
		t.Fatalf("list status=%d body=%s", list.Code, list.Body.String())
	}

	path := "runtime-controls/prompt.yaml"
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

	external := strings.Replace(current.Content, "Advance after the current Step completes.", "Advance after an external edit.", 1)
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
	valid := strings.Replace(refreshed.Content, "Advance after an external edit.", "Advance after a dashboard edit.", 1)
	validYAML := httptest.NewRecorder()
	handler.ServeHTTP(validYAML, saveRequest(t, path, valid, refreshed.SHA256))
	if validYAML.Code != http.StatusOK {
		t.Fatalf("valid YAML status=%d body=%s", validYAML.Code, validYAML.Body.String())
	}
	if content, err := os.ReadFile(target); err != nil || string(content) != valid {
		t.Fatalf("valid YAML bytes=%q err=%v", content, err)
	}

	skillPath := "runtime-controls/skills/common-control-guide/SKILL.md"
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

func TestPublishedHandlerRejectsWrites(t *testing.T) {
	root := copyRuntimeControls(t)
	handler, err := NewHandler(root, false)
	if err != nil {
		t.Fatal(err)
	}
	path := "runtime-controls/prompt.yaml"
	before := readFileResponse(t, handler, path)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, saveRequest(t, path, before.Content+"\n# rejected\n", before.SHA256))
	if response.Code != http.StatusMethodNotAllowed || !strings.Contains(response.Body.String(), `"code":"read_only"`) {
		t.Fatalf("published save status=%d body=%s", response.Code, response.Body.String())
	}
	if after := readFileResponse(t, handler, path); after != before {
		t.Fatal("rejected published save changed file")
	}
}

func TestHandlerRejectsSymlinkedCurrentAncestor(t *testing.T) {
	realRoot := copyRuntimeControls(t)
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

	if _, err := NewHandler(filepath.Join(dataRoot, "current", filepath.Base(realRoot)), false); err == nil || !strings.Contains(err.Error(), "real directory") {
		t.Fatalf("NewHandler accepted symlinked current ancestor: %v", err)
	}
}

func TestHandlerRejectsCandidateRootSwappedToSymlinkAfterStartup(t *testing.T) {
	root := copyRuntimeControls(t)
	handler, err := NewHandler(root, true)
	if err != nil {
		t.Fatal(err)
	}
	path := "runtime-controls/prompt.yaml"
	current := readFileResponse(t, handler, path)
	externalRoot := copyRuntimeControls(t)
	externalPath := filepath.Join(externalRoot, filepath.FromSlash(path))
	externalBefore, err := os.ReadFile(externalPath)
	if err != nil {
		t.Fatal(err)
	}
	backup := root + "-original"
	if err := os.Rename(root, backup); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(externalRoot, root); err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, saveRequest(t, path, current.Content+"\n# must not write\n", current.SHA256))
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"invalid_path"`) {
		t.Fatalf("root swap save status=%d body=%s", response.Code, response.Body.String())
	}
	if externalAfter, err := os.ReadFile(externalPath); err != nil || !bytes.Equal(externalAfter, externalBefore) {
		t.Fatalf("root swap changed external bytes: %v\n%s", err, externalAfter)
	}
}

func TestLiveHandlerAcceptsAtomicallyReplacedWorkflowRoot(t *testing.T) {
	root := copyRuntimeControls(t)
	handler, err := NewHandler(root, false)
	if err != nil {
		t.Fatal(err)
	}
	replacement := copyRuntimeControls(t)
	backup := root + "-previous"
	if err := os.Rename(root, backup); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, root); err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/files", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"path":"runtime-controls/prompt.yaml"`) {
		t.Fatalf("live root replacement status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestAtomicWriteAtStaysOnOpenedCandidateRoot(t *testing.T) {
	root := copyRuntimeControls(t)
	value, err := NewHandler(root, true)
	if err != nil {
		t.Fatal(err)
	}
	handler := value.(*handler)
	opened, err := handler.openRoot()
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	path := "runtime-controls/prompt.yaml"
	originalPath := filepath.Join(root, filepath.FromSlash(path))
	original, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatal(err)
	}
	externalRoot := copyRuntimeControls(t)
	externalPath := filepath.Join(externalRoot, filepath.FromSlash(path))
	externalBefore, err := os.ReadFile(externalPath)
	if err != nil {
		t.Fatal(err)
	}
	backup := root + "-original"
	if err := os.Rename(root, backup); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(externalRoot, root); err != nil {
		t.Fatal(err)
	}
	changed := append(append([]byte(nil), original...), []byte("\n# anchored write\n")...)
	if err := atomicWriteAt(opened, path, changed); err != nil {
		t.Fatal(err)
	}
	if externalAfter, err := os.ReadFile(externalPath); err != nil || !bytes.Equal(externalAfter, externalBefore) {
		t.Fatalf("anchored write changed external bytes: %v\n%s", err, externalAfter)
	}
	if got, err := os.ReadFile(filepath.Join(backup, filepath.FromSlash(path))); err != nil || !bytes.Equal(got, changed) {
		t.Fatalf("anchored write missed original candidate: %v\n%s", err, got)
	}
}

func TestHandlerCanReadAndRepairAFileAfterExternalBundleCorruption(t *testing.T) {
	root := copyRuntimeControls(t)
	handler, err := NewHandler(root, true)
	if err != nil {
		t.Fatal(err)
	}
	path := "runtime-controls/workflow.yaml"
	original := readFileResponse(t, handler, path)
	target := filepath.Join(root, filepath.FromSlash(path))
	invalid := "invalid: true\n"
	if err := os.WriteFile(target, []byte(invalid), 0o600); err != nil {
		t.Fatal(err)
	}

	current := readFileResponse(t, handler, path)
	if current.Content != invalid || current.SHA256 == original.SHA256 {
		t.Fatalf("external invalid bytes were not readable: %#v", current)
	}
	stale := httptest.NewRecorder()
	handler.ServeHTTP(stale, saveRequest(t, path, original.Content, original.SHA256))
	if stale.Code != http.StatusConflict || !strings.Contains(stale.Body.String(), `"code":"stale_file"`) {
		t.Fatalf("stale repair status=%d body=%s", stale.Code, stale.Body.String())
	}
	repair := httptest.NewRecorder()
	handler.ServeHTTP(repair, saveRequest(t, path, original.Content, current.SHA256))
	if repair.Code != http.StatusOK {
		t.Fatalf("repair status=%d body=%s", repair.Code, repair.Body.String())
	}
}

func TestHandlerRejectsSkillSaveWhenAnotherWorkflowIsInvalid(t *testing.T) {
	root := copyRuntimeControls(t)
	copySyntheticWorkflow(t, root, "other-controls")
	handler, err := NewHandler(root, true)
	if err != nil {
		t.Fatal(err)
	}
	skillPath := "runtime-controls/skills/common-control-guide/SKILL.md"
	skill := readFileResponse(t, handler, skillPath)
	brokenWorkflow := filepath.Join(root, "other-controls", "workflow.yaml")
	if err := os.WriteFile(brokenWorkflow, []byte("invalid: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, saveRequest(t, skillPath, skill.Content+"\nDashboard edit.\n", skill.SHA256))
	if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), `"code":"invalid_content"`) {
		t.Fatalf("skill save with invalid root status=%d body=%s", response.Code, response.Body.String())
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(skillPath)))
	if err != nil || string(content) != skill.Content {
		t.Fatalf("rejected skill save changed file: %v %q", err, content)
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
		"if (next.sha256 !== observed) {",
		"const path = selected",
		"if (selected === path)",
		"if (!confirmDiscard()) return",
		"stateMessage = message",
		"updateState(error.message, true)",
		"editor.addEventListener('input', () => { ++loadGeneration; updateState(''); })",
	} {
		if !strings.Contains(script, required) {
			t.Fatalf("editor polling contract missing %q", required)
		}
	}
}

func TestHandlerSerializesConcurrentSavesWithTheSameObservedDigest(t *testing.T) {
	root := copyRuntimeControls(t)
	value, err := NewHandler(root, true)
	if err != nil {
		t.Fatal(err)
	}
	handler := value.(*handler)
	path := "runtime-controls/prompt.yaml"
	current := readFileResponse(t, handler, path)

	const writers = 16
	start := make(chan struct{})
	responses := make(chan int, writers)
	candidates := make(map[string]bool, writers)
	for index := 0; index < writers; index++ {
		content := strings.Replace(current.Content, "Advance after the current Step completes.", fmt.Sprintf("Concurrent save %d.", index), 1)
		if content == current.Content {
			t.Fatal("prompt fixture replacement did not match")
		}
		candidates[content] = true
		// Independent handlers have no shared in-memory mutex or data home.
		writer, err := NewHandler(root, true)
		if err != nil {
			t.Fatal(err)
		}
		go func(content string) {
			<-start
			response := httptest.NewRecorder()
			writer.ServeHTTP(response, saveRequest(t, path, content, current.SHA256))
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

func copySyntheticWorkflow(t *testing.T, root, id string) {
	t.Helper()
	target := filepath.Join(root, id)
	if err := os.CopyFS(target, os.DirFS(filepath.Join("..", "workflow", "testdata", "runtime-controls"))); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(target, "workflow.yaml")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content = bytes.Replace(content, []byte("id: runtime-controls"), []byte("id: "+id), 1)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
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
