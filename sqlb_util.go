package sqlb

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	r "reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"
)

const (
	ordinalParamPrefix = '$'
	namedParamPrefix   = ':'
	doubleColonPrefix  = `::`
	commentLinePrefix  = `--`
	commentBlockPrefix = `/*`
	commentBlockSuffix = `*/`
	quoteSingle        = '\''
	quoteDouble        = '"'
	quoteGrave         = '`'

	byteLen                    = 1
	expectedStructNestingDepth = 8
)

var (
	typeTime        = r.TypeOf((*time.Time)(nil)).Elem()
	typeBytes       = r.TypeOf((*[]byte)(nil)).Elem()
	sqlScannerRtype = r.TypeOf((*sql.Scanner)(nil)).Elem()

	charsetDigitDec   = new(charset).addStr(`0123456789`)
	charsetIdentStart = new(charset).addStr(`ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_`)
	charsetIdent      = new(charset).addSet(charsetIdentStart).addSet(charsetDigitDec)
	charsetSpace      = new(charset).addStr(" \t\v")
	charsetNewline    = new(charset).addStr("\r\n")
	charsetWhitespace = new(charset).addSet(charsetSpace).addSet(charsetNewline)
	charsetDelimStart = new(charset).addSet(charsetWhitespace).addStr(`([{.`)
	charsetDelimEnd   = new(charset).addSet(charsetWhitespace).addStr(`,}])`)
)

type charset [256]bool

func (self *charset) has(val byte) bool { return self[val] }

func (self *charset) addStr(vals string) *charset {
	for _, val := range vals {
		self[val] = true
	}
	return self
}

func (self *charset) addSet(vals *charset) *charset {
	for ind, val := range vals {
		if val {
			self[ind] = true
		}
	}
	return self
}

type structNestedDbField struct {
	Field  r.StructField
	DbPath []string
}

type structPath struct {
	Name        string
	FieldIndex  []int
	MethodIndex int
}

type structFieldValue struct {
	Field r.StructField
	Value r.Value
}

func cacheOf[Key, Val any](fun func(Key) Val) *cache[Key, Val] {
	return &cache[Key, Val]{Func: fun}
}

type cache[Key, Val any] struct {
	sync.Map
	Func func(Key) Val
}

// Susceptible to "thundering herd". An improvement from no caching, but still
// not ideal.
func (self *cache[Key, Val]) Get(key Key) Val {
	iface, ok := self.Load(key)
	if ok {
		return iface.(Val)
	}

	val := self.Func(key)
	self.Store(key, val)
	return val
}

