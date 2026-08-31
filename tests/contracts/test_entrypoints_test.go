package contracts

import (
	"errors"
	"os"
	"path/filepath"
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

func TestMaintainerThreeStageDeliveryAssetsAreComplete(t *testing.T) {
	repo := repositoryRoot(t)
	if _, err := os.Stat(filepath.Join(repo, "FEATURE_MAP.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("retired root FEATURE_MAP.md remains: %v", err)
	}
	contracts := map[string][]string{
		"entrypoints/fanloop-workflow/SKILL.md": {
			"固定控制器", "bound-release-home/current/bin/fanloop", "skill-roots/{codex,agent,trae,claude}", "不得回退到全局 current", "新 Requirement 的 `flow init` 始终使用全局 current", "<REQUIREMENT_CONTROLLER> flow status", "<REQUIREMENT_CONTROLLER> card render",
		},
		".github/workflows/ci.yml": {
			"requirement-e2e", "install-doctor", "governance", "./tests/run-unit", "./tests/run-e2e", "BOTMUX_CHAT_ID", "docs/research",
		},
		".goreleaser.yml": {
			`"skills/**/*"`,
		},
		"skills/fanloop-maintainer/fanloop-dev-grill-with-docs/SKILL.md": {
			"1 至 3", "公开 CLI", "独立预期", "requirements.md", "稳定标题", "唯一飞书需求文档", "语义回读",
		},
		"skills/fanloop-maintainer/fanloop-dev-implement/SKILL.md": {
			"implementation-report.md", "当前 HEAD", "唯一飞书研发实现报告", "语义回读",
		},
		"skills/fanloop-maintainer/fanloop-dev-agent-acceptance/SKILL.md": {
			"candidate_head", "FANLOOP_DATA_HOME", "FANLOOP_CODEX_SKILLS_ROOT", "npm run install:local", "恰好一个", "全新 Sub-agent", "1 至 3", "公开 CLI", "叶子 `--help`", "不得读取源码", "全局 current 未变", "acceptance-report.md", "唯一飞书验收交付报告", "基础设施失败保持 blocked",
		},
		"skills/fanloop-maintainer/fanloop-dev-workflow/SKILL.md": {
			"固定控制器", "bound-release-home", "$HOME/.fanloop/current", "WORKFLOW_MISMATCH", "Sub-agent", "<REQUIREMENT_CONTROLLER> flow report", "<REQUIREMENT_CONTROLLER> flow status", "<REQUIREMENT_CONTROLLER> card render",
		},
		"skills/fanloop-maintainer/fanloop-dev-code-review/SKILL.md": {
			"reviewed HEAD", "./tests/run-unit", "./tests/run-e2e", "implementation-report.md", "同一飞书研发实现报告", "candidate_head_frozen",
		},
		"skills/fanloop-maintainer/fanloop-dev-merge-code/SKILL.md": {
			"唯一", "Ruleset", "required checks", "candidate_head", "acceptance-report.md", "同一飞书验收交付报告", "gh pr merge", "--auto", "--squash", "--match-head-commit", "code_merged",
		},
		"skills/fanloop-maintainer/fanloop-dev-update-local-cli/SKILL.md": {
			"pin-controller-release.sh", "bound-release-home", "origin/main", "detached worktree", "npm run install:local", "version commit", "Doctor", "acceptance-report.md", "飞书验收交付报告", "local_cli_updated",
		},
		"skills/fanloop-maintainer/fanloop-dev-update-local-cli/scripts/pin-controller-release.sh": {
			"ABSOLUTE_INITIALIZED_REQUIREMENT_ROOT", "$HOME/.fanloop/current", "flow status", "__install", "bound-release-home", "--replace-invalid", "doctor", `"status": "healthy"`,
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

func TestTechnicalSolutionTemplateAllowsDynamicSubheadings(t *testing.T) {
	repo := repositoryRoot(t)
	required := map[string][]string{
		"workflows/technical-solution-design/prompt.yaml": {
			"九个二级语义章节", "允许按项目内容生成三级标题", "来源和证据状态", "适用场景、不适用场景",
		},
		"skills/technical-solution-design/technical-solution-writing/SKILL.md": {
			"九个语义章节", "允许 `###`", "证据状态",
		},
		"skills/technical-solution-design/technical-solution-review/SKILL.md": {
			"语义章节", "允许 `###`", "适用边界",
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
