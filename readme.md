## Overview

**SQL** **B**uilder: simple SQL query builder. Oriented towards text and **writing plain SQL**, simplifying parameters, arguments, query interpolation, query composition, and so on. Also provides tools for converting structs into SQL expressions and arguments.

See the full documentation at https://godoc.org/github.com/mitranim/sqlb.

See the sibling library https://github.com/mitranim/gos for scanning SQL rows into structs.

## Changelog

### 0.1.5

Minor bugfix: `Query.String` is now implemented on the non-pointer type, as intended. Also updated the `sqlp` dependency.

### 0.1.4

Breaking changes in `Query`: simpler interface, better performance.

Instead of storing and operating on a parsed AST, `Query` now stores the query text as `[]byte`. We use `sqlp.Tokenizer` to parse inputs without generating an AST, transcoding parameters on the fly. `IQuery` now simply appends to an externally-passed `Query`, instead of having to return a parsed AST representation. All together, this significantly simplifies the implementation of `Query` and any external `IQuery` types.

### 0.1.3

Added `Query.Clear()`.

### 0.1.2

Breaking: methods of `NamedArgs` now return queries, suitable for inclusion into other queries. Separate methods for strings and arg slices have been removed.

### 0.1.1

Dependency update.

### 0.1.0

First tagged release.

## License

https://unlicense.org

## Misc

I'm receptive to suggestions. If this library _almost_ satisfies you but needs changes, open an issue or chat me up. Contacts: https://mitranim.com/#contacts
