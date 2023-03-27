package sqlb

/*
Prealloc tool. Makes a `Bui` with the specified capacity of the text and args
buffers.
*/
func MakeBui(textCap, argsCap int) Bui {
	return Bui{
		make([]byte, 0, textCap),
		make([]any, 0, argsCap),
	}
}

/*
Short for "builder". Tiny shortcut for building SQL expressions. Significantly
simplifies the code and avoids various common mistakes. Used internally by most
`Expr` implementations in this package. Careful use of `Bui` incurs very litte
overhead compared to writing the corresponding code inline. The design may
allow future Go versions to optimize it away completely.
*/
type Bui struct {
	Text []byte
	Args []any
}

// Returns text and args as-is. Useful shortcut for passing them to
// `AppendExpr`.
func (self Bui) Get() ([]byte, []any) {
	return self.Text, self.Args
}

/*
Replaces text and args with the inputs. The following idiom is equivalent to
`bui.Expr` but more efficient if the expression type is concrete, avoiding an
interface-induced allocation:

	bui.Set(SomeExpr{}.AppendExpr(bui.Get()))
*/
func (self *Bui) Set(text []byte, args []any) {
	self.Text = text
	self.Args = args
}

// Shortcut for `self.String(), self.Args`. Go database drivers tend to require
// `string, []any` as inputs for queries and statements.
func (self Bui) Reify() (string, []any) {
	return self.String(), self.Args
}

// Returns inner text as a string, performing a free cast.
func (self Bui) String() string {
	return bytesToMutableString(self.Text)
}

// Increases the capacity (not length) of the text and args buffers by the
// specified amounts. If there's already enough capacity, avoids allocation.
func (self *Bui) Grow(textLen, argsLen int) {
	self.Text = growBytes(self.Text, textLen)
	self.Args = growInterfaces(self.Args, argsLen)
}

// Adds a space if the preceding text doesn't already end with a terminator.
func (self *Bui) Space() {
	self.Text = maybeAppendSpace(self.Text)
}

// Appends the provided string, delimiting it from the previous text with a
// space if necessary.
func (self *Bui) Str(val string) {
	self.Text = appendMaybeSpaced(self.Text, val)
}

/*
Appends an expression, delimited from the preceding text by a space, if
necessary. Nil input is a nop: nothing will be appended.

Should be used only if you already have an `Expr` value. If you have a concrete
value that implements the interface, call `bui.Set(val.AppendExpr(bui.Get())`
instead, to avoid a heap allocation and a minor slowdown.
*/
func (self *Bui) Expr(val Expr) {
	if val != nil {
		self.Space()
		self.Set(val.AppendExpr(self.Get()))
	}
}

/*
Appends a sub-expression wrapped in parens. Nil input is a nop: nothing will be
appended.

Performance note: if you have a concrete value rather than an `Expr`, calling
this method will allocate, so you may want to avoid it. If you already have an
`Expr`, calling this is fine.
*/
func (self *Bui) SubExpr(val Expr) {
	if val != nil {
		self.Str(`(`)
		self.Expr(val)
		self.Str(`)`)
	}
}

// Appends each expr by calling `(*Bui).Expr`. They will be space-separated as
// necessary.
func (self *Bui) Exprs(vals ...Expr) {
	for _, val := range vals {
		self.Expr(val)
	}
}

// Same as `(*Bui).Exprs` but catches panics. Since many functions in this
// package use panics, this should be used for final reification by apps that
// insist on errors-as-values.
func (self *Bui) CatchExprs(vals ...Expr) (err error) {
	defer rec(&err)
	self.Exprs(vals...)
	return
}

/*
Appends an ordinal parameter such as "$1", space-separated from previous text if
necessary. Requires caution: does not verify the existence of the corresponding
argument.
*/
func (self *Bui) OrphanParam(val OrdinalParam) {
	self.Space()
	self.Text = val.AppendTo(self.Text)
}

/*
Appends an arg to the inner slice of args, returning the corresponding ordinal
parameter that should be appended to the text. Requires caution: does not
append the corresponding ordinal parameter.
*/
func (self *Bui) OrphanArg(val any) OrdinalParam {
	self.Args = append(self.Args, val)
	return OrdinalParam(len(self.Args))
}

/*
Appends an argument to `.Args` and a corresponding ordinal parameter to
`.Text`.
*/
func (self *Bui) Arg(val any) { self.OrphanParam(self.OrphanArg(val)) }

/*
Appends an arbitrary value. If the value implements `Expr`, this calls
`(*Bui).Expr`, which may append to the text and args in arbitrary ways.
Otherwise, appends an argument to the inner slice of args, and the
corresponding ordinal parameter such as "$1"/"$2"/.../"$N" to the text.
*/
func (self *Bui) Any(val any) {
	impl, _ := val.(Expr)
	if impl != nil {
		self.Expr(impl)
		return
	}

	/**
	TODO consider the following:

		if val == nil {
			self.Str(`null`)
		} else {
			self.Arg(val)
		}

	Makes some assumptions, and might be the wrong place for such a special case.
	*/

	self.Arg(val)
}

/*
Appends an arbitrary value or sub-expression. Like `(*Bui).Any`, but if the
value implements `Expr`, this uses `(*Bui).SubExpr` in order to parenthesize
the sub-expression.
*/
func (self *Bui) SubAny(val any) {
	impl, _ := val.(Expr)
	if impl != nil {
		self.SubExpr(impl)
		return
	}
	self.Arg(val)
}
