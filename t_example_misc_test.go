package sqlb_test

import (
	"encoding/json"
	"fmt"

	s "github.com/mitranim/sqlb"
)

// Copy of `ExampleStrQ_nested` for package-level docs.
func Example_composition() {
	inner := s.StrQ{
		`select * from some_table where col0 = :val`,
		s.Dict{`val`: 10},
	}

	outer := s.StrQ{
		`select * from (:inner) as _ where col1 = :val`,
		s.Dict{`inner`: inner, `val`: 20},
	}

	fmt.Println(s.Reify(outer))
	// Output:
	// select * from (select * from some_table where col0 = $1) as _ where col1 = $2 [10 20]
}

func ExampleBui_CatchExprs() {
	bui := s.MakeBui(1024, 16)

	err := bui.CatchExprs(
		s.Select{`some_table`, s.Ands{true, false}},
	)
	if err != nil {
		panic(err)
	}

	text, args := bui.Reify()
	fmt.Println(text)
	fmt.Println(args)

	// Output:
	// select * from "some_table" where $1 and $2
	// [true false]
}

func ExampleStr_stringInterpolation() {
	fmt.Println(s.Reify(
		s.StrQ{
			`select :col from some_table where :col <> :val`,
			s.Dict{
				`col`: s.Str(`some_col`),
				`val`: `some_val`,
			},
		},
	))
	// Output:
	// select some_col from some_table where some_col <> $1 [some_val]
}

func ExampleIdent() {
	fmt.Println(s.Ident(``))
	fmt.Println(s.Ident(`one`))
	// Output:
	// ""
	// "one"
}

func ExampleIdent_interpolation() {
	fmt.Println(s.Reify(
		s.StrQ{
			`select :col from some_table where :col <> :val`,
			s.Dict{
				`col`: s.Ident(`some_col`),
				`val`: `some_val`,
			},
		},
	))
	// Output:
	// select "some_col" from some_table where "some_col" <> $1 [some_val]
}

func ExampleIdentifier() {
	fmt.Println(s.Identifier{`one`})
	fmt.Println(s.Identifier{`one`, `two`})
	fmt.Println(s.Identifier{`one`, `two`, `three`})
	// Output:
	// "one"
	// "one"."two"
	// "one"."two"."three"
}

func ExamplePath() {
	fmt.Println(s.Path{`one`})
	fmt.Println(s.Path{`one`, `two`})
	fmt.Println(s.Path{`one`, `two`, `three`})
	// Output:
	// "one"
	// ("one")."two"
	// ("one")."two"."three"
}

func ExamplePseudoPath() {
	fmt.Println(s.PseudoPath{`one`})
	fmt.Println(s.PseudoPath{`one`, `two`})
	fmt.Println(s.PseudoPath{`one`, `two`, `three`})
	// Output:
	// "one"
	// "one.two"
	// "one.two.three"
}

func ExampleAliasedPath() {
	fmt.Println(s.AliasedPath{`one`})
	fmt.Println(s.AliasedPath{`one`, `two`})
	fmt.Println(s.AliasedPath{`one`, `two`, `three`})
	// Output:
	// "one"
	// ("one")."two" as "one.two"
	// ("one")."two"."three" as "one.two.three"
}

func ExampleExprs() {
	type Filter struct {
		Slug string `db:"slug"`
	}

	fmt.Println(s.Reify(
		s.Select{`some_table`, Filter{`some_slug`}},
	))
	// Output:
	// select * from "some_table" where "slug" = $1 [some_slug]
}

func ExampleAny() {
	fmt.Println(s.Reify(s.Any{}))
	fmt.Println(s.Reify(s.Any{[]int{10, 20}}))
	fmt.Println(s.Reify(s.Any{s.Table{`some_table`}}))

	// Output:
	// any ($1) [<nil>]
	// any ($1) [[10 20]]
	// any (table "some_table") []
}

func ExampleAssign() {
	fmt.Println(s.Reify(s.Assign{
		`some_col`,
		`arbitrary_value`,
	}))

	fmt.Println(s.Reify(s.Assign{
		`some_col`,
		s.Path{`some_table`, `another_col`},
	}))

	// Output:
	// "some_col" = $1 [arbitrary_value]
	// "some_col" = (("some_table")."another_col") []
}

