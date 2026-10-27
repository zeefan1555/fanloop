package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/zeefan1555/fanloop/errs"
	"github.com/zeefan1555/fanloop/internal/controller"
	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/idl/storageidl"
	"github.com/zeefan1555/fanloop/internal/release"
	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

func (s *Store) bindingWrite(binding *controller.Binding) error {
	content, err := controller.Encode(binding)
	if err != nil {
		return err
	}
	path := filepath.Join(s.Root, controller.Journal)
	if _, err := os.Lstat(path); err == nil {
		if err := release.RealPath(path, false); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	return atomicWrite(path, content)
}

func (s *Store) businessDigest() (string, error) {
	hash := sha256.New()
	for _, relative := range []string{"flow/state.json", "output/state.json", "trace/events.jsonl"} {
		path := filepath.Join(s.directory, filepath.FromSlash(relative))
		if err := release.RealPath(path, false); err != nil {
			return "", err
		}
		digest, err := release.FileDigest(path)
		if err != nil {
			return "", err
		}
		_, _ = hash.Write([]byte(relative + "\x00" + strings.TrimPrefix(digest, "sha256:") + "\n"))
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func (s *Store) checkIntent(resources *controller.Resources) error {
	binding, err := controller.Read(s.Root)
	if err != nil {
		return err
	}
	if binding == nil || !binding.DeepEqual(resources.Binding) {
		return fmt.Errorf("controller transaction changed")
	}
	pin, _, err := controller.ReadPin(s.Root)
	if err != nil {
		return err
	}
	if pin != binding.PreviousConfigurationPin && pin != binding.ConfigurationPin {
		return fmt.Errorf("configuration pin changed outside this transaction")
	}
	if binding.Operation == storageidl.ControllerBindingOperation_protect {
		digest, err := s.businessDigest()
		if err != nil {
			return err
		}
		if digest != binding.GetProtectedFilesSha256() {
			return fmt.Errorf("accepted business files changed")
		}
	}
	return nil
}

func (s *Store) archiveBinding(binding *controller.Binding) error {
	content, err := os.ReadFile(filepath.Join(s.Root, controller.Journal))
	if err != nil {
		return err
	}
	path := filepath.Join(s.directory, "audit", "controller-bindings", binding.TransactionId+".json")
	if err := release.MakeDirectory(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if _, err := os.Lstat(path); err == nil {
		if err := release.RealPath(path, false); err != nil {
			return err
		}
		actual, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if string(actual) != string(content) {
			return fmt.Errorf("archived controller binding differs")
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return atomicWrite(path, content)
}

func bindingFailure(err error) *erroridl.PublicError {
	return errs.NewCode(erroridl.ErrorCode_INTERNAL, err.Error(), nil)
}

func (s *Store) ProtectController(ctx context.Context, sourceRelease string, checkpoint func(string) error) (string, error) {
	if checkpoint == nil {
		checkpoint = func(string) error { return nil }
	}
	if err := release.RealPath(s.Root, true); err != nil {
		return "", err
	}
	if err := release.MakeDirectory(filepath.Dir(s.lockPath), 0700); err != nil {
		return "", err
	}
	binding, err := controller.Read(s.Root)
	if err != nil {
		return "", err
	}
	if binding != nil && binding.Phase == storageidl.ControllerBindingPhase_prepared && binding.Operation == storageidl.ControllerBindingOperation_initialize {
		return "", fmt.Errorf("prepared initialize is blocked; explicit recovery review is required")
	}
	var resources *controller.Resources
	if binding != nil && binding.Phase == storageidl.ControllerBindingPhase_prepared {
		resources, err = controller.Resume(s.Root, binding)
		if err != nil {
			return "", err
		}
		if failure := s.withExclusiveLock(func() *erroridl.PublicError {
			if err := s.checkIntent(resources); err != nil {
				return bindingFailure(err)
			}
			return nil
		}); failure != nil {
			return "", failure
		}
	} else {
		pin, found, err := controller.ReadPin(s.Root)
		if err != nil {
			return "", err
		}
		if binding != nil {
			if _, err := controller.ValidatePin(s.Root, binding); err != nil {
				return "", err
			}
			if binding.ConfigurationMode != storageidl.ControllerConfigurationMode_author_live && pin == binding.ConfigurationPin {
				if _, err := controller.Select(s.Root, ""); err != nil {
					return "", err
				}
				return filepath.Join(s.Root, controller.Home, binding.ControllerReleasePath, "bin", "fanloop"), nil
			}
			if binding.ConfigurationMode != storageidl.ControllerConfigurationMode_author_live || pin != binding.ConfigurationPin {
				return "", fmt.Errorf("actual configuration pin differs from the protected target; not rolling back rebind")
			}
			sourceRelease = filepath.Join(s.Root, controller.Home, binding.ControllerReleasePath)
		} else if _, err := os.Lstat(filepath.Join(s.Root, controller.Home)); err == nil {
			return "", fmt.Errorf("legacy controller can be selected but is not a protection transaction")
		} else if !os.IsNotExist(err) {
			return "", err
		}
		if sourceRelease == "" {
			home, err := release.DefaultDataRoot()
			if err != nil {
				return "", err
			}
			sourceRelease, err = release.BoundCurrent(home)
			if err != nil {
				return "", err
			}
		}
		configSource := pin
		if !found {
			home, err := release.DefaultDataRoot()
			if err != nil {
				return "", err
			}
			root, err := workflow.ConfigurationRoot(home)
			if err != nil {
				return "", err
			}
			configSource = filepath.Dir(root)
		}
		resources, err = controller.Plan(s.Root, sourceRelease, configSource, pin, storageidl.ControllerBindingOperation_protect, false)
		if err != nil {
			return "", err
		}
		digest, err := s.businessDigest()
		if err != nil {
			return "", err
		}
		resources.Binding.ProtectedFilesSha256 = &digest
		if binding != nil {
			if resources.Binding.ReleaseResourcesSha256 != binding.ReleaseResourcesSha256 {
				return "", fmt.Errorf("reused controller resource identity changed")
			}
			resources.Binding.PreviousBindingTransactionId = &binding.TransactionId
		}
		env := os.Environ()
		if binding != nil {
			env = controller.Environment(filepath.Join(s.Root, controller.Home), configSource, "")
		}
		env = append(env, "FANLOOP_CONFIG_SOURCE="+configSource, "FANLOOP_NO_AUTO_UPDATE=1", "FANLOOP_NO_AUTO_CONFIG=1")
		for _, args := range [][]string{{"flow", "status", "--root", s.Root}, {"doctor", "--root", s.Root}} {
			command := exec.CommandContext(ctx, filepath.Join(sourceRelease, "bin", "fanloop"), args...)
			command.Env = env
			if output, err := command.CombinedOutput(); err != nil {
				return "", fmt.Errorf("exact source controller health failed: %w: %s", err, output)
			}
		}
		after, err := s.businessDigest()
		if err != nil {
			return "", err
		}
		if after != digest {
			return "", fmt.Errorf("business files changed during health validation")
		}
		if err := resources.CheckSource(); err != nil {
			return "", err
		}
		if err := checkpoint("before-prepared"); err != nil {
			return "", err
		}
		if failure := s.withExclusiveLock(func() *erroridl.PublicError {
			latest, err := controller.Read(s.Root)
			if err != nil {
				return bindingFailure(err)
			}
			if (latest == nil) != (binding == nil) || latest != nil && !latest.DeepEqual(binding) {
				return bindingFailure(fmt.Errorf("controller binding changed during health validation"))
			}
			actual, _, err := controller.ReadPin(s.Root)
			if err != nil {
				return bindingFailure(err)
			}
			if actual != pin {
				return bindingFailure(fmt.Errorf("configuration pin changed during health validation"))
			}
			current, err := s.businessDigest()
			if err != nil {
				return bindingFailure(err)
			}
			if current != digest {
				return bindingFailure(fmt.Errorf("business files changed before prepared"))
			}
			if _, err := os.Lstat(filepath.Join(s.Root, controller.Pending)); err == nil {
				return bindingFailure(fmt.Errorf("unknown controller staging path exists"))
			} else if !os.IsNotExist(err) {
				return bindingFailure(err)
			}
			if binding != nil {
				if err := s.archiveBinding(binding); err != nil {
					return bindingFailure(err)
				}
			}
			if err := s.bindingWrite(resources.Binding); err != nil {
				return bindingFailure(err)
			}
			return nil
		}); failure != nil {
			return "", failure
		}
	}
	if _, err := os.Lstat(filepath.Join(s.Root, controller.Home)); os.IsNotExist(err) || resources.Binding.PreviousBindingTransactionId != nil {
		if err := resources.Prepare(ctx, checkpoint); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}
	if err := checkpoint("before-publish"); err != nil {
		return "", err
	}
	unlock, err := resources.LockPublication()
	if err != nil {
		return "", err
	}
	defer unlock()
	failure := s.withExclusiveLock(func() *erroridl.PublicError {
		if err := s.checkIntent(resources); err != nil {
			return bindingFailure(err)
		}
		if err := resources.CheckSource(); err != nil {
			return bindingFailure(err)
		}
		if err := resources.Publish(); err != nil {
			return bindingFailure(err)
		}
		if err := resources.CheckPublished(); err != nil {
			return bindingFailure(err)
		}
		if err := checkpoint("published"); err != nil {
			return bindingFailure(err)
		}
		if err := s.PinConfiguration(resources.Binding.ConfigurationPin); err != nil {
			return bindingFailure(err)
		}
		if err := checkpoint("pinned"); err != nil {
			return bindingFailure(err)
		}
		committed := *resources.Binding
		committed.Phase = storageidl.ControllerBindingPhase_committed
		now := time.Now().UTC().Format(time.RFC3339Nano)
		committed.CommittedAt = &now
		if err := s.bindingWrite(&committed); err != nil {
			return bindingFailure(err)
		}
		return nil
	})
	if failure != nil {
		rollbackFailure := s.withExclusiveLock(func() *erroridl.PublicError {
			actual, _, err := controller.ReadPin(s.Root)
			if err != nil {
				return bindingFailure(err)
			}
			if actual != resources.Binding.ConfigurationPin {
				return nil
			}
			if err := s.checkIntent(resources); err != nil {
				return bindingFailure(err)
			}
			path := filepath.Join(s.directory, "configuration-snapshot")
			if resources.Binding.PreviousConfigurationPin == "" {
				err = os.Remove(path)
			} else {
				err = atomicWriteMode(path, []byte(resources.Binding.PreviousConfigurationPin+"\n"), 0600)
			}
			if err != nil {
				return bindingFailure(err)
			}
			return nil
		})
		if rollbackFailure != nil {
			return "", fmt.Errorf("%w; precise pin rollback failed: %v", failure, rollbackFailure)
		}
		return "", failure
	}
	return filepath.Join(s.Root, controller.Home, resources.Binding.ControllerReleasePath, "bin", "fanloop"), nil
}

func (s *Store) InitializeController(resources *controller.Resources, next state.State, event state.Event, checkpoint func(string) error) *erroridl.PublicError {
	if checkpoint == nil {
		checkpoint = func(string) error { return nil }
	}
	if err := release.MakeDirectory(filepath.Dir(s.lockPath), 0700); err != nil {
		return bindingFailure(err)
	}
	return s.withExclusiveLock(func() *erroridl.PublicError {
		binding, err := controller.Read(s.Root)
		if err != nil {
			return bindingFailure(err)
		}
		if binding != nil {
			return bindingFailure(fmt.Errorf("controller binding already exists; prepared initialize is never resumed"))
		}
		if s.Exists() {
			return errs.NewCode(erroridl.ErrorCode_ALREADY_INITIALIZED, "requirement is already initialized", nil)
		}
		for _, path := range []string{s.outputPath, s.eventsPath, filepath.Join(s.directory, "configuration-snapshot"), filepath.Join(s.Root, controller.Home)} {
			if _, err := os.Lstat(path); err == nil {
				return bindingFailure(fmt.Errorf("unknown initialization path exists: %s", path))
			} else if !os.IsNotExist(err) {
				return bindingFailure(err)
			}
		}
		if err := resources.CheckSource(); err != nil {
			return bindingFailure(err)
		}
		if err := resources.CheckTarget(filepath.Join(s.Root, controller.Pending)); err != nil {
			return bindingFailure(err)
		}
		if err := checkpoint("before-prepared"); err != nil {
			return bindingFailure(err)
		}
		if err := s.bindingWrite(resources.Binding); err != nil {
			return bindingFailure(err)
		}
		if err := checkpoint("prepared"); err != nil {
			return bindingFailure(err)
		}
		if err := resources.Publish(); err != nil {
			return bindingFailure(err)
		}
		if err := resources.CheckPublished(); err != nil {
			return bindingFailure(err)
		}
		if err := checkpoint("published"); err != nil {
			return bindingFailure(err)
		}
		if err := s.PinConfiguration(resources.Binding.ConfigurationPin); err != nil {
			return bindingFailure(err)
		}
		if err := checkpoint("pinned"); err != nil {
			return bindingFailure(err)
		}
		if failure := s.commitLocked(next, event); failure != nil {
			return failure
		}
		if err := checkpoint("accepted"); err != nil {
			return bindingFailure(err)
		}
		committed := *resources.Binding
		committed.Phase = storageidl.ControllerBindingPhase_committed
		now := time.Now().UTC().Format(time.RFC3339Nano)
		committed.CommittedAt = &now
		if err := s.bindingWrite(&committed); err != nil {
			return bindingFailure(err)
		}
		return nil
	})
}
