package codegen

import (
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
		sourceRef, targetRef       string
		sourceLayout, targetLayout transformLayout
	}

	transformLayout struct {
		pointer, ignoreRequired, useDefault, jsonPresence, elementPresence bool
	}

	transformFailureNode struct {
		fails    bool
		children []transformPair
		helper   string
	}

	transformFailures struct {
		nodes   map[transformPair]*transformFailureNode
		helpers map[string]bool
	}
)

// transformFailurePlan explores selected conversion pairs, then propagates local
// failures to a fixed point. A visited node stops graph discovery only: it never
// asserts that a recursive conversion is infallible.
func transformFailurePlan(source, target *expr.AttributeExpr, ta *TransformAttrs) (*transformFailures, error) {
	graph := &transformFailures{nodes: make(map[transformPair]*transformFailureNode), helpers: make(map[string]bool)}
	if err := graph.discover(source, target, ta); err != nil {
		return nil, err
	}
	for changed := true; changed; {
		changed = false
		for _, node := range graph.nodes {
			if node.fails {
				continue
			}
			for _, child := range node.children {
				if graph.nodes[child].fails {
					node.fails, changed = true, true
					break
				}
			}
		}
	}
	for _, node := range graph.nodes {
		if node.helper != "" {
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
	node := graph.nodes[transformPairKey(source, target, ta)]
	if node != nil {
		return node.fails
	}
	return graph.helpers[transformHelperName(source, target, ta)]
}

func (graph *transformFailures) discover(source, target *expr.AttributeExpr, ta *TransformAttrs) error {
	ta = transformPresenceContext(source, target, ta)
	key := transformPairKey(source, target, ta)
	if graph.nodes[key] != nil {
		return nil
	}
	node := &transformFailureNode{fails: stringRepresentationCrossing(source, target, ta)}
	if transformUsesHelper(source, target) {
		node.helper = transformHelperName(source, target, ta)
	}
	graph.nodes[key] = node
	if matchingExplicitType(source, target) {
		return nil
	}
	child := func(src, tgt *expr.AttributeExpr) error {
		node.children = append(node.children, transformPairKey(src, tgt, ta))
		return graph.discover(src, tgt, ta)
	}
	switch {
	case expr.IsArray(source.Type) && expr.IsArray(target.Type):
		return child(expr.AsArray(source.Type).ElemType, expr.AsArray(target.Type).ElemType)
	case expr.IsMap(source.Type) && expr.IsMap(target.Type):
		src, tgt := expr.AsMap(source.Type), expr.AsMap(target.Type)
		if err := child(src.KeyType, tgt.KeyType); err != nil {
			return err
		}
		return child(src.ElemType, tgt.ElemType)
	case expr.IsUnion(source.Type) && expr.IsUnion(target.Type):
		for i, value := range expr.AsUnion(source.Type).Values {
			if matched, ok := matchingTransformUnionValue(value, expr.AsUnion(target.Type), i); ok {
				if err := child(value.Attribute, matched.Attribute); err != nil {
					return err
				}
			}
		}
	case expr.IsObject(source.Type) && expr.IsObject(target.Type):
		var err error
		walkMatches(source, target, func(srcParent, tgtParent *expr.MappedAttributeExpr, src, tgt *expr.AttributeExpr, name string) {
			if err != nil {
				return
			}
			application := ta.DefaultForField(srcParent, tgtParent, name)
			if application.Apply {
				var value DefaultValueCode
				value, err = ta.TargetCtx.DefaultValue(tgt, application.Value)
				if err != nil {
					return
				}
				node.fails = node.fails || value.ErrorAware
			}
			_, canonicalSource := source.FindAttribute(name)
			_, canonicalTarget := target.FindAttribute(name)
			err = child(canonicalSource, canonicalTarget)
		})
		return err
	}
	return nil
}
