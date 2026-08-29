package flow

import (
	"errors"
	"io/fs"

	"github.com/zeefan1555/commonloop/errs"
	"github.com/zeefan1555/commonloop/internal/idl/erroridl"
	"github.com/zeefan1555/commonloop/internal/workflow"
)

func newFlowError(code erroridl.ErrorCode, message string, details map[string]string) *erroridl.PublicError {
	return errs.NewCode(code, message, details)
}

func requestError(err error) *erroridl.PublicError {
	return newFlowError(erroridl.ErrorCode_INVALID_ARGUMENT, err.Error(), nil)
}

func storeError(failure *erroridl.PublicError) *erroridl.PublicError { return failure }

func loadWorkflow(selector string) (workflow.Loaded, *erroridl.PublicError) {
	loaded, err := workflow.LoadSelector(selector)
	if err == nil {
		return loaded, nil
	}
	switch {
	case errors.Is(err, workflow.ErrInvalidSelector):
		return workflow.Loaded{}, newFlowError(erroridl.ErrorCode_INVALID_ARGUMENT, err.Error(), nil)
	case errors.Is(err, fs.ErrNotExist):
		return workflow.Loaded{}, newFlowError(erroridl.ErrorCode_WORKFLOW_NOT_FOUND, err.Error(), nil)
	default:
		return workflow.Loaded{}, newFlowError(erroridl.ErrorCode_WORKFLOW_INVALID, err.Error(), nil)
	}
}

func PublicError(err error) *erroridl.PublicError {
	var failure *erroridl.PublicError
	if errors.As(err, &failure) {
		return failure
	}
	return newFlowError(erroridl.ErrorCode_INTERNAL, err.Error(), nil)
}
