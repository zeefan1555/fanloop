package release

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBoundCurrentReusesManagedLayoutInspection(t *testing.T) {
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	physical := filepath.Join(home, "releases", "1.2.3")
	current := filepath.Join(home, "current")
	for _, directory := range []string{physical, current} {
		if err := os.MkdirAll(directory, 0755); err != nil {
			t.Fatal(err)
		}
	}
	for name, target := range map[string]string{
		"release": "../releases/1.2.3", "bin": "release/bin",
		"entrypoints": "release/entrypoints", "release.json": "release/release.json",
	} {
		if err := os.Symlink(target, filepath.Join(current, name)); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := BoundCurrent(home); err != nil || got != physical {
		t.Fatalf("healthy managed current: %s %v", got, err)
	}
	unknown := filepath.Join(current, "user-file")
	if err := os.WriteFile(unknown, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := BoundCurrent(home); err == nil {
		t.Fatal("managed current with unknown file was accepted")
	}
	if content, err := os.ReadFile(unknown); err != nil || string(content) != "preserve" {
		t.Fatalf("unknown file changed: %v", err)
	}
}

func TestControllerResourceIdentityIncludesModesAndEmptyDirectories(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "resource")
	if err := os.WriteFile(file, []byte("bytes"), 0644); err != nil {
		t.Fatal(err)
	}
	identity := func() string {
		t.Helper()
		digest, modes, err := StrictDirectoryIdentity(root)
		if err != nil {
			t.Fatal(err)
		}
		if !modes["."].IsDir() {
			t.Fatal("root directory is missing from resource identity")
		}
		return ControllerResourcesDigest(digest, modes)
	}
	original := identity()
	if err := os.Chmod(file, 0755); err != nil {
		t.Fatal(err)
	}
	executable := identity()
	if executable == original {
		t.Fatal("file mode change did not change resource identity")
	}
	if err := os.Mkdir(filepath.Join(root, "empty"), 0755); err != nil {
		t.Fatal(err)
	}
	withDirectory := identity()
	if withDirectory == executable {
		t.Fatal("empty directory did not change resource identity")
	}
	if err := os.Chmod(filepath.Join(root, "empty"), 0700); err != nil {
		t.Fatal(err)
	}
	if got := identity(); got != withDirectory {
		t.Fatal("directory permissions entered resource identity")
	}
}

func TestBoundCurrentRejectsUnresolvableNormalizedAlias(t *testing.T) {
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, directory := range []string{filepath.Join(home, "releases", "1.2.3", "bin"), filepath.Join(home, "current")} {
		if err := os.MkdirAll(directory, 0755); err != nil {
			t.Fatal(err)
		}
	}
	for name, target := range map[string]string{
		"release": "../releases/1.2.3", "bin": "missing/../release/bin",
		"entrypoints": "release/entrypoints", "release.json": "release/release.json",
	} {
		if err := os.Symlink(target, filepath.Join(home, "current", name)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(home, "current", "bin")); !os.IsNotExist(err) {
		t.Fatalf("fixture alias unexpectedly resolves: %v", err)
	}
	if _, err := BoundCurrent(home); err == nil {
		t.Fatal("unresolvable alias accepted after lexical normalization")
	}
	if got, err := os.Readlink(filepath.Join(home, "current", "bin")); err != nil || got != "missing/../release/bin" {
		t.Fatalf("rejection changed alias: %s %v", got, err)
	}
}

func TestBoundCurrentRejectsUnresolvableNormalizedPrimaryLink(t *testing.T) {
	for _, test := range []struct {
		name    string
		managed bool
		target  string
	}{
		{"managed", true, "missing/../../releases/1.2.3"},
		{"symlink", false, "missing/../releases/1.2.3"},
	} {
		t.Run(test.name, func(t *testing.T) {
			home, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(home, "releases", "1.2.3"), 0755); err != nil {
				t.Fatal(err)
			}
			link := filepath.Join(home, "current")
			if test.managed {
				if err := os.Mkdir(link, 0755); err != nil {
					t.Fatal(err)
				}
				for name, target := range map[string]string{"bin": "release/bin", "entrypoints": "release/entrypoints", "release.json": "release/release.json"} {
					if err := os.Symlink(target, filepath.Join(link, name)); err != nil {
						t.Fatal(err)
					}
				}
				link = filepath.Join(link, "release")
			}
			if err := os.Symlink(test.target, link); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(link); !os.IsNotExist(err) {
				t.Fatalf("fixture primary link unexpectedly resolves: %v", err)
			}
			if _, err := BoundCurrent(home); err == nil {
				t.Fatal("unresolvable primary link accepted after lexical normalization")
			}
			if actual, err := os.Readlink(link); err != nil || actual != test.target {
				t.Fatalf("rejection changed primary link: %s %v", actual, err)
			}
		})
	}
}

func TestBoundCurrentCanonicalPrimaryLinks(t *testing.T) {
	for _, managed := range []bool{false, true} {
		for _, absolute := range []bool{false, true} {
			name := "symlink/relative"
			if managed {
				name = "managed/relative"
			}
			if absolute {
				name += "/absolute"
			}
			t.Run(name, func(t *testing.T) {
				home, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				physical := filepath.Join(home, "releases", "1.2.3")
				if err := os.MkdirAll(physical, 0755); err != nil {
					t.Fatal(err)
				}
				link := filepath.Join(home, "current")
				if managed {
					if err := os.Mkdir(link, 0755); err != nil {
						t.Fatal(err)
					}
					for name, target := range map[string]string{"bin": "release/bin", "entrypoints": "release/entrypoints", "release.json": "release/release.json"} {
						if err := os.Symlink(target, filepath.Join(link, name)); err != nil {
							t.Fatal(err)
						}
					}
					link = filepath.Join(link, "release")
				}
				target := physical
				if !absolute {
					target, err = filepath.Rel(filepath.Dir(link), physical)
					if err != nil {
						t.Fatal(err)
					}
				}
				if err := os.Symlink(target, link); err != nil {
					t.Fatal(err)
				}
				if got, err := BoundCurrent(home); err != nil || got != physical {
					t.Fatalf("canonical primary link rejected: %s %v", got, err)
				}
			})
		}
	}
}
