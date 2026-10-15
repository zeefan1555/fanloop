package distribution

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeefan1555/fanloop/internal/release"
	"github.com/zeefan1555/fanloop/internal/state"
)

func TestSelectPlatform(t *testing.T) {
	cases := []struct {
		goos, goarch string
		want         Platform
		ok           bool
	}{
		{"linux", "amd64", Platform{OS: "linux", Arch: "amd64"}, true},
		{"linux", "x64", Platform{OS: "linux", Arch: "amd64"}, true},
		{"linux", "arm64", Platform{OS: "linux", Arch: "arm64"}, true},
		{"darwin", "amd64", Platform{OS: "darwin", Arch: "amd64"}, true},
		{"darwin", "arm64", Platform{OS: "darwin", Arch: "arm64"}, true},
		{"windows", "amd64", Platform{}, false},
		{"linux", "386", Platform{}, false},
	}
	for _, tc := range cases {
		got, err := SelectPlatform(tc.goos, tc.goarch)
		if tc.ok != (err == nil) || got != tc.want {
			t.Errorf("SelectPlatform(%s, %s) = %+v, %v", tc.goos, tc.goarch, got, err)
		}
	}
	name, err := AssetName("1.2.3", Platform{OS: "linux", Arch: "amd64"})
	if err != nil || name != "fanloop-1.2.3-linux-amd64.tar.gz" {
		t.Fatalf("asset name = %s, %v", name, err)
	}
}

func TestVerifyChecksumRejectsMismatch(t *testing.T) {
	body := []byte("fanloop")
	sum := sha256.Sum256(body)
	good := hex.EncodeToString(sum[:])
	name := "fanloop-1.2.3-linux-amd64.tar.gz"
	if _, err := VerifyChecksum(good+"  "+name+"\n", name, body); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyChecksum(strings.Repeat("0", 64)+"  "+name+"\n", name, body); err == nil || !strings.Contains(err.Error(), "sha256 mismatch") {
		t.Fatalf("mismatch error = %v", err)
	}
	if _, err := VerifyChecksum(good+"  other.tar.gz\n", name, body); err == nil || !strings.Contains(err.Error(), "no entry") {
		t.Fatalf("missing entry error = %v", err)
	}
}

func TestSwitchCurrentIsAtomic(t *testing.T) {
	root := t.TempDir()
	install := func(version string) {
		t.Helper()
		dir := filepath.Join(root, "releases", version, "bin")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "fanloop"), []byte(version), 0o755); err != nil {
			t.Fatal(err)
		}
		changed, err := SwitchCurrent(root, version)
		if err != nil || !changed {
			t.Fatalf("switch %s: changed=%v err=%v", version, changed, err)
		}
	}
	install("1.2.3")
	install("1.2.4")
	target, err := os.Readlink(filepath.Join(root, "current"))
	if err != nil || filepath.Clean(target) != filepath.Join("releases", "1.2.4") {
		t.Fatalf("current -> %s (%v)", target, err)
	}
	if body, err := os.ReadFile(filepath.Join(root, "releases", "1.2.3", "bin", "fanloop")); err != nil || string(body) != "1.2.3" {
		t.Fatalf("previous release = %q, %v", body, err)
	}
	changed, err := SwitchCurrent(root, "1.2.4")
	if err != nil || changed {
		t.Fatalf("repeat switch changed=%v err=%v", changed, err)
	}
	if err := os.Remove(filepath.Join(root, "current")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "current"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "releases", "9.9.9"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := SwitchCurrent(root, "9.9.9"); err == nil || !strings.Contains(err.Error(), "not a symlink") {
		t.Fatalf("directory current error = %v", err)
	}
	info, err := os.Lstat(filepath.Join(root, "current"))
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("directory current was replaced: %v", err)
	}
}

