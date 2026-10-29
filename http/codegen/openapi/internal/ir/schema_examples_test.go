package ir

import (
	"encoding/json/v2"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
)

func TestSchemaPreservesOrderedExamples(t *testing.T) {
	for _, test := range []struct {
		name      string
		attribute *expr.AttributeExpr
		want      string
	}{
		{"strings", &expr.AttributeExpr{Type: expr.String, UserExamples: []*expr.ExampleExpr{{Value: "red"}, {Value: "blue"}}}, `["red","blue"]`},
		{"null and duplicate", &expr.AttributeExpr{Type: expr.String, Nullable: true, UserExamples: []*expr.ExampleExpr{{ExplicitNull: true}, {Value: "blue"}, {Value: "blue"}}}, `[null,"blue","blue"]`},
		{"bytes", &expr.AttributeExpr{Type: expr.Bytes, UserExamples: []*expr.ExampleExpr{{Value: []byte("a")}, {Value: []byte("b")}}}, `["YQ==","Yg=="]`},
		{"incomplete object omitted", &expr.AttributeExpr{Type: &expr.Object{{Name: "id", Attribute: &expr.AttributeExpr{Type: expr.String}}}, Validation: &expr.ValidationExpr{Required: []string{"id"}}, UserExamples: []*expr.ExampleExpr{{Value: map[string]any{}}, {Value: map[string]any{"id": "blue"}}}}, `[{"id":"blue"}]`},
	} {
		t.Run(test.name, func(t *testing.T) {
			schema := NewAnalyzer(expr.NewRandom("schema-examples"), false).AnalyzeSchema(test.attribute)
			data, err := json.Marshal(RenderSchema(schema), json.Deterministic(true))
			require.NoError(t, err)
			var object map[string]any
			require.NoError(t, json.Unmarshal(data, &object))
			require.Contains(t, object, "examples")
			require.NotContains(t, object, "example")
			values, err := json.Marshal(object["examples"], json.Deterministic(true))
			require.NoError(t, err)
			require.JSONEq(t, test.want, string(values))
		})
	}
}

func TestSchemaExampleSnapshotsOwnPluralValues(t *testing.T) {
	original := componentExamples{values: []any{map[string]any{"value": []any{"red"}}, NullExample{}}}
	copy := original.snapshot()
	copy.values[0].(map[string]any)["value"].([]any)[0] = "blue"
	require.Equal(t, "red", original.values[0].(map[string]any)["value"].([]any)[0])
	require.Equal(t, NullExample{}, copy.values[1])
}

func TestSchemaExamplesCustomProjection(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: expr.String, UserExamples: []*expr.ExampleExpr{{Value: "red"}, {Value: "blue"}}}
	calls := 0
	analyzer := NewAnalyzer(expr.NewRandom("custom"), false, WithExampleValue(func(attr *expr.AttributeExpr, value any) (any, bool) {
		require.Same(t, attribute, attr)
		calls++
		return "custom:" + value.(string), true
	}))
	schema := analyzer.AnalyzeSchema(attribute)
	require.Equal(t, []any{"custom:red", "custom:blue"}, schema.Examples)
	require.Equal(t, 2, calls)
}

func TestSchemaExampleOverlayReplacesWholeGroup(t *testing.T) {
	for _, local := range []*Schema{
		{Example: "local"},
		{Example: NullExample{}},
		{Example: "last", Examples: []any{"first", "last"}},
	} {
		base := &Schema{Example: "base-last", Examples: []any{"base-first", "base-last"}}
		result := overlayAsyncAnnotations(base, local)
		require.Equal(t, local.Example, result.Example)
		require.Equal(t, local.Examples, result.Examples)
	}
}

func TestSchemaExamplesCustomProjectionKeepsEarlierNull(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: expr.String, Nullable: true, UserExamples: []*expr.ExampleExpr{{ExplicitNull: true}, {Value: "blue"}}}
	analyzer := NewAnalyzer(expr.NewRandom("custom-null"), false, WithExampleValue(OpenAPIExampleValue))
	require.Equal(t, []any{NullExample{}, "blue"}, analyzer.AnalyzeSchema(attribute).Examples)
}

func TestSchemaExamplesNullableOverlayIsIdempotent(t *testing.T) {
	minimum := 1
	base := &expr.UserTypeExpr{TypeName: "Color", AttributeExpr: &expr.AttributeExpr{Type: expr.String}}
	attribute := &expr.AttributeExpr{Type: base, Nullable: true, Validation: &expr.ValidationExpr{MinLength: &minimum}, UserExamples: []*expr.ExampleExpr{{Value: "red"}, {Value: "blue"}}}
	schema := NewAnalyzer(expr.NewRandom("nullable-overlay"), false).AnalyzeSchema(attribute)
	require.Equal(t, []any{"red", "blue"}, schema.Examples)
}

func TestSchemaExamplesRespectSourceScope(t *testing.T) {
	for _, scopes := range [][]string{{"schema"}, {"media"}, {"schema", "media"}, {"schema", "schema", "media"}} {
		t.Run(strings.Join(scopes, "/"), func(t *testing.T) {
			attribute := &expr.AttributeExpr{Type: expr.String}
			var wantSchema, wantMedia []any
			for index, scope := range scopes {
				value := fmt.Sprintf("value-%d", index)
				attribute.UserExamples = append(attribute.UserExamples, &expr.ExampleExpr{Value: value, Meta: expr.MetaExpr{"openapi:example:scope": {scope}}})
				if scope == "schema" {
					wantSchema = append(wantSchema, value)
				} else {
					wantMedia = append(wantMedia, value)
				}
			}
			rand := expr.NewRandom("scoped")
			target, err := representation.PrepareStandaloneExamples(attribute, rand)
			require.NoError(t, err)
			parameter := paramFor(attribute, target, "color", "query", rand).Value
			if len(wantSchema) == 1 {
				require.Equal(t, wantSchema[0], parameter.Schema.Example)
			} else {
				require.Equal(t, wantSchema, parameter.Schema.Examples)
			}
			if len(wantMedia) == 0 {
				require.Nil(t, parameter.Example)
				require.Empty(t, parameter.Examples)
			} else {
				require.Equal(t, wantMedia[0], parameter.Example)
			}
		})
	}
}
