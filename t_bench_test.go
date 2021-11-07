package sqlb

import (
	"fmt"
	r "reflect"
	"testing"
)

func Benchmark_delete_ast_create(b *testing.B) {
	for range counter(b.N) {
		_ = benchDeleteAst()
	}
}

//go:noinline
func benchDeleteAst() Expr {
	return qDeleteAst(`some_table`, External{})
}

func qDeleteAst(from Ident, where interface{}) Expr {
	return Delete{from, where}
}

func Benchmark_delete_ast_append(b *testing.B) {
	for range counter(b.N) {
		benchDeleteAstAppend()
	}
}

var exprDeleteAst = benchDeleteAst()

//go:noinline
func benchDeleteAstAppend() {
	exprDeleteAst.AppendExpr(hugeBui.Get())
}

func Benchmark_delete_ast_reify(b *testing.B) {
	for range counter(b.N) {
		benchDeleteAstReify()
	}
}

//go:noinline
func benchDeleteAstReify() {
	bui := hugeBui
	bui.Expr(exprDeleteAst)
	bui.Reify()
}

func Benchmark_delete_text_create(b *testing.B) {
	for range counter(b.N) {
		_ = benchDeleteText()
	}
}

//go:noinline
func benchDeleteText() Expr {
	return qDeleteText(`some_table`, External{})
}

func qDeleteText(ident Ident, where interface{}) Expr {
	return StrQ{`delete from :ident where :where returning *`, Dict{
		`ident`: ident,
		`where`: And{where},
	}}
}

func Benchmark_delete_text_append(b *testing.B) {
	for range counter(b.N) {
		benchDeleteTextAppend()
	}
}

var exprDeleteText = benchDeleteText()

//go:noinline
func benchDeleteTextAppend() {
	exprDeleteText.AppendExpr(hugeBui.Get())
}

func Benchmark_delete_text_reify(b *testing.B) {
	for range counter(b.N) {
		benchDeleteTextReify()
	}
}

//go:noinline
func benchDeleteTextReify() {
	bui := hugeBui
	bui.Expr(exprDeleteText)
	bui.Reify()
}

func Benchmark_struct_walk(b *testing.B) {
	for range counter(b.N) {
		benchStructWalk()
	}
}

//go:noinline
func benchStructWalk() { tCols() }

func Benchmark_TypeColsDeep(b *testing.B) {
	for range counter(b.N) {
		benchLoadCols()
	}
}

var outerType = r.TypeOf((*Outer)(nil)).Elem()

//go:noinline
func benchLoadCols() { TypeColsDeep(outerType) }

func Benchmark_loadStructDbFields(b *testing.B) {
	for range counter(b.N) {
		benchLoadColumnFields()
	}
}

//go:noinline
func benchLoadColumnFields() { loadStructDbFields(outerType) }

func Benchmark_loadStructPathMap(b *testing.B) {
	for range counter(b.N) {
		benchLoadStructPathMap()
	}
}

//go:noinline
func benchLoadStructPathMap() { loadStructPathMap(outerType) }

func Benchmark_huge_query_wrap(b *testing.B) {
	for range counter(b.N) {
		_ = benchHugeQueryWrap()
	}
}

//go:noinline
func benchHugeQueryWrap() StrQ {
	return DictQ(`:one`, Dict{`one`: hugeQueryUnwrapped})
}

var hugeQueryUnwrapped = DictQ(hugeQuery, hugeQueryArgs)

func Benchmark_huge_query_append_unwrapped(b *testing.B) {
	for range counter(b.N) {
		benchHugeQueryAppendUnwrapped()
	}
}

//go:noinline
func benchHugeQueryAppendUnwrapped() {
	hugeQueryUnwrapped.AppendExpr(hugeBui.Get())
}

func Benchmark_huge_query_append_wrapped(b *testing.B) {
	for range counter(b.N) {
		benchHugeQueryAppendWrapped()
	}
}

var hugeQueryWrapped = benchHugeQueryWrap()

//go:noinline
func benchHugeQueryAppendWrapped() {
	hugeQueryWrapped.AppendExpr(hugeBui.Get())
}

func Benchmark_make_list(b *testing.B) {
	for range counter(b.N) {
		benchMakeList()
	}
}

