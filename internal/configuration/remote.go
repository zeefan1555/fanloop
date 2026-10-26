package configuration

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"sync"
	"time"
)

// RepositorySSH is the SSH URL for the same production repository as Repository.
const RepositorySSH = "git@github.com:zeefan1555/second-brain-skills.git"

// RemoteNotice receives the one-line SSH fallback notice. It defaults to stderr.
var RemoteNotice io.Writer = os.Stderr

// Remote probes and Git pack transfers use separate budgets.
var httpsAttemptTimeout = 15 * time.Second
var fetchAttemptTimeout = 2 * time.Minute

const remoteFallbackNotice = "notice: HTTPS configuration fetch failed; falling back to " + RepositorySSH

type remoteSession struct {
	mu  sync.Mutex
	ssh bool
}

type remoteSessionKey struct{}

func withRemoteSession(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if session, ok := ctx.Value(remoteSessionKey{}).(*remoteSession); ok && session != nil {
		return ctx
	}
	return context.WithValue(ctx, remoteSessionKey{}, &remoteSession{})
}

func sessionFrom(ctx context.Context) *remoteSession {
	if ctx != nil {
		if session, ok := ctx.Value(remoteSessionKey{}).(*remoteSession); ok && session != nil {
			return session
		}
	}
	return &remoteSession{}
}

func (session *remoteSession) usingSSH() bool {
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.ssh
}

func (session *remoteSession) fallBack() {
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.ssh {
		return
	}
	session.ssh = true
	writer := RemoteNotice
	if writer == nil {
		writer = os.Stderr
	}
	fmt.Fprintln(writer, remoteFallbackNotice)
}

// runProduction resolves the production remote once per operation.
// The HTTPS URL is tried first, with an independent attempt budget. One failure prints
// a notice and retries RepositorySSH. Later calls in the same session use SSH.
// Any other repository, including a local import path, is used unchanged.
func runProduction(ctx context.Context, repository string, timeout time.Duration, operation func(ctx context.Context, remote string) error) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return repository, gitFailure("remote", ctx, nil)
	}
	attempt := func(remote string) error {
		attemptCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return operation(attemptCtx, remote)
	}
	if repository != Repository {
		return repository, operation(ctx, repository)
	}
	session := sessionFrom(ctx)
	if session.usingSSH() {
		err := attempt(RepositorySSH)
		if err != nil {
			err = fmt.Errorf("SSH: %w", err)
		}
		return RepositorySSH, err
	}
	err := attempt(Repository)
	if err == nil {
		return Repository, nil
	}
	if ctx.Err() != nil {
		return Repository, fmt.Errorf("HTTPS: %w", err)
	}
	session.fallBack()
	sshErr := attempt(RepositorySSH)
	if sshErr != nil {
		return RepositorySSH, fmt.Errorf("HTTPS: %w; SSH: %w", err, sshErr)
	}
	return RepositorySSH, nil
}

// manifestsMatch compares snapshot identity. The stored repository URL is not
// part of that comparison, so an SSH-era snapshot remains valid for its commit.
func manifestsMatch(recorded, actual Manifest) bool {
	aligned := recorded
	aligned.Repository = actual.Repository
	return reflect.DeepEqual(aligned, actual)
}
