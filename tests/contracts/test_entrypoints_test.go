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
	for _, contract := range []string{"执行子 Agent 创建并持有 Requirement Root", "直接修改受管代码", "只向主 Agent回报", "不得自行批准", "自动 squash 合并唯一 PR", "只更新本 Requirement 的源码 worktree", "精确 merge commit"} {
		if !strings.Contains(string(agents), contract) {
			t.Errorf("AGENTS.md does not define maintainer contract %q", contract)
		}
	}
	if strings.Contains(string(agents), "./tests/run-e2e") {
		t.Error("AGENTS.md still requires the retired local E2E entrypoint")
	}
}

func TestLocalBuildReplacesDistributionEntrypoints(t *testing.T) {
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
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&pkg); err != nil {
		t.Fatal(err)
	}
	if !pkg.Private || len(pkg.Scripts) != 2 ||
		pkg.Scripts["build"] != "./scripts/build-local.sh" ||
		pkg.Scripts["install:local"] != "./scripts/install-local.sh" {
		t.Fatal("package.json must only expose the two private local build scripts")
	}
	for _, relative := range []string{"scripts/build-local.sh", "scripts/install-local.sh"} {
		info, err := os.Stat(filepath.Join(repo, filepath.FromSlash(relative)))
		if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
			t.Fatalf("local entrypoint %s is not executable: %v", relative, err)
		}
	}
	for _, relative := range []string{
		".goreleaser.yml", ".github/workflows/release.yml",
		"scripts/build-release.sh", "scripts/prepare-release.sh", "scripts/package-release.sh",
		"scripts/package-release.js", "scripts/resolve-release-version.js",
		"scripts/verify-published-package.sh", "scripts/install.js", "scripts/run.js",
	} {
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(relative))); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("retired distribution entrypoint remains: %s (%v)", relative, err)
		}
	}
}

