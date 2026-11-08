package openapiv3_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	commontestdata "github.com/CaliLuke/loom/codegen/testdata"
)

func TestRenderedByteProjectionPreservesAnnotationPaths(t *testing.T) {
	for _, version := range []struct{ target, version string }{{"3.1", "3.1.1"}, {"3.2", "3.2.0"}} {
		t.Run(version.target, func(t *testing.T) {
			artifacts := renderOpenAPIArtifactsForVersion(t, commontestdata.DeclarationAuthorityBytesDSL, version.target, version.version)
			var document struct {
				Components struct {
					Schemas map[string]jsontext.Value `json:"schemas"`
				} `json:"components"`
			}
			require.NoError(t, json.Unmarshal(artifacts.JSON, &document))
			schemas := document.Components.Schemas
			envelope := resolveRenderedAnnotationRef(t, schemas, renderedAnnotationField(t, artifacts.JSON, "paths", "/ordinary", "post", "responses", "200", "content", "application/json", "schema"))
			for _, test := range []struct {
				name, blob string
				count      int64
			}{
				{"base", "SWQgZmFjaWxpcy4=", 1698367705951450038},
				{"entries", "SXBzdW0gYXV0IGFkIHF1aWJ1c2RhbS4=", 7317769671929827049},
				{"items", "TWFnbmkgZHVjaW11cyBhdXQgcmVwZWxsZW5kdXMgZG9sb3Iu", 4929712490902895459},
			} {
				slot := renderedAnnotationField(t, envelope, "properties", test.name)
				if test.name == "entries" {
					slot = renderedAnnotationField(t, slot, "additionalProperties")
				}
				if test.name == "items" {
					slot = renderedAnnotationField(t, slot, "items")
				}
				var use map[string]jsontext.Value
				require.NoError(t, json.Unmarshal(slot, &use))
				require.Len(t, use, 1, "parent use has only a reference, no annotation or assertion layer")
				var component struct {
					Type    string `json:"type"`
					Ref     string `json:"$ref"`
					Example struct {
						Blob  string `json:"blob"`
						Count int64  `json:"count"`
					} `json:"example"`
					Properties map[string]jsontext.Value `json:"properties"`
				}
				require.NoError(t, json.Unmarshal(resolveRenderedAnnotationRef(t, schemas, slot), &component))
				require.Equal(t, "object", component.Type)
				require.Empty(t, component.Ref, "selected annotation owner remains an object, not an added alias layer")
				require.Equal(t, test.blob, component.Example.Blob)
				require.Equal(t, test.count, component.Example.Count)
				var blob struct {
					ContentEncoding string         `json:"contentEncoding"`
					Not             jsontext.Value `json:"not"`
				}
				require.NoError(t, json.Unmarshal(component.Properties["blob"], &blob))
				require.Equal(t, "base64", blob.ContentEncoding)
				require.NotEmpty(t, blob.Not)
			}
		})
	}
}

func renderedAnnotationField(t *testing.T, raw jsontext.Value, path ...string) jsontext.Value {
	t.Helper()
	for _, name := range path {
		var object map[string]jsontext.Value
		require.NoError(t, json.Unmarshal(raw, &object))
		require.Contains(t, object, name)
		raw = object[name]
	}
	return raw
}

func resolveRenderedAnnotationRef(t *testing.T, schemas map[string]jsontext.Value, use jsontext.Value) jsontext.Value {
	t.Helper()
	var ref string
	require.NoError(t, json.Unmarshal(renderedAnnotationField(t, use, "$ref"), &ref))
	require.True(t, strings.HasPrefix(ref, "#/components/schemas/"))
	name := strings.TrimPrefix(ref, "#/components/schemas/")
	require.NotContains(t, name, "/")
	name = strings.ReplaceAll(strings.ReplaceAll(name, "~1", "/"), "~0", "~")
	require.Contains(t, schemas, name, "the actual public reference must resolve")
	return schemas[name]
}
