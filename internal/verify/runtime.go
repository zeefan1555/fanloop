package verify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	clierrs "github.com/zeefan1555/fanloop/errs"
	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/idl/verifyidl"
	"github.com/zeefan1555/fanloop/internal/release"
	"github.com/zeefan1555/fanloop/internal/skillconfig"
	"github.com/zeefan1555/fanloop/internal/state"
)

var runIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

type Runtime struct {
	Executable string
	DataRoot   string
	ConfigRoot string
	Now        func() time.Time
	NewRunID   func() string
	Run        func(context.Context, string, []string, []string) (commandResult, error)
}

type commandResult struct {
	ID        string
	Argv      []string
	StartedAt time.Time
	Duration  time.Duration
	ExitCode  int
	Stdout    []byte
	Stderr    []byte
}

type cliEnvelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

var _ verifyidl.VerifyService = Runtime{}

func DefaultRuntime() Runtime {
	executable, _ := os.Executable()
	dataRoot, _ := release.DefaultDataRoot()
	configRoot, _ := skillconfig.DefaultRoot()
	return Runtime{
		Executable: executable,
		DataRoot:   dataRoot,
		ConfigRoot: configRoot,
		Now:        time.Now,
		NewRunID:   state.NewEventID,
		Run:        executeCommand,
	}
}

func (runtime Runtime) Doctor(ctx context.Context, request *verifyidl.VerifyDoctorRequest) (*verifyidl.VerifyDoctorResponse, error) {
	if request == nil {
		return nil, publicError(erroridl.ErrorCode_INVALID_ARGUMENT, "request is required")
	}
	checks := []*verifyidl.VerificationCheck{}
	if runtime.Executable == "" || !filepath.IsAbs(runtime.Executable) {
		checks = append(checks, failedCheck("executable", "Running executable path is unavailable.", "Run an absolute Fanloop candidate binary."))
	} else if digest, err := fileSHA256(runtime.Executable); err != nil {
		checks = append(checks, failedCheck("executable", "Cannot hash the running executable: "+err.Error(), "Repair or rebuild the candidate."))
	} else {
		checks = append(checks, passedCheck("executable", "Running executable is readable: "+digest))
	}
	if runtime.DataRoot == "" || !filepath.IsAbs(runtime.DataRoot) {
		checks = append(checks, failedCheck("data_root", "Verification data root is not absolute.", "Set FANLOOP_DATA_HOME to an absolute path."))
	} else {
		checks = append(checks, passedCheck("data_root", "Verification data root is absolute."))
	}
	if os.Getenv("BOTMUX_CHAT_ID") != "" || os.Getenv("BOTMUX_SESSION_ID") != "" {
		checks = append(checks, failedCheck("remote_binding", "Botmux binding is present in the verification environment.", "Unset BOTMUX_CHAT_ID and BOTMUX_SESSION_ID."))
	} else {
		checks = append(checks, passedCheck("remote_binding", "No Botmux binding is present."))
	}
	for _, probe := range []struct {
		id   string
		args []string
	}{
		{id: "version", args: []string{"version"}},
		{id: "doctor", args: []string{"doctor"}},
	} {
		result, err := runtime.run(ctx, os.Environ(), probe.id, probe.args...)
		if err != nil {
			checks = append(checks, failedCheck(probe.id, err.Error(), "Repair or rebuild the candidate."))
			continue
		}
		if result.ExitCode != 0 {
			checks = append(checks, failedCheck(probe.id, string(result.Stderr), "Follow the public command error and retry."))
			continue
		}
		envelope, err := decodeEnvelope(result.Stdout)
		if err != nil || !envelope.OK {
			summary := "Public command returned ok=false."
			if err != nil {
				summary = "Public command returned invalid JSON: " + err.Error()
			}
			checks = append(checks, failedCheck(probe.id, summary, "Repair the candidate output contract."))
			continue
		}
		checks = append(checks, passedCheck(probe.id, "Public command completed successfully."))
	}
	outcome := verifyidl.VerificationOutcome_passed
	for _, check := range checks {
		if check.Status == verifyidl.VerificationCheckStatus_failed {
			outcome = verifyidl.VerificationOutcome_blocked
			break
		}
	}
	return &verifyidl.VerifyDoctorResponse{Outcome: outcome, Checks: checks}, nil
}

