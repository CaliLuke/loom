package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	commontestdata "github.com/CaliLuke/loom/codegen/testdata"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

type (
	annotationPathStep struct {
		Example  any
		Children map[string]any
	}
	annotationPathBinding struct {
		steps  []annotationPathStep
		owners []componentAnnotationIdentity
	}
)

func TestBytesPreservesEntireBaselineAnnotationPath(t *testing.T) {
	for _, neutral := range []bool{true, false} {
		t.Run(map[bool]string{true: "neutral", false: "planned"}[neutral], func(t *testing.T) {
			root := codegen.RunDSL(t, commontestdata.DeclarationAuthorityBytesDSL)
			a, bodies := annotationBodies(t, root, neutral)
			endpoint := bodies.Services["declaration-authority"]["ordinary"]
			binding := endpoint.ResponseBodies[200][0]
			require.NotNil(t, binding)
			before := map[string]annotationPathBinding{}
			for _, field := range []string{"entries", "items"} {
				before[field] = captureAnnotationPath(t, a, annotationUse(t, binding, a.schemas, field))
				want := 3
				if field == "items" {
					want = 4
				}
				require.Len(t, before[field].steps, want)
				terminal := before[field].steps[want-1]
				for _, child := range []string{"count", "blob"} {
					require.Contains(t, terminal.Children, child)
					require.NotNil(t, terminal.Children[child])
				}
				require.Nil(t, before[field].steps[0].Example, "outer reference must not borrow a component's annotation")
				first := 1
				if neutral {
					first = want - 1
				}
				for _, step := range before[field].steps[first:] {
					require.NotNil(t, step.Example)
				}
			}
			entries, items := before["entries"].owners, before["items"].owners
			if neutral {
				require.NotEqual(t, entries[len(entries)-1], items[len(items)-1], "neutral aliases retain distinct acquired terminal owners")
			} else {
				require.Equal(t, entries[len(entries)-1], items[len(items)-1], "planned aliases share the selected terminal owner, not merely equal values")
			}
			owners := snapshotAnnotationOwners(a)
			a.finalizeRepresentations()
			require.NoError(t, a.Err())
			require.Equal(t, owners, snapshotAnnotationOwners(a))
			for _, field := range []string{"entries", "items"} {
				actual := annotationPath(t, annotationUse(t, binding, a.schemas, field), a.schemas)
				require.Equal(t, before[field].steps, actual, "each acquired root and child annotation survives final naming")
			}
			if neutral {
				document, err := BuildDocument(root.API, root.Types, root.ResultTypes)
				require.NoError(t, err)
				for _, field := range []string{"entries", "items"} {
					require.Equal(t, before[field].steps, annotationPath(t, annotationUse(t, binding, document.Components.Schemas, field), document.Components.Schemas))
				}
			}
		})
	}
}

// annotationBodies exercises the normal owners while retaining their actual
// returned roots for assertions on either side of final naming.
func annotationBodies(t *testing.T, root *expr.RootExpr, declarations bool, options ...AnalyzerOption) (*Analyzer, *BodyTypes) {
	t.Helper()
	for _, typ := range root.Types {
		require.False(t, metaBool(typ.Attribute().Meta, "openapi:typename:canonical"), "this fixture exercises preferred names, not canonical claims")
	}
	a := NewAnalyzer(root.API.ExampleGenerator, false, options...)
	a.initializeDiscriminatorHints(root.API, root.Types, root.ResultTypes)
	if declarations {
		analyzeComponentTypes(a, root.Types, root.ResultTypes)
	}
	a.recordDiscriminatorChildren(root.Types, root.ResultTypes)
	bodies := &BodyTypes{
		Services:  make(map[string]map[string]*EndpointBodies),
		prepared:  make(map[string]*transportir.Service),
		locations: make(map[*expr.AttributeExpr]*Schema),
	}
	analyzeServiceBodies(a, root.API, bodies)
	for _, service := range orderedStringKeys(bodies.Services) {
		for _, method := range orderedStringKeys(bodies.Services[service]) {
			analyzeAsyncSchemas(a, bodies.Services[service][method].prepared)
		}
	}
	require.NoError(t, a.Err())
	return a, bodies
}

func annotationUse(t *testing.T, root *Schema, schemas map[string]*Schema, field string) *Schema {
	t.Helper()
	object := annotationObject(t, root, schemas)
	require.Contains(t, object.Properties, field)
	if field == "entries" {
		require.NotNil(t, object.Properties[field].AdditionalProperties)
		return object.Properties[field].AdditionalProperties.Schema
	}
	return object.Properties[field].Items
}

