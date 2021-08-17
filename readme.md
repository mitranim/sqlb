## Overview

**SQL** **B**uilder: simple SQL query builder. Oriented towards text and **writing plain SQL**, simplifying parameters, arguments, query interpolation, query composition, and so on. Also provides tools for converting structs into SQL expressions and arguments.

See the full documentation at https://pkg.go.dev/github.com/mitranim/sqlb.

See the sibling library https://github.com/mitranim/gos for scanning SQL rows into structs.

## Changelog

### `v0.1.16`

Added `NamedArg.Norm`. Improved `NamedArg.IsNil` and `NamedArgs.Conditions`. They now use the `driver.Valuer.Value` method, if present, to determine null-ness, which works for non-pointer "nullable" types.

### `v0.1.15`

`Ords.Or` is now a value method that returns a modified version, rather than a pointer method that mutated the original.

### `v0.1.14`

`StructMap` and `StructNamedArgs` now tolerate `nil` inputs. Previously, they tolerated non-nil interfaces where the underlying value was a nil struct pointer. Now they also allow nil interfaces.

### `v0.1.13`

Fixed the bug where the `Ords.Lax` mode was appending malformed ords, instead of skipping them entirely.

### `v0.1.12`

* `StrQuery` now interpolates directly, without invoking `(*Query).Append` on the provided query. This allows to interpolate `StrQuery` strings that contain parameter placeholders. Use at your own risk.

* `(*Query).Append` no longer has an argument length limit.

### `v0.1.11`

Added `Ords.Lax`: a boolean that causes `Ords` to skip unknown fields during parsing.

### `v0.1.10`

Breaking changes in the name of efficiency:

* `NamedArgs.Conditions` now uses `=` and `is null`, as appropriate, instead of previous `is not distinct from`. At the time of writing, Postgres (version <= 12) is unable to use indexes for `is not distinct from`, which may result in much slower queries. The new approach avoids this gotcha.

* In `Ord`, `nulls last` is now opt-in rather than default. In addition, `asc/desc` in input strings is now optional. This more precisely reflects SQL semantics and allows finer-grained control. More importantly, it avoids a potential performance gotcha. At the time of writing, Postgres (version <= 12) is unable to use normal indexes for `nulls last` ordering. Instead it requires specialized indexes where `nulls last` is set explicitly. Making it opt-in reduces the chance of accidental slowness.

  * Added `OrdAscNl` and `OrdDescNl` for convenient construction.

  * Minor breaking change: `Ord.IsDesc` is now `Ord.Desc`.

* Minor breaking change: removed `Ord.IsValid`.

Non-breaking additions:

* `Ords.RowNumber()`: generates a Postgres window function expression `row_number() over (order by ...)`, falling back on a constant value when the ordering is empty.

* `QueryOrd()`: shortcut for making a `Query` with a single `.Append()` invocation.

* `QueryNamed()`: shortcut for making a `Query` with a single `.AppendNamed()` invocation.

### 0.1.9

Added `Ords` and `Ord`: structured representation of `order by`, able to decode from external input such as JSON, but also flexible enough to store arbitrary sub-queries. Ported from `github.com/mitranim/jel`, while also adding the ability to store sub-queries rather than only identifiers.

### 0.1.8

Added `StrQuery`.

### 0.1.7

Corrected `CheckUnused` to be `true` by default, which was always intended.

### 0.1.6

Added `CheckUnused` which allows to opt out of unused parameter checks in `Query.Append` and `Query.AppendNamed`. Can be convenient for development.

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
