package sqlb

import (
	"bytes"
	"encoding/json"
	"fmt"
	r "reflect"
)

/*
Known SQL operations used in JEL. Serves as a whitelist, allowing us to
differentiate them from casts, and describes how to transform JEL Lisp-style
calls into SQL expressions (prefix, infix, etc.). This is case-sensitive and
whitespace-sensitive.
*/
var Ops = map[string]Op{
	`and`:                  OpInfix,
	`or`:                   OpInfix,
	`not`:                  OpPrefix,
	`is null`:              OpPostfix,
	`is not null`:          OpPostfix,
	`is true`:              OpPostfix,
	`is not true`:          OpPostfix,
	`is false`:             OpPostfix,
	`is not false`:         OpPostfix,
	`is unknown`:           OpPostfix,
	`is not unknown`:       OpPostfix,
	`is distinct from`:     OpInfix,
	`is not distinct from`: OpInfix,
	`=`:                    OpInfix,
	`~`:                    OpInfix,
	`~*`:                   OpInfix,
	`~=`:                   OpInfix,
	`<>`:                   OpInfix,
	`<`:                    OpInfix,
	`>`:                    OpInfix,
	`>=`:                   OpInfix,
	`<=`:                   OpInfix,
	`@@`:                   OpInfix,
	`any`:                  OpAny,
	`between`:              OpBetween,
}

/*
Syntax type of SQL operator expressions used in JEL. Allows us to convert JEL
Lisp-style "calls" into SQL-style operations that use prefix, infix, etc.
*/
type Op byte

const (
	OpPrefix Op = iota + 1
	OpPostfix
	OpInfix
	OpFunc
	OpAny
	OpBetween
)

/*
Shortcut for instantiating `Jel` with the type of the given value. The input is
used only as a type carrier.
*/
func JelFor(typ interface{}) Jel { return Jel{Type: typeElemOf(typ)} }

/*
Short for "JSON Expression Language". Provides support for expressing a
whitelisted subset of SQL with JSON, as Lisp-style nested lists. Transcodes
JSON to SQL on the fly. Implements `Expr`. Can be transparently used as a
sub-expression in other `sqlb` expressions. See the provided example.

Expressions are Lisp-style, using nested lists to express "calls". This syntax
is used for all SQL operations. Binary infix operators are considered
variadic.

Lists are used for calls and casts. The first element must be a string. It may
be one of the whitelisted operators or functions, listed in `Ops`. If not, it
must be a field name or a dot-separated field path. Calls are arbitrarily
nestable.

	["and", true, ["or", true, ["and", true, false]]]

	["<=", 10, 20]

	["=", "someField", "otherField"]

	["and",
		["=", "someField", "otherField"],
		["<=", "dateField", ["dateField", "9999-01-01T00:00:00Z"]]
	]

Transcoding from JSON to SQL is done by consulting two things: the built-in
whitelist of SQL operations (`Ops`, shared), and a struct type provided to that
particular decoder. The struct serves as a whitelist of available identifiers,
and allows to determine value types via casting.

Casting allows to decode arbitrary JSON directly into the corresponding Go type:

	["someDateField", "9999-01-01T00:00:00Z"]

	["someGeoField", {"lng": 10, "lat": 20}]

Such decoded values are substituted with ordinal parameters such as $1, and
appended to the slice of arguments (see below).

A string not in a call position and not inside a cast is interpreted as an
identifier: field name or nested field path, dot-separated. It must be found on
the reference struct, otherwise transcoding fails with an error.

	"someField"

	"outerField.innerField"

Literal numbers, booleans, and nulls that occur outside of casts are decoded
into their Go equivalents. Like casts, they're substituted with ordinal
parameters and appended to the slice of arguments.

JSON queries are transcoded against a struct, by matching fields tagged with
`json` against fields tagged with `db`. Literal values are JSON-decoded into
the types of the corresponding struct fields.

	type Input struct {
		FieldOne string `json:"fieldOne" db:"field_one"`
		FieldTwo struct {
			FieldThree *time.Time `json:"fieldThree" db:"field_three"`
		} `json:"fieldTwo" db:"field_two"`
	}

	const src = `
		["and",
			["=", "fieldOne", ["fieldOne", "literal string"]],
			["<", "fieldTwo.fieldThree", ["fieldTwo.fieldThree", "9999-01-01T00:00:00Z"]]
		]
	`

	expr := Jel{Type: reflect.TypeOf((*Input)(nil)).Elem(), Text: src}
	text, args := Reify(expr)

The result is roughly equivalent to the following (formatted for clarity):

	text := `
		"field_one" = 'literal string'
		and
		("field_two")."field_three" < '9999-01-01T00:00:00Z'
	`
	args := []interface{}{"literal string", time.Time("9999-01-01T00:00:00Z")}
*/
type Jel struct {
	Type r.Type
	Text string
}

