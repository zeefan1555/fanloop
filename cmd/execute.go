package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/zeefan1555/commonloop/errs"
	"github.com/zeefan1555/commonloop/internal/doctor"
	"github.com/zeefan1555/commonloop/internal/idl"
	"github.com/zeefan1555/commonloop/internal/idl/commonidl"
	"github.com/zeefan1555/commonloop/internal/idl/erroridl"
	"github.com/zeefan1555/commonloop/internal/idl/opsidl"
	"github.com/zeefan1555/commonloop/internal/output"
	"github.com/zeefan1555/commonloop/internal/workflow"
)

type operationControls struct {
	input  string
	dryRun bool
}

type operationResult struct {
	data     any
	workflow workflow.Ref
	partial  bool
	exitCode int
	warnings []string
}

func operationCommand(id string) *cobra.Command {
	spec, ok := idl.LookupCommand(id)
	if !ok {
		panic("missing command spec: " + id)
	}
	use := id
	if separator := strings.LastIndexByte(id, '.'); separator >= 0 {
		use = id[separator+1:]
	}
	return &cobra.Command{Use: use, Short: spec.Summary, Args: cobra.NoArgs, Annotations: map[string]string{"command_id": id}}
}

func addOperationControls(command *cobra.Command, controls *operationControls) {
	command.Flags().StringVar(&controls.input, "input", "", "complete JSON request, @file, or - for stdin")
	spec, ok := idl.LookupCommand(command.Annotations["command_id"])
	if ok && spec.SupportsDryRun {
		command.Flags().BoolVar(&controls.dryRun, "dry-run", false, "validate and plan without business side effects")
	}
	if ok && spec.RequirementScope != commonidl.RequirementScope_none {
		command.Long += `

Diagnostics:
  Every real invocation with a valid --root best-effort appends execution metadata plus the complete unredacted arguments, stdin, stdout, and stderr to
  .commonloop/log/cli.jsonl. This also applies to read-only and --dry-run calls; the file may contain secrets, and logging never changes command output or exit status.`
	}
}

func runOperation[T any](
	ctx context.Context,
	command *cobra.Command,
	root string,
	controls operationControls,
	ioStreams streams,
	typed func() (T, *erroridl.PublicError),
	run func(context.Context, T, bool) (operationResult, *erroridl.PublicError),
) error {
	id := command.Annotations["command_id"]
	meta := output.Meta{Command: id, RequirementRoot: root, DryRun: controls.dryRun}
	fail := func(failure *erroridl.PublicError) error {
		_ = writeFailure(ioStreams.err, failure, meta)
		return &errs.ExitError{Code: errs.ExitCode(failure), ErrorCode: failure.Code.String()}
	}
	spec, ok := idl.LookupCommand(id)
	if !ok {
		return fail(errs.NewCode(erroridl.ErrorCode_INTERNAL, "command contract is missing", nil))
	}
	if root == "" && (spec.RequirementScope == commonidl.RequirementScope_new || spec.RequirementScope == commonidl.RequirementScope_existing) {
		return fail(errs.NewCode(erroridl.ErrorCode_ROOT_REQUIRED, "--root is required", nil))
	}
	if controls.dryRun && !spec.SupportsDryRun {
		return fail(invalidArgument("--dry-run is not supported by this command"))
	}
	request, failure := commandRequest(command, controls.input, ioStreams.in, typed)
	if failure != nil {
		return fail(failure)
	}
	result, failure := run(ctx, request, controls.dryRun)
	if failure != nil {
		return fail(failure)
	}
	if result.workflow.ID != "" {
		meta.Workflow = &result.workflow
	}
	for _, warning := range result.warnings {
		_, _ = fmt.Fprintln(ioStreams.err, "warning:", warning)
	}
	notice := localNotice()
	if result.partial {
		if err := output.PartialWithNotice(ioStreams.out, result.data, meta, notice); err != nil {
			return &errs.ExitError{Code: 5}
		}
		return &errs.ExitError{Code: result.exitCode}
	}
	if err := output.SuccessWithNotice(ioStreams.out, result.data, meta, notice); err != nil {
		return &errs.ExitError{Code: 5}
	}
	return nil
}

