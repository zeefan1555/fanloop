package flow

import (
	"encoding/json"
	"sort"

	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/idl/flowidl"
	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

type resultEvaluation struct {
	conditionResults  []state.ConditionResult
	accepted          map[string]state.RegisteredOutput
	invalidated       []string
	effect            flowidl.ResultEffect
	transition        *flowidl.Transition
	durableEffect     state.ResultEffect
	durableTransition state.Transition
}

func validateResultRequest(request *flowidl.FlowResultRequest) *erroridl.PublicError {
	if request == nil {
		return requestError(errText("request is required"))
	}
	if err := request.IsValid(); err != nil {
		return requestError(err)
	}
	if request.Route.CountSetFieldsRouteSelection() != 1 {
		return requestError(errText("route requires exactly one of next_step_id, back_step_id, or terminal"))
	}
	if request.Route.Terminal != nil && !*request.Route.Terminal {
		return requestError(errText("route.terminal must be true"))
	}
	if failure := validateEvidence(request.Evidence); failure != nil {
		return failure
	}
	for _, result := range request.ConditionResults {
		if result == nil || result.Output == nil || result.Output.Value == nil || result.Output.Value.CountSetFieldsJsonValue() != 1 {
			return requestError(errText("each ConditionResult requires one Output value"))
		}
	}
	return nil
}

func evaluateResult(definition workflow.Workflow, current state.State, request *flowidl.FlowResultRequest) (resultEvaluation, *erroridl.PublicError) {
	relevant := map[string]bool{}
	for _, id := range definition.RelevantConditionIDs(request.StepId) {
		relevant[id] = true
	}
	conditionIDs := map[string]bool{}
	exclusive := map[string]bool{}
	accepted := map[string]state.RegisteredOutput{}
	durableResults := make([]state.ConditionResult, 0, len(request.ConditionResults))
	for _, result := range request.ConditionResults {
		if conditionIDs[result.ConditionId] {
			return resultEvaluation{}, newFlowError(erroridl.ErrorCode_CONDITION_CONFLICT, "ConditionResult is duplicated", map[string]string{"condition_id": result.ConditionId})
		}
		condition, ok := definition.Condition(result.ConditionId)
		if !ok || !relevant[result.ConditionId] {
			return resultEvaluation{}, newFlowError(erroridl.ErrorCode_CONDITION_NOT_ALLOWED, "Condition is not available at the current Step", map[string]string{"condition_id": result.ConditionId})
		}
		if condition.ExclusiveGroup != "" {
			if exclusive[condition.ExclusiveGroup] {
				return resultEvaluation{}, newFlowError(erroridl.ErrorCode_CONDITION_CONFLICT, "Condition exclusive group is reported more than once", map[string]string{"exclusive_group": condition.ExclusiveGroup})
			}
			exclusive[condition.ExclusiveGroup] = true
		}
		outputType, ok := durableOutputType(result.Output.Type)
		if !ok || outputType != condition.Output.Type {
			return resultEvaluation{}, newFlowError(erroridl.ErrorCode_OUTPUT_INVALID, "Condition Output type does not match its definition", map[string]string{"condition_id": result.ConditionId})
		}
		raw, err := json.Marshal(result.Output.Value)
		if err != nil || workflow.ValidateOutput(condition.Output, raw) != nil {
			return resultEvaluation{}, newFlowError(erroridl.ErrorCode_OUTPUT_INVALID, "Condition Output value does not satisfy its definition", map[string]string{"condition_id": result.ConditionId})
		}
		if condition.Output.Source == workflow.OutputSourceTraceDocumentURL {
			var documentURL string
			if current.Integrations.Trace == nil || json.Unmarshal(raw, &documentURL) != nil || documentURL != current.Integrations.Trace.DocumentURL {
				return resultEvaluation{}, newFlowError(erroridl.ErrorCode_OUTPUT_INVALID, "Condition Output value does not match its authoritative source", map[string]string{"condition_id": result.ConditionId, "source": condition.Output.Source})
			}
		}
		if _, exists := accepted[condition.Output.Key]; exists {
			return resultEvaluation{}, newFlowError(erroridl.ErrorCode_CONDITION_CONFLICT, "multiple Conditions produce the same Output key", map[string]string{"output_key": condition.Output.Key})
		}
		conditionIDs[result.ConditionId] = true
		value := append(json.RawMessage(nil), raw...)
		durableResults = append(durableResults, state.ConditionResult{ConditionID: result.ConditionId, Output: state.OutputValue{Type: outputType, Value: value}})
		accepted[condition.Output.Key] = state.RegisteredOutput{Type: outputType, Value: value, ProducerStepID: request.StepId}
	}

	if request.Route.NextStepId != nil || request.Route.Terminal != nil {
		available, flowMatches := matchingFlowRoutes(definition.Flows[request.StepId], request.Route, conditionIDs)
		if !available {
			return resultEvaluation{}, newFlowError(erroridl.ErrorCode_ROUTE_NOT_ALLOWED, "selected Route is not available at the current Step", nil)
		}
		if len(flowMatches) == 0 {
			return resultEvaluation{}, newFlowError(erroridl.ErrorCode_ROUTE_NOT_MATCHED, "Condition results do not match the selected Route", nil)
		}
		if len(flowMatches) > 1 {
			return resultEvaluation{}, newFlowError(erroridl.ErrorCode_ROUTE_AMBIGUOUS, "Condition results match more than one selected Flow Route", nil)
		}
		route := flowMatches[0]
		evaluation := resultEvaluation{conditionResults: durableResults, accepted: accepted}
		if route.Terminal {
			evaluation.effect = flowidl.ResultEffect_completed
			evaluation.transition = &flowidl.Transition{Direction: flowidl.TransitionDirection_flow, FromStepId: request.StepId}
			evaluation.durableEffect = state.ResultCompleted
			evaluation.durableTransition = state.Transition{Direction: state.TransitionFlow, FromStepID: request.StepId}
		} else {
			evaluation.effect = flowidl.ResultEffect_advanced
			evaluation.transition = &flowidl.Transition{Direction: flowidl.TransitionDirection_flow, FromStepId: request.StepId, ToStepId: stringPointer(route.NextStepID)}
			evaluation.durableEffect = state.ResultAdvanced
			evaluation.durableTransition = state.Transition{Direction: state.TransitionFlow, FromStepID: request.StepId, ToStepID: route.NextStepID}
		}
		return evaluation, nil
	}

	available, loopMatches := matchingLoopRoutes(definition.Loops[request.StepId], request.Route.GetBackStepId(), conditionIDs)
	if !available {
		return resultEvaluation{}, newFlowError(erroridl.ErrorCode_ROUTE_NOT_ALLOWED, "selected Route is not available at the current Step", nil)
	}
	if len(loopMatches) == 0 {
		return resultEvaluation{}, newFlowError(erroridl.ErrorCode_ROUTE_NOT_MATCHED, "Condition results do not match the selected Route", nil)
	}
	if len(loopMatches) > 1 {
		return resultEvaluation{}, newFlowError(erroridl.ErrorCode_ROUTE_AMBIGUOUS, "Condition results match more than one selected Loop Route", nil)
	}
	merged := make(map[string]state.RegisteredOutput, len(current.Outputs)+len(accepted))
	for key, output := range current.Outputs {
		merged[key] = output
	}
	for key, output := range accepted {
		merged[key] = output
	}
	backStepID := request.Route.GetBackStepId()
	invalidated, err := outputsFromStep(definition, backStepID, merged)
	if err != nil {
		return resultEvaluation{}, newFlowError(erroridl.ErrorCode_WORKFLOW_INVALID, err.Error(), nil)
	}
	return resultEvaluation{
		conditionResults: durableResults, accepted: accepted, invalidated: invalidated,
		effect:            flowidl.ResultEffect_looped,
		transition:        &flowidl.Transition{Direction: flowidl.TransitionDirection_loop, FromStepId: request.StepId, ToStepId: stringPointer(backStepID)},
		durableEffect:     state.ResultLooped,
		durableTransition: state.Transition{Direction: state.TransitionLoop, FromStepID: request.StepId, ToStepID: backStepID},
	}, nil
}

func matchingFlowRoutes(routes []workflow.FlowRoute, selected *flowidl.RouteSelection, conditions map[string]bool) (bool, []workflow.FlowRoute) {
	available := false
	matches := make([]workflow.FlowRoute, 0)
	for _, route := range routes {
		selectedRoute := selected.Terminal != nil && route.Terminal || selected.NextStepId != nil && !route.Terminal && route.NextStepID == selected.GetNextStepId()
		if !selectedRoute {
			continue
		}
		available = true
		if route.When.Matches(conditions) {
			matches = append(matches, route)
		}
	}
	return available, matches
}

func matchingLoopRoutes(routes []workflow.LoopRoute, backStepID string, conditions map[string]bool) (bool, []workflow.LoopRoute) {
	available := false
	matches := make([]workflow.LoopRoute, 0)
	for _, route := range routes {
		if route.BackStepID != backStepID {
			continue
		}
		available = true
		if route.When.Matches(conditions) {
			matches = append(matches, route)
		}
	}
	return available, matches
}

func outputsFromStep(definition workflow.Workflow, target string, outputs map[string]state.RegisteredOutput) ([]string, error) {
	_, targetPosition, ok := definition.FindStep(target)
	if !ok {
		return nil, errText("Loop target is unknown")
	}
	result := make([]string, 0)
	for key, output := range outputs {
		_, producerPosition, ok := definition.FindStep(output.ProducerStepID)
		if !ok {
			return nil, errText("registered Output has an unknown producer")
		}
		if producerPosition >= targetPosition {
			result = append(result, key)
		}
	}
	sort.Strings(result)
	return result, nil
}

func sortedOutputKeys(values map[string]state.RegisteredOutput) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func durableOutputType(value flowidl.OutputType) (workflow.OutputType, bool) {
	switch value {
	case flowidl.OutputType_string:
		return workflow.OutputString, true
	case flowidl.OutputType_boolean:
		return workflow.OutputBoolean, true
	case flowidl.OutputType_integer:
		return workflow.OutputInteger, true
	case flowidl.OutputType_path:
		return workflow.OutputPath, true
	case flowidl.OutputType_url:
		return workflow.OutputURL, true
	case flowidl.OutputType_url_list:
		return workflow.OutputURLList, true
	case flowidl.OutputType_enum_value:
		return workflow.OutputEnum, true
	case flowidl.OutputType_object:
		return workflow.OutputObject, true
	default:
		return 0, false
	}
}
