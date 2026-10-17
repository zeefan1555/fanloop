package fornaxmirror

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/zeefan1555/fanloop/internal/configuration"
	"golang.org/x/sys/unix"
)

type fakeClient struct {
	source   string
	commit   string
	missing  bool
	failure  bool
	drift    bool
	modeOnly bool
}

func (client fakeClient) Skills(_ context.Context, _, key string) ([]Skill, error) {
	if client.failure {
		return nil, errors.New("offline")
	}
	if client.missing {
		return nil, nil
	}
	return []Skill{{ID: "123", Key: key}}, nil
}

func (client fakeClient) Releases(context.Context, string, string) ([]Release, error) {
	return []Release{{Version: "1.0.0", UpdatedAt: 1}}, nil
}

func (client fakeClient) ListByPrefix(ctx context.Context, workspaceID, prefix string) ([]Skill, error) {
	return client.Skills(ctx, workspaceID, prefix)
}

func (client fakeClient) Install(_ context.Context, _, key, version, directory string) (string, error) {
	archive := filepath.Join(directory, "candidate.zip")
	if _, err := BuildArchive(client.source, "linear-runtime", key, client.commit, version, archive); err != nil {
		return "", err
	}
	root := filepath.Join(directory, "fanloop-workflow-linear-runtime")
	extractPackage(archive, root)
	if client.drift {
		path := filepath.Join(root, "linear-runtime", "workflow.yaml")
		content, _ := os.ReadFile(path)
		_ = os.WriteFile(path, append(content, '\n'), 0644)
	}
	if client.modeOnly {
		path := filepath.Join(root, "linear-runtime", "skills", "test-guide", "SKILL.md")
		_ = os.Chmod(path, 0755)
	}
	return root, nil
}

func TestCheckUsesReleasedPackageAsReference(t *testing.T) {
	unsetConfigSource(t)
	dataRoot := t.TempDir()
	writeFornaxSources(t, dataRoot)
	client := &syncClient{personalVersion: "1.0.0", loopVersion: "1.0.2"}
	if _, err := Sync(context.Background(), dataRoot, client); err != nil {
		t.Fatal(err)
	}
	report, err := Check(context.Background(), dataRoot, client)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != DriftInSync || report.ConfigurationCommit != "" || len(report.Workflows) != 1 || report.Workflows[0].Status != WorkflowInSync || report.Workflows[0].WorkflowID != "linear-runtime" || report.Workflows[0].ReleasedVersion != "1.0.0" || report.Workflows[0].SkillKey != "fanloop.workflow.linear_runtime" {
		t.Fatalf("in-sync report = %#v", report)
	}
	client.personalVersion = "1.0.1"
	client.mutate = true
	report, err = Check(context.Background(), dataRoot, client)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != DriftLocalBehind || report.Workflows[0].Status != WorkflowLocalBehind || report.Workflows[0].Detail != "released version is 1.0.1" {
		t.Fatalf("behind report = %#v", report)
	}
	client.personalVersion = "1.0.0"
	report, err = Check(context.Background(), dataRoot, client)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != DriftLocalModified || report.Workflows[0].Status != WorkflowLocalModified || report.Workflows[0].Detail != "package content differs" {
		t.Fatalf("modified report = %#v", report)
	}
	client.mutate = false
	client.failInstall = true
	report, err = Check(context.Background(), dataRoot, client)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != DriftUnavailable || report.Workflows[0].Status != WorkflowUnavailable || report.Workflows[0].Detail != "Fornax package download failed" {
		t.Fatalf("unavailable report = %#v", report)
	}
}

type lockProbeClient struct {
	*syncClient
	lockPath string
	checked  *bool
}

func (client lockProbeClient) ListByPrefix(ctx context.Context, workspaceID, prefix string) ([]Skill, error) {
	lock, err := os.OpenFile(client.lockPath, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	defer lock.Close()
	if err = unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return nil, errors.New("configuration lock remained held during Fornax query")
	}
	_ = unix.Flock(int(lock.Fd()), unix.LOCK_UN)
	*client.checked = true
	return client.syncClient.ListByPrefix(ctx, workspaceID, prefix)
}

