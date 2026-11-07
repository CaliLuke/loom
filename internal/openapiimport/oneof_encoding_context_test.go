package openapiimport

import (
	"encoding/json/v2"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/httpencoding"
)

func TestOneOfRootEncodingContext(t *testing.T) {
	for _, tc := range []struct {
		name, schema, encoding string
		accepted               bool
	}{
		{"inline", `{"oneOf":[{"type":"object","properties":{"value":{"oneOf":[{"type":"string"},{"type":"integer"}]}}},{"type":"object","properties":{"other":{"type":"string"}}}]}`, `{"value":{"contentType":"application/json"}}`, true},
		{"referenced", `{"$ref":"#/components/schemas/Choice"}`, `{"value":{"contentType":"application/json"}}`, true},
		{"nested union", `{"oneOf":[{"$ref":"#/components/schemas/Choice"},{"type":"object","properties":{"third":{"type":"integer"}}}]}`, `{"value":{"contentType":"application/json"}}`, true},
		{"empty encoding keeps captured default", `{"oneOf":[{"type":"object","properties":{"value":{"type":"array","oneOf":[{"type":"array","items":{"type":"string"}},{"type":"array","items":{"type":"integer"}}]}}},{"type":"object","properties":{"other":{"type":"string"}}}]}`, `{"value":{}}`, true},
		{"missing override", `{"$ref":"#/components/schemas/Choice"}`, `{}`, false},
		{"unknown property", `{"$ref":"#/components/schemas/Choice"}`, `{"missing":{"contentType":"application/json"}}`, false},
		{"wrong explicit codec", `{"$ref":"#/components/schemas/Choice"}`, `{"value":{"contentType":"application/octet-stream"}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := rootEncodingSource(tc.schema, tc.encoding)
			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			if !tc.accepted {
				requireDiagnosticCode(t, diagnostics, "media-encoding")
				switch tc.name {
				case "unknown property":
					require.Contains(t, diagnostics.Error(), "/content/application~1x-www-form-urlencoded/encoding/missing:")
				case "wrong explicit codec":
					require.Contains(t, diagnostics.Error(), `field media type "application/octet-stream"`)
				case "missing override":
					require.Contains(t, diagnostics.Error(), `field media type "application/octet-stream"`)
				}
				return
			}
			require.Empty(t, diagnostics)
			design, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			if tc.name == "empty encoding keeps captured default" {
				require.Contains(t, string(design), `Meta("http:encoding:sourceType", "array")`)
			} else {
				require.Contains(t, string(design), `Meta("openapi:encoding:value:contentType", "application/json")`)
			}
		})
	}
}

func TestOneOfRootEncodingContextIsolation(t *testing.T) {
	var source map[string]any
	require.NoError(t, json.Unmarshal(rootEncodingSource(`{"$ref":"#/components/schemas/Choice"}`, `{"value":{"contentType":"application/json"}}`), &source))
	paths := source["paths"].(map[string]any)
	paths["/without-encoding"] = map[string]any{"post": map[string]any{
		"operationId": "withoutEncoding",
		"requestBody": map[string]any{"content": map[string]any{
			"application/x-www-form-urlencoded": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/Choice"}},
		}},
		"responses": map[string]any{"204": map[string]any{"description": "done"}},
	}}
	wire, err := json.Marshal(source, json.Deterministic(true))
	require.NoError(t, err)
	document, diagnostics, err := Analyze(wire)
	require.NoError(t, err)
	require.Len(t, diagnostics, 1)
	require.Equal(t, "media-encoding", diagnostics[0].Code)
	require.Contains(t, diagnostics[0].Message, "octet-stream")
	for _, named := range document.Components.Schemas {
		require.Empty(t, named.Schema.MediaEncoding, "the explicit occurrence must not mutate declaration %s", named.Name)
	}
}

func TestOneOfRootEncodingOverridesCapturedDefault(t *testing.T) {
	schema := `{"oneOf":[{"type":"object","properties":{"value":{"type":"string","oneOf":[{"type":"string","const":"a"},{"type":"string","const":"b"}]}}},{"type":"object","properties":{"other":{"type":"string"}}}]}`
	document, diagnostics, err := Analyze(rootEncodingSource(schema, `{"value":{"contentType":"application/json"}}`))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	// Capture retains intrinsic declaration defaults. The root occurrence
	// overrides them in the traversal context, rather than rewriting types.
	planner := documentPlanner{schemas: planDocument(document).schemas}
	properties := make(map[string]*Schema)
	planner.formProperties(document.Operations[0].RequestBody.Schema.OneOf[0], properties, make(map[*Schema]bool))
	require.Equal(t, "string", properties["value"].sourceType)
	require.Equal(t, "application/json", document.Operations[0].RequestBody.Schema.MediaEncoding["value"].MediaType)
}

func TestOneOfRootEncodingGenerates(t *testing.T) {
	document, diagnostics, err := Analyze(rootEncodingSource(`{"$ref":"#/components/schemas/Choice"}`, `{"value":{"contentType":"application/json"}}`))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	requireRenderedDesignGenerates(t, design)
}

func rootEncodingSource(schema, encoding string) []byte {
	return fmt.Appendf(nil, `{"openapi":"3.2.0","info":{"title":"Root codecs","version":"1"},"paths":{"/value":{"post":{"operationId":"value","requestBody":{"required":true,"content":{"application/x-www-form-urlencoded":{"schema":%s,"encoding":%s}}},"responses":{"204":{"description":"done"}}}}},"components":{"schemas":{"Choice":{"oneOf":[{"type":"object","properties":{"value":{"oneOf":[{"type":"string"},{"type":"integer"}]}}},{"type":"object","properties":{"other":{"type":"string"}}}]}}}}`, schema, encoding)
}

func TestMediaEncodingContextRetainsExplicitFalse(t *testing.T) {
	yes, no := true, false
	inner := map[string]httpencoding.Serialization{"value": {MediaType: "application/json", Explode: &yes, AllowReserved: &yes}}
	outer := map[string]httpencoding.Serialization{"value": {Style: "form", Explode: &no, AllowReserved: &no}}
	effective := mergeMediaEncoding(outer, inner)
	require.Equal(t, "application/json", effective["value"].MediaType)
	require.Equal(t, "form", effective["value"].Style)
	require.False(t, *effective["value"].Explode)
	require.False(t, *effective["value"].AllowReserved)
	effective["value"] = httpencoding.Serialization{}
	require.Equal(t, "application/json", inner["value"].MediaType)
	require.Equal(t, "form", outer["value"].Style)
}
