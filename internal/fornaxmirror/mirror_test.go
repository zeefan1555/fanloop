package fornaxmirror

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestMirrorWritesSnapshotBytesIntoCheckout(t *testing.T) {
	snapshot := t.TempDir()
	workflow := filepath.Join(snapshot, "fanloop-workflow", "linear-runtime")
	scriptDir := filepath.Join(workflow, "skills", "test-guide", "scripts")
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workflow, "workflow.yaml"), []byte("id: linear-runtime\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scriptDir, "run.sh"), []byte("#!/bin/sh\necho ok\n"), 0755); err != nil {
		t.Fatal(err)
	}
	checkout := t.TempDir()
	stale := filepath.Join(checkout, "fanloop-workflow", "linear-runtime")
	if err := os.MkdirAll(stale, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stale, "workflow.yaml"), []byte("id: old\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stale, "old.md"), []byte("remove me\n"), 0644); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join("..", "..", "scripts", "mirror-fornax-to-git")
	command := exec.Command("python3", script, snapshot, checkout)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("mirror: %v\n%s", err, output)
	}
	content, err := os.ReadFile(filepath.Join(checkout, "fanloop-workflow", "linear-runtime", "workflow.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "id: linear-runtime\n" {
		t.Fatalf("workflow.yaml = %q", content)
	}
	if _, err = os.Stat(filepath.Join(checkout, "fanloop-workflow", "linear-runtime", "old.md")); !os.IsNotExist(err) {
		t.Fatalf("old.md remained: %v", err)
	}
	info, err := os.Stat(filepath.Join(checkout, "fanloop-workflow", "linear-runtime", "skills", "test-guide", "scripts", "run.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0555 {
		t.Fatalf("run.sh mode = %#o", info.Mode().Perm())
	}
	scriptContent, err := os.ReadFile(filepath.Join(checkout, "fanloop-workflow", "linear-runtime", "skills", "test-guide", "scripts", "run.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if string(scriptContent) != "#!/bin/sh\necho ok\n" {
		t.Fatalf("run.sh = %q", scriptContent)
	}
}

func TestUpdateSkillNamesReadWriteSourceOnly(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", ".agents", "skills", "update-fanloop-workflow", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, want := range []string{
		"只使用 `access` 为 `read_write` 的那一项",
		"scripts/mirror-fornax-to-git",
		"tools/workflow-package build` 只用于恢复",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("skill missing %q", want)
		}
	}
	if regexp.MustCompile(`[0-9]{10,}`).MatchString(text) {
		t.Fatal("skill contains a long numeric id")
	}
}
