package workflowview

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

func TestProjectResolvesPromptsFromWorkflowBundle(t *testing.T) {
	loaded, err := workflow.Load("technical-solution-design")
	if err != nil {
		t.Fatal(err)
	}
	definition := loaded.Workflow
	stepID := "confirm_technical_problem"
	route := definition.Flows[stepID][0]
	conditionID := "panorama_card_published"
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
		filepath.FromSlash("skills/technical-solution-design/technical-problem-approval/SKILL.md"),
	) {
		t.Fatalf("source Skill path was not resolved from the bound Workflow group: %#v", projected.Current.Prompt.Skills)
	}
	for _, view := range projected.Current.Conditions {
		if view.Id != conditionID {
			continue
		}
		if view.Prompt.Content != conditionPrompt.Prompt || len(view.Prompt.Skills) != 1 ||
			view.Prompt.Skills[0].Id != "technical-solution-panorama" ||
			!strings.HasSuffix(view.Prompt.Skills[0].Path,
				filepath.FromSlash("skills/technical-solution-design/technical-solution-panorama/SKILL.md")) {
			t.Fatalf("Condition Prompt or Skill was not resolved from the Workflow Bundle: %#v", view.Prompt)
		}
		return
	}
	t.Fatalf("Condition Prompt %q was not resolved from the Workflow Bundle", conditionID)
}

func TestFormatPanoramaStageShowsMaintainerSteps(t *testing.T) {
	loaded, err := workflow.Load("fanloop-maintainer")
	if err != nil {
		t.Fatal(err)
	}
	got := FormatPanoramaStage(loaded.Workflow.Stages[0], func(step workflow.Step) string { return step.Name })
	want := "需求确认：工作区准备 → 需求澄清 → 需求确认"
	if got != want {
		t.Fatalf("Panorama Stage = %q, want %q", got, want)
	}
}
