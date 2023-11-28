package sqlb

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

/*
Partial SQL tokenizer used internally by `(*Prep).Parse` to parse queries, in
particular to convert named parameters into other expressions.

Goals:

	* Correctly parse whitespace, comments, quoted strings and identifiers,
	  ordinal parameters, named parameters.

	* Decently fast and allocation-free tokenization.

Non-goals:

	* Full SQL parser.

Notable limitations:

	* No special support for dollar-quoted strings, which are rarely if ever used
	  in dynamically-generated queries.
*/
type Tokenizer struct {
	Source    string
	Transform func(Token) Token
	cursor    int
	next      Token
}

/*
Returns the next token if possible. When the tokenizer reaches the end, this
returns an empty `Token{}`. Call `Token.IsInvalid` to detect the end.
*/
func (self *Tokenizer) Next() Token {
	for {
		token := self.nextToken()
		if token.IsInvalid() {
			return Token{}
		}

		if self.Transform != nil {
			token = self.Transform(token)
			if token.IsInvalid() {
				continue
			}
		}

		return token
	}
}

func (self *Tokenizer) nextToken() Token {
	next := self.next
	if !next.IsInvalid() {
		self.next = Token{}
		return next
	}

	start := self.cursor

	for self.more() {
		mid := self.cursor
		if self.maybeWhitespace(); self.cursor > mid {
			return self.choose(start, mid, TokenTypeWhitespace)
		}
		if self.maybeQuotedSingle(); self.cursor > mid {
			return self.choose(start, mid, TokenTypeQuotedSingle)
		}
		if self.maybeQuotedDouble(); self.cursor > mid {
			return self.choose(start, mid, TokenTypeQuotedDouble)
		}
		if self.maybeQuotedGrave(); self.cursor > mid {
			return self.choose(start, mid, TokenTypeQuotedGrave)
		}
		if self.maybeCommentLine(); self.cursor > mid {
			return self.choose(start, mid, TokenTypeCommentLine)
		}
		if self.maybeCommentBlock(); self.cursor > mid {
			return self.choose(start, mid, TokenTypeCommentBlock)
		}
		if self.maybeDoubleColon(); self.cursor > mid {
			return self.choose(start, mid, TokenTypeDoubleColon)
		}
		if self.maybeOrdinalParam(); self.cursor > mid {
			return self.choose(start, mid, TokenTypeOrdinalParam)
		}
		if self.maybeNamedParam(); self.cursor > mid {
			return self.choose(start, mid, TokenTypeNamedParam)
		}
		self.skipChar()
	}

	if self.cursor > start {
		return Token{self.from(start), TokenTypeText}
	}
	return Token{}
}

func (self *Tokenizer) choose(start, mid int, typ TokenType) Token {
	tok := Token{self.from(mid), typ}
	if mid > start {
		self.setNext(tok)
		return Token{self.Source[start:mid], TokenTypeText}
	}
	return tok
}

func (self *Tokenizer) setNext(val Token) {
	if !self.next.IsInvalid() {
		panic(ErrInternal{Err{
			`parsing SQL`,
			errf(
				`internal error: attempted to overwrite non-empty pending token %#v with %#v`,
				self.next, val,
			),
		}})
	}
	self.next = val
}

func (self *Tokenizer) maybeWhitespace() {
	for self.more() && charsetWhitespace.has(self.headByte()) {
		self.skipBytes(1)
	}
}

func (self *Tokenizer) maybeQuotedSingle() {
	self.maybeStringBetweenBytes(quoteSingle, quoteSingle)
}

func (self *Tokenizer) maybeQuotedDouble() {
	self.maybeStringBetweenBytes(quoteDouble, quoteDouble)
}

func (self *Tokenizer) maybeQuotedGrave() {
	self.maybeStringBetweenBytes(quoteGrave, quoteGrave)
}

func (self *Tokenizer) maybeCommentLine() {
	if !self.skippedString(commentLinePrefix) {
		return
	}
	for self.more() && !self.skippedNewline() && self.skippedChar() {
	}
}

// TODO support nested block comments, which are valid in SQL.
func (self *Tokenizer) maybeCommentBlock() {
	self.maybeStringBetween(commentBlockPrefix, commentBlockSuffix)
}

func (self *Tokenizer) maybeDoubleColon() {
	self.maybeString(doubleColonPrefix)
}

func (self *Tokenizer) maybeOrdinalParam() {
	start := self.cursor
	if !self.skippedByte(ordinalParamPrefix) {
		return
	}
	if !self.skippedDigits() {
		self.cursor = start
	}
}

func (self *Tokenizer) maybeNamedParam() {
	start := self.cursor
	if !self.skippedByte(namedParamPrefix) {
		return
	}
	if !self.skippedIdent() {
		self.cursor = start
	}
}

func (self *Tokenizer) maybeString(val string) {
	_ = self.skippedString(val)
}

