package sqlb

import (
	"database/sql"
	"fmt"
	"reflect"
	"time"
	"unicode/utf8"
	"unsafe"

	"github.com/mitranim/refut"
	"github.com/mitranim/sqlp"
)

func copyIfaceSlice(src []interface{}) []interface{} {
	if src == nil {
		return nil
	}
	out := make([]interface{}, len(src), cap(src))
	copy(out, src)
	return out
}

func appendDelimited(buf []byte, prefix, infix, suffix string) []byte {
	buf = append(buf, prefix...)
	buf = append(buf, infix...)
	buf = append(buf, suffix...)
	return buf
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

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func isWhitespaceChar(char rune) bool {
	switch char {
	case ' ', '\n', '\r', '\t', '\v':
		return true
	default:
		return false
	}
}

func ordToIndex(ord sqlp.NodeOrdinalParam) (int, error) {
	if ord > 0 {
		return int(ord - 1), nil
	}
	return 0, Err{
		Code:  ErrCodeIndexMismatch,
		While: `converting ordinal parameter to argument index`,
		Cause: fmt.Errorf(`can't convert ordinal %d to index: must be >= 1`, ord),
	}
}

func indexToOrd(index int) (sqlp.NodeOrdinalParam, error) {
	if index >= 0 {
		return sqlp.NodeOrdinalParam(index + 1), nil
	}
	return 0, Err{
		Code:  ErrCodeIndexMismatch,
		While: `converting argument index to ordinal parameter`,
		Cause: fmt.Errorf(`can't convert index %d to ordinal: must be >= 0`, index),
	}
}

/*
Similar to `append(left, right...)`, but ensures at least one whitespace
character between them.
*/
func appendNodesWithSpace(left sqlp.Nodes, right sqlp.Nodes) sqlp.Nodes {
	if len(left) > 0 && len(right) > 0 && !nodesEndWithWhitespace(left) && !nodesStartWithWhitespace(right) {
		left = append(left, sqlp.NodeText(` `))
	}
	return append(left, right...)
}

func nodesStartWithWhitespace(nodes sqlp.Nodes) bool {
	text, _ := sqlp.FirstLeaf(nodes).(sqlp.NodeText)
	char, size := utf8.DecodeRuneInString(string(text))
	return size > 0 && isWhitespaceChar(char)
}

func nodesEndWithWhitespace(nodes sqlp.Nodes) bool {
	text, _ := sqlp.LastLeaf(nodes).(sqlp.NodeText)
	char, size := utf8.DecodeLastRuneInString(string(text))
	return size > 0 && isWhitespaceChar(char)
}

func argsHaveQueries(args []interface{}) bool {
	for _, arg := range args {
		if isQuery(arg) {
			return true
		}
	}
	return false
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

func queryFrom(str string, args []interface{}) Query {
	var query Query
	query.Append(str, args...)
	return query
}
