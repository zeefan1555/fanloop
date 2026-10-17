package fornaxmirror

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/zeefan1555/fanloop/internal/configuration"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

const receiptSchema = 1

type Outcome string

const (
	OutcomeUpdated   Outcome = "updated"
	OutcomeUnchanged Outcome = "unchanged"
)

// SyncedWorkflow is one workflow activated from a released package.
type SyncedWorkflow struct {
	ID      string
	Key     string
	Version string
}

// SyncResult is what update prints after a configuration sync.
type SyncResult struct {
	Outcome   Outcome
	Workflows []SyncedWorkflow
	Digest    string
}

type discoveredRelease struct {
	SourceIndex int    `json:"source_index"`
	WorkspaceID string `json:"workspace_id"`
	Key         string `json:"key"`
	Version     string `json:"version"`
	Access      Access `json:"access"`
}

type syncReceipt struct {
	SchemaVersion  int                        `json:"schema_version"`
	Discovered     []discoveredRelease        `json:"discovered"`
	Activated      []configuration.ReleaseRef `json:"activated"`
	SnapshotDigest string                     `json:"snapshot_digest"`
	TreeIdentity   string                     `json:"tree_identity"`
}

var commitReceipt = writeReceipt

var activateSnapshot = configuration.Activate

var warnSkippedPackage = func(key, version string, err error) {
	fmt.Fprintf(os.Stderr, "warning: skipping read-only package %s@%s: %s\n", key, version, err)
}

// Sync installs the latest Online/prod release for every configured source.
// A failed download or validation leaves config/current where it was.
func Sync(ctx context.Context, dataRoot string, client Client) (SyncResult, error) {
	if !filepath.IsAbs(dataRoot) {
		return SyncResult{}, fmt.Errorf("FANLOOP_DATA_HOME must be absolute")
	}
	config, err := LoadConfig(filepath.Join(dataRoot, "fornax.json"))
	if err != nil {
		return SyncResult{}, err
	}
	unlock, err := workflow.LockConfiguration(dataRoot, true)
	if err != nil {
		return SyncResult{}, err
	}
	defer unlock()
	discovered, err := discoverReleases(ctx, config, client)
	if err != nil {
		return SyncResult{}, err
	}
	if len(discovered) == 0 {
		return SyncResult{}, fmt.Errorf("Fornax released no workflows")
	}
	if result, ok := alreadyCurrent(dataRoot, discovered); ok {
		return result, nil
	}
	temp, err := os.MkdirTemp(dataRoot, ".fornax-sync-")
	if err != nil {
		return SyncResult{}, err
	}
	defer os.RemoveAll(temp)
	type heldPackage struct {
		claim       PackageClaim
		workflowDir string
	}
	held := make([]heldPackage, 0, len(discovered))
	claims := make([]PackageClaim, 0, len(discovered))
	for index, item := range discovered {
		destination := filepath.Join(temp, "download", strconv.Itoa(index))
		packageRoot, installErr := client.Install(ctx, item.WorkspaceID, item.Key, item.Version, destination)
		if installErr != nil {
			return SyncResult{}, fmt.Errorf("download %s@%s: %w", item.Key, item.Version, installErr)
		}
		parent := filepath.Join(temp, "validated", strconv.Itoa(index))
		accepted, acceptErr := MaterializePackage(packageRoot, parent)
		if acceptErr != nil {
			if item.Access == AccessReadOnly {
				warnSkippedPackage(item.Key, item.Version, acceptErr)
				continue
			}
			return SyncResult{}, fmt.Errorf("validate %s@%s: %w", item.Key, item.Version, acceptErr)
		}
		claim := PackageClaim{WorkflowID: accepted.WorkflowID, Key: item.Key, Version: item.Version, Access: item.Access, SourceIndex: item.SourceIndex}
		claims = append(claims, claim)
		held = append(held, heldPackage{claim: claim, workflowDir: filepath.Join(parent, accepted.WorkflowID)})
	}
	if len(claims) == 0 {
		return SyncResult{}, fmt.Errorf("Fornax released no workflows")
	}
	selected, err := SelectClaims(claims)
	if err != nil {
		return SyncResult{}, err
	}
	stage := filepath.Join(temp, "stage")
	workflowRoot := filepath.Join(stage, "fanloop-workflow")
	if err = os.MkdirAll(workflowRoot, 0755); err != nil {
		return SyncResult{}, err
	}
	releases := make([]configuration.ReleaseRef, 0, len(selected))
	workflows := make([]SyncedWorkflow, 0, len(selected))
	for _, claim := range selected {
		sourceDir := ""
		for _, item := range held {
			if item.claim == claim {
				sourceDir = item.workflowDir
				break
			}
		}
		if sourceDir == "" {
			return SyncResult{}, fmt.Errorf("selected workflow %s has no package", claim.WorkflowID)
		}
		if err = os.Rename(sourceDir, filepath.Join(workflowRoot, claim.WorkflowID)); err != nil {
			return SyncResult{}, err
		}
		releases = append(releases, configuration.ReleaseRef{Key: claim.Key, Version: claim.Version, WorkflowID: claim.WorkflowID})
		workflows = append(workflows, SyncedWorkflow{ID: claim.WorkflowID, Key: claim.Key, Version: claim.Version})
	}
	if _, err = workflow.ListFrom(workflowRoot); err != nil {
		return SyncResult{}, err
	}
	currentLink := filepath.Join(dataRoot, "config", "current")
	previous, _ := os.Readlink(currentLink)
	snapshot, err := configuration.PublishRelease(dataRoot, stage, releases)
	if err != nil {
		return SyncResult{}, err
	}
	if err = activateSnapshot(dataRoot, snapshot); err != nil {
		if previous != "" {
			_ = configuration.Activate(dataRoot, previous)
		}
		return SyncResult{}, err
	}
	identity, err := treeIdentity(filepath.Join(snapshot, "fanloop-workflow"))
	if err != nil {
		if previous != "" {
			_ = configuration.Activate(dataRoot, previous)
		}
		return SyncResult{}, err
	}
	receipt := syncReceipt{
		SchemaVersion:  receiptSchema,
		Discovered:     discovered,
		Activated:      releases,
		SnapshotDigest: configuration.ReleaseSetDigest(releases),
		TreeIdentity:   identity,
	}
	if err = commitReceipt(dataRoot, receipt); err != nil {
		if previous != "" {
			_ = configuration.Activate(dataRoot, previous)
		}
		return SyncResult{}, err
	}
	return SyncResult{Outcome: OutcomeUpdated, Workflows: workflows, Digest: receipt.SnapshotDigest}, nil
}

