package dsl

import (
	"maps"
	"slices"
	"strings"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

// OneOf defines or constructs a union type.
//
// OneOf supports three forms:
//
//  1. Inside an object DSL, OneOf defines a named union attribute from a name,
//     an optional description and a function that lists the union branches.
//  2. Wherever a data type can be used, OneOf constructs a union type directly
//     from two or more existing types.
//  3. A sole function constructs a union by declaring named Attribute branches.
//     The function may call Untagged, set union naming/envelope or closure
//     metadata, and set each branch's constraints. Nullable, annotations and
//     constraints on the whole union belong in the enclosing Type, Attribute,
//     Payload or Result DSL; the constructor rejects them rather than dropping
//     settings that the returned data type cannot retain.
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
// branch may be primitive, Any, a scalar alias, a named object, an array, a
// string-keyed map, or another untagged union. Nullable branches preserve null
// in their generated carriers. Nested fields and elements must use primitives,
// named objects, untagged unions, or arrays and string-keyed maps of these.
// Encoding and decoding require the same unique branch in the schema and Go
// representation. Overlapping branches are legal; ambiguous values are rejected.
// In OpenAPIBody and OpenAPIRequestBody, Untagged describes only the schema:
// scalar and nested union branches are allowed, and runtime types are unchanged.
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
	attribute.SetUntagged()
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
	Attribute(name, &expr.Union{TypeName: expr.AttributeName(name)}, desc, fn)
	applyOneOfAttributeMeta(name)
}

func applyOneOfAttributeMeta(name string) {
	applyOneOfMetadata(currentOneOfAttribute(name))
}

// applyOneOfMetadata completes the retained union before its owner publishes copies.
func applyOneOfMetadata(attr *expr.AttributeExpr) {
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
	if fn, ok := arg.(func()); ok && len(args) == 0 {
		union := &expr.Union{TypeName: "OneOf"}
		attribute := &expr.AttributeExpr{Type: union}
		if !eval.Execute(fn, attribute) || !validateOneOfConstructor(attribute) {
			return invalidOneOfType()
		}
		names := make([]string, len(union.Values))
		for index, branch := range union.Values {
			names[index] = branch.Name
		}
		union.TypeName = expr.DerivedUnionTypeName(names)
		applyUnionMetaFromAttribute(attribute)
		captured, err := attribute.CaptureUnionClosure()
		if err != nil {
			eval.ReportError("%s", err)
			return invalidOneOfType()
		}
		return captured
	}
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

// validateOneOfConstructor admits only settings carried by the returned Union.
// Branch attributes keep their own complete DSL; this check concerns the
// temporary root attribute that construction discards.
func validateOneOfConstructor(attribute *expr.AttributeExpr) bool {
	valid := true
	for _, setting := range []struct {
		name    string
		present bool
	}{
		{"Nullable", attribute.Nullable},
		{"Description", attribute.Description != ""},
		{"Title", attribute.Title != ""},
		{"Docs", attribute.Docs != nil},
		{"validations", attribute.Validation != nil},
		{"Default", attribute.DefaultValue != nil},
		{"Example", len(attribute.UserExamples) > 0},
		{"Extend", len(attribute.Bases) > 0},
		{"Reference", len(attribute.References) > 0},
		{"AdditionalAttributes", attribute.AdditionalAttributes != nil},
	} {
		if setting.present {
			eval.ReportError("OneOf constructor cannot set %s on the whole union; use the enclosing Type, Attribute, Payload or Result DSL", setting.name)
			valid = false
		}
	}
	for _, key := range slices.Sorted(maps.Keys(attribute.Meta)) {
		switch key {
		case "oneof:typename", "oneof:type:field", "oneof:value:field", "openapi:unevaluatedProperties":
		default:
			eval.ReportError("OneOf constructor cannot retain metadata %q; use the enclosing Type, Attribute, Payload or Result DSL where that metadata is valid", key)
			valid = false
		}
	}
	return valid
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
	if typeName, ok := attr.Meta.First("oneof:typename"); ok {
		if typeName == "" {
			eval.ReportError("oneof:typename meta cannot be empty")
			return
		}
		union.TypeName = typeName
		union.ExplicitTypeName = true
	}
	if typeKey, ok := attr.Meta.First("oneof:type:field"); ok {
		if typeKey == "" {
			eval.ReportError("oneof:type:field meta cannot be empty")
			return
		}
		union.TypeKey = typeKey
	}
	if valueKey, ok := attr.Meta.First("oneof:value:field"); ok {
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
