package openapiimport

import (
	"encoding/json/v2"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchemaBooleanAnnotationBlocks(t *testing.T) {
	for _, keyword := range []string{"deprecated", "readOnly", "writeOnly"} {
		for _, state := range []string{"absent", "false", "true"} {
			source := map[string]any{"type": "string"}
			if state != "absent" {
				source[keyword] = state == "true"
			}
			document := analyzeBooleanAnnotationSchemas(t, map[string]any{"Value": source})
			schema := document.Components.Schemas[0].Schema
			for _, role := range []string{"attribute", "named", "body", "array-item", "map-value", "additional-property"} {
				t.Run(keyword+"/"+state+"/"+role, func(t *testing.T) {
					r := renderer{}
					var err error
					switch role {
					case "attribute":
						err = r.attribute("value", schema, nil, "#/value")
					case "named":
						err = r.namedSchema(NamedSchema{Name: "Value", GoName: "Value", Schema: schema})
					case "body":
						err = r.responseSchemaBody("OpenAPIBody", schema, "#/value")
					case "array-item":
						err = r.attribute("values", &Schema{Type: "array", Items: schema}, nil, "#/values")
					case "map-value":
						err = r.attribute("values", &Schema{Type: "object", AdditionalProperties: &AdditionalProperties{Schema: schema}}, nil, "#/values")
					case "additional-property":
						err = r.additionalAttributes(&Schema{Type: "object", Properties: []NamedProperty{{Name: "id", Schema: &Schema{Type: "string"}}}, AdditionalProperties: &AdditionalProperties{Schema: schema}}, "#/values")
					}
					require.NoError(t, err)
					output := r.builder.String()
					if state == "absent" {
						require.NotContains(t, output, "openapi:"+keyword)
						return
					}
					annotation := fmt.Sprintf("Meta(%q, %q)", "openapi:"+keyword, state)
					require.Equal(t, 1, strings.Count(output, annotation), output)
				})
			}
		}
	}
}

func TestPrimitiveBooleanAnnotationOwners(t *testing.T) {
	schemas := make(map[string]any)
	var checks strings.Builder
	for index, keyword := range []string{"deprecated", "readOnly", "writeOnly"} {
		for baseIndex, baseState := range []string{"absent", "false", "true"} {
			for localIndex, localState := range []string{"absent", "false", "true"} {
				name := fmt.Sprintf("Value%d%d%d", index, baseIndex, localIndex)
				base := map[string]any{"type": "string"}
				if baseState != "absent" {
					base[keyword] = baseState == "true"
				}
				local := map[string]any{"allOf": []any{base}}
				if localState != "absent" {
					local[keyword] = localState == "true"
				}
				schemas[name] = local
				for _, owner := range []struct{ name, state string }{{name, localState}, {name + "Base", baseState}} {
					fmt.Fprintf(&checks, `
 {
  owner := expr.Root.UserType(%q)
  if owner == nil {
   t.Fatalf("missing owner %%q", %q)
  }
  values, present := owner.Attribute().Meta[%q]
  if %t {
   if !present || len(values) != 1 || values[0] != %q {
    t.Errorf("owner %%s: annotation = %%#v, present=%%t", owner.Name(), values, present)
   }
  } else if present {
   t.Errorf("owner %%s: absent annotation became %%#v", owner.Name(), values)
  }
 }
`, owner.name, owner.name, "openapi:"+keyword, owner.state != "absent", owner.state)
				}
			}
		}
	}
	document := analyzeBooleanAnnotationSchemas(t, schemas)
	for _, named := range document.Components.Schemas {
		if named.constructionOnly {
			continue
		}
		require.Len(t, named.Schema.Bases, 1)
	}
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	requireRenderedDesignEvaluates(t, rendered, 1, checks.String())
}

func analyzeBooleanAnnotationSchemas(t *testing.T, schemas map[string]any) *Document {
	t.Helper()
	source := map[string]any{
		"openapi": "3.2.0", "info": map[string]any{"title": "Annotation ownership", "version": "1"},
		"paths": map[string]any{"/value": map[string]any{"get": map[string]any{
			"operationId": "value", "responses": map[string]any{"200": map[string]any{
				"description": "value", "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "string"}}},
			}},
		}}},
		"components": map[string]any{"schemas": schemas},
	}
	input, err := json.Marshal(source, json.Deterministic(true))
	require.NoError(t, err)
	document, diagnostics, err := Analyze(input)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	return document
}
