package fornaxmirror

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/workflow"
	"golang.org/x/sys/unix"
)

const personalReleaseDigest = "8ad2b03e5cbf97096c3fa421c6ea9cd5a273e87cef72c330fff555a22119de99"

type syncClient struct {
	personalVersion string
	loopVersion     string
	installs        int
	failInstall     bool
	mutate          bool
	badReadOnly     bool
	badReadWrite    bool
	lockPath        string
	lockChecked     bool
}

func (client *syncClient) Skills(context.Context, string, string) ([]Skill, error) {
	return nil, errors.New("exact key lookup is not used by sync")
}

func (client *syncClient) ListByPrefix(_ context.Context, workspaceID, prefix string) ([]Skill, error) {
	if client.lockPath != "" {
		lock, err := os.OpenFile(client.lockPath, os.O_RDWR, 0)
		if err != nil {
			return nil, err
		}
		defer lock.Close()
		if err = unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err == nil {
			_ = unix.Flock(int(lock.Fd()), unix.LOCK_UN)
			return nil, errors.New("configuration lock was not held")
		}
		client.lockChecked = true
	}
	if prefix != "fanloop.workflow." && prefix != "loop.workflow." {
		return nil, errors.New("unexpected prefix " + prefix)
	}
	return []Skill{{ID: workspaceID + "-skill", Key: prefix + "linear_runtime"}}, nil
}

func (client *syncClient) Releases(_ context.Context, _, skillID string) ([]Release, error) {
	version := client.personalVersion
	if skillID == "2-skill" {
		version = client.loopVersion
	}
	return []Release{{Version: "9.0.0", UpdatedAt: 1}, {Version: version, UpdatedAt: 20}}, nil
}

func (client *syncClient) Install(_ context.Context, _, key, _, directory string) (string, error) {
	client.installs++
	if client.failInstall {
		return "", errors.New("offline")
	}
	name := "loop-workflow-linear-runtime"
	wrapperName := name
	if strings.HasPrefix(key, "fanloop.workflow.") {
		name = "fanloop-workflow-linear-runtime"
		wrapperName = name
	}
	if strings.HasPrefix(key, "loop.workflow.") && client.badReadOnly {
		wrapperName = "other"
	}
	if strings.HasPrefix(key, "fanloop.workflow.") && client.badReadWrite {
		wrapperName = "other"
	}
	root := filepath.Join(directory, name)
	if err := copyLinearPackage(root, wrapperName); err != nil {
		return "", err
	}
	if client.mutate {
		path := filepath.Join(root, "linear-runtime", "workflow.yaml")
		content, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		if err = os.WriteFile(path, append(content, []byte("\n# mutated\n")...), 0644); err != nil {
			return "", err
		}
	}
	return root, nil
}

