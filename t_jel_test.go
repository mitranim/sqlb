package sqlb

import (
	"testing"
	"time"
)

func Test_Jel_Transcode(t *testing.T) {
	type Internal struct {
		InternalTime *time.Time `json:"internalTime" db:"internal_time"`
	}

	type External struct {
		ExternalName string   `json:"externalName" db:"external_name"`
		Internal     Internal `json:"internal"     db:"internal"`
	}

	const src = `
		["and",
			["or",
				false,
				["=", "externalName", ["externalName", "literal string"]]
			],
			["and",
				true,
				["<", "internal.internalTime", ["internal.internalTime", "9999-01-01T00:00:00Z"]]
			]
		]
	`

	expr := Jel{elemTypeOf((*External)(nil)), src}
	text, args := Reify(expr)

	eq(
		t,
		`(($1 or ("external_name" = $2)) and ($3 and (("internal")."internal_time" < $4)))`,
		text,
	)

	eq(
		t,
		[]interface{}{false, `literal string`, true, parseTime(`9999-01-01T00:00:00Z`)},
		args,
	)
}
