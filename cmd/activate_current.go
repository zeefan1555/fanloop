package cmd

import (
	"io"

	"github.com/spf13/cobra"
	clierrs "github.com/zeefan1555/fanloop/errs"
	"github.com/zeefan1555/fanloop/internal/release/currentlayout"
)

func newActivateCurrentCommand(stderr io.Writer) *cobra.Command {
	var root, version, restore string
	command := &cobra.Command{
		Use: "__activate-current", Hidden: true, Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if restore != "" {
				if err := currentlayout.RestoreRelease(root, version, restore); err != nil {
					_ = writeInstallFailure(stderr, err)
					return &clierrs.ExitError{Code: 1}
				}
				return nil
			}
			if _, err := currentlayout.Switch(root, version); err != nil {
				_ = writeInstallFailure(stderr, err)
				return &clierrs.ExitError{Code: 1}
			}
			return nil
		},
	}
	command.Flags().StringVar(&root, "data-root", "", "Fanloop data directory")
	command.Flags().StringVar(&version, "version", "", "installed release version")
	command.Flags().StringVar(&restore, "restore-release", "", "retained release backup basename")
	return command
}
