package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/errs"
	"github.com/zeefan1555/fanloop/internal/configuration"
	"github.com/zeefan1555/fanloop/internal/controller"
	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/output"
)

func TestControllerEntrypointFailuresKeepPublicEnvelopeAndApprovedLoggingBoundary(t *testing.T) {
	dataHome, _, _ := configRebindSnapshots(t)
	t.Setenv("FANLOOP_DATA_HOME", dataHome)
	t.Setenv("FANLOOP_NO_AUTO_CONFIG", "1")
	t.Setenv("FANLOOP_NO_AUTO_UPDATE", "1")
	configUnsetSource(t)
	binary := filepath.Join(newFlowRuntime().ControllerRelease, "bin", "fanloop")
	launcher, err := filepath.Abs("../packaging/npm/scripts/run.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, damage := range []string{"journal", "release", "configuration-workflow-ids"} {
		t.Run(damage, func(t *testing.T) {
			root := t.TempDir()
			if result, err := exec.Command(binary, "flow", "init", "--root", root, "--workflow", syntheticWorkflowID, "--title", "preflight failure").CombinedOutput(); err != nil {
				t.Fatalf("initialize real controller: %v\n%s", err, result)
			}
			manifestRelative := ""
			switch damage {
			case "journal":
				err = os.WriteFile(filepath.Join(root, controller.Journal), []byte("{bad journal\n"), 0600)
			case "release":
				err = os.WriteFile(filepath.Join(root, controller.Home, "releases", "test", "added-resource"), []byte("drift\n"), 0600)
			case "configuration-workflow-ids":
				pin := strings.TrimSpace(string(configRead(t, filepath.Join(root, ".fanloop/configuration-snapshot"))))
				manifestPath := filepath.Join(pin, "configuration.json")
				var manifest configuration.Manifest
				if err := json.Unmarshal(configRead(t, manifestPath), &manifest); err != nil {
					t.Fatal(err)
				}
				manifest.WorkflowIDs = []string{"missing-workflow"}
				content, failure := json.MarshalIndent(manifest, "", "  ")
				if failure != nil {
					t.Fatal(failure)
				}
				if err := os.Chmod(manifestPath, 0600); err != nil {
					t.Fatal(err)
				}
				err = os.WriteFile(manifestPath, append(content, '\n'), 0600)
				manifestRelative, failure = filepath.Rel(root, manifestPath)
				if failure != nil {
					t.Fatal(failure)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			facts := map[string][]byte{}
			for _, relative := range []string{".fanloop/flow/state.json", ".fanloop/output/state.json", ".fanloop/trace/events.jsonl", ".fanloop/configuration-snapshot", controller.Journal} {
				facts[relative] = configRead(t, filepath.Join(root, relative))
			}
			if manifestRelative != "" {
				facts[manifestRelative] = configRead(t, filepath.Join(root, manifestRelative))
			}
			for _, invocation := range []struct {
				name, commandID string
				args            []string
				dryRun          bool
			}{
				{"status", "flow.status", []string{"flow", "status", "--root", root}, false},
				{"doctor", "doctor", []string{"doctor", "--root", root}, false},
				{"progress-dry-run", "flow.report.progress", []string{"--root=" + root, "flow", "report", "progress", "--dry-run"}, true},
				{"progress-no-dry-run", "flow.report.progress", []string{"flow", "report", "progress", "--dry-run=false", "--root", root}, false},
				{"progress-zero-dry-run", "flow.report.progress", []string{"flow", "report", "progress", "--dry-run=0", "--root", root}, false},
				{"progress-uppercase-false", "flow.report.progress", []string{"flow", "report", "progress", "--dry-run=FALSE", "--root", root}, false},
				{"progress-mixedcase-true", "flow.report.progress", []string{"flow", "report", "progress", "--dry-run=True", "--root", root}, true},
			} {
				t.Run(invocation.name, func(t *testing.T) {
					var directEnvelope map[string]any
					for _, surface := range []string{"direct", "node"} {
						t.Run(surface, func(t *testing.T) {
							before := readExecutionLog(t, root)
							command := exec.Command(binary, invocation.args...)
							if surface == "node" {
								command = exec.Command("node", append([]string{launcher}, invocation.args...)...)
							}
							var stdout, stderr bytes.Buffer
							command.Stdout, command.Stderr = &stdout, &stderr
							failure := command.Run()
							if exit, ok := failure.(*exec.ExitError); !ok || exit.ExitCode() != 5 || stdout.Len() != 0 {
								t.Fatalf("invalid preflight exit/stdout: %v\n%s\n%s", failure, stdout.String(), stderr.String())
							}
							var envelope map[string]json.RawMessage
							if err := json.Unmarshal(stderr.Bytes(), &envelope); err != nil {
								t.Fatalf("preflight did not return one JSON error envelope: %v\n%s", err, stderr.String())
							}
							if len(envelope) != 4 || string(envelope["ok"]) != "false" || string(envelope["_notice"]) != "{}" {
								t.Errorf("public error envelope differs: %s", stderr.String())
							}
							var public errs.Presentation
							if err := json.Unmarshal(envelope["error"], &public); err != nil || public.Type != erroridl.ErrorType_internal || public.Code != erroridl.ErrorCode_INTERNAL || public.Hint != erroridl.ERROR_SPECS[erroridl.ErrorCode_INTERNAL].Hint || public.Retryable || public.Message == "" || len(public.Details) != 0 {
								t.Errorf("public error fields differ: %+v %v", public, err)
							}
							var meta output.Meta
							if err := json.Unmarshal(envelope["meta"], &meta); err != nil || meta.Command != invocation.commandID || meta.RequirementRoot != root || meta.DryRun != invocation.dryRun || meta.Workflow != nil {
								t.Errorf("public error metadata differs: %+v %v", meta, err)
							}
							var protocol map[string]any
							if err := json.Unmarshal(stderr.Bytes(), &protocol); err != nil {
								t.Fatal(err)
							}
							protocol["error"].(map[string]any)["message"] = "controller preflight failure"
							if surface == "direct" {
								directEnvelope = protocol
							} else if !reflect.DeepEqual(protocol, directEnvelope) {
								t.Errorf("actual Node/Go public protocols differ: Node=%v Go=%v", protocol, directEnvelope)
							}
							after := readExecutionLog(t, root)
							if surface == "node" {
								if !reflect.DeepEqual(after, before) {
									t.Error("Node preflight started no trusted CLI but changed its execution log")
								}
							} else if len(after) != len(before)+1 {
								t.Errorf("Go preflight did not append exactly one execution log: before=%d after=%d", len(before), len(after))
							} else {
								entry := after[len(after)-1]
								if entry.CommandId != invocation.commandID || entry.ExitCode != 5 || entry.GetErrorCode() != "INTERNAL" || entry.DryRun != invocation.dryRun || !reflect.DeepEqual(entry.Arguments, invocation.args) || entry.Stdin != "" || entry.Stdout != stdout.String() || entry.Stderr != stderr.String() {
									t.Errorf("preflight execution transcript differs: %+v", entry)
								}
							}
							for relative, expected := range facts {
								if actual := configRead(t, filepath.Join(root, relative)); !bytes.Equal(actual, expected) {
									t.Errorf("preflight changed %s", relative)
								}
							}
						})
					}
				})
			}
		})
	}
}

func TestControllerLegacyNoPinScrubsCallerOverrideOnEveryEntrypoint(t *testing.T) {
	dataHome, _, _ := configRebindSnapshots(t)
	t.Setenv("FANLOOP_DATA_HOME", dataHome)
	t.Setenv("FANLOOP_NO_AUTO_CONFIG", "1")
	t.Setenv("FANLOOP_NO_AUTO_UPDATE", "1")
	configUnsetSource(t)
	binary := filepath.Join(newFlowRuntime().ControllerRelease, "bin", "fanloop")
	root := t.TempDir()
	if result, err := exec.Command(binary, "flow", "init", "--root", root, "--workflow", syntheticWorkflowID, "--title", "legacy no pin").CombinedOutput(); err != nil {
		t.Fatalf("initialize real controller: %v\n%s", err, result)
	}
	for _, relative := range []string{controller.Journal, ".fanloop/configuration-snapshot"} {
		file := filepath.Join(root, relative)
		if err := os.Rename(file, file+".preserved"); err != nil {
			t.Fatal(err)
		}
	}
	facts := map[string][]byte{}
	for _, relative := range []string{"flow/state.json", "output/state.json", "trace/events.jsonl"} {
		facts[relative] = configRead(t, filepath.Join(root, ".fanloop", relative))
	}
	t.Setenv("FANLOOP_CONFIG_SOURCE", filepath.Join(t.TempDir(), "broken-caller-override"))
	launcher, err := filepath.Abs("../packaging/npm/scripts/run.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, surface := range []string{"same-binary", "different-binary", "node"} {
		t.Run(surface, func(t *testing.T) {
			args := []string{"flow", "status", "--root", root}
			command := exec.Command(binary, args...)
			if surface == "same-binary" {
				command = exec.Command(filepath.Join(root, controller.Home, "releases", "test", "bin", "fanloop"), args...)
			} else if surface == "node" {
				command = exec.Command("node", append([]string{launcher}, args...)...)
			}
			if result, err := command.CombinedOutput(); err != nil || !strings.Contains(string(result), "legacy no pin") {
				t.Errorf("legacy controller did not ignore caller override: %v\n%s", err, result)
			}
			for relative, expected := range facts {
				if actual := configRead(t, filepath.Join(root, ".fanloop", relative)); !bytes.Equal(actual, expected) {
					t.Errorf("legacy selection changed %s", relative)
				}
			}
		})
	}
}

func TestControllerRelativeRootKeepsArgumentValidation(t *testing.T) {
	dataHome, snapshot, _ := configRebindSnapshots(t)
	t.Setenv("FANLOOP_DATA_HOME", dataHome)
	t.Setenv("FANLOOP_NO_AUTO_CONFIG", "1")
	t.Setenv("FANLOOP_NO_AUTO_UPDATE", "1")
	configUnsetSource(t)
	sourceRelease := newFlowRuntime().ControllerRelease
	binary := filepath.Join(sourceRelease, "bin", "fanloop")
	launcher, err := filepath.Abs("../packaging/npm/scripts/run.js")
	if err != nil {
		t.Fatal(err)
	}
	installArgs := []string{"__install", "--source", sourceRelease, "--data-root", dataHome}
	for index, name := range []string{"codex", "agent", "trae", "claude"} {
		installArgs = append(installArgs, "--"+name+"-skills-root", controller.SkillRoots(dataHome).Values()[index])
	}
	if output, err := exec.Command(binary, installArgs...).CombinedOutput(); err != nil {
		t.Fatalf("install isolated caller Release: %v\n%s", err, output)
	}
	for _, mode := range []string{"unbound", "bound"} {
		t.Run(mode, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "root with space and 'quote'")
			if err := os.Mkdir(root, 0700); err != nil {
				t.Fatal(err)
			}
			if output, err := exec.Command(binary, "flow", "init", "--root", root, "--workflow", syntheticWorkflowID, "--title", "relative root validation").CombinedOutput(); err != nil {
				t.Fatalf("initialize absolute Root: %v\n%s", err, output)
			}
			files := []string{".fanloop/flow/state.json", ".fanloop/output/state.json", ".fanloop/trace/events.jsonl", ".fanloop/configuration-snapshot", ".fanloop/log/cli.jsonl"}
			if mode == "unbound" {
				preserved := t.TempDir()
				for _, name := range []string{controller.Journal, controller.Home} {
					if err := os.Rename(filepath.Join(root, name), filepath.Join(preserved, filepath.Base(name))); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.WriteFile(filepath.Join(root, ".fanloop/configuration-snapshot"), []byte(snapshot+"\n"), 0600); err != nil {
					t.Fatal(err)
				}
			} else {
				files = append(files, controller.Journal)
			}
			facts := map[string][]byte{}
			for _, name := range files {
				facts[name] = configRead(t, filepath.Join(root, name))
			}
			for _, invocation := range []struct {
				name, cwd, root string
				args            []string
			}{
				{"separate", filepath.Dir(root), filepath.Base(root), []string{"flow", "status", "--root", filepath.Base(root)}},
				{"equals", filepath.Dir(root), filepath.Base(root), []string{"flow", "status", "--root=" + filepath.Base(root)}},
				{"dot", root, ".", []string{"flow", "status", "--root", "."}},
			} {
				for _, surface := range []string{"direct", "node"} {
					t.Run(invocation.name+"/"+surface, func(t *testing.T) {
						command := exec.Command(binary, invocation.args...)
						if surface == "node" {
							command = exec.Command("node", append([]string{launcher}, invocation.args...)...)
						}
						command.Dir = invocation.cwd
						var stdout, stderr bytes.Buffer
						command.Stdout, command.Stderr = &stdout, &stderr
						failure := command.Run()
						exitCode := 0
						if failure != nil {
							exit, ok := failure.(*exec.ExitError)
							if !ok {
								t.Fatal(failure)
							}
							exitCode = exit.ExitCode()
						}
						t.Logf("%s %s args=%q exit=%d stdout=%q stderr=%s", mode, surface, invocation.args, exitCode, stdout.String(), stderr.String())
						var envelope struct {
							Error struct{ Code, Message string }
							Meta  output.Meta
						}
						if err := json.Unmarshal(stderr.Bytes(), &envelope); err != nil || exitCode != 2 || stdout.Len() != 0 || envelope.Error.Code != "INVALID_ARGUMENT" || envelope.Error.Message != "--root must be an absolute path" || envelope.Meta.Command != "flow.status" || envelope.Meta.RequirementRoot != invocation.root {
							t.Errorf("relative Root did not retain public argument validation: %v", err)
						}
						for name, before := range facts {
							if after := configRead(t, filepath.Join(root, name)); !bytes.Equal(before, after) {
								t.Errorf("relative Root changed %s", name)
							}
						}
					})
				}
			}
		})
	}
}

func TestControllerDoctorKeepsWorkflowErrorProtocol(t *testing.T) {
	source := t.TempDir()
	configCopyTree(t, "../internal/workflow/testdata/linear-runtime", filepath.Join(source, "fanloop-workflow", "linear-runtime"))
	t.Setenv("FANLOOP_DATA_HOME", t.TempDir())
	t.Setenv("FANLOOP_CONFIG_SOURCE", source)
	t.Setenv("FANLOOP_NO_AUTO_UPDATE", "1")
	t.Setenv("FANLOOP_NO_AUTO_CONFIG", "1")
	binary := filepath.Join(newFlowRuntime().ControllerRelease, "bin", "fanloop")
	launcher, err := filepath.Abs("../packaging/npm/scripts/run.js")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if output, err := exec.Command(binary, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Workflow error owner").CombinedOutput(); err != nil {
		t.Fatalf("author fixture init: %v\n%s", err, output)
	}
	binding, err := controller.Read(root)
	if err != nil || binding == nil {
		t.Fatalf("missing author binding: %v", err)
	}
	facts := map[string][]byte{}
	for _, relative := range []string{".fanloop/flow/state.json", ".fanloop/output/state.json", ".fanloop/trace/events.jsonl", ".fanloop/configuration-snapshot", controller.Journal} {
		facts[relative] = configRead(t, filepath.Join(root, relative))
	}
	if err := os.WriteFile(filepath.Join(source, "fanloop-workflow", "linear-runtime", "workflow.yaml"), []byte("schema_version: [\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var expectedStdout, expectedStderr string
	for _, surface := range []string{"direct", "same-binary", "node"} {
		t.Run(surface, func(t *testing.T) {
			args := []string{"doctor", "--root", root}
			command := exec.Command(binary, args...)
			if surface == "same-binary" {
				command = exec.Command(filepath.Join(root, controller.Home, binding.ControllerReleasePath, "bin", "fanloop"), args...)
			} else if surface == "node" {
				command = exec.Command("node", append([]string{launcher}, args...)...)
			}
			before := readExecutionLog(t, root)
			var stdout, stderr bytes.Buffer
			command.Stdout, command.Stderr = &stdout, &stderr
			failure := command.Run()
			if exit, ok := failure.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
				t.Errorf("Workflow diagnosis did not retain Doctor exit 1: %v\nstdout=%s\nstderr=%s", failure, stdout.String(), stderr.String())
			}
			var report struct {
				OK   bool `json:"ok"`
				Data struct {
					Scope  string                        `json:"scope"`
					Status string                        `json:"status"`
					Checks []struct{ ID, Status string } `json:"checks"`
				} `json:"data"`
			}
			if err := json.Unmarshal(stdout.Bytes(), &report); err != nil || report.OK || report.Data.Scope != "requirement" || report.Data.Status != "unhealthy" {
				t.Errorf("Workflow error replaced Doctor's stdout report: %v\n%s", err, stdout.String())
			}
			workflowFailed := false
			for _, check := range report.Data.Checks {
				workflowFailed = workflowFailed || check.ID == "workflows" && check.Status == "failed"
			}
			if !workflowFailed || strings.Count(stderr.String(), "author/live") != 1 || strings.Contains(stderr.String(), "\"error\"") {
				t.Errorf("invalid author Workflow was not diagnosed by Doctor: stdout=%s stderr=%s", stdout.String(), stderr.String())
			}
			if surface == "direct" {
				expectedStdout, expectedStderr = stdout.String(), stderr.String()
			} else if stdout.String() != expectedStdout || stderr.String() != expectedStderr {
				t.Error("Go/Node Doctor protocols differ")
			}
			after := readExecutionLog(t, root)
			if len(after) != len(before)+1 {
				t.Errorf("Doctor did not append exactly one CLI entry: before=%d after=%d", len(before), len(after))
			} else {
				entry := after[len(after)-1]
				if entry.CommandId != "doctor" || entry.ExitCode != 1 || entry.CliVersion != "test" || entry.Stdout != stdout.String() || entry.Stderr != stderr.String() {
					t.Errorf("Doctor execution transcript differs: %+v", entry)
				}
			}
			for relative, expected := range facts {
				if actual := configRead(t, filepath.Join(root, relative)); !bytes.Equal(actual, expected) {
					t.Errorf("Workflow error changed %s", relative)
				}
			}
		})
	}
}
