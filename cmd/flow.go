package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/zeefan1555/commonloop/internal/flow"
	"github.com/zeefan1555/commonloop/internal/idl/commonidl"
	"github.com/zeefan1555/commonloop/internal/idl/erroridl"
	"github.com/zeefan1555/commonloop/internal/idl/flowidl"
	"github.com/zeefan1555/commonloop/internal/store"
	"github.com/zeefan1555/commonloop/internal/workflow"
)

func newFlowCommand(ioStreams streams, root *string) *cobra.Command {
	command := &cobra.Command{
		Use:   "flow",
		Short: "Initialize and report a requirement through its Workflow",
		Long: `Initialize and report a requirement through its Workflow.

Use init only after status reports NOT_INITIALIZED. Use status immediately before each action.
Use report progress for execution facts, and report result for Conditions and routing.`,
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return command.Help()
		},
	}
	command.AddCommand(
		newFlowInitCommand(ioStreams, root),
		newFlowStatusCommand(ioStreams, root),
		newFlowReportCommand(ioStreams, root),
	)
	return command
}

func newFlowInitCommand(ioStreams streams, root *string) *cobra.Command {
	var controls operationControls
	request := flowidl.NewFlowInitRequest()
	request.Requirement = flowidl.NewRequirement()
	var sourceURL string
	command := operationCommand("flow.init")
	command.Long = `Purpose:
  Initialize a new Requirement with one Workflow, only after flow status reports NOT_INITIALIZED.

Effect:
  Local write. A non-dry-run call creates the Requirement's local Flow State; dry-run validates without writing it.

Request JSON:
  {
    "workflow": "<WORKFLOW_ID>",
    "requirement": {
      "title": "<TITLE>",
      "source_url": "<SOURCE_URL>"
    }
  }

Typed flags:
  commonloop flow init --root <ABSOLUTE_REQUIREMENT_ROOT> --workflow <WORKFLOW_ID> --title <TITLE> --source-url <SOURCE_URL>

Constraints:
  workflow and requirement.title are required non-empty strings. requirement.source_url is optional.
  Use either --input or the typed flags, not both.

Controls:
  --root is required and is not part of Request JSON. --input accepts inline JSON, @file, or - for stdin.
  JSON input rejects unknown fields, type errors, invalid enums, and trailing JSON.
  --dry-run validates and plans without business side effects. --input and --dry-run are also Request-external controls.

Next step:
  After a successful non-dry-run initialization, run flow status immediately before acting.`
	command.Example = `  commonloop flow init --root <ABSOLUTE_REQUIREMENT_ROOT> --input @request.json --dry-run
  commonloop flow init --root <ABSOLUTE_REQUIREMENT_ROOT> --workflow <WORKFLOW_ID> --title <TITLE> --source-url <SOURCE_URL>
  commonloop flow status --root <ABSOLUTE_REQUIREMENT_ROOT>`
	command.RunE = func(command *cobra.Command, _ []string) error {
		return runOperation(command.Context(), command, *root, controls, ioStreams,
			func() (*flowidl.FlowInitRequest, *erroridl.PublicError) {
				if command.Flags().Changed("source-url") {
					request.Requirement.SourceUrl = &sourceURL
				}
				return request, nil
			},
			func(ctx context.Context, request *flowidl.FlowInitRequest, dryRun bool) (operationResult, *erroridl.PublicError) {
				runtime, warnings := flowRuntime()
				response, err := runtime.Init(ctx, *root, request, dryRun)
				return fromFlow(*root, response, warnings, err)
			})
	}
	addOperationControls(command, &controls)
	command.Flags().StringVar(&request.Workflow, "workflow", "", "Workflow id")
	command.Flags().StringVar(&request.Requirement.Title, "title", "", "requirement title")
	command.Flags().StringVar(&sourceURL, "source-url", "", "requirement source URL")
	return command
}

