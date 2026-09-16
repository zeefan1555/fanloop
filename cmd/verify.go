package cmd

import (
	"context"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/idl/verifyidl"
	verifyruntime "github.com/zeefan1555/fanloop/internal/verify"
)

func newVerifyCommand(ioStreams streams, root *string) *cobra.Command {
	command := &cobra.Command{
		Use:   "verify",
		Short: "Drive public CLI verification and preserve evidence",
		Long: `Drive public CLI verification and preserve evidence.

Use doctor before a run, smoke for the fixed local journey, snapshot to capture one Requirement,
and cleanup to remove only resources owned by one verification run.`,
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error { return command.Help() },
	}
	command.AddCommand(
		newVerifyDoctorCommand(ioStreams),
		newVerifySmokeCommand(ioStreams),
		newVerifySnapshotCommand(ioStreams, root),
		newVerifyCleanupCommand(ioStreams),
	)
	return command
}

func newVerifyDoctorCommand(ioStreams streams) *cobra.Command {
	var controls operationControls
	command := operationCommand("verify.doctor")
	command.Long = `Purpose:
  Diagnose whether the running candidate can execute an isolated public-CLI verification run.

Effect:
  Read only. It checks the executable, data root, remote-binding isolation, version output, and installation Doctor.
  A blocked result is returned as structured output with a non-zero status.

Request JSON:
  {}

Typed flags:
  fanloop verify doctor

Constraints:
  The business Request is empty. The running executable must be readable and FANLOOP_DATA_HOME must resolve absolutely.

Controls:
  --input is not part of Request JSON. --input accepts {}, @file, or - for stdin.
  JSON input rejects unknown fields, type errors, invalid enums, and trailing JSON.
  This command does not support --dry-run because its business operation is read-only.

Next step:
  Resolve every failed check, then run fanloop verify smoke.`
	command.Example = "  fanloop verify doctor"
	command.RunE = func(command *cobra.Command, _ []string) error {
		return runOperation(command.Context(), command, "", controls, ioStreams,
			func() (*verifyidl.VerifyDoctorRequest, *erroridl.PublicError) {
				return verifyidl.NewVerifyDoctorRequest(), nil
			},
			func(ctx context.Context, request *verifyidl.VerifyDoctorRequest, _ bool) (operationResult, *erroridl.PublicError) {
				response, err := verifyruntime.DefaultRuntime().Doctor(ctx, request)
				result := operationResult{data: response}
				if response != nil && response.Outcome != verifyidl.VerificationOutcome_passed {
					result.partial, result.exitCode = true, 1
				}
				return result, normalizePublicErrorOrNil(err)
			})
	}
	addOperationControls(command, &controls)
	return command
}

func newVerifySmokeCommand(ioStreams streams) *cobra.Command {
	var controls operationControls
	request := verifyidl.NewVerifySmokeRequest()
	var evidenceDir string
	command := operationCommand("verify.smoke")
	command.Long = `Purpose:
  Drive the fixed local verification journey through the running public Fanloop CLI.

Effect:
  Local write. It creates an isolated data home and Requirement, records every command and durable side effect,
  captures a final snapshot, removes run-owned mutable resources, and preserves the evidence bundle.

Request JSON:
  {
    "evidence_dir": "<ABSOLUTE_EVIDENCE_DIRECTORY>"
  }

Typed flags:
  fanloop verify smoke [--evidence-dir <ABSOLUTE_EVIDENCE_DIRECTORY>]

Constraints:
  evidence_dir is optional. When omitted, evidence is stored under FANLOOP_DATA_HOME/verification/runs/<RUN_ID>.
  A supplied directory must be absolute, absent, and outside verification/work.

Controls:
  --input is not part of Request JSON. --input accepts inline JSON, @file, or - for stdin.
  JSON input rejects unknown fields, type errors, invalid enums, and trailing JSON.
  This command does not support --dry-run because the smoke journey proves both dry-run and real writes before cleanup.

Next step:
  Read report.md and manifest.json from the returned evidence_dir; a failed or blocked outcome retains its evidence.`
	command.Example = `  fanloop verify smoke
  fanloop verify smoke --evidence-dir <ABSOLUTE_EVIDENCE_DIRECTORY>`
	command.RunE = func(command *cobra.Command, _ []string) error {
		typed := func() (*verifyidl.VerifySmokeRequest, *erroridl.PublicError) {
			if command.Flags().Changed("evidence-dir") {
				request.EvidenceDir = &evidenceDir
			}
			return request, nil
		}
		return runOperation(command.Context(), command, "", controls, ioStreams, typed,
			func(ctx context.Context, request *verifyidl.VerifySmokeRequest, _ bool) (operationResult, *erroridl.PublicError) {
				response, err := verifyruntime.DefaultRuntime().Smoke(ctx, request)
				result := operationResult{data: response}
				if response != nil && response.Outcome != verifyidl.VerificationOutcome_passed {
					result.partial, result.exitCode = true, 1
				}
				return result, normalizePublicErrorOrNil(err)
			})
	}
	addOperationControls(command, &controls)
	command.Flags().StringVar(&evidenceDir, "evidence-dir", "", "absolute directory for the evidence bundle")
	return command
}

