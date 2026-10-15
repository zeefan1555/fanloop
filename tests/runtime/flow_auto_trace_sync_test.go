package runtime_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/traceconfig"
)

func TestDryRunAcceptsPathOutput(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	contract := filepath.Join(root, "requirements.md")
	if err := os.WriteFile(contract, []byte("# Contract\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Path Output"), "flow.init")
	reported := run(binary, "flow", "report", "result",
		"--root", root,
		"--dry-run",
		"--step-id", "first",
		"--condition-result", conditionResult("first_completed", "path", `"requirements.md"`),
		"--next-step-id", "second",
		"--summary", "Path output prepared",
	)
	assertSuccess(t, reported, "flow.report.result")
}

func TestFlowReportAutomaticallySyncsBoundTraceThroughCLI(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	fakeBin := filepath.Dir(binary)
	logPath := filepath.Join(t.TempDir(), "lark.log")
	traceContentPath := filepath.Join(t.TempDir(), "trace.md")
	registryFieldsPath := filepath.Join(t.TempDir(), "registry-fields.json")
	recordExistsPath := filepath.Join(t.TempDir(), "record-exists")
	writeAutoSyncLarkCLI(t, filepath.Join(fakeBin, "lark-cli"))
	t.Setenv("FAKE_LARK_LOG", logPath)
	t.Setenv("FAKE_TRACE_CONTENT", traceContentPath)
	t.Setenv("FAKE_REGISTRY_FIELDS", registryFieldsPath)
	t.Setenv("FAKE_RECORD_EXISTS", recordExistsPath)
	t.Setenv("TRACE_REGISTRY_BASE_TOKEN", "must_not_be_read")
	t.Setenv("TRACE_REGISTRY_TABLE_ID", "must_not_be_read")

	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Auto Sync"), "flow.init")
	assertSuccess(t, run(binary, "trace", "bind", "--root", root, "--document-url", "https://docs.example.invalid/docx/AutoSyncTrace", "--registry", "test"), "trace.bind")
	registry, ok := traceconfig.Resolve(traceconfig.RegistryTest, "linear-runtime")
	if !ok {
		t.Fatal("resolve synthetic Workflow test Registry")
	}
	configBytes := readFile(t, filepath.Join(root, ".fanloop", "trace", "config.json"))
	var config map[string]any
	if err := json.Unmarshal(configBytes, &config); err != nil {
		t.Fatalf("decode Trace config: %v", err)
	}
	for key, want := range map[string]string{
		"registry_profile":    string(registry.Profile),
		"registry_url":        registry.URL,
		"registry_base_token": registry.BaseToken,
		"registry_table_id":   registry.TableID,
		"registry_view_id":    registry.ViewID,
	} {
		if config[key] != want {
			t.Fatalf("Trace config %s = %#v, want %q:\n%s", key, config[key], want, configBytes)
		}
	}
	first := run(binary, "flow", "report", "progress", "--root", root, "--step-id", "first", "--status", "in_progress", "--summary", "started")
	assertSuccess(t, first, "flow.report.progress")
	assertRegistryFields(t, registryFieldsPath, registry.Fields, "In Progress", "Runtime / Lifecycle / First")

	second := run(binary, "flow", "report", "progress", "--root", root, "--step-id", "first", "--status", "blocked", "--summary", "waiting")
	assertSuccess(t, second, "flow.report.progress")
	assertRegistryFields(t, registryFieldsPath, registry.Fields, "Blocked", "Runtime / Lifecycle / First")

	third := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "first",
		"--condition-result", conditionResult("first_completed", "path", `"first.md"`),
		"--next-step-id", "second", "--summary", "first complete")
	assertSuccess(t, third, "flow.report.result")
	assertRegistryFields(t, registryFieldsPath, registry.Fields, "In Progress", "Runtime / Lifecycle / Second")

	log := string(readFile(t, logPath))
	if got := strings.Count(log, "docs +update"); got != 3 {
		t.Fatalf("each accepted Flow update must sync the Trace document: calls=%d\n%s", got, log)
	}
	if got := strings.Count(log, "base +record-upsert"); got != 3 {
		t.Fatalf("each accepted Flow update must sync the Registry: calls=%d\n%s", got, log)
	}
	for _, want := range []string{"--base-token " + registry.BaseToken, "--table-id " + registry.TableID, "--view-id " + registry.ViewID} {
		if !strings.Contains(log, want) {
			t.Fatalf("automatic sync did not use fixed test Registry setting %q:\n%s", want, log)
		}
	}
	if strings.Contains(log, "must_not_be_read") {
		t.Fatalf("automatic sync still reads the removed Registry environment override:\n%s", log)
	}
	if !strings.Contains(log, "whoami --as bot") || strings.Contains(log, "--as user") || strings.Contains(log, "auth status") {
		t.Fatalf("automatic sync must use only the lark-cli bot identity:\n%s", log)
	}
	traceContent := string(readFile(t, traceContentPath))
	for _, want := range []string{"# Workflow Trace", "Runtime/Lifecycle/First → Runtime/Lifecycle/Second", "first_artifact"} {
		if !strings.Contains(traceContent, want) {
			t.Fatalf("auto-synced Trace content does not contain %q:\n%s", want, traceContent)
		}
	}
	status := run(binary, "trace", "status", "--root", root)
	assertSuccess(t, status, "trace.status")
	for _, want := range []string{`"event_count": 11`, `"outcome": "succeeded"`, `"target": "trace_document"`, `"target": "registry"`} {
		if !strings.Contains(status.stdout, want) {
			t.Fatalf("Trace status does not contain %s:\n%s", want, status.stdout)
		}
	}
}

