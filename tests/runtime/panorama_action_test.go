package runtime_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type pendingFlowAction struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	TriggerEventID string `json:"trigger_event_id"`
	StepID         string `json:"step_id"`
	Skill          struct {
		ID       string `json:"id"`
		Prompt   string `json:"prompt"`
		Optional bool   `json:"optional"`
		Path     string `json:"path"`
	} `json:"skill"`
}

type flowActionEnvelope struct {
	Data struct {
		EventID string `json:"event_id"`
		State   struct {
			PendingActions []pendingFlowAction `json:"pending_actions"`
		} `json:"state"`
	} `json:"data"`
}

func TestPanoramaActionBlocksBusinessReportsUntilAcknowledged(t *testing.T) {
	t.Setenv("FANLOOP_TEST_KEEP_PENDING", "1")
	binary, root := buildCLI(t), t.TempDir()
	initialized := run(binary, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Panorama action")
	assertSuccess(t, initialized, "flow.init")
	initial := decodeFlowActionEnvelope(t, initialized.stdout)
	action := onlyPendingAction(t, initial)
	if action.Type != "panorama_delivery_required" || action.StepID != "first" || action.TriggerEventID == "" {
		t.Fatalf("unexpected initial action: %#v\n%s", action, initialized.stdout)
	}
	if action.ID != "panorama-"+action.TriggerEventID {
		t.Fatalf("action id = %q, want trigger-derived id for %q", action.ID, action.TriggerEventID)
	}
	if action.Skill.ID != "fanloop-panorama-delivery" || action.Skill.Optional || action.Skill.Prompt == "" || !filepath.IsAbs(action.Skill.Path) {
		t.Fatalf("unexpected system Skill: %#v", action.Skill)
	}
	if _, err := os.Stat(action.Skill.Path); err != nil {
		t.Fatalf("system Skill path is not readable: %v", err)
	}
	skillContent := string(readFile(t, action.Skill.Path))
	for _, want := range []string{"fanloop flow action acknowledge", "local_agent", "panorama_delivery_required"} {
		if !strings.Contains(skillContent, want) {
			t.Fatalf("system Skill does not tell the Agent how to handle %q:\n%s", want, skillContent)
		}
	}

	status := run(binary, "flow", "status", "--root", root)
	assertSuccess(t, status, "flow.status")
	statusAction := onlyPendingAction(t, decodeFlowActionEnvelope(t, status.stdout))
	if statusAction.ID != action.ID || statusAction.TriggerEventID != action.TriggerEventID {
		t.Fatalf("Status changed pending action: init=%#v status=%#v", action, statusAction)
	}

	blocked := run(binary, "flow", "report", "progress", "--root", root,
		"--step-id", "first", "--status", "in_progress", "--summary", "must wait")
	assertError(t, blocked, 1, "REPORT_NOT_ALLOWED")

	rendered := run(binary, "card", "render", "--root", root, "--view", "panorama", "--format", "markdown")
	assertSuccess(t, rendered, "card.render")
	snapshot := decodeCard(t, rendered.stdout).Data.SnapshotPath
	if snapshot == "" {
		t.Fatal("non-dry-run Panorama did not return snapshot_path")
	}

	acknowledged := run(binary, "flow", "action", "acknowledge", "--root", root,
		"--action-id", action.ID, "--panorama-snapshot-path", snapshot)
	assertSuccess(t, acknowledged, "flow.action.acknowledge")
	ack := decodeFlowActionEnvelope(t, acknowledged.stdout)
	if ack.Data.EventID == "" || len(ack.Data.State.PendingActions) != 0 {
		t.Fatalf("acknowledgment did not clear the action: %s", acknowledged.stdout)
	}

	events := string(readFile(t, filepath.Join(root, ".fanloop", "trace", "events.jsonl")))
	for _, want := range []string{`"kind":"flow_action_acknowledged"`, `"action_id":"` + action.ID + `"`, `"panorama_snapshot_path":"` + snapshot + `"`} {
		if !strings.Contains(events, want) {
			t.Fatalf("Event audit missing %s:\n%s", want, events)
		}
	}

	assertSuccess(t, run(binary, "flow", "report", "progress", "--root", root,
		"--step-id", "first", "--status", "in_progress", "--summary", "working"), "flow.report.progress")
}

func TestPanoramaActionAcknowledgmentRejectsStaleAbsentAndInvalidReceipts(t *testing.T) {
	t.Setenv("FANLOOP_TEST_KEEP_PENDING", "1")
	binary, root := buildCLI(t), t.TempDir()
	initialized := run(binary, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Action receipt validation")
	assertSuccess(t, initialized, "flow.init")
	action := onlyPendingAction(t, decodeFlowActionEnvelope(t, initialized.stdout))

	rendered := run(binary, "card", "render", "--root", root, "--view", "panorama", "--format", "markdown")
	assertSuccess(t, rendered, "card.render")
	snapshot := decodeCard(t, rendered.stdout).Data.SnapshotPath

	dryRun := run(binary, "flow", "action", "acknowledge", "--root", root, "--dry-run",
		"--action-id", action.ID, "--panorama-snapshot-path", snapshot)
	assertSuccess(t, dryRun, "flow.action.acknowledge")
	dryResponse := decodeFlowActionEnvelope(t, dryRun.stdout)
	if dryResponse.Data.EventID != "" || len(dryResponse.Data.State.PendingActions) != 0 {
		t.Fatalf("dry-run response did not project a cleared action without an Event: %s", dryRun.stdout)
	}
	assertPendingActionID(t, binary, root, action.ID)

	stale := run(binary, "flow", "action", "acknowledge", "--root", root,
		"--action-id", "panorama-stale", "--panorama-snapshot-path", ".fanloop/card/missing.json")
	assertError(t, stale, 1, "STATE_CONFLICT")
	assertPendingActionID(t, binary, root, action.ID)

	invalid := filepath.Join(root, ".fanloop", "card", "invalid.json")
	if err := os.WriteFile(invalid, []byte("not-json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	invalidReceipt := run(binary, "flow", "action", "acknowledge", "--root", root,
		"--action-id", action.ID, "--panorama-snapshot-path", ".fanloop/card/invalid.json")
	assertError(t, invalidReceipt, 2, "INVALID_ARGUMENT")
	assertPendingActionID(t, binary, root, action.ID)
	validButNotPanorama := filepath.Join(root, ".fanloop", "card", "empty-object.json")
	if err := os.WriteFile(validButNotPanorama, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	nonPanoramaReceipt := run(binary, "flow", "action", "acknowledge", "--root", root,
		"--action-id", action.ID, "--panorama-snapshot-path", ".fanloop/card/empty-object.json")
	assertError(t, nonPanoramaReceipt, 2, "INVALID_ARGUMENT")
	assertPendingActionID(t, binary, root, action.ID)

	symlink := filepath.Join(root, ".fanloop", "card", "link.json")
	if err := os.Symlink(filepath.Join(root, filepath.FromSlash(snapshot)), symlink); err != nil {
		t.Fatal(err)
	}
	symlinkReceipt := run(binary, "flow", "action", "acknowledge", "--root", root,
		"--action-id", action.ID, "--panorama-snapshot-path", ".fanloop/card/link.json")
	assertError(t, symlinkReceipt, 2, "INVALID_ARGUMENT")
	assertPendingActionID(t, binary, root, action.ID)

	acknowledged := run(binary, "flow", "action", "acknowledge", "--root", root,
		"--action-id", action.ID, "--panorama-snapshot-path", snapshot)
	assertSuccess(t, acknowledged, "flow.action.acknowledge")
	assertNoPendingAction(t, acknowledged.stdout)

	absent := run(binary, "flow", "action", "acknowledge", "--root", root,
		"--action-id", action.ID, "--panorama-snapshot-path", ".fanloop/card/missing.json")
	assertError(t, absent, 1, "REPORT_NOT_ALLOWED")
	status := run(binary, "flow", "status", "--root", root)
	assertSuccess(t, status, "flow.status")
	assertNoPendingAction(t, status.stdout)
}

func TestPanoramaActionsFollowEntryAndCompletionEffects(t *testing.T) {
	t.Setenv("FANLOOP_TEST_KEEP_PENDING", "1")
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Action matrix"), "flow.init")
	initial := acknowledgePendingPanorama(t, binary, root)

	progress := run(binary, "flow", "report", "progress", "--root", root,
		"--step-id", "first", "--status", "in_progress", "--summary", "working")
	assertSuccess(t, progress, "flow.report.progress")
	assertNoPendingAction(t, progress.stdout)

	sameStepLoop := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "first", "--condition-result", conditionResult("first_rework", "enum_value", `"rework"`),
		"--back-step-id", "first", "--summary", "revise first in place")
	assertSuccess(t, sameStepLoop, "flow.report.result")
	sameStepLoopAction := onlyPendingAction(t, decodeFlowActionEnvelope(t, sameStepLoop.stdout))
	if sameStepLoopAction.StepID != "first" || sameStepLoopAction.TriggerEventID == initial.TriggerEventID {
		t.Fatalf("same-Step loop action = %#v, initial = %#v", sameStepLoopAction, initial)
	}
	acknowledgePendingPanorama(t, binary, root)

	dryRun := run(binary, "flow", "report", "result", "--root", root, "--dry-run",
		"--step-id", "first", "--condition-result", conditionResult("first_completed", "path", `"first.md"`),
		"--next-step-id", "second", "--summary", "plan first")
	assertSuccess(t, dryRun, "flow.report.result")
	assertNoPendingAction(t, dryRun.stdout)
	status := run(binary, "flow", "status", "--root", root)
	assertSuccess(t, status, "flow.status")
	assertNoPendingAction(t, status.stdout)

	rejected := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "first", "--condition-result", conditionResult("first_completed", "path", `"first.md"`),
		"--terminal", "--summary", "invalid target")
	assertError(t, rejected, 2, "ROUTE_NOT_ALLOWED")
	status = run(binary, "flow", "status", "--root", root)
	assertSuccess(t, status, "flow.status")
	assertNoPendingAction(t, status.stdout)

	advanced := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "first", "--condition-result", conditionResult("first_completed", "path", `"first.md"`),
		"--next-step-id", "second", "--summary", "first complete")
	assertSuccess(t, advanced, "flow.report.result")
	advancedAction := onlyPendingAction(t, decodeFlowActionEnvelope(t, advanced.stdout))
	if advancedAction.StepID != "second" || advancedAction.TriggerEventID == initial.TriggerEventID {
		t.Fatalf("advanced action = %#v, initial = %#v", advancedAction, initial)
	}
	acknowledgePendingPanorama(t, binary, root)

	looped := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "second", "--condition-result", conditionResult("second_rework", "enum_value", `"rework"`),
		"--back-step-id", "first", "--summary", "return to first")
	assertSuccess(t, looped, "flow.report.result")
	loopedAction := onlyPendingAction(t, decodeFlowActionEnvelope(t, looped.stdout))
	if loopedAction.StepID != "first" || loopedAction.ID == advancedAction.ID {
		t.Fatalf("looped action = %#v, advanced = %#v", loopedAction, advancedAction)
	}
	acknowledgePendingPanorama(t, binary, root)

	assertSuccess(t, run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "first", "--condition-result", conditionResult("first_completed", "path", `"first-again.md"`),
		"--next-step-id", "second", "--summary", "first complete again"), "flow.report.result")
	acknowledgePendingPanorama(t, binary, root)
	assertSuccess(t, run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "second",
		"--condition-result", conditionResult("second_completed", "path", `"second.md"`),
		"--condition-result", conditionResult("second_evidence", "path", `"evidence.md"`),
		"--next-step-id", "third", "--summary", "second complete"), "flow.report.result")
	acknowledgePendingPanorama(t, binary, root)

	completed := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "third", "--condition-result", conditionResult("third_completed", "path", `"third.md"`),
		"--terminal", "--summary", "workflow complete")
	assertSuccess(t, completed, "flow.report.result")
	completedAction := onlyPendingAction(t, decodeFlowActionEnvelope(t, completed.stdout))
	if completedAction.StepID != "" || completedAction.TriggerEventID == "" {
		t.Fatalf("completed action = %#v", completedAction)
	}
	status = run(binary, "flow", "status", "--root", root)
	assertSuccess(t, status, "flow.status")
	if got := onlyPendingAction(t, decodeFlowActionEnvelope(t, status.stdout)); got.ID != completedAction.ID {
		t.Fatalf("completed Status action = %#v, want %#v", got, completedAction)
	}
	acknowledgePendingPanorama(t, binary, root)
	status = run(binary, "flow", "status", "--root", root)
	assertSuccess(t, status, "flow.status")
	assertNoPendingAction(t, status.stdout)
	if strings.Contains(status.stdout, `"current"`) || !strings.Contains(status.stdout, `"status": "completed"`) {
		t.Fatalf("completed Status changed after acknowledgment:\n%s", status.stdout)
	}
}

