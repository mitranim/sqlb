package sqlb

import (
	"fmt"
	r "reflect"
	"testing"
	"time"
)

func Test_Reify(t *testing.T) {
	t.Run(`nil`, func(t *testing.T) {
		text, args := Reify(nil)
		eq(t, ``, text)
		eq(t, list(nil), args)
	})

	t.Run(`empty`, func(t *testing.T) {
		text, args := Reify(rei(``))
		eq(t, ``, text)
		eq(t, list(nil), args)
	})

	t.Run(`non-empty`, func(t *testing.T) {
		text, args := Reify(rei(`select A from B where C`, 10, 20, 30))
		eq(t, `select A from B where C`, text)
		eq(t, list{10, 20, 30}, args)
	})

	t.Run(`multiple`, func(t *testing.T) {
		text, args := Reify(
			nil,
			rei(`one`, 10),
			nil,
			rei(`two`, 20),
			nil,
		)
		eq(t, `one two`, text)
		eq(t, list{10, 20}, args)
	})
}

func Test_Str(t *testing.T) {
	testExpr(t, rei(``), Str(``))
	testExpr(t, rei(`one`), Str(`one`))

	testExprs(t, rei(``))
	testExprs(t, rei(``), Str(``))
	testExprs(t, rei(`one`), Str(`one`))
	testExprs(t, rei(`one_two`), Str(`one_two`))
	testExprs(t, rei(`one_two three_four`), Str(`one_two`), Str(`three_four`))
}

func Test_Ident(t *testing.T) {
	testExpr(t, rei(`""`), Ident(``))
	testExpr(t, rei(`" "`), Ident(` `))
	testExpr(t, rei(`"one.two"`), Ident(`one.two`))
	testExpr(t, rei(`"one.two.three"`), Ident(`one.two.three`))

	testExprs(t, rei(``))
	testExprs(t, rei(`""`), Ident(``))
	testExprs(t, rei(`"" ""`), Ident(``), Ident(``))
	testExprs(t, rei(`"one" ""`), Ident(`one`), Ident(``))
	testExprs(t, rei(`"" "two"`), Ident(``), Ident(`two`))
	testExprs(t, rei(`"one" "two"`), Ident(`one`), Ident(`two`))
}

func Test_Identifier(t *testing.T) {
	testExpr(t, rei(``), Identifier(nil))
	testExpr(t, rei(``), Identifier{})
	testExpr(t, rei(`""`), Identifier{``})
	testExpr(t, rei(`"one"`), Identifier{`one`})
	testExpr(t, rei(`"one".""`), Identifier{`one`, ``})
	testExpr(t, rei(`""."two"`), Identifier{``, `two`})
	testExpr(t, rei(`"one"."two"`), Identifier{`one`, `two`})
	testExpr(t, rei(`"one".""."three"`), Identifier{`one`, ``, `three`})
	testExpr(t, rei(`"one"."two"."three"`), Identifier{`one`, `two`, `three`})

	testExprs(
		t,
		rei(`"one" "two"."three"`),
		Identifier(nil),
		Identifier{},
		Identifier{`one`},
		Identifier{`two`, `three`},
	)
}

func Test_Path(t *testing.T) {
	testExpr(t, rei(``), Path(nil))
	testExpr(t, rei(``), Path{})
	testExpr(t, rei(`""`), Path{``})
	testExpr(t, rei(`"one"`), Path{`one`})
	testExpr(t, rei(`("one").""`), Path{`one`, ``})
	testExpr(t, rei(`("")."two"`), Path{``, `two`})
	testExpr(t, rei(`("one")."two"`), Path{`one`, `two`})
	testExpr(t, rei(`("one").""."three"`), Path{`one`, ``, `three`})
	testExpr(t, rei(`("one")."two"."three"`), Path{`one`, `two`, `three`})

	testExprs(
		t,
		rei(`"one" ("two")."three"`),
		Path(nil),
		Path{},
		Path{`one`},
		Path{`two`, `three`},
	)
}

func Test_PseudoPath(t *testing.T) {
	testExpr(t, rei(``), PseudoPath(nil))
	testExpr(t, rei(``), PseudoPath{})
	testExpr(t, rei(`""`), PseudoPath{``})
	testExpr(t, rei(`"one"`), PseudoPath{`one`})
	testExpr(t, rei(`"one.two"`), PseudoPath{`one`, `two`})
	testExpr(t, rei(`".."`), PseudoPath{``, ``, ``})
	testExpr(t, rei(`".two."`), PseudoPath{``, `two`, ``})
	testExpr(t, rei(`"one.two.three"`), PseudoPath{`one`, `two`, `three`})

	testExprs(
		t,
		rei(`$1 "one.two.three" $2`, 10, 20),
		PseudoPath(nil),
		rei(`$1`, 10),
		PseudoPath{`one`, `two`, `three`},
		rei(`$2`, 20),
	)
}

func Test_AliasedPath(t *testing.T) {
	testExpr(t, rei(``), AliasedPath(nil))
	testExpr(t, rei(``), AliasedPath{})
	testExpr(t, rei(`""`), AliasedPath{``})
	testExpr(t, rei(`"one"`), AliasedPath{`one`})
	testExpr(t, rei(`("one")."two" as "one.two"`), AliasedPath{`one`, `two`})
	testExpr(t, rei(`("one")."two"."three" as "one.two.three"`), AliasedPath{`one`, `two`, `three`})

	testExprs(
		t,
		rei(`$1 "one" ("one")."two"."three" as "one.two.three" $2`, 10, 20),
		AliasedPath(nil),
		rei(`$1`, 10),
		AliasedPath{`one`},
		AliasedPath{`one`, `two`, `three`},
		rei(`$2`, 20),
	)
}

func Test_Table(t *testing.T) {
	testExpr(t, rei(``), Table(nil))
	testExpr(t, rei(``), Table{})
	testExpr(t, rei(`table ""`), Table{``})
	testExpr(t, rei(`table "one"`), Table{`one`})
	testExpr(t, rei(`table "one"."two"`), Table{`one`, `two`})
	testExpr(t, rei(`table ""."two"`), Table{``, `two`})
	testExpr(t, rei(`table "one".""`), Table{`one`, ``})
	testExpr(t, rei(`table "one"."two"."three"`), Table{`one`, `two`, `three`})

	testExprs(
		t,
		rei(`$1 table "one" table "two" $2`, 10, 20),
		rei(`$1`, 10),
		Table{},
		Table{`one`},
		Table{},
		Table{`two`},
		Table{},
		rei(`$2`, 20),
	)
}

func Test_Exprs(t *testing.T) {
	testExprs(t, rei(``))
	testExprs(t, rei(``), Exprs{})
	testExprs(t, rei(``), Exprs{nil})

	testExprs(
		t,
		rei(`one two`, 10, 20),
		Exprs{
			nil,
			rei(`one`, 10),
			nil,
			rei(`two`, 20),
			nil,
		},
	)
}

func Test_Any(t *testing.T) {
	testExpr(t, rei(`any ($1)`, nil), Any{})
	testExpr(t, rei(`any ($1)`, list{10, 20, 30}), Any{list{10, 20, 30}})
	testExpr(t, rei(`any (one)`), Any{Str(`one`)})

	testExprs(
		t,
		rei(`any ($1) any ($2)`, 10, 20),
		Any{10},
		Any{20},
	)
}

func Test_Assign(t *testing.T) {
	testExpr(t, rei(`"" = $1`, nil), Assign{})
	testExpr(t, rei(`"one" = $1`, nil), Assign{`one`, nil})
	testExpr(t, rei(`"" = $1`, 10), Assign{``, 10})
	testExpr(t, rei(`"one" = (two)`), Assign{`one`, Str(`two`)})

	testExprs(
		t,
		rei(`"" = $1 "one" = $2 "two" = (three) "four" = $3`, nil, 10, 40),
		Assign{},
		Assign{`one`, 10},
		Assign{`two`, Str(`three`)},
		Assign{`four`, 40},
	)
}

func Test_Eq(t *testing.T) {
	testExpr(t, rei(`$1 is null`, nil), Eq{nil, nil})
	testExpr(t, rei(`$1 is null`, 10), Eq{10, nil})
	testExpr(t, rei(`$1 = $2`, nil, 10), Eq{nil, 10})
	testExpr(t, rei(`$1 = $2`, true, false), Eq{true, false})
	testExpr(t, rei(`$1 = $2`, 10, 20), Eq{10, 20})
	testExpr(t, rei(`$1 = $2`, 10, []int{20}), Eq{10, []int{20}})
	testExpr(t, rei(`$1 = $2`, []int{10}, 20), Eq{[]int{10}, 20})
	testExpr(t, rei(`$1 = $2`, []int{10}, []int{20}), Eq{[]int{10}, []int{20}})
	testExpr(t, rei(`(one) is null`), Eq{Str(`one`), nil})
	testExpr(t, rei(`(one) = $1`, 10), Eq{Str(`one`), 10})
	testExpr(t, rei(`$1 = (two)`, 10), Eq{10, Str(`two`)})
	testExpr(t, rei(`(one) = (two)`), Eq{Str(`one`), Str(`two`)})

	testExprs(
		t,
		rei(`$1 = $2 $3 = $4`, 10, 20, 30, 40),
		Eq{10, 20},
		Eq{30, 40},
	)
}

func Test_Neq(t *testing.T) {
	testExpr(t, rei(`$1 is not null`, nil), Neq{nil, nil})
	testExpr(t, rei(`$1 is not null`, 10), Neq{10, nil})
	testExpr(t, rei(`$1 <> $2`, nil, 10), Neq{nil, 10})
	testExpr(t, rei(`$1 <> $2`, true, false), Neq{true, false})
	testExpr(t, rei(`$1 <> $2`, 10, 20), Neq{10, 20})
	testExpr(t, rei(`$1 <> $2`, 10, []int{20}), Neq{10, []int{20}})
	testExpr(t, rei(`$1 <> $2`, []int{10}, 20), Neq{[]int{10}, 20})
	testExpr(t, rei(`$1 <> $2`, []int{10}, []int{20}), Neq{[]int{10}, []int{20}})
	testExpr(t, rei(`(one) is not null`), Neq{Str(`one`), nil})
	testExpr(t, rei(`(one) <> $1`, 10), Neq{Str(`one`), 10})
	testExpr(t, rei(`$1 <> (two)`, 10), Neq{10, Str(`two`)})
	testExpr(t, rei(`(one) <> (two)`), Neq{Str(`one`), Str(`two`)})

	testExprs(
		t,
		rei(`$1 <> $2 $3 <> $4`, 10, 20, 30, 40),
		Neq{10, 20},
		Neq{30, 40},
	)
}

