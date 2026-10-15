package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/fornaxmirror"
)

func TestBuildPackageSetLeavesNoPartialOutputWhenLaterPackageFails(t *testing.T) {
	output := filepath.Join(t.TempDir(), "packages")
	if err := os.Mkdir(output, 0755); err != nil {
		t.Fatal(err)
	}
	_, err := buildPackageSet(output, []string{"first", "second"}, func(workflowID string) string {
		return workflowID + ".zip"
	}, func(workflowID, target string) (fornaxmirror.BuiltArchive, error) {
		if workflowID == "second" {
			return fornaxmirror.BuiltArchive{}, errors.New("later package failed")
		}
		if err := os.WriteFile(target, []byte("partial"), 0644); err != nil {
			return fornaxmirror.BuiltArchive{}, err
		}
		return fornaxmirror.BuiltArchive{WorkflowID: workflowID, Path: target}, nil
	})
	if err == nil {
		t.Fatal("later package failure passed")
	}
	entries, readErr := os.ReadDir(output)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("partial package output remains: %v", entries)
	}
}

func TestCompareCommandPassesModeOnlyDifference(t *testing.T) {
	source := t.TempDir()
	packageRoot := filepath.Join(source, "fanloop-workflow", "linear-runtime")
	copyFixture(t, filepath.Join("..", "..", "internal", "workflow", "testdata", "linear-runtime"), packageRoot)
	script := filepath.Join(packageRoot, "skills", "test-guide", "scripts", "run.sh")
	if err := os.MkdirAll(filepath.Dir(script), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho ok\n"), 0755); err != nil {
		t.Fatal(err)
	}
	key, err := fornaxmirror.SkillKey("linear-runtime")
	if err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(t.TempDir(), "linear-runtime.zip")
	if _, err = fornaxmirror.BuildArchive(source, "linear-runtime", key, "0123456789abcdef0123456789abcdef01234567", "1.0.0", archive); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(t.TempDir(), "installed")
	extractZip(t, archive, directory)
	installed := filepath.Join(directory, "linear-runtime", "skills", "test-guide", "scripts", "run.sh")
	if err = os.Chmod(installed, 0644); err != nil {
		t.Fatal(err)
	}

	output := captureStdout(t, func() error {
		return compare([]string{"--archive", archive, "--directory", directory})
	})
	var payload struct {
		Outcome string   `json:"outcome"`
		Notes   []string `json:"notes"`
	}
	if err = json.Unmarshal(output, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Outcome != "passed" || len(payload.Notes) != 1 || !strings.Contains(payload.Notes[0], "scripts/run.sh") {
		t.Fatalf("compare payload = %s", output)
	}

	if err = os.WriteFile(installed, []byte("#!/bin/sh\necho changed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err = compare([]string{"--archive", archive, "--directory", directory}); err == nil || !strings.Contains(err.Error(), "package content differs") {
		t.Fatalf("content difference error = %v", err)
	}
	if err = os.Remove(installed); err != nil {
		t.Fatal(err)
	}
	if err = compare([]string{"--archive", archive, "--directory", directory}); err == nil {
		t.Fatal("missing file compared equal")
	}
	if err = os.WriteFile(installed, []byte("#!/bin/sh\necho ok\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(installed+"-extra", []byte("extra\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err = compare([]string{"--archive", archive, "--directory", directory}); err == nil {
		t.Fatal("extra file compared equal")
	}
}

func captureStdout(t *testing.T, fn func() error) []byte {
	t.Helper()
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	runErr := fn()
	_ = writer.Close()
	os.Stdout = original
	output, readErr := io.ReadAll(reader)
	_ = reader.Close()
	if runErr != nil {
		t.Fatal(runErr)
	}
	if readErr != nil {
		t.Fatal(readErr)
	}
	return output
}

func copyFixture(t *testing.T, source, destination string) {
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
		content, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		info, err := item.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, info.Mode().Perm())
	})
	if err != nil {
		t.Fatal(err)
	}
}

func extractZip(t *testing.T, archive, destination string) {
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
