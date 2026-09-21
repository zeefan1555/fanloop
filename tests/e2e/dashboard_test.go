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

	client := &http.Client{Timeout: 10 * time.Second}
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
	var topology struct {
		Workflow struct {
			Stages []struct {
				Jobs []struct {
					Steps []struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"steps"`
				} `json:"jobs"`
			} `json:"stages"`
		} `json:"workflow"`
		Fragments map[string]map[string][]struct {
			Path      string `json:"path"`
			StartLine int    `json:"start_line"`
			EndLine   int    `json:"end_line"`
		} `json:"fragments"`
	}
	if err := json.Unmarshal(get("/api/topology?workflow=fanloop-maintainer"), &topology); err != nil {
		t.Fatal(err)
	}
	if len(topology.Workflow.Stages) != 4 || len(topology.Fragments["define_verification_contract"]["step"]) != 1 {
		t.Fatalf("live topology is incomplete: %#v", topology)
	}

	postFile := func(path, content, observed string) (int, []byte) {
		t.Helper()
		body, err := json.Marshal(map[string]string{"path": path, "content": content, "observed_sha256": observed})
		if err != nil {
			t.Fatal(err)
		}
		response, err := client.Post(started.Data.URL+"/api/file", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		responseBody, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		return response.StatusCode, responseBody
	}

	workflowRelative := "fanloop-maintainer/workflow.yaml"
	var workflowFile struct {
		Content string `json:"content"`
		SHA256  string `json:"sha256"`
	}
	if err := json.Unmarshal(get("/api/file?path="+url.QueryEscape(workflowRelative)), &workflowFile); err != nil {
		t.Fatal(err)
	}
	updatedWorkflow := strings.Replace(workflowFile.Content, "name: 目标与验收契约", "name: Dashboard 目标与验收契约", 1)
	if updatedWorkflow == workflowFile.Content {
		t.Fatal("workflow fixture replacement did not match")
	}
	statusCode, responseBody := postFile(workflowRelative, updatedWorkflow, workflowFile.SHA256)
	if statusCode != http.StatusOK {
		t.Fatalf("save Workflow = %d\n%s", statusCode, responseBody)
	}
	if refreshed := string(get("/api/topology?workflow=fanloop-maintainer")); !strings.Contains(refreshed, `"name":"Dashboard 目标与验收契约"`) {
		t.Fatalf("topology did not refresh after save:\n%s", refreshed)
	}
	var updatedFile struct {
		Content string `json:"content"`
		SHA256  string `json:"sha256"`
	}
	if err := json.Unmarshal(get("/api/file?path="+url.QueryEscape(workflowRelative)), &updatedFile); err != nil {
		t.Fatal(err)
	}
	statusCode, responseBody = postFile(workflowRelative, "invalid: true\n", updatedFile.SHA256)
	if statusCode != http.StatusUnprocessableEntity || !bytes.Contains(responseBody, []byte(`"code":"invalid_content"`)) {
		t.Fatalf("invalid Workflow save = %d\n%s", statusCode, responseBody)
	}
	workflowTarget := filepath.Join(dataRoot, "current", "workflows", filepath.FromSlash(workflowRelative))
	if content, err := os.ReadFile(workflowTarget); err != nil || string(content) != updatedWorkflow {
		t.Fatalf("invalid save changed Workflow bytes: %v\n%s", err, content)
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
	statusCode, responseBody = postFile(skillRelative, savedContent, skill.SHA256)
	if statusCode != http.StatusOK {
		t.Fatalf("save Skill = %d\n%s", statusCode, responseBody)
	}
	target := filepath.Join(dataRoot, "current", "workflows", filepath.FromSlash(skillRelative))
	if content, err := os.ReadFile(target); err != nil || string(content) != savedContent {
		t.Fatalf("saved Skill bytes = %q, %v", content, err)
	}

	externalContent := savedContent + "external edit\n"
	if err := os.WriteFile(target, []byte(externalContent), 0o644); err != nil {
		t.Fatal(err)
	}
	statusCode, responseBody = postFile(skillRelative, savedContent, skill.SHA256)
	if statusCode != http.StatusConflict || !bytes.Contains(responseBody, []byte(`"code":"stale_file"`)) {
		t.Fatalf("stale save = %d\n%s", statusCode, responseBody)
	}
	if content, err := os.ReadFile(target); err != nil || string(content) != externalContent {
		t.Fatalf("stale save changed external bytes: %v\n%s", err, content)
	}
}
