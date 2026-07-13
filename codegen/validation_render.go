//nolint:errcheck // Generator helpers write only to in-memory buffers/builders.
package codegen

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

type (
	validationRenderData struct {
		Attribute    *expr.AttributeExpr
		AttributeCtx *AttributeContext
		IsPointer    bool
		Context      string
		Target       string
		TargetValue  string
		IsString     bool
		IsArray      bool
		IsMap        bool
		Values       []any
		Format       string
		Pattern      string
		Number       any
		NumberFlag   bool
		MinLength    *int
		MaxLength    *int
		IsMinLength  bool
		RequiredName string
		RequiredAttr *expr.AttributeExpr
	}

	unionValidationCase struct {
		TypeTag       string
		FieldName     string
		RequiresValue bool
		Context       string
		Validation    string
	}
)

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

	data := newValidationRenderData(att, attCtx, req, alias, target, context)
	res := make([]string, 0, 8) // preallocate with typical validation count
	if values := validation.Values; values != nil {
		data.Values = values
		appendRenderedValidation(&res, renderEnumValidation(data))
	}
	appendValidationString(&res, string(validation.Format), func(v string) string {
		data.Format = v
		return renderFormatValidation(data)
	})
	appendValidationString(&res, validation.Pattern, func(v string) string {
		data.Pattern = v
		return renderPatternValidation(data)
	})
	appendValidationNumber(&res, validation.ExclusiveMinimum, func(v any) string {
		data.Number = v
		data.NumberFlag = true
		return renderExclMinMaxValidation(data)
	})
	appendValidationNumber(&res, validation.Minimum, func(v any) string {
		data.Number = v
		data.NumberFlag = true
		return renderMinMaxValidation(data)
	})
	appendValidationNumber(&res, validation.ExclusiveMaximum, func(v any) string {
		data.Number = v
		data.NumberFlag = false
		return renderExclMinMaxValidation(data)
	})
	appendValidationNumber(&res, validation.Maximum, func(v any) string {
		data.Number = v
		data.NumberFlag = false
		return renderMinMaxValidation(data)
	})
	appendValidationLength(&res, validation.MinLength, func(v *int) string {
		data.MinLength = v
		data.MaxLength = nil
		data.IsMinLength = true
		return renderLengthValidation(data)
	})
	appendValidationLength(&res, validation.MaxLength, func(v *int) string {
		data.MaxLength = v
		data.MinLength = nil
		data.IsMinLength = false
		return renderLengthValidation(data)
	})
	appendRequiredValidations(&res, att, validation, attCtx, data)
	return strings.Join(res, "\n")
}

// mergedValidation returns the validation that applies to att, accumulating the
// validations declared at every level of the user-type chain rooted at att's
// type. Outer levels take precedence over inner ones (outer values win and
// tighter numeric bounds are kept), matching Merge semantics. Shared expr
// validation state is never mutated: each level is merged into a fresh copy
// (dup-before-merge), so the returned value is always safe to discard.
func mergedValidation(att *expr.AttributeExpr) *expr.ValidationExpr {
	validation := att.Validation
	ut, ok := att.Type.(expr.UserType)
	for ok {
		if val := ut.Attribute().Validation; val != nil {
			if validation == nil {
				validation = val.Dup()
			} else {
				validation = validation.Dup()
				validation.Merge(val)
			}
		}
		ut, ok = ut.Attribute().Type.(expr.UserType)
	}
	return validation
}

