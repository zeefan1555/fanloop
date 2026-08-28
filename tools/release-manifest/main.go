package main

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	pathpkg "path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ulikunitz/xz"
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
	dist := flag.String("dist", "dist", "GoReleaser output directory")
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
		Skills: []*release.Skill{}, Workflows: []*release.Workflow{}, Assets: []*release.Asset{},
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
	selectorPath := filepath.Join(source, "skills", "common", "fanloop-workflow-selector", "SKILL.md")
	if err := validateSelectorSkillRoutes(selectorPath, manifest); err != nil {
		return manifest, err
	}

	for _, target := range []struct{ os, arch string }{
		{"darwin", "amd64"}, {"darwin", "arm64"}, {"linux", "amd64"}, {"linux", "arm64"},
	} {
		sourceName := fmt.Sprintf("fanloop-%s-%s-%s.tar", version, target.os, target.arch)
		name := fmt.Sprintf("fanloop-%s-%s-%s.tar.xz", version, target.os, target.arch)
		path := filepath.Join(dist, name)
		if err := compressArchive(filepath.Join(dist, sourceName), path); err != nil {
			return manifest, err
		}
		archiveDigest, err := release.FileDigest(path)
		if err != nil {
			return manifest, err
		}
		binaryDigest, err := verifyArchive(path, manifest)
		if err != nil {
			return manifest, err
		}
		manifest.Assets = append(manifest.Assets, &release.Asset{
			Os: target.os, Arch: target.arch, File: name, Sha256: archiveDigest, BinarySha256: binaryDigest,
		})
	}
	if err := manifest.Validate(); err != nil {
		return manifest, err
	}
	return manifest, nil
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
		parts := strings.Split(skill.Path, "/")
		if len(parts) != 3 || parts[0] != "skills" {
			return fmt.Errorf("Skill %q uses invalid group path %q", skill.Name, skill.Path)
		}
		group := parts[1]
		if group == "common" {
			continue
		}
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
				common := strings.HasPrefix(path, "skills/common/")
				owned := strings.HasPrefix(path, "skills/"+item.Workflow.ID+"/")
				if !common && !owned {
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

const (
	selectorRoutesStart = "<!-- fanloop-selector-routes:start -->"
	selectorRoutesEnd   = "<!-- fanloop-selector-routes:end -->"
)

func validateSelectorSkillRoutes(path string, manifest release.Manifest) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read Workflow selector: %w", err)
	}
	text := string(content)
	if strings.Count(text, selectorRoutesStart) != 1 || strings.Count(text, selectorRoutesEnd) != 1 {
		return fmt.Errorf("Workflow selector must contain exactly one embedded routes block")
	}
	start := strings.Index(text, selectorRoutesStart) + len(selectorRoutesStart)
	end := strings.Index(text[start:], selectorRoutesEnd)
	if end < 0 {
		return fmt.Errorf("Workflow selector routes block is not closed")
	}
	block := strings.TrimSpace(text[start : start+end])
	const fenceStart = "```yaml\n"
	const fenceEnd = "\n```"
	if !strings.HasPrefix(block, fenceStart) || !strings.HasSuffix(block, fenceEnd) {
		return fmt.Errorf("Workflow selector routes block must contain one yaml fence")
	}
	routeYAML := strings.TrimSuffix(strings.TrimPrefix(block, fenceStart), fenceEnd)
	var routes selectorRoutes
	decoder := yaml.NewDecoder(strings.NewReader(routeYAML))
	decoder.KnownFields(true)
	if err := decoder.Decode(&routes); err != nil {
		return fmt.Errorf("decode Workflow selector: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("Workflow selector routes block must contain exactly one YAML document")
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
	paths := []string{}
	err := filepath.WalkDir(filepath.Join(source, "skills"), func(path string, entry os.DirEntry, walkErr error) error {
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

func compressArchive(source, destination string) (err error) {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.CreateTemp(filepath.Dir(destination), "."+filepath.Base(destination)+".*")
	if err != nil {
		return err
	}
	temporary := output.Name()
	defer os.Remove(temporary)
	command := exec.Command("xz", "-6", "--stdout")
	command.Stdin = input
	command.Stdout = output
	var stderr strings.Builder
	command.Stderr = &stderr
	commandErr := command.Run()
	closeErr := output.Close()
	if commandErr != nil {
		return fmt.Errorf("compress %s: %w: %s", source, commandErr, strings.TrimSpace(stderr.String()))
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(temporary, destination)
}

type archivedFile struct {
	digest  string
	content []byte
}

func verifyArchive(archivePath string, manifest release.Manifest) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	xzReader, err := xz.ReaderConfig{DictCap: release.ArchiveXZDictionarySize}.NewReader(file)
	if err != nil {
		return "", err
	}
	reader := tar.NewReader(xzReader)
	files := map[string]archivedFile{}
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		name := strings.TrimSuffix(strings.TrimPrefix(filepath.ToSlash(header.Name), "./"), "/")
		if name == "" || name == "." {
			continue
		}
		if pathpkg.IsAbs(name) || pathpkg.Clean(name) != name || name == ".." || strings.HasPrefix(name, "../") {
			return "", fmt.Errorf("%s contains unsafe path %q", archivePath, header.Name)
		}
		if header.Typeflag == tar.TypeDir {
			continue
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			return "", fmt.Errorf("%s contains unsupported entry %q", archivePath, name)
		}
		if _, duplicate := files[name]; duplicate {
			return "", fmt.Errorf("%s contains duplicate entry %q", archivePath, name)
		}
		content, err := io.ReadAll(reader)
		if err != nil {
			return "", err
		}
		hash := sha256.Sum256(content)
		files[name] = archivedFile{digest: "sha256:" + hex.EncodeToString(hash[:]), content: content}
	}
	binary, ok := files["bin/fanloop"]
	if !ok {
		return "", fmt.Errorf("%s does not contain bin/fanloop", archivePath)
	}
	binaryDigest := binary.digest
	wantedSkills := map[string]bool{}
	for _, skill := range manifest.Skills {
		wantedSkills[skill.Path] = true
		digest, err := archivedDirectoryDigest(files, skill.Path)
		if err != nil {
			return "", fmt.Errorf("%s: %w", archivePath, err)
		}
		if digest != skill.Sha256 {
			return "", fmt.Errorf("%s Skill %s checksum mismatch", archivePath, skill.Path)
		}
	}
	for name := range files {
		if !strings.HasPrefix(name, "skills/") {
			continue
		}
		manifested := false
		for root := range wantedSkills {
			if strings.HasPrefix(name, root+"/") {
				manifested = true
				break
			}
		}
		if !manifested {
			return "", fmt.Errorf("%s contains unmanifested Skill %q", archivePath, name)
		}
	}
	wantedWorkflows := map[string]bool{}
	for _, item := range manifest.Workflows {
		wantedWorkflows[item.Path] = true
		allowedFiles := map[string]bool{}
		for _, name := range workflow.BundleFileNames() {
			allowedFiles[name] = true
		}
		prefix := item.Path + "/"
		for name := range files {
			if strings.HasPrefix(name, prefix) && !allowedFiles[strings.TrimPrefix(name, prefix)] {
				return "", fmt.Errorf("%s Workflow %s contains unexpected file %s", archivePath, item.Path, name)
			}
		}
		bundle := map[string]archivedFile{}
		for _, name := range workflow.BundleFileNames() {
			file, ok := files[item.Path+"/"+name]
			if !ok {
				return "", fmt.Errorf("%s Workflow %s is incomplete", archivePath, item.Path)
			}
			bundle[name] = file
		}
		loaded, err := workflow.DecodeBundle(
			bundle["workflow.yaml"].content,
			bundle["flow.yaml"].content,
			bundle["condition.yaml"].content,
			bundle["loop.yaml"].content,
			bundle["prompt.yaml"].content,
		)
		if err != nil || loaded.Ref.ID != item.Id || loaded.Ref.Digest != item.Sha256 {
			return "", fmt.Errorf("%s Workflow %s checksum mismatch or invalid Bundle", archivePath, item.Path)
		}
	}
	for name := range files {
		if strings.HasPrefix(name, "workflows/") && strings.HasSuffix(name, "/workflow.yaml") && !wantedWorkflows[strings.TrimSuffix(name, "/workflow.yaml")] {
			return "", fmt.Errorf("%s contains unmanifested Workflow %q", archivePath, name)
		}
	}
	return binaryDigest, nil
}

func archivedDirectoryDigest(files map[string]archivedFile, root string) (string, error) {
	prefix := strings.TrimSuffix(root, "/") + "/"
	paths := []string{}
	for name := range files {
		if strings.HasPrefix(name, prefix) {
			paths = append(paths, strings.TrimPrefix(name, prefix))
		}
	}
	if len(paths) == 0 {
		return "", fmt.Errorf("missing %s", root)
	}
	sort.Strings(paths)
	hash := sha256.New()
	for _, relative := range paths {
		_, _ = hash.Write([]byte(relative))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(strings.TrimPrefix(files[prefix+relative].digest, "sha256:")))
		_, _ = hash.Write([]byte{'\n'})
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
