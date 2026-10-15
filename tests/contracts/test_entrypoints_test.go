package contracts

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/idl"
	"go.yaml.in/yaml/v3"
)

func TestCIClassificationFailsClosed(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(repositoryRoot(t), ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		Jobs map[string]struct {
			Needs string `yaml:"needs"`
			If    string `yaml:"if"`
			Steps []struct {
				Env map[string]string `yaml:"env"`
				Run string            `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"test", "requirement-e2e", "install-doctor", "governance"} {
		t.Run(name, func(t *testing.T) {
			job, ok := workflow.Jobs[name]
			if !ok || job.Needs != "changes" {
				t.Fatal("existing CI job must depend on classification")
			}
			if job.If != "always() && (needs.changes.result != 'success' || needs.changes.outputs.code != 'false')" {
				t.Fatal("only successful explicit configuration classification may skip an existing check")
			}
			if len(job.Steps) == 0 {
				t.Fatal("missing failure-propagation gate")
			}
			gate := job.Steps[0]
			if gate.Env["CLASSIFICATION_RESULT"] != "${{ needs.changes.result }}" || gate.Env["CODE_CHANGED"] != "${{ needs.changes.outputs.code }}" || gate.Run == "" {
				t.Fatal("first step must gate engine work on the actual classification result and output")
			}
			for _, test := range []struct {
				result, code string
				pass         bool
			}{
				{"success", "true", true},
				{"success", "false", false},
				{"failure", "false", false},
				{"failure", "true", false},
				{"cancelled", "", false},
				{"skipped", "", false},
				{"success", "", false},
				{"success", "invalid", false},
			} {
				command := exec.Command("bash", "-c", gate.Run)
				command.Env = append(os.Environ(), "CLASSIFICATION_RESULT="+test.result, "CODE_CHANGED="+test.code)
				output, err := command.CombinedOutput()
				if (err == nil) != test.pass {
					t.Errorf("gate(%q, %q) pass=%v, want %v: %s", test.result, test.code, err == nil, test.pass, output)
				}
			}
		})
	}
}

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
	for _, contract := range []string{"执行子 Agent 创建并持有 Requirement Root", "直接修改受管代码", "只向主 Agent回报", "不得自行批准", "自动 squash 合并唯一 PR", "只更新本 Requirement 的源码 worktree", "本地安装"} {
		if !strings.Contains(string(agents), contract) {
			t.Errorf("AGENTS.md does not define maintainer contract %q", contract)
		}
	}
	if strings.Contains(string(agents), "./tests/run-e2e") {
		t.Error("AGENTS.md still requires the retired local E2E entrypoint")
	}

	runUnit, err := os.ReadFile(filepath.Join(repo, "tests", "run-unit"))
	if err != nil {
		t.Fatal(err)
	}
	for _, contract := range []string{
		"export GOTOOLCHAIN=auto",
		"run_check vet",
		"run_check go-test",
		"run_check route-matrix",
		"wait_for_checks",
	} {
		if !strings.Contains(string(runUnit), contract) {
			t.Errorf("tests/run-unit is missing parallel gate contract %q", contract)
		}
	}
}

func TestSelfIterationUsesOneFrozenCandidate(t *testing.T) {
	repo := repositoryRoot(t)
	read := func(relative string) string {
		t.Helper()
		content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		return string(content)
	}

	catalog := read("AGENTS.md") + "\n" + read("docs/outer-loop.md") + "\n" + read(".cursor/skills/verify-fanloop/SKILL.md")
	for _, contract := range []string{
		"low|medium|high",
		"Pre-Certify",
		"小批",
		"并行认证",
		"一次性交付",
		"返工来源",
		"candidate identity",
		"configuration_manifest_sha256",
	} {
		if !strings.Contains(catalog, contract) {
			t.Errorf("self-iteration guidance is missing %q", contract)
		}
	}
	technicalDesign := read("docs/technical-design.md")
	if strings.Contains(technicalDesign, "Certify 先执行独立 Review，确认身份未变后再执行黑盒验收") {
		t.Error("technical design retains the retired serial certification rule")
	}
	agents := read("AGENTS.md")
	if strings.Contains(agents, "审查通过后冻结 `candidate_head`，再由一个无实现上下文的全新 Sub-agent") {
		t.Error("AGENTS.md retains the retired serial certification rule")
	}
}

func TestRunUnitWaitsForAllChecksAndAggregatesFailure(t *testing.T) {
	repo := repositoryRoot(t)
	marker := filepath.Join(t.TempDir(), "delayed-success-finished")
	helper := filepath.Join(repo, "tests", "lib", "check-group.sh")
	logRoot := t.TempDir()
	script := `
set -u
source "$1"
check_log_root="$2"
run_check delayed-success bash -c 'sleep 0.2; printf "delayed body\n"; : > "$1"' _ "$3"
run_check fast-failure bash -c 'printf "failure body\n"; exit 7'
set +e
wait_for_checks
result=$?
set -e
test -f "$3"
exit "$result"
`
	command := exec.Command("bash", "-c", script, "run-unit-check-group", helper, logRoot, marker)
	output, err := command.CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("wait_for_checks error = %v, output:\n%s", err, output)
	}
	for _, want := range []string{
		"[START] delayed-success",
		"[START] fast-failure",
		"[PASS] delayed-success",
		"[FAIL] fast-failure",
		"delayed body",
		"failure body",
	} {
		if !strings.Contains(string(output), want) {
			t.Errorf("output missing %q:\n%s", want, output)
		}
	}
}

func TestGitHubReleaseDistribution(t *testing.T) {
	repo := repositoryRoot(t)
	data, err := os.ReadFile(filepath.Join(repo, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pkg struct {
		Private bool              `json:"private"`
		Scripts map[string]string `json:"scripts"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	if err := decoder.Decode(&pkg); err != nil {
		t.Fatal(err)
	}
	if !pkg.Private || len(pkg.Scripts) != 2 || pkg.Scripts["build"] != "./scripts/build-local.sh" || pkg.Scripts["install:local"] != "./scripts/install-local.sh" {
		t.Fatal("package.json must expose only the local build and install shortcuts")
	}
	for _, relative := range []string{"scripts/build-local.sh", "scripts/install-local.sh", "scripts/build-release.sh"} {
		info, err := os.Stat(filepath.Join(repo, filepath.FromSlash(relative)))
		if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
			t.Fatalf("entrypoint %s is not executable: %v", relative, err)
		}
	}
	for _, relative := range []string{
		".github/workflows/release.yml",
		"scripts/verify-release-assets.js",
		"packaging/npm/scripts/run.js",
		"packaging/npm/scripts/install.js",
		"docs/adr/0136-distribute-cli-from-github-releases.md",
	} {
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(relative))); err != nil {
			t.Errorf("distribution file missing: %s (%v)", relative, err)
		}
	}
	workflow, err := os.ReadFile(filepath.Join(repo, ".github/workflows/release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(workflow), "ubuntu-latest") || strings.Contains(string(workflow), "macos-") {
		t.Fatalf("release workflow must cross-compile on ubuntu runners only:\n%s", workflow)
	}
	if !strings.Contains(string(workflow), "v[0-9]+.[0-9]+.[0-9]+") || strings.Contains(string(workflow), `"v*"`) {
		t.Fatalf("release workflow tag filter must match only vX.Y.Z:\n%s", workflow)
	}
	nodeTests := "node --test packaging/npm/install.test.js scripts/verify-release-assets.test.js"
	if !strings.Contains(string(workflow), nodeTests) {
		t.Fatal("release workflow does not run the Node distribution tests")
	}
	ci, err := os.ReadFile(filepath.Join(repo, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ci), nodeTests) {
		t.Fatal("CI does not run the Node distribution tests")
	}
	readme, err := os.ReadFile(filepath.Join(repo, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"--allow-scripts=fanloop", "--foreground-scripts", "go 1.23.0"} {
		if !strings.Contains(string(readme), required) {
			t.Errorf("README is missing %q", required)
		}
	}
	for _, relative := range []string{
		".goreleaser.yml", "scripts/prepare-release.sh", "scripts/build-npm-release.sh",
		"scripts/install.js", "scripts/package-release.js", "scripts/publish-npm-package.js",
		"scripts/resolve-release-version.js", "scripts/run.js", "scripts/verify-published-package.js",
	} {
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(relative))); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("retired distribution entrypoint remains: %s (%v)", relative, err)
		}
	}
}

