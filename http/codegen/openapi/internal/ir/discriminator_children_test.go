package ir

import (
	"maps"
	"slices"
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/stretchr/testify/require"
)

type preferredFamily struct {
	root                             *Schema
	parentRef, childRef, backedgeRef string
	parent, child                    *Schema
}

func TestDiscriminatorParentHintChildren(t *testing.T) {
	for _, tc := range []struct {
		name                         string
		hint, occurrenceHint         bool
		reference, composed, wrapped bool
		want                         map[string][]string
	}{
		{name: "direct allOf child", hint: true, reference: true, want: map[string][]string{"Pet": {"Cat"}}},
		{name: "composed parent", hint: true, reference: true, composed: true, wrapped: true, want: map[string][]string{"Pet": {"Cat"}}},
		{name: "child without allOf reference", hint: true},
		{name: "parent without hint", reference: true},
		{name: "composed parent without hint", reference: true, composed: true},
		// An occurrence hint describes its own variant, which no child references.
		{name: "occurrence hint", occurrenceHint: true, reference: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				dsl.API("parent", func() {})
				base := dsl.Type("Base", func() {
					dsl.Attribute("base", dsl.String)
				})
				parent := dsl.Type("Pet", func() {
					if !tc.occurrenceHint {
						dsl.Meta("openapi:typename:canonical", "true")
					}
					if tc.composed {
						dsl.Extend(base)
						dsl.Meta("openapi:allOf:reference", "true")
					}
					dsl.Attribute("kind", dsl.String)
					dsl.Required("kind")
					if tc.hint {
						dsl.Meta("openapi:discriminator:propertyName", "kind")
					}
				})
				cat := dsl.Type("Cat", func() {
					dsl.Meta("openapi:typename:canonical", "true")
					dsl.Extend(parent)
					dsl.Attribute("name", dsl.String)
					if tc.reference {
						dsl.Meta("openapi:allOf:reference", "true")
					}
				})
				// Grandchildren are direct children of Cat, not of Pet.
				dsl.Type("Kitten", func() {
					dsl.Extend(cat)
					dsl.Attribute("age", dsl.Int)
					dsl.Meta("openapi:allOf:reference", "true")
				})
				dsl.Type("Other", func() {
					dsl.Attribute("kind", dsl.String)
				})
				dsl.Service("parents", func() {
					dsl.Method("Show", func() {
						dsl.Result(parent, func() {
							if tc.occurrenceHint {
								dsl.Meta("openapi:discriminator:propertyName", "kind")
							}
						})
						dsl.HTTP(func() {
							dsl.GET("/parent")
						})
					})
				})
			})
			document, err := BuildDocument(root.API, root.Types, root.ResultTypes, WithExampleValue(nil))
			require.NoError(t, err)
			require.Equal(t, tc.want, document.Components.DiscriminatorChildren)
			if tc.wrapped {
				pet := document.Components.Schemas["Pet"]
				require.Nil(t, pet.Discriminator, "the hint is owned by the composed member, not the wrapper")
				require.Equal(t, toRef("Base"), pet.AllOf[0].Ref)
			}
			for parent, children := range tc.want {
				for _, child := range children {
					schema := document.Components.Schemas[child]
					require.NotNil(t, schema)
					require.Equal(t, toRef(parent), schema.AllOf[0].Ref, "children reference the published parent")
				}
			}
		})
	}
}

