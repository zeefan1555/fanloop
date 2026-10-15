package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandPathCheckWarnsForDistributionInstall(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FANLOOP_DATA_HOME", root)
	if err := os.MkdirAll(filepath.Join(root, "releases", "1.2.3", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("releases", "1.2.3"), filepath.Join(root, "current")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())
	check := (Runtime{}).commandPathCheck()
	if check == nil || check.Status != statusWarn || check.Hint == nil || !strings.Contains(*check.Hint, `export PATH="`) {
		t.Fatalf("missing path check = %+v", check)
	}
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "fanloop"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	check = (Runtime{}).commandPathCheck()
	if check == nil || check.Status != statusPass {
		t.Fatalf("reachable path check = %+v", check)
	}
	if err := os.Remove(filepath.Join(root, "current")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "current"), 0o755); err != nil {
		t.Fatal(err)
	}
	if (Runtime{}).commandPathCheck() != nil {
		t.Fatal("development current should not add a PATH check")
	}
}
