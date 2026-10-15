package main

import (
	"errors"
	"os"
	"path/filepath"
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
