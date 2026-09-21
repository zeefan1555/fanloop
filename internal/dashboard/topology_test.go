package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/workflow"
)

func TestTopologyProjectsLiveWorkflowAndExactSourceAnchors(t *testing.T) {
	root := copyWorkflowTree(t)
	handler, err := NewHandler(root)
	if err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/topology?workflow="+url.QueryEscape("fanloop-maintainer"), nil))
	if response.Code != http.StatusOK {
		t.Fatalf("topology status=%d body=%s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), `"content"`) {
		t.Fatalf("topology must not expose generated source content: %s", response.Body.String())
	}

	var value topologyResponse
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	if value.Workflow.ID != "fanloop-maintainer" || len(value.Workflow.Stages) != 4 {
		t.Fatalf("workflow=%#v", value.Workflow)
	}
	steps := flattenTopologySteps(value.Workflow)
	if len(steps) != 4 || steps[0].ID != "define_verification_contract" || steps[len(steps)-1].ID != "merge_and_update_local" {
		t.Fatalf("steps=%#v", steps)
	}
	assertRoute(t, value.Routes, "forward", "define_verification_contract", "build_until_verified")
	assertRoute(t, value.Routes, "terminal", "merge_and_update_local", terminalStepID)
	assertRoute(t, value.Routes, "return", "define_verification_contract", "define_verification_contract")

	fragments, ok := value.Fragments["certify_candidate"]
	if !ok {
		t.Fatal("certify_candidate fragments are missing")
	}
	checks := []struct {
		name      string
		fragments []sourceFragment
		contains  string
	}{
		{"step", fragments.Step, "certify_candidate"},
		{"prompt", fragments.Prompt, "certify_candidate_flow"},
		{"forward", fragments.Forward, "next_step_id: merge_and_update_local"},
		{"return", fragments.Return, "back_step_id: build_until_verified"},
		{"conditions", fragments.Conditions, "independent_review_passed"},
		{"skills", fragments.Skills, "fanloop-dev-code-review"},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if len(check.fragments) == 0 {
				t.Fatal("no fragments")
			}
			found := false
			for _, fragment := range check.fragments {
				if fragment.StartLine < 1 || fragment.EndLine < fragment.StartLine {
					t.Fatalf("invalid range %#v", fragment)
				}
				content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(fragment.Path)))
				if err != nil {
					t.Fatal(err)
				}
				selected := selectLines(string(content), fragment.StartLine, fragment.EndLine)
				if strings.Contains(selected, check.contains) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("no fragment contains %q: %#v", check.contains, check.fragments)
			}
		})
	}
	assertFragmentLabel(t, fragments.Prompt, "panorama_delivery_condition")
	assertFragmentLabel(t, fragments.Skills, "fanloop-dev-panorama")
}

func TestTopologyProjectsCommonControlsForEveryStep(t *testing.T) {
	root := copyWorkflowTree(t)
	handler, err := NewHandler(root)
	if err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/topology?workflow=technical-solution-design", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("topology status=%d body=%s", response.Code, response.Body.String())
	}
	var value topologyResponse
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	for _, step := range flattenTopologySteps(value.Workflow) {
		fragments := value.Fragments[step.ID]
		for _, promptID := range []string{"step_start_control", "human_step_jump_control"} {
			assertFragmentLabel(t, fragments.Prompt, promptID)
		}
		for _, conditionID := range []string{"step_scope_confirmed", "human_step_jump_requested"} {
			assertFragmentLabel(t, fragments.Conditions, conditionID)
		}
		for _, skillID := range []string{"grill-with-docs", "human-step-jump"} {
			assertFragmentLabel(t, fragments.Skills, skillID)
		}
	}
	for _, route := range value.Routes {
		if route.Kind != "forward" && route.Kind != "terminal" && route.Kind != "return" {
			t.Fatalf("common controls must not invent graph edges: %#v", route)
		}
	}
}

func TestTopologySupportsEveryProductionWorkflow(t *testing.T) {
	root := copyWorkflowTree(t)
	handler, err := NewHandler(root)
	if err != nil {
		t.Fatal(err)
	}
	definitions, err := workflow.ListFrom(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, loaded := range definitions {
		workflowID := loaded.Ref.ID
		t.Run(workflowID, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/topology?workflow="+workflowID, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("topology status=%d body=%s", response.Code, response.Body.String())
			}
			var value topologyResponse
			if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
				t.Fatal(err)
			}
			expectedRoutes := 0
			for _, routes := range loaded.Workflow.Flows {
				expectedRoutes += len(routes)
			}
			for _, routes := range loaded.Workflow.Loops {
				expectedRoutes += len(routes)
			}
			steps := flattenTopologySteps(value.Workflow)
			if len(steps) != len(loaded.Workflow.OrderedStepIDs()) || len(value.Fragments) != len(steps) || len(value.Routes) != expectedRoutes {
				t.Fatalf("projection steps=%d fragments=%d routes=%d, want steps=%d routes=%d", len(steps), len(value.Fragments), len(value.Routes), len(loaded.Workflow.OrderedStepIDs()), expectedRoutes)
			}
		})
	}
}

