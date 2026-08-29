package state

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/zeefan1555/commonloop/internal/idl/erroridl"
	"github.com/zeefan1555/commonloop/internal/idl/storageidl"
	"github.com/zeefan1555/commonloop/internal/traceconfig"
	"github.com/zeefan1555/commonloop/internal/workflow"
)

const (
	CurrentStateSchemaVersion = storageidl.FLOW_STATE_SCHEMA_VERSION
	CurrentEventSchemaVersion = storageidl.EVENT_SCHEMA_VERSION
)

type State struct {
	SchemaVersion      int
	Requirement        Requirement
	Release            Release
	CurrentStepID      *string
	CurrentStepStatus  StepStatus
	CurrentStepSummary string
	CurrentEvidence    []Evidence
	Outputs            map[string]RegisteredOutput
	Integrations       Integrations
	LastEventID        string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Requirement struct {
	Title     string
	SourceURL string
}

type WorkflowRef struct {
	ID     string
	Digest string
}

func WorkflowRefFrom(value workflow.Ref) WorkflowRef {
	return WorkflowRef{ID: value.ID, Digest: value.Digest}
}

func (value WorkflowRef) Ref() workflow.Ref {
	return workflow.Ref{ID: value.ID, Digest: value.Digest}
}

type Release struct {
	Version  string
	Workflow WorkflowRef
}

type Integrations struct {
	Trace *TraceBinding
}

type TraceBinding struct {
	DocumentURL       string
	Registry          traceconfig.RegistryProfile
	CLILogDocumentURL string
}

type StepStatus string

const (
	StepReady      StepStatus = "ready"
	StepInProgress StepStatus = "in_progress"
	StepFixing     StepStatus = "fixing"
	StepBlocked    StepStatus = "blocked"
)

type StepState struct {
	StepID string
	Status StepStatus
}

type Evidence struct {
	Source  EvidenceSource
	Content string
	Ref     string
}

type EvidenceSource string

const (
	EvidenceHuman  EvidenceSource = "human"
	EvidenceSystem EvidenceSource = "system"
	EvidenceAI     EvidenceSource = "ai"
	EvidenceFile   EvidenceSource = "file"
	EvidenceURL    EvidenceSource = "url"
)

type OutputValue struct {
	Type  workflow.OutputType
	Value json.RawMessage
}

type RegisteredOutput struct {
	Type           workflow.OutputType
	Value          json.RawMessage
	ProducerStepID string
}

type ConditionResult struct {
	ConditionID string
	Output      OutputValue
}

type ResultEffect string

const (
	ResultAdvanced  ResultEffect = "advanced"
	ResultLooped    ResultEffect = "looped"
	ResultCompleted ResultEffect = "completed"
)

type TransitionDirection string

const (
	TransitionFlow TransitionDirection = "flow"
	TransitionLoop TransitionDirection = "loop"
)

type Transition struct {
	Direction  TransitionDirection
	FromStepID string
	ToStepID   string
}

type OutputChanges struct {
	Accepted    []string
	Invalidated []string
}

type Event struct {
	SchemaVersion   int
	ID              string
	OccurredAt      time.Time
	Kind            EventKind
	Command         string
	Workflow        WorkflowRef
	CausedByEventID string
	Payload         any
}

type EventKind string

const (
	EventFlowInitialized    EventKind = "flow.initialized"
	EventFlowProgressed     EventKind = "flow.progressed"
	EventFlowResult         EventKind = "flow.result"
	EventTraceDocumentBound EventKind = "trace.document_bound"
	EventTraceSyncStarted   EventKind = "trace.sync_started"
	EventTraceSynced        EventKind = "trace.synced"
)

type FlowInitializedPayload struct {
	StepID     string
	StepStatus StepStatus
}

type FlowProgressPayload struct {
	FromStepID     string
	FromStepStatus StepStatus
	ToStepStatus   StepStatus
	Summary        string
	Evidence       []Evidence
}

type FlowResultPayload struct {
	ConditionResults []ConditionResult
	Evidence         []Evidence
	Summary          string
	Effect           ResultEffect
	Transition       Transition
	OutputChanges    OutputChanges
}

type TraceDocumentBoundPayload struct {
	PreviousDocumentURL       string
	DocumentURL               string
	PreviousRegistry          traceconfig.RegistryProfile
	Registry                  traceconfig.RegistryProfile
	PreviousCLILogDocumentURL string
	CLILogDocumentURL         string
}

type TraceSyncStartedPayload struct {
	Targets []string
}

type TraceSyncOutcome string

const (
	TraceSyncSucceeded TraceSyncOutcome = "succeeded"
	TraceSyncPartial   TraceSyncOutcome = "partial"
	TraceSyncSkipped   TraceSyncOutcome = "skipped"
)

type TraceSyncedPayload struct {
	Outcome TraceSyncOutcome
	Targets []TraceTargetResult
}

type TraceTargetResult struct {
	Name   string
	Status string
	Reason string
	Error  *TraceTargetError
}

type TraceTargetError struct {
	Code      string
	Message   string
	Retryable bool
}

var ErrSchemaUnsupported = errors.New("state schema version is unsupported")

func (value State) Validate() error {
	if value.SchemaVersion != CurrentStateSchemaVersion {
		if value.SchemaVersion != 0 {
			return fmt.Errorf("%w: %d", ErrSchemaUnsupported, value.SchemaVersion)
		}
		return fmt.Errorf("invalid state header")
	}
	if strings.TrimSpace(value.Requirement.Title) == "" || value.Release.Version == "" ||
		value.Release.Workflow.ID == "" || value.Release.Workflow.Digest == "" ||
		value.Outputs == nil || value.LastEventID == "" || value.CreatedAt.IsZero() || value.UpdatedAt.IsZero() {
		return fmt.Errorf("invalid state header")
	}
	if !validEvidence(value.CurrentEvidence) {
		return fmt.Errorf("invalid current Evidence")
	}
	if binding := value.Integrations.Trace; binding != nil {
		registry, registryOK := traceconfig.Resolve(binding.Registry, value.Release.Workflow.ID)
		if !ValidTraceDocumentURL(binding.DocumentURL) || !registryOK ||
			binding.CLILogDocumentURL != "" && !ValidTraceDocumentURL(binding.CLILogDocumentURL) ||
			SameTraceDocumentURL(binding.DocumentURL, binding.CLILogDocumentURL) ||
			registry.RequireCLILogDocument != (binding.CLILogDocumentURL != "") {
			return fmt.Errorf("invalid Trace binding")
		}
	}
	if value.CurrentStepID == nil {
		if value.CurrentStepStatus != "" || value.CurrentStepSummary != "" || len(value.CurrentEvidence) > 0 {
			return fmt.Errorf("completed Workflow has current Step facts")
		}
	} else if *value.CurrentStepID == "" || !validStepStatus(value.CurrentStepStatus) {
		return fmt.Errorf("invalid current Step facts")
	}
	for key, output := range value.Outputs {
		if key == "" || output.ProducerStepID == "" || validateOutputValue(OutputValue{Type: output.Type, Value: output.Value}) != nil {
			return fmt.Errorf("invalid registered Output %q", key)
		}
	}
	return nil
}

func (value State) ValidateAgainst(definition workflow.Workflow) error {
	currentPosition := len(definition.OrderedStepIDs())
	if value.CurrentStepID != nil {
		_, position, ok := definition.FindStep(*value.CurrentStepID)
		if !ok {
			return fmt.Errorf("current_step_id references an unknown Step")
		}
		currentPosition = position
	}
	for key, output := range value.Outputs {
		if err := definition.ValidateRegisteredOutput(key, output.Type, output.Value); err != nil {
			return err
		}
		_, producerPosition, ok := definition.FindStep(output.ProducerStepID)
		if !ok {
			return fmt.Errorf("Output %q has an unknown producer Step", key)
		}
		if producerPosition >= currentPosition {
			return fmt.Errorf("Output %q is not valid at the current Step", key)
		}
	}
	return nil
}

func ValidateEventPayload(event Event) error {
	var destination any
	expectedCommand := ""
	switch event.Kind {
	case EventFlowInitialized:
		value, ok := EventPayloadAs[FlowInitializedPayload](event)
		if !ok {
			return fmt.Errorf("invalid %s payload type", event.Kind)
		}
		destination, expectedCommand = value, "flow.init"
	case EventFlowProgressed:
		value, ok := EventPayloadAs[FlowProgressPayload](event)
		if !ok {
			return fmt.Errorf("invalid %s payload type", event.Kind)
		}
		destination, expectedCommand = value, "flow.report.progress"
	case EventFlowResult:
		value, ok := EventPayloadAs[FlowResultPayload](event)
		if !ok {
			return fmt.Errorf("invalid %s payload type", event.Kind)
		}
		destination, expectedCommand = value, "flow.report.result"
	case EventTraceDocumentBound:
		value, ok := EventPayloadAs[TraceDocumentBoundPayload](event)
		if !ok {
			return fmt.Errorf("invalid %s payload type", event.Kind)
		}
		destination, expectedCommand = value, "trace.bind"
	case EventTraceSyncStarted:
		value, ok := EventPayloadAs[TraceSyncStartedPayload](event)
		if !ok {
			return fmt.Errorf("invalid %s payload type", event.Kind)
		}
		destination, expectedCommand = value, "trace.sync"
	case EventTraceSynced:
		value, ok := EventPayloadAs[TraceSyncedPayload](event)
		if !ok {
			return fmt.Errorf("invalid %s payload type", event.Kind)
		}
		destination, expectedCommand = value, "trace.sync"
	default:
		return fmt.Errorf("unknown event kind %q", event.Kind)
	}
	if event.Command != expectedCommand {
		return fmt.Errorf("event kind %q requires command %q", event.Kind, expectedCommand)
	}
	if event.Kind != EventFlowInitialized && event.CausedByEventID == "" {
		return fmt.Errorf("event %q has no cause", event.Kind)
	}
	switch value := destination.(type) {
	case FlowInitializedPayload:
		if value.StepID == "" || value.StepStatus != StepReady {
			return fmt.Errorf("invalid initialized Step")
		}
	case FlowProgressPayload:
		if value.FromStepID == "" || !validStepStatus(value.FromStepStatus) ||
			value.ToStepStatus != StepInProgress && value.ToStepStatus != StepFixing && value.ToStepStatus != StepBlocked ||
			strings.TrimSpace(value.Summary) == "" || !validEvidence(value.Evidence) {
			return fmt.Errorf("invalid progress facts")
		}
	case FlowResultPayload:
		if err := validateResultPayload(value); err != nil {
			return err
		}
	case TraceDocumentBoundPayload:
		registry, registryOK := traceconfig.Resolve(value.Registry, event.Workflow.ID)
		if !ValidTraceDocumentURL(value.DocumentURL) || value.PreviousDocumentURL != "" && !ValidTraceDocumentURL(value.PreviousDocumentURL) ||
			!registryOK || value.PreviousRegistry != "" && !traceconfig.Valid(value.PreviousRegistry) ||
			value.CLILogDocumentURL != "" && !ValidTraceDocumentURL(value.CLILogDocumentURL) ||
			value.PreviousCLILogDocumentURL != "" && !ValidTraceDocumentURL(value.PreviousCLILogDocumentURL) ||
			SameTraceDocumentURL(value.DocumentURL, value.CLILogDocumentURL) ||
			registry.RequireCLILogDocument != (value.CLILogDocumentURL != "") {
			return fmt.Errorf("invalid Trace binding")
		}
	case TraceSyncStartedPayload:
		if !validTraceTargetNames(value.Targets) {
			return fmt.Errorf("invalid Trace sync targets")
		}
	case TraceSyncedPayload:
		if value.Outcome != TraceSyncSucceeded && value.Outcome != TraceSyncPartial && value.Outcome != TraceSyncSkipped || !validTraceTargetResults(value.Targets) {
			return fmt.Errorf("invalid Trace sync result")
		}
	}
	return nil
}

func validTraceTargetNames(values []string) bool {
	seen := map[string]bool{}
	for _, value := range values {
		if value != "trace_document" && value != "cli_log_document" && value != "registry" || seen[value] {
			return false
		}
		seen[value] = true
	}
	return len(seen) > 0
}

func validTraceTargetResults(values []TraceTargetResult) bool {
	names := make([]string, 0, len(values))
	for _, value := range values {
		names = append(names, value.Name)
		if value.Status != "succeeded" && value.Status != "failed" && value.Status != "skipped" {
			return false
		}
		if value.Error == nil {
			if value.Status == "failed" {
				return false
			}
			continue
		}
		code, err := erroridl.ErrorCodeFromString(value.Error.Code)
		if value.Status != "failed" || err != nil || code == erroridl.ErrorCode_unspecified || strings.TrimSpace(value.Error.Message) == "" {
			return false
		}
	}
	return validTraceTargetNames(names)
}

func (value Event) ValidateAgainst(definition workflow.Workflow) error {
	switch value.Kind {
	case EventFlowInitialized:
		payload, _ := EventPayloadAs[FlowInitializedPayload](value)
		first, ok := definition.FirstStepID()
		if !ok || payload.StepID != first {
			return fmt.Errorf("initialization does not enter the first Step")
		}
	case EventFlowProgressed:
		payload, _ := EventPayloadAs[FlowProgressPayload](value)
		if _, _, ok := definition.FindStep(payload.FromStepID); !ok {
			return fmt.Errorf("progress references an unknown Step")
		}
	case EventFlowResult:
		payload, _ := EventPayloadAs[FlowResultPayload](value)
		if err := validateResultAgainst(definition, payload); err != nil {
			return err
		}
	}
	return nil
}

func ValidateHistory(events []Event, current State, definition workflow.Workflow) error {
	if len(events) == 0 {
		return fmt.Errorf("event history is empty")
	}
	outputs := map[string]RegisteredOutput{}
	var cursor *StepState
	summary := ""
	var evidence []Evidence
	traceDocumentURL := ""
	traceRegistry := traceconfig.RegistryProfile("")
	cliLogDocumentURL := ""
	seen := map[string]bool{}

	for index, event := range events {
		if seen[event.ID] || event.Workflow != current.Release.Workflow {
			return fmt.Errorf("event history has duplicate IDs or a different Workflow binding")
		}
		if index > 0 && event.OccurredAt.Before(events[index-1].OccurredAt) {
			return fmt.Errorf("event history is out of order")
		}
		if index == 0 && event.CausedByEventID != "" || index > 0 && event.CausedByEventID != events[index-1].ID {
			return fmt.Errorf("event history cause does not match the previous event")
		}
		if err := event.ValidateAgainst(definition); err != nil {
			return fmt.Errorf("event %q: %w", event.ID, err)
		}
		switch event.Kind {
		case EventFlowInitialized:
			if index != 0 || cursor != nil {
				return fmt.Errorf("event %q reinitializes the Workflow", event.ID)
			}
			payload, _ := EventPayloadAs[FlowInitializedPayload](event)
			cursor = &StepState{StepID: payload.StepID, Status: payload.StepStatus}
			summary, evidence = "workflow initialized", nil
		case EventFlowProgressed:
			payload, _ := EventPayloadAs[FlowProgressPayload](event)
			if !sameCursor(cursor, payload.FromStepID, payload.FromStepStatus) {
				return fmt.Errorf("event %q does not continue from current Step", event.ID)
			}
			cursor = &StepState{StepID: payload.FromStepID, Status: payload.ToStepStatus}
			summary, evidence = payload.Summary, cloneEvidence(payload.Evidence)
		case EventFlowResult:
			payload, _ := EventPayloadAs[FlowResultPayload](event)
			if cursor == nil || cursor.StepID != payload.Transition.FromStepID {
				return fmt.Errorf("event %q does not continue from current Step", event.ID)
			}
			accepted, err := registeredOutputs(definition, payload.Transition.FromStepID, payload.ConditionResults)
			if err != nil || !slices.Equal(sortedKeys(accepted), sortedCopy(payload.OutputChanges.Accepted)) {
				return fmt.Errorf("event %q has invalid accepted Output changes", event.ID)
			}
			for key, output := range accepted {
				outputs[key] = output
			}
			if payload.Effect == ResultLooped {
				want, err := invalidatedOutputs(definition, payload.Transition.ToStepID, outputs)
				if err != nil || !slices.Equal(want, sortedCopy(payload.OutputChanges.Invalidated)) {
					return fmt.Errorf("event %q has invalidated %v, want %v", event.ID, payload.OutputChanges.Invalidated, want)
				}
				for _, key := range want {
					delete(outputs, key)
				}
			} else if len(payload.OutputChanges.Invalidated) > 0 {
				return fmt.Errorf("event %q invalidates Outputs on a Flow transition", event.ID)
			}
			switch payload.Effect {
			case ResultCompleted:
				cursor, summary, evidence = nil, "", nil
			case ResultAdvanced, ResultLooped:
				cursor = &StepState{StepID: payload.Transition.ToStepID, Status: StepReady}
				summary, evidence = payload.Summary, cloneEvidence(payload.Evidence)
			}
		case EventTraceDocumentBound:
			payload, _ := EventPayloadAs[TraceDocumentBoundPayload](event)
			if payload.PreviousDocumentURL != traceDocumentURL || payload.PreviousRegistry != traceRegistry ||
				payload.PreviousCLILogDocumentURL != cliLogDocumentURL {
				return fmt.Errorf("event %q does not continue from previous Trace binding", event.ID)
			}
			traceDocumentURL, traceRegistry, cliLogDocumentURL = payload.DocumentURL, payload.Registry, payload.CLILogDocumentURL
		}
		seen[event.ID] = true
	}
	if !equalOptionalStepState(cursor, stepStatePointer(StateRef(current))) ||
		summary != current.CurrentStepSummary || !slices.Equal(evidence, current.CurrentEvidence) ||
		!maps.EqualFunc(outputs, current.Outputs, equalRegisteredOutput) {
		return fmt.Errorf("event history tail does not match current State")
	}
	if current.Integrations.Trace == nil && (traceDocumentURL != "" || traceRegistry != "" || cliLogDocumentURL != "") ||
		current.Integrations.Trace != nil && (current.Integrations.Trace.DocumentURL != traceDocumentURL || current.Integrations.Trace.Registry != traceRegistry ||
			current.Integrations.Trace.CLILogDocumentURL != cliLogDocumentURL) {
		return fmt.Errorf("event history Trace binding does not match current State")
	}
	if current.LastEventID != events[len(events)-1].ID {
		return fmt.Errorf("state last_event_id does not match event history")
	}
	return nil
}

func validateResultPayload(value FlowResultPayload) error {
	if len(value.ConditionResults) == 0 || strings.TrimSpace(value.Summary) == "" || !validEvidence(value.Evidence) ||
		value.Transition.FromStepID == "" || !uniqueNonEmpty(value.OutputChanges.Accepted) || !uniqueNonEmpty(value.OutputChanges.Invalidated) {
		return fmt.Errorf("invalid Result facts")
	}
	seen := map[string]bool{}
	for _, result := range value.ConditionResults {
		if result.ConditionID == "" || seen[result.ConditionID] || validateOutputValue(result.Output) != nil {
			return fmt.Errorf("invalid ConditionResult")
		}
		seen[result.ConditionID] = true
	}
	switch value.Effect {
	case ResultAdvanced:
		if value.Transition.Direction != TransitionFlow || value.Transition.ToStepID == "" {
			return fmt.Errorf("invalid advanced Transition")
		}
	case ResultLooped:
		if value.Transition.Direction != TransitionLoop || value.Transition.ToStepID == "" {
			return fmt.Errorf("invalid looped Transition")
		}
	case ResultCompleted:
		if value.Transition.Direction != TransitionFlow || value.Transition.ToStepID != "" {
			return fmt.Errorf("invalid completed Transition")
		}
	default:
		return fmt.Errorf("invalid Result effect")
	}
	return nil
}

func validateResultAgainst(definition workflow.Workflow, payload FlowResultPayload) error {
	accepted, err := registeredOutputs(definition, payload.Transition.FromStepID, payload.ConditionResults)
	if err != nil {
		return err
	}
	if !slices.Equal(sortedKeys(accepted), sortedCopy(payload.OutputChanges.Accepted)) {
		return fmt.Errorf("accepted Output changes do not match ConditionResults")
	}
	conditionIDs := make(map[string]bool, len(payload.ConditionResults))
	for _, result := range payload.ConditionResults {
		conditionIDs[result.ConditionID] = true
	}
	flowMatches := matchingFlowRoutes(definition.Flows[payload.Transition.FromStepID], conditionIDs)
	if len(flowMatches) > 0 {
		if len(flowMatches) != 1 || payload.Transition.Direction != TransitionFlow || len(payload.OutputChanges.Invalidated) > 0 {
			return fmt.Errorf("Result does not select exactly one Flow Route")
		}
		route := flowMatches[0]
		if route.Terminal {
			if payload.Effect != ResultCompleted || payload.Transition.ToStepID != "" {
				return fmt.Errorf("Result does not match terminal Flow Route")
			}
		} else if payload.Effect != ResultAdvanced || payload.Transition.ToStepID != route.NextStepID {
			return fmt.Errorf("Result does not match Flow Route target")
		}
		return nil
	}
	if payload.Transition.Direction != TransitionLoop || payload.Effect != ResultLooped {
		return fmt.Errorf("Result matches no Flow Route")
	}
	loopMatches := matchingLoopRoutes(definition.Loops[payload.Transition.FromStepID], conditionIDs, payload.Transition.ToStepID)
	if len(loopMatches) != 1 {
		return fmt.Errorf("Result does not select exactly one Loop Route")
	}
	return nil
}

func registeredOutputs(definition workflow.Workflow, stepID string, results []ConditionResult) (map[string]RegisteredOutput, error) {
	if _, _, ok := definition.FindStep(stepID); !ok {
		return nil, fmt.Errorf("Result references an unknown Step")
	}
	relevant := make(map[string]bool)
	for _, id := range definition.RelevantConditionIDs(stepID) {
		relevant[id] = true
	}
	accepted := make(map[string]RegisteredOutput, len(results))
	exclusive := map[string]bool{}
	for _, result := range results {
		condition, ok := definition.Condition(result.ConditionID)
		if !ok || !relevant[result.ConditionID] {
			return nil, fmt.Errorf("Condition %q is not available at Step %q", result.ConditionID, stepID)
		}
		if condition.ExclusiveGroup != "" {
			if exclusive[condition.ExclusiveGroup] {
				return nil, fmt.Errorf("Condition exclusive group %q conflicts", condition.ExclusiveGroup)
			}
			exclusive[condition.ExclusiveGroup] = true
		}
		if result.Output.Type != condition.Output.Type || workflow.ValidateOutput(condition.Output, result.Output.Value) != nil {
			return nil, fmt.Errorf("Condition %q has invalid Output", result.ConditionID)
		}
		if _, exists := accepted[condition.Output.Key]; exists {
			return nil, fmt.Errorf("multiple Conditions produce Output %q", condition.Output.Key)
		}
		accepted[condition.Output.Key] = RegisteredOutput{
			Type: condition.Output.Type, Value: cloneRaw(result.Output.Value), ProducerStepID: stepID,
		}
	}
	return accepted, nil
}

func invalidatedOutputs(definition workflow.Workflow, target string, outputs map[string]RegisteredOutput) ([]string, error) {
	_, targetPosition, ok := definition.FindStep(target)
	if !ok {
		return nil, fmt.Errorf("Loop target is unknown")
	}
	result := make([]string, 0)
	for key, output := range outputs {
		_, producerPosition, ok := definition.FindStep(output.ProducerStepID)
		if !ok {
			return nil, fmt.Errorf("Output %q has an unknown producer", key)
		}
		if producerPosition >= targetPosition {
			result = append(result, key)
		}
	}
	sort.Strings(result)
	return result, nil
}

func matchingFlowRoutes(routes []workflow.FlowRoute, conditions map[string]bool) []workflow.FlowRoute {
	result := make([]workflow.FlowRoute, 0)
	for _, route := range routes {
		if route.When.Matches(conditions) {
			result = append(result, route)
		}
	}
	return result
}

func matchingLoopRoutes(routes []workflow.LoopRoute, conditions map[string]bool, target string) []workflow.LoopRoute {
	result := make([]workflow.LoopRoute, 0)
	for _, route := range routes {
		if route.BackStepID == target && route.When.Matches(conditions) {
			result = append(result, route)
		}
	}
	return result
}

func StateRef(value State) StepState {
	if value.CurrentStepID == nil {
		return StepState{}
	}
	return StepState{StepID: *value.CurrentStepID, Status: value.CurrentStepStatus}
}

func ValidTraceDocumentURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return false
	}
	host := strings.ToLower(parsed.Host)
	return (strings.HasSuffix(host, ".larkoffice.com") || strings.HasSuffix(host, ".feishu.cn") || strings.HasSuffix(host, ".larksuite.com")) &&
		(strings.Contains(parsed.Path, "/docx/") || strings.Contains(parsed.Path, "/wiki/"))
}