func TestPanoramaActionsHandleStartAndJump(t *testing.T) {
	t.Setenv("FANLOOP_TEST_KEEP_PENDING", "1")
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "runtime-controls", "--title", "Action controls"), "flow.init")
	acknowledgePendingPanorama(t, binary, root)

	started := run(binary, "flow", "report", "result", "--root", root, "--step-id", "first",
		"--condition-result", conditionResult("start_confirmed", "enum_value", `"confirmed"`),
		"--evidence", `{"source":"human","content":"confirmed"}`,
		"--summary", "step confirmed", "--start-current-step")
	assertSuccess(t, started, "flow.report.result")
	assertNoPendingAction(t, started.stdout)

	sameStepJump := run(binary, "flow", "report", "result", "--root", root, "--step-id", "first",
		"--condition-result", conditionResult("jump_requested", "string", `"first"`),
		"--evidence", `{"source":"human","content":"re-enter first"}`,
		"--summary", "re-enter first", "--jump-step-id", "first")
	assertSuccess(t, sameStepJump, "flow.report.result")
	sameStepJumpAction := onlyPendingAction(t, decodeFlowActionEnvelope(t, sameStepJump.stdout))
	if sameStepJumpAction.StepID != "first" || sameStepJumpAction.TriggerEventID == "" {
		t.Fatalf("same-Step jump action = %#v", sameStepJumpAction)
	}
	acknowledgePendingPanorama(t, binary, root)

	jumped := run(binary, "flow", "report", "result", "--root", root, "--step-id", "first",
		"--condition-result", conditionResult("jump_requested", "string", `"third"`),
		"--evidence", `{"source":"human","content":"jump to third"}`,
		"--summary", "jump requested", "--jump-step-id", "third")
	assertSuccess(t, jumped, "flow.report.result")
	if action := onlyPendingAction(t, decodeFlowActionEnvelope(t, jumped.stdout)); action.StepID != "third" {
		t.Fatalf("jumped action = %#v", action)
	}
}

