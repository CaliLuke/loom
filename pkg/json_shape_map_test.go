package loom

import (
	"encoding/json/jsontext"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONShapeMaps(t *testing.T) {
	text := &JSONShape{Kind: "string", Rules: JSONShapeRules{MinLength: new(2)}}
	dictionary := &JSONShape{Kind: "map", Child: text}
	bounded := &JSONShape{Kind: "alias", Child: dictionary, Rules: JSONShapeRules{MinLength: new(1), MaxLength: new(2)}}
	nullable := &JSONShape{Kind: "map", Nullable: true, Child: text, Rules: JSONShapeRules{Enums: [][]jsontext.Value{{[]byte(`{"a":"ok"}`)}}}}
	for _, tc := range []struct {
		name  string
		shape *JSONShape
		wire  string
		want  bool
	}{
		{"empty", dictionary, `{}`, true},
		{"string keys", dictionary, `{"":"ok","escaped\"":"yes","雪":"ok"}`, true},
		{"invalid member", dictionary, `{"a":"ok","b":"x"}`, false},
		{"wrong member type", dictionary, `{"a":2}`, false},
		{"null member", dictionary, `{"a":null}`, false},
		{"null map", dictionary, `null`, false},
		{"wrong container", dictionary, `[]`, false},
		{"duplicate keys", dictionary, `{"a":"ok","a":"yes"}`, false},
		{"bounded valid", bounded, `{"a":"ok"}`, true},
		{"bounded empty", bounded, `{}`, false},
		{"bounded too long", bounded, `{"a":"ok","b":"ok","c":"ok"}`, false},
		{"nullable enum null", nullable, `null`, true},
		{"nullable enum member", nullable, `{"a":"ok"}`, true},
		{"nullable enum other", nullable, `{"b":"ok"}`, false},
		{"nested map", &JSONShape{Kind: "map", Child: dictionary}, `{"outer":{"a":"ok"}}`, true},
		{"nested invalid", &JSONShape{Kind: "map", Child: dictionary}, `{"outer":{"a":"x"}}`, false},
		{"array of maps", &JSONShape{Kind: "array", Child: dictionary}, `[{"a":"ok"},{}]`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			matched, err := tc.shape.Match([]byte(tc.wire))
			require.NoError(t, err)
			require.Equal(t, tc.want, matched)
		})
	}
}

func TestJSONShapeMapDescriptionsValidateBeforeValues(t *testing.T) {
	for _, shape := range []*JSONShape{
		{Kind: "map"},
		{Kind: "map", Nullable: true, Child: &JSONShape{Kind: "unknown"}},
	} {
		for _, wire := range []string{`{}`, `null`} {
			_, err := shape.Match([]byte(wire))
			require.Error(t, err)
		}
	}
}

func TestJSONShapeRecursiveMapValues(t *testing.T) {
	node := &JSONShape{Kind: "object", Closed: true}
	node.Fields = []JSONShapeField{
		{Name: "name", Required: true, Shape: &JSONShape{Kind: "string"}},
		{Name: "children", Shape: &JSONShape{Kind: "map", Child: node}},
	}
	for _, tc := range []struct {
		wire string
		want bool
	}{
		{`{"name":"root","children":{"leaf":{"name":"leaf"}}}`, true},
		{`{"name":"root","children":{}}`, true},
		{`{"name":"root","children":{"leaf":{}}}`, false},
		{`{"name":"root","children":{"leaf":null}}`, false},
	} {
		matched, err := node.Match([]byte(tc.wire))
		require.NoError(t, err)
		require.Equal(t, tc.want, matched, tc.wire)
	}
}
