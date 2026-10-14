package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// transformElement returns the code that converts an array element or a map
// key or value. It calls the transform helper of an element user type that
// reaches itself through inlined collections, because inlining such a type
// never terminates. transformAttribute inlines every other element.
func transformElement(physicalSource, source, target *expr.AttributeExpr, sourceVar, targetVar string, newVar bool, ta *transformAttrs) (string, error) {
	var code string
	var err error
	if isInlineRecursive(source) {
		code = transformScalarAssignment(source, target, sourceVar, targetVar, newVar, ta)
	} else {
		code, err = transformAttribute(source, target, sourceVar, targetVar, newVar, ta)
	}
	if err != nil {
		return "", err
	}
	if elementNeedsAdmission(physicalSource, ta.SourceCtx) {
		code = "if " + sourceVar + " == nil {\n\t*transformErr = errors.New(\"gRPC conversion: nil message collection element\")\n\tcontinue\n}\n" + code
	}
	return code, nil
}

// transformElementHelpers returns the transform helpers that transformElement
// requires for an array element or a map key or value.
func transformElementHelpers(source, target *expr.AttributeExpr, ta *transformAttrs, seen map[string]*codegen.TransformFunctionData) ([]*codegen.TransformFunctionData, error) {
	if !isInlineRecursive(source) {
		return transformAttributeHelpers(source, target, ta, seen)
	}
	source, target, err := normalizedTransformAttrs(source, target, ta)
	if err != nil {
		return nil, err
	}
	return collectObjectHelpers(source, target, false, ta, seen)
}

// isInlineRecursive reports whether att is an object user type that reaches
// itself through array elements and map keys or values alone, including the
// elements of named arrays and maps and of the messages that wrap them.
// Transform code inlines those collection elements, while it converts object
// fields of a user type and union branches with helper calls, so only this
// kind of cycle requires a helper call.
func isInlineRecursive(att *expr.AttributeExpr) bool {
	ut, ok := att.Type.(expr.UserType)
	if !ok || !expr.IsObject(ut) || isCollectionMessage(ut) {
		// The message that wraps a named array or map is converted inline
		// like the collection of the service type, whose recursive elements
		// call helpers.
		return false
	}
	return reachesInline(ut.Attribute(), ut.ID(), false, make(map[string]struct{}))
}

// reachesInline reports whether att reaches the user type with the given ID
// through inlined transform code. element reports whether att is an array
// element or a map key or value. The message generated for an anonymous
// object and a named array or map, or the message that wraps it, are always
// inlined.
func reachesInline(att *expr.AttributeExpr, id string, element bool, seen map[string]struct{}) bool {
	switch dt := att.Type.(type) {
	case expr.UserType:
		named := expr.IsArray(dt) || expr.IsMap(dt) || isCollectionMessage(dt)
		if !named && (!(element || isAnonymousMessage(dt)) || !expr.IsObject(dt)) {
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
