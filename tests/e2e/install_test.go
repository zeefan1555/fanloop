package e2e

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/release"
	"github.com/zeefan1555/fanloop/internal/state"
)

type releaseFixture struct {
	Directory    string
	ConfigSource string
	Version      string
}

func TestLocalInstallerActivatesOneVerifiedReleaseAndIsIdempotent(t *testing.T) {
	repository := repositoryRoot(t)
	fixture := makeReleaseFixture(t, repository, "1.2.3", "1.2.3")
	dataRoot, codexRoot, agentsRoot, traeRoot, claudeRoot := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()

	first := runInstaller(t, fixture, dataRoot, codexRoot, agentsRoot, traeRoot, claudeRoot)
	if first.err != nil {
		t.Fatalf("clean install: %v\nstdout: %s\nstderr: %s", first.err, first.stdout, first.stderr)
	}
	if !strings.Contains(first.stdout, `"release_version": "1.2.3"`) || !strings.Contains(first.stdout, `"command": "__install"`) {
		t.Fatalf("install did not return matched release: %s", first.stdout)
	}
	assertInstalledRelease(t, dataRoot, codexRoot, agentsRoot, fixture, traeRoot, claudeRoot)
	assertSkillLink(t, dataRoot, traeRoot)
	assertSkillLink(t, dataRoot, claudeRoot)
	currentBefore, _ := os.Readlink(filepath.Join(dataRoot, "current", "release"))

	second := runInstaller(t, fixture, dataRoot, codexRoot, agentsRoot, traeRoot, claudeRoot)
	if second.err != nil {
		t.Fatalf("repeat install: %v\nstdout: %s\nstderr: %s", second.err, second.stdout, second.stderr)
	}
	if currentAfter, _ := os.Readlink(filepath.Join(dataRoot, "current", "release")); currentAfter != currentBefore {
		t.Fatalf("repeat install changed current/release from %q to %q", currentBefore, currentAfter)
	}

	runtimeCache := filepath.Join(dataRoot, "current", "entrypoints", "fanloop-workflow", "__pycache__", "runtime.pyc")
	if err := os.MkdirAll(filepath.Dir(runtimeCache), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runtimeCache, []byte("runtime cache"), 0o600); err != nil {
		t.Fatal(err)
	}
	repaired := runInstaller(t, fixture, dataRoot, codexRoot, agentsRoot, traeRoot, claudeRoot)
	if repaired.err != nil {
		t.Fatalf("repair install: %v\nstdout: %s\nstderr: %s", repaired.err, repaired.stdout, repaired.stderr)
	}
	if _, err := os.Stat(runtimeCache); !os.IsNotExist(err) {
		t.Fatalf("repair install retained runtime cache: %v", err)
	}
	assertInstalledRelease(t, dataRoot, codexRoot, agentsRoot, fixture, traeRoot, claudeRoot)

	launcher := exec.Command(filepath.Join(dataRoot, "current", "bin", "fanloop"), "version")
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
		{"card", "render", "--help"}, {"verify", "doctor", "--help"},
		{"verify", "smoke", "--help"}, {"verify", "snapshot", "--help"},
		{"verify", "cleanup", "--help"}, {"version", "--help"}, {"doctor", "--help"}, {"dashboard", "--help"},
	} {
		result := runCurrent(dataRoot, codexRoot, agentsRoot, args...)
		if result.err != nil || result.stderr != "" || !strings.Contains(result.stdout, "Request JSON:") {
			t.Fatalf("installed fanloop %s: %v\nstdout: %s\nstderr: %s", strings.Join(args, " "), result.err, result.stdout, result.stderr)
		}
	}
}

func TestLocalInstallerBootstrapsEmptyCurrentWithoutConfigurationSource(t *testing.T) {
	repository := repositoryRoot(t)
	fixture := makeReleaseFixture(t, repository, "1.2.3", "1.2.3")
	fixture.ConfigSource = ""
	dataRoot, codexRoot, agentsRoot := t.TempDir(), t.TempDir(), t.TempDir()

	result := runInstaller(t, fixture, dataRoot, codexRoot, agentsRoot)
	if result.err != nil {
		t.Fatalf("fresh offline install: %v\nstdout: %s\nstderr: %s", result.err, result.stdout, result.stderr)
	}
	assertCurrent(t, dataRoot, filepath.Join("..", "releases", "1.2.3"))
	workflowRoot := filepath.Join(dataRoot, "current", "workflows")
	info, err := os.Lstat(workflowRoot)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("bootstrap Workflow root is not a real directory: %#v, %v", info, err)
	}
	entries, err := os.ReadDir(workflowRoot)
	if err != nil || len(entries) != 0 {
		t.Fatalf("bootstrap Workflow root entries = %#v, %v; want empty", entries, err)
	}
	version := runCurrent(dataRoot, codexRoot, agentsRoot, "version")
	if version.err != nil || !strings.Contains(version.stdout, `"release_version": "1.2.3"`) {
		t.Fatalf("bootstrap version: %v\nstdout: %s\nstderr: %s", version.err, version.stdout, version.stderr)
	}
	repeated := runInstaller(t, fixture, dataRoot, codexRoot, agentsRoot)
	if repeated.err != nil {
		t.Fatalf("repeat bootstrap install: %v\nstdout: %s\nstderr: %s", repeated.err, repeated.stdout, repeated.stderr)
	}
	entries, err = os.ReadDir(workflowRoot)
	if err != nil || len(entries) != 0 {
		t.Fatalf("repeat install changed bootstrap Workflow root: %#v, %v", entries, err)
	}
}

