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
		return ConfigurationRoot(dataRoot)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return ConfigurationRoot(filepath.Join(home, ".fanloop"))
}

func ConfigurationRoot(dataRoot string) (string, error) {
	source, explicit := os.LookupEnv("FANLOOP_CONFIG_SOURCE")
	if !explicit {
		var err error
		source, err = os.Readlink(filepath.Join(dataRoot, "config", "current"))
		if err != nil {
			return "", fmt.Errorf("local configuration binding: %w", err)
		}
	}
	return SourceRoot(source)
}

// SourceRoot accepts one explicit repository; it never searches for alternatives.
func SourceRoot(source string) (string, error) {
	if !filepath.IsAbs(source) || filepath.Clean(source) == string(filepath.Separator) {
		return "", fmt.Errorf("configuration source must be an absolute repository directory")
	}
	source = filepath.Clean(source)
	if err := requireRealDirectory(source); err != nil {
		return "", fmt.Errorf("local configuration source: %w", err)
	}
	root := filepath.Join(source, "skills", "fanloop-workflow")
	if err := requireRealWorkflowRoot(root); err != nil {
		return "", fmt.Errorf("local configuration source: %w", err)
	}
	return filepath.EvalSymlinks(root)
}

func isWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != "." && relative != ".." && !filepath.IsAbs(relative) && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
