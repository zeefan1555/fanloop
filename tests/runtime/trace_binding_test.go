package runtime_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestTraceBindIsImmutableForRequirement(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "One Trace"), "flow.init")

	const firstURL = "https://example.invalid/redacted"
	assertSuccess(t, run(binary, "trace", "bind", "--root", root, "--document-url", firstURL), "trace.bind")
	statePath := filepath.Join(root, ".fanloop", "flow", "state.json")
	eventsPath := filepath.Join(root, ".fanloop", "trace", "events.jsonl")
	stateBefore, eventsBefore := readFile(t, statePath), readFile(t, eventsPath)

	unchanged := run(binary, "trace", "bind", "--root", root, "--document-url", firstURL+"?from=duplicate")
	assertSuccess(t, unchanged, "trace.bind")
	if !strings.Contains(unchanged.stdout, `"effect": "unchanged"`) {
		t.Fatalf("same Trace document was not idempotent:\n%s", unchanged.stdout)
	}

	rebound := run(binary, "trace", "bind", "--root", root, "--document-url", "https://example.invalid/redacted")
	assertError(t, rebound, 2, "INVALID_ARGUMENT")
	registryChange := run(binary, "trace", "bind", "--root", root, "--document-url", firstURL, "--registry", "test")
	assertError(t, registryChange, 2, "INVALID_ARGUMENT")
	if stateAfter := readFile(t, statePath); !bytes.Equal(stateAfter, stateBefore) {
		t.Fatalf("rejected Trace rebind changed Flow State:\n%s", stateAfter)
	}
	if eventsAfter := readFile(t, eventsPath); !bytes.Equal(eventsAfter, eventsBefore) {
		t.Fatalf("rejected Trace rebind appended an Event:\n%s", eventsAfter)
	}

	status := run(binary, "trace", "status", "--root", root)
	assertSuccess(t, status, "trace.status")
	if !strings.Contains(status.stdout, firstURL) || strings.Contains(status.stdout, "RequirementTraceB") {
		t.Fatalf("rejected rebind changed the authoritative Trace:\n%s", status.stdout)
	}
}
