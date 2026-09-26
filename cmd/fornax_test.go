package cmd

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestFornaxPublishedConfigurationUpdatesExplicitlyAndFailsClosed(t *testing.T) {
	repository, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	dataRoot := t.TempDir()
	if err := os.CopyFS(filepath.Join(dataRoot, "current", "workflows"), os.DirFS(filepath.Join(repository, "internal", "workflow", "testdata"))); err != nil {
		t.Fatal(err)
	}
	const workflowID = "linear-runtime"
	const secondWorkflowID = "runtime-controls"
	makePackage := func(id, marker string) []byte {
		var buffer bytes.Buffer
		writer := zip.NewWriter(&buffer)
		entry, _ := writer.Create("wrapper/SKILL.md")
		_, _ = entry.Write([]byte("configuration"))
		source := filepath.Join(repository, "internal", "workflow", "testdata", id)
		if err := filepath.WalkDir(source, func(path string, item fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if item.IsDir() {
				return nil
			}
			relative, _ := filepath.Rel(source, path)
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			entry, err := writer.Create("wrapper/workflows/" + id + "/" + filepath.ToSlash(relative))
			if err != nil {
				return err
			}
			_, err = entry.Write(content)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		markerSkill := "test-guide"
		if id == secondWorkflowID {
			markerSkill = "common-control-guide"
		}
		entry, _ = writer.Create("wrapper/workflows/" + id + "/skills/" + markerSkill + "/" + marker + ".txt")
		_, _ = entry.Write([]byte(marker))
		_ = writer.Close()
		return buffer.Bytes()
	}
	archives := map[string][]byte{
		"main":      makePackage(workflowID, "first"),
		"secondary": makePackage(secondWorkflowID, "first"),
	}
	var mutex sync.Mutex
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		defer mutex.Unlock()
		_, _ = w.Write(archives[strings.TrimPrefix(r.URL.Path, "/")])
	}))
	defer server.Close()
	previousTransport := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	defer func() { http.DefaultTransport = previousTransport }()
	testRoot := t.TempDir()
	settings := map[string]any{"workspace_id": "123", "workflows": []any{
		map[string]string{"id": workflowID, "skill_key": "test.workflow", "skill_id": "456"},
		map[string]string{"id": secondWorkflowID, "skill_key": "test.pstack", "skill_id": "789"},
	}}
	writeJSON := func(path string, value any) {
		t.Helper()
		content, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0600); err != nil {
			t.Fatal(err)
		}
	}
	writeJSON(filepath.Join(dataRoot, "fornax.json"), settings)
	statePath := filepath.Join(testRoot, "platform.json")
	logPath := filepath.Join(testRoot, "calls.jsonl")
	versions := map[string]string{"456": "1.0.0", "789": "1.0.0"}
	urls := map[string]string{"456": server.URL + "/main", "789": server.URL + "/secondary"}
	state := map[string]any{"versions": versions, "urls": urls, "workspace": "123", "key": "test.workflow", "fail": false, "ambiguous": false}
	writeJSON(statePath, state)
	fake := filepath.Join(testRoot, "fornax-cli")
	script := `#!/usr/bin/env python3
import json,os,sys,pathlib
args=sys.argv[1:]
s=json.loads(pathlib.Path(os.environ['TEST_FORNAX_STATE']).read_text())
with open(os.environ['TEST_FORNAX_LOG'],'a') as f:f.write(json.dumps(args)+'\n')
if s['fail']:sys.exit(1)
if args[:2]==['skill','list-releases']:
 skill_id=args[args.index('--skill-id')+1]
 result={'code':0,'items':[{'skill_id':skill_id,'env':'Online','feature':'prod','release_status':1,'version':'0.9.0','updated_at':'1'},{'skill_id':skill_id,'env':'Online','feature':'prod','release_status':1,'version':'0.8.0','updated_at':'1'},{'skill_id':skill_id,'env':'Online','feature':'prod','release_status':1,'version':s['versions'][skill_id],'updated_at':'2'}]}
 if s['ambiguous']:result['items'].append(dict(result['items'][2],version='9.0.0'))
elif args[:2]==['skill','batch-get-by-key']:
 version=args[args.index('--version')+1]
 key=args[args.index('--skill-key')+1]
 skill_id={'test.workflow':'456','test.pstack':'789'}[key]
 returned_key=key if key!='test.workflow' or s['key']=='test.workflow' else s['key']
 result={'code':0,'items':[{'skill':{'id':skill_id,'skill_key':returned_key,'workspace_id':s['workspace'],'skill_commit':{'commit_info':{'version':s.get('returned_version',version)},'skill_detail':{'tos_url':s['urls'][skill_id]}}}}]}
else:sys.exit(2)
pathlib.Path(args[args.index('-o')+1],'result.json').write_text(json.dumps(result))
`
	if err := os.WriteFile(fake, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}

	t.Setenv("FANLOOP_CONFIG_SOURCE", "")
	t.Setenv("FANLOOP_DATA_HOME", dataRoot)
	t.Setenv("PATH", testRoot+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("TEST_FORNAX_STATE", statePath)
	t.Setenv("TEST_FORNAX_LOG", logPath)
	t.Setenv("BOTMUX_CHAT_ID", "")
	t.Setenv("BOTMUX_SESSION_ID", "")
	call := func(args ...string) ([]byte, error) {
		var stdout, stderr bytes.Buffer
		code := Execute(context.Background(), args, strings.NewReader(""), &stdout, &stderr)
		output := append(stdout.Bytes(), stderr.Bytes()...)
		if code != 0 {
			return output, fmt.Errorf("exit %d", code)
		}
		return output, nil
	}
	root := t.TempDir()
	if output, err := call("flow", "init", "--root", root, "--workflow", workflowID, "--title", "Fornax test"); err != nil {
		t.Fatalf("init: %v %s", err, output)
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatal("ordinary flow init contacted Fornax")
	}
	if output, err := call("update"); err != nil {
		t.Fatalf("initial update: %v %s", err, output)
	}
	liveRoot := filepath.Join(dataRoot, "current", "workflows")
	live := filepath.Join(liveRoot, workflowID)
	secondLive := filepath.Join(liveRoot, secondWorkflowID)
	if _, err := os.Stat(filepath.Join(live, "skills/test-guide/first.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(secondLive, "skills/common-control-guide/first.txt")); err != nil {
		t.Fatal(err)
	}
	calls, err := os.ReadFile(logPath)
	if err != nil || !bytes.Contains(calls, []byte(`"--version", "1.0.0"`)) {
		t.Fatalf("did not pin published version: %s", calls)
	}
	mutex.Lock()
	archives["main"] = makePackage(workflowID, "second")
	archives["secondary"] = makePackage(secondWorkflowID, "second")
	mutex.Unlock()
	versions["456"] = "1.0.1"
	versions["789"] = "1.0.1"
	writeJSON(statePath, state)
	if output, err := call("update"); err != nil {
		t.Fatalf("refresh: %v %s", err, output)
	}
	if _, err := os.Stat(filepath.Join(live, "skills/test-guide/first.txt")); !os.IsNotExist(err) {
		t.Fatal("removed resource survived refresh")
	}
	if _, err := os.Stat(filepath.Join(secondLive, "skills/common-control-guide/first.txt")); !os.IsNotExist(err) {
		t.Fatal("removed resource in non-selected Workflow survived refresh")
	}
	if _, err := os.Stat(filepath.Join(secondLive, "skills/common-control-guide/second.txt")); err != nil {
		t.Fatal("non-selected Workflow was not refreshed with the complete set")
	}
	before := treeDigest(t, liveRoot)
	stateBefore, err := os.ReadFile(filepath.Join(root, ".fanloop/flow/state.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		key   string
		value any
	}{{"fail", true}, {"ambiguous", true}, {"workspace", "999"}, {"key", "other"}, {"returned_version", "9.0.0"}} {
		previous := state[test.key]
		state[test.key] = test.value
		writeJSON(statePath, state)
		if output, err := call("update"); err == nil {
			t.Fatalf("accepted %s failure: %s", test.key, output)
		}
		if treeDigest(t, liveRoot) != before {
			t.Fatal("failed refresh changed live configuration")
		}
		after, err := os.ReadFile(filepath.Join(root, ".fanloop/flow/state.json"))
		if err != nil || !bytes.Equal(stateBefore, after) {
			t.Fatal("failed refresh changed Requirement")
		}
		if previous == nil {
			delete(state, test.key)
		} else {
			state[test.key] = previous
		}
	}

	writeJSON(statePath, state)
	state["fail"] = true
	writeJSON(statePath, state)
	callsBefore, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	failures := make(chan string, 8)
	for index := 0; index < 8; index++ {
		group.Add(1)
		go func() {
			defer group.Done()
			output, err := call("flow", "status", "--root", root)
			if err != nil {
				failures <- string(output)
			}
		}()
	}
	group.Wait()
	close(failures)
	for failure := range failures {
		t.Fatal(failure)
	}
	callsAfter, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(callsBefore, callsAfter) {
		t.Fatal("ordinary concurrent reads contacted Fornax")
	}
	if treeDigest(t, liveRoot) != before {
		t.Fatal("concurrent reads changed complete configuration")
	}
	source := filepath.Join(testRoot, "nonexistent-release")
	if output, err := call("__install", "--source", source, "--data-root", dataRoot, "--codex-skills-root", filepath.Join(testRoot, "codex"), "--agent-skills-root", filepath.Join(testRoot, "agent"), "--trae-skills-root", filepath.Join(testRoot, "trae"), "--claude-skills-root", filepath.Join(testRoot, "claude")); err == nil {
		t.Fatalf("invalid release accepted: %s", output)
	}
	if treeDigest(t, liveRoot) != before {
		t.Fatal("invalid release install changed live tree")
	}
	for _, args := range [][]string{{"--help"}, {"version"}, {"flow", "status", "--help"}} {
		if output, err := call(args...); err != nil {
			t.Fatalf("offline help/version: %v %s", err, output)
		}
	}
	for _, args := range [][]string{{"flow", "status"}, {"flow", "status", "--root", root, "--input", "{"}} {
		if _, err := call(args...); err == nil {
			t.Fatal("invalid invocation accepted")
		}
	}
	callsAfter, _ = os.ReadFile(logPath)
	if !bytes.Equal(callsBefore, callsAfter) {
		t.Fatal("help/version or invalid invocation contacted Fornax")
	}
}

func treeDigest(t *testing.T, root string) string {
	t.Helper()
	digest := sha256.New()
	if err := filepath.WalkDir(root, func(path string, item fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if item.IsDir() {
			return nil
		}
		relative, _ := filepath.Rel(root, path)
		_, _ = digest.Write([]byte(relative))
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, _ = digest.Write(content)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", digest.Sum(nil))
}
