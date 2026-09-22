package runtime_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTechnicalSolutionWorkflowRejectsAgentApproval(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "technical-solution-design", "--title", "Human approval required"), "flow.init")

	for _, report := range []struct {
		step       string
		conditions []string
		next       string
	}{
		{step: "frame_requirement_background", conditions: []string{conditionResult("background_defined", "path", `".technical-solution/sections/01-business-background.md"`)}, next: "define_goals_and_problems"},
		{step: "define_goals_and_problems", conditions: []string{conditionResult("goals_and_problems_defined", "path", `".technical-solution/sections/02-goals-and-problems.md"`)}, next: "define_business_constraints"},
		{step: "define_business_constraints", conditions: []string{conditionResult("business_constraints_defined", "path", `".technical-solution/sections/03-business-constraints.md"`)}, next: "confirm_technical_problem"},
	} {
		ensureTechnicalStepStarted(t, binary, root, report.step)
		args := []string{"flow", "report", "result", "--root", root, "--step-id", report.step, "--next-step-id", report.next, "--summary", "accepted"}
		for _, condition := range report.conditions {
			args = append(args, "--condition-result", condition)
		}
		result := run(binary, args...)
		assertSuccess(t, result, "flow.report.result")
		assertFlowEffect(t, result.stdout, "advanced", report.next)
	}

	ensureTechnicalStepStarted(t, binary, root, "confirm_technical_problem")
	rejected := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "confirm_technical_problem",
		"--condition-result", conditionResult("agent_approved", "enum_value", `"approved"`),
		"--next-step-id", "research_solution_options", "--summary", "agent tried to approve")
	if rejected.exitCode == 0 || !strings.Contains(rejected.stderr, "agent_approved") {
		t.Fatalf("retired Agent approval was accepted:\nstdout: %s\nstderr: %s", rejected.stdout, rejected.stderr)
	}
}

func TestMaintainerFourStepLifecycleMergesAndUpdatesLocal(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "fanloop-maintainer", "--title", "Four-step delivery"), "flow.init")
	advanceMaintainerToDelivery(t, binary, root)
	completeMaintainerDelivery(t, binary, root, conditionResult("delivery_main_unchanged", "enum_value", "\"unchanged\""))
}

func TestMaintainerAlreadyMergedRecoveryCompletesWithoutUnchangedMain(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "fanloop-maintainer", "--title", "Merged recovery"), "flow.init")
	advanceMaintainerToDelivery(t, binary, root)
	completeMaintainerDelivery(t, binary, root, conditionResult("delivery_pr_already_merged", "enum_value", "\"merged\""))
}

func completeMaintainerDelivery(t *testing.T, binary, root, deliveryState string) {
	t.Helper()

	mergeCommit := "3333333333333333333333333333333333333333"
	args := []string{"flow", "report", "result", "--root", root,
		"--step-id", "merge_and_update_local",
		"--condition-result", deliveryState,
		"--condition-result", conditionResult("merge_request_published", "url_list", `["https://github.com/zeefan1555/fanloop/pull/7"]`),
		"--condition-result", conditionResult("remote_checks_passed", "enum_value", "\"passed\""),
		"--condition-result", conditionResult("review_comment_synced", "string", "\"comment-7\""),
		"--condition-result", conditionResult("code_merged", "string", "\""+mergeCommit+"\""),
		"--condition-result", conditionResult("merge_tree_verified", "enum_value", "\"passed\""),
		"--condition-result", conditionResult("source_repository_updated", "string", "\""+mergeCommit+"\""),
		"--condition-result", conditionResult("local_cli_updated", "string", "\""+mergeCommit+"\""),
		"--condition-result", conditionResult("post_merge_smoke_passed", "enum_value", "\"passed\""),
		"--condition-result", conditionResult("delivery_record_written", "path", "\"delivery-record.md\""),
		"--condition-result", conditionResult("panorama_presented", "path", "\".fanloop/card/handoff.md\""),
		"--terminal", "--summary", "PR merged and local CLI updated"}
	assertError(t, run(binary, args...), 2, "OUTPUT_INVALID")
	completed := run(binary, append(args, "--evidence", verifyManifestEvidence(t, "passed"))...)
	assertSuccess(t, completed, "flow.report.result")
	assertFlowEffect(t, completed.stdout, "completed", "")
}

func TestMaintainerAdvancedMainSurvivesReturnToBuild(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "fanloop-maintainer", "--title", "Advanced main"), "flow.init")
	advanceMaintainerToDelivery(t, binary, root)

	advanced := "4444444444444444444444444444444444444444"
	looped := run(binary, "flow", "report", "result", "--root", root,
		"--step-id", "merge_and_update_local",
		"--condition-result", conditionResult("delivery_main_advanced", "string", "\""+advanced+"\""),
		"--condition-result", conditionResult("panorama_presented", "path", "\".fanloop/card/main-advanced.md\""),
		"--back-step-id", "build_until_verified", "--summary", "main advanced")
	assertSuccess(t, looped, "flow.report.result")
	assertFlowEffect(t, looped.stdout, "looped", "build_until_verified")

	status := run(binary, "flow", "status", "--root", root)
	assertSuccess(t, status, "flow.status")
	for _, want := range []string{`"step_id": "build_until_verified"`, `"delivery_main_head"`, advanced, "git fetch origin"} {
		if !strings.Contains(status.stdout, want) {
			t.Fatalf("advanced-main handoff is missing %q:\n%s", want, status.stdout)
		}
	}
}

