package workflowview

import (
	"testing"

	"github.com/zeefan1555/fanloop/internal/state"
)

func TestDisplaySnapshotStatusMatrix(t *testing.T) {
	loaded, err := loadRuntimeControls()
	if err != nil {
		t.Fatal(err)
	}
	first := "first"
	third := "third"
	tests := []struct {
		name     string
		current  state.State
		status   DisplayStatus
		progress int
		found    bool
		human    bool
	}{
		{name: "completed", current: state.State{}, status: DisplayDone, progress: 100, found: true},
		{name: "awaiting confirmation", current: state.State{CurrentStepID: &first, CurrentStepStatus: state.StepAwaitingConfirmation}, status: DisplayAwaitingConfirmation, progress: 0, found: true},
		{name: "human review", current: state.State{CurrentStepID: &third, CurrentStepStatus: state.StepReady}, status: DisplayReady, progress: 67, found: true, human: true},
		{name: "blocked", current: state.State{CurrentStepID: &first, CurrentStepStatus: state.StepBlocked}, status: DisplayBlocked, progress: 0, found: true},
		{name: "fixing", current: state.State{CurrentStepID: &first, CurrentStepStatus: state.StepFixing}, status: DisplayFixing, progress: 0, found: true},
		{name: "in progress", current: state.State{CurrentStepID: &first, CurrentStepStatus: state.StepInProgress}, status: DisplayInProgress, progress: 0, found: true},
		{name: "ready", current: state.State{CurrentStepID: &first, CurrentStepStatus: state.StepReady}, status: DisplayReady, progress: 0, found: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			view := Display(loaded.Workflow, test.current)
			if view.Status != test.status || view.ProgressPercent != test.progress || view.Location.Found != test.found || view.RequiresHumanReview != test.human {
				t.Fatalf("display = %#v, want status=%q progress=%d found=%v human=%v", view, test.status, test.progress, test.found, test.human)
			}
		})
	}
}

func TestDisplaySnapshotOrdersSkippedCurrentAndPendingSteps(t *testing.T) {
	loaded, err := loadRuntimeControls()
	if err != nil {
		t.Fatal(err)
	}
	third := "third"
	view := Display(loaded.Workflow, state.State{
		CurrentStepID: &third, CurrentStepStatus: state.StepInProgress,
		SkippedStepIDs: []string{"first", "second"},
	})

	if view.Location.StepID != third || view.ProgressPercent != 67 {
		t.Fatalf("location/progress = %#v / %d", view.Location, view.ProgressPercent)
	}
	steps := view.Stages[0].Jobs[0].Steps
	want := []DisplayStepState{DisplayStepSkipped, DisplayStepSkipped, DisplayStepCurrent}
	for index := range want {
		if steps[index].State != want[index] {
			t.Fatalf("step %d state = %q, want %q", index, steps[index].State, want[index])
		}
	}
}

func TestDisplaySnapshotKeepsUnknownCurrentExplicit(t *testing.T) {
	loaded, err := loadRuntimeControls()
	if err != nil {
		t.Fatal(err)
	}
	missing := "missing"
	view := Display(loaded.Workflow, state.State{CurrentStepID: &missing, CurrentStepStatus: state.StepReady})
	if view.Location.Found || view.Location.StepID != missing || view.ProgressPercent != 0 {
		t.Fatalf("unknown display = %#v", view)
	}
	for _, stage := range view.Stages {
		for _, job := range stage.Jobs {
			for _, step := range job.Steps {
				if step.State != DisplayStepPending {
					t.Fatalf("unknown current marked %q as %q", step.ID, step.State)
				}
			}
		}
	}
}
