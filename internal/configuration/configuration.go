package configuration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/zeefan1555/fanloop/internal/workflow"
)

const Repository = "https://github.com/zeefan1555/second-brain-skills.git"
const Ref = "refs/heads/main"
const Category = "fanloop-workflow"
const OriginFornax = "fornax"

var ErrFetch = errors.New("configuration fetch failed")

func fetchFailed() error {
	return fmt.Errorf("%w over HTTPS (%s) and SSH (%s): check network connectivity, HTTPS Git credentials (e.g. gh auth setup-git / credential helper), and SSH key access", ErrFetch, Repository, RepositorySSH)
}

type File struct {
	Path       string `json:"path"`
	SHA256     string `json:"sha256"`
	Executable bool   `json:"executable"`
}

// ReleaseRef is one released Fornax package that was activated into a snapshot.
type ReleaseRef struct {
	Key        string `json:"key"`
	Version    string `json:"version"`
	WorkflowID string `json:"workflow_id"`
}

type Manifest struct {
	SchemaVersion    int          `json:"schema_version"`
	Origin           string       `json:"origin,omitempty"`
	Repository       string       `json:"repository,omitempty"`
	Ref              string       `json:"ref,omitempty"`
	CommitSHA        string       `json:"commit_sha,omitempty"`
	ReleaseSetDigest string       `json:"release_set_digest,omitempty"`
	Releases         []ReleaseRef `json:"releases,omitempty"`
	WorkflowIDs      []string     `json:"workflow_ids"`
	Files            []File       `json:"files"`
	SHA256           string       `json:"configuration_manifest_sha256"`
}

// IsSnapshotSource preserves managed identity even if its manifest is missing.
func IsSnapshotSource(source string) bool {
	source = filepath.Clean(source)
	if filepath.Base(filepath.Dir(source)) == "snapshots" && filepath.Base(filepath.Dir(filepath.Dir(source))) == "config" {
		return true
	}
	_, err := os.Lstat(filepath.Join(source, "configuration.json"))
	return err == nil
}

// PrepareCurrent keeps repeated loaders on one binding for the whole operation.
func PrepareCurrent(_ context.Context, dataRoot string) (func(), error) {
	unlockBinding, err := workflow.LockConfiguration(dataRoot, false)
	if err != nil {
		return nil, err
	}
	root, err := workflow.ConfigurationRoot(dataRoot)
	if err != nil {
		unlockBinding()
		return nil, err
	}
	unlock, err := workflow.LockSource(root, false)
	if err != nil {
		unlockBinding()
		return nil, err
	}
	source := filepath.Dir(root)
	if _, explicit := os.LookupEnv("FANLOOP_CONFIG_SOURCE"); !explicit || IsSnapshotSource(source) {
		_, err = Read(source)
	} else {
		_, err = workflow.ListFrom(root)
	}
	if err != nil {
		unlock()
		unlockBinding()
		return nil, err
	}
	return func() { unlock(); unlockBinding() }, nil
}

// Prepare keeps activation serialized until Init has consumed the selected tree.
func Prepare(ctx context.Context, dataRoot string, dryRun bool) (string, func(), error) {
	if source, explicit := os.LookupEnv("FANLOOP_CONFIG_SOURCE"); explicit {
		root, err := workflow.SourceRoot(source)
		if err != nil {
			return "", nil, err
		}
		unlock, err := workflow.LockSource(root, false)
		if err != nil {
			return "", nil, err
		}
		if IsSnapshotSource(source) {
			_, err = Read(source)
		} else {
			_, err = inspect(source, "")
		}
		if err != nil {
			unlock()
			return "", nil, err
		}
		return root, unlock, nil
	}
	ctx = withRemoteSession(ctx)
	if !filepath.IsAbs(dataRoot) {
		return "", nil, fmt.Errorf("configuration data root must be absolute")
	}
	if err := os.MkdirAll(dataRoot, 0700); err != nil {
		return "", nil, err
	}
	staging, err := os.MkdirTemp(dataRoot, ".configuration-")
	if err != nil {
		return "", nil, err
	}
	unlock := func() {}
	cleanup := func() { _ = os.RemoveAll(staging); unlock() }
	candidate, remote, err := checkout(ctx, staging, Repository, Ref)
	if err != nil {
		cleanup()
		return "", nil, fetchFailed()
	}
	root := filepath.Join(staging, "checkout")
	manifest, err := inspect(root, candidate)
	if err == nil {
		manifest.Repository = remote
	}
	if err != nil {
		cleanup()
		return "", nil, err
	}
	if !dryRun {
		unlock, err = workflow.LockConfiguration(dataRoot, true)
		if err != nil {
			_ = os.RemoveAll(staging)
			return "", nil, err
		}
		root, err = publish(dataRoot, root, manifest)
		if err == nil {
			err = Activate(dataRoot, root)
		}
		if err != nil {
			cleanup()
			return "", nil, err
		}
	}
	return filepath.Join(root, filepath.FromSlash(Category)), cleanup, nil
}

