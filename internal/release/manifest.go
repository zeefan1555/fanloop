package release

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/zeefan1555/commonloop/internal/idl/releaseidl"
)

const (
	ArchiveXZDictionarySize = 8 * 1024 * 1024
	ExposedSkillName        = "commonloop-workflow"
	ExposedSkillPath        = "entrypoints/commonloop-workflow"
)

var skillGroupPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

type Manifest releaseidl.ReleaseManifest
type CLIRelease = releaseidl.CLIRelease
type Skill = releaseidl.SkillArtifact
type Workflow = releaseidl.WorkflowArtifact
type Asset = releaseidl.PlatformAsset

func Load(root string) (Manifest, error) {
	content, err := os.ReadFile(filepath.Join(root, "release.json"))
	if err != nil {
		return Manifest{}, err
	}
	return Decode(content)
}

func Decode(content []byte) (Manifest, error) {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var value Manifest
	if err := decoder.Decode(&value); err != nil {
		return Manifest{}, err
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Manifest{}, fmt.Errorf("trailing JSON value")
		}
		return Manifest{}, err
	}
	if err := value.Validate(); err != nil {
		return Manifest{}, err
	}
	return value, nil
}

func (value Manifest) Validate() error {
	generated := releaseidl.ReleaseManifest(value)
	if err := generated.IsValid(); err != nil {
		return fmt.Errorf("invalid release manifest: %w", err)
	}
	if value.Cli.Version != value.ReleaseVersion {
		return fmt.Errorf("CLI version must match release version")
	}
	if value.StateSchema.WriteVersion <= 0 || !contains(value.StateSchema.ReadVersions, value.StateSchema.WriteVersion) {
		return fmt.Errorf("writable state schema must also be readable")
	}
	workflowIDs := map[string]bool{}
	for _, item := range value.Workflows {
		if item == nil {
			return fmt.Errorf("release contains nil workflow")
		}
		if err := item.IsValid(); err != nil {
			return fmt.Errorf("invalid workflow %q: %w", item.Id, err)
		}
		if item.Path != "workflows/"+item.Id || workflowIDs[item.Id] {
			return fmt.Errorf("invalid or duplicate workflow %q", item.Id)
		}
		workflowIDs[item.Id] = true
	}
	names := map[string]bool{}
	skillGroups := map[string]bool{}
	for _, skill := range value.Skills {
		if skill == nil {
			return fmt.Errorf("release contains nil skill")
		}
		if err := skill.IsValid(); err != nil {
			return fmt.Errorf("invalid skill %q: %w", skill.Name, err)
		}
		if !ValidSkillPath(skill.Path, skill.Name) || names[skill.Name] {
			return fmt.Errorf("invalid or duplicate skill %q", skill.Name)
		}
		names[skill.Name] = true
		if skill.Name == ExposedSkillName {
			continue
		}
		group := strings.Split(skill.Path, "/")[1]
		if !workflowIDs[group] {
			return fmt.Errorf("Skill %q uses unknown Workflow group %q", skill.Name, group)
		}
		skillGroups[group] = true
	}
	if !names[ExposedSkillName] {
		return fmt.Errorf("release is missing exposed Skill %q", ExposedSkillName)
	}
	for workflowID := range workflowIDs {
		if !skillGroups[workflowID] {
			return fmt.Errorf("Workflow %q is missing matching skills/%s group", workflowID, workflowID)
		}
	}
	platforms := map[string]bool{}
	for _, asset := range value.Assets {
		if asset == nil {
			return fmt.Errorf("release contains nil asset")
		}
		if err := asset.IsValid(); err != nil {
			return fmt.Errorf("invalid asset %q: %w", asset.File, err)
		}
		key := asset.Os + "/" + asset.Arch
		file := fmt.Sprintf("commonloop-%s-%s-%s.tar.xz", value.ReleaseVersion, asset.Os, asset.Arch)
		if asset.File != file || platforms[key] {
			return fmt.Errorf("invalid or duplicate asset %q", key)
		}
		platforms[key] = true
	}
	for _, key := range []string{"darwin/amd64", "darwin/arm64", "linux/amd64", "linux/arm64"} {
		if !platforms[key] {
			return fmt.Errorf("release is missing asset %q", key)
		}
	}
	return nil
}

func ValidSkillPath(path, name string) bool {
	if name == ExposedSkillName {
		return path == ExposedSkillPath
	}
	parts := strings.Split(path, "/")
	return len(parts) == 3 && parts[0] == "skills" && skillGroupPattern.MatchString(parts[1]) && parts[2] == name
}

func (value Manifest) Asset(osName, arch string) (Asset, bool) {
	for _, asset := range value.Assets {
		if asset != nil && asset.Os == osName && asset.Arch == arch {
			return *asset, true
		}
	}
	return Asset{}, false
}

func (value Manifest) Skill(name string) (Skill, bool) {
	for _, skill := range value.Skills {
		if skill != nil && skill.Name == name {
			return *skill, true
		}
	}
	return Skill{}, false
}

func RootForExecutable(path string) string {
	directory := filepath.Dir(path)
	if filepath.Base(directory) == "bin" {
		return filepath.Dir(directory)
	}
	return directory
}

func FileDigest(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a regular file", path)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func DirectoryDigest(root string) (string, error) {
	paths := []string{}
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type().IsRegular() {
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			paths = append(paths, filepath.ToSlash(relative))
		}
		return nil
	}); err != nil {
		return "", err
	}
	sort.Strings(paths)
	hash := sha256.New()
	for _, relative := range paths {
		digest, err := FileDigest(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			return "", err
		}
		_, _ = hash.Write([]byte(relative))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(strings.TrimPrefix(digest, "sha256:")))
		_, _ = hash.Write([]byte{'\n'})
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func Resolve(root, relative string) (string, error) {
	if !validPath(relative) {
		return "", fmt.Errorf("invalid release path %q", relative)
	}
	return filepath.Join(root, filepath.FromSlash(relative)), nil
}

func validPath(path string) bool {
	if path == "" || filepath.IsAbs(path) || filepath.Clean(filepath.FromSlash(path)) != filepath.FromSlash(path) {
		return false
	}
	return path != "." && path != ".." && !strings.HasPrefix(filepath.FromSlash(path), ".."+string(filepath.Separator))
}

func ValidVersion(value string) bool {
	return (&releaseidl.CLIRelease{Version: value}).IsValid() == nil
}

func contains[T comparable](values []T, target T) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
