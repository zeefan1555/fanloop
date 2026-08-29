package release

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultDataRootUsesHomeCommonloopDirectory(t *testing.T) {
	t.Setenv("COMMONLOOP_DATA_HOME", "")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	got, err := DefaultDataRoot()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".commonloop"); got != want {
		t.Fatalf("default data root = %q, want %q", got, want)
	}
}

func TestDefaultSkillRootsIncludeClaudeHomeDirectory(t *testing.T) {
	t.Setenv("COMMONLOOP_CLAUDE_SKILLS_ROOT", "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	roots, err := DefaultSkillRoots()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".claude", "skills"); roots.Claude != want {
		t.Fatalf("Claude Skill root = %q, want %q", roots.Claude, want)
	}
}

func TestDefaultSkillRootsUseAllFourClientOverrides(t *testing.T) {
	base := t.TempDir()
	t.Setenv("COMMONLOOP_CODEX_SKILLS_ROOT", filepath.Join(base, "codex"))
	t.Setenv("COMMONLOOP_AGENT_SKILLS_ROOT", filepath.Join(base, "agents"))
	t.Setenv("COMMONLOOP_TRAE_SKILLS_ROOT", filepath.Join(base, "trae"))
	t.Setenv("COMMONLOOP_CLAUDE_SKILLS_ROOT", filepath.Join(base, "claude"))
	roots, err := DefaultSkillRoots()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(base, "codex"), filepath.Join(base, "agents"), filepath.Join(base, "trae"), filepath.Join(base, "claude")}
	if got := roots.Values(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Skill roots = %#v, want %#v", got, want)
	}
}
