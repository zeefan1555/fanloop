package workflow

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/zeefan1555/fanloop/internal/idl/yamlidl"
	"go.yaml.in/yaml/v3"
)

var workflowIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var configIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
var ErrInvalidSelector = errors.New("invalid workflow selector")
var ErrInvalidConfiguration = errors.New("invalid workflow configuration")

func invalidConfiguration(err error) error {
	if errors.Is(err, ErrInvalidConfiguration) {
		return err
	}
	return fmt.Errorf("%w: %v", ErrInvalidConfiguration, err)
}

func Load(id string) (Loaded, error) {
	root, err := DefaultRoot()
	if err != nil {
		return Loaded{}, err
	}
	return LoadFrom(root, id)
}

func LoadFrom(root, id string) (Loaded, error) {
	if !workflowIDPattern.MatchString(id) {
		return Loaded{}, fmt.Errorf("%w %q", ErrInvalidSelector, id)
	}
	return loadFromRef(root, Ref{ID: id})
}

func LoadRef(ref Ref) (Loaded, error) {
	root, err := DefaultRoot()
	if err != nil {
		return Loaded{}, err
	}
	return loadFromRef(root, ref)
}

func loadFromRef(root string, ref Ref) (Loaded, error) {
	if !workflowIDPattern.MatchString(ref.ID) {
		return Loaded{}, fmt.Errorf("invalid workflow reference")
	}
	if err := requireRealWorkflowRoot(root); err != nil {
		return Loaded{}, err
	}
	bundleRoot := filepath.Join(root, ref.ID)
	if err := requireRealDirectory(bundleRoot); err != nil {
		return Loaded{}, fmt.Errorf("Workflow %q: %w", ref.ID, err)
	}
	if err := filepath.WalkDir(bundleRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && !entry.Type().IsRegular() {
			return fmt.Errorf("package entry %s is not a regular file or directory", path)
		}
		return nil
	}); err != nil {
		return Loaded{}, err
	}
	loaded, err := loadBundle(os.DirFS(root), ref.ID)
	if err != nil {
		return Loaded{}, err
	}
	if loaded.Ref.ID != ref.ID {
		return Loaded{}, invalidConfiguration(fmt.Errorf("workflow path does not match its id"))
	}
	loaded.PackageRoot = bundleRoot
	loaded.SkillPaths, err = loadSkillPaths(bundleRoot, loaded.Workflow)
	if err != nil {
		return Loaded{}, err
	}
	return loaded, nil
}

func LoadSelector(selector string) (Loaded, error) {
	if !workflowIDPattern.MatchString(selector) {
		return Loaded{}, fmt.Errorf("%w %q", ErrInvalidSelector, selector)
	}
	return Load(selector)
}

func LoadDirectory(root string) (Loaded, error) {
	if err := requireRealWorkflowRoot(root); err != nil {
		return Loaded{}, err
	}
	if err := requireExactBundleFiles(os.DirFS(root), "."); err != nil {
		return Loaded{}, err
	}
	contents := map[string][]byte{}
	for _, name := range BundleFileNames() {
		content, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return Loaded{}, fmt.Errorf("%s: %w", name, err)
		}
		contents[name] = content
	}
	return DecodeBundle(contents["workflow.yaml"], contents["flow.yaml"], contents["condition.yaml"], contents["loop.yaml"], contents["prompt.yaml"])
}

func List() ([]Loaded, error) {
	root, err := DefaultRoot()
	if err != nil {
		return nil, err
	}
	return ListFrom(root)
}

func ListFrom(root string) ([]Loaded, error) {
	if err := requireRealWorkflowRoot(root); err != nil {
		return nil, err
	}
	result := []Loaded{}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() || !workflowIDPattern.MatchString(entry.Name()) {
			return nil, fmt.Errorf("invalid Workflow directory %q", entry.Name())
		}
		loaded, err := LoadFrom(root, entry.Name())
		if err != nil {
			return nil, err
		}
		result = append(result, loaded)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Ref.ID < result[j].Ref.ID
	})
	return result, nil
}

func requireRealWorkflowRoot(root string) error {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	// The managed source binding is resolved before this check; package roots
	// and their parent source must be real directories.
	for _, path := range []string{filepath.Dir(absolute), absolute} {
		if err := requireRealDirectory(path); err != nil {
			return fmt.Errorf("Workflow root must use real directories: %w", err)
		}
	}
	return nil
}

func requireRealDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is not a real directory", path)
	}
	return nil
}

func loadBundle(source fs.FS, root string) (Loaded, error) {
	if err := requireExactBundleFiles(source, root); err != nil {
		return Loaded{}, fmt.Errorf("%s: %w", root, err)
	}
	contents := map[string][]byte{}
	for _, name := range BundleFileNames() {
		content, err := fs.ReadFile(source, root+"/"+name)
		if err != nil {
			return Loaded{}, fmt.Errorf("%s/%s: %w", root, name, err)
		}
		contents[name] = content
	}
	loaded, err := DecodeBundle(contents["workflow.yaml"], contents["flow.yaml"], contents["condition.yaml"], contents["loop.yaml"], contents["prompt.yaml"])
	if err != nil {
		return Loaded{}, fmt.Errorf("%s: %w", root, err)
	}
	return loaded, nil
}

