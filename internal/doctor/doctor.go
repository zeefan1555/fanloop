package doctor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	gort "runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/zeefan1555/commonloop/internal/buildinfo"
	cardruntime "github.com/zeefan1555/commonloop/internal/card"
	idl "github.com/zeefan1555/commonloop/internal/idl/opsidl"
	"github.com/zeefan1555/commonloop/internal/release"
	"github.com/zeefan1555/commonloop/internal/state"
	"github.com/zeefan1555/commonloop/internal/store"
	"github.com/zeefan1555/commonloop/internal/workflow"
)

const (
	statusPass = idl.DoctorCheckStatus_passed
	statusWarn = idl.DoctorCheckStatus_warning
	statusFail = idl.DoctorCheckStatus_failed
	statusSkip = idl.DoctorCheckStatus_skipped
)

type Runtime struct {
	ReleaseRoot string
	BinaryPath  string
	SkillRoots  []string
}

func DefaultRuntime() Runtime {
	executable, _ := os.Executable()
	releaseRoot := release.RootForExecutable(executable)
	if dataRoot, err := release.DefaultDataRoot(); err == nil {
		resolvedReleases, releasesErr := filepath.EvalSymlinks(filepath.Join(dataRoot, "releases"))
		resolvedRelease, releaseErr := filepath.EvalSymlinks(releaseRoot)
		if releasesErr == nil && releaseErr == nil {
			if relative, err := filepath.Rel(resolvedReleases, resolvedRelease); err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				releaseRoot = filepath.Join(dataRoot, "current")
			}
		}
	}
	var skillRoots []string
	if roots, err := release.DefaultSkillRoots(); err == nil {
		skillRoots = roots.Values()
	}
	return Runtime{ReleaseRoot: releaseRoot, BinaryPath: executable, SkillRoots: skillRoots}
}

func (runtime Runtime) Run(requirementRoot string) *idl.DoctorResponse {
	checks := runtime.installationChecks()
	scope := idl.DoctorScope_installation
	if requirementRoot != "" {
		scope = idl.DoctorScope_requirement
		checks = append(checks, requirementChecks(requirementRoot)...)
	}
	status := idl.DoctorStatus_healthy
	for _, check := range checks {
		if check.Status == statusFail {
			status = idl.DoctorStatus_unhealthy
			break
		}
		if check.Status == statusWarn {
			status = idl.DoctorStatus_warning
		}
	}
	return &idl.DoctorResponse{Scope: scope, Status: status, Checks: checks}
}

func (runtime Runtime) installationChecks() []*idl.DoctorCheck {
	manifest, err := release.Load(runtime.ReleaseRoot)
	if os.IsNotExist(err) {
		return []*idl.DoctorCheck{
			check("release_manifest", statusWarn, "No installed release manifest; this appears to be a source build.", "Install a matched Commonloop release for production use."),
			skipped("binary_checksum", "Release manifest is unavailable."),
			skipped("skills", "Release manifest is unavailable."),
			skipped("workflows", "Release manifest is unavailable."),
			skipped("version_drift", "Release manifest is unavailable."),
		}
	}
	if err != nil {
		return []*idl.DoctorCheck{
			check("release_manifest", statusFail, "Release manifest is invalid: "+err.Error(), "Reinstall this Commonloop release."),
			skipped("binary_checksum", "Release manifest is invalid."),
			skipped("skills", "Release manifest is invalid."),
			skipped("workflows", "Release manifest is invalid."),
			skipped("version_drift", "Release manifest is invalid."),
		}
	}
	checks := []*idl.DoctorCheck{check("release_manifest", statusPass, "Release manifest is valid.", "")}
	checks = append(checks, runtime.binaryCheck(manifest), skillCheck(runtime.ReleaseRoot, manifest))
	if len(runtime.SkillRoots) > 0 {
		checks = append(checks, skillLinkCheck(runtime.ReleaseRoot, runtime.SkillRoots, manifest))
	}
	checks = append(checks, workflowCheck(runtime.ReleaseRoot, manifest), versionCheck(manifest))
	return checks
}

