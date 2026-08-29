package store

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/zeefan1555/fanloop/internal/idl/storageidl"
	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/traceconfig"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

var beijingTimezone = time.FixedZone("UTC+8", 8*60*60)

// RenderEvents rebuilds the human-facing Trace from committed State and Events.
func RenderEvents(root string, current state.State, definition workflow.Workflow, events []state.Event) []byte {
	var output strings.Builder
	output.WriteString("# Workflow Trace\n\n")
	fmt.Fprintf(&output, "- Requirement：%s\n", current.Requirement.Title)
	fmt.Fprintf(&output, "- 来源：%s\n", firstNonEmpty(current.Requirement.SourceURL, "未提供"))
	fmt.Fprintf(&output, "- Requirement Root：%s\n", root)
	fmt.Fprintf(&output, "- Workflow：%s\n", current.Release.Workflow.ID)
	fmt.Fprintf(&output, "- Release：%s\n", current.Release.Version)
	fmt.Fprintf(&output, "- 更新时间：%s\n", beijingTime(current.UpdatedAt))
	fmt.Fprintf(&output, "- 回流次数：共 %d 次\n", traceLoopCount(events))

	output.WriteString("\n## 状态全景\n\n")
	output.WriteString(tracePanorama(current, definition))
	output.WriteString("\n\n")
	output.WriteString(traceDocumentLink(current))

	output.WriteString("\n\n## 当前可上报 Condition\n\n")
	output.WriteString(strings.Join(currentConditionLines(current, definition), "\n"))

	output.WriteString("\n\n## 当前有效 Output\n\n")
	output.WriteString(strings.Join(currentOutputLines(current), "\n"))

	output.WriteString("\n\n## Trace Log\n\n")
	output.WriteString(strings.Join(traceLogLines(definition, events), "\n"))
	output.WriteByte('\n')
	return []byte(output.String())
}

// RenderTraceConfig projects synchronization metadata. It is never a runtime source.
func RenderTraceConfig(current state.State, events []state.Event) ([]byte, bool) {
	config := &storageidl.TraceConfig{SchemaVersion: storageidl.TRACE_CONFIG_SCHEMA_VERSION}
	if current.Integrations.Trace != nil {
		config.TraceDocumentUrl = projectionStringPointer(current.Integrations.Trace.DocumentURL)
		if current.Integrations.Trace.CLILogDocumentURL != "" {
			config.CliLogDocumentUrl = projectionStringPointer(current.Integrations.Trace.CLILogDocumentURL)
		}
		if registry, ok := traceconfig.Resolve(current.Integrations.Trace.Registry, current.Release.Workflow.ID); ok {
			profile, err := storageidl.RegistryProfileFromString(string(registry.Profile))
			if err != nil {
				return nil, false
			}
			config.RegistryProfile = &profile
			config.RegistryUrl = projectionStringPointer(registry.URL)
			config.RegistryBaseToken = projectionStringPointer(registry.BaseToken)
			config.RegistryTableId = projectionStringPointer(registry.TableID)
			config.RegistryViewId = projectionStringPointer(registry.ViewID)
		}
	}
	required := config.TraceDocumentUrl != nil
	for _, event := range events {
		if event.Kind != state.EventTraceSynced {
			continue
		}
		required = true
		payload, ok := state.EventPayloadAs[state.TraceSyncedPayload](event)
		if !ok {
			continue
		}
		timestamp := beijingTime(event.OccurredAt)
		errors := make([]string, 0)
		for _, target := range payload.Targets {
			errorText := traceTargetErrorText(target)
			switch target.Name {
			case "trace_document":
				if target.Status == "succeeded" {
					config.TraceDocumentLastSyncAt, config.TraceDocumentLastSyncError = timestamp, ""
				} else if target.Status == "failed" {
					config.TraceDocumentLastSyncError = errorText
				}
			case "cli_log_document":
				if target.Status == "succeeded" {
					config.CliLogDocumentLastSyncAt, config.CliLogDocumentLastSyncError = timestamp, ""
				} else if target.Status == "failed" {
					config.CliLogDocumentLastSyncError = errorText
				}
			case "registry":
				if target.Status == "succeeded" {
					config.RegistryLastSyncAt, config.RegistryLastSyncError = timestamp, ""
				} else if target.Status == "failed" {
					config.RegistryLastSyncError = errorText
				}
			}
			if target.Status == "failed" {
				errors = append(errors, target.Name+": "+firstNonEmpty(errorText, target.Reason, "failed"))
			}
		}
		if payload.Outcome == state.TraceSyncSucceeded {
			config.LastSyncAt, config.LastSyncError = timestamp, ""
		} else if payload.Outcome == state.TraceSyncPartial {
			config.LastSyncError = strings.Join(errors, "; ")
		}
	}
	if !required {
		return nil, false
	}
	if err := config.IsValid(); err != nil {
		return nil, false
	}
	content, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, false
	}
	return append(content, '\n'), true
}

