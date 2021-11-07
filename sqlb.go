package sqlb

import r "reflect"

/*
Short for "expression". Defines an arbitrary SQL expression. The method appends
arbitrary SQL text. In both the input and output, the arguments must correspond
to the parameters in the SQL text. Different databases support different styles
of ordinal parameters. This package always generates Postgres-style ordinal
parameters such as "$1", renumerating them as necessary.

This method is allowed to panic. Use `(*Bui).CatchExprs` to catch
expression-encoding panics and convert them to errors.

All `Expr` types in this package also implement `Appender` and `fmt.Stringer`.
*/
type Expr interface {
	AppendExpr([]byte, []interface{}) ([]byte, []interface{})
}

/*
Short for "parametrized expression". Similar to `Expr`, but requires an external
input in order to be a valid expression. Implemented by preparsed query types,
namely by `Prep`.
*/
type ParamExpr interface {
	AppendParamExpr([]byte, []interface{}, ArgDict) ([]byte, []interface{})
}

/*
Appends a text repesentation. Sometimes allows better efficiency than
`fmt.Stringer`. Implemented by all `Expr` types in this package.
*/
type Appender interface{ Append([]byte) []byte }

/*
Dictionary of arbitrary arguments, ordinal and/or named. Used as input to
`ParamExpr`(parametrized expressions). This package provides multiple
implementations: slice-based `List`, map-based `Dict`, and struct-based
`StructDict`. May optionally implement `OrdinalRanger` and `NamedRanger` to
validate used/unused arguments.
*/
type ArgDict interface {
	IsEmpty() bool
	Len() int
	GotOrdinal(int) (interface{}, bool)
	GotNamed(string) (interface{}, bool)
}

/*
Optional extension for `ArgDict`. If implemented, this is used to validate
used/unused ordinal arguments after building a parametrized SQL expression such
as `StrQ`/`Prep`.
*/
type OrdinalRanger interface {
	/**
	Must iterate over argument indexes from 0 to N, calling the function for each
	index. The func is provided by this package, and will panic for each unused
	argument.
	*/
	RangeOrdinal(func(int))
}

/*
Optional extension for `ArgDict`. If implemented, this is used to validate
used/unused named arguments after building a parametrized SQL expression such
as `StrQ`/`Prep`.
*/
type NamedRanger interface {
	/**
	Must iterate over known argument names, calling the function for each name.
	The func is provided by this package, and will panic for each unused
	argument.
	*/
	RangeNamed(func(string))
}

/*
Used by `Partial` for filtering struct fields. See `Sparse` and `Partial` for
explanations.
*/
type Haser interface{ Has(string) bool }

/*
Represents an arbitrary struct where not all fields are "present". Calling
`.Get` returns the underlying struct value. Calling `.HasField` answers the
question "is this field present?".

Secretly supported by struct-scanning expressions such as `StructInsert`,
`StructAssign`, `StructValues`, `Cond`, and more. These types attempt to upcast
the inner value to `Sparse`, falling back on using the inner value as-is. This
allows to correctly implement REST PATCH semantics by using only the fields
that were present in a particular HTTP request, while keeping this
functionality optional.

Concrete implementation: `Partial`.
*/
type Sparse interface {
	Get() interface{}
	HasField(r.StructField) bool
}