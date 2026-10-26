package configuration

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zeefan1555/fanloop/internal/workflow"
)

// SyncResult is the Workflow snapshot current points at after update config.
type SyncResult struct {
	CommitSHA   string
	WorkflowIDs []string
	Unchanged   bool
}

// afterConfigurationLock is invoked while the exclusive configuration lock is held.
var afterConfigurationLock = func() {}

// Sync downloads production Workflow packages when the remote commit changed.
// Concurrent callers share the configuration lock. An unchanged commit skips checkout.
func Sync(ctx context.Context, dataRoot string) (SyncResult, error) {
	if !filepath.IsAbs(dataRoot) {
		return SyncResult{}, fmt.Errorf("configuration data root must be absolute")
	}
	if err := os.MkdirAll(dataRoot, 0o700); err != nil {
		return SyncResult{}, err
	}
	unlock, err := workflow.LockConfiguration(dataRoot, true)
	if err != nil {
		return SyncResult{}, err
	}
	defer unlock()
	afterConfigurationLock()
	ctx = withRemoteSession(ctx)

	sha, err := remoteCommit(ctx, dataRoot)
	if err != nil {
		return SyncResult{}, err
	}
	if manifest, ok := currentManifest(dataRoot); ok && manifest.CommitSHA == sha {
		return resultFrom(manifest, true), nil
	}

	staging, err := os.MkdirTemp(dataRoot, ".configuration-")
	if err != nil {
		return SyncResult{}, err
	}
	defer os.RemoveAll(staging)
	candidate, remote, err := checkout(ctx, staging, Repository, Ref)
	if err != nil {
		return SyncResult{}, fetchFailed(err)
	}
	manifest, err := inspect(filepath.Join(staging, "checkout"), candidate)
	if err == nil {
		manifest.Repository = remote
	}
	if err != nil {
		return SyncResult{}, err
	}
	published, err := publish(dataRoot, filepath.Join(staging, "checkout"), manifest)
	if err != nil {
		return SyncResult{}, err
	}
	if err := Activate(dataRoot, published); err != nil {
		return SyncResult{}, err
	}
	return resultFrom(manifest, false), nil
}

func resultFrom(manifest Manifest, unchanged bool) SyncResult {
	ids := append([]string{}, manifest.WorkflowIDs...)
	sort.Strings(ids)
	return SyncResult{CommitSHA: manifest.CommitSHA, WorkflowIDs: ids, Unchanged: unchanged}
}

func currentManifest(dataRoot string) (Manifest, bool) {
	target, err := os.Readlink(filepath.Join(dataRoot, "config", "current"))
	if err != nil {
		return Manifest{}, false
	}
	manifest, err := Read(target)
	if err != nil {
		return Manifest{}, false
	}
	return manifest, true
}

func remoteCommit(ctx context.Context, dataRoot string) (string, error) {
	var out string
	_, err := runProduction(ctx, Repository, httpsAttemptTimeout, func(ctx context.Context, remote string) error {
		var callErr error
		out, callErr = git(ctx, dataRoot, "ls-remote", remote, Ref)
		return callErr
	})
	if err != nil {
		return "", fetchFailed(err)
	}
	fields := strings.Fields(out)
	if len(fields) < 2 || fields[1] != Ref {
		return "", fmt.Errorf("%w: remote ref %s not found", ErrFetch, Ref)
	}
	sha := fields[0]
	decoded, err := hex.DecodeString(sha)
	if err != nil || len(decoded) != 20 || sha != strings.ToLower(sha) {
		return "", fmt.Errorf("%w: invalid remote commit", ErrFetch)
	}
	return sha, nil
}
