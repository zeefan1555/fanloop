package skillconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/zeefan1555/fanloop/internal/buildinfo"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

type Skill struct {
	Name       string
	WorkflowID string
	Path       string
}

func DefaultRoot() (string, error) {
	if value := os.Getenv("FANLOOP_CONFIG_ROOT"); value != "" {
		if !filepath.IsAbs(value) {
			return "", fmt.Errorf("FANLOOP_CONFIG_ROOT must be absolute")
		}
		return filepath.Clean(value), nil
	}
	if buildinfo.ReleaseVersion == "dev" {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return "", fmt.Errorf("locate source configuration")
		}
		return filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..")), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dataRoot := filepath.Join(home, ".fanloop")
	if value := os.Getenv("FANLOOP_DATA_HOME"); value != "" {
		dataRoot = filepath.Clean(value)
	}
	return InstalledRoot(dataRoot), nil
}

func InstalledRoot(dataRoot string) string {
	return filepath.Join(dataRoot, "config", "current")
}

func Validate(root string, definitions []workflow.Loaded) ([]Skill, error) {
	skills, groups, err := discover(root)
	if err != nil {
		return nil, err
	}
	wantedGroups := make([]string, 0, len(definitions))
	workflowIDs := map[string]bool{}
	for _, definition := range definitions {
		workflowIDs[definition.Ref.ID] = true
		wantedGroups = append(wantedGroups, definition.Ref.ID)
	}
	sort.Strings(wantedGroups)
	if strings.Join(groups, "\x00") != strings.Join(wantedGroups, "\x00") {
		return nil, fmt.Errorf("Workflow and Skill directories must match: workflows=%v skills=%v", wantedGroups, groups)
	}

	byWorkflow := map[string]map[string]bool{}
	seen := map[string]bool{}
	for _, skill := range skills {
		if !workflowIDs[skill.WorkflowID] {
			return nil, fmt.Errorf("Skill %q uses unknown Workflow group %q", skill.Name, skill.WorkflowID)
		}
		if seen[skill.Name] {
			return nil, fmt.Errorf("duplicate Skill %q", skill.Name)
		}
		seen[skill.Name] = true
		if byWorkflow[skill.WorkflowID] == nil {
			byWorkflow[skill.WorkflowID] = map[string]bool{}
		}
		byWorkflow[skill.WorkflowID][skill.Name] = true
	}
	for _, definition := range definitions {
		promptIDs := make([]string, 0, len(definition.Workflow.Prompts))
		for promptID := range definition.Workflow.Prompts {
			promptIDs = append(promptIDs, promptID)
		}
		sort.Strings(promptIDs)
		for _, promptID := range promptIDs {
			for _, binding := range definition.Workflow.Prompts[promptID].Skills {
				if !byWorkflow[definition.Ref.ID][binding.ID] {
					return nil, fmt.Errorf("Workflow %s prompt %s uses unknown Skill %q", definition.Ref.ID, promptID, binding.ID)
				}
			}
		}
	}
	return skills, nil
}

func Paths(root, workflowID string) (map[string]string, error) {
	skills, _, err := discover(root)
	if err != nil {
		return nil, err
	}
	result := map[string]string{}
	for _, skill := range skills {
		if skill.WorkflowID != workflowID {
			continue
		}
		if _, exists := result[skill.Name]; exists {
			return nil, fmt.Errorf("duplicate Skill %q", skill.Name)
		}
		result[skill.Name] = skill.Path
	}
	return result, nil
}

func discover(root string) ([]Skill, []string, error) {
	entries, err := os.ReadDir(filepath.Join(root, "skills"))
	if err != nil {
		return nil, nil, fmt.Errorf("read Skill configuration: %w", err)
	}
	result := []Skill{}
	groups := []string{}
	for _, group := range entries {
		if group.Type()&os.ModeSymlink != 0 {
			return nil, nil, fmt.Errorf("Skill group %q must not be a symlink", group.Name())
		}
		if !group.IsDir() {
			continue
		}
		if !namePattern.MatchString(group.Name()) {
			return nil, nil, fmt.Errorf("invalid Skill group %q", group.Name())
		}
		groups = append(groups, group.Name())
		skillEntries, err := os.ReadDir(filepath.Join(root, "skills", group.Name()))
		if err != nil {
			return nil, nil, err
		}
		for _, entry := range skillEntries {
			if entry.Type()&os.ModeSymlink != 0 {
				return nil, nil, fmt.Errorf("Skill %q must not be a symlink", entry.Name())
			}
			if !entry.IsDir() {
				continue
			}
			if !namePattern.MatchString(entry.Name()) {
				return nil, nil, fmt.Errorf("invalid Skill %q", entry.Name())
			}
			path := filepath.Join(root, "skills", group.Name(), entry.Name(), "SKILL.md")
			info, err := os.Lstat(path)
			if err != nil || !info.Mode().IsRegular() {
				return nil, nil, fmt.Errorf("Skill %q is missing regular SKILL.md", entry.Name())
			}
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				return nil, nil, err
			}
			result = append(result, Skill{Name: entry.Name(), WorkflowID: group.Name(), Path: resolved})
		}
	}
	sort.Strings(groups)
	sort.Slice(result, func(i, j int) bool {
		if result[i].WorkflowID == result[j].WorkflowID {
			return result[i].Name < result[j].Name
		}
		return result[i].WorkflowID < result[j].WorkflowID
	})
	return result, groups, nil
}
