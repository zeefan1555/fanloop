package card

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/zeefan1555/fanloop/errs"
	"github.com/zeefan1555/fanloop/internal/idl/cardidl"
	"github.com/zeefan1555/fanloop/internal/idl/commonidl"
	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/idl/flowidl"
	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/store"
	"github.com/zeefan1555/fanloop/internal/workflow"
	"github.com/zeefan1555/fanloop/internal/workflowview"
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
	markdown := renderMarkdown(request.View, current, definition)
	if request.Format == cardidl.CardFormat_markdown {
		value, err := commonidl.FromAny(markdown)
		return value, markdown, err
	}
	lark := renderLarkCard(request.View, current, definition)
	value, err := commonidl.FromAny(lark)
	return value, lark, err
}

func renderMarkdown(view cardidl.CardView, current state.State, definition workflow.Workflow) string {
	lines := []string{"# " + current.Requirement.Title, ""}
	projected := workflowview.Project(definition, current)
	if projected.Current == nil {
		lines = append(lines, "- 当前状态：`workflow completed`")
	} else {
		task := projected.Current
		lines = append(lines,
			fmt.Sprintf("- 当前阶段：`%s` %s", task.Context.StageId, task.Context.StageName),
			fmt.Sprintf("- 当前 Job：`%s` %s", task.Context.JobId, task.Context.JobName),
			fmt.Sprintf("- 当前步骤：`%s` %s", task.Context.StepId, task.Context.StepName),
			"- 执行状态：`"+task.Execution.Status.String()+"`",
			"- 执行方：`"+task.Context.Executor.String()+"`",
			"- "+traceLink(current),
		)
		if link := cliLogLink(current); link != "" {
			lines = append(lines, "- "+link)
		}
		lines = append(lines,
			"- 当前 Prompt："+strings.TrimSpace(task.Prompt.Content),
			"- 可上报 Condition："+conditionSummary(task),
			"- 正常方向："+flowRouteSummary(task),
			"- 回流方向："+loopRouteSummary(task),
		)
	}
	if current.CurrentStepSummary != "" {
		lines = append(lines, "- 摘要："+current.CurrentStepSummary)
	}
	if view == cardidl.CardView_current {
		return strings.Join(lines, "\n")
	}
	lines = append(lines, "", "## Workflow 全景")
	for _, stage := range definition.Stages {
		lines = append(lines, "", "### "+stage.Name)
		for _, job := range stage.Jobs {
			lines = append(lines, "#### "+job.Name)
			for _, step := range job.Steps {
				marker := "-"
				if current.CurrentStepID != nil && *current.CurrentStepID == step.ID {
					marker = "- →"
				}
				lines = append(lines, fmt.Sprintf("%s `%s`：%s", marker, step.ID, step.Name))
			}
		}
	}
	if len(current.Outputs) > 0 {
		keys := make([]string, 0, len(current.Outputs))
		for key := range current.Outputs {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		lines = append(lines, "", "## 当前有效 Output")
		for _, key := range keys {
			output := current.Outputs[key]
			lines = append(lines, fmt.Sprintf("- `%s`（%s，producer=%s）：`%s`", key, output.Type, output.ProducerStepID, output.Value))
		}
	}
	return strings.Join(lines, "\n")
}

func conditionSummary(task interface {
	GetConditions() []*flowidl.ConditionView
}) string {
	conditions := task.GetConditions()
	if len(conditions) == 0 {
		return "无"
	}
	ids := make([]string, 0, len(conditions))
	for _, condition := range conditions {
		ids = append(ids, condition.Id)
	}
	return "`" + strings.Join(ids, "`、`") + "`"
}

func flowRouteSummary(task interface {
	GetAvailableRoutes() []*flowidl.AvailableRoute
}) string {
	targets := make([]string, 0)
	for _, route := range task.GetAvailableRoutes() {
		if route.Direction != flowidl.RouteDirection_flow {
			continue
		}
		if route.Route.GetTerminal() {
			targets = append(targets, "terminal")
		} else {
			targets = append(targets, route.Route.GetNextStepId())
		}
	}
	return quotedTargets(targets)
}

func loopRouteSummary(task interface {
	GetAvailableRoutes() []*flowidl.AvailableRoute
}) string {
	targets := make([]string, 0)
	for _, route := range task.GetAvailableRoutes() {
		if route.Direction == flowidl.RouteDirection_loop {
			targets = append(targets, route.Route.GetBackStepId())
		}
	}
	return quotedTargets(targets)
}

func quotedTargets(values []string) string {
	if len(values) == 0 {
		return "无"
	}
	return "`" + strings.Join(values, "`、`") + "`"
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
