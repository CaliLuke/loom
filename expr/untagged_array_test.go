package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestUntaggedNamedArrayBranch(t *testing.T) {
	root := expr.RunDSL(t, func() {
		item := Type("Item", func() {
			Attribute("name", String)
			Required("name")
		})
		items := Type("Items", ArrayOf(item))
		page := Type("Page", func() {
			Attribute("total", Int)
			Required("total")
		})
		Service("listing", func() {
			Method("list", func() {
				Result(OneOf(items, page), func() {
					Untagged()
				})
				HTTP(func() {
					GET("/items")
				})
			})
		})
	})
	union := expr.AsUnion(root.Services[0].Methods[0].Result.Type)
	require.True(t, union.Untagged)
	require.True(t, expr.IsArray(union.Values[0].Attribute.Type))
}

func TestUntaggedRejectsUnsupportedNestedShapes(t *testing.T) {
	for _, array := range []bool{false, true} {
		err := expr.RunInvalidDSL(t, func() {
			nested := Type("Nested", func() {
				Attribute("mapping", MapOf(Boolean, String))
			})
			wrapped := Type("Wrapped", func() {
				Attribute("nested", nested)
			})
			other := Type("Other", func() {
				Attribute("other", String)
			})
			if array {
				wrapped = Type("Items", ArrayOf(nested))
			}
			Service("unsupported", func() {
				Method("show", func() {
					Result(OneOf(wrapped, other), func() {
						Untagged()
					})
				})
			})
		})
		require.ErrorContains(t, err, `field "mapping" must be primitive`)
	}
}

func TestUntaggedRejectsOpaqueBranchSemantics(t *testing.T) {
	for _, tc := range []struct {
		name      string
		configure func()
		message   string
	}{
		{"codec", func() {
			Meta("struct:field:type", "Custom", "example.com/custom")
		}, "opaque codec"},
		{"schema codec", func() {
			Meta("openapi:contentEncoding", "hex")
		}, "schema encoding override"},
		{"case folding", func() {
			Meta("struct:tag:json", "name,case:ignore")
		}, "JSON codec option"},
		{"name-only case folding", func() {
			Meta("struct:tag:json:name", "name,case:ignore")
		}, "JSON codec option"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				item := Type("Item", func() {
					Attribute("name", String, tc.configure)
				})
				items := Type("Items", ArrayOf(item))
				page := Type("Page", func() {
					Attribute("total", Int)
				})
				Service("listing", func() {
					Method("list", func() {
						Result(OneOf(items, page), func() {
							Untagged()
						})
					})
				})
			})
			require.ErrorContains(t, err, tc.message)
		})
	}
}

func TestUntaggedJSONTagPrecedence(t *testing.T) {
	expr.RunDSL(t, func() {
		item := Type("Item", func() {
			Attribute("name", String, func() {
				Meta("struct:tag:json:name", "ignored,case:ignore")
				Meta("struct:tag:json", "name,case:strict")
			})
		})
		items := Type("Items", ArrayOf(item))
		page := Type("Page", func() {
			Attribute("total", Int)
		})
		Service("listing", func() {
			Method("list", func() {
				Result(OneOf(items, page), func() {
					Untagged()
				})
			})
		})
	})
}