func Configured(dataRoot string) bool {
	info, err := os.Lstat(filepath.Join(dataRoot, "fornax.json"))
	return err == nil && info.Mode().IsRegular()
}

// UseReleasedSnapshot is the same opt-in flow init uses: FANLOOP_USE_FORNAX
// is 1, true, or yes, and fornax.json is a regular file.
func UseReleasedSnapshot(dataRoot string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("FANLOOP_USE_FORNAX"))) {
	case "1", "true", "yes":
		return Configured(dataRoot)
	default:
		return false
	}
}

// CurrentRoot returns the active snapshot, syncing once when none is installed.
func CurrentRoot(ctx context.Context, dataRoot string, dryRun bool) (string, func(), error) {
	current := filepath.Join(dataRoot, "config", "current")
	target, err := os.Readlink(current)
	if err != nil {
		if dryRun {
			return "", nil, fmt.Errorf("Fornax configuration is not installed")
		}
		if _, err = Sync(ctx, dataRoot, DefaultClient()); err != nil {
			return "", nil, err
		}
		target, err = os.Readlink(current)
		if err != nil {
			return "", nil, err
		}
	}
	root, err := workflow.SourceRoot(target)
	if err != nil {
		return "", nil, err
	}
	unlockConfig, err := workflow.LockConfiguration(dataRoot, false)
	if err != nil {
		return "", nil, err
	}
	unlockSource, err := workflow.LockSource(root, false)
	if err != nil {
		unlockConfig()
		return "", nil, err
	}
	if _, err = configuration.Read(target); err != nil {
		unlockSource()
		unlockConfig()
		return "", nil, err
	}
	return root, func() { unlockSource(); unlockConfig() }, nil
}