func (runtime Runtime) binaryCheck(manifest release.Manifest) *idl.DoctorCheck {
	asset, ok := manifest.Asset(gort.GOOS, gort.GOARCH)
	if !ok {
		return check("binary_checksum", statusFail, "Release has no asset for this platform.", "Install a release built for this OS and architecture.")
	}
	digest, err := release.FileDigest(runtime.BinaryPath)
	if err != nil || digest != asset.BinarySha256 {
		return check("binary_checksum", statusFail, "CLI binary checksum does not match the release manifest.", "Reinstall this Commonloop release.")
	}
	return check("binary_checksum", statusPass, "CLI binary checksum matches.", "")
}

func skillCheck(root string, manifest release.Manifest) *idl.DoctorCheck {
	for _, skill := range manifest.Skills {
		path, err := release.Resolve(root, skill.Path)
		if err != nil {
			return check("skills", statusFail, err.Error(), "Reinstall this Commonloop release.")
		}
		digest, err := release.DirectoryDigest(path)
		if err != nil || digest != skill.Sha256 {
			return check("skills", statusFail, fmt.Sprintf("Skill %q checksum does not match.", skill.Name), "Reinstall this Commonloop release.")
		}
	}
	return check("skills", statusPass, "All packaged Skills match the release manifest.", "")
}

func skillLinkCheck(releaseRoot string, roots []string, manifest release.Manifest) *idl.DoctorCheck {
	skill, _ := manifest.Skill(release.ExposedSkillName)
	for _, root := range roots {
		path := filepath.Join(root, skill.Name)
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			return check("skill_links", statusFail, fmt.Sprintf("Skill link %q is missing or is not managed by Commonloop.", path), "Reinstall Commonloop without replacing user-owned paths.")
		}
		actual, actualErr := os.Readlink(path)
		if !filepath.IsAbs(actual) {
			actual = filepath.Join(filepath.Dir(path), actual)
		}
		expected := filepath.Join(releaseRoot, filepath.FromSlash(skill.Path))
		if actualErr != nil || filepath.Clean(actual) != filepath.Clean(expected) {
			return check("skill_links", statusFail, fmt.Sprintf("Skill link %q does not point to the current release.", path), "Reinstall Commonloop to restore the managed Skill links.")
		}
	}
	return check("skill_links", statusPass, "Codex, Agent, Trae, and Claude Workflow Skill links point to the current release.", "")
}

func workflowCheck(root string, manifest release.Manifest) *idl.DoctorCheck {
	for _, item := range manifest.Workflows {
		path, err := release.Resolve(root, item.Path)
		if err != nil {
			return check("workflows", statusFail, err.Error(), "Reinstall this Commonloop release.")
		}
		loaded, err := workflow.LoadDirectory(path)
		if err != nil || loaded.Ref.Digest != item.Sha256 {
			return check("workflows", statusFail, fmt.Sprintf("Workflow %q checksum does not match.", item.Id), "Reinstall this Commonloop release.")
		}
		if loaded.Ref.ID != item.Id {
			return check("workflows", statusFail, fmt.Sprintf("Workflow %q is invalid.", item.Id), "Reinstall this Commonloop release.")
		}
	}
	return check("workflows", statusPass, "All packaged Workflows match the release manifest.", "")
}

func versionCheck(manifest release.Manifest) *idl.DoctorCheck {
	if manifest.ReleaseVersion != buildinfo.ReleaseVersion || manifest.Cli.Version != buildinfo.CLIVersion {
		return check("version_drift", statusFail, "Release manifest and running CLI versions differ.", "Switch to one complete Commonloop release.")
	}
	current, err := buildinfo.GetEmbedded()
	if err != nil || current.StateSchema.WriteVersion != manifest.StateSchema.WriteVersion || !equalInts(current.StateSchema.ReadVersions, manifest.StateSchema.ReadVersions) {
		return check("version_drift", statusFail, "State Schema support differs from the release manifest.", "Switch to one complete Commonloop release.")
	}
	actual := map[string]string{}
	embedded, listErr := workflow.List()
	if listErr != nil {
		return check("version_drift", statusFail, "Embedded Workflows cannot be read.", "Switch to one complete Commonloop release.")
	}
	for _, item := range embedded {
		actual[item.Ref.ID] = item.Ref.Digest
	}
	if len(actual) != len(manifest.Workflows) {
		return check("version_drift", statusFail, "Embedded Workflows differ from the release manifest.", "Switch to one complete Commonloop release.")
	}
	for _, item := range manifest.Workflows {
		if actual[item.Id] != item.Sha256 {
			return check("version_drift", statusFail, "Embedded Workflows differ from the release manifest.", "Switch to one complete Commonloop release.")
		}
	}
	return check("version_drift", statusPass, "CLI and packaged component versions are aligned.", "")
}

