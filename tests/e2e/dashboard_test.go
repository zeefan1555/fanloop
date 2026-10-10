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

func TestDashboardServesReadOnlyInstalledWorkflows(t *testing.T) {
	repository := repositoryRoot(t)
	fixture := makeReleaseFixture(t, repository, "1.2.3", "1.2.3")
	fixture.ConfigSource = makeSyntheticConfigSource(t, repository, "sample-custom")
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
	if list := string(get("/api/files")); !strings.Contains(list, `"path":"sample-custom/prompt.yaml"`) || !strings.Contains(list, `"path":"sample-custom/skills/example-analyze-materials/SKILL.md"`) || !strings.Contains(list, `"editable":false`) {
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
	if err := json.Unmarshal(get("/api/topology?workflow=sample-custom"), &topology); err != nil {
		t.Fatal(err)
	}
	if len(topology.Workflow.Stages) != 3 || len(topology.Fragments["analyze_materials"]["step"]) != 1 {
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

	workflowRelative := "sample-custom/workflow.yaml"
	var workflowFile struct {
		Content string `json:"content"`
		SHA256  string `json:"sha256"`
	}
	if err := json.Unmarshal(get("/api/file?path="+url.QueryEscape(workflowRelative)), &workflowFile); err != nil {
		t.Fatal(err)
	}

	statusCode, responseBody := postFile(workflowRelative, "invalid: true\n", workflowFile.SHA256)
	if statusCode != http.StatusMethodNotAllowed {
		t.Fatalf("write=%d %s", statusCode, responseBody)
	}
	content, err := os.ReadFile(filepath.Join(fixture.ConfigSource, "skills", "fanloop-workflow", workflowRelative))
	if err != nil || string(content) != workflowFile.Content {
		t.Fatal("rejected write changed configuration")
	}
}
