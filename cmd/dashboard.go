package cmd

import (
	"context"
	"net"
	"net/http"

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
  Local write. The foreground server listens only on 127.0.0.1 with an OS-selected port and displays synchronized Fornax configuration under current/workflows as read-only.

Request JSON:
  {}

Typed flags:
  fanloop dashboard

Constraints:
  The business Request is empty. The command does not create, delete, or rename Workflows or Skills.

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
		root, err := workflow.DefaultRoot()
		if err != nil {
			return dashboardFailure(ioStreams, err)
		}

		dataRoot, err := release.DefaultDataRoot()
		if err != nil {
			return dashboardFailure(ioStreams, err)
		}
		unlock, err := workflow.PrepareConfiguration(command.Context(), dataRoot)
		if err != nil {
			return dashboardFailure(ioStreams, err)
		}
		handler, err := dashboard.NewHandler(root)
		unlock()
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

func dashboardFailure(ioStreams streams, err error) error {
	failure := normalizePublicError(err)
	_ = writeFailure(ioStreams.err, failure, output.Meta{Command: "dashboard"})
	return &clierrs.ExitError{Code: clierrs.ExitCode(failure), ErrorCode: failure.Code.String()}
}