func TestWorkflowAuthoringAssetsAreAvailable(t *testing.T) {
	repo := repositoryRoot(t)
	for _, relative := range []string{
		".agents/skills/create-fanloop-workflow/SKILL.md",
		"examples/workflows/example/workflow.yaml",
		"examples/workflows/example/condition.yaml",
		"examples/workflows/example/flow.yaml",
		"examples/workflows/example/loop.yaml",
		"examples/workflows/example/prompt.yaml",
		"examples/workflows/manage-fanloop-workflow/SKILL.md",
		"examples/workflows/manage-fanloop-workflow/workflow.yaml",
		"examples/workflows/manage-fanloop-workflow/condition.yaml",
		"examples/workflows/manage-fanloop-workflow/flow.yaml",
		"examples/workflows/manage-fanloop-workflow/loop.yaml",
		"examples/workflows/manage-fanloop-workflow/prompt.yaml",
	} {
		if info, err := os.Stat(filepath.Join(repo, relative)); err != nil || !info.Mode().IsRegular() {
			t.Fatalf("authoring asset %s: %v", relative, err)
		}
	}
	skill, err := os.ReadFile(filepath.Join(repo, ".agents/skills/create-fanloop-workflow/SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"<repo>/fanloop-workflow/<workflow-id>/", "SKILL.md", "FANLOOP_CONFIG_SOURCE"} {
		if !strings.Contains(string(skill), required) {
			t.Errorf("Workflow authoring Skill is missing %q", required)
		}
	}
	for _, forbidden := range []string{"Fornax", "fanloop update"} {
		if strings.Contains(string(skill), forbidden) {
			t.Errorf("Workflow authoring Skill retains repository-delivery instruction %q", forbidden)
		}
	}
}

func TestProductionConfigurationUsesCompleteLocalPackages(t *testing.T) {
	repo := repositoryRoot(t)
	if _, err := os.Stat(filepath.Join(repo, "workflows")); !os.IsNotExist(err) {
		t.Fatal("Workflow packages must not have an extra workflows wrapper")
	}
	if _, err := os.Stat(filepath.Join(repo, "skills")); !os.IsNotExist(err) {
		t.Fatal("production packages must live only in second-brain-skills")
	}
	for _, name := range []string{"entrypoints/fanloop-workflow/SKILL.md", ".cursor/skills/verify-fanloop/SKILL.md", "scripts/pin-controller-release.sh"} {
		if _, err := os.Stat(filepath.Join(repo, name)); err != nil {
			t.Fatal(err)
		}
	}
	technicalDesign, err := os.ReadFile(filepath.Join(repo, "docs", "technical-design.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(technicalDesign), "[ADR-0079](./adr/0079-add-technical-solution-design-workflow.md)") {
		t.Fatal("technical design still treats the retired technical-solution ADR as current routing authority")
	}
	if !strings.Contains(string(technicalDesign), "[ADR-0132](./adr/0132-git-workflow-configuration-center.md)") {
		t.Fatal("technical design is missing the local package source authority")
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
		"code-change",
		"remnote-flashcards",
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

func TestVerificationDocsDoNotTreatSnapshotAsOutcomeManifest(t *testing.T) {
	repo := repositoryRoot(t)
	for _, relative := range []string{
		".cursor/skills/verify-fanloop/SKILL.md",
		".cursor/skills/verify-fanloop/features/progress-and-result.md",
	} {
		content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		text := string(content)
		if !strings.Contains(text, "fanloop verify smoke") || !strings.Contains(text, "fanloop verify snapshot") {
			t.Fatalf("%s must distinguish smoke and snapshot", relative)
		}
		for _, stale := range []string{
			"fanloop verify smoke` or `fanloop verify snapshot",
			"fanloop verify smoke` 或 `fanloop verify snapshot",
		} {
			if strings.Contains(text, stale) {
				t.Fatalf("%s still treats snapshot as an outcome-manifest producer", relative)
			}
		}
	}
}
