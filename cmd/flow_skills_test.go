package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/idl/flowidl"
)

func TestFlowSkillsDiscoversUnboundSkillsFromCurrentSourceWithoutBusinessWrites(t *testing.T) {
	source, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	packageRoot := filepath.Join(source, "skills", "fanloop-workflow", syntheticWorkflowID)
	if err := os.CopyFS(packageRoot, os.DirFS(filepath.Join("..", "internal", "workflow", "testdata", syntheticWorkflowID))); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FANLOOP_CONFIG_SOURCE", source)
	root := t.TempDir()
	run := func(args ...string) (int, string, string) {
		t.Helper()
		var stdout, stderr bytes.Buffer
		code := Execute(context.Background(), args, strings.NewReader(""), &stdout, &stderr)
		return code, stdout.String(), stderr.String()
	}
	if code, _, stderr := run("flow", "init", "--root", root, "--workflow", syntheticWorkflowID, "--title", "Skill discovery"); code != 0 {
		t.Fatal(stderr)
	}
	before := catalogBusinessFiles(t, root)
	skillRoot := filepath.Join(packageRoot, "skills")
	writeSkill := func(id, content string) {
		t.Helper()
		dir := filepath.Join(skillRoot, id)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	writeSkill("aaa-unbound", "---\nname: different-name\ndescription: |\n  Explain a design.\n  Read why when needed.\n---\n# Skill\n")
	writeSkill("zzz-no-metadata", "# An existing legal skill\n")
	writeSkill("yyy-unusable-metadata", "---\ndescription: [not, a, string]\n---\n# Skill\n")
	read := func() *flowidl.FlowSkillsResponse {
		t.Helper()
		code, stdout, stderr := run("flow", "skills", "--root", root, "--input", "{}")
		if code != 0 {
			t.Fatalf("skills exit=%d stderr=%s", code, stderr)
		}
		var envelope struct{ Data *flowidl.FlowSkillsResponse }
		if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
			t.Fatal(err)
		}
		return envelope.Data
	}
	response := read()
	if response.PackageRoot != packageRoot || response.SkillRoot != skillRoot || response.Workflow.Id != syntheticWorkflowID {
		t.Fatalf("catalog roots: %#v", response)
	}
	if response.Skills[0].Id != "aaa-unbound" || response.Skills[0].Description != "Explain a design.\nRead why when needed.\n" {
		t.Fatalf("unbound description = %#v", response.Skills[0])
	}
	last := ""
	for _, skill := range response.Skills {
		if skill.Id <= last || skill.Path != filepath.Join(skillRoot, skill.Id, "SKILL.md") || !filepath.IsAbs(skill.Path) {
			t.Fatalf("catalog order or path: %#v", skill)
		}
		if strings.HasPrefix(skill.Id, "yyy-") || strings.HasPrefix(skill.Id, "zzz-") {
			if skill.Description != "" {
				t.Fatalf("unusable metadata must be empty: %#v", skill)
			}
		}
		last = skill.Id
	}
	writeSkill("aaa-unbound", "---\r\ndescription: Updated live guidance\r\n---\r\n# Skill\r\n")
	if got := read().Skills[0].Description; got != "Updated live guidance" {
		t.Fatalf("live description = %q", got)
	}
	writeSkill("aaa-unbound", "---\nsummary: &summary Resolve referenced Skills.\ndescription: *summary\n---\n# Skill\n")
	if got := read().Skills[0].Description; got != "Resolve referenced Skills." {
		t.Fatalf("string-alias description = %q", got)
	}
	writeSkill("aaa-unbound", "---\ndescription: true\n---\n# Skill\n")
	if got := read().Skills[0].Description; got != "" {
		t.Fatalf("non-string description = %q", got)
	}
	writeSkill("aaa-unbound", "---\ndescription: [\n---\n# Malformed optional metadata\n")
	if got := read().Skills[0].Description; got != "" {
		t.Fatalf("malformed metadata description = %q", got)
	}
	if after := catalogBusinessFiles(t, root); !reflect.DeepEqual(after, before) {
		t.Fatal("catalog reads changed business files")
	}
	for _, args := range [][]string{
		{"flow", "skills", "--root", root, "--input", `{"unknown":true}`},
		{"flow", "skills", "--root", root, "--dry-run"},
	} {
		if code, _, stderr := run(args...); code != 2 || !strings.Contains(stderr, "INVALID_") {
			t.Fatalf("invalid request exit=%d stderr=%s", code, stderr)
		}
	}
	if code, _, stderr := run("flow", "skills", "--root", t.TempDir()); code != 1 || !strings.Contains(stderr, "NOT_INITIALIZED") {
		t.Fatalf("uninitialized exit=%d stderr=%s", code, stderr)
	}
	for _, invalid := range []string{"", "relative", filepath.Join(source, "missing")} {
		t.Run("bad-source-"+invalid, func(t *testing.T) {
			t.Setenv("FANLOOP_CONFIG_SOURCE", invalid)
			if code, _, stderr := run("flow", "skills", "--root", root); code != 5 || !strings.Contains(stderr, "INTERNAL") {
				t.Fatalf("bad source exit=%d stderr=%s", code, stderr)
			}
		})
	}
	if err := os.Remove(filepath.Join(skillRoot, "aaa-unbound", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if code, _, stderr := run("flow", "skills", "--root", root); code != 5 || !strings.Contains(stderr, "INTERNAL") {
		t.Fatalf("broken package exit=%d stderr=%s", code, stderr)
	}
	if after := catalogBusinessFiles(t, root); !reflect.DeepEqual(after, before) {
		t.Fatal("rejected reads changed business files")
	}
}

func catalogBusinessFiles(t *testing.T, root string) map[string][32]byte {
	t.Helper()
	files := map[string][32]byte{}
	if err := filepath.WalkDir(filepath.Join(root, ".fanloop"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || strings.Contains(path, string(filepath.Separator)+"log"+string(filepath.Separator)) {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[path] = sha256.Sum256(content)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return files
}