func Test_EqAny(t *testing.T) {
	testExpr(t, rei(`$1 = any ($2)`, nil, nil), EqAny{})
	testExpr(t, rei(`$1 = any ($2)`, 10, 20), EqAny{10, 20})
	testExpr(t, rei(`(one) = any ($1)`, 20), EqAny{Str(`one`), 20})
	testExpr(t, rei(`$1 = any (two)`, 10), EqAny{10, Str(`two`)})
	testExpr(t, rei(`(one) = any (two)`), EqAny{Str(`one`), Str(`two`)})

	testExprs(
		t,
		rei(`$1 = any ($2) $3 = any ($4)`, 10, 20, 30, 40),
		EqAny{10, 20},
		EqAny{30, 40},
	)
}

func Test_NeqAny(t *testing.T) {
	testExpr(t, rei(`$1 <> any ($2)`, nil, nil), NeqAny{})
	testExpr(t, rei(`$1 <> any ($2)`, 10, 20), NeqAny{10, 20})
	testExpr(t, rei(`(one) <> any ($1)`, 20), NeqAny{Str(`one`), 20})
	testExpr(t, rei(`$1 <> any (two)`, 10), NeqAny{10, Str(`two`)})
	testExpr(t, rei(`(one) <> any (two)`), NeqAny{Str(`one`), Str(`two`)})

	testExprs(
		t,
		rei(`$1 <> any ($2) $3 <> any ($4)`, 10, 20, 30, 40),
		NeqAny{10, 20},
		NeqAny{30, 40},
	)
}

func Test_Not(t *testing.T) {
	testExpr(t, rei(`not $1`, nil), Not{})
	testExpr(t, rei(`not $1`, 10), Not{10})
	testExpr(t, rei(`not ()`), Not{Str(``)})
	testExpr(t, rei(`not (one)`), Not{Str(`one`)})

	testExprs(
		t,
		rei(`not $1 not $2`, nil, nil),
		Not{},
		Not{},
	)

	testExprs(
		t,
		rei(`not $1 not (two) not $2`, 10, 20),
		Not{10},
		Not{Str(`two`)},
		Not{20},
	)
}

func Test_Seq(t *testing.T) {
	testExpr(t, rei(``), Seq{})
	testExpr(t, rei(`empty`), Seq{Empty: `empty`})
	testExpr(t, rei(`empty`), Seq{`empty`, `delim`, list(nil)})
	testExpr(t, rei(`empty`), Seq{`empty`, `delim`, list{}})
	testExpr(t, rei(`$1`, nil), Seq{`empty`, `delim`, list{nil}})
	testExpr(t, rei(`$1`, 10), Seq{`empty`, `delim`, list{10}})
	testExpr(t, rei(`one`), Seq{Val: Str(`one`)})
	testExpr(t, rei(`one`, 10), Seq{Val: rei(`one`, 10)})

	testExpr(
		t,
		rei(`$1 delim $2`, 10, 20),
		Seq{`empty`, `delim`, list{10, 20}},
	)

	testExpr(
		t,
		rei(`$1 delim $2 delim $3`, 10, 20, 30),
		Seq{`empty`, `delim`, list{10, 20, 30}},
	)

	testExpr(
		t,
		rei(`(one) delim $1 delim (two)`, 10),
		Seq{`empty`, `delim`, list{Str(`one`), 10, Str(`two`)}},
	)

	testExprs(
		t,
		rei(`one two`),
		Str(`one`),
		Seq{},
		Str(`two`),
	)

	testExprs(
		t,
		rei(`one empty two`),
		Str(`one`),
		Seq{`empty`, ``, nil},
		Str(`two`),
	)

	testExprs(
		t,
		rei(`one $1 two`, 10, 20),
		Str(`one`),
		Seq{`empty`, `delim`, list{10}},
		rei(`two`, 20),
	)
}

func Test_Comma(t *testing.T) {
	testExpr(t, rei(``), Comma{})
	testExpr(t, rei(``), Comma{Comma{Comma{}}})
	testExpr(t, rei(``), Comma{list{}})
	testExpr(t, rei(``), Comma{list{Comma{list{}}}})
	testExpr(t, rei(`one`), Comma{Str(`one`)})
	testExpr(t, rei(`one`, 10), Comma{rei(`one`, 10)})
	testExpr(t, rei(`$1`, 10), Comma{list{10}})
	testExpr(t, rei(`$1, $2`, 10, 20), Comma{list{10, 20}})
	testExpr(t, rei(`$1, $2, $3`, 10, 20, 30), Comma{list{10, 20, 30}})
	testExpr(t, rei(`(one), $1`, 10), Comma{list{Str(`one`), 10}})
	testExpr(t, rei(`$1, (one)`, 10), Comma{list{10, Str(`one`)}})
}

func Test_And(t *testing.T) {
	testExpr(t, rei(`true`), And{})

	t.Run(`slice`, func(t *testing.T) {
		testExpr(t, rei(`$1`, nil), And{list{nil}})
		testExpr(t, rei(`$1`, 10), And{[]int{10}})
		testExpr(t, rei(`$1 and $2`, 10, 20), And{[]int{10, 20}})
		testExpr(t, rei(`$1 and $2 and $3`, 10, 20, 30), And{[]int{10, 20, 30}})
		testExpr(t, rei(`one`), And{[]Str{`one`}})
		testExpr(t, rei(`(one) and (two)`), And{[]Str{`one`, `two`}})
		testExpr(t, rei(`(one) and (two) and (three)`), And{[]Str{`one`, `two`, `three`}})

		testExpr(
			t,
			rei(`$1 and (one) and $2 and (two) and $3`, 10, 20, 30),
			And{list{10, Str(`one`), 20, Str(`two`), 30}},
		)

		testExprs(t, rei(`true true`), And{}, And{})
		testExprs(t, rei(`true $1 and $2`, 10, 20), And{}, And{[]int{10, 20}})
		testExprs(t, rei(`$1 and $2 true`, 10, 20), And{[]int{10, 20}}, And{})
		testExprs(t, rei(`$1 and $2 $3 and $4`, 10, 20, 30, 40), And{[]int{10, 20}}, And{[]int{30, 40}})
	})

	t.Run(`struct`, func(t *testing.T) {
		testExpr(t, rei(`true`), And{Void{}})
		testExpr(t, rei(`true`), And{struct{ _ string }{}})
		testExpr(t, rei(`true`), And{struct{ Col string }{}})

		testExpr(t, rei(`true`), And{struct {
			_ string `db:"col"`
		}{}})

		testExpr(t, rei(`"col" = $1`, ``), And{struct {
			Col string `db:"col"`
		}{}})

		testExpr(t, rei(`"col" is null`), And{struct {
			Col *string `db:"col"`
		}{}})

		str := `one`
		testExpr(t, rei(`"col" = $1`, &str), And{struct {
			Col *string `db:"col"`
		}{&str}})

		testExpr(
			t,
			rei(
				`"one" = $1 and "embed_id" = $2 and "embed_name" = $3 and "two" = $4`,
				`outer one`, `embed id`, `embed name`, 20,
			),
			And{
				struct {
					One string `db:"one"`
					Embed
					Two int `db:"two"`
				}{
					One: `outer one`,
					Embed: Embed{
						Id:   `embed id`,
						Name: `embed name`,
					},
					Two: 20,
				},
			},
		)
	})
}

func Test_Or(t *testing.T) {
	testExpr(t, rei(`false`), Or{})

	t.Run(`slice`, func(t *testing.T) {
		testExpr(t, rei(`$1`, nil), Or{list{nil}})
		testExpr(t, rei(`$1`, 10), Or{[]int{10}})
		testExpr(t, rei(`$1 or $2`, 10, 20), Or{[]int{10, 20}})
		testExpr(t, rei(`$1 or $2 or $3`, 10, 20, 30), Or{[]int{10, 20, 30}})
		testExpr(t, rei(`one`), Or{[]Str{`one`}})
		testExpr(t, rei(`(one) or (two)`), Or{[]Str{`one`, `two`}})
		testExpr(t, rei(`(one) or (two) or (three)`), Or{[]Str{`one`, `two`, `three`}})

		testExpr(
			t,
			rei(`$1 or (one) or $2 or (two) or $3`, 10, 20, 30),
			Or{list{10, Str(`one`), 20, Str(`two`), 30}},
		)

		testExprs(t, rei(`false false`), Or{}, Or{})
		testExprs(t, rei(`false $1 or $2`, 10, 20), Or{}, Or{[]int{10, 20}})
		testExprs(t, rei(`$1 or $2 false`, 10, 20), Or{[]int{10, 20}}, Or{})
		testExprs(t, rei(`$1 or $2 $3 or $4`, 10, 20, 30, 40), Or{[]int{10, 20}}, Or{[]int{30, 40}})
	})

	t.Run(`struct`, func(t *testing.T) {
		testExpr(t, rei(`false`), Or{Void{}})
		testExpr(t, rei(`false`), Or{struct{ _ string }{}})
		testExpr(t, rei(`false`), Or{struct{ Col string }{}})

		testExpr(t, rei(`false`), Or{struct {
			_ string `db:"col"`
		}{}})

		testExpr(t, rei(`"col" = $1`, ``), Or{struct {
			Col string `db:"col"`
		}{}})

		testExpr(t, rei(`"col" is null`), Or{struct {
			Col *string `db:"col"`
		}{}})

		str := `one`
		testExpr(t, rei(`"col" = $1`, &str), Or{struct {
			Col *string `db:"col"`
		}{&str}})

		testExpr(
			t,
			rei(
				`"one" = $1 or "embed_id" = $2 or "embed_name" = $3 or "two" = $4`,
				`outer one`, `embed id`, `embed name`, 20,
			),
			Or{
				struct {
					One string `db:"one"`
					Embed
					Two int `db:"two"`
				}{
					One: `outer one`,
					Embed: Embed{
						Id:   `embed id`,
						Name: `embed name`,
					},
					Two: 20,
				},
			},
		)
	})
}

