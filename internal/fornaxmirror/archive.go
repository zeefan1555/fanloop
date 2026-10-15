package fornaxmirror

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/zeefan1555/fanloop/internal/workflow"
	"go.yaml.in/yaml/v3"
)

const SourceRepository = "git@github.com:zeefan1555/second-brain-skills.git"

const maxArchiveFileSize = 16 << 20
const maxArchiveTotalSize = 64 << 20

var commitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)
var stableVersionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

type Metadata struct {
	WorkflowID       string `yaml:"workflow_id" json:"workflow_id"`
	SkillKey         string `yaml:"skill_key" json:"skill_key"`
	SourceRepository string `yaml:"source_repository" json:"source_repository"`
	SourceCommit     string `yaml:"source_commit" json:"source_commit"`
	Version          string `yaml:"version" json:"version"`
}

type Entry struct {
	Path       string `json:"path"`
	SHA256     string `json:"sha256"`
	Executable bool   `json:"executable"`
}

type Archive struct {
	Metadata Metadata `json:"metadata"`
	Entries  []Entry  `json:"entries"`
}

type BuiltArchive struct {
	WorkflowID string `json:"workflow_id"`
	SkillKey   string `json:"skill_key"`
	Version    string `json:"version"`
	Path       string `json:"path"`
	SHA256     string `json:"sha256"`
	FileCount  int    `json:"file_count"`
}

type wrapperFrontmatter struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Metadata    Metadata `yaml:"metadata"`
}

func Discover(source string) ([]string, error) {
	root, err := workflow.SourceRoot(source)
	if err != nil {
		return nil, err
	}
	loaded, err := workflow.ListFrom(root)
	if err != nil {
		return nil, err
	}
	result := make([]string, len(loaded))
	for index, item := range loaded {
		result[index] = item.Ref.ID
	}
	return result, nil
}

func ValidateInventory(source string, config Config) ([]string, error) {
	actual, err := Discover(source)
	if err != nil {
		return nil, err
	}
	expected := config.WorkflowIDs()
	if !reflect.DeepEqual(actual, expected) {
		return nil, fmt.Errorf("Fornax Workflow inventory is %v, want %v", expected, actual)
	}
	return actual, nil
}

func BuildArchive(source, workflowID, skillKey, commit, version, target string) (BuiltArchive, error) {
	metadata := Metadata{WorkflowID: workflowID, SkillKey: skillKey, SourceRepository: SourceRepository, SourceCommit: commit, Version: version}
	if err := metadata.validate(); err != nil {
		return BuiltArchive{}, err
	}
	root, err := workflow.SourceRoot(source)
	if err != nil {
		return BuiltArchive{}, err
	}
	if _, err = workflow.LoadFrom(root, workflowID); err != nil {
		return BuiltArchive{}, err
	}
	files, err := packageFiles(filepath.Join(root, workflowID), workflowID, wrapper(metadata))
	if err != nil {
		return BuiltArchive{}, err
	}
	if err = os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return BuiltArchive{}, err
	}
	temporary, err := os.CreateTemp(filepath.Dir(target), ".workflow-package-*.zip")
	if err != nil {
		return BuiltArchive{}, err
	}
	temporaryPath := temporary.Name()
	keep := false
	defer func() {
		_ = temporary.Close()
		if !keep {
			_ = os.Remove(temporaryPath)
		}
	}()
	writer := zip.NewWriter(temporary)
	for _, file := range files {
		header := &zip.FileHeader{Name: file.path, Method: zip.Deflate}
		header.SetModTime(time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))
		header.SetMode(file.mode)
		entry, failure := writer.CreateHeader(header)
		if failure != nil {
			return BuiltArchive{}, failure
		}
		if _, failure = entry.Write(file.content); failure != nil {
			return BuiltArchive{}, failure
		}
	}
	if err = writer.Close(); err != nil {
		return BuiltArchive{}, err
	}
	if err = temporary.Sync(); err != nil {
		return BuiltArchive{}, err
	}
	if err = temporary.Close(); err != nil {
		return BuiltArchive{}, err
	}
	if err = os.Rename(temporaryPath, target); err != nil {
		return BuiltArchive{}, err
	}
	keep = true
	if _, err = ValidateArchive(target); err != nil {
		return BuiltArchive{}, err
	}
	digest, err := fileDigest(target)
	if err != nil {
		return BuiltArchive{}, err
	}
	return BuiltArchive{WorkflowID: workflowID, SkillKey: skillKey, Version: version, Path: target, SHA256: digest, FileCount: len(files)}, nil
}

