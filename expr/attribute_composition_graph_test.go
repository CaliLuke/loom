package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestExtendCompositionGraphs(t *testing.T) {
	t.Run("diamond coalesces", func(t *testing.T) {
		var leaf expr.UserType
		expr.RunDSL(t, func() {
			base := Type("Base", func() {
				Attribute("n", Int, func() {
					Minimum(0)
					Default(1)
				})
				Required("n")
			})
			left := Type("Left", func() {
				Extend(base)
			})
			right := Type("Right", func() {
				Extend(base)
			})
			leaf = Type("Leaf", func() {
				Extend(left)
				Extend(right)
			})
		})
		require.Len(t, *expr.AsObject(leaf), 1)
		require.True(t, leaf.Attribute().IsRequired("n"))
		require.Equal(t, 1, leaf.Attribute().GetDefault("n"))
	})
	t.Run("recursive member admitted", func(t *testing.T) {
		var node expr.UserType
		expr.RunDSL(t, func() {
			base := Type("Base", func() {
				Attribute("n", Int)
			})
			node = Type("Node", func() {
				Extend(base)
				Attribute("next", node)
			})
		})
		require.NotNil(t, node.Attribute().Find("n"))
		require.NotNil(t, node.Attribute().Find("next"))
	})
	t.Run("inheritance cycle rejected", func(t *testing.T) {
		err := expr.RunInvalidDSL(t, func() {
			var a, b expr.UserType
			a = Type("A", func() {
				Extend(b)
			})
			b = Type("B", func() {
				Extend(a)
			})
		})
		require.ErrorContains(t, err, "recursive Extend")
	})
	t.Run("named object descendant admitted", func(t *testing.T) {
		expr.RunDSL(t, func() {
			a := Type("A", func() {
				Attribute("n", Int)
			})
			b := Type("B", func() {
				Extend(a)
				Attribute("more", String)
			})
			base := Type("Base", func() {
				Attribute("item", a)
			})
			Type("Derived", func() {
				Extend(base)
				Attribute("item", b)
			})
		})
	})
}

func TestExtendCompositionMetadata(t *testing.T) {
	for _, test := range []struct {
		name    string
		base    func()
		local   func()
		failure string
	}{
		{"field identity uses first override", func() {
			Meta("struct:field:name", "N", "ignored")
		}, func() {
			Meta("struct:field:name", "N")
		}, ""},
		{"field number uses last tag", func() {
			Meta("rpc:tag", "2", "1")
		}, func() {
			Meta("rpc:tag", "1")
		}, ""},
		{"custom type tuple conflicts", func() {
			Meta("struct:field:type", "string")
		}, func() {
			Meta("struct:field:type", "Custom")
		}, "incompatible inherited metadata"},
		{"name cannot hide under full JSON tag", func() {
			Meta("struct:tag:json", "original", "omitempty")
		}, func() {
			Meta("struct:tag:json:name", "changed")
		}, "incompatible inherited JSON tag"},
		{"documentation local choice", func() {
			Meta("documentation", "base")
		}, func() {
			Meta("documentation", "local")
		}, ""},
		{"documentation removal", func() {
			Meta("documentation", "base")
		}, func() {
			RemoveMeta("documentation")
		}, ""},
		{"role marker removal", func() {
			Meta("security:token")
		}, func() {
			RemoveMeta("security:token")
		}, "cannot remove inherited identity"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var derived expr.UserType
			design := func() {
				base := Type("Base", func() {
					Attribute("n", String, test.base)
				})
				derived = Type("Derived", func() {
					Extend(base)
					Attribute("n", String, test.local)
				})
			}
			if test.failure != "" {
				err := expr.RunInvalidDSL(t, design)
				require.ErrorContains(t, err, test.failure)
				return
			}
			expr.RunDSL(t, design)
			if test.name == "documentation removal" {
				require.NotContains(t, derived.Attribute().Find("n").Meta, "documentation")
			}
		})
	}
}

// Repeated preparation must not duplicate inherited fields or lose predicates.
func TestExtendCompositionRepeatedPreparation(t *testing.T) {
	var base, derived expr.UserType
	root := expr.RunDSL(t, func() {
		base = Type("Base", func() {
			Attribute("n", Int, func() {
				Minimum(0)
			})
			Required("n")
		})
		derived = Type("Derived", func() {
			Extend(base)
			Attribute("n", Int, func() {
				Maximum(100)
			})
		})
	})
	field := derived.Attribute().Find("n")
	root.Prepare()
	root.Prepare()
	require.Len(t, *expr.AsObject(derived), 1)
	require.Same(t, field, derived.Attribute().Find("n"))
	require.True(t, derived.Attribute().IsRequired("n"))
	constraints, err := expr.EffectiveConstraintsFor(field)
	require.NoError(t, err)
	require.Equal(t, 0.0, constraints.LowerBound().Value)
	require.Equal(t, 100.0, constraints.UpperBound().Value)
	require.Nil(t, base.Attribute().Find("n").Validation.Maximum)
}
