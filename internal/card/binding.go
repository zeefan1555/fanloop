package card

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/zeefan1555/fanloop/internal/idl/storageidl"
)

const (
	CurrentBindingSchemaVersion = storageidl.CARD_BINDING_SCHEMA_VERSION
	bindingRelativePath         = ".fanloop/card/config.json"
)

type BotmuxBinding struct {
	SchemaVersion int
	ChatID        string
	SessionID     string
}

func BindingPath(root string) string {
	return filepath.Join(root, filepath.FromSlash(bindingRelativePath))
}

func LoadBinding(root string) (BotmuxBinding, bool, error) {
	path := BindingPath(root)
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return BotmuxBinding{}, false, nil
	}
	if err != nil {
		return BotmuxBinding{}, false, err
	}
	if !info.Mode().IsRegular() {
		return BotmuxBinding{}, false, fmt.Errorf("Card binding is not a regular file")
	}
	if info.Mode().Perm() != 0o600 {
		return BotmuxBinding{}, false, fmt.Errorf("Card binding permissions must be 0600")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return BotmuxBinding{}, false, err
	}
	binding, err := decodeBinding(content)
	return binding, err == nil, err
}

func LoadOrCaptureBinding(root string) (BotmuxBinding, bool, error) {
	if binding, found, err := LoadBinding(root); found || err != nil {
		return binding, found, err
	}
	chatID := strings.TrimSpace(os.Getenv("BOTMUX_CHAT_ID"))
	sessionID := strings.TrimSpace(os.Getenv("BOTMUX_SESSION_ID"))
	if chatID == "" && sessionID == "" {
		return BotmuxBinding{}, false, nil
	}
	if chatID == "" || sessionID == "" {
		return BotmuxBinding{}, false, fmt.Errorf("capturing a Card binding requires BOTMUX_CHAT_ID and BOTMUX_SESSION_ID")
	}
	binding := BotmuxBinding{SchemaVersion: CurrentBindingSchemaVersion, ChatID: chatID, SessionID: sessionID}
	if err := createBinding(BindingPath(root), binding); err != nil {
		if os.IsExist(err) {
			return LoadBinding(root)
		}
		return BotmuxBinding{}, false, err
	}
	return binding, true, nil
}

func decodeBinding(content []byte) (BotmuxBinding, error) {
	var stored storageidl.CardBinding
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&stored); err != nil {
		return BotmuxBinding{}, err
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("trailing JSON value")
		}
		return BotmuxBinding{}, err
	}
	if stored.SchemaVersion != storageidl.CARD_BINDING_SCHEMA_VERSION {
		return BotmuxBinding{}, fmt.Errorf("unsupported Card binding schema_version %d", stored.SchemaVersion)
	}
	if err := stored.IsValid(); err != nil {
		return BotmuxBinding{}, err
	}
	if strings.TrimSpace(stored.ChatId) == "" || strings.TrimSpace(stored.SessionId) == "" {
		return BotmuxBinding{}, fmt.Errorf("Card binding requires chat_id and session_id")
	}
	return BotmuxBinding{SchemaVersion: int(stored.SchemaVersion), ChatID: stored.ChatId, SessionID: stored.SessionId}, nil
}

func createBinding(path string, binding BotmuxBinding) error {
	stored := &storageidl.CardBinding{SchemaVersion: int32(binding.SchemaVersion), ChatId: binding.ChatID, SessionId: binding.SessionID}
	if err := stored.IsValid(); err != nil {
		return err
	}
	content, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".config-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Link(temporaryPath, path)
}