func ValidateArchive(filename string) (Archive, error) {
	reader, err := zip.OpenReader(filename)
	if err != nil {
		return Archive{}, err
	}
	defer reader.Close()
	contents := map[string][]byte{}
	modes := map[string]fs.FileMode{}
	seen := map[string]bool{}
	top := map[string]bool{}
	var total uint64
	for _, file := range reader.File {
		name, err := safeArchivePath(file.Name)
		if err != nil {
			return Archive{}, err
		}
		if seen[name] {
			return Archive{}, fmt.Errorf("duplicate archive path %s", name)
		}
		seen[name] = true
		first, _, _ := strings.Cut(name, "/")
		top[first] = true
		mode := file.Mode()
		if mode.IsDir() {
			continue
		}
		if !mode.IsRegular() {
			return Archive{}, fmt.Errorf("archive path %s is not a regular file", name)
		}
		if file.UncompressedSize64 > maxArchiveFileSize || total+file.UncompressedSize64 > maxArchiveTotalSize {
			return Archive{}, fmt.Errorf("archive size limit exceeded")
		}
		total += file.UncompressedSize64
		stream, err := file.Open()
		if err != nil {
			return Archive{}, err
		}
		content, readErr := io.ReadAll(io.LimitReader(stream, maxArchiveFileSize+1))
		closeErr := stream.Close()
		if readErr != nil {
			return Archive{}, readErr
		}
		if closeErr != nil {
			return Archive{}, closeErr
		}
		contents[name], modes[name] = content, mode
	}
	metadata, workflowID, err := validateArchiveContents(contents, top)
	if err != nil {
		return Archive{}, err
	}
	if metadata.WorkflowID != workflowID {
		return Archive{}, fmt.Errorf("wrapper Workflow ID does not match archive directory")
	}
	return Archive{Metadata: metadata, Entries: entries(contents, modes)}, nil
}

// ArchiveDirectoryComparison is the result of comparing a built ZIP with an
// installed or downloaded directory. Notes are informational Unix-mode
// differences. Fornax skill install does not preserve file modes, so those
// differences do not fail the comparison.
type ArchiveDirectoryComparison struct {
	Notes []string
}

func CompareArchiveDirectory(archivePath, directory string) (ArchiveDirectoryComparison, error) {
	archive, err := ValidateArchive(archivePath)
	if err != nil {
		return ArchiveDirectoryComparison{}, err
	}
	directoryEntries, err := DirectoryEntries(directory)
	if err != nil {
		return ArchiveDirectoryComparison{}, err
	}
	if !samePackageContent(archive.Entries, directoryEntries) {
		return ArchiveDirectoryComparison{}, fmt.Errorf("package content differs")
	}
	return ArchiveDirectoryComparison{Notes: modeNotes(archive.Entries, directoryEntries)}, nil
}

func samePackageContent(left, right []Entry) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].Path != right[index].Path || left[index].SHA256 != right[index].SHA256 {
			return false
		}
	}
	return true
}

func modeNotes(left, right []Entry) []string {
	var notes []string
	for index := range left {
		if left[index].Executable != right[index].Executable {
			notes = append(notes, "file mode differs for "+left[index].Path)
		}
	}
	return notes
}

func DirectoryEntries(root string) ([]Entry, error) {
	contents := map[string][]byte{}
	modes := map[string]fs.FileMode{}
	top := map[string]bool{}
	var total int64
	err := filepath.WalkDir(root, func(filename string, item fs.DirEntry, failure error) error {
		if failure != nil {
			return failure
		}
		if filename == root {
			return nil
		}
		relative, err := filepath.Rel(root, filename)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(relative)
		first, _, _ := strings.Cut(name, "/")
		top[first] = true
		info, err := item.Info()
		if err != nil {
			return err
		}
		if item.IsDir() {
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("package directory contains a symlink")
			}
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("package entry is not a regular file")
		}
		if info.Size() > maxArchiveFileSize || total+info.Size() > maxArchiveTotalSize {
			return fmt.Errorf("archive size limit exceeded")
		}
		stream, err := os.Open(filename)
		if err != nil {
			return err
		}
		content, readErr := io.ReadAll(io.LimitReader(stream, maxArchiveFileSize+1))
		closeErr := stream.Close()
		if readErr != nil {
			return readErr
		}
		if closeErr != nil {
			return closeErr
		}
		if len(content) > maxArchiveFileSize || total+int64(len(content)) > maxArchiveTotalSize {
			return fmt.Errorf("archive size limit exceeded")
		}
		total += int64(len(content))
		contents[name], modes[name] = content, info.Mode()
		return nil
	})
	if err != nil {
		return nil, err
	}
	if _, _, err = validateArchiveContents(contents, top); err != nil {
		return nil, err
	}
	return entries(contents, modes), nil
}

