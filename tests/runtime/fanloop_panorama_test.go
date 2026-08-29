package runtime_test

import (
	"regexp"
	"strings"
	"testing"
)

// TestTechnicalSolutionPanoramaStagesAreFixed locks the reviewed technical solution topology.
// The panorama is a pure projection of the embedded workflow definition, so this
// is a tripwire: any accidental Step added to or removed from the TechDesign
// Stage (the "状态全景" the boss sees in Feishu) turns this test red.
func TestTechnicalSolutionPanoramaStagesAreFixed(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "Panorama"), "flow.init")

	rendered := run(binary, "card", "render", "--root", root, "--dry-run", "--view", "panorama", "--format", "lark-json")
	assertSuccess(t, rendered, "card.render")
	content := string(decodeCard(t, rendered.stdout).Data.Content)

	for stage, want := range map[string][]string{
		"问题定义": {"技术问题定义", "问题人工确认"},
		"方案推导": {"技术方案推导", "方案方向人工确认"},
		"方案成文": {"技术方案写作", "技术方案审校", "技术方案人工确认"},
	} {
		got := panoramaStageSteps(t, content, stage)
		if len(got) != len(want) {
			t.Fatalf("%s Stage has %d Steps, want %d: %v", stage, len(got), len(want), got)
		}
		for i, name := range want {
			if got[i] != name {
				t.Fatalf("%s Step %d = %q, want %q (full order %v)", stage, i, got[i], name, got)
			}
		}
	}
}

// panoramaStageSteps extracts the ordered Step labels of one Stage line from the
// "状态全景" panorama block, stripping the "✅ ", "**...**" and "（status）"
// decoration the renderer applies to completed and current Steps.
func panoramaStageSteps(t *testing.T, cardContent, stage string) []string {
	t.Helper()
	var line string
	for _, candidate := range strings.Split(cardContent, `\n`) {
		if strings.HasPrefix(candidate, stage+"：") {
			line = strings.TrimPrefix(candidate, stage+"：")
			break
		}
	}
	if line == "" {
		t.Fatalf("panorama does not contain a %q Stage line:\n%s", stage, cardContent)
	}
	decoration := regexp.MustCompile(`（[^（）]*）`)
	steps := make([]string, 0)
	for _, raw := range strings.Split(line, " → ") {
		clean := strings.TrimPrefix(raw, "✅ ")
		clean = strings.ReplaceAll(clean, "**", "")
		clean = decoration.ReplaceAllString(clean, "")
		steps = append(steps, strings.TrimSpace(clean))
	}
	return steps
}
