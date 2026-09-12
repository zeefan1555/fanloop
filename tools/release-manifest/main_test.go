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
	wantSkills := []string{
		"fanloop-workflow",
		"fanloop-dev-agent-acceptance", "fanloop-dev-bootstrap", "fanloop-dev-code-review",
		"fanloop-dev-domain-modeling", "fanloop-dev-grill-with-docs", "fanloop-dev-grilling",
		"fanloop-dev-implement", "fanloop-dev-merge-code", "fanloop-dev-panorama", "fanloop-dev-tdd",
		"fanloop-dev-to-spec", "fanloop-dev-to-tickets", "fanloop-dev-update-local-cli", "fanloop-dev-workflow",
		"flashcard-card-planning", "flashcard-goal-framing", "flashcard-knowledge-selection",
		"flashcard-preview-approval", "flashcard-quality-review", "flashcard-source-understanding", "flashcard", "material-flashcards-panorama",
		"technical-background-framing", "technical-direction-approval", "technical-key-solutions",
		"technical-objective-setting", "technical-overall-solution", "technical-problem-analysis",
		"technical-problem-approval", "technical-solution-approval", "technical-solution-benefits",
		"technical-solution-delivery", "technical-solution-panorama", "technical-solution-research",
		"technical-solution-review", "technical-solution-writing",
	}
	gotSkills := make([]string, len(manifest.Skills))
	for index, skill := range manifest.Skills {
		gotSkills[index] = skill.Name
	}
	if !equalStrings(gotSkills, wantSkills) {
		t.Fatalf("Skills = %v, want %v", gotSkills, wantSkills)
	}
	gotWorkflows := make([]string, len(manifest.Workflows))
	for index, item := range manifest.Workflows {
		gotWorkflows[index] = item.Id
		if item.Sha256 == "" {
			t.Fatalf("Workflow is not pinned: %#v", item)
		}
	}
	if !equalStrings(gotWorkflows, []string{"fanloop-maintainer", "material-flashcards", "technical-solution-design"}) {
		t.Fatalf("Workflows = %v", gotWorkflows)
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
	workflowPath := filepath.Join(dist, "workflows", "technical-solution-design", "workflow.yaml")
	workflowContent, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(workflowContent), "name: 问题定义", "name: 不同的问题定义", 1)
	if changed == string(workflowContent) {
		t.Fatal("test did not change Workflow")
	}
	if err := os.WriteFile(workflowPath, []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := build("1.2.3", source, dist); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("valid but changed Workflow accepted: %v", err)
	}
}

func TestPanoramaSkillsOwnHostRoutingAndPresentationCommands(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "skills"))
	if err != nil {
		t.Fatal(err)
	}
	for _, relative := range []string{
		"fanloop-maintainer/fanloop-dev-panorama/SKILL.md",
		"material-flashcards/material-flashcards-panorama/SKILL.md",
		"technical-solution-design/technical-solution-panorama/SKILL.md",
	} {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		delivery := string(content)
		for _, value := range []string{
			"只依据系统或开发者上下文中已经声明的当前 Agent 人设",
			"Botmux Agent：`botmux`",
			"AIME Agent：`aime`",
			"Aiden Agent：`aiden`",
			"Codex、Claude Code 和 Trae：`local_agent`",
			"--format markdown",
			"--format lark-json",
			"botmux send --card-file",
			"lark-cli im +messages-reply",
			"aiden-bot-cli send-card --card-file",
			"本轮最终普通回复必须完整展示同一份 Panorama",
			"不自行拼装内容",
			`{"condition_id":"panorama_card_published","output":{"type":"path","value":"<data.snapshot_path>"}}`,
			"不得跨模式 fallback、双发、扫描旧快照",
		} {
			if !strings.Contains(delivery, value) {
				t.Fatalf("%s does not contain %q", relative, value)
			}
		}
		if relative == "fanloop-maintainer/fanloop-dev-panorama/SKILL.md" &&
			!strings.Contains(delivery, "选择 `agent_approved` Route 时不得渲染或发送") {
			t.Fatalf("%s does not preserve the maintainer Agent approval path", relative)
		}
		if relative != "fanloop-maintainer/fanloop-dev-panorama/SKILL.md" &&
			strings.Contains(delivery, "agent_approved") {
			t.Fatalf("%s documents the maintainer-only Agent approval path", relative)
		}
		for _, forbidden := range []string{"command -v", "BOTMUX_CHAT_ID:-", "BOTMUX_SESSION_ID:-", "<CURRENT_BOTMUX_SESSION_ID>"} {
			if strings.Contains(delivery, forbidden) {
				t.Fatalf("%s contains forbidden guidance %q", relative, forbidden)
			}
		}
	}
}

