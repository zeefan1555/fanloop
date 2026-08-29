package executionlog

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/zeefan1555/fanloop/internal/idl/storageidl"
	"golang.org/x/sys/unix"
)

const filename = "cli.jsonl"

func ReadAll(root string) ([]byte, error) {
	if !filepath.IsAbs(root) {
		return nil, fmt.Errorf("requirement root must be absolute")
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("requirement root must be an existing directory")
	}
	directory := filepath.Join(root, ".fanloop", "log")
	directoryDescriptor, err := unix.Open(directory, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if errors.Is(err, unix.ENOENT) {
		return []byte{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open CLI execution log directory: %w", err)
	}
	defer unix.Close(directoryDescriptor) //nolint:errcheck -- no recovery is possible during read-only projection
	descriptor, err := unix.Openat(directoryDescriptor, filename, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if errors.Is(err, unix.ENOENT) {
		return []byte{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open CLI execution log file: %w", err)
	}
	file := os.NewFile(uintptr(descriptor), filepath.Join(directory, filename))
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat CLI execution log file: %w", err)
	}
	if !fileInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("CLI execution log is not a regular file")
	}
	if err := unix.Flock(descriptor, unix.LOCK_SH); err != nil {
		return nil, fmt.Errorf("lock CLI execution log file: %w", err)
	}
	defer unix.Flock(descriptor, unix.LOCK_UN) //nolint:errcheck -- closing the descriptor also releases the lock
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read CLI execution log file: %w", err)
	}
	return content, nil
}

func Append(root string, entry *storageidl.CLIExecutionLogEntry) error {
	if !filepath.IsAbs(root) {
		return fmt.Errorf("requirement root must be absolute")
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("requirement root must be an existing directory")
	}
	if entry == nil {
		return fmt.Errorf("CLI execution log entry is required")
	}
	if err := entry.IsValid(); err != nil {
		return err
	}
	content, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	content = append(content, '\n')

	directory := filepath.Join(root, ".fanloop", "log")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create CLI execution log directory: %w", err)
	}
	directoryDescriptor, err := unix.Open(directory, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return fmt.Errorf("open CLI execution log directory: %w", err)
	}
	defer unix.Close(directoryDescriptor) //nolint:errcheck -- no recovery is possible during best-effort logging
	if err := unix.Fchmod(directoryDescriptor, 0o700); err != nil {
		return fmt.Errorf("set CLI execution log directory permissions: %w", err)
	}

	descriptor, err := openFile(directoryDescriptor)
	if err != nil {
		return fmt.Errorf("open CLI execution log file: %w", err)
	}
	file := os.NewFile(uintptr(descriptor), filepath.Join(directory, filename))
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat CLI execution log file: %w", err)
	}
	if !fileInfo.Mode().IsRegular() {
		return fmt.Errorf("CLI execution log is not a regular file")
	}
	if err := file.Chmod(0o600); err != nil {
		return fmt.Errorf("set CLI execution log file permissions: %w", err)
	}
	if err := unix.Flock(descriptor, unix.LOCK_EX); err != nil {
		return fmt.Errorf("lock CLI execution log file: %w", err)
	}
	defer unix.Flock(descriptor, unix.LOCK_UN) //nolint:errcheck -- closing the descriptor also releases the lock
	if written, err := file.Write(content); err != nil {
		return fmt.Errorf("append CLI execution log entry: %w", err)
	} else if written != len(content) {
		return fmt.Errorf("short CLI execution log write: %d of %d bytes", written, len(content))
	}
	return nil
}

func openFile(directoryDescriptor int) (int, error) {
	flags := unix.O_WRONLY | unix.O_APPEND | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK
	descriptor, err := unix.Openat(directoryDescriptor, filename, flags, 0)
	if !errors.Is(err, unix.ENOENT) {
		return descriptor, err
	}
	descriptor, err = unix.Openat(directoryDescriptor, filename, flags|unix.O_CREAT|unix.O_EXCL, 0o600)
	if errors.Is(err, unix.EEXIST) {
		return unix.Openat(directoryDescriptor, filename, flags, 0)
	}
	return descriptor, err
}
