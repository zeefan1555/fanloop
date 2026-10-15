package configuration

import (
	"context"
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
