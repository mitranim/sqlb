package sqlb

import (
	"fmt"
	r "reflect"
	"strconv"
	"strings"
)

/*
Shortcut for interpolating strings into queries. Because this implements `Expr`,
when used as an argument in another expression, this will be directly
interpolated into the resulting query string. See the examples.
*/
type Str string

// Implement the `Expr` interface, making this a sub-expression.
func (self Str) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return self.Append(text), args
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Str) Append(text []byte) []byte {
	return appendMaybeSpaced(text, string(self))
}

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Str) String() string { return string(self) }

// Represents an SQL identifier, always quoted.
type Ident string

// Implement the `Expr` interface, making this a sub-expression.
func (self Ident) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return self.Append(text), args
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Ident) Append(text []byte) []byte {
	validateIdent(string(self))
	text = maybeAppendSpace(text)
	text = append(text, quoteDouble)
	text = append(text, self...)
	text = append(text, quoteDouble)
	return text
}

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Ident) String() string { return AppenderString(&self) }

// Shortcut for internal use.
func (self Ident) BuiAppend(bui *Bui) {
	bui.Text = self.Append(bui.Text)
}

/*
Represents a nested SQL identifier where all elements are quoted but not
parenthesized. Useful for schema-qualified paths. For nested paths that don't
begin with a schema, use `Path` instead.
*/
type Identifier []string

// Implement the `Expr` interface, making this a sub-expression.
func (self Identifier) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return self.Append(text), args
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Identifier) Append(text []byte) []byte {
	if len(self) == 0 {
		return text
	}
	for i, val := range self {
		if i > 0 {
			text = append(text, `.`...)
		}
		text = Ident(val).Append(text)
	}
	return text
}

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Identifier) String() string { return AppenderString(&self) }

// Normalizes the expression, returning nil or a single `Ident` if the length
// allows this. Otherwise returns self as-is.
func (self Identifier) Norm() Expr {
	switch len(self) {
	case 0:
		return Identifier(nil)
	case 1:
		return Ident(self[0])
	default:
		return self
	}
}

/*
Represents a nested SQL identifier where the first outer element is
parenthesized, and every element is quoted. Useful for nested paths that begin
with a table or view name. For schema-qualified paths, use `Identifier`
instead.
*/
type Path []string

// Implement the `Expr` interface, making this a sub-expression.
func (self Path) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return self.Append(text), args
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Path) Append(text []byte) []byte {
	if len(self) == 0 {
		return text
	}

	if len(self) == 1 {
		return Ident(self[0]).Append(text)
	}

	text = appendMaybeSpaced(text, `(`)
	text = Ident(self[0]).Append(text)
	text = append(text, `)`...)

	for _, val := range self[1:] {
		text = append(text, `.`...)
		text = Ident(val).Append(text)
	}
	return text
}

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Path) String() string { return AppenderString(&self) }

// Normalizes the expression, returning nil or a single `Ident` if the length
// allows this. Otherwise returns self as-is.
func (self Path) Norm() Expr {
	switch len(self) {
	case 0:
		return Path(nil)
	case 1:
		return Ident(self[0])
	default:
		return self
	}
}

/*
Represents an arbitrarily-nested SQL path that gets encoded as a SINGLE quoted
identifier, where elements are dot-separated. This is a common convention for
nested structs, supported by SQL-scanning libraries such as
https://github.com/mitranim/gos.
*/
type PseudoPath []string

// Implement the `Expr` interface, making this a sub-expression.
func (self PseudoPath) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return self.Append(text), args
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self PseudoPath) Append(text []byte) []byte {
	if len(self) == 0 {
		return text
	}

	text = maybeAppendSpace(text)
	text = append(text, quoteDouble)

	for i, val := range self {
		validateIdent(val)
		if i > 0 {
			text = append(text, `.`...)
		}
		text = append(text, val...)
	}

	text = append(text, quoteDouble)
	return text
}

// Implement the `fmt.Stringer` interface for debug purposes.
func (self PseudoPath) String() string { return AppenderString(&self) }

// Normalizes the expression, returning nil or a single `Ident` if the length
// allows this. Otherwise returns self as-is.
func (self PseudoPath) Norm() Expr {
	switch len(self) {
	case 0:
		return PseudoPath(nil)
	case 1:
		return Ident(self[0])
	default:
		return self
	}
}

/*
Represents an arbitrarily-nested SQL path that gets encoded as `Path` followed
by `PseudoPath` alias. Useful for building "select" clauses. Used internally by
`ColsDeep`.
*/
type AliasedPath []string

// Implement the `Expr` interface, making this a sub-expression.
func (self AliasedPath) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return self.Append(text), args
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self AliasedPath) Append(text []byte) []byte {
	if len(self) == 0 {
		return text
	}

	if len(self) == 1 {
		return Ident(self[0]).Append(text)
	}

	text = Path(self).Append(text)
	text = append(text, ` as `...)
	text = PseudoPath(self).Append(text)
	return text
}

// Implement the `fmt.Stringer` interface for debug purposes.
func (self AliasedPath) String() string { return AppenderString(&self) }

// Normalizes the expression, returning nil or a single `Ident` if the length
// allows this. Otherwise returns self as-is.
func (self AliasedPath) Norm() Expr {
	switch len(self) {
	case 0:
		return AliasedPath(nil)
	case 1:
		return Ident(self[0])
	default:
		return self
	}
}

/*
Same as `Identifier`, but preceded by the word "table". The SQL clause
"table some_name" is equivalent to "select * from some_name".
*/
type Table Identifier