func Test_Ands(t *testing.T) {
	testExpr(t, rei(`true`), Ands{})
	testExpr(t, rei(`$1`, 10), Ands{10})
	testExpr(t, rei(`$1 and $2`, 10, 20), Ands{10, 20})
	testExpr(t, rei(`$1 and $2 and $3`, 10, 20, 30), Ands{10, 20, 30})
	testExpr(t, rei(`true`), Ands{Ands{Ands{}}})
	testExpr(t, rei(`$1`, 10), Ands{Ands{Ands{10}}})
	testExpr(t, rei(`(true) and ($1)`, 10), Ands{Ands{Ands{}}, Ands{Ands{10}}})
	testExpr(t, rei(`($1) and (true)`, 10), Ands{Ands{Ands{10}}, Ands{Ands{}}})
	testExpr(t, rei(`($1) and ($2)`, 10, 20), Ands{Ands{Ands{10}}, Ands{Ands{20}}})
	testExpr(t, rei(`$1 and $2`, 10, 20), Ands{Ands{10, 20}})
	testExpr(t, rei(`($1 and $2) and ($3 and $4)`, 10, 20, 30, 40), Ands{Ands{10, 20}, Ands{30, 40}})
}

func Test_Ors(t *testing.T) {
	testExpr(t, rei(`false`), Ors{})
	testExpr(t, rei(`$1`, 10), Ors{10})
	testExpr(t, rei(`$1 or $2`, 10, 20), Ors{10, 20})
	testExpr(t, rei(`$1 or $2 or $3`, 10, 20, 30), Ors{10, 20, 30})
	testExpr(t, rei(`false`), Ors{Ors{Ors{}}})
	testExpr(t, rei(`$1`, 10), Ors{Ors{Ors{10}}})
	testExpr(t, rei(`(false) or ($1)`, 10), Ors{Ors{Ors{}}, Ors{Ors{10}}})
	testExpr(t, rei(`($1) or (false)`, 10), Ors{Ors{Ors{10}}, Ors{Ors{}}})
	testExpr(t, rei(`($1) or ($2)`, 10, 20), Ors{Ors{Ors{10}}, Ors{Ors{20}}})
	testExpr(t, rei(`$1 or $2`, 10, 20), Ors{Ors{10, 20}})
	testExpr(t, rei(`($1 or $2) or ($3 or $4)`, 10, 20, 30, 40), Ors{Ors{10, 20}, Ors{30, 40}})
}

func Test_Cond(t *testing.T) {
	testExpr(t, rei(``), Cond{})
	testExpr(t, rei(`empty`), Cond{Empty: `empty`})
	testExpr(t, rei(`one`), Cond{Val: Str(`one`)})
	testExpr(t, rei(`one`, 10), Cond{Val: rei(`one`, 10)})

	t.Run(`slice`, func(t *testing.T) {
		testExpr(t, rei(`empty`), Cond{`empty`, `delim`, list(nil)})
		testExpr(t, rei(`empty`), Cond{`empty`, `delim`, list{}})
		testExpr(t, rei(`$1`, nil), Cond{`empty`, `delim`, list{nil}})
		testExpr(t, rei(`$1`, 10), Cond{`empty`, `delim`, list{10}})

		testExpr(
			t,
			rei(`$1 delim $2`, 10, 20),
			Cond{`empty`, `delim`, list{10, 20}},
		)

		testExpr(
			t,
			rei(`$1 delim $2 delim $3`, 10, 20, 30),
			Cond{`empty`, `delim`, list{10, 20, 30}},
		)

		testExpr(
			t,
			rei(`(one) delim $1 delim (two)`, 10),
			Cond{`empty`, `delim`, list{Str(`one`), 10, Str(`two`)}},
		)
	})

	t.Run(`struct`, func(t *testing.T) {
		testExpr(t, rei(`empty`), Cond{`empty`, `delim`, Void{}})

		testExpr(
			t,
			rei(`empty`),
			Cond{`empty`, `delim`, struct {
				_ string `db:"col"`
			}{}},
		)

		testExpr(
			t,
			rei(`"col" = (one)`),
			Cond{`empty`, `delim`, struct {
				Col interface{} `db:"col"`
			}{Str(`one`)}},
		)
	})
}

func Test_Cond_filter(t *testing.T) {
	test := func(exp R, val interface{}, fil Haser) {
		t.Helper()
		testExpr(t, exp, Cond{`empty`, `delim`, Partial{val, fil}})
	}

	test(rei(`empty`), PairStruct{10, 20}, nil)
	test(rei(`empty`), &PairStruct{10, 20}, nil)
	test(rei(`"one" = $1 delim "two" = $2`, 10, 20), PairStruct{10, 20}, HaserTrue{})
	test(rei(`"one" = $1 delim "two" = $2`, 10, 20), &PairStruct{10, 20}, HaserTrue{})
}

func Test_Cols(t *testing.T) {
	test := func(exp string, typ interface{}) {
		t.Helper()
		testExpr(t, rei(exp), Cols{typ})
	}

	test(`*`, nil)
	test(`*`, int(0))
	test(`*`, (*int)(nil))
	test(`*`, string(``))
	test(`*`, (*string)(nil))
	test(`*`, time.Time{})
	test(`*`, (*time.Time)(nil))

	test(``, Void{})
	test(``, &Void{})
	test(``, []Void{})
	test(``, []*Void{})
	test(``, &[]Void{})
	test(``, &[]*Void{})

	test(`"one"`, UnitStruct{})
	test(`"one"`, &UnitStruct{})
	test(`"one"`, []UnitStruct{})
	test(`"one"`, []*UnitStruct{})
	test(`"one"`, &[]UnitStruct{})
	test(`"one"`, &[]*UnitStruct{})

	test(`"one", "two"`, PairStruct{})
	test(`"one", "two"`, &PairStruct{})
	test(`"one", "two"`, []PairStruct{})
	test(`"one", "two"`, []*PairStruct{})
	test(`"one", "two"`, &[]PairStruct{})
	test(`"one", "two"`, &[]*PairStruct{})

	test(`"one", "two", "three"`, TrioStruct{})
	test(`"one", "two", "three"`, &TrioStruct{})
	test(`"one", "two", "three"`, []TrioStruct{})
	test(`"one", "two", "three"`, []*TrioStruct{})
	test(`"one", "two", "three"`, &[]TrioStruct{})
	test(`"one", "two", "three"`, &[]*TrioStruct{})

	const outer = `"embed_id", "embed_name", "outer_id", "outer_name"`
	test(outer, Outer{})
	test(outer, &Outer{})
	test(outer, []Outer{})
	test(outer, []*Outer{})
	test(outer, []**Outer{})
	test(outer, &[]Outer{})
	test(outer, &[]*Outer{})
	test(outer, &[]**Outer{})

	const external = `"id", "name", "internal"`
	test(external, External{})
	test(external, &External{})
	test(external, []External{})
	test(external, []*External{})
	test(external, []**External{})
	test(external, &[]External{})
	test(external, &[]*External{})
	test(external, &[]**External{})
}

func Test_ColsDeep(t *testing.T) {
	test := func(exp string, typ interface{}) {
		t.Helper()
		eq(t, exp, TypeColsDeep(typeElemOf(typ)))
		testExpr(t, rei(exp), ColsDeep{typ})
	}

	test(`*`, nil)
	test(`*`, int(0))
	test(`*`, (*int)(nil))
	test(`*`, string(``))
	test(`*`, (*string)(nil))
	test(`*`, time.Time{})
	test(`*`, (*time.Time)(nil))

	test(``, Void{})
	test(``, &Void{})
	test(``, (*Void)(nil))
	test(``, []Void{})
	test(``, []*Void{})
	test(``, &[]Void{})
	test(``, &[]*Void{})

	const outer = `"embed_id", "embed_name", "outer_id", "outer_name"`
	test(outer, Outer{})
	test(outer, &Outer{})
	test(outer, []Outer{})
	test(outer, []*Outer{})
	test(outer, []**Outer{})
	test(outer, &[]Outer{})
	test(outer, &[]*Outer{})
	test(outer, &[]**Outer{})

	const external = `"id", "name", ("internal")."id" as "internal.id", ("internal")."name" as "internal.name"`
	test(external, External{})
	test(external, &External{})
	test(external, []External{})
	test(external, []*External{})
	test(external, []**External{})
	test(external, &[]External{})
	test(external, &[]*External{})
	test(external, &[]**External{})
}

func Test_StructValues(t *testing.T) {
	testExpr(t, rei(``), StructValues{})
	testExpr(t, rei(``), StructValues{Void{}})
	testExpr(t, rei(``), StructValues{&Void{}})
	testExpr(t, rei(``), StructValues{(*UnitStruct)(nil)})
	testExpr(t, rei(``), StructValues{(*PairStruct)(nil)})
	testExpr(t, rei(``), StructValues{(*TrioStruct)(nil)})

	testExpr(t, rei(`$1`, nil), StructValues{UnitStruct{}})
	testExpr(t, rei(`$1`, nil), StructValues{&UnitStruct{}})
	testExpr(t, rei(`$1`, 10), StructValues{UnitStruct{10}})
	testExpr(t, rei(`$1`, 10), StructValues{&UnitStruct{10}})
	testExpr(t, rei(`$1, $2`, 10, 20), StructValues{PairStruct{10, 20}})
	testExpr(t, rei(`$1, $2, $3`, 10, 20, 30), StructValues{TrioStruct{10, 20, 30}})
	testExpr(t, rei(`(one), (two), $1`, 30), StructValues{TrioStruct{Str(`one`), Str(`two`), 30}})

	testExpr(t, rei(`$1, $2, $3, $4`, ``, ``, ``, ``), StructValues{Outer{}})
	testExpr(t, rei(`$1, $2, $3`, ``, ``, Internal{}), StructValues{External{}})

	testExprs(
		t,
		rei(`$1, $2 $3, $4, $5`, 10, 20, 30, 40, 50),
		StructValues{PairStruct{10, 20}},
		StructValues{TrioStruct{30, 40, 50}},
	)
}

