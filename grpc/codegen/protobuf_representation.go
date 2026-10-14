package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/representation"
)

// protobufStringPlan delegates authored String representation changes to the
// shared conversion owner. The protobuf scope owns the actual wire carrier,
// including an explicit protobuf override; service Go overrides do not define it.
func protobufStringPlan(source, target *expr.AttributeExpr, value string, ta *transformAttrs) (codegen.TransformPlan, bool) {
	if !representation.CustomString(source) && !representation.CustomString(target) {
		return codegen.TransformPlan{}, false
	}
	if ta.proto {
		target = protobufStringAttribute(target, ta.TargetCtx)
	} else {
		source = protobufStringAttribute(source, ta.SourceCtx)
	}
	plan, err := codegen.GoTransform(source, target, value, "converted", ta.SourceCtx, ta.TargetCtx, "", true)
	if err != nil {
		panic(fmt.Errorf("build protobuf String representation conversion: %w", err))
	}
	return plan, true
}

func protobufStringAttribute(attribute *expr.AttributeExpr, ctx *codegen.AttributeContext) *expr.AttributeExpr {
	physical := &expr.AttributeExpr{Type: expr.String}
	if ref := ctx.Scope.Ref(attribute, ctx.Pkg(attribute)); ref != "string" {
		physical.Meta = expr.MetaExpr{"struct:field:type": {ref}}
	}
	return physical
}

// protobufRepresentationCanFail walks selected protobuf/service pairs. Each
// query owns its visited set: revisiting a cycle cannot cache a partial negative
// result for another helper. Scalar and default fallibility comes from the same
// plans used to emit those operations.
func protobufRepresentationCanFail(source, target *expr.AttributeExpr, sourceCtx, targetCtx *codegen.AttributeContext) bool {
	_, proto := targetCtx.Scope.(*protoBufScope)
	ta := &transformAttrs{TransformAttrs: &codegen.TransformAttrs{SourceCtx: sourceCtx, TargetCtx: targetCtx}, proto: proto}
	type pair struct {
		source, target *expr.AttributeExpr
	}
	seen := make(map[pair]bool)
	var visit func(*expr.AttributeExpr, *expr.AttributeExpr) bool
	visit = func(src, tgt *expr.AttributeExpr) bool {
		var err error
		src, tgt, err = normalizedTransformAttrs(src, tgt, ta)
		if err != nil {
			return false
		}
		// Canonical occurrences preserve metadata and selected defaults without
		// reserving names for unwrapped values that are not protobuf messages.
		key := pair{src, tgt}
		if seen[key] {
			return false
		}
		seen[key] = true
		if plan, ok := protobufStringPlan(src, tgt, "value", ta); ok {
			return plan.ErrorAware
		}
		switch {
		case expr.IsArray(src.Type):
			return visit(expr.AsArray(src.Type).ElemType, expr.AsArray(tgt.Type).ElemType)
		case expr.IsMap(src.Type):
			sm, tm := expr.AsMap(src.Type), expr.AsMap(tgt.Type)
			return visit(sm.KeyType, tm.KeyType) || visit(sm.ElemType, tm.ElemType)
		case expr.IsUnion(src.Type):
			su, tu := expr.AsUnion(src.Type), expr.AsUnion(tgt.Type)
			for i, value := range su.Values {
				if i < len(tu.Values) && visit(value.Attribute, tu.Values[i].Attribute) {
					return true
				}
			}
		case expr.IsObject(src.Type):
			found := false
			walkMatches(src, tgt, func(sm, tm *expr.MappedAttributeExpr, _, childTarget *expr.AttributeExpr, name string) {
				if found {
					return
				}
				application := ta.DefaultForField(sm, tm, name)
				if application.Apply {
					plan, err := targetCtx.DefaultValue(childTarget, application.Value)
					if err != nil {
						panic(fmt.Errorf("build protobuf field default: %w", err))
					}
					found = plan.ErrorAware
				}
				_, canonicalSource := src.FindAttribute(name)
				_, canonicalTarget := tgt.FindAttribute(name)
				found = found || visit(canonicalSource, canonicalTarget)
			})
			return found
		}
		return false
	}
	return visit(source, target)
}
