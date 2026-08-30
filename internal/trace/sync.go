package trace

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/zeefan1555/fanloop/internal/executionlog"
	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/idl/traceidl"
	"github.com/zeefan1555/fanloop/internal/larkexec"
	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/store"
	"github.com/zeefan1555/fanloop/internal/traceconfig"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

func (runtime Runtime) Sync(ctx context.Context, root string, request *traceidl.TraceSyncRequest, dryRun bool) (*traceidl.TraceSyncResponse, error) {
	if request == nil {
		return nil, publicError(erroridl.ErrorCode_INVALID_ARGUMENT, "request is required")
	}
	local, current, loaded, failure := load(root)
	if failure != nil {
		return nil, failure
	}
	events, failure := local.Events()
	if failure != nil {
		return nil, failure
	}
	results := plannedSyncTargets(current)
	if dryRun {
		return syncResponse(traceidl.TraceSyncOutcome_skipped, results), nil
	}

	previousEventID := current.LastEventID
	startedEventID := ""
	if current.Integrations.Trace != nil {
		startedEventID = runtime.eventID()
		current.LastEventID, current.UpdatedAt = startedEventID, runtime.now()
		started := state.Event{
			SchemaVersion: state.CurrentEventSchemaVersion, ID: startedEventID, OccurredAt: current.UpdatedAt,
			Kind: state.EventTraceSyncStarted, Command: "trace.sync", Workflow: state.WorkflowRefFrom(loaded.Ref), CausedByEventID: previousEventID,
			Payload: state.Payload(state.TraceSyncStartedPayload{Targets: targetNames(results)}),
		}
		if failure := local.Commit(current, started); failure != nil {
			return nil, failure
		}
		events, failure = local.Events()
		if failure != nil {
			return nil, failure
		}
		content := store.RenderEvents(root, current, loaded.Workflow, events)
		// The targets are independent network writes, so run them concurrently.
		tasks := []func() *traceidl.TraceTargetResult{
			func() *traceidl.TraceTargetResult {
				return syncTraceDocument(ctx, current.Integrations.Trace.DocumentURL, content)
			},
		}
		if current.Integrations.Trace.CLILogDocumentURL != "" {
			tasks = append(tasks, func() *traceidl.TraceTargetResult {
				return syncCLILogDocument(ctx, root, current.Integrations.Trace.CLILogDocumentURL)
			})
		}
		tasks = append(tasks, func() *traceidl.TraceTargetResult {
			return syncRegistry(ctx, current, loaded.Workflow, events, current.Integrations.Trace.DocumentURL)
		})
		var wait sync.WaitGroup
		wait.Add(len(tasks))
		for index, task := range tasks {
			go func() {
				defer wait.Done()
				results[index] = task()
			}()
		}
		wait.Wait()
	}
	partial := false
	allSkipped := true
	for _, target := range results {
		partial = partial || target.Status == traceidl.TraceTargetStatus_failed
		allSkipped = allSkipped && target.Status == traceidl.TraceTargetStatus_skipped
	}
	outcome := traceidl.TraceSyncOutcome_succeeded
	if partial {
		outcome = traceidl.TraceSyncOutcome_partial
	} else if allSkipped {
		outcome = traceidl.TraceSyncOutcome_skipped
	}
	data := syncResponse(outcome, results)
	now := runtime.now()
	eventID := runtime.eventID()
	causedBy := current.LastEventID
	current.LastEventID, current.UpdatedAt = eventID, now
	event := state.Event{
		SchemaVersion: state.CurrentEventSchemaVersion, ID: eventID, OccurredAt: now, Kind: state.EventTraceSynced,
		Command: "trace.sync", Workflow: state.WorkflowRefFrom(loaded.Ref), CausedByEventID: causedBy,
		Payload: state.Payload(state.TraceSyncedPayload{Outcome: state.TraceSyncOutcome(outcome.String()), Targets: durableTargets(results)}),
	}
	if failure := local.Commit(current, event); failure != nil {
		return nil, failure
	}
	return data, nil
}

