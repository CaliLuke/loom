//nolint:errcheck // Generator helpers write only to in-memory buffers/builders.
package codegen

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

// AttributeValidationCode produces Go code that runs the validations defined
// in the given attribute against the value held by the variable named target.
//
// See ValidationCode for a description of the arguments.
func AttributeValidationCode(att *expr.AttributeExpr, put expr.UserType, attCtx *AttributeContext, req, alias bool, target, attName string) string {
	return recurseValidationCode(att, put, attCtx, req, alias, false, target, attName, nil).String()
}

// ValidationCode produces Go code that runs the validations defined in the
// given attribute and its children recursively against the value held by the
// variable named target.
//
// put is the parent UserType if any. It is used to compute proto oneof type names.
//
// attCtx is the attribute context used to generate attribute name and reference
// in the validation code.
//
// req indicates whether the attribute is required (true) or optional (false)
//
// alias indicates whether the attribute is an alias user type attribute.
//
// view indicates whether the attribute is a view type attribute.
// This only matters for union types: generated Loom view union types have a
// different layout than proto generated union types.
//
// target is the variable name against which the validation code is generated
//
// context is used to produce helpful messages in case of error.
func ValidationCode(att *expr.AttributeExpr, put expr.UserType, attCtx *AttributeContext, req, alias, view bool, target string) string {
	return recurseValidationCode(att, put, attCtx, req, alias, view, target, target, nil).String()
}

