package fornaxmirror

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var keySuffixPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

var warnSkippedKey = func(key string) {
	fmt.Fprintf(os.Stderr, "warning: skipping Fornax Skill key %s\n", key)
}

type Skill struct {
	ID  string
	Key string
}

type Release struct {
	Version   string
	UpdatedAt int64
}

type Client interface {
	Skills(context.Context, string, string) ([]Skill, error)
	Releases(context.Context, string, string) ([]Release, error)
	Install(context.Context, string, string, string, string) (string, error)
	ListByPrefix(context.Context, string, string) ([]Skill, error)
}

type CLIClient struct {
	Executable string
}

const skillPageSize = 100
const maxSkillPages = 100

func DefaultClient() CLIClient { return CLIClient{Executable: "fornax-cli"} }

func (client CLIClient) Skills(ctx context.Context, workspaceID, key string) ([]Skill, error) {
	listed, err := client.allSkills(ctx, workspaceID, key)
	if err != nil {
		return nil, err
	}
	result := []Skill{}
	for _, skill := range listed {
		if skill.Key != key {
			continue
		}
		if skill.ID == "" || len(result) > 0 {
			return nil, fmt.Errorf("duplicate exact Fornax Skill key")
		}
		result = append(result, skill)
	}
	return result, nil
}

func (client CLIClient) ListByPrefix(ctx context.Context, workspaceID, prefix string) ([]Skill, error) {
	listed, err := client.allSkills(ctx, workspaceID, prefix)
	if err != nil {
		return nil, err
	}
	result := []Skill{}
	seen := map[string]bool{}
	for _, skill := range listed {
		if !strings.HasPrefix(skill.Key, prefix) || skill.Key == prefix {
			continue
		}
		suffix := strings.TrimPrefix(skill.Key, prefix)
		if !keySuffixPattern.MatchString(suffix) {
			warnSkippedKey(skill.Key)
			continue
		}
		if skill.ID == "" {
			return nil, fmt.Errorf("invalid Fornax Skill key %s", skill.Key)
		}
		if seen[skill.Key] {
			return nil, fmt.Errorf("duplicate Fornax Skill key %s", skill.Key)
		}
		seen[skill.Key] = true
		result = append(result, skill)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result, nil
}

func (client CLIClient) allSkills(ctx context.Context, workspaceID, keyword string) ([]Skill, error) {
	result := []Skill{}
	total := -1
	seen := 0
	for page := 1; page <= maxSkillPages; page++ {
		var response struct {
			Skills []struct {
				ID       string `json:"id"`
				SkillKey string `json:"skill_key"`
			} `json:"skills"`
			Total int `json:"total"`
		}
		if err := client.runJSON(ctx, workspaceID, &response, "skill", "list", "--keyword", keyword, "--page-no", strconv.Itoa(page), "--page-size", strconv.Itoa(skillPageSize)); err != nil {
			return nil, err
		}
		if response.Total < 0 || response.Total > skillPageSize*maxSkillPages {
			return nil, fmt.Errorf("invalid Fornax Skill total")
		}
		if total == -1 {
			total = response.Total
		} else if response.Total != total {
			return nil, fmt.Errorf("Fornax Skill total changed during pagination")
		}
		seen += len(response.Skills)
		if seen > total {
			return nil, fmt.Errorf("invalid Fornax Skill total")
		}
		for _, skill := range response.Skills {
			result = append(result, Skill{ID: skill.ID, Key: skill.SkillKey})
		}
		if seen >= total {
			break
		}
		if len(response.Skills) == 0 {
			return nil, fmt.Errorf("Fornax Skill pagination ended before total")
		}
	}
	if seen < total {
		return nil, fmt.Errorf("Fornax Skill pagination limit exceeded")
	}
	return result, nil
}

func (client CLIClient) Releases(ctx context.Context, workspaceID, skillID string) ([]Release, error) {
	var response struct {
		Items []struct {
			Version       string          `json:"version"`
			UpdatedAt     string          `json:"updated_at"`
			ReleaseStatus json.RawMessage `json:"release_status"`
		} `json:"items"`
	}
	if err := client.runJSON(ctx, workspaceID, &response, "skill", "list-releases", "--skill-id", skillID, "--env", "Online", "--feature", "prod"); err != nil {
		return nil, err
	}
	result := []Release{}
	for _, item := range response.Items {
		if !released(item.ReleaseStatus) || item.Version == "" {
			continue
		}
		updatedAt, err := timestamp(item.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("invalid Fornax release timestamp")
		}
		result = append(result, Release{Version: item.Version, UpdatedAt: updatedAt})
	}
	return result, nil
}

func (client CLIClient) Install(ctx context.Context, workspaceID, key, version, directory string) (string, error) {
	query, err := json.Marshal([]map[string]string{{"skill_key": key, "version": version}})
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(directory, 0700); err != nil {
		return "", err
	}
	command := exec.CommandContext(ctx, client.Executable, "skill", "install", "--custom-region", "CN", "--workspace-id", workspaceID, "--skill-query-json", string(query), "--dir", directory)
	if output, failure := command.CombinedOutput(); failure != nil {
		_ = output
		return "", fmt.Errorf("fornax-cli skill install failed")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return "", err
	}
	if len(entries) != 1 {
		return "", fmt.Errorf("Fornax install must return exactly one package root")
	}
	entry := entries[0]
	info, err := entry.Info()
	if err != nil {
		return "", err
	}
	if entry.Type()&os.ModeSymlink != 0 || !info.Mode().IsDir() {
		return "", fmt.Errorf("Fornax install package root must be an ordinary directory")
	}
	return filepath.Join(directory, entry.Name()), nil
}

func (client CLIClient) runJSON(ctx context.Context, workspaceID string, destination any, args ...string) error {
	root, err := os.MkdirTemp("", "fanloop-fornax-query-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)
	args = append(args, "--custom-region", "CN", "--workspace-id", workspaceID, "--format", "json", "--output", root)
	command := exec.CommandContext(ctx, client.Executable, args...)
	if output, failure := command.CombinedOutput(); failure != nil {
		_ = output
		return fmt.Errorf("fornax-cli query failed")
	}
	files := []string{}
	err = filepath.WalkDir(root, func(filename string, entry os.DirEntry, failure error) error {
		if failure != nil {
			return failure
		}
		if entry.Type().IsRegular() {
			files = append(files, filename)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(files) != 1 {
		return fmt.Errorf("fornax-cli query returned %d files", len(files))
	}
	content, err := os.ReadFile(files[0])
	if err != nil {
		return err
	}
	if err = json.Unmarshal(content, destination); err != nil {
		return fmt.Errorf("invalid fornax-cli response")
	}
	return nil
}

func released(raw json.RawMessage) bool {
	value := strings.TrimSpace(string(raw))
	if value == "1" || value == `"released"` {
		return true
	}
	return false
}

func timestamp(value string) (int64, error) {
	if number, err := strconv.ParseInt(value, 10, 64); err == nil {
		return number, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return 0, err
	}
	return parsed.UnixNano(), nil
}
