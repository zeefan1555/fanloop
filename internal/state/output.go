package state

import (
	"encoding/json"
	"fmt"

	"github.com/zeefan1555/fanloop/internal/idl/storageidl"
)

const CurrentOutputSchemaVersion = storageidl.OUTPUT_REGISTRY_SCHEMA_VERSION

type OutputRegistry struct {
	SchemaVersion int
	Workflow      WorkflowRef
	Outputs       map[string]RegisteredOutput
}

func NewOutputRegistry(current State) OutputRegistry {
	return OutputRegistry{
		SchemaVersion: CurrentOutputSchemaVersion,
		Workflow:      current.Release.Workflow,
		Outputs:       cloneRegisteredOutputs(current.Outputs),
	}
}

func (value OutputRegistry) Validate() error {
	if value.SchemaVersion != CurrentOutputSchemaVersion || value.Workflow.ID == "" ||
		value.Workflow.Digest == "" || value.Outputs == nil {
		return fmt.Errorf("invalid Output Registry header")
	}
	for key, output := range value.Outputs {
		if key == "" || output.ProducerStepID == "" || validateOutputValue(OutputValue{Type: output.Type, Value: output.Value}) != nil {
			return fmt.Errorf("invalid registered Output %q", key)
		}
	}
	return nil
}

func cloneRegisteredOutputs(values map[string]RegisteredOutput) map[string]RegisteredOutput {
	result := make(map[string]RegisteredOutput, len(values))
	for key, value := range values {
		value.Value = append(json.RawMessage(nil), value.Value...)
		result[key] = value
	}
	return result
}
