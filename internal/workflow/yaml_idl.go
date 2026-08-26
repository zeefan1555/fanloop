package workflow

import (
	"fmt"

	"github.com/zeefan1555/fanloop/internal/idl/yamlidl"
)

func normalizeYAMLOptionalDefaults(flow *yamlidl.FlowDocument, condition *yamlidl.ConditionDocument) {
	for _, routes := range flow.Flow {
		for _, route := range routes {
			if route != nil && route.Terminal != nil && !*route.Terminal {
				route.Terminal = nil
			}
		}
	}
	for _, definition := range condition.Conditions {
		if definition == nil {
			continue
		}
		if definition.ExclusiveGroup != nil && *definition.ExclusiveGroup == "" {
			definition.ExclusiveGroup = nil
		}
		if definition.Output == nil {
			continue
		}
		if definition.Output.Source != nil && *definition.Output.Source == "" {
			definition.Output.Source = nil
		}
		if definition.Output.Description != nil && *definition.Output.Description == "" {
			definition.Output.Description = nil
		}
	}
}

func normalizeYAMLDocuments(
	workflowDocument *yamlidl.WorkflowDocument,
	flowDocument *yamlidl.FlowDocument,
	conditionDocument *yamlidl.ConditionDocument,
	loopDocument *yamlidl.LoopDocument,
	promptDocument *yamlidl.PromptDocument,
) (Workflow, error) {
	stages, err := normalizeStages(workflowDocument.Stages)
	if err != nil {
		return Workflow{}, err
	}
	flows, err := normalizeFlowRoutes(flowDocument.Flow)
	if err != nil {
		return Workflow{}, err
	}
	conditions, err := normalizeConditions(conditionDocument.Conditions)
	if err != nil {
		return Workflow{}, err
	}
	loops, err := normalizeLoopRoutes(loopDocument.Loop)
	if err != nil {
		return Workflow{}, err
	}
	prompts, err := normalizePrompts(promptDocument.Prompts)
	if err != nil {
		return Workflow{}, err
	}
	return Workflow{
		SchemaVersion: workflowDocument.SchemaVersion,
		ID:            workflowDocument.Id,
		Stages:        stages,
		Flows:         flows,
		Conditions:    conditions,
		Loops:         loops,
		Prompts:       prompts,
	}, nil
}

func normalizeStages(values []*yamlidl.Stage) ([]Stage, error) {
	if values == nil {
		return nil, fmt.Errorf("workflow.stages is required")
	}
	result := make([]Stage, len(values))
	for stageIndex, value := range values {
		if value == nil {
			return nil, fmt.Errorf("workflow.stages[%d] is null", stageIndex)
		}
		if err := value.IsValid(); err != nil {
			return nil, fmt.Errorf("workflow.stages[%d]: %w", stageIndex, err)
		}
		if value.Jobs == nil {
			return nil, fmt.Errorf("workflow.stages[%d].jobs is required", stageIndex)
		}
		jobs := make([]Job, len(value.Jobs))
		for jobIndex, jobValue := range value.Jobs {
			if jobValue == nil {
				return nil, fmt.Errorf("workflow.stages[%d].jobs[%d] is null", stageIndex, jobIndex)
			}
			if err := jobValue.IsValid(); err != nil {
				return nil, fmt.Errorf("workflow.stages[%d].jobs[%d]: %w", stageIndex, jobIndex, err)
			}
			if jobValue.Steps == nil {
				return nil, fmt.Errorf("workflow.stages[%d].jobs[%d].steps is required", stageIndex, jobIndex)
			}
			steps := make([]Step, len(jobValue.Steps))
			for stepIndex, stepValue := range jobValue.Steps {
				if stepValue == nil {
					return nil, fmt.Errorf("workflow.stages[%d].jobs[%d].steps[%d] is null", stageIndex, jobIndex, stepIndex)
				}
				if err := stepValue.IsValid(); err != nil {
					return nil, fmt.Errorf("workflow.stages[%d].jobs[%d].steps[%d]: %w", stageIndex, jobIndex, stepIndex, err)
				}
				steps[stepIndex] = Step{ID: stepValue.Id, Name: stepValue.Name, Executor: stepValue.Executor}
			}
			jobs[jobIndex] = Job{ID: jobValue.Id, Name: jobValue.Name, Steps: steps}
		}
		result[stageIndex] = Stage{ID: value.Id, Name: value.Name, Jobs: jobs}
	}
	return result, nil
}

