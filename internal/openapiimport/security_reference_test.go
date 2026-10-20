package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSecuritySchemeReferenceDescriptions(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: Security references, version: "1"}
paths:
  /x:
    get:
      operationId: getX
      responses: {"200": {description: ok}}
security:
  - Override: []
    Empty: []
    Inherit: []
    Middle: []
    EmptyLeaf: []
    Leaf: []
    LeafOverride: []
    SecondOverride: []
    Original: []
components:
  securitySchemes:
    Original:
      type: http
      scheme: basic
      description: Original description.
    Override:
      $ref: '#/components/securitySchemes/Original'
      description: Aliased credential.
    Empty:
      $ref: '#/components/securitySchemes/Original'
      description: ""
    Inherit:
      $ref: '#/components/securitySchemes/Original'
    Middle:
      $ref: '#/components/securitySchemes/Original'
      description: Middle credential.
    EmptyLeaf:
      $ref: '#/components/securitySchemes/Empty'
    Leaf:
      $ref: '#/components/securitySchemes/Middle'
    LeafOverride:
      $ref: '#/components/securitySchemes/Middle'
      description: Leaf credential.
    SecondOverride:
      $ref: '#/components/securitySchemes/Original'
      description: Separate credential.
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)

	want := map[string]string{
		"Original":       "Original description.",
		"Override":       "Aliased credential.",
		"Empty":          "",
		"Inherit":        "Original description.",
		"Middle":         "Middle credential.",
		"EmptyLeaf":      "",
		"Leaf":           "Middle credential.",
		"LeafOverride":   "Leaf credential.",
		"SecondOverride": "Separate credential.",
	}
	for name, description := range want {
		scheme := securitySchemeByName(document.Components.SecuritySchemes, name)
		require.NotNil(t, scheme, name)
		require.Equal(t, description, scheme.Description, name)
	}

	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	moduleDir := requireRenderedDesignGenerates(t, rendered)
	contract := readGeneratedOpenAPIContract(t, moduleDir)
	generatedSchemes := contract["components"].(map[string]any)["securitySchemes"].(map[string]any)
	for name, description := range want {
		generated := generatedSchemes[name].(map[string]any)
		got, _ := generated["description"].(string)
		require.Equal(t, description, got, name)
	}
}

func TestSecuritySchemeReferenceDescriptionsByOpenAPIVersion(t *testing.T) {
	for _, test := range []struct {
		version string
		want    map[string]string
	}{
		{
			version: "3.0.3",
			want: map[string]string{
				"Original": "Original description.",
				"Override": "Original description.",
				"Empty":    "Original description.",
				"Inherit":  "Original description.",
			},
		},
		{
			version: "3.1.1",
			want: map[string]string{
				"Original": "Original description.",
				"Override": "Aliased credential.",
				"Empty":    "",
				"Inherit":  "Original description.",
			},
		},
	} {
		t.Run(test.version, func(t *testing.T) {
			source := []byte(`openapi: ` + test.version + `
info: {title: Security references, version: "1"}
paths: {}
components:
  securitySchemes:
    Original: {type: http, scheme: basic, description: Original description.}
    Override: {$ref: '#/components/securitySchemes/Original', description: Aliased credential.}
    Empty: {$ref: '#/components/securitySchemes/Original', description: ""}
    Inherit: {$ref: '#/components/securitySchemes/Original'}
`)
			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			for name, description := range test.want {
				scheme := securitySchemeByName(document.Components.SecuritySchemes, name)
				require.NotNil(t, scheme, name)
				require.Equal(t, description, scheme.Description, name)
			}
		})
	}
}
