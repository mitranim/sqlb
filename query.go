package sqlb

import (
	"fmt"

	"github.com/mitranim/sqlp"
)

/*
If true (default), unused query parameters cause panics in functions like
`Query.Append`. If false, unused parameters are ok. Turning this off can be
convenient in development, when changing queries rapidly.
*/
var CheckUnused = true

/*
Interface that allows compatibility between different query variants. Subquery
insertion / flattening, supported by `Query.Append()` and `Query.AppendNamed()`,
detects instances of this interface, rather than the concrete type `Query`,
allowing external code to implement its own variants, wrap `Query`, etc.

WTB better name.
*/
type IQuery interface{ QueryAppend(*Query) }

/*
Tool for building SQL queries. Makes it easy to append or insert arbitrary SQL
code while avoiding common errors. Contains both query content (as parsed AST)
and arguments.

Automatically renumerates ordinal placeholders when appending code, making it
easy to avoid mis-numbering. See `.Append()`.

Supports named parameters. See `.AppendNamed()`.

Composable: both `.Append()` and `.AppendNamed()` automatically interpolate
sub-queries found in the arguments, combining the arguments and renumerating the
parameters as appropriate.

Currently biased towards Postgres-style ordinal parameters of the form `$N`. The
code is always converted to this "canonical" form. This can be rectified if
there is enough demand; you can open an issue at
https://github.com/mitranim/sqlb/issues.
*/
type Query struct {
	Text []byte
	Args []interface{}
}

// Implement `fmt.Stringer`.
func (self Query) String() string {
	return bytesToMutableString(self.Text)
}

/*
Implement `IQuery`, allowing compatibility between different implementations,
wrappers, etc.
*/
func (self Query) QueryAppend(out *Query) {
	out.Append(bytesToMutableString(self.Text), self.Args...)
}

/*
Appends code and arguments. Renumerates ordinal parameters, offsetting them by
the previous argument count. The count in the code always starts from `$1`.

Composable: automatically interpolates any instances of `IQuery` found in the
arguments, combining the arguments and renumerating the parameters as
appropriate.

For example, this:

	var query Query
	query.Append(`where true`)
	query.Append(`and one = $1`, 10)
	query.Append(`and two = $1`, 20) // Note the $1.

	text := query.String()
	args := query.Args

Is equivalent to this:

	text := `where true and one = $1 and two = $2`
	args := []interface{}{10, 20}

Panics when: the code is malformed; the code has named parameters; a parameter
doesn't have a corresponding argument; an argument doesn't have a corresponding
parameter.
*/
func (self *Query) Append(src string, args ...interface{}) {
	tokenizer := sqlp.Tokenizer{Source: src}
	startOffset := len(self.Args)
	appendNonQueries(&self.Args, args)

	var used bitset
	if len(args) > bitsetSize {
		panic(Err{
			Code:  ErrCodeTooManyArguments,
			While: `appending to query`,
			Cause: fmt.Errorf(`expected no more than %v args, got %v`, bitsetSize, len(args)),
		})
	}

	appendSpaceIfNeeded(&self.Text)

	for {
		node := tokenizer.Next()
		if node == nil {
			break
		}

		switch node := node.(type) {
		case sqlp.NodeOrdinalParam:
			index := node.Index()
			if index >= len(args) {
				panic(Err{
					Code:  ErrCodeOrdinalOutOfBounds,
					While: `appending to query`,
					Cause: fmt.Errorf(`ordinal parameter %v exceeds argument count %v`, node, len(args)),
				})
			}

			used.set(index)
			query, ok := args[index].(IQuery)
			if ok {
				query.QueryAppend(self)
			} else {
				ord := sqlp.NodeOrdinalParam(int(node) + startOffset - queryArgsBefore(args, node.Index()))
				ord.Append(&self.Text)
			}

		case sqlp.NodeNamedParam:
			panic(Err{
				Code:  ErrCodeUnexpectedParameter,
				While: `appending to query`,
				Cause: fmt.Errorf(`expected only ordinal params, got named param %q`, node),
			})

		default:
			node.Append(&self.Text)
		}
	}

	if CheckUnused {
		for i, arg := range args {
			if !used.has(i) {
				panic(Err{
					Code:  ErrCodeUnusedArgument,
					While: `appending to query`,
					Cause: fmt.Errorf(`unused argument %#v at index %v`, arg, i),
				})
			}
		}
	}
}

