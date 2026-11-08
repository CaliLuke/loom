package ir

import (
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

// TestOccurrenceDeclarationFollowsPlanTarget resolves the declaration of each
// JSON response occurrence node through its value plan. HTTP body copies,
// renamed nested copies and flattened unions are not declarations
// themselves; an inline root denotes no declaration and contributes its
// parent through its own composition.
func TestOccurrenceDeclarationFollowsPlanTarget(t *testing.T) {
	for _, tc := range []struct {
		name        string
		result      func(parent, choice expr.UserType) any
		path        string
		declaration string
		contributor string
	}{
		{name: "body copy", path: "$", declaration: "Parent", result: func(parent, _ expr.UserType) any { return parent }},
		{name: "nested copy", path: "$/alias/pet", declaration: "Parent", result: func(parent, _ expr.UserType) any {
			return dsl.Type("Holder", func() { dsl.Attribute("pet", parent) })
		}},
		{name: "array element copy", path: "$/alias/pets/items", declaration: "Parent", result: func(parent, _ expr.UserType) any {
			return dsl.Type("Holder", func() { dsl.Attribute("pets", dsl.ArrayOf(parent)) })
		}},
		{name: "flattened union", path: "$", declaration: "Choice", result: func(_, choice expr.UserType) any { return choice }},
		{name: "inline root", path: "$/alias", contributor: "Parent", result: func(parent, _ expr.UserType) any {
			return func() {
				dsl.Extend(parent)
				dsl.Meta("openapi:allOf:reference", "true")
				dsl.Attribute("note", dsl.String)
			}
		}},
		{name: "inline field", path: "$/alias/pet", contributor: "Parent", result: func(parent, _ expr.UserType) any {
			return dsl.Type("Holder", func() {
				dsl.Attribute("pet", func() {
					dsl.Extend(parent)
					dsl.Meta("openapi:allOf:reference", "true")
					dsl.Attribute("note", dsl.String)
				})
			})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				dsl.API("occurrence", func() {})
				parent := dsl.Type("Parent", func() {
					dsl.Meta("openapi:discriminator:propertyName", "kind")
					dsl.Attribute("kind", dsl.String)
					dsl.Required("kind")
				})
				cat := dsl.Type("Cat", func() {
					dsl.Attribute("kind", dsl.String)
					dsl.Required("kind")
				})
				choice := dsl.Type("Choice", dsl.OneOf(func() {
					dsl.Untagged()
					dsl.Attribute("Cat", cat)
				}), func() {
					dsl.Meta("openapi:discriminator:propertyName", "kind")
				})
				result := tc.result(parent, choice)
				dsl.Service("occurrences", func() {
					dsl.Method("show", func() {
						dsl.Result(result)
						dsl.HTTP(func() {
							dsl.GET("/occurrence")
						})
					})
				})
			})
			bodies := BuildBodyTypes(root.API, root.Types, root.ResultTypes, WithExampleValue(nil))
			require.NoError(t, bodies.Err)
			a := NewAnalyzer(nil, false)
			for _, typ := range root.Types {
				a.declarationTypes[typ.ID()] = typ
			}
			response := bodies.prepared["occurrences"].Endpoints[0].Response.Responses[0]
			body := attributeForSchemaUsage(response.DocumentBody, schemaUsageResponse)
			attribute, node := occurrenceNodeAt(t, body, representationRoot(body, response.DocumentValue), tc.path)
			require.Equal(t, expr.ValueCodecJSON, node.Codec())
			declaration := a.occurrenceDeclaration(attribute, node)
			if tc.declaration == "" {
				require.Nil(t, declaration, "the node denotes no declaration")
			} else {
				require.NotNil(t, declaration)
				require.Equal(t, tc.declaration, declaration.ID())
				if named, ok := attribute.Type.(expr.UserType); ok {
					require.NotEqual(t, tc.declaration, named.ID(), "the physical type is a copy, not the declaration")
				}
			}
			if tc.contributor != "" {
				var names []string
				for _, neighbor := range a.attributePublicationNeighbors(attribute, declaration) {
					names = append(names, neighbor.ID())
				}
				require.Contains(t, names, tc.contributor, "the attribute's own composition contributes its parent")
			}
		})
	}
}

