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

	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

type releaseFixture struct {
	Directory string
	Version   string
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
	assertInstalledRelease(t, dataRoot, codexRoot, agentsRoot, fixture.Version, traeRoot, claudeRoot)
	assertSkillLink(t, dataRoot, traeRoot)
	assertSkillLink(t, dataRoot, claudeRoot)
	currentBefore, _ := os.Readlink(filepath.Join(dataRoot, "current"))

	second := runInstaller(t, fixture, dataRoot, codexRoot, agentsRoot, traeRoot, claudeRoot)
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
	repaired := runInstaller(t, fixture, dataRoot, codexRoot, agentsRoot, traeRoot, claudeRoot)
	if repaired.err != nil {
		t.Fatalf("repair install: %v\nstdout: %s\nstderr: %s", repaired.err, repaired.stdout, repaired.stderr)
	}
	if _, err := os.Stat(runtimeCache); !os.IsNotExist(err) {
		t.Fatalf("repair install retained runtime cache: %v", err)
	}
	assertInstalledRelease(t, dataRoot, codexRoot, agentsRoot, fixture.Version, traeRoot, claudeRoot)

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
		{"card", "render", "--help"}, {"version", "--help"}, {"doctor", "--help"},
	} {
		result := runCurrent(dataRoot, codexRoot, agentsRoot, args...)
		if result.err != nil || result.stderr != "" || !strings.Contains(result.stdout, "Request JSON:") {
			t.Fatalf("installed fanloop %s: %v\nstdout: %s\nstderr: %s", strings.Join(args, " "), result.err, result.stdout, result.stderr)
		}
	}
}

