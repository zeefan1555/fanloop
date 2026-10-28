package release

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zeefan1555/fanloop/internal/release/currentlayout"
)

// RealPath rejects links at every component, including ancestors outside the tree.
func RealPath(path string, directory bool) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return fmt.Errorf("path must be clean and absolute: %s", path)
	}
	for current := path; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		isDirectory := current != path || directory
		if (isDirectory && !info.IsDir()) || (!isDirectory && !info.Mode().IsRegular()) {
			return fmt.Errorf("path is not a real %s: %s", map[bool]string{true: "directory", false: "file"}[isDirectory], current)
		}
		if current == filepath.Dir(current) {
			return nil
		}
	}
}

func MakeDirectory(path string, mode os.FileMode) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return fmt.Errorf("directory must be clean and absolute")
	}
	for ancestor := path; ; ancestor = filepath.Dir(ancestor) {
		if _, err := os.Lstat(ancestor); err == nil {
			if err := RealPath(ancestor, true); err != nil {
				return err
			}
			break
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.MkdirAll(path, mode); err != nil {
		return err
	}
	return RealPath(path, true)
}

// BoundCurrent permits only the two installed current layouts, not arbitrary links.
func BoundCurrent(home string) (string, error) {
	if err := RealPath(home, true); err != nil {
		return "", err
	}
	target, managed, err := currentlayout.Inspect(home)
	if err != nil {
		return "", err
	}
	if target == "" {
		return "", fmt.Errorf("controller current is missing")
	}
	if managed {
		for name, expected := range map[string]string{"bin": "release/bin", "entrypoints": "release/entrypoints", "release.json": "release/release.json"} {
			actual, err := os.Readlink(filepath.Join(home, "current", name))
			if err != nil || actual != expected {
				return "", fmt.Errorf("invalid controller current alias %s", name)
			}
		}
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(home, target)
	}
	target = filepath.Clean(target)
	if filepath.Dir(target) != filepath.Join(home, "releases") || !ValidVersion(filepath.Base(target)) {
		return "", fmt.Errorf("current escapes installed releases")
	}
	primary := filepath.Join(home, "current")
	if managed {
		primary = filepath.Join(primary, "release")
	}
	raw, err := os.Readlink(primary)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(filepath.Dir(primary), target)
	if err != nil || (raw != target && raw != relative) {
		return "", fmt.Errorf("invalid controller current primary link")
	}
	if err := RealPath(target, true); err != nil {
		return "", err
	}
	return target, nil
}

func ControllerResourcesDigest(directoryDigest string, modes map[string]os.FileMode) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte("fanloop-controller-release-resources-v1\n" + strings.TrimPrefix(directoryDigest, "sha256:") + "\n"))
	paths := make([]string, 0, len(modes))
	for path := range modes {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		mode := modes[path]
		identity := "d"
		if !mode.IsDir() {
			identity = fmt.Sprintf("f:%03o", mode.Perm())
		}
		_, _ = hash.Write([]byte(filepath.ToSlash(path) + "\x00" + identity + "\n"))
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func LoadBound(root string) (Manifest, error) {
	if _, err := StrictDirectoryDigest(root); err != nil {
		return Manifest{}, err
	}
	value, err := Load(root)
	if err != nil {
		return Manifest{}, err
	}
	if err := RealPath(filepath.Join(root, "bin"), true); err != nil {
		return Manifest{}, err
	}
	if err := RealPath(filepath.Join(root, "entrypoints"), true); err != nil {
		return Manifest{}, err
	}
	binary := filepath.Join(root, "bin", "fanloop")
	digest, err := FileDigest(binary)
	if err != nil || digest != value.Cli.BinarySha256 {
		return Manifest{}, fmt.Errorf("controller binary checksum mismatch: %v", err)
	}
	info, err := os.Lstat(binary)
	if err != nil || info.Mode().Perm()&0111 == 0 {
		return Manifest{}, fmt.Errorf("controller binary is not executable")
	}
	for _, skill := range value.Skills {
		digest, err := StrictDirectoryDigest(filepath.Join(root, skill.Path))
		if err != nil || digest != skill.Sha256 {
			return Manifest{}, fmt.Errorf("controller Skill %s checksum mismatch: %v", skill.Name, err)
		}
	}
	return value, nil
}