func TestSyncKeepsPersonalReleaseAndSkipsSecondDownload(t *testing.T) {
	dataRoot := t.TempDir()
	writeFornaxSources(t, dataRoot)
	client := &syncClient{personalVersion: "1.0.0", loopVersion: "1.0.2", lockPath: filepath.Join(dataRoot, ".workflows.lock")}
	result, err := Sync(context.Background(), dataRoot, client)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != OutcomeUpdated || result.Digest != personalReleaseDigest || len(result.Workflows) != 1 || result.Workflows[0].ID != "linear-runtime" || result.Workflows[0].Key != "fanloop.workflow.linear_runtime" || result.Workflows[0].Version != "1.0.0" {
		t.Fatalf("result = %#v", result)
	}
	if !client.lockChecked || client.installs != 2 {
		t.Fatalf("lockChecked=%t installs=%d", client.lockChecked, client.installs)
	}
	current, err := os.Readlink(filepath.Join(dataRoot, "config", "current"))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(current) != personalReleaseDigest {
		t.Fatalf("snapshot = %s", filepath.Base(current))
	}
	loaded, err := workflow.LoadFrom(filepath.Join(current, "fanloop-workflow"), "linear-runtime")
	if err != nil || loaded.Ref.ID != "linear-runtime" {
		t.Fatalf("loaded = %#v, err = %v", loaded.Ref, err)
	}
	info, err := os.Stat(filepath.Join(current, "fanloop-workflow", "linear-runtime", "skills", "test-guide", "scripts", "run.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0555 {
		t.Fatalf("script mode = %#o", info.Mode().Perm())
	}
	again, err := Sync(context.Background(), dataRoot, client)
	if err != nil {
		t.Fatal(err)
	}
	if again.Outcome != OutcomeUnchanged || again.Digest != personalReleaseDigest || client.installs != 2 || len(again.Workflows) != 1 || again.Workflows[0].ID != "linear-runtime" || again.Workflows[0].Version != "1.0.0" {
		t.Fatalf("again = %#v installs=%d", again, client.installs)
	}
}

func TestSyncFailureKeepsCurrent(t *testing.T) {
	dataRoot := t.TempDir()
	writeFornaxSources(t, dataRoot)
	client := &syncClient{personalVersion: "1.0.0", loopVersion: "1.0.2"}
	if _, err := Sync(context.Background(), dataRoot, client); err != nil {
		t.Fatal(err)
	}
	client.failInstall = true
	client.loopVersion = "1.0.3"
	if _, err := Sync(context.Background(), dataRoot, client); err == nil || !strings.Contains(err.Error(), "download") {
		t.Fatalf("err = %v", err)
	}
	current, err := os.Readlink(filepath.Join(dataRoot, "config", "current"))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(current) != personalReleaseDigest {
		t.Fatalf("snapshot = %s", filepath.Base(current))
	}
}

func TestSyncReceiptFailureRestoresCurrent(t *testing.T) {
	dataRoot := t.TempDir()
	writeFornaxSources(t, dataRoot)
	client := &syncClient{personalVersion: "1.0.0", loopVersion: "1.0.2"}
	if _, err := Sync(context.Background(), dataRoot, client); err != nil {
		t.Fatal(err)
	}
	previous := commitReceipt
	commitReceipt = func(string, syncReceipt) error { return errors.New("receipt failed") }
	t.Cleanup(func() { commitReceipt = previous })
	client.personalVersion = "1.0.1"
	if _, err := Sync(context.Background(), dataRoot, client); err == nil || !strings.Contains(err.Error(), "receipt failed") {
		t.Fatalf("err = %v", err)
	}
	current, err := os.Readlink(filepath.Join(dataRoot, "config", "current"))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(current) != personalReleaseDigest {
		t.Fatalf("snapshot = %s", filepath.Base(current))
	}
}

func TestSyncSkipsInvalidReadOnlyPackageAndRejectsInvalidReadWrite(t *testing.T) {
	dataRoot := t.TempDir()
	writeFornaxSources(t, dataRoot)
	var warnings []string
	previous := warnSkippedPackage
	warnSkippedPackage = func(key, version string, err error) {
		warnings = append(warnings, fmt.Sprintf("warning: skipping read-only package %s@%s: %s", key, version, err))
	}
	t.Cleanup(func() { warnSkippedPackage = previous })
	client := &syncClient{personalVersion: "1.0.0", loopVersion: "1.0.2", badReadOnly: true}
	result, err := Sync(context.Background(), dataRoot, client)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != OutcomeUpdated || result.Digest != personalReleaseDigest || len(result.Workflows) != 1 || result.Workflows[0].ID != "linear-runtime" || result.Workflows[0].Version != "1.0.0" {
		t.Fatalf("result = %#v", result)
	}
	if len(warnings) != 1 || warnings[0] != `warning: skipping read-only package loop.workflow.linear_runtime@1.0.2: package name "other" does not match workflow id "linear-runtime"` {
		t.Fatalf("warnings = %#v", warnings)
	}
	client.badReadOnly = false
	client.badReadWrite = true
	client.personalVersion = "1.0.1"
	if _, err = Sync(context.Background(), dataRoot, client); err == nil || err.Error() != `validate fanloop.workflow.linear_runtime@1.0.1: package name "other" does not match workflow id "linear-runtime"` {
		t.Fatalf("err = %v", err)
	}
	current, err := os.Readlink(filepath.Join(dataRoot, "config", "current"))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(current) != personalReleaseDigest {
		t.Fatalf("snapshot = %s", filepath.Base(current))
	}
}

func TestSyncActivateFailureRestoresCurrent(t *testing.T) {
	dataRoot := t.TempDir()
	writeFornaxSources(t, dataRoot)
	client := &syncClient{personalVersion: "1.0.0", loopVersion: "1.0.2"}
	if _, err := Sync(context.Background(), dataRoot, client); err != nil {
		t.Fatal(err)
	}
	previous := activateSnapshot
	activateSnapshot = func(string, string) error { return errors.New("activate failed") }
	t.Cleanup(func() { activateSnapshot = previous })
	client.personalVersion = "1.0.1"
	if _, err := Sync(context.Background(), dataRoot, client); err == nil || err.Error() != "activate failed" {
		t.Fatalf("err = %v", err)
	}
	current, err := os.Readlink(filepath.Join(dataRoot, "config", "current"))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(current) != personalReleaseDigest {
		t.Fatalf("snapshot = %s", filepath.Base(current))
	}
}

func TestSyncRejectsChangedBytesForTheSameRelease(t *testing.T) {
	dataRoot := t.TempDir()
	writeFornaxSources(t, dataRoot)
	client := &syncClient{personalVersion: "1.0.0", loopVersion: "1.0.2"}
	if _, err := Sync(context.Background(), dataRoot, client); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dataRoot, "cache", "fornax-workflows.json")); err != nil {
		t.Fatal(err)
	}
	client.mutate = true
	if _, err := Sync(context.Background(), dataRoot, client); err == nil || !strings.Contains(err.Error(), "immutable configuration snapshot differs") {
		t.Fatalf("err = %v", err)
	}
	current, err := os.Readlink(filepath.Join(dataRoot, "config", "current"))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(current) != personalReleaseDigest {
		t.Fatalf("snapshot = %s", filepath.Base(current))
	}
	content, err := os.ReadFile(filepath.Join(current, "fanloop-workflow", "linear-runtime", "workflow.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "# mutated") {
		t.Fatal("mutated bytes replaced the snapshot")
	}
}