func TestPanoramaAcknowledgmentRejectsOldStateSnapshotAndAcceptsCurrentLarkCard(t *testing.T) {
	t.Setenv("FANLOOP_TEST_KEEP_PENDING", "1")
	binary, root := buildCLI(t), t.TempDir()
	initialized := run(binary, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Current snapshot")
	assertSuccess(t, initialized, "flow.init")
	initialAction := onlyPendingAction(t, decodeFlowActionEnvelope(t, initialized.stdout))
	initialRender := run(binary, "card", "render", "--root", root, "--view", "panorama", "--format", "markdown")
	assertSuccess(t, initialRender, "card.render")
	initialSnapshot := decodeCard(t, initialRender.stdout).Data.SnapshotPath
	assertSuccess(t, run(binary, "flow", "action", "acknowledge", "--root", root,
		"--action-id", initialAction.ID, "--panorama-snapshot-path", initialSnapshot), "flow.action.acknowledge")

	advanced := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "first", "--condition-result", conditionResult("first_completed", "path", `"first.md"`),
		"--next-step-id", "second", "--summary", "first complete")
	assertSuccess(t, advanced, "flow.report.result")
	advancedAction := onlyPendingAction(t, decodeFlowActionEnvelope(t, advanced.stdout))

	oldReceipt := run(binary, "flow", "action", "acknowledge", "--root", root,
		"--action-id", advancedAction.ID, "--panorama-snapshot-path", initialSnapshot)
	assertError(t, oldReceipt, 2, "INVALID_ARGUMENT")
	assertPendingActionID(t, binary, root, advancedAction.ID)

	currentRender := run(binary, "card", "render", "--root", root, "--view", "panorama", "--format", "lark-json")
	assertSuccess(t, currentRender, "card.render")
	currentSnapshot := decodeCard(t, currentRender.stdout).Data.SnapshotPath
	acknowledged := run(binary, "flow", "action", "acknowledge", "--root", root,
		"--action-id", advancedAction.ID, "--panorama-snapshot-path", currentSnapshot)
	assertSuccess(t, acknowledged, "flow.action.acknowledge")
	assertNoPendingAction(t, acknowledged.stdout)
}

