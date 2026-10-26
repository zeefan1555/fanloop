// Package distribution installs Fanloop CLI releases published on GitHub.
package distribution

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/zeefan1555/fanloop/internal/release"
	"github.com/zeefan1555/fanloop/internal/release/currentlayout"
)

const (
	Repository       = "zeefan1555/fanloop"
	LatestReleaseURL = "https://api.github.com/repos/zeefan1555/fanloop/releases/latest"
	downloadLimit    = 256 << 20
)

// downloadLimitBytes is the maximum response body. Tests lower it.
var downloadLimitBytes = downloadLimit

var versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
var checksumPattern = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)

var platforms = []Platform{
	{OS: "darwin", Arch: "amd64"},
	{OS: "darwin", Arch: "arm64"},
	{OS: "linux", Arch: "amd64"},
	{OS: "linux", Arch: "arm64"},
}

var requiredArchivePaths = []string{
	"bin/fanloop",
	"release.json",
	"entrypoints/fanloop-workflow/SKILL.md",
	"entrypoints/create-fanloop-workflow/SKILL.md",
}

// Platform is a published operating system and CPU architecture.
type Platform struct {
	OS   string
	Arch string
}

// Options selects the release endpoint and the data root to update.
type Options struct {
	DataRoot       string
	Client         *http.Client
	LatestURL      string
	Platform       Platform
	SkillRoots     []string
	BeforeActivate func(string) error
}

// Result is the release that current points at after a successful update.
type Result struct {
	Version             string
	Unchanged           bool
	ConfigurationSynced bool
}

type releaseIndex struct {
	SchemaVersion int     `json:"schema_version"`
	Version       string  `json:"version"`
	Tag           string  `json:"tag"`
	Repository    string  `json:"repository"`
	Assets        []asset `json:"assets"`
}

type asset struct {
	Name   string `json:"name"`
	OS     string `json:"os,omitempty"`
	Arch   string `json:"arch,omitempty"`
	Kind   string `json:"kind,omitempty"`
	SHA256 string `json:"sha256"`
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// SelectPlatform accepts Go and Node architecture names for the four published targets.
func SelectPlatform(goos, goarch string) (Platform, error) {
	arch := goarch
	if arch == "x64" {
		arch = "amd64"
	}
	selected := Platform{OS: goos, Arch: arch}
	for _, candidate := range platforms {
		if candidate == selected {
			return selected, nil
		}
	}
	return Platform{}, fmt.Errorf("unsupported platform %s/%s; fanloop publishes darwin and linux for amd64 and arm64", goos, goarch)
}

// AssetName is the platform archive file name inside a GitHub Release.
func AssetName(version string, platform Platform) (string, error) {
	if !versionPattern.MatchString(version) {
		return "", fmt.Errorf("invalid release version %q", version)
	}
	if _, err := SelectPlatform(platform.OS, platform.Arch); err != nil {
		return "", err
	}
	return fmt.Sprintf("fanloop-%s-%s-%s.tar.gz", version, platform.OS, platform.Arch), nil
}

// VerifyChecksum matches filename against a sha256sum-style checksums.txt.
func VerifyChecksum(checksums, filename string, content []byte) (string, error) {
	expected, err := checksumEntry(checksums, filename)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(content)
	actual := hex.EncodeToString(sum[:])
	if subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) != 1 {
		return "", fmt.Errorf("sha256 mismatch for %s", filename)
	}
	return expected, nil
}

