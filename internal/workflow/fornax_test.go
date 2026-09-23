package workflow

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPackageRejectsUnsafePathsBeforeExtraction(t *testing.T) {
	for _, name := range []string{"../escape", "/absolute", "a/../../escape", `a\escape`, "C:/escape"} {
		t.Run(name, func(t *testing.T) {
			var buffer bytes.Buffer
			writer := zip.NewWriter(&buffer)
			entry, _ := writer.Create("SKILL.md")
			_, _ = entry.Write([]byte("valid wrapper"))
			entry, _ = writer.Create(name)
			_, _ = entry.Write([]byte("unsafe"))
			_ = writer.Close()
			target := filepath.Join(t.TempDir(), "out")
			if err := extractPackage(buffer.Bytes(), target, "example"); err == nil {
				t.Fatal("unsafe archive accepted")
			}
			if _, err := os.Stat(target); !os.IsNotExist(err) {
				t.Fatal("wrote files before validating every archive entry")
			}
		})
	}
}
func TestPackageRejectsSymlinkBeforeExtraction(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	entry, _ := writer.Create("SKILL.md")
	_, _ = entry.Write([]byte("wrapper"))
	header := &zip.FileHeader{Name: "workflows/example/skills/example/link"}
	header.SetMode(os.ModeSymlink | 0777)
	entry, _ = writer.CreateHeader(header)
	_, _ = entry.Write([]byte("../../../../outside"))
	_ = writer.Close()
	target := filepath.Join(t.TempDir(), "out")
	if err := extractPackage(buffer.Bytes(), target, "example"); err == nil {
		t.Fatal("symlink accepted")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("wrote before symlink validation")
	}
}

func TestSourceRejectsDuplicateKeys(t *testing.T) {
	for _, input := range []string{`{"workspace_id":"1","workspace_id":"2"}`, `{"workflows":[{"id":"first","id":"second"}]}`} {
		if err := rejectDuplicateKeys([]byte(input)); err == nil {
			t.Fatal("duplicate JSON key accepted")
		}
	}
	if err := rejectDuplicateKeys([]byte(`{"workspace_id":"1","workflows":[{"id":"example"}]}`)); err != nil {
		t.Fatal(err)
	}
}
