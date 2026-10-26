package distribution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/zeefan1555/fanloop/internal/release/currentlayout"
)

// AutoUpdateTimeout bounds a fresh-init CLI refresh. A slow or failed check
// falls back to the binary that is already running.
const AutoUpdateTimeout = 15 * time.Second

// EnvNoAutoUpdate disables the fresh-init CLI refresh when set to 1, true, or yes.
const EnvNoAutoUpdate = "FANLOOP_NO_AUTO_UPDATE"

// envReexec is set only on the process image started after a successful upgrade
// so that image does not refresh again.
const envReexec = "FANLOOP_AUTO_UPDATE_REEXEC"

var (
	// ErrNoReleaseInstall means current is not a versioned GitHub Release symlink.
	// Development installs and missing installs are left untouched.
	ErrNoReleaseInstall = errors.New("fanloop current is not a versioned release")
	// ErrAutomaticUpdate means the fresh-init refresh failed and the caller should
	// continue with the installed binary.
	ErrAutomaticUpdate = errors.New("automatic update failed")
	// ErrProcessReplaced means the updated binary was executed. syscall.Exec does
	// not return on success; this value is returned when the injected executor does.
	ErrProcessReplaced = errors.New("fanloop re-executed the updated binary")
)

// AutoUpdate moves a versioned install to a newer GitHub Release.
// An install that is already the latest, or newer than latest, is left in place
// and does not download release assets. Explicit fanloop update still repairs
// a same-version tree.
func AutoUpdate(ctx context.Context, options Options) (Result, error) {
	if !filepath.IsAbs(options.DataRoot) {
		return Result{}, fmt.Errorf("data root must be absolute")
	}
	if _, managed, err := currentlayout.Inspect(options.DataRoot); err != nil || managed {
		return Result{}, ErrNoReleaseInstall
	}
	installed, ok, err := installedReleaseVersion(options.DataRoot)
	if err != nil || !ok {
		return Result{}, ErrNoReleaseInstall
	}
	ctx, cancel := context.WithTimeout(ctx, AutoUpdateTimeout)
	defer cancel()
	if options.Client == nil {
		options.Client = &http.Client{Timeout: AutoUpdateTimeout}
	}
	latest, err := latestVersion(ctx, options)
	if err != nil {
		return Result{}, err
	}
	if compareSemver(installed, latest) >= 0 {
		return Result{Version: installed, Unchanged: true}, nil
	}
	return Update(ctx, options)
}

func latestVersion(ctx context.Context, options Options) (string, error) {
	latestURL := options.LatestURL
	if latestURL == "" {
		latestURL = LatestReleaseURL
	}
	body, err := get(ctx, options.Client, latestURL)
	if err != nil {
		return "", err
	}
	var published githubRelease
	if err := json.Unmarshal(body, &published); err != nil {
		return "", fmt.Errorf("decode GitHub release: %w", err)
	}
	version := strings.TrimPrefix(published.TagName, "v")
	if published.TagName != "v"+version || !versionPattern.MatchString(version) {
		return "", fmt.Errorf("invalid GitHub release tag %q", published.TagName)
	}
	return version, nil
}

// RefreshCLI updates a versioned install during a fresh init and re-executes
// the new binary when the release changed. A disabled refresh, a development
// install, and an already-current release return nil so the caller continues.
func RefreshCLI(ctx context.Context, options Options, execFn func(argv0 string, argv []string, env []string) error) error {
	if envFlag(EnvNoAutoUpdate) || os.Getenv(envReexec) == "1" {
		return nil
	}
	result, err := AutoUpdate(ctx, options)
	if errors.Is(err, ErrNoReleaseInstall) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAutomaticUpdate, err)
	}
	if result.Unchanged {
		return nil
	}
	if execFn == nil {
		execFn = syscall.Exec
	}
	binary := filepath.Join(options.DataRoot, "current", "bin", "fanloop")
	argv := append([]string{binary}, os.Args[1:]...)
	env := append(os.Environ(), envReexec+"=1")
	if err := execFn(binary, argv, env); err != nil {
		return fmt.Errorf("%w: %v", ErrAutomaticUpdate, err)
	}
	return ErrProcessReplaced
}

func envFlag(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}