func syncResponse(outcome traceidl.TraceSyncOutcome, results []*traceidl.TraceTargetResult) *traceidl.TraceSyncResponse {
	return &traceidl.TraceSyncResponse{Outcome: outcome, Targets: results}
}

func plannedSyncTargets(current state.State) []*traceidl.TraceTargetResult {
	targets := []traceidl.TraceTarget{traceidl.TraceTarget_trace_document}
	profile := traceconfig.RegistryProduction
	if current.Integrations.Trace != nil {
		profile = current.Integrations.Trace.Registry
	}
	registry, registryOK := traceconfig.Resolve(profile, current.Release.Workflow.ID)
	if current.Integrations.Trace != nil && current.Integrations.Trace.CLILogDocumentURL != "" ||
		current.Integrations.Trace == nil && registryOK && registry.RequireCLILogDocument {
		targets = append(targets, traceidl.TraceTarget_cli_log_document)
	}
	targets = append(targets, traceidl.TraceTarget_registry)
	results := make([]*traceidl.TraceTargetResult, 0, len(targets))
	if current.Integrations.Trace == nil {
		for _, target := range targets {
			results = append(results, &traceidl.TraceTargetResult{
				Target: target, Status: traceidl.TraceTargetStatus_skipped, Reason: stringPointer("Trace document is not bound"),
			})
		}
		return results
	}
	for _, target := range targets {
		results = append(results, &traceidl.TraceTargetResult{
			Target: target, Status: traceidl.TraceTargetStatus_skipped, Reason: stringPointer("dry run"),
		})
	}
	return results
}

func targetNames(results []*traceidl.TraceTargetResult) []string {
	names := make([]string, len(results))
	for index, result := range results {
		names[index] = result.Target.String()
	}
	return names
}

func syncTraceDocument(ctx context.Context, documentURL string, content []byte) *traceidl.TraceTargetResult {
	return syncDocument(ctx, traceidl.TraceTarget_trace_document, documentURL, content)
}

func syncCLILogDocument(ctx context.Context, root, documentURL string) *traceidl.TraceTargetResult {
	content, err := executionlog.ReadAll(root)
	if err != nil {
		return failedTarget(traceidl.TraceTarget_cli_log_document, &traceidl.TraceTargetError{
			Code: erroridl.ErrorCode_TRACE_UPDATE_FAILED, Message: err.Error(), Retryable: false,
		})
	}
	return syncDocument(ctx, traceidl.TraceTarget_cli_log_document, documentURL, renderCLILogDocument(content))
}

func syncDocument(ctx context.Context, target traceidl.TraceTarget, documentURL string, content []byte) *traceidl.TraceTargetResult {
	result, err := larkexec.Execute(ctx, []string{
		"docs", "+update", "--as", "bot", "--doc", documentURL, "--command", "overwrite",
		"--doc-format", "markdown", "--content", "-",
	}, bytes.NewReader(content), 35*time.Second)
	if err != nil {
		code := erroridl.ErrorCode_NETWORK_FAILED
		if target == traceidl.TraceTarget_cli_log_document {
			code = erroridl.ErrorCode_TRACE_UPDATE_FAILED
		}
		return failedTarget(target, &traceidl.TraceTargetError{Code: code, Message: err.Error(), Retryable: true})
	}
	if result.ExitCode != 0 {
		return failedTarget(target, larkFailure(result, erroridl.ErrorCode_TRACE_UPDATE_FAILED))
	}
	var response struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil || !response.OK {
		return failedTarget(target, &traceidl.TraceTargetError{
			Code: erroridl.ErrorCode_TRACE_UPDATE_FAILED, Message: firstNonEmpty(errorText(err), strings.TrimSpace(result.Stdout), "Lark document update failed"), Retryable: true,
		})
	}
	return &traceidl.TraceTargetResult{Target: target, Status: traceidl.TraceTargetStatus_succeeded}
}

