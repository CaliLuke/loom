package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

// TestNullableAliasValidation preserves the concrete value type when unwrapping
// nullable attributes, including callers that supply an alias's underlying type.
func TestNullableAliasValidation(t *testing.T) {
	minimum := 2
	attribute := &expr.AttributeExpr{
		Type:       expr.String,
		Nullable:   true,
		Validation: &expr.ValidationExpr{MinLength: &minimum},
	}
	named := &expr.UserTypeExpr{TypeName: "Label", AttributeExpr: attribute}
	cases := []struct {
		name      string
		attribute *expr.AttributeExpr
		alias     bool
		want      string
	}{
		{"native", attribute, false, "utf8.RuneCountInString(actual)"},
		{"alias underlying", attribute, true, "utf8.RuneCountInString(string(actual))"},
		{"named alias", &expr.AttributeExpr{Type: named}, false, "utf8.RuneCountInString(string(actual))"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := NewAttributeContext(false, false, false, "", NewNameScope())
			code := ValidationCode(c.attribute, named, ctx, true, c.alias, false, "body")
			require.Contains(t, code, c.want)
			require.Contains(t, code, "body.Value()")
		})
	}
}

// TestNullableAliasValidationNativeNames checks scalar names whose DSL spelling
// differs from the Go type used in validation expressions.
func TestNullableAliasValidationNativeNames(t *testing.T) {
	minimum := 2
	cases := []struct {
		name       string
		primitive  expr.Primitive
		validation *expr.ValidationExpr
		want       string
	}{
		{"bool", expr.Boolean, &expr.ValidationExpr{Values: []any{true}}, "bool(actual)"},
		{"bytes", expr.Bytes, &expr.ValidationExpr{MinLength: &minimum}, "len([]byte(actual))"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			alias := &expr.UserTypeExpr{TypeName: "Value", AttributeExpr: &expr.AttributeExpr{
				Type: c.primitive, Nullable: true, Validation: c.validation,
			}}
			ctx := NewAttributeContext(false, false, false, "", NewNameScope())
			code := ValidationCode(&expr.AttributeExpr{Type: alias}, nil, ctx, true, true, false, "body")
			require.Contains(t, code, c.want)
		})
	}
}
