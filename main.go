package main

import (
	"context"
	"os"
	"runtime"

	"github.com/zeefan1555/fanloop/cmd"
)

func main() {
	// syscall.Exec from a non-leader thread makes Linux replace the pid.
	// Pin this goroutine to the thread-group leader before any command runs
	// so fanloop update keeps the same pid when it re-execs the new release.
	runtime.LockOSThread()
	os.Exit(cmd.Execute(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
