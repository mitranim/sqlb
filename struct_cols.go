package sqlb

import (
	"fmt"
	"reflect"

	"github.com/mitranim/refut"
)

/*
Takes a struct and generates a string of column names suitable for inclusion
into `select`. Also accepts the following inputs and automatically dereferences
them into a struct type:

	* Struct pointer.
	* Struct slice.
	* Struct slice pointer.

Nil slices and pointers are fine, as long as they carry a struct type. Any other
input causes a panic.

Should be used in conjunction with `Query`. Also see `Query.WrapSelectCols()`.
*/
func Cols(dest interface{}) string {
	rtype := refut.RtypeDeref(reflect.TypeOf(dest))
	if rtype.Kind() == reflect.Slice {
		rtype = refut.RtypeDeref(rtype.Elem())
	}

	if rtype.Kind() != reflect.Struct {
		panic(Err{
			Code:  ErrCodeInvalidInput,
			While: `generating struct columns for select clause`,
			Cause: fmt.Errorf("expected struct, got %q", rtype),
		})
	}

	idents := structRtypeSqlIdents(rtype)
	return sqlIdent{idents: idents}.selectString()
}

func structRtypeSqlIdents(rtype reflect.Type) []sqlIdent {
	var idents []sqlIdent

	err := refut.TraverseStructRtype(rtype, func(sfield reflect.StructField, _ []int) error {
		colName := sfieldColumnName(sfield)
		if colName == "" {
			return nil
		}

		fieldRtype := refut.RtypeDeref(sfield.Type)
		if fieldRtype.Kind() == reflect.Struct && !isScannableRtype(fieldRtype) {
			idents = append(idents, sqlIdent{
				name:   colName,
				idents: structRtypeSqlIdents(fieldRtype),
			})
			return nil
		}

		idents = append(idents, sqlIdent{name: colName})
		return nil
	})
	if err != nil {
		panic(err)
	}

	return idents
}

type sqlIdent struct {
	name   string
	idents []sqlIdent
}

func (self sqlIdent) selectString() string {
	return bytesToMutableString(self.appendSelect(nil, nil))
}

func (self sqlIdent) appendSelect(buf []byte, path []sqlIdent) []byte {
	/**
	If the ident doesn't have a name, it's just a collection of other idents,
	which are considered to be at the "top level". If the ident has a name, it's
	considered to "contain" the other idents.
	*/
	if len(self.idents) > 0 {
		if self.name != "" {
			path = append(path, self)
		}
		for _, ident := range self.idents {
			buf = ident.appendSelect(buf, path)
		}
		return buf
	}

	if self.name == "" {
		return buf
	}

	if len(buf) > 0 {
		buf = append(buf, `, `...)
	}

	if len(path) == 0 {
		buf = self.appendAlias(buf, nil)
	} else {
		buf = self.appendPath(buf, path)
		buf = append(buf, ` as `...)
		buf = self.appendAlias(buf, path)
	}

	return buf
}

func (self sqlIdent) appendPath(buf []byte, path []sqlIdent) []byte {
	for i, ident := range path {
		if i == 0 {
			buf = appendDelimited(buf, `("`, ident.name, `")`)
		} else {
			buf = appendDelimited(buf, `"`, ident.name, `"`)
		}
		buf = append(buf, `.`...)
	}
	buf = appendDelimited(buf, `"`, self.name, `"`)
	return buf
}

func (self sqlIdent) appendAlias(buf []byte, path []sqlIdent) []byte {
	buf = append(buf, `"`...)
	for _, ident := range path {
		buf = append(buf, ident.name...)
		buf = append(buf, `.`...)
	}
	buf = append(buf, self.name...)
	buf = append(buf, `"`...)
	return buf
}