// Implement the `Expr` interface, making this a sub-expression.
func (self Table) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return self.Append(text), args
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Table) Append(text []byte) []byte {
	if len(self) == 0 {
		return text
	}
	text = appendMaybeSpaced(text, `table`)
	text = Identifier(self).Append(text)
	return text
}

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Table) String() string { return AppenderString(&self) }

/*
Variable-sized sequence of expressions. When encoding, expressions will be
space-separated if necessary.
*/
type Exprs []Expr

// Implement the `Expr` interface, making this a sub-expression.
func (self Exprs) AppendExpr(text []byte, args []any) ([]byte, []any) {
	bui := Bui{text, args}
	for _, val := range self {
		bui.Expr(val)
	}
	return bui.Get()
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Exprs) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Exprs) String() string { return exprString(self) }

/*
Represents an SQL "any()" expression. The inner value may be an instance of
`Expr`, or an arbitrary argument.
*/
type Any [1]any

// Implement the `Expr` interface, making this a sub-expression.
func (self Any) AppendExpr(text []byte, args []any) ([]byte, []any) {
	bui := Bui{text, args}
	bui.Str(`any (`)
	bui.Any(self[0])
	bui.Str(`)`)
	return bui.Get()
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Any) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Any) String() string { return exprString(self) }

/*
Represents an SQL assignment such as `"some_col" = arbitrary_expression`. The
LHS must be a column name, while the RHS can be an `Expr` instance or an
arbitrary argument.
*/
type Assign struct {
	Lhs Ident
	Rhs any
}

// Implement the `Expr` interface, making this a sub-expression.
func (self Assign) AppendExpr(text []byte, args []any) ([]byte, []any) {
	bui := Bui{text, args}
	bui.Any(self.Lhs)
	bui.Str(`=`)
	bui.SubAny(self.Rhs)
	return bui.Get()
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Assign) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Assign) String() string { return exprString(self) }

/*
Short for "equal". Represents SQL equality such as `A = B` or `A is null`.
Counterpart to `Neq`.
*/
type Eq [2]any

// Implement the `Expr` interface, making this a sub-expression.
func (self Eq) AppendExpr(text []byte, args []any) ([]byte, []any) {
	text, args = self.AppendLhs(text, args)
	text, args = self.AppendRhs(text, args)
	return text, args
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Eq) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Eq) String() string { return exprString(self) }

/*
Note: LHS and RHS are encoded differently because some SQL equality expressions
are asymmetric. For example, `any` allows an array only on the RHS, and there's
no way to invert it (AFAIK).
*/
func (self Eq) AppendLhs(text []byte, args []any) ([]byte, []any) {
	bui := Bui{text, args}
	bui.SubAny(self[0])
	return bui.Get()
}

func (self Eq) AppendRhs(text []byte, args []any) ([]byte, []any) {
	bui := Bui{text, args}
	val := norm(self[1])

	if val == nil {
		bui.Str(`is null`)
		return bui.Get()
	}

	bui.Str(`=`)
	bui.SubAny(val)
	return bui.Get()
}

/*
Short for "not equal". Represents SQL non-equality such as `A <> B` or
`A is not null`. Counterpart to `Eq`.
*/
type Neq [2]any

// Implement the `Expr` interface, making this a sub-expression.
func (self Neq) AppendExpr(text []byte, args []any) ([]byte, []any) {
	text, args = self.AppendLhs(text, args)
	text, args = self.AppendRhs(text, args)
	return text, args
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Neq) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Neq) String() string { return exprString(self) }

// See the comment on `Eq.AppendLhs`.
func (self Neq) AppendLhs(text []byte, args []any) ([]byte, []any) {
	bui := Bui{text, args}
	bui.SubAny(self[0])
	return bui.Get()
}

func (self Neq) AppendRhs(text []byte, args []any) ([]byte, []any) {
	bui := Bui{text, args}
	val := norm(self[1])

	if val == nil {
		bui.Str(`is not null`)
		return bui.Get()
	}

	bui.Str(`<>`)
	bui.SubAny(val)
	return bui.Get()
}

// Represents an SQL expression `A = any(B)`. Counterpart to `NeqAny`.
type EqAny [2]any

// Implement the `Expr` interface, making this a sub-expression.
func (self EqAny) AppendExpr(text []byte, args []any) ([]byte, []any) {
	bui := Bui{text, args}
	bui.SubAny(self[0])
	bui.Str(`=`)
	bui.Set(Any{self[1]}.AppendExpr(bui.Get()))
	return bui.Get()
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self EqAny) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self EqAny) String() string { return exprString(self) }

// Represents an SQL expression `A <> any(B)`. Counterpart to `EqAny`.
type NeqAny [2]any

// Implement the `Expr` interface, making this a sub-expression.
func (self NeqAny) AppendExpr(text []byte, args []any) ([]byte, []any) {
	bui := Bui{text, args}
	bui.SubAny(self[0])
	bui.Str(`<>`)
	bui.Set(Any{self[1]}.AppendExpr(bui.Get()))
	return bui.Get()
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self NeqAny) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self NeqAny) String() string { return exprString(self) }

// Represents SQL logical negation such as `not A`. The inner value can be an
// instance of `Expr` or an arbitrary argument.
type Not [1]any

// Implement the `Expr` interface, making this a sub-expression.
func (self Not) AppendExpr(text []byte, args []any) ([]byte, []any) {
	bui := Bui{text, args}
	bui.Str(`not`)
	bui.SubAny(self[0])
	return bui.Get()
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Not) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Not) String() string { return exprString(self) }

/*
Represents a sequence of arbitrary sub-expressions or arguments, joined with a
customizable delimiter, with a customizable fallback in case of empty list.
This is mostly an internal tool for building other sequences, such as `And` and
`Or`. The inner value may be nil or a single `Expr`, otherwise it must be a
slice.
*/
type Seq struct {
	Empty string
	Delim string
	Val   any
}

