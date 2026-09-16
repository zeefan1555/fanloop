package verify

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/zeefan1555/fanloop/internal/idl/verifyidl"
)

const manifestSchemaVersion = 1

type manifest struct {
	SchemaVersion   int                            `json:"schema_version"`
	RunID           string                         `json:"run_id"`
	StartedAt       string                         `json:"started_at"`
	FinishedAt      string                         `json:"finished_at,omitempty"`
	Feature         string                         `json:"feature"`
	Variant         string                         `json:"variant"`
	EvidenceDir     string                         `json:"evidence_dir"`
	RequirementRoot string                         `json:"requirement_root"`
	Candidate       candidateIdentity              `json:"candidate"`
	Commands        []commandEvidence              `json:"commands"`
	Checks          []*verifyidl.VerificationCheck `json:"checks"`
	StatusChanges   []statusChange                 `json:"status_changes"`
	SideEffects     []sideEffect                   `json:"side_effects"`
	OwnedPaths      []string                       `json:"owned_paths"`
	Cleanup         cleanupEvidence                `json:"cleanup"`
	Outcome         string                         `json:"outcome"`
}

type candidateIdentity struct {
	ReleaseVersion string `json:"release_version"`
	CommitSHA      string `json:"commit_sha"`
	BinarySHA256   string `json:"binary_sha256"`
}

type commandEvidence struct {
	Sequence   int      `json:"sequence"`
	ID         string   `json:"id"`
	Argv       []string `json:"argv"`
	StartedAt  string   `json:"started_at"`
	DurationMS int64    `json:"duration_ms"`
	ExitCode   int      `json:"exit_code"`
	StdoutRef  string   `json:"stdout_ref"`
	StderrRef  string   `json:"stderr_ref"`
}

type statusChange struct {
	CommandRef string `json:"command_ref"`
	From       string `json:"from"`
	To         string `json:"to"`
}

type sideEffect struct {
	Name         string   `json:"name"`
	Expected     string   `json:"expected"`
	ChangedPaths []string `json:"changed_paths"`
	Passed       bool     `json:"passed"`
}

type cleanupEvidence struct {
	DryRunPassed bool     `json:"dry_run_passed"`
	Removed      bool     `json:"removed"`
	Paths        []string `json:"paths"`
}

type evidenceRun struct {
	dir      string
	manifest manifest
}

func newEvidenceRun(dataRoot, requested, runID, requirementRoot, workRoot string, now time.Time) (*evidenceRun, error) {
	directory := filepath.Join(dataRoot, "verification", "runs", runID)
	if requested != "" {
		if !filepath.IsAbs(requested) {
			return nil, fmt.Errorf("evidence_dir must be absolute")
		}
		directory = filepath.Clean(requested)
	}
	verificationWork := filepath.Join(dataRoot, "verification", "work")
	if containsPath(verificationWork, directory) {
		return nil, fmt.Errorf("evidence_dir must be outside verification work")
	}
	if err := os.MkdirAll(filepath.Dir(directory), 0o700); err != nil {
		return nil, err
	}
	if err := os.Mkdir(directory, 0o700); err != nil {
		return nil, fmt.Errorf("create evidence directory: %w", err)
	}
	run := &evidenceRun{
		dir: directory,
		manifest: manifest{
			SchemaVersion:   manifestSchemaVersion,
			RunID:           runID,
			StartedAt:       now.UTC().Format(time.RFC3339Nano),
			Feature:         "verification-smoke",
			Variant:         "local-public-cli",
			EvidenceDir:     directory,
			RequirementRoot: requirementRoot,
			OwnedPaths:      []string{workRoot},
			Outcome:         "blocked",
		},
	}
	if err := run.writeManifest(); err != nil {
		return nil, err
	}
	return run, nil
}

func (run *evidenceRun) recordCommand(result commandResult) (string, error) {
	sequence := len(run.manifest.Commands) + 1
	directory := filepath.Join(run.dir, "commands", fmt.Sprintf("%02d-%s", sequence, commandSlug(result.ID)))
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", err
	}
	argvRef := relativeRef(run.dir, filepath.Join(directory, "argv.json"))
	stdoutRef := relativeRef(run.dir, filepath.Join(directory, "stdout.txt"))
	stderrRef := relativeRef(run.dir, filepath.Join(directory, "stderr.txt"))
	if err := writeJSON(filepath.Join(run.dir, filepath.FromSlash(argvRef)), result.Argv); err != nil {
		return "", err
	}
	if err := writePrivate(filepath.Join(run.dir, filepath.FromSlash(stdoutRef)), result.Stdout); err != nil {
		return "", err
	}
	if err := writePrivate(filepath.Join(run.dir, filepath.FromSlash(stderrRef)), result.Stderr); err != nil {
		return "", err
	}
	run.manifest.Commands = append(run.manifest.Commands, commandEvidence{
		Sequence: sequence, ID: result.ID, Argv: result.Argv,
		StartedAt: result.StartedAt.UTC().Format(time.RFC3339Nano), DurationMS: result.Duration.Milliseconds(),
		ExitCode: result.ExitCode, StdoutRef: stdoutRef, StderrRef: stderrRef,
	})
	if err := run.writeManifest(); err != nil {
		return "", err
	}
	return fmt.Sprintf("commands/%02d-%s", sequence, commandSlug(result.ID)), nil
}

func (run *evidenceRun) writeManifest() error {
	return writeJSON(filepath.Join(run.dir, "manifest.json"), run.manifest)
}

func (run *evidenceRun) writeReport() error {
	var content strings.Builder
	fmt.Fprintf(&content, "# Fanloop Verification Report\n\n- Run: `%s`\n- Outcome: `%s`\n- Candidate: `%s@%s`\n- Binary: `%s`\n- Requirement: `%s`\n- Cleanup complete: `%t`\n\n## Checks\n\n",
		run.manifest.RunID, run.manifest.Outcome, run.manifest.Candidate.ReleaseVersion,
		run.manifest.Candidate.CommitSHA, run.manifest.Candidate.BinarySHA256,
		run.manifest.RequirementRoot, run.manifest.Cleanup.Removed)
	for _, check := range run.manifest.Checks {
		fmt.Fprintf(&content, "- %s: %s - %s\n", check.Id, check.Status.String(), check.Summary)
	}
	fmt.Fprintf(&content, "\n## Side Effects\n\n")
	for _, effect := range run.manifest.SideEffects {
		fmt.Fprintf(&content, "- %s: %t (%s)\n", effect.Name, effect.Passed, effect.Expected)
	}
	fmt.Fprintf(&content, "\n## Commands\n\n")
	for _, command := range run.manifest.Commands {
		fmt.Fprintf(&content, "- `%s` exit %d: `%s`\n", command.ID, command.ExitCode, command.StdoutRef)
	}
	return writePrivate(filepath.Join(run.dir, "report.md"), []byte(content.String()))
}

func writeJSON(path string, value any) error {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writePrivate(path, append(content, '\n'))
}

func writePrivate(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".write-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func relativeRef(root, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(relative)
}

func commandSlug(value string) string {
	value = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return '-'
	}, strings.ToLower(value))
	return strings.Trim(value, "-")
}

func containsPath(parent, child string) bool {
	relative, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(child))
	return err == nil && (relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func changedPaths(before, after map[string]string) []string {
	seen := map[string]bool{}
	for path := range before {
		seen[path] = true
	}
	for path := range after {
		seen[path] = true
	}
	paths := make([]string, 0, len(seen))
	for path := range seen {
		if before[path] != after[path] {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths
}
