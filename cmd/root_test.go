package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/zeefan1555/fanloop/internal/idl"
	"github.com/zeefan1555/fanloop/internal/idl/commonidl"
	"github.com/zeefan1555/fanloop/internal/idl/storageidl"
)

type requestValidator interface {
	IsValid() error
}

func TestMain(main *testing.M) {
	_ = os.Unsetenv("BOTMUX_CHAT_ID")
	_ = os.Unsetenv("BOTMUX_SESSION_ID")
	os.Exit(main.Run())
}

func TestProcessDoesNotInheritBotmuxBinding(t *testing.T) {
	if os.Getenv("FANLOOP_BOTMUX_SCRUB_CHILD") == "1" {
		if os.Getenv("BOTMUX_CHAT_ID") != "" || os.Getenv("BOTMUX_SESSION_ID") != "" {
			t.Fatal("test process inherited Botmux binding")
		}
		return
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(executable, "-test.run=^TestProcessDoesNotInheritBotmuxBinding$")
	command.Env = append(os.Environ(),
		"FANLOOP_BOTMUX_SCRUB_CHILD=1",
		"BOTMUX_CHAT_ID=oc_real_chat_must_not_escape",
		"BOTMUX_SESSION_ID=real_session_must_not_escape",
	)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("isolated test process: %v\n%s", err, output)
	}
}

