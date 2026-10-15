package cmd

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	clierrs "github.com/zeefan1555/fanloop/errs"
	"github.com/zeefan1555/fanloop/internal/distribution"
	"github.com/zeefan1555/fanloop/internal/fornaxmirror"
	"github.com/zeefan1555/fanloop/internal/release"
)

var (
	applyReleaseUpdate = func(ctx context.Context, dataRoot string) (distribution.Result, error) {
		return distribution.Update(ctx, distribution.Options{DataRoot: dataRoot})
	}
	applyConfigSync = func(ctx context.Context, dataRoot string) (fornaxmirror.SyncResult, error) {
		return fornaxmirror.Sync(ctx, dataRoot, fornaxmirror.DefaultClient())
	}
)

func newUpdateCommand(stdout, stderr io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:   "update",
		Short: "Install the latest GitHub Release and synchronize Fornax Workflow packages",
		Long: `Install the latest GitHub Release and synchronize Fornax Workflow packages.

fanloop update downloads the latest release, verifies checksums.txt, and switches
~/.fanloop/current. It then installs the latest Online/prod release for every
source in fornax.json through the logged-in fornax-cli.

fanloop update config synchronizes those Workflow packages without downloading a
CLI release. It skips the download when the released set and the installed tree
are unchanged.`,
		Annotations: map[string]string{"bootstrap_control": "true"},
		Args:        cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runReleaseUpdate(cmd.Context(), stdout, stderr)
		},
	}
	command.AddCommand(&cobra.Command{
		Use:   "config",
		Short: "Synchronize released Workflow packages without downloading a CLI release",
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
	if _, err = fmt.Fprintf(stdout, "%s %s\n", state, result.Version); err != nil {
		return reportUpdate(stderr, err)
	}
	synced, err := applyConfigSync(ctx, dataRoot)
	if err != nil {
		return reportUpdate(stderr, err)
	}
	if err = writeConfigurationSync(stdout, synced); err != nil {
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
	if err = writeConfigurationSync(stdout, result); err != nil {
		return reportUpdate(stderr, err)
	}
	return nil
}

func writeConfigurationSync(stdout io.Writer, result fornaxmirror.SyncResult) error {
	parts := make([]string, len(result.Workflows))
	for index, item := range result.Workflows {
		parts[index] = item.ID + "@" + item.Version
	}
	sort.Strings(parts)
	message := "fanloop configuration updated:"
	if result.Outcome == fornaxmirror.OutcomeUnchanged {
		message = "fanloop configuration is already up to date:"
	}
	_, err := fmt.Fprintln(stdout, message, strings.Join(parts, ", "))
	return err
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