func requirementChecks(root string) []*idl.DoctorCheck {
	if !filepath.IsAbs(root) {
		return rootFailure("Requirement root must be absolute.", "Pass an absolute requirement directory.")
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return rootFailure("Requirement root is not an existing directory.", "Create or select the requirement directory.")
	}
	checks := []*idl.DoctorCheck{check("requirement_root", statusPass, "Requirement root exists.", "")}
	stateContent, readErr := os.ReadFile(filepath.Join(root, ".commonloop", "flow", "state.json"))
	var loose state.State
	jsonErr := readErr
	if jsonErr == nil {
		loose, jsonErr = state.Decode(stateContent, map[string]state.RegisteredOutput{})
	}
	if jsonErr != nil || loose.SchemaVersion != state.CurrentStateSchemaVersion {
		checks = append(checks, check("state_schema", statusFail, "State JSON or schema version is invalid.", "Repair or restore .commonloop/flow/state.json."))
	} else {
		checks = append(checks, check("state_schema", statusPass, "State schema is readable.", ""))
	}

	var loaded workflow.Loaded
	bindingOK := jsonErr == nil && loose.Release.Workflow.ID != ""
	if bindingOK {
		loaded, err = workflow.LoadRef(loose.Release.Workflow.Ref())
		bindingOK = err == nil
	}
	if jsonErr != nil {
		checks = append(checks, skipped("workflow_binding", "State JSON is unavailable."))
	} else if !bindingOK {
		checks = append(checks, check("workflow_binding", statusFail, "Bound Workflow is missing or does not match its digest.", "Use the release containing the bound Workflow."))
	} else {
		checks = append(checks, check("workflow_binding", statusPass, "Bound Workflow is available and immutable.", ""))
	}

	outputContent, outputReadErr := os.ReadFile(filepath.Join(root, ".commonloop", "output", "state.json"))
	registry, outputsErr := state.DecodeOutputRegistry(outputContent)
	if outputReadErr != nil {
		outputsErr = outputReadErr
	} else if outputsErr == nil && jsonErr == nil && registry.Workflow != loose.Release.Workflow {
		outputsErr = fmt.Errorf("Output Registry uses a different Workflow binding")
	}
	outputs := map[string]state.RegisteredOutput{}
	if outputsErr == nil {
		outputs = registry.Outputs
	}
	strict, strictErr := state.Decode(stateContent, outputs)
	if strictErr == nil && bindingOK {
		strictErr = strict.ValidateAgainst(loaded.Workflow)
	}
	if strictErr != nil {
		checks = append(checks, check("state_invariants", statusFail, "State invariants are invalid: "+strictErr.Error(), "Repair or restore .commonloop/flow/state.json."))
	} else if !bindingOK {
		checks = append(checks, skipped("state_invariants", "Bound Workflow is unavailable."))
	} else {
		checks = append(checks, check("state_invariants", statusPass, "State invariants hold.", ""))
	}

	if outputsErr != nil {
		checks = append(checks, check("outputs", statusFail, "Output Registry is invalid: "+outputsErr.Error(), "Repair or restore .commonloop/output/state.json."))
	} else if strictErr != nil || !bindingOK {
		checks = append(checks, skipped("outputs", "Valid State and Workflow are required."))
	} else if err := validateOutputs(strict, loaded.Workflow); err != nil {
		checks = append(checks, check("outputs", statusFail, err.Error(), "Repair invalid current Outputs before continuing."))
	} else {
		checks = append(checks, check("outputs", statusPass, "Current Outputs match the Workflow schema and producer ownership.", ""))
	}

	local, _ := store.New(root)
	events, eventsFailure := local.Events()
	eventsErr := error(nil)
	if eventsFailure != nil {
		eventsErr = eventsFailure
	} else {
		eventsErr = validateEvents(events, strict, strictErr == nil && outputsErr == nil, loaded.Workflow, bindingOK)
	}
	if eventsErr != nil {
		checks = append(checks, check("events", statusFail, "Event history is invalid: "+eventsErr.Error(), "Repair or restore .commonloop/trace/events.jsonl."))
	} else {
		checks = append(checks, check("events", statusPass, "Event history is valid.", ""))
	}

	if eventsErr != nil {
		checks = append(checks, skipped("trace_projection", "Valid Events are required."))
	} else {
		expected := store.RenderEvents(root, strict, loaded.Workflow, events)
		actual, err := os.ReadFile(filepath.Join(root, ".commonloop", "trace", "events.md"))
		if err != nil || !bytes.Equal(actual, expected) {
			checks = append(checks, check("trace_projection", statusWarn, "Events Markdown is missing or stale.", "Run commonloop trace render --root <requirement>."))
		} else {
			checks = append(checks, check("trace_projection", statusPass, "Events Markdown matches Events.", ""))
		}
	}

	checks = append(checks, validateCardBinding(root))
	checks = append(checks, validateCardProjection(root, strict, strictErr == nil && outputsErr == nil))

	if err := validateCardSnapshots(root); err != nil {
		checks = append(checks, check("card_snapshots", statusFail, err.Error(), "Restore the immutable Card snapshot from a trusted copy."))
	} else {
		checks = append(checks, check("card_snapshots", statusPass, "Card snapshots are valid immutable payloads.", ""))
	}
	return checks
}

