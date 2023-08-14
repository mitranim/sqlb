package sqlb

import (
	"fmt"
	"io"
	"testing"
)

type FakeTracedErr string

func (self FakeTracedErr) Error() string { return string(self) }

func (self FakeTracedErr) Format(out fmt.State, _ rune) {
	try1(io.WriteString(out, self.Error()))

	if out.Flag('+') {
		if self != `` {
			try1(io.WriteString(out, `; `))
		}
		try1(io.WriteString(out, `fake stack trace`))
		return
	}
}

func Benchmark_errf(b *testing.B) {
	for ind := 0; ind < b.N; ind++ {
		_ = errf(`error %v`, `message`)
	}
}

func Benchmark_fmt_Errorf(b *testing.B) {
	for ind := 0; ind < b.N; ind++ {
		_ = fmt.Errorf(`error %v`, `message`)
	}
}

func TestErr_formatting(t *testing.T) {
	test := func(src Err, expBase, expPlus string) {
		eq(t, expBase, src.Error())
		eq(t, expBase, fmt.Sprintf(`%v`, src))
		eq(t, expPlus, fmt.Sprintf(`%+v`, src))
	}

	test(Err{}, ``, ``)

	test(
		Err{While: `doing some operation`},
		`[sqlb] error while doing some operation`,
		`[sqlb] error while doing some operation`,
	)

	test(
		Err{Cause: ErrStr(`some cause`)},
		`[sqlb] error: some cause`,
		`[sqlb] error: some cause`,
	)

	test(
		Err{
			While: `doing some operation`,
			Cause: ErrStr(`some cause`),
		},
		`[sqlb] error while doing some operation: some cause`,
		`[sqlb] error while doing some operation: some cause`,
	)

	test(
		Err{
			While: `doing some operation`,
			Cause: FakeTracedErr(`some cause`),
		},
		`[sqlb] error while doing some operation: some cause`,
		`[sqlb] error while doing some operation: some cause; fake stack trace`,
	)
}