// Import prepares a clean committed source offline. Caller holds the configuration lock.
func Import(ctx context.Context, dataRoot, source string) (string, error) {
	if _, err := workflow.SourceRoot(source); err != nil {
		return "", err
	}
	actual, err := git(ctx, source, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("configuration source must be a Git repository")
	}
	canonical, err := filepath.EvalSymlinks(source)
	if err != nil || actual != canonical {
		return "", fmt.Errorf("configuration source must be the Git repository root")
	}
	dirty, err := git(ctx, source, "status", "--porcelain", "--untracked-files=all")
	if err != nil || dirty != "" {
		return "", fmt.Errorf("configuration source must be a clean Git commit")
	}
	commit, err := git(ctx, source, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	staging, err := os.MkdirTemp(dataRoot, ".configuration-import-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(staging)
	got, _, err := checkout(ctx, staging, source, commit)
	if err != nil {
		return "", fmt.Errorf("read committed configuration: %w", err)
	}
	root := filepath.Join(staging, "checkout")
	manifest, err := inspect(root, got)
	if err != nil {
		return "", err
	}
	return publish(dataRoot, root, manifest)
}

func checkout(ctx context.Context, staging, repository, ref string) (string, string, error) {
	root := filepath.Join(staging, "checkout")
	if err := os.Mkdir(root, 0700); err != nil {
		return "", "", err
	}
	for _, args := range [][]string{{"init", "--quiet"}, {"sparse-checkout", "set", "--no-cone", "/" + Category + "/"}} {
		if _, err := git(ctx, root, args...); err != nil {
			return "", "", err
		}
	}
	remote, err := fetchRemote(ctx, root, repository, ref)
	if err != nil {
		return "", "", err
	}
	if _, err := git(ctx, root, "checkout", "--quiet", "--detach", "FETCH_HEAD"); err != nil {
		return "", "", err
	}
	commit, err := git(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		return "", "", err
	}
	return commit, remote, nil
}

func git(ctx context.Context, dir string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = dir
	// GIT_TERMINAL_PROMPT=0 keeps the HTTPS attempt from waiting on a credential prompt.
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_SSH_COMMAND=ssh -o BatchMode=yes -o ConnectTimeout=15")
	// Git diagnostics may contain credential-bearing URLs from local configuration.
	data, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("git %s failed", args[0])
	}
	return strings.TrimSpace(string(data)), nil
}

func inspect(source, commit string) (Manifest, error) {
	source, err := filepath.EvalSymlinks(source)
	if err != nil {
		return Manifest{}, err
	}
	root, err := workflow.SourceRoot(source)
	if err != nil {
		return Manifest{}, err
	}
	loaded, err := workflow.ListFrom(root)
	if err != nil {
		return Manifest{}, err
	}
	if len(loaded) == 0 {
		return Manifest{}, fmt.Errorf("configuration contains no Workflow packages")
	}
	manifest := Manifest{SchemaVersion: 1, Repository: Repository, Ref: Ref, CommitSHA: commit, WorkflowIDs: []string{}, Files: []File{}}
	for _, item := range loaded {
		manifest.WorkflowIDs = append(manifest.WorkflowIDs, item.Ref.ID)
	}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, failure error) error {
		if failure != nil {
			return failure
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("configuration entry is not a regular file")
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(content)
		manifest.Files = append(manifest.Files, File{Path: filepath.ToSlash(relative), SHA256: hex.EncodeToString(sum[:]), Executable: info.Mode().Perm()&0111 != 0})
		return nil
	})
	if err != nil {
		return Manifest{}, err
	}
	data, _ := json.Marshal(manifest.Files)
	sum := sha256.Sum256(data)
	manifest.SHA256 = "sha256:" + hex.EncodeToString(sum[:])
	return manifest, nil
}

// Read verifies the stored identity against the complete immutable package tree.
func Read(source string) (Manifest, error) {
	info, err := os.Lstat(filepath.Join(source, "configuration.json"))
	if err != nil {
		return Manifest{}, err
	}
	if !info.Mode().IsRegular() {
		return Manifest{}, fmt.Errorf("configuration manifest must be a regular file")
	}
	content, err := os.ReadFile(filepath.Join(source, "configuration.json"))
	if err != nil {
		return Manifest{}, err
	}
	var recorded Manifest
	if err = json.Unmarshal(content, &recorded); err != nil {
		return Manifest{}, err
	}
	if recorded.Origin == OriginFornax {
		return readFornax(source, recorded)
	}
	if decoded, err := hex.DecodeString(recorded.CommitSHA); err != nil || len(decoded) != 20 || recorded.CommitSHA != strings.ToLower(recorded.CommitSHA) {
		return Manifest{}, fmt.Errorf("invalid configuration commit")
	}
	if filepath.Base(filepath.Clean(source)) != recorded.CommitSHA {
		return Manifest{}, fmt.Errorf("configuration commit does not match snapshot directory")
	}
	actual, err := inspect(source, recorded.CommitSHA)
	if err != nil {
		return Manifest{}, err
	}
	if !manifestsMatch(recorded, actual) {
		return Manifest{}, fmt.Errorf("configuration manifest mismatch")
	}
	return recorded, nil
}

