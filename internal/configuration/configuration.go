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
	"strings"

	"github.com/zeefan1555/fanloop/internal/workflow"
)

const Repository = "git@github.com:zeefan1555/second-brain-agent.git"
const Ref = "refs/heads/main"
const Category = "skills/fanloop-workflow"

var ErrFetch = errors.New("configuration fetch failed")

type File struct {
	Path       string `json:"path"`
	SHA256     string `json:"sha256"`
	Executable bool   `json:"executable"`
}
type Manifest struct {
	SchemaVersion int      `json:"schema_version"`
	Repository    string   `json:"repository"`
	Ref           string   `json:"ref"`
	CommitSHA     string   `json:"commit_sha"`
	WorkflowIDs   []string `json:"workflow_ids"`
	Files         []File   `json:"files"`
	SHA256        string   `json:"configuration_manifest_sha256"`
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
	source := filepath.Dir(filepath.Dir(root))
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
	candidate, err := checkout(ctx, staging, Repository, Ref)
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("%w: check Git network connectivity and SSH permissions", ErrFetch)
	}
	root := filepath.Join(staging, "checkout")
	manifest, err := inspect(root, candidate)
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
	got, err := checkout(ctx, staging, source, commit)
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

func checkout(ctx context.Context, staging, repository, ref string) (string, error) {
	root := filepath.Join(staging, "checkout")
	if err := os.Mkdir(root, 0700); err != nil {
		return "", err
	}
	for _, args := range [][]string{{"init", "--quiet"}, {"sparse-checkout", "set", "--no-cone", "/" + Category + "/"}, {"fetch", "--quiet", "--depth=1", "--no-tags", repository, ref}, {"checkout", "--quiet", "--detach", "FETCH_HEAD"}} {
		if _, err := git(ctx, root, args...); err != nil {
			return "", err
		}
	}
	return git(ctx, root, "rev-parse", "HEAD")
}

func git(ctx context.Context, dir string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = dir
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
	if !reflect.DeepEqual(recorded, actual) {
		return Manifest{}, fmt.Errorf("configuration manifest mismatch")
	}
	return recorded, nil
}

func publish(dataRoot, source string, manifest Manifest) (string, error) {
	snapshots := filepath.Join(dataRoot, "config", "snapshots")
	if err := os.MkdirAll(snapshots, 0700); err != nil {
		return "", err
	}
	target := filepath.Join(snapshots, manifest.CommitSHA)
	if info, err := os.Lstat(target); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("invalid snapshot path")
		}
		recorded, err := Read(target)
		if err != nil {
			return "", err
		}
		if !reflect.DeepEqual(recorded, manifest) {
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
	staging := filepath.Join(stagingRoot, manifest.CommitSHA)
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