func TestCheckSkipsInvalidReadOnlyPackage(t *testing.T) {
	unsetConfigSource(t)
	dataRoot := t.TempDir()
	writeFornaxSources(t, dataRoot)
	var warnings []string
	previous := warnSkippedPackage
	warnSkippedPackage = func(key, version string, err error) {
		warnings = append(warnings, fmt.Sprintf("warning: skipping read-only package %s@%s: %s", key, version, err))
	}
	t.Cleanup(func() { warnSkippedPackage = previous })
	client := &syncClient{personalVersion: "1.0.0", loopVersion: "1.0.2", badReadOnly: true}
	if _, err := Sync(context.Background(), dataRoot, client); err != nil {
		t.Fatal(err)
	}
	warnings = nil
	report, err := Check(context.Background(), dataRoot, client)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != DriftInSync || len(report.Workflows) != 1 || report.Workflows[0].Status != WorkflowInSync || report.Workflows[0].WorkflowID != "linear-runtime" || report.Workflows[0].SkillKey != "fanloop.workflow.linear_runtime" {
		t.Fatalf("in-sync report = %#v", report)
	}
	if len(warnings) != 1 || warnings[0] != `warning: skipping read-only package loop.workflow.linear_runtime@1.0.2: package name "other" does not match workflow id "linear-runtime"` {
		t.Fatalf("warnings = %#v", warnings)
	}
	client.badReadOnly = false
	client.badReadWrite = true
	report, err = Check(context.Background(), dataRoot, client)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != DriftUnavailable || len(report.Workflows) != 1 || report.Workflows[0].Status != WorkflowUnavailable || report.Workflows[0].Detail != "Fornax package is invalid" {
		t.Fatalf("unavailable report = %#v", report)
	}
}

type readOnlyPairClient struct {
	includeBroken bool
	secondResidue []string
}

func (client *readOnlyPairClient) Skills(context.Context, string, string) ([]Skill, error) {
	return nil, errors.New("exact key lookup is not used")
}

func (client *readOnlyPairClient) ListByPrefix(context.Context, string, string) ([]Skill, error) {
	skills := []Skill{{ID: "good", Key: "loop.workflow.linear_runtime"}}
	if client.includeBroken {
		skills = append([]Skill{{ID: "broken", Key: "loop.workflow.broken_pack"}}, skills...)
	}
	return skills, nil
}

func (client *readOnlyPairClient) Releases(context.Context, string, string) ([]Release, error) {
	return []Release{{Version: "1.0.0", UpdatedAt: 1}}, nil
}

func (client *readOnlyPairClient) Install(_ context.Context, _, key, _, directory string) (string, error) {
	root := filepath.Join(directory, "loop-workflow-linear-runtime")
	wrapperName := "loop-workflow-linear-runtime"
	if key == "loop.workflow.broken_pack" {
		wrapperName = "other"
	}
	if err := copyLinearPackage(root, wrapperName); err != nil {
		return "", err
	}
	marker := filepath.Join(root, "linear-runtime", "skills", "test-guide", "from-first.txt")
	if key == "loop.workflow.broken_pack" {
		if err := os.WriteFile(marker, []byte("from-first\n"), 0644); err != nil {
			return "", err
		}
		return root, nil
	}
	if _, err := os.Stat(marker); err == nil {
		client.secondResidue = append(client.secondResidue, "linear-runtime/skills/test-guide/from-first.txt")
	}
	return root, nil
}

