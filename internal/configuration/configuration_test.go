package configuration

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"golang.org/x/sys/unix"
)

func command(t *testing.T, root string, args ...string) string {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = root
	b, e := c.CombinedOutput()
	if e != nil {
		t.Fatalf("git %v: %s %v", args, b, e)
	}
	return string(b)
}
func sourceRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	fixture := filepath.Join("..", "workflow", "testdata", "linear-runtime")
	target := filepath.Join(root, "fanloop-workflow", "linear-runtime")
	err := filepath.WalkDir(fixture, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(fixture, path)
		dst := filepath.Join(target, rel)
		if entry.IsDir() {
			return os.MkdirAll(dst, 0755)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, b, 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(filepath.Join(root, "fanloop-workflow", "runtime-controls"), os.DirFS(filepath.Join("..", "workflow", "testdata", "runtime-controls"))); err != nil {
		t.Fatal(err)
	}
	command(t, root, "init", "-b", "main")
	command(t, root, "config", "user.name", "Configuration Test")
	command(t, root, "config", "user.email", "config@example.test")
	command(t, root, "add", ".")
	command(t, root, "commit", "-m", "fixture")
	return root
}
func remoteFixture(t *testing.T) string {
	t.Helper()
	root := sourceRepository(t)
	writeGitInsteadOf(t, [2]string{root, Repository}, [2]string{root, RepositorySSH})
	unsetConfigSource(t)
	return root
}

// writeGitInsteadOf replaces the process Git config so a user global insteadOf
// cannot rewrite both production URLs back to HTTPS.
func writeGitInsteadOf(t *testing.T, pairs ...[2]string) {
	t.Helper()
	var builder strings.Builder
	for _, pair := range pairs {
		builder.WriteString("[url \"")
		builder.WriteString(pair[0])
		builder.WriteString("\"]\n\tinsteadOf = ")
		builder.WriteString(pair[1])
		builder.WriteString("\n")
	}
	path := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(path, []byte(builder.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", path)
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	t.Setenv("GIT_CONFIG_COUNT", "0")
}

func unsetConfigSource(t *testing.T) {
	t.Helper()
	t.Setenv("FANLOOP_CONFIG_SOURCE", "")
	if err := os.Unsetenv("FANLOOP_CONFIG_SOURCE"); err != nil {
		t.Fatal(err)
	}
}

func denyProductionRemotes(t *testing.T) {
	t.Helper()
	missingHTTPS := filepath.Join(t.TempDir(), "missing-https")
	missingSSH := filepath.Join(t.TempDir(), "missing-ssh")
	writeGitInsteadOf(t, [2]string{missingHTTPS, Repository}, [2]string{missingSSH, RepositorySSH})
}

func captureNotice(t *testing.T) *bytes.Buffer {
	t.Helper()
	previous := RemoteNotice
	var buf bytes.Buffer
	RemoteNotice = &buf
	t.Cleanup(func() { RemoteNotice = previous })
	return &buf
}

func fileRemote(path string) string {
	return "file://" + path
}
func TestSnapshotRefreshDryRunAndFailure(t *testing.T) {
	source := remoteFixture(t)
	data := t.TempDir()
	ctx := context.Background()
	root, done, err := Prepare(ctx, data, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(root); err != nil {
		t.Fatal(err)
	}
	done()
	current := filepath.Join(data, "config", "current")
	if _, err = os.Lstat(current); !os.IsNotExist(err) {
		t.Fatalf("dry run activated: %v", err)
	}
	root, done, err = Prepare(ctx, data, false)
	if err != nil {
		t.Fatal(err)
	}
	first := filepath.Dir(root)
	done()
	initial, err := Read(first)
	if err != nil {
		t.Fatal(err)
	}
	if initial.Repository != Repository || initial.Ref != Ref {
		t.Fatalf("configuration identity = %s %s", initial.Repository, initial.Ref)
	}
	for _, file := range initial.Files {
		if !strings.HasPrefix(file.Path, "fanloop-workflow/") {
			t.Fatalf("configuration file escaped category: %s", file.Path)
		}
	}
	root, done, err = Prepare(ctx, data, false)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(root) != first {
		t.Fatal("same commit did not reuse snapshot")
	}
	done()
	extra := filepath.Join(source, Category, "linear-runtime", "skills", "test-guide", "run.sh")
	if err = os.WriteFile(extra, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	command(t, source, "add", ".")
	command(t, source, "commit", "-m", "executable")
	root, done, err = Prepare(ctx, data, false)
	if err != nil {
		t.Fatal(err)
	}
	second := filepath.Dir(root)
	done()
	updated, err := Read(second)
	if err != nil {
		t.Fatal(err)
	}
	if initial.CommitSHA == updated.CommitSHA || initial.SHA256 == updated.SHA256 {
		t.Fatal("identity did not advance")
	}
	info, err := os.Stat(filepath.Join(root, "linear-runtime", "skills", "test-guide", "run.sh"))
	if err != nil || info.Mode()&0111 == 0 {
		t.Fatal("executable bit lost")
	}
	command(t, source, "rm", filepath.Join(Category, "linear-runtime", "skills", "test-guide", "run.sh"))
	command(t, source, "rm", "-r", filepath.Join(Category, "runtime-controls"))
	command(t, source, "commit", "-m", "delete")
	root, done, err = Prepare(ctx, data, false)
	if err != nil {
		t.Fatal(err)
	}
	done()
	if _, err = os.Stat(filepath.Join(root, "linear-runtime", "skills", "test-guide", "run.sh")); !os.IsNotExist(err) {
		t.Fatal("deleted resource persisted")
	}
	if _, err = os.Stat(filepath.Join(root, "runtime-controls")); !os.IsNotExist(err) {
		t.Fatal("removed Workflow persisted in snapshot")
	}
	prior, _ := os.Readlink(current)
	if err = os.WriteFile(filepath.Join(source, Category, "linear-runtime", "workflow.yaml"), []byte("invalid: true\n"), 0644); err != nil {
		t.Fatal(err)
	}
	command(t, source, "add", ".")
	command(t, source, "commit", "-m", "invalid")
	if _, _, err = Prepare(ctx, data, false); err == nil {
		t.Fatal("invalid package accepted")
	}
	after, _ := os.Readlink(current)
	if after != prior {
		t.Fatal("invalid candidate replaced current")
	}
	denyProductionRemotes(t)
	if _, _, err = Prepare(ctx, data, false); err == nil {
		t.Fatal("fetch failure accepted")
	}
	after, _ = os.Readlink(current)
	if after != prior {
		t.Fatal("fetch failure replaced current")
	}
}
func TestImportRequiresCleanCommitAndRejectsSymlink(t *testing.T) {
	source := sourceRepository(t)
	data := t.TempDir()
	ctx := context.Background()
	snapshot, err := Import(ctx, data, source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Read(snapshot); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(source, Category, "linear-runtime", "skills", "test-guide", "escape")
	if err = os.Symlink("/etc/passwd", path); err != nil {
		t.Fatal(err)
	}
	if _, err = Import(ctx, data, source); err == nil {
		t.Fatal("dirty source accepted")
	}
	command(t, source, "add", ".")
	command(t, source, "commit", "-m", "symlink")
	if _, err = Import(ctx, data, source); err == nil {
		t.Fatal("symlink source accepted")
	}
}
func TestConcurrentRefreshAndActivationFailure(t *testing.T) {
	remoteFixture(t)
	data := t.TempDir()
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, done, err := Prepare(ctx, data, false)
			if err != nil {
				t.Error(err)
				return
			}
			done()
		}()
	}
	wg.Wait()
	target, err := os.Readlink(filepath.Join(data, "config", "current"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Read(target); err != nil {
		t.Fatal(err)
	}
	other := t.TempDir()
	if err = os.MkdirAll(filepath.Join(other, "config", "current"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, _, err = Prepare(ctx, other, false); err == nil {
		t.Fatal("replaced non-link current")
	}
	if info, err := os.Stat(filepath.Join(other, "config", "current")); err != nil || !info.IsDir() {
		t.Fatal("activation damaged current")
	}
}

func TestCurrentReaderSerializesActivationAndChecksIdentity(t *testing.T) {
	remoteFixture(t)
	data := t.TempDir()
	ctx := context.Background()
	root, done, err := Prepare(ctx, data, false)
	if err != nil {
		t.Fatal(err)
	}
	done()
	reader, err := PrepareCurrent(ctx, data)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(filepath.Join(data, ".workflows.lock"), os.O_RDWR, 0)
	if err != nil {
		reader()
		t.Fatal(err)
	}
	err = unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	file.Close()
	if err == nil {
		reader()
		t.Fatal("activation can switch current while a reader is still consuming it")
	}
	reader()
	path := filepath.Join(root, "linear-runtime", "SKILL.md")
	if err = os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, []byte("# Valid but modified Skill\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err = PrepareCurrent(ctx, data); err == nil {
		t.Fatal("accepted modified immutable snapshot")
	}
}

func TestExplicitManagedSnapshotCannotLoseItsIdentity(t *testing.T) {
	remoteFixture(t)
	data := t.TempDir()
	ctx := context.Background()
	root, done, err := Prepare(ctx, data, false)
	if err != nil {
		t.Fatal(err)
	}
	done()
	source := filepath.Dir(root)
	t.Setenv("FANLOOP_CONFIG_SOURCE", source)
	if err = os.Remove(filepath.Join(source, "configuration.json")); err != nil {
		t.Fatal(err)
	}
	if unlock, err := PrepareCurrent(ctx, data); err == nil {
		unlock()
		t.Fatal("explicit managed snapshot became an unchecked author source")
	}
	if _, unlock, err := Prepare(ctx, data, true); err == nil {
		unlock()
		t.Fatal("init accepted a managed snapshot without identity")
	}
}

func TestSnapshotCommitMustMatchDirectoryIdentity(t *testing.T) {
	source := sourceRepository(t)
	snapshot, err := Import(context.Background(), t.TempDir(), source)
	if err != nil {
		t.Fatal(err)
	}
	original, err := Read(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	for _, commit := range []string{strings.Repeat("1", 40), strings.Repeat("z", 40), "short"} {
		t.Run(commit, func(t *testing.T) {
			changed := original
			changed.CommitSHA = commit
			content, err := json.Marshal(changed)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(snapshot, "configuration.json")
			if err := os.Chmod(path, 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, content, 0444); err != nil {
				t.Fatal(err)
			}
			if _, err := Read(snapshot); err == nil {
				t.Fatal("accepted a mismatched snapshot commit")
			}
		})
	}
}