// Implement the `Expr` interface, making this a sub-expression.
func (self Seq) AppendExpr(text []byte, args []any) ([]byte, []any) {
	bui := Bui{text, args}
	val := self.Val

	impl, _ := val.(Expr)
	if impl != nil {
		bui.Expr(impl)
	} else {
		self.any(&bui, val)
	}

	return bui.Get()
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Seq) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Seq) String() string { return exprString(self) }

func (self *Seq) any(bui *Bui, val any) {
	switch kindOf(val) {
	case r.Invalid:
		self.appendEmpty(bui)
	case r.Slice:
		self.appendSlice(bui, val)
	default:
		panic(errExpectedSlice(`building SQL expression`, val))
	}
}

func (self *Seq) appendEmpty(bui *Bui) {
	bui.Str(self.Empty)
}

func (self Seq) appendSlice(bui *Bui, src any) {
	val := valueOf(src)

	if val.Len() == 0 {
		self.appendEmpty(bui)
		return
	}

	if val.Len() == 1 {
		bui.Any(val.Index(0).Interface())
		return
	}

	for i := range counter(val.Len()) {
		if i > 0 {
			bui.Str(self.Delim)
		}
		bui.SubAny(val.Index(i).Interface())
	}
}

/*
Represents a comma-separated list of arbitrary sub-expressions. The inner value
may be nil or a single `Expr`, otherwise it must be a slice.
*/
type Comma [1]any

// Implement the `Expr` interface, making this a sub-expression.
func (self Comma) AppendExpr(text []byte, args []any) ([]byte, []any) {
	// May revise in the future. Some SQL expressions, such as composite literals
	// expressed as strings, are sensitive to whitespace around commas.
	return Seq{``, `, `, self[0]}.AppendExpr(text, args)
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Comma) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Comma) String() string { return exprString(self) }

/*
Represents a sequence of arbitrary sub-expressions or arguments joined by the
SQL `and` operator. Rules for the inner value:

	* nil or empty     -> fallback to `true`
	* single `Expr`    -> render it as-is
	* non-empty slice  -> render its individual elements joined by `and`
	* non-empty struct -> render column equality conditions joined by `and`
*/
type And [1]any

// Implement the `Expr` interface, making this a sub-expression.
func (self And) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return Cond{`true`, `and`, self[0]}.AppendExpr(text, args)
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self And) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self And) String() string { return exprString(self) }

/*
Represents a sequence of arbitrary sub-expressions or arguments joined by the
SQL `or` operator. Rules for the inner value:

	* nil or empty     -> fallback to `false`
	* single `Expr`    -> render it as-is
	* non-empty slice  -> render its individual elements joined by `or`
	* non-empty struct -> render column equality conditions joined by `or`
*/
type Or [1]any

// Implement the `Expr` interface, making this a sub-expression.
func (self Or) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return Cond{`false`, `or`, self[0]}.AppendExpr(text, args)
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Or) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Or) String() string { return exprString(self) }

// Syntactic shortcut, same as `And` with a slice of sub-expressions or arguments.
type Ands []any

// Implement the `Expr` interface, making this a sub-expression.
func (self Ands) AppendExpr(text []byte, args []any) ([]byte, []any) {
	if len(self) == 0 {
		return And{}.AppendExpr(text, args)
	}
	return And{[]any(self)}.AppendExpr(text, args)
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Ands) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Ands) String() string { return exprString(self) }

// Syntactic shortcut, same as `Or` with a slice of sub-expressions or arguments.
type Ors []any

// Implement the `Expr` interface, making this a sub-expression.
func (self Ors) AppendExpr(text []byte, args []any) ([]byte, []any) {
	if len(self) == 0 {
		return Or{}.AppendExpr(text, args)
	}
	return Or{[]any(self)}.AppendExpr(text, args)
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Ors) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Ors) String() string { return exprString(self) }

/*
Superset of `Seq` with additional support for structs. When the inner value is
a struct, this generates a sequence of equality expressions, comparing the
struct's column names against the corresponding field values. Field values may
be arbitrary sub-expressions or arguments.

This is mostly an internal tool for building other expression types. Used
internally by `And` and `Or`.
*/
type Cond Seq

// Implement the `Expr` interface, making this a sub-expression.
func (self Cond) AppendExpr(text []byte, args []any) ([]byte, []any) {
	bui := Bui{text, args}
	val := self.Val

	impl, _ := val.(Expr)
	if impl != nil {
		bui.Expr(impl)
	} else {
		self.any(&bui, val)
	}

	return bui.Get()
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Cond) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Cond) String() string { return exprString(self) }

func (self *Cond) any(bui *Bui, val any) {
	switch kindOf(val) {
	case r.Invalid:
		self.appendEmpty(bui)
	case r.Struct:
		self.appendStruct(bui, val)
	case r.Slice:
		self.appendSlice(bui, val)
	default:
		bui.Any(val)
	}
}

func (self *Cond) appendEmpty(bui *Bui) {
	(*Seq)(self).appendEmpty(bui)
}

// TODO consider if we should support nested non-embedded structs.
func (self *Cond) appendStruct(bui *Bui, src any) {
	iter := makeIter(src)

	for iter.next() {
		if !iter.first() {
			bui.Str(self.Delim)
		}

		lhs := Ident(FieldDbName(iter.field))
		rhs := Eq{nil, iter.value.Interface()}

		// Equivalent to using `Eq` for the full expression, but avoids an
		// allocation caused by converting `Ident` to `Expr`. As a bonus, this also
		// avoids unnecessary parens around the ident.
		bui.Set(lhs.AppendExpr(bui.Get()))
		bui.Set(rhs.AppendRhs(bui.Get()))
	}

	if iter.empty() {
		self.appendEmpty(bui)
	}
}

