package sqlb_test

import (
	"fmt"

	s "github.com/mitranim/sqlb"
)

func ExampleBui() {
	fmt.Println(s.Reify(SomeExpr{}))
	// Output:
	// select $1 [some_value]
}

type SomeExpr struct{}

func (self SomeExpr) AppendExpr(text []byte, args []interface{}) ([]byte, []interface{}) {
	bui := s.Bui{text, args}
	bui.Str(`select`)
	bui.Any(`some_value`)
	return bui.Get()
}
