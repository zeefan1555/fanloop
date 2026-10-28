package controller

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zeefan1555/fanloop/internal/idl/storageidl"
	"github.com/zeefan1555/fanloop/internal/release"
)

type Selection struct {
	Binary      string
	Environment []string
	AuthorLive  bool
}

func validateSkillRoots(home string) error {
	for _, skillRoot := range SkillRoots(home).Values() {
		if err := release.RealPath(skillRoot, true); err != nil {
			return err
		}
		for _, exposed := range release.ExposedSkills() {
			target, err := os.Readlink(filepath.Join(skillRoot, exposed.Name))
			if err != nil || target != filepath.Join(home, "current", exposed.Path) {
				return fmt.Errorf("controller Skill root link differs")
			}
		}
	}
	return nil
}

func Select(root, callerHome string) (*Selection, error) {
	binding, err := Read(root)
	if err != nil {
		return nil, err
	}
	if binding != nil && binding.Phase != storageidl.ControllerBindingPhase_committed {
		return nil, fmt.Errorf("prepared controller binding is blocked; run explicit protect only for a protect transaction")
	}
	home := filepath.Join(root, Home)
	if _, err := os.Lstat(home); os.IsNotExist(err) {
		if binding != nil {
			return nil, fmt.Errorf("committed controller home is missing")
		}
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	physical, err := release.BoundCurrent(home)
	if err != nil {
		return nil, err
	}
	manifest, err := release.LoadBound(physical)
	if err != nil {
		return nil, err
	}
	if binding != nil {
		if physical != filepath.Join(home, filepath.FromSlash(binding.ControllerReleasePath)) || manifest.Cli.BinarySha256 != binding.BinarySha256 {
			return nil, fmt.Errorf("controller current differs from committed binding")
		}
		digest, err := release.FileDigest(filepath.Join(physical, "release.json"))
		if err != nil {
			return nil, err
		}
		if digest != binding.ReleaseManifestSha256 {
			return nil, fmt.Errorf("controller manifest differs from committed binding")
		}
		directoryDigest, modes, err := release.StrictDirectoryIdentity(physical)
		if err != nil {
			return nil, err
		}
		if release.ControllerResourcesDigest(directoryDigest, modes) != binding.ReleaseResourcesSha256 {
			return nil, fmt.Errorf("controller resources differ from committed binding")
		}
	}
	if err := validateSkillRoots(home); err != nil {
		return nil, err
	}
	pin := ""
	if binding != nil {
		pin, err = ValidatePin(root, binding)
	} else {
		pin, _, err = ReadPin(root)
		if err == nil && pin != "" {
			_, err = ValidatePin(root, nil)
		}
	}
	if err != nil {
		return nil, err
	}
	if callerHome != "" {
		if err := release.RealPath(filepath.Join(callerHome, "config", "snapshots"), true); err != nil {
			return nil, fmt.Errorf("unsafe caller configuration target: %w", err)
		}
	}
	env := Environment(home, pin, callerHome)
	if pin == "" {
		filtered := []string{}
		for _, item := range env {
			if len(item) >= len("FANLOOP_CONFIG_SOURCE=") && item[:len("FANLOOP_CONFIG_SOURCE=")] == "FANLOOP_CONFIG_SOURCE=" {
				continue
			}
			filtered = append(filtered, item)
		}
		env = filtered
	}
	return &Selection{Binary: filepath.Join(physical, "bin", "fanloop"), Environment: env, AuthorLive: binding != nil && binding.ConfigurationMode == storageidl.ControllerConfigurationMode_author_live && pin == binding.ConfigurationPin}, nil
}
