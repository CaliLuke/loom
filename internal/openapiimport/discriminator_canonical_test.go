package openapiimport

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiscriminatorInlineTargetCanonicalUse(t *testing.T) {
	for _, selectPath := range []bool{false, true} {
		t.Run(fmt.Sprint(selectPath), func(t *testing.T) {
			selection := Selection{}
			if selectPath {
				selection.Paths = []string{"/child"}
			}
			document, diagnostics, _, err := AnalyzeSelected(Source{Bytes: canonicalParentInlineSource()}, selection)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			require.Len(t, document.Operations, 1)
			parent := namedCanonicalSchema(t, document, "Parent")
			require.Len(t, parent.Discriminator.Explicit, 1)
			target := parent.Discriminator.Explicit[0].Target
			var targetName string
			for _, named := range document.Components.Schemas {
				if named.Schema == target {
					targetName = named.Name
				}
			}
			require.NotEmpty(t, targetName)
			response := document.Operations[0].Responses[0].Response.Schema
			require.Equal(t, componentDeclaration(targetName), response.Ref,
				"mapping and authored inline response must use the same canonical declaration")
			require.Len(t, document.Components.Schemas, 2)
			design, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			require.Equal(t, 1, strings.Count(string(design), "Extend(ImportedParent)"))
		})
	}
}

func TestDiscriminatorInlineAlternativesHaveOneDeclaration(t *testing.T) {
	document, diagnostics, err := Analyze(canonicalInlineChoiceSource())
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	choice := namedCanonicalSchema(t, document, "Choice")
	require.Len(t, choice.OneOf, 2)
	require.Len(t, document.Components.Schemas, 3, "one declaration per distinct authored alternative")
	for index, branch := range choice.OneOf {
		name, ok := componentReferenceName(branch.Ref, "#/components/schemas/")
		require.True(t, ok)
		require.Same(t, choice.Discriminator.Explicit[index].Target, namedCanonicalSchema(t, document, name))
	}
	require.NotEqual(t, choice.OneOf[0].Ref, choice.OneOf[1].Ref, "equal shape is not source identity")
}

func TestDiscriminatorCanonicalTargetsGenerated(t *testing.T) {
	choice := strings.Replace(string(canonicalInlineChoiceSource()), "paths: {}", `paths:
  /choice:
    get:
      operationId: getChoice
      responses:
        '200': {description: OK, content: {application/json: {schema: {$ref: '#/components/schemas/Choice'}}}}`, 1)
	for _, tc := range []struct {
		name, path string
		source     []byte
	}{
		{name: "parent", path: "/child", source: canonicalParentInlineSource()},
		{name: "union", path: "/choice", source: []byte(choice)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, diagnostics, err := Analyze(tc.source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			design, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			directory := requireRenderedDesignGenerates(t, design)
			requireIndependentRegeneration(t, directory)
			requireDiscriminatorSpecifications(t, directory)
			data, err := os.ReadFile(filepath.Join(directory, "gen", "http", "openapi.json"))
			require.NoError(t, err)
			var spec map[string]any
			require.NoError(t, json.Unmarshal(data, &spec))
			components := spec["components"].(map[string]any)["schemas"].(map[string]any)
			if tc.name == "parent" {
				parent := components["Parent"].(map[string]any)
				mapping := parent["discriminator"].(map[string]any)["mapping"].(map[string]any)
				response := spec["paths"].(map[string]any)[tc.path].(map[string]any)["get"].(map[string]any)["responses"].(map[string]any)["200"].(map[string]any)
				body := response["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
				require.NotEmpty(t, body["$ref"])
				require.Equal(t, body["$ref"], mapping["inline"])
				require.Len(t, components, 2)
			} else {
				union := components["Choice"].(map[string]any)
				branches := union["oneOf"].([]any)
				mapping := union["discriminator"].(map[string]any)["mapping"].(map[string]any)
				require.Equal(t, branches[0].(map[string]any)["$ref"], mapping["first"])
				require.Equal(t, branches[1].(map[string]any)["$ref"], mapping["second"])
				require.NotEqual(t, mapping["first"], mapping["second"])
				require.Len(t, components, 3)
			}
		})
	}
}

func TestDiscriminatorCanonicalEnvelopeUse(t *testing.T) {
	for _, nested := range []bool{false, true} {
		t.Run(fmt.Sprint(nested), func(t *testing.T) {
			payload := "{type: string}"
			if nested {
				payload = "{$ref: '#/components/schemas/Created'}"
			}
			source := fmt.Sprintf(`openapi: 3.2.0
info: {title: Envelopes, version: '1'}
paths: {}
components:
  schemas:
    Created:
      type: object
      properties:
        kind: {type: string, enum: [created]}
        data: {type: string}
      required: [kind, data]
    Event:
      oneOf:
        - {$ref: '#/components/schemas/Created'}
        - type: object
          properties:
            kind: {type: string, enum: [wrapped]}
            data: %s
          required: [kind, data]
      discriminator:
        propertyName: kind
        mapping: {created: Created, wrapped: '#/components/schemas/Event/oneOf/1'}
`, payload)
			document, diagnostics, err := Analyze([]byte(source))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			require.Len(t, document.Components.Schemas, 3)
			event := namedCanonicalSchema(t, document, "Event")
			require.Equal(t, !nested, event.Discriminator.Tagged != nil,
				"authored nested payload reuse prevents envelope consumption; promotion alone does not")
			design, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			require.Equal(t, !nested, strings.Contains(string(design), `Meta("oneof:value:field", "data")`))
		})
	}
}

func namedCanonicalSchema(t *testing.T, document *Document, name string) *Schema {
	t.Helper()
	for _, named := range document.Components.Schemas {
		if named.Name == name {
			return named.Schema
		}
	}
	t.Errorf("missing schema %q", name)
	return nil
}

func canonicalParentInlineSource() []byte {
	return []byte(`openapi: 3.2.0
info: {title: Parent inline mapping, version: '1'}
paths:
  /child:
    get:
      operationId: getChild
      responses:
        '200':
          description: child
          content:
            application/json:
              schema:
                allOf:
                  - {$ref: '#/components/schemas/Parent'}
                  - {type: object, properties: {name: {type: string}}}
components:
  schemas:
    Parent:
      type: object
      properties: {kind: {type: string}}
      required: [kind]
      discriminator:
        propertyName: kind
        mapping: {inline: '#/paths/~1child/get/responses/200/content/application~1json/schema'}
`)
}

func canonicalInlineChoiceSource() []byte {
	return []byte(`openapi: 3.2.0
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
}
