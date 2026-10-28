package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/zeefan1555/fanloop/internal/controller"
	"github.com/zeefan1555/fanloop/internal/testfixture"
)

func TestPinControllerFailurePreservesFactsAndUnknownHome(t *testing.T) {
	source := t.TempDir()
	if err := testfixture.BuildRelease(source, "test"); err != nil {
		t.Fatal(err)
	}
	config := makeSyntheticConfigSource(t, repositoryRoot(t), "sample-custom")
	for _, unknownHome := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing-release", true: "unknown-home"}[unknownHome], func(t *testing.T) {
			root := t.TempDir()
			before := map[string][]byte{
				".fanloop/flow/state.json":        []byte("{}\n"),
				".fanloop/output/state.json":      []byte("{}\n"),
				".fanloop/trace/events.jsonl":     []byte("{}\n"),
				".fanloop/configuration-snapshot": []byte(config + "\n"),
			}
			if unknownHome {
				before[filepath.Join(controller.Home, "user-owned")] = []byte("preserve me\n")
			}
			for relative, content := range before {
				path := filepath.Join(root, relative)
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, content, 0600); err != nil {
					t.Fatal(err)
				}
			}
			controllerRelease := source
			if !unknownHome {
				controllerRelease = filepath.Join(source, "missing-release")
			}
			command := exec.Command("/bin/bash", filepath.Join(repositoryRoot(t), "scripts", "pin-controller-release.sh"), root)
			command.Env = append(withoutBotmuxBinding(os.Environ()),
				"FANLOOP_PROTECT_BINARY="+filepath.Join(source, "bin", "fanloop"),
				"FANLOOP_CONTROLLER_RELEASE="+controllerRelease,
				"FANLOOP_DATA_HOME="+t.TempDir(), "FANLOOP_NO_AUTO_UPDATE=1", "FANLOOP_NO_AUTO_CONFIG=1")
			var stdout, stderr bytes.Buffer
			command.Stdout, command.Stderr = &stdout, &stderr
			failure, ok := command.Run().(*exec.ExitError)
			if !ok || failure.ExitCode() != 5 || stdout.Len() != 0 || stderr.Len() == 0 {
				t.Fatalf("protect failure: %v\n%s\n%s", failure, stdout.String(), stderr.String())
			}
			for relative, expected := range before {
				if content, err := os.ReadFile(filepath.Join(root, relative)); err != nil || !bytes.Equal(content, expected) {
					t.Fatalf("failed protect changed %s: %v\n%s", relative, err, content)
				}
			}
			for _, relative := range []string{controller.Journal, controller.Pending} {
				if _, err := os.Lstat(filepath.Join(root, relative)); !os.IsNotExist(err) {
					t.Fatalf("failed protect created %s: %v", relative, err)
				}
			}
			if !unknownHome {
				if _, err := os.Lstat(filepath.Join(root, controller.Home)); !os.IsNotExist(err) {
					t.Fatalf("failed protect created controller: %v", err)
				}
			}
		})
	}
}
