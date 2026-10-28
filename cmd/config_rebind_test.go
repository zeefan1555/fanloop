package cmd

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/configuration"
	"github.com/zeefan1555/fanloop/internal/controller"
)

func TestBoundPublicRebindPreservesCallerTargetsAndActualPin(t *testing.T) {
	dataRoot, oldSnapshot, newSnapshot := configRebindSnapshots(t)
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_NO_AUTO_CONFIG", "1")
	t.Setenv("FANLOOP_NO_AUTO_UPDATE", "1")
	configUnsetSource(t)
	root := filepath.Join(t.TempDir(), "root with space and 'quote'")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(newFlowRuntime().ControllerRelease, "bin", "fanloop")
	launcher, err := filepath.Abs(filepath.Join("..", "packaging", "npm", "scripts", "run.js"))
	if err != nil {
		t.Fatal(err)
	}
	cwd := t.TempDir()
	run := func(surface string, args []string, want int) string {
		t.Helper()
		command := exec.Command(binary, args...)
		if surface == "node" {
			command = exec.Command("node", append([]string{launcher}, args...)...)
		}
		command.Dir = cwd
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		err := command.Run()
		code := 0
		if err != nil {
			exit, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatal(err)
			}
			code = exit.ExitCode()
		}
		if code != want || want != 0 && (stdout.Len() != 0 || stderr.Len() == 0) {
			t.Fatalf("%s %v exit %d want %d\nstdout=%s\nstderr=%s", surface, args, code, want, stdout.String(), stderr.String())
		}
		return stdout.String() + stderr.String()
	}
	run("direct", []string{"flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Rebind process"}, 0)
	journal, err := os.ReadFile(filepath.Join(root, controller.Journal))
	if err != nil {
		t.Fatal(err)
	}
	business := map[string]string{}
	for _, file := range []string{"flow/state.json", "output/state.json", "trace/events.jsonl"} {
		content, err := os.ReadFile(filepath.Join(root, ".fanloop", file))
		if err != nil {
			t.Fatal(err)
		}
		business[file] = string(content)
	}
	if err := configuration.Activate(dataRoot, newSnapshot); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FANLOOP_CONFIG_SOURCE", filepath.Join(dataRoot, "ignored-bad-override"))
	for _, surface := range []string{"direct", "node"} {
		selection, err := ControllerSelection([]string{"config", "rebind", "--root", root, "--reason", "caller default"})
		if err != nil || selection == nil || !strings.Contains("\n"+strings.Join(selection.Environment, "\n")+"\n", "\nFANLOOP_DATA_HOME="+dataRoot+"\n") {
			t.Fatalf("caller home was not retained: selection_present=%t error=%v", selection != nil, err)
		}
		result := run(surface, []string{"config", "rebind", "--root=" + root, "--reason", "caller default"}, 0)
		if !strings.Contains(result, filepath.Base(newSnapshot)) {
			t.Fatalf("caller default was not selected: %s", result)
		}
		result = run(surface, []string{"config", "rebind", "--root", root, "--snapshot", filepath.Base(oldSnapshot), "--reason", "caller named"}, 0)
		if !strings.Contains(result, filepath.Base(oldSnapshot)) {
			t.Fatalf("caller named was not selected: %s", result)
		}
	}
	run("node", []string{"config", "rebind", "--root", root, "--reason", "final caller default"}, 0)
	current := filepath.Join(root, controller.Home, "config", "current")
	if err := os.Remove(current); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "ignored-bad-history"), current); err != nil {
		t.Fatal(err)
	}
	for _, surface := range []string{"direct", "node"} {
		result := run(surface, []string{"flow", "status", "--root=" + root}, 0)
		if !strings.Contains(result, newSnapshot) {
			t.Fatalf("actual rebind pin was lost: %s", result)
		}
	}
	for _, file := range []string{controller.Journal, ".fanloop/configuration-snapshot"} {
		content, err := os.ReadFile(filepath.Join(root, file))
		if err != nil || file == controller.Journal && !bytes.Equal(content, journal) || file != controller.Journal && string(content) != newSnapshot+"\n" {
			t.Fatalf("ordinary command changed pin or historical binding: %s %v", file, err)
		}
	}
	for file, original := range business {
		content, err := os.ReadFile(filepath.Join(root, ".fanloop", file))
		if err != nil || string(content) != original {
			t.Fatalf("rebind altered accepted business bytes %s: %v", file, err)
		}
	}
	facts := map[string][]byte{}
	for _, file := range []string{controller.Journal, ".fanloop/configuration-snapshot", ".fanloop/audit/configuration-rebindings.jsonl", ".fanloop/flow/state.json", ".fanloop/output/state.json", ".fanloop/trace/events.jsonl"} {
		content, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			t.Fatal(err)
		}
		facts[file] = content
	}
	for _, target := range []struct {
		name     string
		snapshot string
		exit     int
	}{
		{name: "linked-home", exit: 5},
		{name: "linked-config", exit: 5},
		{name: "linked-snapshots", exit: 5},
		{name: "escaped-current", exit: 2},
		{name: "linked-default", exit: 2},
		{name: "linked-named", snapshot: filepath.Base(oldSnapshot), exit: 2},
		{name: "relative-named", snapshot: "../" + filepath.Base(oldSnapshot), exit: 2},
		{name: "absolute-named", snapshot: oldSnapshot, exit: 2},
		{name: "ordinary-current", exit: 2},
	} {
		t.Run(target.name, func(t *testing.T) {
			caller := filepath.Join(t.TempDir(), "caller")
			outside := t.TempDir()
			link := caller
			switch target.name {
			case "linked-home", "linked-config", "linked-snapshots":
				if target.name == "linked-config" {
					if err := os.Mkdir(caller, 0700); err != nil {
						t.Fatal(err)
					}
					link = filepath.Join(caller, "config")
				} else if target.name == "linked-snapshots" {
					if err := os.MkdirAll(filepath.Join(caller, "config"), 0700); err != nil {
						t.Fatal(err)
					}
					link = filepath.Join(caller, "config", "snapshots")
				}
				if err := os.Symlink(outside, link); err != nil {
					t.Fatal(err)
				}
			default:
				snapshots := filepath.Join(caller, "config", "snapshots")
				if err := os.MkdirAll(snapshots, 0700); err != nil {
					t.Fatal(err)
				}
				current := filepath.Join(caller, "config", "current")
				switch target.name {
				case "linked-default", "linked-named":
					leaf := filepath.Join(snapshots, filepath.Base(oldSnapshot))
					if err := os.Symlink(oldSnapshot, leaf); err != nil {
						t.Fatal(err)
					}
					err = os.Symlink(leaf, current)
				case "ordinary-current":
					err = os.WriteFile(current, []byte(oldSnapshot+"\n"), 0600)
				default:
					err = os.Symlink(oldSnapshot, current)
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("FANLOOP_DATA_HOME", caller)
			args := []string{"config", "rebind", "--root", root, "--reason", "unsafe caller"}
			if target.snapshot != "" {
				args = append(args, "--snapshot", target.snapshot)
			}
			for _, surface := range []string{"direct", "node"} {
				result := run(surface, args, target.exit)
				if target.exit == 2 && !strings.Contains(result, `"code": "INVALID_ARGUMENT"`) {
					t.Fatalf("%s invalid target returned the wrong error: %s", surface, result)
				}
				for file, original := range facts {
					content, err := os.ReadFile(filepath.Join(root, file))
					if err != nil || !bytes.Equal(content, original) {
						t.Fatalf("%s unsafe target altered %s: %v", surface, file, err)
					}
				}
			}
		})
	}
}

func TestConfigRebindPublicCLIChangesPinnedSnapshotAndRecordsAudit(t *testing.T) {
	dataRoot, oldSnapshot, newSnapshot := configRebindSnapshots(t)
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_NO_AUTO_CONFIG", "1")
	configUnsetSource(t)

	root := t.TempDir()
	configRebindExecute(t, []string{"flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Rebind fixture"}, 0)
	initialPin := filepath.Join(root, "bound-release-home", "config", "snapshots", filepath.Base(oldSnapshot))
	before := configRebindExecute(t, []string{"flow", "status", "--root", root}, 0)
	if !strings.Contains(before, initialPin) || strings.Contains(before, newSnapshot) {
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
	if !strings.Contains(stillPinned, initialPin) || strings.Contains(stillPinned, newSnapshot) {
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
