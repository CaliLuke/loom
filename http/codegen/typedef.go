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
//     required when layout.Pointer is true so that the generated code may validate
//     explicitly.
//
// layout.UseDefault directs whether fields holding primitive types with default values
// should hold pointers when layout.Pointer is false. If it is true then the fields are
// values even when not required (to account for the fact that they have a
// default value so cannot be nil) otherwise the fields are values only when
// required.
func goTypeDef(scope *codegen.NameScope, att *expr.AttributeExpr, layout *codegen.AttributeContext) string {
	if t, _ := codegen.GetMetaType(att); codegen.IsExplicitPresenceType(att) && t != "" {
		return t
	}
	if expr.IsNullable(att) {
		return "loom.Nullable[" + goValueTypeDef(scope, att, layout) + "]"
	}
	return goValueTypeDef(scope, att, layout)
}

func goValueTypeDef(scope *codegen.NameScope, att *expr.AttributeExpr, layout *codegen.AttributeContext) string {
	switch actual := att.Type.(type) {
	case expr.Primitive:
		return goPrimitiveTypeDef(att, actual)
	case *expr.Array:
		return goArrayTypeDef(scope, actual, layout)
	case *expr.Map:
		return goMapTypeDef(scope, actual, layout)
	case *expr.Object:
		return goObjectTypeDef(scope, att, layout)
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

func goArrayTypeDef(scope *codegen.NameScope, actual *expr.Array, layout *codegen.AttributeContext) string {
	return "[]" + goArrayElemTypeDef(scope, actual, layout)
}

func goMapTypeDef(scope *codegen.NameScope, actual *expr.Map, layout *codegen.AttributeContext) string {
	keyDef := goMapKeyTypeDef(scope, actual.KeyType, layout)
	elemDef := goCollectionElemTypeDef(scope, actual.ElemType, layout)
	if layout.CollectionElementPresence && !expr.MapValuesAllowNull(actual) {
		elemDef = "loom.Optional[" + elemDef + "]"
	}
	return fmt.Sprintf("map[%s]%s", keyDef, elemDef)
}

func goMapKeyTypeDef(scope *codegen.NameScope, att *expr.AttributeExpr, layout *codegen.AttributeContext) string {
	if expr.IsAny(att.Type) {
		if metaType, _ := codegen.GetMetaType(att); metaType == "" {
			return "any"
		}
	}
	return goCollectionElemTypeDef(scope, att, layout)
}

func goCollectionElemTypeDef(scope *codegen.NameScope, att *expr.AttributeExpr, layout *codegen.AttributeContext) string {
	def := goTypeDef(scope, att, layout)
	if expr.IsObject(att.Type) && !codegen.IsExplicitPresenceType(att) {
		def = "*" + def
	}
	return def
}

func goArrayElemTypeDef(scope *codegen.NameScope, array *expr.Array, layout *codegen.AttributeContext) string {
	def := goCollectionElemTypeDef(scope, array.ElemType, layout)
	if layout.CollectionElementPresence && !expr.ArrayElementsAllowNull(array) {
		def = "loom.Optional[" + def + "]"
	}
	return def
}

func goBodyTypeRef(scope *codegen.NameScope, attribute *expr.AttributeExpr, context *codegen.AttributeContext) string {
	_, userType := attribute.Type.(expr.UserType)
	collection := expr.IsArray(attribute.Type) || expr.IsMap(attribute.Type)
	if context.JSONPresence && !userType && collection && expr.ContainsNonNullableCollectionElement(attribute) {
		return goTypeDef(scope, attribute, context)
	}
	return scope.GoTypeRef(attribute)
}

func goObjectTypeDef(scope *codegen.NameScope, att *expr.AttributeExpr, layout *codegen.AttributeContext) string {
	lines := []string{"struct {"}
	ma := expr.NewObjectFields(att)
	for name, selection := range ma.Fields() {
		at := att.PhysicalMemberAttribute(selection)
		lines = append(lines, goObjectFieldDef(scope, ma, name, at, layout))
	}
	if carrier := att.AdditionalMembersCarrier(); carrier != nil {
		lines = append(lines, goAdditionalMembersFieldDef(scope, carrier, layout))
	}
	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

// goAdditionalMembersFieldDef declares the map that carries a typed object's
// undeclared members. JSON v2 inlines it as the object's remaining members.
func goAdditionalMembersFieldDef(scope *codegen.NameScope, carrier *expr.AttributeExpr, layout *codegen.AttributeContext) string {
	return fmt.Sprintf("\t%s %s `json:\",embed\"`", expr.AdditionalAttributesFieldName, goTypeDef(scope, carrier, layout))
}

func goObjectFieldDef(scope *codegen.NameScope, ma *expr.ObjectFields, name string, att *expr.AttributeExpr, layout *codegen.AttributeContext) string {
	fieldName := codegen.GoifyAtt(att, name, true)
	typeDef := goTypeDef(scope, att, layout)
	field := representation.ResolveField(ma, name, att, layout.Pointer, layout.UseDefault, layout.JSONPresence)
	if field.Layout.WrapOptional {
		typeDef = "loom.Optional[" + typeDef + "]"
	}
	if field.Layout.Pointer {
		typeDef = "*" + typeDef
	}
	description := ""
	if att.Description != "" {
		description = codegen.Comment(att.Description) + "\n\t"
	}
	return fmt.Sprintf("\t%s%s %s%s", description, fieldName, typeDef, field.Encoding.Tags)
}
