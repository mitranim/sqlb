package sqlb

import (
	"encoding/json"
	"fmt"
	r "reflect"
)

/*
Parses text into `Ords`. Supports parsing JSON and arbitrary string slices, such
as from `url.Values`. External input is a list of ordering strings. Each
ordering string is parsed against the struct type stored in `OrdsParser.Type`.
The type must be provided before parsing any inputs.

`OrdsParser.Type` serves two purposes: mapping JSON field names to DB column
names, and whitelisting. The identifiers in the input must match JSON field
names or paths (for nested structs). The output contains only DB column names.
By default, any unknown identifiers in the input cause a parsing error. Setting
`OrdsParser.Lax = true` causes unknown identifiers to be completely ignored.

While `Ords` can technically contain arbitrary `Expr` values, the `Ords`
generated by `OrdsParser` consists only of specialized ordering types such as
`OrdAsc`, `OrdDesc`, etc.

See the examples for `(*OrdsParser).ParseSlice` and `(*OrdsParser).UnmarshalJSON`.
*/
type OrdsParser struct {
	Ords
	Type r.Type
	Lax  bool
}

/*
Shortcut for empty `OrdsParser` intended for parsing. The input is used only as
a type carrier. The parsing process will consult the provided type. See the
example on `OrdsParser`.
*/
func OrdsParserFor(val interface{}) (out OrdsParser) {
	out.OrType(val)
	return
}

/*
Implement decoding from JSON. Consults `.Type` to determine known field paths,
and converts them to DB column paths, rejecting unknown identifiers.
*/
func (self *OrdsParser) UnmarshalJSON(input []byte) error {
	var vals []string
	err := json.Unmarshal(input, &vals)
	if err != nil {
		return err
	}
	return self.ParseSlice(vals)
}

/*
Parses a string slice which must consist of individual ordering strings such
as "one.two.three desc". Ignores empty strings. Used internally for parsing
JSON. String slices may also come from URL queries, form-encoded data, and so
on. Supported input format:

	<path> <asc|desc>? <nulls first | nulls last>?

Each path can be a single identifier or dot-separated:

	one
	one.two
	one.two.three

The path MUST correspond to JSON-tagged fields in the reference struct type,
which MUST have corresponding DB column names. The parsed ordering uses DB
column names, rather than the original JSON names.
*/
func (self *OrdsParser) ParseSlice(vals []string) error {
	self.Ords = self.Ords.Grow(countNonEmptyStrings(vals))[:0]

	for _, val := range vals {
		err := self.parseAppend(val)
		if err != nil {
			return err
		}
	}

	return nil
}

func (self *OrdsParser) parseAppend(src string) (err error) {
	defer rec(&err)

	if src == `` {
		return nil
	}

	match := ordReg.FindStringSubmatch(src)
	if match == nil {
		return ErrInvalidInput{Err{
			`parsing ordering expression`,
			fmt.Errorf(
				`%q is not a valid ordering string; expected format: "<ident> (asc|desc)? (nulls (?:first|last))?"`,
				src,
			),
		}}
	}

	typ := self.Type
	pathStr := match[1]
	path, ok := loadStructJsonPathToDbPathReflectValueMap(typ)[pathStr]

	if !ok || !path.IsValid() {
		if self.Lax {
			return nil
		}
		panic(errUnknownField(`converting JSON identifier path to DB path`, pathStr, typeName(typ)))
	}

	dir := strDir(match[2])
	nulls := strNulls(match[3])

	// This weird trickery saves some allocations. If we had unwrapped the
	// concrete type `[]string` or `Path`, converted it to another concrete type,
	// and then converted back to an indirect `Expr`, the final conversion would
	// allocate an exact copy of the original slice header, even though it's
	// immutable and still points to the same original backing array. As far as
	// I'm concerned, that's a language defect. This is a workaround.
	switch {
	case dir == DirAsc && nulls == NullsFirst:
		path = path.Convert(typOrdAscNullsFirst)
	case dir == DirAsc && nulls == NullsLast:
		path = path.Convert(typOrdAscNullsLast)
	case dir == DirDesc && nulls == NullsFirst:
		path = path.Convert(typOrdDescNullsFirst)
	case dir == DirDesc && nulls == NullsLast:
		path = path.Convert(typOrdDescNullsLast)
	case dir == DirAsc:
		path = path.Convert(typOrdAsc)
	case dir == DirDesc:
		path = path.Convert(typOrdDesc)
	case nulls == NullsFirst:
		path = path.Convert(typOrdNullsFirst)
	case nulls == NullsLast:
		path = path.Convert(typOrdNullsLast)
	default:
		path = path.Convert(typPath)
	}

	self.Ords.AppendVals(path.Interface().(Expr))
	return nil
}

