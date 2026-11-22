package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	clierrs "github.com/zeefan1555/fanloop/errs"
	"github.com/zeefan1555/fanloop/internal/buildinfo"
	"github.com/zeefan1555/fanloop/internal/executionlog"
	"github.com/zeefan1555/fanloop/internal/idl"
	"github.com/zeefan1555/fanloop/internal/idl/commonidl"
	"github.com/zeefan1555/fanloop/internal/idl/storageidl"
	"github.com/zeefan1555/fanloop/internal/output"
	"github.com/zeefan1555/fanloop/internal/state"
)

type streams struct {
	in  io.Reader
	out io.Writer
	err io.Writer
}

func Execute(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) (exitCode int) {
	started := time.Now()
	arguments := append([]string{}, args...)
	var capturedStdin, capturedStdout, capturedStderr bytes.Buffer
	stdin = io.TeeReader(stdin, &capturedStdin)
	stdout = io.MultiWriter(stdout, &capturedStdout)
	stderr = io.MultiWriter(stderr, &capturedStderr)
	root := NewRoot(stdin, stdout, stderr)
	root.SetArgs(args)
	invocationCommand, invocationCommandID := executionCommand(root, args)
	var errorCode string
	defer func() {
		if invocationCommandID == "" || boolFlag(invocationCommand, "help") {
			return
		}
		requirementRoot, err := invocationCommand.Flags().GetString("root")
		if err != nil {
			return
		}
		requirementRoot = eligibleRoot(requirementRoot)
		if requirementRoot == "" {
			return
		}
		entry := &storageidl.CLIExecutionLogEntry{
			SchemaVersion:  storageidl.CLI_EXECUTION_LOG_SCHEMA_VERSION,
			InvocationId:   state.NewEventID(),
			StartedAt:      started.UTC().Format(time.RFC3339Nano),
			DurationMs:     time.Since(started).Milliseconds(),
			CommandId:      invocationCommandID,
			CliVersion:     buildinfo.CLIVersion,
			ReleaseVersion: buildinfo.ReleaseVersion,
			CommitSha:      buildinfo.Commit,
			DryRun:         boolFlag(invocationCommand, "dry-run"),
			ExitCode:       int32(exitCode),
			Arguments:      arguments,
			Stdin:          capturedStdin.String(),
			Stdout:         capturedStdout.String(),
			Stderr:         capturedStderr.String(),
		}
		if errorCode != "" {
			entry.ErrorCode = &errorCode
		}
		_ = executionlog.Append(requirementRoot, entry)
	}()
	if err := root.ExecuteContext(ctx); err != nil {
		var exitErr *clierrs.ExitError
		if errors.As(err, &exitErr) {
			errorCode = exitErr.ErrorCode
			return exitErr.Code
		}
		message := err.Error()
		if strings.HasPrefix(message, "unknown command ") {
			message = "command is not supported"
		} else if strings.HasPrefix(message, "unknown flag: ") {
			message = strings.Replace(message, "unknown flag: ", "unknown flag ", 1)
		}
		failure := invalidArgument(message)
		_ = writeFailure(stderr, failure, output.Meta{Command: commandID(args), RequirementRoot: rootArgument(args)})
		errorCode = failure.Code.String()
		return clierrs.ExitCode(failure)
	}
	return 0
}

func executionCommand(root *cobra.Command, args []string) (*cobra.Command, string) {
	command, _, _ := root.Find(args)
	if command == nil {
		return nil, ""
	}
	id := command.Annotations["command_id"]
	spec, ok := idl.LookupCommand(id)
	if !ok || spec.RequirementScope == commonidl.RequirementScope_none {
		return command, ""
	}
	return command, id
}

func boolFlag(command *cobra.Command, name string) bool {
	if command == nil || command.Flags().Lookup(name) == nil {
		return false
	}
	value, err := command.Flags().GetBool(name)
	return err == nil && value
}

func rootArgument(args []string) string {
	for index, argument := range args {
		if strings.HasPrefix(argument, "--root=") {
			return strings.TrimPrefix(argument, "--root=")
		}
		if argument == "--root" && index+1 < len(args) {
			return args[index+1]
		}
	}
	return ""
}

func eligibleRoot(root string) string {
	if !filepath.IsAbs(root) {
		return ""
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return ""
	}
	return root
}

func NewRoot(stdin io.Reader, stdout io.Writer, stderr io.Writer) *cobra.Command {
	ioStreams := streams{in: stdin, out: stdout, err: stderr}
	var requirementRoot string
	root := &cobra.Command{
		Use:   "fanloop",
		Short: "Legacy Requirement runtime and Workflow engine",
		Long: `Fanloop legacy Requirement runtime and general Workflow engine.

Current product workflows live in second-brain-agent/plugins and execute directly in Codex. This CLI continues
existing pinned Requirements and runs synthetic Runtime fixtures.

For an existing Requirement:
1. Use its bound-release-home/current/bin/fanloop controller.
2. Run flow status --root <ABSOLUTE_REQUIREMENT_ROOT> immediately before acting.
3. Execute current.prompt and its required Skills from that latest Status.
4. Report progress or a Condition result, then read flow status again.

Production routes are empty. Do not run flow init for a new product task.`,
		Version:           buildinfo.CLIVersion,
		SilenceErrors:     true,
		SilenceUsage:      true,
		Args:              cobra.NoArgs,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		RunE:              func(command *cobra.Command, _ []string) error { return command.Help() },
	}
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetVersionTemplate("{{.Version}}\n")
	root.PersistentFlags().StringVar(&requirementRoot, "root", "", "absolute requirement directory")

	root.AddCommand(
		newFlowCommand(ioStreams, &requirementRoot),
		newTraceCommand(ioStreams, &requirementRoot),
		newCardCommand(ioStreams, &requirementRoot),
		newConfigCommand(ioStreams),
		newVerifyCommand(ioStreams, &requirementRoot),
		newVersionCommand(ioStreams),
		newDoctorCommand(ioStreams, &requirementRoot),
		newDashboardCommand(ioStreams),
		newUpdateCommand(stdout, stderr),
		newInstallCommand(stdout, stderr),
		newActivateCurrentCommand(stderr),
	)
	return root
}

func commandID(args []string) string {
	parts := make([]string, 0, 2)
	for index := 0; index < len(args) && len(parts) < 2; index++ {
		if args[index] == "--root" {
			index++
			continue
		}
		if strings.HasPrefix(args[index], "--root=") || strings.HasPrefix(args[index], "-") {
			continue
		}
		parts = append(parts, args[index])
	}
	if len(parts) > 0 {
		return strings.Join(parts, ".")
	}
	return "unknown"
}