func newFlowStatusCommand(ioStreams streams, root *string) *cobra.Command {
	var controls operationControls
	command := operationCommand("flow.status")
	command.Long = `Purpose:
  Read the current Workflow state immediately before acting on a Requirement.

Effect:
  Read only. It returns the current Step's resolved Prompt and Skills, reusable Conditions, required Output schemas,
  all normal and return Routes, and accepted Outputs.

Request JSON:
  {}

Typed flags:
  commonloop flow status --root <ABSOLUTE_REQUIREMENT_ROOT>

Constraints:
  The business Request is empty. Evidence in the response is audit data and never affects Route matching.

Controls:
  --root is required and is not part of Request JSON. --input accepts {}, @file, or - for stdin.
  JSON input rejects unknown fields, type errors, invalid enums, and trailing JSON.
  This command does not support --dry-run because its business operation is read-only.

Next step:
  Execute current.prompt and its required Skills from this Status, then read the target report command's --help.`
	command.Example = "  commonloop flow status --root <ABSOLUTE_REQUIREMENT_ROOT>"
	command.RunE = func(command *cobra.Command, _ []string) error {
		return runOperation(command.Context(), command, *root, controls, ioStreams,
			func() (*flowidl.FlowStatusRequest, *erroridl.PublicError) { return flowidl.NewFlowStatusRequest(), nil },
			func(ctx context.Context, request *flowidl.FlowStatusRequest, _ bool) (operationResult, *erroridl.PublicError) {
				runtime, warnings := flowRuntime()
				response, err := runtime.Status(ctx, *root, request)
				return fromFlow(*root, response, warnings, err)
			})
	}
	addOperationControls(command, &controls)
	return command
}

func newFlowReportCommand(ioStreams streams, root *string) *cobra.Command {
	command := &cobra.Command{
		Use:   "report",
		Short: "Report progress or a Condition result",
		Long: `Report progress or a Condition result.

Run flow status immediately before reporting. Use progress for non-terminal
execution facts. Use result to submit ConditionResults, Evidence, and Summary;
the CLI validates Output schemas and selects one configured Flow or Loop Route.`,
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return command.Help()
		},
	}
	command.AddCommand(
		newFlowProgressCommand(ioStreams, root),
		newFlowResultCommand(ioStreams, root),
	)
	return command
}

func newFlowProgressCommand(ioStreams streams, root *string) *cobra.Command {
	var controls operationControls
	request := flowidl.NewFlowProgressRequest()
	var status string
	var evidence []string
	command := operationCommand("flow.report.progress")
	command.Long = `Purpose:
  Update execution facts for the current Step without moving the Workflow.

Effect:
  Local write. A non-dry-run call records the Step status, Summary, and Evidence; dry-run validates without committing them.

Request JSON:
  {
    "step_id": "<CURRENT_STEP_ID>",
    "status": "in_progress",
    "summary": "<SUMMARY>",
    "evidence": [
      {
        "source": "file",
        "content": "<EVIDENCE>",
        "ref": "<OPTIONAL_REF>"
      }
    ]
  }

Typed flags:
  commonloop flow report progress --root <ABSOLUTE_REQUIREMENT_ROOT> --step-id <CURRENT_STEP_ID> --status in_progress --summary <SUMMARY> --evidence '{"source":"file","content":"<EVIDENCE>","ref":"<OPTIONAL_REF>"}'

Constraints:
  step_id and status are required. step_id must equal latest Status current.context.step_id;
  status is in_progress, fixing, or blocked. summary is required.
  evidence may be omitted or empty; each item requires source and content, with optional ref.
  Evidence source is human, system, ai, file, or url. Use either --input or the typed flags, not both.

Controls:
  --root is required and is not part of Request JSON. --input accepts inline JSON, @file, or - for stdin.
  JSON input rejects unknown fields, type errors, invalid enums, and trailing JSON.
  --dry-run validates and plans without business side effects. --input and --dry-run are also Request-external controls.

Next step:
  After an accepted non-dry-run update, run flow status again before the next action.`
	command.Example = `  commonloop flow report progress --root <ABSOLUTE_REQUIREMENT_ROOT> --input @request.json --dry-run
  commonloop flow report progress --root <ABSOLUTE_REQUIREMENT_ROOT> --step-id <CURRENT_STEP_ID> --status in_progress --summary <SUMMARY>`
	command.RunE = func(command *cobra.Command, _ []string) error {
		typed := func() (*flowidl.FlowProgressRequest, *erroridl.PublicError) {
			var failure *erroridl.PublicError
			if request.Status, failure = progressStatus(status); failure != nil {
				return request, failure
			}
			if request.Evidence, failure = decodeFlowEvidence(evidence); failure != nil {
				return request, failure
			}
			return request, nil
		}
		return runOperation(command.Context(), command, *root, controls, ioStreams, typed,
			func(ctx context.Context, request *flowidl.FlowProgressRequest, dryRun bool) (operationResult, *erroridl.PublicError) {
				runtime, warnings := flowRuntime()
				response, err := runtime.Progress(ctx, *root, request, dryRun)
				return fromFlow(*root, response, warnings, err)
			})
	}
	addOperationControls(command, &controls)
	command.Flags().StringVar(&request.StepId, "step-id", "", "current Step id")
	command.Flags().StringVar(&status, "status", "", "in_progress, fixing, or blocked")
	command.Flags().StringVar(&request.Summary, "summary", "", "concise execution summary")
	command.Flags().StringArrayVar(&evidence, "evidence", nil, "Evidence JSON; repeatable")
	return command
}

