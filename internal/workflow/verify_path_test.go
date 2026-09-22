package workflow

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

var verifyOutcomeAllowlist = map[string]bool{
	// RemNote content/structure checks are business facts, not Fanloop candidate verification.
	"result_verified":              true,
	"main_agent_acceptance_passed": true,
	"merge_tree_verified":          true,
}

var verifyOutcomeTokens = map[string]bool{
	"verification": true,
	"verified":     true,
	"smoke":        true,
	"blackbox":     true,
}

func TestVerifyOutcomeClaimRule(t *testing.T) {
	cases := []struct {
		id, key string
		typ     OutputType
		values  []string
		want    bool
	}{
		{"result_verified", "verification_result", OutputEnum, []string{"passed"}, false},
		{"delivery_verified", "delivery_verification_result", OutputEnum, []string{"passed"}, true},
		{"post_merge_smoke_passed", "post_merge_smoke_result", OutputEnum, []string{"passed"}, true},
		{"blackbox_verification_failed", "blackbox_verification_result", OutputEnum, []string{"failed"}, true},
		{"main_agent_acceptance_passed", "main_agent_acceptance_result", OutputEnum, []string{"passed"}, false},
		{"merge_tree_verified", "merge_tree_verification", OutputEnum, []string{"passed"}, false},
		{"main_agent_changes_requested", "main_agent_acceptance_result", OutputEnum, []string{"changes_requested"}, false},
		{"self_validation_passed", "self_validation_result", OutputEnum, []string{"passed"}, false},
		{"remote_checks_passed", "remote_checks_result", OutputEnum, []string{"passed"}, false},
		{"card_quality_passed", "card_quality_result", OutputEnum, []string{"passed"}, false},
		{"verification_contract_written", "verification_contract_path", OutputPath, nil, false},
	}
	for _, tc := range cases {
		got := claimsVerifyOutcome(tc.id, ConditionDefinition{Output: OutputDefinition{Key: tc.key, Type: tc.typ, Values: tc.values}})
		if got != tc.want {
			t.Errorf("claimsVerifyOutcome(%s) = %v, want %v", tc.id, got, tc.want)
		}
	}
}

func TestProductionVerifyClaimsUseEvidenceOutcome(t *testing.T) {
	loaded, err := List()
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, bundle := range loaded {
		for id, condition := range productionConditions(bundle.Workflow) {
			if verifyOutcomeAllowlist[id] {
				found[id] = true
				if condition.Output.Source == OutputSourceVerifyEvidenceOutcome {
					t.Errorf("%s/%s is allowlisted and must not use %s", bundle.Ref.ID, id, OutputSourceVerifyEvidenceOutcome)
				}
				continue
			}
			if !claimsVerifyOutcome(id, condition) {
				continue
			}
			if condition.Output.Source != OutputSourceVerifyEvidenceOutcome {
				t.Errorf("%s/%s claims a verify outcome without %s", bundle.Ref.ID, id, OutputSourceVerifyEvidenceOutcome)
			}
		}
	}
	for id := range verifyOutcomeAllowlist {
		if !found[id] {
			t.Errorf("allowlist condition %s is not in a production workflow", id)
		}
	}
}

func TestMaintainerSkillCatalogIsFrozen(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join(sourceRepo(t), "workflows", "fanloop-maintainer", "skills"))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, entry := range entries {
		if !entry.IsDir() {
			t.Fatalf("maintainer skills has non-directory %s", entry.Name())
		}
		got = append(got, entry.Name())
	}
	want := []string{
		"fanloop-dev-agent-acceptance",
		"fanloop-dev-bootstrap",
		"fanloop-dev-code-review",
		"fanloop-dev-decision-receipt",
		"fanloop-dev-domain-modeling",
		"fanloop-dev-grill-with-docs",
		"fanloop-dev-grilling",
		"fanloop-dev-implement",
		"fanloop-dev-maintain-verification",
		"fanloop-dev-merge-and-update-local",
		"fanloop-dev-panorama",
		"fanloop-dev-tdd",
		"fanloop-dev-to-spec",
		"fanloop-dev-to-tickets",
		"fanloop-dev-verify",
		"fanloop-dev-workflow",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("maintainer skill directories = %v, want %v; add a fanloop-dev skill only by deleting another in the same change", got, want)
	}
}

func TestCursorVerifyEntryIsTheOnlyFeatureMap(t *testing.T) {
	repo := sourceRepo(t)
	skills, err := os.ReadDir(filepath.Join(repo, ".cursor", "skills"))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range skills {
		names = append(names, entry.Name())
	}
	if !reflect.DeepEqual(names, []string{"verify-fanloop"}) {
		t.Fatalf(".cursor/skills = %v, want [verify-fanloop]", names)
	}

	link := filepath.Join(repo, ".cursor", "skills", "verify-fanloop", "features")
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal(".cursor/skills/verify-fanloop/features is a real directory")
	}
	target, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	const wantTarget = "../../../workflows/fanloop-maintainer/skills/fanloop-dev-verify/references/features"
	if target != wantTarget {
		t.Fatalf("features symlink = %s, want %s", target, wantTarget)
	}
	resolvedLink, err := filepath.EvalSymlinks(link)
	if err != nil {
		t.Fatal(err)
	}
	resolvedMap, err := filepath.EvalSymlinks(filepath.Join(repo, "workflows", "fanloop-maintainer", "skills", "fanloop-dev-verify", "references", "features"))
	if err != nil {
		t.Fatal(err)
	}
	if resolvedLink != resolvedMap {
		t.Fatalf("features symlink resolves to %s, map is %s", resolvedLink, resolvedMap)
	}

	var maps []string
	err = filepath.WalkDir(repo, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "node_modules") {
			return filepath.SkipDir
		}
		if entry.Name() == "README.md" && filepath.Base(filepath.Dir(path)) == "features" {
			maps = append(maps, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	wantMap := filepath.Join(resolvedMap, "README.md")
	if !reflect.DeepEqual(maps, []string{wantMap}) {
		t.Fatalf("feature map indexes = %v, want [%s]", maps, wantMap)
	}
}

func sourceRepo(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate source repository")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
}

func claimsVerifyOutcome(id string, condition ConditionDefinition) bool {
	if verifyOutcomeAllowlist[id] || condition.Output.Type != OutputEnum {
		return false
	}
	if !intersectsVerifyOutcome(condition.Output.Values) {
		return false
	}
	return hasVerifyOutcomeToken(id) || hasVerifyOutcomeToken(condition.Output.Key)
}

func hasVerifyOutcomeToken(value string) bool {
	for _, part := range strings.FieldsFunc(value, func(r rune) bool { return r == '_' || r == '-' }) {
		if verifyOutcomeTokens[part] {
			return true
		}
	}
	return false
}

func intersectsVerifyOutcome(values []string) bool {
	for _, value := range values {
		switch value {
		case "passed", "failed", "blocked":
			return true
		}
	}
	return false
}

func productionConditions(definition Workflow) map[string]ConditionDefinition {
	conditions := make(map[string]ConditionDefinition, len(definition.Conditions)+len(definition.CommonConditions))
	for id, condition := range definition.Conditions {
		conditions[id] = condition
	}
	for id, condition := range definition.CommonConditions {
		conditions[id] = condition
	}
	return conditions
}
