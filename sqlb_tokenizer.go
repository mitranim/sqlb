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

	* Correctly parse whitespace, comments, quoted content, ordinal parameters,
	  named parameters.

	* Decently fast and allocation-free tokenization.

Non-goals:

	* Full SQL parser.
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
		self.char()
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
			fmt.Errorf(
				`internal error: attempted to overwrite non-empty pending token %#v with %#v`,
				self.next, val,
			),
		}})
	}
	self.next = val
}

func (self *Tokenizer) maybeWhitespace() {
	for self.more() && charsetWhitespace.has(self.headByte()) {
		self.scan(1)
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
	if !self.scannedString(commentLinePrefix) {
		return
	}
	for self.more() && !self.scannedNewline() && self.scannedChar() {
	}
}

func (self *Tokenizer) maybeCommentBlock() {
	self.maybeStringBetween(commentBlockPrefix, commentBlockSuffix)
}

func (self *Tokenizer) maybeDoubleColon() {
	self.maybeString(doubleColonPrefix)
}

func (self *Tokenizer) maybeOrdinalParam() {
	start := self.cursor
	if !self.scannedByte(ordinalParamPrefix) {
		return
	}
	if !self.scannedDigits() {
		self.cursor = start
	}
}

func (self *Tokenizer) maybeNamedParam() {
	start := self.cursor
	if !self.scannedByte(namedParamPrefix) {
		return
	}
	if !self.scannedIdent() {
		self.cursor = start
	}
}

func (self *Tokenizer) maybeString(val string) {
	_ = self.scannedString(val)
}

func (self *Tokenizer) scannedNewline() bool {
	start := self.cursor
	self.maybeNewline()
	return self.cursor > start
}

func (self *Tokenizer) maybeNewline() {
	self.scan(leadingNewlineSize(self.rest()))
}

func (self *Tokenizer) scannedChar() bool {
	start := self.cursor
	self.char()
	return self.cursor > start
}

func (self *Tokenizer) char() {
	_, size := utf8.DecodeRuneInString(self.rest())
	self.scan(size)
}

func (self *Tokenizer) scannedDigits() bool {
	start := self.cursor
	self.maybeDigits()
	return self.cursor > start
}

func (self *Tokenizer) maybeDigits() {
	for self.more() && charsetDigitDec.has(self.headByte()) {
		self.scan(1)
	}
}

func (self *Tokenizer) scannedIdent() bool {
	start := self.cursor
	self.maybeIdent()
	return self.cursor > start
}

func (self *Tokenizer) maybeIdent() {
	if !self.scannedByteIn(charsetIdentStart) {
		return
	}
	for self.more() && self.scannedByteIn(charsetIdent) {
	}
}

func (self *Tokenizer) maybeStringBetween(prefix, suffix string) {
	if !self.scannedString(prefix) {
		return
	}

	for self.more() {
		if self.scannedString(suffix) {
			return
		}
		self.char()
	}

	panic(ErrUnexpectedEOF{Err{
		`parsing SQL`,
		fmt.Errorf(`expected closing %q, got unexpected %w`, suffix, io.EOF),
	}})
}

func (self *Tokenizer) maybeStringBetweenBytes(prefix, suffix byte) {
	if !self.scannedByte(prefix) {
		return
	}

	for self.more() {
		if self.scannedByte(suffix) {
			return
		}
		self.char()
	}

	panic(ErrUnexpectedEOF{Err{
		`parsing SQL`,
		fmt.Errorf(`expected closing %q, got unexpected %w`, rune(suffix), io.EOF),
	}})
}

func (self *Tokenizer) scan(val int) {
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

func (self *Tokenizer) scannedByte(val byte) bool {
	if self.headByte() == val {
		self.scan(1)
		return true
	}
	return false
}

func (self *Tokenizer) scannedByteIn(val *charset) bool {
	if val.has(self.headByte()) {
		self.scan(1)
		return true
	}
	return false
}

func (self *Tokenizer) scannedString(val string) bool {
	if strings.HasPrefix(self.rest(), val) {
		self.scan(len(val))
		return true
	}
	return false
}

// Part of `Token`.
type TokenType byte

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

// Represents an arbitrary chunk of SQL text parsed by `Tokenizer`.
type Token struct {
	Text string
	Type TokenType
}

// True if the token's type is `TokenTypeInvalid`. This is used to detect end of
// iteration when calling `(*Tokenizer).Next`.
func (self Token) IsInvalid() bool {
	return self.Type == TokenTypeInvalid
}

// Implement `fmt.Stringer` for debug purposes.
func (self Token) String() string { return self.Text }

// Assumes that the token has `TokenTypeOrdinalParam` and looks like a
// Postgres-style ordinal param: "$1", "$2" and so on. Parses and returns the
// number. Panics if the text had the wrong structure.
func (self Token) ParseOrdinalParam() OrdinalParam {
	rest, err := trimPrefixByte(self.Text, ordinalParamPrefix)
	try(errOrdinal(err))

	val, err := strconv.Atoi(rest)
	try(errOrdinal(err))

	return OrdinalParam(val)
}

// Assumes that the token has `TokenTypeNamedParam` and looks like a
// Postgres-style named param: ":one", ":two" and so on. Parses and returns the
// parameter's name without the leading ":". Panics if the text had the wrong
// structure.
func (self Token) ParseNamedParam() NamedParam {
	rest, err := trimPrefixByte(self.Text, namedParamPrefix)
	try(errNamed(err))
	return NamedParam(rest)
}