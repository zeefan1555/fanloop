package card

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zeefan1555/fanloop/internal/idl/storageidl"
	"github.com/zeefan1555/fanloop/internal/state"
	"github.com/zeefan1555/fanloop/internal/traceconfig"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

const (
	CurrentProjectionSchemaVersion = storageidl.CARD_PROJECTION_SCHEMA_VERSION
	projectionRelativePath         = ".fanloop/card/projection.json"
)

type Projection struct {
	SchemaVersion      int
	Requirement        state.Requirement
	Release            state.Release
	CurrentStepID      *string
	CurrentStepStatus  state.StepStatus
	CurrentStepSummary string
	CurrentEvidence    []state.Evidence
	Outputs            map[string]state.RegisteredOutput
	TraceDocumentURL   string
	CLILogDocumentURL  string
	SourceEventID      string
	UpdatedAt          time.Time
}

func ProjectionPath(root string) string {
	return filepath.Join(root, filepath.FromSlash(projectionRelativePath))
}

func WriteProjection(root string, current state.State) error {
	projection := Projection{
		SchemaVersion: CurrentProjectionSchemaVersion, Requirement: current.Requirement, Release: current.Release,
		CurrentStepID: cloneString(current.CurrentStepID), CurrentStepStatus: current.CurrentStepStatus,
		CurrentStepSummary: current.CurrentStepSummary, CurrentEvidence: append([]state.Evidence(nil), current.CurrentEvidence...),
		Outputs: cloneOutputs(current.Outputs), TraceDocumentURL: traceDocumentURL(current), CLILogDocumentURL: cliLogDocumentURL(current),
		SourceEventID: current.LastEventID, UpdatedAt: current.UpdatedAt,
	}
	if err := projection.Validate(); err != nil {
		return err
	}
	storedState, err := state.ToStorageIDL(current)
	if err != nil {
		return err
	}
	outputs, err := state.OutputsToStorageIDL(current.Outputs)
	if err != nil {
		return err
	}
	stored := &storageidl.CardProjection{
		SchemaVersion: int32(CurrentProjectionSchemaVersion), Requirement: storedState.Requirement, Release: storedState.Release,
		CurrentStepId: storedState.CurrentStepId, CurrentStepStatus: storedState.CurrentStepStatus,
		CurrentStepSummary: storedState.CurrentStepSummary, CurrentEvidence: storedState.CurrentEvidence,
		Outputs: outputs, SourceEventId: projection.SourceEventID, UpdatedAt: storedState.UpdatedAt,
	}
	if projection.TraceDocumentURL != "" {
		stored.TraceDocumentUrl = stringPointer(projection.TraceDocumentURL)
	}
	if projection.CLILogDocumentURL != "" {
		stored.CliLogDocumentUrl = stringPointer(projection.CLILogDocumentURL)
	}
	if err := stored.IsValid(); err != nil {
		return err
	}
	content, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}
	return replaceFile(ProjectionPath(root), append(content, '\n'))
}

func LoadProjection(root string) (Projection, error) {
	content, err := os.ReadFile(ProjectionPath(root))
	if err != nil {
		return Projection{}, err
	}
	var stored storageidl.CardProjection
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&stored); err != nil {
		return Projection{}, err
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("trailing JSON value")
		}
		return Projection{}, err
	}
	if stored.SchemaVersion != storageidl.CARD_PROJECTION_SCHEMA_VERSION {
		return Projection{}, fmt.Errorf("unsupported Card projection schema_version %d", stored.SchemaVersion)
	}
	if err := stored.IsValid(); err != nil {
		return Projection{}, err
	}
	outputs, err := state.OutputsFromStorageIDL(stored.Outputs)
	if err != nil {
		return Projection{}, err
	}
	integrations := &storageidl.Integrations{}
	if stored.TraceDocumentUrl != nil {
		registry, err := storageidl.RegistryProfileFromString(string(projectionRegistry(stored.Release.Workflow.Id, stored.GetCliLogDocumentUrl())))
		if err != nil {
			return Projection{}, err
		}
		integrations.Trace = &storageidl.TraceBinding{
			DocumentUrl: stored.GetTraceDocumentUrl(), Registry: registry,
		}
		if stored.CliLogDocumentUrl != nil {
			integrations.Trace.CliLogDocumentUrl = stringPointer(stored.GetCliLogDocumentUrl())
		}
	}
	current, err := state.FromStorageIDL(&storageidl.FlowState{
		SchemaVersion: state.CurrentStateSchemaVersion, Requirement: stored.Requirement, Release: stored.Release,
		CurrentStepId: stored.CurrentStepId, CurrentStepStatus: stored.CurrentStepStatus,
		CurrentStepSummary: stored.CurrentStepSummary, CurrentEvidence: stored.CurrentEvidence,
		Integrations: integrations, LastEventId: stored.SourceEventId, CreatedAt: stored.UpdatedAt, UpdatedAt: stored.UpdatedAt,
	}, outputs)
	if err != nil {
		return Projection{}, err
	}
	projection := Projection{
		SchemaVersion: int(stored.SchemaVersion), Requirement: current.Requirement, Release: current.Release,
		CurrentStepID: current.CurrentStepID, CurrentStepStatus: current.CurrentStepStatus,
		CurrentStepSummary: current.CurrentStepSummary, CurrentEvidence: current.CurrentEvidence,
		Outputs: current.Outputs, TraceDocumentURL: stored.GetTraceDocumentUrl(), CLILogDocumentURL: stored.GetCliLogDocumentUrl(),
		SourceEventID: stored.SourceEventId, UpdatedAt: current.UpdatedAt,
	}
	if err := projection.Validate(); err != nil {
		return Projection{}, err
	}
	return projection, nil
}

