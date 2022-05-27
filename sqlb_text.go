package sqlb

import (
	"encoding"
	"fmt"
	r "reflect"
	"strconv"
)

/*
Tiny shortcut for encoding an `Appender` implementation to a string by using its
`.Append` method, without paying for a string-to-byte conversion. Used
internally by many `Expr` implementations. Exported because it's handy for
defining new types.
*/
func AppenderString(val Appender) string {
	if val != nil {
		return bytesToMutableString(val.Append(nil))
	}
	return ``
}

// Variant of `String` that panics on error.
func TryString(val any) string { return try1(String(val)) }

/*
Missing feature of the standard library: return a string representation of an
arbitrary value intended only for machine use, only for "intentionally"
encodable types, without swallowing errors. Differences from `fmt.Sprint`:

	* Nil input = "" output.

	* Returns errors separately, without encoding them into the output. This is
	  important when the output is intended to be passed to another system rather
	  than read by humans.

	* Supports ONLY the following types, in this order of priority. For other
	  types, returns an error.

		* `fmt.Stringer`
		* `Appender`
		* `encoding.TextMarshaler`
		* Built-in primitive types.
			* Encodes floats without the scientific notation.
		* Aliases of `[]byte`.
*/
func String(src any) (string, error) {
	if src == nil {
		return ``, nil
	}

	stringer, _ := src.(fmt.Stringer)
	if stringer != nil {
		return stringer.String(), nil
	}

	appender, _ := src.(Appender)
	if appender != nil {
		return bytesToMutableString(appender.Append(nil)), nil
	}

	marshaler, _ := src.(encoding.TextMarshaler)
	if marshaler != nil {
		chunk, err := marshaler.MarshalText()
		str := bytesToMutableString(chunk)
		if err != nil {
			return ``, ErrInternal{Err{`generating string representation`, err}}
		}
		return str, nil
	}

	typ := typeOf(src)
	val := valueOf(src)

	switch typ.Kind() {
	case r.Int8, r.Int16, r.Int32, r.Int64, r.Int:
		if val.IsValid() {
			return strconv.FormatInt(val.Int(), 10), nil
		}
		return ``, nil

	case r.Uint8, r.Uint16, r.Uint32, r.Uint64, r.Uint:
		if val.IsValid() {
			return strconv.FormatUint(val.Uint(), 10), nil
		}
		return ``, nil

	case r.Float32, r.Float64:
		if val.IsValid() {
			return strconv.FormatFloat(val.Float(), 'f', -1, 64), nil
		}
		return ``, nil

	case r.Bool:
		if val.IsValid() {
			return strconv.FormatBool(val.Bool()), nil
		}
		return ``, nil

	case r.String:
		if val.IsValid() {
			return val.String(), nil
		}
		return ``, nil

	default:
		if typ.ConvertibleTo(typeBytes) {
			if val.IsValid() {
				return bytesToMutableString(val.Bytes()), nil
			}
			return ``, nil
		}

		return ``, errUnsupportedType(`generating string representation`, typ)
	}
}

// Variant of `Append` that panics on error.
func TryAppend(buf []byte, src any) []byte { return try1(Append(buf, src)) }

/*
Missing feature of the standard library: append the text representation of an
arbitrary value to the buffer, prioritizing "append"-style encoding functions
over "string"-style functions for efficiency, using only "intentional"
representations, and without swallowing errors.

Supports ONLY the following types, in this order of priority. For other types,
returns an error.

	* `Appender`
	* `encoding.TextMarshaler`
	* `fmt.Stringer`
	* Built-in primitive types.
	* Aliases of `[]byte`.

Special cases:

	* Nil: append nothing, return buffer as-is.
	* Integers: use `strconv.AppendInt` in base 10.
	* Floats: use `strconv.AppendFloat` without scientific notation.

Used internally by `CommaAppender`, exported for advanced users.
*/
func Append(buf []byte, src any) ([]byte, error) {
	if src == nil {
		return buf, nil
	}

	appender, _ := src.(Appender)
	if appender != nil {
		return appender.Append(buf), nil
	}

	marshaler, _ := src.(encoding.TextMarshaler)
	if marshaler != nil {
		chunk, err := marshaler.MarshalText()
		if err != nil {
			return buf, ErrInternal{Err{`appending string representation`, err}}
		}
		return append(buf, chunk...), nil
	}

	stringer, _ := src.(fmt.Stringer)
	if stringer != nil {
		return append(buf, stringer.String()...), nil
	}

	typ := typeOf(src)
	val := valueOf(src)

	switch typ.Kind() {
	case r.Int8, r.Int16, r.Int32, r.Int64, r.Int:
		if val.IsValid() {
			return strconv.AppendInt(buf, val.Int(), 10), nil
		}
		return buf, nil

	case r.Uint8, r.Uint16, r.Uint32, r.Uint64, r.Uint:
		if val.IsValid() {
			return strconv.AppendUint(buf, val.Uint(), 10), nil
		}
		return buf, nil

	case r.Float32, r.Float64:
		if val.IsValid() {
			return strconv.AppendFloat(buf, val.Float(), 'f', -1, 64), nil
		}
		return buf, nil

	case r.Bool:
		if val.IsValid() {
			return strconv.AppendBool(buf, val.Bool()), nil
		}
		return buf, nil

	case r.String:
		if val.IsValid() {
			return append(buf, val.String()...), nil
		}
		return buf, nil

	default:
		if typ.ConvertibleTo(typeBytes) {
			if val.IsValid() {
				return append(buf, val.Bytes()...), nil
			}
			return buf, nil
		}

		return buf, errUnsupportedType(`appending string representation`, typ)
	}
}

// Variant of `AppendWith` that panics on error.
func TryAppendWith(buf *[]byte, delim string, val any) bool {
	return try1(AppendWith(buf, delim, val))
}

/*
Attempts to append the given delimiter and the text representation of the given
value, via `Append`. If after delimiter non-zero amount of bytes was appended,
returns true. Otherwise reverts the buffer to the original length and returns
false. If the buffer got reallocated with increased capacity, preserves the new
capacity.
*/
func AppendWith(buf *[]byte, delim string, val any) (bool, error) {
	if buf == nil {
		return false, nil
	}

	pre := len(*buf)
	*buf = append(*buf, delim...)

	mid := len(*buf)
	out, err := Append(*buf, val)
	if err != nil {
		return false, err
	}
	*buf = out

	/**
	Note: there's a difference between snapshotting the length to reslice the
	buffer after the append, versus naively snapshotting the slice itself. The
	append may allocate additional capacity, perform a copy, and return a larger
	buffer. Reslicing preserves the new capacity, which is important for
	avoiding a "hot split" where the added capacity is repeatedly discarded.
	*/
	if mid == len(*buf) {
		*buf = (*buf)[:pre]
		return false, nil
	}
	return true, nil
}
