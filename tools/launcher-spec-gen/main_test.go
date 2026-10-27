package main

import (
	"bytes"
	"os"
	"testing"
)

func TestLauncherMetadataIsFresh(t *testing.T) {
	expected, err := generate()
	if err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile("../../packaging/npm/scripts/commands.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, expected) {
		t.Fatal("launcher metadata differs from the current Cobra commands and Go error envelope; run scripts/generate-cli-idl.sh")
	}
}
