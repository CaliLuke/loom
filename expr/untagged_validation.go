package expr

import (
	"strings"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/internal/encodingmeta"
)

func validateUntaggedBranches(union *Union, ctx string, parent eval.Expression) *eval.ValidationErrors {
	errors := new(eval.ValidationErrors)
	for _, branch := range union.Values {
		ut, named := branch.Attribute.Type.(UserType)
		if !named || !IsObject(ut) && !IsArray(ut) {
			errors.Add(parent, "%suntagged OneOf branch %q must be a concrete named object or array type", ctx, branch.Name)
			continue
		}
		if AllowsNull(branch.Attribute) {
			errors.Add(parent, "%suntagged OneOf branch %q cannot admit null; declare nullability on the enclosing union occurrence", ctx, branch.Name)
		}
		validateUntaggedShape(branch.Attribute, ctx, branch.Name, parent, errors, make(map[*AttributeExpr]bool))
	}
	return errors
}

func validateUntaggedShape(att *AttributeExpr, ctx, branch string, parent eval.Expression, errors *eval.ValidationErrors, seen map[*AttributeExpr]bool) {
	if seen[att] {
		return
	}
	seen[att] = true
	validateUntaggedCodec(att, ctx, branch, parent, errors)
	if ut, named := att.Type.(UserType); named {
		validateUntaggedShape(ut.Attribute(), ctx, branch, parent, errors, seen)
		return
	}
	if array := AsArray(att.Type); array != nil {
		if !isUntaggedBranchFieldType(array.ElemType.Type) {
			errors.Add(parent, "%suntagged OneOf branch %q array elements must be primitive, a concrete named object type, or an array of either", ctx, branch)
		}
		validateUntaggedShape(array.ElemType, ctx, branch, parent, errors, seen)
		return
	}
	object := AsObject(att.Type)
	if object == nil {
		return
	}
	if additional, ok := att.Meta.Last("openapi:additionalProperties"); ok && additional != "false" {
		errors.Add(parent, "%suntagged OneOf branch %q must use the default open object or openapi:additionalProperties false", ctx, branch)
	}
	names := make(map[string]bool)
	for _, field := range *object {
		if !isUntaggedBranchFieldType(field.Attribute.Type) {
			errors.Add(parent, "%suntagged OneOf branch %q field %q must be primitive, a concrete named object type, or an array of either", ctx, branch, field.Name)
		}
		name := JSONFieldName(AttributeName(field.Name), field.Attribute)
		switch {
		case name == "":
			errors.Add(parent, "%suntagged OneOf branch %q field %q cannot use an empty JSON tag name", ctx, branch, field.Name)
		case name == "-":
			errors.Add(parent, "%suntagged OneOf branch %q field %q cannot use json tag %q", ctx, branch, field.Name, name)
		case names[name]:
			errors.Add(parent, "%suntagged OneOf branch %q has duplicate JSON field name %q", ctx, branch, name)
		}
		names[name] = true
		validateUntaggedShape(field.Attribute, ctx, branch, parent, errors, seen)
	}
}

func validateUntaggedCodec(att *AttributeExpr, ctx, branch string, parent eval.Expression, errors *eval.ValidationErrors) {
	if tag, exists := attributeJSONTag(att); exists {
		for _, option := range strings.Split(tag, ",")[1:] {
			switch option {
			case "", "omitempty", "omitzero", "case:strict":
			default:
				errors.Add(parent, "%suntagged OneOf branch %q cannot use JSON codec option %q", ctx, branch, option)
			}
		}
	}
	override := encodingmeta.SchemaOverride(att.Meta)
	if format, exists := att.Meta.Last("openapi:format"); exists && format == "" && IsPrimitive(att.Type) && att.Type != Bytes {
		// Import preserves the absence of an optional format annotation. This
		// changes neither JSON membership nor the concrete decoder contract.
		override = len(att.Meta["openapi:contentEncoding"]) > 0 || len(att.Meta["openapi:contentMediaType"]) > 0
	}
	if encodingmeta.Replacement(att.Meta) != "" && !legacyNullableWrapper(att) || override {
		errors.Add(parent, "%suntagged OneOf branch %q cannot use an opaque codec or schema encoding override", ctx, branch)
	}
}
