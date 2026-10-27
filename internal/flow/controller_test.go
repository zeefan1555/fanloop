package flow

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zeefan1555/fanloop/internal/controller"
	"github.com/zeefan1555/fanloop/internal/idl/storageidl"
	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/store"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

func TestPreparedInitializeNeverResumes(t *testing.T) {
	for _, phase := range []string{"prepared", "published", "pinned", "accepted"} {
		t.Run(phase, func(t *testing.T) {
			dataRoot, source := authorSource(t)
			t.Setenv("FANLOOP_DATA_HOME", dataRoot)
			t.Setenv("FANLOOP_CONFIG_SOURCE", source)
			root := t.TempDir()
			loaded, err := workflow.Load("linear-runtime")
			if err != nil {
				t.Fatal(err)
			}
			step, _ := loaded.Workflow.FirstStepID()
			now := time.Now().UTC()
			next := state.State{SchemaVersion: state.CurrentStateSchemaVersion, Requirement: state.Requirement{Title: "Refresh"}, Release: state.Release{Version: "test", Workflow: state.WorkflowRefFrom(loaded.Ref)}, CurrentStepID: &step, CurrentStepStatus: state.StepReady, CurrentStepSummary: "workflow initialized", Outputs: map[string]state.RegisteredOutput{}, Integrations: state.Integrations{}, LastEventID: "initial-event", CreatedAt: now, UpdatedAt: now}
			event := state.Event{SchemaVersion: state.CurrentEventSchemaVersion, ID: next.LastEventID, OccurredAt: now, Kind: state.EventFlowInitialized, Command: "flow.init", Workflow: next.Release.Workflow, Payload: state.Payload(state.FlowInitializedPayload{StepID: step, StepStatus: state.StepReady})}
			resources, err := controller.Plan(root, testControllerRelease, source, "", storageidl.ControllerBindingOperation_initialize, true)
			if err != nil {
				t.Fatal(err)
			}
			if err := resources.Prepare(context.Background(), func(string) error { return nil }); err != nil {
				t.Fatal(err)
			}
			local, failure := store.New(root)
			if failure != nil {
				t.Fatal(failure)
			}
			unlock, err := resources.LockPublication()
			if err != nil {
				t.Fatal(err)
			}
			interrupted := false
			failure = local.InitializeController(resources, next, event, func(current string) error {
				if current == phase {
					interrupted = true
					return errors.New("interrupt " + phase)
				}
				return nil
			})
			unlock()
			if failure == nil || !interrupted {
				t.Fatalf("did not reach initialization failure %s: %v", phase, failure)
			}
			paths := []string{controller.Journal, ".fanloop/flow/state.json", ".fanloop/output/state.json", ".fanloop/trace/events.jsonl", ".fanloop/configuration-snapshot"}
			before := map[string]string{}
			for _, path := range paths {
				content, err := os.ReadFile(filepath.Join(root, path))
				if err == nil {
					before[path] = string(content)
				} else if !os.IsNotExist(err) {
					t.Fatal(err)
				}
			}
			if phase == "accepted" && len(before) != len(paths) {
				t.Fatal("accepted fixture is missing first State, Output, Events or pin")
			}
			for _, changed := range []string{"same", "title", "workflow", "source-url"} {
				request := initRequest()
				switch changed {
				case "title":
					request.Requirement.Title = "Changed title"
				case "workflow":
					request.Workflow = "other-workflow"
				case "source-url":
					url := "https://example.com/changed"
					request.Requirement.SourceUrl = &url
				}
				if _, err := quietRuntime().Init(context.Background(), root, request, false); err == nil || !strings.Contains(err.Error(), "prepared") {
					t.Fatalf("%s request resumed %s: %v", changed, phase, err)
				}
				args := []string{"flow", "init", "--root", root, "--workflow", request.Workflow, "--title", request.Requirement.Title}
				if request.Requirement.SourceUrl != nil {
					args = append(args, "--source-url", *request.Requirement.SourceUrl)
				}
				command := exec.Command(filepath.Join(testControllerRelease, "bin", "fanloop"), args...)
				command.Env = controller.Environment(dataRoot, source, "")
				output, err := command.CombinedOutput()
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.ExitCode() != 5 || !strings.Contains(string(output), "prepared") {
					t.Fatalf("public init resumed %s/%s: %v\n%s", phase, changed, err, output)
				}
			}
			if failure := local.InitializeController(resources, next, event, nil); failure == nil {
				t.Fatal("Store resumed prepared initialize")
			}
			if binary, err := local.ProtectController(context.Background(), testControllerRelease, nil); err == nil || binary != "" {
				t.Fatalf("protect resumed initialize: %s %v", binary, err)
			}
			for _, path := range paths {
				content, err := os.ReadFile(filepath.Join(root, path))
				original, existed := before[path]
				if existed && (err != nil || string(content) != original) || !existed && !os.IsNotExist(err) {
					t.Fatalf("retry changed accepted facts or journal %s: %v", path, err)
				}
			}
		})
	}
}
