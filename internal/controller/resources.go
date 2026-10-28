package controller

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/zeefan1555/fanloop/internal/configuration"
	"github.com/zeefan1555/fanloop/internal/idl/storageidl"
	"github.com/zeefan1555/fanloop/internal/release"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

type Resources struct {
	Binding       *Binding
	Release       release.Manifest
	Configuration configuration.Manifest
	ReleaseModes  map[string]os.FileMode
	Root          string
}

func Plan(root, sourceRelease, sourceConfig, previousPin string, operation storageidl.ControllerBindingOperation, live bool) (*Resources, error) {
	if err := release.RealPath(root, true); err != nil {
		return nil, err
	}
	manifest, err := release.LoadBound(sourceRelease)
	if err != nil {
		return nil, err
	}
	if err := release.RealPath(filepath.Join(sourceConfig, configuration.Category), true); err != nil {
		return nil, err
	}
	unlock, err := workflow.LockSource(filepath.Join(sourceConfig, configuration.Category), false)
	if err != nil {
		return nil, err
	}
	defer unlock()
	config, err := configuration.InspectBinding(sourceConfig)
	if err != nil {
		return nil, err
	}
	digest, err := release.FileDigest(filepath.Join(sourceRelease, "release.json"))
	if err != nil {
		return nil, err
	}
	treeDigest, modes, err := release.StrictDirectoryIdentity(sourceRelease)
	if err != nil {
		return nil, err
	}
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return nil, err
	}
	mode := storageidl.ControllerConfigurationMode_author_frozen
	pin := filepath.Join(root, Home, "config", "author", strings.TrimPrefix(config.SHA256, "sha256:"))
	configDigest := &config.SHA256
	if configuration.IsSnapshotSource(sourceConfig) {
		mode = storageidl.ControllerConfigurationMode_snapshot
		pin = filepath.Join(root, Home, "config", "snapshots", filepath.Base(sourceConfig))
	} else if live {
		mode = storageidl.ControllerConfigurationMode_author_live
		pin = sourceConfig
		configDigest = nil
	}
	binding := &Binding{SchemaVersion: 1, TransactionId: hex.EncodeToString(id), Phase: storageidl.ControllerBindingPhase_prepared,
		ConfigurationMode: mode, SourceReleasePath: sourceRelease, ControllerReleasePath: "releases/" + manifest.ReleaseVersion,
		ReleaseManifestSha256: digest, BinarySha256: manifest.Cli.BinarySha256, PreviousConfigurationPin: previousPin,
		ReleaseResourcesSha256: release.ControllerResourcesDigest(treeDigest, modes),
		ConfigurationPin:       pin, ConfigurationManifestSha256: configDigest, PreparedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Operation: operation, ConfigurationSourcePath: sourceConfig}
	return &Resources{Binding: binding, Release: manifest, Configuration: config, ReleaseModes: modes, Root: root}, nil
}

func Resume(root string, binding *Binding) (*Resources, error) {
	resources, err := Plan(root, binding.SourceReleasePath, binding.ConfigurationSourcePath, binding.PreviousConfigurationPin, binding.Operation, binding.ConfigurationMode == storageidl.ControllerConfigurationMode_author_live)
	if err != nil {
		return nil, err
	}
	candidate := resources.Binding
	if candidate.ControllerReleasePath != binding.ControllerReleasePath || candidate.ReleaseManifestSha256 != binding.ReleaseManifestSha256 || candidate.BinarySha256 != binding.BinarySha256 || candidate.ReleaseResourcesSha256 != binding.ReleaseResourcesSha256 || candidate.ConfigurationPin != binding.ConfigurationPin || candidate.GetConfigurationManifestSha256() != binding.GetConfigurationManifestSha256() {
		return nil, fmt.Errorf("prepared resource identity changed")
	}
	resources.Binding = binding
	return resources, nil
}

