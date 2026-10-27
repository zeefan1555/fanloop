package configuration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func bindingConfiguration(t *testing.T, kind string) (string, Manifest) {
	t.Helper()
	source, err := filepath.EvalSymlinks(sourceRepository(t))
	if err != nil {
		t.Fatal(err)
	}
	var manifest Manifest
	switch kind {
	case "author":
		manifest, err = inspect(source, "")
	case "git":
		source, err = Import(context.Background(), t.TempDir(), source)
	case "fornax":
		source, err = PublishRelease(t.TempDir(), source, bindingReleaseRefs())
	default:
		t.Fatalf("unknown binding fixture %q", kind)
	}
	if err != nil {
		t.Fatal(err)
	}
	if kind != "author" {
		manifest, err = Read(source)
		if err != nil {
			t.Fatal(err)
		}
	}
	return source, manifest
}

func bindingReleaseRefs() []ReleaseRef {
	return []ReleaseRef{
		{Key: "fanloop.workflow.linear_runtime", Version: "1.0.0", WorkflowID: "linear-runtime"},
		{Key: "fanloop.workflow.runtime_controls", Version: "1.0.0", WorkflowID: "runtime-controls"},
	}
}

func bindingWriteManifest(t *testing.T, source string, manifest Manifest) {
	t.Helper()
	path := filepath.Join(source, "configuration.json")
	content, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
}

func bindingCheckNode(t *testing.T, source string, expected Manifest, valid bool) {
	t.Helper()
	module, err := filepath.Abs("../../packaging/npm/scripts/controller.js")
	if err != nil {
		t.Fatal(err)
	}
	script := "const inspect = require(process.argv[1]).configurationIdentity;" +
		"try { process.stdout.write(JSON.stringify({valid:true,digest:inspect(process.argv[2])})); }" +
		"catch(error) { process.stdout.write(JSON.stringify({valid:false,error:error.message})); }"
	command := exec.Command("node", "-e", script, module, source)
	content, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("Node resource inspector failed to run: %v\n%s", err, content)
	}
	var result struct {
		Valid  bool
		Digest string
		Error  string
	}
	if err := json.Unmarshal(content, &result); err != nil {
		t.Fatalf("invalid Node inspector result: %v\n%s", err, content)
	}
	if result.Valid != valid || valid && result.Digest != expected.SHA256 || !valid && result.Error == "" {
		t.Errorf("Node resource identity differs from Go: valid=%t expected_digest=%q result=%+v", valid, expected.SHA256, result)
	}
}

func TestInspectBindingSeparatesResourcesFromWorkflowSemantics(t *testing.T) {
	for _, kind := range []string{"author", "git", "fornax"} {
		t.Run(kind, func(t *testing.T) {
			source, expected := bindingConfiguration(t, kind)
			actual, err := InspectBinding(source)
			if err != nil || !reflect.DeepEqual(actual, expected) {
				t.Fatalf("valid binding identity changed: %v\nwant=%+v\ngot=%+v", err, expected, actual)
			}
			bindingCheckNode(t, source, expected, true)
			relative := Category + "/linear-runtime/workflow.yaml"
			path := filepath.Join(source, filepath.FromSlash(relative))
			content := []byte("schema_version: [\n")
			if err := os.Chmod(path, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, content, 0600); err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(content)
			found := false
			for index := range expected.Files {
				if expected.Files[index].Path == relative {
					expected.Files[index].SHA256 = hex.EncodeToString(digest[:])
					found = true
				}
			}
			if !found {
				t.Fatal("valid fixture did not contain workflow.yaml")
			}
			files, err := json.Marshal(expected.Files)
			if err != nil {
				t.Fatal(err)
			}
			digest = sha256.Sum256(files)
			expected.SHA256 = "sha256:" + hex.EncodeToString(digest[:])
			if kind != "author" {
				bindingWriteManifest(t, source, expected)
			}
			actual, err = InspectBinding(source)
			if err != nil || !reflect.DeepEqual(actual, expected) {
				t.Errorf("physical binding invoked Workflow semantics: %v\nwant=%+v\ngot=%+v", err, expected, actual)
			}
			bindingCheckNode(t, source, expected, true)
			if unlock, err := PrepareSource(source); err == nil {
				unlock()
				t.Error("ordinary source preparation accepted invalid YAML")
			}
			if kind != "author" {
				if _, err := Read(source); err == nil {
					t.Error("ordinary snapshot read accepted invalid YAML")
				}
			}
			if _, err := PublishRelease(t.TempDir(), source, bindingReleaseRefs()); err == nil {
				t.Error("ordinary publication accepted invalid YAML")
			}
		})
	}
}

