package openapiimport

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportSSEComparisonRejectsInvalidReferenceProjection(t *testing.T) {
	for _, nested := range []bool{false, true} {
		for _, tc := range []struct {
			name, child string
			valid       bool
		}{
			{"items sibling", "{$ref: '#/components/schemas/Child', items: {type: string}}", false},
			{"allOf sibling", "{$ref: '#/components/schemas/Child', allOf: [{type: string}]}", false},
			{"bare reference", "{$ref: '#/components/schemas/Child'}", true},
		} {
			name := tc.name
			if nested {
				name = "nested/" + name
			}
			t.Run(name, func(t *testing.T) {
				child, event := tc.child, "{type: string}"
				if nested {
					child = "{type: object, properties: {value: " + child + "}}"
					event = "{type: object, properties: {value: {type: string}}}"
				}
				message := "{type: object, required: [message], properties: {message: " + child + "}}"
				source := string(sseInverseSource([]string{"201"}, "", sseAsyncContract("['201']", message)))
				source = strings.Replace(source, "message: {type: string}\n", "message: "+event+"\n", 1)
				source += "    Child: {type: string}\n"
				document, diagnostics, err := Analyze([]byte(source))
				require.NoError(t, err)
				if tc.valid {
					require.Empty(t, diagnostics)
					_, err = Render(document, Options{})
					require.NoError(t, err)
					return
				}
				require.Contains(t, diagnostics.Error(), "sse-async")
				require.Contains(t, diagnostics.Error(), "/x-loom-async/messages/outbound/schema")
				_, err = Render(document, Options{})
				require.Error(t, err, "rendering must not silently consume an invalid reserved reference")
			})
		}
	}
}

func TestImportSSEComparisonInvalidReferenceSymmetry(t *testing.T) {
	inline := &Schema{Type: "string"}
	invalid := &Schema{
		Ref: "#/components/schemas/Child",
		referenceError: &Diagnostic{
			Code: "schema-reference-siblings", Path: "#/reserved/child", Message: "unsupported reference intersection",
		},
	}
	planner := documentPlanner{schemas: map[string]NamedSchema{
		"Child": {Schema: inline},
		"Alias": {Schema: invalid},
	}}
	for _, tc := range []struct {
		name   string
		schema *Schema
		valid  bool
	}{
		{"invalid occurrence", invalid, false},
		{"invalid declaration chain", &Schema{Ref: "#/components/schemas/Alias"}, false},
		{"bare reference", &Schema{Ref: "#/components/schemas/Child"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, direction := range []struct {
				name string
				pair sseSchemaPair
			}{
				{"reference first", sseSchemaPair{tc.schema, inline}},
				{"inline first", sseSchemaPair{inline, tc.schema}},
			} {
				t.Run(direction.name, func(t *testing.T) {
					pair := direction.pair
					require.Equal(t, tc.valid, planner.sameSSEGraph(pair.left, pair.right, make(map[sseSchemaPair]bool)))
				})
			}
		})
	}
}
