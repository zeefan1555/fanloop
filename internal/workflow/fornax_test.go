package workflow

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestDashboardCandidateRejectsLiveAndSymlinkedSources(t *testing.T) {
	dataRoot := t.TempDir()
	live := filepath.Join(dataRoot, "current")
	if err := os.MkdirAll(filepath.Join(live, "workflows"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateDashboardCandidate(dataRoot, live); err == nil {
		t.Fatal("live configuration accepted as an editable candidate")
	}

	candidate := t.TempDir()
	if err := os.CopyFS(filepath.Join(candidate, "workflows", "runtime-controls"), os.DirFS(filepath.Join("testdata", "runtime-controls"))); err != nil {
		t.Fatal(err)
	}
	resolved, err := ValidateDashboardCandidate(dataRoot, candidate)
	want, resolveErr := filepath.EvalSymlinks(candidate)
	if resolveErr != nil {
		t.Fatal(resolveErr)
	}
	if err != nil || resolved != want {
		t.Fatalf("candidate = %q, %v", resolved, err)
	}

	link := filepath.Join(t.TempDir(), "candidate")
	if err := os.Symlink(candidate, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateDashboardCandidate(dataRoot, link); err == nil {
		t.Fatal("symlinked candidate accepted")
	}
}

func TestConfigurationSourceRejectsGlobalLiveRootFromIsolatedDataRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	globalCurrent := filepath.Join(home, ".fanloop", "current")
	if err := os.CopyFS(filepath.Join(globalCurrent, "workflows", "runtime-controls"), os.DirFS(filepath.Join("testdata", "runtime-controls"))); err != nil {
		t.Fatal(err)
	}
	isolatedDataRoot := t.TempDir()

	if _, err := ValidateDashboardCandidate(isolatedDataRoot, globalCurrent); err == nil {
		t.Fatal("global live configuration accepted as an editable Dashboard candidate")
	}
}

func TestUpdateFornaxDoesNotHoldExclusiveLockDuringRemotePreparation(t *testing.T) {
	dataRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataRoot, "fornax.json"), []byte(`{"workspace_id":"1","workflows":[{"id":"runtime-controls","skill_key":"fanloop.runtime-controls","skill_id":"2"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	started := filepath.Join(t.TempDir(), "started")
	release := filepath.Join(t.TempDir(), "release")
	script := filepath.Join(bin, "fornax-cli")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nset -eu\ntouch \"$FANLOOP_TEST_STARTED\"\nwhile [ ! -e \"$FANLOOP_TEST_RELEASE\" ]; do sleep 0.01; done\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FANLOOP_TEST_STARTED", started)
	t.Setenv("FANLOOP_TEST_RELEASE", release)

	updateDone := make(chan error, 1)
	go func() { _, err := UpdateFornax(context.Background(), dataRoot); updateDone <- err }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(started); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("fake Fornax command did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}

	lockDone := make(chan func(), 1)
	go func() {
		unlock, _ := LockConfiguration(dataRoot, false)
		lockDone <- unlock
	}()
	blocked := false
	select {
	case unlock := <-lockDone:
		unlock()
	case <-time.After(250 * time.Millisecond):
		blocked = true
	}
	if err := os.WriteFile(release, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := <-updateDone; err == nil {
		t.Fatal("fake failed Fornax command unexpectedly succeeded")
	}
	if blocked {
		unlock := <-lockDone
		unlock()
		t.Fatal("remote preparation held the exclusive configuration lock")
	}
}