func newVerifySnapshotCommand(ioStreams streams, root *string) *cobra.Command {
	var controls operationControls
	request := verifyidl.NewVerifySnapshotRequest()
	var evidenceDir string
	command := operationCommand("verify.snapshot")
	command.Long = `Purpose:
  Capture public Status, Trace and Card reads plus the Requirement's durable files as verification evidence.

Effect:
  Local write. The Requirement is read only; the command writes a restricted evidence directory and normal CLI audit log.

Request JSON:
  {
    "evidence_dir": "<ABSOLUTE_SNAPSHOT_DIRECTORY>"
  }

Typed flags:
  fanloop verify snapshot --root <ABSOLUTE_REQUIREMENT_ROOT> [--evidence-dir <ABSOLUTE_SNAPSHOT_DIRECTORY>]

Constraints:
  The Requirement must be initialized. evidence_dir is optional; a supplied path must be absolute, absent,
  and outside the Requirement's .fanloop state directory.

Controls:
  --root is required and is not part of Request JSON. --input accepts inline JSON, @file, or - for stdin.
  JSON input rejects unknown fields, type errors, invalid enums, and trailing JSON.
  This command does not support --dry-run because it never changes Requirement business state.

Next step:
  Inspect the returned inventory and compare the public projections with the copied durable files.`
	command.Example = `  fanloop verify snapshot --root <ABSOLUTE_REQUIREMENT_ROOT>
  fanloop verify snapshot --root <ABSOLUTE_REQUIREMENT_ROOT> --evidence-dir <ABSOLUTE_SNAPSHOT_DIRECTORY>`
	command.RunE = func(command *cobra.Command, _ []string) error {
		typed := func() (*verifyidl.VerifySnapshotRequest, *erroridl.PublicError) {
			if command.Flags().Changed("evidence-dir") {
				request.EvidenceDir = &evidenceDir
			}
			return request, nil
		}
		return runOperation(command.Context(), command, *root, controls, ioStreams, typed,
			func(ctx context.Context, request *verifyidl.VerifySnapshotRequest, _ bool) (operationResult, *erroridl.PublicError) {
				response, err := verifyruntime.DefaultRuntime().Snapshot(ctx, *root, request)
				return operationResult{data: response, workflow: reportWorkflowRef(*root)}, normalizePublicErrorOrNil(err)
			})
	}
	addOperationControls(command, &controls)
	command.Flags().StringVar(&evidenceDir, "evidence-dir", "", "absolute directory for the snapshot")
	return command
}

func newVerifyCleanupCommand(ioStreams streams) *cobra.Command {
	var controls operationControls
	request := verifyidl.NewVerifyCleanupRequest()
	command := operationCommand("verify.cleanup")
	command.Long = `Purpose:
  Remove mutable resources owned by one verification run while preserving its evidence.

Effect:
  Local write. It removes only FANLOOP_DATA_HOME/verification/work/<RUN_ID>.
  A dry-run returns the exact path without deleting it.

Request JSON:
  {
    "run_id": "<RUN_ID>"
  }

Typed flags:
  fanloop verify cleanup --run-id <RUN_ID> [--dry-run]

Constraints:
  run_id is required and must be the 32-character lowercase hexadecimal identifier issued by verify smoke.

Controls:
  --input and --dry-run are not part of Request JSON. --input accepts inline JSON, @file, or - for stdin.
  JSON input rejects unknown fields, type errors, invalid enums, and trailing JSON.
  --dry-run reports the owned path without removing it. Use either --input or --run-id, not both.

Next step:
  Confirm removed is true, then read the preserved evidence manifest and report.`
	command.Example = `  fanloop verify cleanup --run-id <RUN_ID> --dry-run
  fanloop verify cleanup --run-id <RUN_ID>`
	command.RunE = func(command *cobra.Command, _ []string) error {
		typed := func() (*verifyidl.VerifyCleanupRequest, *erroridl.PublicError) {
			request.RunId = strings.TrimSpace(request.RunId)
			return request, nil
		}
		return runOperation(command.Context(), command, "", controls, ioStreams, typed,
			func(ctx context.Context, request *verifyidl.VerifyCleanupRequest, dryRun bool) (operationResult, *erroridl.PublicError) {
				response, err := verifyruntime.DefaultRuntime().Cleanup(ctx, dryRun, request)
				return operationResult{data: response}, normalizePublicErrorOrNil(err)
			})
	}
	addOperationControls(command, &controls)
	command.Flags().StringVar(&request.RunId, "run-id", "", "verification run id")
	return command
}