func TestStaleProjectionWithPendingPanoramaIsRecoverable(t *testing.T) {
	t.Setenv("FANLOOP_TEST_KEEP_PENDING", "1")
	binary, root := buildCLI(t), t.TempDir()
	initialized := run(binary, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Stale projection")
	assertSuccess(t, initialized, "flow.init")
	projectionPath := filepath.Join(root, ".fanloop", "card", "projection.json")
	initialProjection := readFile(t, projectionPath)
	acknowledgePendingPanorama(t, binary, root)

	advanced := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "first", "--condition-result", conditionResult("first_completed", "path", `"first.md"`),
		"--next-step-id", "second", "--summary", "first complete")
	assertSuccess(t, advanced, "flow.report.result")
	action := onlyPendingAction(t, decodeFlowActionEnvelope(t, advanced.stdout))
	if err := os.WriteFile(projectionPath, initialProjection, 0o600); err != nil {
		t.Fatal(err)
	}

	diagnosed := run(binary, "doctor", "--root", root)
	if diagnosed.exitCode != 1 || diagnosed.stderr != "" {
		t.Fatalf("doctor exit = %d stderr = %s\n%s", diagnosed.exitCode, diagnosed.stderr, diagnosed.stdout)
	}
	stale := doctorCheck(t, diagnosed.stdout, "card_projection")
	if stale.Status != "failed" || stale.Summary != "Card projection is stale." || !strings.Contains(stale.Hint, "non-dry-run fanloop card render") || strings.Contains(stale.Hint, "accepted Flow report") {
		t.Fatalf("stale projection hint = %#v", stale)
	}
	blocked := run(binary, "flow", "report", "progress", "--root", root,
		"--step-id", "second", "--status", "in_progress", "--summary", "must wait")
	assertError(t, blocked, 1, "REPORT_NOT_ALLOWED")

	dryRun := run(binary, "card", "render", "--root", root, "--dry-run", "--view", "panorama", "--format", "markdown")
	assertSuccess(t, dryRun, "card.render")
	if dryRunCard := decodeCard(t, dryRun.stdout); dryRunCard.Data.SnapshotPath != "" || !strings.Contains(string(dryRunCard.Data.Content), "Second") {
		t.Fatalf("dry-run did not render the committed Step without a snapshot: %#v", dryRunCard.Data)
	}
	if got := readFile(t, projectionPath); !bytes.Equal(got, initialProjection) {
		t.Fatal("dry-run changed the stale Card projection")
	}

	rendered := run(binary, "card", "render", "--root", root, "--view", "panorama", "--format", "markdown")
	assertSuccess(t, rendered, "card.render")
	rebuilt := readFile(t, projectionPath)
	for _, want := range []string{`"current_step_id": "second"`, `"first_artifact"`} {
		if !bytes.Contains(rebuilt, []byte(want)) {
			t.Fatalf("rebuilt projection missing %s:\n%s", want, rebuilt)
		}
	}
	snapshot := decodeCard(t, rendered.stdout).Data.SnapshotPath
	acknowledged := run(binary, "flow", "action", "acknowledge", "--root", root,
		"--action-id", action.ID, "--panorama-snapshot-path", snapshot)
	assertSuccess(t, acknowledged, "flow.action.acknowledge")
	assertNoPendingAction(t, acknowledged.stdout)

	healthy := run(binary, "doctor", "--root", root)
	assertSuccess(t, healthy, "doctor")
	if check := doctorCheck(t, healthy.stdout, "card_projection"); check.Status != "passed" {
		t.Fatalf("card projection after rebuild = %#v", check)
	}
}

