package openapiimport

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanonicalSchemaTargetOwnsOriginalOccurrence(t *testing.T) {
	for _, pointer := range []string{
		"/components/schemas/Container/properties/id",
		"/components/schemas/Choice/oneOf/0",
		"/paths/~1inline/get/responses/200/content/application~1json/schema",
		"/components/schemas/Container/properties/node",
	} {
		t.Run(pointer, func(t *testing.T) {
			resolved, diagnostics, err := resolveDescription(Source{Bytes: localSchemaTargetSource("#" + pointer)})
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			bundle, err := bundleResolution(resolved)
			require.NoError(t, err)
			canonical := bundle.references["#/paths/~1value/get/responses/200/content/application~1json/schema/$ref"]
			require.NotEmpty(t, canonical)
			original, err := descriptionNode(bundle.root, pointer+"/$ref")
			require.NoError(t, err, "the authored Schema occurrence must consume its canonical declaration")
			require.Equal(t, canonical, original.Value)
			require.Equal(t, canonical, bundle.references["#"+pointer+"/$ref"])
			require.Equal(t, SourceLocation{Document: "entry", Pointer: "#" + pointer}, bundle.origins[canonical])
			require.Equal(t, bundle.origins[canonical], bundle.origins["#"+pointer+"/$ref"])
			body, err := descriptionNode(bundle.root, strings.TrimPrefix(canonical, "#")+"/type")
			require.NoError(t, err, "canonical declaration must materialize rather than reference itself")
			require.NotEmpty(t, body.Value)
		})
	}
}

func TestCanonicalSchemaOccurrenceKeepsAliasAssertions(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: Alias assertions, version: '1'}
paths:
  /value:
    get:
      responses:
        '200': {description: OK, content: {application/json: {schema: {$ref: '#/components/schemas/Holder/properties/value'}}}}
components:
  schemas:
    Base: {type: integer, format: int64}
    Alias: {$ref: '#/components/schemas/Holder/properties/value'}
    Holder:
      type: object
      properties:
        value: {$ref: '#/components/schemas/Base', description: Original assertion, minimum: 9007199254740993}
        other: {$ref: '#/components/schemas/Base', description: Original assertion, minimum: 9007199254740993}
`)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	schemas := make(map[string]*Schema)
	for _, named := range document.Components.Schemas {
		schemas[named.Name] = named.Schema
	}
	require.Len(t, schemas, 4)
	canonical := schemas["value"]
	require.NotNil(t, canonical)
	require.Equal(t, "#/components/schemas/Base", canonical.Ref)
	require.True(t, canonical.referenceSiblings)
	require.Equal(t, "Original assertion", canonical.Description)
	require.Equal(t, "9007199254740993", canonical.Minimum.String())
	require.Equal(t, "#/components/schemas/value", schemas["Alias"].Ref)
	require.NotSame(t, canonical, schemas["Alias"], "authored alias is its own declaration")
	for _, property := range schemas["Holder"].Properties {
		if property.Name == "value" {
			require.Equal(t, "#/components/schemas/value", property.Schema.Ref)
		} else {
			require.Equal(t, "#/components/schemas/Base", property.Schema.Ref,
				"structurally identical but unaddressed occurrence keeps its authored identity")
			require.Equal(t, "9007199254740993", property.Schema.Minimum.String())
		}
	}
	require.Equal(t, "#/components/schemas/value", document.Operations[0].Responses[0].Response.Schema.Ref)
	require.Equal(t, SourceLocation{Document: "entry", Pointer: "#/components/schemas/Holder/properties/value/minimum"},
		document.bundle.origins["#/components/schemas/value/minimum"])
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(design), `Minimum("9007199254740993")`)
}
