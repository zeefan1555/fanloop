package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestLocalSourceBindingAndOverride(t *testing.T) {
	dataRoot, source, override := t.TempDir(), t.TempDir(), t.TempDir()
	writeValidWorkflow(t, filepath.Join(source, "skills"), "bound-loop")
	writeValidWorkflow(t, filepath.Join(override, "skills"), "override-loop")
	if err := os.MkdirAll(filepath.Join(dataRoot, "config"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(source, filepath.Join(dataRoot, "config", "current")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_CONFIG_SOURCE", "")
	if err := os.Unsetenv("FANLOOP_CONFIG_SOURCE"); err != nil {
		t.Fatal(err)
	}
	if _, err := Load("bound-loop"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FANLOOP_CONFIG_SOURCE", override)
	if _, err := Load("override-loop"); err != nil {
		t.Fatal(err)
	}
	if _, err := Load("bound-loop"); err == nil {
		t.Fatal("explicit source fell back to installed binding")
	}
	for _, invalid := range []string{"", " ", "relative", filepath.Join(override, "missing")} {
		t.Setenv("FANLOOP_CONFIG_SOURCE", invalid)
		if _, err := Load("bound-loop"); err == nil {
			t.Fatalf("invalid source %q fell back to installed binding", invalid)
		}
	}
}

func TestPackageSourceLockAndResourceValidation(t *testing.T) {
	root := t.TempDir()
	writeValidWorkflow(t, root, "local-loop")
	unlock, err := LockSource(root, true)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	file, err := os.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := unix.Flock(int(file.Fd()), unix.LOCK_SH|unix.LOCK_NB); err == nil {
		t.Fatal("source lock did not exclude a second reader")
	}
	resource := filepath.Join(root, "local-loop", "skills", "writer", "outside")
	if err := os.Symlink(t.TempDir(), resource); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFrom(root, "local-loop"); err == nil {
		t.Fatal("package accepted a symlinked resource")
	}
	if err := os.Remove(resource); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "local-loop", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFrom(root, "local-loop"); err == nil {
		t.Fatal("package accepted a missing root SKILL.md")
	}
}
