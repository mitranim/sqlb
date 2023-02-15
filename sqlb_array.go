package sqlb

import "database/sql/driver"

/*
Intermediary tool for implementing SQL array encoding. Has the same behavior as
`CommaAppender`, but the text output is always enclosed in `{}`.
*/
type ArrayAppender[A Appender] []A

/*
Implement `fmt.Stringer`. Same as `CommaAppender.String`, but the output is
always enclosed in `{}`.
*/
func (self ArrayAppender[_]) String() string { return AppenderString(&self) }

/*
Implement `Appender`. Same as `CommaAppender.Append`, but the output is always
enclosed in `{}`.
*/
func (self ArrayAppender[A]) Append(buf []byte) []byte {
	buf = append(buf, `{`...)
	buf = CommaAppender[A](self).Append(buf)
	buf = append(buf, `}`...)
	return buf
}

func (self ArrayAppender[_]) Get() any { return self.String() }

func (self ArrayAppender[_]) Value() (driver.Value, error) { return self.Get(), nil }

/*
Intermediary tool for implementing SQL array encoding. Combines multiple
arbitrary text encoders. On demand (on a call to `.Append` or `.String`),
combines their text representations, separating them with a comma, while
skipping any empty representations. The output will never contain a dangling
leading comma, double comma, or leading trailing comma, unless they were
explicitly generated by the inner encoders. Compare `SliceCommaAppender`
which takes an arbitrary slice.
*/
type CommaAppender[A Appender] []A

// Implement `fmt.Stringer` by calling `.Append`.
func (self CommaAppender[_]) String() string { return AppenderString(&self) }

/*
Implement `Appender`. Appends comma-separated text representations of the inner
encoders to the output buffer, skipping any empty representations.
*/
func (self CommaAppender[_]) Append(buf []byte) []byte {
	var found bool

	for _, val := range self {
		if (found && TryAppendWith(&buf, `,`, val)) || TryAppendWith(&buf, ``, val) {
			found = true
		}
	}

	return buf
}

/*
Intermediary tool for implementing SQL array encoding. The inner value must be
either nil, a slice/array, or a pointer to a slice/array, where each element
must implement `Appender`. When `.Append` or `.String` is called, this combines
the text representations of the elements, separating them with a comma, while
skipping any empty representations. The output will never contain a dangling
leading comma, double comma, or leading trailing comma, unless they were
explicitly generated by the inner encoders. Compare `CommaAppender` which
itself is a slice.
*/
type SliceCommaAppender [1]any

// Implement `fmt.Stringer` by calling `.Append`.
func (self SliceCommaAppender) String() string { return AppenderString(&self) }

/*
Implement `Appender`. Appends comma-separated text representations of the inner
encoders to the output buffer, skipping any empty representations.
*/
func (self SliceCommaAppender) Append(buf []byte) []byte {
	if self[0] == nil {
		return buf
	}

	val, _ := self[0].(Appender)
	if val != nil {
		return val.Append(buf)
	}

	src := valueOf(self[0])
	if !src.IsValid() {
		return buf
	}

	var found bool
	for ind := range counter(src.Len()) {
		elem := src.Index(ind)
		if !elem.IsValid() {
			continue
		}

		iface := elem.Interface()
		if iface == nil {
			continue
		}

		val := iface.(Appender)
		if (found && TryAppendWith(&buf, `,`, val)) || TryAppendWith(&buf, ``, val) {
			found = true
		}
	}

	return buf
}
