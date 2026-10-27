package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPinnedRequirementDoesNotPrepareUnrelatedCurrent(t *testing.T) {
	dataRoot, _, _ := configRebindSnapshots(t)
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FANLOOP_NO_AUTO_CONFIG", "1")
	configUnsetSource(t)
	root := t.TempDir()
	configRebindExecute(t, []string{"flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Pin authority"}, 0)
	facts := map[string][]byte{}
	for _, file := range []string{"flow/state.json", "output/state.json", "trace/events.jsonl", "configuration-snapshot"} {
		content, err := os.ReadFile(filepath.Join(root, ".fanloop", file))
		if err != nil {
			t.Fatal(err)
		}
		facts[file] = content
	}
	if err := os.Remove(filepath.Join(dataRoot, "config", "current")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dataRoot, "missing"), filepath.Join(dataRoot, "config", "current")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FANLOOP_CONFIG_SOURCE", filepath.Join(dataRoot, "invalid-author"))
	result := configRebindExecute(t, []string{"flow", "status", "--root", root}, 0)
	if !strings.Contains(result, strings.TrimSpace(string(facts["configuration-snapshot"]))) {
		t.Fatal("status did not use the actual configuration pin")
	}
	for file, before := range facts {
		after, err := os.ReadFile(filepath.Join(root, ".fanloop", file))
		if err != nil || !bytes.Equal(before, after) {
			t.Fatalf("status changed %s: %v", file, err)
		}
	}
}
