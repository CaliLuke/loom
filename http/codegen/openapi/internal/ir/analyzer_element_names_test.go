package ir

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

// TestAnalyzerUsesElementNamesForSuffixedFields checks that the properties,
// required names, defaults, examples and enum values of an object whose keys
// declare a transport element name, such as "n:m", use the element name, and
// that a JSON tag name still takes precedence over it.
func TestAnalyzerUsesElementNamesForSuffixedFields(t *testing.T) {
	t.Parallel()

	leaf := &expr.AttributeExpr{
		Type:       &expr.Object{{Name: "count:c", Attribute: &expr.AttributeExpr{Type: expr.Int}}},
		Validation: &expr.ValidationExpr{Required: []string{"count:c"}},
	}
	tagged := &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:tag:json:name": {"jt"}}}
	attribute := &expr.AttributeExpr{
		Type: &expr.Object{
			{Name: "n:m", Attribute: &expr.AttributeExpr{Type: expr.String}},
			{Name: "req:r", Attribute: &expr.AttributeExpr{Type: expr.Int}},
			{Name: "tag:tg", Attribute: tagged},
			{Name: "obj:o", Attribute: leaf},
		},
		DefaultValue: map[string]any{"n:m": "default", "req:r": 1, "obj:o": map[string]any{"count:c": 2}},
		UserExamples: []*expr.ExampleExpr{{Value: map[string]any{
			"n:m": "example", "req:r": 3, "tag:tg": "t", "obj:o": map[string]any{"count:c": 4},
		}}},
		Validation: &expr.ValidationExpr{
			Required: []string{"req:r", "tag:tg", "obj:o"},
			Values: []any{map[string]any{
				"n:m": "enum", "req:r": 5, "tag:tg": "e", "obj:o": map[string]any{"count:c": 6},
			}},
		},
	}
	analyzer := NewAnalyzer(expr.NewRandom("element-names"), false, WithExampleValue(openAPIExampleValueForTest))
	schema := analyzer.AnalyzeSchema(attribute)

	assert.ElementsMatch(t, []string{"m", "r", "jt", "o"}, keys(schema.Properties))
	assert.Equal(t, []string{"r", "jt", "o"}, schema.Required)
	require.Contains(t, schema.Properties, "o")
	assert.ElementsMatch(t, []string{"c"}, keys(schema.Properties["o"].Properties))
	assert.Equal(t, []string{"c"}, schema.Properties["o"].Required)
	assert.Equal(t, map[string]any{"m": "default", "r": 1, "o": map[string]any{"c": 2}}, schema.DefaultValue)
	assert.Equal(t, map[string]any{"m": "example", "r": 3, "jt": "t", "o": map[string]any{"c": 4}}, schema.Example)
	assert.Equal(t, []any{map[string]any{"m": "enum", "r": 5, "jt": "e", "o": map[string]any{"c": 6}}}, schema.Enum)

	plain := &expr.AttributeExpr{
		Type: &expr.Object{
			{Name: "m", Attribute: &expr.AttributeExpr{Type: expr.String}},
			{Name: "r", Attribute: &expr.AttributeExpr{Type: expr.Int}},
			{Name: "tg", Attribute: tagged},
			{Name: "o", Attribute: &expr.AttributeExpr{
				Type:       &expr.Object{{Name: "c", Attribute: &expr.AttributeExpr{Type: expr.Int}}},
				Validation: &expr.ValidationExpr{Required: []string{"c"}},
			}},
		},
		Validation: &expr.ValidationExpr{Required: []string{"r", "tg", "o"}},
	}
	suffixed := &expr.AttributeExpr{Type: attribute.Type, Validation: &expr.ValidationExpr{Required: attribute.Validation.Required}}
	assert.Equal(t, analyzer.FingerprintAttribute(plain), analyzer.FingerprintAttribute(suffixed),
		"a suffixed object has the fingerprint of the object named after its element names")
}

// TestOpenAPIExampleValueMatchesSuffixedUntaggedBranches checks that the
// example of an untagged union whose branch fields declare element names
// selects its branch by the element names and keeps the example.
func TestOpenAPIExampleValueMatchesSuffixedUntaggedBranches(t *testing.T) {
	t.Parallel()

	branch := func(name, key string) *expr.NamedAttributeExpr {
		return &expr.NamedAttributeExpr{Name: name, Attribute: &expr.AttributeExpr{Type: &expr.UserTypeExpr{
			TypeName: name,
			AttributeExpr: &expr.AttributeExpr{
				Type:       &expr.Object{{Name: key, Attribute: &expr.AttributeExpr{Type: expr.String}}},
				Validation: &expr.ValidationExpr{Required: []string{key}},
				Meta:       expr.MetaExpr{"openapi:additionalProperties": {"false"}},
			},
		}}}
	}
	union := &expr.AttributeExpr{Type: &expr.Union{
		Untagged: true,
		Values:   []*expr.NamedAttributeExpr{branch("Data", "data:dt"), branch("Fail", "reason:rs")},
	}}

	value, ok := OpenAPIExampleValue(union, map[string]any{"reason:rs": "no"})

	require.True(t, ok)
	assert.Equal(t, map[string]any{"rs": "no"}, value)
}

func keys[V any](m map[string]V) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	return names
}