func writeFornaxSources(t *testing.T, dataRoot string) {
	t.Helper()
	content := []byte(`{"sources":[{"workspace_id":"1","key_prefix":"fanloop.workflow.","access":"read_write"},{"workspace_id":"2","key_prefix":"loop.workflow.","access":"read_only"}]}` + "\n")
	if err := os.WriteFile(filepath.Join(dataRoot, "fornax.json"), content, 0600); err != nil {
		t.Fatal(err)
	}
}

func copyLinearPackage(packageRoot, wrapperName string) error {
	source := filepath.Join("..", "workflow", "testdata", "linear-runtime")
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, failure error) error {
		if failure != nil {
			return failure
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "SKILL.md" {
			return nil
		}
		target := filepath.Join(packageRoot, "linear-runtime", relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, 0644)
	})
	if err != nil {
		return err
	}
	script := filepath.Join(packageRoot, "linear-runtime", "skills", "test-guide", "scripts", "run.sh")
	if err = os.MkdirAll(filepath.Dir(script), 0755); err != nil {
		return err
	}
	if err = os.WriteFile(script, []byte("#!/bin/sh\necho ok\n"), 0644); err != nil {
		return err
	}
	wrapper := "---\nname: " + wrapperName + "\ndescription: Managed workflow bundle for linear-runtime.\n---\n\n# Workflow bundle\n"
	return os.WriteFile(filepath.Join(packageRoot, "SKILL.md"), []byte(wrapper), 0644)
}
