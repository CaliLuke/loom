package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/expr"
)

type validationEligibilityKey struct {
	attribute                                         *expr.AttributeExpr
	pointer, ignoreRequired, jsonPresence, useDefault bool
}

// hasValidations reports whether a named declaration has predicates in its
// physical layout. Call-site value storage must not redefine named field layout.
func hasValidations(ctx *AttributeContext, user expr.UserType) bool {
	return hasAttributeValidations(&expr.AttributeExpr{Type: user}, ctx, make(map[validationEligibilityKey]bool))
}

func hasAttributeValidations(attribute *expr.AttributeExpr, ctx *AttributeContext, seen map[validationEligibilityKey]bool) bool {
	if _, named := attribute.Type.(expr.UserType); named {
		ctx = presenceValueContext(ctx, attribute)
	}
	key := validationEligibilityKey{attribute, ctx.Pointer, ctx.IgnoreRequired, ctx.JSONPresence, ctx.UseDefault}
	if seen[key] {
		return false
	}
	seen[key] = true
	if attribute.Validation != nil && (!attribute.Validation.HasRequiredOnly() || len(generatedRequiredValidation(attribute, ctx)) > 0) {
		return true
	}
	switch actual := attribute.Type.(type) {
	case expr.Primitive:
		return false
	case expr.UserType:
		return hasAttributeValidations(actual.Attribute(), ctx, seen)
	case *expr.Object:
		for _, field := range *actual {
			if hasAttributeValidations(field.Attribute, ctx, seen) {
				return true
			}
		}
	case *expr.Union:
		_, sum := ctx.Scope.(sumTypeUnionScope)
		for _, field := range actual.Values {
			branch := ctx
			if sum {
				var requiresValue bool
				branch, requiresValue = unionValidationContext(ctx, field.Attribute)
				if requiresValue {
					return true
				}
			}
			if hasAttributeValidations(field.Attribute, branch, seen) {
				return true
			}
		}
	case *expr.Array:
		return hasAttributeValidations(actual.ElemType, ctx, seen)
	case *expr.Map:
		return hasAttributeValidations(actual.KeyType, ctx, seen) || hasAttributeValidations(actual.ElemType, ctx, seen)
	default:
		panic(NewError(nil, attribute, fmt.Errorf("unknown attribute type %T", actual)))
	}
	return false
}
