package trace

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zeefan1555/commonloop/errs"
	"github.com/zeefan1555/commonloop/internal/idl/erroridl"
	"github.com/zeefan1555/commonloop/internal/idl/traceidl"
	"github.com/zeefan1555/commonloop/internal/state"
	"github.com/zeefan1555/commonloop/internal/store"
	"github.com/zeefan1555/commonloop/internal/traceconfig"
	"github.com/zeefan1555/commonloop/internal/workflow"
)

type Runtime struct {
	Clock   func() time.Time
	EventID func() string
}

var _ traceidl.TraceService = Runtime{}

func DefaultRuntime() Runtime { return Runtime{Clock: time.Now, EventID: state.NewEventID} }

func (runtime Runtime) Bind(_ context.Context, root string, request *traceidl.TraceBindRequest, dryRun bool) (*traceidl.TraceBindResponse, error) {
	if request == nil {
		return nil, publicError(erroridl.ErrorCode_INVALID_ARGUMENT, "request is required")
	}
	if err := request.IsValid(); err != nil {
		return nil, publicError(erroridl.ErrorCode_INVALID_ARGUMENT, err.Error())
	}
	local, current, _, failure := load(root)
	if failure != nil {
		return nil, failure
	}
	if !state.ValidTraceDocumentURL(request.DocumentUrl) {
		return nil, publicError(erroridl.ErrorCode_INVALID_ARGUMENT, "document_url must be an HTTP URL")
	}
	registry, ok := traceconfig.Resolve(traceconfig.RegistryProfile(request.GetRegistry()), current.Release.Workflow.ID)
	if !ok {
		return nil, publicError(erroridl.ErrorCode_INVALID_ARGUMENT, "registry must be production or test")
	}
	if protectedDocument(current, request.DocumentUrl) {
		return nil, publicError(erroridl.ErrorCode_PROTECTED_DOCUMENT, "Trace document must not reuse the Requirement source or an existing Workflow Output document")
	}
	cliLogDocumentURL := request.GetCliLogDocumentUrl()
	if registry.RequireCLILogDocument != (cliLogDocumentURL != "") {
		return nil, publicError(erroridl.ErrorCode_INVALID_ARGUMENT, "cli_log_document_url must match the selected Trace Registry policy")
	}
	if cliLogDocumentURL != "" {
		if !state.ValidTraceDocumentURL(cliLogDocumentURL) {
			return nil, publicError(erroridl.ErrorCode_INVALID_ARGUMENT, "cli_log_document_url must be an HTTP URL")
		}
		if state.SameTraceDocumentURL(request.DocumentUrl, cliLogDocumentURL) || protectedDocument(current, cliLogDocumentURL) {
			return nil, publicError(erroridl.ErrorCode_PROTECTED_DOCUMENT, "CLI log document must be distinct from Trace, the Requirement source, and existing Workflow Output documents")
		}
	}
	previous := ""
	previousRegistry := traceconfig.RegistryProfile("")
	previousCLILogDocumentURL := ""
	if current.Integrations.Trace != nil {
		previous = current.Integrations.Trace.DocumentURL
		previousRegistry = current.Integrations.Trace.Registry
		previousCLILogDocumentURL = current.Integrations.Trace.CLILogDocumentURL
	}
	if previous != "" {
		if state.SameTraceDocumentURL(previous, request.DocumentUrl) && previousRegistry == registry.Profile &&
			sameOptionalDocument(previousCLILogDocumentURL, cliLogDocumentURL) {
			return &traceidl.TraceBindResponse{Effect: traceidl.TraceBindEffect_unchanged}, nil
		}
		return nil, publicError(erroridl.ErrorCode_INVALID_ARGUMENT, "this requirement already has a Trace binding")
	}
	previousEventID := current.LastEventID
	current.Integrations.Trace = &state.TraceBinding{
		DocumentURL: request.DocumentUrl, Registry: registry.Profile, CLILogDocumentURL: cliLogDocumentURL,
	}
	current.UpdatedAt = runtime.now()
	response := &traceidl.TraceBindResponse{Effect: traceidl.TraceBindEffect_bound}
	if dryRun {
		return response, nil
	}
	eventID := runtime.eventID()
	current.LastEventID = eventID
	event := state.Event{
		SchemaVersion: state.CurrentEventSchemaVersion, ID: eventID, OccurredAt: current.UpdatedAt, Kind: state.EventTraceDocumentBound,
		Command: "trace.bind", Workflow: current.Release.Workflow, CausedByEventID: previousEventID,
		Payload: state.Payload(state.TraceDocumentBoundPayload{
			PreviousDocumentURL: previous, DocumentURL: request.DocumentUrl,
			PreviousRegistry: previousRegistry, Registry: registry.Profile,
			PreviousCLILogDocumentURL: previousCLILogDocumentURL, CLILogDocumentURL: cliLogDocumentURL,
		}),
	}
	if err := current.Validate(); err != nil {
		return nil, publicError(erroridl.ErrorCode_INTERNAL, err.Error())
	}
	if failure := local.Commit(current, event); failure != nil {
		return nil, failure
	}
	return response, nil
}

