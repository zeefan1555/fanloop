package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDashboardServesAndEditsInstalledLiveWorkflows(t *testing.T) {
	repository := repositoryRoot(t)
	fixture := makeReleaseFixture(t, repository, "1.2.3", "1.2.3")
	dataRoot, codexRoot, agentsRoot := t.TempDir(), t.TempDir(), t.TempDir()
	if result := runInstaller(t, fixture, dataRoot, codexRoot, agentsRoot); result.err != nil {
		t.Fatalf("install release: %v\n%s", result.err, result.stderr)
	}

	command := exec.Command(filepath.Join(dataRoot, "current", "bin", "fanloop"), "dashboard")
	command.Env = append(withoutBotmuxBinding(os.Environ()),
		"FANLOOP_DATA_HOME="+dataRoot,
		"FANLOOP_CODEX_SKILLS_ROOT="+codexRoot,
		"FANLOOP_AGENT_SKILLS_ROOT="+agentsRoot,
	)
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if command.Process != nil {
			_ = command.Process.Kill()
			_ = command.Wait()
		}
	})

	var started struct {
		OK   bool `json:"ok"`
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(stdout).Decode(&started); err != nil || !started.OK {
		t.Fatalf("decode dashboard start: %v\nstderr: %s", err, stderr.String())
	}
	parsed, err := url.Parse(started.Data.URL)
	if err != nil || parsed.Hostname() != "127.0.0.1" || parsed.Port() == "" {
		t.Fatalf("dashboard URL = %q, error = %v", started.Data.URL, err)
	}

	client := &http.Client{Timeout: 3 * time.Second}
	get := func(path string) []byte {
		t.Helper()
		response, err := client.Get(started.Data.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		content, err := io.ReadAll(response.Body)
		if err != nil || response.StatusCode != http.StatusOK {
			t.Fatalf("GET %s = %d, %v\n%s", path, response.StatusCode, err, content)
		}
		return content
	}
	if page := string(get("/")); !strings.Contains(page, "Fanloop Workflow Dashboard") || !strings.Contains(page, "id=\"editor\"") {
		t.Fatalf("dashboard page is incomplete:\n%s", page)
	}
	if list := string(get("/api/files")); !strings.Contains(list, `"path":"fanloop-maintainer/prompt.yaml"`) || !strings.Contains(list, `"path":"fanloop-maintainer/skills/fanloop-dev-tdd/SKILL.md"`) {
		t.Fatalf("dashboard file list is incomplete:\n%s", list)
	}

	skillRelative := "fanloop-maintainer/skills/fanloop-dev-tdd/SKILL.md"
	var skill struct {
		Content string `json:"content"`
		SHA256  string `json:"sha256"`
	}
	if err := json.Unmarshal(get("/api/file?path="+url.QueryEscape(skillRelative)), &skill); err != nil {
		t.Fatal(err)
	}
	savedContent := skill.Content + "\ndashboard save marker\n"
	body, _ := json.Marshal(map[string]string{"path": skillRelative, "content": savedContent, "observed_sha256": skill.SHA256})
	response, err := client.Post(started.Data.URL+"/api/file", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	responseBody, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("save Skill = %d\n%s", response.StatusCode, responseBody)
	}
	target := filepath.Join(dataRoot, "current", "workflows", filepath.FromSlash(skillRelative))
	if content, err := os.ReadFile(target); err != nil || string(content) != savedContent {
		t.Fatalf("saved Skill bytes = %q, %v", content, err)
	}

	externalContent := savedContent + "external edit\n"
	if err := os.WriteFile(target, []byte(externalContent), 0o644); err != nil {
		t.Fatal(err)
	}
	stale, err := client.Post(started.Data.URL+"/api/file", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	staleBody, _ := io.ReadAll(stale.Body)
	_ = stale.Body.Close()
	if stale.StatusCode != http.StatusConflict || !bytes.Contains(staleBody, []byte(`"code":"stale_file"`)) {
		t.Fatalf("stale save = %d\n%s", stale.StatusCode, staleBody)
	}
	if content, err := os.ReadFile(target); err != nil || string(content) != externalContent {
		t.Fatalf("stale save changed external bytes: %v\n%s", err, content)
	}
}
