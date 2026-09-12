package releaseidl

import (
	"encoding/json"
	"testing"

	"github.com/zeefan1555/fanloop/internal/idl/opsidl"
)

func TestReleaseManifestGeneratedContract(t *testing.T) {
	digest := "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	manifest := &ReleaseManifest{
		SchemaVersion:  RELEASE_MANIFEST_SCHEMA_VERSION,
		ReleaseVersion: "1.2.3",
		Cli:            &CLIRelease{Version: "1.2.3", BinarySha256: digest},
		StateSchema:    &opsidl.StateSchemaSupport{ReadVersions: []int32{11}, WriteVersion: 11},
		Skills:         []*SkillArtifact{{Name: "ai-test", Version: "1.2.3", Path: "skills/technical-solution-design/ai-test", Sha256: digest}},
		Workflows:      []*WorkflowArtifact{{Id: "technical-solution-design", Path: "workflows/technical-solution-design", Sha256: digest}},
	}
	if err := manifest.IsValid(); err != nil {
		t.Fatalf("valid generated manifest rejected: %v", err)
	}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"schema_version", "release_version", "cli", "state_schema", "skills", "workflows"} {
		if _, ok := document[field]; !ok {
			t.Fatalf("generated JSON is missing %q: %s", field, encoded)
		}
	}

	if _, ok := document["assets"]; ok {
		t.Fatal("retired assets field remains in generated JSON")
	}
	if document["cli"].(map[string]any)["binary_sha256"] != digest {
		t.Fatal("generated CLI JSON is missing its binary checksum")
	}

	manifest.SchemaVersion = 2
	if err := manifest.IsValid(); err == nil {
		t.Fatal("expected schema_version other than 3 to fail")
	}
}