func Test_StructValues_filter(t *testing.T) {
	testExpr(t, rei(``), StructValues{Partial{PairStruct{10, 20}, nil}})
	testExpr(t, rei(``), StructValues{Partial{&PairStruct{10, 20}, nil}})
	testExpr(t, rei(`$1, $2`, 10, 20), StructValues{Partial{PairStruct{10, 20}, HaserTrue{}}})
	testExpr(t, rei(`$1, $2`, 10, 20), StructValues{Partial{&PairStruct{10, 20}, HaserTrue{}}})
}

func Test_StructInsert(t *testing.T) {
	testExpr(t, rei(`default values`), StructInsert{})
	testExpr(t, rei(`default values`), StructInsert{Void{}})
	testExpr(t, rei(`default values`), StructInsert{&Void{}})

	testExpr(t, rei(`("one") values ($1)`, nil), StructInsert{UnitStruct{}})
	testExpr(t, rei(`("one") values ($1)`, nil), StructInsert{&UnitStruct{}})
	testExpr(t, rei(`("one") values ($1)`, `two`), StructInsert{UnitStruct{`two`}})
	testExpr(t, rei(`("one") values ($1)`, `two`), StructInsert{&UnitStruct{`two`}})
	testExpr(t, rei(`("one") values ($1)`, 10), StructInsert{UnitStruct{10}})
	testExpr(t, rei(`("one") values ($1)`, 10), StructInsert{&UnitStruct{10}})
	testExpr(t, rei(`("one") values ((two))`), StructInsert{&UnitStruct{Str(`two`)}})

	testExpr(t, rei(`("one", "two") values ($1, $2)`, nil, nil), StructInsert{PairStruct{}})
	testExpr(t, rei(`("one", "two") values ($1, $2)`, nil, nil), StructInsert{&PairStruct{}})
	testExpr(t, rei(`("one", "two") values ($1, $2)`, 10, 20), StructInsert{PairStruct{10, 20}})
	testExpr(t, rei(`("one", "two") values ((one), $1)`, 20), StructInsert{&PairStruct{Str(`one`), 20}})
	testExpr(t, rei(`("one", "two") values ($1, (two))`, 10), StructInsert{&PairStruct{10, Str(`two`)}})

	testExpr(
		t,
		rei(
			`("embed_id", "embed_name", "outer_id", "outer_name") values ($1, $2, $3, $4)`,
			`embed id`, `embed name`, `outer id`, `outer name`,
		),
		StructInsert{Outer{
			Id:   `outer id`,
			Name: `outer name`,
			Embed: Embed{
				Id:   `embed id`,
				Name: `embed name`,
			},
		}},
	)

	testExpr(
		t,
		rei(
			`("id", "name", "internal") values ($1, $2, $3)`,
			`external id`, `external name`, Internal{`internal id`, `internal name`},
		),
		StructInsert{External{
			Id:   `external id`,
			Name: `external name`,
			Internal: Internal{
				Id:   `internal id`,
				Name: `internal name`,
			},
		}},
	)

	testExprs(
		t,
		rei(`default values ("one") values ($1) ("one") values ($2) default values`, 10, 20),
		StructInsert{},
		StructInsert{UnitStruct{10}},
		StructInsert{UnitStruct{20}},
		StructInsert{},
	)
}

func Test_StructInsert_filter(t *testing.T) {
	testExpr(t, rei(`default values`), StructInsert{Partial{PairStruct{10, 20}, nil}})
	testExpr(t, rei(`default values`), StructInsert{Partial{&PairStruct{10, 20}, nil}})
	testExpr(t, rei(`("one", "two") values ($1, $2)`, 10, 20), StructInsert{Partial{PairStruct{10, 20}, HaserTrue{}}})
	testExpr(t, rei(`("one", "two") values ($1, $2)`, 10, 20), StructInsert{Partial{&PairStruct{10, 20}, HaserTrue{}}})
}

func Test_StructAssign(t *testing.T) {
	panics(t, `assignment must have at least one field`, func() {
		StructAssign{}.AppendExpr(nil, nil)
	})

	panics(t, `assignment must have at least one field`, func() {
		StructAssign{Void{}}.AppendExpr(nil, nil)
	})

	panics(t, `assignment must have at least one field`, func() {
		StructAssign{&Void{}}.AppendExpr(nil, nil)
	})

	testExpr(t, rei(`"one" = $1`, nil), StructAssign{UnitStruct{}})
	testExpr(t, rei(`"one" = $1`, 10), StructAssign{UnitStruct{10}})
	testExpr(t, rei(`"one" = (two)`), StructAssign{UnitStruct{Str(`two`)}})

	testExpr(t, rei(`"one" = $1, "two" = $2`, nil, nil), StructAssign{PairStruct{}})
	testExpr(t, rei(`"one" = $1, "two" = $2`, 10, 20), StructAssign{PairStruct{10, 20}})
	testExpr(t, rei(`"one" = (three), "two" = $1`, 20), StructAssign{PairStruct{Str(`three`), 20}})
	testExpr(t, rei(`"one" = $1, "two" = (three)`, 10), StructAssign{PairStruct{10, Str(`three`)}})
	testExpr(t, rei(`"one" = (three), "two" = (four)`), StructAssign{PairStruct{Str(`three`), Str(`four`)}})

	testExprs(
		t,
		rei(`"one" = $1 "one" = $2`, 10, 20),
		StructAssign{UnitStruct{10}},
		StructAssign{UnitStruct{20}},
	)
}

func Test_StructAssign_filter(t *testing.T) {
	testExpr(t, rei(`"one" = $1, "two" = $2`, 10, 20), StructAssign{Partial{PairStruct{10, 20}, HaserTrue{}}})
	testExpr(t, rei(`"one" = $1, "two" = $2`, 10, 20), StructAssign{Partial{&PairStruct{10, 20}, HaserTrue{}}})

	panics(t, `assignment must have at least one field`, func() {
		testExpr(t, rei(`default values`), StructAssign{Partial{PairStruct{10, 20}, nil}})
	})

	panics(t, `assignment must have at least one field`, func() {
		testExpr(t, rei(`default values`), StructAssign{Partial{&PairStruct{10, 20}, nil}})
	})
}

func Test_SelectCols(t *testing.T) {
	testExpr(t, rei(``), SelectCols{})
	testExpr(t, rei(`select "one"`), SelectCols{nil, UnitStruct{}})
	testExpr(t, rei(`table "some_table"`), SelectCols{Table{`some_table`}, nil})

	testExpr(
		t,
		rei(`with _ as (table "some_table") select "one" from _`),
		SelectCols{Table{`some_table`}, UnitStruct{}},
	)

	testExpr(
		t,
		rei(`with _ as (table "some_table") select "embed_id", "embed_name", "outer_id", "outer_name" from _`),
		SelectCols{Table{`some_table`}, Outer{}},
	)

	testExpr(
		t,
		rei(`with _ as (table "some_table") select "id", "name", "internal" from _`),
		SelectCols{Table{`some_table`}, External{}},
	)
}

func Test_SelectColsDeep(t *testing.T) {
	testExpr(t, rei(``), SelectColsDeep{})
	testExpr(t, rei(`select "one"`), SelectColsDeep{nil, UnitStruct{}})
	testExpr(t, rei(`table "some_table"`), SelectColsDeep{Table{`some_table`}, nil})

	testExpr(
		t,
		rei(`with _ as (table "some_table") select "one" from _`),
		SelectColsDeep{Table{`some_table`}, UnitStruct{}},
	)

	testExpr(
		t,
		rei(`with _ as (table "some_table") select "embed_id", "embed_name", "outer_id", "outer_name" from _`),
		SelectColsDeep{Table{`some_table`}, Outer{}},
	)

	testExpr(
		t,
		rei(`with _ as (table "some_table") select "id", "name", ("internal")."id" as "internal.id", ("internal")."name" as "internal.name" from _`),
		SelectColsDeep{Table{`some_table`}, External{}},
	)
}

func Test_Prefix(t *testing.T) {
	testExpr(t, rei(``), Prefix{})
	testExpr(t, rei(``), Prefix{`prefix`, nil})
	testExpr(t, rei(`prefix `), Prefix{`prefix`, Str(``)})
	testExpr(t, rei(`prefix one`), Prefix{`prefix`, Str(`one`)})
	testExpr(t, rei(`one`), Prefix{``, Str(`one`)})
	testExpr(t, rei(`one`, 10), Prefix{``, rei(`one`, 10)})
	testExpr(t, rei(`prefix one`, 10), Prefix{`prefix`, rei(`one`, 10)})
	testExpr(t, rei(`prefix one`, 10), Prefix{`prefix `, rei(`one`, 10)})

	testExprs(
		t,
		rei(`one two three four`, 10, 20, 30, 40),
		Prefix{`one`, rei(`two`, 10, 20)},
		Prefix{`three`, rei(`four`, 30, 40)},
	)
}

func Test_Wrap(t *testing.T) {
	testExpr(t, rei(``), Wrap{})
	testExpr(t, rei(``), Wrap{`prefix`, nil, ``})
	testExpr(t, rei(``), Wrap{``, nil, `suffix`})
	testExpr(t, rei(``), Wrap{`prefix`, nil, `suffix`})
	testExpr(t, rei(`prefix `), Wrap{`prefix`, Str(``), ``})
	testExpr(t, rei(`suffix`), Wrap{``, Str(``), `suffix`})
	testExpr(t, rei(`prefix suffix`), Wrap{`prefix`, Str(``), `suffix`})
	testExpr(t, rei(`one`, 10), Wrap{``, rei(`one`, 10), ``})
	testExpr(t, rei(`prefix one`, 10), Wrap{`prefix`, rei(`one`, 10), ``})
	testExpr(t, rei(`one suffix`, 10), Wrap{``, rei(`one`, 10), `suffix`})
	testExpr(t, rei(`prefix one suffix`, 10), Wrap{`prefix`, rei(`one`, 10), `suffix`})

	testExprs(
		t,
		rei(`one two three four five six`, 10, 20, 30, 40),
		Wrap{`one`, rei(`two`, 10, 20), `three`},
		Wrap{`four`, rei(`five`, 30, 40), `six`},
	)
}

