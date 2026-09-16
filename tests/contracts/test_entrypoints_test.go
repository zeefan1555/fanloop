package contracts

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestRepositoryHasTwoPublicTestEntrypoints(t *testing.T) {
	repo := repositoryRoot(t)
	entrypoints := []string{"tests/run-unit", "tests/run-e2e"}
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
	for _, command := range []string{"./tests/run-unit", "./tests/run-e2e"} {
		if !strings.Contains(string(agents), command) {
			t.Errorf("AGENTS.md does not require %s", command)
		}
	}
}

func TestLocalBuildReplacesDistributionEntrypoints(t *testing.T) {
	repo := repositoryRoot(t)
	for _, relative := range []string{"scripts/build-local.sh", "scripts/install-local.sh"} {
		info, err := os.Stat(filepath.Join(repo, filepath.FromSlash(relative)))
		if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
			t.Fatalf("local entrypoint %s is not executable: %v", relative, err)
		}
	}
	for _, relative := range []string{
		"package.json", ".goreleaser.yml", ".github/workflows/release.yml",
		"scripts/build-release.sh", "scripts/prepare-release.sh", "scripts/package-release.sh",
		"scripts/package-release.js", "scripts/resolve-release-version.js",
		"scripts/verify-published-package.sh", "scripts/install.js", "scripts/run.js",
	} {
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(relative))); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("retired distribution entrypoint remains: %s (%v)", relative, err)
		}
	}
}

