//nolint:errcheck // Generator helpers write only to in-memory buffers/builders.
package codegen

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

func recurseValidationCode(att *expr.AttributeExpr, put expr.UserType, attCtx *AttributeContext, req, alias, view bool, target, context string, seen map[string]*bytes.Buffer) *bytes.Buffer {
	if seen == nil {
		seen = make(map[string]*bytes.Buffer)
	}
	var (
		buf      = new(bytes.Buffer)
		first    = true
		ut, isUT = att.Type.(expr.UserType)
	)
	// Break infinite recursions before presence wrappers recurse into their
	// concrete named value. Keep wrapped and concrete states distinct.
	if isUT && !alias {
		key := ut.ID()
		if isNullableAttribute(att) {
			key += ":nullable"
		}
		if existing, ok := seen[key]; ok {
			return existing
		}
		seen[key] = buf
	}
	if isNullableAttribute(att) {
		fmt.Fprint(buf, validateNullableAttribute(attCtx, att, put, target, context, req, view, seen))
		return buf
	}

	// Write validations on attribute if any.
	validation := validationCode(att, attCtx, req, alias, target, context)
	if validation != "" {
		fmt.Fprint(buf, validation)
		first = false
	}

	// Recurse down depending on attribute type.
	switch {
	case expr.IsObject(att.Type):
		renderObjectValidation(buf, &first, att, put, ut, isUT, attCtx, view, target, context, seen)
	case expr.IsArray(att.Type):
		renderArrayValidationCode(buf, &first, expr.AsArray(att.Type), put, attCtx, view, target, context, seen)
	case expr.IsMap(att.Type):
		renderMapValidationCode(buf, &first, expr.AsMap(att.Type), put, attCtx, view, target, context, seen)
	case expr.IsUnion(att.Type):
		renderUnionValidationCode(buf, &first, expr.AsUnion(att.Type), put, attCtx, view, target, context, seen)
	}

	return buf
}

func appendValidationBlock(buf *bytes.Buffer, first *bool, val string) {
	if val == "" {
		return
	}
	if !*first {
		buf.WriteByte('\n')
	} else {
		*first = false
	}
	fmt.Fprint(buf, val)
}

func renderObjectValidation(buf *bytes.Buffer, first *bool, att *expr.AttributeExpr, put, ut expr.UserType, isUT bool, attCtx *AttributeContext, view bool, target, context string, seen map[string]*bytes.Buffer) {
	if isUT {
		put = ut
	}
	mapped := expr.NewMappedAttributeExpr(att)
	parentRequired := parentRequiredValidationFields(att, attCtx)
	for _, nat := range *(expr.AsObject(att.Type)) {
		tgt := target + "." + attCtx.Scope.Field(nat.Attribute, nat.Name, true)
		ctx := context + "." + nat.Name
		var val string
		switch attCtx.FieldPresence(mapped, nat.Name, nat.Attribute) {
		case OptionalPresence:
			val = validateOptionalAttribute(attCtx, nat.Attribute, put, tgt, ctx, view, seen)
		case NullablePresence:
			_, requiredByParent := parentRequired[nat.Name]
			required := att.IsRequired(nat.Name) && !requiredByParent
			val = validateNullableAttribute(attCtx, nat.Attribute, put, tgt, ctx, required, view, seen)
		default:
			val = validateAttribute(attCtx, nat.Attribute, put, tgt, ctx, att.IsRequired(nat.Name), view, seen)
		}
		appendValidationBlock(buf, first, val)
	}
}

func parentRequiredValidationFields(att *expr.AttributeExpr, attCtx *AttributeContext) map[string]struct{} {
	validation := mergedValidation(att)
	if validation == nil {
		return nil
	}
	fields := generatedRequiredValidationFrom(att, validation, attCtx)
	result := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		result[field] = struct{}{}
	}
	return result
}

func renderArrayValidationCode(buf *bytes.Buffer, first *bool, arr *expr.Array, put expr.UserType, attCtx *AttributeContext, view bool, target, context string, seen map[string]*bytes.Buffer) {
	ctx := attCtx
	if ctx.Pointer && expr.IsPrimitive(arr.ElemType.Type) {
		ctx = attCtx.Dup()
		ctx.Pointer = false
	}
	val := validateAttribute(ctx, arr.ElemType, put, "e", context+"[*]", true, view, seen)
	if val != "" || arr.NonNullableElems {
		appendValidationBlock(buf, first, renderArrayValidation(target, val, arr.NonNullableElems, context))
	}
}

func renderMapValidationCode(buf *bytes.Buffer, first *bool, m *expr.Map, put expr.UserType, attCtx *AttributeContext, view bool, target, context string, seen map[string]*bytes.Buffer) {
	keyCtx := attCtx.Dup()
	keyCtx.Pointer = false
	valueCtx := attCtx
	if valueCtx.Pointer && expr.IsPrimitive(m.ElemType.Type) {
		valueCtx = attCtx.Dup()
		valueCtx.Pointer = false
	}
	keyVal := prefixedValidation(validateAttribute(keyCtx, m.KeyType, put, "k", context+".key", true, view, seen))
	valueVal := prefixedValidation(validateAttribute(valueCtx, m.ElemType, put, "v", context+"[key]", true, view, seen))
	if keyVal != "" || valueVal != "" {
		appendValidationBlock(buf, first, renderMapValidation(target, keyVal, valueVal))
	}
}