func (self *Tokenizer) skippedNewline() bool {
	start := self.cursor
	self.maybeNewline()
	return self.cursor > start
}

func (self *Tokenizer) maybeNewline() {
	self.skipBytes(leadingNewlineSize(self.rest()))
}

func (self *Tokenizer) skippedChar() bool {
	start := self.cursor
	self.skipChar()
	return self.cursor > start
}

func (self *Tokenizer) skipChar() {
	_, size := utf8.DecodeRuneInString(self.rest())
	self.skipBytes(size)
}

func (self *Tokenizer) skippedDigits() bool {
	start := self.cursor
	self.maybeSkipDigits()
	return self.cursor > start
}

func (self *Tokenizer) maybeSkipDigits() {
	for self.more() && charsetDigitDec.has(self.headByte()) {
		self.skipBytes(1)
	}
}

func (self *Tokenizer) skippedIdent() bool {
	start := self.cursor
	self.maybeIdent()
	return self.cursor > start
}

func (self *Tokenizer) maybeIdent() {
	if !self.skippedByteFromCharset(charsetIdentStart) {
		return
	}
	for self.more() && self.skippedByteFromCharset(charsetIdent) {
	}
}

func (self *Tokenizer) maybeStringBetween(prefix, suffix string) {
	if !self.skippedString(prefix) {
		return
	}

	for self.more() {
		if self.skippedString(suffix) {
			return
		}
		self.skipChar()
	}

	panic(ErrUnexpectedEOF{Err{
		`parsing SQL`,
		fmt.Errorf(`expected closing %q, got unexpected %w`, suffix, io.EOF),
	}})
}

func (self *Tokenizer) maybeStringBetweenBytes(prefix, suffix byte) {
	if !self.skippedByte(prefix) {
		return
	}

	for self.more() {
		if self.skippedByte(suffix) {
			return
		}
		self.skipChar()
	}

	panic(ErrUnexpectedEOF{Err{
		`parsing SQL`,
		fmt.Errorf(`expected closing %q, got unexpected %w`, rune(suffix), io.EOF),
	}})
}

func (self *Tokenizer) skipBytes(val int) {
	self.cursor += val
}

func (self *Tokenizer) more() bool {
	return self.cursor < len(self.Source)
}

func (self *Tokenizer) rest() string {
	return self.Source[self.cursor:]
}

func (self *Tokenizer) from(start int) string {
	return self.Source[start:self.cursor]
}

func (self *Tokenizer) headByte() byte {
	return self.Source[self.cursor]
}

func (self *Tokenizer) skippedByte(val byte) bool {
	if self.headByte() == val {
		self.skipBytes(1)
		return true
	}
	return false
}

func (self *Tokenizer) skippedByteFromCharset(val *charset) bool {
	if val.has(self.headByte()) {
		self.skipBytes(1)
		return true
	}
	return false
}

func (self *Tokenizer) skippedString(val string) bool {
	if strings.HasPrefix(self.rest(), val) {
		self.skipBytes(len(val))
		return true
	}
	return false
}

const (
	TokenTypeInvalid TokenType = iota
	TokenTypeText
	TokenTypeWhitespace
	TokenTypeQuotedSingle
	TokenTypeQuotedDouble
	TokenTypeQuotedGrave
	TokenTypeCommentLine
	TokenTypeCommentBlock
	TokenTypeDoubleColon
	TokenTypeOrdinalParam
	TokenTypeNamedParam
)

// Part of `Token`.
type TokenType byte

// Represents an arbitrary chunk of SQL text parsed by `Tokenizer`.
type Token struct {
	Text string
	Type TokenType
}

/*
True if the token's type is `TokenTypeInvalid`. This is used to detect end of
iteration when calling `(*Tokenizer).Next`.
*/
func (self Token) IsInvalid() bool {
	return self.Type == TokenTypeInvalid
}

// Implement `fmt.Stringer` for debug purposes.
func (self Token) String() string { return self.Text }

/*
Assumes that the token has `TokenTypeOrdinalParam` and looks like a
Postgres-style ordinal param: "$1", "$2" and so on. Parses and returns the
number. Panics if the text had the wrong structure.
*/
func (self Token) ParseOrdinalParam() OrdinalParam {
	rest, err := trimPrefixByte(self.Text, ordinalParamPrefix)
	try(errOrdinal(err))

	val, err := strconv.Atoi(rest)
	try(errOrdinal(err))

	return OrdinalParam(val)
}

/*
Assumes that the token has `TokenTypeNamedParam` and looks like a Postgres-style
named param: ":one", ":two" and so on. Parses and returns the parameter's name
without the leading ":". Panics if the text had the wrong structure.
*/
func (self Token) ParseNamedParam() NamedParam {
	rest, err := trimPrefixByte(self.Text, namedParamPrefix)
	try(errNamed(err))
	return NamedParam(rest)
}
