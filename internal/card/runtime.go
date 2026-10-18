package card

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/zeefan1555/fanloop/errs"
	"github.com/zeefan1555/fanloop/internal/cardsnapshot"
	"github.com/zeefan1555/fanloop/internal/idl/cardidl"
	"github.com/zeefan1555/fanloop/internal/idl/commonidl"
	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/store"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

type Runtime struct {
	Clock func() time.Time
}

var _ cardidl.CardService = Runtime{}

func DefaultRuntime() Runtime { return Runtime{Clock: time.Now} }

func (runtime Runtime) Render(_ context.Context, root string, request *cardidl.CardRenderRequest, dryRun bool) (*cardidl.CardRenderResponse, error) {
	if request == nil {
		return nil, invalidArgument("request is required")
	}
	if err := request.IsValid(); err != nil {
		return nil, invalidArgument(err.Error())
	}
	if request.View != cardidl.CardView_current && request.View != cardidl.CardView_panorama {
		return nil, invalidArgument("view must be current or panorama")
	}
	if request.Format != cardidl.CardFormat_markdown && request.Format != cardidl.CardFormat_lark_json {
		return nil, invalidArgument("format must be markdown or lark_json")
	}
	local, failure := store.New(root)
	if failure != nil {
		return nil, failure
	}
	projection, _, err := LoadProjection(root)
	if err != nil {
		return nil, errs.NewCode(erroridl.ErrorCode_STATE_CORRUPT, err.Error(), nil)
	}
	current, loaded, loadFailure := local.LoadBound()
	if loadFailure != nil {
		return nil, loadFailure
	}
	events, eventsFailure := local.Events()
	if eventsFailure != nil {
		return nil, eventsFailure
	}
	pending, pendingErr := state.PendingPanoramaAction(events)
	if pendingErr != nil {
		return nil, errs.NewCode(erroridl.ErrorCode_STATE_CORRUPT, pendingErr.Error(), nil)
	}
	if ProjectionNeedsRebuild(projection, current, pending) && !dryRun {
		if err := rebuildProjection(root, current); err != nil {
			return nil, errs.NewCode(erroridl.ErrorCode_LOCAL_COMMIT_FAILED, err.Error(), nil)
		}
		projection, _, err = LoadProjection(root)
		if err != nil {
			return nil, errs.NewCode(erroridl.ErrorCode_STATE_CORRUPT, err.Error(), nil)
		}
		if ProjectionNeedsRebuild(projection, current, pending) {
			return nil, errs.NewCode(erroridl.ErrorCode_STATE_CORRUPT, "Card projection is stale", nil)
		}
	}
	content, snapshotContent, err := renderContent(request, current, loaded)
	if err != nil {
		return nil, errs.NewCode(erroridl.ErrorCode_INTERNAL, err.Error(), nil)
	}
	response := &cardidl.CardRenderResponse{Format: request.Format, Content: content}
	if dryRun {
		return response, nil
	}

	now := runtime.now()
	snapshotPath, err := nextCardPath(root, now, projection.SourceEventID)
	if err != nil {
		return nil, errs.NewCode(erroridl.ErrorCode_INTERNAL, err.Error(), nil)
	}
	response.SnapshotPath = &snapshotPath
	snapshot, err := json.MarshalIndent(snapshotContent, "", "  ")
	if err != nil {
		return nil, errs.NewCode(erroridl.ErrorCode_INTERNAL, err.Error(), nil)
	}
	if err := writeSnapshot(root, snapshotPath, append(snapshot, '\n')); err != nil {
		return nil, errs.NewCode(erroridl.ErrorCode_LOCAL_COMMIT_FAILED, err.Error(), nil)
	}
	return response, nil
}

func renderContent(request *cardidl.CardRenderRequest, current state.State, loaded workflow.Loaded) (*commonidl.JsonValue, any, error) {
	content := buildCardContent(request.View, current, loaded)
	if request.Format == cardidl.CardFormat_markdown {
		markdown := renderMarkdownContent(content)
		value, err := commonidl.FromAny(markdown)
		return value, markdown, err
	}
	lark := renderLarkCardContent(content)
	value, err := commonidl.FromAny(lark)
	return value, lark, err
}

func IsCurrentPanoramaSnapshot(root, relative string, current state.State, expectedSourceEventID string) (bool, error) {
	actual, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		return false, err
	}
	projection, loaded, err := LoadProjection(root)
	if err != nil {
		return false, err
	}
	if projection.SourceEventID != expectedSourceEventID || !projectionMatchesState(projection.State(), current) {
		return false, nil
	}
	for _, format := range []cardidl.CardFormat{cardidl.CardFormat_markdown, cardidl.CardFormat_lark_json} {
		_, snapshotContent, err := renderContent(&cardidl.CardRenderRequest{View: cardidl.CardView_panorama, Format: format}, current, loaded)
		if err != nil {
			return false, err
		}
		expected, err := json.MarshalIndent(snapshotContent, "", "  ")
		if err != nil {
			return false, err
		}
		if bytes.Equal(actual, append(expected, '\n')) {
			return true, nil
		}
	}
	return false, nil
}

