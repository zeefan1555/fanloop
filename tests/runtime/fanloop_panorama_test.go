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
		"问题定义": {"需求背景", "核心问题", "设计目标", "问题审核"},
		"方案设计": {"方案调研", "总体方案", "难点解法", "方案审核"},
		"方案成文": {"方案收益", "落地规划", "方案成文", "方案审校", "方案终审"},
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

func TestTechnicalSolutionInitialPromptExposesEvidenceContract(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	initialized := run(binary, "flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "Evidence Contract")
	assertSuccess(t, initialized, "flow.init")
	content := string(initialized.stdout)
	for _, want := range []string{"具体业务场景", "定量事实", "来源和证据状态"} {
		if !strings.Contains(content, want) {
			t.Errorf("flow.init response does not expose %q:\n%s", want, content)
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