func leadingNewlineSize(val string) int {
	if len(val) >= 2 && val[0] == '\r' && val[1] == '\n' {
		return 2
	}
	if len(val) >= 1 && (val[0] == '\r' || val[0] == '\n') {
		return 1
	}
	return 0
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
Allocation-free conversion. Returns a byte slice backed by the provided string.
Mutations are reflected in the source string, unless it's backed by constant
storage, in which case they trigger a segfault. Reslicing is ok. Should be safe
as long as the resulting bytes are not mutated.
*/
func stringToBytesUnsafe(val string) []byte {
	type sliceHeader struct {
		_   uintptr
		len int
		cap int
	}
	slice := *(*sliceHeader)(unsafe.Pointer(&val))
	slice.cap = slice.len
	return *(*[]byte)(unsafe.Pointer(&slice))
}

func isScannableRtype(typ r.Type) bool {
	typ = typeDeref(typ)
	return typ != nil && (typ == typeTime || r.PtrTo(typ).Implements(sqlScannerRtype))
}

// WTB more specific name.
func isStructType(typ r.Type) bool {
	return typ != nil && typ.Kind() == r.Struct && !isScannableRtype(typ)
}

func maybeAppendSpace(val []byte) []byte {
	if hasDelimSuffix(bytesToMutableString(val)) {
		return val
	}
	return append(val, ` `...)
}

func appendMaybeSpaced(text []byte, suffix string) []byte {
	if !hasDelimSuffix(bytesToMutableString(text)) && !hasDelimPrefix(suffix) {
		text = append(text, ` `...)
	}
	text = append(text, suffix...)
	return text
}

func hasDelimPrefix(text string) bool {
	return len(text) == 0 || charsetDelimEnd.has(text[0])
}

func hasDelimSuffix(text string) bool {
	return len(text) == 0 || charsetDelimStart.has(text[len(text)-1])
}

var ordReg = regexp.MustCompile(
	`^\s*((?:\w+\.)*\w+)(?i)(?:\s+(asc|desc))?(?:\s+nulls\s+(first|last))?\s*$`,
)

func try(err error) {
	if err != nil {
		panic(err)
	}
}

func try1[A any](val A, err error) A {
	try(err)
	return val
}

// Must be deferred.
func rec(ptr *error) {
	val := recover()
	if val == nil {
		return
	}

	err, _ := val.(error)
	if err != nil {
		*ptr = err
		return
	}

	panic(val)
}

/*
Questionable. Could be avoided by using `is [not] distinct from` which works for
both nulls and non-nulls, but at the time of writing, that operator doesn't
work on indexes in PG, resulting in atrocious performance.
*/
func norm(val any) any {
	val = normNil(val)
	if val == nil {
		return nil
	}

	nullable, _ := val.(Nullable)
	if nullable != nil {
		if nullable.IsNull() {
			return nil
		}
		return val
	}

	valuer, _ := val.(driver.Valuer)
	if valuer != nil {
		return try1(valuer.Value())
	}

	return val
}

func normNil(val any) any {
	if isNil(val) {
		return nil
	}
	return val
}

func counter(val int) []struct{} { return make([]struct{}, val) }

// Generics when?
func resliceStrings(val *[]string, length int) { *val = (*val)[:length] }

// Generics when?
func resliceInts(val *[]int, length int) { *val = (*val)[:length] }

// Generics when?
func copyStrings(val []string) []string {
	if val == nil {
		return nil
	}
	out := make([]string, len(val))
	copy(out, val)
	return out
}

// Generics when?
func copyInts(src []int) []int {
	if src == nil {
		return nil
	}
	out := make([]int, len(src))
	copy(out, src)
	return out
}

func trimPrefixByte(val string, prefix byte) (string, error) {
	if !(len(val) >= byteLen && val[0] == prefix) {
		return ``, errf(`expected %q to begin with %q`, val, rune(prefix))
	}
	return val[byteLen:], nil
}

func exprAppend[A Expr](expr A, text []byte) []byte {
	text, _ = expr.AppendExpr(text, nil)
	return text
}

func exprString[A Expr](expr A) string {
	return bytesToMutableString(exprAppend(expr, nil))
}

// Copied from `github.com/mitranim/gax` and tested there.
func growBytes(prev []byte, size int) []byte {
	len, cap := len(prev), cap(prev)
	if cap-len >= size {
		return prev
	}

	next := make([]byte, len, 2*cap+size)
	copy(next, prev)
	return next
}

// Same as `growBytes`. WTB generics.
func growInterfaces(prev []any, size int) []any {
	len, cap := len(prev), cap(prev)
	if cap-len >= size {
		return prev
	}

	next := make([]any, len, 2*cap+size)
	copy(next, prev)
	return next
}

// Same as `growBytes`. WTB generics.
func growExprs(prev []Expr, size int) []Expr {
	len, cap := len(prev), cap(prev)
	if cap-len >= size {
		return prev
	}

	next := make([]Expr, len, 2*cap+size)
	copy(next, prev)
	return next
}

var argTrackerPool = sync.Pool{New: newArgTracker}

func newArgTracker() any { return new(argTracker) }

func getArgTracker() *argTracker {
	return argTrackerPool.Get().(*argTracker)
}

/**
Should be pooled via `sync.Pool`. All fields should be allocated lazily on
demand, but only once. Pre-binding the methods is a one-time expense which
allows to avoid repeated allocs that would be caused by passing any
key-validating functions to the "ranger" interfaces. Because "range" methods
are dynamically-dispatched, Go can't perform escape analysis, and must assume
that any inputs will escape.
*/
type argTracker struct {
	Ordinal         map[OrdinalParam]OrdinalParam
	Named           map[NamedParam]OrdinalParam
	ValidateOrdinal func(int)
	ValidateNamed   func(string)
}

func (self *argTracker) GotOrdinal(key OrdinalParam) (OrdinalParam, bool) {
	val, ok := self.Ordinal[key]
	return val, ok
}

func (self *argTracker) GotNamed(key NamedParam) (OrdinalParam, bool) {
	val, ok := self.Named[key]
	return val, ok
}

func (self *argTracker) SetOrdinal(key OrdinalParam, val OrdinalParam) {
	if self.Ordinal == nil {
		self.Ordinal = make(map[OrdinalParam]OrdinalParam, 16)
	}
	self.Ordinal[key] = val
}

func (self *argTracker) SetNamed(key NamedParam, val OrdinalParam) {
	if self.Named == nil {
		self.Named = make(map[NamedParam]OrdinalParam, 16)
	}
	self.Named[key] = val
}

func (self *argTracker) validateOrdinal(key int) {
	param := OrdinalParam(key).FromIndex()
	_, ok := self.Ordinal[param]
	if !ok {
		panic(errUnusedOrdinal(param))
	}
}

func (self *argTracker) validateNamed(key string) {
	param := NamedParam(key)
	_, ok := self.Named[param]
	if !ok {
		panic(errUnusedNamed(param))
	}
}

func (self *argTracker) validate(dict ArgDict) {
	impl0, _ := dict.(OrdinalRanger)
	if impl0 != nil {
		if self.ValidateOrdinal == nil {
			self.ValidateOrdinal = self.validateOrdinal
		}
		impl0.RangeOrdinal(self.ValidateOrdinal)
	}

	impl1, _ := dict.(NamedRanger)
	if impl1 != nil {
		if self.ValidateNamed == nil {
			self.ValidateNamed = self.validateNamed
		}
		impl1.RangeNamed(self.ValidateNamed)
	}
}

func (self *argTracker) put() {
	for key := range self.Ordinal {
		delete(self.Ordinal, key)
	}
	for key := range self.Named {
		delete(self.Named, key)
	}
	argTrackerPool.Put(self)
}

func strDir(val string) Dir {
	if strings.EqualFold(val, `asc`) {
		return DirAsc
	}
	if strings.EqualFold(val, `desc`) {
		return DirDesc
	}
	return DirNone
}

func strNulls(val string) Nulls {
	if strings.EqualFold(val, `first`) {
		return NullsFirst
	}
	if strings.EqualFold(val, `last`) {
		return NullsLast
	}
	return NullsNone
}

func countNonEmptyStrings(vals []string) (count int) {
	for _, val := range vals {
		if val != `` {
			count++
		}
	}
	return
}

func validateIdent(val string) {
	if strings.ContainsRune(val, quoteDouble) {
		panic(ErrInvalidInput{Err{
			`encoding ident`,
			errf(`unexpected %q in SQL identifier %q`, rune(quoteDouble), val),
		}})
	}
}

var prepCache = cacheOf(func(src string) Prep {
	prep := Prep{Source: src}
	prep.Parse()
	return prep
})

var colsCache = cacheOf(func(typ r.Type) string {
	typ = typeElem(typ)
	if isStructType(typ) {
		return structCols(typ)
	}
	return `*`
})

var colsDeepCache = cacheOf(func(typ r.Type) string {
	typ = typeElem(typ)
	if isStructType(typ) {
		return structColsDeep(typ)
	}
	return `*`
})

func loadStructDbFields(typ r.Type) []r.StructField {
	return structDbFieldsCache.Get(typeElem(typ))
}

var structDbFieldsCache = cacheOf(func(typ r.Type) []r.StructField {
	// No `make` because `typ.NumField()` doesn't give us the full count.
	var out []r.StructField

	typ = typeElem(typ)
	if typ == nil {
		return out
	}

	reqStructType(`scanning DB-related struct fields`, typ)

	path := make([]int, 0, expectedStructNestingDepth)
	for ind := range counter(typ.NumField()) {
		appendStructDbFields(&out, &path, typ, ind)
	}

	return out
})

func loadStructPaths(typ r.Type) []structPath {
	return structPathsCache.Get(typeElem(typ))
}

var structPathsCache = cacheOf(func(typ r.Type) []structPath {
	var out []structPath

	typ = typeElem(typ)
	if typ == nil {
		return out
	}

	reqStructType(`scanning struct field and method paths`, typ)

	path := make([]int, 0, expectedStructNestingDepth)
	for ind := range counter(typ.NumField()) {
		appendStructFieldPaths(&out, &path, typ, ind)
	}

	for ind := range counter(typ.NumMethod()) {
		meth := typ.Method(ind)
		if isPublic(meth.PkgPath) {
			out = append(out, structPath{Name: meth.Name, MethodIndex: ind})
		}
	}

	return out
})

func loadStructPathMap(typ r.Type) map[string]structPath {
	return structPathMapCache.Get(typeElem(typ))
}

var structPathMapCache = cacheOf(func(typ r.Type) map[string]structPath {
	paths := loadStructPaths(typ)
	out := make(map[string]structPath, len(paths))
	for _, val := range paths {
		out[val.Name] = val
	}
	return out
})

func loadStructJsonPathToNestedDbFieldMap(typ r.Type) map[string]structNestedDbField {
	return structJsonPathToNestedDbFieldMapCache.Get(typeElem(typ))
}

var structJsonPathToNestedDbFieldMapCache = cacheOf(func(typ r.Type) map[string]structNestedDbField {
	typ = typeElem(typ)
	if typ == nil {
		return nil
	}

	reqStructType(`generating JSON-DB path mapping from struct type`, typ)

	buf := map[string]structNestedDbField{}
	jsonPath := make([]string, 0, expectedStructNestingDepth)
	dbPath := make([]string, 0, expectedStructNestingDepth)

	for ind := range counter(typ.NumField()) {
		addJsonPathsToDbPaths(buf, &jsonPath, &dbPath, typ.Field(ind))
	}
	return buf
})

func loadStructJsonPathToDbPathFieldValueMap(typ r.Type) map[string]structFieldValue {
	return structJsonPathToDbPathFieldValueMapCache.Get(typeElem(typ))
}

var structJsonPathToDbPathFieldValueMapCache = cacheOf(func(typ r.Type) map[string]structFieldValue {
	src := loadStructJsonPathToNestedDbFieldMap(typ)
	out := make(map[string]structFieldValue, len(src))
	for key, val := range src {
		out[key] = structFieldValue{val.Field, r.ValueOf(val.DbPath)}
	}
	return out
})

func appendStructDbFields(buf *[]r.StructField, path *[]int, typ r.Type, index int) {
	field := typ.Field(index)
	if !isPublic(field.PkgPath) {
		return
	}

	defer resliceInts(path, len(*path))
	*path = append(*path, index)

	tag, ok := field.Tag.Lookup(TagNameDb)
	if ok {
		if tagIdent(tag) != `` {
			field.Index = copyInts(*path)
			*buf = append(*buf, field)
		}
		return
	}

	typ = typeDeref(field.Type)
	if field.Anonymous && typ.Kind() == r.Struct {
		for ind := range counter(typ.NumField()) {
			appendStructDbFields(buf, path, typ, ind)
		}
	}
}

func appendStructFieldPaths(buf *[]structPath, path *[]int, typ r.Type, index int) {
	field := typ.Field(index)
	if !isPublic(field.PkgPath) {
		return
	}

	defer resliceInts(path, len(*path))
	*path = append(*path, index)
	*buf = append(*buf, structPath{Name: field.Name, FieldIndex: copyInts(*path)})

	typ = typeDeref(field.Type)
	if field.Anonymous && typ.Kind() == r.Struct {
		for ind := range counter(typ.NumField()) {
			appendStructFieldPaths(buf, path, typ, ind)
		}
	}
}

func makeIter(val any) (out iter) {
	out.init(val)
	return
}

/*
Allows clearer code. Seems to incur no measurable overhead compared to
equivalent inline code. However, be aware that converting a stack-allocated
source value to `any` tends to involve copying.
*/
type iter struct {
	field r.StructField
	value r.Value
	index int
	count int

	root   r.Value
	fields []r.StructField
	filter Filter
}

func (self *iter) init(src any) {
	if src == nil {
		return
	}

	sparse, _ := src.(Sparse)
	if sparse != nil {
		self.root = valueOf(sparse.Get())
		self.filter = sparse
	} else {
		self.root = valueOf(src)
	}

	if self.root.IsValid() {
		self.fields = loadStructDbFields(self.root.Type())
	}
}

//nolint:unused
func (self *iter) reinit() {
	self.index = 0
	self.count = 0
}

func (self *iter) next() bool {
	fil := self.filter

	for self.index < len(self.fields) {
		field := self.fields[self.index]

		if fil != nil && !fil.AllowField(field) {
			self.index++
			continue
		}

		self.field = field
		self.value = self.root.FieldByIndex(field.Index)
		self.count++
		self.index++
		return true
	}

	return false
}

func (self *iter) empty() bool { return self.count == 0 }
func (self *iter) first() bool { return self.count == 1 }

/*
Returns true if the iterator would visit at least one field/value, otherwise
returns false. Requires `.init`.
*/
func (self *iter) has() bool {
	fil := self.filter

	if fil == nil {
		return len(self.fields) > 0
	}

	for _, field := range self.fields {
		if fil.AllowField(field) {
			return true
		}
	}
	return false
}

func typeElem(typ r.Type) r.Type {
	for typ != nil && (typ.Kind() == r.Ptr || typ.Kind() == r.Slice) {
		typ = typ.Elem()
	}
	return typ
}

func valueDeref(val r.Value) r.Value {
	for val.Kind() == r.Ptr {
		if val.IsNil() {
			return r.Value{}
		}
		val = val.Elem()
	}
	return val
}

func typeElemOf(typ any) r.Type {
	return typeElem(r.TypeOf(typ))
}

func typeOf(typ any) r.Type {
	return typeDeref(r.TypeOf(typ))
}

func valueOf(val any) r.Value {
	return valueDeref(r.ValueOf(val))
}

func kindOf(val any) r.Kind {
	typ := typeOf(val)
	if typ != nil {
		return typ.Kind()
	}
	return r.Invalid
}

func isStructTypeEmpty(typ r.Type) bool {
	typ = typeDeref(typ)
	return typ == nil || typ.Kind() != r.Struct || typ.NumField() == 0
}

func reqGetter(val, method r.Type, name string) {
	inputs := method.NumIn()
	if inputs != 0 {
		panic(ErrInternal{Err{
			`evaluating method`,
			errf(
				`can't evaluate %q of %v: expected 0 parameters, found %v parameters`,
				name, val, inputs,
			),
		}})
	}

	outputs := method.NumOut()
	if outputs != 1 {
		panic(ErrInternal{Err{
			`evaluating method`,
			errf(
				`can't evaluate %q of %v: expected 1 return parameter, found %v return parameters`,
				name, val, outputs,
			),
		}})
	}
}

func reqStructType(while string, typ r.Type) {
	if typ.Kind() != r.Struct {
		panic(errExpectedStruct(while, typ.Name()))
	}
}

func typeName(typ r.Type) string {
	typ = typeDeref(typ)
	if typ == nil {
		return `nil`
	}
	return typ.Name()
}

func typeNameOf[A any](val A) string { return typeName(r.TypeOf(val)) }

func isNil(val any) bool {
	return val == nil || isValueNil(r.ValueOf(val))
}

func isValueNil(val r.Value) bool {
	return !val.IsValid() || isNilable(val.Kind()) && val.IsNil()
}

func isNilable(kind r.Kind) bool {
	switch kind {
	case r.Chan, r.Func, r.Interface, r.Map, r.Ptr, r.Slice:
		return true
	default:
		return false
	}
}

func isPublic(pkgPath string) bool { return pkgPath == `` }

func typeDeref(typ r.Type) r.Type {
	if typ == nil {
		return nil
	}
	return typeDerefCache.Get(typ)
}

var typeDerefCache = cacheOf(func(typ r.Type) r.Type {
	for typ != nil {
		if typ.Kind() == r.Ptr {
			typ = typ.Elem()
			continue
		}

		if typ.Kind() == r.Struct && typ.NumField() > 0 {
			field := typ.Field(0)
			if field.Tag.Get(`role`) == `ref` {
				typ = field.Type
				continue
			}
		}

		break
	}

	return typ
})

/*
TODO: consider validating that the name doesn't contain double quotes. We might
return an error, or panic.
*/
func tagIdent(tag string) string {
	index := strings.IndexRune(tag, ',')
	if index >= 0 {
		return tagIdent(tag[:index])
	}
	if tag == "-" {
		return ""
	}
	return tag
}

func structCols(typ r.Type) string {
	reqStructType(`generating struct columns string from struct type`, typ)

	var buf []byte
	for ind, field := range loadStructDbFields(typ) {
		if ind > 0 {
			buf = append(buf, `, `...)
		}
		buf = Ident(FieldDbName(field)).AppendTo(buf)
	}
	return bytesToMutableString(buf)
}

func structColsDeep(typ r.Type) string {
	reqStructType(`generating deep struct columns string from struct type`, typ)

	var buf []byte
	var path []string

	for ind := range counter(typ.NumField()) {
		appendFieldCols(&buf, &path, typ.Field(ind))
	}
	return bytesToMutableString(buf)
}

func appendFieldCols(buf *[]byte, path *[]string, field r.StructField) {
	if !isPublic(field.PkgPath) {
		return
	}

	typ := typeDeref(field.Type)
	tag, ok := field.Tag.Lookup(TagNameDb)
	dbName := tagIdent(tag)

	if dbName == `` {
		if !ok {
			if field.Anonymous && typ.Kind() == r.Struct {
				for ind := range counter(typ.NumField()) {
					appendFieldCols(buf, path, typ.Field(ind))
				}
			}
		}
		return
	}

	defer resliceStrings(path, len(*path))
	*path = append(*path, dbName)

	if isStructType(typ) {
		for ind := range counter(typ.NumField()) {
			appendFieldCols(buf, path, typ.Field(ind))
		}
		return
	}

	text := *buf

	if len(text) > 0 {
		text = append(text, `, `...)
	}
	text = AliasedPath(*path).AppendTo(text)

	*buf = text
}

func addJsonPathsToDbPaths(
	buf map[string]structNestedDbField, jsonPath *[]string, dbPath *[]string, field r.StructField,
) {
	if !isPublic(field.PkgPath) {
		return
	}

	typ := typeDeref(field.Type)
	jsonName := FieldJsonName(field)
	tag, ok := field.Tag.Lookup(TagNameDb)
	dbName := tagIdent(tag)

	if dbName == `` {
		if !ok {
			if field.Anonymous && typ.Kind() == r.Struct {
				for ind := range counter(typ.NumField()) {
					addJsonPathsToDbPaths(buf, jsonPath, dbPath, typ.Field(ind))
				}
			}
		}
		return
	}

	defer resliceStrings(jsonPath, len(*jsonPath))
	*jsonPath = append(*jsonPath, jsonName)

	defer resliceStrings(dbPath, len(*dbPath))
	*dbPath = append(*dbPath, dbName)

	buf[strings.Join(*jsonPath, `.`)] = structNestedDbField{
		Field:  field,
		DbPath: copyStrings(*dbPath),
	}

	if isStructType(typ) {
		for ind := range counter(typ.NumField()) {
			addJsonPathsToDbPaths(buf, jsonPath, dbPath, typ.Field(ind))
		}
	}
}

func trimWhitespaceAndComments(val Token) Token {
	switch val.Type {
	case TokenTypeWhitespace:
		return Token{` `, TokenTypeWhitespace}
	case TokenTypeCommentLine, TokenTypeCommentBlock:
		return Token{}
	default:
		return val
	}
}

func isJsonDict(val []byte) bool   { return headByte(val) == '{' }
func isJsonList(val []byte) bool   { return headByte(val) == '[' }
func isJsonString(val []byte) bool { return headByte(val) == '"' }

func headByte(val []byte) byte {
	if len(val) > 0 {
		return val[0]
	}
	return 0
}

func appendIntWith(text []byte, delim string, val int64) []byte {
	if val == 0 {
		return text
	}

	text = maybeAppendSpace(text)
	text = append(text, delim...)
	text = maybeAppendSpace(text)
	text = strconv.AppendInt(text, val, 10)
	return text
}

func appendPrefixSub(
	text []byte, args []any, prefix string, val any,
) (
	[]byte, []any,
) {
	if val == nil {
		return text, args
	}

	bui := Bui{text, args}
	bui.Str(prefix)
	bui.SubAny(val)
	return bui.Get()
}

// Borrowed from the standard	library.
func noescape(src unsafe.Pointer) unsafe.Pointer {
	out := uintptr(src)
	// nolint:staticcheck
	return unsafe.Pointer(out ^ 0)
}

type formatState []byte

var _ = fmt.Stringer(formatState(nil))

func (self formatState) String() string { return bytesToMutableString(self) }

var _ = fmt.State((*formatState)(nil))

func (self *formatState) Write(src []byte) (int, error) {
	*self = append(*self, src...)
	return len(src), nil
}

func (self *formatState) Width() (int, bool)     { return 0, false }
func (self *formatState) Precision() (int, bool) { return 0, false }
func (self *formatState) Flag(int) bool          { return false }
