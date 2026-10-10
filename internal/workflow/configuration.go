package workflow

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// LockConfiguration serializes installation and source-binding changes.
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
	file := os.NewFile(uintptr(fd), "configuration lock")
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		file.Close()
		return nil, fmt.Errorf("configuration lock must be a regular file")
	}
	return lockFile(file, exclusive)
}

// LockSource coordinates every CLI and Dashboard using the same package tree,
// even when their installation data homes differ. It creates no source files.
func LockSource(root string, exclusive bool) (func(), error) {
	fd, err := unix.Open(root, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	return lockFile(os.NewFile(uintptr(fd), root), exclusive)
}

func lockFile(file *os.File, exclusive bool) (func(), error) {
	mode := unix.LOCK_SH
	if exclusive {
		mode = unix.LOCK_EX
	}
	if err := unix.Flock(int(file.Fd()), mode); err != nil {
		file.Close()
		return nil, err
	}
	return func() { _ = file.Close() }, nil
}
