package requiremente2e

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type routeEnvelope[T any] struct {
	OK   bool `json:"ok"`
	Data T    `json:"data"`
}

type routeStatusData struct {
	State routeState `json:"state"`
}

type routeState struct {
	Status  string                     `json:"status"`
	Current *routeCurrent              `json:"current"`
	Outputs map[string]registeredValue `json:"outputs"`
}

type routeCurrent struct {
	Context struct {
		StageID   string `json:"stage_id"`
		StageName string `json:"stage_name"`
		JobID     string `json:"job_id"`
		JobName   string `json:"job_name"`
		StepID    string `json:"step_id"`
		StepName  string `json:"step_name"`
		Executor  string `json:"executor"`
	} `json:"context"`
	Prompt struct {
		Content string `json:"content"`
	} `json:"prompt"`
	Conditions      []routeCondition `json:"conditions"`
	AvailableRoutes []struct {
		Direction string         `json:"direction"`
		When      routeWhen      `json:"when"`
		Route     routeSelection `json:"route"`
	} `json:"available_routes"`
}

type routeCondition struct {
	ID     string          `json:"id"`
	Output routeOutputSpec `json:"output"`
}

type routeOutputSpec struct {
	Key      string   `json:"key"`
	Type     string   `json:"type"`
	Source   string   `json:"source"`
	Values   []string `json:"values"`
	Minimum  *int64   `json:"minimum"`
	Maximum  *int64   `json:"maximum"`
	MinItems *int     `json:"min_items"`
	MaxItems *int     `json:"max_items"`
}

type routeWhen struct {
	AnyOf [][]string `json:"any_of"`
}

type routeSelection struct {
	NextStepID *string `json:"next_step_id,omitempty"`
	BackStepID *string `json:"back_step_id,omitempty"`
	Terminal   *bool   `json:"terminal,omitempty"`
}

type routeCase struct {
	ID             string
	Direction      string
	SourceStepID   string
	TargetStepID   string
	ExpectedEffect string
	ConditionIDs   []string
	Route          routeSelection
}

type agentConditionResult struct {
	ConditionID string `json:"condition_id"`
	Output      struct {
		Type  string `json:"type"`
		Value any    `json:"value"`
	} `json:"output"`
}

type agentResultRequest struct {
	StepID           string                 `json:"step_id"`
	ConditionResults []agentConditionResult `json:"condition_results"`
	Evidence         []struct {
		Source  string `json:"source"`
		Content string `json:"content"`
		Ref     string `json:"ref"`
	} `json:"evidence"`
	Summary string         `json:"summary"`
	Route   routeSelection `json:"route"`
}

type agentProgressRequest struct {
	StepID   string `json:"step_id"`
	Status   string `json:"status"`
	Summary  string `json:"summary"`
	Evidence []struct {
		Source  string `json:"source"`
		Content string `json:"content"`
		Ref     string `json:"ref"`
	} `json:"evidence"`
}

type routeResultData struct {
	Effect             string     `json:"effect"`
	EventID            string     `json:"event_id"`
	Transition         transition `json:"transition"`
	State              routeState `json:"state"`
	InvalidatedOutputs []string   `json:"invalidated_outputs"`
}

type transition struct {
	Direction  string `json:"direction"`
	FromStepID string `json:"from_step_id"`
	ToStepID   string `json:"to_step_id"`
}

type registeredValue struct {
	Type           string `json:"type"`
	Value          any    `json:"value"`
	ProducerStepID string `json:"producer_step_id"`
}

type durableRouteState struct {
	LastEventID       string                     `json:"last_event_id"`
	CardSourceEventID string                     `json:"-"`
	Outputs           map[string]registeredValue `json:"outputs"`
}

type routeEvent struct {
	EventID         string `json:"event_id"`
	Kind            string `json:"kind"`
	CausedByEventID string `json:"caused_by_event_id"`
	Payload         struct {
		FlowProgressed *struct {
			FromStepID string `json:"from_step_id"`
		} `json:"flow_progressed"`
		FlowResult *struct {
			Effect           string                 `json:"effect"`
			Transition       transition             `json:"transition"`
			ConditionResults []agentConditionResult `json:"condition_results"`
			OutputChanges    struct {
				Accepted    []string `json:"accepted"`
				Invalidated []string `json:"invalidated"`
			} `json:"output_changes"`
		} `json:"flow_result"`
	} `json:"payload"`
}

var linearLoopConditions = map[string][][]string{
	"frame_requirement_background": {{"background_changed"}},
	"analyze_core_problem":         {{"background_changed"}},
	"define_design_objectives":     {{"problem_changed"}},
	"confirm_technical_problem":    {{"problem_document_published", "panorama_card_published", "objectives_changed"}},
	"research_solution_options":    {{"objectives_changed"}},
	"design_overall_solution":      {{"research_changed"}},
	"design_key_solutions":         {{"overall_solution_changed"}},
	"confirm_solution_direction":   {{"solution_document_published", "panorama_card_published", "key_solutions_changed"}},
	"evaluate_solution_benefits":   {{"key_solutions_changed"}},
	"plan_solution_delivery":       {{"benefits_changed"}},
	"write_technical_solution":     {{"delivery_changed"}},
	"review_technical_solution":    {{"technical_solution_review_written", "presentation_changed"}},
	"confirm_technical_solution":   {{"technical_solution_document_published", "panorama_card_published", "presentation_changed"}},
}

