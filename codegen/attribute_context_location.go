package codegen

import "github.com/CaliLuke/loom/expr"

// ForAttribute derives the physical declaration context for attribute. A named
// relocated parent supplies the destination inherited by its bare nested types.
// The new context preserves representation policy and never mutates its caller;
// source and target conversions must derive their contexts independently.
// Attributes without a location retain the enclosing physical context.
func (a *AttributeContext) ForAttribute(attribute *expr.AttributeExpr) *AttributeContext {
	if a == nil || attribute == nil || a.Scope == nil {
		return a
	}
	location := attributeTypeLocation(attribute)
	if location == nil {
		return a
	}
	scope := a.Scope.Scope()
	if scope.typeLocation != nil && scope.typeLocation.RelImportPath == location.RelImportPath {
		return a
	}
	nested := a.Dup()
	nested.Scope = a.Scope.WithScope(scope.WithTypeNameLocation(location))
	if pkg := scope.PackageName(location); pkg != a.DefaultPkg {
		nested.DefaultPkg = pkg
		nested.SamePackageConversion = false
	}
	return nested
}

// WithScope preserves native attribute representation with a detached name scope.
func (a *AttributeScope) WithScope(scope *NameScope) Attributor {
	return NewAttributeScope(scope)
}