func ExampleEq() {
	fmt.Println(s.Reify(s.Eq{10, 20}))

	fmt.Println(s.Reify(s.Eq{
		s.Ident(`some_col`),
		nil,
	}))

	fmt.Println(s.Reify(s.Eq{
		s.Ident(`some_col`),
		s.Ident(`another_col`),
	}))

	fmt.Println(s.Reify(s.Eq{
		s.Ident(`some_col`),
		s.Path{`some_table`, `another_col`},
	}))

	// Output:
	// $1 = $2 [10 20]
	// ("some_col") is null []
	// ("some_col") = ("another_col") []
	// ("some_col") = (("some_table")."another_col") []
}

func ExampleNeq() {
	fmt.Println(s.Reify(s.Neq{10, 20}))

	fmt.Println(s.Reify(s.Neq{
		s.Ident(`some_col`),
		nil,
	}))

	fmt.Println(s.Reify(s.Neq{
		s.Ident(`some_col`),
		s.Ident(`another_col`),
	}))

	fmt.Println(s.Reify(s.Neq{
		s.Ident(`some_col`),
		s.Path{`some_table`, `another_col`},
	}))

	// Output:
	// $1 <> $2 [10 20]
	// ("some_col") is not null []
	// ("some_col") <> ("another_col") []
	// ("some_col") <> (("some_table")."another_col") []
}

func ExampleEqAny() {
	fmt.Println(s.Reify(s.EqAny{
		10,
		[]int{20, 30},
	}))

	fmt.Println(s.Reify(s.EqAny{
		s.Ident(`some_col`),
		[]int{20, 30},
	}))

	fmt.Println(s.Reify(s.EqAny{
		s.Ident(`some_col`),
		s.Table{`some_table`},
	}))

	// Output:
	// $1 = any ($2) [10 [20 30]]
	// ("some_col") = any ($1) [[20 30]]
	// ("some_col") = any (table "some_table") []
}

func ExampleNeqAny() {
	fmt.Println(s.Reify(s.NeqAny{
		10,
		[]int{20, 30},
	}))

	fmt.Println(s.Reify(s.NeqAny{
		s.Ident(`some_col`),
		[]int{20, 30},
	}))

	fmt.Println(s.Reify(s.NeqAny{
		s.Ident(`some_col`),
		s.Table{`some_table`},
	}))

	// Output:
	// $1 <> any ($2) [10 [20 30]]
	// ("some_col") <> any ($1) [[20 30]]
	// ("some_col") <> any (table "some_table") []
}

func ExampleNot() {
	fmt.Println(s.Reify(s.Not{}))
	fmt.Println(s.Reify(s.Not{true}))
	fmt.Println(s.Reify(s.Not{s.Ident(`some_col`)}))
	// Output:
	// not $1 [<nil>]
	// not $1 [true]
	// not ("some_col") []
}

func ExampleAnd_struct() {
	fmt.Println(s.Reify(s.And{struct{}{}}))

	fmt.Println(s.Reify(s.And{struct {
		Col0 bool        `db:"col0"`
		Col1 interface{} `db:"col1"`
		Col2 interface{} `db:"col2"`
	}{
		true,
		nil,
		s.Call{`some_func`, []int{10}},
	}}))

	// Output:
	// true []
	// "col0" = $1 and "col1" is null and "col2" = (some_func ($2)) [true 10]
}

func ExampleAnd_slice() {
	type list = []interface{}

	fmt.Println(s.Reify(s.And{nil}))
	fmt.Println(s.Reify(s.And{list{}}))
	fmt.Println(s.Reify(s.And{list{true, false, s.Ident(`some_col`)}}))

	// Output:
	// true []
	// true []
	// $1 and $2 and ("some_col") [true false]
}

func ExampleOr_struct() {
	fmt.Println(s.Reify(s.Or{struct{}{}}))

	fmt.Println(s.Reify(s.Or{struct {
		Col0 bool        `db:"col0"`
		Col1 interface{} `db:"col1"`
		Col2 interface{} `db:"col2"`
	}{
		true,
		nil,
		s.Call{`some_func`, []int{10}},
	}}))

	// Output:
	// false []
	// "col0" = $1 or "col1" is null or "col2" = (some_func ($2)) [true 10]
}

func ExampleOr_slice() {
	type list = []interface{}

	fmt.Println(s.Reify(s.Or{nil}))
	fmt.Println(s.Reify(s.Or{list{}}))
	fmt.Println(s.Reify(s.Or{list{true, false, s.Ident(`some_col`)}}))

	// Output:
	// false []
	// false []
	// $1 or $2 or ("some_col") [true false]
}

func ExampleAnds() {
	fmt.Println(s.Reify(s.Ands{}))
	fmt.Println(s.Reify(s.Ands{true, false, s.Ident(`some_col`)}))
	// Output:
	// true []
	// $1 and $2 and ("some_col") [true false]
}

