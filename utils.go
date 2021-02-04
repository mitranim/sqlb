package sqlb

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
	"unsafe"

	"github.com/mitranim/refut"
)

const bitsetSize = int(unsafe.Sizeof(bitset(0)) * 8)

type bitset uint64

func (self bitset) has(index int) bool { return self&(1<<index) != 0 }
func (self *bitset) set(index int)     { *self |= (1 << index) }
func (self *bitset) unset(index int)   { *self ^= (1 << index) }

func appendStr(buf *[]byte, str string) {
	*buf = append(*buf, str...)
}

func appendEnclosed(buf *[]byte, prefix, infix, suffix string) {
	*buf = append(*buf, prefix...)
	*buf = append(*buf, infix...)
	*buf = append(*buf, suffix...)
}

/*
Allocation-free conversion. Reinterprets a byte slice as a string. Borrowed from
the standard library. Reasonably safe. Should not be used when the underlying
byte array is volatile, for example when it's part of a scratch buffer during
SQL scanning.
*/
func bytesToMutableString(bytes []byte) string {
	return *(*string)(unsafe.Pointer(&bytes))
}

/*
TODO: consider validating that the column name doesn't contain double quotes. We
might return an error, or panic.
*/
func sfieldColumnName(sfield reflect.StructField) string {
	return refut.TagIdent(sfield.Tag.Get("db"))
}

func isWhitespaceChar(char rune) bool {
	switch char {
	case ' ', '\n', '\r', '\t', '\v':
		return true
	default:
		return false
	}
}

func isQuery(val interface{}) bool {
	_, ok := val.(IQuery)
	return ok
}

var timeRtype = reflect.TypeOf(time.Time{})
var sqlScannerRtype = reflect.TypeOf((*sql.Scanner)(nil)).Elem()

func isScannableRtype(rtype reflect.Type) bool {
	return rtype != nil &&
		(rtype == timeRtype || reflect.PtrTo(rtype).Implements(sqlScannerRtype))
}

func traverseStructDbFields(input interface{}, fun func(string, interface{})) {
	rval := reflect.ValueOf(input)
	rtype := refut.RtypeDeref(rval.Type())

	if rtype.Kind() != reflect.Struct {
		panic(Err{
			Code:  ErrCodeInvalidInput,
			While: `traversing struct for DB fields`,
			Cause: fmt.Errorf(`expected struct, got %q`, rtype),
		})
	}

	if refut.IsRvalNil(rval) {
		return
	}

	err := refut.TraverseStructRval(rval, func(rval reflect.Value, sfield reflect.StructField, _ []int) error {
		colName := sfieldColumnName(sfield)
		if colName == "" {
			return nil
		}
		fun(colName, rval.Interface())
		return nil
	})
	if err != nil {
		panic(err)
	}
}

func appendNonQueries(out *[]interface{}, more []interface{}) {
	for _, val := range more {
		if !isQuery(val) {
			*out = append(*out, val)
		}
	}
}

func queryArgsBefore(args []interface{}, index int) int {
	var count int
	for i, arg := range args {
		if i >= index {
			break
		}
		if isQuery(arg) {
			count++
		}
	}
	return count
}

func appendSpaceIfNeeded(buf *[]byte) {
	if buf != nil && len(*buf) > 0 && !endsWithWhitespace(*buf) {
		*buf = append(*buf, ` `...)
	}
}

func endsWithWhitespace(chunk []byte) bool {
	char, _ := utf8.DecodeLastRune(chunk)
	return isWhitespaceChar(char)
}

const dottedPath = `(?:\w+\.)*\w+`

var dottedPathReg = regexp.MustCompile(`^` + dottedPath + `$`)
var ordReg = regexp.MustCompile(`^(` + dottedPath + `)\s+(?i)(asc|desc)$`)

/*
Takes a struct type and a dot-separated path of JSON field names
like "one.two.three". Finds the nested struct field corresponding to that path,
returning an error if a field could not be found.

Note that this can't use `reflect.Value.FieldByName` because it searches by JSON
field name, not by Go field name.

Copied from `github.com/mitranim/jel`. Should consolidate.
*/
func structFieldByJsonPath(rtype reflect.Type, pathStr string) (sfield reflect.StructField, path []string, err error) {
	if !dottedPathReg.MatchString(pathStr) {
		err = fmt.Errorf(`[sqlb] expected a valid dot-separated identifier, got %q`, pathStr)
		return
	}

	if rtype == nil {
		err = fmt.Errorf(`[sqlb] can't find field by path %q: no type provided`, pathStr)
		return
	}

	path = strings.Split(pathStr, ".")

	for i, segment := range path {
		err = sfieldByJsonName(rtype, segment, &sfield)
		if err != nil {
			return
		}

		colName := sfieldColumnName(sfield)
		if colName == "" {
			err = fmt.Errorf(`[sqlb] no column name corresponding to %q in type %v for path %q`,
				segment, rtype, pathStr)
			return
		}

		path[i] = colName
		rtype = sfield.Type
	}
	return
}

var errBreak = errors.New("")

/*
Finds the struct field that has the given JSON field name. The field may be in
an embedded struct, but not in any non-embedded nested structs.

Copied from `github.com/mitranim/jel`. Should consolidate.
*/
func sfieldByJsonName(rtype reflect.Type, name string, out *reflect.StructField) error {
	if rtype == nil {
		return fmt.Errorf(`[sqlb] can't find field %q: no type provided`, name)
	}

	err := refut.TraverseStructRtype(rtype, func(sfield reflect.StructField, _ []int) error {
		if sfieldJsonFieldName(sfield) == name {
			*out = sfield
			return errBreak
		}
		return nil
	})
	if errors.Is(err, errBreak) {
		return nil
	}
	if err != nil {
		return err
	}

	return fmt.Errorf(`[sqlb] no struct field corresponding to JSON field name %q in type %v`, name, rtype)
}

func sfieldJsonFieldName(sfield reflect.StructField) string {
	return refut.TagIdent(sfield.Tag.Get("json"))
}

// Copied from `github.com/mitranim/jel`. Should consolidate.
func appendSqlPath(buf *[]byte, path []string) {
	for i, str := range path {
		// Just a sanity check. We probably shouldn't allow to decode such
		// identifiers in the first place.
		if strings.Contains(str, `"`) {
			panic(fmt.Errorf(`[sqlb] unexpected %q in SQL identifier %q`, `"`, str))
		}

		if i == 0 {
			if len(path) > 1 {
				appendEnclosed(buf, `("`, str, `")`)
			} else {
				appendEnclosed(buf, `"`, str, `"`)
			}
		} else {
			appendStr(buf, `.`)
			appendEnclosed(buf, `"`, str, `"`)
		}
	}
}
