package release

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/idl/opsidl"
	"github.com/zeefan1555/fanloop/internal/idl/releaseidl"
)

func TestDecodeRejectsWorkflowBusinessVersionField(t *testing.T) {
	manifest := validTestManifest()
	content, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(content, &document); err != nil {
		t.Fatal(err)
	}
	document["workflows"] = []any{map[string]any{"id": "technical-solution-design"}}
	content, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(content); err == nil {
		t.Fatal("release.json with retired Workflow artifacts was accepted")
	}
}

func TestDecodeUsesGeneratedChildValidation(t *testing.T) {
	manifest := validTestManifest()
	manifest.Skills[0].Name = "AI-Test"
	content, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(content); err == nil {
		t.Fatal("release.json with a Skill rejected by release.thrift was accepted")
	}
}

func TestValidateAllowsOnlyPackagedWorkflowEntrypoint(t *testing.T) {
	manifest := validTestManifest()
	if err := manifest.Validate(); err != nil {
		t.Fatalf("entrypoint-only release rejected: %v", err)
	}
}

func TestValidateRejectsInvalidGroupedSkillPaths(t *testing.T) {
	tests := []struct{ path, want string }{
		{"skills/ai-test", "invalid or duplicate skill"},
		{"skills/common/ai-test", "invalid or duplicate skill"},
		{"entrypoints/ai-test", "invalid or duplicate skill"},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			manifest := validTestManifest()
			manifest.Skills[0].Name = "ai-test"
			manifest.Skills[0].Path = test.path
			if err := manifest.Validate(); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestDecodeRequiresExposedWorkflowSkill(t *testing.T) {
	manifest := validTestManifest()
	manifest.Skills = nil
	content, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(content); err == nil {
		t.Fatalf("release without %s was accepted", ExposedSkillName)
	}
}

func TestLocalBuildManifestRequiresBinaryChecksum(t *testing.T) {
	manifest := validTestManifest()
	manifest.Cli.BinarySha256 = ""
	if err := manifest.Validate(); err == nil {
		t.Fatal("local build without binary checksum was accepted")
	}
}

func TestLocalBuildManifestRejectsDistributionFields(t *testing.T) {
	content, err := json.Marshal(validTestManifest())
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(content, &document); err != nil {
		t.Fatal(err)
	}
	document["assets"] = []any{}
	content, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(content); err == nil {
		t.Fatal("retired distribution assets were accepted")
	}
}

func TestValidVersionDoesNotRequireBuildMetadata(t *testing.T) {
	for _, value := range []string{"1.2.3", "local-acfd1a5-20260908", "dev"} {
		if !ValidVersion(value) {
			t.Errorf("valid version rejected: %q", value)
		}
	}
	for _, value := range []string{"", "../build", "/tmp/build", "version with spaces"} {
		if ValidVersion(value) {
			t.Errorf("invalid version accepted: %q", value)
		}
	}
}

func validTestManifest() Manifest {
	digest := "sha256:" + strings.Repeat("1", 64)
	return Manifest{
		SchemaVersion:  releaseidl.RELEASE_MANIFEST_SCHEMA_VERSION,
		ReleaseVersion: "1.2.3",
		Cli:            &CLIRelease{Version: "1.2.3", BinarySha256: digest},
		StateSchema:    &opsidl.StateSchemaSupport{ReadVersions: []int32{11}, WriteVersion: 11},
		Skills: []*Skill{
			{Name: ExposedSkillName, Version: "1.2.3", Path: ExposedSkillPath, Sha256: digest},
		},
	}
}
