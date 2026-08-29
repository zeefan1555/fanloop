package workflow

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"slices"
	"strings"

	"github.com/zeefan1555/commonloop/internal/idl/yamlidl"
)

var (
	ErrOutputType       = errors.New("output type mismatch")
	ErrOutputConstraint = errors.New("output constraint violation")
)

type outputValidationError struct {
	kind    error
	message string
}

func (value outputValidationError) Error() string { return value.message }
func (value outputValidationError) Unwrap() error { return value.kind }

func outputTypeError(message string) error {
	return outputValidationError{kind: ErrOutputType, message: message}
}

func outputConstraintError(message string) error {
	return outputValidationError{kind: ErrOutputConstraint, message: message}
}

func ValidateBundle(value *Workflow, flowSchema, conditionSchema, loopSchema, promptSchema int32) error {
	if value.SchemaVersion != yamlidl.WORKFLOW_SCHEMA_VERSION {
		return fmt.Errorf("unsupported workflow schema version %d", value.SchemaVersion)
	}
	if flowSchema != yamlidl.FLOW_SCHEMA_VERSION || conditionSchema != yamlidl.CONDITION_SCHEMA_VERSION || loopSchema != yamlidl.LOOP_SCHEMA_VERSION || promptSchema != yamlidl.PROMPT_SCHEMA_VERSION {
		return fmt.Errorf("unsupported Bundle schema versions: flow=%d condition=%d loop=%d prompt=%d", flowSchema, conditionSchema, loopSchema, promptSchema)
	}
	if !workflowIDPattern.MatchString(value.ID) {
		return fmt.Errorf("workflow id is required")
	}

	stepIDs, positions, err := validateWorkflowTree(*value)
	if err != nil {
		return err
	}
	if len(value.Flows) != len(stepIDs) {
		return fmt.Errorf("flow.yaml must define Routes for every Workflow Step")
	}
	if len(value.Loops) != len(stepIDs) {
		return fmt.Errorf("loop.yaml must define Routes for every Workflow Step")
	}

	for promptID, prompt := range value.Prompts {
		if err := validatePrompt(promptID, prompt); err != nil {
			return err
		}
	}
	usedPrompts, usedConditions := map[string]bool{}, map[string]bool{}
	outputTypes := map[string]OutputType{}
	for conditionID, condition := range value.Conditions {
		if err := validateCondition(conditionID, condition, value.Prompts, usedPrompts); err != nil {
			return err
		}
		if previous, exists := outputTypes[condition.Output.Key]; exists && previous != condition.Output.Type {
			return fmt.Errorf("Output %q uses types %q and %q", condition.Output.Key, previous, condition.Output.Type)
		}
		outputTypes[condition.Output.Key] = condition.Output.Type
	}

	for stepID, routes := range value.Flows {
		if !stepIDs[stepID] || len(routes) == 0 {
			return fmt.Errorf("Flow %q is unknown or has no Route", stepID)
		}
		for index, route := range routes {
			owner := fmt.Sprintf("flow.%s[%d]", stepID, index)
			if err := validatePromptRef(owner, route.PromptRef, value.Prompts, usedPrompts); err != nil {
				return err
			}
			if err := validateWhen(owner, route.When, value.Conditions, usedConditions); err != nil {
				return err
			}
			if route.Terminal == (route.NextStepID != "") {
				return fmt.Errorf("%s must declare exactly one of next_step_id or terminal", owner)
			}
			if route.NextStepID != "" && (!stepIDs[route.NextStepID] || positions[route.NextStepID] <= positions[stepID]) {
				return fmt.Errorf("%s has invalid next_step_id %q", owner, route.NextStepID)
			}
		}
		if err := validateFlowAmbiguity(stepID, routes, value.Conditions); err != nil {
			return err
		}
	}
	if err := validateFlowReachability(*value); err != nil {
		return err
	}

	for stepID, routes := range value.Loops {
		if !stepIDs[stepID] || len(routes) == 0 {
			return fmt.Errorf("Loop %q is unknown or has no Route", stepID)
		}
		for index, route := range routes {
			owner := fmt.Sprintf("loop.%s[%d]", stepID, index)
			if err := validatePromptRef(owner, route.PromptRef, value.Prompts, usedPrompts); err != nil {
				return err
			}
			if err := validateWhen(owner, route.When, value.Conditions, usedConditions); err != nil {
				return err
			}
			if !stepIDs[route.BackStepID] || positions[route.BackStepID] > positions[stepID] {
				return fmt.Errorf("%s has invalid back_step_id %q", owner, route.BackStepID)
			}
		}
		if err := validateLoopAmbiguity(stepID, routes, value.Conditions); err != nil {
			return err
		}
	}

	for conditionID := range value.Conditions {
		if !usedConditions[conditionID] {
			return fmt.Errorf("Condition %q is not referenced", conditionID)
		}
	}
	for promptID := range value.Prompts {
		if !usedPrompts[promptID] {
			return fmt.Errorf("Prompt %q is not referenced", promptID)
		}
	}
	return nil
}