func (self *Cond) appendSlice(bui *Bui, val any) {
	(*Seq)(self).appendSlice(bui, val)
}

/*
Represents a column list for a "select" expression. The inner value may be of
any type, and is used as a type carrier; its actual value is ignored. If the
inner value is a struct or struct slice, the resulting expression is a list of
column names corresponding to its fields, using a "db" tag. Otherwise the
expression is `*`.

Unlike many other struct-scanning expressions, this doesn't support filtering
via `Sparse`. It operates at the level of a struct type, not an individual
struct value.

TODO actually support `Sparse` because it's used for insert.
*/
type Cols [1]any

// Implement the `Expr` interface, making this a sub-expression.
func (self Cols) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return self.Append(text), args
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Cols) Append(text []byte) []byte {
	return appendMaybeSpaced(text, self.String())
}

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Cols) String() string {
	return TypeCols(r.TypeOf(self[0]))
}

/*
Represents a column list for a "select" expression. The inner value may be of
any type, and is used as a type carrier; its actual value is ignored. If the
inner value is a struct or struct slice, the resulting expression is a list of
column names corresponding to its fields, using a "db" tag. Otherwise the
expression is `*`.

Unlike `Cols`, this has special support for nested structs and nested column
paths. See the examples.

Unlike many other struct-scanning expressions, this doesn't support filtering
via `Sparse`. It operates at the level of a struct type, not an individual
struct value.
*/
type ColsDeep [1]any

// Implement the `Expr` interface, making this a sub-expression.
func (self ColsDeep) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return self.Append(text), args
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self ColsDeep) Append(text []byte) []byte {
	return appendMaybeSpaced(text, self.String())
}

// Implement the `fmt.Stringer` interface for debug purposes.
func (self ColsDeep) String() string {
	return TypeColsDeep(r.TypeOf(self[0]))
}

/*
Represents comma-separated values from the "db"-tagged fields of an arbitrary
struct. Field/column names are ignored. Values may be arbitrary sub-expressions
or arguments. The value passed to `StructValues` may be nil, which is
equivalent to an empty struct. It may also be an arbitrarily-nested struct
pointer, which is automatically dereferenced.

Supports filtering. If the inner value implements `Sparse`, then not all fields
are considered to be "present", which is useful for PATCH semantics. See the
docs on `Sparse` and `Part`.
*/
type StructValues [1]any

// Implement the `Expr` interface, making this a sub-expression.
func (self StructValues) AppendExpr(text []byte, args []any) ([]byte, []any) {
	bui := Bui{text, args}
	iter := makeIter(self[0])

	// TODO consider panicking when empty.
	for iter.next() {
		if !iter.first() {
			bui.Str(`,`)
		}
		bui.SubAny(iter.value.Interface())
	}

	return bui.Get()
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self StructValues) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self StructValues) String() string { return exprString(self) }

/*
Represents a names-and-values clause suitable for insertion. The inner value
must be nil or a struct. Nil or empty struct generates a "default values"
clause. Otherwise the resulting expression has SQL column names and values
generated by scanning the input struct. See the examples.

Supports filtering. If the inner value implements `Sparse`, then not all fields
are considered to be "present", which is useful for PATCH semantics. See the
docs on `Sparse` and `Part`.
*/
type StructInsert [1]any

// Implement the `Expr` interface, making this a sub-expression.
func (self StructInsert) AppendExpr(text []byte, args []any) ([]byte, []any) {
	bui := Bui{text, args}
	iter := makeIter(self[0])

	for iter.next() {
		if iter.first() {
			bui.Str(`(`)
			// TODO use `Cols` with support for `Sparse`.
			bui.Str(TypeCols(iter.root.Type()))
			bui.Str(`)`)
			bui.Str(`values (`)
		} else {
			bui.Str(`,`)
		}
		bui.SubAny(iter.value.Interface())
	}

	if iter.empty() {
		bui.Str(`default values`)
	} else {
		bui.Str(`)`)
	}

	return bui.Get()
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self StructInsert) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self StructInsert) String() string { return exprString(self) }

/*
Shortcut for creating `StructsInsert` from the given values.
Workaround for lack of type inference in type literals.
*/
func StructsInsertOf[A any](val ...A) StructsInsert[A] { return val }

/*
Variant of `StructInsert` that supports multiple structs. Generates a
names-and-values clause suitable for bulk insertion. The inner type must be a
struct. An empty slice generates an empty expression. See the examples.
*/
type StructsInsert[A any] []A

// Implement the `Expr` interface, making this a sub-expression.
func (self StructsInsert[A]) AppendExpr(text []byte, args []any) ([]byte, []any) {
	if len(self) == 0 {
		return text, args
	}

	bui := Bui{text, args}

	bui.Str(`(`)
	bui.Str(TypeCols(typeOf((*A)(nil))))
	bui.Str(`) values`)

	for ind, val := range self {
		if ind > 0 {
			bui.Str(`, `)
		}
		bui.Str(`(`)
		bui.Set(StructValues{val}.AppendExpr(bui.Get()))
		bui.Str(`)`)
	}

	return bui.Get()
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self StructsInsert[_]) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self StructsInsert[_]) String() string { return exprString(self) }

/*
Represents an SQL assignment clause suitable for "update set" operations. The
inner value must be a struct. The resulting expression consists of
comma-separated assignments with column names and values derived from the
provided struct. See the example.

Supports filtering. If the inner value implements `Sparse`, then not all fields
are considered to be "present", which is useful for PATCH semantics. See the
docs on `Sparse` and `Part`. If there are NO fields, panics with
`ErrEmptyAssign`, which can be detected by user code via `errors.Is`.
*/
type StructAssign [1]any

