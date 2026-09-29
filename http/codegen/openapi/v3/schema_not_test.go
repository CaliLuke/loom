package openapiv3

import (
	"encoding/json/v2"
	"testing"

	"github.com/CaliLuke/loom/http/codegen/openapi"
	openapiir "github.com/CaliLuke/loom/http/codegen/openapi/internal/ir"
	"github.com/stretchr/testify/require"
)

func TestSchemaNotRoundTrip(t *testing.T) {
	source := &openapi.Schema{Type: openapi.String, ContentEncoding: "base64", Not: &openapi.Schema{}}
	copy := openapiir.RenderSchema(schemaToIR(source))
	data, err := json.Marshal(copy)
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"string","contentEncoding":"base64","not":{}}`, string(data))
	copy.Not.Pattern = "changed"
	require.Empty(t, source.Not.Pattern)
}

func TestSchemaNotWalkers(t *testing.T) {
	schema := &openapi.Schema{Ref: "#/components/schemas/Parent", Not: &openapi.Schema{Ref: "#/components/schemas/Before"}}
	var refs []string
	collectSchemaRefs(schema, func(ref string) {
		refs = append(refs, ref)
	})
	require.Equal(t, []string{"#/components/schemas/Parent", "#/components/schemas/Before"}, refs)
	rewriteSchemaRefs(schema, func(string) string {
		return "#/components/schemas/After"
	})
	require.Equal(t, "#/components/schemas/After", schema.Not.Ref)
	require.Equal(t, "#/components/schemas/After", schema.Ref)
	require.True(t, isPureRefSchema(&openapi.Schema{Ref: "#/components/schemas/Other"}))
	require.False(t, isPureRefSchema(&openapi.Schema{Ref: "#/components/schemas/Other", Not: &openapi.Schema{}}))
	schema.Not = &openapi.Schema{XML: &openapi.XML{NodeType: "text"}, ContentEncoding: "base64"}
	filterSchemaCompatibility(schema, make(map[*openapi.Schema]struct{}))
	require.Empty(t, schema.Not.XML.NodeType)
	require.Equal(t, "base64", schema.Not.ContentEncoding)
}
