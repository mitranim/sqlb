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
	ErrCodeInvalidInput        ErrCode = "InvalidInput"
	ErrCodeMissingArgument     ErrCode = "MissingArgument"
	ErrCodeUnexpectedParameter ErrCode = "UnexpectedParameter"
	ErrCodeUnusedArgument      ErrCode = "UnusedArgument"
	ErrCodeTooManyArguments    ErrCode = "TooManyArguments"
	ErrCodeOrdinalOutOfBounds  ErrCode = "OrdinalOutOfBounds"
	ErrCodeUnknownField        ErrCode = "UnknownField"
	ErrCodeInternal            ErrCode = "Internal"
)

/*
Use blank error variables to detect error types:

	if errors.Is(err, sqlb.ErrIndexMismatch) {
		// Handle specific error.
	}

Note that errors returned by this package can't be compared via `==` because
they may include additional details about the circumstances. When compared by
`errors.Is`, they compare `.Cause` and fall back on `.Code`.
*/
var (
	ErrInvalidInput        Err = Err{Code: ErrCodeInvalidInput, Cause: errors.New(`invalid input`)}
	ErrMissingArgument     Err = Err{Code: ErrCodeMissingArgument, Cause: errors.New(`missing argument`)}
	ErrUnexpectedParameter Err = Err{Code: ErrCodeUnexpectedParameter, Cause: errors.New(`unexpected parameter`)}
	ErrUnusedArgument      Err = Err{Code: ErrCodeUnusedArgument, Cause: errors.New(`unused argument`)}
	ErrTooManyArguments    Err = Err{Code: ErrCodeTooManyArguments, Cause: errors.New(`too many arguments`)}
	ErrOrdinalOutOfBounds  Err = Err{Code: ErrCodeOrdinalOutOfBounds, Cause: errors.New(`ordinal parameter exceeds arguments`)}
	ErrUnknownField        Err = Err{Code: ErrCodeUnknownField, Cause: errors.New(`unknown field`)}
	ErrInternal            Err = Err{Code: ErrCodeInternal, Cause: errors.New(`internal error`)}
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
	msg := `[sqlb]`
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