func (runtime Runtime) Smoke(ctx context.Context, request *verifyidl.VerifySmokeRequest) (*verifyidl.VerifySmokeResponse, error) {
	if request == nil {
		return nil, publicError(erroridl.ErrorCode_INVALID_ARGUMENT, "request is required")
	}
	if runtime.Executable == "" || runtime.DataRoot == "" || runtime.ConfigRoot == "" || runtime.Now == nil || runtime.NewRunID == nil {
		return nil, publicError(erroridl.ErrorCode_INTERNAL, "verification runtime is not configured")
	}
	runID := runtime.NewRunID()
	if !runIDPattern.MatchString(runID) {
		return nil, publicError(erroridl.ErrorCode_INTERNAL, "verification run id is invalid")
	}
	workRoot := filepath.Join(runtime.DataRoot, "verification", "work", runID)
	requirementRoot := filepath.Join(workRoot, "requirement")
	evidenceDir := ""
	if request.EvidenceDir != nil {
		evidenceDir = strings.TrimSpace(*request.EvidenceDir)
		if evidenceDir == "" {
			return nil, publicError(erroridl.ErrorCode_INVALID_ARGUMENT, "evidence_dir must not be empty")
		}
	}
	evidence, err := newEvidenceRun(runtime.DataRoot, evidenceDir, runID, requirementRoot, workRoot, runtime.Now())
	if err != nil {
		return nil, publicError(erroridl.ErrorCode_LOCAL_COMMIT_FAILED, err.Error())
	}
	response := &verifyidl.VerifySmokeResponse{RunId: runID, EvidenceDir: evidence.dir, Outcome: verifyidl.VerificationOutcome_blocked}
	finish := func() (*verifyidl.VerifySmokeResponse, error) {
		evidence.manifest.FinishedAt = runtime.Now().UTC().Format(time.RFC3339Nano)
		evidence.manifest.Outcome = response.Outcome.String()
		evidence.manifest.Checks = response.Checks
		if err := evidence.writeManifest(); err != nil {
			return nil, publicError(erroridl.ErrorCode_LOCAL_COMMIT_FAILED, err.Error())
		}
		if err := evidence.writeReport(); err != nil {
			return nil, publicError(erroridl.ErrorCode_LOCAL_COMMIT_FAILED, err.Error())
		}
		return response, nil
	}
	if err := os.MkdirAll(requirementRoot, 0o700); err != nil {
		return nil, publicError(erroridl.ErrorCode_LOCAL_COMMIT_FAILED, err.Error())
	}
	env, err := runtime.isolatedEnvironment(workRoot)
	if err != nil {
		return nil, publicError(erroridl.ErrorCode_LOCAL_COMMIT_FAILED, err.Error())
	}
	run := func(id string, args ...string) (commandResult, string, error) {
		result, runErr := runtime.run(ctx, env, id, args...)
		if runErr != nil {
			return result, "", runErr
		}
		ref, recordErr := evidence.recordCommand(result)
		return result, ref, recordErr
	}
	fail := func(id, summary, hint string, blocked bool) (*verifyidl.VerifySmokeResponse, error) {
		response.Checks = append(response.Checks, failedCheck(id, summary, hint))
		response.Outcome = verifyidl.VerificationOutcome_failed
		if blocked {
			response.Outcome = verifyidl.VerificationOutcome_blocked
		}
		runtime.finishSmokeCleanup(ctx, env, evidence, response, runID, workRoot)
		return finish()
	}

	versionResult, versionRef, err := run("version", "version")
	if err != nil {
		return fail("version", err.Error(), "Repair or rebuild the candidate.", true)
	}
	if versionResult.ExitCode != 0 {
		return fail("version", commandSummary(versionResult), "Repair or rebuild the candidate.", true)
	}
	var versionData struct {
		ReleaseVersion string `json:"release_version"`
		CommitSHA      string `json:"commit_sha"`
		Workflows      []struct {
			ID string `json:"id"`
		} `json:"workflows"`
	}
	if err := decodeData(versionResult.Stdout, &versionData); err != nil {
		return fail("version", err.Error(), "Repair the version output contract.", true)
	}
	if strings.TrimSpace(versionData.ReleaseVersion) == "" || strings.TrimSpace(versionData.CommitSHA) == "" || len(versionData.Workflows) == 0 || strings.TrimSpace(versionData.Workflows[0].ID) == "" {
		return fail("version", "Version output omitted candidate identity.", "Repair the version output contract.", true)
	}
	workflowID := versionData.Workflows[0].ID
	digest, err := fileSHA256(runtime.Executable)
	if err != nil {
		return fail("binary_digest", err.Error(), "Repair or rebuild the candidate.", true)
	}
	response.ReleaseVersion, response.CommitSha, response.BinarySha256 = versionData.ReleaseVersion, versionData.CommitSHA, digest
	evidence.manifest.Candidate = candidateIdentity{ReleaseVersion: versionData.ReleaseVersion, CommitSHA: versionData.CommitSHA, BinarySHA256: digest}
	response.Checks = append(response.Checks, checkWithRef("version", "Candidate identity captured.", versionRef))

	doctorResult, doctorRef, err := run("verify.doctor", "verify", "doctor")
	if err != nil || doctorResult.ExitCode != 0 {
		if err != nil {
			return fail("verify_doctor", err.Error(), "Repair the isolated verification environment.", true)
		}
		return fail("verify_doctor", commandSummary(doctorResult), "Repair the isolated verification environment.", true)
	}
	response.Checks = append(response.Checks, checkWithRef("verify_doctor", "Verification environment is healthy.", doctorRef))

	initial, initialRef, err := run("flow.status.uninitialized", "flow", "status", "--root", requirementRoot)
	if err != nil {
		return fail("not_initialized", err.Error(), "Repair the candidate command runner.", true)
	}
	if initial.ExitCode != 1 || commandErrorCode(initial) != "NOT_INITIALIZED" {
		return fail("not_initialized", commandSummary(initial), "Restore the public NOT_INITIALIZED contract.", false)
	}
	response.Checks = append(response.Checks, checkWithRef("not_initialized", "Fresh Requirement reports NOT_INITIALIZED.", initialRef))
	evidence.manifest.StatusChanges = append(evidence.manifest.StatusChanges, statusChange{CommandRef: initialRef, From: "absent", To: "not_initialized"})

	before, err := businessFingerprint(requirementRoot)
	if err != nil {
		return fail("before_snapshot", err.Error(), "Inspect the Requirement filesystem.", true)
	}
	if err := writeJSON(filepath.Join(evidence.dir, "snapshots", "before", "fingerprint.json"), before); err != nil {
		return nil, publicError(erroridl.ErrorCode_LOCAL_COMMIT_FAILED, err.Error())
	}
	initArgs := []string{"flow", "init", "--root", requirementRoot, "--workflow", workflowID, "--title", "Fanloop verification smoke"}
	initDry, initDryRef, err := run("flow.init.dry-run", append(initArgs, "--dry-run")...)
	if err != nil || initDry.ExitCode != 0 {
		if err != nil {
			return fail("init_dry_run", err.Error(), "Repair the candidate command runner.", true)
		}
		return fail("init_dry_run", commandSummary(initDry), "Restore flow init dry-run.", false)
	}
	afterInitDry, err := businessFingerprint(requirementRoot)
	if err != nil {
		return fail("init_dry_run", err.Error(), "Inspect the Requirement filesystem.", true)
	}
	changed := changedPaths(before, afterInitDry)
	passed := len(changed) == 0
	evidence.manifest.SideEffects = append(evidence.manifest.SideEffects, sideEffect{Name: "init_dry_run", Expected: "no business durable mutation", ChangedPaths: changed, Passed: passed})
	if !passed {
		return fail("init_dry_run", "Dry-run changed business durable files.", "Restore dry-run isolation.", false)
	}
	response.Checks = append(response.Checks, checkWithRef("init_dry_run", "Init dry-run preserved business durable state.", initDryRef))

	initResult, initRef, err := run("flow.init", initArgs...)
	if err != nil || initResult.ExitCode != 0 {
		if err != nil {
			return fail("init", err.Error(), "Repair the candidate command runner.", true)
		}
		return fail("init", commandSummary(initResult), "Restore flow initialization.", false)
	}
	response.Checks = append(response.Checks, checkWithRef("init", "Requirement initialized.", initRef))
	afterInit, err := businessFingerprint(requirementRoot)
	if err != nil {
		return fail("init", err.Error(), "Inspect the Requirement filesystem.", true)
	}
	initChanges := changedPaths(afterInitDry, afterInit)
	evidence.manifest.SideEffects = append(evidence.manifest.SideEffects, sideEffect{Name: "init", Expected: "business durable state created", ChangedPaths: initChanges, Passed: len(initChanges) > 0})
	if len(initChanges) == 0 {
		return fail("init", "Initialization created no business durable files.", "Restore Flow persistence.", false)
	}
	statusAfterInit, statusAfterInitRef, err := run("flow.status.after-init", "flow", "status", "--root", requirementRoot)
	if err != nil || statusAfterInit.ExitCode != 0 {
		if err != nil {
			return fail("status_after_init", err.Error(), "Repair the candidate command runner.", true)
		}
		return fail("status_after_init", commandSummary(statusAfterInit), "Restore flow status after init.", false)
	}
	statusBefore, err := parseSmokeStatus(statusAfterInit.Stdout)
	if err != nil || statusBefore.State.Current == nil || statusBefore.State.Current.Context.StepID == "" {
		return fail("status_after_init", fmt.Sprintf("current Step is unavailable: %v", err), "Restore the initialized Workflow status.", false)
	}
	stepBefore := statusBefore.State.Current.Context.StepID
	evidence.manifest.StatusChanges = append(evidence.manifest.StatusChanges, statusChange{CommandRef: statusAfterInitRef, From: "not_initialized", To: stepBefore})
	response.Checks = append(response.Checks, checkWithRef("status_after_init", "Status exposes the initialized entry Step.", statusAfterInitRef))

	resultInput, expectedStatus, err := smokeResultInput(statusBefore)
	if err != nil {
		return fail("result_input", err.Error(), "Fix the Workflow so its first public Route has satisfiable typed Conditions.", false)
	}
	beforeResultDry, err := businessFingerprint(requirementRoot)
	if err != nil {
		return fail("result_dry_run", err.Error(), "Inspect the Requirement filesystem.", true)
	}
	resultDry, resultDryRef, err := run("flow.report.result.dry-run", "flow", "report", "result", "--root", requirementRoot, "--input", resultInput, "--dry-run")
	if err != nil || resultDry.ExitCode != 0 {
		if err != nil {
			return fail("result_dry_run", err.Error(), "Repair the candidate command runner.", true)
		}
		return fail("result_dry_run", commandSummary(resultDry), "Restore result dry-run.", false)
	}
	afterResultDry, err := businessFingerprint(requirementRoot)
	if err != nil {
		return fail("result_dry_run", err.Error(), "Inspect the Requirement filesystem.", true)
	}
	changed = changedPaths(beforeResultDry, afterResultDry)
	passed = len(changed) == 0
	evidence.manifest.SideEffects = append(evidence.manifest.SideEffects, sideEffect{Name: "result_dry_run", Expected: "no business durable mutation", ChangedPaths: changed, Passed: passed})
	if !passed {
		return fail("result_dry_run", "Dry-run changed business durable files.", "Restore dry-run isolation.", false)
	}
	response.Checks = append(response.Checks, checkWithRef("result_dry_run", "Result dry-run preserved business durable state.", resultDryRef))

	resultReal, resultRef, err := run("flow.report.result", "flow", "report", "result", "--root", requirementRoot, "--input", resultInput)
	if err != nil || resultReal.ExitCode != 0 {
		if err != nil {
			return fail("result", err.Error(), "Repair the candidate command runner.", true)
		}
		return fail("result", commandSummary(resultReal), "Restore result routing.", false)
	}
	response.Checks = append(response.Checks, checkWithRef("result", "Real result advanced the Workflow.", resultRef))
	afterResult, err := businessFingerprint(requirementRoot)
	if err != nil {
		return fail("result", err.Error(), "Inspect the Requirement filesystem.", true)
	}
	resultChanges := changedPaths(afterResultDry, afterResult)
	evidence.manifest.SideEffects = append(evidence.manifest.SideEffects, sideEffect{Name: "result", Expected: "business durable state advanced", ChangedPaths: resultChanges, Passed: len(resultChanges) > 0})
	if len(resultChanges) == 0 {
		return fail("result", "Real result changed no business durable files.", "Restore Flow persistence.", false)
	}
	statusAfterResult, statusAfterResultRef, err := run("flow.status.after-result", "flow", "status", "--root", requirementRoot)
	if err != nil || statusAfterResult.ExitCode != 0 {
		if err != nil {
			return fail("status_after_result", err.Error(), "Repair the candidate command runner.", true)
		}
		return fail("status_after_result", commandSummary(statusAfterResult), "Restore flow status after result.", false)
	}
	statusAfter, err := parseSmokeStatus(statusAfterResult.Stdout)
	if err != nil {
		return fail("status_after_result", err.Error(), "Restore flow status after result.", false)
	}
	stepAfter := statusAfter.State.Status
	if statusAfter.State.Current != nil {
		stepAfter = statusAfter.State.Current.Context.StepID
	}
	if stepAfter != expectedStatus {
		return fail("status_after_result", fmt.Sprintf("current state = %q, want %q", stepAfter, expectedStatus), "Restore the configured Flow Route.", false)
	}
	evidence.manifest.StatusChanges = append(evidence.manifest.StatusChanges, statusChange{CommandRef: statusAfterResultRef, From: stepBefore, To: stepAfter})
	response.Checks = append(response.Checks, checkWithRef("status_after_result", "Status confirms the next Step.", statusAfterResultRef))

	requirementDoctor, requirementDoctorRef, err := run("doctor.requirement", "doctor", "--root", requirementRoot)
	if err != nil || requirementDoctor.ExitCode != 0 {
		if err != nil {
			return fail("requirement_doctor", err.Error(), "Repair the candidate command runner.", true)
		}
		return fail("requirement_doctor", commandSummary(requirementDoctor), "Repair the Requirement state.", false)
	}
	response.Checks = append(response.Checks, checkWithRef("requirement_doctor", "Requirement Doctor completed without unhealthy checks.", requirementDoctorRef))

	snapshotDir := filepath.Join(evidence.dir, "snapshots", "after")
	snapshotResult, snapshotRef, err := run("verify.snapshot", "verify", "snapshot", "--root", requirementRoot, "--evidence-dir", snapshotDir)
	if err != nil || snapshotResult.ExitCode != 0 {
		if err != nil {
			return fail("snapshot", err.Error(), "Repair the candidate command runner.", true)
		}
		return fail("snapshot", commandSummary(snapshotResult), "Repair verification snapshot capture.", false)
	}
	var snapshotData struct {
		SnapshotDir string   `json:"snapshot_dir"`
		Files       []string `json:"files"`
	}
	requiredSnapshotFiles := []string{
		"public/flow-status.stdout.json",
		"public/trace-status.stdout.json",
		"public/trace-render.stdout.json",
		"public/card-current.stdout.json",
		"public/card-panorama.stdout.json",
		"files/.fanloop/flow/state.json",
		"files/.fanloop/output/state.json",
		"files/.fanloop/trace/events.jsonl",
		"files/.fanloop/log/cli.jsonl",
	}
	if err := decodeData(snapshotResult.Stdout, &snapshotData); err != nil || filepath.Clean(snapshotData.SnapshotDir) != filepath.Clean(snapshotDir) || !containsAll(snapshotData.Files, requiredSnapshotFiles) {
		return fail("snapshot", fmt.Sprintf("snapshot output is incomplete: %v", err), "Restore snapshot evidence reporting.", false)
	}
	response.Checks = append(response.Checks, checkWithRef("snapshot", "Status and durable projections were captured.", snapshotRef))
	if content, readErr := os.ReadFile(filepath.Join(requirementRoot, ".fanloop", "log", "cli.jsonl")); readErr == nil {
		if err := writePrivate(filepath.Join(evidence.dir, "requirement-cli.jsonl"), content); err != nil {
			return nil, publicError(erroridl.ErrorCode_LOCAL_COMMIT_FAILED, err.Error())
		}
	}

	response.Outcome = verifyidl.VerificationOutcome_passed
	runtime.finishSmokeCleanup(ctx, env, evidence, response, runID, workRoot)
	if !response.CleanupComplete || !evidence.manifest.Cleanup.DryRunPassed {
		response.Outcome = verifyidl.VerificationOutcome_blocked
	}
	return finish()
}

