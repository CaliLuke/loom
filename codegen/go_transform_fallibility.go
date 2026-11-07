package codegen

import (
	"github.com/CaliLuke/loom/internal/representation"
	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/expr"
)

type (
	// TransformPlan owns conversion code, its helper declarations and the error
	// channel required by authored representation conversions. ErrorAware plans
	// require a nonnil transformErr *error shared with their helper calls.
	TransformPlan struct {
		// Code initializes the selected destination from the source.
		Code string
		// Helpers contains the required recursive conversion functions.
		Helpers []*TransformFunctionData
		// ErrorAware reports whether conversion can fail at runtime. The caller
		// must check *transformErr before publishing the converted destination.
		ErrorAware bool
	}

	transformPair struct {
		source, target             expr.DataType
		sourceCopy, targetCopy     *expr.AttributeExpr
		sourceRef, targetRef       string
		sourceLayout, targetLayout transformLayout
	}

	transformLayout struct {
		pointer, ignoreRequired, useDefault, jsonPresence, elementPresence bool
	}

	transformFailureNode struct {
		fails, sourceDefaults          bool
		children                       []transformPair
		helper                         string
		source, target                 *expr.AttributeExpr
		concreteSource, concreteTarget *expr.AttributeExpr
		attrs                          *TransformAttrs
		path                           string
	}

	transformFailures struct {
		nodes   map[transformPair]*transformFailureNode
		helpers map[string]bool
		order   []transformPair
	}
)

// transformFailurePlan explores selected conversion pairs, then propagates local
// failures to a fixed point. A visited node stops graph discovery only: it never
// asserts that a recursive conversion is infallible.
func transformFailurePlan(source, target *expr.AttributeExpr, ta *TransformAttrs) (*transformFailures, error) {
	graph := &transformFailures{nodes: make(map[transformPair]*transformFailureNode), helpers: make(map[string]bool)}
	if err := graph.discover(source, target, ta, "root"); err != nil {
		return nil, err
	}
	for changed := true; changed; {
		changed = false
		for _, key := range graph.order {
			node := graph.nodes[key]
			for _, child := range node.children {
				dependency := graph.nodes[child]
				if dependency.fails && !node.fails {
					node.fails, changed = true, true
				}
				if dependency.sourceDefaults && !node.sourceDefaults {
					node.sourceDefaults, changed = true, true
				}
			}
		}
	}
	// Keep the physical unwrapping of a nullable named root attached to its
	// discovered occurrence. Emission must reuse this exact detached pair;
	// separately cloning it would lose both helper and failure identity.
	for _, key := range graph.order {
		node := graph.nodes[key]
		node.concreteSource = representation.ConcreteAttribute(node.source)
		node.concreteTarget = representation.ConcreteAttribute(node.target)
		copyKey := transformPairKey(node.concreteSource, node.concreteTarget, node.attrs)
		copyKey.sourceCopy, copyKey.targetCopy = node.concreteSource, node.concreteTarget
		graph.nodes[copyKey] = node
	}
	// Allocate only after the complete semantic/failure closure is known, in
	// discovery order rather than map iteration order.
	for _, key := range graph.order {
		node := graph.nodes[key]
		if transformUsesHelper(node.source, node.target) {
			attrs := *node.attrs
			attrs.failures = graph
			node.helper = transformHelperName(node.source, node.target, &attrs)
			graph.helpers[node.helper] = graph.helpers[node.helper] || node.fails
		}
	}
	return graph, nil
}

func transformPairKey(source, target *expr.AttributeExpr, ta *TransformAttrs) transformPair {
	ta = transformPresenceContext(source, target, ta)
	return transformPair{
		source:       source.Type,
		target:       target.Type,
		sourceRef:    ta.SourceCtx.Scope.Ref(source, ta.SourceCtx.Pkg(source)),
		targetRef:    ta.TargetCtx.Scope.Ref(target, ta.TargetCtx.Pkg(target)),
		sourceLayout: transformContextLayout(ta.SourceCtx),
		targetLayout: transformContextLayout(ta.TargetCtx),
	}
}

func transformContextLayout(ctx *AttributeContext) transformLayout {
	return transformLayout{ctx.Pointer, ctx.IgnoreRequired, ctx.UseDefault, ctx.JSONPresence, ctx.CollectionElementPresence}
}

func transformPresenceContext(source, target *expr.AttributeExpr, ta *TransformAttrs) *TransformAttrs {
	sourceContext := transformPhysicalContext(source, ta.SourceCtx)
	targetContext := transformPhysicalContext(target, ta.TargetCtx)
	if sourceContext == ta.SourceCtx && targetContext == ta.TargetCtx {
		return ta
	}
	nested := *ta
	nested.SourceCtx, nested.TargetCtx = sourceContext, targetContext
	return &nested
}

func transformPhysicalContext(attribute *expr.AttributeExpr, context *AttributeContext) *AttributeContext {
	context = context.ForAttribute(attribute)
	if attribute == nil || context == nil {
		return context
	}
	named, ok := attribute.Type.(expr.UserType)
	if !ok {
		return context
	}
	key := named.Hash()
	presence, found := context.JSONPresenceTypes[key]
	if !found {
		return context
	}
	nested := context.Dup()
	nested.JSONPresence, nested.CollectionElementPresence = presence, presence
	if pointer, found := context.PresencePointerTypes[key]; found {
		nested.Pointer = pointer
	}
	if defaults, found := context.PresenceUseDefaultTypes[key]; found {
		nested.UseDefault = defaults
	}
	return nested
}

