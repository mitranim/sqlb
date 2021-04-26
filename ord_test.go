package sqlb

import (
	"encoding/json"
	"time"
)

type Internal struct {
	InternalTime *time.Time `json:"internalTime" db:"internal_time"`
}

type External struct {
	ExternalName string   `json:"externalName" db:"external_name"`
	Internal     Internal `json:"internal"     db:"internal"`
}

func TestOrdAsc(t *T) {
	eq(t,
		Ord{Path: []string{`one`, `two`}, Desc: false, NullsLast: false},
		OrdAsc(`one`, `two`),
	)
}

func TestOrdDesc(t *T) {
	eq(t,
		Ord{Path: []string{`one`, `two`}, Desc: true, NullsLast: false},
		OrdDesc(`one`, `two`),
	)
}

func TestOrdAscNl(t *T) {
	eq(t,
		Ord{Path: []string{`one`, `two`}, Desc: false, NullsLast: true},
		OrdAscNl(`one`, `two`),
	)
}

func TestOrdDescNl(t *T) {
	eq(t,
		Ord{Path: []string{`one`, `two`}, Desc: true, NullsLast: true},
		OrdDescNl(`one`, `two`),
	)
}

func TestOrdString(t *T) {
	t.Run(`singular`, func(t *T) {
		eq(t, `"one" asc`, OrdAsc(`one`).String())
		eq(t, `"one" desc`, OrdDesc(`one`).String())
		eq(t, `"one" asc nulls last`, OrdAscNl(`one`).String())
		eq(t, `"one" desc nulls last`, OrdDescNl(`one`).String())
	})

	t.Run(`binary`, func(t *T) {
		eq(t, `("one")."two" asc`, OrdAsc(`one`, `two`).String())
		eq(t, `("one")."two" desc`, OrdDesc(`one`, `two`).String())
		eq(t, `("one")."two" asc nulls last`, OrdAscNl(`one`, `two`).String())
		eq(t, `("one")."two" desc nulls last`, OrdDescNl(`one`, `two`).String())
	})

	t.Run(`plural`, func(t *T) {
		eq(t, `("one")."two"."three" asc`, OrdAsc(`one`, `two`, `three`).String())
		eq(t, `("one")."two"."three" desc`, OrdDesc(`one`, `two`, `three`).String())
		eq(t, `("one")."two"."three" asc nulls last`, OrdAscNl(`one`, `two`, `three`).String())
		eq(t, `("one")."two"."three" desc nulls last`, OrdDescNl(`one`, `two`, `three`).String())
	})
}

func TestOrdsLen(t *T) {
	eq(t, 0, Ords{}.Len())
	eq(t, 0, OrdsFrom(nil).Len())
	eq(t, 1, OrdsFrom(OrdAsc(`one`)).Len())
	eq(t, 1, OrdsFrom(nil, OrdAsc(`one`), nil).Len())
	eq(t, 2, OrdsFrom(nil, OrdAsc(`one`), nil, Query{}).Len())
}

func TestOrdsIsEmpty(t *T) {
	eq(t, true, Ords{}.IsEmpty())
	eq(t, true, OrdsFrom(nil).IsEmpty())
	eq(t, false, OrdsFrom(OrdAsc(`one`)).IsEmpty())
	eq(t, false, OrdsFrom(nil, OrdAsc(`one`), nil).IsEmpty())
	eq(t, false, OrdsFrom(nil, OrdAsc(`one`), nil, Query{}).IsEmpty())
}

func TestOrdsSimpleString(t *T) {
	str := func(ords Ords) string {
		var query Query
		ords.QueryAppend(&query)
		eq(t, 0, len(query.Args))
		return query.String()
	}

	t.Run(`empty`, func(t *T) {
		eq(t, ``, str(Ords{}))
	})
	t.Run(`singular`, func(t *T) {
		eq(t, `order by "one" asc`, str(OrdsFrom(OrdAsc(`one`))))
		eq(t, `order by "one" desc`, str(OrdsFrom(OrdDesc(`one`))))
		eq(t, `order by "one" asc nulls last`, str(OrdsFrom(OrdAscNl(`one`))))
		eq(t, `order by "one" desc nulls last`, str(OrdsFrom(OrdDescNl(`one`))))
	})
	t.Run(`binary`, func(t *T) {
		eq(t, `order by ("one")."two" asc`, str(OrdsFrom(OrdAsc(`one`, `two`))))
		eq(t, `order by ("one")."two" desc`, str(OrdsFrom(OrdDesc(`one`, `two`))))
		eq(t, `order by ("one")."two" asc nulls last`, str(OrdsFrom(OrdAscNl(`one`, `two`))))
		eq(t, `order by ("one")."two" desc nulls last`, str(OrdsFrom(OrdDescNl(`one`, `two`))))
	})
	t.Run(`plural`, func(t *T) {
		eq(t, `order by ("one")."two"."three" asc`, str(OrdsFrom(OrdAsc(`one`, `two`, `three`))))
		eq(t, `order by ("one")."two"."three" desc`, str(OrdsFrom(OrdDesc(`one`, `two`, `three`))))
		eq(t, `order by ("one")."two"."three" asc nulls last`, str(OrdsFrom(OrdAscNl(`one`, `two`, `three`))))
		eq(t, `order by ("one")."two"."three" desc nulls last`, str(OrdsFrom(OrdDescNl(`one`, `two`, `three`))))
	})
}

