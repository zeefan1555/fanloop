package verify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var businessDirectories = []string{"flow", "output", "trace", "card"}

func (runtime Runtime) captureSnapshot(ctx context.Context, root, destination string, env []string) ([]string, error) {
	if err := createEmptyPrivateDirectory(destination); err != nil {
		return nil, err
	}
	observations := []struct {
		name string
		args []string
	}{
		{name: "flow-status", args: []string{"flow", "status", "--root", root}},
		{name: "trace-status", args: []string{"trace", "status", "--root", root}},
		{name: "trace-render", args: []string{"trace", "render", "--root", root, "--dry-run"}},
		{name: "card-current", args: []string{"card", "render", "--root", root, "--view", "current", "--format", "markdown", "--dry-run"}},
		{name: "card-panorama", args: []string{"card", "render", "--root", root, "--view", "panorama", "--format", "lark-json", "--dry-run"}},
	}
	for _, observation := range observations {
		result, err := runtime.run(ctx, env, observation.name, observation.args...)
		if err != nil {
			return nil, err
		}
		if err := writePrivate(filepath.Join(destination, "public", observation.name+".stdout.json"), result.Stdout); err != nil {
			return nil, err
		}
		if err := writePrivate(filepath.Join(destination, "public", observation.name+".stderr.txt"), result.Stderr); err != nil {
			return nil, err
		}
		if result.ExitCode != 0 {
			return nil, commandFailure(result)
		}
	}
	if err := copyTree(filepath.Join(root, ".fanloop"), filepath.Join(destination, "files", ".fanloop")); err != nil {
		return nil, err
	}
	files, err := regularFiles(destination)
	if err != nil {
		return nil, err
	}
	if err := writeJSON(filepath.Join(destination, "inventory.json"), files); err != nil {
		return nil, err
	}
	return append(files, "inventory.json"), nil
}

func businessFingerprint(root string) (map[string]string, error) {
	result := map[string]string{}
	for _, directory := range businessDirectories {
		base := filepath.Join(root, ".fanloop", directory)
		if _, err := os.Lstat(base); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		}
		if err := filepath.WalkDir(base, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("business state contains symlink %s", path)
			}
			if !entry.Type().IsRegular() {
				return nil
			}
			digest, err := fileSHA256(path)
			if err != nil {
				return err
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			result[filepath.ToSlash(relative)] = digest
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func copyTree(source, destination string) error {
	if _, err := os.Lstat(source); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to copy symlink %s", path)
		}
		if entry.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			input.Close()
			return err
		}
		output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			input.Close()
			return err
		}
		_, copyErr := io.Copy(output, input)
		inputErr := input.Close()
		closeErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		if inputErr != nil {
			return inputErr
		}
		return closeErr
	})
}

func regularFiles(root string) ([]string, error) {
	files := []string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type().IsRegular() {
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(relative))
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func createEmptyPrivateDirectory(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("destination must be absolute")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		return fmt.Errorf("create snapshot directory: %w", err)
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func safeAbsolute(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("path must be absolute")
	}
	clean := filepath.Clean(path)
	if clean == string(filepath.Separator) || strings.TrimSpace(clean) == "" {
		return "", fmt.Errorf("path must not be the filesystem root")
	}
	return clean, nil
}
