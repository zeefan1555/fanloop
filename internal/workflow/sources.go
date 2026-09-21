package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func DefaultRoot() (string, error) {
	if dataRoot := os.Getenv("FANLOOP_DATA_HOME"); dataRoot != "" {
		if !filepath.IsAbs(dataRoot) {
			return "", fmt.Errorf("FANLOOP_DATA_HOME must be absolute")
		}
		return filepath.Join(filepath.Clean(dataRoot), "current", "workflows"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dataRoot := filepath.Join(home, ".fanloop")
	executable, executableErr := os.Executable()
	if executableErr == nil {
		resolved, resolveErr := filepath.EvalSymlinks(executable)
		if resolveErr == nil && isWithin(filepath.Join(dataRoot, "releases"), resolved) {
			return filepath.Join(dataRoot, "current", "workflows"), nil
		}
	}
	{
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return "", fmt.Errorf("locate source Workflows")
		}
		return filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", "workflows")), nil
	}
}

func isWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != "." && relative != ".." && !filepath.IsAbs(relative) && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