//go:noinline
func benchMakeList() ArgDict {
	list := make(List, 24)
	for i := range list {
		list[i] = (i + 1) * 10
	}
	return list
}

func Benchmark_make_dict(b *testing.B) {
	for range counter(b.N) {
		benchMakeDict()
	}
}

//go:noinline
func benchMakeDict() ArgDict {
	return Dict{
		`Key_c603c58746a69833a1528050c33d`: `val_e1436c61440383a80ebdc245b930`,
		`Key_abfbb9e94e4093a47683e8ef606b`: `val_a6108ccd40789cecf4da1052c5ae`,
		`Key_907b548d45948a206907ed9c9097`: `val_9271789147789ecb2beb11c97a78`,
		`Key_5ee2513a41a88d173cd53d389c14`: `val_2b6205fb4bf882ab65f3795b2384`,
		`Key_0ac8b89b46bba5d4d076e71d6232`: `val_226b2c3a4c5591084d3a120de2d8`,
		`Key_b754f88b42fcbd6c30e3bb544909`: `val_9c639ea74d099446ec3aa2a736a8`,
		`Key_e52daa684071891a1dae084bfd00`: `val_71fc2d114b2aaa3b5c1c399d28f6`,
		`Key_3106dc324be4b3ff5d477e71c593`: `val_9183d36e4b53a5e2b26ca950a721`,
		`Key_a7b558a54d178bdb6fcf3368939b`: `val_f0bc980a408c81a959168aa8aabc`,
		`Key_1622da954c8a8f6fec82e6fd3c34`: `val_4afe6fa84722a214e4e777aa6bcf`,
		`Key_fa3892644f1392aee8e66b799b3f`: `val_c45ce5ec46b7809d5df5cd1c815b`,
		`Key_b9aa15254438b0b7a32489947c50`: `val_6b119aad4bc280a3dfa675fe88a5`,

		`Key_ce59b8e14f77b6e6e9cd28cecacd`: `val_c76bd35c42d49ccb4408f92fb222`,
		`Key_87819a034834a3530b8255e76e4d`: `val_a185f0a946e894d1628bb98b673e`,
		`Key_c31042674737a95d1cba33b61687`: `val_02bae4964cfa9ebd23b5d3f57ee6`,
		`Key_7bc7a0d346c2b87e3110b2d192d3`: `val_2208de3a476299877d36f149ab94`,
		`Key_3b17f4454d44abbbeb2eb5b61235`: `val_dfb68e4d459aa5c649dcb07e0bfb`,
		`Key_83e52b714a9d8a0ba6dd87658acf`: `val_2ec2ca5046038e80cfa3cb23dff2`,
		`Key_82c96b4d4965a08fa6735e973caa`: `val_fae699f449a1aaf138b1ae2bb9b0`,
		`Key_7580ec1f4d42a7aafddf4f818b97`: `val_fc6b97924798b1b790cfb6e31750`,
		`Key_bc03a581465c873ceea04027d6ab`: `val_ab22ce72453cb2577aa731dae72c`,
		`Key_dcfa83ed4be89cf05d5e3eba6f2a`: `val_b773e8ce401c8313b1400b973fa1`,
		`Key_2bc5f64447879c1152ae9b904718`: `val_e9d6438d42339e4c62db260c458b`,
		`Key_4f0e9d9b4d1ea77c510337ae6c2a`: `val_60a4b1bf406f98826c706ab153d1`,
	}
}

func Benchmark_make_struct_dict(b *testing.B) {
	for range counter(b.N) {
		benchMakeStructDict()
	}
}

