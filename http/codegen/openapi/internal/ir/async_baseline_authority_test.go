package ir

import (
	"testing"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAsyncFreshBaselineKeepsSiblingAnnotations(t *testing.T) {
	plain := &expr.UserTypeExpr{TypeName: "Plain", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{
		{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String}},
	}}}
	annotated := &expr.UserTypeExpr{TypeName: "Annotated", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{
		{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String, DefaultValue: "owned-default"}},
	}}}
	attribute := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "plain", Attribute: &expr.AttributeExpr{Type: plain}},
		{Name: "annotated", Attribute: &expr.AttributeExpr{Type: annotated}},
	}}
	analyzer := NewAnalyzer(expr.NewRandom("async-baseline-authority"), false)
	endpoint := inlineAsyncTestEndpoint(t, attribute)
	prepared := analyzeAsyncSchema(analyzer, attribute, endpoint, false, "owned-message")
	analyzer.finalizeRepresentations()
	schema := materializeAsyncSchema(prepared, analyzer.schemas).schema
	for _, field := range []string{"plain", "annotated"} {
		child := schema.Properties[field]
		require.NotNil(t, child)
		value := child.Properties["value"]
		require.NotNil(t, value)
		if field == "plain" {
			assert.Nil(t, value.DefaultValue)
		} else {
			assert.Equal(t, "owned-default", value.DefaultValue,
				"a fresh message analyzer must not merge distinct inline annotation owners")
		}
	}
}

func TestAsyncBaselineReservesNewRecursiveCut(t *testing.T) {
	node := &expr.UserTypeExpr{TypeName: "RootlessRecursive", AttributeExpr: &expr.AttributeExpr{}}
	node.Type = &expr.Object{
		{Name: "label", Attribute: &expr.AttributeExpr{Type: expr.String, DefaultValue: "owned"}},
		{Name: "child", Attribute: &expr.AttributeExpr{Type: node}},
	}
	attribute := &expr.AttributeExpr{Type: node}
	analyzer := NewAnalyzer(expr.NewRandom("rootless-recursive"), false)
	endpoint := inlineAsyncTestEndpoint(t, attribute)
	prepared := analyzeAsyncSchema(analyzer, attribute, endpoint, false, "recursive")
	analyzer.finalizeRepresentations()
	result := materializeAsyncSchema(prepared, analyzer.schemas).schema
	child := result.Properties["child"]
	require.NotNil(t, child)
	require.NotEmpty(t, child.Ref)
	name, local := schemaComponentName(child.Ref)
	require.True(t, local)
	component := analyzer.schemas[name]
	require.NotNil(t, component)
	require.Equal(t, child.Ref, component.Properties["child"].Ref)
	require.Equal(t, "owned", component.Properties["label"].DefaultValue)
}

func TestAsyncBaselineMemoKeepsOccurrenceOwnership(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String}},
	}}
	endpoint := inlineAsyncTestEndpoint(t, attribute)
	target := representation.PrepareStreamSchema(endpoint, attribute, false)
	plan := representationRoot(attribute, target)
	calls := 0
	analyzer := materializationAnalyzer(&calls)
	sampler := asyncSamplerAttribute(attribute)
	analyzer.asyncAcquisition = &asyncBaselineAcquisition{shape: asyncInlineShape{attribute: sampler}, schemas: make(map[asyncBaselineKey]*Schema)}
	analyzer.suppressExamples = func(*expr.AttributeExpr, bool) bool {
		return true
	}
	first := analyzer.analyzeSchemaPlan(attribute, "one", plan)
	second := analyzer.analyzeSchemaPlan(attribute, "one", plan)
	require.Same(t, first, second, "same captured occurrence and usage context reuses its owned acquisition")
	other := analyzer.analyzeSchemaPlan(attribute, "two", plan)
	require.NotSame(t, first, other, "another usage context cannot inherit this memo binding")
	require.Zero(t, calls, "acquisition never invokes example materialization")
	analyzer.asyncAcquisition, analyzer.suppressExamples = nil, nil
	prepared := &asyncSchema{schema: first, sampler: sampler, context: "one", constructions: analyzer.constructions}
	materialized := materializeAsyncSchema(prepared, analyzer.schemas)
	applyPreparedAsyncExamples(analyzer, materialized.schema, prepared, materialized.structures)
	require.Equal(t, 2, calls, "one callback for the inline object and one for its child")
}

func TestAsyncBaselineBranchRouting(t *testing.T) {
	for _, tagged := range []bool{false, true} {
		t.Run(map[bool]string{false: "untagged", true: "tagged"}[tagged], func(t *testing.T) {
			attribute := &expr.AttributeExpr{Type: &expr.Union{TypeName: "Route", Untagged: !tagged, Values: []*expr.NamedAttributeExpr{
				{Name: "zeta", Attribute: &expr.AttributeExpr{Type: expr.String}},
				{Name: "alpha", Attribute: &expr.AttributeExpr{Type: expr.Int}},
			}}}
			endpoint := inlineAsyncTestEndpoint(t, attribute)
			target := representation.PrepareStreamSchema(endpoint, attribute, false)
			plan := representationRoot(attribute, target)
			sampler := asyncSamplerAttribute(attribute)
			analyzer := NewAnalyzer(expr.NewRandom("branch-routing"), false)
			analyzer.plan = plan
			acquisition := &asyncBaselineAcquisition{shape: asyncInlineShape{attribute: sampler}, schemas: make(map[asyncBaselineKey]*Schema)}
			analyzer.asyncAcquisition = acquisition
			for _, branch := range plan.Branches() {
				restore := analyzer.asyncShapeScope(branch.Node)
				if tagged {
					require.True(t, acquisition.shape.reference)
				} else {
					expected := expr.String
					if branch.Tag == "alpha" {
						expected = expr.Int
					}
					require.Equal(t, expected, acquisition.shape.attribute.Type, "branch %s owns its own sampler position", branch.Tag)
				}
				restore()
				require.Same(t, sampler, acquisition.shape.attribute)
			}
			if tagged {
				union := attribute.Type.(*expr.Union)
				ref := analyzer.ensureUnionBranchSchema(union, union.Values[0])
				require.NotEmpty(t, ref)
				require.Empty(t, acquisition.schemas, "synthetic envelopes use their existing component producer, not the inline acquisition memo")
				require.Same(t, acquisition, analyzer.asyncAcquisition)
			}
		})
	}
}

func TestAsyncBaselineMemoNoRefPresence(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: expr.String}
	analyzer := NewAnalyzer(expr.NewRandom("no-ref-presence"), false)
	analyzer.asyncAcquisition = &asyncBaselineAcquisition{shape: asyncInlineShape{attribute: attribute}, schemas: make(map[asyncBaselineKey]*Schema)}
	omitted := analyzer.analyzeSchemaPlan(attribute, "same", expr.ValuePlanNode{})
	explicitFalse := analyzer.analyzeSchemaPlan(attribute, "same", expr.ValuePlanNode{}, false)
	explicitTrue := analyzer.analyzeSchemaPlan(attribute, "same", expr.ValuePlanNode{}, true)
	require.NotSame(t, omitted, explicitFalse, "analyzeSchema treats any supplied variadic argument as noRef")
	require.Same(t, explicitFalse, explicitTrue)
}
