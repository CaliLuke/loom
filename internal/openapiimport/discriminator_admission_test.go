package openapiimport

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiscriminatorNullableAlternativeAdmission(t *testing.T) {
	for _, version := range []string{"3.0.3", "3.1.1", "3.2.0"} {
		forms := []string{"declaration", "alias"}
		if version != "3.0.3" {
			forms = append(forms, "reference occurrence")
		}
		for _, fallback := range []bool{false, true} {
			for _, form := range forms {
				t.Run(fmt.Sprintf("%s/default=%t/%s", version, fallback, form), func(t *testing.T) {
					hint := "{propertyName: kind}"
					if fallback {
						hint = "{propertyName: kind, defaultMapping: Cat}"
					}
					source := discriminatorNullableDocument(version, hint, form)
					_, diagnostics, err := Analyze(source)
					require.NoError(t, err)
					switch {
					case fallback && version != "3.2.0":
						requireDiagnosticCode(t, diagnostics, "versioned-field")
					case version == "3.0.3" || version == "3.2.0" && !fallback:
						requireDiagnosticCode(t, diagnostics, "schema-discriminator")
					default:
						require.Empty(t, diagnostics)
					}
				})
			}
		}
	}
}

func discriminatorNullableDocument(version, hint, form string) []byte {
	branches := `[{$ref: '#/components/schemas/Cat'}, {$ref: '#/components/schemas/Dog'}]`
	source := string(discriminatorDocument(version, hint, branches, ""))
	nullable := "type: [object, 'null']"
	if version == "3.0.3" {
		nullable = "type: object\n      nullable: true"
	}
	switch form {
	case "declaration":
		source = strings.Replace(source, "    Cat:\n      type: object", "    Cat:\n      "+nullable, 1)
	case "reference occurrence":
		source = strings.Replace(source, branches, `[{anyOf: [{$ref: '#/components/schemas/Cat'}, {type: 'null'}]}, {$ref: '#/components/schemas/Dog'}]`, 1)
	case "alias":
		source = strings.Replace(source, "    Cat:\n      type: object", "    Cat:\n      "+nullable, 1)
		source = strings.Replace(source, branches, `[{$ref: '#/components/schemas/CatAlias'}, {$ref: '#/components/schemas/Dog'}]`, 1)
		source += "    CatAlias: {$ref: '#/components/schemas/Cat'}\n"
	}
	return []byte(source)
}

func TestDiscriminatorNullableParentAdmission(t *testing.T) {
	for _, version := range []string{"3.0.3", "3.1.1", "3.2.0"} {
		t.Run(version, func(t *testing.T) {
			nullable := "type: [object, 'null']"
			if version == "3.0.3" {
				nullable = "type: object\n      nullable: true"
			}
			source := []byte(fmt.Sprintf(`openapi: %s
info: {title: Nullable parent, version: '1'}
paths: {}
components:
  schemas:
    Parent:
      %s
      properties: {kind: {type: string}}
      required: [kind]
      discriminator: {propertyName: kind}
    Child:
      allOf: [{$ref: '#/components/schemas/Parent'}]
`, version, nullable))
			_, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			if version != "3.1.1" {
				requireDiagnosticCode(t, diagnostics, "schema-discriminator")
			} else {
				require.Empty(t, diagnostics)
			}
		})
	}
}

func TestNullableRequiredSiblingRemainsRedundant(t *testing.T) {
	source := discriminatorNullableDocument("3.1.1", "{propertyName: kind}", "declaration")
	source = []byte(strings.Replace(string(source), "      discriminator:", "      required: [kind]\n      discriminator:", 1))
	_, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics, "required only constrains objects; null does not make this sibling nonredundant")
}

func TestDiscriminatorExtensionAdmissionOwner(t *testing.T) {
	for _, tc := range []struct {
		name, value string
		invalid     bool
	}{
		{name: "nonfinite", value: ".nan", invalid: true},
		{name: "nested nonfinite", value: "{numbers: [.inf]}", invalid: true},
		{name: "object", value: "{z: 17, a: [true, 1.5, null]}"},
		{name: "numeric string key", value: `{"1": true}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := discriminatorDocument("3.2.0", "{propertyName: kind, x-value: "+tc.value+"}", `[{$ref: '#/components/schemas/Cat'}, {$ref: '#/components/schemas/Dog'}]`, "      x-value: "+tc.value)
			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			if tc.invalid {
				var paths []string
				for _, diagnostic := range diagnostics {
					if diagnostic.Code == "vendor-extension" {
						paths = append(paths, diagnostic.Path)
					}
				}
				require.Contains(t, paths, "#/components/schemas/Pet/discriminator/x-value")
				require.Contains(t, paths, "#/components/schemas/Pet/x-value")
				return
			}
			require.Empty(t, diagnostics)
			for _, named := range document.Components.Schemas {
				if named.Name == "Pet" {
					require.Equal(t, named.Schema.Extensions["x-value"], named.Schema.Discriminator.Extensions["x-value"])
				}
			}
		})
	}
}