func TestUpdateVerifiesChecksumBeforeSwitch(t *testing.T) {
	version := "1.2.3"
	platform := Platform{OS: "linux", Arch: "amd64"}
	name, err := AssetName(version, platform)
	if err != nil {
		t.Fatal(err)
	}
	archive := validArchive(t, version, "#!/bin/sh\necho 1.2.3\n", "# workflow\n", "# author\n")
	archiveSum := sha256.Sum256(archive)
	archiveHash := hex.EncodeToString(archiveSum[:])
	index := releaseIndex{
		SchemaVersion: 1, Version: version, Tag: "v" + version, Repository: Repository,
		Assets: []asset{{Name: name, OS: "linux", Arch: "amd64", SHA256: archiveHash}},
	}
	manifest, err := json.Marshal(index)
	if err != nil {
		t.Fatal(err)
	}
	manifest = append(manifest, '\n')
	manifestSum := sha256.Sum256(manifest)
	manifestHash := hex.EncodeToString(manifestSum[:])
	goodChecksums := archiveHash + "  " + name + "\n" + manifestHash + "  release.json\n"
	badChecksums := strings.Repeat("ab", 32) + "  " + name + "\n" + manifestHash + "  release.json\n"

	root := t.TempDir()
	skills := testSkillRoots(t, root)
	config := filepath.Join(root, "config")
	if err := os.MkdirAll(config, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(config, "marker"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	serve := func(checksums string) *httptest.Server {
		mux := http.NewServeMux()
		var server *httptest.Server
		mux.HandleFunc("/latest", func(w http.ResponseWriter, _ *http.Request) {
			payload := githubRelease{TagName: "v" + version, Assets: []struct {
				Name               string `json:"name"`
				BrowserDownloadURL string `json:"browser_download_url"`
			}{
				{Name: "checksums.txt", BrowserDownloadURL: server.URL + "/checksums.txt"},
				{Name: "release.json", BrowserDownloadURL: server.URL + "/release.json"},
				{Name: name, BrowserDownloadURL: server.URL + "/" + name},
			}}
			_ = json.NewEncoder(w).Encode(payload)
		})
		mux.HandleFunc("/checksums.txt", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(checksums)) })
		mux.HandleFunc("/release.json", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(manifest) })
		mux.HandleFunc("/"+name, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(archive) })
		server = httptest.NewServer(mux)
		return server
	}

	bad := serve(badChecksums)
	defer bad.Close()
	if _, err := Update(context.Background(), Options{DataRoot: root, Client: bad.Client(), LatestURL: bad.URL + "/latest", Platform: platform, SkillRoots: skills}); err == nil || !strings.Contains(err.Error(), "sha256 mismatch") {
		t.Fatalf("bad checksum error = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(root, "current")); !os.IsNotExist(err) {
		t.Fatal("checksum failure switched current")
	}
	if body, err := os.ReadFile(filepath.Join(config, "marker")); err != nil || string(body) != "keep" {
		t.Fatalf("update touched configuration: %q %v", body, err)
	}

	good := serve(goodChecksums)
	defer good.Close()
	result, err := Update(context.Background(), Options{DataRoot: root, Client: good.Client(), LatestURL: good.URL + "/latest", Platform: platform, SkillRoots: skills})
	if err != nil || result.Version != version || result.Unchanged {
		t.Fatalf("update = %+v, %v", result, err)
	}
	target, err := os.Readlink(filepath.Join(root, "current"))
	if err != nil || filepath.Clean(target) != filepath.Join("releases", version) {
		t.Fatalf("current -> %s (%v)", target, err)
	}
	for _, skillRoot := range skills {
		link, linkErr := os.Readlink(filepath.Join(skillRoot, "fanloop-workflow"))
		if linkErr != nil || filepath.Clean(link) != filepath.Join(root, "current", "entrypoints", "fanloop-workflow") {
			t.Fatalf("workflow skill link = %s (%v)", link, linkErr)
		}
	}
	again, err := Update(context.Background(), Options{DataRoot: root, Client: good.Client(), LatestURL: good.URL + "/latest", Platform: platform, SkillRoots: skills})
	if err != nil || !again.Unchanged || again.Version != version {
		t.Fatalf("second update = %+v, %v", again, err)
	}
}

func testArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	compressor := gzip.NewWriter(&buf)
	archive := tar.NewWriter(compressor)
	for name, body := range files {
		mode := int64(0o644)
		if strings.HasSuffix(name, "/fanloop") || name == "bin/fanloop" {
			mode = 0o755
		}
		header := &tar.Header{Name: name, Mode: mode, Size: int64(len(body)), Typeflag: tar.TypeReg}
		if err := archive.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := archive.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressor.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func testSkillRoots(t *testing.T, root string) []string {
	t.Helper()
	roots := []string{}
	for _, name := range []string{"codex", "agents", "trae", "claude"} {
		dir := filepath.Join(root, "skills", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		roots = append(roots, dir)
	}
	return roots
}

func validArchive(t *testing.T, version, binary, workflow, author string) []byte {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"bin/fanloop":                                  binary,
		"entrypoints/fanloop-workflow/SKILL.md":        workflow,
		"entrypoints/create-fanloop-workflow/SKILL.md": author,
	}
	for name, body := range files {
		target := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		mode := os.FileMode(0o644)
		if name == "bin/fanloop" {
			mode = 0o755
		}
		if err := os.WriteFile(target, []byte(body), mode); err != nil {
			t.Fatal(err)
		}
	}
	binaryDigest, err := release.FileDigest(filepath.Join(root, "bin", "fanloop"))
	if err != nil {
		t.Fatal(err)
	}
	skills := []map[string]any{}
	for _, exposed := range release.ExposedSkills() {
		digest, err := release.DirectoryDigest(filepath.Join(root, filepath.FromSlash(exposed.Path)))
		if err != nil {
			t.Fatal(err)
		}
		body := workflow
		if exposed.Name == release.AuthoringSkillName {
			body = author
		}
		files[exposed.Path+"/SKILL.md"] = body
		skills = append(skills, map[string]any{
			"name": exposed.Name, "version": version, "path": exposed.Path, "sha256": digest,
		})
	}
	manifest := map[string]any{
		"schema_version": 4, "release_version": version,
		"cli":          map[string]any{"version": version, "binary_sha256": binaryDigest},
		"state_schema": map[string]any{"read_versions": []int{state.CurrentStateSchemaVersion}, "write_version": state.CurrentStateSchemaVersion},
		"skills":       skills,
	}
	content, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	files["release.json"] = string(content) + "\n"
	return testArchive(t, files)
}

func serveRelease(t *testing.T, version string, archive []byte, hits *int) *httptest.Server {
	t.Helper()
	platform := Platform{OS: "linux", Arch: "amd64"}
	name, err := AssetName(version, platform)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(archive)
	archiveHash := hex.EncodeToString(sum[:])
	index := releaseIndex{
		SchemaVersion: 1, Version: version, Tag: "v" + version, Repository: Repository,
		Assets: []asset{{Name: name, OS: "linux", Arch: "amd64", SHA256: archiveHash}},
	}
	manifest, err := json.Marshal(index)
	if err != nil {
		t.Fatal(err)
	}
	manifest = append(manifest, '\n')
	manifestSum := sha256.Sum256(manifest)
	checksums := archiveHash + "  " + name + "\n" + hex.EncodeToString(manifestSum[:]) + "  release.json\n"
	mux := http.NewServeMux()
	var server *httptest.Server
	mux.HandleFunc("/latest", func(w http.ResponseWriter, _ *http.Request) {
		payload := githubRelease{TagName: "v" + version, Assets: []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		}{
			{Name: "checksums.txt", BrowserDownloadURL: server.URL + "/checksums.txt"},
			{Name: "release.json", BrowserDownloadURL: server.URL + "/release.json"},
			{Name: name, BrowserDownloadURL: server.URL + "/" + name},
		}}
		_ = json.NewEncoder(w).Encode(payload)
	})
	mux.HandleFunc("/checksums.txt", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(checksums)) })
	mux.HandleFunc("/release.json", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(manifest) })
	mux.HandleFunc("/"+name, func(w http.ResponseWriter, _ *http.Request) {
		if hits != nil {
			*hits++
		}
		_, _ = w.Write(archive)
	})
	server = httptest.NewServer(mux)
	return server
}

func TestUpdateRejectsIncompleteManifestBeforeSwitch(t *testing.T) {
	version := "1.2.3"
	archive := testArchive(t, map[string]string{
		"bin/fanloop":                                  "#!/bin/sh\n",
		"release.json":                                 "{\"schema_version\":4,\"release_version\":\"1.2.3\"}\n",
		"entrypoints/fanloop-workflow/SKILL.md":        "# workflow\n",
		"entrypoints/create-fanloop-workflow/SKILL.md": "# author\n",
	})
	root := t.TempDir()
	server := serveRelease(t, version, archive, nil)
	defer server.Close()
	_, err := Update(context.Background(), Options{
		DataRoot: root, Client: server.Client(), LatestURL: server.URL + "/latest",
		Platform: Platform{OS: "linux", Arch: "amd64"}, SkillRoots: testSkillRoots(t, root),
	})
	if err == nil {
		t.Fatal("incomplete manifest was accepted")
	}
	if _, statErr := os.Lstat(filepath.Join(root, "current")); !os.IsNotExist(statErr) {
		t.Fatal("incomplete manifest switched current")
	}
	if _, statErr := os.Lstat(filepath.Join(root, "releases", version)); !os.IsNotExist(statErr) {
		t.Fatal("incomplete manifest published the release directory")
	}
}