func TestDiscoverSkillsUsesWorkflowGroups(t *testing.T) {
	root := t.TempDir()
	for _, relative := range []string{
		"entrypoints/fanloop-workflow/SKILL.md",
		"skills/technical-solution-design/write/SKILL.md",
		"skills/fanloop-maintainer/maintain/SKILL.md",
	} {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(relative), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	skills, err := discoverSkills(root, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(skills))
	for index, skill := range skills {
		got[index] = skill.Name + "=" + skill.Path
	}
	want := []string{
		"fanloop-workflow=entrypoints/fanloop-workflow",
		"maintain=skills/fanloop-maintainer/maintain",
		"write=skills/technical-solution-design/write",
	}
	if !equalStrings(got, want) {
		t.Fatalf("discoverSkills() = %v, want %v", got, want)
	}
	nested := filepath.Join(root, "skills", "legacy", "nested", "skill", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(nested), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nested, []byte("legacy"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := discoverSkills(root, "1.2.3"); err == nil || !strings.Contains(err.Error(), "skills/<workflow-id>/<skill-id>/SKILL.md") {
		t.Fatalf("nested Skill layout error = %v", err)
	}
}

func TestValidateWorkflowSkillDirectoriesRequiresExactMatch(t *testing.T) {
	root := t.TempDir()
	for _, relative := range []string{"workflows/flow", "skills/flow"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(relative)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := validateWorkflowSkillDirectories(root); err != nil {
		t.Fatalf("matching directories rejected: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "workflows", "orphan"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := validateWorkflowSkillDirectories(root); err == nil || !strings.Contains(err.Error(), "orphan") {
		t.Fatalf("mismatched directories accepted: %v", err)
	}
}

func TestValidateWorkflowSkillBindingsEnforcesGroups(t *testing.T) {
	manifest := release.Manifest{
		Skills: []*release.Skill{
			{Name: release.ExposedSkillName, Path: release.ExposedSkillPath},
			{Name: "write", Path: "skills/technical-solution-design/write"},
			{Name: "maintain", Path: "skills/fanloop-maintainer/maintain"},
		},
		Workflows: []*release.Workflow{{Id: "technical-solution-design"}, {Id: "fanloop-maintainer"}},
	}
	loaded := []workflow.Loaded{
		{Workflow: workflow.Workflow{ID: "technical-solution-design", Prompts: map[string]workflow.PromptDefinition{"step": {Skills: []workflow.SkillBinding{{ID: "write"}}}}}},
		{Workflow: workflow.Workflow{ID: "fanloop-maintainer", Prompts: map[string]workflow.PromptDefinition{"step": {Skills: []workflow.SkillBinding{{ID: "maintain"}}}}}},
	}
	if err := validateWorkflowSkillBindings(manifest, loaded); err != nil {
		t.Fatalf("valid bindings rejected: %v", err)
	}
	loaded[0].Workflow.Prompts["step"] = workflow.PromptDefinition{Skills: []workflow.SkillBinding{{ID: "maintain"}}}
	if err := validateWorkflowSkillBindings(manifest, loaded); err == nil || !strings.Contains(err.Error(), "cannot use") {
		t.Fatalf("cross-Workflow binding error = %v", err)
	}
	loaded[0].Workflow.Prompts["step"] = workflow.PromptDefinition{Skills: []workflow.SkillBinding{{ID: "missing"}}}
	if err := validateWorkflowSkillBindings(manifest, loaded); err == nil || !strings.Contains(err.Error(), "unknown Skill") {
		t.Fatalf("missing binding error = %v", err)
	}
	manifest.Skills = append(manifest.Skills, &release.Skill{Name: "orphan", Path: "skills/orphan/orphan"})
	if err := validateWorkflowSkillBindings(manifest, loaded); err == nil || !strings.Contains(err.Error(), `unknown Workflow group "orphan"`) {
		t.Fatalf("unknown Workflow Skill group error = %v", err)
	}
	manifest.Skills = manifest.Skills[:len(manifest.Skills)-1]
	manifest.Workflows = append(manifest.Workflows, &release.Workflow{Id: "orphan"})
	if err := validateWorkflowSkillBindings(manifest, loaded); err == nil || !strings.Contains(err.Error(), "missing matching skills/orphan group") {
		t.Fatalf("missing Workflow Skill group error = %v", err)
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
		content, err := os.ReadFile(filepath.Join(repository, "workflows", "technical-solution-design", name))
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
	loaded, err := workflow.LoadDirectory(bundleRoot)
	if err != nil {
		t.Fatal(err)
	}
	entrypoint := filepath.Join(root, "entrypoints", release.ExposedSkillName, "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(entrypoint), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(entrypoint, []byte("---\nname: fanloop-workflow\n---\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	seenSkills := map[string]bool{}
	for _, prompt := range loaded.Workflow.Prompts {
		for _, binding := range prompt.Skills {
			seenSkills[binding.ID] = true
		}
	}
	for skillID := range seenSkills {
		path := filepath.Join(root, "skills", workflowID, skillID, "SKILL.md")
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
	if err := validateWorkflowSkillDirectories(root); err != nil {
		t.Fatal(err)
	}
	skills, err := discoverSkills(root, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	manifest := release.Manifest{
		Skills: skills,
		Workflows: []*release.Workflow{{
			Id: workflowID, Path: "workflows/" + workflowID, Sha256: loaded.Ref.Digest,
		}},
	}
	if err := validateWorkflowSkillBindings(manifest, []workflow.Loaded{loaded}); err != nil {
		t.Fatal(err)
	}
	if err := validateSelectorRoutes(filepath.Join(filepath.Dir(entrypoint), "routes.yaml"), manifest); err != nil {
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
	manifest := release.Manifest{Workflows: []*release.Workflow{{Id: "fanloop-maintainer"}, {Id: "material-flashcards"}, {Id: "technical-solution-design"}}}
	if err := validateSelectorRoutes(path, manifest); err != nil {
		t.Fatalf("production selector is invalid: %v", err)
	}
}

func TestValidateSelectorRejectsUnknownWorkflow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "routes.yaml")
	content := []byte("schema_version: 2\nscenarios:\n  missing:\n    workflow: missing\n    description: missing\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := release.Manifest{Workflows: []*release.Workflow{{Id: "technical-solution-design"}}}
	if err := validateSelectorRoutes(path, manifest); err == nil || !strings.Contains(err.Error(), `unknown Workflow "missing"`) {
		t.Fatalf("unknown selector target error = %v", err)
	}
}

func TestWorkflowEntryOwnsProtocolAndFinalPanorama(t *testing.T) {
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
		"flow status --root <ABSOLUTE_REQUIREMENT_ROOT>",
		"card render --root <ABSOLUTE_REQUIREMENT_ROOT> --view panorama --format markdown --dry-run",
		"`data.content`",
		"本轮最终普通回复必须完整展示同一份 Panorama",
		"不展示 JSON envelope，不自行拼装、压缩或重排内容",
		"任一命令失败即以真实错误阻塞并停止",
		"不得手工 fallback、复用旧 render 或快照",
	} {
		if !strings.Contains(skill, value) {
			t.Fatalf("fanloop-workflow Skill does not contain %q", value)
		}
	}
	if strings.Contains(skill, "fanloop update") {
		t.Fatal("local Workflow entry still requires an online update")
	}
}

func TestMaintainerEntryInitializesWithoutOnlineUpdate(t *testing.T) {
	path := filepath.Join("..", "..", "skills", "fanloop-maintainer", "fanloop-dev-workflow", "SKILL.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	skill := string(content)
	if strings.Contains(skill, "fanloop update") {
		t.Fatal("maintainer Workflow entry still requires an online update")
	}
	for _, value := range []string{
		"通用 `fanloop-workflow` 的 renderer-owned 最终回复契约",
		"flow status --root <ABSOLUTE_REQUIREMENT_ROOT>",
		"card render --root <ABSOLUTE_REQUIREMENT_ROOT> --view panorama --format markdown --dry-run",
		"本轮最终普通回复必须完整原样展示 render 响应的 `data.content`",
		"任一命令失败即以真实错误阻塞并停止",
		"不得手工 fallback、复用旧 render 或快照",
	} {
		if !strings.Contains(skill, value) {
			t.Fatalf("maintainer Workflow entry does not contain %q", value)
		}
	}
}

func TestDirectoryVerificationRejectsIncompleteOrChangedBuild(t *testing.T) {
	for _, test := range []struct {
		name, path, content, want string
		remove, symlink           bool
	}{
		{name: "missing binary", path: "bin/fanloop", remove: true, want: "bin/fanloop"},
		{name: "missing Skill", path: "skills/technical-solution-design/example/SKILL.md", remove: true, want: "checksum mismatch"},
		{name: "changed Skill", path: "skills/technical-solution-design/example/SKILL.md", content: "changed", want: "checksum mismatch"},
		{name: "missing Workflow", path: "workflows/technical-solution-design/flow.yaml", remove: true, want: "flow.yaml"},
		{name: "invalid Workflow", path: "workflows/technical-solution-design/flow.yaml", content: "invalid: true", want: "Workflow"},
		{name: "extra Workflow file", path: "workflows/technical-solution-design/guard.yaml", content: "extra", want: "guard.yaml"},
		{name: "unmanifested Workflow", path: "workflows/orphan/README.md", content: "extra", want: "workflows/orphan/README.md"},
		{name: "unmanifested Skill", path: "skills/orphan/SKILL.md", content: "extra", want: "skills/orphan/SKILL.md"},
		{name: "symlink", path: "skills/technical-solution-design/example/link.md", symlink: true, want: "unsupported entry"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeTestFile(t, root, "bin/fanloop", []byte("binary"))
			skillPath := "skills/technical-solution-design/example"
			writeTestFile(t, root, skillPath+"/SKILL.md", []byte("example Skill"))
			digest, err := release.DirectoryDigest(filepath.Join(root, skillPath))
			if err != nil {
				t.Fatal(err)
			}
			bundlePath := "workflows/technical-solution-design"
			for _, name := range workflow.BundleFileNames() {
				content, err := os.ReadFile(filepath.Join("..", "..", bundlePath, name))
				if err != nil {
					t.Fatal(err)
				}
				writeTestFile(t, root, bundlePath+"/"+name, content)
			}
			loaded, err := workflow.LoadDirectory(filepath.Join(root, bundlePath))
			if err != nil {
				t.Fatal(err)
			}
			manifest := release.Manifest{
				Skills:    []*release.Skill{{Name: "example", Path: skillPath, Sha256: digest}},
				Workflows: []*release.Workflow{{Id: loaded.Ref.ID, Path: bundlePath, Sha256: loaded.Ref.Digest}},
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
	for _, top := range []string{"entrypoints", "skills", "workflows"} {
		if err := filepath.WalkDir(filepath.Join(source, top), func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() {
				return walkErr
			}
			relative, err := filepath.Rel(source, path)
			if err != nil {
				return err
			}
			if top == "workflows" && strings.Count(filepath.ToSlash(relative), "/") != 2 {
				return nil
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
