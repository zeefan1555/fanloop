package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/zeefan1555/commonloop/internal/idl/erroridl"
	"github.com/zeefan1555/commonloop/internal/idl/traceidl"
	traceruntime "github.com/zeefan1555/commonloop/internal/trace"
)

func newTraceCommand(ioStreams streams, root *string) *cobra.Command {
	command := &cobra.Command{
		Use:   "trace",
		Short: "Inspect local audit facts and project them to configured Lark targets",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return command.Help()
		},
	}
	command.AddCommand(
		newTraceBindCommand(ioStreams, root),
		newTraceStatusCommand(ioStreams, root),
		newTraceRenderCommand(ioStreams, root),
		newTraceSyncCommand(ioStreams, root),
	)
	return command
}

func newTraceBindCommand(ioStreams streams, root *string) *cobra.Command {
	var controls operationControls
	request := traceidl.NewTraceBindRequest()
	registry := "production"
	cliLogDocumentURL := ""
	command := operationCommand("trace.bind")
	command.Long = `Purpose:
  Bind one Trace document and Registry profile to an initialized Requirement.

Effect:
  Local write. A non-dry-run call records the binding in local State/Event facts; dry-run validates without committing it.

Request JSON:
  {
    "document_url": "<TRACE_DOCUMENT_URL>",
    "registry": "production",
    "cli_log_document_url": "<CLI_LOG_DOCUMENT_URL>"
  }

Typed flags:
  commonloop trace bind --root <ABSOLUTE_REQUIREMENT_ROOT> --document-url <TRACE_DOCUMENT_URL> --registry production --cli-log-document-url <CLI_LOG_DOCUMENT_URL>

Constraints:
  document_url is required and must be an allowed HTTP URL. registry is optional and must be production or test.
  cli_log_document_url must be present exactly when the selected Workflow/profile Registry policy requires it.
  After the first successful bind, the same document bundle and Registry are idempotent; a different binding is rejected.
  The typed flag defaults registry to production. Use either --input or the typed flags, not both.

Controls:
  --root is required and is not part of Request JSON. --input accepts inline JSON, @file, or - for stdin.
  JSON input rejects unknown fields, type errors, invalid enums, and trailing JSON.
  --dry-run validates and plans without business side effects. --input and --dry-run are also Request-external controls.

Next step:
  Run trace status to confirm the accepted binding before rendering or syncing.`
	command.Example = `  commonloop trace bind --root <ABSOLUTE_REQUIREMENT_ROOT> --input @request.json --dry-run
  commonloop trace bind --root <ABSOLUTE_REQUIREMENT_ROOT> --document-url <TRACE_DOCUMENT_URL> --registry production --cli-log-document-url <CLI_LOG_DOCUMENT_URL>`
	command.RunE = func(command *cobra.Command, _ []string) error {
		return runOperation(command.Context(), command, *root, controls, ioStreams,
			func() (*traceidl.TraceBindRequest, *erroridl.PublicError) {
				request.Registry = &registry
				if cliLogDocumentURL != "" {
					request.CliLogDocumentUrl = &cliLogDocumentURL
				}
				return request, nil
			},
			func(ctx context.Context, request *traceidl.TraceBindRequest, dryRun bool) (operationResult, *erroridl.PublicError) {
				response, err := traceruntime.DefaultRuntime().Bind(ctx, *root, request, dryRun)
				return fromTrace(*root, response, err)
			})
	}
	addOperationControls(command, &controls)
	command.Flags().StringVar(&request.DocumentUrl, "document-url", "", "Trace document URL")
	command.Flags().StringVar(&registry, "registry", "production", "fixed Registry profile: production or test")
	command.Flags().StringVar(&cliLogDocumentURL, "cli-log-document-url", "", "stable CLI log document URL when required by the selected Trace Registry policy")
	return command
}

func newTraceStatusCommand(ioStreams streams, root *string) *cobra.Command {
	var controls operationControls
	command := operationCommand("trace.status")
	command.Long = `Purpose:
  Show the Trace binding and projection status for an initialized Requirement.

Effect:
  Read only. It returns the accepted Trace/CLI-log document binding, Registry, local Event count, and latest sync result.

Request JSON:
  {}

Typed flags:
  commonloop trace status --root <ABSOLUTE_REQUIREMENT_ROOT>

Constraints:
  The business Request is empty.

Controls:
  --root is required and is not part of Request JSON. --input accepts {}, @file, or - for stdin.
  JSON input rejects unknown fields, type errors, invalid enums, and trailing JSON.
  This command does not support --dry-run because its business operation is read-only.

Next step:
  Use the returned binding and projection status to decide whether to bind, render, or sync.`
	command.Example = "  commonloop trace status --root <ABSOLUTE_REQUIREMENT_ROOT>"
	command.RunE = func(command *cobra.Command, _ []string) error {
		return runOperation(command.Context(), command, *root, controls, ioStreams,
			func() (*traceidl.TraceStatusRequest, *erroridl.PublicError) {
				return traceidl.NewTraceStatusRequest(), nil
			},
			func(ctx context.Context, request *traceidl.TraceStatusRequest, _ bool) (operationResult, *erroridl.PublicError) {
				response, err := traceruntime.DefaultRuntime().Status(ctx, *root, request)
				return fromTrace(*root, response, err)
			})
	}
	addOperationControls(command, &controls)
	return command
}

