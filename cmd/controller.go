package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/zeefan1555/fanloop/errs"
	"github.com/zeefan1555/fanloop/internal/controller"
	"github.com/zeefan1555/fanloop/internal/idl"
	"github.com/zeefan1555/fanloop/internal/idl/commonidl"
	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/output"
	"github.com/zeefan1555/fanloop/internal/release"
	"github.com/zeefan1555/fanloop/internal/store"
)

func ExecuteProcess(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return execute(ctx, args, stdin, stdout, stderr, func(capturedStderr io.Writer) error {
		return dispatchController(args, capturedStderr)
	})
}

func dispatchController(args []string, stderr io.Writer) error {
	selection, err := ControllerSelection(args)
	if err != nil {
		return err
	}
	if selection != nil {
		executable, err := os.Executable()
		if err != nil {
			return err
		}
		physical, err := filepath.EvalSymlinks(executable)
		if err != nil {
			return err
		}
		if physical != selection.Binary {
			if err := syscall.Exec(selection.Binary, append([]string{selection.Binary}, args...), selection.Environment); err != nil {
				return fmt.Errorf("cannot execute controller: %w", err)
			}
		}
		if err := os.Unsetenv("FANLOOP_CONFIG_SOURCE"); err != nil {
			return err
		}
		if selection.AuthorLive {
			fmt.Fprintln(stderr, "fanloop: author/live configuration remains editable; use explicit protect to freeze it")
		}
		for _, item := range selection.Environment {
			name, value, _ := strings.Cut(item, "=")
			if err := os.Setenv(name, value); err != nil {
				return err
			}
		}
	}
	return nil
}

func controllerInvocation(args []string) (string, bool) {
	root := NewRoot(strings.NewReader(""), io.Discard, io.Discard)
	command, remaining, err := root.Find(args)
	if err != nil || command == nil {
		return "", false
	}
	command.InitDefaultHelpFlag()
	if err := command.ParseFlags(remaining); err != nil {
		return "", false
	}
	if boolFlag(command, "help") {
		return "", false
	}
	spec, ok := idl.LookupCommand(command.Annotations["command_id"])
	if !ok || spec.RequirementScope == commonidl.RequirementScope_none {
		return "", false
	}
	path, err := command.Flags().GetString("root")
	if err != nil || !filepath.IsAbs(path) {
		return "", false
	}
	return path, spec.ID == "config.rebind"
}

func ControllerSelection(args []string) (*controller.Selection, error) {
	path, rebind := controllerInvocation(args)
	if path == "" {
		return nil, nil
	}
	caller := ""
	if rebind {
		var err error
		caller, err = release.DefaultDataRoot()
		if err != nil {
			return nil, err
		}
	}
	return controller.Select(path, caller)
}

func newProtectCommand(stdout, stderr io.Writer) *cobra.Command {
	var source string
	command := &cobra.Command{Use: "__protect <absolute-root>", Hidden: true, Args: cobra.ExactArgs(1), RunE: func(command *cobra.Command, args []string) error {
		local, failure := store.New(args[0])
		if failure != nil {
			return failure
		}
		binary, err := local.ProtectController(command.Context(), source, nil)
		if err != nil {
			failure := errs.NewCode(erroridl.ErrorCode_INTERNAL, err.Error(), nil)
			_ = output.FailureWithNotice(stderr, failure, output.Meta{Command: "__protect", RequirementRoot: args[0]}, commonidl.Notice{})
			return &errs.ExitError{Code: 5, ErrorCode: failure.Code.String()}
		}
		_, err = fmt.Fprintln(stdout, binary)
		return err
	}}
	command.Flags().StringVar(&source, "source", "", "explicit physical source Release")
	return command
}

type LauncherCommand struct {
	Scope string          `json:"scope"`
	Flags map[string]bool `json:"flags"`
}

func LauncherCommands() map[string]LauncherCommand {
	root := NewRoot(strings.NewReader(""), io.Discard, io.Discard)
	result := map[string]LauncherCommand{}
	var walk func(*cobra.Command)
	walk = func(command *cobra.Command) {
		id := command.Annotations["command_id"]
		if spec, ok := idl.LookupCommand(id); ok {
			command.InitDefaultHelpFlag()
			command.InheritedFlags()
			flags := map[string]bool{}
			command.Flags().VisitAll(func(flag *pflag.Flag) {
				flags["--"+flag.Name] = flag.NoOptDefVal != ""
				if flag.Shorthand != "" {
					flags["-"+flag.Shorthand] = flag.NoOptDefVal != ""
				}
			})
			command.InheritedFlags().VisitAll(func(flag *pflag.Flag) { flags["--"+flag.Name] = flag.NoOptDefVal != "" })
			result[strings.ReplaceAll(id, ".", " ")] = LauncherCommand{Scope: spec.RequirementScope.String(), Flags: flags}
		}
		for _, child := range command.Commands() {
			walk(child)
		}
	}
	walk(root)
	return result
}
