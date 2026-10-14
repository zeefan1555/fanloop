package card

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/zeefan1555/fanloop/internal/idl/cardidl"
	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/workflow"
	"github.com/zeefan1555/fanloop/internal/workflowview"
)

type larkCard struct {
	Schema string     `json:"schema"`
	Config larkConfig `json:"config"`
	Header larkHeader `json:"header"`
	Body   larkBody   `json:"body"`
}

type larkConfig struct {
	UpdateMulti bool `json:"update_multi"`
}

type larkHeader struct {
	Template    string        `json:"template"`
	Title       larkPlainText `json:"title"`
	Subtitle    larkPlainText `json:"subtitle"`
	TextTagList []larkTextTag `json:"text_tag_list"`
}

type larkBody struct {
	Elements []any `json:"elements"`
}

type larkPlainText struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

type larkTextTag struct {
	Tag  string        `json:"tag"`
	Text larkPlainText `json:"text"`
}

type larkMarkdown struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

type larkColumn struct {
	Tag             string         `json:"tag"`
	Width           string         `json:"width"`
	Weight          int            `json:"weight"`
	VerticalAlign   string         `json:"vertical_align"`
	BackgroundStyle string         `json:"background_style"`
	Elements        []larkMarkdown `json:"elements"`
}

type larkColumnSet struct {
	Tag     string       `json:"tag"`
	Columns []larkColumn `json:"columns"`
}

type cardAction struct {
	Theme   string
	Title   string
	Summary string
	Prompt  string
}

type cardStage struct {
	Name   string
	Output string
}

type cardContent struct {
	Title     string
	StageName string
	StepName  string
	Status    string
	Progress  int
	Panorama  string
	Stages    []cardStage
	Evidence  string
	Action    cardAction
}

func buildCardContent(view cardidl.CardView, current state.State, loaded workflow.Loaded) cardContent {
	display := workflowview.Display(loaded.Workflow, current)
	stageName, stepName := cardLocation(display)
	stages := make([]cardStage, 0, len(loaded.Workflow.Stages))
	for _, stage := range loaded.Workflow.Stages {
		stages = append(stages, cardStage{Name: stage.Name, Output: stageOutputContent(current, loaded.Workflow, stage)})
	}
	return cardContent{
		Title: current.Requirement.Title, StageName: stageName, StepName: stepName,
		Status: cardStatus(display), Progress: display.ProgressPercent,
		Panorama: statusPanorama(view, display), Stages: stages,
		Evidence: currentEvidence(current), Action: buildCardAction(current, loaded, display),
	}
}

func renderLarkCard(view cardidl.CardView, current state.State, loaded workflow.Loaded) larkCard {
	return renderLarkCardContent(buildCardContent(view, current, loaded))
}

func renderLarkCardContent(content cardContent) larkCard {
	return larkCard{
		Schema: "2.0",
		Config: larkConfig{UpdateMulti: true},
		Header: larkHeader{
			Template: "default",
			Title:    plainText(content.Title),
			Subtitle: plainText(content.StageName + " · " + content.StepName),
			TextTagList: []larkTextTag{
				{Tag: "text_tag", Text: plainText(content.Status)},
				{Tag: "text_tag", Text: plainText(fmt.Sprintf("%d%%", content.Progress))},
			},
		},
		Body: larkBody{Elements: []any{
			larkColumnSet{Tag: "column_set", Columns: []larkColumn{newLarkColumn("grey", "**状态全景**\n"+content.Panorama)}},
			larkMarkdown{Tag: "markdown", Content: "**各阶段 Output**"},
			larkColumnSet{Tag: "column_set", Columns: outputColumns(content.Stages)},
			larkMarkdown{Tag: "markdown", Content: content.Evidence},
			larkColumnSet{Tag: "column_set", Columns: []larkColumn{newLarkColumn(content.Action.Theme, actionMarkdown(content.Action))}},
		}},
	}
}

func plainText(content string) larkPlainText {
	return larkPlainText{Tag: "plain_text", Content: content}
}

func newLarkColumn(background, content string) larkColumn {
	return larkColumn{
		Tag: "column", Width: "weighted", Weight: 1, VerticalAlign: "top", BackgroundStyle: background,
		Elements: []larkMarkdown{{Tag: "markdown", Content: content}},
	}
}

