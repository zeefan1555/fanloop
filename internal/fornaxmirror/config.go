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

type Config struct {
	WorkspaceID string            `json:"workspace_id"`
	Workflows   map[string]string `json:"workflows"`
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
	if !workspaceIDPattern.MatchString(value.WorkspaceID) {
		return Config{}, fmt.Errorf("invalid Fornax workspace_id")
	}
	if len(value.Workflows) == 0 {
		return Config{}, fmt.Errorf("Fornax configuration contains no Workflows")
	}
	for id, key := range value.Workflows {
		expected, err := SkillKey(id)
		if err != nil {
			return Config{}, err
		}
		if key != expected {
			return Config{}, fmt.Errorf("Workflow %q key is %q, want %q", id, key, expected)
		}
	}
	return value, nil
}

func SkillKey(workflowID string) (string, error) {
	if !workflowIDPattern.MatchString(workflowID) {
		return "", fmt.Errorf("invalid Workflow ID %q", workflowID)
	}
	return SkillKeyPrefix + strings.ReplaceAll(workflowID, "-", "_"), nil
}

func (config Config) WorkflowIDs() []string {
	result := make([]string, 0, len(config.Workflows))
	for id := range config.Workflows {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}
