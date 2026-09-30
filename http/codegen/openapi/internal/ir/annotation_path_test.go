package ir

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	commontestdata "github.com/CaliLuke/loom/codegen/testdata"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
)

type annotationPathStep struct {
	Ref     string
	Example any
}

func TestBytesPreservesEntireBaselineAnnotationPath(t *testing.T) {
	for _, neutral := range []bool{true, false} {
		t.Run(map[bool]string{true: "neutral", false: "planned"}[neutral], func(t *testing.T) {
			root := codegen.RunDSL(t, commontestdata.DeclarationAuthorityBytesDSL)
			var schemas map[string]*Schema
			if neutral {
				document, err := BuildDocument(root.API, root.Types, root.ResultTypes)
				require.NoError(t, err)
				schemas = document.Components.Schemas
			} else {
				a := NewAnalyzer(root.API.ExampleGenerator, false)
				prepared, err := representation.PrepareService(root.API.HTTP.Services[0], nil)
				require.NoError(t, err)
				for _, endpoint := range prepared.Endpoints {
					analyzeEndpointBodies(a, endpoint)
				}
				for _, endpoint := range prepared.Endpoints {
					analyzeAsyncSchemas(a, endpoint)
				}
				a.finalizeRepresentations()
				schemas = a.schemas
			}
			assert.Contains(t, schemas, "AuthorityBase", "the original canonical object must not be displaced by a new alias annotation layer")
			envelope := annotationObject(t, schemas["AuthorityEnvelope"], schemas)
			for _, field := range []string{"entries", "items"} {
				current := envelope.Properties[field].Items
				if field == "entries" {
					current = envelope.Properties[field].AdditionalProperties.Schema
				}
				var actual []annotationPathStep
				seen := map[*Schema]bool{}
				for current != nil && !seen[current] {
					seen[current] = true
					actual = append(actual, annotationPathStep{current.Ref, current.Example})
					if current.Ref == "" {
						break
					}
					name, ok := schemaComponentName(current.Ref)
					require.True(t, ok)
					current = schemas[name]
				}
				terminal := map[string]any{"blob": "RGVzZXJ1bnQgY3VtIG1vZGkgcXVhbSBkZWxlbml0aSBmYWNlcmUu", "count": 4069425969333182118}
				expected := []annotationPathStep{{Ref: toRef("AuthorityBase_2977d6da5f160a78")}, {Example: terminal}}
				if neutral && field == "items" {
					expected = []annotationPathStep{{Ref: toRef("AuthorityBase_2977d6da5f160a78_2")}, {Example: map[string]any{"blob": "RWxpZ2VuZGkgcXVpYSBkZWxlbml0aSBmdWdpYXQgcG9ycm8gdXQgdGVtcG9yYS4=", "count": 4575513339977884504}}}
				}
				if neutral {
					// Independently captured parent IR retains pure alias references;
					// the public renderer later collapses them without annotations.
					aliases := make([]annotationPathStep, 0, 2+len(expected))
					aliases = append(aliases, annotationPathStep{Ref: toRef("AuthorityAliasOne")})
					if field == "items" {
						aliases = append(aliases[:0], annotationPathStep{Ref: toRef("AuthorityAliasTwo")}, annotationPathStep{Ref: toRef("AuthorityAliasOne_2977d6da5f160a78")})
					}
					expected = append(aliases, expected...)
				}
				if !neutral {
					expected = []annotationPathStep{{Ref: toRef("AuthorityAliasOne")}, {Ref: toRef("AuthorityBase_2977d6da5f160a78"), Example: map[string]any{"blob": "VXQgdGVuZXR1ciBzaXQu", "count": 3748162180375200810}}, {Example: terminal}}
					if field == "items" {
						expected = append([]annotationPathStep{{Ref: toRef("AuthorityAliasTwo")}, {Ref: toRef("AuthorityAliasOne"), Example: map[string]any{"blob": "TmlzaSBuYXR1cy4=", "count": 7371582765372296462}}}, expected[1:]...)
					}
				}
				assert.Equal(t, expected, actual, "%s preserves every original annotation and absent annotation along its entire reachable reference path", field)
			}
		})
	}
}
