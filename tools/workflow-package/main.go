package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/zeefan1555/fanloop/internal/fornaxmirror"
)

func main() {
	if len(os.Args) < 2 {
		fail("usage: workflow-package <build|validate|compare>")
	}
	var err error
	switch os.Args[1] {
	case "build":
		err = build(os.Args[2:])
	case "validate":
		err = validate(os.Args[2:])
	case "compare":
		err = compare(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fail(err.Error())
	}
}

func build(args []string) error {
	flags := flag.NewFlagSet("build", flag.ContinueOnError)
	source := flags.String("source", "", "clean second-brain-skills repository root")
	configPath := flags.String("config", "", "strict credential-free fornax.json")
	version := flags.String("version", "", "stable package SemVer")
	output := flags.String("output", "", "empty output directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *source == "" || *configPath == "" || *version == "" || *output == "" || flags.NArg() != 0 {
		return fmt.Errorf("build requires --source, --config, --version and --output")
	}
	commit, err := cleanCommit(*source)
	if err != nil {
		return err
	}
	config, err := fornaxmirror.LoadConfig(*configPath)
	if err != nil {
		return err
	}
	writable, err := config.ReadWriteSource()
	if err != nil {
		return err
	}
	workflowIDs, err := fornaxmirror.ValidateInventory(*source, config)
	if err != nil {
		return err
	}
	packages, err := buildPackageSet(*output, workflowIDs, func(workflowID string) string {
		return workflowID + "-" + *version + ".zip"
	}, func(workflowID, target string) (fornaxmirror.BuiltArchive, error) {
		key, keyErr := fornaxmirror.KeyForPrefix(writable.KeyPrefix, workflowID)
		if keyErr != nil {
			return fornaxmirror.BuiltArchive{}, keyErr
		}
		return fornaxmirror.BuildArchive(*source, workflowID, key, commit, *version, target)
	})
	if err != nil {
		return err
	}
	return writeJSON(map[string]any{"outcome": "passed", "source_commit": commit, "packages": packages})
}

func validate(args []string) error {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() == 0 {
		return fmt.Errorf("validate requires at least one ZIP path")
	}
	results := make([]map[string]any, 0, flags.NArg())
	for _, filename := range flags.Args() {
		archive, err := fornaxmirror.ValidateArchive(filename)
		if err != nil {
			return fmt.Errorf("%s: %w", filename, err)
		}
		results = append(results, map[string]any{"path": filename, "metadata": archive.Metadata, "file_count": len(archive.Entries)})
	}
	return writeJSON(map[string]any{"outcome": "passed", "packages": results})
}

func compare(args []string) error {
	flags := flag.NewFlagSet("compare", flag.ContinueOnError)
	archive := flags.String("archive", "", "candidate ZIP path")
	directory := flags.String("directory", "", "downloaded package root")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *archive == "" || *directory == "" || flags.NArg() != 0 {
		return fmt.Errorf("compare requires --archive and --directory")
	}
	comparison, err := fornaxmirror.CompareArchiveDirectory(*archive, *directory)
	if err != nil {
		return err
	}
	result := map[string]any{"outcome": "passed", "archive": *archive, "directory": *directory}
	if len(comparison.Notes) > 0 {
		result["notes"] = comparison.Notes
	}
	return writeJSON(result)
}

func cleanCommit(source string) (string, error) {
	root, err := git(source, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("source must be a Git repository")
	}
	canonical, err := filepath.EvalSymlinks(source)
	if err != nil || root != canonical {
		return "", fmt.Errorf("source must be the Git repository root")
	}
	status, err := git(source, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return "", err
	}
	if status != "" {
		return "", fmt.Errorf("source must be a clean Git commit")
	}
	commit, err := git(source, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return commit, nil
}

func git(directory string, args ...string) (string, error) {
	command := exec.Command("git", args...)
	command.Dir = directory
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("git %s failed", args[0])
	}
	return strings.TrimSpace(string(output)), nil
}

func buildPackageSet(output string, workflowIDs []string, archiveName func(string) string, buildArchive func(string, string) (fornaxmirror.BuiltArchive, error)) ([]fornaxmirror.BuiltArchive, error) {
	if err := requireEmptyDirectory(output); err != nil {
		return nil, err
	}
	staging, err := os.MkdirTemp(filepath.Dir(output), ".workflow-packages-")
	if err != nil {
		return nil, err
	}
	keep := false
	defer func() {
		if !keep {
			_ = os.RemoveAll(staging)
		}
	}()
	packages := make([]fornaxmirror.BuiltArchive, 0, len(workflowIDs))
	for _, workflowID := range workflowIDs {
		target := filepath.Join(staging, archiveName(workflowID))
		built, buildErr := buildArchive(workflowID, target)
		if buildErr != nil {
			return nil, buildErr
		}
		packages = append(packages, built)
	}
	if err = replaceEmptyDirectory(output, staging); err != nil {
		return nil, err
	}
	keep = true
	for index := range packages {
		packages[index].Path = filepath.Join(output, filepath.Base(packages[index].Path))
	}
	return packages, nil
}

func requireEmptyDirectory(directory string) error {
	if err := os.MkdirAll(filepath.Dir(directory), 0755); err != nil {
		return err
	}
	info, err := os.Lstat(directory)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("output path must be an empty directory")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("output directory must be empty")
	}
	return nil
}

func replaceEmptyDirectory(output, staging string) error {
	if err := requireEmptyDirectory(output); err != nil {
		return err
	}
	if err := os.Remove(output); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(staging, output); err != nil {
		return err
	}
	return nil
}

func writeJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func fail(message string) {
	_, _ = fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