func TestOrdsQueryAppend(t *T) {
	t.Run(`simple_direct`, func(t *T) {
		var query Query
		query.Append(`select from where`)
		query.AppendQuery(OrdsFrom(OrdAsc(`one`, `two`, `three`)))
		eq(t, `select from where order by ("one")."two"."three" asc`, query.String())
	})

	t.Run(`simple_parametrized`, func(t *T) {
		var query Query
		query.Append(`select from where $1`, OrdsFrom(OrdAsc(`one`, `two`, `three`)))
		eq(t, `select from where order by ("one")."two"."three" asc`, query.String())
	})

	t.Run(`parametrized_by_parametrized`, func(t *T) {
		var clause Query
		clause.Append(`geo_point <-> $1 desc`, `(20,30)`)

		var query Query
		query.Append(`select from where $1 $2`, 10, OrdsFrom(OrdAsc(`five`), clause))

		eq(t, `select from where $1 order by "five" asc, geo_point <-> $2 desc`, query.String())
		eq(t, []interface{}{10, `(20,30)`}, query.Args)
	})
}

func TestOrdsDec(t *T) {
	dec := func(t *T, out *Ords, input string) {
		err := json.Unmarshal([]byte(input), out)
		if err != nil {
			t.Fatalf("failed to decode ord from JSON: %+v", err)
		}
	}

	t.Run(`decode_from_json`, func(t *T) {
		t.Run(`minimal`, func(t *T) {
			ords := OrdsFor(External{})
			dec(t, &ords, `["externalName", "internal.internalTime"]`)
			eq(t, ords.Items, OrdsFrom(OrdAsc(`external_name`), OrdAsc(`internal`, `internal_time`)).Items)
		})

		t.Run(`asc_desc`, func(t *T) {
			ords := OrdsFor(External{})
			dec(t, &ords, `["externalName asc", "internal.internalTime  DESC"]`)
			eq(t, ords.Items, OrdsFrom(OrdAsc(`external_name`), OrdDesc(`internal`, `internal_time`)).Items)
		})

		t.Run(`nulls_last`, func(t *T) {
			ords := OrdsFor(External{})
			dec(t, &ords, `["externalName nulls last", "internal.internalTime  NULLS  LAST"]`)
			eq(t, ords.Items, OrdsFrom(OrdAscNl(`external_name`), OrdAscNl(`internal`, `internal_time`)).Items)
		})

		t.Run(`asc_desc_nulls_last`, func(t *T) {
			ords := OrdsFor(External{})
			dec(t, &ords, `["externalName ASC nulls last", "internal.internalTime  DESC  NULLS  LAST"]`)
			eq(t, ords.Items, OrdsFrom(OrdAscNl(`external_name`), OrdDescNl(`internal`, `internal_time`)).Items)
		})
	})

	t.Run(`decode_from_strings`, func(t *T) {
		input := []string{"externalName asc", "internal.internalTime desc"}
		ords := OrdsFor(External{})

		err := ords.ParseSlice(input)
		if err != nil {
			t.Fatalf("failed to decode ord from strings: %+v", err)
		}

		eq(t, ords.Items, OrdsFrom(OrdAsc(`external_name`), OrdDesc(`internal`, `internal_time`)).Items)
	})

	t.Run(`reject_malformed`, func(t *T) {
		test := func(str string) {
			ords := OrdsFor(struct {
				Asc   string `json:"asc"`
				Nulls string `json:"nulls"`
			}{})

			err := ords.ParseSlice([]string{str})
			if err == nil {
				t.Fatalf("expected decoding %q to fail; decoded into %+v", str, ords)
			}
		}

		test("")
		test(" ")
		test("asc")
		test(" asc")
		test("nulls last")
		test(" nulls last")
		test("asc nulls last")
		test(" asc nulls last")
	})

	t.Run(`reject_unknown_fields`, func(t *T) {
		input := []string{"external_name asc"}
		ords := OrdsFor(External{})

		err := ords.ParseSlice(input)
		if err == nil {
			t.Fatalf("expected decoding to fail")
		}

		t.Run(`unless_lax`, func(t *T) {
			ords = Ords{Type: ords.Type, Lax: true}

			err := ords.ParseSlice(input)
			if err != nil {
				t.Fatalf("failed to decode ord from strings: %+v", err)
			}

			if len(ords.Items) > 0 {
				t.Fatalf("when decoding in lax mode, accidentally added an ord; items: %#v", ords.Items)
			}
		})
	})

	t.Run(`fail_when_type_is_not_provided`, func(t *T) {
		const str = "some_ident asc"
		ords := OrdsFor(nil)

		err := ords.ParseSlice([]string{str})
		if err == nil {
			t.Fatalf("expected decoding %q to fail", str)
		}
	})
}
