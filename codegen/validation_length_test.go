package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestLengthValidationDiagnosticArguments(t *testing.T) {
	minimum, maximum := 2, 3
	for _, tc := range []struct {
		name   string
		typ    expr.DataType
		length string
	}{
		{"string", expr.String, "utf8.RuneCountInString(value)"},
		{"bytes", expr.Bytes, "len(value)"},
		{"array", &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}, "len(value)"},
		{"map", &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}, "len(value)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			att := &expr.AttributeExpr{Type: tc.typ, Validation: &expr.ValidationExpr{MinLength: &minimum, MaxLength: &maximum}}
			ctx := NewAttributeContext(false, false, false, "", NewNameScope())
			code := ValidationCode(att, nil, ctx, true, false, false, "value")
			require.Contains(t, code, `loom.InvalidLengthError("value", `+tc.length+`, 2, true)`)
			require.Contains(t, code, `loom.InvalidLengthError("value", `+tc.length+`, 3, false)`)
		})
	}
}