var _ = Expr(Jel{})

/*
Implement `Expr`, allowing this to be used as a sub-expression in queries built
with "github.com/mitranim/sqlb". Always generates a valid boolean expression,
falling back on "true" if empty.
*/
func (self Jel) AppendExpr(text []byte, args []interface{}) ([]byte, []interface{}) {
	bui := Bui{text, args}

	if len(self.Text) == 0 {
		bui.Str(`true`)
	} else {
		self.decode(&bui, stringToBytesUnsafe(self.Text))
	}

	return bui.Get()
}

// Implement the `Appender` interface, sometimes allowing more efficient text
// encoding.
func (self Jel) Append(text []byte) []byte { return exprAppend(&self, text) }

// Implement the `fmt.Stringer` interface for debug purposes.
func (self Jel) String() string { return exprString(&self) }

// Stores the input for future use in `.AppendExpr`. Input must be valid JSON.
func (self *Jel) Parse(val string) error {
	self.Text = val
	return nil
}

// Stores the input for future use in `.AppendExpr`. Input must be valid JSON.
func (self *Jel) UnmarshalText(val []byte) error {
	// TODO consider using unsafe conversions.
	self.Text = string(val)
	return nil
}

// Stores the input for future use in `.AppendExpr`. Input must be valid JSON.
func (self *Jel) UnmarshalJSON(val []byte) error {
	// TODO consider using unsafe conversions.
	self.Text = string(val)
	return nil
}

/*
If `.Type` is empty, sets the type of the provided value. Otherwise this is a
nop. The input is used only as a type carrier; its actual value is ignored.
*/
func (self *Jel) OrType(typ interface{}) {
	if self.Type == nil {
		self.Type = typeElemOf(typ)
	}
}

func (self *Jel) decode(bui *Bui, input []byte) {
	input = bytes.TrimSpace(input)

	if isJsonDict(input) {
		panic(ErrInvalidInput{Err{
			`decoding JEL`,
			fmt.Errorf(`unexpected dict in input: %q`, input),
		}})
	} else if isJsonList(input) {
		self.decodeList(bui, input)
	} else if isJsonString(input) {
		self.decodeString(bui, input)
	} else {
		self.decodeAny(bui, input)
	}
}

func (self *Jel) decodeList(bui *Bui, input []byte) {
	var list []json.RawMessage
	err := json.Unmarshal(input, &list)
	if err != nil {
		panic(ErrInvalidInput{Err{
			`decoding JEL list`,
			fmt.Errorf(`failed to unmarshal as JSON list: %w`, err),
		}})
	}

	if !(len(list) > 0) {
		panic(ErrInvalidInput{Err{
			`decoding JEL list`,
			fmt.Errorf(`lists must have at least one element, found empty list`),
		}})
	}

	head, args := list[0], list[1:]
	if !isJsonString(head) {
		panic(ErrInvalidInput{Err{
			`decoding JEL list`,
			fmt.Errorf(`first list element must be a string, found %q`, head),
		}})
	}

	var name string
	err = json.Unmarshal(head, &name)
	if err != nil {
		panic(ErrInvalidInput{Err{
			`decoding JEL list`,
			fmt.Errorf(`failed to unmarshal JSON list head as string: %w`, err),
		}})
	}

	switch Ops[name] {
	case OpPrefix:
		self.decodeOpPrefix(bui, name, args)
	case OpPostfix:
		self.decodeOpPostfix(bui, name, args)
	case OpInfix:
		self.decodeOpInfix(bui, name, args)
	case OpFunc:
		self.decodeOpFunc(bui, name, args)
	case OpAny:
		self.decodeOpAny(bui, name, args)
	case OpBetween:
		self.decodeOpBetween(bui, name, args)
	default:
		self.decodeCast(bui, name, args)
	}
}