func (runtime Runtime) Snapshot(ctx context.Context, requirementRoot string, request *verifyidl.VerifySnapshotRequest) (*verifyidl.VerifySnapshotResponse, error) {
	if request == nil {
		return nil, publicError(erroridl.ErrorCode_INVALID_ARGUMENT, "request is required")
	}
	root, err := safeAbsolute(requirementRoot)
	if err != nil {
		return nil, publicError(erroridl.ErrorCode_INVALID_ARGUMENT, "requirement root "+err.Error())
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return nil, publicError(erroridl.ErrorCode_INVALID_ARGUMENT, "requirement root is not an existing directory")
	}
	destination := ""
	if request.EvidenceDir != nil {
		destination = strings.TrimSpace(*request.EvidenceDir)
		if destination == "" {
			return nil, publicError(erroridl.ErrorCode_INVALID_ARGUMENT, "evidence_dir must not be empty")
		}
		if _, err := safeAbsolute(destination); err != nil {
			return nil, publicError(erroridl.ErrorCode_INVALID_ARGUMENT, "evidence_dir "+err.Error())
		}
	} else {
		destination = filepath.Join(runtime.DataRoot, "verification", "runs", "snapshot-"+runtime.NewRunID(), "snapshots", "current")
	}
	if containsPath(filepath.Join(root, ".fanloop"), destination) {
		return nil, publicError(erroridl.ErrorCode_INVALID_ARGUMENT, "evidence_dir must be outside requirement .fanloop state")
	}
	files, err := runtime.captureSnapshot(ctx, root, filepath.Clean(destination), os.Environ())
	if err != nil {
		var failure *erroridl.PublicError
		if errors.As(err, &failure) {
			return nil, failure
		}
		return nil, publicError(erroridl.ErrorCode_LOCAL_COMMIT_FAILED, err.Error())
	}
	return &verifyidl.VerifySnapshotResponse{SnapshotDir: filepath.Clean(destination), Files: files}, nil
}

