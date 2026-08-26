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

func renderLarkCard(view cardidl.CardView, current state.State, definition workflow.Workflow) larkCard {
	stageName, stepName := currentLocation(current, definition)
	action := buildCardAction(current, definition)
	return larkCard{
		Schema: "2.0",
		Config: larkConfig{UpdateMulti: true},
		Header: larkHeader{
			Template: "default",
			Title:    plainText("后端研发交付 · " + current.Requirement.Title),
			Subtitle: plainText(stageName + " · " + stepName),
			TextTagList: []larkTextTag{
				{Tag: "text_tag", Text: plainText(cardStatus(current, definition))},
				{Tag: "text_tag", Text: plainText(fmt.Sprintf("%d%%", progressPercent(current, definition)))},
			},
		},
		Body: larkBody{Elements: []any{
			larkColumnSet{Tag: "column_set", Columns: []larkColumn{newLarkColumn("grey", "**状态全景**\n"+statusPanorama(view, current, definition))}},
			larkMarkdown{Tag: "markdown", Content: "**各阶段 Output**"},
			larkColumnSet{Tag: "column_set", Columns: outputColumns(current, definition)},
			larkMarkdown{Tag: "markdown", Content: currentEvidence(current)},
			larkColumnSet{Tag: "column_set", Columns: []larkColumn{newLarkColumn(action.Theme, actionMarkdown(action))}},
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

func currentLocation(current state.State, definition workflow.Workflow) (string, string) {
	if current.CurrentStepID == nil {
		return "Done", "流程已完成"
	}
	context, _, ok := definition.FindStep(*current.CurrentStepID)
	if !ok {
		return "Unknown", *current.CurrentStepID
	}
	return context.Stage.Name, context.Step.Name
}

func cardStatus(current state.State, definition workflow.Workflow) string {
	if current.CurrentStepID == nil {
		return "Done"
	}
	if context, _, ok := definition.FindStep(*current.CurrentStepID); ok && context.Step.Executor == workflow.StepExecutorHuman {
		return "Human Review"
	}
	switch current.CurrentStepStatus {
	case state.StepBlocked:
		return "Blocked"
	case state.StepFixing:
		return "Fixing"
	case state.StepInProgress:
		return "In Progress"
	default:
		return "Ready"
	}
}

func progressPercent(current state.State, definition workflow.Workflow) int {
	steps := definition.OrderedStepIDs()
	if current.CurrentStepID == nil {
		return 100
	}
	_, position, ok := definition.FindStep(*current.CurrentStepID)
	if !ok || len(steps) == 0 {
		return 0
	}
	return (position*100 + len(steps)/2) / len(steps)
}

func statusPanorama(view cardidl.CardView, current state.State, definition workflow.Workflow) string {
	_, currentPosition, found := currentStepPosition(current, definition)
	lines := make([]string, 0, len(definition.Stages)+1)
	position := 0
	for _, stage := range definition.Stages {
		if view == cardidl.CardView_current && found {
			containsCurrent := false
			for _, job := range stage.Jobs {
				for _, step := range job.Steps {
					if current.CurrentStepID != nil && step.ID == *current.CurrentStepID {
						containsCurrent = true
						break
					}
				}
			}
			if !containsCurrent {
				position += stageStepCount(stage)
				continue
			}
		}
		steps := make([]string, 0, stageStepCount(stage))
		for _, job := range stage.Jobs {
			for _, step := range job.Steps {
				label := step.Name
				switch {
				case !found || position < currentPosition:
					label = "✅ " + label
				case position == currentPosition:
					label = "**" + label + "（" + cardStatus(current, definition) + "）**"
				}
				steps = append(steps, label)
				position++
			}
		}
		lines = append(lines, stage.Name+"："+strings.Join(steps, " → "))
	}
	if len(lines) == 0 {
		return "流程已完成"
	}
	lines = append(lines, fmt.Sprintf("整体进度：%d%%", progressPercent(current, definition)))
	return strings.Join(lines, "\n")
}

func stageStepCount(stage workflow.Stage) int {
	count := 0
	for _, job := range stage.Jobs {
		count += len(job.Steps)
	}
	return count
}

func currentStepPosition(current state.State, definition workflow.Workflow) (workflow.StepContext, int, bool) {
	if current.CurrentStepID == nil {
		return workflow.StepContext{}, len(definition.OrderedStepIDs()), false
	}
	return definition.FindStep(*current.CurrentStepID)
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

func outputColumns(current state.State, definition workflow.Workflow) []larkColumn {
	columns := make([]larkColumn, 0, len(definition.Stages))
	for _, stage := range definition.Stages {
		urlOutputs := make([]string, 0)
		for _, job := range stage.Jobs {
			for _, step := range job.Steps {
				for _, conditionID := range definition.RelevantConditionIDs(step.ID) {
					condition, ok := definition.Condition(conditionID)
					if ok && condition.Output.Source == "" && isURLOutput(condition.Output.Type) {
						urlOutputs = append(urlOutputs, condition.Output.Key)
					}
				}
			}
		}
		sort.Strings(urlOutputs)
		urlOutputs = uniqueStrings(urlOutputs)
		outputs := make([]string, 0, len(urlOutputs))
		for _, output := range urlOutputs {
			registered := current.Outputs[output]
			outputs = append(outputs, renderOutput(output, registered.Value))
		}
		if len(outputs) == 0 {
			outputs = append(outputs, "暂未生成")
		}
		columns = append(columns, newLarkColumn("grey", "**"+stage.Name+"**\n"+strings.Join(outputs, "\n")))
	}
	return columns
}

func isURLOutput(outputType workflow.OutputType) bool {
	switch outputType {
	case workflow.OutputURL, workflow.OutputURLList:
		return true
	default:
		return false
	}
}

func uniqueStrings(values []string) []string {
	result := values[:0]
	for index, value := range values {
		if index == 0 || value != values[index-1] {
			result = append(result, value)
		}
	}
	return result
}

var outputLabels = map[string]string{
	"requirement_document_url":      "需求澄清文档",
	"technical_design_document_url": "技术方案文档",
	"merge_request_urls":            "MR",
	"code_review_document_url":      "代码评审文档",
	"test_case_document_url":        "测试用例文档",
}

func renderOutput(name string, raw json.RawMessage) string {
	label := outputLabels[name]
	if label == "" {
		label = name
	}
	if len(raw) == 0 {
		switch name {
		case "requirement_document_url":
			return "暂未生成需求澄清文档"
		case "technical_design_document_url":
			return "暂未生成技术方案文档"
		case "merge_request_urls":
			return "暂未生成 MR"
		default:
			return label + "：暂未生成"
		}
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
			if label == "MR" {
				itemLabel = "MR !" + urlTail(value)
			}
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

func urlTail(value string) string {
	trimmed := strings.TrimSuffix(value, "/")
	if index := strings.LastIndex(trimmed, "/"); index >= 0 {
		return trimmed[index+1:]
	}
	return trimmed
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

func buildCardAction(current state.State, definition workflow.Workflow) cardAction {
	if current.CurrentStepID == nil {
		return cardAction{Theme: "green-50", Title: "✅ 流程已完成", Summary: "全部 Workflow Step 已完成。"}
	}
	context, _, ok := definition.FindStep(*current.CurrentStepID)
	if !ok {
		return cardAction{Theme: "red-50", Title: "⛔ 当前阻塞", Summary: "当前 Step 不在绑定的 Workflow 中。"}
	}
	summary := strings.TrimSpace(current.CurrentStepSummary)
	projected := workflowview.Project(definition, current).Current
	if summary == "" && projected != nil && projected.Prompt != nil {
		summary = strings.TrimSpace(projected.Prompt.Content)
	}
	switch current.CurrentStepStatus {
	case state.StepBlocked:
		return cardAction{Theme: "red-50", Title: "⛔ 当前阻塞", Summary: summary, Prompt: "请补充所缺的关键信息。"}
	case state.StepFixing:
		return cardAction{Theme: "orange-50", Title: "🔁 正在修复", Summary: summary}
	}
	if context.Step.Executor == workflow.StepExecutorHuman {
		prompt := "如认可，请回复：`批准继续`"
		if routes := definition.Flows[*current.CurrentStepID]; len(routes) > 0 {
			flow := routes[0]
			if flow.Terminal {
				prompt = "如认可，请回复：`批准完成`"
			} else if nextContext, _, nextFound := definition.FindStep(flow.NextStepID); nextFound && nextContext.Stage.ID != context.Stage.ID {
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