var linearFlowConditions = map[string][][]string{}

type workflowDemoPaths struct {
	RunRoot         string
	RequirementRoot string
	RemoteRoot      string
}

func TestRequirementWorkflowE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("run through ./tests/run-e2e")
	}
	binary := os.Getenv("FANLOOP_E2E_BINARY")
	if binary == "" {
		t.Fatal("FANLOOP_E2E_BINARY is required; run ./tests/run-e2e")
	}
	info, err := os.Stat(binary)
	if err != nil || info.Mode()&0o111 == 0 {
		t.Fatalf("FANLOOP_E2E_BINARY is not executable: %s: %v", binary, err)
	}
	if _, err := exec.LookPath("lark-cli"); err != nil {
		t.Fatalf("fake lark-cli is not installed by ./tests/run-e2e: %v", err)
	}
	var demoInput *bufio.Scanner
	if os.Getenv("FANLOOP_E2E_INTERACTIVE") == "1" {
		demoTTY, err := os.Open("/dev/tty")
		if err != nil {
			t.Fatalf("打开交互终端 /dev/tty：%v", err)
		}
		t.Cleanup(func() { _ = demoTTY.Close() })
		demoInput = bufio.NewScanner(demoTTY)
		fmt.Println("=== 完整 Requirement 生命周期：逐 Step 验证 Progress、Flow、关键 Loop、Trace、Card 与 Doctor ===")
	}
	runLinearRouteDemo(t, binary, demoInput)
}

