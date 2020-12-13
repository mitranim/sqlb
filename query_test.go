package sqlb

import (
	"reflect"
	"testing"
)

type Dict = map[string]interface{}

func TestQueryAppend(t *testing.T) {
	t.Run("without_nested", func(t *testing.T) {
		var query Query
		query.Append(`one = $1 and two = $2`, 10, 20)
		query.Append(`and three = $1 and four = $1`, 30)
		query.Append(`and five = $1 and six = $2`, 40, 50)

		strExpected := `one = $1 and two = $2 and three = $3 and four = $3 and five = $4 and six = $5`
		strActual := query.String()
		if strExpected != strActual {
			t.Fatalf("expected query:\n%q\ngot:\n%q", strExpected, strActual)
		}

		argsExpected := []interface{}{10, 20, 30, 40, 50}
		argsActual := query.Args
		if !reflect.DeepEqual(argsExpected, argsActual) {
			t.Fatalf("expected args:\n%#v\ngot:\n%#v", argsExpected, argsActual)
		}
	})

	t.Run("with_nested", func(t *testing.T) {
		var sub0 Query
		sub0.Append(`two = $1 and three = $2`, 20, 30)

		var sub1 Query
		sub1.Append(`five = $1 and six = $2`, 50, 60)

		var query Query
		query.Append(`one = $1 and $2 and $2 and four = $3 and $4 and seven = $5`, 10, sub0, 40, sub1, 70)

		strExpected := `one = $1 and two = $4 and three = $5 and two = $6 and three = $7 and four = $2 and five = $8 and six = $9 and seven = $3`
		strActual := query.String()
		if strExpected != strActual {
			t.Fatalf("expected query:\n%q\ngot:\n%q", strExpected, strActual)
		}

		argsExpected := []interface{}{10, 40, 70, 20, 30, 20, 30, 50, 60}
		argsActual := query.Args
		if !reflect.DeepEqual(argsExpected, argsActual) {
			t.Fatalf("expected args:\n%#v\ngot:\n%#v", argsExpected, argsActual)
		}
	})
}

func TestQueryAppendNamed(t *testing.T) {
	t.Run("without_nested", func(t *testing.T) {
		var query Query
		query.AppendNamed(`one = :one::text and two = :two`, Dict{"one": 10, "two": 20})
		query.AppendNamed(`and three = :three and four = :three`, Dict{"three": 30})
		query.AppendNamed(`and five = :five and six = :six`, Dict{"five": 40, "six": 50})

		strExpected := `one = $1::text and two = $2 and three = $3 and four = $3 and five = $4 and six = $5`
		strActual := query.String()
		if strExpected != strActual {
			t.Fatalf("expected query:\n%q\ngot:\n%q", strExpected, strActual)
		}

		argsExpected := []interface{}{10, 20, 30, 40, 50}
		argsActual := query.Args
		if !reflect.DeepEqual(argsExpected, argsActual) {
			t.Fatalf("expected args:\n%#v\ngot:\n%#v", argsExpected, argsActual)
		}
	})

	t.Run("with_nested", func(t *testing.T) {
		var sub0 Query
		sub0.AppendNamed(`two = :two and three = :three`, Dict{"two": 20, "three": 30})

		var sub1 Query
		sub1.AppendNamed(`five = :five and six = :six`, Dict{"five": 50, "six": 60})

		var query Query
		query.AppendNamed(`one = :one and :sub0 and :sub0 and four = :four and :sub1 and seven = :seven`, Dict{
			"one":   10,
			"sub0":  sub0,
			"four":  40,
			"sub1":  sub1,
			"seven": 70,
		})

		strExpected := `one = $1 and two = $2 and three = $3 and two = $4 and three = $5 and four = $6 and five = $7 and six = $8 and seven = $9`
		strActual := query.String()
		if strExpected != strActual {
			t.Fatalf("expected query:\n%q\ngot:\n%q", strExpected, strActual)
		}

		argsExpected := []interface{}{10, 20, 30, 20, 30, 40, 50, 60, 70}
		argsActual := query.Args
		if !reflect.DeepEqual(argsExpected, argsActual) {
			t.Fatalf("expected args:\n%#v\ngot:\n%#v", argsExpected, argsActual)
		}
	})
}

func TestQueryAppendQuery(t *testing.T) {
	var inner Query
	inner.Append(`$1 $2 $3`, 30, 40, 50)

	var outer Query
	outer.Append(`$1 $2`, 10, 20)
	outer.AppendQuery(inner)

	strExpected := `$1 $2 $3 $4 $5`
	strActual := outer.String()
	if strExpected != strActual {
		t.Fatalf("expected query:\n%q\ngot:\n%q", strExpected, strActual)
	}

	argsExpected := []interface{}{10, 20, 30, 40, 50}
	argsActual := outer.Args
	if !reflect.DeepEqual(argsExpected, argsActual) {
		t.Fatalf("expected args:\n%#v\ngot:\n%#v", argsExpected, argsActual)
	}
}
