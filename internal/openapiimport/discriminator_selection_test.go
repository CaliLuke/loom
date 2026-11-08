package openapiimport

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiscriminatorParentSelection(t *testing.T) {
	for _, tc := range []struct {
		name, mapping                      string
		supported, unsupported, dependency bool
	}{
		{name: "supported implicit", supported: true},
		{name: "mixed implicit", supported: true, unsupported: true},
		{name: "unsupported implicit", unsupported: true},
		{name: "transitive unsupported", supported: true, unsupported: true, dependency: true},
		{name: "default unsupported", mapping: ", defaultMapping: Broken", supported: true, unsupported: true},
		{name: "explicit supported", mapping: ", mapping: {child: Child}", supported: true, unsupported: true},
		{name: "explicit unsupported mixed", mapping: ", mapping: {broken: Broken}", supported: true, unsupported: true},
		{name: "explicit unsupported only", mapping: ", mapping: {broken: Broken}", unsupported: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := discriminatorParentSelectionSource(tc.supported, tc.unsupported, tc.mapping)
			if tc.dependency {
				source = []byte(strings.Replace(string(source), "{name: {type: string}}", "{name: {$ref: '#/components/schemas/Broken'}}", 1))
			}
			selection := Selection{Tags: []string{"parents"}}
			document, diagnostics, _, err := AnalyzeSelected(Source{Bytes: source}, selection)
			require.NoError(t, err)
			names := discriminatorSelectionNames(document)
			t.Logf("strict names=%v diagnostics=%v", names, diagnostics)
			partial, _, err := AnalyzePartial(Source{Bytes: source}, selection, false)
			require.NoError(t, err)
			t.Logf("partial names=%v skipped=%v omitted=%v operation omissions=%v", discriminatorSelectionNames(partial.Document), partial.Skipped, partial.Omitted, partial.OperationOmissions)
			switch {
			case strings.Contains(tc.mapping, "Broken"):
				require.NotEmpty(t, diagnostics)
				require.Len(t, partial.Skipped, 1)
			case tc.supported && !tc.dependency:
				require.Empty(t, diagnostics)
				require.Contains(t, names, "Child", "a supported implicit child keeps the retained ParentHint defined")
				require.Contains(t, discriminatorSelectionNames(partial.Document), "Child")
			default:
				requireDiagnosticCode(t, diagnostics, "schema-discriminator-projection")
				require.Empty(t, partial.Skipped)
				require.Len(t, partial.OperationOmissions, 1)
				requireDiagnosticCode(t, partial.OperationOmissions[0].Diagnostics, "schema-discriminator-omission")
				for _, named := range partial.Document.Components.Schemas {
					if named.Name == "Parent" {
						require.Nil(t, named.Schema.Discriminator)
						require.Equal(t, []string{"kind"}, named.Schema.Required)
					}
				}
			}
		})
	}
}

func discriminatorParentSelectionSource(supported, unsupported bool, mapping string) []byte {
	source := fmt.Sprintf(`openapi: 3.2.0
info: {title: Parent selection, version: '1'}
paths:
  /parents:
    get:
      tags: [parents]
      operationId: parents.get
      responses:
        '200':
          description: parent
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Parent'}
components:
  schemas:
    Parent:
      type: object
      properties: {kind: {type: string}}
      required: [kind]
      discriminator: {propertyName: kind%s}
`, mapping)
	if supported {
		source += `    Child:
      allOf: [{$ref: '#/components/schemas/Parent'}, {type: object, properties: {name: {type: string}}}]
`
	}
	if unsupported {
		source += `    Broken:
      allOf:
        - {$ref: '#/components/schemas/Parent'}
        - {type: object, properties: {name: {type: string}}}
        - {type: object, properties: {other: {type: string}}}
`
	}
	return []byte(source)
}

func discriminatorSelectionNames(document *Document) []string {
	var names []string
	if document == nil {
		return names
	}
	for _, named := range document.Components.Schemas {
		names = append(names, named.Name)
	}
	return names
}

func TestDiscriminatorParentInlineSelection(t *testing.T) {
	source := discriminatorParentInlineSource()
	for _, selected := range []string{"parents", "children"} {
		t.Run(selected, func(t *testing.T) {
			discriminatorParentInlineSelection(t, source, selected)
		})
	}
}

func discriminatorParentInlineSource() string {
	return strings.Replace(string(discriminatorParentSelectionSource(false, false, "")), "components:\n", `  /child:
    get:
      tags: [children]
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
`, 1)
}

func discriminatorParentInlineSelection(t *testing.T, source, selected string) {
	t.Helper()
	selection := Selection{Tags: []string{selected}}
	document, diagnostics, _, err := AnalyzeSelected(Source{Bytes: []byte(source)}, selection)
	require.NoError(t, err)
	if selected == "children" {
		require.Empty(t, diagnostics)
		design, err := Render(document, Options{PackageName: "design"})
		require.NoError(t, err)
		require.Contains(t, string(design), `Meta("openapi:discriminator:propertyName", "kind")`)
		require.NotContains(t, string(design), `Meta("openapi:discriminator:mapping"`)
	} else {
		requireDiagnosticCode(t, diagnostics, "schema-discriminator-projection")
		for _, diagnostic := range diagnostics {
			require.NotEqual(t, "schema-discriminator", diagnostic.Code, "the complete source hint is valid")
		}
		partial, _, err := AnalyzePartial(Source{Bytes: []byte(source)}, selection, false)
		require.NoError(t, err)
		require.Empty(t, partial.Skipped)
		require.Len(t, partial.OperationOmissions, 1)
		requireDiagnosticCode(t, partial.OperationOmissions[0].Diagnostics, "schema-discriminator-omission")
	}
}

