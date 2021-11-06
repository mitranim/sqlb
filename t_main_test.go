package sqlb

import (
	"fmt"
	r "reflect"
	"runtime"
	"strings"
	"testing"
	"time"
	"unsafe"
)

type Internal struct {
	Id   string `json:"internalId"   db:"id"`
	Name string `json:"internalName" db:"name"`
}

type External struct {
	Id       string   `json:"externalId"       db:"id"`
	Name     string   `json:"externalName"     db:"name"`
	Internal Internal `json:"externalInternal" db:"internal"`
}

// nolint:govet
type Embed struct {
	Id        string `json:"embedId"      db:"embed_id"`
	Name      string `json:"embedName"    db:"embed_name"`
	private   string `json:"embedPrivate" db:"embed_private"`
	Untagged0 string ``
	Untagged1 string `db:"-"`
	_         string `db:"blank"`
}

type Outer struct {
	Embed
	Id       string `json:"outerId"   db:"outer_id"`
	Name     string `json:"outerName" db:"outer_name"`
	OnlyJson string `json:"onlyJson"`
}

var testOuter = Outer{
	Id:   `outer id`,
	Name: `outer name`,
	Embed: Embed{
		Id:        `embed id`,
		Name:      `embed name`,
		private:   `private`,
		Untagged0: `untagged 0`,
		Untagged1: `untagged 1`,
	},
}

type Void struct{}

func (Void) GetVal() interface{} { return `val` }

type UnitStruct struct {
	One interface{} `db:"one" json:"one"`
}

func (self UnitStruct) GetOne() interface{}             { return self.One }
func (self UnitStruct) UnaryVoid(interface{})           {}
func (self UnitStruct) NullaryPair() (_, _ interface{}) { return }

type PairStruct struct {
	One interface{} `db:"one" json:"one"`
	Two interface{} `db:"two" json:"two"`
}

func (self PairStruct) GetOne() interface{} { return self.One }
func (self PairStruct) GetTwo() interface{} { return self.Two }

type TrioStruct struct {
	One   interface{} `db:"one" json:"one"`
	Two   interface{} `db:"two" json:"two"`
	Three interface{} `db:"three" json:"three"`
}

func (self TrioStruct) GetOne() interface{}   { return self.One }
func (self TrioStruct) GetTwo() interface{}   { return self.Two }
func (self TrioStruct) GetThree() interface{} { return self.Three }

type list = []interface{}

type Encoder interface {
	fmt.Stringer
	Appender
	Expr
}

func testEncoder(t testing.TB, exp string, val Encoder) {
	t.Helper()
	eq(t, exp, val.String())
	eq(t, exp, string(val.Append(nil)))
	eq(t, exp, string(reify(val).Text))
}

func testExpr(t testing.TB, exp R, val Encoder) {
	t.Helper()
	testEncoder(t, string(exp.Text), val)
	testExprs(t, exp, val)
}

func testExprs(t testing.TB, exp R, vals ...Expr) {
	t.Helper()
	eq(t, exp, reify(vals...))
}

func reify(vals ...Expr) R {
	text, args := Reify(vals...)
	return R{text, args}.Norm()
}

// Short for "reified".
func rei(text string, args ...interface{}) R { return R{text, args}.Norm() }

func reiFrom(text []byte, args []interface{}) R {
	return R{bytesToMutableString(text), args}.Norm()
}

func reifyParamExpr(expr ParamExpr, dict ArgDict) R {
	return reiFrom(expr.AppendParamExpr(nil, nil, dict))
}

func reifyUnparamPreps(vals ...string) (text []byte, args []interface{}) {
	for _, val := range vals {
		text, args = Preparse(val).AppendParamExpr(text, args, nil)
	}
	return
}

/*
Short for "reified". Test-only for now. Similar to `StrQ` but uses ordinal params.
Implements `Expr` incorrectly, see below.
*/
type R struct {
	Text string
	Args list
}

/*
Note: this is NOT a valid implementation of an expr with ordinal params. When
the input args are non-empty, a real implementation would have to parse its own
text to renumerate the params, appending that modified text to the output.
*/
func (self R) AppendExpr(text []byte, args list) ([]byte, list) {
	text = append(text, self.Text...)
	args = append(args, self.Args...)
	return text, args
}

/*
Without this equivalence, tests break due to slice prealloc/growth in
`StrQ.AppendExpr`, violating some equality tests. We don't really care about the
difference between nil and zero-length arg lists.
*/
func (self R) Norm() R {
	if self.Args == nil {
		self.Args = list{}
	}
	return self
}

func eq(t testing.TB, exp, act interface{}) {
	t.Helper()
	if !r.DeepEqual(exp, act) {
		t.Fatalf(`
expected (detailed):
	%#[1]v
actual (detailed):
	%#[2]v
expected (simple):
	%[1]v
actual (simple):
	%[2]v
`, exp, act)
	}
}

