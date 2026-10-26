package configuration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSyncSkipsCheckoutWhenCommitMatches(t *testing.T) {
	remoteFixture(t)
	data := t.TempDir()
	ctx := context.Background()
	if _, done, err := Prepare(ctx, data, false); err != nil {
		t.Fatal(err)
	} else {
		done()
	}
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(t.TempDir(), "git.log")
	bin := t.TempDir()
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$*\" >> %q\nexec %q \"$@\"\n", logPath, realGit)
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	result, err := Sync(ctx, data)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Unchanged || len(result.WorkflowIDs) != 2 || result.CommitSHA == "" {
		t.Fatalf("sync = %+v", result)
	}
	if result.WorkflowIDs[0] != "linear-runtime" || result.WorkflowIDs[1] != "runtime-controls" {
		t.Fatalf("workflows = %#v", result.WorkflowIDs)
	}
	log, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(log)
	if !strings.Contains(text, "ls-remote") || strings.Contains(text, "fetch") || strings.Contains(text, "checkout") {
		t.Fatalf("download was not skipped:\n%s", text)
	}
}

func TestSyncSerializesConcurrentCalls(t *testing.T) {
	remoteFixture(t)
	data := t.TempDir()
	entered := make(chan int, 2)
	releaseFirst := make(chan struct{})
	releaseSecond := make(chan struct{})
	var mu sync.Mutex
	inside := 0
	nextID := 0
	previous := afterConfigurationLock
	afterConfigurationLock = func() {
		mu.Lock()
		inside++
		nextID++
		id := nextID
		if inside > 1 {
			t.Errorf("overlapping configuration syncs: %d", inside)
		}
		mu.Unlock()
		entered <- id
		if id == 1 {
			<-releaseFirst
		} else {
			<-releaseSecond
		}
		mu.Lock()
		inside--
		mu.Unlock()
	}
	t.Cleanup(func() { afterConfigurationLock = previous })

	errs := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := Sync(context.Background(), data)
			errs <- err
		}()
	}
	select {
	case id := <-entered:
		if id != 1 {
			t.Fatalf("first critical section = %d", id)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("first sync did not acquire the configuration lock")
	}
	select {
	case id := <-entered:
		t.Fatalf("second sync entered the critical section as %d while the first held the lock", id)
	case <-time.After(400 * time.Millisecond):
	}
	close(releaseFirst)
	select {
	case id := <-entered:
		if id != 2 {
			t.Fatalf("second critical section = %d", id)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("second sync did not acquire the lock after the first released it")
	}
	close(releaseSecond)
	for range 2 {
		select {
		case err := <-errs:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(20 * time.Second):
			t.Fatal("sync did not finish")
		}
	}
}

func TestSyncFetchFailureLeavesCurrentUnchanged(t *testing.T) {
	remoteFixture(t)
	data := t.TempDir()
	if _, err := Sync(context.Background(), data); err != nil {
		t.Fatal(err)
	}
	current := filepath.Join(data, "config", "current")
	before, err := os.Readlink(current)
	if err != nil {
		t.Fatal(err)
	}
	denyProductionRemotes(t)
	_, err = Sync(context.Background(), data)
	if err == nil || !errors.Is(err, ErrFetch) || !strings.Contains(err.Error(), "HTTPS Git credentials") || !strings.Contains(err.Error(), "SSH") {
		t.Fatalf("sync error = %v", err)
	}
	after, err := os.Readlink(current)
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("current changed from %s to %s", before, after)
	}
}

