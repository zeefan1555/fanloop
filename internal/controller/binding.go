package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/zeefan1555/fanloop/internal/configuration"
	"github.com/zeefan1555/fanloop/internal/idl/storageidl"
	"github.com/zeefan1555/fanloop/internal/release"
)

const Journal = ".fanloop/controller-binding.json"
const Home = "bound-release-home"
const Pending = "bound-release-home.pending"

type Binding = storageidl.ControllerBinding

func Decode(content []byte) (*Binding, error) {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var value Binding
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if err := decoder.Decode(new(json.RawMessage)); err != io.EOF {
		return nil, fmt.Errorf("controller binding has trailing JSON: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(content, &fields); err != nil {
		return nil, err
	}
	shape := reflect.TypeOf(value)
	for i := 0; i < shape.NumField(); i++ {
		field := shape.Field(i)
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		raw, present := fields[name]
		if !present && strings.HasSuffix(field.Tag.Get("thrift"), ",required") {
			return nil, fmt.Errorf("missing required controller field %s", name)
		}
		if present && name != "schema_version" && (len(raw) == 0 || raw[0] != '"') {
			return nil, fmt.Errorf("controller field %s must be a string", name)
		}
		delete(fields, name)
	}
	for name := range fields {
		return nil, fmt.Errorf("json: unknown field %q", name)
	}
	if err := Validate(&value); err != nil {
		return nil, err
	}
	return &value, nil
}

func Encode(value *Binding) ([]byte, error) {
	if err := Validate(value); err != nil {
		return nil, err
	}
	content, err := json.MarshalIndent(value, "", "  ")
	return append(content, '\n'), err
}

func Validate(value *Binding) error {
	if err := value.IsValid(); err != nil {
		return fmt.Errorf("invalid controller binding: %w", err)
	}
	for _, path := range []string{value.SourceReleasePath, value.ConfigurationSourcePath, value.ConfigurationPin} {
		if !absolute(path) {
			return fmt.Errorf("binding paths must be clean and absolute")
		}
	}
	if value.PreviousConfigurationPin != "" && !absolute(value.PreviousConfigurationPin) {
		return fmt.Errorf("previous pin must be clean and absolute")
	}
	parts := strings.Split(value.ControllerReleasePath, "/")
	if len(parts) != 2 || parts[0] != "releases" || !release.ValidVersion(parts[1]) {
		return fmt.Errorf("controller release must be releases/<version>")
	}
	if _, err := time.Parse(time.RFC3339Nano, value.PreparedAt); err != nil {
		return fmt.Errorf("invalid prepared_at")
	}
	if value.Phase == storageidl.ControllerBindingPhase_committed {
		if value.CommittedAt == nil {
			return fmt.Errorf("committed binding requires committed_at")
		}
		if _, err := time.Parse(time.RFC3339Nano, *value.CommittedAt); err != nil {
			return fmt.Errorf("invalid committed_at")
		}
	} else if value.CommittedAt != nil {
		return fmt.Errorf("prepared binding forbids committed_at")
	}
	if value.ConfigurationMode == storageidl.ControllerConfigurationMode_author_live {
		if value.ConfigurationManifestSha256 != nil || value.ConfigurationPin != value.ConfigurationSourcePath {
			return fmt.Errorf("author/live must retain its source without immutable identity")
		}
	} else if value.ConfigurationManifestSha256 == nil {
		return fmt.Errorf("immutable configuration requires content identity")
	}
	if value.Operation == storageidl.ControllerBindingOperation_initialize {
		if value.PreviousConfigurationPin != "" || value.ProtectedFilesSha256 != nil || value.PreviousBindingTransactionId != nil {
			return fmt.Errorf("initialize cannot contain protect history")
		}
	} else if value.ProtectedFilesSha256 == nil {
		return fmt.Errorf("protect requires original business file identity")
	}
	return nil
}

func Read(root string) (*Binding, error) {
	path := filepath.Join(root, Journal)
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	if err := release.RealPath(path, false); err != nil {
		return nil, err
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		return nil, fmt.Errorf("controller journal must be a 0600 regular file")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	value, err := Decode(content)
	if err != nil {
		return nil, err
	}
	if value.ConfigurationMode != storageidl.ControllerConfigurationMode_author_live {
		expected := filepath.Join(root, Home, "config", "snapshots", filepath.Base(value.ConfigurationSourcePath))
		if value.ConfigurationMode == storageidl.ControllerConfigurationMode_author_frozen {
			expected = filepath.Join(root, Home, "config", "author", strings.TrimPrefix(value.GetConfigurationManifestSha256(), "sha256:"))
		}
		if value.ConfigurationPin != expected {
			return nil, fmt.Errorf("journal configuration target escapes its fixed controller home")
		}
	}
	if value.PreviousBindingTransactionId != nil {
		archive := filepath.Join(root, ".fanloop", "audit", "controller-bindings", *value.PreviousBindingTransactionId+".json")
		if err := release.RealPath(archive, false); err != nil {
			return nil, err
		}
		content, err := os.ReadFile(archive)
		if err != nil {
			return nil, err
		}
		previous, err := Decode(content)
		if err != nil || previous.TransactionId != *value.PreviousBindingTransactionId || previous.Phase != storageidl.ControllerBindingPhase_committed || previous.ConfigurationMode != storageidl.ControllerConfigurationMode_author_live || previous.ConfigurationPin != value.PreviousConfigurationPin || previous.ControllerReleasePath != value.ControllerReleasePath || previous.ReleaseManifestSha256 != value.ReleaseManifestSha256 || previous.ReleaseResourcesSha256 != value.ReleaseResourcesSha256 {
			return nil, fmt.Errorf("previous committed author/live binding does not match")
		}
	}
	return value, nil
}

func ReadPin(root string) (string, bool, error) {
	path := filepath.Join(root, ".fanloop", "configuration-snapshot")
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		return "", false, nil
	} else if err != nil {
		return "", false, err
	}
	if err := release.RealPath(path, false); err != nil {
		return "", false, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", false, err
	}
	pin := strings.TrimSuffix(string(content), "\n")
	if !absolute(pin) || strings.ContainsAny(pin, "\r\n") {
		return "", false, fmt.Errorf("configuration pin must contain one clean absolute path")
	}
	return pin, true, nil
}

func absolute(path string) bool {
	return path != "/" && filepath.IsAbs(path) && filepath.Clean(path) == path && !strings.ContainsAny(path, "\r\n\x00")
}

func ValidatePin(root string, binding *Binding) (string, error) {
	pin, found, err := ReadPin(root)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("bound requirement configuration pin is missing")
	}
	manifest, err := configuration.InspectBinding(pin)
	if err != nil {
		return "", err
	}
	if binding != nil && binding.ConfigurationMode != storageidl.ControllerConfigurationMode_author_live && pin == binding.ConfigurationPin && manifest.SHA256 != binding.GetConfigurationManifestSha256() {
		return "", fmt.Errorf("protected configuration identity differs")
	}
	return pin, nil
}
