package store

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/zeefan1555/fanloop/internal/configuration"
	"github.com/zeefan1555/fanloop/internal/controller"
	"github.com/zeefan1555/fanloop/internal/idl/storageidl"
	"github.com/zeefan1555/fanloop/internal/release"
	"github.com/zeefan1555/fanloop/internal/testfixture"
)

func TestAuthorLiveFreezeFiniteRecoveryKeepsExactReleaseAndHistory(t *testing.T) {
	source := filepath.Join(t.TempDir(), "release")
	if err := testfixture.BuildRelease(source, "test"); err != nil {
		t.Fatal(err)
	}
	for _, phase := range []string{"before-prepared", "copy", "before-publish", "published", "pinned"} {
		t.Run(phase, func(t *testing.T) {
			legacy := t.TempDir()
			_, _ = committedWorkflow(t, legacy)
			physical := controllerSource(t, source, legacy)
			root := t.TempDir()
			command := exec.Command(filepath.Join(physical, "bin", "fanloop"), "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Live freeze recovery")
			command.Env = controller.Environment(os.Getenv("FANLOOP_DATA_HOME"), os.Getenv("FANLOOP_CONFIG_SOURCE"), "")
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("real author init: %v\n%s", err, output)
			}
			local, failure := New(root)
			if failure != nil {
				t.Fatal(failure)
			}
			live, err := controller.Read(root)
			if err != nil || live == nil || live.ConfigurationMode != storageidl.ControllerConfigurationMode_author_live {
				t.Fatalf("missing live binding: %v %v", live, err)
			}
			journal, err := os.ReadFile(filepath.Join(root, controller.Journal))
			if err != nil {
				t.Fatal(err)
			}
			boundRelease := filepath.Join(root, controller.Home, live.ControllerReleasePath)
			digest, modes, err := release.StrictDirectoryIdentity(boundRelease)
			if err != nil {
				t.Fatal(err)
			}
			resource := filepath.Join(live.ConfigurationPin, "fanloop-workflow", "linear-runtime", "skills", "test-guide", "dirty-resource.sh")
			if err := os.WriteFile(resource, []byte("#!/bin/sh\nprintf 'dirty resource\\n'\n"), 0755); err != nil {
				t.Fatal(err)
			}
			configurationBefore, err := configuration.InspectBinding(live.ConfigurationPin)
			if err != nil {
				t.Fatal(err)
			}
			businessBefore, err := local.businessDigest()
			if err != nil {
				t.Fatal(err)
			}
			injected := false
			binary, err := local.ProtectController(context.Background(), filepath.Join(root, "ignored-new-release"), func(current string) error {
				if current == phase {
					injected = true
					return errors.New("injected freeze " + phase)
				}
				return nil
			})
			if !injected || err == nil || binary != "" {
				t.Fatalf("freeze checkpoint was not a safe failure: %s %v", binary, err)
			}
			pin, found, err := controller.ReadPin(root)
			if err != nil || !found || pin != live.ConfigurationPin {
				t.Fatalf("freeze rollback changed live pin: %s %t %v", pin, found, err)
			}
			prepared, err := controller.Read(root)
			if err != nil {
				t.Fatal(err)
			}
			if phase == "before-prepared" {
				if !prepared.DeepEqual(live) {
					t.Fatal("before-prepared failure changed the live journal")
				}
			} else if prepared.Phase != storageidl.ControllerBindingPhase_prepared || prepared.GetPreviousBindingTransactionId() != live.TransactionId {
				t.Fatalf("freeze did not retain exact prepared history: %v", prepared)
			}
			binary, err = local.ProtectController(context.Background(), filepath.Join(root, "still-ignored-release"), nil)
			if err != nil || binary != filepath.Join(boundRelease, "bin", "fanloop") {
				t.Fatalf("freeze retry did not reuse original controller: %s %v", binary, err)
			}
			frozen, err := controller.Read(root)
			if err != nil || frozen.Phase != storageidl.ControllerBindingPhase_committed || frozen.ConfigurationMode != storageidl.ControllerConfigurationMode_author_frozen || frozen.GetPreviousBindingTransactionId() != live.TransactionId || phase != "before-prepared" && frozen.TransactionId != prepared.TransactionId {
				t.Fatalf("freeze retry changed transaction or history: %v %v", frozen, err)
			}
			archive, err := os.ReadFile(filepath.Join(root, ".fanloop", "audit", "controller-bindings", live.TransactionId+".json"))
			if err != nil || !bytes.Equal(archive, journal) {
				t.Fatalf("freeze archive differs: %v", err)
			}
			configurationAfter, err := configuration.InspectBinding(frozen.ConfigurationPin)
			if err != nil || !reflect.DeepEqual(configurationAfter, configurationBefore) {
				t.Fatalf("freeze capture lost bytes or executable bits: %v", err)
			}
			afterDigest, afterModes, err := release.StrictDirectoryIdentity(boundRelease)
			if err != nil || afterDigest != digest || !reflect.DeepEqual(afterModes, modes) || live.ReleaseResourcesSha256 != release.ControllerResourcesDigest(digest, modes) || prepared.ReleaseResourcesSha256 != live.ReleaseResourcesSha256 || frozen.ReleaseResourcesSha256 != live.ReleaseResourcesSha256 {
				t.Fatalf("freeze retry replaced healthy Release: %v", err)
			}
			businessAfter, err := local.businessDigest()
			if err != nil || businessAfter != businessBefore {
				t.Fatalf("freeze failure/retry changed accepted facts: %v", err)
			}
		})
	}
}

