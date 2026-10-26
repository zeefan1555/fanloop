package configuration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func wrapGit(t *testing.T, body string) string {
	t.Helper()
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	log := filepath.Join(bin, "calls")
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$1\" >> %q\n%s\nexec %q \"$@\"\n", log, body, realGit)
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

func TestFetchRegressionCanceledParentDoesNotRetry(t *testing.T) {
	remoteFixture(t)
	notice := captureNotice(t)
	log := wrapGit(t, "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Sync(ctx, t.TempDir())
	if !errors.Is(err, ErrFetch) || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost its cause: %v", err)
	}
	if notice.Len() != 0 {
		t.Fatalf("canceled parent started fallback: %q", notice.String())
	}
	if data, err := os.ReadFile(log); err == nil && len(data) != 0 {
		t.Fatalf("canceled parent ran Git: %q", data)
	}
}

func TestFetchRegressionParentCanceledDuringFetchDoesNotRetry(t *testing.T) {
	remoteFixture(t)
	notice := captureNotice(t)
	marker := filepath.Join(t.TempDir(), "fetch-started")
	log := wrapGit(t, fmt.Sprintf("if [ \"$1\" = fetch ]; then printf interrupted > .git/shallow.lock; printf ready > %q; exec sleep 5; fi", marker))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	data := t.TempDir()
	finished := make(chan error, 1)
	go func() {
		_, done, err := Prepare(ctx, data, false)
		if done != nil {
			done()
		}
		finished <- err
	}()
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	waiting := true
	for waiting {
		select {
		case <-deadline.C:
			t.Fatal("fetch did not create its interrupted lock")
		case <-tick.C:
			if _, err := os.Stat(marker); err == nil {
				waiting = false
			}
		}
	}
	cancel()
	select {
	case err := <-finished:
		if !errors.Is(err, ErrFetch) || !errors.Is(err, context.Canceled) {
			t.Fatalf("active fetch cancellation lost its cause: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("active cancellation did not return")
	}
	if notice.Len() != 0 || strings.Count(readGitLog(t, log), "fetch\n") != 1 {
		t.Fatal("canceled active fetch started an invalid retry")
	}
}

func TestFetchRegressionHealthyFetchExceedsProbeBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("real Git fetch delayed 16 seconds to exercise the production budget")
	}
	source := sourceRepository(t)
	want := strings.TrimSpace(command(t, source, "rev-parse", "HEAD"))
	writeGitInsteadOf(t, [2]string{fileRemote(source), Repository}, [2]string{fileRemote(source), RepositorySSH})
	unsetConfigSource(t)
	notice := captureNotice(t)
	wrapGit(t, "if [ \"$1\" = fetch ]; then sleep 16; fi")
	data := t.TempDir()
	started := time.Now()
	result, err := Sync(context.Background(), data)
	if err != nil {
		t.Fatalf("healthy fetch failed: %v", err)
	}
	if time.Since(started) < 16*time.Second || result.CommitSHA != want {
		t.Fatalf("real fetch did not publish the expected commit: %+v", result)
	}
	if notice.Len() != 0 || recordedRepository(t, data) != Repository {
		t.Fatalf("healthy HTTPS fetch was incorrectly interrupted: %q", notice.String())
	}
	current, err := os.Readlink(filepath.Join(data, "config", "current"))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := Read(current)
	if err != nil || manifest.CommitSHA != want || manifest.CommitSHA != result.CommitSHA {
		t.Fatalf("published manifest differs from real Git HEAD: %+v, %v", manifest, err)
	}
}

func TestFetchRegressionFailedAttemptLockDoesNotPoisonRetry(t *testing.T) {
	source := sourceRepository(t)
	want := strings.TrimSpace(command(t, source, "rev-parse", "HEAD"))
	writeGitInsteadOf(t, [2]string{fileRemote(source), Repository}, [2]string{fileRemote(source), RepositorySSH})
	unsetConfigSource(t)
	captureNotice(t)
	userLock := filepath.Join(source, ".git", "shallow.lock")
	if err := os.WriteFile(userLock, []byte("user owned lock"), 0o600); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf("if [ \"$1\" = fetch ]; then\n case \"$*\" in\n *%s*) printf 'interrupted attempt' > .git/shallow.lock; printf 'fatal: connection reset\\n' >&2; exit 128 ;;\n esac\nfi", Repository)
	wrapGit(t, body)
	data := t.TempDir()
	result, err := Sync(context.Background(), data)
	if err != nil || result.CommitSHA != want {
		t.Fatalf("failed attempt lock poisoned real retry: %+v, %v", result, err)
	}
	if recordedRepository(t, data) != RepositorySSH {
		t.Fatal("successful retry did not record SSH")
	}
	if content, err := os.ReadFile(userLock); err != nil || string(content) != "user owned lock" {
		t.Fatal("user lock was changed")
	}
}