func validateWorkflowTree(value Workflow) (map[string]bool, map[string]int, error) {
	if len(value.Stages) == 0 {
		return nil, nil, fmt.Errorf("workflow has no stages")
	}
	stageIDs, jobIDs, stepIDs, positions := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]int{}
	position := 0
	for _, stage := range value.Stages {
		if !configIDPattern.MatchString(stage.ID) || strings.TrimSpace(stage.Name) == "" || stageIDs[stage.ID] || len(stage.Jobs) == 0 {
			return nil, nil, fmt.Errorf("invalid, duplicate, or empty stage %q", stage.ID)
		}
		stageIDs[stage.ID] = true
		for _, job := range stage.Jobs {
			if !configIDPattern.MatchString(job.ID) || strings.TrimSpace(job.Name) == "" || jobIDs[job.ID] || len(job.Steps) == 0 {
				return nil, nil, fmt.Errorf("invalid, duplicate, or empty Job %q", job.ID)
			}
			jobIDs[job.ID] = true
			for _, step := range job.Steps {
				if err := validateStep(step, stepIDs); err != nil {
					return nil, nil, err
				}
				stepIDs[step.ID], positions[step.ID] = true, position
				position++
			}
		}
	}
	return stepIDs, positions, nil
}

func validateStep(step Step, seen map[string]bool) error {
	if !configIDPattern.MatchString(step.ID) || strings.TrimSpace(step.Name) == "" || seen[step.ID] {
		return fmt.Errorf("workflow has invalid or duplicate Step %q", step.ID)
	}
	switch step.Executor {
	case StepExecutorAgent, StepExecutorHuman:
	default:
		return fmt.Errorf("Step %q has invalid executor %q", step.ID, step.Executor)
	}
	return nil
}

func validatePrompt(id string, prompt PromptDefinition) error {
	if !configIDPattern.MatchString(id) || strings.TrimSpace(prompt.Prompt) == "" || strings.Contains(prompt.Prompt, "{{") || strings.Contains(prompt.Prompt, "${") {
		return fmt.Errorf("Prompt %q is invalid", id)
	}
	seen := map[string]bool{}
	for _, skill := range prompt.Skills {
		if strings.TrimSpace(skill.ID) == "" || strings.TrimSpace(skill.Prompt) == "" || skill.Optional == nil || seen[skill.ID] {
			return fmt.Errorf("Prompt %q has an invalid or duplicate Skill %q", id, skill.ID)
		}
		seen[skill.ID] = true
	}
	return nil
}

func validatePromptRef(owner string, ref PromptRef, prompts map[string]PromptDefinition, used map[string]bool) error {
	if ref.File != yamlidl.PROMPT_FILE || !configIDPattern.MatchString(ref.PromptID) {
		return fmt.Errorf("%q has an invalid PromptRef", owner)
	}
	if _, ok := prompts[ref.PromptID]; !ok {
		return fmt.Errorf("%q references unknown Prompt %q", owner, ref.PromptID)
	}
	used[ref.PromptID] = true
	return nil
}

func validateCondition(id string, condition ConditionDefinition, prompts map[string]PromptDefinition, usedPrompts map[string]bool) error {
	if !configIDPattern.MatchString(id) {
		return fmt.Errorf("Condition %q is invalid", id)
	}
	if condition.ExclusiveGroup != "" && !configIDPattern.MatchString(condition.ExclusiveGroup) {
		return fmt.Errorf("Condition %q has invalid exclusive_group %q", id, condition.ExclusiveGroup)
	}
	if err := validatePromptRef("condition."+id, condition.PromptRef, prompts, usedPrompts); err != nil {
		return err
	}
	if !configIDPattern.MatchString(condition.Output.Key) {
		return fmt.Errorf("Condition %q has invalid Output key %q", id, condition.Output.Key)
	}
	return validateOutputDefinition(condition.Output.Key, condition.Output)
}

