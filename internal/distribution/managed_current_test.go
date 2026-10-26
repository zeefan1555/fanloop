package distribution

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func managedCurrent(t *testing.T, root string) os.FileInfo {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "releases", "1.2.2"), 0o755); err != nil {
		t.Fatal(err)
	}
	current := filepath.Join(root, "current")
	if err := os.Mkdir(current, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, target := range map[string]string{"bin": "release/bin", "entrypoints": "release/entrypoints", "release.json": "release/release.json", "release": "../releases/1.2.2"} {
		if err := os.Symlink(target, filepath.Join(current, name)); err != nil {
			t.Fatal(err)
		}
	}
	info, err := os.Lstat(current)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func TestUpdateManagedCurrentMigratesAndPreservesDirectory(t *testing.T) {
	root := t.TempDir()
	before := managedCurrent(t, root)
	server := serveRelease(t, "1.2.3", validArchive(t, "1.2.3", "new binary\n", "# workflow\n", "# author\n"), nil)
	defer server.Close()
	result, err := Update(context.Background(), Options{DataRoot: root, Client: server.Client(), LatestURL: server.URL + "/latest", Platform: Platform{OS: "linux", Arch: "amd64"}, SkillRoots: testSkillRoots(t, root)})
	if err != nil || result.Version != "1.2.3" || result.Unchanged {
		t.Fatalf("managed directory update = %+v, %v", result, err)
	}
	if target, err := os.Readlink(filepath.Join(root, "current")); err != nil || target != "releases/1.2.3" {
		t.Fatalf("current = %q, %v", target, err)
	}
	backups, err := filepath.Glob(filepath.Join(root, "current.backup-*"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("backups = %v, %v", backups, err)
	}
	after, err := os.Lstat(backups[0])
	if err != nil || !os.SameFile(before, after) {
		t.Fatalf("original directory inode was not retained: %v", err)
	}
	if target, err := os.Readlink(filepath.Join(backups[0], "release")); err != nil || target != "../releases/1.2.2" {
		t.Fatalf("backup release = %q, %v", target, err)
	}
}

func TestAutoUpdateLeavesManagedDirectoryOffline(t *testing.T) {
	root := t.TempDir()
	before := managedCurrent(t, root)
	_, err := AutoUpdate(context.Background(), Options{DataRoot: root, LatestURL: "http://127.0.0.1:1/must-not-connect"})
	if !errors.Is(err, ErrNoReleaseInstall) {
		t.Fatalf("native directory entered automatic update: %v", err)
	}
	after, err := os.Lstat(filepath.Join(root, "current"))
	if err != nil || !os.SameFile(before, after) {
		t.Fatalf("automatic update changed current: %v", err)
	}
}

func TestSameVersionFailedRepairRestoresActiveReleaseBytes(t *testing.T) {
	for _, stage := range []string{"skill", "configuration", "activation"} {
		t.Run(stage, func(t *testing.T) {
			root := t.TempDir()
			before := managedCurrent(t, root)
			old := filepath.Join(root, "releases", "1.2.2")
			if err := extractTarGz(validArchive(t, "1.2.2", "old binary", "# workflow\n", "# author\n"), old); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(old, "user-marker"), []byte("keep"), 0o600); err != nil {
				t.Fatal(err)
			}
			oldInfo, _ := os.Lstat(old)
			roots := testSkillRoots(t, root)
			if stage == "skill" {
				if err := os.WriteFile(filepath.Join(roots[0], "fanloop-workflow"), []byte("user Skill"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			server := serveRelease(t, "1.2.2", validArchive(t, "1.2.2", "new binary", "# workflow\n", "# author\n"), nil)
			defer server.Close()
			options := Options{DataRoot: root, Client: server.Client(), LatestURL: server.URL + "/latest", Platform: Platform{OS: "linux", Arch: "amd64"}, SkillRoots: roots}
			if stage == "configuration" {
				options.BeforeActivate = func(string) error { return fmt.Errorf("configuration failed") }
			}
			if stage == "activation" {
				if os.Geteuid() == 0 {
					t.Skip("root bypasses permissions")
				}
				if err := os.Chmod(root, 0o555); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(root, 0o755) })
			}
			if _, err := Update(context.Background(), options); err == nil {
				t.Fatal("expected failure")
			}
			after, _ := os.Lstat(filepath.Join(root, "current"))
			newInfo, _ := os.Lstat(old)
			if !os.SameFile(before, after) || !os.SameFile(oldInfo, newInfo) {
				t.Fatal("original current/release inode changed")
			}
			if body, err := os.ReadFile(filepath.Join(old, "bin/fanloop")); err != nil || string(body) != "old binary" {
				t.Fatalf("active bytes changed: %q %v", body, err)
			}
			if body, err := os.ReadFile(filepath.Join(old, "user-marker")); err != nil || string(body) != "keep" {
				t.Fatalf("user bytes lost: %q %v", body, err)
			}
		})
	}
}

func TestUpdateManagedCurrentFailuresPreserveDirectory(t *testing.T) {
	for _, variant := range []string{"archive", "skill", "unknown", "escape", "newer", "configuration"} {
		t.Run(variant, func(t *testing.T) {
			root := t.TempDir()
			before := managedCurrent(t, root)
			archive := validArchive(t, "1.2.3", "new binary\n", "# workflow\n", "# author\n")
			if variant == "archive" {
				archive = []byte("invalid archive")
			}
			roots := testSkillRoots(t, root)
			if variant == "skill" {
				if err := os.WriteFile(filepath.Join(roots[0], "fanloop-workflow"), []byte("user data"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if variant == "unknown" {
				if err := os.WriteFile(filepath.Join(root, "current", "user-data"), []byte("keep"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if variant == "escape" {
				if err := os.Remove(filepath.Join(root, "current", "release")); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("../../outside", filepath.Join(root, "current", "release")); err != nil {
					t.Fatal(err)
				}
			}
			version := "1.2.3"
			if variant == "newer" {
				version = "1.2.1"
			}
			server := serveRelease(t, version, archive, nil)
			defer server.Close()
			options := Options{DataRoot: root, Client: server.Client(), LatestURL: server.URL + "/latest", Platform: Platform{OS: "linux", Arch: "amd64"}, SkillRoots: roots}
			if variant == "configuration" {
				options.BeforeActivate = func(string) error { return fmt.Errorf("configuration sync failed") }
			}
			_, err := Update(context.Background(), options)
			if variant == "newer" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil {
				t.Fatal("expected refusal")
			}
			if variant == "skill" && !strings.Contains(err.Error(), "non-Fanloop") {
				t.Fatalf("failure occurred before Skill validation: %v", err)
			}
			if variant == "configuration" && !strings.Contains(err.Error(), "configuration sync failed") {
				t.Fatalf("failure occurred before configuration sync: %v", err)
			}
			after, statErr := os.Lstat(filepath.Join(root, "current"))
			if statErr != nil || !os.SameFile(before, after) || !after.IsDir() {
				t.Fatalf("current changed: %v", statErr)
			}
			backups, _ := filepath.Glob(filepath.Join(root, "current.backup-*"))
			if len(backups) != 0 {
				t.Fatalf("failure left backups: %v", backups)
			}
		})
	}
}
