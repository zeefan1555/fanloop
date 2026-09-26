package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/idl/opsidl"
	"github.com/zeefan1555/fanloop/internal/ops"
)

func newUpdateCommand(ioStreams streams) *cobra.Command {
	var controls operationControls
	command := operationCommand("update")
	command.Long = `Purpose:
  Synchronize the complete published Workflow configuration from Fornax.

Effect:
  Local write. It stages and validates all configured Workflows, then atomically replaces current/workflows.

Request JSON:
  {}

Typed flags:
  fanloop update

Constraints:
  The business Request is empty. FANLOOP_CONFIG_SOURCE is rejected because update targets production configuration.

Controls:
  This command does not require --root. --input accepts {}, @file, or - from stdin and is not part of Request JSON.
  JSON input rejects unknown fields, type errors, and trailing JSON. This command does not support --dry-run.

Next step:
  Run fanloop doctor, then continue ordinary commands from the synchronized local configuration.`
	command.Example = "  fanloop update\n  fanloop doctor"
	command.RunE = func(command *cobra.Command, _ []string) error {
		return runOperation(command.Context(), command, "", controls, ioStreams,
			func() (*opsidl.ConfigurationUpdateRequest, *erroridl.PublicError) {
				return opsidl.NewConfigurationUpdateRequest(), nil
			},
			func(ctx context.Context, request *opsidl.ConfigurationUpdateRequest, _ bool) (operationResult, *erroridl.PublicError) {
				response, err := ops.DefaultRuntime().Update(ctx, request)
				return operationResult{data: response}, normalizePublicErrorOrNil(err)
			})
	}
	addOperationControls(command, &controls)
	return command
}
