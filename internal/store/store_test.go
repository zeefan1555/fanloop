package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/traceconfig"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

func TestCommitAndLoadBoundValidateStateEventTail(t *testing.T) {
	root := t.TempDir()
	local, failure := New(root)
	if failure != nil {
		t.Fatal(failure)
	}
	loaded, err := workflow.Load("technical-solution-design")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	step, _ := loaded.Workflow.FirstStepID()
	current := state.State{
		SchemaVersion: state.CurrentStateSchemaVersion, Requirement: state.Requirement{Title: "Store"},
		Release:       state.Release{Version: "dev", Workflow: state.WorkflowRefFrom(loaded.Ref)},
		CurrentStepID: &step, CurrentStepStatus: state.StepReady, CurrentStepSummary: "workflow initialized",
		Outputs: map[string]state.RegisteredOutput{}, Integrations: state.Integrations{}, LastEventID: "e1", CreatedAt: now, UpdatedAt: now,
	}
	event := state.Event{
		SchemaVersion: state.CurrentEventSchemaVersion, ID: "e1", OccurredAt: now, Kind: state.EventFlowInitialized,
		Command: "flow.init", Workflow: state.WorkflowRefFrom(loaded.Ref),
		Payload: state.Payload(state.FlowInitializedPayload{StepID: step, StepStatus: state.StepReady}),
	}
	if failure := local.Commit(current, event); failure != nil {
		t.Fatal(failure)
	}
	read, bound, failure := local.LoadBound()
	if failure != nil || read.LastEventID != "e1" || bound.Ref != loaded.Ref {
		t.Fatalf("read = %#v, bound = %#v, failure = %v", read, bound.Ref, failure)
	}
	if content, err := os.ReadFile(filepath.Join(root, ".fanloop", "trace", "events.md")); err != nil || !strings.Contains(string(content), "# PRD Flow Trace") || !strings.Contains(string(content), "Workflow 已初始化") {
		t.Fatalf("projection = %q, error = %v", content, err)
	}
}

func TestCommitWritesStorageThriftJSON(t *testing.T) {
	root := t.TempDir()
	committedWorkflow(t, root)

	for relative, wantVersion := range map[string]float64{
		".fanloop/flow/state.json":   12,
		".fanloop/output/state.json": 3,
	} {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		var document map[string]any
		if err := json.Unmarshal(content, &document); err != nil {
			t.Fatal(err)
		}
		if document["schema_version"] != wantVersion {
			t.Fatalf("%s schema_version = %v, want %v", relative, document["schema_version"], wantVersion)
		}
	}

	content, err := os.ReadFile(filepath.Join(root, ".fanloop", "trace", "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	line := strings.TrimSpace(string(content))
	var event map[string]any
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		t.Fatal(err)
	}
	if event["schema_version"] != float64(12) || event["kind"] != "flow_initialized" {
		t.Fatalf("Event header = %#v", event)
	}
	payload, ok := event["payload"].(map[string]any)
	if !ok || len(payload) != 1 || payload["flow_initialized"] == nil {
		t.Fatalf("Event payload = %#v, want one flow_initialized union member", event["payload"])
	}
}

func TestLoadBoundRejectsTraceURLTamperedOutsideEvents(t *testing.T) {
	root := t.TempDir()
	local, current := committedWorkflow(t, root)
	current.Integrations.Trace = &state.TraceBinding{DocumentURL: "https://bytedance.larkoffice.com/docx/forged", Registry: traceconfig.RegistryProduction}
	content, err := state.Encode(current)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".fanloop", "flow", "state.json"), content, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, failure := local.LoadBound(); failure == nil || failure.Code != erroridl.ErrorCode_STATE_CORRUPT {
		t.Fatalf("failure = %#v", failure)
	}
}

func TestTraceConfigWritesStorageThriftSchema(t *testing.T) {
	_, current := committedWorkflow(t, t.TempDir())
	current.Integrations.Trace = &state.TraceBinding{
		DocumentURL: "https://bytedance.larkoffice.com/docx/Trace",
		Registry:    traceconfig.RegistryProduction,
	}
	content, ok := RenderTraceConfig(current, nil)
	if !ok {
		t.Fatal("Trace config was not rendered")
	}
	var document map[string]any
	if err := json.Unmarshal(content, &document); err != nil {
		t.Fatal(err)
	}
	if document["schema_version"] != float64(2) {
		t.Fatalf("schema_version = %v, want 2", document["schema_version"])
	}
	if document["trace_document_url"] != current.Integrations.Trace.DocumentURL {
		t.Fatalf("trace_document_url = %v", document["trace_document_url"])
	}
	if _, retired := document["trace_doc_url"]; retired {
		t.Fatal("retired trace_doc_url is still present")
	}
}

