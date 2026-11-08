package openapiimport

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiscriminatorParentVersionMatrix(t *testing.T) {
	for _, version := range []string{"3.0.3", "3.1.1", "3.2.0"} {
		for _, required := range []bool{false, true} {
			for _, fallback := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/required=%t/default=%t", version, required, fallback), func(t *testing.T) {
					requiredText, hint := "", `{propertyName: kind, mapping: {cat: Cat, kitty: Cat, dog: Dog}`
					if required {
						requiredText = "      required: [kind]\n"
					}
					if fallback {
						hint += ", defaultMapping: Cat"
					}
					hint += "}"
					source := []byte(fmt.Sprintf(`openapi: %s
info: {title: Parent, version: '1'}
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties: {kind: {type: string}}
%s      discriminator: %s
    Cat:
      allOf:
        - {$ref: '#/components/schemas/Pet'}
        - {type: object, properties: {name: {type: string}}}
    Dog:
      allOf:
        - {$ref: '#/components/schemas/Pet'}
        - {type: object, properties: {age: {type: integer}}}
`, version, requiredText, hint))
					doc, diagnostics, err := Analyze(source)
					require.NoError(t, err)
					reject := fallback && version != "3.2.0" || !required && (version == "3.0.3" || version == "3.2.0" && !fallback)
					if reject {
						require.NotEmpty(t, diagnostics)
						return
					}
					require.Empty(t, diagnostics)
					design, err := Render(doc, Options{PackageName: "design"})
					require.NoError(t, err)
					require.Contains(t, string(design), `Meta("openapi:discriminator:mapping", "cat=Cat", "kitty=Cat", "dog=Dog")`)
					require.Contains(t, string(design), `Extend(ImportedPet)`)
					require.NotContains(t, string(design), `OneOf(`)
				})
			}
		}
	}
}

func TestDiscriminatorUnionVersionMatrix(t *testing.T) {
	for _, version := range []string{"3.0.3", "3.1.1", "3.2.0"} {
		for _, required := range []bool{false, true} {
			for _, fallback := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/required=%t/default=%t", version, required, fallback), func(t *testing.T) {
					hint := `{propertyName: kind`
					if fallback {
						hint += ", defaultMapping: Cat"
					}
					hint += "}"
					source := discriminatorDocument(version, hint, `[{$ref: '#/components/schemas/Cat'}, {$ref: '#/components/schemas/Dog'}]`, "")
					if !required {
						source = []byte(strings.ReplaceAll(string(source), "required: [kind, age]", "required: [age]"))
					}
					_, diagnostics, err := Analyze(source)
					require.NoError(t, err)
					reject := fallback && version != "3.2.0" || !required && (version == "3.0.3" || version == "3.2.0" && !fallback)
					if reject {
						require.NotEmpty(t, diagnostics)
					} else {
						require.Empty(t, diagnostics)
					}
				})
			}
		}
	}
}

func TestDiscriminatorUndefinedForms(t *testing.T) {
	for _, schema := range []string{
		`type: object
      discriminator: {propertyName: kind}`,
		`anyOf: [{type: string}, {type: integer}]
      discriminator: {propertyName: kind}`,
		`oneOf: [{type: string}, {type: integer}]
      allOf: [{type: string}]
      discriminator: {propertyName: kind}`,
		`oneOf: [{type: string}, {type: integer}]
      anyOf: [{type: string}]
      discriminator: {propertyName: kind}`,
	} {
		t.Run(schema, func(t *testing.T) {
			_, diagnostics, err := Analyze([]byte("openapi: 3.2.0\ninfo: {title: Undefined, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    Value:\n      " + schema + "\n"))
			require.NoError(t, err)
			found := false
			for _, d := range diagnostics {
				found = found || d.Code == "schema-discriminator"
			}
			require.True(t, found, "%#v", diagnostics)
		})
	}
}

func TestDiscriminatorParentWithInlineChild(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: Inline child, version: '1'}
paths:
  /child:
    get:
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
      discriminator: {propertyName: kind}
`)
	for _, explicit := range []bool{false, true} {
		t.Run(fmt.Sprintf("explicit=%t", explicit), func(t *testing.T) {
			input := source
			if explicit {
				input = []byte(strings.Replace(string(source), "discriminator: {propertyName: kind}", "discriminator: {propertyName: kind, mapping: {inline: '#/paths/~1child/get/responses/200/content/application~1json/schema'}}", 1))
			}
			document, diagnostics, err := Analyze(input)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			design, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			require.Contains(t, string(design), `Meta("openapi:discriminator:propertyName", "kind")`)
			if explicit {
				require.Contains(t, string(design), `Meta("openapi:discriminator:mapping", "inline=`)
			} else {
				require.NotContains(t, string(design), `Meta("openapi:discriminator:mapping"`)
			}
			requireRenderedDesignEvaluates(t, design, 1)
		})
	}
}

func TestDiscriminatorRequiredAdditionalMember(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: Required map tag, version: '1'}
paths:
  /value:
    get:
      operationId: value.get
      responses:
        '200':
          description: value
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Choice'}
components:
  schemas:
    Additional:
      type: object
      additionalProperties: {type: string}
      required: [kind]
    Declared:
      type: object
      properties: {kind: {type: string}, value: {type: integer}}
      required: [kind, value]
    Choice:
      oneOf: [{$ref: '#/components/schemas/Additional'}, {$ref: '#/components/schemas/Declared'}]
      discriminator: {propertyName: kind}
`)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	requireRenderedDesignGenerates(t, design)
}