func TestCheckSkippedReadOnlyPackageDoesNotReuseTempFolders(t *testing.T) {
	unsetConfigSource(t)
	dataRoot := t.TempDir()
	content := []byte(`{"sources":[{"workspace_id":"2","key_prefix":"loop.workflow.","access":"read_only"}]}` + "\n")
	if err := os.WriteFile(filepath.Join(dataRoot, "fornax.json"), content, 0600); err != nil {
		t.Fatal(err)
	}
	client := &readOnlyPairClient{}
	if _, err := Sync(context.Background(), dataRoot, client); err != nil {
		t.Fatal(err)
	}
	client.includeBroken = true
	var warnings []string
	previous := warnSkippedPackage
	warnSkippedPackage = func(key, version string, err error) {
		warnings = append(warnings, fmt.Sprintf("warning: skipping read-only package %s@%s: %s", key, version, err))
	}
	t.Cleanup(func() { warnSkippedPackage = previous })
	report, err := Check(context.Background(), dataRoot, client)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != DriftInSync || len(report.Workflows) != 1 || report.Workflows[0].Status != WorkflowInSync || report.Workflows[0].WorkflowID != "linear-runtime" || report.Workflows[0].SkillKey != "loop.workflow.linear_runtime" {
		t.Fatalf("in-sync report = %#v", report)
	}
	if len(client.secondResidue) != 0 {
		t.Fatalf("second package contains files from the first: %#v", client.secondResidue)
	}
	if len(warnings) != 1 || warnings[0] != `warning: skipping read-only package loop.workflow.broken_pack@1.0.0: package name "other" does not match workflow id "linear-runtime"` {
		t.Fatalf("warnings = %#v", warnings)
	}
}

func TestCheckReleasesConfigurationLockBeforeFornaxQuery(t *testing.T) {
	unsetConfigSource(t)
	dataRoot := t.TempDir()
	writeFornaxSources(t, dataRoot)
	inner := &syncClient{personalVersion: "1.0.0", loopVersion: "1.0.2"}
	if _, err := Sync(context.Background(), dataRoot, inner); err != nil {
		t.Fatal(err)
	}
	checked := false
	client := lockProbeClient{syncClient: inner, lockPath: filepath.Join(dataRoot, ".workflows.lock"), checked: &checked}
	report, err := Check(context.Background(), dataRoot, client)
	if err != nil {
		t.Fatal(err)
	}
	if !checked || report.Status != DriftInSync {
		t.Fatalf("lock probe checked=%t report=%#v", checked, report)
	}
}

func TestCheckRejectsInvalidLocalConfiguration(t *testing.T) {
	dataRoot, _, _ := activeSnapshot(t)
	if err := os.WriteFile(filepath.Join(dataRoot, "fornax.json"), []byte(`{"workspace_id":"bad"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(context.Background(), dataRoot, fakeClient{}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("error = %v", err)
	}
}

func activeSnapshot(t *testing.T) (string, string, string) {
	t.Helper()
	unsetConfigSource(t)
	source := fixtureSource(t)
	git(t, source, "init", "--quiet")
	git(t, source, "config", "user.name", "Mirror Test")
	git(t, source, "config", "user.email", "mirror@example.test")
	git(t, source, "add", ".")
	git(t, source, "commit", "--quiet", "-m", "fixture")
	commit := git(t, source, "rev-parse", "HEAD")
	dataRoot := t.TempDir()
	snapshot, err := configuration.Import(context.Background(), dataRoot, source)
	if err != nil {
		t.Fatal(err)
	}
	if err = configuration.Activate(dataRoot, snapshot); err != nil {
		t.Fatal(err)
	}
	config := `{"sources":[{"workspace_id":"1","key_prefix":"fanloop.workflow.","access":"read_write"}]}`
	if err = os.WriteFile(filepath.Join(dataRoot, "fornax.json"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	return dataRoot, source, commit
}

func git(t *testing.T, directory string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(bytesTrim(output))
}

func bytesTrim(value []byte) []byte {
	for len(value) > 0 && (value[len(value)-1] == '\n' || value[len(value)-1] == '\r') {
		value = value[:len(value)-1]
	}
	return value
}

func unsetConfigSource(t *testing.T) {
	t.Helper()
	value, exists := os.LookupEnv("FANLOOP_CONFIG_SOURCE")
	if err := os.Unsetenv("FANLOOP_CONFIG_SOURCE"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if exists {
			_ = os.Setenv("FANLOOP_CONFIG_SOURCE", value)
		} else {
			_ = os.Unsetenv("FANLOOP_CONFIG_SOURCE")
		}
	})
}

func extractPackage(archive, root string) {
	reader, _ := zip.OpenReader(archive)
	defer reader.Close()
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		path := filepath.Join(root, filepath.FromSlash(file.Name))
		_ = os.MkdirAll(filepath.Dir(path), 0755)
		stream, _ := file.Open()
		content, _ := io.ReadAll(stream)
		_ = stream.Close()
		_ = os.WriteFile(path, content, file.Mode().Perm())
	}
}
