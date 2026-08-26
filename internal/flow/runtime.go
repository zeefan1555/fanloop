package flow

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/zeefan1555/fanloop/internal/buildinfo"
	cardruntime "github.com/zeefan1555/fanloop/internal/card"
	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/idl/flowidl"
	"github.com/zeefan1555/fanloop/internal/idl/traceidl"
	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/store"
	"github.com/zeefan1555/fanloop/internal/trace"
	"github.com/zeefan1555/fanloop/internal/workflow"
	"github.com/zeefan1555/fanloop/internal/workflowview"
)

type Runtime struct {
	Clock           func() time.Time
	EventID         func() string
	ReleaseVersion  string
	DeliverPanorama func(context.Context, string, string) string
	ProjectCard     func(string, state.State) error
	ProvisionTrace  func(context.Context, string, string) string
	AutoSyncTrace   func(context.Context, string)
	Warn            func(string)
}

var _ flowidl.FlowService = Runtime{}

func DefaultRuntime() Runtime {
	return Runtime{
		Clock: time.Now, EventID: state.NewEventID, ReleaseVersion: buildinfo.ReleaseVersion,
		DeliverPanorama: cardruntime.AttemptPanoramaDelivery,
		ProjectCard:     cardruntime.WriteProjection,
		ProvisionTrace:  provisionTraceDocument,
		AutoSyncTrace: func(ctx context.Context, root string) {
			_, _ = trace.DefaultRuntime().Sync(ctx, root, traceidl.NewTraceSyncRequest(), false)
		},
	}
}

