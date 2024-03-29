package sqlb

import (
	"encoding/json"
	r "reflect"
	"testing"
)

func Test_Ords_Expr(t *testing.T) {
	testExpr(t, rei(``), Ords(nil))
	testExpr(t, rei(``), Ords{})
	testExpr(t, rei(``), Ords{nil, nil, nil})
	testExpr(t, rei(`order by one`), Ords{nil, Str(`one`), nil})
	testExpr(t, rei(`order by $1, $2`, 10, 20), Ords{rei(`$1`, 10), nil, rei(`$2`, 20)})

	testExprs(
		t,
		rei(`one order by two`, 10, 20),
		rei(`one`, 10),
		Ords{rei(`two`, 20)},
	)
}

func Test_Ords_Len(t *testing.T) {
	eq(t, 0, Ords(nil).Len())
	eq(t, 0, Ords{}.Len())
	eq(t, 0, Ords{nil}.Len())
	eq(t, 1, Ords{nil, Str(``), nil}.Len())
	eq(t, 2, Ords{nil, Str(``), nil, Str(``), nil}.Len())
}

func Test_Ords_IsEmpty(t *testing.T) {
	eq(t, true, Ords(nil).IsEmpty())
	eq(t, true, Ords{}.IsEmpty())
	eq(t, true, Ords{nil}.IsEmpty())
	eq(t, false, Ords{nil, Str(``), nil}.IsEmpty())
	eq(t, false, Ords{nil, Str(``), nil, Str(``), nil}.IsEmpty())
}

func Test_Ords_RowNumberOver(t *testing.T) {
	eq(t, RowNumberOver{}, Ords(nil).RowNumberOver())
	eq(t, RowNumberOver{}, Ords{}.RowNumberOver())
	eq(t, RowNumberOver{}, Ords{nil}.RowNumberOver())
	eq(t, RowNumberOver{Ords{nil, Str(``)}}, Ords{nil, Str(``)}.RowNumberOver())
}

func Test_Ords_Or(t *testing.T) {
	test := func(exp, tar, args Ords) {
		t.Helper()
		tar.Or(args...)
		eq(t, exp, tar)
	}

	test(Ords(nil), Ords(nil), Ords(nil))
	test(Ords{}, Ords{}, Ords{})
	test(Ords{}, Ords{nil}, Ords{})
	test(Ords{Str(``)}, Ords{nil}, Ords{Str(``)})
	test(Ords{Str(``)}, Ords{nil}, Ords{nil, Str(``), nil})
	test(Ords{Str(`one`), Str(`two`)}, Ords{}, Ords{nil, Str(`one`), nil, Str(`two`)})
	test(Ords{Str(`one`), Str(`two`)}, Ords{nil}, Ords{nil, Str(`one`), nil, Str(`two`)})
	test(Ords{Str(`one`)}, Ords{Str(`one`)}, Ords{Str(`two`)})
}

