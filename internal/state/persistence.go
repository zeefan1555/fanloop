package state

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/zeefan1555/commonloop/internal/idl/commonidl"
	"github.com/zeefan1555/commonloop/internal/idl/storageidl"
	"github.com/zeefan1555/commonloop/internal/traceconfig"
	"github.com/zeefan1555/commonloop/internal/workflow"
)

func Encode(value State) ([]byte, error) {
	stored, err := ToStorageIDL(value)
	if err != nil {
		return nil, err
	}
	return marshalIndented(stored)
}

func Decode(content []byte, outputs map[string]RegisteredOutput) (State, error) {
	var stored storageidl.FlowState
	if err := decodeStorageJSON(content, &stored); err != nil {
		return State{}, err
	}
	return FromStorageIDL(&stored, outputs)
}

func ToStorageIDL(value State) (*storageidl.FlowState, error) {
	if err := value.Validate(); err != nil {
		return nil, err
	}
	stored, err := flowStateToIDL(value)
	if err != nil {
		return nil, err
	}
	if err := stored.IsValid(); err != nil {
		return nil, err
	}
	return stored, nil
}

func FromStorageIDL(stored *storageidl.FlowState, outputs map[string]RegisteredOutput) (State, error) {
	if stored == nil {
		return State{}, fmt.Errorf("Flow State is required")
	}
	if stored.SchemaVersion != storageidl.FLOW_STATE_SCHEMA_VERSION {
		return State{}, fmt.Errorf("%w: %d", ErrSchemaUnsupported, stored.SchemaVersion)
	}
	if err := stored.IsValid(); err != nil {
		return State{}, err
	}
	value, err := flowStateFromIDL(stored)
	if err != nil {
		return State{}, err
	}
	value.Outputs = cloneRegisteredOutputs(outputs)
	if err := value.Validate(); err != nil {
		return State{}, err
	}
	return value, nil
}

func EncodeOutputRegistry(value OutputRegistry) ([]byte, error) {
	if err := value.Validate(); err != nil {
		return nil, err
	}
	outputs, err := OutputsToStorageIDL(value.Outputs)
	if err != nil {
		return nil, err
	}
	stored := &storageidl.OutputRegistry{
		SchemaVersion: int32(value.SchemaVersion),
		Workflow:      workflowRefToIDL(value.Workflow),
		Outputs:       outputs,
	}
	if err := stored.IsValid(); err != nil {
		return nil, err
	}
	return marshalIndented(stored)
}

func DecodeOutputRegistry(content []byte) (OutputRegistry, error) {
	var stored storageidl.OutputRegistry
	if err := decodeStorageJSON(content, &stored); err != nil {
		return OutputRegistry{}, err
	}
	if stored.SchemaVersion != storageidl.OUTPUT_REGISTRY_SCHEMA_VERSION {
		return OutputRegistry{}, fmt.Errorf("unsupported Output Registry schema_version %d", stored.SchemaVersion)
	}
	if err := stored.IsValid(); err != nil {
		return OutputRegistry{}, err
	}
	outputs, err := OutputsFromStorageIDL(stored.Outputs)
	if err != nil {
		return OutputRegistry{}, err
	}
	value := OutputRegistry{
		SchemaVersion: int(stored.SchemaVersion),
		Workflow:      workflowRefFromIDL(stored.Workflow),
		Outputs:       outputs,
	}
	if err := value.Validate(); err != nil {
		return OutputRegistry{}, err
	}
	return value, nil
}

func EncodeEvent(value Event) ([]byte, error) {
	if err := validateEventEnvelope(value); err != nil {
		return nil, err
	}
	payload, err := eventPayloadToIDL(value)
	if err != nil {
		return nil, err
	}
	kind, err := storageidl.EventKindFromString(strings.Replace(string(value.Kind), ".", "_", 1))
	if err != nil {
		return nil, err
	}
	stored := &storageidl.Event{
		SchemaVersion: int32(value.SchemaVersion),
		EventId:       value.ID,
		OccurredAt:    formatStorageTime(value.OccurredAt),
		Kind:          kind,
		Command:       value.Command,
		Workflow:      workflowRefToIDL(value.Workflow),
		Payload:       payload,
	}
	if value.CausedByEventID != "" {
		stored.CausedByEventId = stringPointer(value.CausedByEventID)
	}
	if stored.Payload.CountSetFieldsEventPayload() != 1 {
		return nil, fmt.Errorf("Event payload must contain exactly one union member")
	}
	if err := stored.IsValid(); err != nil {
		return nil, err
	}
	return json.Marshal(stored)
}

