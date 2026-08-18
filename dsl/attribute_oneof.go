package dsl

import (
	"strings"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

// OneOf defines or constructs a union type.
//
// OneOf supports two forms:
//
//  1. Inside an object DSL, OneOf defines a named union attribute from a name,
//     an optional description and a function that lists the union branches.
//  2. Wherever a data type can be used, OneOf constructs a union type directly
//     from two or more existing types.
//
// Example:
//
//	var PetOwner = Type("PetOwner", func() {
//	    Name("name", String)
//	    OneOf("pet", "Owner's pet", func() {
//	        Attribute("cat", Cat, "Cats are cool")
//	        Attribute("dog", Dog, "Dogs are cool too")
//	    })
//	})
//
//	var ResultUnion = OneOf(TextResult, JSONResult)
//
//	var _ = Service("result", func() {
//	    Method("show", func() {
//	        Result(OneOf(TextResult, JSONResult))
//	    })
//	})
func OneOf(arg any, args ...any) expr.DataType {
	if looksLikeOneOfAttributeDeclarationSignature(arg, args...) && !isOneOfDeclarationContext() {
		eval.IncompatibleDSL()
		return invalidOneOfType()
	}
	if isOneOfAttributeDeclaration(arg, args...) {
		oneOfAttribute(arg.(string), args...)
		return nil
	}
	if name, ok := arg.(string); ok {
		if isMalformedOneOfAttributeDeclaration(arg, args...) {
			oneOfAttribute(name, args...)
			return nil
		}
		if len(args) > 2 && !areOneOfTypeConstructorArgs(append([]any{arg}, args...)...) {
			eval.TooManyArgError()
			return nil
		}
	}
	return oneOfType(arg, args...)
}

// Untagged configures the current OneOf attribute to encode its selected JSON
// branch directly, without the canonical discriminator/value wrapper. Every
// branch must be a named flat object type with primitive fields so generated
// decoders can validate all candidates and require exactly one match.
func Untagged() {
	attribute, ok := eval.Current().(*expr.AttributeExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	union := expr.AsUnion(attribute.Type)
	if union == nil {
		eval.ReportError("Untagged can only be used with OneOf")
		return
	}
	union.Untagged = true
}

func oneOfAttribute(name string, args ...any) {
	if len(args) == 0 {
		eval.TooFewArgError()
		return
	}
	if len(args) > 2 {
		eval.TooManyArgError()
		return
	}
	fn, ok := args[len(args)-1].(func())
	if !ok {
		eval.InvalidArgError("function", args[len(args)-1])
	}
	var desc string
	if len(args) > 1 {
		desc, ok = args[0].(string)
		if !ok {
			eval.InvalidArgError("string", args[0])
		}
	}
	Attribute(name, &expr.Union{TypeName: name}, desc, fn)
	applyOneOfAttributeMeta(name)
}

func applyOneOfAttributeMeta(name string) {
	attr := currentOneOfAttribute(name)
	if attr == nil || attr.Meta == nil {
		return
	}
	union, ok := attr.Type.(*expr.Union)
	if !ok {
		return
	}
	if !applyOneOfTypeNameMeta(attr, union) {
		return
	}
	if !applyOneOfFieldMeta(attr.Meta, "oneof:type:field", func(value string) {
		union.TypeKey = value
	}) {
		return
	}
	if !applyOneOfFieldMeta(attr.Meta, "oneof:value:field", func(value string) {
		union.ValueKey = value
	}) {
		return
	}
	if union.TypeKey != "" && union.ValueKey != "" && union.TypeKey == union.ValueKey {
		eval.ReportError("oneof:type:field and oneof:value:field cannot be the same (%q)", union.TypeKey)
	}
}

func currentOneOfAttribute(name string) *expr.AttributeExpr {
	parent := currentOneOfParent()
	if parent == nil {
		return nil
	}
	return parent.Find(name)
}

func currentOneOfParent() *expr.AttributeExpr {
	switch def := eval.Current().(type) {
	case *expr.AttributeExpr:
		return def
	case expr.CompositeExpr:
		return def.Attribute()
	default:
		return nil
	}
}

func applyOneOfTypeNameMeta(attr *expr.AttributeExpr, union *expr.Union) bool {
	return applyOneOfFieldMeta(attr.Meta, "oneof:typename", func(value string) {
		union.TypeName = value
		union.ExplicitTypeName = true
	})
}

func applyOneOfFieldMeta(meta expr.MetaExpr, key string, apply func(string)) bool {
	values, ok := meta[key]
	if !ok || len(values) == 0 {
		return true
	}
	if values[0] == "" {
		eval.ReportError("%s meta cannot be empty", key)
		return false
	}
	apply(values[0])
	return true
}

func isOneOfAttributeDeclaration(arg any, args ...any) bool {
	if !looksLikeOneOfAttributeDeclarationSignature(arg, args...) {
		return false
	}
	switch eval.Current().(type) {
	case *expr.AttributeExpr, expr.CompositeExpr:
		return true
	default:
		return false
	}
}

func looksLikeOneOfAttributeDeclarationSignature(arg any, args ...any) bool {
	name, ok := arg.(string)
	if !ok || name == "" {
		return false
	}
	if len(args) == 0 || len(args) > 2 {
		return false
	}
	if _, ok := args[len(args)-1].(func()); !ok {
		return false
	}
	return true
}

func isMalformedOneOfAttributeDeclaration(arg any, args ...any) bool {
	if !isOneOfDeclarationContext() || len(args) == 0 {
		return false
	}
	if _, ok := args[len(args)-1].(func()); ok {
		return true
	}
	return !areOneOfTypeConstructorArgs(append([]any{arg}, args...)...)
}

func isOneOfDeclarationContext() bool {
	switch eval.Current().(type) {
	case *expr.AttributeExpr, expr.CompositeExpr:
		return true
	default:
		return false
	}
}

func areOneOfTypeConstructorArgs(args ...any) bool {
	for _, arg := range args {
		switch arg.(type) {
		case expr.DataType, string:
			continue
		default:
			return false
		}
	}
	return len(args) >= 2
}

func oneOfType(arg any, args ...any) expr.DataType {
	variants := append([]any{arg}, args...)
	if len(variants) < 2 {
		eval.TooFewArgError()
		return invalidOneOfType()
	}

	types := make([]expr.DataType, 0, len(variants))
	invalid := false
	for _, variant := range variants {
		dt := resolveOneOfVariantType(variant)
		if dt == nil {
			invalid = true
			continue
		}
		if !isStableOneOfVariantType(dt) {
			eval.ReportError("constructor OneOf variants of type %s must use a named Type to produce stable discriminators", expr.QualifiedTypeName(dt))
			invalid = true
			continue
		}
		types = append(types, dt)
	}
	if invalid {
		return invalidOneOfType()
	}

	names := expr.DerivedUnionVariantNames(types)
	values := make([]*expr.NamedAttributeExpr, 0, len(types))
	for i, dt := range types {
		values = append(values, &expr.NamedAttributeExpr{
			Name: names[i],
			Attribute: &expr.AttributeExpr{
				Type: dt,
				Meta: expr.MetaExpr{
					"oneof:variant:derived": {"true"},
				},
			},
		})
	}
	return &expr.Union{
		TypeName: expr.DerivedUnionTypeName(names),
		Values:   values,
	}
}

func resolveOneOfVariantType(variant any) expr.DataType {
	switch actual := variant.(type) {
	case expr.DataType:
		return actual
	case string:
		if dt := expr.Root.UserType(actual); dt != nil {
			return dt
		}
		return &expr.UserTypeExpr{
			TypeName: actual,
			UID:      "$type-ref:" + actual,
			AttributeExpr: &expr.AttributeExpr{
				Meta: expr.MetaExpr{"dsl:type:ref": []string{actual}},
			},
		}
	default:
		eval.InvalidArgError("type", variant)
		return nil
	}
}

func isStableOneOfVariantType(dt expr.DataType) bool {
	switch dt.(type) {
	case *expr.Array, *expr.Map, *expr.Object:
		return false
	default:
		return strings.TrimSpace(dt.Name()) != ""
	}
}

func invalidOneOfType() expr.DataType {
	return &expr.Union{TypeName: "InvalidOneOf"}
}

func applyUnionMetaFromAttribute(attr *expr.AttributeExpr) {
	union := expr.AsUnion(attr.Type)
	if union == nil || attr.Meta == nil {
		return
	}
	if typeNames, ok := attr.Meta["oneof:typename"]; ok && len(typeNames) > 0 {
		typeName := typeNames[0]
		if typeName == "" {
			eval.ReportError("oneof:typename meta cannot be empty")
			return
		}
		union.TypeName = typeName
		union.ExplicitTypeName = true
	}
	if typeKeys, ok := attr.Meta["oneof:type:field"]; ok && len(typeKeys) > 0 {
		typeKey := typeKeys[0]
		if typeKey == "" {
			eval.ReportError("oneof:type:field meta cannot be empty")
			return
		}
		union.TypeKey = typeKey
	}
	if valueKeys, ok := attr.Meta["oneof:value:field"]; ok && len(valueKeys) > 0 {
		valueKey := valueKeys[0]
		if valueKey == "" {
			eval.ReportError("oneof:value:field meta cannot be empty")
			return
		}
		union.ValueKey = valueKey
	}
	if union.TypeKey != "" && union.ValueKey != "" && union.TypeKey == union.ValueKey {
		eval.ReportError("oneof:type:field and oneof:value:field cannot be the same (%q)", union.TypeKey)
	}
}
