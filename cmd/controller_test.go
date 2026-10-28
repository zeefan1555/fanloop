package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/zeefan1555/fanloop/internal/controller"
	"github.com/zeefan1555/fanloop/internal/release"
)

func TestControllerSharedResourceFixtures(t *testing.T) {
	content, err := os.ReadFile("../internal/controller/testdata/resources.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct {
		Name  string `json:"name"`
		Valid bool   `json:"valid"`
	}
	if err := json.Unmarshal(content, &fixtures); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(newFlowRuntime().ControllerRelease, "bin", "fanloop")
	launcher, err := filepath.Abs("../packaging/npm/scripts/run.js")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("FANLOOP_NO_AUTO_UPDATE", "1")
	t.Setenv("FANLOOP_NO_AUTO_CONFIG", "1")
	roots := map[string]string{}
	fixtureRoot := t.TempDir()
	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			root := filepath.Join(fixtureRoot, fixture.Name)
			if err := os.Mkdir(root, 0700); err != nil {
				t.Fatal(err)
			}
			roots[fixture.Name] = root
			command := exec.Command(binary, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", fixture.Name)
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("real fixture initialization: %v\n%s", err, output)
			}
			home := filepath.Join(root, controller.Home)
			physical := filepath.Join(home, "releases", "test")
			current := filepath.Join(home, "current")
			replaceLink := func(link, target string) {
				t.Helper()
				if err := os.Remove(link); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, link); err != nil {
					t.Fatal(err)
				}
			}
			switch fixture.Name {
			case "symlink-current", "symlink-current-absolute", "normalized-symlink-current", "recursive-current", "ordinary-current":
				if err := os.Rename(current, current+".preserved"); err != nil {
					t.Fatal(err)
				}
				if fixture.Name == "ordinary-current" {
					err = os.WriteFile(current, []byte("not a current link"), 0600)
				} else {
					target := "releases/test"
					if fixture.Name == "symlink-current-absolute" {
						target = physical
					}
					if fixture.Name == "normalized-symlink-current" {
						target = "missing/../releases/test"
					}
					if fixture.Name == "recursive-current" {
						target = "current"
					}
					err = os.Symlink(target, current)
				}
			case "legacy-controller":
				err = os.Rename(filepath.Join(root, controller.Journal), filepath.Join(root, controller.Journal)+".preserved")
			case "binary-drift", "entrypoint-drift":
				file := filepath.Join(physical, "bin", "fanloop")
				if fixture.Name == "entrypoint-drift" {
					file = filepath.Join(physical, "entrypoints", "fanloop-workflow", "SKILL.md")
				}
				stream, failure := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0)
				if failure != nil {
					t.Fatal(failure)
				}
				_, err = stream.WriteString("external drift")
				if closeErr := stream.Close(); err == nil {
					err = closeErr
				}
			case "extra-resource-link":
				err = os.Symlink(filepath.Join(physical, "bin", "fanloop"), filepath.Join(physical, "entrypoints", "extra-link"))
			case "resource-fifo":
				err = syscall.Mkfifo(filepath.Join(physical, "entrypoints", "extra-fifo"), 0600)
			case "linked-release-ancestor":
				directory := filepath.Join(home, "releases")
				if err := os.Rename(directory, directory+".preserved"); err != nil {
					t.Fatal(err)
				}
				err = os.Symlink(directory+".preserved", directory)
			case "escaped-current":
				replaceLink(filepath.Join(current, "release"), newFlowRuntime().ControllerRelease)
			case "directory-current-absolute":
				replaceLink(filepath.Join(current, "release"), physical)
			case "normalized-directory-current":
				replaceLink(filepath.Join(current, "release"), "missing/../../releases/test")
			case "normalized-current-alias":
				replaceLink(filepath.Join(current, "bin"), "missing/../release/bin")
			case "unmanaged-current-file":
				err = os.WriteFile(filepath.Join(current, "preserve-user-file"), []byte("preserve\n"), 0600)
			case "unmanaged-current-directory":
				err = os.Mkdir(filepath.Join(current, "preserve-user-directory"), 0700)
			case "ordinary-skill":
				file := filepath.Join(home, "skill-roots", "codex", "fanloop-workflow")
				if err := os.Remove(file); err != nil {
					t.Fatal(err)
				}
				err = os.WriteFile(file, []byte("ordinary conflict"), 0600)
			}
			if err != nil {
				t.Fatal(err)
			}
			business := map[string][]byte{}
			journal := controller.Journal
			if fixture.Name == "legacy-controller" {
				journal += ".preserved"
			}
			for _, file := range []string{".fanloop/flow/state.json", ".fanloop/output/state.json", ".fanloop/trace/events.jsonl", ".fanloop/configuration-snapshot", journal} {
				content, err := os.ReadFile(filepath.Join(root, file))
				if err != nil {
					t.Fatal(err)
				}
				business[file] = content
			}
			args := []string{"flow", "status", "--root", root}
			selection, failure := ControllerSelection(args)
			if fixture.Valid && (failure != nil || selection == nil) || !fixture.Valid && failure == nil {
				t.Fatalf("Go selection %s: selection_present=%t error=%v", fixture.Name, selection != nil, failure)
			}
			for _, surface := range []string{"direct", "node"} {
				command := exec.Command(binary, args...)
				if surface == "node" {
					command = exec.Command("node", append([]string{launcher}, args...)...)
				}
				output, failure := command.CombinedOutput()
				if fixture.Valid {
					if failure != nil || !strings.Contains(string(output), fixture.Name) {
						t.Fatalf("%s healthy %s: %v\n%s", surface, fixture.Name, failure, output)
					}
				} else if exit, ok := failure.(*exec.ExitError); !ok || exit.ExitCode() != 5 {
					t.Fatalf("%s accepted invalid %s: %v\n%s", surface, fixture.Name, failure, output)
				}
				for file, original := range business {
					content, err := os.ReadFile(filepath.Join(root, file))
					if err != nil || !bytes.Equal(content, original) {
						t.Fatalf("%s fixture altered %s: %v", surface, file, err)
					}
				}
				if fixture.Name == "unmanaged-current-file" {
					if content, err := os.ReadFile(filepath.Join(current, "preserve-user-file")); err != nil || string(content) != "preserve\n" {
						t.Fatalf("%s changed the unknown current file: %v", surface, err)
					}
				}
				if fixture.Name == "unmanaged-current-directory" {
					if info, err := os.Lstat(filepath.Join(current, "preserve-user-directory")); err != nil || !info.IsDir() {
						t.Fatalf("%s changed the unknown current directory: %v", surface, err)
					}
				}
			}
		})
	}
	index, err := json.Marshal(roots)
	if err != nil {
		t.Fatal(err)
	}
	indexPath := filepath.Join(t.TempDir(), "roots.json")
	if err := os.WriteFile(indexPath, index, 0600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("node", "../packaging/npm/controller-fixtures.js", indexPath)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("Node shared selection fixtures: %v\n%s", err, output)
	}
}