func TestFlowReportKeepsCommittedUpdateWhenAutoSyncIsPartial(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	fakeBin := filepath.Dir(binary)
	logPath := filepath.Join(t.TempDir(), "lark.log")
	writeAutoSyncLarkCLI(t, filepath.Join(fakeBin, "lark-cli"))
	t.Setenv("FAKE_LARK_LOG", logPath)
	t.Setenv("FAKE_TRACE_CONTENT", filepath.Join(t.TempDir(), "trace.md"))
	t.Setenv("FAKE_REGISTRY_FIELDS", filepath.Join(t.TempDir(), "registry-fields.json"))
	t.Setenv("FAKE_RECORD_EXISTS", filepath.Join(t.TempDir(), "record-exists"))
	t.Setenv("FAKE_FAIL_TARGET", "registry")

	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Partial Auto Sync"), "flow.init")
	assertSuccess(t, run(binary, "trace", "bind", "--root", root, "--document-url", "https://docs.example.invalid/docx/AutoSyncTrace", "--registry", "test"), "trace.bind")
	reported := run(binary, "flow", "report", "progress", "--root", root, "--step-id", "first", "--status", "blocked", "--summary", "local fact wins")
	assertSuccess(t, reported, "flow.report.progress")
	if !strings.Contains(reported.stdout, `"effect": "status_updated"`) {
		t.Fatalf("Flow update was not committed: %s", reported.stdout)
	}

	flowStatus := run(binary, "flow", "status", "--root", root)
	assertSuccess(t, flowStatus, "flow.status")
	if !strings.Contains(flowStatus.stdout, `"status": "blocked"`) || !strings.Contains(flowStatus.stdout, "local fact wins") {
		t.Fatalf("partial remote sync rolled back the Flow update: %s", flowStatus.stdout)
	}
	traceStatus := run(binary, "trace", "status", "--root", root)
	assertSuccess(t, traceStatus, "trace.status")
	for _, want := range []string{`"outcome": "partial"`, `"target": "trace_document"`, `"status": "succeeded"`, `"target": "registry"`, `"status": "failed"`} {
		if !strings.Contains(traceStatus.stdout, want) {
			t.Fatalf("partial auto-sync outcome does not contain %s:\n%s", want, traceStatus.stdout)
		}
	}
}

func TestFlowReportDoesNotAutoSyncRejectedOrDryRunUpdates(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	fakeBin := filepath.Dir(binary)
	logPath := filepath.Join(t.TempDir(), "lark.log")
	writeAutoSyncLarkCLI(t, filepath.Join(fakeBin, "lark-cli"))
	t.Setenv("FAKE_LARK_LOG", logPath)
	t.Setenv("FAKE_TRACE_CONTENT", filepath.Join(t.TempDir(), "trace.md"))
	t.Setenv("FAKE_REGISTRY_FIELDS", filepath.Join(t.TempDir(), "registry-fields.json"))
	t.Setenv("FAKE_RECORD_EXISTS", filepath.Join(t.TempDir(), "record-exists"))

	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "No Auto Sync"), "flow.init")
	assertSuccess(t, run(binary, "trace", "bind", "--root", root, "--document-url", "https://docs.example.invalid/docx/AutoSyncTrace"), "trace.bind")
	rejected := run(binary, "flow", "report", "result", "--root", root, "--step-id", "second", "--condition-result", conditionResult("second_completed", "path", `"second.md"`), "--condition-result", conditionResult("second_evidence", "path", `"evidence.md"`), "--next-step-id", "third", "--summary", "stale report")
	if rejected.exitCode == 0 || !strings.Contains(rejected.stderr, `"code": "STEP_NOT_CURRENT"`) {
		t.Fatalf("expected a rejected report:\nstdout=%s\nstderr=%s", rejected.stdout, rejected.stderr)
	}
	dryRun := run(binary, "flow", "report", "progress", "--root", root, "--dry-run", "--step-id", "first", "--status", "in_progress", "--summary", "dry run")
	assertSuccess(t, dryRun, "flow.report.progress")

	if log, err := os.ReadFile(logPath); err == nil && len(log) > 0 {
		t.Fatalf("rejected and dry-run reports must not touch Lark:\n%s", log)
	} else if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	traceStatus := run(binary, "trace", "status", "--root", root)
	assertSuccess(t, traceStatus, "trace.status")
	if !strings.Contains(traceStatus.stdout, `"event_count": 2`) || strings.Contains(traceStatus.stdout, `"last_sync"`) {
		t.Fatalf("non-committed reports changed Trace sync history: %s", traceStatus.stdout)
	}
}

