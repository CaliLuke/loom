//nolint:errcheck // Generator helpers write only to in-memory buffers/builders.
package codegen

import (
	"errors"
	"fmt"
	"strconv"
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

func renderSimplePointerWrappedValidation(isPointer bool, target, body string) string {
	body = strings.Trim(body, "\n")
	if !isPointer {
		return body
	}
	return "if " + target + " != nil {\n" + indentCode(body) + "}"
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
	return generatedRequiredValidationFrom(att, att.Validation, attCtx)
}

func generatedRequiredValidationFrom(att *expr.AttributeExpr, validation *expr.ValidationExpr, attCtx *AttributeContext) (res []string) {
	if validation == nil {
		return
	}
	obj := expr.AsObject(att.Type)
	for _, req := range validation.Required {
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

// toSlice returns Go code that represents the given slice.
func toSlice(val []any) string {
	elems := make([]string, len(val))
	for i, v := range val {
		elems[i] = validationGoLiteral(v)
	}
	return "[]any{" + strings.Join(elems, ", ") + "}"
}

// oneof produces code that compares target with each element of vals and ORs
// the result, e.g. "target == 1 || target == 2".
func oneof(target string, vals []any) string {
	elems := make([]string, len(vals))
	for i, v := range vals {
		elems[i] = target + " == " + validationGoLiteral(v)
	}
	return strings.Join(elems, " || ")
}

func quoteString(v string) string {
	return strconv.Quote(v)
}

func validationGoLiteral(v any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%#v", v)
	return b.String()
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