func TestFreshInstallerRejectsExplicitMissingConfigurationSource(t *testing.T) {
	repository := repositoryRoot(t)
	fixture := makeReleaseFixture(t, repository, "1.2.3", "1.2.3")
	fixture.ConfigSource = filepath.Join(t.TempDir(), "missing-configuration-source")
	dataRoot, codexRoot, agentsRoot := t.TempDir(), t.TempDir(), t.TempDir()

	result := runInstaller(t, fixture, dataRoot, codexRoot, agentsRoot)
	if result.err == nil || !strings.Contains(result.stderr, "source") {
		t.Fatalf("explicit missing configuration source accepted: %v\nstdout: %s\nstderr: %s", result.err, result.stdout, result.stderr)
	}
	if _, err := os.Lstat(filepath.Join(dataRoot, "current")); !os.IsNotExist(err) {
		t.Fatalf("failed explicit-source install activated current: %v", err)
	}
}

func TestLocalInstallerMigratesLegacyCurrentWithoutLosingWorkflows(t *testing.T) {
	repository := repositoryRoot(t)
	legacy := makeReleaseFixture(t, repository, "1.2.2", "1.2.2")
	upgrade := makeReleaseFixture(t, repository, "1.2.3", "1.2.3")
	upgrade.ConfigSource = ""
	dataRoot, codexRoot, agentsRoot := t.TempDir(), t.TempDir(), t.TempDir()
	legacyTarget := filepath.Join(dataRoot, "releases", legacy.Version)
	copyFixtureDirectory(t, legacy.Directory, legacyTarget)
	copyFixtureDirectory(t, filepath.Join(legacy.ConfigSource, "workflows"), filepath.Join(legacyTarget, "workflows"))
	if err := os.Symlink(filepath.Join("releases", legacy.Version), filepath.Join(dataRoot, "current")); err != nil {
		t.Fatal(err)
	}
	wantDigest := fixtureDirectoryDigest(t, filepath.Join(legacyTarget, "workflows"))

	result := runInstaller(t, upgrade, dataRoot, codexRoot, agentsRoot)
	if result.err != nil {
		t.Fatalf("legacy current migration: %v\nstdout: %s\nstderr: %s", result.err, result.stdout, result.stderr)
	}
	assertCurrent(t, dataRoot, filepath.Join("..", "releases", upgrade.Version))
	if got := fixtureDirectoryDigest(t, filepath.Join(dataRoot, "current", "workflows")); got != wantDigest {
		t.Fatalf("legacy Workflow digest = %s, want preserved %s", got, wantDigest)
	}
}

