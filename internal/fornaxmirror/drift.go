package fornaxmirror

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/zeefan1555/fanloop/internal/configuration"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

type DriftStatus string

const (
	DriftInSync        DriftStatus = "in_sync"
	DriftLocalBehind   DriftStatus = "local_behind"
	DriftLocalModified DriftStatus = "local_modified"
	DriftUnavailable   DriftStatus = "unavailable"
)

type WorkflowStatus string

const (
	WorkflowInSync        WorkflowStatus = "in_sync"
	WorkflowLocalBehind   WorkflowStatus = "local_behind"
	WorkflowLocalModified WorkflowStatus = "local_modified"
	WorkflowUnavailable   WorkflowStatus = "unavailable"
)

type WorkflowCheck struct {
	WorkflowID      string
	SkillKey        string
	Status          WorkflowStatus
	SkillID         string
	ReleasedVersion string
	Detail          string
}

type Report struct {
	Status              DriftStatus
	ConfigurationCommit string
	Workflows           []WorkflowCheck
}

var ErrInvalidInput = errors.New("invalid config drift input")

type snapshot struct {
	config   Config
	source   string
	commit   string
	manifest configuration.Manifest
}

func Check(ctx context.Context, dataRoot string, client Client) (Report, error) {
	resolved, err := resolveSnapshot(dataRoot)
	if err != nil {
		return Report{}, err
	}
	checks, err := compareReleased(ctx, dataRoot, resolved, client)
	if err != nil {
		return Report{}, err
	}
	if len(checks) == 0 {
		return Report{}, fmt.Errorf("drift report contains no Workflows")
	}
	sort.Slice(checks, func(i, j int) bool { return checks[i].WorkflowID < checks[j].WorkflowID })
	commit := resolved.commit
	if resolved.manifest.Origin == configuration.OriginFornax {
		commit = ""
	}
	return Report{Status: summarizeDrift(checks), ConfigurationCommit: commit, Workflows: checks}, nil
}

func resolveSnapshot(dataRoot string) (snapshot, error) {
	if _, explicit := os.LookupEnv("FANLOOP_CONFIG_SOURCE"); explicit {
		return snapshot{}, invalidInputError("config drift requires the managed snapshot")
	}
	config, err := LoadConfig(filepath.Join(dataRoot, "fornax.json"))
	if err != nil {
		return snapshot{}, invalidInputError(err.Error())
	}
	unlock, err := workflow.LockConfiguration(dataRoot, false)
	if err != nil {
		return snapshot{}, err
	}
	defer unlock()
	workflowRoot, err := workflow.ConfigurationRoot(dataRoot)
	if err != nil {
		return snapshot{}, fmt.Errorf("active configuration is unavailable: %w", err)
	}
	source := filepath.Dir(workflowRoot)
	identity, err := configuration.Read(source)
	if err != nil {
		return snapshot{}, fmt.Errorf("active configuration is invalid: %w", err)
	}
	return snapshot{config: config, source: source, commit: identity.CommitSHA, manifest: identity}, nil
}

func invalidInputError(message string) error { return fmt.Errorf("%w: %s", ErrInvalidInput, message) }

type heldRelease struct {
	claim       PackageClaim
	skillID     string
	workflowDir string
}

