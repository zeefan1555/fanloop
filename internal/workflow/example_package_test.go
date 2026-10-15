package workflow

import (
	"path/filepath"
	"runtime"
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
