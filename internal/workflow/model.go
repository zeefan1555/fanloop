package workflow

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/zeefan1555/fanloop/internal/idl/yamlidl"
)

// Workflow is the resolved, immutable five-file Workflow Bundle.
type Workflow struct {
	SchemaVersion int32
	ID            string
	Stages        []Stage
	Flows         map[string][]FlowRoute
	Conditions    map[string]ConditionDefinition
	Loops         map[string][]LoopRoute
	Prompts       map[string]PromptDefinition
}

type Stage struct {
	ID   string
	Name string
	Jobs []Job
}

type Job struct {
	ID    string
	Name  string
	Steps []Step
}

type Step struct {
	ID       string
	Name     string
	Executor StepExecutor
}

type StepExecutor = yamlidl.Executor

const (
	StepExecutorAgent = yamlidl.Executor_agent
	StepExecutorHuman = yamlidl.Executor_human
)

type PromptRef struct {
	File     string
	PromptID string
}

type When struct {
	AnyOf [][]string
}

func (when When) Matches(conditions map[string]bool) bool {
	for _, group := range when.AnyOf {
		matched := true
		for _, conditionID := range group {
			if !conditions[conditionID] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

type FlowRoute struct {
	PromptRef  PromptRef
	When       When
	NextStepID string
	Terminal   bool
}

type LoopRoute struct {
	PromptRef  PromptRef
	When       When
	BackStepID string
}

type ConditionDefinition struct {
	PromptRef      PromptRef
	Output         OutputDefinition
	ExclusiveGroup string
}

type OutputDefinition struct {
	Key         string
	Type        OutputType
	Source      string
	Description string
	Values      []string
	Minimum     *int64
	Maximum     *int64
	MinItems    *int32
	MaxItems    *int32
}

const OutputSourceTraceDocumentURL = yamlidl.OUTPUT_SOURCE_TRACE_DOCUMENT_URL

type OutputType = yamlidl.OutputType

const (
	OutputString  = yamlidl.OutputType_string
	OutputBoolean = yamlidl.OutputType_boolean
	OutputInteger = yamlidl.OutputType_integer
	OutputPath    = yamlidl.OutputType_path
	OutputURL     = yamlidl.OutputType_url
	OutputURLList = yamlidl.OutputType_url_list
	OutputEnum    = yamlidl.OutputType_enum_value
	OutputObject  = yamlidl.OutputType_object
)

func OutputTypeFromString(value string) (OutputType, error) {
	return yamlidl.OutputTypeFromString(value)
}

type PromptDefinition struct {
	Prompt string
	Skills []SkillBinding
}

type SkillBinding struct {
	ID       string
	Prompt   string
	Optional *bool
}

type StepContext struct {
	Stage ContextRef `json:"stage"`
	Job   ContextRef `json:"job"`
	Step  Step       `json:"step"`
}

type ContextRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Ref struct {
	ID     string `json:"id"`
	Digest string `json:"digest"`
}

type Loaded struct {
	Workflow Workflow
	Ref      Ref
}

func (workflow Workflow) OrderedStepIDs() []string {
	result := make([]string, 0)
	for _, stage := range workflow.Stages {
		for _, job := range stage.Jobs {
			for _, step := range job.Steps {
				result = append(result, step.ID)
			}
		}
	}
	return result
}

func (workflow Workflow) FindStep(id string) (StepContext, int, bool) {
	position := 0
	for _, stage := range workflow.Stages {
		for _, job := range stage.Jobs {
			for _, step := range job.Steps {
				if step.ID == id {
					return StepContext{
						Stage: ContextRef{ID: stage.ID, Name: stage.Name},
						Job:   ContextRef{ID: job.ID, Name: job.Name},
						Step:  step,
					}, position, true
				}
				position++
			}
		}
	}
	return StepContext{}, -1, false
}

func (workflow Workflow) FirstStepID() (string, bool) {
	ids := workflow.OrderedStepIDs()
	if len(ids) == 0 {
		return "", false
	}
	return ids[0], true
}

func (workflow Workflow) FlowRoutesForStep(stepID string) ([]FlowRoute, bool) {
	routes, ok := workflow.Flows[stepID]
	return routes, ok
}

func (workflow Workflow) LoopRoutesForStep(stepID string) ([]LoopRoute, bool) {
	routes, ok := workflow.Loops[stepID]
	return routes, ok
}

func (workflow Workflow) Condition(id string) (ConditionDefinition, bool) {
	condition, ok := workflow.Conditions[id]
	return condition, ok
}

func (workflow Workflow) Prompt(ref PromptRef) (PromptDefinition, bool) {
	prompt, ok := workflow.Prompts[ref.PromptID]
	return prompt, ok
}

func (workflow Workflow) RelevantConditionIDs(stepID string) []string {
	seen := map[string]bool{}
	for _, route := range workflow.Flows[stepID] {
		collectConditionIDs(route.When, seen)
	}
	for _, route := range workflow.Loops[stepID] {
		collectConditionIDs(route.When, seen)
	}
	result := make([]string, 0, len(seen))
	for id := range seen {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}

func (workflow Workflow) ValidateRegisteredOutput(key string, outputType OutputType, value json.RawMessage) error {
	found := false
	for _, condition := range workflow.Conditions {
		if condition.Output.Key != key {
			continue
		}
		found = true
		if condition.Output.Type != outputType {
			return fmt.Errorf("Output %q type does not match its definition", key)
		}
		if ValidateOutput(condition.Output, value) == nil {
			return nil
		}
	}
	if !found {
		return fmt.Errorf("Output %q is not declared by the bound Workflow", key)
	}
	return fmt.Errorf("Output %q is invalid", key)
}

func collectConditionIDs(when When, seen map[string]bool) {
	for _, group := range when.AnyOf {
		for _, id := range group {
			seen[id] = true
		}
	}
}