func is(t testing.TB, exp, act interface{}) {
	t.Helper()

	expIface := *(*iface)(unsafe.Pointer(&exp))
	actIface := *(*iface)(unsafe.Pointer(&act))

	if expIface != actIface {
		t.Fatalf(`
expected (interface):
	%#[1]v
actual (interface):
	%#[2]v
expected (detailed):
	%#[3]v
actual (detailed):
	%#[4]v
expected (simple):
	%[3]v
actual (simple):
	%[4]v
`, expIface, actIface, exp, act)
	}
}

// nolint:structcheck
type iface struct {
	typ unsafe.Pointer
	dat unsafe.Pointer
}

func notEq(t testing.TB, exp, act interface{}) {
	t.Helper()
	if r.DeepEqual(exp, act) {
		fatalNotEq(t, exp, act)
	}
}

func fatalNotEq(t testing.TB, exp, act interface{}) {
	t.Helper()
	t.Fatalf(`
unexpected equality (detailed):
	%#[1]v
unexpected equality (simple):
	%[1]v
`, exp, act)
}

func panics(t testing.TB, msg string, fun func()) {
	t.Helper()
	val := catchAny(fun)

	if val == nil {
		t.Fatalf(`expected %v to panic, found no panic`, funcName(fun))
	}

	str := fmt.Sprint(val)
	if !strings.Contains(str, msg) {
		t.Fatalf(
			`expected %v to panic with a message containing %q, found %q`,
			funcName(fun), msg, str,
		)
	}
}

func funcName(val interface{}) string {
	return runtime.FuncForPC(r.ValueOf(val).Pointer()).Name()
}

func catchAny(fun func()) (val interface{}) {
	defer recAny(&val)
	fun()
	return
}

func recAny(ptr *interface{}) { *ptr = recover() }

var hugeBui = MakeBui(len(hugeQuery)*2, len(hugeQueryArgs)*2)

func tHugeBuiEmpty(t testing.TB) {
	eq(t, Bui{[]byte{}, list{}}, hugeBui)
}

func parseTime(str string) *time.Time {
	inst, err := time.Parse(time.RFC3339, str)
	try(err)
	return &inst
}

const hugeQuery = /*pgsql*/ `
	select col_name
	from
		table_name

		left join table_name using (col_name)

		inner join (
			select agg(col_name) as col_name
			from table_name
			where (
				false
				or col_name = 'enum_value'
				or (:arg_one and (:arg_two or col_name = :arg_three))
			)
			group by col_name
		) as table_name using (col_name)

		left join (
			select
				table_name.col_name
			from
				table_name
				left join table_name on table_name.col_name = table_name.col_name
			where
				false
				or :arg_four::type_name is null
				or table_name.col_name between :arg_four and (:arg_four + 'literal input'::some_type)
		) as table_name using (col_name)

		left join (
			select distinct col_name as col_name
			from table_name
			where (:arg_five::type_name[] is null or col_name = any(:arg_five))
		) as table_name using (col_name)

		left join (
			select distinct col_name as col_name
			from table_name
			where (:arg_six::type_name[] is null or col_name = any(:arg_six))
		) as table_name using (col_name)
	where
		true
		and (:arg_seven or col_name in (table table_name))
		and (:arg_four :: type_name   is null or table_name.col_name is not null)
		and (:arg_five :: type_name[] is null or table_name.col_name is not null)
		and (:arg_six  :: type_name[] is null or table_name.col_name is not null)
		and (
			false
			or not col_name
			or (:arg_eight and (:arg_two or col_name = :arg_three))
		)
		and (
			false
			or not col_name
			or (:arg_nine and (:arg_two or col_name = :arg_three))
		)
		and (:arg_ten or not col_name)
		and (:arg_eleven   :: type_name is null or col_name            @@ func_name(:arg_eleven))
		and (:arg_fifteen  :: type_name is null or col_name            <> :arg_fifteen)
		and (:arg_sixteen  :: type_name is null or col_name            =  :arg_sixteen)
		and (:arg_twelve   :: type_name is null or col_name            =  :arg_twelve)
		and (:arg_thirteen :: type_name is null or func_name(col_name) <= :arg_thirteen)
	:arg_fourteen
`

var hugeQueryArgs = Dict{
	`arg_one`:      nil,
	`arg_two`:      nil,
	`arg_three`:    nil,
	`arg_four`:     nil,
	`arg_five`:     nil,
	`arg_six`:      nil,
	`arg_seven`:    nil,
	`arg_eight`:    nil,
	`arg_nine`:     nil,
	`arg_ten`:      nil,
	`arg_eleven`:   nil,
	`arg_twelve`:   nil,
	`arg_thirteen`: nil,
	`arg_fourteen`: nil,
	`arg_fifteen`:  nil,
	`arg_sixteen`:  nil,
}

type HaserTrue struct{}

func (HaserTrue) Has(string) bool { return true }

type HaserFalse struct{}

func (HaserFalse) Has(string) bool { return false }