func renderCLILogDocument(content []byte) []byte {
	longest, current := 0, 0
	for _, value := range content {
		if value == '`' {
			current++
			if current > longest {
				longest = current
			}
		} else {
			current = 0
		}
	}
	if longest < 2 {
		longest = 2
	}
	fence := bytes.Repeat([]byte{'`'}, longest+1)
	result := append([]byte("# Fanloop CLI 日志\n\n> 完整、未脱敏且未截断的 Requirement CLI 输入输出；本地 `.fanloop/log/cli.jsonl` 是事实源。\n\n"), fence...)
	result = append(result, []byte("jsonl\n")...)
	result = append(result, content...)
	if len(content) == 0 || content[len(content)-1] != '\n' {
		result = append(result, '\n')
	}
	result = append(result, fence...)
	return append(result, '\n')
}

func syncRegistry(ctx context.Context, current state.State, definition workflow.Workflow, events []state.Event, traceURL string) *traceidl.TraceTargetResult {
	registry, ok := traceconfig.Resolve(current.Integrations.Trace.Registry, current.Release.Workflow.ID)
	if !ok {
		return failedTarget(traceidl.TraceTarget_registry, &traceidl.TraceTargetError{Code: erroridl.ErrorCode_REGISTRY_UPDATE_FAILED, Message: "Trace Registry profile is invalid"})
	}
	key := state.TraceDocumentKey(traceURL)
	if key == "" {
		return failedTarget(traceidl.TraceTarget_registry, &traceidl.TraceTargetError{Code: erroridl.ErrorCode_REGISTRY_UPDATE_FAILED, Message: "Trace URL cannot be converted to a registry key"})
	}
	identity, failure := runLarkJSON(ctx, []string{"whoami", "--as", "bot"}, nil, erroridl.ErrorCode_UPSTREAM_AUTH_FAILED)
	if failure != nil {
		return failedTarget(traceidl.TraceTarget_registry, failure)
	}
	if identity["identity"] != "bot" || identity["available"] != true {
		return failedTarget(traceidl.TraceTarget_registry, &traceidl.TraceTargetError{Code: erroridl.ErrorCode_UPSTREAM_AUTH_FAILED, Message: "lark-cli bot identity is not ready"})
	}
	filter, _ := json.Marshal(map[string]any{"logic": "and", "conditions": [][]string{{registry.Fields.TraceKey, "==", key}}})
	listArgs := []string{
		"base", "+record-list", "--as", "bot", "--base-token", registry.BaseToken, "--table-id", registry.TableID, "--view-id", registry.ViewID,
		"--field-id", registry.Fields.TraceKey, "--filter-json", string(filter), "--limit", "2", "--format", "json",
	}
	listed, failure := runLarkJSON(ctx, listArgs, nil, erroridl.ErrorCode_REGISTRY_UPDATE_FAILED)
	if failure != nil {
		return failedTarget(traceidl.TraceTarget_registry, failure)
	}
	records := recordsFromResponse(listed)
	if len(records) > 1 {
		return failedTarget(traceidl.TraceTarget_registry, &traceidl.TraceTargetError{Code: erroridl.ErrorCode_REGISTRY_UPDATE_FAILED, Message: "duplicate Trace Registry rows"})
	}
	recordID := ""
	if len(records) == 1 {
		recordID = stringField(records[0], "record_id", "id", "recordId")
	}
	fields, _ := json.Marshal(registryFields(registry, current, definition, events, traceURL, key, ""))
	args := []string{
		"base", "+record-upsert", "--as", "bot", "--base-token", registry.BaseToken, "--table-id", registry.TableID,
		"--json", string(fields), "--format", "json",
	}
	if recordID != "" {
		args = append(args, "--record-id", recordID)
	}
	upserted, failure := runLarkJSON(ctx, args, nil, erroridl.ErrorCode_REGISTRY_UPDATE_FAILED)
	if failure != nil {
		return failedTarget(traceidl.TraceTarget_registry, failure)
	}
	if recordID == "" {
		recordID = responseRecordID(upserted)
	}
	if recordID == "" {
		listed, failure = runLarkJSON(ctx, listArgs, nil, erroridl.ErrorCode_REGISTRY_UPDATE_FAILED)
		if failure != nil {
			return failedTarget(traceidl.TraceTarget_registry, failure)
		}
		created := recordsFromResponse(listed)
		if len(created) != 1 {
			return failedTarget(traceidl.TraceTarget_registry, &traceidl.TraceTargetError{Code: erroridl.ErrorCode_REGISTRY_UPDATE_FAILED, Message: "Trace Registry create could not be resolved to one row", Retryable: true})
		}
		recordID = stringField(created[0], "record_id", "id", "recordId")
	}
	if recordID == "" {
		return failedTarget(traceidl.TraceTarget_registry, &traceidl.TraceTargetError{Code: erroridl.ErrorCode_REGISTRY_UPDATE_FAILED, Message: "Trace Registry upsert returned no record_id", Retryable: true})
	}
	verified, failure := runLarkJSON(ctx, []string{
		"base", "+record-get", "--as", "bot", "--base-token", registry.BaseToken, "--table-id", registry.TableID,
		"--record-id", recordID, "--field-id", registry.Fields.TraceKey, "--format", "json",
	}, nil, erroridl.ErrorCode_REGISTRY_UPDATE_FAILED)
	if failure != nil {
		return failedTarget(traceidl.TraceTarget_registry, failure)
	}
	verifiedRecords := recordsFromResponse(verified)
	data := responseData(verified)
	if len(verifiedRecords) > 0 {
		data = verifiedRecords[0]
	} else if record, ok := data["record"].(map[string]any); ok {
		data = record
	}
	fieldsMap, _ := data["fields"].(map[string]any)
	if fmt.Sprint(fieldsMap[registry.Fields.TraceKey]) != key {
		return failedTarget(traceidl.TraceTarget_registry, &traceidl.TraceTargetError{Code: erroridl.ErrorCode_REGISTRY_UPDATE_FAILED, Message: "Trace Registry record verification failed", Retryable: true})
	}
	return &traceidl.TraceTargetResult{Target: traceidl.TraceTarget_registry, Status: traceidl.TraceTargetStatus_succeeded}
}