func cardLocation(display workflowview.DisplaySnapshot) (string, string) {
	if display.Status == workflowview.DisplayDone {
		return "Done", "流程已完成"
	}
	if !display.Location.Found {
		return "Unknown", display.Location.StepID
	}
	return display.Location.StageName, display.Location.StepName
}

func cardStatus(display workflowview.DisplaySnapshot) string {
	if display.RequiresHumanReview && display.Status != workflowview.DisplayAwaitingConfirmation {
		return "Human Review"
	}
	switch display.Status {
	case workflowview.DisplayDone:
		return "Done"
	case workflowview.DisplayAwaitingConfirmation:
		return "Awaiting Confirmation"
	case workflowview.DisplayBlocked:
		return "Blocked"
	case workflowview.DisplayFixing:
		return "Fixing"
	case workflowview.DisplayInProgress:
		return "In Progress"
	default:
		return "Ready"
	}
}

func statusPanorama(view cardidl.CardView, display workflowview.DisplaySnapshot) string {
	lines := make([]string, 0, len(display.Stages)+1)
	for _, stage := range display.Stages {
		if view == cardidl.CardView_current && display.Location.Found && display.Status != workflowview.DisplayDone && !displayStageContainsCurrent(stage) {
			continue
		}
		lines = append(lines, workflowview.FormatPanoramaStage(stage, func(step workflowview.DisplayStep) string {
			label := step.Name
			switch step.State {
			case workflowview.DisplayStepSkipped:
				label = "已跳过 " + label
			case workflowview.DisplayStepCompleted:
				label = "✅ " + label
			case workflowview.DisplayStepCurrent:
				label = "**" + label + "（" + cardStatus(display) + "）**"
			}
			return label
		}))
	}
	if len(lines) == 0 {
		return "流程已完成"
	}
	lines = append(lines, fmt.Sprintf("整体进度：%d%%", display.ProgressPercent))
	return strings.Join(lines, "\n")
}

func displayStageContainsCurrent(stage workflowview.DisplayStage) bool {
	for _, job := range stage.Jobs {
		for _, step := range job.Steps {
			if step.State == workflowview.DisplayStepCurrent {
				return true
			}
		}
	}
	return false
}

func traceLink(current state.State) string {
	if current.Integrations.Trace != nil && current.Integrations.Trace.DocumentURL != "" {
		return "📄 Trace 文档：[查看完整 Trace](" + current.Integrations.Trace.DocumentURL + ")"
	}
	return "📄 Trace 文档：未绑定"
}

func cliLogLink(current state.State) string {
	if current.Integrations.Trace != nil && current.Integrations.Trace.CLILogDocumentURL != "" {
		return "📜 CLI 日志：[查看完整输入输出](" + current.Integrations.Trace.CLILogDocumentURL + ")"
	}
	return ""
}

func outputColumns(stages []cardStage) []larkColumn {
	columns := make([]larkColumn, 0, len(stages))
	for _, stage := range stages {
		columns = append(columns, newLarkColumn("grey", "**"+stage.Name+"**\n"+stage.Output))
	}
	return columns
}

func stageOutputContent(current state.State, definition workflow.Workflow, stage workflow.Stage) string {
	urlOutputs := map[string]workflow.OutputDefinition{}
	for _, job := range stage.Jobs {
		for _, step := range job.Steps {
			for _, conditionID := range definition.RelevantConditionIDs(step.ID) {
				condition, ok := definition.Condition(conditionID)
				if ok && condition.Output.Source == "" && isURLOutput(condition.Output.Type) {
					if _, exists := urlOutputs[condition.Output.Key]; !exists {
						urlOutputs[condition.Output.Key] = condition.Output
					}
				}
			}
		}
	}
	keys := make([]string, 0, len(urlOutputs))
	for key := range urlOutputs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	outputs := make([]string, 0, len(urlOutputs))
	for _, key := range keys {
		registered := current.Outputs[key]
		outputs = append(outputs, renderOutput(urlOutputs[key], registered.Value))
	}
	if len(outputs) == 0 {
		return "暂未生成"
	}
	return strings.Join(outputs, "\n")
}

func isURLOutput(outputType workflow.OutputType) bool {
	switch outputType {
	case workflow.OutputURL, workflow.OutputURLList:
		return true
	default:
		return false
	}
}