/*
Appends code and named arguments. The code must have named parameters in the
form ":identifier". The keys in the arguments map must have the form
"identifier", without a leading ":".

Internally, converts named parameters to ordinal parameters of the form `$N`,
such as the ones used by `.Append()`.

Composable: automatically interpolates any instances of `IQuery` found in the
arguments, combining the arguments and renumerating the parameters as
appropriate.

For example, this:

	var query Query
	query.AppendNamed(
		`select col where col = :value`,
		map[string]interface{}{"value": 10},
	)

	text := query.String()
	args := query.Args

Is equivalent to this:

	text := `select col where col = $1`
	args := []interface{}{10}

Panics when: the code is malformed; the code has ordinal parameters; a parameter
doesn't have a corresponding argument; an argument doesn't have a corresponding
parameter.
*/
func (self *Query) AppendNamed(src string, args map[string]interface{}) {
	tokenizer := sqlp.Tokenizer{Source: src}
	namedToOrd := make(map[sqlp.NodeNamedParam]sqlp.NodeOrdinalParam, len(args))
	appendSpaceIfNeeded(&self.Text)

	for {
		node := tokenizer.Next()
		if node == nil {
			break
		}

		switch node := node.(type) {
		case sqlp.NodeOrdinalParam:
			panic(Err{
				Code:  ErrCodeUnexpectedParameter,
				While: `appending to query`,
				Cause: fmt.Errorf(`expected only named params, got ordinal param %q`, node),
			})

		case sqlp.NodeNamedParam:
			arg, found := args[string(node)]
			if !found {
				panic(Err{
					Code:  ErrCodeMissingArgument,
					While: `appending to query`,
					Cause: fmt.Errorf(`missing named argument %q`, node),
				})
			}

			query, ok := arg.(IQuery)
			if ok {
				// Value doesn't matter. This allows detection of unused arguments.
				namedToOrd[node] = 0
				query.QueryAppend(self)
				continue
			}

			ord, ok := namedToOrd[node]
			if !ok {
				self.Args = append(self.Args, arg)
				ord = sqlp.NodeOrdinalParam(len(self.Args))
				namedToOrd[node] = ord
			}
			ord.Append(&self.Text)

		default:
			node.Append(&self.Text)
		}
	}

	if CheckUnused {
		for key := range args {
			_, ok := namedToOrd[sqlp.NodeNamedParam(key)]
			if !ok {
				panic(Err{
					Code:  ErrCodeUnusedArgument,
					While: `appending to query`,
					Cause: fmt.Errorf(`unused named argument %q`, key),
				})
			}
		}
	}
}

/*
Convenience method, inverse of `IQuery.QueryAppend`. Appends the other query to
this one, combining the arguments and renumerating the ordinal parameters as
appropriate.
*/
func (self *Query) AppendQuery(query IQuery) {
	if query != nil {
		query.QueryAppend(self)
	}
}

/*
"Zeroes" the query, keeping any already-allocated capacity. Similar to
`query = sqlb.Query{}`, but slightly clearer and marginally more efficient for
subsequent query building.
*/
func (self *Query) Clear() {
	self.Text = self.Text[:0]
	self.Args = self.Args[:0]
}

/*
Wraps the query to select only the specified expressions.

For example, this:

	var query Query
	query.Append(`select * from some_table`)
	query.WrapSelect(`one, two`)

	text := query.String()

Is equivalent to this:

	text := `with _ as (select * from some_table) select one, two from _`
*/
func (self *Query) WrapSelect(exprs string) {
	const (
		s0 = `with _ as (`
		s1 = `) select `
		s2 = ` from _`
	)

	buf := make([]byte, 0, len(s0)+len(self.Text)+len(s1)+len(exprs)+len(s2))
	appendStr(&buf, s0)
	buf = append(buf, self.Text...)
	appendStr(&buf, s1)
	appendStr(&buf, exprs)
	appendStr(&buf, s2)

	self.Text = buf
}

/*
Wraps the query to select the fields derived by calling `Cols(dest)`.

For example, this:

	var query SqlQuery
	query.Append(`select * from some_table`)

	var out struct{Id int64 `db:"id"`}
	query.WrapSelectCols(out)

	text := query.String()

Is equivalent to this:

	text := `with _ as (select * from some_table) select "id" from _`
*/
func (self *Query) WrapSelectCols(dest interface{}) {
	self.WrapSelect(Cols(dest))
}
