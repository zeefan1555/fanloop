package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstalledReleaseUsesSyntheticConditionRoutingAcrossDomains(t *testing.T) {
	repository := repositoryRoot(t)
	fixture := makeReleaseFixture(t, repository, "1.2.3", "1.2.3")
	fixture.ConfigSource = makeLinearRuntimeConfigSource(t, repository, "sample-runtime")
	dataRoot, codexRoot, agentsRoot := t.TempDir(), t.TempDir(), t.TempDir()
	if result := runInstaller(t, fixture, dataRoot, codexRoot, agentsRoot); result.err != nil {
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
		result := runCurrent(dataRoot, codexRoot, agentsRoot, args...)
		if result.err != nil || result.stderr != "" {
			t.Fatalf("fanloop %s: %v\nstdout: %s\nstderr: %s", strings.Join(args, " "), result.err, result.stdout, result.stderr)
		}
		return result
	}
	run("flow", "init", "--root", root, "--workflow", "sample-runtime", "--title", "Release E2E")
	run("trace", "bind", "--root", root, "--document-url", "https://example.invalid/redacted")

	run("flow", "report", "result", "--root", root, "--input", `{"step_id":"first","condition_results":[{"condition_id":"first_completed","output":{"type":"path","value":"first.md"}}],"route":{"next_step_id":"second"},"summary":"first complete","evidence":[]}`)
	incomplete := runCurrent(dataRoot, codexRoot, agentsRoot, "flow", "report", "result", "--root", root, "--input", `{"step_id":"second","condition_results":[{"condition_id":"second_completed","output":{"type":"path","value":"second.md"}}],"route":{"next_step_id":"third"},"summary":"missing evidence","evidence":[]}`)
	if incomplete.err == nil || !strings.Contains(incomplete.stderr, `"code": "ROUTE_NOT_MATCHED"`) {
		t.Fatalf("incomplete Condition set advanced:\nstdout: %s\nstderr: %s", incomplete.stdout, incomplete.stderr)
	}
	advanced := run("flow", "report", "result", "--root", root, "--input", `{"step_id":"second","condition_results":[{"condition_id":"second_completed","output":{"type":"path","value":"second.md"}},{"condition_id":"second_evidence","output":{"type":"path","value":"evidence.md"}}],"route":{"next_step_id":"third"},"summary":"second complete","evidence":[]}`)
	if !strings.Contains(advanced.stdout, `"effect": "advanced"`) || !strings.Contains(advanced.stdout, `"step_id": "third"`) {
		t.Fatalf("matching Conditions did not advance: %s", advanced.stdout)
	}
	looped := run("flow", "report", "result", "--root", root, "--input", `{"step_id":"third","condition_results":[{"condition_id":"third_rework","output":{"type":"enum_value","value":"rework"}}],"route":{"back_step_id":"second"},"summary":"third needs rework","evidence":[]}`)
	if !strings.Contains(looped.stdout, `"effect": "looped"`) || !strings.Contains(looped.stdout, `"step_id": "second"`) {
		t.Fatalf("matching Loop Conditions did not return: %s", looped.stdout)
	}

	run("trace", "render", "--root", root)
	synced := run("trace", "sync", "--root", root)
	if strings.Count(synced.stdout, `"status": "succeeded"`) != 2 {
		t.Fatalf("trace did not sync both targets: %s", synced.stdout)
	}
	if content, err := os.ReadFile(traceContent); err != nil || !strings.Contains(string(content), "# Workflow Trace") || !strings.Contains(string(content), "condition=third_rework:rework") || !strings.Contains(string(content), "looped") {
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
	if strings.Contains(version.stdout, `"workflows"`) || strings.Contains(version.stdout, `"digest"`) {
		t.Fatalf("version still exposes Workflow release identity: %s", version.stdout)
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
    printf '%s\n' '{"identity":"bot","available":true,"tokenStatus":"ready"}'
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