func TestControllerSharedJournalTypes(t *testing.T) {
	content, err := os.ReadFile("../internal/controller/testdata/journal-types.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Binding                map[string]json.RawMessage `json:"binding"`
		RequiredStrings        []string                   `json:"required_strings"`
		OptionalStrings        []string                   `json:"optional_strings"`
		WrongStringValues      []json.RawMessage          `json:"wrong_string_values"`
		WrongSchemaValues      []json.RawMessage          `json:"wrong_schema_values"`
		UnknownFields          []string                   `json:"unknown_fields"`
		InvalidResourceDigests []json.RawMessage          `json:"invalid_resource_digests"`
		ValidTimestamps        []json.RawMessage          `json:"valid_timestamps"`
		InvalidTimestamps      []json.RawMessage          `json:"invalid_timestamps"`
	}
	if err := json.Unmarshal(content, &fixture); err != nil {
		t.Fatal(err)
	}
	base, err := json.Marshal(fixture.Binding)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := controller.Decode(base); err != nil {
		t.Fatalf("shared healthy journal: %v", err)
	}
	dataRoot, author := t.TempDir(), t.TempDir()
	configCopyTree(t, "../internal/workflow/testdata/linear-runtime", filepath.Join(author, "fanloop-workflow", "linear-runtime"))
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_CONFIG_SOURCE", author)
	t.Setenv("FANLOOP_NO_AUTO_UPDATE", "1")
	t.Setenv("FANLOOP_NO_AUTO_CONFIG", "1")
	root := t.TempDir()
	binary := filepath.Join(newFlowRuntime().ControllerRelease, "bin", "fanloop")
	launcher, err := filepath.Abs("../packaging/npm/scripts/run.js")
	if err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(binary, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Journal types").CombinedOutput(); err != nil {
		t.Fatalf("real journal initialization: %v\n%s", err, output)
	}
	if output, err := exec.Command(binary, "__protect", root, "--source", newFlowRuntime().ControllerRelease).CombinedOutput(); err != nil {
		t.Fatalf("real journal freeze and archive: %v\n%s", err, output)
	}
	committed, err := controller.Read(root)
	if err != nil || committed.GetPreviousBindingTransactionId() == "" {
		t.Fatalf("missing committed archive: %v %v", committed, err)
	}
	journalPath := filepath.Join(root, controller.Journal)
	original, err := os.ReadFile(journalPath)
	if err != nil {
		t.Fatal(err)
	}
	var binding map[string]json.RawMessage
	if err := json.Unmarshal(original, &binding); err != nil {
		t.Fatal(err)
	}
	facts := map[string][]byte{}
	for _, file := range []string{".fanloop/flow/state.json", ".fanloop/output/state.json", ".fanloop/trace/events.jsonl", ".fanloop/configuration-snapshot", ".fanloop/audit/controller-bindings/" + committed.GetPreviousBindingTransactionId() + ".json"} {
		content, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			t.Fatal(err)
		}
		facts[file] = content
	}
	args := []string{"flow", "status", "--root", root}
	if selection, err := ControllerSelection(args); err != nil || selection == nil {
		t.Fatalf("healthy Go selection: selection_present=%t error=%v", selection != nil, err)
	}
	fields := append(append(append([]string{}, fixture.RequiredStrings...), fixture.OptionalStrings...), "schema_version")
	fields = append(fields, fixture.UnknownFields...)
	for _, field := range fields {
		_, known := fixture.Binding[field]
		values := fixture.WrongSchemaValues
		if !known {
			values = []json.RawMessage{json.RawMessage("1")}
		} else if field != "schema_version" {
			value := binding[field]
			if value == nil {
				value = fixture.Binding[field]
			}
			values = append(append([]json.RawMessage{}, fixture.WrongStringValues...), json.RawMessage("["+string(value)+"]"))
			if field == "release_resources_sha256" {
				values = append(append(values, nil), fixture.InvalidResourceDigests...)
			}
			if field == "prepared_at" || field == "committed_at" {
				values = append(values, fixture.InvalidTimestamps...)
			}
		}
		for index, value := range values {
			t.Run(fmt.Sprintf("%s/%d", field, index), func(t *testing.T) {
				previous, present := binding[field]
				if value == nil {
					delete(binding, field)
				} else {
					binding[field] = value
				}
				content, err := json.Marshal(binding)
				if present {
					binding[field] = previous
				} else {
					delete(binding, field)
				}
				if err != nil {
					t.Fatal(err)
				}
				if _, err := controller.Decode(content); err == nil {
					t.Errorf("Go decode accepted %s=%s", field, value)
				}
				if err := os.WriteFile(journalPath, content, 0600); err != nil {
					t.Fatal(err)
				}
				if selection, err := ControllerSelection(args); err == nil || selection != nil {
					t.Errorf("Go selection accepted %s=%s: selection_present=%t error=%v", field, value, selection != nil, err)
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
						t.Errorf("%s accepted %s=%s: %v\nstdout=%s\nstderr=%s", surface, field, value, failure, stdout.String(), stderr.String())
					}
					if surface == "node" && !known && !strings.Contains(stderr.String(), "invalid controller journal fields") {
						t.Fatalf("Node rejected unknown %s for a different reason: %s", field, stderr.String())
					}
					if surface == "node" && known && field != "schema_version" && (value == nil || value[0] != '"') && !strings.Contains(stderr.String(), "field must be a string: "+field) {
						t.Fatalf("Node rejected %s for a different reason: %s", field, stderr.String())
					}
					for file, original := range facts {
						current, err := os.ReadFile(filepath.Join(root, file))
						if err != nil || !bytes.Equal(current, original) {
							t.Fatalf("%s rejection altered %s: %v", surface, file, err)
						}
					}
					current, err := os.ReadFile(journalPath)
					if err != nil || !bytes.Equal(current, content) {
						t.Fatalf("%s rejection altered the journal: %v", surface, err)
					}
				}
			})
		}
	}
	for _, field := range []string{"prepared_at", "committed_at"} {
		previous := binding[field]
		for _, value := range fixture.ValidTimestamps {
			binding[field] = value
			content, err := json.Marshal(binding)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := controller.Decode(content); err != nil {
				t.Fatalf("Go rejected valid %s=%s: %v", field, value, err)
			}
			if err := os.WriteFile(journalPath, content, 0600); err != nil {
				t.Fatal(err)
			}
			for _, command := range []*exec.Cmd{exec.Command(binary, args...), exec.Command("node", append([]string{launcher}, args...)...)} {
				if output, err := command.CombinedOutput(); err != nil || !strings.Contains(string(output), "Journal types") {
					t.Fatalf("valid timestamp through actual entrypoint: %v\n%s", err, output)
				}
				for file, original := range facts {
					current, err := os.ReadFile(filepath.Join(root, file))
					if err != nil || !bytes.Equal(current, original) {
						t.Fatalf("valid timestamp altered %s: %v", file, err)
					}
				}
			}
		}
		binding[field] = previous
	}
	if err := os.WriteFile(journalPath, original, 0600); err != nil {
		t.Fatal(err)
	}
	for _, command := range []*exec.Cmd{exec.Command(binary, args...), exec.Command("node", append([]string{launcher}, args...)...)} {
		if output, err := command.CombinedOutput(); err != nil || !strings.Contains(string(output), "Journal types") {
			t.Fatalf("healthy journal after rejection matrix: %v\n%s", err, output)
		}
	}
}

