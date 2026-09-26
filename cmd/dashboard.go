package cmd

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	clierrs "github.com/zeefan1555/fanloop/errs"
	"github.com/zeefan1555/fanloop/internal/dashboard"
	"github.com/zeefan1555/fanloop/internal/idl/dashboardidl"
	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/output"
	"github.com/zeefan1555/fanloop/internal/release"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

func newDashboardCommand(ioStreams streams) *cobra.Command {
	var controls operationControls
	command := operationCommand("dashboard")
	command.Long = `Purpose:
  Start the local Workflow configuration dashboard.

Effect:
  Local write. The foreground server listens only on 127.0.0.1. Published configuration is read-only;
  an explicit FANLOOP_CONFIG_SOURCE opens an isolated candidate configuration for whole-file editing.

Request JSON:
  {}

Typed flags:
  fanloop dashboard

Constraints:
  The business Request is empty. The command does not create, delete, or rename Workflows or Skills.
  FANLOOP_CONFIG_SOURCE must be an absolute isolated root containing workflows and cannot resolve to the live data root.

Controls:
  --input accepts {}, @file, or - for stdin and is not part of Request JSON.
  This command does not support --root or --dry-run.

Next step:
  Open the returned loopback URL. Stop the foreground process when viewing is complete.`
	command.Example = "  fanloop dashboard"
	command.RunE = func(command *cobra.Command, _ []string) error {
		request, failure := commandRequest(command, controls.input, ioStreams.in, func() (*dashboardidl.DashboardStartRequest, *erroridl.PublicError) {
			return dashboardidl.NewDashboardStartRequest(), nil
		})
		if failure != nil {
			_ = writeFailure(ioStreams.err, failure, output.Meta{Command: "dashboard"})
			return &clierrs.ExitError{Code: clierrs.ExitCode(failure), ErrorCode: failure.Code.String()}
		}
		if err := request.IsValid(); err != nil {
			failure = invalidArgument(err.Error())
			_ = writeFailure(ioStreams.err, failure, output.Meta{Command: "dashboard"})
			return &clierrs.ExitError{Code: clierrs.ExitCode(failure), ErrorCode: failure.Code.String()}
		}
		dataRoot, err := release.DefaultDataRoot()
		if err != nil {
			return dashboardFailure(ioStreams, err)
		}
		root := filepath.Join(dataRoot, "current", "workflows")
		editable, lockRoot := false, dataRoot
		if source := os.Getenv("FANLOOP_CONFIG_SOURCE"); source != "" {
			resolved, err := workflow.ValidateDashboardCandidate(dataRoot, source)
			if err != nil {
				return dashboardFailure(ioStreams, err)
			}
			root, editable, lockRoot = filepath.Join(resolved, "workflows"), true, ""
		}
		handler, err := newDashboardHandler(root, editable, lockRoot)
		if err != nil {
			return dashboardFailure(ioStreams, err)
		}
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return dashboardFailure(ioStreams, err)
		}
		defer listener.Close()
		response := &dashboardidl.DashboardStartResponse{Url: "http://" + listener.Addr().String()}
		if err := response.IsValid(); err != nil {
			return dashboardFailure(ioStreams, err)
		}
		if err := output.SuccessWithNotice(ioStreams.out, response, output.Meta{Command: "dashboard"}, localNotice()); err != nil {
			return &clierrs.ExitError{Code: 5}
		}
		server := &http.Server{Handler: handler}
		go func(ctx context.Context) {
			<-ctx.Done()
			_ = server.Shutdown(context.Background())
		}(command.Context())
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			return dashboardFailure(ioStreams, err)
		}
		return nil
	}
	addOperationControls(command, &controls)
	return command
}

func newDashboardHandler(root string, editable bool, lockRoot string) (http.Handler, error) {
	if lockRoot == "" {
		return dashboard.NewHandler(root, editable, lockRoot)
	}
	unlock, err := workflow.LockConfiguration(lockRoot, false)
	if err != nil {
		return nil, err
	}
	defer unlock()
	return dashboard.NewHandler(root, editable, lockRoot)
}

func dashboardFailure(ioStreams streams, err error) error {
	failure := normalizePublicError(err)
	_ = writeFailure(ioStreams.err, failure, output.Meta{Command: "dashboard"})
	return &clierrs.ExitError{Code: clierrs.ExitCode(failure), ErrorCode: failure.Code.String()}
}
