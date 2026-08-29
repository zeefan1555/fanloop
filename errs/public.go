package errs

import "github.com/zeefan1555/commonloop/internal/idl/erroridl"

type Presentation struct {
	Type      erroridl.ErrorType `json:"type"`
	Code      erroridl.ErrorCode `json:"code"`
	Message   string             `json:"message"`
	Hint      string             `json:"hint"`
	Retryable bool               `json:"retryable"`
	Details   map[string]string  `json:"details,omitempty"`
}

func NewCode(code erroridl.ErrorCode, message string, details map[string]string) *erroridl.PublicError {
	if erroridl.ERROR_SPECS[code] == nil {
		code = erroridl.ErrorCode_INTERNAL
	}
	return &erroridl.PublicError{Code: code, Message: message, Details: details}
}

func Present(failure *erroridl.PublicError) *Presentation {
	if failure == nil {
		return nil
	}
	spec := erroridl.ERROR_SPECS[failure.Code]
	if spec == nil {
		spec = erroridl.ERROR_SPECS[erroridl.ErrorCode_INTERNAL]
	}
	return &Presentation{
		Type: spec.Type, Code: spec.Code, Message: failure.Message, Hint: spec.Hint,
		Retryable: spec.Retryable, Details: failure.Details,
	}
}

func ExitCode(failure *erroridl.PublicError) int {
	if failure == nil {
		return 0
	}
	if spec := erroridl.ERROR_SPECS[failure.Code]; spec != nil {
		return int(spec.ExitCode)
	}
	return 5
}
