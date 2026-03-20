package ir

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderSchemaConvertsNestedShapes(t *testing.T) {
	t.Parallel()

	schema := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"names": {
				Type: "array",
				Items: &Schema{
					Type: "string",
				},
			},
		},
		AdditionalProperties: &BoolOrSchema{Bool: boolPtr(false)},
	}

	rendered := RenderSchema(schema)

	require.Equal(t, "object", string(rendered.Type))
	require.Equal(t, "array", string(rendered.Properties["names"].Type))
	require.Equal(t, "string", string(rendered.Properties["names"].Items.Type))
	require.Equal(t, false, rendered.AdditionalProperties)
}

func TestRenderSchemaConvertsUnionMetadata(t *testing.T) {
	t.Parallel()

	schema := &Schema{
		Type: "object",
		OneOf: []*Schema{
			{Ref: "#/components/schemas/SelectionAlphaEnvelope"},
			{Ref: "#/components/schemas/SelectionBetaEnvelope"},
		},
		Discriminator: &Discriminator{
			PropertyName: "kind",
			Mapping: map[string]string{
				"alpha": "#/components/schemas/SelectionAlphaEnvelope",
				"beta":  "#/components/schemas/SelectionBetaEnvelope",
			},
		},
	}

	rendered := RenderSchema(schema)

	require.Len(t, rendered.OneOf, 2)
	require.Equal(t, "kind", rendered.Discriminator.PropertyName)
	require.Equal(t, "#/components/schemas/SelectionAlphaEnvelope", rendered.Discriminator.Mapping["alpha"])
}
