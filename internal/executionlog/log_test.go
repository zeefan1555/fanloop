package executionlog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/zeefan1555/commonloop/internal/idl/storageidl"
	"golang.org/x/sys/unix"
)

func TestAppendCreatesSecureJSONL(t *testing.T) {
	root := t.TempDir()
	entry := testEntry("invocation-1")
	if err := Append(root, entry); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, ".commonloop", "log")
	path := filepath.Join(directory, filename)
	for _, item := range []struct {
		path string
		mode os.FileMode
	}{{directory, 0o700}, {path, 0o600}} {
		info, err := os.Stat(item.path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != item.mode {
			t.Fatalf("%s mode = %04o, want %04o", item.path, got, item.mode)
		}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var decoded storageidl.CLIExecutionLogEntry
	if err := json.Unmarshal(content, &decoded); err != nil {
		t.Fatal(err)
	}
	if err := decoded.IsValid(); err != nil {
		t.Fatal(err)
	}
	if !decoded.DeepEqual(entry) {
		t.Fatalf("decoded = %#v, want %#v", decoded, *entry)
	}
}

func TestAppendRejectsSymlinkTargets(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		root := t.TempDir()
		directory := filepath.Join(root, ".commonloop", "log")
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		outside := filepath.Join(t.TempDir(), "outside")
		if err := os.WriteFile(outside, []byte("sentinel"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(directory, filename)); err != nil {
			t.Fatal(err)
		}
		if err := Append(root, testEntry("file-link")); err == nil {
			t.Fatal("Append followed a symlinked log file")
		}
		content, err := os.ReadFile(outside)
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != "sentinel" {
			t.Fatalf("symlink target changed: %q", content)
		}
	})

	t.Run("directory", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, ".commonloop"), 0o700); err != nil {
			t.Fatal(err)
		}
		outside := t.TempDir()
		if err := os.Symlink(outside, filepath.Join(root, ".commonloop", "log")); err != nil {
			t.Fatal(err)
		}
		if err := Append(root, testEntry("directory-link")); err == nil {
			t.Fatal("Append followed a symlinked log directory")
		}
		if _, err := os.Stat(filepath.Join(outside, filename)); !os.IsNotExist(err) {
			t.Fatalf("symlinked directory received log file: %v", err)
		}
	})
}

func TestAppendRejectsNamedPipeWithoutBlocking(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, ".commonloop", "log")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, filename)
	if err := unix.Mkfifo(path, 0o600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- Append(root, testEntry("named-pipe")) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Append accepted a named pipe")
		}
	case <-time.After(time.Second):
		t.Fatal("Append blocked while opening a named pipe")
	}
}

func TestAppendSerializesConcurrentLines(t *testing.T) {
	root := t.TempDir()
	const count = 32
	errors := make(chan error, count)
	var wait sync.WaitGroup
	for index := 0; index < count; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			errors <- Append(root, testEntry(fmt.Sprintf("invocation-%d", index)))
		}(index)
	}
	wait.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}

	file, err := os.Open(filepath.Join(root, ".commonloop", "log", filename))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	seen := map[string]bool{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var entry storageidl.CLIExecutionLogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Fatalf("decode complete line: %v: %q", err, scanner.Bytes())
		}
		seen[entry.InvocationId] = true
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if len(seen) != count {
		t.Fatalf("unique lines = %d, want %d", len(seen), count)
	}
}

func TestReadAllReturnsExactSnapshotAndRejectsUnsafeFiles(t *testing.T) {
	t.Run("exact bytes", func(t *testing.T) {
		root := t.TempDir()
		content := []byte("first\nsecond without newline ` and ```\n")
		path := filepath.Join(root, ".commonloop", "log", filename)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := ReadAll(root)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(content) {
			t.Fatalf("ReadAll = %q, want exact %q", got, content)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		got, err := ReadAll(t.TempDir())
		if err != nil || len(got) != 0 {
			t.Fatalf("ReadAll missing file = %q, %v", got, err)
		}
	})

	t.Run("symlink", func(t *testing.T) {
		root := t.TempDir()
		directory := filepath.Join(root, ".commonloop", "log")
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		outside := filepath.Join(t.TempDir(), "outside")
		if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(directory, filename)); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadAll(root); err == nil {
			t.Fatal("ReadAll followed a symlinked log file")
		}
	})

	t.Run("symlinked directory", func(t *testing.T) {
		root := t.TempDir()
		if err := os.Mkdir(filepath.Join(root, ".commonloop"), 0o700); err != nil {
			t.Fatal(err)
		}
		outside := t.TempDir()
		if err := os.WriteFile(filepath.Join(outside, filename), []byte("secret"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(root, ".commonloop", "log")); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadAll(root); err == nil {
			t.Fatal("ReadAll followed a symlinked log directory")
		}
	})

	t.Run("named pipe", func(t *testing.T) {
		root := t.TempDir()
		directory := filepath.Join(root, ".commonloop", "log")
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := unix.Mkfifo(filepath.Join(directory, filename), 0o600); err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() {
			_, err := ReadAll(root)
			done <- err
		}()
		select {
		case err := <-done:
			if err == nil {
				t.Fatal("ReadAll accepted a named pipe")
			}
		case <-time.After(time.Second):
			t.Fatal("ReadAll blocked while opening a named pipe")
		}
	})
}

func testEntry(id string) *storageidl.CLIExecutionLogEntry {
	return &storageidl.CLIExecutionLogEntry{
		SchemaVersion:  storageidl.CLI_EXECUTION_LOG_SCHEMA_VERSION,
		InvocationId:   id,
		StartedAt:      time.Unix(0, 0).UTC().Format(time.RFC3339Nano),
		DurationMs:     1,
		CommandId:      "flow.status",
		CliVersion:     "test-cli",
		ReleaseVersion: "test-release",
		CommitSha:      "test-commit",
		ExitCode:       0,
		Arguments:      []string{"flow", "status"},
		Stdout:         "{}\n",
	}
}
