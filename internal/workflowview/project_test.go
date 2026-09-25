package workflowview

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

func TestProjectAwaitingConfirmationShowsOnlyCommonControls(t *testing.T) {
	loaded, err := loadRuntimeControls()
	if err != nil {
		t.Fatal(err)
	}
	stepID := "first"
	projected := ProjectLoaded(loaded, state.State{CurrentStepID: &stepID, CurrentStepStatus: state.StepAwaitingConfirmation, Outputs: map[string]state.RegisteredOutput{}})
	if projected.Current.Execution.Status.String() != "awaiting_confirmation" || len(projected.Current.Prompt.Skills) != 0 || len(projected.Current.Conditions) != 0 {
		t.Fatalf("awaiting current = %#v", projected.Current)
	}
	if len(projected.Current.CommonSkills) != 1 || len(projected.Current.CommonConditions) != 2 || !projected.Current.AvailableRoutes[0].Route.GetStartCurrentStep() {
		t.Fatalf("common controls = %#v", projected.Current)
	}
}

func TestProjectResolvesPromptsFromWorkflowBundle(t *testing.T) {
	loaded, err := loadRuntimeControls()
	if err != nil {
		t.Fatal(err)
	}
	definition := loaded.Workflow
	stepID := "first"
	route := definition.Flows[stepID][0]
	conditionID := "first_completed"
	condition, _ := definition.Condition(conditionID)

	routePrompt := definition.Prompts[route.PromptRef.PromptID]
	routePrompt.Prompt = "route prompt from test bundle"
	definition.Prompts[route.PromptRef.PromptID] = routePrompt
	conditionPrompt := definition.Prompts[condition.PromptRef.PromptID]
	conditionPrompt.Prompt = "condition prompt from test bundle"
	definition.Prompts[condition.PromptRef.PromptID] = conditionPrompt
	loaded.Workflow = definition
	t.Setenv("FANLOOP_DATA_HOME", t.TempDir())

	projected := ProjectLoaded(loaded, state.State{
		CurrentStepID: &stepID, CurrentStepStatus: state.StepReady,
		Outputs: map[string]state.RegisteredOutput{},
	})
	if projected.Current.Prompt.Content != routePrompt.Prompt || projected.Current.AvailableRoutes[0].Prompt.Content != routePrompt.Prompt {
		t.Fatalf("Step or Route Prompt was not resolved from the Workflow Bundle: %#v", projected.Current)
	}
	if len(projected.Current.CommonSkills) != 1 || !strings.HasSuffix(
		projected.Current.CommonSkills[0].Path,
		filepath.FromSlash("runtime-controls/skills/common-control-guide/SKILL.md"),
	) {
		t.Fatalf("source Skill path was not resolved from the bound Workflow group: %#v", projected.Current.CommonSkills)
	}
	for _, view := range projected.Current.Conditions {
		if view.Id != conditionID {
			continue
		}
		if view.Prompt.Content != conditionPrompt.Prompt || len(view.Prompt.Skills) != 0 {
			t.Fatalf("Condition Prompt or Skill was not resolved from the Workflow Bundle: %#v", view.Prompt)
		}
		return
	}
	t.Fatalf("Condition Prompt %q was not resolved from the Workflow Bundle", conditionID)
}

func TestFormatPanoramaStageShowsConfiguredSteps(t *testing.T) {
	loaded, err := loadRuntimeControls()
	if err != nil {
		t.Fatal(err)
	}
	display := Display(loaded.Workflow, state.State{})
	got := FormatPanoramaStage(display.Stages[0], func(step DisplayStep) string { return step.Name })
	want := "Runtime：First → Second → Third"
	if got != want {
		t.Fatalf("Panorama Stage = %q, want %q", got, want)
	}
}

func loadRuntimeControls() (workflow.Loaded, error) {
	return workflow.LoadFrom(filepath.Join("..", "workflow", "testdata"), "runtime-controls")
}
