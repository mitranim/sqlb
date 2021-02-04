package sqlb

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

type Internal struct {
	InternalTime *time.Time `json:"internalTime" db:"internal_time"`
}

type External struct {
	ExternalName string   `json:"externalName" db:"external_name"`
	Internal     Internal `json:"internal"     db:"internal"`
}

func TestOrdAsc(t *testing.T) {
	eq(t,
		Ord{Path: []string{`one`, `two`}, IsDesc: false},
		OrdAsc(`one`, `two`),
	)
}

func TestOrdDesc(t *testing.T) {
	eq(t,
		Ord{Path: []string{`one`, `two`}, IsDesc: true},
		OrdDesc(`one`, `two`),
	)
}

func TestOrdString(t *testing.T) {
	t.Run(`singular`, func(t *testing.T) {
		eq(t, `"one" asc nulls last`, OrdAsc(`one`).String())
		eq(t, `"one" desc nulls last`, OrdDesc(`one`).String())
	})
	t.Run(`binary`, func(t *testing.T) {
		eq(t, `("one")."two" asc nulls last`, OrdAsc(`one`, `two`).String())
		eq(t, `("one")."two" desc nulls last`, OrdDesc(`one`, `two`).String())
	})
	t.Run(`plural`, func(t *testing.T) {
		eq(t, `("one")."two"."three" asc nulls last`, OrdAsc(`one`, `two`, `three`).String())
		eq(t, `("one")."two"."three" desc nulls last`, OrdDesc(`one`, `two`, `three`).String())
	})
}

func TestOrdsLen(t *testing.T) {
	eq(t, 0, Ords{}.Len())
	eq(t, 0, OrdsFrom(nil).Len())
	eq(t, 1, OrdsFrom(OrdAsc(`one`)).Len())
	eq(t, 1, OrdsFrom(nil, OrdAsc(`one`), nil).Len())
	eq(t, 2, OrdsFrom(nil, OrdAsc(`one`), nil, Query{}).Len())
}

func TestOrdsIsEmpty(t *testing.T) {
	eq(t, true, Ords{}.IsEmpty())
	eq(t, true, OrdsFrom(nil).IsEmpty())
	eq(t, false, OrdsFrom(OrdAsc(`one`)).IsEmpty())
	eq(t, false, OrdsFrom(nil, OrdAsc(`one`), nil).IsEmpty())
	eq(t, false, OrdsFrom(nil, OrdAsc(`one`), nil, Query{}).IsEmpty())
}

func TestOrdsSimpleString(t *testing.T) {
	str := func(ords Ords) string {
		var query Query
		ords.QueryAppend(&query)
		eq(t, 0, len(query.Args))
		return query.String()
	}

	t.Run(`empty`, func(t *testing.T) {
		eq(t, ``, str(Ords{}))
	})
	t.Run(`singular`, func(t *testing.T) {
		eq(t, `order by "one" asc nulls last`, str(OrdsFrom(OrdAsc(`one`))))
		eq(t, `order by "one" desc nulls last`, str(OrdsFrom(OrdDesc(`one`))))
	})
	t.Run(`binary`, func(t *testing.T) {
		eq(t, `order by ("one")."two" asc nulls last`, str(OrdsFrom(OrdAsc(`one`, `two`))))
		eq(t, `order by ("one")."two" desc nulls last`, str(OrdsFrom(OrdDesc(`one`, `two`))))
	})
	t.Run(`plural`, func(t *testing.T) {
		eq(t, `order by ("one")."two"."three" asc nulls last`, str(OrdsFrom(OrdAsc(`one`, `two`, `three`))))
		eq(t, `order by ("one")."two"."three" desc nulls last`, str(OrdsFrom(OrdDesc(`one`, `two`, `three`))))
	})
}

func TestOrdsQueryAppend(t *testing.T) {
	t.Run(`simple_direct`, func(t *testing.T) {
		var query Query
		query.Append(`select from where`)
		query.AppendQuery(OrdsFrom(OrdAsc(`one`, `two`, `three`)))
		eq(t, `select from where order by ("one")."two"."three" asc nulls last`, query.String())
	})

	t.Run(`simple_parametrized`, func(t *testing.T) {
		var query Query
		query.Append(`select from where $1`, OrdsFrom(OrdAsc(`one`, `two`, `three`)))
		eq(t, `select from where order by ("one")."two"."three" asc nulls last`, query.String())
	})

	t.Run(`parametrized_parametrized`, func(t *testing.T) {
		var clause Query
		clause.Append(`geo_point <-> $1 desc`, `(20,30)`)

		var query Query
		query.Append(`select from where $1 $2`, 10, OrdsFrom(OrdAsc(`five`), clause))

		eq(t, `select from where $1 order by "five" asc nulls last, geo_point <-> $2 desc`, query.String())
		eq(t, []interface{}{10, `(20,30)`}, query.Args)
	})
}

func TestOrdsDec(t *testing.T) {
	t.Run(`decode_from_json`, func(t *testing.T) {
		const input = `["externalName asc", "internal.internalTime desc"]`
		ords := OrdsFor(External{})

		err := json.Unmarshal([]byte(input), &ords)
		if err != nil {
			t.Fatalf("failed to decode ord from JSON: %+v", err)
		}

		eq(t, ords.Items, OrdsFrom(OrdAsc(`external_name`), OrdDesc(`internal`, `internal_time`)).Items)
	})

	t.Run(`decode_from_strings`, func(t *testing.T) {
		input := []string{"externalName asc", "internal.internalTime desc"}
		ords := OrdsFor(External{})

		err := ords.ParseSlice(input)
		if err != nil {
			t.Fatalf("failed to decode ord from strings: %+v", err)
		}

		eq(t, ords.Items, OrdsFrom(OrdAsc(`external_name`), OrdDesc(`internal`, `internal_time`)).Items)
	})

	t.Run(`reject_unknown_fields`, func(t *testing.T) {
		input := []string{"external_name asc nulls last"}
		ords := OrdsFor(External{})

		err := ords.ParseSlice(input)
		if err == nil {
			t.Fatalf("expected decoding to fail")
		}
	})

	t.Run(`fail_when_type_is_not_provided`, func(t *testing.T) {
		input := []string{"some_ident asc nulls last"}
		ords := OrdsFor(nil)

		err := ords.ParseSlice(input)
		if err == nil {
			t.Fatalf("expected decoding to fail")
		}
	})
}

func eq(t *testing.T, expected interface{}, actual interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected:\n%#v\nactual:\n%#v", expected, actual)
	}
}
