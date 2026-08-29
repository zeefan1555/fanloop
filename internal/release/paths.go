package release

import (
	"os"
	"path/filepath"
)

type SkillRoots struct {
	Codex  string
	Agent  string
	Trae   string
	Claude string
}

func DefaultSkillRoots() (SkillRoots, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return SkillRoots{}, err
	}
	return SkillRoots{
		Codex:  environmentPath("COMMONLOOP_CODEX_SKILLS_ROOT", filepath.Join(home, ".codex", "skills")),
		Agent:  environmentPath("COMMONLOOP_AGENT_SKILLS_ROOT", filepath.Join(home, ".agents", "skills")),
		Trae:   environmentPath("COMMONLOOP_TRAE_SKILLS_ROOT", filepath.Join(home, ".trae", "skills")),
		Claude: environmentPath("COMMONLOOP_CLAUDE_SKILLS_ROOT", filepath.Join(home, ".claude", "skills")),
	}, nil
}

func (roots SkillRoots) Values() []string {
	return []string{roots.Codex, roots.Agent, roots.Trae, roots.Claude}
}

func DefaultDataRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if value := os.Getenv("COMMONLOOP_DATA_HOME"); value != "" {
		return filepath.Clean(value), nil
	}
	return filepath.Join(home, ".commonloop"), nil
}

func environmentPath(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return filepath.Clean(value)
	}
	return fallback
}
