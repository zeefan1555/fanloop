package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zeefan1555/fanloop/internal/cardsnapshot"
	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/traceconfig"
	"github.com/zeefan1555/fanloop/internal/workflow"
	"github.com/zeefan1555/fanloop/internal/workflowview"
)

func TestCommitAndLoadBoundValidateStateEventTail(t *testing.T) {
	root := t.TempDir()
	local, failure := New(root)
	if failure != nil {
		t.Fatal(failure)
	}
	loaded, err := loadRuntimeControls(t)
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
		".fanloop/flow/state.json":   float64(state.CurrentStateSchemaVersion),
		".fanloop/output/state.json": 4,
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
	if event["schema_version"] != float64(state.CurrentEventSchemaVersion) || event["kind"] != "flow_initialized" {
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
	current.Integrations.Trace = &state.TraceBinding{DocumentURL: "https://docs.example.invalid/docx/forged", Registry: traceconfig.RegistryProduction}
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
		DocumentURL: "https://docs.example.invalid/docx/Trace",
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
	loaded, err := loadRuntimeControls(t)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 19, 0, 0, 0, 0, time.UTC)
	step, _ := loaded.Workflow.FirstStepID()
	current := state.State{
		SchemaVersion: state.CurrentStateSchemaVersion,
		Requirement: state.Requirement{
			Title:     "Work-item source",
			SourceURL: "https://work-items.example.invalid/story/detail/example-1",
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
		"- Requirement：Work-item source",
		"- 来源：https://work-items.example.invalid/story/detail/example-1",
		"- Requirement Root：/tmp/requirement",
		"- Workflow：linear-runtime",
		"- Release：dev",
	} {
		if !strings.Contains(projection, want) {
			t.Fatalf("Trace projection does not contain %q:\n%s", want, projection)
		}
	}
}

func TestTracePanoramaShowsConfiguredSteps(t *testing.T) {
	loaded, err := loadRuntimeControls(t)
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
		"Runtime：**First（Ready）** → Second → Third",
	} {
		if !strings.Contains(projection, want) {
			t.Fatalf("Trace projection does not contain %q:\n%s", want, projection)
		}
	}
}

func TestTracePanoramaRendersSkippedStepsWithoutCompletionMark(t *testing.T) {
	loaded, err := loadRuntimeControls(t)
	if err != nil {
		t.Fatal(err)
	}
	stepID := "second"
	current := state.State{CurrentStepID: &stepID, CurrentStepStatus: state.StepAwaitingConfirmation, SkippedStepIDs: []string{"first"}, Outputs: map[string]state.RegisteredOutput{}}
	panorama := tracePanorama(workflowview.Display(loaded.Workflow, current))
	if !strings.Contains(panorama, "已跳过 First") || strings.Contains(panorama, "✅ First") {
		t.Fatalf("skipped Step was rendered as completed:\n%s", panorama)
	}
}

func TestTraceProjectionListsOutputsWithoutBusinessSpecificSections(t *testing.T) {
	loaded, err := loadRuntimeControls(t)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	step, _ := loaded.Workflow.FirstStepID()
	current := state.State{
		SchemaVersion: state.CurrentStateSchemaVersion,
		Requirement:   state.Requirement{Title: "Synthetic workflow"},
		Release:       state.Release{Version: "dev", Workflow: state.WorkflowRefFrom(loaded.Ref)},
		CurrentStepID: &step, CurrentStepStatus: state.StepReady,
		Outputs: map[string]state.RegisteredOutput{
			"first_artifact":  {Type: workflow.OutputPath, Value: json.RawMessage(`"first.md"`), ProducerStepID: "first"},
			"second_artifact": {Type: workflow.OutputPath, Value: json.RawMessage(`"second.md"`), ProducerStepID: "second"},
			"artifact_urls":   {Type: workflow.OutputURLList, Value: json.RawMessage(`["https://example.com/artifact/123"]`), ProducerStepID: "third"},
		},
		Integrations: state.Integrations{Trace: &state.TraceBinding{
			DocumentURL: "https://docs.example.invalid/docx/Trace", Registry: traceconfig.RegistryProduction,
			CLILogDocumentURL: "https://docs.example.invalid/docx/CLILog",
		}},
		LastEventID: "e1", CreatedAt: now, UpdatedAt: now,
	}
	projection := string(RenderEvents("/tmp/requirement", current, loaded.Workflow, nil))
	for _, want := range []string{
		"| first_artifact | path | first | first.md |",
		"| second_artifact | path | second | second.md |",
		"| artifact_urls | url_list | third | [\"https://example.com/artifact/123\"] |",
		"📜 CLI 日志：[查看完整输入输出](https://docs.example.invalid/docx/CLILog)",
	} {
		if !strings.Contains(projection, want) {
			t.Fatalf("Trace projection does not contain %q:\n%s", want, projection)
		}
	}
}

