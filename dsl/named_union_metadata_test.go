package dsl_test

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestNamedUnionTypeHonorsUnionMetadata(t *testing.T) {
	root := expr.RunDSL(t, func() {
		choice := Type("Choice", OneOf(String, Int), func() {
			Meta("oneof:type:field", "kind")
			Meta("oneof:value:field", "data")
			Meta("oneof:typename", "Selection")
		})
		Type("Holder", func() {
			Attribute("choice", choice, func() {
				Description("local occurrence")
			})
			OneOf("inline", func() {
				Meta("oneof:type:field", "kind")
				Meta("oneof:value:field", "data")
				Meta("oneof:typename", "InlineSelection")
				Attribute("String", String)
				Attribute("Int", Int)
			})
		})
	})
	for _, att := range []*expr.AttributeExpr{root.UserType("Choice").Attribute(), root.UserType("Holder").Attribute().Find("choice")} {
		union := expr.AsUnion(att.Type)
		require.NotNil(t, union)
		require.Equal(t, "kind", union.GetTypeKey())
		require.Equal(t, "data", union.GetValueKey())
		require.Equal(t, "Selection", union.TypeName)
		require.True(t, union.ExplicitTypeName)
	}
	inline := expr.AsUnion(root.UserType("Holder").Attribute().Find("inline").Type)
	require.Equal(t, "kind", inline.GetTypeKey())
	require.Equal(t, "data", inline.GetValueKey())
}