func Test_OrderBy(t *testing.T) {
	testExpr(t, rei(``), OrderBy{})
	testExpr(t, rei(`order by ""`), OrderBy{Ident(``)})
	testExpr(t, rei(`order by "one"`), OrderBy{Ident(`one`)})
	testExpr(t, rei(`order by one`, 10), OrderBy{rei(`one`, 10)})

	testExprs(
		t,
		rei(`one order by two`, 10, 20),
		rei(`one`, 10),
		OrderBy{rei(`two`, 20)},
	)
}

func Test_Select(t *testing.T) {
	test := exprTest(t)

	test(rei(`select * from ""`), Select{})
	test(rei(`select * from "table_name"`), Select{`table_name`, nil})
	test(rei(`select * from "" where $1`, 10), Select{``, 10})

	test(
		rei(`select * from "table_name" where (one) = $1`, 10),
		Select{`table_name`, Eq{Str(`one`), 10}},
	)

	test(
		rei(`select * from "table_name" where "one" = $1 and "two" = $2`, 10, 20),
		Select{`table_name`, PairStruct{10, 20}},
	)

	test(
		rei(`select * from "table_name" where $1 and $2`, 10, 20),
		Select{`table_name`, []interface{}{10, 20}},
	)

	test(
		rei(`select * from "table_name" where $1 and $2`, 10, 20),
		Select{`table_name`, Ands{10, 20}},
	)

	test(
		rei(`select * from "table_name" where $1 or $2`, 10, 20),
		Select{`table_name`, Ors{10, 20}},
	)

	testExprs(
		t,
		rei(`select * from "one" where $1 select * from "two" where $2`, 10, 20),
		Select{`one`, 10},
		Select{`two`, 20},
	)
}

func Test_Insert(t *testing.T) {
	test := exprTest(t)

	test(
		rei(`insert into "" default values returning *`),
		Insert{},
	)

	test(
		rei(`insert into "table_name" default values returning *`),
		Insert{`table_name`, nil},
	)

	test(
		rei(`insert into "" ("one") values ($1) returning *`, 10),
		Insert{``, UnitStruct{10}},
	)

	test(
		rei(`insert into "table_name" ("one") values ($1) returning *`, 10),
		Insert{`table_name`, UnitStruct{10}},
	)

	test(
		rei(`insert into "" ("one", "two") values ($1, $2) returning *`, 10, 20),
		Insert{``, PairStruct{10, 20}},
	)

	test(
		rei(`insert into "table_name" ("one", "two") values ($1, $2) returning *`, 10, 20),
		Insert{`table_name`, PairStruct{10, 20}},
	)

	test(
		rei(`insert into "table_name" default values returning *`),
		Insert{`table_name`, Partial{UnitStruct{10}, nil}},
	)

	testExprs(
		t,
		rei(
			`insert into "table0" ("one") values ($1) returning * `+
				`insert into "table1" ("one") values ($2) returning *`,
			10, 20,
		),
		Insert{`table0`, UnitStruct{10}},
		Insert{`table1`, UnitStruct{20}},
	)
}

func Test_Update(t *testing.T) {
	test := exprTest(t)

	test(rei(`update "" returning *`), Update{})

	test(
		rei(`update "" set "one" = $1 returning *`, 10),
		Update{``, nil, UnitStruct{10}},
	)

	test(
		rei(`update "" where "one" = $1 returning *`, 10),
		Update{``, UnitStruct{10}, nil},
	)

	test(
		rei(`update "" set "one" = $1, "two" = $2 where "one" = $3 returning *`, 20, 30, 10),
		Update{``, UnitStruct{10}, PairStruct{20, 30}},
	)

	test(rei(`update "some_table" returning *`), Update{`some_table`, nil, nil})

	test(
		rei(`update "some_table" set "one" = $1 returning *`, 10),
		Update{`some_table`, nil, UnitStruct{10}},
	)

	test(
		rei(`update "some_table" where "one" = $1 returning *`, 10),
		Update{`some_table`, UnitStruct{10}, nil},
	)

	test(
		rei(`update "some_table" set "one" = $1, "two" = $2 where "one" = $3 returning *`, 20, 30, 10),
		Update{`some_table`, UnitStruct{10}, PairStruct{20, 30}},
	)

	test(
		rei(`update "some_table" set "one" = $1 where true returning *`, 20),
		Update{`some_table`, Partial{UnitStruct{10}, nil}, UnitStruct{20}},
	)

	testExprs(
		t,
		rei(
			`update "table0" where "one" = $1 returning * `+
				`update "table1" set "one" = $2 returning *`,
			10, 20,
		),
		Update{`table0`, UnitStruct{10}, nil},
		Update{`table1`, nil, UnitStruct{20}},
	)
}

func Test_Delete(t *testing.T) {
	test := exprTest(t)

	test(rei(`delete from "" where null returning *`), Delete{})

	test(
		rei(`delete from "some_table" where null returning *`),
		Delete{`some_table`, nil},
	)

	test(
		rei(`delete from "" where "one" = $1 and "two" = $2 returning *`, 10, 20),
		Delete{``, PairStruct{10, 20}},
	)

	test(
		rei(`delete from "some_table" where "one" = $1 and "two" = $2 returning *`, 10, 20),
		Delete{`some_table`, PairStruct{10, 20}},
	)

	test(
		rei(`delete from "some_table" where "one" = $1 or "two" = $2 returning *`, 10, 20),
		Delete{`some_table`, Or{PairStruct{10, 20}}},
	)

	test(
		rei(`delete from "" where null returning *`),
		Delete{``, Partial{PairStruct{10, 20}, nil}},
	)

	test(
		rei(`delete from "some_table" where null returning *`),
		Delete{`some_table`, Partial{PairStruct{10, 20}, nil}},
	)

	testExprs(
		t,
		rei(
			`delete from "table0" where "one" = $1 returning * `+
				`delete from "table1" where "one" = $2 returning *`,
			10, 20,
		),
		Delete{`table0`, UnitStruct{10}},
		Delete{`table1`, UnitStruct{20}},
	)
}

func Test_Call(t *testing.T) {
	testExpr(t, rei(`()`), Call{})
	testExpr(t, rei(`prefix ()`), Call{`prefix`, nil})
	testExpr(t, rei(`()`), Call{``, Str(``)})

	// TODO reconsider. When the input is a single `Expr`, we may want to always
	// additionally parenthesize it.
	testExpr(t, rei(`(one)`), Call{``, Str(`one`)})

	testExpr(t, rei(`()`), Call{``, list{}})
	testExpr(t, rei(`($1)`, nil), Call{``, list{nil}})
	testExpr(t, rei(`($1)`, 10), Call{``, list{10}})
	testExpr(t, rei(`($1, $2)`, 10, 20), Call{``, list{10, 20}})
	testExpr(t, rei(`((one), $1)`, 20), Call{``, list{Str(`one`), 20}})
	testExpr(t, rei(`($1, (two))`, 10), Call{``, list{10, Str(`two`)}})
	testExpr(t, rei(`((one), (two))`), Call{``, list{Str(`one`), Str(`two`)}})

	testExpr(t, rei(`prefix ()`), Call{`prefix`, list{}})
	testExpr(t, rei(`prefix ($1)`, nil), Call{`prefix`, list{nil}})
	testExpr(t, rei(`prefix ($1)`, 10), Call{`prefix`, list{10}})
	testExpr(t, rei(`prefix ($1, $2)`, 10, 20), Call{`prefix`, list{10, 20}})
	testExpr(t, rei(`prefix ((one), $1)`, 20), Call{`prefix`, list{Str(`one`), 20}})
	testExpr(t, rei(`prefix ($1, (two))`, 10), Call{`prefix`, list{10, Str(`two`)}})
	testExpr(t, rei(`prefix ((one), (two))`), Call{`prefix`, list{Str(`one`), Str(`two`)}})

	testExprs(
		t,
		rei(`() () ($1) prefix ($2)`, 10, 20),
		Call{},
		Call{},
		Call{``, list{10}},
		Call{`prefix`, list{20}},
	)
}

func Test_RowNumberOver(t *testing.T) {
	testExpr(t, rei(`0`), RowNumberOver{})
	testExpr(t, rei(`row_number() over ()`), RowNumberOver{Str(``)})
	testExpr(t, rei(`row_number() over (one)`), RowNumberOver{Str(`one`)})
	testExpr(t, rei(`row_number() over (one)`, 10), RowNumberOver{rei(`one`, 10)})

	testExprs(
		t,
		rei(`0 row_number() over (one) row_number() over (two)`, 10, 20),
		RowNumberOver{},
		RowNumberOver{rei(`one`, 10)},
		RowNumberOver{rei(`two`, 20)},
	)
}

func Test_StrQ_without_args(t *testing.T) {
	testExpr(t, rei(``), StrQ{})
	testExpr(t, rei(`one`), StrQ{`one`, nil})
	testExpr(t, rei(`one`), StrQ{`one`, Dict(nil)})
	testExpr(t, rei(`one two`), StrQ{`one two`, nil})
	testExprs(t, rei(`one two `), StrQ{}, StrQ{`one`, nil}, StrQ{}, StrQ{`two`, nil}, StrQ{})

	panics(t, `expected arguments, got none`, func() {
		StrQ{`$1`, nil}.AppendExpr(nil, nil)
	})

	panics(t, `expected arguments, got none`, func() {
		StrQ{`:one`, nil}.AppendExpr(nil, nil)
	})
}

func Test_ListQ_invalid(t *testing.T) {
	panics(t, `non-parametrized expression "" expected no arguments`, func() {
		ListQ(``, nil).AppendExpr(nil, nil)
	})

	panics(t, `expected arguments, got none`, func() {
		ListQ(`$1`).AppendExpr(nil, nil)
	})

	panics(t, `expected arguments, got none`, func() {
		ListQ(`:one`).AppendExpr(nil, nil)
	})

	panics(t, `missing ordinal argument "$2" (index 1)`, func() {
		ListQ(`$2`, 10).AppendExpr(nil, nil)
	})

	panics(t, `missing named argument ":one" (key "one")`, func() {
		ListQ(`:one`, 10).AppendExpr(nil, nil)
	})

	panics(t, `unused ordinal argument "$2" (index 1)`, func() {
		ListQ(`$1`, 10, 20).AppendExpr(nil, nil)
	})
}

