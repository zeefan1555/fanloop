package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/configuration"
)

func TestConfigRebindPublicCLIChangesPinnedSnapshotAndRecordsAudit(t *testing.T) {
	dataRoot, oldSnapshot, newSnapshot := configRebindSnapshots(t)
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_NO_AUTO_CONFIG", "1")
	configUnsetSource(t)

	root := t.TempDir()
	configRebindExecute(t, []string{"flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Rebind fixture"}, 0)
	before := configRebindExecute(t, []string{"flow", "status", "--root", root}, 0)
	if !strings.Contains(before, oldSnapshot) || strings.Contains(before, newSnapshot) {
		t.Fatalf("status before update does not use old snapshot: %s", before)
	}

	previousSync := applyGitConfigSync
	t.Cleanup(func() { applyGitConfigSync = previousSync })
	applyGitConfigSync = func(context.Context, string) (configuration.SyncResult, error) {
		if err := configuration.Activate(dataRoot, newSnapshot); err != nil {
			return configuration.SyncResult{}, err
		}
		manifest, err := configuration.Read(newSnapshot)
		return configuration.SyncResult{CommitSHA: manifest.CommitSHA, WorkflowIDs: manifest.WorkflowIDs}, err
	}
	configRebindExecute(t, []string{"update", "config"}, 0)
	stillPinned := configRebindExecute(t, []string{"flow", "status", "--root", root}, 0)
	if !strings.Contains(stillPinned, oldSnapshot) || strings.Contains(stillPinned, newSnapshot) {
		t.Fatalf("update config changed an existing requirement pin: %s", stillPinned)
	}

	result := configRebindExecute(t, []string{"config", "rebind", "--root", root, "--reason", "AGE-119 refresh broken run-doc hashing"}, 0)
	for _, want := range []string{filepath.Base(oldSnapshot), filepath.Base(newSnapshot), "AGE-119 refresh broken run-doc hashing", ".fanloop/audit/configuration-rebindings.jsonl"} {
		if !strings.Contains(result, want) {
			t.Fatalf("rebind result missing %q: %s", want, result)
		}
	}
	after := configRebindExecute(t, []string{"flow", "status", "--root", root}, 0)
	if !strings.Contains(after, newSnapshot) || strings.Contains(after, oldSnapshot) {
		t.Fatalf("status after rebind does not use new snapshot: %s", after)
	}
	explicit := configRebindExecute(t, []string{"config", "rebind", "--root", root, "--snapshot", filepath.Base(oldSnapshot), "--reason", "explicit installed snapshot"}, 0)
	if !strings.Contains(explicit, filepath.Base(newSnapshot)) || !strings.Contains(explicit, filepath.Base(oldSnapshot)) {
		t.Fatalf("explicit rebind returned the wrong identities: %s", explicit)
	}
	explicitStatus := configRebindExecute(t, []string{"flow", "status", "--root", root}, 0)
	if !strings.Contains(explicitStatus, oldSnapshot) || strings.Contains(explicitStatus, newSnapshot) {
		t.Fatalf("status after explicit rebind does not use requested snapshot: %s", explicitStatus)
	}
	audit, err := os.ReadFile(filepath.Join(root, ".fanloop", "audit", "configuration-rebindings.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{filepath.Base(oldSnapshot), filepath.Base(newSnapshot), "AGE-119 refresh broken run-doc hashing", "explicit installed snapshot", `"rebound_at"`, `"phase":"prepared"`, `"phase":"committed"`} {
		if !bytes.Contains(audit, []byte(want)) {
			t.Fatalf("audit missing %q: %s", want, audit)
		}
	}
}

func TestConfigRebindPublicCLIRejectsUninitializedRequirement(t *testing.T) {
	dataRoot, _, _ := configRebindSnapshots(t)
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	configUnsetSource(t)
	root := t.TempDir()
	result := configRebindExecute(t, []string{"config", "rebind", "--root", root, "--reason", "negative test"}, 1)
	if !strings.Contains(result, `"code": "NOT_INITIALIZED"`) {
		t.Fatalf("uninitialized rebind returned the wrong error: %s", result)
	}
	if _, err := os.Stat(filepath.Join(root, ".fanloop", "configuration-snapshot")); !os.IsNotExist(err) {
		t.Fatalf("uninitialized rebind mutated requirement: %v", err)
	}
}

func TestConfigRebindPublicCLIRollsBackPreparedAuditWhenPinWriteFails(t *testing.T) {
	dataRoot, _, newSnapshot := configRebindSnapshots(t)
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_NO_AUTO_CONFIG", "1")
	configUnsetSource(t)
	root := t.TempDir()
	configRebindExecute(t, []string{"flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Rebind rollback fixture"}, 0)
	if err := configuration.Activate(dataRoot, newSnapshot); err != nil {
		t.Fatal(err)
	}
	metadata := filepath.Join(root, ".fanloop")
	if err := os.MkdirAll(filepath.Join(metadata, "audit"), 0o700); err != nil {
		t.Fatal(err)
	}
	beforePin, err := os.ReadFile(filepath.Join(metadata, "configuration-snapshot"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(metadata, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(metadata, 0o700) })
	result := configRebindExecute(t, []string{"config", "rebind", "--root", root, "--reason", "force pin write failure"}, 5)
	if !strings.Contains(result, `"code": "LOCAL_COMMIT_FAILED"`) {
		t.Fatalf("pin write failure returned the wrong error: %s", result)
	}
	pin, err := os.ReadFile(filepath.Join(metadata, "configuration-snapshot"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(beforePin, pin) {
		t.Fatalf("failed transaction changed pin: before=%q after=%q", beforePin, pin)
	}
	if _, err = os.Stat(filepath.Join(metadata, "audit", "configuration-rebindings.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("failed transaction retained prepared audit: %v", err)
	}
}

func TestConfigRebindPublicCLIRejectsInvalidTargetWithoutMutation(t *testing.T) {
	dataRoot, _, _ := configRebindSnapshots(t)
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_NO_AUTO_CONFIG", "1")
	configUnsetSource(t)
	root := t.TempDir()
	configRebindExecute(t, []string{"flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Rebind invalid fixture"}, 0)
	pinPath := filepath.Join(root, ".fanloop", "configuration-snapshot")
	before, err := os.ReadFile(pinPath)
	if err != nil {
		t.Fatal(err)
	}

	configRebindExecute(t, []string{"config", "rebind", "--root", root, "--snapshot", "missing", "--reason", "negative test"}, 2)
	after, err := os.ReadFile(pinPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("failed rebind mutated pin: before=%q after=%q", before, after)
	}
	if _, err = os.Stat(filepath.Join(root, ".fanloop", "audit", "configuration-rebindings.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("failed rebind created audit: %v", err)
	}
}

func configRebindSnapshots(t *testing.T) (string, string, string) {
	t.Helper()
	source := t.TempDir()
	configCopyTree(t, filepath.Join("..", "internal", "workflow", "testdata", "linear-runtime"), filepath.Join(source, "fanloop-workflow", "linear-runtime"))
	promptPath := filepath.Join(source, "fanloop-workflow", "linear-runtime", "prompt.yaml")
	prompt, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatal(err)
	}
	prompt = bytes.Replace(prompt, []byte("    skills: []"), []byte("    skills:\n      - id: test-guide\n        prompt: Read the test guide.\n        optional: false"), 1)
	if err = os.WriteFile(promptPath, prompt, 0o644); err != nil {
		t.Fatal(err)
	}
	configGit(t, source, "init", "--quiet")
	configGit(t, source, "config", "user.name", "Config Rebind Test")
	configGit(t, source, "config", "user.email", "config-rebind@example.test")
	configGit(t, source, "add", ".")
	configGit(t, source, "commit", "--quiet", "-m", "old snapshot")
	dataRoot := t.TempDir()
	oldSnapshot, err := configuration.Import(context.Background(), dataRoot, source)
	if err != nil {
		t.Fatal(err)
	}
	if err = configuration.Activate(dataRoot, oldSnapshot); err != nil {
		t.Fatal(err)
	}
	prompt, err = os.ReadFile(promptPath)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(promptPath, append(prompt, []byte("\n# new snapshot\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	configGit(t, source, "add", ".")
	configGit(t, source, "commit", "--quiet", "-m", "new snapshot")
	newSnapshot, err := configuration.Import(context.Background(), dataRoot, source)
	if err != nil {
		t.Fatal(err)
	}
	return dataRoot, oldSnapshot, newSnapshot
}

func configRebindExecute(t *testing.T, args []string, wantCode int) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Execute(context.Background(), args, strings.NewReader(""), &stdout, &stderr)
	if code != wantCode {
		t.Fatalf("%v exit=%d want=%d stdout=%s stderr=%s", args, code, wantCode, stdout.String(), stderr.String())
	}
	return stdout.String() + stderr.String()
}
