package sqlb_test

import (
	"fmt"

	"github.com/mitranim/sqlb"
)

func ExampleCols() {
	type Internal struct {
		Id   string `db:"id"`
		Name string `db:"name"`
	}

	type External struct {
		Id       string   `db:"id"`
		Name     string   `db:"name"`
		Internal Internal `db:"internal"`
	}

	fmt.Println(sqlb.Cols(External{}))

	/**
	Formatted here for readability:

	"id",
	"name",
	("internal")."id"   as "internal.id",
	("internal")."name" as "internal.name"
	*/
}
