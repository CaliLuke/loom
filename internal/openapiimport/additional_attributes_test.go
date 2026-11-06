package openapiimport

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestImportObjectsWithTypedAdditionalProperties checks that objects that
// combine declared properties with additional properties, including required
// members that only additionalProperties describes, render as typed objects.
func TestImportObjectsWithTypedAdditionalProperties(t *testing.T) {
	for _, tc := range []struct {
		name   string
		schema string
		want   []string
	}{
		{"declared and typed additional properties", `{type: object, properties: {id: {type: string}}, required: [id, note], additionalProperties: {type: integer, minimum: 0}}`,
			[]string{`Attribute("id", String)`, "AdditionalAttributes(Int, func() {", `Required("id", "note")`}},
		{"required members described only by additional properties", `{type: object, required: [foo], additionalProperties: {type: integer}}`,
			[]string{"AdditionalAttributes(Int", `Required("foo")`}},
		{"declared properties and additionalProperties true", `{type: object, properties: {id: {type: string}}, additionalProperties: true}`,
			[]string{`Attribute("id", String)`, "AdditionalAttributes(Any)"}},
		{"required undeclared member of an open object", `{type: object, properties: {id: {type: string}}, required: [id, note]}`,
			[]string{"AdditionalAttributes(Any)", `Required("id", "note")`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte(`openapi: 3.2.0
info: {title: Typed, version: "1"}
paths:
  /probe:
    post:
      operationId: probe
      requestBody:
        required: true
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Probe'}
      responses: {"204": {description: done}}
components:
  schemas:
    Probe: ` + tc.schema + `
`)
			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			rendered, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			for _, want := range tc.want {
				require.Contains(t, string(rendered), want)
			}
			requireRenderedDesignEvaluates(t, rendered, 1)
		})
	}
}

// TestImportTypedReferenceSibling checks that a reference sibling with typed
// additionalProperties constrains members only the target declares, by
// redeclaring them as refinements, and gives the derived type the policy.
func TestImportTypedReferenceSibling(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: Typed, version: "1"}
paths:
  /probe:
    post:
      operationId: probe
      requestBody:
        required: true
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Holder'}
      responses: {"204": {description: done}}
components:
  schemas:
    Base:
      type: object
      properties:
        count: {type: integer, format: int64, minimum: 0}
    Holder:
      type: object
      properties:
        record:
          $ref: '#/components/schemas/Base'
          properties:
            total: {type: integer, format: int64}
          additionalProperties: {type: integer, format: int64, maximum: 10}