// Implement the `Expr` interface, making this a sub-expression.
func (self StructAssign) AppendExpr(text []byte, args []any) ([]byte, []any) {
	bui := Bui{text, args}
	iter := makeIter(self[0])

	for iter.next() {
		if !iter.first() {
			bui.Str(`,`)
		}
		bui.Set(Assign{
			Ident(FieldDbName(iter.field)),
			iter.value.Interface(),
		}.AppendExpr(bui.Get()))
	}

	if iter.empty() {
		panic(ErrEmptyAssign)
	}

	return bui.Get()
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self StructAssign) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self StructAssign) String() string { return exprString(self) }

/*
Wraps an arbitrary sub-expression, using `Cols{.Type}` to select specific
columns from it. If `.Type` doesn't specify a set of columns, for example
because it's not a struct type, then this uses the sub-expression as-is without
wrapping. Counterpart to `SelectColsDeep`.
*/
type SelectCols struct {
	From Expr
	Type any
}

// Implement the `Expr` interface, making this a sub-expression.
func (self SelectCols) AppendExpr(text []byte, args []any) ([]byte, []any) {
	// Type-to-string is nearly free due to caching.
	return SelectString{self.From, Cols{self.Type}.String()}.AppendExpr(text, args)
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self SelectCols) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self SelectCols) String() string { return exprString(self) }

/*
Wraps an arbitrary sub-expression, using `ColsDeep{.Type}` to select specific
columns from it. If `.Type` doesn't specify a set of columns, for example
because it's not a struct type, then this uses the sub-expression as-is without
wrapping. Counterpart to `SelectCols`.
*/
type SelectColsDeep struct {
	From Expr
	Type any
}

// Implement the `Expr` interface, making this a sub-expression.
func (self SelectColsDeep) AppendExpr(text []byte, args []any) ([]byte, []any) {
	// Type-to-string is nearly free due to caching.
	return SelectString{self.From, ColsDeep{self.Type}.String()}.AppendExpr(text, args)
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self SelectColsDeep) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self SelectColsDeep) String() string { return exprString(self) }

/*
Represents an SQL expression "select .What from (.From) as _". Mostly an
internal tool for building other expression types. Used internally by `Cols`
and `ColsDeep`; see their docs and examples.
*/
type SelectString struct {
	From Expr
	What string
}

// Implement the `Expr` interface, making this a sub-expression.
func (self SelectString) AppendExpr(text []byte, args []any) ([]byte, []any) {
	bui := Bui{text, args}

	if self.What == `*` {
		bui.Expr(self.From)
		return bui.Get()
	}

	if self.From != nil {
		bui.Str(`with _ as (`)
		bui.Expr(self.From)
		bui.Str(`)`)
	}

	bui.Str(`select`)
	bui.Str(self.What)

	if self.From != nil {
		bui.Str(`from _`)
	}

	return bui.Get()
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self SelectString) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self SelectString) String() string { return exprString(self) }

/*
Combines an expression with a string prefix. If the expr is nil, this is a nop,
and the prefix is ignored. Mostly an internal tool for building other
expression types.
*/
type Prefix struct {
	Prefix string
	Expr   Expr
}

// Implement the `Expr` interface, making this a sub-expression.
func (self Prefix) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return Wrap{self.Prefix, self.Expr, ``}.AppendExpr(text, args)
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Prefix) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Prefix) String() string { return exprString(self) }

/*
Combines an expression with a string prefix and suffix. If the expr is nil, this
is a nop, and the prefix and suffix are ignored. Mostly an internal tool for
building other expression types.
*/
type Wrap struct {
	Prefix string
	Expr   Expr
	Suffix string
}

// Difference from `Trio`: if the expr is nil, nothing is appended.
// Implement the `Expr` interface, making this a sub-expression.
func (self Wrap) AppendExpr(text []byte, args []any) ([]byte, []any) {
	bui := Bui{text, args}

	if self.Expr != nil {
		bui.Str(self.Prefix)
		bui.Expr(self.Expr)
		bui.Str(self.Suffix)
	}

	return bui.Get()
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Wrap) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Wrap) String() string { return exprString(self) }

/*
If the provided expression is not nil, prepends the keywords "order by" to it.
If the provided expression is nil, this is a nop.
*/
type OrderBy [1]Expr

// Implement the `Expr` interface, making this a sub-expression.
func (self OrderBy) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return Prefix{`order by`, self[0]}.AppendExpr(text, args)
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self OrderBy) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self OrderBy) String() string { return exprString(self) }

// Shortcut for simple "select * from A where B" expressions. See the examples.
type Select struct {
	From  Ident
	Where any
}

// Implement the `Expr` interface, making this a sub-expression.
func (self Select) AppendExpr(text []byte, args []any) ([]byte, []any) {
	bui := Bui{text, args}

	bui.Str(`select * from`)
	bui.Set(self.From.AppendExpr(bui.Get()))

	if self.Where != nil {
		bui.Str(`where`)
		bui.Set(And{self.Where}.AppendExpr(bui.Get()))
	}

	return bui.Get()
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Select) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Select) String() string { return exprString(self) }

// Shortcut for simple "insert into A (B) values (C) returning *" expressions.
// See the examples.
type Insert struct {
	Into   Ident
	Fields any
}

// Implement the `Expr` interface, making this a sub-expression.
func (self Insert) AppendExpr(text []byte, args []any) ([]byte, []any) {
	bui := Bui{text, args}

	bui.Str(`insert into`)
	bui.Set(self.Into.AppendExpr(bui.Get()))
	bui.Set(StructInsert{self.Fields}.AppendExpr(bui.Get()))
	bui.Str(`returning *`)

	return bui.Get()
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Insert) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Insert) String() string { return exprString(self) }