func Test_Ordering_Expr(t *testing.T) {
	testExpr(t, rei(``), Ordering{})
	testExpr(t, rei(``), Ordering{Dir: DirDesc, Nulls: NullsLast, Using: Str(`<`)})

	testExpr(t, rei(`one`), Ordering{Expr: Str(`one`)})
	testExpr(t, rei(`one`, 10), Ordering{Expr: rei(`one`, 10)})
	testExpr(t, rei(`one asc`, 10), Ordering{Expr: rei(`one`, 10), Dir: DirAsc})
	testExpr(t, rei(`one desc`, 10), Ordering{Expr: rei(`one`, 10), Dir: DirDesc})
	testExpr(t, rei(`one nulls first`, 10), Ordering{Expr: rei(`one`, 10), Nulls: NullsFirst})
	testExpr(t, rei(`one nulls last`, 10), Ordering{Expr: rei(`one`, 10), Nulls: NullsLast})
	testExpr(t, rei(`one asc nulls first`, 10), Ordering{Expr: rei(`one`, 10), Dir: DirAsc, Nulls: NullsFirst})
	testExpr(t, rei(`one asc nulls last`, 10), Ordering{Expr: rei(`one`, 10), Dir: DirAsc, Nulls: NullsLast})
	testExpr(t, rei(`one desc nulls first`, 10), Ordering{Expr: rei(`one`, 10), Dir: DirDesc, Nulls: NullsFirst})
	testExpr(t, rei(`one desc nulls last`, 10), Ordering{Expr: rei(`one`, 10), Dir: DirDesc, Nulls: NullsLast})

	testExpr(t, rei(`one using <`), Ordering{Expr: Str(`one`), Using: Str(`<`)})
	testExpr(t, rei(`one using <`, 10), Ordering{Expr: rei(`one`, 10), Using: Str(`<`)})
	testExpr(t, rei(`one asc using <`, 10), Ordering{Expr: rei(`one`, 10), Dir: DirAsc, Using: Str(`<`)})
	testExpr(t, rei(`one desc using <`, 10), Ordering{Expr: rei(`one`, 10), Dir: DirDesc, Using: Str(`<`)})
	testExpr(t, rei(`one nulls first using <`, 10), Ordering{Expr: rei(`one`, 10), Nulls: NullsFirst, Using: Str(`<`)})
	testExpr(t, rei(`one nulls last using <`, 10), Ordering{Expr: rei(`one`, 10), Nulls: NullsLast, Using: Str(`<`)})
	testExpr(t, rei(`one asc nulls first using <`, 10), Ordering{Expr: rei(`one`, 10), Dir: DirAsc, Nulls: NullsFirst, Using: Str(`<`)})
	testExpr(t, rei(`one asc nulls last using <`, 10), Ordering{Expr: rei(`one`, 10), Dir: DirAsc, Nulls: NullsLast, Using: Str(`<`)})
	testExpr(t, rei(`one desc nulls first using <`, 10), Ordering{Expr: rei(`one`, 10), Dir: DirDesc, Nulls: NullsFirst, Using: Str(`<`)})
	testExpr(t, rei(`one desc nulls last using <`, 10), Ordering{Expr: rei(`one`, 10), Dir: DirDesc, Nulls: NullsLast, Using: Str(`<`)})
}

func Test_Ord_Expr(t *testing.T) {
	testExpr(t, rei(``), Ord{})
	testExpr(t, rei(``), Ord{Dir: DirDesc, Nulls: NullsLast})

	testExpr(t, rei(`"one"`), Ord{Path: Path{`one`}})
	testExpr(t, rei(`"one"`), Ord{Path: Path{`one`}})
	testExpr(t, rei(`"one" asc`), Ord{Path: Path{`one`}, Dir: DirAsc})
	testExpr(t, rei(`"one" desc`), Ord{Path: Path{`one`}, Dir: DirDesc})
	testExpr(t, rei(`"one" nulls first`), Ord{Path: Path{`one`}, Nulls: NullsFirst})
	testExpr(t, rei(`"one" nulls last`), Ord{Path: Path{`one`}, Nulls: NullsLast})
	testExpr(t, rei(`"one" asc nulls first`), Ord{Path: Path{`one`}, Dir: DirAsc, Nulls: NullsFirst})
	testExpr(t, rei(`"one" asc nulls last`), Ord{Path: Path{`one`}, Dir: DirAsc, Nulls: NullsLast})
	testExpr(t, rei(`"one" desc nulls first`), Ord{Path: Path{`one`}, Dir: DirDesc, Nulls: NullsFirst})
	testExpr(t, rei(`"one" desc nulls last`), Ord{Path: Path{`one`}, Dir: DirDesc, Nulls: NullsLast})
}

func Test_Ord_combos(t *testing.T) {
	testExpr(t, rei(``), OrdAsc{})
	testExpr(t, rei(``), OrdDesc{})
	testExpr(t, rei(``), OrdNullsFirst{})
	testExpr(t, rei(``), OrdNullsLast{})
	testExpr(t, rei(``), OrdAscNullsFirst{})
	testExpr(t, rei(``), OrdAscNullsLast{})
	testExpr(t, rei(``), OrdDescNullsFirst{})
	testExpr(t, rei(``), OrdDescNullsLast{})

	testExpr(t, rei(`("one")."two" asc`), OrdAsc{`one`, `two`})
	testExpr(t, rei(`("one")."two" desc`), OrdDesc{`one`, `two`})
	testExpr(t, rei(`("one")."two" nulls first`), OrdNullsFirst{`one`, `two`})
	testExpr(t, rei(`("one")."two" nulls last`), OrdNullsLast{`one`, `two`})
	testExpr(t, rei(`("one")."two" asc nulls first`), OrdAscNullsFirst{`one`, `two`})
	testExpr(t, rei(`("one")."two" asc nulls last`), OrdAscNullsLast{`one`, `two`})
	testExpr(t, rei(`("one")."two" desc nulls first`), OrdDescNullsFirst{`one`, `two`})
	testExpr(t, rei(`("one")."two" desc nulls last`), OrdDescNullsLast{`one`, `two`})
}