func (r *Resources) CheckSource() error {
	manifest, err := release.LoadBound(r.Binding.SourceReleasePath)
	if err != nil {
		return err
	}
	digest, modes, err := release.StrictDirectoryIdentity(r.Binding.SourceReleasePath)
	if err != nil {
		return err
	}
	config, err := configuration.InspectBinding(r.Binding.ConfigurationSourcePath)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(manifest, r.Release) || release.ControllerResourcesDigest(digest, modes) != r.Binding.ReleaseResourcesSha256 || !reflect.DeepEqual(config, r.Configuration) {
		return fmt.Errorf("controller source changed during protection")
	}
	return nil
}

func (r *Resources) StagedConfiguration() string {
	if r.Binding.ConfigurationMode == storageidl.ControllerConfigurationMode_author_live {
		return r.Binding.ConfigurationPin
	}
	relative, _ := filepath.Rel(filepath.Join(r.Root, Home), r.Binding.ConfigurationPin)
	return filepath.Join(r.Root, Pending, relative)
}

func (r *Resources) CheckTarget(home string) error {
	targetHome := home
	if r.Binding.PreviousBindingTransactionId != nil {
		targetHome = filepath.Join(r.Root, Home)
	}
	target := filepath.Join(targetHome, filepath.FromSlash(r.Binding.ControllerReleasePath))
	manifest, err := release.LoadBound(target)
	if err != nil {
		return err
	}
	digest, modes, err := release.StrictDirectoryIdentity(target)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(manifest, r.Release) || release.ControllerResourcesDigest(digest, modes) != r.Binding.ReleaseResourcesSha256 {
		return fmt.Errorf("controller copy differs")
	}
	if r.Binding.ConfigurationMode == storageidl.ControllerConfigurationMode_author_live {
		return nil
	}
	configPath := r.Binding.ConfigurationPin
	if home == filepath.Join(r.Root, Pending) {
		configPath = r.StagedConfiguration()
	}
	config, err := configuration.InspectBinding(configPath)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(config, r.Configuration) {
		return fmt.Errorf("configuration copy differs")
	}
	return nil
}

func SkillRoots(home string) release.SkillRoots {
	root := filepath.Join(home, "skill-roots")
	return release.SkillRoots{Codex: filepath.Join(root, "codex"), Agent: filepath.Join(root, "agent"), Trae: filepath.Join(root, "trae"), Claude: filepath.Join(root, "claude")}
}

func Environment(home, pin string, callerHome string) []string {
	values := map[string]string{"FANLOOP_DATA_HOME": home, "FANLOOP_CONFIG_SOURCE": pin, "FANLOOP_NO_AUTO_UPDATE": "1", "FANLOOP_NO_AUTO_CONFIG": "1"}
	if callerHome != "" {
		values["FANLOOP_DATA_HOME"] = callerHome
	}
	roots := SkillRoots(home)
	for index, name := range []string{"FANLOOP_CODEX_SKILLS_ROOT", "FANLOOP_AGENT_SKILLS_ROOT", "FANLOOP_TRAE_SKILLS_ROOT", "FANLOOP_CLAUDE_SKILLS_ROOT"} {
		values[name] = roots.Values()[index]
	}
	env := []string{}
	for _, item := range os.Environ() {
		name, _, _ := strings.Cut(item, "=")
		if _, replace := values[name]; !replace {
			env = append(env, item)
		}
	}
	for name, value := range values {
		env = append(env, name+"="+value)
	}
	return env
}

