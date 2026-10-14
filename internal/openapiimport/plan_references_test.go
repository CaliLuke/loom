package openapiimport

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReferencePlanPreservesParallelConstructors(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		document := &Document{OpenAPIVersion: "3.1.1", Components: Components{Schemas: []NamedSchema{
			{Name: "Derived", GoName: "Derived", Schema: &Schema{Type: "object", Bases: []*Schema{componentReference("Base")}, Properties: []NamedProperty{{Name: "base", Schema: componentReference("Base")}}}},
			{Name: "Base", GoName: "Base", Schema: &Schema{Type: "object", Properties: []NamedProperty{{Name: "child", Schema: componentReference("Derived")}}}},
		}}}
		if reverse {
			slices.Reverse(document.Components.Schemas)
		}
		plan := planDocument(document)
		require.Empty(t, plan.diagnostics)
		source, err := Render(document, Options{})
		require.NoError(t, err)
		require.Contains(t, string(source), "Extend(ImportedBase)")
		require.Contains(t, string(source), `Attribute("base", "Base")`)
		require.Contains(t, string(source), `Attribute("child", "Derived")`)
		require.NotContains(t, plan.references.lookups, referenceOccurrenceKey{owner: componentDeclaration("Derived"), path: componentDeclaration("Derived") + "/bases/0"})
	}
}

func TestReferencePlanConstructorContexts(t *testing.T) {
	for _, test := range []struct {
		name       string
		root       *Schema
		lookupPath string
	}{
		{"array", &Schema{Type: "array", Items: componentReference("Leaf")}, ""},
		{"map", &Schema{Type: "object", AdditionalProperties: &AdditionalProperties{Schema: componentReference("Leaf")}}, ""},
		{"alias", componentReference("Leaf"), ""},
		{"symbolic union", &Schema{OneOf: []*Schema{componentReference("Leaf"), componentReference("Other")}}, "/oneOf/0"},
	} {
		t.Run(test.name, func(t *testing.T) {
			schemas := []NamedSchema{
				{Name: "Root", GoName: "Root", Schema: test.root},
				{Name: "Leaf", GoName: "Leaf", Schema: &Schema{Type: "object", Properties: []NamedProperty{{Name: "root", Schema: componentReference("Root")}}}},
				{Name: "Other", GoName: "Other", Schema: &Schema{Type: "object"}},
			}
			// Test the reference owner independently of the narrower untagged-union
			// admission policy. OneOf symbolic capability is not general eligibility.
			document := &Document{Components: Components{Schemas: schemas}}
			index := make(map[string]NamedSchema)
			for _, schema := range schemas {
				index[schema.Name] = schema
			}
			plan, diagnostics := planSchemaReferences(document, index, nil)
			require.Empty(t, diagnostics)
			require.Equal(t, "Root", plan.lookups[referenceOccurrenceKey{owner: componentDeclaration("Leaf"), path: componentDeclaration("Leaf") + "/properties/root"}])
			want := 1
			if test.lookupPath != "" {
				want++
				require.Equal(t, "Leaf", plan.lookups[referenceOccurrenceKey{owner: componentDeclaration("Root"), path: componentDeclaration("Root") + test.lookupPath}])
			}
			require.Len(t, plan.lookups, want)
		})
	}
}

func TestReferencePlanRetainsAcyclicDiamondAndNames(t *testing.T) {
	leaf := NamedSchema{Name: "Leaf", GoName: "Leaf", Schema: &Schema{Type: "object"}}
	node := NamedSchema{Name: "node-wire-name", GoName: "NodeAllocated2", Schema: &Schema{Type: "object", Properties: []NamedProperty{
		{Name: "self", Schema: componentReference("node-wire-name")},
		{Name: "left", Schema: componentReference("Leaf")},
		{Name: "right", Schema: componentReference("Leaf")},
	}}}
	document := &Document{OpenAPIVersion: "3.1.1", Components: Components{Schemas: []NamedSchema{node, leaf}}}
	plan := planDocument(document)
	require.Empty(t, plan.diagnostics)
	require.Len(t, plan.references.lookups, 1)
	source, err := Render(document, Options{})
	require.NoError(t, err)
	require.Contains(t, string(source), `Attribute("self", "node-wire-name")`)
	require.Contains(t, string(source), `Attribute("left", ImportedLeaf)`)
	require.Contains(t, string(source), `Attribute("right", ImportedLeaf)`)
	require.Contains(t, string(source), `var ImportedNodeAllocated2 = Type("node-wire-name"`)
}