func renderUnionValidationCode(buf *bytes.Buffer, first *bool, u *expr.Union, put expr.UserType, attCtx *AttributeContext, view bool, target, context string, seen map[string]*bytes.Buffer) {
	if _, ok := attCtx.Scope.(*AttributeScope); ok {
		cases := renderUnionSumValidationCases(u, put, attCtx, view, context, seen)
		if len(cases) > 0 {
			appendValidationBlock(buf, first, renderUnionSumValidation(target, cases))
		}
		return
	}
	types, vals := renderUnionInterfaceValidationCases(u, put, attCtx, view, context, seen)
	if len(vals) > 0 {
		appendValidationBlock(buf, first, renderUnionValidation(target, types, vals))
	}
}

func renderUnionSumValidationCases(u *expr.Union, put expr.UserType, attCtx *AttributeContext, view bool, context string, seen map[string]*bytes.Buffer) []unionValidationCase {
	cases := make([]unionValidationCase, 0, len(u.Values))
	for _, v := range u.Values {
		// Sum-type unions (struct-based, with Kind/AsX accessors) store each
		// branch as either a value (primitives, arrays, maps) or a pointer
		// (object user types). Request-body validation may already use value
		// semantics for nested objects, so preserve the enclosing context and
		// only keep pointer semantics when both layers use pointers.
		unionCtx := attCtx.Dup()
		_, named := v.Attribute.Type.(expr.UserType)
		if !named {
			unionCtx.JSONPresence = false
			unionCtx.UseDefault = true
		}
		if expr.IsObject(v.Attribute.Type) && !named {
			// HTTP sum types store inline object branches with native pointer
			// presence, even when the union field uses JSON presence.
			unionCtx.Pointer = false
		} else {
			unionCtx.Pointer = unionCtx.Pointer && expr.IsObject(v.Attribute.Type)
		}
		val := validateAttribute(unionCtx, v.Attribute, put, "actual", context+".value", true, view, seen)
		if val == "" {
			continue
		}
		cases = append(cases, unionValidationCase{
			TypeTag:       expr.UnionVariantTag(v),
			FieldName:     Goify(v.Name, true),
			RequiresValue: strings.HasPrefix(strings.TrimSpace(val), "if actual != nil {"),
			Context:       context + ".value",
			Validation:    val,
		})
	}
	return cases
}

func renderUnionInterfaceValidationCases(u *expr.Union, put expr.UserType, attCtx *AttributeContext, view bool, context string, seen map[string]*bytes.Buffer) ([]string, []string) {
	var (
		vals  []string
		types []string
	)
	for _, v := range u.Values {
		vatt := v.Attribute
		if view {
			unionCtx := attCtx.Dup()
			unionCtx.Pointer = false
			val := validateAttribute(unionCtx, vatt, put, "v", context+".value", true, view, seen)
			if val != "" {
				types = append(types, attCtx.Scope.Ref(vatt, attCtx.DefaultPkg))
				vals = append(vals, val)
			}
			continue
		}
		fieldName := attCtx.Scope.Field(vatt, v.Name, true)
		val := validateAttribute(attCtx, vatt, put, "v."+fieldName, context+".value", true, view, seen)
		if val != "" {
			tref := attCtx.Scope.Ref(&expr.AttributeExpr{Type: put}, attCtx.DefaultPkg)
			types = append(types, tref+"_"+fieldName)
			vals = append(vals, val)
		}
	}
	return types, vals
}

func prefixedValidation(val string) string {
	if val == "" {
		return ""
	}
	return "\n" + val
}