func projectionStringPointer(value string) *string { return &value }

func traceTargetErrorText(target state.TraceTargetResult) string {
	if target.Error != nil {
		return firstNonEmpty(target.Error.Message, target.Error.Code)
	}
	return target.Reason
}

func tracePanorama(current state.State, definition workflow.Workflow) string {
	currentPosition := len(definition.OrderedStepIDs())
	currentFound := current.CurrentStepID == nil
	if current.CurrentStepID != nil {
		_, position, ok := definition.FindStep(*current.CurrentStepID)
		if ok {
			currentPosition, currentFound = position, true
		}
	}
	position := 0
	lines := make([]string, 0, len(definition.Stages)+1)
	for _, stage := range definition.Stages {
		steps := make([]string, 0)
		for _, job := range stage.Jobs {
			for _, step := range job.Steps {
				label := step.Name
				switch {
				case current.CurrentStepID == nil || currentFound && position < currentPosition:
					label = "✅ " + label
				case currentFound && position == currentPosition:
					label = "**" + label + "（" + traceStatusLabel(current, definition) + "）**"
				}
				steps = append(steps, label)
				position++
			}
		}
		lines = append(lines, stage.Name+"："+strings.Join(steps, " → "))
	}
	lines = append(lines, fmt.Sprintf("整体进度：%d%%", traceProgressPercent(current, definition)))
	return strings.Join(lines, "\n")
}

