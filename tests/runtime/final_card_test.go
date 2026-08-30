package runtime_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type cardEnvelope struct {
	OK   bool `json:"ok"`
	Data struct {
		Format       string          `json:"format"`
		Content      json.RawMessage `json:"content"`
		SnapshotPath string          `json:"snapshot_path"`
	} `json:"data"`
}

func TestFlowCommandsKeepTraceAndProjectionWithoutSendingPanorama(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	fakeBin := filepath.Dir(binary)
	callLog := filepath.Join(t.TempDir(), "calls.log")
	botmuxCalled := filepath.Join(t.TempDir(), "botmux-called")
	const traceURL = "https://bytedance.larkoffice.com/docx/BootstrapPanoramaTrace"
	writeExecutable(t, filepath.Join(fakeBin, "lark-cli"), `#!/bin/sh
set -eu
printf '%s\n' lark >> "$CALL_LOG"
printf '%s\n' '{"ok":true,"data":{"document":{"url":"https://bytedance.larkoffice.com/docx/BootstrapPanoramaTrace"}}}'
`)
	writeExecutable(t, filepath.Join(fakeBin, "botmux"), `#!/bin/sh
set -eu
: > "$BOTMUX_CALLED"
`)

	command := commandFor(binary, "flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "Bootstrap panorama")
	command.Env = append(command.Env,
		"BOTMUX_CHAT_ID=oc_bootstrap", "BOTMUX_SESSION_ID=session_bootstrap",
		"CALL_LOG="+callLog, "BOTMUX_CALLED="+botmuxCalled,
	)
	assertSuccess(t, collect(command), "flow.init")

	status := run(binary, "trace", "status", "--root", root)
	assertSuccess(t, status, "trace.status")
	if !strings.Contains(status.stdout, `"document_url": "`+traceURL+`"`) {
		t.Fatalf("flow init did not bind the provisioned Trace:\n%s", status.stdout)
	}
	if got := strings.TrimSpace(string(readFile(t, callLog))); got != "lark" {
		t.Fatalf("flow init external calls = %q, want Trace provisioning only", got)
	}
	if _, err := os.Stat(botmuxCalled); !os.IsNotExist(err) {
		t.Fatalf("flow init unexpectedly invoked botmux: %v", err)
	}
	projection := string(readFile(t, filepath.Join(root, ".fanloop", "card", "projection.json")))
	if !strings.Contains(projection, `"trace_document_url": "`+traceURL+`"`) {
		t.Fatalf("Card projection did not retain Trace binding:\n%s", projection)
	}

	assertSuccess(t, run(binary, "flow", "report", "progress", "--root", root,
		"--step-id", "frame_requirement_background", "--status", "in_progress", "--summary", "framing"), "flow.report.progress")
	assertSuccess(t, run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "frame_requirement_background",
		"--condition-result", conditionResult("background_defined", "path", `".technical-solution/sections/01-background.md"`),
		"--next-step-id", "analyze_core_problem", "--summary", "background framed"), "flow.report.result")
	if _, err := os.Stat(botmuxCalled); !os.IsNotExist(err) {
		t.Fatalf("flow report unexpectedly invoked botmux: %v", err)
	}
}

