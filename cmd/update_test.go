package cmd

import (
	"bytes"
	"context"
	"fmt"
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