//go:noinline
func benchMakeStructDict() ArgDict {
	return StructDict{r.ValueOf(BenchStructDict{
		Key_c603c58746a69833a1528050c33d: `val_e1436c61440383a80ebdc245b930`,
		Key_abfbb9e94e4093a47683e8ef606b: `val_a6108ccd40789cecf4da1052c5ae`,
		Key_907b548d45948a206907ed9c9097: `val_9271789147789ecb2beb11c97a78`,
		Key_5ee2513a41a88d173cd53d389c14: `val_2b6205fb4bf882ab65f3795b2384`,
		Key_0ac8b89b46bba5d4d076e71d6232: `val_226b2c3a4c5591084d3a120de2d8`,
		Key_b754f88b42fcbd6c30e3bb544909: `val_9c639ea74d099446ec3aa2a736a8`,
		Key_e52daa684071891a1dae084bfd00: `val_71fc2d114b2aaa3b5c1c399d28f6`,
		Key_3106dc324be4b3ff5d477e71c593: `val_9183d36e4b53a5e2b26ca950a721`,
		Key_a7b558a54d178bdb6fcf3368939b: `val_f0bc980a408c81a959168aa8aabc`,
		Key_1622da954c8a8f6fec82e6fd3c34: `val_4afe6fa84722a214e4e777aa6bcf`,
		Key_fa3892644f1392aee8e66b799b3f: `val_c45ce5ec46b7809d5df5cd1c815b`,
		Key_b9aa15254438b0b7a32489947c50: `val_6b119aad4bc280a3dfa675fe88a5`,
	})}
}

type BenchStructDict struct {
	Key_c603c58746a69833a1528050c33d interface{}
	Key_abfbb9e94e4093a47683e8ef606b interface{}
	Key_907b548d45948a206907ed9c9097 interface{}
	Key_5ee2513a41a88d173cd53d389c14 interface{}
	Key_0ac8b89b46bba5d4d076e71d6232 interface{}
	Key_b754f88b42fcbd6c30e3bb544909 interface{}
	Key_e52daa684071891a1dae084bfd00 interface{}
	Key_3106dc324be4b3ff5d477e71c593 interface{}
	Key_a7b558a54d178bdb6fcf3368939b interface{}
	Key_1622da954c8a8f6fec82e6fd3c34 interface{}
	Key_fa3892644f1392aee8e66b799b3f interface{}
	Key_b9aa15254438b0b7a32489947c50 interface{}
}

func (self BenchStructDict) Key_ce59b8e14f77b6e6e9cd28cecacd() string {
	return `val_c76bd35c42d49ccb4408f92fb222`
}

func (self BenchStructDict) Key_87819a034834a3530b8255e76e4d() string {
	return `val_a185f0a946e894d1628bb98b673e`
}

func (self BenchStructDict) Key_c31042674737a95d1cba33b61687() string {
	return `val_02bae4964cfa9ebd23b5d3f57ee6`
}

func (self BenchStructDict) Key_7bc7a0d346c2b87e3110b2d192d3() string {
	return `val_2208de3a476299877d36f149ab94`
}

func (self BenchStructDict) Key_3b17f4454d44abbbeb2eb5b61235() string {
	return `val_dfb68e4d459aa5c649dcb07e0bfb`
}

func (self BenchStructDict) Key_83e52b714a9d8a0ba6dd87658acf() string {
	return `val_2ec2ca5046038e80cfa3cb23dff2`
}

func (self BenchStructDict) Key_82c96b4d4965a08fa6735e973caa() string {
	return `val_fae699f449a1aaf138b1ae2bb9b0`
}

func (self BenchStructDict) Key_7580ec1f4d42a7aafddf4f818b97() string {
	return `val_fc6b97924798b1b790cfb6e31750`
}

func (self BenchStructDict) Key_bc03a581465c873ceea04027d6ab() string {
	return `val_ab22ce72453cb2577aa731dae72c`
}

func (self BenchStructDict) Key_dcfa83ed4be89cf05d5e3eba6f2a() string {
	return `val_b773e8ce401c8313b1400b973fa1`
}

func (self BenchStructDict) Key_2bc5f64447879c1152ae9b904718() string {
	return `val_e9d6438d42339e4c62db260c458b`
}

func (self BenchStructDict) Key_4f0e9d9b4d1ea77c510337ae6c2a() string {
	return `val_60a4b1bf406f98826c706ab153d1`
}

func Benchmark_list_access(b *testing.B) {
	list := benchList
	benchTestListAccess(b, list)
	b.ResetTimer()

	for range counter(b.N) {
		benchListAccess(list)
	}
}

var benchList = benchMakeList()