func TestMaintainerFlowInitBindsTraceAndCLILogWithoutSendingPanorama(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	fakeBin := filepath.Dir(binary)
	callLog := filepath.Join(t.TempDir(), "calls.log")
	createCount := filepath.Join(t.TempDir(), "create-count")
	botmuxCalled := filepath.Join(t.TempDir(), "botmux-called")
	const traceURL = "https://bytedance.larkoffice.com/docx/MaintainerPanoramaTrace"
	const logURL = "https://bytedance.larkoffice.com/docx/MaintainerPanoramaCLILog"
	writeExecutable(t, filepath.Join(fakeBin, "lark-cli"), `#!/bin/sh
set -eu
printf '%s\n' lark >> "$CALL_LOG"
if [ -f "$CREATE_COUNT" ]; then
  printf '%s\n' '{"ok":true,"data":{"document":{"url":"https://bytedance.larkoffice.com/docx/MaintainerPanoramaCLILog"}}}'
else
  : > "$CREATE_COUNT"
  printf '%s\n' '{"ok":true,"data":{"document":{"url":"https://bytedance.larkoffice.com/docx/MaintainerPanoramaTrace"}}}'
fi
`)
	writeExecutable(t, filepath.Join(fakeBin, "botmux"), `#!/bin/sh
set -eu
: > "$BOTMUX_CALLED"
`)

	command := commandFor(binary, "flow", "init", "--root", root, "--workflow", "fanloop-maintainer", "--title", "Maintainer panorama")
	command.Env = append(command.Env,
		"BOTMUX_CHAT_ID=oc_bootstrap", "BOTMUX_SESSION_ID=session_bootstrap",
		"CALL_LOG="+callLog, "CREATE_COUNT="+createCount, "BOTMUX_CALLED="+botmuxCalled,
	)
	assertSuccess(t, collect(command), "flow.init")

	status := run(binary, "trace", "status", "--root", root)
	assertSuccess(t, status, "trace.status")
	for _, want := range []string{traceURL, `"cli_log_document_url": "` + logURL + `"`} {
		if !strings.Contains(status.stdout, want) {
			t.Fatalf("flow init did not bind %q:\n%s", want, status.stdout)
		}
	}
	if got := strings.TrimSpace(string(readFile(t, callLog))); got != "lark\nlark" {
		t.Fatalf("flow init external calls = %q, want two Trace document creates", got)
	}
	if _, err := os.Stat(botmuxCalled); !os.IsNotExist(err) {
		t.Fatalf("flow init unexpectedly invoked botmux: %v", err)
	}
	projection := string(readFile(t, filepath.Join(root, ".fanloop", "card", "projection.json")))
	for _, want := range []string{`"trace_document_url": "` + traceURL + `"`, `"cli_log_document_url": "` + logURL + `"`} {
		if !strings.Contains(projection, want) {
			t.Fatalf("Card projection did not retain %q:\n%s", want, projection)
		}
	}
}

func TestFlowInitKeepsLocalFlowButDoesNotSendCardWhenTraceProvisionFails(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	fakeBin := filepath.Dir(binary)
	botmuxCalled := filepath.Join(t.TempDir(), "botmux-called")
	writeExecutable(t, filepath.Join(fakeBin, "lark-cli"), "#!/bin/sh\nexit 9\n")
	writeExecutable(t, filepath.Join(fakeBin, "botmux"), "#!/bin/sh\n: > \"$BOTMUX_CALLED\"\n")

	command := commandFor(binary, "flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "Provision failure")
	command.Env = append(command.Env,
		"BOTMUX_CHAT_ID=oc_bootstrap", "BOTMUX_SESSION_ID=session_bootstrap", "BOTMUX_CALLED="+botmuxCalled,
	)
	initialized := collect(command)
	if initialized.exitCode != 0 || !strings.Contains(initialized.stdout, `"ok": true`) || !strings.Contains(initialized.stderr, "Trace document provisioning") {
		t.Fatalf("flow init did not preserve success with a Trace warning: exit=%d\nstdout=%s\nstderr=%s", initialized.exitCode, initialized.stdout, initialized.stderr)
	}
	if _, err := os.Stat(filepath.Join(root, ".fanloop", "flow", "state.json")); err != nil {
		t.Fatalf("flow init did not preserve the committed local Flow: %v", err)
	}
	for _, path := range []string{
		filepath.Join(root, ".fanloop", "card", "projection.json"),
		botmuxCalled,
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("Trace provisioning failure unexpectedly produced %s: %v", path, err)
		}
	}
}