func (r *Resources) checkHome(staging string) error {
	if err := release.RealPath(staging, true); err != nil {
		return err
	}
	info, err := os.Lstat(staging)
	if err != nil || info.Mode().Perm() != 0700 {
		return fmt.Errorf("controller staging must be 0700")
	}
	directories := map[string]bool{".": true}
	files := map[string]bool{".workflows.lock": true}
	links := map[string][]string{}
	addParents := func(relative string) {
		for directory := filepath.Dir(relative); directory != "."; directory = filepath.Dir(directory) {
			directories[directory] = true
		}
	}
	if r.Binding.PreviousBindingTransactionId == nil || staging == filepath.Join(r.Root, Home) {
		files[filepath.Join("releases", ".current.lock")] = true
		for relative, mode := range r.ReleaseModes {
			name := filepath.Join(r.Binding.ControllerReleasePath, filepath.FromSlash(relative))
			if mode.IsDir() {
				directories[name] = true
			} else {
				files[name] = true
			}
			addParents(name)
		}
		directories["current"] = true
		for name, target := range map[string]string{"release": "../" + r.Binding.ControllerReleasePath, "bin": "release/bin", "entrypoints": "release/entrypoints", "release.json": "release/release.json"} {
			links[filepath.Join("current", name)] = []string{target}
		}
		for _, root := range SkillRoots(staging).Values() {
			for _, exposed := range release.ExposedSkills() {
				name, _ := filepath.Rel(staging, filepath.Join(root, exposed.Name))
				links[name] = []string{filepath.Join(staging, "current", exposed.Path), filepath.Join(r.Root, Home, "current", exposed.Path)}
				addParents(name)
			}
		}
		links[filepath.Join("config", "current")] = []string{r.Binding.ConfigurationPin, r.Binding.PreviousConfigurationPin}
		directories["config"] = true
	}
	if r.Binding.ConfigurationMode != storageidl.ControllerConfigurationMode_author_live {
		base, _ := filepath.Rel(filepath.Join(r.Root, Home), r.Binding.ConfigurationPin)
		directories[base] = true
		addParents(base)
		for _, file := range r.Configuration.Files {
			name := filepath.Join(base, filepath.FromSlash(file.Path))
			files[name] = true
			addParents(name)
		}
		if r.Binding.ConfigurationMode == storageidl.ControllerConfigurationMode_snapshot {
			name := filepath.Join(base, "configuration.json")
			files[name] = true
			addParents(name)
		}
	}
	return filepath.WalkDir(staging, func(full string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name, err := filepath.Rel(staging, full)
		if err != nil {
			return err
		}
		if entry.IsDir() && directories[name] {
			return nil
		}
		if entry.Type().IsRegular() && files[name] {
			if name == ".workflows.lock" || name == filepath.Join("releases", ".current.lock") {
				info, err := entry.Info()
				if err != nil {
					return err
				}
				if info.Size() != 0 || info.Mode().Perm() != 0600 {
					return fmt.Errorf("unknown controller lock content")
				}
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			target, err := os.Readlink(full)
			if err != nil {
				return err
			}
			for _, allowed := range links[name] {
				if target == allowed {
					return nil
				}
			}
		}
		return fmt.Errorf("unknown or unsupported staging path: %s", full)
	})
}

func ownedLink(path, target, previous string) error {
	actual, err := os.Readlink(path)
	if err == nil {
		if actual == target {
			return nil
		}
		if actual != previous {
			return fmt.Errorf("known link changed: %s", path)
		}
		if err := os.Remove(path); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.Symlink(target, path)
}

func (r *Resources) Prepare(ctx context.Context, checkpoint func(string) error) error {
	staging := filepath.Join(r.Root, Pending)
	if err := os.Mkdir(staging, 0700); err != nil && !os.IsExist(err) {
		return err
	}
	if err := release.RealPath(staging, true); err != nil {
		return err
	}
	if err := r.checkHome(staging); err != nil {
		return err
	}
	unlock, err := workflow.LockSource(filepath.Join(r.Binding.ConfigurationSourcePath, configuration.Category), false)
	if err != nil {
		return err
	}
	if err := r.CheckSource(); err != nil {
		unlock()
		return err
	}
	if r.Binding.PreviousBindingTransactionId == nil {
		if _, err := os.Lstat(filepath.Join(staging, "current")); err == nil {
			if _, err := release.BoundCurrent(staging); err != nil {
				unlock()
				return err
			}
			if err := r.CheckTarget(staging); err != nil {
				unlock()
				return err
			}
		} else if !os.IsNotExist(err) {
			unlock()
			return err
		}
		if err := copyRelease(r.Binding.SourceReleasePath, filepath.Join(staging, r.Binding.ControllerReleasePath), checkpoint); err != nil {
			unlock()
			return err
		}
	}
	if r.Binding.ConfigurationMode != storageidl.ControllerConfigurationMode_author_live {
		if err := copyConfiguration(r.Binding.ConfigurationSourcePath, r.StagedConfiguration(), r.Configuration, checkpoint); err != nil {
			unlock()
			return err
		}
	}
	err = r.CheckSource()
	if err == nil {
		err = r.CheckTarget(staging)
	}
	unlock()
	if err != nil {
		return err
	}
	if r.Binding.PreviousBindingTransactionId != nil {
		command := exec.CommandContext(ctx, filepath.Join(r.Binding.SourceReleasePath, "bin", "fanloop"), "doctor")
		command.Env = Environment(filepath.Join(r.Root, Home), r.StagedConfiguration(), "")
		if output, err := command.CombinedOutput(); err != nil {
			return fmt.Errorf("reused controller Doctor failed: %w: %s", err, output)
		}
		if err := r.CheckSource(); err != nil {
			return err
		}
		if err := r.CheckTarget(staging); err != nil {
			return err
		}
		if err := r.checkHome(staging); err != nil {
			return err
		}
		return checkpoint("validated")
	}
	for _, root := range SkillRoots(staging).Values() {
		for _, exposed := range release.ExposedSkills() {
			link := filepath.Join(root, exposed.Name)
			if _, err := os.Lstat(link); os.IsNotExist(err) {
				continue
			} else if err != nil {
				return err
			}
			if err := ownedLink(link, filepath.Join(staging, "current", exposed.Path), filepath.Join(r.Root, Home, "current", exposed.Path)); err != nil {
				return err
			}
		}
	}
	current := filepath.Join(staging, "config", "current")
	if actual, err := os.Readlink(current); err == nil {
		if actual != r.Binding.ConfigurationPin {
			return fmt.Errorf("staging configuration current changed")
		}
		if err := os.Remove(current); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	roots := SkillRoots(staging)
	binary := filepath.Join(r.Binding.SourceReleasePath, "bin", "fanloop")
	command := exec.CommandContext(ctx, binary, "__install", "--source", r.Binding.SourceReleasePath, "--data-root", staging,
		"--codex-skills-root", roots.Codex, "--agent-skills-root", roots.Agent, "--trae-skills-root", roots.Trae, "--claude-skills-root", roots.Claude)
	command.Env = Environment(staging, r.StagedConfiguration(), "")
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("controller installation failed: %w: %s", err, output)
	}
	command = exec.CommandContext(ctx, filepath.Join(staging, r.Binding.ControllerReleasePath, "bin", "fanloop"), "doctor")
	command.Env = Environment(staging, r.StagedConfiguration(), "")
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("controller Doctor failed: %w: %s", err, output)
	}
	if err := r.CheckSource(); err != nil {
		return err
	}
	if err := r.CheckTarget(staging); err != nil {
		return err
	}
	if err := r.checkHome(staging); err != nil {
		return err
	}
	return checkpoint("validated")
}

func copyRelease(source, target string, checkpoint func(string) error) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return realMkdir(destination, info.Mode().Perm())
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported release file %s", path)
		}
		if err := copyFile(path, destination, info.Mode().Perm()); err != nil {
			return err
		}
		return checkpoint("copy")
	})
}

