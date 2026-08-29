package install

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/zeefan1555/commonloop/internal/doctor"
	"github.com/zeefan1555/commonloop/internal/idl/opsidl"
	"github.com/zeefan1555/commonloop/internal/release"
)

var skillNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

type Request struct {
	Source         string
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
	manifest, err := release.Load(request.Source)
	if err != nil {
		return Result{}, fmt.Errorf("release manifest: %w", err)
	}
	sourceBinary := filepath.Join(request.Source, "bin", "commonloop")
	if result := (doctor.Runtime{ReleaseRoot: request.Source, BinaryPath: sourceBinary}).Run(""); result.Status == opsidl.DoctorStatus_unhealthy {
		return Result{}, fmt.Errorf("Doctor rejected staged release %s", manifest.ReleaseVersion)
	}

	releasesRoot := filepath.Join(request.DataRoot, "releases")
	target := filepath.Join(releasesRoot, manifest.ReleaseVersion)
	current := filepath.Join(request.DataRoot, "current")
	if err := preflightCurrent(current, releasesRoot); err != nil {
		return Result{}, err
	}
	linksToCreate, externalLinks, err := preflightSkillLinks(request, manifest)
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
			(doctor.Runtime{ReleaseRoot: target, BinaryPath: filepath.Join(target, "bin", "commonloop")}).Run("").Status != opsidl.DoctorStatus_unhealthy
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
		if result := (doctor.Runtime{ReleaseRoot: temporary, BinaryPath: filepath.Join(temporary, "bin", "commonloop")}).Run(""); result.Status == opsidl.DoctorStatus_unhealthy {
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

	removedExternalLinks, err := removeSkillLinks(externalLinks)
	if err != nil {
		rollbackInstall()
		return Result{}, err
	}
	createdLinks, err := createSkillLinks(linksToCreate)
	if err != nil {
		rollbackInstall()
		return Result{}, restoreSkillLinks(err, removedExternalLinks)
	}
	removedLinks, err := removeSkillLinks(obsoleteLinks)
	if err != nil {
		err = rollbackSkillLinkChanges(err, createdLinks, removedExternalLinks)
		rollbackInstall()
		return Result{}, err
	}
	if err := activateCurrent(current, filepath.Join("releases", manifest.ReleaseVersion)); err != nil {
		if _, restoreErr := createSkillLinks(removedLinks); restoreErr != nil {
			err = fmt.Errorf("activate current: %v; restore retired Skill links: %w", err, restoreErr)
		}
		err = rollbackSkillLinkChanges(err, createdLinks, removedExternalLinks)
		rollbackInstall()
		return Result{}, err
	}
	if backup != "" {
		_ = os.RemoveAll(backup)
	}

	skills := make([]string, len(manifest.Skills))
	for index, skill := range manifest.Skills {
		skills[index] = skill.Name
	}
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

func preflightSkillLinks(request Request, manifest release.Manifest) ([]linkPlan, []linkPlan, error) {
	toCreate, external := []linkPlan{}, []linkPlan{}
	skill, _ := manifest.Skill(release.ExposedSkillName)
	for _, root := range request.SkillRoots.Values() {
		plan := linkPlan{
			path:   filepath.Join(root, skill.Name),
			target: filepath.Join(request.DataRoot, "current", filepath.FromSlash(skill.Path)),
		}
		info, err := os.Lstat(plan.path)
		if os.IsNotExist(err) {
			toCreate = append(toCreate, plan)
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			return nil, nil, fmt.Errorf("refusing to replace non-Commonloop path %s", plan.path)
		}
		target, err := os.Readlink(plan.path)
		if err != nil {
			return nil, nil, err
		}
		if filepath.Clean(target) != filepath.Clean(plan.target) {
			toCreate = append(toCreate, plan)
			external = append(external, linkPlan{path: plan.path, target: target})
		}
	}
	return toCreate, external, nil
}

func preflightObsoleteSkillLinks(request Request) ([]linkPlan, error) {
	packagedSkills := currentSkills(filepath.Join(request.DataRoot, "current"))
	wanted := map[string]bool{release.ExposedSkillName: true}
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
	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("refusing to replace non-Commonloop current path %s", current)
	}
	target, err := os.Readlink(current)
	if err != nil {
		return err
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(current), target)
	}
	relative, err := filepath.Rel(releasesRoot, filepath.Clean(target))
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("refusing to replace external current link %s", current)
	}
	return nil
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

func restoreSkillLinks(cause error, plans []linkPlan) error {
	if len(plans) == 0 {
		return cause
	}
	if _, err := createSkillLinks(plans); err != nil {
		return fmt.Errorf("%w; restore external Skill links: %v", cause, err)
	}
	return cause
}

func rollbackSkillLinkChanges(cause error, created []string, external []linkPlan) error {
	rollbackLinks(created)
	return restoreSkillLinks(cause, external)
}

func activateCurrent(current, target string) error {
	if existing, err := os.Readlink(current); err == nil && filepath.Clean(existing) == filepath.Clean(target) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(current), 0o755); err != nil {
		return err
	}
	placeholder, err := os.CreateTemp(filepath.Dir(current), ".current-")
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
	return os.Rename(temporary, current)
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