func rootFailure(summary, hint string) []*idl.DoctorCheck {
	checks := []*idl.DoctorCheck{check("requirement_root", statusFail, summary, hint)}
	for _, id := range []string{"state_schema", "workflow_binding", "state_invariants", "outputs", "events", "trace_projection", "card_binding", "card_projection", "card_snapshots"} {
		checks = append(checks, skipped(id, "Requirement root is unavailable."))
	}
	return checks
}

func validateOutputs(current state.State, definition workflow.Workflow) error {
	keys := make([]string, 0, len(current.Outputs))
	for key := range current.Outputs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		output := current.Outputs[key]
		if err := definition.ValidateRegisteredOutput(key, output.Type, output.Value); err != nil {
			return err
		}
		if _, _, ok := definition.FindStep(output.ProducerStepID); !ok {
			return fmt.Errorf("Output %q has unknown producer Step %q", key, output.ProducerStepID)
		}
	}
	return nil
}

func validateEvents(events []state.Event, current state.State, hasState bool, definition workflow.Workflow, hasWorkflow bool) error {
	if hasState && hasWorkflow {
		return state.ValidateHistory(events, current, definition)
	}
	if len(events) == 0 {
		return fmt.Errorf("event history is empty")
	}
	seen := map[string]bool{}
	for index, event := range events {
		if event.OccurredAt.IsZero() || seen[event.ID] {
			return fmt.Errorf("event %q has an invalid time or duplicate ID", event.ID)
		}
		if index > 0 && event.OccurredAt.Before(events[index-1].OccurredAt) {
			return fmt.Errorf("event %q is out of order", event.ID)
		}
		if (index == 0 && event.CausedByEventID != "") || (index > 0 && event.CausedByEventID != events[index-1].ID) {
			return fmt.Errorf("event %q cause does not match the previous event", event.ID)
		}
		if hasState && event.Workflow != current.Release.Workflow {
			return fmt.Errorf("event %q uses a different Workflow binding", event.ID)
		}
		seen[event.ID] = true
	}
	if hasState && current.LastEventID != events[len(events)-1].ID {
		return fmt.Errorf("state last_event_id does not match the event tail")
	}
	return nil
}