func DecodeEvent(content []byte) (Event, error) {
	var stored storageidl.Event
	if err := decodeStorageJSON(content, &stored); err != nil {
		return Event{}, err
	}
	if stored.SchemaVersion != storageidl.EVENT_SCHEMA_VERSION {
		return Event{}, fmt.Errorf("unsupported Event schema_version %d", stored.SchemaVersion)
	}
	if err := stored.IsValid(); err != nil {
		return Event{}, err
	}
	if stored.Payload == nil || stored.Payload.CountSetFieldsEventPayload() != 1 {
		return Event{}, fmt.Errorf("Event payload must contain exactly one union member")
	}
	occurredAt, err := parseStorageTime(stored.OccurredAt)
	if err != nil {
		return Event{}, fmt.Errorf("invalid Event occurred_at: %w", err)
	}
	kind := EventKind(strings.Replace(stored.Kind.String(), "_", ".", 1))
	value := Event{
		SchemaVersion:   int(stored.SchemaVersion),
		ID:              stored.EventId,
		OccurredAt:      occurredAt,
		Kind:            kind,
		Command:         stored.Command,
		Workflow:        workflowRefFromIDL(stored.Workflow),
		CausedByEventID: stored.GetCausedByEventId(),
	}
	value.Payload, err = eventPayloadFromIDL(kind, stored.Payload)
	if err != nil {
		return Event{}, err
	}
	if err := validateEventEnvelope(value); err != nil {
		return Event{}, err
	}
	return value, nil
}

func flowStateToIDL(value State) (*storageidl.FlowState, error) {
	stored := &storageidl.FlowState{
		SchemaVersion: int32(value.SchemaVersion),
		Requirement:   requirementToIDL(value.Requirement),
		Release:       releaseToIDL(value.Release),
		Integrations:  &storageidl.Integrations{},
		LastEventId:   value.LastEventID,
		CreatedAt:     formatStorageTime(value.CreatedAt),
		UpdatedAt:     formatStorageTime(value.UpdatedAt),
	}
	if value.CurrentStepID != nil {
		status, err := storageidl.StepStatusFromString(string(value.CurrentStepStatus))
		if err != nil {
			return nil, err
		}
		stored.CurrentStepId = cloneString(value.CurrentStepID)
		stored.CurrentStepStatus = &status
	}
	if value.CurrentStepSummary != "" {
		stored.CurrentStepSummary = stringPointer(value.CurrentStepSummary)
	}
	if len(value.CurrentEvidence) > 0 {
		evidence, err := evidenceToIDL(value.CurrentEvidence)
		if err != nil {
			return nil, err
		}
		stored.CurrentEvidence = evidence
	}
	if value.Integrations.Trace != nil {
		registry, err := storageidl.RegistryProfileFromString(string(value.Integrations.Trace.Registry))
		if err != nil {
			return nil, err
		}
		stored.Integrations.Trace = &storageidl.TraceBinding{
			DocumentUrl: value.Integrations.Trace.DocumentURL,
			Registry:    registry,
		}
		if value.Integrations.Trace.CLILogDocumentURL != "" {
			stored.Integrations.Trace.CliLogDocumentUrl = stringPointer(value.Integrations.Trace.CLILogDocumentURL)
		}
	}
	return stored, nil
}

func flowStateFromIDL(stored *storageidl.FlowState) (State, error) {
	createdAt, err := parseStorageTime(stored.CreatedAt)
	if err != nil {
		return State{}, fmt.Errorf("invalid State created_at: %w", err)
	}
	updatedAt, err := parseStorageTime(stored.UpdatedAt)
	if err != nil {
		return State{}, fmt.Errorf("invalid State updated_at: %w", err)
	}
	value := State{
		SchemaVersion:      int(stored.SchemaVersion),
		Requirement:        requirementFromIDL(stored.Requirement),
		Release:            releaseFromIDL(stored.Release),
		CurrentStepID:      cloneString(stored.CurrentStepId),
		CurrentStepSummary: stored.GetCurrentStepSummary(),
		Integrations:       Integrations{},
		LastEventID:        stored.LastEventId,
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
	}
	if stored.CurrentStepStatus != nil {
		value.CurrentStepStatus = StepStatus(stored.CurrentStepStatus.String())
	}
	value.CurrentEvidence, err = evidenceFromIDL(stored.CurrentEvidence)
	if err != nil {
		return State{}, err
	}
	if stored.Integrations != nil && stored.Integrations.Trace != nil {
		value.Integrations.Trace = &TraceBinding{
			DocumentURL:       stored.Integrations.Trace.DocumentUrl,
			Registry:          traceconfig.RegistryProfile(stored.Integrations.Trace.Registry.String()),
			CLILogDocumentURL: stored.Integrations.Trace.GetCliLogDocumentUrl(),
		}
	}
	return value, nil
}

