package trace

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/zeefan1555/commonloop/internal/idl/erroridl"
	"github.com/zeefan1555/commonloop/internal/idl/traceidl"
	"github.com/zeefan1555/commonloop/internal/larkexec"
	"github.com/zeefan1555/commonloop/internal/state"
	"github.com/zeefan1555/commonloop/internal/traceconfig"
	"github.com/zeefan1555/commonloop/internal/workflow"
)

func TestLatestFlowAuditUsesConditionResultEvent(t *testing.T) {
	events := []state.Event{{
		Kind: state.EventFlowResult,
		Payload: state.Payload(state.FlowResultPayload{
			ConditionResults: []state.ConditionResult{{
				ConditionID: "technical_solution_designed",
				Output: state.OutputValue{
					Type: workflow.OutputURL, Value: json.RawMessage(`"https://example.com/design"`),
				},
			}},
			Effect: state.ResultAdvanced,
			Transition: state.Transition{
				Direction: state.TransitionFlow, FromStepID: "design_technical_solution", ToStepID: "confirm_technical_solution",
			},
			OutputChanges: state.OutputChanges{Accepted: []string{"technical_design_document_url"}},
		}),
	}}
	audit := latestFlowAudit(events)
	for _, want := range []string{
		"report=result",
		"effect=advanced",
		"from=design_technical_solution",
		"to=confirm_technical_solution",
		"conditions=technical_solution_designed",
		"accepted=technical_design_document_url",
	} {
		if !strings.Contains(audit, want) {
			t.Fatalf("audit is missing %q: %s", want, audit)
		}
	}
}

func TestRegistryFieldsNeverProjectsMeegoAsPRD(t *testing.T) {
	tests := []struct {
		name    string
		current state.State
		wantPRD any
	}{
		{
			name: "Meego locator only",
			current: state.State{Requirement: state.Requirement{
				SourceURL: "https://meego.larkoffice.com/aweme/story/detail/7363677776",
			}},
		},
		{
			name: "direct PRD",
			current: state.State{Requirement: state.Requirement{
				SourceURL: "https://bytedance.larkoffice.com/docx/DirectPRD",
			}},
			wantPRD: "https://bytedance.larkoffice.com/docx/DirectPRD",
		},
		{
			name: "clarified PRD overrides Meego locator",
			current: state.State{
				Requirement: state.Requirement{SourceURL: "https://meego.larkoffice.com/aweme/story/detail/7363677776"},
				Outputs: map[string]state.RegisteredOutput{
					"requirement_document_url": {
						Type: workflow.OutputURL, Value: json.RawMessage(`"https://bytedance.larkoffice.com/wiki/ClarifiedPRD"`),
					},
				},
			},
			wantPRD: "https://bytedance.larkoffice.com/wiki/ClarifiedPRD",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry, ok := traceconfig.Resolve(traceconfig.RegistryProduction, "unconfigured-workflow")
			if !ok {
				t.Fatal("production default Registry is missing")
			}
			fields := registryFields(registry, test.current, workflow.Workflow{}, nil, "", "trace-key", "owner")
			if got := fields["PRD"]; got != test.wantPRD {
				t.Fatalf("Registry PRD = %#v, want %#v", got, test.wantPRD)
			}
		})
	}
}

func TestRegistryFieldsProjectsMaintainerArtifactsWithoutReusingPRD(t *testing.T) {
	current := state.State{
		Requirement: state.Requirement{Title: "Self iteration"},
		Release:     state.Release{Workflow: state.WorkflowRef{ID: "commonloop-maintainer"}},
		Integrations: state.Integrations{Trace: &state.TraceBinding{
			DocumentURL:       "https://bytedance.larkoffice.com/docx/Trace",
			Registry:          "production",
			CLILogDocumentURL: "https://bytedance.larkoffice.com/docx/CLILog",
		}},
		Outputs: map[string]state.RegisteredOutput{
			"requirement_document_url":      {Value: json.RawMessage(`"https://bytedance.larkoffice.com/docx/Requirements"`)},
			"technical_design_document_url": {Value: json.RawMessage(`"https://bytedance.larkoffice.com/docx/Design"`)},
			"merge_request_urls":            {Value: json.RawMessage(`["https://github.com/zeefan1555/commonloop/merge_requests/123"]`)},
		},
	}
	registry, ok := traceconfig.Resolve(traceconfig.RegistryProduction, current.Release.Workflow.ID)
	if !ok {
		t.Fatal("Workflow Registry override is missing")
	}
	fields := registryFields(registry, current, workflow.Workflow{}, nil, current.Integrations.Trace.DocumentURL, "trace-key", "owner")
	want := map[string]any{
		"PRD":    nil,
		"需求澄清":   "https://bytedance.larkoffice.com/docx/Requirements",
		"技术方案":   "https://bytedance.larkoffice.com/docx/Design",
		"MR":     "https://github.com/zeefan1555/commonloop/merge_requests/123",
		"CLI 日志": "https://bytedance.larkoffice.com/docx/CLILog",
	}
	for key, value := range want {
		if got := fields[key]; got != value {
			t.Fatalf("Registry field %q = %#v, want %#v; fields=%#v", key, got, value, fields)
		}
	}
	current.Requirement.SourceURL = "https://bytedance.larkoffice.com/docx/RealPRD"
	if got := registryFields(registry, current, workflow.Workflow{}, nil, current.Integrations.Trace.DocumentURL, "trace-key", "owner")["PRD"]; got != current.Requirement.SourceURL {
		t.Fatalf("maintainer real PRD = %#v, want %q", got, current.Requirement.SourceURL)
	}

	ordinary := current
	ordinary.Release.Workflow.ID = "unconfigured-workflow"
	ordinary.Integrations.Trace.CLILogDocumentURL = ""
	ordinaryRegistry, ok := traceconfig.Resolve(traceconfig.RegistryProduction, ordinary.Release.Workflow.ID)
	if !ok {
		t.Fatal("production default Registry is missing")
	}
	ordinaryFields := registryFields(ordinaryRegistry, ordinary, workflow.Workflow{}, nil, ordinary.Integrations.Trace.DocumentURL, "trace-key", "owner")
	for _, key := range []string{"需求澄清", "技术方案", "MR", "CLI 日志"} {
		if _, exists := ordinaryFields[key]; exists {
			t.Fatalf("ordinary Registry unexpectedly contains %q: %#v", key, ordinaryFields)
		}
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

	result := syncDocument(context.Background(), traceidl.TraceTarget_cli_log_document, "https://bytedance.larkoffice.com/docx/CLILog", nil)
	if result.Error == nil || result.Error.Code != erroridl.ErrorCode_TRACE_UPDATE_FAILED {
		t.Fatalf("CLI log failure = %#v, want TRACE_UPDATE_FAILED", result)
	}
}