func (runtime Runtime) Cleanup(_ context.Context, dryRun bool, request *verifyidl.VerifyCleanupRequest) (*verifyidl.VerifyCleanupResponse, error) {
	if request == nil {
		return nil, publicError(erroridl.ErrorCode_INVALID_ARGUMENT, "request is required")
	}
	runID := strings.TrimSpace(request.RunId)
	if !runIDPattern.MatchString(runID) {
		return nil, publicError(erroridl.ErrorCode_INVALID_ARGUMENT, "run_id must be 32 lowercase hexadecimal characters")
	}
	dataRoot, err := safeAbsolute(runtime.DataRoot)
	if err != nil {
		return nil, publicError(erroridl.ErrorCode_INVALID_ARGUMENT, "data root "+err.Error())
	}
	workParent := filepath.Join(dataRoot, "verification", "work")
	if info, err := os.Lstat(workParent); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, publicError(erroridl.ErrorCode_LOCAL_COMMIT_FAILED, "verification work directory must not be a symlink")
	} else if err != nil && !os.IsNotExist(err) {
		return nil, publicError(erroridl.ErrorCode_LOCAL_COMMIT_FAILED, err.Error())
	}
	workRoot := filepath.Join(workParent, runID)
	paths := []string{}
	if _, err := os.Lstat(workRoot); err == nil {
		paths = append(paths, workRoot)
	} else if !os.IsNotExist(err) {
		return nil, publicError(erroridl.ErrorCode_LOCAL_COMMIT_FAILED, err.Error())
	}
	if dryRun || len(paths) == 0 {
		return &verifyidl.VerifyCleanupResponse{RunId: runID, Paths: paths, Removed: false}, nil
	}
	if err := os.RemoveAll(workRoot); err != nil {
		return nil, publicError(erroridl.ErrorCode_LOCAL_COMMIT_FAILED, err.Error())
	}
	return &verifyidl.VerifyCleanupResponse{RunId: runID, Paths: paths, Removed: true}, nil
}

