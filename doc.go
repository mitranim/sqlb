/*
SQL Builder: simple SQL query builder. Oriented towards text and writing PLAIN
SQL, simplifying parameters, arguments, query interpolation, query composition,
and so on. Also provides tools for converting structs into SQL expressions and
arguments.

See the sibling library https://github.com/mitranim/gos for scanning SQL rows
into structs.

Key Features

• You write plain SQL. There's no DSL in Go.

• Automatically renumerates ordinal parameters such as $1, $2, and so on. In the
code, the count always starts at 1.

• Supports named parameters such as :ident, automatically converting them into
ordinals.

• Avoids parameter collisions.

• Composable: query objects used as arguments are automatically inserted,
combining the arguments and automatically renumerating the parameters.

• Supports converting structs to SQL clauses such as `select A, B, C`,
`names (...) values (...)`, etc.

• Supports converting structs to named argument maps.

Examples

See `Query()`, `Query.Append()`, `Query.AppendNamed()` for examples.
*/
package sqlb
