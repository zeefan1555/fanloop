package card

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zeefan1555/fanloop/errs"
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
	if _, failure := store.New(root); failure != nil {
		return nil, failure
	}
	projection, err := LoadProjection(root)
	if err != nil {
		return nil, errs.NewCode(erroridl.ErrorCode_STATE_CORRUPT, err.Error(), nil)
	}
	loaded, err := workflow.LoadRef(projection.Release.Workflow.Ref())
	if err != nil {
		return nil, errs.NewCode(erroridl.ErrorCode_WORKFLOW_MISMATCH, err.Error(), nil)
	}
	current := projection.State()
	content, snapshotContent, err := renderContent(request, current, loaded.Workflow)
	if err != nil {
		return nil, errs.NewCode(erroridl.ErrorCode_INTERNAL, err.Error(), nil)
	}
	response := &cardidl.CardRenderResponse{Format: request.Format, Content: content}
	if dryRun {
		return response, nil
	}

	now := runtime.now()
	snapshotPath, err := nextCardPath(root, now)
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

func renderContent(request *cardidl.CardRenderRequest, current state.State, definition workflow.Workflow) (*commonidl.JsonValue, any, error) {
	content := buildCardContent(request.View, current, definition)
	if request.Format == cardidl.CardFormat_markdown {
		markdown := renderMarkdownContent(content)
		value, err := commonidl.FromAny(markdown)
		return value, markdown, err
	}
	lark := renderLarkCardContent(content)
	value, err := commonidl.FromAny(lark)
	return value, lark, err
}

func renderMarkdown(view cardidl.CardView, current state.State, definition workflow.Workflow) string {
	return renderMarkdownContent(buildCardContent(view, current, definition))
}

func renderMarkdownContent(content cardContent) string {
	lines := []string{
		fmt.Sprintf("# 后端研发交付 · %s `%s` `%d%%`", markdownTableCell(content.Title), content.Status, content.Progress),
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

func nextCardPath(root string, now time.Time) (string, error) {
	stamp := now.In(time.FixedZone("UTC+8", 8*60*60)).Format("20060102T150405.000000-0700")
	for suffix := 0; ; suffix++ {
		name := stamp + ".json"
		if suffix > 0 {
			name = fmt.Sprintf("%s-%d.json", stamp, suffix)
		}
		relative := filepath.ToSlash(filepath.Join(".fanloop", "card", name))
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