func DecodeBundle(workflowYAML, flowYAML, conditionYAML, loopYAML, promptYAML []byte) (Loaded, error) {
	var authoring yamlidl.WorkflowDocument
	var flow yamlidl.FlowDocument
	var condition yamlidl.ConditionDocument
	var loop yamlidl.LoopDocument
	var prompt yamlidl.PromptDocument
	for _, item := range []struct {
		name    string
		content []byte
		value   interface{ IsValid() error }
	}{
		{"workflow.yaml", workflowYAML, &authoring},
		{"flow.yaml", flowYAML, &flow},
		{"condition.yaml", conditionYAML, &condition},
		{"loop.yaml", loopYAML, &loop},
		{"prompt.yaml", promptYAML, &prompt},
	} {
		if err := decodeYAML(item.content, item.value); err != nil {
			return Loaded{}, invalidConfiguration(fmt.Errorf("%s: %w", item.name, err))
		}
		if err := item.value.IsValid(); err != nil {
			return Loaded{}, invalidConfiguration(fmt.Errorf("%s: %w", item.name, err))
		}
	}
	normalizeYAMLOptionalDefaults(&flow, &condition)
	value, err := normalizeYAMLDocuments(&authoring, &flow, &condition, &loop, &prompt)
	if err != nil {
		return Loaded{}, invalidConfiguration(err)
	}
	if err := ValidateBundle(&value, flow.SchemaVersion, condition.SchemaVersion, loop.SchemaVersion, prompt.SchemaVersion); err != nil {
		return Loaded{}, invalidConfiguration(err)
	}
	return Loaded{Workflow: value, Ref: Ref{ID: value.ID}}, nil
}

func decodeYAML(content []byte, destination any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing YAML document")
		}
		return err
	}
	return nil
}

func requireExactBundleFiles(files fs.FS, root string) error {
	entries, err := fs.ReadDir(files, root)
	if err != nil {
		return err
	}
	missing := map[string]bool{"SKILL.md": true}
	for _, name := range BundleFileNames() {
		missing[name] = true
	}
	for _, entry := range entries {
		if entry.Name() == "skills" && entry.IsDir() && entry.Type()&fs.ModeSymlink == 0 {
			continue
		}
		if entry.Name() == "trace.yaml" {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("Workflow Bundle entry %s is not a regular file", entry.Name())
			}
			continue
		}
		if !missing[entry.Name()] {
			return invalidConfiguration(fmt.Errorf("unexpected Workflow Bundle entry %s", entry.Name()))
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("Workflow Bundle entry %s is not a regular file", entry.Name())
		}
		if entry.Name() == "SKILL.md" {
			content, err := fs.ReadFile(files, path.Join(root, "SKILL.md"))
			if err != nil || len(bytes.TrimSpace(content)) == 0 {
				return invalidConfiguration(fmt.Errorf("package SKILL.md must not be empty"))
			}
		}
		delete(missing, entry.Name())
	}
	for _, name := range append([]string{"SKILL.md"}, BundleFileNames()...) {
		if !missing[name] {
			continue
		}
		return invalidConfiguration(fmt.Errorf("missing Workflow Bundle file %s: %w", name, fs.ErrNotExist))
	}
	return nil
}

func loadSkillPaths(bundleRoot string, definition Workflow) (map[string]string, error) {
	root := filepath.Join(bundleRoot, "skills")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read Workflow Skills: %w", err)
	}
	paths := make(map[string]string, len(entries))
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() || !workflowIDPattern.MatchString(entry.Name()) {
			return nil, fmt.Errorf("invalid Skill directory %q", entry.Name())
		}
		path := filepath.Join(root, entry.Name(), "SKILL.md")
		info, err := os.Lstat(path)
		if errors.Is(err, fs.ErrNotExist) {
			return nil, invalidConfiguration(fmt.Errorf("Skill %q is missing regular SKILL.md", entry.Name()))
		}
		if err != nil || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("Skill %q is missing regular SKILL.md", entry.Name())
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(string(content)) == "" {
			return nil, invalidConfiguration(fmt.Errorf("Skill %q has empty SKILL.md", entry.Name()))
		}
		paths[entry.Name()] = path
	}
	for _, binding := range allSkillBindings(definition) {
		if paths[binding.ID] == "" {
			return nil, invalidConfiguration(fmt.Errorf("Workflow %s uses unknown Skill %q", definition.ID, binding.ID))
		}
	}
	return paths, nil
}

func allSkillBindings(definition Workflow) []SkillBinding {
	result := append([]SkillBinding(nil), definition.CommonSkills...)
	for _, prompt := range definition.Prompts {
		result = append(result, prompt.Skills...)
	}
	return result
}

func BundleFileNames() []string {
	return []string{"workflow.yaml", "flow.yaml", "condition.yaml", "loop.yaml", "prompt.yaml"}
}
