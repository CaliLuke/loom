package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// conversionCanFail reports whether this source/target conversion needs an
// error channel, including errors from nested conversions. Callers and shared
// helpers query their own conversion pair and physical source context.
func conversionCanFail(source, target *expr.AttributeExpr, sourceCtx *codegen.AttributeContext) bool {
	if containsAny(source) || containsAny(target) {
		return true
	}
	found := false
	if err := codegen.Walk(source, func(current *expr.AttributeExpr) error {
		if array := expr.AsArray(current.Type); array != nil {
			found = found || elementNeedsAdmission(array.ElemType, sourceCtx)
		} else if mapping := expr.AsMap(current.Type); mapping != nil {
			found = found || elementNeedsAdmission(mapping.ElemType, sourceCtx)
		}
		return nil
	}); err != nil {
		panic(err)
	}
	return found
}

// elementNeedsAdmission classifies the physical source before protobuf wrappers
// are unwrapped. Object elements are pointers; protobuf unions are message
// pointers too, while service union elements are values. Native collection
// values and Any retain their existing nil semantics. Optional fields do not
// call this occurrence policy.
func elementNeedsAdmission(source *expr.AttributeExpr, ctx *codegen.AttributeContext) bool {
	_, protobuf := ctx.Scope.(*protoBufScope)
	if expr.IsObject(source.Type) {
		return protobuf || !codegen.IsExplicitPresenceType(source)
	}
	return protobuf && expr.IsUnion(source.Type)
}

func containsAny(attribute *expr.AttributeExpr) bool {
	found := false
	if err := codegen.Walk(attribute, func(current *expr.AttributeExpr) error {
		if current.Type.Kind() == expr.AnyKind {
			found = true
		}
		return nil
	}); err != nil {
		panic(err)
	}
	return found
}
