package runtime_test

import (
	"encoding/json"
	"testing"
)

func TestVersionExposesThriftReleaseContract(t *testing.T) {
	binary := buildCLI(t)
	version := run(binary, "version")
	assertSuccess(t, version, "version")
	var response struct {
		Data struct {
			StateSchema struct {
				WriteVersion int32 `json:"write_version"`
			} `json:"state_schema"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(version.stdout), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.StateSchema.WriteVersion != 16 {
		t.Fatalf("version response = %s", version.stdout)
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(version.stdout), &raw); err != nil {
		t.Fatal(err)
	}
	data := raw["data"].(map[string]any)
	if _, found := data["workflows"]; found {
		t.Fatalf("version response still exposes Workflow release identity: %s", version.stdout)
	}
}
