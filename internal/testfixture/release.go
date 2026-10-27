package testfixture

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/zeefan1555/fanloop/internal/idl/opsidl"
	"github.com/zeefan1555/fanloop/internal/release"
	"github.com/zeefan1555/fanloop/internal/state"
)

// BuildRelease creates real companion CLI resources for in-process Runtime tests.
func BuildRelease(root, version string) error {
	_, file, _, _ := runtime.Caller(0)
	repo := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0755); err != nil {
		return err
	}
	command := exec.Command("go", "build", "-buildvcs=false", "-ldflags", fmt.Sprintf("-X github.com/zeefan1555/fanloop/internal/buildinfo.CLIVersion=%s -X github.com/zeefan1555/fanloop/internal/buildinfo.ReleaseVersion=%s", version, version), "-o", filepath.Join(root, "bin", "fanloop"), ".")
	command.Dir = repo
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("build real test CLI: %w: %s", err, output)
	}
	if err := os.CopyFS(filepath.Join(root, "entrypoints"), os.DirFS(filepath.Join(repo, "entrypoints"))); err != nil {
		return err
	}
	if err := os.CopyFS(filepath.Join(root, release.AuthoringSkillPath), os.DirFS(filepath.Join(repo, ".agents", "skills", "create-fanloop-workflow"))); err != nil {
		return err
	}
	binary, err := release.FileDigest(filepath.Join(root, "bin", "fanloop"))
	if err != nil {
		return err
	}
	manifest := release.Manifest{SchemaVersion: 4, ReleaseVersion: version, Cli: &release.CLIRelease{Version: version, BinarySha256: binary}, StateSchema: &opsidl.StateSchemaSupport{ReadVersions: []int32{int32(state.CurrentStateSchemaVersion)}, WriteVersion: int32(state.CurrentStateSchemaVersion)}}
	for _, exposed := range release.ExposedSkills() {
		digest, err := release.DirectoryDigest(filepath.Join(root, exposed.Path))
		if err != nil {
			return err
		}
		manifest.Skills = append(manifest.Skills, &release.Skill{Name: exposed.Name, Path: exposed.Path, Version: version, Sha256: digest})
	}
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "release.json"), append(content, '\n'), 0644)
}

func CanonicalTemp() error {
	path, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		return err
	}
	return os.Setenv("TMPDIR", path)
}