func assertRegistryFields(t *testing.T, path string, mapping traceconfig.RegistryFields, wantStatus, wantStageStep string) {
	t.Helper()
	var fields map[string]any
	if err := json.Unmarshal(readFile(t, path), &fields); err != nil {
		t.Fatalf("decode Registry fields: %v", err)
	}
	if got := fields[mapping.Status]; got != wantStatus {
		t.Fatalf("Registry status = %#v, want %q; fields=%#v", got, wantStatus, fields)
	}
	if got := fields[mapping.TraceKey]; got != "docx:AutoSyncTrace" {
		t.Fatalf("Registry key = %#v", got)
	}
	if _, ok := fields[mapping.Owner]; ok {
		t.Fatalf("bot-backed Registry sync must not invent a human owner: %#v", fields[mapping.Owner])
	}
	stageAndAudit, _ := fields[mapping.Location].(string)
	if stageAndAudit != wantStageStep && !strings.HasPrefix(stageAndAudit, wantStageStep+"\n") {
		t.Fatalf("Registry location = %#v, want prefix %q", stageAndAudit, wantStageStep)
	}
	updated, _ := fields[mapping.UpdatedAt].(string)
	if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`).MatchString(updated) {
		t.Fatalf("Registry updated time must use the Base datetime format, got %q", updated)
	}
}

func writeAutoSyncLarkCLI(t *testing.T, path string) {
	t.Helper()
	content := `#!/bin/sh
set -eu
printf '%s\n' "$*" >> "$FAKE_LARK_LOG"
case "$1 $2" in
  "docs +update")
		doc=""
		for arg in "$@"; do
			if [ "$doc" = next ]; then doc="$arg"; break; fi
			if [ "$arg" = "--doc" ]; then doc=next; fi
		done
		case "$doc" in
		*AutoSyncCLILog*)
			/bin/cat > "$FAKE_CLI_LOG_CONTENT"
			if [ "${FAKE_FAIL_TARGET:-}" = "cli_log_document" ]; then
				printf '%s\n' 'CLI log unavailable' >&2
				exit 8
			fi
			;;
		*)
			/bin/cat > "$FAKE_TRACE_CONTENT"
			;;
		esac
    printf '%s\n' '{"ok":true}'
    ;;
	  "whoami --as")
	    printf '%s\n' '{"identity":"bot","available":true,"tokenStatus":"ready"}'
    ;;
	  "base +record-list")
	    field_id=""
	    previous=""
	    for arg in "$@"; do
	      if [ "$previous" = "--field-id" ]; then field_id="$arg"; break; fi
	      previous="$arg"
	    done
	    if [ -f "$FAKE_RECORD_EXISTS" ]; then
	      printf '{"ok":true,"data":{"data":[["docx:AutoSyncTrace"]],"fields":["%s"],"record_id_list":["rec_runtime"]}}\n' "$field_id"
	    else
	      printf '{"ok":true,"data":{"data":[],"fields":["%s"],"record_id_list":[]}}\n' "$field_id"
    fi
    ;;
  "base +record-upsert")
    if [ "${FAKE_FAIL_TARGET:-}" = "registry" ]; then
      printf '%s\n' 'registry unavailable' >&2
      exit 8
    fi
    existing=false
    record_id=""
    if [ -f "$FAKE_RECORD_EXISTS" ]; then
      existing=true
    fi
    shift 2
    while [ "$#" -gt 0 ]; do
      if [ "$1" = "--json" ]; then
        printf '%s\n' "$2" > "$FAKE_REGISTRY_FIELDS"
        shift 2
        continue
      fi
      if [ "$1" = "--record-id" ]; then
        record_id="$2"
        shift 2
        continue
      fi
      shift
    done
    if [ "$existing" = true ] && [ "$record_id" != "rec_runtime" ]; then
      printf '%s\n' 'existing Trace row must be updated by record ID' >&2
      exit 9
    fi
	    : > "$FAKE_RECORD_EXISTS"
	    if [ "$existing" = true ]; then
	      printf '%s\n' '{"ok":true,"data":{"record":{"update":{}},"updated":true}}'
	    else
	      printf '%s\n' '{"ok":true,"data":{"record":{"create":{}},"created":true}}'
	    fi
	    ;;
	  "base +record-get")
	    field_id=""
	    previous=""
	    for arg in "$@"; do
	      if [ "$previous" = "--field-id" ]; then field_id="$arg"; break; fi
	      previous="$arg"
	    done
	    printf '{"ok":true,"data":{"data":[["docx:AutoSyncTrace"]],"fields":["%s"],"record_id_list":["rec_runtime"]}}\n' "$field_id"
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