func TestFetchRegressionTimedOutAttemptLockDoesNotPoisonRetry(t *testing.T) {
	source := sourceRepository(t)
	want := strings.TrimSpace(command(t, source, "rev-parse", "HEAD"))
	writeGitInsteadOf(t, [2]string{fileRemote(source), Repository}, [2]string{fileRemote(source), RepositorySSH})
	previous := fetchAttemptTimeout
	fetchAttemptTimeout = 2 * time.Second
	t.Cleanup(func() { fetchAttemptTimeout = previous })
	captureNotice(t)
	marker := filepath.Join(t.TempDir(), "interrupted-lock")
	body := fmt.Sprintf("if [ \"$1\" = fetch ]; then\n case \"$*\" in\n *%s*) printf 'interrupted' > .git/shallow.lock; test -f .git/shallow.lock && printf 'lock existed' > %q; exec sleep 30 ;;\n esac\nfi", Repository, marker)
	wrapGit(t, body)
	staging := t.TempDir()
	started := time.Now()
	commit, remote, err := checkout(withRemoteSession(context.Background()), staging, Repository, Ref)
	if err != nil || commit != want || remote != RepositorySSH {
		t.Fatalf("timed out attempt lock poisoned real retry: commit=%s remote=%s error=%v", commit, remote, err)
	}
	if content, err := os.ReadFile(marker); err != nil || string(content) != "lock existed" {
		t.Fatal("timed out fetch did not create its real lock")
	}
	if elapsed := time.Since(started); elapsed < 2*time.Second || elapsed > 20*time.Second {
		t.Fatalf("fetch did not time out before its 30-second wait: %s", elapsed)
	}
}

func TestFetchRegressionDeadlineRetainsCauseAndBoundsPipes(t *testing.T) {
	root := t.TempDir()
	command(t, root, "init", "--quiet")
	wrapGit(t, "if [ \"$1\" = fetch ]; then printf interrupted > .git/shallow.lock; sleep 4 & exec sleep 5; fi")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	started := time.Now()
	_, err := git(ctx, root, "fetch", Repository, Ref)
	if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("deadline lost its safe classification or cause: %v", err)
	}
	if time.Since(started) > 3*time.Second {
		t.Fatal("orphan-held pipes exceeded the cancellation bound")
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "shallow.lock")); err != nil {
		t.Fatal("interrupted fetch did not leave the real lock fixture")
	}
}

func TestFetchRegressionBothFailuresKeepCurrentAndClassify(t *testing.T) {
	source := remoteFixture(t)
	data := t.TempDir()
	if _, err := Sync(context.Background(), data); err != nil {
		t.Fatal(err)
	}
	current := filepath.Join(data, "config", "current")
	before, err := os.Readlink(current)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, Category, "linear-runtime", "new.txt"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	command(t, source, "add", ".")
	command(t, source, "commit", "-m", "new fixture\n\nCo-Authored-By: Aiden")
	captureNotice(t)
	body := fmt.Sprintf("if [ \"$1\" = fetch ]; then\n case \"$*\" in\n *%s*) printf 'fatal: Authentication failed for https://credential:secret@example.test/repo\\n' >&2; exit 128 ;;\n *) printf 'held' > .git/shallow.lock ;;\n esac\nfi", Repository)
	wrapGit(t, body)
	_, err = Sync(context.Background(), data)
	if !errors.Is(err, ErrFetch) || !strings.Contains(err.Error(), "HTTPS: authentication") || !strings.Contains(err.Error(), "SSH: lock conflict") {
		t.Fatalf("transport failure classifications lost: %v", err)
	}
	if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "example.test") {
		t.Fatal("credential-bearing stderr escaped")
	}
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatal("underlying Git exit error was lost")
	}
	after, err := os.Readlink(current)
	if err != nil || after != before {
		t.Fatal("failed transports changed current")
	}
}

func TestGitFailureClassifiesWithoutPrintingDiagnostics(t *testing.T) {
	for _, test := range []struct {
		stderr string
		kind   string
	}{
		{"fatal: unable to create '.git/shallow.lock': Permission denied", "lock conflict"},
		{"fatal: could not read Password: terminal prompts disabled", "authentication"},
		{"fatal: connection timed out", "timeout"},
		{"fatal: could not resolve host", "network"},
		{"fatal: unexpected protocol failure", "other"},
	} {
		t.Run(test.kind, func(t *testing.T) {
			exit := &exec.ExitError{Stderr: []byte(test.stderr + " https://credential:secret@example.test/repo")}
			err := gitFailure("fetch", context.Background(), exit)
			if !strings.HasPrefix(err.Error(), test.kind) || strings.Contains(err.Error(), "secret") || !errors.Is(err, exit) {
				t.Fatalf("safe classification = %v", err)
			}
		})
	}
}
