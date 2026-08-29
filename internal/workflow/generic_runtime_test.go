package workflow

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestProductionRuntimeDoesNotHardcodeWorkflowFacts(t *testing.T) {
	loaded, err := List()
	if err != nil {
		t.Fatal(err)
	}
	facts := map[string]bool{}
	for _, item := range loaded {
		facts[item.Workflow.ID] = true
		for _, stepID := range item.Workflow.OrderedStepIDs() {
			facts[stepID] = true
		}
		for conditionID, condition := range item.Workflow.Conditions {
			facts[conditionID] = true
			facts[condition.Output.Key] = true
		}
		for _, prompt := range item.Workflow.Prompts {
			for _, skill := range prompt.Skills {
				facts[skill.ID] = true
			}
		}
	}
	literals := make([]string, 0, len(facts))
	for fact := range facts {
		literals = append(literals, fact)
	}
	sort.Strings(literals)

	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate repository root")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
	for _, directory := range []string{"cmd", "internal", "scripts"} {
		err := filepath.WalkDir(filepath.Join(root, directory), func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() {
				return walkErr
			}
			name := entry.Name()
			if strings.HasSuffix(name, "_test.go") || strings.Contains(name, ".test.") {
				return nil
			}
			switch filepath.Ext(name) {
			case ".go", ".js", ".sh":
			default:
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, fact := range literals {
				if strings.Contains(string(content), fact) {
					relative, _ := filepath.Rel(root, path)
					t.Errorf("production source %s hardcodes Workflow fact %q", relative, fact)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
