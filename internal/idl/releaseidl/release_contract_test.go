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
		Cli:            &CLIRelease{Version: "1.2.3"},
		StateSchema:    &opsidl.StateSchemaSupport{ReadVersions: []int32{11}, WriteVersion: 11},
		Skills:         []*SkillArtifact{{Name: "ai-test", Version: "1.2.3", Path: "skills/common/ai-test", Sha256: digest}},
		Workflows:      []*WorkflowArtifact{{Id: "technical-solution-design", Path: "workflows/technical-solution-design", Sha256: digest}},
		Assets: []*PlatformAsset{
			{Os: "darwin", Arch: "amd64", File: "fanloop-1.2.3-darwin-amd64.tar.xz", Sha256: digest, BinarySha256: digest},
			{Os: "darwin", Arch: "arm64", File: "fanloop-1.2.3-darwin-arm64.tar.xz", Sha256: digest, BinarySha256: digest},
			{Os: "linux", Arch: "amd64", File: "fanloop-1.2.3-linux-amd64.tar.xz", Sha256: digest, BinarySha256: digest},
			{Os: "linux", Arch: "arm64", File: "fanloop-1.2.3-linux-arm64.tar.xz", Sha256: digest, BinarySha256: digest},
		},
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
	for _, field := range []string{"schema_version", "release_version", "cli", "state_schema", "skills", "workflows", "assets"} {
		if _, ok := document[field]; !ok {
			t.Fatalf("generated JSON is missing %q: %s", field, encoded)
		}
	}

	manifest.SchemaVersion = 1
	if err := manifest.IsValid(); err == nil {
		t.Fatal("expected schema_version other than 2 to fail")
	}
}
