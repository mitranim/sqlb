package sqlb

import r "reflect"

/*
Variant of `[]any` conforming to the `ArgDict` interface. Supports only
ordinal parameters, not named parameters. Used for `StrQ`. See the `ListQ`
shortcut.
*/
type List []any

// Implement part of the `ArgDict` interface.
func (self List) IsEmpty() bool { return self.Len() == 0 }

// Implement part of the `ArgDict` interface.
func (self List) Len() int { return len(self) }

// Implement part of the `ArgDict` interface.
func (self List) GotOrdinal(key int) (any, bool) {
	if key >= 0 && key < len(self) {
		return self[key], true
	}
	return nil, false
}

// Implement part of the `ArgDict` interface. Always returns `nil, false`.
func (self List) GotNamed(string) (any, bool) { return nil, false }

// Implement `OrdinalRanger` to automatically validate used/unused arguments.
func (self List) RangeOrdinal(fun func(int)) {
	if fun != nil {
		for ind := range counter(len(self)) {
			fun(ind)
		}
	}
}

/*
Variant of `map[string]any` conforming to the `ArgDict` interface.
Supports only named parameters, not ordinal parameters. Used for `StrQ`. See
the `DictQ` shortcut.
*/
type Dict map[string]any

// Implement part of the `ArgDict` interface.
func (self Dict) IsEmpty() bool { return self.Len() == 0 }

// Implement part of the `ArgDict` interface.
func (self Dict) Len() int { return len(self) }

// Implement part of the `ArgDict` interface. Always returns `nil, false`.
func (self Dict) GotOrdinal(int) (any, bool) { return nil, false }

// Implement part of the `ArgDict` interface.
func (self Dict) GotNamed(key string) (any, bool) {
	val, ok := self[key]
	return val, ok
}

// Implement `NamedRanger` to automatically validate used/unused arguments.
func (self Dict) RangeNamed(fun func(string)) {
	if fun != nil {
		for key := range self {
			fun(key)
		}
	}
}

/*
Variant of `Dict` without support for validating unused arguments. Note that
missing arguments are still detected and cause errors. Useful when generating
the dictionary dynamically, rather than hardcoding the set of keys. Must be
used with `StrQ` or `Prep`, rather than with `DictQ`, because the latter always
converts the given dictionary to `Dict`.
*/
type LaxDict Dict

// Implement part of the `ArgDict` interface.
func (self LaxDict) IsEmpty() bool { return Dict(self).IsEmpty() }

// Implement part of the `ArgDict` interface.
func (self LaxDict) Len() int { return Dict(self).Len() }

// Implement part of the `ArgDict` interface. Always returns `nil, false`.
func (self LaxDict) GotOrdinal(int) (any, bool) { return nil, false }

// Implement part of the `ArgDict` interface.
func (self LaxDict) GotNamed(key string) (any, bool) { return Dict(self).GotNamed(key) }

/*
Implements `ArgDict` by reading struct fields and methods by name. Supports only
named parameters, not ordinal parameters. The inner value must be either
invalid or a struct. Compared to `Dict`, a struct is way faster to construct,
but reading fields by name is way slower. Used for `StrQ`. See the `StructQ`
shortcut.
*/
type StructDict [1]r.Value

// Implement part of the `ArgDict` interface.
func (self StructDict) IsEmpty() bool {
	return !self[0].IsValid() || isStructTypeEmpty(self[0].Type())
}

// Implement part of the `ArgDict` interface. Always returns 0.
func (self StructDict) Len() int { return 0 }

// Implement part of the `ArgDict` interface. Always returns `nil, false`.
func (self StructDict) GotOrdinal(int) (any, bool) { return nil, false }

// Implement part of the `ArgDict` interface.
func (self StructDict) GotNamed(key string) (any, bool) {
	/**
	(Tested in Go 1.17.)

	In our benchmarks, making a struct dict is about 15 times faster than a normal
	dict (100ns vs 1500ns for 12 fields and 12 methods), but accessing various
	fields and methods is about 25 times slower on average (5000ns vs 200ns for
	12 fields and 12 methods). When using only fields without methods, the
	access time numbers are much closer (700ns vs 100ns for 12 fields). The
	total numbers are close enough, and small enough, to justify both, depending
	on the use case.

	Compared to using `reflect.Value.FieldByName` and `reflect.Value.MethodByName`
	every time, using a cached dict with field and method indexes improves
	average access performance by about 3 times in our benchmarks.
	*/

	val := valueDeref(self[0])
	if !val.IsValid() {
		return nil, false
	}

	path, ok := loadStructPathMap(val.Type())[key]
	if !ok {
		return nil, false
	}

	if path.FieldIndex != nil {
		return val.FieldByIndex(path.FieldIndex).Interface(), true
	}

	meth := val.Method(path.MethodIndex)
	if meth.IsValid() {
		reqGetter(val.Type(), meth.Type(), key)
		return meth.Call(nil)[0].Interface(), true
	}

	return nil, false
}