`)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	design := string(rendered)
	require.Contains(t, design, "Extend(ImportedBase)")
	require.Contains(t, design, `Attribute("count", Int64, func() {`)
	require.Contains(t, design, `Attribute("total", Int64)`)
	require.Contains(t, design, "AdditionalAttributes(Int64, func() {")
	requireRenderedDesignEvaluates(t, rendered, 1)
}

// typedImportSource wraps component schemas in a document whose single
// operation accepts the Probe schema as a JSON request body.
func typedImportSource(schemas string) []byte {
	return []byte(`openapi: 3.2.0
info: {title: Typed, version: "1"}
paths:
  /probe:
    post:
      operationId: probe
      requestBody:
        required: true
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Probe'}
      responses: {"204": {description: done}}
components:
  schemas:
` + schemas)
}

// TestImportTypedIntersectionsLoomCanRepresent checks that intersections with
// additionalProperties that Loom can express import faithfully and evaluate.
func TestImportTypedIntersectionsLoomCanRepresent(t *testing.T) {
	for _, tc := range []struct {
		name     string
		schemas  string
		want     []string
		reject   []string
		generate bool
	}{
		{"typed sibling policy under a closed base is vacuous", `    Base:
      type: object
      additionalProperties: false
      properties:
        id: {type: string}
        count: {type: integer}
    Probe:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            id: {type: string}
          additionalProperties: {type: integer}
`, []string{"Extend(ImportedBase)", `Attribute("count", Int`}, []string{"AdditionalAttributes("}, true},
		{"derived member refines the inherited policy carrier", `    Base:
      type: object
      properties:
        id: {type: string}
      additionalProperties: {type: integer, format: int64}
    Probe:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            total: {type: integer}
`, []string{"Extend(ImportedBase)", `Attribute("total", Int64)`}, nil, false},
		{"derived member under an unvalidated policy", `    Base:
      type: object
      properties:
        id: {type: string}
      additionalProperties: true
    Probe:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            name: {type: string}
`, []string{"Extend(ImportedBase)", `Attribute("name", String)`}, nil, true},
		{"disjoint sibling and base policies", `    Base:
      type: object
      properties:
        id: {type: string}
      additionalProperties: {type: integer}
    Probe:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            id: {type: string}
          additionalProperties: {type: string}
`, []string{"Extend(ImportedBase)", "AdditionalAttributes(String)"}, nil, true},
		{"nullable base policy refined by a non-nullable sibling policy", `    Base:
      type: object
      properties:
        id: {type: string}
      additionalProperties: {type: [integer, "null"]}
    Probe:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            id: {type: string}
          additionalProperties: {type: integer}
`, []string{"Extend(ImportedBase)", "AdditionalAttributes(Int"}, nil, true},
		{"required name covered by an inherited true policy", `    Base:
      type: object
      properties:
        id: {type: string}
      additionalProperties: true
    Probe:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            n: {type: integer}
          required: [count]
`, []string{"Extend(ImportedBase)", `Required("count")`}, nil, true},
		{"required name covered by an inherited empty schema policy", `    Base:
      type: object
      properties:
        id: {type: string}
      additionalProperties: {}
    Probe:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            n: {type: integer}
          required: [count]
`, []string{"Extend(ImportedBase)", `Required("count")`}, nil, false},
		{"required name covered by an inherited policy beside allOf", `    Base:
      type: object
      properties:
        id: {type: string}
      additionalProperties: {type: integer}
    Probe:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            n: {type: integer}
          required: [count]
`, []string{"Extend(ImportedBase)", `Required("count")`}, nil, true},
		{"required name covered by an inherited policy beside $ref", `    Base:
      type: object
      properties:
        id: {type: string}
      additionalProperties: {type: integer}
    Probe:
      type: object
      properties:
        rec:
          $ref: '#/components/schemas/Base'
          required: [count]
`, []string{`Required("count")`}, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, diagnostics, err := Analyze(typedImportSource(tc.schemas))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			rendered, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			design := string(rendered)
			for _, want := range tc.want {
				require.Contains(t, design, want)
			}
			for _, reject := range tc.reject {
				require.NotContains(t, design, reject)
			}
			if tc.generate {
				requireRenderedDesignGenerates(t, rendered)
				return
			}
			requireRenderedDesignEvaluates(t, rendered, 1)
		})
	}
}

// TestImportReportsTypedIntersectionsLoomRejects checks that intersections
// whose Loom translation is invalid are diagnosed at import.
func TestImportReportsTypedIntersectionsLoomRejects(t *testing.T) {
	for _, tc := range []struct {
		name    string
		schemas string
		path    string
	}{
		{"derived member violates the inherited policy type", `    Base:
      type: object
      properties:
        id: {type: string}
      additionalProperties: {type: integer}
    Probe:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            name: {type: string}
`, "#/components/schemas/Probe/allOf/1/properties/name"},
		{"reference sibling member violates the inherited policy type", `    Base:
      type: object
      properties:
        id: {type: string}
      additionalProperties: {type: integer}
    Probe:
      type: object
      properties:
        rec:
          $ref: '#/components/schemas/Base'
          properties:
            name: {type: string}
`, "#/components/schemas/Probe/properties/rec/properties/name"},
		{"derived member changes the inherited policy carrier format", `    Base:
      type: object
      properties:
        id: {type: string}
      additionalProperties: {type: integer, format: int64}
    Probe:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            total: {type: integer, format: int32}
`, "#/components/schemas/Probe/allOf/1/properties/total"},
		{"sibling and base policies overlap as integer and number", `    Base:
      type: object
      properties:
        id: {type: string}
      additionalProperties: {type: integer}
    Probe:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            id: {type: string}
          additionalProperties: {type: number}
`, "#/components/schemas/Probe/allOf/1"},
		{"sibling policy adds null to the base policy", `    Base:
      type: object
      properties:
        id: {type: string}
      additionalProperties: {type: integer}
    Probe:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            id: {type: string}
          additionalProperties: {type: [integer, "null"]}
`, "#/components/schemas/Probe/allOf/1"},
		{"sibling and base policies differ in integer format", `    Base:
      type: object
      properties:
        id: {type: string}
      additionalProperties: {type: integer, format: int64}
    Probe:
      type: object
      properties:
        rec:
          $ref: '#/components/schemas/Base'
          properties:
            id: {type: string}
          additionalProperties: {type: integer, format: int32}
`, "#/components/schemas/Probe/properties/rec"},
		{"sibling and base policies are different inline objects", `    Base:
      type: object
      properties:
        id: {type: string}
      additionalProperties: {type: object, properties: {a: {type: string}}}
    Probe:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            id: {type: string}
          additionalProperties: {type: object, properties: {b: {type: string}}}
`, "#/components/schemas/Probe/allOf/1"},
		{"derived reference member violates the inherited policy", `    Name: {type: string}
    Base:
      type: object
      properties:
        id: {type: string}
      additionalProperties: {type: integer}
    Probe:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            name: {$ref: '#/components/schemas/Name'}
`, "#/components/schemas/Probe/allOf/1/properties/name"},
		{"derived array member items violate the inherited policy", `    Base:
      type: object
      properties:
        id: {type: string}
      additionalProperties: {type: array, items: {type: integer}}
    Probe:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            tags: {type: array, items: {type: string}}
`, "#/components/schemas/Probe/allOf/1/properties/tags"},
		{"typed sibling under a closed base adds a property", `    Base:
      type: object
      additionalProperties: false
      properties:
        id: {type: string}
    Probe:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            id: {type: string}
            extra: {type: integer}
          additionalProperties: {type: integer}
`, "#/components/schemas/Probe/allOf/1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, diagnostics, err := Analyze(typedImportSource(tc.schemas))
			require.NoError(t, err)
			requireDiagnosticCode(t, diagnostics, "schema-reference-siblings")
			paths := make([]string, 0, len(diagnostics))
			for _, diagnostic := range diagnostics {
				paths = append(paths, diagnostic.Path)
			}
			require.Contains(t, paths, tc.path)
		})
	}
}

// TestImportTypedRecordReservesCarrierFieldName checks that a property whose
// Go field name collides with the carrier of undeclared members keeps its
// JSON name and receives a distinct Go field name.
func TestImportTypedRecordReservesCarrierFieldName(t *testing.T) {
	for _, tc := range []struct {
		name    string
		schemas string
		want    []string
	}{
		{"own typed policy", `    Probe:
      type: object
      properties:
        AdditionalAttributes: {type: string}
        additional_attributes: {type: string}
        note: {type: string}
      required: [Note]
      additionalProperties: {type: string}
`, []string{
			"Attribute(\"AdditionalAttributes\", String, func() {\n\t\tMeta(\"struct:field:name\", \"AdditionalAttributesField\")",
			"Attribute(\"additional_attributes\", String, func() {\n\t\tMeta(\"struct:field:name\", \"AdditionalAttributesField2\")",
			"Attribute(\"note\", String, func() {\n\t\tMeta(\"struct:field:name\", \"NoteField\")",
		}},
		{"inherited typed policy", `    Base:
      type: object
      properties:
        id: {type: string}
      additionalProperties: {type: string}
    Probe:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            additionalAttributes: {type: string}
`, []string{"Attribute(\"additionalAttributes\", String, func() {\n\t\tMeta(\"struct:field:name\", \"AdditionalAttributesField\")"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, diagnostics, err := Analyze(typedImportSource(tc.schemas))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			rendered, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			for _, want := range tc.want {
				require.Contains(t, string(rendered), want)
			}
			requireRenderedDesignGenerates(t, rendered)
		})
	}
}

// TestImportTypedRecordClosesDisjointPolicies checks that a sibling policy
// whose JSON kind is disjoint from the base's composes to a closed object.
func TestImportTypedRecordClosesDisjointPolicies(t *testing.T) {
	document, diagnostics, err := Analyze(typedImportSource(`    Base:
      type: object
      properties:
        id: {type: string}
      additionalProperties: {type: integer}
    Probe:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            id: {type: string}
          additionalProperties: {type: string}
`))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	requireRenderedDesignEvaluates(t, rendered, 1, `probe := expr.Root.UserType("Probe").Attribute()
	if got := probe.ObjectOpenness(); got != expr.ObjectClosed {
		t.Fatalf("got openness %v, want closed", got)
	}
	if probe.AdditionalAttributes != nil {
		t.Fatal("a closed object has no additional attributes")
	}`)
}

// TestImportReportsRequiredGoNameCollisions checks that required undeclared
// members whose fixed Go field names coincide are diagnosed at import.
func TestImportReportsRequiredGoNameCollisions(t *testing.T) {
	for _, tc := range []struct {
		name     string
		required string
		want     string
	}{
		{"two required undeclared names", "[foo, Foo]", `"Foo"`},
		{"required undeclared name and the carrier", "[additionalAttributes]", `"AdditionalAttributes"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, diagnostics, err := Analyze(typedImportSource(`    Probe:
      type: object
      properties:
        id: {type: string}
      required: ` + tc.required + `
      additionalProperties: {type: string}
`))
			require.NoError(t, err)
			require.NotEmpty(t, diagnostics)
			require.Equal(t, "#/components/schemas/Probe", diagnostics[0].Path)
			require.Contains(t, diagnostics[0].Message, "generates Go field "+tc.want)
		})
	}
}

