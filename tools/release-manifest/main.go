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
	"github.com/zeefan1555/fanloop/internal/skillconfig"
	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/workflow"
	"go.yaml.in/yaml/v3"
)

func main() {
	version := flag.String("version", "", "release version")
	source := flag.String("source", ".", "repository root")
	dist := flag.String("dist", "dist", "local build directory containing bin/fanloop, entrypoints and workflows")
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
	entrypoint, err := discoverEntrypoint(source, version)
	if err != nil {
		return manifest, err
	}
	manifest.Skills = []*release.Skill{entrypoint}

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
	if _, err := skillconfig.Validate(source, loadedWorkflows); err != nil {
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

func discoverEntrypoint(source, version string) (*release.Skill, error) {
	entrypoint := filepath.Join(source, release.ExposedSkillPath, "SKILL.md")
	info, err := os.Stat(entrypoint)
	if err != nil {
		return nil, fmt.Errorf("find exposed Skill: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("exposed Skill is not a regular file: %s", entrypoint)
	}
	root := filepath.Dir(entrypoint)
	digest, err := release.DirectoryDigest(root)
	if err != nil {
		return nil, err
	}
	return &release.Skill{Name: release.ExposedSkillName, Version: version, Path: release.ExposedSkillPath, Sha256: digest}, nil
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