func SameTraceDocumentURL(left, right string) bool {
	if left == "" || right == "" {
		return false
	}
	leftURL, leftErr := url.Parse(left)
	rightURL, rightErr := url.Parse(right)
	leftKey, rightKey := TraceDocumentKey(left), TraceDocumentKey(right)
	return leftErr == nil && rightErr == nil && leftKey != "" && leftKey == rightKey &&
		strings.EqualFold(strings.TrimSuffix(leftURL.Hostname(), "."), strings.TrimSuffix(rightURL.Hostname(), "."))
}

// TraceDocumentKey returns the stable path identity used by Trace bindings and Registry rows.
func TraceDocumentKey(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	parts := strings.FieldsFunc(parsed.Path, func(r rune) bool { return r == '/' })
	for index, part := range parts {
		if (part == "wiki" || part == "docx") && index+1 < len(parts) {
			return part + ":" + parts[index+1]
		}
	}
	return ""
}

func Payload(value any) any { return value }

func EventPayloadAs[T any](event Event) (T, bool) {
	if value, ok := event.Payload.(T); ok {
		return value, true
	}
	if value, ok := event.Payload.(*T); ok && value != nil {
		return *value, true
	}
	var zero T
	return zero, false
}

func NewEventID() string {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err == nil {
		return hex.EncodeToString(random)
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func validateOutputValue(value OutputValue) error {
	if value.Type == workflow.OutputEnum {
		return workflow.ValidateOutput(workflow.OutputDefinition{Key: "value", Type: workflow.OutputString}, value.Value)
	}
	return workflow.ValidateOutput(workflow.OutputDefinition{Key: "value", Type: value.Type}, value.Value)
}

func validStepStatus(value StepStatus) bool {
	return value == StepReady || value == StepInProgress || value == StepFixing || value == StepBlocked
}

func validEvidence(values []Evidence) bool {
	for _, value := range values {
		if value.Source != EvidenceHuman && value.Source != EvidenceSystem && value.Source != EvidenceAI &&
			value.Source != EvidenceFile && value.Source != EvidenceURL || strings.TrimSpace(value.Content) == "" {
			return false
		}
	}
	return true
}

func uniqueNonEmpty(values []string) bool {
	seen := map[string]bool{}
	for _, value := range values {
		if value == "" || seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}

func sortedCopy(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func sortedKeys[V any](values map[string]V) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func sameCursor(cursor *StepState, stepID string, status StepStatus) bool {
	return cursor != nil && cursor.StepID == stepID && cursor.Status == status
}

func stepStatePointer(value StepState) *StepState {
	if value.StepID == "" && value.Status == "" {
		return nil
	}
	copy := value
	return &copy
}

func equalOptionalStepState(left, right *StepState) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func equalRegisteredOutput(left, right RegisteredOutput) bool {
	return left.Type == right.Type && left.ProducerStepID == right.ProducerStepID && equalJSON(left.Value, right.Value)
}

func cloneEvidence(values []Evidence) []Evidence {
	return append([]Evidence(nil), values...)
}

func cloneRaw(value json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), value...)
}

func equalJSON(left, right json.RawMessage) bool {
	var leftCompact, rightCompact bytes.Buffer
	return json.Compact(&leftCompact, left) == nil && json.Compact(&rightCompact, right) == nil &&
		bytes.Equal(leftCompact.Bytes(), rightCompact.Bytes())
}