func traceStatusLabel(current state.State, definition workflow.Workflow) string {
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

func traceProgressPercent(current state.State, definition workflow.Workflow) int {
	steps := definition.OrderedStepIDs()
	if len(steps) == 0 || current.CurrentStepID == nil {
		if current.CurrentStepID == nil {
			return 100
		}
		return 0
	}
	_, position, ok := definition.FindStep(*current.CurrentStepID)
	if !ok {
		return 0
	}
	return (position*100 + len(steps)/2) / len(steps)
}

func traceDocumentLink(current state.State) string {
	lines := []string{}
	if current.Integrations.Trace != nil && current.Integrations.Trace.DocumentURL != "" {
		lines = append(lines, "📄 Trace 文档：[点此查看飞书审查文档]("+current.Integrations.Trace.DocumentURL+")")
		if current.Integrations.Trace.CLILogDocumentURL != "" {
			lines = append(lines, "📜 CLI 日志：[查看完整输入输出]("+current.Integrations.Trace.CLILogDocumentURL+")")
		}
		return strings.Join(lines, "\n")
	}
	return "📄 Trace 文档：未绑定"
}

func currentConditionLines(current state.State, definition workflow.Workflow) []string {
	if current.CurrentStepID == nil {
		return []string{"- Workflow 已完成"}
	}
	ids := definition.RelevantConditionIDs(*current.CurrentStepID)
	if len(ids) == 0 {
		return []string{"- 无"}
	}
	lines := []string{"| Condition | Output key | Type |", "|---|---|---|"}
	for _, id := range ids {
		condition, ok := definition.Condition(id)
		if ok {
			lines = append(lines, fmt.Sprintf("| %s | %s | %s |", mdCell(id), mdCell(condition.Output.Key), mdCell(condition.Output.Type)))
		}
	}
	return lines
}

func currentOutputLines(current state.State) []string {
	if len(current.Outputs) == 0 {
		return []string{"- 无"}
	}
	keys := make([]string, 0, len(current.Outputs))
	for key := range current.Outputs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := []string{"| Output | Type | Producer Step | Value |", "|---|---|---|---|"}
	for _, key := range keys {
		value := current.Outputs[key]
		lines = append(lines, fmt.Sprintf("| %s | %s | %s | %s |", mdCell(key), mdCell(value.Type), mdCell(value.ProducerStepID), mdCell(compactRaw(value.Value))))
	}
	return lines
}

func traceLogLines(definition workflow.Workflow, events []state.Event) []string {
	lines := []string{
		"| 时间 | 事件 | Skill | 状态变化 | 结果 | 用户对话 | 判断依据 | 证据 |",
		"|---|---|---|---|---|---|---|---|",
	}
	for index := len(events) - 1; index >= 0; index-- {
		row := traceEventRow(definition, events[index])
		lines = append(lines, fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s | %s |",
			mdCell(beijingTime(events[index].OccurredAt)), mdCell(row.event), mdCell(row.skill), mdCell(row.transition),
			mdCell(row.result), mdCell(row.conversation), mdCell(row.reason), mdCell(strings.Join(row.evidence, "；"))))
	}
	return lines
}

type traceRow struct {
	event        string
	skill        string
	transition   string
	result       string
	conversation string
	reason       string
	evidence     []string
}

func traceEventRow(definition workflow.Workflow, event state.Event) traceRow {
	row := traceRow{event: string(event.Kind), skill: "未记录", transition: "— → —", result: "progress"}
	switch event.Kind {
	case state.EventFlowInitialized:
		payload, ok := state.EventPayloadAs[state.FlowInitializedPayload](event)
		if !ok {
			return row
		}
		row.event = stageNameForStep(definition, payload.StepID)
		row.skill = skillForStep(definition, payload.StepID)
		row.transition = "new/— → " + locationForStep(definition, payload.StepID)
		row.result, row.reason = "started", "Workflow 已初始化"
	case state.EventFlowProgressed:
		payload, ok := state.EventPayloadAs[state.FlowProgressPayload](event)
		if !ok {
			return row
		}
		row.event = stageNameForStep(definition, payload.FromStepID)
		row.skill = skillForStep(definition, payload.FromStepID)
		row.transition = locationForStep(definition, payload.FromStepID) + " → " + locationForStep(definition, payload.FromStepID)
		row.result, row.reason = string(payload.ToStepStatus), payload.Summary
		row.conversation = humanEvidence(payload.Evidence)
		row.evidence = executionEvidence(payload.Evidence, state.OutputChanges{})
	case state.EventFlowResult:
		payload, ok := state.EventPayloadAs[state.FlowResultPayload](event)
		if !ok {
			return row
		}
		row.event = stageNameForStep(definition, payload.Transition.FromStepID)
		row.skill = skillForStep(definition, payload.Transition.FromStepID)
		row.transition = locationForStep(definition, payload.Transition.FromStepID) + " → " + locationForStep(definition, payload.Transition.ToStepID)
		row.result, row.reason = string(payload.Effect), payload.Summary
		row.conversation = humanEvidence(payload.Evidence)
		row.evidence = executionEvidence(payload.Evidence, payload.OutputChanges)
		for _, result := range payload.ConditionResults {
			row.evidence = append(row.evidence, "condition="+result.ConditionID+":"+compactRaw(result.Output.Value))
		}
	case state.EventTraceDocumentBound:
		payload, _ := state.EventPayloadAs[state.TraceDocumentBoundPayload](event)
		row.event, row.result, row.reason = "Trace", "passed", "绑定独立 Trace 文档"
		row.evidence = nonEmptyStrings(payload.DocumentURL)
	case state.EventTraceSyncStarted:
		payload, _ := state.EventPayloadAs[state.TraceSyncStartedPayload](event)
		row.event, row.result, row.reason = "Trace", "started", "开始同步 Trace"
		row.evidence = append([]string(nil), payload.Targets...)
	case state.EventTraceSynced:
		payload, _ := state.EventPayloadAs[state.TraceSyncedPayload](event)
		row.event, row.result, row.reason = "Trace", string(payload.Outcome), "Trace 同步完成"
		for _, target := range payload.Targets {
			item := target.Name + "=" + target.Status
			if target.Reason != "" {
				item += ":" + target.Reason
			}
			if target.Error != nil {
				item += ":" + target.Error.Code + ":" + target.Error.Message
			}
			row.evidence = append(row.evidence, item)
		}
	}
	if row.skill == "" {
		row.skill = "未记录"
	} else if !strings.Contains(row.skill, "@") {
		row.skill += "@未记录"
	}
	return row
}