func (runtime Runtime) finishSmokeCleanup(ctx context.Context, env []string, evidence *evidenceRun, response *verifyidl.VerifySmokeResponse, runID, workRoot string) {
	if err := evidence.writeManifest(); err != nil {
		return
	}
	cleanupEnv := replaceEnvironment(env, "FANLOOP_DATA_HOME", runtime.DataRoot)
	dry, dryErr := runtime.run(ctx, cleanupEnv, "verify.cleanup.dry-run", "verify", "cleanup", "--run-id", runID, "--dry-run")
	dryRef := ""
	var dryRecordErr error
	if dryErr == nil {
		dryRef, dryRecordErr = evidence.recordCommand(dry)
	}
	dryData := struct {
		RunID   string   `json:"run_id"`
		Paths   []string `json:"paths"`
		Removed bool     `json:"removed"`
	}{}
	dryDecodeErr := decodeData(dry.Stdout, &dryData)
	dryPassed := dryErr == nil && dryRecordErr == nil && dry.ExitCode == 0 && dryDecodeErr == nil && dryData.RunID == runID && !dryData.Removed && len(dryData.Paths) == 1 && filepath.Clean(dryData.Paths[0]) == filepath.Clean(workRoot)
	if _, err := os.Stat(workRoot); err != nil {
		dryPassed = false
	}
	if !dryPassed {
		summary := commandSummary(dry)
		if dryErr != nil {
			summary = dryErr.Error()
		} else if dryRecordErr != nil {
			summary = "record cleanup dry-run evidence: " + dryRecordErr.Error()
		}
		response.Checks = append(response.Checks, failedCheck("cleanup_dry_run", summary, "Repair cleanup dry-run before relying on cleanup."))
		evidence.manifest.Cleanup = cleanupEvidence{DryRunPassed: false, Removed: false, Paths: []string{workRoot}}
		return
	}
	response.Checks = append(response.Checks, checkWithRef("cleanup_dry_run", "Cleanup dry-run preserved run-owned resources.", dryRef))
	real, realErr := runtime.run(ctx, cleanupEnv, "verify.cleanup", "verify", "cleanup", "--run-id", runID)
	realRef := ""
	var realRecordErr error
	if realErr == nil {
		realRef, realRecordErr = evidence.recordCommand(real)
	}
	realData := struct {
		RunID   string   `json:"run_id"`
		Paths   []string `json:"paths"`
		Removed bool     `json:"removed"`
	}{}
	realDecodeErr := decodeData(real.Stdout, &realData)
	removed := realErr == nil && real.ExitCode == 0 && realDecodeErr == nil && realData.RunID == runID && realData.Removed && len(realData.Paths) == 1 && filepath.Clean(realData.Paths[0]) == filepath.Clean(workRoot)
	if _, err := os.Stat(workRoot); !os.IsNotExist(err) {
		removed = false
	}
	evidencePreserved := false
	if info, err := os.Stat(evidence.dir); err == nil && info.IsDir() {
		evidencePreserved = true
	}
	evidence.manifest.Cleanup = cleanupEvidence{DryRunPassed: dryPassed, Removed: removed, Paths: []string{workRoot}}
	response.CleanupComplete = removed && realRecordErr == nil && evidencePreserved
	if response.CleanupComplete {
		response.Checks = append(response.Checks, checkWithRef("cleanup", "Run-owned resources were removed and evidence was preserved.", realRef))
	} else {
		summary := commandSummary(real)
		if realErr != nil {
			summary = realErr.Error()
		} else if realRecordErr != nil {
			summary = "record cleanup evidence: " + realRecordErr.Error()
		} else if !evidencePreserved {
			summary = "evidence directory was not preserved"
		}
		response.Checks = append(response.Checks, failedCheck("cleanup", summary, "Inspect and remove only the reported run-owned path."))
	}
}