func Test_ParseOpt_OrType(t *testing.T) {
	test := func(exp r.Type, src, typ any) {
		t.Helper()

		opt := ParseOpt{Type: r.TypeOf(src)}
		opt.OrType(typ)

		eq(t, exp, opt.Type)
	}

	test(nil, nil, nil)

	test(r.TypeOf(Outer{}), nil, Outer{})
	test(r.TypeOf(Outer{}), nil, &Outer{})
	test(r.TypeOf(Outer{}), nil, (*Outer)(nil))
	test(r.TypeOf(Outer{}), nil, []Outer(nil))
	test(r.TypeOf(Outer{}), nil, []*Outer(nil))
	test(r.TypeOf(Outer{}), nil, (*[]Outer)(nil))
	test(r.TypeOf(Outer{}), nil, (*[]*Outer)(nil))

	test(r.TypeOf(Internal{}), Internal{}, nil)
	test(r.TypeOf(Internal{}), Internal{}, Outer{})
	test(r.TypeOf(Internal{}), Internal{}, &Outer{})
	test(r.TypeOf(Internal{}), Internal{}, (*Outer)(nil))
	test(r.TypeOf(Internal{}), Internal{}, []Outer(nil))
	test(r.TypeOf(Internal{}), Internal{}, []*Outer(nil))
	test(r.TypeOf(Internal{}), Internal{}, (*[]Outer)(nil))
	test(r.TypeOf(Internal{}), Internal{}, (*[]*Outer)(nil))
}

// Delegates to `(*ParserOrds).ParseSlice` which is tested separately.
func Test_ParserOrds_UnmarshalJSON(t *testing.T) {
	test := func(exp Ords, src string, typ any) {
		t.Helper()

		var par ParserOrds
		par.OrType(typ)

		eq(t, nil, par.UnmarshalJSON([]byte(src)))
		eq(t, exp, par.Ords)
	}

	test(Ords(nil), `null`, nil)
	test(Ords(nil), `null`, Outer{})
	test(Ords(nil), `[]`, nil)
	test(Ords(nil), `[]`, Outer{})
	test(Ords(nil), `[""]`, nil)
	test(Ords(nil), `[""]`, Outer{})
	test(Ords{Path{`outer_id`}}, `["outerId"]`, Outer{})

	test(
		Ords{OrdAsc{`outer_id`}, OrdDesc{`outer_name`}},
		`["outerId asc", "outerName desc"]`,
		Outer{},
	)
}

func Test_ParserOrds_ParseSlice_invalid(t *testing.T) {
	test := func(src, msg string, typ any) {
		t.Helper()

		var par ParserOrds
		par.OrType(typ)

		panics(t, msg, func() {
			try(par.ParseSlice([]string{src}))
		})
	}

	test(`one two three`, `is not a valid ordering string`, nil)
	test(`one asc nulls`, `is not a valid ordering string`, nil)
	test(`one`, `expected struct, found int`, 10)
	test(`one`, `expected struct, found string`, `some string`)
	test(`one`, `error "ErrUnknownField" while converting JSON identifier path to DB path: no DB path corresponding to JSON path "one" in type nil`, nil)
	test(`one.two`, `error "ErrUnknownField" while converting JSON identifier path to DB path: no DB path corresponding to JSON path "one.two" in type nil`, nil)
	test(`one`, `error "ErrUnknownField" while converting JSON identifier path to DB path: no DB path corresponding to JSON path "one" in type Outer`, Outer{})
	test(`one.two`, `error "ErrUnknownField" while converting JSON identifier path to DB path: no DB path corresponding to JSON path "one.two" in type Outer`, Outer{})
	test(`outer_id`, `error "ErrUnknownField" while converting JSON identifier path to DB path: no DB path corresponding to JSON path "outer_id" in type Outer`, Outer{})
	test(`OuterId`, `error "ErrUnknownField" while converting JSON identifier path to DB path: no DB path corresponding to JSON path "OuterId" in type Outer`, Outer{})
	test(`Id`, `error "ErrUnknownField" while converting JSON identifier path to DB path: no DB path corresponding to JSON path "Id" in type Outer`, Outer{})
	test(`onlyJson`, `error "ErrUnknownField" while converting JSON identifier path to DB path: no DB path corresponding to JSON path "onlyJson" in type Outer`, Outer{})
}

