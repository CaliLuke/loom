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
				require.NoError(t, representation.PrepareServiceExamples(
					prepared, root.API.HTTP.Services[0], root.API.ExampleGenerator,
				))
				for _, endpoint := range prepared.Endpoints {
					require.NoError(t, representation.PrepareEndpointExamples(endpoint, root.API.ExampleGenerator))
				}
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
				expected := []annotationPathStep{{Ref: toRef("AuthorityBase_2977d6da5f160a78")}, {}}
				if neutral && field == "items" {
					expected = []annotationPathStep{{Ref: toRef("AuthorityBase_2977d6da5f160a78_2")}, {}}
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
					expected = []annotationPathStep{{Ref: toRef("AuthorityAliasOne")}, {Ref: toRef("AuthorityBase_2977d6da5f160a78")}, {}}
					if field == "items" {
						expected = append([]annotationPathStep{{Ref: toRef("AuthorityAliasTwo")}, {Ref: toRef("AuthorityAliasOne")}}, expected[1:]...)
					}
				}
				require.Len(t, actual, len(expected))
				for index := range expected {
					assert.Equal(t, expected[index].Ref, actual[index].Ref,
						"%s preserves reference step %d", field, index)
				}
				assert.Nil(t, actual[0].Example,
					"%s keeps the outer reference free of its target component's annotation", field)
				start := 1
				if neutral {
					start = len(actual) - 1
				}
				for index := start; index < len(actual); index++ {
					assert.NotNil(t, actual[index].Example,
						"%s retains the representative owned by annotation step %d", field, index)
				}
			}
		})
	}
}
