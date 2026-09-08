package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zeefan1555/fanloop/internal/idl/opsidl"
	"github.com/zeefan1555/fanloop/internal/idl/releaseidl"
	"github.com/zeefan1555/fanloop/internal/release"
	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/workflow"
	"go.yaml.in/yaml/v3"
)

func main() {
	version := flag.String("version", "", "release version")
	source := flag.String("source", ".", "repository root")
	dist := flag.String("dist", "dist", "local build directory containing bin/fanloop, entrypoints, skills and workflows")
	output := flag.String("output", "release.json", "manifest output path")
	flag.Parse()
	if *version == "" {
		fatal(fmt.Errorf("--version is required"))
	}
	manifest, err := build(*version, *source, *dist)
	if err != nil {
		fatal(err)
	}
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(*output, append(content, '\n'), 0o600); err != nil {
		fatal(err)
	}
}

func build(version, source, dist string) (release.Manifest, error) {
	manifest := release.Manifest{
		SchemaVersion: releaseidl.RELEASE_MANIFEST_SCHEMA_VERSION, ReleaseVersion: version, Cli: &release.CLIRelease{Version: version},
		StateSchema: &opsidl.StateSchemaSupport{
			ReadVersions: []int32{int32(state.CurrentStateSchemaVersion)}, WriteVersion: int32(state.CurrentStateSchemaVersion),
		},
		Skills: []*release.Skill{}, Workflows: []*release.Workflow{},
	}
	if err := validateWorkflowSkillDirectories(source); err != nil {
		return manifest, err
	}
	skills, err := discoverSkills(source, version)
	if err != nil {
		return manifest, err
	}
	manifest.Skills = skills

	workflowPaths, err := filepath.Glob(filepath.Join(source, "workflows", "*", "workflow.yaml"))
	if err != nil {
		return manifest, fmt.Errorf("find Workflows: %w", err)
	}
	if len(workflowPaths) == 0 {
		return manifest, fmt.Errorf("no Workflows found")
	}
	sort.Strings(workflowPaths)
	loadedWorkflows := make([]workflow.Loaded, 0, len(workflowPaths))
	for _, path := range workflowPaths {
		root := filepath.Dir(path)
		relative, err := filepath.Rel(source, root)
		if err != nil {
			return manifest, err
		}
		loaded, err := workflow.LoadDirectory(root)
		if err != nil {
			return manifest, fmt.Errorf("%s: %w", root, err)
		}
		manifest.Workflows = append(manifest.Workflows, &release.Workflow{
			Id: loaded.Ref.ID, Path: filepath.ToSlash(relative), Sha256: loaded.Ref.Digest,
		})
		loadedWorkflows = append(loadedWorkflows, loaded)
	}
	if err := validateWorkflowSkillBindings(manifest, loadedWorkflows); err != nil {
		return manifest, err
	}
	selectorPath := filepath.Join(source, "entrypoints", release.ExposedSkillName, "routes.yaml")
	if err := validateSelectorRoutes(selectorPath, manifest); err != nil {
		return manifest, err
	}

	binaryDigest, err := verifyDirectory(dist, manifest)
	if err != nil {
		return manifest, err
	}
	manifest.Cli.BinarySha256 = binaryDigest
	if err := manifest.Validate(); err != nil {
		return manifest, err
	}
	return manifest, nil
}

func validateWorkflowSkillDirectories(source string) error {
	groups := func(root string) ([]string, error) {
		entries, err := os.ReadDir(filepath.Join(source, root))
		if err != nil {
			return nil, err
		}
		result := []string{}
		for _, entry := range entries {
			if entry.IsDir() {
				result = append(result, entry.Name())
			}
		}
		sort.Strings(result)
		return result, nil
	}
	workflows, err := groups("workflows")
	if err != nil {
		return fmt.Errorf("list Workflow directories: %w", err)
	}
	skills, err := groups("skills")
	if err != nil {
		return fmt.Errorf("list Skill directories: %w", err)
	}
	if strings.Join(workflows, "\x00") != strings.Join(skills, "\x00") {
		return fmt.Errorf("Workflow and Skill directories must match: workflows=%v skills=%v", workflows, skills)
	}
	return nil
}

func validateWorkflowSkillBindings(manifest release.Manifest, loaded []workflow.Loaded) error {
	skills := make(map[string]string, len(manifest.Skills))
	workflowIDs := make(map[string]bool, len(manifest.Workflows))
	workflowGroups := make(map[string]bool, len(manifest.Workflows))
	for _, item := range manifest.Workflows {
		workflowIDs[item.Id] = true
	}
	for _, skill := range manifest.Skills {
		if _, exists := skills[skill.Name]; exists {
			return fmt.Errorf("duplicate Skill %q", skill.Name)
		}
		skills[skill.Name] = skill.Path
		if skill.Name == release.ExposedSkillName {
			if skill.Path != release.ExposedSkillPath {
				return fmt.Errorf("exposed Skill %q uses invalid path %q", skill.Name, skill.Path)
			}
			continue
		}
		parts := strings.Split(skill.Path, "/")
		if len(parts) != 3 || parts[0] != "skills" {
			return fmt.Errorf("Skill %q uses invalid group path %q", skill.Name, skill.Path)
		}
		group := parts[1]
		if !workflowIDs[group] {
			return fmt.Errorf("Skill %q uses unknown Workflow group %q", skill.Name, group)
		}
		workflowGroups[group] = true
	}
	for workflowID := range workflowIDs {
		if !workflowGroups[workflowID] {
			return fmt.Errorf("Workflow %q is missing matching skills/%s group", workflowID, workflowID)
		}
	}
	for _, item := range loaded {
		for promptID, prompt := range item.Workflow.Prompts {
			for _, binding := range prompt.Skills {
				path, ok := skills[binding.ID]
				if !ok {
					return fmt.Errorf("Workflow %s prompt %s uses unknown Skill %q", item.Workflow.ID, promptID, binding.ID)
				}
				owned := strings.HasPrefix(path, "skills/"+item.Workflow.ID+"/")
				if !owned {
					return fmt.Errorf("Workflow %s cannot use Skill %q from %s", item.Workflow.ID, binding.ID, path)
				}
			}
		}
	}
	return nil
}

