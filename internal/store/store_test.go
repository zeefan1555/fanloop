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
	loaded, err := workflow.Load("material-flashcards")
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
	if content, err := os.ReadFile(filepath.Join(root, ".fanloop", "trace", "events.md")); err != nil || !strings.Contains(string(content), "# Workflow Trace") || !strings.Contains(string(content), "Workflow 已初始化") {
		t.Fatalf("projection = %q, error = %v", content, err)
	}
}

func TestCommitWritesStorageThriftJSON(t *testing.T) {
	root := t.TempDir()
	committedWorkflow(t, root)

	for relative, wantVersion := range map[string]float64{
		".fanloop/flow/state.json":   13,
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
	if event["schema_version"] != float64(13) || event["kind"] != "flow_initialized" {
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

func TestTraceProjectionUsesGenericRequirementAndWorkflowFacts(t *testing.T) {
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
		Outputs:           map[string]state.RegisteredOutput{},
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	projection := string(RenderEvents("/tmp/requirement", current, loaded.Workflow, nil))
	for _, want := range []string{
		"# Workflow Trace",
		"- Requirement：Meego source",
		"- 来源：https://meego.larkoffice.com/aweme/story/detail/7363677776",
		"- Requirement Root：/tmp/requirement",
		"- Workflow：technical-solution-design",
		"- Release：dev",
	} {
		if !strings.Contains(projection, want) {
			t.Fatalf("Trace projection does not contain %q:\n%s", want, projection)
		}
	}
}

func TestMaintainerTracePanoramaShowsThreeStageDelivery(t *testing.T) {
	loaded, err := workflow.Load("fanloop-maintainer")
	if err != nil {
		t.Fatal(err)
	}
	step, _ := loaded.Workflow.FirstStepID()
	current := state.State{
		Requirement:       state.Requirement{Title: "Job hierarchy"},
		Release:           state.Release{Version: "dev", Workflow: state.WorkflowRefFrom(loaded.Ref)},
		CurrentStepID:     &step,
		CurrentStepStatus: state.StepReady,
		Outputs:           map[string]state.RegisteredOutput{},
	}
	projection := string(RenderEvents("/tmp/requirement", current, loaded.Workflow, nil))
	for _, want := range []string{
		"TechDesign：**仓库范围确定（Ready）** → 需求澄清 → 方案设计 → 方案自主评审",
		"Implement：代码实现与过程 CR → 整体 Code Review",
		"Test：Agent 端到端测试 → 人类端到端测试 → PR 合码与本地更新",
	} {
		if !strings.Contains(projection, want) {
			t.Fatalf("Trace projection does not contain %q:\n%s", want, projection)
		}
	}
}

func TestTracePanoramaRendersSkippedStepsWithoutCompletionMark(t *testing.T) {
	loaded, err := workflow.Load("technical-solution-design")
	if err != nil {
		t.Fatal(err)
	}
	stepID := "research_solution_options"
	current := state.State{CurrentStepID: &stepID, CurrentStepStatus: state.StepAwaitingConfirmation, SkippedStepIDs: []string{"frame_requirement_background"}, Outputs: map[string]state.RegisteredOutput{}}
	panorama := tracePanorama(current, loaded.Workflow)
	if !strings.Contains(panorama, "已跳过 业务背景") || strings.Contains(panorama, "✅ 业务背景") {
		t.Fatalf("skipped Step was rendered as completed:\n%s", panorama)
	}
}

func TestTraceProjectionListsOutputsWithoutBusinessSpecificSections(t *testing.T) {
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
			"artifact_urls":                 {Type: workflow.OutputURLList, Value: json.RawMessage(`["https://example.com/artifacts/123"]`), ProducerStepID: "review_code"},
		},
		Integrations: state.Integrations{Trace: &state.TraceBinding{
			DocumentURL: "https://bytedance.larkoffice.com/docx/Trace", Registry: traceconfig.RegistryProduction,
			CLILogDocumentURL: "https://bytedance.larkoffice.com/docx/CLILog",
		}},
		LastEventID: "e1", CreatedAt: now, UpdatedAt: now,
	}
	projection := string(RenderEvents("/tmp/requirement", current, loaded.Workflow, nil))
	for _, want := range []string{
		"| requirement_document_url | url | clarify_requirements | https://bytedance.larkoffice.com/docx/Requirements |",
		"| technical_design_document_url | url | design_technical_solution | https://bytedance.larkoffice.com/docx/Design |",
		"| artifact_urls | url_list | review_code | [\"https://example.com/artifacts/123\"] |",
		"📜 CLI 日志：[查看完整输入输出](https://bytedance.larkoffice.com/docx/CLILog)",
	} {
		if !strings.Contains(projection, want) {
			t.Fatalf("Trace projection does not contain %q:\n%s", want, projection)
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

func TestCommitValidatesNewResultAgainstCurrentRoutesAcrossDigests(t *testing.T) {
	root := t.TempDir()
	local, failure := New(root)
	if failure != nil {
		t.Fatal(failure)
	}
	loaded, err := workflow.Load("fanloop-maintainer")
	if err != nil {
		t.Fatal(err)
	}
	oldRef := state.WorkflowRef{ID: loaded.Ref.ID, Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	now := time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC)
	step, _ := loaded.Workflow.FirstStepID()
	current := state.State{
		SchemaVersion: state.CurrentStateSchemaVersion, Requirement: state.Requirement{Title: "Cross digest Store"},
		Release: state.Release{Version: "old", Workflow: oldRef}, CurrentStepID: &step, CurrentStepStatus: state.StepReady,
		CurrentStepSummary: "workflow initialized", Outputs: map[string]state.RegisteredOutput{}, Integrations: state.Integrations{},
		LastEventID: "e1", CreatedAt: now, UpdatedAt: now,
	}
	initialized := state.Event{
		SchemaVersion: state.CurrentEventSchemaVersion, ID: "e1", OccurredAt: now, Kind: state.EventFlowInitialized,
		Command: "flow.init", Workflow: oldRef, Payload: state.Payload(state.FlowInitializedPayload{StepID: step, StepStatus: state.StepReady}),
	}
	if failure := local.Commit(current, initialized); failure != nil {
		t.Fatal(failure)
	}

	target := "clarify_requirements"
	next := current
	next.CurrentStepID = &target
	next.CurrentStepSummary = "invalid route"
	next.Outputs = map[string]state.RegisteredOutput{
		"issue_workspace_path": {Type: workflow.OutputPath, Value: json.RawMessage(`"issue-workspace"`), ProducerStepID: step},
	}
	next.LastEventID = "e2"
	next.UpdatedAt = now.Add(time.Minute)
	result := state.Event{
		SchemaVersion: state.CurrentEventSchemaVersion, ID: "e2", OccurredAt: next.UpdatedAt, Kind: state.EventFlowResult,
		Command: "flow.report.result", Workflow: oldRef, CausedByEventID: "e1",
		Payload: state.Payload(state.FlowResultPayload{
			ConditionResults: []state.ConditionResult{{ConditionID: "repository_workspace_prepared", Output: state.OutputValue{Type: workflow.OutputPath, Value: json.RawMessage(`"issue-workspace"`)}}},
			Summary:          "invalid route", Effect: state.ResultAdvanced,
			Transition:    state.Transition{Direction: state.TransitionFlow, FromStepID: step, ToStepID: target},
			OutputChanges: state.OutputChanges{Accepted: []string{"issue_workspace_path"}},
		}),
	}
	if failure := local.Commit(next, result); failure == nil || failure.Code != erroridl.ErrorCode_STATE_CORRUPT || !strings.Contains(failure.Message, "Flow Route") {
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
	loaded, err := workflow.Load("material-flashcards")
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
