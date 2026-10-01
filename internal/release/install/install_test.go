package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/release"
)

func TestPreflightSkillLinksExposesBothPublicSkills(t *testing.T) {
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
		{Name: "test-guide", Path: "skills/linear-runtime/test-guide"},
		{Name: release.WorkflowSkillName, Path: release.WorkflowSkillPath},
		{Name: release.AuthoringSkillName, Path: release.AuthoringSkillPath},
		{Name: "fanloop-dev-tdd", Path: "skills/fanloop-maintainer/fanloop-dev-tdd"},
	}}
	plans, err := preflightSkillLinks(request, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 8 {
		t.Fatalf("plans = %#v", plans)
	}
	for index, plan := range plans {
		exposed := release.ExposedSkills()[index/4]
		if filepath.Base(plan.path) != exposed.Name || plan.target != filepath.Join(root, "current", filepath.FromSlash(exposed.Path)) {
			t.Fatalf("unexpected link plan: %#v", plan)
		}
	}
}

func TestCurrentSkillsPreserveGroupedTargets(t *testing.T) {
	root := t.TempDir()
	manifest := release.Manifest{Skills: []*release.Skill{
		{Name: "retired", Path: "skills/linear-runtime/retired"},
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
	if len(got) != 2 || got[0].name != "retired" || got[0].path != "skills/linear-runtime/retired" || got[1].name != "internal" || got[1].path != "skills/fanloop-maintainer/internal" {
		t.Fatalf("currentSkills() = %#v", got)
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

func TestActivateInstallationRollsBackBindingWhenCurrentConflicts(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprint(existing), func(t *testing.T) {
			root := t.TempDir()
			binding := filepath.Join(root, "config", "current")
			previous := t.TempDir()
			if existing {
				if err := atomicSymlink(binding, previous); err != nil {
					t.Fatal(err)
				}
			}
			current := filepath.Join(root, "current")
			if err := os.WriteFile(current, []byte("user owned"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := activateInstallation(root, t.TempDir(), "../releases/new"); err == nil {
				t.Fatal("conflicting current was accepted")
			}
			got, err := os.Readlink(binding)
			if existing && (err != nil || got != previous) || !existing && !os.IsNotExist(err) {
				t.Fatalf("binding after failed activation = %q, %v", got, err)
			}
			if content, err := os.ReadFile(current); err != nil || string(content) != "user owned" {
				t.Fatalf("current changed: %q, %v", content, err)
			}
		})
	}
}

func TestActivateInstallationBindsSourceWithoutCopyingPackages(t *testing.T) {
	root, source := t.TempDir(), t.TempDir()
	if err := activateInstallation(root, source, "../releases/new"); err != nil {
		t.Fatal(err)
	}
	if got, err := os.Readlink(filepath.Join(root, "config", "current")); err != nil || got != source {
		t.Fatalf("binding = %q, %v", got, err)
	}
	if _, err := os.Lstat(filepath.Join(root, "current", "workflows")); !os.IsNotExist(err) {
		t.Fatalf("unexpected copied configuration: %v", err)
	}
}
