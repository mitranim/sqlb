package sqlb

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

/*
Short for "orderings". Structured representation of an SQL ordering such as:

	`order by "some_col" asc`

	`order by "some_col" asc, "nested"."other_col" desc`

For flexibility, the sequence of `Ords` may include arbitrary SQL expressions
expressed as `IQuery` instances. But when decoding external input, every
created element is an instance of `Ord`.

When encoding an `Ord` to a string, identifiers are quoted for safety. An
ordering with empty `.Items` represents no ordering: "".

`.Type` is used for parsing external input. It must be a struct type. Every
field name or path must be found in the struct type, possibly in nested
structs. The decoding process will convert every JSON field name into the
corresponding DB column name. Identifiers without the corresponding pair of
`json` and `db` tags cause a parse error.

Usage for parsing:

	input := []byte(`["one asc", "two.three desc"]`)

	ords := OrdsFor(SomeStructType{})

	err := ords.UnmarshalJSON(input)
	panic(err)

The result is equivalent to:

	OrdsFrom(OrdAsc(`one`), OrdDesc(`two`, `three`))

Usage for SQL:

	text, args := ords.Query()

`Ords` implements `IQuery` and can be directly used as a sub-query:

	var query Query
	query.Append(`select from where $1`, OrdsFrom(OrdAsc(`some_col`)))
*/
type Ords struct {
	Items []IQuery
	Type  reflect.Type
}

// Shortcut for creating `Ords` without a type.
func OrdsFrom(items ...IQuery) Ords { return Ords{Items: items} }

/*
Shortcut for empty `Ords` intended for parsing. The input is used only as a type
carrier. The parsing process will consult the provided type; see
`Ords.UnmarshalJSON`.
*/
func OrdsFor(val interface{}) Ords { return Ords{Type: reflect.TypeOf(val)} }

/*
Implement decoding from JSON. Consults `.Type` to determine known field paths,
and converts them to DB column paths, rejecting unknown identifiers.
*/
func (self *Ords) UnmarshalJSON(input []byte) error {
	var vals []string
	err := json.Unmarshal(input, &vals)
	if err != nil {
		return err
	}
	return self.ParseSlice(vals)
}

/*
Convenience method for parsing string slices, which may come from URL queries,
form-encoded data, and so on.
*/
func (self *Ords) ParseSlice(vals []string) error {
	self.Items = make([]IQuery, 0, len(vals))

	for _, val := range vals {
		var ord Ord
		err := self.parseOrd(val, &ord)
		if err != nil {
			return err
		}
		self.Items = append(self.Items, ord)
	}

	return nil
}

func (self Ords) parseOrd(str string, ord *Ord) error {
	match := ordReg.FindStringSubmatch(str)
	if match == nil {
		return fmt.Errorf(`[sqlb] %q is not a valid ordering string; expected format: "<ident> (asc|desc)? (nulls last)?"`, str)
	}

	_, path, err := structFieldByJsonPath(self.Type, match[1])
	if err != nil {
		return err
	}

	ord.Path = path
	ord.Desc = strings.EqualFold(match[2], `desc`)
	ord.NullsLast = match[3] != ""
	return nil
}

/*
Implement `IQuery`, allowing this to be used as a sub-query for `Query`. When
used as an argument for `Query.Append` or `Query.AppendNamed`, this will be
automatically interpolated.
*/
func (self Ords) QueryAppend(out *Query) {
	first := true

	for _, val := range self.Items {
		if val == nil {
			continue
		}

		if first {
			out.Append(`order by `)
			first = false
		} else {
			appendStr(&out.Text, `, `)
		}

		val.QueryAppend(out)
	}
}

/*
Returns a query for the Postgres window function `row_number`:

	OrdsFrom().RowNumber()
	-> `0`

	OrdsFrom(OrdAsc(`col`)).RowNumber()
	-> `row_number() over (order by "col" asc nulls last)`

As shown above, an empty `Ords` generates a constant `0`. The Postgres query
planner should optimize away any ordering by this constant column.
*/
func (self Ords) RowNumber() Query {
	var query Query

	if self.IsEmpty() {
		query.Append(`0`)
	} else {
		query.Append(`row_number() over (`)
		self.QueryAppend(&query)
		query.Append(`)`)
	}

	return query
}

// Returns true if there are no non-nil items.
func (self Ords) IsEmpty() bool { return self.Len() == 0 }

// Returns the amount of non-nil items.
func (self Ords) Len() (count int) {
	for _, val := range self.Items {
		if val != nil {
			count++
		}
	}
	return
}

// Convenience method for appending.
func (self *Ords) Append(items ...IQuery) {
	self.Items = append(self.Items, items...)
}

// If empty, replaces items with the provided fallback. Otherwise does nothing.
func (self *Ords) Or(items ...IQuery) {
	if self.IsEmpty() {
		self.Items = items
	}
}

/*
Shortcut:

	OrdAsc(`one`, `two) ≡ Ord{Path: []string{`one`, `two`}}
*/
func OrdAsc(path ...string) Ord { return Ord{Path: path} }

/*
Shortcut:

	OrdDesc(`one`, `two) ≡ Ord{Path: []string{`one`, `two`}, Desc: true}
*/
func OrdDesc(path ...string) Ord { return Ord{Path: path, Desc: true} }

/*
Shortcut:

	OrdAscNl(`one`, `two) ≡ Ord{Path: []string{`one`, `two`}, NullsLast: true}
*/
func OrdAscNl(path ...string) Ord { return Ord{Path: path, NullsLast: true} }

/*
Shortcut:

	OrdDescNl(`one`, `two) ≡ Ord{Path: []string{`one`, `two`}, Desc: true, NullsLast: true}
*/
func OrdDescNl(path ...string) Ord { return Ord{Path: path, Desc: true, NullsLast: true} }

/*
Short for "ordering". Describes an SQL ordering like:

	`"some_col" asc`

	`("nested")."other_col" desc`

but in a structured format. When encoding for SQL, identifiers are quoted for
safety. Identifier case is preserved. Parsing of keyword such
as "asc", "desc", "nulls last" is case-insensitive and non-case-preserving
since they're converted to bools.

Note on `Desc`: the default value `false` corresponds to "ascending", which is
the default in SQL.

Also see `Ords`.
*/
type Ord struct {
	Path      []string
	Desc      bool
	NullsLast bool
}

/*
Returns an SQL string like:

	"some_col" asc

	("some_col")."other_col" asc
*/
func (self Ord) String() string {
	var buf []byte
	self.AppendBytes(&buf)
	return bytesToMutableString(buf)
}

// Appends an SQL string to the buffer. See `.String()`.
func (self Ord) AppendBytes(buf *[]byte) {
	appendSqlPath(buf, self.Path)

	if self.Desc {
		appendStr(buf, " desc")
	} else {
		appendStr(buf, " asc")
	}

	if self.NullsLast {
		appendStr(buf, " nulls last")
	}
}

// Implement `IQuery`, allowing this to be placed in `Ords`.
func (self Ord) QueryAppend(out *Query) {
	self.AppendBytes(&out.Text)
}
