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

func TestMaintainerVerificationAndDeliveryAssetsAreComplete(t *testing.T) {
	repo := repositoryRoot(t)
	if _, err := os.Stat(filepath.Join(repo, "FEATURE_MAP.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("root FEATURE_MAP.md must not duplicate the project verification map: %v", err)
	}
	contracts := map[string][]string{
		".github/workflows/ci.yml": {
			"requirement-e2e", "install-doctor", "governance", "./tests/run-unit", "./tests/run-e2e", "BOTMUX_CHAT_ID", "docs/research",
		},
		".goreleaser.yml": {
			`"skills/**/*"`,
		},
		".agents/skills/verify-fanloop/SKILL.md": {
			"Launch", "Doctor", "Drive", "Evidence", "Cleanup", "unset BOTMUX_CHAT_ID BOTMUX_SESSION_ID",
		},
		".agents/skills/verify-fanloop/features/README.md": {
			"Fanloop 功能地图", "Requirement 与 Flow", "Output 与 State", "安装、Release 与 Skills",
		},
		"skills/fanloop-maintainer/fanloop-dev-grill-with-docs/SKILL.md": {
			".agents/skills/verify-fanloop/features/README.md", "baseline",
		},
		"skills/fanloop-maintainer/fanloop-dev-verify/SKILL.md": {
			".agents/skills/verify-fanloop/features/README.md", "baseline", "candidate", "./tests/run-unit", "./tests/run-e2e", "local-test-report.md",
		},
		"skills/fanloop-maintainer/fanloop-dev-create-verification/SKILL.md": {
			"Launch", "Doctor", "Drive", "Evidence", "Cleanup", ".agents/skills/verify-fanloop",
		},
		"skills/fanloop-maintainer/fanloop-dev-maintain-verification/SKILL.md": {
			".agents/skills/verify-fanloop", "用户入口", "真实操作", "可观察结果", "clean", "updated", "blocked",
		},
		"skills/fanloop-maintainer/fanloop-dev-eval-coordinator/SKILL.md": {
			"恰好两个", "10 分", "Rubric", "随机", "不同模型", "eval-playbook.<sha256>.md", "brief_sha256", "rubric_sha256",
		},
		"skills/fanloop-maintainer/fanloop-dev-eval-candidate/SKILL.md": {
			"并行", "独立随机目录", "candidate_head", "eval-candidates-report.md",
		},
		"skills/fanloop-maintainer/fanloop-dev-eval-judge/SKILL.md": {
			"不同于候选模型", "10/10", "最多三轮", "acceptance-report.md",
		},
		"skills/fanloop-maintainer/fanloop-dev-publish-candidate/SKILL.md": {
			"candidate_head", "base=main", "唯一", "pull_request_url",
		},
		"skills/fanloop-maintainer/fanloop-dev-ci-gate/SKILL.md": {
			"Ruleset", "strict", "squash", "linear", "./tests/run-unit", "./tests/run-e2e", "candidate_head",
		},
		"skills/fanloop-maintainer/fanloop-dev-agent-acceptance/SKILL.md": {
			"candidate_head", "fanloop version", "fanloop doctor", "ref/lark-agent-e2e.md", "acceptance-report.md", "brief_sha256", "rubric_sha256", "--brief-file", "禁止生成", "Card Binding", "Trace Integration", "governance_failed",
		},
		"skills/fanloop-maintainer/fanloop-dev-agent-acceptance/ref/lark-agent-e2e.md": {
			"cli_aafadbc67e799cdc", "cli_a9245f0fddf8dbc8", "oc_d532c3a5eda84c60728ab174b0ef671a", "botmux dispatch", "FROZEN_BRIEF_PATH", "brief_sha256", "rubric_sha256", "不得再生成", "用户 token", "lark-cli whoami --as bot", "env -u BOTMUX_CHAT_ID -u BOTMUX_SESSION_ID", "trace_document_bound",
		},
		"skills/fanloop-maintainer/fanloop-dev-code-review/SKILL.md": {
			"Review 之后", "reviewed HEAD", "technical_solution_changes_requested",
		},
		"skills/fanloop-maintainer/fanloop-dev-merge-code/SKILL.md": {
			"acceptance-report.md", "gh pr merge", "--auto", "--squash", "--match-head-commit", "code_merged",
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
	verify, err := os.ReadFile(filepath.Join(repo, "skills", "fanloop-maintainer", "fanloop-dev-verify", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"botmux dispatch", "ref/eval-playbook.md", "npm run install:local"} {
		if strings.Contains(string(verify), forbidden) {
			t.Errorf("fanloop-dev-verify still owns Agent acceptance detail %q", forbidden)
		}
	}
}
