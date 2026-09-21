package dashboard

import (
	"errors"
	"fmt"
	"strings"

	"github.com/zeefan1555/fanloop/internal/workflow"
	"go.yaml.in/yaml/v3"
)

const terminalStepID = "__terminal__"

var errUnknownWorkflow = errors.New("Workflow is not available")

type topologyResponse struct {
	Workflow  topologyWorkflow         `json:"workflow"`
	Routes    []topologyRoute          `json:"routes"`
	Fragments map[string]stepFragments `json:"fragments"`
}

type topologyWorkflow struct {
	ID     string          `json:"id"`
	Stages []topologyStage `json:"stages"`
}

type topologyStage struct {
	ID   string        `json:"id"`
	Name string        `json:"name"`
	Jobs []topologyJob `json:"jobs"`
}

type topologyJob struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Steps []topologyStep `json:"steps"`
}

type topologyStep struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Executor string `json:"executor"`
}

type topologyRoute struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	FromStepID string `json:"from_step_id"`
	ToStepID   string `json:"to_step_id"`
}

type sourceFragment struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

type stepFragments struct {
	Step       []sourceFragment `json:"step"`
	Prompt     []sourceFragment `json:"prompt"`
	Forward    []sourceFragment `json:"forward"`
	Return     []sourceFragment `json:"return"`
	Conditions []sourceFragment `json:"conditions"`
	Skills     []sourceFragment `json:"skills"`
}

type yamlSource struct {
	root      *yaml.Node
	lineCount int
}

func buildTopology(workflowID string, readFile func(string) ([]byte, error)) (topologyResponse, error) {
	contents := make(map[string][]byte, len(workflow.BundleFileNames()))
	sources := make(map[string]yamlSource, len(workflow.BundleFileNames()))
	for _, name := range workflow.BundleFileNames() {
		content, err := readFile(workflowID + "/" + name)
		if err != nil {
			return topologyResponse{}, err
		}
		contents[name] = content
		source, err := parseYAMLSource(content)
		if err != nil {
			return topologyResponse{}, fmt.Errorf("%s: %w", name, err)
		}
		sources[name] = source
	}
	loaded, err := workflow.DecodeBundle(contents["workflow.yaml"], contents["flow.yaml"], contents["condition.yaml"], contents["loop.yaml"], contents["prompt.yaml"])
	if err != nil {
		return topologyResponse{}, err
	}
	if loaded.Ref.ID != workflowID {
		return topologyResponse{}, fmt.Errorf("workflow path does not match its id")
	}

	result := topologyResponse{
		Workflow:  projectWorkflow(loaded.Workflow),
		Fragments: make(map[string]stepFragments),
	}
	for _, stepID := range loaded.Workflow.OrderedStepIDs() {
		fragments, routes, err := projectStep(readFile, workflowID, loaded.Workflow, sources, stepID)
		if err != nil {
			return topologyResponse{}, err
		}
		result.Fragments[stepID] = fragments
		result.Routes = append(result.Routes, routes...)
	}
	return result, nil
}

func projectWorkflow(value workflow.Workflow) topologyWorkflow {
	result := topologyWorkflow{ID: value.ID, Stages: make([]topologyStage, 0, len(value.Stages))}
	for _, stage := range value.Stages {
		projectedStage := topologyStage{ID: stage.ID, Name: stage.Name, Jobs: make([]topologyJob, 0, len(stage.Jobs))}
		for _, job := range stage.Jobs {
			projectedJob := topologyJob{ID: job.ID, Name: job.Name, Steps: make([]topologyStep, 0, len(job.Steps))}
			for _, step := range job.Steps {
				projectedJob.Steps = append(projectedJob.Steps, topologyStep{ID: step.ID, Name: step.Name, Executor: step.Executor.String()})
			}
			projectedStage.Jobs = append(projectedStage.Jobs, projectedJob)
		}
		result.Stages = append(result.Stages, projectedStage)
	}
	return result
}

