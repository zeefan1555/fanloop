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

func TestGitHubReleaseInstallerUsesItsMatchedReleaseManifest(t *testing.T) {
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
		writeArchive := exec.Command("tar", "-cf", archive, "-C", staging, "bin", "entrypoints", "skills", "workflows")
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

	assemble := exec.Command(filepath.Join(repository, "scripts", "assemble-release.sh"), "1.2.3", dist)
	assemble.Dir = repository
	if output, err := assemble.CombinedOutput(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			t.Fatalf("assemble GitHub Release: %v\n%s\n%s", err, output, exitError.Stderr)
		}
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(dist, "release.json"))
	if err != nil || !bytes.Contains(manifest, []byte(`"release_version": "1.2.3"`)) {
		t.Fatalf("GitHub Release has no matched release.json: %v\n%s", err, manifest)
	}
	for _, asset := range []string{"fanloop-install.sh", "fanloop-install.js", "fanloop-launcher.sh"} {
		if _, err := os.Stat(filepath.Join(dist, asset)); err != nil {
			t.Fatalf("GitHub Release has no %s: %v", asset, err)
		}
	}

	dataRoot, codexRoot, agentsRoot, traeRoot, claudeRoot, binRoot := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
	install := exec.Command("bash", filepath.Join(repository, "scripts", "install-github.sh"))
	install.Env = append(os.Environ(),
		"FANLOOP_RELEASE_DIR="+dist,
		"FANLOOP_DATA_HOME="+dataRoot,
		"FANLOOP_BIN_DIR="+binRoot,
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
		t.Fatalf("install GitHub Release: %v\n%s\ndoctor:\n%s", installErr, installOutput, diagnosticOutput)
	}
	if want := "Fanloop 1.2.3 installed successfully\n"; string(installOutput) != want {
		t.Fatalf("GitHub Release install stdout = %q, want %q", installOutput, want)
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
	if err := os.RemoveAll(dist); err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(binRoot, "fanloop")
	version := exec.Command(launcher, "version")
	version.Env = install.Env
	if output, err := version.CombinedOutput(); err != nil || !bytes.Contains(output, []byte(`"release_version": "1.2.3"`)) {
		t.Fatalf("persistent launcher failed after release assets were removed: %v\n%s", err, output)
	}
}
