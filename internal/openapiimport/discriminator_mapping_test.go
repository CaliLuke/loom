package openapiimport

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiscriminatorMappingSourceIdentity(t *testing.T) {
	entry := []byte(`openapi: 3.2.0
info: {title: Mapping, version: '1'}
paths: {}
components:
  schemas:
    Cat: {type: object, properties: {kind: {type: string}}, required: [kind]}
    Choice:
      oneOf:
        - {$ref: '#/components/schemas/Cat'}
        - {$ref: './pets.yaml#/components/schemas/Dog'}
      discriminator:
        propertyName: kind
        mapping: {cat: Cat, dog: './pets.yaml#/components/schemas/Dog', hound: './pets.yaml#/components/schemas/Dog'}
        defaultMapping: 'https://E.TEST:443/model#/Other'
`)
	source := resolverFiles(t, entry, map[string]string{"pets.yaml": "components:\n  schemas:\n    Dog: {type: object, properties: {kind: {type: string}}, required: [kind]}\n"})
	doc, diagnostics, _, err := AnalyzeSelected(source, Selection{})
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(doc, Options{PackageName: "design"})
	require.NoError(t, err)
	for _, fragment := range []string{`Meta("openapi:discriminator:mapping", "dog")`, `Meta("openapi:discriminator:mapping", "hound")`, `Meta("openapi:discriminator:defaultMapping", "https://e.test/model#/Other")`} {
		require.Contains(t, string(design), fragment)
	}
	require.Less(t, strings.Index(string(design), `"dog")`), strings.Index(string(design), `"hound")`))
	// A supplied external fallback becomes the same retained canonical schema
	// identity as ordinary references and mapping outcomes.
	localFallback := strings.Replace(string(entry), "https://E.TEST:443/model#/Other", "./pets.yaml#/components/schemas/Dog", 1)
	fallbackSource := resolverFiles(t, []byte(localFallback), map[string]string{"pets.yaml": "components:\n  schemas:\n    Dog: {type: object, properties: {kind: {type: string}}, required: [kind]}\n"})
	fallbackDoc, fallbackDiagnostics, _, err := AnalyzeSelected(fallbackSource, Selection{})
	require.NoError(t, err)
	require.Empty(t, fallbackDiagnostics)
	fallbackDesign, err := Render(fallbackDoc, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(fallbackDesign), `Meta("openapi:discriminator:defaultMapping", "#/components/schemas/Dog")`)
	// Moving the hint into an external document must still resolve bare names in
	// the entry document, and its authored path must survive bundling/promotion.
	external := strings.ReplaceAll(string(entry), "'./pets.yaml#/components/schemas/Dog'", "'./dog.yaml#/components/schemas/Dog'")
	external = strings.ReplaceAll(external, "    Cat: {type: object, properties: {kind: {type: string}}, required: [kind]}\n", "")
	external = strings.ReplaceAll(external, "        - {$ref: '#/components/schemas/Cat'}", "        - {$ref: './openapi.yaml#/components/schemas/Cat'}")
	nested := []byte("openapi: 3.2.0\ninfo: {title: Entry, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    Cat: {type: object, properties: {kind: {type: string}}, required: [kind]}\n    Choice: {$ref: './pets.yaml#/components/schemas/Choice'}\n")
	source = resolverFiles(t, nested, map[string]string{"pets.yaml": external, "dog.yaml": "components:\n  schemas:\n    Dog: {type: object, properties: {kind: {type: string}}, required: [kind]}\n"})
	doc, diagnostics, _, err = AnalyzeSelected(source, Selection{})
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	_, err = Render(doc, Options{PackageName: "design"})
	require.NoError(t, err)
}

func TestDiscriminatorInlineMappingAndDefaultClosure(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: Inline, version: '1'}
paths:
  /pets:
    get:
      tags: [pets]
      responses:
        '200':
          description: result
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Pet'}
components:
  schemas:
    Pet:
      oneOf:
        - {type: object, properties: {kind: {type: string, enum: [cat]}, name: {type: string}}, required: [kind, name]}
        - {type: object, properties: {kind: {type: string, enum: [dog]}, age: {type: integer}}, required: [kind, age]}
      discriminator:
        propertyName: kind
        mapping: {cat: '#/components/schemas/Pet/oneOf/0', dog: '#/components/schemas/Pet/oneOf/1'}
        defaultMapping: Other
    Other: {type: string}
    Unused: {type: string}
