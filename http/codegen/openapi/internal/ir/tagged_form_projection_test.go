package ir

import (
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/CaliLuke/loom/internal/schematest"
	"github.com/stretchr/testify/require"
)

func TestTaggedFormSchemaAcceptance(t *testing.T) {
	root := codegen.RunDSL(t, testdata.DiscriminatorHintDSL)
	document, err := BuildDocument(root.API, root.Types, root.ResultTypes)
	require.NoError(t, err)
	validator := schematest.New(t)
	cases := []struct {
		method string
		media  string
		values []any
		valid  []bool
	}{
		{"FlatRoot", "application/x-www-form-urlencoded", []any{
			map[string]any{"grant_type": "authorization_code", "code": "abc"},
			map[string]any{"grant_type": "authorization_code", "data": map[string]any{"code": "abc"}},
			map[string]any{"grant_type": "authorization_code"},
			map[string]any{"grant_type": "nested_code", "details": map[string]any{"code": "abc"}},
			map[string]any{"grant_type": "labels", "data": map[string]any{"region": "west"}},
			map[string]any{"grant_type": "tags", "data": []any{"a", "b"}},
			map[string]any{"grant_type": "tags", "data": []any{1}},
			map[string]any{"grant_type": "code", "code": "abc"},
		}, []bool{true, false, false, true, true, true, false, false}},
		{"JSONRoot", "application/json", []any{
			map[string]any{"grant_type": "authorization_code", "data": map[string]any{"code": "abc"}},
			map[string]any{"grant_type": "authorization_code", "code": "abc"},
		}, []bool{true, false}},
	}
	for _, tc := range cases {
		media := hintRequestBody(t, document, tc.method).Content[tc.media]
		schema := media.Schema
		require.NotNil(t, media.Example)
		tc.values = append(tc.values, media.Example)
		tc.valid = append(tc.valid, true)
		results := validator.Check(t, []schematest.Batch{{Schema: map[string]any{
			"allOf":      []any{RenderSchema(schema)},
			"components": map[string]any{"schemas": RenderSchemaMap(document.Components.Schemas)},
		}, Instances: tc.values}})[0]
		for index, result := range results {
			require.Equal(t, tc.valid[index], result.Valid, "%s case %d: %s", tc.method, index, result.Errors)
		}
	}
}

func TestTaggedFormObjectConstraintProjection(t *testing.T) {
	minimum, maximum := 1, 1
	closed := false
	original := &Schema{Type: "object", Properties: map[string]*Schema{"code": {Type: "string"}}, Required: []string{"code"}, MinProperties: &minimum, MaxProperties: &maximum, AdditionalProperties: &BoolOrSchema{Bool: &closed}, Enum: []any{map[string]any{"code": "abc"}}, Example: map[string]any{"code": "abc"}}
	analyzer := NewAnalyzer(expr.NewRandom("form"), false, WithExampleValue(nil))
	analyzer.schemas["Original"] = original
	projection := &taggedFormProjection{analyzer: analyzer, key: "kind", tag: "record", components: make(map[string]string), inline: make(map[*Schema]*Schema)}
	projected := projection.object(&Schema{AllOf: []*Schema{{Ref: toRef("Original")}}})
	require.NoError(t, analyzer.Err())
	require.Equal(t, 1, *original.MaxProperties)
	require.NotContains(t, original.Properties, "kind")
	require.Equal(t, map[string]any{"code": "abc"}, original.Example)
	validator := schematest.New(t)
	results := validator.Check(t, []schematest.Batch{{Schema: map[string]any{
		"allOf": []any{RenderSchema(projected)}, "components": map[string]any{"schemas": RenderSchemaMap(analyzer.schemas)},
	}, Instances: []any{
		map[string]any{"kind": "record", "code": "abc"},
		map[string]any{"kind": "record", "code": "different"},
		map[string]any{"kind": "record", "code": "abc", "extra": true},
		map[string]any{"kind": "record"},
		map[string]any{"kind": "other", "code": "abc"},
	}}})[0]
	for index, result := range results {
		require.Equal(t, index == 0, result.Valid, "case %d: %s", index, result.Errors)
	}
}

func TestTaggedFormStructuredExampleOwnership(t *testing.T) {
	payload := map[string]any{"code": "abc"}
	envelope := map[string]any{"kind": "record", "data": payload}
	reference := &ExampleRef{Value: &Example{Value: envelope, DataValue: envelope, Summary: "retained", SerializedValue: "authored literal"}}
	media := &MediaType{Example: envelope, Examples: map[string]*ExampleRef{"named": reference}}
	form := &expr.HTTPFormEncoding{
		Body:           &expr.AttributeExpr{Type: &expr.Union{TypeKey: "kind", ValueKey: "data"}},
		TaggedBranches: []*expr.HTTPFormTaggedBranch{{Tag: "record", Object: &expr.HTTPFormEncoding{}}},
	}
	projectTaggedFormMediaExamples(media, form)
	flat := map[string]any{"kind": "record", "code": "abc"}
	require.Equal(t, flat, media.Example)
	require.Equal(t, flat, media.Examples["named"].Value.Value)
	require.Equal(t, flat, media.Examples["named"].Value.DataValue)
	require.Equal(t, "authored literal", media.Examples["named"].Value.SerializedValue)
	require.Equal(t, "retained", media.Examples["named"].Value.Summary)
	require.Equal(t, envelope, reference.Value.Value)
	require.NotContains(t, payload, "kind")
	for _, value := range []any{nil, map[string]any{}, map[string]any{"kind": "record", "data": map[string]any{}}} {
		example := &MediaType{Example: value}
		projectTaggedFormMediaExamples(example, form)
		if object, ok := value.(map[string]any); ok && object["kind"] == "record" {
			require.Equal(t, map[string]any{"kind": "record"}, example.Example)
		} else {
			require.Equal(t, value, example.Example, "absent/empty examples are not invented")
		}
	}
}
