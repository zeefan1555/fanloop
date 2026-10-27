package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"

	"github.com/zeefan1555/fanloop/cmd"
	"github.com/zeefan1555/fanloop/errs"
	"github.com/zeefan1555/fanloop/internal/idl/commonidl"
	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/output"
)

func main() {
	outputPath := flag.String("output", "packaging/npm/scripts/commands.json", "launcher command metadata")
	flag.Parse()
	content, err := generate()
	if err != nil {
		panic(err)
	}
	if err := os.MkdirAll(filepath.Dir(*outputPath), 0755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(*outputPath, content, 0644); err != nil {
		panic(err)
	}
}

func generate() ([]byte, error) {
	failure := errs.NewCode(erroridl.ErrorCode_INTERNAL, "", nil)
	var envelope bytes.Buffer
	if err := output.FailureWithNotice(&envelope, failure, output.Meta{}, commonidl.Notice{}); err != nil {
		return nil, err
	}
	metadata := struct {
		Commands      map[string]cmd.LauncherCommand `json:"commands"`
		InternalError struct {
			Envelope json.RawMessage `json:"envelope"`
			ExitCode int             `json:"exit_code"`
		} `json:"internal_error"`
	}{Commands: cmd.LauncherCommands()}
	metadata.InternalError.Envelope = envelope.Bytes()
	metadata.InternalError.ExitCode = errs.ExitCode(failure)
	content, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(content, '\n'), nil
}
