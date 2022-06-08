package sqlb

import (
	"encoding/json"
	r "reflect"
	"unsafe"
)

/*
Options related to parsing text into `Ords`. Used by `ParserOrds` and
`OrdsParser`.
*/
type ParseOpt struct {
	/**
	Must be a struct type. Ords parsing uses this to detect which fields are
	allowed, and to convert JSON field names to DB column names.
	*/
	Type r.Type

	/**
	Optional filter. When non-nil, this is invoked for each struct field during
	ords parsing. If this returns false, the field is "unknown" and may generate
	a parse error depending on `.Lax`.
	*/
	Filter Filter

	/**
	When true, unknown JSON fields are skipped/ignored durung parsing. When false,
	unknown JSON fields cause ords parsing to fail with a descriptive error.
	*/
	Lax bool
}

/*
If `.Type` is empty, sets the type of the provided value. Otherwise this is a
nop. The input is used only as a type carrier; its actual value is ignored. The
type is consulted when decoding orderings from an input such as JSON.
*/
func (self *ParseOpt) OrType(typ any) {
	if self.Type == nil {
		self.Type = typeElemOf(typ)
	}
}

/*
Contains `Ords` and parsing options, and implements decoder interfaces such as
`json.Unmarshaler`. Intended to be included into other structs. Unmarshals text
into inner `Ords` in accordance with the parsing options.
*/
type ParserOrds struct {
	Ords
	ParseOpt
}

/*
Implement `json.Unmarshaler`. Consults `.Type` to determine known field paths,
and converts them to DB column paths, rejecting unknown identifiers. The JSON
input must represent an array of strings. See the method `.ParseSlice` for more
docs.
*/
func (self *ParserOrds) UnmarshalJSON(src []byte) error {
	return OrdsParser{&self.Ords, self.ParseOpt}.UnmarshalJSON(src)
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
func (self *ParserOrds) ParseSlice(src []string) error {
	return OrdsParser{&self.Ords, self.ParseOpt}.ParseSlice(src)
}

/*
Similar to `ParserOrds`, but intended to be transient and stackframe-local,
rather than included into other types. Usually obtained by calling
`(*Ords).OrdsParser`.
*/
type OrdsParser struct {
	*Ords
	ParseOpt
}

// Implement `json.Unmarshaler`. See `(*ParserOrds).UnmarshalJSON` for docs.
func (self OrdsParser) UnmarshalJSON(src []byte) (err error) {
	defer rec(&err)
	var vals []string
	try(json.Unmarshal(src, &vals))
	self.noescape().parseSlice(vals)
	return
}

// See `(*ParserOrds).ParseSlice` for docs.
func (self OrdsParser) ParseSlice(src []string) (err error) {
	defer rec(&err)
	self.noescape().parseSlice(src)
	return
}

func (self *OrdsParser) parseSlice(src []string) {
	self.Zero()
	self.Grow(countNonEmptyStrings(src))
	for _, val := range src {
		self.parseAppend(val)
	}
}

func (self *OrdsParser) parseAppend(src string) {
	if src == `` {
		return
	}

	/**
	This regexp-based parsing is simple to implement, but extremely inefficient,
	easily trouncing all our optimizations. TODO rewrite in an efficient manner.
	*/
	match := ordReg.FindStringSubmatch(src)
	if match == nil {
		panic(errInvalidOrd(src))
	}

	typ := self.Type
	pathStr := match[1]
	entry, ok := loadStructJsonPathToDbPathFieldValueMap(typ)[pathStr]

	if !ok || !self.filter(entry.Field) {
		if self.Lax {
			return
		}
		panic(errUnknownField(`converting JSON identifier path to DB path`, pathStr, typeName(typ)))
	}

	dir := strDir(match[2])
	if dir == DirNone {
		def := entry.Field.Tag.Get(`ord.dir`)
		if def != `` {
			dir = strDir(def)
		}
	}

	nulls := strNulls(match[3])
	if nulls == NullsNone {
		def := entry.Field.Tag.Get(`ord.nulls`)
		if def != `` {
			nulls = strNulls(def)
		}
	}

	path := entry.Value

	/**
	This weird trickery saves some allocations. If we had unwrapped the concrete
	type `[]string` or `Path`, converted it to another concrete type such as
	`OrdAsc`, and then converted back to an indirect `Expr`, the final
	conversion would allocate an exact copy of the original slice header, even
	though due to being stored behind an interface it's immutable and still
	points to the same original backing array. As far as I'm concerned, that's a
	language defect. This is a workaround.
	*/
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

	self.Add(path.Interface().(Expr))
}

func (self *OrdsParser) filter(field r.StructField) bool {
	return self.Filter == nil || self.Filter.AllowField(field)
}

// Prevents a weird spurious escape that shows up in benchmarks.
func (self *OrdsParser) noescape() *OrdsParser {
	return (*OrdsParser)(noescape(unsafe.Pointer(self)))
}

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
