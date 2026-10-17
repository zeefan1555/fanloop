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

// httpsAttemptTimeout bounds only the HTTPS attempt. SSH keeps its own connect timeout.
var httpsAttemptTimeout = 15 * time.Second

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
// The HTTPS URL is tried first, with httpsAttemptTimeout. One failure prints
// a notice and retries RepositorySSH. Later calls in the same session use SSH.
// Any other repository, including a local import path, is used unchanged.
func runProduction(ctx context.Context, repository string, operation func(ctx context.Context, remote string) error) (string, error) {
	if repository != Repository {
		return repository, operation(ctx, repository)
	}
	session := sessionFrom(ctx)
	if session.usingSSH() {
		return RepositorySSH, operation(ctx, RepositorySSH)
	}
	httpsCtx, cancel := context.WithTimeout(ctx, httpsAttemptTimeout)
	err := operation(httpsCtx, Repository)
	cancel()
	if err == nil {
		return Repository, nil
	}
	session.fallBack()
	return RepositorySSH, operation(ctx, RepositorySSH)
}

func fetchRemote(ctx context.Context, root, repository, ref string) (string, error) {
	return runProduction(ctx, repository, func(ctx context.Context, remote string) error {
		_, err := git(ctx, root, "fetch", "--quiet", "--depth=1", "--no-tags", remote, ref)
		return err
	})
}

// manifestsMatch compares snapshot identity. The stored repository URL is not
// part of that comparison, so an SSH-era snapshot remains valid for its commit.
func manifestsMatch(recorded, actual Manifest) bool {
	aligned := recorded
	aligned.Repository = actual.Repository
	return reflect.DeepEqual(aligned, actual)
}