func compareReleased(ctx context.Context, dataRoot string, resolved snapshot, client Client) ([]WorkflowCheck, error) {
	temp, err := os.MkdirTemp("", "fanloop-fornax-drift-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(temp)
	held := []heldRelease{}
	claims := []PackageClaim{}
	unreleased := map[string]bool{}
	for sourceIndex, source := range resolved.config.Sources {
		skills, err := client.ListByPrefix(ctx, source.WorkspaceID, source.KeyPrefix)
		if err != nil {
			return unavailableChecks(resolved, "Fornax Skill discovery failed"), nil
		}
		for skillIndex, skill := range skills {
			releases, err := client.Releases(ctx, source.WorkspaceID, skill.ID)
			if err != nil {
				return unavailableChecks(resolved, "Fornax release discovery failed"), nil
			}
			release, ok, err := latestRelease(releases)
			if err != nil {
				return unavailableChecks(resolved, "latest released version is ambiguous"), nil
			}
			if !ok {
				unreleased[skill.Key] = true
				continue
			}
			folder := fmt.Sprintf("%d-%d", sourceIndex, skillIndex)
			destination := filepath.Join(temp, "download", folder)
			packageRoot, err := client.Install(ctx, source.WorkspaceID, skill.Key, release.Version, destination)
			if err != nil {
				return unavailableChecks(resolved, "Fornax package download failed"), nil
			}
			parent := filepath.Join(temp, "validated", folder)
			accepted, err := MaterializePackage(packageRoot, parent)
			if err != nil {
				if source.Access == AccessReadOnly {
					warnSkippedPackage(skill.Key, release.Version, err)
					continue
				}
				return unavailableChecks(resolved, "Fornax package is invalid"), nil
			}
			claim := PackageClaim{WorkflowID: accepted.WorkflowID, Key: skill.Key, Version: release.Version, Access: source.Access, SourceIndex: sourceIndex}
			claims = append(claims, claim)
			held = append(held, heldRelease{claim: claim, skillID: skill.ID, workflowDir: filepath.Join(parent, accepted.WorkflowID)})
		}
	}
	selected, err := SelectClaims(claims)
	if err != nil {
		return unavailableChecks(resolved, err.Error()), nil
	}
	byID := map[string]heldRelease{}
	for _, item := range held {
		for _, claim := range selected {
			if item.claim == claim {
				byID[claim.WorkflowID] = item
			}
		}
	}
	seen := map[string]bool{}
	checks := []WorkflowCheck{}
	for _, id := range resolved.manifest.WorkflowIDs {
		seen[id] = true
		checks = append(checks, checkInstalled(dataRoot, resolved, id, byID[id], unreleased))
	}
	for _, claim := range selected {
		if seen[claim.WorkflowID] {
			continue
		}
		item := byID[claim.WorkflowID]
		checks = append(checks, WorkflowCheck{
			WorkflowID: claim.WorkflowID, SkillKey: claim.Key, SkillID: item.skillID,
			ReleasedVersion: claim.Version, Status: WorkflowLocalBehind, Detail: "workflow is not in the active snapshot",
		})
	}
	if len(checks) == 0 {
		return unavailableChecks(resolved, "Fornax released no workflows"), nil
	}
	return checks, nil
}

func checkInstalled(dataRoot string, resolved snapshot, id string, item heldRelease, unreleased map[string]bool) WorkflowCheck {
	key := item.claim.Key
	if key == "" {
		key = recordedKey(resolved.manifest, id)
	}
	if key == "" {
		derived, err := SkillKey(id)
		if err != nil {
			return WorkflowCheck{WorkflowID: id, Status: WorkflowUnavailable, Detail: err.Error()}
		}
		key = derived
	}
	check := WorkflowCheck{WorkflowID: id, SkillKey: key, SkillID: item.skillID, ReleasedVersion: item.claim.Version}
	if item.workflowDir == "" {
		check.Status = WorkflowUnavailable
		if unreleased[key] {
			check.Detail = "Online/prod has no released version"
		} else {
			check.Detail = "Skill key is missing"
		}
		return check
	}
	same, err := sameWorkflowBytes(filepath.Join(resolved.source, "fanloop-workflow", id), item.workflowDir)
	if err != nil {
		check.Status, check.Detail = WorkflowUnavailable, "workflow tree cannot be compared"
		return check
	}
	if same {
		check.Status = WorkflowInSync
		return check
	}
	recorded, ok := recordedVersion(resolved.manifest, id)
	if ok && versionLess(recorded, item.claim.Version) && snapshotMatchesReceipt(dataRoot, resolved.source) {
		check.Status = WorkflowLocalBehind
		check.Detail = "released version is " + item.claim.Version
		return check
	}
	check.Status, check.Detail = WorkflowLocalModified, "package content differs"
	return check
}

func recordedKey(manifest configuration.Manifest, id string) string {
	for _, release := range manifest.Releases {
		if release.WorkflowID == id {
			return release.Key
		}
	}
	return ""
}

func recordedVersion(manifest configuration.Manifest, id string) (string, bool) {
	for _, release := range manifest.Releases {
		if release.WorkflowID == id && release.Version != "" {
			return release.Version, true
		}
	}
	return "", false
}

func versionLess(local, released string) bool {
	left, right := splitVersion(local), splitVersion(released)
	if left == nil || right == nil {
		return false
	}
	for index := 0; index < 3; index++ {
		if left[index] != right[index] {
			return left[index] < right[index]
		}
	}
	return false
}

func splitVersion(value string) []int {
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return nil
	}
	numbers := make([]int, 3)
	for index, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 || part != strconv.Itoa(number) {
			return nil
		}
		numbers[index] = number
	}
	return numbers
}

