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
	clierrs "github.com/zeefan1555/commonloop/errs"
	"github.com/zeefan1555/commonloop/internal/buildinfo"
	"github.com/zeefan1555/commonloop/internal/executionlog"
	"github.com/zeefan1555/commonloop/internal/idl"
	"github.com/zeefan1555/commonloop/internal/idl/commonidl"
	"github.com/zeefan1555/commonloop/internal/idl/storageidl"
	"github.com/zeefan1555/commonloop/internal/output"
	"github.com/zeefan1555/commonloop/internal/state"
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
		Use:   "commonloop",
		Short: "Backend requirement delivery CLI",
		Long: `Backend requirement delivery CLI.

Agent workflow:
1. Read the commonloop-workflow Skill before advancing a requirement.
2. Run commonloop flow status --root <ABSOLUTE_REQUIREMENT_ROOT> immediately before acting.
3. Execute current.prompt and its required Skills from that latest Status.
4. Read the target leaf command --help for its exact Request and controls.
5. Report progress or a Condition result, then read flow status again.

If flow status reports NOT_INITIALIZED, run flow init once and then read Status.`,
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
	update := &cobra.Command{
		Use:         "update",
		Short:       "Install the latest coordinated release",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{"bootstrap_control": "true"},
		RunE: func(*cobra.Command, []string) error {
			return errors.New("commonloop update requires the npm launcher; reinstall Commonloop with the official npm command")
		},
	}
	update.Flags().String("root", "", "")
	_ = update.Flags().MarkHidden("root")

	root.AddCommand(
		newFlowCommand(ioStreams, &requirementRoot),
		newTraceCommand(ioStreams, &requirementRoot),
		newCardCommand(ioStreams, &requirementRoot),
		newVersionCommand(ioStreams),
		newDoctorCommand(ioStreams, &requirementRoot),
		update,
		newInstallCommand(stdout, stderr),
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