func TestSameStepReentryRejectsByteIdenticalSnapshotWithPriorActionIdentity(t *testing.T) {
	t.Setenv("FANLOOP_TEST_KEEP_PENDING", "1")
	binary, root := buildCLI(t), t.TempDir()
	initialized := run(binary, "flow", "init", "--root", root, "--workflow", "linear-runtime", "--title", "Same-step receipt identity")
	assertSuccess(t, initialized, "flow.init")
	initialAction := onlyPendingAction(t, decodeFlowActionEnvelope(t, initialized.stdout))
	initialRender := run(binary, "card", "render", "--root", root, "--view", "panorama", "--format", "markdown")
	assertSuccess(t, initialRender, "card.render")
	initialSnapshot := decodeCard(t, initialRender.stdout).Data.SnapshotPath
	assertSuccess(t, run(binary, "flow", "action", "acknowledge", "--root", root,
		"--action-id", initialAction.ID, "--panorama-snapshot-path", initialSnapshot), "flow.action.acknowledge")

	looped := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "first", "--condition-result", conditionResult("first_rework", "enum_value", `"rework"`),
		"--back-step-id", "first", "--summary", "re-enter first")
	assertSuccess(t, looped, "flow.report.result")
	loopAction := onlyPendingAction(t, decodeFlowActionEnvelope(t, looped.stdout))
	currentRender := run(binary, "card", "render", "--root", root, "--view", "panorama", "--format", "markdown")
	assertSuccess(t, currentRender, "card.render")
	currentSnapshot := decodeCard(t, currentRender.stdout).Data.SnapshotPath
	currentBytes := readFile(t, filepath.Join(root, filepath.FromSlash(currentSnapshot)))
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(initialSnapshot)), currentBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	wrongIdentity := run(binary, "flow", "action", "acknowledge", "--root", root,
		"--action-id", loopAction.ID, "--panorama-snapshot-path", initialSnapshot)
	assertError(t, wrongIdentity, 2, "INVALID_ARGUMENT")
	assertPendingActionID(t, binary, root, loopAction.ID)
	assertSuccess(t, run(binary, "flow", "action", "acknowledge", "--root", root,
		"--action-id", loopAction.ID, "--panorama-snapshot-path", currentSnapshot), "flow.action.acknowledge")
}

