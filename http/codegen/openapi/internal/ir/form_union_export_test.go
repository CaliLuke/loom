package ir

import (
	"strings"
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/stretchr/testify/require"
)

func TestTaggedFormUnionHonestExport(t *testing.T) {
	root := codegen.RunDSL(t, testdata.DiscriminatorHintDSL)
	document, err := BuildDocument(root.API, root.Types, root.ResultTypes)
	require.NoError(t, err)
	for _, tc := range []struct{ name, method, media, style string }{
		{"json-field", "JSONField", "application/x-www-form-urlencoded", "application/json"},
		{"deep-object-object", "ObjectField", "application/x-www-form-urlencoded", "deepObject"},
		{"deep-object-map", "MapField", "application/x-www-form-urlencoded", "deepObject"},
		{"deep-object-full-object", "FullObjectField", "application/x-www-form-urlencoded", "deepObject"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			media := hintRequestBody(t, document, tc.method).Content[tc.media]
			key := "style"
			if tc.style == "application/json" {
				key = "contentType"
			}
			require.Equal(t, []string{tc.style}, media.Metadata["openapi:encoding:Values:"+key])
		})
	}
	t.Run("multipart-default-json-field", func(t *testing.T) {
		media := hintRequestBody(t, document, "MultipartField").Content["multipart/form-data"]
		root := resolvedHintSchema(t, document, media.Schema)
		field := resolvedHintSchema(t, document, root.Properties["Values"])
		require.NotNil(t, field.Discriminator)
		branch := resolvedHintSchema(t, document, &Schema{Ref: field.Discriminator.Mapping["authorization_code"]})
		require.Contains(t, branch.Properties, "data")
		require.Empty(t, media.Metadata["openapi:encoding:Values:style"], "implicit multipart JSON needs no styled override")
	})
	t.Run("flat-tagged-root", func(t *testing.T) {
		media := hintRequestBody(t, document, "FlatRoot").Content["application/x-www-form-urlencoded"]
		require.Equal(t, map[string]any{"grant_type": "authorization_code", "code": "abc"}, media.Example)
		union := resolvedHintSchema(t, document, media.Schema)
		require.NotNil(t, union.Discriminator)
		branch := resolvedHintSchema(t, document, &Schema{Ref: union.Discriminator.Mapping["authorization_code"]})
		require.Len(t, branch.AllOf, 1)
		payload := resolvedHintSchema(t, document, branch.AllOf[0])
		require.Contains(t, payload.Properties, "code")
		require.Contains(t, branch.Properties, "grant_type")
		require.NotContains(t, branch.Properties, "data")
		require.ElementsMatch(t, []string{"code", "grant_type"}, payload.Required)
		require.Equal(t, []string{"grant_type"}, branch.Required)
		require.Equal(t, union.Discriminator.Mapping["authorization_code"], union.Discriminator.Mapping["code"])
	})
	t.Run("shared-json-root-occurrences", func(t *testing.T) {
		media := hintRequestBody(t, document, "JSONRoot").Content["application/json"]
		require.Equal(t, map[string]any{"grant_type": "authorization_code", "data": map[string]any{"code": "abc"}}, media.Example)
		union := resolvedHintSchema(t, document, media.Schema)
		branch := resolvedHintSchema(t, document, &Schema{Ref: union.Discriminator.Mapping["authorization_code"]})
		require.Contains(t, branch.Properties, "data")
		require.NotContains(t, branch.Properties, "code")
	})
}

func resolvedHintSchema(t *testing.T, document *Document, schema *Schema) *Schema {
	t.Helper()
	if schema.Ref != "" {
		name, ok := schemaComponentName(schema.Ref)
		require.True(t, ok)
		schema = document.Components.Schemas[name]
	}
	require.NotNil(t, schema)
	return schema
}

func hintRequestBody(t *testing.T, document *Document, method string) *RequestBody {
	t.Helper()
	reference := document.Paths["/"+method].Operations["POST"].RequestBody
	if reference.Ref != "" {
		reference = document.Components.RequestBodies[strings.TrimPrefix(reference.Ref, "#/components/requestBodies/")]
	}
	require.NotNil(t, reference)
	require.NotNil(t, reference.Value)
	return reference.Value
}
