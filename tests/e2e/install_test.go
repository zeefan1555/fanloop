package e2e

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

type releaseFixture struct {
	Archive  string
	Manifest string
	Version  string
}

func TestNPMInstallerActivatesOneVerifiedReleaseAndIsIdempotent(t *testing.T) {
	repository := repositoryRoot(t)
	fixture := makeReleaseFixture(t, repository, "1.2.3", "1.2.3")
	dataRoot, codexRoot, agentsRoot, traeRoot, claudeRoot := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()

	first := runInstaller(t, repository, fixture, dataRoot, codexRoot, agentsRoot, traeRoot, claudeRoot)
	if first.err != nil {
		t.Fatalf("clean install: %v\nstdout: %s\nstderr: %s", first.err, first.stdout, first.stderr)
	}
	if want := "Fanloop 1.2.3 installed successfully\n"; first.stdout != want {
		t.Fatalf("install stdout = %q, want %q", first.stdout, want)
	}
	assertInstalledRelease(t, dataRoot, codexRoot, agentsRoot, fixture.Version, traeRoot, claudeRoot)
	assertSkillLink(t, dataRoot, traeRoot)
	assertSkillLink(t, dataRoot, claudeRoot)
	currentBefore, _ := os.Readlink(filepath.Join(dataRoot, "current"))

	second := runInstaller(t, repository, fixture, dataRoot, codexRoot, agentsRoot, traeRoot, claudeRoot)
	if second.err != nil {
		t.Fatalf("repeat install: %v\nstdout: %s\nstderr: %s", second.err, second.stdout, second.stderr)
	}
	if currentAfter, _ := os.Readlink(filepath.Join(dataRoot, "current")); currentAfter != currentBefore {
		t.Fatalf("repeat install changed current from %q to %q", currentBefore, currentAfter)
	}

	runtimeCache := filepath.Join(dataRoot, "current", "entrypoints", "fanloop-workflow", "__pycache__", "runtime.pyc")
	if err := os.MkdirAll(filepath.Dir(runtimeCache), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runtimeCache, []byte("runtime cache"), 0o600); err != nil {
		t.Fatal(err)
	}
	repaired := runInstaller(t, repository, fixture, dataRoot, codexRoot, agentsRoot, traeRoot, claudeRoot)
	if repaired.err != nil {
		t.Fatalf("repair install: %v\nstdout: %s\nstderr: %s", repaired.err, repaired.stdout, repaired.stderr)
	}
	if _, err := os.Stat(runtimeCache); !os.IsNotExist(err) {
		t.Fatalf("repair install retained runtime cache: %v", err)
	}
	assertInstalledRelease(t, dataRoot, codexRoot, agentsRoot, fixture.Version, traeRoot, claudeRoot)

	launcher := exec.Command("node", filepath.Join(repository, "scripts", "run.js"), "version")
	launcher.Env = append(os.Environ(), "FANLOOP_DATA_HOME="+dataRoot)
	output, err := launcher.CombinedOutput()
	if err != nil || !bytes.Contains(output, []byte(`"release_version": "1.2.3"`)) || !bytes.Contains(output, []byte(`"name": "fanloop-workflow"`)) {
		t.Fatalf("launcher did not use current release: %v\n%s", err, output)
	}

	for _, args := range [][]string{
		{"flow", "init", "--help"}, {"flow", "status", "--help"},
		{"flow", "report", "progress", "--help"}, {"flow", "report", "result", "--help"},
		{"trace", "bind", "--help"}, {"trace", "status", "--help"},
		{"trace", "render", "--help"}, {"trace", "sync", "--help"},
		{"card", "render", "--help"}, {"version", "--help"}, {"doctor", "--help"},
	} {
		result := runCurrent(dataRoot, codexRoot, agentsRoot, "", args...)
		if result.err != nil || result.stderr != "" || !strings.Contains(result.stdout, "Request JSON:") {
			t.Fatalf("installed fanloop %s: %v\nstdout: %s\nstderr: %s", strings.Join(args, " "), result.err, result.stdout, result.stderr)
		}
	}
}

