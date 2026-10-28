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
	command.AddCommand(newConfigRebindCommand(ioStreams))
	return command
}

func newConfigRebindCommand(ioStreams streams) *cobra.Command {
	var controls operationControls
	request := opsidl.NewConfigRebindRequest()
	command := operationCommand("config.rebind")
	command.Long = `Purpose:
  Rebind one initialized requirement to an installed Workflow configuration snapshot.

Effect:
  Local write. Validates the current State and complete Event history against the target snapshot, then atomically updates
  .fanloop/configuration-snapshot and appends .fanloop/audit/configuration-rebindings.jsonl under the requirement lock.
  The audit record contains the previous and new snapshot identities, UTC time, and required reason.
  For a bound Requirement, the pinned controller interprets State and keeps its four Skill roots. This command alone retains the validated caller FANLOOP_DATA_HOME for target selection and injects the actual old pin. The next ordinary invocation reads the new actual pin, not historical controller config/current or controller journal intent.

Request JSON:
  {"reason":"why this requirement must use the new snapshot","snapshot":"optional-installed-snapshot-name"}

Typed flags:
  fanloop config rebind --root /absolute/requirement --reason "..." [--snapshot <installed-name>]

Constraints:
  Omit snapshot to bind config/current. An explicit snapshot is a directory name already installed under
  FANLOOP_DATA_HOME/config/snapshots; arbitrary paths are rejected. The target must contain the bound Workflow
  and must validate the existing State and Event history. Use either --input or typed flags, not both.
  JSON input rejects unknown fields, type errors, invalid enums, and trailing JSON.

Controls:
  --root is required and must name an initialized requirement. This command has local persistent side effects
  and does not support --dry-run.

Next step:
  Read fanloop flow status --root <ABSOLUTE_REQUIREMENT_ROOT> and use only the absolute Skill paths it returns.`
	command.Example = `  fanloop config rebind --root /work/requirement --reason "adopt the repaired configuration"
  fanloop config rebind --root /work/requirement --snapshot 0123456789abcdef0123456789abcdef01234567 --reason "pin reviewed snapshot"`
	command.RunE = func(command *cobra.Command, _ []string) error {
		root, _ := command.Flags().GetString("root")
		return runOperation(command.Context(), command, root, controls, ioStreams,
			func() (*opsidl.ConfigRebindRequest, *erroridl.PublicError) { return request, nil },
			func(ctx context.Context, request *opsidl.ConfigRebindRequest, _ bool) (operationResult, *erroridl.PublicError) {
				response, err := ops.DefaultRuntime().ConfigRebind(ctx, root, request)
				return operationResult{data: response}, normalizePublicErrorOrNil(err)
			})
	}
	command.Flags().StringVar(&request.Reason, "reason", "", "required audit reason")
	var snapshot string
	command.Flags().StringVar(&snapshot, "snapshot", "", "installed snapshot name; defaults to config/current")
	command.PreRunE = func(command *cobra.Command, _ []string) error {
		if command.Flags().Changed("snapshot") {
			request.Snapshot = &snapshot
		}
		return nil
	}
	addOperationControls(command, &controls)
	return command
}

func newConfigDriftCommand(ioStreams streams) *cobra.Command {
	var controls operationControls
	command := operationCommand("config.drift")
	command.Long = `Purpose:
  Compare the active Workflow snapshot with second-brain-skills main.

Effect:
  Read only. It reads the managed config/current snapshot, checks every manifest file sha256, and runs git ls-remote on second-brain-skills main.
  It does not change the snapshot or Requirements. Equal content and an equal commit are in_sync. A different remote commit is local_behind.
  Any byte difference is local_modified with detail "package content differs". An ls-remote failure is unavailable.
  A Fornax-origin snapshot is local_behind with detail "snapshot came from Fornax; run fanloop update config".
  Unix file modes are ignored. A missing fornax.json is ignored.
  FANLOOP_USE_FORNAX=1, true, or yes, together with a credential-free fornax.json, compares the snapshot with the latest Fornax release instead.
  That path queries every source through the user's logged-in fornax-cli and downloads released packages into a private temporary directory.

Request JSON:
  {}

Typed flags:
  fanloop config drift

Constraints:
  The business Request is empty. The command reads the managed config/current snapshot. A credential-free fornax.json is used only with FANLOOP_USE_FORNAX.
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