func newValidationRenderData(att *expr.AttributeExpr, attCtx *AttributeContext, req, alias bool, target, context string) validationRenderData {
	kind := att.Type.Kind()
	unaliased := unalias(att.Type)
	isNativePointer := unaliased.Kind() == expr.BytesKind || unaliased.Kind() == expr.AnyKind
	isPointer := attCtx.Pointer || (!req && (att.DefaultValue == nil || !attCtx.UseDefault))
	targetValue := target
	if isPointer && expr.IsPrimitive(att.Type) && !isNativePointer {
		targetValue = "*" + targetValue
	}
	if alias {
		targetValue = unaliased.Name() + "(" + targetValue + ")"
		kind = unaliased.Kind()
	}
	return validationRenderData{
		Attribute:    att,
		AttributeCtx: attCtx,
		IsPointer:    isPointer,
		Context:      context,
		Target:       target,
		TargetValue:  targetValue,
		IsString:     kind == expr.StringKind,
		IsArray:      expr.IsArray(att.Type),
		IsMap:        expr.IsMap(att.Type),
	}
}

func appendValidationString(res *[]string, value string, render func(string) string) {
	if value == "" {
		return
	}
	appendRenderedValidation(res, render(value))
}

func appendValidationNumber(res *[]string, value any, render func(any) string) {
	if value == nil {
		return
	}
	v := reflect.ValueOf(value)
	if !v.IsValid() || (v.Kind() == reflect.Pointer && v.IsNil()) {
		return
	}
	appendRenderedValidation(res, render(v.Elem().Interface()))
}

func appendValidationLength(res *[]string, value *int, render func(*int) string) {
	if value == nil {
		return
	}
	appendRenderedValidation(res, render(value))
}

func appendRequiredValidations(res *[]string, att *expr.AttributeExpr, validation *expr.ValidationExpr, attCtx *AttributeContext, data validationRenderData) {
	obj := expr.AsObject(att.Type)
	for _, r := range generatedRequiredValidationFrom(att, validation, attCtx) {
		data.RequiredName = r
		data.RequiredAttr = obj.Attribute(r)
		appendRenderedValidation(res, renderRequiredValidation(data))
	}
}

func appendRenderedValidation(res *[]string, validation string) {
	if validation != "" {
		*res = append(*res, validation)
	}
}

func renderEnumValidation(data validationRenderData) string {
	var b sourceBuilder
	if data.IsPointer {
		b.Add("if " + data.Target + " != nil {\n")
	}
	b.Add("if !(" + oneof(data.TargetValue, data.Values) + ") {\n")
	b.Add("\terr = loom.MergeErrors(err, loom.InvalidEnumValueError(" + quoteString(data.Context) + ", " + data.TargetValue + ", " + toSlice(data.Values) + "))\n")
	b.Add("}")
	if data.IsPointer {
		b.Add("\n}")
	}
	return strings.Trim(b.String(), "\n")
}

func renderFormatValidation(data validationRenderData) string {
	return renderSimplePointerWrappedValidation(data.IsPointer, data.Target,
		"err = loom.MergeErrors(err, loom.ValidateFormat("+quoteString(data.Context)+", "+data.TargetValue+", "+constant(data.Format, data.Attribute)+"))")
}

func renderPatternValidation(data validationRenderData) string {
	return renderSimplePointerWrappedValidation(data.IsPointer, data.Target,
		"err = loom.MergeErrors(err, loom.ValidatePattern("+quoteString(data.Context)+", "+data.TargetValue+", "+quoteString(data.Pattern)+"))")
}

func renderExclMinMaxValidation(data validationRenderData) string {
	var (
		op    string
		bound any
		flag  bool
	)
	if data.NumberFlag {
		op = "<="
		bound = data.Number
		flag = true
	} else {
		op = ">="
		bound = data.Number
		flag = false
	}
	body := "if " + data.TargetValue + " " + op + " " + validationGoLiteral(bound) + " {\n\terr = loom.MergeErrors(err, loom.InvalidRangeError(" + quoteString(data.Context) + ", " + data.TargetValue + ", " + validationGoLiteral(bound) + ", " + validationGoLiteral(flag) + "))\n}"
	return renderSimplePointerWrappedValidation(data.IsPointer, data.Target, body)
}