func eventPayloadToIDL(event Event) (*storageidl.EventPayload, error) {
	payload := &storageidl.EventPayload{}
	switch event.Kind {
	case EventFlowInitialized:
		value, ok := EventPayloadAs[FlowInitializedPayload](event)
		if !ok {
			return nil, fmt.Errorf("invalid %s payload type", event.Kind)
		}
		status, err := storageidl.StepStatusFromString(string(value.StepStatus))
		if err != nil {
			return nil, err
		}
		payload.FlowInitialized = &storageidl.FlowInitializedPayload{StepId: value.StepID, StepStatus: status}
	case EventFlowProgressed:
		value, ok := EventPayloadAs[FlowProgressPayload](event)
		if !ok {
			return nil, fmt.Errorf("invalid %s payload type", event.Kind)
		}
		fromStatus, err := storageidl.StepStatusFromString(string(value.FromStepStatus))
		if err != nil {
			return nil, err
		}
		toStatus, err := storageidl.StepStatusFromString(string(value.ToStepStatus))
		if err != nil {
			return nil, err
		}
		evidence, err := evidenceToIDL(value.Evidence)
		if err != nil {
			return nil, err
		}
		payload.FlowProgressed = &storageidl.FlowProgressedPayload{
			FromStepId: value.FromStepID, FromStepStatus: fromStatus, ToStepStatus: toStatus, Summary: value.Summary,
		}
		if len(evidence) > 0 {
			payload.FlowProgressed.Evidence = evidence
		}
	case EventFlowResult:
		value, ok := EventPayloadAs[FlowResultPayload](event)
		if !ok {
			return nil, fmt.Errorf("invalid %s payload type", event.Kind)
		}
		converted, err := flowResultToIDL(value)
		if err != nil {
			return nil, err
		}
		payload.FlowResult = converted
	case EventTraceDocumentBound:
		value, ok := EventPayloadAs[TraceDocumentBoundPayload](event)
		if !ok {
			return nil, fmt.Errorf("invalid %s payload type", event.Kind)
		}
		registry, err := storageidl.RegistryProfileFromString(string(value.Registry))
		if err != nil {
			return nil, err
		}
		converted := &storageidl.TraceDocumentBoundPayload{DocumentUrl: value.DocumentURL, Registry: registry}
		if value.PreviousDocumentURL != "" {
			converted.PreviousDocumentUrl = stringPointer(value.PreviousDocumentURL)
		}
		if value.PreviousRegistry != "" {
			previous, err := storageidl.RegistryProfileFromString(string(value.PreviousRegistry))
			if err != nil {
				return nil, err
			}
			converted.PreviousRegistry = &previous
		}
		if value.PreviousCLILogDocumentURL != "" {
			converted.PreviousCliLogDocumentUrl = stringPointer(value.PreviousCLILogDocumentURL)
		}
		if value.CLILogDocumentURL != "" {
			converted.CliLogDocumentUrl = stringPointer(value.CLILogDocumentURL)
		}
		payload.TraceDocumentBound = converted
	case EventTraceSyncStarted:
		value, ok := EventPayloadAs[TraceSyncStartedPayload](event)
		if !ok {
			return nil, fmt.Errorf("invalid %s payload type", event.Kind)
		}
		converted := make([]storageidl.TraceTarget, len(value.Targets))
		for index, target := range value.Targets {
			parsed, err := storageidl.TraceTargetFromString(target)
			if err != nil {
				return nil, err
			}
			converted[index] = parsed
		}
		payload.TraceSyncStarted = &storageidl.TraceSyncStartedPayload{Targets: converted}
	case EventTraceSynced:
		value, ok := EventPayloadAs[TraceSyncedPayload](event)
		if !ok {
			return nil, fmt.Errorf("invalid %s payload type", event.Kind)
		}
		converted, err := traceSyncedToIDL(value)
		if err != nil {
			return nil, err
		}
		payload.TraceSynced = converted
	default:
		return nil, fmt.Errorf("unknown event kind %q", event.Kind)
	}
	return payload, nil
}

