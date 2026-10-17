package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/configuration"
	"github.com/zeefan1555/fanloop/internal/distribution"
	"github.com/zeefan1555/fanloop/internal/fornaxmirror"
)

func TestUpdateAndUpdateConfigPrintReleasedWorkflows(t *testing.T) {
	dataRoot := t.TempDir()
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_USE_FORNAX", "1")
	t.Setenv(envUpdateReexec, "1")
	if err := os.WriteFile(filepath.Join(dataRoot, "fornax.json"), []byte(`{"sources":[{"workspace_id":"1","key_prefix":"fanloop.workflow.","access":"read_write"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	unsetUpdateConfigSource(t)
	previousRelease := applyReleaseUpdate
	previousConfig := applyFornaxConfigSync
	t.Cleanup(func() {
		applyReleaseUpdate = previousRelease
		applyFornaxConfigSync = previousConfig
	})

	releaseCalls := 0
	configCalls := 0
	applyReleaseUpdate = func(context.Context, string) (distribution.Result, error) {
		releaseCalls++
		return distribution.Result{Version: "1.2.3"}, nil
	}
	applyFornaxConfigSync = func(context.Context, string) (fornaxmirror.SyncResult, error) {
		configCalls++
		return fornaxmirror.SyncResult{
			Outcome: fornaxmirror.OutcomeUpdated,
			Workflows: []fornaxmirror.SyncedWorkflow{
				{ID: "pstack-native", Version: "1.0.2"},
				{ID: "linear-runtime", Version: "1.0.0"},
			},
		}, nil
	}

	var stdout, stderr bytes.Buffer
	if code := Execute(context.Background(), []string{"update"}, strings.NewReader(""), &stdout, &stderr); code != 0 || stderr.Len() != 0 {
		t.Fatalf("update exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	want := "updated 1.2.3\nfanloop configuration updated: linear-runtime@1.0.0, pstack-native@1.0.2\n"
	if stdout.String() != want || releaseCalls != 1 || configCalls != 1 {
		t.Fatalf("update stdout=%q release=%d config=%d", stdout.String(), releaseCalls, configCalls)
	}

	stdout.Reset()
	applyFornaxConfigSync = func(context.Context, string) (fornaxmirror.SyncResult, error) {
		configCalls++
		return fornaxmirror.SyncResult{
			Outcome: fornaxmirror.OutcomeUnchanged,
			Workflows: []fornaxmirror.SyncedWorkflow{
				{ID: "pstack-native", Version: "1.0.2"},
				{ID: "linear-runtime", Version: "1.0.0"},
			},
		}, nil
	}
	if code := Execute(context.Background(), []string{"update", "config"}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("update config exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	want = "fanloop configuration is already up to date: linear-runtime@1.0.0, pstack-native@1.0.2\n"
	if stdout.String() != want || releaseCalls != 1 || configCalls != 2 {
		t.Fatalf("update config stdout=%q release=%d config=%d", stdout.String(), releaseCalls, configCalls)
	}
}

func TestFailedUpdateAndUpdateConfigExitNonZero(t *testing.T) {
	dataRoot := t.TempDir()
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_USE_FORNAX", "1")
	t.Setenv(envUpdateReexec, "1")
	if err := os.WriteFile(filepath.Join(dataRoot, "fornax.json"), []byte(`{"sources":[{"workspace_id":"1","key_prefix":"fanloop.workflow.","access":"read_write"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	unsetUpdateConfigSource(t)
	previousRelease := applyReleaseUpdate
	previousConfig := applyFornaxConfigSync
	t.Cleanup(func() {
		applyReleaseUpdate = previousRelease
		applyFornaxConfigSync = previousConfig
	})
	applyReleaseUpdate = func(context.Context, string) (distribution.Result, error) {
		return distribution.Result{Version: "1.2.3"}, nil
	}
	applyFornaxConfigSync = func(context.Context, string) (fornaxmirror.SyncResult, error) {
		return fornaxmirror.SyncResult{}, fmt.Errorf("discover Fornax workflows: boom")
	}
	var stdout, stderr bytes.Buffer
	if code := Execute(context.Background(), []string{"update"}, strings.NewReader(""), &stdout, &stderr); code != 1 {
		t.Fatalf("update exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stdout.String() != "updated 1.2.3\n" || stderr.String() != "fanloop update: discover Fornax workflows: boom\n" {
		t.Fatalf("update stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute(context.Background(), []string{"update", "config"}, strings.NewReader(""), &stdout, &stderr); code != 1 {
		t.Fatalf("update config exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stdout.Len() != 0 || stderr.String() != "fanloop update: discover Fornax workflows: boom\n" {
		t.Fatalf("update config stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestUpdateConfigUsesGitWithoutFornaxFile(t *testing.T) {
	dataRoot := updateGitFixture(t)
	var stdout, stderr bytes.Buffer
	if code := Execute(context.Background(), []string{"update", "config"}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	sha := updateCurrentCommit(t, dataRoot)
	want := fmt.Sprintf("fanloop configuration updated: %s (linear-runtime, runtime-controls)\n", sha)
	if stdout.String() != want || stderr.Len() != 0 || len(sha) != 40 {
		t.Fatalf("stdout=%q stderr=%q sha=%q", stdout.String(), stderr.String(), sha)
	}
	if _, err := os.Stat(filepath.Join(dataRoot, "fornax.json")); !os.IsNotExist(err) {
		t.Fatal("git sync created fornax.json")
	}
}

func TestUpdateConfigKeepsGitWhenFornaxFileHasNoOptIn(t *testing.T) {
	dataRoot := updateGitFixture(t)
	if err := os.WriteFile(filepath.Join(dataRoot, "fornax.json"), []byte(`{"sources":[{"workspace_id":"1","key_prefix":"fanloop.workflow.","access":"read_write"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Execute(context.Background(), []string{"update", "config"}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	sha := updateCurrentCommit(t, dataRoot)
	if !strings.Contains(stdout.String(), sha) || strings.Contains(stdout.String(), "@") {
		t.Fatalf("stdout=%q sha=%s", stdout.String(), sha)
	}
}

func TestUpdateConfigUsesFornaxOnlyWhenOptedIn(t *testing.T) {
	dataRoot := t.TempDir()
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_USE_FORNAX", "1")
	unsetUpdateConfigSource(t)
	if err := os.WriteFile(filepath.Join(dataRoot, "fornax.json"), []byte(`{"sources":[{"workspace_id":"1","key_prefix":"fanloop.workflow.","access":"read_write"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	previousGit := applyGitConfigSync
	previousFornax := applyFornaxConfigSync
	t.Cleanup(func() {
		applyGitConfigSync = previousGit
		applyFornaxConfigSync = previousFornax
	})
	gitCalls := 0
	applyGitConfigSync = func(context.Context, string) (configuration.SyncResult, error) {
		gitCalls++
		return configuration.SyncResult{}, fmt.Errorf("git sync should not run")
	}
	applyFornaxConfigSync = func(context.Context, string) (fornaxmirror.SyncResult, error) {
		return fornaxmirror.SyncResult{Outcome: fornaxmirror.OutcomeUpdated, Workflows: []fornaxmirror.SyncedWorkflow{{ID: "linear-runtime", Version: "1.0.0"}}}, nil
	}
	var stdout, stderr bytes.Buffer
	if code := Execute(context.Background(), []string{"update", "config"}, strings.NewReader(""), &stdout, &stderr); code != 0 || gitCalls != 0 {
		t.Fatalf("exit=%d gitCalls=%d stdout=%q stderr=%q", code, gitCalls, stdout.String(), stderr.String())
	}
	if stdout.String() != "fanloop configuration updated: linear-runtime@1.0.0\n" {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestUpdateConfigGitFailureLeavesCurrentUnchanged(t *testing.T) {
	dataRoot := updateGitFixture(t)
	var stdout, stderr bytes.Buffer
	if code := Execute(context.Background(), []string{"update", "config"}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("setup exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	current := updateReadlink(t, filepath.Join(dataRoot, "config", "current"))
	t.Setenv("GIT_CONFIG_KEY_0", "url."+filepath.Join(t.TempDir(), "missing")+".insteadOf")
	stdout.Reset()
	stderr.Reset()
	if code := Execute(context.Background(), []string{"update", "config"}, strings.NewReader(""), &stdout, &stderr); code != 1 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stdout.Len() != 0 || !strings.HasPrefix(stderr.String(), "fanloop update: configuration fetch failed:") {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if again := updateReadlink(t, filepath.Join(dataRoot, "config", "current")); again != current {
		t.Fatalf("current changed from %s to %s", current, again)
	}
}

func updateGitFixture(t *testing.T) string {
	t.Helper()
	source := t.TempDir()
	for _, id := range []string{"linear-runtime", "runtime-controls"} {
		destination := filepath.Join(source, "fanloop-workflow", id)
		if err := os.MkdirAll(destination, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.CopyFS(destination, os.DirFS(filepath.Join("..", "internal", "workflow", "testdata", id))); err != nil {
			t.Fatal(err)
		}
	}
	updateGit(t, source, "init", "-b", "main")
	updateGit(t, source, "config", "user.name", "Update Test")
	updateGit(t, source, "config", "user.email", "update@example.test")
	updateGit(t, source, "add", ".")
	updateGit(t, source, "commit", "-m", "fixture")
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "url."+source+".insteadOf")
	t.Setenv("GIT_CONFIG_VALUE_0", configuration.Repository)
	t.Setenv("FANLOOP_USE_FORNAX", "")
	unsetUpdateConfigSource(t)
	dataRoot := t.TempDir()
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	return dataRoot
}

func updateCurrentCommit(t *testing.T, dataRoot string) string {
	t.Helper()
	target := updateReadlink(t, filepath.Join(dataRoot, "config", "current"))
	manifest, err := configuration.Read(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(target, string(os.PathSeparator)+"snapshots"+string(os.PathSeparator)+manifest.CommitSHA) && filepath.Base(target) != manifest.CommitSHA {
		t.Fatalf("current %s does not end at snapshot %s", target, manifest.CommitSHA)
	}
	return manifest.CommitSHA
}

func updateReadlink(t *testing.T, path string) string {
	t.Helper()
	target, err := os.Readlink(path)
	if err != nil {
		t.Fatal(err)
	}
	return target
}

func updateGit(t *testing.T, directory string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %s %v", args, output, err)
	}
}

func TestChangedReleaseExecsUpdatedBinaryOnce(t *testing.T) {
	dataRoot := t.TempDir()
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_USE_FORNAX", "")
	t.Setenv(envUpdateReexec, "")
	if err := os.Unsetenv(envUpdateReexec); err != nil {
		t.Fatal(err)
	}
	unsetUpdateConfigSource(t)
	restoreUpdateHooks(t)

	applyReleaseUpdate = func(context.Context, string) (distribution.Result, error) {
		return distribution.Result{Version: "1.2.4"}, nil
	}
	syncCalls := 0
	applyGitConfigSync = func(context.Context, string) (configuration.SyncResult, error) {
		syncCalls++
		return configuration.SyncResult{}, fmt.Errorf("in-process sync should not run")
	}
	binary := filepath.Join(dataRoot, "current", "bin", "fanloop")
	execCalls := 0
	var gotArgv []string
	var gotEnv []string
	execUpdatedBinary = func(argv0 string, argv []string, env []string) error {
		execCalls++
		if argv0 != binary {
			t.Errorf("argv0=%s", argv0)
		}
		gotArgv = append([]string{}, argv...)
		gotEnv = append([]string{}, env...)
		return nil
	}

	var stdout, stderr bytes.Buffer
	if code := Execute(context.Background(), []string{"update"}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if execCalls != 1 || syncCalls != 0 {
		t.Fatalf("exec=%d sync=%d stdout=%q stderr=%q", execCalls, syncCalls, stdout.String(), stderr.String())
	}
	if strings.Join(gotArgv, "\x00") != strings.Join([]string{binary, "update", "config"}, "\x00") {
		t.Fatalf("argv=%#v", gotArgv)
	}
	if !updateEnvHas(gotEnv, envUpdateReexec+"=1") {
		t.Fatalf("env missing %s=1", envUpdateReexec)
	}
	if stdout.String() != "updated 1.2.4\n" || stderr.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestUnchangedReleaseSyncsInProcess(t *testing.T) {
	dataRoot := t.TempDir()
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_USE_FORNAX", "")
	if err := os.Unsetenv(envUpdateReexec); err != nil {
		t.Fatal(err)
	}
	unsetUpdateConfigSource(t)
	restoreUpdateHooks(t)

	applyReleaseUpdate = func(context.Context, string) (distribution.Result, error) {
		return distribution.Result{Version: "1.2.4", Unchanged: true}, nil
	}
	syncCalls := 0
	applyGitConfigSync = func(context.Context, string) (configuration.SyncResult, error) {
		syncCalls++
		return configuration.SyncResult{CommitSHA: strings.Repeat("a", 40), WorkflowIDs: []string{"linear-runtime"}}, nil
	}
	execCalls := 0
	execUpdatedBinary = func(string, []string, []string) error {
		execCalls++
		return fmt.Errorf("exec should not run")
	}

	var stdout, stderr bytes.Buffer
	if code := Execute(context.Background(), []string{"update"}, strings.NewReader(""), &stdout, &stderr); code != 0 || execCalls != 0 || syncCalls != 1 {
		t.Fatalf("exit=%d exec=%d sync=%d stdout=%q stderr=%q", code, execCalls, syncCalls, stdout.String(), stderr.String())
	}
	want := "unchanged 1.2.4\nfanloop configuration updated: " + strings.Repeat("a", 40) + " (linear-runtime)\n"
	if stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestUpdateReexecMarkerSkipsExec(t *testing.T) {
	dataRoot := t.TempDir()
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_USE_FORNAX", "")
	t.Setenv(envUpdateReexec, "1")
	unsetUpdateConfigSource(t)
	restoreUpdateHooks(t)

	applyReleaseUpdate = func(context.Context, string) (distribution.Result, error) {
		return distribution.Result{Version: "1.2.4"}, nil
	}
	syncCalls := 0
	applyGitConfigSync = func(context.Context, string) (configuration.SyncResult, error) {
		syncCalls++
		return configuration.SyncResult{CommitSHA: strings.Repeat("b", 40), WorkflowIDs: []string{"linear-runtime"}}, nil
	}
	execCalls := 0
	execUpdatedBinary = func(string, []string, []string) error {
		execCalls++
		return nil
	}

	var stdout, stderr bytes.Buffer
	if code := Execute(context.Background(), []string{"update"}, strings.NewReader(""), &stdout, &stderr); code != 0 || execCalls != 0 || syncCalls != 1 {
		t.Fatalf("exit=%d exec=%d sync=%d stdout=%q stderr=%q", code, execCalls, syncCalls, stdout.String(), stderr.String())
	}
	if !strings.HasPrefix(stdout.String(), "updated 1.2.4\n") || stderr.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestUpdateExecErrorFallsBackToInProcessSync(t *testing.T) {
	dataRoot := t.TempDir()
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_USE_FORNAX", "")
	if err := os.Unsetenv(envUpdateReexec); err != nil {
		t.Fatal(err)
	}
	unsetUpdateConfigSource(t)
	restoreUpdateHooks(t)

	applyReleaseUpdate = func(context.Context, string) (distribution.Result, error) {
		return distribution.Result{Version: "1.2.4"}, nil
	}
	execUpdatedBinary = func(string, []string, []string) error {
		return errors.New("exec format error")
	}
	warning := "warning: could not start the updated binary (exec format error); synchronizing with the running binary\n"

	syncCalls := 0
	applyGitConfigSync = func(context.Context, string) (configuration.SyncResult, error) {
		syncCalls++
		return configuration.SyncResult{CommitSHA: strings.Repeat("c", 40), WorkflowIDs: []string{"linear-runtime"}}, nil
	}
	var stdout, stderr bytes.Buffer
	if code := Execute(context.Background(), []string{"update"}, strings.NewReader(""), &stdout, &stderr); code != 0 || syncCalls != 1 {
		t.Fatalf("success exit=%d sync=%d stdout=%q stderr=%q", code, syncCalls, stdout.String(), stderr.String())
	}
	if stderr.String() != warning || !strings.Contains(stdout.String(), "fanloop configuration updated:") {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	applyGitConfigSync = func(context.Context, string) (configuration.SyncResult, error) {
		syncCalls++
		return configuration.SyncResult{}, errors.New("sync failed")
	}
	if code := Execute(context.Background(), []string{"update"}, strings.NewReader(""), &stdout, &stderr); code != 1 || syncCalls != 2 {
		t.Fatalf("failure exit=%d sync=%d stdout=%q stderr=%q", code, syncCalls, stdout.String(), stderr.String())
	}
	if stderr.String() != warning+"fanloop update: sync failed\n" {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestReleaseUpdateReexecPropagatesStubExit(t *testing.T) {
	if os.Getenv("FANLOOP_UPDATE_EXEC_HELPER") == "1" {
		helperReleaseUpdateExec()
		os.Exit(2)
	}
	for _, want := range []int{0, 1} {
		t.Run(fmt.Sprintf("exit-%d", want), func(t *testing.T) {
			dataRoot := t.TempDir()
			record := t.TempDir()
			command := exec.Command(os.Args[0], "-test.run=^TestReleaseUpdateReexecPropagatesStubExit$", "-test.v")
			command.Env = append(updateEnvWithout("FANLOOP_DATA_HOME", envUpdateReexec, "FANLOOP_UPDATE_EXEC_HELPER", "FANLOOP_UPDATE_EXEC_RECORD", "FANLOOP_STUB_EXIT"),
				"FANLOOP_UPDATE_EXEC_HELPER=1",
				"FANLOOP_DATA_HOME="+dataRoot,
				"FANLOOP_UPDATE_EXEC_RECORD="+record,
				fmt.Sprintf("FANLOOP_STUB_EXIT=%d", want),
			)
			output, err := command.CombinedOutput()
			argv, readErr := os.ReadFile(filepath.Join(record, "argv"))
			if readErr != nil {
				t.Fatalf("stub did not run: %v\n%s", err, output)
			}
			lines := strings.Split(strings.TrimSuffix(string(argv), "\n"), "\n")
			// The child test process has its own TestMain data root. The stub
			// records the binary that syscall.Exec actually started.
			if len(lines) != 3 || !strings.HasSuffix(lines[0], string(os.PathSeparator)+"current"+string(os.PathSeparator)+"bin"+string(os.PathSeparator)+"fanloop") || lines[1] != "update" || lines[2] != "config" || !filepath.IsAbs(lines[0]) {
				t.Fatalf("argv=%q", argv)
			}
			envText, envErr := os.ReadFile(filepath.Join(record, "env"))
			if envErr != nil {
				t.Fatal(envErr)
			}
			if !strings.Contains(string(envText), envUpdateReexec+"=1\n") && !strings.Contains(string(envText), envUpdateReexec+"=1") {
				t.Fatalf("env missing marker:\n%s", envText)
			}
			if want == 0 {
				if err != nil {
					t.Fatalf("exit err=%v\n%s", err, output)
				}
				return
			}
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
				t.Fatalf("exit err=%v\n%s", err, output)
			}
		})
	}
}

func helperReleaseUpdateExec() {
	fail := func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
		os.Exit(2)
	}
	dataRoot := os.Getenv("FANLOOP_DATA_HOME")
	record := os.Getenv("FANLOOP_UPDATE_EXEC_RECORD")
	exitCode := os.Getenv("FANLOOP_STUB_EXIT")
	if dataRoot == "" || record == "" || (exitCode != "0" && exitCode != "1") {
		fail("helper env data=%q record=%q exit=%q", dataRoot, record, exitCode)
	}
	binary := filepath.Join(dataRoot, "current", "bin", "fanloop")
	if err := os.MkdirAll(filepath.Dir(binary), 0o755); err != nil {
		fail("mkdir: %v", err)
	}
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$0\" \"$@\" > %q\nprintenv > %q\nexit %s\n", filepath.Join(record, "argv"), filepath.Join(record, "env"), exitCode)
	if err := os.WriteFile(binary, []byte(script), 0o755); err != nil {
		fail("write stub: %v", err)
	}
	if err := os.Unsetenv(envUpdateReexec); err != nil {
		fail("unset marker: %v", err)
	}
	applyReleaseUpdate = func(context.Context, string) (distribution.Result, error) {
		return distribution.Result{Version: "9.9.9"}, nil
	}
	if err := runReleaseUpdate(context.Background(), io.Discard, io.Discard); err != nil {
		fail("exec returned: %v", err)
	}
	fail("exec returned nil")
}

func restoreUpdateHooks(t *testing.T) {
	t.Helper()
	previousRelease := applyReleaseUpdate
	previousGit := applyGitConfigSync
	previousExec := execUpdatedBinary
	t.Cleanup(func() {
		applyReleaseUpdate = previousRelease
		applyGitConfigSync = previousGit
		execUpdatedBinary = previousExec
	})
}

func updateEnvHas(env []string, want string) bool {
	for _, item := range env {
		if item == want {
			return true
		}
	}
	return false
}

func updateEnvWithout(keys ...string) []string {
	skip := map[string]bool{}
	for _, key := range keys {
		skip[key] = true
	}
	var kept []string
	for _, item := range os.Environ() {
		name, _, _ := strings.Cut(item, "=")
		if skip[name] {
			continue
		}
		kept = append(kept, item)
	}
	return kept
}

func unsetUpdateConfigSource(t *testing.T) {
	t.Helper()
	value, exists := os.LookupEnv("FANLOOP_CONFIG_SOURCE")
	if err := os.Unsetenv("FANLOOP_CONFIG_SOURCE"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if exists {
			_ = os.Setenv("FANLOOP_CONFIG_SOURCE", value)
		} else {
			_ = os.Unsetenv("FANLOOP_CONFIG_SOURCE")
		}
	})
}