// TestDeclarationNeighborsTraverseAttributes reads composition contributors
// from every attribute of a declaration graph: an inline field, and the same
// inline field reached through array elements, map values and a recursive
// declaration. A flat extension without the allOf reference marker
// contributes nothing.
func TestDeclarationNeighborsTraverseAttributes(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.API("neighbors", func() {})
		parent := dsl.Type("Parent", func() {
			dsl.Attribute("kind", dsl.String)
		})
		composed := func() {
			dsl.Extend(parent)
			dsl.Meta("openapi:allOf:reference", "true")
			dsl.Attribute("note", dsl.String)
		}
		inline := dsl.Type("InlineField", func() {
			dsl.Attribute("pet", composed)
		})
		dsl.Type("ArrayElement", func() {
			dsl.Attribute("items", dsl.ArrayOf(inline))
		})
		dsl.Type("MapValue", func() {
			dsl.Attribute("byName", dsl.MapOf(dsl.String, inline))
		})
		dsl.Type("Tree", func() {
			dsl.Attribute("children", dsl.ArrayOf("Tree"))
			dsl.Attribute("pet", composed)
		})
		dsl.Type("Flat", func() {
			dsl.Attribute("pet", func() {
				dsl.Extend(parent)
				dsl.Attribute("note", dsl.String)
			})
		})
	})
	a := NewAnalyzer(nil, false)
	for _, typ := range root.Types {
		a.declarationTypes[typ.ID()] = typ
	}
	for _, name := range []string{"InlineField", "ArrayElement", "MapValue", "Tree", "Flat"} {
		t.Run(name, func(t *testing.T) {
			var neighbors []string
			for _, neighbor := range a.declarationNeighbors(root.UserType(name)) {
				neighbors = append(neighbors, neighbor.ID())
			}
			if name == "Flat" {
				require.NotContains(t, neighbors, "Parent", "a flat extension has no allOf contributor")
			} else {
				require.Contains(t, neighbors, "Parent")
			}
		})
	}
}

// occurrenceNodeAt pairs attributes with value-plan nodes exactly as
// projectionIdentity does and returns the pair at path.
func occurrenceNodeAt(t *testing.T, attribute *expr.AttributeExpr, node expr.ValuePlanNode, path string) (*expr.AttributeExpr, expr.ValuePlanNode) {
	t.Helper()
	var foundAttribute *expr.AttributeExpr
	var foundNode expr.ValuePlanNode
	active := map[expr.ValuePlanNode]bool{}
	var visit func(*expr.AttributeExpr, expr.ValuePlanNode, string)
	visit = func(attribute *expr.AttributeExpr, node expr.ValuePlanNode, current string) {
		if attribute == nil || !node.Valid() || active[node] || foundAttribute != nil {
			return
		}
		if current == path {
			foundAttribute, foundNode = attribute, node
			return
		}
		active[node] = true
		defer delete(active, node)
		switch typ := attribute.Type.(type) {
		case expr.UserType:
			visit(typ.Attribute(), node.Underlying(), current+"/alias")
		case *expr.Array:
			visit(typ.ElemType, node.Element(), current+"/items")
		case *expr.Map:
			visit(typ.ElemType, node.Element(), current+"/values")
		case *expr.Object:
			for _, field := range *typ {
				for _, member := range node.Members() {
					if field.Name == member.Name && member.Visible {
						visit(field.Attribute, member.Node, current+"/"+field.Name)
					}
				}
			}
		}
	}
	visit(attribute, node, "$")
	require.NotNil(t, foundAttribute, "occurrence node %s", path)
	return foundAttribute, foundNode
}

