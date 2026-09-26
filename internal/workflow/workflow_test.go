package workflow

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestListFromLoadsArbitraryWorkflowInventory(t *testing.T) {
	root := t.TempDir()
	want := []string{"alpha-loop", "beta-loop", "delta-loop", "gamma-loop"}
	for _, id := range want {
		writeValidWorkflow(t, root, id)
	}

	items, err := ListFrom(root)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(items))
	for index, item := range items {
		got[index] = item.Ref.ID
		if item.SkillPaths["writer"] == "" {
			t.Fatalf("%s did not resolve its bound Skill", item.Ref.ID)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Workflow IDs = %v, want %v", got, want)
	}
}

func TestRuntimeFixtureSourcesAreFlatAndVersionless(t *testing.T) {
	root := "testdata"
	if _, err := os.Stat(filepath.Join(root, "defaults.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("defaults.json must not exist: %v", err)
	}
	paths, err := filepath.Glob(filepath.Join(root, "*", "workflow.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("flat test Workflow sources = %v, want two", paths)
	}
	versioned, err := filepath.Glob(filepath.Join(root, "*", "*", "workflow.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(versioned) != 0 {
		t.Fatalf("versioned production Workflow sources remain: %v", versioned)
	}
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(content), "\nversion:") {
			t.Fatalf("%s retains Workflow business version", path)
		}
	}
}

func TestWorkflowSelectorRejectsVersionedID(t *testing.T) {
	if _, err := LoadSelector("fanloop@14.0.0"); !errors.Is(err, ErrInvalidSelector) {
		t.Fatalf("versioned selector error = %v, want ErrInvalidSelector", err)
	}
}

func TestLoadUsesLiveWorkflowRootOnEveryCall(t *testing.T) {
	dataRoot := t.TempDir()
	liveRoot := filepath.Join(dataRoot, "current", "workflows")
	workflowID := "reloadable-loop"
	targetRoot := filepath.Join(liveRoot, workflowID)
	writeValidWorkflow(t, liveRoot, workflowID)
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)

	before, err := Load(workflowID)
	if err != nil {
		t.Fatal(err)
	}
	promptPath := filepath.Join(targetRoot, "prompt.yaml")
	promptYAML, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := bytes.Replace(promptYAML, []byte("Write the note"), []byte("Write the live note"), 1)
	if bytes.Equal(updated, promptYAML) {
		t.Fatal("test fixture prompt replacement did not match")
	}
	if err := os.WriteFile(promptPath, updated, 0o600); err != nil {
		t.Fatal(err)
	}

	after, err := Load(workflowID)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(after.Workflow.Prompts, before.Workflow.Prompts) {
		t.Fatal("second Load did not observe the edited live prompt.yaml")
	}
	if err := os.WriteFile(promptPath, []byte("invalid: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(workflowID); err == nil {
		t.Fatal("invalid live prompt.yaml did not fail closed")
	}
}

func writeValidWorkflow(t *testing.T, root, id string) {
	t.Helper()
	bundleRoot := filepath.Join(root, id)
	if err := os.MkdirAll(filepath.Join(bundleRoot, "skills", "writer"), 0o755); err != nil {
		t.Fatal(err)
	}
	workflowYAML, flowYAML, conditionYAML, loopYAML, promptYAML := validBundle()
	workflowYAML = bytes.Replace(workflowYAML, []byte("id: sample"), []byte("id: "+id), 1)
	for name, content := range map[string][]byte{
		"workflow.yaml": workflowYAML, "flow.yaml": flowYAML, "condition.yaml": conditionYAML,
		"loop.yaml": loopYAML, "prompt.yaml": promptYAML,
	} {
		if err := os.WriteFile(filepath.Join(bundleRoot, name), content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(bundleRoot, "skills", "writer", "SKILL.md"), []byte("---\nname: writer\n---\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestListFromRejectsSymlinkedLiveRootChain(t *testing.T) {
	source := "testdata"
	for _, test := range []struct {
		name string
		root func(*testing.T) string
	}{
		{
			name: "Workflow root",
			root: func(t *testing.T) string {
				realRoot := filepath.Join(t.TempDir(), "real-workflows")
				if err := os.CopyFS(realRoot, os.DirFS(source)); err != nil {
					t.Fatal(err)
				}
				link := filepath.Join(t.TempDir(), "workflows")
				if err := os.Symlink(realRoot, link); err != nil {
					t.Fatal(err)
				}
				return link
			},
		},
		{
			name: "current ancestor",
			root: func(t *testing.T) string {
				dataRoot := t.TempDir()
				legacy := filepath.Join(dataRoot, "releases", "legacy")
				if err := os.CopyFS(filepath.Join(legacy, "workflows"), os.DirFS(source)); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(legacy, filepath.Join(dataRoot, "current")); err != nil {
					t.Fatal(err)
				}
				return filepath.Join(dataRoot, "current", "workflows")
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ListFrom(test.root(t)); err == nil || !strings.Contains(err.Error(), "real directory") {
				t.Fatalf("ListFrom accepted symlinked live root: %v", err)
			}
		})
	}
}

func TestBundleRejectsInvalidFiveFileRelationships(t *testing.T) {
	workflowYAML, flowYAML, conditionYAML, loopYAML, promptYAML := validBundle()
	for name, mutate := range map[string]func() ([]byte, []byte, []byte, []byte, []byte){
		"old Stage Steps": func() ([]byte, []byte, []byte, []byte, []byte) {
			return []byte(strings.Replace(string(workflowYAML), "jobs:", "steps:", 1)), flowYAML, conditionYAML, loopYAML, promptYAML
		},
		"missing Step Flow": func() ([]byte, []byte, []byte, []byte, []byte) {
			return workflowYAML, []byte(strings.Replace(string(flowYAML), "  confirm_note:\n", "  omitted:\n", 1)), conditionYAML, loopYAML, promptYAML
		},
		"unknown Prompt": func() ([]byte, []byte, []byte, []byte, []byte) {
			return workflowYAML, []byte(strings.Replace(string(flowYAML), "write_note_flow", "missing_prompt", 1)), conditionYAML, loopYAML, promptYAML
		},
		"unknown Condition": func() ([]byte, []byte, []byte, []byte, []byte) {
			return workflowYAML, []byte(strings.Replace(string(flowYAML), "note_written", "missing_condition", 1)), conditionYAML, loopYAML, promptYAML
		},
		"unknown Output type": func() ([]byte, []byte, []byte, []byte, []byte) {
			return workflowYAML, flowYAML, []byte(strings.Replace(string(conditionYAML), "type: path", "type: script", 1)), loopYAML, promptYAML
		},
		"unknown Output source": func() ([]byte, []byte, []byte, []byte, []byte) {
			return workflowYAML, flowYAML, []byte(strings.Replace(string(conditionYAML), "type: path", "type: path, source: unknown", 1)), loopYAML, promptYAML
		},
		"verify evidence source on path": func() ([]byte, []byte, []byte, []byte, []byte) {
			return workflowYAML, flowYAML, []byte(strings.Replace(string(conditionYAML), "type: path", "type: path, source: verify.evidence.outcome", 1)), loopYAML, promptYAML
		},
		"verify evidence source on disallowed enum": func() ([]byte, []byte, []byte, []byte, []byte) {
			return workflowYAML, flowYAML, []byte(strings.Replace(string(conditionYAML), "values: [approved]", "values: [approved], source: verify.evidence.outcome", 1)), loopYAML, promptYAML
		},
		"forward back": func() ([]byte, []byte, []byte, []byte, []byte) {
			return workflowYAML, flowYAML, conditionYAML, []byte(strings.Replace(string(loopYAML), "back_step_id: write_note", "back_step_id: confirm_note", 1)), promptYAML
		},
		"optional omitted": func() ([]byte, []byte, []byte, []byte, []byte) {
			return workflowYAML, flowYAML, conditionYAML, loopYAML, []byte(strings.Replace(string(promptYAML), "        optional: false\n", "", 1))
		},
		"required skills omitted": func() ([]byte, []byte, []byte, []byte, []byte) {
			return workflowYAML, flowYAML, conditionYAML, loopYAML, []byte(strings.Replace(string(promptYAML), "    skills: []\n", "", 1))
		},
		"empty AND group": func() ([]byte, []byte, []byte, []byte, []byte) {
			return workflowYAML, []byte(strings.Replace(string(flowYAML), "- [note_written]", "- []", 1)), conditionYAML, loopYAML, promptYAML
		},
		"exclusive AND group": func() ([]byte, []byte, []byte, []byte, []byte) {
			return workflowYAML, []byte(strings.Replace(string(flowYAML), "- [note_approved]", "- [note_approved, note_rejected]", 1)), conditionYAML, loopYAML, promptYAML
		},
		"trailing YAML document": func() ([]byte, []byte, []byte, []byte, []byte) {
			return append(workflowYAML, []byte("---\nid: trailing\n")...), flowYAML, conditionYAML, loopYAML, promptYAML
		},
	} {
		t.Run(name, func(t *testing.T) {
			one, two, three, four, five := mutate()
			if _, err := DecodeBundle(one, two, three, four, five); err == nil {
				t.Fatal("expected Bundle rejection")
			}
		})
	}
}

func TestFlowAllowsExplicitlySelectedTargetsWithSharedConditions(t *testing.T) {
	conditions := map[string]ConditionDefinition{
		"jump": {ExclusiveGroup: "decision"},
	}
	routes := []FlowRoute{
		{When: When{AnyOf: [][]string{{"jump"}}}, NextStepID: "second"},
		{When: When{AnyOf: [][]string{{"jump"}}}, NextStepID: "third"},
	}
	if err := validateFlowAmbiguity("first", routes, conditions); err != nil {
		t.Fatalf("explicitly selected targets rejected: %v", err)
	}
	routes[1].NextStepID = "second"
	if err := validateFlowAmbiguity("first", routes, conditions); err == nil {
		t.Fatal("overlapping Routes to the same target were accepted")
	}
}

func TestRuntimeWorkflowModelDoesNotOwnYAMLTags(t *testing.T) {
	for _, value := range []any{
		Stage{}, Job{}, Step{}, PromptRef{}, When{}, FlowRoute{}, LoopRoute{},
		ConditionDefinition{}, OutputDefinition{}, PromptDefinition{}, SkillBinding{},
	} {
		typeOf := reflect.TypeOf(value)
		for index := range typeOf.NumField() {
			if tag := typeOf.Field(index).Tag.Get("yaml"); tag != "" {
				t.Errorf("%s.%s retains YAML tag %q", typeOf.Name(), typeOf.Field(index).Name, tag)
			}
		}
	}
}

func TestLoadDirectoryRequiresTheFixedFiveFiles(t *testing.T) {
	root := t.TempDir()
	workflowYAML, flowYAML, _, loopYAML, promptYAML := validBundle()
	for name, content := range map[string][]byte{
		"workflow.yaml": workflowYAML, "flow.yaml": flowYAML,
		"loop.yaml": loopYAML, "prompt.yaml": promptYAML,
	} {
		if err := os.WriteFile(filepath.Join(root, name), content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := LoadDirectory(root); err == nil || !strings.Contains(err.Error(), "condition.yaml") {
		t.Fatalf("error = %v", err)
	}
}

func TestLoadDirectoryRejectsAnythingBeyondTheFixedFiveFiles(t *testing.T) {
	for _, extra := range []string{"guard.yaml", "README.md"} {
		t.Run(extra, func(t *testing.T) {
			root := t.TempDir()
			workflowYAML, flowYAML, conditionYAML, loopYAML, promptYAML := validBundle()
			for name, content := range map[string][]byte{
				"workflow.yaml":  workflowYAML,
				"flow.yaml":      flowYAML,
				"condition.yaml": conditionYAML,
				"loop.yaml":      loopYAML,
				"prompt.yaml":    promptYAML,
				extra:            []byte("unexpected\n"),
			} {
				if err := os.WriteFile(filepath.Join(root, name), content, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := LoadDirectory(root); err == nil || !strings.Contains(err.Error(), extra) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestOutputTypesAndBounds(t *testing.T) {
	minimum, maximum := int64(1), int64(2)
	tests := []struct {
		definition OutputDefinition
		value      string
		valid      bool
	}{
		{OutputDefinition{Type: OutputPath}, `"notes/a.md"`, true},
		{OutputDefinition{Type: OutputURL}, `"https://example.com/a"`, true},
		{OutputDefinition{Type: OutputBoolean}, `true`, true},
		{OutputDefinition{Type: OutputURLList}, `["https://example.com/a"]`, true},
		{OutputDefinition{Type: OutputEnum, Values: []string{"ready"}}, `"ready"`, true},
		{OutputDefinition{Type: OutputEnum, Values: []string{"passed"}, Source: OutputSourceVerifyEvidenceOutcome}, `"passed"`, true},
		{OutputDefinition{Type: OutputInteger, Minimum: &minimum, Maximum: &maximum}, `2`, true},
		{OutputDefinition{Type: OutputObject}, `{"ok":true}`, true},
		{OutputDefinition{Type: OutputPath}, `"/tmp/outside.md"`, false},
		{OutputDefinition{Type: OutputPath}, `"../outside.md"`, false},
		{OutputDefinition{Type: OutputBoolean}, `"true"`, false},
		{OutputDefinition{Type: OutputInteger, Minimum: &minimum, Maximum: &maximum}, `3`, false},
		{OutputDefinition{Type: OutputObject}, `[]`, false},
	}
	for index, test := range tests {
		err := ValidateOutput(test.definition, json.RawMessage(test.value))
		if (err == nil) != test.valid {
			t.Fatalf("case %d: error = %v, valid = %t", index, err, test.valid)
		}
	}
}

func TestVerifyEvidenceOutcomeSourceLoadsOnPassedEnum(t *testing.T) {
	workflowYAML, flowYAML, conditionYAML, loopYAML, promptYAML := validBundle()
	conditionYAML = []byte(strings.Replace(string(conditionYAML),
		"output: {key: note_status, type: enum_value, values: [rework]}",
		"output: {key: note_status, type: enum_value, values: [passed], source: verify.evidence.outcome}",
		1))
	loaded, err := DecodeBundle(workflowYAML, flowYAML, conditionYAML, loopYAML, promptYAML)
	if err != nil {
		t.Fatal(err)
	}
	condition, ok := loaded.Workflow.Condition("note_rework")
	if !ok || condition.Output.Source != OutputSourceVerifyEvidenceOutcome || !reflect.DeepEqual(condition.Output.Values, []string{"passed"}) {
		t.Fatalf("loaded output = %#v, ok = %t", condition.Output, ok)
	}
}

func validBundle() ([]byte, []byte, []byte, []byte, []byte) {
	workflowYAML := []byte(`schema_version: 7
id: sample
stages:
  - id: notes
    name: Notes
    jobs:
      - id: notes
        name: Notes
        steps:
          - id: write_note
            name: Write note
            executor: agent
          - id: confirm_note
            name: Confirm note
            executor: human
`)
	flowYAML := []byte(`schema_version: 5
flow:
  write_note:
    - prompt_ref: {file: prompt.yaml, prompt_id: write_note_flow}
      when:
        any_of:
          - [note_written]
      next_step_id: confirm_note
  confirm_note:
    - prompt_ref: {file: prompt.yaml, prompt_id: confirm_note_flow}
      when:
        any_of:
          - [note_approved]
      terminal: true
`)
	conditionYAML := []byte(`schema_version: 3
conditions:
  note_written:
    prompt_ref: {file: prompt.yaml, prompt_id: note_condition}
    output: {key: note_path, type: path}
    exclusive_group: note_outcome
  note_rework:
    prompt_ref: {file: prompt.yaml, prompt_id: note_condition}
    output: {key: note_status, type: enum_value, values: [rework]}
    exclusive_group: note_outcome
  note_approved:
    prompt_ref: {file: prompt.yaml, prompt_id: approval_condition}
    output: {key: note_decision, type: enum_value, values: [approved]}
    exclusive_group: approval_outcome
  note_rejected:
    prompt_ref: {file: prompt.yaml, prompt_id: approval_condition}
    output: {key: note_decision, type: enum_value, values: [rejected]}
    exclusive_group: approval_outcome
`)
	loopYAML := []byte(`schema_version: 4
loop:
  write_note:
    - prompt_ref: {file: prompt.yaml, prompt_id: back_to_write}
      when:
        any_of:
          - [note_rework]
      back_step_id: write_note
  confirm_note:
    - prompt_ref: {file: prompt.yaml, prompt_id: back_to_write}
      when:
        any_of:
          - [note_rejected]
      back_step_id: write_note
`)
	promptYAML := []byte(`schema_version: 2
prompts:
  write_note_flow:
    prompt: Write the note
    skills:
      - id: writer
        prompt: Use the writer Skill.
        optional: false
  confirm_note_flow:
    prompt: Wait for approval.
    skills: []
  note_condition:
    prompt: Report whether the note is written or needs rework.
    skills: []
  approval_condition:
    prompt: Report the approval decision.
    skills: []
  back_to_write:
    prompt: Return to write_note and revise the note.
    skills: []
`)
	return workflowYAML, flowYAML, conditionYAML, loopYAML, promptYAML
}
