package openapiimport

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolverSchemaTargetRoles(t *testing.T) {
	const targetDocument = `openapi: 3.2.0
info: {title: Schema targets, version: '1'}
paths:
  /pets:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema: {type: integer}
components:
  schemas:
    Pet:
      oneOf:
        - type: object
          properties:
            id: {type: string}
            'a/b': {type: boolean}
          required: [id]
        - {type: string}
      example: {type: object, properties: {fake: {type: string}}}
      default: {type: object}
    Booleans:
      additionalProperties: false
`
	for _, tc := range []struct {
		name, pointer string
		valid         bool
	}{
		{"component", "/components/schemas/Pet", true},
		{"inlineAlternative", "/components/schemas/Pet/oneOf/0", true},
		{"nestedProperty", "/components/schemas/Pet/oneOf/0/properties/id", true},
		{"escapedProperty", "/components/schemas/Pet/oneOf/0/properties/a~1b", true},
		{"inlineResponse", "/paths/~1pets/get/responses/200/content/application~1json/schema", true},
		{"boolean", "/components/schemas/Booleans/additionalProperties", true},
		{"propertiesDictionary", "/components/schemas/Pet/oneOf/0/properties", false},
		{"required", "/components/schemas/Pet/oneOf/0/required", false},
		{"operation", "/paths/~1pets/get", false},
		{"example", "/components/schemas/Pet/example", false},
		{"exampleNested", "/components/schemas/Pet/example/properties/fake", false},
		{"default", "/components/schemas/Pet/default", false},
	} {
		for _, external := range []bool{false, true} {
			for _, mapping := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/external=%t/mapping=%t", tc.name, external, mapping), func(t *testing.T) {
					target := "#" + tc.pointer
					entry := targetDocument
					if external {
						target = "target.yaml" + target
						entry = "openapi: 3.2.0\ninfo: {title: Entry, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n"
					}
					if mapping {
						entry += "    Carrier:\n      type: object\n      discriminator:\n        propertyName: kind\n        mapping: {selected: '" + target + "'}\n"
					} else {
						entry += "    Carrier: {$ref: '" + target + "'}\n"
					}
					source := resolverFiles(t, []byte(entry), map[string]string{"target.yaml": targetDocument})
					resolved, diagnostics, err := resolveDescription(source)
					require.NoError(t, err)
					if tc.valid || mapping {
						require.Empty(t, diagnostics)
					} else {
						require.NotEmpty(t, diagnostics)
					}
					require.NotNil(t, resolved)
					var matches []*edge
					for _, e := range resolved.edges {
						if e.authored == target && strings.HasPrefix(e.at.pointer, "/components/schemas/Carrier/") {
							matches = append(matches, e)
						}
					}
					require.Len(t, matches, 1)
					if tc.valid {
						require.Nil(t, matches[0].failure)
						require.NotNil(t, matches[0].target)
						require.Equal(t, tc.pointer, matches[0].target.pointer)
						bundle, bundleErr := bundleResolution(resolved)
						require.NoError(t, bundleErr)
						bundled := matches[0].bundled
						if !mapping {
							bundled = bundle.references["#/components/schemas/Carrier/$ref"]
						}
						require.NotEmpty(t, bundled)
						origin := bundle.origins[bundled]
						require.Equal(t, "#"+tc.pointer, origin.Pointer)
						require.Equal(t, resolved.registry.docs[matches[0].target.doc].portable, origin.Document)
					} else {
						require.NotNil(t, matches[0].failure)
						require.Contains(t, matches[0].failure.Message, "object kind")
					}
				})
			}
		}
	}
}
