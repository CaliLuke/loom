package openapiv3_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strings"
	"testing"

	commontestdata "github.com/CaliLuke/loom/codegen/testdata"
	"github.com/stretchr/testify/require"
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
			for name := range schemas {
				require.False(t, strings.HasPrefix(name, "AuthorityAlias"), "no new observable alias component: %s", name)
			}
			var envelope struct {
				Properties map[string]struct {
					Items                jsontext.Value `json:"items"`
					AdditionalProperties jsontext.Value `json:"additionalProperties"`
				} `json:"properties"`
			}
			require.NoError(t, json.Unmarshal(schemas["AuthorityEnvelope"], &envelope))
			for _, test := range []struct {
				name, component, blob string
				count                 int64
			}{
				{"base", "AuthorityBase", "QXV0IGV0IGl0YXF1ZSBtb2xlc3RpYXMgZGljdGEu", 8644872746071626424},
				{"entries", "AuthorityBase_2977d6da5f160a78", "RGVzZXJ1bnQgY3VtIG1vZGkgcXVhbSBkZWxlbml0aSBmYWNlcmUu", 4069425969333182118},
				{"items", "AuthorityBase_2977d6da5f160a78_2", "RWxpZ2VuZGkgcXVpYSBkZWxlbml0aSBmdWdpYXQgcG9ycm8gdXQgdGVtcG9yYS4=", 4575513339977884504},
			} {
				var component struct {
					Type    string `json:"type"`
					Ref     string `json:"$ref"`
					Example struct {
						Blob  string `json:"blob"`
						Count int64  `json:"count"`
					} `json:"example"`
					Properties map[string]jsontext.Value `json:"properties"`
				}
				require.NoError(t, json.Unmarshal(schemas[test.component], &component))
				require.Equal(t, "object", component.Type)
				require.Empty(t, component.Ref, "canonical annotation owner remains an object, not an added alias layer")
				require.Equal(t, test.blob, component.Example.Blob)
				require.Equal(t, test.count, component.Example.Count)
				var blob struct {
					ContentEncoding string         `json:"contentEncoding"`
					Not             jsontext.Value `json:"not"`
				}
				require.NoError(t, json.Unmarshal(component.Properties["blob"], &blob))
				require.Equal(t, "base64", blob.ContentEncoding)
				require.NotEmpty(t, blob.Not)
				if test.name == "base" {
					continue
				}
				slot := envelope.Properties[test.name].Items
				if test.name == "entries" {
					slot = envelope.Properties[test.name].AdditionalProperties
				}
				var use map[string]jsontext.Value
				require.NoError(t, json.Unmarshal(slot, &use))
				require.Len(t, use, 1, "parent use has only a reference, no annotation or assertion layer")
				var ref string
				require.NoError(t, json.Unmarshal(use["$ref"], &ref))
				require.Equal(t, "#/components/schemas/"+test.component, ref)
			}
		})
	}
}
