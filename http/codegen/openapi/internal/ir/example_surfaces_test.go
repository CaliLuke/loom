package ir

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func TestSchemaExampleSurfacesUseOpenAPIValues(t *testing.T) {
	allBytes := make([]byte, 256)
	for i := range allBytes {
		allBytes[i] = byte(i)
	}
	object := &expr.Object{{Name: "id", Attribute: &expr.AttributeExpr{Type: expr.String}}}
	for _, tc := range []struct {
		name string
		attr *expr.AttributeExpr
		want any
	}{
		{"text bytes", &expr.AttributeExpr{Type: expr.Bytes, UserExamples: []*expr.ExampleExpr{{Value: "hello"}}}, "aGVsbG8="},
		{"all bytes", &expr.AttributeExpr{Type: expr.Bytes, UserExamples: []*expr.ExampleExpr{{Value: allBytes}}}, base64.StdEncoding.EncodeToString(allBytes)},
		{"empty bytes", &expr.AttributeExpr{Type: expr.Bytes, UserExamples: []*expr.ExampleExpr{{Value: []byte{}}}}, ""},
		{"complete object", &expr.AttributeExpr{Type: object, Validation: &expr.ValidationExpr{Required: []string{"id"}}, UserExamples: []*expr.ExampleExpr{{Value: map[string]any{"id": "present"}}}}, map[string]any{"id": "present"}},
		{"incomplete object", &expr.AttributeExpr{Type: object, Validation: &expr.ValidationExpr{Required: []string{"id"}}, UserExamples: []*expr.ExampleExpr{{Value: map[string]any{}}}}, nil},
		{"explicit null", &expr.AttributeExpr{Type: expr.String, Nullable: true, UserExamples: []*expr.ExampleExpr{{ExplicitNull: true}}}, NullExample{}},
	} {
		for _, surface := range []string{"default", "parameter item", "response header", "async"} {
			t.Run(tc.name+"/"+surface, func(t *testing.T) {
				attr := expr.DupAtt(tc.attr)
				rand := expr.NewRandom("example-surfaces")
				var schema *Schema
				switch surface {
				case "default":
					schema = NewAnalyzer(rand, false).AnalyzeSchema(attr)
				case "parameter item":
					array := &expr.AttributeExpr{
						Type:         &expr.Array{ElemType: attr},
						UserExamples: []*expr.ExampleExpr{{Value: []any{tc.attr.UserExamples[0].Value}}},
					}
					schema = paramFor(array, "values", "query", false, rand, false).Value.Schema.Items
				case "response header":
					headers := []*transportir.Header{{Name: "value", HTTPName: "X-Value", Attribute: attr}}
					schema = headersFromIR(headers, rand, false)["X-Value"].Value.Schema
				case "async":
					schema = buildInlineAsyncSchema(attr, rand, false)
				}
				require.Equal(t, tc.want, schema.Example)
			})
		}
	}
}

func TestAnalyzerExampleProjectionOverride(t *testing.T) {
	attr := &expr.AttributeExpr{Type: expr.String, UserExamples: []*expr.ExampleExpr{{Value: "authored"}}}
	analyzer := NewAnalyzer(expr.NewRandom("override"), false, WithExampleValue(func(got *expr.AttributeExpr, raw any) (any, bool) {
		require.Same(t, attr, got)
		require.Equal(t, "authored", raw)
		return "custom", true
	}))
	require.Equal(t, "custom", analyzer.AnalyzeSchema(attr).Example)
}
