package codegen

import (
	"bytes"
	"errors"
	"fmt"
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

	newline := func() {
		if !first {
			buf.WriteByte('\n')
		} else {
			first = false
		}
	}

	// Write validations on attribute if any.
	validation := validationCode(att, attCtx, req, alias, target, context)
	if validation != "" {
		buf.WriteString(validation)
		first = false
	}

	// Recurse down depending on attribute type.
	switch {
	case expr.IsObject(att.Type):
		if isUT {
			put = ut
		}
		for _, nat := range *(expr.AsObject(att.Type)) {
			tgt := fmt.Sprintf("%s.%s", target, attCtx.Scope.Field(nat.Attribute, nat.Name, true))
			ctx := fmt.Sprintf("%s.%s", context, nat.Name)
			val := validateAttribute(attCtx, nat.Attribute, put, tgt, ctx, att.IsRequired(nat.Name), view, seen)
			if val != "" {
				newline()
				buf.WriteString(val)
			}
		}
	case expr.IsArray(att.Type):
		arr := expr.AsArray(att.Type)
		elem := arr.ElemType
		ctx := attCtx
		if ctx.Pointer && expr.IsPrimitive(elem.Type) {
			// Array elements of primitive type are never pointers
			ctx = attCtx.Dup()
			ctx.Pointer = false
		}
		val := validateAttribute(ctx, elem, put, "e", context+"[*]", true, view, seen)
		if val != "" || arr.NonNullableElems {
			newline()
			buf.WriteString(renderArrayValidation(target, val, arr.NonNullableElems, context))
		}
	case expr.IsMap(att.Type):
		m := expr.AsMap(att.Type)
		ctx := attCtx.Dup()
		ctx.Pointer = false
		keyVal := validateAttribute(ctx, m.KeyType, put, "k", context+".key", true, view, seen)
		if keyVal != "" {
			keyVal = "\n" + keyVal
		}
		valueVal := validateAttribute(ctx, m.ElemType, put, "v", context+"[key]", true, view, seen)
		if valueVal != "" {
			valueVal = "\n" + valueVal
		}
		if keyVal != "" || valueVal != "" {
			newline()
			buf.WriteString(renderMapValidation(target, keyVal, valueVal))
		}
	case expr.IsUnion(att.Type):
		u := expr.AsUnion(att.Type)
		if _, ok := attCtx.Scope.(*AttributeScope); ok {
			cases := make([]map[string]any, 0, len(u.Values))
			for _, v := range u.Values {
				// Sum-type unions (struct-based, with Kind/AsX accessors) store each
				// branch as either a value (primitives, arrays, maps) or a pointer
				// (object user types). The union validation template binds the branch
				// value to a local named "actual"; the pointer semantics must match
				// that local, not the enclosing field context.
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
			if len(cases) > 0 {
				newline()
				buf.WriteString(renderUnionSumValidation(target, cases))
			}
			break
		}

		// Validate unions represented as interfaces (e.g., protobuf oneof wrappers).
		var vals []string
		var types []string
		for _, v := range u.Values {
			vatt := v.Attribute
			if view {
				// Union values in views are never pointers - they are concrete typed values
				unionCtx := attCtx.Dup()
				unionCtx.Pointer = false
				val := validateAttribute(unionCtx, vatt, put, "v", context+".value", true, view, seen)
				if val != "" {
					types = append(types, attCtx.Scope.Ref(vatt, attCtx.DefaultPkg))
					vals = append(vals, val)
				}
			} else {
				fieldName := attCtx.Scope.Field(vatt, v.Name, true)
				val := validateAttribute(attCtx, vatt, put, "v."+fieldName, context+".value", true, view, seen)
				if val != "" {
					tref := attCtx.Scope.Ref(&expr.AttributeExpr{Type: put}, attCtx.DefaultPkg)
					types = append(types, tref+"_"+fieldName)
					vals = append(vals, val)
				}
			}
		}
		if len(vals) > 0 {
			newline()
			buf.WriteString(renderUnionValidation(target, types, vals))
		}
	}

	return buf
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
	buf.WriteString(renderUserValidation(name, target))
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
	validation := att.Validation
	if ut, ok := att.Type.(expr.UserType); ok {
		val := ut.Attribute().Validation
		if val != nil {
			if validation == nil {
				validation = val
			} else {
				validation.Merge(val)
			}
			att.Validation = validation
		}
	}
	if validation == nil {
		return ""
	}

	var (
		kind            = att.Type.Kind()
		unaliased       = unalias(att.Type)
		isNativePointer = unaliased.Kind() == expr.BytesKind || unaliased.Kind() == expr.AnyKind
		isPointer       = attCtx.Pointer || (!req && (att.DefaultValue == nil || !attCtx.UseDefault))
		tval            = target
	)
	if isPointer && expr.IsPrimitive(att.Type) && !isNativePointer {
		tval = "*" + tval
	}
	if alias {
		tval = fmt.Sprintf("%s(%s)", unaliased.Name(), tval)
		// When validating alias types, use the underlying type's kind
		// for string detection (needed for utf8.RuneCountInString usage)
		kind = unaliased.Kind()
	}
	data := map[string]any{
		"attribute": att,
		"attCtx":    attCtx,
		"isPointer": isPointer,
		"context":   context,
		"target":    target,
		"targetVal": tval,
		"string":    kind == expr.StringKind,
		"array":     expr.IsArray(att.Type),
		"map":       expr.IsMap(att.Type),
	}
	res := make([]string, 0, 8) // preallocate with typical validation count
	if values := validation.Values; values != nil {
		data["values"] = values
		if val := renderValidationTemplate("enum", data); val != "" {
			res = append(res, val)
		}
	}
	if format := validation.Format; format != "" {
		data["format"] = string(format)
		if val := renderValidationTemplate("format", data); val != "" {
			res = append(res, val)
		}
	}
	if pattern := validation.Pattern; pattern != "" {
		data["pattern"] = pattern
		if val := renderValidationTemplate("pattern", data); val != "" {
			res = append(res, val)
		}
	}
	if exclMin := validation.ExclusiveMinimum; exclMin != nil {
		data["exclMin"] = *exclMin
		data["isExclMin"] = true
		if val := renderValidationTemplate("exclMinMax", data); val != "" {
			res = append(res, val)
		}
	}
	if minVal := validation.Minimum; minVal != nil {
		data["min"] = *minVal
		data["isMin"] = true
		if val := renderValidationTemplate("minMax", data); val != "" {
			res = append(res, val)
		}
	}
	if exclMax := validation.ExclusiveMaximum; exclMax != nil {
		data["exclMax"] = *exclMax
		data["isExclMax"] = true
		if val := renderValidationTemplate("exclMinMax", data); val != "" {
			res = append(res, val)
		}
	}
	if maxVal := validation.Maximum; maxVal != nil {
		data["max"] = *maxVal
		data["isMin"] = false
		if val := renderValidationTemplate("minMax", data); val != "" {
			res = append(res, val)
		}
	}
	if minLength := validation.MinLength; minLength != nil {
		data["minLength"] = minLength
		data["isMinLength"] = true
		delete(data, "maxLength")
		if val := renderValidationTemplate("length", data); val != "" {
			res = append(res, val)
		}
	}
	if maxLength := validation.MaxLength; maxLength != nil {
		data["maxLength"] = maxLength
		data["isMinLength"] = false
		delete(data, "minLength")
		if val := renderValidationTemplate("length", data); val != "" {
			res = append(res, val)
		}
	}
	reqs := generatedRequiredValidation(att, attCtx)
	obj := expr.AsObject(att.Type)
	for _, r := range reqs {
		reqAtt := obj.Attribute(r)
		data["req"] = r
		data["reqAtt"] = reqAtt
		res = append(res, renderValidationTemplate("required", data))
	}
	return strings.Join(res, "\n")
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
	var b strings.Builder
	if data["isPointer"].(bool) {
		fmt.Fprintf(&b, "if %s != nil {\n", data["target"])
	}
	fmt.Fprintf(&b, "if !(%s) {\n", oneof(data["targetVal"].(string), data["values"].([]any)))
	fmt.Fprintf(&b, "\terr = goa.MergeErrors(err, goa.InvalidEnumValueError(%q, %s, %s))\n", data["context"], data["targetVal"], toSlice(data["values"].([]any)))
	fmt.Fprintf(&b, "}")
	if data["isPointer"].(bool) {
		fmt.Fprintf(&b, "\n}")
	}
	return strings.Trim(b.String(), "\n")
}

func renderFormatValidation(data map[string]any) string {
	return renderSimplePointerWrappedValidation(data["isPointer"].(bool), data["target"].(string),
		fmt.Sprintf("err = goa.MergeErrors(err, goa.ValidateFormat(%q, %s, %s))",
			data["context"], data["targetVal"], constant(data["format"].(string))))
}

func renderPatternValidation(data map[string]any) string {
	return renderSimplePointerWrappedValidation(data["isPointer"].(bool), data["target"].(string),
		fmt.Sprintf("err = goa.MergeErrors(err, goa.ValidatePattern(%q, %s, %q))",
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
	body := fmt.Sprintf("if %s %s %v {\n\terr = goa.MergeErrors(err, goa.InvalidRangeError(%q, %s, %v, %t))\n}",
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
	body := fmt.Sprintf("if %s %s %v {\n\terr = goa.MergeErrors(err, goa.InvalidRangeError(%q, %s, %v, %t))\n}",
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
	body := fmt.Sprintf("if %s %s %v {\n\terr = goa.MergeErrors(err, goa.InvalidLengthError(%q, %s, %s, %v, %t))\n}",
		lengthExpr, op, bound, data["context"], targetExpr, lengthExpr, bound, flag)
	return renderSimplePointerWrappedValidation(data["isPointer"].(bool) && data["string"].(bool), data["target"].(string), body)
}

func renderRequiredValidation(data map[string]any) string {
	reqAtt := data["reqAtt"].(*expr.AttributeExpr)
	field := data["attCtx"].(*AttributeContext).Scope.Field(reqAtt, data["req"].(string), true)
	if expr.IsUnion(reqAtt.Type) {
		if _, ok := data["attCtx"].(*AttributeContext).Scope.(*AttributeScope); ok {
			return fmt.Sprintf("if %s.%s.Kind() == \"\" {\n\terr = goa.MergeErrors(err, goa.MissingFieldError(%q, %q))\n}",
				data["target"], field, data["req"], data["context"])
		}
	}
	return fmt.Sprintf("if %s.%s == nil {\n\terr = goa.MergeErrors(err, goa.MissingFieldError(%q, %q))\n}",
		data["target"], field, data["req"], data["context"])
}

func renderArrayValidation(target, validation string, nonNullable bool, context string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "for _, e := range %s {\n", target)
	if nonNullable {
		fmt.Fprintf(&b, "\tif e == nil {\n")
		fmt.Fprintf(&b, "\t\terr = goa.MergeErrors(err, goa.MissingFieldError(%q, \"[*]\"))\n", context)
		fmt.Fprintf(&b, "\t}\n")
	}
	if validation != "" {
		b.WriteString(indentCode(validation, "\t"))
	}
	fmt.Fprintf(&b, "}")
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
	var b strings.Builder
	fmt.Fprintf(&b, "for %s, %s := range %s {\n", keyVar, valueVar, target)
	if keyValidation != "" {
		b.WriteString(indentCode(strings.TrimPrefix(keyValidation, "\n"), "\t"))
	}
	if valueValidation != "" {
		b.WriteString(indentCode(strings.TrimPrefix(valueValidation, "\n"), "\t"))
	}
	fmt.Fprintf(&b, "}")
	return b.String()
}

func renderUnionValidation(target string, types, values []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "switch v := %s.(type) {\n", target)
	for i, val := range values {
		fmt.Fprintf(&b, "case %s:\n", types[i])
		b.WriteString(indentCode(val, "\t"))
	}
	fmt.Fprintf(&b, "}")
	return b.String()
}

func renderUnionSumValidation(target string, cases []map[string]any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "switch string(%s.Kind()) {\n", target)
	for _, c := range cases {
		fmt.Fprintf(&b, "case %q:\n", c["typeTag"])
		fmt.Fprintf(&b, "\tactual, _ := %s.As%s()\n", target, c["fieldName"])
		if c["requiresValue"].(bool) {
			fmt.Fprintf(&b, "\tif actual == nil {\n")
			fmt.Fprintf(&b, "\t\terr = goa.MergeErrors(err, goa.MissingFieldError(\"value\", %q))\n", c["context"])
			fmt.Fprintf(&b, "\t\tbreak\n")
			fmt.Fprintf(&b, "\t}\n")
		}
		b.WriteString(indentCode(c["validation"].(string), "\t"))
	}
	fmt.Fprintf(&b, "}")
	return b.String()
}

func renderUserValidation(name, target string) string {
	return fmt.Sprintf("if err2 := Validate%s(%s); err2 != nil {\n\terr = goa.MergeErrors(err, err2)\n}", name, target)
}

func renderSimplePointerWrappedValidation(isPointer bool, target, body string) string {
	body = strings.Trim(body, "\n")
	if !isPointer {
		return body
	}
	return fmt.Sprintf("if %s != nil {\n%s}", target, indentCode(body, "\t"))
}

func indentCode(code, indent string) string {
	trimmed := strings.Trim(code, "\n")
	if trimmed == "" {
		return ""
	}
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
		return "goa.FormatDate"
	case "date-time":
		return "goa.FormatDateTime"
	case "uuid":
		return "goa.FormatUUID"
	case "email":
		return "goa.FormatEmail"
	case "hostname":
		return "goa.FormatHostname"
	case "ipv4":
		return "goa.FormatIPv4"
	case "ipv6":
		return "goa.FormatIPv6"
	case "ip":
		return "goa.FormatIP"
	case "uri":
		return "goa.FormatURI"
	case "mac":
		return "goa.FormatMAC"
	case "cidr":
		return "goa.FormatCIDR"
	case "regexp":
		return "goa.FormatRegexp"
	case "json":
		return "goa.FormatJSON"
	case "rfc1123":
		return "goa.FormatRFC1123"
	}
	panic("unknown format") // bug
}
