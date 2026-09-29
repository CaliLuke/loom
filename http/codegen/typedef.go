package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
)

// goTypeDef returns the Go code that defines the struct corresponding to ma.
// It differs from the function defined in the codegen package in the following
// ways:
//
//   - It defines marshaler tags on each fields using the HTTP element names.
//
//   - It produced fields with pointers even if the corresponding attribute is
//     required when ptr is true so that the generated code may validate
//     explicitly.
//
// useDefault directs whether fields holding primitive types with default values
// should hold pointers when ptr is false. If it is true then the fields are
// values even when not required (to account for the fact that they have a
// default value so cannot be nil) otherwise the fields are values only when
// required.
func goTypeDef(scope *codegen.NameScope, att *expr.AttributeExpr, ptr, useDefault, jsonPresence bool) string {
	if t, _ := codegen.GetMetaType(att); codegen.IsExplicitPresenceType(att) && t != "" {
		return t
	}
	if expr.IsNullable(att) {
		return "loom.Nullable[" + goValueTypeDef(scope, att, ptr, useDefault, jsonPresence) + "]"
	}
	return goValueTypeDef(scope, att, ptr, useDefault, jsonPresence)
}

func goValueTypeDef(scope *codegen.NameScope, att *expr.AttributeExpr, ptr, useDefault, jsonPresence bool) string {
	switch actual := att.Type.(type) {
	case expr.Primitive:
		return goPrimitiveTypeDef(att, actual)
	case *expr.Array:
		return goArrayTypeDef(scope, actual, ptr, useDefault, jsonPresence)
	case *expr.Map:
		return goMapTypeDef(scope, actual, ptr, useDefault, jsonPresence)
	case *expr.Object:
		return goObjectTypeDef(scope, att, actual, ptr, useDefault, jsonPresence)
	case expr.UserType, *expr.Union:
		return scope.GoValueTypeName(att)
	default:
		panic(codegen.NewError(nil, att, fmt.Errorf("unknown HTTP type definition %T", actual)))
	}
}

func goPrimitiveTypeDef(att *expr.AttributeExpr, actual expr.Primitive) string {
	if t, _ := codegen.GetMetaType(att); t != "" {
		return t
	}
	if actual.Kind() == expr.AnyKind {
		return "loom.JSONValue"
	}
	return codegen.GoNativeTypeName(actual)
}

func goArrayTypeDef(scope *codegen.NameScope, actual *expr.Array, ptr, useDefault, jsonPresence bool) string {
	return "[]" + goArrayElemTypeDef(scope, actual, ptr, useDefault, jsonPresence)
}

func goMapTypeDef(scope *codegen.NameScope, actual *expr.Map, ptr, useDefault, jsonPresence bool) string {
	keyDef := goMapKeyTypeDef(scope, actual.KeyType, ptr, useDefault, jsonPresence)
	elemDef := goCollectionElemTypeDef(scope, actual.ElemType, ptr, useDefault, jsonPresence)
	if jsonPresence && !expr.MapValuesAllowNull(actual) {
		elemDef = "loom.Nullable[" + elemDef + "]"
	}
	return fmt.Sprintf("map[%s]%s", keyDef, elemDef)
}

func goMapKeyTypeDef(scope *codegen.NameScope, att *expr.AttributeExpr, ptr, useDefault, jsonPresence bool) string {
	if expr.IsAny(att.Type) {
		if metaType, _ := codegen.GetMetaType(att); metaType == "" {
			return "any"
		}
	}
	return goCollectionElemTypeDef(scope, att, ptr, useDefault, jsonPresence)
}

func goCollectionElemTypeDef(scope *codegen.NameScope, att *expr.AttributeExpr, ptr, useDefault, jsonPresence bool) string {
	def := goTypeDef(scope, att, ptr, useDefault, jsonPresence)
	if expr.IsObject(att.Type) && !codegen.IsExplicitPresenceType(att) {
		def = "*" + def
	}
	return def
}

func goArrayElemTypeDef(scope *codegen.NameScope, array *expr.Array, ptr, useDefault, jsonPresence bool) string {
	def := goCollectionElemTypeDef(scope, array.ElemType, ptr, useDefault, jsonPresence)
	if jsonPresence && !expr.ArrayElementsAllowNull(array) {
		def = "loom.Nullable[" + def + "]"
	}
	return def
}

func goBodyTypeRef(scope *codegen.NameScope, attribute *expr.AttributeExpr, context *codegen.AttributeContext) string {
	_, userType := attribute.Type.(expr.UserType)
	collection := expr.IsArray(attribute.Type) || expr.IsMap(attribute.Type)
	if context.JSONPresence && !userType && collection && expr.ContainsNonNullableCollectionElement(attribute) {
		return goTypeDef(scope, attribute, context.Pointer, context.UseDefault, true)
	}
	return scope.GoTypeRef(attribute)
}

func goObjectTypeDef(scope *codegen.NameScope, att *expr.AttributeExpr, actual *expr.Object, ptr, useDefault, jsonPresence bool) string {
	_ = actual
	lines := []string{"struct {"}
	ma := expr.NewMappedAttributeExpr(att)
	codegen.WalkMappedAttr(ma, func(name, elem string, _ bool, at *expr.AttributeExpr) error { // nolint: errcheck
		lines = append(lines, goObjectFieldDef(scope, ma, name, elem, at, ptr, useDefault, jsonPresence))
		return nil
	})
	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

func goObjectFieldDef(scope *codegen.NameScope, ma *expr.MappedAttributeExpr, name, elem string, att *expr.AttributeExpr, ptr, useDefault, jsonPresence bool) string {
	fieldName := codegen.GoifyAtt(att, name, true)
	typeDef := goTypeDef(scope, att, ptr, useDefault, jsonPresence)
	wireOptional := !ma.IsRequiredNoDefault(name)
	if expr.AllowsNull(att) && !expr.IsNullable(att) && !expr.IsAny(att.Type) {
		typeDef = "loom.Nullable[" + goValueTypeDef(scope, att, ptr, useDefault, jsonPresence) + "]"
	} else if jsonPresence && wireOptional && !expr.AllowsNull(att) {
		typeDef = "loom.Optional[" + typeDef + "]"
	}
	switch {
	case codegen.IsExplicitPresenceType(att), jsonPresence && wireOptional:
		// Explicit field types define their own presence semantics.
	case expr.IsPrimitive(att.Type):
		if (ptr || ma.IsPrimitivePointer(name, useDefault)) && att.Type != expr.Bytes && !expr.IsAny(att.Type) {
			typeDef = "*" + typeDef
		}
	case expr.IsObject(att.Type) || (expr.IsUnion(att.Type) && !ma.IsRequired(name)):
		typeDef = "*" + typeDef
	}
	description := ""
	if att.Description != "" {
		description = codegen.Comment(att.Description) + "\n\t"
	}
	optional, omitZero := representation.FieldOmission(ma, name, att, ptr, useDefault, jsonPresence)
	tags := representation.AttributeTags(att, elem, optional, omitZero)
	return fmt.Sprintf("\t%s%s %s%s", description, fieldName, typeDef, tags)
}