// inheritedPolicySchemas declares Probe as an allOf derivation of a Base whose
// typed additionalProperties Probe inherits.
const inheritedPolicySchemas = `components:
  schemas:
    Base:
      type: object
      properties:
        id: {type: string}
      additionalProperties: {type: string}
    Probe:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            name: {type: string}
`

// TestImportInheritedPolicyFormBodyStaysRaw checks that a form body whose
// object inherits a typed policy through allOf stays raw, since typed form
// decoding has no carrier for undeclared members.
func TestImportInheritedPolicyFormBodyStaysRaw(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: Typed, version: "1"}
paths:
  /probe:
    post:
      operationId: probe
      requestBody:
        required: true
        content:
          application/x-www-form-urlencoded:
            schema: {$ref: '#/components/schemas/Probe'}
      responses: {"204": {description: done}}
` + inheritedPolicySchemas)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(rendered), "SkipRequestBodyEncodeDecode()")
	requireRenderedDesignGenerates(t, rendered)
}

// TestImportReportsInheritedPolicyURLField checks that a URL-serialized form
// field whose object inherits a typed policy through allOf is reported, while
// the same field over an untyped base keeps its object serialization.
func TestImportReportsInheritedPolicyURLField(t *testing.T) {
	for _, tc := range []struct {
		name    string
		schemas string
		want    string
	}{
		{"untyped base", strings.Replace(inheritedPolicySchemas, "      additionalProperties: {type: string}\n", "", 1), ""},
		{"typed base", inheritedPolicySchemas, "form-serialization"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte(`openapi: 3.2.0
info: {title: Typed, version: "1"}
paths:
  /probe:
    post:
      operationId: probe
      requestBody:
        required: true
        content:
          application/x-www-form-urlencoded:
            schema:
              type: object
              properties:
                filter: {$ref: '#/components/schemas/Probe'}
            encoding:
              filter: {contentType: application/json}
      responses: {"204": {description: done}}
` + tc.schemas)
			_, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			codes := make([]string, 0, len(diagnostics))
			for _, diagnostic := range diagnostics {
				codes = append(codes, diagnostic.Code)
			}
			if tc.want == "" {
				require.Empty(t, diagnostics)
				return
			}
			require.Contains(t, codes, tc.want)
		})
	}
}
