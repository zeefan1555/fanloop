package store

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zeefan1555/fanloop/errs"
	"github.com/zeefan1555/fanloop/internal/controller"
	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/workflow"
	"golang.org/x/sys/unix"
)

type Store struct {
	Root       string
	directory  string
	statePath  string
	outputPath string
	lockPath   string
	eventsPath string
	tracePath  string
	configPath string
}

func New(root string) (*Store, *erroridl.PublicError) {
	if !filepath.IsAbs(root) {
		return nil, errs.NewCode(erroridl.ErrorCode_INVALID_ARGUMENT, "--root must be an absolute path", nil)
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, errs.NewCode(erroridl.ErrorCode_INVALID_ARGUMENT, "--root must reference an existing directory", nil)
	}
	directory := filepath.Join(root, ".fanloop")
	return &Store{
		Root: root, directory: directory,
		statePath:  filepath.Join(directory, "flow", "state.json"),
		outputPath: filepath.Join(directory, "output", "state.json"),
		lockPath:   filepath.Join(directory, "flow", "state.lock"),
		eventsPath: filepath.Join(directory, "trace", "events.jsonl"),
		tracePath:  filepath.Join(directory, "trace", "events.md"),
		configPath: filepath.Join(directory, "trace", "config.json"),
	}, nil
}

func (s *Store) Exists() bool {
	_, err := os.Stat(s.statePath)
	return err == nil
}

func (s *Store) Load() (state.State, *erroridl.PublicError) {
	content, err := os.ReadFile(s.statePath)
	if os.IsNotExist(err) {
		return state.State{}, errs.NewCode(erroridl.ErrorCode_NOT_INITIALIZED, "requirement is not initialized", nil)
	}
	if err != nil {
		return state.State{}, errs.NewCode(erroridl.ErrorCode_STATE_CORRUPT, err.Error(), nil)
	}
	var header struct {
		SchemaVersion int `json:"schema_version"`
	}
	if err := json.Unmarshal(content, &header); err != nil || header.SchemaVersion == 0 {
		return state.State{}, errs.NewCode(erroridl.ErrorCode_STATE_CORRUPT, "invalid state header", nil)
	}
	if header.SchemaVersion != state.CurrentStateSchemaVersion {
		return state.State{}, errs.NewCode(erroridl.ErrorCode_STATE_SCHEMA_UNSUPPORTED, fmt.Sprintf("state schema version is unsupported: %d", header.SchemaVersion), nil)
	}
	registry, registryFailure := s.loadOutputs()
	if registryFailure != nil {
		return state.State{}, registryFailure
	}
	value, err := state.Decode(content, registry.Outputs)
	if err != nil {
		if errors.Is(err, state.ErrSchemaUnsupported) {
			return state.State{}, errs.NewCode(erroridl.ErrorCode_STATE_SCHEMA_UNSUPPORTED, err.Error(), nil)
		}
		return state.State{}, errs.NewCode(erroridl.ErrorCode_STATE_CORRUPT, err.Error(), nil)
	}
	if registry.Workflow != value.Release.Workflow {
		return state.State{}, errs.NewCode(erroridl.ErrorCode_STATE_CORRUPT, "Output Registry uses a different Workflow binding", nil)
	}
	return value, nil
}

func (s *Store) loadOutputs() (state.OutputRegistry, *erroridl.PublicError) {
	content, err := os.ReadFile(s.outputPath)
	if err != nil {
		return state.OutputRegistry{}, errs.NewCode(erroridl.ErrorCode_STATE_CORRUPT, err.Error(), nil)
	}
	value, err := state.DecodeOutputRegistry(content)
	if err != nil {
		return state.OutputRegistry{}, errs.NewCode(erroridl.ErrorCode_STATE_CORRUPT, err.Error(), nil)
	}
	return value, nil
}

