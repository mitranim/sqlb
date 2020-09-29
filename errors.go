package sqlb

import (
	"errors"
	"fmt"
)

/*
Error codes. You probably shouldn't use this directly; instead, use the `Err`
variables with `errors.Is`.
*/
type ErrCode string

const (
	ErrCodeUnknown             ErrCode = ""
	ErrCodeIndexMismatch       ErrCode = "IndexMismatch"
	ErrCodeInvalidInput        ErrCode = "InvalidInput"
	ErrCodeMissingArgument     ErrCode = "MissingArgument"
	ErrCodeMissingParameter    ErrCode = "MissingParameter"
	ErrCodeUnexpectedParameter ErrCode = "UnexpectedParameter"
	ErrCodeUnusedArgument      ErrCode = "UnusedArgument"
)

/*
Use blank error variables to detect error types:

	if errors.Is(err, sqlb.ErrNoRows) {
		// Handle specific error.
	}

Note that errors returned by this package can't be compared via `==` because
they may include additional details about the circumstances. When compared by
`errors.Is`, they compare `.Cause` and fall back on `.Code`.
*/
var (
	ErrIndexMismatch       Err = Err{Code: ErrCodeIndexMismatch, Cause: errors.New(`index mismatch`)}
	ErrInvalidInput        Err = Err{Code: ErrCodeInvalidInput, Cause: errors.New(`invalid input`)}
	ErrMissingArgument     Err = Err{Code: ErrCodeMissingArgument, Cause: errors.New(`missing argument`)}
	ErrMissingParameter    Err = Err{Code: ErrCodeMissingParameter, Cause: errors.New(`missing parameter`)}
	ErrUnexpectedParameter Err = Err{Code: ErrCodeUnexpectedParameter, Cause: errors.New(`unexpected parameter`)}
	ErrUnusedArgument      Err = Err{Code: ErrCodeUnusedArgument, Cause: errors.New(`unused argument`)}
)

// Type of errors returned by this package.
type Err struct {
	Code  ErrCode
	While string
	Cause error
}

// Implement `error`.
func (self Err) Error() string {
	if self == (Err{}) {
		return ""
	}
	msg := `SQL error`
	if self.Code != ErrCodeUnknown {
		msg += fmt.Sprintf(` %s`, self.Code)
	}
	if self.While != "" {
		msg += fmt.Sprintf(` while %v`, self.While)
	}
	if self.Cause != nil {
		msg += `: ` + self.Cause.Error()
	}
	return msg
}

// Implement a hidden interface in "errors".
func (self Err) Is(other error) bool {
	if self.Cause != nil && errors.Is(self.Cause, other) {
		return true
	}
	err, ok := other.(Err)
	return ok && err.Code == self.Code
}

// Implement a hidden interface in "errors".
func (self Err) Unwrap() error {
	return self.Cause
}

func (self Err) while(while string) Err {
	self.While = while
	return self
}

func (self Err) because(cause error) Err {
	self.Cause = cause
	return self
}
