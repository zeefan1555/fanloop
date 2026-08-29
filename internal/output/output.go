package output

import (
	"encoding/json"
	"io"

	"github.com/zeefan1555/fanloop/errs"
	"github.com/zeefan1555/fanloop/internal/idl/commonidl"
	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/workflow"
)

type Meta struct {
	Command         string        `json:"command"`
	RequirementRoot string        `json:"requirement_root,omitempty"`
	Workflow        *workflow.Ref `json:"workflow,omitempty"`
	DryRun          bool          `json:"dry_run,omitempty"`
}

type envelope struct {
	OK     bool               `json:"ok"`
	Data   any                `json:"data,omitempty"`
	Error  *errs.Presentation `json:"error,omitempty"`
	Meta   Meta               `json:"meta"`
	Notice commonidl.Notice   `json:"_notice"`
}

func Success(writer io.Writer, data any, meta Meta) error {
	return SuccessWithNotice(writer, data, meta, commonidl.Notice{})
}

func SuccessWithNotice(writer io.Writer, data any, meta Meta, notice commonidl.Notice) error {
	return write(writer, envelope{OK: true, Data: data, Meta: meta, Notice: notice})
}

func Partial(writer io.Writer, data any, meta Meta) error {
	return PartialWithNotice(writer, data, meta, commonidl.Notice{})
}

func PartialWithNotice(writer io.Writer, data any, meta Meta, notice commonidl.Notice) error {
	return write(writer, envelope{OK: false, Data: data, Meta: meta, Notice: notice})
}

func Failure(writer io.Writer, failure *erroridl.PublicError, meta Meta) error {
	return FailureWithNotice(writer, failure, meta, commonidl.Notice{})
}

func FailureWithNotice(writer io.Writer, failure *erroridl.PublicError, meta Meta, notice commonidl.Notice) error {
	return write(writer, envelope{OK: false, Error: errs.Present(failure), Meta: meta, Notice: notice})
}

func write(writer io.Writer, value envelope) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
