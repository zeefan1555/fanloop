package workflowview

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeefan1555/commonloop/internal/state"
	"github.com/zeefan1555/commonloop/internal/workflow"
)

func TestProjectResolvesPromptsFromWorkflowBundle(t *testing.T) {
	loaded, err := workflow.Load("technical-solution-design")
	if err != nil {
		t.Fatal(err)
	}
	definition := loaded.Workflow
	stepID, _ := definition.FirstStepID()
	route := definition.Flows[stepID][0]
	conditionID := route.When.AnyOf[0][0]
	condition, _ := definition.Condition(conditionID)

	routePrompt := definition.Prompts[route.PromptRef.PromptID]
	routePrompt.Prompt = "route prompt from test bundle"
	definition.Prompts[route.PromptRef.PromptID] = routePrompt
	conditionPrompt := definition.Prompts[condition.PromptRef.PromptID]
	conditionPrompt.Prompt = "condition prompt from test bundle"
	definition.Prompts[condition.PromptRef.PromptID] = conditionPrompt

	projected := Project(definition, state.State{
		CurrentStepID: &stepID, CurrentStepStatus: state.StepReady,
		Outputs: map[string]state.RegisteredOutput{},
	})
	if projected.Current.Prompt.Content != routePrompt.Prompt || projected.Current.AvailableRoutes[0].Prompt.Content != routePrompt.Prompt {
		t.Fatalf("Step or Route Prompt was not resolved from the Workflow Bundle: %#v", projected.Current)
	}
	if len(projected.Current.Prompt.Skills) != 1 || !strings.HasSuffix(
		projected.Current.Prompt.Skills[0].Path,
		filepath.FromSlash("skills/technical-solution-design/technical-problem-framing/SKILL.md"),
	) {
		t.Fatalf("source Skill path was not resolved from the bound Workflow group: %#v", projected.Current.Prompt.Skills)
	}
	for _, view := range projected.Current.Conditions {
		if view.Id == conditionID && view.Prompt.Content == conditionPrompt.Prompt {
			return
		}
	}
	t.Fatalf("Condition Prompt %q was not resolved from the Workflow Bundle", conditionID)
}