func TestControllerResourceIdentityMatchesNode(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range []struct {
		name, content string
		mode          os.FileMode
	}{{"ordinary", "plain\n", 0644}, {string(rune(0xe000)), "private\n", 0755}, {"\U00010000", "astral\n", 0600}} {
		path := filepath.Join(root, file.name)
		if err := os.WriteFile(path, []byte(file.content), file.mode); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, file.mode); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(root, "empty"), 0700); err != nil {
		t.Fatal(err)
	}
	layout, err := filepath.Abs("../packaging/npm/scripts/layout.js")
	if err != nil {
		t.Fatal(err)
	}
	identity := func(t *testing.T) (string, string) {
		t.Helper()
		directory, modes, err := release.StrictDirectoryIdentity(root)
		if err != nil {
			t.Fatal(err)
		}
		goDigest := release.ControllerResourcesDigest(directory, modes)
		command := exec.Command("node", "-e", "process.stdout.write(require(process.argv[1]).controllerResourcesDigest(process.argv[2]))", layout, root)
		nodeDigest, err := command.CombinedOutput()
		if err != nil || string(nodeDigest) != goDigest {
			t.Fatalf("resource identity differs: Go=%s Node=%s error=%v", goDigest, nodeDigest, err)
		}
		return goDigest, directory
	}
	before, directory := identity(t)
	if before != "sha256:109f78039cfbe5b2781671c668441bff0a7ab4f672e482c5610902cf05b2e130" || directory != "sha256:ed73215cf466fc08cec0405792ab29c1438d379d88df8262ce4fc346def58985" {
		t.Fatalf("canonical UTF-8 resource vector differs: %s %s", before, directory)
	}
	for _, change := range []struct {
		name    string
		changed bool
		mutate  func() error
	}{
		{"bytes", true, func() error { return os.WriteFile(filepath.Join(root, "ordinary"), []byte("changed\n"), 0644) }},
		{"file-mode", true, func() error { return os.Chmod(filepath.Join(root, "ordinary"), 0775) }},
		{"add-file", true, func() error { return os.WriteFile(filepath.Join(root, "added"), []byte("added\n"), 0600) }},
		{"remove-file", true, func() error { return os.Remove(filepath.Join(root, "added")) }},
		{"add-empty-directory", true, func() error { return os.Mkdir(filepath.Join(root, "extra-empty"), 0700) }},
		{"remove-empty-directory", true, func() error { return os.Remove(filepath.Join(root, "extra-empty")) }},
		{"directory-mode", false, func() error { return os.Chmod(filepath.Join(root, "empty"), 0755) }},
		{"root-mode", false, func() error { return os.Chmod(root, 0755) }},
	} {
		t.Run(change.name, func(t *testing.T) {
			previousDirectory := directory
			if err := change.mutate(); err != nil {
				t.Fatal(err)
			}
			after, actualDirectory := identity(t)
			if (after != before) != change.changed {
				t.Fatalf("resource identity change=%t, expected %t", after != before, change.changed)
			}
			if change.name == "file-mode" && actualDirectory != previousDirectory {
				t.Fatal("DirectoryDigest unexpectedly includes file mode")
			}
			before, directory = after, actualDirectory
		})
	}
}

func TestControllerArgvMatchesCobra(t *testing.T) {
	content, err := os.ReadFile("../internal/controller/testdata/argv.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Args   []string `json:"args"`
		Root   string   `json:"root"`
		Rebind bool     `json:"rebind"`
		DryRun *bool    `json:"dry_run"`
	}
	if err := json.Unmarshal(content, &cases); err != nil {
		t.Fatal(err)
	}
	for _, test := range cases {
		root, rebind := controllerInvocation(test.Args)
		if root != test.Root || rebind != test.Rebind {
			t.Fatalf("%q -> root=%q rebind=%t; wanted %q %t", test.Args, root, rebind, test.Root, test.Rebind)
		}
		if test.DryRun != nil {
			commandRoot := NewRoot(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
			command, remaining, err := commandRoot.Find(test.Args)
			if err != nil {
				t.Fatal(err)
			}
			if err := command.ParseFlags(remaining); err != nil {
				t.Fatal(err)
			}
			if boolFlag(command, "dry-run") != *test.DryRun {
				t.Fatalf("%q: Cobra dry-run differs", test.Args)
			}
		}
	}
}