func executionEvidence(evidence []state.Evidence, changes state.OutputChanges) []string {
	result := make([]string, 0, len(evidence)+2)
	for _, item := range evidence {
		result = append(result, string(item.Source)+":"+item.Content)
	}
	if len(changes.Accepted) > 0 {
		result = append(result, "Output="+strings.Join(changes.Accepted, ","))
	}
	if len(changes.Invalidated) > 0 {
		result = append(result, "失效 Output="+strings.Join(changes.Invalidated, ","))
	}
	return result
}

func traceLoopCount(events []state.Event) int {
	total := 0
	for _, event := range events {
		if event.Kind != state.EventFlowResult {
			continue
		}
		payload, ok := state.EventPayloadAs[state.FlowResultPayload](event)
		if ok && payload.Effect == state.ResultLooped {
			total++
		}
	}
	return total
}

func stageNameForStep(definition workflow.Workflow, stepID string) string {
	if context, _, ok := definition.FindStep(stepID); ok {
		return context.Stage.Name
	}
	return "Workflow"
}

func skillForStep(definition workflow.Workflow, stepID string) string {
	routes := definition.Flows[stepID]
	if len(routes) == 0 {
		return "未记录"
	}
	prompt, ok := definition.Prompt(routes[0].PromptRef)
	if ok && len(prompt.Skills) > 0 {
		return prompt.Skills[0].ID
	}
	return "未记录"
}

func locationForStep(definition workflow.Workflow, stepID string) string {
	if stepID == "" {
		return "Done/完成"
	}
	if context, _, ok := definition.FindStep(stepID); ok {
		return context.Stage.Name + "/" + context.Job.Name + "/" + context.Step.Name
	}
	return "Workflow/" + stepID
}

func humanEvidence(values []state.Evidence) string {
	result := make([]string, 0)
	for _, item := range values {
		if item.Source == state.EvidenceHuman {
			result = append(result, item.Content)
		}
	}
	return strings.Join(result, "；")
}

func formatEvidence(values []state.Evidence) string {
	result := make([]string, 0, len(values))
	for _, item := range values {
		result = append(result, string(item.Source)+":"+item.Content)
	}
	return strings.Join(result, "；")
}

func compactRaw(raw json.RawMessage) string {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return "数据无效"
	}
	if text, ok := value.(string); ok {
		return text
	}
	content, err := json.Marshal(value)
	if err != nil {
		return "数据无效"
	}
	return string(content)
}

func mdCell(value any) string {
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" {
		return "—"
	}
	return strings.NewReplacer("|", "/", "\n", " ", "\r", " ").Replace(text)
}

func beijingTime(value time.Time) string {
	if value.IsZero() {
		return "—"
	}
	return value.In(beijingTimezone).Format(time.RFC3339)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func nonEmptyStrings(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}
