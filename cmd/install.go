package cmd

import (
	"encoding/json"
	"io"

	"github.com/spf13/cobra"
	clierrs "github.com/zeefan1555/fanloop/errs"
	"github.com/zeefan1555/fanloop/internal/output"
	releaseinstall "github.com/zeefan1555/fanloop/internal/release/install"
)

func newInstallCommand(stdout, stderr io.Writer) *cobra.Command {
	var request releaseinstall.Request
	command := &cobra.Command{
		Use:    "__install",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			result, err := releaseinstall.Run(request)
			if err != nil {
				_ = writeInstallFailure(stderr, err)
				return &clierrs.ExitError{Code: 1}
			}
			if err := output.Success(stdout, result, output.Meta{Command: "__install"}); err != nil {
				return &clierrs.ExitError{Code: 5}
			}
			return nil
		},
	}
	command.Flags().StringVar(&request.Source, "source", "", "extracted release directory")
	command.Flags().StringVar(&request.DataRoot, "data-root", "", "Fanloop user data directory")
	command.Flags().StringVar(&request.SkillRoots.Codex, "codex-skills-root", "", "Codex Skills directory")
	command.Flags().StringVar(&request.SkillRoots.Agent, "agent-skills-root", "", "Agent Skills directory")
	command.Flags().StringVar(&request.SkillRoots.Trae, "trae-skills-root", "", "Trae Skills directory")
	command.Flags().StringVar(&request.SkillRoots.Claude, "claude-skills-root", "", "Claude Skills directory")
	command.Flags().BoolVar(&request.ReplaceInvalid, "replace-invalid", false, "replace a retained invalid release")
	return command
}

func writeInstallFailure(writer io.Writer, failure error) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(map[string]any{
		"ok": false,
		"error": map[string]any{
			"type": "internal", "code": "INSTALL_FAILED", "message": failure.Error(),
			"hint": "Keep the current release and fix the reported local installation problem.", "retryable": false,
		},
		"meta": output.Meta{Command: "__install"}, "_notice": map[string]any{},
	})
}