func eventPayloadFromIDL(kind EventKind, payload *storageidl.EventPayload) (any, error) {
	switch kind {
	case EventFlowInitialized:
		if payload.FlowInitialized == nil {
			return nil, payloadMismatch(kind)
		}
		return FlowInitializedPayload{StepID: payload.FlowInitialized.StepId, StepStatus: StepStatus(payload.FlowInitialized.StepStatus.String())}, nil
	case EventFlowProgressed:
		if payload.FlowProgressed == nil {
			return nil, payloadMismatch(kind)
		}
		evidence, err := evidenceFromIDL(payload.FlowProgressed.Evidence)
		if err != nil {
			return nil, err
		}
		return FlowProgressPayload{
			FromStepID: payload.FlowProgressed.FromStepId, FromStepStatus: StepStatus(payload.FlowProgressed.FromStepStatus.String()),
			ToStepStatus: StepStatus(payload.FlowProgressed.ToStepStatus.String()), Summary: payload.FlowProgressed.Summary, Evidence: evidence,
		}, nil
	case EventFlowResult:
		if payload.FlowResult == nil {
			return nil, payloadMismatch(kind)
		}
		return flowResultFromIDL(payload.FlowResult)
	case EventTraceDocumentBound:
		if payload.TraceDocumentBound == nil {
			return nil, payloadMismatch(kind)
		}
		previousRegistry := traceconfig.RegistryProfile("")
		if payload.TraceDocumentBound.PreviousRegistry != nil {
			previousRegistry = traceconfig.RegistryProfile(payload.TraceDocumentBound.PreviousRegistry.String())
		}
		return TraceDocumentBoundPayload{
			PreviousDocumentURL:       payload.TraceDocumentBound.GetPreviousDocumentUrl(),
			DocumentURL:               payload.TraceDocumentBound.DocumentUrl,
			PreviousRegistry:          previousRegistry,
			Registry:                  traceconfig.RegistryProfile(payload.TraceDocumentBound.Registry.String()),
			PreviousCLILogDocumentURL: payload.TraceDocumentBound.GetPreviousCliLogDocumentUrl(),
			CLILogDocumentURL:         payload.TraceDocumentBound.GetCliLogDocumentUrl(),
		}, nil
	case EventTraceSyncStarted:
		if payload.TraceSyncStarted == nil {
			return nil, payloadMismatch(kind)
		}
		targets := make([]string, len(payload.TraceSyncStarted.Targets))
		for index, target := range payload.TraceSyncStarted.Targets {
			targets[index] = target.String()
		}
		return TraceSyncStartedPayload{Targets: targets}, nil
	case EventTraceSynced:
		if payload.TraceSynced == nil {
			return nil, payloadMismatch(kind)
		}
		return traceSyncedFromIDL(payload.TraceSynced)
	default:
		return nil, fmt.Errorf("unknown event kind %q", kind)
	}
}

func flowResultToIDL(value FlowResultPayload) (*storageidl.FlowResultPayload, error) {
	results := make([]*storageidl.ConditionResult, len(value.ConditionResults))
	for index, result := range value.ConditionResults {
		output, err := outputValueToIDL(result.Output)
		if err != nil {
			return nil, err
		}
		results[index] = &storageidl.ConditionResult{ConditionId: result.ConditionID, Output: output}
	}
	evidence, err := evidenceToIDL(value.Evidence)
	if err != nil {
		return nil, err
	}
	effect, err := storageidl.ResultEffectFromString(string(value.Effect))
	if err != nil {
		return nil, err
	}
	direction, err := storageidl.TransitionDirectionFromString(string(value.Transition.Direction))
	if err != nil {
		return nil, err
	}
	transition := &storageidl.Transition{Direction: direction, FromStepId: value.Transition.FromStepID}
	if value.Transition.ToStepID != "" {
		transition.ToStepId = stringPointer(value.Transition.ToStepID)
	}
	changes := &storageidl.OutputChanges{}
	if len(value.OutputChanges.Accepted) > 0 {
		changes.Accepted = append([]string(nil), value.OutputChanges.Accepted...)
	}
	if len(value.OutputChanges.Invalidated) > 0 {
		changes.Invalidated = append([]string(nil), value.OutputChanges.Invalidated...)
	}
	result := &storageidl.FlowResultPayload{
		ConditionResults: results, Summary: value.Summary, Effect: effect, Transition: transition, OutputChanges: changes,
	}
	if len(evidence) > 0 {
		result.Evidence = evidence
	}
	return result, nil
}

