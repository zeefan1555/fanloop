package cmd

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/configuration"
	"github.com/zeefan1555/fanloop/internal/fornaxmirror"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

func TestConfigDriftPublicCLIReportsInSyncAndDrift(t *testing.T) {
	dataRoot, source, commit := configDriftSnapshot(t)
	packageRoot := filepath.Join(t.TempDir(), "package")
	archive := filepath.Join(t.TempDir(), "linear-runtime.zip")
	key, _ := fornaxmirror.SkillKey("linear-runtime")
	if _, err := fornaxmirror.BuildArchive(source, "linear-runtime", key, commit, "1.0.0", archive); err != nil {
		t.Fatal(err)
	}
	configExtract(t, archive, packageRoot)
	fakeDir := t.TempDir()
	configWriteFakeFornax(t, filepath.Join(fakeDir, "fornax-cli"))
	configUnsetSource(t)
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FAKE_FORNAX_PACKAGE", packageRoot)
	t.Setenv("PATH", fakeDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	unlock, err := workflow.LockConfiguration(dataRoot, false)
	if err != nil {
		t.Fatal(err)
	}
	unlock()

	var stdout, stderr bytes.Buffer
	if code := Execute(context.Background(), []string{"config", "drift"}, strings.NewReader(""), &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), `"status": "in_sync"`) || stderr.Len() != 0 {
		t.Fatalf("in-sync exit=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	workflowYAML := filepath.Join(packageRoot, "linear-runtime", "workflow.yaml")
	if err := os.WriteFile(workflowYAML, append(configRead(t, workflowYAML), '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute(context.Background(), []string{"config", "drift"}, strings.NewReader(""), &stdout, &stderr); code != 1 || !strings.Contains(stdout.String(), `"status": "drifted"`) || !strings.Contains(stdout.String(), `"detail": "package content differs"`) {
		t.Fatalf("drift exit=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
}

func TestConfigDriftPublicCLICoversMirrorStatesAndInputErrors(t *testing.T) {
	dataRoot, source, commit := configDriftSnapshot(t)
	archive := filepath.Join(t.TempDir(), "linear-runtime.zip")
	key, _ := fornaxmirror.SkillKey("linear-runtime")
	if _, err := fornaxmirror.BuildArchive(source, "linear-runtime", key, commit, "1.0.0", archive); err != nil {
		t.Fatal(err)
	}
	packageRoot := filepath.Join(t.TempDir(), "package")
	configExtract(t, archive, packageRoot)
	fakeDir := t.TempDir()
	configWriteFakeFornax(t, filepath.Join(fakeDir, "fornax-cli"))
	configUnsetSource(t)
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("FAKE_FORNAX_PACKAGE", packageRoot)
	t.Setenv("PATH", fakeDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	unlock, err := workflow.LockConfiguration(dataRoot, false)
	if err != nil {
		t.Fatal(err)
	}
	unlock()

	run := func(t *testing.T, mode string, wantCode int, fragments ...string) {
		t.Helper()
		t.Setenv("FAKE_FORNAX_MODE", mode)
		before := configDataSnapshot(t, dataRoot)
		var stdout, stderr bytes.Buffer
		code := Execute(context.Background(), []string{"config", "drift"}, strings.NewReader(""), &stdout, &stderr)
		if code != wantCode {
			t.Fatalf("exit=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
		}
		combined := stdout.String() + stderr.String()
		for _, fragment := range fragments {
			if !strings.Contains(combined, fragment) {
				t.Fatalf("missing %q in stdout=%s stderr=%s", fragment, stdout.String(), stderr.String())
			}
		}
		if after := configDataSnapshot(t, dataRoot); !reflect.DeepEqual(before, after) {
			t.Fatalf("config drift mutated FANLOOP_DATA_HOME\nbefore=%v\nafter=%v", before, after)
		}
	}

	run(t, "good", 0, `"status": "in_sync"`)
	for _, test := range []struct {
		name      string
		mode      string
		fragments []string
	}{
		{name: "missing key", mode: "missing", fragments: []string{`"status": "missing"`, `"detail": "Skill key is missing"`}},
		{name: "unreleased", mode: "unreleased", fragments: []string{`"status": "unreleased"`, `"detail": "Online/prod has no released version"`}},
		{name: "fornax failure", mode: "failure", fragments: []string{`"status": "unavailable"`, `"detail": "Fornax Skill discovery failed"`}},
	} {
		t.Run(test.name, func(t *testing.T) { run(t, test.mode, 1, test.fragments...) })
	}

	t.Run("mode-only difference", func(t *testing.T) {
		t.Setenv("FAKE_FORNAX_MODE", "executable-drift")
		before := configDataSnapshot(t, dataRoot)
		var stdout, stderr bytes.Buffer
		code := Execute(context.Background(), []string{"config", "drift"}, strings.NewReader(""), &stdout, &stderr)
		combined := stdout.String() + stderr.String()
		if code != 0 || strings.Contains(combined, `"drifted"`) || strings.Contains(combined, "package content differs") || !strings.Contains(combined, `"status": "in_sync"`) || !strings.Contains(combined, "file mode differs for linear-runtime/skills/test-guide/SKILL.md") {
			t.Fatalf("exit=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
		}
		if after := configDataSnapshot(t, dataRoot); !reflect.DeepEqual(before, after) {
			t.Fatalf("config drift mutated FANLOOP_DATA_HOME\nbefore=%v\nafter=%v", before, after)
		}
	})

	t.Run("fornax command unavailable", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		run(t, "good", 1, `"status": "unavailable"`, `"detail": "Fornax Skill discovery failed"`)
	})

	configPath := filepath.Join(dataRoot, "fornax.json")
	original := configRead(t, configPath)
	for _, test := range []struct {
		name    string
		prepare func()
	}{
		{name: "invalid JSON", prepare: func() { _ = os.WriteFile(configPath, []byte(`{"workspace_id":`), 0600) }},
		{name: "missing file", prepare: func() { _ = os.Remove(configPath) }},
		{name: "inventory mismatch", prepare: func() {
			_ = os.WriteFile(configPath, []byte(`{"workspace_id":"1","workflows":{"daily-journal":"fanloop.workflow.daily_journal"}}`), 0600)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := os.WriteFile(configPath, original, 0600); err != nil {
				t.Fatal(err)
			}
			test.prepare()
			run(t, "good", 2, `"code": "INVALID_ARGUMENT"`)
		})
	}
	if err := os.WriteFile(configPath, original, 0600); err != nil {
		t.Fatal(err)
	}
}

func configDriftSnapshot(t *testing.T) (string, string, string) {
	t.Helper()
	source := t.TempDir()
	configCopyTree(t, filepath.Join("..", "internal", "workflow", "testdata", "linear-runtime"), filepath.Join(source, "fanloop-workflow", "linear-runtime"))
	configGit(t, source, "init", "--quiet")
	configGit(t, source, "config", "user.name", "Config Drift Test")
	configGit(t, source, "config", "user.email", "config-drift@example.test")
	configGit(t, source, "add", ".")
	configGit(t, source, "commit", "--quiet", "-m", "fixture")
	commit := configGit(t, source, "rev-parse", "HEAD")
	dataRoot := t.TempDir()
	snapshot, err := configuration.Import(context.Background(), dataRoot, source)
	if err != nil {
		t.Fatal(err)
	}
	if err = configuration.Activate(dataRoot, snapshot); err != nil {
		t.Fatal(err)
	}
	content := `{"workspace_id":"1","workflows":{"linear-runtime":"fanloop.workflow.linear_runtime"}}`
	if err = os.WriteFile(filepath.Join(dataRoot, "fornax.json"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return dataRoot, source, commit
}

func configWriteFakeFornax(t *testing.T, filename string) {
	t.Helper()
	content := `#!/bin/sh
set -eu
command="$1 $2"
mode="${FAKE_FORNAX_MODE:-good}"
if [ "$mode" = "failure" ]; then exit 7; fi
if [ "$command" = "skill install" ]; then
  shift 2
  destination=""
  while [ "$#" -gt 0 ]; do
    if [ "$1" = "--dir" ]; then destination="$2"; shift 2; else shift; fi
  done
  mkdir -p "$destination/fake-package"
  cp -R "$FAKE_FORNAX_PACKAGE/." "$destination/fake-package/"
  if [ "$mode" = "executable-drift" ]; then chmod +x "$destination/fake-package/linear-runtime/skills/test-guide/SKILL.md"; fi
  exit 0
fi
output=""
previous=""
for argument in "$@"; do
  if [ "$previous" = "--output" ]; then output="$argument"; fi
  previous="$argument"
done
mkdir -p "$output"
if [ "$command" = "skill list-releases" ]; then
  if [ "$mode" = "unreleased" ]; then
    printf '%s\n' '{"items":[]}' > "$output/releases.json"
  else
    printf '%s\n' '{"items":[{"version":"1.0.0","updated_at":"1","release_status":1}]}' > "$output/releases.json"
  fi
else
  if [ "$mode" = "missing" ]; then
    printf '%s\n' '{"skills":[],"total":0}' > "$output/skills.json"
  else
    printf '%s\n' '{"skills":[{"id":"123","skill_key":"fanloop.workflow.linear_runtime"}],"total":1}' > "$output/skills.json"
  fi
fi
`
	if err := os.WriteFile(filename, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
}

func configDataSnapshot(t *testing.T, root string) []string {
	t.Helper()
	result := []string{}
	err := filepath.WalkDir(root, func(filename string, item fs.DirEntry, failure error) error {
		if failure != nil {
			return failure
		}
		relative, err := filepath.Rel(root, filename)
		if err != nil {
			return err
		}
		info, err := os.Lstat(filename)
		if err != nil {
			return err
		}
		value := fmt.Sprintf("%s %s", filepath.ToSlash(relative), info.Mode().String())
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(filename)
			if err != nil {
				return err
			}
			value += " -> " + target
		} else if info.Mode().IsRegular() {
			content, err := os.ReadFile(filename)
			if err != nil {
				return err
			}
			digest := sha256.Sum256(content)
			value += " " + hex.EncodeToString(digest[:])
		}
		result = append(result, value)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(result)
	return result
}

func configExtract(t *testing.T, archive, destination string) {
	t.Helper()
	reader, err := zip.OpenReader(archive)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		filename := filepath.Join(destination, filepath.FromSlash(file.Name))
		if err = os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			t.Fatal(err)
		}
		stream, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, err := io.ReadAll(stream)
		_ = stream.Close()
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filename, content, file.Mode().Perm()); err != nil {
			t.Fatal(err)
		}
	}
}

func configCopyTree(t *testing.T, source, destination string) {
	t.Helper()
	err := filepath.WalkDir(source, func(filename string, item fs.DirEntry, failure error) error {
		if failure != nil {
			return failure
		}
		relative, err := filepath.Rel(source, filename)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if item.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		info, err := item.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(target, configRead(t, filename), info.Mode().Perm())
	})
	if err != nil {
		t.Fatal(err)
	}
}

func configGit(t *testing.T, directory string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(output))
}

func configRead(t *testing.T, filename string) []byte {
	t.Helper()
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func configUnsetSource(t *testing.T) {
	t.Helper()
	value, exists := os.LookupEnv("FANLOOP_CONFIG_SOURCE")
	_ = os.Unsetenv("FANLOOP_CONFIG_SOURCE")
	t.Cleanup(func() {
		if exists {
			_ = os.Setenv("FANLOOP_CONFIG_SOURCE", value)
		} else {
			_ = os.Unsetenv("FANLOOP_CONFIG_SOURCE")
		}
	})
}
