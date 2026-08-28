package workflow

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestApprovedTechnicalSolutionDesignBundleLoads(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate repository")
	}
	review := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "workflows", "technical-solution-design"))

	loaded, err := LoadDirectory(review)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Ref.ID != "technical-solution-design" || loaded.Ref.Digest == "" {
		t.Fatalf("loaded workflow = %#v", loaded.Ref)
	}
	production, err := LoadRef(loaded.Ref)
	if err != nil {
		t.Fatal(err)
	}
	if production.Ref.Digest != loaded.Ref.Digest {
		t.Fatalf("production digest %s differs from approved review %s", production.Ref.Digest, loaded.Ref.Digest)
	}
}
