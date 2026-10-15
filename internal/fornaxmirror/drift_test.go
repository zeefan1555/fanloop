package fornaxmirror

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestCheckReportsInSyncAndDrift(t *testing.T) {
	dataRoot, source, commit := activeSnapshot(t)
	client := fakeClient{source: source, commit: commit}
	report, err := Check(context.Background(), dataRoot, client)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != DriftInSync || len(report.Workflows) != 1 || report.Workflows[0].Status != WorkflowInSync || report.ConfigurationCommit != commit {
		t.Fatalf("in-sync report = %#v", report)
	}
	client.drift = true
	report, err = Check(context.Background(), dataRoot, client)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != Drifted || report.Workflows[0].Status != WorkflowDrifted || report.Workflows[0].Detail != "package content differs" {
		t.Fatalf("drift report = %#v", report)
	}
	client = fakeClient{source: source, commit: commit, modeOnly: true}
	report, err = Check(context.Background(), dataRoot, client)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != DriftInSync || report.Workflows[0].Status != WorkflowInSync || !strings.Contains(report.Workflows[0].Detail, "file mode differs for linear-runtime/skills/test-guide/SKILL.md") {
		t.Fatalf("mode-only report = %#v", report)
	}
	client = fakeClient{source: source, commit: commit, missing: true}
	report, err = Check(context.Background(), dataRoot, client)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != Drifted || report.Workflows[0].Status != WorkflowMissing {
		t.Fatalf("missing report = %#v", report)
	}
	client = fakeClient{source: source, commit: commit, failure: true}
	report, err = Check(context.Background(), dataRoot, client)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != DriftUnavailable || report.Workflows[0].Status != WorkflowUnavailable {
		t.Fatalf("unavailable report = %#v", report)
	}
}

type lockProbeClient struct {
	fakeClient
	lockPath string
	checked  *bool
}

func (client lockProbeClient) Skills(ctx context.Context, workspaceID, key string) ([]Skill, error) {
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
	return client.fakeClient.Skills(ctx, workspaceID, key)
}

func TestCheckReleasesConfigurationLockBeforeFornaxQuery(t *testing.T) {
	dataRoot, source, commit := activeSnapshot(t)
	checked := false
	client := lockProbeClient{fakeClient: fakeClient{source: source, commit: commit}, lockPath: filepath.Join(dataRoot, ".workflows.lock"), checked: &checked}
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