func (runtime Runtime) Init(ctx context.Context, root string, request *flowidl.FlowInitRequest, dryRun bool) (*flowidl.FlowInitResponse, error) {
	if request == nil {
		return nil, requestError(errText("request is required"))
	}
	if err := request.IsValid(); err != nil {
		return nil, requestError(err)
	}
	if request.Requirement.SourceUrl != nil && !validHTTPURL(*request.Requirement.SourceUrl) {
		return nil, requestError(errText("requirement.source_url must be an HTTP URL"))
	}
	local, failure := store.New(root)
	if failure != nil {
		return nil, storeError(failure)
	}
	if local.Exists() {
		return nil, newFlowError(erroridl.ErrorCode_ALREADY_INITIALIZED, "requirement is already initialized", nil)
	}
	loaded, flowFailure := loadWorkflow(request.Workflow)
	if flowFailure != nil {
		return nil, flowFailure
	}
	first, ok := loaded.Workflow.FirstStepID()
	if !ok {
		return nil, newFlowError(erroridl.ErrorCode_WORKFLOW_INVALID, "workflow has no steps", nil)
	}
	now := runtime.now()
	next := state.State{
		SchemaVersion: state.CurrentStateSchemaVersion,
		Requirement: state.Requirement{
			Title: request.Requirement.Title, SourceURL: request.Requirement.GetSourceUrl(),
		},
		Release:            state.Release{Version: runtime.releaseVersion(), Workflow: state.WorkflowRefFrom(loaded.Ref)},
		CurrentStepID:      &first,
		CurrentStepStatus:  state.StepReady,
		CurrentStepSummary: "workflow initialized",
		Outputs:            map[string]state.RegisteredOutput{},
		Integrations:       state.Integrations{},
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	response := &flowidl.FlowInitResponse{
		Effect: flowidl.InitEffect_initialized, Workflow: workflowview.WorkflowRef(loaded.Ref), State: workflowview.Project(loaded.Workflow, next),
	}
	if err := response.IsValid(); err != nil {
		return nil, internalError(err)
	}
	if dryRun {
		return response, nil
	}
	eventID := runtime.eventID()
	next.LastEventID = eventID
	event := state.Event{
		SchemaVersion: state.CurrentEventSchemaVersion, ID: eventID, OccurredAt: now,
		Kind: state.EventFlowInitialized, Command: "flow.init", Workflow: state.WorkflowRefFrom(loaded.Ref),
		Payload: state.Payload(state.FlowInitializedPayload{StepID: first, StepStatus: state.StepReady}),
	}
	if failure := validateAndCommit(local, loaded.Workflow, next, event); failure != nil {
		return nil, failure
	}
	traceWarning := runtime.provisionTrace(ctx, root, request.Requirement.Title)
	runtime.warn(traceWarning)
	if traceWarning != "" {
		return response, nil
	}
	current, _, loadFailure := local.LoadBound()
	if loadFailure != nil {
		runtime.warn(fmt.Sprintf("Card projection for Flow Event %s could not load current State: %v", event.ID, loadFailure))
		return response, nil
	}
	projected := runtime.projectCard(root, current, event.ID)
	runtime.warn(projected)
	if projected == "" {
		runtime.warn(runtime.deliverPanorama(ctx, root, event.ID))
	}
	return response, nil
}

func (runtime Runtime) Status(_ context.Context, root string, request *flowidl.FlowStatusRequest) (*flowidl.FlowStatusResponse, error) {
	if request == nil {
		return nil, requestError(errText("request is required"))
	}
	if err := request.IsValid(); err != nil {
		return nil, requestError(err)
	}
	_, current, loaded, failure := load(root)
	if failure != nil {
		return nil, failure
	}
	response := &flowidl.FlowStatusResponse{
		Requirement: workflowview.Requirement(current.Requirement), Workflow: workflowview.WorkflowRef(loaded.Ref), State: workflowview.Project(loaded.Workflow, current),
	}
	if err := response.IsValid(); err != nil {
		return nil, internalError(err)
	}
	return response, nil
}

func (runtime Runtime) Progress(ctx context.Context, root string, request *flowidl.FlowProgressRequest, dryRun bool) (*flowidl.FlowProgressResponse, error) {
	if err := validateProgressRequest(request); err != nil {
		return nil, err
	}
	local, current, loaded, failure := load(root)
	if failure != nil {
		return nil, failure
	}
	if failure := requireCurrentStep(current, request.StepId); failure != nil {
		return nil, failure
	}
	from := state.StateRef(current)
	current.CurrentStepStatus = durableProgressStatus(request.Status)
	current.CurrentStepSummary = request.Summary
	current.CurrentEvidence = durableEvidence(request.Evidence)
	current.UpdatedAt = runtime.now()
	response := &flowidl.FlowProgressResponse{Effect: flowidl.ProgressEffect_status_updated, State: workflowview.Project(loaded.Workflow, current)}
	if err := response.IsValid(); err != nil {
		return nil, internalError(err)
	}
	if dryRun {
		return response, nil
	}
	previousEventID := current.LastEventID
	eventID := runtime.eventID()
	current.LastEventID = eventID
	event := state.Event{
		SchemaVersion: state.CurrentEventSchemaVersion, ID: eventID, OccurredAt: current.UpdatedAt,
		Kind: state.EventFlowProgressed, Command: "flow.report.progress",
		Workflow: current.Release.Workflow, CausedByEventID: previousEventID,
		Payload: state.Payload(state.FlowProgressPayload{
			FromStepID: from.StepID, FromStepStatus: from.Status, ToStepStatus: current.CurrentStepStatus,
			Summary: request.Summary, Evidence: durableEvidence(request.Evidence),
		}),
	}
	if failure := validateAndCommit(local, loaded.Workflow, current, event); failure != nil {
		return nil, failure
	}
	runtime.afterAcceptedReport(ctx, root, current, event.ID)
	return response, nil
}

func (runtime Runtime) Result(ctx context.Context, root string, request *flowidl.FlowResultRequest, dryRun bool) (*flowidl.FlowResultResponse, error) {
	if failure := validateResultRequest(request); failure != nil {
		return nil, failure
	}
	local, current, loaded, failure := load(root)
	if failure != nil {
		return nil, failure
	}
	if failure := requireCurrentStep(current, request.StepId); failure != nil {
		return nil, failure
	}
	evaluation, resultFailure := evaluateResult(loaded.Workflow, current, request)
	if resultFailure != nil {
		return nil, resultFailure
	}
	previousEventID := current.LastEventID
	for key, output := range evaluation.accepted {
		current.Outputs[key] = output
	}
	for _, key := range evaluation.invalidated {
		delete(current.Outputs, key)
	}
	current.UpdatedAt = runtime.now()
	switch evaluation.effect {
	case flowidl.ResultEffect_completed:
		current.CurrentStepID = nil
		current.CurrentStepStatus = ""
		current.CurrentStepSummary = ""
		current.CurrentEvidence = nil
	default:
		current.CurrentStepID = stringPointer(evaluation.transition.GetToStepId())
		current.CurrentStepStatus = state.StepReady
		current.CurrentStepSummary = request.Summary
		current.CurrentEvidence = durableEvidence(request.Evidence)
	}
	response := &flowidl.FlowResultResponse{
		Effect: evaluation.effect, Transition: evaluation.transition, State: workflowview.Project(loaded.Workflow, current),
		InvalidatedOutputs: append([]string{}, evaluation.invalidated...),
	}
	if !dryRun {
		eventID := runtime.eventID()
		response.EventId = stringPointer(eventID)
		current.LastEventID = eventID
	}
	if err := response.IsValid(); err != nil {
		return nil, internalError(err)
	}
	if dryRun {
		return response, nil
	}
	event := state.Event{
		SchemaVersion: state.CurrentEventSchemaVersion, ID: response.GetEventId(), OccurredAt: current.UpdatedAt,
		Kind: state.EventFlowResult, Command: "flow.report.result", Workflow: current.Release.Workflow, CausedByEventID: previousEventID,
		Payload: state.Payload(state.FlowResultPayload{
			ConditionResults: evaluation.conditionResults, Evidence: durableEvidence(request.Evidence), Summary: request.Summary,
			Effect: evaluation.durableEffect, Transition: evaluation.durableTransition,
			OutputChanges: state.OutputChanges{Accepted: sortedOutputKeys(evaluation.accepted), Invalidated: append([]string(nil), evaluation.invalidated...)},
		}),
	}
	if failure := validateAndCommit(local, loaded.Workflow, current, event); failure != nil {
		return nil, failure
	}
	runtime.afterAcceptedReport(ctx, root, current, event.ID)
	return response, nil
}

func validateProgressRequest(request *flowidl.FlowProgressRequest) *erroridl.PublicError {
	if request == nil {
		return requestError(errText("request is required"))
	}
	if err := request.IsValid(); err != nil {
		return requestError(err)
	}
	if request.Status != flowidl.ProgressStatus_in_progress && request.Status != flowidl.ProgressStatus_fixing && request.Status != flowidl.ProgressStatus_blocked {
		return requestError(errText("status must be in_progress, fixing, or blocked"))
	}
	return validateEvidence(request.Evidence)
}

func validateEvidence(values []*flowidl.Evidence) *erroridl.PublicError {
	for _, value := range values {
		if value == nil {
			return requestError(errText("evidence item is required"))
		}
		if err := value.IsValid(); err != nil {
			return requestError(err)
		}
		if value.Source == flowidl.EvidenceSource_url && !validHTTPURL(value.Content) {
			return requestError(errText("url Evidence content must be an HTTP URL"))
		}
	}
	return nil
}

func requireCurrentStep(current state.State, stepID string) *erroridl.PublicError {
	if current.CurrentStepID == nil {
		return newFlowError(erroridl.ErrorCode_REPORT_NOT_ALLOWED, "completed Workflow does not accept reports", nil)
	}
	if stepID != *current.CurrentStepID {
		return newFlowError(erroridl.ErrorCode_STEP_NOT_CURRENT, "step_id is not the current Step", map[string]string{"current_step_id": *current.CurrentStepID})
	}
	return nil
}

func durableProgressStatus(value flowidl.ProgressStatus) state.StepStatus {
	switch value {
	case flowidl.ProgressStatus_fixing:
		return state.StepFixing
	case flowidl.ProgressStatus_blocked:
		return state.StepBlocked
	default:
		return state.StepInProgress
	}
}

func durableEvidence(values []*flowidl.Evidence) []state.Evidence {
	result := make([]state.Evidence, 0, len(values))
	for _, value := range values {
		result = append(result, state.Evidence{Source: durableEvidenceSource(value.Source), Content: value.Content, Ref: value.GetRef()})
	}
	return result
}

func durableEvidenceSource(value flowidl.EvidenceSource) state.EvidenceSource {
	switch value {
	case flowidl.EvidenceSource_human:
		return state.EvidenceHuman
	case flowidl.EvidenceSource_system:
		return state.EvidenceSystem
	case flowidl.EvidenceSource_ai:
		return state.EvidenceAI
	case flowidl.EvidenceSource_file:
		return state.EvidenceFile
	default:
		return state.EvidenceURL
	}
}

func validateAndCommit(local *store.Store, definition workflow.Workflow, next state.State, event state.Event) *erroridl.PublicError {
	if err := next.Validate(); err != nil {
		return internalError(err)
	}
	if err := next.ValidateAgainst(definition); err != nil {
		return internalError(err)
	}
	return storeError(local.Commit(next, event))
}

func load(root string) (*store.Store, state.State, workflow.Loaded, *erroridl.PublicError) {
	local, failure := store.New(root)
	if failure != nil {
		return nil, state.State{}, workflow.Loaded{}, storeError(failure)
	}
	current, loaded, failure := local.LoadBound()
	if failure != nil {
		return nil, state.State{}, workflow.Loaded{}, storeError(failure)
	}
	return local, current, loaded, nil
}

func internalError(err error) *erroridl.PublicError {
	return newFlowError(erroridl.ErrorCode_INTERNAL, err.Error(), nil)
}

func validHTTPURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func (runtime Runtime) afterAcceptedReport(ctx context.Context, root string, current state.State, eventID string) {
	projected := runtime.projectCard(root, current, eventID)
	runtime.warn(projected)
	if projected == "" {
		runtime.warn(runtime.deliverPanorama(ctx, root, eventID))
	}
	if current.Integrations.Trace != nil && runtime.AutoSyncTrace != nil {
		runtime.AutoSyncTrace(ctx, root)
	}
}

func (runtime Runtime) projectCard(root string, current state.State, eventID string) string {
	if runtime.ProjectCard == nil {
		return ""
	}
	if err := runtime.ProjectCard(root, current); err != nil {
		return fmt.Sprintf("Card projection for Flow Event %s could not update: %v", eventID, err)
	}
	return ""
}

func (runtime Runtime) deliverPanorama(ctx context.Context, root, eventID string) string {
	if runtime.DeliverPanorama == nil {
		return ""
	}
	return runtime.DeliverPanorama(ctx, root, eventID)
}

func (runtime Runtime) provisionTrace(ctx context.Context, root, title string) string {
	if runtime.ProvisionTrace == nil {
		return ""
	}
	return runtime.ProvisionTrace(ctx, root, title)
}

func (runtime Runtime) warn(value string) {
	if value != "" && runtime.Warn != nil {
		runtime.Warn(value)
	}
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

func (runtime Runtime) releaseVersion() string {
	if runtime.ReleaseVersion == "" {
		return buildinfo.ReleaseVersion
	}
	return runtime.ReleaseVersion
}

func stringPointer(value string) *string { return &value }

type errText string

func (value errText) Error() string { return string(value) }
