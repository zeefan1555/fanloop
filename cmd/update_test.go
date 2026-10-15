package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/configuration"
	"github.com/zeefan1555/fanloop/internal/distribution"
)

func TestUpdateAndUpdateConfigStaySeparate(t *testing.T) {
	t.Setenv("FANLOOP_DATA_HOME", t.TempDir())
	previousRelease := applyReleaseUpdate
	previousConfig := applyConfigSync
	t.Cleanup(func() {
		applyReleaseUpdate = previousRelease
		applyConfigSync = previousConfig
	})

	releaseCalls := 0
	configCalls := 0
	applyReleaseUpdate = func(context.Context, string) (distribution.Result, error) {
		releaseCalls++
		return distribution.Result{Version: "1.2.3"}, nil
	}
	applyConfigSync = func(context.Context, string) (configuration.SyncResult, error) {
		configCalls++
		return configuration.SyncResult{}, nil
	}

	var stdout, stderr bytes.Buffer
	if code := Execute(context.Background(), []string{"update"}, strings.NewReader(""), &stdout, &stderr); code != 0 || stdout.String() != "updated 1.2.3\n" || stderr.Len() != 0 {
		t.Fatalf("update exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if releaseCalls != 1 || configCalls != 0 {
		t.Fatalf("update called release=%d config=%d", releaseCalls, configCalls)
	}

	stdout.Reset()
	applyConfigSync = func(context.Context, string) (configuration.SyncResult, error) {
		configCalls++
		return configuration.SyncResult{
			CommitSHA:   "0123456789abcdef0123456789abcdef01234567",
			WorkflowIDs: []string{"playbook-flow", "remnote-flashcards"},
			Unchanged:   true,
		}, nil
	}
	if code := Execute(context.Background(), []string{"update", "config"}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("update config exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	want := "unchanged playbook-flow 0123456789abcdef0123456789abcdef01234567\nunchanged remnote-flashcards 0123456789abcdef0123456789abcdef01234567\n"
	if stdout.String() != want || releaseCalls != 1 || configCalls != 1 {
		t.Fatalf("update config stdout=%q release=%d config=%d", stdout.String(), releaseCalls, configCalls)
	}
}