// TestDiscriminatorParentHintChildrenFollowAllocatedNames covers a hinted root
// whose JSON byte projection claims the public name. The child is published in
// that representation: its projected base and dependency edge use the
// allocated parent name. Independently prepared annotations remain distinct;
// no unreachable raw registration representation remains.
func TestDiscriminatorParentHintChildrenFollowAllocatedNames(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.API("parent", func() {})
		parent := dsl.Type("Pet", func() {
			dsl.Meta("openapi:discriminator:propertyName", "kind")
			dsl.Attribute("kind", dsl.String)
			dsl.Attribute("blob", dsl.Bytes, func() {
				dsl.MinLength(2)
			})
			dsl.Required("kind")
		})
		dsl.Type("Cat", func() {
			dsl.Extend(parent)
			dsl.Meta("openapi:allOf:reference", "true")
			dsl.Attribute("name", dsl.String)
		})
		dsl.Service("parents", func() {
			dsl.Method("Show", func() {
				dsl.Payload(parent)
				dsl.Result(parent)
				dsl.HTTP(func() {
					dsl.POST("/parent")
				})
			})
		})
	})
	a, bodies := annotationBodies(t, root, true, WithExampleValue(nil))
	endpoint := bodies.Services["parents"]["Show"]
	families := captureBodyFamilies(t, a, endpoint)
	owners := snapshotAnnotationOwners(a)
	a.finalizeRepresentations()
	require.NoError(t, a.Err())
	require.Equal(t, owners, snapshotAnnotationOwners(a))
	for _, family := range families {
		child := assertPreferredFamily(t, a, family)
		require.Equal(t, "base64", child.AllOf[1].Properties["blob"].ContentEncoding, "inherited bytes use the child's JSON representation")
		require.NotNil(t, child.AllOf[1].Properties["blob"].Not)
	}
	assertFamilyClosure(t, a, families)
	document, err := BuildDocument(root.API, root.Types, root.ResultTypes, WithExampleValue(nil))
	require.NoError(t, err)
	require.Equal(t, a.schemas, document.Components.Schemas)
	require.Equal(t, a.discriminatorChildren, document.Components.DiscriminatorChildren)
}

// TestDiscriminatorDefaultMappingDeclaredLater resolves a defaultMapping target
// declared after its parent once every declaration has been analyzed.
func TestDiscriminatorDefaultMappingDeclaredLater(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.API("parent", func() {})
		parent := dsl.Type("Pet", func() {
			dsl.Meta("openapi:discriminator:propertyName", "kind")
			dsl.Meta("openapi:discriminator:defaultMapping", "Cat")
			dsl.Attribute("kind", dsl.String)
			dsl.Required("kind")
		})
		dsl.Type("Cat", func() {
			dsl.Extend(parent)
			dsl.Meta("openapi:allOf:reference", "true")
			dsl.Attribute("name", dsl.String)
		})
		dsl.Service("parents", func() {
			dsl.Method("Show", func() {
				dsl.Result(parent)
				dsl.HTTP(func() {
					dsl.GET("/parent")
				})
			})
		})
	})
	a, bodies := annotationBodies(t, root, true, WithExampleValue(nil))
	binding := bodies.Services["parents"]["Show"].ResponseBodies[200][0]
	_, parent := annotationComponent(t, binding, a.schemas)
	target := a.declarationRoots["Cat"]
	require.NotNil(t, target, "the later declaration is acquired before resolving the default")
	require.Equal(t, target.Ref, parent.Discriminator.DefaultMapping, "default selects the declaration, not a contributor")
	targetName, child := annotationComponent(t, target, a.schemas)
	owner := annotationOwner(t, a, targetName)
	assertOwnedAnnotation(t, a, owner, child)
	before := copyAnnotationSchema(child)
	originalParentRef := child.AllOf[0].Ref
	parentDeclaration := a.declarationRoots["Pet"]
	require.Equal(t, parentDeclaration.Ref, originalParentRef)
	owners := snapshotAnnotationOwners(a)
	// Declaration roots are not mutable publication roots. Keep the existing
	// allocator's exact association rather than fabricate a tracked occurrence.
	names := finalizeAnnotationNames(t, a)
	require.Equal(t, owners, snapshotAnnotationOwners(a))
	_, parent = annotationComponent(t, binding, a.schemas)
	require.Equal(t, toRef(names[targetName]), parent.Discriminator.DefaultMapping)
	child = a.schemas[names[targetName]]
	require.NotNil(t, child)
	expected := mapSchemaReferences(before, func(ref, _ string) string {
		if local, ok := schemaComponentName(ref); ok {
			final, found := names[local]
			require.True(t, found)
			return toRef(final)
		}
		return ref
	})
	require.Equal(t, expected, child, "same completed declaration and all its samples survive naming")
	require.Contains(t, child.AllOf[1].Properties, "name")
	parentLocal, ok := schemaComponentName(originalParentRef)
	require.True(t, ok)
	require.Equal(t, toRef(names[parentLocal]), child.AllOf[0].Ref)
	document, err := BuildDocument(root.API, root.Types, root.ResultTypes, WithExampleValue(nil))
	require.NoError(t, err)
	require.Equal(t, a.schemas, document.Components.Schemas)
}