// TestConstructionContextReadsRecordedContributors decides a projection from
// the contributors that constructions record below it, paired with the plan
// nodes projection uses: its own composition slots, a position-preserving
// wrapper, a local component reference, an inline holder member and a
// recursive declaration. A position without a plan contributes nothing, and
// each result is memoized under its plan node.
func TestConstructionContextReadsRecordedContributors(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.API("construction", func() {})
		parent := dsl.Type("Parent", func() {
			dsl.Attribute("kind", dsl.String)
			dsl.Attribute("thumb", dsl.Bytes)
		})
		composed := func() {
			dsl.Extend(parent)
			dsl.Meta("openapi:allOf:reference", "true")
			dsl.Attribute("note", dsl.String)
		}
		dsl.Type("Holder", func() {
			dsl.Attribute("pet", composed)
		})
		dsl.Type("Tree", func() {
			dsl.Attribute("children", dsl.ArrayOf("Tree"))
			dsl.Attribute("pet", composed)
		})
	})
	parent := root.UserType("Parent")
	a := NewAnalyzer(nil, false)
	for _, typ := range root.Types {
		a.declarationTypes[typ.ID()] = typ
	}
	_, parentPlan := a.declarationRepresentation(parent)
	_, holderPlan := a.declarationRepresentation(root.UserType("Holder"))
	_, treePlan := a.declarationRepresentation(root.UserType("Tree"))
	require.Equal(t, expr.ValueCodecJSON, holderPlan.Codec())

	composition := func() *Schema {
		schema := &Schema{AllOf: []*Schema{{Ref: toRef("Parent")}, {Type: "object"}}}
		a.constructions[schema] = schemaConstruction{slots: []schemaSlot{
			{location: schemaAllOf, index: 0, base: parent},
			{location: schemaAllOf, index: 1},
		}}
		return schema
	}
	own := composition()
	wrapper := &Schema{AnyOf: []*Schema{own, {Type: "null"}}}
	a.constructions[wrapper] = schemaConstruction{slots: []schemaSlot{{location: schemaAnyOf}}}
	a.schemas["Composed"] = own
	reference := &Schema{Ref: toRef("Composed")}
	holder := &Schema{Properties: map[string]*Schema{"pet": composition()}}
	a.constructions[holder] = schemaConstruction{kind: expr.ObjectKind, slots: []schemaSlot{{location: schemaProperty, name: "pet", member: "pet"}}}
	children := &Schema{Items: &Schema{Ref: toRef("Tree")}}
	a.constructions[children] = schemaConstruction{kind: expr.ArrayKind, slots: []schemaSlot{{location: schemaItems, element: true}}}
	tree := &Schema{Properties: map[string]*Schema{"children": children, "pet": composition()}}
	a.constructions[tree] = schemaConstruction{kind: expr.ObjectKind, slots: []schemaSlot{
		{location: schemaProperty, name: "children", member: "children"},
		{location: schemaProperty, name: "pet", member: "pet"},
	}}
	a.schemas["Tree"] = tree

	for _, tc := range []struct {
		name     string
		baseline *Schema
		plan     expr.ValuePlanNode
	}{
		{"composition", own, parentPlan},
		{"wrapper", wrapper, parentPlan},
		{"reference", reference, parentPlan},
		{"inline holder member", holder, holderPlan},
		{"recursive declaration", &Schema{Ref: toRef("Tree")}, treePlan},
	} {
		t.Run(tc.name, func(t *testing.T) {
			context := a.constructionContext(tc.baseline, tc.plan)
			require.Contains(t, context, parent.ID()+"=")
			require.Equal(t, context, a.constructionContexts[schemaPosition{tc.baseline, tc.plan}], "memoized under its plan node")
			require.Equal(t, context, a.constructionContext(tc.baseline, tc.plan))
		})
	}
	require.Empty(t, a.constructionContext(holder, expr.ValuePlanNode{}), "a position without a plan carries no codec context")
	require.Contains(t, a.constructionContext(holder, holderPlan), parent.ID()+"=", "the plan-less entry does not shadow the planned one")
}

// TestDeclarationGraphMemoization reuses per-declaration neighbors and
// dependencies without sharing the cached slices with callers.
func TestDeclarationGraphMemoization(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.API("memo", func() {})
		parent := dsl.Type("Parent", func() {
			dsl.Attribute("kind", dsl.String)
		})
		dsl.Type("Holder", func() {
			dsl.Attribute("pet", func() {
				dsl.Extend(parent)
				dsl.Meta("openapi:allOf:reference", "true")
				dsl.Attribute("note", dsl.String)
			})
		})
	})
	a := NewAnalyzer(nil, false)
	for _, typ := range root.Types {
		a.declarationTypes[typ.ID()] = typ
	}
	holder := root.UserType("Holder")
	first := a.declarationNeighbors(holder)
	require.Contains(t, first, root.UserType("Parent"))
	require.Contains(t, a.declarationNeighborCache, holder.ID())
	_ = append(first, holder)
	second := a.declarationNeighbors(holder)
	require.Equal(t, len(first), len(second), "appending to a returned slice does not change the cache")
	require.NotContains(t, second, holder)
}

