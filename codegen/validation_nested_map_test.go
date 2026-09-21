package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestNestedMapValidationVariables(t *testing.T) {
	root := RunDSL(t, func() {
		item := dsl.Type("Item", dsl.String, func() {
			dsl.MinLength(1)
		})
		dsl.Type("Nested", dsl.MapOf(dsl.String, dsl.MapOf(dsl.String, item)))
	})
	ctx := NewAttributeContext(false, false, true, "", NewNameScope())
	code := ValidationCode(&expr.AttributeExpr{Type: root.UserType("Nested")}, nil, ctx, true, false, false, "value")
	require.Contains(t, code, "for _, v := range v {")
	require.Contains(t, code, "utf8.RuneCountInString(string(v))")
}