func TestTraceProjectionDoesNotPresentMeegoSourceAsPRD(t *testing.T) {
	loaded, err := workflow.Load("technical-solution-design")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 19, 0, 0, 0, 0, time.UTC)
	step, _ := loaded.Workflow.FirstStepID()
	current := state.State{
		SchemaVersion: state.CurrentStateSchemaVersion,
		Requirement: state.Requirement{
			Title:     "Meego source",
			SourceURL: "https://meego.larkoffice.com/aweme/story/detail/7363677776",
		},
		Release:           state.Release{Version: "dev", Workflow: state.WorkflowRefFrom(loaded.Ref)},
		CurrentStepID:     &step,
		CurrentStepStatus: state.StepReady,
		Outputs: map[string]state.RegisteredOutput{
			"requirement_document_url": {Type: workflow.OutputURL, Value: json.RawMessage(`"https://bytedance.larkoffice.com/docx/ReadablePRD"`), ProducerStepID: "frame_technical_problem"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	projection := string(RenderEvents("/tmp/requirement", current, loaded.Workflow, nil))
	for _, want := range []string{
		"- Meego 需求：https://meego.larkoffice.com/aweme/story/detail/7363677776",
		"- PRD 文档：https://bytedance.larkoffice.com/docx/ReadablePRD",
	} {
		if !strings.Contains(projection, want) {
			t.Fatalf("Trace projection does not contain %q:\n%s", want, projection)
		}
	}
	if strings.Contains(projection, "- PRD 文档：https://meego.larkoffice.com") {
		t.Fatalf("Trace projection still presents Meego as PRD:\n%s", projection)
	}
}

func TestMaintainerTraceProjectionSeparatesArtifactsFromPRD(t *testing.T) {
	loaded, err := workflow.Load("fanloop-maintainer")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	step, _ := loaded.Workflow.FirstStepID()
	current := state.State{
		SchemaVersion: state.CurrentStateSchemaVersion,
		Requirement:   state.Requirement{Title: "Self iteration"},
		Release:       state.Release{Version: "dev", Workflow: state.WorkflowRefFrom(loaded.Ref)},
		CurrentStepID: &step, CurrentStepStatus: state.StepReady,
		Outputs: map[string]state.RegisteredOutput{
			"requirement_document_url":      {Type: workflow.OutputURL, Value: json.RawMessage(`"https://bytedance.larkoffice.com/docx/Requirements"`), ProducerStepID: "clarify_requirements"},
			"technical_design_document_url": {Type: workflow.OutputURL, Value: json.RawMessage(`"https://bytedance.larkoffice.com/docx/Design"`), ProducerStepID: "design_technical_solution"},
			"merge_request_urls":            {Type: workflow.OutputURLList, Value: json.RawMessage(`["https://github.com/zeefan1555/fanloop/merge_requests/123"]`), ProducerStepID: "handoff_merge_request"},
		},
		Integrations: state.Integrations{Trace: &state.TraceBinding{
			DocumentURL: "https://bytedance.larkoffice.com/docx/Trace", Registry: traceconfig.RegistryProduction,
			CLILogDocumentURL: "https://bytedance.larkoffice.com/docx/CLILog",
		}},
		LastEventID: "e1", CreatedAt: now, UpdatedAt: now,
	}
	projection := string(RenderEvents("/tmp/requirement", current, loaded.Workflow, nil))
	for _, want := range []string{
		"- PRD 文档：未提供",
		"- 需求澄清文档：https://bytedance.larkoffice.com/docx/Requirements",
		"- 技术方案文档：https://bytedance.larkoffice.com/docx/Design",
		"- MR：https://github.com/zeefan1555/fanloop/merge_requests/123",
		"- CLI 日志：https://bytedance.larkoffice.com/docx/CLILog",
	} {
		if !strings.Contains(projection, want) {
			t.Fatalf("maintainer Trace projection does not contain %q:\n%s", want, projection)
		}
	}
	if strings.Contains(projection, "- PRD 文档：https://bytedance.larkoffice.com/docx/Requirements") {
		t.Fatalf("maintainer requirements document still masquerades as PRD:\n%s", projection)
	}
	current.Requirement.SourceURL = "https://bytedance.larkoffice.com/docx/RealPRD"
	if projection := string(RenderEvents("/tmp/requirement", current, loaded.Workflow, nil)); !strings.Contains(projection, "- PRD 文档："+current.Requirement.SourceURL) {
		t.Fatalf("maintainer Trace omitted a real PRD:\n%s", projection)
	}
}

func TestRequirementLinksOnlyProjectSupportedDocumentSourcesAsPRD(t *testing.T) {
	for _, test := range []struct {
		source string
		prd    string
	}{
		{source: "https://bytedance.larkoffice.com/docx/ReadablePRD", prd: "https://bytedance.larkoffice.com/docx/ReadablePRD"},
		{source: "https://bytedance.larkoffice.com/wiki/ReadablePRD", prd: "https://bytedance.larkoffice.com/wiki/ReadablePRD"},
		{source: "https://github.com/zeefan1555/fanloop", prd: ""},
		{source: "https://example.com/requirements", prd: ""},
	} {
		_, got := RequirementLinks(state.State{Requirement: state.Requirement{SourceURL: test.source}})
		if got != test.prd {
			t.Fatalf("RequirementLinks(%q) PRD = %q, want %q", test.source, got, test.prd)
		}
	}
}

func TestCommitRejectsProgressWithAnOutputPayload(t *testing.T) {
	root := t.TempDir()
	local, current := committedWorkflow(t, root)
	from := state.StateRef(current)
	current.CurrentStepStatus = state.StepInProgress
	current.LastEventID = "e2"
	current.UpdatedAt = current.UpdatedAt.Add(time.Minute)
	event := state.Event{
		SchemaVersion: state.CurrentEventSchemaVersion, ID: "e2", OccurredAt: current.UpdatedAt, Kind: state.EventFlowProgressed,
		Command: "flow.report.progress", Workflow: current.Release.Workflow, CausedByEventID: "e1",
		Payload: state.Payload(struct {
			state.FlowProgressPayload
			OutputChanges state.OutputChanges `json:"output_changes"`
		}{FlowProgressPayload: state.FlowProgressPayload{
			FromStepID: from.StepID, FromStepStatus: from.Status, ToStepStatus: state.StepInProgress, Summary: "working",
		}, OutputChanges: state.OutputChanges{
			Accepted: []string{"repository_scope_path"},
		}}),
	}
	if failure := local.Commit(current, event); failure == nil || failure.Code != erroridl.ErrorCode_STATE_CORRUPT {
		t.Fatalf("failure = %#v", failure)
	}
	events, failure := local.Events()
	if failure != nil || len(events) != 1 {
		t.Fatalf("events = %d, failure = %v", len(events), failure)
	}
}

func TestConcurrentCommitsFromOneEventTailAreSerialized(t *testing.T) {
	root := t.TempDir()
	local, current := committedWorkflow(t, root)
	stepID := *current.CurrentStepID
	start := make(chan struct{})
	results := make(chan *erroridl.PublicError, 2)
	var ready sync.WaitGroup
	ready.Add(2)

	for index, eventID := range []string{"e2", "e3"} {
		next := current
		next.CurrentStepStatus = state.StepInProgress
		next.CurrentStepSummary = "working"
		next.LastEventID = eventID
		next.UpdatedAt = current.UpdatedAt.Add(time.Duration(index+1) * time.Minute)
		event := state.Event{
			SchemaVersion: state.CurrentEventSchemaVersion, ID: eventID, OccurredAt: next.UpdatedAt, Kind: state.EventFlowProgressed,
			Command: "flow.report.progress", Workflow: current.Release.Workflow, CausedByEventID: current.LastEventID,
			Payload: state.Payload(state.FlowProgressPayload{
				FromStepID: stepID, FromStepStatus: state.StepReady, ToStepStatus: state.StepInProgress, Summary: "working",
			}),
		}
		go func() {
			ready.Done()
			<-start
			results <- local.Commit(next, event)
		}()
	}
	ready.Wait()
	close(start)

	succeeded, failed := 0, 0
	for range 2 {
		if failure := <-results; failure == nil {
			succeeded++
		} else {
			failed++
		}
	}
	if succeeded != 1 || failed != 1 {
		t.Fatalf("concurrent commits: succeeded=%d failed=%d", succeeded, failed)
	}
	loaded, _, failure := local.LoadBound()
	if failure != nil {
		t.Fatal(failure)
	}
	events, failure := local.Events()
	if failure != nil || len(events) != 2 || loaded.LastEventID != events[1].ID {
		t.Fatalf("events=%d last_event_id=%q failure=%v", len(events), loaded.LastEventID, failure)
	}
	if info, err := os.Stat(filepath.Join(root, ".fanloop", "flow", "state.lock")); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("persistent Requirement lock is missing or invalid: info=%v error=%v", info, err)
	}
}

func committedWorkflow(t *testing.T, root string) (*Store, state.State) {
	t.Helper()
	local, failure := New(root)
	if failure != nil {
		t.Fatal(failure)
	}
	loaded, err := workflow.Load("technical-solution-design")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	step, _ := loaded.Workflow.FirstStepID()
	current := state.State{SchemaVersion: state.CurrentStateSchemaVersion, Requirement: state.Requirement{Title: "Store"}, Release: state.Release{Version: "dev", Workflow: state.WorkflowRefFrom(loaded.Ref)}, CurrentStepID: &step, CurrentStepStatus: state.StepReady, CurrentStepSummary: "workflow initialized", Outputs: map[string]state.RegisteredOutput{}, Integrations: state.Integrations{}, LastEventID: "e1", CreatedAt: now, UpdatedAt: now}
	event := state.Event{SchemaVersion: state.CurrentEventSchemaVersion, ID: "e1", OccurredAt: now, Kind: state.EventFlowInitialized, Command: "flow.init", Workflow: current.Release.Workflow, Payload: state.Payload(state.FlowInitializedPayload{StepID: step, StepStatus: state.StepReady})}
	if failure := local.Commit(current, event); failure != nil {
		t.Fatal(failure)
	}
	return local, current
}