func benchTestListAccess(t testing.TB, list ArgDict) {
	test := func(key int, expVal interface{}, expOk bool) {
		t.Helper()
		val, ok := list.GotOrdinal(key)
		eq(t, expVal, val)
		eq(t, expOk, ok)
	}

	test(-1, nil, false)
	test(0, 10, true)
	test(1, 20, true)
	test(2, 30, true)
	test(3, 40, true)
	test(4, 50, true)
	test(5, 60, true)
	test(6, 70, true)
	test(7, 80, true)
	test(8, 90, true)
	test(9, 100, true)
	test(10, 110, true)
	test(11, 120, true)
	test(12, 130, true)
	test(13, 140, true)
	test(14, 150, true)
	test(15, 160, true)
	test(16, 170, true)
	test(17, 180, true)
	test(18, 190, true)
	test(19, 200, true)
	test(20, 210, true)
	test(21, 220, true)
	test(22, 230, true)
	test(23, 240, true)
	test(24, nil, false)
}

//go:noinline
func benchListAccess(list ArgDict) {
	list.GotOrdinal(0)
	list.GotOrdinal(1)
	list.GotOrdinal(2)
	list.GotOrdinal(3)
	list.GotOrdinal(4)
	list.GotOrdinal(5)
	list.GotOrdinal(6)
	list.GotOrdinal(7)
	list.GotOrdinal(8)
	list.GotOrdinal(9)
	list.GotOrdinal(10)
	list.GotOrdinal(11)
	list.GotOrdinal(12)
	list.GotOrdinal(13)
	list.GotOrdinal(14)
	list.GotOrdinal(15)
	list.GotOrdinal(16)
	list.GotOrdinal(17)
	list.GotOrdinal(18)
	list.GotOrdinal(19)
	list.GotOrdinal(20)
	list.GotOrdinal(21)
	list.GotOrdinal(22)
	list.GotOrdinal(23)
}

func Benchmark_dict_access(b *testing.B) {
	for range counter(b.N) {
		benchDictAccess(benchDict)
	}
}

var benchDict = benchMakeDict()

func Benchmark_struct_dict_access(b *testing.B) {
	for range counter(b.N) {
		benchDictAccess(benchStructDict)
	}
}

var benchStructDict = benchMakeStructDict()

//go:noinline
func benchDictAccess(dict ArgDict) {
	dict.GotNamed(`Key_c603c58746a69833a1528050c33d`)
	dict.GotNamed(`Key_abfbb9e94e4093a47683e8ef606b`)
	dict.GotNamed(`Key_907b548d45948a206907ed9c9097`)
	dict.GotNamed(`Key_5ee2513a41a88d173cd53d389c14`)
	dict.GotNamed(`Key_0ac8b89b46bba5d4d076e71d6232`)
	dict.GotNamed(`Key_b754f88b42fcbd6c30e3bb544909`)
	dict.GotNamed(`Key_e52daa684071891a1dae084bfd00`)
	dict.GotNamed(`Key_3106dc324be4b3ff5d477e71c593`)
	dict.GotNamed(`Key_a7b558a54d178bdb6fcf3368939b`)
	dict.GotNamed(`Key_1622da954c8a8f6fec82e6fd3c34`)
	dict.GotNamed(`Key_fa3892644f1392aee8e66b799b3f`)
	dict.GotNamed(`Key_b9aa15254438b0b7a32489947c50`)

	dict.GotNamed(`Key_ce59b8e14f77b6e6e9cd28cecacd`)
	dict.GotNamed(`Key_87819a034834a3530b8255e76e4d`)
	dict.GotNamed(`Key_c31042674737a95d1cba33b61687`)
	dict.GotNamed(`Key_7bc7a0d346c2b87e3110b2d192d3`)
	dict.GotNamed(`Key_3b17f4454d44abbbeb2eb5b61235`)
	dict.GotNamed(`Key_83e52b714a9d8a0ba6dd87658acf`)
	dict.GotNamed(`Key_82c96b4d4965a08fa6735e973caa`)
	dict.GotNamed(`Key_7580ec1f4d42a7aafddf4f818b97`)
	dict.GotNamed(`Key_bc03a581465c873ceea04027d6ab`)
	dict.GotNamed(`Key_dcfa83ed4be89cf05d5e3eba6f2a`)
	dict.GotNamed(`Key_2bc5f64447879c1152ae9b904718`)
	dict.GotNamed(`Key_4f0e9d9b4d1ea77c510337ae6c2a`)
}

