package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/distribution"
	"github.com/zeefan1555/fanloop/internal/fornaxmirror"
)

func TestUpdateAndUpdateConfigPrintReleasedWorkflows(t *testing.T) {
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
	applyConfigSync = func(context.Context, string) (fornaxmirror.SyncResult, error) {
		configCalls++
		return fornaxmirror.SyncResult{
			Outcome: fornaxmirror.OutcomeUpdated,
			Workflows: []fornaxmirror.SyncedWorkflow{
				{ID: "pstack-native", Version: "1.0.2"},
				{ID: "linear-runtime", Version: "1.0.0"},
			},
		}, nil
	}

	var stdout, stderr bytes.Buffer
	if code := Execute(context.Background(), []string{"update"}, strings.NewReader(""), &stdout, &stderr); code != 0 || stderr.Len() != 0 {
		t.Fatalf("update exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	want := "updated 1.2.3\nfanloop configuration updated: linear-runtime@1.0.0, pstack-native@1.0.2\n"
	if stdout.String() != want || releaseCalls != 1 || configCalls != 1 {
		t.Fatalf("update stdout=%q release=%d config=%d", stdout.String(), releaseCalls, configCalls)
	}

	stdout.Reset()
	applyConfigSync = func(context.Context, string) (fornaxmirror.SyncResult, error) {
		configCalls++
		return fornaxmirror.SyncResult{
			Outcome: fornaxmirror.OutcomeUnchanged,
			Workflows: []fornaxmirror.SyncedWorkflow{
				{ID: "pstack-native", Version: "1.0.2"},
				{ID: "linear-runtime", Version: "1.0.0"},
			},
		}, nil
	}
	if code := Execute(context.Background(), []string{"update", "config"}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("update config exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	want = "fanloop configuration is already up to date: linear-runtime@1.0.0, pstack-native@1.0.2\n"
	if stdout.String() != want || releaseCalls != 1 || configCalls != 2 {
		t.Fatalf("update config stdout=%q release=%d config=%d", stdout.String(), releaseCalls, configCalls)
	}
}
