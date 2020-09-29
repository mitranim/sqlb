package sqlb

import (
	"reflect"
	"testing"

	"github.com/mitranim/sqlp"
)

type dict = map[string]interface{}

func TestNamedToOrdinal(t *testing.T) {
	nodes, err := sqlp.Parse(`one = :one and two = :two and three = :one`)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	args, err := namedToOrdinal(nodes, dict{"one": 10, "two": 20})
	if err != nil {
		t.Fatalf("%+v", err)
	}

	expectedStr := `one = $1 and two = $2 and three = $1`
	str := nodes.String()
	if expectedStr != str {
		t.Fatalf(`expected query %q, got %q`, expectedStr, str)
	}

	expectedArgs := []interface{}{10, 20}
	if !reflect.DeepEqual(expectedArgs, args) {
		t.Fatalf(`expected args %#v, got %#v`, expectedArgs, args)
	}
}

func TestRenumerateOrdinalParams(t *testing.T) {
	nodes, err := sqlp.Parse(`one = $1 and two = $1 and three = $2`)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	err = renumerateOrdinalParams(nodes, 5)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	expected := `one = $6 and two = $6 and three = $7`
	actual := nodes.String()
	if expected != actual {
		t.Fatalf(`expected query %q, got %q`, expected, actual)
	}
}

func TestQueryAppend(t *testing.T) {
	t.Run("without_nested", func(t *testing.T) {
		var query Query
		query.Append(`one = $1 and two = $2`, 10, 20)
		query.Append(`and three = $1 and four = $1`, 30)
		query.Append(`and five = $1 and six = $2`, 40, 50)

		strExpected := `one = $1 and two = $2 and three = $3 and four = $3 and five = $4 and six = $5`
		strActual := query.String()
		if strExpected != strActual {
			t.Fatalf(`expected query %q, got %q`, strExpected, strActual)
		}

		argsExpected := []interface{}{10, 20, 30, 40, 50}
		argsActual := query.Args
		if !reflect.DeepEqual(argsExpected, argsActual) {
			t.Fatalf(`expected args %#v, got %#v`, argsExpected, argsActual)
		}
	})

	t.Run("with_nested", func(t *testing.T) {
		var sub0 Query
		sub0.Append(`two = $1 and three = $2`, 20, 30)

		var sub1 Query
		sub1.Append(`five = $1 and six = $2`, 50, 60)

		var query Query
		query.Append(`one = $1 and $2 and $2 and four = $3 and $4 and seven = $5`, 10, sub0, 40, sub1, 70)

		strExpected := `one = $1 and two = $4 and three = $5 and two = $4 and three = $5 and four = $2 and five = $6 and six = $7 and seven = $3`
		strActual := query.String()
		if strExpected != strActual {
			t.Fatalf(`expected query %q, got %q`, strExpected, strActual)
		}

		argsExpected := []interface{}{10, 40, 70, 20, 30, 50, 60}
		argsActual := query.Args
		if !reflect.DeepEqual(argsExpected, argsActual) {
			t.Fatalf(`expected args %#v, got %#v`, argsExpected, argsActual)
		}
	})
}

func TestQueryAppendNamed(t *testing.T) {
	t.Run("without_nested", func(t *testing.T) {
		var query Query
		query.AppendNamed(`one = :one::text and two = :two`, dict{"one": 10, "two": 20})
		query.AppendNamed(`and three = :three and four = :three`, dict{"three": 30})
		query.AppendNamed(`and five = :five and six = :six`, dict{"five": 40, "six": 50})

		strExpected := `one = $1::text and two = $2 and three = $3 and four = $3 and five = $4 and six = $5`
		strActual := query.String()
		if strExpected != strActual {
			t.Fatalf(`expected query %q, got %q`, strExpected, strActual)
		}

		argsExpected := []interface{}{10, 20, 30, 40, 50}
		argsActual := query.Args
		if !reflect.DeepEqual(argsExpected, argsActual) {
			t.Fatalf(`expected args %#v, got %#v`, argsExpected, argsActual)
		}
	})

	t.Run("with_nested", func(t *testing.T) {
		var sub0 Query
		sub0.AppendNamed(`two = :two and three = :three`, dict{"two": 20, "three": 30})

		var sub1 Query
		sub1.AppendNamed(`five = :five and six = :six`, dict{"five": 50, "six": 60})

		var query Query
		query.AppendNamed(`one = :one and :sub0 and :sub0 and four = :four and :sub1 and seven = :seven`, dict{
			"one":   10,
			"sub0":  sub0,
			"four":  40,
			"sub1":  sub1,
			"seven": 70,
		})

		strExpected := `one = $1 and two = $4 and three = $5 and two = $4 and three = $5 and four = $2 and five = $6 and six = $7 and seven = $3`
		strActual := query.String()
		if strExpected != strActual {
			t.Fatalf(`expected query %q, got %q`, strExpected, strActual)
		}

		argsExpected := []interface{}{10, 40, 70, 20, 30, 50, 60}
		argsActual := query.Args
		if !reflect.DeepEqual(argsExpected, argsActual) {
			t.Fatalf(`expected args %#v, got %#v`, argsExpected, argsActual)
		}
	})
}
