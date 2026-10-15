package fornaxmirror

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIClientSkillsFollowsPaginationAndRejectsDuplicateExactKey(t *testing.T) {
	executable := filepath.Join(t.TempDir(), "fornax-cli")
	script := `#!/bin/sh
set -eu
page=1
output=
previous=
for argument in "$@"; do
  if [ "$previous" = "--page-no" ]; then page="$argument"; fi
  if [ "$previous" = "--output" ]; then output="$argument"; fi
  previous="$argument"
done
mkdir -p "$output"
if [ "$page" = "1" ]; then
  printf '%s\n' '{"skills":[{"id":"other","skill_key":"fanloop.workflow.target-old"}],"total":2}' > "$output/page.json"
else
  printf '%s\n' '{"skills":[{"id":"target","skill_key":"fanloop.workflow.target"}],"total":2}' > "$output/page.json"
fi
`
	if err := os.WriteFile(executable, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	client := CLIClient{Executable: executable}
	skills, err := client.Skills(context.Background(), "1", "fanloop.workflow.target")
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 || skills[0].ID != "target" {
		t.Fatalf("skills = %#v", skills)
	}
	duplicate := strings.Replace(script, `"skill_key":"fanloop.workflow.target-old"`, `"skill_key":"fanloop.workflow.target"`, 1)
	if err = os.WriteFile(executable, []byte(duplicate), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err = client.Skills(context.Background(), "1", "fanloop.workflow.target"); err == nil || !strings.Contains(err.Error(), "duplicate exact") {
		t.Fatalf("error = %v", err)
	}
}

func TestListByPrefixSkipsHyphenKeyAndKeepsValidKeys(t *testing.T) {
	executable := filepath.Join(t.TempDir(), "fornax-cli")
	script := `#!/bin/sh
set -eu
output=
previous=
for argument in "$@"; do
  if [ "$previous" = "--output" ]; then output="$argument"; fi
  previous="$argument"
done
mkdir -p "$output"
printf '%s\n' '{"skills":[{"id":"a","skill_key":"fanloop.workflow.linear_runtime"},{"id":"b","skill_key":"fanloop.workflow.pstack-native"},{"id":"c","skill_key":"fanloop.workflow.pstack_native"}],"total":3}' > "$output/page.json"
`
	if err := os.WriteFile(executable, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	var warned []string
	previous := warnSkippedKey
	warnSkippedKey = func(key string) { warned = append(warned, key) }
	t.Cleanup(func() { warnSkippedKey = previous })
	skills, err := (CLIClient{Executable: executable}).ListByPrefix(context.Background(), "1", "fanloop.workflow.")
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 2 || skills[0].ID != "a" || skills[0].Key != "fanloop.workflow.linear_runtime" || skills[1].ID != "c" || skills[1].Key != "fanloop.workflow.pstack_native" {
		t.Fatalf("skills = %#v", skills)
	}
	if len(warned) != 1 || warned[0] != "fanloop.workflow.pstack-native" {
		t.Fatalf("warned = %#v", warned)
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stderr := os.Stderr
	os.Stderr = writer
	previous("fanloop.workflow.pstack-native")
	os.Stderr = stderr
	_ = writer.Close()
	message, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(message) != "warning: skipping Fornax Skill key fanloop.workflow.pstack-native\n" {
		t.Fatalf("warning = %q", message)
	}
	empty := strings.Replace(script, `"id":"a"`, `"id":""`, 1)
	if err = os.WriteFile(executable, []byte(empty), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err = (CLIClient{Executable: executable}).ListByPrefix(context.Background(), "1", "fanloop.workflow."); err == nil || err.Error() != "invalid Fornax Skill key fanloop.workflow.linear_runtime" {
		t.Fatalf("error = %v", err)
	}
}

func TestCLIClientInstallRejectsAnythingExceptOneOrdinaryDirectory(t *testing.T) {
	for name, commands := range map[string]string{
		"extra file":      `mkdir -p "$destination/package"; touch "$destination/extra"`,
		"symlink wrapper": `mkdir -p "$destination/real"; ln -s real "$destination/package"`,
	} {
		t.Run(name, func(t *testing.T) {
			executable := filepath.Join(t.TempDir(), "fornax-cli")
			script := `#!/bin/sh
set -eu
destination=
previous=
for argument in "$@"; do
  if [ "$previous" = "--dir" ]; then destination="$argument"; fi
  previous="$argument"
done
` + commands + "\n"
			if err := os.WriteFile(executable, []byte(script), 0755); err != nil {
				t.Fatal(err)
			}
			client := CLIClient{Executable: executable}
			if _, err := client.Install(context.Background(), "1", "fanloop.workflow.target", "1.0.0", t.TempDir()); err == nil {
				t.Fatal("invalid install output passed")
			}
		})
	}
}
