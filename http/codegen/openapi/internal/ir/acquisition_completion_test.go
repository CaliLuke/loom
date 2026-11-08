package ir

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func TestAcquisitionCompletionBindings(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.API("bindings", func() {
		})
		detail := dsl.Type("Detail", func() {
			dsl.Attribute("note", dsl.String)
			dsl.Example(map[string]any{"note": "detail"})
		})
		parent := dsl.Type("Parent", func() {
			dsl.Meta("openapi:discriminator:propertyName", "kind")
			dsl.Attribute("kind", dsl.String)
			dsl.Attribute("detail", detail, func() {
				dsl.Description("local detail")
			})
			dsl.Required("kind")
		})
		child := dsl.Type("Child", func() {
			dsl.Extend(parent)
			dsl.Meta("openapi:allOf:reference", "true")
			dsl.Attribute("thumb", dsl.Bytes)
		})
		dsl.Service("binding", func() {
			for _, typ := range []expr.UserType{parent, child} {
				dsl.Method(typ.Name(), func() {
					dsl.Result(typ)
					dsl.HTTP(func() {
						dsl.GET("/" + typ.Name())
					})
				})
			}
		})
	})
	a := NewAnalyzer(root.API.ExampleGenerator, false)
	a.initializeDiscriminatorHints(root.API, root.Types, root.ResultTypes)
	analyzeComponentTypes(a, root.Types, root.ResultTypes)
	a.recordDiscriminatorChildren(root.Types, root.ResultTypes)
	bodies := &BodyTypes{Services: make(map[string]map[string]*EndpointBodies), prepared: make(map[string]*transportir.Service), locations: make(map[*expr.AttributeExpr]*Schema)}
	analyzeServiceBodies(a, root.API, bodies)
	require.NoError(t, a.Err())
	require.Zero(t, a.schemaDepth)
	require.Nil(t, a.contributorRoot)
	require.NotEmpty(t, a.contributors)
	componentCount, contributorCount := len(a.componentAcquisitions), len(a.contributors)
	owners := snapshotAnnotationOwners(a)
	a.completeSchemaScope()
	require.Len(t, a.componentAcquisitions, componentCount)
	require.Len(t, a.contributors, contributorCount)
	require.Equal(t, owners, snapshotAnnotationOwners(a), "completion cannot acquire a new component or example binding")
	for _, key := range slices.Sorted(maps.Keys(a.contributors)) {
		contributor := a.contributors[key]
		require.Equal(t, key, contributor.key)
		require.True(t, contributor.complete)
		require.NotNil(t, contributor.component)
		entry := contributor.component
		require.True(t, entry.complete)
		require.Same(t, entry, a.componentAcquisitions[entry.name])
		require.NotEqual(t, root.UserType("Detail").ID(), a.schemaNames[entry.name].logical, "a nested named property cannot consume the contributor's root binding")
		var captured preparedComponentBinding
		for _, registration := range a.schemasByFingerprint[contributor.fingerprint] {
			if registration.ref == toRef(entry.name) {
				captured = registration.binding
				break
			}
		}
		selected := findMatchingSchemaRef(a.schemasByFingerprint[contributor.fingerprint], contributor.hint, contributor.claim != "", captured)
		require.Equal(t, toRef(entry.name), selected)
		t.Logf("C=%s F=%s hint=%q claim=%q handle=%s", key, contributor.fingerprint, contributor.hint, contributor.claim, entry.name)
		projection := a.planBaseline(contributor.schema, contributor.attribute, contributor.plan)
		require.Equal(t, projectionComplete, projection.state)
		require.Same(t, projection.result, a.projectBaseline(contributor.schema, contributor.attribute, contributor.plan), "post-scope lookup must reuse the drained projection")
	}
	for _, projection := range a.projectionComponents {
		require.Equal(t, projectionComplete, projection.state)
		require.Same(t, projection.body.result, a.schemas[projection.name], "publication retains a completed result cell rather than an unfinished value snapshot")
	}
}

func TestAcquisitionFailureRollsBackProvisionalOwners(t *testing.T) {
	failure := errors.New("authored example failure")
	fail := false
	a := NewAnalyzer(expr.NewRandom("rollback"), false, WithExampleValue(func(_ *expr.AttributeExpr, value any) (any, bool) {
		if fail {
			panic(failure)
		}
		return value, true
	}))
	prior := a.AnalyzeSchema(acquisitionProtocolAttribute("Prior", "prior", expr.String, nil))
	beforeSchemas := maps.Clone(a.schemas)
	beforeReferences := maps.Clone(a.schemasByFingerprint)
	fail = true
	require.PanicsWithValue(t, failure, func() {
		a.AnalyzeSchema(acquisitionProtocolAttribute("Rejected", "rejected", expr.Int, nil))
	})
	require.Zero(t, a.schemaDepth)
	require.Equal(t, beforeSchemas, a.schemas)
	require.Equal(t, beforeReferences, a.schemasByFingerprint)
	require.Len(t, a.componentAcquisitions, 1)
	require.Empty(t, a.contributors)
	fail = false
	reused := a.AnalyzeSchema(acquisitionProtocolAttribute("Again", "again", expr.String, nil))
	require.Equal(t, prior.Ref, reused.Ref)
	require.Len(t, a.Components(), 1)
}