func (value Projection) Validate() error {
	if value.SchemaVersion != CurrentProjectionSchemaVersion {
		return fmt.Errorf("unsupported Card projection schema_version %d", value.SchemaVersion)
	}
	if strings.TrimSpace(value.Requirement.Title) == "" || value.Release.Version == "" ||
		value.Release.Workflow.ID == "" || value.Release.Workflow.Digest == "" ||
		value.Outputs == nil || value.SourceEventID == "" || value.UpdatedAt.IsZero() {
		return fmt.Errorf("invalid Card projection header")
	}
	if value.TraceDocumentURL != "" && !state.ValidTraceDocumentURL(value.TraceDocumentURL) {
		return fmt.Errorf("invalid Card projection Trace document URL")
	}
	if value.CLILogDocumentURL != "" && !state.ValidTraceDocumentURL(value.CLILogDocumentURL) {
		return fmt.Errorf("invalid Card projection CLI log document URL")
	}
	if value.CLILogDocumentURL != "" && value.TraceDocumentURL == "" {
		return fmt.Errorf("Card projection CLI log document requires a Trace document")
	}
	current := value.State()
	if err := current.Validate(); err != nil {
		return fmt.Errorf("invalid Card projection: %w", err)
	}
	loaded, err := workflow.LoadRef(value.Release.Workflow.Ref())
	if err != nil {
		return err
	}
	if err := current.ValidateAgainst(loaded.Workflow); err != nil {
		return fmt.Errorf("invalid Card projection: %w", err)
	}
	return nil
}

func (value Projection) State() state.State {
	current := state.State{
		SchemaVersion: state.CurrentStateSchemaVersion, Requirement: value.Requirement, Release: value.Release,
		CurrentStepID: cloneString(value.CurrentStepID), CurrentStepStatus: value.CurrentStepStatus,
		CurrentStepSummary: value.CurrentStepSummary, CurrentEvidence: append([]state.Evidence(nil), value.CurrentEvidence...),
		Outputs: cloneOutputs(value.Outputs), Integrations: state.Integrations{}, LastEventID: value.SourceEventID,
		CreatedAt: value.UpdatedAt, UpdatedAt: value.UpdatedAt,
	}
	if value.TraceDocumentURL != "" {
		current.Integrations.Trace = &state.TraceBinding{
			DocumentURL: value.TraceDocumentURL, Registry: projectionRegistry(value.Release.Workflow.ID, value.CLILogDocumentURL), CLILogDocumentURL: value.CLILogDocumentURL,
		}
	}
	return current
}

func traceDocumentURL(current state.State) string {
	if current.Integrations.Trace == nil {
		return ""
	}
	return current.Integrations.Trace.DocumentURL
}

func cliLogDocumentURL(current state.State) string {
	if current.Integrations.Trace == nil {
		return ""
	}
	return current.Integrations.Trace.CLILogDocumentURL
}

func projectionRegistry(workflowID, cliLogDocumentURL string) traceconfig.RegistryProfile {
	registry, ok := traceconfig.Resolve(traceconfig.RegistryProduction, workflowID)
	if ok && registry.RequireCLILogDocument && cliLogDocumentURL == "" {
		return traceconfig.RegistryTest
	}
	return traceconfig.RegistryProduction
}

func replaceFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".projection-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err == nil {
		_, err = temporary.Write(content)
	}
	if syncErr := temporary.Sync(); err == nil {
		err = syncErr
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func cloneOutputs(values map[string]state.RegisteredOutput) map[string]state.RegisteredOutput {
	result := make(map[string]state.RegisteredOutput, len(values))
	for key, value := range values {
		value.Value = append(json.RawMessage(nil), value.Value...)
		result[key] = value
	}
	return result
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func stringPointer(value string) *string { return &value }