type packageFile struct {
	path    string
	content []byte
	mode    fs.FileMode
}

func packageFiles(packageRoot, workflowID string, wrapperContent []byte) ([]packageFile, error) {
	result := []packageFile{{path: "SKILL.md", content: wrapperContent, mode: 0644}}
	for _, name := range workflow.BundleFileNames() {
		content, err := os.ReadFile(filepath.Join(packageRoot, name))
		if err != nil {
			return nil, err
		}
		result = append(result, packageFile{path: path.Join(workflowID, name), content: content, mode: 0644})
	}
	trace := filepath.Join(packageRoot, "trace.yaml")
	if info, err := os.Lstat(trace); err == nil {
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("trace.yaml is not a regular file")
		}
		content, err := os.ReadFile(trace)
		if err != nil {
			return nil, err
		}
		result = append(result, packageFile{path: path.Join(workflowID, "trace.yaml"), content: content, mode: 0644})
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	skills := filepath.Join(packageRoot, "skills")
	err := filepath.WalkDir(skills, func(filename string, item fs.DirEntry, failure error) error {
		if failure != nil {
			return failure
		}
		if item.IsDir() {
			if item.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("Skill directory contains a symlink")
			}
			return nil
		}
		info, err := item.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("Skill entry is not a regular file")
		}
		relative, err := filepath.Rel(packageRoot, filename)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		mode := fs.FileMode(0644)
		if info.Mode().Perm()&0111 != 0 {
			mode = 0755
		}
		result = append(result, packageFile{path: path.Join(workflowID, filepath.ToSlash(relative)), content: content, mode: mode})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(result, func(i, j int) bool { return result[i].path < result[j].path })
	return result, nil
}

func validateArchiveContents(contents map[string][]byte, top map[string]bool) (Metadata, string, error) {
	wrapperContent, ok := contents["SKILL.md"]
	if !ok {
		return Metadata{}, "", fmt.Errorf("archive root is missing SKILL.md")
	}
	if len(top) != 2 || !top["SKILL.md"] {
		return Metadata{}, "", fmt.Errorf("archive root must contain only SKILL.md and one Workflow directory")
	}
	workflowID := ""
	for name := range top {
		if name != "SKILL.md" {
			workflowID = name
		}
	}
	if !workflowIDPattern.MatchString(workflowID) {
		return Metadata{}, "", fmt.Errorf("invalid Workflow directory %q", workflowID)
	}
	metadata, err := parseWrapper(wrapperContent)
	if err != nil {
		return Metadata{}, "", err
	}
	bundle := map[string][]byte{}
	for _, name := range workflow.BundleFileNames() {
		filename := path.Join(workflowID, name)
		content, ok := contents[filename]
		if !ok {
			return Metadata{}, "", fmt.Errorf("archive is missing %s", filename)
		}
		bundle[name] = content
	}
	loaded, err := workflow.DecodeBundle(bundle["workflow.yaml"], bundle["flow.yaml"], bundle["condition.yaml"], bundle["loop.yaml"], bundle["prompt.yaml"])
	if err != nil {
		return Metadata{}, "", err
	}
	if loaded.Ref.ID != workflowID {
		return Metadata{}, "", fmt.Errorf("Workflow directory does not match workflow.yaml id")
	}
	requiredSkills := map[string]bool{}
	for _, binding := range loaded.Workflow.CommonSkills {
		requiredSkills[binding.ID] = true
	}
	for _, prompt := range loaded.Workflow.Prompts {
		for _, binding := range prompt.Skills {
			requiredSkills[binding.ID] = true
		}
	}
	for skillID := range requiredSkills {
		if _, ok := contents[path.Join(workflowID, "skills", skillID, "SKILL.md")]; !ok {
			return Metadata{}, "", fmt.Errorf("archive is missing required Skill %s", skillID)
		}
	}
	childSkills := map[string]bool{}
	prefix := path.Join(workflowID, "skills") + "/"
	for name := range contents {
		if name == "SKILL.md" {
			continue
		}
		relative := strings.TrimPrefix(name, workflowID+"/")
		first, _, _ := strings.Cut(relative, "/")
		if first != "workflow.yaml" && first != "condition.yaml" && first != "flow.yaml" && first != "loop.yaml" && first != "prompt.yaml" && first != "trace.yaml" && first != "skills" {
			return Metadata{}, "", fmt.Errorf("unexpected Workflow package entry %s", name)
		}
		if first != "skills" && relative != first {
			return Metadata{}, "", fmt.Errorf("unexpected Workflow package entry %s", name)
		}
		if strings.HasPrefix(name, prefix) {
			child, remainder, found := strings.Cut(strings.TrimPrefix(name, prefix), "/")
			if !found || child == "" || remainder == "" {
				return Metadata{}, "", fmt.Errorf("unexpected Workflow package entry %s", name)
			}
			childSkills[child] = true
		}
	}
	if len(childSkills) == 0 {
		return Metadata{}, "", fmt.Errorf("archive contains no child Skills")
	}
	for child := range childSkills {
		if _, ok := contents[path.Join(workflowID, "skills", child, "SKILL.md")]; !ok {
			return Metadata{}, "", fmt.Errorf("Skill %s is missing SKILL.md", child)
		}
	}
	return metadata, workflowID, nil
}