func RefreshProjectionForPendingAction(root string, current state.State) error {
	if _, err := os.Stat(ProjectionPath(root)); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	local, failure := store.New(root)
	if failure != nil {
		return fmt.Errorf("%s", failure.Error())
	}
	events, failure := local.Events()
	if failure != nil {
		return fmt.Errorf("%s", failure.Error())
	}
	pending, err := state.PendingPanoramaAction(events)
	if err != nil {
		return err
	}
	projectionState := current
	if pending != nil {
		projectionState.LastEventID = pending.TriggerEventID
	}
	return WriteProjection(root, projectionState)
}

// ProjectionNeedsRebuild reports that a valid projection does not match the
// committed State, or does not belong to the pending Panorama action.
// Missing and corrupt projections are rejected by LoadProjection and are not rebuilt.
func ProjectionNeedsRebuild(projection Projection, current state.State, pending *state.PendingAction) bool {
	if !projectionMatchesState(projection.State(), current) {
		return true
	}
	return pending != nil && projection.SourceEventID != pending.TriggerEventID
}

func rebuildProjection(root string, current state.State) error {
	return RefreshProjectionForPendingAction(root, current)
}

func projectionMatchesState(projected, current state.State) bool {
	return reflect.DeepEqual(projected.Requirement, current.Requirement) &&
		reflect.DeepEqual(projected.Release, current.Release) &&
		reflect.DeepEqual(projected.CurrentStepID, current.CurrentStepID) &&
		projected.CurrentStepStatus == current.CurrentStepStatus &&
		projected.CurrentStepSummary == current.CurrentStepSummary &&
		slices.Equal(projected.CurrentEvidence, current.CurrentEvidence) &&
		reflect.DeepEqual(projected.Outputs, current.Outputs) &&
		slices.Equal(projected.SkippedStepIDs, current.SkippedStepIDs) &&
		projectionTraceMatchesState(projected.Integrations.Trace, current.Integrations.Trace)
}

func projectionTraceMatchesState(projected, current *state.TraceBinding) bool {
	if projected == nil || current == nil {
		return projected == nil && current == nil
	}
	return projected.DocumentURL == current.DocumentURL && projected.CLILogDocumentURL == current.CLILogDocumentURL
}

func renderMarkdown(view cardidl.CardView, current state.State, loaded workflow.Loaded) string {
	return renderMarkdownContent(buildCardContent(view, current, loaded))
}

func renderMarkdownContent(content cardContent) string {
	lines := []string{
		fmt.Sprintf("# %s `%s` `%d%%`", markdownTableCell(content.Title), content.Status, content.Progress),
		"",
		content.StageName + " · " + content.StepName,
		"",
		"## 状态全景",
		"",
		content.Panorama,
		"",
		"## 各阶段 Output",
		"",
	}
	headings := make([]string, 0, len(content.Stages))
	separators := make([]string, 0, len(content.Stages))
	outputs := make([]string, 0, len(content.Stages))
	for _, stage := range content.Stages {
		headings = append(headings, stage.Name)
		separators = append(separators, "---")
		outputs = append(outputs, stage.Output)
	}
	if len(headings) > 0 {
		lines = append(lines, markdownTableRow(headings...), markdownTableRow(separators...), markdownTableRow(outputs...))
	}
	lines = append(lines, "", content.Evidence, "", actionMarkdown(content.Action))
	return strings.Join(lines, "\n")
}

func markdownTableRow(cells ...string) string {
	for index := range cells {
		cells[index] = markdownTableCell(cells[index])
	}
	return "| " + strings.Join(cells, " | ") + " |"
}

func markdownTableCell(value string) string {
	return strings.NewReplacer("\r\n", "<br>", "\n", "<br>", "\r", "<br>", "|", "\\|").Replace(strings.TrimSpace(value))
}

func invalidArgument(message string) *erroridl.PublicError {
	return errs.NewCode(erroridl.ErrorCode_INVALID_ARGUMENT, message, nil)
}

func (runtime Runtime) now() time.Time {
	if runtime.Clock == nil {
		return time.Now().UTC()
	}
	return runtime.Clock().UTC()
}

func nextCardPath(root string, now time.Time, sourceEventID string) (string, error) {
	for suffix := 0; ; suffix++ {
		relative := cardsnapshot.Path(now, sourceEventID, suffix)
		_, err := os.Stat(filepath.Join(root, filepath.FromSlash(relative)))
		if os.IsNotExist(err) {
			return relative, nil
		}
		if err != nil {
			return "", err
		}
	}
}

func writeSnapshot(root, relative string, content []byte) error {
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	remove := true
	defer func() {
		if remove {
			_ = os.Remove(path)
		}
	}()
	if _, err := file.Write(content); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	remove = false
	return nil
}