func TestPinnedControllerKeepsRequirementOnInitializingReleaseWhenCurrentChanges(t *testing.T) {
	repository := repositoryRoot(t)
	home := t.TempDir()
	dataRoot := filepath.Join(home, ".fanloop")
	codexRoot := filepath.Join(home, ".codex", "skills")
	agentsRoot := filepath.Join(home, ".agents", "skills")
	oldRelease := makeReleaseFixture(t, repository, "1.2.3", "1.2.3")
	if result := runInstaller(t, oldRelease, dataRoot, codexRoot, agentsRoot); result.err != nil {
		t.Fatalf("install initializing release: %v\nstdout: %s\nstderr: %s", result.err, result.stdout, result.stderr)
	}

	oldRoot := filepath.Join(home, "fanloop", "issues", "old-requirement")
	if err := os.MkdirAll(oldRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	initialized := runCurrent(dataRoot, codexRoot, agentsRoot, "flow", "init", "--root", oldRoot, "--workflow", "fanloop-maintainer", "--title", "Pinned controller E2E")
	if initialized.err != nil {
		t.Fatalf("initialize old Requirement: %v\nstdout: %s\nstderr: %s", initialized.err, initialized.stdout, initialized.stderr)
	}

	pinner := filepath.Join(dataRoot, "current", "skills", "fanloop-maintainer", "fanloop-dev-update-local-cli", "scripts", "pin-controller-release.sh")
	pinned := exec.Command(pinner, oldRoot)
	pinned.Env = append(os.Environ(), "HOME="+home)
	if output, err := pinned.CombinedOutput(); err != nil {
		t.Fatalf("pin initializing controller: %v\n%s", err, output)
	}

	newRelease := makeReleaseFixtureWithChangedMaintainerPrompt(t, repository, "1.2.4", "1.2.4")
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
	globalOldStatus := runCurrent(dataRoot, codexRoot, agentsRoot, "flow", "status", "--root", oldRoot)
	if globalOldStatus.err == nil || !strings.Contains(globalOldStatus.stderr, "WORKFLOW_MISMATCH") {
		t.Fatalf("candidate current unexpectedly controlled old Requirement: %v\nstdout: %s\nstderr: %s", globalOldStatus.err, globalOldStatus.stdout, globalOldStatus.stderr)
	}

	controllerHome := filepath.Join(oldRoot, "bound-release-home")
	controllerStatus := runBoundController(controllerHome, "flow", "status", "--root", oldRoot)
	if controllerStatus.err != nil {
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
	newInitialized := runCurrent(dataRoot, codexRoot, agentsRoot, "flow", "init", "--root", newRoot, "--workflow", "fanloop-maintainer", "--title", "Candidate current E2E")
	if newInitialized.err != nil {
		t.Fatalf("candidate current did not initialize new Requirement: %v\nstdout: %s\nstderr: %s", newInitialized.err, newInitialized.stdout, newInitialized.stderr)
	}
	candidateVersion := runCurrent(dataRoot, codexRoot, agentsRoot, "version")
	if candidateVersion.err != nil || !strings.Contains(candidateVersion.stdout, `"release_version": "1.2.4"`) {
		t.Fatalf("candidate current version: %v\nstdout: %s\nstderr: %s", candidateVersion.err, candidateVersion.stdout, candidateVersion.stderr)
	}
}

func TestLocalInstallerExposesOnlyWorkflowSkillAndPreservesAtomicSkillDirectories(t *testing.T) {
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

	result := runInstaller(t, fixture, dataRoot, codexRoot, agentsRoot, traeRoot, claudeRoot)
	if result.err != nil {
		t.Fatalf("install with atomic Skill directories: %v\nstdout: %s\nstderr: %s", result.err, result.stdout, result.stderr)
	}
	for _, skillID := range []string{
		"technical-background-framing", "technical-problem-analysis", "technical-objective-setting",
		"technical-problem-approval", "technical-solution-research", "technical-overall-solution",
		"technical-key-solutions", "technical-direction-approval", "technical-solution-benefits",
		"technical-solution-delivery", "technical-solution-writing", "technical-solution-review",
		"technical-solution-approval",
	} {
		path := filepath.Join(dataRoot, "releases", fixture.Version, "skills", "technical-solution-design", skillID, "SKILL.md")
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("packaged %s Skill: %v", skillID, err)
		}
	}
	for _, skillID := range []string{
		"flashcard", "flashcard-card-planning", "flashcard-goal-framing", "flashcard-knowledge-selection",
		"flashcard-preview-approval", "flashcard-quality-review", "flashcard-source-understanding",
		"material-flashcards-panorama",
	} {
		path := filepath.Join(dataRoot, "releases", fixture.Version, "skills", "material-flashcards", skillID, "SKILL.md")
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("packaged %s Skill: %v", skillID, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dataRoot, "releases", fixture.Version, "skills", "material-flashcards", "flashcard", "references", "term-concept-card.md")); err != nil {
		t.Fatalf("packaged flashcard concept-card reference: %v", err)
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
	initialized := runCurrent(dataRoot, codexRoot, agentsRoot, "flow", "init", "--root", requirementRoot, "--workflow", "technical-solution-design", "--title", "Technical solution Skill path E2E")
	if initialized.err != nil {
		t.Fatalf("initialize installed release: %v\nstdout: %s\nstderr: %s", initialized.err, initialized.stdout, initialized.stderr)
	}
	assertFlowSkillPaths(t, initialized.stdout, filepath.Join(dataRoot, "releases", fixture.Version))

	flashcardRoot := t.TempDir()
	flashcards := runCurrent(dataRoot, codexRoot, agentsRoot, "flow", "init", "--root", flashcardRoot, "--workflow", "material-flashcards", "--title", "Material flashcards Skill path E2E")
	if flashcards.err != nil {
		t.Fatalf("initialize installed material-flashcards release: %v\nstdout: %s\nstderr: %s", flashcards.err, flashcards.stdout, flashcards.stderr)
	}
	assertFlowSkillPaths(t, flashcards.stdout, filepath.Join(dataRoot, "releases", fixture.Version))
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
			if result.err == nil || !strings.Contains(result.stderr, "refusing to replace") {
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
	currentBefore, _ := os.Readlink(filepath.Join(dataRoot, "current"))

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
	assertInstalledRelease(t, dataRoot, codexRoot, agentsRoot, good.Version)
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

func TestLocalInstallerAdoptsExternalSkillLinksWithoutDeletingTheirTargets(t *testing.T) {
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

	result := runInstaller(t, fixture, dataRoot, codexRoot, agentsRoot, traeRoot, claudeRoot)
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

func makeReleaseFixtureWithChangedMaintainerPrompt(t *testing.T, repository, releaseVersion, compiledVersion string) releaseFixture {
	t.Helper()
	candidateRepository := copyRepositorySource(t, repository)
	path := filepath.Join(candidateRepository, "workflows", "fanloop-maintainer", "prompt.yaml")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(content, []byte("main 固定基线"), []byte("candidate 固定基线"), 1)
	if bytes.Equal(changed, content) {
		t.Fatal("maintainer prompt fixture replacement did not match")
	}
	if err := os.WriteFile(path, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	return makeReleaseFixture(t, candidateRepository, releaseVersion, compiledVersion)
}

func writeReleaseFixture(t *testing.T, repository, staging, binary, releaseVersion string) releaseFixture {
	t.Helper()
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
		copyFixtureDirectory(t, skillSource, skill)
		skillItems = append(skillItems, map[string]any{
			"name": name, "version": releaseVersion, "path": relative, "sha256": fixtureDirectoryDigest(t, skill),
		})
	}
	workflowItems := []map[string]any{}
	workflowPaths, err := filepath.Glob(filepath.Join(repository, "workflows", "*", "workflow.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range workflowPaths {
		sourceRoot := filepath.Dir(source)
		relative, err := filepath.Rel(repository, sourceRoot)
		if err != nil {
			t.Fatal(err)
		}
		relative = filepath.ToSlash(relative)
		targetRoot := filepath.Join(staging, filepath.FromSlash(relative))
		for _, name := range workflow.BundleFileNames() {
			sourcePath := filepath.Join(sourceRoot, name)
			targetPath := filepath.Join(targetRoot, name)
			copyTreeFile(t, sourcePath, targetPath)
		}
		loaded, decodeErr := workflow.LoadDirectory(targetRoot)
		if decodeErr != nil {
			t.Fatalf("read workflow Bundle %s: %v", targetRoot, decodeErr)
		}
		workflowItems = append(workflowItems, map[string]any{
			"id": loaded.Ref.ID, "path": relative, "sha256": loaded.Ref.Digest,
		})
	}

	manifest := map[string]any{
		"schema_version": 3, "release_version": releaseVersion,
		"cli": map[string]any{"version": releaseVersion, "binary_sha256": fixtureFileDigest(t, binary)},
		"state_schema": map[string]any{
			"read_versions": []int{state.CurrentStateSchemaVersion},
			"write_version": state.CurrentStateSchemaVersion,
		},
		"skills":    skillItems,
		"workflows": workflowItems,
	}
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(staging, "release.json")
	if err := os.WriteFile(manifestPath, append(content, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return releaseFixture{Directory: staging, Version: releaseVersion}
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
	content, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(source)
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
