package codegen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestNestedNamedObjectRendersOccurrenceAndIntrinsicValidation(t *testing.T) {
	object := &expr.Object{{Name: "s", Attribute: &expr.AttributeExpr{Type: expr.String}}}
	ctx := NewAttributeContext(true, false, true, "", NewNameScope())
	tests := []struct {
		name          string
		intrinsic     *expr.ValidationExpr
		occurrence    *expr.ValidationExpr
		wantEnumCount int
		wantCall      bool
		wantRequired  bool
	}{
		{
			name:          "local only",
			occurrence:    &expr.ValidationExpr{Values: []any{map[string]any{"s": "x"}}},
			wantEnumCount: 1,
		},
		{
			name:          "local refinement and intrinsic",
			intrinsic:     &expr.ValidationExpr{Values: []any{map[string]any{"s": "x"}, map[string]any{"s": "y"}}},
			occurrence:    &expr.ValidationExpr{Values: []any{map[string]any{"s": "x"}}},
			wantEnumCount: 1,
			wantCall:      true,
		},
		{
			name:          "exact duplicate uses named validator",
			intrinsic:     &expr.ValidationExpr{Values: []any{map[string]any{"s": "x"}}},
			occurrence:    &expr.ValidationExpr{Values: []any{map[string]any{"s": "x"}}},
			wantEnumCount: 0,
			wantCall:      true,
		},
		{
			name:         "local required",
			occurrence:   &expr.ValidationExpr{Required: []string{"s"}},
			wantRequired: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inner := &expr.UserTypeExpr{TypeName: "Inner", UID: "Inner", AttributeExpr: &expr.AttributeExpr{
				Type: object, Validation: test.intrinsic,
			}}
			field := &expr.AttributeExpr{Type: inner, Validation: test.occurrence}
			parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "inner", Attribute: field}}}
			code := ValidationCode(parent, nil, ctx, true, false, false, "body")

			require.Equal(t, test.wantEnumCount, strings.Count(code, "InvalidEnumValueError"), code)
			require.Equal(t, test.wantCall, strings.Contains(code, "ValidateInner(body.Inner)"), code)
			require.Equal(t, test.wantRequired, strings.Contains(code, "MissingFieldError"), code)
			if test.wantEnumCount > 0 {
				require.Contains(t, code, "loom.JSONValueFrom(body.Inner)")
			}
		})
	}
}

func TestNestedResultTypeRendersOccurrenceValidation(t *testing.T) {
	result := &expr.ResultTypeExpr{
		UserTypeExpr: &expr.UserTypeExpr{TypeName: "Result", UID: "Result", AttributeExpr: &expr.AttributeExpr{
			Type: &expr.Object{{Name: "s", Attribute: &expr.AttributeExpr{Type: expr.String}}},
		}},
		Identifier: "application/vnd.result",
	}
	field := &expr.AttributeExpr{Type: result, Validation: &expr.ValidationExpr{
		Values: []any{map[string]any{"s": "x"}},
	}}
	parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "result", Attribute: field}}}
	code := ValidationCode(parent, nil, NewAttributeContext(true, false, true, "", NewNameScope()),
		true, false, false, "body")

	require.Contains(t, code, "loom.JSONValueFrom(body.Result)")
	require.Contains(t, code, "InvalidEnumValueError")
}
