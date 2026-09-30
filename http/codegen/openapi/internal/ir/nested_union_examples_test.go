package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

type selectionRandomizer struct {
	expr.Randomizer
	choices []int
	next    int
}

func TestSynthesizedNestedUnionExamples(t *testing.T) {
	leaf := &expr.UserTypeExpr{TypeName: "Leaf", AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Object{{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}}},
	}}
	other := &expr.UserTypeExpr{TypeName: "Other", AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Object{{Name: "count", Attribute: &expr.AttributeExpr{Type: expr.Int}}},
	}}
	inner := &expr.Union{TypeName: "Choice", Values: []*expr.NamedAttributeExpr{
		{Name: "Leaf", Attribute: &expr.AttributeExpr{Type: leaf}},
		{Name: "Other", Attribute: &expr.AttributeExpr{Type: other}},
	}}
	outer := &expr.Union{TypeName: "Block", Values: []*expr.NamedAttributeExpr{
		{Name: "s", Attribute: &expr.AttributeExpr{Type: inner}},
		{Name: "t", Attribute: &expr.AttributeExpr{Type: inner}},
	}}
	record := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "c", Attribute: &expr.AttributeExpr{Type: inner}},
		{Name: "also", Attribute: &expr.AttributeExpr{Type: inner}},
		{Name: "block", Attribute: &expr.AttributeExpr{Type: outer}},
	}}
	for _, test := range []struct {
		name string
		attr *expr.AttributeExpr
	}{
		{"scalar", record},
		{"collection", &expr.AttributeExpr{Type: &expr.Array{ElemType: record}}},
		{"map", &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: record}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			schema := NewAnalyzer(expr.NewRandom("nested-union"), false).AnalyzeSchema(test.attr)
			require.NotNil(t, schema.Example)
			var records []any
			switch test.name {
			case "scalar":
				records = []any{schema.Example}
			case "collection":
				records = schema.Example.([]any)
			case "map":
				for _, record := range schema.Example.(map[string]any) {
					records = append(records, record)
				}
			}
			require.NotEmpty(t, records)
			for _, record := range records {
				object := record.(map[string]any)
				for _, name := range []string{"c", "also"} {
					value := object[name].(map[string]any)
					require.Contains(t, []any{"Leaf", "Other"}, value["type"])
					require.Contains(t, value, "value")
				}
				block := object["block"].(map[string]any)
				require.Contains(t, []any{"s", "t"}, block["type"])
				nested := block["value"].(map[string]any)
				require.Contains(t, []any{"Leaf", "Other"}, nested["type"])
				require.Contains(t, nested, "value")
			}
		})
	}
}

func TestSynthesizedExamplePreservesOccurrenceSelections(t *testing.T) {
	leaf := &expr.UserTypeExpr{TypeName: "Leaf", AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Object{{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}}},
	}}
	inner := &expr.UserTypeExpr{TypeName: "Choice", AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Union{Values: []*expr.NamedAttributeExpr{
			{Name: "Leaf", Attribute: &expr.AttributeExpr{Type: leaf, Meta: expr.MetaExpr{"oneof:type:tag": {"leaf-kind"}}}},
		}},
	}}
	outer := &expr.Union{TypeKey: "kind", ValueKey: "data", Values: []*expr.NamedAttributeExpr{
		{Name: "s", Attribute: &expr.AttributeExpr{Type: inner}},
		{Name: "t", Attribute: &expr.AttributeExpr{Type: inner}},
	}}
	attribute := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "first", Attribute: &expr.AttributeExpr{Type: outer}},
		{Name: "second", Attribute: &expr.AttributeExpr{Type: outer}},
	}}
	generator := &expr.ExampleGenerator{Randomizer: &selectionRandomizer{
		Randomizer: expr.NewFakerRandomizer("selections"), choices: []int{0, 0, 1},
	}}
	var cached any = map[string]any{"name": "raw cached example"}
	generator.HaveSeen(inner.ID(), &cached)
	source := synthesizedOpenAPIExample(attribute, generator)
	require.True(t, source.present)
	require.True(t, source.declared)
	value, ok := openAPIDeclaredExampleValue(attribute, source.value)
	require.True(t, ok)
	object := value.(map[string]any)
	first := object["first"].(map[string]any)
	second := object["second"].(map[string]any)
	require.Equal(t, "s", first["kind"])
	require.Equal(t, "t", second["kind"])
	require.Equal(t, first["data"], second["data"], "only the inner selected value is shared")
	require.Equal(t, "leaf-kind", first["data"].(map[string]any)["type"])
	prior, seen := generator.PreviouslySeen(inner.ID())
	require.True(t, seen)
	require.Same(t, &cached, prior, "selection-preserving generation must not overwrite the raw memo")
	require.Same(t, inner, outer.Values[0].Attribute.Type, "generation must not mutate the source design")
	require.IsType(t, &expr.Union{}, inner.Attribute().Type)
}

