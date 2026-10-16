package flow

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/configuration"
	"github.com/zeefan1555/fanloop/internal/store"
)

func TestFreshInitPullsGitEvenWhenFornaxConfigExists(t *testing.T) {
	source := gitWorkflowFixture(t)
	dataRoot := t.TempDir()
	writeFornaxFile(t, dataRoot)
	useGitRemote(t, source)
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	if err := os.Unsetenv(envUseFornax); err != nil {
		t.Fatal(err)
	}
	if err := os.Unsetenv(envNoAutoConfig); err != nil {
		t.Fatal(err)
	}
	runtime := quietRuntime()
	firstRoot := t.TempDir()
	if _, err := runtime.Init(context.Background(), firstRoot, initRequest(), false); err != nil {
		t.Fatal(err)
	}
	current, err := os.Readlink(filepath.Join(dataRoot, "config", "current"))
	if err != nil {
		t.Fatal(err)
	}
	if len(filepath.Base(current)) != 40 {
		t.Fatalf("snapshot %s is not a git commit", current)
	}
	secondRoot := t.TempDir()
	if _, err = runtime.Init(context.Background(), secondRoot, initRequest(), false); err != nil {
		t.Fatal(err)
	}
	again, err := os.Readlink(filepath.Join(dataRoot, "config", "current"))
	if err != nil || again != current {
		t.Fatalf("same commit snapshot = %s, want %s (%v)", again, current, err)
	}
	git(t, source, "commit", "--allow-empty", "-m", "advance")
	thirdRoot := t.TempDir()
	if _, err = runtime.Init(context.Background(), thirdRoot, initRequest(), false); err != nil {
		t.Fatal(err)
	}
	advanced, err := os.Readlink(filepath.Join(dataRoot, "config", "current"))
	if err != nil || advanced == current {
		t.Fatalf("fresh init did not pull the new commit: %s", advanced)
	}
	pin, err := os.ReadFile(filepath.Join(firstRoot, ".fanloop", "configuration-snapshot"))
	if err != nil || filepath.Base(strings.TrimSpace(string(pin))) != filepath.Base(current) || filepath.Base(current) == filepath.Base(advanced) {
		t.Fatalf("first pin = %q, original current %s, advanced %s (%v)", pin, current, advanced, err)
	}
	local, failure := store.New(firstRoot)
	if failure != nil {
		t.Fatal(failure)
	}
	_, loaded, failure := local.LoadBound()
	if failure != nil || loaded.Ref.ID != "linear-runtime" {
		t.Fatalf("pinned requirement loaded %#v %v", loaded.Ref, failure)
	}
}

func TestFreshInitKeepsSnapshotWhenGitPullFails(t *testing.T) {
	source := gitWorkflowFixture(t)
	dataRoot := t.TempDir()
	useGitRemote(t, source)
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	root, unlock, err := prepareConfiguration(context.Background(), dataRoot, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	unlock()
	installed, err := os.Readlink(filepath.Join(dataRoot, "config", "current"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_KEY_0", "url."+filepath.Join(t.TempDir(), "missing")+".insteadOf")
	warnings := []string{}
	fallback, unlock, err := prepareConfiguration(context.Background(), dataRoot, false, func(message string) {
		warnings = append(warnings, message)
	})
	if err != nil {
		t.Fatal(err)
	}
	unlock()
	if fallback != root {
		t.Fatalf("fallback root = %s, installed = %s", fallback, root)
	}
	if len(warnings) != 1 || warnings[0] != "automatic configuration pull failed; continuing with the installed snapshot" {
		t.Fatalf("warnings = %#v", warnings)
	}
	after, err := os.Readlink(filepath.Join(dataRoot, "config", "current"))
	if err != nil || after != installed {
		t.Fatalf("current moved from %s to %s (%v)", installed, after, err)
	}
	warnings = nil
	t.Setenv(envNoAutoConfig, "1")
	skipped, unlock, err := prepareConfiguration(context.Background(), dataRoot, false, func(message string) {
		warnings = append(warnings, message)
	})
	if err != nil {
		t.Fatal(err)
	}
	unlock()
	if skipped != root || len(warnings) != 0 {
		t.Fatalf("opt-out root=%s warnings=%v", skipped, warnings)
	}
}

func TestFreshInitWithoutSnapshotStillReportsFetchFailure(t *testing.T) {
	dataRoot := t.TempDir()
	useGitRemote(t, filepath.Join(t.TempDir(), "missing"))
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	_, _, err := prepareConfiguration(context.Background(), dataRoot, false, nil)
	if !errors.Is(err, configuration.ErrFetch) {
		t.Fatalf("err = %v", err)
	}
	if _, statErr := os.Lstat(filepath.Join(dataRoot, "config", "current")); !os.IsNotExist(statErr) {
		t.Fatal("failed fetch activated a snapshot")
	}
}

func TestFornaxSnapshotIsOptIn(t *testing.T) {
	dataRoot := t.TempDir()
	writeFornaxFile(t, dataRoot)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv(envUseFornax, "1")
	if err := os.Unsetenv("FANLOOP_CONFIG_SOURCE"); err != nil {
		t.Fatal(err)
	}
	_, _, err := prepareConfiguration(context.Background(), dataRoot, false, nil)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "fornax") {
		t.Fatalf("opt-in err = %v", err)
	}
}

func gitWorkflowFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.CopyFS(filepath.Join(root, "fanloop-workflow", "linear-runtime"), os.DirFS(filepath.Join("..", "workflow", "testdata", "linear-runtime"))); err != nil {
		t.Fatal(err)
	}
	git(t, root, "init", "-b", "main")
	git(t, root, "config", "user.name", "Configuration Test")
	git(t, root, "config", "user.email", "config@example.test")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "fixture")
	return root
}

func useGitRemote(t *testing.T, source string) {
	t.Helper()
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "url."+source+".insteadOf")
	t.Setenv("GIT_CONFIG_VALUE_0", configuration.Repository)
	t.Setenv("FANLOOP_CONFIG_SOURCE", "unset")
	if err := os.Unsetenv("FANLOOP_CONFIG_SOURCE"); err != nil {
		t.Fatal(err)
	}
}

func writeFornaxFile(t *testing.T, dataRoot string) {
	t.Helper()
	content := []byte(`{"sources":[{"workspace_id":"1","key_prefix":"fanloop.workflow.","access":"read_write"}]}` + "\n")
	if err := os.WriteFile(filepath.Join(dataRoot, "fornax.json"), content, 0o600); err != nil {
		t.Fatal(err)
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %s %v", args, output, err)
	}
}
