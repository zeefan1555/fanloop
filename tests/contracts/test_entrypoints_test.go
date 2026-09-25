package contracts

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/idl"
)

func TestRepositoryHasOnePublicTestEntrypoint(t *testing.T) {
	repo := repositoryRoot(t)
	entrypoints := []string{"tests/run-unit"}
	actual, err := filepath.Glob(filepath.Join(repo, "tests/run-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(actual) != len(entrypoints) {
		t.Errorf("public test entrypoints = %v, want exactly %v", actual, entrypoints)
	}

	for _, relative := range entrypoints {
		info, err := os.Stat(filepath.Join(repo, relative))
		if err != nil {
			t.Errorf("%s: %v", relative, err)
			continue
		}
		if info.Mode()&0o111 == 0 {
			t.Errorf("%s is not executable", relative)
		}
	}
	ciRunner := filepath.Join(repo, ".github", "scripts", "run-requirement-validation")
	info, err := os.Stat(ciRunner)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		t.Fatalf("CI Requirement validation runner is not executable: %v", err)
	}
	content, err := os.ReadFile(ciRunner)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"TestRequirementWorkflowE2E", "route_matrix.py"} {
		if !strings.Contains(string(content), required) {
			t.Errorf("CI Requirement validation runner is missing %q", required)
		}
	}
	if strings.Contains(string(content), "verify-smoke") || strings.Contains(string(content), "verify smoke") {
		t.Error("CI Requirement validation runner still duplicates candidate verify smoke")
	}

	for _, relative := range []string{
		"scripts/test.sh",
		"scripts/run-workflow-demo-e2e",
		"scripts/run-route-matrix-e2e",
	} {
		_, err := os.Stat(filepath.Join(repo, relative))
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("retired test entrypoint remains: %s", relative)
		}
	}

	agents, err := os.ReadFile(filepath.Join(repo, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"./tests/run-unit"} {
		if !strings.Contains(string(agents), command) {
			t.Errorf("AGENTS.md does not require %s", command)
		}
	}
	for _, contract := range []string{"执行子 Agent 创建并持有 Requirement Root", "直接修改受管代码", "只向主 Agent回报", "不得自行批准", "自动 squash 合并唯一 PR", "只更新本 Requirement 的源码 worktree", "公开 npm 包"} {
		if !strings.Contains(string(agents), contract) {
			t.Errorf("AGENTS.md does not define maintainer contract %q", contract)
		}
	}
	if strings.Contains(string(agents), "./tests/run-e2e") {
		t.Error("AGENTS.md still requires the retired local E2E entrypoint")
	}
}

