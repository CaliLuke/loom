package codegen

import (
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/representation"
)

// CollectionElementRef returns the physical Go reference stored in a collection
// slot. Objects use pointers; unions and explicit presence types use values.
// mapKey selects key storage, including an unconstrained key's any carrier.
// Named declarations retain their own nested field and collection layouts.
// The returned reference excludes the containing slot's Optional wrapper; the
// caller owns that wrapper through CollectionElementPresence.
func (context *AttributeContext) CollectionElementRef(attribute *expr.AttributeExpr, mapKey bool) string {
	// Named declarations own their nested layout. Inline collections need the
	// same recursive element carriers as those declarations, but this function
	// does not wrap the current element itself: callers own that container slot.
	context = transformPhysicalContext(attribute, context)
	metaType, _ := GetMetaType(attribute)
	if _, named := attribute.Type.(expr.UserType); !named && metaType == "" {
		concrete := representation.ConcreteAttribute(attribute)
		var ref string
		switch typ := concrete.Type.(type) {
		case *expr.Array:
			element := context.CollectionElementRef(typ.ElemType, false)
			ref = "[]" + collectionCarrierRef(element, context.CollectionElementPresence && !expr.ArrayElementsAllowNull(typ))
		case *expr.Map:
			key := context.CollectionElementRef(typ.KeyType, true)
			element := context.CollectionElementRef(typ.ElemType, false)
			ref = "map[" + key + "]" + collectionCarrierRef(element, context.CollectionElementPresence && !expr.MapValuesAllowNull(typ))
		}
		if ref != "" {
			if expr.IsNullable(attribute) {
				return "loom.Nullable[" + ref + "]"
			}
			return ref
		}
	}
	pkg := ""
	targetPkg := context.Pkg(attribute)
	if context.SamePackageConversion {
		pkg = context.DefaultPkg
		targetPkg = ""
	}
	if pkg == "" && containsInlineObject(attribute) {
		return attributorCollectionElemTypeRef(attribute, context, targetPkg)
	}
	scope := context.Scope.Scope()
	if mapKey {
		return scope.mapKeyTypeDef(
			attribute,
			context.Pointer,
			context.UseDefault,
			pkg,
			targetPkg,
		)
	}
	return scope.collectionElemTypeDef(
		attribute,
		context.Pointer,
		context.UseDefault,
		pkg,
		targetPkg,
	)
}

// attributorCollectionElemTypeRef renders the type of a collection element
// that holds an inline object at any depth of arrays, maps and nullability.
// The inline struct comes from the context Attributor so that it matches the
// declaring type, which a transport may render differently from the shared
// scope. It returns the same rendering as NameScope.collectionElemTypeDef when
// the Attributor is AttributeScope.
func attributorCollectionElemTypeRef(attribute *expr.AttributeExpr, context *AttributeContext, targetPkg string) string {
	scope := context.Scope.Scope()
	fallback := func() string {
		return scope.collectionElemTypeDef(attribute, context.Pointer, context.UseDefault, "", targetPkg)
	}
	if metaType, _ := GetMetaType(attribute); metaType != "" {
		return fallback()
	}
	var value string
	switch actual := attribute.Type.(type) {
	case *expr.Object:
		name := context.Scope.Name(attribute, targetPkg, context)
		if IsExplicitPresenceType(attribute) {
			// A presence type such as loom.Nullable[T] holds the struct value.
			return name
		}
		return "*" + name
	case *expr.Array:
		value = "[]" + attributorCollectionElemTypeRef(actual.ElemType, context, targetPkg)
	case *expr.Map:
		key := scope.mapKeyTypeDef(actual.KeyType, context.Pointer, context.UseDefault, "", targetPkg)
		value = "map[" + key + "]" + attributorCollectionElemTypeRef(actual.ElemType, context, targetPkg)
	default:
		return fallback()
	}
	switch {
	case expr.IsNullable(attribute):
		return "loom.Nullable[" + value + "]"
	case IsExplicitPresenceType(attribute):
		return fallback()
	default:
		return value
	}
}

// containsInlineObject reports whether att is an inline object or a
// collection whose elements hold one at any depth, without looking inside user
// types.
func containsInlineObject(att *expr.AttributeExpr) bool {
	switch actual := att.Type.(type) {
	case *expr.Object:
		return true
	case *expr.Array:
		return containsInlineObject(actual.ElemType)
	case *expr.Map:
		return containsInlineObject(actual.ElemType)
	default:
		return false
	}
}