func runLarkJSON(ctx context.Context, args []string, stdin io.Reader, fallbackCode erroridl.ErrorCode) (map[string]any, *traceidl.TraceTargetError) {
	result, err := larkexec.Execute(ctx, args, stdin, 35*time.Second)
	if err != nil {
		return nil, &traceidl.TraceTargetError{Code: erroridl.ErrorCode_NETWORK_FAILED, Message: err.Error(), Retryable: true}
	}
	if result.ExitCode != 0 {
		return nil, larkFailure(result, fallbackCode)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &payload); err != nil {
		return nil, &traceidl.TraceTargetError{Code: fallbackCode, Message: err.Error(), Retryable: true}
	}
	if ok, exists := payload["ok"].(bool); exists && !ok {
		return nil, larkFailure(result, fallbackCode)
	}
	return payload, nil
}

func registryFields(registry traceconfig.Registry, current state.State, definition workflow.Workflow, events []state.Event, traceURL, key, openID string) map[string]any {
	loops := 0
	for _, event := range events {
		if event.Kind == state.EventFlowResult {
			payload, ok := state.EventPayloadAs[state.FlowResultPayload](event)
			if ok && payload.Effect == state.ResultLooped {
				loops++
			}
		}
	}
	mapping := registry.Fields
	fields := map[string]any{
		mapping.TraceKey: key, mapping.Title: current.Requirement.Title,
		mapping.Location: registryStageAndAudit(current, definition, events), mapping.Status: registryStatus(current, definition), mapping.LoopCount: loops,
		mapping.SourceURL: registrySourceValue(registry, current), mapping.TraceURL: traceURL,
		mapping.UpdatedAt: registryTime(current.UpdatedAt), mapping.Origin: "runtime",
	}
	if openID != "" {
		fields[mapping.Owner] = []map[string]string{{"id": openID}}
	}
	if mapping.CLILogURL != "" {
		fields[mapping.CLILogURL] = nil
		if current.Integrations.Trace != nil {
			fields[mapping.CLILogURL] = current.Integrations.Trace.CLILogDocumentURL
		}
	}
	for outputKey, fieldName := range mapping.Outputs {
		fields[fieldName] = registryOutputValue(current.Outputs[outputKey])
	}
	return fields
}

