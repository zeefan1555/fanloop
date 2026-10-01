package verify

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/idl/verifyidl"
)

func TestSmokeResultInputUsesStatusDefinedConditionsAndRoute(t *testing.T) {
	raw := []byte(`{"ok":true,"data":{"state":{"status":"running","current":{"context":{"step_id":"step-from-status"},"conditions":[{"id":"path-condition","output":{"type":"path"}},{"id":"enum-condition","output":{"type":"enum_value","values":["approved"]}},{"id":"count-condition","output":{"type":"integer","minimum":3,"maximum":5}}],"available_routes":[{"direction":"loop","when":{"any_of":[["path-condition"]]},"route":{"back_step_id":"step-from-status"}},{"direction":"flow","when":{"any_of":[["path-condition","enum-condition","count-condition"]]},"route":{"next_step_id":"next-from-status"}}]}}}}`)
	status, err := parseSmokeStatus(raw)
	if err != nil {
		t.Fatal(err)
	}
	input, expected, err := smokeResultInput(status)
	if err != nil {
		t.Fatal(err)
	}
	if expected != "next-from-status" {
		t.Fatalf("expected status = %q", expected)
	}
	var request struct {
		StepID           string `json:"step_id"`
		ConditionResults []struct {
			ConditionID string `json:"condition_id"`
			Output      struct {
				Type  string `json:"type"`
				Value any    `json:"value"`
			} `json:"output"`
		} `json:"condition_results"`
		Route map[string]any `json:"route"`
	}
	if err := json.Unmarshal([]byte(input), &request); err != nil {
		t.Fatal(err)
	}
	if request.StepID != "step-from-status" || len(request.ConditionResults) != 3 || request.Route["next_step_id"] != "next-from-status" {
		t.Fatalf("generated request = %#v", request)
	}
	if request.ConditionResults[1].Output.Value != "approved" || request.ConditionResults[2].Output.Value != float64(3) {
		t.Fatalf("generated outputs = %#v", request.ConditionResults)
	}
}

func TestBusinessFingerprintIgnoresCLIExecutionLog(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, ".fanloop", "log", "cli.jsonl"), "first\n")
	before, err := businessFingerprint(root)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(root, ".fanloop", "log", "cli.jsonl"), "second\n")
	afterLog, err := businessFingerprint(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, afterLog) {
		t.Fatalf("CLI audit changed business fingerprint: before=%v after=%v", before, afterLog)
	}

	writeTestFile(t, filepath.Join(root, ".fanloop", "flow", "state.json"), "{}\n")
	afterState, err := businessFingerprint(root)
	if err != nil {
		t.Fatal(err)
	}
	if changed := changedPaths(afterLog, afterState); !reflect.DeepEqual(changed, []string{".fanloop/flow/state.json"}) {
		t.Fatalf("business changes = %v", changed)
	}
}

func TestCleanupRemovesOnlyCanonicalRunWorkAndPreservesEvidence(t *testing.T) {
	dataRoot := t.TempDir()
	runID := "0123456789abcdef0123456789abcdef"
	workRoot := filepath.Join(dataRoot, "verification", "work", runID)
	evidence := filepath.Join(dataRoot, "verification", "runs", runID, "manifest.json")
	outside := filepath.Join(dataRoot, "outside")
	writeTestFile(t, filepath.Join(workRoot, "requirement", "owned"), "owned")
	writeTestFile(t, evidence, "evidence")
	writeTestFile(t, outside, "preserve")
	runtime := Runtime{DataRoot: dataRoot}

	dry, err := runtime.Cleanup(context.Background(), true, &verifyidl.VerifyCleanupRequest{RunId: runID})
	if err != nil || dry.Removed || !reflect.DeepEqual(dry.Paths, []string{workRoot}) {
		t.Fatalf("dry cleanup = %#v, %v", dry, err)
	}
	if _, err := os.Stat(workRoot); err != nil {
		t.Fatalf("dry cleanup removed work: %v", err)
	}

	done, err := runtime.Cleanup(context.Background(), false, &verifyidl.VerifyCleanupRequest{RunId: runID})
	if err != nil || !done.Removed {
		t.Fatalf("cleanup = %#v, %v", done, err)
	}
	if _, err := os.Stat(workRoot); !os.IsNotExist(err) {
		t.Fatalf("work root remains: %v", err)
	}
	for _, path := range []string{evidence, outside} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("cleanup removed %s: %v", path, err)
		}
	}
}