func renderOutput(definition workflow.OutputDefinition, raw json.RawMessage) string {
	label := strings.TrimSpace(definition.Description)
	if label == "" {
		label = definition.Key
	}
	if len(raw) == 0 {
		return label + "：暂未生成"
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return label + "：数据无效"
	}
	switch typed := value.(type) {
	case string:
		if isHTTPURL(typed) {
			return "[" + label + "](" + typed + ")"
		}
		return label + "：`" + typed + "`"
	case []any:
		links := make([]string, 0, len(typed))
		for index, item := range typed {
			value, ok := item.(string)
			if !ok {
				continue
			}
			itemLabel := fmt.Sprintf("%s %d", label, index+1)
			if isHTTPURL(value) {
				links = append(links, "["+itemLabel+"]("+value+")")
			}
		}
		if len(links) > 0 {
			return strings.Join(links, " · ")
		}
	}
	compact, err := json.Marshal(value)
	if err != nil {
		return label + "：数据无效"
	}
	return label + "：`" + string(compact) + "`"
}

func isHTTPURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func currentEvidence(current state.State) string {
	lines := []string{"> **当前执行证据**", "> • " + traceLink(current)}
	if link := cliLogLink(current); link != "" {
		lines = append(lines, "> • "+link)
	}
	if summary := strings.TrimSpace(current.CurrentStepSummary); summary != "" {
		lines = append(lines, "> • "+summary)
	}
	for _, item := range current.CurrentEvidence {
		line := "> • " + string(item.Source) + "：" + item.Content
		if item.Ref != "" {
			line += "（" + item.Ref + "）"
		}
		lines = append(lines, line)
	}
	if len(lines) == 2 && current.Integrations.Trace == nil {
		lines = append(lines, "> • 本轮尚无可核验证据。")
	}
	return strings.Join(lines, "\n")
}

func buildCardAction(current state.State, loaded workflow.Loaded, display workflowview.DisplaySnapshot) cardAction {
	if display.Status == workflowview.DisplayDone {
		return cardAction{Theme: "green-50", Title: "✅ 流程已完成", Summary: "全部 Workflow Step 已完成。"}
	}
	if !display.Location.Found {
		return cardAction{Theme: "red-50", Title: "⛔ 当前阻塞", Summary: "当前 Step 不在绑定的 Workflow 中。"}
	}
	summary := strings.TrimSpace(current.CurrentStepSummary)
	projected := workflowview.ProjectLoaded(loaded, current).Current
	if summary == "" && projected != nil && projected.Prompt != nil {
		summary = strings.TrimSpace(projected.Prompt.Content)
	}
	switch display.Status {
	case workflowview.DisplayBlocked:
		return cardAction{Theme: "red-50", Title: "⛔ 当前阻塞", Summary: summary, Prompt: "请补充所缺的关键信息。"}
	case workflowview.DisplayFixing:
		return cardAction{Theme: "orange-50", Title: "🔄 正在修复", Summary: summary}
	}
	if display.Location.Executor == workflow.StepExecutorHuman {
		prompt := "如认可，请回复：`批准继续`"
		if routes := loaded.Workflow.Flows[display.Location.StepID]; len(routes) > 0 {
			flow := routes[0]
			if flow.Terminal {
				prompt = "如认可，请回复：`批准完成`"
			} else if nextContext, _, nextFound := loaded.Workflow.FindStep(flow.NextStepID); nextFound && nextContext.Stage.ID != display.Location.StageID {
				prompt = "如认可，请回复：`批准进入 " + nextContext.Stage.Name + "`"
			}
		}
		return cardAction{Theme: "yellow-50", Title: "⚠️ 需要你确认", Summary: "等待 Agent 识别人工反馈并提交对应 ConditionResult。", Prompt: prompt}
	}
	return cardAction{Theme: "grey", Title: "🚧 当前进行中", Summary: summary}
}

func actionMarkdown(action cardAction) string {
	lines := []string{"**" + action.Title + "**"}
	if action.Summary != "" {
		lines = append(lines, action.Summary)
	}
	if action.Prompt != "" {
		lines = append(lines, action.Prompt)
	}
	return strings.Join(lines, "\n")
}
