package runtime_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorValidatesConditionRoutingRequirementFiles(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "promotion-design", "--title", "Doctor"), "flow.init")
	healthy := run(binary, "doctor", "--root", root)
	assertSuccess(t, healthy, "doctor")
	for _, id := range []string{"state_schema", "workflow_binding", "outputs", "events", "trace_projection", "card_projection", "card_snapshots"} {
		if !strings.Contains(healthy.stdout, `"id": "`+id+`"`) {
			t.Fatalf("doctor response missing %s:\n%s", id, healthy.stdout)
		}
	}

	statePath := filepath.Join(root, ".fanloop", "output", "state.json")
	content := readFile(t, statePath)
	content = bytes.Replace(content, []byte(`"outputs": {}`), []byte(`"outputs": {"forged": {"type": "string", "value": "x", "producer_step_id": "bootstrap_techdesign"}}`), 1)
	if err := os.WriteFile(statePath, content, 0o600); err != nil {
		t.Fatal(err)
	}
	unhealthy := run(binary, "doctor", "--root", root)
	if unhealthy.exitCode != 1 || unhealthy.stderr != "" || !strings.Contains(unhealthy.stdout, `"status": "unhealthy"`) || !strings.Contains(unhealthy.stdout, `Output \"forged\"`) {
		t.Fatalf("doctor did not detect Output drift: %s", unhealthy.stdout)
	}
}

func TestDoctorChecksCardIndependentlyWhenTraceIsCorrupt(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "promotion-design", "--title", "Doctor domains"), "flow.init")
	if err := os.WriteFile(filepath.Join(root, ".fanloop", "trace", "events.jsonl"), []byte("not-json\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	unhealthy := run(binary, "doctor", "--root", root)
	if unhealthy.exitCode != 1 || unhealthy.stderr != "" || !strings.Contains(unhealthy.stdout, `"status": "unhealthy"`) {
		t.Fatalf("doctor did not report corrupt Trace: %s", unhealthy.stdout)
	}
	for _, want := range []string{
		`"id": "events",` + "\n" + `        "status": "failed"`,
		`"id": "card_projection",` + "\n" + `        "status": "passed"`,
		`"id": "card_snapshots",` + "\n" + `        "status": "passed"`,
	} {
		if !strings.Contains(unhealthy.stdout, want) {
			t.Fatalf("doctor did not keep Card checks independent from Trace (%q):\n%s", want, unhealthy.stdout)
		}
	}
}