func flowResultFromIDL(value *storageidl.FlowResultPayload) (FlowResultPayload, error) {
	results := make([]ConditionResult, len(value.ConditionResults))
	for index, result := range value.ConditionResults {
		if result == nil || result.Output == nil {
			return FlowResultPayload{}, fmt.Errorf("nil ConditionResult")
		}
		output, err := outputValueFromIDL(result.Output)
		if err != nil {
			return FlowResultPayload{}, err
		}
		results[index] = ConditionResult{ConditionID: result.ConditionId, Output: output}
	}
	evidence, err := evidenceFromIDL(value.Evidence)
	if err != nil {
		return FlowResultPayload{}, err
	}
	if value.Transition == nil || value.OutputChanges == nil {
		return FlowResultPayload{}, fmt.Errorf("Result transition and output_changes are required")
	}
	return FlowResultPayload{
		ConditionResults: results,
		Evidence:         evidence,
		Summary:          value.Summary,
		Effect:           ResultEffect(value.Effect.String()),
		Transition: Transition{
			Direction:  TransitionDirection(value.Transition.Direction.String()),
			FromStepID: value.Transition.FromStepId,
			ToStepID:   value.Transition.GetToStepId(),
		},
		OutputChanges: OutputChanges{
			Accepted:    append([]string(nil), value.OutputChanges.Accepted...),
			Invalidated: append([]string(nil), value.OutputChanges.Invalidated...),
		},
	}, nil
}

func traceSyncedToIDL(value TraceSyncedPayload) (*storageidl.TraceSyncedPayload, error) {
	outcome, err := storageidl.TraceSyncOutcomeFromString(string(value.Outcome))
	if err != nil {
		return nil, err
	}
	targets := make([]*storageidl.TraceTargetResult, len(value.Targets))
	for index, target := range value.Targets {
		name, err := storageidl.TraceTargetFromString(target.Name)
		if err != nil {
			return nil, err
		}
		status, err := storageidl.TraceTargetStatusFromString(target.Status)
		if err != nil {
			return nil, err
		}
		converted := &storageidl.TraceTargetResult{Target: name, Status: status}
		if target.Reason != "" {
			converted.Reason = stringPointer(target.Reason)
		}
		if target.Error != nil {
			code, err := storageidl.TraceTargetErrorCodeFromString(strings.ToLower(target.Error.Code))
			if err != nil {
				return nil, err
			}
			converted.Error = &storageidl.TraceTargetError{Code: code, Message: target.Error.Message, Retryable: target.Error.Retryable}
		}
		targets[index] = converted
	}
	return &storageidl.TraceSyncedPayload{Outcome: outcome, Targets: targets}, nil
}

func traceSyncedFromIDL(value *storageidl.TraceSyncedPayload) (TraceSyncedPayload, error) {
	targets := make([]TraceTargetResult, len(value.Targets))
	for index, target := range value.Targets {
		if target == nil {
			return TraceSyncedPayload{}, fmt.Errorf("nil Trace target")
		}
		converted := TraceTargetResult{Name: target.Target.String(), Status: target.Status.String(), Reason: target.GetReason()}
		if target.Error != nil {
			converted.Error = &TraceTargetError{
				Code: strings.ToUpper(target.Error.Code.String()), Message: target.Error.Message, Retryable: target.Error.Retryable,
			}
		}
		targets[index] = converted
	}
	return TraceSyncedPayload{Outcome: TraceSyncOutcome(value.Outcome.String()), Targets: targets}, nil
}

func requirementToIDL(value Requirement) *storageidl.Requirement {
	result := &storageidl.Requirement{Title: value.Title}
	if value.SourceURL != "" {
		result.SourceUrl = stringPointer(value.SourceURL)
	}
	return result
}

func requirementFromIDL(value *storageidl.Requirement) Requirement {
	if value == nil {
		return Requirement{}
	}
	return Requirement{Title: value.Title, SourceURL: value.GetSourceUrl()}
}

func releaseToIDL(value Release) *storageidl.Release {
	return &storageidl.Release{Version: value.Version, Workflow: workflowRefToIDL(value.Workflow)}
}

func releaseFromIDL(value *storageidl.Release) Release {
	if value == nil {
		return Release{}
	}
	return Release{Version: value.Version, Workflow: workflowRefFromIDL(value.Workflow)}
}

