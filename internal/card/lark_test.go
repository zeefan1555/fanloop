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

func TestCardShowsHumanReadableStateOutputsAndEvidence(t *testing.T) {
	loaded, err := workflow.Load("fanloop-maintainer")
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
			"requirement_document_url": {
				Type: workflow.OutputURL, Value: json.RawMessage(`"https://bytedance.larkoffice.com/docx/requirements"`), ProducerStepID: "clarify_requirements",
			},
		},
	}
	markdown := renderMarkdown(cardidl.CardView_current, current, loaded.Workflow)
	for _, want := range []string{
		"需求定义 · 需求确认",
		"[需求确认报告](https://bytedance.larkoffice.com/docx/requirements)",
		"等待 Agent 识别人工反馈并提交对应 ConditionResult",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("card Markdown is missing %q:\n%s", want, markdown)
		}
	}
	for _, internalID := range []string{"requirement_document_url", "requirements_approved", "design_technical_solution", "bootstrap_techdesign", "confirm_requirements"} {
		if strings.Contains(markdown, internalID) {
			t.Fatalf("card Markdown exposes internal ID %q:\n%s", internalID, markdown)
		}
	}
	evidence := currentEvidence(current)
	if !strings.Contains(evidence, "human：请修改方案（om_feedback）") {
		t.Fatalf("card Evidence = %q", evidence)
	}
}

func TestCardShowsOnlyCurrentExecutionEvidence(t *testing.T) {
	loaded, err := workflow.Load("technical-solution-design")
	if err != nil {
		t.Fatal(err)
	}
	stepID := "write_technical_solution"
	current := state.State{
		Requirement: state.Requirement{Title: "Loop card"}, CurrentStepID: &stepID, CurrentStepStatus: state.StepReady,
		CurrentStepSummary: "正在修复代码", CurrentEvidence: []state.Evidence{{Source: state.EvidenceSystem, Content: "go test failed", Ref: "test.log"}},
		Outputs: map[string]state.RegisteredOutput{},
	}
	markdown := renderMarkdown(cardidl.CardView_panorama, current, loaded.Workflow)
	for _, want := range []string{"方案成文 · 方案成文", "正在修复代码"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("Card is missing current fact %q:\n%s", want, markdown)
		}
	}
	for _, retired := range []string{"write_technical_solution", "最近一次 Result", "unit_tests_failed", "automated_checks_result"} {
		if strings.Contains(markdown, retired) {
			t.Fatalf("Card contains retired latest Result fact %q:\n%s", retired, markdown)
		}
	}
}

func TestPanoramaMarkdownMatchesCompactCardHierarchy(t *testing.T) {
	loaded, err := workflow.Load("fanloop-maintainer")
	if err != nil {
		t.Fatal(err)
	}
	stepID := "bootstrap_techdesign"
	current := state.State{
		Requirement:        state.Requirement{Title: "Compact card"},
		CurrentStepID:      &stepID,
		CurrentStepStatus:  state.StepReady,
		CurrentStepSummary: "working",
		Outputs:            map[string]state.RegisteredOutput{},
	}

	markdown := renderMarkdown(cardidl.CardView_panorama, current, loaded.Workflow)
	for _, want := range []string{
		"# 后端研发交付 · Compact card `Ready` `0%`",
		"需求定义 · 工作区准备",
		"## 状态全景",
		"需求定义：**工作区准备（Ready）** → 需求澄清 → 需求确认",
		"需求实现：方案设计 → 代码实现 → 本地验证 → 代码审查",
		"变更交付：Agent 自动化验收 → 人类端到端验收 → MR 交接",
		"整体进度：0%",
		"## 各阶段 Output",
		"| 需求定义 | 需求实现 | 变更交付 |",
		"> **当前执行证据**",
		"**🚧 当前进行中**",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("compact Markdown is missing %q:\n%s", want, markdown)
		}
	}
	for _, oldLayout := range []string{"`bootstrap_techdesign`", "当前 Prompt", "可上报 Condition", "正常方向", "## Workflow 全景"} {
		if strings.Contains(markdown, oldLayout) {
			t.Fatalf("compact Markdown still contains old layout %q:\n%s", oldLayout, markdown)
		}
	}
}

func TestMarkdownAndLarkUseTheSamePanoramaContent(t *testing.T) {
	loaded, err := workflow.Load("fanloop-maintainer")
	if err != nil {
		t.Fatal(err)
	}
	stepID := "implement_code"
	current := state.State{
		Requirement: state.Requirement{Title: "Shared panorama"}, CurrentStepID: &stepID,
		CurrentStepStatus: state.StepInProgress, CurrentStepSummary: "running focused tests",
		Outputs: map[string]state.RegisteredOutput{},
	}
	content := buildCardContent(cardidl.CardView_panorama, current, loaded.Workflow)
	markdown := renderMarkdownContent(content)
	lark, err := json.Marshal(renderLarkCardContent(content))
	if err != nil {
		t.Fatal(err)
	}
	for _, fact := range []string{
		content.Title, content.StageName, content.StepName, content.Status,
		content.Stages[0].Name, content.Action.Title, content.Action.Summary,
	} {
		if !strings.Contains(markdown, fact) || !strings.Contains(string(lark), fact) {
			t.Fatalf("shared Panorama fact %q missing from a renderer\nMarkdown:\n%s\nLark:\n%s", fact, markdown, lark)
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
