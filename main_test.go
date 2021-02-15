package sqlb

import (
	"reflect"
	"testing"
)

type (
	B  = testing.B
	T  = testing.T
	TB = testing.TB
)

func eq(t TB, expected interface{}, actual interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected:\n%#v\nactual:\n%#v", expected, actual)
	}
}
