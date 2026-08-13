package openapiv3

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/openapi"
)

func TestOpenAPI32MediaTypePassesVisitParameterContent(t *testing.T) {
	tests := []struct {
		name      string
		parameter func(*OpenAPI) *Parameter
	}{
		{
			name: "path parameter",
			parameter: func(spec *OpenAPI) *Parameter {
				return spec.Paths["/widgets"].Parameters[0].Value
			},
		},
		{
			name: "operation parameter",
			parameter: func(spec *OpenAPI) *Parameter {
				return spec.Paths["/widgets"].Get.Parameters[0].Value
			},
		},
		{
			name: "component parameter",
			parameter: func(spec *OpenAPI) *Parameter {
				return spec.Components.Parameters["Filter"].Value
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spec := parameterContentSpec()
			parameter := test.parameter(spec)
			mediaType := parameter.Content["application/json"]

			promoteConfiguredItemSchemas(spec)
			require.Nil(t, mediaType.Schema)
			require.NotNil(t, mediaType.ItemSchema)

			componentizeMediaTypes(spec)
			require.Equal(t, "#/components/mediaTypes/WidgetFilter", mediaType.Ref)
			require.Contains(t, spec.Components.MediaTypes, "WidgetFilter")
			component := spec.Components.MediaTypes["WidgetFilter"].Value
			require.NotNil(t, component.ItemSchema)
			require.EqualValues(t, openapi.String, component.ItemSchema.Type)
		})
	}
}

func parameterContentSpec() *OpenAPI {
	newParameter := func() *ParameterRef {
		return &ParameterRef{Value: &Parameter{Content: map[string]*MediaType{
			"application/json": {
				Schema:        &openapi.Schema{Type: openapi.String},
				UseItemSchema: true,
				ComponentName: "WidgetFilter",
			},
		}}}
	}
	return &OpenAPI{
		Paths: map[string]*PathItem{
			"/widgets": {
				Parameters: []*ParameterRef{newParameter()},
				Get: &Operation{
					Parameters: []*ParameterRef{newParameter()},
				},
			},
		},
		Components: &Components{
			Parameters: map[string]*ParameterRef{"Filter": newParameter()},
		},
	}
}