func (runtime Runtime) isolatedEnvironment(workRoot string) ([]string, error) {
	dataRoot := filepath.Join(workRoot, "data")
	skillRoot := filepath.Join(workRoot, "skills")
	roots := map[string]string{
		"FANLOOP_CODEX_SKILLS_ROOT":  filepath.Join(skillRoot, "codex"),
		"FANLOOP_AGENT_SKILLS_ROOT":  filepath.Join(skillRoot, "agent"),
		"FANLOOP_TRAE_SKILLS_ROOT":   filepath.Join(skillRoot, "trae"),
		"FANLOOP_CLAUDE_SKILLS_ROOT": filepath.Join(skillRoot, "claude"),
	}
	for _, root := range roots {
		if err := os.MkdirAll(root, 0o700); err != nil {
			return nil, err
		}
	}
	releaseRoot := release.RootForExecutable(runtime.Executable)
	target := filepath.Join(releaseRoot, filepath.FromSlash(release.ExposedSkillPath))
	if info, err := os.Stat(target); err == nil && info.IsDir() {
		for _, root := range roots {
			if err := os.Symlink(target, filepath.Join(root, release.ExposedSkillName)); err != nil {
				return nil, err
			}
		}
	}
	environment := withoutEnvironment(os.Environ(), "FANLOOP_DATA_HOME", "FANLOOP_CONFIG_ROOT", "FANLOOP_CODEX_SKILLS_ROOT", "FANLOOP_AGENT_SKILLS_ROOT", "FANLOOP_TRAE_SKILLS_ROOT", "FANLOOP_CLAUDE_SKILLS_ROOT", "BOTMUX_CHAT_ID", "BOTMUX_SESSION_ID")
	environment = append(environment, "FANLOOP_DATA_HOME="+dataRoot, "FANLOOP_CONFIG_ROOT="+runtime.ConfigRoot)
	for name, path := range roots {
		environment = append(environment, name+"="+path)
	}
	return environment, nil
}

