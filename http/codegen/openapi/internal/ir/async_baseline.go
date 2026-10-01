package ir

import (
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

type (
	// asyncBaselineAcquisition owns inline annotation positions before any
	// ordinary component fingerprint can merge them. It lasts one message.
	asyncBaselineAcquisition struct {
		shape   asyncInlineShape
		schemas map[asyncBaselineKey]*Schema
	}

	asyncBaselineKey struct {
		plan        expr.ValuePlanNode
		examplePlan expr.ValuePlanNode
		position    *expr.AttributeExpr
		context     string
		reference   bool
		noRef       bool
	}
)

func (a *Analyzer) acquireAsyncBaseline(
	attr *expr.AttributeExpr,
	plan expr.ValuePlanNode,
	sampler *expr.AttributeExpr,
	context string,
) *asyncSchema {
	return a.acquireAsyncPreparedBaseline(attr, plan, nil, sampler, context)
}

func (a *Analyzer) acquireAsyncTargetBaseline(
	attr *expr.AttributeExpr,
	target *transportir.ValueTarget,
	sampler *expr.AttributeExpr,
	context string,
) *asyncSchema {
	plan := representationRoot(attr, target)
	return a.acquireAsyncPreparedBaseline(attr, plan, target, sampler, context)
}

func (a *Analyzer) acquireAsyncPreparedBaseline(
	attr *expr.AttributeExpr,
	plan expr.ValuePlanNode,
	target *transportir.ValueTarget,
	sampler *expr.AttributeExpr,
	context string,
) *asyncSchema {
	prior, priorAcquisition := a.suppressExamples, a.asyncAcquisition
	a.asyncAcquisition = &asyncBaselineAcquisition{shape: asyncInlineShape{attribute: sampler}, schemas: make(map[asyncBaselineKey]*Schema)}
	a.suppressExamples = func(*expr.AttributeExpr, bool) bool {
		return true
	}
	defer func() {
		a.suppressExamples, a.asyncAcquisition = prior, priorAcquisition
	}()
	return &asyncSchema{
		schema: a.analyzeOccurrence(attr, context, plan), attribute: attr, sampler: sampler,
		context: context, target: target, constructions: a.constructions,
	}
}

func (a *Analyzer) analyzeAsyncNamedBaseline(attr *expr.AttributeExpr, typ expr.UserType, context string, noRef bool) *Schema {
	acquisition := a.asyncAcquisition
	if schema, overlaid := a.analyzeUserTypeOverlay(attr, typ, context, noRef); overlaid {
		return schema
	}
	// A declared cut uses the actual handle produced by declaration registration,
	// not a new selection from shape-equivalent ordinary components. Overlays
	// remain owned by analyzeUserTypeOverlay before this boundary.
	if binding := a.declarationRoots[a.plan.TargetDeclarationID()]; binding != nil {
		result := *binding
		return &result
	}
	// Rootless recursive and synthetic definitions retain the ordinary owner's
	// registration/reservation rules. Inline acquisition cannot leak into them.
	a.asyncAcquisition = nil
	defer func() {
		a.asyncAcquisition = acquisition
	}()
	return a.analyzeUserType(attr, typ, context, noRef)
}

// inline identifies the existing consumer's sampler role. Named recursive cuts
// and synthetic envelopes keep their ordinary component policy.
func (a *asyncBaselineAcquisition) inline() bool {
	_, named := asyncSamplerType(a.shape.attribute).(expr.UserType)
	return !named && !a.shape.reference
}

func (a *Analyzer) asyncShapeScope(child expr.ValuePlanNode) func() {
	acquisition := a.asyncAcquisition
	if acquisition == nil {
		return func() {}
	}
	previous := acquisition.shape
	parent := a.plan
	if parent.Valid() && child != parent && child != parent.Underlying() {
		switch {
		case child == parent.Element():
			name := "items"
			if _, ok := asyncSamplerType(previous.attribute).(*expr.Map); ok {
				name = "additional-properties"
			}
			acquisition.shape = asyncSamplerChild(previous.attribute, name, 0)
		default:
			for _, member := range parent.Members() {
				if member.Node == child {
					acquisition.shape = asyncSamplerChild(previous.attribute, member.WireName, 0)
					break
				}
			}
			for _, branch := range parent.Branches() {
				if branch.Node == child {
					acquisition.shape = asyncSamplerBranch(previous.attribute, branch.Tag)
					break
				}
			}
		}
	}
	return func() {
		acquisition.shape = previous
	}
}

// Branch plans retain declaration order while schema arms are sorted. Pair by
// the wire tag before selecting the existing sampler's structural position.
func asyncSamplerBranch(sampler *expr.AttributeExpr, tag string) asyncInlineShape {
	if union, ok := asyncSamplerType(sampler).(*expr.Union); ok {
		for index, branch := range sortedUnionValues(union) {
			if expr.UnionVariantTag(branch) == tag {
				return asyncSamplerChild(sampler, "branch", index)
			}
		}
	}
	return asyncInlineShape{}
}