func transformCallArguments(source, target *expr.AttributeExpr, value string, ta *TransformAttrs) []jen.Code {
	args := []jen.Code{Expr(value)}
	if ta.failures.fallible(source, target, ta) {
		args = append(args, jen.Id("transformErr"))
	}
	return args
}

func (graph *transformFailures) fallible(source, target *expr.AttributeExpr, ta *TransformAttrs) bool {
	if graph == nil {
		return false
	}
	node := graph.node(source, target, ta)
	if node != nil {
		return node.fails
	}
	return graph.helpers[transformHelperName(source, target, ta)]
}

func (graph *transformFailures) discover(source, target *expr.AttributeExpr, ta *TransformAttrs, path string) error {
	ta = transformPresenceContext(source, target, ta)
	key := transformPairKey(source, target, ta)
	if graph.nodes[key] != nil {
		return nil
	}
	node := &transformFailureNode{
		fails:  stringRepresentationCrossing(source, target, ta),
		source: source, target: target, attrs: ta, path: path,
	}
	graph.nodes[key] = node
	graph.order = append(graph.order, key)
	if matchingExplicitType(source, target) {
		return nil
	}
	child := func(src, tgt *expr.AttributeExpr, member string) error {
		node.children = append(node.children, transformPairKey(src, tgt, ta))
		return graph.discover(src, tgt, ta, path+"/"+member)
	}
	switch {
	case expr.IsArray(source.Type) && expr.IsArray(target.Type):
		return child(expr.AsArray(source.Type).ElemType, expr.AsArray(target.Type).ElemType, "[]")
	case expr.IsMap(source.Type) && expr.IsMap(target.Type):
		src, tgt := expr.AsMap(source.Type), expr.AsMap(target.Type)
		if err := child(src.KeyType, tgt.KeyType, "[key]"); err != nil {
			return err
		}
		return child(src.ElemType, tgt.ElemType, "[value]")
	case expr.IsUnion(source.Type) && expr.IsUnion(target.Type):
		for i, value := range expr.AsUnion(source.Type).Values {
			if matched, ok := matchingTransformUnionValue(value, expr.AsUnion(target.Type), i); ok {
				if err := child(value.Attribute, matched.Attribute, value.Name); err != nil {
					return err
				}
			}
		}
	case expr.IsObject(source.Type) && expr.IsObject(target.Type):
		return graph.discoverObject(source, target, ta, node, child)
	}
	return nil
}

// discoverObject records the fallible conversions of the members of an object
// pair, including the map of undeclared members of typed objects, whose keys
// may collide with declared names.
func (graph *transformFailures) discoverObject(source, target *expr.AttributeExpr, ta *TransformAttrs, node *transformFailureNode, child func(src, tgt *expr.AttributeExpr, member string) error) error {
	if srcCarrier, tgtCarrier := source.AdditionalMembersCarrier(), target.AdditionalMembersCarrier(); srcCarrier != nil && tgtCarrier != nil {
		node.fails = true
		if err := child(srcCarrier, tgtCarrier, "[additional]"); err != nil {
			return err
		}
	}
	var err error
	walkMatches(source, target, func(srcParent, tgtParent *expr.MappedAttributeExpr, sourceChild, targetChild *expr.AttributeExpr, name string) {
		if err != nil {
			return
		}
		application := ta.DefaultForField(srcParent, tgtParent, name)
		node.sourceDefaults = node.sourceDefaults || application.admittedSource
		if application.Apply {
			var value DefaultValueCode
			value, err = application.materialization(ta)
			if err != nil {
				return
			}
			node.fails = node.fails || value.ErrorAware
		}
		err = child(sourceChild, targetChild, name)
	})
	return err
}

func (graph *transformFailures) sourceDefaultNode(source, target *expr.AttributeExpr, ta *TransformAttrs) *transformFailureNode {
	if graph == nil {
		return nil
	}
	node := graph.node(source, target, ta)
	if node == nil || !node.sourceDefaults {
		return nil
	}
	return node
}

// concreteTransformPair returns the discovered physical clone without changing
// its selected semantic owner. With no plan it retains the ordinary copy contract.
func concreteTransformPair(source, target *expr.AttributeExpr, ta *TransformAttrs) (*expr.AttributeExpr, *expr.AttributeExpr) {
	if ta.failures != nil {
		if node := ta.failures.node(source, target, ta); node != nil {
			return node.concreteSource, node.concreteTarget
		}
	}
	return representation.ConcreteAttribute(source), representation.ConcreteAttribute(target)
}

// node recognizes an exact plan-owned concrete pair before canonical lookup.
// Copy keys occupy a separate key role, so they never overwrite another
// canonical occurrence even when its Go reference and structural type match.
func (graph *transformFailures) node(source, target *expr.AttributeExpr, ta *TransformAttrs) *transformFailureNode {
	key := transformPairKey(source, target, ta)
	key.sourceCopy, key.targetCopy = source, target
	if node := graph.nodes[key]; node != nil {
		return node
	}
	key.sourceCopy, key.targetCopy = nil, nil
	return graph.nodes[key]
}
