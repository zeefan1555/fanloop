package currentlayout

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func layoutFixture(t *testing.T) (string, os.FileInfo) {
	t.Helper()
	root := t.TempDir()
	for _, version := range []string{"1.2.2", "1.2.3"} {
		if err := os.MkdirAll(filepath.Join(root, "releases", version, "bin"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "releases", version, "bin", "fanloop"), []byte(version), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	current := filepath.Join(root, "current")
	if err := os.Mkdir(current, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, target := range managedLinks {
		if err := os.Symlink(target, filepath.Join(current, name)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("../releases/1.2.2", filepath.Join(current, "release")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(current)
	if err != nil {
		t.Fatal(err)
	}
	return root, info
}

func TestConcurrentMigrationRetainsOneOriginalDirectory(t *testing.T) {
	root, before := layoutFixture(t)
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if _, err := Switch(root, "1.2.3"); err != nil {
				t.Error(err)
			}
		}()
	}
	workers.Wait()
	backups, err := filepath.Glob(filepath.Join(root, "current.backup-*"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("backups = %v, %v", backups, err)
	}
	after, err := os.Lstat(backups[0])
	if err != nil || !os.SameFile(before, after) {
		t.Fatalf("original directory changed: %v", err)
	}
}

func TestMigrationNeverRemovesCurrentFromReaders(t *testing.T) {
	root, _ := layoutFixture(t)
	var reader sync.WaitGroup
	reader.Add(1)
	go func() {
		defer reader.Done()
		for i := 0; i < 1000; i++ {
			body, err := os.ReadFile(filepath.Join(root, "current", "bin", "fanloop"))
			if err != nil || (string(body) != "1.2.2" && string(body) != "1.2.3") {
				t.Errorf("current disappeared: %q, %v", body, err)
				return
			}
		}
	}()
	if _, err := Switch(root, "1.2.3"); err != nil {
		t.Error(err)
	}
	reader.Wait()
}

func TestFailedExchangePreservesOriginalDirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory write permissions")
	}
	root, before := layoutFixture(t)
	prepared := filepath.Join(root, "current.backup-prepared")
	if err := os.Symlink("releases/1.2.3", prepared); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(root, 0o755); err != nil {
			t.Error(err)
		}
	})
	if err := exchange(prepared, filepath.Join(root, "current")); err == nil {
		t.Fatal("expected exchange failure")
	}
	after, err := os.Lstat(filepath.Join(root, "current"))
	if err != nil || !os.SameFile(before, after) || !after.IsDir() {
		t.Fatalf("failed exchange changed current: %v", err)
	}
	if target, err := os.Readlink(prepared); err != nil || target != "releases/1.2.3" {
		t.Fatalf("prepared link changed: %q, %v", target, err)
	}
}
