package skillconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/workflow"
)

func TestValidateProductionConfiguration(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	definitions, err := workflow.List()
	if err != nil {
		t.Fatal(err)
	}
	skills, err := Validate(root, definitions)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) == 0 {
		t.Fatal("production Skill configuration is empty")
	}
	paths, err := Paths(root, "technical-solution-design")
	if err != nil {
		t.Fatal(err)
	}
	if path := paths["technical-background-framing"]; !filepath.IsAbs(path) {
		t.Fatalf("Skill path = %q, want absolute path", path)
	}
}

func TestValidateRejectsMissingBoundSkill(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "skills", "example", "present"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "skills", "example", "present", "SKILL.md"), []byte("example\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	definitions := []workflow.Loaded{{
		Ref: workflow.Ref{ID: "example"},
		Workflow: workflow.Workflow{ID: "example", Prompts: map[string]workflow.PromptDefinition{
			"step": {Skills: []workflow.SkillBinding{{ID: "missing"}}},
		}},
	}}
	if _, err := Validate(root, definitions); err == nil || !strings.Contains(err.Error(), `unknown Skill "missing"`) {
		t.Fatalf("Validate() error = %v", err)
	}
}