func projectStep(readFile func(string) ([]byte, error), workflowID string, value workflow.Workflow, sources map[string]yamlSource, stepID string) (stepFragments, []topologyRoute, error) {
	prefix := workflowID + "/"
	result := stepFragments{}
	stepRange, err := findStepRange(sources["workflow.yaml"], stepID)
	if err != nil {
		return stepFragments{}, nil, err
	}
	result.Step = []sourceFragment{newFragment("step:"+stepID, stepID, prefix+"workflow.yaml", stepRange)}

	seenPrompts := map[string]bool{}
	seenConditions := map[string]bool{}
	seenSkills := map[string]bool{}
	var routes []topologyRoute
	if err := appendCommonReferences(readFile, workflowID, value, sources, &result, seenPrompts, seenConditions, seenSkills); err != nil {
		return stepFragments{}, nil, err
	}
	for index, route := range value.Flows[stepID] {
		routeRange, err := findRouteRange(sources["flow.yaml"], "flow", stepID, index)
		if err != nil {
			return stepFragments{}, nil, err
		}
		kind, target := "forward", route.NextStepID
		label := "Forward to " + target
		if route.Terminal {
			kind, target, label = "terminal", terminalStepID, "Terminal"
		}
		id := fmt.Sprintf("flow:%s:%d", stepID, index)
		routes = append(routes, topologyRoute{ID: id, Kind: kind, FromStepID: stepID, ToStepID: target})
		result.Forward = append(result.Forward, newFragment(id, label, prefix+"flow.yaml", routeRange))
		if err := appendRouteReferences(readFile, workflowID, value, sources, route.PromptRef, route.When, &result, seenPrompts, seenConditions, seenSkills); err != nil {
			return stepFragments{}, nil, err
		}
	}
	for index, route := range value.Loops[stepID] {
		routeRange, err := findRouteRange(sources["loop.yaml"], "loop", stepID, index)
		if err != nil {
			return stepFragments{}, nil, err
		}
		id := fmt.Sprintf("loop:%s:%d", stepID, index)
		routes = append(routes, topologyRoute{ID: id, Kind: "return", FromStepID: stepID, ToStepID: route.BackStepID})
		result.Return = append(result.Return, newFragment(id, "Return to "+route.BackStepID, prefix+"loop.yaml", routeRange))
		if err := appendRouteReferences(readFile, workflowID, value, sources, route.PromptRef, route.When, &result, seenPrompts, seenConditions, seenSkills); err != nil {
			return stepFragments{}, nil, err
		}
	}
	return result, routes, nil
}

func appendCommonReferences(readFile func(string) ([]byte, error), workflowID string, value workflow.Workflow, sources map[string]yamlSource, result *stepFragments, seenPrompts, seenConditions, seenSkills map[string]bool) error {
	for _, control := range []*workflow.CommonControlRoute{value.StepStart, value.Jump} {
		if control == nil {
			continue
		}
		if err := appendPromptReference(readFile, workflowID, value, sources, control.PromptRef, result, seenPrompts, seenSkills); err != nil {
			return err
		}
		for _, group := range control.When.AnyOf {
			for _, conditionID := range group {
				if seenConditions[conditionID] {
					continue
				}
				conditionRange, err := findNamedRange(sources["condition.yaml"], "common_conditions", conditionID)
				if err != nil {
					return err
				}
				result.Conditions = append(result.Conditions, newFragment("common-condition:"+conditionID, conditionID, workflowID+"/condition.yaml", conditionRange))
				seenConditions[conditionID] = true
				condition, ok := value.CommonConditions[conditionID]
				if !ok {
					return fmt.Errorf("Common Condition %q is not defined", conditionID)
				}
				if err := appendPromptReference(readFile, workflowID, value, sources, condition.PromptRef, result, seenPrompts, seenSkills); err != nil {
					return err
				}
			}
		}
	}
	for _, binding := range value.CommonSkills {
		if err := appendSkillReference(readFile, workflowID, binding, result, seenSkills); err != nil {
			return err
		}
	}
	return nil
}