func TestMaintainerThreeStageDeliveryAssetsAreComplete(t *testing.T) {
	repo := repositoryRoot(t)
	if _, err := os.Stat(filepath.Join(repo, "FEATURE_MAP.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("retired root FEATURE_MAP.md remains: %v", err)
	}
	contracts := map[string][]string{
		"scripts/install-local.sh": {
			"--config-source", "FANLOOP_CONFIG_SOURCE", "$HOME/.fanloop",
		},
		"entrypoints/fanloop-workflow/SKILL.md": {
			"固定控制器", "bound-release-home/current/bin/fanloop", "skill-roots/{codex,agent,trae,claude}", "不得回退到全局 current", "新 Requirement 的 `flow init` 始终使用全局 current", "live 配置目录 `skills/**`", "不创建或初始化 `fanloop-maintainer`", "<REQUIREMENT_CONTROLLER> flow status", "<REQUIREMENT_CONTROLLER> card render",
		},
		".github/workflows/ci.yml": {
			"requirement-e2e", "install-doctor", "governance", "./tests/run-unit", "./tests/run-e2e", "BOTMUX_CHAT_ID", "docs/research",
		},
		"skills/fanloop-maintainer/fanloop-dev-grill-with-docs/SKILL.md": {
			"1 至 3", "公开 CLI", "独立预期", "requirements.md", "稳定标题", "唯一飞书需求文档", "语义回读",
		},
		"skills/fanloop-maintainer/fanloop-dev-implement/SKILL.md": {
			"implementation-report.md", "当前 HEAD", "唯一飞书研发实现报告", "语义回读",
		},
		"skills/fanloop-maintainer/fanloop-dev-agent-acceptance/SKILL.md": {
			"candidate_head", "FANLOOP_DATA_HOME", "FANLOOP_CODEX_SKILLS_ROOT", "./scripts/install-local.sh", "恰好一个", "全新 Sub-agent", "1 至 3", "公开 CLI", "叶子 `--help`", "不得读取源码", "全局 current 未变", "acceptance-report.md", "唯一飞书验收交付报告", "基础设施失败保持 blocked",
		},
		"skills/fanloop-maintainer/fanloop-dev-workflow/SKILL.md": {
			"纯 live Skill 配置变更直接交付", "若全部变更位于 `skills/**`", "不创建、不初始化 `fanloop-maintainer`", "固定控制器", "bound-release-home", "$HOME/.fanloop/current", "WORKFLOW_MISMATCH", "Sub-agent", "expectedApprover", "cli_aaf6cd8160b89bda", "ou_3b0b9cf8364168c5eb999bd6c5a33b95", "Stage/Job/Step", "目标、现状问题、逐项改造、影响文件/契约、保持不变与非目标、验证计划、交付边界", "精确授权口令", "turn boundary", "senderType=user", "botmux quoted", "批准进入 需求实现", "<REQUIREMENT_CONTROLLER> flow report", "<REQUIREMENT_CONTROLLER> flow status", "<REQUIREMENT_CONTROLLER> card render",
		},
		"skills/fanloop-maintainer/fanloop-dev-workflow/ref/role.md": {
			"live 配置目录 `skills/**`", "直接修改和聚焦验证", "不启动 `fanloop-maintainer`",
		},
		"skills/fanloop-maintainer/fanloop-dev-code-review/SKILL.md": {
			"reviewed HEAD", "./tests/run-unit", "./tests/run-e2e", "implementation-report.md", "同一飞书研发实现报告", "candidate_head_frozen",
		},
		"skills/fanloop-maintainer/fanloop-dev-merge-code/SKILL.md": {
			"唯一", "Ruleset", "required checks", "candidate_head", "acceptance-report.md", "同一飞书验收交付报告", "gh pr merge", "--auto", "--squash", "--match-head-commit", "code_merged",
		},
		"skills/fanloop-maintainer/fanloop-dev-update-local-cli/SKILL.md": {
			"pin-controller-release.sh", "bound-release-home", "origin/main", "detached worktree", "./scripts/install-local.sh", "version commit", "Doctor", "acceptance-report.md", "飞书验收交付报告", "local_cli_updated",
		},
		"skills/fanloop-maintainer/fanloop-dev-update-local-cli/scripts/pin-controller-release.sh": {
			"ABSOLUTE_INITIALIZED_REQUIREMENT_ROOT", "$HOME/.fanloop/current", "$HOME/.fanloop/config/current", "FANLOOP_CONFIG_ROOT=$controller_home/config/current", "--config-source", "flow status", "__install", "bound-release-home", "--replace-invalid", "doctor", `"status": "healthy"`,
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
	maintainerEntry, err := os.ReadFile(filepath.Join(repo, "skills", "fanloop-maintainer", "fanloop-dev-workflow", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	approval := string(maintainerEntry)
	panorama := strings.Index(approval, "先按 `panorama_card_published`")
	card := strings.Index(approval, "再从最新 `requirements.md`")
	boundary := strings.Index(approval, "确认卡发送成功为 turn boundary")
	if panorama < 0 || card <= panorama || boundary <= card {
		t.Errorf("human approval order must be Panorama -> full card -> turn boundary")
	}
	for _, retired := range []string{
		".agents/skills/verify-fanloop",
		"skills/fanloop-maintainer/fanloop-dev-create-verification",
		"skills/fanloop-maintainer/fanloop-dev-maintain-verification",
		"skills/fanloop-maintainer/fanloop-dev-verify",
		"skills/fanloop-maintainer/fanloop-dev-eval-coordinator",
		"skills/fanloop-maintainer/fanloop-dev-eval-candidate",
		"skills/fanloop-maintainer/fanloop-dev-eval-judge",
		"skills/fanloop-maintainer/fanloop-dev-publish-candidate",
		"skills/fanloop-maintainer/fanloop-dev-ci-gate",
		"skills/fanloop-maintainer/fanloop-dev-agent-acceptance/ref/lark-agent-e2e.md",
		"skills/fanloop-maintainer/fanloop-dev-agent-acceptance/scripts/pin-controller-release.sh",
	} {
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(retired))); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("retired maintainer asset remains: %s", retired)
		}
	}
}

func TestMaterialFlashcardsAssetsEnforceApprovedBoundary(t *testing.T) {
	repo := repositoryRoot(t)
	contracts := map[string][]string{
		"workflows/material-flashcards/prompt.yaml": {
			"候选数量由材料决定，不设固定数量或类型配额", "create-exclusive/no-replace", "不得自动创建第二个目标",
		},
		"skills/material-flashcards/flashcard/SKILL.md": {
			"Decks", "references/term-concept-card.md", "文件级校验",
			`r"^##\s+"`, `r"^#{3,6}\s+"`, `r"decks/[^\s]+"`,
		},
		"skills/material-flashcards/flashcard/references/term-concept-card.md": {
			"定义型概念卡", "区分型概念卡", "运作型概念卡", "【Why", "【What", "【How",
		},
		"skills/material-flashcards/flashcard-goal-framing/SKILL.md": {
			"write_mode=create_new_only", "预览投递位置", "Vault 相对", "create-exclusive/no-replace", "blocked",
		},
		"skills/material-flashcards/flashcard-source-understanding/SKILL.md": {
			"来源事实", "被归因的判断", "用户理解", "unknown", "私密材料不得进入 Event / Trace / CLI 日志或未确认的群聊",
		},
		"skills/material-flashcards/flashcard-knowledge-selection/SKILL.md": {
			"0..N", "长期价值", "现实适用性", "分类级排除理由", "Vault 零写入", "新 Requirement",
		},
		"skills/material-flashcards/flashcard-card-planning/SKILL.md": {
			"一张卡一个", "集合", "长枚举", "8 adopted / 8 partial / 4 non-goal", "term-concept-card.md",
		},
		"skills/material-flashcards/flashcard-quality-review/SKILL.md": {
			"pass / fail / not-applicable", "不得修改任何输入", "最早受影响层", "20 rules", "不复制其规则",
		},
		"skills/material-flashcards/flashcard-preview-approval/SKILL.md": {
			"quality_review", "preview_record", "approval_record", "sender_type=human", "不得自行批准", "新的明确人类批准",
		},
		"skills/material-flashcards/material-flashcards-panorama/SKILL.md": {
			"只依据系统或开发者上下文中已经声明的当前 Agent 人设", "Botmux Agent：`botmux`",
			"AIME Agent：`aime`", "Aiden Agent：`aiden`", "Codex、Claude Code 和 Trae：`local_agent`",
			"botmux send --card-file", "本轮最终普通回复必须完整展示同一份 Panorama", "不自行拼装内容",
			`--content "$(cat -- "$card_file")"`, "渲染前确认 Current Evidence 为空",
			"卡片正文", "个人细节", "来源内容", "findings", "反馈",
		},
		"entrypoints/fanloop-workflow/routes.yaml": {
			"material-flashcards:", "workflow: material-flashcards", "description: 从材料生成经人工确认的长期复习闪卡组",
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

	for _, root := range []string{
		filepath.Join(repo, "workflows", "material-flashcards"),
		filepath.Join(repo, "skills", "material-flashcards"),
	} {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if strings.Contains(string(content), "/Users/") {
				t.Errorf("%s contains a machine-specific absolute path", path)
			}
			return nil
		})
		if err != nil {
			t.Errorf("walk %s: %v", root, err)
		}
	}
}

func TestTechnicalSolutionTemplateAllowsDynamicSubheadings(t *testing.T) {
	repo := repositoryRoot(t)
	required := map[string][]string{
		"workflows/technical-solution-design/prompt.yaml": {
			"0. 摘要", "10. 复盘与后续规划", "5 至 8 行", "2 至 3 个最关键模块", "晋升场景突出能力证据",
		},
		"skills/technical-solution-design/technical-solution-writing/SKILL.md": {
			"十一个规定 `##`", "允许 `###`", "证据状态",
		},
		"skills/technical-solution-design/technical-solution-review/SKILL.md": {
			"第 0 至第 10 章", "允许 `###`", "适用边界",
		},
		"skills/technical-solution-design/technical-problem-approval/SKILL.md": {
			"允许 `###`",
		},
		"skills/technical-solution-design/technical-direction-approval/SKILL.md": {
			"允许 `###`",
		},
		"skills/technical-solution-design/technical-solution-approval/SKILL.md": {
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
	group := filepath.Join(repositoryRoot(t), "skills", "technical-solution-design")
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
	group := filepath.Join(repositoryRoot(t), "skills", "technical-solution-design")
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
