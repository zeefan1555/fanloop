package install

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/zeefan1555/fanloop/internal/doctor"
	"github.com/zeefan1555/fanloop/internal/idl/opsidl"
	"github.com/zeefan1555/fanloop/internal/release"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

var skillNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

type Request struct {
	Source         string
	ConfigSource   string
	DataRoot       string
	SkillRoots     release.SkillRoots
	ReplaceInvalid bool
}

type Result struct {
	ReleaseVersion string   `json:"release_version"`
	ReleasePath    string   `json:"release_path"`
	CurrentPath    string   `json:"current_path"`
	Skills         []string `json:"skills"`
	Reused         bool     `json:"reused"`
}

func Run(request Request) (Result, error) {
	for name, path := range map[string]string{
		"source": request.Source, "data root": request.DataRoot,
		"Codex Skills root": request.SkillRoots.Codex, "Agent Skills root": request.SkillRoots.Agent,
		"Trae Skills root": request.SkillRoots.Trae, "Claude Skills root": request.SkillRoots.Claude,
	} {
		if !filepath.IsAbs(path) || filepath.Clean(path) == string(filepath.Separator) {
			return Result{}, fmt.Errorf("%s must be an absolute path", name)
		}
	}

	unlock, err := workflow.LockConfiguration(request.DataRoot, true)
	if err != nil {
		return Result{}, err
	}
	defer unlock()

	manifest, err := release.Load(request.Source)
	if err != nil {
		return Result{}, fmt.Errorf("release manifest: %w", err)
	}
	sourceBinary := filepath.Join(request.Source, "bin", "fanloop")
	releasesRoot := filepath.Join(request.DataRoot, "releases")
	target := filepath.Join(releasesRoot, manifest.ReleaseVersion)
	current := filepath.Join(request.DataRoot, "current")
	if err := preflightCurrent(current, releasesRoot); err != nil {
		return Result{}, err
	}
	workflowRoot, err := workflow.SourceRoot(request.ConfigSource)
	if err != nil {
		return Result{}, err
	}
	unlockSource, err := workflow.LockSource(workflowRoot, false)
	if err != nil {
		return Result{}, err
	}
	defer unlockSource()
	definitions, err := workflow.ListFrom(workflowRoot)
	if err != nil {
		return Result{}, fmt.Errorf("local Workflows: %w", err)
	}
	if len(definitions) == 0 {
		return Result{}, fmt.Errorf("local source contains no Workflow packages")
	}
	configuredSkills := []string{}
	for _, definition := range definitions {
		for name := range definition.SkillPaths {
			configuredSkills = append(configuredSkills, name)
		}
	}
	sort.Strings(configuredSkills)
	doctorWorkflowRoot := workflowRoot
	if result := (doctor.Runtime{ReleaseRoot: request.Source, BinaryPath: sourceBinary, WorkflowRoot: doctorWorkflowRoot}).Run(""); result.Status == opsidl.DoctorStatus_unhealthy {
		return Result{}, fmt.Errorf("Doctor rejected staged release %s", manifest.ReleaseVersion)
	}

	linksToCreate, err := preflightSkillLinks(request, manifest)
	if err != nil {
		return Result{}, err
	}
	obsoleteLinks, err := preflightObsoleteSkillLinks(request)
	if err != nil {
		return Result{}, err
	}

	reused, replaceExisting := false, false
	if info, statErr := os.Lstat(target); statErr == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return Result{}, fmt.Errorf("refusing to replace existing release path %s", target)
		}
		existing, loadErr := release.Load(target)
		healthy := loadErr == nil && reflect.DeepEqual(existing, manifest) &&
			(doctor.Runtime{ReleaseRoot: target, BinaryPath: filepath.Join(target, "bin", "fanloop"), WorkflowRoot: doctorWorkflowRoot}).Run("").Status != opsidl.DoctorStatus_unhealthy
		if !healthy {
			if !request.ReplaceInvalid {
				return Result{}, fmt.Errorf("refusing to replace immutable release %s", manifest.ReleaseVersion)
			}
			replaceExisting = true
		} else {
			reused = true
		}
	} else if !os.IsNotExist(statErr) {
		return Result{}, statErr
	}

	if err := os.MkdirAll(releasesRoot, 0o755); err != nil {
		return Result{}, err
	}
	installedNow, backup := false, ""
	if !reused {
		temporary, err := os.MkdirTemp(releasesRoot, ".install-")
		if err != nil {
			return Result{}, err
		}
		defer os.RemoveAll(temporary)
		if err := copyTree(request.Source, temporary); err != nil {
			return Result{}, err
		}
		if result := (doctor.Runtime{ReleaseRoot: temporary, BinaryPath: filepath.Join(temporary, "bin", "fanloop"), WorkflowRoot: doctorWorkflowRoot}).Run(""); result.Status == opsidl.DoctorStatus_unhealthy {
			return Result{}, fmt.Errorf("Doctor rejected copied release %s", manifest.ReleaseVersion)
		}
		if replaceExisting {
			backup, err = reservePath(releasesRoot, ".replaced-")
			if err != nil {
				return Result{}, err
			}
			if err = os.Rename(target, backup); err != nil {
				return Result{}, err
			}
		}
		if err := os.Rename(temporary, target); err != nil {
			if backup != "" {
				_ = os.Rename(backup, target)
			}
			return Result{}, err
		}
		installedNow = true
	}
	rollbackInstall := func() {
		if !installedNow {
			return
		}
		_ = os.RemoveAll(target)
		if backup != "" {
			_ = os.Rename(backup, target)
		}
	}
	createdLinks, err := createSkillLinks(linksToCreate)
	if err != nil {
		rollbackInstall()
		return Result{}, err
	}
	removedLinks, err := removeSkillLinks(obsoleteLinks)
	if err != nil {
		rollbackLinks(createdLinks)
		rollbackInstall()
		return Result{}, err
	}
	if err := activateInstallation(request.DataRoot, filepath.Dir(workflowRoot), filepath.Join("..", "releases", manifest.ReleaseVersion)); err != nil {
		if _, restoreErr := createSkillLinks(removedLinks); restoreErr != nil {
			err = fmt.Errorf("activate current: %v; restore retired Skill links: %w", err, restoreErr)
		}
		rollbackLinks(createdLinks)
		rollbackInstall()
		return Result{}, err
	}
	if backup != "" {
		_ = os.RemoveAll(backup)
	}

	skills := append([]string(nil), configuredSkills...)
	return Result{
		ReleaseVersion: manifest.ReleaseVersion, ReleasePath: target, CurrentPath: current,
		Skills: skills, Reused: reused,
	}, nil
}