func TestProtectAndAuthorFreezeRejectCompletedResourceDrift(t *testing.T) {
	source := filepath.Join(t.TempDir(), "release")
	if err := testfixture.BuildRelease(source, "test"); err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []struct {
		name, phase, target string
		freeze              bool
	}{
		{"protect-staged-source", "before-publish", "source", false},
		{"protect-staged-copy", "before-publish", "copy", false},
		{"protect-published-source", "published", "source", false},
		{"protect-published-copy", "published", "copy", false},
		{"freeze-before-transaction", "", "copy", true},
		{"freeze-staged", "before-publish", "copy", true},
		{"freeze-published", "published", "copy", true},
	} {
		for _, drift := range []string{"bytes", "file-mode", "add-file", "remove-file", "add-empty-directory", "remove-empty-directory"} {
			t.Run(scenario.name+"/"+drift, func(t *testing.T) {
				root := t.TempDir()
				local, _ := committedWorkflow(t, root)
				physical := controllerSource(t, source, root)
				if err := os.WriteFile(filepath.Join(physical, "resource.txt"), []byte("original resource"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(filepath.Join(physical, "empty"), 0700); err != nil {
					t.Fatal(err)
				}
				if scenario.freeze {
					root = t.TempDir()
					command := exec.Command(filepath.Join(physical, "bin", "fanloop"), "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Resource identity recovery")
					if output, err := command.CombinedOutput(); err != nil {
						t.Fatalf("real live initialize: %v\n%s", err, output)
					}
					initialized, failure := New(root)
					if failure != nil {
						t.Fatal(failure)
					}
					local = initialized
				} else if err := local.PinConfiguration(os.Getenv("FANLOOP_CONFIG_SOURCE")); err != nil {
					t.Fatal(err)
				}
				if scenario.phase != "" {
					injected := false
					binary, err := local.ProtectController(context.Background(), physical, func(phase string) error {
						if phase == scenario.phase {
							injected = true
							return errors.New("interrupt completed resources")
						}
						return nil
					})
					if !injected || err == nil || binary != "" {
						t.Fatalf("completed resource checkpoint was not reached: %s %v", binary, err)
					}
				}
				binding, err := controller.Read(root)
				if err != nil || binding == nil {
					t.Fatalf("missing original journal: %v %v", binding, err)
				}
				facts := map[string][]byte{}
				paths := []string{".fanloop/flow/state.json", ".fanloop/output/state.json", ".fanloop/trace/events.jsonl", ".fanloop/configuration-snapshot", controller.Journal}
				if binding.GetPreviousBindingTransactionId() != "" {
					paths = append(paths, ".fanloop/audit/controller-bindings/"+binding.GetPreviousBindingTransactionId()+".json")
				}
				for _, file := range paths {
					content, err := os.ReadFile(filepath.Join(root, file))
					if err != nil {
						t.Fatal(err)
					}
					facts[file] = content
				}
				target := physical
				if scenario.target == "copy" {
					home := controller.Home
					if !scenario.freeze && scenario.phase == "before-publish" {
						home = controller.Pending
					}
					target = filepath.Join(root, home, binding.ControllerReleasePath)
				}
				switch drift {
				case "bytes":
					err = os.WriteFile(filepath.Join(target, "resource.txt"), []byte("external drift"), 0644)
				case "file-mode":
					err = os.Chmod(filepath.Join(target, "resource.txt"), 0775)
				case "add-file":
					err = os.WriteFile(filepath.Join(target, "new-resource.txt"), []byte("external drift"), 0600)
				case "remove-file":
					err = os.Remove(filepath.Join(target, "resource.txt"))
				case "add-empty-directory":
					err = os.Mkdir(filepath.Join(target, "new-empty"), 0700)
				case "remove-empty-directory":
					err = os.Remove(filepath.Join(target, "empty"))
				}
				if err != nil {
					t.Fatal(err)
				}
				if _, err := release.LoadBound(target); err != nil {
					t.Fatalf("resource drift masked by old manifest validation: %v", err)
				}
				digest, modes, err := release.StrictDirectoryIdentity(target)
				if err != nil {
					t.Fatal(err)
				}
				if binary, err := local.ProtectController(context.Background(), physical, nil); err == nil || binary != "" {
					t.Fatalf("accepted completed %s: %s %v", drift, binary, err)
				}
				for file, original := range facts {
					content, err := os.ReadFile(filepath.Join(root, file))
					if err != nil || !bytes.Equal(content, original) {
						t.Fatalf("rejection altered %s: %v", file, err)
					}
				}
				afterDigest, afterModes, err := release.StrictDirectoryIdentity(target)
				if err != nil || afterDigest != digest || !reflect.DeepEqual(afterModes, modes) {
					t.Fatalf("rejection overwrote resource drift: %v", err)
				}
				if scenario.phase == "" {
					if _, err := os.Lstat(filepath.Join(root, ".fanloop", "audit", "controller-bindings", binding.TransactionId+".json")); !os.IsNotExist(err) {
						t.Fatalf("rejected freeze archived or changed prior transaction: %v", err)
					}
				}
			})
		}
	}
}