func Benchmark_bui_expr_alloc(b *testing.B) {
	{
		benchBuiExprAlloc()
		tHugeBuiEmpty(b)
	}
	b.ResetTimer()

	for range counter(b.N) {
		benchBuiExprAlloc()
	}
}

//go:noinline
func benchBuiExprAlloc() {
	bui := hugeBui
	var expr Eq
	// Go defect: this allocates and uses dynamic dispatch, even though it should
	// be possible to specialize the code to avoid both (Go 1.17).
	bui.Expr(&expr)
}

func Benchmark_bui_expr_zero_alloc(b *testing.B) {
	{
		benchBuiExprZeroAlloc()
		tHugeBuiEmpty(b)
	}
	b.ResetTimer()

	for range counter(b.N) {
		benchBuiExprZeroAlloc()
	}
}

//go:noinline
func benchBuiExprZeroAlloc() {
	bui := hugeBui
	var expr Eq
	bui.Set(expr.AppendExpr(bui.Get()))
}

func Benchmark_Cond_AppendExpr(b *testing.B) {
	for range counter(b.N) {
		benchCondAppendExpr()
	}
}

var benchCond = Cond{``, ``, External{}}

//go:noinline
func benchCondAppendExpr() {
	benchCond.AppendExpr(hugeBui.Get())
}

func Benchmark_huge_query_preparse(b *testing.B) {
	for range counter(b.N) {
		benchHugeQueryPreparse()
	}
}

//go:noinline
func benchHugeQueryPreparse() Prep {
	prep := Prep{Source: hugeQuery}
	prep.Parse()
	return prep
}

func Benchmark_huge_query_append_preparsed(b *testing.B) {
	for range counter(b.N) {
		benchHugeQueryAppendPreparsed()
	}
}

var hugePrep = Preparse(hugeQuery)

//go:noinline
func benchHugeQueryAppendPreparsed() {
	text, args := hugeBui.Get()
	hugePrep.AppendParamExpr(text, args, hugeQueryArgs)
}

func Benchmark_error_make(b *testing.B) {
	for range counter(b.N) {
		_ = benchErrorMake()
	}
}

var (
	benchErr      = benchErrorMake()
	benchErrCause = fmt.Errorf(`some cause`)
)

//go:noinline
func benchErrorMake() error {
	return ErrInternal{Err{`doing something`, benchErrCause}}
}

func Benchmark_error_format(b *testing.B) {
	for range counter(b.N) {
		benchErrorFormat()
	}
}

//go:noinline
func benchErrorFormat() {
	_ = benchErr.Error()
}

func Benchmark_loadPrep(b *testing.B) {
	for range counter(b.N) {
		benchLoadPrep()
	}
}

//go:noinline
func benchLoadPrep() { Preparse(hugeQuery) }

func Benchmark_dict_to_iface_alloc(b *testing.B) {
	dict := benchMakeDict().(Dict)
	b.ResetTimer()
	for range counter(b.N) {
		benchDictToIfaceAlloc(dict)
	}
}

//go:noinline
func benchDictToIfaceAlloc(val Dict) { nop(val) }

var nop = func(ArgDict) {}

func Benchmark_parse_ords(b *testing.B) {
	for range counter(b.N) {
		benchParseOrds()
	}
}

//go:noinline
func benchParseOrds() {
	parser := benchOrdsParser
	try(parser.ParseSlice(benchOrderingStrings))
}

var benchOrdsParser = OrdsParser{
	Ords: make(Ords, 0, len(benchOrderingStrings)),
	Type: r.TypeOf((*External)(nil)).Elem(),
}

var benchOrderingStrings = []string{
	`externalId`,
	`externalName desc`,
	`externalInternal.internalId nulls last`,
	`externalInternal.internalName desc nulls first`,
}

func Benchmark_StructInsert_AppendExpr(b *testing.B) {
	expr := StructInsert{testOuter}
	b.ResetTimer()

	for range counter(b.N) {
		expr.AppendExpr(hugeBui.Get())
	}
}

func Benchmark_StructValues_AppendExpr(b *testing.B) {
	expr := StructValues{testOuter}
	b.ResetTimer()

	for range counter(b.N) {
		expr.AppendExpr(hugeBui.Get())
	}
}
