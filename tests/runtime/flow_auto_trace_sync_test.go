package runtime_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestMaintainerTraceUsesSelfIterationRegistry(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	fakeBin := filepath.Dir(binary)
	logPath := filepath.Join(t.TempDir(), "lark.log")
	writeAutoSyncLarkCLI(t, filepath.Join(fakeBin, "lark-cli"))
	t.Setenv("FAKE_LARK_LOG", logPath)
	t.Setenv("FAKE_TRACE_CONTENT", filepath.Join(t.TempDir(), "trace.md"))
	t.Setenv("FAKE_CLI_LOG_CONTENT", filepath.Join(t.TempDir(), "cli-log.md"))
	t.Setenv("FAKE_REGISTRY_FIELDS", filepath.Join(t.TempDir(), "registry-fields.json"))
	t.Setenv("FAKE_RECORD_EXISTS", filepath.Join(t.TempDir(), "record-exists"))

	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "fanloop-maintainer", "--title", "Self Iteration"), "flow.init")
	assertSuccess(t, run(binary, "trace", "bind", "--root", root, "--document-url", "https://bytedance.larkoffice.com/docx/AutoSyncTrace", "--cli-log-document-url", "https://bytedance.larkoffice.com/docx/AutoSyncCLILog"), "trace.bind")
	configBytes := readFile(t, filepath.Join(root, ".fanloop", "trace", "config.json"))
	var config map[string]any
	if err := json.Unmarshal(configBytes, &config); err != nil {
		t.Fatalf("decode Trace config: %v", err)
	}
	status := run(binary, "trace", "status", "--root", root)
	assertSuccess(t, status, "trace.status")
	for key, want := range map[string]string{
		"registry_url":        "https://bytedance.larkoffice.com/wiki/MTuwwC3DHiPSmNkX4GIcKDv0n7b?table=tblW1KFyrKtUeNF4&view=vew5zFcUtJ",
		"registry_base_token": "Lu15bIcOuaAscosQe9ecddhtnBg",
		"registry_table_id":   "tblW1KFyrKtUeNF4",
		"registry_view_id":    "vew5zFcUtJ",
	} {
		if config[key] != want || !strings.Contains(status.stdout, want) {
			t.Fatalf("maintainer Trace Registry %s = %#v, want %q:\nconfig=%s\nstatus=%s", key, config[key], want, configBytes, status.stdout)
		}
	}

	assertSuccess(t, run(binary, "flow", "report", "progress", "--root", root, "--step-id", "bootstrap_techdesign", "--status", "in_progress", "--summary", "started"), "flow.report.progress")
	log := string(readFile(t, logPath))
	for _, want := range []string{"--base-token Lu15bIcOuaAscosQe9ecddhtnBg", "--table-id tblW1KFyrKtUeNF4", "--view-id vew5zFcUtJ"} {
		if !strings.Contains(log, want) {
			t.Fatalf("maintainer Trace sync did not use %q:\n%s", want, log)
		}
	}
}

