package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstalledReleaseUsesConditionRoutingAcrossFlowTraceCardAndDoctor(t *testing.T) {
	repository := repositoryRoot(t)
	releaseFixture := makeReleaseFixture(t, repository, "1.2.3", "1.2.3")
	dataRoot, codexRoot, agentsRoot := t.TempDir(), t.TempDir(), t.TempDir()
	if result := runInstaller(t, repository, releaseFixture, dataRoot, codexRoot, agentsRoot); result.err != nil {
		t.Fatalf("install release: %v\n%s", result.err, result.stderr)
	}

	bin := t.TempDir()
	logPath, traceContent := filepath.Join(t.TempDir(), "lark.log"), filepath.Join(t.TempDir(), "trace.md")
	writeE2ELarkCLI(t, filepath.Join(bin, "lark-cli"))
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_LARK_LOG", logPath)
	t.Setenv("FAKE_TRACE_CONTENT", traceContent)

	root := t.TempDir()
	run := func(args ...string) cliResult {
		t.Helper()
		result := runCurrent(dataRoot, codexRoot, agentsRoot, "", args...)
		if result.err != nil || result.stderr != "" {
			t.Fatalf("fanloop %s: %v\nstdout: %s\nstderr: %s", strings.Join(args, " "), result.err, result.stdout, result.stderr)
		}
		return result
	}
	run("flow", "init", "--root", root, "--workflow", "promotion-design", "--title", "Release E2E")
	run("trace", "bind", "--root", root, "--document-url", "https://bytedance.larkoffice.com/docx/TraceE2E")
	first := run("flow", "report", "result", "--root", root, "--input", `{"step_id":"clarify_requirements","condition_results":[{"condition_id":"requirements_ready","output":{"type":"path","value":"require_points.md"}}],"route":{"next_step_id":"confirm_requirements"},"summary":"requirements ready","evidence":[]}`)
	if !strings.Contains(first.stdout, `"effect": "advanced"`) {
		t.Fatalf("matching Condition result did not advance: %s", first.stdout)
	}
	incomplete := runCurrent(dataRoot, codexRoot, agentsRoot, "", "flow", "report", "result", "--root", root, "--input", `{"step_id":"confirm_requirements","condition_results":[{"condition_id":"requirements_rejected","output":{"type":"enum_value","value":"rejected"}}],"route":{"next_step_id":"write_promotion_design"},"summary":"rejected requirements cannot advance","evidence":[]}`)
	if incomplete.err == nil || !strings.Contains(incomplete.stderr, `"code": "ROUTE_NOT_MATCHED"`) {
		t.Fatalf("document-only result bypassed requirement review:\nstdout: %s\nstderr: %s", incomplete.stdout, incomplete.stderr)
	}
	waitingForReview := run("flow", "status", "--root", root)
	if !strings.Contains(waitingForReview.stdout, `"step_id": "confirm_requirements"`) {
		t.Fatalf("rejected document-only result changed current Step: %s", waitingForReview.stdout)
	}
	approved := run("flow", "report", "result", "--root", root, "--input", `{"step_id":"confirm_requirements","condition_results":[{"condition_id":"requirements_approved","output":{"type":"enum_value","value":"approved"}}],"route":{"next_step_id":"write_promotion_design"},"summary":"requirements approved","evidence":[{"source":"human","content":"approved requirements","ref":"requirement-e2e"}]}`)
	if !strings.Contains(approved.stdout, `"step_id": "write_promotion_design"`) {
		t.Fatalf("approved requirements did not enter solution design: %s", approved.stdout)
	}
	looped := run("flow", "report", "result", "--root", root, "--input", `{"step_id":"write_promotion_design","condition_results":[{"condition_id":"requirements_changed","output":{"type":"enum_value","value":"changed"}}],"route":{"back_step_id":"clarify_requirements"},"summary":"requirements changed","evidence":[]}`)
	if !strings.Contains(looped.stdout, `"effect": "looped"`) {
		t.Fatalf("matching Loop Condition did not return: %s", looped.stdout)
	}
	status := run("flow", "status", "--root", root)
	if !strings.Contains(status.stdout, `"status": "running"`) || !strings.Contains(status.stdout, `"step_id": "clarify_requirements"`) || !strings.Contains(status.stdout, `"status": "ready"`) {
		t.Fatalf("requirement did not preserve its returned Step: %s", status.stdout)
	}

	run("trace", "render", "--root", root)
	synced := run("trace", "sync", "--root", root)
	if strings.Count(synced.stdout, `"status": "succeeded"`) != 2 {
		t.Fatalf("trace did not sync both targets: %s", synced.stdout)
	}
	if content, err := os.ReadFile(traceContent); err != nil ||
		!strings.Contains(string(content), "# PRD Flow Trace") ||
		!strings.Contains(string(content), "| 时间 | 事件 | Skill | 状态变化 | 结果 | 用户对话 | 判断依据 | 证据 |") ||
		!strings.Contains(string(content), "condition=requirements_changed:changed") ||
		!strings.Contains(string(content), "looped") ||
		strings.Contains(string(content), "loop.feedback") {
		t.Fatalf("trace projection does not keep the Driver audit layout: %v\n%s", err, content)
	}
	run("card", "render", "--root", root, "--view", "panorama", "--format", "lark-json")
	diagnosed := run("doctor", "--root", root)
	if !strings.Contains(diagnosed.stdout, `"status": "healthy"`) || !strings.Contains(diagnosed.stdout, `"id": "workflows"`) {
		t.Fatalf("installed requirement failed Doctor: %s", diagnosed.stdout)
	}
	version := run("version")
	for _, want := range []string{`"release_version": "1.2.3"`, `"name": "fanloop-workflow"`} {
		if !strings.Contains(version.stdout, want) {
			t.Fatalf("matched release omitted %s: %s", want, version.stdout)
		}
	}
	var response struct {
		Data struct {
			Workflows []map[string]any `json:"workflows"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(version.stdout), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Data.Workflows) != 2 {
		t.Fatalf("Workflow releases = %#v, want two", response.Data.Workflows)
	}
	for _, item := range response.Data.Workflows {
		id, idOK := item["id"].(string)
		digest, ok := item["digest"].(string)
		if len(item) != 2 || !idOK || id == "" || !ok || !strings.HasPrefix(digest, "sha256:") {
			t.Fatalf("Workflow release = %#v, want only id and digest", item)
		}
	}
}

func writeE2ELarkCLI(t *testing.T, path string) {
	t.Helper()
	content := `#!/bin/sh
printf '%s\n' "$*" >> "$FAKE_LARK_LOG"
case "$1 $2" in
  "docs +update")
    cat > "$FAKE_TRACE_CONTENT"
    printf '%s\n' '{"ok":true}'
    ;;
  "whoami --as")
    printf '%s\n' '{"identity":"user","available":true,"tokenStatus":"ready","onBehalfOf":{"openId":"ou_e2e"}}'
    ;;
  "base +record-list")
    printf '%s\n' '{"ok":true,"data":{"items":[]}}'
    ;;
  "base +record-upsert")
    printf '%s\n' '{"ok":true,"data":{"record":{"record_id":"rec_e2e"}}}'
    ;;
  "base +record-get")
    printf '%s\n' '{"ok":true,"data":{"record":{"record_id":"rec_e2e","fields":{"trace_key":"docx:TraceE2E"}}}}'
    ;;
  *)
    printf '%s\n' "unexpected command: $*" >&2
    exit 2
    ;;
esac
`
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}
