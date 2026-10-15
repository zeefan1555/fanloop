package fornaxmirror

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/zeefan1555/fanloop/internal/workflow"
	"go.yaml.in/yaml/v3"
)

// AcceptedPackage is a downloaded Fornax package after the wrapper directory
// is checked. WorkflowID comes from workflow.yaml. SkillKey and Version are
// set only when the wrapper recorded them.
type AcceptedPackage struct {
	WorkflowID  string
	WrapperName string
	SkillKey    string
	Version     string
}

type downloadedWrapper struct {
	Name        string              `yaml:"name"`
	Description string              `yaml:"description"`
	Metadata    *downloadedMetadata `yaml:"metadata"`
}

type downloadedMetadata struct {
	WorkflowID       string `yaml:"workflow_id"`
	SkillKey         string `yaml:"skill_key"`
	SourceRepository string `yaml:"source_repository"`
	SourceCommit     string `yaml:"source_commit"`
	Version          string `yaml:"version"`
}

type workflowIDDocument struct {
	ID string `yaml:"id"`
}

// AcceptPackageRoot checks one fornax-cli install directory and loads it.
// The directory is <package-name>/SKILL.md plus <package-name>/<workflow-id>/.
func AcceptPackageRoot(root string) (AcceptedPackage, error) {
	parent, err := os.MkdirTemp("", "fanloop-package-")
	if err != nil {
		return AcceptedPackage{}, err
	}
	defer os.RemoveAll(parent)
	return MaterializePackage(root, parent)
}

// MaterializePackage copies one downloaded package into workflowParent/<id>,
// writes the wrapper SKILL.md into that directory when the package has none,
// and makes shebang scripts executable.
func MaterializePackage(packageRoot, workflowParent string) (AcceptedPackage, error) {
	info, err := os.Lstat(packageRoot)
	if err != nil {
		return AcceptedPackage{}, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return AcceptedPackage{}, fmt.Errorf("package root must be a real directory")
	}
	wrapperPath := filepath.Join(packageRoot, "SKILL.md")
	wrapperInfo, err := os.Lstat(wrapperPath)
	if err != nil || !wrapperInfo.Mode().IsRegular() || wrapperInfo.Mode()&os.ModeSymlink != 0 {
		return AcceptedPackage{}, fmt.Errorf("package root is missing SKILL.md")
	}
	if wrapperInfo.Size() > maxArchiveFileSize {
		return AcceptedPackage{}, fmt.Errorf("archive size limit exceeded")
	}
	wrapperContent, err := os.ReadFile(wrapperPath)
	if err != nil {
		return AcceptedPackage{}, err
	}
	header, err := parseDownloadedWrapper(wrapperContent)
	if err != nil {
		return AcceptedPackage{}, err
	}
	workflowDir, err := singleWorkflowDirectory(packageRoot)
	if err != nil {
		return AcceptedPackage{}, err
	}
	id, err := readWorkflowID(filepath.Join(packageRoot, workflowDir, "workflow.yaml"))
	if err != nil {
		return AcceptedPackage{}, err
	}
	if workflowDir != id {
		return AcceptedPackage{}, fmt.Errorf("workflow directory %q does not match workflow.yaml id %q", workflowDir, id)
	}
	if header.Name != "loop-workflow-"+id && header.Name != "fanloop-workflow-"+id {
		return AcceptedPackage{}, fmt.Errorf("package name %q does not match workflow id %q", header.Name, id)
	}
	accepted := AcceptedPackage{WorkflowID: id, WrapperName: header.Name}
	if header.Metadata != nil {
		if header.Metadata.WorkflowID != "" && header.Metadata.WorkflowID != id {
			return AcceptedPackage{}, fmt.Errorf("wrapper workflow_id %q does not match workflow.yaml id %q", header.Metadata.WorkflowID, id)
		}
		if header.Metadata.SkillKey != "" {
			if err = checkOptionalSkillKey(id, header.Metadata.SkillKey); err != nil {
				return AcceptedPackage{}, err
			}
			accepted.SkillKey = header.Metadata.SkillKey
		}
		if header.Metadata.SourceCommit != "" && !commitPattern.MatchString(header.Metadata.SourceCommit) {
			return AcceptedPackage{}, fmt.Errorf("invalid source commit")
		}
		if header.Metadata.Version != "" && !stableVersionPattern.MatchString(header.Metadata.Version) {
			return AcceptedPackage{}, fmt.Errorf("invalid stable SemVer %q", header.Metadata.Version)
		}
		accepted.Version = header.Metadata.Version
	}
	destination := filepath.Join(workflowParent, id)
	if _, err = os.Lstat(destination); err == nil {
		return AcceptedPackage{}, fmt.Errorf("workflow %s is already materialized", id)
	}
	if err = copyWorkflowTree(filepath.Join(packageRoot, workflowDir), destination, wrapperContent); err != nil {
		return AcceptedPackage{}, err
	}
	if _, err = workflow.LoadFrom(workflowParent, id); err != nil {
		return AcceptedPackage{}, err
	}
	return accepted, nil
}

