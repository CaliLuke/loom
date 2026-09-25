package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestViewKeepsLaterDeclaredResultTypes verifies that a view attribute keeps
// the complete type of the result type attribute it selects, whichever order
// the DSL declares the result types in.
func TestViewKeepsLaterDeclaredResultTypes(t *testing.T) {
	cases := []struct {
		Name  string
		Field string
		Elem  func(*testing.T, *expr.AttributeExpr) *expr.AttributeExpr
	}{
		{"direct", "child", directField},
		{"array", "children", arrayElem},
		{"map", "by_key", mapElem},
		{"inline-object", "wrapper", innerField},
	}
	orders := []struct {
		Name string
		DSL  func()
	}{
		{"declared-later", func() {
			holderResultType()
			laterResultType()
		}},
		{"declared-earlier", func() {
			laterResultType()
			holderResultType()
		}},
	}
	for _, order := range orders {
		for _, c := range cases {
			t.Run(order.Name+"/"+c.Name, func(t *testing.T) {
				root := expr.RunDSL(t, order.DSL)
				holder, ok := root.UserType("Holder").(*expr.ResultTypeExpr)
				require.True(t, ok)
				view := holder.View(expr.DefaultView)
				require.NotNil(t, view)
				field := view.Find(c.Field)
				require.NotNil(t, field)

				later, ok := c.Elem(t, field).Type.(*expr.ResultTypeExpr)
				require.True(t, ok)
				require.Equal(t, "Later", later.Name())
				require.NotNil(t, expr.AsObject(later).Attribute("name"))
				require.NotNil(t, expr.AsObject(later).Attribute("count"))
				require.NotNil(t, later.View("tiny"))

				projected, err := expr.Project(holder, expr.DefaultView)
				require.NoError(t, err)
				require.NotNil(t, projected.Find(c.Field))
			})
		}
	}
}

// TestViewAttributeViewDoesNotChangeResultType verifies that a view that
// selects the view of an attribute leaves the result type attribute unchanged.
func TestViewAttributeViewDoesNotChangeResultType(t *testing.T) {
	root := expr.RunDSL(t, func() {
		holderResultType()
		laterResultType()
	})
	holder, ok := root.UserType("Holder").(*expr.ResultTypeExpr)
	require.True(t, ok)

	viewed := holder.View(expr.DefaultView).Find("child")
	require.NotNil(t, viewed)
	view, ok := viewed.Meta.Last(expr.ViewMetaKey)
	require.True(t, ok)
	require.Equal(t, "tiny", view)

	canonical := holder.Find("child")
	require.NotNil(t, canonical)
	_, ok = canonical.Meta.Last(expr.ViewMetaKey)
	require.False(t, ok)

	projected, err := expr.Project(holder, expr.DefaultView)
	require.NoError(t, err)
	child, ok := projected.Find("child").Type.(*expr.ResultTypeExpr)
	require.True(t, ok)
	require.NotNil(t, expr.AsObject(child).Attribute("name"))
	require.Nil(t, expr.AsObject(child).Attribute("count"))
}

func holderResultType() {
	ResultType("application/vnd.holder", func() {
		TypeName("Holder")
		Attributes(func() {
			Attribute("id", String)
			Attribute("child", "Later")
			Attribute("children", ArrayOf("Later"))
			Attribute("by_key", MapOf(String, "Later"))
			Attribute("wrapper", func() {
				Attribute("inner", "Later")
			})
		})
		View("default", func() {
			Attribute("id")
			Attribute("child", func() {
				View("tiny")
			})
			Attribute("children")
			Attribute("by_key")
			Attribute("wrapper")
		})
	})
}

func laterResultType() {
	ResultType("application/vnd.later", func() {
		TypeName("Later")
		Attributes(func() {
			Attribute("name", String)
			Attribute("count", Int)
		})
		View("default", func() {
			Attribute("name")
			Attribute("count")
		})
		View("tiny", func() {
			Attribute("name")
		})
	})
}

func innerField(t *testing.T, field *expr.AttributeExpr) *expr.AttributeExpr {
	inner := field.Find("inner")
	require.NotNil(t, inner)
	return inner
}
