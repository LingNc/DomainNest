package errs

import (
	"errors"
	"fmt"
)

// Code is a machine-readable error identifier.
type Code string

// CodedError carries an error code, optional params for interpolation,
// and an underlying error for debugging.
type CodedError struct {
	Code   Code
	Params []interface{}
	Msg    string // developer-facing message (fallback)
	Err    error  // underlying error for wrapping
}

func (e *CodedError) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	return string(e.Code)
}

func (e *CodedError) Unwrap() error {
	return e.Err
}

// New creates a simple coded error with no params.
func New(code Code, msg string) error {
	return &CodedError{Code: code, Msg: msg}
}

// Newf creates a coded error with formatted message (for debugging).
func Newf(code Code, format string, args ...interface{}) error {
	return &CodedError{Code: code, Msg: fmt.Sprintf(format, args...)}
}

// WithParams creates a coded error with interpolation params.
func WithParams(code Code, params ...interface{}) error {
	return &CodedError{Code: code, Params: params, Msg: string(code)}
}

// WithParamsMsg creates a coded error with both params and a debug message.
func WithParamsMsg(code Code, msg string, params ...interface{}) error {
	return &CodedError{Code: code, Msg: msg, Params: params}
}

// Wrap wraps an existing error with a code.
func Wrap(code Code, err error) error {
	return &CodedError{Code: code, Msg: err.Error(), Err: err}
}

// WrapMsg wraps an error with a code and custom message.
func WrapMsg(code Code, msg string, err error) error {
	return &CodedError{Code: code, Msg: msg, Err: err}
}

// IsCoded checks if an error is a CodedError.
func IsCoded(err error) (*CodedError, bool) {
	var ce *CodedError
	if errors.As(err, &ce) {
		return ce, true
	}
	return nil, false
}