func parseDownloadedWrapper(content []byte) (downloadedWrapper, error) {
	if !bytes.HasPrefix(content, []byte("---\n")) {
		return downloadedWrapper{}, fmt.Errorf("wrapper SKILL.md has no frontmatter")
	}
	end := bytes.Index(content[4:], []byte("\n---"))
	if end < 0 {
		return downloadedWrapper{}, fmt.Errorf("wrapper SKILL.md frontmatter is invalid")
	}
	var header downloadedWrapper
	decoder := yaml.NewDecoder(bytes.NewReader(content[4 : 4+end]))
	decoder.KnownFields(true)
	if err := decoder.Decode(&header); err != nil {
		return downloadedWrapper{}, fmt.Errorf("wrapper SKILL.md: %w", err)
	}
	if header.Name == "" || header.Description == "" {
		return downloadedWrapper{}, fmt.Errorf("wrapper SKILL.md requires name and description")
	}
	return header, nil
}

func singleWorkflowDirectory(root string) (string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	name := ""
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("package entry %s is a symlink", entry.Name())
		}
		if entry.Name() == "SKILL.md" {
			if !info.Mode().IsRegular() {
				return "", fmt.Errorf("package SKILL.md is not a regular file")
			}
			continue
		}
		if !info.IsDir() {
			return "", fmt.Errorf("unexpected package entry %s", entry.Name())
		}
		if name != "" {
			return "", fmt.Errorf("package root must contain one workflow directory")
		}
		name = entry.Name()
	}
	if name == "" {
		return "", fmt.Errorf("package root is missing a workflow directory")
	}
	return name, nil
}

func readWorkflowID(filename string) (string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("workflow.yaml: %w", err)
	}
	if len(content) > maxArchiveFileSize {
		return "", fmt.Errorf("archive size limit exceeded")
	}
	var document workflowIDDocument
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	if err = decoder.Decode(&document); err != nil {
		return "", fmt.Errorf("workflow.yaml: %w", err)
	}
	if !workflowIDPattern.MatchString(document.ID) {
		return "", fmt.Errorf("invalid workflow id %q", document.ID)
	}
	return document.ID, nil
}

func checkOptionalSkillKey(workflowID, key string) error {
	fanloopKey, err := SkillKey(workflowID)
	if err != nil {
		return err
	}
	loopKey := "loop.workflow." + strings.ReplaceAll(workflowID, "-", "_")
	if key != fanloopKey && key != loopKey {
		return fmt.Errorf("Skill key is %q, want %q", key, fanloopKey)
	}
	return nil
}

func copyWorkflowTree(source, destination string, wrapper []byte) error {
	if err := os.MkdirAll(destination, 0755); err != nil {
		return err
	}
	sawSkill := false
	var total uint64
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, failure error) error {
		if failure != nil {
			return failure
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
			return fmt.Errorf("package entry %s is not a regular file or directory", entry.Name())
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if relative == ".." || strings.HasPrefix(relative, "../") || strings.Contains(relative, `\`) {
			return fmt.Errorf("unsafe archive path %q", relative)
		}
		target := filepath.Join(destination, filepath.FromSlash(relative))
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		if info.Size() > maxArchiveFileSize || total+uint64(info.Size()) > maxArchiveTotalSize {
			return fmt.Errorf("archive size limit exceeded")
		}
		total += uint64(info.Size())
		content, err := readRegularFile(path, info.Size())
		if err != nil {
			return err
		}
		mode := fs.FileMode(0444)
		if bytes.HasPrefix(content, []byte("#!")) {
			mode = 0555
		}
		if relative == "SKILL.md" {
			sawSkill = true
		}
		return os.WriteFile(target, content, mode)
	})
	if err != nil {
		return err
	}
	if !sawSkill {
		if err = os.WriteFile(filepath.Join(destination, "SKILL.md"), wrapper, 0444); err != nil {
			return err
		}
	}
	return nil
}

func readRegularFile(filename string, size int64) ([]byte, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() != size {
		return nil, fmt.Errorf("package entry changed during read")
	}
	content, err := io.ReadAll(io.LimitReader(file, maxArchiveFileSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > maxArchiveFileSize {
		return nil, fmt.Errorf("archive size limit exceeded")
	}
	return content, nil
}