func TestCustomExampleProjectionRetainsRawUnionInput(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: &expr.Union{Values: []*expr.NamedAttributeExpr{
		{Name: "word", Attribute: &expr.AttributeExpr{Type: expr.String}},
	}}}
	analyzer := NewAnalyzer(expr.NewRandom("custom-projection"), false, WithExampleValue(func(_ *expr.AttributeExpr, raw any) (any, bool) {
		require.IsType(t, "", raw)
		return "custom", true
	}))
	require.Equal(t, "custom", analyzer.AnalyzeSchema(attribute).Example)
}

func TestSynthesizedUnionRetainsBranchCoercion(t *testing.T) {
	for _, test := range []struct {
		name     string
		untagged bool
		kind     expr.DataType
		want     any
	}{
		{"tagged bytes", false, expr.Bytes, map[string]any{"type": "data", "value": map[string]any{"data": "aGk="}}},
		{"untagged text", true, expr.String, map[string]any{"data": "hi"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			attribute := &expr.AttributeExpr{Type: &expr.Union{Untagged: test.untagged, Values: []*expr.NamedAttributeExpr{
				{Name: "data", Attribute: &expr.AttributeExpr{Type: &expr.Object{
					{Name: "data", Attribute: &expr.AttributeExpr{Type: test.kind, UserExamples: []*expr.ExampleExpr{{Value: "hi"}}}},
				}}},
			}}}
			source := synthesizedOpenAPIExample(attribute, expr.NewRandom("bytes"))
			require.True(t, source.present)
			require.True(t, source.declared)
			value, ok := openAPIDeclaredExampleValue(attribute, source.value)
			require.True(t, ok)
			require.Equal(t, test.want, value)
		})
	}
}

func TestSynthesizedRecursiveUnionExample(t *testing.T) {
	tree := &expr.UserTypeExpr{TypeName: "Tree", AttributeExpr: &expr.AttributeExpr{}}
	tree.Type = &expr.Union{Values: []*expr.NamedAttributeExpr{
		{Name: "node", Attribute: &expr.AttributeExpr{Type: &expr.Object{
			{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}},
			{Name: "child", Attribute: &expr.AttributeExpr{Type: tree}},
		}}},
	}}
	attribute := &expr.AttributeExpr{Type: tree}
	source := synthesizedOpenAPIExample(attribute, expr.NewRandom("recursive"))
	require.True(t, source.present)
	require.True(t, source.declared)
	value, ok := openAPIDeclaredExampleValue(attribute, source.value)
	require.True(t, ok)
	envelope := value.(map[string]any)
	require.Equal(t, "node", envelope["type"])
	node := envelope["value"].(map[string]any)
	require.IsType(t, "", node["name"])
	require.NotContains(t, node, "child", "an in-progress recursive value is omitted")
}

func (random *selectionRandomizer) Int() int {
	value := random.choices[random.next]
	random.next++
	return value
}
