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

	"github.com/zeefan1555/fanloop/internal/idl/releaseidl"
)

const (
	WorkflowSkillName  = "fanloop-workflow"
	WorkflowSkillPath  = "entrypoints/fanloop-workflow"
	AuthoringSkillName = "create-fanloop-workflow"
	AuthoringSkillPath = "entrypoints/create-fanloop-workflow"
)

type ExposedSkill struct {
	Name string
	Path string
}

var exposedSkills = []ExposedSkill{
	{Name: WorkflowSkillName, Path: WorkflowSkillPath},
	{Name: AuthoringSkillName, Path: AuthoringSkillPath},
}

var versionPattern = regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z._+-]*$`)

type Manifest releaseidl.ReleaseManifest
type CLIRelease = releaseidl.CLIRelease
type Skill = releaseidl.SkillArtifact

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
	names := map[string]bool{}
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
	}
	for _, exposed := range ExposedSkills() {
		if !names[exposed.Name] {
			return fmt.Errorf("release is missing exposed Skill %q", exposed.Name)
		}
	}

	return nil
}

func ValidSkillPath(path, name string) bool {
	for _, exposed := range exposedSkills {
		if name == exposed.Name && path == exposed.Path {
			return true
		}
	}
	return false
}

func ExposedSkills() []ExposedSkill {
	return append([]ExposedSkill(nil), exposedSkills...)
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
	return directoryDigest(root, false, nil)
}

func StrictDirectoryDigest(root string) (string, error) {
	return directoryDigest(root, true, nil)
}

func StrictDirectoryIdentity(root string) (string, map[string]os.FileMode, error) {
	modes := map[string]os.FileMode{}
	digest, err := directoryDigest(root, true, modes)
	return digest, modes, err
}

func directoryDigest(root string, strict bool, modes map[string]os.FileMode) (string, error) {
	if strict {
		if err := RealPath(root, true); err != nil {
			return "", err
		}
	}
	paths := []string{}
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if strict && !entry.IsDir() && !entry.Type().IsRegular() {
			return fmt.Errorf("release contains unsupported file %s", path)
		}
		if modes != nil && entry.IsDir() {
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			modes[filepath.ToSlash(relative)] = os.ModeDir
		}
		if entry.Type().IsRegular() {
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			paths = append(paths, filepath.ToSlash(relative))
			if modes != nil {
				info, err := entry.Info()
				if err != nil {
					return err
				}
				modes[filepath.ToSlash(relative)] = info.Mode().Perm()
			}
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
	return versionPattern.MatchString(value)
}

func contains[T comparable](values []T, target T) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