func TestCanonicalComparisonBindsContributorBeforeTraversal(t *testing.T) {
	cycle := func(size int) *expr.AttributeExpr {
		nodes := make([]*expr.UserTypeExpr, size)
		for index := range nodes {
			nodes[index] = &expr.UserTypeExpr{TypeName: "Node", UID: fmt.Sprintf("node-%d-%d", size, index), AttributeExpr: &expr.AttributeExpr{Meta: expr.MetaExpr{"openapi:typename": {"Node"}, "openapi:typename:canonical": {"true"}}}}
		}
		for index, node := range nodes {
			node.Type = &expr.Object{{Name: "next", Attribute: &expr.AttributeExpr{Type: nodes[(index+1)%size]}}, {Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String}}}
		}
		return &expr.AttributeExpr{Type: nodes[0]}
	}
	var binding *projectedContributor
	observed := 0
	a := NewAnalyzer(expr.NewRandom("comparison-binding"), false, WithExampleValue(func(*expr.AttributeExpr, any) (any, bool) {
		if binding != nil {
			observed++
			if binding.component == nil {
				t.Error("the selected canonical owner must be bound before candidate body traversal")
			}
		}
		return nil, false
	}))
	first := a.AnalyzeSchema(cycle(1))
	attribute := cycle(2)
	binding = &projectedContributor{}
	func() {
		defer a.schemaAnalysisScope()()
		result := a.analyzeUserType(attribute, attribute.Type.(expr.UserType), "comparison", false, binding)
		require.Equal(t, first.Ref, result.Ref)
	}()
	require.Positive(t, observed)
	require.Same(t, a.componentAcquisitions["Node"], binding.component)
}

func TestAcquisitionCompletionMappingBatch(t *testing.T) {
	var first *Document
	for _, reverse := range []bool{false, true} {
		t.Run(fmt.Sprint(reverse), func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				dsl.API("mapping batch", func() {
					dsl.Meta("openapi:example", "false")
				})
				parent := dsl.Type("BatchParent", func() {
					dsl.Meta("openapi:typename:canonical", "true")
					dsl.Attribute("kind", dsl.String)
					dsl.Required("kind")
					dsl.Meta("openapi:discriminator:propertyName", "kind")
					mappings := []string{"one=BatchOne", "two=BatchTwo"}
					if reverse {
						slices.Reverse(mappings)
					}
					dsl.Meta("openapi:discriminator:mapping", mappings...)
				})
				for index, name := range []string{"BatchOne", "BatchTwo"} {
					dsl.Type(name, func() {
						dsl.Meta("openapi:typename:canonical", "true")
						dsl.Extend(parent)
						dsl.Meta("openapi:allOf:reference", "true")
						dsl.Attribute("bytes", dsl.Bytes, func() {
							dsl.MinLength(index + 1)
						})
					})
				}
				dsl.Service("mapping", func() {
					dsl.Method("parent", func() {
						dsl.Result(parent)
						dsl.HTTP(func() {
							dsl.GET("/parent")
						})
					})
				})
			})
			document, err := BuildDocument(root.API, root.Types, root.ResultTypes)
			require.NoError(t, err)
			parent := document.Components.Schemas["BatchParent"]
			require.NotNil(t, parent)
			require.NotNil(t, parent.Discriminator)
			require.Equal(t, map[string]string{"one": toRef("BatchOne"), "two": toRef("BatchTwo")}, parent.Discriminator.Mapping)
			for _, name := range []string{"BatchOne", "BatchTwo"} {
				child := document.Components.Schemas[name]
				require.NotNil(t, child)
				require.Len(t, child.AllOf, 2)
				require.Equal(t, toRef("BatchParent"), child.AllOf[0].Ref)
				require.Equal(t, "base64", child.AllOf[1].Properties["bytes"].ContentEncoding)
			}
			require.NotEqual(t, document.Components.Schemas["BatchOne"].AllOf[1].Properties["bytes"], document.Components.Schemas["BatchTwo"].AllOf[1].Properties["bytes"], "each mapped child retains its own byte constraint")
			if first != nil {
				require.Equal(t, first.Components, document.Components, "all mapping slots must be installed before any copied graph or cache observes them")
			}
			first = document
		})
	}
}
