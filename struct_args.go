package sqlb

import (
	"database/sql/driver"

	"github.com/mitranim/refut"
)

/*
Scans a struct, accumulating fields tagged with `db` into a map suitable for
`Query.AppendNamed`. The input must be a struct or a struct pointer. A nil
pointer is fine and produces an empty non-nil map. Panics on other inputs.
Treats embedded structs as part of enclosing structs.
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
as part of enclosing structs.
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
clause. Should be included into other queries via `Query.Append` or
`Query.AppendNamed`.

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
	var query Query
	self.queryAppendNames(&query)
	return query
}

func (self NamedArgs) queryAppendNames(query *Query) {
	for i, arg := range self {
		if i > 0 {
			appendStr(&query.Text, `, `)
		}
		arg.queryAppendName(query)
	}
}

/*
Returns a query whose string representation is suitable for an SQL `values()`
clause, with arguments. Should be included into other queries via
`Query.Append` or `Query.AppendNamed`.

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
	query := Query{Args: make([]interface{}, 0, len(self))}
	self.queryAppendValues(&query)
	return query
}

func (self NamedArgs) queryAppendValues(query *Query) {
	for i, arg := range self {
		if i > 0 {
			appendStr(&query.Text, `, `)
		}
		arg.queryAppendValue(query)
	}
}

/*
Returns a query whose string representation is suitable for an SQL `insert`
clause, with arguments. Should be included into other queries via
`Query.Append` or `Query.AppendNamed`.

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
		return Query{Text: []byte(`default values`)}
	}

	query := Query{Args: make([]interface{}, 0, len(self))}

	appendStr(&query.Text, `(`)
	self.queryAppendNames(&query)
	appendStr(&query.Text, `) values (`)
	self.queryAppendValues(&query)
	appendStr(&query.Text, `)`)

	return query
}

/*
Returns a query whose string representation is suitable for an SQL `update set`
clause, with arguments. Should be included into other queries via
`Query.Append` or `Query.AppendNamed`.

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

Known issue: when empty, this generates an empty query which is invalid SQL.
Don't use this when `NamedArgs` is empty.
*/
func (self NamedArgs) Assignments() Query {
	query := Query{Args: make([]interface{}, 0, len(self))}

	for i, arg := range self {
		if i > 0 {
			appendStr(&query.Text, `, `)
		}
		arg.queryAppendName(&query)
		query.Append(`= $1`, arg.Value)
	}

	return query
}

/*
Returns a query whose string representation is suitable for an SQL `where` or
`on` clause, with arguments. Should be included into other queries via
`Query.Append` or `Query.AppendNamed`.

For example, this:

	val := struct {
		One   int64  `db:"one"`
		Two   int64  `db:"two"`
		Three *int64 `db:"three"`
	}{
		One: 10,
		Two: 20,
	}

	query := StructNamedArgs(val).Conditions()
	text := query.String()
	args := query.Args

Is equivalent to:

	text := `"one" = $1 and "two" = $2 and "three" is null`
	args := []interface{}{10, 20}
*/
func (self NamedArgs) Conditions() Query {
	if len(self) == 0 {
		return Query{Text: []byte(`true`)}
	}

	var query Query
	for i, arg := range self {
		if i > 0 {
			query.Append(`and`)
		}
		arg.queryAppendCondition(&query)
	}
	return query
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

// Convenience function for creating a named arg without struct field labels.
func Named(name string, value interface{}) NamedArg {
	return NamedArg{Name: name, Value: value}
}

// Same as `sql.NamedArg`, with additional methods. See `NamedArgs`.
type NamedArg struct {
	Name  string
	Value interface{}
}

// Normalizes the inner value by attempting SQL encoding. Used internally for
// detecting nils, which influences `NamedArgs.Conditions`.
func (self NamedArg) Norm() (interface{}, error) {
	val := self.Value

	valuer, ok := val.(driver.Valuer)
	var err error
	if ok {
		if refut.IsNil(valuer) {
			return nil, nil
		}

		val, err = valuer.Value()
		if err != nil {
			return nil, err
		}
	}

	return normNil(val), nil
}

/*
Returns true if the value would be equivalent to `null` in SQL. Caution: this is
NOT the same as comparing the value to `nil`:

	NamedArg{}.Value == nil                      // true
	NamedArg{}.IsNil()                           // true

	NamedArg{Value: (*string)(nil)}.Value == nil // false
	NamedArg{Value: (*string)(nil)}.IsNil()      // true
*/
func (self NamedArg) IsNil() bool {
	val, _ := self.Norm()
	return val == nil
}

func (self NamedArg) queryAppendName(query *Query) {
	appendSpaceIfNeeded(&query.Text)
	appendEnclosed(&query.Text, `"`, self.Name, `"`)
}

func (self NamedArg) queryAppendValue(query *Query) {
	query.Append(`$1`, self.Value)
}

func (self NamedArg) queryAppendCondition(query *Query) {
	val, err := self.Norm()
	try(err)

	self.queryAppendName(query)

	if val == nil {
		appendStr(&query.Text, ` is null`)
	} else {
		query.Append(` = $1`, val)
	}
}
