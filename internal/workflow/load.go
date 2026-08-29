package workflow

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/zeefan1555/commonloop/internal/idl/yamlidl"
	"go.yaml.in/yaml/v3"
)

var workflowIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var configIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
var ErrInvalidSelector = errors.New("invalid workflow selector")

func Load(id string) (Loaded, error) {
	if !workflowIDPattern.MatchString(id) {
		return Loaded{}, fmt.Errorf("invalid workflow id %q", id)
	}
	return LoadRef(Ref{ID: id})
}

func LoadRef(ref Ref) (Loaded, error) {
	if !workflowIDPattern.MatchString(ref.ID) {
		return Loaded{}, fmt.Errorf("invalid workflow reference")
	}
	source, err := sourceForBundle(ref.ID)
	if err != nil {
		return Loaded{}, err
	}
	loaded, err := loadBundle(source, ref.ID)
	if err != nil {
		return Loaded{}, err
	}
	if loaded.Ref.ID != ref.ID {
		return Loaded{}, fmt.Errorf("workflow path does not match its id")
	}
	if ref.Digest != "" && loaded.Ref.Digest != ref.Digest {
		return Loaded{}, fmt.Errorf("workflow digest mismatch")
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
	result := []Loaded{}
	for _, source := range sources() {
		paths, err := fs.Glob(source, "*/workflow.yaml")
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			id := strings.SplitN(path, "/", 2)[0]
			loaded, err := LoadRef(Ref{ID: id})
			if err != nil {
				return nil, err
			}
			result = append(result, loaded)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Ref.ID < result[j].Ref.ID
	})
	return result, nil
}

func sourceForBundle(root string) (fs.FS, error) {
	var found fs.FS
	for _, source := range sources() {
		if err := requireExactBundleFiles(source, root); err == nil {
			if found != nil {
				return nil, fmt.Errorf("duplicate workflow bundle %q", root)
			}
			found = source
		}
	}
	if found == nil {
		return nil, fmt.Errorf("workflow %q not found: %w", root, fs.ErrNotExist)
	}
	return found, nil
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
			return Loaded{}, fmt.Errorf("%s: %w", item.name, err)
		}
		if err := item.value.IsValid(); err != nil {
			return Loaded{}, fmt.Errorf("%s: %w", item.name, err)
		}
	}
	normalizeYAMLOptionalDefaults(&flow, &condition)
	value, err := normalizeYAMLDocuments(&authoring, &flow, &condition, &loop, &prompt)
	if err != nil {
		return Loaded{}, err
	}
	if err := ValidateBundle(&value, flow.SchemaVersion, condition.SchemaVersion, loop.SchemaVersion, prompt.SchemaVersion); err != nil {
		return Loaded{}, err
	}
	digest, err := bundleDigest(authoring, flow, condition, loop, prompt)
	if err != nil {
		return Loaded{}, err
	}
	return Loaded{Workflow: value, Ref: Ref{ID: value.ID, Digest: digest}}, nil
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

func bundleDigest(authoring yamlidl.WorkflowDocument, flow yamlidl.FlowDocument, condition yamlidl.ConditionDocument, loop yamlidl.LoopDocument, prompt yamlidl.PromptDocument) (string, error) {
	canonical := struct {
		Workflow  yamlidl.WorkflowDocument  `json:"workflow"`
		Flow      yamlidl.FlowDocument      `json:"flow"`
		Condition yamlidl.ConditionDocument `json:"condition"`
		Loop      yamlidl.LoopDocument      `json:"loop"`
		Prompt    yamlidl.PromptDocument    `json:"prompt"`
	}{authoring, flow, condition, loop, prompt}
	content, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func requireExactBundleFiles(files fs.FS, root string) error {
	entries, err := fs.ReadDir(files, root)
	if err != nil {
		return err
	}
	missing := map[string]bool{}
	for _, name := range BundleFileNames() {
		missing[name] = true
	}
	for _, entry := range entries {
		if !missing[entry.Name()] {
			return fmt.Errorf("unexpected Workflow Bundle entry %s", entry.Name())
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("Workflow Bundle entry %s is not a regular file", entry.Name())
		}
		delete(missing, entry.Name())
	}
	for _, name := range BundleFileNames() {
		if missing[name] {
			return fmt.Errorf("missing Workflow Bundle file %s: %w", name, fs.ErrNotExist)
		}
	}
	return nil
}

func BundleFileNames() []string {
	return []string{"workflow.yaml", "flow.yaml", "condition.yaml", "loop.yaml", "prompt.yaml"}
}
