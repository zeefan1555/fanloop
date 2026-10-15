package cmd

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/spf13/cobra"
	clierrs "github.com/zeefan1555/fanloop/errs"
	"github.com/zeefan1555/fanloop/internal/configuration"
	"github.com/zeefan1555/fanloop/internal/distribution"
	"github.com/zeefan1555/fanloop/internal/release"
)

var (
	applyReleaseUpdate = func(ctx context.Context, dataRoot string) (distribution.Result, error) {
		return distribution.Update(ctx, distribution.Options{DataRoot: dataRoot})
	}
	applyConfigSync = configuration.Sync
)

func newUpdateCommand(stdout, stderr io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:   "update",
		Short: "Install the latest GitHub Release of the Fanloop CLI",
		Long: `Install the latest GitHub Release of the Fanloop CLI.

fanloop update downloads the latest release, verifies checksums.txt, and switches
~/.fanloop/current. It does not sync Workflow configuration.

fanloop update config syncs only Workflow packages from the configuration repository.
It skips the download when that commit is already installed, and it prints each
Workflow id with the synced commit.`,
		Annotations: map[string]string{"bootstrap_control": "true"},
		Args:        cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runReleaseUpdate(cmd.Context(), stdout, stderr)
		},
	}
	command.AddCommand(&cobra.Command{
		Use:   "config",
		Short: "Sync Workflow configuration without downloading a CLI release",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runConfigUpdate(cmd.Context(), stdout, stderr)
		},
	})
	return command
}

func runReleaseUpdate(ctx context.Context, stdout, stderr io.Writer) error {
	dataRoot, err := installedDataRoot()
	if err != nil {
		return reportUpdate(stderr, err)
	}
	result, err := applyReleaseUpdate(ctx, dataRoot)
	if err != nil {
		return reportUpdate(stderr, err)
	}
	state := "updated"
	if result.Unchanged {
		state = "unchanged"
	}
	_, err = fmt.Fprintf(stdout, "%s %s\n", state, result.Version)
	if err != nil {
		return reportUpdate(stderr, err)
	}
	return nil
}

func runConfigUpdate(ctx context.Context, stdout, stderr io.Writer) error {
	dataRoot, err := installedDataRoot()
	if err != nil {
		return reportUpdate(stderr, err)
	}
	result, err := applyConfigSync(ctx, dataRoot)
	if err != nil {
		return reportUpdate(stderr, err)
	}
	state := "synced"
	if result.Unchanged {
		state = "unchanged"
	}
	for _, id := range result.WorkflowIDs {
		if _, err := fmt.Fprintf(stdout, "%s %s %s\n", state, id, result.CommitSHA); err != nil {
			return reportUpdate(stderr, err)
		}
	}
	return nil
}

func installedDataRoot() (string, error) {
	dataRoot, err := release.DefaultDataRoot()
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(dataRoot) {
		return "", fmt.Errorf("FANLOOP_DATA_HOME must be absolute")
	}
	return dataRoot, nil
}

func reportUpdate(stderr io.Writer, err error) error {
	fmt.Fprintf(stderr, "fanloop update: %s\n", err)
	return &clierrs.ExitError{Code: 1}
}
