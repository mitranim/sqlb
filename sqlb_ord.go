package sqlb

import (
	"encoding/json"
)

/*
Short for "orderings". Sequence of arbitrary expressions used for an SQL
"order by" clause. Nil elements are treated as non-existent. If there are no
non-nil elements, the resulting expression is empty. Otherwise, the resulting
expression is "order by" followed by comma-separated sub-expressions. You can
construct `Ords` manually, or parse client inputs via `OrdsParser`. See the
examples.
*/
type Ords []Expr

/*
Allows types that embed `Ords` to behave like a slice in JSON encoding, avoiding
some edge issues. For example, this allows an empty `ParserOrds` to be encoded
as JSON `null` rather than a struct, allowing types that include it as a field
to be used for encoding JSON, not just decoding it. However, this doesn't make
ords encoding/decoding actually reversible. Decoding "consults" a struct type
to convert JSON field names to DB column names. Ideally, JSON marshaling would
perform the same process in reverse, which is not currently implemented.
*/
func (self Ords) MarshalJSON() ([]byte, error) {
	return json.Marshal([]Expr(self))
}

/*
Returns an `OrdsParser` that can decode arbitrary JSON or a string slice into
the given `*Ords` pointer. Initializes the parser to the provided type, using
`typ` only as a type carrier.
*/
func (self *Ords) OrdsParser(typ any) (out OrdsParser) {
	out.OrType(typ)
	out.Ords = self
	return
}

// Implement the `Expr` interface, making this a sub-expression.
func (self Ords) AppendExpr(text []byte, args []any) ([]byte, []any) {
	bui := Bui{text, args}
	var found bool

	for _, val := range self {
		if val == nil {
			continue
		}

		if !found {
			found = true
			bui.Str(`order by`)
		} else {
			bui.Str(`,`)
		}

		bui.Expr(val)
	}

	return bui.Get()
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Ords) Append(text []byte) []byte { return exprAppend(&self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Ords) String() string { return exprString(&self) }

/*
Returns an expression for the Postgres window function `row_number`:

	Ords{}.RowNumber()
	-> `0`

	Ords{OrdAsc(`col`)}.RowNumber()
	-> `row_number() over (order by "col" asc)`

As shown above, empty `Ords` generates `0`. The Postgres query planner
should optimize away any ordering by this constant column.
*/
func (self Ords) RowNumberOver() RowNumberOver {
	if self.IsEmpty() {
		return RowNumberOver{}
	}
	return RowNumberOver{self}
}

// Returns true if there are no non-nil items.
func (self Ords) IsEmpty() bool { return self.Len() == 0 }

// Returns the amount of non-nil items.
func (self Ords) Len() (count int) {
	for _, val := range self {
		if val != nil {
			count++
		}
	}
	return
}

/*
Empties the receiver. If the receiver was non-nil, its length is reduced to 0
while keeping any capacity, and it remains non-nil.
*/
func (self *Ords) Zero() {
	if self != nil && *self != nil {
		*self = (*self)[:0]
	}
}

// Convenience method for appending.
func (self *Ords) Add(vals ...Expr) {
	for _, val := range vals {
		if val != nil {
			*self = append(*self, val)
		}
	}
}

// If empty, sets the given vals. Otherwise it's a nop.
func (self *Ords) Or(vals ...Expr) {
	if self.IsEmpty() {
		self.Zero()
		self.Add(vals...)
	}
}

// Resizes to ensure that space capacity is `<= size`.
func (self *Ords) Grow(size int) {
	*self = growExprs(*self, size)
}

// Sometimes handy for types that embed `Ords`.
func (self *Ords) OrdsPtr() *Ords { return self }

/*
Structured representation of an arbitrary SQL ordering expression. This is not
the entire "order by" clause (see `Ords`), but rather just one element in that
clause. This is the general-case representation, but because most ordering
expressions use only column names and direction, a more specialized
representation is preferred: `Ord`. This is provided just-in-case.
*/
type Ordering struct {
	Expr  Expr
	Dir   Dir
	Nulls Nulls
	Using Expr
}

// Implement the `Expr` interface, making this a sub-expression.
func (self Ordering) AppendExpr(text []byte, args []any) ([]byte, []any) {
	if self.Expr == nil {
		return text, args
	}

	text, args = self.Expr.AppendExpr(text, args)
	text = self.Dir.Append(text)
	text = self.Nulls.Append(text)

	if self.Using != nil {
		text = appendMaybeSpaced(text, `using `)
		text, args = self.Using.AppendExpr(text, args)
	}

	return text, args
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Ordering) Append(text []byte) []byte { return exprAppend(&self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Ordering) String() string { return exprString(&self) }

/*
Structured representation of an arbitrary SQL ordering expression. This is not
the entire "order by" clause (see `Ords`), but rather just one element in that
clause. Also see `Ords`, `OrdsParser`, and the various provided examples.
*/
type Ord struct {
	Path  Path
	Dir   Dir
	Nulls Nulls
}

// Implement the `Expr` interface, making this a sub-expression.
func (self Ord) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return self.Append(text), args
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Ord) Append(text []byte) []byte {
	if len(self.Path) > 0 {
		text = self.Path.Append(text)
		text = self.Dir.Append(text)
		text = self.Nulls.Append(text)
	}
	return text
}

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Ord) String() string { return AppenderString(&self) }

