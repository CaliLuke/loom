package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiscriminatorMappingEdges(t *testing.T) {
	source := resolverFiles(t, []byte(`openapi: 3.2.0
info: {title: Mapping, version: '1'}
paths: {}
components:
  schemas:
    Cat: {type: string}
    Choice:
      type: object
      discriminator:
        propertyName: kind
        mapping:
          name: Cat
          uri: ./Cat
          external: https://E.TEST:443/model#/Dog
          missing: Unknown
`), map[string]string{"Cat": "type: string"})
	resolved, diagnostics, err := resolveDescription(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics, "mapping admission belongs to #1005")
	_, err = bundleResolution(resolved)
	require.NoError(t, err)
	outcomes := resolved.discriminatorEdges()
	require.Len(t, outcomes, 4)
	byPath := make(map[string]mappingOutcome)
	for _, outcome := range outcomes {
		byPath[outcome.source.pointer] = outcome
	}
	base := "/components/schemas/Choice/discriminator/mapping/"
	require.Equal(t, "Cat", byPath[base+"name"].entry)
	require.Equal(t, "#/components/schemas/Cat_2", byPath[base+"uri"].bundled)
	require.Equal(t, "https://e.test/model#/Dog", byPath[base+"external"].external)
	require.NotNil(t, byPath[base+"missing"].failure)
	_, diagnostics, _, err = AnalyzeSelected(source, Selection{})
	require.NoError(t, err)
	requireDiagnosticCode(t, diagnostics, "schema-discriminator")
}
