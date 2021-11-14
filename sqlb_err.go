package sqlb

import (
	"fmt"
	r "reflect"
)

/*
Used by `StructAssign` to indicate that no fields were provided, and therefore
it was unable to generate valid SQL for an "update set" clause. This can happen
because the input struct was missing or empty, or because all fields were
excluded through the use of `Sparse`. User code should detect this error to
skip the DB request altogether.
*/
var ErrEmptyAssign = error(ErrEmptyExpr{Err{
	`building SQL assignment expression`,
	fmt.Errorf(`assignment must have at least one field`),
}})

// All errors generated by this package have this type, usually wrapped into a
// more specialized one: `ErrInvalidInput{Err{...}}`.
type Err struct {
	While string
	Cause error
}

// Implement the `error` interface.
func (self Err) Error() string { return self.format(``) }

// Implement a hidden interface in "errors".
func (self Err) Unwrap() error { return self.Cause }

func (self Err) format(typ string) string {
	bui := MakeBui(128, 0)
	bui.Str(`[sqlb] error`)

	if typ != `` {
		bui.Space()
		bui.Text = Ident(typ).Append(bui.Text)
	}

	if self.While != `` {
		bui.Str(` while `)
		bui.Str(self.While)
	}

	if self.Cause != nil {
		// No `.Str` to avoid prepending space.
		bui.Text = append(bui.Text, `: `...)
		bui.Str(self.Cause.Error())
	}

	return bui.String()
}

// Specialized type for errors reported by some functions.
type ErrInvalidInput struct{ Err }

// Implement the `error` interface.
func (self ErrInvalidInput) Error() string {
	return self.format(typeName(typeOf((*ErrInvalidInput)(nil))))
}

// Specialized type for errors reported by some functions.
type ErrMissingArgument struct{ Err }

// Implement the `error` interface.
func (self ErrMissingArgument) Error() string {
	return self.format(typeName(typeOf((*ErrMissingArgument)(nil))))
}

// Specialized type for errors reported by some functions.
type ErrUnexpectedParameter struct{ Err }

// Implement the `error` interface.
func (self ErrUnexpectedParameter) Error() string {
	return self.format(typeName(typeOf((*ErrUnexpectedParameter)(nil))))
}

// Specialized type for errors reported by some functions.
type ErrUnusedArgument struct{ Err }

// Implement the `error` interface.
func (self ErrUnusedArgument) Error() string {
	return self.format(typeName(typeOf((*ErrUnusedArgument)(nil))))
}

// Specialized type for errors reported by some functions.
type ErrOrdinalOutOfBounds struct{ Err }

// Implement the `error` interface.
func (self ErrOrdinalOutOfBounds) Error() string {
	return self.format(typeName(typeOf((*ErrOrdinalOutOfBounds)(nil))))
}

// Specialized type for errors reported by some functions.
type ErrUnknownField struct{ Err }

// Implement the `error` interface.
func (self ErrUnknownField) Error() string {
	return self.format(typeName(typeOf((*ErrUnknownField)(nil))))
}

// Specialized type for errors reported by some functions.
type ErrInternal struct{ Err }

// Implement the `error` interface.
func (self ErrInternal) Error() string {
	return self.format(typeName(typeOf((*ErrInternal)(nil))))
}

// Specialized type for errors reported by some functions.
type ErrUnexpectedEOF struct{ Err }

// Implement the `error` interface.
func (self ErrUnexpectedEOF) Error() string {
	return self.format(typeName(typeOf((*ErrUnexpectedEOF)(nil))))
}

// Specialized type for errors reported by some functions.
type ErrEmptyExpr struct{ Err }

// Implement the `error` interface.
func (self ErrEmptyExpr) Error() string {
	return self.format(typeName(typeOf((*ErrEmptyExpr)(nil))))
}

func errOrdinal(err error) error {
	if err == nil {
		return nil
	}
	return ErrInternal{Err{`parsing ordinal parameter`, err}}
}

func errNamed(err error) error {
	if err == nil {
		return nil
	}
	return ErrInternal{Err{`parsing named parameter`, err}}
}

func errMissingOrdinal(val OrdinalParam) ErrMissingArgument {
	return ErrMissingArgument{Err{
		`building SQL expression`,
		fmt.Errorf(`missing ordinal argument %q (index %v)`, val, val.Index()),
	}}
}

func errMissingNamed(val NamedParam) ErrMissingArgument {
	return ErrMissingArgument{Err{
		`building SQL expression`,
		fmt.Errorf(`missing named argument %q (key %q)`, val, val.Key()),
	}}
}

func errUnusedOrdinal(val OrdinalParam) ErrUnusedArgument {
	return ErrUnusedArgument{Err{
		`building SQL expression`,
		fmt.Errorf(`unused ordinal argument %q (index %v)`, val, val.Index()),
	}}
}

func errUnusedNamed(val NamedParam) ErrUnusedArgument {
	return ErrUnusedArgument{Err{
		`building SQL expression`,
		fmt.Errorf(`unused named argument %q (key %q)`, val, val.Key()),
	}}
}

func errExpectedX(desc, while string, val interface{}) ErrInvalidInput {
	return ErrInvalidInput{Err{
		while,
		fmt.Errorf(`expected %v, found %v`, desc, val),
	}}
}

func errExpectedSlice(while string, val interface{}) ErrInvalidInput {
	return errExpectedX(`slice`, while, val)
}

func errExpectedStruct(while string, val interface{}) ErrInvalidInput {
	return errExpectedX(`struct`, while, val)
}

func errUnexpectedArgs(desc, input interface{}) ErrInvalidInput {
	return ErrInvalidInput{Err{
		`building SQL expression`,
		fmt.Errorf(`%v expected no arguments, got %#v`, desc, input),
	}}
}

func errMissingArgs(desc interface{}) ErrInvalidInput {
	return ErrInvalidInput{Err{
		`building SQL expression`,
		fmt.Errorf(`%v expected arguments, got none`, desc),
	}}
}

func errUnknownField(while, jsonPath, typeName string) ErrUnknownField {
	return ErrUnknownField{Err{
		while,
		fmt.Errorf(`no DB path corresponding to JSON path %q in type %v`, jsonPath, typeName),
	}}
}

func errUnsupportedType(while string, typ r.Type) ErrInvalidInput {
	return ErrInvalidInput{Err{
		while,
		fmt.Errorf(`unsupported type %q of kind %q`, typ, typ.Kind()),
	}}
}

func errInvalidOrd(src string) ErrInvalidInput {
	return ErrInvalidInput{Err{
		`parsing ordering expression`,
		fmt.Errorf(
			`%q is not a valid ordering string; expected format: "<ident> (asc|desc)? (nulls (?:first|last))?"`,
			src,
		),
	}}
}