`)
	doc, diagnostics, _, err := AnalyzeSelected(Source{Bytes: source}, Selection{Tags: []string{"pets"}})
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	names := map[string]bool{}
	for _, named := range doc.Components.Schemas {
		names[named.Name] = true
	}
	require.True(t, names["Other"], "defaultMapping is a retained schema dependency")
	require.False(t, names["Unused"])
	design, err := Render(doc, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(design), `Meta("openapi:discriminator:mapping", "cat")`)
	require.Contains(t, string(design), `Meta("openapi:discriminator:mapping", "dog")`)
}

func TestDiscriminatorMappingPreservesAliasTargetIdentity(t *testing.T) {
	source := string(discriminatorDocument("3.2.0", `{propertyName: kind, mapping: {alias: Alias, dog: Dog}}`, `[{$ref: '#/components/schemas/Alias'}, {$ref: '#/components/schemas/Dog'}]`, "")) + "    Alias: {$ref: '#/components/schemas/Cat'}\n"
	document, diagnostics, err := Analyze([]byte(source))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(design), `Meta("openapi:discriminator:mapping", "alias")`)
	// A mapping to the alias's target is not a mapping to the listed Alias schema.
	source = strings.Replace(source, "alias: Alias", "alias: Cat", 1)
	_, diagnostics, err = Analyze([]byte(source))
	require.NoError(t, err)
	requireDiagnosticCode(t, diagnostics, "schema-discriminator")
}

func TestDiscriminatorOccurrenceMetadataOwnership(t *testing.T) {
	source := string(discriminatorDocument("3.2.0", `{propertyName: kind, mapping: {cat: Cat, dog: Dog}, x-owner: original}`, `[{$ref: '#/components/schemas/Cat'}, {$ref: '#/components/schemas/Dog'}]`, "      description: Original union")) + `    NullablePet:
      anyOf: [{$ref: '#/components/schemas/Pet'}, {type: 'null'}]
      description: Nullable occurrence
      x-occurrence: nullable
    ClosedPet:
      $ref: '#/components/schemas/Pet'
      description: Closed occurrence
      unevaluatedProperties: false
    Other:
      oneOf: [{$ref: '#/components/schemas/Cat'}, {$ref: '#/components/schemas/Dog'}]
      discriminator: {propertyName: kind, mapping: {kitty: Cat, hound: Dog}, x-owner: other}
`
	document, diagnostics, err := Analyze([]byte(source))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	schemas := map[string]*Schema{}
	for _, named := range document.Components.Schemas {
		schemas[named.Name] = named.Schema
	}
	require.NotSame(t, schemas["Pet"].Discriminator, schemas["Other"].Discriminator)
	require.Equal(t, "original", schemas["Pet"].Discriminator.Extensions["x-owner"])
	require.Equal(t, "other", schemas["Other"].Discriminator.Extensions["x-owner"])
	require.Nil(t, schemas["NullablePet"].Discriminator)
	require.Nil(t, schemas["ClosedPet"].Discriminator)
	require.Equal(t, "Original union", schemas["Pet"].Description)
	require.False(t, schemas["Pet"].UnionClosed)
	require.True(t, schemas["ClosedPet"].UnionClosed)
	require.Equal(t, "#/components/schemas/Pet", schemas["Pet"].sourcePath)
	require.Equal(t, "#/components/schemas/NullablePet", schemas["NullablePet"].sourcePath)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(design), `Description("Nullable occurrence")`)
	require.Contains(t, string(design), `Description("Closed occurrence")`)
}

func TestDiscriminatorInlineTargetsKeepAuthoredIdentity(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: Inline identities, version: '1'}
paths: {}
components:
  schemas:
    Choice:
      oneOf:
        - {type: object, properties: {kind: {type: string}}, required: [kind]}
        - {type: object, properties: {kind: {type: string}}, required: [kind]}
      discriminator:
        propertyName: kind
        mapping: {first: '#/components/schemas/Choice/oneOf/0', second: '#/components/schemas/Choice/oneOf/1'}
`)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	var choice *Schema
	for _, named := range document.Components.Schemas {
		if named.Name == "Choice" {
			choice = named.Schema
		}
	}
	require.NotNil(t, choice)
	require.Len(t, choice.OneOf, 2)
	require.Len(t, choice.Discriminator.Explicit, 2)
	require.NotEqual(t, discriminatorIdentity(document.bundle, choice.Discriminator.Explicit[0].Target), discriminatorIdentity(document.bundle, choice.Discriminator.Explicit[1].Target))
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(string(design), `Meta("openapi:discriminator:mapping", "first")`))
	require.Equal(t, 1, strings.Count(string(design), `Meta("openapi:discriminator:mapping", "second")`))
}

func TestDiscriminatorUnnamedAlternativesInventNoMappings(t *testing.T) {
	source := discriminatorDocument("3.2.0", `{propertyName: kind}`, `[{type: object, properties: {kind: {type: string}, name: {type: string}}, required: [kind, name]}, {type: object, properties: {kind: {type: string}, age: {type: integer}}, required: [kind, age]}]`, "")
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(design), `Meta("openapi:discriminator:propertyName", "kind")`)
	require.NotContains(t, string(design), `Meta("openapi:discriminator:mapping"`)
}