// TestDiscriminatorPlainParentPublishesJSONChild publishes a byte-free hinted
// parent through a JSON occurrence: its byte-bearing child is published in the
// JSON representation and the raw registrations are discarded.
func TestDiscriminatorPlainParentPublishesJSONChild(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.API("parent", func() {})
		parent := dsl.Type("Pet", func() {
			dsl.Meta("openapi:discriminator:propertyName", "kind")
			dsl.Attribute("kind", dsl.String)
			dsl.Required("kind")
		})
		dsl.Type("Cat", func() {
			dsl.Extend(parent)
			dsl.Meta("openapi:allOf:reference", "true")
			dsl.Attribute("thumb", dsl.Bytes)
		})
		dsl.Service("parents", func() {
			dsl.Method("Show", func() {
				dsl.Payload(parent)
				dsl.Result(parent)
				dsl.HTTP(func() {
					dsl.POST("/parent")
				})
			})
		})
	})
	a, bodies := annotationBodies(t, root, true, WithExampleValue(nil))
	families := captureBodyFamilies(t, a, bodies.Services["parents"]["Show"])
	owners := snapshotAnnotationOwners(a)
	a.finalizeRepresentations()
	require.NoError(t, a.Err())
	require.Equal(t, owners, snapshotAnnotationOwners(a))
	for _, family := range families {
		child := assertPreferredFamily(t, a, family)
		require.Equal(t, "base64", child.AllOf[1].Properties["thumb"].ContentEncoding)
		require.NotNil(t, child.AllOf[1].Properties["thumb"].Not)
	}
	assertFamilyClosure(t, a, families)
	document, err := BuildDocument(root.API, root.Types, root.ResultTypes, WithExampleValue(nil))
	require.NoError(t, err)
	require.Equal(t, a.schemas, document.Components.Schemas)
	require.Equal(t, a.discriminatorChildren, document.Components.DiscriminatorChildren)
}

func captureBodyFamilies(t *testing.T, a *Analyzer, endpoint *EndpointBodies) []preferredFamily {
	t.Helper()
	response := endpoint.prepared.Response.Responses[0]
	return []preferredFamily{
		capturePreferredFamily(t, a, endpoint.RequestBody, endpoint.prepared.Request.BodyValue),
		capturePreferredFamily(t, a, endpoint.ResponseBodies[200][0], response.DocumentValues["application/json"]),
	}
}

