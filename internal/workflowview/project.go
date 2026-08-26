package workflowview

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/zeefan1555/fanloop/internal/buildinfo"
	"github.com/zeefan1555/fanloop/internal/idl/commonidl"
	"github.com/zeefan1555/fanloop/internal/idl/flowidl"
	"github.com/zeefan1555/fanloop/internal/release"
	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

func Project(definition workflow.Workflow, current state.State) *flowidl.FlowState {
	result := &flowidl.FlowState{
		Status:  flowidl.WorkflowStatus_completed,
		Outputs: registeredOutputs(current.Outputs),
	}
	if current.CurrentStepID == nil {
		return result
	}
	context, _, ok := definition.FindStep(*current.CurrentStepID)
	if !ok {
		return result
	}
	flowRoutes := definition.Flows[context.Step.ID]
	if len(flowRoutes) == 0 {
		return result
	}
	stepPrompt, ok := definition.Prompt(flowRoutes[0].PromptRef)
	if !ok {
		return result
	}
	paths := skillPaths()
	result.Status = flowidl.WorkflowStatus_running
	result.Current = &flowidl.CurrentTask{
		Context: &flowidl.CurrentContext{
			StageId: context.Stage.ID, StageName: context.Stage.Name,
			JobId: context.Job.ID, JobName: context.Job.Name,
			StepId: context.Step.ID, StepName: context.Step.Name, Executor: executor(context.Step.Executor),
		},
		Execution: &flowidl.Execution{
			Status: stepStatus(current.CurrentStepStatus), Summary: optionalString(current.CurrentStepSummary), Evidence: evidence(current.CurrentEvidence),
		},
		Prompt:          prompt(stepPrompt, paths),
		Conditions:      conditionViews(definition, context.Step.ID, paths),
		AvailableRoutes: availableRouteViews(definition, flowRoutes, definition.Loops[context.Step.ID], paths),
	}
	return result
}

func WorkflowRef(value workflow.Ref) *commonidl.WorkflowRef {
	return &commonidl.WorkflowRef{Id: value.ID, Digest: value.Digest}
}

func Requirement(value state.Requirement) *flowidl.Requirement {
	result := &flowidl.Requirement{Title: value.Title}
	if value.SourceURL != "" {
		result.SourceUrl = stringPointer(value.SourceURL)
	}
	return result
}

func conditionViews(definition workflow.Workflow, stepID string, paths map[string]string) []*flowidl.ConditionView {
	ids := definition.RelevantConditionIDs(stepID)
	result := make([]*flowidl.ConditionView, 0, len(ids))
	for _, id := range ids {
		condition, ok := definition.Condition(id)
		if !ok {
			continue
		}
		conditionPrompt, ok := definition.Prompt(condition.PromptRef)
		if !ok {
			continue
		}
		result = append(result, &flowidl.ConditionView{
			Id: id, Prompt: prompt(conditionPrompt, paths), Output: outputSpec(condition.Output), ExclusiveGroup: optionalString(condition.ExclusiveGroup),
		})
	}
	return result
}

func availableRouteViews(definition workflow.Workflow, flowRoutes []workflow.FlowRoute, loopRoutes []workflow.LoopRoute, paths map[string]string) []*flowidl.AvailableRoute {
	result := make([]*flowidl.AvailableRoute, 0, len(flowRoutes)+len(loopRoutes))
	for _, route := range flowRoutes {
		routePrompt, ok := definition.Prompt(route.PromptRef)
		if !ok {
			continue
		}
		selection := &flowidl.RouteSelection{NextStepId: stringPointer(route.NextStepID)}
		if route.Terminal {
			selection = &flowidl.RouteSelection{Terminal: boolPointer(true)}
		}
		result = append(result, &flowidl.AvailableRoute{
			Direction: flowidl.RouteDirection_flow, When: routeWhen(route.When), Route: selection, Prompt: prompt(routePrompt, paths),
		})
	}
	for _, route := range loopRoutes {
		routePrompt, ok := definition.Prompt(route.PromptRef)
		if !ok {
			continue
		}
		result = append(result, &flowidl.AvailableRoute{
			Direction: flowidl.RouteDirection_loop, When: routeWhen(route.When),
			Route: &flowidl.RouteSelection{BackStepId: stringPointer(route.BackStepID)}, Prompt: prompt(routePrompt, paths),
		})
	}
	return result
}

func routeWhen(value workflow.When) *flowidl.RouteWhen {
	result := make([][]string, len(value.AnyOf))
	for index, group := range value.AnyOf {
		result[index] = append([]string(nil), group...)
	}
	return &flowidl.RouteWhen{AnyOf: result}
}

func prompt(value workflow.PromptDefinition, paths map[string]string) *flowidl.Prompt {
	return &flowidl.Prompt{Content: value.Prompt, Skills: skills(value.Skills, paths)}
}