func runLinearRouteDemo(t *testing.T, binary string, demoInput *bufio.Scanner) {
	t.Helper()
	paths := newWorkflowDemoPaths(t)
	root := paths.RequirementRoot
	t.Setenv("FAKE_LARK_LOG", filepath.Join(paths.RemoteRoot, "lark-cli.log"))
	t.Setenv("FAKE_TRACE_CONTENT", filepath.Join(paths.RemoteRoot, "trace.md"))
	t.Setenv("FAKE_TRACE_KEY", "docx:TraceE2E")
	routeCLISuccess(t, binary, nil, "flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "e2e-mock Requirement lifecycle")
	routeCLISuccess(t, binary, nil, "trace", "bind", "--root", root, "--document-url", "https://bytedance.larkoffice.com/docx/TraceE2E")
	routeCLISuccess(t, binary, nil, "card", "render", "--root", root, "--view", "panorama", "--format", "lark-json")
	if _, err := os.Stat(filepath.Join(root, ".fanloop", "trace", "config.json")); err != nil {
		t.Fatalf("complete lifecycle did not bind Trace: %v", err)
	}
	if cardSnapshots := routeCardSnapshots(t, root); len(cardSnapshots) == 0 {
		t.Fatal("complete lifecycle did not render an initial Card snapshot")
	}

	stepOrder := map[string]int{}
	progressed := map[string]bool{}
	loopIndexes := map[string]int{}
	flowIndexes := map[string]int{}
	loopAlternatives := map[string]bool{}
	flowAlternatives := map[string]bool{}
	reports := 0
	completed := false
	for reports < 200 {
		status := readRouteStatus(t, binary, root)
		if status.State.Status == "completed" {
			completed = true
			break
		}
		if status.State.Current == nil {
			t.Fatal("running Status omitted current Step")
		}
		current := status.State.Current
		stepID := current.Context.StepID
		_, seen := stepOrder[stepID]
		if !seen {
			stepOrder[stepID] = len(stepOrder)
		}
		cases := discoverRouteCases(current)
		if !seen {
			printLinearStep(t, current, len(stepOrder), cases)
		}
		if !progressed[stepID] {
			reportLinearProgress(t, binary, root, current, demoInput)
			progressed[stepID] = true
		}

		var item routeCase
		phase := "Flow 前进或回流恢复"
		loopCase := false
		switch {
		case loopIndexes[stepID] < len(linearLoopConditions[stepID]):
			conditions := linearLoopConditions[stepID][loopIndexes[stepID]]
			item = findRouteCase(t, cases, "loop", conditions)
			phase = fmt.Sprintf("本 Step Loop %d/%d", loopIndexes[stepID]+1, len(linearLoopConditions[stepID]))
			loopCase = true
		case flowIndexes[stepID] < len(linearFlowConditions[stepID]):
			item = findRouteCase(t, cases, "flow", linearFlowConditions[stepID][flowIndexes[stepID]])
		default:
			item = firstFlowCase(t, cases)
		}

		fmt.Printf("\n[执行阶段] %s\n", phase)
		runRouteCase(t, binary, root, item, stepOrder, demoInput)
		reports++
		if loopCase {
			loopIndexes[stepID]++
			loopAlternatives[item.ID] = true
		} else {
			flowIndexes[stepID]++
			flowAlternatives[item.ID] = true
		}
	}

	if !completed {
		t.Fatal("顺序演示超过 200 次 Report，可能发生循环")
	}
	if len(stepOrder) != len(linearLoopConditions) {
		t.Fatalf("顺序演示访问 %d 个 Step，期望 %d", len(stepOrder), len(linearLoopConditions))
	}
	expectedLoops := 0
	for stepID, alternatives := range linearLoopConditions {
		expectedLoops += len(alternatives)
		if loopIndexes[stepID] != len(alternatives) {
			t.Fatalf("%s 执行 %d/%d 个约定 Loop", stepID, loopIndexes[stepID], len(alternatives))
		}
	}
	if len(loopAlternatives) != expectedLoops {
		t.Fatalf("生命周期执行 %d/%d 个唯一 Loop alternative", len(loopAlternatives), expectedLoops)
	}
	if len(flowAlternatives) != len(linearLoopConditions) {
		t.Fatalf("生命周期执行 %d/%d 个唯一 Flow alternative", len(flowAlternatives), len(linearLoopConditions))
	}
	verifyFinalWorkflowDemo(t, binary, paths, reports, len(progressed))
	writeWorkflowDemoReport(t, paths, len(stepOrder), len(flowAlternatives), len(loopAlternatives), reports)
	fmt.Printf("\n=== 完整 Requirement 生命周期完成 ===\nSteps: %d/%d\nFlow alternatives: %d/%d\nLifecycle Loop alternatives: %d/%d\nReports: %d\nStatus: completed\nRequirement Root: %s\nReport: %s\n", len(stepOrder), len(linearLoopConditions), len(flowAlternatives), len(linearLoopConditions), len(loopAlternatives), expectedLoops, reports, root, filepath.Join(paths.RunRoot, "E2E_REPORT.md"))
}

func newWorkflowDemoPaths(t *testing.T) workflowDemoPaths {
	t.Helper()
	runRoot := os.Getenv("FANLOOP_E2E_RUN_ROOT")
	if runRoot == "" {
		runRoot = t.TempDir()
	} else {
		absolute, err := filepath.Abs(runRoot)
		if err != nil {
			t.Fatal(err)
		}
		runRoot = absolute
	}
	paths := workflowDemoPaths{
		RunRoot:         runRoot,
		RequirementRoot: filepath.Join(runRoot, "requirement"),
		RemoteRoot:      filepath.Join(runRoot, "remote"),
	}
	for _, directory := range []string{paths.RunRoot, paths.RequirementRoot, paths.RemoteRoot} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return paths
}

func verifyFinalWorkflowDemo(t *testing.T, binary string, paths workflowDemoPaths, reports, progresses int) {
	t.Helper()
	var sync routeEnvelope[struct {
		Outcome string `json:"outcome"`
		Targets []struct {
			Status string `json:"status"`
		} `json:"targets"`
	}]
	decodeRouteJSON(t, routeCLISuccess(t, binary, nil, "trace", "sync", "--root", paths.RequirementRoot), &sync)
	if !sync.OK || sync.Data.Outcome != "succeeded" || len(sync.Data.Targets) != 2 {
		t.Fatalf("final Trace sync failed: %#v", sync.Data)
	}
	for _, target := range sync.Data.Targets {
		if target.Status != "succeeded" {
			t.Fatalf("final Trace target = %s", target.Status)
		}
	}
	verifyRouteDiagnostics(t, binary, paths.RequirementRoot)

	for _, relative := range []string{
		".fanloop/flow/state.json",
		".fanloop/output/state.json",
		".fanloop/trace/config.json",
		".fanloop/trace/events.jsonl",
		".fanloop/trace/events.md",
		".fanloop/card/projection.json",
	} {
		if _, err := os.Stat(filepath.Join(paths.RequirementRoot, filepath.FromSlash(relative))); err != nil {
			t.Fatalf("complete lifecycle omitted %s: %v", relative, err)
		}
	}
	remoteTrace, err := os.ReadFile(filepath.Join(paths.RemoteRoot, "trace.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Workflow Trace", "e2e-mock", "advanced", "looped", "completed"} {
		if !bytes.Contains(remoteTrace, []byte(want)) {
			t.Fatalf("mock remote Trace omitted %q", want)
		}
	}
	larkLog, err := os.ReadFile(filepath.Join(paths.RemoteRoot, "lark-cli.log"))
	if err != nil || !bytes.Contains(larkLog, []byte("docs +update")) || !bytes.Contains(larkLog, []byte("base +record-upsert")) {
		t.Fatalf("mock Lark projection was not exercised: %v\n%s", err, larkLog)
	}
	snapshots := routeCardSnapshots(t, paths.RequirementRoot)
	if len(snapshots) < 1+progresses+reports {
		t.Fatalf("Card snapshots = %d, want at least %d", len(snapshots), 1+progresses+reports)
	}
	latest, err := os.ReadFile(snapshots[len(snapshots)-1])
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"100%", "流程已完成", "问题定义", "方案设计", "方案成文"} {
		if !bytes.Contains(latest, []byte(want)) {
			t.Fatalf("final Card omitted %q", want)
		}
	}
	if bytes.Contains(latest, []byte("最近一次 Result")) {
		t.Fatal("final Card contains retired 最近一次 Result section")
	}
}

func writeWorkflowDemoReport(t *testing.T, paths workflowDemoPaths, steps, flows, loops, reports int) {
	t.Helper()
	commit := strings.TrimSpace(os.Getenv("FANLOOP_E2E_SOURCE_COMMIT"))
	if commit == "" {
		output, err := exec.Command("git", "rev-parse", "HEAD").Output()
		if err != nil {
			t.Fatal(err)
		}
		commit = strings.TrimSpace(string(output))
	}
	events := len(readRouteEvents(t, paths.RequirementRoot))
	snapshots := len(routeCardSnapshots(t, paths.RequirementRoot))
	outputs := len(readDurableRouteState(t, paths.RequirementRoot).Outputs)
	report := fmt.Sprintf(`# Complete Requirement Lifecycle E2E

- Status: PASS
- Source commit: %s
- Requirement Root: %s
- Steps: %d/%d
- Flow alternatives: %d/%d
- Lifecycle Loop alternatives: %d/%d
- Result reports: %d
- Durable Events: %d
- Valid Outputs: %d
- Immutable Card snapshots: %d
- Trace bind/render/automatic sync/final sync: PASS
- Card projection and immutable snapshots: PASS
- Output Registry and Loop invalidation: PASS
- Doctor: PASS
- External writes: fake lark-cli only
`, commit, paths.RequirementRoot, steps, len(linearLoopConditions), flows, len(linearLoopConditions), loops, totalLinearLoops(), reports, events, outputs, snapshots)
	if err := os.WriteFile(filepath.Join(paths.RunRoot, "E2E_REPORT.md"), []byte(report), 0o644); err != nil {
		t.Fatal(err)
	}
}

func totalLinearLoops() int {
	total := 0
	for _, alternatives := range linearLoopConditions {
		total += len(alternatives)
	}
	return total
}

func routeCardSnapshots(t *testing.T, root string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(root, ".fanloop", "card", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	snapshots := make([]string, 0, len(matches))
	for _, match := range matches {
		switch filepath.Base(match) {
		case "config.json", "projection.json":
		default:
			snapshots = append(snapshots, match)
		}
	}
	return snapshots
}

func printLinearStep(t *testing.T, current *routeCurrent, index int, cases []routeCase) {
	t.Helper()
	fmt.Printf("\n############################################################\n[Step %02d/%02d] %s / %s\nStep ID: %s\nPrompt: %s\n\n| 执行 | Conditions | 目标 |\n| --- | --- | --- |\n", index, len(linearLoopConditions), current.Context.StageName, current.Context.StepName, current.Context.StepID, strings.TrimSpace(current.Prompt.Content))
	loops, ok := linearLoopConditions[current.Context.StepID]
	if !ok {
		t.Fatalf("顺序演示未定义 %s 的 Loop 场景", current.Context.StepID)
	}
	for _, conditions := range loops {
		printRouteCaseRow(findRouteCase(t, cases, "loop", conditions))
	}
	flows := linearFlowConditions[current.Context.StepID]
	if len(flows) == 0 {
		printRouteCaseRow(firstFlowCase(t, cases))
		return
	}
	for _, conditions := range flows {
		printRouteCaseRow(findRouteCase(t, cases, "flow", conditions))
	}
}

func printRouteCaseRow(item routeCase) {
	fmt.Printf("| %s | %s | %s |\n", item.Direction, strings.Join(item.ConditionIDs, " + "), item.TargetStepID)
}

func findRouteCase(t *testing.T, cases []routeCase, direction string, conditions []string) routeCase {
	t.Helper()
	want := sortedUnique(conditions)
	for _, item := range cases {
		if direction != "" && item.Direction != direction {
			continue
		}
		if reflect.DeepEqual(sortedUnique(item.ConditionIDs), want) {
			return item
		}
	}
	t.Fatalf("Status 未声明 %s Route Conditions %v", direction, conditions)
	return routeCase{}
}

func firstFlowCase(t *testing.T, cases []routeCase) routeCase {
	t.Helper()
	for _, item := range cases {
		if item.Direction == "flow" {
			return item
		}
	}
	t.Fatal("Status 未声明 Flow Route")
	return routeCase{}
}

func discoverRouteCases(current *routeCurrent) []routeCase {
	stepID := current.Context.StepID
	result := []routeCase{}
	indexes := map[string]int{"flow": 0, "loop": 0}
	for _, route := range current.AvailableRoutes {
		indexes[route.Direction]++
		target, effect := "", "advanced"
		switch route.Direction {
		case "flow":
			switch {
			case route.Route.NextStepID != nil:
				target = *route.Route.NextStepID
			case route.Route.Terminal != nil && *route.Route.Terminal:
				target, effect = "Done", "completed"
			default:
				continue
			}
		case "loop":
			if route.Route.BackStepID == nil {
				continue
			}
			target, effect = *route.Route.BackStepID, "looped"
		default:
			continue
		}
		for alternativeIndex, conditions := range route.When.AnyOf {
			result = append(result, routeCase{
				ID:        fmt.Sprintf("%s-%s-r%02d-a%02d", route.Direction, stepID, indexes[route.Direction], alternativeIndex+1),
				Direction: route.Direction, SourceStepID: stepID, TargetStepID: target,
				ExpectedEffect: effect, ConditionIDs: append([]string(nil), conditions...), Route: route.Route,
			})
		}
	}
	return result
}

func reportLinearProgress(t *testing.T, binary, root string, current *routeCurrent, demoInput *bufio.Scanner) {
	t.Helper()
	stepID := current.Context.StepID
	summary := "e2e-mock progress: " + stepID
	request := agentProgressRequest{StepID: stepID, Status: "in_progress", Summary: summary}
	request.Evidence = append(request.Evidence, struct {
		Source  string `json:"source"`
		Content string `json:"content"`
		Ref     string `json:"ref"`
	}{Source: "ai", Content: summary, Ref: "progress:" + stepID})
	requestJSON := mustPrettyJSON(t, request)
	fmt.Printf("\n[模拟 Agent -> CLI Progress 请求]\n%s\n[CLI 命令]\nfanloop flow report progress --root <requirement> --input -\n\n", requestJSON)
	waitForRouteDemo(t, demoInput, "按 Enter 上报本 Step 开始执行；输入 q/quit 后回车退出：")
	raw := routeCLISuccess(t, binary, requestJSON, "flow", "report", "progress", "--root", root, "--input", "-")
	var response routeEnvelope[struct {
		Effect string     `json:"effect"`
		State  routeState `json:"state"`
	}]
	decodeRouteJSON(t, raw, &response)
	if !response.OK || response.Data.Effect != "status_updated" || response.Data.State.Current == nil || response.Data.State.Current.Context.StepID != stepID {
		t.Fatalf("Progress did not update %s: %s", stepID, raw)
	}
	durable := readDurableRouteState(t, root)
	events := readRouteEvents(t, root)
	event, ok := findRouteEvent(events, durable.CardSourceEventID)
	if !ok || event.Kind != "flow_progressed" || event.Payload.FlowProgressed == nil || event.Payload.FlowProgressed.FromStepID != stepID {
		t.Fatalf("Progress Event for %s is missing", stepID)
	}
	verifyRouteDiagnostics(t, binary, root)
	fmt.Printf("[Progress 返回]\nEffect: status_updated\nCurrent Step: %s\nExecution: in_progress\nTrace: PASS\nCard: PASS\nDoctor: PASS\n", stepID)
}

func runRouteCase(t *testing.T, binary, root string, item routeCase, stepOrder map[string]int, demoInput *bufio.Scanner) {
	t.Helper()
	before := readRouteStatus(t, binary, root)
	if before.State.Current == nil || before.State.Current.Context.StepID != item.SourceStepID {
		t.Fatalf("%s root is not at %s", item.ID, item.SourceStepID)
	}
	preState := readDurableRouteState(t, root)
	results, outputKeys := mockAgentResults(t, root, before.State.Current, item)
	summary := "e2e-mock " + item.ID + ": " + strings.Join(item.ConditionIDs, ",")
	request := agentResultRequest{StepID: item.SourceStepID, ConditionResults: results, Summary: summary, Route: item.Route}
	request.Evidence = append(request.Evidence, struct {
		Source  string `json:"source"`
		Content string `json:"content"`
		Ref     string `json:"ref"`
	}{Source: "ai", Content: summary, Ref: item.ID})
	requestJSON := mustPrettyJSON(t, request)

	fmt.Printf("\n=== %s ===\n[当前状态与目标]\nStep: %s\nRoute: %s -> %s\nConditions: %s\n\n[模拟 Agent -> CLI 请求]\n%s\n\n[CLI 命令]\nfanloop flow report result --root <requirement> --input -\n\n", item.ID, item.SourceStepID, item.Direction, item.TargetStepID, strings.Join(item.ConditionIDs, " + "), requestJSON)
	waitForRouteDemo(t, demoInput, "按 Enter 执行以上 CLI 调用；输入 q/quit 后回车退出：")
	raw := routeCLISuccess(t, binary, requestJSON, "flow", "report", "result", "--root", root, "--input", "-")
	if os.Getenv("FANLOOP_E2E_FULL_RESPONSE") == "1" {
		fmt.Printf("[CLI flow report result 完整返回体（推进/回流）]\n%s\n", raw)
	}

	var response routeEnvelope[routeResultData]
	decodeRouteJSON(t, raw, &response)
	if !response.OK {
		t.Fatalf("%s returned ok=false", item.ID)
	}
	after := readRouteStatus(t, binary, root)
	invalidated := verifyRouteCase(t, root, item, response.Data, after, preState, outputKeys, stepOrder)
	verifyRouteDiagnostics(t, binary, root)

	current := "Done"
	if after.State.Current != nil {
		current = after.State.Current.Context.StepID
	}
	fmt.Printf("[推进结果]\nEffect: %s\nTransition: %s --%s--> %s\nCurrent Step: %s\nAccepted Outputs: %s\nInvalidated Outputs: %s\nTrace: PASS\nCard: PASS\nDoctor: PASS\n", response.Data.Effect, item.SourceStepID, item.Direction, item.TargetStepID, current, displayList(outputKeys), displayList(invalidated))
	state := map[string]any{"status": after.State.Status}
	if after.State.Current != nil {
		state["current_step_id"] = after.State.Current.Context.StepID
	}
	compact := map[string]any{
		"ok": response.OK,
		"data": map[string]any{
			"effect":              response.Data.Effect,
			"event_id":            response.Data.EventID,
			"transition":          response.Data.Transition,
			"state":               state,
			"invalidated_outputs": response.Data.InvalidatedOutputs,
		},
	}
	fmt.Printf("\n[CLI 返回体（紧凑视图；设置 FANLOOP_E2E_FULL_RESPONSE=1 查看完整 state）]\n%s", mustPrettyJSON(t, compact))
	waitForRouteDemo(t, demoInput, "按 Enter 确认返回结果并继续下一步；输入 q/quit 后回车退出：")
}

func waitForRouteDemo(t *testing.T, input *bufio.Scanner, prompt string) {
	t.Helper()
	if input == nil {
		return
	}
	fmt.Print(prompt)
	if !input.Scan() {
		if err := input.Err(); err != nil {
			t.Fatalf("读取交互输入：%v", err)
		}
		t.Skip("交互演示在标准输入结束时退出")
	}
	switch strings.ToLower(strings.TrimSpace(input.Text())) {
	case "q", "quit":
		t.Skip("用户退出交互演示")
	}
}

func mockAgentResults(t *testing.T, root string, current *routeCurrent, item routeCase) ([]agentConditionResult, []string) {
	t.Helper()
	conditions := make(map[string]routeCondition, len(current.Conditions))
	for _, condition := range current.Conditions {
		conditions[condition.ID] = condition
	}
	results := make([]agentConditionResult, 0, len(item.ConditionIDs))
	keys := make([]string, 0, len(item.ConditionIDs))
	for _, conditionID := range item.ConditionIDs {
		condition, ok := conditions[conditionID]
		if !ok {
			t.Fatalf("Status omitted Condition %s", conditionID)
		}
		result := agentConditionResult{ConditionID: conditionID}
		result.Output.Type = condition.Output.Type
		result.Output.Value = mockAgentValue(t, root, item.ID, conditionID, condition.Output)
		results = append(results, result)
		keys = append(keys, condition.Output.Key)
	}
	return results, keys
}

func mockAgentValue(t *testing.T, root, scenarioID, conditionID string, spec routeOutputSpec) any {
	t.Helper()
	if spec.Source == "integration.trace.document_url" {
		return "https://bytedance.larkoffice.com/docx/TraceE2E"
	}
	switch spec.Type {
	case "enum_value":
		if len(spec.Values) == 0 {
			t.Fatalf("%s enum_value has no values", conditionID)
		}
		return spec.Values[0]
	case "string":
		return "e2e-agent-assertion:" + conditionID
	case "boolean":
		return true
	case "integer":
		if spec.Minimum != nil {
			return *spec.Minimum
		}
		if spec.Maximum != nil && *spec.Maximum < 0 {
			return *spec.Maximum
		}
		return int64(0)
	case "path":
		relative := filepath.Join("mock", conditionID+".txt")
		path := filepath.Join(root, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("e2e-mock for "+scenarioID+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return filepath.ToSlash(relative)
	case "url":
		digest := sha256.Sum256([]byte(scenarioID + ":" + conditionID))
		return "https://bytedance.larkoffice.com/docx/E2EMock" + hex.EncodeToString(digest[:10])
	case "url_list":
		count := 1
		if spec.MinItems != nil && *spec.MinItems > count {
			count = *spec.MinItems
		}
		if spec.MaxItems != nil && count > *spec.MaxItems {
			t.Fatalf("%s has incompatible url_list bounds", conditionID)
		}
		values := make([]string, count)
		for index := range values {
			values[index] = fmt.Sprintf("https://code.byted.org/e2e-mock/fanloop_cli/merge_requests/%d", 7000+index)
		}
		return values
	case "object":
		return map[string]any{"e2e_mock": true}
	default:
		t.Fatalf("unsupported Output type %q for %s", spec.Type, conditionID)
		return nil
	}
}

func verifyRouteCase(t *testing.T, root string, item routeCase, result routeResultData, after routeStatusData, pre durableRouteState, outputKeys []string, stepOrder map[string]int) []string {
	t.Helper()
	observedTarget := result.Transition.ToStepID
	if observedTarget == "" {
		observedTarget = "Done"
	}
	if result.Effect != item.ExpectedEffect || result.Transition.Direction != item.Direction || result.Transition.FromStepID != item.SourceStepID || observedTarget != item.TargetStepID {
		t.Fatalf("%s transition = %#v, effect=%s", item.ID, result.Transition, result.Effect)
	}
	if result.EventID == "" {
		t.Fatalf("%s omitted EventID", item.ID)
	}
	if item.ExpectedEffect == "completed" {
		if after.State.Status != "completed" || after.State.Current != nil {
			t.Fatalf("%s did not complete", item.ID)
		}
	} else if after.State.Current == nil || after.State.Current.Context.StepID != item.TargetStepID {
		t.Fatalf("%s current Step mismatch", item.ID)
	}

	durable := readDurableRouteState(t, root)
	if !reflect.DeepEqual(result.State, after.State) || !reflect.DeepEqual(after.State.Outputs, durable.Outputs) {
		t.Fatalf("%s Result, Status, or Output Registry differs", item.ID)
	}
	events := readRouteEvents(t, root)
	tail := events[len(events)-1]
	if durable.LastEventID != tail.EventID {
		t.Fatalf("%s durable State/Event tail differs", item.ID)
	}
	event, ok := findRouteEvent(events, result.EventID)
	if !ok || event.Payload.FlowResult == nil {
		t.Fatalf("%s Event %s is missing", item.ID, result.EventID)
	}
	if durable.CardSourceEventID != result.EventID {
		t.Fatalf("%s Card projection source = %s, want %s", item.ID, durable.CardSourceEventID, result.EventID)
	}
	payload := event.Payload.FlowResult
	if payload.Effect != result.Effect || payload.Transition != result.Transition {
		t.Fatalf("%s Result/Event transition differs", item.ID)
	}
	actualConditions := make([]string, 0, len(payload.ConditionResults))
	for _, condition := range payload.ConditionResults {
		actualConditions = append(actualConditions, condition.ConditionID)
	}
	if !reflect.DeepEqual(actualConditions, item.ConditionIDs) {
		t.Fatalf("%s Event Conditions = %v", item.ID, actualConditions)
	}
	if !reflect.DeepEqual(sortedUnique(payload.OutputChanges.Accepted), sortedUnique(outputKeys)) {
		t.Fatalf("%s accepted Outputs = %v, want %v", item.ID, payload.OutputChanges.Accepted, outputKeys)
	}

	invalidated := sortedUnique(payload.OutputChanges.Invalidated)
	if !reflect.DeepEqual(sortedUnique(result.InvalidatedOutputs), invalidated) {
		t.Fatalf("%s Result/Event invalidated Outputs differ", item.ID)
	}
	if item.Direction == "flow" {
		for _, key := range outputKeys {
			output, ok := durable.Outputs[key]
			if !ok || output.ProducerStepID != item.SourceStepID {
				t.Fatalf("%s Flow Output %s missing or wrong producer", item.ID, key)
			}
		}
		return invalidated
	}

	backIndex, ok := stepOrder[item.TargetStepID]
	if !ok {
		t.Fatalf("%s Loop target %s has no Step order", item.ID, item.TargetStepID)
	}
	expected := append([]string(nil), outputKeys...)
	for key, output := range pre.Outputs {
		producerIndex, exists := stepOrder[output.ProducerStepID]
		if !exists {
			t.Fatalf("%s Output %s has unknown producer %s", item.ID, key, output.ProducerStepID)
		}
		if producerIndex >= backIndex {
			expected = append(expected, key)
		}
	}
	expected = sortedUnique(expected)
	if !reflect.DeepEqual(invalidated, expected) {
		t.Fatalf("%s invalidated = %v, want %v", item.ID, invalidated, expected)
	}
	for _, key := range expected {
		if _, exists := durable.Outputs[key]; exists {
			t.Fatalf("%s invalidated Output %s remains", item.ID, key)
		}
	}
	for key, output := range pre.Outputs {
		if contains(expected, key) {
			continue
		}
		if current, exists := durable.Outputs[key]; !exists || !reflect.DeepEqual(current, output) {
			t.Fatalf("%s changed preserved Output %s", item.ID, key)
		}
	}
	return invalidated
}

func verifyRouteDiagnostics(t *testing.T, binary, root string) {
	t.Helper()
	var trace routeEnvelope[struct {
		EventCount int `json:"event_count"`
	}]
	decodeRouteJSON(t, routeCLISuccess(t, binary, nil, "trace", "render", "--root", root), &trace)
	if !trace.OK || trace.Data.EventCount == 0 {
		t.Fatal("Trace did not rebuild Events")
	}
	var traceStatus routeEnvelope[struct {
		DocumentURL string `json:"document_url"`
		LastSync    *struct {
			Outcome string `json:"outcome"`
			Targets []struct {
				Status string `json:"status"`
			} `json:"targets"`
		} `json:"last_sync"`
	}]
	decodeRouteJSON(t, routeCLISuccess(t, binary, nil, "trace", "status", "--root", root), &traceStatus)
	if !traceStatus.OK || traceStatus.Data.DocumentURL != "https://bytedance.larkoffice.com/docx/TraceE2E" || traceStatus.Data.LastSync == nil || traceStatus.Data.LastSync.Outcome != "succeeded" || len(traceStatus.Data.LastSync.Targets) != 2 {
		t.Fatalf("Trace binding or automatic sync is unhealthy: %#v", traceStatus.Data)
	}
	for _, target := range traceStatus.Data.LastSync.Targets {
		if target.Status != "succeeded" {
			t.Fatalf("Trace target status = %s", target.Status)
		}
	}
	var card routeEnvelope[struct {
		Content      json.RawMessage `json:"content"`
		SnapshotPath string          `json:"snapshot_path"`
	}]
	decodeRouteJSON(t, routeCLISuccess(t, binary, nil, "card", "render", "--root", root, "--view", "panorama", "--format", "lark-json"), &card)
	if !card.OK || card.Data.SnapshotPath == "" || len(card.Data.Content) == 0 || bytes.Contains(card.Data.Content, []byte("最近一次 Result")) {
		t.Fatalf("Card snapshot is invalid: %#v", card.Data)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(card.Data.SnapshotPath))); err != nil {
		t.Fatalf("Card snapshot was not persisted: %v", err)
	}
	var doctor routeEnvelope[struct {
		Status string `json:"status"`
		Checks []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"checks"`
	}]
	decodeRouteJSON(t, routeCLISuccess(t, binary, nil, "doctor", "--root", root), &doctor)
	if !doctor.OK {
		t.Fatal("Doctor returned ok=false")
	}
	for _, check := range doctor.Data.Checks {
		if check.Status == "failed" || check.Status == "warning" && check.ID != "release_manifest" {
			t.Fatalf("Doctor check %s = %s", check.ID, check.Status)
		}
	}
}

func readRouteStatus(t *testing.T, binary, root string) routeStatusData {
	t.Helper()
	var envelope routeEnvelope[routeStatusData]
	decodeRouteJSON(t, routeCLISuccess(t, binary, nil, "flow", "status", "--root", root), &envelope)
	if !envelope.OK {
		t.Fatal("flow status returned ok=false")
	}
	return envelope.Data
}

func routeCLISuccess(t *testing.T, binary string, input []byte, args ...string) []byte {
	t.Helper()
	command := exec.Command(binary, args...)
	command.Env = withoutBotmuxBinding(os.Environ())
	command.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil || stderr.Len() > 0 {
		t.Fatalf("fanloop %s: %v\nstdout=%s\nstderr=%s", strings.Join(args, " "), err, stdout.Bytes(), stderr.Bytes())
	}
	return stdout.Bytes()
}

func withoutBotmuxBinding(environment []string) []string {
	filtered := make([]string, 0, len(environment))
	for _, item := range environment {
		key, _, _ := strings.Cut(item, "=")
		if key != "BOTMUX_CHAT_ID" && key != "BOTMUX_SESSION_ID" {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func readDurableRouteState(t *testing.T, root string) durableRouteState {
	t.Helper()
	var state durableRouteState
	decodeRouteFile(t, filepath.Join(root, ".fanloop", "flow", "state.json"), &state)
	var registry struct {
		Outputs map[string]registeredValue `json:"outputs"`
	}
	decodeRouteFile(t, filepath.Join(root, ".fanloop", "output", "state.json"), &registry)
	state.Outputs = registry.Outputs
	var projection struct {
		Outputs       map[string]registeredValue `json:"outputs"`
		SourceEventID string                     `json:"source_event_id"`
	}
	decodeRouteFile(t, filepath.Join(root, ".fanloop", "card", "projection.json"), &projection)
	if projection.SourceEventID == "" || !reflect.DeepEqual(projection.Outputs, state.Outputs) {
		t.Fatal("Card projection differs from Output Registry")
	}
	state.CardSourceEventID = projection.SourceEventID
	return state
}

func readRouteEvents(t *testing.T, root string) []routeEvent {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, ".fanloop", "trace", "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(bytes.TrimSpace(content), []byte("\n"))
	if len(lines) == 0 || len(lines[0]) == 0 {
		t.Fatal("Event log is empty")
	}
	events := make([]routeEvent, 0, len(lines))
	for _, line := range lines {
		var event routeEvent
		decodeRouteJSON(t, line, &event)
		events = append(events, event)
	}
	return events
}

func findRouteEvent(events []routeEvent, eventID string) (routeEvent, bool) {
	for _, event := range events {
		if event.EventID == eventID {
			return event, true
		}
	}
	return routeEvent{}, false
}

func decodeRouteFile(t *testing.T, path string, target any) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	decodeRouteJSON(t, content, target)
}

func decodeRouteJSON(t *testing.T, content []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(content, target); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, content)
	}
}

func mustPrettyJSON(t *testing.T, value any) []byte {
	t.Helper()
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(content, '\n')
}

func sortedUnique(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func displayList(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}
