package representation

import "github.com/CaliLuke/loom/expr"

// ConcreteAttribute returns the representation of the concrete value inside a
// root presence carrier. It detaches the root and each nullable named alias in
// that root chain, preserving nominal identity and all child declarations,
// recursive edges, layout, constraints, and authored default ownership. It does
// not remove presence from structural children or mutate the supplied graph.
func ConcreteAttribute(att *expr.AttributeExpr) *expr.AttributeExpr {
	underlying := *att
	underlying.Nullable = false
	// Unwrapping consumes the whole inherited root carrier. Structural children
	// retain their original declarations, including recursive nullable members.
	copies := make(map[expr.UserType]expr.UserType)
	for current := &underlying; ; {
		userType, ok := current.Type.(expr.UserType)
		if !ok || !expr.IsNullable(userType.Attribute()) {
			break
		}
		if copied, ok := copies[userType]; ok {
			current.Type = copied
			break
		}
		concreteAttribute := *userType.Attribute()
		concreteAttribute.Nullable = false
		concreteType := userType.Dup(&concreteAttribute)
		copies[userType] = concreteType
		current.Type = concreteType
		current = &concreteAttribute
	}
	return &underlying
}
