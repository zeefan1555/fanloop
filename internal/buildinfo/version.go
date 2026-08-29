package buildinfo

import (
	"os"

	idl "github.com/zeefan1555/commonloop/internal/idl/opsidl"
	"github.com/zeefan1555/commonloop/internal/release"
	"github.com/zeefan1555/commonloop/internal/state"
	"github.com/zeefan1555/commonloop/internal/workflow"
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
		workflows := make([]*idl.WorkflowRelease, len(manifest.Workflows))
		for index, item := range manifest.Workflows {
			workflows[index] = &idl.WorkflowRelease{Id: item.Id, Digest: item.Sha256}
		}
		return idl.VersionResponse{
			ReleaseVersion: ReleaseVersion, StateSchema: manifest.StateSchema,
			Skills: skills, Workflows: workflows, CommitSha: Commit,
		}, nil
	}
	return GetEmbedded()
}

func GetEmbedded() (idl.VersionResponse, error) {
	loaded, err := workflow.List()
	if err != nil {
		return idl.VersionResponse{}, err
	}
	workflows := make([]*idl.WorkflowRelease, 0, len(loaded))
	for _, item := range loaded {
		workflows = append(workflows, &idl.WorkflowRelease{Id: item.Ref.ID, Digest: item.Ref.Digest})
	}
	return idl.VersionResponse{
		ReleaseVersion: ReleaseVersion,
		StateSchema: &idl.StateSchemaSupport{
			ReadVersions: []int32{int32(state.CurrentStateSchemaVersion)}, WriteVersion: int32(state.CurrentStateSchemaVersion),
		},
		Skills:    []*idl.SkillRelease{},
		Workflows: workflows,
		CommitSha: Commit,
	}, nil
}
