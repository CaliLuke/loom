package openapiimport

import (
	"strings"
	"testing"

	"github.com/CaliLuke/loom/internal/openapiref"
	"github.com/stretchr/testify/require"
)

func TestReusableReferencePlanning(t *testing.T) {
	for _, kind := range []string{"requestBodies", "responses", "headers"} {
		t.Run(kind, func(t *testing.T) {
			prefix := "#/components/" + kind + "/"
			description := "base"
			values := map[string]openapiref.Object{
				"Alias":   {Ref: prefix + "ZTarget", Summary: new("alias")},
				"ZTarget": {Description: &description},
				"Cycle":   {Ref: prefix + "Cycle"},
			}
			index := reusableIndex[openapiref.Object]{kind, values, map[string]bool{}, func(value openapiref.Object) openapiref.Object {
				return value
			}}
			for _, tc := range []struct {
				name, ref string
				rejected  bool
			}{
				{"forward alias", prefix + "Alias", false},
				{"encoded target", prefix + "%5ATarget", false},
				{"missing", prefix + "Absent", true},
				{"cycle", prefix + "Cycle", true},
				{"wrong kind", "#/components/schemas/ZTarget", true},
				{"invalid key", prefix + "bad~1name", true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					value, chain, err := index.resolve(openapiref.Object{Ref: tc.ref, Description: new("")})
					if tc.rejected {
						require.Error(t, err)
						return
					}
					require.NoError(t, err)
					require.Equal(t, "base", *value.Description)
					require.Equal(t, "", *chain[0].Description)
					require.Equal(t, prefix+"ZTarget", chain[len(chain)-1].Ref)
					require.Equal(t, "base", description)
					require.Equal(t, prefix+"ZTarget", values["Alias"].Ref)
				})
			}
		})
	}
}

func TestReusableReferenceAnnotationVersions(t *testing.T) {
	for _, version := range []string{"3.0.4", "3.1.2", "3.2.0"} {
		source := []byte("openapi: " + version + `
info: {title: References, version: "1"}
paths:
 /value:
  get:
   responses:
    "204": {$ref: '#/components/responses/Done', description: '', summary: 'local'}
components:
 responses:
  Done: {description: 'base'}
`)
		document, diagnostics, err := Analyze(source)
		require.NoError(t, err)
		require.Empty(t, diagnostics)
		occurrence := document.Operations[0].Responses[0].Response
		if strings.HasPrefix(version, "3.0.") {
			require.Nil(t, occurrence.ReferenceDescription)
			require.Nil(t, occurrence.ReferenceSummary)
		} else {
			require.Equal(t, new(""), occurrence.ReferenceDescription)
			require.Equal(t, new("local"), occurrence.ReferenceSummary)
		}
		require.Equal(t, "base", document.Components.Responses[0].Response.Description)
	}
}
