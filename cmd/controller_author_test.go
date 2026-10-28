package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/configuration"
	"github.com/zeefan1555/fanloop/internal/controller"
	"github.com/zeefan1555/fanloop/internal/idl/storageidl"
	"github.com/zeefan1555/fanloop/internal/release"
)

func TestAuthorLivePublicEditingAndExplicitFreeze(t *testing.T) {
	dataRoot := t.TempDir()
	source := filepath.Join(t.TempDir(), "author")
	configCopyTree(t, "../internal/workflow/testdata/linear-runtime", filepath.Join(source, "fanloop-workflow", "linear-runtime"))
	configGit(t, source, "init", "--quiet")
	configGit(t, source, "config", "user.name", "Author Test")
	configGit(t, source, "config", "user.email", "author@example.test")
	configGit(t, source, "add", ".")
	configGit(t, source, "commit", "--quiet", "-m", "author baseline")
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_CONFIG_SOURCE", source)
	t.Setenv("FANLOOP_NO_AUTO_UPDATE", "1")
	t.Setenv("FANLOOP_NO_AUTO_CONFIG", "1")
	binary := filepath.Join(newFlowRuntime().ControllerRelease, "bin", "fanloop")
	launcher, err := filepath.Abs("../packaging/npm/scripts/run.js")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	run := func(surface string, args ...string) (string, string) {
		t.Helper()
		command := exec.Command(binary, args...)
		if surface == "node" {
			command = exec.Command("node", append([]string{launcher}, args...)...)
		}
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		if err := command.Run(); err != nil {
			t.Fatalf("%s %v: %v\n%s\n%s", surface, args, err, stdout.String(), stderr.String())
		}
		return stdout.String(), stderr.String()
	}
	_, warning := run("direct", "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Editable author")
	if strings.Count(warning, "author/live") != 1 {
		t.Fatalf("author init warning count: %q", warning)
	}
	live, err := controller.Read(root)
	if err != nil || live.ConfigurationMode != storageidl.ControllerConfigurationMode_author_live || live.ConfigurationPin != source {
		t.Fatalf("author init was not live: %v %v", live, err)
	}
	journal, err := os.ReadFile(filepath.Join(root, controller.Journal))
	if err != nil {
		t.Fatal(err)
	}
	physical := filepath.Join(root, controller.Home, live.ControllerReleasePath)
	releaseDigest, modes, err := release.StrictDirectoryIdentity(physical)
	if err != nil {
		t.Fatal(err)
	}
	prompt := filepath.Join(source, "fanloop-workflow", "linear-runtime", "prompt.yaml")
	content, err := os.ReadFile(prompt)
	if err != nil {
		t.Fatal(err)
	}
	content = bytes.Replace(content, []byte("Advance after the current Step completes."), []byte("Fresh author prompt."), 1)
	content = bytes.Replace(content, []byte("    skills: []"), []byte("    skills:\n      - id: test-guide\n        prompt: Fresh author skill guidance.\n        optional: false"), 1)
	if err := os.WriteFile(prompt, content, 0644); err != nil {
		t.Fatal(err)
	}
	guide := filepath.Join(source, "fanloop-workflow", "linear-runtime", "skills", "test-guide", "SKILL.md")
	if err := os.WriteFile(guide, []byte("---\ndescription: Fresh author skill content\n---\n# Edited skill\n"), 0644); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(filepath.Dir(guide), "dirty-check.sh")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nprintf 'dirty resource\\n'\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if output := configGit(t, source, "status", "--porcelain"); output == "" {
		t.Fatal("author freeze fixture is not dirty")
	}
	t.Setenv("FANLOOP_CONFIG_SOURCE", filepath.Join(dataRoot, "ignored-override"))
	for _, surface := range []string{"direct", "node"} {
		output, warning := run(surface, "flow", "status", "--root", root)
		for _, expected := range []string{"Fresh author prompt", "Fresh author skill guidance", guide} {
			if !strings.Contains(output, expected) {
				t.Fatalf("live edit not visible in %s Status: %s", surface, output)
			}
		}
		if strings.Count(warning, "author/live") != 1 {
			t.Fatalf("%s status repeated live warning: %q", surface, warning)
		}
		catalog, _ := run(surface, "flow", "skills", "--root", root)
		if !strings.Contains(catalog, "Fresh author skill content") {
			t.Fatalf("live Skill content was not reloaded: %s", catalog)
		}
	}
	before, err := configuration.InspectBinding(source)
	if err != nil {
		t.Fatal(err)
	}
	protected, _ := run("direct", "__protect", root, "--source", newFlowRuntime().ControllerRelease)
	frozen, err := controller.Read(root)
	if err != nil || frozen.ConfigurationMode != storageidl.ControllerConfigurationMode_author_frozen || frozen.TransactionId == live.TransactionId || frozen.GetPreviousBindingTransactionId() != live.TransactionId || frozen.ConfigurationSourcePath != source || frozen.PreviousConfigurationPin != source {
		t.Fatalf("author freeze did not preserve its previous transaction: %v %v", frozen, err)
	}
	if strings.TrimSpace(protected) != filepath.Join(physical, "bin", "fanloop") {
		t.Fatalf("freeze did not reuse exact Release: %s", protected)
	}
	archive, err := os.ReadFile(filepath.Join(root, ".fanloop", "audit", "controller-bindings", live.TransactionId+".json"))
	if err != nil || !bytes.Equal(archive, journal) {
		t.Fatalf("old journal was not archived byte-for-byte: %v", err)
	}
	captured, err := configuration.InspectBinding(frozen.ConfigurationPin)
	if err != nil || !reflect.DeepEqual(before, captured) {
		t.Fatalf("dirty configuration capture differs: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(frozen.ConfigurationPin, "configuration.json")); !os.IsNotExist(err) {
		t.Fatalf("author capture fabricated a managed manifest: %v", err)
	}
	afterDigest, afterModes, err := release.StrictDirectoryIdentity(physical)
	if err != nil || afterDigest != releaseDigest || !reflect.DeepEqual(afterModes, modes) || live.ReleaseResourcesSha256 != release.ControllerResourcesDigest(releaseDigest, modes) || frozen.ReleaseResourcesSha256 != live.ReleaseResourcesSha256 {
		t.Fatalf("author freeze overwrote healthy Release: %v", err)
	}
	retired := source + ".retired"
	if err := os.Rename(source, retired); err != nil {
		t.Fatal(err)
	}
	for _, surface := range []string{"direct", "node"} {
		output, warning := run(surface, "flow", "status", "--root", root)
		if !strings.Contains(output, "Fresh author prompt") || strings.Contains(warning, "author/live") {
			t.Fatalf("frozen continuation depends on author or claims live: %s %s", output, warning)
		}
	}
	protectedAgain, _ := run("direct", "__protect", root, "--source", filepath.Join(dataRoot, "ignored-source"))
	if protectedAgain != protected {
		t.Fatalf("committed freeze was not idempotent: %s %s", protectedAgain, protected)
	}
}

func TestCommittedControllerRejectsResourceDrift(t *testing.T) {
	launcher, err := filepath.Abs("../packaging/npm/scripts/run.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, drift := range []string{"bytes", "file-mode", "add-file", "remove-file", "add-empty-directory", "remove-empty-directory"} {
		t.Run(drift, func(t *testing.T) {
			source := filepath.Join(t.TempDir(), "release")
			if err := os.CopyFS(source, os.DirFS(newFlowRuntime().ControllerRelease)); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(source, "resource.txt"), []byte("original resource"), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(filepath.Join(source, "empty"), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("FANLOOP_DATA_HOME", t.TempDir())
			t.Setenv("FANLOOP_NO_AUTO_UPDATE", "1")
			t.Setenv("FANLOOP_NO_AUTO_CONFIG", "1")
			binary := filepath.Join(source, "bin", "fanloop")
			root := t.TempDir()
			for _, args := range [][]string{
				{"flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Resource drift"},
				{"__protect", root, "--source", source},
			} {
				if output, err := exec.Command(binary, args...).CombinedOutput(); err != nil {
					t.Fatalf("real author initialize/freeze: %v\n%s", err, output)
				}
			}
			binding, err := controller.Read(root)
			if err != nil || binding.GetPreviousBindingTransactionId() == "" {
				t.Fatalf("missing real freeze history: %v %v", binding, err)
			}
			physical := filepath.Join(root, controller.Home, binding.ControllerReleasePath)
			if err := os.Rename(source, source+".retired"); err != nil {
				t.Fatal(err)
			}
			binary = filepath.Join(newFlowRuntime().ControllerRelease, "bin", "fanloop")
			args := []string{"flow", "status", "--root", root}
			for _, command := range []*exec.Cmd{exec.Command(binary, args...), exec.Command("node", append([]string{launcher}, args...)...)} {
				if output, err := command.CombinedOutput(); err != nil || !strings.Contains(string(output), "Resource drift") {
					t.Fatalf("healthy committed source retirement: %v\n%s", err, output)
				}
			}
			facts := map[string][]byte{}
			for _, file := range []string{".fanloop/flow/state.json", ".fanloop/output/state.json", ".fanloop/trace/events.jsonl", ".fanloop/configuration-snapshot", controller.Journal, ".fanloop/audit/controller-bindings/" + binding.GetPreviousBindingTransactionId() + ".json"} {
				content, err := os.ReadFile(filepath.Join(root, file))
				if err != nil {
					t.Fatal(err)
				}
				facts[file] = content
			}
			switch drift {
			case "bytes":
				err = os.WriteFile(filepath.Join(physical, "resource.txt"), []byte("external drift"), 0644)
			case "file-mode":
				err = os.Chmod(filepath.Join(physical, "resource.txt"), 0775)
			case "add-file":
				err = os.WriteFile(filepath.Join(physical, "new-resource.txt"), []byte("external drift"), 0600)
			case "remove-file":
				err = os.Remove(filepath.Join(physical, "resource.txt"))
			case "add-empty-directory":
				err = os.Mkdir(filepath.Join(physical, "new-empty"), 0700)
			case "remove-empty-directory":
				err = os.Remove(filepath.Join(physical, "empty"))
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := release.LoadBound(physical); err != nil {
				t.Fatalf("drift masked by old manifest validation: %v", err)
			}
			digest, modes, err := release.StrictDirectoryIdentity(physical)
			if err != nil {
				t.Fatal(err)
			}
			if selection, err := ControllerSelection(args); err == nil || selection != nil {
				t.Fatalf("Go committed selection accepted %s: selection_present=%t error=%v", drift, selection != nil, err)
			}
			for _, surface := range []string{"direct", "node"} {
				command := exec.Command(binary, args...)
				if surface == "node" {
					command = exec.Command("node", append([]string{launcher}, args...)...)
				}
				var stdout, stderr bytes.Buffer
				command.Stdout, command.Stderr = &stdout, &stderr
				failure := command.Run()
				if exit, ok := failure.(*exec.ExitError); !ok || exit.ExitCode() != 5 || stdout.Len() != 0 || stderr.Len() == 0 {
					t.Fatalf("%s accepted committed %s: %v\n%s\n%s", surface, drift, failure, stdout.String(), stderr.String())
				}
				for file, original := range facts {
					current, err := os.ReadFile(filepath.Join(root, file))
					if err != nil || !bytes.Equal(current, original) {
						t.Fatalf("%s rejection altered %s: %v", surface, file, err)
					}
				}
				currentDigest, currentModes, err := release.StrictDirectoryIdentity(physical)
				if err != nil || currentDigest != digest || !reflect.DeepEqual(currentModes, modes) {
					t.Fatalf("%s rejection overwrote Release drift: %v", surface, err)
				}
			}
		})
	}
}
