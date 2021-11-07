package sqlb

import "database/sql/driver"

/*
Intermediary tool for implementing SQL array encoding. Has the same behavior as
`CommaAppender`, but the text output is always enclosed in `{}`.
*/
type ArrayAppender []Appender

/*
Implement `fmt.Stringer`. Same as `CommaAppender.String`, but the output is
always enclosed in `{}`.
*/
func (self ArrayAppender) String() string { return AppenderString(&self) }

/*
Implement `Appender`. Same as `CommaAppender.Append`, but the output is always
enclosed in `{}`.
*/
func (self ArrayAppender) Append(buf []byte) []byte {
	buf = append(buf, `{`...)
	buf = CommaAppender(self).Append(buf)
	buf = append(buf, `}`...)
	return buf
}

func (self ArrayAppender) Get() interface{} { return self.String() }

func (self ArrayAppender) Value() (driver.Value, error) { return self.Get(), nil }

/*
Intermediary tool for implementing SQL array encoding. Combines multiple
arbitrary text encoders. On demand (on a call to `.Append` or `.String`),
combines their text representations, separating them with a comma, while
skipping any empty representations. The output will never contain a dangling
leading comma, double comma, or leading trailing comma, unless they were
explicitly generated by the inner encoders.
*/
type CommaAppender []Appender

// Implement `fmt.Stringer` by calling `.Append`.
func (self CommaAppender) String() string { return AppenderString(&self) }

/*
Implement `Appender`. Appends comma-separated text representations of the inner
encoders	to the output buffer, skipping any empty representations.
*/
func (self CommaAppender) Append(buf []byte) []byte {
	var found bool

	for _, val := range self {
		if (found && TryAppendWith(&buf, `,`, val)) || TryAppendWith(&buf, ``, val) {
			found = true
		}
	}

	return buf
}
