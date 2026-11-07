package codegen

import (
	"strings"

	"github.com/CaliLuke/loom/expr"
)

// UnionBranchContext returns a detached validation context for a value stored in
// a sum-union branch. The value has the physical type returned by Scope.Ref;
// it does not inherit the enclosing field's pointer or extracted-value mode.
// The context retains the field layout inside named object values. The boolean
// reports pointer storage, whose nil value must be rejected before validation.
// Nil slices and maps are concrete branch values and still run their predicates.
func (attCtx *AttributeContext) UnionBranchContext(attribute *expr.AttributeExpr) (*AttributeContext, bool) {
	// Resolve branch storage through the declaration's type reference. A
	// nil pointer is missing, while nil collections are concrete values
	// whose predicates must run. Nullable carriers own their presence.
	requiresValue := strings.HasPrefix(attCtx.Scope.Ref(attribute, attCtx.DefaultPkg), "*")
	unionCtx := attCtx.Dup()
	unionCtx.validationValue = !requiresValue
	_, named := attribute.Type.(expr.UserType)
	if !named {
		unionCtx.JSONPresence = false
		unionCtx.UseDefault = true
	}
	if expr.IsObject(attribute.Type) && !named {
		// HTTP sum types store inline object branches with native pointer
		// presence, even when the union field uses JSON presence.
		unionCtx.Pointer = false
	} else {
		unionCtx.Pointer = unionCtx.Pointer && expr.IsObject(attribute.Type)
	}
	return unionCtx, requiresValue
}

// validationTargetIsPointer distinguishes the current value from its field
// layout. Unwrapping an object yields a concrete struct even when Pointer must
// remain set for that struct's fields. Child traversal resets validationValue.
func (a *AttributeContext) validationTargetIsPointer(required, hasDefault bool) bool {
	return !a.validationValue && (a.Pointer || (!required && (!hasDefault || !a.UseDefault)))
}
