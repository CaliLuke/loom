package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func TestLocationSchemaDescriptions(t *testing.T) {
	for _, tc := range []struct {
		name, outer, schema string
		explicit            bool
	}{
		{"both", "outer", "schema", true},
		{"schema only", "", "schema", true},
		{"empty schema", "outer", "", true},
		{"unscoped", "outer", "", false},
	} {
		for _, cached := range []bool{false, true} {
			for _, role := range []string{"parameter", "header"} {
				t.Run(tc.name+"/"+role+"/"+map[bool]string{false: "direct", true: "prepared"}[cached], func(t *testing.T) {
					attribute := &expr.AttributeExpr{Type: expr.String, Description: tc.outer}
					if tc.explicit {
						attribute.Meta = expr.MetaExpr{"openapi:description:schema": {tc.schema}}
					}
					random := expr.NewRandom("location-descriptions")
					value, err := representation.PrepareStandaloneExamples(attribute, random)
					require.NoError(t, err)
					original := &Schema{Type: "string", Description: tc.outer}
					var prepared []map[*expr.AttributeExpr]*Schema
					if cached {
						prepared = []map[*expr.AttributeExpr]*Schema{{attribute: original}}
					}
					var outer string
					var schema *Schema
					if role == "parameter" {
						parameter := preparedParamFor(attribute, value, "label", "query", false, random, false, prepared...).Value
						outer, schema = parameter.Description, parameter.Schema
					} else {
						headers := headersFromIR([]*transportir.Header{{Name: "label", HTTPName: "X-Label", Attribute: attribute, Value: value}}, random, false, prepared...)
						header := headers["X-Label"].Value
						outer, schema = header.Description, header.Schema
					}
					require.Equal(t, tc.outer, outer)
					want := tc.schema
					if !tc.explicit && role == "header" {
						want = tc.outer
					}
					require.Equal(t, want, schema.Description)
					require.Equal(t, tc.outer, attribute.Description)
					require.Equal(t, tc.outer, original.Description, "prepared shared schema must remain unchanged")
					if cached {
						require.NotSame(t, original, schema)
					}
				})
			}
		}
	}
}