func TestInspectBindingRetainsManagedIdentityChecks(t *testing.T) {
	for _, kind := range []string{"git", "fornax"} {
		t.Run(kind, func(t *testing.T) {
			original, _ := bindingConfiguration(t, kind)
			cases := []string{"changed-file", "missing-file", "extra-file", "executable-bit", "missing-manifest", "manifest-symlink", "workflow-ids", "files", "digest", "schema", "ref", "origin", "commit", "directory"}
			if kind == "fornax" {
				cases = append(cases, "release-version", "release-workflow", "release-order", "release-digest", "repository")
			}
			for _, damage := range cases {
				t.Run(damage, func(t *testing.T) {
					source := filepath.Join(t.TempDir(), "config", "snapshots", filepath.Base(original))
					if err := os.CopyFS(source, os.DirFS(original)); err != nil {
						t.Fatal(err)
					}
					manifest, err := InspectBinding(source)
					if err != nil {
						t.Fatal(err)
					}
					file := filepath.Join(source, Category, "linear-runtime", "skills", "test-guide", "SKILL.md")
					changedManifest := false
					switch damage {
					case "changed-file":
						err = os.WriteFile(file, []byte("changed\n"), 0600)
					case "missing-file":
						err = os.Remove(file)
					case "extra-file":
						err = os.WriteFile(filepath.Join(filepath.Dir(file), "extra"), []byte("extra\n"), 0600)
					case "executable-bit":
						var info os.FileInfo
						info, err = os.Stat(file)
						if err == nil {
							err = os.Chmod(file, info.Mode().Perm()^0100)
						}
					case "missing-manifest", "manifest-symlink":
						manifestPath := filepath.Join(source, "configuration.json")
						if err = os.Remove(manifestPath); err == nil && damage == "manifest-symlink" {
							err = os.Symlink(filepath.Join(original, "configuration.json"), manifestPath)
						}
					case "directory":
						renamed := source + "-mismatch"
						err = os.Rename(source, renamed)
						source = renamed
					default:
						changedManifest = true
						switch damage {
						case "workflow-ids":
							manifest.WorkflowIDs = append(manifest.WorkflowIDs, "missing-workflow")
						case "files":
							manifest.Files = manifest.Files[1:]
						case "digest":
							manifest.SHA256 = "sha256:" + strings.Repeat("0", 64)
						case "schema":
							manifest.SchemaVersion = 2
						case "ref":
							manifest.Ref = "refs/heads/other"
						case "origin":
							manifest.Origin = "unknown"
						case "commit":
							manifest.CommitSHA = strings.Repeat("1", 40)
						case "release-version":
							manifest.Releases[0].Version = "changed"
						case "release-workflow":
							manifest.Releases[0].WorkflowID = "missing-workflow"
						case "release-order":
							manifest.Releases[0], manifest.Releases[1] = manifest.Releases[1], manifest.Releases[0]
						case "release-digest":
							manifest.ReleaseSetDigest = strings.Repeat("0", 64)
						case "repository":
							manifest.Repository = Repository
						default:
							t.Fatalf("unknown damage %q", damage)
						}
					}
					if err != nil {
						t.Fatal(err)
					}
					if changedManifest {
						bindingWriteManifest(t, source, manifest)
					}
					if _, err := InspectBinding(source); err == nil {
						t.Fatal("resource binding accepted managed identity damage")
					}
					bindingCheckNode(t, source, Manifest{}, false)
				})
			}
			if kind == "git" {
				t.Run("recorded-repository-url", func(t *testing.T) {
					manifest, err := Read(original)
					if err != nil {
						t.Fatal(err)
					}
					manifest.Repository = RepositorySSH
					bindingWriteManifest(t, original, manifest)
					actual, err := InspectBinding(original)
					if err != nil || !reflect.DeepEqual(actual, manifest) {
						t.Fatalf("existing repository URL exception changed: %v", err)
					}
					bindingCheckNode(t, original, manifest, true)
				})
			}
		})
	}
}

