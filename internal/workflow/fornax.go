package workflow

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// FornaxSource selects published configuration, never credentials or YAML content.
type FornaxSource struct {
	WorkspaceID string           `json:"workspace_id"`
	Workflows   []FornaxWorkflow `json:"workflows"`
}
type FornaxWorkflow struct {
	ID       string `json:"id"`
	SkillKey string `json:"skill_key"`
	SkillID  string `json:"skill_id"`
}

var numericID = regexp.MustCompile(`^[1-9][0-9]*$`)
var skillKey = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

const maxPackageBytes = 128 << 20
const maxExpandedBytes = 256 << 20

// LockConfiguration coordinates whole-command readers, installers and Dashboard requests.
// Closing releases the flock even when a command fails or is killed.
func LockConfiguration(dataRoot string, exclusive bool) (func(), error) {
	if !filepath.IsAbs(dataRoot) {
		return nil, fmt.Errorf("configuration data root must be absolute")
	}
	if err := os.MkdirAll(dataRoot, 0700); err != nil {
		return nil, err
	}
	fd, err := unix.Open(filepath.Join(dataRoot, ".workflows.lock"), unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), "workflow lock")
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		file.Close()
		return nil, fmt.Errorf("configuration lock must be a regular file")
	}
	mode := unix.LOCK_SH
	if exclusive {
		mode = unix.LOCK_EX
	}
	if err := unix.Flock(fd, mode); err != nil {
		file.Close()
		return nil, err
	}
	return func() { _ = file.Close() }, nil
}

// PrepareConfiguration refreshes once and retains the lock for the command's reads.
// FANLOOP_CONFIG_SOURCE is an explicit isolated verification source, not a fallback.
func PrepareConfiguration(ctx context.Context, dataRoot string, selection ...string) (func(), error) {
	unlock, err := LockConfiguration(dataRoot, true)
	if err != nil {
		return nil, err
	}
	if source := os.Getenv("FANLOOP_CONFIG_SOURCE"); source != "" {
		if err := ValidateIsolatedSource(dataRoot, source); err != nil {
			unlock()
			return nil, err
		}
		return unlock, nil
	}
	if err := SyncFornax(ctx, dataRoot, selection...); err != nil {
		unlock()
		return nil, err
	}
	return unlock, nil
}

// SyncFornax is called while holding the configuration lock. No previously downloaded
// tree is used to turn a failed remote read into a successful command.
func SyncFornax(ctx context.Context, dataRoot string, selection ...string) error {
	sourcePath := filepath.Join(dataRoot, "fornax.json")
	info, err := os.Lstat(sourcePath)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 1<<20 {
		return fmt.Errorf("configure a regular %s with workspace_id and workflows before starting Fanloop", sourcePath)
	}
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	var source FornaxSource
	if err := rejectDuplicateKeys(content); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&source); err != nil {
		return fmt.Errorf("fornax.json: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("fornax.json must contain exactly one object")
	}
	if !numericID.MatchString(source.WorkspaceID) || len(source.Workflows) == 0 {
		return fmt.Errorf("fornax.json requires workspace_id and at least one workflow")
	}
	ids, keys, skills := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, w := range source.Workflows {
		if !workflowIDPattern.MatchString(w.ID) || !skillKey.MatchString(w.SkillKey) || !numericID.MatchString(w.SkillID) || ids[w.ID] || keys[w.SkillKey] || skills[w.SkillID] {
			return fmt.Errorf("invalid or duplicate Fornax workflow selector")
		}
		ids[w.ID], keys[w.SkillKey], skills[w.SkillID] = true, true, true
	}

	for _, selected := range selection {
		if selected == "" {
			continue
		}
		if !ids[selected] {
			return fmt.Errorf("Workflow %q is not selected in fornax.json", selected)
		}
	}
	current := filepath.Join(dataRoot, "current")
	if err := os.MkdirAll(current, 0700); err != nil {
		return err
	}
	if err := requireRealDirectory(current); err != nil {
		return err
	}
	staged, err := os.MkdirTemp(current, ".fornax-")
	if err != nil {
		return err
	}
	cleanupStaged := true
	defer func() {
		if cleanupStaged {
			_ = os.RemoveAll(staged)
		}
	}()
	tree := filepath.Join(staged, "workflows")
	if err := os.Mkdir(tree, 0700); err != nil {
		return err
	}
	for _, w := range source.Workflows {
		version, err := releasedVersion(ctx, staged, source.WorkspaceID, w)
		if err != nil {
			return fmt.Errorf("Fornax %s: %w", w.ID, err)
		}
		archiveURL, err := publishedZIP(ctx, staged, source.WorkspaceID, w, version)
		if err != nil {
			return fmt.Errorf("Fornax %s: %w", w.ID, err)
		}
		archive, err := downloadZIP(ctx, archiveURL)
		if err != nil {
			return fmt.Errorf("Fornax %s: %w", w.ID, err)
		}
		extracted := filepath.Join(staged, "package-"+w.ID)
		if err := extractPackage(archive, extracted, w.ID); err != nil {
			return fmt.Errorf("Fornax %s: %w", w.ID, err)
		}
		if err := os.Rename(filepath.Join(extracted, "workflows", w.ID), filepath.Join(tree, w.ID)); err != nil {
			return err
		}
	}
	if _, err := ListFrom(tree); err != nil {
		return fmt.Errorf("downloaded Workflows: %w", err)
	}

	live := filepath.Join(current, "workflows")
	backup := filepath.Join(staged, "previous")
	existed := false
	if info, err := os.Lstat(live); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("current/workflows must be a real directory")
		}
		if err := os.Rename(live, backup); err != nil {
			return err
		}
		existed = true
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(tree, live); err != nil {
		if existed {
			if rollback := os.Rename(backup, live); rollback != nil {
				cleanupStaged = false
				return fmt.Errorf("activate configuration: %v; restore previous: %w", err, rollback)
			}
		}
		return err
	}
	return nil
}