func copyConfiguration(source, target string, manifest configuration.Manifest, checkpoint func(string) error) error {
	if err := realMkdir(target, 0755); err != nil {
		return err
	}
	paths := []string{}
	for _, file := range manifest.Files {
		paths = append(paths, file.Path)
	}
	if configuration.IsSnapshotSource(source) {
		paths = append(paths, "configuration.json")
	}
	for _, relative := range paths {
		path := filepath.Join(source, filepath.FromSlash(relative))
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, filepath.FromSlash(relative))
		if err := realMkdir(filepath.Dir(destination), 0755); err != nil {
			return err
		}
		if err := copyFile(path, destination, info.Mode().Perm()); err != nil {
			return err
		}
		if err := checkpoint("copy"); err != nil {
			return err
		}
	}
	return nil
}

func realMkdir(path string, mode os.FileMode) error {
	return release.MakeDirectory(path, mode)
}

func copyFile(source, target string, mode os.FileMode) error {
	if err := release.RealPath(source, false); err != nil {
		return err
	}
	content, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if info, err := os.Lstat(target); err == nil {
		if !info.Mode().IsRegular() || info.Mode().Perm() != mode {
			return fmt.Errorf("existing copy is not owned content: %s", target)
		}
		actual, err := os.ReadFile(target)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(actual, content) {
			return fmt.Errorf("existing copy differs: %s", target)
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if err := file.Chmod(mode); err != nil {
		_ = file.Close()
		return err
	}
	_, err = file.Write(content)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return err
}

func (r *Resources) CheckPublished() error {
	home := filepath.Join(r.Root, Home)
	if err := r.checkHome(home); err != nil {
		return err
	}
	if err := r.CheckTarget(home); err != nil {
		return err
	}
	physical, err := release.BoundCurrent(home)
	if err != nil {
		return err
	}
	if physical != filepath.Join(home, r.Binding.ControllerReleasePath) {
		return fmt.Errorf("published current changed")
	}
	return validateSkillRoots(home)
}

func (r *Resources) LockPublication() (func(), error) {
	home := filepath.Join(r.Root, Home)
	if _, err := os.Lstat(home); os.IsNotExist(err) {
		home = filepath.Join(r.Root, Pending)
	} else if err != nil {
		return nil, err
	}
	if err := release.RealPath(home, true); err != nil {
		return nil, err
	}
	configurationUnlock, err := workflow.LockConfiguration(home, true)
	if err != nil {
		return nil, err
	}
	sourceUnlock, err := workflow.LockSource(filepath.Join(r.Binding.ConfigurationSourcePath, configuration.Category), false)
	if err != nil {
		configurationUnlock()
		return nil, err
	}
	return func() { sourceUnlock(); configurationUnlock() }, nil
}

func (r *Resources) Publish() error {
	home := filepath.Join(r.Root, Home)
	staging := filepath.Join(r.Root, Pending)
	if _, err := os.Lstat(home); err == nil {
		if r.Binding.PreviousBindingTransactionId == nil {
			return r.CheckTarget(home)
		}
		if _, err := os.Lstat(r.Binding.ConfigurationPin); err == nil {
			return r.CheckTarget(home)
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := r.CheckTarget(staging); err != nil {
			return err
		}
		if r.Binding.ConfigurationMode != storageidl.ControllerConfigurationMode_author_frozen {
			return fmt.Errorf("unexpected controller reuse")
		}
		if _, err := os.Lstat(r.Binding.ConfigurationPin); os.IsNotExist(err) {
			if err := realMkdir(filepath.Dir(r.Binding.ConfigurationPin), 0755); err != nil {
				return err
			}
			if err := os.Rename(r.StagedConfiguration(), r.Binding.ConfigurationPin); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := r.checkHome(staging); err != nil {
		return err
	}
	if err := r.CheckTarget(staging); err != nil {
		return err
	}
	for _, root := range SkillRoots(staging).Values() {
		for _, exposed := range release.ExposedSkills() {
			link := filepath.Join(root, exposed.Name)
			if err := ownedLink(link, filepath.Join(home, "current", exposed.Path), filepath.Join(staging, "current", exposed.Path)); err != nil {
				return err
			}
		}
	}
	if err := realMkdir(filepath.Join(staging, "config"), 0755); err != nil {
		return err
	}
	link := filepath.Join(staging, "config", "current")
	if err := ownedLink(link, r.Binding.ConfigurationPin, r.Binding.ConfigurationPin); err != nil {
		return err
	}
	return os.Rename(staging, home)
}
