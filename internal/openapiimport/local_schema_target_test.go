package openapiimport

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func localSchemaTargetSource(target string) []byte {
	return []byte(fmt.Sprintf(`openapi: 3.2.0
info: {title: Local targets, version: '1'}
paths:
  /value:
    get:
      operationId: getValue
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema: {$ref: '%s'}
  /inline:
    get:
      operationId: getInline
      responses:
        '200':
          description: Inline
          content:
            application/json:
              schema: {type: string}
components:
  schemas:
    id: {type: boolean}
    id_2: {type: number}
    Container:
      type: object
      properties:
        id: {type: string}
        node:
          type: object
          properties:
            next: {$ref: '#/components/schemas/Container/properties/node'}
    Choice:
      oneOf:
        - {type: object, properties: {name: {type: string}}, required: [name]}
        - {type: boolean}
`, target))
}

func TestLocalSchemaTargetsImport(t *testing.T) {
	for _, pointer := range []string{
		"#/components/schemas/Container/properties/id",
		"#/components/schemas/Choice/oneOf/0",
		"#/paths/~1inline/get/responses/200/content/application~1json/schema",
		"#/components/schemas/Container/properties/node",
		"#/components/schemas/Container",
	} {
		t.Run(pointer, func(t *testing.T) {
			document, diagnostics, err := Analyze(localSchemaTargetSource(pointer))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			rendered, err := Render(document, Options{})
			require.NoError(t, err)
			require.NotEmpty(t, rendered)
			for _, named := range document.Components.Schemas {
				if named.Name == "id" {
					require.Equal(t, "boolean", named.Schema.Type)
				}
				if named.Name == "id_2" {
					require.Equal(t, "number", named.Schema.Type)
				}
			}
			if strings.HasSuffix(pointer, "/properties/id") {
				found := false
				for _, named := range document.Components.Schemas {
					if named.Name == "id_3" {
						found = true
						require.Equal(t, "string", named.Schema.Type)
					}
				}
				require.True(t, found, "promotion reserves authored names before allocating")
			}
		})
	}
}

func TestLocalSchemaTargetSharedIdentity(t *testing.T) {
	const pointer = "/components/schemas/Container/properties/id"
	entry := string(localSchemaTargetSource("#"+pointer)) + "    Mapping:\n      type: object\n      discriminator:\n        propertyName: kind\n        mapping: {explicit: '#" + pointer + "'}\n"
	resolved, diagnostics, err := resolveDescription(Source{Bytes: []byte(entry)})
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	bundle, err := bundleResolution(resolved)
	require.NoError(t, err)
	target := bundle.references["#/paths/~1value/get/responses/200/content/application~1json/schema/$ref"]
	require.Equal(t, "#/components/schemas/id_3", target)
	mappings := resolved.discriminatorEdges()
	require.Len(t, mappings, 1)
	require.Equal(t, target, mappings[0].bundled)
	require.Empty(t, mappings[0].entry, "URI promotion does not invent an authored component-name mapping")
	require.Equal(t, bundle.origins["#"+pointer], bundle.origins[target])
	require.True(t, bundle.changed(), "local promotion must reach normalization")
}

func TestLocalPromotionPreservesAuthoredNames(t *testing.T) {
	document, diagnostics, err := Analyze([]byte(`openapi: 3.2.0
info: {title: Names, version: '1'}
paths:
  /value:
    get:
      operationId: value
      responses:
        '200': {description: OK, content: {application/json: {schema: {$ref: '#/components/schemas/Holder/properties/foo-bar'}}}}
components:
  schemas:
    foo_bar: {type: string}
    Holder: {type: object, properties: {foo-bar: {type: string}}}
`))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	names := map[string]string{}
	for _, schema := range document.Components.Schemas {
		names[schema.Name] = schema.GoName
	}
	require.Equal(t, "FooBar", names["foo_bar"])
	require.Equal(t, "FooBar2", names["foo-bar"])
}

func TestLocalSchemaTargetAliasOrder(t *testing.T) {
	const pointer = "/components/schemas/Container/properties/id"
	const absolute = "https://entry.test/api.yaml#" + pointer
	for _, reverse := range []bool{false, true} {
		values := []string{"          short: '#" + pointer + "'\n", "          absolute: '" + absolute + "'\n"}
		if reverse {
			values[0], values[1] = values[1], values[0]
		}
		entry := strings.Replace(string(localSchemaTargetSource("#"+pointer)), "openapi: 3.2.0\n", "openapi: 3.2.0\n$self: https://entry.test/api.yaml\n", 1)
		entry += "    Mapping:\n      type: object\n      discriminator:\n        propertyName: kind\n        mapping:\n" + strings.Join(values, "")
		resolved, diagnostics, err := resolveDescription(Source{Bytes: []byte(entry)})
		require.NoError(t, err)
		require.Empty(t, diagnostics)
		bundle, err := bundleResolution(resolved)
		require.NoError(t, err)
		outcomes := resolved.discriminatorEdges()
		require.Len(t, outcomes, 2)
		for _, outcome := range outcomes {
			require.Nil(t, outcome.failure)
			require.Equal(t, "#/components/schemas/id_3", outcome.bundled)
			require.Empty(t, outcome.entry)
		}
		require.Equal(t, SourceLocation{Document: "entry", Pointer: "#" + pointer}, bundle.origins["#/components/schemas/id_3"])
	}
}
