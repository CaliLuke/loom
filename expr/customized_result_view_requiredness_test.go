package expr_test

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

// TestCustomizedResultViewRequiredness preserves authored view overrides while
// applying method-level requiredness without changing the original result type.
func TestCustomizedResultViewRequiredness(t *testing.T) {
	root := expr.RunDSL(t, func() {
		menu := ResultType("application/vnd.customized-menu", "Menu", func() {
			Attributes(func() {
				Attribute("x", String)
				Attribute("y", String)
			})
			View("default", func() {
				Attribute("x")
				Attribute("y")
			})
			View("optional", func() {
				Attribute("x")
				ViewOptional("x")
			})
			View("required", func() {
				Attribute("x")
				Attribute("y")
				ViewRequired("y")
			})
			View("tiny", func() {
				Attribute("x")
			})
			View("other", func() {
				Attribute("y")
			})
		})
		Service("menus", func() {
			Method("custom", func() {
				Result(menu, func() {
					Required("x")
				})
			})
			Method("original", func() {
				Result(menu)
			})
		})
	})
	custom := root.Services[0].Methods[0].Result.Type.(*expr.ResultTypeExpr)
	original := root.Services[0].Methods[1].Result.Type.(*expr.ResultTypeExpr)
	for _, tc := range []struct {
		name             string
		custom, original []string
	}{
		{"default", []string{"x"}, nil},
		{"optional", nil, nil},
		{"required", []string{"x", "y"}, []string{"y"}},
		{"tiny", []string{"x"}, nil},
		{"other", nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.ElementsMatch(t, tc.custom, custom.View(tc.name).AllRequired())
			require.ElementsMatch(t, tc.original, original.View(tc.name).AllRequired())
			projected, err := expr.Project(custom, tc.name)
			require.NoError(t, err)
			require.ElementsMatch(t, tc.custom, projected.AllRequired())
		})
	}
}