func TestCommitRejectsProgressWithAnOutputPayload(t *testing.T) {
	root := t.TempDir()
	local, current := committedWorkflow(t, root)
	current = acknowledgeStorePanorama(t, local, current, "e1", "e2")
	from := state.StateRef(current)
	cause := current.LastEventID
	current.CurrentStepStatus = state.StepInProgress
	current.LastEventID = "e3"
	current.UpdatedAt = current.UpdatedAt.Add(time.Minute)
	event := state.Event{
		SchemaVersion: state.CurrentEventSchemaVersion, ID: "e3", OccurredAt: current.UpdatedAt, Kind: state.EventFlowProgressed,
		Command: "flow.report.progress", Workflow: current.Release.Workflow, CausedByEventID: cause,
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
	if failure != nil || len(events) != 2 {
		t.Fatalf("events = %d, failure = %v", len(events), failure)
	}
}

func TestCommitValidatesNewResultAgainstCurrentRoutes(t *testing.T) {
	root := t.TempDir()
	local, failure := New(root)
	if failure != nil {
		t.Fatal(failure)
	}
	loaded, err := loadRuntimeControls(t)
	if err != nil {
		t.Fatal(err)
	}
	oldRef := state.WorkflowRef{ID: loaded.Ref.ID}
	now := time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC)
	step, _ := loaded.Workflow.FirstStepID()
	current := state.State{
		SchemaVersion: state.CurrentStateSchemaVersion, Requirement: state.Requirement{Title: "Current Workflow Store"},
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
	current = acknowledgeStorePanorama(t, local, current, "e1", "e-ack")

	target := "third"
	next := current
	next.CurrentStepID = &target
	next.CurrentStepSummary = "invalid route"
	next.Outputs = map[string]state.RegisteredOutput{"first_artifact": {Type: workflow.OutputPath, Value: json.RawMessage(`"first.md"`), ProducerStepID: step}}
	next.LastEventID = "e2"
	next.UpdatedAt = now.Add(time.Minute)
	result := state.Event{
		SchemaVersion: state.CurrentEventSchemaVersion, ID: "e2", OccurredAt: next.UpdatedAt, Kind: state.EventFlowResult,
		Command: "flow.report.result", Workflow: oldRef, CausedByEventID: current.LastEventID,
		Payload: state.Payload(state.FlowResultPayload{
			ConditionResults: []state.ConditionResult{{ConditionID: "first_completed", Output: state.OutputValue{Type: workflow.OutputPath, Value: json.RawMessage(`"first.md"`)}}},
			Summary:          "invalid route", Effect: state.ResultAdvanced,
			Transition:    state.Transition{Direction: state.TransitionFlow, FromStepID: step, ToStepID: target},
			OutputChanges: state.OutputChanges{Accepted: []string{"first_artifact"}},
		}),
	}
	if failure := local.Commit(next, result); failure == nil || failure.Code != erroridl.ErrorCode_STATE_CORRUPT || !strings.Contains(failure.Message, "Flow Route") {
		t.Fatalf("failure = %#v", failure)
	}
	events, failure := local.Events()
	if failure != nil || len(events) != 2 {
		t.Fatalf("events = %d, failure = %v", len(events), failure)
	}
}

func TestConcurrentCommitsFromOneEventTailAreSerialized(t *testing.T) {
	root := t.TempDir()
	local, current := committedWorkflow(t, root)
	current = acknowledgeStorePanorama(t, local, current, "e1", "e2")
	stepID := *current.CurrentStepID
	start := make(chan struct{})
	results := make(chan *erroridl.PublicError, 2)
	var ready sync.WaitGroup
	ready.Add(2)

	for index, eventID := range []string{"e3", "e4"} {
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
	if failure != nil || len(events) != 3 || loaded.LastEventID != events[2].ID {
		t.Fatalf("events=%d last_event_id=%q failure=%v", len(events), loaded.LastEventID, failure)
	}
	if info, err := os.Stat(filepath.Join(root, ".fanloop", "flow", "state.lock")); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("persistent Requirement lock is missing or invalid: info=%v error=%v", info, err)
	}
}

func acknowledgeStorePanorama(t *testing.T, local *Store, current state.State, trigger, id string) state.State {
	t.Helper()
	next := current
	next.LastEventID = id
	next.UpdatedAt = current.UpdatedAt.Add(time.Minute)
	event := state.Event{
		SchemaVersion: state.CurrentEventSchemaVersion, ID: id, OccurredAt: next.UpdatedAt,
		Kind: state.EventFlowActionAcknowledged, Command: "flow.action.acknowledge",
		Workflow: current.Release.Workflow, CausedByEventID: current.LastEventID,
		Payload: state.Payload(state.FlowActionAcknowledgedPayload{
			ActionID: "panorama-" + trigger, ActionType: state.ActionPanoramaDeliveryRequired,
			TriggerEventID: trigger, PanoramaSnapshotPath: cardsnapshot.Path(next.UpdatedAt, trigger, 0),
		}),
	}
	if failure := local.Commit(next, event); failure != nil {
		t.Fatal(failure)
	}
	return next
}

func TestTraceProjectionDescribesPanoramaActionAcknowledgment(t *testing.T) {
	loaded, err := loadRuntimeControls(t)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	event := state.Event{
		SchemaVersion: state.CurrentEventSchemaVersion, ID: "e2", OccurredAt: now,
		Kind: state.EventFlowActionAcknowledged, Command: "flow.action.acknowledge",
		Workflow: state.WorkflowRefFrom(loaded.Ref), CausedByEventID: "e1",
		Payload: state.Payload(state.FlowActionAcknowledgedPayload{
			ActionID: "panorama-e1", ActionType: state.ActionPanoramaDeliveryRequired,
			TriggerEventID: "e1", PanoramaSnapshotPath: cardsnapshot.Path(now, "e1", 0),
		}),
	}
	row := traceEventRow(loaded.Workflow, event)
	if row.event != "Panorama" || row.skill != "fanloop-panorama-delivery@CLI" || row.result != "acknowledged" || row.reason != "全景图已投递并确认" {
		t.Fatalf("row = %#v", row)
	}
	for _, want := range []string{"action=panorama-e1", "trigger_event_id=e1", "snapshot="} {
		found := false
		for _, item := range row.evidence {
			if strings.Contains(item, want) {
				found = true
			}
		}
		if !found {
			t.Fatalf("evidence missing %q: %#v", want, row.evidence)
		}
	}
}

func committedWorkflow(t *testing.T, root string) (*Store, state.State) {
	t.Helper()
	local, failure := New(root)
	if failure != nil {
		t.Fatal(failure)
	}
	loaded, err := loadRuntimeControls(t)
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

func loadRuntimeControls(t *testing.T) (workflow.Loaded, error) {
	t.Helper()
	dataRoot := t.TempDir()
	workflowsRoot := filepath.Join(dataRoot, "source", "fanloop-workflow")
	if err := os.CopyFS(filepath.Join(workflowsRoot, "linear-runtime"), os.DirFS(filepath.Join("..", "workflow", "testdata", "linear-runtime"))); err != nil {
		return workflow.Loaded{}, err
	}
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_CONFIG_SOURCE", filepath.Join(dataRoot, "source"))
	return workflow.Load("linear-runtime")
}