func newFlowResultCommand(ioStreams streams, root *string) *cobra.Command {
	var controls operationControls
	request := flowidl.NewFlowResultRequest()
	var conditionResults, evidence []string
	var nextStepID, backStepID string
	var terminal bool
	command := operationCommand("flow.report.result")
	command.Long = `Purpose:
  Submit current Step Condition results and select one configured Flow, Loop, or terminal Route.

Effect:
  Local write. A non-dry-run call advances, loops, or completes the Workflow and commits State/Event facts.
  dry-run performs the same validation and route planning without committing facts.

Request JSON:
  {
    "step_id": "<CURRENT_STEP_ID>",
    "condition_results": [
      {
        "condition_id": "<CONDITION_ID>",
        "output": {
          "type": "string",
          "value": "<VALUE>"
        }
      }
    ],
    "evidence": [
      {
        "source": "file",
        "content": "<EVIDENCE>",
        "ref": "<OPTIONAL_REF>"
      }
    ],
    "summary": "<SUMMARY>",
    "route": {
      "next_step_id": "<NEXT_STEP_ID>"
    }
  }

Typed flags:
  commonloop flow report result --root <ABSOLUTE_REQUIREMENT_ROOT> --step-id <CURRENT_STEP_ID> --condition-result '{"condition_id":"<CONDITION_ID>","output":{"type":"string","value":"<VALUE>"}}' --evidence '{"source":"file","content":"<EVIDENCE>","ref":"<OPTIONAL_REF>"}' --summary <SUMMARY> --next-step-id <NEXT_STEP_ID>

Constraints:
  step_id is required and must equal latest Status current.context.step_id. Submit only current.conditions, using each listed Output type and constraints.
  condition_results is required and must be non-empty. Each condition_results item requires condition_id and output.type/value.
  evidence may be omitted or empty; each item requires source and content, with optional ref.
  Evidence source is human, system, ai, file, or url. Evidence never participates in Route matching; summary is required.
  route is required, must equal one current.available_routes[].route, and selects exactly one next_step_id, back_step_id, or terminal: true.
  Use either --input or the typed flags, not both. The CLI validates the configured Route; it does not verify business truth.

Controls:
  --root is required and is not part of Request JSON. --input accepts inline JSON, @file, or - for stdin.
  JSON input rejects unknown fields, type errors, invalid enums, and trailing JSON.
  --dry-run validates and plans without business side effects. --input and --dry-run are also Request-external controls.

Next step:
  After an accepted non-dry-run result, run flow status again before the next action.`
	command.Example = `  commonloop flow status --root <ABSOLUTE_REQUIREMENT_ROOT>
  commonloop flow report result --root <ABSOLUTE_REQUIREMENT_ROOT> --input @result.json --dry-run
  commonloop flow report result --root <ABSOLUTE_REQUIREMENT_ROOT> --input @result.json
  commonloop flow status --root <ABSOLUTE_REQUIREMENT_ROOT>`
	command.RunE = func(command *cobra.Command, _ []string) error {
		typed := func() (*flowidl.FlowResultRequest, *erroridl.PublicError) {
			var failure *erroridl.PublicError
			if request.ConditionResults, failure = decodeRepeated[*flowidl.ConditionResult](conditionResults); failure != nil {
				return request, failure
			}
			if request.Evidence, failure = decodeFlowEvidence(evidence); failure != nil {
				return request, failure
			}
			request.Route = flowidl.NewRouteSelection()
			if command.Flags().Changed("next-step-id") {
				request.Route.NextStepId = &nextStepID
			}
			if command.Flags().Changed("back-step-id") {
				request.Route.BackStepId = &backStepID
			}
			if command.Flags().Changed("terminal") {
				request.Route.Terminal = &terminal
			}
			return request, nil
		}
		return runOperation(command.Context(), command, *root, controls, ioStreams, typed,
			func(ctx context.Context, request *flowidl.FlowResultRequest, dryRun bool) (operationResult, *erroridl.PublicError) {
				runtime, warnings := flowRuntime()
				response, err := runtime.Result(ctx, *root, request, dryRun)
				return fromFlow(*root, response, warnings, err)
			})
	}
	addOperationControls(command, &controls)
	command.Flags().StringVar(&request.StepId, "step-id", "", "current Step id")
	command.Flags().StringArrayVar(&conditionResults, "condition-result", nil, "ConditionResult JSON; repeatable")
	command.Flags().StringArrayVar(&evidence, "evidence", nil, "Evidence JSON; repeatable")
	command.Flags().StringVar(&request.Summary, "summary", "", "concise result summary")
	command.Flags().StringVar(&nextStepID, "next-step-id", "", "matching Flow target Step id")
	command.Flags().StringVar(&backStepID, "back-step-id", "", "matching Loop target Step id")
	command.Flags().BoolVar(&terminal, "terminal", false, "select a matching terminal Flow Route")
	return command
}