func (s *Store) LoadBound() (state.State, workflow.Loaded, *erroridl.PublicError) {
	current, failure := s.Load()
	if failure != nil {
		return state.State{}, workflow.Loaded{}, failure
	}
	loaded, err := s.LoadWorkflow(current.Release.Workflow.ID)
	if err != nil {
		return state.State{}, workflow.Loaded{}, errs.NewCode(erroridl.ErrorCode_WORKFLOW_MISMATCH, err.Error(), nil)
	}
	if current.CurrentStepID != nil {
		if _, _, ok := loaded.Workflow.FindStep(*current.CurrentStepID); !ok {
			message := fmt.Sprintf("current_step_id %q references an unknown Step; current legal Step IDs: %s", *current.CurrentStepID, strings.Join(loaded.Workflow.OrderedStepIDs(), ", "))
			return state.State{}, workflow.Loaded{}, errs.NewCode(erroridl.ErrorCode_WORKFLOW_MISMATCH, message, nil)
		}
	}
	if err := current.ValidateAgainst(loaded.Workflow); err != nil {
		return state.State{}, workflow.Loaded{}, errs.NewCode(erroridl.ErrorCode_STATE_CORRUPT, err.Error(), nil)
	}
	events, failure := s.Events()
	if failure != nil {
		return state.State{}, workflow.Loaded{}, failure
	}
	if err := state.ValidateHistory(events, current, loaded); err != nil {
		return state.State{}, workflow.Loaded{}, errs.NewCode(erroridl.ErrorCode_STATE_CORRUPT, err.Error(), nil)
	}
	return current, loaded, nil
}

func (s *Store) Events() ([]state.Event, *erroridl.PublicError) {
	file, err := os.Open(s.eventsPath)
	if os.IsNotExist(err) {
		return []state.Event{}, nil
	}
	if err != nil {
		return nil, errs.NewCode(erroridl.ErrorCode_STATE_CORRUPT, err.Error(), nil)
	}
	defer file.Close()
	result := []state.Event{}
	reader := bufio.NewReader(file)
	for {
		line, readErr := reader.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			event, decodeErr := state.DecodeEvent(line)
			if decodeErr != nil {
				return nil, errs.NewCode(erroridl.ErrorCode_STATE_CORRUPT, decodeErr.Error(), nil)
			}
			result = append(result, event)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, errs.NewCode(erroridl.ErrorCode_STATE_CORRUPT, readErr.Error(), nil)
		}
	}
	return result, nil
}

func (s *Store) Commit(next state.State, event state.Event) *erroridl.PublicError {
	return s.withExclusiveLock(func() *erroridl.PublicError {
		return s.commitLocked(next, event)
	})
}

// WithLockedState serializes a projection or other derived write with Flow
// commits and reloads the latest committed State after acquiring the lock.
func (s *Store) WithLockedState(operation func(state.State) *erroridl.PublicError) *erroridl.PublicError {
	return s.withExclusiveLock(func() *erroridl.PublicError {
		current, _, failure := s.LoadBound()
		if failure != nil {
			return failure
		}
		return operation(current)
	})
}