func appendRouteReferences(readFile func(string) ([]byte, error), workflowID string, value workflow.Workflow, sources map[string]yamlSource, promptRef workflow.PromptRef, when workflow.When, result *stepFragments, seenPrompts, seenConditions, seenSkills map[string]bool) error {
	prefix := workflowID + "/"
	if err := appendPromptReference(readFile, workflowID, value, sources, promptRef, result, seenPrompts, seenSkills); err != nil {
		return err
	}
	for _, group := range when.AnyOf {
		for _, conditionID := range group {
			if seenConditions[conditionID] {
				continue
			}
			conditionRange, err := findNamedRange(sources["condition.yaml"], "conditions", conditionID)
			if err != nil {
				return err
			}
			result.Conditions = append(result.Conditions, newFragment("condition:"+conditionID, conditionID, prefix+"condition.yaml", conditionRange))
			seenConditions[conditionID] = true
			condition, ok := value.Conditions[conditionID]
			if !ok {
				return fmt.Errorf("Condition %q is not defined", conditionID)
			}
			if err := appendPromptReference(readFile, workflowID, value, sources, condition.PromptRef, result, seenPrompts, seenSkills); err != nil {
				return err
			}
		}
	}
	return nil
}

func appendPromptReference(readFile func(string) ([]byte, error), workflowID string, value workflow.Workflow, sources map[string]yamlSource, promptRef workflow.PromptRef, result *stepFragments, seenPrompts, seenSkills map[string]bool) error {
	if seenPrompts[promptRef.PromptID] {
		return nil
	}
	prefix := workflowID + "/"
	promptRange, err := findNamedRange(sources["prompt.yaml"], "prompts", promptRef.PromptID)
	if err != nil {
		return err
	}
	result.Prompt = append(result.Prompt, newFragment("prompt:"+promptRef.PromptID, promptRef.PromptID, prefix+"prompt.yaml", promptRange))
	seenPrompts[promptRef.PromptID] = true
	prompt, ok := value.Prompts[promptRef.PromptID]
	if !ok {
		return fmt.Errorf("Prompt %q is not defined", promptRef.PromptID)
	}
	for _, binding := range prompt.Skills {
		if err := appendSkillReference(readFile, workflowID, binding, result, seenSkills); err != nil {
			return err
		}
	}
	return nil
}

func appendSkillReference(readFile func(string) ([]byte, error), workflowID string, binding workflow.SkillBinding, result *stepFragments, seenSkills map[string]bool) error {
	if seenSkills[binding.ID] {
		return nil
	}
	relative := workflowID + "/skills/" + binding.ID + "/SKILL.md"
	content, err := readFile(relative)
	if err != nil {
		return fmt.Errorf("Skill %q is missing regular SKILL.md: %w", binding.ID, err)
	}
	if strings.TrimSpace(string(content)) == "" {
		return fmt.Errorf("Skill %q has empty SKILL.md", binding.ID)
	}
	result.Skills = append(result.Skills, sourceFragment{ID: "skill:" + binding.ID, Label: binding.ID, Path: relative, StartLine: 1, EndLine: lineCount(content)})
	seenSkills[binding.ID] = true
	return nil
}

type lineRange struct {
	start int
	end   int
}

func newFragment(id, label, path string, lines lineRange) sourceFragment {
	return sourceFragment{ID: id, Label: label, Path: path, StartLine: lines.start, EndLine: lines.end}
}

func parseYAMLSource(content []byte) (yamlSource, error) {
	var document yaml.Node
	if err := yaml.Unmarshal(content, &document); err != nil {
		return yamlSource{}, err
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return yamlSource{}, fmt.Errorf("YAML root must be a mapping")
	}
	return yamlSource{root: document.Content[0], lineCount: lineCount(content)}, nil
}

