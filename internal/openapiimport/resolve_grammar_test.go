package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolverGrammar(t *testing.T) {
	root, _, err := parseDescriptionTree([]byte(`openapi: 3.2.0
info: {title: Grammar, version: '1'}
paths:
  /pets:
    post:
      callbacks:
        completed:
          '{$request.body#/url}': {$ref: 'callbacks.yaml#/done'}
      requestBody:
        content:
          multipart/form-data:
            encoding:
              upload:
                headers:
                  X-Trace: {$ref: 'headers.yaml#/trace'}
            schema:
              type: object
              properties:
                x-value: {$ref: 'models.yaml#/Value'}
              example: {$ref: 'data-example'}
              default: {$ref: 'data-default'}
              const: {$ref: 'data-const'}
              enum: [{$ref: 'data-enum'}]
              x-extension: {$ref: 'data-extension'}
      responses:
        '204': {description: Done}
components:
  examples:
    Literal: {value: {$ref: 'data-example-value'}}
`))
	require.NoError(t, err)
	edges, err := descriptionEdges(&document{key: "entry", root: root}, "", "openapi")
	require.NoError(t, err)
	got := make(map[string]string)
	for _, edge := range edges {
		got[edge.authored] = edge.class + ":" + edge.kind
	}
	require.Equal(t, map[string]string{
		"callbacks.yaml#/done": "pathItem:pathItems",
		"headers.yaml#/trace":  "component:headers",
		"models.yaml#/Value":   "component:schemas",
	}, got)
}

func TestResolverInvalidReference(t *testing.T) {
	source := Source{Bytes: []byte(`openapi: 3.2.0
info: {title: Invalid, version: '1'}
paths: {}
components:
  schemas:
    Bad: {$ref: false}
`)}
	_, diagnostics, err := resolveDescription(source)
	require.NoError(t, err)
	require.Len(t, diagnostics, 1)
	require.Equal(t, "reference must be a string", diagnostics[0].Message)
}

func TestResolverDocumentClasses(t *testing.T) {
	for _, tc := range []struct {
		name, body, ref, message string
	}{
		{"schemaRoot", "type: string", "model.yaml", ""},
		{"booleanSchema", "true", "model.yaml", ""},
		{"resource", "$id: https://example.test/schema\ntype: string", "model.yaml", "schema-resource"},
		{"ancestorResource", "$schema: https://example.test/dialect\nX: {type: string}", "model.yaml#/X", "schema-resource"},
		{"resourceNamedProperty", "Pet: {type: object, properties: {$id: {type: string}, value: {type: string}}}", "model.yaml#/Pet/properties/value", ""},
		{"resourceNamedComponent", "$id: {type: string}\nPet: {type: string}", "model.yaml#/Pet", ""},
		{"keywordNamedProperty", "Pet: {type: object, properties: {$id: {type: string}, properties: {items: {type: string}}}}", "model.yaml#/Pet/properties/properties/items", ""},
		{"keywordNamedComponent", "$id: {type: string}\nitems: {properties: {value: {type: string}}}", "model.yaml#/items/properties/value", ""},
		{"schemaNamedProperties", "Pet: {type: object, properties: {properties: {$id: 'https://example.test/resource', items: {type: string}}}}", "model.yaml#/Pet/properties/properties/items", "schema-resource"},
		{"schemaNamedItems", "Pet: {type: object, properties: {items: {$id: 'https://example.test/resource', properties: {value: {type: string}}}}}", "model.yaml#/Pet/properties/items/properties/value", "schema-resource"},
		{"schemaArrayAncestor", "Pet: {oneOf: [{$id: 'https://example.test/resource', properties: {value: {type: string}}}]}", "model.yaml#/Pet/oneOf/0/properties/value", "schema-resource"},
		{"malformedSchemaAncestor", "Pet: {properties: {nested: {$id: false, properties: {value: {type: string}}}}}", "model.yaml#/Pet/properties/nested/properties/value", "schema-resource"},
		{"malformedSchemaRoot", "Pet: {$id: false, properties: {value: {type: string}}}", "model.yaml#/Pet/properties/value", "schema-resource"},
		{"dynamicReference", "type: object\nproperties:\n  value: {$dynamicRef: '#node'}", "model.yaml", "schema-resource"},
		{"dynamicDefinitions", "definitions: {Value: {$dynamicRef: '#node'}}", "model.yaml", "schema-resource"},
		{"dynamicUnevaluatedItems", "type: array\nunevaluatedItems: {$dynamicRef: '#node'}", "model.yaml", "schema-resource"},
		{"dynamicData", "type: object\nexample: {$dynamicRef: '#node'}", "model.yaml", ""},
		{"wrongVersion", "openapi: 3.1.1\ncomponents:\n  schemas:\n    X: {type: string}", "model.yaml#/components/schemas/X", "version mismatch"},
		{"wrongKind", "openapi: 3.2.0\ncomponents:\n  headers:\n    X: {schema: {type: string}}", "model.yaml#/components/headers/X", "required OpenAPI object kind"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := resolverFiles(t, resolverEntry("", tc.ref), map[string]string{"model.yaml": tc.body})
			_, diagnostics, err := resolveDescription(source)
			require.NoError(t, err)
			if tc.message == "" {
				require.Empty(t, diagnostics)
			} else {
				require.Contains(t, diagnostics.Error(), tc.message)
			}
		})
	}
}

func TestResolverContextConflict(t *testing.T) {
	source := resolverFiles(t, []byte(`openapi: 3.2.0
info: {title: Context, version: '1'}
paths:
  /pets: {$ref: 'shared.yaml#/X'}
components:
  schemas:
    X: {$ref: 'shared.yaml#/X'}
`), map[string]string{"shared.yaml": "X: {}"})
	_, diagnostics, err := resolveDescription(source)
	require.NoError(t, err)
	require.Len(t, diagnostics, 2)
	for _, diagnostic := range diagnostics {
		require.Equal(t, "reference-context", diagnostic.Code)
	}
}
