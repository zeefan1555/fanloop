package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	clierrs "github.com/zeefan1555/fanloop/errs"
	"github.com/zeefan1555/fanloop/internal/configuration"
	"github.com/zeefan1555/fanloop/internal/distribution"
	"github.com/zeefan1555/fanloop/internal/fornaxmirror"
	"github.com/zeefan1555/fanloop/internal/release"
)

var (
	applyReleaseUpdate = func(ctx context.Context, dataRoot string) (distribution.Result, error) {
		return distribution.Update(ctx, distribution.Options{DataRoot: dataRoot})
	}
	applyGitConfigSync    = configuration.Sync
	applyFornaxConfigSync = func(ctx context.Context, dataRoot string) (fornaxmirror.SyncResult, error) {
		return fornaxmirror.Sync(ctx, dataRoot, fornaxmirror.DefaultClient())
	}
)

func newUpdateCommand(stdout, stderr io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:   "update",
		Short: "Install the latest GitHub Release and synchronize Workflow configuration",
		Long: `Install the latest GitHub Release and synchronize Workflow configuration from second-brain-skills main.

fanloop update downloads the latest release, verifies checksums.txt, and switches
~/.fanloop/current. It then runs git ls-remote on second-brain-skills main. When
config/current already has that commit it skips checkout. Otherwise it checks out,
validates, publishes config/snapshots/<sha>, and activates it under the configuration lock.

fanloop update config synchronizes that Git snapshot without downloading a CLI
release. It prints the full commit and the workflow ids. A missing fornax.json is
ignored. FANLOOP_CONFIG_SOURCE is an author source and is not synced.

FANLOOP_USE_FORNAX=1, true, or yes, together with a regular fornax.json, selects
the Fornax Online/prod sync instead. That path prints each id@version.`,
		Annotations: map[string]string{"bootstrap_control": "true"},
		Args:        cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runReleaseUpdate(cmd.Context(), stdout, stderr)
		},
	}
	command.AddCommand(&cobra.Command{
		Use:   "config",
		Short: "Synchronize Workflow configuration from second-brain-skills main",
		Long: `Synchronize Workflow configuration from second-brain-skills main without downloading a CLI release.

The command prints "fanloop configuration updated: <sha> (<workflow-id>, ...)" or
"fanloop configuration is already up to date: <sha> (<workflow-id>, ...)".
The sha is the full 40-character commit. A missing fornax.json is ignored.
FANLOOP_CONFIG_SOURCE is an author source and is not synced.
Git failure exits 1 and leaves config/current unchanged.

FANLOOP_USE_FORNAX=1, true, or yes, together with a regular fornax.json, synchronizes
released Fornax packages instead and prints each id@version.`,
		Args: cobra.NoArgs,
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
	return synchronizeConfiguration(ctx, dataRoot, stdout, stderr)
}

func runConfigUpdate(ctx context.Context, stdout, stderr io.Writer) error {
	dataRoot, err := installedDataRoot()
	if err != nil {
		return reportUpdate(stderr, err)
	}
	return synchronizeConfiguration(ctx, dataRoot, stdout, stderr)
}

func synchronizeConfiguration(ctx context.Context, dataRoot string, stdout, stderr io.Writer) error {
	if _, explicit := os.LookupEnv("FANLOOP_CONFIG_SOURCE"); explicit {
		return reportUpdate(stderr, fmt.Errorf("FANLOOP_CONFIG_SOURCE is an author source and is not synced"))
	}
	if fornaxmirror.UseReleasedSnapshot(dataRoot) {
		result, err := applyFornaxConfigSync(ctx, dataRoot)
		if err != nil {
			return reportUpdate(stderr, err)
		}
		if err = writeFornaxConfigurationSync(stdout, result); err != nil {
			return reportUpdate(stderr, err)
		}
		return nil
	}
	result, err := applyGitConfigSync(ctx, dataRoot)
	if err != nil {
		return reportUpdate(stderr, err)
	}
	if err = writeGitConfigurationSync(stdout, result); err != nil {
		return reportUpdate(stderr, err)
	}
	return nil
}

func writeGitConfigurationSync(stdout io.Writer, result configuration.SyncResult) error {
	message := "fanloop configuration updated:"
	if result.Unchanged {
		message = "fanloop configuration is already up to date:"
	}
	_, err := fmt.Fprintf(stdout, "%s %s (%s)\n", message, result.CommitSHA, strings.Join(result.WorkflowIDs, ", "))
	return err
}

func writeFornaxConfigurationSync(stdout io.Writer, result fornaxmirror.SyncResult) error {
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