func lineCount(content []byte) int {
	if len(content) == 0 {
		return 1
	}
	count := strings.Count(string(content), "\n") + 1
	if content[len(content)-1] == '\n' {
		count--
	}
	if count < 1 {
		return 1
	}
	return count
}

func findStepRange(source yamlSource, stepID string) (lineRange, error) {
	stages, ok := mappingValue(source.root, "stages")
	if !ok || stages.Kind != yaml.SequenceNode {
		return lineRange{}, fmt.Errorf("workflow.yaml has no stages")
	}
	for _, stage := range stages.Content {
		jobs, ok := mappingValue(stage, "jobs")
		if !ok || jobs.Kind != yaml.SequenceNode {
			continue
		}
		for _, job := range jobs.Content {
			steps, ok := mappingValue(job, "steps")
			if !ok || steps.Kind != yaml.SequenceNode {
				continue
			}
			for index, step := range steps.Content {
				if mappingScalar(step, "id") == stepID {
					return sequenceItemRange(steps, index, source.lineCount), nil
				}
			}
		}
	}
	return lineRange{}, fmt.Errorf("Step %q has no source anchor", stepID)
}

func findRouteRange(source yamlSource, section, stepID string, index int) (lineRange, error) {
	routes, ok := mappingValue(source.root, section)
	if !ok || routes.Kind != yaml.MappingNode {
		return lineRange{}, fmt.Errorf("%s has no %s mapping", section+".yaml", section)
	}
	items, ok := mappingValue(routes, stepID)
	if !ok || items.Kind != yaml.SequenceNode || index < 0 || index >= len(items.Content) {
		return lineRange{}, fmt.Errorf("%s Route %q[%d] has no source anchor", section, stepID, index)
	}
	return sequenceItemRange(items, index, source.lineCount), nil
}

func findNamedRange(source yamlSource, section, id string) (lineRange, error) {
	values, ok := mappingValue(source.root, section)
	if !ok || values.Kind != yaml.MappingNode {
		return lineRange{}, fmt.Errorf("YAML has no %s mapping", section)
	}
	for index := 0; index+1 < len(values.Content); index += 2 {
		if values.Content[index].Value == id {
			return mappingEntryRange(values, index, source.lineCount), nil
		}
	}
	return lineRange{}, fmt.Errorf("%s %q has no source anchor", section, id)
}

func mappingValue(node *yaml.Node, key string) (*yaml.Node, bool) {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil, false
	}
	for index := 0; index+1 < len(node.Content); index += 2 {
		if node.Content[index].Value == key {
			return node.Content[index+1], true
		}
	}
	return nil, false
}

func mappingScalar(node *yaml.Node, key string) string {
	value, ok := mappingValue(node, key)
	if !ok || value.Kind != yaml.ScalarNode {
		return ""
	}
	return value.Value
}

func mappingEntryRange(mapping *yaml.Node, keyIndex, fallbackEnd int) lineRange {
	start := mapping.Content[keyIndex].Line
	end := nodeEndLine(mapping.Content[keyIndex+1])
	return normalizeRange(start, end, fallbackEnd)
}

func sequenceItemRange(sequence *yaml.Node, index, fallbackEnd int) lineRange {
	start := sequence.Content[index].Line
	end := nodeEndLine(sequence.Content[index])
	return normalizeRange(start, end, fallbackEnd)
}

func normalizeRange(start, end, fallbackEnd int) lineRange {
	if start < 1 {
		start = 1
	}
	if end < start {
		end = start
	}
	if fallbackEnd >= start && end > fallbackEnd {
		end = fallbackEnd
	}
	return lineRange{start: start, end: end}
}

func nodeEndLine(node *yaml.Node) int {
	if node == nil {
		return 0
	}
	end := node.Line
	if node.Kind == yaml.ScalarNode {
		end += strings.Count(node.Value, "\n")
	}
	for _, child := range node.Content {
		if childEnd := nodeEndLine(child); childEnd > end {
			end = childEnd
		}
	}
	return end
}
