package runtime_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestTraceBindIsImmutableForRequirement(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "One Trace"), "flow.init")

	const firstURL = "https://bytedance.larkoffice.com/docx/RequirementTraceA"
	assertSuccess(t, run(binary, "trace", "bind", "--root", root, "--document-url", firstURL), "trace.bind")
	statePath := filepath.Join(root, ".commonloop", "flow", "state.json")
	eventsPath := filepath.Join(root, ".commonloop", "trace", "events.jsonl")
	stateBefore, eventsBefore := readFile(t, statePath), readFile(t, eventsPath)

	unchanged := run(binary, "trace", "bind", "--root", root, "--document-url", firstURL+"?from=duplicate")
	assertSuccess(t, unchanged, "trace.bind")
	if !strings.Contains(unchanged.stdout, `"effect": "unchanged"`) {
		t.Fatalf("same Trace document was not idempotent:\n%s", unchanged.stdout)
	}

	rebound := run(binary, "trace", "bind", "--root", root, "--document-url", "https://bytedance.larkoffice.com/docx/RequirementTraceB")
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

func TestMaintainerTraceBindRequiresStableCLILogDocument(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "commonloop-maintainer", "--title", "Maintainer Trace"), "flow.init")

	const traceURL = "https://bytedance.larkoffice.com/docx/MaintainerTrace"
	const logURL = "https://bytedance.larkoffice.com/docx/MaintainerCLILog"
	missing := run(binary, "trace", "bind", "--root", root, "--document-url", traceURL)
	assertError(t, missing, 2, "INVALID_ARGUMENT")

	bound := run(binary, "trace", "bind", "--root", root, "--document-url", traceURL, "--cli-log-document-url", logURL)
	assertSuccess(t, bound, "trace.bind")
	status := run(binary, "trace", "status", "--root", root)
	assertSuccess(t, status, "trace.status")
	for _, want := range []string{traceURL, `"cli_log_document_url": "` + logURL + `"`} {
		if !strings.Contains(status.stdout, want) {
			t.Fatalf("Trace status is missing %q:\n%s", want, status.stdout)
		}
	}

	unchanged := run(binary, "trace", "bind", "--root", root, "--document-url", traceURL+"?duplicate=1", "--cli-log-document-url", logURL+"?duplicate=1")
	assertSuccess(t, unchanged, "trace.bind")
	if !strings.Contains(unchanged.stdout, `"effect": "unchanged"`) {
		t.Fatalf("same Trace bundle was not idempotent:\n%s", unchanged.stdout)
	}

	ordinaryRoot := t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", ordinaryRoot, "--workflow", "technical-solution-design", "--title", "Ordinary Trace"), "flow.init")
	forbidden := run(binary, "trace", "bind", "--root", ordinaryRoot, "--document-url", traceURL, "--cli-log-document-url", logURL)
	assertError(t, forbidden, 2, "INVALID_ARGUMENT")

	testRoot := t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", testRoot, "--workflow", "commonloop-maintainer", "--title", "Test Trace"), "flow.init")
	forbidden = run(binary, "trace", "bind", "--root", testRoot, "--document-url", traceURL, "--registry", "test", "--cli-log-document-url", logURL)
	assertError(t, forbidden, 2, "INVALID_ARGUMENT")
	assertSuccess(t, run(binary, "trace", "bind", "--root", testRoot, "--document-url", traceURL, "--registry", "test"), "trace.bind")
	testSync := run(binary, "trace", "sync", "--root", testRoot, "--dry-run")
	assertSuccess(t, testSync, "trace.sync")
	if strings.Contains(testSync.stdout, `"target": "cli_log_document"`) {
		t.Fatalf("maintainer test sync unexpectedly contains a CLI log target:\n%s", testSync.stdout)
	}

	sameRoot := t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", sameRoot, "--workflow", "commonloop-maintainer", "--title", "Distinct Trace"), "flow.init")
	same := run(binary, "trace", "bind", "--root", sameRoot, "--document-url", traceURL, "--cli-log-document-url", traceURL+"?same=1")
	assertError(t, same, 1, "PROTECTED_DOCUMENT")
}
