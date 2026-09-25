package workflowview

import (
	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

type DisplayStatus string

const (
	DisplayDone                 DisplayStatus = "done"
	DisplayAwaitingConfirmation DisplayStatus = "awaiting_confirmation"
	DisplayBlocked              DisplayStatus = "blocked"
	DisplayFixing               DisplayStatus = "fixing"
	DisplayInProgress           DisplayStatus = "in_progress"
	DisplayReady                DisplayStatus = "ready"
)

type DisplayStepState string

const (
	DisplayStepPending   DisplayStepState = "pending"
	DisplayStepCurrent   DisplayStepState = "current"
	DisplayStepCompleted DisplayStepState = "completed"
	DisplayStepSkipped   DisplayStepState = "skipped"
)

type DisplayLocation struct {
	Found     bool
	StageID   string
	StageName string
	JobID     string
	JobName   string
	StepID    string
	StepName  string
	Executor  workflow.StepExecutor
}

type DisplayStep struct {
	ID       string
	Name     string
	Executor workflow.StepExecutor
	State    DisplayStepState
}

type DisplayJob struct {
	ID    string
	Name  string
	Steps []DisplayStep
}

type DisplayStage struct {
	ID   string
	Name string
	Jobs []DisplayJob
}

type DisplaySnapshot struct {
	Location            DisplayLocation
	Status              DisplayStatus
	RequiresHumanReview bool
	ProgressPercent     int
	Stages              []DisplayStage
}

func Display(definition workflow.Workflow, current state.State) DisplaySnapshot {
	ordered := definition.OrderedStepIDs()
	location, currentPosition := displayLocation(definition, current, ordered)
	skipped := make(map[string]bool, len(current.SkippedStepIDs))
	for _, stepID := range current.SkippedStepIDs {
		skipped[stepID] = true
	}

	result := DisplaySnapshot{
		Location: location, Status: displayStatus(current),
		RequiresHumanReview: current.CurrentStepID != nil && location.Found && location.Executor == workflow.StepExecutorHuman,
		Stages:              make([]DisplayStage, 0, len(definition.Stages)),
	}
	if current.CurrentStepID == nil {
		result.ProgressPercent = 100
	} else if location.Found && len(ordered) > 0 {
		result.ProgressPercent = (currentPosition*100 + len(ordered)/2) / len(ordered)
	}

	position := 0
	for _, stage := range definition.Stages {
		displayStage := DisplayStage{ID: stage.ID, Name: stage.Name, Jobs: make([]DisplayJob, 0, len(stage.Jobs))}
		for _, job := range stage.Jobs {
			displayJob := DisplayJob{ID: job.ID, Name: job.Name, Steps: make([]DisplayStep, 0, len(job.Steps))}
			for _, step := range job.Steps {
				displayJob.Steps = append(displayJob.Steps, DisplayStep{
					ID: step.ID, Name: step.Name, Executor: step.Executor,
					State: displayStepState(current, location.Found, skipped[step.ID], position, currentPosition),
				})
				position++
			}
			displayStage.Jobs = append(displayStage.Jobs, displayJob)
		}
		result.Stages = append(result.Stages, displayStage)
	}
	return result
}

func displayLocation(definition workflow.Workflow, current state.State, ordered []string) (DisplayLocation, int) {
	if current.CurrentStepID != nil {
		context, position, ok := definition.FindStep(*current.CurrentStepID)
		if !ok {
			return DisplayLocation{StepID: *current.CurrentStepID}, 0
		}
		return locationFromContext(context), position
	}
	if len(ordered) == 0 {
		return DisplayLocation{Found: true}, 0
	}
	context, position, ok := definition.FindStep(ordered[len(ordered)-1])
	if !ok {
		return DisplayLocation{Found: true}, len(ordered)
	}
	return locationFromContext(context), position
}

func locationFromContext(context workflow.StepContext) DisplayLocation {
	return DisplayLocation{
		Found: true, StageID: context.Stage.ID, StageName: context.Stage.Name,
		JobID: context.Job.ID, JobName: context.Job.Name,
		StepID: context.Step.ID, StepName: context.Step.Name, Executor: context.Step.Executor,
	}
}

func displayStatus(current state.State) DisplayStatus {
	if current.CurrentStepID == nil {
		return DisplayDone
	}
	if current.CurrentStepStatus == state.StepAwaitingConfirmation {
		return DisplayAwaitingConfirmation
	}
	switch current.CurrentStepStatus {
	case state.StepBlocked:
		return DisplayBlocked
	case state.StepFixing:
		return DisplayFixing
	case state.StepInProgress:
		return DisplayInProgress
	default:
		return DisplayReady
	}
}

func displayStepState(current state.State, currentFound, skipped bool, position, currentPosition int) DisplayStepState {
	if skipped {
		return DisplayStepSkipped
	}
	if current.CurrentStepID == nil || currentFound && position < currentPosition {
		return DisplayStepCompleted
	}
	if currentFound && position == currentPosition {
		return DisplayStepCurrent
	}
	return DisplayStepPending
}