func validateWhen(owner string, when When, conditions map[string]ConditionDefinition, used map[string]bool) error {
	if len(when.AnyOf) == 0 {
		return fmt.Errorf("%s has empty when.any_of", owner)
	}
	seenGroups := map[string]bool{}
	for index, group := range when.AnyOf {
		if len(group) == 0 {
			return fmt.Errorf("%s has empty AND group %d", owner, index)
		}
		seenIDs, groups := map[string]bool{}, map[string]string{}
		for _, conditionID := range group {
			condition, ok := conditions[conditionID]
			if !ok || seenIDs[conditionID] {
				return fmt.Errorf("%s references unknown or duplicate Condition %q", owner, conditionID)
			}
			seenIDs[conditionID], used[conditionID] = true, true
			if condition.ExclusiveGroup != "" {
				if previous := groups[condition.ExclusiveGroup]; previous != "" && previous != conditionID {
					return fmt.Errorf("%s requires mutually exclusive Conditions %q and %q", owner, previous, conditionID)
				}
				groups[condition.ExclusiveGroup] = conditionID
			}
		}
		canonical := append([]string(nil), group...)
		slices.Sort(canonical)
		key := strings.Join(canonical, "\x00")
		if seenGroups[key] {
			return fmt.Errorf("%s has duplicate AND group", owner)
		}
		seenGroups[key] = true
	}
	return nil
}

func validateFlowAmbiguity(stepID string, routes []FlowRoute, conditions map[string]ConditionDefinition) error {
	for left := range routes {
		for right := left + 1; right < len(routes); right++ {
			if whensOverlap(routes[left].When, routes[right].When, conditions) {
				return fmt.Errorf("Flow %q has statically ambiguous Routes %d and %d", stepID, left, right)
			}
		}
	}
	return nil
}

func validateLoopAmbiguity(stepID string, routes []LoopRoute, conditions map[string]ConditionDefinition) error {
	for left := range routes {
		for right := left + 1; right < len(routes); right++ {
			if routes[left].BackStepID == routes[right].BackStepID && whensOverlap(routes[left].When, routes[right].When, conditions) {
				return fmt.Errorf("Loop %q has statically ambiguous Routes %d and %d for %q", stepID, left, right, routes[left].BackStepID)
			}
		}
	}
	return nil
}

func whensOverlap(left, right When, conditions map[string]ConditionDefinition) bool {
	for _, leftGroup := range left.AnyOf {
		for _, rightGroup := range right.AnyOf {
			if groupsCompatible(leftGroup, rightGroup, conditions) {
				return true
			}
		}
	}
	return false
}

func groupsCompatible(left, right []string, conditions map[string]ConditionDefinition) bool {
	groups := map[string]string{}
	for _, conditionID := range append(append([]string(nil), left...), right...) {
		group := conditions[conditionID].ExclusiveGroup
		if group == "" {
			continue
		}
		if previous := groups[group]; previous != "" && previous != conditionID {
			return false
		}
		groups[group] = conditionID
	}
	return true
}

func validateFlowReachability(value Workflow) error {
	first, ok := value.FirstStepID()
	if !ok {
		return fmt.Errorf("workflow has no Steps")
	}
	visited, queue, terminal := map[string]bool{}, []string{first}, false
	for len(queue) > 0 {
		stepID := queue[0]
		queue = queue[1:]
		if visited[stepID] {
			continue
		}
		visited[stepID] = true
		for _, route := range value.Flows[stepID] {
			if route.Terminal {
				terminal = true
			} else {
				queue = append(queue, route.NextStepID)
			}
		}
	}
	if !terminal {
		return fmt.Errorf("Flow has no terminal Route")
	}
	if len(visited) != len(value.OrderedStepIDs()) {
		return fmt.Errorf("Flow has unreachable Steps")
	}
	return nil
}