func advanceMaintainerToDelivery(t *testing.T, binary, root string) {
	t.Helper()

	advance := func(step, next string, conditions ...string) {
		t.Helper()
		args := []string{"flow", "report", "result", "--root", root, "--step-id", step, "--next-step-id", next, "--summary", step + " complete"}
		for _, condition := range conditions {
			args = append(args, "--condition-result", condition)
		}
		if step == "certify_candidate" {
			args = append(args, "--evidence", verifyManifestEvidence(t, "passed"))
		}
		assertSuccess(t, run(binary, args...), "flow.report.result")
	}
	advance("define_verification_contract", "build_until_verified",
		conditionResult("verification_contract_written", "path", "\"requirements.md\""),
		conditionResult("requirements_approved", "enum_value", "\"approved\""),
		conditionResult("requirements_decision_recorded", "string", "\"decision-requirements\""),
		conditionResult("panorama_presented", "path", "\".fanloop/card/define.md\""))
	base := "1111111111111111111111111111111111111111"
	head := "2222222222222222222222222222222222222222"
	identity := `{"candidate_head":"` + head + `","git_tree":"tree-1","binary_sha256":"sha256:binary","verification_contract_sha256":"sha256:contract","feature_impact_sha256":"sha256:impact"}`
	advance("build_until_verified", "certify_candidate",
		conditionResult("self_validation_passed", "enum_value", "\"passed\""),
		conditionResult("verification_report_written", "path", "\"verification-report.md\""),
		conditionResult("candidate_identity_recorded", "object", identity),
		conditionResult("panorama_presented", "path", "\".fanloop/card/build.md\""))
	certification := `{"certification_base":"` + base + `","certification_head":"` + head + `","git_tree":"tree-1","binary_sha256":"sha256:binary","verification_assets_sha256":"sha256:assets"}`
	advance("certify_candidate", "merge_and_update_local",
		conditionResult("independent_review_passed", "enum_value", "\"passed\""),
		conditionResult("blackbox_verification_passed", "enum_value", "\"passed\""),
		conditionResult("main_agent_acceptance_passed", "enum_value", "\"passed\""),
		conditionResult("main_agent_acceptance_recorded", "string", "\"decision-final-candidate\""),
		conditionResult("certification_report_written", "path", "\"certification-report.md\""),
		conditionResult("certification_identity_frozen", "object", certification),
		conditionResult("panorama_presented", "path", "\".fanloop/card/certify.md\""))
}

func TestPstackVerifyBehaviorRequiresPassedManifest(t *testing.T) {
	binary, root := buildCLI(t), t.TempDir()
	assertSuccess(t, run(binary, "flow", "init", "--root", root, "--workflow", "pstack-native", "--title", "Verify manifest"), "flow.init")

	advance := func(step, next string, conditions ...string) {
		t.Helper()
		args := []string{"flow", "report", "result", "--root", root, "--step-id", step, "--next-step-id", next, "--summary", step + " complete"}
		for _, condition := range conditions {
			args = append(args, "--condition-result", condition)
		}
		result := run(binary, args...)
		assertSuccess(t, result, "flow.report.result")
		assertFlowEffect(t, result.stdout, "advanced", next)
	}
	advance("select_playbook", "define_acceptance", conditionResult("playbook_selected", "path", `"playbook-selection.md"`))
	advance("define_acceptance", "design_solution", conditionResult("acceptance_defined", "path", `"acceptance.md"`))
	advance("design_solution", "implement_increments", conditionResult("design_ready", "path", `"design-record.md"`))
	advance("implement_increments", "verify_behavior", conditionResult("increments_done", "path", `"execution-report.md"`))

	report := func(evidence ...string) result {
		t.Helper()
		args := []string{"flow", "report", "result", "--root", root, "--step-id", "verify_behavior",
			"--condition-result", conditionResult("delivery_verified", "enum_value", `"passed"`),
			"--next-step-id", "merge_and_release", "--summary", "verification manifest"}
		for _, item := range evidence {
			args = append(args, "--evidence", item)
		}
		return run(binary, args...)
	}
	assertError(t, report(), 2, "OUTPUT_INVALID")
	assertError(t, report(verifyManifestEvidence(t, "failed")), 2, "OUTPUT_INVALID")
	assertError(t, report(verifyManifestEvidenceAt(t, filepath.Join(t.TempDir(), "manifest.json"), "")), 2, "OUTPUT_INVALID")
	assertError(t, report(verifyManifestEvidenceAt(t, filepath.Join(t.TempDir(), "manifest.json"), "{\"schema_version\":0,\"outcome\":\"passed\"}\n")), 2, "OUTPUT_INVALID")
	advanced := report(verifyManifestEvidence(t, "passed"))
	assertSuccess(t, advanced, "flow.report.result")
	assertFlowEffect(t, advanced.stdout, "advanced", "merge_and_release")
}

func verifyManifestEvidence(t *testing.T, outcome string) string {
	t.Helper()
	body, err := json.Marshal(map[string]any{"schema_version": 1, "outcome": outcome})
	if err != nil {
		t.Fatal(err)
	}
	return verifyManifestEvidenceAt(t, filepath.Join(t.TempDir(), "manifest.json"), string(append(body, '\n')))
}

func verifyManifestEvidenceAt(t *testing.T, path, body string) string {
	t.Helper()
	if body != "" {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	evidence, err := json.Marshal(map[string]string{
		"source":  "file",
		"content": "fanloop verify manifest",
		"ref":     path,
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(evidence)
}
