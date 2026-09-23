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

func TestBuildCreatesMatchedFanloopManifest(t *testing.T) {
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
	wantSkills := []string{"fanloop-workflow"}
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

func TestDiscoverEntrypoint(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, release.ExposedSkillPath, "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("entrypoint"), 0o600); err != nil {
		t.Fatal(err)
	}
	skill, err := discoverEntrypoint(root, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if skill.Name != release.ExposedSkillName || skill.Path != release.ExposedSkillPath || skill.Version != "1.2.3" || skill.Sha256 == "" {
		t.Fatalf("discoverEntrypoint() = %#v", skill)
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
	for _, name := range workflow.BundleFileNames() {
		content, err := os.ReadFile(filepath.Join(repository, "tests", "testdata", "workflows", "technical-solution-design", name))
		if err != nil {
			t.Fatal(err)
		}
		if name == "workflow.yaml" {
			content = []byte(strings.Replace(string(content), "id: technical-solution-design", "id: "+workflowID, 1))
		}
		if err := os.WriteFile(filepath.Join(bundleRoot, name), content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	entrypoint := filepath.Join(root, "entrypoints", release.ExposedSkillName, "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(entrypoint), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(entrypoint, []byte("---\nname: fanloop-workflow\n---\n"), 0o600); err != nil {
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
	exposed, err := discoverEntrypoint(root, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if exposed.Name != release.ExposedSkillName {
		t.Fatalf("entrypoint = %#v", exposed)
	}
	if err := validateSelectorRoutes(filepath.Join(filepath.Dir(entrypoint), "routes.yaml")); err != nil {
		t.Fatal(err)
	}
}

func TestProductionSelectorRequiresExplicitScenario(t *testing.T) {
	entrypoint, err := filepath.Abs(filepath.Join("..", "..", "entrypoints", "fanloop-workflow", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(entrypoint)
	if err != nil {
		t.Fatal(err)
	}
	skill := string(content)
	for _, value := range []string{"已初始化 State", "显式选择的场景", "不得初始化默认 Workflow", "routes.yaml"} {
		if !strings.Contains(skill, value) {
			t.Fatalf("Workflow entry does not contain %q", value)
		}
	}
	path := filepath.Join(filepath.Dir(entrypoint), "routes.yaml")

	if err := validateSelectorRoutes(path); err != nil {
		t.Fatalf("production selector is invalid: %v", err)
	}
}

func TestValidateSelectorRejectsEmptyWorkflow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "routes.yaml")
	content := []byte("schema_version: 2\nscenarios:\n  missing:\n    workflow: \n    description: missing\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateSelectorRoutes(path); err == nil || !strings.Contains(err.Error(), `invalid Workflow selector scenario rule`) {
		t.Fatalf("unknown selector target error = %v", err)
	}
}

func TestWorkflowEntryOwnsProtocolAndPanoramaFirst(t *testing.T) {
	path, err := filepath.Abs(filepath.Join("..", "..", "entrypoints", "fanloop-workflow", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	skill := string(content)
	for _, value := range []string{
		"flow status",
		"routes.yaml",
		"flow init",
		"`current.prompt`",
		"`available_routes`",
		"`panorama_presented` 或 `panorama_card_published`",
		"`data.content`",
		"Panorama 是进入新 Step 后第一条用户可见消息",
		"真正的人工问题只能随后发送",
		"最终回复不得重新 render、重复或改写它",
		"展示失败时按 Skill 上报 blocked",
	} {
		if !strings.Contains(skill, value) {
			t.Fatalf("fanloop-workflow Skill does not contain %q", value)
		}
	}
	if strings.Contains(skill, "fanloop update") {
		t.Fatal("local Workflow entry still requires an online update")
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
			skillPath := release.ExposedSkillPath
			writeTestFile(t, root, skillPath+"/SKILL.md", []byte("example Skill"))
			digest, err := release.DirectoryDigest(filepath.Join(root, skillPath))
			if err != nil {
				t.Fatal(err)
			}
			manifest := release.Manifest{
				Skills: []*release.Skill{{Name: release.ExposedSkillName, Path: skillPath, Sha256: digest}},
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
}