func safeArchivePath(name string) (string, error) {
	if name == "" || strings.Contains(name, `\`) || strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("unsafe archive path %q", name)
	}
	clean := path.Clean(name)
	if clean == "." || clean != strings.TrimSuffix(name, "/") || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("unsafe archive path %q", name)
	}
	return clean, nil
}

func wrapper(metadata Metadata) []byte {
	return []byte(fmt.Sprintf("---\nname: fanloop-workflow-%s\ndescription: Fornax mirror of Fanloop Workflow %s.\nmetadata:\n  workflow_id: %s\n  skill_key: %s\n  source_repository: %s\n  source_commit: %s\n  version: %s\n---\n\n# Fanloop Workflow mirror\n\nThis package mirrors second-brain-skills Git configuration. Fanloop runtime continues to read Git snapshots only.\n", metadata.WorkflowID, metadata.WorkflowID, metadata.WorkflowID, metadata.SkillKey, metadata.SourceRepository, metadata.SourceCommit, metadata.Version))
}

func parseWrapper(content []byte) (Metadata, error) {
	if !bytes.HasPrefix(content, []byte("---\n")) {
		return Metadata{}, fmt.Errorf("wrapper SKILL.md has no frontmatter")
	}
	end := bytes.Index(content[4:], []byte("\n---"))
	if end < 0 {
		return Metadata{}, fmt.Errorf("wrapper SKILL.md frontmatter is invalid")
	}
	var frontmatter wrapperFrontmatter
	decoder := yaml.NewDecoder(bytes.NewReader(content[4 : 4+end]))
	decoder.KnownFields(true)
	if err := decoder.Decode(&frontmatter); err != nil {
		return Metadata{}, fmt.Errorf("wrapper SKILL.md: %w", err)
	}
	if frontmatter.Name == "" || frontmatter.Description == "" {
		return Metadata{}, fmt.Errorf("wrapper SKILL.md requires name and description")
	}
	if err := frontmatter.Metadata.validate(); err != nil {
		return Metadata{}, err
	}
	return frontmatter.Metadata, nil
}

func (metadata Metadata) validate() error {
	expected, err := SkillKey(metadata.WorkflowID)
	if err != nil {
		return err
	}
	if metadata.SkillKey != expected {
		return fmt.Errorf("Skill key is %q, want %q", metadata.SkillKey, expected)
	}
	if metadata.SourceRepository != SourceRepository {
		return fmt.Errorf("invalid source repository")
	}
	if !commitPattern.MatchString(metadata.SourceCommit) {
		return fmt.Errorf("invalid source commit")
	}
	if !stableVersionPattern.MatchString(metadata.Version) {
		return fmt.Errorf("invalid stable SemVer %q", metadata.Version)
	}
	return nil
}

func entries(contents map[string][]byte, modes map[string]fs.FileMode) []Entry {
	result := make([]Entry, 0, len(contents))
	for name, content := range contents {
		sum := sha256.Sum256(content)
		result = append(result, Entry{Path: name, SHA256: hex.EncodeToString(sum[:]), Executable: modes[name].Perm()&0111 != 0})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}

func fileDigest(filename string) (string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}