func TestDiscriminatorParentProjectionIsolation(t *testing.T) {
	source := Source{Bytes: []byte(discriminatorParentInlineSource())}
	document, diagnostics, _, err := AnalyzeSelected(source, Selection{})
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	original, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	var parent *Schema
	for _, named := range document.Components.Schemas {
		if named.Name == "Parent" {
			parent = named.Schema
		}
	}
	require.NotNil(t, parent)
	hint := parent.Discriminator
	for _, operation := range document.Operations {
		candidate := documentForOperations(document, []Operation{operation})
		pruneComponents(candidate)
		projected, omitted := omitUnprojectableParentHints(candidate)
		if operation.Path == "/parents" {
			require.Len(t, omitted, 1)
			require.NotSame(t, candidate, projected)
			require.Empty(t, parentHintProjectionDiagnostics(unprojectableParentHints(projected)))
		} else {
			require.Empty(t, omitted)
			require.Same(t, candidate, projected)
		}
		require.Same(t, hint, parent.Discriminator)
	}
	after, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Equal(t, original, after)
	partial, _, err := AnalyzePartial(source, Selection{}, false)
	require.NoError(t, err)
	require.Empty(t, partial.Skipped)
	require.Empty(t, partial.OperationOmissions, "the combined output retains the inline child and hint")
	partialDesign, err := Render(partial.Document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(partialDesign), `Meta("openapi:discriminator:propertyName", "kind")`)
	full, diagnostics, err := Analyze(source.Bytes)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	fullDesign, err := Render(full, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Equal(t, original, fullDesign)
}

func TestDiscriminatorParentCanonicalChildIdentity(t *testing.T) {
	source := resolverFiles(t, []byte(`openapi: 3.2.0
info: {title: External parent, version: '1'}
paths:
  /value:
    get:
      responses:
        '200':
          description: child
          content:
            application/json:
              schema: {$ref: 'models.yaml#/components/schemas/Child'}
`), map[string]string{"models.yaml": `openapi: 3.2.0
$self: https://example.test/models.yaml
info: {title: Models, version: '1'}
paths: {}
components:
  schemas:
    Parent:
      type: object
      properties: {kind: {type: string}}
      required: [kind]
      discriminator: {propertyName: kind, mapping: {child: './models.yaml#/components/schemas/Child'}}
    Child:
      allOf:
        - {$ref: 'https://example.test/models.yaml#/components/schemas/Parent'}
        - {type: object, properties: {name: {type: string}}}
`})
	document, diagnostics, _, err := AnalyzeSelected(source, Selection{Paths: []string{"/value"}})
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	var hints int
	for _, named := range document.Components.Schemas {
		if hint := named.Schema.Discriminator; hint != nil {
			hints++
			require.Len(t, hint.children, 1)
			require.Len(t, hint.Explicit, 1)
			require.Equal(t, hint.children[0], discriminatorIdentity(document.bundle, hint.Explicit[0].Target))
			require.NotEqual(t, sourceID("entry"), hint.children[0].doc)
		}
	}
	require.Equal(t, 1, hints)
}

func TestDocumentSchemaOccurrencesCycles(t *testing.T) {
	shared := &Schema{sourcePath: "#/components/schemas/Shared"}
	root := &Schema{sourcePath: "#/components/schemas/Root", Items: shared}
	shared.Items = root
	document := &Document{Components: Components{Schemas: []NamedSchema{{Name: "Root", Schema: root}, {Name: "Shared", Schema: shared}}}}
	require.Len(t, documentSchemaOccurrences(document), 2)
}

func TestDiscriminatorChildInvalidExamplePreservesSchema(t *testing.T) {
	source := strings.Replace(string(discriminatorParentSelectionSource(true, false, "")), "    Child:\n", "    Child:\n      examples: [{kind: cat, name: 42}]\n", 1)
	document, diagnostics, _, err := AnalyzeSelected(Source{Bytes: []byte(source)}, Selection{Tags: []string{"parents"}})
	require.NoError(t, err)
	require.Contains(t, discriminatorSelectionNames(document), "Child")
	require.Len(t, diagnostics, 1)
	require.Equal(t, "examples", diagnostics[0].Code)
	require.Equal(t, "#/components/schemas/Child/examples/0", diagnostics[0].Path)
	for _, named := range document.Components.Schemas {
		if named.Name == "Child" {
			require.Empty(t, named.Schema.Examples)
		}
	}
	partial, _, err := AnalyzePartial(Source{Bytes: []byte(source)}, Selection{Tags: []string{"parents"}}, true)
	require.NoError(t, err)
	require.Empty(t, partial.Skipped)
	require.Contains(t, discriminatorSelectionNames(partial.Document), "Child")
}
