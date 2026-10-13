package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestProjectAuthoredIntrinsicNullableResult(t *testing.T) {
	var child *expr.ResultTypeExpr
	expr.RunDSL(t, func() {
		child = dsl.ResultType("application/vnd.nullable-child", "Child", func() {
			dsl.Attributes(func() {
				dsl.Nullable()
				dsl.Attribute("text", dsl.String)
				dsl.Attribute("label", dsl.String)
			})
			dsl.View("default", func() {
				dsl.Attribute("text")
				dsl.Attribute("label")
			})
			dsl.View("tiny", func() {
				dsl.Attribute("label")
			})
		})
	})
	require.True(t, expr.IsNullable(&expr.AttributeExpr{Type: child}))
	for _, selected := range []string{"default", "tiny"} {
		t.Run(selected, func(t *testing.T) {
			projected, err := expr.Project(child, selected)
			require.NoError(t, err)
			require.True(t, expr.IsNullable(&expr.AttributeExpr{Type: projected}))
			require.True(t, child.Nullable)
		})
	}
}