func (s *Store) commitLocked(next state.State, event state.Event) *erroridl.PublicError {
	events, failure := s.Events()
	if failure != nil {
		return failure
	}
	if failure := s.validateCommit(events, next, event); failure != nil {
		return failure
	}
	events = append(events, event)
	loaded, err := s.LoadWorkflow(next.Release.Workflow.ID)
	if err != nil {
		return corrupt(err.Error())
	}
	stateContent, err := state.Encode(next)
	if err != nil {
		return commitError(err)
	}
	outputContent, err := state.EncodeOutputRegistry(state.NewOutputRegistry(next))
	if err != nil {
		return commitError(err)
	}
	eventsContent := bytes.Buffer{}
	for _, item := range events {
		content, marshalErr := state.EncodeEvent(item)
		if marshalErr != nil {
			return commitError(marshalErr)
		}
		eventsContent.Write(content)
		eventsContent.WriteByte('\n')
	}
	contents := map[string][]byte{
		s.eventsPath: eventsContent.Bytes(),
		s.outputPath: outputContent,
		s.tracePath:  RenderEvents(s.Root, next, loaded.Workflow, events),
		s.statePath:  stateContent,
	}
	if config, ok := RenderTraceConfig(next, events); ok {
		contents[s.configPath] = config
	}
	for _, directory := range []string{filepath.Join(s.directory, "flow"), filepath.Join(s.directory, "output"), filepath.Join(s.directory, "trace")} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return commitError(err)
		}
	}
	paths := []string{s.outputPath}
	if _, ok := contents[s.configPath]; ok {
		paths = append(paths, s.configPath)
	}
	paths = append(paths, s.eventsPath, s.tracePath, s.statePath)
	before := map[string]fileSnapshot{}
	for path := range contents {
		snapshot, snapshotErr := snapshot(path)
		if snapshotErr != nil {
			return commitError(snapshotErr)
		}
		before[path] = snapshot
	}
	for _, path := range paths {
		if err := atomicWrite(path, contents[path]); err != nil {
			for restorePath, previous := range before {
				_ = previous.restore(restorePath)
			}
			return commitError(err)
		}
	}
	return nil
}

func (s *Store) withExclusiveLock(commit func() *erroridl.PublicError) *erroridl.PublicError {
	if err := os.MkdirAll(filepath.Dir(s.lockPath), 0o700); err != nil {
		return commitError(err)
	}
	descriptor, err := unix.Open(s.lockPath, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return commitError(err)
	}
	lock := os.NewFile(uintptr(descriptor), s.lockPath)
	defer lock.Close()
	info, err := lock.Stat()
	if err != nil || !info.Mode().IsRegular() {
		if err == nil {
			err = fmt.Errorf("commit lock is not a regular file")
		}
		return commitError(err)
	}
	if err := unix.Flock(descriptor, unix.LOCK_EX); err != nil {
		return commitError(err)
	}
	defer unix.Flock(descriptor, unix.LOCK_UN) //nolint:errcheck -- closing the descriptor also releases the lock
	return commit()
}

func (s *Store) validateCommit(events []state.Event, next state.State, event state.Event) *erroridl.PublicError {
	if err := next.Validate(); err != nil {
		return corrupt(err.Error())
	}
	eventContent, err := state.EncodeEvent(event)
	if err != nil {
		return corrupt(err.Error())
	}
	if _, err := state.DecodeEvent(eventContent); err != nil {
		return corrupt(err.Error())
	}
	loaded, err := s.LoadWorkflow(next.Release.Workflow.ID)
	if err != nil {
		return corrupt(err.Error())
	}
	if err := next.ValidateAgainst(loaded.Workflow); err != nil {
		return corrupt(err.Error())
	}
	if next.LastEventID != event.ID || event.Workflow != next.Release.Workflow || event.ID == "" || event.OccurredAt.IsZero() {
		return corrupt("new event does not match the next state")
	}
	if err := event.ValidateAgainst(loaded.Workflow); err != nil {
		return corrupt(err.Error())
	}
	if s.Exists() {
		previous, failure := s.Load()
		if failure != nil {
			return failure
		}
		if len(events) == 0 || previous.LastEventID != events[len(events)-1].ID {
			return corrupt("state last_event_id does not match event history")
		}
	} else if len(events) != 0 {
		return corrupt("event history exists without state")
	}
	if err := state.ValidateHistory(append(events, event), next, loaded); err != nil {
		return corrupt(err.Error())
	}
	return nil
}

func (s *Store) RebuildProjection(dryRun bool) ([]byte, int, *erroridl.PublicError) {
	current, loaded, failure := s.LoadBound()
	if failure != nil {
		return nil, 0, failure
	}
	events, failure := s.Events()
	if failure != nil {
		return nil, 0, failure
	}
	content := RenderEvents(s.Root, current, loaded.Workflow, events)
	if !dryRun {
		if err := atomicWrite(s.tracePath, content); err != nil {
			return nil, 0, commitError(err)
		}
	}
	return content, len(events), nil
}