func (self *Jel) decodeOpPrefix(bui *Bui, name string, args []json.RawMessage) {
	if len(args) != 1 {
		panic(ErrInvalidInput{Err{
			`decoding JEL op (prefix)`,
			fmt.Errorf(`prefix operation %q must have exactly 1 argument, found %v`, name, len(args)),
		}})
	}

	bui.Str(`(`)
	bui.Str(name)
	self.decode(bui, args[0])
	bui.Str(`)`)
}

func (self *Jel) decodeOpPostfix(bui *Bui, name string, args []json.RawMessage) {
	if len(args) != 1 {
		panic(ErrInvalidInput{Err{
			`decoding JEL op (postfix)`,
			fmt.Errorf(`postfix operation %q must have exactly 1 argument, found %v`, name, len(args)),
		}})
	}

	bui.Str(`(`)
	self.decode(bui, args[0])
	bui.Str(name)
	bui.Str(`)`)
}

func (self *Jel) decodeOpInfix(bui *Bui, name string, args []json.RawMessage) {
	if !(len(args) >= 2) {
		panic(ErrInvalidInput{Err{
			`decoding JEL op (infix)`,
			fmt.Errorf(`infix operation %q must have at least 2 arguments, found %v`, name, len(args)),
		}})
	}

	bui.Str(`(`)
	for i, arg := range args {
		if i > 0 {
			bui.Str(name)
		}
		self.decode(bui, arg)
	}
	bui.Str(`)`)
}

func (self *Jel) decodeOpFunc(bui *Bui, name string, args []json.RawMessage) {
	bui.Str(name)
	bui.Str(`(`)
	for i, arg := range args {
		if i > 0 {
			bui.Str(`,`)
		}
		self.decode(bui, arg)
	}
	bui.Str(`)`)
}

func (self *Jel) decodeOpAny(bui *Bui, name string, args []json.RawMessage) {
	if len(args) != 2 {
		panic(ErrInvalidInput{Err{
			`decoding JEL op`,
			fmt.Errorf(`operation %q must have exactly 2 arguments, found %v`, name, len(args)),
		}})
	}

	bui.Str(`(`)
	self.decode(bui, args[0])
	bui.Str(`=`)
	bui.Str(name)
	bui.Str(`(`)
	self.decode(bui, args[1])
	bui.Str(`)`)
	bui.Str(`)`)
}

func (self *Jel) decodeOpBetween(bui *Bui, name string, args []json.RawMessage) {
	if len(args) != 3 {
		panic(ErrInvalidInput{Err{
			`decoding JEL op (between)`,
			fmt.Errorf(`operation %q must have exactly 3 arguments, found %v`, name, len(args)),
		}})
	}

	bui.Str(`(`)
	self.decode(bui, args[0])
	bui.Str(`between`)
	self.decode(bui, args[1])
	bui.Str(`and`)
	self.decode(bui, args[2])
	bui.Str(`)`)
}

func (self *Jel) decodeCast(bui *Bui, name string, args []json.RawMessage) {
	if len(args) != 1 {
		panic(ErrInvalidInput{Err{
			`decoding JEL op (cast)`,
			fmt.Errorf(`cast into %q must have exactly 1 argument, found %v`, name, len(args)),
		}})
	}

	typ := self.Type
	field, ok := loadStructJsonPathToNestedDbFieldMap(typ)[name]
	if !ok {
		panic(errUnknownField(`decoding JEL op (cast)`, name, typeName(typ)))
	}

	val := r.New(field.Field.Type)
	try(json.Unmarshal(args[0], val.Interface()))

	bui.Param(bui.Arg(val.Elem().Interface()))
}

func (self *Jel) decodeString(bui *Bui, input []byte) {
	var str string
	try(json.Unmarshal(input, &str))

	typ := self.Type
	val, ok := loadStructJsonPathToNestedDbFieldMap(typ)[str]
	if !ok {
		panic(errUnknownField(`decoding JEL string`, str, typeName(typ)))
	}

	bui.Set(Path(val.DbPath).AppendExpr(bui.Get()))
}

// Should be used only for numbers, bools, nulls.
// TODO: unmarshal integers into `int64` rather than `float64`.
func (self *Jel) decodeAny(bui *Bui, input []byte) {
	var val interface{}
	try(json.Unmarshal(input, &val))
	bui.Param(bui.Arg(val))
}