func (runtime Runtime) run(ctx context.Context, env []string, id string, args ...string) (commandResult, error) {
	if runtime.Run != nil {
		result, err := runtime.Run(ctx, runtime.Executable, env, args)
		result.ID = id
		return result, err
	}
	result, err := executeCommand(ctx, runtime.Executable, env, args)
	result.ID = id
	return result, err
}

func executeCommand(ctx context.Context, executable string, env []string, args []string) (commandResult, error) {
	started := time.Now()
	command := exec.CommandContext(ctx, executable, args...)
	command.Env = env
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	result := commandResult{ID: strings.Join(args, " "), Argv: append([]string{executable}, args...), StartedAt: started, Duration: time.Since(started), Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	if err == nil {
		return result, nil
	}
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		result.ExitCode = exit.ExitCode()
		return result, nil
	}
	return result, err
}

func decodeEnvelope(content []byte) (cliEnvelope, error) {
	var envelope cliEnvelope
	decoder := json.NewDecoder(bytes.NewReader(content))
	if err := decoder.Decode(&envelope); err != nil {
		return envelope, err
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("trailing JSON value")
		}
		return envelope, err
	}
	return envelope, nil
}

func decodeData(content []byte, target any) error {
	envelope, err := decodeEnvelope(content)
	if err != nil {
		return err
	}
	if !envelope.OK {
		if envelope.Error != nil {
			return fmt.Errorf("%s: %s", envelope.Error.Code, envelope.Error.Message)
		}
		return fmt.Errorf("command returned ok=false")
	}
	return json.Unmarshal(envelope.Data, target)
}

type smokeStatus struct {
	State struct {
		Status  string `json:"status"`
		Current *struct {
			Context struct {
				StepID string `json:"step_id"`
			} `json:"context"`
			Conditions []struct {
				ID     string `json:"id"`
				Output struct {
					Type     string   `json:"type"`
					Values   []string `json:"values"`
					Minimum  *int64   `json:"minimum"`
					Maximum  *int64   `json:"maximum"`
					MinItems *int32   `json:"min_items"`
					MaxItems *int32   `json:"max_items"`
				} `json:"output"`
			} `json:"conditions"`
			AvailableRoutes []struct {
				Direction string `json:"direction"`
				When      struct {
					AnyOf [][]string `json:"any_of"`
				} `json:"when"`
				Route map[string]any `json:"route"`
			} `json:"available_routes"`
		} `json:"current"`
	} `json:"state"`
}

func parseSmokeStatus(content []byte) (smokeStatus, error) {
	var value smokeStatus
	if err := decodeData(content, &value); err != nil {
		return value, err
	}
	return value, nil
}

