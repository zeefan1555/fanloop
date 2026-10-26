package currentlayout

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/sys/unix"
)

var managedLinks = map[string]string{
	"bin": "release/bin", "entrypoints": "release/entrypoints", "release.json": "release/release.json",
}

var versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

// Lock serializes native installation and distribution activation at one data root.
func Lock(dataRoot string) (func(), error) {
	root := filepath.Join(dataRoot, "releases")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	if info, err := os.Lstat(root); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("invalid releases directory")
	}
	file, err := os.OpenFile(filepath.Join(root, ".current.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, err
	}
	return func() { _ = unix.Flock(int(file.Fd()), unix.LOCK_UN); _ = file.Close() }, nil
}

// RestoreRelease atomically restores retained release bytes after a failed update.
func RestoreRelease(dataRoot, version, backup string) error {
	if !filepath.IsAbs(dataRoot) || !versionPattern.MatchString(version) || filepath.Base(backup) != backup || !strings.HasPrefix(backup, ".backup-"+version+"-") {
		return fmt.Errorf("invalid release backup")
	}
	unlock, err := Lock(dataRoot)
	if err != nil {
		return err
	}
	defer unlock()
	root := filepath.Join(dataRoot, "releases")
	for _, dir := range []string{root, filepath.Join(root, version), filepath.Join(root, backup)} {
		info, err := os.Lstat(dir)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("invalid release restore directory")
		}
	}
	return exchange(filepath.Join(root, version), filepath.Join(root, backup))
}

// ValidateManaged checks the native installer's stable-directory layout.
func ValidateManaged(current, releasesRoot string) (string, error) {
	for name, target := range managedLinks {
		actual, err := os.Readlink(filepath.Join(current, name))
		if err != nil || filepath.Clean(actual) != filepath.Clean(target) {
			return "", fmt.Errorf("refusing invalid stable current link %s", filepath.Join(current, name))
		}
	}
	link := filepath.Join(current, "release")
	target, err := os.Readlink(link)
	if err != nil {
		return "", fmt.Errorf("refusing invalid stable current link %s", link)
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(current, target)
	}
	relative, err := filepath.Rel(releasesRoot, filepath.Clean(target))
	if err != nil || relative == "." || relative == ".." || filepath.Dir(relative) != "." {
		return "", fmt.Errorf("refusing invalid stable current link %s", link)
	}
	for _, dir := range []string{releasesRoot, filepath.Join(releasesRoot, relative)} {
		info, err := os.Lstat(dir)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("refusing invalid stable current link %s", link)
		}
	}
	return filepath.Join("releases", relative), nil
}

// Inspect rejects directories whose extra entries would disappear after migration.
func Inspect(dataRoot string) (target string, managed bool, err error) {
	current := filepath.Join(dataRoot, "current")
	info, err := os.Lstat(current)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err = os.Readlink(current)
		return target, false, err
	}
	if !info.IsDir() {
		return "", false, fmt.Errorf("refusing current because it is not a symlink or managed directory")
	}
	target, err = ValidateManaged(current, filepath.Join(dataRoot, "releases"))
	if err != nil {
		return "", false, err
	}
	entries, err := os.ReadDir(current)
	if err != nil {
		return "", false, err
	}
	for _, entry := range entries {
		if _, ok := managedLinks[entry.Name()]; !ok && entry.Name() != "release" {
			return "", false, fmt.Errorf("refusing current directory with unmanaged entry %s", entry.Name())
		}
	}
	return target, true, nil
}

// Switch atomically activates a release and retains a migrated directory beside current.
func Switch(dataRoot, version string) (bool, error) {
	if !filepath.IsAbs(dataRoot) || !versionPattern.MatchString(version) {
		return false, fmt.Errorf("invalid data root or release version")
	}
	for _, dir := range []string{filepath.Join(dataRoot, "releases"), filepath.Join(dataRoot, "releases", version)} {
		info, err := os.Lstat(dir)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return false, fmt.Errorf("release %s is not an installed directory", version)
		}
	}
	target, managed, err := Inspect(dataRoot)
	if err != nil {
		return false, fmt.Errorf("current is not a symlink or valid managed directory: %w", err)
	}
	unlock, err := Lock(dataRoot)
	if err != nil {
		return false, err
	}
	defer unlock()
	target, managed, err = Inspect(dataRoot)
	for _, dir := range []string{filepath.Join(dataRoot, "releases"), filepath.Join(dataRoot, "releases", version)} {
		info, statErr := os.Lstat(dir)
		if statErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return false, fmt.Errorf("release %s disappeared before activation", version)
		}
	}
	if err != nil {
		return false, fmt.Errorf("current is not a symlink or valid managed directory: %w", err)
	}
	releaseName := filepath.Join("releases", version)
	if !managed && filepath.Clean(target) == releaseName {
		return false, nil
	}
	prefix := ".current-"
	if managed {
		prefix = "current.backup-"
	}
	file, err := os.CreateTemp(dataRoot, prefix)
	if err != nil {
		return false, err
	}
	temporary := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(temporary)
		return false, err
	}
	if err := os.Remove(temporary); err != nil {
		return false, err
	}
	if err := os.Symlink(releaseName, temporary); err != nil {
		return false, err
	}
	current := filepath.Join(dataRoot, "current")
	if managed {
		err = exchange(temporary, current)
	} else {
		err = os.Rename(temporary, current)
	}
	if err != nil {
		_ = os.Remove(temporary)
		return false, err
	}
	return true, nil
}