/*
If `.Type` is empty, sets the type of the provided value. Otherwise this is a
nop. The input is used only as a type carrier; its actual value is ignored. The
type is consulted when decoding orderings from an input such as JSON.
*/
func (self *OrdsParser) OrType(typ interface{}) {
	if self.Type == nil {
		self.Type = typeElemOf(typ)
	}
}

/*
Short for "orderings". Sequence of arbitrary expressions used for an SQL
"order by" clause. Nil elements are treated as non-existent. If there are no
non-nil elements, the resulting expression is empty. Otherwise, the resulting
expression is "order by" followed by comma-separated sub-expressions. You can
construct `Ords` manually or via `OrdsParser`. See the examples.
*/
type Ords []Expr

// Implement the `Expr` interface, making this a sub-expression.
func (self Ords) AppendExpr(text []byte, args []interface{}) ([]byte, []interface{}) {
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

// Convenience method for appending.
func (self *Ords) AppendVals(vals ...Expr) {
	for _, val := range vals {
		if val != nil {
			*self = append(*self, val)
		}
	}
}

// If empty, sets the given vals. Otherwise it's a nop.
func (self *Ords) Or(vals ...Expr) {
	if self.IsEmpty() {
		*self = (*self)[:0]
		self.AppendVals(vals...)
	}
}

// Resizes to ensure that space capacity is `<= size`, returning the modified
// version.
func (self Ords) Grow(size int) Ords {
	// Copied from `growBytes`. WTB generics.

	len, cap := len(self), cap(self)
	if cap-len >= size {
		return self
	}

	next := make(Ords, len, 2*cap+size)
	copy(next, self)
	return next
}

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
func (self Ordering) AppendExpr(text []byte, args []interface{}) ([]byte, []interface{}) {
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
func (self Ord) AppendExpr(text []byte, args []interface{}) ([]byte, []interface{}) {
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
func (self OrdAsc) AppendExpr(text []byte, args []interface{}) ([]byte, []interface{}) {
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
func (self OrdDesc) AppendExpr(text []byte, args []interface{}) ([]byte, []interface{}) {
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
func (self OrdNullsFirst) AppendExpr(text []byte, args []interface{}) ([]byte, []interface{}) {
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
func (self OrdNullsLast) AppendExpr(text []byte, args []interface{}) ([]byte, []interface{}) {
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
func (self OrdAscNullsFirst) AppendExpr(text []byte, args []interface{}) ([]byte, []interface{}) {
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
func (self OrdAscNullsLast) AppendExpr(text []byte, args []interface{}) ([]byte, []interface{}) {
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
func (self OrdDescNullsFirst) AppendExpr(text []byte, args []interface{}) ([]byte, []interface{}) {
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
func (self OrdDescNullsLast) AppendExpr(text []byte, args []interface{}) ([]byte, []interface{}) {
	return Ord{Path: Path(self), Dir: DirDesc, Nulls: NullsLast}.AppendExpr(text, args)
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self OrdDescNullsLast) Append(text []byte) []byte { return exprAppend(&self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self OrdDescNullsLast) String() string { return exprString(&self) }

var (
	typOrdAsc            = r.TypeOf((*OrdAsc)(nil)).Elem()
	typOrdDesc           = r.TypeOf((*OrdDesc)(nil)).Elem()
	typOrdNullsFirst     = r.TypeOf((*OrdNullsFirst)(nil)).Elem()
	typOrdNullsLast      = r.TypeOf((*OrdNullsLast)(nil)).Elem()
	typOrdAscNullsFirst  = r.TypeOf((*OrdAscNullsFirst)(nil)).Elem()
	typOrdAscNullsLast   = r.TypeOf((*OrdAscNullsLast)(nil)).Elem()
	typOrdDescNullsFirst = r.TypeOf((*OrdDescNullsFirst)(nil)).Elem()
	typOrdDescNullsLast  = r.TypeOf((*OrdDescNullsLast)(nil)).Elem()
	typPath              = r.TypeOf((*Path)(nil)).Elem()
)