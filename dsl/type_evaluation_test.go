package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestAttributeDSLCopiesInProgressTypeOnceComplete verifies that an attribute
// DSL that refines a reference to a type whose DSL is still running refines a
// complete copy of the type made once the DSL of the type ends.
func TestAttributeDSLCopiesInProgressTypeOnceComplete(t *testing.T) {
	root := expr.RunDSL(t, func() {
		Type("Node", func() {
			Attribute("name", String)
			Attribute("holder", func() {
				Attribute("direct", "Node", func() {
					Required("name")
				})
				Attribute("list", ArrayOf("Node"), func() {
					MinLength(1)
				})
			})
			Attribute("tail", String)
		})
	})

	node := root.UserType("Node")
	require.False(t, node.Attribute().IsRequired("name"), "shared type must not change")
	holder := node.Attribute().Find("holder")
	require.NotNil(t, holder)
	direct, ok := holder.Find("direct").Type.(expr.UserType)
	require.True(t, ok)
	elem, ok := expr.AsArray(holder.Find("list").Type).ElemType.Type.(expr.UserType)
	require.True(t, ok)
	for name, local := range map[string]expr.UserType{"direct": direct, "list": elem} {
		require.NotSame(t, node, local, name)
		for _, field := range []string{"name", "holder", "tail"} {
			require.NotNil(t, local.Attribute().Find(field), "%s lacks %s", name, field)
		}
	}
	require.NotNil(t, direct.Attribute().Validation)
	require.Equal(t, []string{"name"}, direct.Attribute().Validation.Required)
	require.False(t, elem.Attribute().IsRequired("name"))
}

// TestAttributeDSLCopyIgnoresLaterSharedTypeChanges verifies that a later DSL
// that changes a shared type does not change the local copies made before.
func TestAttributeDSLCopyIgnoresLaterSharedTypeChanges(t *testing.T) {
	root := expr.RunDSL(t, func() {
		item := Type("LeakItem", func() {
			Attribute("name", String)
			Attribute("id", Int)
		})
		holder := Type("LeakHolder", func() {
			Attribute("a", item, func() {
				Required("name")
			})
		})
		Service("LeakService", func() {
			Method("Show", func() {
				Payload(holder)
				Error("bad", item, func() {
					Required("id")
				})
			})
		})
	})

	local := root.UserType("LeakHolder").Attribute().Find("a").Type.(expr.UserType)
	require.Equal(t, []string{"name"}, local.Attribute().Validation.Required)
}

// TestPayloadCopyResolvesCycleThroughLocalCopy verifies that a customized
// payload copy of a type reached back through a local copy of another type
// references the payload copy, like a type declared in dependency order.
func TestPayloadCopyResolvesCycleThroughLocalCopy(t *testing.T) {
	root := expr.RunDSL(t, func() {
		y := Type("CycleY", func() {
			Attribute("v", String)
			Attribute("x", "CycleX")
		})
		x := Type("CycleX", func() {
			Attribute("name", String)
			Attribute("y", y, func() {
				Required("v")
			})
		})
		Service("CycleService", func() {
			Method("Show", func() {
				Payload(x, func() {
					Required("name")
				})
			})
		})
	})

	payload := root.Service("CycleService").Method("Show").Payload.Type
	require.NotSame(t, root.UserType("CycleX"), payload)
	y := payload.(expr.UserType).Attribute().Find("y").Type.(expr.UserType)
	require.Same(t, payload, y.Attribute().Find("x").Type)
}