func reservePath(root, prefix string) (string, error) {
	path, err := os.MkdirTemp(root, prefix)
	if err != nil {
		return "", err
	}
	if err := os.Remove(path); err != nil {
		return "", err
	}
	return path, nil
}

type linkPlan struct{ path, target string }

func preflightSkillLinks(request Request, manifest release.Manifest) ([]linkPlan, error) {
	toCreate := []linkPlan{}
	for _, exposed := range release.ExposedSkills() {
		skill, _ := manifest.Skill(exposed.Name)
		for _, root := range request.SkillRoots.Values() {
			plan := linkPlan{
				path:   filepath.Join(root, skill.Name),
				target: filepath.Join(request.DataRoot, "current", filepath.FromSlash(skill.Path)),
			}
			create, err := preflightManagedLink(plan)
			if err != nil {
				return nil, err
			}
			toCreate = append(toCreate, create...)
		}
	}
	return toCreate, nil
}

func preflightManagedLink(plan linkPlan) ([]linkPlan, error) {
	info, err := os.Lstat(plan.path)
	if os.IsNotExist(err) {
		return []linkPlan{plan}, nil
	}
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return nil, fmt.Errorf("refusing to replace non-Fanloop path %s", plan.path)
	}
	target, err := os.Readlink(plan.path)
	if err != nil {
		return nil, err
	}
	if filepath.Clean(target) == filepath.Clean(plan.target) {
		return nil, nil
	}
	return nil, fmt.Errorf("refusing to replace non-Fanloop Skill link %s", plan.path)
}

