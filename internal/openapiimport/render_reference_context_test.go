package openapiimport

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

const recursiveMetadataChecks = `
node := expr.Root.UserType("Node")
field := node.Attribute().Find("name")
if !node.Attribute().IsRequired("name") || field.Validation == nil || field.Validation.MinLength == nil || *field.Validation.MinLength != 2 {
 t.Fatal("required field validation lost")
}
if values := node.Attribute().Meta["openapi:typename:canonical"]; len(values) != 1 || values[0] != "true" {
 t.Fatal("canonical schema-name metadata lost")
}
`

func TestRenderReferenceContexts(t *testing.T) {
	cases := []struct {
		name, schemas string
		failures      bool
	}{
		{"top_array_object", `Node: {type: array, items: {$ref: '#/components/schemas/Branch'}}
    Branch: {type: object, properties: {child: {$ref: '#/components/schemas/Node'}}}`, false},
		{"top_map_object", `Node: {type: object, additionalProperties: {$ref: '#/components/schemas/Branch'}}
    Branch: {type: object, properties: {child: {$ref: '#/components/schemas/Node'}}}`, false},
		{"alias_object", `Node: {$ref: '#/components/schemas/Branch'}
    Branch: {type: object, properties: {child: {$ref: '#/components/schemas/Node'}}}`, false},
		{"nested_promoted", `Node:
      type: object
      required: [name]
      properties:
        name: {type: string, minLength: 2}
        children:
          type: array
          items:
            type: object
            additionalProperties:
              type: array
              minItems: 1
              items: {anyOf: [{$ref: '#/components/schemas/Node'}, {type: 'null'}]}`, false},
		{"nested_annotated", `Node:
      type: object
      required: [name]
      properties:
        name: {type: string, minLength: 2}
        children:
          type: array
          items:
            type: array
            minItems: 1
            items: {anyOf: [{$ref: '#/components/schemas/Node'}, {type: 'null'}]}`, false},
		{"error_clones", `Node: {type: object, properties: {message: {type: string}, child: {$ref: '#/components/schemas/Node'}}}`, true},
		{"union_forward_array", `Node: {oneOf: [{$ref: '#/components/schemas/Branches'}, {$ref: '#/components/schemas/Leaf'}]}
    Branches: {type: array, items: {$ref: '#/components/schemas/Leaf'}}
    Leaf: {type: object, properties: {child: {$ref: '#/components/schemas/Leaf'}}}`, false},
	}
	for _, tc := range cases {
		var yamlRendered []byte
		for _, format := range []string{"YAML", "JSON"} {
			t.Run(tc.name+"/"+format, func(t *testing.T) {
				failures := ""
				if tc.failures {
					failures = `
        "400": {description: bad request, content: {application/json: {schema: {$ref: '#/components/schemas/Node'}}}}
        "500": {description: failure, content: {application/json: {schema: {$ref: '#/components/schemas/Node'}}}}`
				}
				source := []byte(`openapi: 3.1.1
info: {title: Recursive, version: "1"}
paths:
  /nodes:
    get:
      operationId: getNodes
      responses:
        "200": {description: ok, content: {application/json: {schema: {$ref: '#/components/schemas/Node'}}}}` + failures + `
components:
  schemas:
    ` + tc.schemas + "\n")
				if format == "JSON" {
					var data map[string]any
					require.NoError(t, yaml.Unmarshal(source, &data))
					encoded, err := json.Marshal(data)
					require.NoError(t, err)
					source = encoded
				}
				doc, diagnostics, err := Analyze(source)
				require.NoError(t, err)
				if tc.name == "nested_promoted" {
					require.Len(t, diagnostics, 1)
					requireDiagnosticCode(t, diagnostics, "schema-inline-array-item-promoted")
				} else {
					require.Empty(t, diagnostics)
				}
				rendered, err := Render(doc, Options{PackageName: "design"})
				require.NoError(t, err)
				if tc.failures {
					require.Contains(t, string(rendered), "Extend(ImportedNode)")
				}
				requireRenderedDesignEvaluates(t, rendered, 1, referenceContextChecks(tc.name))
				if format == "JSON" {
					require.Equal(t, yamlRendered, rendered, "source syntax must not change reference lowering")
				} else {
					yamlRendered = rendered
					switch tc.name {
					case "nested_promoted", "error_clones", "top_array_object", "alias_object", "union_forward_array":
						requireRecursiveDesignGenerates(t, rendered, tc.name)
					}
				}
			})
		}
	}
}

func referenceContextChecks(name string) string {
	switch name {
	case "top_array_object":
		return `
node, branch := expr.Root.UserType("Node"), expr.Root.UserType("Branch")
if expr.AsArray(node).ElemType.Type != branch || branch.Attribute().Find("child").Type != node {
 t.Fatal("array/object recursion lost canonical identity")
}
`
	case "top_map_object":
		return `
node, branch := expr.Root.UserType("Node"), expr.Root.UserType("Branch")
if expr.AsMap(node).ElemType.Type != branch || branch.Attribute().Find("child").Type != node {
 t.Fatal("map/object recursion lost canonical identity")
}
`
	case "alias_object":
		return `
node, branch := expr.Root.UserType("Node"), expr.Root.UserType("Branch")
if node.Attribute().Type != branch || branch.Attribute().Find("child").Type != node {
 t.Fatal("alias/object recursion lost canonical identity")
}
`
	case "nested_promoted":
		return recursiveMetadataChecks + `
promoted := expr.Root.UserType("NodeChildrenItem")
if expr.AsArray(node.Attribute().Find("children").Type).ElemType.Type != promoted {
 t.Fatal("inline promotion identity lost")
}
element := expr.AsArray(expr.AsMap(promoted).ElemType.Type).ElemType
if element.Type.(expr.UserType).ID() != node.ID() || !expr.IsNullable(element) {
 t.Fatal("promoted recursive element identity/nullability lost")
}
`
	case "nested_annotated":
		return recursiveMetadataChecks + `
inner := expr.AsArray(node.Attribute().Find("children").Type).ElemType
if inner.Validation == nil || inner.Validation.MinLength == nil || *inner.Validation.MinLength != 1 {
 t.Fatal("nested array validation lost")
}
element := expr.AsArray(inner.Type).ElemType
if element.Type != node || !expr.IsNullable(element) {
 t.Fatal("nested recursive element identity/nullability lost")
}
`
	case "error_clones":
		return `
node := expr.Root.UserType("Node")
if node.Attribute().Find("child").Type != node {
 t.Fatal("recursive source error identity lost")
}
failures := expr.Root.Services[0].Methods[0].Errors
if len(failures) != 2 || failures[0].Type == failures[1].Type {
 t.Fatal("error clones must retain separate declaration identities")
}
for _, failure := range failures {
 if failure.Find("child").Type.(expr.UserType).ID() != node.ID() {
  t.Fatal("error clone must retain the source's recursive target")
 }
}
`
	case "union_forward_array":
		return `
node, leaf, branches := expr.Root.UserType("Node"), expr.Root.UserType("Leaf"), expr.Root.UserType("Branches")
union := expr.AsUnion(node)
if union == nil || !union.Untagged || len(union.Values) != 2 || union.Values[0].Attribute.Type != branches || union.Values[1].Attribute.Type != leaf {
 t.Fatal("named union variants lost canonical identities")
}
if expr.AsArray(branches).ElemType.Type != leaf || leaf.Attribute().Find("child").Type != leaf {
 t.Fatal("array/recursive object branch identity lost")
}
`
	default:
		panic("missing recursive contract assertions")
	}
}
