package contracts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/idl"
)

var updateContracts = flag.Bool("update-contracts", false, "replace reviewed contract goldens")

type contractCase struct {
	SchemaVersion int               `json:"schema_version"`
	ID            string            `json:"id"`
	Environment   map[string]string `json:"environment,omitempty"`
	Commands      []contractCommand `json:"commands"`
}

type contractCommand struct {
	Args        []string          `json:"args"`
	Stdin       string            `json:"stdin,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
}

type commandResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

type fileResult struct {
	Mode    string `json:"mode"`
	SHA256  string `json:"sha256"`
	Content string `json:"content"`
}

type contractResult struct {
	SchemaVersion int                   `json:"schema_version"`
	ID            string                `json:"id"`
	Commands      []commandResult       `json:"commands"`
	Files         map[string]fileResult `json:"files"`
}

func TestGeneratedIDLGoPackagesStayUnderInternalIDL(t *testing.T) {
	repo := repositoryRoot(t)
	packages := map[string]string{
		"cardidl": "card.go", "cliidl": "cli.go", "commonidl": "common.go", "erroridl": "error.go",
		"flowidl": "flow.go", "opsidl": "ops.go", "releaseidl": "release.go", "storageidl": "storage.go", "traceidl": "trace.go",
		"yamlidl": "yaml.go",
	}
	for name, generatedFile := range packages {
		if _, err := os.Stat(filepath.Join(repo, name)); !os.IsNotExist(err) {
			t.Errorf("root-level generated package %s still exists: %v", name, err)
		}
		path := filepath.Join(repo, "internal", "idl", name, generatedFile)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("generated package %s is not under internal/idl: %v", name, err)
		}
	}
}

func TestPublicContracts(t *testing.T) {
	repo := repositoryRoot(t)
	binary := filepath.Join(t.TempDir(), "fanloop")
	build := exec.Command("go", "build", "-buildvcs=false", "-o", binary, ".")
	build.Dir = repo
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fanloop: %v\n%s", err, output)
	}
	paths, err := filepath.Glob(filepath.Join(repo, "tests", "contracts", "testdata", "*", "*", "case.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no contract cases found")
	}
	sort.Strings(paths)
	covered := map[string]bool{}
	for _, path := range paths {
		value := loadCase(t, path)
		for _, command := range value.Commands {
			if id := contractCommandID(command.Args); id != "" {
				covered[id] = true
			}
		}
		t.Run(value.ID, func(t *testing.T) {
			got := runCase(t, binary, path, value)
			for index, command := range value.Commands {
				if id := contractCommandID(command.Args); id != "" {
					assertCatalogedError(t, id, got.Commands[index])
				}
			}
			expectedPath := filepath.Join(filepath.Dir(path), "expected", "result.json")
			if *updateContracts {
				writeGolden(t, expectedPath, got)
			}
			want := loadGolden(t, expectedPath)
			if !reflect.DeepEqual(got, want) {
				gotJSON, _ := json.MarshalIndent(got, "", "  ")
				wantJSON, _ := json.MarshalIndent(want, "", "  ")
				t.Fatalf("contract changed; review before running -update-contracts\nwant:\n%s\ngot:\n%s", wantJSON, gotJSON)
			}
		})
	}
	for _, spec := range idl.CommandSpecs() {
		if !covered[spec.ID] {
			t.Errorf("public command %s has no reviewed Contract command", spec.ID)
		}
	}
}

func assertCatalogedError(t *testing.T, commandID string, result commandResult) {
	t.Helper()
	for _, content := range []string{result.Stdout, result.Stderr} {
		var value struct {
			Error *struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if json.Unmarshal([]byte(content), &value) != nil || value.Error == nil {
			continue
		}
		spec, ok := idl.LookupCommand(commandID)
		if !ok {
			t.Fatalf("missing CommandSpec for %s", commandID)
		}
		for _, cataloged := range spec.Errors {
			if cataloged.Code.String() == value.Error.Code && int(cataloged.ExitCode) == result.ExitCode {
				return
			}
		}
		t.Fatalf("%s emitted uncataloged error %s with exit code %d", commandID, value.Error.Code, result.ExitCode)
	}
}

func contractCommandID(args []string) string {
	for index := 0; index < len(args); {
		switch args[index] {
		case "--root", "--input":
			index += 2
		case "--dry-run":
			index++
		case "--version":
			return "version"
		default:
			if strings.HasPrefix(args[index], "-") {
				index++
				continue
			}
			for length := min(3, len(args)-index); length > 0; length-- {
				candidate := strings.Join(args[index:index+length], ".")
				if _, ok := idl.LookupCommand(candidate); ok {
					return candidate
				}
			}
			return ""
		}
	}
	return ""
}

func TestNormalizeTextNormalizesNativeEventIDs(t *testing.T) {
	got := normalizeText("{\"last_event_id\":\"evt_0123456789abcdef01234567\",\"event_id\":\"0123456789abcdef0123456789abcdef\",\"stdout\":\"{\\\"event_id\\\":\\\"fedcba9876543210fedcba9876543210\\\"}\\n\"} caused_by_event_id=evt_89abcdef0123456789abcdef `abcdef0123456789abcdef0123456789`", "/unused")
	want := `{"last_event_id":"<EVENT_ID>","event_id":"<EVENT_ID>","stdout":"{\"event_id\":\"<EVENT_ID>\"}\n"} caused_by_event_id=<EVENT_ID> ` + "`<EVENT_ID>`"
	if got != want {
		t.Fatalf("normalized text = %q, want %q", got, want)
	}
}

func TestNormalizeTextNormalizesExecutionLogFields(t *testing.T) {
	got := normalizeText(`{"invocation_id":"0123456789abcdef0123456789abcdef","duration_ms":37}`, "/unused")
	want := `{"invocation_id":"<INVOCATION_ID>","duration_ms":0}`
	if got != want {
		t.Fatalf("normalized text = %q, want %q", got, want)
	}
}

func TestNormalizeTextNormalizesSourceSkillPaths(t *testing.T) {
	want := "<SKILL_ROOT>/skills/techdesign/SKILL.md"
	if got := normalizeText(filepath.Join(repositoryRoot(t), "skills", "techdesign", "SKILL.md"), "/unused"); got != want {
		t.Fatalf("normalized Skill path = %q, want %q", got, want)
	}
}

func TestContractCommandParserCoversTopLevelOperations(t *testing.T) {
	for _, test := range []struct {
		id   string
		args []string
	}{
		{id: "version", args: []string{"version"}},
		{id: "doctor", args: []string{"doctor"}},
		{id: "flow.report.progress", args: []string{"flow", "report", "progress"}},
		{id: "flow.report.result", args: []string{"flow", "report", "result"}},
	} {
		if got := contractCommandID(test.args); got != test.id {
			t.Fatalf("contract command = %q, want %q", got, test.id)
		}
	}
}

func TestMergeEnvironmentScrubsBotmuxBindingByDefault(t *testing.T) {
	t.Setenv("BOTMUX_CHAT_ID", "oc_real_chat_must_not_escape")
	t.Setenv("BOTMUX_SESSION_ID", "real_session_must_not_escape")
	root := t.TempDir()
	environment := mergeEnvironment(root)
	if !slices.Contains(environment, "FANLOOP_DATA_HOME="+filepath.Join(root, ".fanloop-user")) {
		t.Fatal("contract command inherited the real Fanloop data root")
	}
	for _, item := range environment {
		if strings.HasPrefix(item, "BOTMUX_CHAT_ID=") || strings.HasPrefix(item, "BOTMUX_SESSION_ID=") {
			t.Fatalf("contract command inherited Botmux binding: %q", item)
		}
	}
}

func runCase(t *testing.T, binary, casePath string, value contractCase) contractResult {
	t.Helper()
	root := t.TempDir()
	before := filepath.Join(filepath.Dir(casePath), "before")
	if _, err := os.Stat(before); err == nil {
		copyTree(t, before, root)
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	results := make([]commandResult, 0, len(value.Commands))
	for _, step := range value.Commands {
		args := expandAll(step.Args, root)
		command := exec.Command(binary, args...)
		command.Dir = root
		command.Env = mergeEnvironment(root, value.Environment, step.Environment)
		command.Stdin = strings.NewReader(expand(step.Stdin, root))
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		exitCode := 0
		if err := command.Run(); err != nil {
			if exit, ok := err.(*exec.ExitError); ok {
				exitCode = exit.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		results = append(results, commandResult{
			ExitCode: exitCode,
			Stdout:   normalizeText(stdout.String(), root),
			Stderr:   normalizeText(stderr.String(), root),
		})
	}
	return contractResult{SchemaVersion: 1, ID: value.ID, Commands: results, Files: snapshot(t, root)}
}

func loadCase(t *testing.T, path string) contractCase {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var value contractCase
	if err := decoder.Decode(&value); err != nil {
		t.Fatal(err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("unexpected trailing JSON content: %v", err)
	}
	if value.SchemaVersion != 1 || value.ID == "" || len(value.Commands) == 0 {
		t.Fatalf("invalid contract case: %#v", value)
	}
	return value
}

func loadGolden(t *testing.T, path string) contractResult {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("load %s: %v; create it with go test ./tests/contracts -update-contracts", path, err)
	}
	var value contractResult
	if err := json.Unmarshal(content, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func writeGolden(t *testing.T, path string, value contractResult) {
	t.Helper()
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(content, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func copyTree(t *testing.T, source, destination string) {
	t.Helper()
	if err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, info.Mode().Perm())
	}); err != nil {
		t.Fatal(err)
	}
}

func snapshot(t *testing.T, root string) map[string]fileResult {
	t.Helper()
	result := map[string]fileResult{}
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		// trace sync writes the two targets concurrently (internal/trace/sync.go),
		// so the shared fake lark-cli log interleaves non-deterministically. Assert
		// the set of calls, not their racing order.
		if filepath.Base(path) == "lark-calls.log" {
			lines := strings.Split(strings.TrimRight(string(content), "\n"), "\n")
			sort.Strings(lines)
			content = []byte(strings.Join(lines, "\n") + "\n")
		}
		normalized := normalizeText(string(content), root)
		digest := sha256.Sum256([]byte(normalized))
		result[normalizePath(filepath.ToSlash(relative))] = fileResult{
			Mode: fmt.Sprintf("%04o", info.Mode().Perm()), SHA256: hex.EncodeToString(digest[:]), Content: normalized,
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return result
}

func mergeEnvironment(root string, groups ...map[string]string) []string {
	values := map[string]string{}
	for _, item := range os.Environ() {
		key, value, _ := strings.Cut(item, "=")
		values[key] = value
	}
	delete(values, "BOTMUX_CHAT_ID")
	delete(values, "BOTMUX_SESSION_ID")
	values["FANLOOP_DATA_HOME"] = filepath.Join(root, ".fanloop-user")
	values["FANLOOP_PYTHON"] = filepath.Join(root, "missing-python")
	for _, group := range groups {
		for key, value := range group {
			value = expand(value, root)
			values[key] = strings.ReplaceAll(value, "{{PATH}}", values["PATH"])
		}
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	output := make([]string, 0, len(keys))
	for _, key := range keys {
		output = append(output, key+"="+values[key])
	}
	return output
}

func expandAll(values []string, root string) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = expand(value, root)
	}
	return result
}

func expand(value, root string) string { return strings.ReplaceAll(value, "{{ROOT}}", root) }

var (
	timestampPattern    = regexp.MustCompile(`\b[0-9]{4}-[0-9]{2}-[0-9]{2}[T ][0-9]{2}:[0-9]{2}:[0-9]{2}(?:\.[0-9]+)?(?:Z|[+-][0-9]{2}:?[0-9]{2})?\b`)
	eventIDFieldPattern = regexp.MustCompile(
		`("[^"]*event_id"\s*:\s*")(?:(?:legacy-)?[0-9a-f]{24}|evt_[0-9a-f]{24}|[0-9a-f]{32})(")`,
	)
	escapedEventIDFieldPattern = regexp.MustCompile(
		`((?:\\)+"[^"]*event_id(?:\\)+"\s*:\s*(?:\\)+")(?:(?:legacy-)?[0-9a-f]{24}|evt_[0-9a-f]{24}|[0-9a-f]{32})((?:\\)+")`,
	)
	eventIDTextPattern = regexp.MustCompile(
		`(\b(?:event_id|last_event_id|caused_by_event_id)=)(?:(?:legacy-)?[0-9a-f]{24}|evt_[0-9a-f]{24}|[0-9a-f]{32})\b`,
	)
	invocationIDFieldPattern = regexp.MustCompile(`("invocation_id"\s*:\s*")[^"]+(")`)
	durationMSFieldPattern   = regexp.MustCompile(`("duration_ms"\s*:\s*)[0-9]+`)
	nativeEventIDPattern     = regexp.MustCompile("(`)(?:evt_[0-9a-f]{24}|[0-9a-f]{32})(`)")
	nativeCardPattern        = regexp.MustCompile(`\b(?:evt_[0-9a-f]{24}|[0-9a-f]{32})\.json\b`)
	cardPattern              = regexp.MustCompile(`\b[0-9]{8}T[0-9]{6}\.[0-9]{6}[+-][0-9]{4}(?:-[0-9]+)?\.json\b`)
)