func Test_ListQ_empty_args(t *testing.T) {
	testExpr(t, rei(``), ListQ(``))
	testExpr(t, rei(`one`), ListQ(`one`))
}

func Test_ListQ_normal(t *testing.T) {
	test := exprTest(t)

	test(rei(`one = $1`, nil), ListQ(`one = $1`, nil))
	test(rei(`one = $1`, 10), ListQ(`one = $1`, 10))
	test(rei(`one = $1`, 10), ListQ(`one = $1 `, 10))

	test(rei(`one = $1, two = $1`, 10), ListQ(`one = $1, two = $1`, 10))
	test(rei(`one = $1, two = $2`, 10, 20), ListQ(`one = $1, two = $2`, 10, 20))

	test(
		rei(`one = $1, two = $2, three = $1, four = $2`, 10, 20),
		ListQ(`one = $1, two = $2, three = $1, four = $2`, 10, 20),
	)

	test(rei(`one = one`), ListQ(`one = $1`, Str(`one`)))

	test(
		rei(`one = one, two = one`),
		ListQ(`one = $1, two = $1`, Str(`one`)),
	)

	test(
		rei(`one = one`, 10),
		ListQ(`one = $1`, rei(`one`, 10)),
	)

	test(
		rei(`one = one, two = $1, three = three`, 20),
		ListQ(
			`one = $1, two = $2, three = $3`,
			Str(`one`),
			20,
			Str(`three`),
		),
	)
}

func Test_DictQ_invalid(t *testing.T) {
	panics(t, `non-parametrized expression "" expected no arguments`, func() {
		DictQ(``, Dict{`one`: 10}).AppendExpr(nil, nil)
	})

	panics(t, `expected arguments, got none`, func() {
		DictQ(`$1`, nil).AppendExpr(nil, nil)
	})

	panics(t, `expected arguments, got none`, func() {
		DictQ(`:one`, nil).AppendExpr(nil, nil)
	})

	panics(t, `expected arguments, got none`, func() {
		DictQ(`$1`, Dict{}).AppendExpr(nil, nil)
	})

	panics(t, `expected arguments, got none`, func() {
		DictQ(`:one`, Dict{}).AppendExpr(nil, nil)
	})

	panics(t, `missing ordinal argument "$1" (index 0)`, func() {
		DictQ(`$1`, Dict{`one`: 10}).AppendExpr(nil, nil)
	})

	panics(t, `missing ordinal argument "$1" (index 0)`, func() {
		DictQ(`$1 :one`, Dict{`one`: 10}).AppendExpr(nil, nil)
	})

	panics(t, `missing ordinal argument "$1" (index 0)`, func() {
		DictQ(`:one $1`, Dict{`one`: 10}).AppendExpr(nil, nil)
	})

	panics(t, `missing named argument ":one" (key "one")`, func() {
		DictQ(`:one`, Dict{`two`: 20}).AppendExpr(nil, nil)
	})

	panics(t, `missing named argument ":one" (key "one")`, func() {
		DictQ(`one:one`, Dict{`two`: 20}).AppendExpr(nil, nil)
	})

	panics(t, `unused named argument ":two" (key "two")`, func() {
		DictQ(`:one`, Dict{`one`: 10, `two`: 20}).AppendExpr(nil, nil)
	})
}

func Test_DictQ_empty_args(t *testing.T) {
	testExpr(t, rei(``), DictQ(``, nil))
	testExpr(t, rei(`one`), DictQ(`one`, nil))
	testExpr(t, rei(`one two`), DictQ(`one two`, nil))
	testExpr(t, rei(``), DictQ(``, Dict(nil)))
	testExpr(t, rei(``), DictQ(``, Dict{}))
}

func Test_DictQ_normal(t *testing.T) {
	test := exprTest(t)

	test(rei(`one = $1`, nil), DictQ(`one = :one`, Dict{`one`: nil}))
	test(rei(`one = $1`, 10), DictQ(`one = :one`, Dict{`one`: 10}))

	// There was a parser bug that broke this.
	test(rei(`one = $1`, 10), DictQ(`one = :one `, Dict{`one`: 10}))

	test(rei(`one = $1, two = $1`, 10), DictQ(`one = :one, two = :one`, Dict{`one`: 10}))
	test(rei(`one = $1, two = $2`, 10, 20), DictQ(`one = :one, two = :two`, Dict{`one`: 10, `two`: 20}))

	test(
		rei(`one = $1, two = $2, three = $1, four = $2`, 10, 20),
		DictQ(`one = :one, two = :two, three = :one, four = :two`, Dict{`one`: 10, `two`: 20}),
	)

	test(
		rei(`one = one`),
		DictQ(`one = :one`, Dict{`one`: Str(`one`)}),
	)

	test(
		rei(`one = one, two = one`),
		DictQ(`one = :one, two = :one`, Dict{`one`: Str(`one`)}),
	)

	test(
		rei(`one = one`, 10),
		DictQ(`one = :one`, Dict{`one`: rei(`one`, 10)}),
	)

	test(
		rei(`one = one, two = $1, three = three`, 20),
		DictQ(`one = :one, two = :two, three = :three`, Dict{
			`one`:   Str(`one`),
			`two`:   20,
			`three`: Str(`three`),
		}),
	)
}

func Test_StructQ_invalid(t *testing.T) {
	panics(t, `non-parametrized expression "" expected no arguments`, func() {
		StructQ(``, Void{}).AppendExpr(nil, nil)
	})

	panics(t, `expected arguments, got none`, func() {
		StructQ(`$1`, nil).AppendExpr(nil, nil)
	})

	panics(t, `expected arguments, got none`, func() {
		StructQ(`:one`, nil).AppendExpr(nil, nil)
	})

	panics(t, `missing ordinal argument "$1" (index 0)`, func() {
		StructQ(`$1`, Void{}).AppendExpr(nil, nil)
	})

	panics(t, `missing named argument ":one" (key "one")`, func() {
		StructQ(`:one`, Void{}).AppendExpr(nil, nil)
	})
}

func Test_StructQ_empty_args(t *testing.T) {
	testExpr(t, rei(``), StructQ(``, nil))
	testExpr(t, rei(``), StructQ(``, nil))
	testExpr(t, rei(`one`), StructQ(`one`, nil))
	testExpr(t, rei(`one two`), StructQ(`one two`, nil))
	testExpr(t, rei(``), StructQ(``, (*Void)(nil)))
	testExpr(t, rei(`one`), StructQ(`one`, (*Void)(nil)))
}

func Test_StructQ_fields(t *testing.T) {
	test := exprTest(t)

	panics(t, `missing named argument ":one" (key "one")`, func() {
		StructQ(`:one`, UnitStruct{10}).AppendExpr(nil, nil)
	})

	test(rei(`one = $1`, nil), StructQ(`one = :One`, UnitStruct{}))
	test(rei(`one = $1`, 10), StructQ(`one = :One`, UnitStruct{10}))

	panics(t, `missing named argument ":one" (key "one")`, func() {
		StructQ(`:one`, PairStruct{10, 20}).AppendExpr(nil, nil)
	})

	panics(t, `missing named argument ":two" (key "two")`, func() {
		StructQ(`:two`, PairStruct{10, 20}).AppendExpr(nil, nil)
	})

	test(rei(`one = $1, two = $1`, 10), StructQ(`one = :One, two = :One`, PairStruct{10, 20}))
	test(rei(`one = $1, two = $2`, 10, 20), StructQ(`one = :One, two = :Two`, PairStruct{10, 20}))

	test(
		rei(`one = $1, two = $2, three = $1, four = $2`, 10, 20),
		StructQ(`one = :One, two = :Two, three = :One, four = :Two`, PairStruct{10, 20}),
	)

	test(
		rei(`one = one, two = $1, three = three`, 20),
		StructQ(`one = :One, two = :Two, three = :Three`, TrioStruct{
			Str(`one`),
			20,
			Str(`three`),
		}),
	)
}

func Test_StructQ_methods(t *testing.T) {
	test := exprTest(t)

	panics(t, `missing named argument ":GetVal" (key "GetVal")`, func() {
		StructQ(`:GetVal`, UnitStruct{}).AppendExpr(nil, nil)
	})

	test(rei(`$1`, `val`), StructQ(`:GetVal`, Void{}))

	panics(t, `expected 0 parameters, found 1 parameters`, func() {
		StructQ(`:UnaryVoid`, UnitStruct{}).AppendExpr(nil, nil)
	})

	panics(t, `expected 1 return parameter, found 2 return parameters`, func() {
		StructQ(`:NullaryPair`, UnitStruct{}).AppendExpr(nil, nil)
	})

	test(rei(`one = $1`, nil), StructQ(`one = :GetOne`, UnitStruct{}))
	test(rei(`one = $1`, 10), StructQ(`one = :GetOne`, UnitStruct{10}))

	panics(t, `missing named argument ":one" (key "one")`, func() {
		StructQ(`:one`, PairStruct{10, 20}).AppendExpr(nil, nil)
	})

	panics(t, `missing named argument ":two" (key "two")`, func() {
		StructQ(`:two`, PairStruct{10, 20}).AppendExpr(nil, nil)
	})

	test(rei(`one = $1, two = $1`, 10), StructQ(`one = :GetOne, two = :GetOne`, PairStruct{10, 20}))
	test(rei(`one = $1, two = $2`, 10, 20), StructQ(`one = :GetOne, two = :GetTwo`, PairStruct{10, 20}))

	test(
		rei(`one = $1, two = $2, three = $1, four = $2`, 10, 20),
		StructQ(`one = :GetOne, two = :GetTwo, three = :GetOne, four = :GetTwo`, PairStruct{10, 20}),
	)

	test(
		rei(`one = one, two = $1, three = three`, 20),
		StructQ(`one = :GetOne, two = :GetTwo, three = :GetThree`, TrioStruct{
			Str(`one`),
			20,
			Str(`three`),
		}),
	)
}

