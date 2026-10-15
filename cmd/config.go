package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/idl/opsidl"
	"github.com/zeefan1555/fanloop/internal/ops"
)

func newConfigCommand(ioStreams streams) *cobra.Command {
	command := &cobra.Command{
		Use:   "config",
		Short: "Inspect Workflow configuration",
		Args:  cobra.NoArgs,
		RunE:  func(command *cobra.Command, _ []string) error { return command.Help() },
	}
	command.AddCommand(newConfigDriftCommand(ioStreams))
	return command
}

func newConfigDriftCommand(ioStreams streams) *cobra.Command {
	var controls operationControls
	command := operationCommand("config.drift")
	command.Long = `Purpose:
  Compare the active Workflow snapshot with the latest released Fornax packages.

Effect:
  Read only. It queries every source in fornax.json through the user's logged-in fornax-cli,
  downloads released packages into a private temporary directory, and does not change the snapshot, Fornax, or Requirements.
  The released package is the reference. Equal file bytes are in_sync. An older recorded version whose tree still matches its receipt is local_behind.
  Any other byte difference is local_modified. Discovery or download failure is unavailable.
  Unix file modes are ignored.

Request JSON:
  {}

Typed flags:
  fanloop config drift

Constraints:
  The business Request is empty. The command reads the managed config/current snapshot and the credential-free fornax.json.
  FANLOOP_CONFIG_SOURCE author overrides are not accepted.

Controls:
  This command does not require --root. --input accepts {}, @file, or - for stdin and is not part of Request JSON.
  JSON input rejects unknown fields, type errors, invalid enums, and trailing JSON.
  This command does not support --dry-run because it has no persistent side effects.

Next step:
  An in_sync result is exit 0. local_behind, local_modified, and unavailable exit 1.`
	command.Example = "  fanloop config drift"
	command.RunE = func(command *cobra.Command, _ []string) error {
		return runOperation(command.Context(), command, "", controls, ioStreams,
			func() (*opsidl.ConfigDriftRequest, *erroridl.PublicError) { return opsidl.NewConfigDriftRequest(), nil },
			func(ctx context.Context, request *opsidl.ConfigDriftRequest, _ bool) (operationResult, *erroridl.PublicError) {
				response, err := ops.DefaultRuntime().ConfigDrift(ctx, request)
				result := operationResult{data: response}
				if response != nil && response.Status != opsidl.ConfigDriftStatus_in_sync {
					result.partial, result.exitCode = true, 1
				}
				return result, normalizePublicErrorOrNil(err)
			})
	}
	addOperationControls(command, &controls)
	return command
}
