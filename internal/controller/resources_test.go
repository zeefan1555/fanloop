package controller

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/zeefan1555/fanloop/internal/idl/storageidl"
	"github.com/zeefan1555/fanloop/internal/release"
	"github.com/zeefan1555/fanloop/internal/testfixture"
)

func TestControllerPreparationAcceptsCurrentInstallerLock(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(t.TempDir(), "release")
	if err := testfixture.BuildRelease(source, "test"); err != nil {
		t.Fatal(err)
	}
	configuration := filepath.Join(t.TempDir(), "configuration")
	if err := os.CopyFS(filepath.Join(configuration, "fanloop-workflow", "linear-runtime"), os.DirFS("../workflow/testdata/linear-runtime")); err != nil {
		t.Fatal(err)
	}
	resources, err := Plan(root, source, configuration, "", storageidl.ControllerBindingOperation_initialize, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := resources.Prepare(context.Background(), func(string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(root, Pending, "releases", ".current.lock")
	info, err := os.Lstat(lock)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() != 0 {
		t.Fatalf("expected current installer lock: %v", err)
	}
	if err := os.WriteFile(lock, []byte("unknown content"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := resources.checkHome(filepath.Join(root, Pending)); err == nil {
		t.Fatal("nonempty installer lock was accepted")
	}
}

func TestControllerResourceIdentityIncludesModesAndUnknownPaths(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(t.TempDir(), "release")
	if err := testfixture.BuildRelease(source, "test"); err != nil {
		t.Fatal(err)
	}
	configuration := filepath.Join(t.TempDir(), "configuration")
	if err := os.CopyFS(filepath.Join(configuration, "fanloop-workflow", "linear-runtime"), os.DirFS("../workflow/testdata/linear-runtime")); err != nil {
		t.Fatal(err)
	}
	resources, err := Plan(root, source, configuration, "", storageidl.ControllerBindingOperation_initialize, true)
	if err != nil {
		t.Fatal(err)
	}
	guide := filepath.Join(source, "entrypoints", "fanloop-workflow", "SKILL.md")
	info, err := os.Stat(guide)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(guide, info.Mode().Perm()^0200); err != nil {
		t.Fatal(err)
	}
	if err := resources.CheckSource(); err == nil {
		t.Fatal("source mode drift was accepted")
	}
	if err := os.Chmod(guide, info.Mode().Perm()); err != nil {
		t.Fatal(err)
	}
	staging := filepath.Join(root, Pending)
	if err := os.Mkdir(staging, 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(staging, resources.Binding.ControllerReleasePath)
	if err := copyRelease(source, target, func(string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := resources.CheckTarget(staging); err != nil {
		t.Fatal(err)
	}
	copied := filepath.Join(target, "entrypoints", "fanloop-workflow", "SKILL.md")
	if err := os.Chmod(copied, info.Mode().Perm()^0200); err != nil {
		t.Fatal(err)
	}
	if err := resources.CheckTarget(staging); err == nil {
		t.Fatal("copied mode drift was accepted")
	}
	if err := os.Chmod(copied, info.Mode().Perm()); err != nil {
		t.Fatal(err)
	}
	if err := resources.checkHome(staging); err != nil {
		t.Fatal(err)
	}
	unknown := filepath.Join(staging, "unknown")
	if err := os.WriteFile(unknown, []byte("not owned"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := resources.checkHome(staging); err == nil {
		t.Fatal("unknown staging content was accepted")
	}
	content, err := os.ReadFile(unknown)
	if err != nil || string(content) != "not owned" {
		t.Fatalf("unknown content changed: %q %v", content, err)
	}
}

func TestBindingDirectoryCreationRejectsLinkedAncestorBeforeWriting(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	link := filepath.Join(root, "linked")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if err := release.MakeDirectory(filepath.Join(link, "must-not-exist"), 0700); err == nil {
		t.Fatal("linked ancestor was accepted")
	}
	if _, err := os.Lstat(filepath.Join(outside, "must-not-exist")); !os.IsNotExist(err) {
		t.Fatalf("escaped directory was written: %v", err)
	}
}
