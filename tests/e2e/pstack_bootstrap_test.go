package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPstackBootstrapKeepsSourceImmutable(t *testing.T) {
	if testing.Short() {
		t.Skip("first-run tools require Bun and a dependency download")
	}
	bun, err := exec.LookPath("bun")
	if err != nil {
		t.Skip("Bun is not installed")
	}
	repository, home := repositoryRoot(t), t.TempDir()
	source, data := filepath.Join(home, "source"), filepath.Join(home, "data")
	copyFixtureDirectory(t, filepath.Join(repository, "skills"), filepath.Join(source, "skills"))
	before := fixtureDirectoryDigest(t, source)
	environment := append(withoutBotmuxBinding(os.Environ()),
		"FANLOOP_DATA_HOME="+data, "FANLOOP_CONFIG_SOURCE="+source,
		"FANLOOP_CODEX_SKILLS_ROOT="+filepath.Join(home, "codex"),
		"FANLOOP_AGENT_SKILLS_ROOT="+filepath.Join(home, "agent"),
		"FANLOOP_TRAE_SKILLS_ROOT="+filepath.Join(home, "trae"),
		"FANLOOP_CLAUDE_SKILLS_ROOT="+filepath.Join(home, "claude"),
	)
	install := exec.Command(filepath.Join(repository, "scripts", "install-local.sh"), filepath.Join(home, "build"))
	install.Env, install.Dir = environment, home
	if output, err := install.CombinedOutput(); err != nil {
		t.Fatalf("isolated install: %v\n%s", err, output)
	}
	cli := func(args ...string) {
		t.Helper()
		command := exec.Command(filepath.Join(data, "current", "bin", "fanloop"), args...)
		command.Env, command.Dir = environment, home
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("fanloop %v: %v\n%s", args, err, output)
		}
	}
	root := filepath.Join(home, "requirement")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	cli("doctor")
	cli("flow", "init", "--root", root, "--workflow", "fanloop-maintainer", "--title", "Tool source integrity")
	scripts := filepath.Join(source, "skills", "pstack-native", "skills", "poteto-mode", "scripts")
	toolEnvironment := []string{"HOME=" + home, "PATH=" + os.Getenv("PATH"), "XDG_CACHE_HOME=" + filepath.Join(home, "cache"), "BUN_INSTALL_CACHE_DIR=" + filepath.Join(home, "bun-cache")}
	for _, phase := range []string{"cold concurrent", "cache hit"} {
		results := make(chan error, 2)
		for _, tool := range []string{"orch/orch.ts", "watch-pr/watch-pr"} {
			go func(tool string) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				defer cancel()
				command := exec.CommandContext(ctx, bun, filepath.Join(scripts, filepath.FromSlash(tool)), "--help")
				command.Env, command.Dir = toolEnvironment, home
				output, err := command.CombinedOutput()
				t.Logf("%s %s --help: %v\n%s", phase, tool, err, output)
				if err == nil && !strings.Contains(string(output), "Usage:") {
					err = fmt.Errorf("help output missing")
				}
				results <- err
			}(tool)
		}
		for range 2 {
			if err := <-results; err != nil {
				t.Errorf("%s tool invocation: %v", phase, err)
			}
		}
		if t.Failed() {
			return
		}
		if after := fixtureDirectoryDigest(t, source); after != before {
			t.Fatalf("%s tool invocation mutated the configuration source", phase)
		}
		cli("doctor")
		cli("flow", "status", "--root", root)
	}
}

func TestPstackBootstrapFailureCleansOnlyItsTemporaryCache(t *testing.T) {
	bun, err := exec.LookPath("bun")
	if err != nil {
		t.Skip("Bun is not installed")
	}
	home := t.TempDir()
	source := filepath.Join(home, "scripts")
	copyFixtureDirectory(t, filepath.Join(repositoryRoot(t), "skills", "pstack-native", "skills", "poteto-mode", "scripts"), source)
	if err := os.WriteFile(filepath.Join(source, "bun.lock"), []byte("invalid lockfile\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(home, "cache", "poteto-mode-tools")
	if err := os.MkdirAll(cache, 0700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(cache, "existing-user-file")
	if err := os.WriteFile(marker, []byte("keep\n"), 0600); err != nil {
		t.Fatal(err)
	}
	before := fixtureDirectoryDigest(t, source)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, bun, filepath.Join(source, "orch", "orch.ts"), "--help")
	command.Dir = home
	command.Env = []string{"HOME=" + home, "PATH=" + os.Getenv("PATH"), "XDG_CACHE_HOME=" + filepath.Join(home, "cache"), "BUN_INSTALL_CACHE_DIR=" + filepath.Join(home, "bun-cache")}
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "bun install exited") {
		t.Fatalf("expected failed dependency installation: %v\n%s", err, output)
	}
	entries, err := os.ReadDir(cache)
	if err != nil || len(entries) != 1 || entries[0].Name() != "existing-user-file" {
		t.Fatalf("failed bootstrap changed other cache files or leaked a temporary directory: %v, %v", entries, err)
	}
	if content, err := os.ReadFile(marker); err != nil || string(content) != "keep\n" {
		t.Fatalf("existing cache marker changed: %q, %v", content, err)
	}
	if after := fixtureDirectoryDigest(t, source); after != before {
		t.Fatal("failed bootstrap mutated source")
	}
}
