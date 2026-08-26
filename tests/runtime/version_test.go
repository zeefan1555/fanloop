package runtime_test

import (
	"encoding/json"
	"strings"
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
			Workflows []map[string]any `json:"workflows"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(version.stdout), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.StateSchema.WriteVersion != 12 || len(response.Data.Workflows) != 2 {
		t.Fatalf("version response = %s", version.stdout)
	}
	for _, item := range response.Data.Workflows {
		id, idOK := item["id"].(string)
		digest, ok := item["digest"].(string)
		if len(item) != 2 || !idOK || id == "" || !ok || !strings.HasPrefix(digest, "sha256:") {
			t.Fatalf("Workflow release = %#v, want only id and digest", item)
		}
	}
}
