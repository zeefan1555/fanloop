package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/release"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

func TestBuildCreatesMatchedLoopManifest(t *testing.T) {
	source, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dist := t.TempDir()
	writeTestReleaseDirectory(t, source, dist)
	manifest, err := build("1.2.3", source, dist)
	if err != nil {
		t.Fatal(err)
	}
	wantSkills := []string{"fanloop-workflow", "create-fanloop-workflow"}
	gotSkills := make([]string, len(manifest.Skills))
	for index, skill := range manifest.Skills {
		gotSkills[index] = skill.Name
	}
	if !equalStrings(gotSkills, wantSkills) {
		t.Fatalf("Skills = %v, want %v", gotSkills, wantSkills)
	}
	content, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := release.Decode(content); err != nil {
		t.Fatalf("local manifest rejected: %v", err)
	}
	digest, err := release.FileDigest(filepath.Join(dist, "bin", "fanloop"))
	if err != nil || manifest.Cli.BinarySha256 != digest {
		t.Fatalf("binary digest = %q, want %q: %v", manifest.Cli.BinarySha256, digest, err)
	}
	if strings.Contains(string(content), `"assets"`) {
		t.Fatal("local manifest contains distribution assets")
	}
	if strings.Contains(string(content), `"workflows"`) {
		t.Fatal("local manifest contains live Workflow artifacts")
	}
}