func ExampleOrs() {
	fmt.Println(s.Reify(s.Ors{}))
	fmt.Println(s.Reify(s.Ors{true, false, s.Ident(`some_col`)}))
	// Output:
	// false []
	// $1 or $2 or ("some_col") [true false]
}

func ExampleCols_nonStruct() {
	fmt.Println(s.Cols{})
	fmt.Println(s.Cols{(*int)(nil)})
	fmt.Println(s.Cols{(*[]string)(nil)})
	// Output:
	// *
	// *
	// *
}

func ExampleCols_struct() {
	type Internal struct {
		Id   string `db:"id"`
		Name string `db:"name"`
	}

	type External struct {
		Id       string   `db:"id"`
		Name     string   `db:"name"`
		Internal Internal `db:"internal"`
	}

	fmt.Println(s.Cols{(*External)(nil)})
	// Output:
	// "id", "name", "internal"
}

func ExampleColsDeep_nonStruct() {
	fmt.Println(s.ColsDeep{})
	fmt.Println(s.ColsDeep{(*int)(nil)})
	fmt.Println(s.ColsDeep{(*[]string)(nil)})
	// Output:
	// *
	// *
	// *
}

func ExampleColsDeep_struct() {
	type Internal struct {
		Id   string `db:"id"`
		Name string `db:"name"`
	}

	type External struct {
		Id       string   `db:"id"`
		Name     string   `db:"name"`
		Internal Internal `db:"internal"`
	}

	fmt.Println(s.ColsDeep{(*External)(nil)})
	// Output:
	// "id", "name", ("internal")."id" as "internal.id", ("internal")."name" as "internal.name"
}

func ExampleStructValues() {
	fmt.Println(s.Reify(s.StructValues{struct {
		Col0 bool        `db:"col0"`
		Col1 interface{} `db:"col1"`
		Col2 interface{} `db:"col2"`
	}{
		true,
		nil,
		s.Call{`some_func`, []int{10}},
	}}))
	// Output:
	// $1, $2, (some_func ($3)) [true <nil> 10]
}

func ExampleStructInsert_empty() {
	fmt.Println(s.StructInsert{})
	// Output:
	// default values
}

func ExampleStructInsert_nonEmpty() {
	fmt.Println(s.Reify(s.StructInsert{struct {
		Col0 bool        `db:"col0"`
		Col1 interface{} `db:"col1"`
		Col2 interface{} `db:"col2"`
	}{
		true,
		nil,
		s.Call{`some_func`, []int{10}},
	}}))
	// Output:
	// ("col0", "col1", "col2") values ($1, $2, (some_func ($3))) [true <nil> 10]
}

func ExampleStructAssign() {
	fmt.Println(s.Reify(s.StructAssign{struct {
		Col0 bool        `db:"col0"`
		Col1 interface{} `db:"col1"`
		Col2 interface{} `db:"col2"`
	}{
		true,
		nil,
		s.Call{`some_func`, []int{10}},
	}}))
	// Output:
	// "col0" = $1, "col1" = $2, "col2" = (some_func ($3)) [true <nil> 10]
}

func ExampleSelectCols_asIs() {
	fmt.Println(s.SelectCols{s.Table{`some_table`}, nil})
	// Output:
	// table "some_table"
}

func ExampleSelectCols_cols() {
	type SomeStruct struct {
		Col0 string `db:"col0"`
		Col1 string `db:"col1"`
		Col2 string `db:"-"`
	}

	fmt.Println(s.SelectCols{s.Table{`some_table`}, (*SomeStruct)(nil)})
	// Output:
	// with _ as (table "some_table") select "col0", "col1" from _
}

func ExampleSelectColsDeep_asIs() {
	fmt.Println(s.SelectColsDeep{s.Table{`some_table`}, nil})
	// Output:
	// table "some_table"
}

func ExampleSelectColsDeep_cols() {
	type SomeStruct struct {
		Outer string `db:"outer"`
		Inner struct {
			Name string `db:"name"`
		} `db:"inner"`
	}

	fmt.Println(s.SelectColsDeep{s.Table{`some_table`}, (*SomeStruct)(nil)})
	// Output:
	// with _ as (table "some_table") select "outer", ("inner")."name" as "inner.name" from _
}

func ExampleSelect_unfiltered() {
	fmt.Println(s.Reify(s.Select{`some_table`, nil}))
	// Output:
	// select * from "some_table" []
}