func recurseValidationCode(att *expr.AttributeExpr, put expr.UserType, attCtx *AttributeContext, req, alias, view bool, target, context string, seen map[string]*bytes.Buffer) *bytes.Buffer {
	if seen == nil {
		seen = make(map[string]*bytes.Buffer)
	}
	var (
		buf      = new(bytes.Buffer)
		first    = true
		ut, isUT = att.Type.(expr.UserType)
	)

	// Break infinite recursions
	// Note: when alias=true, we're validating the underlying base type,
	// so alias types shouldn't use the recursion guard. Only non-alias user
	// types need cycle protection.
	if isUT && !alias {
		if buf, ok := seen[ut.ID()]; ok {
			return buf
		}
		seen[ut.ID()] = buf
	}

	flattenValidations(att, make(map[string]struct{}))

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
	for _, nat := range *(expr.AsObject(att.Type)) {
		tgt := fmt.Sprintf("%s.%s", target, attCtx.Scope.Field(nat.Attribute, nat.Name, true))
		ctx := fmt.Sprintf("%s.%s", context, nat.Name)
		val := validateAttribute(attCtx, nat.Attribute, put, tgt, ctx, att.IsRequired(nat.Name), view, seen)
		appendValidationBlock(buf, first, val)
	}
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
	ctx := attCtx.Dup()
	ctx.Pointer = false
	keyVal := prefixedValidation(validateAttribute(ctx, m.KeyType, put, "k", context+".key", true, view, seen))
	valueVal := prefixedValidation(validateAttribute(ctx, m.ElemType, put, "v", context+"[key]", true, view, seen))
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

func renderUnionSumValidationCases(u *expr.Union, put expr.UserType, attCtx *AttributeContext, view bool, context string, seen map[string]*bytes.Buffer) []map[string]any {
	cases := make([]map[string]any, 0, len(u.Values))
	for _, v := range u.Values {
		unionCtx := attCtx.Dup()
		unionCtx.Pointer = expr.IsObject(v.Attribute.Type)
		val := validateAttribute(unionCtx, v.Attribute, put, "actual", context+".value", true, view, seen)
		if val == "" {
			continue
		}
		cases = append(cases, map[string]any{
			"typeTag":       expr.UnionVariantTag(v),
			"fieldName":     Goify(v.Name, true),
			"requiresValue": strings.HasPrefix(strings.TrimSpace(val), "if actual != nil {"),
			"context":       context + ".value",
			"validation":    val,
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
		cond := fmt.Sprintf("if %s != nil {\n", target)
		if strings.HasPrefix(code, cond) {
			return code
		}
		return fmt.Sprintf("%s%s\n}", cond, code)
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
		cond := fmt.Sprintf("if %s != nil {\n", target)
		if strings.HasPrefix(code, cond) {
			return code
		}
		return fmt.Sprintf("%s%s\n}", cond, code)
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
	return fmt.Sprintf("if %s != nil {\n\t%s\n}", target, buf.String())
}

// validationCode produces Go code that runs the validations defined in the
// given attribute definition if any against the content of the variable named
// target. The generated code assumes that there is a pre-existing "err"
// variable of type error. It initializes that variable in case a validation
// fails.
//
// attCtx is the attribute context
//
// req indicates whether the attribute is required (true) or optional (false)
//
// alias indicates whether the attribute is an alias user type attribute.
//
// view indicates whether the attribute is a view type attribute.
// This only matters for union types: generated Loom view union types have a
// different layout than proto generated union types.
//
// target is the variable name against which the validation code is generated
//
// context is used to produce helpful messages in case of error.
func validationCode(att *expr.AttributeExpr, attCtx *AttributeContext, req, alias bool, target, context string) string {
	validation := mergedValidation(att)
	if validation == nil {
		return ""
	}

	data := validationTemplateData(att, attCtx, req, alias, target, context)
	res := make([]string, 0, 8) // preallocate with typical validation count
	if values := validation.Values; values != nil {
		data["values"] = values
		appendRenderedValidation(&res, "enum", data)
	}
	appendValidationString(&res, string(validation.Format), data, "format", "format")
	appendValidationString(&res, validation.Pattern, data, "pattern", "pattern")
	appendValidationNumber(&res, validation.ExclusiveMinimum, data, "exclMin", "isExclMin", true, "exclMinMax")
	appendValidationNumber(&res, validation.Minimum, data, "min", "isMin", true, "minMax")
	appendValidationNumber(&res, validation.ExclusiveMaximum, data, "exclMax", "isExclMax", true, "exclMinMax")
	appendValidationNumber(&res, validation.Maximum, data, "max", "isMin", false, "minMax")
	appendValidationLength(&res, validation.MinLength, data, "minLength", "maxLength", true)
	appendValidationLength(&res, validation.MaxLength, data, "maxLength", "minLength", false)
	appendRequiredValidations(&res, att, attCtx, data)
	return strings.Join(res, "\n")
}

func mergedValidation(att *expr.AttributeExpr) *expr.ValidationExpr {
	validation := att.Validation
	if ut, ok := att.Type.(expr.UserType); ok {
		val := ut.Attribute().Validation
		if val == nil {
			return validation
		}
		if validation == nil {
			validation = val
		} else {
			validation.Merge(val)
		}
		att.Validation = validation
	}
	return validation
}

func validationTemplateData(att *expr.AttributeExpr, attCtx *AttributeContext, req, alias bool, target, context string) map[string]any {
	kind := att.Type.Kind()
	unaliased := unalias(att.Type)
	isNativePointer := unaliased.Kind() == expr.BytesKind || unaliased.Kind() == expr.AnyKind
	isPointer := attCtx.Pointer || (!req && (att.DefaultValue == nil || !attCtx.UseDefault))
	targetVal := target
	if isPointer && expr.IsPrimitive(att.Type) && !isNativePointer {
		targetVal = "*" + targetVal
	}
	if alias {
		targetVal = fmt.Sprintf("%s(%s)", unaliased.Name(), targetVal)
		kind = unaliased.Kind()
	}
	return map[string]any{
		"attribute": att,
		"attCtx":    attCtx,
		"isPointer": isPointer,
		"context":   context,
		"target":    target,
		"targetVal": targetVal,
		"string":    kind == expr.StringKind,
		"array":     expr.IsArray(att.Type),
		"map":       expr.IsMap(att.Type),
	}
}

func appendValidationString(res *[]string, value string, data map[string]any, key, template string) {
	if value == "" {
		return
	}
	data[key] = value
	appendRenderedValidation(res, template, data)
}

func appendValidationNumber(res *[]string, value any, data map[string]any, valueKey, flagKey string, flagValue bool, template string) {
	if value == nil {
		return
	}
	v := reflect.ValueOf(value)
	if !v.IsValid() || (v.Kind() == reflect.Ptr && v.IsNil()) {
		return
	}
	data[valueKey] = v.Elem().Interface()
	data[flagKey] = flagValue
	appendRenderedValidation(res, template, data)
}

func appendValidationLength(res *[]string, value *int, data map[string]any, key, otherKey string, isMin bool) {
	if value == nil {
		return
	}
	data[key] = value
	data["isMinLength"] = isMin
	delete(data, otherKey)
	appendRenderedValidation(res, "length", data)
}

func appendRequiredValidations(res *[]string, att *expr.AttributeExpr, attCtx *AttributeContext, data map[string]any) {
	obj := expr.AsObject(att.Type)
	for _, r := range generatedRequiredValidation(att, attCtx) {
		data["req"] = r
		data["reqAtt"] = obj.Attribute(r)
		*res = append(*res, renderValidationTemplate("required", data))
	}
}

func appendRenderedValidation(res *[]string, template string, data map[string]any) {
	if val := renderValidationTemplate(template, data); val != "" {
		*res = append(*res, val)
	}
}

func renderValidationTemplate(kind string, data map[string]any) string {
	switch kind {
	case "enum":
		return renderEnumValidation(data)
	case "format":
		return renderFormatValidation(data)
	case "pattern":
		return renderPatternValidation(data)
	case "exclMinMax":
		return renderExclMinMaxValidation(data)
	case "minMax":
		return renderMinMaxValidation(data)
	case "length":
		return renderLengthValidation(data)
	case "required":
		return renderRequiredValidation(data)
	default:
		panic("unknown validation template kind") // bug
	}
}

func renderEnumValidation(data map[string]any) string {
	var b sourceBuilder
	if data["isPointer"].(bool) {
		b.Add(fmt.Sprintf("if %s != nil {\n", data["target"]))
	}
	b.Add(fmt.Sprintf("if !(%s) {\n", oneof(data["targetVal"].(string), data["values"].([]any))))
	b.Add(fmt.Sprintf("\terr = loom.MergeErrors(err, loom.InvalidEnumValueError(%q, %s, %s))\n", data["context"], data["targetVal"], toSlice(data["values"].([]any))))
	b.Add("}")
	if data["isPointer"].(bool) {
		b.Add("\n}")
	}
	return strings.Trim(b.String(), "\n")
}

func renderFormatValidation(data map[string]any) string {
	return renderSimplePointerWrappedValidation(data["isPointer"].(bool), data["target"].(string),
		fmt.Sprintf("err = loom.MergeErrors(err, loom.ValidateFormat(%q, %s, %s))",
			data["context"], data["targetVal"], constant(data["format"].(string))))
}

func renderPatternValidation(data map[string]any) string {
	return renderSimplePointerWrappedValidation(data["isPointer"].(bool), data["target"].(string),
		fmt.Sprintf("err = loom.MergeErrors(err, loom.ValidatePattern(%q, %s, %q))",
			data["context"], data["targetVal"], data["pattern"]))
}

func renderExclMinMaxValidation(data map[string]any) string {
	var (
		op    string
		bound any
		flag  bool
	)
	if data["isExclMin"] == true {
		op = "<="
		bound = data["exclMin"]
		flag = true
	} else {
		op = ">="
		bound = data["exclMax"]
		flag = false
	}
	body := fmt.Sprintf("if %s %s %v {\n\terr = loom.MergeErrors(err, loom.InvalidRangeError(%q, %s, %v, %t))\n}",
		data["targetVal"], op, bound, data["context"], data["targetVal"], bound, flag)
	return renderSimplePointerWrappedValidation(data["isPointer"].(bool), data["target"].(string), body)
}

func renderMinMaxValidation(data map[string]any) string {
	var (
		op    string
		bound any
		flag  bool
	)
	if data["isMin"] == true {
		op = "<"
		bound = data["min"]
		flag = true
	} else {
		op = ">"
		bound = data["max"]
		flag = false
	}
	body := fmt.Sprintf("if %s %s %v {\n\terr = loom.MergeErrors(err, loom.InvalidRangeError(%q, %s, %v, %t))\n}",
		data["targetVal"], op, bound, data["context"], data["targetVal"], bound, flag)
	return renderSimplePointerWrappedValidation(data["isPointer"].(bool), data["target"].(string), body)
}

func renderLengthValidation(data map[string]any) string {
	targetExpr := data["targetVal"].(string)
	if ((data["array"] == true || data["map"] == true) || data["nonzero"] == true) && data["target"] != nil {
		targetExpr = data["target"].(string)
	}
	lengthExpr := fmt.Sprintf("len(%s)", targetExpr)
	if data["string"].(bool) {
		lengthExpr = fmt.Sprintf("utf8.RuneCountInString(%s)", targetExpr)
	}
	var (
		op    string
		bound int
		flag  bool
	)
	if data["isMinLength"] == true {
		op = "<"
		bound = *data["minLength"].(*int)
		flag = true
	} else {
		op = ">"
		bound = *data["maxLength"].(*int)
		flag = false
	}
	body := fmt.Sprintf("if %s %s %v {\n\terr = loom.MergeErrors(err, loom.InvalidLengthError(%q, %s, %s, %v, %t))\n}",
		lengthExpr, op, bound, data["context"], targetExpr, lengthExpr, bound, flag)
	return renderSimplePointerWrappedValidation(data["isPointer"].(bool) && data["string"].(bool), data["target"].(string), body)
}

func renderRequiredValidation(data map[string]any) string {
	reqAtt := data["reqAtt"].(*expr.AttributeExpr)
	field := data["attCtx"].(*AttributeContext).Scope.Field(reqAtt, data["req"].(string), true)
	if expr.IsUnion(reqAtt.Type) {
		if _, ok := data["attCtx"].(*AttributeContext).Scope.(*AttributeScope); ok {
			return fmt.Sprintf("if %s.%s.Kind() == \"\" {\n\terr = loom.MergeErrors(err, loom.MissingFieldError(%q, %q))\n}",
				data["target"], field, data["req"], data["context"])
		}
	}
	return fmt.Sprintf("if %s.%s == nil {\n\terr = loom.MergeErrors(err, loom.MissingFieldError(%q, %q))\n}",
		data["target"], field, data["req"], data["context"])
}

func renderArrayValidation(target, validation string, nonNullable bool, context string) string {
	var b sourceBuilder
	b.Add(fmt.Sprintf("for _, e := range %s {\n", target))
	if nonNullable {
		b.Add("\tif e == nil {\n")
		b.Add(fmt.Sprintf("\t\terr = loom.MergeErrors(err, loom.MissingFieldError(%q, \"[*]\"))\n", context))
		b.Add("\t}\n")
	}
	if validation != "" {
		b.Add(indentCode(validation))
	}
	b.Add("}")
	return b.String()
}

func renderMapValidation(target, keyValidation, valueValidation string) string {
	keyVar := "_"
	if keyValidation != "" {
		keyVar = "k"
	}
	valueVar := "_"
	if valueValidation != "" {
		valueVar = "v"
	}
	var b sourceBuilder
	fmt.Fprintf(&b, "for %s, %s := range %s {\n", keyVar, valueVar, target)
	if keyValidation != "" {
		b.Add(indentCode(strings.TrimPrefix(keyValidation, "\n")))
	}
	if valueValidation != "" {
		b.Add(indentCode(strings.TrimPrefix(valueValidation, "\n")))
	}
	b.Add("}")
	return b.String()
}

func renderUnionValidation(target string, types, values []string) string {
	var b sourceBuilder
	fmt.Fprintf(&b, "switch v := %s.(type) {\n", target)
	for i, val := range values {
		fmt.Fprintf(&b, "case %s:\n", types[i])
		b.Add(indentCode(val))
	}
	fmt.Fprintf(&b, "}")
	return b.String()
}

func renderUnionSumValidation(target string, cases []map[string]any) string {
	var b sourceBuilder
	fmt.Fprintf(&b, "switch string(%s.Kind()) {\n", target)
	for _, c := range cases {
		fmt.Fprintf(&b, "case %q:\n", c["typeTag"])
		fmt.Fprintf(&b, "\tactual, _ := %s.As%s()\n", target, c["fieldName"])
		if c["requiresValue"].(bool) {
			fmt.Fprintf(&b, "\tif actual == nil {\n")
			fmt.Fprintf(&b, "\t\terr = loom.MergeErrors(err, loom.MissingFieldError(\"value\", %q))\n", c["context"])
			fmt.Fprintf(&b, "\t\tbreak\n")
			fmt.Fprintf(&b, "\t}\n")
		}
		b.Add(indentCode(c["validation"].(string)))
	}
	fmt.Fprintf(&b, "}")
	return b.String()
}

func renderUserValidation(name, target string) string {
	return fmt.Sprintf("if err2 := Validate%s(%s); err2 != nil {\n\terr = loom.MergeErrors(err, err2)\n}", name, target)
}

func renderSimplePointerWrappedValidation(isPointer bool, target, body string) string {
	body = strings.Trim(body, "\n")
	if !isPointer {
		return body
	}
	return fmt.Sprintf("if %s != nil {\n%s}", target, indentCode(body))
}

func indentCode(code string) string {
	trimmed := strings.Trim(code, "\n")
	if trimmed == "" {
		return ""
	}
	const indent = "\t"
	return indent + strings.ReplaceAll(trimmed, "\n", "\n"+indent) + "\n"
}

// hasValidations returns true if a UserType contains validations.
func hasValidations(attCtx *AttributeContext, ut expr.UserType) bool {
	// We need to check empirically whether there are validations to be
	// generated, we can't just generate and check whether something was
	// generated to avoid infinite recursions.
	res := false
	done := errors.New("done")
	Walk(ut.Attribute(), func(a *expr.AttributeExpr) error { // nolint: errcheck
		if a.Validation == nil {
			return nil
		}
		if attCtx.Pointer || !a.Validation.HasRequiredOnly() {
			res = true
			return done
		}
		res = len(generatedRequiredValidation(a, attCtx)) > 0
		if res {
			return done
		}
		return nil
	})
	return res
}

// There is a case where there is validation but no actual validation code: if
// the validation is a required validation that applies to attributes that
// cannot be nil i.e. primitive types.
func generatedRequiredValidation(att *expr.AttributeExpr, attCtx *AttributeContext) (res []string) {
	if att.Validation == nil {
		return
	}
	obj := expr.AsObject(att.Type)
	for _, req := range att.Validation.Required {
		reqAtt := obj.Attribute(req)
		if reqAtt == nil {
			continue
		}
		if !attCtx.Pointer && expr.IsPrimitive(reqAtt.Type) &&
			reqAtt.Type.Kind() != expr.BytesKind &&
			reqAtt.Type.Kind() != expr.AnyKind {
			continue
		}
		if attCtx.IgnoreRequired && expr.IsPrimitive(reqAtt.Type) {
			continue
		}
		res = append(res, req)
	}
	return
}

func flattenValidations(att *expr.AttributeExpr, seen map[string]struct{}) {
	switch actual := att.Type.(type) {
	case *expr.Array:
		flattenValidations(actual.ElemType, seen)
	case *expr.Map:
		flattenValidations(actual.KeyType, seen)
		flattenValidations(actual.ElemType, seen)
	case *expr.Object:
		for _, nat := range *actual {
			flattenValidations(nat.Attribute, seen)
		}
	case *expr.Union:
		for _, nat := range actual.Values {
			flattenValidations(nat.Attribute, seen)
		}
	case expr.UserType:
		if _, ok := seen[actual.ID()]; ok {
			return
		}
		seen[actual.ID()] = struct{}{}
		v := att.Validation
		ut, ok := actual.Attribute().Type.(expr.UserType)
		for ok {
			if val := ut.Attribute().Validation; val != nil {
				if v == nil {
					v = val
				} else {
					v.Merge(val)
				}
			}
			ut, ok = ut.Attribute().Type.(expr.UserType)
		}
		att.Validation = v
		flattenValidations(actual.Attribute(), seen)
	}
}

// toSlice returns Go code that represents the given slice.
func toSlice(val []any) string {
	elems := make([]string, len(val))
	for i, v := range val {
		elems[i] = fmt.Sprintf("%#v", v)
	}
	return fmt.Sprintf("[]any{%s}", strings.Join(elems, ", "))
}

// oneof produces code that compares target with each element of vals and ORs
// the result, e.g. "target == 1 || target == 2".
func oneof(target string, vals []any) string {
	elems := make([]string, len(vals))
	for i, v := range vals {
		elems[i] = fmt.Sprintf("%s == %#v", target, v)
	}
	return strings.Join(elems, " || ")
}

// constant returns the Go constant name of the format with the given value.
func constant(formatName string) string {
	switch formatName {
	case "date":
		return "loom.FormatDate"
	case "date-time":
		return "loom.FormatDateTime"
	case "uuid":
		return "loom.FormatUUID"
	case "email":
		return "loom.FormatEmail"
	case "hostname":
		return "loom.FormatHostname"
	case "ipv4":
		return "loom.FormatIPv4"
	case "ipv6":
		return "loom.FormatIPv6"
	case "ip":
		return "loom.FormatIP"
	case "uri":
		return "loom.FormatURI"
	case "mac":
		return "loom.FormatMAC"
	case "cidr":
		return "loom.FormatCIDR"
	case "regexp":
		return "loom.FormatRegexp"
	case "json":
		return "loom.FormatJSON"
	case "rfc1123":
		return "loom.FormatRFC1123"
	}
	panic("unknown format") // bug
}
