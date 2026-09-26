package ops

import (
	"context"
	"path/filepath"

	"github.com/zeefan1555/fanloop/errs"
	"github.com/zeefan1555/fanloop/internal/buildinfo"
	"github.com/zeefan1555/fanloop/internal/doctor"
	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/idl/opsidl"
	"github.com/zeefan1555/fanloop/internal/release"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

type Runtime struct{}

var _ opsidl.OpsService = Runtime{}

func DefaultRuntime() Runtime { return Runtime{} }

func (Runtime) Version(_ context.Context, request *opsidl.VersionRequest) (*opsidl.VersionResponse, error) {
	if request == nil {
		return nil, errs.NewCode(erroridl.ErrorCode_INVALID_ARGUMENT, "request is required", nil)
	}
	result, err := buildinfo.Get()
	if err != nil {
		return nil, errs.NewCode(erroridl.ErrorCode_INTERNAL, err.Error(), nil)
	}
	return &result, nil
}

func (Runtime) Update(ctx context.Context, request *opsidl.ConfigurationUpdateRequest) (*opsidl.ConfigurationUpdateResponse, error) {
	if request == nil {
		return nil, errs.NewCode(erroridl.ErrorCode_INVALID_ARGUMENT, "request is required", nil)
	}
	dataRoot, err := release.DefaultDataRoot()
	if err != nil {
		return nil, errs.NewCode(erroridl.ErrorCode_INTERNAL, err.Error(), nil)
	}
	workflowIDs, err := workflow.UpdateFornax(ctx, dataRoot)
	if err != nil {
		return nil, errs.NewCode(erroridl.ErrorCode_INTERNAL, err.Error(), nil)
	}
	return &opsidl.ConfigurationUpdateResponse{
		WorkflowIds:  workflowIDs,
		WorkflowRoot: filepath.Join(dataRoot, "current", "workflows"),
	}, nil
}

func (Runtime) Doctor(_ context.Context, root string, request *opsidl.DoctorRequest) (*opsidl.DoctorResponse, error) {
	if request == nil {
		return nil, errs.NewCode(erroridl.ErrorCode_INVALID_ARGUMENT, "request is required", nil)
	}
	return doctor.DefaultRuntime().Run(root), nil
}
