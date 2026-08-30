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

func TestMaintainerVerificationAssetsAreReleaseBound(t *testing.T) {
	repo := repositoryRoot(t)
	contracts := map[string][]string{
		".goreleaser.yml": {
			`"skills/**/*"`,
		},
		"FEATURE_MAP.md": {
			"症状 / 意图", "公开命令", "稳定代码 Seam", "最小真实验证", "证据",
		},
		"skills/fanloop-maintainer/fanloop-dev-grill-with-docs/SKILL.md": {
			"FEATURE_MAP.md", "baseline",
		},
		"skills/fanloop-maintainer/fanloop-dev-verify/SKILL.md": {
			"FEATURE_MAP.md", "baseline", "candidate", "./tests/run-unit", "./tests/run-e2e", "local-test-report.md",
		},
		"skills/fanloop-maintainer/fanloop-dev-maintain-verification/SKILL.md": {
			"FEATURE_MAP.md", "Source", "Live", "clean", "changed", "blocked", "acceptance-report.md",
		},
		"skills/fanloop-maintainer/fanloop-dev-agent-acceptance/SKILL.md": {
			"review-report.md", "npm run install:local", "fanloop version", "fanloop doctor", "ref/eval-playbook.md", "ref/lark-agent-e2e.md", "acceptance-report.md", "Card Binding", "trace_synced", "governance_failed",
		},
		"skills/fanloop-maintainer/fanloop-dev-agent-acceptance/ref/eval-playbook.md": {
			"Verification Case", "Rubric", "candidate_commit",
		},
		"skills/fanloop-maintainer/fanloop-dev-agent-acceptance/ref/lark-agent-e2e.md": {
			"cli_aafadbc67e799cdc", "cli_a9245f0fddf8dbc8", "oc_d532c3a5eda84c60728ab174b0ef671a", "botmux dispatch", "用户 token", "lark-cli whoami --as bot", "trace_document_bound", "registry",
		},
		"skills/fanloop-maintainer/fanloop-dev-code-review/SKILL.md": {
			"Review 之后", "reviewed HEAD", "technical_solution_changes_requested",
		},
		"skills/fanloop-maintainer/fanloop-dev-merge-code/SKILL.md": {
			"acceptance-report.md", "gh pr merge", "--squash", "--match-head-commit", "code_merged",
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
