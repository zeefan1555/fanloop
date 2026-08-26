package regression

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var adrReferencePattern = regexp.MustCompile(`^ADR-[0-9]{4}$`)

type CapabilityMatrix struct {
	SchemaVersion   int          `json:"schema_version"`
	LegacyTestCount int          `json:"legacy_test_count"`
	Capabilities    []Capability `json:"capabilities"`
}

type Capability struct {
	ID            string   `json:"id"`
	Domain        string   `json:"domain"`
	Description   string   `json:"description"`
	LegacyTests   []string `json:"legacy_tests"`
	Disposition   string   `json:"disposition"`
	ContractCases []string `json:"contract_cases,omitempty"`
	GoTests       []string `json:"go_tests,omitempty"`
	ADR           string   `json:"adr,omitempty"`
}

func LoadCapabilityMatrix(path string) (CapabilityMatrix, error) {
	file, err := os.Open(path)
	if err != nil {
		return CapabilityMatrix{}, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var matrix CapabilityMatrix
	if err := decoder.Decode(&matrix); err != nil {
		return CapabilityMatrix{}, fmt.Errorf("decode capability matrix: %w", err)
	}
	if err := ensureJSONEnd(decoder, "capability matrix"); err != nil {
		return CapabilityMatrix{}, err
	}
	return matrix, nil
}

func HistoricalTestIDs(matrix CapabilityMatrix) []string {
	result := make([]string, 0, matrix.LegacyTestCount)
	for _, capability := range matrix.Capabilities {
		result = append(result, capability.LegacyTests...)
	}
	sort.Strings(result)
	return result
}

func ValidateCapabilityMatrix(matrix CapabilityMatrix, actualTests []string) error {
	if matrix.SchemaVersion != 1 {
		return fmt.Errorf("unsupported capability matrix schema version %d", matrix.SchemaVersion)
	}
	if matrix.LegacyTestCount != len(actualTests) {
		return fmt.Errorf(
			"legacy_test_count = %d, discovered %d",
			matrix.LegacyTestCount,
			len(actualTests),
		)
	}
	actual := make(map[string]struct{}, len(actualTests))
	for _, testID := range actualTests {
		if _, exists := actual[testID]; exists {
			return fmt.Errorf("legacy suite contains duplicate test %q", testID)
		}
		actual[testID] = struct{}{}
	}

	capabilityIDs := make(map[string]struct{}, len(matrix.Capabilities))
	mapped := make(map[string]string, len(actualTests))
	for _, capability := range matrix.Capabilities {
		if capability.ID == "" || capability.Domain == "" || capability.Description == "" {
			return fmt.Errorf("capability id, domain, and description are required: %#v", capability)
		}
		if _, exists := capabilityIDs[capability.ID]; exists {
			return fmt.Errorf("capability %q is declared more than once", capability.ID)
		}
		capabilityIDs[capability.ID] = struct{}{}
		if len(capability.LegacyTests) == 0 {
			return fmt.Errorf("capability %q has no legacy tests", capability.ID)
		}
		switch capability.Disposition {
		case "contract":
			if len(capability.ContractCases) == 0 {
				return fmt.Errorf("contract capability %q has no contract cases", capability.ID)
			}
		case "go_unit":
			if len(capability.GoTests) == 0 {
				return fmt.Errorf("go_unit capability %q has no Go tests", capability.ID)
			}
		case "retired":
			if !adrReferencePattern.MatchString(capability.ADR) {
				return fmt.Errorf(
					"retired capability %q must reference ADR-xxxx",
					capability.ID,
				)
			}
		default:
			return fmt.Errorf(
				"capability %q has invalid disposition %q",
				capability.ID,
				capability.Disposition,
			)
		}
		for _, testID := range capability.LegacyTests {
			if previous, exists := mapped[testID]; exists {
				return fmt.Errorf(
					"legacy test %q mapped more than once: %s and %s",
					testID,
					previous,
					capability.ID,
				)
			}
			if _, exists := actual[testID]; !exists {
				return fmt.Errorf("capability %q maps unknown legacy test %q", capability.ID, testID)
			}
			mapped[testID] = capability.ID
		}
	}
	missing := make([]string, 0)
	for testID := range actual {
		if _, exists := mapped[testID]; !exists {
			missing = append(missing, testID)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("legacy tests missing from capability matrix: %v", missing)
	}
	if len(mapped) != matrix.LegacyTestCount {
		return fmt.Errorf(
			"mapped legacy tests = %d, legacy_test_count = %d",
			len(mapped),
			matrix.LegacyTestCount,
		)
	}
	return nil
}

func ValidateCapabilityReferences(matrix CapabilityMatrix, repository string) error {
	for _, capability := range matrix.Capabilities {
		switch capability.Disposition {
		case "contract":
			for _, id := range capability.ContractCases {
				if filepath.Clean(id) != id || strings.HasPrefix(id, "..") {
					return fmt.Errorf("capability %q has invalid contract case %q", capability.ID, id)
				}
				path := filepath.Join(repository, "tests", "contracts", "testdata", filepath.FromSlash(id), "case.json")
				if _, err := os.Stat(path); err != nil {
					return fmt.Errorf("capability %q contract case %q: %w", capability.ID, id, err)
				}
			}
		case "go_unit":
			for _, reference := range capability.GoTests {
				path, testName, ok := strings.Cut(reference, "#")
				if !ok || path == "" || !strings.HasPrefix(testName, "Test") || filepath.Clean(path) != path || strings.HasPrefix(path, "..") {
					return fmt.Errorf("capability %q has invalid Go test reference %q", capability.ID, reference)
				}
				content, err := os.ReadFile(filepath.Join(repository, filepath.FromSlash(path)))
				if err != nil {
					return fmt.Errorf("capability %q Go test %q: %w", capability.ID, reference, err)
				}
				pattern := regexp.MustCompile(`(?m)^func\s+` + regexp.QuoteMeta(testName) + `\s*\(`)
				if !pattern.Match(content) {
					return fmt.Errorf("capability %q Go test %q does not exist", capability.ID, reference)
				}
			}
		case "retired":
			number := strings.TrimPrefix(capability.ADR, "ADR-")
			matches, err := filepath.Glob(filepath.Join(repository, "docs", "adr", number+"-*.md"))
			if err != nil || len(matches) != 1 {
				return fmt.Errorf("capability %q ADR %q does not resolve to one file", capability.ID, capability.ADR)
			}
		}
	}
	return nil
}

func ensureJSONEnd(decoder *json.Decoder, source string) error {
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("decode %s trailer: %w", source, err)
	}
	return fmt.Errorf("%s contains trailing JSON", source)
}