type doctorCheckResult struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Summary string `json:"summary"`
	Hint    string `json:"hint"`
}

func doctorCheck(t *testing.T, stdout, id string) doctorCheckResult {
	t.Helper()
	var envelope struct {
		Data struct {
			Checks []doctorCheckResult `json:"checks"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatal(err)
	}
	for _, check := range envelope.Data.Checks {
		if check.ID == id {
			return check
		}
	}
	t.Fatalf("doctor response missing %s:\n%s", id, stdout)
	return doctorCheckResult{}
}

func decodeFlowActionEnvelope(t *testing.T, content string) flowActionEnvelope {
	t.Helper()
	var envelope flowActionEnvelope
	if err := json.Unmarshal([]byte(content), &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope
}

func onlyPendingAction(t *testing.T, envelope flowActionEnvelope) pendingFlowAction {
	t.Helper()
	if len(envelope.Data.State.PendingActions) != 1 {
		t.Fatalf("pending actions = %d, want 1", len(envelope.Data.State.PendingActions))
	}
	return envelope.Data.State.PendingActions[0]
}

func assertNoPendingAction(t *testing.T, content string) {
	t.Helper()
	if actions := decodeFlowActionEnvelope(t, content).Data.State.PendingActions; len(actions) != 0 {
		t.Fatalf("pending actions = %#v, want none\n%s", actions, content)
	}
}

func assertPendingActionID(t *testing.T, binary, root, want string) {
	t.Helper()
	status := run(binary, "flow", "status", "--root", root)
	assertSuccess(t, status, "flow.status")
	if got := onlyPendingAction(t, decodeFlowActionEnvelope(t, status.stdout)).ID; got != want {
		t.Fatalf("pending action id = %q, want %q", got, want)
	}
}

func acknowledgePendingPanorama(t *testing.T, binary, root string) pendingFlowAction {
	t.Helper()
	status := run(binary, "flow", "status", "--root", root)
	assertSuccess(t, status, "flow.status")
	action := onlyPendingAction(t, decodeFlowActionEnvelope(t, status.stdout))
	rendered := run(binary, "card", "render", "--root", root, "--view", "panorama", "--format", "markdown")
	assertSuccess(t, rendered, "card.render")
	snapshot := decodeCard(t, rendered.stdout).Data.SnapshotPath
	acknowledged := run(binary, "flow", "action", "acknowledge", "--root", root,
		"--action-id", action.ID, "--panorama-snapshot-path", snapshot)
	assertSuccess(t, acknowledged, "flow.action.acknowledge")
	assertNoPendingAction(t, acknowledged.stdout)
	return action
}

func TestReservedPanoramaConditionIsRejected(t *testing.T) {
	t.Setenv("FANLOOP_TEST_KEEP_PENDING", "1")
	binary := buildCLI(t)
	conditionPath := filepath.Join(filepath.Dir(binary), ".fanloop-user", "source", "fanloop-workflow", "linear-runtime", "condition.yaml")
	content := string(readFile(t, conditionPath))
	updated := strings.Replace(content, "conditions:\n", "conditions:\n  panorama_presented:\n    output: {key: panorama_snapshot_path, type: path}\n", 1)
	if updated == content {
		t.Fatal("synthetic condition file has no conditions block")
	}
	if err := os.WriteFile(conditionPath, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
	got := run(binary, "flow", "init", "--root", t.TempDir(), "--workflow", "linear-runtime", "--title", "Reserved")
	assertError(t, got, 5, "CONDITION_RESERVED")
	var envelope struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(got.stderr), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Error.Message != `Condition "panorama_presented" is reserved for the Loop Runtime Panorama action` {
		t.Fatalf("reserved diagnostic = %s", envelope.Error.Message)
	}
}