func Test_Prep_Parse(t *testing.T) {
	testPrepParse(t, func(src string, tokens []Token, hasParams bool) {
		t.Helper()
		prep := Prep{Source: src}
		prep.Parse()
		eq(t, Prep{src, tokens, hasParams}, prep)
	})
}

func Test_Preparse(t *testing.T) {
	testPrepParse(t, func(src string, tokens []Token, hasParams bool) {
		t.Helper()
		eq(t, Prep{src, tokens, hasParams}, Preparse(src))
	})
}

func testPrepParse(t testing.TB, test func(string, []Token, bool)) {
	test(``, nil, false)
	test(`one`, []Token{Token{`one`, TokenTypeText}}, false)
	test(`$1`, []Token{Token{`$1`, TokenTypeOrdinalParam}}, true)
	test(`:one`, []Token{Token{`:one`, TokenTypeNamedParam}}, true)

	test(
		`one $1 two :three four $2 five :six`,
		[]Token{
			Token{`one `, TokenTypeText},
			Token{`$1`, TokenTypeOrdinalParam},
			Token{` two `, TokenTypeText},
			Token{`:three`, TokenTypeNamedParam},
			Token{` four `, TokenTypeText},
			Token{`$2`, TokenTypeOrdinalParam},
			Token{` five `, TokenTypeText},
			Token{`:six`, TokenTypeNamedParam},
		},
		true,
	)

	test(
		/*pgsql*/ `
one
-- line comment $1 :one
:two
/* block comment $1 :one */
three
`,
		[]Token{
			Token{`one `, TokenTypeText},
			Token{`:two`, TokenTypeNamedParam},
			Token{`  three`, TokenTypeText},
		},
		true,
	)
}

func Test_Preparse_dedup(t *testing.T) {
	test := func(val string) {
		t.Helper()

		eq(t, Preparse(val), Preparse(val))
		is(t, prepCache.Get(val), prepCache.Get(val))

		prep := Preparse(val)

		eq(t, prep, Preparse(val))
		eq(t, prep, Preparse(val))
	}

	test(``)
	test(` `)
	test(`one`)
	test(`one two`)
	test(`one :two`)
	test(`one :two three`)

	notEq(t, Preparse(``), Preparse(` `))
	notEq(t, Preparse(``), Preparse(`one`))
	notEq(t, Preparse(` `), Preparse(`one`))
}

/*
Note: parametrized expression building is verified in various tests for `StrQ`,
which uses a `Prep` internally. This is mostly for verifying the automatic
"unparam" mode and associated assertions.
*/
func Test_Prep_AppendParamExpr_unparam(t *testing.T) {
	test := func(exp R, vals ...string) {
		t.Helper()
		eq(t, exp, reiFrom(reifyUnparamPreps(vals...)))
	}

	test(rei(``))
	test(rei(``), ``)
	test(rei(``), ``, ``)
	test(rei(`one`), `one`)
	test(rei(`one`), ``, `one`, ``)
	test(rei(`one two`), ``, `one`, ``, `two`, ``)

	testNilArgs := func(val string) {
		t.Helper()
		eq(t, rei(val), reifyParamExpr(Preparse(val), nil))
		eq(t, rei(val), reifyParamExpr(Preparse(val), Dict(nil)))
		eq(t, rei(val), reifyParamExpr(Preparse(val), (*Dict)(nil)))
		eq(t, rei(val), reifyParamExpr(Preparse(val), (*StructDict)(nil)))
	}

	testNilArgs(``)
	testNilArgs(`one`)
	testNilArgs(`one two`)

	testPanic := func(val string) {
		t.Helper()

		prep := Preparse(val)
		msg := fmt.Sprintf(`non-parametrized expression %q expected no arguments, got`, val)

		panics(t, msg, func() {
			prep.AppendParamExpr(nil, nil, Dict{})
		})

		panics(t, msg, func() {
			prep.AppendParamExpr(nil, nil, StructDict{})
		})
	}

	testPanic(``)
	testPanic(`one`)
	testPanic(`one two`)
}

func Test_combinations(t *testing.T) {
	testExpr(
		t,
		rei(`($1 or $2) = ($3 or $4)`, 10, 20, 30, 40),
		Eq{Ors{10, 20}, Ors{30, 40}},
	)

	testExpr(
		t,
		rei(`($1 or $2) and ($3 or $4) and $5`, 10, 20, 30, 40, 50),
		Ands{Ors{10, 20}, Ors{30, 40}, 50},
	)

	testExpr(
		t,
		rei(`select "some_column" from "some_table"`),
		StrQ{`select :col from :tab`, Dict{
			`col`: Ident(`some_column`),
			`tab`: Ident(`some_table`),
		}},
	)
}

func Test_column_fields(t *testing.T) {
	eq(
		t,
		[][2]string{
			{`embed_id`, `embed id`},
			{`embed_name`, `embed name`},
			{`outer_id`, `outer id`},
			{`outer_name`, `outer name`},
		},
		tCols(),
	)
}

func tCols() (out [][2]string) {
	val := r.ValueOf(testOuter)
	for _, field := range loadStructDbFields(val.Type()) {
		out = append(out, [2]string{
			FieldDbName(field),
			val.FieldByIndex(field.Index).String(),
		})
	}
	return
}

func Test_List(t *testing.T) {
	zero := List(nil)
	empty := List{}
	full := List{10, 20, 30, 40, 50, 60, 70, 80}

	eq(t, true, zero.IsEmpty())
	eq(t, true, empty.IsEmpty())
	eq(t, false, full.IsEmpty())

	eq(t, 0, zero.Len())
	eq(t, 0, empty.Len())
	eq(t, 8, full.Len())

	testNamedEmpty := func(dict ArgDict) {
		t.Helper()

		test := func(key string) {
			t.Helper()
			val, ok := dict.GotNamed(key)
			eq(t, nil, val)
			eq(t, false, ok)
		}

		test(`-1`)
		test(`0`)
		test(`1`)
		test(`2`)
		test(`$-1`)
		test(`-$1`)
		test(`$0`)
		test(`$1`)
		test(`$2`)
	}

	testNamedEmpty(zero)
	testNamedEmpty(empty)
	testNamedEmpty(full)

	testOrdEmpty := func(dict ArgDict) {
		t.Helper()
		for key := range counter(64) {
			val, ok := dict.GotOrdinal(key)
			eq(t, nil, val)
			eq(t, false, ok)
		}
	}

	testOrdEmpty(zero)
	testOrdEmpty(empty)

	testOrdFull := func(key int, expVal interface{}, expOk bool) {
		t.Helper()
		val, ok := full.GotOrdinal(key)
		eq(t, expVal, val)
		eq(t, expOk, ok)
	}

	testOrdFull(-1, nil, false)
	testOrdFull(0, 10, true)
	testOrdFull(1, 20, true)
	testOrdFull(2, 30, true)
	testOrdFull(3, 40, true)
	testOrdFull(4, 50, true)
	testOrdFull(5, 60, true)
	testOrdFull(6, 70, true)
	testOrdFull(7, 80, true)
	testOrdFull(8, nil, false)
	testOrdFull(9, nil, false)
	testOrdFull(10, nil, false)
	testOrdFull(11, nil, false)
	testOrdFull(12, nil, false)
	testOrdFull(13, nil, false)
	testOrdFull(14, nil, false)
	testOrdFull(15, nil, false)
	testOrdFull(16, nil, false)
}

func Test_Dict(t *testing.T) {
	zero := Dict(nil)
	empty := Dict{}
	full := benchDict

	eq(t, 0, zero.Len())
	eq(t, 0, empty.Len())
	eq(t, 24, full.Len())

	testArgDictNamed(t, zero, empty, full)
}

func Test_StructDict(t *testing.T) {
	zero := StructDict{}
	empty := StructDict{r.ValueOf(Void{})}
	full := benchStructDict

	eq(t, 0, zero.Len())
	eq(t, 0, empty.Len())
	eq(t, 0, full.Len())

	testArgDictNamed(t, zero, empty, full)
}

func testArgDictNamed(t testing.TB, zero, empty, full ArgDict) {
	eq(t, true, zero.IsEmpty())
	eq(t, true, empty.IsEmpty())
	eq(t, false, full.IsEmpty())

	testOrd := func(val ArgDict) {
		t.Helper()
		for key := range counter(64) {
			val, ok := zero.GotOrdinal(key)
			eq(t, nil, val)
			eq(t, false, ok)
		}
	}

	testOrd(zero)
	testOrd(empty)
	testOrd(full)

	testKeyVal := func(key, exp string) {
		t.Helper()

		val, ok := zero.GotNamed(key)
		eq(t, nil, val)
		eq(t, false, ok)

		val, ok = empty.GotNamed(key)
		eq(t, nil, val)
		eq(t, false, ok)

		val, ok = full.GotNamed(key)
		notEq(t, nil, val)
		eq(t, true, ok)
		eq(t, exp, val.(string))
	}

	testKeyVal(`Key_c603c58746a69833a1528050c33d`, `val_e1436c61440383a80ebdc245b930`)
	testKeyVal(`Key_abfbb9e94e4093a47683e8ef606b`, `val_a6108ccd40789cecf4da1052c5ae`)
	testKeyVal(`Key_907b548d45948a206907ed9c9097`, `val_9271789147789ecb2beb11c97a78`)
	testKeyVal(`Key_5ee2513a41a88d173cd53d389c14`, `val_2b6205fb4bf882ab65f3795b2384`)
	testKeyVal(`Key_0ac8b89b46bba5d4d076e71d6232`, `val_226b2c3a4c5591084d3a120de2d8`)
	testKeyVal(`Key_b754f88b42fcbd6c30e3bb544909`, `val_9c639ea74d099446ec3aa2a736a8`)
	testKeyVal(`Key_e52daa684071891a1dae084bfd00`, `val_71fc2d114b2aaa3b5c1c399d28f6`)
	testKeyVal(`Key_3106dc324be4b3ff5d477e71c593`, `val_9183d36e4b53a5e2b26ca950a721`)
	testKeyVal(`Key_a7b558a54d178bdb6fcf3368939b`, `val_f0bc980a408c81a959168aa8aabc`)
	testKeyVal(`Key_1622da954c8a8f6fec82e6fd3c34`, `val_4afe6fa84722a214e4e777aa6bcf`)
	testKeyVal(`Key_fa3892644f1392aee8e66b799b3f`, `val_c45ce5ec46b7809d5df5cd1c815b`)
	testKeyVal(`Key_b9aa15254438b0b7a32489947c50`, `val_6b119aad4bc280a3dfa675fe88a5`)

	testKeyVal(`Key_ce59b8e14f77b6e6e9cd28cecacd`, `val_c76bd35c42d49ccb4408f92fb222`)
	testKeyVal(`Key_87819a034834a3530b8255e76e4d`, `val_a185f0a946e894d1628bb98b673e`)
	testKeyVal(`Key_c31042674737a95d1cba33b61687`, `val_02bae4964cfa9ebd23b5d3f57ee6`)
	testKeyVal(`Key_7bc7a0d346c2b87e3110b2d192d3`, `val_2208de3a476299877d36f149ab94`)
	testKeyVal(`Key_3b17f4454d44abbbeb2eb5b61235`, `val_dfb68e4d459aa5c649dcb07e0bfb`)
	testKeyVal(`Key_83e52b714a9d8a0ba6dd87658acf`, `val_2ec2ca5046038e80cfa3cb23dff2`)
	testKeyVal(`Key_82c96b4d4965a08fa6735e973caa`, `val_fae699f449a1aaf138b1ae2bb9b0`)
	testKeyVal(`Key_7580ec1f4d42a7aafddf4f818b97`, `val_fc6b97924798b1b790cfb6e31750`)
	testKeyVal(`Key_bc03a581465c873ceea04027d6ab`, `val_ab22ce72453cb2577aa731dae72c`)
	testKeyVal(`Key_dcfa83ed4be89cf05d5e3eba6f2a`, `val_b773e8ce401c8313b1400b973fa1`)
	testKeyVal(`Key_2bc5f64447879c1152ae9b904718`, `val_e9d6438d42339e4c62db260c458b`)
	testKeyVal(`Key_4f0e9d9b4d1ea77c510337ae6c2a`, `val_60a4b1bf406f98826c706ab153d1`)
}