type selectorRoutes struct {
	SchemaVersion int32 `yaml:"schema_version"`
	Scenarios     map[string]struct {
		Workflow    string `yaml:"workflow"`
		Description string `yaml:"description"`
	} `yaml:"scenarios"`
}

func validateSelectorRoutes(path string, manifest release.Manifest) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read Workflow selector: %w", err)
	}
	var routes selectorRoutes
	decoder := yaml.NewDecoder(strings.NewReader(string(content)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&routes); err != nil {
		return fmt.Errorf("decode Workflow selector: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("Workflow selector must contain exactly one YAML document")
	}
	if routes.SchemaVersion != 2 || len(routes.Scenarios) == 0 {
		return fmt.Errorf("invalid Workflow selector schema or scenarios")
	}
	wanted := map[string]bool{}
	for _, item := range manifest.Workflows {
		wanted[item.Id] = true
	}
	covered := map[string]bool{}
	for scenario, rule := range routes.Scenarios {
		if scenario == "" || rule.Workflow == "" || rule.Description == "" {
			return fmt.Errorf("invalid Workflow selector scenario rule")
		}
		if !wanted[rule.Workflow] {
			return fmt.Errorf("Workflow selector uses unknown Workflow %q", rule.Workflow)
		}
		covered[rule.Workflow] = true
	}
	for workflowID := range wanted {
		if !covered[workflowID] {
			return fmt.Errorf("Workflow selector has no scenario for Workflow %q", workflowID)
		}
	}
	return nil
}

func discoverSkills(source, version string) ([]*release.Skill, error) {
	entrypoint := filepath.Join(source, release.ExposedSkillPath, "SKILL.md")
	info, err := os.Stat(entrypoint)
	if err != nil {
		return nil, fmt.Errorf("find exposed Skill: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("exposed Skill is not a regular file: %s", entrypoint)
	}
	paths := []string{entrypoint}
	err = filepath.WalkDir(filepath.Join(source, "skills"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || entry.Name() != "SKILL.md" {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if parts := strings.Split(filepath.ToSlash(relative), "/"); len(parts) != 4 || parts[0] != "skills" {
			return fmt.Errorf("Skill entry must use skills/<workflow-id>/<skill-id>/SKILL.md: %s", relative)
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("find Skills: %w", err)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no Skills found")
	}
	sort.Strings(paths)
	result := make([]*release.Skill, 0, len(paths))
	for _, skillFile := range paths {
		root := filepath.Dir(skillFile)
		relative, err := filepath.Rel(source, root)
		if err != nil {
			return nil, err
		}
		digest, err := release.DirectoryDigest(root)
		if err != nil {
			return nil, err
		}
		result = append(result, &release.Skill{
			Name: filepath.Base(root), Version: version, Path: filepath.ToSlash(relative), Sha256: digest,
		})
	}
	return result, nil
}

// Verify the copied build against the source manifest before writing release.json.
func verifyDirectory(root string, manifest release.Manifest) (string, error) {
	allowedFiles := map[string]bool{"bin/fanloop": true, "release.json": true}
	skillRoots := make([]string, 0, len(manifest.Skills))
	for _, skill := range manifest.Skills {
		skillRoots = append(skillRoots, skill.Path+"/")
	}
	for _, item := range manifest.Workflows {
		for _, name := range workflow.BundleFileNames() {
			allowedFiles[item.Path+"/"+name] = true
		}
	}
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("local build contains unsupported entry %s", path)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(relative)
		if allowedFiles[name] {
			return nil
		}
		for _, prefix := range skillRoots {
			if strings.HasPrefix(name, prefix) {
				return nil
			}
		}
		return fmt.Errorf("local build contains unmanifested file %s", name)
	}); err != nil {
		return "", err
	}
	binaryPath := filepath.Join(root, "bin", "fanloop")
	binary, err := os.Stat(binaryPath)
	if err != nil {
		return "", err
	}
	if !binary.Mode().IsRegular() || binary.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("%s is not an executable regular file", binaryPath)
	}
	for _, skill := range manifest.Skills {
		digest, err := release.DirectoryDigest(filepath.Join(root, filepath.FromSlash(skill.Path)))
		if err != nil {
			return "", err
		}
		if digest != skill.Sha256 {
			return "", fmt.Errorf("local build Skill %s checksum mismatch", skill.Path)
		}
	}
	for _, item := range manifest.Workflows {
		loaded, err := workflow.LoadDirectory(filepath.Join(root, filepath.FromSlash(item.Path)))
		if err != nil {
			return "", fmt.Errorf("local build Workflow %s: %w", item.Path, err)
		}
		if loaded.Ref.ID != item.Id || loaded.Ref.Digest != item.Sha256 {
			return "", fmt.Errorf("local build Workflow %s checksum mismatch", item.Path)
		}
	}
	return release.FileDigest(binaryPath)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
