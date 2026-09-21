package install

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/release"
)

func TestPreflightSkillLinksExposesOnlyWorkflowSkill(t *testing.T) {
	root := t.TempDir()
	request := Request{
		DataRoot: root,
		SkillRoots: release.SkillRoots{
			Codex:  filepath.Join(root, "codex"),
			Agent:  filepath.Join(root, "agent"),
			Trae:   filepath.Join(root, "trae"),
			Claude: filepath.Join(root, "claude"),
		},
	}
	manifest := release.Manifest{Skills: []*release.Skill{
		{Name: "ai-test", Path: "skills/technical-solution-design/ai-test"},
		{Name: release.ExposedSkillName, Path: release.ExposedSkillPath},
		{Name: "fanloop-dev-tdd", Path: "skills/fanloop-maintainer/fanloop-dev-tdd"},
	}}
	plans, external, err := preflightSkillLinks(request, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 4 || len(external) != 0 {
		t.Fatalf("plans = %#v, external = %#v", plans, external)
	}
	for _, plan := range plans {
		if filepath.Base(plan.path) != release.ExposedSkillName || plan.target != filepath.Join(root, "current", "entrypoints", "fanloop-workflow") {
			t.Fatalf("unexpected link plan: %#v", plan)
		}
	}
}

func TestCurrentSkillsPreserveGroupedTargets(t *testing.T) {
	root := t.TempDir()
	manifest := release.Manifest{Skills: []*release.Skill{
		{Name: "retired", Path: "skills/technical-solution-design/retired"},
		{Name: "internal", Path: "skills/fanloop-maintainer/internal"},
	}}
	content, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	// Decode validation is intentionally bypassed here so the seam only exercises
	// the installed manifest's path-to-link retirement mapping.
	if err := os.WriteFile(filepath.Join(root, "release.json"), content, 0o600); err != nil {
		t.Fatal(err)
	}

	got := currentSkills(root)
	if len(got) != 2 || got[0].name != "retired" || got[0].path != "skills/technical-solution-design/retired" || got[1].name != "internal" || got[1].path != "skills/fanloop-maintainer/internal" {
		t.Fatalf("currentSkills() = %#v", got)
	}
}

func TestRollbackSkillLinkChangesRestoresExternalLink(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "skill")
	externalTarget := filepath.Join(root, "external")
	marker := filepath.Join(externalTarget, "marker")
	if err := os.Mkdir(externalTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, []byte("preserve me\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "current", "skills", "skill"), path); err != nil {
		t.Fatal(err)
	}

	cause := errors.New("activate current")
	err := rollbackSkillLinkChanges(cause, []string{path}, []linkPlan{{path: path, target: externalTarget}})
	if !errors.Is(err, cause) {
		t.Fatalf("rollback error = %v, want original cause", err)
	}
	if target, readErr := os.Readlink(path); readErr != nil || target != externalTarget {
		t.Fatalf("restored Skill link -> %q (%v), want %q", target, readErr, externalTarget)
	}
	if content, readErr := os.ReadFile(marker); readErr != nil || string(content) != "preserve me\n" {
		t.Fatalf("external target changed: %v\n%s", readErr, content)
	}
}

func TestRemoveSkillLinksRejectsLinkChangedAfterPreflight(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "skill")
	oldTarget, changedTarget := filepath.Join(root, "old"), filepath.Join(root, "changed")
	if err := os.Symlink(changedTarget, path); err != nil {
		t.Fatal(err)
	}

	if _, err := removeSkillLinks([]linkPlan{{path: path, target: oldTarget}}); err == nil || !strings.Contains(err.Error(), "changed Skill link") {
		t.Fatalf("remove changed Skill link error = %v", err)
	}
	if target, err := os.Readlink(path); err != nil || target != changedTarget {
		t.Fatalf("changed Skill link -> %q (%v), want untouched %q", target, err, changedTarget)
	}
}

func TestActivateCurrentKeepsStableDirectoryAndSwitchesNestedRelease(t *testing.T) {
	dataRoot := t.TempDir()
	current := filepath.Join(dataRoot, "current")
	if err := os.MkdirAll(filepath.Join(current, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(current, "workflows", "marker")
	if err := os.WriteFile(marker, []byte("user-owned\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := activateCurrent(current, filepath.Join("..", "releases", "2.0.0")); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(current); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("current is not a stable directory: info=%v err=%v", info, err)
	}
	if target, err := os.Readlink(filepath.Join(current, "release")); err != nil || target != filepath.Join("..", "releases", "2.0.0") {
		t.Fatalf("current/release -> %q (%v)", target, err)
	}
	if content, err := os.ReadFile(marker); err != nil || string(content) != "user-owned\n" {
		t.Fatalf("live Workflow marker changed: %q (%v)", content, err)
	}
}

func TestPreflightCurrentRejectsUnsafeReleaseEntry(t *testing.T) {
	for _, kind := range []string{"file", "external-link"} {
		t.Run(kind, func(t *testing.T) {
			dataRoot := t.TempDir()
			releasesRoot := filepath.Join(dataRoot, "releases")
			current := filepath.Join(dataRoot, "current")
			if err := os.MkdirAll(filepath.Join(current, "workflows"), 0o755); err != nil {
				t.Fatal(err)
			}
			for name, target := range map[string]string{"bin": "release/bin", "entrypoints": "release/entrypoints", "release.json": "release/release.json"} {
				if err := os.Symlink(target, filepath.Join(current, name)); err != nil {
					t.Fatal(err)
				}
			}
			releasePath := filepath.Join(current, "release")
			switch kind {
			case "file":
				if err := os.WriteFile(releasePath, []byte("owned by user\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "external-link":
				if err := os.Symlink(t.TempDir(), releasePath); err != nil {
					t.Fatal(err)
				}
			}

			if err := preflightCurrent(current, releasesRoot); err == nil || !strings.Contains(err.Error(), "current/release") {
				t.Fatalf("preflightCurrent accepted unsafe release entry: %v", err)
			}
		})
	}
}

func TestAtomicSymlinkPreservesNonLinkPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "release")
	if err := os.WriteFile(path, []byte("owned by user\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := atomicSymlink(path, "../releases/2.0.0"); err == nil || !strings.Contains(err.Error(), "non-link") {
		t.Fatalf("atomicSymlink accepted non-link path: %v", err)
	}
	if content, err := os.ReadFile(path); err != nil || string(content) != "owned by user\n" {
		t.Fatalf("atomicSymlink changed user path: %v %q", err, content)
	}
}
