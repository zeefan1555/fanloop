package larkexec

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"time"
)

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Execute is the only Lark subprocess seam used by Fanloop.
var Execute = execute

func execute(ctx context.Context, args []string, stdin io.Reader, timeout time.Duration) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	command := exec.CommandContext(ctx, "lark-cli", args...)
	command.Stdin = stdin
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	result := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	if err == nil {
		return result, nil
	}
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		result.ExitCode = exitError.ExitCode()
		return result, nil
	}
	return result, err
}
