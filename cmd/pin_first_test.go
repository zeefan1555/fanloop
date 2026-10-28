package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/configuration"
	"github.com/zeefan1555/fanloop/internal/controller"
)

func TestExistingActualPinIgnoresUnrelatedSources(t *testing.T) {
	for _, mode := range []string{"bound", "unbound"} {
		t.Run(mode, func(t *testing.T) {
			dataRoot, snapshot, _ := configRebindSnapshots(t)
			t.Setenv("FANLOOP_DATA_HOME", dataRoot)
			t.Setenv("FANLOOP_NO_AUTO_CONFIG", "1")
			configUnsetSource(t)
			root := t.TempDir()
			configRebindExecute(t, []string{"flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Actual pin"}, 0)
			if mode == "unbound" {
				preserved := t.TempDir()
				for _, file := range []string{controller.Journal, controller.Home} {
					if err := os.Rename(filepath.Join(root, file), filepath.Join(preserved, filepath.Base(file))); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.WriteFile(filepath.Join(root, ".fanloop", "configuration-snapshot"), []byte(snapshot+"\n"), 0644); err != nil {
					t.Fatal(err)
				}
			} else {
				snapshot = filepath.Join(root, controller.Home, "config", "snapshots", filepath.Base(snapshot))
			}
			facts := map[string][]byte{}
			for _, file := range []string{"flow/state.json", "output/state.json", "trace/events.jsonl", "configuration-snapshot"} {
				content, err := os.ReadFile(filepath.Join(root, ".fanloop", file))
				if err != nil {
					t.Fatal(err)
				}
				facts[file] = content
			}
			if err := os.Remove(filepath.Join(dataRoot, "config", "current")); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join(dataRoot, "missing"), filepath.Join(dataRoot, "config", "current")); err != nil {
				t.Fatal(err)
			}
			t.Setenv("FANLOOP_CONFIG_SOURCE", filepath.Join(dataRoot, "invalid-author"))
			result := configRebindExecute(t, []string{"flow", "status", "--root", root}, 0)
			if !strings.Contains(result, snapshot) {
				t.Fatalf("status did not read the actual pin: %s", result)
			}
			for file, original := range facts {
				content, err := os.ReadFile(filepath.Join(root, ".fanloop", file))
				if err != nil || !bytes.Equal(content, original) {
					t.Fatalf("status altered %s: %v", file, err)
				}
			}
		})
	}
}

