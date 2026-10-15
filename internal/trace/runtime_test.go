package trace

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/idl/traceidl"
	"github.com/zeefan1555/fanloop/internal/larkexec"
	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/traceconfig"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

func TestLatestFlowAuditUsesConditionResultEvent(t *testing.T) {
	events := []state.Event{{
		Kind: state.EventFlowResult,
		Payload: state.Payload(state.FlowResultPayload{
			ConditionResults: []state.ConditionResult{{
				ConditionID: "artifact_ready",
				Output: state.OutputValue{
					Type: workflow.OutputURL, Value: json.RawMessage(`"https://example.com/artifact"`),
				},
			}},
			Effect: state.ResultAdvanced,
			Transition: state.Transition{
				Direction: state.TransitionFlow, FromStepID: "draft", ToStepID: "review",
			},
			OutputChanges: state.OutputChanges{Accepted: []string{"artifact_url"}},
		}),
	}}
	audit := latestFlowAudit(events)
	for _, want := range []string{
		"report=result",
		"effect=advanced",
		"from=draft",
		"to=review",
		"conditions=artifact_ready",
		"accepted=artifact_url",
	} {
		if !strings.Contains(audit, want) {
			t.Fatalf("audit is missing %q: %s", want, audit)
		}
	}
}

func TestRegistryFieldsNeverProjectsWorkItemLocatorAsPRD(t *testing.T) {
	registry := traceconfig.Registry{Fields: syntheticRegistryFields()}
	registry.Fields.SourceDocumentOnly = true
	tests := []struct {
		name    string
		current state.State
		want    any
	}{
		{
			name: "work-item locator only",
			current: state.State{Requirement: state.Requirement{
				SourceURL: "https://work-items.example.invalid/story/detail/example-1",
			}},
		},
		{
			name: "direct PRD",
			current: state.State{Requirement: state.Requirement{
				SourceURL: "https://docs.example.invalid/docx/DirectPRD",
			}},
			want: "https://docs.example.invalid/docx/DirectPRD",
		},
		{
			name: "clarified PRD overrides work-item locator",
			current: state.State{
				Requirement: state.Requirement{SourceURL: "https://work-items.example.invalid/story/detail/example-1"},
				Outputs: map[string]state.RegisteredOutput{
					"source_document_url": {
						Type: workflow.OutputURL, Value: json.RawMessage(`"https://docs.example.invalid/wiki/ClarifiedPRD"`),
					},
				},
			},
			want: "https://docs.example.invalid/wiki/ClarifiedPRD",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fields := registryFields(registry, test.current, workflow.Workflow{}, nil, "", "trace-key", "owner")
			if got := fields["Source"]; got != test.want {
				t.Fatalf("Registry Source = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestRegistryFieldsProjectsConfiguredOutputsWithoutReusingSource(t *testing.T) {
	current := state.State{
		Requirement: state.Requirement{Title: "Custom flow"},
		Release:     state.Release{Workflow: state.WorkflowRef{ID: "custom-flow"}},
		Integrations: state.Integrations{Trace: &state.TraceBinding{
			DocumentURL:       "https://docs.example.invalid/docx/Trace",
			Registry:          traceconfig.RegistryProduction,
			CLILogDocumentURL: "https://docs.example.invalid/docx/CLILog",
		}},
		Outputs: map[string]state.RegisteredOutput{
			"artifact_links": {Value: json.RawMessage(`["https://example.com/artifacts/123"]`)},
		},
	}
	registry := traceconfig.Registry{Fields: syntheticRegistryFields()}
	registry.Fields.SourceDocumentOnly = true
	registry.Fields.CLILogURL = "CLI log"
	registry.Fields.Outputs = map[string]string{"artifact_links": "Artifact"}
	fields := registryFields(registry, current, workflow.Workflow{}, nil, current.Integrations.Trace.DocumentURL, "trace-key", "owner")
	want := map[string]any{
		"Source":   nil,
		"Artifact": "https://example.com/artifacts/123",
		"CLI log":  "https://docs.example.invalid/docx/CLILog",
	}
	for key, value := range want {
		if got := fields[key]; got != value {
			t.Fatalf("Registry field %q = %#v, want %#v; fields=%#v", key, got, value, fields)
		}
	}
	current.Requirement.SourceURL = "https://docs.example.invalid/docx/RealPRD"
	if got := registryFields(registry, current, workflow.Workflow{}, nil, current.Integrations.Trace.DocumentURL, "trace-key", "owner")["Source"]; got != current.Requirement.SourceURL {
		t.Fatalf("real source document = %#v, want %q", got, current.Requirement.SourceURL)
	}

	ordinary := current
	ordinary.Release.Workflow.ID = "unconfigured-workflow"
	ordinary.Integrations.Trace.CLILogDocumentURL = ""
	ordinaryRegistry := traceconfig.Registry{Fields: syntheticRegistryFields()}
	ordinary.Outputs = nil
	ordinaryFields := registryFields(ordinaryRegistry, ordinary, workflow.Workflow{}, nil, ordinary.Integrations.Trace.DocumentURL, "trace-key", "owner")
	for _, key := range []string{"Artifact", "CLI log"} {
		if _, exists := ordinaryFields[key]; exists {
			t.Fatalf("ordinary Registry unexpectedly contains %q: %#v", key, ordinaryFields)
		}
	}
}

func TestRegistryFieldsPreserveWorkflowDisplaySemantics(t *testing.T) {
	loaded, err := workflow.LoadFrom(filepath.Join("..", "workflow", "testdata"), "runtime-controls")
	if err != nil {
		t.Fatal(err)
	}
	first, third := "first", "third"
	tests := []struct {
		name     string
		current  state.State
		location string
		status   string
	}{
		{name: "blocked agent", current: state.State{CurrentStepID: &first, CurrentStepStatus: state.StepBlocked}, location: "Runtime / Controls / First", status: "Blocked"},
		{name: "human review", current: state.State{CurrentStepID: &third, CurrentStepStatus: state.StepReady}, location: "Runtime / Controls / Third", status: "Human Review"},
		{name: "completed", current: state.State{}, location: "Runtime / Controls / Third", status: "已完成"},
	}
	registry := traceconfig.Registry{Fields: syntheticRegistryFields()}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fields := registryFields(registry, test.current, loaded.Workflow, nil, "", "trace-key", "")
			if fields["Location"] != test.location || fields["Status"] != test.status {
				t.Fatalf("registry display = location %#v status %#v, want %q / %q", fields["Location"], fields["Status"], test.location, test.status)
			}
		})
	}
}

func TestPlannedSyncTargetsIncludeConfiguredCLILog(t *testing.T) {
	current := state.State{Integrations: state.Integrations{Trace: &state.TraceBinding{
		DocumentURL:       "https://example.com/trace",
		Registry:          traceconfig.RegistryProduction,
		CLILogDocumentURL: "https://example.com/cli-log",
	}}}
	targets := plannedSyncTargets(current)
	if len(targets) != 3 || targets[0].Target != traceidl.TraceTarget_trace_document ||
		targets[1].Target != traceidl.TraceTarget_cli_log_document || targets[2].Target != traceidl.TraceTarget_registry {
		t.Fatalf("planned targets = %#v", targets)
	}
}

func syntheticRegistryFields() traceconfig.RegistryFields {
	return traceconfig.RegistryFields{
		TraceKey: "Trace key", Title: "Title", Owner: "Owner", Location: "Location", Status: "Status",
		LoopCount: "Loop count", SourceURL: "Source", SourceOutput: "source_document_url",
		TraceURL: "Trace", UpdatedAt: "Updated at", Origin: "Origin",
	}
}

func TestRenderCLILogDocumentPreservesBytesAndExtendsFence(t *testing.T) {
	raw := []byte("{\"stdout\":\"``` nested\"}\nlast line")
	document := renderCLILogDocument(raw)
	if !bytes.Contains(document, raw) {
		t.Fatalf("rendered CLI log changed raw bytes:\n%s", document)
	}
	if !bytes.Contains(document, []byte("````jsonl\n")) || !bytes.HasSuffix(document, []byte("\n````\n")) {
		t.Fatalf("rendered CLI log did not use a safe Markdown fence:\n%s", document)
	}
}

func TestCLILogDocumentExecutionFailureUsesTraceUpdateError(t *testing.T) {
	original := larkexec.Execute
	larkexec.Execute = func(context.Context, []string, io.Reader, time.Duration) (larkexec.Result, error) {
		return larkexec.Result{}, errors.New("lark-cli unavailable")
	}
	t.Cleanup(func() { larkexec.Execute = original })

	result := syncDocument(context.Background(), traceidl.TraceTarget_cli_log_document, "https://docs.example.invalid/docx/CLILog", nil)
	if result.Error == nil || result.Error.Code != erroridl.ErrorCode_TRACE_UPDATE_FAILED {
		t.Fatalf("CLI log failure = %#v, want TRACE_UPDATE_FAILED", result)
	}
}
