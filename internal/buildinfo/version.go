package buildinfo

import (
	"os"

	idl "github.com/zeefan1555/fanloop/internal/idl/opsidl"
	"github.com/zeefan1555/fanloop/internal/release"
	"github.com/zeefan1555/fanloop/internal/state"
)

var (
	ReleaseVersion = "dev"
	CLIVersion     = "dev"
	Commit         = "unknown"
)

func Get() (idl.VersionResponse, error) {
	executable, err := os.Executable()
	if err != nil {
		return GetEmbedded()
	}
	if manifest, err := release.Load(release.RootForExecutable(executable)); err == nil {
		skills := make([]*idl.SkillRelease, len(manifest.Skills))
		for index, item := range manifest.Skills {
			skills[index] = &idl.SkillRelease{Name: item.Name, Version: item.Version}
		}
		return idl.VersionResponse{
			ReleaseVersion: ReleaseVersion, StateSchema: manifest.StateSchema,
			Skills: skills, CommitSha: Commit,
		}, nil
	}
	return GetEmbedded()
}

func GetEmbedded() (idl.VersionResponse, error) {
	return idl.VersionResponse{
		ReleaseVersion: ReleaseVersion,
		StateSchema: &idl.StateSchemaSupport{
			ReadVersions: []int32{int32(state.CurrentStateSchemaVersion)}, WriteVersion: int32(state.CurrentStateSchemaVersion),
		},
		Skills: []*idl.SkillRelease{}, CommitSha: Commit,
	}, nil
}