func registrySourceValue(registry traceconfig.Registry, current state.State) any {
	value := strings.TrimSpace(current.Requirement.SourceURL)
	if outputKey := registry.Fields.SourceOutput; outputKey != "" {
		if candidate, ok := registryOutputValue(current.Outputs[outputKey]).(string); ok && candidate != "" {
			value = candidate
		}
	}
	if value == "" || registry.Fields.SourceDocumentOnly && !state.ValidTraceDocumentURL(value) {
		return nil
	}
	return value
}

func registryOutputValue(output state.RegisteredOutput) any {
	if len(output.Value) == 0 {
		return nil
	}
	var value any
	if json.Unmarshal(output.Value, &value) != nil {
		return nil
	}
	if values, ok := value.([]any); ok {
		for _, item := range values {
			if text, ok := item.(string); ok && text != "" {
				return text
			}
		}
		return nil
	}
	return value
}

func registryStageAndAudit(current state.State, definition workflow.Workflow, events []state.Event) string {
	stage := currentStageStep(current, definition)
	if audit := latestFlowAudit(events); audit != "" {
		return stage + "\n" + audit
	}
	return stage
}

func latestFlowAudit(events []state.Event) string {
	for index := len(events) - 1; index >= 0; index-- {
		switch events[index].Kind {
		case state.EventFlowProgressed:
			payload, ok := state.EventPayloadAs[state.FlowProgressPayload](events[index])
			if !ok {
				return ""
			}
			return "event: report=progress; from=" + payload.FromStepID + "; status=" + string(payload.ToStepStatus)
		case state.EventFlowResult:
			payload, ok := state.EventPayloadAs[state.FlowResultPayload](events[index])
			if !ok {
				return ""
			}
			parts := []string{"report=result", "effect=" + string(payload.Effect), "from=" + payload.Transition.FromStepID}
			if payload.Transition.ToStepID == "" {
				parts = append(parts, "to=completed")
			} else {
				parts = append(parts, "to="+payload.Transition.ToStepID)
			}
			conditions := make([]string, 0, len(payload.ConditionResults))
			for _, result := range payload.ConditionResults {
				conditions = append(conditions, result.ConditionID)
			}
			sort.Strings(conditions)
			if len(conditions) > 0 {
				parts = append(parts, "conditions="+strings.Join(conditions, ","))
			}
			if keys := sortedStrings(payload.OutputChanges.Accepted); len(keys) > 0 {
				parts = append(parts, "accepted="+strings.Join(keys, ","))
			}
			if keys := sortedStrings(payload.OutputChanges.Invalidated); len(keys) > 0 {
				parts = append(parts, "invalidated="+strings.Join(keys, ","))
			}
			return "event: " + strings.Join(parts, "; ")
		}
	}
	return ""
}

func sortedStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func currentStageStep(current state.State, definition workflow.Workflow) string {
	stepID := current.CurrentStepID
	if stepID == nil {
		ordered := definition.OrderedStepIDs()
		if len(ordered) == 0 {
			return "completed"
		}
		stepID = &ordered[len(ordered)-1]
	}
	context, _, ok := definition.FindStep(*stepID)
	if !ok {
		return *stepID
	}
	return context.Stage.Name + " / " + context.Job.Name + " / " + context.Step.Name
}

func registryStatus(current state.State, definition workflow.Workflow) string {
	if current.CurrentStepID == nil {
		return "已完成"
	}
	if current.CurrentStepStatus == state.StepBlocked {
		return "Blocked"
	}
	if context, _, ok := definition.FindStep(*current.CurrentStepID); ok && context.Step.Executor == workflow.StepExecutorHuman {
		return "Human Review"
	}
	return "In Progress"
}

