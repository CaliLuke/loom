package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// httpContext returns a context for attributes of types used to marshal and
// unmarshal HTTP requests and responses.
//
// pkg is the package name where the body type exists
//
// scope is the named scope
//
// request if true indicates that the type is a request type, else response
// type
//
// svr if true indicates that the type is a server type, else client type
func httpContext(scope *codegen.NameScope, request, svr bool) *codegen.AttributeContext {
	marshal := !request && svr || request && !svr
	return codegen.NewAttributeContext(!marshal, false, marshal, "", scope)
}

// serviceContext returns an attribute context for service types.
func serviceContext(pkg string, scope *codegen.NameScope) *codegen.AttributeContext {
	return codegen.NewAttributeContext(false, false, true, pkg, scope)
}

// viewContext returns an attribute context for projected types.
func viewContext(pkg string, scope *codegen.NameScope) *codegen.AttributeContext {
	return codegen.NewAttributeContext(true, false, true, pkg, scope)
}

// pkgWithDefault returns the package name of the given location if not nil, def otherwise.
func pkgWithDefault(loc *codegen.Location, def string) string {
	if loc == nil {
		return def
	}
	return loc.PackageName()
}

// unmarshal initializes a data structure defined by target type from a data
// structure defined by source type. The attributes in the source data
// structure are pointers and the attributes in the target data structure that
// have default values are non-pointers. Fields in target type are initialized
// with their default values (if any).
//
// source, target are the attributes used in the transformation
//
// sourceVar, targetVar are the variable names for source and target used in
// the transformation code
//
// sourceCtx, targetCtx are the source and target attribute contexts
func unmarshal(source, target *expr.AttributeExpr, sourceVar string, sourceCtx, targetCtx *codegen.AttributeContext) (string, []*codegen.TransformFunctionData, error) {
	return codegen.GoTransform(source, target, sourceVar, "v", sourceCtx, targetCtx, "unmarshal", true)
}

// marshal initializes a data structure defined by target type from a data
// structure defined by source type. The fields in the source and target
// data structure use non-pointers for attributes with default values.
//
// source, target are the attributes used in the transformation
//
// sourceVar, targetVar are the variable names for source and target used in
// the transformation code
//
// sourceCtx, targetCtx are the source and target attribute contexts
func marshal(source, target *expr.AttributeExpr, sourceVar, targetVar string, sourceCtx, targetCtx *codegen.AttributeContext) (string, []*codegen.TransformFunctionData, error) {
	return codegen.GoTransform(source, target, sourceVar, targetVar, sourceCtx, targetCtx, "marshal", true)
}

// needConversion returns true if the type needs to be converted from a string.
func needConversion(dt expr.DataType) bool {
	if dt == expr.Empty {
		return false
	}
	switch actual := dt.(type) {
	case expr.Primitive:
		if actual.Kind() == expr.StringKind ||
			actual.Kind() == expr.AnyKind ||
			actual.Kind() == expr.BytesKind {
			return false
		}
		return true
	case *expr.Array:
		return needConversion(actual.ElemType.Type)
	case *expr.Map:
		return needConversion(actual.KeyType.Type) ||
			needConversion(actual.ElemType.Type)
	default:
		return true
	}
}

// addMarshalTags adds JSON, XML and Form tags to all inline object attributes recursively.
func addMarshalTags(att *expr.AttributeExpr, seen map[string]struct{}) {
	if ut, ok := att.Type.(expr.UserType); ok {
		if _, ok := seen[ut.Hash()]; ok {
			return // avoid infinite recursions
		}
		seen[ut.Hash()] = struct{}{}
		if expr.IsObject(ut.Attribute().Type) {
			for _, att := range *(expr.AsObject(att.Type)) {
				addMarshalTags(att.Attribute, seen)
			}
		}
		return
	}
	if expr.IsArray(att.Type) {
		addMarshalTags(expr.AsArray(att.Type).ElemType, seen)
		return
	}
	if expr.IsMap(att.Type) {
		addMarshalTags(expr.AsMap(att.Type).KeyType, seen)
		addMarshalTags(expr.AsMap(att.Type).ElemType, seen)
		return
	}
	if !expr.IsObject(att.Type) {
		return
	}
	// inline object
	for _, natt := range *(expr.AsObject(att.Type)) {
		if natt.Attribute.Meta == nil {
			natt.Attribute.Meta = expr.MetaExpr{}
		}
		ns := []string{natt.Name}
		natt.Attribute.Meta["struct:tag:form"] = ns
		natt.Attribute.Meta["struct:tag:json"] = ns
		natt.Attribute.Meta["struct:tag:xml"] = ns
	}
}

func containsUnionType(dt expr.DataType) bool {
	return containsUnionTypeRecursive(dt, make(map[string]struct{}))
}

func containsUnionTypeRecursive(dt expr.DataType, seen map[string]struct{}) bool {
	switch actual := dt.(type) {
	case nil:
		return false
	case *expr.Union:
		return true
	case expr.UserType:
		if _, ok := seen[actual.ID()]; ok {
			return false
		}
		seen[actual.ID()] = struct{}{}
		return containsUnionTypeRecursive(actual.Attribute().Type, seen)
	case *expr.Object:
		for _, nat := range *actual {
			if containsUnionTypeRecursive(nat.Attribute.Type, seen) {
				return true
			}
		}
	case *expr.Array:
		return containsUnionTypeRecursive(actual.ElemType.Type, seen)
	case *expr.Map:
		return containsUnionTypeRecursive(actual.KeyType.Type, seen) || containsUnionTypeRecursive(actual.ElemType.Type, seen)
	}
	return false
}

// needInit returns true if and only if the given type is or makes use of user
// types.
func needInit(dt expr.DataType) bool {
	if dt == expr.Empty {
		return false
	}
	switch actual := dt.(type) {
	case expr.Primitive:
		return false
	case *expr.Array:
		return needInit(actual.ElemType.Type)
	case *expr.Map:
		return needInit(actual.KeyType.Type) ||
			needInit(actual.ElemType.Type)
	case *expr.Object:
		for _, nat := range *actual {
			if needInit(nat.Attribute.Type) {
				return true
			}
		}
		return false
	case expr.UserType:
		return true
	default:
		panic(fmt.Sprintf("unknown data type %T", actual)) // bug
	}
}