func preflightObsoleteSkillLinks(request Request) ([]linkPlan, error) {
	packagedSkills := currentSkills(filepath.Join(request.DataRoot, "current"))
	wanted := map[string]bool{}
	for _, exposed := range release.ExposedSkills() {
		wanted[exposed.Name] = true
	}
	obsolete := []linkPlan{}
	for _, skill := range packagedSkills {
		if wanted[skill.name] {
			continue
		}
		for _, root := range request.SkillRoots.Values() {
			plan := linkPlan{
				path:   filepath.Join(root, skill.name),
				target: filepath.Join(request.DataRoot, "current", filepath.FromSlash(skill.path)),
			}
			info, err := os.Lstat(plan.path)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return nil, err
			}
			if info.Mode()&os.ModeSymlink == 0 {
				continue
			}
			target, err := os.Readlink(plan.path)
			if err != nil {
				return nil, err
			}
			if filepath.Clean(target) != filepath.Clean(plan.target) {
				continue
			}
			obsolete = append(obsolete, plan)
		}
	}
	return obsolete, nil
}

type installedSkill struct{ name, path string }

func currentSkills(root string) []installedSkill {
	if current, err := release.Load(root); err == nil {
		result := make([]installedSkill, 0, len(current.Skills))
		for _, skill := range current.Skills {
			result = append(result, installedSkill{name: skill.Name, path: skill.Path})
		}
		return result
	}
	content, err := os.ReadFile(filepath.Join(root, "release.json"))
	if err != nil {
		return nil
	}
	var legacy struct {
		Skills []struct {
			Name string `json:"name"`
			Path string `json:"path"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(content, &legacy); err != nil {
		return nil
	}
	result, seen := []installedSkill{}, map[string]bool{}
	for _, skill := range legacy.Skills {
		if !skillNamePattern.MatchString(skill.Name) || seen[skill.Name] {
			continue
		}
		path := skill.Path
		if path == "" {
			path = "skills/" + skill.Name
		}
		result = append(result, installedSkill{name: skill.Name, path: path})
		seen[skill.Name] = true
	}
	return result
}

func preflightCurrent(current, releasesRoot string) error {
	info, err := os.Lstat(current)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
		for name, target := range map[string]string{"bin": "release/bin", "entrypoints": "release/entrypoints", "release.json": "release/release.json"} {
			actual, readErr := os.Readlink(filepath.Join(current, name))
			if readErr != nil || filepath.Clean(actual) != filepath.Clean(target) {
				return fmt.Errorf("refusing invalid stable current link %s", filepath.Join(current, name))
			}
		}
		releaseLink := filepath.Join(current, "release")
		releaseTarget, readErr := os.Readlink(releaseLink)
		if readErr != nil {
			return fmt.Errorf("refusing invalid stable current link %s", releaseLink)
		}
		if !filepath.IsAbs(releaseTarget) {
			releaseTarget = filepath.Join(current, releaseTarget)
		}
		relative, relativeErr := filepath.Rel(releasesRoot, filepath.Clean(releaseTarget))
		if relativeErr != nil || relative == "." || filepath.Dir(relative) != "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return fmt.Errorf("refusing invalid stable current link %s", releaseLink)
		}
		targetInfo, targetErr := os.Lstat(filepath.Join(releasesRoot, relative))
		if targetErr != nil || !targetInfo.IsDir() || targetInfo.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing invalid stable current link %s", releaseLink)
		}
		return nil
	}
	return fmt.Errorf("refusing non-directory current path %s", current)
}

func createSkillLinks(plans []linkPlan) ([]string, error) {
	created := []string{}
	for _, plan := range plans {
		if err := os.MkdirAll(filepath.Dir(plan.path), 0o755); err != nil {
			rollbackLinks(created)
			return nil, err
		}
		if err := os.Symlink(plan.target, plan.path); err != nil {
			rollbackLinks(created)
			return nil, err
		}
		created = append(created, plan.path)
	}
	return created, nil
}

func removeSkillLinks(plans []linkPlan) ([]linkPlan, error) {
	removed := []linkPlan{}
	restoreRemoved := func(cause error) ([]linkPlan, error) {
		_, restoreErr := createSkillLinks(removed)
		if restoreErr != nil {
			return nil, fmt.Errorf("%w; restore links: %v", cause, restoreErr)
		}
		return nil, cause
	}
	for _, plan := range plans {
		info, err := os.Lstat(plan.path)
		if err != nil {
			return restoreRemoved(fmt.Errorf("inspect Skill link %s: %w", plan.path, err))
		}
		if info.Mode()&os.ModeSymlink == 0 {
			return restoreRemoved(fmt.Errorf("refusing to remove non-link Skill path %s", plan.path))
		}
		target, err := os.Readlink(plan.path)
		if err != nil {
			return restoreRemoved(fmt.Errorf("read Skill link %s: %w", plan.path, err))
		}
		if filepath.Clean(target) != filepath.Clean(plan.target) {
			return restoreRemoved(fmt.Errorf("refusing to remove changed Skill link %s", plan.path))
		}
		if err := os.Remove(plan.path); err != nil {
			return restoreRemoved(fmt.Errorf("remove Skill link %s: %w", plan.path, err))
		}
		removed = append(removed, plan)
	}
	return removed, nil
}

func rollbackLinks(paths []string) {
	for _, path := range paths {
		_ = os.Remove(path)
	}
}

func activateCurrent(current, target string) error {
	if err := os.MkdirAll(current, 0o755); err != nil {
		return err
	}
	return atomicSymlink(filepath.Join(current, "release"), target)
}

func atomicSymlink(path, target string) error {
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("refusing to replace non-link path %s", path)
		}
		existing, readErr := os.Readlink(path)
		if readErr != nil {
			return readErr
		}
		if filepath.Clean(existing) == filepath.Clean(target) {
			return nil
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	placeholder, err := os.CreateTemp(filepath.Dir(path), ".link-")
	if err != nil {
		return err
	}
	temporary := placeholder.Name()
	if closeErr := placeholder.Close(); closeErr != nil {
		_ = os.Remove(temporary)
		return closeErr
	}
	if err := os.Remove(temporary); err != nil {
		return err
	}
	defer os.Remove(temporary)
	if err := os.Symlink(target, temporary); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("refusing to replace non-link path %s", path)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(temporary, path)
}

func activateInstallation(dataRoot, source, releaseTarget string) error {
	binding := filepath.Join(dataRoot, "config", "current")
	previous, err := os.Readlink(binding)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("configuration binding: %w", err)
	}
	if err := atomicSymlink(binding, source); err != nil {
		return err
	}
	if err := ensureCurrent(filepath.Join(dataRoot, "current"), releaseTarget); err != nil {
		var restoreErr error
		if previous == "" {
			restoreErr = os.Remove(binding)
		} else {
			restoreErr = atomicSymlink(binding, previous)
		}
		if restoreErr != nil {
			return fmt.Errorf("activate release: %v; restore configuration binding: %w", err, restoreErr)
		}
		return err
	}
	return nil
}

func ensureCurrent(current, releaseTarget string) error {
	info, err := os.Lstat(current)
	if err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing non-directory current path %s", current)
		}
		return activateCurrent(current, releaseTarget)
	}
	if !os.IsNotExist(err) {
		return err
	}
	parent := filepath.Dir(current)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	staged, err := os.MkdirTemp(parent, ".current-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staged)
	for name, target := range map[string]string{"bin": "release/bin", "entrypoints": "release/entrypoints", "release.json": "release/release.json", "release": releaseTarget} {
		if err := os.Symlink(target, filepath.Join(staged, name)); err != nil {
			return err
		}
	}
	return os.Rename(staged, current)
}

func copyTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("release contains unsupported file %s", relative)
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			_ = input.Close()
			return err
		}
		_, copyErr := io.Copy(output, input)
		inputErr, outputErr := input.Close(), output.Close()
		if copyErr != nil {
			return copyErr
		}
		if inputErr != nil {
			return inputErr
		}
		return outputErr
	})
}