func TestNpmDistributionRetainsLocalBuildEntrypoints(t *testing.T) {
	repo := repositoryRoot(t)
	data, err := os.ReadFile(filepath.Join(repo, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pkg struct {
		Private bool              `json:"private"`
		Name    string            `json:"name"`
		Bin     map[string]string `json:"bin"`
		Scripts map[string]string `json:"scripts"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	if err := decoder.Decode(&pkg); err != nil {
		t.Fatal(err)
	}
	if pkg.Private || pkg.Name != "@zeefan1555/fanloop-cli" || pkg.Bin["fanloop"] != "scripts/run.js" ||
		pkg.Scripts["build"] != "./scripts/build-local.sh" ||
		pkg.Scripts["install:local"] != "./scripts/install-local.sh" ||
		pkg.Scripts["build:release"] != "./scripts/build-npm-release.sh" {
		t.Fatal("npm distribution must retain local build scripts and expose the official launcher")
	}
	for _, relative := range []string{"scripts/build-local.sh", "scripts/install-local.sh", "scripts/build-npm-release.sh"} {
		info, err := os.Stat(filepath.Join(repo, filepath.FromSlash(relative)))
		if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
			t.Fatalf("local entrypoint %s is not executable: %v", relative, err)
		}
	}
	for _, relative := range []string{
		".goreleaser.yml",
		"scripts/build-release.sh", "scripts/prepare-release.sh", "scripts/package-release.sh",
		"scripts/verify-published-package.sh",
	} {
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(relative))); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("retired distribution entrypoint remains: %s (%v)", relative, err)
		}
	}
}

func TestProductionConfigurationIsNotARepositoryDefault(t *testing.T) {
	repo := repositoryRoot(t)
	if _, err := os.Stat(filepath.Join(repo, "workflows")); !os.IsNotExist(err) {
		t.Fatal("production workflows must live in Fornax")
	}
	for _, name := range []string{"entrypoints/fanloop-workflow/SKILL.md", ".cursor/skills/verify-fanloop/SKILL.md", "scripts/pin-controller-release.sh"} {
		if _, err := os.Stat(filepath.Join(repo, name)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestVerificationFeatureMapIsNavigable(t *testing.T) {
	repo := repositoryRoot(t)
	root := filepath.Join(repo, ".cursor", "skills", "verify-fanloop", "features")
	index, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}

	links := regexp.MustCompile(`\[[^\]]+\]\(([^)]+\.md)\)`).FindAllStringSubmatch(string(index), -1)
	if len(links) == 0 {
		t.Fatal("Feature Map index has no feature links")
	}
	wanted := map[string]bool{"README.md": true}
	catalog := string(index)
	for _, match := range links {
		name := match[1]
		if filepath.Base(name) != name || wanted[name] {
			t.Fatalf("Feature Map has invalid or duplicate link %q", name)
		}
		wanted[name] = true
		content, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatalf("read Feature Map entry %s: %v", name, err)
		}
		catalog += "\n" + string(content)
		for _, heading := range []string{
			"## Sub-features",
			"## How to get to it (user POV)",
			"## Driving it with fanloop",
			"## Gotchas",
		} {
			if strings.Count(string(content), heading) != 1 {
				t.Errorf("Feature Map entry %s must contain exactly one %q", name, heading)
			}
		}
	}

	files, err := filepath.Glob(filepath.Join(root, "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != len(wanted) {
		t.Fatalf("Feature Map files = %d, indexed files = %d", len(files), len(wanted))
	}
	for _, file := range files {
		if !wanted[filepath.Base(file)] {
			t.Errorf("Feature Map file is not indexed: %s", filepath.Base(file))
		}
	}

	for _, spec := range idl.CommandSpecs() {
		command := strings.ReplaceAll(spec.ID, ".", " ")
		if !strings.Contains(catalog, command) {
			t.Errorf("Feature Map does not cover public command %q", spec.ID)
		}
	}
	workflowFiles, err := filepath.Glob(filepath.Join(repo, "workflows", "*", "workflow.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range workflowFiles {
		workflowID := filepath.Base(filepath.Dir(file))
		if !strings.Contains(catalog, workflowID) {
			t.Errorf("Feature Map does not cover production Workflow %q", workflowID)
		}
	}
}

func TestAgentNavigationPointsAtThePavedVerifyPath(t *testing.T) {
	repo := repositoryRoot(t)
	read := func(relative string) string {
		t.Helper()
		content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		return string(content)
	}

	const skill = ".cursor/skills/verify-fanloop/SKILL.md"
	agents := read("AGENTS.md")
	ownership := read("docs/ownership.md")
	readme := read("README.md")
	design := read("docs/technical-design.md")
	glossary := read("CONTEXT.md")
	entry := read(skill)

	for _, file := range []struct{ name, text string }{
		{"AGENTS.md", agents},
		{"docs/ownership.md", ownership},
		{"README.md", readme},
		{"docs/technical-design.md", design},
	} {
		if !strings.Contains(file.text, skill) {
			t.Errorf("%s missing %s", file.name, skill)
		}
	}
	if !strings.Contains(agents, "docs/ownership.md") {
		t.Error("AGENTS.md does not point at docs/ownership.md")
	}
	if !strings.Contains(glossary, ".cursor/skills/verify-fanloop/features/") {
		t.Error("CONTEXT.md does not point at the Feature Map")
	}
	for _, suite := range []string{
		"fanloop-maintainer",
		"remnote-flashcards",
		"pstack-native",
	} {
		if !strings.Contains(ownership, suite) {
			t.Errorf("docs/ownership.md missing production suite %s", suite)
		}
	}
	for _, anchor := range []string{
		"./scripts/install-local.sh",
		"features/README.md",
		"post_merge_smoke_passed",
		"verify.evidence.outcome",
		"cmd/verify.go",
		"internal/verify",
	} {
		if !strings.Contains(entry, anchor) && !strings.Contains(ownership, anchor) {
			t.Errorf("paved path missing %s", anchor)
		}
	}
	if strings.Contains(readme, "fanloop verify snapshot --root") {
		t.Error("README.md still lists a second verify command recipe")
	}
}