// TestDeclarationTimeCollectionCopiesTypeOnceComplete verifies that ArrayOf
// and MapOf with a DSL evaluated outside any DSL body refine a complete copy
// of the element type made once its DSL runs, without running that DSL
// before the types it references by name are declared.
func TestDeclarationTimeCollectionCopiesTypeOnceComplete(t *testing.T) {
	item := func() expr.UserType {
		return Type("DeclaredItem", func() {
			Attribute("u", "DeclaredLater")
			Required("u")
		})
	}
	later := func() {
		Type("DeclaredLater", func() {
			Attribute("z", String)
		})
	}
	cases := []struct {
		Name    string
		DSL     func()
		Element func(*expr.RootExpr) *expr.AttributeExpr
	}{
		{"array-type-argument", func() {
			it := item()
			Type("DeclaredHolder", ArrayOf(it, func() {
				Description("elem")
			}))
			later()
		}, func(root *expr.RootExpr) *expr.AttributeExpr {
			return expr.AsArray(root.UserType("DeclaredHolder").Attribute().Type).ElemType
		}},
		{"map-type-argument", func() {
			it := item()
			Type("DeclaredHolder", MapOf(String, it, func() {
				Elem(func() {
					Description("elem")
				})
			}))
			later()
		}, func(root *expr.RootExpr) *expr.AttributeExpr {
			return expr.AsMap(root.UserType("DeclaredHolder").Attribute().Type).ElemType
		}},
		{"variable-used-by-later-type", func() {
			// Like a package-level var, the collection is evaluated
			// outside any DSL body.
			elem := ArrayOf(item(), func() {
				Description("elem")
			})
			Type("DeclaredHolder", func() {
				Attribute("list", elem)
			})
			later()
		}, func(root *expr.RootExpr) *expr.AttributeExpr {
			return expr.AsArray(root.UserType("DeclaredHolder").Attribute().Find("list").Type).ElemType
		}},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := expr.RunDSL(t, c.DSL)
			shared := root.UserType("DeclaredItem")
			elem := c.Element(root)
			require.Equal(t, "elem", elem.Description)
			local, ok := elem.Type.(expr.UserType)
			require.True(t, ok)
			require.NotSame(t, shared, local)
			u := local.Attribute().Find("u")
			require.NotNil(t, u)
			require.Equal(t, "DeclaredLater", u.Type.Name())
			require.NotNil(t, expr.AsObject(u.Type).Attribute("z"))
			require.True(t, local.Attribute().IsRequired("u"))
		})
	}
}

// TestKeptReferenceCopiesKeepRequiredFields verifies that the required fields
// of a kept reference apply to its complete copy of the type even when the
// kept reference is itself copied before the DSL of the type ends.
func TestKeptReferenceCopiesKeepRequiredFields(t *testing.T) {
	cases := []struct {
		Name    string
		DSL     func()
		Element func(*expr.RootExpr) *expr.AttributeExpr
	}{
		{"nested-array", func() {
			item := Type("KeptItem", func() {
				Attribute("id", String)
			})
			Type("KeptHolder", ArrayOf(ArrayOf(item, func() {
				Required("id")
			}), func() {
				MinLength(1)
			}))
		}, func(root *expr.RootExpr) *expr.AttributeExpr {
			outer := expr.AsArray(root.UserType("KeptHolder").Attribute().Type)
			return expr.AsArray(outer.ElemType.Type).ElemType
		}},
		{"array-in-map", func() {
			item := Type("KeptItem", func() {
				Attribute("id", String)
			})
			Type("KeptHolder", MapOf(String, ArrayOf(item, func() {
				Required("id")
			}), func() {
				MinLength(1)
			}))
		}, func(root *expr.RootExpr) *expr.AttributeExpr {
			m := expr.AsMap(root.UserType("KeptHolder").Attribute().Type)
			return expr.AsArray(m.ElemType.Type).ElemType
		}},
		{"refined-variable", func() {
			item := Type("KeptItem", func() {
				Attribute("id", String)
			})
			inner := ArrayOf(item, func() {
				Required("id")
			})
			Type("KeptHolder", ArrayOf(inner, func() {
				MinLength(1)
			}))
		}, func(root *expr.RootExpr) *expr.AttributeExpr {
			outer := expr.AsArray(root.UserType("KeptHolder").Attribute().Type)
			return expr.AsArray(outer.ElemType.Type).ElemType
		}},
		{"in-progress-cycle", func() {
			Type("KeptItem", func() {
				Attribute("id", String)
				Attribute("b", "KeptHolder", func() {
					Description("x")
				})
			})
			Type("KeptHolder", func() {
				Attribute("as", ArrayOf(ArrayOf("KeptItem", func() {
					Required("id")
				}), func() {
					MinLength(1)
				}))
			})
		}, func(root *expr.RootExpr) *expr.AttributeExpr {
			outer := expr.AsArray(root.UserType("KeptHolder").Attribute().Find("as").Type)
			return expr.AsArray(outer.ElemType.Type).ElemType
		}},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := expr.RunDSL(t, c.DSL)
			shared := root.UserType("KeptItem")
			require.False(t, shared.Attribute().IsRequired("id"), "shared type must not change")
			local, ok := c.Element(root).Type.(expr.UserType)
			require.True(t, ok)
			require.NotSame(t, shared, local)
			require.NotNil(t, local.Attribute().Find("id"))
			require.NotNil(t, local.Attribute().Validation)
			require.Equal(t, []string{"id"}, local.Attribute().Validation.Required)
		})
	}
}