func checksumEntry(checksums, filename string) (string, error) {
	if filename == "" || strings.ContainsAny(filename, `/\`) || filename == "." || filename == ".." {
		return "", fmt.Errorf("invalid checksum filename %q", filename)
	}
	found := ""
	for _, raw := range strings.Split(checksums, "\n") {
		line := strings.TrimSpace(strings.TrimRight(raw, "\r"))
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 || !checksumPattern.MatchString(fields[0]) || strings.ContainsAny(fields[1], `/\`) {
			return "", fmt.Errorf("invalid checksums.txt line: %s", raw)
		}
		name := strings.TrimPrefix(fields[1], "*")
		if name != filename {
			continue
		}
		if found != "" {
			return "", fmt.Errorf("duplicate checksum entry: %s", filename)
		}
		found = strings.ToLower(fields[0])
	}
	if found == "" {
		return "", fmt.Errorf("checksums.txt has no entry for %s", filename)
	}
	return found, nil
}

// SwitchCurrent atomically points dataRoot/current at releases/version.
// It reports whether the link changed and preserves migrated managed directories.
func SwitchCurrent(dataRoot, version string) (bool, error) {
	return currentlayout.Switch(dataRoot, version)
}

func assertCurrentReplaceable(current string) error {
	_, _, err := currentlayout.Inspect(filepath.Dir(current))
	if err != nil {
		return fmt.Errorf("current is not a symlink or valid managed directory: %w", err)
	}
	return err
}

// Update downloads the latest GitHub Release for this platform and switches current.
func Update(ctx context.Context, options Options) (result Result, failure error) {
	if !filepath.IsAbs(options.DataRoot) {
		return Result{}, fmt.Errorf("data root must be absolute")
	}
	if err := assertCurrentReplaceable(filepath.Join(options.DataRoot, "current")); err != nil {
		return Result{}, err
	}
	platform := options.Platform
	if platform == (Platform{}) {
		var err error
		platform, err = SelectPlatform(goos(), goarch())
		if err != nil {
			return Result{}, err
		}
	}
	if _, err := SelectPlatform(platform.OS, platform.Arch); err != nil {
		return Result{}, err
	}
	client := options.Client
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Minute}
	}
	latestURL := options.LatestURL
	if latestURL == "" {
		latestURL = LatestReleaseURL
	}
	body, err := get(ctx, client, latestURL)
	if err != nil {
		return Result{}, err
	}
	var published githubRelease
	if err := json.Unmarshal(body, &published); err != nil {
		return Result{}, fmt.Errorf("decode GitHub release: %w", err)
	}
	version := strings.TrimPrefix(published.TagName, "v")
	if published.TagName != "v"+version || !versionPattern.MatchString(version) {
		return Result{}, fmt.Errorf("invalid GitHub release tag %q", published.TagName)
	}
	installed, installedOK, err := installedReleaseVersion(options.DataRoot)
	if err != nil {
		return Result{}, err
	}
	if installedOK && compareSemver(installed, version) > 0 {
		return Result{Version: installed, Unchanged: true}, nil
	}
	name, err := AssetName(version, platform)
	if err != nil {
		return Result{}, err
	}
	urls := map[string]string{}
	for _, item := range published.Assets {
		urls[item.Name] = item.BrowserDownloadURL
	}
	checksums, err := downloadNamed(ctx, client, urls, "checksums.txt")
	if err != nil {
		return Result{}, err
	}
	manifestBytes, err := downloadNamed(ctx, client, urls, "release.json")
	if err != nil {
		return Result{}, err
	}
	archive, err := downloadNamed(ctx, client, urls, name)
	if err != nil {
		return Result{}, err
	}
	archiveHash, err := VerifyChecksum(string(checksums), name, archive)
	if err != nil {
		return Result{}, err
	}
	if _, err := VerifyChecksum(string(checksums), "release.json", manifestBytes); err != nil {
		return Result{}, err
	}
	var index releaseIndex
	if err := json.Unmarshal(manifestBytes, &index); err != nil {
		return Result{}, fmt.Errorf("decode release.json: %w", err)
	}
	if index.SchemaVersion != 1 || index.Version != version || index.Tag != published.TagName || index.Repository != Repository {
		return Result{}, errors.New("release.json does not match the GitHub release")
	}
	var listed *asset
	for i := range index.Assets {
		if index.Assets[i].Name == name {
			listed = &index.Assets[i]
			break
		}
	}
	if listed == nil || listed.SHA256 != archiveHash || listed.OS != platform.OS || listed.Arch != platform.Arch {
		return Result{}, fmt.Errorf("release.json does not match checksums.txt for %s", name)
	}
	previous, _, _ := currentlayout.Inspect(options.DataRoot)
	backup, err := installArchive(options.DataRoot, version, archive)
	if err != nil {
		return Result{}, err
	}
	defer func() {
		if failure != nil && backup != "" {
			if err := currentlayout.RestoreRelease(options.DataRoot, version, filepath.Base(backup)); err != nil {
				failure = fmt.Errorf("%w; release rollback failed, backup retained: %v", failure, err)
			}
		}
	}()
	roots, err := skillRoots(options)
	if err != nil {
		return Result{}, err
	}
	created, err := ensureSkillLinks(options.DataRoot, roots)
	if err != nil {
		return Result{}, err
	}
	if options.BeforeActivate != nil {
		if err := options.BeforeActivate(filepath.Join(options.DataRoot, "releases", version)); err != nil {
			removeSkillLinks(created)
			return Result{}, err
		}
	}
	changed, err := SwitchCurrent(options.DataRoot, version)
	if err != nil {
		removeSkillLinks(created)
		return Result{}, err
	}
	return Result{Version: version, Unchanged: !changed && filepath.Clean(previous) == filepath.Clean(filepath.Join("releases", version)), ConfigurationSynced: options.BeforeActivate != nil}, nil
}

func skillRoots(options Options) ([]string, error) {
	if options.SkillRoots != nil {
		return options.SkillRoots, nil
	}
	roots, err := release.DefaultSkillRoots()
	if err != nil {
		return nil, err
	}
	return roots.Values(), nil
}

func installedReleaseVersion(dataRoot string) (string, bool, error) {
	target, _, err := currentlayout.Inspect(dataRoot)
	if err == nil && target == "" {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	relative := filepath.Clean(target)
	if filepath.IsAbs(relative) {
		relative, err = filepath.Rel(dataRoot, relative)
		if err != nil {
			return "", false, fmt.Errorf("refusing to replace current link %s", target)
		}
	}
	prefix := "releases" + string(filepath.Separator)
	version := strings.TrimPrefix(relative, prefix)
	if !strings.HasPrefix(relative, prefix) || version == "" || strings.Contains(version, string(filepath.Separator)) || !versionPattern.MatchString(version) {
		return "", false, fmt.Errorf("refusing to replace current link %s", target)
	}
	return version, true, nil
}

func compareSemver(left, right string) int {
	leftParts := strings.Split(left, ".")
	rightParts := strings.Split(right, ".")
	for i := 0; i < 3; i++ {
		lv, _ := strconv.Atoi(leftParts[i])
		rv, _ := strconv.Atoi(rightParts[i])
		if lv != rv {
			return lv - rv
		}
	}
	return 0
}

func ensureSkillLinks(dataRoot string, roots []string) ([]string, error) {
	created := []string{}
	for _, exposed := range release.ExposedSkills() {
		target := filepath.Join(dataRoot, "current", filepath.FromSlash(exposed.Path))
		for _, root := range roots {
			link := filepath.Join(root, exposed.Name)
			info, err := os.Lstat(link)
			if os.IsNotExist(err) {
				if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
					removeSkillLinks(created)
					return nil, err
				}
				if err := os.Symlink(target, link); err != nil {
					removeSkillLinks(created)
					return nil, err
				}
				created = append(created, link)
				continue
			}
			if err != nil {
				removeSkillLinks(created)
				return nil, err
			}
			if info.Mode()&os.ModeSymlink == 0 {
				removeSkillLinks(created)
				return nil, fmt.Errorf("refusing to replace non-Fanloop path %s", link)
			}
			actual, err := os.Readlink(link)
			if err != nil {
				removeSkillLinks(created)
				return nil, err
			}
			if !filepath.IsAbs(actual) {
				actual = filepath.Join(filepath.Dir(link), actual)
			}
			if filepath.Clean(actual) != filepath.Clean(target) {
				removeSkillLinks(created)
				return nil, fmt.Errorf("refusing to replace non-Fanloop Skill link %s", link)
			}
		}
	}
	return created, nil
}

func removeSkillLinks(paths []string) {
	for _, path := range paths {
		_ = os.Remove(path)
	}
}

func installArchive(dataRoot, version string, archive []byte) (string, error) {
	releases := filepath.Join(dataRoot, "releases")
	if err := os.MkdirAll(releases, 0o755); err != nil {
		return "", err
	}
	finalDir := filepath.Join(releases, version)
	staging, err := os.MkdirTemp(releases, ".extract-")
	if err != nil {
		return "", err
	}
	moved := false
	defer func() {
		if !moved {
			_ = os.RemoveAll(staging)
		}
	}()
	if err := extractTarGz(archive, staging); err != nil {
		return "", err
	}
	if err := validateReleaseTree(staging, version); err != nil {
		return "", err
	}
	info, err := os.Lstat(finalDir)
	if err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("refusing to replace immutable release %s", version)
		}
		match, err := treesMatch(finalDir, staging)
		if err != nil {
			return "", err
		}
		if match {
			return "", nil
		}
		backup := filepath.Join(releases, fmt.Sprintf(".backup-%s-", version))
		backup, err = os.MkdirTemp(releases, filepath.Base(backup))
		if err != nil {
			return "", err
		}
		if err := os.Remove(backup); err != nil {
			return "", err
		}
		if err := os.Rename(finalDir, backup); err != nil {
			return "", err
		}
		if err := os.Rename(staging, finalDir); err != nil {
			_ = os.Rename(backup, finalDir)
			return "", err
		}
		moved = true
		return backup, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	if err := os.Rename(staging, finalDir); err != nil {
		return "", err
	}
	moved = true
	return "", nil
}

func extractTarGz(content []byte, destination string) error {
	reader, err := gzip.NewReader(bytes.NewReader(content))
	if err != nil {
		return fmt.Errorf("read release archive: %w", err)
	}
	defer reader.Close()
	archive := tar.NewReader(reader)
	seen := map[string]bool{}
	for {
		header, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read release archive: %w", err)
		}
		name := path.Clean(header.Name)
		if name == "." {
			continue
		}
		if header.Name == "" || path.IsAbs(header.Name) || name == ".." || strings.HasPrefix(name, "../") {
			return fmt.Errorf("archive path %q escapes the release", header.Name)
		}
		target := filepath.Join(destination, filepath.FromSlash(name))
		if !strings.HasPrefix(target, destination+string(os.PathSeparator)) {
			return fmt.Errorf("archive path %q escapes the release", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeXHeader, tar.TypeXGlobalHeader:
			continue
		case tar.TypeDir:
			if metadataArchivePath(name) {
				continue
			}
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if metadataArchivePath(name) {
				continue
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(0o644)
			if header.Mode&0o111 != 0 {
				mode = 0o755
			}
			file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
			if err != nil {
				return err
			}
			if _, err := io.Copy(file, archive); err != nil {
				_ = file.Close()
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}
			seen[name] = true
		default:
			return fmt.Errorf("archive entry %s is not a regular file", header.Name)
		}
	}
	for _, required := range requiredArchivePaths {
		if !seen[required] {
			return fmt.Errorf("archive is missing %s", required)
		}
	}
	return nil
}

func validateReleaseTree(root, version string) error {
	manifest, err := release.Load(root)
	if err != nil {
		return err
	}
	if manifest.ReleaseVersion != version {
		return fmt.Errorf("internal release.json version %s does not match %s", manifest.ReleaseVersion, version)
	}
	digest, err := release.FileDigest(filepath.Join(root, "bin", "fanloop"))
	if err != nil || digest != manifest.Cli.BinarySha256 {
		return fmt.Errorf("binary checksum does not match release.json")
	}
	for _, skill := range manifest.Skills {
		if skill == nil {
			return errors.New("release contains nil skill")
		}
		skillPath, err := release.Resolve(root, skill.Path)
		if err != nil {
			return err
		}
		skillDigest, err := release.DirectoryDigest(skillPath)
		if err != nil || skillDigest != skill.Sha256 {
			return fmt.Errorf("skill %s checksum does not match release.json", skill.Name)
		}
	}
	return nil
}

func treesMatch(left, right string) (bool, error) {
	leftSnapshot, err := snapshotTree(left)
	if err != nil {
		return false, err
	}
	rightSnapshot, err := snapshotTree(right)
	if err != nil {
		return false, err
	}
	return leftSnapshot == rightSnapshot, nil
}

func snapshotTree(root string) (string, error) {
	lines := []string{}
	err := filepath.WalkDir(root, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == root {
			return nil
		}
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			return nil
		}
		if info.Mode().IsRegular() {
			sum, err := fileSHA256(current)
			if err != nil {
				return err
			}
			flag := "-"
			if info.Mode()&0o111 != 0 {
				flag = "x"
			}
			lines = append(lines, relative+"\x00"+sum+"\x00"+flag)
			return nil
		}
		lines = append(lines, relative+"\x00other")
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n"), nil
}

func metadataArchivePath(name string) bool {
	for _, part := range strings.Split(path.Clean(name), "/") {
		if part == "PaxHeader" || part == "PaxHeaders" || strings.HasPrefix(part, "PaxHeaders.") || strings.HasPrefix(part, "._") {
			return true
		}
	}
	return false
}

func fileSHA256(file string) (string, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}

func downloadNamed(ctx context.Context, client *http.Client, urls map[string]string, name string) ([]byte, error) {
	url := urls[name]
	if url == "" {
		return nil, fmt.Errorf("GitHub release is missing %s", name)
	}
	return get(ctx, client, url)
}

func get(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "fanloop")
	request.Header.Set("Accept", "application/vnd.github+json")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("download %s failed: HTTP 404. Fanloop releases require a public repository or authentication; this client does not send a token", url)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s failed: %s", url, response.Status)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, int64(downloadLimitBytes)+1))
	if err != nil {
		return nil, err
	}
	if len(body) > downloadLimitBytes {
		return nil, fmt.Errorf("download %s exceeds %d bytes", url, downloadLimitBytes)
	}
	return body, nil
}

func goos() string   { return goRuntimeOS }
func goarch() string { return goRuntimeArch }

// Replaced in tests that need a fixed platform without changing the host.
var (
	goRuntimeOS   = runtimeGOOS()
	goRuntimeArch = runtimeGOARCH()
)