// Shortcut for simple "update A set B where C returning *" expressions. See the
// examples.
type Update struct {
	What   Ident
	Where  any
	Fields any
}

// Implement the `Expr` interface, making this a sub-expression.
func (self Update) AppendExpr(text []byte, args []any) ([]byte, []any) {
	bui := Bui{text, args}

	bui.Str(`update`)
	bui.Set(self.What.AppendExpr(bui.Get()))

	if self.Fields != nil {
		bui.Str(`set`)
		bui.Set(StructAssign{self.Fields}.AppendExpr(bui.Get()))
	}

	// TODO: when empty, panic with `ErrEmptyAssign` (rename to `ErrEmpty`).
	if self.Where != nil {
		bui.Str(`where`)
		bui.Set(Cond{`null`, `and`, self.Where}.AppendExpr(bui.Get()))
	}

	bui.Str(`returning *`)
	return bui.Get()
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Update) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Update) String() string { return exprString(self) }

// Shortcut for simple "delete from A where B returning *" expressions. See the examples.
type Delete struct {
	From  Ident
	Where any
}

// Implement the `Expr` interface, making this a sub-expression.
func (self Delete) AppendExpr(text []byte, args []any) ([]byte, []any) {
	bui := Bui{text, args}

	bui.Str(`delete from`)
	bui.Set(self.From.AppendExpr(bui.Get()))

	// TODO: when empty, panic with `ErrEmptyAssign` (rename to `ErrEmpty`).
	bui.Str(`where`)
	bui.Set(Cond{`null`, `and`, self.Where}.AppendExpr(bui.Get()))

	bui.Str(`returning *`)
	return bui.Get()
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Delete) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Delete) String() string { return exprString(self) }

/*
Represents an SQL upsert query like this:

	insert into some_table
		(key_0, key_1, col_2, col_3)
	values
		($1, $2, $3, $4)
	on conflict (key_0, key_1)
	do update set
		key_0 = excluded.key_0,
		key_1 = excluded.key_1,
		col_2 = excluded.col_2,
		col_3 = excluded.col_3
	returning *

Notes:

	* `.Keys` must be a struct.
	* `.Keys` supports `Sparse` and may be empty.
	* When `.Keys` is empty, this is equivalent to the `Insert` type.
	* `.Cols` must be a struct.
	* `.Cols` supports `Sparse` and may be empty.
	* `.Keys` provides names and values for key columns which participate
	  in the `on conflict` clause.
	* `.Cols` provides names and values for other columns.
*/
type Upsert struct {
	What Ident
	Keys any
	Cols any
}

// Implement the `Expr` interface, making this a sub-expression.
func (self Upsert) AppendExpr(text []byte, args []any) ([]byte, []any) {
	keysIter := makeIter(self.Keys)
	if !keysIter.has() {
		return Insert{self.What, self.Cols}.AppendExpr(text, args)
	}

	bui := Bui{text, args}
	colsIter := makeIter(self.Cols)

	bui.Str(`insert into`)
	bui.Set(self.What.AppendExpr(bui.Get()))

	// Set of column names for insertion.
	// Adapted from `StructInsert`.
	{
		bui.Str(`(`)
		bui.Str(TypeCols(keysIter.root.Type()))
		for colsIter.next() {
			bui.Str(`,`)
			Ident(FieldDbName(colsIter.field)).BuiAppend(&bui)
		}
		bui.Str(`)`)
	}

	bui.Str(`values`)

	// Set of column values for insertion.
	// Adapted from `StructInsert`.
	{
		bui.Str(`(`)
		keysIter.reinit()
		colsIter.reinit()

		for keysIter.next() {
			if !keysIter.first() {
				bui.Str(`,`)
			}
			bui.SubAny(keysIter.value.Interface())
		}

		for colsIter.next() {
			bui.Str(`,`)
			bui.SubAny(colsIter.value.Interface())
		}

		bui.Str(`)`)
	}

	// Conflict clause with key column names.
	{
		bui.Str(`on conflict (`)
		bui.Str(TypeCols(keysIter.root.Type()))
		bui.Str(`)`)
	}

	// Assignment clauses for all columns.
	{
		bui.Str(`do update set`)
		keysIter.reinit()
		colsIter.reinit()

		for keysIter.next() {
			if !keysIter.first() {
				bui.Str(`,`)
			}
			appendAssignExcluded(&bui, &keysIter)
		}

		for colsIter.next() {
			bui.Str(`,`)
			appendAssignExcluded(&bui, &colsIter)
		}
	}

	bui.Str(`returning *`)
	return bui.Get()
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Upsert) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Upsert) String() string { return exprString(self) }

func appendAssignExcluded(bui *Bui, iter *iter) {
	name := Ident(FieldDbName(iter.field))
	name.BuiAppend(bui)
	bui.Str(` = excluded.`)
	name.BuiAppend(bui)
}

/*
Shortcut for selecting `count(*)` from an arbitrary sub-expression. Equivalent
to `s.SelectString{expr, "count(*)"}`.
*/
type SelectCount [1]Expr

// Implement the `Expr` interface, making this a sub-expression.
func (self SelectCount) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return SelectString{self[0], `count(*)`}.AppendExpr(text, args)
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self SelectCount) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self SelectCount) String() string { return exprString(self) }

/*
Represents an SQL function call expression. The text prefix is optional and
usually represents a function name. The args must be either nil, a single
`Expr`, or a slice of arbitrary sub-expressions or arguments.
*/
type Call struct {
	Text string
	Args any
}