func TestNPMInstallerExposesOnlyWorkflowSkillAndPreservesAtomicSkillDirectories(t *testing.T) {
	repository := repositoryRoot(t)
	fixture := makeReleaseFixture(t, repository, "1.2.3", "1.2.3")
	dataRoot, codexRoot, agentsRoot, traeRoot, claudeRoot := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()

	for _, root := range []string{codexRoot, agentsRoot, traeRoot, claudeRoot} {
		path := filepath.Join(root, "techdesign")
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "owned-by-user"), []byte("preserve me\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	result := runInstaller(t, repository, fixture, dataRoot, codexRoot, agentsRoot, traeRoot, claudeRoot)
	if result.err != nil {
		t.Fatalf("install with atomic Skill directories: %v\nstdout: %s\nstderr: %s", result.err, result.stdout, result.stderr)
	}
	for _, skillID := range []string{
		"technical-problem-framing", "technical-problem-approval", "technical-solution-derivation",
		"technical-direction-approval", "technical-solution-writing", "technical-solution-review",
		"technical-solution-approval",
	} {
		path := filepath.Join(dataRoot, "releases", fixture.Version, "skills", "technical-solution-design", skillID, "SKILL.md")
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("packaged %s Skill: %v", skillID, err)
		}
	}
	for _, root := range []string{codexRoot, agentsRoot, traeRoot, claudeRoot} {
		marker := filepath.Join(root, "techdesign", "owned-by-user")
		if content, err := os.ReadFile(marker); err != nil || string(content) != "preserve me\n" {
			t.Fatalf("user Skill directory changed at %s: %v\n%s", root, err, content)
		}
		assertSkillLink(t, dataRoot, root)
		links := 0
		entries, err := os.ReadDir(root)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			info, err := os.Lstat(filepath.Join(root, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode()&os.ModeSymlink != 0 {
				links++
			}
		}
		if links != 1 {
			t.Fatalf("managed Skill links in %s = %d, want 1", root, links)
		}
	}

	requirementRoot := t.TempDir()
	initialized := runCurrent(dataRoot, codexRoot, agentsRoot, "", "flow", "init", "--root", requirementRoot, "--workflow", "technical-solution-design", "--title", "Technical solution Skill path E2E")
	if initialized.err != nil {
		t.Fatalf("initialize installed release: %v\nstdout: %s\nstderr: %s", initialized.err, initialized.stdout, initialized.stderr)
	}
	assertFlowSkillPaths(t, initialized.stdout, filepath.Join(dataRoot, "releases", fixture.Version))
}

func TestNPMInstallerUpgradesFromLegacyCurrentManifest(t *testing.T) {
	repository := repositoryRoot(t)
	oldRelease := makeReleaseFixture(t, repository, "1.2.3", "1.2.3", "legacy-only")
	newRelease := makeReleaseFixture(t, repository, "1.2.4", "1.2.4")
	dataRoot, codexRoot, agentsRoot, traeRoot := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()

	if result := runInstaller(t, repository, oldRelease, dataRoot, codexRoot, agentsRoot, traeRoot); result.err != nil {
		t.Fatalf("seed legacy install: %v\nstdout: %s\nstderr: %s", result.err, result.stdout, result.stderr)
	}
	rewriteCurrentWorkflowPaths(t, dataRoot, func(path string) string { return path + "/workflow.yaml" })

	result := runInstaller(t, repository, newRelease, dataRoot, codexRoot, agentsRoot, traeRoot)
	if result.err != nil {
		t.Fatalf("upgrade from legacy current manifest: %v\nstdout: %s\nstderr: %s", result.err, result.stdout, result.stderr)
	}
	assertInstalledRelease(t, dataRoot, codexRoot, agentsRoot, newRelease.Version, traeRoot)
	for _, root := range []string{codexRoot, agentsRoot, traeRoot} {
		if _, err := os.Lstat(filepath.Join(root, "legacy-only")); !os.IsNotExist(err) {
			t.Fatalf("legacy-only Skill link was not retired from %s: %v", root, err)
		}
	}
}

func TestNPMInstallerKeepsCurrentOnChecksumDoctorAndNameConflicts(t *testing.T) {
	repository := repositoryRoot(t)
	good := makeReleaseFixture(t, repository, "1.2.3", "1.2.3")
	dataRoot, codexRoot, agentsRoot := t.TempDir(), t.TempDir(), t.TempDir()
	if result := runInstaller(t, repository, good, dataRoot, codexRoot, agentsRoot); result.err != nil {
		t.Fatalf("seed install: %v\n%s", result.err, result.stderr)
	}
	currentBefore, _ := os.Readlink(filepath.Join(dataRoot, "current"))

	badChecksum := good
	badChecksum.Manifest = replaceAssetDigest(t, good.Manifest, "sha256:"+strings.Repeat("0", 64))
	if result := runInstaller(t, repository, badChecksum, dataRoot, codexRoot, agentsRoot); result.err == nil || !strings.Contains(result.stderr, "checksum") {
		t.Fatalf("checksum failure = %v\nstdout: %s\nstderr: %s", result.err, result.stdout, result.stderr)
	}
	assertCurrent(t, dataRoot, currentBefore)

	badDoctor := makeReleaseFixture(t, repository, "1.2.4", "9.9.9")
	if result := runInstaller(t, repository, badDoctor, dataRoot, codexRoot, agentsRoot); result.err == nil || !strings.Contains(result.stderr, "Doctor") {
		t.Fatalf("doctor failure = %v\nstdout: %s\nstderr: %s", result.err, result.stdout, result.stderr)
	}
	assertCurrent(t, dataRoot, currentBefore)
	assertInstalledRelease(t, dataRoot, codexRoot, agentsRoot, good.Version)
	if _, err := os.Stat(filepath.Join(dataRoot, "releases", "1.2.4")); !os.IsNotExist(err) {
		t.Fatalf("failed release was retained: %v", err)
	}

	conflictData, conflictCodex, conflictAgents := t.TempDir(), t.TempDir(), t.TempDir()
	conflict := filepath.Join(conflictCodex, "fanloop-workflow")
	if err := os.WriteFile(conflict, []byte("owned by user\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if result := runInstaller(t, repository, good, conflictData, conflictCodex, conflictAgents); result.err == nil || !strings.Contains(result.stderr, "refusing to replace") {
		t.Fatalf("name conflict = %v\nstdout: %s\nstderr: %s", result.err, result.stdout, result.stderr)
	}
	if got, _ := os.ReadFile(conflict); string(got) != "owned by user\n" {
		t.Fatal("installer overwrote the conflicting user file")
	}
	if _, err := os.Lstat(filepath.Join(conflictData, "current")); !os.IsNotExist(err) {
		t.Fatalf("conflicting install activated current: %v", err)
	}
}

func TestNPMInstallerAdoptsExternalSkillLinksWithoutDeletingTheirTargets(t *testing.T) {
	repository := repositoryRoot(t)
	fixture := makeReleaseFixture(t, repository, "1.2.3", "1.2.3")
	dataRoot, codexRoot, agentsRoot, traeRoot, claudeRoot := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()

	externalTargets := []string{}
	for _, root := range []string{codexRoot, agentsRoot, traeRoot, claudeRoot} {
		externalTarget := t.TempDir()
		marker := filepath.Join(externalTarget, "owned-by-another-manager")
		if err := os.WriteFile(marker, []byte("preserve me\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(externalTarget, filepath.Join(root, "fanloop-workflow")); err != nil {
			t.Fatal(err)
		}
		externalTargets = append(externalTargets, marker)
	}

	result := runInstaller(t, repository, fixture, dataRoot, codexRoot, agentsRoot, traeRoot, claudeRoot)
	if result.err != nil {
		t.Fatalf("install with external Skill links: %v\nstdout: %s\nstderr: %s", result.err, result.stdout, result.stderr)
	}
	assertInstalledRelease(t, dataRoot, codexRoot, agentsRoot, fixture.Version, traeRoot, claudeRoot)
	for _, marker := range externalTargets {
		content, err := os.ReadFile(marker)
		if err != nil || string(content) != "preserve me\n" {
			t.Fatalf("external Skill target changed: %v\n%s", err, content)
		}
	}
}

func TestDoctorChecksExposedWorkflowSkillLinks(t *testing.T) {
	repository := repositoryRoot(t)
	fixture := makeReleaseFixture(t, repository, "1.2.3", "1.2.3")
	dataRoot, codexRoot, agentsRoot := t.TempDir(), t.TempDir(), t.TempDir()
	traeRoot := filepath.Join(agentsRoot, ".trae-skills")
	claudeRoot := filepath.Join(agentsRoot, ".claude-skills")
	if result := runInstaller(t, repository, fixture, dataRoot, codexRoot, agentsRoot); result.err != nil {
		t.Fatalf("install: %v\n%s", result.err, result.stderr)
	}
	if healthy := runCurrent(dataRoot, codexRoot, agentsRoot, "", "doctor"); healthy.err != nil || !strings.Contains(healthy.stdout, `"status": "healthy"`) {
		t.Fatalf("healthy install failed Doctor: %#v", healthy)
	}
	pinned := filepath.Join(codexRoot, "fanloop-workflow")
	if err := os.Remove(pinned); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dataRoot, "releases", fixture.Version, "entrypoints", "fanloop-workflow"), pinned); err != nil {
		t.Fatal(err)
	}
	if diagnosed := runCurrent(dataRoot, codexRoot, agentsRoot, "", "doctor"); diagnosed.err == nil || !strings.Contains(diagnosed.stdout, `"id": "skill_links"`) || !strings.Contains(diagnosed.stdout, `"status": "failed"`) {
		t.Fatalf("Doctor accepted a pinned Skill link: %#v", diagnosed)
	}
	if err := os.Remove(pinned); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dataRoot, "current", "entrypoints", "fanloop-workflow"), pinned); err != nil {
		t.Fatal(err)
	}
	for client, root := range map[string]string{"Trae": traeRoot, "Claude": claudeRoot} {
		link := filepath.Join(root, "fanloop-workflow")
		if err := os.Remove(link); err != nil {
			t.Fatal(err)
		}
		broken := runCurrent(dataRoot, codexRoot, agentsRoot, "", "doctor")
		if broken.err == nil || !strings.Contains(broken.stdout, `"id": "skill_links"`) || !strings.Contains(broken.stdout, `"status": "failed"`) {
			t.Fatalf("Doctor missed broken %s Skill link: %#v", client, broken)
		}
		if err := os.Symlink(filepath.Join(dataRoot, "current", "entrypoints", "fanloop-workflow"), link); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDoctorAcceptsManagedLinksWithSymlinkedDataRoot(t *testing.T) {
	repository := repositoryRoot(t)
	fixture := makeReleaseFixture(t, repository, "1.2.3", "1.2.3")
	parent := t.TempDir()
	realDataRoot := filepath.Join(parent, "real-data")
	dataRoot := filepath.Join(parent, "data-link")
	if err := os.Mkdir(realDataRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realDataRoot, dataRoot); err != nil {
		t.Fatal(err)
	}
	codexRoot, agentsRoot := t.TempDir(), t.TempDir()
	if result := runInstaller(t, repository, fixture, dataRoot, codexRoot, agentsRoot); result.err != nil {
		t.Fatalf("install through symlinked data root: %v\n%s", result.err, result.stderr)
	}
	binary := filepath.Join(realDataRoot, "releases", fixture.Version, "bin", "fanloop")
	command := exec.Command(binary, "doctor")
	command.Env = append(os.Environ(),
		"FANLOOP_DATA_HOME="+dataRoot,
		"FANLOOP_CODEX_SKILLS_ROOT="+codexRoot,
		"FANLOOP_AGENT_SKILLS_ROOT="+agentsRoot,
		"FANLOOP_TRAE_SKILLS_ROOT="+filepath.Join(agentsRoot, ".trae-skills"),
		"FANLOOP_CLAUDE_SKILLS_ROOT="+filepath.Join(agentsRoot, ".claude-skills"),
	)
	output, err := command.CombinedOutput()
	if err != nil || !bytes.Contains(output, []byte(`"status": "healthy"`)) {
		t.Fatalf("Doctor rejected symlinked data root: %v\n%s", err, output)
	}
}

type installResult struct {
	stdout string
	stderr string
	err    error
}

func runInstaller(t *testing.T, repository string, fixture releaseFixture, dataRoot, codexRoot, agentsRoot string, additionalRoots ...string) installResult {
	t.Helper()
	traeRoot := filepath.Join(agentsRoot, ".trae-skills")
	claudeRoot := filepath.Join(agentsRoot, ".claude-skills")
	if len(additionalRoots) > 0 {
		traeRoot = additionalRoots[0]
	}
	if len(additionalRoots) > 1 {
		claudeRoot = additionalRoots[1]
	}
	command := exec.Command("node", filepath.Join(repository, "scripts", "install.js"))
	command.Env = append(os.Environ(),
		"FANLOOP_RELEASE_ARCHIVE="+fixture.Archive,
		"FANLOOP_RELEASE_MANIFEST="+fixture.Manifest,
		"FANLOOP_DATA_HOME="+dataRoot,
		"FANLOOP_CODEX_SKILLS_ROOT="+codexRoot,
		"FANLOOP_AGENT_SKILLS_ROOT="+agentsRoot,
		"FANLOOP_TRAE_SKILLS_ROOT="+traeRoot,
		"FANLOOP_CLAUDE_SKILLS_ROOT="+claudeRoot,
	)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	return installResult{stdout: stdout.String(), stderr: stderr.String(), err: err}
}

func assertSkillLink(t *testing.T, dataRoot, skillsRoot string) {
	t.Helper()
	want := filepath.Join(dataRoot, "current", "entrypoints", "fanloop-workflow")
	path := filepath.Join(skillsRoot, "fanloop-workflow")
	target, err := os.Readlink(path)
	if err != nil || target != want {
		t.Fatalf("skill link %s -> %q (%v), want %q", path, target, err, want)
	}
}

func assertFlowSkillPaths(t *testing.T, output, releaseRoot string) {
	t.Helper()
	resolvedReleaseRoot, err := filepath.EvalSymlinks(releaseRoot)
	if err != nil {
		t.Fatal(err)
	}
	manifestContent, err := os.ReadFile(filepath.Join(resolvedReleaseRoot, "release.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Skills []struct {
			Name string `json:"name"`
			Path string `json:"path"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(manifestContent, &manifest); err != nil {
		t.Fatal(err)
	}
	skillPaths := map[string]string{}
	for _, skill := range manifest.Skills {
		skillPaths[skill.Name] = filepath.Join(resolvedReleaseRoot, filepath.FromSlash(skill.Path), "SKILL.md")
	}
	var response any
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("decode Flow response: %v\n%s", err, output)
	}
	count := 0
	var visit func(any)
	visit = func(value any) {
		switch typed := value.(type) {
		case []any:
			for _, item := range typed {
				visit(item)
			}
		case map[string]any:
			id, hasID := typed["id"].(string)
			_, hasPrompt := typed["prompt"].(string)
			_, hasOptional := typed["optional"].(bool)
			if hasID && hasPrompt && hasOptional {
				count++
				path, ok := typed["path"].(string)
				if !ok {
					t.Errorf("Flow Skill %q has no path: %#v", id, typed)
				} else {
					want := skillPaths[id]
					if path != want {
						t.Errorf("Flow Skill %q path = %q, want %q", id, path, want)
					} else if _, err := os.Stat(path); err != nil {
						t.Errorf("Flow Skill %q path: %v", id, err)
					}
				}
			}
			for _, item := range typed {
				visit(item)
			}
		}
	}
	visit(response)
	if count == 0 {
		t.Fatal("Flow response contains no structured Skills")
	}
}

func assertInstalledRelease(t *testing.T, dataRoot, codexRoot, agentsRoot, version string, additionalRoots ...string) {
	t.Helper()
	wantCurrent := filepath.Join("releases", version)
	assertCurrent(t, dataRoot, wantCurrent)
	if _, err := os.Stat(filepath.Join(dataRoot, "releases", version, "bin", "fanloop")); err != nil {
		t.Fatalf("installed binary: %v", err)
	}
	traeRoot := filepath.Join(agentsRoot, ".trae-skills")
	claudeRoot := filepath.Join(agentsRoot, ".claude-skills")
	if len(additionalRoots) > 0 {
		traeRoot = additionalRoots[0]
	}
	if len(additionalRoots) > 1 {
		claudeRoot = additionalRoots[1]
	}
	manifestContent, err := os.ReadFile(filepath.Join(dataRoot, "releases", version, "release.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Skills []struct {
			Name string `json:"name"`
			Path string `json:"path"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(manifestContent, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{codexRoot, agentsRoot, traeRoot, claudeRoot} {
		assertSkillLink(t, dataRoot, root)
	}
	for _, skill := range manifest.Skills {
		if _, err := os.Stat(filepath.Join(dataRoot, "releases", version, filepath.FromSlash(skill.Path), "SKILL.md")); err != nil {
			t.Fatalf("packaged Skill %s: %v", skill.Name, err)
		}
		if skill.Name == "fanloop-workflow" {
			continue
		}
		for _, root := range []string{codexRoot, agentsRoot, traeRoot, claudeRoot} {
			path := filepath.Join(root, skill.Name)
			if _, err := os.Lstat(path); !os.IsNotExist(err) {
				t.Fatalf("atomic Skill %s was globally exposed in %s: %v", skill.Name, root, err)
			}
		}
	}
}

func assertCurrent(t *testing.T, dataRoot, want string) {
	t.Helper()
	got, err := os.Readlink(filepath.Join(dataRoot, "current"))
	if err != nil || got != want {
		t.Fatalf("current -> %q (%v), want %q", got, err, want)
	}
}

func rewriteCurrentWorkflowPaths(t *testing.T, dataRoot string, rewrite func(string) string) {
	t.Helper()
	manifestPath := filepath.Join(dataRoot, "current", "release.json")
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(content, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, raw := range manifest["workflows"].([]any) {
		workflow := raw.(map[string]any)
		workflow["path"] = rewrite(workflow["path"].(string))
	}
	updated, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, append(updated, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func makeReleaseFixture(t *testing.T, repository, releaseVersion, compiledVersion string, additionalSkillNames ...string) releaseFixture {
	t.Helper()
	staging := t.TempDir()
	binary := filepath.Join(staging, "bin", "fanloop")
	if err := os.MkdirAll(filepath.Dir(binary), 0o755); err != nil {
		t.Fatal(err)
	}
	linker := strings.Join([]string{
		"-X github.com/zeefan1555/fanloop/internal/buildinfo.ReleaseVersion=" + compiledVersion,
		"-X github.com/zeefan1555/fanloop/internal/buildinfo.CLIVersion=" + compiledVersion,
		"-X github.com/zeefan1555/fanloop/internal/buildinfo.Commit=install-test",
	}, " ")
	build := exec.Command("go", "build", "-buildvcs=false", "-ldflags", linker, "-o", binary, ".")
	build.Dir = repository
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build release binary: %v\n%s", err, output)
	}

	skillItems := []map[string]any{}
	skillSources := []string{filepath.Join(repository, "entrypoints", "fanloop-workflow", "SKILL.md")}
	matches, err := filepath.Glob(filepath.Join(repository, "skills", "*", "*", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	skillSources = append(skillSources, matches...)
	sort.Strings(skillSources)
	for _, skillFile := range skillSources {
		skillSource := filepath.Dir(skillFile)
		name := filepath.Base(skillSource)
		relative, err := filepath.Rel(repository, skillSource)
		if err != nil {
			t.Fatal(err)
		}
		relative = filepath.ToSlash(relative)
		skill := filepath.Join(staging, filepath.FromSlash(relative))
		if err := filepath.WalkDir(skillSource, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() {
				return walkErr
			}
			relative, err := filepath.Rel(skillSource, path)
			if err != nil {
				return err
			}
			copyTreeFile(t, path, filepath.Join(skill, relative))
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		skillItems = append(skillItems, map[string]any{
			"name": name, "version": releaseVersion, "path": relative, "sha256": fixtureDirectoryDigest(t, skill),
		})
	}
	for _, name := range additionalSkillNames {
		relative := filepath.ToSlash(filepath.Join("skills", "fanloop-maintainer", name))
		directory := filepath.Join(staging, filepath.FromSlash(relative))
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "SKILL.md"), []byte("---\nname: "+name+"\n---\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		skillItems = append(skillItems, map[string]any{
			"name": name, "version": releaseVersion, "path": relative, "sha256": fixtureDirectoryDigest(t, directory),
		})
	}
	workflowItems := []map[string]any{}
	workflowPaths, err := filepath.Glob(filepath.Join(repository, "workflows", "*", "workflow.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range workflowPaths {
		sourceRoot := filepath.Dir(source)
		loaded, decodeErr := workflow.LoadDirectory(sourceRoot)
		if decodeErr != nil {
			t.Fatalf("read workflow Bundle %s: %v", sourceRoot, decodeErr)
		}
		relative, err := filepath.Rel(repository, sourceRoot)
		if err != nil {
			t.Fatal(err)
		}
		relative = filepath.ToSlash(relative)
		for _, name := range workflow.BundleFileNames() {
			copyTreeFile(t, filepath.Join(sourceRoot, name), filepath.Join(staging, filepath.FromSlash(relative), name))
		}
		workflowItems = append(workflowItems, map[string]any{
			"id": loaded.Ref.ID, "path": relative, "sha256": loaded.Ref.Digest,
		})
	}

	archive := filepath.Join(t.TempDir(), fmt.Sprintf("fanloop-%s-%s-%s.tar.xz", releaseVersion, runtime.GOOS, runtime.GOARCH))
	writeTarXZ(t, staging, archive)
	assets := []map[string]any{}
	for _, target := range []struct{ os, arch string }{
		{"darwin", "amd64"}, {"darwin", "arm64"}, {"linux", "amd64"}, {"linux", "arm64"},
	} {
		archiveDigest, binaryDigest := "sha256:"+strings.Repeat("0", 64), "sha256:"+strings.Repeat("0", 64)
		if target.os == runtime.GOOS && target.arch == runtime.GOARCH {
			archiveDigest, binaryDigest = fixtureFileDigest(t, archive), fixtureFileDigest(t, binary)
		}
		assets = append(assets, map[string]any{
			"os": target.os, "arch": target.arch,
			"file":   fmt.Sprintf("fanloop-%s-%s-%s.tar.xz", releaseVersion, target.os, target.arch),
			"sha256": archiveDigest, "binary_sha256": binaryDigest,
		})
	}
	manifest := map[string]any{
		"schema_version": 2, "release_version": releaseVersion,
		"cli": map[string]any{"version": releaseVersion},
		"state_schema": map[string]any{
			"read_versions": []int{state.CurrentStateSchemaVersion},
			"write_version": state.CurrentStateSchemaVersion,
		},
		"skills":    skillItems,
		"workflows": workflowItems,
		"assets":    assets,
	}
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(t.TempDir(), "release.json")
	if err := os.WriteFile(manifestPath, append(content, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return releaseFixture{Archive: archive, Manifest: manifestPath, Version: releaseVersion}
}

func replaceAssetDigest(t *testing.T, source, digest string) string {
	t.Helper()
	content, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(content, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, raw := range manifest["assets"].([]any) {
		asset := raw.(map[string]any)
		if asset["os"] == runtime.GOOS && asset["arch"] == runtime.GOARCH {
			asset["sha256"] = digest
		}
	}
	updated, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "release.json")
	if err := os.WriteFile(path, append(updated, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func copyTreeFile(t *testing.T, source, target string) {
	t.Helper()
	content, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, content, 0o600); err != nil {
		t.Fatal(err)
	}
}

func fixtureFileDigest(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func fixtureDirectoryDigest(t *testing.T, root string) string {
	t.Helper()
	paths := []string{}
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type().IsRegular() {
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			paths = append(paths, filepath.ToSlash(relative))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	hash := sha256.New()
	for _, relative := range paths {
		hash.Write([]byte(relative))
		hash.Write([]byte{0})
		hash.Write([]byte(strings.TrimPrefix(fixtureFileDigest(t, filepath.Join(root, filepath.FromSlash(relative))), "sha256:")))
		hash.Write([]byte{'\n'})
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func writeTarXZ(t *testing.T, root, target string) {
	t.Helper()
	command := exec.Command("tar", "-cJf", target, "-C", root, "bin", "entrypoints", "skills", "workflows")
	command.Env = append(os.Environ(), "COPYFILE_DISABLE=1", "XZ_OPT=-0")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("write XZ fixture: %v\n%s", err, output)
	}
}
