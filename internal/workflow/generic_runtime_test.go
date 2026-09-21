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
			ext := filepath.Ext(name)
			for _, fact := range literals {
				if sourceHardcodesWorkflowFact(string(content), fact, ext) {
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

// sourceHardcodesWorkflowFact reports whether production source embeds a Workflow
// fact as a quoted string literal. Bare identifiers (e.g. reflect.Type) and Go
// import paths (import "reflect") are ignored so short Skill IDs such as
// "reflect" do not false-positive against the Go standard library.
func sourceHardcodesWorkflowFact(content, fact, ext string) bool {
	switch ext {
	case ".go":
		return goSourceHardcodesWorkflowFact(content, fact)
	default:
		return strings.Contains(content, `"`+fact+`"`) || strings.Contains(content, `'`+fact+`'`)
	}
}

func goSourceHardcodesWorkflowFact(content, fact string) bool {
	stripped := stripGoImportDeclarations(content)
	quoted := `"` + fact + `"`
	return strings.Contains(stripped, quoted)
}

func stripGoImportDeclarations(src string) string {
	lines := strings.Split(src, "\n")
	out := make([]string, 0, len(lines))
	inBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if inBlock {
			if trimmed == ")" {
				inBlock = false
			}
			continue
		}
		if trimmed == "import (" {
			inBlock = true
			continue
		}
		if strings.HasPrefix(trimmed, "import ") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