// Implement the `Expr` interface, making this a sub-expression.
func (self Call) AppendExpr(text []byte, args []any) ([]byte, []any) {
	bui := Bui{text, args}
	bui.Str(self.Text)

	// TODO: when `self.Args` is a single expression, consider always additionally
	// parenthesizing it. `Comma` doesn't do that.
	bui.Str(`(`)
	bui.Set(Comma{self.Args}.AppendExpr(bui.Get()))
	bui.Str(`)`)

	return bui.Get()
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Call) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Call) String() string { return exprString(self) }

/*
Represents the Postgres window function `row_number`:

	RowNumberOver{}
	-> `0`

	RowNumberOver{Ords{OrdDesc{Ident(`some_col`)}}}
	-> `row_number() over (order by "col" desc)`

When the inner expression is nil and the output is `0`, the Postgres query
planner should be able to optimize it away.
*/
type RowNumberOver [1]Expr

// Implement the `Expr` interface, making this a sub-expression.
func (self RowNumberOver) AppendExpr(text []byte, args []any) ([]byte, []any) {
	if self[0] == nil {
		return appendMaybeSpaced(text, `0`), args
	}

	bui := Bui{text, args}
	bui.Str(`row_number() over (`)
	bui.Expr(self[0])
	bui.Str(`)`)

	return bui.Get()
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self RowNumberOver) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self RowNumberOver) String() string { return exprString(self) }

/*
Short for "string query". Represents an SQL query with parameters such as "$1"
or ":param_name". Args may be a list of ordinal args (via `List`), a dictionary
(via `Dict`), a struct (via `StructDict`), or an arbitrary user-defined
implementation conforming to the interface. When generating the final
expression, parameters are converted to Postgres-style ordinal parameters such
as "$1".

Expressions/queries are composable. Named arguments that implement the `Expr`
interface do not become ordinal parameters/arguments. Instead, they're treated
as sub-expressions, and may include arbitrary text with their own arguments.
Parameter collisions between outer and inner queries are completely avoided.

Uses `Preparse` to avoid redundant parsing. Each source string is parsed only
once, and the resulting `Prep` is cached. As a result, `StrQ` has little
measurable overhead.
*/
type StrQ struct {
	Text string
	Args ArgDict
}

// Implement the `Expr` interface, making this a sub-expression.
func (self StrQ) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return Preparse(self.Text).AppendParamExpr(text, args, self.Args)
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self StrQ) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self StrQ) String() string { return exprString(self) }

/*
Short for "preparsed" or "prepared". Partially parsed representation of
parametrized SQL expressions, suited for efficiently building SQL queries by
providing arguments. Supports both ordinal and named parameters/arguments. To
avoid redundant work, this should be parsed and cached only once for each SQL
query; this deduplication is done by `Preparse` which is also used internally
by `StrQ`. User code doesn't need to construct this.
*/
type Prep struct {
	Source    string
	Tokens    []Token
	HasParams bool
}

// Parses `self.Source`, modifying the receiver. Panics if parsing fails.
func (self *Prep) Parse() {
	/**
	This has all sorts of avoidable allocations, and could be significantly better
	optimized. However, parsing is ALWAYS slow. We cache the resulting `Prep`
	for each source string to avoid redundant parsing.
	*/

	src := strings.TrimSpace(self.Source)
	tok := Tokenizer{Source: src, Transform: trimWhitespaceAndComments}

	// Suboptimal, could be avoided.
	buf := make([]byte, 0, 128)

	flush := func() {
		if len(buf) > 0 {
			// Suboptimal. It would be better to reslice the source string instead of
			// allocating new strings.
			self.Tokens = append(self.Tokens, Token{string(buf), TokenTypeText})
		}
		buf = buf[:0]
	}

	for {
		tok := tok.Next()
		if tok.IsInvalid() {
			break
		}

		switch tok.Type {
		case TokenTypeOrdinalParam, TokenTypeNamedParam:
			flush()
			self.Tokens = append(self.Tokens, tok)
			self.HasParams = true

		default:
			buf = append(buf, tok.Text...)
		}
	}

	flush()
}

// Implement the `ParamExpr` interface. Builds the expression by using the
// provided named args. Used internally by `StrQ`.
func (self Prep) AppendParamExpr(text []byte, args []any, dict ArgDict) ([]byte, []any) {
	if !self.HasParams {
		return self.appendUnparametrized(text, args, dict)
	}
	return self.appendParametrized(text, args, dict)
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Prep) Append(text []byte) []byte {
	return Str(self.Source).Append(text)
}

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Prep) String() string { return self.Source }

func (self Prep) appendUnparametrized(text []byte, args []any, dict ArgDict) ([]byte, []any) {
	src := self.Source
	if !isNil(dict) {
		panic(errUnexpectedArgs(fmt.Sprintf(`non-parametrized expression %q`, src), dict))
	}
	return Str(src).AppendExpr(text, args)
}

func (self Prep) appendParametrized(text []byte, args []any, dict ArgDict) ([]byte, []any) {
	if dict == nil {
		panic(errMissingArgs(fmt.Sprintf(`parametrized expression %q`, self.Source)))
	}

	bui := Bui{text, args}
	bui.Grow(len(self.Source), dict.Len())

	tracker := getArgTracker()
	defer tracker.put()

	for _, tok := range self.Tokens {
		switch tok.Type {
		case TokenTypeOrdinalParam:
			// Parsing the token here is slightly suboptimal, we should preparse numbers.
			appendOrdinal(&bui, dict, tracker, tok.ParseOrdinalParam())

		case TokenTypeNamedParam:
			appendNamed(&bui, dict, tracker, tok.ParseNamedParam())

		default:
			bui.Text = append(bui.Text, tok.Text...)
		}
	}

	tracker.validate(dict)
	return bui.Get()
}

