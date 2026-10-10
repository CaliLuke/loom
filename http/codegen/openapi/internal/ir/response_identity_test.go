package ir

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponseRetainsHistoricalSlotAfterSemanticSplit(t *testing.T) {
	paths := map[string]*PathItem{}
	add := func(path string, schemaExample, mediaExample any) {
		paths[path] = &PathItem{Operations: map[string]*Operation{"GET": {OperationID: path, Responses: map[string]*ResponseRef{"200": {Value: &Response{Description: "OK", Content: map[string]*MediaType{"application/json": {Schema: &Schema{Ref: "#/components/schemas/Shared", Example: schemaExample}, Example: mediaExample}}}}}}}}
	}
	add("/a", "first", nil)
	add("/b", "second", nil)
	add("/c", nil, "later")
	add("/d", nil, "later")
	components := componentizeResponses(paths, map[string]*Schema{"Shared": {Type: "string"}})
	require.NotEmpty(t, components)
	// Status/shape naming still reserves the slot before semantic-class splitting.
	require.Equal(t, "#/components/responses/SharedStatus200Response_b630d408", paths["/c"].Operations["GET"].Responses["200"].Ref)
	require.Equal(t, paths["/c"].Operations["GET"].Responses["200"].Ref, paths["/d"].Operations["GET"].Responses["200"].Ref)
}
