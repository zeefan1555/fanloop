package workflow

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestExampleWorkflowValidatesWithWorkflowOwnedPanoramaCondition(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate repository")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "examples", "workflows")
	loaded, err := LoadFrom(root, "example")
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := loaded.Workflow.Conditions["panorama_presented"]; !exists {
		t.Fatal("example does not define its panorama condition")
	}
	if loaded.SkillPaths["example-panorama"] == "" {
		t.Fatal("example does not bind its panorama delivery Skill")
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
	wantSkills := []string{
		"analyze-workflow-change",
		"build-workflow-candidate",
		"explain-workflow-design",
		"manage-fanloop-panorama",
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
	if _, exists := loaded.Workflow.Conditions["panorama_presented"]; !exists {
		t.Fatal("manage-fanloop-workflow does not declare its Panorama condition")
	}
	for stepID, routes := range loaded.Workflow.Flows {
		for _, route := range routes {
			assertEveryGroupContains(t, stepID+" flow", route.When, "panorama_presented")
		}
	}
	for stepID, routes := range loaded.Workflow.Loops {
		for _, route := range routes {
			assertEveryGroupContains(t, stepID+" loop", route.When, "panorama_presented")
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
	panoramaSkill := string(mustReadFile(t, filepath.Join(packageRoot, "skills", "manage-fanloop-panorama", "SKILL.md")))
	for _, required := range []string{
		"### 完成态 botmux",
		"### 完成态 local_agent",
		"### 完成态 aime",
		"### 完成态 aiden",
		"botmux send --card-file",
		"lark-cli im +messages-reply",
		"aiden-bot-cli send-card --card-file",
		"after a successful terminal Route, render panorama once more",
	} {
		if !strings.Contains(panoramaSkill, required) {
			t.Errorf("manage Panorama Skill is missing %q", required)
		}
	}
	if got := strings.Count(panoramaSkill, "fanloop card render --root <ABSOLUTE_REQUIREMENT_ROOT> --view panorama"); got < 8 {
		t.Fatalf("manage Panorama Skill render commands = %d, want initial and completed commands for four hosts", got)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func assertEveryGroupContains(t *testing.T, owner string, when When, conditionID string) {
	t.Helper()
	for index, group := range when.AnyOf {
		found := false
		for _, id := range group {
			found = found || id == conditionID
		}
		if !found {
			t.Fatalf("%s route group %d does not contain %s: %v", owner, index, conditionID, group)
		}
	}
}