func TestReleaseUpgradePreservesLiveWorkflowWhilePinnedControllerRemainsUsable(t *testing.T) {
	repository := repositoryRoot(t)
	home := t.TempDir()
	dataRoot := filepath.Join(home, ".fanloop")
	codexRoot := filepath.Join(home, ".codex", "skills")
	agentsRoot := filepath.Join(home, ".agents", "skills")
	oldRelease := makeReleaseFixture(t, repository, "1.2.3", "1.2.3")
	oldRelease.ConfigSource = makeSyntheticControllerConfigSource(t, repository, "sample-custom")
	if result := runInstaller(t, oldRelease, dataRoot, codexRoot, agentsRoot); result.err != nil {
		t.Fatalf("install initializing release: %v\nstdout: %s\nstderr: %s", result.err, result.stdout, result.stderr)
	}

	oldRoot := filepath.Join(home, "fanloop", "issues", "old-requirement")
	if err := os.MkdirAll(oldRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	initialized := runCurrent(dataRoot, codexRoot, agentsRoot, "flow", "init", "--root", oldRoot, "--workflow", "sample-custom", "--title", "Pinned controller E2E")
	if initialized.err != nil {
		t.Fatalf("initialize old Requirement: %v\nstdout: %s\nstderr: %s", initialized.err, initialized.stdout, initialized.stderr)
	}

	pinner := filepath.Join(oldRelease.ConfigSource, "workflows", "sample-custom", "skills", "controller-tools", "scripts", "pin-controller-release.sh")
	pinned := exec.Command(pinner, oldRoot)
	pinned.Env = append(os.Environ(), "HOME="+home)
	if output, err := pinned.CombinedOutput(); err != nil {
		t.Fatalf("pin initializing controller: %v\n%s", err, output)
	}

	newRelease := makeReleaseFixtureWithChangedSyntheticPrompt(t, repository, "1.2.4", "1.2.4")
	if result := runInstaller(t, newRelease, dataRoot, codexRoot, agentsRoot); result.err != nil {
		t.Fatalf("install candidate release: %v\nstdout: %s\nstderr: %s", result.err, result.stdout, result.stderr)
	}
	repinned := exec.Command(pinner, oldRoot)
	repinned.Env = append(os.Environ(), "HOME="+home)
	if output, err := repinned.CombinedOutput(); err != nil {
		t.Fatalf("reuse pinned controller after current changed: %v\n%s", err, output)
	}
	controllerBinary := filepath.Join(oldRoot, "bound-release-home", "current", "bin", "fanloop")
	if err := os.Chmod(controllerBinary, 0o600); err != nil {
		t.Fatal(err)
	}
	invalidPinned := exec.Command(pinner, oldRoot)
	invalidPinned.Env = append(os.Environ(), "HOME="+home)
	invalidOutput, invalidErr := invalidPinned.CombinedOutput()
	if invalidErr == nil || !strings.Contains(string(invalidOutput), "existing pinned controller is invalid") || strings.Contains(string(invalidOutput), "WORKFLOW_MISMATCH") {
		t.Fatalf("invalid pinned controller did not fail closed: %v\n%s", invalidErr, invalidOutput)
	}
	if err := os.Chmod(controllerBinary, 0o755); err != nil {
		t.Fatal(err)
	}
	oldState, err := os.ReadFile(filepath.Join(oldRoot, ".fanloop", "flow", "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	globalOldStatus := runCurrent(dataRoot, codexRoot, agentsRoot, "flow", "status", "--root", oldRoot)
	if globalOldStatus.err != nil || strings.Contains(globalOldStatus.stdout, "candidate prompt") || !strings.Contains(globalOldStatus.stdout, "读取用户材料") {
		t.Fatalf("upgraded CLI did not preserve and reload the existing live Workflow: %v\nstdout: %s\nstderr: %s", globalOldStatus.err, globalOldStatus.stdout, globalOldStatus.stderr)
	}
	after, err := os.ReadFile(filepath.Join(oldRoot, ".fanloop", "flow", "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, oldState) {
		t.Fatal("candidate status rewrote old Requirement provenance")
	}
	assertCandidate := func(name string, args ...string) {
		t.Helper()
		result := runCurrent(dataRoot, codexRoot, agentsRoot, args...)
		if result.err != nil {
			t.Fatalf("candidate %s failed on old Requirement: %v\nstdout: %s\nstderr: %s", name, result.err, result.stdout, result.stderr)
		}
	}
	assertCandidate("progress", "flow", "report", "progress", "--root", oldRoot, "--step-id", "analyze_materials", "--status", "in_progress", "--summary", "continued by candidate")
	assertCandidate("card", "card", "render", "--root", oldRoot, "--view", "current", "--format", "markdown")
	assertCandidate("trace", "trace", "status", "--root", oldRoot)
	assertCandidate("sync", "trace", "sync", "--root", oldRoot)
	assertCandidate("doctor", "doctor", "--root", oldRoot)
	for _, path := range []string{
		filepath.Join(oldRoot, ".fanloop", "flow", "state.json"),
		filepath.Join(oldRoot, ".fanloop", "output", "state.json"),
		filepath.Join(oldRoot, ".fanloop", "trace", "events.jsonl"),
		filepath.Join(oldRoot, ".fanloop", "card", "projection.json"),
	} {
		content, err := os.ReadFile(path)
		if err != nil || bytes.Contains(content, []byte(`"digest"`)) {
			t.Fatalf("durable Requirement fact unexpectedly contains Workflow digest in %s: %v\n%s", path, err, content)
		}
	}

	controllerHome := filepath.Join(oldRoot, "bound-release-home")
	controllerStatus := runBoundController(controllerHome, "flow", "status", "--root", oldRoot)
	if controllerStatus.err != nil || strings.Contains(controllerStatus.stdout, "candidate prompt") {
		t.Fatalf("pinned controller did not continue old Requirement: %v\nstdout: %s\nstderr: %s", controllerStatus.err, controllerStatus.stdout, controllerStatus.stderr)
	}
	controllerVersion := runBoundController(controllerHome, "version")
	if controllerVersion.err != nil || !strings.Contains(controllerVersion.stdout, `"release_version": "1.2.3"`) {
		t.Fatalf("pinned controller version: %v\nstdout: %s\nstderr: %s", controllerVersion.err, controllerVersion.stdout, controllerVersion.stderr)
	}

	newRoot := filepath.Join(home, "fanloop", "issues", "new-requirement")
	if err := os.MkdirAll(newRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	newInitialized := runCurrent(dataRoot, codexRoot, agentsRoot, "flow", "init", "--root", newRoot, "--workflow", "sample-custom", "--title", "Candidate current E2E")
	if newInitialized.err != nil || strings.Contains(newInitialized.stdout, "candidate prompt") {
		t.Fatalf("candidate current did not initialize new Requirement: %v\nstdout: %s\nstderr: %s", newInitialized.err, newInitialized.stdout, newInitialized.stderr)
	}
	candidateVersion := runCurrent(dataRoot, codexRoot, agentsRoot, "version")
	if candidateVersion.err != nil || !strings.Contains(candidateVersion.stdout, `"release_version": "1.2.4"`) {
		t.Fatalf("candidate current version: %v\nstdout: %s\nstderr: %s", candidateVersion.err, candidateVersion.stdout, candidateVersion.stderr)
	}
}

func TestReleaseUpgradeDoesNotRequireConfigurationSource(t *testing.T) {
	repository := repositoryRoot(t)
	dataRoot, codexRoot, agentsRoot := t.TempDir(), t.TempDir(), t.TempDir()
	initial := makeReleaseFixture(t, repository, "1.2.3", "1.2.3")
	if result := runInstaller(t, initial, dataRoot, codexRoot, agentsRoot); result.err != nil {
		t.Fatalf("initial install: %v\nstdout: %s\nstderr: %s", result.err, result.stdout, result.stderr)
	}

	upgrade := makeReleaseFixture(t, repository, "1.2.4", "1.2.4")
	upgrade.ConfigSource = filepath.Join(t.TempDir(), "missing-configuration-source")
	result := runInstaller(t, upgrade, dataRoot, codexRoot, agentsRoot)
	if result.err != nil {
		t.Fatalf("upgrade depends on configuration source: %v\nstdout: %s\nstderr: %s", result.err, result.stdout, result.stderr)
	}
	assertCurrent(t, dataRoot, filepath.Join("..", "releases", "1.2.4"))
}

func TestLocalInstallerUsesLiveSkillConfiguration(t *testing.T) {
	repository := copyRepositorySource(t, repositoryRoot(t))
	fixture := makeReleaseFixture(t, repository, "1.2.3", "1.2.3")
	fixture.ConfigSource = makeSyntheticConfigSource(t, repository, "sample-custom")
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

	result := runInstaller(t, fixture, dataRoot, codexRoot, agentsRoot, traeRoot, claudeRoot)
	if result.err != nil {
		t.Fatalf("install with atomic Skill directories: %v\nstdout: %s\nstderr: %s", result.err, result.stdout, result.stderr)
	}
	if _, err := os.Stat(filepath.Join(dataRoot, "releases", fixture.Version, "skills")); !os.IsNotExist(err) {
		t.Fatalf("installed release contains live Skills: %v", err)
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
		if links != len(release.ExposedSkills()) {
			t.Fatalf("managed Skill links in %s = %d, want %d", root, links, len(release.ExposedSkills()))
		}
	}

	requirementRoot := t.TempDir()
	initialized := runCurrent(dataRoot, codexRoot, agentsRoot, "flow", "init", "--root", requirementRoot, "--workflow", "sample-custom", "--title", "Custom Skill path E2E")
	if initialized.err != nil {
		t.Fatalf("initialize installed release: %v\nstdout: %s\nstderr: %s", initialized.err, initialized.stdout, initialized.stderr)
	}
	liveRoot := filepath.Join(dataRoot, "current", "workflows")
	assertFlowSkillPaths(t, initialized.stdout, liveRoot)
	liveSkill := firstSkillPath(t, filepath.Join(liveRoot, "sample-custom", "skills"))
	content, err := os.ReadFile(liveSkill)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(liveSkill, append(content, []byte("\nlive config marker\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	status := runCurrent(dataRoot, codexRoot, agentsRoot, "flow", "status", "--root", requirementRoot)
	if status.err != nil || !strings.Contains(status.stdout, liveSkill) {
		t.Fatalf("Flow did not reuse the live Skill path after an edit: %v\n%s", status.err, status.stdout)
	}
	version := runCurrent(dataRoot, codexRoot, agentsRoot, "version")
	if version.err != nil || !strings.Contains(version.stdout, `"release_version": "1.2.3"`) {
		t.Fatalf("live Skill edit changed the CLI version: %v\n%s", version.err, version.stdout)
	}

}

func TestLocalInstallerPreservesConflictingCurrentPaths(t *testing.T) {
	repository := repositoryRoot(t)
	fixture := makeReleaseFixture(t, repository, "1.2.3", "1.2.3")
	for _, kind := range []string{"file", "directory", "external-link"} {
		t.Run(kind, func(t *testing.T) {
			dataRoot, codexRoot, agentsRoot := t.TempDir(), t.TempDir(), t.TempDir()
			current := filepath.Join(dataRoot, "current")
			marker := current
			switch kind {
			case "directory":
				if err := os.Mkdir(current, 0o755); err != nil {
					t.Fatal(err)
				}
				marker = filepath.Join(current, "owned-by-user")
			case "external-link":
				external := t.TempDir()
				if err := os.Symlink(external, current); err != nil {
					t.Fatal(err)
				}
				marker = filepath.Join(external, "owned-by-user")
			}
			if err := os.WriteFile(marker, []byte("preserve me\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			result := runInstaller(t, fixture, dataRoot, codexRoot, agentsRoot)
			if result.err == nil || (!strings.Contains(result.stderr, "refusing to replace") && !strings.Contains(result.stderr, "refusing invalid stable current path")) {
				t.Fatalf("current conflict accepted: %v\n%s", result.err, result.stderr)
			}
			if content, err := os.ReadFile(marker); err != nil || string(content) != "preserve me\n" {
				t.Fatalf("current conflict changed user content: %v\n%s", err, content)
			}
			if _, err := os.Lstat(filepath.Join(codexRoot, "fanloop-workflow")); !os.IsNotExist(err) {
				t.Fatalf("failed install created a Skill link: %v", err)
			}
		})
	}
}

func TestLocalInstallerKeepsCurrentOnChecksumDoctorAndNameConflicts(t *testing.T) {
	repository := repositoryRoot(t)
	good := makeReleaseFixture(t, repository, "1.2.3", "1.2.3")
	dataRoot, codexRoot, agentsRoot := t.TempDir(), t.TempDir(), t.TempDir()
	if result := runInstaller(t, good, dataRoot, codexRoot, agentsRoot); result.err != nil {
		t.Fatalf("seed install: %v\n%s", result.err, result.stderr)
	}
	currentBefore, _ := os.Readlink(filepath.Join(dataRoot, "current", "release"))

	badChecksum := good
	badChecksum.Directory = t.TempDir()
	copyFixtureDirectory(t, good.Directory, badChecksum.Directory)
	replaceBinaryDigest(t, filepath.Join(badChecksum.Directory, "release.json"), "sha256:"+strings.Repeat("0", 64))
	if result := runInstaller(t, badChecksum, dataRoot, codexRoot, agentsRoot); result.err == nil || !strings.Contains(result.stderr, "Doctor") {
		t.Fatalf("checksum failure = %v\nstdout: %s\nstderr: %s", result.err, result.stdout, result.stderr)
	}
	assertCurrent(t, dataRoot, currentBefore)

	badDoctor := makeReleaseFixture(t, repository, "1.2.4", "9.9.9")
	if result := runInstaller(t, badDoctor, dataRoot, codexRoot, agentsRoot); result.err == nil || !strings.Contains(result.stderr, "Doctor") {
		t.Fatalf("doctor failure = %v\nstdout: %s\nstderr: %s", result.err, result.stdout, result.stderr)
	}
	assertCurrent(t, dataRoot, currentBefore)
	assertInstalledRelease(t, dataRoot, codexRoot, agentsRoot, good)
	if _, err := os.Stat(filepath.Join(dataRoot, "releases", "1.2.4")); !os.IsNotExist(err) {
		t.Fatalf("failed release was retained: %v", err)
	}

	conflictData, conflictCodex, conflictAgents := t.TempDir(), t.TempDir(), t.TempDir()
	conflict := filepath.Join(conflictCodex, "fanloop-workflow")
	if err := os.WriteFile(conflict, []byte("owned by user\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if result := runInstaller(t, good, conflictData, conflictCodex, conflictAgents); result.err == nil || !strings.Contains(result.stderr, "refusing to replace") {
		t.Fatalf("name conflict = %v\nstdout: %s\nstderr: %s", result.err, result.stdout, result.stderr)
	}
	if got, _ := os.ReadFile(conflict); string(got) != "owned by user\n" {
		t.Fatal("installer overwrote the conflicting user file")
	}
	if _, err := os.Lstat(filepath.Join(conflictData, "current")); !os.IsNotExist(err) {
		t.Fatalf("conflicting install activated current: %v", err)
	}
}

func TestLocalInstallerRejectsUserOwnedSkillLinks(t *testing.T) {
	repository := repositoryRoot(t)
	for _, exposed := range release.ExposedSkills() {
		t.Run(exposed.Name, func(t *testing.T) {
			fixture := makeReleaseFixture(t, repository, "1.2.3", "1.2.3")
			dataRoot, codexRoot, agentsRoot, traeRoot, claudeRoot := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
			externalTarget := t.TempDir()
			marker := filepath.Join(externalTarget, "owned-by-user")
			if err := os.WriteFile(marker, []byte("preserve me\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			link := filepath.Join(codexRoot, exposed.Name)
			if err := os.Symlink(externalTarget, link); err != nil {
				t.Fatal(err)
			}

			result := runInstaller(t, fixture, dataRoot, codexRoot, agentsRoot, traeRoot, claudeRoot)
			if result.err == nil || !strings.Contains(result.stderr, "refusing to replace") {
				t.Fatalf("install with user-owned Skill link = %v\nstdout: %s\nstderr: %s", result.err, result.stdout, result.stderr)
			}
			if target, err := os.Readlink(link); err != nil || target != externalTarget {
				t.Fatalf("user-owned Skill link changed to %q: %v", target, err)
			}
			if content, err := os.ReadFile(marker); err != nil || string(content) != "preserve me\n" {
				t.Fatalf("user-owned Skill target changed: %v\n%s", err, content)
			}
			if _, err := os.Lstat(filepath.Join(dataRoot, "current")); !os.IsNotExist(err) {
				t.Fatalf("rejected install activated current: %v", err)
			}
		})
	}
}

func TestDoctorChecksExposedWorkflowSkillLinks(t *testing.T) {
	repository := repositoryRoot(t)
	fixture := makeReleaseFixture(t, repository, "1.2.3", "1.2.3")
	dataRoot, codexRoot, agentsRoot := t.TempDir(), t.TempDir(), t.TempDir()
	traeRoot := filepath.Join(agentsRoot, ".trae-skills")
	claudeRoot := filepath.Join(agentsRoot, ".claude-skills")
	if result := runInstaller(t, fixture, dataRoot, codexRoot, agentsRoot); result.err != nil {
		t.Fatalf("install: %v\n%s", result.err, result.stderr)
	}
	if healthy := runCurrent(dataRoot, codexRoot, agentsRoot, "doctor"); healthy.err != nil || !strings.Contains(healthy.stdout, `"status": "healthy"`) {
		t.Fatalf("healthy install failed Doctor: %#v", healthy)
	}
	pinned := filepath.Join(codexRoot, "fanloop-workflow")
	if err := os.Remove(pinned); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dataRoot, "releases", fixture.Version, "entrypoints", "fanloop-workflow"), pinned); err != nil {
		t.Fatal(err)
	}
	if diagnosed := runCurrent(dataRoot, codexRoot, agentsRoot, "doctor"); diagnosed.err == nil || !strings.Contains(diagnosed.stdout, `"id": "skill_links"`) || !strings.Contains(diagnosed.stdout, `"status": "failed"`) {
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
		broken := runCurrent(dataRoot, codexRoot, agentsRoot, "doctor")
		if broken.err == nil || !strings.Contains(broken.stdout, `"id": "skill_links"`) || !strings.Contains(broken.stdout, `"status": "failed"`) {
			t.Fatalf("Doctor missed broken %s Skill link: %#v", client, broken)
		}
		if err := os.Symlink(filepath.Join(dataRoot, "current", "entrypoints", "fanloop-workflow"), link); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDoctorRejectsBrokenLiveSkillConfiguration(t *testing.T) {
	repository := copyRepositorySource(t, repositoryRoot(t))
	fixture := makeReleaseFixture(t, repository, "1.2.3", "1.2.3")
	fixture.ConfigSource = makeSyntheticConfigSource(t, repository, "sample-custom")
	dataRoot, codexRoot, agentsRoot := t.TempDir(), t.TempDir(), t.TempDir()
	if result := runInstaller(t, fixture, dataRoot, codexRoot, agentsRoot); result.err != nil {
		t.Fatalf("install: %v\n%s", result.err, result.stderr)
	}
	path := firstSkillPath(t, filepath.Join(dataRoot, "current", "workflows", "sample-custom", "skills"))
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	diagnosed := runCurrent(dataRoot, codexRoot, agentsRoot, "doctor")
	if diagnosed.err == nil || !strings.Contains(diagnosed.stdout, `"id": "workflows"`) || !strings.Contains(diagnosed.stdout, `"status": "failed"`) {
		t.Fatalf("Doctor accepted broken live Skill configuration: %#v", diagnosed)
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
	if result := runInstaller(t, fixture, dataRoot, codexRoot, agentsRoot); result.err != nil {
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

func runInstaller(t *testing.T, fixture releaseFixture, dataRoot, codexRoot, agentsRoot string, additionalRoots ...string) installResult {
	t.Helper()
	traeRoot := filepath.Join(agentsRoot, ".trae-skills")
	claudeRoot := filepath.Join(agentsRoot, ".claude-skills")
	if len(additionalRoots) > 0 {
		traeRoot = additionalRoots[0]
	}
	if len(additionalRoots) > 1 {
		claudeRoot = additionalRoots[1]
	}
	command := exec.Command(filepath.Join(fixture.Directory, "bin", "fanloop"),
		"__install", "--source", fixture.Directory, "--data-root", dataRoot,
		"--config-source", fixture.ConfigSource,
		"--codex-skills-root", codexRoot, "--agent-skills-root", agentsRoot,
		"--trae-skills-root", traeRoot, "--claude-skills-root", claudeRoot, "--replace-invalid",
	)
	command.Env = append(withoutBotmuxBinding(os.Environ()),
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
	for _, exposed := range release.ExposedSkills() {
		want := filepath.Join(dataRoot, "current", filepath.FromSlash(exposed.Path))
		path := filepath.Join(skillsRoot, exposed.Name)
		target, err := os.Readlink(path)
		if err != nil || target != want {
			t.Fatalf("skill link %s -> %q (%v), want %q", path, target, err, want)
		}
	}
}

func assertFlowSkillPaths(t *testing.T, output, configRoot string) {
	t.Helper()
	skillPaths := map[string]string{}
	matches, err := filepath.Glob(filepath.Join(configRoot, "*", "skills", "*", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range matches {
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			t.Fatal(err)
		}
		skillPaths[filepath.Base(filepath.Dir(path))] = resolved
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
					resolvedPath, err := filepath.EvalSymlinks(path)
					if err != nil || resolvedPath != want {
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

func assertInstalledRelease(t *testing.T, dataRoot, codexRoot, agentsRoot string, fixture releaseFixture, additionalRoots ...string) {
	t.Helper()
	version := fixture.Version
	wantCurrent := filepath.Join("..", "releases", version)
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
	liveRoot := filepath.Join(dataRoot, "current", "workflows")
	info, err := os.Lstat(liveRoot)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("live Workflow root is not a real directory: %#v, %v", info, err)
	}
	if fixtureDirectoryDigest(t, liveRoot) != fixtureDirectoryDigest(t, filepath.Join(fixture.ConfigSource, "workflows")) {
		t.Fatal("first install did not seed the source Workflow tree byte-for-byte")
	}
	if _, err := os.Lstat(filepath.Join(dataRoot, "releases", version, "workflows")); !os.IsNotExist(err) {
		t.Fatalf("immutable release contains live Workflows: %v", err)
	}
	for _, skill := range manifest.Skills {
		if _, err := os.Stat(filepath.Join(dataRoot, "releases", version, filepath.FromSlash(skill.Path), "SKILL.md")); err != nil {
			t.Fatalf("packaged Skill %s: %v", skill.Name, err)
		}
		if release.ValidSkillPath(skill.Path, skill.Name) {
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
	current := filepath.Join(dataRoot, "current")
	info, err := os.Lstat(current)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("current is not a stable directory: %#v (%v)", info, err)
	}
	got, err := os.Readlink(filepath.Join(current, "release"))
	if err != nil || got != want {
		t.Fatalf("current/release -> %q (%v), want %q", got, err, want)
	}
	for name, target := range map[string]string{"bin": "release/bin", "entrypoints": "release/entrypoints", "release.json": "release/release.json"} {
		got, err := os.Readlink(filepath.Join(current, name))
		if err != nil || got != target {
			t.Fatalf("current/%s -> %q (%v), want %q", name, got, err, target)
		}
	}
}

func makeReleaseFixture(t *testing.T, repository, releaseVersion, compiledVersion string) releaseFixture {
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

	return writeReleaseFixture(t, repository, staging, binary, releaseVersion)
}

func makeSyntheticConfigSource(t *testing.T, repository, workflowID string) string {
	t.Helper()
	root := t.TempDir()
	bundle := filepath.Join(root, "workflows", workflowID)
	if err := os.CopyFS(bundle, os.DirFS(filepath.Join(repository, "examples", "workflows", "example"))); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(bundle, "workflow.yaml")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content = bytes.Replace(content, []byte("id: example"), []byte("id: "+workflowID), 1)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func makeLinearRuntimeConfigSource(t *testing.T, repository, workflowID string) string {
	t.Helper()
	root := t.TempDir()
	bundle := filepath.Join(root, "workflows", workflowID)
	if err := os.CopyFS(bundle, os.DirFS(filepath.Join(repository, "internal", "workflow", "testdata", "linear-runtime"))); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(bundle, "workflow.yaml")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content = bytes.Replace(content, []byte("id: linear-runtime"), []byte("id: "+workflowID), 1)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func firstSkillPath(t *testing.T, root string) string {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(root, entry.Name(), "SKILL.md")
		}
	}
	t.Fatal("synthetic Workflow has no Skill")
	return ""
}

func makeReleaseFixtureWithChangedSyntheticPrompt(t *testing.T, repository, releaseVersion, compiledVersion string) releaseFixture {
	t.Helper()
	fixture := makeReleaseFixture(t, repository, releaseVersion, compiledVersion)
	fixture.ConfigSource = makeSyntheticControllerConfigSource(t, repository, "sample-custom")
	path := filepath.Join(fixture.ConfigSource, "workflows", "sample-custom", "prompt.yaml")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(content, []byte("读取用户材料"), []byte("candidate prompt"), 1)
	if bytes.Equal(changed, content) {
		t.Fatal("synthetic prompt fixture replacement did not match")
	}
	if err := os.WriteFile(path, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func makeSyntheticControllerConfigSource(t *testing.T, repository, workflowID string) string {
	t.Helper()
	root := makeSyntheticConfigSource(t, repository, workflowID)
	target := filepath.Join(root, "workflows", workflowID, "skills", "controller-tools", "scripts", "pin-controller-release.sh")
	copyTreeFile(t, filepath.Join(repository, "tests", "e2e", "testdata", "pin-controller-release.sh"), target)
	if err := os.WriteFile(filepath.Join(filepath.Dir(filepath.Dir(target)), "SKILL.md"), []byte("---\nname: controller-tools\ndescription: Synthetic controller pinning fixture.\n---\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func writeReleaseFixture(t *testing.T, repository, staging, binary, releaseVersion string) releaseFixture {
	t.Helper()
	skillItems := []map[string]any{}
	for _, exposed := range release.ExposedSkills() {
		skillSource := filepath.Join(repository, filepath.FromSlash(exposed.Path))
		if exposed.Name == release.AuthoringSkillName {
			skillSource = filepath.Join(repository, ".agents", "skills", exposed.Name)
		}
		skill := filepath.Join(staging, filepath.FromSlash(exposed.Path))
		copyFixtureDirectory(t, skillSource, skill)
		skillItems = append(skillItems, map[string]any{
			"name": exposed.Name, "version": releaseVersion, "path": exposed.Path, "sha256": fixtureDirectoryDigest(t, skill),
		})
	}
	manifest := map[string]any{
		"schema_version": 4, "release_version": releaseVersion,
		"cli": map[string]any{"version": releaseVersion, "binary_sha256": fixtureFileDigest(t, binary)},
		"state_schema": map[string]any{
			"read_versions": []int{state.CurrentStateSchemaVersion},
			"write_version": state.CurrentStateSchemaVersion,
		},
		"skills": skillItems,
	}
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(staging, "release.json")
	if err := os.WriteFile(manifestPath, append(content, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return releaseFixture{Directory: staging, ConfigSource: makeSyntheticConfigSource(t, repository, "sample-custom"), Version: releaseVersion}
}

func runBoundController(controllerHome string, args ...string) cliResult {
	skillRoot := filepath.Join(controllerHome, "skill-roots")
	command := exec.Command(filepath.Join(controllerHome, "current", "bin", "fanloop"), args...)
	command.Env = append(withoutBotmuxBinding(os.Environ()),
		"FANLOOP_DATA_HOME="+controllerHome,
		"FANLOOP_CODEX_SKILLS_ROOT="+filepath.Join(skillRoot, "codex"),
		"FANLOOP_AGENT_SKILLS_ROOT="+filepath.Join(skillRoot, "agent"),
		"FANLOOP_TRAE_SKILLS_ROOT="+filepath.Join(skillRoot, "trae"),
		"FANLOOP_CLAUDE_SKILLS_ROOT="+filepath.Join(skillRoot, "claude"),
	)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	exitCode := 0
	err := command.Run()
	if exit, ok := err.(*exec.ExitError); ok {
		exitCode = exit.ExitCode()
	} else if err != nil {
		exitCode = -1
	}
	return cliResult{stdout: stdout.String(), stderr: stderr.String(), exitCode: exitCode, err: err}
}

func replaceBinaryDigest(t *testing.T, source, digest string) {
	t.Helper()
	content, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(content, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest["cli"].(map[string]any)["binary_sha256"] = digest
	updated, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, append(updated, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func copyTreeFile(t *testing.T, source, target string) {
	t.Helper()
	info, err := os.Lstat(source)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		link, err := os.Readlink(source)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(link, target); err != nil {
			t.Fatal(err)
		}
		return
	}
	content, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, content, info.Mode().Perm()); err != nil {
		t.Fatal(err)
	}
}

func copyRepositorySource(t *testing.T, repository string) string {
	t.Helper()
	command := exec.Command("git", "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	command.Dir = repository
	output, err := command.Output()
	if err != nil {
		t.Fatalf("list tracked repository files: %v", err)
	}
	target := t.TempDir()
	for _, raw := range bytes.Split(output, []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		relative := string(raw)
		if _, err := os.Stat(filepath.Join(repository, relative)); os.IsNotExist(err) {
			continue
		}
		copyTreeFile(t, filepath.Join(repository, relative), filepath.Join(target, relative))
	}
	return target
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

func copyFixtureDirectory(t *testing.T, source, target string) {
	t.Helper()
	if err := filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		copyTreeFile(t, path, filepath.Join(target, relative))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
