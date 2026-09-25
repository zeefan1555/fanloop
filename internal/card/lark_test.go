package card

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/idl/cardidl"
	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/traceconfig"
	"github.com/zeefan1555/fanloop/internal/workflow"
	"github.com/zeefan1555/fanloop/internal/workflowview"
)

func TestCardShowsHumanReadableStateOutputsAndEvidence(t *testing.T) {
	loaded, err := loadLinearRuntime()
	if err != nil {
		t.Fatal(err)
	}
	firstCompleted := loaded.Workflow.Conditions["first_completed"]
	firstCompleted.Output = workflow.OutputDefinition{Key: "artifact_url", Type: workflow.OutputURL, Description: "Artifact"}
	loaded.Workflow.Conditions["first_completed"] = firstCompleted
	stepID := "second"
	current := state.State{
		Requirement:        state.Requirement{Title: "Flow card"},
		CurrentStepID:      &stepID,
		CurrentStepStatus:  state.StepReady,
		CurrentStepSummary: "waiting for approval",
		CurrentEvidence: []state.Evidence{{
			Source: state.EvidenceHuman, Content: "请修改方案", Ref: "om_feedback",
		}},
		Outputs: map[string]state.RegisteredOutput{
			"artifact_url": {
				Type: workflow.OutputURL, Value: json.RawMessage(`"https://example.com/artifact/123"`), ProducerStepID: "first",
			},
		},
	}
	markdown := renderMarkdown(cardidl.CardView_current, current, loaded)
	for _, want := range []string{
		"Runtime · Second",
		"[Artifact](https://example.com/artifact/123)",
		"waiting for approval",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("card Markdown is missing %q:\n%s", want, markdown)
		}
	}
	for _, internalID := range []string{"artifact_url", "first_completed", "second", "third"} {
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
	loaded, err := loadLinearRuntime()
	if err != nil {
		t.Fatal(err)
	}
	stepID := "second"
	current := state.State{
		Requirement: state.Requirement{Title: "Loop card"}, CurrentStepID: &stepID, CurrentStepStatus: state.StepReady,
		CurrentStepSummary: "正在修复代码", CurrentEvidence: []state.Evidence{{Source: state.EvidenceSystem, Content: "go test failed", Ref: "test.log"}},
		Outputs: map[string]state.RegisteredOutput{},
	}
	markdown := renderMarkdown(cardidl.CardView_panorama, current, loaded)
	for _, want := range []string{"Runtime · Second", "正在修复代码"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("Card is missing current fact %q:\n%s", want, markdown)
		}
	}
	for _, retired := range []string{"second", "最近一次 Result", "first_completed", "second_completed"} {
		if strings.Contains(markdown, retired) {
			t.Fatalf("Card contains retired latest Result fact %q:\n%s", retired, markdown)
		}
	}
}

func TestPanoramaRendersSkippedStepsWithoutCompletionMark(t *testing.T) {
	loaded, err := loadLinearRuntime()
	if err != nil {
		t.Fatal(err)
	}
	stepID := "second"
	current := state.State{Requirement: state.Requirement{Title: "Skipped"}, CurrentStepID: &stepID, CurrentStepStatus: state.StepReady, SkippedStepIDs: []string{"first"}, Outputs: map[string]state.RegisteredOutput{}}
	markdown := renderMarkdown(cardidl.CardView_panorama, current, loaded)
	if !strings.Contains(markdown, "已跳过 First") || strings.Contains(markdown, "✅ First") {
		t.Fatalf("skipped Step was rendered as completed:\n%s", markdown)
	}
}

func TestPanoramaMarkdownMatchesCompactCardHierarchy(t *testing.T) {
	loaded, err := loadLinearRuntime()
	if err != nil {
		t.Fatal(err)
	}
	stepID := "first"
	current := state.State{
		Requirement:        state.Requirement{Title: "Compact card"},
		CurrentStepID:      &stepID,
		CurrentStepStatus:  state.StepReady,
		CurrentStepSummary: "working",
		Outputs:            map[string]state.RegisteredOutput{},
	}

	markdown := renderMarkdown(cardidl.CardView_panorama, current, loaded)
	for _, want := range []string{
		"# 后端研发交付 · Compact card `Ready` `0%`",
		"Runtime · First",
		"## 状态全景",
		"Runtime：**First（Ready）** → Second → Third",
		"整体进度：0%",
		"## 各阶段 Output",
		"| Runtime |",
		"> **当前执行证据**",
		"**🚧 当前进行中**",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("compact Markdown is missing %q:\n%s", want, markdown)
		}
	}
	for _, oldLayout := range []string{"`first`", "当前 Prompt", "可上报 Condition", "正常方向", "## Workflow 全景"} {
		if strings.Contains(markdown, oldLayout) {
			t.Fatalf("compact Markdown still contains old layout %q:\n%s", oldLayout, markdown)
		}
	}
}

