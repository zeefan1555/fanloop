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
