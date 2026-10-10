package ir

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestResponseMediaAlternatives(t *testing.T) {
	root := codegen.RunDSL(t, testdata.ErrorMediaDSL)
	document, err := BuildDocument(root.API, root.Types, root.ResultTypes)
	require.NoError(t, err)
	for _, path := range []string{"/mixed", "/reversed"} {
		response := document.Paths[path].Operations["GET"].Responses["500"]
		if response.Ref != "" {
			response = document.Components.Responses[strings.TrimPrefix(response.Ref, "#/components/responses/")]
		}
		require.NotNil(t, response)
		require.Len(t, response.Value.Content, 2, path)
		require.Contains(t, response.Value.Content, "application/json")
		require.Equal(t, "string", response.Value.Content["text/html; charset=utf-8"].Schema.Type)
		require.Contains(t, response.Value.Description, "html")
		require.Contains(t, response.Value.Description, "object")
	}
}

func TestResponseMediaConflictReportsEndpointAndStatus(t *testing.T) {
	root := codegen.RunDSL(t, testdata.ErrorMediaDSL)
	endpoint := root.API.HTTP.Services[0].HTTPEndpoints[0]
	for _, response := range endpoint.HTTPErrors {
		response.Response.Meta = expr.MetaExpr{"openapi:component:response": {response.Name}}
	}
	_, err := BuildDocument(root.API, root.Types, root.ResultTypes)
	require.ErrorContains(t, err, "OpenAPI response 500 for projection.mixed: conflicting response component names")
}

func TestMediaTypeMetadataOwnsOnlyMediaAnnotations(t *testing.T) {
	meta := expr.MetaExpr{
		"http:body": {"body"}, "openapi:typename": {"Body"},
		"openapi:itemSchema": {"true"}, "openapi:encoding:data:contentType": {"application/json"},
		"openapi:prefixEncoding:0:style": {"form"}, "openapi:itemEncoding:explode": {"false"},
	}
	selected := mediaTypeMetadata(meta)
	require.Len(t, selected, 4)
	require.NotContains(t, selected, "http:body")
	require.NotContains(t, selected, "openapi:typename")
	selected["openapi:itemSchema"][0] = "false"
	require.Equal(t, "true", meta["openapi:itemSchema"][0], "captured metadata must be detached")
}

func TestResponseMediaDeclarationOrder(t *testing.T) {
	var first *Document
	for _, reversed := range []bool{false, true} {
		root := codegen.RunDSL(t, testdata.ErrorMediaDSL)
		if reversed {
			for _, endpoint := range root.API.HTTP.Services[0].HTTPEndpoints {
				slices.Reverse(endpoint.HTTPErrors)
			}
		}
		document, err := BuildDocument(root.API, root.Types, root.ResultTypes)
		require.NoError(t, err)
		if first == nil {
			first = document
		} else {
			require.Equal(t, first, document)
		}
	}
}

func TestMergeResponseAlternatives(t *testing.T) {
	stringSchema := &Schema{Type: "string"}
	integerSchema := &Schema{Type: "integer"}
	first := &Response{
		Description: "First error", Content: map[string]*MediaType{"application/json": {Schema: stringSchema, Example: "failed"}},
		Headers: map[string]*HeaderRef{
			"X-Common": {Value: &Header{Required: true, Schema: stringSchema}},
			"X-First":  {Value: &Header{Required: true, Schema: stringSchema}},
		},
	}
	second := &Response{
		Description: "Second error", Content: map[string]*MediaType{"application/json": {Schema: integerSchema, Example: 42}},
		Headers: map[string]*HeaderRef{"x-common": {Value: &Header{Required: true, Schema: integerSchema}}},
	}
	merged, err := mergeResponseAlternatives([]responseAlternative{{"second", second}, {"first", first}})
	require.NoError(t, err)
	require.Len(t, merged.Content["application/json"].Schema.AnyOf, 2)
	require.Len(t, merged.Content["application/json"].Examples, 2)
	require.Nil(t, merged.Content["application/json"].Example)
	require.True(t, merged.Headers["X-Common"].Value.Required)
	require.Len(t, merged.Headers["X-Common"].Value.Schema.AnyOf, 2)
	require.False(t, merged.Headers["X-First"].Value.Required)
	require.True(t, first.Headers["X-First"].Value.Required, "merging must not mutate a source contract")
	require.Equal(t, "failed", first.Content["application/json"].Example)
	duplicate, err := mergeResponseSchemas(merged.Content["application/json"].Schema, stringSchema)
	require.NoError(t, err)
	require.Equal(t, merged.Content["application/json"].Schema, duplicate)

	for _, tc := range []struct {
		name        string
		left, right *Response
	}{
		{"link", &Response{Links: map[string]*ResponseLinkRef{"retry": {Ref: "first"}}}, &Response{Links: map[string]*ResponseLinkRef{"retry": {Ref: "second"}}}},
		{"extension", &Response{Extensions: map[string]any{"x-mode": "first"}}, &Response{Extensions: map[string]any{"x-mode": "second"}}},
		{"component", &Response{ComponentName: "First"}, &Response{ComponentName: "Second"}},
		{"description", &Response{OmitDescription: true}, &Response{}},
		{"header", &Response{Headers: map[string]*HeaderRef{"X-Test": {Value: &Header{AllowReserved: true}}}}, &Response{Headers: map[string]*HeaderRef{"X-Test": {Value: &Header{}}}}},
		{"media type", &Response{Content: map[string]*MediaType{"application/json": {Extensions: map[string]any{"x-mode": 1}}}}, &Response{Content: map[string]*MediaType{"application/json": {Extensions: map[string]any{"x-mode": 2}}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, alternatives := range [][]responseAlternative{{{"first", tc.left}, {"second", tc.right}}, {{"second", tc.right}, {"first", tc.left}}} {
				_, err := mergeResponseAlternatives(alternatives)
				require.ErrorContains(t, err, "conflicting response "+tc.name)
			}
		})
	}
}