func testOrdsParsing(t testing.TB, exp Ords, src []string, typ any) {
	t.Helper()

	var par ParserOrds
	par.OrType(typ)

	eq(t, nil, par.ParseSlice(src))
	eq(t, exp, par.Ords)
}

func Test_ParserOrds_ParseSlice_empty(t *testing.T) {
	testOrdsParsing(t, Ords(nil), nil, nil)
	testOrdsParsing(t, Ords(nil), []string{}, nil)
	testOrdsParsing(t, Ords(nil), []string{``}, nil)
	testOrdsParsing(t, Ords(nil), []string{``, ``}, nil)
}

func Test_ParserOrds_ParseSlice_single(t *testing.T) {
	test := func(exp Expr, src string, typ any) {
		t.Helper()
		testOrdsParsing(t, Ords{exp}, []string{src}, typ)
	}

	test(Path{`outer_id`}, `outerId`, Outer{})
	test(Path{`outer_name`}, `outerName`, Outer{})
	test(Path{`embed_id`}, `embedId`, Outer{})
	test(Path{`embed_name`}, `embedName`, Outer{})
	test(Path{`internal`}, `externalInternal`, External{})
	test(Path{`internal`, `id`}, `externalInternal.internalId`, External{})
	test(Path{`internal`, `name`}, `externalInternal.internalName`, External{})
	test(OrdAsc{`outer_id`}, `outerId asc`, Outer{})
	test(OrdDesc{`outer_id`}, `outerId desc`, Outer{})
	test(OrdNullsFirst{`outer_id`}, `outerId nulls first`, Outer{})
	test(OrdNullsLast{`outer_id`}, `outerId nulls last`, Outer{})
	test(OrdAscNullsFirst{`outer_id`}, `outerId asc nulls first`, Outer{})
	test(OrdAscNullsLast{`outer_id`}, `outerId asc nulls last`, Outer{})
	test(OrdDescNullsFirst{`outer_id`}, `outerId desc nulls first`, Outer{})
	test(OrdDescNullsLast{`outer_id`}, `outerId desc nulls last`, Outer{})
	test(OrdDescNullsLast{`outer_id`}, `  outerId   dEsC   nUlLs   LaSt  `, Outer{})
}

func Test_ParserOrds_ParseSlice_multiple(t *testing.T) {
	test := func(exp Ords, src []string, typ any) {
		t.Helper()
		testOrdsParsing(t, exp, src, typ)
	}

	test(
		Ords{Path{`outer_id`}},
		[]string{``, `outerId`, ``},
		Outer{},
	)

	test(
		Ords{Path{`outer_id`}, Path{`outer_name`}},
		[]string{`outerId`, ``, `outerName`, ``},
		Outer{},
	)

	test(
		Ords{OrdAscNullsFirst{`outer_id`}, OrdDescNullsLast{`outer_name`}},
		[]string{``, `outerId asc nulls first`, ``, ``, `outerName desc nulls last`},
		Outer{},
	)
}

func Test_ParserOrds_ParseSlice_lax(t *testing.T) {
	test := func(src []string, typ any) {
		t.Helper()

		panics(t, `no DB path corresponding to JSON path`, func() {
			var par ParserOrds
			par.OrType(typ)
			try(par.ParseSlice(src))
		})

		var par ParserOrds
		par.OrType(typ)
		par.Lax = true

		eq(t, nil, par.ParseSlice(src))
		eq(t, 0, len(par.Ords))
	}

	test([]string{`outerId`}, nil)
	test([]string{`outer_id`}, Outer{})
}

