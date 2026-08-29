package ops

import (
	"context"

	"github.com/zeefan1555/commonloop/errs"
	"github.com/zeefan1555/commonloop/internal/buildinfo"
	"github.com/zeefan1555/commonloop/internal/doctor"
	"github.com/zeefan1555/commonloop/internal/idl/erroridl"
	"github.com/zeefan1555/commonloop/internal/idl/opsidl"
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

func (Runtime) Doctor(_ context.Context, root string, request *opsidl.DoctorRequest) (*opsidl.DoctorResponse, error) {
	if request == nil {
		return nil, errs.NewCode(erroridl.ErrorCode_INVALID_ARGUMENT, "request is required", nil)
	}
	return doctor.DefaultRuntime().Run(root), nil
}