func ExampleSelect_filtered() {
	type Filter struct {
		Col0 int64 `db:"col0"`
		Col1 int64 `db:"col1"`
	}

	fmt.Println(s.Reify(s.Select{`some_table`, Filter{10, 20}}))
	// Output:
	// select * from "some_table" where "col0" = $1 and "col1" = $2 [10 20]
}

func ExampleInsert_empty() {
	fmt.Println(s.Reify(s.Insert{`some_table`, nil}))
	// Output:
	// insert into "some_table" default values returning * []
}

func ExampleInsert_nonEmpty() {
	type Fields struct {
		Col0 int64 `db:"col0"`
		Col1 int64 `db:"col1"`
	}

	fmt.Println(s.Reify(s.Insert{`some_table`, Fields{10, 20}}))

	// Output:
	// insert into "some_table" ("col0", "col1") values ($1, $2) returning * [10 20]
}

func ExampleUpdate() {
	type Filter struct {
		Col0 int64 `db:"col0"`
		Col1 int64 `db:"col1"`
	}

	type Fields struct {
		Col2 int64 `db:"col2"`
		Col3 int64 `db:"col3"`
	}

	fmt.Println(s.Reify(
		s.Update{`some_table`, Filter{10, 20}, Fields{30, 40}},
	))

	// Output:
	// update "some_table" set "col2" = $1, "col3" = $2 where "col0" = $3 and "col1" = $4 returning * [30 40 10 20]
}

func ExampleDelete_unfiltered() {
	fmt.Println(s.Reify(s.Delete{`some_table`, nil}))
	// Output:
	// delete from "some_table" where null returning * []
}

func ExampleDelete_filtered() {
	type Filter struct {
		Col0 int64 `db:"col0"`
		Col1 int64 `db:"col1"`
	}

	fmt.Println(s.Reify(s.Delete{`some_table`, Filter{10, 20}}))

	// Output:
	// delete from "some_table" where "col0" = $1 and "col1" = $2 returning * [10 20]
}

func ExampleCall_empty() {
	fmt.Println(s.Call{`some_func`, nil})
	// Output:
	// some_func ()
}

func ExampleCall_arguments() {
	fmt.Println(s.Reify(s.Call{`some_func`, []int{10, 20, 30}}))
	// Output:
	// some_func ($1, $2, $3) [10 20 30]
}

func ExampleCall_subExpression() {
	fmt.Println(s.Call{`exists`, s.Table{`some_table`}})
	// Output:
	// exists (table "some_table")
}

func ExampleCall_subExpressions() {
	fmt.Println(s.Call{`some_func`, []s.Ident{`one`, `two`}})
	// Output:
	// some_func (("one"), ("two"))
}

func ExampleRowNumberOver_empty() {
	fmt.Println(s.RowNumberOver{})
	// Output:
	// 0
}

func ExampleRowNumberOver_nonEmpty() {
	fmt.Println(s.RowNumberOver{s.Ords{s.OrdDesc{`some_col`}}})
	// Output:
	// row_number() over (order by "some_col" desc)
}

func ExampleListQ() {
	fmt.Println(s.Reify(
		s.ListQ(`
			select * from some_table where col_one = $1 and col_two = $2
		`, 10, 20),
	))
	// Output:
	// select * from some_table where col_one = $1 and col_two = $2 [10 20]
}

func ExampleDictQ() {
	fmt.Println(s.Reify(
		s.DictQ(`
			select * from some_table where col_one = :one and col_two = :two
		`, map[string]interface{}{
			`one`: 10,
			`two`: 20,
		}),
	))
	// Output:
	// select * from some_table where col_one = $1 and col_two = $2 [10 20]
}

func ExampleStrQ_empty() {
	fmt.Println(s.Reify(s.StrQ{}))
	// Output:
	// []
}

func ExampleStrQ_simple() {
	fmt.Println(s.Reify(
		s.StrQ{
			`select * from some_table where col_one = :one and col_two = :two`,
			s.Dict{
				`one`: 10,
				`two`: 20,
			},
		},
	))
	// Output:
	// select * from some_table where col_one = $1 and col_two = $2 [10 20]
}

func ExampleStrQ_structs() {
	type Output struct {
		Col0 string `db:"col0"`
		Col1 string `db:"col1"`
	}

	type Filter struct {
		Col2 int64 `db:"col2"`
		Col3 int64 `db:"col3"`
	}

	fmt.Println(s.Reify(
		s.StrQ{
			`select :cols from some_table where :filter`,
			s.Dict{
				`cols`:   s.Cols{(*Output)(nil)},
				`filter`: s.And{Filter{10, 20}},
			},
		},
	))
	// Output:
	// select "col0", "col1" from some_table where "col2" = $1 and "col3" = $2 [10 20]
}