func fornaxJSON(ctx context.Context, temporary, workspace string, destination any, args ...string) error {
	directory, err := os.MkdirTemp(temporary, "query-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	args = append(args, "--workspace-id", workspace, "--format", "json", "-o", directory)
	command := exec.CommandContext(ctx, "fornax-cli", args...)
	// stdout may mix progress with JSON; -o is the documented machine result path.
	if err := command.Run(); err != nil {
		return fmt.Errorf("fornax-cli %s failed (check Fornax login and workspace access): %w", strings.Join(args[:2], " "), err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}
	if len(entries) != 1 || entries[0].IsDir() || !strings.HasSuffix(entries[0].Name(), ".json") {
		return fmt.Errorf("fornax-cli did not write exactly one JSON result")
	}
	result := filepath.Join(directory, entries[0].Name())
	info, err := os.Lstat(result)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 4<<20 {
		return fmt.Errorf("invalid Fornax JSON result")
	}
	content, err := os.ReadFile(result)
	if err != nil {
		return err
	}
	var status struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(content, &status); err != nil || status.Code != 0 {
		return fmt.Errorf("Fornax query returned invalid or failed response")
	}
	if err := json.Unmarshal(content, destination); err != nil {
		return fmt.Errorf("decode Fornax result: %w", err)
	}
	return nil
}

func releasedVersion(ctx context.Context, temporary, workspace string, w FornaxWorkflow) (string, error) {
	var response struct {
		Items []struct {
			SkillID string `json:"skill_id"`
			Env     string `json:"env"`
			Feature string `json:"feature"`
			Status  int    `json:"release_status"`
			Version string `json:"version"`
			Updated string `json:"updated_at"`
		} `json:"items"`
	}
	if err := fornaxJSON(ctx, temporary, workspace, &response, "skill", "list-releases", "--skill-id", w.SkillID, "--env", "Online", "--feature", "prod"); err != nil {
		return "", err
	}
	version := ""

	for _, item := range response.Items {
		if item.SkillID != w.SkillID || item.Env != "Online" || item.Feature != "prod" || item.Status != 1 {
			continue
		}
		timestamp, err := strconv.ParseInt(item.Updated, 10, 64)
		if err != nil || timestamp <= 0 || item.Version == "" {
			return "", fmt.Errorf("invalid release record")
		}
		if version != "" && version != item.Version {
			return "", fmt.Errorf("ambiguous published versions")
		}
		version = item.Version
	}
	if version == "" {
		return "", fmt.Errorf("no successful Online/prod release")
	}
	return version, nil
}
func publishedZIP(ctx context.Context, temporary, workspace string, w FornaxWorkflow, version string) (string, error) {
	var response struct {
		Items []struct {
			Skill struct {
				ID        string `json:"id"`
				Key       string `json:"skill_key"`
				Workspace string `json:"workspace_id"`
				Commit    struct {
					Info struct {
						Version string `json:"version"`
					} `json:"commit_info"`
					Detail struct {
						URL string `json:"tos_url"`
					} `json:"skill_detail"`
				} `json:"skill_commit"`
			} `json:"skill"`
		} `json:"items"`
	}
	if err := fornaxJSON(ctx, temporary, workspace, &response, "skill", "batch-get-by-key", "--skill-key", w.SkillKey, "--version", version); err != nil {
		return "", err
	}
	if len(response.Items) != 1 {
		return "", fmt.Errorf("published Skill lookup must return one item")
	}
	s := response.Items[0].Skill
	if s.ID != w.SkillID || s.Key != w.SkillKey || s.Workspace != workspace || s.Commit.Info.Version != version {
		return "", fmt.Errorf("published Skill identity does not match selection")
	}
	return s.Commit.Detail.URL, nil
}
func downloadZIP(ctx context.Context, address string) ([]byte, error) {
	parsed, err := url.Parse(address)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil {
		return nil, fmt.Errorf("package URL must be HTTPS without credentials")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid package URL")
	}
	client := http.Client{Timeout: 60 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("package download failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("package download returned HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxPackageBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read package download failed")
	}
	if len(body) > maxPackageBytes {
		return nil, fmt.Errorf("package exceeds size limit")
	}
	return body, nil
}

func extractPackage(content []byte, destination, workflowID string) error {
	archive, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return fmt.Errorf("invalid ZIP: %w", err)
	}
	if len(archive.File) == 0 || len(archive.File) > 10000 {
		return fmt.Errorf("invalid ZIP entry count")
	}
	names := map[string]bool{}
	var expanded uint64
	wrapper := ""
	// Validate every entry before writing any bytes. One optional platform wrapper is allowed.
	for _, file := range archive.File {
		name := strings.TrimSuffix(file.Name, "/")
		if name == "" || strings.Contains(name, `\`) || path.Clean(name) != name || !filepath.IsLocal(name) || strings.Contains(name, ":") || names[name] {
			return fmt.Errorf("unsafe or duplicate ZIP path")
		}
		names[name] = true
		if !file.Mode().IsRegular() && !file.Mode().IsDir() {
			return fmt.Errorf("ZIP contains a link or special file")
		}
		expanded += file.UncompressedSize64
		if expanded > maxExpandedBytes || file.UncompressedSize64 > maxPackageBytes {
			return fmt.Errorf("expanded package exceeds size limit")
		}
	}
	if !names["SKILL.md"] {
		for name := range names {
			if strings.HasSuffix(name, "/SKILL.md") && strings.Count(name, "/") == 1 {
				if wrapper != "" {
					return fmt.Errorf("ambiguous package wrapper")
				}
				wrapper = strings.Split(name, "/")[0] + "/"
			}
		}
		if wrapper == "" {
			return fmt.Errorf("package has no root SKILL.md")
		}
	}
	required := map[string]bool{"SKILL.md": false}
	for _, file := range archive.File {
		name := strings.TrimSuffix(file.Name, "/")
		if wrapper != "" && name == strings.TrimSuffix(wrapper, "/") && file.FileInfo().IsDir() {
			continue
		}
		if !strings.HasPrefix(name, wrapper) {
			return fmt.Errorf("ZIP escapes its package wrapper")
		}
		name = strings.TrimPrefix(name, wrapper)
		if name != "SKILL.md" && name != "workflows" && name != "workflows/"+workflowID && !strings.HasPrefix(name, "workflows/"+workflowID+"/") {
			return fmt.Errorf("unexpected package entry %q", name)
		}

		if name == "SKILL.md" {
			if !file.Mode().IsRegular() || file.UncompressedSize64 == 0 {
				return fmt.Errorf("invalid package SKILL.md")
			}
			required[name] = true
		}
	}
	if !required["SKILL.md"] {
		return fmt.Errorf("missing package SKILL.md")
	}
	for _, file := range archive.File {
		name := strings.TrimSuffix(file.Name, "/")
		if wrapper != "" && name == strings.TrimSuffix(wrapper, "/") {
			continue
		}
		name = strings.TrimPrefix(name, wrapper)
		target := filepath.Join(destination, filepath.FromSlash(name))
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0700); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		input, err := file.Open()
		if err != nil {
			return err
		}
		mode := os.FileMode(0600)
		if file.Mode().Perm()&0111 != 0 {
			mode = 0700
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
		if err != nil {
			input.Close()
			return err
		}
		n, copyErr := io.Copy(output, io.LimitReader(input, int64(file.UncompressedSize64)+1))
		inErr, outErr := input.Close(), output.Close()
		if copyErr != nil {
			return copyErr
		}
		if inErr != nil {
			return inErr
		}
		if outErr != nil {
			return outErr
		}
		if uint64(n) != file.UncompressedSize64 {
			return fmt.Errorf("ZIP size mismatch")
		}
	}
	_, err = LoadFrom(filepath.Join(destination, "workflows"), workflowID)
	return err
}

// ValidateIsolatedSource keeps local fixtures an explicit verification-only seam.
func ValidateIsolatedSource(dataRoot, source string) error {
	if !filepath.IsAbs(source) {
		return fmt.Errorf("FANLOOP_CONFIG_SOURCE must be absolute")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	absolute, err := filepath.EvalSymlinks(dataRoot)
	if err != nil {
		return err
	}
	global := filepath.Join(home, ".fanloop")
	if resolved, err := filepath.EvalSymlinks(global); err == nil {
		global = resolved
	}
	if absolute == global || isWithin(global, absolute) {
		return fmt.Errorf("local configuration sources are only allowed in isolated verification data homes")
	}
	if _, err := ListFrom(filepath.Join(source, "workflows")); err != nil {
		return fmt.Errorf("explicit verification source: %w", err)
	}
	return nil
}

func rejectDuplicateKeys(content []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(content))
	var value func() error
	value = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			keys := map[string]bool{}
			for decoder.More() {
				token, err := decoder.Token()
				if err != nil {
					return err
				}
				key, ok := token.(string)
				if !ok || keys[key] {
					return fmt.Errorf("duplicate JSON object key")
				}
				keys[key] = true
				if err := value(); err != nil {
					return err
				}
			}
		case '[':
			for decoder.More() {
				if err := value(); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("invalid JSON delimiter")
		}
		_, err = decoder.Token()
		return err
	}
	if err := value(); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("trailing JSON value")
	}
	return nil
}