func TestHiddenInstallFailureKeepsItsPrivateExitContract(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "missing-release")
	var stdout, stderr bytes.Buffer
	code := Execute(context.Background(), []string{
		"__install", "--source", missing, "--data-root", filepath.Join(root, "data"),
		"--codex-skills-root", filepath.Join(root, "codex"),
		"--agent-skills-root", filepath.Join(root, "agents"),
		"--trae-skills-root", filepath.Join(root, "trae"),
		"--claude-skills-root", filepath.Join(root, "claude"),
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), `"code": "INSTALL_FAILED"`) {
		t.Fatalf("exit = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}
}

func TestPayloadUpdateRequiresGitHubReleaseLauncher(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute(context.Background(), []string{"update"}, strings.NewReader(""), &stdout, &stderr)
	if code != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "update requires the GitHub Release launcher") {
		t.Fatalf("exit = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
	stderr.Reset()
	code = Execute(context.Background(), []string{"--update"}, strings.NewReader(""), &stdout, &stderr)
	if code != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "unknown flag --update") {
		t.Fatalf("retired flag exit = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
}

func TestTypedOptionalFlagsPreserveExplicitEmptyValues(t *testing.T) {
	cases := [][]string{
		{"flow", "init", "--root", filepath.Join(t.TempDir(), "typed"), "--workflow", "technical-solution-design", "--title", "test", "--source-url="},
		{"flow", "init", "--root", filepath.Join(t.TempDir(), "json"), "--input", `{"workflow":"fanloop","requirement":{"title":"test","source_url":""}}`},
	}
	for _, args := range cases {
		var stdout, stderr bytes.Buffer
		if code := Execute(context.Background(), args, strings.NewReader(""), &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), `"code": "INVALID_ARGUMENT"`) {
			t.Errorf("%v: exit = %d, stdout = %s, stderr = %s", args, code, stdout.String(), stderr.String())
		}
	}
}

func TestRootOwnsRealDomainSubcommands(t *testing.T) {
	root := NewRoot(bytes.NewReader(nil), io.Discard, io.Discard)
	want := map[string][]string{
		"flow":  {"init", "report", "status"},
		"trace": {"bind", "render", "status", "sync"},
		"card":  {"render"},
	}
	for domain, operations := range want {
		child, _, err := root.Find([]string{domain})
		if err != nil || child.Name() != domain {
			t.Fatalf("find %s: %v", domain, err)
		}
		got := []string{}
		for _, operation := range child.Commands() {
			if !operation.Hidden && operation.Name() != "help" {
				got = append(got, operation.Name())
				if operation.DisableFlagParsing {
					t.Fatalf("%s.%s disables Cobra flag parsing", domain, operation.Name())
				}
			}
		}
		if !reflect.DeepEqual(got, operations) {
			t.Fatalf("%s operations = %#v, want %#v", domain, got, operations)
		}
	}
}

func TestRootCommandTreeMatchesIDLRegistry(t *testing.T) {
	root := NewRoot(bytes.NewReader(nil), io.Discard, io.Discard)
	actual := map[string]bool{}
	var collect func(*cobra.Command, []string)
	collect = func(parent *cobra.Command, path []string) {
		children := make([]*cobra.Command, 0)
		for _, child := range parent.Commands() {
			if !child.Hidden && child.Name() != "help" {
				children = append(children, child)
			}
		}
		if len(children) == 0 {
			actual[strings.Join(path, ".")] = true
			return
		}
		for _, child := range children {
			collect(child, append(path, child.Name()))
		}
	}
	for _, child := range root.Commands() {
		if !child.Hidden && child.Name() != "help" && child.Annotations["bootstrap_control"] != "true" {
			collect(child, []string{child.Name()})
		}
	}
	expected := map[string]bool{}
	for _, spec := range idl.CommandSpecs() {
		expected[spec.ID] = true
	}
	if len(actual) != 11 || len(expected) != 11 {
		t.Fatalf("public command counts = Cobra %d, IDL %d; want 11", len(actual), len(expected))
	}
	if command, _, err := root.Find([]string{"schema"}); err == nil && command.Name() == "schema" {
		t.Fatal("root still exposes the retired schema command")
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("Cobra commands = %#v, IDL registry = %#v", actual, expected)
	}
}

func TestRootExposesOnlyFinalPublicDomains(t *testing.T) {
	var stdout bytes.Buffer
	root := NewRoot(bytes.NewReader(nil), &stdout, io.Discard)
	root.SetArgs([]string{"--help"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, command := range root.Commands() {
		if !command.Hidden && command.Name() != "help" {
			got[command.Name()] = true
		}
	}
	want := map[string]bool{
		"flow":    true,
		"trace":   true,
		"card":    true,
		"version": true,
		"doctor":  true,
		"update":  true,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("domains = %#v, want %#v", got, want)
	}
	if strings.Contains(stdout.String(), "completion") {
		t.Fatalf("root help exposed an unsupported completion domain:\n%s", stdout.String())
	}
	if strings.Contains(stdout.String(), "schema") {
		t.Fatalf("root help exposed the retired schema domain:\n%s", stdout.String())
	}
	stdout.Reset()
	root.SetArgs([]string{"update", "--help"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout.String(), "--root") {
		t.Fatalf("launcher update help exposed a Requirement flag:\n%s", stdout.String())
	}
	stdout.Reset()
	root.SetArgs([]string{"trace", "--help"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"bind", "status", "render", "sync"} {
		if !strings.Contains(stdout.String(), operation) {
			t.Fatalf("trace help does not expose %q:\n%s", operation, stdout.String())
		}
	}
	for _, retired := range []string{"record", "list", "migrate"} {
		if strings.Contains(stdout.String(), retired) {
			t.Fatalf("trace help still exposes retired operation %q:\n%s", retired, stdout.String())
		}
	}
}

func TestAgentHelpProgressivelyDisclosesWorkflow(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		want      []string
		forbidden []string
	}{
		{
			name: "root routes the agent",
			args: []string{"--help"},
			want: []string{
				"Agent workflow",
				"fanloop-workflow Skill",
				"fanloop flow status",
				"current.prompt and its required Skills",
				"target leaf command --help",
				"Report progress or a Condition result",
				"read flow status again",
				"NOT_INITIALIZED",
			},
			forbidden: []string{"schema describe", "botmux", "Panorama", "fanloop install"},
		},
		{
			name: "flow selects one operation",
			args: []string{"flow", "--help"},
			want: []string{
				"Use init only after status reports NOT_INITIALIZED",
				"Use status immediately before each action",
				"Use report progress for execution facts",
				"report result for Conditions and routing",
			},
		},
		{
			name: "init returns to status",
			args: []string{"flow", "init", "--help"},
			want: []string{
				"only after flow status reports NOT_INITIALIZED",
				"After a successful non-dry-run initialization, run flow status",
				"Request JSON:",
			},
			forbidden: []string{"schema describe"},
		},
		{
			name: "progress returns to status",
			args: []string{"flow", "report", "progress", "--help"},
			want: []string{
				"After an accepted non-dry-run update, run flow status again",
			},
		},
		{
			name: "result discovers its current contract",
			args: []string{"flow", "report", "result", "--help"},
			want: []string{
				"Request JSON:",
				"After an accepted non-dry-run result, run flow status again",
			},
			forbidden: []string{"schema describe", "check_merge_request_gate", "unit_tests_failed", "review_code"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			root := NewRoot(bytes.NewReader(nil), &stdout, io.Discard)
			root.SetArgs(test.args)
			if err := root.ExecuteContext(context.Background()); err != nil {
				t.Fatal(err)
			}
			for _, value := range test.want {
				if !strings.Contains(stdout.String(), value) {
					t.Fatalf("%v help does not contain %q:\n%s", test.args, value, stdout.String())
				}
			}
			for _, value := range test.forbidden {
				if strings.Contains(stdout.String(), value) {
					t.Fatalf("%v help contains workflow-specific detail %q:\n%s", test.args, value, stdout.String())
				}
			}
		})
	}
}

func TestLeafHelpPublishesCompleteRequestContract(t *testing.T) {
	tests := []struct {
		name      string
		commandID string
		args      []string
		want      []string
		forbidden []string
	}{
		{
			name:      "flow init",
			commandID: "flow.init",
			args:      []string{"flow", "init", "--help"},
			want: []string{
				`"workflow": "<WORKFLOW_ID>"`,
				`"requirement"`,
				`"title": "<TITLE>"`,
				`"source_url": "<SOURCE_URL>"`,
				"--workflow <WORKFLOW_ID> --title <TITLE> --source-url <SOURCE_URL>",
				"NOT_INITIALIZED",
			},
		},
		{
			name:      "flow status",
			commandID: "flow.status",
			args:      []string{"flow", "status", "--help"},
			want: []string{
				"Request JSON:\n  {}",
				"Read only",
				"immediately before acting",
			},
		},
		{
			name:      "flow progress",
			commandID: "flow.report.progress",
			args:      []string{"flow", "report", "progress", "--help"},
			want: []string{
				`"step_id": "<CURRENT_STEP_ID>"`,
				`"status": "in_progress"`,
				`"summary": "<SUMMARY>"`,
				`"evidence": [`,
				`"source": "file"`,
				"current.context.step_id",
				"in_progress, fixing, or blocked",
				"step_id and status are required",
				"evidence may be omitted or empty",
			},
		},
		{
			name:      "flow result",
			commandID: "flow.report.result",
			args:      []string{"flow", "report", "result", "--help"},
			want: []string{
				`"step_id": "<CURRENT_STEP_ID>"`,
				`"condition_results"`,
				`"condition_id": "<CONDITION_ID>"`,
				`"output"`,
				`"evidence": [`,
				`"source": "file"`,
				`"summary": "<SUMMARY>"`,
				`"route"`,
				`"next_step_id": "<NEXT_STEP_ID>"`,
				"current.conditions",
				"current.available_routes[].route",
				"next_step_id, back_step_id, or terminal: true",
				"Evidence source is human, system, ai, file, or url",
				"Each condition_results item requires condition_id and output.type/value",
				"evidence may be omitted or empty",
			},
			forbidden: []string{"check_merge_request_gate", "unit_tests_failed", "review_code"},
		},
		{
			name:      "trace bind",
			commandID: "trace.bind",
			args:      []string{"trace", "bind", "--help"},
			want: []string{
				`"document_url": "<TRACE_DOCUMENT_URL>"`,
				`"registry": "production"`,
				`"cli_log_document_url": "<CLI_LOG_DOCUMENT_URL>"`,
				"--document-url <TRACE_DOCUMENT_URL> --registry production",
				"--cli-log-document-url <CLI_LOG_DOCUMENT_URL>",
				"production or test",
				"After the first successful bind",
			},
		},
		{
			name:      "trace status",
			commandID: "trace.status",
			args:      []string{"trace", "status", "--help"},
			want: []string{
				"Request JSON:\n  {}",
				"Read only",
				"binding and projection status",
				"Trace/CLI-log document binding",
			},
		},
		{
			name:      "trace render",
			commandID: "trace.render",
			args:      []string{"trace", "render", "--help"},
			want: []string{
				"Request JSON:\n  {}",
				"Local write",
				"Events projection",
			},
		},
		{
			name:      "trace sync",
			commandID: "trace.sync",
			args:      []string{"trace", "sync", "--help"},
			want: []string{
				"Request JSON:\n  {}",
				"External write",
				"partial",
				"Trace document and Registry",
				"CLI-log document",
				"without redaction or truncation",
			},
		},
		{
			name:      "card render",
			commandID: "card.render",
			args:      []string{"card", "render", "--help"},
			want: []string{
				`"view": "current"`,
				`"format": "lark_json"`,
				"--view current --format lark-json",
				"current or panorama",
				"markdown or lark_json",
				"view and format are required",
			},
		},
		{
			name:      "version",
			commandID: "version",
			args:      []string{"version", "--help"},
			want: []string{
				"Request JSON:\n  {}",
				"Read only",
				"does not require --root",
			},
		},
		{
			name:      "doctor",
			commandID: "doctor",
			args:      []string{"doctor", "--help"},
			want: []string{
				"Request JSON:\n  {}",
				"Read only",
				"--root is optional",
				"unhealthy",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			help := renderHelp(t, test.args)
			for _, section := range []string{"Purpose:", "Effect:", "Request JSON:", "Typed flags:", "Constraints:", "Controls:", "Next step:"} {
				if !strings.Contains(help, section) {
					t.Fatalf("%s help does not contain section %q:\n%s", test.commandID, section, help)
				}
			}
			if !strings.Contains(help, "JSON input rejects unknown fields, type errors, invalid enums, and trailing JSON") {
				t.Fatalf("%s help does not explain strict JSON rejection:\n%s", test.commandID, help)
			}
			for _, value := range test.want {
				if !strings.Contains(help, value) {
					t.Fatalf("%s help does not contain %q:\n%s", test.commandID, value, help)
				}
			}
			for _, value := range test.forbidden {
				if strings.Contains(help, value) {
					t.Fatalf("%s help contains Workflow-specific value %q:\n%s", test.commandID, value, help)
				}
			}
			if spec, ok := idl.LookupCommand(test.commandID); ok && spec.RequirementScope != commonidl.RequirementScope_none &&
				(!strings.Contains(help, ".fanloop/log/cli.jsonl") || !strings.Contains(help, "complete unredacted arguments, stdin, stdout, and stderr")) {
				t.Fatalf("%s help does not disclose the complete Requirement execution transcript:\n%s", test.commandID, help)
			}
			validateHelpRequest(t, test.commandID, requestJSONFromHelp(t, help))
		})
	}
}

func renderHelp(t *testing.T, args []string) string {
	t.Helper()
	var stdout bytes.Buffer
	root := NewRoot(bytes.NewReader(nil), &stdout, io.Discard)
	root.SetArgs(args)
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	return stdout.String()
}

func requestJSONFromHelp(t *testing.T, help string) string {
	t.Helper()
	const start = "Request JSON:\n"
	const end = "\n\nTyped flags:"
	index := strings.Index(help, start)
	if index < 0 {
		t.Fatalf("help does not contain %q:\n%s", start, help)
	}
	after, ok := strings.CutPrefix(help[index:], start)
	if !ok {
		t.Fatalf("help does not contain %q:\n%s", start, help)
	}
	request, _, ok := strings.Cut(after, end)
	if !ok {
		t.Fatalf("help does not terminate Request JSON with %q:\n%s", end, help)
	}
	return strings.TrimSpace(request)
}

func validateHelpRequest(t *testing.T, commandID, raw string) {
	t.Helper()
	spec, ok := idl.LookupCommand(commandID)
	if !ok {
		t.Fatalf("missing CommandSpec %q", commandID)
	}
	request := reflect.New(spec.RequestType).Interface()
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(request); err != nil {
		t.Fatalf("decode %s Help Request %s: %v", commandID, raw, err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("%s Help Request has trailing JSON: %v", commandID, err)
	}
	validator, ok := request.(requestValidator)
	if !ok {
		t.Fatalf("%s Request does not expose IsValid", commandID)
	}
	if err := validator.IsValid(); err != nil {
		t.Fatalf("%s Help Request is invalid: %v", commandID, err)
	}
}

func TestFlowHelpExplainsProgressConditionAndRoutes(t *testing.T) {
	tests := []struct {
		args []string
		want []string
	}{
		{
			args: []string{"flow", "status", "--help"},
			want: []string{
				"immediately before acting",
				"resolved Prompt and Skills",
				"reusable Conditions",
				"required Output schemas",
				"normal and return Routes",
			},
		},
		{
			args: []string{"flow", "report", "--help"},
			want: []string{
				"Run flow status immediately before reporting",
				"progress for non-terminal",
				"ConditionResults",
				"Flow or Loop Route",
			},
		},
	}
	for _, test := range tests {
		t.Run(strings.Join(test.args, "_"), func(t *testing.T) {
			var stdout bytes.Buffer
			root := NewRoot(bytes.NewReader(nil), &stdout, io.Discard)
			root.SetArgs(test.args)
			if err := root.ExecuteContext(context.Background()); err != nil {
				t.Fatal(err)
			}
			for _, value := range test.want {
				if !strings.Contains(stdout.String(), value) {
					t.Fatalf("%v help does not contain %q:\n%s", test.args, value, stdout.String())
				}
			}
			for _, retired := range []string{"GuardResult", "Agent-submitted Artifacts", "Automatic Loop", "Manual Loop", "flow report output", "flow report loop"} {
				if strings.Contains(stdout.String(), retired) {
					t.Fatalf("%v help contains retired %q:\n%s", test.args, retired, stdout.String())
				}
			}
		})
	}
}

func TestExecuteRoutesFlowWithoutPython(t *testing.T) {
	t.Setenv("FANLOOP_PYTHON", filepath.Join(t.TempDir(), "missing-python"))
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Execute(
		context.Background(),
		[]string{"flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "Native"},
		strings.NewReader(""), &stdout, &stderr,
	)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"command": "flow.init"`) || !strings.Contains(stdout.String(), `"ok": true`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".fanloop", "flow", "state.json")); err != nil {
		t.Fatalf("final Flow did not write nested .fanloop state: %v", err)
	}
}

func TestExecuteAppendsRequirementExecutionLog(t *testing.T) {
	root := t.TempDir()
	args := []string{"flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "complete transcript"}
	var stdout, stderr bytes.Buffer
	code := Execute(
		context.Background(),
		args,
		strings.NewReader(""), &stdout, &stderr,
	)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	entries := readExecutionLog(t, root)
	if len(entries) != 1 {
		t.Fatalf("log entries = %d, want 1", len(entries))
	}
	entry := entries[0]
	if entry.SchemaVersion != 2 || entry.InvocationId == "" || entry.DurationMs < 0 ||
		entry.CommandId != "flow.init" || entry.CliVersion == "" || entry.ReleaseVersion == "" || entry.CommitSha == "" ||
		entry.DryRun || entry.ExitCode != 0 || entry.ErrorCode != nil ||
		!reflect.DeepEqual(entry.Arguments, args) || entry.Stdin != "" || entry.Stdout != stdout.String() || entry.Stderr != stderr.String() {
		t.Fatalf("unexpected log entry: %#v", entry)
	}
	if _, err := time.Parse(time.RFC3339Nano, entry.StartedAt); err != nil {
		t.Fatalf("started_at = %q: %v", entry.StartedAt, err)
	}
}

func TestExecuteLogsDryRunReadFailureAndPartialResults(t *testing.T) {
	root := t.TempDir()
	commands := []struct {
		args      []string
		stdin     string
		exitCode  int
		commandID string
		dryRun    bool
		errorCode string
		stdout    string
		stderr    string
	}{
		{args: []string{"flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "Dry", "--dry-run"}, exitCode: 0, commandID: "flow.init", dryRun: true},
		{args: []string{"flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "Real"}, exitCode: 0, commandID: "flow.init"},
		{args: []string{"flow", "status", "--root", root, "--input", "-"}, stdin: "{}\n", exitCode: 0, commandID: "flow.status"},
		{args: []string{"flow", "status", "--root", root}, exitCode: 0, commandID: "flow.status"},
		{args: []string{"flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "Again"}, exitCode: 1, commandID: "flow.init", errorCode: "ALREADY_INITIALIZED"},
		{args: []string{"flow", "status", "--root", root, "--unknown"}, exitCode: 2, commandID: "flow.status", errorCode: "INVALID_ARGUMENT"},
	}
	for index := range commands {
		test := &commands[index]
		var stdout, stderr bytes.Buffer
		if code := Execute(context.Background(), test.args, strings.NewReader(test.stdin), &stdout, &stderr); code != test.exitCode {
			t.Fatalf("%v exit = %d, want %d; stdout=%s stderr=%s", test.args, code, test.exitCode, stdout.String(), stderr.String())
		}
		test.stdout = stdout.String()
		test.stderr = stderr.String()
	}
	if err := os.WriteFile(filepath.Join(root, ".fanloop", "flow", "state.json"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Execute(context.Background(), []string{"doctor", "--root", root}, strings.NewReader(""), &stdout, &stderr); code != 1 {
		t.Fatalf("doctor exit = %d, want partial exit 1; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	commands = append(commands, struct {
		args      []string
		stdin     string
		exitCode  int
		commandID string
		dryRun    bool
		errorCode string
		stdout    string
		stderr    string
	}{args: []string{"doctor", "--root", root}, commandID: "doctor", exitCode: 1, stdout: stdout.String(), stderr: stderr.String()})

	entries := readExecutionLog(t, root)
	if len(entries) != len(commands) {
		t.Fatalf("log entries = %d, want %d", len(entries), len(commands))
	}
	for index, want := range commands {
		got := entries[index]
		if got.CommandId != want.commandID || got.DryRun != want.dryRun || int(got.ExitCode) != want.exitCode {
			t.Errorf("entry %d = %#v, want command=%s dry_run=%v exit=%d", index, got, want.commandID, want.dryRun, want.exitCode)
		}
		if !reflect.DeepEqual(got.Arguments, want.args) || got.Stdin != want.stdin || got.Stdout != want.stdout || got.Stderr != want.stderr {
			t.Errorf("entry %d transcript = %#v, want arguments=%#v stdin=%q stdout=%q stderr=%q", index, got, want.args, want.stdin, want.stdout, want.stderr)
		}
		if want.errorCode == "" {
			if got.ErrorCode != nil {
				t.Errorf("entry %d error_code = %q, want absent", index, *got.ErrorCode)
			}
		} else if got.ErrorCode == nil || *got.ErrorCode != want.errorCode {
			t.Errorf("entry %d error_code = %v, want %q", index, got.ErrorCode, want.errorCode)
		}
	}
}

func TestExecuteLogsInputFilePathButNotFileContentsAsStdin(t *testing.T) {
	root := t.TempDir()
	if code := Execute(context.Background(), []string{"flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "File input"}, strings.NewReader(""), io.Discard, io.Discard); code != 0 {
		t.Fatalf("initialize exit = %d", code)
	}
	inputPath := filepath.Join(t.TempDir(), "request.json")
	if err := os.WriteFile(inputPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{"flow", "status", "--root", root, "--input", "@" + inputPath}
	var stdout, stderr bytes.Buffer
	if code := Execute(context.Background(), args, strings.NewReader("stdin-must-remain-unread"), &stdout, &stderr); code != 0 {
		t.Fatalf("exit=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	entries := readExecutionLog(t, root)
	got := entries[len(entries)-1]
	if !reflect.DeepEqual(got.Arguments, args) || got.Stdin != "" || got.Stdout != stdout.String() || got.Stderr != stderr.String() {
		t.Fatalf("unexpected file-input transcript: %#v", got)
	}
}

func TestExecuteSkipsHelpRootlessAndUnknownOperations(t *testing.T) {
	root := t.TempDir()
	for _, args := range [][]string{
		{"flow", "status", "--root", root, "--help"},
		{"version", "--root", root},
		{"flow", "nope", "--root", root},
		{"flow", "status", "--root", "relative-root"},
	} {
		Execute(context.Background(), args, strings.NewReader(""), io.Discard, io.Discard)
	}
	if _, err := os.Stat(filepath.Join(root, ".fanloop", "log", "cli.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("excluded invocations created a log: %v", err)
	}
}

func TestExecuteUsesParsedLoggingControls(t *testing.T) {
	t.Run("persistent root before command is logged", func(t *testing.T) {
		root := t.TempDir()
		var stdout, stderr bytes.Buffer
		code := Execute(
			context.Background(),
			[]string{"--root", root, "flow", "status"},
			strings.NewReader(""), &stdout, &stderr,
		)
		if code != 1 || !strings.Contains(stderr.String(), `"code": "NOT_INITIALIZED"`) {
			t.Fatalf("exit=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
		}
		entries := readExecutionLog(t, root)
		if len(entries) != 1 || entries[0].CommandId != "flow.status" {
			t.Fatalf("entries = %#v, want one flow.status invocation", entries)
		}
	})

	t.Run("flag-looking input value is not a root", func(t *testing.T) {
		root := t.TempDir()
		var stdout, stderr bytes.Buffer
		code := Execute(
			context.Background(),
			[]string{"flow", "status", "--input", "--root=" + root},
			strings.NewReader(""), &stdout, &stderr,
		)
		if code != 2 || !strings.Contains(stderr.String(), `"code": "ROOT_REQUIRED"`) {
			t.Fatalf("exit=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
		}
		if _, err := os.Stat(filepath.Join(root, ".fanloop", "log", "cli.jsonl")); !os.IsNotExist(err) {
			t.Fatalf("flag value was treated as --root: %v", err)
		}
	})

	t.Run("flag-looking title value is not dry-run", func(t *testing.T) {
		root := t.TempDir()
		var stdout, stderr bytes.Buffer
		code := Execute(
			context.Background(),
			[]string{"flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "--dry-run"},
			strings.NewReader(""), &stdout, &stderr,
		)
		if code != 0 {
			t.Fatalf("exit=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
		}
		entries := readExecutionLog(t, root)
		if len(entries) != 1 || entries[0].DryRun {
			t.Fatalf("entries = %#v, want one non-dry-run invocation", entries)
		}
		if _, err := os.Stat(filepath.Join(root, ".fanloop", "flow", "state.json")); err != nil {
			t.Fatalf("command did not perform its non-dry-run write: %v", err)
		}
	})
}

func TestExecuteIgnoresExecutionLogFailure(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, ".fanloop", "log")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("sentinel"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(directory, "cli.jsonl")); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute(context.Background(), []string{"flow", "status", "--root", root}, strings.NewReader(""), &stdout, &stderr)
	if code != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), `"code": "NOT_INITIALIZED"`) {
		t.Fatalf("exit=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	content, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "sentinel" {
		t.Fatalf("symlink target changed: %q", content)
	}
}

func readExecutionLog(t *testing.T, root string) []storageidl.CLIExecutionLogEntry {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, ".fanloop", "log", "cli.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	entries := make([]storageidl.CLIExecutionLogEntry, len(lines))
	for index, line := range lines {
		if err := json.Unmarshal([]byte(line), &entries[index]); err != nil {
			t.Fatalf("decode line %d: %v: %q", index, err, line)
		}
		if err := entries[index].IsValid(); err != nil {
			t.Fatalf("validate line %d: %v", index, err)
		}
	}
	return entries
}

func TestExecuteRejectsRemovedLoopCommand(t *testing.T) {
	t.Setenv("FANLOOP_PYTHON", filepath.Join(t.TempDir(), "missing-python"))
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Execute(
		context.Background(),
		[]string{"loop", "feedback", "--root", root},
		strings.NewReader(""), &stdout, &stderr,
	)
	if code != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), `"code": "INVALID_ARGUMENT"`) {
		t.Fatalf("exit = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}
}

func TestExecuteRoutesTraceWithoutPython(t *testing.T) {
	t.Setenv("FANLOOP_PYTHON", filepath.Join(t.TempDir(), "missing-python"))
	root := t.TempDir()
	if code := Execute(context.Background(), []string{"flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "Native"}, strings.NewReader(""), io.Discard, io.Discard); code != 0 {
		t.Fatalf("initialize exit = %d", code)
	}
	var stdout, stderr bytes.Buffer
	code := Execute(context.Background(), []string{"trace", "status", "--root", root}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"event_count": 1`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestExecuteRoutesCardWithoutPython(t *testing.T) {
	t.Setenv("FANLOOP_PYTHON", filepath.Join(t.TempDir(), "missing-python"))
	root := t.TempDir()
	if code := Execute(context.Background(), []string{"flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "Native"}, strings.NewReader(""), io.Discard, io.Discard); code != 0 {
		t.Fatalf("initialize exit = %d", code)
	}
	var stdout, stderr bytes.Buffer
	code := Execute(context.Background(), []string{"card", "render", "--root", root, "--view", "current", "--format", "lark-json"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"schema": "2.0"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestExecuteRejectsRetiredPublicOperations(t *testing.T) {
	root := t.TempDir()
	if code := Execute(context.Background(), []string{"flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "Final"}, strings.NewReader(""), io.Discard, io.Discard); code != 0 {
		t.Fatalf("initialize exit = %d", code)
	}
	for _, args := range [][]string{
		{"update", "--action", "update"},
		{"trace", "record", "--root", root},
		{"trace", "migrate", "--root", root},
		{"card", "panorama", "--root", root},
	} {
		var stdout, stderr bytes.Buffer
		if code := Execute(context.Background(), args, strings.NewReader(""), &stdout, &stderr); code != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), `"code": "INVALID_ARGUMENT"`) {
			t.Fatalf("%v: exit=%d stdout=%s stderr=%s", args, code, stdout.String(), stderr.String())
		}
	}
}

func TestErrorMetaSkipsPersistentFlagsBeforeCommand(t *testing.T) {
	root := t.TempDir()
	var stderr bytes.Buffer
	code := Execute(context.Background(), []string{"--root", root, "flow", "nope"}, strings.NewReader(""), io.Discard, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), `"command": "flow.nope"`) || !strings.Contains(stderr.String(), `"requirement_root": "`+root+`"`) {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
}
