package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPackagedNPMInstallerUsesItsMatchedReleaseManifest(t *testing.T) {
	repository := repositoryRoot(t)
	fixture := makeReleaseFixture(t, repository, "1.2.3", "1.2.3")
	dist := t.TempDir()
	staging := t.TempDir()
	extractFixture := exec.Command("tar", "-xf", fixture.Archive, "-C", staging)
	if output, err := extractFixture.CombinedOutput(); err != nil {
		t.Fatalf("extract release fixture: %v\n%s", err, output)
	}
	hostArchive := ""
	for _, target := range []struct{ os, arch string }{
		{"darwin", "amd64"}, {"darwin", "arm64"}, {"linux", "amd64"}, {"linux", "arm64"},
	} {
		name := "fanloop-1.2.3-" + target.os + "-" + target.arch + ".tar"
		binary := filepath.Join(staging, "bin", "fanloop")
		linker := strings.Join([]string{
			"-s",
			"-w",
			"-X github.com/zeefan1555/fanloop/internal/buildinfo.ReleaseVersion=1.2.3",
			"-X github.com/zeefan1555/fanloop/internal/buildinfo.CLIVersion=1.2.3",
			"-X github.com/zeefan1555/fanloop/internal/buildinfo.Commit=package-test",
		}, " ")
		build := exec.Command("go", "build", "-buildvcs=false", "-ldflags", linker, "-o", binary, ".")
		build.Dir = repository
		build.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+target.os, "GOARCH="+target.arch)
		if output, err := build.CombinedOutput(); err != nil {
			t.Fatalf("build %s/%s release: %v\n%s", target.os, target.arch, err, output)
		}
		archive := filepath.Join(dist, name)
		writeArchive := exec.Command("tar", "-cf", archive, "-C", staging, "bin", "skills", "workflows")
		writeArchive.Env = append(os.Environ(), "COPYFILE_DISABLE=1")
		if output, err := writeArchive.CombinedOutput(); err != nil {
			t.Fatalf("archive %s/%s release: %v\n%s", target.os, target.arch, err, output)
		}
		if target.os == runtime.GOOS && target.arch == runtime.GOARCH {
			hostArchive = strings.TrimSuffix(archive, ".tar") + ".tar.xz"
		}
	}
	if hostArchive == "" {
		t.Fatalf("test host %s/%s has no release archive", runtime.GOOS, runtime.GOARCH)
	}

	pack := exec.Command(filepath.Join(repository, "scripts", "package-release.sh"), "1.2.3", dist)
	pack.Dir = repository
	output, err := pack.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			t.Fatalf("package release: %v\n%s", err, exitError.Stderr)
		}
		t.Fatal(err)
	}
	artifact := strings.TrimSpace(string(output))
	artifactInfo, err := os.Stat(artifact)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("npm package = %d bytes", artifactInfo.Size())
	if artifactInfo.Size() > 15*1024*1024 {
		t.Fatalf("npm package = %d bytes, want at most 15 MiB", artifactInfo.Size())
	}
	unpacked := t.TempDir()
	extract := exec.Command("tar", "-xzf", artifact, "-C", unpacked)
	if output, err := extract.CombinedOutput(); err != nil {
		t.Fatalf("extract npm package: %v\n%s", err, output)
	}
	packageRoot := filepath.Join(unpacked, "package")
	manifest, err := os.ReadFile(filepath.Join(packageRoot, "release.json"))
	if err != nil || !bytes.Contains(manifest, []byte(`"release_version": "1.2.3"`)) {
		t.Fatalf("npm package has no matched release.json: %v\n%s", err, manifest)
	}
	if readme, err := os.ReadFile(filepath.Join(packageRoot, "README.md")); err != nil || !bytes.Contains(readme, []byte("~/.fanloop/current")) || !bytes.Contains(readme, []byte("fanloop-workflow")) {
		t.Fatalf("npm package has no installation README: %v\n%s", err, readme)
	}

	dataRoot, codexRoot, agentsRoot, traeRoot, claudeRoot, npmPrefix, npmCache := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
	install := exec.Command("npx", "--yes", "--package", artifact, "fanloop", "install")
	install.Env = append(os.Environ(),
		"NPM_CONFIG_PREFIX="+npmPrefix,
		"NPM_CONFIG_CACHE="+npmCache,
		"NPM_CONFIG_UPDATE_NOTIFIER=false",
		"FANLOOP_DATA_HOME="+dataRoot,
		"FANLOOP_CODEX_SKILLS_ROOT="+codexRoot,
		"FANLOOP_AGENT_SKILLS_ROOT="+agentsRoot,
		"FANLOOP_TRAE_SKILLS_ROOT="+traeRoot,
		"FANLOOP_CLAUDE_SKILLS_ROOT="+claudeRoot,
	)
	installOutput, installErr := install.CombinedOutput()
	if installErr != nil {
		diagnosticRoot := t.TempDir()
		_ = exec.Command("tar", "-xf", hostArchive, "-C", diagnosticRoot).Run()
		_ = os.WriteFile(filepath.Join(diagnosticRoot, "release.json"), manifest, 0o600)
		diagnostic := exec.Command(filepath.Join(diagnosticRoot, "bin", "fanloop"), "doctor")
		diagnosticOutput, _ := diagnostic.CombinedOutput()
		t.Fatalf("install packaged release: %v\n%s\ndoctor:\n%s", installErr, installOutput, diagnosticOutput)
	}
	if want := "Fanloop 1.2.3 installed successfully\n"; string(installOutput) != want {
		t.Fatalf("npx install stdout = %q, want %q", installOutput, want)
	}
	assertInstalledRelease(t, dataRoot, codexRoot, agentsRoot, "1.2.3", traeRoot, claudeRoot)
	assertSkillLink(t, dataRoot, traeRoot)
	assertSkillLink(t, dataRoot, claudeRoot)
	for _, root := range []string{traeRoot, claudeRoot} {
		installedLinks, err := os.ReadDir(root)
		if err != nil || len(installedLinks) != 1 || installedLinks[0].Name() != "fanloop-workflow" {
			t.Fatalf("installed Skill links in %s = %#v, want only fanloop-workflow: %v", root, installedLinks, err)
		}
	}
	if err := os.RemoveAll(npmCache); err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(npmPrefix, "bin", "fanloop")
	version := exec.Command(launcher, "version")
	version.Env = install.Env
	if output, err := version.CombinedOutput(); err != nil || !bytes.Contains(output, []byte(`"release_version": "1.2.3"`)) {
		t.Fatalf("persistent launcher failed after npx exited: %v\n%s", err, output)
	}
}