func discoverReleases(ctx context.Context, config Config, client Client) ([]discoveredRelease, error) {
	found := []discoveredRelease{}
	for index, source := range config.Sources {
		skills, err := client.ListByPrefix(ctx, source.WorkspaceID, source.KeyPrefix)
		if err != nil {
			return nil, fmt.Errorf("discover Fornax workflows: %w", err)
		}
		for _, skill := range skills {
			releases, err := client.Releases(ctx, source.WorkspaceID, skill.ID)
			if err != nil {
				return nil, fmt.Errorf("discover Fornax releases: %w", err)
			}
			release, ok, err := latestRelease(releases)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			found = append(found, discoveredRelease{
				SourceIndex: index, WorkspaceID: source.WorkspaceID, Key: skill.Key,
				Version: release.Version, Access: source.Access,
			})
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i].Key < found[j].Key })
	return found, nil
}

func alreadyCurrent(dataRoot string, discovered []discoveredRelease) (SyncResult, bool) {
	receipt, ok := readReceipt(dataRoot)
	if !ok || !sameDiscovered(receipt.Discovered, discovered) {
		return SyncResult{}, false
	}
	target, err := os.Readlink(filepath.Join(dataRoot, "config", "current"))
	if err != nil || filepath.Base(target) != receipt.SnapshotDigest {
		return SyncResult{}, false
	}
	identity, err := treeIdentity(filepath.Join(target, "fanloop-workflow"))
	if err != nil || identity != receipt.TreeIdentity {
		return SyncResult{}, false
	}
	workflows := make([]SyncedWorkflow, 0, len(receipt.Activated))
	for _, item := range receipt.Activated {
		workflows = append(workflows, SyncedWorkflow{ID: item.WorkflowID, Key: item.Key, Version: item.Version})
	}
	sort.Slice(workflows, func(i, j int) bool { return workflows[i].ID < workflows[j].ID })
	return SyncResult{Outcome: OutcomeUnchanged, Workflows: workflows, Digest: receipt.SnapshotDigest}, true
}

func sameDiscovered(left, right []discoveredRelease) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func treeIdentity(root string) (string, error) {
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("workflow root must be a real directory")
	}
	lines := []string{}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, failure error) error {
		if failure != nil {
			return failure
		}
		if path == root {
			return nil
		}
		meta, err := entry.Info()
		if err != nil {
			return err
		}
		if meta.Mode()&os.ModeSymlink != 0 || (!meta.IsDir() && !meta.Mode().IsRegular()) {
			return fmt.Errorf("workflow tree contains a special file")
		}
		if meta.IsDir() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		mode := os.FileMode(0644)
		if meta.Mode().Perm()&0111 != 0 {
			mode = 0755
		}
		sum := sha256.Sum256(content)
		lines = append(lines, fmt.Sprintf("%s %04o %s", filepath.ToSlash(relative), mode, hex.EncodeToString(sum[:])))
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n") + "\n"))
	return hex.EncodeToString(sum[:]), nil
}

func receiptPath(dataRoot string) string {
	return filepath.Join(dataRoot, "cache", "fornax-workflows.json")
}

func readReceipt(dataRoot string) (syncReceipt, bool) {
	path := receiptPath(dataRoot)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > 1<<20 {
		return syncReceipt{}, false
	}
	file, err := os.Open(path)
	if err != nil {
		return syncReceipt{}, false
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 1<<20))
	decoder.DisallowUnknownFields()
	var receipt syncReceipt
	if err = decoder.Decode(&receipt); err != nil || receipt.SchemaVersion != receiptSchema {
		return syncReceipt{}, false
	}
	var trailing any
	if err = decoder.Decode(&trailing); err != io.EOF {
		return syncReceipt{}, false
	}
	return receipt, true
}

func writeReceipt(dataRoot string, receipt syncReceipt) error {
	directory := filepath.Join(dataRoot, "cache")
	if err := os.MkdirAll(directory, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".fornax-workflows-")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	_, writeErr := temporary.Write(append(data, '\n'))
	closeErr := temporary.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(temporaryName)
		if writeErr != nil {
			return writeErr
		}
		return closeErr
	}
	return os.Rename(temporaryName, receiptPath(dataRoot))
}