type fileSnapshot struct {
	exists  bool
	content []byte
	mode    os.FileMode
}

func snapshot(path string) (fileSnapshot, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fileSnapshot{}, nil
	}
	if err != nil {
		return fileSnapshot{}, err
	}
	content, err := os.ReadFile(path)
	return fileSnapshot{exists: true, content: content, mode: info.Mode().Perm()}, err
}

func (value fileSnapshot) restore(path string) error {
	if !value.exists {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	return atomicWriteMode(path, value.content, value.mode)
}

func atomicWrite(path string, content []byte) error {
	return atomicWriteMode(path, content, 0o600)
}

func atomicWriteMode(path string, content []byte, mode os.FileMode) error {
	return atomicWriteWithTemp(path, content, mode, func() (atomicTempFile, error) {
		return os.CreateTemp(filepath.Dir(path), ".fanloop-*")
	})
}

type atomicTempFile interface {
	Name() string
	Chmod(os.FileMode) error
	Write([]byte) (int, error)
	Close() error
}

func atomicWriteWithTemp(path string, content []byte, mode os.FileMode, create func() (atomicTempFile, error)) error {
	temporary, err := create()
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	err = temporary.Chmod(mode)
	if err == nil {
		var written int
		written, err = temporary.Write(content)
		if err == nil && written != len(content) {
			err = io.ErrShortWrite
		}
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

// PinConfiguration records the snapshot directory this requirement keeps using
// after config/current moves.
func (s *Store) PinConfiguration(source string) error {
	source = filepath.Clean(source)
	if _, err := workflow.SourceRoot(source); err != nil {
		return err
	}
	if err := os.MkdirAll(s.directory, 0700); err != nil {
		return err
	}
	return atomicWriteMode(filepath.Join(s.directory, "configuration-snapshot"), []byte(source+"\n"), 0600)
}

const configurationRebindAuditPath = "audit/configuration-rebindings.jsonl"

type configurationRebindAudit struct {
	SchemaVersion    int    `json:"schema_version"`
	TransactionID    string `json:"transaction_id"`
	Phase            string `json:"phase"`
	PreviousSnapshot string `json:"previous_snapshot"`
	NewSnapshot      string `json:"new_snapshot"`
	ReboundAt        string `json:"rebound_at"`
	Reason           string `json:"reason"`
}

// RebindConfiguration atomically replaces an initialized requirement's
// configuration pin and appends the matching audit record under the same
// requirement lock. The target Workflow must accept the current State and
// complete Event history before either file changes.
func (s *Store) RebindConfiguration(source, reason string, occurredAt time.Time) (string, string, *erroridl.PublicError) {
	source = filepath.Clean(source)
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "", "", errs.NewCode(erroridl.ErrorCode_INVALID_ARGUMENT, "rebind reason is required", nil)
	}
	root, err := workflow.SourceRoot(source)
	if err != nil {
		return "", "", errs.NewCode(erroridl.ErrorCode_WORKFLOW_INVALID, err.Error(), nil)
	}
	var previous string
	failure := s.withExclusiveLock(func() *erroridl.PublicError {
		current, failure := s.Load()
		if failure != nil {
			return failure
		}
		var pinned bool
		previous, pinned, err = s.readPin()
		if err != nil {
			return corrupt(err.Error())
		}
		if !pinned {
			return corrupt("configuration snapshot pin is missing")
		}
		loaded, loadErr := workflow.LoadFrom(root, current.Release.Workflow.ID)
		if loadErr != nil {
			return errs.NewCode(erroridl.ErrorCode_WORKFLOW_MISMATCH, loadErr.Error(), nil)
		}
		if validateErr := current.ValidateAgainst(loaded.Workflow); validateErr != nil {
			return errs.NewCode(erroridl.ErrorCode_WORKFLOW_MISMATCH, validateErr.Error(), nil)
		}
		events, eventsFailure := s.Events()
		if eventsFailure != nil {
			return eventsFailure
		}
		if validateErr := state.ValidateHistory(events, current, loaded); validateErr != nil {
			return errs.NewCode(erroridl.ErrorCode_WORKFLOW_MISMATCH, validateErr.Error(), nil)
		}

		transactionID := state.NewEventID()
		audit := configurationRebindAudit{
			SchemaVersion:    1,
			TransactionID:    transactionID,
			Phase:            "prepared",
			PreviousSnapshot: filepath.Base(previous),
			NewSnapshot:      filepath.Base(source),
			ReboundAt:        occurredAt.UTC().Format(time.RFC3339Nano),
			Reason:           reason,
		}
		preparedLine, marshalErr := json.Marshal(audit)
		if marshalErr != nil {
			return commitError(marshalErr)
		}
		preparedLine = append(preparedLine, '\n')
		audit.Phase = "committed"
		committedLine, marshalErr := json.Marshal(audit)
		if marshalErr != nil {
			return commitError(marshalErr)
		}
		committedLine = append(committedLine, '\n')
		pinPath := filepath.Join(s.directory, "configuration-snapshot")
		auditPath := filepath.Join(s.directory, filepath.FromSlash(configurationRebindAuditPath))
		if mkdirErr := os.MkdirAll(filepath.Dir(auditPath), 0o700); mkdirErr != nil {
			return commitError(mkdirErr)
		}
		beforePin, snapshotErr := snapshot(pinPath)
		if snapshotErr != nil {
			return commitError(snapshotErr)
		}
		beforeAudit, snapshotErr := snapshot(auditPath)
		if snapshotErr != nil {
			return commitError(snapshotErr)
		}
		auditContent := append(append([]byte(nil), beforeAudit.content...), preparedLine...)
		if writeErr := atomicWrite(auditPath, auditContent); writeErr != nil {
			return commitError(writeErr)
		}
		if writeErr := atomicWriteMode(pinPath, []byte(source+"\n"), 0o600); writeErr != nil {
			_ = beforeAudit.restore(auditPath)
			_ = beforePin.restore(pinPath)
			return commitError(writeErr)
		}
		auditContent = append(auditContent, committedLine...)
		if writeErr := atomicWrite(auditPath, auditContent); writeErr != nil {
			_ = beforePin.restore(pinPath)
			_ = beforeAudit.restore(auditPath)
			return commitError(writeErr)
		}
		return nil
	})
	if failure != nil {
		return "", "", failure
	}
	return previous, filepath.ToSlash(filepath.Join(".fanloop", configurationRebindAuditPath)), nil
}

// LoadWorkflow reads the Workflow bound to this requirement. A pin at
// .fanloop/configuration-snapshot wins over config/current. A requirement
// without that file still follows the active configuration root.
func (s *Store) LoadWorkflow(id string) (workflow.Loaded, error) {
	source, pinned, err := s.readPin()
	if err != nil {
		return workflow.Loaded{}, err
	}
	if !pinned {
		return workflow.Load(id)
	}
	root, err := workflow.SourceRoot(source)
	if err != nil {
		return workflow.Loaded{}, err
	}
	return workflow.LoadFrom(root, id)
}

func (s *Store) ConfigurationPin() (string, bool, error) {
	return s.readPin()
}

func (s *Store) readPin() (string, bool, error) {
	return controller.ReadPin(s.Root)
}

func commitError(err error) *erroridl.PublicError {
	return errs.NewCode(erroridl.ErrorCode_LOCAL_COMMIT_FAILED, err.Error(), nil)
}

func corrupt(message string) *erroridl.PublicError {
	return errs.NewCode(erroridl.ErrorCode_STATE_CORRUPT, message, nil)
}