func TestMaintainerTraceSyncPublishesCompleteCLILogAsThirdTarget(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	fakeBin := filepath.Dir(binary)
	logPath := filepath.Join(t.TempDir(), "lark.log")
	traceContentPath := filepath.Join(t.TempDir(), "trace.md")
	cliLogContentPath := filepath.Join(t.TempDir(), "cli-log.md")
	writeAutoSyncLarkCLI(t, filepath.Join(fakeBin, "lark-cli"))
	t.Setenv("FAKE_LARK_LOG", logPath)
	t.Setenv("FAKE_TRACE_CONTENT", traceContentPath)
	t.Setenv("FAKE_CLI_LOG_CONTENT", cliLogContentPath)
	t.Setenv("FAKE_REGISTRY_FIELDS", filepath.Join(t.TempDir(), "registry-fields.json"))
	t.Setenv("FAKE_RECORD_EXISTS", filepath.Join(t.TempDir(), "record-exists"))

	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "fanloop-maintainer", "--title", "Complete CLI log"), "flow.init")
	assertSuccess(t, run(binary, "trace", "bind", "--root", root,
		"--document-url", "https://bytedance.larkoffice.com/docx/AutoSyncTrace",
		"--cli-log-document-url", "https://bytedance.larkoffice.com/docx/AutoSyncCLILog"), "trace.bind")
	localLogPath := filepath.Join(root, ".fanloop", "log", "cli.jsonl")
	before := readFile(t, localLogPath)

	synced := run(binary, "trace", "sync", "--root", root)
	assertSuccess(t, synced, "trace.sync")
	wantTargets := []string{`"target": "trace_document"`, `"target": "cli_log_document"`, `"target": "registry"`}
	lastIndex := -1
	for _, want := range wantTargets {
		index := strings.Index(synced.stdout, want)
		if index <= lastIndex {
			t.Fatalf("Trace targets are missing or out of order at %q:\n%s", want, synced.stdout)
		}
		lastIndex = index
	}
	remote := readFile(t, cliLogContentPath)
	if !bytes.Contains(remote, before) {
		t.Fatalf("CLI log document omitted or changed the pre-sync bytes:\nwant bytes=%q\nremote=%q", before, remote)
	}
	for _, want := range []string{`"arguments"`, `"stdin"`, `"stdout"`, `"stderr"`} {
		if !bytes.Contains(remote, []byte(want)) {
			t.Fatalf("CLI log document is missing %s:\n%s", want, remote)
		}
	}
	if got := readFile(t, localLogPath); len(got) <= len(before) {
		t.Fatalf("trace sync invocation was not appended after the uploaded snapshot")
	}
}

func TestMaintainerTraceSyncRecordsIndependentCLILogFailure(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	fakeBin := filepath.Dir(binary)
	writeAutoSyncLarkCLI(t, filepath.Join(fakeBin, "lark-cli"))
	t.Setenv("FAKE_LARK_LOG", filepath.Join(t.TempDir(), "lark.log"))
	t.Setenv("FAKE_TRACE_CONTENT", filepath.Join(t.TempDir(), "trace.md"))
	t.Setenv("FAKE_CLI_LOG_CONTENT", filepath.Join(t.TempDir(), "cli-log.md"))
	t.Setenv("FAKE_REGISTRY_FIELDS", filepath.Join(t.TempDir(), "registry-fields.json"))
	t.Setenv("FAKE_RECORD_EXISTS", filepath.Join(t.TempDir(), "record-exists"))
	t.Setenv("FAKE_FAIL_TARGET", "cli_log_document")

	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "fanloop-maintainer", "--title", "Partial CLI log"), "flow.init")
	assertSuccess(t, run(binary, "trace", "bind", "--root", root,
		"--document-url", "https://bytedance.larkoffice.com/docx/AutoSyncTrace",
		"--cli-log-document-url", "https://bytedance.larkoffice.com/docx/AutoSyncCLILog"), "trace.bind")

	synced := run(binary, "trace", "sync", "--root", root)
	if synced.exitCode != 1 || !strings.Contains(synced.stdout, `"outcome": "partial"`) {
		t.Fatalf("trace.sync did not return partial: exit=%d\nstdout=%s\nstderr=%s", synced.exitCode, synced.stdout, synced.stderr)
	}
	for _, want := range []string{
		`"target": "trace_document"`, `"target": "cli_log_document"`, `"code": "TRACE_UPDATE_FAILED"`, `"target": "registry"`,
	} {
		if !strings.Contains(synced.stdout, want) {
			t.Fatalf("partial sync is missing %s:\n%s", want, synced.stdout)
		}
	}
	config := string(readFile(t, filepath.Join(root, ".fanloop", "trace", "config.json")))
	for _, want := range []string{
		`"schema_version": 2`,
		`"cli_log_document_url": "https://bytedance.larkoffice.com/docx/AutoSyncCLILog"`,
		`"trace_document_last_sync_at": "`,
		`"cli_log_document_last_sync_error": "CLI log unavailable"`,
		`"registry_last_sync_at": "`,
	} {
		if !strings.Contains(config, want) {
			t.Fatalf("Trace config is missing %q:\n%s", want, config)
		}
	}
}