func TestYAMLAnchorsExcludeSiblingSeparatorsAndComments(t *testing.T) {
	source, err := parseYAMLSource([]byte("conditions:\n  first:\n    output: one\n\n  # belongs to second\n  second:\n    output: two\n"))
	if err != nil {
		t.Fatal(err)
	}
	lines, err := findNamedRange(source, "conditions", "first")
	if err != nil {
		t.Fatal(err)
	}
	if lines != (lineRange{start: 2, end: 3}) {
		t.Fatalf("mapping range includes sibling separator/comment: %#v", lines)
	}

	source, err = parseYAMLSource([]byte("stages:\n  - id: one\n    name: One\n\n  # belongs to second\n  - id: two\n    name: Two\n"))
	if err != nil {
		t.Fatal(err)
	}
	stages, ok := mappingValue(source.root, "stages")
	if !ok {
		t.Fatal("stages are missing")
	}
	lines = sequenceItemRange(stages, 0, source.lineCount)
	if lines != (lineRange{start: 2, end: 3}) {
		t.Fatalf("sequence range includes sibling separator/comment: %#v", lines)
	}
}

func TestTopologyRejectsUnknownWorkflow(t *testing.T) {
	root := copyWorkflowTree(t)
	handler, err := NewHandler(root)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/topology?workflow=missing", nil))
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"invalid_workflow"`) {
		t.Fatalf("topology status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestTopologyRejectsAFileReplacedBySymlinkAfterStartup(t *testing.T) {
	root := copyWorkflowTree(t)
	handler, err := NewHandler(root)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "fanloop-maintainer", "workflow.yaml")
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(filepath.Dir(root), "outside-workflow.yaml")
	if err := os.WriteFile(outside, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, target); err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/topology?workflow=fanloop-maintainer", nil))
	if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), `"code":"invalid_workflow"`) {
		t.Fatalf("symlinked topology status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestEmbeddedStudioUsesLiveTopologyAndWholeFileSaves(t *testing.T) {
	page, err := assets.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		`id="workflowSelect"`,
		`id="viewport"`,
		`id="edges"`,
		`id="nodes"`,
		`id="files"`,
		`data-tab="step"`,
		`data-tab="prompt"`,
		`data-tab="forward"`,
		`data-tab="return"`,
		`data-tab="conditions"`,
		`data-tab="skills"`,
		`id="editor"`,
	} {
		if !strings.Contains(string(page), required) {
			t.Fatalf("studio page missing %q", required)
		}
	}

	script, err := assets.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	value := string(script)
	for _, required := range []string{
		"/api/topology?workflow=",
		"/api/file?path=",
		"setSelectionRange",
		"confirmDiscard()",
		"method: 'POST'",
		"selectedFragment",
		"loadTopology",
		"renderGraph",
	} {
		if !strings.Contains(value, required) {
			t.Fatalf("studio script missing %q", required)
		}
	}
	for _, forbidden := range []string{"const steps = [", "const snippets =", "savePrototype"} {
		if strings.Contains(value, forbidden) {
			t.Fatalf("studio script contains prototype-only behavior %q", forbidden)
		}
	}
	if strings.Count(value, "method: 'POST'") != 1 || !strings.Contains(value, "json('/api/file', { method: 'POST'") {
		t.Fatal("POST /api/file must remain the sole browser write path")
	}

	styles, err := assets.ReadFile("static/style.css")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(styles), "[hidden] { display: none !important; }") {
		t.Fatal("loaded graph must hide its loading overlay")
	}
}

func flattenTopologySteps(value topologyWorkflow) []topologyStep {
	var result []topologyStep
	for _, stage := range value.Stages {
		for _, job := range stage.Jobs {
			result = append(result, job.Steps...)
		}
	}
	return result
}

func assertRoute(t *testing.T, routes []topologyRoute, kind, from, to string) {
	t.Helper()
	for _, route := range routes {
		if route.Kind == kind && route.FromStepID == from && route.ToStepID == to {
			return
		}
	}
	t.Fatalf("missing %s route %s -> %s in %#v", kind, from, to, routes)
}

func assertFragmentLabel(t *testing.T, fragments []sourceFragment, label string) {
	t.Helper()
	for _, fragment := range fragments {
		if fragment.Label == label {
			return
		}
	}
	t.Fatalf("missing fragment %q in %#v", label, fragments)
}

func selectLines(content string, start, end int) string {
	lines := strings.Split(content, "\n")
	if start > len(lines) {
		return ""
	}
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[start-1:end], "\n")
}