func registryTime(value time.Time) string {
	return value.In(time.FixedZone("UTC+8", 8*60*60)).Format("2006-01-02 15:04:05")
}

func recordsFromResponse(payload map[string]any) []map[string]any {
	data := responseData(payload)
	for _, key := range []string{"items", "data", "records"} {
		values, ok := data[key].([]any)
		if !ok {
			continue
		}
		fieldNames := stringValues(data["fields"])
		recordIDs := stringValues(data["record_id_list"])
		result := make([]map[string]any, 0, len(values))
		for index, value := range values {
			if record, ok := value.(map[string]any); ok {
				result = append(result, record)
				continue
			}
			row, ok := value.([]any)
			if !ok {
				continue
			}
			fields := make(map[string]any, len(row))
			for column, cell := range row {
				if column < len(fieldNames) {
					fields[fieldNames[column]] = cell
				}
			}
			record := map[string]any{"fields": fields}
			if index < len(recordIDs) {
				record["record_id"] = recordIDs[index]
			}
			result = append(result, record)
		}
		return result
	}
	if record, ok := data["record"].(map[string]any); ok {
		return []map[string]any{record}
	}
	return nil
}

func stringValues(value any) []string {
	values, _ := value.([]any)
	result := make([]string, 0, len(values))
	for _, value := range values {
		if text, ok := value.(string); ok {
			result = append(result, text)
		}
	}
	return result
}

func responseRecordID(payload map[string]any) string {
	if records := recordsFromResponse(payload); len(records) > 0 {
		return stringField(records[0], "record_id", "id", "recordId")
	}
	return stringField(responseData(payload), "record_id", "id", "recordId")
}

func responseData(payload map[string]any) map[string]any {
	if data, ok := payload["data"].(map[string]any); ok {
		return data
	}
	return payload
}

func stringField(value map[string]any, keys ...string) string {
	for _, key := range keys {
		if result, ok := value[key].(string); ok && result != "" {
			return result
		}
	}
	return ""
}

func failedTarget(target traceidl.TraceTarget, failure *traceidl.TraceTargetError) *traceidl.TraceTargetResult {
	return &traceidl.TraceTargetResult{Target: target, Status: traceidl.TraceTargetStatus_failed, Error: failure}
}

func larkFailure(result larkexec.Result, fallbackCode erroridl.ErrorCode) *traceidl.TraceTargetError {
	failure := &traceidl.TraceTargetError{Code: fallbackCode, Message: commandError(result), Retryable: true}
	var response struct {
		Error struct {
			Code      string `json:"code"`
			Message   string `json:"message"`
			Retryable bool   `json:"retryable"`
		} `json:"error"`
	}
	if json.Unmarshal([]byte(firstNonEmpty(strings.TrimSpace(result.Stdout), strings.TrimSpace(result.Stderr))), &response) == nil {
		if response.Error.Message != "" {
			failure.Message = response.Error.Message
		}
		failure.Retryable = response.Error.Retryable
		code := strings.ToUpper(response.Error.Code)
		if parsed, err := erroridl.ErrorCodeFromString(code); err == nil {
			switch parsed {
			case erroridl.ErrorCode_UPSTREAM_AUTH_FAILED, erroridl.ErrorCode_NETWORK_FAILED,
				erroridl.ErrorCode_TRACE_UPDATE_FAILED, erroridl.ErrorCode_REGISTRY_UPDATE_FAILED:
				failure.Code = parsed
			}
		} else if strings.Contains(code, "AUTH") {
			failure.Code = erroridl.ErrorCode_UPSTREAM_AUTH_FAILED
		}
	}
	return failure
}

func commandError(result larkexec.Result) string {
	return firstNonEmpty(strings.TrimSpace(result.Stderr), strings.TrimSpace(result.Stdout), fmt.Sprintf("lark-cli exited with %d", result.ExitCode))
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