func validateCardSnapshots(root string) error {
	directory := filepath.Join(root, ".commonloop", "card")
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() == filepath.Base(cardruntime.BindingPath(root)) || entry.Name() == filepath.Base(cardruntime.ProjectionPath(root)) {
			continue
		}
		relative := ".commonloop/card/" + entry.Name()
		if !validCardSnapshotPath(relative) || !entry.Type().IsRegular() {
			return fmt.Errorf("unexpected Card snapshot %q", relative)
		}
		content, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			return err
		}
		if !json.Valid(content) {
			return fmt.Errorf("Card snapshot %q is not valid JSON", relative)
		}
		if err := validateRenderedCardContent(content); err != nil {
			return fmt.Errorf("Card snapshot %q is invalid: %w", relative, err)
		}
	}
	return nil
}

func validateCardProjection(root string, current state.State, hasCurrent bool) *idl.DoctorCheck {
	projection, err := cardruntime.LoadProjection(root)
	if err != nil {
		return check("card_projection", statusFail, "Card projection is invalid: "+err.Error(), "Repair or restore .commonloop/card/projection.json.")
	}
	if hasCurrent {
		projected := projection.State()
		if projected.Requirement != current.Requirement || projected.Release != current.Release ||
			!reflect.DeepEqual(projected.CurrentStepID, current.CurrentStepID) || projected.CurrentStepStatus != current.CurrentStepStatus ||
			projected.CurrentStepSummary != current.CurrentStepSummary || !reflect.DeepEqual(projected.CurrentEvidence, current.CurrentEvidence) ||
			!reflect.DeepEqual(projected.Outputs, current.Outputs) {
			return check("card_projection", statusFail, "Card projection is stale.", "Run an accepted Flow report to refresh .commonloop/card/projection.json.")
		}
	}
	return check("card_projection", statusPass, "Card projection is valid and current.", "")
}

func validateCardBinding(root string) *idl.DoctorCheck {
	_, configured, err := cardruntime.LoadBinding(root)
	if err != nil {
		return check("card_binding", statusFail, "Card binding config is invalid: "+err.Error(), "Repair .commonloop/card/config.json and keep its permissions at 0600.")
	}
	if !configured {
		return check("card_binding", statusPass, "Card binding config is not configured.", "")
	}
	return check("card_binding", statusPass, "Card binding config is valid.", "")
}

func validCardSnapshotPath(path string) bool {
	const prefix = ".commonloop/card/"
	if !strings.HasPrefix(path, prefix) || strings.Contains(strings.TrimPrefix(path, prefix), "/") || !strings.HasSuffix(path, ".json") {
		return false
	}
	stem := strings.TrimSuffix(strings.TrimPrefix(path, prefix), ".json")
	const stampLength = len("20260807T004805.880691+0800")
	if len(stem) < stampLength {
		return false
	}
	if _, err := time.Parse("20060102T150405.000000-0700", stem[:stampLength]); err != nil {
		return false
	}
	if suffix := stem[stampLength:]; suffix != "" {
		value, err := strconv.Atoi(strings.TrimPrefix(suffix, "-"))
		if err != nil || !strings.HasPrefix(suffix, "-") || value < 1 {
			return false
		}
	}
	return true
}

func validateRenderedCardContent(content []byte) error {
	var markdown string
	if json.Unmarshal(content, &markdown) == nil {
		return nil
	}
	var card struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(content, &card); err != nil || card.Schema != "2.0" {
		return fmt.Errorf("Card must be a Markdown JSON string or Lark Schema 2.0 object")
	}
	return nil
}

func check(id string, status idl.DoctorCheckStatus, summary, hint string) *idl.DoctorCheck {
	value := &idl.DoctorCheck{Id: id, Status: status, Summary: summary}
	if hint != "" {
		value.Hint = &hint
	}
	return value
}

func skipped(id, summary string) *idl.DoctorCheck { return check(id, statusSkip, summary, "") }

func equalInts(left, right []int32) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