func validateAttribute(ctx *AttributeContext, att *expr.AttributeExpr, put expr.UserType, target, context string, req, view bool, seen map[string]*bytes.Buffer) string {
	if isNullableAttribute(att) {
		return validateNullableAttribute(ctx, att, put, target, context, req, view, seen)
	}
	ut, isUT := att.Type.(expr.UserType)
	if !isUT {
		code := recurseValidationCode(att, put, ctx, req, false, view, target, context, seen).String()
		if code == "" {
			return ""
		}
		if expr.IsArray(att.Type) || expr.IsMap(att.Type) || expr.IsUnion(att.Type) {
			return code
		}
		if !ctx.Pointer && (req || (att.DefaultValue != nil && ctx.UseDefault)) {
			return code
		}
		cond := "if " + target + " != nil {\n"
		if strings.HasPrefix(code, cond) {
			return code
		}
		return cond + code + "\n}"
	}
	// Alias user types: validate underlying attribute with alias flag so that
	// validation operates on the base value type while preserving pointer
	// semantics from the current attribute context.
	if expr.IsAlias(ut) {
		// Preserve field-level attributes (e.g., DefaultValue, Required) while
		// validating alias user types against their underlying base. Passing
		// the original attribute with alias=true ensures validations operate
		// on the correct value type without dropping field defaults.
		code := recurseValidationCode(att, put, ctx, req, true, view, target, context, seen).String()
		if code == "" {
			return ""
		}
		// For optional pointer fields, wrap validation code in nil check
		if !ctx.Pointer && (req || (att.DefaultValue != nil && ctx.UseDefault)) {
			return code
		}
		cond := "if " + target + " != nil {\n"
		if strings.HasPrefix(code, cond) {
			return code
		}
		return cond + code + "\n}"
	}
	if expr.IsUnion(ut.Attribute().Type) {
		return recurseValidationCode(att, put, ctx, req, false, view, target, context, seen).String()
	}
	if !hasValidations(ctx, ut) {
		return ""
	}
	var buf bytes.Buffer
	name := ctx.Scope.Name(att, "", ctx.Pointer, ctx.UseDefault)
	// Use the scoped type name directly to preserve identifiers such as
	// protocol buffer-reserved names that include a trailing underscore
	// (e.g., Message_). Applying Goify here would drop underscores and
	// cause mismatches between function declarations and call sites.
	fmt.Fprint(&buf, renderUserValidation(name, target))
	return "if " + target + " != nil {\n\t" + buf.String() + "\n}"
}

func isNullableAttribute(att *expr.AttributeExpr) bool {
	if att == nil {
		return false
	}
	if expr.IsNullable(att) {
		return true
	}
	value, ok := att.Meta.Last("openapi:nullable")
	if ok && value != "false" {
		return true
	}
	metaType, _ := GetMetaType(att)
	return strings.HasPrefix(metaType, "loom.Nullable[")
}

func validateNullableAttribute(
	ctx *AttributeContext,
	att *expr.AttributeExpr,
	put expr.UserType,
	target string,
	context string,
	required bool,
	view bool,
	seen map[string]*bytes.Buffer,
) string {
	underlying := concretePresenceAttribute(att)
	underlying.Meta = make(expr.MetaExpr, len(att.Meta))
	for name, values := range att.Meta {
		if name == "openapi:nullable" || name == "struct:field:type" {
			continue
		}
		underlying.Meta[name] = append([]string(nil), values...)
	}

	valueCtx := presenceValueContext(ctx, underlying)
	validation := recurseValidationCode(underlying, put, valueCtx, true, false, view, "actual", context, seen).String()
	var lines []string
	if required {
		field, parent := nullableValidationContext(context)
		lines = append(lines, "if !"+target+".Present() {\n\terr = loom.MergeErrors(err, loom.MissingFieldError("+quoteString(field)+", "+quoteString(parent)+"))\n}")
	}
	if validation != "" {
		lines = append(lines, "if actual, ok := "+target+".Value(); ok {\n"+indentCode(validation)+"}")
	}
	return strings.Join(lines, "\n")
}

func validateOptionalAttribute(
	ctx *AttributeContext,
	att *expr.AttributeExpr,
	put expr.UserType,
	target string,
	context string,
	view bool,
	seen map[string]*bytes.Buffer,
) string {
	underlying := concretePresenceAttribute(att)
	valueCtx := presenceValueContext(ctx, underlying)
	validation := recurseValidationCode(underlying, put, valueCtx, true, false, view, "actual", context, seen).String()
	if validation == "" {
		return ""
	}
	return "if actual, ok := " + target + ".Value(); ok {\n" + indentCode(validation) + "}"
}

func presenceValueContext(ctx *AttributeContext, underlying *expr.AttributeExpr) *AttributeContext {
	valueCtx := ctx.Dup()
	primitive := expr.IsPrimitive(underlying.Type)
	if primitive {
		valueCtx.Pointer = false
	}
	if userType, ok := underlying.Type.(expr.UserType); ok && !primitive {
		if jsonPresence, recorded := ctx.JSONPresenceTypes[userType.ID()]; recorded {
			valueCtx.JSONPresence = jsonPresence
		}
		if pointer, recorded := ctx.PresencePointerTypes[userType.ID()]; recorded {
			valueCtx.Pointer = pointer
		}
		if useDefault, recorded := ctx.PresenceUseDefaultTypes[userType.ID()]; recorded {
			valueCtx.UseDefault = useDefault
		}
	}
	return valueCtx
}

func concretePresenceAttribute(att *expr.AttributeExpr) *expr.AttributeExpr {
	underlying := *att
	underlying.Nullable = false
	userType, ok := att.Type.(expr.UserType)
	if !ok || !expr.IsNullable(userType.Attribute()) {
		return &underlying
	}
	concreteType := userType.Dup(nil)
	concreteAttribute := expr.DupAtt(userType.Attribute())
	concreteAttribute.Nullable = false
	concreteType.SetAttribute(concreteAttribute)
	underlying.Type = concreteType
	return &underlying
}

func nullableValidationContext(context string) (field, parent string) {
	index := strings.LastIndex(context, ".")
	if index < 0 {
		return context, "body"
	}
	return context[index+1:], context[:index]
}
