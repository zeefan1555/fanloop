package ops

import (
	"context"
	"errors"

	"github.com/zeefan1555/fanloop/errs"
	"github.com/zeefan1555/fanloop/internal/buildinfo"
	"github.com/zeefan1555/fanloop/internal/doctor"
	"github.com/zeefan1555/fanloop/internal/fornaxmirror"
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

func (Runtime) Doctor(_ context.Context, root string, request *opsidl.DoctorRequest) (*opsidl.DoctorResponse, error) {
	if request == nil {
		return nil, errs.NewCode(erroridl.ErrorCode_INVALID_ARGUMENT, "request is required", nil)
	}
	dataRoot, err := release.DefaultDataRoot()
	if err != nil {
		return nil, err
	}
	unlock, err := workflow.LockConfiguration(dataRoot, false)
	if err != nil {
		return nil, err
	}
	defer unlock()
	return doctor.DefaultRuntime().Run(root), nil
}

func (Runtime) ConfigDrift(ctx context.Context, request *opsidl.ConfigDriftRequest) (*opsidl.ConfigDriftResponse, error) {
	if request == nil {
		return nil, errs.NewCode(erroridl.ErrorCode_INVALID_ARGUMENT, "request is required", nil)
	}
	dataRoot, err := release.DefaultDataRoot()
	if err != nil {
		return nil, err
	}
	report, err := fornaxmirror.Check(ctx, dataRoot, fornaxmirror.DefaultClient())
	if err != nil {
		code := erroridl.ErrorCode_INTERNAL
		if errors.Is(err, fornaxmirror.ErrInvalidInput) {
			code = erroridl.ErrorCode_INVALID_ARGUMENT
		}
		return nil, errs.NewCode(code, err.Error(), nil)
	}
	response := &opsidl.ConfigDriftResponse{Status: driftStatus(report.Status), Workflows: make([]*opsidl.WorkflowMirrorCheck, 0, len(report.Workflows))}
	if report.ConfigurationCommit != "" {
		response.ConfigurationCommit = stringPointer(report.ConfigurationCommit)
	}
	for _, item := range report.Workflows {
		check := &opsidl.WorkflowMirrorCheck{WorkflowId: item.WorkflowID, SkillKey: item.SkillKey, Status: workflowStatus(item.Status)}
		if item.SkillID != "" {
			check.SkillId = stringPointer(item.SkillID)
		}
		if item.ReleasedVersion != "" {
			check.ReleasedVersion = stringPointer(item.ReleasedVersion)
		}
		if item.Detail != "" {
			check.Detail = stringPointer(item.Detail)
		}
		response.Workflows = append(response.Workflows, check)
	}
	return response, nil
}

func stringPointer(value string) *string { return &value }

func driftStatus(value fornaxmirror.DriftStatus) opsidl.ConfigDriftStatus {
	switch value {
	case fornaxmirror.DriftInSync:
		return opsidl.ConfigDriftStatus_in_sync
	case fornaxmirror.DriftLocalBehind:
		return opsidl.ConfigDriftStatus_local_behind
	case fornaxmirror.DriftLocalModified:
		return opsidl.ConfigDriftStatus_local_modified
	default:
		return opsidl.ConfigDriftStatus_unavailable
	}
}

func workflowStatus(value fornaxmirror.WorkflowStatus) opsidl.WorkflowMirrorStatus {
	switch value {
	case fornaxmirror.WorkflowInSync:
		return opsidl.WorkflowMirrorStatus_in_sync
	case fornaxmirror.WorkflowLocalBehind:
		return opsidl.WorkflowMirrorStatus_local_behind
	case fornaxmirror.WorkflowLocalModified:
		return opsidl.WorkflowMirrorStatus_local_modified
	default:
		return opsidl.WorkflowMirrorStatus_unavailable
	}
}