func ExampleStrQ_structInput() {
	type Output struct {
		Col0 string `db:"col0"`
		Col1 string `db:"col1"`
	}

	type Filter struct {
		Col2 int64 `db:"col2"`
		Col3 int64 `db:"col3"`
	}

	type Input struct {
		Cols   s.Cols
		Filter s.And
	}

	fmt.Println(s.Reify(
		s.StructQ(`
			select :Cols from some_table where :Filter
		`, Input{
			s.Cols{(*Output)(nil)},
			s.And{Filter{10, 20}},
		}),
	))
	// Output:
	// select "col0", "col1" from some_table where "col2" = $1 and "col3" = $2 [10 20]
}

func ExampleStrQ_nested() {
	inner := s.StrQ{
		`select * from some_table where col0 = :val`,
		s.Dict{`val`: 10},
	}

	outer := s.StrQ{
		`select * from (:inner) as _ where col1 = :val`,
		s.Dict{`inner`: inner, `val`: 20},
	}

	fmt.Println(s.Reify(outer))
	// Output:
	// select * from (select * from some_table where col0 = $1) as _ where col1 = $2 [10 20]
}

func ExampleOrds_empty() {
	fmt.Println(s.Ords{})
	// Output:
}

func ExampleOrds_manual() {
	fmt.Println(s.Ords{
		s.OrdDesc{`col0`},
		s.Str(`random() asc`),
	})
	// Output:
	// order by "col0" desc, random() asc
}

func ExampleOrds_parse() {
	type SomeStruct struct {
		Col0 string `json:"jsonField0" db:"dbCol0"`
		Col1 string `json:"jsonField1" db:"dbCol1"`
	}

	parser := s.OrdsParserFor((*SomeStruct)(nil))

	err := parser.ParseSlice([]string{`jsonField0 asc`, `jsonField1 desc nulls last`})
	if err != nil {
		panic(err)
	}

	fmt.Printf("%#v\n\n", parser.Ords)
	fmt.Println(parser.Ords)
	// Output:
	// sqlb.Ords{sqlb.OrdAsc{"dbCol0"}, sqlb.OrdDescNullsLast{"dbCol1"}}
	//
	// order by "dbCol0" asc, "dbCol1" desc nulls last
}

func ExampleOrdsParser_ParseSlice() {
	type SomeStruct struct {
		Col0 string `json:"jsonField0" db:"dbCol0"`
		Col1 string `json:"jsonField1" db:"dbCol1"`
	}

	parser := s.OrdsParserFor((*SomeStruct)(nil))

	err := parser.ParseSlice([]string{`jsonField0 asc`, `jsonField1 desc nulls last`})
	if err != nil {
		panic(err)
	}

	fmt.Printf("%#v\n\n", parser.Ords)
	fmt.Println(parser.Ords)
	// Output:
	// sqlb.Ords{sqlb.OrdAsc{"dbCol0"}, sqlb.OrdDescNullsLast{"dbCol1"}}
	//
	// order by "dbCol0" asc, "dbCol1" desc nulls last
}

func ExampleOrdsParser_UnmarshalJSON() {
	type SomeStruct struct {
		Col0 string `json:"jsonField0" db:"dbCol0"`
		Col1 string `json:"jsonField1" db:"dbCol1"`
	}

	parser := s.OrdsParserFor((*SomeStruct)(nil))

	err := json.Unmarshal(
		[]byte(`["jsonField0 asc", "jsonField1 desc nulls last"]`),
		&parser,
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%#v\n\n", parser.Ords)
	fmt.Println(parser.Ords)
	// Output:
	// sqlb.Ords{sqlb.OrdAsc{"dbCol0"}, sqlb.OrdDescNullsLast{"dbCol1"}}
	//
	// order by "dbCol0" asc, "dbCol1" desc nulls last
}

func ExampleLimitUint() {
	fmt.Println(s.Reify(
		s.Exprs{s.Select{`some_table`, nil}, s.LimitUint(10)},
	))
	// Output:
	// select * from "some_table" limit 10 []
}

func ExampleOffsetUint() {
	fmt.Println(s.Reify(
		s.Exprs{s.Select{`some_table`, nil}, s.OffsetUint(10)},
	))
	// Output:
	// select * from "some_table" offset 10 []
}