func TestMaintainerDryRunAcceptsIssueWorkspaceDirectoryOutput(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	workspace := filepath.Join(root, "issue-workspace")
	if err := os.Mkdir(workspace, 0o700); err != nil {
		t.Fatal(err)
	}

	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "fanloop-maintainer", "--title", "Directory Output"), "flow.init")
	reported := run(binary, "flow", "report", "result",
		"--root", root,
		"--dry-run",
		"--step-id", "bootstrap_techdesign",
		"--condition-result", conditionResult("repository_workspace_prepared", "path", `"issue-workspace"`),
		"--next-step-id", "clarify_requirements",
		"--summary", "Issue Workspace prepared",
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

	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "Auto Sync"), "flow.init")
	assertSuccess(t, run(binary, "trace", "bind", "--root", root, "--document-url", "https://bytedance.larkoffice.com/docx/AutoSyncTrace", "--registry", "test"), "trace.bind")
	configBytes := readFile(t, filepath.Join(root, ".fanloop", "trace", "config.json"))
	var config map[string]any
	if err := json.Unmarshal(configBytes, &config); err != nil {
		t.Fatalf("decode Trace config: %v", err)
	}
	for key, want := range map[string]string{
		"registry_profile":    "test",
		"registry_url":        "https://bytedance.larkoffice.com/wiki/GPw5wU3O3ia00MkhN4acCtQDnne?table=tblGrG1epTVE9tHs&view=vew5zFcUtJ",
		"registry_base_token": "A3zNbz0sFapWzVsjHCDcIebdnfg",
		"registry_table_id":   "tblGrG1epTVE9tHs",
		"registry_view_id":    "vew5zFcUtJ",
	} {
		if config[key] != want {
			t.Fatalf("Trace config %s = %#v, want %q:\n%s", key, config[key], want, configBytes)
		}
	}
	first := run(binary, "flow", "report", "progress", "--root", root, "--step-id", "frame_technical_problem", "--status", "in_progress", "--summary", "started")
	assertSuccess(t, first, "flow.report.progress")
	assertRegistryFields(t, registryFieldsPath, "In Progress", "问题定义 / 问题定义 / 技术问题定义")

	second := run(binary, "flow", "report", "progress", "--root", root, "--step-id", "frame_technical_problem", "--status", "blocked", "--summary", "waiting")
	assertSuccess(t, second, "flow.report.progress")
	assertRegistryFields(t, registryFieldsPath, "Blocked", "问题定义 / 问题定义 / 技术问题定义")

	third := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "frame_technical_problem",
		"--condition-result", conditionResult("technical_problem_defined", "path", `".technical-solution/problem.md"`),
		"--next-step-id", "confirm_technical_problem", "--summary", "technical problem defined")
	assertSuccess(t, third, "flow.report.result")
	assertRegistryFields(t, registryFieldsPath, "Human Review", "问题定义 / 问题定义 / 问题人工确认")

	log := string(readFile(t, logPath))
	if got := strings.Count(log, "docs +update"); got != 3 {
		t.Fatalf("each accepted Flow update must sync the Trace document: calls=%d\n%s", got, log)
	}
	if got := strings.Count(log, "base +record-upsert"); got != 3 {
		t.Fatalf("each accepted Flow update must sync the Registry: calls=%d\n%s", got, log)
	}
	for _, want := range []string{"--base-token A3zNbz0sFapWzVsjHCDcIebdnfg", "--table-id tblGrG1epTVE9tHs", "--view-id vew5zFcUtJ"} {
		if !strings.Contains(log, want) {
			t.Fatalf("automatic sync did not use fixed test Registry setting %q:\n%s", want, log)
		}
	}
	if strings.Contains(log, "must_not_be_read") {
		t.Fatalf("automatic sync still reads the removed Registry environment override:\n%s", log)
	}
	if !strings.Contains(log, "whoami --as user") || strings.Contains(log, "auth status") {
		t.Fatalf("automatic sync must resolve the stable user identity through lark-cli whoami:\n%s", log)
	}
	traceContent := string(readFile(t, traceContentPath))
	for _, want := range []string{"# Workflow Trace", "问题定义/问题定义/技术问题定义 → 问题定义/问题定义/问题人工确认", "problem_definition_path"} {
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

	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "Partial Auto Sync"), "flow.init")
	assertSuccess(t, run(binary, "trace", "bind", "--root", root, "--document-url", "https://bytedance.larkoffice.com/docx/AutoSyncTrace"), "trace.bind")
	reported := run(binary, "flow", "report", "progress", "--root", root, "--step-id", "frame_technical_problem", "--status", "blocked", "--summary", "local fact wins")
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

	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "No Auto Sync"), "flow.init")
	assertSuccess(t, run(binary, "trace", "bind", "--root", root, "--document-url", "https://bytedance.larkoffice.com/docx/AutoSyncTrace"), "trace.bind")
	rejected := run(binary, "flow", "report", "result", "--root", root, "--step-id", "write_technical_solution", "--condition-result", conditionResult("technical_solution_written", "path", `"technical-solution.md"`), "--condition-result", conditionResult("architecture_diagram_written", "path", `".technical-solution/architecture.mmd"`), "--next-step-id", "review_technical_solution", "--summary", "stale report")
	if rejected.exitCode == 0 || !strings.Contains(rejected.stderr, `"code": "STEP_NOT_CURRENT"`) {
		t.Fatalf("expected a rejected report:\nstdout=%s\nstderr=%s", rejected.stdout, rejected.stderr)
	}
	dryRun := run(binary, "flow", "report", "progress", "--root", root, "--dry-run", "--step-id", "frame_technical_problem", "--status", "in_progress", "--summary", "dry run")
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

