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
		{Name: "ai-test", Path: "skills/common/ai-test"},
		{Name: release.ExposedSkillName, Path: "skills/common/fanloop-workflow"},
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
		if filepath.Base(plan.path) != release.ExposedSkillName || plan.target != filepath.Join(root, "current", "skills", "common", "fanloop-workflow") {
			t.Fatalf("unexpected link plan: %#v", plan)
		}
	}
}

func TestCurrentSkillsPreserveGroupedTargets(t *testing.T) {
	root := t.TempDir()
	manifest := release.Manifest{Skills: []*release.Skill{
		{Name: "retired", Path: "skills/common/retired"},
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
	if len(got) != 2 || got[0].name != "retired" || got[0].path != "skills/common/retired" || got[1].name != "internal" || got[1].path != "skills/fanloop-maintainer/internal" {
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