func Test_ParseOpt_Filter(t *testing.T) {
	type Target struct {
		Tagged   string `json:"jsonTagged"   db:"db_tagged" ord:""`
		Untagged string `json:"jsonUntagged" db:"db_untagged"`
	}

	t.Run(`without filter`, func(t *testing.T) {
		par := ParserOrds{ParseOpt: ParseOpt{
			Type: r.TypeOf(Target{}),
		}}

		try(par.ParseSlice([]string{
			`jsonTagged asc`,
			`jsonUntagged desc`,
		}))

		eq(
			t,
			Ords{
				OrdAsc{`db_tagged`},
				OrdDesc{`db_untagged`},
			},
			par.Ords,
		)
	})

	t.Run(`with filter`, func(t *testing.T) {
		par := ParserOrds{ParseOpt: ParseOpt{
			Type:   r.TypeOf(Target{}),
			Filter: TagFilter(`ord`),
		}}

		panics(
			t,
			`no DB path corresponding to JSON path "jsonUntagged" in type Target`,
			func() {
				try(par.ParseSlice([]string{`jsonUntagged asc`}))
			},
		)
	})
}

func Test_ParserOrds_default_dir(t *testing.T) {
	test := func(typ any, src []string, exp Ords) {
		t.Helper()

		var par ParserOrds
		par.OrType(typ)

		try(par.ParseSlice(src))
		eq(t, exp, par.Ords)
	}

	test(
		struct {
			Name string `json:"name" db:"name" ord.dir:""`
		}{},
		[]string{`name`},
		Ords{Path{`name`}},
	)

	test(
		struct {
			Name string `json:"name" db:"name" ord.dir:"asc"`
		}{},
		[]string{`name`},
		Ords{OrdAsc{`name`}},
	)

	test(
		struct {
			Name string `json:"name" db:"name" ord.dir:"asc"`
		}{},
		[]string{`name desc`},
		Ords{OrdDesc{`name`}},
	)

	test(
		struct {
			Name string `json:"name" db:"name" ord.dir:"desc"`
		}{},
		[]string{`name`},
		Ords{OrdDesc{`name`}},
	)

	test(
		struct {
			Name string `json:"name" db:"name" ord.dir:"desc"`
		}{},
		[]string{`name asc`},
		Ords{OrdAsc{`name`}},
	)
}

func Test_ParserOrds_default_nulls(t *testing.T) {
	test := func(typ any, src []string, exp Ords) {
		t.Helper()

		var par ParserOrds
		par.OrType(typ)

		try(par.ParseSlice(src))
		eq(t, exp, par.Ords)
	}

	test(
		struct {
			Name string `json:"name" db:"name" ord.nulls:""`
		}{},
		[]string{`name`},
		Ords{Path{`name`}},
	)

	test(
		struct {
			Name string `json:"name" db:"name" ord.nulls:"first"`
		}{},
		[]string{`name`},
		Ords{OrdNullsFirst{`name`}},
	)

	test(
		struct {
			Name string `json:"name" db:"name" ord.nulls:"first"`
		}{},
		[]string{`name nulls last`},
		Ords{OrdNullsLast{`name`}},
	)

	test(
		struct {
			Name string `json:"name" db:"name" ord.nulls:"last"`
		}{},
		[]string{`name`},
		Ords{OrdNullsLast{`name`}},
	)

	test(
		struct {
			Name string `json:"name" db:"name" ord.nulls:"last"`
		}{},
		[]string{`name nulls first`},
		Ords{OrdNullsFirst{`name`}},
	)
}

func TestDir(t *testing.T) {
	t.Run(`String`, func(t *testing.T) {
		eq(t, ``, DirNone.String())
		eq(t, `asc`, DirAsc.String())
		eq(t, `desc`, DirDesc.String())
	})

	t.Run(`Parse`, func(t *testing.T) {
		test := func(exp Dir, src string) {
			t.Helper()
			var tar Dir
			try(tar.Parse(src))
			eq(t, exp, tar)
		}

		test(DirNone, ``)
		test(DirAsc, `asc`)
		test(DirDesc, `desc`)
	})

	t.Run(`MarshalJSON`, func(t *testing.T) {
		test := func(exp string, src Dir) {
			t.Helper()
			eq(t, exp, string(try1(json.Marshal(src))))
		}

		test(`null`, DirNone)
		test(`"asc"`, DirAsc)
		test(`"desc"`, DirDesc)
	})

	t.Run(`UnmarshalJSON`, func(t *testing.T) {
		test := func(exp Dir, src string) {
			t.Helper()
			var tar Dir
			try(json.Unmarshal([]byte(src), &tar))
			eq(t, exp, tar)
		}

		test(DirNone, `null`)
		test(DirNone, `""`)
		test(DirAsc, `"asc"`)
		test(DirDesc, `"desc"`)
	})
}