func writeFailure(writer io.Writer, failure *erroridl.PublicError, meta output.Meta) error {
	return output.FailureWithNotice(writer, failure, meta, localNotice())
}

func localNotice() commonidl.Notice {
	components := []string{}
	for _, check := range doctor.DefaultRuntime().Run("").Checks {
		if check.Status != opsidl.DoctorCheckStatus_failed {
			continue
		}
		switch check.Id {
		case "binary_checksum":
			components = append(components, "cli")
		case "skills":
			components = append(components, "skills")
		case "workflows":
			components = append(components, "workflows")
		case "release_manifest", "version_drift":
			components = append(components, "release")
		}
	}
	if len(components) == 0 {
		return commonidl.Notice{}
	}
	sort.Strings(components)
	return commonidl.Notice{Drift: &commonidl.DriftNotice{Components: components, Command: "commonloop doctor"}}
}

func commandRequest[T any](command *cobra.Command, input string, stdin io.Reader, typed func() (T, *erroridl.PublicError)) (T, *erroridl.PublicError) {
	if !command.Flags().Changed("input") {
		return typed()
	}
	typedFlag := ""
	command.Flags().Visit(func(flag *pflag.Flag) {
		if flag.Name != "input" && flag.Name != "dry-run" && flag.Name != "root" && typedFlag == "" {
			typedFlag = flag.Name
		}
	})
	if typedFlag != "" {
		var zero T
		return zero, invalidArgument("--input and typed flags are mutually exclusive")
	}
	content, failure := resolveInput(input, stdin)
	if failure != nil {
		var zero T
		return zero, failure
	}
	return decodeRequest[T](content)
}

func resolveInput(input string, stdin io.Reader) (string, *erroridl.PublicError) {
	if input == "-" {
		content, err := io.ReadAll(stdin)
		if err != nil {
			return "", invalidArgument("cannot read --input from stdin: " + err.Error())
		}
		return string(content), nil
	}
	if strings.HasPrefix(input, "@") {
		path := strings.TrimPrefix(input, "@")
		if path == "" {
			return "", invalidArgument("--input @ requires a file path")
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return "", invalidArgument("cannot read --input file: " + err.Error())
		}
		return string(content), nil
	}
	return input, nil
}

func decodeRequest[T any](input string) (T, *erroridl.PublicError) {
	var value T
	decoder := json.NewDecoder(bytes.NewBufferString(input))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, invalidInputJSON(err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("trailing JSON value")
		}
		return value, invalidInputJSON(err)
	}
	return value, nil
}

func decodeRepeated[T any](values []string) ([]T, *erroridl.PublicError) {
	if len(values) == 0 {
		return nil, nil
	}
	result := make([]T, 0, len(values))
	for _, raw := range values {
		value, failure := decodeRequest[T](raw)
		if failure != nil {
			return nil, failure
		}
		result = append(result, value)
	}
	return result, nil
}

func decodeNamedJSON(values []string, flag string) (map[string]json.RawMessage, *erroridl.PublicError) {
	if len(values) == 0 {
		return nil, nil
	}
	result := map[string]json.RawMessage{}
	for _, value := range values {
		key, raw, ok := strings.Cut(value, "=")
		if !ok || strings.TrimSpace(key) == "" || !json.Valid([]byte(raw)) {
			return nil, invalidArgument(flag + " must be name=<JSON value>")
		}
		if _, exists := result[key]; exists {
			return nil, invalidArgument(fmt.Sprintf("name %q was provided more than once", key))
		}
		result[key] = json.RawMessage(raw)
	}
	return result, nil
}

func invalidArgument(message string) *erroridl.PublicError {
	return errs.NewCode(erroridl.ErrorCode_INVALID_ARGUMENT, message, nil)
}

func invalidInputJSON(err error) *erroridl.PublicError {
	return errs.NewCode(erroridl.ErrorCode_INVALID_INPUT_JSON, err.Error(), nil)
}

func normalizePublicError(err error) *erroridl.PublicError {
	var failure *erroridl.PublicError
	if errors.As(err, &failure) {
		return failure
	}
	return errs.NewCode(erroridl.ErrorCode_INTERNAL, err.Error(), nil)
}

func normalizePublicErrorOrNil(err error) *erroridl.PublicError {
	if err == nil {
		return nil
	}
	return normalizePublicError(err)
}
