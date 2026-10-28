package openapiimport

import (
	"testing"

	"github.com/pb33f/go-yaml"

	"github.com/stretchr/testify/require"
)

func TestReferenceAllOfBranchSiblingsFailClosed(t *testing.T) {
	for _, extra := range []string{"", `, {type: object, properties: {enabled: {type: boolean}}}`} {
		source := []byte(`openapi: 3.2.0
info: {title: References, version: "1"}
paths: {}
components:
 schemas:
  Base: {type: object, properties: {name: {type: string, maxLength: 8}}}
  Refined:
   allOf: [{$ref: '#/components/schemas/Base', type: object, properties: {name: {type: string, minLength: 3}}, required: [name]}` + extra + "]\n")
		document, diagnostics, err := Analyze(source)
		require.NoError(t, err)
		requireDiagnosticCode(t, diagnostics, "schema-composition")
		_, err = Render(document, Options{})
		require.Error(t, err)
	}
}

func TestReferenceInheritedNumericNullMembership(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: References, version: "1"}
paths: {}
components:
 schemas:
  Base: {type: object, properties: {n: {type: [integer, 'null'], enum: [null, 1]}}}
  Refined:
   $ref: '#/components/schemas/Base'
   properties:
    n: {enum: [1]}
`)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	requireDiagnosticCode(t, diagnostics, "schema-reference-siblings")
	_, err = Render(document, Options{})
	require.Error(t, err)
}

func TestReferenceInheritedArrayNumericSource(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: Reference contracts, version: "1"}
paths:
 /products:
  post:
   operationId: createProduct
   requestBody:
    required: true
    content:
     application/json:
      schema:
       type: object
       properties:
        original: {$ref: '#/components/schemas/Base'}
        refined:
         $ref: '#/components/schemas/Base'
         properties:
          ns: {maxItems: 2}
       required: [original, refined]
   responses: {"204": {description: Accepted}}
components:
 schemas:
  Base:
   type: object
   properties:
    ns: {type: array, items: {type: integer, enum: [1, 2], minimum: 1}}
`)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{})
	require.NoError(t, err)
	require.Contains(t, string(design), "Enum(1, 2)")
	require.Contains(t, string(design), `Minimum("1")`)
	module := requireRenderedDesignGenerates(t, design)
	validateReferenceSpec(t, module)
	validateReferenceRuntime(t, module, "inherited integer items")
}

func TestNumericSourceIdentitySurvivesRelocation(t *testing.T) {
	for _, source := range []string{
		`{"value":9007199254740993,"quoted":"9007199254740993"}`,
		"value: 9007199254740993\nquoted: \"9007199254740993\"\n",
		"<<: {value: 9007199254740993, quoted: \"9007199254740993\"}\n",
	} {
		values, err := numericSources([]byte(source))
		require.NoError(t, err)
		var root yaml.Node
		require.NoError(t, yaml.Unmarshal([]byte(source), &root))
		owner := analyzer{numericPositions: numericSourcePositions(&root, values)}
		members, err := effectiveYAMLMapping(root.Content[0], make(map[*yaml.Node]bool))
		require.NoError(t, err)
		for _, key := range []string{"value", "quoted"} {
			node := *members[key]
			node.Value, node.Tag = "0", "!!int"
			token, found := owner.nodeNumericSource(&node)
			require.True(t, found)
			number, err := token.number()
			if key == "quoted" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, "9007199254740993", number.String())
			}
		}
	}
}