func snapshotMatchesReceipt(dataRoot, source string) bool {
	receipt, ok := readReceipt(dataRoot)
	if !ok {
		return true
	}
	identity, err := treeIdentity(filepath.Join(source, "fanloop-workflow"))
	return err == nil && identity == receipt.TreeIdentity
}

func sameWorkflowBytes(left, right string) (bool, error) {
	leftFiles, err := fileHashes(left)
	if err != nil {
		return false, err
	}
	rightFiles, err := fileHashes(right)
	if err != nil {
		return false, err
	}
	if len(leftFiles) != len(rightFiles) {
		return false, nil
	}
	for path, sum := range leftFiles {
		if rightFiles[path] != sum {
			return false, nil
		}
	}
	return true, nil
}

func fileHashes(root string) (map[string]string, error) {
	hashes := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, failure error) error {
		if failure != nil {
			return failure
		}
		if path == root || entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("special file")
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(content)
		hashes[filepath.ToSlash(relative)] = hex.EncodeToString(sum[:])
		return nil
	})
	return hashes, err
}

func unavailableChecks(resolved snapshot, detail string) []WorkflowCheck {
	if len(resolved.manifest.WorkflowIDs) == 0 {
		return []WorkflowCheck{{WorkflowID: "configuration", SkillKey: SkillKeyPrefix, Status: WorkflowUnavailable, Detail: detail}}
	}
	checks := make([]WorkflowCheck, 0, len(resolved.manifest.WorkflowIDs))
	for _, id := range resolved.manifest.WorkflowIDs {
		key := recordedKey(resolved.manifest, id)
		if key == "" {
			key, _ = SkillKey(id)
		}
		if key == "" {
			key = SkillKeyPrefix + id
		}
		checks = append(checks, WorkflowCheck{WorkflowID: id, SkillKey: key, Status: WorkflowUnavailable, Detail: detail})
	}
	return checks
}

func summarizeDrift(checks []WorkflowCheck) DriftStatus {
	status := DriftInSync
	for _, check := range checks {
		switch check.Status {
		case WorkflowUnavailable:
			return DriftUnavailable
		case WorkflowLocalModified:
			status = DriftLocalModified
		case WorkflowLocalBehind:
			if status == DriftInSync {
				status = DriftLocalBehind
			}
		}
	}
	return status
}

func latestRelease(releases []Release) (Release, bool, error) {
	if len(releases) == 0 {
		return Release{}, false, nil
	}
	sort.Slice(releases, func(i, j int) bool {
		if releases[i].UpdatedAt == releases[j].UpdatedAt {
			return releases[i].Version < releases[j].Version
		}
		return releases[i].UpdatedAt > releases[j].UpdatedAt
	})
	if len(releases) > 1 && releases[0].UpdatedAt == releases[1].UpdatedAt && releases[0].Version != releases[1].Version {
		return Release{}, false, fmt.Errorf("ambiguous release timestamp")
	}
	return releases[0], true, nil
}

func (report Report) Validate() error {
	if len(report.Workflows) == 0 {
		return fmt.Errorf("drift report contains no Workflows")
	}
	return nil
}