func validateOutputDefinition(id string, definition OutputDefinition) error {
	validTypes := map[OutputType]bool{
		OutputString: true, OutputBoolean: true, OutputInteger: true,
		OutputPath: true, OutputURL: true, OutputURLList: true,
		OutputEnum: true, OutputObject: true,
	}
	if !validTypes[definition.Type] {
		return fmt.Errorf("output %q has invalid type %q", id, definition.Type)
	}
	if definition.Source != "" && (definition.Source != OutputSourceTraceDocumentURL || definition.Type != OutputURL) {
		return fmt.Errorf("output %q has invalid source %q for type %q", id, definition.Source, definition.Type)
	}
	if definition.Type != OutputEnum && len(definition.Values) > 0 {
		return fmt.Errorf("output %q uses enum values with non-enum type", id)
	}
	if definition.Type == OutputEnum {
		seenValues := map[string]bool{}
		for _, value := range definition.Values {
			if strings.TrimSpace(value) == "" || seenValues[value] {
				return fmt.Errorf("enum output %q has empty or duplicate values", id)
			}
			seenValues[value] = true
		}
		if len(seenValues) == 0 {
			return fmt.Errorf("enum output %q has no values", id)
		}
	}
	if (definition.Minimum != nil || definition.Maximum != nil) && definition.Type != OutputInteger {
		return fmt.Errorf("output %q uses integer bounds with non-integer type", id)
	}
	if definition.Minimum != nil && definition.Maximum != nil && *definition.Minimum > *definition.Maximum {
		return fmt.Errorf("output %q has minimum above maximum", id)
	}
	if (definition.MinItems != nil || definition.MaxItems != nil) && definition.Type != OutputURLList {
		return fmt.Errorf("output %q uses list bounds with non-list type", id)
	}
	if definition.MinItems != nil && *definition.MinItems < 0 || definition.MaxItems != nil && *definition.MaxItems < 0 {
		return fmt.Errorf("output %q has a negative list bound", id)
	}
	if definition.MinItems != nil && definition.MaxItems != nil && *definition.MinItems > *definition.MaxItems {
		return fmt.Errorf("output %q has min_items above max_items", id)
	}
	return nil
}

func ValidateOutput(definition OutputDefinition, raw json.RawMessage) error {
	decode := func(destination any) error {
		decoder := json.NewDecoder(bytes.NewReader(raw))
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
	switch definition.Type {
	case OutputString:
		var value string
		if err := decode(&value); err != nil || strings.TrimSpace(value) == "" {
			return outputTypeError("must be a non-empty string")
		}
	case OutputBoolean:
		var value bool
		if err := decode(&value); err != nil {
			return outputTypeError("must be a boolean")
		}
	case OutputPath:
		var value string
		if err := decode(&value); err != nil || strings.TrimSpace(value) == "" {
			return outputTypeError("must be a non-empty path string")
		}
		clean := filepath.Clean(value)
		if filepath.IsAbs(value) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return outputTypeError("must be relative to the Requirement Root")
		}
	case OutputURL:
		var value string
		if err := decode(&value); err != nil || !validOutputURL(value) {
			return outputTypeError("must be an HTTP URL string")
		}
	case OutputURLList:
		var values []string
		if err := decode(&values); err != nil || len(values) == 0 {
			return outputTypeError("must be a non-empty URL list")
		}
		if definition.MinItems != nil && int32(len(values)) < *definition.MinItems || definition.MaxItems != nil && int32(len(values)) > *definition.MaxItems {
			return outputConstraintError("must satisfy configured list bounds")
		}
		for _, value := range values {
			if !validOutputURL(value) {
				return outputTypeError("contains an invalid URL")
			}
		}
	case OutputEnum:
		var value string
		if err := decode(&value); err != nil {
			return outputTypeError("must be a string enum")
		}
		for _, candidate := range definition.Values {
			if value == candidate {
				return nil
			}
		}
		return outputConstraintError(fmt.Sprintf("value %q is not allowed", value))
	case OutputInteger:
		var value int64
		if err := decode(&value); err != nil {
			return outputTypeError("must be an integer within the configured bounds")
		}
		if definition.Minimum != nil && value < *definition.Minimum || definition.Maximum != nil && value > *definition.Maximum {
			return outputConstraintError("must be an integer within the configured bounds")
		}
	case OutputObject:
		var value map[string]json.RawMessage
		if err := decode(&value); err != nil || value == nil {
			return outputTypeError("must be a JSON object")
		}
	}
	return nil
}

func validOutputURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}
