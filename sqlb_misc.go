package sqlb

import (
	"fmt"
	r "reflect"
)

const (
	TagNameDb   = `db`
	TagNameJson = `json`
)

/*
Encodes the provided expressions and returns the resulting text and args.
Shortcut for using `(*Bui).Exprs` and `Bui.Reify`. Provided mostly for
examples. Actual code may want to use `Bui`:

	bui := MakeBui(4096, 64)
	panic(bui.CatchExprs(someExprs...))
	text, args := bui.Reify()
*/
func Reify(vals ...Expr) (string, []any) {
	var bui Bui
	bui.Exprs(vals...)
	return bui.Reify()
}

/*
Returns the output of `Cols` for the given type, but takes `reflect.Type` as
input, rather than a type-carrying `any`. Used internally by `Cols`.
The result is cached and reused. Subsequent calls for the same type are nearly
free.
*/
func TypeCols(typ r.Type) string {
	return colsCache.Get(typeElem(typ))
}

/*
Returns the output of `ColsDeep` for the given type, but takes `reflect.Type` as
input, rather than a type-carrying `any`. Used internally by
`ColsDeep`. The result is cached and reused. Subsequent calls for the same type
are nearly free.
*/
func TypeColsDeep(typ r.Type) string {
	return colsDeepCache.Get(typeElem(typ))
}

/*
Returns a parsed `Prep` for the given source string. Panics if parsing fails.
Caches the result for each source string, reusing it for future calls. Used
internally by `StrQ`. User code shouldn't have to call this, but it's exported
just in case.
*/
func Preparse(val string) Prep {
	return prepCache.Get(val)
}

// Shortcut for `StrQ{text, List(args)}`.
func ListQ(text string, args ...any) StrQ {
	if len(args) == 0 {
		return StrQ{text, nil}
	}
	return StrQ{text, List(args)}
}

// Shortcut for `StrQ{text, Dict(args)}`.
func DictQ(text string, args map[string]any) StrQ {
	if len(args) == 0 {
		return StrQ{text, nil}
	}
	return StrQ{text, Dict(args)}
}

// Shortcut for `StrQ{text, StructDict{reflect.ValueOf(args)}}`.
func StructQ(text string, args any) StrQ {
	val := valueOf(args)
	if !val.IsValid() {
		return StrQ{text, nil}
	}
	return StrQ{text, StructDict{val}}
}

// Returns the field's DB column name from the "db" tag, following the JSON
// convention of eliding anything after a comma and treating "-" as a
// non-name.
func FieldDbName(field r.StructField) string {
	return tagIdent(field.Tag.Get(TagNameDb))
}

// Returns the field's JSON column name from the "json" tag, following the same
// conventions as the `encoding/json` package.
func FieldJsonName(field r.StructField) string {
	return tagIdent(field.Tag.Get(TagNameJson))
}

const (
	DirNone Dir = 0
	DirAsc  Dir = 1
	DirDesc Dir = 2
)

// Short for "direction". Enum for ordering direction: none, "asc", "desc".
type Dir byte

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Dir) Append(text []byte) []byte {
	return appendMaybeSpaced(text, self.String())
}

// Implement `fmt.Stringer` for debug purposes.
func (self Dir) String() string {
	switch self {
	default:
		return ``
	case DirAsc:
		return `asc`
	case DirDesc:
		return `desc`
	}
}

// Parses from a string, which must be either empty, "asc" or "desc".
func (self *Dir) Parse(src string) error {
	switch src {
	case ``:
		*self = DirNone
		return nil
	case `asc`:
		*self = DirAsc
		return nil
	case `desc`:
		*self = DirDesc
		return nil
	default:
		return ErrInvalidInput{Err{
			`parsing order direction`,
			fmt.Errorf(`unrecognized direction %q`, src),
		}}
	}
}

// Implement `encoding.TextUnmarshaler`.
func (self Dir) MarshalText() ([]byte, error) {
	return stringToBytesUnsafe(self.String()), nil
}

// Implement `encoding.TextMarshaler`.
func (self *Dir) UnmarshalText(src []byte) error {
	return self.Parse(bytesToMutableString(src))
}

// Implement `json.Marshaler`.
func (self Dir) MarshalJSON() ([]byte, error) {
	switch self {
	default:
		return stringToBytesUnsafe(`null`), nil
	case DirAsc:
		return stringToBytesUnsafe(`"asc"`), nil
	case DirDesc:
		return stringToBytesUnsafe(`"desc"`), nil
	}
}

// Implement `fmt.GoStringer` for debug purposes. Returns valid Go code
// representing this value.
func (self Dir) GoString() string {
	switch self {
	default:
		return `sqlb.DirNone`
	case DirAsc:
		return `sqlb.DirAsc`
	case DirDesc:
		return `sqlb.DirDesc`
	}
}

const (
	NullsNone  Nulls = 0
	NullsFirst Nulls = 1
	NullsLast  Nulls = 2
)

// Enum for nulls handling in ordering: none, "nulls first", "nulls last".
type Nulls byte

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Nulls) Append(text []byte) []byte {
	return appendMaybeSpaced(text, self.String())
}

// Implement `fmt.Stringer` for debug purposes.
func (self Nulls) String() string {
	switch self {
	case NullsFirst:
		return `nulls first`
	case NullsLast:
		return `nulls last`
	default:
		return ``
	}
}

// Implement `fmt.GoStringer` for debug purposes. Returns valid Go code
// representing this value.
func (self Nulls) GoString() string {
	switch self {
	case NullsFirst:
		return `sqlb.NullsFirst`
	case NullsLast:
		return `sqlb.NullsLast`
	default:
		return `sqlb.NullsNone`
	}
}

/*
Implements `Sparse` by filtering fields on their JSON names, using only
explicit "json" tags. Fields without explicit "json" names are automatically
considered missing. Fields with "json" tags must be present in the provided
string set represented by `.Fil`.

Designed for compatibility with HTTP request decoders provided
by "github.com/mitranim/rd", which either implement `Haser` or can easily
generate one. Example PATCH endpoint using "rd":

	import "github.com/mitranim/rd"
	import "github.com/mitranim/try"
	import s "github.com/mitranim/sqlb"

	dec := rd.TryDownload(req)

	var input SomeStructType
	try.To(dec.Decode(&input))

	expr := s.Exprs{
		s.Update{s.Ident(`some_table`)},
		s.Set{s.StructAssign{s.Partial{input, dec.Haser()}}},
	}
*/
type Partial struct {
	Val any
	Fil Haser
}

var _ = Sparse(Partial{})

// Implement `Sparse`, returning the underlying value.
func (self Partial) Get() any { return self.Val }

// Implement `Sparse`, using the underlying filter.
func (self Partial) AllowField(field r.StructField) bool {
	name := FieldJsonName(field)
	return name != `` && self.Fil != nil && self.Fil.Has(name)
}

/*
Implements `Filter` by requiring that the struct field has this specific tag.
The tag's value for any given field is ignored, only its existence is checked.
*/
type TagFilter string

func (self TagFilter) AllowField(field r.StructField) bool {
	_, ok := field.Tag.Lookup(string(self))
	return ok
}
