package card

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/idl/cardidl"
	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/traceconfig"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

func TestCardShowsConditionRoutesOutputsAndEvidence(t *testing.T) {
	loaded, err := workflow.Load("promotion-design")
	if err != nil {
		t.Fatal(err)
	}
	stepID := "confirm_requirements"
	current := state.State{
		Requirement:        state.Requirement{Title: "Flow card"},
		CurrentStepID:      &stepID,
		CurrentStepStatus:  state.StepReady,
		CurrentStepSummary: "waiting for approval",
		CurrentEvidence: []state.Evidence{{
			Source: state.EvidenceHuman, Content: "请修改方案", Ref: "om_feedback",
		}},
		Outputs: map[string]state.RegisteredOutput{
			"require_points_path": {
				Type: workflow.OutputPath, Value: json.RawMessage(`"require_points.md"`), ProducerStepID: "clarify_requirements",
			},
		},
	}
	markdown := renderMarkdown(cardidl.CardView_current, current, loaded.Workflow)
	for _, want := range []string{
		"requirements_approved",
		"write_promotion_design",
		"clarify_requirements",
		"confirm_requirements",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("card Markdown is missing %q:\n%s", want, markdown)
		}
	}
	evidence := currentEvidence(current)
	if !strings.Contains(evidence, "human：请修改方案（om_feedback）") {
		t.Fatalf("card Evidence = %q", evidence)
	}
}

func TestCardShowsOnlyCurrentExecutionEvidence(t *testing.T) {
	loaded, err := workflow.Load("promotion-design")
	if err != nil {
		t.Fatal(err)
	}
	stepID := "write_promotion_design"
	current := state.State{
		Requirement: state.Requirement{Title: "Loop card"}, CurrentStepID: &stepID, CurrentStepStatus: state.StepReady,
		CurrentStepSummary: "正在修复代码", CurrentEvidence: []state.Evidence{{Source: state.EvidenceSystem, Content: "go test failed", Ref: "test.log"}},
		Outputs: map[string]state.RegisteredOutput{},
	}
	markdown := renderMarkdown(cardidl.CardView_panorama, current, loaded.Workflow)
	for _, want := range []string{"write_promotion_design", "正在修复代码"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("Card is missing current fact %q:\n%s", want, markdown)
		}
	}
	for _, retired := range []string{"最近一次 Result", "unit_tests_failed", "automated_checks_result"} {
		if strings.Contains(markdown, retired) {
			t.Fatalf("Card contains retired latest Result fact %q:\n%s", retired, markdown)
		}
	}
}

func TestCardMarkdownShowsCLILogBesideTrace(t *testing.T) {
	loaded, err := workflow.Load("fanloop-maintainer")
	if err != nil {
		t.Fatal(err)
	}
	stepID := "implement_code"
	current := state.State{
		Requirement: state.Requirement{Title: "Maintainer card"}, CurrentStepID: &stepID, CurrentStepStatus: state.StepReady,
		Outputs: map[string]state.RegisteredOutput{}, Integrations: state.Integrations{Trace: &state.TraceBinding{
			DocumentURL: "https://bytedance.larkoffice.com/docx/Trace", Registry: traceconfig.RegistryProduction,
			CLILogDocumentURL: "https://bytedance.larkoffice.com/docx/CLILog",
		}},
	}
	markdown := renderMarkdown(cardidl.CardView_current, current, loaded.Workflow)
	traceIndex, logIndex := strings.Index(markdown, "📄 Trace 文档"), strings.Index(markdown, "📜 CLI 日志")
	if traceIndex < 0 || logIndex <= traceIndex {
		t.Fatalf("Card Markdown links are missing or out of order:\n%s", markdown)
	}
}
