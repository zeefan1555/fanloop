package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalBuildGuardDetectsContentChangesWithUnchangedGitStatus(t *testing.T) {
	for _, name := range []string{"tracked.go", "untracked source [1].go"} {
		t.Run(name, func(t *testing.T) {
			repository, environment := localBuildGuardRepository(t)
			source := filepath.Join(repository, name)
			if err := os.WriteFile(source, []byte("dirty before build\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			before := localBuildGuardGit(t, repository, "status", "--porcelain")
			command := exec.Command(filepath.Join(repository, "scripts", "build-local.sh"))
			command.Env = append(environment, "FANLOOP_GUARD_MUTATE="+source)
			output, err := command.CombinedOutput()
			if err == nil || !strings.Contains(string(output), "Source changed during build") {
				t.Fatalf("source modification was not rejected: %v\n%s", err, output)
			}
			if after := localBuildGuardGit(t, repository, "status", "--porcelain"); after != before {
				t.Fatalf("test changed Git status instead of only content: before=%q after=%q", before, after)
			}
		})
	}
}

func TestLocalBuildGuardAllowsRepositoryOutputWithLiteralPath(t *testing.T) {
	repository, environment := localBuildGuardRepository(t)
	outputRoot := filepath.Join(repository, "new build [1]")
	command := exec.Command(filepath.Join(repository, "scripts", "build-local.sh"), outputRoot)
	command.Env = environment
	output, err := command.Output()
	if err != nil {
		t.Fatalf("local output rejected: %v\n%s", err, commandStderr(err))
	}
	if strings.TrimSpace(string(output)) != outputRoot {
		t.Fatalf("build stdout = %q, want %q", output, outputRoot)
	}
	if _, err := os.Stat(filepath.Join(outputRoot, "release.json")); err != nil {
		t.Fatal(err)
	}
}

func TestLocalBuildGuardDoesNotVersionLiveSkillChanges(t *testing.T) {
	repository, environment := localBuildGuardRepository(t)
	commit := strings.TrimSpace(localBuildGuardGit(t, repository, "rev-parse", "HEAD"))
	skill := filepath.Join(repository, "skills", "example", "SKILL.md")
	if err := os.WriteFile(skill, []byte("changed live configuration\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(filepath.Join(repository, "scripts", "build-local.sh"), filepath.Join(t.TempDir(), "build"))
	command.Env = environment
	output, err := command.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "Building 1.2.3-dev."+commit[:12]+" from") {
		t.Fatalf("Skill-only change affected the CLI version: %v\n%s", err, output)
	}
}

func TestLocalBuildGuardUsesDistinctStableVersionsForCleanCommits(t *testing.T) {
	repository, environment := localBuildGuardRepository(t)
	buildVersion := func(outputRoot string) string {
		t.Helper()
		commit := strings.TrimSpace(localBuildGuardGit(t, repository, "rev-parse", "HEAD"))
		command := exec.Command(filepath.Join(repository, "scripts", "build-local.sh"), outputRoot)
		command.Env = environment
		output, err := command.CombinedOutput()
		version := "1.2.3-dev." + commit[:12]
		if err != nil || !strings.Contains(string(output), "Building "+version+" from "+commit) {
			t.Fatalf("clean local build did not use the commit identity: %v\n%s", err, output)
		}
		return version
	}

	first := buildVersion(filepath.Join(t.TempDir(), "first"))
	localBuildGuardGit(t, repository, "-c", "user.name=Guard Test", "-c", "user.email=guard@example.invalid", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null", "commit", "--allow-empty", "-qm", "second")
	second := buildVersion(filepath.Join(t.TempDir(), "second"))
	repeated := buildVersion(filepath.Join(t.TempDir(), "repeated"))
	if first == second || second != repeated {
		t.Fatalf("clean build versions = first %q, second %q, repeated %q", first, second, repeated)
	}
}

func TestLocalBuildGuardRejectsOutputInsideCopiedOrGitTrees(t *testing.T) {
	for _, parent := range []string{"skills", "entrypoints", "workflows", ".git", "skills-link"} {
		t.Run(parent, func(t *testing.T) {
			repository, environment := localBuildGuardRepository(t)
			if parent == "skills-link" {
				if err := os.Symlink(filepath.Join(repository, "skills"), filepath.Join(repository, parent)); err != nil {
					t.Fatal(err)
				}
			}
			outputRoot := filepath.Join(repository, parent, "nested build")
			command := exec.Command(filepath.Join(repository, "scripts", "build-local.sh"), outputRoot)
			command.Env = environment
			output, err := command.CombinedOutput()
			if err == nil || !strings.Contains(string(output), "outside copied component trees and Git metadata") {
				t.Fatalf("unsafe output was not rejected: %v\n%s", err, output)
			}
			if _, err := os.Lstat(outputRoot); !os.IsNotExist(err) {
				t.Fatalf("rejected output path was created: %v", err)
			}
		})
	}
}

func TestLocalBuildGuardRejectsInvalidVersion(t *testing.T) {
	repository, environment := localBuildGuardRepository(t)
	if err := os.WriteFile(filepath.Join(repository, "VERSION"), []byte("local-build\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(filepath.Join(repository, "scripts", "build-local.sh"))
	command.Env = environment
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "VERSION must contain a semantic version") {
		t.Fatalf("invalid version was not rejected: %v\n%s", err, output)
	}
}

// Only Go is replaced: the guard runs the real script and Git against a tiny repo.
func localBuildGuardRepository(t *testing.T) (string, []string) {
	t.Helper()
	repository, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	script, err := os.ReadFile(filepath.Join(repositoryRoot(t), "scripts", "build-local.sh"))
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string][]byte{
		"scripts/build-local.sh": script,
		"VERSION":                []byte("1.2.3\n"),
		".gitignore":             []byte("/dist/\n"), "tracked.go": []byte("initial\n"),
		"skills/example/SKILL.md":         []byte("example\n"),
		"entrypoints/example/SKILL.md":    []byte("entry\n"),
		"workflows/example/workflow.yaml": []byte("example\n"),
	} {
		path := filepath.Join(repository, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	localBuildGuardGit(t, repository, "init", "-q")
	localBuildGuardGit(t, repository, "config", "core.fsmonitor", "false")
	localBuildGuardGit(t, repository, "config", "maintenance.auto", "false")
	localBuildGuardGit(t, repository, "config", "gc.auto", "0")
	localBuildGuardGit(t, repository, "add", ".")
	localBuildGuardGit(t, repository, "-c", "user.name=Guard Test", "-c", "user.email=guard@example.invalid", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null", "commit", "-qm", "fixture")
	bin := t.TempDir()
	fakeGo := `#!/usr/bin/env bash
set -euo pipefail
case "$1" in
  env)
    case "$2" in GOHOSTOS) echo linux;; GOHOSTARCH) echo amd64;; *) exit 1;; esac
    ;;
  build)
    while [[ $# -gt 0 ]]; do
      if [[ "$1" == "-o" ]]; then printf 'binary\n' > "$2"; chmod +x "$2"; break; fi
      shift
    done
    if [[ -n "${FANLOOP_GUARD_MUTATE:-}" ]]; then printf 'changed after compilation\n' >> "$FANLOOP_GUARD_MUTATE"; fi
    ;;
  run)
    while [[ $# -gt 0 ]]; do
      if [[ "$1" == "--output" ]]; then printf '{}\n' > "$2"; break; fi
      shift
    done
    ;;
  *) exit 1;;
esac
`
	if err := os.WriteFile(filepath.Join(bin, "go"), []byte(fakeGo), 0o755); err != nil {
		t.Fatal(err)
	}
	return repository, append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "FANLOOP_GUARD_MUTATE=")
}

func localBuildGuardGit(t *testing.T, repository string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = repository
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return string(output)
}
