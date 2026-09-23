package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// httpAttributeScope is the attribute scope of HTTP body types. Transforms
// that build an HTTP body declare its inline object fields again as composite
// literals, so it renders inline object types with the same definition as the
// body type declaration. Other types use the shared attribute scope.
type httpAttributeScope struct {
	*codegen.AttributeScope
}

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
	ctx := codegen.NewAttributeContext(!marshal, false, marshal, "", scope)
	ctx.Scope = &httpAttributeScope{AttributeScope: codegen.NewAttributeScope(scope)}
	return ctx
}

func responseHTTPContext(scope *codegen.NameScope) *codegen.AttributeContext {
	ctx := httpContext(scope, false, false)
	ctx.JSONPresence = true
	return ctx
}

// serviceContext returns an attribute context for service types.
func serviceContext(pkg string, scope *codegen.NameScope) *codegen.AttributeContext {
	return codegen.NewAttributeContext(false, false, true, pkg, scope)
}

// viewContext returns an attribute context for projected types.
func viewContext(pkg string, scope *codegen.NameScope) *codegen.AttributeContext {
	return codegen.NewAttributeContext(true, false, true, pkg, scope)
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

func serviceFieldTransformAttribute(parent *expr.AttributeExpr, name string, attribute *expr.AttributeExpr) *expr.AttributeExpr {
	if parent == nil || name == "" || attribute == nil || expr.IsNullable(attribute) || expr.IsAny(attribute.Type) || !expr.AllowsNull(attribute) {
		return attribute
	}
	clone := expr.DupAtt(attribute)
	clone.Nullable = true
	return clone
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

// addMarshalTags adds JSON, XML and Form tags to the fields of the inline
// objects that belong to the body attribute att: its own fields and the
// elements of its arrays and maps. It does not tag inside nested user types.
// A nested user type is declared once per side from whichever copy of the
// body reaches the type registry first, so tagging only some copies would
// make its declaration and the transforms that build it disagree.
func addMarshalTags(att *expr.AttributeExpr) {
	if ut, ok := att.Type.(expr.UserType); ok {
		if expr.IsObject(ut.Attribute().Type) {
			for _, att := range *(expr.AsObject(att.Type)) {
				addInlineMarshalTags(att.Attribute)
			}
		}
		return
	}
	addInlineMarshalTags(att)
}

// addInlineMarshalTags tags the fields of att when it is an inline object, or
// of the inline objects held by att when it is an array or a map.
func addInlineMarshalTags(att *expr.AttributeExpr) {
	if _, ok := att.Type.(expr.UserType); ok {
		return
	}
	if expr.IsArray(att.Type) {
		addInlineMarshalTags(expr.AsArray(att.Type).ElemType)
		return
	}
	if expr.IsMap(att.Type) {
		addInlineMarshalTags(expr.AsMap(att.Type).KeyType)
		addInlineMarshalTags(expr.AsMap(att.Type).ElemType)
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

// needInit returns true if and only if the given attribute type is or makes use
// of user types.
func needInit(att *expr.AttributeExpr) bool {
	if att == nil {
		return false
	}
	return needInitType(att, att.Type)
}

func needInitType(root *expr.AttributeExpr, dt expr.DataType) bool {
	if dt == expr.Empty {
		return false
	}
	switch actual := dt.(type) {
	case expr.Primitive:
		return false
	case *expr.Array:
		return needInitType(root, actual.ElemType.Type)
	case *expr.Map:
		return needInitType(root, actual.KeyType.Type) ||
			needInitType(root, actual.ElemType.Type)
	case *expr.Union:
		return true
	case *expr.Object:
		for _, nat := range *actual {
			if needInitType(root, nat.Attribute.Type) {
				return true
			}
		}
		return false
	case expr.UserType:
		return true
	default:
		panic(codegen.NewError(nil, root, fmt.Errorf("unknown transform initialization data type %T", actual)))
	}
}

// Name returns the HTTP body type definition of an inline object attribute and
// the shared attribute type name otherwise.
func (s *httpAttributeScope) Name(att *expr.AttributeExpr, pkg string, ptr, useDefault bool) string {
	if _, ok := att.Type.(expr.UserType); !ok && expr.IsObject(att.Type) {
		return goTypeDef(s.Scope(), att, ptr, useDefault, false)
	}
	return s.AttributeScope.Name(att, pkg, ptr, useDefault)
}
