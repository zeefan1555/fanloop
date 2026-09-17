package e2e

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestLocalBuildAndInstallKeepMatchedSourceWithoutExemplarMedia(t *testing.T) {
	repository := repositoryRoot(t)
	git := exec.Command("git", "rev-parse", "HEAD")
	git.Dir = repository
	commit, err := git.Output()
	if err != nil {
		t.Fatal(err)
	}
	dataRoot, codexRoot, agentsRoot, traeRoot, claudeRoot := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
	environment := append(withoutBotmuxBinding(os.Environ()),
		"GOOS=js", "GOARCH=wasm",
		"FANLOOP_DATA_HOME="+dataRoot,
		"FANLOOP_CODEX_SKILLS_ROOT="+codexRoot,
		"FANLOOP_AGENT_SKILLS_ROOT="+agentsRoot,
		"FANLOOP_TRAE_SKILLS_ROOT="+traeRoot,
		"FANLOOP_CLAUDE_SKILLS_ROOT="+claudeRoot,
	)
	buildRoot := filepath.Join(t.TempDir(), "build")
	build := exec.Command(filepath.Join(repository, "scripts", "build-local.sh"), buildRoot)
	build.Dir, build.Env = t.TempDir(), environment
	output, err := build.Output()
	if err != nil {
		t.Fatalf("local build: %v\n%s", err, commandStderr(err))
	}
	resolvedBuild, err := filepath.EvalSymlinks(buildRoot)
	if err != nil {
		t.Fatal(err)
	}
	outputPath := strings.TrimSpace(string(output))
	resolvedOutput, err := filepath.EvalSymlinks(outputPath)
	if err != nil || !filepath.IsAbs(outputPath) || resolvedOutput != resolvedBuild {
		t.Fatalf("build stdout = %q, want only absolute build directory %q: %v", output, buildRoot, err)
	}
	assertLocalBuildContents(t, repository, buildRoot)
	if _, err := os.Lstat(filepath.Join(dataRoot, "current")); !os.IsNotExist(err) {
		t.Fatalf("build activated an installation: %v", err)
	}

	installBuild := filepath.Join(t.TempDir(), "install-build")
	install := exec.Command(filepath.Join(repository, "scripts", "install-local.sh"), installBuild)
	install.Dir, install.Env = t.TempDir(), environment
	installOutput, err := install.Output()
	if err != nil {
		t.Fatalf("local install: %v\n%s", err, commandStderr(err))
	}
	var installed struct {
		OK   bool `json:"ok"`
		Data struct {
			ReleaseVersion string `json:"release_version"`
		} `json:"data"`
	}
	baseVersion, err := os.ReadFile(filepath.Join(repository, "VERSION"))
	if err != nil {
		t.Fatal(err)
	}
	versionPattern := regexp.MustCompile("^" + regexp.QuoteMeta(strings.TrimSpace(string(baseVersion))) + `-dev\.[0-9a-f]{12}$`)
	if err := json.Unmarshal(installOutput, &installed); err != nil || !installed.OK || !versionPattern.MatchString(installed.Data.ReleaseVersion) {
		t.Fatalf("local install did not return its local release: %v\n%s", err, installOutput)
	}
	version := installed.Data.ReleaseVersion
	assertInstalledRelease(t, dataRoot, codexRoot, agentsRoot, releaseFixture{ConfigSource: repository, Version: version}, traeRoot, claudeRoot)
	assertLocalBuildContents(t, repository, filepath.Join(dataRoot, "current"))
	if fixtureDirectoryDigest(t, installBuild) != fixtureDirectoryDigest(t, filepath.Join(dataRoot, "releases", version)) {
		t.Fatal("installation changed the built local directory")
	}

	for _, command := range []string{"version", "doctor"} {
		check := exec.Command(filepath.Join(dataRoot, "current", "bin", "fanloop"), command)
		check.Env = environment
		output, err := check.CombinedOutput()
		if err != nil {
			t.Fatalf("installed %s: %v\n%s", command, err, output)
		}
		if command == "doctor" {
			if !bytes.Contains(output, []byte(`"status": "healthy"`)) {
				t.Fatalf("installed Doctor is unhealthy: %s", output)
			}
			continue
		}
		var response struct {
			Data struct {
				Commit  string `json:"commit_sha"`
				Version string `json:"release_version"`
			} `json:"data"`
		}
		if err := json.Unmarshal(output, &response); err != nil || response.Data.Commit != strings.TrimSpace(string(commit)) || response.Data.Version != version {
			t.Fatalf("installed version does not match source commit and manifest: %v\n%s", err, output)
		}
	}
}

func assertLocalBuildContents(t *testing.T, repository, root string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, len(entries))
	for index, entry := range entries {
		names[index] = entry.Name()
	}
	if want := []string{"bin", "entrypoints", "release.json", "workflows"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("local build entries = %v, want one copy of each component: %v", names, want)
	}
	for _, component := range []string{"entrypoints"} {
		if fixtureDirectoryDigest(t, filepath.Join(repository, component)) != fixtureDirectoryDigest(t, filepath.Join(root, component)) {
			t.Fatalf("local build changed or omitted %s files", component)
		}
	}
	workflows, err := filepath.Glob(filepath.Join(repository, "workflows", "*", "workflow.yaml"))
	if err != nil || len(workflows) == 0 {
		t.Fatalf("source workflows: %v", err)
	}
	installedWorkflows, err := os.ReadDir(filepath.Join(root, "workflows"))
	if err != nil || len(installedWorkflows) != len(workflows) {
		t.Fatalf("local workflow count = %d, want %d: %v", len(installedWorkflows), len(workflows), err)
	}
	for _, workflow := range workflows {
		name := filepath.Base(filepath.Dir(workflow))
		if fixtureDirectoryDigest(t, filepath.Dir(workflow)) != fixtureDirectoryDigest(t, filepath.Join(root, "workflows", name)) {
			t.Fatalf("local build changed or omitted workflow %s", name)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "skills")); !os.IsNotExist(err) {
		t.Fatalf("local build contains live Skill configuration: %v", err)
	}
	sourceImages, err := filepath.Glob(filepath.Join(repository, "exemplars", "technical-solution", "*", "images", "*"))
	if err != nil || len(sourceImages) == 0 {
		t.Fatalf("source-only exemplar images = %d: %v", len(sourceImages), err)
	}
	var manifest struct {
		Schema int `json:"schema_version"`
		CLI    struct {
			BinarySHA256 string `json:"binary_sha256"`
		} `json:"cli"`
	}
	content, err := os.ReadFile(filepath.Join(root, "release.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(content, &manifest); err != nil || manifest.Schema != 3 || manifest.CLI.BinarySHA256 != fixtureFileDigest(t, filepath.Join(root, "bin", "fanloop")) {
		t.Fatalf("local manifest does not bind the native binary: %v\n%s", err, content)
	}
}

func commandStderr(err error) string {
	if exit, ok := err.(*exec.ExitError); ok {
		return string(exit.Stderr)
	}
	return ""
}
