package fornaxmirror

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zeefan1555/fanloop/internal/configuration"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

type DriftStatus string

const (
	DriftInSync      DriftStatus = "in_sync"
	Drifted          DriftStatus = "drifted"
	DriftUnavailable DriftStatus = "unavailable"
)

type WorkflowStatus string

const (
	WorkflowInSync      WorkflowStatus = "in_sync"
	WorkflowMissing     WorkflowStatus = "missing"
	WorkflowUnreleased  WorkflowStatus = "unreleased"
	WorkflowDrifted     WorkflowStatus = "drifted"
	WorkflowUnavailable WorkflowStatus = "unavailable"
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
	config      Config
	source      string
	commit      string
	workflowIDs []string
}

func Check(ctx context.Context, dataRoot string, client Client) (Report, error) {
	resolved, err := resolveSnapshot(dataRoot)
	if err != nil {
		return Report{}, err
	}
	config := resolved.config
	source, err := config.ReadWriteSource()
	if err != nil {
		return Report{}, invalidInputError(err.Error())
	}
	if source.KeyPrefix != SkillKeyPrefix {
		return Report{}, invalidInputError("read_write source must use the fanloop.workflow. prefix")
	}
	report := Report{Status: DriftInSync, ConfigurationCommit: resolved.commit, Workflows: make([]WorkflowCheck, 0, len(resolved.workflowIDs))}
	for _, workflowID := range resolved.workflowIDs {
		check := inspectWorkflow(ctx, resolved.source, resolved.commit, source.WorkspaceID, workflowID, client)
		report.Workflows = append(report.Workflows, check)
		if check.Status == WorkflowUnavailable {
			report.Status = DriftUnavailable
		} else if check.Status != WorkflowInSync && report.Status == DriftInSync {
			report.Status = Drifted
		}
	}
	return report, nil
}

func resolveSnapshot(dataRoot string) (snapshot, error) {
	if _, explicit := os.LookupEnv("FANLOOP_CONFIG_SOURCE"); explicit {
		return snapshot{}, invalidInputError("config drift requires the managed Git snapshot")
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
		return snapshot{}, fmt.Errorf("active Git configuration is unavailable: %w", err)
	}
	source := filepath.Dir(workflowRoot)
	identity, err := configuration.Read(source)
	if err != nil {
		return snapshot{}, fmt.Errorf("active Git configuration is invalid: %w", err)
	}
	return snapshot{config: config, source: source, commit: identity.CommitSHA, workflowIDs: identity.WorkflowIDs}, nil
}

func invalidInputError(message string) error { return fmt.Errorf("%w: %s", ErrInvalidInput, message) }

func inspectWorkflow(ctx context.Context, source, commit, workspaceID, workflowID string, client Client) WorkflowCheck {
	key, err := SkillKey(workflowID)
	if err != nil {
		return WorkflowCheck{WorkflowID: workflowID, Status: WorkflowUnavailable, Detail: err.Error()}
	}
	check := WorkflowCheck{WorkflowID: workflowID, SkillKey: key}
	skills, err := client.Skills(ctx, workspaceID, key)
	if err != nil {
		check.Status, check.Detail = WorkflowUnavailable, "Fornax Skill discovery failed"
		return check
	}
	if len(skills) == 0 {
		check.Status, check.Detail = WorkflowMissing, "Skill key is missing"
		return check
	}
	if len(skills) != 1 {
		check.Status, check.Detail = WorkflowUnavailable, "Skill key is ambiguous"
		return check
	}
	check.SkillID = skills[0].ID
	releases, err := client.Releases(ctx, workspaceID, check.SkillID)
	if err != nil {
		check.Status, check.Detail = WorkflowUnavailable, "Fornax release discovery failed"
		return check
	}
	release, ok, releaseErr := latestRelease(releases)
	if releaseErr != nil {
		check.Status, check.Detail = WorkflowUnavailable, "latest released version is ambiguous"
		return check
	}
	if !ok {
		check.Status, check.Detail = WorkflowUnreleased, "Online/prod has no released version"
		return check
	}
	check.ReleasedVersion = release.Version
	root, err := os.MkdirTemp("", "fanloop-fornax-drift-")
	if err != nil {
		check.Status, check.Detail = WorkflowUnavailable, "temporary directory creation failed"
		return check
	}
	defer os.RemoveAll(root)
	downloaded, err := client.Install(ctx, workspaceID, key, release.Version, filepath.Join(root, "download"))
	if err != nil {
		check.Status, check.Detail = WorkflowUnavailable, "Fornax package download failed"
		return check
	}
	candidate := filepath.Join(root, workflowID+".zip")
	if _, err = BuildArchive(source, workflowID, key, commit, release.Version, candidate); err != nil {
		check.Status, check.Detail = WorkflowUnavailable, "local Git package is invalid"
		return check
	}
	comparison, err := CompareArchiveDirectory(candidate, downloaded)
	if err != nil {
		check.Status, check.Detail = WorkflowDrifted, "package content differs"
		return check
	}
	if len(comparison.Notes) > 0 {
		check.Detail = strings.Join(comparison.Notes, "; ")
	}
	check.Status = WorkflowInSync
	return check
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