func TestDiscoverEntrypoints(t *testing.T) {
	root := t.TempDir()
	for _, exposed := range release.ExposedSkills() {
		sourcePath := exposed.Path
		if exposed.Name == release.AuthoringSkillName {
			sourcePath = ".agents/skills/" + exposed.Name
		}
		path := filepath.Join(root, filepath.FromSlash(sourcePath), "SKILL.md")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(exposed.Name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	skills, err := discoverEntrypoints(root, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 2 {
		t.Fatalf("discoverEntrypoints() = %#v", skills)
	}
	for index, exposed := range release.ExposedSkills() {
		skill := skills[index]
		if skill.Name != exposed.Name || skill.Path != exposed.Path || skill.Version != "1.2.3" || skill.Sha256 == "" {
			t.Fatalf("discoverEntrypoints()[%d] = %#v", index, skill)
		}
	}
}

func TestConfigOnlyWorkflowNeedsNoRuntimeRegistration(t *testing.T) {
	repository, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	workflowID := "sample-loop"
	bundleRoot := filepath.Join(root, "workflows", workflowID)
	if err := os.MkdirAll(bundleRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	example := filepath.Join(repository, "examples", "workflows", "example")
	if err := os.CopyFS(bundleRoot, os.DirFS(example)); err != nil {
		t.Fatal(err)
	}
	workflowPath := filepath.Join(bundleRoot, "workflow.yaml")
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatal(err)
	}
	content = []byte(strings.Replace(string(content), "id: example", "id: "+workflowID, 1))
	if err := os.WriteFile(workflowPath, content, 0o600); err != nil {
		t.Fatal(err)
	}
	entrypoint := filepath.Join(root, release.WorkflowSkillPath, "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(entrypoint), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(entrypoint, []byte("---\nname: fanloop-workflow\n---\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	authoring := filepath.Join(root, ".agents", "skills", release.AuthoringSkillName, "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(authoring), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(authoring, []byte("---\nname: create-fanloop-workflow\n---\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := workflow.LoadDirectory(bundleRoot)
	if err != nil {
		t.Fatal(err)
	}
	seenSkills := map[string]bool{}
	for _, binding := range loaded.Workflow.CommonSkills {
		seenSkills[binding.ID] = true
	}
	for _, prompt := range loaded.Workflow.Prompts {
		for _, binding := range prompt.Skills {
			seenSkills[binding.ID] = true
		}
	}
	for skillID := range seenSkills {
		path := filepath.Join(bundleRoot, "skills", skillID, "SKILL.md")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("---\nname: "+skillID+"\n---\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	routes := []byte("schema_version: 2\nscenarios:\n  sample:\n    workflow: " + workflowID + "\n    description: Sample config-only Loop\n")
	if err := os.WriteFile(filepath.Join(filepath.Dir(entrypoint), "routes.yaml"), routes, 0o600); err != nil {
		t.Fatal(err)
	}
	definitions, err := workflow.ListFrom(filepath.Join(root, "workflows"))
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != 1 || len(definitions[0].SkillPaths) == 0 {
		t.Fatal("config-only Workflow has no Skills")
	}
	exposed, err := discoverEntrypoints(root, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if exposed[0].Name != release.WorkflowSkillName {
		t.Fatalf("entrypoint = %#v", exposed)
	}
	if err := validateSelectorRoutes(filepath.Join(filepath.Dir(entrypoint), "routes.yaml")); err != nil {
		t.Fatal(err)
	}
}

func TestValidateSelectorAllowsFornaxManagedWorkflow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "routes.yaml")
	content := []byte("schema_version: 2\nscenarios:\n  missing:\n    workflow: missing\n    description: missing\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateSelectorRoutes(path); err != nil {
		t.Fatalf("Fornax-managed selector rejected: %v", err)
	}
}

func TestDirectoryVerificationRejectsIncompleteOrChangedBuild(t *testing.T) {
	for _, test := range []struct {
		name, path, content, want string
		remove, symlink           bool
	}{
		{name: "missing binary", path: "bin/fanloop", remove: true, want: "bin/fanloop"},
		{name: "missing entrypoint", path: "entrypoints/fanloop-workflow/SKILL.md", remove: true, want: "checksum mismatch"},
		{name: "changed entrypoint", path: "entrypoints/fanloop-workflow/SKILL.md", content: "changed", want: "checksum mismatch"},
		{name: "unmanifested Workflow", path: "workflows/orphan/README.md", content: "extra", want: "workflows/orphan/README.md"},
		{name: "unmanifested Skill", path: "skills/orphan/SKILL.md", content: "extra", want: "skills/orphan/SKILL.md"},
		{name: "symlink", path: "entrypoints/fanloop-workflow/link.md", symlink: true, want: "unsupported entry"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeTestFile(t, root, "bin/fanloop", []byte("binary"))
			skillPath := release.WorkflowSkillPath
			writeTestFile(t, root, skillPath+"/SKILL.md", []byte("example Skill"))
			digest, err := release.DirectoryDigest(filepath.Join(root, skillPath))
			if err != nil {
				t.Fatal(err)
			}
			manifest := release.Manifest{
				Skills: []*release.Skill{{Name: release.WorkflowSkillName, Path: skillPath, Sha256: digest}},
			}
			if _, err := verifyDirectory(root, manifest); err != nil {
				t.Fatalf("valid directory rejected: %v", err)
			}
			path := filepath.Join(root, test.path)
			switch {
			case test.remove:
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case test.symlink:
				if err := os.Symlink("SKILL.md", path); err != nil {
					t.Fatal(err)
				}
			default:
				writeTestFile(t, root, test.path, []byte(test.content))
			}
			if _, err := verifyDirectory(root, manifest); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("verification error = %v, want %q", err, test.want)
			}
		})
	}
}

func equalStrings(got, want []string) bool { return strings.Join(got, "|") == strings.Join(want, "|") }

func writeTestFile(t *testing.T, root, relative string, content []byte) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeTestReleaseDirectory(t *testing.T, source, destination string) {
	t.Helper()
	writeTestFile(t, destination, "bin/fanloop", []byte("test binary"))
	for _, top := range []string{"entrypoints"} {
		if err := filepath.WalkDir(filepath.Join(source, top), func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() {
				return walkErr
			}
			relative, err := filepath.Rel(source, path)
			if err != nil {
				return err
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			writeTestFile(t, destination, relative, content)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	authoringSource := filepath.Join(source, ".agents", "skills", release.AuthoringSkillName)
	authoringTarget := filepath.Join(destination, filepath.FromSlash(release.AuthoringSkillPath))
	if err := os.CopyFS(authoringTarget, os.DirFS(authoringSource)); err != nil {
		t.Fatal(err)
	}
}
