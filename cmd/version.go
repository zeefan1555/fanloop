package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/zeefan1555/commonloop/internal/idl/erroridl"
	"github.com/zeefan1555/commonloop/internal/idl/opsidl"
	"github.com/zeefan1555/commonloop/internal/ops"
)

func newVersionCommand(ioStreams streams) *cobra.Command {
	var controls operationControls
	command := operationCommand("version")
	command.Long = `Purpose:
  Show the active coordinated Release, commit, State Schema support, Skills, and Workflows.

Effect:
  Read only. It inspects installed release metadata and does not change local or remote state.

Request JSON:
  {}

Typed flags:
  commonloop version

Constraints:
  The business Request is empty.

Controls:
  This command does not require --root. --input accepts {}, @file, or - for stdin and is not part of Request JSON.
  JSON input rejects unknown fields, type errors, invalid enums, and trailing JSON.
  This command does not support --dry-run because it has no side effects.

Next step:
  Compare the returned release components, or run doctor when installation health must be checked.`
	command.Example = "  commonloop version"
	command.RunE = func(command *cobra.Command, _ []string) error {
		return runOperation(command.Context(), command, "", controls, ioStreams,
			func() (*opsidl.VersionRequest, *erroridl.PublicError) { return opsidl.NewVersionRequest(), nil },
			func(ctx context.Context, request *opsidl.VersionRequest, _ bool) (operationResult, *erroridl.PublicError) {
				response, err := ops.DefaultRuntime().Version(ctx, request)
				return operationResult{data: response}, normalizePublicErrorOrNil(err)
			})
	}
	addOperationControls(command, &controls)
	return command
}