func capturePreferredFamily(t *testing.T, a *Analyzer, root *Schema, target *transportir.ValueTarget) preferredFamily {
	t.Helper()
	parentName, parent := annotationComponent(t, root, a.schemas)
	owner := annotationOwner(t, a, parentName)
	assertOwnedAnnotation(t, a, owner, parent)
	assertPreparedAnnotation(t, a, target, owner, parent)
	children := a.discriminatorChildren[parentName]
	require.Len(t, children, 1, "one direct child belongs to this selected parent occurrence")
	childName := children[0]
	child := a.schemas[childName]
	require.NotNil(t, child)
	childOwner := annotationOwner(t, a, childName)
	assertOwnedAnnotation(t, a, childOwner, child)
	matched := 0
	for _, contributor := range a.contributors {
		if contributor.plan == childOwner.examplePlan {
			require.True(t, contributor.complete)
			require.Equal(t, "Cat", contributor.plan.TargetDeclarationID())
			matched++
		}
	}
	require.Equal(t, 1, matched, "child annotations belong to the exact selected contributor plan")
	require.Len(t, child.AllOf, 2)
	backedgeName, backedge := annotationComponent(t, child.AllOf[0], a.schemas)
	require.Equal(t, owner, annotationOwner(t, a, backedgeName), "the inherited edge selects the same acquired annotation binding")
	require.Equal(t, parent, backedge, "projection copies retain the complete parent contract")
	require.Equal(t, []string{"kind"}, parent.Required)
	return preferredFamily{root, root.Ref, toRef(childName), child.AllOf[0].Ref, copyAnnotationSchema(parent), copyAnnotationSchema(child)}
}

func assertPreferredFamily(t *testing.T, a *Analyzer, family preferredFamily) *Schema {
	t.Helper()
	parentName, parent := annotationComponent(t, family.root, a.schemas)
	children := a.discriminatorChildren[parentName]
	require.Len(t, children, 1)
	child := a.schemas[children[0]]
	require.NotNil(t, child)
	rewrite := func(ref, _ string) string {
		switch ref {
		case family.parentRef, family.backedgeRef:
			return family.root.Ref
		case family.childRef:
			return toRef(children[0])
		default:
			return ref
		}
	}
	require.Equal(t, mapSchemaReferences(family.parent, rewrite), parent, "selected parent shape and all acquired samples remain intact")
	require.Equal(t, mapSchemaReferences(family.child, rewrite), child, "selected contributor shape and all acquired samples remain intact")
	require.Equal(t, family.root.Ref, child.AllOf[0].Ref)
	require.Equal(t, []string{"kind"}, parent.Required)
	return child
}

func assertFamilyClosure(t *testing.T, a *Analyzer, families []preferredFamily) {
	t.Helper()
	reached := map[string]bool{}
	var visit func(*Schema)
	visit = func(schema *Schema) {
		require.NotNil(t, schema)
		mapSchemaReferences(schema, func(ref, _ string) string {
			if ref == "" {
				return ref
			}
			name, ok := schemaComponentName(ref)
			require.True(t, ok)
			require.Contains(t, a.schemas, name)
			if !reached[name] {
				reached[name] = true
				visit(a.schemas[name])
				for _, child := range a.discriminatorChildren[name] {
					visit(&Schema{Ref: toRef(child)})
				}
			}
			return ref
		})
	}
	for _, family := range families {
		visit(family.root)
	}
	require.ElementsMatch(t, slices.Collect(maps.Keys(a.schemas)), slices.Collect(maps.Keys(reached)), "only body roots and their actual publication closure remain")
}

func annotationComponent(t *testing.T, root *Schema, schemas map[string]*Schema) (string, *Schema) {
	t.Helper()
	require.NotNil(t, root)
	name, ok := schemaComponentName(root.Ref)
	require.True(t, ok)
	schema := schemas[name]
	require.NotNil(t, schema, "every final local reference resolves")
	return name, schema
}

func copyAnnotationSchema(schema *Schema) *Schema {
	result := mapSchemaReferences(schema, func(ref, _ string) string {
		return ref
	}, func(source, target *Schema) {
		target.Example = copyComponentAnnotation(source.Example)
		if source.Examples != nil {
			target.Examples = copyComponentAnnotation(source.Examples).([]any)
		}
	})
	return result
}

func finalizeAnnotationNames(t *testing.T, a *Analyzer) map[string]string {
	t.Helper()
	require.NoError(t, a.Err())
	a.err = a.Err()
	a.discriminatorReferences = nil
	a.discardRegistrationOnlyVariants()
	components, reserved := a.representationComponents()
	allocated, names := allocateRepresentations(components, reserved)
	a.installRepresentations(allocated, names)
	require.NoError(t, a.Err())
	return names[a]
}
