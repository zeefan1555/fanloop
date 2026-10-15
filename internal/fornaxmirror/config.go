package fornaxmirror

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
)

const SkillKeyPrefix = "fanloop.workflow."

var workflowIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$`)
var workspaceIDPattern = regexp.MustCompile(`^[1-9][0-9]*$`)
var keyPrefixPattern = regexp.MustCompile(`^[a-z][a-z0-9]*\.workflow\.$`)

type Access string

const (
	AccessReadWrite Access = "read_write"
	AccessReadOnly  Access = "read_only"
)

// Source is one Fornax workspace scanned by key prefix.
type Source struct {
	WorkspaceID string `json:"workspace_id"`
	KeyPrefix   string `json:"key_prefix"`
	Access      Access `json:"access"`
}

// Config is ~/.fanloop/fornax.json. It stores workspaces and prefixes only.
type Config struct {
	Sources []Source `json:"sources"`
}

// PackageClaim is one released package before duplicate workflow ids are resolved.
type PackageClaim struct {
	WorkflowID  string
	Key         string
	Version     string
	Access      Access
	SourceIndex int
}

func LoadConfig(path string) (Config, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return Config{}, err
	}
	if !info.Mode().IsRegular() {
		return Config{}, fmt.Errorf("Fornax configuration must be a regular file")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	return DecodeConfig(content)
}

func DecodeConfig(content []byte) (Config, error) {
	var value Config
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return Config{}, fmt.Errorf("invalid Fornax configuration: %w", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("trailing JSON value")
		}
		return Config{}, fmt.Errorf("invalid Fornax configuration: %w", err)
	}
	if len(value.Sources) == 0 {
		return Config{}, fmt.Errorf("Fornax configuration contains no sources")
	}
	seen := map[string]bool{}
	for _, source := range value.Sources {
		if !workspaceIDPattern.MatchString(source.WorkspaceID) {
			return Config{}, fmt.Errorf("invalid Fornax workspace_id")
		}
		if !keyPrefixPattern.MatchString(source.KeyPrefix) {
			return Config{}, fmt.Errorf("invalid Fornax key_prefix %q", source.KeyPrefix)
		}
		if source.Access != AccessReadWrite && source.Access != AccessReadOnly {
			return Config{}, fmt.Errorf("invalid Fornax source access %q", source.Access)
		}
		identity := source.WorkspaceID + "\n" + source.KeyPrefix
		if seen[identity] {
			return Config{}, fmt.Errorf("duplicate Fornax source %s %s", source.WorkspaceID, source.KeyPrefix)
		}
		seen[identity] = true
	}
	return value, nil
}

// ReadWriteSource returns the single personal source that may be edited.
func (config Config) ReadWriteSource() (Source, error) {
	var found *Source
	for index := range config.Sources {
		if config.Sources[index].Access != AccessReadWrite {
			continue
		}
		if found != nil {
			return Source{}, fmt.Errorf("multiple read_write Fornax sources")
		}
		copy := config.Sources[index]
		found = &copy
	}
	if found == nil {
		return Source{}, fmt.Errorf("Fornax configuration has no read_write source")
	}
	return *found, nil
}

func SkillKey(workflowID string) (string, error) {
	return KeyForPrefix(SkillKeyPrefix, workflowID)
}

func KeyForPrefix(prefix, workflowID string) (string, error) {
	if !keyPrefixPattern.MatchString(prefix) {
		return "", fmt.Errorf("invalid Fornax key_prefix %q", prefix)
	}
	if !workflowIDPattern.MatchString(workflowID) {
		return "", fmt.Errorf("invalid Workflow ID %q", workflowID)
	}
	return prefix + strings.ReplaceAll(workflowID, "-", "_"), nil
}

// SelectClaims keeps one package per workflow id.
// A read_write source wins over a read_only source.
// Two claims in one source, or a tie across sources, reject the whole set.
func SelectClaims(claims []PackageClaim) ([]PackageClaim, error) {
	withinSource := map[string]bool{}
	for _, claim := range claims {
		marker := fmt.Sprintf("%d\n%s", claim.SourceIndex, claim.WorkflowID)
		if withinSource[marker] {
			return nil, fmt.Errorf("workflow id %s appears twice in one source", claim.WorkflowID)
		}
		withinSource[marker] = true
	}
	grouped := map[string][]PackageClaim{}
	order := []string{}
	for _, claim := range claims {
		if _, ok := grouped[claim.WorkflowID]; !ok {
			order = append(order, claim.WorkflowID)
		}
		grouped[claim.WorkflowID] = append(grouped[claim.WorkflowID], claim)
	}
	sort.Strings(order)
	selected := make([]PackageClaim, 0, len(order))
	for _, id := range order {
		group := grouped[id]
		if len(group) == 1 {
			selected = append(selected, group[0])
			continue
		}
		writable := []PackageClaim{}
		for _, claim := range group {
			if claim.Access == AccessReadWrite {
				writable = append(writable, claim)
			}
		}
		if len(writable) != 1 {
			return nil, fmt.Errorf("workflow id %s is claimed by %d sources", id, len(group))
		}
		selected = append(selected, writable[0])
	}
	return selected, nil
}
