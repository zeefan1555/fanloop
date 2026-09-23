package workflow

import (
	"fmt"
	"os"
	"path/filepath"
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
	return filepath.Join(home, ".fanloop", "current", "workflows"), nil
}

func isWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != "." && relative != ".." && !filepath.IsAbs(relative) && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
