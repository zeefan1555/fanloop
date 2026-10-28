package controller

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/idl/storageidl"
	"github.com/zeefan1555/fanloop/internal/release"
	"golang.org/x/sys/unix"
)

func validBinding() *Binding {
	digest := "sha256:" + strings.Repeat("1", 64)
	return &Binding{
		SchemaVersion: 1, TransactionId: strings.Repeat("1", 32),
		Phase: storageidl.ControllerBindingPhase_prepared, ConfigurationMode: storageidl.ControllerConfigurationMode_snapshot,
		Operation:         storageidl.ControllerBindingOperation_initialize,
		SourceReleasePath: "/source/release", ControllerReleasePath: "releases/1.2.3",
		ReleaseManifestSha256: digest, BinarySha256: digest, ReleaseResourcesSha256: digest, ConfigurationPin: "/root/bound-release-home/config/snapshots/identity",
		ConfigurationManifestSha256: &digest, ConfigurationSourcePath: "/source/config", PreparedAt: "2026-10-24T00:00:00Z",
	}
}

func TestBindingRejectsInvalidJournal(t *testing.T) {
	base, err := Encode(validBinding())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(base); err != nil {
		t.Fatal(err)
	}
	for _, content := range [][]byte{append(append([]byte{}, base...), []byte("{}")...), []byte(strings.Replace(string(base), `"schema_version": 1`, `"unknown": 1, "schema_version": 1`, 1))} {
		if _, err := Decode(content); err == nil {
			t.Fatal("invalid JSON boundary accepted")
		}
	}
	for _, mutate := range []func(*Binding){
		func(b *Binding) { b.ControllerReleasePath = "releases/../escape" },
		func(b *Binding) { b.ConfigurationSourcePath = "relative" },
		func(b *Binding) { b.Phase = storageidl.ControllerBindingPhase_committed },
		func(b *Binding) { b.Operation = storageidl.ControllerBindingOperation_protect },
		func(b *Binding) { b.ConfigurationMode = storageidl.ControllerConfigurationMode_author_live },
		func(b *Binding) { b.PreviousConfigurationPin = "/previous" },
	} {
		binding := validBinding()
		mutate(binding)
		content, _ := json.Marshal(binding)
		if _, err := Decode(content); err == nil {
			t.Fatalf("illegal binding accepted: %s", content)
		}
	}
}

func TestBindingSharedRawSchemaTokens(t *testing.T) {
	content, err := os.ReadFile("testdata/journal-types.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Binding json.RawMessage `json:"binding"`
		Cases   []struct {
			Name  string `json:"name"`
			Raw   string `json:"raw"`
			Valid bool   `json:"valid"`
		} `json:"raw_schema_members"`
	}
	if err := json.Unmarshal(content, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, example := range fixture.Cases {
		t.Run(example.Name, func(t *testing.T) {
			raw := strings.Replace(string(fixture.Binding), `"schema_version": 1`, example.Raw, 1)
			if raw == string(fixture.Binding) {
				t.Fatal("schema token replacement did not match")
			}
			_, err := Decode([]byte(raw))
			if (err == nil) != example.Valid {
				t.Fatalf("valid=%t error=%v", example.Valid, err)
			}
		})
	}
}

func TestBindingSharedTimestamps(t *testing.T) {
	content, err := os.ReadFile("testdata/journal-types.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Binding map[string]json.RawMessage `json:"binding"`
		Valid   []string                   `json:"valid_timestamps"`
		Invalid []string                   `json:"invalid_timestamps"`
	}
	if err := json.Unmarshal(content, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"prepared_at", "committed_at"} {
		previous := fixture.Binding[field]
		for _, group := range []struct {
			valid  bool
			values []string
		}{{true, fixture.Valid}, {false, fixture.Invalid}} {
			for _, value := range group.values {
				t.Run(field+"/"+value, func(t *testing.T) {
					fixture.Binding[field], _ = json.Marshal(value)
					content, err := json.Marshal(fixture.Binding)
					if err != nil {
						t.Fatal(err)
					}
					_, err = Decode(content)
					if (err == nil) != group.valid {
						t.Fatalf("%s=%q valid=%t error=%v", field, value, group.valid, err)
					}
				})
			}
		}
		fixture.Binding[field] = previous
	}
}

func TestStrictDigestRejectsLinksAndSpecialFiles(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "regular"), []byte("content"), 0600); err != nil {
		t.Fatal(err)
	}
	old, err := release.DirectoryDigest(root)
	if err != nil {
		t.Fatal(err)
	}
	strict, err := release.StrictDirectoryDigest(root)
	if err != nil || strict != old {
		t.Fatalf("healthy digest changed: %s %v", strict, err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink("regular", link); err != nil {
		t.Fatal(err)
	}
	if legacy, err := release.DirectoryDigest(root); err != nil || legacy != old {
		t.Fatalf("legacy digest contract changed: %s %v", legacy, err)
	}
	if _, err := release.StrictDirectoryDigest(root); err == nil {
		t.Fatal("resource symlink accepted")
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := unix.Mkfifo(filepath.Join(root, "fifo"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := release.StrictDirectoryDigest(root); err == nil {
		t.Fatal("FIFO accepted")
	}
	ancestor := filepath.Join(root, "ancestor")
	if err := os.Symlink(root, ancestor); err != nil {
		t.Fatal(err)
	}
	if err := release.RealPath(filepath.Join(ancestor, "regular"), false); err == nil {
		t.Fatal("linked ancestor accepted")
	}
}
