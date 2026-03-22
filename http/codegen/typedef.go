package codegen

import (
	"fmt"
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
func goTypeDef(scope *codegen.NameScope, att *expr.AttributeExpr, ptr, useDefault bool) string {
	switch actual := att.Type.(type) {
	case expr.Primitive:
		return goPrimitiveTypeDef(att, actual)
	case *expr.Array:
		return goArrayTypeDef(scope, actual, ptr, useDefault)
	case *expr.Map:
		return goMapTypeDef(scope, actual, ptr, useDefault)
	case *expr.Object:
		return goObjectTypeDef(scope, att, actual, ptr, useDefault)
	case expr.UserType, *expr.Union:
		return scope.GoTypeName(att)
	default:
		panic(fmt.Sprintf("unknown data type %T", actual)) // bug
	}
}

func goPrimitiveTypeDef(att *expr.AttributeExpr, actual expr.Primitive) string {
	if t, _ := codegen.GetMetaType(att); t != "" {
		return t
	}
	return codegen.GoNativeTypeName(actual)
}

func goArrayTypeDef(scope *codegen.NameScope, actual *expr.Array, ptr, useDefault bool) string {
	return "[]" + goCollectionElemTypeDef(scope, actual.ElemType, ptr, useDefault)
}

func goMapTypeDef(scope *codegen.NameScope, actual *expr.Map, ptr, useDefault bool) string {
	keyDef := goCollectionElemTypeDef(scope, actual.KeyType, ptr, useDefault)
	elemDef := goCollectionElemTypeDef(scope, actual.ElemType, ptr, useDefault)
	return fmt.Sprintf("map[%s]%s", keyDef, elemDef)
}

func goCollectionElemTypeDef(scope *codegen.NameScope, att *expr.AttributeExpr, ptr, useDefault bool) string {
	def := goTypeDef(scope, att, ptr, useDefault)
	if expr.IsObject(att.Type) {
		def = "*" + def
	}
	return def
}

func goObjectTypeDef(scope *codegen.NameScope, att *expr.AttributeExpr, actual *expr.Object, ptr, useDefault bool) string {
	_ = actual
	lines := []string{"struct {"}
	ma := expr.NewMappedAttributeExpr(att)
	parent := ma.Attribute()
	codegen.WalkMappedAttr(ma, func(name, elem string, _ bool, at *expr.AttributeExpr) error { // nolint: errcheck
		lines = append(lines, goObjectFieldDef(scope, ma, parent, name, elem, at, ptr, useDefault))
		return nil
	})
	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

func goObjectFieldDef(scope *codegen.NameScope, ma *expr.MappedAttributeExpr, parent *expr.AttributeExpr, name, elem string, att *expr.AttributeExpr, ptr, useDefault bool) string {
	fieldName := codegen.GoifyAtt(att, name, true)
	typeDef := goTypeDef(scope, att, ptr, useDefault)
	if expr.IsPrimitive(att.Type) {
		if (ptr || parent.IsPrimitivePointer(name, useDefault)) && att.Type != expr.Bytes && att.Type != expr.Any {
			typeDef = "*" + typeDef
		}
	} else if expr.IsObject(att.Type) {
		typeDef = "*" + typeDef
	}
	description := ""
	if att.Description != "" {
		description = codegen.Comment(att.Description) + "\n\t"
	}
	optional := objectFieldOptional(ma, name, ptr, useDefault)
	tags := attributeTags(parent, att, elem, optional)
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
func attributeTags(parent, att *expr.AttributeExpr, t string, optional bool) string {
	if tags := codegen.AttributeTags(parent, att); tags != "" {
		return tags
	}
	var o string
	// Always use omitempty for JSON-RPC ID attributes, even when required
	// since it is part of a different top-level field in the transport
	if optional || isJSONRPCID(att) {
		o = ",omitempty"
	}
	jsonName := t
	if att != nil && att.Meta != nil {
		if v := att.Meta["struct:tag:json:name"]; len(v) > 0 && v[0] != "" {
			jsonName = strings.Join(v, ",")
		}
	}
	return fmt.Sprintf(" `form:\"%s%s\" json:\"%s%s\" xml:\"%s%s\"`", t, o, jsonName, o, t, o)
}

// isJSONRPCID checks if the attribute is marked as a JSON-RPC ID attribute
func isJSONRPCID(att *expr.AttributeExpr) bool {
	if att.Meta == nil {
		return false
	}
	_, ok := att.Meta["jsonrpc:id"]
	return ok
}