/**
The implementation of both `appendOrdinal` and `appendNamed` should be roughly
equivalent to `bui.Any(arg)`, but more efficient for parameters that occur more
than once. We map each SOURCE PARAMETER to exactly one TARGET PARAMETER and one
ARGUMENT. If it was simply appended via `bui.Any(arg)`, then every occurrence
would generate another argument.

The cost of keeping track of found parameters is amortized by recycling them in
a pool. It saves us the cost of redundant encoding of those arguments, which is
potentially much larger, for example when an argument is a huge array.

Keeping track of found PARAMETERS also allows us to validate that all ARGUMENTS
are used.
*/
func appendOrdinal(bui *Bui, args ArgDict, tracker *argTracker, key OrdinalParam) {
	arg, ok := args.GotOrdinal(key.Index())
	if !ok {
		panic(errMissingOrdinal(key))
	}

	impl, _ := arg.(Expr)
	if impl != nil {
		// Allows validation of used args.
		tracker.SetOrdinal(key, 0)
		bui.Expr(impl)
		return
	}

	ord, ok := tracker.GotOrdinal(key)
	if ok {
		bui.OrphanParam(ord)
		return
	}

	ord = bui.OrphanArg(arg)
	bui.OrphanParam(ord)
	tracker.SetOrdinal(key, ord)
}

func appendNamed(bui *Bui, args ArgDict, tracker *argTracker, key NamedParam) {
	arg, ok := args.GotNamed(key.Key())
	if !ok {
		panic(errMissingNamed(key))
	}

	impl, _ := arg.(Expr)
	if impl != nil {
		// Allows validation of used args.
		tracker.SetNamed(key, 0)
		bui.Expr(impl)
		return
	}

	ord, ok := tracker.GotNamed(key)
	if ok {
		bui.OrphanParam(ord)
		return
	}

	ord = bui.OrphanArg(arg)
	bui.OrphanParam(ord)
	tracker.SetNamed(key, ord)
}

// Represents an ordinal parameter such as "$1". Mostly for internal use.
type OrdinalParam int

// Implement the `Expr` interface, making this a sub-expression.
func (self OrdinalParam) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return self.Append(text), args
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self OrdinalParam) Append(text []byte) []byte {
	text = append(text, ordinalParamPrefix)
	text = strconv.AppendInt(text, int64(self), 10)
	return text
}

// Implement the `fmt.Stringer` interface for debug purposes.
func (self OrdinalParam) String() string { return AppenderString(&self) }

// Returns the corresponding Go index (starts at zero).
func (self OrdinalParam) Index() int { return int(self) - 1 }

// Inverse of `OrdinalParam.Index`: increments by 1, converting index to param.
func (self OrdinalParam) FromIndex() OrdinalParam { return self + 1 }

// Represents a named parameter such as ":blah". Mostly for internal use.
type NamedParam string

// Implement the `Expr` interface, making this a sub-expression.
func (self NamedParam) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return self.Append(text), args
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self NamedParam) Append(text []byte) []byte {
	text = append(text, namedParamPrefix)
	text = append(text, self...)
	return text
}

// Implement the `fmt.Stringer` interface for debug purposes.
func (self NamedParam) String() string { return AppenderString(&self) }

// Converts to the corresponding dictionary key, which is a plain string. This
// is a free cast, used to increase code clarity.
func (self NamedParam) Key() string { return string(self) }

/*
Represents SQL expression "limit N" with an arbitrary argument or
sub-expression. Implements `Expr`:

  * If nil  -> append nothing.
  * If expr -> append "limit (<sub-expression>)".
  * If val  -> append "limit $N" with the corresponding argument.
*/
type Limit [1]any

// Implement the `Expr` interface, making this a sub-expression.
func (self Limit) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return appendPrefixSub(text, args, `limit`, self[0])
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Limit) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Limit) String() string { return AppenderString(&self) }

/*
Represents SQL expression "offset N" with an arbitrary sub-expression.
Implements `Expr`:

  * If nil  -> append nothing.
  * If expr -> append "offset (<sub-expression>)".
  * If val  -> append "offset $N" with the corresponding argument.
*/
type Offset [1]any

// Implement the `Expr` interface, making this a sub-expression.
func (self Offset) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return appendPrefixSub(text, args, `offset`, self[0])
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Offset) Append(text []byte) []byte { return exprAppend(self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Offset) String() string { return AppenderString(&self) }

/*
Represents SQL expression "limit N" with a number. Implements `Expr`:

	* If 0      -> append nothing.
	* Otherwise -> append literal "limit <N>" such as "limit 1".

Because this is uint64, you can safely and correctly decode arbitrary user input
into this value, for example into a struct field of this type.
*/
type LimitUint uint64

// Implement the `Expr` interface, making this a sub-expression.
func (self LimitUint) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return self.Append(text), args
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self LimitUint) Append(text []byte) []byte {
	return appendIntWith(text, `limit`, int64(self))
}

// Implement the `fmt.Stringer` interface for debug purposes.
func (self LimitUint) String() string { return AppenderString(&self) }

/*
Represents SQL expression "offset N" with a number. Implements `Expr`:

	* If 0      -> append nothing.
	* Otherwise -> append literal "offset <N>" such as "offset 1".

Because this is uint64, you can safely and correctly decode arbitrary user input
into this value, for example into a struct field of this type.
*/
type OffsetUint uint64

// Implement the `Expr` interface, making this a sub-expression.
func (self OffsetUint) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return self.Append(text), args
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self OffsetUint) Append(text []byte) []byte {
	return appendIntWith(text, `offset`, int64(self))
}

// Implement the `fmt.Stringer` interface for debug purposes.
func (self OffsetUint) String() string { return AppenderString(&self) }
