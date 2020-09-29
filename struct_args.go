package sqlb

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"

	"github.com/mitranim/refut"
)

/*
Scans a struct, accumulating fields tagged with `db` into a map suitable for
`Query.AppendNamed()`. The input must be a struct or a struct pointer. A nil
pointer is fine and produces a nil result. Panics on other inputs. Treats an
embedded struct as part of the enclosing struct.
*/
func StructMap(input interface{}) map[string]interface{} {
	dict := map[string]interface{}{}
	traverseStructDbFields(input, func(name string, value interface{}) {
		dict[name] = value
	})
	return dict
}

/*
Scans a struct, converting fields tagged with `db` into a sequence of named
`NamedArgs`. The input must be a struct or a struct pointer. A nil pointer is
fine and produces a nil result. Panics on other inputs. Treats embedded structs
part of enclosing structs.
*/
func StructNamedArgs(input interface{}) NamedArgs {
	var args NamedArgs
	traverseStructDbFields(input, func(name string, value interface{}) {
		args = append(args, Named(name, value))
	})
	return args
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

/*
Sequence of named SQL arguments with utility methods for query building.
Usually obtained by calling `StructNamedArgs()`.
*/
type NamedArgs []NamedArg

/*
Returns the argument names.
*/
func (self NamedArgs) Names() []string {
	var names []string
	for _, arg := range self {
		names = append(names, arg.Name)
	}
	return names
}

/*
Returns the argument values.
*/
func (self NamedArgs) Values() []interface{} {
	var values []interface{}
	for _, arg := range self {
		values = append(values, arg.Value)
	}
	return values
}

/*
Returns comma-separated argument names, suitable for a `select` clause. Example:

	val := struct {
		One int64 `db:"one"`
		Two int64 `db:"two"`
	}{
		One: 10,
		Two: 20,
	}

	args := StructNamedArgs(val)

	fmt.Sprintf(`select %v`, args.NamesString())

	// Output:
	`select "one", "two"`
*/
func (self NamedArgs) NamesString() string {
	var buf []byte
	for i, arg := range self {
		if i > 0 {
			buf = append(buf, `, `...)
		}
		buf = appendDelimited(buf, `"`, arg.Name, `"`)
	}
	return bytesToMutableString(buf)
}

/*
Returns parameter placeholders in the Postgres style `$N`, comma-separated,
suitable for a `values` clause. Example:

	val := struct {
		One int64 `db:"one"`
		Two int64 `db:"two"`
	}{
		One: 10,
		Two: 20,
	}

	args := StructNamedArgs(val)

	fmt.Sprintf(`values (%v)`, args.ValuesString())

	// Output:
	`values ($1, $2)`
*/

func (self NamedArgs) ValuesString() string {
	var buf []byte
	for i := range self {
		if i > 0 {
			buf = append(buf, `, `...)
		}
		buf = append(buf, `$`...)
		buf = strconv.AppendInt(buf, int64(i+1), 10)
	}
	return bytesToMutableString(buf)
}

/*
Returns the string of names and values suitable for an `insert` clause. Example:

	val := struct {
		One int64 `db:"one"`
		Two int64 `db:"two"`
	}{
		One: 10,
		Two: 20,
	}

	args := StructNamedArgs(val)

	fmt.Sprintf(`insert into some_table %v`, args.NamesAndValuesString())

	// Output:
	`insert into some_table ("one", "two") values ($1, $2)`
*/
func (self NamedArgs) NamesAndValuesString() string {
	if len(self) == 0 {
		return "default values"
	}
	return fmt.Sprintf("(%v) values (%v)", self.NamesString(), self.ValuesString())
}

/*
Returns the string of assignments suitable for an `update set` clause. Example:

	val := struct {
		One int64 `db:"one"`
		Two int64 `db:"two"`
	}{
		One: 10,
		Two: 20,
	}

	args := StructNamedArgs(val)

	fmt.Sprintf(`update some_table set %v`, args.AssignmentsString())

	// Output:
	`update some_table set "one" = $1, "two" = $2`
*/
func (self NamedArgs) AssignmentsString() string {
	var buf []byte
	for i, arg := range self {
		if i > 0 {
			buf = append(buf, `, `...)
		}
		buf = appendDelimited(buf, `"`, arg.Name, `"`)
		buf = append(buf, ` = $`...)
		buf = strconv.AppendInt(buf, int64(i+1), 10)
	}
	return bytesToMutableString(buf)
}

/*
Returns the string of conditions suitable for a `where` or `join` clause.
Example:

	val := struct {
		One int64 `db:"one"`
		Two int64 `db:"two"`
	}{
		One: 10,
		Two: 20,
	}

	args := StructNamedArgs(val)

	fmt.Sprintf(`select * from some_table where %v`, args.ConditionsString())

	// Output (formatted for readability):
	`
	select * from some_table
	where
		"one" is not distinct from $1 and
		"two" is not distinct from $2
	`
*/
func (self NamedArgs) ConditionsString() string {
	if len(self) == 0 {
		return "true"
	}

	var buf []byte

	for i, arg := range self {
		if i > 0 {
			buf = append(buf, ` and `...)
		}
		buf = appendDelimited(buf, `"`, arg.Name, `"`)
		buf = append(buf, ` is not distinct from $`...)
		buf = strconv.AppendInt(buf, int64(i+1), 10)
	}

	return bytesToMutableString(buf)
}

/*
Returns true if at least one argument satisfies the predicate function. Example:

  args.Some(NamedArg.IsNil)
*/
func (self NamedArgs) Some(fun func(NamedArg) bool) bool {
	for _, arg := range self {
		if fun != nil && fun(arg) {
			return true
		}
	}
	return false
}

/*
Returns true if every argument satisfies the predicate function. Example:

  args.Every(NamedArg.IsNil)
*/
func (self NamedArgs) Every(fun func(NamedArg) bool) bool {
	for _, arg := range self {
		if fun == nil || !fun(arg) {
			return false
		}
	}
	return true
}

// Same as `sql.NamedArg`, with additional methods. See `NamedArgs`.
type NamedArg struct {
	Name  string
	Value interface{}
}

func Named(name string, value interface{}) NamedArg {
	return NamedArg{Name: name, Value: value}
}

func (self NamedArg) IsValid() bool {
	return columnNameRegexp.MatchString(self.Name)
}

var columnNameRegexp = regexp.MustCompile(`^(?:\w+(?:\.\w+)*|"\w+(?:\.\w+)*")$`)

func (self NamedArg) IsNil() bool {
	return refut.IsNil(self.Value)
}
