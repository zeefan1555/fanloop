package cmd

import (
	"context"
	"strings"

	"github.com/spf13/cobra"
	cardruntime "github.com/zeefan1555/commonloop/internal/card"
	"github.com/zeefan1555/commonloop/internal/idl/cardidl"
	"github.com/zeefan1555/commonloop/internal/idl/erroridl"
)

func newCardCommand(ioStreams streams, root *string) *cobra.Command {
	command := &cobra.Command{
		Use:   "card",
		Short: "Render deterministic current or panorama cards",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return command.Help()
		},
	}
	command.AddCommand(newCardRenderCommand(ioStreams, root))
	return command
}

func newCardRenderCommand(ioStreams streams, root *string) *cobra.Command {
	var controls operationControls
	var view, format string
	command := operationCommand("card.render")
	command.Long = `Purpose:
  Render a deterministic current or panorama Card from committed Requirement facts.

Effect:
  Local write. A non-dry-run call returns Card content and writes a snapshot; dry-run returns content without a snapshot write.

Request JSON:
  {
    "view": "current",
    "format": "lark_json"
  }

Typed flags:
  commonloop card render --root <ABSOLUTE_REQUIREMENT_ROOT> --view current --format lark-json

Constraints:
  view and format are required. view is current or panorama. JSON format is markdown or lark_json;
  the equivalent typed flag is markdown or lark-json.
  Use either --input or the typed flags, not both.

Controls:
  --root is required and is not part of Request JSON. --input accepts inline JSON, @file, or - for stdin.
  JSON input rejects unknown fields, type errors, invalid enums, and trailing JSON.
  --dry-run renders without writing a snapshot. --input and --dry-run are also Request-external controls.

Next step:
  Use the returned content directly and, for non-dry-run calls, retain the returned snapshot path for audit.`
	command.Example = `  commonloop card render --root <ABSOLUTE_REQUIREMENT_ROOT> --input @request.json --dry-run
  commonloop card render --root <ABSOLUTE_REQUIREMENT_ROOT> --view current --format lark-json`
	command.RunE = func(command *cobra.Command, _ []string) error {
		typed := func() (*cardidl.CardRenderRequest, *erroridl.PublicError) {
			parsedView, err := cardidl.CardViewFromString(view)
			if err != nil || parsedView == cardidl.CardView_unspecified {
				return nil, invalidArgument("--view must be current or panorama")
			}
			parsedFormat, err := cardidl.CardFormatFromString(strings.ReplaceAll(format, "-", "_"))
			if err != nil || parsedFormat == cardidl.CardFormat_unspecified {
				return nil, invalidArgument("--format must be markdown or lark-json")
			}
			return &cardidl.CardRenderRequest{View: parsedView, Format: parsedFormat}, nil
		}
		return runOperation(command.Context(), command, *root, controls, ioStreams, typed,
			func(ctx context.Context, request *cardidl.CardRenderRequest, dryRun bool) (operationResult, *erroridl.PublicError) {
				response, err := cardruntime.DefaultRuntime().Render(ctx, *root, request, dryRun)
				if err != nil {
					return operationResult{}, normalizePublicError(err)
				}
				return operationResult{data: response, workflow: reportWorkflowRef(*root)}, nil
			})
	}
	addOperationControls(command, &controls)
	command.Flags().StringVar(&view, "view", "", "current or panorama")
	command.Flags().StringVar(&format, "format", "", "markdown or lark-json")
	return command
}
