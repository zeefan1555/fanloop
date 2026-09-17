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
	stepID := "certify_candidate"
	current := state.State{
		Requirement:        state.Requirement{Title: "Flow card"},
		CurrentStepID:      &stepID,
		CurrentStepStatus:  state.StepReady,
		CurrentStepSummary: "waiting for approval",
		CurrentEvidence: []state.Evidence{{
			Source: state.EvidenceHuman, Content: "请修改方案", Ref: "om_feedback",
		}},
		Outputs: map[string]state.RegisteredOutput{
			"merge_request_urls": {
				Type: workflow.OutputURLList, Value: json.RawMessage(`["https://github.com/zeefan1555/fanloop/pull/123"]`), ProducerStepID: "merge_and_update_local",
			},
		},
	}
	markdown := renderMarkdown(cardidl.CardView_current, current, loaded.Workflow)
	for _, want := range []string{
		"Certify · 独立候选认证",
		"[PR 1](https://github.com/zeefan1555/fanloop/pull/123)",
		"waiting for approval",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("card Markdown is missing %q:\n%s", want, markdown)
		}
	}
	for _, internalID := range []string{"merge_request_urls", "main_agent_acceptance_passed", "certify_candidate", "merge_and_update_local"} {
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
	for _, want := range []string{"结果与规划 · 文档组装", "正在修复代码"} {
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

func TestPanoramaRendersSkippedStepsWithoutCompletionMark(t *testing.T) {
	loaded, err := workflow.Load("technical-solution-design")
	if err != nil {
		t.Fatal(err)
	}
	stepID := "research_solution_options"
	current := state.State{Requirement: state.Requirement{Title: "Skipped"}, CurrentStepID: &stepID, CurrentStepStatus: state.StepAwaitingConfirmation, SkippedStepIDs: []string{"frame_requirement_background"}, Outputs: map[string]state.RegisteredOutput{}}
	markdown := renderMarkdown(cardidl.CardView_panorama, current, loaded.Workflow)
	if !strings.Contains(markdown, "已跳过 业务背景") || strings.Contains(markdown, "✅ 业务背景") {
		t.Fatalf("skipped Step was rendered as completed:\n%s", markdown)
	}
}

func TestPanoramaMarkdownMatchesCompactCardHierarchy(t *testing.T) {
	loaded, err := workflow.Load("fanloop-maintainer")
	if err != nil {
		t.Fatal(err)
	}
	stepID := "define_verification_contract"
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
		"Define · 目标与验收契约",
		"## 状态全景",
		"Define：**目标与验收契约（Ready）**",
		"Build：实现与自主验证",
		"Certify：独立候选认证",
		"Deliver：PR 合码与本地更新",
		"整体进度：0%",
		"## 各阶段 Output",
		"| Define | Build | Certify | Deliver |",
		"> **当前执行证据**",
		"**🚧 当前进行中**",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("compact Markdown is missing %q:\n%s", want, markdown)
		}
	}
	for _, oldLayout := range []string{"`define_verification_contract`", "当前 Prompt", "可上报 Condition", "正常方向", "## Workflow 全景"} {
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
	stepID := "build_until_verified"
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
	stepID := "build_until_verified"
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