func workflowRefToIDL(value WorkflowRef) *commonidl.WorkflowRef {
	return &commonidl.WorkflowRef{Id: value.ID, Digest: value.Digest}
}

func workflowRefFromIDL(value *commonidl.WorkflowRef) WorkflowRef {
	if value == nil {
		return WorkflowRef{}
	}
	return WorkflowRef{ID: value.Id, Digest: value.Digest}
}

func evidenceToIDL(values []Evidence) ([]*storageidl.Evidence, error) {
	result := make([]*storageidl.Evidence, len(values))
	for index, value := range values {
		source, err := storageidl.EvidenceSourceFromString(string(value.Source))
		if err != nil {
			return nil, err
		}
		item := &storageidl.Evidence{Source: source, Content: value.Content}
		if value.Ref != "" {
			item.Ref = stringPointer(value.Ref)
		}
		result[index] = item
	}
	return result, nil
}

func evidenceFromIDL(values []*storageidl.Evidence) ([]Evidence, error) {
	if len(values) == 0 {
		return nil, nil
	}
	result := make([]Evidence, len(values))
	for index, value := range values {
		if value == nil {
			return nil, fmt.Errorf("nil Evidence")
		}
		result[index] = Evidence{Source: EvidenceSource(value.Source.String()), Content: value.Content, Ref: value.GetRef()}
	}
	return result, nil
}

func outputValueToIDL(value OutputValue) (*storageidl.OutputValue, error) {
	outputType, err := storageidl.OutputTypeFromString(value.Type.String())
	if err != nil {
		return nil, err
	}
	jsonValue, err := commonidl.FromJSON(value.Value)
	if err != nil {
		return nil, err
	}
	return &storageidl.OutputValue{Type: outputType, Value: jsonValue}, nil
}

func outputValueFromIDL(value *storageidl.OutputValue) (OutputValue, error) {
	if value == nil || value.Value == nil {
		return OutputValue{}, fmt.Errorf("Output value is required")
	}
	content, err := json.Marshal(value.Value)
	if err != nil {
		return OutputValue{}, err
	}
	outputType, err := workflow.OutputTypeFromString(value.Type.String())
	if err != nil {
		return OutputValue{}, err
	}
	return OutputValue{Type: outputType, Value: content}, nil
}

func OutputsToStorageIDL(values map[string]RegisteredOutput) (map[string]*storageidl.RegisteredOutput, error) {
	result := make(map[string]*storageidl.RegisteredOutput, len(values))
	for key, value := range values {
		output, err := outputValueToIDL(OutputValue{Type: value.Type, Value: value.Value})
		if err != nil {
			return nil, err
		}
		result[key] = &storageidl.RegisteredOutput{Type: output.Type, Value: output.Value, ProducerStepId: value.ProducerStepID}
	}
	return result, nil
}

func OutputsFromStorageIDL(values map[string]*storageidl.RegisteredOutput) (map[string]RegisteredOutput, error) {
	result := make(map[string]RegisteredOutput, len(values))
	for key, value := range values {
		if value == nil {
			return nil, fmt.Errorf("nil registered Output %q", key)
		}
		output, err := outputValueFromIDL(&storageidl.OutputValue{Type: value.Type, Value: value.Value})
		if err != nil {
			return nil, err
		}
		result[key] = RegisteredOutput{Type: output.Type, Value: output.Value, ProducerStepID: value.ProducerStepId}
	}
	return result, nil
}

func validateEventEnvelope(value Event) error {
	if value.SchemaVersion != CurrentEventSchemaVersion || value.ID == "" || value.OccurredAt.IsZero() ||
		value.Kind == "" || value.Command == "" || value.Workflow.ID == "" ||
		value.Workflow.Digest == "" || value.Payload == nil {
		return fmt.Errorf("invalid event envelope")
	}
	return ValidateEventPayload(value)
}

func payloadMismatch(kind EventKind) error {
	return fmt.Errorf("Event kind %q does not match its payload union member", kind)
}

func marshalIndented(value any) ([]byte, error) {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(content, '\n'), nil
}

func decodeStorageJSON(content []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing JSON value")
		}
		return err
	}
	return nil
}

func formatStorageTime(value time.Time) string { return value.Format(time.RFC3339Nano) }

func parseStorageTime(value string) (time.Time, error) { return time.Parse(time.RFC3339Nano, value) }

func stringPointer(value string) *string { return &value }

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
