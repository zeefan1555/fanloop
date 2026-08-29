package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/zeefan1555/commonloop/internal/idl/erroridl"
	"github.com/zeefan1555/commonloop/internal/idl/opsidl"
	"github.com/zeefan1555/commonloop/internal/ops"
)

func newDoctorCommand(ioStreams streams, root *string) *cobra.Command {
	var controls operationControls
	command := operationCommand("doctor")
	command.Long = `Purpose:
  Diagnose either the Commonloop installation or one initialized Requirement.

Effect:
  Read only. Without --root it checks installation health; with --root it also checks that Requirement.
  An unhealthy result is returned as structured output with a non-zero status.

Request JSON:
  {}

Typed flags:
  commonloop doctor [--root <ABSOLUTE_REQUIREMENT_ROOT>]

Constraints:
  The business Request is empty. --root is optional; when present it must name the Requirement to diagnose.

Controls:
  --root and --input are not part of Request JSON. --input accepts {}, @file, or - for stdin.
  JSON input rejects unknown fields, type errors, invalid enums, and trailing JSON.
  This command does not support --dry-run because its business operation is read-only.

Next step:
  Follow any failed check hint, then rerun doctor to confirm the installation or Requirement is healthy.`
	command.Example = `  commonloop doctor
  commonloop doctor --root <ABSOLUTE_REQUIREMENT_ROOT>`
	command.RunE = func(command *cobra.Command, _ []string) error {
		return runOperation(command.Context(), command, *root, controls, ioStreams,
			func() (*opsidl.DoctorRequest, *erroridl.PublicError) { return opsidl.NewDoctorRequest(), nil },
			func(ctx context.Context, request *opsidl.DoctorRequest, _ bool) (operationResult, *erroridl.PublicError) {
				response, err := ops.DefaultRuntime().Doctor(ctx, *root, request)
				result := operationResult{data: response}
				if response != nil && response.Status == opsidl.DoctorStatus_unhealthy {
					result.partial, result.exitCode = true, 1
				}
				return result, normalizePublicErrorOrNil(err)
			})
	}
	addOperationControls(command, &controls)
	return command
}
