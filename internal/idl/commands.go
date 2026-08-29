package idl

import (
	"reflect"

	"github.com/zeefan1555/commonloop/internal/idl/commonidl"
	"github.com/zeefan1555/commonloop/internal/idl/erroridl"
)

type CommandSpec struct {
	ID               string
	Summary          string
	RequestType      reflect.Type
	ResponseType     reflect.Type
	Risk             commonidl.CommandRisk
	RequirementScope commonidl.RequirementScope
	SupportsDryRun   bool
	Errors           []*erroridl.ErrorSpec
}

func CommandSpecs() []CommandSpec { return append([]CommandSpec(nil), commandSpecs...) }

func LookupCommand(id string) (CommandSpec, bool) {
	for _, spec := range commandSpecs {
		if spec.ID == id {
			return spec, true
		}
	}
	return CommandSpec{}, false
}

func errorSpecs(codes ...erroridl.ErrorCode) []*erroridl.ErrorSpec {
	result := make([]*erroridl.ErrorSpec, 0, len(codes))
	for _, code := range codes {
		spec := erroridl.ERROR_SPECS[code]
		if spec == nil {
			panic("missing generated error spec: " + code.String())
		}
		clone := *spec
		result = append(result, &clone)
	}
	return result
}