func TestUpdateRepairsTamperedReleaseContents(t *testing.T) {
	version := "1.2.3"
	archive := validArchive(t, version, "#!/bin/sh\necho 1.2.3\n", "# workflow\n", "# author\n")
	root := t.TempDir()
	skills := testSkillRoots(t, root)
	server := serveRelease(t, version, archive, nil)
	defer server.Close()
	options := Options{DataRoot: root, Client: server.Client(), LatestURL: server.URL + "/latest", Platform: Platform{OS: "linux", Arch: "amd64"}, SkillRoots: skills}
	if _, err := Update(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	skill := filepath.Join(root, "releases", version, "entrypoints", "fanloop-workflow", "SKILL.md")
	if err := os.WriteFile(skill, []byte("tampered\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "releases", version, "extra.txt"), []byte("extra\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Update(context.Background(), options)
	if err != nil || !result.Unchanged || result.Version != version {
		t.Fatalf("repair = %+v, %v", result, err)
	}
	body, err := os.ReadFile(skill)
	if err != nil || string(body) != "# workflow\n" {
		t.Fatalf("skill = %q, %v", body, err)
	}
	if _, err := os.Lstat(filepath.Join(root, "releases", version, "extra.txt")); !os.IsNotExist(err) {
		t.Fatal("tampered extra file remained")
	}
}

func TestUpdateDoesNotDowngradeNewerInstall(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "releases", "1.2.4"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("releases", "1.2.4"), filepath.Join(root, "current")); err != nil {
		t.Fatal(err)
	}
	hits := 0
	archive := validArchive(t, "1.2.3", "old\n", "# workflow\n", "# author\n")
	server := serveRelease(t, "1.2.3", archive, &hits)
	defer server.Close()
	result, err := Update(context.Background(), Options{
		DataRoot: root, Client: server.Client(), LatestURL: server.URL + "/latest",
		Platform: Platform{OS: "linux", Arch: "amd64"}, SkillRoots: testSkillRoots(t, root),
	})
	if err != nil || !result.Unchanged || result.Version != "1.2.4" {
		t.Fatalf("downgrade result = %+v, %v", result, err)
	}
	if hits != 0 {
		t.Fatalf("older archive downloads = %d", hits)
	}
	target, err := os.Readlink(filepath.Join(root, "current"))
	if err != nil || filepath.Clean(target) != filepath.Join("releases", "1.2.4") {
		t.Fatalf("current -> %s (%v)", target, err)
	}
}

func TestUpdateRefusesNonVersionCurrent(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "releases", "local-dev"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("releases", "local-dev"), filepath.Join(root, "current")); err != nil {
		t.Fatal(err)
	}
	hits := 0
	server := serveRelease(t, "1.2.3", validArchive(t, "1.2.3", "bin\n", "# workflow\n", "# author\n"), &hits)
	defer server.Close()
	_, err := Update(context.Background(), Options{
		DataRoot: root, Client: server.Client(), LatestURL: server.URL + "/latest",
		Platform: Platform{OS: "linux", Arch: "amd64"}, SkillRoots: testSkillRoots(t, root),
	})
	if err == nil || !strings.Contains(err.Error(), "refusing to replace current link") {
		t.Fatalf("non-version current error = %v", err)
	}
	if hits != 0 {
		t.Fatalf("archive downloads = %d", hits)
	}
	target, readErr := os.Readlink(filepath.Join(root, "current"))
	if readErr != nil || filepath.Clean(target) != filepath.Join("releases", "local-dev") {
		t.Fatalf("current -> %s (%v)", target, readErr)
	}
}

func TestUpdateSkipsPaxHeadersAndAppleDouble(t *testing.T) {
	version := "1.2.3"
	binary := "#!/bin/sh\necho 1.2.3\n"
	workflow := "# workflow\n"
	author := "# author\n"
	root := t.TempDir()
	files := map[string]string{
		"bin/fanloop":                                  binary,
		"entrypoints/fanloop-workflow/SKILL.md":        workflow,
		"entrypoints/create-fanloop-workflow/SKILL.md": author,
	}
	for name, body := range files {
		target := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	binaryDigest, err := release.FileDigest(filepath.Join(root, "bin", "fanloop"))
	if err != nil {
		t.Fatal(err)
	}
	skills := []map[string]any{}
	for _, exposed := range release.ExposedSkills() {
		digest, err := release.DirectoryDigest(filepath.Join(root, filepath.FromSlash(exposed.Path)))
		if err != nil {
			t.Fatal(err)
		}
		skills = append(skills, map[string]any{
			"name": exposed.Name, "version": version, "path": exposed.Path, "sha256": digest,
		})
	}
	manifest, err := json.Marshal(map[string]any{
		"schema_version": 4, "release_version": version,
		"cli":          map[string]any{"version": version, "binary_sha256": binaryDigest},
		"state_schema": map[string]any{"read_versions": []int{state.CurrentStateSchemaVersion}, "write_version": state.CurrentStateSchemaVersion},
		"skills":       skills,
	})
	if err != nil {
		t.Fatal(err)
	}
	files["release.json"] = string(manifest) + "\n"
	var buf bytes.Buffer
	compressor := gzip.NewWriter(&buf)
	archive := tar.NewWriter(compressor)
	for name, body := range files {
		mode := int64(0o644)
		if name == "bin/fanloop" {
			mode = 0o755
		}
		header := &tar.Header{
			Name: name, Mode: mode, Size: int64(len(body)), Typeflag: tar.TypeReg,
			PAXRecords: map[string]string{"atime": "0"},
		}
		if err := archive.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := archive.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	meta := []byte("appledouble")
	metaHeader := &tar.Header{
		Name: "entrypoints/fanloop-workflow/._SKILL.md", Mode: 0o644, Size: int64(len(meta)), Typeflag: tar.TypeReg,
	}
	if err := archive.WriteHeader(metaHeader); err != nil {
		t.Fatal(err)
	}
	if _, err := archive.Write(meta); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil || compressor.Close() != nil {
		t.Fatal("close archive")
	}
	dataRoot := t.TempDir()
	server := serveRelease(t, version, buf.Bytes(), nil)
	defer server.Close()
	if _, err := Update(context.Background(), Options{
		DataRoot: dataRoot, Client: server.Client(), LatestURL: server.URL + "/latest",
		Platform: Platform{OS: "linux", Arch: "amd64"}, SkillRoots: testSkillRoots(t, dataRoot),
	}); err != nil {
		t.Fatal(err)
	}
	installed := filepath.Join(dataRoot, "releases", version, "entrypoints", "fanloop-workflow")
	if _, err := os.Lstat(filepath.Join(installed, "._SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("AppleDouble file was installed: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(installed, "SKILL.md"))
	if err != nil || string(body) != workflow {
		t.Fatalf("skill = %q, %v", body, err)
	}
}

func TestUpdateRejectsNonRegularArchiveEntries(t *testing.T) {
	var buf bytes.Buffer
	compressor := gzip.NewWriter(&buf)
	archive := tar.NewWriter(compressor)
	header := &tar.Header{Name: "bin/fanloop", Mode: int64(0o777), Typeflag: tar.TypeSymlink, Linkname: "elsewhere"}
	if err := archive.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil || compressor.Close() != nil {
		t.Fatal("close archive")
	}
	root := t.TempDir()
	server := serveRelease(t, "1.2.3", buf.Bytes(), nil)
	defer server.Close()
	_, err := Update(context.Background(), Options{
		DataRoot: root, Client: server.Client(), LatestURL: server.URL + "/latest",
		Platform: Platform{OS: "linux", Arch: "amd64"}, SkillRoots: testSkillRoots(t, root),
	})
	if err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("symlink archive error = %v", err)
	}
	if _, statErr := os.Lstat(filepath.Join(root, "current")); !os.IsNotExist(statErr) {
		t.Fatal("symlink archive switched current")
	}
}

func TestUpdateExplainsPrivateRepository404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.NotFound(w, nil)
	}))
	defer server.Close()
	root := t.TempDir()
	_, err := Update(context.Background(), Options{DataRoot: root, Client: server.Client(), LatestURL: server.URL + "/latest", Platform: Platform{OS: "linux", Arch: "amd64"}, SkillRoots: testSkillRoots(t, root)})
	if err == nil || !strings.Contains(err.Error(), "HTTP 404") || !strings.Contains(err.Error(), "public repository or authentication") {
		t.Fatalf("404 error = %v", err)
	}
}

func TestDownloadRejectsBodiesOverTheLimit(t *testing.T) {
	previous := downloadLimitBytes
	downloadLimitBytes = 8
	t.Cleanup(func() { downloadLimitBytes = previous })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("a"), 32))
	}))
	defer server.Close()
	root := t.TempDir()
	_, err := Update(context.Background(), Options{DataRoot: root, Client: server.Client(), LatestURL: server.URL, Platform: Platform{OS: "linux", Arch: "amd64"}, SkillRoots: testSkillRoots(t, root)})
	if err == nil || !strings.Contains(err.Error(), "exceeds 8 bytes") {
		t.Fatalf("limit error = %v", err)
	}
}