func TestSyncPrefersFileHTTPSRemoteOverSSH(t *testing.T) {
	httpsRepo := sourceRepository(t)
	sshRepo := sourceRepository(t)
	if err := os.WriteFile(filepath.Join(sshRepo, Category, "linear-runtime", "ssh-only.txt"), []byte("ssh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	command(t, sshRepo, "add", ".")
	command(t, sshRepo, "commit", "-m", "ssh only")
	httpsCommit := strings.TrimSpace(command(t, httpsRepo, "rev-parse", "HEAD"))
	sshCommit := strings.TrimSpace(command(t, sshRepo, "rev-parse", "HEAD"))
	if httpsCommit == "" || httpsCommit == sshCommit {
		t.Fatalf("https=%q ssh=%q", httpsCommit, sshCommit)
	}
	notice := captureNotice(t)
	writeGitInsteadOf(t, [2]string{fileRemote(httpsRepo), Repository}, [2]string{fileRemote(sshRepo), RepositorySSH})
	unsetConfigSource(t)
	logPath := logGitCommands(t)

	data := t.TempDir()
	result, err := Sync(context.Background(), data)
	if err != nil {
		t.Fatal(err)
	}
	if result.Unchanged || result.CommitSHA != httpsCommit {
		t.Fatalf("sync = %+v, https=%s ssh=%s", result, httpsCommit, sshCommit)
	}
	if got := recordedRepository(t, data); got != Repository {
		t.Fatalf("recorded repository = %s", got)
	}
	if notice.Len() != 0 {
		t.Fatalf("notice = %q", notice.String())
	}
	log := readGitLog(t, logPath)
	if !strings.Contains(log, "ls-remote "+Repository+" ") || !strings.Contains(log, "fetch --quiet --depth=1 --no-tags "+Repository+" ") || strings.Contains(log, RepositorySSH) {
		t.Fatalf("remote order:\n%s", log)
	}
}

func TestSyncFallsBackToSSHFileRemoteOnce(t *testing.T) {
	sshRepo := sourceRepository(t)
	want := strings.TrimSpace(command(t, sshRepo, "rev-parse", "HEAD"))
	notice := captureNotice(t)
	missing := filepath.Join(t.TempDir(), "missing-https")
	writeGitInsteadOf(t, [2]string{fileRemote(missing), Repository}, [2]string{fileRemote(sshRepo), RepositorySSH})
	unsetConfigSource(t)
	logPath := logGitCommands(t)

	data := t.TempDir()
	result, err := Sync(context.Background(), data)
	if err != nil {
		t.Fatal(err)
	}
	if result.Unchanged || result.CommitSHA != want {
		t.Fatalf("sync = %+v, want %s", result, want)
	}
	if got := recordedRepository(t, data); got != RepositorySSH {
		t.Fatalf("recorded repository = %s", got)
	}
	if notice.String() != remoteFallbackNotice+"\n" {
		t.Fatalf("notice = %q", notice.String())
	}
	log := readGitLog(t, logPath)
	if strings.Count(log, "ls-remote "+Repository+" ") != 1 || strings.Count(log, "ls-remote "+RepositorySSH+" ") != 1 {
		t.Fatalf("ls-remote order:\n%s", log)
	}
	if strings.Contains(log, "fetch --quiet --depth=1 --no-tags "+Repository+" ") || !strings.Contains(log, "fetch --quiet --depth=1 --no-tags "+RepositorySSH+" ") {
		t.Fatalf("fetch order:\n%s", log)
	}
}

func TestSSHSnapshotIsReusedWithoutFetch(t *testing.T) {
	remoteFixture(t)
	data := t.TempDir()
	ctx := context.Background()
	first, err := Sync(ctx, data)
	if err != nil {
		t.Fatal(err)
	}
	current, err := os.Readlink(filepath.Join(data, "config", "current"))
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(current, "configuration.json")
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `"`+Repository+`"`) {
		t.Fatalf("snapshot did not record HTTPS: %s", content)
	}
	if err = os.Chmod(manifestPath, 0o644); err != nil {
		t.Fatal(err)
	}
	rewritten := strings.Replace(string(content), `"`+Repository+`"`, `"`+RepositorySSH+`"`, 1)
	if err = os.WriteFile(manifestPath, []byte(rewritten), 0o444); err != nil {
		t.Fatal(err)
	}
	if _, err = Read(current); err != nil {
		t.Fatal(err)
	}
	report, err := Drift(ctx, data)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != DriftInSync || report.ConfigurationCommit != first.CommitSHA || report.RemoteCommit != first.CommitSHA {
		t.Fatalf("drift = %+v", report)
	}
	notice := captureNotice(t)
	logPath := logGitCommands(t)
	second, err := Sync(ctx, data)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Unchanged || second.CommitSHA != first.CommitSHA {
		t.Fatalf("second = %+v", second)
	}
	again, err := os.Readlink(filepath.Join(data, "config", "current"))
	if err != nil || again != current {
		t.Fatalf("current = %s, want %s (%v)", again, current, err)
	}
	body, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != rewritten {
		t.Fatal("snapshot repository was rewritten")
	}
	log := readGitLog(t, logPath)
	if !strings.Contains(log, "ls-remote "+Repository+" ") || strings.Contains(log, "fetch") || strings.Contains(log, "checkout") {
		t.Fatalf("download was not skipped:\n%s", log)
	}
	if notice.Len() != 0 {
		t.Fatalf("notice = %q", notice.String())
	}

	if err = os.Remove(filepath.Join(data, "config", "current")); err != nil {
		t.Fatal(err)
	}
	republished, err := Sync(ctx, data)
	if err != nil {
		t.Fatal(err)
	}
	if republished.Unchanged || republished.CommitSHA != first.CommitSHA {
		t.Fatalf("republish = %+v", republished)
	}
	restored, err := os.Readlink(filepath.Join(data, "config", "current"))
	if err != nil || restored != current {
		t.Fatalf("restored = %s, want %s (%v)", restored, current, err)
	}
	kept, err := os.ReadFile(manifestPath)
	if err != nil || string(kept) != rewritten {
		t.Fatal("publish rewrote the SSH repository URL")
	}

	if err = os.Chmod(manifestPath, 0o644); err != nil {
		t.Fatal(err)
	}
	other := strings.Replace(rewritten, RepositorySSH, "https://github.com/example/other.git", 1)
	if err = os.WriteFile(manifestPath, []byte(other), 0o444); err != nil {
		t.Fatal(err)
	}
	if _, err = Read(current); err != nil {
		t.Fatal(err)
	}
}

func TestHTTPSTimeoutFallsBackOnce(t *testing.T) {
	previous := httpsAttemptTimeout
	httpsAttemptTimeout = 2 * time.Second
	t.Cleanup(func() { httpsAttemptTimeout = previous })
	sshRepo := sourceRepository(t)
	want := strings.TrimSpace(command(t, sshRepo, "rev-parse", "HEAD"))
	notice := captureNotice(t)
	writeGitInsteadOf(t, [2]string{fileRemote(sshRepo), RepositorySSH})
	unsetConfigSource(t)
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	script := fmt.Sprintf("#!/bin/sh\ncase \"$*\" in\n*%s*) exec sleep 5 ;;\nesac\nexec %q \"$@\"\n", Repository, realGit)
	if err = os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	data := t.TempDir()
	started := time.Now()
	result, err := Sync(context.Background(), data)
	elapsed := time.Since(started)
	if err != nil {
		t.Fatal(err)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("HTTPS attempt took %s", elapsed)
	}
	if result.Unchanged || result.CommitSHA != want {
		t.Fatalf("sync = %+v, want %s", result, want)
	}
	if notice.String() != remoteFallbackNotice+"\n" {
		t.Fatalf("notice = %q", notice.String())
	}
	if got := recordedRepository(t, data); got != RepositorySSH {
		t.Fatalf("recorded repository = %s", got)
	}
}

func recordedRepository(t *testing.T, data string) string {
	t.Helper()
	current, err := os.Readlink(filepath.Join(data, "config", "current"))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := Read(current)
	if err != nil {
		t.Fatal(err)
	}
	return manifest.Repository
}

func logGitCommands(t *testing.T) string {
	t.Helper()
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(t.TempDir(), "git.log")
	bin := t.TempDir()
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$*\" >> %q\nexec %q \"$@\"\n", logPath, realGit)
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return logPath
}

func readGitLog(t *testing.T, path string) string {
	t.Helper()
	log, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(log)
}