func TestCardRenderKeepsDriverLayoutAndRawSnapshot(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "Driver layout"), "flow.init")
	projectionPath := filepath.Join(root, ".fanloop", "card", "projection.json")
	if _, err := os.Stat(projectionPath); err != nil {
		t.Fatalf("flow init did not create Card projection: %v", err)
	}
	eventsPath := filepath.Join(root, ".fanloop", "trace", "events.jsonl")
	eventsBefore := readFile(t, eventsPath)

	dryRun := run(binary, "card", "render", "--root", root, "--dry-run", "--view", "panorama", "--format", "lark-json")
	assertSuccess(t, dryRun, "card.render")
	dryRunCard := decodeCard(t, dryRun.stdout)
	if dryRunCard.Data.SnapshotPath != "" {
		t.Fatalf("dry-run snapshot path = %q", dryRunCard.Data.SnapshotPath)
	}
	assertDriverCardLayout(t, dryRunCard.Data.Content)
	if got := readFile(t, projectionPath); len(got) == 0 {
		t.Fatal("card dry-run damaged Card projection")
	}

	first := run(binary, "card", "render", "--root", root, "--view", "panorama", "--format", "lark-json")
	second := run(binary, "card", "render", "--root", root, "--view", "panorama", "--format", "lark-json")
	assertSuccess(t, first, "card.render")
	assertSuccess(t, second, "card.render")
	left, right := decodeCard(t, first.stdout), decodeCard(t, second.stdout)
	if left.Data.SnapshotPath == "" || left.Data.SnapshotPath == right.Data.SnapshotPath {
		t.Fatalf("snapshot paths are not immutable: %q %q", left.Data.SnapshotPath, right.Data.SnapshotPath)
	}
	for _, path := range []string{left.Data.SnapshotPath, right.Data.SnapshotPath} {
		if !strings.HasPrefix(path, ".fanloop/card/") || !strings.HasSuffix(path, ".json") {
			t.Fatalf("snapshot path does not use Driver layout: %q", path)
		}
	}
	if !bytes.Equal(left.Data.Content, right.Data.Content) {
		t.Fatal("the same committed facts rendered different Card content")
	}

	snapshot := readFile(t, filepath.Join(root, filepath.FromSlash(left.Data.SnapshotPath)))
	if !bytes.HasPrefix(snapshot, []byte("{\n  \"schema\": \"2.0\",")) {
		t.Fatalf("snapshot lost the Driver's indented JSON format:\n%s", snapshot)
	}
	if !bytes.Equal(compactJSON(t, snapshot), compactJSON(t, left.Data.Content)) {
		t.Fatalf("snapshot is not the raw sendable Card payload:\n%s", snapshot)
	}
	assertDriverCardLayout(t, snapshot)

	if events := readFile(t, eventsPath); !bytes.Equal(events, eventsBefore) || bytes.Contains(events, []byte(`"kind":"card.rendered"`)) {
		t.Fatalf("Card render changed Trace Events:\n%s", events)
	}
}

func TestCardRenderUsesIndependentProjection(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "Independent card"), "flow.init")

	projectionPath := filepath.Join(root, ".fanloop", "card", "projection.json")
	projection := readFile(t, projectionPath)
	if !bytes.Contains(projection, []byte(`"current_step_id": "frame_requirement_background"`)) {
		t.Fatalf("initial Card projection does not contain the current Step:\n%s", projection)
	}

	reported := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "frame_requirement_background",
		"--condition-result", conditionResult("background_defined", "path", `".technical-solution/sections/01-background.md"`),
		"--next-step-id", "analyze_core_problem", "--summary", "background defined")
	assertSuccess(t, reported, "flow.report.result")
	projection = readFile(t, projectionPath)
	for _, want := range []string{`"current_step_id": "analyze_core_problem"`, `"background_section_path"`} {
		if !bytes.Contains(projection, []byte(want)) {
			t.Fatalf("updated Card projection does not contain %s:\n%s", want, projection)
		}
	}

	eventsPath := filepath.Join(root, ".fanloop", "trace", "events.jsonl")
	brokenTrace := []byte("not-json\n")
	if err := os.WriteFile(eventsPath, brokenTrace, 0o600); err != nil {
		t.Fatal(err)
	}
	rendered := run(binary, "card", "render", "--root", root, "--view", "panorama", "--format", "lark-json")
	assertSuccess(t, rendered, "card.render")
	card := decodeCard(t, rendered.stdout)
	if bytes.Contains(card.Data.Content, []byte("最近一次 Result")) {
		t.Fatalf("Card still renders latest Result:\n%s", card.Data.Content)
	}
	if got := readFile(t, eventsPath); !bytes.Equal(got, brokenTrace) {
		t.Fatalf("Card render changed Trace Events:\n%s", got)
	}
}

func TestTechnicalSolutionCardHasNoInventedURLOutputs(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "URL artifacts"), "flow.init")

	rendered := run(binary, "card", "render", "--root", root, "--dry-run", "--view", "panorama", "--format", "lark-json")
	assertSuccess(t, rendered, "card.render")
	outputs := outputColumnContents(t, decodeCard(t, rendered.stdout).Data.Content)

	if !strings.Contains(outputs, "暂未生成") {
		t.Fatalf("technical solution path Outputs must keep the URL section empty:\n%s", outputs)
	}
}

