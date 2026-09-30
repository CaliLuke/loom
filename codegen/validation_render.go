//nolint:errcheck // Generator helpers write only to in-memory buffers/builders.
package codegen

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/enumvalue"
)

type (
	validationRenderSnapshot struct {
		rules   *expr.ValidationExpr
		clauses []expr.EffectiveValidationClause
	}

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
	constraints, err := expr.EffectiveConstraintsFor(att)
	if err != nil {
		panic(fmt.Sprintf("invalid effective constraints during generation: %v", err))
	}
	_, hasDefault := constraints.Default()
	effective := constraints.Validation()
	snapshot := validationRenderSnapshot{rules: effective.Lowered(), clauses: effective.Clauses()}
	return validationCodeFromSnapshot(att, snapshot, attCtx, req, alias,
		hasDefault, true, target, context)
}

func validationCodeFromSnapshot(
	att *expr.AttributeExpr,
	snapshot validationRenderSnapshot,
	attCtx *AttributeContext,
	req, alias, hasDefault, pointerGuard bool,
	target, context string,
) string {
	validation := snapshot.rules
	if validation == nil {
		return ""
	}
	if validation.HasRequiredOnly() && len(validation.Required) == 0 {
		return ""
	}

	data := newValidationRenderData(att, attCtx, req, alias, hasDefault, target, context)
	if !pointerGuard {
		data.IsPointer = false
	}
	res := make([]string, 0, 8) // preallocate with typical validation count
	for _, values := range validation.Enums() {
		data.Values = values
		appendRenderedValidation(&res, renderEnumValidation(data))
	}
	for _, clause := range snapshot.clauses {
		switch clause.Kind {
		case expr.EffectivePatternClause:
			data.Pattern = clause.Value
			appendRenderedValidation(&res, renderPatternValidation(data))
		case expr.EffectiveFormatClause:
			data.Format = clause.Value
			appendRenderedValidation(&res, renderFormatValidation(data))
		}
	}
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

func validationWithoutExactPredicates(current, enforced expr.EffectiveValidation) validationRenderSnapshot {
	currentRules := current.Lowered()
	enforcedRules := enforced.Lowered()
	result := currentRules.Dup()
	result.Values = nil
	result.EnumClauses = nil
	for _, clause := range currentRules.Enums() {
		if !slices.ContainsFunc(enforcedRules.Enums(), func(existing []any) bool {
			return reflect.DeepEqual(existing, clause)
		}) {
			result.EnumClauses = append(result.EnumClauses, clause)
		}
	}
	result.Pattern = ""
	result.PatternClauses = nil
	for _, pattern := range currentRules.Patterns() {
		if !slices.Contains(enforcedRules.Patterns(), pattern) {
			result.PatternClauses = append(result.PatternClauses, pattern)
		}
	}
	result.Format = ""
	result.FormatClauses = nil
	for _, format := range currentRules.Formats() {
		if !slices.Contains(enforcedRules.Formats(), format) {
			result.FormatClauses = append(result.FormatClauses, format)
		}
	}
	if reflect.DeepEqual(currentRules.ExclusiveMinimum, enforcedRules.ExclusiveMinimum) {
		result.ExclusiveMinimum = nil
	}
	if reflect.DeepEqual(currentRules.Minimum, enforcedRules.Minimum) {
		result.Minimum = nil
	}
	if reflect.DeepEqual(currentRules.ExclusiveMaximum, enforcedRules.ExclusiveMaximum) {
		result.ExclusiveMaximum = nil
	}
	if reflect.DeepEqual(currentRules.Maximum, enforcedRules.Maximum) {
		result.Maximum = nil
	}
	if reflect.DeepEqual(currentRules.MinLength, enforcedRules.MinLength) {
		result.MinLength = nil
	}
	if reflect.DeepEqual(currentRules.MaxLength, enforcedRules.MaxLength) {
		result.MaxLength = nil
	}
	result.Required = slices.DeleteFunc(result.Required, func(name string) bool {
		return slices.Contains(enforcedRules.Required, name)
	})
	clauses := slices.DeleteFunc(current.Clauses(), func(clause expr.EffectiveValidationClause) bool {
		return slices.ContainsFunc(enforced.Clauses(), func(existing expr.EffectiveValidationClause) bool {
			return existing.Kind == clause.Kind && existing.Value == clause.Value
		})
	})
	if result.HasRequiredOnly() && len(result.Required) == 0 {
		return validationRenderSnapshot{}
	}
	return validationRenderSnapshot{rules: result, clauses: clauses}
}

// mergedValidation returns the expr-owned immutable effective validation for
// att. Invalid declarations have already failed design validation; reaching one
// while generating is an internal lifecycle error.
func mergedValidation(att *expr.AttributeExpr) *expr.ValidationExpr {
	constraints, err := expr.EffectiveConstraintsFor(att)
	if err != nil {
		panic(fmt.Sprintf("invalid effective constraints during generation: %v", err))
	}
	validation := constraints.Validation().Lowered()
	if validation.HasRequiredOnly() && len(validation.Required) == 0 {
		return nil
	}
	return validation
}

func newValidationRenderData(att *expr.AttributeExpr, attCtx *AttributeContext, req, alias, hasDefault bool, target, context string) validationRenderData {
	kind := att.Type.Kind()
	unaliased := unalias(att.Type)
	isNativePointer := unaliased.Kind() == expr.BytesKind || unaliased.Kind() == expr.AnyKind
	isPointer := attCtx.Pointer || (!req && (!hasDefault || !attCtx.UseDefault))
	// Forced-pointer contexts wrap named byte slices, but leave native byte
	// slices and all Any values unchanged. Outside those contexts byte aliases
	// remain slice values, including optional attributes without defaults.
	isForcedBytesPointer := attCtx.Pointer && unaliased.Kind() == expr.BytesKind && kind != expr.BytesKind
	targetValue := target
	if isPointer && expr.IsPrimitive(att.Type) && (!isNativePointer || isForcedBytesPointer) {
		targetValue = "*" + targetValue
	}
	if alias {
		targetValue = GoNativeTypeName(unaliased) + "(" + targetValue + ")"
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
	predicate := "false"
	if len(data.Values) > 0 {
		predicate = oneof(data.TargetValue, data.Values)
	}
	if len(data.Values) > 0 && unalias(data.Attribute.Type).Kind() == expr.AnyKind {
		predicate = jsonValueOneof(data.TargetValue, data.Values)
	} else if len(data.Values) > 0 && compositeEnumValidation(data.Attribute.Type) {
		predicate = collectionEnumPredicate(data)
	}
	b.Add("if !(" + predicate + ") {\n")
	b.Add("\terr = loom.MergeErrors(err, loom.InvalidEnumValueError(" + quoteString(data.Context) + ", " + data.TargetValue + ", " + toSlice(data.Values) + "))\n")
	b.Add("}")
	if data.IsPointer {
		b.Add("\n}")
	}
	return strings.Trim(b.String(), "\n")
}

func compositeEnumValidation(datatype expr.DataType) bool {
	kind := unalias(datatype).Kind()
	return kind == expr.ObjectKind || kind == expr.ArrayKind || kind == expr.MapKind ||
		kind == expr.UnionKind || kind == expr.BytesKind
}

// collectionEnumPredicate compares the JSON values represented by a collection.
// Canonicalization follows the design, so an ArrayOf(UInt) enum authored as
// []uint8 remains an array of numbers rather than a base64-encoded byte string.
func collectionEnumPredicate(data validationRenderData) string {
	values := make([]string, len(data.Values))
	for index, value := range data.Values {
		canonical := enumvalue.Normalize(data.Attribute, value)
		values[index] = "loom.JSONValueEqual(encoded, " + formatRawJSONLiteral(canonical) + ")"
	}
	return "func() bool {\nencoded, encodeErr := loom.JSONValueFrom(" + data.TargetValue + ")\n" +
		"return encodeErr == nil && (" + strings.Join(values, " || ") + ")\n}()"
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
	if scope, ok := data.AttributeCtx.Scope.(messageFieldScope); ok {
		field, _, _ = scope.FieldNames(data.Attribute, data.RequiredName)
	}
	name := expr.AttributeName(data.RequiredName)
	missing := "\n\terr = loom.MergeErrors(err, loom.MissingFieldError(" + quoteString(name) + ", " + quoteString(data.Context) + "))\n}"
	mapped := expr.NewMappedAttributeExpr(data.Attribute)
	presence := data.AttributeCtx.FieldPresence(mapped, name, data.RequiredAttr)
	if presence == OptionalPresence || presence == NullablePresence {
		return "if !" + data.Target + "." + field + ".Present() {" + missing
	}
	if expr.IsUnion(data.RequiredAttr.Type) {
		if _, ok := data.AttributeCtx.Scope.(sumTypeUnionScope); ok {
			return "if " + data.Target + "." + field + ".Kind() == \"\" {" + missing
		}
	}
	return "if " + data.Target + "." + field + " == nil {" + missing
}

func renderArrayValidation(target, validation string, rejectNativeNil, jsonPresence bool, context string) string {
	var b sourceBuilder
	index := "_"
	if rejectNativeNil || jsonPresence {
		index = "i"
	}
	b.Add("for " + index + ", e := range " + target + " {\n")
	if jsonPresence {
		if validation == "" {
			b.Add("\tif _, ok := e.Value(); !ok {\n")
			b.Add("\t\terr = loom.MergeErrors(err, loom.InvalidNullElementError(" + quoteString(context) + ", i))\n")
			b.Add("\t}\n")
			b.Add("}")
			return b.String()
		}
		b.Add("\tif actual, ok := e.Value(); ok {\n")
		b.Add(indentCode(indentCode(validation)))
		b.Add("\t} else {\n")
		b.Add("\t\terr = loom.MergeErrors(err, loom.InvalidNullElementError(" + quoteString(context) + ", i))\n")
		b.Add("\t}\n")
		b.Add("}")
		return b.String()
	}
	if rejectNativeNil {
		b.Add("\tif e == nil {\n")
		b.Add("\t\terr = loom.MergeErrors(err, loom.InvalidNullElementError(" + quoteString(context) + ", i))\n")
		b.Add("\t}\n")
	}
	if validation != "" {
		b.Add(indentCode(validation))
	}
	b.Add("}")
	return b.String()
}

func renderMapValidation(target, keyValidation, valueValidation string, jsonPresence bool, context string) string {
	keyVar := "_"
	if keyValidation != "" {
		keyVar = "k"
	}
	valueVar := "_"
	if valueValidation != "" || jsonPresence {
		valueVar = "v"
	}
	var b sourceBuilder
	fmt.Fprintf(&b, "for %s, %s := range %s {\n", keyVar, valueVar, target)
	if keyValidation != "" {
		b.Add(indentCode(strings.TrimPrefix(keyValidation, "\n")))
	}
	if jsonPresence {
		if valueValidation == "" {
			b.Add("\tif _, ok := " + valueVar + ".Value(); !ok {\n")
			b.Add("\t\terr = loom.MergeErrors(err, loom.InvalidNullMapValueError(" + quoteString(context+"[key]") + "))\n")
			b.Add("\t}\n")
			b.Add("}")
			return b.String()
		}
		b.Add("\tif actual, ok := " + valueVar + ".Value(); ok {\n")
		b.Add(indentCode(indentCode(strings.TrimPrefix(valueValidation, "\n"))))
		b.Add("\t} else {\n")
		b.Add("\t\terr = loom.MergeErrors(err, loom.InvalidNullMapValueError(" + quoteString(context+"[key]") + "))\n")
		b.Add("\t}\n")
	} else if valueValidation != "" {
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
	return "if err2 := " + name + "(" + target + "); err2 != nil {\n\terr = loom.MergeErrors(err, err2)\n}"
}