func annotationPath(t *testing.T, start *Schema, schemas map[string]*Schema) []annotationPathStep {
	t.Helper()
	var steps []annotationPathStep
	seen := map[*Schema]bool{}
	for current := start; ; {
		require.NotNil(t, current, "every local reference resolves")
		require.False(t, seen[current], "alias path terminates")
		seen[current] = true
		children := map[string]any{}
		for name, child := range current.Properties {
			children[name] = copyComponentAnnotation(child.Example)
		}
		steps = append(steps, annotationPathStep{copyComponentAnnotation(current.Example), children})
		if current.Ref == "" {
			return steps
		}
		name, ok := schemaComponentName(current.Ref)
		require.True(t, ok)
		current = schemas[name]
	}
}

func captureAnnotationPath(t *testing.T, a *Analyzer, start *Schema) annotationPathBinding {
	t.Helper()
	result := annotationPathBinding{steps: annotationPath(t, start, a.schemas)}
	for current := start; current.Ref != ""; {
		name, ok := schemaComponentName(current.Ref)
		require.True(t, ok)
		current = a.schemas[name]
		require.NotNil(t, current)
		owner := annotationOwner(t, a, name)
		assertOwnedAnnotation(t, a, owner, current)
		result.owners = append(result.owners, owner)
	}
	return result
}

func annotationOwner(t *testing.T, a *Analyzer, name string) componentAnnotationIdentity {
	t.Helper()
	naming, found := a.schemaNames[name]
	require.True(t, found)
	var owner componentAnnotationIdentity
	matches := 0
	for identity := range a.componentAnnotations {
		if identity.declaration == naming.logical && identity.baseline == naming.baseline && identity.context == exampleContext("component", naming.annotationName) {
			owner = identity
			matches++
		}
	}
	require.Equal(t, 1, matches, "the acquired component must select one annotation binding")
	require.True(t, owner.examplePlan.Valid())
	return owner
}

func assertOwnedAnnotation(t *testing.T, a *Analyzer, owner componentAnnotationIdentity, schema *Schema) {
	t.Helper()
	local := schema
	if len(local.AllOf) > 0 {
		local = local.AllOf[len(local.AllOf)-1]
	}
	node := owner.examplePlan
	for node.Underlying().Valid() {
		node = node.Underlying()
	}
	assertAnnotationPosition(t, a, owner, owner.context, local.Example)
	if len(local.Properties) > 0 {
		require.NotNil(t, local.Example)
	}
	for _, member := range node.Members() {
		child := local.Properties[member.WireName]
		if len(local.Properties) > 0 {
			require.NotNil(t, child, "owned object retains member %s", member.WireName)
			require.NotNil(t, child.Example, "owned member retains its representative")
		}
		if child != nil {
			assertAnnotationPosition(t, a, owner, childExampleContext(owner.context, "property", member.WireName), child.Example)
		}
	}
}

func assertAnnotationPosition(t *testing.T, a *Analyzer, owner componentAnnotationIdentity, context string, actual any) {
	t.Helper()
	positions := a.componentAnnotations[owner]
	var value componentExamples
	matches := 0
	for position, sample := range positions {
		if position.context == context {
			value = sample
			matches++
		}
	}
	if matches == 0 {
		require.Nil(t, actual, "an annotation-free acquired layer stays empty")
		return
	}
	require.Equal(t, 1, matches)
	require.Equal(t, value.representative, actual, "sample comes from this exact acquired owner and position")
}

func assertPreparedAnnotation(t *testing.T, a *Analyzer, target *transportir.ValueTarget, owner componentAnnotationIdentity, schema *Schema) {
	t.Helper()
	require.NotNil(t, target)
	node := owner.examplePlan
	require.Contains(t, target.ExampleSets, node, "selected owner belongs to this prepared target")
	for node.Underlying().Valid() {
		node = node.Underlying()
	}
	local := schema
	if len(local.AllOf) > 0 {
		local = local.AllOf[len(local.AllOf)-1]
	}
	positions := []struct {
		node   expr.ValuePlanNode
		schema *Schema
	}{{node, local}}
	for _, member := range node.Members() {
		if child := local.Properties[member.WireName]; child != nil {
			positions = append(positions, struct {
				node   expr.ValuePlanNode
				schema *Schema
			}{member.Node, child})
		}
	}
	for _, position := range positions {
		set := target.ExampleSets[position.node]
		require.NotNil(t, set)
		require.True(t, set.Prepared)
		require.NotNil(t, set.Representative)
		project := preparedOpenAPIExampleValue
		if a.declaredExampleValue {
			project = preparedDeclaredExampleValue
		}
		value, ok := project(*set.Representative)
		require.True(t, ok)
		require.Equal(t, value, position.schema.Example)
	}
}
