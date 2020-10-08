package sqlb

import (
	"fmt"

	"github.com/mitranim/sqlp"
)

/*
Interface that allows compatibility between different query variants. Subquery
insertion / flattening, supported by `Query.Append()` and `Query.AppendNamed()`,
detects instances of this interface, rather than the concrete type `Query`,
allowing external code to implement its own variants, wrap `Query`, etc.

WTB better name.
*/
type IQuery interface {
	Unwrap() (sqlp.Nodes, []interface{})
}

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
	sqlp.Nodes
	Args []interface{}
}

/*
Implement `IQuery`, allowing compatibility between different implementations,
wrappers, etc.
*/
func (self Query) Unwrap() (sqlp.Nodes, []interface{}) {
	return self.Nodes, self.Args
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
func (self *Query) Append(code string, args ...interface{}) {
	nodes, err := sqlp.Parse(code)
	must(err)
	must(self.append(nodes, args))
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
	query.AppendNamed(`select col where col = :value`, map[string]interface{}{
		"value": 10,
	})

	text := query.String()
	args := query.Args

Is equivalent to this:

	text := `select col where col = $1`
	args := []interface{}{10}

Panics when: the code is malformed; the code has ordinal parameters; a parameter
doesn't have a corresponding argument; an argument doesn't have a corresponding
parameter.
*/
func (self *Query) AppendNamed(code string, namedArgs map[string]interface{}) {
	nodes, err := sqlp.Parse(code)
	must(err)

	args, err := namedToOrdinal(nodes, namedArgs)
	must(err)

	must(self.append(nodes, args))
}

/*
Appends the other query's AST and arguments to the end of this query while
renumerating the ordinal parameters as appropriate.
*/
func (self *Query) AppendQuery(query IQuery) {
	nodes, args := query.Unwrap()
	nodes = sqlp.CopyDeep(nodes)
	must(self.append(nodes, args))
}

/*
Makes a copy that doesn't share any mutable state with the original. Useful when
you want to "fork" a query and modify both versions.
*/
func (self Query) Copy() Query {
	return Query{
		Nodes: sqlp.CopyDeep(self.Nodes),
		Args:  copyIfaceSlice(self.Args),
	}
}

/*
"Zeroes" the query by resetting inner data structures to `len() == 0` while
keeping their capacity. Similar to `query = sqlb.Query{}`, but slightly clearer
and marginally more efficient for subsequent query building.
*/
func (self *Query) Clear() {
	self.Nodes = self.Nodes[:0]
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
	self.Nodes = sqlp.Nodes{
		sqlp.NodeText(`with _ as (`),
		self.Nodes,
		sqlp.NodeText(`) select ` + exprs + ` from _`),
	}
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

/*
Appends the provided AST and arguments.

Feature: validation. The provided AST must have ordinal parameters matching the
provided arguments. It must not have named parameters.

Feature: renumeration. Ordinal parameters in the provided AST must start at $1.
They're automatically offset by the amount of arguments this query had
previously.

Feature: subquery insertion / flattening. Detects `IQuery` instances among
the arguments, moves them into the AST, and appends their args, renumerating
ordinal parameters to avoid collisions.

Warning: the provided AST may be immediately mutated; it's also stored inside
this query's AST and may be mutated later. If the AST is owned by another query,
it should be copied via `sqlp.CopyDeep()` before calling this.

The provided slice of args is not mutated.
*/
func (self *Query) append(nodes sqlp.Nodes, args []interface{}) error {
	err := validateOrdinalParams(nodes, args)
	if err != nil {
		return err
	}

	args, err = flattenQueries(nodes, args)
	if err != nil {
		return err
	}

	err = renumerateOrdinalParams(nodes, len(self.Args))
	if err != nil {
		return err
	}

	self.Nodes = appendNodesWithSpace(self.Nodes, nodes)
	self.Args = append(self.Args, args...)
	return nil
}

/*
"Flattens" instances of `IQuery` into the AST and into the args. For every
instance of `IQuery`, its args are appended to the base args, and every
occurrence of its ordinal param in the base AST is replaced with its own AST.
All ordinal parameters are appropriately renumerated to avoid collisions.
*/
func flattenQueries(nodes sqlp.Nodes, args []interface{}) ([]interface{}, error) {
	if !argsHaveQueries(args) {
		return args, nil
	}

	var pendingArgs []interface{}

	for _, arg := range args {
		if !isQuery(arg) {
			pendingArgs = append(pendingArgs, arg)
		}
	}

	argOffsets := make([]int, len(args))
	prevQueries := 0

	for i, arg := range args {
		query, ok := arg.(IQuery)

		if ok {
			_, queryArgs := query.Unwrap()
			argOffsets[i] = len(pendingArgs)
			pendingArgs = append(pendingArgs, queryArgs...)
			prevQueries++
			continue
		}

		argOffsets[i] = -prevQueries
	}

	err := sqlp.TraverseDeep(nodes, func(ptr *sqlp.Node) error {
		ord, ok := (*ptr).(sqlp.NodeOrdinalParam)
		if !ok {
			return nil
		}

		index, err := ordToIndex(ord)
		if err != nil {
			return err
		}

		if !(index >= 0 && index < len(args)) {
			return Err{
				Code:  ErrCodeMissingArgument,
				While: `flattening sub-queries`,
				Cause: fmt.Errorf(`missing argument for ordinal parameter $%d`, ord),
			}
		}

		offset := argOffsets[index]

		query, ok := args[index].(IQuery)
		if !ok {
			ord, err := indexToOrd(index + offset)
			if err != nil {
				return err
			}

			*ptr = ord
			return nil
		}

		queryNodes, _ := query.Unwrap()

		/**
		Note: we must duplicate this structure for every occurrence of this ordinal
		parameter. If the AST contains multiple occurrences of it, and if we had
		reused the same sub-AST slice for each occurrence, then future renumerations
		of this AST would subtract from each param contained in the sub-AST multiple
		times, which is incorrect.
		*/
		queryNodes = sqlp.CopyDeep(queryNodes)
		err = renumerateOrdinalParams(queryNodes, offset)
		if err != nil {
			return err
		}

		*ptr = queryNodes
		return nil
	})

	return pendingArgs, err
}

/*
Validates the following:

	* The AST doesn't have any named parameters.
	* Parameters, if any, start at 1.
	* Parameters, if any, don't have any gaps (from 1 to N with step +1).
	* Each parameter has an argument.
	* Each argument has a parameter.
*/
func validateOrdinalParams(nodes sqlp.Nodes, args []interface{}) error {
	/**
	Minor note: this could be 8 times more compact if we used a single bit for
	every index, rather than a byte (sizeof bool). Avoiding an allocation might
	not be practical because the size of `len(args)` might exceed 64 bits, 128
	bits, etc.
	*/
	foundOrds := make([]bool, len(args))

	err := sqlp.TraverseDeep(nodes, func(ptr *sqlp.Node) error {
		unwanted, ok := (*ptr).(sqlp.NodeNamedParam)
		if ok {
			return Err{
				Code:  ErrCodeUnexpectedParameter,
				While: `validating query ordinal parameters`,
				Cause: fmt.Errorf(`unexpected named parameter %q`, unwanted),
			}
		}

		ord, ok := (*ptr).(sqlp.NodeOrdinalParam)
		if !ok {
			return nil
		}

		index, err := ordToIndex(ord)
		if err != nil {
			return err
		}

		if !(index >= 0 && index < len(args)) {
			return Err{
				Code:  ErrCodeMissingArgument,
				While: `validating query ordinal parameters`,
				Cause: fmt.Errorf(`missing argument for ordinal parameter $%d`, ord),
			}
		}
		foundOrds[index] = true

		return nil
	})
	if err != nil {
		return err
	}

	for i, found := range foundOrds {
		if !found {
			return Err{
				Code:  ErrCodeMissingParameter,
				While: `validating query ordinal parameters`,
				Cause: fmt.Errorf(`missing ordinal parameter for argument with index %d, value %v`, i, args[i]),
			}
		}
	}

	return nil
}

/*
Rewrites the provided AST, offsetting each ordinal parameter by the provided
amount.
*/
func renumerateOrdinalParams(nodes sqlp.Nodes, offset int) error {
	return sqlp.TraverseDeep(nodes, func(ptr *sqlp.Node) error {
		ord, ok := (*ptr).(sqlp.NodeOrdinalParam)
		if !ok {
			return nil
		}

		index, err := ordToIndex(ord)
		if err != nil {
			return err
		}

		ord, err = indexToOrd(index + offset)
		if err != nil {
			return err
		}

		*ptr = ord
		return nil
	})
}

/*
Rewrites the provided AST, replacing named parameters with ordinal parameters,
and returns a slice of ordinal args. Also validates the following:

	* The AST doesn't have any previously existing ordinal parameters.
	* Every named parameter has a named argument.
	* Every named argument has a named parameter.
*/
func namedToOrdinal(nodes sqlp.Nodes, namedArgs map[string]interface{}) ([]interface{}, error) {
	namesToOrdinals := make(map[string]sqlp.NodeOrdinalParam, len(namedArgs))
	ordinalArgs := make([]interface{}, 0, len(namedArgs))

	err := sqlp.TraverseDeep(nodes, func(ptr *sqlp.Node) error {
		unwanted, ok := (*ptr).(sqlp.NodeOrdinalParam)
		if ok {
			return Err{
				Code:  ErrCodeUnexpectedParameter,
				While: `converting named parameters to ordinal parameters`,
				Cause: fmt.Errorf(`unexpected ordinal parameter $%d`, unwanted),
			}
		}

		named, ok := (*ptr).(sqlp.NodeNamedParam)
		if !ok {
			return nil
		}

		name := string(named)

		// Get or create an ordinal corresponding to this named.
		ord, ok := namesToOrdinals[name]
		if !ok {
			arg, ok := namedArgs[name]
			if !ok {
				return Err{
					Code:  ErrCodeMissingParameter,
					While: `converting named parameters to ordinal parameters`,
					Cause: fmt.Errorf(`missing named parameter %q`, name),
				}
			}

			ord = sqlp.NodeOrdinalParam(len(ordinalArgs) + 1)
			namesToOrdinals[name] = ord
			ordinalArgs = append(ordinalArgs, arg)
		}

		*ptr = ord
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Ensure that every named argument is used.
	for name := range namedArgs {
		_, ok := namesToOrdinals[name]
		if !ok {
			return nil, Err{
				Code:  ErrCodeUnusedArgument,
				While: `converting named parameters to ordinal parameters`,
				Cause: fmt.Errorf(`unused named argument %q`, name),
			}
		}
	}

	return ordinalArgs, nil
}
