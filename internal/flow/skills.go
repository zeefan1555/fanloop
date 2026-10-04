package flow

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zeefan1555/fanloop/internal/idl/flowidl"
	"github.com/zeefan1555/fanloop/internal/workflowview"
	"go.yaml.in/yaml/v3"
)

// Skills discovers the complete current package, independently of Step bindings.
func (runtime Runtime) Skills(_ context.Context, root string, request *flowidl.FlowSkillsRequest) (*flowidl.FlowSkillsResponse, error) {
	if request == nil {
		return nil, requestError(errText("request is required"))
	}
	if err := request.IsValid(); err != nil {
		return nil, requestError(err)
	}
	_, _, loaded, failure := load(root)
	if failure != nil {
		return nil, failure
	}
	ids := make([]string, 0, len(loaded.SkillPaths))
	for id := range loaded.SkillPaths {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	response := &flowidl.FlowSkillsResponse{
		Workflow: workflowview.WorkflowRef(loaded.Ref), PackageRoot: loaded.PackageRoot,
		SkillRoot: filepath.Join(loaded.PackageRoot, "skills"), Skills: make([]*flowidl.SkillCatalogEntry, 0, len(ids)),
	}
	for _, id := range ids {
		path := loaded.SkillPaths[id]
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, internalError(err)
		}
		response.Skills = append(response.Skills, &flowidl.SkillCatalogEntry{
			Id: id, Description: skillDescription(string(content)), Path: path,
		})
	}
	if err := response.IsValid(); err != nil {
		return nil, internalError(err)
	}
	return response, nil
}

// Metadata is optional navigation text, never a new package validity condition.
func skillDescription(content string) string {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	if lines[0] != "---" {
		return ""
	}
	for index := 1; index < len(lines); index++ {
		if lines[index] != "---" && lines[index] != "..." {
			continue
		}
		var metadata struct {
			Description yaml.Node `yaml:"description"`
		}
		if err := yaml.Unmarshal([]byte(strings.Join(lines[1:index], "\n")+"\n"), &metadata); err != nil {
			return ""
		}
		if metadata.Description.ShortTag() == "!!str" {
			var description string
			if err := metadata.Description.Decode(&description); err == nil {
				return description
			}
		}
		return ""
	}
	return ""
}