func TestCleanupRejectsUntrustedRunID(t *testing.T) {
	dataRoot := t.TempDir()
	outside := filepath.Join(dataRoot, "outside")
	writeTestFile(t, outside, "preserve")
	runtime := Runtime{DataRoot: dataRoot}
	_, err := runtime.Cleanup(context.Background(), false, &verifyidl.VerifyCleanupRequest{RunId: "../outside"})
	var failure *erroridl.PublicError
	if !errors.As(err, &failure) || failure.Code != erroridl.ErrorCode_INVALID_ARGUMENT {
		t.Fatalf("cleanup error = %v", err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("invalid cleanup changed outside file: %v", err)
	}
}

func TestSnapshotRejectsEvidenceInsideRequirementState(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, ".fanloop", "snapshot")
	runtime := Runtime{}
	_, err := runtime.Snapshot(context.Background(), root, &verifyidl.VerifySnapshotRequest{EvidenceDir: &destination})
	var failure *erroridl.PublicError
	if !errors.As(err, &failure) || failure.Code != erroridl.ErrorCode_INVALID_ARGUMENT {
		t.Fatalf("snapshot error = %v", err)
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatalf("snapshot created recursive destination: %v", err)
	}
}

func TestSnapshotFreezesOneConfigurationForEveryPublicObservation(t *testing.T) {
	repository, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dataRoot := t.TempDir()
	workflowRoot := filepath.Join(dataRoot, "source", "skills")
	if err := os.CopyFS(filepath.Join(workflowRoot, "linear-runtime"), os.DirFS(filepath.Join(repository, "internal", "workflow", "testdata", "linear-runtime"))); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_CONFIG_SOURCE", filepath.Join(dataRoot, "source"))
	marker := filepath.Join(workflowRoot, "linear-runtime", "skills", "test-guide", "SKILL.md")
	want, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	var frozenRoot string
	calls := 0
	runtime := Runtime{
		DataRoot:     dataRoot,
		WorkflowRoot: t.TempDir(), // The binding may change after Runtime construction.
		NewRunID:     func() string { return "0123456789abcdef0123456789abcdef" },
		Run: func(_ context.Context, executable string, env []string, args []string) (commandResult, error) {
			calls++
			childDataRoot := environmentValue(env, "FANLOOP_DATA_HOME")
			if childDataRoot == "" || childDataRoot == dataRoot {
				t.Fatalf("snapshot child data root = %q", childDataRoot)
			}
			if frozenRoot == "" {
				frozenRoot = childDataRoot
				if err := os.WriteFile(marker, []byte("changed after freeze\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			} else if childDataRoot != frozenRoot {
				t.Fatalf("snapshot child roots changed: %q != %q", childDataRoot, frozenRoot)
			}
			frozenMarker := filepath.Join(childDataRoot, "source", "skills", "linear-runtime", "skills", "test-guide", "SKILL.md")
			got, err := os.ReadFile(frozenMarker)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("frozen configuration changed: %v", err)
			}
			if environmentValue(env, "FANLOOP_CONFIG_SOURCE") != filepath.Join(childDataRoot, "source") {
				t.Fatal("snapshot child did not use its frozen configuration source")
			}
			return commandResult{Argv: append([]string{executable}, args...), StartedAt: time.Now(), Stdout: []byte("{}\n")}, nil
		},
	}
	root := t.TempDir()
	destination := filepath.Join(t.TempDir(), "snapshot")
	if _, err := runtime.Snapshot(context.Background(), root, &verifyidl.VerifySnapshotRequest{EvidenceDir: &destination}); err != nil {
		t.Fatal(err)
	}
	if calls != 5 {
		t.Fatalf("snapshot observations = %d, want 5", calls)
	}
	if _, err := os.Stat(frozenRoot); !os.IsNotExist(err) {
		t.Fatalf("frozen configuration root remains: %v", err)
	}
}

func TestSmokeEnvironmentUsesCurrentBindingInsteadOfCachedRoot(t *testing.T) {
	data, source, work := t.TempDir(), t.TempDir(), t.TempDir()
	writeTestFile(t, filepath.Join(source, "skills", "marker"), "current binding")
	if err := os.Mkdir(filepath.Join(data, "config"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(source, filepath.Join(data, "config", "current")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FANLOOP_CONFIG_SOURCE", "")
	if err := os.Unsetenv("FANLOOP_CONFIG_SOURCE"); err != nil {
		t.Fatal(err)
	}
	runtime := Runtime{DataRoot: data, WorkflowRoot: t.TempDir()}
	if _, err := runtime.isolatedEnvironment(work); err != nil {
		t.Fatal(err)
	}
	if content, err := os.ReadFile(filepath.Join(work, "data", "source", "skills", "marker")); err != nil || string(content) != "current binding" {
		t.Fatalf("snapshot did not use the current binding: %q, %v", content, err)
	}
}

func TestFinishSmokeCleanupStopsWhenDryRunFails(t *testing.T) {
	dataRoot := t.TempDir()
	runID := "0123456789abcdef0123456789abcdef"
	workRoot := filepath.Join(dataRoot, "verification", "work", runID)
	writeTestFile(t, filepath.Join(workRoot, "requirement", "owned"), "owned")
	evidence, err := newEvidenceRun(dataRoot, filepath.Join(dataRoot, "evidence"), runID, filepath.Join(workRoot, "requirement"), workRoot, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	runtime := Runtime{DataRoot: dataRoot, Run: func(_ context.Context, executable string, _ []string, args []string) (commandResult, error) {
		calls++
		return commandResult{Argv: append([]string{executable}, args...), StartedAt: time.Now(), ExitCode: 2, Stderr: []byte(`{"ok":false,"error":{"code":"INTERNAL","message":"dry-run failed"}}`)}, nil
	}}
	response := &verifyidl.VerifySmokeResponse{}
	runtime.finishSmokeCleanup(context.Background(), nil, evidence, response, runID, workRoot)
	if calls != 1 || response.CleanupComplete || evidence.manifest.Cleanup.DryRunPassed || evidence.manifest.Cleanup.Removed {
		t.Fatalf("cleanup state = calls %d, response %#v, manifest %#v", calls, response, evidence.manifest.Cleanup)
	}
	if _, err := os.Stat(workRoot); err != nil {
		t.Fatalf("failed dry-run removed work: %v", err)
	}
}

func TestSyntheticOutputHonorsURLListMaximum(t *testing.T) {
	maximum := int32(0)
	value, err := syntheticOutput("urls", "url_list", nil, nil, nil, nil, &maximum)
	if err != nil || len(value.([]string)) != 0 {
		t.Fatalf("synthetic output = %#v, %v", value, err)
	}
}

func TestManifestOutcomeReadsSchemaOneRegularFile(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "manifest.json")
	writeTestFile(t, path, "{\"schema_version\":1,\"outcome\":\"passed\",\"run_id\":\"extra\"}\n")
	outcome, err := ManifestOutcome(path)
	if err != nil || outcome != "passed" {
		t.Fatalf("outcome = %q, err = %v", outcome, err)
	}

	relative, err := filepath.Rel(mustGetwd(t), path)
	if err != nil || filepath.IsAbs(relative) {
		t.Fatalf("relative path = %q, err = %v", relative, err)
	}
	if _, err := ManifestOutcome(relative); err == nil {
		t.Fatal("relative path accepted")
	}

	link := filepath.Join(directory, "manifest-link.json")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ManifestOutcome(link); err == nil {
		t.Fatal("symlink accepted")
	}

	writeTestFile(t, filepath.Join(directory, "schema-zero.json"), "{\"schema_version\":0,\"outcome\":\"passed\"}\n")
	if _, err := ManifestOutcome(filepath.Join(directory, "schema-zero.json")); err == nil {
		t.Fatal("schema_version 0 accepted")
	}
	writeTestFile(t, filepath.Join(directory, "approved.json"), "{\"schema_version\":1,\"outcome\":\"approved\"}\n")
	if _, err := ManifestOutcome(filepath.Join(directory, "approved.json")); err == nil {
		t.Fatal("outcome approved accepted")
	}
	writeTestFile(t, filepath.Join(directory, "not-manifest.json"), "{\"schema_version\":1,\"outcome\":\"passed\"}\n")
	if _, err := ManifestOutcome(filepath.Join(directory, "not-manifest.json")); err == nil {
		t.Fatal("non-manifest file name accepted")
	}
	if _, err := ManifestOutcome(filepath.Join(directory, "missing", "manifest.json")); err == nil {
		t.Fatal("missing manifest accepted")
	}
	schemaZero := filepath.Join(t.TempDir(), "manifest.json")
	writeTestFile(t, schemaZero, "{\"schema_version\":0,\"outcome\":\"passed\"}\n")
	if _, err := ManifestOutcome(schemaZero); err == nil {
		t.Fatal("schema_version 0 manifest accepted")
	}
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return directory
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func environmentValue(environment []string, name string) string {
	prefix := name + "="
	for _, item := range environment {
		if strings.HasPrefix(item, prefix) {
			return strings.TrimPrefix(item, prefix)
		}
	}
	return ""
}
