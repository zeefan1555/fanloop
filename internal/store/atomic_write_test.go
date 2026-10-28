package store

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type faultedAtomicTemp struct {
	*os.File
	chmodErr, writeErr, closeErr error
	short, emptyWrite            bool
	writes, closes               int
}

func (file *faultedAtomicTemp) Chmod(mode os.FileMode) error {
	if file.chmodErr != nil {
		return file.chmodErr
	}
	return file.File.Chmod(mode)
}

func (file *faultedAtomicTemp) Write(content []byte) (int, error) {
	file.writes++
	if file.writeErr != nil || file.short {
		length := len(content) / 2
		if file.emptyWrite {
			length = 0
		}
		written, err := file.File.Write(content[:length])
		if err != nil {
			return written, err
		}
		return written, file.writeErr
	}
	return file.File.Write(content)
}

func (file *faultedAtomicTemp) Close() error {
	file.closes++
	if err := file.File.Close(); err != nil {
		return err
	}
	return file.closeErr
}

func TestAtomicWriteModePublishesCompleteFile(t *testing.T) {
	for _, present := range []bool{false, true} {
		for _, mode := range []os.FileMode{0600, 0640} {
			t.Run(fmt.Sprintf("present-%t-mode-%o", present, mode), func(t *testing.T) {
				target := filepath.Join(t.TempDir(), "controller-binding.json")
				if present {
					if err := os.WriteFile(target, []byte("previous"), 0644); err != nil {
						t.Fatal(err)
					}
				}
				if err := atomicWriteMode(target, []byte("complete replacement"), mode); err != nil {
					t.Fatal(err)
				}
				content, err := os.ReadFile(target)
				if err != nil || string(content) != "complete replacement" {
					t.Fatalf("published content=%q error=%v", content, err)
				}
				info, err := os.Stat(target)
				if err != nil || info.Mode().Perm() != mode {
					t.Fatalf("published mode: %v %v", info, err)
				}
				staged, err := filepath.Glob(filepath.Join(filepath.Dir(target), ".fanloop-*"))
				if err != nil || len(staged) != 0 {
					t.Fatalf("temporary files remain: %v %v", staged, err)
				}
			})
		}
	}
}

func TestAtomicWriteFailuresNeverPublish(t *testing.T) {
	chmodErr, writeErr, closeErr := errors.New("injected chmod failure"), errors.New("injected write failure"), errors.New("injected close failure")
	for _, example := range []struct {
		name  string
		fault faultedAtomicTemp
		want  error
	}{
		{"chmod", faultedAtomicTemp{chmodErr: chmodErr}, chmodErr},
		{"write-empty", faultedAtomicTemp{writeErr: writeErr, emptyWrite: true}, writeErr},
		{"write-partial", faultedAtomicTemp{writeErr: writeErr}, writeErr},
		{"short-write", faultedAtomicTemp{short: true}, io.ErrShortWrite},
		{"close", faultedAtomicTemp{closeErr: closeErr}, closeErr},
		{"chmod-and-close", faultedAtomicTemp{chmodErr: chmodErr, closeErr: closeErr}, chmodErr},
		{"write-and-close", faultedAtomicTemp{writeErr: writeErr, closeErr: closeErr}, writeErr},
	} {
		for _, present := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/present-%t", example.name, present), func(t *testing.T) {
				target := filepath.Join(t.TempDir(), "controller-binding.json")
				if present {
					if err := os.WriteFile(target, []byte("previous complete journal"), 0640); err != nil {
						t.Fatal(err)
					}
					if err := os.Chmod(target, 0640); err != nil {
						t.Fatal(err)
					}
				}
				fault := example.fault
				err := atomicWriteWithTemp(target, []byte("new complete journal"), 0600, func() (atomicTempFile, error) {
					file, err := os.CreateTemp(filepath.Dir(target), ".fanloop-*")
					fault.File = file
					return &fault, err
				})
				if !errors.Is(err, example.want) {
					t.Errorf("lost write failure: got %v, want %v", err, example.want)
				}
				wantWrites := 1
				if fault.chmodErr != nil {
					wantWrites = 0
				}
				if fault.writes != wantWrites || fault.closes != 1 {
					t.Errorf("unexpected I/O after fault: writes=%d closes=%d", fault.writes, fault.closes)
				}
				content, readErr := os.ReadFile(target)
				if present {
					if readErr != nil || string(content) != "previous complete journal" {
						t.Errorf("failure replaced target: content=%q error=%v", content, readErr)
					}
					info, statErr := os.Stat(target)
					if statErr != nil || info.Mode().Perm() != 0640 {
						t.Errorf("failure replaced target mode: %v %v", info, statErr)
					}
				} else if !os.IsNotExist(readErr) {
					t.Errorf("failure published absent target: content=%q error=%v", content, readErr)
				}
				staged, err := filepath.Glob(filepath.Join(filepath.Dir(target), ".fanloop-*"))
				if err != nil || len(staged) != 0 {
					t.Errorf("failed temporary files remain: %v %v", staged, err)
				}
			})
		}
	}
}
