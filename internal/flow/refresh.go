package flow

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/zeefan1555/fanloop/internal/distribution"
)

const (
	envNoAutoConfig = "FANLOOP_NO_AUTO_CONFIG"
	envUseFornax    = "FANLOOP_USE_FORNAX"
)

var refreshOptions = func(dataRoot string) distribution.Options {
	return distribution.Options{DataRoot: dataRoot}
}

var refreshRelease = distribution.RefreshCLI

var execProcess = syscall.Exec

var exitProcess = os.Exit

// refreshCLI updates a versioned install before a fresh non-dry-run init.
// A failure warns and returns nil so initialization continues. A successful
// upgrade re-executes the new binary and does not return.
func (runtime Runtime) refreshCLI(ctx context.Context, dataRoot string) error {
	err := refreshRelease(ctx, refreshOptions(dataRoot), execProcess)
	if errors.Is(err, distribution.ErrProcessReplaced) {
		exitProcess(0)
		return err
	}
	if err != nil {
		runtime.note("automatic update failed; checking for an installed fallback")
	}
	return nil
}

func envFlag(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

func (runtime Runtime) note(message string) {
	if runtime.Warn == nil {
		fmt.Fprintf(os.Stderr, "warning: %s\n", message)
		return
	}
	runtime.warn(message)
}
