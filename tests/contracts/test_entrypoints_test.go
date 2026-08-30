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
			"FEATURE_MAP.md", "ref/eval-playbook.md", "ref/lark-agent-e2e.md", "baseline", "candidate", "npm run install:local",
		},
		"skills/fanloop-maintainer/fanloop-dev-verify/ref/eval-playbook.md": {
			"Verification Case", "Rubric", "candidate_commit",
		},
		"skills/fanloop-maintainer/fanloop-dev-verify/ref/lark-agent-e2e.md": {
			"cli_aafadbc67e799cdc", "cli_a9245f0fddf8dbc8", "oc_d532c3a5eda84c60728ab174b0ef671a", "botmux dispatch", "用户 token",
		},
		"skills/fanloop-maintainer/fanloop-dev-code-review/SKILL.md": {
			"Agent Eval", "reviewed HEAD",
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
}
