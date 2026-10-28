package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/controller"
	"github.com/zeefan1555/fanloop/internal/idl/storageidl"
)

func TestControllerPinnerUsesExplicitPhysicalManagerAndRelease(t *testing.T) {
	directory := t.TempDir()
	root := filepath.Join(directory, "root with 'quotes'")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(directory, "release with 'quotes'")
	configCopyTree(t, newFlowRuntime().ControllerRelease, source)
	binary := filepath.Join(source, "bin", "fanloop")
	t.Setenv("FANLOOP_NO_AUTO_UPDATE", "1")
	t.Setenv("FANLOOP_NO_AUTO_CONFIG", "1")
	command := exec.Command(binary, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Pinner")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("real init: %v\n%s", err, output)
	}
	for _, relative := range []string{controller.Journal, controller.Home} {
		if err := os.Rename(filepath.Join(root, relative), filepath.Join(directory, filepath.Base(relative)+".preserved")); err != nil {
			t.Fatal(err)
		}
	}
	before := map[string][]byte{}
	for _, relative := range []string{".fanloop/flow/state.json", ".fanloop/output/state.json", ".fanloop/trace/events.jsonl", ".fanloop/configuration-snapshot"} {
		before[relative] = configRead(t, filepath.Join(root, relative))
	}
	managerLink := filepath.Join(directory, "manager-link")
	releaseLink := filepath.Join(directory, "release-link")
	if err := os.Symlink(binary, managerLink); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(source, releaseLink); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(directory, "home with 'quotes'")
	roots := controller.SkillRoots(home)
	command = exec.Command(binary, "__install", "--source", source, "--data-root", home,
		"--codex-skills-root", roots.Codex, "--agent-skills-root", roots.Agent, "--trae-skills-root", roots.Trae, "--claude-skills-root", roots.Claude)
	command.Env = controller.Environment(home, os.Getenv("FANLOOP_CONFIG_SOURCE"), "")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("real original install: %v\n%s", err, output)
	}
	source = filepath.Join(home, "releases", "test")
	for index, name := range []string{"FANLOOP_CODEX_SKILLS_ROOT", "FANLOOP_AGENT_SKILLS_ROOT", "FANLOOP_TRAE_SKILLS_ROOT", "FANLOOP_CLAUDE_SKILLS_ROOT"} {
		t.Setenv(name, roots.Values()[index])
	}
	currentBin := filepath.Join(home, "current", "bin")
	if err := os.Rename(currentBin, currentBin+".preserved"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(currentBin, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/usr/bin/false", filepath.Join(currentBin, "fanloop")); err != nil {
		t.Fatal(err)
	}
	unsupportedManager := filepath.Join(directory, "unsupported-manager")
	if err := os.WriteFile(unsupportedManager, []byte("#!/bin/sh\nprintf 'Usage: fanloop [flags]\\n'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	script, err := filepath.Abs("../scripts/pin-controller-release.sh")
	if err != nil {
		t.Fatal(err)
	}
	run := func(manager, release string, explicit bool) (string, string, error) {
		t.Helper()
		command := exec.Command("bash", script, root)
		command.Dir = directory
		for _, item := range os.Environ() {
			name, _, _ := strings.Cut(item, "=")
			if name != "FANLOOP_PROTECT_BINARY" && name != "FANLOOP_CONTROLLER_RELEASE" && name != "FANLOOP_DATA_HOME" && name != "FANLOOP_CONFIG_SOURCE" {
				command.Env = append(command.Env, item)
			}
		}
		command.Env = append(command.Env, "FANLOOP_DATA_HOME="+home, "FANLOOP_CONFIG_SOURCE="+filepath.Join(directory, "ignored-override"), "FANLOOP_CONTROLLER_RELEASE="+release)
		if explicit {
			command.Env = append(command.Env, "FANLOOP_PROTECT_BINARY="+manager)
		}
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		err := command.Run()
		return stdout.String(), stderr.String(), err
	}
	for _, test := range []struct {
		name, manager, release string
		explicit               bool
		code                   int
	}{
		{"unsupported-default", "", source, false, 1},
		{"generic-help-is-not-protect", unsupportedManager, source, true, 1},
		{"empty-manager", "", source, true, 1},
		{"relative-manager", "bin/fanloop", source, true, 1},
		{"linked-manager", managerLink, source, true, 1},
		{"linked-manager-ancestor", filepath.Join(releaseLink, "bin", "fanloop"), source, true, 1},
		{"empty-source", binary, "", true, 1},
		{"relative-source", binary, "release", true, 1},
		{"linked-source", binary, releaseLink, true, 5},
		{"missing-source", binary, filepath.Join(directory, "missing-release"), true, 5},
	} {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, failure := run(test.manager, test.release, test.explicit)
			exit, ok := failure.(*exec.ExitError)
			if !ok || exit.ExitCode() != test.code || stdout != "" || stderr == "" {
				t.Fatalf("unsafe pinner succeeded: %v\n%s\n%s", failure, stdout, stderr)
			}
			for relative, expected := range before {
				if !bytes.Equal(configRead(t, filepath.Join(root, relative)), expected) {
					t.Fatalf("failed pinner changed %s", relative)
				}
			}
			if _, err := os.Lstat(filepath.Join(root, controller.Journal)); !os.IsNotExist(err) {
				t.Fatalf("failed pinner created journal: %v", err)
			}
			if _, err := os.Lstat(filepath.Join(root, controller.Home)); !os.IsNotExist(err) {
				t.Fatalf("failed pinner created controller: %v", err)
			}
		})
	}
	stdout, stderr, err := run(binary, source, true)
	want := filepath.Join(root, controller.Home, "releases", "test", "bin", "fanloop") + "\n"
	if err != nil || stdout != want {
		t.Fatalf("explicit manager/source protect: %v\n%s\n%s", err, stdout, stderr)
	}
	binding, err := controller.Read(root)
	if err != nil || binding.SourceReleasePath != source || binding.Phase != storageidl.ControllerBindingPhase_committed {
		t.Fatalf("pinner did not preserve explicit physical source: %v %v", binding, err)
	}
	for relative, expected := range before {
		if relative != ".fanloop/configuration-snapshot" && !bytes.Equal(configRead(t, filepath.Join(root, relative)), expected) {
			t.Fatalf("protect changed accepted %s", relative)
		}
	}
	stdoutAgain, stderrAgain, err := run(binary, filepath.Join(directory, "ignored-new-source"), true)
	if err != nil || stdoutAgain != stdout {
		t.Fatalf("committed pinner was not idempotent: %v\n%s\n%s", err, stdoutAgain, stderrAgain)
	}
}