func loadLinearRuntime() (workflow.Loaded, error) {
	return workflow.LoadFrom(filepath.Join("..", "workflow", "testdata"), "linear-runtime")
}

func TestMarkdownAndLarkUseTheSamePanoramaContent(t *testing.T) {
	loaded, err := loadLinearRuntime()
	if err != nil {
		t.Fatal(err)
	}
	stepID := "second"
	current := state.State{
		Requirement: state.Requirement{Title: "Shared panorama"}, CurrentStepID: &stepID,
		CurrentStepStatus: state.StepInProgress, CurrentStepSummary: "running focused tests",
		Outputs: map[string]state.RegisteredOutput{},
	}
	content := buildCardContent(cardidl.CardView_panorama, current, loaded)
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

func TestFixingCardUsesFixedLoopEmojiInBothFormats(t *testing.T) {
	loaded, err := loadLinearRuntime()
	if err != nil {
		t.Fatal(err)
	}
	stepID := "second"
	current := state.State{
		Requirement: state.Requirement{Title: "Rework card"}, CurrentStepID: &stepID,
		CurrentStepStatus: state.StepFixing, CurrentStepSummary: "fixing review feedback",
		Outputs: map[string]state.RegisteredOutput{},
	}
	content := buildCardContent(cardidl.CardView_panorama, current, loaded)
	if content.Action.Title != "🔄 正在修复" {
		t.Fatalf("fixing action title = %q, want %q", content.Action.Title, "🔄 正在修复")
	}
	markdown := renderMarkdownContent(content)
	lark, err := json.Marshal(renderLarkCardContent(content))
	if err != nil {
		t.Fatal(err)
	}
	for format, rendered := range map[string]string{"markdown": markdown, "lark-json": string(lark)} {
		if !strings.Contains(rendered, "🔄 正在修复") || strings.Contains(rendered, "🔁 正在修复") {
			t.Fatalf("%s fixing action does not use the fixed loop emoji:\n%s", format, rendered)
		}
	}
}

func TestCardActionIconsRemainStableOutsideFixing(t *testing.T) {
	loaded, err := loadLinearRuntime()
	if err != nil {
		t.Fatal(err)
	}
	agentStep, humanStep := "second", "third"
	for _, test := range []struct {
		name    string
		current state.State
		want    string
	}{
		{name: "ready", current: state.State{CurrentStepID: &agentStep, CurrentStepStatus: state.StepReady, Outputs: map[string]state.RegisteredOutput{}}, want: "🚧 当前进行中"},
		{name: "in progress", current: state.State{CurrentStepID: &agentStep, CurrentStepStatus: state.StepInProgress, Outputs: map[string]state.RegisteredOutput{}}, want: "🚧 当前进行中"},
		{name: "blocked", current: state.State{CurrentStepID: &agentStep, CurrentStepStatus: state.StepBlocked, Outputs: map[string]state.RegisteredOutput{}}, want: "⛔ 当前阻塞"},
		{name: "human review", current: state.State{CurrentStepID: &humanStep, CurrentStepStatus: state.StepReady, Outputs: map[string]state.RegisteredOutput{}}, want: "⚠️ 需要你确认"},
		{name: "completed", current: state.State{Outputs: map[string]state.RegisteredOutput{}}, want: "✅ 流程已完成"},
	} {
		t.Run(test.name, func(t *testing.T) {
			display := workflowview.Display(loaded.Workflow, test.current)
			if got := buildCardAction(test.current, loaded, display).Title; got != test.want {
				t.Fatalf("action title = %q, want %q", got, test.want)
			}
		})
	}
}

func TestCardMarkdownShowsCLILogBesideTrace(t *testing.T) {
	loaded, err := loadLinearRuntime()
	if err != nil {
		t.Fatal(err)
	}
	stepID := "second"
	current := state.State{
		Requirement: state.Requirement{Title: "Maintainer card"}, CurrentStepID: &stepID, CurrentStepStatus: state.StepReady,
		Outputs: map[string]state.RegisteredOutput{}, Integrations: state.Integrations{Trace: &state.TraceBinding{
			DocumentURL: "https://example.invalid/redacted", Registry: traceconfig.RegistryProduction,
			CLILogDocumentURL: "https://example.invalid/redacted",
		}},
	}
	markdown := renderMarkdown(cardidl.CardView_current, current, loaded)
	traceIndex, logIndex := strings.Index(markdown, "📄 Trace 文档"), strings.Index(markdown, "📜 CLI 日志")
	if traceIndex < 0 || logIndex <= traceIndex {
		t.Fatalf("Card Markdown links are missing or out of order:\n%s", markdown)
	}
}
