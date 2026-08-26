package card

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSendBotmuxDoesNotExposeProviderOutputOrBinding(t *testing.T) {
	directory := t.TempDir()
	writeBotmuxTestExecutable(t, directory, `#!/bin/sh
printf '%s\n' 'provider leaked oc_sensitive session-sensitive credential-sensitive' >&2
exit 17
`)
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))

	err := sendBotmux(context.Background(), BotmuxBinding{ChatID: "oc_sensitive", SessionID: "session-sensitive"}, filepath.Join(t.TempDir(), "card.json"))
	if err == nil || !strings.Contains(err.Error(), "exit code 17") {
		t.Fatalf("send error = %v", err)
	}
	for _, secret := range []string{"oc_sensitive", "session-sensitive", "credential-sensitive", "provider leaked"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("send error exposed %q: %v", secret, err)
		}
	}
}

func TestSendBotmuxTimeoutReportsUnknownOutcome(t *testing.T) {
	directory := t.TempDir()
	writeBotmuxTestExecutable(t, directory, "#!/bin/sh\nexec /bin/sleep 5\n")
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := sendBotmux(ctx, BotmuxBinding{ChatID: "oc_timeout", SessionID: "session-timeout"}, filepath.Join(t.TempDir(), "card.json"))
	if err == nil || !strings.Contains(err.Error(), "outcome unknown") || !strings.Contains(err.Error(), "not retrying") {
		t.Fatalf("timeout error = %v", err)
	}
	if strings.Contains(err.Error(), "oc_timeout") || strings.Contains(err.Error(), "session-timeout") {
		t.Fatalf("timeout error exposed binding: %v", err)
	}
}

func writeBotmuxTestExecutable(t *testing.T, directory, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(directory, "botmux"), []byte(content), 0o700); err != nil {
		t.Fatal(err)
	}
}
