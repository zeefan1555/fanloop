package regression

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestCapabilityMatrixRejectsDuplicateLegacyTest(t *testing.T) {
	matrix := CapabilityMatrix{
		SchemaVersion:   1,
		LegacyTestCount: 1,
		Capabilities: []Capability{
			{
				ID:            "flow.first",
				Domain:        "flow",
				Description:   "first mapping",
				LegacyTests:   []string{"test_flow.FlowTest.test_progress"},
				Disposition:   "contract",
				ContractCases: []string{"flow/progress"},
			},
			{
				ID:            "flow.second",
				Domain:        "flow",
				Description:   "duplicate mapping",
				LegacyTests:   []string{"test_flow.FlowTest.test_progress"},
				Disposition:   "contract",
				ContractCases: []string{"flow/progress-again"},
			},
		},
	}

	err := ValidateCapabilityMatrix(matrix, []string{"test_flow.FlowTest.test_progress"})
	if err == nil || !strings.Contains(err.Error(), "mapped more than once") {
		t.Fatalf("error = %v, want duplicate mapping failure", err)
	}
}

func TestRetiredCapabilityRequiresADRReference(t *testing.T) {
	matrix := CapabilityMatrix{
		SchemaVersion:   1,
		LegacyTestCount: 1,
		Capabilities: []Capability{
			{
				ID:          "flow.retired",
				Domain:      "flow",
				Description: "retired behavior",
				LegacyTests: []string{"test_flow.FlowTest.test_old_alias"},
				Disposition: "retired",
				ADR:         "decision pending",
			},
		},
	}

	err := ValidateCapabilityMatrix(matrix, []string{"test_flow.FlowTest.test_old_alias"})
	if err == nil || !strings.Contains(err.Error(), "ADR-xxxx-full-name") {
		t.Fatalf("error = %v, want explicit ADR reference failure", err)
	}
}

func TestCapabilityReferencesAllowSameNumberADRsWithExactFilename(t *testing.T) {
	root := t.TempDir()
	adrDir := filepath.Join(root, "docs", "adr")
	if err := os.MkdirAll(adrDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"0077-first-decision.md", "0077-second-decision.md"} {
		if err := os.WriteFile(filepath.Join(adrDir, name), []byte("# ADR\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	matrix := CapabilityMatrix{Capabilities: []Capability{{
		ID:          "flow.retired",
		Disposition: "retired",
		ADR:         "ADR-0077-first-decision",
	}}}
	if err := ValidateCapabilityReferences(matrix, root); err != nil {
		t.Fatalf("ValidateCapabilityReferences() error = %v", err)
	}
}

func TestCapabilityMatrixCoversHistoricalSuite(t *testing.T) {
	root := repositoryRoot(t)
	matrix, err := LoadCapabilityMatrix(filepath.Join(root, "tests", "capabilities.json"))
	if err != nil {
		t.Fatal(err)
	}
	actualTests := HistoricalTestIDs(matrix)
	digest := sha256.Sum256([]byte(strings.Join(actualTests, "\n") + "\n"))
	if got, want := hex.EncodeToString(digest[:]), "8e327053c3531bec3f939c61103f5bf42bfebe9d62420fe52af337ca66cbccb9"; got != want {
		t.Fatalf("historical test inventory changed: sha256=%s, want %s", got, want)
	}
	if err := ValidateCapabilityMatrix(matrix, actualTests); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCapabilityReferences(matrix, root); err != nil {
		t.Fatal(err)
	}
	if got, want := len(actualTests), 140; got != want {
		t.Fatalf("legacy tests = %d, want %d", got, want)
	}
}

func TestCurrentWorkflowPreservesCapabilityOwnership(t *testing.T) {
	root := repositoryRoot(t)
	matrix, err := LoadCapabilityMatrix(filepath.Join(root, "tests", "capabilities.json"))
	if err != nil {
		t.Fatal(err)
	}
	entries := make([]string, 0, len(matrix.Capabilities))
	for _, capability := range matrix.Capabilities {
		contractCases := append([]string(nil), capability.ContractCases...)
		goTests := append([]string(nil), capability.GoTests...)
		sort.Strings(contractCases)
		sort.Strings(goTests)
		entries = append(entries, strings.Join([]string{
			capability.ID, capability.Disposition, strings.Join(contractCases, ","), strings.Join(goTests, ","), capability.ADR,
		}, "\t"))
	}
	sort.Strings(entries)
	digest := sha256.Sum256([]byte(strings.Join(entries, "\n") + "\n"))
	if got, want := hex.EncodeToString(digest[:]), "042f889d19d4efbb43d2b7a6cccfb2cfdf51f4101078aad880c2c10daacbb67b"; got != want {
		t.Fatalf("capability ownership changed: sha256=%s, want %s", got, want)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate regression test")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