// True if the path is empty.
func (self Ord) IsEmpty() bool { return len(self.Path) == 0 }

// Same as `Ord{Path: path, Dir: DirAsc}` but more syntactically convenient
// and uses less memory.
type OrdAsc []string

// Implement the `Expr` interface, making this a sub-expression.
func (self OrdAsc) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return Ord{Path: Path(self), Dir: DirAsc}.AppendExpr(text, args)
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self OrdAsc) Append(text []byte) []byte { return exprAppend(&self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self OrdAsc) String() string { return exprString(&self) }

// Same as `Ord{Path: path, Dir: DirDesc}` but more syntactically
// convenient and uses less memory.
type OrdDesc []string

// Implement the `Expr` interface, making this a sub-expression.
func (self OrdDesc) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return Ord{Path: Path(self), Dir: DirDesc}.AppendExpr(text, args)
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self OrdDesc) Append(text []byte) []byte { return exprAppend(&self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self OrdDesc) String() string { return exprString(&self) }

// Same as `Ord{Path: path, Nulls: NullsFirst}` but more syntactically
// convenient and uses less memory.
type OrdNullsFirst []string

// Implement the `Expr` interface, making this a sub-expression.
func (self OrdNullsFirst) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return Ord{Path: Path(self), Nulls: NullsFirst}.AppendExpr(text, args)
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self OrdNullsFirst) Append(text []byte) []byte { return exprAppend(&self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self OrdNullsFirst) String() string { return exprString(&self) }

// Same as `Ord{Path: path, Nulls: NullsLast}` but more syntactically
// convenient and uses less memory.
type OrdNullsLast []string

// Implement the `Expr` interface, making this a sub-expression.
func (self OrdNullsLast) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return Ord{Path: Path(self), Nulls: NullsLast}.AppendExpr(text, args)
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self OrdNullsLast) Append(text []byte) []byte { return exprAppend(&self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self OrdNullsLast) String() string { return exprString(&self) }

// Same as `Ord{Path: path, Dir: DirAsc, Nulls: NullsFirst}` but more
// syntactically convenient and uses less memory.
type OrdAscNullsFirst []string

// Implement the `Expr` interface, making this a sub-expression.
func (self OrdAscNullsFirst) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return Ord{Path: Path(self), Dir: DirAsc, Nulls: NullsFirst}.AppendExpr(text, args)
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self OrdAscNullsFirst) Append(text []byte) []byte { return exprAppend(&self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self OrdAscNullsFirst) String() string { return exprString(&self) }

// Same as `Ord{Path: path, Dir: DirAsc, Nulls: NullsLast}` but more
// syntactically convenient and uses less memory.
type OrdAscNullsLast []string

// Implement the `Expr` interface, making this a sub-expression.
func (self OrdAscNullsLast) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return Ord{Path: Path(self), Dir: DirAsc, Nulls: NullsLast}.AppendExpr(text, args)
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self OrdAscNullsLast) Append(text []byte) []byte { return exprAppend(&self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self OrdAscNullsLast) String() string { return exprString(&self) }

// Same as `Ord{Path: path, Dir: DirDesc, Nulls: NullsFirst}` but more
// syntactically convenient and uses less memory.
type OrdDescNullsFirst []string

// Implement the `Expr` interface, making this a sub-expression.
func (self OrdDescNullsFirst) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return Ord{Path: Path(self), Dir: DirDesc, Nulls: NullsFirst}.AppendExpr(text, args)
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self OrdDescNullsFirst) Append(text []byte) []byte { return exprAppend(&self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self OrdDescNullsFirst) String() string { return exprString(&self) }

// Same as `Ord{Path: path, Dir: DirDesc, Nulls: NullsLast}` but more
// syntactically convenient and uses less memory.
type OrdDescNullsLast []string

// Implement the `Expr` interface, making this a sub-expression.
func (self OrdDescNullsLast) AppendExpr(text []byte, args []any) ([]byte, []any) {
	return Ord{Path: Path(self), Dir: DirDesc, Nulls: NullsLast}.AppendExpr(text, args)
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self OrdDescNullsLast) Append(text []byte) []byte { return exprAppend(&self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self OrdDescNullsLast) String() string { return exprString(&self) }
