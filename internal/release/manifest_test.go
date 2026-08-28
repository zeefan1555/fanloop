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
	document["workflows"].([]any)[0].(map[string]any)["version"] = "1.0.0"
	content, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(content); err == nil {
		t.Fatal("release.json with a retired Workflow version field was accepted")
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

func TestValidateAcceptsGroupedSkillPaths(t *testing.T) {
	manifest := validTestManifest()
	manifest.Skills = append(manifest.Skills, &Skill{
		Name: "fanloop-dev-tdd", Version: "1.2.3",
		Path: "skills/fanloop-maintainer/fanloop-dev-tdd", Sha256: manifest.Skills[0].Sha256,
	})
	if err := manifest.Validate(); err != nil {
		t.Fatalf("grouped Skill paths were rejected: %v", err)
	}
}

func TestValidateRejectsInvalidGroupedSkillPaths(t *testing.T) {
	tests := []string{
		"skills/ai-test",
		"skills/common/not-ai-test",
		"skills/fanloop-workflow/common/ai-test",
		"skills/Common/ai-test",
	}
	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			manifest := validTestManifest()
			manifest.Skills[0].Path = path
			if err := manifest.Validate(); err == nil || !strings.Contains(err.Error(), "invalid or duplicate skill") {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestDecodeRequiresExposedWorkflowSkill(t *testing.T) {
	manifest := validTestManifest()
	manifest.Skills = manifest.Skills[:1]
	content, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(content); err == nil || !strings.Contains(err.Error(), ExposedSkillName) {
		t.Fatalf("release without %s = %v", ExposedSkillName, err)
	}
}

func validTestManifest() Manifest {
	digest := "sha256:" + strings.Repeat("1", 64)
	return Manifest{
		SchemaVersion:  releaseidl.RELEASE_MANIFEST_SCHEMA_VERSION,
		ReleaseVersion: "1.2.3",
		Cli:            &CLIRelease{Version: "1.2.3"},
		StateSchema:    &opsidl.StateSchemaSupport{ReadVersions: []int32{11}, WriteVersion: 11},
		Skills: []*Skill{
			{Name: "ai-test", Version: "1.2.3", Path: "skills/common/ai-test", Sha256: digest},
			{Name: ExposedSkillName, Version: "1.2.3", Path: "skills/common/" + ExposedSkillName, Sha256: digest},
		},
		Workflows: []*Workflow{
			{Id: "technical-solution-design", Path: "workflows/technical-solution-design", Sha256: digest},
		},
		Assets: []*Asset{
			{Os: "darwin", Arch: "amd64", File: "fanloop-1.2.3-darwin-amd64.tar.xz", Sha256: digest, BinarySha256: digest},
			{Os: "darwin", Arch: "arm64", File: "fanloop-1.2.3-darwin-arm64.tar.xz", Sha256: digest, BinarySha256: digest},
			{Os: "linux", Arch: "amd64", File: "fanloop-1.2.3-linux-amd64.tar.xz", Sha256: digest, BinarySha256: digest},
			{Os: "linux", Arch: "arm64", File: "fanloop-1.2.3-linux-arm64.tar.xz", Sha256: digest, BinarySha256: digest},
		},
	}
}
