package openapiimport

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// sourceRetentionSource gives target a raw-only reference: the closed union U
// drops its redundant sibling property x, whose source $ref still names target.
// The other operation references target through the normalized model. extra
// appends component schema declarations.
func sourceRetentionSource(target, union, other string, reverse bool, extra string) []byte {
	operations := []string{
		"  " + union + ":\n    get:\n      operationId: union.get\n      responses:\n        '200': {description: u, content: {application/json: {schema: {$ref: '#/components/schemas/U'}}}}\n",
		"  " + other + ":\n    get:\n      operationId: target.get\n      responses:\n        '200': {description: t, content: {application/json: {schema: {$ref: '#/components/schemas/" + target + "'}}}}\n",
	}
	if reverse {
		slices.Reverse(operations)
	}
	return []byte(`openapi: 3.2.0
info: {title: Source retention, version: '1'}
paths:
` + strings.Join(operations, "") + `components:
  schemas:
    U:
      oneOf: [{$ref: '#/components/schemas/A'}, {$ref: '#/components/schemas/B'}]
      properties: {x: {$ref: '#/components/schemas/` + target + `'}}
    A: {type: object, properties: {a: {type: string}}, required: [a], additionalProperties: false}
    B: {type: object, properties: {b: {type: string}}, required: [b], additionalProperties: false}
    Y: {type: object, additionalProperties: {type: object, properties: {v: {type: string}}}}
` + extra)
}

// sourceRetentionOrders varies path names and document order around one union
// and one target operation.
var sourceRetentionOrders = []struct {
	union, other string
	reverse      bool
}{{"/a", "/z", false}, {"/a", "/z", true}, {"/z", "/a", false}, {"/z", "/a", true}}

// requireSourceRetention checks strict and partial selections of the union
// operation alone and with the target operation against the exact retained set.
func requireSourceRetention(t *testing.T, name, target, extra string, retained []string) {
	t.Helper()
	for _, order := range sourceRetentionOrders {
		source := sourceRetentionSource(target, order.union, order.other, order.reverse, extra)
		label := name + " " + order.union + " reverse=" + strconv.FormatBool(order.reverse)
		full, diagnostics, err := Analyze(source)
		require.NoError(t, err)
		require.Empty(t, diagnostics, label)
		require.Empty(t, unionSiblingProperties(full, "U"), "the redundant sibling is normalized away")
		for _, paths := range [][]string{{order.union}, {order.union, order.other}} {
			t.Run(label+" "+strings.Join(paths, ","), func(t *testing.T) {
				selection := Selection{Paths: paths}
				document, diagnostics, _, err := AnalyzeSelected(Source{Bytes: source}, selection)
				require.NoError(t, err)
				require.Empty(t, diagnostics)
				require.ElementsMatch(t, retained, retainedSchemaNames(document))
				_, err = Render(document, Options{PackageName: "design"})
				require.NoError(t, err)
				partial, _, err := AnalyzePartial(Source{Bytes: source}, selection, false)
				require.NoError(t, err)
				require.Empty(t, partial.Skipped)
				require.Empty(t, partial.OperationOmissions)
				require.ElementsMatch(t, retained, retainedSchemaNames(partial.Document))
			})
		}
	}
}

// A component first retained through a raw source reference must still have
// its normalized body walked: analyzer-promoted declarations exist only there.
func TestSelectionSourceRetainedBodyVisited(t *testing.T) {
	requireSourceRetention(t, "promoted declaration", "Y", "", []string{"A", "B", "U", "Y", "YAdditionalProperty"})
}

// Synthetic seam, not public input: one name in every kind with a normalized
// body. Each kind is retained by its own source prefix, in order, and its body
// alone references a distinct schema, so neither retention nor another kind's
// walk of the same name can stand in for that body.
func TestComponentPrunerSourceRetainedKinds(t *testing.T) {
	ref := func(name string) *Schema {
		return &Schema{Ref: componentDeclaration(name)}
	}
	components := Components{
		Schemas: []NamedSchema{
			{Name: "Shared", Schema: &Schema{Type: "object", Properties: []NamedProperty{{Name: "x", Schema: ref("SchemaBody")}}}},
			{Name: "SchemaBody", Schema: &Schema{Type: "string"}},
			{Name: "ParameterBody", Schema: &Schema{Type: "string"}},
			{Name: "RequestBodySchema", Schema: &Schema{Type: "string"}},
			{Name: "ResponseBodySchema", Schema: &Schema{Type: "string"}},
			{Name: "HeaderBodySchema", Schema: &Schema{Type: "string"}},
			{Name: "Unused", Schema: &Schema{Type: "string"}},
		},
		Parameters:    []NamedParameter{{Name: "Shared", Parameter: Parameter{Name: "q", In: "query", Schema: ref("ParameterBody")}}},
		RequestBodies: []NamedRequestBody{{Name: "Shared", RequestBody: RequestBody{Schema: ref("RequestBodySchema")}}},
		Responses:     []NamedResponse{{Name: "Shared", Response: Response{Schema: ref("ResponseBodySchema")}}},
		Headers:       []NamedHeader{{Name: "Shared", Header: Header{Schema: ref("HeaderBodySchema")}}},
	}
	kinds := []string{"schemas", "parameters", "requestBodies", "responses", "headers"}
	references := make(map[string]string, len(kinds))
	for _, kind := range kinds {
		references["#/paths/~1"+kind+"/get/x"] = "#/components/" + kind + "/Shared"
	}
	bundle := &bundledDescription{references: references}
	for _, normalizedAfter := range []bool{false, true} {
		pruner := newComponentPruner(components)
		for _, kind := range kinds {
			pruner.visitSourcePrefix(bundle, "#/paths/~1"+kind+"/get", make(map[string]bool))
		}
		if normalizedAfter {
			pruner.visitSchema(ref("Shared"))
			pruner.visitParameter(Parameter{Ref: "#/components/parameters/Shared"})
			pruner.visitRequestBody(RequestBody{Ref: "#/components/requestBodies/Shared"})
			pruner.visitResponse(Response{Ref: "#/components/responses/Shared"})
			pruner.visitHeader(Header{Ref: "#/components/headers/Shared"})
		}
		label := "normalizedAfter=" + strconv.FormatBool(normalizedAfter)
		for kind, retained := range map[string]map[string]struct{}{
			"parameters": pruner.closure.parameters, "requestBodies": pruner.closure.requestBodies,
			"responses": pruner.closure.responses, "headers": pruner.closure.headers,
		} {
			require.Equal(t, map[string]struct{}{"Shared": {}}, retained, "%s %s", kind, label)
		}
		schemas := make([]string, 0, len(pruner.closure.schemas))
		for name := range pruner.closure.schemas {
			schemas = append(schemas, name)
		}
		require.ElementsMatch(t, []string{"Shared", "SchemaBody", "ParameterBody", "RequestBodySchema", "ResponseBodySchema", "HeaderBodySchema"}, schemas, label)
	}
}

func unionSiblingProperties(document *Document, name string) []NamedProperty {
	for _, named := range document.Components.Schemas {
		if named.Name == name {
			return named.Schema.Properties
		}
	}
	return nil
}

func retainedSchemaNames(document *Document) []string {
	names := make([]string, 0, len(document.Components.Schemas))
	for _, named := range document.Components.Schemas {
		names = append(names, named.Name)
	}
	return names
}
