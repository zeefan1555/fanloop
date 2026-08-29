package workflow

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestProductionWorkflowsAreValidFiveFileBundles(t *testing.T) {
	items, err := List()
	if err != nil {
		t.Fatal(err)
	}
	refs := make([]string, len(items))
	for index, item := range items {
		refs[index] = item.Ref.ID
	}
	wantRefs := []string{"fanloop-maintainer", "technical-solution-design"}
	if !reflect.DeepEqual(refs, wantRefs) {
		t.Fatalf("Workflow IDs = %v", refs)
	}
	for _, item := range items {
		if len(item.Workflow.OrderedStepIDs()) != len(item.Workflow.Flows) || len(item.Workflow.OrderedStepIDs()) != len(item.Workflow.Loops) {
			t.Fatalf("%s has steps=%d flow=%d loop=%d", item.Ref.ID, len(item.Workflow.OrderedStepIDs()), len(item.Workflow.Flows), len(item.Workflow.Loops))
		}
		if !strings.HasPrefix(item.Ref.Digest, "sha256:") {
			t.Fatalf("invalid digest %q", item.Ref.Digest)
		}
	}
	loaded, err := Load("technical-solution-design")
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Workflow.OrderedStepIDs()) != 7 || len(loaded.Workflow.Conditions) != 17 {
		t.Fatalf("real Bundle shape = steps:%d conditions:%d", len(loaded.Workflow.OrderedStepIDs()), len(loaded.Workflow.Conditions))
	}
	pinned, err := LoadRef(loaded.Ref)
	if err != nil {
		t.Fatal(err)
	}
	if pinned.Ref != loaded.Ref {
		t.Fatalf("pinned Workflow = %#v, want %#v", pinned.Ref, loaded.Ref)
	}
	if _, err := LoadRef(Ref{ID: "technical-solution-design", Digest: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"}); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("digest mismatch error = %v", err)
	}
	problemPrompt := loaded.Workflow.Prompts["frame_technical_problem_flow"]
	if len(problemPrompt.Skills) != 1 || problemPrompt.Skills[0].ID != "technical-problem-framing" || problemPrompt.Skills[0].Optional == nil || *problemPrompt.Skills[0].Optional {
		t.Fatalf("problem Prompt Skills = %#v", problemPrompt.Skills)
	}
	if routes := loaded.Workflow.Loops["confirm_technical_solution"]; len(routes) != 3 {
		t.Fatalf("confirm_technical_solution Loop Routes = %#v", routes)
	}
	if _, err := Load("fixture"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("fixture workflow is still loadable: %v", err)
	}
}

func TestProductionWorkflowSourcesAreFlatAndVersionless(t *testing.T) {
	root := filepath.Join("..", "..", "workflows")
	if _, err := os.Stat(filepath.Join(root, "defaults.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("defaults.json must not exist: %v", err)
	}
	paths, err := filepath.Glob(filepath.Join(root, "*", "workflow.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("flat production Workflow sources = %v, want two", paths)
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

func TestWorkflowSelectorRejectsRetiredBusinessVersion(t *testing.T) {
	if _, err := LoadSelector("fanloop@14.0.0"); !errors.Is(err, ErrInvalidSelector) {
		t.Fatalf("versioned selector error = %v, want ErrInvalidSelector", err)
	}
}

func TestBundleDigestUsesNormalizedFiveFileSemantics(t *testing.T) {
	workflowYAML, flowYAML, conditionYAML, loopYAML, promptYAML := validBundle()
	first, err := DecodeBundle(workflowYAML, flowYAML, conditionYAML, loopYAML, promptYAML)
	if err != nil {
		t.Fatal(err)
	}
	formatted := append([]byte("\n# comment\n"), workflowYAML...)
	second, err := DecodeBundle(formatted, flowYAML, conditionYAML, loopYAML, promptYAML)
	if err != nil {
		t.Fatal(err)
	}
	if first.Ref.Digest != second.Ref.Digest {
		t.Fatalf("format-only edit changed digest: %s != %s", first.Ref.Digest, second.Ref.Digest)
	}
	explicitDefault := strings.Replace(string(flowYAML), "      next_step_id: confirm_note", "      next_step_id: confirm_note\n      terminal: false", 1)
	second, err = DecodeBundle(workflowYAML, []byte(explicitDefault), conditionYAML, loopYAML, promptYAML)
	if err != nil {
		t.Fatal(err)
	}
	if first.Ref.Digest != second.Ref.Digest {
		t.Fatalf("explicit default changed digest: %s != %s", first.Ref.Digest, second.Ref.Digest)
	}
	changed := strings.Replace(string(promptYAML), "Write the note", "Write a reviewed note", 1)
	third, err := DecodeBundle(workflowYAML, flowYAML, conditionYAML, loopYAML, []byte(changed))
	if err != nil {
		t.Fatal(err)
	}
	if first.Ref.Digest == third.Ref.Digest {
		t.Fatal("prompt.yaml semantic edit did not change digest")
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
	flowYAML := []byte(`schema_version: 4
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
	conditionYAML := []byte(`schema_version: 2
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
	promptYAML := []byte(`schema_version: 1
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
