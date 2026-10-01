package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestCIInstallDoctorUsesCompleteLocalPackages(t *testing.T) {
	repository := repositoryRoot(t)
	content, err := os.ReadFile(filepath.Join(repository, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		Jobs map[string]struct {
			Steps []struct{ Name, Run string }
		}
	}
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		t.Fatal(err)
	}
	var script string
	for _, step := range workflow.Jobs["install-doctor"].Steps {
		if step.Name == "Install current candidate in isolation" {
			script = step.Run
		}
	}
	if script == "" {
		t.Fatal("CI installation step is missing")
	}
	git := exec.Command("git", "rev-parse", "HEAD")
	git.Dir = repository
	head, err := git.Output()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command("bash", "-euo", "pipefail", "-c", script)
	command.Dir = repository
	command.Env = append(withoutBotmuxBinding(os.Environ()), "TMPDIR="+t.TempDir(), "CANDIDATE_HEAD="+strings.TrimSpace(string(head)))
	output, err := command.CombinedOutput()
	t.Logf("CI install/version/Doctor output:\n%s", output)
	if err != nil {
		t.Fatalf("CI installation entrance failed: %v", err)
	}
}
