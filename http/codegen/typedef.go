package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
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
	return codegen.GoNativeTypeName(actual)
}

func goArrayTypeDef(scope *codegen.NameScope, actual *expr.Array, ptr, useDefault, jsonPresence bool) string {
	return "[]" + goArrayElemTypeDef(scope, actual, ptr, useDefault, jsonPresence)
}

func goMapTypeDef(scope *codegen.NameScope, actual *expr.Map, ptr, useDefault, jsonPresence bool) string {
	keyDef := goCollectionElemTypeDef(scope, actual.KeyType, ptr, useDefault, jsonPresence)
	elemDef := goCollectionElemTypeDef(scope, actual.ElemType, ptr, useDefault, jsonPresence)
	if jsonPresence && !expr.MapValuesAllowNull(actual) {
		elemDef = "loom.Nullable[" + elemDef + "]"
	}
	return fmt.Sprintf("map[%s]%s", keyDef, elemDef)
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
	parent := ma.Attribute()
	codegen.WalkMappedAttr(ma, func(name, elem string, _ bool, at *expr.AttributeExpr) error { // nolint: errcheck
		lines = append(lines, goObjectFieldDef(scope, ma, parent, name, elem, at, ptr, useDefault, jsonPresence))
		return nil
	})
	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

func goObjectFieldDef(scope *codegen.NameScope, ma *expr.MappedAttributeExpr, parent *expr.AttributeExpr, name, elem string, att *expr.AttributeExpr, ptr, useDefault, jsonPresence bool) string {
	fieldName := codegen.GoifyAtt(att, name, true)
	typeDef := goTypeDef(scope, att, ptr, useDefault, jsonPresence)
	wireOptional := !ma.IsRequiredNoDefault(name)
	if expr.AllowsNull(att) && !expr.IsNullable(att) {
		typeDef = "loom.Nullable[" + goValueTypeDef(scope, att, ptr, useDefault, jsonPresence) + "]"
	} else if jsonPresence && wireOptional && !expr.AllowsNull(att) {
		typeDef = "loom.Optional[" + typeDef + "]"
	}
	switch {
	case codegen.IsExplicitPresenceType(att), jsonPresence && wireOptional:
		// Explicit field types define their own presence semantics.
	case expr.IsPrimitive(att.Type):
		if (ptr || parent.IsPrimitivePointer(name, useDefault)) && att.Type != expr.Bytes && att.Type != expr.Any {
			typeDef = "*" + typeDef
		}
	case expr.IsObject(att.Type) || (expr.IsUnion(att.Type) && !parent.IsRequired(name)):
		typeDef = "*" + typeDef
	}
	description := ""
	if att.Description != "" {
		description = codegen.Comment(att.Description) + "\n\t"
	}
	optional := objectFieldOptional(ma, name, ptr, useDefault)
	omitZero := wireOptional && (jsonPresence || codegen.IsExplicitPresenceType(att))
	tags := attributeTags(att, elem, optional, omitZero)
	return fmt.Sprintf("\t%s%s %s%s", description, fieldName, typeDef, tags)
}

func objectFieldOptional(ma *expr.MappedAttributeExpr, name string, ptr, useDefault bool) bool {
	switch {
	case ptr:
		return true
	case useDefault:
		return !ma.IsRequired(name) && !ma.HasDefaultValue(name)
	default:
		return !ma.IsRequired(name)
	}
}

// attributeTags computes the struct field tags.
func attributeTags(att *expr.AttributeExpr, t string, optional, omitZero bool) string {
	var omitEmpty string
	// Always use omitempty for JSON-RPC ID attributes, even when required
	// since it is part of a different top-level field in the transport
	if optional || isJSONRPCID(att) {
		omitEmpty = ",omitempty"
	}
	jsonOption := omitEmpty
	if omitZero {
		jsonOption = ",omitzero"
	}
	jsonName := t
	explicitJSON := false
	if att != nil && att.Meta != nil {
		if v := att.Meta["struct:tag:json"]; len(v) > 0 {
			jsonName = strings.Join(v, ",")
			explicitJSON = true
		}
		if v := att.Meta["struct:tag:json:name"]; len(v) > 0 && v[0] != "" {
			if !explicitJSON {
				jsonName = strings.Join(v, ",")
			}
		}
	}
	if omitZero && strings.Split(jsonName, ",")[0] == "-" {
		panic(codegen.NewError(nil, att, fmt.Errorf("JSON field %q uses presence semantics and cannot be omitted with tag '-'", t)))
	}
	jsonName = mergeJSONOmitOption(jsonName, strings.TrimPrefix(jsonOption, ","))
	if custom := mergedAttributeTags(att, jsonName, omitZero || explicitJSON || hasJSONTagName(att)); custom != "" {
		return custom
	}
	return fmt.Sprintf(" `form:\"%s%s\" json:\"%s\" xml:\"%s%s\"`", t, omitEmpty, jsonName, t, omitEmpty)
}

func mergedAttributeTags(att *expr.AttributeExpr, jsonTag string, includeJSON bool) string {
	if att == nil || att.Meta == nil {
		return ""
	}
	tags := make(map[string]string)
	for key, values := range att.Meta {
		if !strings.HasPrefix(key, "struct:tag:") || key == "struct:tag:json:name" {
			continue
		}
		tags[strings.TrimPrefix(key, "struct:tag:")] = strings.Join(values, ",")
	}
	if len(tags) == 0 {
		return ""
	}
	if includeJSON {
		tags["json"] = jsonTag
	}
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s:\"%s\"", key, tags[key]))
	}
	return " `" + strings.Join(parts, " ") + "`"
}

func hasJSONTagName(att *expr.AttributeExpr) bool {
	if att == nil || att.Meta == nil {
		return false
	}
	return len(att.Meta["struct:tag:json:name"]) > 0
}

func mergeJSONOmitOption(tag, option string) string {
	if option == "" {
		return tag
	}
	parts := strings.Split(tag, ",")
	options := make([]string, 0, len(parts))
	seen := false
	for _, part := range parts[1:] {
		if part == "omitempty" || part == "omitzero" {
			if !seen {
				options = append(options, option)
				seen = true
			}
			continue
		}
		options = append(options, part)
	}
	if !seen {
		options = append(options, option)
	}
	return strings.Join(append([]string{parts[0]}, options...), ",")
}

// isJSONRPCID checks if the attribute is marked as a JSON-RPC ID attribute
func isJSONRPCID(att *expr.AttributeExpr) bool {
	if att.Meta == nil {
		return false
	}
	_, ok := att.Meta["jsonrpc:id"]
	return ok
}