func TestReferencePlanRejectsMandatoryCycles(t *testing.T) {
	for _, test := range []struct {
		kind, constructor, pathSuffix, code, reason string
	}{
		{"array", "ArrayOf", "/items", "schema-reference-initialization", "ArrayOf requires a concrete reference"},
		{"map", "MapOf", "/additionalProperties", "schema-reference-initialization", "MapOf requires a concrete reference"},
		{"alias", "Type", "", "schema", `cyclic object base "Node"`},
	} {
		t.Run(test.kind, func(t *testing.T) {
			ref := componentReference("Node")
			schema := ref
			if test.kind == "array" {
				schema = &Schema{Type: "array", Items: ref}
			}
			if test.kind == "map" {
				schema = &Schema{Type: "object", AdditionalProperties: &AdditionalProperties{Schema: ref}}
			}
			named := NamedSchema{Name: "Node", GoName: "Node", Schema: schema}
			document := &Document{OpenAPIVersion: "3.1.1", Components: Components{Schemas: []NamedSchema{named}}}

			// Keep the graph owner's mandatory-cycle obligation independent of
			// earlier shape admission, which rejects the cyclic alias base.
			_, diagnostics := planSchemaReferences(document, map[string]NamedSchema{"Node": named}, nil)
			require.Len(t, diagnostics, 1)
			require.Equal(t, "schema-reference-initialization", diagnostics[0].Code)
			require.Equal(t, componentDeclaration("Node")+test.pathSuffix, diagnostics[0].Path)
			require.Contains(t, diagnostics[0].Message, test.constructor+" requires a concrete reference")
			fatal, warnings := diagnostics.Classify(true)
			require.Equal(t, diagnostics, fatal)
			require.Empty(t, warnings)

			plan := planDocument(document)
			require.Len(t, plan.diagnostics, 1)
			require.Equal(t, test.code, plan.diagnostics[0].Code)
			require.Equal(t, componentDeclaration("Node")+test.pathSuffix, plan.diagnostics[0].Path)
			require.Contains(t, plan.diagnostics[0].Message, test.reason)
			fatal, warnings = plan.diagnostics.Classify(true)
			require.Equal(t, plan.diagnostics, fatal)
			require.Empty(t, warnings)
			_, err := Render(document, Options{})
			require.ErrorContains(t, err, "("+test.code+")")
			require.ErrorContains(t, err, test.reason)
		})
	}
}

func TestAnalyzeRejectsMandatoryReferenceCycles(t *testing.T) {
	for _, test := range []struct {
		name, schemas, constructor string
	}{
		{"self array", `Node: {type: array, items: {$ref: '#/components/schemas/Node'}}`, "ArrayOf"},
		{"self map", `Node: {type: object, additionalProperties: {$ref: '#/components/schemas/Node'}}`, "MapOf"},
		{"mutual containers", `Node: {type: array, items: {$ref: '#/components/schemas/Other'}}
    Other: {type: object, additionalProperties: {$ref: '#/components/schemas/Node'}}`, "ArrayOf"},
	} {
		t.Run(test.name, func(t *testing.T) {
			document, diagnostics, err := Analyze([]byte(`openapi: 3.1.1
info: {title: Recursive, version: "1"}
paths: {}
components:
  schemas:
    ` + test.schemas + "\n"))
			require.NoError(t, err)
			requireDiagnosticCode(t, diagnostics, "schema-reference-initialization")
			fatal, _ := diagnostics.Classify(true)
			require.NotEmpty(t, fatal, "allow-lossy must not admit an uninitializable declaration")
			require.Contains(t, diagnostics[0].Message, test.constructor)
			require.Contains(t, diagnostics[0].Path, "#/components/schemas/")
			_, err = Render(document, Options{})
			require.ErrorContains(t, err, "schema-reference-initialization")
		})
	}
}

func TestReferenceComponentsMatchReachability(t *testing.T) {
	// Independent transitive closure checks the implementation of the model's
	// reachability relation, including self edges and parallel occurrences.
	names := []string{"A", "B", "C"}
	for graph := range 1 << 9 {
		var occurrences []referenceOccurrence
		var reachable [3][3]bool
		var present [3]bool
		for source := range names {
			for target := range names {
				if graph&(1<<(source*3+target)) == 0 {
					continue
				}
				reachable[source][target] = true
				present[source], present[target] = true, true
				for _, path := range []string{"mandatory", "eligible"} {
					occurrences = append(occurrences, referenceOccurrence{
						key:    referenceOccurrenceKey{owner: names[source], path: path},
						target: names[target], eligible: path == "eligible",
					})
				}
			}
		}
		for via := range names {
			for source := range names {
				for target := range names {
					reachable[source][target] = reachable[source][target] || reachable[source][via] && reachable[via][target]
				}
			}
		}
		components := referenceComponents(occurrences)
		for source := range names {
			for target := range names {
				if !present[source] || !present[target] {
					continue
				}
				want := source == target || reachable[source][target] && reachable[target][source]
				require.Equal(t, want, components[names[source]] == components[names[target]], "graph %d, %s -> %s", graph, names[source], names[target])
			}
		}
	}
}

func TestReferencePlanPreservesRejectedConstructors(t *testing.T) {
	for _, test := range []struct {
		name, schemas string
	}{
		{"self alias", `Node: {$ref: '#/components/schemas/Node'}`},
		{"mutual alias", `Node: {$ref: '#/components/schemas/Other'}
    Other: {$ref: '#/components/schemas/Node'}`},
		{"union through array", `Node: {oneOf: [{$ref: '#/components/schemas/Nodes'}, {$ref: '#/components/schemas/Leaf'}]}
    Nodes: {type: array, items: {$ref: '#/components/schemas/Node'}}
    Leaf: {type: object, properties: {value: {type: string}}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			document, diagnostics, err := Analyze([]byte(`openapi: 3.1.1
info: {title: Recursive, version: "1"}
paths: {}
components:
  schemas:
    ` + test.schemas + "\n"))
			if test.name != "union through array" {
				require.ErrorContains(t, err, "infinite circular reference")
				return
			}
			require.NoError(t, err)
			requireDiagnosticCode(t, diagnostics, "schema-oneof-branch")
			_, err = Render(document, Options{})
			require.ErrorContains(t, err, "schema-oneof-branch")
		})
	}
}

func componentReference(name string) *Schema {
	return &Schema{Ref: "#/components/schemas/" + escapeJSONPointer(name)}
}