func TestCardRenderDoesNotFallBackToNonURLOutputs(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "No URL artifacts"), "flow.init")

	rendered := run(binary, "card", "render", "--root", root, "--dry-run", "--view", "panorama", "--format", "lark-json")
	assertSuccess(t, rendered, "card.render")
	outputs := outputColumnContents(t, decodeCard(t, rendered.stdout).Data.Content)
	if strings.Contains(outputs, "background_section_path") {
		t.Fatalf("non-URL Condition Output must not be rendered as a stage output:\n%s", outputs)
	}
	if !strings.Contains(outputs, "暂未生成") {
		t.Fatalf("a stage without URL Outputs must keep an empty-state message:\n%s", outputs)
	}
}

func assertDriverCardLayout(t *testing.T, content []byte) {
	t.Helper()
	var root map[string]json.RawMessage
	if err := json.Unmarshal(content, &root); err != nil {
		t.Fatal(err)
	}
	if len(root) != 4 {
		t.Fatalf("Card root keys changed: %v", root)
	}
	for _, key := range []string{"schema", "config", "header", "body"} {
		if _, ok := root[key]; !ok {
			t.Fatalf("Card root is missing %q: %s", key, content)
		}
	}
	var value struct {
		Schema string `json:"schema"`
		Header struct {
			Template string `json:"template"`
			Title    struct {
				Content string `json:"content"`
			} `json:"title"`
			Subtitle struct {
				Content string `json:"content"`
			} `json:"subtitle"`
			TextTagList []json.RawMessage `json:"text_tag_list"`
		} `json:"header"`
		Body struct {
			Elements []json.RawMessage `json:"elements"`
		} `json:"body"`
	}
	if err := json.Unmarshal(content, &value); err != nil {
		t.Fatal(err)
	}
	if value.Schema != "2.0" || value.Header.Template != "default" ||
		value.Header.Title.Content != "后端研发交付 · Driver layout" ||
		value.Header.Subtitle.Content != "问题定义 · 需求背景" || len(value.Header.TextTagList) != 2 {
		t.Fatalf("Driver header contract was lost: %s", content)
	}
	if len(value.Body.Elements) != 5 {
		t.Fatalf("Driver body element count = %d, want 5: %s", len(value.Body.Elements), content)
	}
	wantTags := []string{"column_set", "markdown", "column_set", "markdown", "column_set"}
	for index, want := range wantTags {
		var element struct {
			Tag     string `json:"tag"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(value.Body.Elements[index], &element); err != nil {
			t.Fatal(err)
		}
		if element.Tag != want {
			t.Fatalf("body element %d tag = %q, want %q", index, element.Tag, want)
		}
		if index == 1 && element.Content != "**各阶段 Output**" {
			t.Fatalf("Output heading = %q", element.Content)
		}
	}
	for _, want := range []string{"状态全景", "需求背景", "总体方案", "方案终审", "当前执行证据", "当前进行中"} {
		if !bytes.Contains(content, []byte(want)) {
			t.Fatalf("Driver panorama does not contain %q: %s", want, content)
		}
	}
}

func decodeCard(t *testing.T, raw string) cardEnvelope {
	t.Helper()
	var value cardEnvelope
	if err := json.Unmarshal([]byte(raw), &value); err != nil || !value.OK {
		t.Fatalf("decode Card envelope: %v\n%s", err, raw)
	}
	return value
}

func outputColumnContents(t *testing.T, content []byte) string {
	t.Helper()
	var card struct {
		Body struct {
			Elements []json.RawMessage `json:"elements"`
		} `json:"body"`
	}
	if err := json.Unmarshal(content, &card); err != nil {
		t.Fatal(err)
	}
	if len(card.Body.Elements) < 3 {
		t.Fatalf("Card does not contain the stage Output column set: %s", content)
	}
	var columnSet struct {
		Columns []struct {
			Elements []struct {
				Content string `json:"content"`
			} `json:"elements"`
		} `json:"columns"`
	}
	if err := json.Unmarshal(card.Body.Elements[2], &columnSet); err != nil {
		t.Fatal(err)
	}
	contents := make([]string, 0, len(columnSet.Columns))
	for _, column := range columnSet.Columns {
		for _, element := range column.Elements {
			contents = append(contents, element.Content)
		}
	}
	return strings.Join(contents, "\n")
}

func compactJSON(t *testing.T, raw []byte) []byte {
	t.Helper()
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	output, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return output
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}