func renderMinMaxValidation(data validationRenderData) string {
	var (
		op    string
		bound any
		flag  bool
	)
	if data.NumberFlag {
		op = "<"
		bound = data.Number
		flag = true
	} else {
		op = ">"
		bound = data.Number
		flag = false
	}
	body := "if " + data.TargetValue + " " + op + " " + validationGoLiteral(bound) + " {\n\terr = loom.MergeErrors(err, loom.InvalidRangeError(" + quoteString(data.Context) + ", " + data.TargetValue + ", " + validationGoLiteral(bound) + ", " + validationGoLiteral(flag) + "))\n}"
	return renderSimplePointerWrappedValidation(data.IsPointer, data.Target, body)
}

func renderLengthValidation(data validationRenderData) string {
	targetExpr := data.TargetValue
	if (data.IsArray || data.IsMap) && data.Target != "" {
		targetExpr = data.Target
	}
	lengthExpr := "len(" + targetExpr + ")"
	if data.IsString {
		lengthExpr = "utf8.RuneCountInString(" + targetExpr + ")"
	}
	var (
		op    string
		bound int
		flag  bool
	)
	if data.IsMinLength {
		op = "<"
		bound = *data.MinLength
		flag = true
	} else {
		op = ">"
		bound = *data.MaxLength
		flag = false
	}
	body := "if " + lengthExpr + " " + op + " " + validationGoLiteral(bound) + " {\n\terr = loom.MergeErrors(err, loom.InvalidLengthError(" + quoteString(data.Context) + ", " + targetExpr + ", " + lengthExpr + ", " + validationGoLiteral(bound) + ", " + validationGoLiteral(flag) + "))\n}"
	return renderSimplePointerWrappedValidation(data.IsPointer && data.IsString, data.Target, body)
}

func renderRequiredValidation(data validationRenderData) string {
	field := data.AttributeCtx.Scope.Field(data.RequiredAttr, data.RequiredName, true)
	if expr.IsUnion(data.RequiredAttr.Type) {
		if _, ok := data.AttributeCtx.Scope.(*AttributeScope); ok {
			return "if " + data.Target + "." + field + ".Kind() == \"\" {\n\terr = loom.MergeErrors(err, loom.MissingFieldError(" + quoteString(data.RequiredName) + ", " + quoteString(data.Context) + "))\n}"
		}
	}
	return "if " + data.Target + "." + field + " == nil {\n\terr = loom.MergeErrors(err, loom.MissingFieldError(" + quoteString(data.RequiredName) + ", " + quoteString(data.Context) + "))\n}"
}

func renderArrayValidation(target, validation string, nonNullable bool, context string) string {
	var b sourceBuilder
	b.Add("for _, e := range " + target + " {\n")
	if nonNullable {
		b.Add("\tif e == nil {\n")
		b.Add("\t\terr = loom.MergeErrors(err, loom.MissingFieldError(" + quoteString(context) + ", \"[*]\"))\n")
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

func renderUnionSumValidation(target string, cases []unionValidationCase) string {
	var b sourceBuilder
	fmt.Fprintf(&b, "switch string(%s.Kind()) {\n", target)
	for _, c := range cases {
		fmt.Fprintf(&b, "case %q:\n", c.TypeTag)
		fmt.Fprintf(&b, "\tactual, _ := %s.As%s()\n", target, c.FieldName)
		if c.RequiresValue {
			fmt.Fprintf(&b, "\tif actual == nil {\n")
			fmt.Fprintf(&b, "\t\terr = loom.MergeErrors(err, loom.MissingFieldError(\"value\", %q))\n", c.Context)
			fmt.Fprintf(&b, "\t\tbreak\n")
			fmt.Fprintf(&b, "\t}\n")
		}
		b.Add(indentCode(c.Validation))
	}
	fmt.Fprintf(&b, "}")
	return b.String()
}

func renderUserValidation(name, target string) string {
	return "if err2 := Validate" + name + "(" + target + "); err2 != nil {\n\terr = loom.MergeErrors(err, err2)\n}"
}
