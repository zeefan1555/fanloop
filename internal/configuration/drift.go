package configuration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/zeefan1555/fanloop/internal/workflow"
)

// DriftStatus is the Git comparison of the active snapshot with second-brain-skills main.
type DriftStatus string

const (
	DriftInSync        DriftStatus = "in_sync"
	DriftLocalBehind   DriftStatus = "local_behind"
	DriftLocalModified DriftStatus = "local_modified"
	DriftUnavailable   DriftStatus = "unavailable"
)

// DriftWorkflow is one workflow id in a Git drift report. Git mode has no skill key.
type DriftWorkflow struct {
	WorkflowID string
	Status     DriftStatus
	Detail     string
}

// DriftReport is the read-only Git comparison of config/current with main.
type DriftReport struct {
	Status              DriftStatus
	ConfigurationCommit string
	RemoteCommit        string
	Workflows           []DriftWorkflow
}

// ErrDriftInput is an author-source or other caller mistake. It is not a fetch failure.
var ErrDriftInput = errors.New("invalid config drift input")

// Drift compares the managed snapshot with second-brain-skills main.
// FANLOOP_CONFIG_SOURCE is refused. A Fornax-origin snapshot is local_behind.
func Drift(ctx context.Context, dataRoot string) (DriftReport, error) {
	if _, explicit := os.LookupEnv("FANLOOP_CONFIG_SOURCE"); explicit {
		return DriftReport{}, fmt.Errorf("%w: config drift requires the managed snapshot", ErrDriftInput)
	}
	if !filepath.IsAbs(dataRoot) {
		return DriftReport{}, fmt.Errorf("configuration data root must be absolute")
	}
	unlock, err := workflow.LockConfiguration(dataRoot, false)
	if err != nil {
		return DriftReport{}, err
	}
	defer unlock()
	target, err := os.Readlink(filepath.Join(dataRoot, "config", "current"))
	if err != nil {
		return DriftReport{}, fmt.Errorf("active configuration is unavailable: %w", err)
	}
	recorded, err := recordedManifest(target)
	if err != nil {
		return DriftReport{}, fmt.Errorf("active configuration is invalid: %w", err)
	}
	if len(recorded.WorkflowIDs) == 0 {
		return DriftReport{}, fmt.Errorf("drift report contains no Workflows")
	}
	ctx = withRemoteSession(ctx)
	remote, remoteErr := remoteCommit(ctx, dataRoot)
	if remoteErr != nil {
		remote = ""
	}
	if recorded.Origin == OriginFornax {
		return driftReport(recorded, DriftLocalBehind, "snapshot came from Fornax; run fanloop update config", remote), nil
	}
	matched, matchErr := snapshotFilesMatch(target, recorded)
	if matchErr != nil || !matched {
		return driftReport(recorded, DriftLocalModified, "package content differs", remote), nil
	}
	if remoteErr != nil {
		return driftReport(recorded, DriftUnavailable, "configuration fetch failed", ""), nil
	}
	status := DriftLocalBehind
	if remote == recorded.CommitSHA {
		status = DriftInSync
	}
	return driftReport(recorded, status, "", remote), nil
}

func driftReport(recorded Manifest, status DriftStatus, detail, remote string) DriftReport {
	workflows := make([]DriftWorkflow, 0, len(recorded.WorkflowIDs))
	for _, id := range recorded.WorkflowIDs {
		workflows = append(workflows, DriftWorkflow{WorkflowID: id, Status: status, Detail: detail})
	}
	sort.Slice(workflows, func(i, j int) bool { return workflows[i].WorkflowID < workflows[j].WorkflowID })
	commit := recorded.CommitSHA
	if recorded.Origin == OriginFornax {
		commit = ""
	}
	report := DriftReport{Status: status, ConfigurationCommit: commit, RemoteCommit: remote, Workflows: workflows}
	return report
}

func recordedManifest(source string) (Manifest, error) {
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
	if recorded.SchemaVersion != 1 {
		return Manifest{}, fmt.Errorf("invalid configuration manifest")
	}
	return recorded, nil
}

func snapshotFilesMatch(source string, recorded Manifest) (bool, error) {
	actual, err := inspect(source, recorded.CommitSHA)
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(recorded.Files, actual.Files) && reflect.DeepEqual(recorded.WorkflowIDs, actual.WorkflowIDs), nil
}