func assertRegistryFields(t *testing.T, path, wantStatus, wantStageStep string) {
	t.Helper()
	var fields map[string]any
	if err := json.Unmarshal(readFile(t, path), &fields); err != nil {
		t.Fatalf("decode Registry fields: %v", err)
	}
	if got := fields["状态"]; got != wantStatus {
		t.Fatalf("Registry status = %#v, want %q; fields=%#v", got, wantStatus, fields)
	}
	if got := fields["trace_key"]; got != "docx:AutoSyncTrace" {
		t.Fatalf("Registry trace_key = %#v", got)
	}
	stageAndAudit, _ := fields["阶段 / 子状态"].(string)
	if stageAndAudit != wantStageStep && !strings.HasPrefix(stageAndAudit, wantStageStep+"\n") {
		t.Fatalf("Registry 阶段 / 子状态 = %#v, want prefix %q", stageAndAudit, wantStageStep)
	}
	updated, _ := fields["更新时间"].(string)
	if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`).MatchString(updated) {
		t.Fatalf("Registry 更新时间 must use the Base datetime format, got %q", updated)
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
	    printf '%s\n' '{"identity":"user","available":true,"tokenStatus":"ready","onBehalfOf":{"openId":"ou_runtime"}}'
    ;;
  "base +record-list")
    if [ -f "$FAKE_RECORD_EXISTS" ]; then
      printf '%s\n' '{"ok":true,"data":{"data":[["docx:AutoSyncTrace"]],"fields":["trace_key"],"record_id_list":["rec_runtime"]}}'
    else
      printf '%s\n' '{"ok":true,"data":{"data":[],"fields":["trace_key"],"record_id_list":[]}}'
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
      printf '%s\n' '{"ok":true,"data":{"record":{"update":{"trace_key":"docx:AutoSyncTrace"}},"updated":true}}'
    else
      printf '%s\n' '{"ok":true,"data":{"record":{"create":{"trace_key":"docx:AutoSyncTrace"}},"created":true}}'
    fi
    ;;
  "base +record-get")
    printf '%s\n' '{"ok":true,"data":{"data":[["docx:AutoSyncTrace"]],"fields":["trace_key"],"record_id_list":["rec_runtime"]}}'
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