func snapshotDirectoryName(manifest Manifest) (string, error) {
	if manifest.Origin == OriginFornax {
		if manifest.ReleaseSetDigest == "" || manifest.ReleaseSetDigest != ReleaseSetDigest(manifest.Releases) {
			return "", fmt.Errorf("invalid Fornax release set digest")
		}
		return manifest.ReleaseSetDigest, nil
	}
	if manifest.CommitSHA == "" {
		return "", fmt.Errorf("invalid configuration commit")
	}
	return manifest.CommitSHA, nil
}

// ReleaseSetDigest is the sha256 hex of the sorted key@version lines.
func ReleaseSetDigest(releases []ReleaseRef) string {
	lines := make([]string, len(releases))
	for index, release := range releases {
		lines[index] = release.Key + "@" + release.Version
	}
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n") + "\n"))
	return hex.EncodeToString(sum[:])
}

// PublishRelease stores an immutable snapshot of a materialized Fornax set.
func PublishRelease(dataRoot, source string, releases []ReleaseRef) (string, error) {
	manifest, err := inspectRelease(source, releases)
	if err != nil {
		return "", err
	}
	return publish(dataRoot, source, manifest)
}

func inspectRelease(source string, releases []ReleaseRef) (Manifest, error) {
	manifest, err := inspect(source, "")
	if err != nil {
		return Manifest{}, err
	}
	copied := append([]ReleaseRef(nil), releases...)
	sort.Slice(copied, func(i, j int) bool { return copied[i].Key < copied[j].Key })
	present := map[string]bool{}
	for _, id := range manifest.WorkflowIDs {
		present[id] = true
	}
	if len(copied) != len(present) {
		return Manifest{}, fmt.Errorf("release set does not match workflow packages")
	}
	for _, release := range copied {
		if release.Key == "" || release.Version == "" || release.WorkflowID == "" || !present[release.WorkflowID] {
			return Manifest{}, fmt.Errorf("release set does not match workflow packages")
		}
	}
	manifest.Origin = OriginFornax
	manifest.Repository = ""
	manifest.Ref = ""
	manifest.CommitSHA = ""
	manifest.Releases = copied
	manifest.ReleaseSetDigest = ReleaseSetDigest(copied)
	return manifest, nil
}

func readFornax(source string, recorded Manifest) (Manifest, error) {
	if recorded.SchemaVersion != 1 || recorded.Repository != "" || recorded.Ref != "" || recorded.CommitSHA != "" {
		return Manifest{}, fmt.Errorf("invalid Fornax configuration snapshot")
	}
	if filepath.Base(filepath.Clean(source)) != recorded.ReleaseSetDigest || recorded.ReleaseSetDigest != ReleaseSetDigest(recorded.Releases) {
		return Manifest{}, fmt.Errorf("configuration release set does not match snapshot directory")
	}
	actual, err := inspectRelease(source, recorded.Releases)
	if err != nil {
		return Manifest{}, err
	}
	if !reflect.DeepEqual(recorded, actual) {
		return Manifest{}, fmt.Errorf("configuration manifest mismatch")
	}
	return recorded, nil
}

func publish(dataRoot, source string, manifest Manifest) (string, error) {
	name, err := snapshotDirectoryName(manifest)
	if err != nil {
		return "", err
	}
	snapshots := filepath.Join(dataRoot, "config", "snapshots")
	if err = os.MkdirAll(snapshots, 0700); err != nil {
		return "", err
	}
	target := filepath.Join(snapshots, name)
	if info, err := os.Lstat(target); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("invalid snapshot path")
		}
		recorded, err := Read(target)
		if err != nil {
			return "", err
		}
		if !manifestsMatch(recorded, manifest) {
			return "", fmt.Errorf("immutable configuration snapshot differs")
		}
		return target, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	stagingRoot, err := os.MkdirTemp(snapshots, ".snapshot-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stagingRoot)
	staging := filepath.Join(stagingRoot, name)
	for _, file := range manifest.Files {
		path := filepath.Join(staging, filepath.FromSlash(file.Path))
		if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return "", err
		}
		content, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(file.Path)))
		if err != nil {
			return "", err
		}
		mode := os.FileMode(0444)
		if file.Executable {
			mode = 0555
		}
		if err = os.WriteFile(path, content, mode); err != nil {
			return "", err
		}
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", err
	}
	if err = os.WriteFile(filepath.Join(staging, "configuration.json"), append(data, '\n'), 0444); err != nil {
		return "", err
	}
	if _, err = Read(staging); err != nil {
		return "", err
	}
	if err = os.Rename(staging, target); err != nil {
		return "", err
	}
	return target, nil
}

// Activate is called under the stable configuration lock after all validation.
func Activate(dataRoot, source string) error {
	current := filepath.Join(dataRoot, "config", "current")
	if info, err := os.Lstat(current); err == nil && info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("configuration current must be a symlink")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(current), 0700); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(filepath.Dir(current), ".activate-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	link := filepath.Join(tmp, "current")
	if err = os.Symlink(source, link); err != nil {
		return err
	}
	return os.Rename(link, current)
}
