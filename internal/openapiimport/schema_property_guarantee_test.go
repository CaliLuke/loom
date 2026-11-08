package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchemaPropertyGuarantees(t *testing.T) {
	a := &analyzer{schemas: map[string]*Schema{
		"Base":     {Type: "object", Required: []string{"kind"}},
		"Nullable": {Type: "object", Nullable: true, Required: []string{"kind"}},
	}}
	for _, tc := range []struct {
		name              string
		schema            *Schema
		keyword, presence bool
	}{
		{"object", a.schemas["Base"], true, true},
		{"nullable object", a.schemas["Nullable"], true, false},
		{"nullable occurrence", &Schema{Ref: "#/components/schemas/Base", Nullable: true}, true, false},
		{"nullable declaration reference", &Schema{Ref: "#/components/schemas/Nullable"}, true, false},
		{"nonnullable intersection", &Schema{Type: "object", Bases: []*Schema{{Ref: "#/components/schemas/Nullable"}}}, true, true},
		{"inherited required", &Schema{Type: "object", Bases: []*Schema{{Ref: "#/components/schemas/Base"}}}, true, true},
		{"optional member", &Schema{Type: "object"}, false, false},
		{"nullable alternative", &Schema{OneOf: []*Schema{a.schemas["Base"], a.schemas["Nullable"]}}, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.keyword, a.oneOfRequires(tc.schema, []string{"kind"}, make(map[*Schema]bool)))
			require.Equal(t, tc.presence, a.schemaDefinesProperties(tc.schema, []string{"kind"}))
		})
	}
}