func smokeResultInput(status smokeStatus) (string, string, error) {
	if status.State.Current == nil {
		return "", "", fmt.Errorf("status has no current Step")
	}
	conditions := map[string]struct {
		Type     string
		Values   []string
		Minimum  *int64
		Maximum  *int64
		MinItems *int32
		MaxItems *int32
	}{}
	for _, condition := range status.State.Current.Conditions {
		conditions[condition.ID] = struct {
			Type     string
			Values   []string
			Minimum  *int64
			Maximum  *int64
			MinItems *int32
			MaxItems *int32
		}{condition.Output.Type, condition.Output.Values, condition.Output.Minimum, condition.Output.Maximum, condition.Output.MinItems, condition.Output.MaxItems}
	}
	for _, route := range status.State.Current.AvailableRoutes {
		if route.Direction != "flow" || len(route.When.AnyOf) == 0 {
			continue
		}
		for _, alternative := range route.When.AnyOf {
			results := make([]map[string]any, 0, len(alternative))
			valid := len(alternative) > 0
			for _, conditionID := range alternative {
				definition, ok := conditions[conditionID]
				if !ok {
					valid = false
					break
				}
				value, err := syntheticOutput(conditionID, definition.Type, definition.Values, definition.Minimum, definition.Maximum, definition.MinItems, definition.MaxItems)
				if err != nil {
					valid = false
					break
				}
				results = append(results, map[string]any{"condition_id": conditionID, "output": map[string]any{"type": definition.Type, "value": value}})
			}
			if !valid {
				continue
			}
			expected := ""
			if next, ok := route.Route["next_step_id"].(string); ok && next != "" {
				expected = next
			} else if terminal, ok := route.Route["terminal"].(bool); ok && terminal {
				expected = "completed"
			} else {
				continue
			}
			request := map[string]any{
				"step_id":           status.State.Current.Context.StepID,
				"condition_results": results,
				"evidence":          []map[string]any{{"source": "system", "content": "synthetic verification evidence", "ref": "verify-smoke"}},
				"summary":           "verification smoke advances one Step",
				"route":             route.Route,
			}
			content, err := json.Marshal(request)
			return string(content), expected, err
		}
	}
	return "", "", fmt.Errorf("current Step has no satisfiable forward Route")
}

func syntheticOutput(id, outputType string, values []string, minimum, maximum *int64, minItems, maxItems *int32) (any, error) {
	switch outputType {
	case "string":
		return "verification-" + id, nil
	case "boolean":
		return true, nil
	case "integer":
		value := int64(1)
		if minimum != nil {
			value = *minimum
		}
		if maximum != nil && value > *maximum {
			value = *maximum
		}
		return value, nil
	case "path":
		return filepath.ToSlash(filepath.Join(".fanloop-verification", id+".txt")), nil
	case "url":
		return "https://example.com/fanloop-verification/" + id, nil
	case "url_list":
		count := int32(1)
		if minItems != nil {
			count = *minItems
		}
		if maxItems != nil && count > *maxItems {
			count = *maxItems
		}
		result := make([]string, count)
		for index := range result {
			result[index] = fmt.Sprintf("https://example.com/fanloop-verification/%s/%d", id, index+1)
		}
		return result, nil
	case "enum_value":
		if len(values) == 0 {
			return nil, fmt.Errorf("Condition %s has no enum values", id)
		}
		return values[0], nil
	case "object":
		return map[string]any{"verification": true}, nil
	default:
		return nil, fmt.Errorf("Condition %s has unsupported Output type %q", id, outputType)
	}
}

func commandErrorCode(result commandResult) string {
	envelope, err := decodeEnvelope(result.Stderr)
	if err != nil || envelope.Error == nil {
		return ""
	}
	return envelope.Error.Code
}

func commandFailure(result commandResult) error {
	envelope, err := decodeEnvelope(result.Stderr)
	if err == nil && envelope.Error != nil {
		code, parseErr := erroridl.ErrorCodeFromString(envelope.Error.Code)
		if parseErr == nil && code != erroridl.ErrorCode_unspecified {
			return publicError(code, envelope.Error.Message)
		}
	}
	return publicError(erroridl.ErrorCode_INTERNAL, commandSummary(result))
}

func commandSummary(result commandResult) string {
	message := strings.TrimSpace(string(result.Stderr))
	if message == "" {
		message = strings.TrimSpace(string(result.Stdout))
	}
	if message == "" {
		message = fmt.Sprintf("command exited %d", result.ExitCode)
	}
	return message
}

func passedCheck(id, summary string) *verifyidl.VerificationCheck {
	return &verifyidl.VerificationCheck{Id: id, Status: verifyidl.VerificationCheckStatus_passed, Summary: summary}
}

func checkWithRef(id, summary, reference string) *verifyidl.VerificationCheck {
	return &verifyidl.VerificationCheck{Id: id, Status: verifyidl.VerificationCheckStatus_passed, Summary: summary, EvidenceRef: &reference}
}

func failedCheck(id, summary, hint string) *verifyidl.VerificationCheck {
	return &verifyidl.VerificationCheck{Id: id, Status: verifyidl.VerificationCheckStatus_failed, Summary: summary, Hint: &hint}
}

func publicError(code erroridl.ErrorCode, message string) *erroridl.PublicError {
	return clierrs.NewCode(code, message, nil)
}

func withoutEnvironment(environment []string, keys ...string) []string {
	blocked := map[string]bool{}
	for _, key := range keys {
		blocked[key] = true
	}
	result := make([]string, 0, len(environment))
	for _, item := range environment {
		key, _, _ := strings.Cut(item, "=")
		if !blocked[key] {
			result = append(result, item)
		}
	}
	return result
}

func replaceEnvironment(environment []string, key, value string) []string {
	result := withoutEnvironment(environment, key)
	return append(result, key+"="+value)
}

func containsAll(values, required []string) bool {
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		seen[value] = true
	}
	for _, value := range required {
		if !seen[value] {
			return false
		}
	}
	return true
}