func normalizeFlowRoutes(values map[string][]*yamlidl.FlowRoute) (map[string][]FlowRoute, error) {
	if values == nil {
		return nil, fmt.Errorf("flow.flow is required")
	}
	result := make(map[string][]FlowRoute, len(values))
	for stepID, routes := range values {
		normalized := make([]FlowRoute, len(routes))
		for index, route := range routes {
			if route == nil {
				return nil, fmt.Errorf("flow.%s[%d] is null", stepID, index)
			}
			if err := route.IsValid(); err != nil {
				return nil, fmt.Errorf("flow.%s[%d]: %w", stepID, index, err)
			}
			normalized[index] = FlowRoute{
				PromptRef:  normalizePromptRef(route.PromptRef),
				When:       normalizeWhen(route.When),
				NextStepID: route.GetNextStepId(),
				Terminal:   route.GetTerminal(),
			}
		}
		result[stepID] = normalized
	}
	return result, nil
}

func normalizeLoopRoutes(values map[string][]*yamlidl.LoopRoute) (map[string][]LoopRoute, error) {
	if values == nil {
		return nil, fmt.Errorf("loop.loop is required")
	}
	result := make(map[string][]LoopRoute, len(values))
	for stepID, routes := range values {
		normalized := make([]LoopRoute, len(routes))
		for index, route := range routes {
			if route == nil {
				return nil, fmt.Errorf("loop.%s[%d] is null", stepID, index)
			}
			if err := route.IsValid(); err != nil {
				return nil, fmt.Errorf("loop.%s[%d]: %w", stepID, index, err)
			}
			normalized[index] = LoopRoute{
				PromptRef:  normalizePromptRef(route.PromptRef),
				When:       normalizeWhen(route.When),
				BackStepID: route.BackStepId,
			}
		}
		result[stepID] = normalized
	}
	return result, nil
}

func normalizeConditions(values map[string]*yamlidl.ConditionDefinition) (map[string]ConditionDefinition, error) {
	if values == nil {
		return nil, fmt.Errorf("condition.conditions is required")
	}
	result := make(map[string]ConditionDefinition, len(values))
	for id, value := range values {
		if value == nil {
			return nil, fmt.Errorf("condition.%s is null", id)
		}
		if err := value.IsValid(); err != nil {
			return nil, fmt.Errorf("condition.%s: %w", id, err)
		}
		output := value.Output
		result[id] = ConditionDefinition{
			PromptRef: normalizePromptRef(value.PromptRef),
			Output: OutputDefinition{
				Key:         output.Key,
				Type:        output.Type,
				Source:      output.GetSource(),
				Description: output.GetDescription(),
				Values:      append([]string(nil), output.Values...),
				Minimum:     copyPointer(output.Minimum),
				Maximum:     copyPointer(output.Maximum),
				MinItems:    copyPointer(output.MinItems),
				MaxItems:    copyPointer(output.MaxItems),
			},
			ExclusiveGroup: value.GetExclusiveGroup(),
		}
	}
	return result, nil
}

func normalizePrompts(values map[string]*yamlidl.PromptDefinition) (map[string]PromptDefinition, error) {
	if values == nil {
		return nil, fmt.Errorf("prompt.prompts is required")
	}
	result := make(map[string]PromptDefinition, len(values))
	for id, value := range values {
		if value == nil {
			return nil, fmt.Errorf("prompt.%s is null", id)
		}
		if err := value.IsValid(); err != nil {
			return nil, fmt.Errorf("prompt.%s: %w", id, err)
		}
		if value.Skills == nil {
			return nil, fmt.Errorf("prompt.%s.skills is required", id)
		}
		skills := make([]SkillBinding, len(value.Skills))
		for index, skill := range value.Skills {
			if skill == nil {
				return nil, fmt.Errorf("prompt.%s.skills[%d] is null", id, index)
			}
			if err := skill.IsValid(); err != nil {
				return nil, fmt.Errorf("prompt.%s.skills[%d]: %w", id, index, err)
			}
			skills[index] = SkillBinding{ID: skill.Id, Prompt: skill.Prompt, Optional: copyPointer(skill.Optional)}
		}
		result[id] = PromptDefinition{Prompt: value.Prompt, Skills: skills}
	}
	return result, nil
}

func normalizePromptRef(value *yamlidl.PromptRef) PromptRef {
	return PromptRef{File: value.File, PromptID: value.PromptId}
}

func normalizeWhen(value *yamlidl.When) When {
	groups := make([][]string, len(value.AnyOf))
	for index, group := range value.AnyOf {
		groups[index] = append([]string(nil), group...)
	}
	return When{AnyOf: groups}
}

func copyPointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
