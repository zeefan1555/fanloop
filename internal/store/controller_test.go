package store

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/controller"
	"github.com/zeefan1555/fanloop/internal/idl/storageidl"
	"github.com/zeefan1555/fanloop/internal/testfixture"
)

func controllerSource(t *testing.T, source, root string) string {
	t.Helper()
	home := os.Getenv("FANLOOP_DATA_HOME")
	configuration := os.Getenv("FANLOOP_CONFIG_SOURCE")
	roots := controller.SkillRoots(home)
	command := exec.Command(filepath.Join(source, "bin", "fanloop"), "__install", "--source", source, "--data-root", home,
		"--codex-skills-root", roots.Codex, "--agent-skills-root", roots.Agent, "--trae-skills-root", roots.Trae, "--claude-skills-root", roots.Claude)
	command.Env = controller.Environment(home, configuration, "")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("real companion install: %v\n%s", err, output)
	}
	for index, name := range []string{"FANLOOP_CODEX_SKILLS_ROOT", "FANLOOP_AGENT_SKILLS_ROOT", "FANLOOP_TRAE_SKILLS_ROOT", "FANLOOP_CLAUDE_SKILLS_ROOT"} {
		t.Setenv(name, roots.Values()[index])
	}
	physical := filepath.Join(home, "releases", "test")
	command = exec.Command(filepath.Join(physical, "bin", "fanloop"), "flow", "report", "progress", "--root", root, "--step-id", "first", "--status", "in_progress", "--summary", "prepare healthy test projections")
	command.Env = controller.Environment(home, configuration, "")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("real companion projection: %v\n%s", err, output)
	}
	return physical
}

func TestProtectControllerFiniteRecoveryPreservesBusinessFiles(t *testing.T) {
	source := filepath.Join(t.TempDir(), "release")
	if err := testfixture.BuildRelease(source, "test"); err != nil {
		t.Fatal(err)
	}
	for _, phase := range []string{"before-prepared", "copy", "before-publish", "published", "pinned"} {
		for _, pinned := range []bool{false, true} {
			t.Run(phase+"-pin-"+map[bool]string{true: "present", false: "missing"}[pinned], func(t *testing.T) {
				root := t.TempDir()
				local, _ := committedWorkflow(t, root)
				physical := controllerSource(t, source, root)
				previous := ""
				if pinned {
					previous = os.Getenv("FANLOOP_CONFIG_SOURCE")
					if err := local.PinConfiguration(previous); err != nil {
						t.Fatal(err)
					}
				}
				before, err := local.businessDigest()
				if err != nil {
					t.Fatal(err)
				}
				injected := false
				binary, err := local.ProtectController(context.Background(), physical, func(current string) error {
					if current == phase {
						injected = true
						return errors.New("injected " + phase)
					}
					return nil
				})
				if !injected || err == nil || binary != "" {
					t.Fatalf("phase %s was not a safe failure: %q %v", phase, binary, err)
				}
				after, err := local.businessDigest()
				if err != nil || before != after {
					t.Fatalf("business bytes changed: %s %s %v", before, after, err)
				}
				pin, found, err := controller.ReadPin(root)
				if err != nil || pin != previous || found != pinned {
					t.Fatalf("precise pin rollback: %q %t %v", pin, found, err)
				}
				binding, err := controller.Read(root)
				if err != nil {
					t.Fatal(err)
				}
				transaction := ""
				if phase == "before-prepared" {
					if binding != nil {
						t.Fatal("journal appeared before prepared")
					}
				} else {
					if binding == nil || binding.Phase != storageidl.ControllerBindingPhase_prepared {
						t.Fatalf("missing prepared intent: %v", binding)
					}
					transaction = binding.TransactionId
					if _, err := controller.Select(root, ""); err == nil || !strings.Contains(err.Error(), "prepared") {
						t.Fatalf("ordinary entry did not block prepared: %v", err)
					}
				}
				retrySource := physical
				if transaction != "" {
					retrySource = filepath.Join(root, "ignored-new-source")
					t.Setenv("FANLOOP_CONFIG_SOURCE", filepath.Join(root, "ignored-new-override"))
				}
				binary, err = local.ProtectController(context.Background(), retrySource, nil)
				if err != nil || binary == "" {
					t.Fatalf("finite retry failed: %s %v", binary, err)
				}
				committed, err := controller.Read(root)
				if err != nil || committed.Phase != storageidl.ControllerBindingPhase_committed || transaction != "" && committed.TransactionId != transaction {
					t.Fatalf("retry changed intent: %v %v", committed, err)
				}
				after, err = local.businessDigest()
				if err != nil || after != before {
					t.Fatalf("retry changed business bytes: %v", err)
				}
			})
		}
	}
}

func TestProtectControllerPreparedRejectsDriftWithoutOverwriting(t *testing.T) {
	source := filepath.Join(t.TempDir(), "release")
	if err := testfixture.BuildRelease(source, "test"); err != nil {
		t.Fatal(err)
	}
	for _, drift := range []string{"business", "source", "unknown-staging", "release-mode", "release-resource", "release-collection"} {
		t.Run(drift, func(t *testing.T) {
			root := t.TempDir()
			local, _ := committedWorkflow(t, root)
			physical := controllerSource(t, source, root)
			if drift == "release-resource" {
				if err := os.WriteFile(filepath.Join(physical, "extra-resource.txt"), []byte("original resource"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			interrupted := false
			if _, err := local.ProtectController(context.Background(), physical, func(phase string) error {
				if phase == "copy" {
					interrupted = true
					return errors.New("interrupt copy")
				}
				return nil
			}); err == nil || !interrupted {
				t.Fatalf("copy interruption was not reached: %v", err)
			}
			var changed string
			switch drift {
			case "business":
				changed = local.statePath
			case "source":
				changed = filepath.Join(os.Getenv("FANLOOP_CONFIG_SOURCE"), "fanloop-workflow", "linear-runtime", "skills", "test-guide", "SKILL.md")
			case "unknown-staging":
				changed = filepath.Join(root, controller.Pending, "unknown")
			case "release-mode":
				changed = filepath.Join(physical, "entrypoints", "fanloop-workflow", "SKILL.md")
			case "release-resource":
				changed = filepath.Join(physical, "extra-resource.txt")
			case "release-collection":
				changed = filepath.Join(physical, "new-resource.txt")
			}
			original, err := os.ReadFile(changed)
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			var changedMode os.FileMode
			if drift == "release-mode" {
				info, err := os.Lstat(changed)
				if err != nil {
					t.Fatal(err)
				}
				changedMode = info.Mode().Perm() ^ 0200
				if err := os.Chmod(changed, changedMode); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(changed, []byte("external drift"), 0600); err != nil {
				t.Fatal(err)
			}
			if binary, err := local.ProtectController(context.Background(), physical, nil); err == nil || binary != "" {
				t.Fatalf("drift accepted: %s %v", binary, err)
			}
			content, err := os.ReadFile(changed)
			if drift == "release-mode" {
				info, statErr := os.Lstat(changed)
				if err != nil || statErr != nil || string(content) != string(original) || info.Mode().Perm() != changedMode {
					t.Fatalf("Release mode drift overwritten: %v %v", err, statErr)
				}
			} else if err != nil || string(content) != "external drift" {
				t.Fatalf("drift overwritten: %q %v", content, err)
			}
		})
	}
}
