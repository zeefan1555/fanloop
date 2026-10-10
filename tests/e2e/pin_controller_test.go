package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPinControllerCleansOnlyFailedNewHome(t *testing.T) {
	repository := repositoryRoot(t)
	script := filepath.Join(repository, "scripts", "pin-controller-release.sh")
	production, err := os.ReadFile(script)
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := os.ReadFile(filepath.Join(repository, "tests", "e2e", "testdata", "pin-controller-release.sh"))
	if err != nil || !bytes.Equal(production, fixture) {
		t.Fatalf("controller test asset differs: %v", err)
	}
	home := t.TempDir()
	root, source, data := filepath.Join(home, "requirement"), filepath.Join(home, "source"), filepath.Join(home, "data")
	for _, dir := range []string{filepath.Join(root, ".fanloop", "flow"), filepath.Join(source, "skills"), filepath.Join(data, "current", "release", "bin")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".fanloop", "flow", "state.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	stub := "#!/bin/sh\nif [ \"$1\" = __install ]; then echo deliberate-install-failure >&2; exit 1; fi\nexit 0\n"
	if err := os.WriteFile(filepath.Join(data, "current", "release", "bin", "fanloop"), []byte(stub), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(data, "config"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(source, filepath.Join(data, "config", "current")); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("/bin/bash", script, root)
	command.Env = append(withoutBotmuxBinding(os.Environ()), "HOME="+home, "FANLOOP_DATA_HOME="+data)
	var output bytes.Buffer
	command.Stdout, command.Stderr = &output, &output
	controller := filepath.Join(root, "bound-release-home")
	err = command.Run()
	if err == nil || !strings.Contains(output.String(), "deliberate-install-failure") {
		t.Fatalf("unexpected pin failure: %v %s", err, output.String())
	}
	if _, err := os.Lstat(controller); !os.IsNotExist(err) {
		t.Fatalf("failed new home retained: %v", err)
	}
	if err := os.Mkdir(controller, 0700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(controller, "user-owned")
	if err := os.WriteFile(marker, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	retry := exec.Command("/bin/bash", script, root)
	retry.Env = command.Env
	if output, err := retry.CombinedOutput(); err == nil || !strings.Contains(string(output), "existing pinned controller is invalid") {
		t.Fatalf("existing invalid controller accepted: %v\n%s", err, output)
	}
	if content, err := os.ReadFile(marker); err != nil || string(content) != "keep" {
		t.Fatalf("existing controller was changed: %q, %v", content, err)
	}
}
