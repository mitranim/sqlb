package sqlb

import (
	"database/sql"
	"fmt"
	"reflect"
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