func (runtime Runtime) Status(_ context.Context, root string, request *traceidl.TraceStatusRequest) (*traceidl.TraceStatusResponse, error) {
	if request == nil {
		return nil, publicError(erroridl.ErrorCode_INVALID_ARGUMENT, "request is required")
	}
	local, current, _, failure := load(root)
	if failure != nil {
		return nil, failure
	}
	events, failure := local.Events()
	if failure != nil {
		return nil, failure
	}
	lastSync, err := latestSync(events)
	if err != nil {
		return nil, publicError(erroridl.ErrorCode_STATE_CORRUPT, err.Error())
	}
	response := &traceidl.TraceStatusResponse{EventCount: int32(len(events)), LastSync: lastSync}
	if current.Integrations.Trace != nil {
		response.DocumentUrl = stringPointer(current.Integrations.Trace.DocumentURL)
		if current.Integrations.Trace.CLILogDocumentURL != "" {
			response.CliLogDocumentUrl = stringPointer(current.Integrations.Trace.CLILogDocumentURL)
		}
		if registry, ok := traceconfig.Resolve(current.Integrations.Trace.Registry, current.Release.Workflow.ID); ok {
			response.Registry = &traceidl.TraceRegistry{
				Profile: string(registry.Profile), Url: registry.URL, BaseToken: registry.BaseToken,
				TableId: registry.TableID, ViewId: registry.ViewID,
			}
		}
	}
	return response, nil
}

func (runtime Runtime) Render(_ context.Context, root string, request *traceidl.TraceRenderRequest, dryRun bool) (*traceidl.TraceRenderResponse, error) {
	if request == nil {
		return nil, publicError(erroridl.ErrorCode_INVALID_ARGUMENT, "request is required")
	}
	local, _, _, failure := load(root)
	if failure != nil {
		return nil, failure
	}
	_, count, failure := local.RebuildProjection(dryRun)
	if failure != nil {
		return nil, failure
	}
	return &traceidl.TraceRenderResponse{EventCount: int32(count), ProjectionPath: ".commonloop/trace/events.md"}, nil
}

func load(root string) (*store.Store, state.State, workflow.Loaded, *erroridl.PublicError) {
	local, failure := store.New(root)
	if failure != nil {
		return nil, state.State{}, workflow.Loaded{}, failure
	}
	current, loaded, failure := local.LoadBound()
	if failure != nil {
		return nil, state.State{}, workflow.Loaded{}, failure
	}
	return local, current, loaded, nil
}

func protectedDocument(current state.State, candidate string) bool {
	if state.SameTraceDocumentURL(candidate, current.Requirement.SourceURL) {
		return true
	}
	for _, output := range current.Outputs {
		switch output.Type {
		case workflow.OutputURL:
			var value string
			if json.Unmarshal(output.Value, &value) == nil && state.SameTraceDocumentURL(candidate, value) {
				return true
			}
		case workflow.OutputURLList:
			var values []string
			if json.Unmarshal(output.Value, &values) == nil {
				for _, value := range values {
					if state.SameTraceDocumentURL(candidate, value) {
						return true
					}
				}
			}
		}
	}
	return false
}

func sameOptionalDocument(left, right string) bool {
	if left == "" || right == "" {
		return left == right
	}
	return state.SameTraceDocumentURL(left, right)
}

func latestSync(events []state.Event) (*traceidl.TraceLastSync, error) {
	for index := len(events) - 1; index >= 0; index-- {
		if events[index].Kind != state.EventTraceSynced {
			continue
		}
		payload, ok := state.EventPayloadAs[state.TraceSyncedPayload](events[index])
		if !ok {
			return nil, fmt.Errorf("invalid Trace synced payload")
		}
		outcome, err := traceidl.TraceSyncOutcomeFromString(string(payload.Outcome))
		if err != nil {
			return nil, err
		}
		targets, err := publicTargets(payload.Targets)
		if err != nil {
			return nil, err
		}
		return &traceidl.TraceLastSync{
			OccurredAt: events[index].OccurredAt.Format(time.RFC3339Nano),
			Outcome:    outcome,
			Targets:    targets,
		}, nil
	}
	return nil, nil
}

func publicTargets(values []state.TraceTargetResult) ([]*traceidl.TraceTargetResult, error) {
	result := make([]*traceidl.TraceTargetResult, 0, len(values))
	for _, value := range values {
		target, err := traceidl.TraceTargetFromString(value.Name)
		if err != nil {
			return nil, fmt.Errorf("invalid Trace target %q", value.Name)
		}
		status, err := traceidl.TraceTargetStatusFromString(value.Status)
		if err != nil {
			return nil, fmt.Errorf("invalid Trace target status %q", value.Status)
		}
		item := &traceidl.TraceTargetResult{Target: target, Status: status}
		if value.Reason != "" {
			item.Reason = stringPointer(value.Reason)
		}
		if value.Error != nil {
			code, err := erroridl.ErrorCodeFromString(value.Error.Code)
			if err != nil || code == erroridl.ErrorCode_unspecified {
				return nil, fmt.Errorf("invalid Trace target error code %q", value.Error.Code)
			}
			item.Error = &traceidl.TraceTargetError{Code: code, Message: value.Error.Message, Retryable: value.Error.Retryable}
		}
		result = append(result, item)
	}
	return result, nil
}

func durableTargets(values []*traceidl.TraceTargetResult) []state.TraceTargetResult {
	result := make([]state.TraceTargetResult, 0, len(values))
	for _, value := range values {
		item := state.TraceTargetResult{Name: value.Target.String(), Status: value.Status.String(), Reason: value.GetReason()}
		if value.Error != nil {
			item.Error = &state.TraceTargetError{Code: value.Error.Code.String(), Message: value.Error.Message, Retryable: value.Error.Retryable}
		}
		result = append(result, item)
	}
	return result
}

func publicError(code erroridl.ErrorCode, message string) *erroridl.PublicError {
	return errs.NewCode(code, message, nil)
}

func (runtime Runtime) now() time.Time {
	if runtime.Clock == nil {
		return time.Now().UTC()
	}
	return runtime.Clock().UTC()
}

func (runtime Runtime) eventID() string {
	if runtime.EventID == nil {
		return state.NewEventID()
	}
	return runtime.EventID()
}

func stringPointer(value string) *string { return &value }
