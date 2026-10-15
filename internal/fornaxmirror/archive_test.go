package fornaxmirror

import (
	"archive/zip"
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

const testCommit = "0123456789abcdef0123456789abcdef01234567"

func TestBuildArchiveIsDeterministicAndComparable(t *testing.T) {
	source := fixtureSource(t)
	key, err := SkillKey("linear-runtime")
	if err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(t.TempDir(), "first.zip")
	second := filepath.Join(t.TempDir(), "second.zip")
	if _, err = BuildArchive(source, "linear-runtime", key, testCommit, "1.0.0", first); err != nil {
		t.Fatal(err)
	}
	if _, err = BuildArchive(source, "linear-runtime", key, testCommit, "1.0.0", second); err != nil {
		t.Fatal(err)
	}
	firstBytes, _ := os.ReadFile(first)
	secondBytes, _ := os.ReadFile(second)
	if !bytes.Equal(firstBytes, secondBytes) {
		t.Fatal("identical inputs produced different ZIP bytes")
	}
	archive, err := ValidateArchive(first)
	if err != nil {
		t.Fatal(err)
	}
	if archive.Metadata.WorkflowID != "linear-runtime" || archive.Metadata.SkillKey != key || archive.Metadata.SourceCommit != testCommit || archive.Metadata.Version != "1.0.0" {
		t.Fatalf("metadata = %#v", archive.Metadata)
	}
	downloaded := filepath.Join(t.TempDir(), "package")
	extract(t, first, downloaded)
	if err = CompareArchiveDirectory(first, downloaded); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(downloaded, "linear-runtime", "workflow.yaml")
	if err = os.WriteFile(path, append(mustRead(t, path), '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	if err = CompareArchiveDirectory(first, downloaded); err == nil {
		t.Fatal("changed package compared equal")
	}
}

func TestValidateArchiveRejectsInnerPackageEntrySkill(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "bad.zip")
	file, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for name, content := range map[string]string{
		"SKILL.md":                string(wrapper(Metadata{WorkflowID: "linear-runtime", SkillKey: "fanloop.workflow.linear_runtime", SourceRepository: SourceRepository, SourceCommit: testCommit, Version: "1.0.0"})),
		"linear-runtime/SKILL.md": "extra",
	} {
		entry, createErr := writer.Create(name)
		if createErr != nil {
			t.Fatal(createErr)
		}
		_, _ = io.WriteString(entry, content)
	}
	_ = writer.Close()
	_ = file.Close()
	if _, err = ValidateArchive(filename); err == nil {
		t.Fatal("archive with unexpected inner SKILL.md passed")
	}
}

func TestValidateArchiveRejectsDuplicateDirectoriesAndExtraRoots(t *testing.T) {
	valid := filepath.Join(t.TempDir(), "valid.zip")
	source := fixtureSource(t)
	key, _ := SkillKey("linear-runtime")
	if _, err := BuildArchive(source, "linear-runtime", key, testCommit, "1.0.0", valid); err != nil {
		t.Fatal(err)
	}
	for name, extras := range map[string][]zipEntry{
		"duplicate directory": {{name: "linear-runtime/", directory: true}, {name: "linear-runtime/", directory: true}},
		"extra empty root":    {{name: "unexpected/", directory: true}},
		"file below skills":   {{name: "linear-runtime/skills/README.md", content: "bad"}},
	} {
		t.Run(name, func(t *testing.T) {
			bad := filepath.Join(t.TempDir(), "bad.zip")
			copyArchiveWithExtras(t, valid, bad, extras)
			if _, err := ValidateArchive(bad); err == nil {
				t.Fatal("invalid archive passed")
			}
		})
	}
}

func TestDirectoryEntriesRejectsOversizeFile(t *testing.T) {
	root := filepath.Join(t.TempDir(), "package")
	extract(t, mustBuildArchive(t), root)
	filename := filepath.Join(root, "linear-runtime", "skills", "test-guide", "large.bin")
	file, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	if err = file.Truncate(maxArchiveFileSize + 1); err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	if _, err = DirectoryEntries(root); err == nil {
		t.Fatal("oversize directory package passed")
	}
}

func TestDirectoryEntriesRejectsExtraEmptyRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "package")
	extract(t, mustBuildArchive(t), root)
	if err := os.Mkdir(filepath.Join(root, "unexpected"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := DirectoryEntries(root); err == nil {
		t.Fatal("package with an extra root directory passed")
	}
}

type zipEntry struct {
	name      string
	content   string
	directory bool
}

func copyArchiveWithExtras(t *testing.T, source, target string, extras []zipEntry) {
	t.Helper()
	reader, err := zip.OpenReader(source)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	file, err := os.Create(target)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for _, item := range reader.File {
		header := item.FileHeader
		entry, createErr := writer.CreateHeader(&header)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if !item.FileInfo().IsDir() {
			stream, openErr := item.Open()
			if openErr != nil {
				t.Fatal(openErr)
			}
			_, _ = io.Copy(entry, stream)
			_ = stream.Close()
		}
	}
	for _, extra := range extras {
		header := &zip.FileHeader{Name: extra.name}
		if extra.directory {
			header.SetMode(fs.ModeDir | 0755)
		}
		entry, createErr := writer.CreateHeader(header)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if extra.content != "" {
			_, _ = io.WriteString(entry, extra.content)
		}
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
}

func mustBuildArchive(t *testing.T) string {
	t.Helper()
	filename := filepath.Join(t.TempDir(), "valid.zip")
	key, _ := SkillKey("linear-runtime")
	if _, err := BuildArchive(fixtureSource(t), "linear-runtime", key, testCommit, "1.0.0", filename); err != nil {
		t.Fatal(err)
	}
	return filename
}

func fixtureSource(t *testing.T) string {
	t.Helper()
	source := t.TempDir()
	destination := filepath.Join(source, "fanloop-workflow", "linear-runtime")
	copyTree(t, filepath.Join("..", "workflow", "testdata", "linear-runtime"), destination)
	return source
}

func copyTree(t *testing.T, source, destination string) {
	t.Helper()
	err := filepath.WalkDir(source, func(filename string, item fs.DirEntry, failure error) error {
		if failure != nil {
			return failure
		}
		relative, err := filepath.Rel(source, filename)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if item.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		info, err := item.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(target, mustRead(t, filename), info.Mode().Perm())
	})
	if err != nil {
		t.Fatal(err)
	}
}

func extract(t *testing.T, archive, destination string) {
	t.Helper()
	reader, err := zip.OpenReader(archive)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		path := filepath.Join(destination, filepath.FromSlash(file.Name))
		if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		stream, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, err := io.ReadAll(stream)
		_ = stream.Close()
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, content, file.Mode().Perm()); err != nil {
			t.Fatal(err)
		}
	}
}

func mustRead(t *testing.T, filename string) []byte {
	t.Helper()
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	return content
}
