package workflow

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestExampleWorkflowValidatesWithoutReservedPanoramaCondition(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate repository")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "examples", "workflows")
	loaded, err := LoadFrom(root, "example")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"panorama_presented", "panorama_card_published"} {
		if _, exists := loaded.Workflow.Conditions[id]; exists {
			t.Fatalf("example still defines reserved condition %s", id)
		}
	}
	if len(loaded.Workflow.OrderedStepIDs()) == 0 {
		t.Fatal("example workflow has no steps")
	}
}

func TestManageFanloopWorkflowPackageValidates(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate repository")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "examples", "workflows")
	loaded, err := LoadFrom(root, "manage-fanloop-workflow")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Workflow.ID != "manage-fanloop-workflow" {
		t.Fatalf("workflow id = %s", loaded.Workflow.ID)
	}
	stages := []string{}
	for _, stage := range loaded.Workflow.Stages {
		stages = append(stages, stage.ID)
	}
	if strings.Join(stages, ",") != "analyze,build,review,publish" {
		t.Fatalf("stages = %v", stages)
	}
	if got := loaded.Workflow.OrderedStepIDs(); len(got) != 4 {
		t.Fatalf("steps = %v", got)
	}
	for _, id := range []string{"panorama_presented", "panorama_card_published"} {
		if _, exists := loaded.Workflow.Conditions[id]; exists {
			t.Fatalf("package defines reserved condition %s", id)
		}
	}
	wantSkills := []string{
		"analyze-workflow-change",
		"build-workflow-candidate",
		"explain-workflow-design",
		"manage-workflow-document",
		"publish-workflow-candidate",
	}
	if len(loaded.SkillPaths) != len(wantSkills) {
		t.Fatalf("skills = %#v", loaded.SkillPaths)
	}
	for _, id := range wantSkills {
		if loaded.SkillPaths[id] == "" {
			t.Fatalf("missing skill %s", id)
		}
	}
	packageRoot := filepath.Join(root, "manage-fanloop-workflow")
	var combined strings.Builder
	err = filepath.WalkDir(packageRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		combined.Write(content)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	text := combined.String()
	for _, required := range []string{
		"7543941556223377409",
		"fanloop.workflow.",
		"fanloop update config",
		"fanloop doctor",
		"pending_actions",
		"lark-cli docs +create --as user",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("package is missing %q", required)
		}
	}
	if !strings.Contains(text, "不传 AK/SK") || !strings.Contains(text, "AK/SK、JWT、签名和下载 token 不进入") {
		t.Fatal("package does not forbid embedded Fornax credentials")
	}
	if regexp.MustCompile(`(?:^|[^A-Za-z])loop (?:update|doctor)|(?:^|[^A-Za-z])loop-cli|(?:^|[^A-Za-z])loop\.workflow\.`).MatchString(text) {
		t.Fatal("package still uses a loop_cli command, key prefix, or product name")
	}
	if strings.Count(text, "7590123861772503042") != 2 {
		t.Fatalf("team workspace mentions = %d, want the two prohibition lines", strings.Count(text, "7590123861772503042"))
	}
	if !strings.Contains(text, "不得查询或写入团队空间 `7590123861772503042`") || !strings.Contains(text, "不得读取或写入团队空间\n`7590123861772503042`") {
		t.Fatal("team workspace id is not limited to prohibition text")
	}
}