func TestInspectBindingRejectsUnsafeAuthorResources(t *testing.T) {
	for _, damage := range []string{"file-link", "directory-link", "fifo", "linked-source", "linked-ancestor", "missing-category"} {
		t.Run(damage, func(t *testing.T) {
			source, _ := bindingConfiguration(t, "author")
			guide := filepath.Join(source, Category, "linear-runtime", "skills", "test-guide")
			var err error
			switch damage {
			case "file-link":
				err = os.Symlink(filepath.Join(guide, "SKILL.md"), filepath.Join(guide, "linked-file"))
			case "directory-link":
				err = os.Symlink(t.TempDir(), filepath.Join(guide, "linked-directory"))
			case "fifo":
				err = unix.Mkfifo(filepath.Join(guide, "fifo"), 0600)
			case "linked-source":
				link := filepath.Join(t.TempDir(), "source-link")
				err = os.Symlink(source, link)
				source = link
			case "linked-ancestor":
				link := filepath.Join(t.TempDir(), "ancestor-link")
				err = os.Symlink(filepath.Dir(source), link)
				source = filepath.Join(link, filepath.Base(source))
			case "missing-category":
				err = os.Rename(filepath.Join(source, Category), filepath.Join(source, "preserved-category"))
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := InspectBinding(source); err == nil {
				t.Fatal("unsafe author resource accepted")
			}
			bindingCheckNode(t, source, Manifest{}, false)
		})
	}
}

func TestInspectBindingManifestEncodingMatchesNode(t *testing.T) {
	for _, kind := range []string{"git", "fornax"} {
		t.Run(kind, func(t *testing.T) {
			original, _ := bindingConfiguration(t, kind)
			for _, fixture := range []struct {
				name  string
				valid bool
				edit  func(map[string]any)
			}{
				{"reordered-object-keys", true, func(map[string]any) {}},
				{"ignored-unknown-fields", true, func(manifest map[string]any) {
					manifest["unknown_manifest_field"] = "ignored"
					manifest["files"].([]any)[0].(map[string]any)["unknown_file_field"] = true
					if kind == "fornax" {
						manifest["releases"].([]any)[0].(map[string]any)["unknown_release_field"] = 1
					}
				}},
				{"null-empty-optionals", true, func(manifest map[string]any) {
					manifest["repository"] = nil
					if kind == "git" {
						manifest["origin"], manifest["release_set_digest"], manifest["releases"] = nil, nil, nil
					} else {
						manifest["ref"], manifest["commit_sha"] = nil, nil
					}
				}},
				{"schema-type", false, func(manifest map[string]any) { manifest["schema_version"] = "1" }},
				{"repository-type", false, func(manifest map[string]any) { manifest["repository"] = 7 }},
				{"workflow-ids-type", false, func(manifest map[string]any) { manifest["workflow_ids"] = "linear-runtime" }},
				{"workflow-ids-null", false, func(manifest map[string]any) { manifest["workflow_ids"] = nil }},
				{"workflow-ids-missing", false, func(manifest map[string]any) { delete(manifest, "workflow_ids") }},
				{"workflow-ids-order", false, func(manifest map[string]any) {
					ids := manifest["workflow_ids"].([]any)
					ids[0], ids[1] = ids[1], ids[0]
				}},
				{"files-null", false, func(manifest map[string]any) { manifest["files"] = nil }},
				{"files-missing", false, func(manifest map[string]any) { delete(manifest, "files") }},
				{"files-order", false, func(manifest map[string]any) {
					files := manifest["files"].([]any)
					files[0], files[1] = files[1], files[0]
				}},
				{"file-boolean-type", false, func(manifest map[string]any) {
					manifest["files"].([]any)[0].(map[string]any)["executable"] = "false"
				}},
				{"releases-list-type", false, func(manifest map[string]any) { manifest["releases"] = map[string]any{} }},
				{"releases-empty-array", false, func(manifest map[string]any) { manifest["releases"] = []any{} }},
				{"release-set-digest", false, func(manifest map[string]any) { manifest["release_set_digest"] = strings.Repeat("0", 64) }},
			} {
				t.Run(fixture.name, func(t *testing.T) {
					source := filepath.Join(t.TempDir(), "config", "snapshots", filepath.Base(original))
					if err := os.CopyFS(source, os.DirFS(original)); err != nil {
						t.Fatal(err)
					}
					manifestPath := filepath.Join(source, "configuration.json")
					content, err := os.ReadFile(manifestPath)
					if err != nil {
						t.Fatal(err)
					}
					var manifest map[string]any
					if err := json.Unmarshal(content, &manifest); err != nil {
						t.Fatal(err)
					}
					fixture.edit(manifest)
					content, err = json.MarshalIndent(manifest, "", "  ")
					if err != nil {
						t.Fatal(err)
					}
					if err := os.Chmod(manifestPath, 0600); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(manifestPath, content, 0600); err != nil {
						t.Fatal(err)
					}
					actual, err := InspectBinding(source)
					if (err == nil) != fixture.valid {
						t.Fatalf("Go manifest acceptance differs from existing contract: want_valid=%t error=%v", fixture.valid, err)
					}
					bindingCheckNode(t, source, actual, fixture.valid)
				})
			}
		})
	}
}

func TestInspectBindingPreservesFornaxMappingAndUTF8Order(t *testing.T) {
	for _, variant := range []string{"utf8-release-order", "nonunique-workflow-mapping"} {
		t.Run(variant, func(t *testing.T) {
			source, _ := bindingConfiguration(t, "author")
			releases := bindingReleaseRefs()
			if variant == "utf8-release-order" {
				releases[0].Key, releases[1].Key = "skill.\U00010000", "skill.\uE000"
			} else {
				releases[1].WorkflowID = releases[0].WorkflowID
			}
			snapshot, err := PublishRelease(t.TempDir(), source, releases)
			if err != nil {
				t.Fatal(err)
			}
			manifest, err := InspectBinding(snapshot)
			if err != nil {
				t.Fatal(err)
			}
			bindingCheckNode(t, snapshot, manifest, true)
		})
	}
}