func newTraceRenderCommand(ioStreams streams, root *string) *cobra.Command {
	var controls operationControls
	command := operationCommand("trace.render")
	command.Long = `Purpose:
  Rebuild the local human-readable Events projection for an initialized Requirement.

Effect:
  Local write. A non-dry-run call writes the Events projection; dry-run computes its event count and path without writing.

Request JSON:
  {}

Typed flags:
  commonloop trace render --root <ABSOLUTE_REQUIREMENT_ROOT>

Constraints:
  The business Request is empty. Rendering uses committed local State/Event facts only.

Controls:
  --root is required and is not part of Request JSON. --input accepts {}, @file, or - for stdin.
  JSON input rejects unknown fields, type errors, invalid enums, and trailing JSON.
  --dry-run validates and renders without business side effects. --input and --dry-run are Request-external controls.

Next step:
  Inspect the returned projection path, or run trace sync when configured remote projections must be updated.`
	command.Example = `  commonloop trace render --root <ABSOLUTE_REQUIREMENT_ROOT> --dry-run
  commonloop trace render --root <ABSOLUTE_REQUIREMENT_ROOT>`
	command.RunE = func(command *cobra.Command, _ []string) error {
		return runOperation(command.Context(), command, *root, controls, ioStreams,
			func() (*traceidl.TraceRenderRequest, *erroridl.PublicError) {
				return traceidl.NewTraceRenderRequest(), nil
			},
			func(ctx context.Context, request *traceidl.TraceRenderRequest, dryRun bool) (operationResult, *erroridl.PublicError) {
				response, err := traceruntime.DefaultRuntime().Render(ctx, *root, request, dryRun)
				return fromTrace(*root, response, err)
			})
	}
	addOperationControls(command, &controls)
	return command
}

func newTraceSyncCommand(ioStreams streams, root *string) *cobra.Command {
	var controls operationControls
	command := operationCommand("trace.sync")
	command.Long = `Purpose:
  Sync committed Requirement facts to the configured remote projections.

Effect:
  External write. A non-dry-run call updates the Trace document and Registry; policies with a bound CLI-log document also update
  that document with every byte already present in .commonloop/log/cli.jsonl, without redaction or truncation.
  The response may be partial when one target succeeds and another fails. dry-run reports skipped targets without remote writes.

Request JSON:
  {}

Typed flags:
  commonloop trace sync --root <ABSOLUTE_REQUIREMENT_ROOT>

Constraints:
  The business Request is empty. A Trace binding is required for remote updates.

Controls:
  --root is required and is not part of Request JSON. --input accepts {}, @file, or - for stdin.
  JSON input rejects unknown fields, type errors, invalid enums, and trailing JSON.
  --dry-run validates without external side effects. --input and --dry-run are Request-external controls.

Next step:
  Read every target result, including partial failures, then run trace status to inspect the recorded outcome.`
	command.Example = `  commonloop trace sync --root <ABSOLUTE_REQUIREMENT_ROOT> --dry-run
  commonloop trace sync --root <ABSOLUTE_REQUIREMENT_ROOT>`
	command.RunE = func(command *cobra.Command, _ []string) error {
		return runOperation(command.Context(), command, *root, controls, ioStreams,
			func() (*traceidl.TraceSyncRequest, *erroridl.PublicError) { return traceidl.NewTraceSyncRequest(), nil },
			func(ctx context.Context, request *traceidl.TraceSyncRequest, dryRun bool) (operationResult, *erroridl.PublicError) {
				response, err := traceruntime.DefaultRuntime().Sync(ctx, *root, request, dryRun)
				return fromTrace(*root, response, err)
			})
	}
	addOperationControls(command, &controls)
	return command
}

func fromTrace(root string, data any, err error) (operationResult, *erroridl.PublicError) {
	if err != nil {
		return operationResult{}, normalizePublicError(err)
	}
	result := operationResult{data: data, workflow: reportWorkflowRef(root)}
	if response, ok := data.(*traceidl.TraceSyncResponse); ok && response.Outcome == traceidl.TraceSyncOutcome_partial {
		result.partial, result.exitCode = true, 1
	}
	return result, nil
}
