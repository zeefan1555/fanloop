package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestLeafHelpListsErrorCodes(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute(context.Background(), []string{"flow", "status", "--help"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	help := stdout.String()
	if !strings.Contains(help, "Errors:") || !strings.Contains(help, "NOT_INITIALIZED (type=state, exit=1, retryable=false): Run flow init for this requirement root.") {
		t.Fatalf("help = %s", help)
	}
}