func TestMaintainerFourStepVerificationAssetsAreComplete(t *testing.T) {
	repo := repositoryRoot(t)
	if _, err := os.Stat(filepath.Join(repo, "FEATURE_MAP.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("retired root FEATURE_MAP.md remains: %v", err)
	}
	contracts := map[string][]string{
		"scripts/install-local.sh": {
			"--config-source", "FANLOOP_CONFIG_SOURCE", "$HOME/.fanloop",
		},
		"entrypoints/fanloop-workflow/SKILL.md": {
			"固定控制器", "bound-release-home/current/bin/fanloop", "skill-roots/{codex,agent,trae,claude}", "不得回退到全局 current", "新 Requirement 的 `flow init` 始终使用全局 current", "$FANLOOP_DATA_HOME/current/workflows/**", "Dashboard", "不创建 Git diff", "不初始化 `fanloop-maintainer`", "common_skills", "common_conditions", "execution.status=awaiting_confirmation", "不执行业务 `current.prompt`", "--start-current-step", "--jump-step-id <ID>", "Panorama 是进入新 Step 后第一条用户可见消息", "真正的人工问题只能随后发送", "最终回复不重复",
		},
		"entrypoints/fanloop-workflow/agents/openai.yaml": {
			"execution.status", "common_skills", "common_conditions", "start/jump Route", "开工后再执行业务 Prompt", "Panorama 作为第一条用户可见消息", "人工问题随后发送",
		},
		".github/workflows/ci.yml": {
			"requirement-e2e", "install-doctor", "governance", "./tests/run-unit", "./.github/scripts/run-requirement-validation", "BOTMUX_CHAT_ID", "docs/research",
		},
		"workflows/fanloop-maintainer/skills/fanloop-dev-grill-with-docs/SKILL.md": {
			"一至三个", "公开入口", "独立预期", "requirements.md", "Acceptance Set", "Feature Impact Set", "执行子 Agent", "向主 Agent申请", "批准前不得修改",
		},
		"workflows/fanloop-maintainer/skills/fanloop-dev-implement/SKILL.md": {
			"verification-report.md", "./tests/run-unit", "真实公开 CLI", "fanloop-dev-maintain-verification/SKILL.md", "candidate_head", "git_tree", "binary_sha256", "可证伪假设", "delivery_main_head", "git fetch origin", "集成", "本 Step 不 push",
		},
		"workflows/fanloop-maintainer/skills/fanloop-dev-agent-acceptance/SKILL.md": {
			"独立 Review 已通过", "身份未变", "certification_head", "FANLOOP_DATA_HOME", "./scripts/install-local.sh", "fanloop-dev-verify/SKILL.md", "恰好一个", "全新 Sub-agent", "公开 CLI", "叶子 `--help`", "不得读取源码", "全局 current 未变", "verify smoke", "blocked",
		},
		"workflows/fanloop-maintainer/skills/fanloop-dev-verify/SKILL.md": {
			"Launch", "Doctor", "Drive", "Evidence", "Cleanup", "references/features/README.md", "INVALID_ARGUMENT", ".fanloop/log/cli.jsonl", ".fanloop/trace/events.jsonl",
			"证据门禁", "OUTPUT_INVALID", "post_merge_smoke_passed", "禁止手写 `manifest.json`",
		},
		".cursor/skills/verify-fanloop/SKILL.md": {
			"fanloop verify smoke", "fanloop verify snapshot", "OUTPUT_INVALID", "post_merge_smoke_passed", "Do not hand-write `manifest.json`",
		},
		"workflows/fanloop-maintainer/skills/fanloop-dev-maintain-verification/SKILL.md": {
			"clean", "changed", "blocked", "doc drift", "harness gap", "product gap",
		},
		"workflows/fanloop-maintainer/skills/fanloop-dev-workflow/SKILL.md": {
			"本机 live Workflow 配置直接编辑", "$FANLOOP_DATA_HOME/current/workflows/**", "Dashboard", "仓库 Git diff", "不初始化 `fanloop-maintainer`", "固定控制器", "define_verification_contract", "build_until_verified", "certify_candidate", "merge_and_update_local", "持有 Requirement", "受管实现", "主 Agent负责需求批准和最终候选决定", "Review 通过", "delivery_main_head", "全局 current", "Panorama 必须作为第一条用户可见消息",
		},
		"workflows/fanloop-maintainer/skills/fanloop-dev-workflow/ref/role.md": {
			"~/.fanloop/current/workflows/fanloop-maintainer/skills/fanloop-dev-workflow/SKILL.md", "Dashboard", "不创建仓库 Git diff", "不启动 `fanloop-maintainer`", "Define -> Build -> Certify -> Deliver", "前进时回 Build",
		},
		"workflows/fanloop-maintainer/skills/fanloop-dev-workflow/scripts/pin-controller-release.sh": {
			"ABSOLUTE_INITIALIZED_REQUIREMENT_ROOT", "$HOME/.fanloop/current/release", "$HOME/.fanloop/current", "--config-source", "flow status", "__install", "bound-release-home", "--replace-invalid", "doctor", `"status": "healthy"`,
		},
		"workflows/fanloop-maintainer/skills/fanloop-dev-code-review/SKILL.md": {
			"requirements.md", "Feature Impact", "./tests/run-unit", "verification-report.md", "不得在本任务中修改代码", "passed 或 failed",
		},
		"workflows/fanloop-maintainer/skills/fanloop-dev-decision-receipt/SKILL.md": {
			"decision-receipts.jsonl", "idempotency_key", "fanloop-maintainer:<step_id>", "actor_type=agent", "host_turn", "主 Agent明确决定", "main_agent_acceptance_receipt_id",
		},
		"workflows/fanloop-maintainer/skills/fanloop-dev-merge-and-update-local/SKILL.md": {
			"先回读", "MERGED", "delivery_pr_already_merged", "不要求", "origin/main", "certification_base", "certification_head", "delivery_main_advanced", "唯一 PR", "恰好一个 parent", "Review", "ADR impact", "required checks", "active Ruleset", "精确名称集合", "--match-head-commit", "禁止 `--admin`", "detached merge", "./scripts/install-local.sh", "delivery-record.md", "^{tree}", "fanloop verify smoke",
		},
	}
	for relative, snippets := range contracts {
		content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(relative)))
		if err != nil {
			t.Errorf("read %s: %v", relative, err)
			continue
		}
		for _, snippet := range snippets {
			if !strings.Contains(string(content), snippet) {
				t.Errorf("%s is missing %q", relative, snippet)
			}
		}
	}
	entrypoint, err := os.ReadFile(filepath.Join(repo, "entrypoints", "fanloop-workflow", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(entrypoint), "issue-workspace/bound-release-home") {
		t.Error("fanloop-workflow trusts a candidate-writable controller path")
	}
	if strings.Contains(string(entrypoint), "/current/skills/") {
		t.Error("fanloop-workflow still references packaged atomic Skills")
	}
	maintainerEntry, err := os.ReadFile(filepath.Join(repo, "workflows", "fanloop-maintainer", "skills", "fanloop-dev-workflow", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	workflowContract := string(maintainerEntry)
	if strings.Contains(workflowContract, "human-step-jump") || strings.Contains(workflowContract, "confirm_main_agent_acceptance") {
		t.Error("maintainer entry still documents retired nine-step behavior")
	}
	mergeSkill, err := os.ReadFile(filepath.Join(repo, "workflows", "fanloop-maintainer", "skills", "fanloop-dev-merge-and-update-local", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, fixedCheck := range []string{"test (ubuntu-latest)", "test (macos-latest)", "requirement-e2e", "install-doctor", "governance"} {
		if strings.Contains(string(mergeSkill), fixedCheck) {
			t.Errorf("merge Skill hard-codes required check %q", fixedCheck)
		}
	}
	technicalDesign, err := os.ReadFile(filepath.Join(repo, "docs", "technical-design.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, retired := range []string{"3 Stage / 3 Job / 9 Step", "Certify 并行执行", "九步 Panorama", "再次申请集成确认"} {
		if strings.Contains(string(technicalDesign), retired) {
			t.Errorf("technical design still contains retired maintainer prose %q", retired)
		}
	}
	for _, retired := range []string{
		".agents/skills/verify-fanloop",
		"workflows/fanloop-maintainer/skills/fanloop-dev-create-verification",
		"workflows/fanloop-maintainer/skills/fanloop-dev-eval-coordinator",
		"workflows/fanloop-maintainer/skills/fanloop-dev-eval-candidate",
		"workflows/fanloop-maintainer/skills/fanloop-dev-eval-judge",
		"workflows/fanloop-maintainer/skills/fanloop-dev-publish-candidate",
		"workflows/fanloop-maintainer/skills/fanloop-dev-ci-gate",
		"workflows/fanloop-maintainer/skills/fanloop-dev-agent-acceptance/ref/lark-agent-e2e.md",
		"workflows/fanloop-maintainer/skills/fanloop-dev-agent-acceptance/scripts/pin-controller-release.sh",
		"workflows/fanloop-maintainer/skills/fanloop-dev-merge-code",
		"workflows/fanloop-maintainer/skills/fanloop-dev-update-local-cli",
		"workflows/fanloop-maintainer/skills/fanloop-dev-human-step-jump",
		"workflows/fanloop-maintainer/skills/resolving-merge-conflicts",
	} {
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(retired))); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("retired maintainer asset remains: %s", retired)
		}
	}
}

func TestVerificationFeatureMapIsNavigable(t *testing.T) {
	repo := repositoryRoot(t)
	root := filepath.Join(repo, "workflows", "fanloop-maintainer", "skills", "fanloop-dev-verify", "references", "features")
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

func TestRemnoteSkillsAreBundleLocal(t *testing.T) {
	repo := repositoryRoot(t)
	root := filepath.Join(repo, "workflows", "remnote-flashcards", "skills")
	want := []string{"control-remnote", "remnote-cli", "remnote-flashcards", "remnote-structure", "remnote-verify", "remnote-wiki"}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(want) {
		t.Fatalf("Skill count = %d", len(entries))
	}
	linkPattern := regexp.MustCompile(`\]\(([^)]+\.md)\)`)
	for i, entry := range entries {
		if entry.Name() != want[i] || !entry.IsDir() {
			t.Fatalf("unexpected Skill %s", entry.Name())
		}
	}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			t.Errorf("Skill uses symlink: %s", path)
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, forbidden := range []string{".agents/skills/", "/Users/", "docs/adr/0002-remnote-content-organization.md"} {
			if strings.Contains(string(content), forbidden) {
				t.Errorf("nonportable reference %s in %s", forbidden, path)
			}
		}
		for _, match := range linkPattern.FindAllStringSubmatch(string(content), -1) {
			if strings.Contains(match[1], "://") {
				continue
			}
			target := filepath.Clean(filepath.Join(filepath.Dir(path), match[1]))
			relative, err := filepath.Rel(root, target)
			if err != nil || strings.HasPrefix(relative, "..") {
				t.Errorf("out-of-bundle reference %s in %s", target, path)
				continue
			}
			if info, err := os.Stat(target); err != nil || !info.Mode().IsRegular() {
				t.Errorf("broken reference %s in %s", target, path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repo, "workflows", "material-flashcards")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("retired workflow remains")
	}
}

func TestPanoramaPrecedesHumanQuestion(t *testing.T) {
	repo := repositoryRoot(t)
	paths := []string{
		"entrypoints/fanloop-workflow/SKILL.md",
		"workflows/fanloop-maintainer/skills/fanloop-dev-workflow/SKILL.md",
		"workflows/fanloop-maintainer/skills/fanloop-dev-panorama/SKILL.md",
		"workflows/technical-solution-design/skills/technical-solution-panorama/SKILL.md",
	}
	for _, relative := range paths {
		content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		text := string(content)
		for _, required := range []string{"第一条用户可见消息", "真正的人工问题只能随后发送", "最终回复不"} {
			if !strings.Contains(text, required) {
				t.Errorf("%s is missing Panorama-first rule %q", relative, required)
			}
		}
		if strings.Contains(text, "本轮最终普通回复必须完整展示同一份 Panorama") {
			t.Errorf("%s still requires Panorama at the bottom of the final reply", relative)
		}
	}

	entrypoint, err := os.ReadFile(filepath.Join(repo, "entrypoints", "fanloop-workflow", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	panorama := strings.Index(string(entrypoint), "Panorama 是进入新 Step 后第一条用户可见消息")
	question := strings.Index(string(entrypoint), "## 人类提问顺序")
	if panorama < 0 || question < 0 || panorama > question {
		t.Fatalf("Panorama-first rule must precede the human-question protocol")
	}
}

func TestTechnicalSolutionTemplateAllowsDynamicSubheadings(t *testing.T) {
	repo := repositoryRoot(t)
	required := map[string][]string{
		"workflows/technical-solution-design/prompt.yaml": {
			"0. 摘要", "10. 复盘与后续规划", "5 至 8 行", "2 至 3 个最关键模块", "晋升场景突出能力证据",
		},
		"workflows/technical-solution-design/skills/technical-solution-writing/SKILL.md": {
			"十一个规定 `##`", "允许 `###`", "证据状态",
		},
		"workflows/technical-solution-design/skills/technical-solution-review/SKILL.md": {
			"第 0 至第 10 章", "允许 `###`", "适用边界",
		},
		"workflows/technical-solution-design/skills/technical-problem-approval/SKILL.md": {
			"允许 `###`",
		},
		"workflows/technical-solution-design/skills/technical-direction-approval/SKILL.md": {
			"允许 `###`",
		},
		"workflows/technical-solution-design/skills/technical-solution-approval/SKILL.md": {
			"允许 `###`",
		},
	}
	for relative, snippets := range required {
		content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		for _, snippet := range snippets {
			if !strings.Contains(string(content), snippet) {
				t.Errorf("%s is missing %q", relative, snippet)
			}
		}
		for _, forbidden := range []string{"不得出现 `###`", "禁止 `###`", "且无 `###`"} {
			if strings.Contains(string(content), forbidden) {
				t.Errorf("%s still contains obsolete flat-heading rule %q", relative, forbidden)
			}
		}
	}
}

func TestTechnicalSolutionReasoningReferences(t *testing.T) {
	group := filepath.Join(repositoryRoot(t), "workflows", "technical-solution-design", "skills")
	want := filepath.Join(group, "technical-solution-review", "references", "reasoning.md")
	info, err := os.Stat(want)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		t.Fatalf("reasoning reference is not a nonempty file: %s", want)
	}
	link := regexp.MustCompile(`\[[^\]]+\]\(([^)]+reasoning\.md)\)`)
	for _, name := range []string{
		"technical-background-framing", "technical-goals-and-problems", "technical-business-constraints",
		"technical-solution-research", "technical-overall-solution", "technical-key-solutions",
		"technical-decision-recording", "technical-solution-delivery", "technical-solution-benefits",
		"technical-retrospective-planning", "technical-summary-writing", "technical-solution-writing",
		"technical-solution-review",
	} {
		t.Run(name, func(t *testing.T) {
			directory := filepath.Join(group, name)
			content, err := os.ReadFile(filepath.Join(directory, "SKILL.md"))
			if err != nil {
				t.Fatal(err)
			}
			matches := link.FindAllSubmatch(content, -1)
			if len(matches) != 1 {
				t.Fatalf("want one reasoning reference, got %d", len(matches))
			}
			got := filepath.Clean(filepath.Join(directory, string(matches[0][1])))
			if got != want {
				t.Fatalf("reasoning reference resolves to %s, want %s", got, want)
			}
		})
	}
}

func TestTechnicalSolutionStepArtifactLists(t *testing.T) {
	group := filepath.Join(repositoryRoot(t), "workflows", "technical-solution-design", "skills")
	rows := regexp.MustCompile(`(?m)^\|\s*([^|]+?)\s*\|`)
	for _, want := range []struct {
		skill     string
		artifacts []string
	}{
		{"technical-background-framing", []string{"业务阶段与目标", "现状与瓶颈", "关键事实与证据"}},
		{"technical-goals-and-problems", []string{"核心问题", "目标与验收", "非目标"}},
		{"technical-business-constraints", []string{"业务特点", "技术约束", "取舍优先级"}},
		{"technical-problem-approval", []string{"汇总的业务问题文档", "审核结论与反馈"}},
		{"technical-solution-research", []string{"候选方案对比", "适用条件", "优势与代价"}},
		{"technical-overall-solution", []string{"选型结论与依据", "总体架构图", "组件职责", "关键链路"}},
		{"technical-key-solutions", []string{"关键模块设计", "必要接口与数据模型", "异常与恢复设计"}},
		{"technical-decision-recording", []string{"决策清单", "备选比较", "后果与边界"}},
		{"technical-direction-approval", []string{"汇总的技术判断文档", "审核结论与反馈"}},
		{"technical-solution-delivery", []string{"实施阶段", "依赖与责任", "发布验证", "风险与回滚", "协作与经验"}},
		{"technical-solution-benefits", []string{"目标与收益映射", "验证计划", "已有结果及证据状态", "替代证据"}},
		{"technical-retrospective-planning", []string{"正确判断", "弯路与经验", "后续规划"}},
		{"technical-summary-writing", []string{"摘要"}},
		{"technical-solution-writing", []string{"完整技术文档"}},
		{"technical-solution-review", []string{"审校报告", "问题清单"}},
		{"technical-solution-approval", []string{"最终发布文档", "人的审核结论与反馈"}},
	} {
		t.Run(want.skill, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(group, want.skill, "SKILL.md"))
			if err != nil {
				t.Fatal(err)
			}
			_, list, ok := strings.Cut(string(content), "\n## 产物列表\n")
			if !ok {
				t.Fatal("missing step artifact list")
			}
			list, _, _ = strings.Cut(list, "\n## ")
			var got []string
			for _, row := range rows.FindAllStringSubmatch(list, -1) {
				label := strings.TrimSpace(row[1])
				if label != "逻辑产物" && strings.Trim(label, "-: ") != "" {
					got = append(got, label)
				}
			}
			if strings.Join(got, "\n") != strings.Join(want.artifacts, "\n") {
				t.Fatalf("artifacts = %v, want %v", got, want.artifacts)
			}
		})
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
		"technical-solution-design",
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