func normalizeText(value, root string) string {
	resolved, _ := filepath.EvalSymlinks(root)
	for _, path := range []string{resolved, root} {
		if path != "" {
			value = strings.ReplaceAll(value, path, "<ROOT>")
		}
	}
	_, source, _, _ := runtime.Caller(0)
	repository := filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
	resolvedRepository, _ := filepath.EvalSymlinks(repository)
	for _, path := range []string{resolvedRepository, repository} {
		if path != "" {
			value = strings.ReplaceAll(value, filepath.Join(path, "skills"), "<SKILL_ROOT>/skills")
		}
	}
	value = timestampPattern.ReplaceAllString(value, "<TIMESTAMP>")
	value = eventIDFieldPattern.ReplaceAllString(value, `${1}<EVENT_ID>${2}`)
	value = escapedEventIDFieldPattern.ReplaceAllString(value, `${1}<EVENT_ID>${2}`)
	value = eventIDTextPattern.ReplaceAllString(value, `${1}<EVENT_ID>`)
	value = invocationIDFieldPattern.ReplaceAllString(value, `${1}<INVOCATION_ID>${2}`)
	value = durationMSFieldPattern.ReplaceAllString(value, `${1}0`)
	value = nativeEventIDPattern.ReplaceAllString(value, `${1}<EVENT_ID>${2}`)
	value = nativeCardPattern.ReplaceAllString(value, `<EVENT_ID>.json`)
	return cardPattern.ReplaceAllString(value, "<TIMESTAMP>.json")
}

func normalizePath(value string) string {
	value = nativeCardPattern.ReplaceAllString(value, "<EVENT_ID>.json")
	return cardPattern.ReplaceAllString(value, "<TIMESTAMP>.json")
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate repository")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
