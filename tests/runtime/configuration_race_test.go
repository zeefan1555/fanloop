package runtime_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zeefan1555/fanloop/internal/configuration"
	"golang.org/x/sys/unix"
)

func TestReportKeepsSnapshotWhileAnotherInitRefreshes(t *testing.T) {
	binary := buildCLI(t)
	data := filepath.Join(filepath.Dir(binary), ".fanloop-user")
	source := filepath.Join(data, "source")
	git := func(args ...string) string {
		t.Helper()
		c := exec.Command("git", append([]string{"-C", source}, args...)...)
		out, err := c.CombinedOutput()
		if err != nil {
			t.Fatalf("fixture git: %v %s", err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-b", "main")
	git("config", "user.name", "Fixture")
	git("config", "user.email", "fixture@example.invalid")
	git("add", ".")
	git("commit", "-m", "A")
	fake := t.TempDir()
	if err := os.WriteFile(filepath.Join(fake, "lark-cli"), []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
		t.Fatal(err)
	}
	environment := append(withoutEnvironment(os.Environ(), "FANLOOP_CONFIG_SOURCE", "FANLOOP_DATA_HOME", "FANLOOP_NO_AUTO_CONFIG", "BOTMUX_CHAT_ID", "BOTMUX_SESSION_ID", "GIT_CONFIG_COUNT", "GIT_CONFIG_KEY_0", "GIT_CONFIG_VALUE_0", "GIT_CONFIG_KEY_1", "GIT_CONFIG_VALUE_1", "PATH"), "FANLOOP_DATA_HOME="+data, "GIT_CONFIG_COUNT=2", "GIT_CONFIG_KEY_0=url."+source+".insteadOf", "GIT_CONFIG_VALUE_0="+configuration.Repository, "GIT_CONFIG_KEY_1=url."+source+".insteadOf", "GIT_CONFIG_VALUE_1="+configuration.RepositorySSH, "PATH="+fake+string(os.PathListSeparator)+os.Getenv("PATH"))
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	command := func(args ...string) *exec.Cmd {
		c := exec.CommandContext(ctx, binary, args...)
		c.Env = environment
		return c
	}
	first, second := t.TempDir(), t.TempDir()
	if out, err := command("flow", "init", "--root", first, "--workflow", "linear-runtime", "--title", "A").CombinedOutput(); err != nil {
		t.Fatalf("init A: %v %s", err, out)
	}
	actualPin := strings.TrimSpace(string(readFile(t, filepath.Join(first, ".fanloop", "configuration-snapshot"))))
	stateLock, err := os.OpenFile(filepath.Join(first, ".fanloop", "flow", "state.lock"), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer stateLock.Close()
	if err = unix.Flock(int(stateLock.Fd()), unix.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	report := command("flow", "report", "result", "--root", first, "--step-id", "first", "--condition-result", `{"condition_id":"first_completed","output":{"type":"path","value":"artifact.md"}}`, "--next-step-id", "second", "--summary", "complete under A")
	var reportOutput bytes.Buffer
	report.Stdout = &reportOutput
	report.Stderr = &reportOutput
	if err = report.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		lock, err := os.Open(filepath.Join(actualPin, "fanloop-workflow"))
		if err != nil {
			t.Fatal(err)
		}
		err = unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		lock.Close()
		if err == unix.EWOULDBLOCK || err == unix.EAGAIN {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if time.Now().After(deadline) {
			t.Fatal("report did not hold the configuration binding while waiting for State")
		}
		time.Sleep(10 * time.Millisecond)
	}
	paths, err := filepath.Glob(filepath.Join(source, "fanloop-workflow", "linear-runtime", "*.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, bytes.ReplaceAll(content, []byte("first_completed"), []byte("first_done")), 0644); err != nil {
			t.Fatal(err)
		}
	}
	git("add", ".")
	git("commit", "-m", "B")
	newCommit := git("rev-parse", "HEAD")
	refresh := command("flow", "init", "--root", second, "--workflow", "linear-runtime", "--title", "B")
	var refreshOutput bytes.Buffer
	refresh.Stdout = &refreshOutput
	refresh.Stderr = &refreshOutput
	if err = refresh.Start(); err != nil {
		t.Fatal(err)
	}
	if err = refresh.Wait(); err != nil {
		t.Fatalf("refresh B while A is locked: %v %s", err, refreshOutput.String())
	}
	if current, _ := os.Readlink(filepath.Join(data, "config", "current")); filepath.Base(current) != newCommit {
		t.Fatal("unrelated current could not activate B while report waited on A")
	}
	if pin := strings.TrimSpace(string(readFile(t, filepath.Join(first, ".fanloop", "configuration-snapshot")))); pin != actualPin {
		t.Fatal("refresh replaced the actual pin used by report")
	}
	if err = unix.Flock(int(stateLock.Fd()), unix.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	if err = report.Wait(); err != nil {
		t.Fatalf("pending report mixed snapshots: %v %s", err, reportOutput.String())
	}
	if current, _ := os.Readlink(filepath.Join(data, "config", "current")); filepath.Base(current) != newCommit {
		t.Fatal("report changed the unrelated current after completing on A")
	}
}
