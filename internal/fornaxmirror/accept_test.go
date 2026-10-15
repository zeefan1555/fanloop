package fornaxmirror

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMaterializeLoopPackageUsesWorkflowIDAndShebang(t *testing.T) {
	packageRoot := filepath.Join(t.TempDir(), "loop-workflow-linear-runtime")
	if err := copyFixtureWithoutPackageSkill(t, packageRoot); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(packageRoot, "linear-runtime", "skills", "test-guide", "scripts", "run.sh")
	if err := os.MkdirAll(filepath.Dir(script), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho ok\n"), 0644); err != nil {
		t.Fatal(err)
	}
	wrapper := "---\nname: loop-workflow-linear-runtime\ndescription: Managed Loop Workflow bundle for linear-runtime.\n---\n\n# Loop Workflow Bundle\n"
	if err := os.WriteFile(filepath.Join(packageRoot, "SKILL.md"), []byte(wrapper), 0644); err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	accepted, err := MaterializePackage(packageRoot, parent)
	if err != nil {
		t.Fatal(err)
	}
	if accepted.WorkflowID != "linear-runtime" || accepted.WrapperName != "loop-workflow-linear-runtime" || accepted.SkillKey != "" || accepted.Version != "" {
		t.Fatalf("accepted = %#v", accepted)
	}
	installedSkill, err := os.ReadFile(filepath.Join(parent, "linear-runtime", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(installedSkill) != wrapper {
		t.Fatalf("installed SKILL.md = %q", installedSkill)
	}
	info, err := os.Stat(filepath.Join(parent, "linear-runtime", "skills", "test-guide", "scripts", "run.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0555 {
		t.Fatalf("script mode = %#o", info.Mode().Perm())
	}
	yamlInfo, err := os.Stat(filepath.Join(parent, "linear-runtime", "workflow.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if yamlInfo.Mode().Perm() != 0444 {
		t.Fatalf("workflow.yaml mode = %#o", yamlInfo.Mode().Perm())
	}
}

func TestMaterializeFanloopWrapperAllowsMissingSourceCommit(t *testing.T) {
	packageRoot := filepath.Join(t.TempDir(), "fanloop-workflow-linear-runtime")
	if err := copyFixtureWithoutPackageSkill(t, packageRoot); err != nil {
		t.Fatal(err)
	}
	wrapper := "---\nname: fanloop-workflow-linear-runtime\ndescription: Fornax mirror of Fanloop Workflow linear-runtime.\nmetadata:\n  workflow_id: linear-runtime\n  skill_key: fanloop.workflow.linear_runtime\n  version: 1.2.3\n---\n\n# Fanloop Workflow mirror\n"
	if err := os.WriteFile(filepath.Join(packageRoot, "SKILL.md"), []byte(wrapper), 0644); err != nil {
		t.Fatal(err)
	}
	accepted, err := MaterializePackage(packageRoot, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if accepted.WorkflowID != "linear-runtime" || accepted.SkillKey != "fanloop.workflow.linear_runtime" || accepted.Version != "1.2.3" {
		t.Fatalf("accepted = %#v", accepted)
	}
}

func TestAcceptPackageRootRejectsSymlinkAndNameMismatch(t *testing.T) {
	packageRoot := filepath.Join(t.TempDir(), "loop-workflow-linear-runtime")
	if err := copyFixtureWithoutPackageSkill(t, packageRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, "SKILL.md"), []byte("---\nname: loop-workflow-other\ndescription: wrong name\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := AcceptPackageRoot(packageRoot); err == nil {
		t.Fatal("mismatched package name was accepted")
	}
	linked := filepath.Join(t.TempDir(), "linked")
	if err := os.Symlink(packageRoot, linked); err != nil {
		t.Fatal(err)
	}
	if _, err := AcceptPackageRoot(linked); err == nil {
		t.Fatal("symlink package root was accepted")
	}
}

func TestMaterializeRealLoopCLISample(t *testing.T) {
	sample := os.Getenv("FANLOOP_LOOPCLI_SAMPLE")
	if sample == "" {
		sample = "/tmp/loopcli/samples/team/loop-workflow-pstack-native"
	}
	if _, err := os.Stat(sample); err != nil {
		t.Skip("loop_cli sample package is not on this machine")
	}
	parent := t.TempDir()
	accepted, err := MaterializePackage(sample, parent)
	if err != nil {
		t.Fatal(err)
	}
	if accepted.WorkflowID != "pstack-native" || accepted.WrapperName != "loop-workflow-pstack-native" || accepted.SkillKey != "" || accepted.Version != "" {
		t.Fatalf("accepted = %#v", accepted)
	}
	info, err := os.Stat(filepath.Join(parent, "pstack-native", "skills", "poteto-mode", "scripts", "worktree-audit.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0555 {
		t.Fatalf("worktree-audit.sh mode = %#o", info.Mode().Perm())
	}
}

func copyFixtureWithoutPackageSkill(t *testing.T, packageRoot string) error {
	t.Helper()
	source := filepath.Join("..", "workflow", "testdata", "linear-runtime")
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "SKILL.md" {
			return nil
		}
		target := filepath.Join(packageRoot, "linear-runtime", relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, 0644)
	})
}