// TestOccurrenceContextReadsDeclarationGraph gives a node that denotes a
// declaration the publication context of that declaration's whole graph,
// including an inline composed field whose contributor its own attribute
// copies may not carry.
func TestOccurrenceContextReadsDeclarationGraph(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.API("graph", func() {})
		parent := dsl.Type("Parent", func() {
			dsl.Attribute("kind", dsl.String)
			dsl.Attribute("thumb", dsl.Bytes)
		})
		dsl.Type("Holder", func() {
			dsl.Attribute("pet", func() {
				dsl.Extend(parent)
				dsl.Meta("openapi:allOf:reference", "true")
				dsl.Attribute("note", dsl.String)
			})
		})
		dsl.Type("Plain", func() {
			dsl.Attribute("name", dsl.String)
		})
	})
	a := NewAnalyzer(nil, false)
	for _, typ := range root.Types {
		a.declarationTypes[typ.ID()] = typ
	}
	for _, tc := range []struct {
		name, declaration string
		want              bool
	}{
		{name: "inline contributor in declaration graph", declaration: "Holder", want: true},
		{name: "declaration without publication neighbors", declaration: "Plain"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			attribute, plan := a.declarationRepresentation(root.UserType(tc.declaration))
			// The node's own attribute contributes nothing: the context comes
			// from the declaration the node denotes.
			copied := &expr.AttributeExpr{Type: expr.String}
			context := a.occurrenceContext(copied, plan)
			if tc.want {
				require.Contains(t, context, "Parent=")
				// The copied attribute names no declaration, so this resolves
				// through the plan node's target declaration.
				require.Same(t, root.UserType(tc.declaration), a.occurrenceDeclaration(copied, plan))
				require.NotNil(t, attribute)
			} else {
				require.Empty(t, context)
			}
		})
	}
}

// TestParameterCompositionReadsGraphSource checks the acquisition owner of a
// text, form or multipart occurrence: the plan copy analyzed at a composed
// position keeps the reference marker but no composition bases, and the
// original graph attribute paired with that position supplies them. An
// attribute without the marker at the same position acquires none, and
// outside parameter acquisition the copy is its own owner.
func TestParameterCompositionReadsGraphSource(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.API("parameter composition", func() {})
		parent := dsl.Type("Parent", func() {
			dsl.Attribute("kind", dsl.String)
		})
		dsl.Type("Holder", func() {
			dsl.Attribute("pet", func() {
				dsl.Extend(parent)
				dsl.Meta("openapi:allOf:reference", "true")
				dsl.Attribute("note", dsl.String)
			})
		})
	})
	parent := root.UserType("Parent")
	attribute := &expr.AttributeExpr{Type: root.UserType("Holder")}
	context := expr.NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{
		TargetOccurrence: occurrence, Codec: expr.ValueCodecForm, Use: expr.ValuePlanSchema,
	})
	require.NoError(t, err)
	pet := valuePlanMemberNode(plan.Root().Underlying(), "pet")
	require.True(t, pet.Valid())
	copied := pet.Attribute()
	require.Equal(t, []string{"true"}, copied.Meta["openapi:allOf:reference"])
	require.Empty(t, copied.CompositionBases(), "value capture keeps only the constraint-local contract")

	a := NewAnalyzer(nil, false)
	compose := func(attribute *expr.AttributeExpr) *Schema {
		restore := a.schemaPlanScope(pet, expr.ValuePlanNode{})
		defer restore()
		return a.analyzeCompositionReferences(&Schema{Type: "object"}, attribute, "pet")
	}
	_, restoreSources := a.parameterSchemaTarget(attribute, plan.Root(), expr.ValueCodecForm)
	composed := compose(copied)
	require.Len(t, composed.AllOf, 2)
	require.Equal(t, toRef("Parent"), composed.AllOf[0].Ref)
	require.Same(t, parent, a.constructions[composed].slots[0].base)
	require.Empty(t, compose(&expr.AttributeExpr{Type: expr.String}).AllOf, "the occurrence marker requests reference identity")
	restoreSources()
	require.Empty(t, compose(copied).AllOf, "without a paired graph position the copy has no bases")
}