func Test_Partial(t *testing.T) {
	t.Run(`Get`, func(t *testing.T) {
		test := func(val interface{}) {
			eq(t, val, Partial{val, nil}.Get())
		}

		test(nil)
		test(10)
		test(`str`)
		test((*Void)(nil))
		test(Void{})
		test(&Void{})
		test(Outer{})
		test(&Outer{})
		test(Partial{})
		test(&Partial{})
	})

	t.Run(`HasField`, func(t *testing.T) {
		test := func(exp bool, val interface{}, fil Haser, tag r.StructTag) {
			t.Helper()
			eq(t, exp, Partial{val, fil}.HasField(r.StructField{Tag: tag}))
		}

		test(false, nil, nil, ``)
		test(false, UnitStruct{}, nil, ``)
		test(false, nil, HaserFalse{}, ``)
		test(false, UnitStruct{}, HaserFalse{}, ``)
		test(false, nil, HaserTrue{}, ``)
		test(false, UnitStruct{}, HaserTrue{}, ``)

		test(false, nil, nil, `json:""`)
		test(false, UnitStruct{}, nil, `json:""`)
		test(false, nil, HaserFalse{}, `json:""`)
		test(false, UnitStruct{}, HaserFalse{}, `json:""`)
		test(false, nil, HaserTrue{}, `json:""`)
		test(false, UnitStruct{}, HaserTrue{}, `json:""`)

		test(false, nil, nil, `json:"-"`)
		test(false, UnitStruct{}, nil, `json:"-"`)
		test(false, nil, HaserFalse{}, `json:"-"`)
		test(false, UnitStruct{}, HaserFalse{}, `json:"-"`)
		test(false, nil, HaserTrue{}, `json:"-"`)
		test(false, UnitStruct{}, HaserTrue{}, `json:"-"`)

		test(false, nil, nil, `db:"someName"`)
		test(false, UnitStruct{}, nil, `db:"someName"`)
		test(false, nil, HaserFalse{}, `db:"someName"`)
		test(false, UnitStruct{}, HaserFalse{}, `db:"someName"`)
		test(false, nil, HaserTrue{}, `db:"someName"`)
		test(false, UnitStruct{}, HaserTrue{}, `db:"someName"`)

		test(false, nil, nil, `json:"someName"`)
		test(false, UnitStruct{}, nil, `json:"someName"`)
		test(false, nil, HaserFalse{}, `json:"someName"`)
		test(false, UnitStruct{}, HaserFalse{}, `json:"someName"`)
		test(true, nil, HaserTrue{}, `json:"someName"`)
		test(true, UnitStruct{}, HaserTrue{}, `json:"someName"`)
	})
}

// Incomplete, TODO more.
func TestTryString(t *testing.T) {
	test := func(exp string, src interface{}) {
		t.Helper()
		eq(t, exp, TryString(src))
	}

	test(``, nil)
	test(``, ``)
	test(``, (*string)(nil))
	test(``, (**string)(nil))
	test(``, (*int)(nil))
	test(``, (**int)(nil))
	test(`one`, `one`)
	test(`one`, []byte(`one`))
	test(`0`, 0)
	test(`10`, 10)
	test(`123.456`, 123.456)
	test(`true`, true)
	test(`false`, false)
	test(`0001-02-03 04:05:06.000000007 +0000 UTC`, time.Date(1, 2, 3, 4, 5, 6, 7, time.UTC))
	test(`""`, Ident(``))
	test(`"one"`, Ident(`one`))

	panics(t, `unsupported type "[]int" of kind "slice"`, func() {
		TryString([]int(nil))
	})

	panics(t, `unsupported type "struct {}" of kind "struct"`, func() {
		TryString(struct{}{})
	})

	panics(t, `unsupported type "map[int]int" of kind "map"`, func() {
		TryString((map[int]int)(nil))
	})

	panics(t, `unsupported type "func(interface {}) string" of kind "func"`, func() {
		TryString(TryString)
	})

	panics(t, `unsupported type "chan int" of kind "chan"`, func() {
		TryString((chan int)(nil))
	})
}

// Incomplete, TODO more.
func TestTryAppend(t *testing.T) {
	test := func(exp string, src interface{}) {
		t.Helper()
		const prefix = `this prefix must be preserved `
		eq(t, prefix+exp, string(TryAppend([]byte(prefix), src)))
	}

	test(``, nil)
	test(``, ``)
	test(``, (*string)(nil))
	test(``, (**string)(nil))
	test(``, (*int)(nil))
	test(``, (**int)(nil))
	test(`one`, `one`)
	test(`one`, []byte(`one`))
	test(`0`, 0)
	test(`10`, 10)
	test(`123.456`, 123.456)
	test(`true`, true)
	test(`false`, false)
	test(`0001-02-03T04:05:06.000000007Z`, time.Date(1, 2, 3, 4, 5, 6, 7, time.UTC))
	test(`""`, Ident(``))
	test(`"one"`, Ident(`one`))

	panics(t, `unsupported type "[]int" of kind "slice"`, func() {
		TryAppend(nil, []int(nil))
	})

	panics(t, `unsupported type "struct {}" of kind "struct"`, func() {
		TryAppend(nil, struct{}{})
	})

	panics(t, `unsupported type "map[int]int" of kind "map"`, func() {
		TryAppend(nil, (map[int]int)(nil))
	})

	panics(t, `unsupported type "func([]uint8, interface {}) []uint8" of kind "func"`, func() {
		TryAppend(nil, TryAppend)
	})

	panics(t, `unsupported type "chan int" of kind "chan"`, func() {
		TryAppend(nil, (chan int)(nil))
	})
}

func TestCommaAppender(t *testing.T) {
	test := func(exp string, val Encoder) { testEncoder(t, exp, val) }

	test(``, CommaAppender{})
	test(``, CommaAppender{nil, nil, nil})
	test(``, CommaAppender{nil, CommaAppender{}, CommaAppender{nil}, nil, Stringer{}})
	test(``, CommaAppender{Stringer{nil}})
	test(``, CommaAppender{nil, Stringer{}, nil, Stringer{``}})
	test(`10`, CommaAppender{Stringer{10}})
	test(`10`, CommaAppender{nil, Stringer{10}, nil})
	test(`10`, CommaAppender{nil, Stringer{``}, Stringer{10}, nil, Stringer{``}})
	test(`10,20`, CommaAppender{Stringer{10}, nil, Stringer{20}})
	test(`10,20`, CommaAppender{nil, Stringer{10}, nil, Stringer{20}, nil, Stringer{``}})
	test(`10,20,30`, CommaAppender{CommaAppender{Stringer{10}}, CommaAppender{Stringer{20}, Stringer{``}, Stringer{30}}})
}

func TestArrayAppender(t *testing.T) {
	test := func(exp string, val Encoder) { testEncoder(t, exp, val) }

	test(`{}`, ArrayAppender{})
	test(`{}`, ArrayAppender{nil, nil, nil})
	test(`{}`, ArrayAppender{nil, CommaAppender{}, CommaAppender{nil}, nil, Stringer{}})
	test(`{{},{}}`, ArrayAppender{nil, ArrayAppender{}, ArrayAppender{nil}, nil, Stringer{}})
	test(`{}`, ArrayAppender{Stringer{nil}})
	test(`{}`, ArrayAppender{nil, Stringer{}, nil, Stringer{``}})
	test(`{10}`, ArrayAppender{Stringer{10}})
	test(`{10}`, ArrayAppender{nil, Stringer{10}, nil})
	test(`{10}`, ArrayAppender{nil, Stringer{``}, Stringer{10}, nil, Stringer{``}})
	test(`{10,20}`, ArrayAppender{Stringer{10}, nil, Stringer{20}})
	test(`{10,20}`, ArrayAppender{nil, Stringer{10}, nil, Stringer{20}, nil, Stringer{``}})
	test(`{10,20,30}`, ArrayAppender{CommaAppender{Stringer{10}}, CommaAppender{Stringer{20}, Stringer{``}, Stringer{30}}})
	test(`{{10},{20,30}}`, ArrayAppender{ArrayAppender{Stringer{10}}, ArrayAppender{Stringer{20}, Stringer{``}, Stringer{30}}})
}