func flowRuntime() (flow.Runtime, *[]string) {
	runtime := flow.DefaultRuntime()
	warnings := []string{}
	runtime.Warn = func(value string) {
		warnings = append(warnings, value)
	}
	return runtime, &warnings
}

func fromFlow(root string, data any, warnings *[]string, err error) (operationResult, *erroridl.PublicError) {
	if err != nil {
		return operationResult{}, flow.PublicError(err)
	}
	result := operationResult{data: data}
	if warnings != nil {
		result.warnings = append(result.warnings, (*warnings)...)
	}
	switch response := data.(type) {
	case *flowidl.FlowInitResponse:
		result.workflow = workflowRef(response.Workflow)
	case *flowidl.FlowStatusResponse:
		result.workflow = workflowRef(response.Workflow)
	default:
		result.workflow = reportWorkflowRef(root)
	}
	return result, nil
}

func workflowRef(value *commonidl.WorkflowRef) workflow.Ref {
	if value == nil {
		return workflow.Ref{}
	}
	return workflow.Ref{ID: value.Id, Digest: value.Digest}
}

func reportWorkflowRef(root string) workflow.Ref {
	local, failure := store.New(root)
	if failure != nil {
		return workflow.Ref{}
	}
	current, failure := local.Load()
	if failure != nil {
		return workflow.Ref{}
	}
	return current.Release.Workflow.Ref()
}

func decodeFlowEvidence(values []string) ([]*flowidl.Evidence, *erroridl.PublicError) {
	return decodeRepeated[*flowidl.Evidence](values)
}

func progressStatus(raw string) (flowidl.ProgressStatus, *erroridl.PublicError) {
	value, err := flowidl.ProgressStatusFromString(raw)
	if err != nil || value == flowidl.ProgressStatus_unspecified {
		return flowidl.ProgressStatus_unspecified, invalidArgument("--status must be in_progress, fixing, or blocked")
	}
	return value, nil
}