func outputSpec(value workflow.OutputDefinition) *flowidl.OutputSpec {
	return &flowidl.OutputSpec{
		Key: value.Key, Type: outputType(value.Type), Description: optionalString(value.Description),
		Values: append([]string(nil), value.Values...), Minimum: value.Minimum, Maximum: value.Maximum,
		MinItems: value.MinItems, MaxItems: value.MaxItems, Source: optionalString(value.Source),
	}
}

func registeredOutputs(values map[string]state.RegisteredOutput) map[string]*flowidl.RegisteredOutput {
	result := make(map[string]*flowidl.RegisteredOutput, len(values))
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := values[key]
		var dynamic commonidl.JsonValue
		if json.Unmarshal(value.Value, &dynamic) != nil {
			continue
		}
		result[key] = &flowidl.RegisteredOutput{
			Type: outputType(value.Type), Value: &dynamic, ProducerStepId: value.ProducerStepID,
		}
	}
	return result
}

func evidence(values []state.Evidence) []*flowidl.Evidence {
	result := make([]*flowidl.Evidence, 0, len(values))
	for _, value := range values {
		result = append(result, &flowidl.Evidence{
			Source: evidenceSource(value.Source), Content: value.Content, Ref: optionalString(value.Ref),
		})
	}
	return result
}

func skills(values []workflow.SkillBinding, paths map[string]string) []*flowidl.Skill {
	result := make([]*flowidl.Skill, 0, len(values))
	for _, value := range values {
		result = append(result, &flowidl.Skill{Id: value.ID, Prompt: value.Prompt, Optional: value.Optional != nil && *value.Optional, Path: paths[value.ID]})
	}
	return result
}

func skillPaths() map[string]string {
	root, packaged := skillRoot()
	paths := map[string]string{}
	if packaged {
		manifest, err := release.Load(root)
		if err != nil {
			return paths
		}
		for _, skill := range manifest.Skills {
			directory, err := release.Resolve(root, skill.Path)
			if err == nil {
				addSkillPath(paths, skill.Name, filepath.Join(directory, "SKILL.md"))
			}
		}
		return paths
	}
	entries, err := os.ReadDir(filepath.Join(root, "skills"))
	if err != nil {
		return paths
	}
	for _, entry := range entries {
		if entry.IsDir() {
			addSkillPath(paths, entry.Name(), filepath.Join(root, "skills", entry.Name(), "SKILL.md"))
		}
	}
	return paths
}

func skillRoot() (string, bool) {
	if buildinfo.ReleaseVersion == "dev" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return "", false
		}
		return filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..")), false
	}
	executable, err := os.Executable()
	if err != nil {
		return "", true
	}
	resolved, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return "", true
	}
	return release.RootForExecutable(resolved), true
}

func addSkillPath(paths map[string]string, name, path string) {
	info, err := os.Lstat(path)
	if err == nil && info.Mode().IsRegular() {
		paths[name] = path
	}
}

func executor(value workflow.StepExecutor) flowidl.Executor {
	if value == workflow.StepExecutorHuman {
		return flowidl.Executor_human
	}
	return flowidl.Executor_agent
}

func stepStatus(value state.StepStatus) flowidl.StepStatus {
	switch value {
	case state.StepInProgress:
		return flowidl.StepStatus_in_progress
	case state.StepFixing:
		return flowidl.StepStatus_fixing
	case state.StepBlocked:
		return flowidl.StepStatus_blocked
	default:
		return flowidl.StepStatus_ready
	}
}

func outputType(value workflow.OutputType) flowidl.OutputType {
	switch value {
	case workflow.OutputBoolean:
		return flowidl.OutputType_boolean
	case workflow.OutputInteger:
		return flowidl.OutputType_integer
	case workflow.OutputPath:
		return flowidl.OutputType_path
	case workflow.OutputURL:
		return flowidl.OutputType_url
	case workflow.OutputURLList:
		return flowidl.OutputType_url_list
	case workflow.OutputEnum:
		return flowidl.OutputType_enum_value
	case workflow.OutputObject:
		return flowidl.OutputType_object
	default:
		return flowidl.OutputType_string
	}
}

func evidenceSource(value state.EvidenceSource) flowidl.EvidenceSource {
	switch value {
	case state.EvidenceHuman:
		return flowidl.EvidenceSource_human
	case state.EvidenceSystem:
		return flowidl.EvidenceSource_system
	case state.EvidenceAI:
		return flowidl.EvidenceSource_ai
	case state.EvidenceFile:
		return flowidl.EvidenceSource_file
	default:
		return flowidl.EvidenceSource_url
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return stringPointer(value)
}

func stringPointer(value string) *string { return &value }
func boolPointer(value bool) *bool       { return &value }
