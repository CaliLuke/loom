package openapiv3

import (
	"testing"

	"github.com/CaliLuke/loom/http/codegen/openapi"
	"github.com/stretchr/testify/require"
)

func TestComponentizeMediaTypesRejectsConflictsBeforeMutation(t *testing.T) {
	left := &MediaType{
		ComponentName: "Shared",
		Schema:        &openapi.Schema{Type: openapi.String},
		ItemSchema:    &openapi.Schema{Type: openapi.Integer},
		Encoding:      map[string]*Encoding{"payload": {ContentType: "application/json"}},
		Extensions:    map[string]any{"x-source": "left"},
	}
	right := &MediaType{
		ComponentName: "Shared",
		Schema:        &openapi.Schema{Type: openapi.String},
		ItemSchema:    &openapi.Schema{Type: openapi.Integer},
		Encoding:      map[string]*Encoding{"payload": {ContentType: "application/json"}},
		Extensions:    map[string]any{"x-source": "right"},
	}
	spec := &OpenAPI{
		Components: &Components{},
		Paths: map[string]*PathItem{
			"/left":  {Get: mediaOperation(left)},
			"/right": {Get: mediaOperation(right)},
		},
	}

	err := componentizeMediaTypes(spec)
	require.ErrorContains(t, err, `mediaTypes component "Shared"`)
	require.Nil(t, spec.Components.MediaTypes)
	require.Equal(t, "Shared", left.ComponentName)
	require.Equal(t, "Shared", right.ComponentName)

	reversedLeft := *right
	reversedRight := *left
	reversed := &OpenAPI{
		Components: &Components{},
		Paths: map[string]*PathItem{
			"/right": {Get: mediaOperation(&reversedLeft)},
			"/left":  {Get: mediaOperation(&reversedRight)},
		},
	}
	require.EqualError(t, componentizeMediaTypes(reversed), err.Error())
}

func TestComponentizeMediaTypesSharesIdenticalClaims(t *testing.T) {
	left := &MediaType{
		ComponentName: "Shared",
		Schema:        &openapi.Schema{Type: openapi.String},
		ItemSchema:    &openapi.Schema{Type: openapi.Integer},
		Encoding:      map[string]*Encoding{"payload": {ContentType: "application/json"}},
	}
	right := &MediaType{
		ComponentName: "Shared",
		Schema:        &openapi.Schema{Type: openapi.String},
		ItemSchema:    &openapi.Schema{Type: openapi.Integer},
		Encoding:      map[string]*Encoding{"payload": {ContentType: "application/json"}},
	}
	spec := &OpenAPI{
		Components: &Components{},
		Paths: map[string]*PathItem{
			"/left":  {Get: mediaOperation(left)},
			"/right": {Get: mediaOperation(right)},
		},
	}

	require.NoError(t, componentizeMediaTypes(spec))
	require.Len(t, spec.Components.MediaTypes, 1)
	require.Equal(t, "#/components/mediaTypes/Shared", left.Ref)
	require.Equal(t, left.Ref, right.Ref)
	require.Equal(t, openapi.Type(openapi.Integer), spec.Components.MediaTypes["Shared"].Value.ItemSchema.Type)
}

func TestComponentizeMediaTypesChecksExistingComponents(t *testing.T) {
	mediaType := &MediaType{
		ComponentName: "Shared",
		Schema:        &openapi.Schema{Type: openapi.String},
	}
	existing := &MediaTypeRef{Value: &MediaType{Schema: &openapi.Schema{Type: openapi.Integer}}}
	spec := &OpenAPI{
		Components: &Components{MediaTypes: map[string]*MediaTypeRef{"Shared": existing}},
		Paths:      map[string]*PathItem{"/items": {Get: mediaOperation(mediaType)}},
	}

	err := componentizeMediaTypes(spec)
	require.ErrorContains(t, err, `mediaTypes component "Shared"`)
	require.Same(t, existing, spec.Components.MediaTypes["Shared"])
	require.Equal(t, "Shared", mediaType.ComponentName)
}

func TestComponentizeMediaTypesComparesVersion32Fields(t *testing.T) {
	tests := []struct {
		name  string
		left  MediaType
		right MediaType
	}{
		{
			name: "item schema",
			left: MediaType{
				ComponentName: "Shared",
				Schema:        &openapi.Schema{Type: openapi.String},
				ItemSchema:    &openapi.Schema{Type: openapi.Integer},
			},
			right: MediaType{
				ComponentName: "Shared",
				Schema:        &openapi.Schema{Type: openapi.String},
				ItemSchema:    &openapi.Schema{Type: openapi.Boolean},
			},
		},
		{
			name: "encoding",
			left: MediaType{
				ComponentName: "Shared",
				Schema:        &openapi.Schema{Type: openapi.String},
				Encoding:      map[string]*Encoding{"payload": {ContentType: "application/json"}},
			},
			right: MediaType{
				ComponentName: "Shared",
				Schema:        &openapi.Schema{Type: openapi.String},
				Encoding:      map[string]*Encoding{"payload": {ContentType: "application/cbor"}},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			left, right := test.left, test.right
			spec := &OpenAPI{
				Components: &Components{},
				Paths: map[string]*PathItem{
					"/left":  {Get: mediaOperation(&left)},
					"/right": {Get: mediaOperation(&right)},
				},
			}
			require.ErrorContains(t, componentizeMediaTypes(spec), `mediaTypes component "Shared"`)
		})
	}
}

func mediaOperation(mediaType *MediaType) *Operation {
	return &Operation{
		Responses: map[string]*ResponseRef{
			"200": {Value: &Response{Content: map[string]*MediaType{"application/json": mediaType}}},
		},
	}
}
