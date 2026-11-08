package ir

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNullableDiscriminatorMappingRecordSurvivesWrapping checks that a deferred
// mapping targets the assertion's discriminator, not its nullable wrapper.
func TestNullableDiscriminatorMappingRecordSurvivesWrapping(t *testing.T) {
	analyzer := NewAnalyzer(nil, false)
	mapping := &Discriminator{
		PropertyName: "kind",
		Mapping:      make(map[string]string),
		Object:       true,
	}
	parent := &Schema{
		Type:          "object",
		Title:         "Nullable parent",
		Required:      []string{"kind"},
		Properties:    map[string]*Schema{"kind": {Type: "string"}},
		Discriminator: mapping,
	}
	analyzer.applyNullableSchema(parent)

	mapping.Mapping["child"] = "#/components/schemas/Child"
	require.Nil(t, parent.Discriminator)
	require.Equal(t, "Nullable parent", parent.Title)
	require.Len(t, parent.AnyOf, 2)
	require.Same(t, mapping, parent.AnyOf[0].Discriminator)
	require.Equal(t, map[string]string{"child": "#/components/schemas/Child"}, parent.AnyOf[0].Discriminator.Mapping)
	require.Equal(t, []string{"kind"}, parent.AnyOf[0].Required)
	require.Equal(t, "string", parent.AnyOf[0].Properties["kind"].Type)
	require.Equal(t, &Schema{Type: "null"}, parent.AnyOf[1])
}
