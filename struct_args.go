package sqlb

import (
	"regexp"
	"strconv"

	"github.com/mitranim/refut"
)

/*
Scans a struct, accumulating fields tagged with `db` into a map suitable for
`Query.AppendNamed()`. The input must be a struct or a struct pointer. A nil
pointer is fine and produces a nil result. Panics on other inputs. Treats
embedded structs as part of enclosing structs.
*/
func StructMap(input interface{}) map[string]interface{} {
	dict := map[string]interface{}{}
	traverseStructDbFields(input, func(name string, value interface{}) {
		dict[name] = value
	})
	return dict
}

/*
Scans a struct, converting fields tagged with `db` into a sequence of named
`NamedArgs`. The input must be a struct or a struct pointer. A nil pointer is
fine and produces a nil result. Panics on other inputs. Treats embedded structs
part of enclosing structs.
*/
func StructNamedArgs(input interface{}) NamedArgs {
	var args NamedArgs
	traverseStructDbFields(input, func(name string, value interface{}) {
		args = append(args, Named(name, value))
	})
	return args
}

/*
Sequence of named SQL arguments with utility methods for query building. Usually
obtained by calling `StructNamedArgs()`.
*/
type NamedArgs []NamedArg

/*
Returns a query whose string representation is suitable for an SQL `select`
clause. Should be included into other queries via `Query.Append()` or
`Query.AppendNamed()`.

For example, this:

	val := struct {
		One int64 `db:"one"`
		Two int64 `db:"two"`
	}{
		One: 10,
		Two: 20,
	}

	text := StructNamedArgs(val).Names().String()

Is equivalent to:

	text := `"one", "two"`
*/
func (self NamedArgs) Names() Query {
	var buf []byte

	for i, arg := range self {
		if i > 0 {
			buf = append(buf, `, `...)
		}
		buf = appendDelimited(buf, `"`, arg.Name, `"`)
	}

	return queryFrom(bytesToMutableString(buf), nil)
}

/*
Returns a query whose string representation is suitable for an SQL `values()`
clause, with arguments. Should be included into other queries via
`Query.Append()` or `Query.AppendNamed()`.

For example, this:

	val := struct {
		One int64 `db:"one"`
		Two int64 `db:"two"`
	}{
		One: 10,
		Two: 20,
	}

	query := StructNamedArgs(val).Values()
	text := query.String()
	args := query.Args

Is equivalent to:

	text := `$1, $2`
	args := []interface{}{10, 20}
*/
func (self NamedArgs) Values() Query {
	args := make([]interface{}, 0, len(self))
	var buf []byte

	for i, arg := range self {
		if i > 0 {
			buf = append(buf, `, `...)
		}
		buf = append(buf, `$`...)
		buf = strconv.AppendInt(buf, int64(i+1), 10)
		args = append(args, arg.Value)
	}

	return queryFrom(bytesToMutableString(buf), args)
}

/*
Returns a query whose string representation is suitable for an SQL `insert`
clause, with arguments. Should be included into other queries via
`Query.Append()` or `Query.AppendNamed()`.

For example, this:

	val := struct {
		One int64 `db:"one"`
		Two int64 `db:"two"`
	}{
		One: 10,
		Two: 20,
	}

	query := StructNamedArgs(val).NamesAndValues()
	text := query.String()
	args := query.Args

Is equivalent to:

	text := `("one", "two") values ($1, $2)`
	args := []interface{}{10, 20}
*/
func (self NamedArgs) NamesAndValues() Query {
	if len(self) == 0 {
		return queryFrom("default values", nil)
	}

	args := make([]interface{}, 0, len(self))
	var buf []byte

	buf = append(buf, `(`...)

	for i, arg := range self {
		if i > 0 {
			buf = append(buf, `, `...)
		}
		buf = appendDelimited(buf, `"`, arg.Name, `"`)
	}

	buf = append(buf, `) values (`...)

	for i, arg := range self {
		if i > 0 {
			buf = append(buf, `, `...)
		}
		buf = append(buf, `$`...)
		buf = strconv.AppendInt(buf, int64(i+1), 10)

		args = append(args, arg.Value)
	}

	buf = append(buf, `)`...)

	return queryFrom(bytesToMutableString(buf), args)
}

/*
Returns a query whose string representation is suitable for an SQL `update set`
clause, with arguments. Should be included into other queries via
`Query.Append()` or `Query.AppendNamed()`.

For example, this:

	val := struct {
		One int64 `db:"one"`
		Two int64 `db:"two"`
	}{
		One: 10,
		Two: 20,
	}

	query := StructNamedArgs(val).Assignments()
	text := query.String()
	args := query.Args

Is equivalent to:

	text := `"one" = $1, "two" = $2`
	args := []interface{}{10, 20}
*/
func (self NamedArgs) Assignments() Query {
	args := make([]interface{}, 0, len(self))
	var buf []byte

	for i, arg := range self {
		if i > 0 {
			buf = append(buf, `, `...)
		}
		buf = appendDelimited(buf, `"`, arg.Name, `"`)
		buf = append(buf, ` = $`...)
		buf = strconv.AppendInt(buf, int64(i+1), 10)
		args = append(args, arg.Value)
	}

	return queryFrom(bytesToMutableString(buf), args)
}

/*
Returns a query whose string representation is suitable for an SQL `where` or
`on` clause, with arguments. Should be included into other queries via
`Query.Append()` or `Query.AppendNamed()`.

For example, this:

	val := struct {
		One int64 `db:"one"`
		Two int64 `db:"two"`
	}{
		One: 10,
		Two: 20,
	}

	query := StructNamedArgs(val).Conditions()
	text := query.String()
	args := query.Args

Is equivalent to:

	text := `"one" is not distinct from $1 and "two" is not distinct from $2`
	args := []interface{}{10, 20}
*/
func (self NamedArgs) Conditions() Query {
	if len(self) == 0 {
		return queryFrom("true", nil)
	}

	args := make([]interface{}, 0, len(self))
	var buf []byte

	for i, arg := range self {
		if i > 0 {
			buf = append(buf, ` and `...)
		}
		buf = appendDelimited(buf, `"`, arg.Name, `"`)
		buf = append(buf, ` is not distinct from $`...)
		buf = strconv.AppendInt(buf, int64(i+1), 10)
		args = append(args, arg.Value)
	}

	return queryFrom(bytesToMutableString(buf), args)
}

/*
Returns true if at least one argument satisfies the predicate function. Example:

  ok := args.Some(NamedArg.IsNil)
*/
func (self NamedArgs) Some(fun func(NamedArg) bool) bool {
	for _, arg := range self {
		if fun != nil && fun(arg) {
			return true
		}
	}
	return false
}

/*
Returns true if every argument satisfies the predicate function. Example:

  ok := args.Every(NamedArg.IsNil)
*/
func (self NamedArgs) Every(fun func(NamedArg) bool) bool {
	for _, arg := range self {
		if fun == nil || !fun(arg) {
			return false
		}
	}
	return true
}

// Same as `sql.NamedArg`, with additional methods. See `NamedArgs`.
type NamedArg struct {
	Name  string
	Value interface{}
}

func Named(name string, value interface{}) NamedArg {
	return NamedArg{Name: name, Value: value}
}

func (self NamedArg) IsValid() bool {
	return columnNameRegexp.MatchString(self.Name)
}

// WTF is this?
var columnNameRegexp = regexp.MustCompile(`^(?:\w+(?:\.\w+)*|"\w+(?:\.\w+)*")$`)

func (self NamedArg) IsNil() bool {
	return refut.IsNil(self.Value)
}
