package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// transformElement returns the code that converts an array element or a map
// key or value. It calls the transform helper of an element user type that
// reaches itself through inlined collections, because inlining such a type
// never terminates. transformAttribute inlines every other element.
func transformElement(source, target *expr.AttributeExpr, sourceVar, targetVar string, newVar bool, ta *transformAttrs) (string, error) {
	if isInlineRecursive(source) {
		return transformScalarAssignment(source, target, sourceVar, targetVar, newVar, ta), nil
	}
	return transformAttribute(source, target, sourceVar, targetVar, newVar, ta)
}

// transformElementHelpers returns the transform helpers that transformElement
// requires for an array element or a map key or value.
func transformElementHelpers(source, target *expr.AttributeExpr, ta *transformAttrs, seen map[string]*codegen.TransformFunctionData) ([]*codegen.TransformFunctionData, error) {
	if !isInlineRecursive(source) {
		return transformAttributeHelpers(source, target, ta, seen)
	}
	source, target, err := compatibleTransformAttributes(source, target, ta)
	if err != nil {
		return nil, err
	}
	return collectObjectHelpers(source, target, false, ta, seen)
}

// isInlineRecursive reports whether att is an object user type that reaches
// itself through array elements and map keys or values alone. Transform code
// inlines those collection elements, while it converts object fields of a user
// type with helper calls, so only this kind of cycle requires a helper call.
func isInlineRecursive(att *expr.AttributeExpr) bool {
	ut, ok := att.Type.(expr.UserType)
	if !ok || !expr.IsObject(ut) {
		return false
	}
	return reachesInline(ut.Attribute(), ut.ID(), false, make(map[string]struct{}))
}

// reachesInline reports whether att reaches the user type with the given ID
// through inlined transform code. element reports whether att is an array
// element or a map key or value. The message generated for an anonymous
// object is always inlined.
func reachesInline(att *expr.AttributeExpr, id string, element bool, seen map[string]struct{}) bool {
	switch dt := att.Type.(type) {
	case expr.UserType:
		if !(element || isAnonymousMessage(dt)) || !expr.IsObject(dt) {
			return false
		}
		if dt.ID() == id {
			return true
		}
		if _, ok := seen[dt.ID()]; ok {
			return false
		}
		seen[dt.ID()] = struct{}{}
		return reachesInline(dt.Attribute(), id, false, seen)
	case *expr.Object:
		for _, nat := range *dt {
			if reachesInline(nat.Attribute, id, false, seen) {
				return true
			}
		}
	case *expr.Array:
		return reachesInline(dt.ElemType, id, true, seen)
	case *expr.Map:
		return reachesInline(dt.KeyType, id, true, seen) || reachesInline(dt.ElemType, id, true, seen)
	}
	return false
}