func TestManagedActualPinRejectsDriftWithoutFallback(t *testing.T) {
	dataRoot, snapshot, _ := configRebindSnapshots(t)
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_NO_AUTO_CONFIG", "1")
	t.Setenv("FANLOOP_NO_AUTO_UPDATE", "1")
	configUnsetSource(t)
	sourceRelease := newFlowRuntime().ControllerRelease
	binary := filepath.Join(sourceRelease, "bin", "fanloop")
	launcher, err := filepath.Abs("../packaging/npm/scripts/run.js")
	if err != nil {
		t.Fatal(err)
	}
	installArgs := []string{"__install", "--source", sourceRelease, "--data-root", dataRoot}
	for index, name := range []string{"codex", "agent", "trae", "claude"} {
		installArgs = append(installArgs, "--"+name+"-skills-root", controller.SkillRoots(dataRoot).Values()[index])
	}
	if output, err := exec.Command(binary, installArgs...).CombinedOutput(); err != nil {
		t.Fatalf("install isolated caller Release: %v\n%s", err, output)
	}
	for _, mode := range []string{"bound", "unbound"} {
		t.Run(mode, func(t *testing.T) {
			for _, drift := range []string{"manifest-missing", "manifest-corrupt", "metadata-drift", "content-drift", "executable-drift", "linked-snapshot"} {
				t.Run(drift, func(t *testing.T) {
					root := t.TempDir()
					command := exec.Command(binary, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", drift)
					if output, err := command.CombinedOutput(); err != nil {
						t.Fatalf("initialize healthy snapshot: %v\n%s", err, output)
					}
					pinPath := filepath.Join(root, ".fanloop", "configuration-snapshot")
					content, err := os.ReadFile(pinPath)
					if err != nil {
						t.Fatal(err)
					}
					actualPin := strings.TrimSpace(string(content))
					if actualPin == snapshot {
						t.Fatal("snapshot was not protected under the Root")
					}
					files := []string{".fanloop/flow/state.json", ".fanloop/output/state.json", ".fanloop/trace/events.jsonl", ".fanloop/configuration-snapshot"}
					if mode == "unbound" {
						preserved := t.TempDir()
						externalPin := filepath.Join(preserved, "config", "snapshots", filepath.Base(actualPin))
						if err := os.MkdirAll(filepath.Dir(externalPin), 0755); err != nil {
							t.Fatal(err)
						}
						if err := os.Rename(actualPin, externalPin); err != nil {
							t.Fatal(err)
						}
						for _, file := range []string{controller.Journal, controller.Home} {
							if err := os.Rename(filepath.Join(root, file), filepath.Join(preserved, filepath.Base(file))); err != nil {
								t.Fatal(err)
							}
						}
						actualPin = externalPin
						if err := os.WriteFile(pinPath, []byte(actualPin+"\n"), 0600); err != nil {
							t.Fatal(err)
						}
					} else {
						files = append(files, controller.Journal)
					}
					facts := map[string][]byte{}
					for _, file := range files {
						content, err := os.ReadFile(filepath.Join(root, file))
						if err != nil {
							t.Fatal(err)
						}
						facts[file] = content
					}
					t.Setenv("FANLOOP_CONFIG_SOURCE", snapshot)
					if err := os.Chmod(actualPin, 0755); err != nil {
						t.Fatal(err)
					}
					manifestPath := filepath.Join(actualPin, "configuration.json")
					switch drift {
					case "manifest-missing":
						err = os.Remove(manifestPath)
					case "manifest-corrupt", "metadata-drift":
						content := []byte("not-json\n")
						if drift == "metadata-drift" {
							manifest, failure := configuration.Read(actualPin)
							if failure != nil {
								t.Fatal(failure)
							}
							manifest.SchemaVersion++
							content, err = json.Marshal(manifest)
							if err != nil {
								t.Fatal(err)
							}
						}
						if err := os.Chmod(manifestPath, 0644); err != nil {
							t.Fatal(err)
						}
						err = os.WriteFile(manifestPath, content, 0644)
					case "content-drift", "executable-drift":
						file := filepath.Join(actualPin, "fanloop-workflow", "linear-runtime", "skills", "test-guide", "SKILL.md")
						if err := os.Chmod(file, 0644); err != nil {
							t.Fatal(err)
						}
						if drift == "executable-drift" {
							err = os.Chmod(file, 0755)
						} else {
							err = os.WriteFile(file, []byte("# Valid but modified Skill\n"), 0644)
						}
					case "linked-snapshot":
						if err := os.Rename(actualPin, actualPin+".preserved"); err != nil {
							t.Fatal(err)
						}
						err = os.Symlink(actualPin+".preserved", actualPin)
					}
					if err != nil {
						t.Fatal(err)
					}
					for _, surface := range []string{"direct", "node"} {
						args := []string{"flow", "status", "--root", root}
						command := exec.Command(binary, args...)
						if surface == "node" {
							command = exec.Command("node", append([]string{launcher}, args...)...)
						}
						var stdout, stderr bytes.Buffer
						command.Stdout, command.Stderr = &stdout, &stderr
						failure := command.Run()
						if exit, ok := failure.(*exec.ExitError); !ok || exit.ExitCode() != 5 || stdout.Len() != 0 || stderr.Len() == 0 {
							t.Fatalf("%s accepted %s or fell back: %v\nstdout=%s\nstderr=%s", surface, drift, failure, stdout.String(), stderr.String())
						}
						for file, original := range facts {
							content, err := os.ReadFile(filepath.Join(root, file))
							if err != nil || !bytes.Equal(content, original) {
								t.Fatalf("%s rejection altered %s: %v", surface, file, err)
							}
						}
						if mode == "unbound" {
							for _, file := range []string{controller.Journal, controller.Home} {
								if _, err := os.Lstat(filepath.Join(root, file)); !os.IsNotExist(err) {
									t.Fatalf("%s rejection created %s: %v", surface, file, err)
								}
							}
						}
					}
				})
			}
		})
	}
}
